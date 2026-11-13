package reconcile

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func second[T any](_ T, err error) error { return err }

func TestConstruction(t *testing.T) {
	clock := &testClock{at: start}
	for index, change := range []func(*Limits){func(l *Limits) { l.Runs = 0 }, func(l *Limits) { l.Parts = 0 }, func(l *Limits) { l.Lines = 0 }, func(l *Limits) { l.Page = 0 }, func(l *Limits) { l.Listed = 0 }} {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(limits, testRounding, testProfile(t), clock.read); err != ErrInvalid {
			t.Fatalf("limit %d: %v", index, err)
		}
	}
	if _, err := NewRegistry(testLimits(), testRounding, decimal.Profile{}, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testLimits(), nil, decimal.Profile{}, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testLimits(), testRounding, testProfile(t), nil); err != ErrInvalid {
		t.Fatal(err)
	}
	for name, rounding := range map[string]map[string]string{"currency": {"usd": "1"}, "amount": {"USD": "one"}, "negative": {"USD": "-0.01"}, "beyond the input bound": {"USD": "0.00001"}} {
		if _, err := NewRegistry(testLimits(), rounding, testProfile(t), clock.read); err != ErrInvalid {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, rounding := range []map[string]string{nil, {}, {"USD": "0"}} {
		if _, err := NewRegistry(testLimits(), rounding, testProfile(t), clock.read); err != nil {
			t.Fatal(err)
		}
	}
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		for index, err := range []error{second(r.Open(pooled("r"))), second(r.Post("ws1", "r", bill("p", "1"))), second(r.Explain("ws1", "r", explain("l", Tax, "1", "i"))), third(r.Close("ws1", "r")),
			r.Void("ws1", "r", VoidedBy), second(r.Readiness("ws1", "r")), second(r.Run("ws1", "r")), second(r.Parts("ws1", "r", 0, 1)), second(r.Lines("ws1", "r", 0, 1)),
			second(r.Holder("ws1", Subject{Basis: InternalCost, Customer: "c", Period: "p", Pool: "x"})), second(r.Current("ws1", "a", "b", "c"))} {
			if err != ErrInvalid {
				t.Fatalf("operation %d on a registry that was not built: %v", index, err)
			}
		}
	}
}

func TestRulesForStringsAndTime(t *testing.T) {
	long := ""
	for len(long) < 128 {
		long += "a"
	}
	for text, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, long + "a": false, "": false, "a b": false, "a/b": false, "a@b": false, "a[b": false, "a`b": false, "a{b": false} {
		if validID(text) != want {
			t.Errorf("identifier %q", text)
		}
	}
	for text, want := range map[string]bool{"USD": true, "AZZ": true, "ZAA": true, "US": false, "USDT": false, "uSD": false, "UsD": false, "USd": false, "@SD": false, "U[D": false, "US1": false, "": false} {
		if validCurrency(text) != want {
			t.Errorf("currency %q", text)
		}
	}
	h := newHarness(t, nil)
	h.clock.at = start.Add(time.Hour).In(time.FixedZone("kst", 9*3600))
	h.open(pooled("p1"))
	h.clock.advance(-30 * time.Minute)
	later := pooled("p2")
	later.Pool = "dc-busan"
	h.open(later)
	for _, id := range []string{"p1", "p2"} {
		if view := h.view(id); !view.Opened.Equal(start.Add(time.Hour)) || view.Opened.Location() != time.UTC {
			t.Fatalf("%s: %+v", id, view)
		}
	}
}

func third[A, B any](_ A, _ B, err error) error { return err }

