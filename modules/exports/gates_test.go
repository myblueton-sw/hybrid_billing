package exports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestRegistryForms(t *testing.T) {
	clock := &testClock{now: start}
	good := Config{Generator: "g", Schema: "s", Numbers: numbers(t)}
	must(NewRegistry(good, Limits{1, 1, 1}, clock.read))
	must(NewRegistry(good, Limits{100000, 1000000, 200000}, clock.read))
	for _, c := range []Config{{Schema: "s", Numbers: good.Numbers}, {Generator: "g", Numbers: good.Numbers}, {Generator: "g", Schema: "s"}} {
		if _, err := NewRegistry(c, Limits{1, 1, 1}, clock.read); err != ErrInvalid {
			t.Fatalf("%+v: %v", c, err)
		}
	}
	for _, l := range []Limits{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}, {100001, 1, 1}, {1, 1000001, 1}, {1, 1, 200001}} {
		if _, err := NewRegistry(good, l, clock.read); err != ErrInvalid {
			t.Fatalf("%+v: %v", l, err)
		}
	}
	if _, err := NewRegistry(good, Limits{1, 1, 1}, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	err := Refusal{Reason: ReasonAmountChanged, Row: "r1"}
	if !errors.Is(err, ErrRejected) || err.Error() != ErrRejected.Error()+": amount_changed at r1" || refuse("x").Error() != ErrRejected.Error()+": x" {
		t.Fatal(err)
	}
}

