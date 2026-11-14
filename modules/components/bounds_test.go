package components

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestComponentForms(t *testing.T) {
	r, _ := openWith(t, testLimits())
	priced(t, r)
	for name, c := range map[string]Component{
		"kind":          {ID: "x", Contract: "k", Currency: "USD", Kind: "power", From: "2026-01-01", Fee: "1"},
		"id":            {ID: "", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1"},
		"contract":      {ID: "x", Contract: "a b", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1"},
		"currency":      {ID: "x", Contract: "k", Currency: "usd", Kind: KindRack, From: "2026-01-01", Fee: "1"},
		"from":          {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-02-30", Fee: "1"},
		"to":            {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", To: "2025-12-31", Fee: "1"},
		"to date":       {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", To: "2026-13-01", Fee: "1"},
		"no fee":        {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01"},
		"negative fee":  {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "-1"},
		"fixed meter":   {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1", Meter: "m"},
		"fixed rate":    {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1", Rate: "1"},
		"fee scale":     {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1.001"},
		"inclusion":     {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1", Includes: []Inclusion{{Meter: "m", Quantity: "0"}}},
		"inclusion id":  {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1", Includes: []Inclusion{{Meter: "", Quantity: "1"}}},
		"repeat meter":  {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1", Includes: []Inclusion{{Meter: "m", Quantity: "1"}, {Meter: "m", Quantity: All}}},
		"repeat all":    {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1", Includes: []Inclusion{{Meter: "m", Quantity: All}, {Meter: "m", Quantity: "1"}}},
		"usage fee":     {ID: "x", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Meter: "m", Rate: "1", Fee: "1"},
		"usage no rate": {ID: "x", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Meter: "m"},
		"usage meter":   {ID: "x", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Rate: "1"},
		"usage incl":    {ID: "x", Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Meter: "m", Rate: "1", Includes: []Inclusion{{Meter: "m", Quantity: "1"}}},
		"long":          {ID: "x", Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: "1." + strings.Repeat("0", 70)},
	} {
		expect(t, "invalid", second(r.Define("ws", c, "clerk")), name)
	}
	c := rack("r", "1.50", Inclusion{Meter: "m", Quantity: "2.50"}, Inclusion{Meter: "n", Quantity: All})
	v := must(r.Define("ws", c, "clerk"))
	if v.Component.Fee != "1.5" || v.Component.Includes[0].Quantity != "2.5" || c.Includes[0].Quantity != "2.50" || v.State != StatePending {
		t.Errorf("canonical forms: %+v", v)
	}
	expect(t, "ok", second(r.Define("ws", c, "clerk")), "repeat")
	expect(t, "conflict", second(r.Define("ws", c, "other")), "another actor")
	expect(t, "conflict", second(r.Define("ws", rack("r", "2"), "clerk")), "other content")
	expect(t, "conflict", second(r.Define("ws", rack("r", "1.5", Inclusion{Meter: "m", Quantity: "2.5"}, Inclusion{Meter: "o", Quantity: All}), "clerk")), "other inclusions")
	expect(t, "conflict", second(r.Define("ws", rack("r", "1.5", Inclusion{Meter: "m", Quantity: "2.5"}), "clerk")), "fewer inclusions")
	v.Component.Includes[0].Meter = "mutated"
	if must(r.ComponentOf("ws", "r")).Component.Includes[0].Meter != "m" {
		t.Error("a view shares the inclusions")
	}
	expect(t, "rejected", second(r.ApproveComponent("ws", "r", "clerk")), "self approval")
	must(r.ApproveComponent("ws", "r", "lead"))
	expect(t, "ok", second(r.ApproveComponent("ws", "r", "lead")), "repeat approval")
	expect(t, "rejected", second(r.ApproveComponent("ws", "r", "boss")), "another approval")
	expect(t, "not_found", second(r.ApproveComponent("ws", "zz", "lead")), "unknown component")
	expect(t, "not_found", second(r.ComponentOf("ws", "zz")), "unknown component read")
	expect(t, "invalid", second(r.ComponentOf("", "r")), "malformed read")
	expect(t, "limit", second(r.Define("ws", rack("big", "1", Inclusion{Meter: "a", Quantity: "1"}, Inclusion{Meter: "b", Quantity: "1"}, Inclusion{Meter: "c", Quantity: "1"}, Inclusion{Meter: "d", Quantity: "1"}), "clerk")), "inclusion room")
	expect(t, "ok", second(r.Define("ws", rack("three", "1", Inclusion{Meter: "a", Quantity: "1"}, Inclusion{Meter: "b", Quantity: "1"}, Inclusion{Meter: "c", Quantity: "1"}), "clerk")), "inclusion room edge")
	expect(t, "conflict", second(r.SetPricing("ws", Pricing{Contract: "k", Scale: 3, Mode: "half_even"}, "admin")), "repricing")
	expect(t, "conflict", second(r.SetPricing("ws", Pricing{Contract: "k", Scale: 2, Mode: "half_even"}, "other")), "pricing by another")
	expect(t, "ok", second(r.SetPricing("ws", Pricing{Contract: "k", Scale: 2, Mode: "half_even"}, "admin")), "pricing repeat")
	for name, p := range map[string]Pricing{"scale": {Contract: "x", Scale: 9, Mode: "floor"}, "negative": {Contract: "x", Scale: -1, Mode: "floor"}, "mode": {Contract: "x", Scale: 2, Mode: "bankers"}, "contract": {Scale: 2, Mode: "floor"}} {
		expect(t, "invalid", second(r.SetPricing("ws", p, "admin")), name)
	}
	if v := must(r.SetPricing("ws", Pricing{Contract: "x", Scale: 8, Mode: "ceiling"}, "admin")); len(v.Digest) != 64 {
		t.Error(v.Digest)
	}
	expect(t, "ok", second(r.SetPricing("ws", Pricing{Contract: "y", Scale: 0, Mode: "floor"}, "admin")), "scale zero")
}

func TestRegistryLimits(t *testing.T) {
	if _, err := NewRegistry(testProfile(t), testLimits(), nil); kind(err) != "invalid" {
		t.Error("no clock")
	}
	caps := Limits{Workspaces: 100000, Components: 1000, Inclusions: 100, Readings: 1000, Incidents: 100000, IncidentsPerCredit: 100, Clauses: 1000, Credits: 10000}
	if _, err := NewRegistry(testProfile(t), caps, time.Now); err != nil {
		t.Error(err)
	}
	for name, change := range map[string]func(*Limits){
		"w": func(l *Limits) { l.Workspaces++ }, "c": func(l *Limits) { l.Components++ }, "i": func(l *Limits) { l.Inclusions++ }, "r": func(l *Limits) { l.Readings++ },
		"n": func(l *Limits) { l.Incidents++ }, "p": func(l *Limits) { l.IncidentsPerCredit++ }, "cl": func(l *Limits) { l.Clauses++ }, "cr": func(l *Limits) { l.Credits++ }, "zero": func(l *Limits) { l.Credits = 0 },
	} {
		l := caps
		change(&l)
		if _, err := NewRegistry(testProfile(t), l, time.Now); kind(err) != "invalid" {
			t.Errorf("%s accepted", name)
		}
	}
	l := testLimits()
	l.Workspaces, l.Components, l.Incidents, l.Clauses, l.Credits = 1, 2, 2, 2, 1
	r, _ := openWith(t, l)
	priced(t, r)
	expect(t, "limit", second(r.SetPricing("w2", Pricing{Contract: "k", Scale: 2, Mode: "floor"}, "admin")), "workspaces")
	approved(t, r, rack("a", "10"))
	approved(t, r, Component{ID: "b", Contract: "k", Currency: "USD", Kind: KindLine, From: "2026-01-01", Fee: "5"})
	expect(t, "limit", second(r.Define("ws", rack("c", "1"), "clerk")), "components")
	for i, id := range []string{"x1", "x2"} {
		must(r.RecordIncident("ws", Incident{ID: id, Contract: "k", Component: "a", Start: time.Date(2026, 10, 2+i, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 2+i, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	}
	expect(t, "limit", second(r.RecordIncident("ws", Incident{ID: "x3", Contract: "k", Component: "a", Start: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc")), "incidents")
	must(r.AddClause("ws", Clause{ID: "s1", Contract: "k", Component: "a", Rate: "0.5", From: "2026-01-01"}, "legal"))
	must(r.AddClause("ws", Clause{ID: "s2", Contract: "k", Component: "b", Rate: "0.5", From: "2026-01-01"}, "legal"))
	expect(t, "limit", second(r.AddClause("ws", Clause{ID: "s3", Contract: "k", Component: "b", Rate: "0.5", From: "2026-01-01"}, "legal")), "clauses")
	must(r.ApproveClause("ws", "s1", "lead"))
	must(r.ProposeCredit("ws", october("c1", "s1", "x1"), "clerk"))
	must(r.WithdrawCredit("ws", "c1", "clerk"))
	expect(t, "limit", second(r.ProposeCredit("ws", october("c2", "s1", "x1"), "clerk")), "credits with a withdrawn one")
}

func TestIncidentsAndMalformedCalls(t *testing.T) {
	r, clock := openWith(t, testLimits())
	priced(t, r)
	approved(t, r, rack("a", "10"))
	good := Incident{ID: "i", Contract: "k", Component: "a", Start: time.Date(2026, 10, 2, 0, 0, 0, 0, time.FixedZone("x", 3600)), End: time.Date(2026, 10, 2, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}
	v := must(r.RecordIncident("ws", good, "noc"))
	if v.Incident.Start.Location() != time.UTC || len(v.Digest) != 64 {
		t.Errorf("incident: %+v", v)
	}
	expect(t, "ok", second(r.RecordIncident("ws", good, "noc")), "repeat")
	expect(t, "conflict", second(r.RecordIncident("ws", good, "other")), "another actor")
	other := good
	other.Evidence = digest('b')
	expect(t, "conflict", second(r.RecordIncident("ws", other, "noc")), "other content")
	future := good
	future.ID, future.End = "f", start.Add(time.Second)
	expect(t, "rejected", second(r.RecordIncident("ws", future, "noc")), "an incident that has not ended")
	clock.set(start.Add(time.Second))
	expect(t, "ok", second(r.RecordIncident("ws", future, "noc")), "an incident that just ended")
	for name, i := range map[string]Incident{
		"end":      {ID: "z", Contract: "k", Component: "a", Start: good.End, End: good.End, Evidence: digest('a')},
		"zero":     {ID: "z", Contract: "k", Component: "a", End: good.End, Evidence: digest('a')},
		"evidence": {ID: "z", Contract: "k", Component: "a", Start: good.Start, End: good.End, Evidence: "x"},
		"old":      {ID: "z", Contract: "k", Component: "a", Start: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), End: good.End, Evidence: digest('a')},
		"id":       {ID: "", Contract: "k", Component: "a", Start: good.Start, End: good.End, Evidence: digest('a')},
	} {
		expect(t, "invalid", second(r.RecordIncident("ws", i, "noc")), name)
	}
	wrong := good
	wrong.ID, wrong.Contract = "w", "j"
	expect(t, "not_found", second(r.RecordIncident("ws", wrong, "noc")), "another contract")
	expect(t, "not_found", second(r.IncidentOf("ws", "zz")), "unknown incident")
	for name, call := range map[string]func() error{
		"incident": func() error { return second(r.IncidentOf("", "i")) },
		"clause": func() error {
			return second(r.AddClause("ws", Clause{ID: "c", Contract: "k", Component: "a", Rate: "0.1", From: "2026-13-01"}, "l"))
		},
		"approveCl": func() error { return second(r.ApproveClause("ws", "", "l")) },
		"propose":   func() error { return second(r.ProposeCredit("ws", Credit{}, "c")) },
		"approveCr": func() error { return second(r.ApproveCredit("ws", "c", "")) },
		"withdraw":  func() error { return second(r.WithdrawCredit("", "c", "x")) },
		"link":      func() error { return second(r.LinkCredit("ws", "c", Link{Document: "d"}, "x")) },
		"credit":    func() error { return second(r.CreditOf("ws", "")) },
		"statement": func() error { return second(r.Statement("", november())) },
		"define":    func() error { return second(r.Define("ws", rack("q", "1"), "")) },
		"approve":   func() error { return second(r.ApproveComponent("ws", "a", "")) },
		"pricing":   func() error { return second(r.SetPricing("ws", Pricing{Contract: "k", Scale: 2, Mode: "floor"}, "")) },
		"record":    func() error { return second(r.RecordIncident("ws", good, "")) },
	} {
		expect(t, "invalid", call(), name)
	}
	for name, call := range map[string]func() error{
		"approveCl": func() error { return second(r.ApproveClause("ws", "zz", "l")) },
		"approveCr": func() error { return second(r.ApproveCredit("ws", "zz", "l")) },
		"withdraw":  func() error { return second(r.WithdrawCredit("ws", "zz", "x")) },
		"link":      func() error { return second(r.LinkCredit("ws", "zz", Link{Document: "d", Increase: "i"}, "x")) },
		"credit":    func() error { return second(r.CreditOf("ws", "zz")) },
		"clause": func() error {
			return second(r.AddClause("none", Clause{ID: "c", Contract: "k", Component: "a", Rate: "0.1", From: "2026-01-01"}, "l"))
		},
		"propose": func() error { return second(r.ProposeCredit("none", october("c", "s", "i"), "c")) },
	} {
		expect(t, "not_found", call(), name)
	}
	must(r.AddClause("ws", Clause{ID: "mid", Contract: "k", Component: "a", Rate: "0.5", From: "2026-11-01"}, "legal"))
	expect(t, "ok", second(r.AddClause("ws", Clause{ID: "mid", Contract: "k", Component: "a", Rate: "0.50", From: "2026-11-01"}, "legal")), "clause repeat")
	expect(t, "conflict", second(r.AddClause("ws", Clause{ID: "mid", Contract: "k", Component: "a", Rate: "0.4", From: "2026-11-01"}, "legal")), "clause conflict")
	expect(t, "conflict", second(r.AddClause("ws", Clause{ID: "mid", Contract: "k", Component: "a", Rate: "0.5", From: "2026-11-01"}, "x")), "clause by another")
	must(r.ApproveClause("ws", "mid", "lead"))
	expect(t, "ok", second(r.ApproveClause("ws", "mid", "lead")), "clause approval repeat")
	expect(t, "rejected", second(r.ApproveClause("ws", "mid", "boss")), "another clause approver")
	clock.set(time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC))
	must(r.AddClause("ws", Clause{ID: "late", Contract: "k", Component: "a", Rate: "0.5", From: "2026-03-15", To: "2026-03-31"}, "legal"))
	must(r.RecordIncident("ws", Incident{ID: "n", Contract: "k", Component: "a", Start: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 3, 20, 1, 0, 0, 0, time.UTC), Evidence: digest('a')}, "noc"))
	expect(t, "ok", second(r.ApproveClause("ws", "late", "lead")), "a clause that does not overlap")
	mid := must(r.AddClause("ws", Clause{ID: "mid2", Contract: "k", Component: "a", Rate: "0.5", From: "2026-11-15", To: "2026-11-30"}, "legal"))
	if mid.State != StatePending {
		t.Fatal(mid.State)
	}
	expect(t, ReasonClauseOverlap, second(r.ApproveClause("ws", "mid2", "lead")), "a clause inside another")
	expect(t, "rejected", second(r.ProposeCredit("ws", Credit{ID: "march", Clause: "late", Incidents: []string{"n"}, PeriodStart: "2026-03-01", PeriodEnd: "2026-03-31"}, "clerk")), "a month that starts before its clause")
}

func TestPinnedDigest(t *testing.T) {
	r, _ := openWith(t, testLimits())
	v := must(r.SetPricing("ws", Pricing{Contract: "k", Scale: 2, Mode: "half_even"}, "admin"))
	sum := sha256.New()
	for _, field := range []string{pricingLabel, "ws", "k", "2", "half_even", "admin"} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		sum.Write(size[:])
		sum.Write([]byte(field))
	}
	if want := hex.EncodeToString(sum.Sum(nil)); v.Digest != want {
		t.Errorf("%s, want %s", v.Digest, want)
	}
}

func TestConcurrentApprovalsKeepOneChargePerMeter(t *testing.T) {
	l := testLimits()
	l.Components = 40
	r, _ := openWith(t, l)
	priced(t, r)
	for i := 0; i < 30; i++ {
		must(r.Define("ws", usage("u"+itoa2(i), "power", "1"), "clerk"))
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	approvedCount := 0
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := r.ApproveComponent("ws", "u"+itoa2(i), "lead"); err == nil {
				mu.Lock()
				approvedCount++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if approvedCount != 1 {
		t.Errorf("approved %d", approvedCount)
	}
}

func itoa2(i int) string { return string(rune('a'+i/26)) + string(rune('a'+i%26)) }

func TestSourceRules(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "flo"+"at") || strings.Contains(text, "/"+"/ ") || strings.Contains(text, "/"+"*") {
			t.Errorf("%s uses a binary fraction type or comments", e.Name())
		}
		if strings.Count(text, "\n") >= 1500 {
			t.Errorf("%s is too long", e.Name())
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "hybrid_bil"+"ling/") && !strings.Contains(line, "packages/primitives/") {
				t.Errorf("%s imports %s", e.Name(), strings.TrimSpace(line))
			}
		}
	}
}

func TestSmallHelpers(t *testing.T) {
	err := refuse(ReasonZeroCredit)
	if !strings.HasSuffix(err.Error(), ": zero_credit") || !strings.HasPrefix(err.Error(), ErrRejected.Error()) || kind(err) != ReasonZeroCredit {
		t.Error(err)
	}
	if month("2026-02-30", "2026-03-31") || month("2026-11-01", "2026-11-31") || !month("2026-12-01", "2026-12-31") || month("2026-12-01", "2027-01-01") {
		t.Error("months")
	}
	for _, text := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64)} {
		if !validDigest(text) {
			t.Error(text)
		}
	}
	for _, text := range []string{"g" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), strings.Repeat("0", 63)} {
		if validDigest(text) {
			t.Error(text)
		}
	}
	r := slaSetup(t)
	must(r.AddClause("ws", Clause{ID: "s", Contract: "k", Component: "rack", Rate: "0.25", From: "2026-01-01"}, "legal"))
	must(r.ApproveClause("ws", "s", "lead"))
	must(r.ProposeCredit("ws", october("c", "s", "i1", "i2"), "clerk"))
	v := must(r.CreditOf("ws", "c"))
	if v.Amount != "250" || len(v.Credit.Incidents) != 2 {
		t.Fatal(v)
	}
	v.Credit.Incidents[0] = "mutated"
	if must(r.CreditOf("ws", "c")).Credit.Incidents[0] != "i1" {
		t.Error("a view shares the incidents")
	}
	expect(t, "conflict", second(r.ProposeCredit("ws", october("c", "s", "i2", "i1"), "clerk")), "incidents in another order")
	expect(t, "conflict", second(r.ProposeCredit("ws", october("c", "s", "i1"), "clerk")), "fewer incidents")
}
