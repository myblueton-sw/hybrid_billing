package perfgate

import (
	"sync"
	"testing"
	"time"
)

func verdictOf(t testing.TB, f world, x Run) string {
	t.Helper()
	return f.record(x).Verdict
}

func TestVerdicts(t *testing.T) {
	f := standard(t)
	p := f.gate("e1", GateE2E, true)
	f.advance(time.Minute)
	good := runOf(p, "1.0.0")
	v := f.record(good)
	if v.Verdict != VerdictPassed || v.Harness != "h1" || v.Number != 1 || !v.At.Equal(start.Add(time.Minute)) {
		t.Fatalf("passed %+v", v)
	}
	cases := map[string]func(*Run){
		VerdictOtherData:     func(x *Run) { x.Dataset = sum("other") },
		VerdictIncomplete:    func(x *Run) { x.Rows-- },
		VerdictIncorrect:     func(x *Run) { x.Output = sum("wrong") },
		VerdictMissingMetric: func(x *Run) { x.Metrics = x.Metrics[1:] },
		VerdictOtherHardware: func(x *Run) { x.Hardware = "hw2" },
		VerdictOverBudget:    func(x *Run) { x.Metrics[4].Value = 501 },
	}
	for want, mutate := range cases {
		x := runOf(p, "1.0.0")
		mutate(&x)
		if got := verdictOf(t, f, x); got != want {
			t.Errorf("%s got %s", want, got)
		}
	}
	mismatch := runOf(p, "1.0.0")
	mismatch.Mismatches = 1
	if got := verdictOf(t, f, mismatch); got != VerdictIncorrect {
		t.Errorf("mismatch %s", got)
	}
	both := runOf(p, "1.0.0")
	both.Dataset, both.Rows, both.Output = sum("other"), 0, sum("wrong")
	if got := verdictOf(t, f, both); got != VerdictOtherData {
		t.Errorf("precedence %s", got)
	}
	short := runOf(p, "1.0.0")
	short.Rows, short.Output = 0, sum("wrong")
	if got := verdictOf(t, f, short); got != VerdictIncomplete {
		t.Errorf("incomplete first %s", got)
	}
	slow := runOf(p, "1.0.0")
	slow.Metrics[len(slow.Metrics)-2].Value = 11
	if got := verdictOf(t, f, slow); got != VerdictOverBudget {
		t.Errorf("throughput %s", got)
	}
	exact := runOf(p, "1.0.0")
	exact.Metrics[len(exact.Metrics)-2].Value = 12
	exact.Metrics[4].Value = 500
	if got := verdictOf(t, f, exact); got != VerdictPassed {
		t.Errorf("exact limits %s", got)
	}
	for index := range good.Metrics {
		x := runOf(p, "1.0.0")
		x.Metrics = append(append([]Measure{}, x.Metrics[:index]...), x.Metrics[index+1:]...)
		if got := verdictOf(t, f, x); got != VerdictMissingMetric {
			t.Errorf("metric %s missing gives %s", good.Metrics[index].Metric, got)
		}
	}
	q := f.gate("e2", GateE2E, false)
	if got := verdictOf(t, f, runOf(q, "1.0.0")); got != VerdictProvisional {
		t.Errorf("provisional %s", got)
	}
	f.approve(Change{Kind: ChangeProfile, Profile: prof("e3", GateE2E)})
	if got := verdictOf(t, f, runOf(prof("e3", GateE2E), "1.0.0")); got != VerdictUnbudgeted {
		t.Errorf("unbudgeted %s", got)
	}
	early := f.record(runOf(q, "1.0.0"))
	f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b-e2", Distribution: sum("d")}}, ann, eve)
	if again, _ := f.r.Run(early.Number); again.Verdict != VerdictPredatesBudget {
		t.Errorf("predates %s", again.Verdict)
	}
	if got := verdictOf(t, f, runOf(q, "1.0.0")); got != VerdictPassed {
		t.Errorf("after approval %s", got)
	}
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "b-e2-new", Profile: "e2", Hardware: "hw1", Limits: limits(GateE2E)}})
	if again, _ := f.r.Run(early.Number + 1); again.Verdict != VerdictProvisional {
		t.Errorf("replaced budget %s", again.Verdict)
	}
	f.must(f.r.RevokeHarness("h1", ann))
	if again, _ := f.r.Run(1); again.Verdict != VerdictRevoked {
		t.Errorf("revoked %s", again.Verdict)
	}
	reason(t, "revoked record", second(f.r.Record(good, runner)), ReasonNotHarness)
}

