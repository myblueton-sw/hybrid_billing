package perfgate

func (r *Registry) Profile(id string) (Profile, bool, error) {
	r.enter()
	defer r.leave()
	p := r.profiles[id]
	if p == nil {
		return Profile{}, false, ErrNotFound
	}
	return p.view, p.retired, nil
}

func (r *Registry) Hardware(id string) (Hardware, error) {
	r.enter()
	defer r.leave()
	h := r.hardware[id]
	if h == nil {
		return Hardware{}, ErrNotFound
	}
	return *h, nil
}

func (r *Registry) Budget(id string) (BudgetView, error) {
	r.enter()
	defer r.leave()
	return r.budgetView(id)
}

func (r *Registry) budgetView(id string) (BudgetView, error) {
	b := r.budgets[id]
	if b == nil {
		return BudgetView{}, ErrNotFound
	}
	v := b.view
	v.Budget = copyChange(Change{Budget: v.Budget}).Budget
	return v, nil
}

func (r *Registry) ActiveBudget(profile string) (BudgetView, error) {
	r.enter()
	defer r.leave()
	return r.budgetView(r.active[profile])
}

func (r *Registry) Run(number uint64) (RunView, error) {
	r.enter()
	defer r.leave()
	x := r.runs[number]
	if x == nil {
		return RunView{}, ErrNotFound
	}
	return r.view(x), nil
}

func (r *Registry) Runs(profile string, after uint64, limit int) ([]RunView, error) {
	r.enter()
	defer r.leave()
	if limit < 1 || limit > maxPage {
		return nil, ErrInvalid
	}
	p := r.profiles[profile]
	if p == nil {
		return nil, ErrNotFound
	}
	out := []RunView{}
	for _, number := range p.runs {
		if number > after && len(out) < limit {
			out = append(out, r.view(r.runs[number]))
		}
	}
	return out, nil
}

func (r *Registry) status(p *profile, build string) ProfileStatus {
	s := ProfileStatus{Profile: p.view.ID, Status: StatusNotRun}
	b := r.budgets[r.active[p.view.ID]]
	for _, number := range p.runs {
		x := r.runs[number]
		if x.view.Run.Build != build {
			continue
		}
		verdict := r.verdict(x)
		s.Status, s.Run = verdict, number
		if b != nil && b.view.State == BudgetApproved && x.event > b.approved && verdict != VerdictPassed {
			s.Failures++
		}
	}
	return s
}

var failing = map[string]bool{VerdictOtherData: true, VerdictIncomplete: true, VerdictIncorrect: true, VerdictRevoked: true, VerdictSelfVerified: true,
	VerdictMissingMetric: true, VerdictOtherHardware: true, VerdictOverBudget: true}

func (r *Registry) gate(gate, build string) GateView {
	v := GateView{Gate: gate, Build: build, Status: StatusPassed, Profiles: []ProfileStatus{}, Retired: []ProfileStatus{}}
	provisional, failed := false, false
	for _, p := range r.sorted() {
		if p.view.Gate != gate {
			continue
		}
		s := r.status(p, build)
		if p.retired {
			v.Retired = append(v.Retired, s)
			continue
		}
		v.Profiles = append(v.Profiles, s)
		failed = failed || failing[s.Status]
		provisional = provisional || s.Status != VerdictPassed
	}
	notRun := len(v.Profiles) == 0
	for _, s := range v.Profiles {
		notRun = notRun || s.Status == StatusNotRun
	}
	switch {
	case failed:
		v.Status = StatusFailing
	case notRun:
		v.Status = StatusNotRun
	case provisional:
		v.Status = StatusProvisional
	}
	return v
}

func (r *Registry) sorted() []*profile {
	out := make([]*profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		out = append(out, p)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].view.ID < out[j-1].view.ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (r *Registry) Gate(gate, build string) (GateView, error) {
	r.enter()
	defer r.leave()
	if !oneOf(gate, Gates...) || !semver(build) {
		return GateView{}, ErrInvalid
	}
	return r.gate(gate, build), nil
}

func (r *Registry) Readiness(build string) (ReadinessView, error) {
	r.enter()
	defer r.leave()
	if !semver(build) {
		return ReadinessView{}, ErrInvalid
	}
	v := ReadinessView{Build: build, Ready: true}
	for _, gate := range Gates {
		g := r.gate(gate, build)
		v.Gates = append(v.Gates, g)
		v.Ready = v.Ready && g.Status == StatusPassed
	}
	return v, nil
}
