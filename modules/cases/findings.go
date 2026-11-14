package cases

import "sort"

type Transition struct {
	Step     string
	Reason   string
	Evidence string
	Owner    string
	Remedy   Remedy
}

func targetForm(kind string, t Target) bool {
	if t.Customer == "" && t.Contract == "" && t.Document == "" && t.Resource == "" || kind == "unattributed" && t.Customer != "" {
		return false
	}
	return optionalID(t.Customer) && optionalID(t.Contract) && optionalID(t.Document) && optionalID(t.Line) && optionalID(t.Resource) && (t.Line == "" || t.Document != "")
}

func findingForm(f Finding) bool {
	if !validID(f.ID) || !findingKinds[f.Kind] || !targetForm(f.Kind, f.Target) || !validDate(f.PeriodStart) || !validDate(f.PeriodEnd) || f.PeriodStart > f.PeriodEnd {
		return false
	}
	if !meanings[f.Meaning] || !validDigest(f.Evidence) || !severities[f.Severity] || !validID(f.Owner) || !sources[f.Source] {
		return false
	}
	if (f.Source == SourceDetector) != (f.Detector != "") || !optionalID(f.Detector) {
		return false
	}
	if f.Meaning == MeaningNone {
		return f.Amount == "" && f.Currency == ""
	}
	return validNumber(f.Amount, f.Meaning == MeaningDifference) && validCurrency(f.Currency)
}

func viewFinding(f *finding) FindingView {
	return FindingView{Finding: f.f, State: f.state, Owner: f.owner, Author: f.author, Digest: f.digest, Written: f.written, Steps: append([]Step(nil), f.steps...)}
}

