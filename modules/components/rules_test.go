package components

import (
	"testing"
	"time"
)

func TestIncludedPowerIsNeverChargedAgain(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("rack", "1200", Inclusion{Meter: "power", Quantity: All}))
	approved(t, r, usage("power-use", "power", "0.25"))
	approved(t, r, Component{ID: "gpu", Contract: "k", Currency: "USD", Kind: KindGPUReserved, From: "2026-01-01", Fee: "3000"})
	approved(t, r, Component{ID: "ops", Contract: "k", Currency: "USD", Kind: KindService, From: "2026-01-01", Fee: "250.5", Includes: []Inclusion{{Meter: "hands", Quantity: "10"}}})
	approved(t, r, usage("gpu-hours", "gpu_h", "2.105"))
	s := must(r.Statement("ws", november(Reading{Meter: "power", Quantity: "8000"}, Reading{Meter: "gpu_h", Quantity: "10"}, Reading{Meter: "hands", Quantity: "12.5"})))
	if s.Charges != "4471.55" || s.Net != s.Charges || s.Currency != "USD" || len(s.Lines) != 6 {
		t.Fatalf("statement: %+v", s)
	}
	byComponent := map[string]Line{}
	for _, l := range s.Lines {
		byComponent[l.Component+"/"+l.Meter] = l
	}
	if l := byComponent["power-use/power"]; l.Included != All || l.Billable != "0" || l.Amount != "0" || l.Measured != "8000" {
		t.Errorf("included power is charged: %+v", l)
	}
	if l := byComponent["gpu-hours/gpu_h"]; l.Amount != "21.05" || l.Billable != "10" || l.Included != "0" {
		t.Errorf("usage add-on: %+v", l)
	}
	if l := byComponent["/hands"]; l.Kind != LineIncluded || l.Included != "10" || l.Amount != "0" || l.Measured != "12.5" {
		t.Errorf("an included reading: %+v", l)
	}
	if len(s.Unpriced) != 1 || s.Unpriced[0].Excess != "2.5" || s.Unpriced[0].Meter != "hands" {
		t.Errorf("excess beyond an inclusion: %+v", s.Unpriced)
	}
}

func TestPartialInclusionChargesOnlyTheExcess(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("rack", "100", Inclusion{Meter: "power", Quantity: "500.5"}))
	approved(t, r, usage("power-use", "power", "0.3"))
	for measured, want := range map[string]string{"500": "0", "500.5": "0", "501": "0.15", "600.5": "30", "0": "0", "500.51": "0"} {
		s := must(r.Statement("ws", november(Reading{Meter: "power", Quantity: measured})))
		var l Line
		for _, line := range s.Lines {
			if line.Kind == LineUsage {
				l = line
			}
		}
		if l.Amount != want || l.Included != "500.5" {
			t.Errorf("%s: %+v", measured, l)
		}
	}
	s := must(r.Statement("ws", november(Reading{Meter: "power", Quantity: "500.53"})))
	if s.Charges != "100.01" {
		t.Errorf("half-even rounding: %s", s.Charges)
	}
	s = must(r.Statement("ws", november(Reading{Meter: "power", Quantity: "500.55"}, Reading{Meter: "loose", Quantity: "3"})))
	if s.Charges != "100.02" || len(s.Unpriced) != 1 || s.Unpriced[0].Excess != "3" || s.Unpriced[0].Included != "0" {
		t.Errorf("an uncovered meter: %+v", s)
	}
}

