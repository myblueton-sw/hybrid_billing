package recognition

import (
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestRegistryForms(t *testing.T) {
	clock := &testClock{now: start}
	good := Limits{1, 1, 1, 1, 1}
	if _, err := NewRegistry(decimal.Profile{}, good, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testNumbers(t), good, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	must(NewRegistry(testNumbers(t), good, clock.read))
	must(NewRegistry(testNumbers(t), Limits{100000, 100000, 1000000, 10000, 1000000}, clock.read))
	for _, l := range []Limits{
		{0, 1, 1, 1, 1}, {1, 0, 1, 1, 1}, {1, 1, 0, 1, 1}, {1, 1, 1, 0, 1}, {1, 1, 1, 1, 0},
		{100001, 1, 1, 1, 1}, {1, 100001, 1, 1, 1}, {1, 1, 1000001, 1, 1}, {1, 1, 1, 10001, 1}, {1, 1, 1, 1, 1000001},
	} {
		if _, err := NewRegistry(testNumbers(t), l, clock.read); err != ErrInvalid {
			t.Fatalf("%+v: %v", l, err)
		}
	}
	err := refuse(ReasonEmpty)
	if !errors.Is(err, ErrRejected) || err.Error() != ErrRejected.Error()+": empty" {
		t.Fatal(err)
	}
}

func TestProfileForms(t *testing.T) {
	f := newFixture(t)
	bad := []func(*Profile){
		func(p *Profile) { p.ID = "" },
		func(p *Profile) { p.Entity = "a b" },
		func(p *Profile) { p.Ledger = "" },
		func(p *Profile) { p.Contract = "" },
		func(p *Profile) { p.Basis = "cash" },
		func(p *Profile) { p.TaxBasis = "gross" },
		func(p *Profile) { p.Currency = "krw" },
		func(p *Profile) { p.Scale = -1 },
		func(p *Profile) { p.Scale = 9 },
		func(p *Profile) { p.Method = "guess" },
	}
	for i, change := range bad {
		p := testProfile()
		p.ID = "pf"
		change(&p)
		if got := kind(second(f.r.Define("w1", p, human("c")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	p := testProfile()
	p.ID, p.Scale, p.Basis, p.TaxBasis, p.Method = "p8", 8, "sales_billing", "tax_free", "offset"
	must(f.r.Define("w1", p, human("c")))
	for _, m := range []string{"replace", "reference_link"} {
		q := testProfile()
		q.ID, q.Method, q.Basis = "m-"+m, m, "internal_allocation"
		must(f.r.Define("w1", q, human("c")))
	}
	expect(t, "limit", second(f.r.Define("w1", Profile{ID: "p9", Entity: "e", Ledger: "l", Contract: "c", Basis: "provider_cost", TaxBasis: "tax_free", Currency: "USD", Method: "none"}, human("c"))), "limit")
	expect(t, "assistant", second(f.r.Define("w2", testProfile(), assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Define("w2", testProfile(), system("job"))), ReasonNotHuman)
	if again := must(f.r.Define("w1", testProfile(), human("controller"))); again.Profile != testProfile() {
		t.Fatal("repeat")
	}
	expect(t, "other author", second(f.r.Define("w1", testProfile(), human("x"))), "conflict")
	expect(t, "repeat as system", second(f.r.Define("w1", testProfile(), system("controller"))), "conflict")
	must(f.r.Define("w2", testProfile(), human("c")))
	must(f.r.Define("w3", testProfile(), human("c")))
	expect(t, "workspaces", second(f.r.Define("w4", testProfile(), human("c"))), "limit")
	expect(t, "workspace id", second(f.r.Define("a b", testProfile(), human("c"))), "invalid")
	expect(t, "actor", second(f.r.Define("w1", testProfile(), Actor{ID: "x", Kind: "robot"})), "invalid")
	expect(t, "profile of missing", second(f.r.ProfileOf("w1", "nope")), "not_found")
	expect(t, "profile of ws", second(f.r.ProfileOf("w9", "p1")), "not_found")
}

func TestCloseForms(t *testing.T) {
	f := newFixture(t)
	expect(t, "month", second(f.r.Close("w1", "p1", "2026-13", human("c"))), "invalid")
	expect(t, "ws", second(f.r.Close("", "p1", "2026-01", human("c"))), "invalid")
	expect(t, "id", second(f.r.Close("w1", "", "2026-01", human("c"))), "invalid")
	expect(t, "actor", second(f.r.Close("w1", "p1", "2026-01", human(""))), "invalid")
	expect(t, "missing", second(f.r.Close("w1", "nope", "2026-01", human("c"))), "not_found")
	expect(t, "missing ws", second(f.r.Close("w9", "p1", "2026-01", human("c"))), "not_found")
	expect(t, "assistant", second(f.r.Close("w1", "p1", "2026-01", assistant("bot"))), ReasonAINotAllowed)
	must(f.r.Close("w1", "p1", "2026-03", human("c")))
	must(f.r.Close("w1", "p1", "2026-01", human("c")))
	must(f.r.Close("w1", "p1", "2026-02", human("c")))
	v := must(f.r.Close("w1", "p1", "2026-02", human("c")))
	if strings.Join(v.Closed, ",") != "2026-01,2026-02,2026-03" {
		t.Fatalf("closed %v", v.Closed)
	}
	v.Closed[0] = "x"
	if must(f.r.ProfileOf("w1", "p1")).Closed[0] != "2026-01" {
		t.Fatal("aliasing")
	}
}

func TestMonthForms(t *testing.T) {
	for _, m := range []string{"2000-01", "2099-12", "2026-10", "2026-09"} {
		if !validMonth(m) {
			t.Fatalf("%s refused", m)
		}
	}
	for _, m := range []string{"1999-12", "2100-01", "2026-00", "2026-13", "2026/01", "2026-1", "20a6-01", "2026-0a", "202601", "2026-011"} {
		if validMonth(m) {
			t.Fatalf("%s accepted", m)
		}
	}
	if !validID(strings.Repeat("a", 128)) || validID(strings.Repeat("a", 129)) || validID("") {
		t.Fatal("id length")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-"} {
		if !validID(id) {
			t.Fatalf("%q", id)
		}
	}
	for _, id := range []string{"@", "[", "`", "{", "/", ";", " a", "!a"} {
		if validID(id) {
			t.Fatalf("%q", id)
		}
	}
	if !validDigest(strings.Repeat("0", 32)+strings.Repeat("9", 16)+strings.Repeat("f", 16)) || !validDigest("a"+strings.Repeat("0", 63)) {
		t.Fatal("digest")
	}
	for _, d := range []string{"G" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), ":" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63), "g" + strings.Repeat("0", 63), strings.Repeat("0", 63)} {
		if validDigest(d) {
			t.Fatalf("%q", d)
		}
	}
	for _, c := range []string{"KRW", "AZZ", "ZAA"} {
		if !validCurrency(c) {
			t.Fatal(c)
		}
	}
	for _, c := range []string{"KR", "KRWW", "@RW", "K@W", "KR@", "[RW", "K[W", "KR["} {
		if validCurrency(c) {
			t.Fatal(c)
		}
	}
	if show(big.NewInt(-5), 2) != "-0.05" || show(big.NewInt(1234), 2) != "12.34" || show(big.NewInt(7), 0) != "7" || show(big.NewInt(0), 1) != "0.0" {
		t.Fatal("show")
	}
}

func TestBatchForms(t *testing.T) {
	f := newFixture(t)
	good := initial("b", "2026-01", "1", "0")
	bad := []func(*Batch){
		func(b *Batch) { b.ID = "" },
		func(b *Batch) { b.Profile = "" },
		func(b *Batch) { b.Month = "2026-13" },
		func(b *Batch) { b.Posting = "x" },
		func(b *Batch) { b.Kind = "credit" },
		func(b *Batch) { b.Snapshot = "x" },
		func(b *Batch) { b.Reverses = "x" },
		func(b *Batch) { b.Kind, b.Reverses = BatchReversal, "" },
		func(b *Batch) { b.Kind, b.Reverses = BatchReversal, "a b" },
		func(b *Batch) { b.Kind, b.Reverses = BatchReversal, "x" },
		func(b *Batch) { b.Lines = []Line{line("l", "1", "0"), line("l", "1", "0")} },
		func(b *Batch) { b.Lines = []Line{{ID: "", Lineage: "s", Net: "1", Tax: "0"}} },
		func(b *Batch) { b.Lines = []Line{{ID: "l", Lineage: "", Net: "1", Tax: "0"}} },
		func(b *Batch) { b.Lines = []Line{{ID: "l", Lineage: "s", Net: "x", Tax: "0"}} },
		func(b *Batch) { b.Lines = []Line{{ID: "l", Lineage: "s", Net: "1", Tax: "1e2"}} },
		func(b *Batch) {
			b.Lines = []Line{{ID: "l", Lineage: "s", Net: "1." + strings.Repeat("0", 70), Tax: "0"}}
		},
		func(b *Batch) { b.Lines = []Line{line("l", "1.5", "0")} },
		func(b *Batch) { b.Lines = []Line{line("l", "1", "0.1")} },
	}
	for i, change := range bad {
		b := good
		change(&b)
		if got := kind(second(f.r.Open("w1", b, human("a")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Open("", good, human("a"))), "invalid")
	expect(t, "actor", second(f.r.Open("w1", good, human(""))), "invalid")
	expect(t, "unknown profile", second(f.r.Open("w1", Batch{ID: "b", Profile: "nope", Month: "2026-01", Posting: "2026-01", Kind: BatchInitial, Snapshot: snapshot('a'), Lines: good.Lines}, human("a"))), ReasonUnknownProfile)
	expect(t, "unknown ws", second(f.r.Open("w9", good, human("a"))), ReasonUnknownProfile)
	expect(t, "assistant", second(f.r.Open("w1", good, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Open("w1", good, system("job"))), ReasonNotHuman)
	expect(t, "initial posting", second(f.r.Open("w1", Batch{ID: "b", Profile: "p1", Month: "2026-01", Posting: "2026-02", Kind: BatchInitial, Snapshot: snapshot('a'), Lines: good.Lines}, human("a"))), ReasonPostingMismatch)
	many := good
	many.Lines = []Line{line("1", "1", "0"), line("2", "1", "0"), line("3", "1", "0"), line("4", "1", "0"), line("5", "1", "0")}
	expect(t, "lines", second(f.r.Open("w1", many, human("a"))), "limit")
	huge := good
	huge.Lines = []Line{line("1", "99999999999999", "0"), line("2", "99999999999999", "0"), line("3", "99999999999999", "0"), line("4", "99999999999999", "0")}
	expect(t, "bounds ok", second(f.r.Open("w1", huge, human("a"))), "ok")
	if again := must(f.r.Open("w1", huge, human("a"))); again.Total.Net != "399999999999996" {
		t.Fatalf("repeat %+v", again.Total)
	}
	changed := huge
	changed.Lines = append([]Line(nil), huge.Lines...)
	changed.Lines[0].Lineage = "other"
	expect(t, "changed line", second(f.r.Open("w1", changed, human("a"))), "conflict")
	for _, mutate := range []func(*Batch){
		func(b *Batch) { b.Profile = "p2" },
		func(b *Batch) { b.Month = "2026-02" },
		func(b *Batch) { b.Posting = "2026-02" },
		func(b *Batch) { b.Kind = BatchAmendment },
		func(b *Batch) { b.Provisional = true },
		func(b *Batch) { b.Snapshot = snapshot('f') },
		func(b *Batch) { b.Lines = b.Lines[:3] },
	} {
		c := huge
		c.Lines = append([]Line(nil), huge.Lines...)
		mutate(&c)
		expect(t, "conflict", second(f.r.Open("w1", c, human("a"))), "conflict")
	}
	expect(t, "other author", second(f.r.Open("w1", huge, human("z"))), "conflict")
	expect(t, "system repeat", second(f.r.Open("w1", huge, system("a"))), "conflict")
	expect(t, "batch of", second(f.r.BatchOf("w1", "nope")), "not_found")
	expect(t, "batch of ws", second(f.r.BatchOf("w9", "b")), "not_found")
	v := must(f.r.BatchOf("w1", "b"))
	v.Batch.Lines[0].Net = "x"
	if must(f.r.BatchOf("w1", "b")).Batch.Lines[0].Net == "x" {
		t.Fatal("aliasing")
	}
}

func months(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		y, m := 2026+i/12, i%12+1
		out = append(out, string(rune('0'+y/1000))+string(rune('0'+y/100%10))+string(rune('0'+y/10%10))+string(rune('0'+y%10))+"-"+string(rune('0'+m/10))+string(rune('0'+m%10)))
	}
	return out
}

func TestBatchLimit(t *testing.T) {
	f := newFixture(t)
	list := months(41)
	for _, m := range list[:40] {
		must(f.r.Open("w1", initial("b"+m, m, "1", "0"), human("a")))
	}
	expect(t, "batches", second(f.r.Open("w1", initial("b"+list[40], list[40], "1", "0"), human("a"))), "limit")
}

func wide(t *testing.T) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r := must(NewRegistry(testNumbers(t), Limits{Workspaces: 1, Profiles: 1, Batches: 20, Lines: 200, Adjustments: 1}, clock.read))
	must(r.Define("w1", testProfile(), human("controller")))
	return fixture{r: r, clock: clock}
}

func lines(n int, net string) []Line {
	var out []Line
	for i := 0; i < n; i++ {
		out = append(out, line("l"+months(n)[i], net, "0"))
	}
	return out
}

func TestTotalsBound(t *testing.T) {
	f := wide(t)
	b := initial("big", "2026-01", "0", "0")
	b.Lines = lines(101, "99999999999999")
	expect(t, "batch total", second(f.r.Open("w1", b, human("a"))), "limit")
	b.Lines = lines(100, "99999999999999")
	if v := must(f.r.Open("w1", b, human("a"))); v.Total.Net != "9999999999999900" {
		t.Fatalf("total %+v", v.Total)
	}
	tax := initial("tax", "2026-02", "0", "0")
	tax.Lines = nil
	for i, l := range lines(101, "0") {
		l.Tax = "99999999999999"
		_ = i
		tax.Lines = append(tax.Lines, l)
	}
	expect(t, "tax total", second(f.r.Open("w1", tax, human("a"))), "limit")
	tight := must(decimal.NewProfile(decimal.ProfileSpec{ID: "tight", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 14, ResultScale: 8, IntermediateDigits: 80}))
	clock := &testClock{now: start}
	r := must(NewRegistry(tight, testLimits(), clock.read))
	must(r.Define("w1", testProfile(), human("controller")))
	expect(t, "adjustment bound", second(r.Propose("w1", Adjustment{ID: "x", Profile: "p1", Month: "2026-01", Cause: "other", Before: Amounts{"-99999999999999", "0"}, After: Amounts{"99999999999999", "0"}}, human("a"))), "limit")
	expect(t, "adjustment tax bound", second(r.Propose("w1", Adjustment{ID: "x", Profile: "p1", Month: "2026-01", Cause: "other", Before: Amounts{"0", "-99999999999999"}, After: Amounts{"0", "99999999999999"}}, human("a"))), "limit")
	expect(t, "adjustment in bound", second(r.Propose("w1", Adjustment{ID: "x", Profile: "p1", Month: "2026-01", Cause: "other", Before: Amounts{"0", "0"}, After: Amounts{"99999999999999", "0"}}, human("a"))), "ok")
	g := wide(t)
	for _, m := range []string{"2026-01", "2026-02"} {
		c := initial("b"+m, m, "0", "0")
		c.Lines = lines(100, "99999999999999")
		g.post(t, c)
	}
	expect(t, "reconcile bound", second(g.r.Reconcile("w1", "p1", quarter("Q", "0", "0", "2026-01", "2026-02"))), "limit")
}

func TestApproveVoidAbandonGates(t *testing.T) {
	f := newFixture(t)
	v := must(f.r.Open("w1", initial("jan", "2026-01", "1", "0"), human("author")))
	expect(t, "form", second(f.r.Approve("w1", "jan", "x", human("c"))), "invalid")
	expect(t, "ws", second(f.r.Approve("", "jan", v.Digest, human("c"))), "invalid")
	expect(t, "id", second(f.r.Approve("w1", "", v.Digest, human("c"))), "invalid")
	expect(t, "actor", second(f.r.Approve("w1", "jan", v.Digest, human(""))), "invalid")
	expect(t, "missing", second(f.r.Approve("w1", "nope", v.Digest, human("c"))), "not_found")
	expect(t, "missing ws", second(f.r.Approve("w9", "jan", v.Digest, human("c"))), "not_found")
	expect(t, "assistant", second(f.r.Approve("w1", "jan", v.Digest, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "digest", second(f.r.Approve("w1", "jan", snapshot('0'), human("c"))), ReasonDigestChanged)
	expect(t, "digest before self", second(f.r.Approve("w1", "jan", snapshot('0'), human("author"))), ReasonDigestChanged)
	must(f.r.Close("w1", "p1", "2026-01", human("c")))
	expect(t, "closed", second(f.r.Approve("w1", "jan", v.Digest, human("c"))), ReasonPeriodClosed)
	expect(t, "self before closed", second(f.r.Approve("w1", "jan", v.Digest, human("author"))), ReasonSelfApproval)
	g := newFixture(t)
	w := must(g.r.Open("w1", initial("jan", "2026-01", "1", "0"), human("author")))
	a := must(g.r.Approve("w1", "jan", w.Digest, human("c")))
	if a.Approver != "c" || !a.Approved.Equal(start) {
		t.Fatalf("approved %+v", a)
	}
	if again := must(g.r.Approve("w1", "jan", w.Digest, human("c"))); again.Approver != "c" {
		t.Fatal("repeat")
	}
	expect(t, "system repeat", second(g.r.Approve("w1", "jan", w.Digest, system("c"))), ReasonNotHuman)
	expect(t, "other", second(g.r.Approve("w1", "jan", w.Digest, human("d"))), ReasonNotPending)
	expect(t, "void approved", second(g.r.Void("w1", "jan", human("x"))), ReasonNotPending)
	expect(t, "abandon self", second(g.r.Abandon("w1", "jan", human("author"))), ReasonSelfApproval)
	must(g.r.Submit("w1", "jan", "k", human("a")))
	expect(t, "abandon pending", second(g.r.Abandon("w1", "jan", human("x"))), ReasonOutcomeUnknown)
	must(g.r.Record("w1", "k", Result{State: ResultRejected, Reason: "bad_account"}, system("erp")))
	expect(t, "abandon assistant", second(g.r.Abandon("w1", "jan", assistant("bot"))), ReasonAINotAllowed)
	ab := must(g.r.Abandon("w1", "jan", human("x")))
	if ab.State != BatchAbandoned || ab.EndedBy != "x" {
		t.Fatalf("abandoned %+v", ab)
	}
	if again := must(g.r.Abandon("w1", "jan", human("x"))); again.State != BatchAbandoned {
		t.Fatal("abandon repeat")
	}
	expect(t, "abandon other", second(g.r.Abandon("w1", "jan", human("y"))), ReasonNotPending)
	expect(t, "abandon form", second(g.r.Abandon("w1", "", human("y"))), "invalid")
	expect(t, "abandon ws", second(g.r.Abandon("", "jan", human("y"))), "invalid")
	expect(t, "abandon actor", second(g.r.Abandon("w1", "jan", human(""))), "invalid")
	expect(t, "abandon missing", second(g.r.Abandon("w1", "nope", human("y"))), "not_found")
	expect(t, "abandon missing ws", second(g.r.Abandon("w9", "jan", human("y"))), "not_found")
	expect(t, "submit abandoned", second(g.r.Submit("w1", "jan", "k2", human("a"))), ReasonNotApproved)
	g.post(t, initial("feb", "2026-02", "1", "0"))
	expect(t, "abandon posted", second(g.r.Abandon("w1", "feb", human("y"))), ReasonNotPending)
	must(g.r.Open("w1", initial("mar", "2026-03", "1", "0"), human("author")))
	expect(t, "void assistant", second(g.r.Void("w1", "mar", assistant("bot"))), ReasonAINotAllowed)
	vd := must(g.r.Void("w1", "mar", human("author")))
	if vd.State != BatchVoid || vd.EndedBy != "author" {
		t.Fatalf("void %+v", vd)
	}
	if again := must(g.r.Void("w1", "mar", human("author"))); again.State != BatchVoid {
		t.Fatal("void repeat")
	}
	expect(t, "void other", second(g.r.Void("w1", "mar", human("y"))), ReasonNotPending)
	expect(t, "void form", second(g.r.Void("w1", "", human("y"))), "invalid")
	expect(t, "void ws", second(g.r.Void("", "mar", human("y"))), "invalid")
	expect(t, "void actor", second(g.r.Void("w1", "mar", human(""))), "invalid")
	expect(t, "void missing", second(g.r.Void("w1", "nope", human("y"))), "not_found")
	expect(t, "void missing ws", second(g.r.Void("w9", "mar", human("y"))), "not_found")
	expect(t, "approve void", second(g.r.Approve("w1", "mar", vd.Digest, human("c"))), ReasonNotPending)
	must(g.r.Open("w1", initial("mar2", "2026-03", "1", "0"), human("author")))
	g.post(t, amendment("open-c", "2026-02", "2026-02", "-10", "0"))
	must(g.r.Open("w1", amendment("draft-c", "2026-02", "2026-02", "-10", "0"), human("author")))
	must(g.r.Propose("w1", disc("adj", "2026-02", "draft-c"), human("author")))
	expect(t, "void carrier", second(g.r.Void("w1", "draft-c", human("y"))), ReasonCarriesAdjustment)
}

func TestSubmitAndRecordForms(t *testing.T) {
	f := newFixture(t)
	f.approved(t, initial("jan", "2026-01", "1", "0"))
	must(f.r.Open("w1", initial("feb", "2026-02", "1", "0"), human("a")))
	expect(t, "form", second(f.r.Submit("w1", "jan", "", human("a"))), "invalid")
	expect(t, "ws", second(f.r.Submit("", "jan", "k", human("a"))), "invalid")
	expect(t, "id", second(f.r.Submit("w1", "", "k", human("a"))), "invalid")
	expect(t, "actor", second(f.r.Submit("w1", "jan", "k", human(""))), "invalid")
	expect(t, "missing", second(f.r.Submit("w1", "nope", "k", human("a"))), "not_found")
	expect(t, "missing ws", second(f.r.Submit("w9", "jan", "k", human("a"))), "not_found")
	expect(t, "assistant", second(f.r.Submit("w1", "jan", "k", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "open", second(f.r.Submit("w1", "feb", "k", human("a"))), ReasonNotApproved)
	s := must(f.r.Submit("w1", "jan", "k", human("a")))
	if s.Purpose != Purpose || s.Batch != "jan" || s.Result.State != ResultSent || !s.Sent.Equal(start) {
		t.Fatalf("submission %+v", s)
	}
	expect(t, "other submitter", second(f.r.Submit("w1", "jan", "k", human("b"))), "conflict")
	expect(t, "system repeat", second(f.r.Submit("w1", "jan", "k", system("a"))), "conflict")
	for i, res := range []Result{
		{State: ResultSent},
		{State: "done"},
		{State: ResultPosted},
		{State: ResultPosted, External: "je", Reason: "x"},
		{State: ResultFailed},
		{State: ResultFailed, Reason: "r", External: "a b"},
		{State: ResultRejected},
		{State: ResultAccepted, Reason: "x"},
		{State: ResultUnknown, External: "a b"},
	} {
		if got := kind(second(f.r.Record("w1", "k", res, system("erp")))); got != "invalid" {
			t.Fatalf("result %d: %s", i, got)
		}
	}
	expect(t, "record ws", second(f.r.Record("", "k", Result{State: ResultAccepted}, system("erp"))), "invalid")
	expect(t, "record key", second(f.r.Record("w1", "", Result{State: ResultAccepted}, system("erp"))), "invalid")
	expect(t, "record actor", second(f.r.Record("w1", "k", Result{State: ResultAccepted}, system(""))), "invalid")
	expect(t, "record missing", second(f.r.Record("w1", "nope", Result{State: ResultAccepted}, system("erp"))), "not_found")
	expect(t, "record missing ws", second(f.r.Record("w9", "k", Result{State: ResultAccepted}, system("erp"))), "not_found")
	f.clock.advance(time.Minute)
	r := must(f.r.Record("w1", "k", Result{State: ResultAccepted, External: "queue-1"}, human("ops")))
	if r.Reporter != "ops" || !r.Reported.Equal(start.Add(time.Minute)) {
		t.Fatalf("record %+v", r)
	}
	expect(t, "unknown to sent", second(f.r.Record("w1", "k", Result{State: ResultUnknown}, system("erp"))), "ok")
	expect(t, "unknown to accepted", second(f.r.Record("w1", "k", Result{State: ResultAccepted}, system("erp"))), "ok")
	must(f.r.Record("w1", "k", Result{State: ResultRejected, Reason: "dup", External: "je-x"}, system("erp")))
	expect(t, "rejected final", second(f.r.Record("w1", "k", Result{State: ResultAccepted}, system("erp"))), "conflict")
	expect(t, "submission of", second(f.r.SubmissionOf("w1", "nope")), "not_found")
	expect(t, "submission of ws", second(f.r.SubmissionOf("w9", "k")), "not_found")
	must(f.r.Submit("w1", "jan", "k2", human("a")))
	must(f.r.Record("w1", "k2", Result{State: ResultUnknown}, system("erp")))
	expect(t, "unknown to unknown", second(f.r.Record("w1", "k2", Result{State: ResultUnknown}, system("erp"))), "ok")
	must(f.r.Record("w1", "k2", Result{State: ResultPosted, External: "je-x"}, system("erp")))
	f.approved(t, initial("mar", "2026-03", "1", "0"))
	must(f.r.Submit("w1", "mar", "k3", human("a")))
	must(f.r.Record("w1", "k3", Result{State: ResultFailed, Reason: "r"}, system("erp")))
	expect(t, "failed final", second(f.r.Record("w1", "k3", Result{State: ResultFailed, Reason: "other"}, system("erp"))), "conflict")
}

func TestAdjustmentGates(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	good := disc("a", "2026-03", "")
	bad := []func(*Adjustment){
		func(a *Adjustment) { a.ID = "" },
		func(a *Adjustment) { a.Profile = "" },
		func(a *Adjustment) { a.Month = "2026-13" },
		func(a *Adjustment) { a.Cause = "mood" },
		func(a *Adjustment) { a.Before.Net = "x" },
		func(a *Adjustment) { a.Before.Tax = "x" },
		func(a *Adjustment) { a.After.Net = "x" },
		func(a *Adjustment) { a.After.Tax = "x" },
		func(a *Adjustment) { a.Carrier = "a b" },
		func(a *Adjustment) { a.After.Net = "70.5" },
	}
	for i, change := range bad {
		a := good
		change(&a)
		if got := kind(second(f.r.Propose("w1", a, human("p")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	for _, c := range []string{"tier_pricing", "commitment", "late_source", "rounding", "other"} {
		a := good
		a.ID, a.Cause = "c-"+c, c
		expect(t, c, second(f.r.Propose("w1", a, human("p"))), "ok")
	}
	expect(t, "ws", second(f.r.Propose("", good, human("p"))), "invalid")
	expect(t, "actor", second(f.r.Propose("w1", good, human(""))), "invalid")
	expect(t, "unknown profile", second(f.r.Propose("w1", Adjustment{ID: "a", Profile: "nope", Month: "2026-03", Cause: "other", Before: Amounts{"1", "0"}, After: Amounts{"2", "0"}}, human("p"))), ReasonUnknownProfile)
	expect(t, "unknown ws", second(f.r.Propose("w9", good, human("p"))), ReasonUnknownProfile)
	expect(t, "assistant", second(f.r.Propose("w1", good, assistant("bot"))), ReasonAINotAllowed)
	same := good
	same.After = same.Before
	expect(t, "zero", second(f.r.Propose("w1", same, human("p"))), ReasonEmpty)
	tax := good
	tax.ID, tax.After = "tax", Amounts{"80", "9"}
	if v := must(f.r.Propose("w1", tax, human("p"))); v.Delta != (Amounts{"0", "1"}) {
		t.Fatalf("tax delta %+v", v)
	}
	v := must(f.r.Propose("w1", good, human("p")))
	if again := must(f.r.Propose("w1", good, human("p"))); again.Digest != v.Digest {
		t.Fatal("repeat")
	}
	expect(t, "other proposer", second(f.r.Propose("w1", good, human("q"))), "conflict")
	for i := 0; i < 3; i++ {
		must(f.r.Propose("w1", Adjustment{ID: "z" + string(rune('0'+i)), Profile: "p1", Month: "2026-03", Cause: "other", Before: Amounts{"1", "0"}, After: Amounts{"2", "0"}}, human("p")))
	}
	expect(t, "limit", second(f.r.Propose("w1", Adjustment{ID: "z9", Profile: "p1", Month: "2026-03", Cause: "other", Before: Amounts{"1", "0"}, After: Amounts{"2", "0"}}, human("p"))), "limit")
	d := Decision{Verdict: VerdictApprove, Digest: v.Digest}
	expect(t, "decide form", second(f.r.Decide("w1", "a", Decision{Verdict: "maybe", Digest: v.Digest}, human("c"))), "invalid")
	expect(t, "decide digest form", second(f.r.Decide("w1", "a", Decision{Verdict: VerdictApprove, Digest: "x"}, human("c"))), "invalid")
	expect(t, "decide ws", second(f.r.Decide("", "a", d, human("c"))), "invalid")
	expect(t, "decide id", second(f.r.Decide("w1", "", d, human("c"))), "invalid")
	expect(t, "decide actor", second(f.r.Decide("w1", "a", d, human(""))), "invalid")
	expect(t, "decide missing", second(f.r.Decide("w1", "nope", d, human("c"))), "not_found")
	expect(t, "decide missing ws", second(f.r.Decide("w9", "a", d, human("c"))), "not_found")
	expect(t, "decide assistant", second(f.r.Decide("w1", "a", d, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "decide digest", second(f.r.Decide("w1", "a", Decision{Verdict: VerdictApprove, Digest: snapshot('0')}, human("p"))), ReasonDigestChanged)
	expect(t, "decide self", second(f.r.Decide("w1", "a", d, human("p"))), ReasonSelfApproval)
	f.clock.advance(time.Minute)
	got := must(f.r.Decide("w1", "a", d, human("c")))
	if got.State != AdjustmentApproved || got.Decider != "c" || !got.Decided.Equal(start.Add(time.Minute)) {
		t.Fatalf("decided %+v", got)
	}
	if again := must(f.r.Decide("w1", "a", d, human("c"))); again.State != AdjustmentApproved {
		t.Fatal("decide repeat")
	}
	expect(t, "decide flip", second(f.r.Decide("w1", "a", Decision{Verdict: VerdictReject, Digest: v.Digest}, human("c"))), ReasonNotPending)
	expect(t, "decide system repeat", second(f.r.Decide("w1", "a", d, system("c"))), ReasonNotHuman)
	rej := must(f.r.AdjustmentOf("w1", "z0"))
	rd := Decision{Verdict: VerdictReject, Digest: rej.Digest}
	rv := must(f.r.Decide("w1", "z0", rd, human("c")))
	if rv.State != AdjustmentRejected || rv.EndedBy != "c" {
		t.Fatalf("rejected %+v", rv)
	}
	if again := must(f.r.Decide("w1", "z0", rd, human("c"))); again.State != AdjustmentRejected {
		t.Fatal("reject repeat")
	}
	expect(t, "withdraw rejected", second(f.r.Withdraw("w1", "z0", human("p"))), ReasonNotPending)
	expect(t, "link rejected", second(f.r.Link("w1", "z0", "jan", human("p"))), ReasonNotPending)
	expect(t, "withdraw form", second(f.r.Withdraw("w1", "", human("p"))), "invalid")
	expect(t, "withdraw ws", second(f.r.Withdraw("", "a", human("p"))), "invalid")
	expect(t, "withdraw actor", second(f.r.Withdraw("w1", "a", human(""))), "invalid")
	expect(t, "withdraw missing", second(f.r.Withdraw("w1", "nope", human("p"))), "not_found")
	expect(t, "withdraw assistant", second(f.r.Withdraw("w1", "a", assistant("bot"))), ReasonAINotAllowed)
	wd := must(f.r.Withdraw("w1", "a", human("p")))
	if wd.State != AdjustmentWithdrawn || wd.EndedBy != "p" {
		t.Fatalf("withdrawn %+v", wd)
	}
	if again := must(f.r.Withdraw("w1", "a", human("p"))); again.State != AdjustmentWithdrawn {
		t.Fatal("withdraw repeat")
	}
	expect(t, "withdraw other", second(f.r.Withdraw("w1", "a", human("q"))), ReasonNotPending)
	expect(t, "adjustment of", second(f.r.AdjustmentOf("w1", "nope")), "not_found")
	if got := must(f.r.AdjustmentOf("w1", "a")); got.State != AdjustmentWithdrawn {
		t.Fatal("adjustment of")
	}
}

func TestLinkGates(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	f.approved(t, amendment("c", "2026-03", "2026-04", "-10", "0"))
	f.approved(t, amendment("c2", "2026-03", "2026-04", "-10", "0"))
	must(f.r.Propose("w1", disc("a", "2026-03", ""), human("p")))
	expect(t, "form", second(f.r.Link("w1", "a", "", human("p"))), "invalid")
	expect(t, "ws", second(f.r.Link("", "a", "c", human("p"))), "invalid")
	expect(t, "id", second(f.r.Link("w1", "", "c", human("p"))), "invalid")
	expect(t, "actor", second(f.r.Link("w1", "a", "c", human(""))), "invalid")
	expect(t, "missing", second(f.r.Link("w1", "nope", "c", human("p"))), "not_found")
	expect(t, "assistant", second(f.r.Link("w1", "a", "c", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "unknown batch", second(f.r.Link("w1", "a", "nope", human("p"))), ReasonUnknownBatch)
	v := must(f.r.Link("w1", "a", "c", human("p")))
	if v.Carrier != "c" || v.Adjustment.Carrier != "" {
		t.Fatalf("link %+v", v)
	}
	if again := must(f.r.Link("w1", "a", "c", human("p"))); again.Carrier != "c" || again.Linker != "p" {
		t.Fatal("link repeat")
	}
	expect(t, "link repeat other", second(f.r.Link("w1", "a", "c", human("q"))), "conflict")
	expect(t, "link system repeat", second(f.r.Link("w1", "a", "c", system("p"))), "conflict")
	expect(t, "relink", second(f.r.Link("w1", "a", "c2", human("p"))), "conflict")
	if again := must(f.r.Propose("w1", disc("a", "2026-03", ""), human("p"))); again.Carrier != "c" {
		t.Fatal("propose repeat after link")
	}
}

func TestQuarterForms(t *testing.T) {
	f := newFixture(t)
	good := quarter("Q", "0", "0", q1...)
	bad := []func(*Quarter){
		func(q *Quarter) { q.ID = "" },
		func(q *Quarter) { q.Months = nil },
		func(q *Quarter) { q.Months = []string{"2026-02", "2026-01"} },
		func(q *Quarter) { q.Months = []string{"2026-01", "2026-01"} },
		func(q *Quarter) { q.Months = []string{"2026-13"} },
		func(q *Quarter) {
			q.Months = nil
			for i := 1; i <= 13; i++ {
				q.Months = append(q.Months, "20"+string(rune('2'+i/10))+string(rune('0'+i%10))+"-01")
			}
		},
		func(q *Quarter) { q.Currency = "x" },
		func(q *Quarter) { q.Basis = "x" },
		func(q *Quarter) { q.TaxBasis = "x" },
		func(q *Quarter) { q.Billed.Net = "x" },
		func(q *Quarter) { q.Billed.Tax = "x" },
		func(q *Quarter) { q.Billed.Net = "1.5" },
		func(q *Quarter) { q.Billed.Tax = "1.5" },
	}
	for i, change := range bad {
		q := good
		change(&q)
		if got := kind(second(f.r.Reconcile("w1", "p1", q))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	twelve := good
	twelve.Months = []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06", "2026-07", "2026-08", "2026-09", "2026-10", "2026-11", "2026-12"}
	expect(t, "twelve", second(f.r.Reconcile("w1", "p1", twelve)), "ok")
	expect(t, "ws", second(f.r.Reconcile("", "p1", good)), "invalid")
	expect(t, "profile", second(f.r.Reconcile("w1", "", good)), "invalid")
	expect(t, "missing", second(f.r.Reconcile("w1", "nope", good)), "not_found")
	expect(t, "missing ws", second(f.r.Reconcile("w9", "p1", good)), "not_found")
	expect(t, "clear form", second(f.r.Clear("w1", "p1", good, "x", human("c"))), "invalid")
	expect(t, "clear ws", second(f.r.Clear("", "p1", good, snapshot('0'), human("c"))), "invalid")
	expect(t, "clear profile", second(f.r.Clear("w1", "", good, snapshot('0'), human("c"))), "invalid")
	expect(t, "clear quarter", second(f.r.Clear("w1", "p1", Quarter{}, snapshot('0'), human("c"))), "invalid")
	expect(t, "clear actor", second(f.r.Clear("w1", "p1", good, snapshot('0'), human(""))), "invalid")
	expect(t, "clear missing", second(f.r.Clear("w1", "nope", good, snapshot('0'), human("c"))), "not_found")
	expect(t, "clear missing ws", second(f.r.Clear("w9", "p1", good, snapshot('0'), human("c"))), "not_found")
	expect(t, "clear assistant", second(f.r.Clear("w1", "p1", good, snapshot('0'), assistant("bot"))), ReasonAINotAllowed)
	other := good
	other.Currency = "USD"
	expect(t, "clear basis", second(f.r.Clear("w1", "p1", other, snapshot('0'), human("c"))), ReasonBasisMismatch)
	expect(t, "clearance of", second(f.r.ClearanceOf("w1", "p1", "Q")), "not_found")
	expect(t, "clearance of ws", second(f.r.ClearanceOf("w9", "p1", "Q")), "not_found")
}