func (r *Registry) RecordFinding(workspace string, f Finding, actor Actor) (FindingView, error) {
	if !validActor(actor) {
		return FindingView{}, ErrInvalid
	}
	if actor.Side != SideOperator {
		return FindingView{}, ErrRejected
	}
	if !validID(workspace) || !findingForm(f) {
		return FindingView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s != nil {
		if old := s.findings[f.ID]; old != nil {
			if old.f != f || old.author != actor.ID {
				return FindingView{}, ErrConflict
			}
			return viewFinding(old), nil
		}
		if len(s.findings) >= r.limits.Findings {
			return FindingView{}, ErrLimit
		}
	} else if len(r.workspaces) >= r.limits.Workspaces {
		return FindingView{}, ErrLimit
	}
	s = r.space(workspace)
	made := &finding{f: f, state: FindingInReview, owner: f.Owner, author: actor.ID, written: now, digest: findingDigest(workspace, f, actor.ID)}
	s.findings[f.ID] = made
	return viewFinding(made), nil
}

func (r *Registry) ConfirmNormal(workspace, id string, actor Actor, t Transition) (FindingView, error) {
	return r.move(workspace, id, actor, StepConfirm, t)
}

func (r *Registry) RequireAdjustment(workspace, id string, actor Actor, t Transition) (FindingView, error) {
	return r.move(workspace, id, actor, StepAdjust, t)
}

func (r *Registry) Resolve(workspace, id string, actor Actor, t Transition) (FindingView, error) {
	return r.move(workspace, id, actor, StepResolve, t)
}

func (r *Registry) ReopenFinding(workspace, id string, actor Actor, t Transition) (FindingView, error) {
	return r.move(workspace, id, actor, StepReopen, t)
}

func (r *Registry) AssignFinding(workspace, id string, actor Actor, t Transition) (FindingView, error) {
	return r.move(workspace, id, actor, StepAssign, t)
}

func transitionForm(kind string, t Transition) bool {
	if !validID(t.Step) {
		return false
	}
	switch kind {
	case StepAssign:
		return validID(t.Owner) && t.Reason == "" && t.Evidence == "" && t.Remedy == (Remedy{})
	case StepResolve:
		return t.Reason == "" && t.Owner == "" && validDigest(t.Evidence) && remedyKinds[t.Remedy.Kind] && validID(t.Remedy.Reference)
	}
	reasons := map[string]map[string]bool{StepConfirm: normalReasons, StepAdjust: adjustReasons, StepReopen: reopenReasons}[kind]
	return reasons[t.Reason] && validDigest(t.Evidence) && t.Owner == "" && t.Remedy == (Remedy{})
}

var reopenFrom = map[string]map[string]bool{
	FindingConfirmedNormal:  {"new_evidence": true, "decision_revised": true},
	FindingAdjustmentNeeded: {"new_evidence": true, "decision_revised": true},
	FindingResolved:         {"new_evidence": true, "resolution_failed": true},
}

var stepFrom = map[string]map[string]string{
	StepConfirm: {FindingInReview: FindingConfirmedNormal},
	StepAdjust:  {FindingInReview: FindingAdjustmentNeeded},
	StepResolve: {FindingAdjustmentNeeded: FindingResolved},
	StepReopen:  {FindingConfirmedNormal: FindingInReview, FindingAdjustmentNeeded: FindingInReview, FindingResolved: FindingInReview},
	StepAssign:  {FindingInReview: FindingInReview, FindingConfirmedNormal: FindingConfirmedNormal, FindingAdjustmentNeeded: FindingAdjustmentNeeded, FindingResolved: FindingResolved},
}

func (r *Registry) move(workspace, id string, actor Actor, kind string, t Transition) (FindingView, error) {
	if !validActor(actor) {
		return FindingView{}, ErrInvalid
	}
	if actor.Side != SideOperator {
		return FindingView{}, ErrRejected
	}
	if !validID(workspace) || !validID(id) || !transitionForm(kind, t) {
		return FindingView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.findings[id] == nil {
		return FindingView{}, ErrNotFound
	}
	f := s.findings[id]
	if old, ok := s.steps[[2]string{id, t.Step}]; ok {
		if old.Kind != kind || old.Reason != t.Reason || old.Evidence != t.Evidence || old.Owner != t.Owner || old.Remedy != t.Remedy || old.Actor != actor.ID {
			return FindingView{}, ErrConflict
		}
		return viewFinding(f), nil
	}
	to, ok := stepFrom[kind][f.state]
	if kind == StepReopen && !reopenFrom[f.state][t.Reason] {
		ok = false
	}
	if !ok || (kind == StepConfirm || kind == StepAdjust) && (actor.ID == f.author || actor.ID == f.reopener) {
		return FindingView{}, ErrRejected
	}
	if (kind == StepAssign || kind == StepReopen) && f.counted >= r.limits.Steps {
		return FindingView{}, ErrLimit
	}
	step := Step{ID: t.Step, Kind: kind, Reason: t.Reason, Evidence: t.Evidence, Owner: t.Owner, Remedy: t.Remedy, Actor: actor.ID, From: f.state, To: to, At: now}
	step.Digest = stepDigest(workspace, id, step)
	f.steps = append(f.steps, step)
	s.steps[[2]string{id, t.Step}] = step
	f.state = to
	switch kind {
	case StepAssign:
		f.owner, f.counted = t.Owner, f.counted+1
	case StepReopen:
		f.reopener, f.counted = actor.ID, f.counted+1
	}
	return viewFinding(f), nil
}

func (r *Registry) Finding(workspace, id string, actor Actor) (FindingView, error) {
	if !validActor(actor) {
		return FindingView{}, ErrInvalid
	}
	if actor.Side != SideOperator {
		return FindingView{}, ErrRejected
	}
	if !validID(workspace) || !validID(id) {
		return FindingView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.findings[id] == nil {
		return FindingView{}, ErrNotFound
	}
	return viewFinding(s.findings[id]), nil
}

func findingFilterForm(f FindingFilter) bool {
	return (f.State == "" || stepStates[f.State]) && optionalID(f.Customer) && !(f.Unattributed && f.Customer != "") && (f.Severity == "" || severities[f.Severity]) && optionalID(f.After)
}

var stepStates = map[string]bool{FindingInReview: true, FindingConfirmedNormal: true, FindingAdjustmentNeeded: true, FindingResolved: true}

func (r *Registry) Findings(workspace string, actor Actor, filter FindingFilter) ([]FindingView, error) {
	if !validActor(actor) {
		return nil, ErrInvalid
	}
	if actor.Side != SideOperator {
		return nil, ErrRejected
	}
	if !validID(workspace) || !findingFilterForm(filter) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return []FindingView{}, nil
	}
	ids := make([]string, 0, len(s.findings))
	for id, f := range s.findings {
		if id > filter.After && (filter.State == "" || f.state == filter.State) && (filter.Customer == "" || f.f.Target.Customer == filter.Customer) && (!filter.Unattributed || f.f.Target.Customer == "") && (filter.Severity == "" || f.f.Severity == filter.Severity) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > r.limits.Page {
		ids = ids[:r.limits.Page]
	}
	out := make([]FindingView, 0, len(ids))
	for _, id := range ids {
		out = append(out, viewFinding(s.findings[id]))
	}
	return out, nil
}
