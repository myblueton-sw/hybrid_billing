package receivables

import (
	"errors"
	"testing"
	"time"
)

func TestTermsNeedAnotherApproverAndCalendarMode(t *testing.T) {
	r, _ := openWith(t, testLimits())
	v := must(r.RecordTerm("ws", Term{ID: "n30", Revision: 1, NetDays: 30, Mode: ModeCalendar}, "clerk"))
	if v.State != StatePending || len(v.Digest) != 64 || v.Author != "clerk" {
		t.Fatalf("%+v", v)
	}
	expect(t, "ok", second(r.RecordTerm("ws", Term{ID: "n30", Revision: 1, NetDays: 30, Mode: ModeCalendar}, "clerk")), "repeat")
	expect(t, "conflict", second(r.RecordTerm("ws", Term{ID: "n30", Revision: 1, NetDays: 31, Mode: ModeCalendar}, "clerk")), "other content")
	expect(t, "conflict", second(r.RecordTerm("ws", Term{ID: "n30", Revision: 1, NetDays: 30, Mode: ModeCalendar}, "other")), "other actor")
	expect(t, "ok", second(r.RecordTerm("ws", Term{ID: "n30", Revision: 2, NetDays: 45, Mode: ModeCalendar}, "clerk")), "a new revision")
	expect(t, "rejected", second(r.ApproveTerm("ws", "n30", 1, "clerk")), "self approval")
	expect(t, "ok", second(r.ApproveTerm("ws", "n30", 1, "lead")), "approval")
	expect(t, "ok", second(r.ApproveTerm("ws", "n30", 1, "lead")), "repeat approval")
	expect(t, "rejected", second(r.ApproveTerm("ws", "n30", 1, "boss")), "a second approver")
	expect(t, "not_found", second(r.ApproveTerm("ws", "n30", 3, "lead")), "unknown revision")
	must(r.RecordTerm("ws", Term{ID: "bd", Revision: 1, NetDays: 10, Mode: ModeBusinessDay}, "clerk"))
	expect(t, "rejected", second(r.ApproveTerm("ws", "bd", 1, "lead")), "business days without a calendar")
	expect(t, "limit", second(r.RecordTerm("ws", Term{ID: "long", Revision: 1, NetDays: 121, Mode: ModeCalendar}, "clerk")), "beyond the net-day limit")
	expect(t, "ok", second(r.RecordTerm("ws", Term{ID: "edge", Revision: 1, NetDays: 120, Mode: ModeCalendar}, "clerk")), "at the net-day limit")
	for name, term := range map[string]Term{
		"id": {ID: "", Revision: 1, Mode: ModeCalendar}, "revision": {ID: "a", Revision: 0, Mode: ModeCalendar},
		"negative": {ID: "a", Revision: 1, NetDays: -1, Mode: ModeCalendar}, "mode": {ID: "a", Revision: 1, Mode: "month_end"},
	} {
		expect(t, "invalid", second(r.RecordTerm("ws", term, "clerk")), name)
	}
	if v := must(r.Term("ws", "n30", 1)); v.State != StateApproved || v.Approver != "lead" {
		t.Errorf("read: %+v", v)
	}
	expect(t, "not_found", second(r.Term("ws", "zz", 1)), "unknown term")
	expect(t, "invalid", second(r.Term("ws", "zz", 0)), "revision")
}

