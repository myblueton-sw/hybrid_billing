package claims

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestNewRegistry(t *testing.T) {
	good := testLimits()
	if _, err := NewRegistry(decimal.Profile{}, good, time.Now); kind(err) != "invalid" {
		t.Fatal("profile")
	}
	if _, err := NewRegistry(testProfile(t), good, nil); kind(err) != "invalid" {
		t.Fatal("clock")
	}
	for index, change := range []func(*Limits){
		func(l *Limits) { l.Contracts = 0 }, func(l *Limits) { l.Contracts = 100001 }, func(l *Limits) { l.Revisions = 0 }, func(l *Limits) { l.Revisions = 1001 },
		func(l *Limits) { l.Components = 0 }, func(l *Limits) { l.Components = 1001 }, func(l *Limits) { l.Applications = 0 }, func(l *Limits) { l.Applications = 1000001 },
		func(l *Limits) { l.Sources = 0 }, func(l *Limits) { l.Sources = 1000001 }, func(l *Limits) { l.Reversals = 0 }, func(l *Limits) { l.Reversals = 1001 },
		func(l *Limits) { l.Batch = 0 }, func(l *Limits) { l.Batch = 10001 },
	} {
		limits := good
		change(&limits)
		if _, err := NewRegistry(testProfile(t), limits, time.Now); kind(err) != "invalid" {
			t.Errorf("limit %d", index)
		}
	}
	for _, limits := range []Limits{{1, 1, 1, 1, 1, 1, 1}, {100000, 1000, 1000, 1000000, 1000000, 1000, 10000}} {
		if _, err := NewRegistry(testProfile(t), limits, time.Now); err != nil {
			t.Fatalf("%+v", limits)
		}
	}
}

