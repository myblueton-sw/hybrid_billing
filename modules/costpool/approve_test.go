package costpool

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestTheCasesOfTheAcceptance(t *testing.T) {
	h := newHarness(t, nil)
	h.open(header(1, 10, 0))
	h.asset(1, Asset{ID: "srv-1", Kind: Server, Method: Depreciation, Acquisition: "20000000"})
	h.asset(1, Asset{ID: "rack-1", Kind: FacilityRoom, Method: Lease})
	on := func(made Component, asset, container string) Component {
		made.Asset, made.IncludedIn = asset, container
		return made
	}
	h.put(1, cost("fee", Facility, "200000"), on(cost("lease", Lease, "300000"), "rack-1", ""), on(power("p1", "100000", Measured, 10), "", "fee"), on(power("p2", "150000", Estimated, 10), "srv-1", ""),
		cost("lic", Licence, "100000"), on(cost("ops", Operations, "50000"), "srv-1", ""), on(cost("ops2", Operations, ""), "", "lease"))
	h.basis(1, Basis{ID: "basis-1", Measure: ReservedCapacity, Capacity: "100", Targets: []Target{{ID: "org-a", Bearer: Organization, Weight: "60"}, {ID: "org-b", Bearer: Organization, Weight: "20"}, {ID: "spare", Bearer: MSP, Weight: "20", Idle: true}}})
	if got := h.view(1).Blockers; got.Summed != 5 || got.WithoutAmount != 0 || got.AssetsWithoutComponent != 1 || !got.BasisSet || got.Multiple != MultipleYes || got.NamedSuperseded || got.Overlapping.Total != 0 || got.Overlapping.Items != nil {
		t.Fatalf("%+v", got)
	}
	before := h.r.dump()
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrRejected, "an asset without the component of its method")
	h.unchanged(before, "a refused approval")
	h.put(1, on(cost("dep", Depreciation, ""), "srv-1", ""))
	if got := h.view(1).Blockers; got.WithoutAmount != 1 || got.AssetsWithoutComponent != 0 || got.Multiple != MultipleNotDetermined {
		t.Fatalf("%+v", got)
	}
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrRejected, "a depreciation that is not known is not zero")
	if err := h.r.Withdraw("ws1", "dc-1", 1, EnteredInError); err != nil {
		t.Fatal(err)
	}
	h.open(header(2, 10, 0))
	h.asset(2, Asset{ID: "srv-1", Kind: Server, Method: Depreciation, Acquisition: "20000000"})
	h.asset(2, Asset{ID: "rack-1", Kind: FacilityRoom, Method: Lease})
	h.put(2, cost("fee", Facility, "200000"), on(cost("lease", Lease, "300000"), "rack-1", ""), on(power("p1", "100000", Measured, 10), "", "fee"), on(power("p2", "150000", Estimated, 10), "srv-1", ""),
		cost("lic", Licence, "100000"), on(cost("ops", Operations, "50000"), "srv-1", ""), on(cost("ops2", Operations, ""), "", "lease"), on(cost("dep", Depreciation, "400000"), "srv-1", ""))
	h.basis(2, Basis{ID: "basis-1", Measure: ReservedCapacity, Capacity: "100", Targets: []Target{{ID: "spare", Bearer: MSP, Weight: "20", Idle: true}, {ID: "org-b", Bearer: Organization, Weight: "20.0"}, {ID: "org-a", Bearer: Organization, Weight: "60"}}})
	h.clock.advance(time.Hour)
	h.record(2)
	view := h.view(2)
	want := "total=1200000 assets=2 included=1/1 prices=1/1 pool=300000 idle=240000 bearer/msp=240000 bearer/organization=960000 grade/estimated=150000 " +
		"kind/depreciation=400000 kind/facility=200000 kind/lease=300000 kind/licence=100000 kind/operations=50000 kind/power=150000"
	if got := flat(view.Figures); got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if got := h.shares(2); got != "org-a=720000(organization/60) org-b=240000(organization/20.0) spare=240000(msp/20/idle)" {
		t.Fatal(got)
	}
	if got := h.assetSums(2); got != "rack-1=300000 srv-1=600000" {
		t.Fatal(got)
	}
	if view.State != ApprovalRecorded || !view.Recorded || view.Approval != approved || view.Blockers != nil || !view.Changed.Equal(start.Add(time.Hour)) || !view.Opened.Equal(start) || h.standing(2) != "current[]" {
		t.Fatalf("%+v", view)
	}
	view.Figures.ByKind[Depreciation], view.Figures.ByGrade[Estimated], view.Figures.ByBearer[MSP], view.Figures.Total = "9", "9", "9", "9"
	if again := flat(h.view(2).Figures); again != want {
		t.Fatal("figures share state")
	}
	for name, read := range map[string]func() error{
		"shares":     func() error { return second(h.r.Shares("ws1", "dc-1", 1, "", 1)) },
		"asset sums": func() error { return second(h.r.AssetSums("ws1", "dc-1", 1, "", 1)) },
	} {
		wantErr(t, read(), ErrRejected, name+" of a version without figures")
	}
	page, err := h.r.Shares("ws1", "dc-1", 2, "org-a", 1)
	if err != nil || len(page) != 1 || page[0].Target != "org-b" {
		t.Fatalf("%+v %v", page, err)
	}
	page[0].Amount = "1"
	sums, err := h.r.AssetSums("ws1", "dc-1", 2, "rack-1", 4)
	if err != nil || len(sums) != 1 || sums[0] != (AssetSum{Asset: "srv-1", Sum: "600000"}) {
		t.Fatalf("%+v %v", sums, err)
	}
	sums[0].Sum = "1"
	if h.shares(2) != "org-a=720000(organization/60) org-b=240000(organization/20.0) spare=240000(msp/20/idle)" || h.assetSums(2) != "rack-1=300000 srv-1=600000" {
		t.Fatal("pages share state")
	}
}

