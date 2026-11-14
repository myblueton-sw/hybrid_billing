package claims

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestFractionsInViews(t *testing.T) {
	limits := testLimits()
	limits.Reversals = 10
	r, _ := NewRegistry(testProfile(t), limits, time.Now)
	eligible(t, r, "c1", 1, resale())
	a := usage("u1", "c1", "line-1", "10", "5")
	billed(t, r, a)
	cases := []struct{ quantity, amount, reversedQ, reversedA, leftQ, leftA string }{
		{"0.5", "0.25", "0.5", "0.25", "9.5", "4.75"},
		{"0.001", "0.05", "0.501", "0.3", "9.499", "4.7"},
		{"0.0001", "0.00000008", "0.5011", "0.30000008", "9.4989", "4.69999992"},
		{"1.0249", "0.12", "1.526", "0.42000008", "8.474", "4.57999992"},
	}
	for index, c := range cases {
		id := []string{"r1", "r2", "r3", "r4"}[index]
		if _, err := r.Reverse("ws", Reversal{ID: id, Application: "u1", Quantity: c.quantity, Amount: c.amount}, "clerk"); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if _, err := r.ApproveReversal("ws", id, "lead"); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		v, _ := r.Application("ws", "u1")
		src, _ := r.Source("ws", "line-1")
		if v.ReversedQuantity != c.reversedQ || v.ReversedAmount != c.reversedA || src.Remaining[0].Quantity != c.leftQ || src.Remaining[0].Amount != c.leftA ||
			src.UsedQuantity != c.leftQ || src.UsedAmount != c.leftA {
			t.Errorf("%s: %+v %+v", id, v, src)
		}
	}
}

func TestFormsAtTheirBounds(t *testing.T) {
	r, _ := openWith(t)
	rev := resale()
	rev.From, rev.To = "2000-01-01", "2099-12-31"
	rev.Components[0].Instances = []string{strings.Repeat("Z", 128), "AZaz09.-_:"}
	if _, err := r.RecordEligibility("ws", strings.Repeat("a", 128), 1, rev, "author"); err != nil {
		t.Fatalf("bounds: %v", err)
	}
	for _, id := range []string{strings.Repeat("a", 129), "/a", "a@", "a[", "a`", "a{"} {
		if _, err := r.RecordEligibility("ws", id, 1, resale(), "author"); kind(err) != "invalid" {
			t.Errorf("id %q: %v", id, err)
		}
	}
	for _, date := range []string{"1999-12-31", "2100-01-01", "2026-1-01", "2026-02-29", "2024-02-30", "20260101", "2026/01/01"} {
		bad := resale()
		bad.From = date
		if _, err := r.RecordEligibility("ws", "c1", 1, bad, "author"); kind(err) != "invalid" {
			t.Errorf("date %q: %v", date, err)
		}
	}
	leap := resale()
	leap.From, leap.To = "2024-02-29", "2099-12-30"
	if _, err := r.RecordEligibility("ws", "c2", 1, leap, "author"); err != nil {
		t.Fatalf("a leap day: %v", err)
	}
	for _, code := range []string{"AZA", "ZAZ"} {
		a := monthly("cur-"+code, "c3", "fee-1", "2026-03-01", "2026-04-01", "1")
		a.Currency = code
		if _, err := r.Reserve("ws", a, "runner"); err != nil {
			t.Errorf("currency %s: %v", code, err)
		}
	}
	for _, code := range []string{"@AA", "A[A", "AA@", "[AA", "A@A", "AA[", "AAAA", "AA"} {
		a := monthly("cur", "c3", "fee-1", "2026-03-01", "2026-04-01", "1")
		a.Currency = code
		if _, err := r.Reserve("ws", a, "runner"); kind(err) != "invalid" {
			t.Errorf("currency %q: %v", code, err)
		}
	}
	wide, _ := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 70, InputScale: 8, ResultIntegerDigits: 80, ResultScale: 8, IntermediateDigits: 200})
	w, _ := NewRegistry(wide, testLimits(), time.Now)
	eligible(t, w, "c1", 1, resale())
	if d, err := w.Reserve("ws", monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", strings.Repeat("9", 64)), "a"); err != nil || d.Reason != "" {
		t.Fatalf("64 bytes: %v", err)
	}
	if _, err := w.Reserve("ws", monthly("n", "c1", "fee-1", "2026-03-01", "2026-04-01", strings.Repeat("9", 65)), "a"); kind(err) != "invalid" {
		t.Fatal("65 bytes")
	}
	long := monthly("life", "c1", "fee-2", "2026-03-01", "2026-04-01", "1")
	long.Lifetime = 86400
	if d, err := w.Reserve("ws", long, "a"); err != nil || !d.Application.Expiry.Equal(d.Application.ReservedAt.Add(24*time.Hour)) {
		t.Fatalf("a lifetime of a day: %v", err)
	}
}

