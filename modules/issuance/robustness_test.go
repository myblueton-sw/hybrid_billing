package issuance

import (
	"bytes"
	"strings"
	"testing"
)

func rich() []byte {
	return build("/Metadata 5 0 R\x00/Lang (ko\\)x) /Hex <6b6f> /Arr [1 [2] {3}] /Nested << /Deep << /K 1 >> >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>",
		stream("/Filter [/FlateDecode] /Length2 9", "BT\n(obj > << endobj) Tj\nET"),
		stream("/Subtype /XML", "<x/>"))
}

func TestGateNeverPanicsOnDamagedFiles(t *testing.T) {
	data := rich()
	if got := inspect(data); got != "" {
		t.Fatalf("rich: %s", got)
	}
	for end := 0; end < len(data)-2; end++ {
		if got := inspect(data[:end]); got == "" {
			t.Fatalf("prefix %d accepted", end)
		}
	}
	for index := range data {
		for _, b := range []byte{0, '>', '<', '(', ')', '/', '%', '\n', 'R', '#', '0', '9', '[', ']', ' ', 'x'} {
			if data[index] == b {
				continue
			}
			changed := append([]byte(nil), data...)
			changed[index] = b
			inspect(changed)
		}
	}
}

func TestGateBoundaries(t *testing.T) {
	plain := string(catalog(""))
	x := strings.Index(plain, "xref\n")
	cases := map[string]struct {
		data []byte
		want string
	}{
		"reference as a whole object":  {build("", "2 0 R"), ""},
		"generation not a number":      {build("", "1 /X R"), ReasonMalformedOutput},
		"nul whitespace":               {build("\x00/Lang (x)"), ""},
		"nul before catalog":           {[]byte(strings.Replace(plain, "1 0 obj\n<<", "1 0 obj\nx<<", 1)), ReasonMalformedOutput},
		"hash inside a name":           {catalog("/Ab#20c 1"), ReasonUnsafeOutput},
		"hash first":                   {catalog("/#41 1"), ReasonUnsafeOutput},
		"single close in a dictionary": {[]byte(strings.Replace(plain, "/Type /Catalog", "/Type /Catalog > ", 1)), ReasonMalformedOutput},
		"empty hex":                    {build("/H <>"), ""},
		"hex then close":               {build("/H <6b>>"), ReasonMalformedOutput},
		"three pending numbers":        {[]byte(strings.Replace(plain, "1 0 obj", "7 1 0 obj", 1)), ReasonMalformedOutput},
		"pending before xref":          {[]byte(strings.Replace(plain, "xref\n", "7 xref\n", 1)), ReasonMalformedOutput},
		"startxref in an object":       {[]byte(strings.Replace(plain, "/Type /Catalog", "/Type /Catalog startxref", 1)), ReasonMalformedOutput},
		"object in the trailer":        {[]byte(strings.Replace(plain, "trailer\n", "trailer\n1 0 obj ", 1)), ReasonMalformedOutput},
		"endstream glued":              {build("", "<< /Length 3 >>\nstream\nabc\nendstream/X"), ""},
		"length is the rest":           {build("", "<< /Length 4 >>\nstream\nabc\nendstream"), ""},
		"length to the end":            {[]byte(plain[:x-1] + "\n5 0 obj\n<< /Length 999999 >>\nstream\n"), ReasonMalformedOutput},
		"indirect length name":         {build("", "<< /Length 3 /X /Y R >>\nstream\nabc\nendstream"), ReasonMalformedOutput},
		"length then R name":           {build("", "<< /Length 3 1 /R >>\nstream\nabc\nendstream"), ""},
		"stream after array":           {build("", "<< /Length 3 >> [1]\nstream\nabc\nendstream"), ReasonMalformedOutput},
		"stream at object start":       {build("", "stream\nabc\nendstream"), ReasonMalformedOutput},
		"stream inside dictionary":     {build("", "<< /Length 3 << /X 1 >> stream\nabc\nendstream >>"), ReasonMalformedOutput},
		"table count one":              {[]byte("%PDF-1.7\nxref\n0 1\n0000000000 65535 f \ntrailer\n<< /Size 1 >>\nstartxref\n9\n%%EOF"), ReasonMalformedOutput},
		"cr entry":                     {[]byte(strings.Replace(plain, "65535 f \n", "65535 f \r", 1)), ""},
		"crlf entry":                   {[]byte(strings.Replace(plain, "65535 f \n", "65535 f\r\n", 1)), ""},
		"lf only entry":                {[]byte(strings.Replace(plain, "65535 f \n", "65535 f\n\n", 1)), ReasonMalformedOutput},
		"short entry":                  {[]byte(strings.Replace(plain, "65535 f \n", "65535 f\n", 1)), ReasonMalformedOutput},
		"entry offset digit":           {[]byte(strings.Replace(plain, "0000000000 65535 f", "x000000000 65535 f", 1)), ReasonMalformedOutput},
		"entry generation digit":       {[]byte(strings.Replace(plain, "0000000000 65535 f", "0000000000 x5535 f", 1)), ReasonMalformedOutput},
		"startxref crlf":               {[]byte(strings.Replace(plain, "startxref\n", "startxref\r\n", 1)), ""},
		"startxref no eol":             {[]byte(strings.Replace(plain, "startxref\n", "startxref ", 1)), ReasonMalformedOutput},
		"eof crlf":                     {[]byte(strings.Replace(plain, "\n%%EOF", "\r\n%%EOF", 1)), ""},
		"trailer in a dictionary":      {[]byte(strings.Replace(plain, "trailer\n<<", "<< trailer", 1)), ReasonMalformedOutput},
		"startxref in trailer dict":    {[]byte(strings.Replace(plain, "/Size 5", "/Size 5 << startxref", 1)), ReasonMalformedOutput},
		"first entry used":             {[]byte(strings.Replace(plain, "0000000000 65535 f", "0000000000 65535 n", 1)), ReasonMalformedOutput},
		"loose first number":           {[]byte(strings.Replace(plain, "/Contents 4 0 R", "/Contents (x) 0 R", 1)), ReasonMalformedOutput},
		"loose second number":          {[]byte(strings.Replace(plain, "/Contents 4 0 R", "/Contents 4 (x) R", 1)), ReasonMalformedOutput},
		"two catalog types":            {[]byte(strings.Replace(plain, "/Type /Catalog", "/Type /Catalog /Type /Catalog", 1)), ReasonMalformedOutput},
		"type value missing":           {[]byte(strings.Replace(plain, "<< /Type /Pages /Kids [] /Count 0 >>", "<< /Kids [] /Count 0 /Type >>", 1)), ReasonMalformedOutput},
		"type not a name":              {[]byte(strings.Replace(plain, "/Type /Pages", "/Type (Pages)", 1)), ReasonMalformedOutput},
		"type name differs":            {[]byte(strings.Replace(plain, "/Type /Pages", "/Kind /Pages /Type /Pagez", 1)), ReasonMalformedOutput},
		"root last":                    {[]byte(strings.Replace(plain, "<< /Size 5 /Root 1 0 R >>", "<< /Size 5 /Root >>", 1)), ReasonMalformedOutput},
		"root two lexemes":             {[]byte(strings.Replace(plain, "<< /Size 5 /Root 1 0 R >>", "<< /Size 5 /Root 1 0 >>", 1)), ReasonMalformedOutput},
		"root not a reference":         {[]byte(strings.Replace(plain, "/Root 1 0 R", "/Root 1 0 /R", 1)), ReasonMalformedOutput},
		"size last":                    {[]byte(strings.Replace(plain, "<< /Size 5 /Root 1 0 R >>", "<< /Root 1 0 R /Size >>", 1)), ReasonMalformedOutput},
	}
	for label, c := range cases {
		if got := inspect(c.data); got != c.want {
			t.Errorf("%s: got %q want %q", label, got, c.want)
		}
	}
	exact := page("BT")
	padded := page("BT" + strings.Repeat(" ", maxFileBytes-len(exact)))
	for len(padded) != maxFileBytes {
		padded = page("BT" + strings.Repeat(" ", maxFileBytes-len(exact)-(len(padded)-maxFileBytes)))
	}
	if got := inspect(padded); got != "" {
		t.Fatalf("exactly the size limit: %s", got)
	}
	if got := inspect(append(bytes.TrimSuffix(padded, []byte("\n")), '\n', '\n')); got != ReasonMalformedOutput {
		t.Fatalf("over the size limit: %s", got)
	}
}