func TestRecordingAnApproval(t *testing.T) {
	h := newHarness(t, nil)
	h.open(header(1, 10, 0))
	for _, bad := range []Approval{{}, {Approver: "a"}, {Reference: "r"}, {Approver: "kim minsu", Reference: "r"}, {Approver: "a", Reference: "see mail"}} {
		wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, bad)), ErrInvalid, "an approval")
		wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 9, bad)), ErrInvalid, "malformed before the version is looked up")
	}
	step := func(what string) {
		t.Helper()
		before := h.r.dump()
		wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrRejected, what)
		h.unchanged(before, what)
	}
	step("an empty version")
	h.basis(1, shares("1", "1", "1"))
	step("a basis and no component")
	h.put(1, cost("fee", Facility, "100"))
	h.put(1, change(cost("inside", Operations, "5"), func(c *Component) { c.IncludedIn = "fee" }))
	h.put(1, cost("late", Licence, ""))
	step("a summed component without an amount")
	if err := h.r.Withdraw("ws1", "dc-1", 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	step("a withdrawn draft")
	h = newHarness(t, nil)
	h.open(header(1, 10, 0))
	h.put(1, cost("fee", Facility, "100"), change(cost("inside", Operations, ""), func(c *Component) { c.IncludedIn = "fee" }))
	step("components and no basis")
	if got := h.view(1).Blockers; got.BasisSet || got.Summed != 1 || got.Multiple != MultipleYes {
		t.Fatalf("%+v", got)
	}
	h.basis(1, shares("1", "1", "1"))
	h.clock.advance(time.Hour)
	h.record(1)
	twice := newHarness(t, nil)
	twice.open(header(1, 10, 0))
	twice.put(1, power("p1", "10", Measured, 10), power("p2", "20", Measured, 10), power("p3", "5", GradeNotKnown, 10))
	twice.basis(1, shares("1"))
	twice.record(1)
	if got := flat(twice.view(1).Figures); got != "total=35 assets=0 included=0/0 prices=0/0 pool=35 bearer/organization=35 grade/measured=30 grade/not_known=5 kind/power=35" {
		t.Fatal(got)
	}
	if got := h.shares(1); got != "t1=34(organization/1) t2=33(organization/1) t3=33(msp/1)" {
		t.Fatalf("the unit left over goes to the first identifier: %s", got)
	}
	if got := flat(h.view(1).Figures); got != "total=100 assets=0 included=0/1 prices=0/0 pool=100 bearer/msp=33 bearer/organization=67 kind/facility=100" {
		t.Fatal(got)
	}
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.record(1)
	for _, other := range []Approval{{Approver: "approver-2", Reference: approved.Reference}, {Approver: approved.Approver, Reference: "approval-doc-2"}} {
		wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, other)), ErrConflict, "a second approval")
	}
	for name, write := range map[string]func() error{
		"an asset": func() error {
			return second(h.r.PutAsset("ws1", "dc-1", 1, Asset{ID: "a", Kind: Server, Method: Lease}))
		},
		"a component": func() error { return second(h.r.PutComponent("ws1", "dc-1", 1, cost("more", Licence, "1"))) },
	} {
		wantErr(t, write(), ErrRejected, name+" after the approval was recorded")
	}
	wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 1, shares("1", "2"))), ErrConflict, "another basis")
	if state, err := h.r.SetBasis("ws1", "dc-1", 1, shares("1", "1", "1")); err != nil || state != ApprovalRecorded {
		t.Fatal(state, err)
	}
	if state, err := h.r.PutComponent("ws1", "dc-1", 1, cost("fee", Facility, "100")); err != nil || state != ApprovalRecorded {
		t.Fatal(state, err)
	}
	h.unchanged(before, "repeats after the approval")
	if err := h.r.Withdraw("ws1", "dc-1", 1, EnteredInError); err != nil {
		t.Fatal(err)
	}
	if state, err := h.r.RecordApproval("ws1", "dc-1", 1, approved); err != nil || state != Withdrawn {
		t.Fatalf("a repeated approval after a withdrawal: %v %v", state, err)
	}
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, Approval{Approver: "x", Reference: "y"})), ErrConflict, "another approval after a withdrawal")
	if view := h.view(1); view.Figures == nil || h.shares(1) == "" || view.Blockers != nil || view.WithdrawReason != EnteredInError {
		t.Fatalf("figures stay readable: %+v", view)
	}
}

