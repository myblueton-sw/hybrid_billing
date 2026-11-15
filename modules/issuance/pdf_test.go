package issuance

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func catalog(extra string) []byte {
	return build(extra, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>", stream("/Filter /FlateDecode", "BT (x) Tj ET"))
}

func TestGateAcceptsPlainAndBinaryDocuments(t *testing.T) {
	binary := "\x00\xff obj /URI /JS http://x file: 9 0 obj << /S /URI >> endobj %%EOF ( < [ /A#41"
	cases := map[string][]byte{
		"plain":        catalog(""),
		"binary body":  build("", stream("/Filter [/FlateDecode /DCTDecode] /DecodeParms << /Columns 4 >>", binary)),
		"strings":      build("/Title (http://example.com /URI \\) (nested) %%EOF startxref) /Hex <687474703a> /Arr [1 2.5 -3 true null]"),
		"pdf 2":        append([]byte("%PDF-2.0"), build("")[8:]...),
		"no final eol": bytes.TrimSuffix(catalog(""), []byte("\n")),
		"crlf table":   []byte(strings.Replace(string(catalog("")), "xref\n0 5\n", "xref\r\n0 5\r\n", 1)),
		"cr final":     append(bytes.TrimSuffix(catalog(""), []byte("\n")), '\r'),
	}
	for label, data := range cases {
		if got := inspect(data); got != "" {
			t.Errorf("%s: %s", label, got)
		}
	}
}

func TestGateRefusesActionsLinksAndHiddenObjects(t *testing.T) {
	for name := range refusedNames {
		if got := inspect(catalog("/" + name + " 1")); got != ReasonUnsafeOutput {
			t.Errorf("%s: %s", name, got)
		}
	}
	for label, extra := range map[string]string{
		"escaped name": "/J#53 (x)",
		"open action":  "/OpenAction << /S /JavaScript /JS (app.alert(1)) >>",
		"annotations":  "/Annots [<< /Subtype /Link /A << /S /URI /URI (http://x) >> >>]",
	} {
		if got := inspect(catalog(extra)); got != ReasonUnsafeOutput {
			t.Errorf("%s: %s", label, got)
		}
	}
	trailer := bytes.Replace(catalog(""), []byte("/Root 1 0 R"), []byte("/Root 1 0 R /Prev 9"), 1)
	if got := inspect(trailer); got != ReasonUnsafeOutput {
		t.Errorf("prev: %s", got)
	}
	encrypted := bytes.Replace(catalog(""), []byte("/Root 1 0 R"), []byte("/Root 1 0 R /Encrypt 5 0 R"), 1)
	if got := inspect(encrypted); got != ReasonUnsafeOutput {
		t.Errorf("encrypt: %s", got)
	}
	if got := inspect(build("", "<< /Prev 1 /Next 2 >>")); got != ReasonUnsafeOutput {
		t.Errorf("prev anywhere: %s", got)
	}
	if got := inspect(build("", stream("/F (\\\\host\\share)", "x"))); got != ReasonUnsafeOutput {
		t.Errorf("external stream data: %s", got)
	}
	for label, filter := range map[string]string{
		"lzw": "/LZWDecode", "mixed": "[/FlateDecode /LZWDecode]", "number": "5", "open array": "[/FlateDecode", "nested": "[[/FlateDecode]]",
	} {
		if got := inspect(build("", stream("/Filter "+filter, "x"))); got != ReasonUnsafeOutput && got != ReasonMalformedOutput {
			t.Errorf("filter %s: %s", label, got)
		}
	}
	for _, o := range []string{"/Filter", "/Filter [/FlateDecode"} {
		if got := inspect(build("", o)); got != ReasonUnsafeOutput {
			t.Errorf("%s: %s", o, got)
		}
	}
	if got := inspect(build("", stream("/Filter /LZWDecode", "x"))); got != ReasonUnsafeOutput {
		t.Errorf("filter allow-list: %s", got)
	}
	if got := inspect(build("/Metadata << /Filter /JPXDecode >>")); got != ReasonUnsafeOutput {
		t.Errorf("filter outside a stream: %s", got)
	}
}

func TestGateRefusesMalformedStructure(t *testing.T) {
	plain := catalog("")
	text := string(plain)
	x := strings.Index(text, "xref\n")
	first := strings.Index(text, "1 0 obj")
	body := strings.Index(text, "BT (x)")
	cases := map[string][]byte{
		"empty":             nil,
		"header":            []byte(strings.Replace(text, "%PDF-1.7", "%PDF-3.0", 1)),
		"trailing":          append(append([]byte(nil), plain...), []byte("junk")...),
		"two eof":           append(append([]byte(nil), plain...), []byte("%%EOF")...),
		"eof early":         []byte(strings.Replace(text, "xref\n", "%%EOF\nxref\n", 1)),
		"no xref":           []byte(text[:x] + "trailer\n<< >>\nstartxref\n0\n%%EOF\n"),
		"startxref":         []byte(strings.Replace(text, fmt.Sprintf("startxref\n%d\n", x), fmt.Sprintf("startxref\n%d\n", x+1), 1)),
		"startxref nan":     []byte(strings.Replace(text, fmt.Sprintf("startxref\n%d\n", x), "startxref\nabc\n", 1)),
		"offset":            []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d 00000 n", body), 1)),
		"generation":        []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d 00001 n", first), 1)),
		"free in use":       []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d 00000 f", first), 1)),
		"zero in use":       []byte(strings.Replace(text, "0000000000 65535 f", "0000000000 65535 n", 1)),
		"entry form":        []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n \n", first), fmt.Sprintf("%010d 00000 n  ", first), 1)),
		"entry kind":        []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d 00000 x", first), 1)),
		"entry digits":      []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%09dx 00000 n", first/10), 1)),
		"entry space":       []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d-00000 n", first), 1)),
		"entry space2":      []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d 00000-n", first), 1)),
		"gen digits":        []byte(strings.Replace(text, fmt.Sprintf("%010d 00000 n", first), fmt.Sprintf("%010d 0000x n", first), 1)),
		"count":             []byte(strings.Replace(text, "xref\n0 5\n", "xref\n0 6\n", 1)),
		"count low":         []byte(strings.Replace(text, "xref\n0 5\n", "xref\n0 0\n", 1)),
		"subsection":        []byte(strings.Replace(text, "xref\n0 5\n", "xref\n1 5\n", 1)),
		"header short":      []byte(strings.Replace(text, "xref\n0 5\n", "xref\n0\n", 1)),
		"xref tail":         []byte(strings.Replace(text, "xref\n0 5\n", "xref 1\n0 5\n", 1)),
		"no header eol":     []byte(text[:x] + "xref"),
		"short table":       []byte(text[:x] + "xref\n0 5\n0000000000 65535 f \n"),
		"two xref":          []byte(strings.Replace(text, "trailer", "xref\n0 1\n0000000000 65535 f \ntrailer", 1)),
		"two trailer":       []byte(strings.Replace(text, "startxref", "trailer\nstartxref", 1)),
		"trailer early":     []byte(strings.Replace(text, "1 0 obj", "trailer\n1 0 obj", 1)),
		"startxref early":   []byte(strings.Replace(text, "1 0 obj", "startxref\n1 0 obj", 1)),
		"startxref in dict": []byte(strings.Replace(text, "/Size 5", "/Size << startxref", 1)),
		"stray token":       []byte(strings.Replace(text, "xref\n", "extra\nxref\n", 1)),
		"stray between":     []byte(strings.Replace(text, "1 0 obj", "1 0 7 obj", 1)),
		"after table":       []byte(strings.Replace(text, "trailer", "stray trailer", 1)),
		"xref in object":    []byte(strings.Replace(text, "/Type /Catalog", "/Type /Catalog xref", 1)),
		"trailer in object": []byte(strings.Replace(text, "/Type /Catalog", "/Type trailer", 1)),
		"object number":     []byte(strings.Replace(text, "1 0 obj", "a 0 obj", 1)),
		"object gen":        []byte(strings.Replace(text, "1 0 obj", "1 b obj", 1)),
		"nested object":     []byte(strings.Replace(text, "/Type /Catalog", "/Type 7 0 obj", 1)),
		"endobj alone":      []byte(strings.Replace(text, "xref\n", "endobj\nxref\n", 1)),
		"endobj in dict":    []byte(strings.Replace(text, "/Type /Catalog", "/Type endobj", 1)),
		"unclosed dict":     []byte(strings.Replace(text, "/Type /Catalog", "<< /Type /Catalog", 1)),
		"closing":           []byte(strings.Replace(text, "/Type /Catalog", "/Type > /Catalog", 1)),
		"close outside":     []byte(strings.Replace(text, "xref\n", ">>\nxref\n", 1)),
		"paren":             []byte(strings.Replace(text, "/Type /Catalog", "/Type ) /Catalog", 1)),
		"open string":       []byte(text[:x] + "9 0 obj (abc"),
		"open hex":          []byte(text[:x] + "9 0 obj <abc"),
		"stream outside":    []byte(strings.Replace(text, "xref\n", "stream\nxref\n", 1)),
		"duplicate":         build("", "<< >>"),
		"truncated":         []byte("%PDF-1.7\n1 0 obj\n<< >>\nendobj\n"),
		"comment startxref": []byte(strings.Replace(text, "%%EOF", "%startxref 9\n%%EOF", 1)),
		"comment eof":       []byte(strings.Replace(text, "1 0 obj", "% EOF marker\n1 0 obj", 1)),
		"tail comment":      []byte(strings.Replace(text, "\n%%EOF", "\n% x\n%%EOF", 1)),
		"tail space":        []byte(strings.Replace(text, "\n%%EOF", " \n%%EOF", 1)),
		"no eof":            []byte(strings.Replace(text, "%%EOF", "", 1)),
		"missing object":    []byte(strings.Replace(text, "/Contents 4 0 R", "/Contents 9 0 R", 1)),
		"wrong generation":  []byte(strings.Replace(text, "/Contents 4 0 R", "/Contents 4 1 R", 1)),
		"loose reference":   []byte(strings.Replace(text, "/Contents 4 0 R", "/Contents /X 0 R", 1)),
		"bare reference":    []byte(strings.Replace(text, "<< /Size 5 /Root 1 0 R >>", "R << /Size 5 /Root 1 0 R >>", 1)),
		"no root":           []byte(strings.Replace(text, "/Root 1 0 R", "/Info 1 0 R", 1)),
		"root not catalog":  []byte(strings.Replace(text, "/Root 1 0 R", "/Root 3 0 R", 1)),
		"size":              []byte(strings.Replace(text, "/Size 5", "/Size 4", 1)),
		"no size":           []byte(strings.Replace(text, "/Size 5", "/Count 5", 1)),
		"root value":        []byte(strings.Replace(text, "/Root 1 0 R", "/Root 1", 1)),
		"catalog type":      []byte(strings.Replace(text, "/Type /Catalog", "/Type /Page", 1)),
		"catalog untyped":   []byte(strings.Replace(text, "/Type /Catalog", "/Kind /Catalog", 1)),
		"no pages":          []byte(strings.Replace(text, "/Pages 2 0 R", "/Outlines 2 0 R", 1)),
		"pages type":        []byte(strings.Replace(text, "/Type /Pages", "/Type /Page", 1)),
		"wrapped catalog":   []byte(strings.Replace(text, "<< /Type /Catalog /Pages 2 0 R  >>", "[<< /Type /Catalog /Pages 2 0 R >>]", 1)),
		"empty trailer":     []byte(strings.Replace(text, "<< /Size 5 /Root 1 0 R >>", "", 1)),
		"second dictionary": []byte(strings.Replace(text, "/Type /Catalog /Pages 2 0 R  >>", "/Type /Catalog >> << /Pages 2 0 R >>", 1)),
		"two roots":         []byte(strings.Replace(text, "/Root 1 0 R", "/Root 1 0 R /Root 3 0 R", 1)),
		"two types":         []byte(strings.Replace(text, "/Type /Catalog", "/Type /Catalog /Type /Catalog", 1)),
		"short root":        []byte(strings.Replace(text, "/Root 1 0 R >>", "/Root 1 0 >>", 1)),
		"type last":         []byte(strings.Replace(text, "<< /Type /Pages /Kids [] /Count 0 >>", "<< /Kids [] /Count 0 /Type >>", 1)),
	}
	duplicate := string(cases["duplicate"])
	cases["duplicate"] = []byte(strings.Replace(duplicate, "3 0 obj", "1 0 obj", 1))
	extra := strings.Index(text, "2 0 obj")
	cases["unlisted object"] = []byte(text[:extra] + "9 0 obj\n<< >>\nendobj\n" + text[extra:])
	for label, data := range cases {
		if got := inspect(data); got != ReasonMalformedOutput {
			t.Errorf("%s: %s", label, got)
		}
	}
	if got := inspect(bytes.Repeat([]byte("%"), maxFileBytes+1)); got != ReasonMalformedOutput {
		t.Errorf("size: %s", got)
	}
}

func TestGateChecksStreamBodies(t *testing.T) {
	lengths := map[string]string{
		"missing":       "<< /Filter /FlateDecode >>\nstream\nabc\nendstream",
		"indirect":      "<< /Length 5 0 R >>\nstream\nabc\nendstream",
		"not a number":  "<< /Length /X >>\nstream\nabc\nendstream",
		"dangling":      "<< /Length",
		"too long":      "<< /Length 4000 >>\nstream\nabc\nendstream",
		"short":         "<< /Length 2 >>\nstream\nabc\nendstream",
		"early end":     "<< /Length 20 >>\nstream\nab\nendstream\nxxxxxx\nendstream",
		"no eol":        "<< /Length 3 >>\nstream abc\nendstream",
		"glued":         "<< /Length 3 >>\nstream\nabc\nendstreamX",
		"no dict":       "[1 2]\nstream\nabc\nendstream",
		"nested length": "<< /D << /Length 3 >> >>\nstream\nabc\nendstream",
		"two lengths":   "<< /Length 3 /Length 3 >>\nstream\nabc\nendstream",
		"two streams":   "<< /Length 3 >>\nstream\nabc\nendstream << /Length 3 >>\nstream\nabc\nendstream",
		"name length":   "<< /Length /3 >>\nstream\nabc\nendstream",
	}
	for label, o := range lengths {
		if got := inspect(build("", o)); got != ReasonMalformedOutput {
			t.Errorf("%s: %s", label, got)
		}
	}
	for label, o := range map[string]string{
		"crlf":       "<< /Length 3 >>\r\nstream\r\nabc\r\nendstream",
		"cr":         "<< /Length 3 >>\nstream\nabc\rendstream",
		"no eol end": "<< /Length 3 >>\nstream\nabcendstream",
		"glued dict": "<< /Length 3 >>\nstream\nabc\nendstream\n",
		"length 0 R": "<< /Length 3 /X 2 0 R >>\nstream\nabc\nendstream",
		"pair":       "<< /Length 3 /N 4 >>\nstream\nabc\nendstream",
	} {
		if got := inspect(build("", o)); got != "" {
			t.Errorf("%s: %s", label, got)
		}
	}
	if got := inspect(build("", "<< /Length 3 >>\nstream\nabc\nendstream")); got != "" {
		t.Errorf("plain: %s", got)
	}
}
