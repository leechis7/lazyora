package exportdata

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
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

func writeXLSX(ctx context.Context, w io.Writer, data Dataset, options Options) error {
	rows := len(data.Rows)
	if options.Headers {
		rows++
	}
	if rows > 1048576 || len(data.Columns) > 16384 {
		return fmt.Errorf("Excel row/column limit exceeded")
	}
	z := zip.NewWriter(w)
	parts := []struct{ name, text string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Results" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`},
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
	sheet, err := z.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	if _, err = io.WriteString(sheet, `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`); err != nil {
		return err
	}
	writeRow := func(index int, values []string, nulls []bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(sheet, `<row r="%d">`, index); err != nil {
			return err
		}
		for col, value := range values {
			if len(nulls) > col && nulls[col] {
				continue
			}
			text, err := xmlText(value)
			if err != nil {
				return fmt.Errorf("row %d column %d: %w", index, col+1, err)
			}
			if _, err = fmt.Fprintf(sheet, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, columnName(col), index, text); err != nil {
				return err
			}
		}
		_, err := io.WriteString(sheet, "</row>")
		return err
	}
	index := 1
	if options.Headers {
		if err = writeRow(index, data.Columns, nil); err != nil {
			return err
		}
		index++
	}
	for _, row := range data.Rows {
		values := make([]string, len(row))
		nulls := make([]bool, len(row))
		for col, cell := range row {
			values[col], nulls[col] = cell.Text, cell.Null
		}
		if err = writeRow(index, values, nulls); err != nil {
			return err
		}
		index++
	}
	if _, err = io.WriteString(sheet, "</sheetData></worksheet>"); err != nil {
		return err
	}
	return z.Close()
}