func TestBindingRules(t *testing.T) {
	r, _ := openWith(t, testLimits())
	must(r.RecordTerm("ws", Term{ID: "n30", Revision: 1, NetDays: 30, Mode: ModeCalendar}, "clerk"))
	b := Binding{ID: "b1", Contract: "k", Term: "n30", Revision: 1, Zone: "Asia/Seoul", From: "2026-01-01", To: "2026-06-30", Reason: "initial"}
	must(r.Bind("ws", b, "clerk"))
	expect(t, "rejected", second(r.ApproveBinding("ws", "b1", "lead")), "an unapproved term")
	must(r.ApproveTerm("ws", "n30", 1, "lead"))
	expect(t, "rejected", second(r.ApproveBinding("ws", "b1", "clerk")), "self approval")
	must(r.ApproveBinding("ws", "b1", "lead"))
	other := b
	other.ID, other.Zone, other.From, other.To = "b2", "America/New_York", "2026-07-01", ""
	expect(t, ReasonZoneMismatch, second(r.Bind("ws", other, "clerk")), "another zone")
	other.Zone = "Asia/Seoul"
	other.From = "2026-06-30"
	must(r.Bind("ws", other, "clerk"))
	expect(t, "rejected", second(r.ApproveBinding("ws", "b2", "lead")), "an overlapping binding")
	must(r.WithdrawBinding("ws", "b2", "clerk"))
	expect(t, "ok", second(r.WithdrawBinding("ws", "b2", "clerk")), "repeat withdrawal")
	expect(t, "rejected", second(r.WithdrawBinding("ws", "b2", "x")), "withdrawal by another")
	expect(t, "rejected", second(r.WithdrawBinding("ws", "b1", "clerk")), "withdraw an approved binding")
	other.ID, other.From, other.Zone = "b3", "2026-07-01", "America/New_York"
	expect(t, "ok", second(r.Bind("ws", Binding{ID: "solo", Contract: "k2", Term: "n30", Revision: 1, Zone: "America/New_York", From: "2026-01-01", Reason: "initial"}, "clerk")), "another contract has its own zone")
	other.Zone = "Asia/Seoul"
	must(r.Bind("ws", other, "clerk"))
	must(r.ApproveBinding("ws", "b3", "lead"))
	expect(t, "conflict", second(r.Bind("ws", Binding{ID: "b3", Contract: "k", Term: "n30", Revision: 1, Zone: "Asia/Seoul", From: "2026-07-02", Reason: "initial"}, "clerk")), "binding id conflict")
	expect(t, "not_found", second(r.Bind("ws", Binding{ID: "x", Contract: "k", Term: "none", Revision: 1, Zone: "Asia/Seoul", From: "2026-01-01", Reason: "initial"}, "clerk")), "unknown term")
	for name, change := range map[string]func(*Binding){
		"zone":   func(b *Binding) { b.Zone = "Mars/Base" },
		"local":  func(b *Binding) { b.Zone = "Local" },
		"empty":  func(b *Binding) { b.Zone = "" },
		"to":     func(b *Binding) { b.To = "2025-12-31" },
		"from":   func(b *Binding) { b.From = "2026-02-30" },
		"reason": func(b *Binding) { b.Reason = "because" },
		"rev":    func(b *Binding) { b.Revision = 0 },
		"long":   func(b *Binding) { b.Zone = "America/" + string(make([]byte, 60)) },
	} {
		c := b
		c.ID = "f"
		change(&c)
		expect(t, "invalid", second(r.Bind("ws", c, "clerk")), name)
	}
	nilZone, _ := NewRegistry(testProfile(t), testLimits(), time.Now, func(string) (*time.Location, error) { return nil, nil })
	must(nilZone.RecordTerm("ws", Term{ID: "n", Revision: 1, Mode: ModeCalendar}, "c"))
	expect(t, "invalid", second(nilZone.Bind("ws", Binding{ID: "b", Contract: "k", Term: "n", Revision: 1, Zone: "UTC", From: "2026-01-01", Reason: "initial"}, "c")), "a nil location")
	failing, _ := NewRegistry(testProfile(t), testLimits(), time.Now, func(string) (*time.Location, error) { return nil, errors.New("no zone data") })
	expect(t, "invalid", second(failing.Bind("ws", Binding{ID: "b", Contract: "k", Term: "n", Revision: 1, Zone: "UTC", From: "2026-01-01", Reason: "initial"}, "c")), "a failing resolver")
}

