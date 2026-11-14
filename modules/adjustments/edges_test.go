package adjustments

import (
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func tinyRegistry(t *testing.T) *Registry {
	t.Helper()
	tiny, err := decimal.NewProfile(decimal.ProfileSpec{ID: "tiny", Revision: "1", InputIntegerDigits: 3, InputScale: 2, ResultIntegerDigits: 3, ResultScale: 2, IntermediateDigits: 10})
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRegistry(tiny, testLimits(), (&testClock{now: start}).read)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func tinyDraft(amounts ...string) Draft {
	d := Draft{ID: "d1", Customer: "c", Contract: "k", From: "2026-10-01", To: "2026-10-31", Base: base(false)}
	for index, a := range amounts {
		d.Lines = append(d.Lines, SourceLine{ID: "l" + strconv.Itoa(index+1), Charge: "c", Quantity: "1", Amount: a})
	}
	return d
}

func TestBoundsWithFractions(t *testing.T) {
	r := tinyRegistry(t)
	v := must(r.Open("ws", tinyDraft("999", "0.5"), human("w")))
	if v.Source != "999.5" {
		t.Fatalf("source %s", v.Source)
	}
}

func TestRemoveBeyondProfile(t *testing.T) {
	r := tinyRegistry(t)
	v := must(r.Open("ws", tinyDraft("500", "499"), human("w")))
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "l2", "", "-499"), human("w")))
	v = must(r.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "l1", "", "499"), human("w")))
	expect(t, "limit", second(r.Remove("ws", "d1", v.Revision, "e1", human("w"))), "removal leaves total beyond profile")
	after := must(r.Of("ws", "d1"))
	if after.Revision != v.Revision || !after.Entries[0].Active || after.Total != "999" {
		t.Fatalf("removal recorded %+v", after)
	}
}

func TestNumberLength(t *testing.T) {
	wide, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 60, InputScale: 8, ResultIntegerDigits: 62, ResultScale: 8, IntermediateDigits: 200})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewRegistry(wide, testLimits(), (&testClock{now: start}).read)
	d := tinyDraft("1")
	d.Lines[0].Quantity = strings.Repeat("1", 55) + "." + strings.Repeat("1", 8)
	expect(t, "ok", second(r.Open("ws", d, human("w"))), "64 bytes")
	d.ID, d.Lines[0].Quantity = "d2", strings.Repeat("1", 56)+"."+strings.Repeat("1", 8)
	expect(t, "invalid", second(r.Open("ws", d, human("w"))), "65 bytes")
}

func TestFormEdges(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	d.From, d.To = "2026-10-15", "2026-10-15"
	d.Base.Scale = 0
	d.Lines[0].Amount = "25"
	v := must(r.Open("ws", d, human("w")))
	e := item("e1", EntryFixed, "", "", "1")
	e.From, e.To = "2026-10-15", "2026-10-15"
	expect(t, "ok", second(r.Add("ws", "d1", v.Revision, e, human("w"))), "one-day entry")
	d = sample("d2", false)
	d.Base.Scale = 8
	expect(t, "ok", second(r.Open("ws", d, human("w"))), "scale eight")
	for _, c := range []string{"AAA", "ZZZ", "AZA", "ZAZ"} {
		if !validCurrency(c) {
			t.Errorf("currency %s", c)
		}
	}
	for _, c := range []string{"@AA", "A@A", "AA@", "[AA", "A[A", "AA["} {
		if validCurrency(c) {
			t.Errorf("currency %s", c)
		}
	}
	if validID("/a") || validID(" a") || validDigest("g"+digest('a')[1:]) || validDigest("A"+digest('a')[1:]) {
		t.Error("first character")
	}
	if !validDate("2099-12-31") || validDate("2099-12-32") || validDate("2099-13-01") {
		t.Error("date edges")
	}
}

func TestEntryReplayComparesEveryField(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	e := item("e1", EntryFixed, "l1", "", "5")
	e.Finding = "f1"
	must(r.Add("ws", "d1", v.Revision, e, human("clerk")))
	for index, change := range []func(*Entry){
		func(x *Entry) { x.Kind, x.Amount, x.Target = EntryTargetTotal, "", "30" },
		func(x *Entry) { x.Line = "l2" },
		func(x *Entry) { x.Line = "" },
		func(x *Entry) { x.Amount = "6" },
		func(x *Entry) { x.Reason = ReasonGoodwill },
		func(x *Entry) { x.Finding = "f2" },
		func(x *Entry) { x.Finding = "" },
		func(x *Entry) { x.From = "2026-10-02" },
		func(x *Entry) { x.To = "2026-10-30" },
		func(x *Entry) { x.Evidence = []string{"other"} },
		func(x *Entry) { x.Evidence = []string{"ev-e1", "more"} },
	} {
		x := copyEntry(e)
		change(&x)
		expect(t, "conflict", second(r.Add("ws", "d1", v.Revision, x, human("clerk"))), "changed field "+strconv.Itoa(index))
	}
	expect(t, "ok", second(r.Add("ws", "d1", v.Revision, copyEntry(e), human("clerk"))), "exact replay")
}

func TestReleaseReplayComparesEveryField(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	q := release("r1", v, human("ann"), human("bob"))
	must(r.Release("ws", "d1", q, system("runner")))
	for index, change := range []func(*Release){
		func(x *Release) { x.Revision++ },
		func(x *Release) { x.Subject = digest('9') },
		func(x *Release) { x.Base.Run = "run-9" },
		func(x *Release) { x.Approval = "other" },
		func(x *Release) { x.Approvers = x.Approvers[:1] },
		func(x *Release) { x.Approvers = []Actor{human("ann"), human("cy")} },
	} {
		x := q
		x.Approvers = append([]Actor(nil), q.Approvers...)
		change(&x)
		expect(t, "conflict", second(r.Release("ws", "d1", x, system("runner"))), "changed field "+strconv.Itoa(index))
	}
}