func TestGateSpecificRuns(t *testing.T) {
	f := standard(t)
	for _, gate := range Gates {
		p := f.gate("g-"+gate, gate, true)
		if got := verdictOf(t, f, runOf(p, "1.0.0")); got != VerdictPassed {
			t.Errorf("%s fixture %s", gate, got)
		}
	}
	ir := prof("g-"+GateInterruptResume, GateInterruptResume)
	x := runOf(ir, "1.0.0")
	x.Interrupted = false
	if got := verdictOf(t, f, x); got != VerdictIncomplete {
		t.Errorf("not interrupted %s", got)
	}
	x = runOf(ir, "1.0.0")
	x.Rows--
	if got := verdictOf(t, f, x); got != VerdictIncomplete {
		t.Errorf("ir rows %s", got)
	}
	mc := prof("g-"+GateMonthlyClose, GateMonthlyClose)
	x = runOf(mc, "1.0.0")
	x.Rows--
	if got := verdictOf(t, f, x); got != VerdictIncomplete {
		t.Errorf("monthly rows %s", got)
	}
	di := prof("g-"+GateDailyInflow, GateDailyInflow)
	for name, mutate := range map[string]func(*Run){
		"rows":     func(x *Run) { x.Rows-- },
		"duration": func(x *Run) { x.DurationS = 86399 },
		"rate":     func(x *Run) { x.Rows, x.DurationS = 5000000, 5*86400 },
	} {
		x := runOf(di, "1.0.0")
		mutate(&x)
		if got := verdictOf(t, f, x); got != VerdictIncomplete {
			t.Errorf("daily %s %s", name, got)
		}
	}
	long := runOf(di, "1.0.0")
	long.Rows, long.DurationS = 2000002, 2*86400
	if got := verdictOf(t, f, long); got != VerdictPassed {
		t.Errorf("two days at rate %s", got)
	}
	cu := prof("g-"+GateCatchUp, GateCatchUp)
	for name, mutate := range map[string]func(*Run){
		"backlog": func(x *Run) { x.Rows = 999999 },
		"rate":    func(x *Run) { x.Rows, x.DurationS = 1000000, 86400 },
		"zero":    func(x *Run) { x.DurationS = 0 },
	} {
		x := runOf(cu, "1.0.0")
		mutate(&x)
		if got := verdictOf(t, f, x); got != VerdictIncomplete {
			t.Errorf("catch up %s %s", name, got)
		}
	}
	fastCatchUp := runOf(cu, "1.0.0")
	fastCatchUp.Rows, fastCatchUp.DurationS = 1000001, 3600
	if got := verdictOf(t, f, fastCatchUp); got != VerdictPassed {
		t.Errorf("short catch up %s", got)
	}
	e := prof("g-"+GateE2E, GateE2E)
	noResume := runOf(e, "1.0.0")
	if len(noResume.Metrics) != 8 {
		t.Fatal("e2e has no resume metric")
	}
	if got := verdictOf(t, f, noResume); got != VerdictPassed {
		t.Errorf("resume optional %s", got)
	}
}

