package perfgate

import "math/bits"

const (
	maxRows     = 1000000000000000
	maxDuration = 1000000000
)

func (x Run) valid() bool {
	if !validID(x.Profile) || !validID(x.Hardware) || !semver(x.Build) || !validHex(x.Dataset) || !validHex(x.Output) || x.Rows < 0 || x.Rows > maxRows ||
		x.DurationS < 0 || x.DurationS > maxDuration || x.Mismatches < 0 || x.Mismatches > maxRows || len(x.Metrics) > len(Metrics) {
		return false
	}
	for index, m := range x.Metrics {
		if !oneOf(m.Metric, Metrics...) || m.Value < 0 || m.Value > maxValue || index > 0 && x.Metrics[index-1].Metric >= m.Metric {
			return false
		}
	}
	return true
}

func (b *budget) people() []string {
	if b == nil {
		return nil
	}
	return []string{b.view.Proposer, b.view.Approver, b.view.ApprovalBy, b.view.ApprovedBy}
}

func overlaps(h *harness, p *profile, b *budget) bool {
	for _, person := range append([]string{p.by.proposer, p.by.approver}, b.people()...) {
		if person != "" && (person == h.by.proposer || person == h.by.approver) {
			return true
		}
	}
	return false
}

func (r *Registry) Record(x Run, actor Actor) (RunView, error) {
	now := r.enter()
	defer r.leave()
	switch {
	case !actor.valid() || !x.valid():
		return RunView{}, ErrInvalid
	case actor.Kind == KindAssistant:
		return RunView{}, refuse(ReasonAINotAllowed)
	}
	id := r.identities[actor.ID]
	h := r.harnesses[id]
	if actor.Kind != KindSystem || h == nil || h.revoked {
		return RunView{}, refuse(ReasonNotHarness)
	}
	p := r.profiles[x.Profile]
	switch {
	case p == nil || r.hardware[x.Hardware] == nil:
		return RunView{}, ErrNotFound
	case p.retired:
		return RunView{}, refuse(ReasonRetired)
	case overlaps(h, p, r.budgets[r.active[x.Profile]]):
		return RunView{}, refuse(ReasonSelfVerification)
	case len(r.runs) >= maxRunsTotal && len(p.runs) < maxRuns:
		return RunView{}, ErrLimit
	}
	r.sequence++
	r.events++
	x.Metrics = append([]Measure(nil), x.Metrics...)
	stored := &run{view: RunView{Number: r.sequence, Run: x, Harness: id, At: now}, harness: h, event: r.events}
	r.runs[r.sequence] = stored
	p.runs = append(p.runs, r.sequence)
	if len(p.runs) > maxRuns {
		delete(r.runs, p.runs[0])
		p.runs = p.runs[1:]
	}
	return r.view(stored), nil
}

func (r *Registry) RevokeHarness(id string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(id) {
		return ErrInvalid
	}
	h := r.harnesses[id]
	if h == nil {
		return ErrNotFound
	}
	h.revoked = true
	return nil
}

func (r *Registry) view(x *run) RunView {
	v := x.view
	v.Run.Metrics = append([]Measure(nil), v.Run.Metrics...)
	v.Verdict = r.verdict(x)
	return v
}

func short(p Profile, x Run) bool {
	a := p.Axes
	switch p.Gate {
	case GateE2E:
		return x.Rows < a.RowsRun
	case GateMonthlyClose:
		return x.Rows < a.RowsMonth
	case GateInterruptResume:
		return x.Rows < a.RowsRun || !x.Interrupted
	case GateDailyInflow:
		if x.Rows < a.RowsDay || x.DurationS < daySeconds {
			return true
		}
	default:
		if x.Rows < a.BacklogRows {
			return true
		}
	}
	return x.DurationS == 0 || productLess(x.Rows, daySeconds, a.RowsDay, x.DurationS)
}

func productLess(a, b, c, d int64) bool {
	highLeft, lowLeft := bits.Mul64(uint64(a), uint64(b))
	highRight, lowRight := bits.Mul64(uint64(c), uint64(d))
	return highLeft < highRight || highLeft == highRight && lowLeft < lowRight
}

func (r *Registry) verdict(x *run) string {
	p := r.profiles[x.view.Run.Profile]
	b := r.budgets[r.active[x.view.Run.Profile]]
	run := x.view.Run
	switch {
	case run.Dataset != p.view.Dataset:
		return VerdictOtherData
	case short(p.view, run):
		return VerdictIncomplete
	case run.Output != p.view.Expected || run.Mismatches > 0:
		return VerdictIncorrect
	case x.harness.revoked:
		return VerdictRevoked
	case overlaps(x.harness, p, b):
		return VerdictSelfVerified
	}
	measured := map[string]int64{}
	for _, m := range run.Metrics {
		measured[m.Metric] = m.Value
	}
	for _, metric := range Metrics {
		if _, ok := measured[metric]; !ok && (metric != MetricResume || p.view.Gate == GateInterruptResume) {
			return VerdictMissingMetric
		}
	}
	switch {
	case b == nil:
		return VerdictUnbudgeted
	case run.Hardware != b.view.Budget.Hardware:
		return VerdictOtherHardware
	}
	for _, l := range b.view.Budget.Limits {
		value, ok := measured[l.Metric]
		if !ok {
			return VerdictMissingMetric
		}
		if l.Metric == MetricThroughput && value < l.Value || l.Metric != MetricThroughput && value > l.Value {
			return VerdictOverBudget
		}
	}
	switch {
	case b.view.State == BudgetProposed:
		return VerdictProvisional
	case x.event < b.approved:
		return VerdictPredatesBudget
	}
	return VerdictPassed
}
