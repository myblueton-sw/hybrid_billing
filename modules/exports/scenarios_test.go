package exports

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

var payloads = []string{"=HYPERLINK(\"http://x\")", "+cmd|' /C calc'!A0", "@SUM(A1)", "-2+3", "\tlead", "\rlead", "<script>alert(1)</script>", "a\x00b\x07c", "'quoted"}

func TestPayloadsNeutralisedAndValuesPreserved(t *testing.T) {
	r, _ := newRegistry(t, "xlsx-1")
	longID := strings.Repeat("9", 120)
	var lines []Line
	for i, p := range payloads[:5] {
		lines = append(lines, Line{Row: longID[:100] + string(rune('a'+i)), Source: p, Product: payloads[(i+5)%len(payloads)], TaxCode: "VAT10", Quantity: "123456789012345678", Net: "-12.34", Tax: "0.01"})
	}
	req := settlement("art-1", lines...)
	req.Spec.Scale = 8
	req.Spec.Lines[0].Net = "-123456789012345678.12345678"
	a := must(r.Generate("w1", req, human("biller")))
	xlsx := parts(t, file(t, a, ".xlsx"))
	if names(xlsx) != "[Content_Types].xml,_rels/.rels,xl/_rels/workbook.xml.rels,xl/styles.xml,xl/workbook.xml,xl/worksheets/sheet1.xml,xl/worksheets/sheet2.xml,xl/worksheets/sheet3.xml,xl/worksheets/sheet4.xml,xl/worksheets/sheet5.xml" {
		t.Fatalf("parts %s", names(xlsx))
	}
	for name, body := range xlsx {
		for _, banned := range []string{"<f>", "<f ", "vbaProject", "externalLink", "connections", "definedName", "macroEnabled", "<script"} {
			if strings.Contains(body, banned) {
				t.Fatalf("%s contains %s", name, banned)
			}
		}
	}
	detail := xlsx["xl/worksheets/sheet2.xml"]
	for _, want := range []string{"s=\"2\" t=\"inlineStr\"><is><t xml:space=\"preserve\">=HYPERLINK", "s=\"2\" t=\"inlineStr\"><is><t xml:space=\"preserve\">-2+3", "&lt;script&gt;", "a\ufffdb\ufffdc", "s=\"1\" t=\"inlineStr\"><is><t xml:space=\"preserve\">-123456789012345678.12345678", "123456789012345678.00000000", longID[:100] + "a"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail misses %q", want)
		}
	}
	if !strings.Contains(xlsx["xl/styles.xml"], `quotePrefix="1"`) {
		t.Fatal("no quote prefix style")
	}
	viewing := string(file(t, a, ".detail.csv"))
	for _, p := range payloads[:3] {
		if strings.Contains(viewing, ","+csvQuote(p)+",") {
			t.Fatalf("viewing csv has raw %q", p)
		}
	}
	mr := csv.NewReader(bytes.NewReader(file(t, a, ".machine.csv")))
	mr.FieldsPerRecord = -1
	machine := must(mr.ReadAll())
	if machine[0][0] != "#machine-values" || machine[0][1] != "settlement-1" {
		t.Fatalf("marker %v", machine[0])
	}
	found := map[string]bool{}
	for _, row := range machine[2:] {
		found[row[1]] = true
	}
	for _, p := range payloads[:5] {
		if !found[p] {
			t.Fatalf("machine csv lost raw %q", p)
		}
	}
}

func csvQuote(text string) string {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Write([]string{text})
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n")
}