func TestSelfVerification(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangeHardware, Hardware: hardware("hw1")})
	cases := []struct {
		name    string
		harness [2]Actor
	}{
		{"profile proposer", [2]Actor{ann, cat}},
		{"profile approver", [2]Actor{cat, bob}},
	}
	f.approve(Change{Kind: ChangeProfile, Profile: prof("p", GateE2E)})
	for index, c := range cases {
		id := "h" + string(rune('a'+index))
		f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: id, Identity: "r-" + id}}, c.harness[0], c.harness[1])
		reason(t, c.name, second(f.r.Record(runOf(prof("p", GateE2E), "1.0.0"), Actor{ID: "r-" + id, Kind: KindSystem})), ReasonSelfVerification)
	}
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "hx", Identity: "r-x"}}, cat, dan)
	worker := Actor{ID: "r-x", Kind: KindSystem}
	f.approveBy(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "p", Hardware: "hw1", Limits: limits(GateE2E)}}, eve, ann)
	first, err := f.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)
	f.must(err)
	if first.Verdict != VerdictProvisional {
		t.Fatalf("clean %s", first.Verdict)
	}
	f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b", Distribution: sum("d")}}, dan, bob)
	if again, _ := f.r.Run(first.Number); again.Verdict != VerdictSelfVerified {
		t.Errorf("approval proposer overlap %s", again.Verdict)
	}
	reason(t, "record after overlap", second(f.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)), ReasonSelfVerification)
	for _, people := range [][2]Actor{{cat, ann}, {ann, cat}} {
		g := setup(t)
		g.approve(Change{Kind: ChangeHardware, Hardware: hardware("hw1")})
		g.approve(Change{Kind: ChangeProfile, Profile: prof("p", GateE2E)})
		g.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "hx", Identity: "r-x"}}, eve, dan)
		g.approveBy(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "p", Hardware: "hw1", Limits: limits(GateE2E)}}, people[0], people[1])
		g.must(second(g.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)))
		g.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b", Distribution: sum("d")}}, bob, dan)
		reason(t, "approval approver", second(g.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)), ReasonSelfVerification)
	}
	h := setup(t)
	h.approve(Change{Kind: ChangeHardware, Hardware: hardware("hw1")})
	h.approve(Change{Kind: ChangeProfile, Profile: prof("p", GateE2E)})
	h.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "hx", Identity: "r-x"}}, eve, dan)
	h.approveBy(Change{Kind: ChangeBudget, Budget: Budget{ID: "b", Profile: "p", Hardware: "hw1", Limits: limits(GateE2E)}}, dan, cat)
	reason(t, "budget proposer", second(h.r.Record(runOf(prof("p", GateE2E), "1.0.0"), worker)), ReasonSelfVerification)
}

func TestRecordRules(t *testing.T) {
	f := standard(t)
	p := f.gate("e1", GateE2E, true)
	x := runOf(p, "1.0.0")
	reason(t, "assistant", second(f.r.Record(x, Actor{ID: runner.ID, Kind: KindAssistant})), ReasonAINotAllowed)
	reason(t, "human", second(f.r.Record(x, Actor{ID: runner.ID, Kind: KindHuman})), ReasonNotHarness)
	reason(t, "unbound", second(f.r.Record(x, Actor{ID: "runner-9", Kind: KindSystem})), ReasonNotHarness)
	expect(t, "actor", second(f.r.Record(x, Actor{Kind: KindSystem})), ErrInvalid)
	forms := map[string]func(*Run){
		"profile": func(x *Run) { x.Profile = "" }, "hardware": func(x *Run) { x.Hardware = "" }, "build": func(x *Run) { x.Build = "1" },
		"dataset": func(x *Run) { x.Dataset = "" }, "output": func(x *Run) { x.Output = "" }, "rows": func(x *Run) { x.Rows = -1 },
		"many rows": func(x *Run) { x.Rows = maxRows + 1 }, "duration": func(x *Run) { x.DurationS = -1 }, "long": func(x *Run) { x.DurationS = maxDuration + 1 },
		"mismatch": func(x *Run) { x.Mismatches = -1 }, "many mismatches": func(x *Run) { x.Mismatches = maxRows + 1 },
		"metric": func(x *Run) { x.Metrics[0].Metric = "cost" }, "value": func(x *Run) { x.Metrics[0].Value = -1 },
		"huge": func(x *Run) { x.Metrics[0].Value = maxValue + 1 }, "order": func(x *Run) { x.Metrics[0], x.Metrics[1] = x.Metrics[1], x.Metrics[0] },
		"repeat": func(x *Run) { x.Metrics[1] = x.Metrics[0] },
		"too many": func(x *Run) {
			x.Metrics = nil
			for _, m := range Metrics {
				x.Metrics = append(x.Metrics, Measure{m, 1})
			}
			x.Metrics = append(x.Metrics, Measure{"zz", 1})
		},
	}
	for name, mutate := range forms {
		y := runOf(p, "1.0.0")
		mutate(&y)
		expect(t, name, second(f.r.Record(y, runner)), ErrInvalid)
	}
	edge := runOf(p, "1.0.0")
	edge.Rows, edge.DurationS, edge.Mismatches = maxRows, maxDuration, 0
	edge.Metrics[0].Value = maxValue
	f.must(second(f.r.Record(edge, runner)))
	missing := runOf(p, "1.0.0")
	missing.Profile = "nope"
	expect(t, "profile missing", second(f.r.Record(missing, runner)), ErrNotFound)
	missing = runOf(p, "1.0.0")
	missing.Hardware = "hw9"
	expect(t, "hardware missing", second(f.r.Record(missing, runner)), ErrNotFound)
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Profile: "e1"}})
	reason(t, "retired", second(f.r.Record(x, runner)), ReasonRetired)
	reason(t, "revoke system", f.r.RevokeHarness("h1", runner), ReasonNotHuman)
	expect(t, "revoke form", f.r.RevokeHarness("", ann), ErrInvalid)
	expect(t, "revoke missing", f.r.RevokeHarness("nope", ann), ErrNotFound)
}

