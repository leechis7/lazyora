package exportdata

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"lazyora/internal/oracle"
	"strings"
	"unicode/utf8"
)

func xmlText(text string) (string, error) {
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("invalid UTF-8 in Excel cell")
	}
	units := 0
	for _, r := range text {
		units++
		if r > 0xffff {
			units++
		}
		if (r < 32 && r != '\n' && r != '\r' && r != '\t') || r == 0xfffe || r == 0xffff {
			return "", fmt.Errorf("character unsupported by Excel XML")
		}
		if units > 32767 {
			return "", fmt.Errorf("Excel cell exceeds 32767 UTF-16 units")
		}
	}
	var out strings.Builder
	if err := xml.EscapeText(&out, []byte(text)); err != nil {
		return "", err
	}
	return out.String(), nil
}
func columnName(index int) string {
	var out string
	for index++; index > 0; index = (index - 1) / 26 {
		out = string(rune('A'+(index-1)%26)) + out
	}
	return out
}

const excelRowsPerSheet = 1048576

func writeXLSX(ctx context.Context, w io.Writer, r *rowReader, options Options) error {
	return writeXLSXSheets(ctx, w, r, options, excelRowsPerSheet)
}

// sheetLimit is the Excel format limit in production. Keeping it as an argument
// lets boundary tests exercise exact sheet rollover without a million-row fixture.
func writeXLSXSheets(ctx context.Context, w io.Writer, r *rowReader, options Options, sheetLimit int) error {
	if len(r.source.Columns) > 16384 || sheetLimit < 2 {
		return fmt.Errorf("Excel row/column limit exceeded")
	}
	z := zip.NewWriter(w)
	defer z.Close()
	var sheet io.Writer
	sheetIndex, rowIndex := 0, 0
	writeRow := func(index int, cells []oracle.Cell) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(sheet, `<row r="%d">`, index); err != nil {
			return err
		}
		for col, cell := range cells {
			if cell.Null {
				continue
			}
			text, err := xmlText(cell.Text)
			if err != nil {
				return fmt.Errorf("sheet %d row %d column %d: %w", sheetIndex, index, col+1, err)
			}
			if _, err = fmt.Fprintf(sheet, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, columnName(col), index, text); err != nil {
				return err
			}
		}
		_, err := io.WriteString(sheet, "</row>")
		return err
	}
	openSheet := func() error {
		if sheet != nil {
			if _, err := io.WriteString(sheet, "</sheetData></worksheet>"); err != nil {
				return err
			}
		}
		sheetIndex++
		var err error
		sheet, err = z.Create(fmt.Sprintf("xl/worksheets/sheet%d.xml", sheetIndex))
		if err != nil {
			return err
		}
		if _, err = io.WriteString(sheet, `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`); err != nil {
			return err
		}
		rowIndex = 0
		if options.Headers {
			headers := make([]oracle.Cell, len(r.source.Columns))
			for i, name := range r.source.Columns {
				headers[i].Text = name
			}
			rowIndex++
			return writeRow(rowIndex, headers)
		}
		return nil
	}
	if err := openSheet(); err != nil {
		return err
	}
	for {
		cells, err := r.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if rowIndex == sheetLimit {
			if err = openSheet(); err != nil {
				return err
			}
		}
		rowIndex++
		if err = writeRow(rowIndex, cells); err != nil {
			return err
		}
		r.written()
	}
	if _, err := io.WriteString(sheet, "</sheetData></worksheet>"); err != nil {
		return err
	}
	var types, book, rels strings.Builder
	types.WriteString(`<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	book.WriteString(`<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	rels.WriteString(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= sheetIndex; i++ {
		name := "Results"
		if i > 1 {
			name = fmt.Sprintf("Results_%d", i)
		}
		fmt.Fprintf(&types, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i)
		fmt.Fprintf(&book, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, name, i, i)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i)
	}
	types.WriteString("</Types>")
	book.WriteString("</sheets></workbook>")
	rels.WriteString("</Relationships>")
	parts := []struct{ name, text string }{
		{"[Content_Types].xml", types.String()},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", book.String()}, {"xl/_rels/workbook.xml.rels", rels.String()},
	}
	for _, part := range parts {
		entry, err := z.Create(part.name)
		if err != nil {
			return err
		}
		if _, err = io.WriteString(entry, part.text); err != nil {
			return err
		}
	}
	return z.Close()
}
