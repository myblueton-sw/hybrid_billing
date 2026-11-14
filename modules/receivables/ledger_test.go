package receivables

import (
	"testing"
	"time"
)

func ledgerSetup(t *testing.T) (*Registry, *testClock) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.Issue("ws", invoice("d", "k", "100", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	return r, clock
}

func TestPartialPaymentsCreditsAndReversals(t *testing.T) {
	r, clock := ledgerSetup(t)
	must(r.Allocate("ws", receipt("a1", "d", "30.50", "2026-10-02"), "sync"))
	credit := Allocation{ID: "a2", Document: "d", Kind: "credit", Amount: "9.5", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r2"}
	must(r.Allocate("ws", credit, "sync"))
	expect(t, ReasonExceedsOutstanding, second(r.Allocate("ws", receipt("a3", "d", "60.00000001", "2026-10-02"), "sync")), "overpayment")
	if v := must(r.Status("ws", "d")); v.Outstanding != "60" || v.Status != StatusNotDue {
		t.Fatalf("partial: %+v", v)
	}
	expect(t, ReasonFutureEffective, second(r.Allocate("ws", receipt("a3", "d", "1", "2026-10-03"), "sync")), "a future receipt")
	must(r.Allocate("ws", receipt("a3", "d", "60", "2026-10-02"), "sync"))
	if v := must(r.Status("ws", "d")); v.Outstanding != "0" || v.Status != StatusSettled {
		t.Fatalf("settled: %+v", v)
	}
	expect(t, "rejected", second(r.Reverse("ws", Reversal{ID: "v0", Allocation: "a3", Effective: "2026-10-01", Source: SourceAuthority, Ledger: "erp", Revision: "r3"}, "sync")), "a reversal before its allocation")
	clock.set(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC))
	must(r.Confirm("ws", "en", "USD", clock.read(), "sync"))
	must(r.Reverse("ws", Reversal{ID: "v1", Allocation: "a3", Effective: "2026-11-02", Source: SourceAuthority, Ledger: "erp", Revision: "r3"}, "sync"))
	expect(t, "rejected", second(r.Reverse("ws", Reversal{ID: "v2", Allocation: "a3", Effective: "2026-11-02", Source: SourceAuthority, Ledger: "erp", Revision: "r4"}, "sync")), "a second reversal")
	expect(t, "ok", second(r.Reverse("ws", Reversal{ID: "v1", Allocation: "a3", Effective: "2026-11-02", Source: SourceAuthority, Ledger: "erp", Revision: "r3"}, "sync")), "repeat reversal")
	v := must(r.Status("ws", "d"))
	if v.Outstanding != "60" || v.Status != StatusOverdue || v.Days != 3 {
		t.Fatalf("after a chargeback: %+v", v)
	}
	if len(v.Document.Allocations) != 3 || v.Document.Allocations[2].Reversal != "v1" || len(v.Document.Reversals) != 1 {
		t.Fatalf("history: %+v", v.Document)
	}
	cancel := Allocation{ID: "a4", Document: "d", Kind: "cancellation", Amount: "60", Effective: "2026-11-03", Source: SourceAuthority, Ledger: "erp", Revision: "r5"}
	must(r.Allocate("ws", cancel, "sync"))
	if v := must(r.Status("ws", "d")); v.Status != StatusSettled {
		t.Error(v.Status)
	}
	expect(t, ReasonExceedsOutstanding, second(r.Allocate("ws", receipt("a5", "d", "0.1", "2026-11-03"), "sync")), "a rule before the room")
	expect(t, "limit", second(r.Allocate("ws", Allocation{ID: "a5", Document: "d", Kind: "receipt", Amount: "0.1", Effective: "2026-11-03", Source: SourceManual, Evidence: digest('e')}, "clerk")), "allocation room")
	expect(t, "conflict", second(r.Allocate("ws", receipt("a1", "d", "1", "2026-10-02"), "sync")), "conflict")
	expect(t, "conflict", second(r.Allocate("ws", receipt("a1", "d", "30.5", "2026-10-02"), "other")), "another actor")
	expect(t, "ok", second(r.Allocate("ws", receipt("a1", "d", "30.5", "2026-10-02"), "sync")), "a repeat in canonical form")
	expect(t, "not_found", second(r.Allocate("ws", receipt("z", "zz", "1", "2026-10-02"), "sync")), "unknown document")
	expect(t, "not_found", second(r.Reverse("ws", Reversal{ID: "z", Allocation: "zz", Effective: "2026-11-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "sync")), "unknown allocation")
}

func TestManualAllocationsNeedEvidenceAndAnotherApprover(t *testing.T) {
	r, clock := ledgerSetup(t)
	manual := Allocation{ID: "m1", Document: "d", Kind: "receipt", Amount: "70", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}
	v := must(r.Allocate("ws", manual, "clerk"))
	if v.State != StatePending {
		t.Fatal(v.State)
	}
	if s := must(r.Status("ws", "d")); s.Outstanding != "100" {
		t.Error("a pending manual receipt counts")
	}
	must(r.Allocate("ws", receipt("a1", "d", "40", "2026-10-02"), "sync"))
	expect(t, "rejected", second(r.ApproveAllocation("ws", "m1", "clerk")), "self approval")
	expect(t, ReasonExceedsOutstanding, second(r.ApproveAllocation("ws", "m1", "lead")), "exceeds at approval")
	expect(t, "rejected", second(r.Reverse("ws", Reversal{ID: "v", Allocation: "m1", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('b')}, "clerk")), "reverse a pending allocation")
	must(r.WithdrawAllocation("ws", "m1", "clerk"))
	expect(t, "ok", second(r.WithdrawAllocation("ws", "m1", "clerk")), "repeat withdrawal")
	expect(t, "rejected", second(r.WithdrawAllocation("ws", "a1", "clerk")), "withdraw an authority allocation")
	m2 := manual
	m2.ID, m2.Amount = "m2", "60"
	must(r.Allocate("ws", m2, "clerk"))
	must(r.ApproveAllocation("ws", "m2", "lead"))
	expect(t, "ok", second(r.ApproveAllocation("ws", "m2", "lead")), "repeat approval")
	expect(t, "rejected", second(r.ApproveAllocation("ws", "m2", "boss")), "another approval")
	clock.advance(24 * time.Hour)
	rev := Reversal{ID: "v1", Allocation: "m2", Effective: "2026-10-03", Source: SourceManual, Evidence: digest('c')}
	must(r.Reverse("ws", rev, "clerk"))
	if s := must(r.Status("ws", "d")); s.Outstanding != "0" {
		t.Error("a pending reversal already counts")
	}
	expect(t, "rejected", second(r.ApproveReversal("ws", "v1", "clerk")), "self approval of a reversal")
	must(r.WithdrawReversal("ws", "v1", "clerk"))
	if d := must(r.Document("ws", "d")); d.Allocations[2].Reversal != "" {
		t.Error("a withdrawn reversal is still linked")
	}
	rev.ID = "v2"
	must(r.Reverse("ws", rev, "clerk"))
	must(r.ApproveReversal("ws", "v2", "lead"))
	expect(t, "ok", second(r.ApproveReversal("ws", "v2", "lead")), "repeat reversal approval")
	expect(t, "rejected", second(r.WithdrawReversal("ws", "v2", "clerk")), "withdraw an approved reversal")
	if s := must(r.Status("ws", "d")); s.Outstanding != "60" {
		t.Errorf("after the reversal: %s", s.Outstanding)
	}
	for name, a := range map[string]Allocation{
		"no evidence":   {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceManual},
		"manual ledger": {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceManual, Ledger: "erp", Evidence: digest('a')},
		"no ledger":     {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceAuthority, Revision: "r"},
		"no revision":   {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp"},
		"bad evidence":  {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r", Evidence: "x"},
		"source":        {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: "promise", Ledger: "erp", Revision: "r"},
		"kind":          {ID: "x", Document: "d", Kind: "promise", Amount: "1", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"},
		"zero":          {ID: "x", Document: "d", Kind: "receipt", Amount: "0", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"},
		"negative":      {ID: "x", Document: "d", Kind: "receipt", Amount: "-1", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"},
		"scale":         {ID: "x", Document: "d", Kind: "receipt", Amount: "0.000000001", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"},
		"date":          {ID: "x", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-32", Source: SourceAuthority, Ledger: "erp", Revision: "r"},
	} {
		expect(t, "invalid", second(r.Allocate("ws", a, "clerk")), name)
	}
}

func TestInternalDocumentsAndHolds(t *testing.T) {
	r, clock := ledgerSetup(t)
	internal := invoice("i", "nobody", "50", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	internal.Obligation = ObligationInternal
	v := must(r.Issue("ws", internal, "clerk"))
	if v.Snapshot != (Snapshot{}) {
		t.Fatalf("an internal snapshot: %+v", v.Snapshot)
	}
	clock.set(time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
	if s := must(r.Status("ws", "i")); s.Status != StatusNotApplicable || s.Outstanding != "50" {
		t.Errorf("internal: %+v", s)
	}
	expect(t, "rejected", second(r.Allocate("ws", receipt("a", "i", "1", "2026-10-02"), "sync")), "an allocation on an internal document")
	expect(t, "rejected", second(r.Extend("ws", Extension{ID: "x", Document: "i", Due: "2027-12-01", Reason: "billing_error"}, "clerk")), "an extension of an internal document")
	must(r.Confirm("ws", "en", "USD", clock.read(), "sync"))
	before := must(r.Status("ws", "d"))
	must(r.Hold("ws", Hold{ID: "h1", Document: "d", Kind: "dispute", Reference: "case-1"}, "agent"))
	must(r.Hold("ws", Hold{ID: "h2", Document: "d", Kind: "collection_hold", Reference: "memo-1"}, "agent"))
	expect(t, "limit", second(r.Hold("ws", Hold{ID: "h3", Document: "d", Kind: "dispute", Reference: "case-2"}, "agent")), "hold room")
	after := must(r.Status("ws", "d"))
	if after.Status != StatusOverdue || after.Status != before.Status || after.Outstanding != before.Outstanding || after.Days != before.Days || len(after.Document.Holds) != 2 {
		t.Fatalf("a hold changed the obligation: %+v", after)
	}
	must(r.ReleaseHold("ws", "h1", "agent"))
	expect(t, "ok", second(r.ReleaseHold("ws", "h1", "agent")), "repeat release")
	expect(t, "rejected", second(r.ReleaseHold("ws", "h1", "boss")), "another release")
	expect(t, "conflict", second(r.Hold("ws", Hold{ID: "h1", Document: "d", Kind: "dispute", Reference: "case-9"}, "agent")), "hold conflict")
	expect(t, "ok", second(r.Hold("ws", Hold{ID: "h1", Document: "d", Kind: "dispute", Reference: "case-1"}, "agent")), "hold repeat")
	expect(t, "invalid", second(r.Hold("ws", Hold{ID: "h9", Document: "d", Kind: "promise", Reference: "x"}, "agent")), "hold kind")
	expect(t, "not_found", second(r.Hold("ws", Hold{ID: "h9", Document: "zz", Kind: "dispute", Reference: "x"}, "agent")), "hold document")
	expect(t, "not_found", second(r.ReleaseHold("ws", "zz", "agent")), "unknown hold")
	expect(t, "invalid", second(r.Issue("ws", Receivable{Document: "x", Customer: "cu", Entity: "en", Currency: "USD", Contract: "k", Amount: "1", IssuedAt: start, Obligation: ObligationInternal, Replaces: "d", DueBasis: BasisInherit}, "clerk")), "an internal replacement")
}

func TestReplacementsAreSeparateUntilCancelled(t *testing.T) {
	r, clock := ledgerSetup(t)
	clock.set(time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC))
	must(r.Confirm("ws", "en", "USD", clock.read(), "sync"))
	inherit := invoice("d2", "k", "90", time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC))
	inherit.Replaces, inherit.DueBasis = "d", BasisInherit
	v := must(r.Issue("ws", inherit, "clerk"))
	if v.Snapshot.OriginalDue != "2026-10-31" || v.Snapshot.EffectiveDue != "2026-10-31" || v.Snapshot.IssueDate != "2026-11-09" {
		t.Fatalf("inherit: %+v", v.Snapshot)
	}
	again := invoice("d3", "k", "90", time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC))
	again.Replaces, again.DueBasis = "d", BasisReissue
	expect(t, "rejected", second(r.Issue("ws", again, "clerk")), "a second replacement")
	again.Replaces = "d2"
	if w := must(r.Issue("ws", again, "clerk")); w.Snapshot.OriginalDue != "2026-12-09" {
		t.Errorf("reissue: %+v", w.Snapshot)
	}
	if s := must(r.Status("ws", "d")); s.Document.ReplacedBy != "d2" || s.Status != StatusOverdue {
		t.Errorf("replaced: %+v", s)
	}
	a := must(r.Aging("ws", Filter{}))
	if b := a.Currencies[0].Buckets; b[BucketReplaced].Count != 2 || b[BucketReplaced].Amount != "190" || b[BucketNotDue].Amount != "90" || a.Currencies[0].Total != "280" {
		t.Errorf("the replacement window: %+v", b)
	}
	must(r.Allocate("ws", Allocation{ID: "c", Document: "d", Kind: "cancellation", Amount: "100", Effective: "2026-11-09", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "sync"))
	if b := must(r.Aging("ws", Filter{})).Currencies[0].Buckets; b[BucketReplaced].Count != 1 || b[BucketReplaced].Amount != "90" {
		t.Errorf("after cancellation: %+v", b)
	}
	other := invoice("d4", "k", "1", time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC))
	other.Replaces, other.DueBasis, other.Customer = "d3", BasisReissue, "someone"
	expect(t, "rejected", second(r.Issue("ws", other, "clerk")), "another customer's replacement")
	other.Customer, other.Replaces = "cu", "zz"
	expect(t, "not_found", second(r.Issue("ws", other, "clerk")), "unknown replaced document")
	other.Replaces, other.DueBasis = "d3", ""
	expect(t, "invalid", second(r.Issue("ws", other, "clerk")), "no due basis")
}