func TestOneMeterOneChargeOneInclusion(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("rack", "100", Inclusion{Meter: "power", Quantity: All}))
	must(r.Define("ws", Component{ID: "pdu", Contract: "k", Currency: "USD", Kind: KindService, From: "2026-06-01", To: "2026-12-31", Fee: "40", Includes: []Inclusion{{Meter: "power", Quantity: "100"}}}, "clerk"))
	expect(t, ReasonInclusionOverlap, second(r.ApproveComponent("ws", "pdu", "lead")), "a second fixed charge including power")
	approved(t, r, usage("u1", "power", "1"))
	must(r.Define("ws", Component{ID: "u2", Contract: "k", Currency: "EUR", Kind: KindUsage, From: "2026-12-31", Meter: "power", Rate: "2"}, "clerk"))
	expect(t, ReasonMeterOverlap, second(r.ApproveComponent("ws", "u2", "lead")), "a second usage charge on the meter, whatever the currency")
	must(r.Define("ws", Component{ID: "a", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2020-01-01", To: "2020-12-31", Meter: "water", Rate: "1"}, "clerk"))
	must(r.Define("ws", Component{ID: "b", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2020-12-31", Meter: "water", Rate: "1"}, "clerk"))
	must(r.ApproveComponent("ws", "a", "lead"))
	expect(t, ReasonMeterOverlap, second(r.ApproveComponent("ws", "b", "lead")), "two pending components approved one after the other")
	must(r.Define("ws", Component{ID: "c", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2021-01-01", To: "2025-12-31", Meter: "water", Rate: "1"}, "clerk"))
	expect(t, "ok", second(r.ApproveComponent("ws", "c", "lead")), "an adjacent usage component")
	must(r.SetPricing("ws", Pricing{Contract: "j", Scale: 0, Mode: "floor"}, "admin"))
	must(r.Define("ws", Component{ID: "other", Contract: "j", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Meter: "power", Rate: "1"}, "clerk"))
	expect(t, ReasonNoPricing, second(r.Define("ws", Component{ID: "other2", Contract: "j2", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Meter: "power", Rate: "1"}, "clerk")), "no pricing")
}

func TestStatementRefusals(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("rack", "100"))
	approved(t, r, usage("u", "power", "1"))
	expect(t, ReasonMissingReading, second(r.Statement("ws", november())), "a usage component with no reading")
	for name, q := range map[string]Request{
		"not a month": {Contract: "k", PeriodStart: "2026-11-01", PeriodEnd: "2026-11-29"},
		"mid month":   {Contract: "k", PeriodStart: "2026-11-02", PeriodEnd: "2026-12-01"},
		"two months":  {Contract: "k", PeriodStart: "2026-11-01", PeriodEnd: "2026-12-31"},
		"duplicate":   november(Reading{Meter: "power", Quantity: "1"}, Reading{Meter: "power", Quantity: "1"}),
		"negative":    november(Reading{Meter: "power", Quantity: "-1"}),
		"meter":       november(Reading{Meter: "a b", Quantity: "1"}),
		"contract":    {Contract: "", PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"},
		"too many":    november(Reading{Meter: "a", Quantity: "1"}, Reading{Meter: "b", Quantity: "1"}, Reading{Meter: "c", Quantity: "1"}, Reading{Meter: "d", Quantity: "1"}, Reading{Meter: "e", Quantity: "1"}, Reading{Meter: "f", Quantity: "1"}, Reading{Meter: "g", Quantity: "1"}),
	} {
		expect(t, "invalid", second(r.Statement("ws", q)), name)
	}
	if s := must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2028-02-01", PeriodEnd: "2028-02-29", Usage: []Reading{{Meter: "power", Quantity: "2"}}})); s.Charges != "102" {
		t.Errorf("a leap February: %+v", s)
	}
	must(r.Define("ws", Component{ID: "late", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-11-15", Fee: "10"}, "clerk"))
	if s := must(r.Statement("ws", november(Reading{Meter: "power", Quantity: "0"}))); s.Charges != "100" {
		t.Errorf("a pending component takes part: %+v", s)
	}
	must(r.ApproveComponent("ws", "late", "lead"))
	expect(t, ReasonPartialPeriod, second(r.Statement("ws", november(Reading{Meter: "power", Quantity: "0"}))), "a component starting mid month")
	if s := must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2026-12-01", PeriodEnd: "2026-12-31", Usage: []Reading{{Meter: "power", Quantity: "0"}}})); s.Charges != "110" {
		t.Errorf("december: %+v", s)
	}
	must(r.Define("ws", Component{ID: "euro", Contract: "k", Currency: "EUR", Kind: KindSpace, From: "2027-01-01", Fee: "5"}, "clerk"))
	must(r.ApproveComponent("ws", "euro", "lead"))
	expect(t, ReasonCurrencyMismatch, second(r.Statement("ws", Request{Contract: "k", PeriodStart: "2027-01-01", PeriodEnd: "2027-01-31", Usage: []Reading{{Meter: "power", Quantity: "0"}}})), "two currencies")
	expect(t, ReasonNoPricing, second(r.Statement("ws", Request{Contract: "zz", PeriodStart: "2027-01-01", PeriodEnd: "2027-01-31"})), "an unknown contract")
	expect(t, ReasonNoPricing, second(r.Statement("none", november())), "an unknown workspace")
	empty := must(r.Statement("ws", Request{Contract: "k", PeriodStart: "2025-01-01", PeriodEnd: "2025-01-31", Usage: []Reading{{Meter: "x", Quantity: "1"}}}))
	if empty.Charges != "0" || empty.Currency != "" || len(empty.Lines) != 0 || len(empty.Unpriced) != 1 || !empty.Evaluated.Equal(start) {
		t.Errorf("no component: %+v", empty)
	}
	_ = time.Second
}