func TestIssueRefusalPrecedence(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	must(r.RecordTerm("ws", Term{ID: "n", Revision: 1, NetDays: 10, Mode: ModeCalendar}, "clerk"))
	must(r.ApproveTerm("ws", "n", 1, "lead"))
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	expect(t, ReasonTermMissing, second(r.Issue("ws", invoice("d", "k", "1", at), "clerk")), "no binding")
	expect(t, ReasonTermMissing, second(r.Issue("other", invoice("d", "k", "1", at), "clerk")), "no workspace")
	bind := func(id, from, to string) {
		must(r.Bind("ws", Binding{ID: id, Contract: "k", Term: "n", Revision: 1, Zone: "UTC", From: from, To: to, Reason: "initial"}, "clerk"))
	}
	bind("p1", "2026-10-01", "2026-10-31")
	expect(t, ReasonTermPending, second(r.Issue("ws", invoice("d", "k", "1", at), "clerk")), "a pending binding")
	bind("p2", "2026-10-05", "2026-10-05")
	expect(t, ReasonTermPending, second(r.Issue("ws", invoice("d", "k", "1", at), "clerk")), "two pending bindings")
	must(r.WithdrawBinding("ws", "p2", "clerk"))
	must(r.ApproveBinding("ws", "p1", "lead"))
	bind("p3", "2026-10-05", "")
	expect(t, ReasonTermConflict, second(r.Issue("ws", invoice("d", "k", "1", at), "clerk")), "an approved and a pending binding")
	must(r.WithdrawBinding("ws", "p3", "clerk"))
	v := must(r.Issue("ws", invoice("d", "k", "1", at), "clerk"))
	if v.Snapshot.Binding != "p1" || v.Snapshot.OriginalDue != "2026-10-15" {
		t.Fatalf("%+v", v.Snapshot)
	}
	expect(t, ReasonTermMissing, second(r.Issue("ws", invoice("e", "k", "1", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)), "clerk")), "after the binding")
	expect(t, "rejected", second(r.Issue("ws", invoice("f", "k", "1", clock.read().Add(time.Second)), "clerk")), "a future issue time")
	if e := must(r.Estimate("ws", "k", "2026-10-20")); e.Due != "2026-10-30" || e.Binding != "p1" {
		t.Errorf("estimate: %+v", e)
	}
	expect(t, ReasonTermMissing, second(r.Estimate("ws", "k", "2026-11-20")), "estimate outside")
	expect(t, ReasonTermMissing, second(r.Estimate("none", "k", "2026-10-20")), "estimate without workspace")
	expect(t, "invalid", second(r.Estimate("ws", "k", "2026-10-32")), "estimate date")
	if _, err := r.Document("ws", "e"); kind(err) != "not_found" {
		t.Error("a refused issue was recorded")
	}
}

