package marketplace

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func (r *Registry) benefitForm(state, amount, evidence string) (string, bool) {
	switch state {
	case Applied:
		canonical, ok := r.amount(amount)
		return canonical, ok && validDigest(evidence)
	case NotApplied:
		return "", amount == "" && validDigest(evidence)
	case Unknown:
		return "", amount == "" && evidence == ""
	}
	return "", false
}

func (r *Registry) factsForm(f Facts) (Facts, bool) {
	out := Facts{Lines: make([]Line, 0, len(f.Lines)), Recognitions: make([]Recognition, 0, len(f.Recognitions))}
	seen := map[string]bool{}
	for _, line := range f.Lines {
		charged, okC := r.amount(line.Charged)
		discount, okD := r.benefitForm(line.Discount.State, line.Discount.Amount, line.Discount.Evidence)
		if !validID(line.Source) || seen[line.Source] || !lineKinds[line.Kind] || !validID(line.Account) || !okC || !okD {
			return Facts{}, false
		}
		if line.List != "" {
			list, okL := r.amount(line.List)
			if !okL {
				return Facts{}, false
			}
			line.List = list
		}
		seen[line.Source] = true
		line.Charged, line.Discount.Amount = charged, discount
		out.Lines = append(out.Lines, line)
	}
	programs := map[string]bool{}
	for _, rec := range f.Recognitions {
		amount, ok := r.benefitForm(rec.State, rec.Amount, rec.Evidence)
		if !validID(rec.Program) || programs[rec.Program] || !ok {
			return Facts{}, false
		}
		programs[rec.Program] = true
		rec.Amount = amount
		out.Recognitions = append(out.Recognitions, rec)
	}
	sort.Slice(out.Lines, func(a, b int) bool { return out.Lines[a].Source < out.Lines[b].Source })
	sort.Slice(out.Recognitions, func(a, b int) bool { return out.Recognitions[a].Program < out.Recognitions[b].Program })
	return out, true
}

func sameFacts(a, b Facts) bool {
	if len(a.Lines) != len(b.Lines) || len(a.Recognitions) != len(b.Recognitions) {
		return false
	}
	for index := range a.Lines {
		if a.Lines[index] != b.Lines[index] {
			return false
		}
	}
	for index := range a.Recognitions {
		if a.Recognitions[index] != b.Recognitions[index] {
			return false
		}
	}
	return true
}

func copyFacts(f Facts) Facts {
	return Facts{Lines: append([]Line{}, f.Lines...), Recognitions: append([]Recognition{}, f.Recognitions...)}
}

func viewRevision(rev *revision) RevisionState {
	return RevisionState{Revision: rev.number, State: rev.state, Reasons: append([]string{}, rev.reasons...), Digest: rev.digest, WrittenAt: rev.writtenAt, JudgedAt: rev.judgedAt}
}