func TestQuantaTotalsAndBounds(t *testing.T) {
	cases := []struct {
		quantum string
		amounts []string
		weights []string
		want    string
		shares  string
	}{
		{"1", []string{"0"}, []string{"1", "1"}, "total=0", "t1=0 t2=0"},
		{"1", []string{"0.5", "0.5"}, []string{"1", "0"}, "total=1", "t1=1 t2=0"},
		{"1", []string{"100.5"}, []string{"1", "1"}, "", ""},
		{"1", []string{"100.0001", "0.9998"}, []string{"1", "1"}, "", ""},
		{"0.1", []string{"100.5"}, []string{"1", "1"}, "total=100.5", "t1=50.3 t2=50.2"},
		{"0.1", []string{"100.55"}, []string{"1", "1"}, "", ""},
		{"0.01", []string{"100.01"}, []string{"1", "1"}, "total=100.01", "t1=50.01 t2=50"},
		{"0.01", []string{"100"}, []string{"1", "1", "1"}, "total=100", "t1=33.34 t2=33.33 t3=33.33"},
		{"0.001", []string{"0.001"}, []string{"1", "1"}, "total=0.001", "t1=0.001 t2=0"},
		{"0.001", []string{"0.0005"}, []string{"1", "1"}, "", ""},
		{"0.0001", []string{"0.0005", "1.2"}, []string{"3", "1"}, "total=1.2005", "t1=0.9004 t2=0.3001"},
		{"1", []string{"7"}, []string{"0.5", "0.25", "0.25"}, "total=7", "t1=3 t2=2 t3=2"},
		{"1", []string{"999999999999", "999999999999"}, []string{"1"}, "total=1999999999998", "t1=1999999999998"},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := header(1, 10, 0)
		spec.Quantum = c.quantum
		h.open(spec)
		for index, amount := range c.amounts {
			h.put(1, cost(fmt.Sprintf("c%d", index), Operations, amount))
		}
		made := Basis{ID: "b", Measure: FixedAgreement, Capacity: "1"}
		if len(c.weights) == 2 && c.weights[0] == "1" && c.weights[1] == "1" {
			made.Capacity = "2"
		}
		if len(c.weights) == 3 && c.weights[0] == "1" {
			made.Capacity = "3"
		}
		if c.weights[0] == "3" {
			made.Capacity = "4"
		}
		for index, weight := range c.weights {
			made.Targets = append(made.Targets, Target{ID: fmt.Sprintf("t%d", index+1), Bearer: MSP, Weight: weight})
		}
		h.basis(1, made)
		name := fmt.Sprintf("%v at %s", c.amounts, c.quantum)
		if c.want == "" {
			if got := h.view(1).Blockers.Multiple; got != MultipleNo {
				t.Fatalf("%s: %s", name, got)
			}
			before := h.r.dump()
			wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrRejected, name)
			h.unchanged(before, name)
			continue
		}
		if got := h.view(1).Blockers.Multiple; got != MultipleYes {
			t.Fatalf("%s: %s", name, got)
		}
		h.record(1)
		var plain []string
		for _, part := range strings.Fields(h.shares(1)) {
			plain = append(plain, strings.SplitN(part, "(", 2)[0])
		}
		if got := flat(h.view(1).Figures); !strings.HasPrefix(got, c.want+" ") || strings.Join(plain, " ") != c.shares || h.view(1).Figures.ByBearer[MSP] != strings.TrimPrefix(c.want, "total=") {
			t.Fatalf("%s:\n got %s %v\nwant %s %s", name, got, plain, c.want, c.shares)
		}
	}
	clock := &testClock{at: start}
	small, err := NewRegistry(testLimits(), profileOf(t, 3, 3), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, r: small, clock: clock}
	h.open(header(1, 10, 0))
	h.put(1, cost("a", Operations, "999"), cost("b", Operations, "1"))
	h.basis(1, shares("1", "1"))
	if got := h.view(1).Blockers; got.Multiple != MultipleNotDetermined || got.WithoutAmount != 0 || got.Summed != 2 {
		t.Fatalf("%+v", got)
	}
	before := h.r.dump()
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrLimit, "a total the profile cannot hold")
	h.unchanged(before, "a bound of the profile")
	h.open(header(2, 11, 0))
	h.put(2, cost("a", Operations, "999"), cost("b", Operations, "0.5"), cost("c", Operations, "0.5"))
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 2, approved)), ErrRejected, "what stands against an approval is said before a bound")
	h.basis(2, shares("1", "1"))
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 2, approved)), ErrLimit, "a total that the profile cannot hold has no multiple to judge")
	h.open(header(3, 12, 0))
	h.asset(3, Asset{ID: "s", Kind: Server, Method: Lease})
	h.put(3, cost("a", Operations, "499"), cost("b", Licence, "400"), change(cost("l", Lease, "99"), func(c *Component) { c.Asset = "s" }), change(cost("o", Operations, "1"), func(c *Component) { c.Asset = "s" }))
	h.basis(3, shares("1", "1"))
	h.record(3)
	if got := flat(h.view(3).Figures); got != "total=999 assets=1 included=0/0 prices=0/1 pool=899 bearer/organization=999 kind/lease=99 kind/licence=400 kind/operations=500" || h.assetSums(3) != "s=100" {
		t.Fatal(got)
	}
	tight, err := decimal.NewProfile(decimal.ProfileSpec{ID: "tight", Revision: "1", InputIntegerDigits: 3, InputScale: 4, ResultIntegerDigits: 3, ResultScale: 4, IntermediateDigits: 7})
	if err != nil {
		t.Fatal(err)
	}
	bound, err := NewRegistry(testLimits(), tight, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	h = &harness{t: t, r: bound, clock: clock}
	spec := header(1, 10, 0)
	spec.Quantum = "0.0001"
	h.open(spec)
	h.put(1, cost("a", Operations, "100.0001"))
	h.basis(1, shares("1", "1"))
	before = h.r.dump()
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrLimit, "the primitive's bound of intermediates")
	h.unchanged(before, "a bound of the primitive")
	narrow := newHarness(t, func(l *Limits) { l.Intermediate = 6 })
	narrow.fill(header(1, 10, 0), "1200000")
	before = narrow.r.dump()
	wantErr(t, second(narrow.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrLimit, "the allocation's bound of intermediates")
	narrow.unchanged(before, "a bound of the allocation")
	narrow.fill(header(2, 11, 0), "120")
	narrow.record(2)
	if got := narrow.shares(2); got != "t1=72(organization/60) t2=24(organization/20) t3=24(msp/20)" {
		t.Fatal(got)
	}
}

