package exportdata

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"lazyora/internal/oracle"
)

func sampleData() Dataset {
	return Dataset{Columns: []string{"TEXT", "NUMBER", "DATE", "TS", "NULL_VALUE", "EMPTY", "FORMULA"}, Rows: [][]oracle.Cell{{
		{Text: "한글 O'Reilly\n탭\t값", Type: "VARCHAR2"}, {Text: "123456789012345678901234567890.12345678", Type: "NUMBER"},
		{Text: "2026-10-20 12:34:56Z", Type: "DATE"}, {Text: "2026-10-20 12:34:56.123456Z", Type: "TIMESTAMP"},
		{Null: true, Type: "VARCHAR2"}, {Text: "", Type: "VARCHAR2"}, {Text: "=SUM(1,2)", Type: "VARCHAR2"},
	}}}
}

func TestCSVAndTSVRoundTrip(t *testing.T) {
	for _, format := range []string{"csv", "tsv"} {
		var out bytes.Buffer
		data := sampleData()
		if err := Write(context.Background(), &out, data, Options{Format: format, Headers: true, BOM: true, NullText: `\N`}); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(out.Bytes(), []byte{0xef, 0xbb, 0xbf}) {
			t.Fatal("BOM missing")
		}
		reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\ufeff")))
		if format == "tsv" {
			reader.Comma = '\t'
		}
		rows, err := reader.ReadAll()
		if err != nil || len(rows) != 2 || rows[1][0] != data.Rows[0][0].Text || rows[1][4] != `\N` || rows[1][5] != "" || rows[1][6] != "=SUM(1,2)" {
			t.Fatalf("%v %v", rows, err)
		}
	}
}

func TestXLSXTextCellsPreservePrecisionAndPreventFormulas(t *testing.T) {
	var out bytes.Buffer
	if err := Write(context.Background(), &out, sampleData(), Options{Format: "xlsx", Headers: true}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, part := range archive.File {
		reader, err := part.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(bytes.NewReader(data))
		for {
			_, err = decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("invalid %s: %v", part.Name, err)
			}
		}
		parts[part.Name] = string(data)
	}
	if len(parts) != 5 {
		t.Fatal("incomplete XLSX package")
	}
	var worksheet struct {
		Rows []struct {
			Cells []struct {
				Reference string `xml:"r,attr"`
				Type      string `xml:"t,attr"`
				Inline    struct {
					Text string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err = xml.Unmarshal([]byte(parts["xl/worksheets/sheet1.xml"]), &worksheet); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, row := range worksheet.Rows {
		for _, cell := range row.Cells {
			if cell.Type != "inlineStr" {
				t.Fatal("cell may be interpreted as numeric/formula")
			}
			values[cell.Reference] = cell.Inline.Text
		}
	}
	if values["B2"] != sampleData().Rows[0][1].Text || values["G2"] != "=SUM(1,2)" || values["A2"] != sampleData().Rows[0][0].Text {
		t.Fatal("Excel altered precision/text")
	}
	if _, exists := values["E2"]; exists {
		t.Fatal("NULL exported as text")
	}
	if empty, exists := values["F2"]; !exists || empty != "" {
		t.Fatal("empty string lost")
	}
	if strings.Contains(parts["xl/worksheets/sheet1.xml"], "<f>") {
		t.Fatal("formula generated")
	}
	if _, err = xmlText(strings.Repeat("x", 32768)); err == nil {
		t.Fatal("oversized Excel cell accepted")
	}
	if _, err = xmlText("bad\x00cell"); err == nil {
		t.Fatal("invalid XML character accepted")
	}
}

func TestSQLLiteralsAndTargetMapping(t *testing.T) {
	var out bytes.Buffer
	if err := Write(context.Background(), &out, sampleData(), Options{Format: "sql", Schema: "app", Table: "target", Columns: sampleData().Columns}); err != nil {
		t.Fatal(err)
	}
	sql := out.String()
	for _, want := range []string{`INSERT INTO "APP"."TARGET"`, "O''Reilly", sampleData().Rows[0][1].Text, "TO_DATE('", "TO_TIMESTAMP('", ", NULL, '', '=SUM(1,2)'"} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %q in %s", want, sql)
		}
	}
	if strings.Contains(sql, "COMMIT;") {
		t.Fatal("export resolves DB transaction")
	}
	if literal, err := SQLLiteral(oracle.Cell{Text: "NULL", Type: "VARCHAR2"}); err != nil || literal != "'NULL'" {
		t.Fatal("literal NULL confused with SQL NULL")
	}
	if literal, err := SQLLiteral(oracle.Cell{Text: "00ff", Type: "RAW"}); err != nil || literal != "HEXTORAW('00ff')" {
		t.Fatal("RAW conversion failed")
	}
	if _, err := SQLLiteral(oracle.Cell{Text: "blob", Type: "BLOB"}); err == nil {
		t.Fatal("unsupported type accepted")
	}
	if _, err := targetIdentifier("target; DROP TABLE x"); err == nil {
		t.Fatal("invalid target accepted")
	}
}

func TestAtomicSaveFailureCancelConflictAndPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "result.csv")
	if err := os.WriteFile(path, []byte("keep existing"), 0600); err != nil {
		t.Fatal(err)
	}
	revision, _ := Inspect(path)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Save(ctx, path, revision, sampleData(), Options{Format: "csv"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	bad := sampleData()
	bad.Rows[0][0].Text = strings.Repeat("x", 32768)
	if _, err := Save(context.Background(), path, revision, bad, Options{Format: "xlsx"}); err == nil {
		t.Fatal("invalid cell saved")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "keep existing" {
		t.Fatal("failed export replaced file")
	}
	if err := os.WriteFile(path, []byte("external change longer"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(context.Background(), path, revision, sampleData(), Options{Format: "csv"}); !errors.Is(err, ErrChanged) {
		t.Fatal("external modification ignored")
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0440); err != nil {
			t.Fatal(err)
		}
		revision, _ = Inspect(path)
		if _, err := Save(context.Background(), path, revision, sampleData(), Options{Format: "csv"}); !errors.Is(err, os.ErrPermission) {
			t.Fatal("read-only overwritten")
		}
		if err := os.Chmod(path, 0640); err != nil {
			t.Fatal(err)
		}
	}
	revision, _ = Inspect(path)
	if rows, err := Save(context.Background(), path, revision, sampleData(), Options{Format: "csv", Headers: true}); err != nil || rows != 1 {
		t.Fatalf("save: %d %v", rows, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("temporary/lock files leaked")
	}
	limited := &checkedWriter{ctx: context.Background(), writer: io.Discard, written: MaxBytes - 1}
	if _, err := limited.Write([]byte("xx")); err == nil {
		t.Fatal("export byte cap ignored")
	}
}
