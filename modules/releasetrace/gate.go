package releasetrace

import "sort"

func (x Release) valid() bool {
	return validID(x.Version) && semver(x.Build) && ascendingKeys(x.Requirements, 1, maxReleaseReqs) && ascendingIDs(x.Paths, 1, maxReleasePaths)
}

func subject(k Key) string { return k.Namespace + "/" + k.ID }

func retiredSubject(k, replacement Key) string {
	if replacement == (Key{}) {
		return subject(k)
	}
	return subject(k) + ">" + subject(replacement)
}

func (r *Registry) Gate(x Release) (GateView, error) {
	r.enter()
	defer r.leave()
	if !x.valid() {
		return GateView{}, ErrInvalid
	}
	v := GateView{Problems: []Problem{}, States: map[Key]string{}, Templates: []string{}, RetiredBlocking: []Key{}}
	add := func(problem, s string) { v.Problems = append(v.Problems, Problem{Problem: problem, Subject: s}) }
	selected := map[Key]bool{}
	for _, k := range x.Requirements {
		selected[k] = true
		q := r.requirements[k]
		switch {
		case q == nil:
			add(ProblemUnknown, subject(k))
			continue
		case q.retired:
			add(ProblemRetired, retiredSubject(k, q.replacement))
		}
		v.States[k] = r.state(q, x.Build)
		if !r.tested(q, x.Build) {
			add(ProblemNotTested, subject(k))
		}
	}
	common := false
	for k, q := range r.requirements {
		switch {
		case q.retired && q.replacement == (Key{}):
			v.RetiredBlocking = append(v.RetiredBlocking, k)
			continue
		case q.retired:
			continue
		}
		if q.view.Common {
			common = true
			if !selected[k] {
				add(ProblemCommonMissing, subject(k))
			}
		}
		if q.view.Blocking && !r.tested(q, x.Build) {
			add(ProblemBlocking, subject(k))
		}
	}
	if !common {
		add(ProblemCommonUnregistered, "")
	}
	for _, id := range x.Paths {
		p := r.paths[id]
		if p == nil {
			add(ProblemUnknown, id)
			continue
		}
		before := len(v.Problems)
		switch {
		case !p.active:
			add(ProblemPathInactive, id)
		case p.build != x.Build:
			add(ProblemPathBuild, id)
		}
		if n := len(p.recon); n > 0 && !p.recon[n-1].Matched {
			add(ProblemPathReconcile, id)
		}
		for _, k := range p.view.Requirements {
			if !selected[k] {
				add(ProblemPathRequirement, id+">"+subject(k))
			}
		}
		if !r.pathEvidence(p, x.Build) {
			add(ProblemPathEvidence, id)
		}
		if len(v.Problems) == before && p.view.Template != "" {
			v.Templates = append(v.Templates, p.view.Template)
		}
	}
	sort.Slice(v.Problems, func(a, b int) bool {
		x, y := v.Problems[a], v.Problems[b]
		return x.Problem < y.Problem || x.Problem == y.Problem && x.Subject < y.Subject
	})
	sort.Slice(v.RetiredBlocking, func(a, b int) bool { return keyLess(v.RetiredBlocking[a], v.RetiredBlocking[b]) })
	sort.Strings(v.Templates)
	v.Ready = len(v.Problems) == 0
	return v, nil
}

func (r *Registry) pathEvidence(p *path, buildID string) bool {
	for _, k := range p.view.Requirements {
		if q := r.requirements[k]; q.retired || !r.tested(q, buildID) {
			return false
		}
	}
	for _, cat := range Categories {
		if !r.category(p, cat, buildID) {
			return false
		}
	}
	return true
}