func TestOverlapStandingAndCounting(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Versions = 20 })
	from, to := month(10)
	if counted, err := h.r.Counting("ws1", "dc-1", from, to); err != ErrNotFound || counted != (Counted{}) {
		t.Fatal(counted, err)
	}
	h.fill(header(1, 10, 0), "1200000")
	if number, sequence := h.counting(10); number != 0 || sequence != 3 || h.standing(1) != "not_current[no_approval_recorded]" {
		t.Fatal(number, sequence, h.standing(1))
	}
	h.open(header(1, 10, 0))
	h.put(1, cost("fee", Facility, "1200000"))
	h.basis(1, shares("60", "20", "20"))
	for name, call := range map[string]func() error{
		"a refused component":  func() error { return second(h.r.PutComponent("ws1", "dc-1", 1, cost("fee", Facility, "1"))) },
		"a refused asset":      func() error { return second(h.r.PutAsset("ws1", "dc-1", 1, Asset{ID: "a"})) },
		"a refused basis":      func() error { return second(h.r.SetBasis("ws1", "dc-1", 1, shares("1"))) },
		"a refused approval":   func() error { return second(h.r.RecordApproval("ws1", "dc-1", 1, Approval{})) },
		"a refused withdrawal": func() error { return h.r.Withdraw("ws1", "dc-1", 1, "because") },
		"a refused header":     func() error { return second(h.r.Open(change(header(1, 10, 0), func(s *Spec) { s.Owner = "x" }))) },
	} {
		if call() == nil {
			t.Fatal(name)
		}
	}
	if _, sequence := h.counting(10); sequence != 3 {
		t.Fatalf("repeats and refusals do not move the number: %d", sequence)
	}
	h.asset(1, Asset{ID: "a", Kind: Server, Method: Lease})
	h.put(1, change(cost("l", Lease, "0"), func(c *Component) { c.Asset = "a" }))
	h.record(1)
	h.record(1)
	counted, err := h.r.Counting("ws1", "dc-1", from.In(time.FixedZone("kst", 9*3600)), to)
	if err != nil || counted != (Counted{Version: 1, Sequence: 6}) {
		t.Fatalf("%+v %v", counted, err)
	}
	for name, period := range map[string][2]time.Time{"a day of the month": {from, from.Add(day)}, "its last day": {to.Add(-day), to}, "the month and a second": {from, to.Add(time.Second)}} {
		got, err := h.r.Counting("ws1", "dc-1", period[0], period[1])
		if err != ErrNotFound || got != (Counted{Sequence: 6}) {
			t.Fatalf("%s: %+v %v", name, got, err)
		}
	}
	for name, period := range map[string][2]time.Time{"backwards": {to, from}, "empty": {from, from}, "a fraction at the start": {from.Add(time.Millisecond), to}, "a fraction at the end": {from, to.Add(time.Millisecond)}, "too long": {from, from.Add(40*day + time.Second)}, "none": {}} {
		_, err := h.r.Counting("ws1", "dc-1", period[0], period[1])
		wantErr(t, err, ErrInvalid, name)
	}
	for _, c := range []struct {
		ws, pool string
		want     error
	}{{"ws2", "dc-1", ErrNotFound}, {"ws1", "dc-9", ErrNotFound}, {"", "dc-1", ErrInvalid}, {"ws1", "dc 1", ErrInvalid}} {
		_, err := h.r.Counting(c.ws, c.pool, from, to)
		wantErr(t, err, c.want, "counting")
	}
	part := func(number, first, days int) Spec {
		spec := header(number, 10, 0)
		spec.PeriodStart, spec.PeriodEnd = from.Add(time.Duration(first)*day), from.Add(time.Duration(first+days)*day)
		return spec
	}
	h.fill(part(2, 30, 5), "10")
	h.fill(part(3, 0, 1), "10")
	h.fill(part(4, -1, 2), "10")
	h.fill(header(5, 10, 0), "10")
	h.fill(header(6, 11, 0), "10")
	h.fill(header(7, 9, 0), "10")
	for _, number := range []int{2, 3, 4, 5} {
		before := h.r.dump()
		wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", number, approved)), ErrRejected, "a version that overlaps the current one")
		h.unchanged(before, "an overlap")
		if got := h.view(number).Blockers.Overlapping; got.Total != 1 || got.Items[0] != 1 {
			t.Fatalf("%d: %+v", number, got)
		}
	}
	h.record(6)
	h.record(7)
	if h.standing(6) != "current[]" || h.standing(7) != "current[]" || h.standing(1) != "current[]" {
		t.Fatal("periods that only touch do not overlap")
	}
	if got := h.view(4).Blockers.Overlapping; got.Total != 2 || fmt.Sprint(got.Items) != "[1 7]" {
		t.Fatalf("%+v", got)
	}
	if got := h.view(2).Blockers.Overlapping; got.Total != 2 || fmt.Sprint(got.Items) != "[1 6]" {
		t.Fatalf("%+v", got)
	}
	wide := header(8, 10, 0)
	wide.PeriodStart, wide.PeriodEnd = from.Add(-2*day), to.Add(2*day)
	h.fill(wide, "10")
	if got := h.view(8).Blockers.Overlapping; got.Total != 3 || fmt.Sprint(got.Items) != "[1 6]" {
		t.Fatalf("the list is cut at its bound: %+v", got)
	}
	h.fill(header(9, 10, 1), "900000")
	if got := h.view(9).Blockers; got.Overlapping.Total != 0 || got.NamedSuperseded {
		t.Fatalf("the named version is left out: %+v", got)
	}
	h.record(9)
	if number, _ := h.counting(10); number != 9 || h.standing(1) != "not_current[superseded]" {
		t.Fatal(number)
	}
	if got := h.view(5).Blockers.Overlapping; got.Total != 1 || got.Items[0] != 9 {
		t.Fatalf("%+v", got)
	}
	for _, c := range []struct {
		reason WithdrawReason
		want   error
	}{{"", ErrInvalid}, {"replaced", ErrInvalid}, {"customer_rejected", ErrInvalid}} {
		wantErr(t, h.r.Withdraw("ws1", "dc-1", 9, c.reason), c.want, "a reason")
	}
	_, sequence := h.counting(10)
	h.clock.advance(time.Hour)
	h.clock.advance(-3 * time.Hour)
	if err := h.r.Withdraw("ws1", "dc-1", 9, EnteredInError); err != nil {
		t.Fatal(err)
	}
	if err := h.r.Withdraw("ws1", "dc-1", 9, EnteredInError); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Withdraw("ws1", "dc-1", 9, ByOperator), ErrConflict, "a second reason")
	if number, after := h.counting(10); number != 0 || after != sequence+1 || h.standing(9) != "not_current[withdrawn]" {
		t.Fatal(number, after)
	}
	if view := h.view(9); view.WithdrawReason != EnteredInError || view.Changed.Before(start) || view.Figures == nil {
		t.Fatalf("%+v", view)
	}
	h.record(5)
	if number, _ := h.counting(10); number != 5 {
		t.Fatal("after the withdrawal the period is free")
	}
	wantErr(t, second(h.r.RecordApproval("ws1", "dc-1", 2, approved)), ErrRejected, "a version across two current ones")
	rows, err := h.r.Versions("ws1", "dc-1", 0, 4)
	if err != nil || len(rows) != 4 || rows[0] != (Row{Version: 1, PeriodStart: from, PeriodEnd: to, State: ApprovalRecorded, Superseded: true}) || rows[3].Version != 4 || rows[3].State != Draft {
		t.Fatalf("%+v %v", rows, err)
	}
	rows, err = h.r.Versions("ws1", "dc-1", 7, 4)
	if err != nil || len(rows) != 2 || rows[0].Version != 8 || rows[1].Version != 9 || rows[1].State != Withdrawn || rows[1].Superseded {
		t.Fatalf("%+v %v", rows, err)
	}
	if rows, err := h.r.Versions("ws1", "dc-9", 0, 4); err != nil || rows != nil {
		t.Fatal(rows, err)
	}
	if rows, err := h.r.Versions("ws2", "dc-1", 0, 4); err != nil || rows != nil {
		t.Fatal(rows, err)
	}
	for _, c := range []struct{ after, limit int }{{-1, 1}, {0, 0}, {0, 5}} {
		wantErr(t, second(h.r.Versions("ws1", "dc-1", c.after, c.limit)), ErrInvalid, "a page of versions")
	}
	for _, c := range []struct {
		after string
		limit int
	}{{"a b", 1}, {"", 0}, {"", 5}} {
		wantErr(t, second(h.r.Assets("ws1", "dc-1", 1, c.after, c.limit)), ErrInvalid, "a page of assets")
		wantErr(t, second(h.r.Components("ws1", "dc-1", 1, c.after, c.limit)), ErrInvalid, "a page of components")
		wantErr(t, second(h.r.Targets("ws1", "dc-1", 1, c.after, c.limit)), ErrInvalid, "a page of targets")
		wantErr(t, second(h.r.Shares("ws1", "dc-1", 1, c.after, c.limit)), ErrInvalid, "a page of shares")
		wantErr(t, second(h.r.AssetSums("ws1", "dc-1", 1, c.after, c.limit)), ErrInvalid, "a page of asset sums")
	}
	if page, err := h.r.Components("ws1", "dc-1", 1, "zzz", 4); err != nil || page != nil {
		t.Fatal(page, err)
	}
	if page, err := h.r.Components("ws1", "dc-1", 1, "g", 4); err != nil || len(page) != 1 || page[0].ID != "l" {
		t.Fatalf("a position between two identifiers: %+v %v", page, err)
	}
}

