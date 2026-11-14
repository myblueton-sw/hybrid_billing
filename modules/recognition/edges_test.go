package recognition

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func disc(id, month, carrier string) Adjustment {
	return Adjustment{ID: id, Profile: "p1", Month: month, Cause: "discount", Before: Amounts{"80", "8"}, After: Amounts{"70", "8"}, Carrier: carrier}
}

func (f fixture) approveAdjustment(t *testing.T, a Adjustment) AdjustmentView {
	t.Helper()
	v := must(f.r.Propose("w1", a, human("accountant")))
	return must(f.r.Decide("w1", a.ID, Decision{Verdict: VerdictApprove, Digest: v.Digest}, human("controller")))
}

func TestOneAdjustmentPerCarrier(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	f.approved(t, amendment("c", "2026-03", "2026-04", "-10", "0"))
	f.approveAdjustment(t, disc("a1", "2026-03", "c"))
	expect(t, "second on propose", second(f.r.Propose("w1", disc("a2", "2026-03", "c"), human("accountant"))), "conflict")
	must(f.r.Propose("w1", disc("a2", "2026-03", ""), human("accountant")))
	expect(t, "second on link", second(f.r.Link("w1", "a2", "c", human("accountant"))), "conflict")
	expect(t, "void carrier", second(f.r.Void("w1", "c", human("x"))), ReasonNotPending)
	expect(t, "abandon carrier", second(f.r.Abandon("w1", "c", human("x"))), ReasonCarriesAdjustment)
	if b := must(f.r.BatchOf("w1", "c")); b.Carries != "a1" {
		t.Fatalf("carries %+v", b)
	}
}

func TestCarrierChecks(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	f.approved(t, amendment("wrong", "2026-03", "2026-04", "-9", "0"))
	f.approved(t, amendment("other-month", "2026-02", "2026-04", "-10", "0"))
	must(f.r.Open("w1", amendment("open-c", "2026-03", "2026-04", "-10", "0"), human("accountant")))
	expect(t, "missing", second(f.r.Propose("w1", disc("a", "2026-03", "nope"), human("accountant"))), ReasonUnknownBatch)
	expect(t, "amount", second(f.r.Propose("w1", disc("a", "2026-03", "wrong"), human("accountant"))), ReasonCarrierMismatch)
	expect(t, "month", second(f.r.Propose("w1", disc("a", "2026-03", "other-month"), human("accountant"))), ReasonCarrierMismatch)
	expect(t, "initial", second(f.r.Propose("w1", disc("a", "2026-03", "mar"), human("accountant"))), ReasonCarrierMismatch)
	must(f.r.Void("w1", "open-c", human("x")))
	expect(t, "void", second(f.r.Propose("w1", disc("a", "2026-03", "open-c"), human("accountant"))), ReasonCarrierMismatch)
}

func TestReversingCarrierRefused(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	f.post(t, amendment("c", "2026-03", "2026-04", "-10", "0"))
	f.approveAdjustment(t, disc("a1", "2026-03", "c"))
	expect(t, "reverse carrier", second(f.r.Open("w1", reversal("r", "2026-03", "2026-04", "c"), human("a"))), ReasonCarriesAdjustment)
	expect(t, "withdraw included", second(f.r.Withdraw("w1", "a1", human("a"))), ReasonCarrierPosted)
	got := f.state(t, quarter("Q", "290", "30", q1...))
	if got.State != StateMatched || got.Term.Net != "0" {
		t.Fatalf("included %+v", got)
	}
}

