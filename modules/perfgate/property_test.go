package perfgate

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

type mBudget struct {
	id, hardware string
	approved     bool
	byEve        bool
	event        int
	limits       map[string]int64
}

type mRun struct {
	profile, hardware, build string
	rows, duration           int64
	data, output             bool
	interrupted, revoked     bool
	third                    bool
	metrics                  map[string]int64
	event                    int
}

type mProfile struct {
	gate    string
	axes    Axes
	budget  *mBudget
	retired bool
	runs    []int
}

type gateModel struct {
	profiles map[string]*mProfile
	runs     []*mRun
	events   int
}

func (m *gateModel) short(p *mProfile, x *mRun) bool {
	rate := func() bool {
		if x.duration == 0 {
			return true
		}
		left := new(big.Int).Mul(big.NewInt(x.rows), big.NewInt(86400))
		right := new(big.Int).Mul(big.NewInt(p.axes.RowsDay), big.NewInt(x.duration))
		return left.Cmp(right) < 0
	}
	switch p.gate {
	case GateE2E:
		return x.rows < p.axes.RowsRun
	case GateInterruptResume:
		return x.rows < p.axes.RowsRun || !x.interrupted
	case GateMonthlyClose:
		return x.rows < p.axes.RowsMonth
	case GateDailyInflow:
		return x.rows < p.axes.RowsDay || x.duration < 86400 || rate()
	}
	return x.rows < p.axes.BacklogRows || rate()
}

func (m *gateModel) verdict(x *mRun) string {
	p := m.profiles[x.profile]
	b := p.budget
	switch {
	case !x.data:
		return VerdictOtherData
	case m.short(p, x):
		return VerdictIncomplete
	case !x.output:
		return VerdictIncorrect
	case x.revoked:
		return VerdictRevoked
	case x.third && b != nil && b.byEve:
		return VerdictSelfVerified
	}
	for _, metric := range Metrics {
		if _, ok := x.metrics[metric]; !ok && (metric != MetricResume || p.gate == GateInterruptResume) {
			return VerdictMissingMetric
		}
	}
	switch {
	case b == nil:
		return VerdictUnbudgeted
	case b.hardware != x.hardware:
		return VerdictOtherHardware
	}
	for metric := range b.limits {
		if _, ok := x.metrics[metric]; !ok {
			return VerdictMissingMetric
		}
	}
	for metric, limit := range b.limits {
		value := x.metrics[metric]
		if metric == MetricThroughput && value < limit || metric != MetricThroughput && value > limit {
			return VerdictOverBudget
		}
	}
	switch {
	case !b.approved:
		return VerdictProvisional
	case x.event < b.event:
		return VerdictPredatesBudget
	}
	return VerdictPassed
}

func (m *gateModel) gate(gate, build string) string {
	statuses := []string{}
	for _, p := range m.profiles {
		if p.gate != gate || p.retired {
			continue
		}
		status := StatusNotRun
		for _, index := range p.runs {
			if m.runs[index].build == build {
				status = m.verdict(m.runs[index])
			}
		}
		statuses = append(statuses, status)
	}
	bad := map[string]bool{VerdictOtherData: true, VerdictIncomplete: true, VerdictIncorrect: true, VerdictRevoked: true, VerdictMissingMetric: true,
		VerdictOtherHardware: true, VerdictOverBudget: true, VerdictSelfVerified: true}
	for _, s := range statuses {
		if bad[s] {
			return StatusFailing
		}
	}
	if len(statuses) == 0 {
		return StatusNotRun
	}
	for _, s := range statuses {
		if s == StatusNotRun {
			return StatusNotRun
		}
	}
	for _, s := range statuses {
		if s != VerdictPassed {
			return StatusProvisional
		}
	}
	return StatusPassed
}

func TestGateProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	reached := map[string]int{}
	for seed := 0; seed < runs; seed++ {
		runGateProperty(t, int64(seed), reached)
	}
	for _, name := range []string{VerdictPassed, VerdictPredatesBudget, VerdictProvisional, VerdictOverBudget, VerdictIncomplete, VerdictUnbudgeted, VerdictRevoked,
		VerdictOtherData, VerdictMissingMetric, VerdictSelfVerified, VerdictIncorrect, VerdictOtherHardware, "refused-self",
		"gate-" + StatusPassed, "gate-" + StatusFailing, "gate-" + StatusNotRun, "gate-" + StatusProvisional, "ready"} {
		if reached[name] == 0 {
			t.Errorf("%s never reached", name)
		}
	}
	t.Logf("reached %v", reached)
}

func runGateProperty(t *testing.T, seed int64, reached map[string]int) {
	rng := rand.New(rand.NewSource(seed))
	f := setup(t)
	m := &gateModel{profiles: map[string]*mProfile{}}
	apply := func(c Change, proposer, approver Actor) {
		f.approveBy(c, proposer, approver)
		m.events++
	}
	third := Actor{ID: "runner-3", Kind: KindSystem}
	fay := Actor{ID: "fay", Kind: KindHuman}
	apply(Change{Kind: ChangeHardware, Hardware: hardware("hw1")}, ann, bob)
	apply(Change{Kind: ChangeHardware, Hardware: hardware("hw2")}, ann, bob)
	apply(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: runner.ID}}, cat, dan)
	apply(Change{Kind: ChangeHarness, Harness: Harness{ID: "h3", Identity: third.ID}}, eve, dan)
	ids := []string{}
	revoked := false
	budgets := 0
	for step := 0; step < 40; step++ {
		switch op := rng.Intn(10); {
		case op == 0 || len(ids) == 0:
			gate := Gates[rng.Intn(len(Gates))]
			id := "p" + strconv.Itoa(step)
			p := prof(id, gate)
			apply(Change{Kind: ChangeProfile, Profile: p}, ann, bob)
			m.profiles[id] = &mProfile{gate: gate, axes: p.Axes}
			ids = append(ids, id)
		case op == 1:
			id := ids[rng.Intn(len(ids))]
			budgets++
			bid := "b" + strconv.Itoa(budgets)
			hw := []string{"hw1", "hw2"}[rng.Intn(2)]
			l := limits(m.profiles[id].gate)
			l[0].Value = int64(2500000 + rng.Intn(1000000))
			apply(Change{Kind: ChangeBudget, Budget: Budget{ID: bid, Profile: id, Hardware: hw, Limits: l}}, ann, bob)
			model := &mBudget{id: bid, hardware: hw, limits: map[string]int64{}}
			for _, x := range l {
				model.limits[x.Metric] = x.Value
			}
			m.profiles[id].budget = model
		case op == 2:
			id := ids[rng.Intn(len(ids))]
			b := m.profiles[id].budget
			if b == nil || b.approved {
				continue
			}
			approver := fay
			if rng.Intn(2) == 0 {
				approver = eve
			}
			apply(Change{Kind: ChangeApproval, Approval: Approval{Budget: b.id, Distribution: sum("d")}}, ann, approver)
			b.approved, b.event, b.byEve = true, m.events, approver == eve
		case op == 3 && rng.Intn(4) == 0:
			id := ids[rng.Intn(len(ids))]
			if !m.profiles[id].retired {
				apply(Change{Kind: ChangeRetire, Retire: Retire{Profile: id}}, ann, bob)
				m.profiles[id].retired = true
			}
		case op == 4 && rng.Intn(10) == 0 && !revoked:
			f.must(f.r.RevokeHarness("h1", ann))
			f.must(f.r.RevokeHarness("h3", ann))
			revoked = true
			for _, x := range m.runs {
				x.revoked = true
			}
		default:
			if revoked {
				continue
			}
			id := ids[rng.Intn(len(ids))]
			p := m.profiles[id]
			if p.retired {
				continue
			}
			x := runOf(prof(id, p.gate), []string{"1.0.0", "1.0.1"}[rng.Intn(2)])
			model := &mRun{profile: id, build: x.Build, data: true, output: true, metrics: map[string]int64{}}
			switch rng.Intn(11) {
			case 0:
				x.Rows--
			case 1:
				x.Output, model.output = sum("wrong"), false
			case 2:
				x.Hardware = "hw2"
			case 3:
				x.Metrics[0].Value = int64(2000000 + rng.Intn(2000000))
			case 4:
				x.DurationS = 0
			case 5:
				x.Dataset, model.data = sum("other"), false
			case 6:
				drop := rng.Intn(len(x.Metrics))
				x.Metrics = append(append([]Measure{}, x.Metrics[:drop]...), x.Metrics[drop+1:]...)
			case 7:
				x.Interrupted = !x.Interrupted
			}
			model.rows, model.duration, model.hardware, model.interrupted = x.Rows, x.DurationS, x.Hardware, x.Interrupted
			for _, measure := range x.Metrics {
				model.metrics[measure.Metric] = measure.Value
			}
			actor := runner
			if rng.Intn(3) == 0 {
				actor, model.third = third, true
				if p.budget != nil && p.budget.byEve {
					reason(t, "self verification", second(f.r.Record(x, actor)), ReasonSelfVerification)
					reached["refused-self"]++
					continue
				}
			}
			v, err := f.r.Record(x, actor)
			f.must(err)
			m.events++
			model.event = m.events
			m.runs = append(m.runs, model)
			p.runs = append(p.runs, len(m.runs)-1)
			if want := m.verdict(model); v.Verdict != want {
				t.Fatalf("seed %d step %d verdict %s, model %s", seed, step, v.Verdict, want)
			}
		}
		for _, id := range ids {
			for index, number := range f.r.profiles[id].runs {
				v, _ := f.r.Run(number)
				want := m.verdict(m.runs[m.profiles[id].runs[index]])
				reached[want]++
				if v.Verdict != want {
					t.Fatalf("seed %d step %d run %d verdict %s, model %s", seed, step, number, v.Verdict, want)
				}
			}
		}
		for _, build := range []string{"1.0.0", "1.0.1"} {
			ready := true
			for _, gate := range Gates {
				g, _ := f.r.Gate(gate, build)
				want := m.gate(gate, build)
				reached["gate-"+want]++
				if g.Status != want {
					t.Fatalf("seed %d step %d gate %s %s: %s, model %s", seed, step, gate, build, g.Status, want)
				}
				ready = ready && want == StatusPassed
			}
			if r, _ := f.r.Readiness(build); r.Ready != ready {
				t.Fatalf("seed %d step %d readiness %s %v, model %v", seed, step, build, r.Ready, ready)
			}
		}
	}
	if revoked {
		return
	}
	for _, id := range ids {
		p := m.profiles[id]
		if p.retired {
			continue
		}
		budgets++
		bid := "b" + strconv.Itoa(budgets)
		f.approveBy(Change{Kind: ChangeBudget, Budget: Budget{ID: bid, Profile: id, Hardware: "hw1", Limits: limits(p.gate)}}, ann, bob)
		f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: bid, Distribution: sum("d")}}, ann, fay)
		f.must(second(f.r.Record(runOf(prof(id, p.gate), "1.0.0"), runner)))
	}
	for _, gate := range Gates {
		id := "final-" + gate
		f.approveBy(Change{Kind: ChangeProfile, Profile: prof(id, gate)}, ann, bob)
		f.approveBy(Change{Kind: ChangeBudget, Budget: Budget{ID: "bfinal-" + gate, Profile: id, Hardware: "hw1", Limits: limits(gate)}}, ann, bob)
		f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "bfinal-" + gate, Distribution: sum("d")}}, ann, fay)
		f.must(second(f.r.Record(runOf(prof(id, gate), "1.0.0"), runner)))
	}
	if r, _ := f.r.Readiness("1.0.0"); !r.Ready {
		t.Fatalf("seed %d not ready after clean runs %+v", seed, r)
	}
	reached["ready"]++
}
