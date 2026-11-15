package simulation

func (r *Registry) Dataset(id string) (Dataset, error) {
	r.enter()
	defer r.leave()
	d := r.datasets[id]
	if d == nil {
		return Dataset{}, ErrNotFound
	}
	return copyChange(Change{Dataset: *d}).Dataset, nil
}

func (r *Registry) Lineage(id string) ([]Dataset, error) {
	r.enter()
	defer r.leave()
	d := r.datasets[id]
	if d == nil {
		return nil, ErrNotFound
	}
	out := []Dataset{}
	for ; d != nil; d = r.datasets[d.Parent] {
		out = append(out, copyChange(Change{Dataset: *d}).Dataset)
	}
	return out, nil
}

func (r *Registry) Scenario(id string) (Scenario, error) {
	r.enter()
	defer r.leave()
	s := r.scenarios[id]
	if s == nil {
		return Scenario{}, ErrNotFound
	}
	return copyChange(Change{Scenario: s.view}).Scenario, nil
}

func (r *Registry) Expectation(scenario string, version int) (Expectation, error) {
	r.enter()
	defer r.leave()
	s := r.scenarios[scenario]
	if s == nil || version < 1 || version > len(s.expectations) {
		return Expectation{}, ErrNotFound
	}
	return copyChange(Change{Expectation: s.expectations[version-1].view}).Expectation, nil
}

func (r *Registry) Run(number uint64) (RunView, error) {
	r.enter()
	defer r.leave()
	if number < 1 || number > uint64(len(r.runs)) {
		return RunView{}, ErrNotFound
	}
	return r.runView(r.runs[number-1]), nil
}

func (r *Registry) Runs(scenario string, after uint64, limit int) ([]RunView, error) {
	r.enter()
	defer r.leave()
	if limit < 1 || limit > maxPage {
		return nil, ErrInvalid
	}
	s := r.scenarios[scenario]
	if s == nil {
		return nil, ErrNotFound
	}
	out := []RunView{}
	for _, number := range s.runs {
		if number > after && len(out) < limit {
			out = append(out, r.runView(r.runs[number-1]))
		}
	}
	return out, nil
}

func (r *Registry) Current(scenario string) (CurrentView, error) {
	r.enter()
	defer r.leave()
	s := r.scenarios[scenario]
	if s == nil {
		return CurrentView{}, ErrNotFound
	}
	v, _ := r.current(s)
	return CurrentView{Scenario: scenario, Verdict: v.Verdict, Run: v.Number, Engine: v.Engine, Rules: v.Rules, Expected: len(s.expectations)}, nil
}

func (r *Registry) Matrix(profile string) ([]CellView, error) {
	r.enter()
	defer r.leave()
	cells := r.profiles[profile]
	if cells == nil {
		return nil, ErrNotFound
	}
	out, cache := make([]CellView, 0, len(cells)), currents{}
	for _, x := range cells {
		v := CellView{Cell: copyCell(x.view), Status: r.cellStatus(x, cache), Revisions: x.revisions, Failures: x.failures, History: []CellChange{}}
		for _, h := range x.history {
			v.History = append(v.History, CellChange{Replaced: copyCell(h.Replaced), Status: h.Status})
		}
		if x.set && x.view.Coverage == CoverageTested {
			current := r.currentOf(cache, x.view.Scenario)
			v.Engine, v.Rules = current.Engine, current.Rules
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *Registry) Summary() map[string]int {
	r.enter()
	defer r.leave()
	out := map[string]int{CellOpen: 0, CellUnsupported: 0, CellUndecided: 0, CellFailed: 0, CellPassed: 0, CellPending: 0}
	cache := currents{}
	for _, cells := range r.profiles {
		for _, x := range cells {
			out[r.cellStatus(x, cache)]++
		}
	}
	return out
}

func (r *Registry) Status(id string) (StatusView, error) {
	r.enter()
	defer r.leave()
	t := r.targets[id]
	if t == nil {
		return StatusView{}, ErrNotFound
	}
	v := StatusView{Target: copyChange(Change{Target: t.view}).Target, Demo: DemoPassed, Integration: IntegrationNotProven}
	for _, scenario := range t.view.Scenarios {
		current, _ := r.current(r.scenarios[scenario])
		switch current.Verdict {
		case VerdictPass:
		case VerdictFail, VerdictNonReproducible, VerdictRevoked:
			v.Demo = DemoFailed
		default:
			if v.Demo == DemoPassed {
				v.Demo = DemoPending
			}
		}
	}
	if t.proof != nil {
		v.Integration, v.Proof = IntegrationUnclassified, *t.proof
	}
	return v, nil
}
