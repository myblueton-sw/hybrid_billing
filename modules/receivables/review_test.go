package receivables

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReversalSourcesAndDates(t *testing.T) {
	r, clock := ledgerSetup(t)
	must(r.Allocate("ws", receipt("a", "d", "10", "2026-10-02"), "sync"))
	manual := Allocation{ID: "m", Document: "d", Kind: "receipt", Amount: "5", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}
	must(r.Allocate("ws", manual, "clerk"))
	must(r.ApproveAllocation("ws", "m", "lead"))
	expect(t, "rejected", second(r.Reverse("ws", Reversal{ID: "v1", Allocation: "m", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "clerk")), "an authority reversal of a manual allocation")
	expect(t, "rejected", second(r.Reverse("ws", Reversal{ID: "v2", Allocation: "a", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('b')}, "clerk")), "a manual reversal of an authority allocation")
	expect(t, ReasonFutureEffective, second(r.Reverse("ws", Reversal{ID: "v3", Allocation: "a", Effective: "2026-10-03", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "sync")), "a future reversal")
	clock.advance(24 * time.Hour)
	must(r.Reverse("ws", Reversal{ID: "v4", Allocation: "m", Effective: "2026-10-03", Source: SourceManual, Evidence: digest('b')}, "clerk"))
	must(r.ApproveReversal("ws", "v4", "lead"))
	expect(t, "rejected", second(r.ApproveReversal("ws", "v4", "boss")), "another reversal approver")
	expect(t, "rejected", second(r.WithdrawReversal("ws", "v4", "boss")), "withdraw an approved reversal")
	m2 := manual
	m2.ID = "m2"
	must(r.Allocate("ws", m2, "clerk"))
	must(r.WithdrawAllocation("ws", "m2", "clerk"))
	expect(t, "rejected", second(r.WithdrawAllocation("ws", "m2", "boss")), "another withdrawal")
	expect(t, "rejected", second(r.ApproveAllocation("ws", "m2", "lead")), "approve a withdrawn allocation")
	must(r.Extend("ws", Extension{ID: "x", Document: "d", Due: "2026-12-01", Reason: "billing_error"}, "clerk"))
	must(r.WithdrawExtension("ws", "x", "clerk"))
	expect(t, "rejected", second(r.WithdrawExtension("ws", "x", "boss")), "another extension withdrawal")
	expect(t, "rejected", second(r.ApproveExtension("ws", "x", "lead")), "approve a withdrawn extension")
	expect(t, "not_found", second(r.ApproveReversal("ws", "zz", "lead")), "unknown reversal")
}

func TestEstimatesRefuseLikeIssues(t *testing.T) {
	r, _ := openWith(t, testLimits())
	must(r.RecordTerm("ws", Term{ID: "n", Revision: 1, NetDays: 5, Mode: ModeCalendar}, "clerk"))
	must(r.ApproveTerm("ws", "n", 1, "lead"))
	must(r.Bind("ws", Binding{ID: "p", Contract: "k", Term: "n", Revision: 1, Zone: "UTC", From: "2026-10-01", Reason: "initial"}, "clerk"))
	expect(t, ReasonTermPending, second(r.Estimate("ws", "k", "2026-10-05")), "pending")
	must(r.ApproveBinding("ws", "p", "lead"))
	must(r.Bind("ws", Binding{ID: "q", Contract: "k", Term: "n", Revision: 1, Zone: "UTC", From: "2026-10-05", Reason: "correction"}, "clerk"))
	expect(t, ReasonTermConflict, second(r.Estimate("ws", "k", "2026-10-05")), "conflict")
	if e := must(r.Estimate("ws", "k", "2026-10-04")); e.Due != "2026-10-09" {
		t.Error(e.Due)
	}
	expect(t, "rejected", second(r.WithdrawBinding("ws", "p", "clerk")), "withdraw approved")
	must(r.WithdrawBinding("ws", "q", "clerk"))
	expect(t, "rejected", second(r.WithdrawBinding("ws", "q", "boss")), "another binding withdrawal")
	expect(t, "rejected", second(r.ApproveBinding("ws", "q", "lead")), "approve a withdrawn binding")
	expect(t, "rejected", second(r.ApproveBinding("ws", "p", "boss")), "a second binding approver")
}

func TestBindingRepeatAfterAnEnd(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	original := Binding{ID: "k-b", Contract: "k", Term: "net30", Revision: 1, Zone: "UTC", From: "2026-01-01", Reason: "initial"}
	must(r.EndBinding("ws", "k-b", "2026-12-31", "clerk"))
	must(r.ApproveEnd("ws", "k-b", "lead"))
	v := must(r.Bind("ws", original, "clerk"))
	if v.Binding.To != "2026-12-31" || len(v.Ends) != 1 {
		t.Errorf("repeat after an end: %+v", v)
	}
	must(r.EndBinding("ws", "k-b", "2026-11-30", "clerk"))
	v = must(r.ApproveEnd("ws", "k-b", "lead"))
	if len(v.Ends) != 2 || v.Ends[1].From != "2026-12-31" || v.Ends[1].To != "2026-11-30" {
		t.Errorf("a second end: %+v", v.Ends)
	}
	v.Ends[0].To = "mutated"
	if must(r.BindingOf("ws", "k-b")).Ends[0].To != "2026-12-31" {
		t.Error("a view shares the end history")
	}
}

func TestIssueDateRangeComesBeforeRules(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "Pacific/Kiritimati", 30)
	late := invoice("a", "k", "1", time.Date(2099, 12, 31, 12, 0, 0, 0, time.UTC))
	expect(t, "invalid", second(r.Issue("ws", late, "clerk")), "a future issue beyond the range in the contract zone")
	inside := invoice("b", "k", "1", time.Date(2099, 12, 31, 9, 0, 0, 0, time.UTC))
	expect(t, "rejected", second(r.Issue("ws", inside, "clerk")), "a future issue inside the range")
	internal := invoice("c", "k", "1", time.Date(1999, 12, 31, 23, 0, 0, 0, time.UTC))
	internal.Obligation = ObligationInternal
	expect(t, "invalid", second(r.Issue("ws", internal, "clerk")), "an internal document before the range")
	internal.IssuedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	expect(t, "ok", second(r.Issue("ws", internal, "clerk")), "an internal document at the range")
}

func TestStatusAcrossBerlinDaylightSaving(t *testing.T) {
	r, clock := openWith(t, testLimits())
	berlin, _ := time.LoadLocation("Europe/Berlin")
	clock.set(time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "Europe/Berlin", 0)
	must(r.Issue("ws", invoice("d", "k", "1", time.Date(2026, 3, 28, 23, 30, 0, 0, berlin)), "clerk"))
	for _, c := range []struct {
		at   time.Time
		days int
	}{
		{time.Date(2026, 3, 29, 23, 59, 0, 0, berlin), 1},
		{time.Date(2026, 3, 30, 0, 0, 0, 0, berlin), 2},
		{time.Date(2026, 10, 25, 23, 30, 0, 0, berlin), 211},
	} {
		clock.set(c.at)
		must(r.Confirm("ws", "en", "USD", c.at, "sync"))
		if v := must(r.Status("ws", "d")); v.Days != c.days || v.Status != StatusOverdue {
			t.Errorf("%v: %d %s", c.at, v.Days, v.Status)
		}
	}
}

func TestErrorsAndSharedData(t *testing.T) {
	err := refuse(ReasonTermMissing)
	if !errors.Is(err, ErrRejected) || !strings.HasSuffix(err.Error(), ": term_missing") || !strings.HasPrefix(err.Error(), ErrRejected.Error()) {
		t.Errorf("refusal: %v", err)
	}
	names := BucketNames()
	names[0] = "mutated"
	if BucketNames()[0] != BucketNotDue || len(BucketNames()) != 8 {
		t.Error("bucket names are shared")
	}
	r, _ := ledgerSetup(t)
	must(r.Allocate("ws", receipt("a", "d", "10", "2026-10-02"), "sync"))
	must(r.Hold("ws", Hold{ID: "h", Document: "d", Kind: "dispute", Reference: "c"}, "agent"))
	must(r.Extend("ws", Extension{ID: "x", Document: "d", Due: "2026-12-01", Reason: "billing_error"}, "clerk"))
	v := must(r.Document("ws", "d"))
	v.Allocations[0].State, v.Holds[0].State, v.Extensions[0].State = "x", "x", "x"
	again := must(r.Document("ws", "d"))
	if again.Allocations[0].State != StateApproved || again.Holds[0].State != StateApproved || again.Extensions[0].State != StatePending {
		t.Error("a view shares the document")
	}
}

func TestMutationGaps(t *testing.T) {
	r, clock := ledgerSetup(t)
	expect(t, "ok", second(r.ApproveBinding("ws", "k-b", "lead")), "repeat binding approval")
	expect(t, "ok", second(r.EndBinding("ws", "k-b", "2026-10-01", "clerk")), "an end on the last issue date")
	expect(t, "ok", second(r.ApproveEnd("ws", "k-b", "lead")), "approve an end on the last issue date")
	for name, change := range map[string]func(*Receivable){
		"entity":   func(v *Receivable) { v.Entity = "other" },
		"currency": func(v *Receivable) { v.Currency = "EUR" },
		"contract": func(v *Receivable) { v.Contract = "k2" },
	} {
		v := invoice("n-"+name, "k", "1", time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
		v.Replaces, v.DueBasis = "d", BasisReissue
		change(&v)
		got := kind(second(r.Issue("ws", v, "clerk")))
		if got != "rejected" && got != ReasonTermMissing {
			t.Errorf("a replacement with another %s: %s", name, got)
		}
		if name != "contract" && got != "rejected" {
			t.Errorf("%s: %s", name, got)
		}
	}
	must(r.Extend("ws", Extension{ID: "x1", Document: "d", Due: "2026-11-20", Reason: "billing_error"}, "clerk"))
	expect(t, "conflict", second(r.Extend("ws", Extension{ID: "x1", Document: "d", Due: "2026-11-20", Reason: "billing_error"}, "other")), "an extension repeated by another actor")
	must(r.Extend("ws", Extension{ID: "x2", Document: "d", Due: "2026-11-20", Reason: "billing_error"}, "clerk"))
	must(r.ApproveExtension("ws", "x1", "lead"))
	expect(t, "rejected", second(r.ApproveExtension("ws", "x1", "boss")), "another extension approver")
	expect(t, "rejected", second(r.ApproveExtension("ws", "x2", "lead")), "an extension to the current due date")
	must(r.Allocate("ws", Allocation{ID: "m", Document: "d", Kind: "receipt", Amount: "99.5", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}, "clerk"))
	must(r.ApproveAllocation("ws", "m", "lead"))
	if v := must(r.Status("ws", "d")); v.Outstanding != "0.5" {
		t.Errorf("a fraction below one: %q", v.Outstanding)
	}
	rev := Reversal{ID: "v", Allocation: "m", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('b')}
	must(r.Reverse("ws", rev, "clerk"))
	other := rev
	other.Effective = "2026-10-01"
	expect(t, "conflict", second(r.Reverse("ws", other, "clerk")), "a reversal repeated with other content")
	expect(t, "conflict", second(r.Reverse("ws", rev, "other")), "a reversal repeated by another actor")
	must(r.WithdrawReversal("ws", "v", "clerk"))
	expect(t, "ok", second(r.WithdrawReversal("ws", "v", "clerk")), "repeat reversal withdrawal")
	expect(t, "rejected", second(r.WithdrawReversal("ws", "v", "boss")), "another reversal withdrawal")
	clock.advance(time.Hour)
}

func TestPinnedDigest(t *testing.T) {
	r, _ := openWith(t, testLimits())
	v := must(r.RecordTerm("ws", Term{ID: "n30", Revision: 2, NetDays: 30, Mode: ModeCalendar}, "clerk"))
	sum := sha256.New()
	for _, field := range []string{termLabel, "ws", "n30", "2", "30", "calendar", "clerk"} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		sum.Write(size[:])
		sum.Write([]byte(field))
	}
	if want := hex.EncodeToString(sum.Sum(nil)); v.Digest != want {
		t.Errorf("term digest %s, want %s", v.Digest, want)
	}
}