func TestGatesAndReadiness(t *testing.T) {
	f := standard(t)
	r, err := f.r.Readiness("1.0.0")
	if err != nil || r.Ready || len(r.Gates) != 5 {
		t.Fatalf("empty %+v %v", r, err)
	}
	for _, g := range r.Gates {
		if g.Status != StatusNotRun || len(g.Profiles) != 0 || g.Retired == nil {
			t.Errorf("empty gate %+v", g)
		}
	}
	profiles := map[string]Profile{}
	for _, gate := range Gates {
		profiles[gate] = f.gate("g-"+gate, gate, true)
	}
	for _, gate := range Gates[:4] {
		f.record(runOf(profiles[gate], "1.0.0"))
	}
	if r, _ := f.r.Readiness("1.0.0"); r.Ready || r.Gates[4].Status != StatusNotRun || r.Gates[0].Status != StatusPassed {
		t.Errorf("one not run %+v", r)
	}
	f.record(runOf(profiles[GateCatchUp], "1.0.0"))
	if r, _ := f.r.Readiness("1.0.0"); !r.Ready {
		t.Errorf("all passed %+v", r)
	}
	if r, _ := f.r.Readiness("1.0.1"); r.Ready || r.Gates[0].Status != StatusNotRun {
		t.Errorf("other build %+v", r)
	}
	secondProfile := f.gate("g2", GateE2E, true)
	if g, _ := f.r.Gate(GateE2E, "1.0.0"); g.Status != StatusNotRun || len(g.Profiles) != 2 {
		t.Errorf("unrun second profile %+v", g)
	}
	bad := runOf(secondProfile, "1.0.0")
	bad.Output = sum("wrong")
	f.record(bad)
	if g, _ := f.r.Gate(GateE2E, "1.0.0"); g.Status != StatusFailing {
		t.Errorf("failing %+v", g)
	}
	third := f.gate("g3", GateE2E, false)
	f.record(runOf(third, "1.0.0"))
	if g, _ := f.r.Gate(GateE2E, "1.0.0"); g.Status != StatusFailing {
		t.Errorf("failing beats provisional %+v", g)
	}
	f.record(runOf(secondProfile, "1.0.0"))
	if g, _ := f.r.Gate(GateE2E, "1.0.0"); g.Status != StatusProvisional || g.Profiles[1].Failures != 1 {
		t.Errorf("provisional %+v", g)
	}
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Profile: "g3"}})
	if g, _ := f.r.Gate(GateE2E, "1.0.0"); g.Status != StatusPassed || len(g.Retired) != 1 || g.Retired[0].Status != VerdictProvisional {
		t.Errorf("retired %+v", g)
	}
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Profile: "g2"}})
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Profile: "g-" + GateE2E}})
	if g, _ := f.r.Gate(GateE2E, "1.0.0"); g.Status != StatusNotRun {
		t.Errorf("all retired %+v", g)
	}
	for _, args := range [][2]string{{"soak", "1.0.0"}, {GateE2E, "x"}} {
		expect(t, "gate form", second(f.r.Gate(args[0], args[1])), ErrInvalid)
	}
	expect(t, "readiness form", second(f.r.Readiness("x")), ErrInvalid)
	f.approve(Change{Kind: ChangeProfile, Profile: prof("u1", GateMonthlyClose)})
	f.record(runOf(prof("u1", GateMonthlyClose), "1.0.0"))
	if g, _ := f.r.Gate(GateMonthlyClose, "1.0.0"); g.Status != StatusProvisional {
		t.Errorf("unbudgeted counts provisional %+v", g)
	}
}

