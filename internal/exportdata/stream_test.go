package exportdata

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lazyora/internal/oracle"
)

func generatedSource(count int, value string) Source {
	n := 0
	return Source{Columns: []string{"ID", "TEXT", "OPTIONAL"}, Next: func() ([]oracle.Cell, error) {
		if n == count {
			return nil, io.EOF
		}
		n++
		return []oracle.Cell{{Text: fmt.Sprint(n), Type: "NUMBER"}, {Text: value, Type: "VARCHAR2"}, {Null: true, Type: "VARCHAR2"}}, nil
	}}
}

func TestStreamingExportsBeyondDisplayCap(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "sql", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			var progress int64
			options := Options{Format: format, Headers: true, Unlimited: true, NullText: `\N`, Table: "TARGET", Progress: func(n int64) { progress = n }}
			count, err := WriteStream(context.Background(), &out, generatedSource(2507, "한글 ' 줄\n바꿈\t=1"), options)
			if err != nil || count != 2507 || progress != count {
				t.Fatalf("%d %d %v", count, progress, err)
			}
			switch format {
			case "csv", "tsv":
				reader := csv.NewReader(bytes.NewReader(out.Bytes()))
				if format == "tsv" {
					reader.Comma = '\t'
				}
				rows, err := reader.ReadAll()
				if err != nil || len(rows) != 2508 || rows[2507][0] != "2507" || rows[1][2] != `\N` {
					t.Fatal("streamed delimited data truncated/corrupted")
				}
			case "sql":
				if strings.Count(out.String(), "INSERT INTO") != 2507 || !strings.Contains(out.String(), "VALUES (2507,") {
					t.Fatal("INSERTs truncated")
				}
			case "xlsx":
				parts := xlsxParts(t, out.Bytes())
				if strings.Count(parts["xl/worksheets/sheet1.xml"], "<row ") != 2508 || !strings.Contains(parts["xl/worksheets/sheet1.xml"], ">2507<") {
					t.Fatal("XLSX truncated")
				}
			}
		})
	}
}

func xlsxParts(t *testing.T, data []byte) map[string]string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, part := range archive.File {
		reader, err := part.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(bytes.NewReader(content))
		for {
			_, err = decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s: %v", part.Name, err)
			}
		}
		parts[part.Name] = string(content)
	}
	return parts
}

func TestExcelAutomaticSheetRollover(t *testing.T) {
	for _, headers := range []bool{true, false} {
		for _, count := range []int{0, 2, 3, 4, 5, 6, 7} {
			var out bytes.Buffer
			reader := &rowReader{ctx: context.Background(), source: generatedSource(count, "한글")}
			if err := writeXLSXSheets(context.Background(), &out, reader, Options{Headers: headers}, 3); err != nil {
				t.Fatal(err)
			}
			parts := xlsxParts(t, out.Bytes())
			capacity := 3
			if headers {
				capacity--
			}
			sheets := (count + capacity - 1) / capacity
			if sheets == 0 {
				sheets = 1
			}
			if len(parts) != 4+sheets {
				t.Fatalf("%d rows headers %v: wrong sheet count %d", count, headers, len(parts)-4)
			}
			seen := 0
			for i := 1; i <= sheets; i++ {
				sheet := parts[fmt.Sprintf("xl/worksheets/sheet%d.xml", i)]
				rows := strings.Count(sheet, "<row ")
				if rows > 3 {
					t.Fatal("Excel sheet row limit exceeded")
				}
				if headers {
					if !strings.Contains(sheet, ">ID<") {
						t.Fatal("repeated header missing")
					}
					rows--
				}
				seen += rows
				for _, part := range []string{"[Content_Types].xml", "xl/_rels/workbook.xml.rels"} {
					if !strings.Contains(parts[part], fmt.Sprintf("sheet%d.xml", i)) {
						t.Fatal("worksheet relationship missing")
					}
				}
			}
			if seen != count || reader.count != int64(count) {
				t.Fatal("rows lost/duplicated on rollover")
			}
		}
	}
}