func TestEachBasisHasItsExactFields(t *testing.T) {
	type change func(*RunSpec)
	other := []struct {
		name string
		set  change
	}{
		{"issuer", func(s *RunSpec) { s.Issuer = "aws" }}, {"account", func(s *RunSpec) { s.Account = "a" }}, {"document", func(s *RunSpec) { s.Document = "inv" }}, {"revision", func(s *RunSpec) { s.Revision = 1 }},
		{"total", func(s *RunSpec) { s.Total = "1" }}, {"input", func(s *RunSpec) { s.Input = "m" }}, {"parts", func(s *RunSpec) { s.Parts = 1 }}, {"rounding", func(s *RunSpec) { s.Rounding = "0" }},
	}
	for _, c := range other {
		for _, spec := range []RunSpec{priced("r1"), pooled("r1")} {
			h := newHarness(t, nil)
			c.set(&spec)
			_, err := h.r.Open(spec)
			wantErr(t, err, ErrInvalid, string(spec.Basis)+" with "+c.name)
			if h.r.dump() != "" {
				t.Fatal("a refused run wrote something")
			}
		}
	}
	cases := []struct {
		name string
		spec RunSpec
		set  change
	}{
		{"a document run for a customer", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Customer = "c" }},
		{"a document run with a period", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Period = "p" }},
		{"a document run with a charge", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Charge = "c" }},
		{"a document run with a pool", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Pool = "p" }},
		{"a document run with a quantity flag", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.QuantityComplete = true }},
		{"a document run with quantity evidence", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.QuantityEvidence = "e" }},
		{"a document run with price evidence", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.PriceEvidence = "e" }},
		{"a document run with a cost", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Cost = "1" }},
		{"a document run with cost evidence", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.CostEvidence = "e" }},
		{"a document run with pool evidence", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.PoolEvidence = "e" }},
		{"a document run with allocation evidence", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.AllocationEvidence = "e" }},
		{"no issuer", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Issuer = "" }},
		{"no account", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Account = "" }},
		{"no document", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Document = "" }},
		{"a document number with a slash", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Document = "INV/2026/1" }},
		{"revision zero", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Revision = 0 }},
		{"revision negative", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Revision = -1 }},
		{"revision too large", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Revision = 1000000 }},
		{"currency", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Currency = "usd" }},
		{"no total", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Total = "" }},
		{"a total beyond the input bound", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Total = "1000000000000" }},
		{"no input", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Input = "" }},
		{"no parts", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Parts = 0 }},
		{"no rounding", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Rounding = "" }},
		{"negative rounding", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Rounding = "-0.01" }},
		{"workspace", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.WorkspaceID = "" }},
		{"id", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.ID = "r 1" }},
		{"basis", doc("r1", 1, "USD", "10", 1), func(s *RunSpec) { s.Basis = "contract" }},
		{"price without a customer", priced("r1"), func(s *RunSpec) { s.Customer = "" }},
		{"price without a period", priced("r1"), func(s *RunSpec) { s.Period = "" }},
		{"price without a charge", priced("r1"), func(s *RunSpec) { s.Charge = "" }},
		{"price with a pool", priced("r1"), func(s *RunSpec) { s.Pool = "p" }},
		{"price without quantity evidence", priced("r1"), func(s *RunSpec) { s.QuantityEvidence = "" }},
		{"price without price evidence", priced("r1"), func(s *RunSpec) { s.PriceEvidence = "" }},
		{"price with pool evidence", priced("r1"), func(s *RunSpec) { s.PoolEvidence = "e" }},
		{"price with allocation evidence", priced("r1"), func(s *RunSpec) { s.AllocationEvidence = "e" }},
		{"price with a cost and no currency", priced("r1"), func(s *RunSpec) { s.Cost, s.CostEvidence = "1", "e" }},
		{"price with a cost and no evidence", priced("r1"), func(s *RunSpec) { s.Cost, s.Currency = "1", "USD" }},
		{"price with a currency alone", priced("r1"), func(s *RunSpec) { s.Currency = "USD" }},
		{"price with cost evidence alone", priced("r1"), func(s *RunSpec) { s.CostEvidence = "e" }},
		{"price with a negative cost", priced("r1"), func(s *RunSpec) { s.Cost, s.Currency, s.CostEvidence = "-1", "USD", "e" }},
		{"pool without a customer", pooled("r1"), func(s *RunSpec) { s.Customer = "" }},
		{"pool without a period", pooled("r1"), func(s *RunSpec) { s.Period = "" }},
		{"pool without a pool", pooled("r1"), func(s *RunSpec) { s.Pool = "" }},
		{"pool with a charge", pooled("r1"), func(s *RunSpec) { s.Charge = "c" }},
		{"pool without a cost", pooled("r1"), func(s *RunSpec) { s.Cost = "" }},
		{"pool with a negative cost", pooled("r1"), func(s *RunSpec) { s.Cost = "-0.0001" }},
		{"pool without a currency", pooled("r1"), func(s *RunSpec) { s.Currency = "" }},
		{"pool without pool evidence", pooled("r1"), func(s *RunSpec) { s.PoolEvidence = "" }},
		{"pool without allocation evidence", pooled("r1"), func(s *RunSpec) { s.AllocationEvidence = "" }},
		{"pool with a quantity flag", pooled("r1"), func(s *RunSpec) { s.QuantityComplete = true }},
		{"pool with quantity evidence", pooled("r1"), func(s *RunSpec) { s.QuantityEvidence = "e" }},
		{"pool with price evidence", pooled("r1"), func(s *RunSpec) { s.PriceEvidence = "e" }},
		{"pool with cost evidence", pooled("r1"), func(s *RunSpec) { s.CostEvidence = "e" }},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		c.set(&c.spec)
		_, err := h.r.Open(c.spec)
		wantErr(t, err, ErrInvalid, c.name)
		if h.r.dump() != "" {
			t.Fatalf("%s wrote something", c.name)
		}
	}
	h := newHarness(t, nil)
	h.open(doc("d1", 999999, "KRW", "-10", 1000))
	free := pooled("p1")
	free.Cost = "0"
	h.open(free)
	costed := priced("i1")
	costed.Cost, costed.Currency, costed.CostEvidence = "0", "USD", "ledger:1"
	h.open(costed)
	if view := h.view("p1"); view.State != Closed || !view.WasClosed || view.Verdict != "" || view.Residual != "" || view.Spec.Cost != "0" {
		t.Fatalf("%+v", view)
	}
}