func TestRunsReadsAndCaps(t *testing.T) {
	f := standard(t)
	p := f.gate("e1", GateE2E, true)
	for index := 0; index < 5; index++ {
		f.record(runOf(p, "1.0.0"))
	}
	page, err := f.r.Runs("e1", 2, 2)
	if err != nil || len(page) != 2 || page[0].Number != 3 {
		t.Errorf("page %+v %v", page, err)
	}
	page[0].Run.Metrics[0].Value = 1
	if again, _ := f.r.Run(3); again.Run.Metrics[0].Value == 1 {
		t.Error("run read shares memory")
	}
	for _, limit := range []int{0, maxPage + 1} {
		expect(t, "limit", second(f.r.Runs("e1", 0, limit)), ErrInvalid)
	}
	expect(t, "runs missing", second(f.r.Runs("nope", 0, 1)), ErrNotFound)
	expect(t, "run missing", second(f.r.Run(99)), ErrNotFound)
	f.must(second(f.r.Runs("e1", 0, maxPage)))
	stored := f.r.profiles["e1"]
	for len(stored.runs) < maxRuns {
		f.record(runOf(p, "1.0.0"))
	}
	f.record(runOf(p, "1.0.0"))
	expect(t, "evicted", second(f.r.Run(1)), ErrNotFound)
	if len(stored.runs) != maxRuns || len(f.r.runs) != maxRuns {
		t.Errorf("kept %d %d", len(stored.runs), len(f.r.runs))
	}
	g := standard(t)
	q := g.gate("e1", GateE2E, true)
	for len(g.r.runs) < maxRunsTotal-1 {
		g.r.runs[uint64(1000000+len(g.r.runs))] = &run{}
	}
	g.record(runOf(q, "1.0.0"))
	expect(t, "total", second(g.r.Record(runOf(q, "1.0.0"), runner)), ErrLimit)
	caps := []struct {
		name   string
		fill   func(f world, n int)
		cap    int
		change Change
	}{
		{"profiles", func(f world, n int) {
			for len(f.r.profiles) < n {
				f.r.profiles["x"+itoa(len(f.r.profiles))] = &profile{}
			}
		}, maxProfiles, Change{Kind: ChangeProfile, Profile: prof("new", GateE2E)}},
		{"hardware", func(f world, n int) {
			for len(f.r.hardware) < n {
				f.r.hardware["x"+itoa(len(f.r.hardware))] = &Hardware{}
			}
		}, maxHardware, Change{Kind: ChangeHardware, Hardware: hardware("new")}},
		{"budgets", func(f world, n int) {
			for len(f.r.budgets) < n {
				f.r.budgets["x"+itoa(len(f.r.budgets))] = &budget{}
			}
		}, maxBudgets, Change{Kind: ChangeBudget, Budget: Budget{ID: "new", Profile: "e1", Hardware: "hw1", Limits: limits(GateE2E)}}},
		{"harnesses", func(f world, n int) {
			for len(f.r.harnesses) < n {
				f.r.harnesses["x"+itoa(len(f.r.harnesses))] = &harness{}
			}
		}, maxHarnesses, Change{Kind: ChangeHarness, Harness: Harness{ID: "new", Identity: "new"}}},
	}
	for _, c := range caps {
		h := standard(t)
		h.approve(Change{Kind: ChangeProfile, Profile: prof("e1", GateE2E)})
		c.fill(h, c.cap-1)
		if err := h.try(c.change); err != nil {
			t.Errorf("%s exact fit %v", c.name, err)
		}
		k := standard(t)
		k.approve(Change{Kind: ChangeProfile, Profile: prof("e1", GateE2E)})
		c.fill(k, c.cap)
		expect(t, c.name, k.try(c.change), ErrLimit)
	}
}

