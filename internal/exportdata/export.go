// Package exportdata streams result rows to files without retaining the result.
package exportdata

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"

	"lazyora/internal/oracle"
)

// Loaded snapshots retain their original cap; explicit full-query exports opt
// out because rows are streamed rather than accumulated in memory.
const MaxBytes = 64 * 1024 * 1024

// Buffer all formats, including row-oriented INSERT output. The buffer size
// is independent of the result size; only a successful export flushes it.
const outputBufferSize = 256 * 1024

type Dataset struct {
	Columns []string
	Rows    [][]oracle.Cell
}
type Options struct {
	Format                  string
	Headers, BOM            bool
	NullText, Schema, Table string
	Columns                 []string
	Unlimited               bool
	Progress                func(int64)
}

// Source.Next returns io.EOF only after a complete result; other errors abort
// the save. SaveStream owns Close, including failure/cancellation paths.
type Source struct {
	Columns []string
	Next    func() ([]oracle.Cell, error)
	Close   func() error
}

func FromDataset(data Dataset) Source {
	index := 0
	return Source{Columns: data.Columns, Next: func() ([]oracle.Cell, error) {
		if index == len(data.Rows) {
			return nil, io.EOF
		}
		row := data.Rows[index]
		index++
		return row, nil
	}}
}

type checkedWriter struct {
	ctx       context.Context
	writer    io.Writer
	written   int64
	unlimited bool
	err       error
}

func (w *checkedWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.err = w.ctx.Err(); w.err != nil {
		return 0, w.err
	}
	if !w.unlimited && int64(len(p)) > MaxBytes-w.written {
		w.err = fmt.Errorf("export exceeds 64 MiB limit")
		return 0, w.err
	}
	n, err := w.writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.written += int64(n)
	w.err = err
	return n, err
}

type rowReader struct {
	ctx      context.Context
	source   Source
	count    int64
	progress func(int64)
}

func (r *rowReader) next() ([]oracle.Cell, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	row, err := r.source.Next()
	if err != nil {
		return nil, err
	}
	if err = r.ctx.Err(); err != nil {
		return nil, err
	}
	if len(row) != len(r.source.Columns) {
		return nil, fmt.Errorf("result row width does not match columns")
	}
	for _, cell := range row {
		if cell.Partial {
			return nil, fmt.Errorf("result contains a LOB preview; save the individual cell or use Full DB export within the cell limit")
		}
		if !utf8.ValidString(cell.Text) {
			return nil, fmt.Errorf("invalid UTF-8 result value")
		}
	}
	return row, nil
}
func (r *rowReader) written() {
	r.count++
	if r.progress != nil {
		r.progress(r.count)
	}
}

func Write(ctx context.Context, writer io.Writer, data Dataset, options Options) error {
	_, err := WriteStream(ctx, writer, FromDataset(data), options)
	return err
}

// WriteStream doesn't close the source; SaveStream does. Memory usage is
// bounded by the current row, encoder buffers and XLSX sheet metadata.
func WriteStream(ctx context.Context, writer io.Writer, source Source, options Options) (int64, error) {
	if len(source.Columns) == 0 || source.Next == nil {
		return 0, fmt.Errorf("no result columns/source")
	}
	for _, name := range source.Columns {
		if !utf8.ValidString(name) {
			return 0, fmt.Errorf("invalid UTF-8 column name")
		}
	}
	buffer := bufio.NewWriterSize(writer, outputBufferSize)
	w := &checkedWriter{ctx: ctx, writer: buffer, unlimited: options.Unlimited}
	reader := &rowReader{ctx: ctx, source: source, progress: options.Progress}
	var err error
	switch options.Format {
	case "csv", "tsv":
		err = writeDelimited(w, reader, options)
	case "xlsx":
		err = writeXLSX(ctx, w, reader, options)
	case "sql":
		err = writeSQL(ctx, w, reader, options)
	default:
		err = fmt.Errorf("unsupported export format")
	}
	if err == nil {
		err = w.err
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = buffer.Flush()
	}
	return reader.count, err
}
func writeDelimited(w io.Writer, r *rowReader, options Options) error {
	if options.BOM {
		if _, err := io.WriteString(w, "\ufeff"); err != nil {
			return err
		}
	}
	writer := csv.NewWriter(w)
	if options.Format == "tsv" {
		writer.Comma = '\t'
	}
	if options.Headers {
		if err := writer.Write(r.source.Columns); err != nil {
			return err
		}
	}
	for {
		row, err := r.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		values := make([]string, len(row))
		for i, value := range row {
			values[i] = value.Text
			if value.Null {
				values[i] = options.NullText
			}
		}
		if err = writer.Write(values); err != nil {
			return err
		}
		r.written()
	}
	writer.Flush()
	return writer.Error()
}