func TestOpenRefusalsAfterTheShape(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Runs = 3; l.Parts = 4 })
	over := doc("r1", 1, "USD", "10", 5)
	_, err := h.r.Open(over)
	wantErr(t, err, ErrLimit, "more parts than a run may have")
	for name, change := range map[string]func(*RunSpec){
		"rounding over the maximum":                 func(s *RunSpec) { s.Rounding = "0.0501" },
		"rounding for a currency without a maximum": func(s *RunSpec) { s.Currency, s.Rounding = "EUR", "0.0001" },
	} {
		spec := doc("r1", 1, "USD", "10", 1)
		change(&spec)
		_, err := h.r.Open(spec)
		wantErr(t, err, ErrRejected, name)
	}
	if h.r.dump() != "" {
		t.Fatal("refusals wrote something")
	}
	edge := doc("r1", 1, "USD", "10", 4)
	h.open(edge)
	none := doc("r2", 1, "EUR", "10", 1)
	none.Rounding = "0"
	h.open(none)
	won := doc("r3", 1, "KRW", "10", 1)
	won.Rounding = "1.0"
	h.open(won)
	before := h.r.dump()
	_, err = h.r.Open(pooled("r4"))
	wantErr(t, err, ErrLimit, "run limit")
	if state, err := h.r.Open(edge); err != nil || state != Open {
		t.Fatal(state, err)
	}
	h.unchanged(before, "the run limit and a repeat at it")
}

