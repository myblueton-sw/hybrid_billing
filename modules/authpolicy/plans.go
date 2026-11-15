package authpolicy

func (p *plan) chunks() int { return (len(p.view.Users) + p.view.Chunk - 1) / p.view.Chunk }

func (p *plan) render() PlanView {
	v := PlanView{Plan: copyChange(Change{Plan: p.view}).Plan, State: PlanRunning, Chunks: p.chunks(), Completed: p.completed, Next: p.completed,
		Proposer: p.by.proposer, Approver: p.by.approver, Lifted: [2]string{p.liftBy.proposer, p.liftBy.approver}}
	switch {
	case p.lifted:
		v.State = PlanLifted
	case p.completed == v.Chunks:
		v.State = PlanComplete
	}
	return v
}

func (r *Registry) Complete(id string, index int, worker Principal) (PlanView, error) {
	r.enter()
	defer r.leave()
	switch {
	case !validID(worker.User) || !oneOf(worker.Kind, KindHuman, KindSystem, KindAssistant):
		return PlanView{}, ErrInvalid
	case worker.Kind == KindAssistant:
		return PlanView{}, refuse(ReasonAINotAllowed)
	case worker.Kind == KindHuman:
		return PlanView{}, refuse(ReasonNotSystem)
	}
	p := r.plans[id]
	switch {
	case p == nil:
		return PlanView{}, ErrNotFound
	case index < 0 || index >= p.chunks():
		return PlanView{}, ErrInvalid
	case p.lifted:
		return PlanView{}, refuse(ReasonLifted)
	case index > p.completed:
		return PlanView{}, refuse(ReasonOutOfOrder)
	case index == p.completed:
		p.completed++
	}
	return p.render(), nil
}

func (p *plan) users(index int) []string {
	low := index * p.view.Chunk
	high := min(low+p.view.Chunk, len(p.view.Users))
	return append([]string{}, p.view.Users[low:high]...)
}

func (r *Registry) Chunk(id string, index int) ([]string, error) {
	r.enter()
	defer r.leave()
	p := r.plans[id]
	switch {
	case p == nil:
		return nil, ErrNotFound
	case index < 0 || index >= p.chunks():
		return nil, ErrInvalid
	}
	return p.users(index), nil
}

func (r *Registry) Plan(id string) (PlanView, error) {
	r.enter()
	defer r.leave()
	p := r.plans[id]
	if p == nil {
		return PlanView{}, ErrNotFound
	}
	return p.render(), nil
}

func (r *Registry) Policy(space string) (RevisionView, error) {
	r.enter()
	defer r.leave()
	rev := r.current(space)
	if rev == nil {
		return RevisionView{}, ErrNotFound
	}
	return copyRevision(*rev), nil
}

func (r *Registry) Revision(space string, number uint64) (RevisionView, error) {
	r.enter()
	defer r.leave()
	if w := r.workspaces[space]; w != nil {
		for _, v := range w.revisions {
			if v.Number == number {
				return copyRevision(v), nil
			}
		}
	}
	return RevisionView{}, ErrNotFound
}

func copyRevision(v RevisionView) RevisionView {
	v.Revision = copyChange(Change{Revision: v.Revision}).Revision
	return v
}
