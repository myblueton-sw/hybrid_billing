package perfgate

import "testing"

func TestRateOverflow(t *testing.T) {
	f := standard(t)
	p := prof("cu", GateCatchUp)
	p.Axes.RowsDay = maxAxis
	f.approve(Change{Kind: ChangeProfile, Profile: p})
	l := limits(GateCatchUp)
	l[len(l)-1].Value = 11574075
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "cu", Hardware: "hw1", Limits: l}})
	f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b", Distribution: sum("d")}}, ann, eve)
	x := runOf(p, "1.0.0")
	x.Rows, x.DurationS = 1000000, 10000000
	x.Metrics[len(x.Metrics)-2].Value = 12000000
	if got := verdictOf(t, f, x); got != VerdictIncomplete {
		t.Errorf("overflowing rate %s", got)
	}
	big := runOf(p, "1.0.0")
	big.Rows, big.DurationS = maxRows, 86400000
	big.Metrics[len(big.Metrics)-2].Value = 12000000
	if got := verdictOf(t, f, big); got != VerdictPassed {
		t.Errorf("large exact rate %s", got)
	}
	if productLess(maxRows, daySeconds, maxRows, daySeconds) || !productLess(1, 1, 1, 2) || productLess(1, 2, 1, 1) || !productLess(1<<40, 1<<40, 1<<41, 1<<40) {
		t.Error("productLess")
	}
	if !productLess(1, 1<<62, 1<<62, 4) || productLess(1<<62, 4, 1, 1<<62) {
		t.Error("productLess high words")
	}
}

func TestBudgetApproverOverlap(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangeHardware, Hardware: hardware("hw1")})
	f.approve(Change{Kind: ChangeProfile, Profile: prof("p", GateE2E)})
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "hx", Identity: "r-x"}}, eve, dan)
	worker := Actor{ID: "r-x", Kind: KindSystem}
	first, err := f.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)
	f.must(err)
	f.approveBy(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "p", Hardware: "hw1", Limits: limits(GateE2E)}}, cat, dan)
	if again, _ := f.r.Run(first.Number); again.Verdict != VerdictSelfVerified {
		t.Errorf("budget approver on read %s", again.Verdict)
	}
	reason(t, "budget approver at record", second(f.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)), ReasonSelfVerification)
	g, _ := f.r.Gate(GateE2E, "1.0.0")
	if g.Status != StatusFailing {
		t.Errorf("self verified gate %+v", g)
	}
}

func TestSeparateFloors(t *testing.T) {
	ir := prof("p", GateInterruptResume)
	ir.Axes.RowsRun = runRows - 1
	cu := prof("p", GateCatchUp)
	cu.Axes.RowsDay = dayRows
	if ir.floor() || cu.floor() {
		t.Error("floors")
	}
	if !prof("p", GateInterruptResume).floor() || !prof("p", GateCatchUp).floor() {
		t.Error("fixtures")
	}
}

func TestCapsAndLabels(t *testing.T) {
	f := standard(t)
	p := f.gate("e1", GateE2E, true)
	q := f.gate("e2", GateE2E, true)
	for len(f.r.profiles["e1"].runs) < maxRuns {
		f.record(runOf(p, "1.0.0"))
	}
	for len(f.r.runs) < maxRunsTotal {
		f.r.runs[uint64(10000000+len(f.r.runs))] = &run{}
	}
	f.must(second(f.r.Record(runOf(p, "1.0.0"), runner)))
	expect(t, "growing profile", second(f.r.Record(runOf(q, "1.0.0"), runner)), ErrLimit)
	g := standard(t)
	e := g.gate("e1", GateE2E, true)
	for name, mutate := range map[string]func(*Run){
		VerdictOtherData:     func(x *Run) { x.Dataset = sum("x") },
		VerdictMissingMetric: func(x *Run) { x.Metrics = x.Metrics[1:] },
	} {
		x := runOf(e, "2.0.0")
		mutate(&x)
		g.record(x)
		if gv, _ := g.r.Gate(GateE2E, "2.0.0"); gv.Status != StatusFailing || gv.Profiles[0].Status != name {
			t.Errorf("%s gate %+v", name, gv)
		}
	}
	h := standard(t)
	h.approve(Change{Kind: ChangeProfile, Profile: prof("r", GateE2E)})
	withResume := append(limits(GateE2E), Limit{})
	copy(withResume[5:], withResume[4:])
	withResume[4] = Limit{MetricResume, 100}
	h.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "br", Profile: "r", Hardware: "hw1", Limits: withResume}})
	if got := verdictOf(t, h, runOf(prof("r", GateE2E), "1.0.0")); got != VerdictMissingMetric {
		t.Errorf("budgeted optional metric missing %s", got)
	}
}

func TestMutationEdges(t *testing.T) {
	f := standard(t)
	zero := Budget{ID: "b", Profile: "p", Hardware: "h", Limits: []Limit{{MetricCompletion, 0}}}
	if !zero.valid() {
		t.Error("zero limit refused")
	}
	p := f.gate("cu", GateCatchUp, true)
	exact := runOf(p, "1.0.0")
	exact.Rows, exact.DurationS = p.Axes.BacklogRows, 3600
	if got := verdictOf(t, f, exact); got != VerdictPassed {
		t.Errorf("exact backlog %s", got)
	}
	zeroMeasure := runOf(p, "1.0.0")
	zeroMeasure.Metrics[2].Value = 0
	zeroMeasure.Mismatches = 0
	f.must(second(f.r.Record(zeroMeasure, runner)))
	edge := runOf(p, "1.0.0")
	edge.Mismatches = maxRows
	if got := verdictOf(t, f, edge); got != VerdictIncorrect {
		t.Errorf("max mismatches %s", got)
	}
	g := standard(t)
	e := g.gate("e1", GateE2E, true)
	bad := runOf(e, "1.0.0")
	bad.Output = sum("wrong")
	g.record(bad)
	g.record(bad)
	g.record(runOf(e, "1.0.0"))
	if gv, _ := g.r.Gate(GateE2E, "1.0.0"); gv.Profiles[0].Failures != 2 || gv.Profiles[0].Status != VerdictPassed {
		t.Errorf("failures %+v", gv.Profiles)
	}
	g.gate("e3", GateE2E, true)
	g.gate("e0", GateE2E, true)
	g.gate("e2", GateE2E, true)
	gv, _ := g.r.Gate(GateE2E, "1.0.0")
	for index, want := range []string{"e0", "e1", "e2", "e3"} {
		if gv.Profiles[index].Profile != want {
			t.Errorf("order %+v", gv.Profiles)
		}
	}
}
