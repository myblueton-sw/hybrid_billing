package perfgate

import "time"

func oneOf(value string, set ...string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func (a Axes) list() []int64 {
	return []int64{a.RowsRun, a.RowsDay, a.RowsMonth, a.Customers, a.Accounts, a.SharedPools, a.SkewPermille, a.FanOut, a.ConcurrentCloses, a.PeakWindowMinutes,
		a.RetentionMonths, a.FailureInjections, a.BacklogRows}
}

func (a Axes) valid() bool {
	for _, value := range a.list() {
		if value < 0 || value > maxAxis {
			return false
		}
	}
	return a.SkewPermille <= 1000
}

func (p Profile) valid() bool {
	return validID(p.ID) && oneOf(p.Gate, Gates...) && p.Axes.valid() && validHex(p.Dataset) && validHex(p.Expected)
}

func (p Profile) floor() bool {
	a := p.Axes
	switch p.Gate {
	case GateE2E:
		return a.RowsRun >= runRows
	case GateDailyInflow:
		return a.RowsDay > dayRows
	case GateMonthlyClose:
		return a.RowsMonth >= monthRows
	case GateInterruptResume:
		return a.RowsRun >= runRows && a.FailureInjections >= 1
	}
	return a.RowsDay > dayRows && a.BacklogRows >= dayRows
}

func (h Hardware) valid() bool {
	for _, value := range []int64{h.CPUs, h.MemoryMiB, h.DiskGiB} {
		if value < 1 || value > maxSpec {
			return false
		}
	}
	return validID(h.ID) && oneOf(h.Storage, StorageSSD, StorageHDD, StorageNetwork) && validHex(h.Spec)
}

func (b Budget) valid() bool {
	if !validID(b.ID) || !validID(b.Profile) || !validID(b.Hardware) || len(b.Limits) == 0 || len(b.Limits) > len(Metrics) {
		return false
	}
	for index, l := range b.Limits {
		if !oneOf(l.Metric, Metrics...) || l.Value < 0 || l.Value > maxValue || index > 0 && b.Limits[index-1].Metric >= l.Metric {
			return false
		}
	}
	return true
}

func required(gate string) []string {
	out := []string{MetricCompletion, MetricMaxRSS, MetricP95, MetricP99, MetricThroughput}
	if gate == GateInterruptResume {
		out = append(out, MetricResume)
	}
	return out
}

func clean(c Change) (Change, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeProfile:
		out.Profile = c.Profile
		return out, c.Profile.valid()
	case ChangeHardware:
		out.Hardware = c.Hardware
		return out, c.Hardware.valid()
	case ChangeBudget:
		out.Budget = c.Budget
		out = copyChange(out)
		return out, out.Budget.valid()
	case ChangeApproval:
		out.Approval = c.Approval
		return out, validID(c.Approval.Budget) && validHex(c.Approval.Distribution)
	case ChangeHarness:
		out.Harness = c.Harness
		return out, validID(c.Harness.ID) && validID(c.Harness.Identity)
	case ChangeRetire:
		out.Retire = c.Retire
		return out, validID(c.Retire.Profile)
	}
	return out, false
}

