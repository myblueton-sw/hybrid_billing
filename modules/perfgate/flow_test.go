package perfgate

import (
	"errors"
	"testing"
)

func TestConstantsAndForms(t *testing.T) {
	got := []int64{maxProfiles, maxHardware, maxBudgets, maxHarnesses, maxRuns, maxRunsTotal, maxPending, maxMine, maxDone, maxPage, maxAxis, maxSpec, maxValue,
		dayRows, monthRows, runRows, daySeconds, maxRows, maxDuration, maxIDBytes}
	want := []int64{1000, 100, 10000, 100, 1000, 100000, 1000, 250, 1000, 1000, 1000000000000, 1000000000, 1000000000000000000, 1000000, 30000000, 100000, 86400,
		1000000000000000, 1000000000, 128}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d = %d", index, got[index])
		}
	}
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
	for _, gate := range Gates {
		if p := prof("p", gate); !p.valid() || !p.floor() {
			t.Errorf("fixture %s", gate)
		}
	}
	profiles := map[string]func(*Profile){
		"id": func(p *Profile) { p.ID = "" }, "gate": func(p *Profile) { p.Gate = "soak" }, "dataset": func(p *Profile) { p.Dataset = "" },
		"expected": func(p *Profile) { p.Expected = "x" }, "negative": func(p *Profile) { p.Axes.FanOut = -1 }, "huge": func(p *Profile) { p.Axes.Customers = maxAxis + 1 },
		"skew": func(p *Profile) { p.Axes.SkewPermille = 1001 },
	}
	for name, mutate := range profiles {
		p := prof("p", GateE2E)
		mutate(&p)
		if p.valid() {
			t.Errorf("profile %s accepted", name)
		}
	}
	edge := prof("p", GateE2E)
	edge.Axes.Customers, edge.Axes.SkewPermille = maxAxis, 1000
	for index, value := range edge.Axes.list() {
		if index != 3 && index != 6 && value > maxAxis {
			t.Error("axis")
		}
	}
	if !edge.valid() {
		t.Error("axis edges refused")
	}
	for _, mutate := range []func(*Axes){
		func(a *Axes) { a.RowsRun = -1 }, func(a *Axes) { a.RowsDay = -1 }, func(a *Axes) { a.RowsMonth = -1 }, func(a *Axes) { a.Accounts = -1 },
		func(a *Axes) { a.SharedPools = -1 }, func(a *Axes) { a.ConcurrentCloses = -1 }, func(a *Axes) { a.PeakWindowMinutes = -1 },
		func(a *Axes) { a.RetentionMonths = -1 }, func(a *Axes) { a.FailureInjections = -1 }, func(a *Axes) { a.BacklogRows = -1 },
		func(a *Axes) { a.SkewPermille = -1 }, func(a *Axes) { a.Customers = -1 },
	} {
		a := axes(GateE2E)
		mutate(&a)
		if a.valid() {
			t.Error("negative axis accepted")
		}
	}
	floors := map[string]func(*Axes){
		GateE2E:             func(a *Axes) { a.RowsRun-- },
		GateDailyInflow:     func(a *Axes) { a.RowsDay-- },
		GateMonthlyClose:    func(a *Axes) { a.RowsMonth-- },
		GateInterruptResume: func(a *Axes) { a.FailureInjections = 0 },
		GateCatchUp:         func(a *Axes) { a.BacklogRows-- },
	}
	for gate, mutate := range floors {
		p := prof("p", gate)
		mutate(&p.Axes)
		if p.floor() {
			t.Errorf("%s floor", gate)
		}
	}
	ir := prof("p", GateInterruptResume)
	ir.Axes.RowsRun--
	cu := prof("p", GateCatchUp)
	cu.Axes.RowsDay = dayRows
	if ir.floor() || cu.floor() {
		t.Error("second floors")
	}
	if !hardware("h").valid() {
		t.Fatal("hardware fixture")
	}
	hw := map[string]func(*Hardware){
		"id": func(h *Hardware) { h.ID = "" }, "cpus": func(h *Hardware) { h.CPUs = 0 }, "memory": func(h *Hardware) { h.MemoryMiB = maxSpec + 1 },
		"disk": func(h *Hardware) { h.DiskGiB = 0 }, "storage": func(h *Hardware) { h.Storage = "tape" }, "spec": func(h *Hardware) { h.Spec = "" },
	}
	for name, mutate := range hw {
		h := hardware("h")
		mutate(&h)
		if h.valid() {
			t.Errorf("hardware %s accepted", name)
		}
	}
	h := hardware("h")
	h.CPUs, h.MemoryMiB, h.DiskGiB = 1, maxSpec, 1
	if !h.valid() {
		t.Error("hardware edges refused")
	}
	b := Budget{ID: "b", Profile: "p", Hardware: "h", Limits: limits(GateE2E)}
	if !b.valid() {
		t.Fatal("budget fixture")
	}
	budgets := map[string]func(*Budget){
		"id": func(b *Budget) { b.ID = "" }, "profile": func(b *Budget) { b.Profile = "" }, "hardware": func(b *Budget) { b.Hardware = "" },
		"empty": func(b *Budget) { b.Limits = nil }, "metric": func(b *Budget) { b.Limits[0].Metric = "cost" },
		"negative": func(b *Budget) { b.Limits[0].Value = -1 }, "huge": func(b *Budget) { b.Limits[0].Value = maxValue + 1 },
		"order": func(b *Budget) { b.Limits[0], b.Limits[1] = b.Limits[1], b.Limits[0] }, "repeat": func(b *Budget) { b.Limits[1] = b.Limits[0] },
		"many": func(b *Budget) {
			b.Limits = nil
			for _, m := range Metrics {
				b.Limits = append(b.Limits, Limit{m, 1})
			}
			b.Limits = append(b.Limits, Limit{"zz", 1})
		},
	}
	for name, mutate := range budgets {
		x := Budget{ID: "b", Profile: "p", Hardware: "h", Limits: limits(GateE2E)}
		mutate(&x)
		if x.valid() {
			t.Errorf("budget %s accepted", name)
		}
	}
	all := Budget{ID: "b", Profile: "p", Hardware: "h"}
	for _, m := range Metrics {
		all.Limits = append(all.Limits, Limit{m, maxValue})
	}
	if !all.valid() {
		t.Error("all metrics refused")
	}
	for _, c := range []Change{{Kind: "other"}, {Kind: ChangeApproval, Approval: Approval{Budget: "b"}}, {Kind: ChangeApproval, Approval: Approval{Distribution: sum("d")}},
		{Kind: ChangeHarness, Harness: Harness{ID: "h"}}, {Kind: ChangeHarness, Harness: Harness{Identity: "x"}}, {Kind: ChangeRetire}, {Kind: ChangeHardware}, {Kind: ChangeProfile}, {Kind: ChangeBudget}} {
		if _, ok := clean(c); ok {
			t.Errorf("%s accepted", c.Kind)
		}
	}
}

