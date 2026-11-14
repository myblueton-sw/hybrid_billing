package exports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"strconv"
	"strings"
	"time"
)

type sheet struct {
	name string
	rows [][]string
}

var zipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func escape(text string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(text))
	return b.String()
}

func column(index int) string {
	name := ""
	for index++; index > 0; index = (index - 1) / 26 {
		name = string(rune('A'+(index-1)%26)) + name
	}
	return name
}

func sheetXML(rows [][]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		b.WriteString(`<row r="` + strconv.Itoa(r+1) + `">`)
		for c, value := range row {
			style := "1"
			if risky(value) {
				style = "2"
			}
			b.WriteString(`<c r="` + column(c) + strconv.Itoa(r+1) + `" s="` + style + `" t="inlineStr"><is><t xml:space="preserve">`)
			b.WriteString(escape(clean(value)))
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func workbook(sheets []sheet) []byte {
	var types, wb, rels strings.Builder
	types.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	wb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	rels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i, s := range sheets {
		n := strconv.Itoa(i + 1)
		types.WriteString(`<Override PartName="/xl/worksheets/sheet` + n + `.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`)
		wb.WriteString(`<sheet name="` + escape(s.name) + `" sheetId="` + n + `" r:id="rId` + n + `"/>`)
		rels.WriteString(`<Relationship Id="rId` + n + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet` + n + `.xml"/>`)
	}
	styles := strconv.Itoa(len(sheets) + 1)
	rels.WriteString(`<Relationship Id="rId` + styles + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`)
	types.WriteString(`</Types>`)
	wb.WriteString(`</sheets></workbook>`)
	parts := [][2]string{
		{"[Content_Types].xml", types.String()},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", wb.String()},
		{"xl/_rels/workbook.xml.rels", rels.String()},
		{"xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
			`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font/></fonts><fills count="1"><fill/></fills>` +
			`<borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs>` +
			`<cellXfs count="3"><xf/><xf numFmtId="49" applyNumberFormat="1"/><xf numFmtId="49" applyNumberFormat="1" quotePrefix="1"/></cellXfs></styleSheet>`},
	}
	for i, s := range sheets {
		parts = append(parts, [2]string{"xl/worksheets/sheet" + strconv.Itoa(i+1) + ".xml", sheetXML(s.rows)})
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, p := range parts {
		f, _ := w.CreateHeader(&zip.FileHeader{Name: p[0], Method: zip.Store, Modified: zipTime})
		f.Write([]byte(p[1]))
	}
	w.Close()
	return out.Bytes()
}

func csvBytes(rows [][]string, viewing bool) []byte {
	var out bytes.Buffer
	if viewing {
		out.WriteString("\ufeff")
	}
	w := csv.NewWriter(&out)
	for _, row := range rows {
		values := row
		if viewing {
			values = make([]string, len(row))
			for i, v := range row {
				values[i] = neutral(clean(v))
			}
		}
		w.Write(values)
	}
	w.Flush()
	return out.Bytes()
}
