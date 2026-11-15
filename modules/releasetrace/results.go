package releasetrace

func (x Result) valid() bool {
	return validID(x.Namespace) && validID(x.Test) && semver(x.Build) && digestOK(x.Evidence)
}

func overlap(a, b people) bool {
	for _, x := range []string{a.proposer, a.approver} {
		if x != "" && (x == b.proposer || x == b.approver) {
			return true
		}
	}
	return false
}

func (r *Registry) selfVerified(h *harness, namespace, test, buildID string) bool {
	if b := r.builds[buildID]; b != nil && overlap(h.by, b.by) {
		return true
	}
	for _, k := range r.tests[[2]string{namespace, test}] {
		q := r.requirements[k]
		if overlap(h.by, q.by) {
			return true
		}
		if n := len(q.impls); n > 0 && overlap(h.by, q.impls[n-1].by) {
			return true
		}
	}
	return false
}

func (r *Registry) Record(x Result, actor Actor) (ResultView, error) {
	now := r.enter()
	defer r.leave()
	switch {
	case !actor.valid() || !x.valid():
		return ResultView{}, ErrInvalid
	case actor.Kind == KindAssistant:
		return ResultView{}, refuse(ReasonAINotAllowed)
	}
	h := r.harnesses[r.identities[actor.ID]]
	switch {
	case actor.Kind != KindSystem || h == nil || h.revoked:
		return ResultView{}, refuse(ReasonNotHarness)
	case len(r.tests[[2]string{x.Namespace, x.Test}]) == 0:
		return ResultView{}, refuse(ReasonUnknownTest)
	case r.builds[x.Build] == nil:
		return ResultView{}, refuse(ReasonUnknownBuild)
	case r.selfVerified(h, x.Namespace, x.Test, x.Build):
		return ResultView{}, refuse(ReasonSelfVerification)
	}
	key := [3]string{x.Namespace, x.Test, x.Build}
	res := r.results[key]
	if res == nil {
		if len(r.results) >= maxResults {
			return ResultView{}, ErrLimit
		}
		res = &result{}
		r.results[key] = res
	}
	res.latest, res.harness, res.at = x, h, now
	if x.Passed {
		res.passes++
	} else {
		res.failures++
	}
	return r.resultView(res), nil
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

func (r *Registry) counted(q *requirement, test, buildID string) (*result, bool) {
	res := r.results[[3]string{q.view.Namespace, test, buildID}]
	b := r.builds[buildID]
	if res == nil || b == nil || len(q.impls) == 0 || b.seq <= q.impls[len(q.impls)-1].seq || res.harness.revoked ||
		r.selfVerified(res.harness, q.view.Namespace, test, buildID) {
		return res, false
	}
	return res, true
}

func (r *Registry) state(q *requirement, buildID string) string {
	if len(q.impls) == 0 {
		return StateDesigned
	}
	passed := true
	for _, t := range q.view.Tests {
		res, ok := r.counted(q, t.ID, buildID)
		switch {
		case ok && !res.latest.Passed:
			return StateFailing
		case !ok:
			passed = false
		}
	}
	switch {
	case !passed:
		return StateImplemented
	case r.proofs[Key{q.view.Namespace, q.view.ID}][buildID] != nil:
		return StateCustomerProven
	}
	return StateTested
}

func (r *Registry) tested(q *requirement, buildID string) bool {
	s := r.state(q, buildID)
	return s == StateTested || s == StateCustomerProven
}

func (r *Registry) category(p *path, cat, buildID string) bool {
	for _, k := range p.view.Requirements {
		q := r.requirements[k]
		for _, t := range q.view.Tests {
			if t.Category != cat {
				continue
			}
			if res, ok := r.counted(q, t.ID, buildID); ok && res.latest.Passed {
				return true
			}
		}
	}
	return false
}

func (r *Registry) reconciled(p *path) bool {
	n := len(p.recon)
	if n < 2 {
		return false
	}
	a, _ := month(p.recon[n-2].Period)
	b, _ := month(p.recon[n-1].Period)
	now := r.now.Year()*12 + int(r.now.Month()) - 1
	return p.recon[n-2].Matched && p.recon[n-1].Matched && b == a+1 && now-b <= 3
}

func (r *Registry) pathPeople(p *path, buildID string) []string {
	out := []string{}
	for _, by := range p.reconBy {
		out = append(out, by.proposer, by.approver)
	}
	for _, k := range p.view.Requirements {
		q := r.requirements[k]
		for _, t := range q.view.Tests {
			if res, ok := r.counted(q, t.ID, buildID); ok {
				out = append(out, res.harness.by.proposer, res.harness.by.approver)
			}
		}
	}
	return out
}

func (r *Registry) resultView(res *result) ResultView {
	counted := true
	for _, k := range r.tests[[2]string{res.latest.Namespace, res.latest.Test}] {
		if _, ok := r.counted(r.requirements[k], res.latest.Test, res.latest.Build); !ok {
			counted = false
		}
	}
	return ResultView{Result: res.latest, Harness: res.harness.id, At: res.at, Passes: res.passes, Failures: res.failures, Counted: counted}
}
