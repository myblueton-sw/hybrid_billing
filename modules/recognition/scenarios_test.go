package recognition

import (
	"testing"
)

var q1 = []string{"2026-01", "2026-02", "2026-03"}

func (f fixture) threeMonths(t *testing.T) {
	t.Helper()
	f.post(t, initial("jan", "2026-01", "100", "10"))
	f.post(t, initial("feb", "2026-02", "120", "12"))
	f.post(t, initial("mar", "2026-03", "80", "8"))
}

func TestMonthlyRecognitionWithoutCustomerInvoice(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	for _, id := range []string{"jan", "feb", "mar"} {
		sub := must(f.r.SubmissionOf("w1", "key-"+id))
		if sub.Purpose != "monthly_recognition" || sub.Result.State != ResultPosted || sub.Result.External != "je-"+id {
			t.Fatalf("submission %+v", sub)
		}
		b := must(f.r.BatchOf("w1", id))
		if !b.Posted || b.External != "je-"+id || b.State != BatchApproved {
			t.Fatalf("batch %+v", b)
		}
	}
	got := f.state(t, quarter("2026Q1", "300", "30", q1...))
	if got.State != StateMatched || got.Recognised != (Amounts{"300", "30"}) || got.Base != (Amounts{"300", "30"}) || got.BaseGross != "330" {
		t.Fatalf("reconcile %+v", got)
	}
}

func TestQuarterlyDiscountAdjustmentCountedOnce(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	q := quarter("2026Q1", "290", "30", q1...)
	if got := f.state(t, q); got.State != StateAmountDifference || got.Difference != (Amounts{"-10", "0"}) || got.Gross != "-10" {
		t.Fatalf("difference %+v", got)
	}
	adj := Adjustment{ID: "disc", Profile: "p1", Month: "2026-03", Cause: "discount", Before: Amounts{"80", "8"}, After: Amounts{"70", "8"}}
	v := must(f.r.Propose("w1", adj, human("accountant")))
	if v.Delta != (Amounts{"-10", "0"}) {
		t.Fatalf("delta %+v", v)
	}
	if got := f.state(t, q); got.State != StateAdjustmentPending || len(got.Pending) != 1 {
		t.Fatalf("pending %+v", got)
	}
	must(f.r.Decide("w1", "disc", Decision{Verdict: VerdictApprove, Digest: v.Digest}, human("controller")))
	got := f.state(t, q)
	if got.State != StateAdjustmentPosting || got.Term != (Amounts{"-10", "0"}) || got.Base.Net != "290" || got.Difference.Net != "0" {
		t.Fatalf("term %+v", got)
	}
	f.approved(t, amendment("mar-disc", "2026-03", "2026-04", "-10", "0"))
	must(f.r.Link("w1", "disc", "mar-disc", human("accountant")))
	if got := f.state(t, q); got.State != StateAdjustmentPosting || got.Term.Net != "-10" || got.Recognised.Net != "300" {
		t.Fatalf("carrier unposted %+v", got)
	}
	must(f.r.Submit("w1", "mar-disc", "key-mar-disc", human("accountant")))
	must(f.r.Record("w1", "key-mar-disc", Result{State: ResultPosted, External: "je-mar-disc"}, system("erp")))
	got = f.state(t, q)
	if got.State != StateMatched || got.Recognised.Net != "290" || got.Term.Net != "0" || got.Base.Net != "290" {
		t.Fatalf("matched %+v", got)
	}
	if got.Months[2].Net != "70" {
		t.Fatalf("march %+v", got.Months[2])
	}
}

func TestMissingOrUnconfirmedMonthIsNotComplete(t *testing.T) {
	f := newFixture(t)
	f.post(t, initial("jan", "2026-01", "100", "10"))
	f.post(t, initial("feb", "2026-02", "120", "12"))
	q := quarter("2026Q1", "220", "22", q1...)
	if got := f.state(t, q); got.State != StateCollecting || got.Months[2].Live {
		t.Fatalf("missing month %+v", got)
	}
	f.approved(t, initial("mar", "2026-03", "0.0", "0"))
	expect(t, "zero initial", second(f.r.Open("w1", initial("mar2", "2026-03", "0", "0"), human("a"))), ReasonInitialExists)
	if got := f.state(t, q); got.State != StatePostingUnconfirmed || got.Difference.Net != "0" {
		t.Fatalf("unposted month %+v", got)
	}
}