func TestPendingAndDone(t *testing.T) {
	f := setup(t)
	people := []Actor{ann, bob, cat, dan}
	for _, who := range people {
		for index := 0; index < maxMine; index++ {
			f.must(second(f.r.Propose(Change{Kind: ChangeHardware, Hardware: hardware(who.ID + itoa(index))}, who)))
		}
		expect(t, "mine", second(f.r.Propose(Change{Kind: ChangeHardware, Hardware: hardware(who.ID + "-over")}, who)), ErrLimit)
	}
	expect(t, "total", second(f.r.Propose(Change{Kind: ChangeHardware, Hardware: hardware("e")}, eve)), ErrLimit)
	g := setup(t)
	first := g.approve(Change{Kind: ChangeHardware, Hardware: hardware("a")})
	for index := 2; index < maxDone; index++ {
		g.r.done = append(g.r.done, uint64(100000+index))
	}
	kept := g.approve(Change{Kind: ChangeHardware, Hardware: hardware("b")})
	if _, err := g.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	g.approve(Change{Kind: ChangeHardware, Hardware: hardware("c")})
	expect(t, "pruned", second(g.r.Proposal(first.Number)), ErrNotFound)
	if _, err := g.r.Proposal(kept.Number); err != nil || len(g.r.done) != maxDone {
		t.Errorf("prune one %v %d", err, len(g.r.done))
	}
}

func TestConcurrency(t *testing.T) {
	f := standard(t)
	p := f.gate("e1", GateE2E, true)
	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for n := 0; n < 20; n++ {
				f.r.Record(runOf(p, "1.0.0"), runner)
				f.r.Readiness("1.0.0")
				f.r.Gate(GateE2E, "1.0.0")
				f.r.Runs("e1", 0, 5)
				if q, err := f.r.Propose(Change{Kind: ChangeProfile, Profile: prof("c"+itoa(index)+"-"+itoa(n), GateE2E)}, ann); err == nil {
					f.r.Approve(q.Number, bob)
				}
			}
		}(index)
	}
	wg.Wait()
	if len(f.r.runs) != 160 || len(f.r.profiles) != 161 {
		t.Errorf("counts %d %d", len(f.r.runs), len(f.r.profiles))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}

func TestValidators(t *testing.T) {
	long := "a"
	for len(long) < 128 {
		long += "b"
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", "a.b_c:d-e", long} {
		if !validID(id) {
			t.Errorf("id %q refused", id)
		}
	}
	for _, id := range []string{"", "!a", "a b", "a@", "a[", "a`", "a{", "a/", long + "c"} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	if !validHex(sum("a")) || validHex("") || validHex("g"+sum("a")[1:]) || validHex(sum("a")[:63]+"G") || validHex(sum("a")+"0") {
		t.Error("hex")
	}
	for _, v := range []string{"0.0.0", "1.20.3", "999999999.0.1"} {
		if !semver(v) {
			t.Errorf("semver %s refused", v)
		}
	}
	for _, v := range []string{"", "1.2", "1.2.3.4", "01.2.3", "1.2.x", "1..3", "1000000000.0.0", "1.2.:"} {
		if semver(v) {
			t.Errorf("semver %s accepted", v)
		}
	}
}