func TestOneRunPerSubjectKey(t *testing.T) {
	h := newHarness(t, nil)
	h.open(doc("d1", 1, "USD", "10", 1))
	h.open(priced("i1"))
	h.open(pooled("p1"))
	h.clock.advance(time.Hour)
	before := h.r.dump()
	equal := doc("d1", 1, "USD", "10.00", 1)
	equal.Rounding = "0.050"
	if state, err := h.r.Open(equal); err != nil || state != Open {
		t.Fatal(state, err)
	}
	for name, spec := range map[string]RunSpec{
		"a second document run":  doc("d2", 1, "USD", "10", 1),
		"a second price run":     priced("i2"),
		"a second pool run":      pooled("p2"),
		"the total":              doc("d1", 1, "USD", "10.01", 1),
		"the parts":              doc("d1", 1, "USD", "10", 2),
		"the input":              func() RunSpec { s := doc("d1", 1, "USD", "10", 1); s.Input = "manifest:8"; return s }(),
		"the rounding":           func() RunSpec { s := doc("d1", 1, "USD", "10", 1); s.Rounding = "0.04"; return s }(),
		"the key under one id":   doc("d1", 1, "KRW", "10", 1),
		"the basis under one id": func() RunSpec { s := pooled("d1"); s.Pool = "other"; return s }(),
		"the quantity flag":      func() RunSpec { s := priced("i1"); s.QuantityComplete = false; return s }(),
		"the quantity evidence":  func() RunSpec { s := priced("i1"); s.QuantityEvidence = "usage:2"; return s }(),
		"the price evidence":     func() RunSpec { s := priced("i1"); s.PriceEvidence = "pricebook:4"; return s }(),
		"a cost added":           func() RunSpec { s := priced("i1"); s.Cost, s.Currency, s.CostEvidence = "1", "USD", "e"; return s }(),
		"the pool cost":          func() RunSpec { s := pooled("p1"); s.Cost = "1200.51"; return s }(),
		"the pool currency":      func() RunSpec { s := pooled("p1"); s.Currency = "USD"; return s }(),
		"the pool evidence":      func() RunSpec { s := pooled("p1"); s.PoolEvidence = "ledger:8"; return s }(),
		"the allocation":         func() RunSpec { s := pooled("p1"); s.AllocationEvidence = "alloc:3"; return s }(),
	} {
		_, err := h.r.Open(spec)
		wantErr(t, err, ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	for _, spec := range []RunSpec{
		func() RunSpec { s := priced("i3"); s.Charge = "other-charge"; return s }(), func() RunSpec { s := priced("i4"); s.Period = "2026-11"; return s }(), func() RunSpec { s := priced("i5"); s.Customer = "cust-2"; return s }(),
		func() RunSpec { s := pooled("p3"); s.Pool = "dc-busan"; return s }(), func() RunSpec { s := pooled("p4"); s.Period = "2026-11"; return s }(), func() RunSpec { s := pooled("p5"); s.Customer = "cust-2"; return s }(),
		func() RunSpec { s := doc("d3", 1, "USD", "10", 1); s.Issuer = "gcp"; return s }(), func() RunSpec { s := doc("d4", 1, "USD", "10", 1); s.Account = "acct-2"; return s }(),
		func() RunSpec { s := doc("d5", 1, "USD", "10", 1); s.Document = "inv-101"; return s }(), doc("d6", 1, "KRW", "10", 1),
	} {
		h.open(spec)
	}
	far := doc("d1", 1, "USD", "99", 3)
	far.WorkspaceID = "ws2"
	if state, err := h.r.Open(far); err != nil || state != Open {
		t.Fatalf("the same key and identifier in another workspace is another run: %v %v", state, err)
	}
	spelled := doc("d7", 1, "USD", "-0.0", 1)
	spelled.Document, spelled.Rounding = "inv-7", "0.0500"
	h.open(spelled)
	costly := pooled("p7")
	costly.Pool, costly.Cost = "dc-7", "-0"
	h.open(costly)
	if a, b := h.view("d7"), h.view("p7"); a.Spec.Total != "0" || a.Spec.Rounding != "0.05" || b.Spec.Cost != "0" || h.view("p1").Spec.Cost != "1200.5" {
		t.Fatalf("figures are kept in canonical form: %+v %+v", a.Spec, b.Spec)
	}
	if view := h.view("d1"); view.Spec.Total != "10" || !view.Opened.Equal(start) {
		t.Fatalf("%+v", view)
	}
	for _, c := range []struct {
		subject Subject
		want    string
		err     error
	}{
		{Subject{Basis: ProviderDocument, Issuer: "aws", Account: "acct-1", Document: "inv-100", Revision: 1, Currency: "USD"}, "d1", nil},
		{Subject{Basis: ProviderDocument, Issuer: "aws", Account: "acct-1", Document: "inv-100", Revision: 1, Currency: "EUR"}, "", ErrNotFound},
		{Subject{Basis: ProviderDocument, Issuer: "aws", Account: "acct-1", Document: "inv-100", Revision: 2, Currency: "USD"}, "", ErrNotFound},
		{Subject{Basis: IndependentPrice, Customer: "cust-1", Period: "2026-10", Charge: "support-plan"}, "i1", nil},
		{Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul"}, "p1", nil},
		{Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "nowhere"}, "", ErrNotFound},
		{Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10"}, "", ErrInvalid},
		{Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul", Charge: "x"}, "", ErrInvalid},
		{Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul", Issuer: "aws"}, "", ErrInvalid},
		{Subject{Basis: IndependentPrice, Customer: "cust-1", Period: "2026-10", Charge: "support-plan", Pool: "x"}, "", ErrInvalid},
		{Subject{Basis: IndependentPrice, Customer: "cust-1", Period: "2026-10", Charge: "support-plan", Revision: 1}, "", ErrInvalid},
		{Subject{Basis: ProviderDocument, Issuer: "aws", Account: "acct-1", Document: "inv-100", Revision: 1, Currency: "USD", Customer: "c"}, "", ErrInvalid},
		{Subject{Basis: "other"}, "", ErrInvalid},
		{Subject{Basis: IndependentPrice, Customer: "cust-1", Period: "2026-10", Charge: "support-plan", Currency: "USD"}, "", ErrInvalid},
		{Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul", Currency: "KRW"}, "", ErrInvalid},
	} {
		got, err := h.r.Holder("ws1", c.subject)
		if got != c.want || err != c.err {
			t.Fatalf("%+v: %q %v", c.subject, got, err)
		}
	}
	if _, err := h.r.Holder("ws3", Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul"}); err != ErrNotFound {
		t.Fatal(err)
	}
	if _, err := h.r.Holder("", Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul"}); err != ErrInvalid {
		t.Fatal(err)
	}
	for _, id := range []string{"d1", "i1", "p1"} {
		if err := h.r.Void("ws1", id, WrongFigures); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.r.Holder("ws1", Subject{Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul"}); err != ErrNotFound {
		t.Fatal(err)
	}
	h.open(doc("d2", 1, "USD", "10", 1))
	h.open(priced("i2"))
	h.open(pooled("p2"))
	before = h.r.dump()
	for _, spec := range []RunSpec{doc("d1", 1, "USD", "10", 1), priced("i1"), pooled("p1")} {
		if state, err := h.r.Open(spec); err != nil || state != Voided {
			t.Fatalf("an identical open of a voided run whose key another run took: %v %v", state, err)
		}
	}
	h.unchanged(before, "repeats of voided runs")
	if got, _ := h.r.Holder("ws1", Subject{Basis: IndependentPrice, Customer: "cust-1", Period: "2026-10", Charge: "support-plan"}); got != "i2" {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(h.view("d2").Spec, doc("d2", 1, "USD", "10", 1)) {
		t.Fatalf("%+v", h.view("d2").Spec)
	}
}