func TestCountsAtTheirBounds(t *testing.T) {
	limits := Limits{Contracts: 4, Revisions: 3, Components: 4, Applications: 12, Sources: 3, Reversals: 3, Batch: 2}
	r, _ := NewRegistry(testProfile(t), limits, time.Now)
	rev := resale()
	rev.Components[0].Instances = make([]string, 1000)
	for index := range rev.Components[0].Instances {
		rev.Components[0].Instances[index] = "i" + strings.Repeat("0", 4-len(itoa(index))) + itoa(index)
	}
	if _, err := r.RecordEligibility("ws", "c1", 1, rev, "author"); err != nil {
		t.Fatalf("a thousand instances: %v", err)
	}
	r.ApproveEligibility("ws", "c1", 1, "approver")
	reserved(t, r, monthly("a", "c1", "fee-1", "2026-03-01", "2026-04-01", "1"))
	reserved(t, r, monthly("b", "c1", "fee-1", "2026-04-01", "2026-05-01", "1"))
	if _, _, err := r.Confirm("ws", []string{"a", "b"}, "runner"); err != nil {
		t.Fatalf("a batch at its limit: %v", err)
	}
	for _, number := range []int{1000} {
		if _, err := r.RecordEligibility("ws", "c1", number, resale(), "author"); kind(err) != "rejected" {
			t.Errorf("record %d: %v", number, err)
		}
		if _, err := r.ApproveEligibility("ws", "c1", number, "x"); kind(err) != "not_found" {
			t.Errorf("approve %d: %v", number, err)
		}
		if _, err := r.Eligibility("ws", "c1", number); kind(err) != "not_found" {
			t.Errorf("read %d: %v", number, err)
		}
	}
	for name, call := range map[string]func() error{
		"approve 1001": func() error { _, err := r.ApproveEligibility("ws", "c1", 1001, "x"); return err },
		"read 1001":    func() error { _, err := r.Eligibility("ws", "c1", 1001); return err },
		"read 0":       func() error { _, err := r.Eligibility("ws", "c1", 0); return err },
		"approve ws":   func() error { _, err := r.ApproveEligibility("", "c1", 1, "x"); return err },
		"approve c":    func() error { _, err := r.ApproveEligibility("ws", "", 1, "x"); return err },
		"read ws":      func() error { _, err := r.Eligibility("", "c1", 1); return err },
		"release ws":   func() error { _, err := r.Release("", "a", "x"); return err },
		"release id":   func() error { _, err := r.Release("ws", "", "x"); return err },
		"coverage ws":  func() error { _, err := r.Coverage("", Scope{"c1", "fee", "fee-1"}); return err },
		"coverage c":   func() error { _, err := r.Coverage("ws", Scope{"", "fee", "fee-1"}); return err },
		"coverage i":   func() error { _, err := r.Coverage("ws", Scope{"c1", "fee", ""}); return err },
		"app ws":       func() error { _, err := r.Application("", "a"); return err },
		"source id":    func() error { _, err := r.Source("ws", ""); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.RecordEligibility("ws", "c1", 1, rev, "author"); err != nil {
		t.Fatal("a repeat of the last revision")
	}
	if _, err := r.Eligibility("ws", "c1", 2); kind(err) != "not_found" {
		t.Fatal("a repeat stored a second revision")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

func TestRevisionConflictsFieldByField(t *testing.T) {
	r, _ := openWith(t)
	if _, err := r.RecordEligibility("ws", "c1", 1, resale(), "author"); err != nil {
		t.Fatal(err)
	}
	for index, change := range []func(*Revision){
		func(v *Revision) { v.From = "2026-01-02" }, func(v *Revision) { v.To = "2026-12-31" }, func(v *Revision) { v.Path = PathDirect; v.Components[0].Billable = false },
		func(v *Revision) { v.Components = v.Components[:2] }, func(v *Revision) { v.Components[0].ID = "fee0" },
		func(v *Revision) { v.Components[0].Kind = KindOther }, func(v *Revision) { v.Components[0].Billable = false },
		func(v *Revision) { v.Components[0].Route = RouteUnknown }, func(v *Revision) { v.Components[0].Route, v.Components[0].Reference = RouteExternal, "x" },
		func(v *Revision) { v.Components[1].Instances = []string{"fee-1"} }, func(v *Revision) { v.Components[1].Instances = []string{"fee-1", "fee-3"} },
	} {
		v := resale()
		change(&v)
		if _, err := r.RecordEligibility("ws", "c1", 1, v, "author"); kind(err) != "conflict" {
			t.Errorf("variant %d: %v", index, err)
		}
	}
}

func TestAdjacentRevisionsAndBoundaries(t *testing.T) {
	r, _ := openWith(t)
	late := resale()
	late.From, late.To = "2026-06-01", "2027-01-01"
	eligible(t, r, "c1", 1, late)
	early := resale()
	early.From, early.To = "2026-01-01", "2026-06-01"
	eligible(t, r, "c1", 2, early)
	if v, _ := r.Eligibility("ws", "c1", 1); v.EffectiveTo != "2027-01-01" {
		t.Fatal("an adjacent earlier revision ended a later one")
	}
	billed(t, r, monthly("jan", "c1", "fee-1", "2026-01-01", "2026-02-01", "1"))
	billed(t, r, monthly("jun", "c1", "fee-1", "2026-06-01", "2026-07-01", "1"))
	renamed := resale()
	renamed.From, renamed.To = "2026-02-01", "2026-06-01"
	renamed.Components[1].Instances = []string{"fee-9"}
	eligible(t, r, "c1", 3, renamed)
	if v, _ := r.Eligibility("ws", "c1", 2); v.EffectiveTo != "2026-02-01" {
		t.Fatalf("superseded at the boundary: %+v", v)
	}
	r2, _ := openWith(t)
	eligible(t, r2, "c1", 1, early)
	eligible(t, r2, "c1", 2, late)
	if v, _ := r2.Eligibility("ws", "c1", 1); v.EffectiveTo != "2026-06-01" {
		t.Fatal("an adjacent later revision ended an earlier one")
	}
	billed(t, r2, monthly("mar", "c1", "fee-1", "2026-03-01", "2026-04-01", "1"))
	dropped := resale()
	dropped.From, dropped.To = "2026-02-01", "2026-06-01"
	dropped.Components = dropped.Components[:1]
	if _, err := r2.RecordEligibility("ws", "c1", 3, dropped, "author"); err != nil {
		t.Fatal(err)
	}
	if _, err := r2.ApproveEligibility("ws", "c1", 3, "approver"); kind(err) != "rejected" {
		t.Fatalf("a revision dropping a billed component: %v", err)
	}
}

func TestCoverageOrder(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	reserved(t, r, monthly("z-apr", "c1", "fee-1", "2026-04-01", "2026-05-01", "1"))
	reserved(t, r, monthly("y-mar", "c1", "fee-1", "2026-03-01", "2026-04-01", "1"))
	reserved(t, r, usage("u2", "c1", "line-1", "1", "1"))
	reserved(t, r, usage("u1", "c1", "line-2", "1", "1"))
	later := usage("u0", "c1", "line-3", "1", "1")
	later.Start = "2026-03-15"
	reserved(t, r, later)
	ids := func(scope Scope) []string {
		out := []string{}
		list, _ := r.Coverage("ws", scope)
		for _, v := range list {
			out = append(out, v.Application.ID)
		}
		return out
	}
	if got := ids(Scope{"c1", "fee", "fee-1"}); !reflect.DeepEqual(got, []string{"y-mar", "z-apr"}) {
		t.Fatalf("by start: %v", got)
	}
	if got := ids(Scope{"c1", "usage", "main"}); !reflect.DeepEqual(got, []string{"u1", "u2", "u0"}) {
		t.Fatalf("by start then identifier: %v", got)
	}
}

func TestCoverageOrderWithEqualStarts(t *testing.T) {
	limits := testLimits()
	limits.Sources = 20
	limits.Applications = 20
	r, _ := NewRegistry(testProfile(t), limits, time.Now)
	eligible(t, r, "c1", 1, resale())
	for _, id := range []string{"u5", "u2", "u9", "u1", "u7", "u3", "u8", "u4", "u6"} {
		reserved(t, r, usage(id, "c1", "line-"+id, "1", "1"))
	}
	list, _ := r.Coverage("ws", Scope{"c1", "usage", "main"})
	got := []string{}
	for _, v := range list {
		got = append(got, v.Application.ID)
	}
	if !reflect.DeepEqual(got, []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8", "u9"}) {
		t.Fatalf("order: %v", got)
	}
}

func TestAPathAloneConflicts(t *testing.T) {
	r, _ := openWith(t)
	base := Revision{From: "2026-01-01", To: "2027-01-01", Path: PathResale, Components: []Component{comp("fee", KindMSPService, true, RoutePlatform, "fee-1")}}
	if _, err := r.RecordEligibility("ws", "c1", 1, base, "author"); err != nil {
		t.Fatal(err)
	}
	base.Path = PathDirect
	if _, err := r.RecordEligibility("ws", "c1", 1, base, "author"); kind(err) != "conflict" {
		t.Fatalf("the path: %v", err)
	}
}