func TestGateCasesWithExactOffsets(t *testing.T) {
	pages := "<< /Type /Pages /Kids [] /Count 0 >>"
	cases := map[string]struct {
		data []byte
		want string
	}{
		"second top-level dictionary": {assemble("", "<< /Type /Catalog >> << /Pages 2 0 R >>", pages), ReasonMalformedOutput},
		"two catalog types":           {assemble("", "<< /Type /Catalog /Type /Catalog /Pages 2 0 R >>", pages), ReasonMalformedOutput},
		"type value before close":     {assemble("", "<< /Pages 2 0 R /Type /Catalog >>", pages), ""},
		"type then a number":          {assemble("", "<< /Type 7 /Pages 2 0 R >>", pages), ReasonMalformedOutput},
		"nul separates a reference":   {assemble("", "<< /Type /Catalog /Pages 2 0\x00R >>", pages), ""},
		"lone close inside":           {assemble("", "<< /Type /Catalog /Pages 2 0 R /X << /Y 1 >x >>", pages), ReasonMalformedOutput},
		"adjacent closes":             {assemble("", "<< /Type /Catalog /Pages 2 0 R /N << /K 1 >>>>", pages), ""},
		"adjacent opens":              {assemble("", "<</Type /Catalog /Pages 2 0 R>>", pages), ""},
		"comment with eof":            {assemble("", "<< /Type /Catalog /Pages 2 0 R >>\n% EOF", pages), ReasonMalformedOutput},
		"comment with startxref":      {assemble("", "<< /Type /Catalog /Pages 2 0 R >>\n% startxref 9", pages), ReasonMalformedOutput},
		"plain comment":               {assemble("", "<< /Type /Catalog /Pages 2 0 R >>\n% note", pages), ""},
		"endobj twice":                {assemble("", "<< /Type /Catalog /Pages 2 0 R >>\nendobj\nendobj", pages), ReasonMalformedOutput},
		"name length with empty body": {build("", "<< /Length /X >>\nstream\n\nendstream"), ReasonMalformedOutput},
		"crlf stream without eol":     {build("", "<< /Length 3 >>\nstream\r\nabcendstream"), ""},
		"open trailer dictionary":     {assemble("<< /Size 3 /Root 1 0 R", "<< /Type /Catalog /Pages 2 0 R >>", pages), ReasonMalformedOutput},
		"trailer with extra close":    {assemble("<< /Size 3 /Root 1 0 R >> >>", "<< /Type /Catalog /Pages 2 0 R >>", pages), ReasonMalformedOutput},
		"plain assembled":             {assemble("", "<< /Type /Catalog /Pages 2 0 R >>", pages), ""},
	}
	for label, c := range cases {
		if got := inspect(c.data); got != c.want {
			t.Errorf("%s: got %q want %q", label, got, c.want)
		}
	}
}
