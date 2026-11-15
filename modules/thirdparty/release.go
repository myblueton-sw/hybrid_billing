package thirdparty

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func (x Release) valid() bool {
	if !validID(x.Version) || !hexOf(x.Commit, 40, 64) || len(x.Components) == 0 || len(x.Components) > maxPerRelease {
		return false
	}
	for index, id := range x.Components {
		if !validID(id) || index > 0 && x.Components[index-1] >= id {
			return false
		}
	}
	return true
}

func (r *Registry) checkRelease(x Release) Check {
	out := Check{Problems: []Problem{}}
	listed, names := map[string]bool{}, map[string]string{}
	for _, id := range x.Components {
		listed[id] = true
	}
	for _, id := range x.Components {
		k := r.components[id]
		if k == nil {
			out.Problems = append(out.Problems, Problem{Component: id, Problem: ProblemUnknown})
			continue
		}
		switch k.state {
		case StateRevoked:
			out.Problems = append(out.Problems, Problem{Component: id, Problem: ProblemRevoked})
		case StateCandidate, StateRejected:
			out.Problems = append(out.Problems, Problem{Component: id, Problem: ProblemNotAdopted})
		}
		v := k.view
		if v.Distribution == DistToolOnly || v.Distribution == DistNotShipped {
			out.Problems = append(out.Problems, Problem{Component: id, Problem: ProblemNotShipped})
		}
		for _, dep := range v.Dependencies {
			if (v.Distribution == DistBundled || v.Distribution == DistSeparate) && !listed[dep] {
				out.Problems = append(out.Problems, Problem{Component: id, Problem: ProblemMissingDependency})
				break
			}
		}
		name := strings.ToLower(v.Name)
		if names[name] != "" {
			out.Problems = append(out.Problems, Problem{Component: id, Problem: ProblemDuplicateName})
		}
		names[name] = id
	}
	if len(out.Problems) > 0 {
		return out
	}
	out.Ready = true
	var e encoder
	for _, value := range []string{"hybrid-billing/notice-bundle/v1", x.Version, x.Commit, strconv.Itoa(len(x.Components))} {
		e.text(value)
	}
	for _, id := range x.Components {
		v := r.components[id].view
		for _, value := range []string{v.ID, v.Name, v.Version, v.Commit, v.License, v.LicenseText, v.Notice, v.Artifact, v.Patch, v.Distribution, v.Modification} {
			e.text(value)
		}
	}
	sum := sha256.Sum256(e.out.Bytes())
	out.Bundle = hex.EncodeToString(sum[:])
	return out
}

func (r *Registry) CheckRelease(x Release) (Check, error) {
	r.enter()
	defer r.leave()
	if !x.valid() {
		return Check{}, ErrInvalid
	}
	return r.checkRelease(x), nil
}

func sameRelease(a, b Release) bool {
	if a.Commit != b.Commit || len(a.Components) != len(b.Components) {
		return false
	}
	for index := range a.Components {
		if a.Components[index] != b.Components[index] {
			return false
		}
	}
	return true
}

func (r *Registry) RecordRelease(x Release, actor Actor) (ReleaseView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return ReleaseView{}, err
	}
	if !x.valid() {
		return ReleaseView{}, ErrInvalid
	}
	if old := r.releases[x.Version]; old != nil {
		if sameRelease(old.Release, x) {
			return copyRelease(*old), nil
		}
		return ReleaseView{}, ErrConflict
	}
	c := r.checkRelease(x)
	switch {
	case !c.Ready:
		return ReleaseView{}, refuse(ReasonNotReady)
	case len(r.releases) >= maxReleases:
		return ReleaseView{}, ErrLimit
	}
	v := &ReleaseView{Release: Release{Version: x.Version, Commit: x.Commit, Components: append([]string(nil), x.Components...)}, Bundle: c.Bundle, By: actor.ID, At: now}
	r.releases[x.Version] = v
	return copyRelease(*v), nil
}

func copyRelease(v ReleaseView) ReleaseView {
	v.Release.Components = append([]string(nil), v.Release.Components...)
	return v
}
