package formula

import "sort"

func (r *Registry) EvaluateRevision(workspace, set string, number int, target string, inputs []Input) (Result, error) {
	if len(inputs) > r.limits.Variables {
		return Result{}, ErrLimit
	}
	if !validID(workspace) || !validID(set) || !validRevision(number) || !validID(target) || !formed(r.profile, inputs) {
		return Result{}, ErrInvalid
	}
	r.enter()
	_, rev, ok := r.find(workspace, set, number)
	var compiled *Compiled
	digest := ""
	if ok {
		compiled, digest = rev.compiled, rev.digest
	}
	r.leave()
	if !ok {
		return Result{}, ErrNotFound
	}
	if compiled == nil {
		return Result{}, ErrRejected
	}
	result, err := compiled.Evaluate(target, inputs)
	if err != nil {
		return Result{}, err
	}
	result.Set, result.Revision, result.Digest = set, number, digest
	return result, nil
}

func (r *Registry) Active(workspace, set string) (ActiveView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) {
		return ActiveView{}, ErrInvalid
	}
	f, ok := r.workspaces[workspace][set]
	if !ok {
		return ActiveView{}, ErrNotFound
	}
	out := ActiveView{Revision: f.active, Epoch: f.epoch, Sequence: f.sequence}
	if rev, ok := f.revisions[f.active]; ok {
		out.Digest = rev.digest
	}
	return out, nil
}

func (r *Registry) Revision(workspace, set string, number int) (RevisionView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || !validRevision(number) {
		return RevisionView{}, ErrInvalid
	}
	f, rev, ok := r.find(workspace, set, number)
	if !ok {
		return RevisionView{}, ErrNotFound
	}
	return f.view(set, rev), nil
}

func (r *Registry) Content(workspace, set string, number int) (Set, error) {
	if !validID(workspace) || !validID(set) || !validRevision(number) {
		return Set{}, ErrInvalid
	}
	r.enter()
	_, rev, ok := r.find(workspace, set, number)
	r.leave()
	if !ok {
		return Set{}, ErrNotFound
	}
	return cloneSet(rev.content), nil
}

func (r *Registry) Revisions(workspace, set string, after, limit int) ([]Listed, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || after < 0 || after > maxRevision || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	f, ok := r.workspaces[workspace][set]
	if !ok {
		return nil, ErrNotFound
	}
	numbers := make([]int, 0, len(f.revisions))
	for number := range f.revisions {
		if number > after {
			numbers = append(numbers, number)
		}
	}
	sort.Ints(numbers)
	if len(numbers) > limit {
		numbers = numbers[:limit]
	}
	out := make([]Listed, 0, len(numbers))
	for _, number := range numbers {
		out = append(out, Listed{Number: number, State: f.revisions[number].state})
	}
	return out, nil
}