func TestTheBoundaryOfAWorkspace(t *testing.T) {
	h := newHarness(t, nil)
	h.whole(header(1, 10, 0), "1200000")
	alone := h.r.dump()
	far := header(1, 10, 0)
	far.WorkspaceID = "ws2"
	if _, err := h.r.Open(far); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.PutComponent("ws2", "dc-1", 1, cost("fee", Facility, "5")); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.SetBasis("ws2", "dc-1", 1, shares("1")); err != nil {
		t.Fatal(err)
	}
	if state, err := h.r.RecordApproval("ws2", "dc-1", 1, approved); err != nil || state != ApprovalRecorded {
		t.Fatalf("the same period in another workspace does not overlap: %v %v", state, err)
	}
	if err := h.r.Withdraw("ws2", "dc-1", 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(alone, "\n") {
		if !strings.Contains(h.r.dump(), line) {
			t.Fatalf("another workspace changed: %s", line)
		}
	}
	if number, sequence := h.counting(10); number != 1 || sequence != 4 || h.shares(1) != "t1=720000(organization/60) t2=240000(organization/20) t3=240000(msp/20)" {
		t.Fatal(number, sequence)
	}
	var unbuilt Registry
	var missing *Registry
	from, to := month(10)
	for _, r := range []*Registry{&unbuilt, missing} {
		wantErr(t, second(r.Open(header(1, 10, 0))), ErrInvalid, "open")
		wantErr(t, second(r.PutAsset("ws1", "dc-1", 1, Asset{ID: "a", Kind: Server, Method: Lease})), ErrInvalid, "asset")
		wantErr(t, second(r.PutComponent("ws1", "dc-1", 1, cost("fee", Facility, "5"))), ErrInvalid, "component")
		wantErr(t, second(r.RecordApproval("ws1", "dc-1", 1, approved)), ErrInvalid, "record")
		wantErr(t, r.Withdraw("ws1", "dc-1", 1, ByOperator), ErrInvalid, "withdraw")
		wantErr(t, second(r.Standing("ws1", "dc-1", 1)), ErrInvalid, "standing")
		wantErr(t, second(r.Counting("ws1", "dc-1", from, to)), ErrInvalid, "counting")
		wantErr(t, second(r.Version("ws1", "dc-1", 1)), ErrInvalid, "version")
		wantErr(t, second(r.Versions("ws1", "dc-1", 0, 1)), ErrInvalid, "versions")
		wantErr(t, second(r.Shares("ws1", "dc-1", 1, "", 1)), ErrInvalid, "shares")
	}
}

func TestRefusalsAreFixedValuesAndTheSourceOnlyAdds(t *testing.T) {
	h := newHarness(t, nil)
	secret := "customer-9f3a"
	spec := header(1, 10, 0)
	spec.Pool, spec.Owner, spec.Currency = secret, secret, "xx"
	_, err := h.r.Open(spec)
	if err != ErrInvalid || strings.Contains(err.Error(), secret) {
		t.Fatal(err)
	}
	for _, kind := range []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit, ErrSelfCheck} {
		if strings.ContainsAny(kind.Error(), "%0123456789") {
			t.Fatalf("%q", kind)
		}
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := regexp.MustCompile(`float32|float64|math\.|math/|ParseFloat|big\.|\.Quo\(|\.Div\(|\.Mul\(|time\.Now|rand\.|(?i)sale|price\b|revenue|margin`)
	primitive := regexp.MustCompile(`decimal\.[A-Za-z]+`)
	allowed := map[string]bool{"decimal.Parse": true, "decimal.Add": true, "decimal.Cmp": true, "decimal.Decimal": true, "decimal.Profile": true, "decimal.HalfEven": true,
		"decimal.ErrResultBound": true, "decimal.ErrIntermediateBound": true}
	methods := regexp.MustCompile(`\.(Coefficient|Quanta|RoundToQuantum|DivideRound)\(`)
	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		if strings.Contains(string(content), "primitives/decimal") != strings.Contains(string(content), "\n\t\"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal\"\n") {
			t.Errorf("%s imports the primitive under another name", name)
		}
		for number, line := range strings.Split(string(content), "\n") {
			if found := banned.FindString(line); found != "" {
				t.Errorf("%s:%d uses %s", name, number+1, found)
			}
			if found := methods.FindString(line); found != "" {
				t.Errorf("%s:%d calls %s", name, number+1, found)
			}
			for _, use := range primitive.FindAllString(line, -1) {
				if !allowed[use] {
					t.Errorf("%s:%d uses %s", name, number+1, use)
				}
			}
			if strings.Contains(line, "//") || strings.Contains(line, "/*") {
				t.Errorf("%s:%d has a comment", name, number+1)
			}
		}
	}
	if scanned != 4 {
		t.Fatalf("scanned %d source files", scanned)
	}
}