func TestWithdrawnAdjustmentReleasesCarrier(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	f.approved(t, amendment("c", "2026-03", "2026-04", "-10", "0"))
	f.approveAdjustment(t, disc("a1", "2026-03", "c"))
	q := quarter("Q", "290", "30", q1...)
	if got := f.state(t, q); got.State != StateAdjustmentPosting {
		t.Fatalf("before withdraw %+v", got)
	}
	must(f.r.Withdraw("w1", "a1", human("x")))
	got := f.state(t, q)
	if got.State != StatePostingUnconfirmed || got.Term.Net != "0" {
		t.Fatalf("released carrier blocks %+v", got)
	}
	expect(t, "clear blocked", second(f.r.Clear("w1", "p1", q, got.Digest, human("controller"))), ReasonNotMatched)
	must(f.r.Abandon("w1", "c", human("x")))
	if got := f.state(t, q); got.State != StateAmountDifference {
		t.Fatalf("after abandon %+v", got)
	}
	g := newFixture(t)
	g.threeMonths(t)
	g.approved(t, amendment("c", "2026-03", "2026-04", "-10", "0"))
	v := must(g.r.Propose("w1", disc("a1", "2026-03", "c"), human("accountant")))
	must(g.r.Decide("w1", "a1", Decision{Verdict: VerdictReject, Digest: v.Digest}, human("controller")))
	if got := g.state(t, q); got.State != StatePostingUnconfirmed {
		t.Fatalf("rejected carrier %+v", got)
	}
}

func TestAdjustmentOutsideQuarterIgnored(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	f.post(t, initial("apr", "2026-04", "50", "5"))
	f.approveAdjustment(t, Adjustment{ID: "apr-adj", Profile: "p1", Month: "2026-04", Cause: "credit", Before: Amounts{"50", "5"}, After: Amounts{"40", "5"}})
	got := f.state(t, quarter("Q", "300", "30", q1...))
	if got.State != StateMatched || got.Term.Net != "0" || len(got.Unposted) != 0 {
		t.Fatalf("outside quarter %+v", got)
	}
}

func TestTaxOnlyAmendment(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	b := f.post(t, amendment("tax", "2026-02", "2026-04", "0", "1"))
	if b.Total != (Amounts{"0", "1"}) {
		t.Fatalf("total %+v", b)
	}
	got := f.state(t, quarter("Q", "300", "31", q1...))
	if got.State != StateMatched || got.Recognised.Tax != "31" || got.BaseGross != "331" {
		t.Fatalf("tax only %+v", got)
	}
	expect(t, "zero amendment", second(f.r.Open("w1", amendment("zero", "2026-02", "2026-04", "0", "0"), human("a"))), ReasonEmpty)
	empty := amendment("none", "2026-02", "2026-04", "0", "0")
	empty.Lines = nil
	expect(t, "no lines", second(f.r.Open("w1", empty, human("a"))), ReasonEmpty)
	ie := initial("may", "2026-05", "0", "0")
	ie.Lines = nil
	expect(t, "initial no lines", second(f.r.Open("w1", ie, human("a"))), ReasonEmpty)
}

func TestInitialReplacementRules(t *testing.T) {
	f := newFixture(t)
	f.post(t, initial("jan", "2026-01", "100", "10"))
	f.post(t, amendment("a1", "2026-01", "2026-02", "5", "0"))
	expect(t, "reverse initial with amendment", second(f.r.Open("w1", reversal("r", "2026-01", "2026-02", "jan"), human("a"))), ReasonHasLiveAmendments)
	f.post(t, reversal("ra", "2026-01", "2026-02", "a1"))
	must(f.r.Open("w1", reversal("r", "2026-01", "2026-02", "jan"), human("a")))
	expect(t, "amend while reversing", second(f.r.Open("w1", amendment("a2", "2026-01", "2026-02", "1", "0"), human("a"))), ReasonInitialReversing)
	expect(t, "initial while reversing", second(f.r.Open("w1", initial("jan2", "2026-01", "1", "0"), human("a"))), ReasonInitialExists)
	must(f.r.Void("w1", "r", human("a")))
	if b := must(f.r.BatchOf("w1", "jan")); b.ReversedBy != "" {
		t.Fatalf("void released %+v", b)
	}
	f.post(t, reversal("r2", "2026-01", "2026-02", "jan"))
	f.post(t, initial("jan2", "2026-01", "90", "9"))
	got := f.state(t, quarter("Q", "90", "9", "2026-01"))
	if got.State != StateMatched || got.Recognised.Net != "90" {
		t.Fatalf("replaced %+v", got)
	}
}

