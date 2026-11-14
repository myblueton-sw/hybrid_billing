package components

import (
	"testing"
	"time"
)

func slaSetup(t *testing.T) *Registry {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("rack", "1000", Inclusion{Meter: "power", Quantity: All}))
	must(r.RecordIncident("ws", Incident{ID: "i1", Contract: "k", Component: "rack", Start: time.Date(2026, 10, 3, 1, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 3, 5, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	must(r.RecordIncident("ws", Incident{ID: "i2", Contract: "k", Component: "rack", Start: time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 20, 2, 0, 0, 0, time.UTC), Evidence: digest('b')}, "noc"))
	return r
}

func october(id, clause string, incidents ...string) Credit {
	return Credit{ID: id, Clause: clause, Incidents: incidents, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"}
}

func TestIncidentsAloneNeverCredit(t *testing.T) {
	r := slaSetup(t)
	s := must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"}))
	if len(s.Credits) != 0 || s.Net != "1000" {
		t.Fatalf("an incident changed the statement: %+v", s)
	}
	expect(t, "not_found", second(r.ProposeCredit("ws", october("c", "none", "i1"), "clerk")), "a credit without a clause")
	must(r.AddClause("ws", Clause{ID: "sla", Contract: "k", Component: "rack", Rate: "0.1", From: "2026-01-01"}, "legal"))
	expect(t, "rejected", second(r.ProposeCredit("ws", october("c", "sla", "i1"), "clerk")), "a credit under an unapproved clause")
	expect(t, "rejected", second(r.ApproveClause("ws", "sla", "legal")), "self approval of a clause")
	must(r.ApproveClause("ws", "sla", "lead"))
	v := must(r.ProposeCredit("ws", october("c", "sla", "i1", "i2"), "clerk"))
	if v.State != StatePending || v.Amount != "100" || v.Component != "rack" || len(v.Digest) != 64 {
		t.Fatalf("proposed: %+v", v)
	}
	if s := must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"})); len(s.Credits) != 0 {
		t.Error("a pending credit appears")
	}
	expect(t, "rejected", second(r.ApproveCredit("ws", "c", "clerk")), "self approval of a credit")
	must(r.ApproveCredit("ws", "c", "ops-lead"))
	expect(t, "ok", second(r.ApproveCredit("ws", "c", "ops-lead")), "repeat approval")
	expect(t, "rejected", second(r.ApproveCredit("ws", "c", "boss")), "another approver")
	s = must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"}))
	if len(s.Credits) != 1 || s.Credits[0].Amount != "100" || s.Credits[0].State != CreditUnlinked || s.Net != "1000" || s.Charges != "1000" {
		t.Fatalf("an approved credit: %+v", s)
	}
	expect(t, "rejected", second(r.WithdrawCredit("ws", "c", "clerk")), "withdraw an approved credit")
	must(r.LinkCredit("ws", "c", Link{Document: "cn-1", Increase: "inc-1"}, "ar"))
	expect(t, "ok", second(r.LinkCredit("ws", "c", Link{Document: "cn-1", Increase: "inc-1"}, "ar")), "repeat link")
	expect(t, "conflict", second(r.LinkCredit("ws", "c", Link{Document: "cn-2", Increase: "inc-1"}, "ar")), "relink")
	s = must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"}))
	if s.Credits[0].State != CreditLinked || s.Credits[0].Document != "cn-1" || s.Credits[0].Increase != "inc-1" || s.Net != "1000" {
		t.Errorf("a linked credit lowers the statement: %+v", s)
	}
	if s := must(r.Statement("ws", november())); len(s.Credits) != 0 {
		t.Error("a credit appears in another month")
	}
}

func TestOneBreachOneCredit(t *testing.T) {
	r := slaSetup(t)
	must(r.AddClause("ws", Clause{ID: "a", Contract: "k", Component: "rack", Rate: "0.05", From: "2026-01-01", To: "2026-12-31"}, "legal"))
	must(r.AddClause("ws", Clause{ID: "b", Contract: "k", Component: "rack", Rate: "0.2", From: "2026-10-01"}, "legal"))
	must(r.ApproveClause("ws", "a", "lead"))
	expect(t, ReasonClauseOverlap, second(r.ApproveClause("ws", "b", "lead")), "overlapping clauses")
	must(r.ProposeCredit("ws", october("c1", "a", "i1"), "clerk"))
	expect(t, ReasonCreditOverlap, second(r.ProposeCredit("ws", october("c2", "a", "i2"), "clerk")), "a second credit for the month")
	must(r.WithdrawCredit("ws", "c1", "clerk"))
	expect(t, "ok", second(r.WithdrawCredit("ws", "c1", "clerk")), "repeat withdrawal")
	expect(t, "rejected", second(r.WithdrawCredit("ws", "c1", "boss")), "another withdrawal")
	expect(t, "rejected", second(r.ApproveCredit("ws", "c1", "lead")), "approve a withdrawn credit")
	must(r.ProposeCredit("ws", october("c3", "a", "i1"), "clerk"))
	expect(t, ReasonIncidentOutsidePeriod, second(r.ProposeCredit("ws", Credit{ID: "c4", Clause: "a", Incidents: []string{"i1"}, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"}, "clerk")), "an October incident for November")
	must(r.WithdrawCredit("ws", "c3", "clerk"))
	r2 := slaSetup(t)
	must(r2.AddClause("ws", Clause{ID: "a", Contract: "k", Component: "rack", Rate: "0.05", From: "2026-01-01"}, "legal"))
	must(r2.ApproveClause("ws", "a", "lead"))
	must(r2.RecordIncident("ws", Incident{ID: "i3", Contract: "k", Component: "rack", Start: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), End: time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC), Evidence: digest('c')}, "noc"))
	must(r2.ProposeCredit("ws", october("c1", "a", "i3"), "clerk"))
	expect(t, ReasonIncidentUsed, second(r2.ProposeCredit("ws", Credit{ID: "c2", Clause: "a", Incidents: []string{"i3"}, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"}, "clerk")), "an incident backing a second credit")
	expect(t, ReasonIncidentOutsidePeriod, second(r2.ProposeCredit("ws", Credit{ID: "c5", Clause: "a", Incidents: []string{"i1"}, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"}, "clerk")), "an incident outside the period")
	if v := must(r2.IncidentOf("ws", "i3")); v.Credit != "c1" {
		t.Errorf("incident link: %+v", v)
	}
	must(r2.WithdrawCredit("ws", "c1", "clerk"))
	if v := must(r2.IncidentOf("ws", "i3")); v.Credit != "" {
		t.Error("a withdrawn credit keeps its incident")
	}
	expect(t, "ok", second(r2.ProposeCredit("ws", Credit{ID: "c2", Clause: "a", Incidents: []string{"i3"}, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"}, "clerk")), "the incident is free again")
	must(r2.ProposeCredit("ws", october("c6", "a", "i1"), "clerk"))
	must(r2.ApproveCredit("ws", "c6", "lead"))
	must(r2.ApproveCredit("ws", "c2", "lead"))
	must(r2.LinkCredit("ws", "c6", Link{Document: "cn", Increase: "inc"}, "ar"))
	expect(t, ReasonLinkUsed, second(r2.LinkCredit("ws", "c2", Link{Document: "cn", Increase: "other"}, "ar")), "a document used twice")
	expect(t, ReasonLinkUsed, second(r2.LinkCredit("ws", "c2", Link{Document: "other", Increase: "inc"}, "ar")), "an increase used twice")
}

func TestClauseAndCreditRules(t *testing.T) {
	r := slaSetup(t)
	approved(t, r, usage("u", "gpu", "1"))
	expect(t, ReasonClauseOutside, second(r.AddClause("ws", Clause{ID: "x", Contract: "k", Component: "u", Rate: "0.1", From: "2026-01-01"}, "legal")), "a clause on a usage component")
	expect(t, ReasonClauseOutside, second(r.AddClause("ws", Clause{ID: "x", Contract: "k", Component: "rack", Rate: "0.1", From: "2025-12-31"}, "legal")), "a clause before the component")
	must(r.Define("ws", Component{ID: "short", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-01-01", To: "2026-10-31", Fee: "0"}, "clerk"))
	expect(t, ReasonClauseOutside, second(r.AddClause("ws", Clause{ID: "x", Contract: "k", Component: "short", Rate: "0.1", From: "2026-01-01"}, "legal")), "an open clause on an ending component")
	must(r.AddClause("ws", Clause{ID: "z", Contract: "k", Component: "short", Rate: "1", From: "2026-01-01", To: "2026-10-31"}, "legal"))
	must(r.ApproveClause("ws", "z", "lead"))
	must(r.RecordIncident("ws", Incident{ID: "s1", Contract: "k", Component: "short", Start: time.Date(2026, 10, 3, 1, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 3, 2, 0, 0, 0, time.UTC), Evidence: digest('d')}, "noc"))
	expect(t, "rejected", second(r.ProposeCredit("ws", october("q", "z", "s1"), "clerk")), "a credit on an unapproved component")
	must(r.ApproveComponent("ws", "short", "lead"))
	expect(t, ReasonZeroCredit, second(r.ProposeCredit("ws", october("q", "z", "s1"), "clerk")), "a zero credit")
	for name, rate := range map[string]string{"zero": "0", "above one": "1.01", "negative": "-0.1", "text": "ten"} {
		expect(t, "invalid", second(r.AddClause("ws", Clause{ID: "y", Contract: "k", Component: "rack", Rate: rate, From: "2026-01-01"}, "legal")), name)
	}
	expect(t, "not_found", second(r.AddClause("ws", Clause{ID: "y", Contract: "j", Component: "rack", Rate: "0.1", From: "2026-01-01"}, "legal")), "another contract's component")
	must(r.AddClause("ws", Clause{ID: "c", Contract: "k", Component: "rack", Rate: "0.333", From: "2026-10-01", To: "2026-10-31"}, "legal"))
	must(r.ApproveClause("ws", "c", "lead"))
	if v := must(r.ProposeCredit("ws", october("third", "c", "i1"), "clerk")); v.Amount != "333" {
		t.Errorf("credit amount: %+v", v)
	}
	must(r.WithdrawCredit("ws", "third", "clerk"))
	must(r.AddClause("ws", Clause{ID: "mid", Contract: "k", Component: "rack", Rate: "0.5", From: "2026-11-15"}, "legal"))
	must(r.ApproveClause("ws", "mid", "lead"))
	must(r.RecordIncident("ws", Incident{ID: "n1", Contract: "k", Component: "rack", Start: time.Date(2026, 11, 13, 1, 0, 0, 0, time.UTC), End: time.Date(2026, 11, 13, 2, 0, 0, 0, time.UTC), Evidence: digest('e')}, "noc"))
	expect(t, ReasonIncidentOutsidePeriod, second(r.ProposeCredit("ws", Credit{ID: "nov", Clause: "mid", Incidents: []string{"n1"}, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"}, "clerk")), "an incident before its clause")
	expect(t, "invalid", second(r.ProposeCredit("ws", Credit{ID: "x", Clause: "c", Incidents: []string{"i1"}, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-30"}, "clerk")), "a credit period that is not a month")
	expect(t, "invalid", second(r.ProposeCredit("ws", Credit{ID: "x", Clause: "c", PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"}, "clerk")), "no incident")
	expect(t, "invalid", second(r.ProposeCredit("ws", october("x", "c", "i1", "i1"), "clerk")), "a repeated incident")
	expect(t, "invalid", second(r.ProposeCredit("ws", october("x", "c", "i1", "i2", "s1"), "clerk")), "too many incidents")
	expect(t, "not_found", second(r.ProposeCredit("ws", october("x", "c", "none"), "clerk")), "an unknown incident")
	expect(t, "conflict", second(r.ProposeCredit("ws", october("third", "c", "i2"), "clerk")), "a credit conflict")
	expect(t, "conflict", second(r.ProposeCredit("ws", october("third", "c", "i1"), "other")), "a credit by another actor")
	expect(t, "ok", second(r.ProposeCredit("ws", october("third", "c", "i1"), "clerk")), "a credit repeat")
	expect(t, "rejected", second(r.LinkCredit("ws", "third", Link{Document: "d", Increase: "i"}, "ar")), "link a pending credit")
}

func TestCreditRoomCountsPerContractAndIncidentEnds(t *testing.T) {
	l := testLimits()
	l.Credits = 1
	r, _ := openWith(t, l)
	priced(t, r)
	approved(t, r, rack("a", "100"))
	approved(t, r, Component{ID: "b", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-01-01", Fee: "50"})
	must(r.AddClause("ws", Clause{ID: "sa", Contract: "k", Component: "a", Rate: "0.1", From: "2026-01-01"}, "legal"))
	must(r.AddClause("ws", Clause{ID: "sb", Contract: "k", Component: "b", Rate: "0.1", From: "2026-01-01"}, "legal"))
	must(r.ApproveClause("ws", "sa", "lead"))
	must(r.ApproveClause("ws", "sb", "lead"))
	must(r.RecordIncident("ws", Incident{ID: "ia", Contract: "k", Component: "a", Start: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 2, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	must(r.RecordIncident("ws", Incident{ID: "ib", Contract: "k", Component: "b", Start: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 2, 1, 0, 0, 0, time.UTC), Evidence: digest('b')}, "noc"))
	must(r.ProposeCredit("ws", october("ca", "sa", "ia"), "clerk"))
	expect(t, "limit", second(r.ProposeCredit("ws", october("cb", "sb", "ib"), "clerk")), "credit room per contract")
	must(r.RecordIncident("ws", Incident{ID: "edge", Contract: "k", Component: "a", Start: time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Evidence: digest('c')}, "noc"))
	must(r.WithdrawCredit("ws", "ca", "clerk"))
	l.Credits = 3
	r2, _ := openWith(t, l)
	priced(t, r2)
	approved(t, r2, rack("a", "100"))
	must(r2.AddClause("ws", Clause{ID: "sa", Contract: "k", Component: "a", Rate: "0.1", From: "2026-01-01"}, "legal"))
	must(r2.ApproveClause("ws", "sa", "lead"))
	must(r2.RecordIncident("ws", Incident{ID: "edge", Contract: "k", Component: "a", Start: time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Evidence: digest('c')}, "noc"))
	expect(t, ReasonIncidentOutsidePeriod, second(r2.ProposeCredit("ws", october("c", "sa", "edge"), "clerk")), "an incident that ends at the first instant of the month")
	must(r2.RecordIncident("ws", Incident{ID: "edge2", Contract: "k", Component: "a", Start: time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 1, 0, 0, 0, 1, time.UTC), Evidence: digest('c')}, "noc"))
	expect(t, "ok", second(r2.ProposeCredit("ws", october("c", "sa", "edge2"), "clerk")), "an incident that reaches into the month")
}

func TestUncoveredZeroReadingIsUnpriced(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("a", "100"))
	s := must(r.Statement("ws", november(Reading{Meter: "loose", Quantity: "0"})))
	if len(s.Lines) != 1 || len(s.Unpriced) != 1 || s.Unpriced[0].Excess != "0" || s.Unpriced[0].Included != "0" {
		t.Errorf("an uncovered zero reading: %+v", s)
	}
}
