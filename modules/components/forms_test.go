package components

import (
	"strings"
	"testing"
	"time"
)

func TestValidators(t *testing.T) {
	for _, id := range []string{"Z", "z", "9", "A", "a", "0", ".", "_", ":", "-", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, id := range []string{"", "/a", "a/", "[", "{", "@", "`", " a", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, code := range []string{"AAA", "ZZZ"} {
		if !validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, code := range []string{"@AA", "A@A", "AA@", "[AA", "A[A", "AA[", "AAAA"} {
		if validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, date := range []string{"2000-01-01", "2099-12-31", "2024-02-29", "2026-10-10"} {
		if !validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	for _, date := range []string{"1999-12-31", "2100-01-01", "2026-02-29", "2026-13-01", "2026-1-01", "20261001"} {
		if validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	for _, text := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64), strings.Repeat("9", 64), strings.Repeat("a", 64)} {
		if !validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
	for _, text := range []string{"g" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63), strings.Repeat("0", 63) + ":", strings.Repeat("A", 64), strings.Repeat("0", 63)} {
		if validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
}

func TestRepeatsCompareEveryField(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	must(r.SetPricing("ws", Pricing{Contract: "j", Scale: 2, Mode: "floor"}, "admin"))
	base := Component{ID: "c", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", To: "2026-12-31", Meter: "m", Rate: "1"}
	must(r.Define("ws", base, "clerk"))
	for name, change := range map[string]func(*Component){
		"contract": func(c *Component) { c.Contract = "j" }, "currency": func(c *Component) { c.Currency = "EUR" }, "kind": func(c *Component) { c.Kind, c.Meter, c.Rate, c.Fee = KindRack, "", "", "1" },
		"from": func(c *Component) { c.From = "2026-01-02" }, "to": func(c *Component) { c.To = "2026-12-30" }, "meter": func(c *Component) { c.Meter = "n" }, "rate": func(c *Component) { c.Rate = "2" },
	} {
		c := base
		change(&c)
		expect(t, "conflict", second(r.Define("ws", c, "clerk")), name)
	}
	fixed := rack("f", "1", Inclusion{Meter: "a", Quantity: "1"})
	must(r.Define("ws", fixed, "clerk"))
	fee := fixed
	fee.Fee = "2"
	expect(t, "conflict", second(r.Define("ws", fee, "clerk")), "fee")
	same := Component{ID: "s", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-03-01", To: "2026-03-01", Fee: "1"}
	expect(t, "ok", second(r.Define("ws", same, "clerk")), "a one-day component")
	must(r.AddClause("ws", Clause{ID: "one", Contract: "k", Component: "f", Rate: "0.5", From: "2026-05-05", To: "2026-05-05"}, "legal"))
	approved(t, r, rack("g", "10"))
	must(r.AddClause("ws", Clause{ID: "cl", Contract: "k", Component: "g", Rate: "0.5", From: "2026-01-01"}, "legal"))
	must(r.ApproveClause("ws", "cl", "lead"))
	must(r.RecordIncident("ws", Incident{ID: "i1", Contract: "k", Component: "g", Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	must(r.RecordIncident("ws", Incident{ID: "i2", Contract: "k", Component: "g", Start: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	jan := Credit{ID: "c", Clause: "cl", Incidents: []string{"i1"}, PeriodStart: "2026-01-01", PeriodEnd: "2026-01-31"}
	v := must(r.ProposeCredit("ws", jan, "clerk"))
	if v.Amount != "5" {
		t.Fatal(v)
	}
	for name, c := range map[string]Credit{
		"clause": {ID: "c", Clause: "one", Incidents: []string{"i1"}, PeriodStart: "2026-01-01", PeriodEnd: "2026-01-31"},
		"period": {ID: "c", Clause: "cl", Incidents: []string{"i1"}, PeriodStart: "2026-02-01", PeriodEnd: "2026-02-28"},
		"other":  {ID: "c", Clause: "cl", Incidents: []string{"i2"}, PeriodStart: "2026-01-01", PeriodEnd: "2026-01-31"},
	} {
		expect(t, "conflict", second(r.ProposeCredit("ws", c, "clerk")), name)
	}
	expect(t, "not_found", second(r.RecordIncident("none", Incident{ID: "x", Contract: "k", Component: "g", Start: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc")), "an unknown workspace")
	expect(t, "not_found", second(r.RecordIncident("ws", Incident{ID: "x", Contract: "k", Component: "zz", Start: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc")), "an unknown component")
}

func TestBoundaries(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, Component{ID: "nov", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-11-01", To: "2026-11-30", Fee: "7"})
	approved(t, r, Component{ID: "late", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-12-01", To: "2026-12-31", Fee: "3"})
	approved(t, r, Component{ID: "early", Contract: "k", Currency: "USD", Kind: KindSpace, From: "2026-01-01", To: "2026-10-31", Fee: "2"})
	s := must(r.Statement("ws", november()))
	if s.Charges != "7" || len(s.Lines) != 1 || s.Lines[0].Kind != LineFixed || s.Lines[0].Component != "nov" {
		t.Errorf("exact month coverage: %+v", s)
	}
	approved(t, r, Component{ID: "short", Contract: "k", Currency: "USD", Kind: KindService, From: "2026-11-01", To: "2026-11-29", Fee: "1"})
	expect(t, ReasonPartialPeriod, second(r.Statement("ws", november())), "a component ending a day early")
	must(r.Define("ws", Component{ID: "touch", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-11-30", Meter: "w", Rate: "1"}, "clerk"))
	must(r.Define("ws", Component{ID: "touched", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", To: "2026-11-30", Meter: "w", Rate: "1"}, "clerk"))
	must(r.ApproveComponent("ws", "touch", "lead"))
	expect(t, ReasonMeterOverlap, second(r.ApproveComponent("ws", "touched", "lead")), "components sharing one day")
	r2, _ := openWith(t, testLimits())
	priced(t, r2)
	approved(t, r2, rack("a", "10", Inclusion{Meter: "p", Quantity: All}, Inclusion{Meter: "q", Quantity: "5"}))
	full := []Reading{{Meter: "p", Quantity: "9"}, {Meter: "q", Quantity: "5"}, {Meter: "r", Quantity: "1"}, {Meter: "s", Quantity: "1"}, {Meter: "t", Quantity: "1"}, {Meter: "u", Quantity: "2"}}
	s = must(r2.Statement("ws", november(full...)))
	kinds := map[string]Line{}
	for _, l := range s.Lines {
		kinds[l.Kind+"/"+l.Meter] = l
	}
	if l := kinds[LineFixed+"/"]; l.Component != "a" || l.Amount != "10" || len(l.Includes) != 2 {
		t.Errorf("fixed line: %+v", kinds)
	}
	if l := kinds[LineIncluded+"/p"]; l.Included != All || l.Billable != "0" || l.Amount != "0" || l.Measured != "9" {
		t.Errorf("all included: %+v", l)
	}
	if l := kinds[LineIncluded+"/q"]; l.Included != "5" || l.Billable != "0" || l.Amount != "0" {
		t.Errorf("exactly included: %+v", l)
	}
	if len(s.Unpriced) != 4 || len(s.Lines) != 3 {
		t.Errorf("readings at the room: %+v", s)
	}
	s = must(r2.Statement("ws", november(Reading{Meter: "q", Quantity: "7"})))
	if len(s.Lines) != 2 || s.Lines[1].Kind != LineIncluded || s.Lines[1].Billable != "0" || s.Lines[1].Amount != "0" || s.Lines[1].Measured != "7" || s.Unpriced[0].Excess != "2" {
		t.Errorf("partly included: %+v", s)
	}
	approved(t, r2, Component{ID: "b", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-01-01", Fee: "1"})
	approved(t, r2, Component{ID: "0b", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-01-01", Fee: "1"})
	if s := must(r2.Statement("ws", november())); s.Lines[0].Component != "0b" || s.Lines[1].Component != "a" || s.Lines[2].Component != "b" {
		t.Errorf("order: %+v", s.Lines)
	}
	r3, _ := openWith(t, testLimits())
	priced(t, r3)
	approved(t, r3, Component{ID: "a", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-10-01", Fee: "10"})
	must(r3.AddClause("ws", Clause{ID: "cl", Contract: "k", Component: "a", Rate: "0.5", From: "2026-10-01", To: "2026-10-31"}, "legal"))
	must(r3.ApproveClause("ws", "cl", "lead"))
	must(r3.RecordIncident("ws", Incident{ID: "i", Contract: "k", Component: "a", Start: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	expect(t, "ok", second(r3.ProposeCredit("ws", october("c", "cl", "i"), "clerk")), "a credit at the component, clause and incident edges")
	must(r3.ApproveCredit("ws", "c", "ops"))
	expect(t, "ok", second(r3.ApproveCredit("ws", "c", "ops")), "repeat")
	for name, call := range map[string]func() error{
		"approve ws":   func() error { return second(r3.ApproveCredit("", "c", "ops")) },
		"approve id":   func() error { return second(r3.ApproveCredit("ws", "", "ops")) },
		"withdraw id":  func() error { return second(r3.WithdrawCredit("ws", "", "ops")) },
		"withdraw act": func() error { return second(r3.WithdrawCredit("ws", "c", "")) },
		"link ws":      func() error { return second(r3.LinkCredit("", "c", Link{Document: "d", Increase: "i"}, "a")) },
		"link id":      func() error { return second(r3.LinkCredit("ws", "", Link{Document: "d", Increase: "i"}, "a")) },
		"link actor":   func() error { return second(r3.LinkCredit("ws", "c", Link{Document: "d", Increase: "i"}, "")) },
		"link doc":     func() error { return second(r3.LinkCredit("ws", "c", Link{Increase: "i"}, "a")) },
		"approveC ws":  func() error { return second(r3.ApproveComponent("", "a", "l")) },
		"approveC id":  func() error { return second(r3.ApproveComponent("ws", "", "l")) },
		"clause ws": func() error {
			return second(r3.AddClause("", Clause{ID: "x", Contract: "k", Component: "a", Rate: "0.1", From: "2026-10-01"}, "l"))
		},
		"clause actor": func() error {
			return second(r3.AddClause("ws", Clause{ID: "x", Contract: "k", Component: "a", Rate: "0.1", From: "2026-10-01"}, ""))
		},
		"clause to": func() error {
			return second(r3.AddClause("ws", Clause{ID: "x", Contract: "k", Component: "a", Rate: "0.1", From: "2026-10-01", To: "2026-09-30"}, "l"))
		},
		"clause date": func() error {
			return second(r3.AddClause("ws", Clause{ID: "x", Contract: "k", Component: "a", Rate: "0.1", From: "2026-10-01", To: "2026-10-32"}, "l"))
		},
		"propose ws":  func() error { return second(r3.ProposeCredit("", october("x", "cl", "i"), "c")) },
		"propose act": func() error { return second(r3.ProposeCredit("ws", october("x", "cl", "i"), "")) },
		"propose id":  func() error { return second(r3.ProposeCredit("ws", october("", "cl", "i"), "c")) },
		"propose cl":  func() error { return second(r3.ProposeCredit("ws", october("x", "", "i"), "c")) },
		"propose inc": func() error { return second(r3.ProposeCredit("ws", october("x", "cl", ""), "c")) },
		"define ws":   func() error { return second(r3.Define("", rack("x", "1"), "c")) },
		"define to": func() error {
			return second(r3.Define("ws", Component{ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-10-02", To: "2026-10-01", Fee: "1"}, "c"))
		},
		"pricing ws":  func() error { return second(r3.SetPricing("", Pricing{Contract: "z", Scale: 2, Mode: "floor"}, "a")) },
		"incident ws": func() error { return second(r3.RecordIncident("", Incident{}, "n")) },
		"incident act": func() error {
			return second(r3.RecordIncident("ws", Incident{ID: "z", Contract: "k", Component: "a", Start: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, ""))
		},
		"component":    func() error { return second(r3.ComponentOf("ws", "")) },
		"approveCl ws": func() error { return second(r3.ApproveClause("", "cl", "l")) },
		"approveCl a":  func() error { return second(r3.ApproveClause("ws", "cl", "")) },
		"credit ws":    func() error { return second(r3.CreditOf("", "c")) },
	} {
		expect(t, "invalid", call(), name)
	}
	expect(t, "ok", second(r3.AddClause("ws", Clause{ID: "y", Contract: "k", Component: "a", Rate: "0.1", From: "2026-10-01", To: "2026-10-01"}, "l")), "a one-day clause")
	expect(t, "ok", second(r3.Define("ws", Component{ID: "y", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-10-01", To: "2026-10-01", Fee: "1"}, "c")), "a one-day component")
	if v := must(r3.ComponentOf("ws", "a")); v.State != StateApproved {
		t.Error(v.State)
	}
	if v := must(r3.CreditOf("ws", "c")); v.State != StateApproved || v.Approver != "ops" {
		t.Error(v)
	}
}
