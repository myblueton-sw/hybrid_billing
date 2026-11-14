package exports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func goldenRequest() Request {
	q := settlement("gold",
		Line{Row: "r2", Source: "usage:b", Product: "=cmd", TaxCode: "VAT0", Quantity: "3", Net: "-5.5", Tax: "0"},
		Line{Row: "r1", Source: "usage:a", Product: "Compute", TaxCode: "VAT10", Quantity: "2", Net: "100.25", Tax: "10.03"},
		Line{Row: "r3", Source: "usage:c", Product: "Storage", TaxCode: "VAT10", Quantity: "7", Net: "40", Tax: "4"})
	return q
}

func TestGoldenBytes(t *testing.T) {
	r, _ := newRegistry(t, "xlsx-1")
	a := must(r.Generate("w1", goldenRequest(), human("biller")))
	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(goldenDigests), "\n") {
		name, digest, _ := strings.Cut(line, " ")
		want[name] = digest
	}
	for _, f := range a.Files {
		sum := sha256.Sum256(f.Bytes)
		if got := hex.EncodeToString(sum[:]); got != want[f.Name] || got != f.Digest {
			t.Errorf("%s: %s", f.Name, got)
		}
	}
	x := parts(t, a.Files[0].Bytes)
	for name, body := range x {
		d := xml.NewDecoder(strings.NewReader(body))
		for {
			_, err := d.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s not well formed: %v", name, err)
			}
		}
	}
	if !strings.Contains(x["xl/worksheets/sheet3.xml"], `<row r="2"><c r="A2" s="1" t="inlineStr"><is><t xml:space="preserve">VAT0</t></is></c><c r="B2" s="1" t="inlineStr"><is><t xml:space="preserve">-5.50</t></is></c><c r="C2" s="1" t="inlineStr"><is><t xml:space="preserve">0.00</t></is></c></row><row r="3"><c r="A3" s="1" t="inlineStr"><is><t xml:space="preserve">VAT10</t></is></c><c r="B3" s="1" t="inlineStr"><is><t xml:space="preserve">140.25</t></is></c><c r="C3" s="1" t="inlineStr"><is><t xml:space="preserve">14.03</t></is></c></row>`) {
		t.Fatalf("tax sheet %s", x["xl/worksheets/sheet3.xml"])
	}
	if !strings.Contains(x["xl/worksheets/sheet4.xml"], `>3</t></is></c><c r="B2" s="1" t="inlineStr"><is><t xml:space="preserve">134.75</t></is></c><c r="C2" s="1" t="inlineStr"><is><t xml:space="preserve">14.03</t></is></c><c r="D2" s="1" t="inlineStr"><is><t xml:space="preserve">12.00</t>`) {
		t.Fatalf("reconciliation %s", x["xl/worksheets/sheet4.xml"])
	}
	want2 := Contract{Encoding: "UTF-8", Separator: ",", LineEnd: "\n", Decimal: ".", Sign: "leading_minus", Period: "YYYY-MM", TimeZone: "+09:00", Scale: 2, SortKey: "row", ViewingBOM: true, ViewingPrecision: "not_precision_safe", Profile: "exports-test", ProfileRevision: "1"}
	if a.Contract != want2 {
		t.Fatalf("contract %+v", a.Contract)
	}
	if r.split != 1048575 {
		t.Fatal("split threshold")
	}
}

const goldenDigests = `
gold.xlsx 2df3c32298de8d0da6dba801864b6e6934f904057c0a45e66212bef1c0952a56
gold.detail.csv 2d9583b3956838fb37dad58acecea5f7fda797bab2da03f9156fc66956f6ab72
gold.machine.csv 0394a478fb83efd2ff8177b922759b2a7826a56e3d18b3193b6c4995eef8411c
`