func TestZeroTotalReleases(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryTargetTotal, "", "0", ""))
	out := must(r.Release("ws", "d1", release("r1", v, human("ann")), system("runner")))
	if out.Total != "0" || out.Gross != "100" || out.Adjustment != "-100" {
		t.Fatalf("outcome %+v", out)
	}
}

func TestSignedFinal(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryTargetTotal, "", "1", ""), human("clerk")))
	v = must(r.Validate("ws", "d1", check(v, "-20", "-19"), system("tax")))
	out := must(r.Release("ws", "d1", release("r1", v, human("ann")), system("runner")))
	if out.Final != "-19" || out.TaxAmount != "-20" {
		t.Fatalf("outcome %+v", out)
	}
}

func TestEarlierCheckerCannotApprove(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	v = must(r.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "", "", "1"), human("clerk")))
	v = must(r.Validate("ws", "d1", check(v, "0", v.Total), system("second-engine")))
	expect(t, RefuseSelfApproval, r.Ready("ws", "d1", release("r1", v, human("tax-engine")), system("runner")), "earlier checker")
	expect(t, "ok", r.Ready("ws", "d1", release("r1", v, human("ann")), system("runner")), "independent")
}

func TestOversizedLinesBeforeParsing(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	for index := range 10 {
		d.Lines = append(d.Lines, SourceLine{ID: "bad id " + strconv.Itoa(index)})
	}
	expect(t, "limit", second(r.Open("ws", d, human("w"))), "oversized list")
}

func TestDraftDigestCoversEveryField(t *testing.T) {
	digestOf := func(dc func(*Draft), ec func(*Entry), author Actor) string {
		r, _ := setup(t)
		d := sample("d1", false)
		d.Lines[0].Price = "2.5"
		if dc != nil {
			dc(&d)
		}
		v := must(r.Open("ws", d, human("w")))
		e := item("e1", EntryFixed, "l1", "", "5")
		e.Finding = "f1"
		if ec != nil {
			ec(&e)
		}
		return must(r.Add("ws", "d1", v.Revision, e, author)).Digest
	}
	seen := map[string]int{}
	add := func(name string, value string) {
		if old, ok := seen[value]; ok {
			t.Errorf("%s has the same digest as case %d", name, old)
		}
		seen[value] = len(seen)
	}
	add("base", digestOf(nil, nil, human("clerk")))
	add("author id", digestOf(nil, nil, human("clerk2")))
	add("author kind", digestOf(nil, nil, system("clerk")))
	for index, dc := range []func(*Draft){
		func(d *Draft) { d.Customer = "c2" },
		func(d *Draft) { d.Contract = "k2" },
		func(d *Draft) { d.From = "2026-09-01" },
		func(d *Draft) { d.To = "2026-11-30" },
		func(d *Draft) { d.Base.Run = "run-2" },
		func(d *Draft) { d.Base.Digest = digest('b') },
		func(d *Draft) { d.Base.Issued, d.Base.Document = true, "INV-1" },
		func(d *Draft) { d.Base.Currency = "EUR" },
		func(d *Draft) { d.Base.Scale = 3 },
		func(d *Draft) { d.Base.Mode = "floor" },
		func(d *Draft) { d.Lines[0].Charge = "ch-9" },
		func(d *Draft) { d.Lines[0].Quantity = "11" },
		func(d *Draft) { d.Lines[0].Price = "2.6" },
		func(d *Draft) { d.Lines[1].Amount = "76" },
		func(d *Draft) { d.Lines[1].ID = "l9" },
	} {
		add("draft "+strconv.Itoa(index), digestOf(dc, nil, human("clerk")))
	}
	for index, ec := range []func(*Entry){
		func(e *Entry) { e.ID = "e9" },
		func(e *Entry) { e.Amount = "6" },
		func(e *Entry) { e.Line = "l2" },
		func(e *Entry) { e.Line = "" },
		func(e *Entry) { e.Kind, e.Amount, e.Target = EntryTargetTotal, "", "30" },
		func(e *Entry) { e.Reason = ReasonGoodwill },
		func(e *Entry) { e.Finding = "f2" },
		func(e *Entry) { e.Finding = "" },
		func(e *Entry) { e.From = "2026-10-02" },
		func(e *Entry) { e.To = "2026-10-30" },
		func(e *Entry) { e.Evidence = []string{"other"} },
		func(e *Entry) { e.Evidence = []string{"ev-e1", "x"} },
	} {
		add("entry "+strconv.Itoa(index), digestOf(nil, ec, human("clerk")))
	}
}

func TestDatesValidatedIndividually(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	d.From = "2026-02-30"
	expect(t, "invalid", second(r.Open("ws", d, human("w"))), "invalid from before to")
	v := opened(t, r, sample("d1", false))
	e := item("e1", EntryFixed, "", "", "1")
	e.From = "2026-10-00"
	expect(t, "invalid", second(r.Add("ws", "d1", v.Revision, e, human("w"))), "invalid entry from")
}

func TestEntryKindAloneConflicts(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	must(r.Add("ws", "d1", v.Revision, item("e1", EntryQuantity, "l1", "12", ""), human("clerk")))
	expect(t, "conflict", second(r.Add("ws", "d1", v.Revision, item("e1", EntryTargetTotal, "l1", "12", ""), human("clerk"))), "kind only")
}