func TestAmendmentNeedsPostedInitial(t *testing.T) {
	f := newFixture(t)
	f.approved(t, initial("jan", "2026-01", "100", "10"))
	expect(t, "unposted initial", second(f.r.Open("w1", amendment("a", "2026-01", "2026-01", "5", "0"), human("a"))), ReasonNoInitial)
	expect(t, "no initial", second(f.r.Open("w1", amendment("a", "2026-02", "2026-02", "5", "0"), human("a"))), ReasonNoInitial)
	must(f.r.Abandon("w1", "jan", human("x")))
	if got := f.state(t, quarter("Q", "0", "0", "2026-01")); got.State != StateCollecting {
		t.Fatalf("abandoned initial %+v", got)
	}
	must(f.r.Open("w1", initial("jan2", "2026-01", "1", "0"), human("a")))
	expect(t, "reverse unposted", second(f.r.Open("w1", reversal("r", "2026-01", "2026-01", "jan"), human("a"))), ReasonNoInitial)
}

func TestReversalTargets(t *testing.T) {
	f := newFixture(t)
	f.post(t, initial("jan", "2026-01", "100", "10"))
	f.post(t, initial("feb", "2026-02", "100", "10"))
	f.approved(t, amendment("a1", "2026-01", "2026-01", "5", "0"))
	expect(t, "unknown", second(f.r.Open("w1", reversal("r", "2026-01", "2026-01", "nope"), human("a"))), ReasonUnknownTarget)
	expect(t, "other month", second(f.r.Open("w1", reversal("r", "2026-01", "2026-02", "feb"), human("a"))), ReasonUnknownTarget)
	expect(t, "not posted", second(f.r.Open("w1", reversal("r", "2026-01", "2026-01", "a1"), human("a"))), ReasonTargetNotPosted)
	prov := reversal("r", "2026-01", "2026-01", "jan")
	prov.Provisional = true
	expect(t, "flag mismatch", second(f.r.Open("w1", prov, human("a"))), "invalid")
}

func TestOpenBatchKeepsCollecting(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	must(f.r.Open("w1", amendment("draft", "2026-02", "2026-04", "3", "0"), human("a")))
	if got := f.state(t, quarter("Q", "300", "30", q1...)); got.State != StateCollecting || got.Months[1].Open != 1 {
		t.Fatalf("open batch %+v", got)
	}
}

func TestReversedAmendmentCannotCarry(t *testing.T) {
	for _, posted := range []bool{false, true} {
		f := newFixture(t)
		f.threeMonths(t)
		f.post(t, amendment("c", "2026-03", "2026-04", "-10", "0"))
		if posted {
			f.post(t, reversal("r", "2026-03", "2026-04", "c"))
		} else {
			must(f.r.Open("w1", reversal("r", "2026-03", "2026-04", "c"), human("a")))
		}
		expect(t, "propose", second(f.r.Propose("w1", disc("x", "2026-03", "c"), human("accountant"))), ReasonCarrierMismatch)
		must(f.r.Propose("w1", disc("y", "2026-03", ""), human("accountant")))
		expect(t, "link", second(f.r.Link("w1", "y", "c", human("accountant"))), ReasonCarrierMismatch)
	}
}

