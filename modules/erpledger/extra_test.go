package erpledger

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestHelpers(t *testing.T) {
	f := newFixture(t)
	if f.r.number(strings.Repeat("1", 65)) || !f.r.number("1") {
		t.Fatal("number")
	}
	if show(big.NewInt(-5), 2) != "-0.05" || show(big.NewInt(1234), 2) != "12.34" || show(big.NewInt(7), 0) != "7" {
		t.Fatal("show")
	}
	sum := sha256.New()
	for _, text := range []string{"ab", "c"} {
		sum.Write([]byte{0, 0, 0, 0, 0, 0, 0, byte(len(text))})
		sum.Write([]byte(text))
	}
	if hashOf("ab", "c") != hex.EncodeToString(sum.Sum(nil)) {
		t.Fatal("hash")
	}
	for _, m := range []string{"2026-01", "2000-12", "2099-01"} {
		if !validMonth(m) {
			t.Fatal(m)
		}
	}
	for _, m := range []string{"2026-00", "1999-01", "2126-01", "202:-01", "20:6-01", "2026_01", "2026-1x", "20/6-01", "202/-01", "2026-001"} {
		if validMonth(m) {
			t.Fatal(m)
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
}

func TestScaledAmountsAndDocumentConflict(t *testing.T) {
	f := newFixture(t)
	c := apiConnection("cents")
	c.Scale = 2
	f.connect(t, c)
	v := must(f.r.Prepare("w1", invoice("k", "cents", Line{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "1.5", Net: "10.25", Tax: "1"}), human("b")))
	if v.Total != (Amounts{"10.25", "1.00", "1.50"}) {
		t.Fatalf("scaled %+v", v.Total)
	}
	must(f.r.Document("w1", doc("D", 1, row("r1", "1", "0", "1"), row("r2", "2", "0", "1")), system("c")))
	expect(t, "same length changed", second(f.r.Document("w1", doc("D", 1, row("r1", "1", "0", "1"), row("r2", "3", "0", "1")), system("c"))), "conflict")
}

func TestChangeReviewRegressions(t *testing.T) {
	f := newFixture(t)
	must(f.r.Prepare("w1", invoice("k1", "api", line("l1", "s1", "1")), human("biller")))
	must(f.r.Anchor("w1", Anchor{ID: "pp", Connection: "api", Kind: AnchorPrepaid, External: "PP", Customer: "acme", Sources: []string{"s1"}}, human("admin")))
	w := must(f.r.Withdraw("w1", "k1", human("biller")))
	if w.Diverged {
		t.Fatalf("withdrawn stays diverged %+v", w)
	}
	order := invoice("k2", "api", line("l1", "s1", "1"))
	order.Path, order.Anchor = PathOrderBased, "pp"
	must(f.r.Prepare("w1", order, human("biller")))
	for _, name := range []string{CommandStatus, CommandReadDraft, CommandBalance} {
		expect(t, "withdrawn "+name, second(f.r.Command("w1", "k1", name, human("b"))), ReasonWrongState)
	}
	g := newFixture(t)
	g.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	g.observe(t, "api", "k1", "INV", 1, StatePosted, SourceWebhook)
	g.observe(t, "api", "k1", "INV", 2, StateRejected, SourceWebhook)
	v := must(g.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "e"}, human("lead")))
	v.Resolution.Actor = "mallory"
	if must(g.r.SubmissionOf("w1", "k1")).Resolution.Actor != "lead" {
		t.Fatal("resolution aliasing")
	}
	h := newFixture(t)
	h.send(t, invoice("k1", "api", line("l1", "s1", "10")))
	must(h.r.Document("w1", doc("D", 1, row("r1", "10", "0", "1")), system("c")))
	must(h.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "10", Tax: "0", Quantity: "1"}, human("b")))
	must(h.r.Document("w1", doc("D", 2, row("r1", "8", "0", "1")), system("c")))
	relinked := must(h.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "8", Tax: "0", Quantity: "1"}, human("b")))
	if len(relinked.Links) != 2 || !relinked.Links[0].Stale || relinked.Links[1].Stale {
		t.Fatalf("relink %+v", relinked.Links)
	}
	expect(t, "observe assistant repeat", second(h.r.Observe("w1", Observation{ID: "o", Connection: "api", Key: "k1", Revision: 1, State: StateUnknown, Source: SourcePoll}, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "document assistant repeat", second(h.r.Document("w1", doc("D", 2, row("r1", "8", "0", "1")), assistant("bot"))), ReasonAINotAllowed)
	k := newFixture(t)
	must(k.r.Prepare("w1", invoice("k1", "api", line("l1", "s1", "1")), human("biller")))
	must(k.r.Withdraw("w1", "k1", human("biller")))
	k.observe(t, "api", "k1", "INV-W", 1, StatePosted, SourceWebhook)
	acc := must(k.r.Resolve("w1", "k1", Resolution{Outcome: ResolveAccept, Evidence: "e"}, human("lead")))
	if acc.State != StatePosted || acc.Watch == nil {
		t.Fatalf("accepted without watch %+v", acc)
	}
	if due := must(k.r.Due("w1", start.Add(2*time.Hour))); len(due) != 1 {
		t.Fatalf("due after accept %v", due)
	}
	m := newFixture(t)
	m.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	m.observe(t, "api", "k1", "D", 1, StateDraft, SourceWebhook)
	m.observe(t, "api", "k1", "", 2, StateAbsent, SourcePoll)
	m.observe(t, "api", "k1", "D", 3, StatePosted, SourceWebhook)
	if v := must(m.r.SubmissionOf("w1", "k1")); v.LastState != StatePosted || v.State != StatePosted {
		t.Fatalf("premature last state %+v", v)
	}
}

func TestLimitsOfLinksSurveysAndCommands(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 5; i++ {
		must(f.r.Survey("w1", Survey{Connection: "api", Customer: "acme", Period: "2027-0" + string(rune('1'+i)), Outcome: SurveyComplete, Evidence: "e"}, human("a")))
	}
	expect(t, "surveys", second(f.r.Survey("w1", Survey{Connection: "api", Customer: "acme", Period: "2027-09", Outcome: SurveyComplete, Evidence: "e"}, human("a"))), "limit")
	must(f.r.Survey("w1", Survey{Connection: "api", Customer: "acme", Period: "2026-10", Outcome: SurveyUnknown, Evidence: "e2"}, human("a")))
	g := newFixture(t)
	g.send(t, invoice("k1", "api", line("l1", "s1", "100"), line("l2", "s2", "100"), line("l3", "s3", "100"), line("l4", "s4", "100")))
	must(g.r.Document("w1", doc("D", 1, row("r1", "100", "0", "4"), row("r2", "100", "0", "4")), system("c")))
	for i, target := range []string{"l1", "l2", "l3", "l4"} {
		must(g.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: target, Net: "1", Tax: "0", Quantity: "0"}, human("b")))
		if i < 2 {
			must(g.r.Link("w1", Link{Connection: "api", External: "D", Row: "r2", Key: "k1", Line: target, Net: "1", Tax: "0", Quantity: "0"}, human("b")))
		}
	}
	expect(t, "links", second(g.r.Link("w1", Link{Connection: "api", External: "D", Row: "r2", Key: "k1", Line: "l3", Net: "1", Tax: "0", Quantity: "0"}, human("b"))), "limit")
	h := newFixture(t)
	h.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	for i := 1; i < 1000; i++ {
		must(h.r.Command("w1", "k1", CommandStatus, human("b")))
	}
	expect(t, "commands", second(h.r.Command("w1", "k1", CommandStatus, human("b"))), "limit")
}