func TestProposals(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangeHardware, Hardware: hardware("hw1")}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, runner)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{Kind: KindHuman})), ErrInvalid)
	expect(t, "form", second(f.r.Propose(Change{Kind: ChangeHardware}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if again, _ := f.r.Propose(c, ann); again.Number != p.Number {
		t.Error("repeat")
	}
	q, err := f.r.Propose(c, cat)
	f.must(err)
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve system", second(f.r.Approve(p.Number, runner)), ReasonNotHuman)
	expect(t, "approve missing", second(f.r.Approve(99, bob)), ErrNotFound)
	f.must(second(f.r.Approve(p.Number, bob)))
	if again, err := f.r.Approve(p.Number, bob); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "approve applied", second(f.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "recheck", second(f.r.Approve(q.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(f.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(f.r.Withdraw(q.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := f.r.Withdraw(q.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(f.r.Proposal(q.Number)), ErrNotFound)
	b, err := f.r.Propose(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "p", Hardware: "hw1", Limits: limits(GateE2E)}}, ann)
	expect(t, "budget lookup", err, ErrNotFound)
	_ = b
	f.approve(Change{Kind: ChangeProfile, Profile: prof("p", GateE2E)})
	bp, err := f.r.Propose(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "p", Hardware: "hw1", Limits: limits(GateE2E)}}, ann)
	f.must(err)
	bp.Change.Budget.Limits[0].Value = 1
	if got, _ := f.r.Proposal(bp.Number); got.Change.Budget.Limits[0].Value != 3600000 {
		t.Error("proposal shares memory")
	}
}

func TestRegistrationRules(t *testing.T) {
	f := standard(t)
	expect(t, "hardware repeat", f.try(Change{Kind: ChangeHardware, Hardware: hardware("hw1")}), ErrConflict)
	for _, gate := range Gates {
		p := prof("low-"+gate, gate)
		switch gate {
		case GateE2E:
			p.Axes.RowsRun = runRows - 1
		case GateDailyInflow:
			p.Axes.RowsDay = dayRows
		case GateMonthlyClose:
			p.Axes.RowsMonth = monthRows - 1
		case GateInterruptResume:
			p.Axes.FailureInjections = 0
		case GateCatchUp:
			p.Axes.BacklogRows = dayRows - 1
		}
		reason(t, "below p0 "+gate, f.try(Change{Kind: ChangeProfile, Profile: p}), ReasonBelowP0)
	}
	f.approve(Change{Kind: ChangeProfile, Profile: prof("p1", GateInterruptResume)})
	expect(t, "profile repeat", f.try(Change{Kind: ChangeProfile, Profile: prof("p1", GateE2E)}), ErrConflict)
	reason(t, "resume required", f.try(Change{Kind: ChangeBudget, Budget: Budget{ID: "b1", Profile: "p1", Hardware: "hw1", Limits: limits(GateE2E)}}), ReasonMissingMetric)
	expect(t, "budget hardware", f.try(Change{Kind: ChangeBudget, Budget: Budget{ID: "b1", Profile: "p1", Hardware: "hw9", Limits: limits(GateInterruptResume)}}), ErrNotFound)
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "b1", Profile: "p1", Hardware: "hw1", Limits: limits(GateInterruptResume)}})
	expect(t, "budget repeat", f.try(Change{Kind: ChangeBudget, Budget: Budget{ID: "b1", Profile: "p1", Hardware: "hw1", Limits: limits(GateInterruptResume)}}), ErrConflict)
	for _, m := range []string{MetricCompletion, MetricMaxRSS, MetricP95, MetricP99, MetricThroughput} {
		l := []Limit{}
		for _, x := range limits(GateE2E) {
			if x.Metric != m {
				l = append(l, x)
			}
		}
		f.approve(Change{Kind: ChangeProfile, Profile: prof("q-"+m, GateE2E)})
		reason(t, "missing "+m, f.try(Change{Kind: ChangeBudget, Budget: Budget{ID: "bq-" + m, Profile: "q-" + m, Hardware: "hw1", Limits: l}}), ReasonMissingMetric)
	}
	f.approve(Change{Kind: ChangeProfile, Profile: prof("d1", GateDailyInflow)})
	slow := limits(GateDailyInflow)
	slow[len(slow)-1].Value = 11
	reason(t, "rate floor", f.try(Change{Kind: ChangeBudget, Budget: Budget{ID: "bd", Profile: "d1", Hardware: "hw1", Limits: slow}}), ReasonBelowP0)
	slow[len(slow)-1].Value = 12
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "bd", Profile: "d1", Hardware: "hw1", Limits: slow}})
	big := prof("c1", GateCatchUp)
	big.Axes.RowsDay = 8640001
	f.approve(Change{Kind: ChangeProfile, Profile: big})
	fast := limits(GateCatchUp)
	fast[len(fast)-1].Value = 100
	reason(t, "derived rate", f.try(Change{Kind: ChangeBudget, Budget: Budget{ID: "bc", Profile: "c1", Hardware: "hw1", Limits: fast}}), ReasonBelowP0)
	fast[len(fast)-1].Value = 101
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "bc", Profile: "c1", Hardware: "hw1", Limits: fast}})
	reason(t, "same approver", second(f.r.Approve(must(f.r.Propose(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b1", Distribution: sum("d")}}, cat)).Number, bob)), ReasonSamePeople)
	reason(t, "same proposer", second(f.r.Approve(must(f.r.Propose(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b1", Distribution: sum("d2")}}, cat)).Number, ann)), ReasonSamePeople)
	expect(t, "approval missing", f.try(Change{Kind: ChangeApproval, Approval: Approval{Budget: "nope", Distribution: sum("d")}}), ErrNotFound)
	f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b1", Distribution: sum("d")}}, cat, eve)
	v, err := f.r.Budget("b1")
	if err != nil || v.State != BudgetApproved || v.ApprovalBy != "cat" || v.ApprovedBy != "eve" || v.Distribution != sum("d") || !v.ApprovedAt.Equal(start) {
		t.Errorf("approved %+v %v", v, err)
	}
	expect(t, "re-approve", second(f.r.Propose(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b1", Distribution: sum("d3")}}, dan)), ErrConflict)
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "b2", Profile: "p1", Hardware: "hw2", Limits: limits(GateInterruptResume)}})
	if active, _ := f.r.ActiveBudget("p1"); active.Budget.ID != "b2" || active.State != BudgetProposed {
		t.Errorf("active %+v", active)
	}
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "b3", Profile: "p1", Hardware: "hw2", Limits: limits(GateInterruptResume)}})
	reason(t, "replaced", second(f.r.Propose(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b2", Distribution: sum("d")}}, dan)), ReasonNotActive)
	expect(t, "harness repeat", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: "x"}}), ErrConflict)
	reason(t, "identity", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: runner.ID}}), ReasonIdentityTaken)
	expect(t, "retire missing", f.try(Change{Kind: ChangeRetire, Retire: Retire{Profile: "nope"}}), ErrNotFound)
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Profile: "d1"}})
	reason(t, "retire again", f.try(Change{Kind: ChangeRetire, Retire: Retire{Profile: "d1"}}), ReasonRetired)
	if _, retired, _ := f.r.Profile("d1"); !retired {
		t.Error("retired flag")
	}
	if _, _, err := f.r.Profile("nope"); !errors.Is(err, ErrNotFound) {
		t.Error("profile missing")
	}
	expect(t, "hardware missing", second(f.r.Hardware("nope")), ErrNotFound)
	if h, err := f.r.Hardware("hw1"); err != nil || h != hardware("hw1") {
		t.Error("hardware read")
	}
	expect(t, "budget missing", second(f.r.Budget("nope")), ErrNotFound)
	expect(t, "active missing", second(f.r.ActiveBudget("nope")), ErrNotFound)
}

func must(p ProposalView, err error) ProposalView {
	if err != nil {
		panic(err)
	}
	return p
}