func TestRepeatGuards(t *testing.T) {
	f := newFixture(t)
	f.approved(t, initial("jan", "2026-01", "1", "0"))
	must(f.r.Submit("w1", "jan", "k", human("a")))
	must(f.r.Record("w1", "k", Result{State: ResultAccepted}, system("erp")))
	expect(t, "assistant repeat", second(f.r.Record("w1", "k", Result{State: ResultAccepted}, assistant("bot"))), ReasonAINotAllowed)
	g := newFixture(t)
	g.threeMonths(t)
	q := quarter("Q", "300", "30", q1...)
	digest := g.state(t, q).Digest
	must(g.r.Clear("w1", "p1", q, digest, human("controller")))
	other := quarter("Q", "300", "30", "2026-01", "2026-02")
	expect(t, "different quarter same id", second(g.r.Clear("w1", "p1", other, digest, human("controller"))), ReasonMonthAlreadyCleared)
	billed := quarter("Q", "301", "30", q1...)
	expect(t, "different billed", second(g.r.Clear("w1", "p1", billed, digest, human("controller"))), ReasonMonthAlreadyCleared)
	prov := quarter("Q", "300", "30", q1...)
	prov.Final = false
	expect(t, "different final", second(g.r.Clear("w1", "p1", prov, digest, human("controller"))), ReasonMonthAlreadyCleared)
	for _, change := range []func(*Quarter){
		func(x *Quarter) { x.Currency = "USD" },
		func(x *Quarter) { x.Basis = "sales_billing" },
		func(x *Quarter) { x.TaxBasis = "tax_free" },
	} {
		x := quarter("Q", "300", "30", q1...)
		change(&x)
		if sameQuarter(q, x) {
			t.Fatal("same quarter ignores a field")
		}
	}
	if !sameQuarter(q, quarter("Q", "300", "30", q1...)) || sameQuarter(q, quarter("Q2", "300", "30", q1...)) {
		t.Fatal("same quarter")
	}
}

func TestReversalFlagCheckedAfterTarget(t *testing.T) {
	f := newFixture(t)
	f.post(t, initial("jan", "2026-01", "1", "0"))
	r := reversal("r", "2026-02", "2026-02", "jan")
	r.Provisional = true
	f.post(t, initial("feb", "2026-02", "1", "0"))
	expect(t, "other month first", second(f.r.Open("w1", r, human("a"))), ReasonUnknownTarget)
}

func TestMutationGuards(t *testing.T) {
	f := newFixture(t)
	f.approved(t, initial("jan", "2026-01", "1", "0"))
	must(f.r.Submit("w1", "jan", "k", human("a")))
	must(f.r.Record("w1", "k", Result{State: ResultAccepted}, system("erp")))
	expect(t, "accepted to posted", second(f.r.Record("w1", "k", Result{State: ResultPosted, External: "je"}, system("erp"))), "ok")
	two := testProfile()
	two.ID, two.Scale = "p2", 2
	must(f.r.Define("w1", two, human("controller")))
	b := initial("cents", "2026-01", "1.5", "2")
	b.Profile = "p2"
	if v := must(f.r.Open("w1", b, human("a"))); v.Total != (Amounts{"1.50", "2.00"}) {
		t.Fatalf("scaled %+v", v.Total)
	}
	if flag(true) != "1" || flag(false) != "0" {
		t.Fatal("flag")
	}
	sum := sha256.New()
	for _, text := range []string{"ab", "c"} {
		sum.Write([]byte{0, 0, 0, 0, 0, 0, 0, byte(len(text))})
		sum.Write([]byte(text))
	}
	if hashOf("ab", "c") != hex.EncodeToString(sum.Sum(nil)) {
		t.Fatal("hash")
	}
	g := newFixture(t)
	g.post(t, initial("neg", "2026-05", "-10", "0"))
	expect(t, "initial carrier", second(g.r.Propose("w1", Adjustment{ID: "x", Profile: "p1", Month: "2026-05", Cause: "other", Before: Amounts{"0", "0"}, After: Amounts{"-10", "0"}, Carrier: "neg"}, human("a"))), ReasonCarrierMismatch)
	for _, m := range []string{"2026-01", "2000-12", "2099-01"} {
		if !validMonth(m) {
			t.Fatal(m)
		}
	}
	for _, m := range []string{"2026-00", "1999-01", "2126-01", "202:-01", "20:6-01", "2026_01", "2026-1x", "20/6-01", "202/-01"} {
		if validMonth(m) {
			t.Fatal(m)
		}
	}
}