func TestEligibilityForms(t *testing.T) {
	r, _ := openWith(t)
	bad := map[string]func(*Revision){
		"from":               func(v *Revision) { v.From = "2026-02-30" },
		"to":                 func(v *Revision) { v.To = "2100-01-01" },
		"to before from":     func(v *Revision) { v.To = v.From },
		"path":               func(v *Revision) { v.Path = "agency" },
		"no components":      func(v *Revision) { v.Components = nil },
		"component id":       func(v *Revision) { v.Components[0].ID = "a b" },
		"component twice":    func(v *Revision) { v.Components[1].ID = "usage" },
		"kind":               func(v *Revision) { v.Components[0].Kind = "tax" },
		"route":              func(v *Revision) { v.Components[0].Route = "erp" },
		"no instances":       func(v *Revision) { v.Components[0].Instances = nil },
		"instance form":      func(v *Revision) { v.Components[0].Instances = []string{""} },
		"instance twice":     func(v *Revision) { v.Components[1].Instances = []string{"a", "a"} },
		"external no ref":    func(v *Revision) { v.Components[0].Route = RouteExternal },
		"reference platform": func(v *Revision) { v.Components[0].Reference = "x" },
		"reference form":     func(v *Revision) { v.Components[0].Route, v.Components[0].Reference = RouteExternal, "a b" },
	}
	for name, change := range bad {
		rev := resale()
		change(&rev)
		if _, err := r.RecordEligibility("ws", "c1", 1, rev, "author-1"); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, call := range map[string]func() error{
		"workspace": func() error { _, err := r.RecordEligibility("", "c1", 1, resale(), "a"); return err },
		"contract":  func() error { _, err := r.RecordEligibility("ws", "", 1, resale(), "a"); return err },
		"author":    func() error { _, err := r.RecordEligibility("ws", "c1", 1, resale(), ""); return err },
		"zero":      func() error { _, err := r.RecordEligibility("ws", "c1", 0, resale(), "a"); return err },
		"above":     func() error { _, err := r.RecordEligibility("ws", "c1", 1001, resale(), "a"); return err },
		"approve":   func() error { _, err := r.ApproveEligibility("ws", "c1", 0, "a"); return err },
		"approver":  func() error { _, err := r.ApproveEligibility("ws", "c1", 1, ""); return err },
		"read":      func() error { _, err := r.Eligibility("ws", "", 1); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	many := resale()
	for index := 0; index < 5; index++ {
		many.Components = append(many.Components, Component{})
	}
	if _, err := r.RecordEligibility("ws", "c1", 1, many, "a"); kind(err) != "limit" {
		t.Fatal("components before form")
	}
	wide := resale()
	wide.Components[0].Instances = make([]string, 1001)
	if _, err := r.RecordEligibility("ws", "c1", 1, wide, "a"); kind(err) != "limit" {
		t.Fatal("instances before form")
	}
	for _, call := range []func() error{
		func() error { _, err := r.ApproveEligibility("ws", "c1", 1, "a"); return err },
		func() error { _, err := r.Eligibility("ws", "c1", 1); return err },
		func() error { _, err := r.ApproveEligibility("other", "c1", 1, "a"); return err },
	} {
		if err := call(); kind(err) != "not_found" {
			t.Errorf("not found: %v", err)
		}
	}
	for index := 1; index <= 3; index++ {
		rev := resale()
		rev.From = []string{"", "2026-01-01", "2026-03-01", "2026-05-01"}[index]
		if _, err := r.RecordEligibility("ws", "c1", index, rev, "a"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.RecordEligibility("ws", "c1", 4, resale(), "a"); kind(err) != "limit" {
		t.Fatal("revisions")
	}
	for _, id := range []string{"c2", "c3", "c4"} {
		if _, err := r.RecordEligibility("ws", id, 1, resale(), "a"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.RecordEligibility("ws", "c5", 1, resale(), "a"); kind(err) != "limit" {
		t.Fatal("contracts")
	}
	if _, err := r.RecordEligibility("ws", "c1", 1, func() Revision { v := resale(); return v }(), "a"); err != nil {
		t.Fatal("a repeat at the limit")
	}
}

func TestApplicationForms(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	bad := map[string]func(*Application){
		"id":                  func(a *Application) { a.ID = "" },
		"contract":            func(a *Application) { a.Scope.Contract = "a b" },
		"component":           func(a *Application) { a.Scope.Component = "" },
		"instance":            func(a *Application) { a.Scope.Instance = "" },
		"start":               func(a *Application) { a.Start = "2026-02-29" },
		"end":                 func(a *Application) { a.End = "2026-13-01" },
		"end before start":    func(a *Application) { a.End = a.Start },
		"mode":                func(a *Application) { a.Mode = "flat" },
		"currency":            func(a *Application) { a.Currency = "usd" },
		"calculation":         func(a *Application) { a.Calculation = "" },
		"lifetime zero":       func(a *Application) { a.Lifetime = 0 },
		"lifetime above":      func(a *Application) { a.Lifetime = 86401 },
		"replaces form":       func(a *Application) { a.Replaces = "a b" },
		"replaces itself":     func(a *Application) { a.Replaces = a.ID },
		"corrects without":    func(a *Application) { a.Corrects = "x" },
		"kind without":        func(a *Application) { a.CorrectsKind = CorrectsReversal },
		"corrects kind":       func(a *Application) { a.CorrectsKind, a.Corrects = "invoice", "x" },
		"corrects itself":     func(a *Application) { a.CorrectsKind, a.Corrects = CorrectsApplication, a.ID },
		"corrects form":       func(a *Application) { a.CorrectsKind, a.Corrects = CorrectsApplication, "a b" },
		"quantity":            func(a *Application) { a.Quantity = "-1" },
		"amount":              func(a *Application) { a.Amount = "1e2" },
		"amount scale":        func(a *Application) { a.Amount = "0.123456789" },
		"source in interval":  func(a *Application) { a.Source = "s" },
		"capacity interval":   func(a *Application) { a.CapQuantity = "1" },
		"cap amount interval": func(a *Application) { a.CapAmount = "1" },
	}
	for name, change := range bad {
		a := monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "100")
		change(&a)
		if _, err := r.Reserve("ws", a, "runner-1"); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, change := range map[string]func(*Application){
		"source":       func(a *Application) { a.Source = "" },
		"cap quantity": func(a *Application) { a.CapQuantity = "-1" },
		"cap amount":   func(a *Application) { a.CapAmount = "" },
		"cap size":     func(a *Application) { a.CapAmount = strings.Repeat("1", 65) },
	} {
		a := usage("u", "c1", "line-1", "1", "1")
		change(&a)
		if _, err := r.Reserve("ws", a, "runner-1"); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.Reserve("bad ws", monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "1"), "a"); kind(err) != "invalid" {
		t.Fatal("workspace")
	}
	if _, err := r.Reserve("ws", monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "1"), ""); kind(err) != "invalid" {
		t.Fatal("actor")
	}
	for name, change := range map[string]func(*Application){
		"replaces":             func(a *Application) { a.Replaces = "none" },
		"corrects application": func(a *Application) { a.CorrectsKind, a.Corrects = CorrectsApplication, "none" },
		"corrects reversal":    func(a *Application) { a.CorrectsKind, a.Corrects = CorrectsReversal, "none" },
	} {
		a := monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "100")
		change(&a)
		if _, err := r.Reserve("ws", a, "runner-1"); kind(err) != "not_found" {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOrderOfKinds(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, monthly("m3", "c1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	reserved(t, r, monthly("m4", "c1", "fee-1", "2026-04-01", "2026-05-01", "100"))
	stored := monthly("m4", "c1", "fee-1", "2026-04-01", "2026-05-01", "100")
	stored.Lifetime = 60
	if _, err := r.Reserve("ws", stored, "runner-1"); kind(err) != "conflict" {
		t.Fatal("another lifetime")
	}
	if d, err := r.Reserve("ws", monthly("m4", "c1", "fee-1", "2026-04-01", "2026-05-01", "100.00"), "another"); err != nil || d.Application.ReservedBy != "runner-1" {
		t.Fatal("a repeat by another actor")
	}
	wrong := monthly("x", "c1", "fee-1", "2026-03-01", "2026-04-01", "1")
	wrong.CorrectsKind, wrong.Corrects = CorrectsApplication, "m4"
	if _, err := r.Reserve("ws", wrong, "a"); kind(err) != "rejected" {
		t.Fatal("correcting a reservation")
	}
	foreign := monthly("y", "c1", "fee-2", "2026-03-01", "2026-04-01", "1")
	foreign.CorrectsKind, foreign.Corrects = CorrectsApplication, "m3"
	if _, err := r.Reserve("ws", foreign, "a"); kind(err) != "rejected" {
		t.Fatal("correcting another scope")
	}
	r.Reverse("ws", Reversal{ID: "rv", Application: "m3", Quantity: "1", Amount: "100"}, "clerk-1")
	pending := monthly("z", "c1", "fee-1", "2026-03-01", "2026-04-01", "1")
	pending.CorrectsKind, pending.Corrects = CorrectsReversal, "rv"
	if _, err := r.Reserve("ws", pending, "a"); kind(err) != "rejected" {
		t.Fatal("correcting a pending reversal")
	}
	r.ApproveReversal("ws", "rv", "lead-1")
	other := monthly("z", "c1", "fee-2", "2026-03-01", "2026-04-01", "1")
	other.CorrectsKind, other.Corrects = CorrectsReversal, "rv"
	if _, err := r.Reserve("ws", other, "a"); kind(err) != "rejected" {
		t.Fatal("correcting a reversal of another scope")
	}
	full, _ := openWith(t)
	eligible(t, full, "c1", 1, resale())
	for index := 0; index < 12; index++ {
		a := monthly("f"+string(rune('a'+index)), "c1", "fee-1", "", "", "1")
		a.Start = []string{"2026-01-01", "2026-02-01", "2026-03-01", "2026-04-01", "2026-05-01", "2026-06-01", "2026-07-01", "2026-08-01", "2026-09-01", "2026-10-01", "2026-11-01", "2026-12-01"}[index]
		a.End = []string{"2026-02-01", "2026-03-01", "2026-04-01", "2026-05-01", "2026-06-01", "2026-07-01", "2026-08-01", "2026-09-01", "2026-10-01", "2026-11-01", "2026-12-01", "2027-01-01"}[index]
		reserved(t, full, a)
	}
	if reason := refused(t, full, monthly("over", "c1", "fee-1", "2026-03-01", "2026-04-01", "1")); reason != ReasonAlreadyClaimed {
		t.Fatalf("a refusal before room: %s", reason)
	}
	if _, err := full.Reserve("ws", monthly("over", "c1", "fee-2", "2026-03-01", "2026-04-01", "1"), "a"); kind(err) != "limit" {
		t.Fatal("room")
	}
	sources, _ := openWith(t)
	eligible(t, sources, "c1", 1, resale())
	for _, id := range []string{"s1", "s2", "s3"} {
		reserved(t, sources, usage("u-"+id, "c1", id, "1", "1"))
	}
	if _, err := sources.Reserve("ws", usage("u-s4", "c1", "s4", "1", "1"), "a"); kind(err) != "limit" {
		t.Fatal("sources")
	}
	reserved(t, sources, usage("u-s1b", "c1", "s1", "1", "1"))
}

func TestOrderOfRefusals(t *testing.T) {
	r, _ := openWith(t)
	rev := resale()
	rev.Components = append(rev.Components[:2], comp("ext", KindOther, true, RouteExternal, "e"), comp("unk", KindOther, false, RouteUnknown, "u"))
	eligible(t, r, "c1", 1, rev)
	cases := []struct {
		change func(*Application)
		want   string
	}{
		{func(a *Application) { a.End = "2027-02-01"; a.Scope.Component = "none" }, ReasonNoEligibility},
		{func(a *Application) { a.Scope.Component = "unk" }, ReasonNotEligible},
		{func(a *Application) { a.Scope.Instance = "x"; a.Scope.Component = "ext" }, ReasonUnknownInstance},
		{func(a *Application) { a.Scope.Component, a.Scope.Instance = "ext", "e" }, ReasonRoutedExternally},
	}
	for _, c := range cases {
		a := monthly("q", "c1", "fee-1", "2026-03-01", "2026-04-01", "1")
		c.change(&a)
		if reason := refused(t, r, a); reason != c.want {
			t.Errorf("%s: %s", c.want, reason)
		}
	}
	billed(t, r, usage("u1", "c1", "line-1", "100", "50"))
	both := usage("u2", "c1", "line-1", "1", "1")
	both.Currency, both.CapQuantity = "EUR", "7"
	if reason := refused(t, r, both); reason != ReasonCurrencyMismatch {
		t.Fatalf("currency before capacity: %s", reason)
	}
	cap := usage("u2", "c1", "line-1", "1", "1")
	cap.CapQuantity = "7"
	if reason := refused(t, r, cap); reason != ReasonCapacityMismatch {
		t.Fatalf("capacity before exceeding: %s", reason)
	}
	eligible(t, r, "c2", 1, resale())
	over := usage("u3", "c2", "line-1", "1", "1")
	if reason := refused(t, r, over); reason != ReasonSourceInOtherScope {
		t.Fatalf("scope before exceeding: %s", reason)
	}
	interval := monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "1")
	interval.Scope.Component, interval.Scope.Instance = "usage", "main"
	interval.Currency = "EUR"
	if reason := refused(t, r, interval); reason != ReasonModeMismatch {
		t.Fatalf("mode before currency: %s", reason)
	}
	newSource := usage("u4", "c1", "line-9", "101", "1")
	newSource.CapQuantity = "100"
	if reason := refused(t, r, newSource); reason != ReasonCapacityExceeded {
		t.Fatalf("a new source over its capacity: %s", reason)
	}
}

func TestCapacityIsOpenUntilConfirmed(t *testing.T) {
	r, clock := openWith(t)
	eligible(t, r, "c1", 1, resale())
	typo := usage("u1", "c1", "line-1", "1", "1")
	typo.CapQuantity = "1000"
	reserved(t, r, typo)
	r.Release("ws", "u1", "runner-1")
	reserved(t, r, usage("u2", "c1", "line-1", "1", "1"))
	short := usage("u3", "c1", "line-1", "1", "1")
	short.CapAmount, short.Lifetime = "7", 5
	if reason := refused(t, r, short); reason != ReasonCapacityMismatch {
		t.Fatalf("an active reservation fixes the capacity: %s", reason)
	}
	clock.advance(time.Hour)
	reserved(t, r, short)
	if s, _ := r.Source("ws", "line-1"); s.Amount != "7" {
		t.Fatalf("set again after expiry: %+v", s)
	}
	confirm(t, r, "u3")
	reverse(t, r, "rv", "u3", "1", "1")
	again := usage("u4", "c1", "line-1", "1", "1")
	if reason := refused(t, r, again); reason != ReasonCapacityMismatch {
		t.Fatalf("fixed after a confirmation: %s", reason)
	}
}

func TestReversalRefusals(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, usage("u1", "c1", "line-1", "10", "5"))
	reserved(t, r, usage("u2", "c1", "line-1", "10", "5"))
	for name, call := range map[string]func() error{
		"workspace": func() error { _, err := r.Reverse("", Reversal{"rv", "u1", "1", "1"}, "c"); return err },
		"actor":     func() error { _, err := r.Reverse("ws", Reversal{"rv", "u1", "1", "1"}, ""); return err },
		"id":        func() error { _, err := r.Reverse("ws", Reversal{"", "u1", "1", "1"}, "c"); return err },
		"app":       func() error { _, err := r.Reverse("ws", Reversal{"rv", "", "1", "1"}, "c"); return err },
		"quantity":  func() error { _, err := r.Reverse("ws", Reversal{"rv", "u1", "-1", "1"}, "c"); return err },
		"amount":    func() error { _, err := r.Reverse("ws", Reversal{"rv", "u1", "1", "x"}, "c"); return err },
		"approve":   func() error { _, err := r.ApproveReversal("ws", "", "c"); return err },
		"approver":  func() error { _, err := r.ApproveReversal("ws", "rv", ""); return err },
		"list":      func() error { _, err := r.Reversals("ws", ""); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.Reverse("ws", Reversal{"rv", "none", "1", "1"}, "c"); kind(err) != "not_found" {
		t.Fatal("no application")
	}
	if _, err := r.ApproveReversal("ws", "none", "c"); kind(err) != "not_found" {
		t.Fatal("no reversal")
	}
	if _, err := r.Reversals("ws", "none"); kind(err) != "not_found" {
		t.Fatal("no application to list")
	}
	if _, err := r.Reverse("ws", Reversal{"rv", "u2", "1", "1"}, "c"); kind(err) != "rejected" {
		t.Fatal("reversing a reservation")
	}
	if _, err := r.Reverse("ws", Reversal{"rv", "u1", "10.0001", "1"}, "c"); kind(err) != "rejected" {
		t.Fatal("beyond the quantity")
	}
	if _, err := r.Reverse("ws", Reversal{"rv", "u1", "1", "5.01"}, "c"); kind(err) != "rejected" {
		t.Fatal("beyond the amount")
	}
	if _, err := r.Reverse("ws", Reversal{"rv", "u1", "6", "3"}, "c"); err != nil {
		t.Fatal(err)
	}
	if v, err := r.Reverse("ws", Reversal{"rv", "u1", "6.0", "3"}, "d"); err != nil || v.Author != "c" {
		t.Fatal("a repeat")
	}
	if _, err := r.Reverse("ws", Reversal{"rv", "u1", "5", "3"}, "c"); kind(err) != "conflict" {
		t.Fatal("a conflict")
	}
	if _, err := r.Reverse("ws", Reversal{"rv2", "u1", "6", "3"}, "c"); err != nil {
		t.Fatal(err)
	}
	r.ApproveReversal("ws", "rv", "lead")
	if _, err := r.ApproveReversal("ws", "rv2", "lead"); kind(err) != "rejected" {
		t.Fatal("two pending reversals beyond the application")
	}
	if _, err := r.Reverse("ws", Reversal{"rv3", "u1", "4", "2"}, "c"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reverse("ws", Reversal{"rv4", "u1", "1", "1"}, "c"); kind(err) != "limit" {
		t.Fatalf("reversals: %v", err)
	}
}

func TestReadsAndTheTenantBoundary(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, usage("u1", "c1", "line-1", "10", "5"))
	if reason, _ := r.Reserve("other", usage("u1", "c1", "line-1", "10", "5"), "a"); reason.Reason != ReasonNoEligibility {
		t.Fatal("another workspace has no eligibility")
	}
	for name, call := range map[string]func() error{
		"app":       func() error { _, err := r.Application("other", "u1"); return err },
		"coverage":  func() error { _, err := r.Coverage("other", Scope{"c1", "usage", "main"}); return err },
		"source":    func() error { _, err := r.Source("other", "line-1"); return err },
		"reversals": func() error { _, err := r.Reversals("other", "u1"); return err },
		"confirm":   func() error { _, _, err := r.Confirm("other", []string{"u1"}, "a"); return err },
		"release":   func() error { _, err := r.Release("other", "u1", "a"); return err },
		"no scope":  func() error { _, err := r.Coverage("ws", Scope{"c1", "usage", "x"}); return err },
		"elig":      func() error { _, err := r.Eligibility("other", "c1", 1); return err },
	} {
		if err := call(); kind(err) != "not_found" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, call := range map[string]func() error{
		"app":      func() error { _, err := r.Application("ws", ""); return err },
		"coverage": func() error { _, err := r.Coverage("ws", Scope{"c1", "", "main"}); return err },
		"source":   func() error { _, err := r.Source("", "line-1"); return err },
		"confirm":  func() error { _, _, err := r.Confirm("ws", nil, "a"); return err },
		"actor":    func() error { _, _, err := r.Confirm("ws", []string{"u1"}, ""); return err },
		"bad id":   func() error { _, _, err := r.Confirm("ws", []string{"a b"}, "a"); return err },
		"release":  func() error { _, err := r.Release("ws", "u1", ""); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.Application("other", "u1"); kind(err) != "not_found" {
		t.Fatal("refused reservation created a workspace view")
	}
	if len(r.workspaces) != 1 {
		t.Fatalf("workspaces created by refusals: %d", len(r.workspaces))
	}
	v, _ := r.Application("ws", "u1")
	if v.ReservedAt != start || v.ConfirmedAt != start || v.Expiry != start.Add(time.Hour) || v.ReversedQuantity != "0" {
		t.Fatalf("times: %+v", v)
	}
}

func TestDigests(t *testing.T) {
	r, _ := openWith(t)
	view, _ := r.RecordEligibility("ws", "c1", 1, resale(), "author-1")
	want := printed("billing_eligibility_v1", "ws", "c1", "1", "2026-01-01", "2027-01-01", "resale", "3",
		"fee", "msp_service", "true", "platform", "", "2", "fee-1", "fee-2",
		"support", "other", "false", "platform", "", "1", "s",
		"usage", "csp_usage", "true", "platform", "", "1", "main")
	if view.Digest != want {
		t.Fatalf("eligibility digest")
	}
	r.ApproveEligibility("ws", "c1", 1, "approver-1")
	a := usage("u1", "c1", "line-1", "10", "5.0")
	a.CorrectsKind, a.Corrects, a.Replaces = "", "", ""
	d := reserve(t, r, a)
	if d.Application.Digest != printed("billing_application_v1", "ws", "u1", "c1", "usage", "main", "2026-03-01", "2026-04-01", "source", "line-1", "100", "50",
		"10", "5", "USD", "calc-1", "", "", "") {
		t.Fatal("application digest")
	}
	confirm(t, r, "u1")
	rv, _ := r.Reverse("ws", Reversal{"rv", "u1", "1.0", "1"}, "c")
	if rv.Digest != printed("billing_reversal_v1", "ws", "rv", "u1", "1", "1") {
		t.Fatal("reversal digest")
	}
	a.Amount = "5"
	seen := map[string]bool{d.Application.Digest: true}
	for index, change := range []func(*Application){
		func(a *Application) { a.ID = "u9" }, func(a *Application) { a.Scope.Instance = "x" }, func(a *Application) { a.Start = "2026-03-02" },
		func(a *Application) { a.End = "2026-03-31" }, func(a *Application) { a.Source = "line-9" }, func(a *Application) { a.CapQuantity = "101" },
		func(a *Application) { a.CapAmount = "51" }, func(a *Application) { a.Quantity = "11" }, func(a *Application) { a.Amount = "6" },
		func(a *Application) { a.Currency = "EUR" }, func(a *Application) { a.Calculation = "calc-2" }, func(a *Application) { a.Replaces = "r" },
		func(a *Application) { a.CorrectsKind, a.Corrects = CorrectsApplication, "x" }, func(a *Application) { a.CorrectsKind, a.Corrects = CorrectsReversal, "x" },
		func(a *Application) { a.Scope.Contract = "c9" }, func(a *Application) { a.Scope.Component = "fee" },
	} {
		b := a
		change(&b)
		got := applicationDigest("ws", b)
		if seen[got] {
			t.Errorf("variant %d", index)
		}
		seen[got] = true
	}
	if applicationDigest("ws2", a) == d.Application.Digest {
		t.Fatal("workspace")
	}
	b := a
	b.Lifetime = 1
	if applicationDigest("ws", b) != d.Application.Digest {
		t.Fatal("lifetime printed")
	}
	el, _ := r.Eligibility("ws", "c1", 1)
	el.Revision.Components[0].Instances[0] = "zz"
	if again, _ := r.Eligibility("ws", "c1", 1); again.Revision.Components[0].Instances[0] != "fee-1" {
		t.Fatal("a view shares its instances")
	}
}

func TestConcurrentReservations(t *testing.T) {
	for round := 0; round < 50; round++ {
		r, _ := openWith(t)
		eligible(t, r, "c1", 1, resale())
		var group sync.WaitGroup
		gate := make(chan struct{})
		reasons := make([]string, 8)
		for index := 0; index < 8; index++ {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				<-gate
				var a Application
				if index%2 == 0 {
					a = monthly("m"+string(rune('a'+index)), "c1", "fee-1", "2026-03-01", "2026-04-01", "1")
				} else {
					a = usage("u"+string(rune('a'+index)), "c1", "line-1", "40", "20")
				}
				d, _ := r.Reserve("ws", a, "runner")
				reasons[index] = d.Reason
			}(index)
		}
		close(gate)
		group.Wait()
		intervals, sources := 0, 0
		for index, reason := range reasons {
			if reason == "" && index%2 == 0 {
				intervals++
			}
			if reason == "" && index%2 == 1 {
				sources++
			}
		}
		if intervals != 1 || sources != 2 {
			t.Fatalf("intervals %d sources %d: %v", intervals, sources, reasons)
		}
	}
}

func TestTheSourceHasNoFloatingPointOrComments(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	scanned := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		scanned++
		body := readSource(file)
		for _, word := range []string{"float", "Float", "big.Rat", "//", "/*", "1e"} {
			if strings.Contains(body, word) {
				t.Errorf("%s contains %q", file, word)
			}
		}
	}
	if scanned < 6 {
		t.Fatalf("scanned %d", scanned)
	}
}
