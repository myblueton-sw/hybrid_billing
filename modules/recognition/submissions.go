package recognition

func (r *Registry) Submit(workspace, id, key string, actor Actor) (SubmissionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(key) || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.batches[id] == nil {
		return SubmissionView{}, ErrNotFound
	}
	b := s.batches[id]
	if seen := s.submissions[key]; seen != nil {
		if seen.Batch == id && actor.is(seen.Submitter) {
			return *seen, nil
		}
		return SubmissionView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return SubmissionView{}, err
	}
	if b.view.State != BatchApproved {
		return SubmissionView{}, refuse(ReasonNotApproved)
	}
	if s.profiles[b.view.Batch.Profile].closed[b.view.Batch.Posting] {
		return SubmissionView{}, refuse(ReasonPeriodClosed)
	}
	if b.posted() {
		return SubmissionView{}, refuse(ReasonAlreadyPosted)
	}
	if b.pending() {
		return SubmissionView{}, refuse(ReasonOutcomeUnknown)
	}
	made := &SubmissionView{Key: key, Purpose: Purpose, Batch: id, Digest: b.view.Digest, Submitter: actor.ID, Sent: now, Result: Result{State: ResultSent}}
	s.submissions[key] = made
	s.byKey[key] = b
	b.subs = append(b.subs, made)
	return *made, nil
}

func (res Result) valid() bool {
	switch res.State {
	case ResultPosted:
		return validID(res.External) && res.Reason == ""
	case ResultFailed, ResultRejected:
		return validID(res.Reason) && (res.External == "" || validID(res.External))
	case ResultAccepted, ResultUnknown:
		return res.Reason == "" && (res.External == "" || validID(res.External))
	}
	return false
}

func (r *Registry) Record(workspace, key string, res Result, actor Actor) (SubmissionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) || !res.valid() || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.submissions[key] == nil {
		return SubmissionView{}, ErrNotFound
	}
	sub := s.submissions[key]
	if actor.Kind == KindAssistant {
		return SubmissionView{}, refuse(ReasonAINotAllowed)
	}
	if sub.Result == res {
		return *sub, nil
	}
	if !reports[sub.Result.State][res.State] {
		return SubmissionView{}, ErrConflict
	}
	if res.State == ResultPosted && s.externals[res.External] != "" {
		return SubmissionView{}, ErrConflict
	}
	if res.State == ResultPosted {
		s.externals[res.External] = key
	}
	sub.Result, sub.Reporter, sub.Reported = res, actor.ID, now
	return *sub, nil
}

func (r *Registry) SubmissionOf(workspace, key string) (SubmissionView, error) {
	r.enter()
	defer r.leave()
	s := r.workspace(workspace)
	if s == nil || s.submissions[key] == nil {
		return SubmissionView{}, ErrNotFound
	}
	return *s.submissions[key], nil
}