func tight(t *testing.T) fixture {
	t.Helper()
	numbers := must(decimal.NewProfile(decimal.ProfileSpec{ID: "tight", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 14, ResultScale: 8, IntermediateDigits: 80}))
	clock := &testClock{now: start}
	r := must(NewRegistry(numbers, testLimits(), clock.read))
	f := fixture{r: r, clock: clock}
	f.connect(t, apiConnection("api"))
	return f
}

func TestMutationGuards(t *testing.T) {
	big := "99999999999999"
	for i, l := range [][2]Line{
		{{ID: "a", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "0", Net: big, Tax: "0"}, {ID: "b", Source: "s2", Product: "compute", TaxCode: "vat10", Quantity: "0", Net: big, Tax: "0"}},
		{{ID: "a", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "0", Net: "0", Tax: big}, {ID: "b", Source: "s2", Product: "compute", TaxCode: "vat10", Quantity: "0", Net: "0", Tax: big}},
		{{ID: "a", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: big, Net: "0", Tax: "0"}, {ID: "b", Source: "s2", Product: "compute", TaxCode: "vat10", Quantity: big, Net: "0", Tax: "0"}},
	} {
		f := tight(t)
		expect(t, "submission bound "+string(rune('0'+i)), second(f.r.Prepare("w1", invoice("k", "api", l[0], l[1]), human("b"))), "limit")
		rows := [][2]Row{{row("a", big, "0", "0"), row("b", big, "0", "0")}, {row("a", "0", big, "0"), row("b", "0", big, "0")}, {row("a", "0", "0", big), row("b", "0", "0", big)}}[i]
		expect(t, "document bound", second(f.r.Document("w1", doc("D", 1, rows[0], rows[1]), system("c"))), "limit")
	}
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	o := f.observe(t, "api", "k1", "", 1, StateAbsent, SourcePoll)
	if v := must(f.r.SubmissionOf("w1", "k1")); o.Outcome != "premature" || v.LastState != StateUnknown {
		t.Fatalf("premature %+v", v)
	}
	f.observe(t, "api", "k1", "INV", 2, StatePosted, SourceWebhook)
	f.observe(t, "api", "k1", "INV", 3, StateRejected, SourceWebhook)
	must(f.r.Command("w1", "k1", CommandStatus, human("b")))
	posted := must(f.r.SubmissionOf("w1", "k1")).Posted
	f.clock.advance(time.Hour)
	if v := must(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "e"}, human("lead"))); !v.Posted.Equal(posted) || v.State != StatePosted {
		t.Fatalf("keep changed %+v", v)
	}
	g := newFixture(t)
	must(g.r.Anchor("w1", Anchor{ID: "so", Connection: "api", Kind: AnchorSalesOrder, External: "SO", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	order := invoice("k1", "api", line("l1", "s1", "1"))
	order.Path, order.Anchor = PathOrderBased, "so"
	g.send(t, order)
	g.observe(t, "api", "k1", "INV", 1, StatePosted, SourceWebhook)
	g.observe(t, "api", "k1", "INV", 2, StateRejected, SourceWebhook)
	expect(t, "followed anchor keep", second(g.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "e"}, human("lead"))), "ok")
	changed := order
	changed.Anchor = "so2"
	expect(t, "anchor changed", second(g.r.Prepare("w1", changed, human("biller"))), "conflict")
	h := newFixture(t)
	pay := invoice("k1", "api", line("l1", "s1", "1"))
	pay.Purpose = PurposePayable
	h.send(t, pay)
	must(h.r.Anchor("w1", Anchor{ID: "so", Connection: "api", Kind: AnchorSalesOrder, External: "SO", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	h.observe(t, "api", "k1", "INV", 1, StatePosted, SourceWebhook)
	h.observe(t, "api", "k1", "INV", 2, StateRejected, SourceWebhook)
	expect(t, "non-blocking anchor keep", second(h.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "e"}, human("lead"))), "ok")
	must(h.r.Anchor("w1", Anchor{ID: "ap", Connection: "api", Kind: AnchorPayable, External: "AP", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	if v := must(h.r.SubmissionOf("w1", "k1")); !v.Diverged {
		t.Fatal("ap anchor over payable")
	}
	k := newFixture(t)
	must(k.r.Document("w1", doc("D", 2, row("r1", "1", "0", "1")), system("c")))
	expect(t, "older identical", second(k.r.Document("w1", doc("D", 1, row("r1", "1", "0", "1")), system("c"))), "conflict")
	expect(t, "rows at limit", second(k.r.Document("w1", doc("E", 1, row("1", "1", "0", "1"), row("2", "1", "0", "1"), row("3", "1", "0", "1"), row("4", "1", "0", "1")), system("c"))), "ok")
	k.send(t, invoice("k1", "api", Line{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "1", Net: "1", Tax: "1"}))
	must(k.r.Document("w1", doc("F", 1, row("r1", "1", "1", "1")), system("c")))
	must(k.r.Link("w1", Link{Connection: "api", External: "F", Row: "r1", Key: "k1", Line: "l1", Net: "1", Tax: "0", Quantity: "1"}, human("b")))
	if rec := must(k.r.Reconcile("w1", "k1")); rec.Matched {
		t.Fatal("matched with tax missing")
	}
	m := newFixture(t)
	m.send(t, invoice("k1", "api", Line{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "1", Net: "1", Tax: "1"}))
	must(m.r.Document("w1", doc("F", 1, row("r1", "1", "1", "1")), system("c")))
	must(m.r.Link("w1", Link{Connection: "api", External: "F", Row: "r1", Key: "k1", Line: "l1", Net: "1", Tax: "1", Quantity: "0"}, human("b")))
	if rec := must(m.r.Reconcile("w1", "k1")); rec.Matched {
		t.Fatal("matched with quantity missing")
	}
	if show(big5(), 1) != "0.5" {
		t.Fatal("scale one")
	}
}

func big5() *big.Int { return big.NewInt(5) }

func TestOtherBlockingAnchorOverOrderPath(t *testing.T) {
	f := newFixture(t)
	must(f.r.Anchor("w1", Anchor{ID: "so", Connection: "api", Kind: AnchorSalesOrder, External: "SO", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	order := invoice("k1", "api", line("l1", "s1", "1"))
	order.Path, order.Anchor = PathOrderBased, "so"
	must(f.r.Prepare("w1", order, human("biller")))
	must(f.r.Anchor("w1", Anchor{ID: "k", Connection: "api", Kind: AnchorContract, External: "K", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	expect(t, "other anchor keep", second(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "e"}, human("lead"))), ReasonAlreadyReflected)
}