func TestUnknownResultBlocksNewKey(t *testing.T) {
	f := newFixture(t)
	f.approved(t, initial("jan", "2026-01", "100", "10"))
	sub := must(f.r.Submit("w1", "jan", "k1", human("accountant")))
	if again := must(f.r.Submit("w1", "jan", "k1", human("accountant"))); again != sub {
		t.Fatal("idempotent key")
	}
	expect(t, "sent", second(f.r.Submit("w1", "jan", "k2", human("accountant"))), ReasonOutcomeUnknown)
	must(f.r.Record("w1", "k1", Result{State: ResultAccepted}, system("erp")))
	expect(t, "accepted", second(f.r.Submit("w1", "jan", "k2", human("accountant"))), ReasonOutcomeUnknown)
	must(f.r.Record("w1", "k1", Result{State: ResultUnknown}, system("erp")))
	expect(t, "unknown", second(f.r.Submit("w1", "jan", "k2", human("accountant"))), ReasonOutcomeUnknown)
	must(f.r.Record("w1", "k1", Result{State: ResultFailed, Reason: "timeout_confirmed_absent"}, system("erp")))
	expect(t, "failed final", second(f.r.Record("w1", "k1", Result{State: ResultPosted, External: "je"}, system("erp"))), "conflict")
	must(f.r.Submit("w1", "jan", "k2", human("accountant")))
	must(f.r.Record("w1", "k2", Result{State: ResultPosted, External: "je-1"}, system("erp")))
	expect(t, "posted", second(f.r.Submit("w1", "jan", "k3", human("accountant"))), ReasonAlreadyPosted)
	if again := must(f.r.Record("w1", "k2", Result{State: ResultPosted, External: "je-1"}, system("erp"))); again.Result.External != "je-1" {
		t.Fatal("repeat result")
	}
	expect(t, "other posting", second(f.r.Record("w1", "k2", Result{State: ResultPosted, External: "je-2"}, system("erp"))), "conflict")
	f.approved(t, initial("feb", "2026-02", "1", "0"))
	must(f.r.Submit("w1", "feb", "k4", human("accountant")))
	expect(t, "external unique", second(f.r.Record("w1", "k4", Result{State: ResultPosted, External: "je-1"}, system("erp"))), "conflict")
	expect(t, "assistant report", second(f.r.Record("w1", "k4", Result{State: ResultAccepted}, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "key reuse", second(f.r.Submit("w1", "feb", "k2", human("accountant"))), "conflict")
}

func TestClearanceRequiresMethodMatchAndDigest(t *testing.T) {
	f := newFixture(t)
	none := testProfile()
	none.ID, none.Method = "p0", MethodNone
	must(f.r.Define("w1", none, human("controller")))
	f.threeMonths(t)
	q := quarter("2026Q1", "300", "30", q1...)
	got := f.state(t, q)
	expect(t, "no method", second(f.r.Clear("w1", "p0", q, got.Digest, human("controller"))), ReasonNoMethod)
	bad := quarter("2026Q1", "299", "30", q1...)
	expect(t, "not matched", second(f.r.Clear("w1", "p1", bad, f.state(t, bad).Digest, human("controller"))), ReasonNotMatched)
	expect(t, "digest", second(f.r.Clear("w1", "p1", q, snapshot('0'), human("controller"))), ReasonDigestChanged)
	c := must(f.r.Clear("w1", "p1", q, got.Digest, human("controller")))
	if c.Method != "reverse_then_confirm" || c.Digest != got.Digest {
		t.Fatalf("clearance %+v", c)
	}
	if again := must(f.r.Clear("w1", "p1", q, got.Digest, human("controller"))); again.At != c.At {
		t.Fatal("repeat")
	}
	expect(t, "same quarter again", second(f.r.Clear("w1", "p1", q, got.Digest, human("other"))), ReasonMonthAlreadyCleared)
	overlap := quarter("Q-overlap", "200", "20", "2026-02", "2026-03")
	expect(t, "overlap", second(f.r.Clear("w1", "p1", overlap, f.state(t, overlap).Digest, human("controller"))), ReasonMonthAlreadyCleared)
	expect(t, "frozen open", second(f.r.Open("w1", amendment("late", "2026-02", "2026-05", "1", "0"), human("a"))), ReasonMonthCleared)
	expect(t, "frozen propose", second(f.r.Propose("w1", Adjustment{ID: "x", Profile: "p1", Month: "2026-01", Cause: "other", Before: Amounts{"1", "0"}, After: Amounts{"2", "0"}}, human("a"))), ReasonMonthCleared)
	if got := must(f.r.ClearanceOf("w1", "p1", "2026Q1")); got.Digest != c.Digest {
		t.Fatal("clearance of")
	}
}

func TestClosedPeriodsAndCorrections(t *testing.T) {
	f := newFixture(t)
	f.post(t, initial("jan", "2026-01", "100", "10"))
	f.approved(t, initial("feb", "2026-02", "50", "5"))
	must(f.r.Close("w1", "p1", "2026-01", human("controller")))
	must(f.r.Close("w1", "p1", "2026-02", human("controller")))
	expect(t, "submit closed", second(f.r.Submit("w1", "feb", "k", human("a"))), ReasonPeriodClosed)
	expect(t, "amend closed", second(f.r.Open("w1", amendment("a1", "2026-01", "2026-01", "5", "0"), human("a"))), ReasonPeriodClosed)
	expect(t, "amend earlier", second(f.r.Open("w1", amendment("a1", "2026-01", "2025-12", "5", "0"), human("a"))), ReasonPostingMismatch)
	a := f.post(t, amendment("a1", "2026-01", "2026-03", "5", "0"))
	if a.Batch.Posting != "2026-03" || a.Batch.Month != "2026-01" {
		t.Fatalf("linked %+v", a)
	}
	r := f.post(t, reversal("r1", "2026-01", "2026-03", "a1"))
	if r.Total != (Amounts{"-5", "0"}) || must(f.r.BatchOf("w1", "a1")).ReversedBy != "r1" {
		t.Fatalf("reversal %+v", r)
	}
	expect(t, "reverse twice", second(f.r.Open("w1", reversal("r2", "2026-01", "2026-03", "a1"), human("a"))), ReasonAlreadyReversed)
	expect(t, "reverse reversal", second(f.r.Open("w1", reversal("r2", "2026-01", "2026-03", "r1"), human("a"))), ReasonTargetIsReversal)
	got := f.state(t, quarter("Q", "100", "10", "2026-01"))
	if got.State != StateMatched || got.Recognised.Net != "100" {
		t.Fatalf("net after reversal %+v", got)
	}
	profile := must(f.r.ProfileOf("w1", "p1"))
	if len(profile.Closed) != 2 || profile.Closed[0] != "2026-01" {
		t.Fatalf("closed %+v", profile.Closed)
	}
}

func TestBasisSeparation(t *testing.T) {
	f := newFixture(t)
	f.threeMonths(t)
	for _, q := range []Quarter{
		{ID: "q", Months: q1, Currency: "USD", Basis: "provider_cost", TaxBasis: "tax_separate", Billed: Amounts{"300", "30"}, Final: true},
		{ID: "q", Months: q1, Currency: "KRW", Basis: "sales_billing", TaxBasis: "tax_separate", Billed: Amounts{"300", "30"}, Final: true},
		{ID: "q", Months: q1, Currency: "KRW", Basis: "provider_cost", TaxBasis: "tax_free", Billed: Amounts{"300", "30"}, Final: true},
	} {
		expect(t, "basis", second(f.r.Reconcile("w1", "p1", q)), ReasonBasisMismatch)
	}
	other := testProfile()
	other.ID, other.Entity = "p2", "seller-kr"
	must(f.r.Define("w1", other, human("controller")))
	if got := must(f.r.Reconcile("w1", "p2", quarter("q", "0", "0", q1...))); got.State != StateCollecting || got.Recognised.Net != "0" {
		t.Fatalf("other profile %+v", got)
	}
}

func TestProvisionalAndFinalNeverMix(t *testing.T) {
	f := newFixture(t)
	prov := initial("jan-p", "2026-01", "90", "9")
	prov.Provisional = true
	f.post(t, prov)
	q := quarter("Q", "90", "9", "2026-01")
	if got := f.state(t, q); got.State != StateMixedBasis || got.Months[0].Provisional != 1 {
		t.Fatalf("final over provisional %+v", got)
	}
	pq := q
	pq.Final = false
	if got := f.state(t, pq); got.State != StateMatched {
		t.Fatalf("provisional quarter %+v", got)
	}
	rev := reversal("jan-p-r", "2026-01", "2026-01", "jan-p")
	rev.Provisional = true
	f.post(t, rev)
	f.post(t, initial("jan", "2026-01", "100", "10"))
	q.Billed = Amounts{"100", "10"}
	if got := f.state(t, q); got.State != StateMatched || got.Recognised.Net != "100" || got.Months[0].Provisional != 0 {
		t.Fatalf("final after replacement %+v", got)
	}
	pq.Billed = q.Billed
	if got := f.state(t, pq); got.State != StateMixedBasis {
		t.Fatalf("provisional over final %+v", got)
	}
	noProv := testProfile()
	noProv.ID, noProv.Provisional = "p3", false
	must(f.r.Define("w1", noProv, human("controller")))
	b := initial("x", "2026-01", "1", "0")
	b.Profile, b.Provisional = "p3", true
	expect(t, "not allowed", second(f.r.Open("w1", b, human("a"))), ReasonProvisionalNotAllowed)
}

func TestNonCalendarQuarterAndSeparation(t *testing.T) {
	f := newFixture(t)
	months := []string{"2026-02", "2026-03", "2026-04"}
	for i, m := range months {
		v := must(f.r.Open("w1", initial("b"+m, m, []string{"10", "20", "30"}[i], "0"), human("accountant")))
		expect(t, "self approve", second(f.r.Approve("w1", "b"+m, v.Digest, human("accountant"))), ReasonSelfApproval)
		must(f.r.Approve("w1", "b"+m, v.Digest, human("controller")))
		must(f.r.Submit("w1", "b"+m, "k"+m, human("accountant")))
		must(f.r.Record("w1", "k"+m, Result{State: ResultPosted, External: "je" + m}, system("erp")))
	}
	if got := f.state(t, quarter("FY-Q4", "60", "0", months...)); got.State != StateMatched {
		t.Fatalf("non-calendar %+v", got)
	}
	changed := testProfile()
	changed.Method = "offset"
	expect(t, "profile immutable", second(f.r.Define("w1", changed, human("controller"))), "conflict")
}