func TestDeterministicAndConfirmedBytes(t *testing.T) {
	r, clock := newRegistry(t, "xlsx-1")
	lines := []Line{line("r3", "30"), line("r1", "10"), line("r2", "20.5")}
	a := must(r.Generate("w1", settlement("a1", lines...), human("biller")))
	shuffled := []Line{lines[2], lines[0], lines[1]}
	clock.advance(3600e9)
	b := must(r.Generate("w1", settlement("a2", shuffled...), system("job")))
	if a.Data != b.Data {
		t.Fatal("order changed data")
	}
	for i := range a.Files {
		if i == 0 {
			continue
		}
		if !bytes.Equal(a.Files[i].Bytes, b.Files[i].Bytes) {
			t.Fatalf("file %d differs", i)
		}
	}
	if b.Volatile.At.Equal(a.Volatile.At) {
		t.Fatal("volatile not recorded")
	}
	other, _ := newRegistry(t, "xlsx-2")
	c := must(other.Generate("w1", settlement("a1", lines...), human("biller")))
	if c.Data != a.Data || bytes.Equal(c.Files[0].Bytes, a.Files[0].Bytes) || c.Generator == a.Generator {
		t.Fatalf("generator version not separated")
	}
	cmp := must(other.Compare("w1", "a1", settlement("a1", shuffled...), human("qa")))
	if !cmp.DataEqual || cmp.GeneratorChanged {
		t.Fatalf("compare %+v", cmp)
	}
	must(r.Confirm("w1", "a1", human("lead")))
	got := must(r.Download("w1", "a1", "a1.xlsx", human("ops")))
	if !bytes.Equal(got, a.Files[0].Bytes) {
		t.Fatal("download differs")
	}
	got[0] ^= 1
	if again := must(r.Download("w1", "a1", "a1.xlsx", system("mailer"))); !bytes.Equal(again, a.Files[0].Bytes) {
		t.Fatal("download aliasing")
	}
	changed := settlement("a1", line("r1", "11"))
	expect(t, "replace confirmed", second(r.Generate("w1", changed, human("biller"))), "conflict")
	if cmp := must(r.Compare("w1", "a1", changed, human("qa"))); cmp.DataEqual {
		t.Fatal("changed data equal")
	}
}

func TestCustomerPolicyLeaksNothingInternal(t *testing.T) {
	r, _ := newRegistry(t, "xlsx-1")
	req := settlement("c1", Line{Row: "r1", Source: "secret-usage-feed", Product: "Compute", TaxCode: "VAT10", Quantity: "1", Net: "100", Tax: "10"})
	req.Kind = KindCustomer
	a := must(r.Generate("w1", req, human("biller")))
	if len(a.Files) != 2 {
		t.Fatalf("customer files %d", len(a.Files))
	}
	for _, f := range a.Files {
		body := string(f.Bytes)
		if strings.HasSuffix(f.Name, ".xlsx") {
			var all []string
			for _, v := range parts(t, f.Bytes) {
				all = append(all, v)
			}
			body = strings.Join(all, "\n")
		}
		for _, secret := range []string{"secret-usage-feed", approval, "reconciliation", "erp_reference", "approval"} {
			if strings.Contains(body, secret) {
				t.Fatalf("%s leaks %s", f.Name, secret)
			}
		}
	}
}

func TestResultRegistrationRefusesAmountChanges(t *testing.T) {
	r, _ := newRegistry(t, "xlsx-1")
	must(r.Generate("w1", settlement("a1", line("r1", "10"), line("r2", "20")), human("biller")))
	good := "row,submission,erp_reference,erp_state,net,tax,quantity\nr1,sub-1,INV-1,posted,10.00,0.00,1.00\nr2,sub-1,,,20.00,0.00,1.00\n"
	expect(t, "not confirmed", second(r.RegisterResults("w1", "a1", []byte(good), human("ops"))), ReasonNotConfirmed)
	must(r.Confirm("w1", "a1", human("lead")))
	changed := "row,submission,erp_reference,erp_state,net,tax,quantity\nr1,sub-1,INV-1,posted,9.00,0.00,1.00\n"
	expect(t, "amount", second(r.RegisterResults("w1", "a1", []byte(changed), human("ops"))), ReasonAmountChanged)
	res := must(r.RegisterResults("w1", "a1", []byte(good), system("importer")))
	if len(res.Rows) != 1 || res.Rows[0] != (Result{Row: "r1", Reference: "INV-1", State: "posted"}) || res.Handoff == nil || *res.Handoff != (Handoff{Observation: "a1:1", Connection: "erp-file", Submission: "sub-1", Reference: "INV-1", State: "posted", Revision: 1}) {
		t.Fatalf("results %+v", res)
	}
}