func TestSnapshotsSurviveNewTermsAndEnds(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	first := must(r.Issue("ws", invoice("d1", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	expect(t, "rejected", second(r.EndBinding("ws", "k-b", "2026-09-30", "clerk")), "an end before an issued document")
	must(r.EndBinding("ws", "k-b", "2026-10-05", "clerk"))
	expect(t, "ok", second(r.EndBinding("ws", "k-b", "2026-10-05", "clerk")), "repeat end")
	expect(t, "rejected", second(r.EndBinding("ws", "k-b", "2026-10-06", "clerk")), "a second pending end")
	must(r.Issue("ws", invoice("d2", "k", "10", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)), "clerk"))
	expect(t, "rejected", second(r.ApproveEnd("ws", "k-b", "lead")), "an end before a document issued while pending")
	expect(t, "rejected", second(r.ApproveEnd("ws", "k-b", "clerk")), "self approval of an end")
	r2, c2 := openWith(t, testLimits())
	c2.set(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
	setup(t, r2, "k", "UTC", 30)
	must(r2.Issue("ws", invoice("d1", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	must(r2.EndBinding("ws", "k-b", "2026-10-05", "clerk"))
	v := must(r2.ApproveEnd("ws", "k-b", "lead"))
	if v.Binding.To != "2026-10-05" || len(v.Ends) != 1 || v.Ends[0].From != "" || v.Ends[0].Approver != "lead" {
		t.Fatalf("end: %+v", v)
	}
	expect(t, "ok", second(r2.ApproveEnd("ws", "k-b", "lead")), "repeat end approval")
	expect(t, "rejected", second(r2.ApproveEnd("ws", "k-b", "boss")), "another end approval")
	expect(t, "rejected", second(r2.EndBinding("ws", "k-b", "2026-10-05", "clerk")), "an end that does not shorten")
	must(r2.RecordTerm("ws", Term{ID: "n45", Revision: 1, NetDays: 45, Mode: ModeCalendar}, "clerk"))
	must(r2.ApproveTerm("ws", "n45", 1, "lead"))
	must(r2.Bind("ws", Binding{ID: "b2", Contract: "k", Term: "n45", Revision: 1, Zone: "UTC", From: "2026-10-06", Reason: "renegotiation"}, "clerk"))
	must(r2.ApproveBinding("ws", "b2", "lead"))
	later := must(r2.Issue("ws", invoice("d3", "k", "10", time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)), "clerk"))
	if later.Snapshot.NetDays != 45 || later.Snapshot.OriginalDue != "2026-11-21" {
		t.Errorf("new term: %+v", later.Snapshot)
	}
	if old := must(r2.Document("ws", "d1")); old.Snapshot != first.Snapshot {
		t.Errorf("an issued snapshot changed: %+v", old.Snapshot)
	}
	again := must(r2.Issue("ws", invoice("d1", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	if again.Snapshot != first.Snapshot || !again.Written.Equal(first.Written) {
		t.Error("redelivery restarted the clock")
	}
	expect(t, "conflict", second(r2.Issue("ws", invoice("d1", "k", "11", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk")), "other content")
	expect(t, "conflict", second(r2.Issue("ws", invoice("d1", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "other")), "another actor")
	expect(t, "not_found", second(r2.EndBinding("ws", "zz", "2026-10-05", "clerk")), "unknown binding")
	if v := must(r2.BindingOf("ws", "b2")); v.State != StateApproved {
		t.Error(v.State)
	}
}

func TestExtensionsKeepTheOriginalAndPastOverdue(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.Issue("ws", invoice("d", "k", "100", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	expect(t, "rejected", second(r.Extend("ws", Extension{ID: "x0", Document: "d", Due: "2026-10-31", Reason: "customer_request"}, "clerk")), "not later")
	clock.set(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))
	must(r.Extend("ws", Extension{ID: "x1", Document: "d", Due: "2026-11-30", Reason: "dispute_resolution"}, "clerk"))
	expect(t, "rejected", second(r.ApproveExtension("ws", "x1", "clerk")), "self approval")
	v := must(r.ApproveExtension("ws", "x1", "lead"))
	if v.PreviousDue != "2026-10-31" || v.DaysOverdue != 5 || !v.ApprovedAt.Equal(clock.read()) {
		t.Fatalf("history: %+v", v)
	}
	expect(t, "ok", second(r.ApproveExtension("ws", "x1", "lead")), "repeat approval")
	d := must(r.Document("ws", "d"))
	if d.Snapshot.OriginalDue != "2026-10-31" || d.Snapshot.EffectiveDue != "2026-11-30" || len(d.Extensions) != 1 {
		t.Fatalf("snapshot: %+v", d.Snapshot)
	}
	must(r.Extend("ws", Extension{ID: "x2", Document: "d", Due: "2026-12-10", Reason: "billing_error"}, "clerk"))
	must(r.Extend("ws", Extension{ID: "x3", Document: "d", Due: "2026-12-05", Reason: "billing_error"}, "clerk"))
	must(r.ApproveExtension("ws", "x2", "lead"))
	expect(t, "rejected", second(r.ApproveExtension("ws", "x3", "lead")), "an extension that is no longer later")
	must(r.WithdrawExtension("ws", "x3", "clerk"))
	expect(t, "ok", second(r.WithdrawExtension("ws", "x3", "clerk")), "repeat withdrawal")
	expect(t, "rejected", second(r.WithdrawExtension("ws", "x2", "clerk")), "withdraw an approved extension")
	expect(t, "limit", second(r.Extend("ws", Extension{ID: "x4", Document: "d", Due: "2027-01-10", Reason: "billing_error"}, "clerk")), "extension room")
	if v := must(r.ApproveExtension("ws", "x2", "lead")); v.DaysOverdue != 0 || v.PreviousDue != "2026-11-30" {
		t.Errorf("not overdue at approval: %+v", v)
	}
	expect(t, "conflict", second(r.Extend("ws", Extension{ID: "x1", Document: "d", Due: "2026-12-30", Reason: "billing_error"}, "clerk")), "conflict")
	expect(t, "not_found", second(r.Extend("ws", Extension{ID: "y", Document: "zz", Due: "2026-12-30", Reason: "billing_error"}, "clerk")), "unknown document")
	expect(t, "invalid", second(r.Extend("ws", Extension{ID: "y", Document: "d", Due: "2026-12-30", Reason: "because"}, "clerk")), "reason")
}
