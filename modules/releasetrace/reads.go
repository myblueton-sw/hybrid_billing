package releasetrace

import "sort"

func (r *Registry) Requirement(k Key, buildID string) (RequirementView, error) {
	r.enter()
	defer r.leave()
	q := r.requirements[k]
	if q == nil {
		return RequirementView{}, ErrNotFound
	}
	v := RequirementView{Requirement: copyChange(Change{Requirement: q.view}).Requirement, State: r.state(q, buildID), Retired: q.retired, Replacement: q.replacement,
		Tests: []TestState{}}
	if n := len(q.impls); n > 0 {
		v.Commit = q.impls[n-1].commit
	}
	for _, t := range q.view.Tests {
		s := TestState{Test: t.ID}
		if res, ok := r.counted(q, t.ID, buildID); res != nil {
			s.Counted, s.Passed, s.Passes, s.Failures = ok, res.latest.Passed, res.passes, res.failures
		}
		v.Tests = append(v.Tests, s)
	}
	return v, nil
}

func (r *Registry) Path(id string) (PathView, error) {
	r.enter()
	defer r.leave()
	p := r.paths[id]
	if p == nil {
		return PathView{}, ErrNotFound
	}
	v := PathView{Path: copyChange(Change{Path: p.view}).Path, State: PathDesigned, Build: p.build, Proof: p.proof, Reconciliations: append([]Reconcile{}, p.recon...)}
	if p.active {
		v.State = PathActive
	}
	return v, nil
}

func (r *Registry) Results(namespace, test, buildID string) (ResultView, error) {
	r.enter()
	defer r.leave()
	res := r.results[[3]string{namespace, test, buildID}]
	if res == nil {
		return ResultView{}, ErrNotFound
	}
	return r.resultView(res), nil
}

func (r *Registry) Deactivate(id, reason string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(id) || !validText(reason) {
		return ErrInvalid
	}
	p := r.paths[id]
	if p == nil {
		return ErrNotFound
	}
	p.active, p.build, p.proof = false, "", ""
	return nil
}

func (r *Registry) Replacements(k Key) ([]Key, error) {
	r.enter()
	defer r.leave()
	q := r.requirements[k]
	if q == nil {
		return nil, ErrNotFound
	}
	found := []*requirement{}
	if !q.retired {
		for _, rep := range r.requirements {
			if rep != q && !weaker(q, rep) {
				found = append(found, rep)
			}
		}
	}
	sort.Slice(found, func(a, b int) bool { return found[a].seq < found[b].seq })
	out := make([]Key, 0, len(found))
	for _, rep := range found {
		out = append(out, Key{Namespace: rep.view.Namespace, ID: rep.view.ID})
	}
	return out, nil
}