func TestStreamingByteCapOptOutAndWriterError(t *testing.T) {
	source := func() Source { return generatedSource(2100, strings.Repeat("x", 32768)) }
	if _, err := WriteStream(context.Background(), io.Discard, source(), Options{Format: "csv"}); err == nil {
		t.Fatal("loaded export cap lost")
	}
	if count, err := WriteStream(context.Background(), io.Discard, source(), Options{Format: "csv", Unlimited: true}); err != nil || count != 2100 {
		t.Fatalf("full export still capped: %d %v", count, err)
	}
	diskErr := errors.New("disk full")
	if _, err := WriteStream(context.Background(), errorWriter{diskErr}, generatedSource(2500, "text"), Options{Format: "csv", Unlimited: true}); !errors.Is(err, diskErr) {
		t.Fatal("disk failure hidden", err)
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestStreamingAtomicFailureCancelAndSourceClose(t *testing.T) {
	for _, scenario := range []string{"source error", "cancel", "close error", "bad width", "bad utf8", "external change", "success"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "result.csv")
			if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
				t.Fatal(err)
			}
			revision, _ := Inspect(path)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			index, closed := 0, false
			open := func() (Source, error) {
				return Source{Columns: []string{"N"}, Close: func() error {
					closed = true
					if scenario == "close error" {
						return errors.New("cursor close failure")
					}
					return nil
				}, Next: func() ([]oracle.Cell, error) {
					index++
					if index == 1501 {
						switch scenario {
						case "source error":
							return nil, errors.New("DB disconnected")
						case "cancel":
							cancel()
						case "bad width":
							return []oracle.Cell{}, nil
						case "bad utf8":
							return []oracle.Cell{{Text: "\xff"}}, nil
						case "external change":
							if err := os.WriteFile(path, []byte("external change longer"), 0600); err != nil {
								t.Fatal(err)
							}
						}
					}
					if index > 2000 {
						return nil, io.EOF
					}
					return []oracle.Cell{{Text: fmt.Sprint(index)}}, nil
				}}, nil
			}
			count, err := SaveStream(ctx, path, revision, open, Options{Format: "csv", Unlimited: true})
			if !closed {
				t.Fatal("cursor leaked")
			}
			raw, _ := os.ReadFile(path)
			if scenario == "success" {
				if err != nil || count != 2000 || !bytes.Contains(raw, []byte("2000\n")) {
					t.Fatalf("%d %v", count, err)
				}
			} else {
				want := "existing"
				if scenario == "external change" {
					want = "external change longer"
				}
				if err == nil || count != 0 || string(raw) != want {
					t.Fatalf("partial export replaced file: %d %v %q", count, err, raw)
				}
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 {
				t.Fatal("temporary file or lock leaked")
			}
		})
	}
}

type countingWriter struct {
	calls int
	bytes int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.calls++
	w.bytes += len(p)
	return len(p), nil
}

func TestAllFormatsBatchUnderlyingWrites(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "sql", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			writer := &countingWriter{}
			count, err := WriteStream(context.Background(), writer, generatedSource(10000, "한글 sample value"), Options{Format: format, Headers: true, Unlimited: true, Table: "TARGET"})
			if err != nil || count != 10000 {
				t.Fatalf("%d %v", count, err)
			}
			if writer.bytes == 0 || writer.calls >= 100 {
				t.Fatalf("too many underlying writes: %d for %d rows", writer.calls, count)
			}
		})
	}
}

func TestFinalBufferFlushFailureAndCancellation(t *testing.T) {
	failure := errors.New("final disk flush failed")
	count, err := WriteStream(context.Background(), errorWriter{failure}, generatedSource(3, "short"), Options{Format: "sql", Unlimited: true, Table: "TARGET"})
	if count != 3 || !errors.Is(err, failure) {
		t.Fatalf("final flush failure hidden: %d %v", count, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &countingWriter{}
	_, err = WriteStream(ctx, writer, generatedSource(3, "short"), Options{Format: "csv", Unlimited: true, Progress: func(n int64) {
		if n == 2 {
			cancel()
		}
	}})
	if !errors.Is(err, context.Canceled) || writer.bytes != 0 {
		t.Fatal("cancelled export flushed its remaining buffer", err)
	}
}

func TestCancelAfterBufferedFileWritesPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keep.csv")
	if err := os.WriteFile(path, []byte("keep existing"), 0600); err != nil {
		t.Fatal(err)
	}
	revision, _ := Inspect(path)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closed := false
	source := generatedSource(10000, strings.Repeat("x", 1024))
	source.Close = func() error { closed = true; return nil }
	partialWrite := false
	_, err := SaveStream(ctx, path, revision, func() (Source, error) { return source, nil }, Options{Format: "csv", Unlimited: true, Progress: func(n int64) {
		if n != 3000 {
			return
		}
		// Confirm that cancellation occurs after bytes reached the temporary file,
		// rather than testing only cancellation before the first buffer flush.
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, entry := range entries {
			if entry.Name() == "keep.csv" || strings.HasSuffix(entry.Name(), ".lock") {
				continue
			}
			info, statErr := entry.Info()
			if statErr != nil {
				t.Fatal(statErr)
			}
			partialWrite = partialWrite || info.Size() > 0
		}
		cancel()
	}})
	if !errors.Is(err, context.Canceled) || !closed || !partialWrite {
		t.Fatalf("cancel after disk writes: closed=%v written=%v err=%v", closed, partialWrite, err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "keep existing" {
		t.Fatal("partial export replaced destination")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("partial output/lock leaked")
	}
}
