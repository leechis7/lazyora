// Package exportdata writes the loaded result snapshot; it never queries the DB.
package exportdata

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"

	"lazyora/internal/oracle"
)

const MaxBytes = 64 * 1024 * 1024

type Dataset struct {
	Columns []string
	Rows    [][]oracle.Cell
}
type Options struct {
	Format                  string
	Headers, BOM            bool
	NullText, Schema, Table string
	Columns                 []string
}

type checkedWriter struct {
	ctx     context.Context
	writer  io.Writer
	written int
	err     error
}

func (w *checkedWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.err = w.ctx.Err(); w.err != nil {
		return 0, w.err
	}
	if len(p) > MaxBytes-w.written {
		w.err = fmt.Errorf("export exceeds 64 MiB limit")
		return 0, w.err
	}
	n, err := w.writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.written += n
	w.err = err
	return n, err
}

func Write(ctx context.Context, writer io.Writer, data Dataset, options Options) error {
	if len(data.Columns) == 0 {
		return fmt.Errorf("no result columns")
	}
	for _, name := range data.Columns {
		if !utf8.ValidString(name) {
			return fmt.Errorf("invalid UTF-8 column name")
		}
	}
	for _, row := range data.Rows {
		if len(row) != len(data.Columns) {
			return fmt.Errorf("result row width does not match columns")
		}
		for _, cell := range row {
			if !utf8.ValidString(cell.Text) {
				return fmt.Errorf("invalid UTF-8 result value")
			}
		}
	}
	w := &checkedWriter{ctx: ctx, writer: writer}
	var err error
	switch options.Format {
	case "csv", "tsv":
		if options.BOM {
			_, err = io.WriteString(w, "\ufeff")
			if err != nil {
				return err
			}
		}
		csvWriter := csv.NewWriter(w)
		if options.Format == "tsv" {
			csvWriter.Comma = '\t'
		}
		if options.Headers {
			if err = csvWriter.Write(data.Columns); err != nil {
				return err
			}
		}
		for _, row := range data.Rows {
			if err = ctx.Err(); err != nil {
				return err
			}
			values := make([]string, len(row))
			for i, value := range row {
				values[i] = value.Text
				if value.Null {
					values[i] = options.NullText
				}
			}
			if err = csvWriter.Write(values); err != nil {
				return err
			}
		}
		csvWriter.Flush()
		err = csvWriter.Error()
	case "xlsx":
		err = writeXLSX(ctx, w, data, options)
	case "sql":
		err = writeSQL(ctx, w, data, options)
	default:
		return fmt.Errorf("unsupported export format")
	}
	if err != nil {
		return err
	}
	return w.err
}