func copyChange(c Change) Change {
	c.Budget.Limits = append([]Limit(nil), c.Budget.Limits...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func encodeChange(c Change) []byte {
	var e encoder
	p, h, b := c.Profile, c.Hardware, c.Budget
	for _, value := range []string{c.Kind, p.ID, p.Gate, p.Dataset, p.Expected, h.ID, h.Storage, h.Spec, b.ID, b.Profile, b.Hardware, c.Approval.Budget,
		c.Approval.Distribution, c.Harness.ID, c.Harness.Identity, c.Retire.Profile} {
		e.text(value)
	}
	for _, value := range append(p.Axes.list(), h.CPUs, h.MemoryMiB, h.DiskGiB) {
		e.number(value)
	}
	e.count(len(b.Limits))
	for _, l := range b.Limits {
		e.text(l.Metric)
		e.number(l.Value)
	}
	return e.out.Bytes()
}

func (r *Registry) check(c Change, proposer, approver string) error {
	switch c.Kind {
	case ChangeProfile:
		switch {
		case r.profiles[c.Profile.ID] != nil:
			return ErrConflict
		case !c.Profile.floor():
			return refuse(ReasonBelowP0)
		case len(r.profiles) >= maxProfiles:
			return ErrLimit
		}
	case ChangeHardware:
		switch {
		case r.hardware[c.Hardware.ID] != nil:
			return ErrConflict
		case len(r.hardware) >= maxHardware:
			return ErrLimit
		}
	case ChangeBudget:
		return r.checkBudget(c.Budget)
	case ChangeApproval:
		return r.checkApproval(c.Approval, approver)
	case ChangeRetire:
		p := r.profiles[c.Retire.Profile]
		switch {
		case p == nil:
			return ErrNotFound
		case p.retired:
			return refuse(ReasonRetired)
		}
	case ChangeHarness:
		switch {
		case r.harnesses[c.Harness.ID] != nil:
			return ErrConflict
		case r.identities[c.Harness.Identity] != "":
			return refuse(ReasonIdentityTaken)
		case len(r.harnesses) >= maxHarnesses:
			return ErrLimit
		}
	}
	return nil
}

func (r *Registry) checkBudget(b Budget) error {
	p := r.profiles[b.Profile]
	if p == nil || r.hardware[b.Hardware] == nil {
		return ErrNotFound
	}
	if r.budgets[b.ID] != nil {
		return ErrConflict
	}
	limits := map[string]int64{}
	for _, l := range b.Limits {
		limits[l.Metric] = l.Value
	}
	for _, metric := range required(p.view.Gate) {
		if _, ok := limits[metric]; !ok {
			return refuse(ReasonMissingMetric)
		}
	}
	if (p.view.Gate == GateDailyInflow || p.view.Gate == GateCatchUp) && limits[MetricThroughput] < (p.view.Axes.RowsDay+daySeconds-1)/daySeconds {
		return refuse(ReasonBelowP0)
	}
	if len(r.budgets) >= maxBudgets {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkApproval(a Approval, approver string) error {
	b := r.budgets[a.Budget]
	switch {
	case b == nil:
		return ErrNotFound
	case b.view.State == BudgetApproved:
		return ErrConflict
	case r.active[b.view.Budget.Profile] != a.Budget:
		return refuse(ReasonNotActive)
	case approver != "" && (approver == b.view.Proposer || approver == b.view.Approver):
		return refuse(ReasonSamePeople)
	}
	return nil
}

func (r *Registry) apply(c Change, by people, now time.Time) {
	c = copyChange(c)
	r.events++
	switch c.Kind {
	case ChangeProfile:
		r.profiles[c.Profile.ID] = &profile{view: c.Profile, by: by}
	case ChangeHardware:
		h := c.Hardware
		r.hardware[h.ID] = &h
	case ChangeBudget:
		r.budgets[c.Budget.ID] = &budget{view: BudgetView{Budget: c.Budget, State: BudgetProposed, Proposer: by.proposer, Approver: by.approver}}
		r.active[c.Budget.Profile] = c.Budget.ID
	case ChangeApproval:
		b := r.budgets[c.Approval.Budget]
		b.view.State, b.view.Distribution, b.view.ApprovalBy, b.view.ApprovedBy, b.view.ApprovedAt = BudgetApproved, c.Approval.Distribution, by.proposer, by.approver, now
		b.approved = r.events
	case ChangeRetire:
		r.profiles[c.Retire.Profile].retired = true
	case ChangeHarness:
		r.harnesses[c.Harness.ID] = &harness{identity: c.Harness.Identity, by: by}
		r.identities[c.Harness.Identity] = c.Harness.ID
	}
}
