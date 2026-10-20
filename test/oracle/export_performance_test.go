package oracle_test

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"lazyora/internal/exportdata"
	"lazyora/internal/oracle"
)

// Each performance case runs in a fresh test process via export-bench.sh so
// process RSS high-water marks do not leak across formats or result sizes.
func TestOracleExportPerformance(t *testing.T) {
	if os.Getenv("LAZYORA_EXPORT_BENCH") != "1" {
		t.Skip("run make -C test/oracle export-bench")
	}
	count, err := strconv.Atoi(os.Getenv("LAZYORA_BENCH_ROWS"))
	if err != nil || count <= 0 {
		t.Fatal("invalid row count")
	}
	format := os.Getenv("LAZYORA_BENCH_FORMAT")
	sourceKind := os.Getenv("LAZYORA_BENCH_SOURCE")
	text := "한글 O'Reilly\n" + strings.Repeat("sample-data-", 8)
	columns := []string{"ID", "TEXT_VALUE", "OPTIONAL_VALUE"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var open func() (exportdata.Source, error)
	switch sourceKind {
	case "generated":
		index := 0
		open = func() (exportdata.Source, error) {
			return exportdata.Source{Columns: columns, Next: func() ([]oracle.Cell, error) {
				if index == count {
					return nil, io.EOF
				}
				index++
				return []oracle.Cell{{Text: strconv.Itoa(index), Type: "NUMBER"}, {Text: text, Type: "VARCHAR2"}, {Text: "NULL", Type: "VARCHAR2", Null: index%2 == 0}}, nil
			}}, nil
		}
	case "oracle":
		session := workflowSession(t)
		spec, err := oracle.NewQuerySpec("SELECT level AS id, CAST(:text AS VARCHAR2(256)) AS text_value, CASE WHEN mod(level,2)=0 THEN NULL ELSE 'NULL' END AS optional_value FROM dual CONNECT BY level <= :n", map[string]oracle.BindValue{"TEXT": {Kind: "text", Text: text}, "N": {Kind: "number", Text: strconv.Itoa(count)}})
		if err != nil {
			t.Fatal(err)
		}
		open = func() (exportdata.Source, error) {
			rows, err := session.OpenQuery(ctx, spec)
			if err != nil {
				return exportdata.Source{}, err
			}
			return exportdata.Source{Columns: rows.Columns, Next: rows.Next, Close: rows.Close}, nil
		}
	default:
		t.Fatal("invalid benchmark source")
	}
	path := filepath.Join(t.TempDir(), "export."+format)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	stop := make(chan struct{})
	peakResult := make(chan uint64, 1)
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		peak := before.HeapAlloc
		for {
			var stats runtime.MemStats
			select {
			case <-ticker.C:
				runtime.ReadMemStats(&stats)
				if stats.HeapAlloc > peak {
					peak = stats.HeapAlloc
				}
			case <-stop:
				runtime.ReadMemStats(&stats)
				if stats.HeapAlloc > peak {
					peak = stats.HeapAlloc
				}
				peakResult <- peak
				return
			}
		}
	}()
	// Match UI's throttling work without including terminal rendering or setup.
	lastProgress := time.Time{}
	options := exportdata.Options{Format: format, Headers: true, BOM: true, NullText: `\N`, Table: "EXPORT_BENCH", Columns: columns, Unlimited: true, Progress: func(_ int64) {
		if time.Since(lastProgress) >= 250*time.Millisecond {
			lastProgress = time.Now()
		}
	}}
	started := time.Now()
	saved, err := exportdata.SaveStream(ctx, path, exportdata.Revision{}, open, options)
	elapsed := time.Since(started)
	close(stop)
	peak := <-peakResult
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if err != nil || saved != int64(count) {
		t.Fatalf("saved %d: %v", saved, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	metrics := map[string]any{"source": sourceKind, "format": format, "rows": saved, "seconds": elapsed.Seconds(), "file_bytes": info.Size(), "rows_per_second": float64(saved) / elapsed.Seconds(), "peak_heap_mib": float64(peak) / (1024 * 1024), "total_alloc_mib": float64(after.TotalAlloc-before.TotalAlloc) / (1024 * 1024), "max_rss_mib": processMaxRSSMiB()}
	// Capture RSS before validation; XML/CSV validation itself isn't export work.
	validatePerformanceFile(t, path, format, count)
	encoded, _ := json.Marshal(metrics)
	fmt.Println("LAZYORA_EXPORT_METRICS " + string(encoded))
}

func processMaxRSSMiB() float64 {
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return -1
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "VmHWM:" {
			value, _ := strconv.ParseFloat(fields[1], 64)
			return value / 1024
		}
	}
	return -1
}

func validatePerformanceFile(t *testing.T, path, format string, want int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	count := 0
	switch format {
	case "csv", "tsv":
		reader := csv.NewReader(file)
		if format == "tsv" {
			reader.Comma = '\t'
		}
		if _, err = reader.Read(); err != nil {
			t.Fatal(err)
		}
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			count++
			if record[0] != strconv.Itoa(count) {
				t.Fatal("missing/reordered record")
			}
		}
	case "sql":
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "INSERT INTO ") {
				count++
			}
		}
		if err = scanner.Err(); err != nil {
			t.Fatal(err)
		}
	case "xlsx":
		info, _ := file.Stat()
		archive, err := zip.NewReader(file, info.Size())
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range archive.File {
			if !strings.HasPrefix(part.Name, "xl/worksheets/sheet") {
				continue
			}
			reader, err := part.Open()
			if err != nil {
				t.Fatal(err)
			}
			decoder := xml.NewDecoder(reader)
			sheetRows := 0
			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					reader.Close()
					t.Fatal(err)
				}
				if element, ok := token.(xml.StartElement); ok && element.Name.Local == "row" {
					sheetRows++
				}
			}
			reader.Close()
			count += sheetRows - 1
		}
	default:
		t.Fatal("unsupported format")
	}
	if count != want {
		t.Fatalf("file contains %d rows, want %d", count, want)
	}
}