func TestAVersionAtItsLimits(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Assets, l.Components, l.Targets, l.Page = 300, 900, 200, 50 })
	h.open(header(1, 10, 0))
	for index := 0; index < 300; index++ {
		id := fmt.Sprintf("a%03d", index)
		h.asset(1, Asset{ID: id, Kind: Server, Method: Depreciation, Acquisition: "5000"})
		h.put(1, change(cost("d"+id, Depreciation, "100"), func(c *Component) { c.Asset = id }), change(cost("o"+id, Operations, "10.5"), func(c *Component) { c.Asset = id }), change(cost("p"+id, Licence, "1"), func(c *Component) { c.Asset = id }))
	}
	made := Basis{ID: "b", Measure: ReservedCapacity, Capacity: "200"}
	for index := 0; index < 200; index++ {
		made.Targets = append(made.Targets, Target{ID: fmt.Sprintf("t%03d", 199-index), Bearer: []Bearer{Organization, MSP, Unattributed}[index%3], Weight: "1", Idle: index%3 == 2})
	}
	h.basis(1, made)
	h.record(1)
	figures := h.view(1).Figures
	if figures.HasPoolLevel {
		t.Fatal("no component is at the level of the pool")
	}
	parts := strings.Fields(h.shares(1))
	sums := strings.Fields(h.assetSums(1))
	whole := new(big.Rat)
	for _, part := range parts {
		amount, read := new(big.Rat).SetString(strings.SplitN(strings.SplitN(part, "=", 2)[1], "(", 2)[0])
		if !read || amount.Cmp(big.NewRat(167, 1)) < 0 || amount.Cmp(big.NewRat(168, 1)) > 0 {
			t.Fatalf("a share of %s", part)
		}
		whole.Add(whole, amount)
	}
	if whole.Cmp(big.NewRat(33450, 1)) != 0 {
		t.Fatalf("the shares add up to %s", whole.FloatString(2))
	}
	if len(parts) != 200 || len(sums) != 300 || sums[0] != "a000=111.5" || sums[299] != "a299=111.5" || !strings.HasPrefix(parts[0], "t000=") || !strings.HasPrefix(parts[199], "t199=") {
		t.Fatal(len(parts), len(sums))
	}
	if !strings.HasPrefix(flat(figures), "total=33450 assets=300 included=0/0 prices=300/0 idle=") || figures.ByKind[Depreciation] != "30000" || figures.ByKind[Operations] != "3150" {
		t.Fatal(flat(figures))
	}
}