func (r *Registry) RecordStatement(workspace, id, month string, number int, facts Facts) (RevisionState, error) {
	if len(facts.Lines) > r.limits.Lines || len(facts.Recognitions) > r.limits.Recognitions {
		return RevisionState{}, ErrLimit
	}
	canonical, formed := r.factsForm(facts)
	if !validID(workspace) || !validID(id) || !validMonth(month) || number < 1 || number > 1000 || !formed {
		return RevisionState{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return RevisionState{}, err
	}
	s := st.statements[month]
	count := 0
	if s != nil {
		count = len(s.revisions)
	}
	if number <= count {
		existing := s.revisions[number-1]
		if !sameFacts(existing.facts, canonical) {
			return RevisionState{}, ErrConflict
		}
		return viewRevision(existing), nil
	}
	if number > count+1 || month < st.t.Term.Start || month > st.t.Term.End {
		return RevisionState{}, ErrRejected
	}
	if count >= r.limits.Revisions {
		return RevisionState{}, ErrLimit
	}
	if s == nil {
		s = &statement{}
		st.statements[month] = s
	}
	for _, earlier := range s.revisions {
		if earlier.state == StateWritten {
			earlier.state = StateSuperseded
		}
	}
	rev := &revision{number: number, facts: canonical, state: StateWritten, writtenAt: now, reasons: []string{}}
	rev.digest = statementDigest(workspace, st.t, month, number, canonical)
	s.revisions = append(s.revisions, rev)
	return viewRevision(rev), nil
}

func (r *Registry) Seal(workspace, id, month string, number int) (RevisionState, error) {
	if !validID(workspace) || !validID(id) || !validMonth(month) || number < 1 || number > 1000 {
		return RevisionState{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return RevisionState{}, err
	}
	s := st.statements[month]
	if s == nil || number > len(s.revisions) {
		return RevisionState{}, ErrNotFound
	}
	rev := s.revisions[number-1]
	if rev != s.highest() {
		return RevisionState{}, ErrRejected
	}
	if rev.state == StateSealed {
		return viewRevision(rev), nil
	}
	reasons := []string{}
	if r.duplicate(workspace, st, month, rev) {
		reasons = append(reasons, ReasonDuplicateSource)
	}
	charged := []string{}
	for _, line := range rev.facts.Lines {
		charged = append(charged, line.Charged)
	}
	total := r.sum(charged...)
	for _, rec := range rev.facts.Recognitions {
		if rec.State == Applied && total.Less(r.sum(rec.Amount)) {
			reasons = append(reasons, ReasonRecognitionExceedsCharge)
			break
		}
	}
	rev.reasons, rev.judgedAt, rev.state = reasons, now, StateSealed
	if len(reasons) > 0 {
		rev.state = StateHeld
	}
	return viewRevision(rev), nil
}

func (r *Registry) duplicate(workspace string, own *stored, month string, rev *revision) bool {
	sources := map[string]bool{}
	for _, line := range rev.facts.Lines {
		sources[line.Source] = true
	}
	for _, st := range r.workspaces[workspace].transactions {
		if st.t.Provider != own.t.Provider {
			continue
		}
		for other, s := range st.statements {
			if st == own && other == month {
				continue
			}
			for _, candidate := range []*revision{s.highest(), s.lastSealed()} {
				if candidate == nil {
					continue
				}
				for _, line := range candidate.facts.Lines {
					if sources[line.Source] {
						return true
					}
				}
			}
		}
	}
	return false
}

func currentOf(s *statement) CurrentView {
	top := s.highest()
	state := CurrentNotJudged
	switch top.state {
	case StateSealed:
		state = CurrentSealed
	case StateHeld:
		state = CurrentCorrectionHeld
	}
	return CurrentView{State: state, Revision: top.number, Digest: top.digest}
}

func (r *Registry) Current(workspace, id, month string) (CurrentView, error) {
	if !validID(workspace) || !validID(id) || !validMonth(month) {
		return CurrentView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return CurrentView{}, err
	}
	s := st.statements[month]
	if s == nil {
		return CurrentView{}, ErrNotFound
	}
	return currentOf(s), nil
}

func (r *Registry) Revisions(workspace, id, month string) ([]RevisionState, error) {
	if !validID(workspace) || !validID(id) || !validMonth(month) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	s := st.statements[month]
	if s == nil {
		return nil, ErrNotFound
	}
	out := make([]RevisionState, 0, len(s.revisions))
	for _, rev := range s.revisions {
		out = append(out, viewRevision(rev))
	}
	return out, nil
}

type Statement struct {
	made        bool
	workspace   string
	profile     decimal.Profile
	limits      Limits
	transaction Transaction
	month       string
	revision    int
	facts       Facts
	digest      string
}

type StatementView struct {
	Transaction Transaction
	Month       string
	Revision    int
	Facts       Facts
	Digest      string
}

func (s Statement) View() StatementView {
	return StatementView{Transaction: s.transaction, Month: s.month, Revision: s.revision, Facts: copyFacts(s.facts), Digest: s.digest}
}

func (r *Registry) Statement(workspace, id, month string, number int) (Statement, error) {
	if !validID(workspace) || !validID(id) || !validMonth(month) || number < 1 || number > 1000 {
		return Statement{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return Statement{}, err
	}
	s := st.statements[month]
	if s == nil || number > len(s.revisions) {
		return Statement{}, ErrNotFound
	}
	rev := s.revisions[number-1]
	if rev != s.highest() || rev.state != StateSealed {
		return Statement{}, ErrRejected
	}
	return Statement{made: true, workspace: workspace, profile: r.profile, limits: r.limits, transaction: st.t, month: month, revision: number, facts: copyFacts(rev.facts), digest: rev.digest}, nil
}