func TestRequestForms(t *testing.T) {
	r, _ := newRegistry(t, "g")
	good := settlement("a", line("r1", "1"))
	bad := []func(*Request){
		func(q *Request) { q.ID = "" }, func(q *Request) { q.ID = strings.Repeat("a", 101) }, func(q *Request) { q.Kind = "draft" },
		func(q *Request) { q.TimeZone = "Asia/Seoul" }, func(q *Request) { q.Spec.Connection = "" }, func(q *Request) { q.Spec.Submission = "a b" },
		func(q *Request) { q.Spec.Approval = "x" }, func(q *Request) { q.Spec.Approval = strings.Repeat("g", 64) }, func(q *Request) { q.Spec.Customer = "" }, func(q *Request) { q.Spec.Period = "2026-13" },
		func(q *Request) { q.Spec.Currency = "krw" }, func(q *Request) { q.Spec.Scale = -1 }, func(q *Request) { q.Spec.Scale = 9 },
		func(q *Request) { q.Spec.Lines = nil },
		func(q *Request) { q.Spec.Lines = []Line{line("r1", "1"), line("r1", "2")} },
		func(q *Request) { q.Spec.Lines[0].Row = "a'b" }, func(q *Request) { q.Spec.Lines[0].Source = "" },
		func(q *Request) { q.Spec.Lines[0].Product = strings.Repeat("x", 513) }, func(q *Request) { q.Spec.Lines[0].TaxCode = "\xff" },
		func(q *Request) { q.Spec.Lines[0].Net = "1.234" }, func(q *Request) { q.Spec.Lines[0].Tax = "x" },
		func(q *Request) { q.Spec.Lines[0].Quantity = "1e3" }, func(q *Request) { q.Spec.Lines[0].Net = strings.Repeat("1", 65) },
	}
	for i, change := range bad {
		q := good
		q.Spec.Lines = append([]Line(nil), good.Spec.Lines...)
		change(&q)
		if got := kind(second(r.Generate("w1", q, human("b")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	for _, z := range []string{"UTC", "+14:59", "-00:00"} {
		q := good
		q.ID, q.TimeZone = "z"+strings.ReplaceAll(strings.ReplaceAll(z, ":", ""), "+", "p"), z
		expect(t, z, second(r.Generate("w1", q, human("b"))), "ok")
	}
	for _, z := range []string{"+15:00", "+09:60", "+0900", "09:00", "+a9:00", "+09:0a"} {
		q := good
		q.TimeZone = z
		expect(t, z, second(r.Generate("w1", q, human("b"))), "invalid")
	}
	edge := good
	edge.ID = strings.Repeat("a", 100)
	edge.Spec.Lines = []Line{{Row: "r", Source: strings.Repeat("s", 512), Product: "p", TaxCode: "t", Quantity: "-0", Net: "0.5", Tax: "-1"}}
	v := must(r.Generate("w1", edge, human("b")))
	if !strings.Contains(string(file(t, v, ".machine.csv")), ",0.00,0.50,-1.00") {
		t.Fatalf("normalised %s", file(t, v, ".machine.csv"))
	}
	expect(t, "ws", second(r.Generate("", good, human("b"))), "invalid")
	expect(t, "actor", second(r.Generate("w1", good, Actor{ID: "x", Kind: "robot"})), "invalid")
	expect(t, "assistant", second(r.Generate("w1", good, assistant("bot"))), ReasonAINotAllowed)
}

func TestGenerateRepeatsAndLimits(t *testing.T) {
	r, _ := newRegistry(t, "g")
	q := settlement("a", line("r1", "1"))
	v := must(r.Generate("w1", q, human("b")))
	expect(t, "same data different text", second(r.Generate("w1", settlement("a", line("r1", "1.00")), human("b"))), "conflict")
	if same := must(r.Generate("w1", q, human("b"))); same.Volatile != v.Volatile {
		t.Fatal("repeat")
	}
	expect(t, "other actor", second(r.Generate("w1", q, human("c"))), "conflict")
	v.Files[0].Bytes[0] ^= 1
	v.Request.Spec.Lines[0].Net = "x"
	if got := must(r.ArtifactOf("w1", "a")); got.Files[0].Bytes[0] == v.Files[0].Bytes[0] || got.Request.Spec.Lines[0].Net != "1" {
		t.Fatal("aliasing")
	}
	expect(t, "artifact of", second(r.ArtifactOf("w1", "nope")), "not_found")
	expect(t, "artifact of ws", second(r.ArtifactOf("w9", "a")), "not_found")
	lines := []Line{line("1", "1"), line("2", "1"), line("3", "1"), line("4", "1"), line("5", "1"), line("6", "1"), line("7", "1")}
	expect(t, "lines", second(r.Generate("w1", settlement("big", lines...), human("b"))), "limit")
	for i := 0; i < 5; i++ {
		must(r.Generate("w1", settlement("n"+string(rune('0'+i)), line("r1", "1")), human("b")))
	}
	expect(t, "artifacts", second(r.Generate("w1", settlement("n9", line("r1", "1")), human("b"))), "limit")
	must(r.Generate("w2", q, human("b")))
	must(r.Generate("w3", q, human("b")))
	expect(t, "workspaces", second(r.Generate("w4", q, human("b"))), "limit")
	clock := &testClock{now: start}
	wide := must(NewRegistry(Config{Generator: "g", Schema: "s", Numbers: numbers(t)}, Limits{1, 2, 10001}, clock.read))
	var many []Line
	for i := 0; i < 10001; i++ {
		many = append(many, line("r"+strings.Repeat("0", 6-len(itoa(i)))+itoa(i), "1"))
	}
	expect(t, "settlement cap", second(wide.Generate("w", settlement("s", many...), human("b"))), "limit")
	c := settlement("c", many...)
	c.Kind = KindCustomer
	expect(t, "customer may be larger", second(wide.Generate("w", c, human("b"))), "ok")
	tight := must(NewRegistry(Config{Generator: "g", Schema: "s", Numbers: must(decimal.NewProfile(decimal.ProfileSpec{ID: "t", Revision: "1", InputIntegerDigits: 18, InputScale: 8, ResultIntegerDigits: 18, ResultScale: 8, IntermediateDigits: 80}))}, Limits{1, 1, 6}, clock.read))
	huge := strings.Repeat("9", 18)
	for i, l := range [][2]Line{
		{{Row: "a", Source: "s", Product: "p", TaxCode: "t", Quantity: huge, Net: "0", Tax: "0"}, {Row: "b", Source: "s", Product: "p", TaxCode: "t", Quantity: huge, Net: "0", Tax: "0"}},
		{{Row: "a", Source: "s", Product: "p", TaxCode: "t", Quantity: "0", Net: huge, Tax: "0"}, {Row: "b", Source: "s", Product: "p", TaxCode: "t", Quantity: "0", Net: huge, Tax: "0"}},
		{{Row: "a", Source: "s", Product: "p", TaxCode: "t", Quantity: "0", Net: "0", Tax: huge}, {Row: "b", Source: "s", Product: "p", TaxCode: "t", Quantity: "0", Net: "0", Tax: huge}},
	} {
		q := settlement("t"+itoa(i), l[0], l[1])
		q.Spec.Scale = 0
		expect(t, "total bound", second(tight.Generate("w", q, human("b"))), "limit")
		expect(t, "compare missing", second(tight.Compare("w", "x", q, human("b"))), "not_found")
	}
	small := settlement("x", line("r1", "1"))
	small.Spec.Scale = 0
	must(tight.Generate("w", small, human("b")))
	big := settlement("x", Line{Row: "a", Source: "s", Product: "p", TaxCode: "t", Quantity: "0", Net: huge, Tax: "0"}, Line{Row: "b", Source: "s", Product: "p", TaxCode: "t", Quantity: "0", Net: huge, Tax: "0"})
	big.Spec.Scale = 0
	expect(t, "compare bound", second(tight.Compare("w", "x", big, human("b"))), "limit")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	out := ""
	for ; i > 0; i /= 10 {
		out = string(rune('0'+i%10)) + out
	}
	return out
}

func TestConfirmDownloadCompareGates(t *testing.T) {
	r, _ := newRegistry(t, "g")
	q := settlement("a", line("r1", "1"))
	must(r.Generate("w1", q, system("job")))
	expect(t, "download draft", second(r.Download("w1", "a", "a.xlsx", human("x"))), ReasonNotConfirmed)
	expect(t, "confirm form", second(r.Confirm("w1", "", human("x"))), "invalid")
	expect(t, "confirm ws", second(r.Confirm("", "a", human("x"))), "invalid")
	expect(t, "confirm actor", second(r.Confirm("w1", "a", human(""))), "invalid")
	expect(t, "confirm missing", second(r.Confirm("w1", "nope", human("x"))), "not_found")
	expect(t, "confirm assistant", second(r.Confirm("w1", "a", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "confirm system", second(r.Confirm("w1", "a", system("job2"))), ReasonNotHuman)
	must(r.Generate("w1", settlement("b", line("r1", "1")), human("biller")))
	expect(t, "self confirm", second(r.Confirm("w1", "b", human("biller"))), ReasonSelfApproval)
	c := must(r.Confirm("w1", "a", human("lead")))
	if c.Confirmed == nil || c.Confirmed.Actor != "lead" {
		t.Fatalf("confirmed %+v", c)
	}
	c.Confirmed.Actor = "x"
	if must(r.Confirm("w1", "a", human("lead"))).Confirmed.Actor != "lead" {
		t.Fatal("confirm repeat or aliasing")
	}
	expect(t, "confirm other", second(r.Confirm("w1", "a", human("lead2"))), ReasonConfirmed)
	expect(t, "confirm system again", second(r.Confirm("w1", "a", system("lead"))), ReasonConfirmed)
	expect(t, "download form", second(r.Download("w1", "a", "", human("x"))), "invalid")
	expect(t, "download ws", second(r.Download("", "a", "a.xlsx", human("x"))), "invalid")
	expect(t, "download id", second(r.Download("w1", "", "a.xlsx", human("x"))), "invalid")
	expect(t, "download actor", second(r.Download("w1", "a", "a.xlsx", human(""))), "invalid")
	expect(t, "download missing", second(r.Download("w1", "nope", "a.xlsx", human("x"))), "not_found")
	expect(t, "download assistant", second(r.Download("w1", "a", "a.xlsx", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "download file", second(r.Download("w1", "a", "nope.xlsx", human("x"))), "not_found")
	expect(t, "compare form", second(r.Compare("w1", "a", Request{}, human("x"))), "invalid")
	expect(t, "compare ws", second(r.Compare("", "a", q, human("x"))), "invalid")
	expect(t, "compare id", second(r.Compare("w1", "", q, human("x"))), "invalid")
	expect(t, "compare actor", second(r.Compare("w1", "a", q, human(""))), "invalid")
	expect(t, "compare missing", second(r.Compare("w1", "nope", q, human("x"))), "not_found")
	expect(t, "compare assistant", second(r.Compare("w1", "a", q, assistant("bot"))), ReasonAINotAllowed)
	other := must(NewRegistry(Config{Generator: "g", Schema: "s2", Numbers: numbers(t)}, Limits{1, 2, 6}, (&testClock{now: start}).read))
	must(other.Generate("w1", q, system("job")))
	if cmp := must(other.Compare("w1", "a", q, system("job"))); !cmp.DataEqual || cmp.SchemaChanged || cmp.GeneratorChanged {
		t.Fatalf("same %+v", cmp)
	}
}

func TestCustomerAllowlist(t *testing.T) {
	r, _ := newRegistry(t, "g")
	req := settlement("c1", Line{Row: "r1", Source: "secret", Product: "Compute", TaxCode: "VAT10", Quantity: "1", Net: "100", Tax: "10"})
	req.Kind = KindCustomer
	a := must(r.Generate("w1", req, human("b")))
	if len(a.Files) != 2 || a.Files[0].Name != "c1.xlsx" || a.Files[1].Name != "c1.detail.csv" {
		t.Fatalf("files %+v", a.Files)
	}
	x := parts(t, a.Files[0].Bytes)
	if names(x) != "[Content_Types].xml,_rels/.rels,xl/_rels/workbook.xml.rels,xl/styles.xml,xl/workbook.xml,xl/worksheets/sheet1.xml,xl/worksheets/sheet2.xml,xl/worksheets/sheet3.xml" {
		t.Fatalf("parts %s", names(x))
	}
	if !strings.Contains(x["xl/workbook.xml"], `<sheet name="header" sheetId="1" r:id="rId1"/><sheet name="detail" sheetId="2" r:id="rId2"/><sheet name="tax" sheetId="3" r:id="rId3"/>`) || strings.Contains(x["xl/workbook.xml"], "state=") {
		t.Fatalf("sheets %s", x["xl/workbook.xml"])
	}
	header := x["xl/worksheets/sheet1.xml"]
	for _, field := range []string{"customer", "period", "currency", "scale", "generator", "schema"} {
		if !strings.Contains(header, ">"+field+"<") {
			t.Fatalf("header misses %s", field)
		}
	}
	for _, field := range []string{"submission", "connection", "approval", "secret"} {
		if strings.Contains(header+x["xl/worksheets/sheet2.xml"], field) {
			t.Fatalf("customer leak %s", field)
		}
	}
	rows := must(csv.NewReader(bytes.NewReader(bytes.TrimPrefix(a.Files[1].Bytes, []byte("\ufeff")))).ReadAll())
	if strings.Join(rows[0], ",") != "row,product,tax_code,quantity,net,tax" {
		t.Fatalf("columns %v", rows[0])
	}
	s := must(r.Generate("w1", settlement("s1", line("r1", "1")), human("b")))
	srows := must(csv.NewReader(bytes.NewReader(bytes.TrimPrefix(s.Files[1].Bytes, []byte("\ufeff")))).ReadAll())
	if strings.Join(srows[0], ",") != "row,source,product,tax_code,quantity,net,tax" || !bytes.HasPrefix(s.Files[1].Bytes, []byte("\ufeff")) || bytes.HasPrefix(s.Files[2].Bytes, []byte("\ufeff")) {
		t.Fatalf("settlement columns %v", srows[0])
	}
	if s.Contract.ViewingPrecision != "not_precision_safe" || !s.Contract.ViewingBOM || s.Contract.Profile != "exports-test" || s.Contract.ProfileRevision != "1" {
		t.Fatalf("contract %+v", s.Contract)
	}
}

func TestDetailSplit(t *testing.T) {
	r, _ := newRegistry(t, "g")
	r.split = 2
	a := must(r.Generate("w1", settlement("a", line("r1", "1"), line("r2", "1"), line("r3", "1"), line("r4", "1"), line("r5", "1")), human("b")))
	x := parts(t, a.Files[0].Bytes)
	if !strings.Contains(x["xl/workbook.xml"], `name="detail"`) || !strings.Contains(x["xl/workbook.xml"], `name="detail-2"`) || !strings.Contains(x["xl/workbook.xml"], `name="detail-3"`) || strings.Contains(x["xl/workbook.xml"], `name="detail-4"`) {
		t.Fatalf("split %s", x["xl/workbook.xml"])
	}
	if !strings.Contains(x["xl/worksheets/sheet4.xml"], ">r5<") || !strings.Contains(x["xl/worksheets/sheet4.xml"], ">row<") {
		t.Fatalf("last part %s", x["xl/worksheets/sheet4.xml"])
	}
	if column(0) != "A" || column(25) != "Z" || column(26) != "AA" || column(701) != "ZZ" || column(702) != "AAA" {
		t.Fatal("columns")
	}
}

func TestResultTransitions(t *testing.T) {
	r, _ := newRegistry(t, "g")
	must(r.Generate("w1", settlement("a", line("r1", "1"), line("r2", "2"), line("r3", "3")), human("b")))
	must(r.Generate("w1", Request{ID: "c", Kind: KindCustomer, TimeZone: "UTC", Spec: spec(line("r1", "1"))}, human("b")))
	head := "row,submission,erp_reference,erp_state,net,tax,quantity\n"
	reg := func(body string) (Registration, error) {
		return r.RegisterResults("w1", "a", []byte(body), human("ops"))
	}
	expect(t, "form ws", second(r.RegisterResults("", "a", nil, human("ops"))), "invalid")
	expect(t, "form id", second(r.RegisterResults("w1", "", nil, human("ops"))), "invalid")
	expect(t, "form actor", second(r.RegisterResults("w1", "a", nil, human(""))), "invalid")
	expect(t, "missing", second(r.RegisterResults("w1", "nope", nil, human("ops"))), "not_found")
	expect(t, "assistant", second(r.RegisterResults("w1", "a", nil, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "customer", second(r.RegisterResults("w1", "c", nil, human("ops"))), ReasonNotSettlement)
	must(r.Confirm("w1", "a", human("lead")))
	expect(t, "too big", second(reg(strings.Repeat("x", 32<<20+1))), "limit")
	expect(t, "not utf8", second(reg(head+"r1,sub-1,\xff,,1.00,0.00,1.00\n")), ReasonNotUTF8)
	expect(t, "semicolon", second(reg("row;submission;erp_reference;erp_state;net;tax;quantity\n")), ReasonWrongFormat)
	expect(t, "empty", second(reg("")), ReasonWrongFormat)
	expect(t, "short row", second(reg(head+"r1,sub-1\n")), "invalid")
	expect(t, "bad quote", second(reg(head+"r1,\"sub-1\n")), "invalid")
	expect(t, "duplicate", second(reg(head+"r1,sub-1,,,1.00,0.00,1.00\nr1,sub-1,,,1.00,0.00,1.00\n")), "invalid")
	expect(t, "bad ref", second(reg(head+"r1,sub-1,a b,posted,1.00,0.00,1.00\n")), "invalid")
	expect(t, "bad state", second(reg(head+"r1,sub-1,X,paid,1.00,0.00,1.00\n")), "invalid")
	expect(t, "posted no ref", second(reg(head+"r1,sub-1,,posted,1.00,0.00,1.00\n")), "invalid")
	expect(t, "draft no ref", second(reg(head+"r1,sub-1,,draft,1.00,0.00,1.00\n")), "invalid")
	expect(t, "ref no state", second(reg(head+"r1,sub-1,X,,1.00,0.00,1.00\n")), "invalid")
	expect(t, "unknown row", second(reg(head+"r9,sub-1,X,draft,1.00,0.00,1.00\n")), ReasonUnknownRow)
	expect(t, "submission", second(reg(head+"r1,sub-2,X,draft,1.00,0.00,1.00\n")), ReasonSubmissionMismatch)
	for _, changed := range []string{"r1,sub-1,X,draft,1,0.00,1.00", "r1,sub-1,X,draft,1.00,0.01,1.00", "r1,sub-1,X,draft,1.00,0.00,2.00"} {
		expect(t, "amount "+changed, second(reg(head+changed+"\n")), ReasonAmountChanged)
	}
	expect(t, "mixed state", second(reg(head+"r1,sub-1,X,draft,1.00,0.00,1.00\nr2,sub-1,X,posted,2.00,0.00,1.00\n")), ReasonMixedResults)
	expect(t, "mixed ref", second(reg(head+"r1,sub-1,X,draft,1.00,0.00,1.00\nr2,sub-1,Y,draft,2.00,0.00,1.00\n")), ReasonMixedResults)
	first := must(reg("\ufeff" + strings.ReplaceAll(head+"r1,sub-1,X,draft,1.00,0.00,1.00\nr2,sub-1,,,2.00,0.00,1.00\n", "\n", "\r\n")))
	if first.Handoff == nil || first.Handoff.Revision != 1 || first.Handoff.State != "draft" || first.Handoff.Reference != "X" || len(first.Rows) != 1 {
		t.Fatalf("first %+v", first)
	}
	retry := must(reg(head + "r1,sub-1,X,draft,1.00,0.00,1.00\n"))
	if retry.Handoff == nil || *retry.Handoff != *first.Handoff {
		t.Fatalf("retry %+v", retry)
	}
	none := must(r.RegisterResults("w1", "a", []byte(head+"r3,sub-1,,,3.00,0.00,1.00\n"), system("importer")))
	if len(none.Rows) != 0 || none.Handoff.Revision != 1 {
		t.Fatalf("empty registration %+v", none)
	}
	expect(t, "other ref later", second(reg(head+"r2,sub-1,Y,draft,2.00,0.00,1.00\n")), ReasonMixedResults)
	expect(t, "forward draft to posted", second(reg(head+"r1,sub-1,X,posted,1.00,0.00,1.00\n")), "ok")
	expect(t, "posted to draft", second(reg(head+"r1,sub-1,X,draft,1.00,0.00,1.00\n")), "conflict")
	second2 := must(reg(head + "r2,sub-1,X,posted,2.00,0.00,1.00\n"))
	if second2.Handoff.Revision != 3 || second2.Handoff.Observation != "a:3" {
		t.Fatalf("revision %+v", second2.Handoff)
	}
	rej := must(reg(head + "r3,sub-1,,rejected,3.00,0.00,1.00\n"))
	if rej.Handoff.Reference != "X" || rej.Handoff.State != "rejected" {
		t.Fatalf("rejected keeps reference %+v", rej.Handoff)
	}
	expect(t, "rejected to posted", second(reg(head+"r3,sub-1,X,posted,3.00,0.00,1.00\n")), "conflict")
	must(reg(head + "r1,sub-1,,rejected,1.00,0.00,1.00\n"))
	g, _ := newRegistry(t, "g")
	must(g.Generate("w1", settlement("a", line("r1", "1")), human("b")))
	must(g.Confirm("w1", "a", human("lead")))
	expect(t, "too many rows", second(g.RegisterResults("w1", "a", []byte(head+"r1,sub-1,,,1.00,0.00,1.00\nr2,sub-1,,,1.00,0.00,1.00\n"), human("ops"))), "limit")
	expect(t, "prefixed safe id kept", second(g.RegisterResults("w1", "a", []byte(head+"'r1,sub-1,,,1.00,0.00,1.00\n"), human("ops"))), "invalid")
	if res := must(g.RegisterResults("w1", "a", []byte(head+"r1,sub-1,'-X,posted,1.00,0.00,1.00\n"), human("ops"))); res.Handoff.Reference != "-X" {
		t.Fatalf("prefixed risky reference %+v", res.Handoff)
	}
}

func TestHandoffDrivesLedger(t *testing.T) {
	r, _ := newRegistry(t, "g")
	clock := &testClock{now: start}
	l := must(erpledger.NewRegistry(numbers(t), erpledger.Limits{Workspaces: 1, Connections: 1, Mappings: 5, Anchors: 1, Submissions: 2, Lines: 5, Documents: 1, Rows: 1, Observations: 10, Links: 1, Surveys: 1}, clock.read))
	admin := erpledger.Actor{ID: "admin", Kind: erpledger.KindHuman}
	must(l.Connect("w1", erpledger.Connection{ID: "erp-file", Entity: "e", System: "sap-s4", Transport: "file", Capabilities: []string{"prepare", "validate_mapping", "submit_draft"}, Currency: "KRW", Scale: 2, Interval: time.Hour, Owner: "ops", Template: "standard_xlsx"}, admin))
	must(l.VerifyTemplate("w1", "erp-file", "standard-template-tests", erpledger.Actor{ID: "qa", Kind: erpledger.KindHuman}))
	for _, m := range []erpledger.Mapping{{Connection: "erp-file", Kind: "customer", Platform: "acme", External: "C"}, {Connection: "erp-file", Kind: "product", Platform: "Compute", External: "P"}, {Connection: "erp-file", Kind: "tax", Platform: "VAT10", External: "T"}} {
		must(l.Map("w1", m, admin))
	}
	must(l.Survey("w1", erpledger.Survey{Connection: "erp-file", Customer: "acme", Period: "2026-10", Outcome: erpledger.SurveyComplete, Evidence: "s"}, admin))
	biller := erpledger.Actor{ID: "biller", Kind: erpledger.KindHuman}
	must(l.Prepare("w1", erpledger.Submission{Key: "sub-1", Connection: "erp-file", Purpose: erpledger.PurposeInvoice, Customer: "acme", Period: "2026-10", Path: erpledger.PathDirect, Lines: []erpledger.Line{{ID: "r1", Source: "usage:r1", Product: "Compute", TaxCode: "VAT10", Quantity: "1", Net: "10", Tax: "0"}}}, biller))
	must(l.Command("w1", "sub-1", "submit_draft", biller))
	must(r.Generate("w1", settlement("a", line("r1", "10")), human("biller")))
	must(r.Confirm("w1", "a", human("lead")))
	reg := must(r.RegisterResults("w1", "a", []byte("row,submission,erp_reference,erp_state,net,tax,quantity\nr1,sub-1,INV-7,posted,10.00,0.00,1.00\n"), human("ops")))
	h := reg.Handoff
	o := must(l.Observe("w1", erpledger.Observation{ID: h.Observation, Connection: h.Connection, Key: h.Submission, External: h.Reference, Revision: h.Revision, State: h.State, Source: erpledger.SourceResultFile}, erpledger.Actor{ID: "importer", Kind: erpledger.KindSystem}))
	if o.Outcome != "applied" {
		t.Fatalf("observation %+v", o)
	}
	if v := must(l.SubmissionOf("w1", "sub-1")); v.State != erpledger.StatePosted || v.External != "INV-7" {
		t.Fatalf("ledger %+v", v)
	}
}

func TestPropertyNeutralRoundTripAndDigest(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	alphabet := []string{"=", "+", "-", "@", "'", "\t", "\r", " ", "a", "1", ".", "0", "＝", "＋", "－", "＠", ",", "\""}
	for i := 0; i < 5000; i++ {
		var b strings.Builder
		for j := 0; j < 1+rng.Intn(6); j++ {
			b.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		v := b.String()
		if raw(neutral(v)) != v {
			t.Fatalf("round trip %q -> %q -> %q", v, neutral(v), raw(neutral(v)))
		}
		trimmed := strings.TrimLeft(v, " ")
		dangerous := trimmed != "" && strings.ContainsAny(trimmed[:1], "=+-@'\t\r") || strings.HasPrefix(trimmed, "＝") || strings.HasPrefix(trimmed, "＋") || strings.HasPrefix(trimmed, "－") || strings.HasPrefix(trimmed, "＠")
		if dangerous && !looksDecimal(v) && !strings.HasPrefix(neutral(v), "'") {
			t.Fatalf("unprefixed %q", v)
		}
	}
	r, _ := newRegistry(t, "g")
	for i := 0; i < 200; i++ {
		var lines []Line
		n := 1 + rng.Intn(5)
		for j := 0; j < n; j++ {
			lines = append(lines, Line{Row: "r" + itoa(j), Source: "s" + itoa(rng.Intn(3)), Product: "p", TaxCode: "t" + itoa(rng.Intn(2)), Quantity: itoa(rng.Intn(9)), Net: itoa(rng.Intn(99)), Tax: "0"})
		}
		a := settlement("x", lines...)
		perm := rng.Perm(n)
		shuffled := make([]Line, n)
		for k, p := range perm {
			shuffled[k] = lines[p]
		}
		b := settlement("x", shuffled...)
		ba, _ := r.build(a)
		bb, _ := r.build(b)
		if ba.data != bb.data || ba.inputs != bb.inputs || !bytes.Equal(ba.files[0].Bytes, bb.files[0].Bytes) {
			t.Fatal("order dependent")
		}
		lines[rng.Intn(n)].Net = "100"
		bc, _ := r.build(settlement("x", lines...))
		if bc.data == ba.data {
			t.Fatal("digest ignores a value change")
		}
	}
}

func looksDecimal(v string) bool {
	body := strings.TrimPrefix(v, "-")
	whole, fraction, dotted := strings.Cut(body, ".")
	if whole == "" || dotted && fraction == "" || len(whole) > 1 && whole[0] == '0' {
		return false
	}
	for _, c := range whole + fraction {
		if c < '0' || c > '9' {
			return false
		}
	}
	return !(strings.HasPrefix(v, "-") && strings.Trim(whole+fraction, "0") == "")
}

func TestChangeReviewRegressions(t *testing.T) {
	r, _ := newRegistry(t, "g")
	must(r.Generate("w1", settlement("a", line("r1", "1"), line("r2", "2"), line("r3", "3")), human("b")))
	must(r.Confirm("w1", "a", human("lead")))
	head := "row,submission,erp_reference,erp_state,net,tax,quantity\n"
	reg := func(body string) (Registration, error) {
		return r.RegisterResults("w1", "a", []byte(head+body), human("ops"))
	}
	must(reg("r1,sub-1,X,posted,1.00,0.00,1.00\n"))
	expect(t, "backward hand-off", second(reg("r2,sub-1,X,draft,2.00,0.00,1.00\n")), "conflict")
	first := must(reg("r1,sub-1,,rejected,1.00,0.00,1.00\nr2,sub-1,X,rejected,2.00,0.00,1.00\n"))
	if first.Handoff.State != "rejected" || first.Handoff.Reference != "X" || first.Handoff.Revision != 2 || first.Rows[0].Reference != "X" {
		t.Fatalf("rejection %+v", first)
	}
	retry := must(reg("r1,sub-1,,rejected,1.00,0.00,1.00\nr2,sub-1,X,rejected,2.00,0.00,1.00\n"))
	if *retry.Handoff != *first.Handoff {
		t.Fatalf("retry after inherited reference %+v", retry.Handoff)
	}
	expect(t, "quoted header", second(r.RegisterResults("w1", "a", []byte("\"row,submission,erp_reference,erp_state,net,tax,quantity\"\n"), human("ops"))), ReasonWrongFormat)
	expect(t, "bad submission id", second(reg("r3,a b,,,3.00,0.00,1.00\n")), "invalid")
	g, _ := newRegistry(t, "g")
	must(g.Generate("w1", settlement("b", line("r1", "1")), human("b")))
	must(g.Confirm("w1", "b", human("lead")))
	only := must(g.RegisterResults("w1", "b", []byte(head+"r1,sub-1,,rejected,1.00,0.00,1.00\n"), human("ops")))
	if only.Handoff.Reference != "" || only.Handoff.State != "rejected" {
		t.Fatalf("rejected without reference %+v", only.Handoff)
	}
	q := settlement("tz", line("r1", "1"))
	a := must(g.Generate("w2", q, human("b")))
	q.ID, q.TimeZone = "tz2", "UTC"
	b := must(g.Generate("w2", q, human("b")))
	if a.Data != b.Data || a.Inputs == b.Inputs {
		t.Fatal("time zone in data or missing from inputs")
	}
	z := must(zip.NewReader(bytes.NewReader(a.Files[0].Bytes), int64(len(a.Files[0].Bytes))))
	for _, f := range z.File {
		if f.Method != zip.Store || !f.Modified.Equal(zipTime) {
			t.Fatalf("entry %s method %d time %v", f.Name, f.Method, f.Modified)
		}
	}
	var many []Line
	for i := 0; i < 7; i++ {
		many = append(many, line("m"+itoa(i), "1"))
	}
	expect(t, "compare limit", second(g.Compare("w1", "b", settlement("b", many...), human("x"))), "limit")
	clock := &testClock{now: start}
	tight := must(NewRegistry(Config{Generator: "g", Schema: "s", Numbers: must(decimal.NewProfile(decimal.ProfileSpec{ID: "t", Revision: "1", InputIntegerDigits: 18, InputScale: 8, ResultIntegerDigits: 18, ResultScale: 8, IntermediateDigits: 80}))}, Limits{1, 1, 6}, clock.read))
	huge := strings.Repeat("9", 18)
	cancel := settlement("c", Line{Row: "a", Source: "s", Product: "p", TaxCode: "A", Quantity: "0", Net: huge, Tax: "0"}, Line{Row: "b", Source: "s", Product: "p", TaxCode: "A", Quantity: "0", Net: huge, Tax: "0"},
		Line{Row: "c", Source: "s", Product: "p", TaxCode: "B", Quantity: "0", Net: "-" + huge, Tax: "0"}, Line{Row: "d", Source: "s", Product: "p", TaxCode: "B", Quantity: "0", Net: "-" + huge, Tax: "0"})
	cancel.Spec.Scale = 0
	expect(t, "per code bound", second(tight.Generate("w", cancel, human("b"))), "limit")
	tx := settlement("d", Line{Row: "a", Source: "s", Product: "p", TaxCode: "A", Quantity: "0", Net: "0", Tax: huge}, Line{Row: "b", Source: "s", Product: "p", TaxCode: "A", Quantity: "0", Net: "0", Tax: huge},
		Line{Row: "c", Source: "s", Product: "p", TaxCode: "B", Quantity: "0", Net: "0", Tax: "-" + huge}, Line{Row: "d", Source: "s", Product: "p", TaxCode: "B", Quantity: "0", Net: "0", Tax: "-" + huge})
	tx.Spec.Scale = 0
	expect(t, "per code tax bound", second(tight.Generate("w", tx, human("b"))), "limit")
}

func TestOverHundredThousandRows(t *testing.T) {
	clock := &testClock{now: start}
	r := must(NewRegistry(Config{Generator: "g", Schema: "s", Numbers: numbers(t)}, Limits{1, 1, 200000}, clock.read))
	lines := make([]Line, 120001)
	for i := range lines {
		lines[i] = Line{Row: "r" + itoa(1000000+i), Source: "s", Product: "p", TaxCode: "t", Quantity: "1", Net: "0.01", Tax: "0"}
	}
	q := settlement("big", lines...)
	q.Kind = KindCustomer
	a := must(r.Generate("w", q, system("job")))
	if n := bytes.Count(a.Files[1].Bytes, []byte("\n")); n != 120002 {
		t.Fatalf("rows %d", n)
	}
	if !strings.Contains(string(parts(t, a.Files[0].Bytes)["xl/worksheets/sheet3.xml"]), ">1200.01<") {
		t.Fatal("tax total")
	}
}

func TestBoundaryGuards(t *testing.T) {
	clock := &testClock{now: start}
	r := must(NewRegistry(Config{Generator: "g", Schema: "s", Numbers: numbers(t)}, Limits{1, 2, 10000}, clock.read))
	lines := make([]Line, 10000)
	for i := range lines {
		lines[i] = line("r"+itoa(100000+i), "1")
	}
	expect(t, "settlement at cap", second(r.Generate("w", settlement("s", lines...), human("b"))), "ok")
	must(r.Confirm("w", "s", human("lead")))
	head := "row,submission,erp_reference,erp_state,net,tax,quantity\n"
	big := head + strings.Repeat("x", 32<<20-len(head))
	expect(t, "exactly max bytes", second(r.RegisterResults("w", "s", []byte(big), human("ops"))), "invalid")
	expect(t, "seven wrong names", second(r.RegisterResults("w", "s", []byte("a,b,c,d,e,f,g\n"), human("ops"))), ReasonWrongFormat)
}
