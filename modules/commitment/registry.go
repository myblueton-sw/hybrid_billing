package commitment

import (
	"sort"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type revision struct {
	number    int
	facts     Facts
	state     string
	reasons   []string
	digest    string
	writtenAt time.Time
	judgedAt  time.Time
}

type period struct {
	revisions []*revision
}

func (p *period) highest() *revision { return p.revisions[len(p.revisions)-1] }

func (p *period) lastSealed() *revision {
	for index := len(p.revisions) - 1; index >= 0; index-- {
		if p.revisions[index].state == StateSealed {
			return p.revisions[index]
		}
	}
	return nil
}

type stored struct {
	c       Commitment
	periods map[string]*period
}

type space struct {
	commitments map[string]*stored
}

type Registry struct {
	mu         sync.Mutex
	profile    decimal.Profile
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(profile decimal.Profile, limits Limits, clock func() time.Time) (*Registry, error) {
	if !profile.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{profile: profile, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func (r *Registry) amount(text string) (decimal.Decimal, string, bool) {
	if len(text) > maxNumberBytes {
		return decimal.Decimal{}, "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || value.Sign() < 0 {
		return decimal.Decimal{}, "", false
	}
	return value, value.String(), true
}

func (r *Registry) commitmentForm(c *Commitment) bool {
	prepaid, canonical, ok := r.amount(c.Prepaid)
	_, known := quanta[c.Quantum]
	if !validID(c.ID) || !validID(c.Provider) || !validID(c.PayerAccount) || !kinds[c.Kind] || !validCurrency(c.Currency) ||
		!validMonth(c.First) || !validMonth(c.Last) || c.Last < c.First || !payments[c.Payment] || !ok || !known || !recoveries[c.Recovery] {
		return false
	}
	if _, err := decimal.Parse(r.profile, c.Quantum); err != nil {
		return false
	}
	if (c.Payment == PaymentNone) != (prepaid.Sign() == 0) || (c.Payment == PaymentNone && c.Recovery == RecoveryRecovered) {
		return false
	}
	c.Prepaid = canonical
	return true
}

func (r *Registry) RecordCommitment(workspace string, c Commitment) (Commitment, error) {
	if !validID(workspace) || !r.commitmentForm(&c) {
		return Commitment{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s != nil && s.commitments[c.ID] != nil {
		if s.commitments[c.ID].c != c {
			return Commitment{}, ErrConflict
		}
		return c, nil
	}
	if s != nil && len(s.commitments) >= r.limits.Commitments {
		return Commitment{}, ErrLimit
	}
	if s == nil {
		s = &space{commitments: map[string]*stored{}}
		r.workspaces[workspace] = s
	}
	s.commitments[c.ID] = &stored{c: c, periods: map[string]*period{}}
	return c, nil
}

func (r *Registry) factsForm(f Facts) (Facts, bool) {
	_, recurring, okR := r.amount(f.Recurring)
	_, amortized, okA := r.amount(f.Amortized)
	if !okR || !okA {
		return Facts{}, false
	}
	out := Facts{Recurring: recurring, Amortized: amortized, Lines: make([]Line, 0, len(f.Lines))}
	seen := map[string]bool{}
	for _, line := range f.Lines {
		_, rec, okR := r.amount(line.Recurring)
		_, am, okA := r.amount(line.Amortized)
		if !validID(line.Source) || seen[line.Source] || !validID(line.Commitment) || !okR || !okA {
			return Facts{}, false
		}
		seen[line.Source] = true
		switch line.Kind {
		case LineUsed:
			_, list, okL := r.amount(line.List)
			if !validID(line.Account) || !okL {
				return Facts{}, false
			}
			line.List = list
		case LineUnused:
			if line.Account != "" || line.List != "" {
				return Facts{}, false
			}
		default:
			return Facts{}, false
		}
		line.Recurring, line.Amortized = rec, am
		out.Lines = append(out.Lines, line)
	}
	sort.Slice(out.Lines, func(a, b int) bool { return out.Lines[a].Source < out.Lines[b].Source })
	return out, true
}

func sameFacts(a, b Facts) bool {
	if a.Recurring != b.Recurring || a.Amortized != b.Amortized || len(a.Lines) != len(b.Lines) {
		return false
	}
	for index := range a.Lines {
		if a.Lines[index] != b.Lines[index] {
			return false
		}
	}
	return true
}

func zeroText(text string) bool { return text == "0" }

func (r *Registry) RecordPeriod(workspace, id, month string, number int, facts Facts) (RevisionState, error) {
	if len(facts.Lines) > r.limits.Lines {
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
	p := st.periods[month]
	count := 0
	if p != nil {
		count = len(p.revisions)
	}
	if number <= count {
		existing := p.revisions[number-1]
		if !sameFacts(existing.facts, canonical) {
			return RevisionState{}, ErrConflict
		}
		return viewRevision(existing), nil
	}
	if number > count+1 || month < st.c.First || month > st.c.Last {
		return RevisionState{}, ErrRejected
	}
	if st.c.Payment == PaymentNone {
		if !zeroText(canonical.Amortized) {
			return RevisionState{}, ErrRejected
		}
		for _, line := range canonical.Lines {
			if !zeroText(line.Amortized) {
				return RevisionState{}, ErrRejected
			}
		}
	}
	if count >= r.limits.Revisions {
		return RevisionState{}, ErrLimit
	}
	if p == nil {
		p = &period{}
		st.periods[month] = p
	}
	for _, earlier := range p.revisions {
		if earlier.state == StateWritten {
			earlier.state = StateSuperseded
		}
	}
	rev := &revision{number: number, facts: canonical, state: StateWritten, writtenAt: now, reasons: []string{}}
	rev.digest = periodDigest(workspace, st.c, month, number, canonical)
	p.revisions = append(p.revisions, rev)
	return viewRevision(rev), nil
}

func (r *Registry) find(workspace, id string) (*stored, error) {
	s := r.workspaces[workspace]
	if s == nil || s.commitments[id] == nil {
		return nil, ErrNotFound
	}
	return s.commitments[id], nil
}

func viewRevision(rev *revision) RevisionState {
	return RevisionState{Revision: rev.number, State: rev.state, Reasons: append([]string{}, rev.reasons...), Digest: rev.digest, WrittenAt: rev.writtenAt, JudgedAt: rev.judgedAt}
}

func (r *Registry) sum(texts ...string) exact.Fraction {
	return sumOf(r.profile, texts...)
}

func sumOf(profile decimal.Profile, texts ...string) exact.Fraction {
	total := exact.Whole(0)
	for _, text := range texts {
		value, _ := decimal.Parse(profile, text)
		made, _ := exact.Of(value)
		total = total.Plus(made)
	}
	return total
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }

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
	p := st.periods[month]
	if p == nil || number > len(p.revisions) {
		return RevisionState{}, ErrNotFound
	}
	rev := p.revisions[number-1]
	if rev != p.highest() {
		return RevisionState{}, ErrRejected
	}
	if rev.state == StateSealed {
		return viewRevision(rev), nil
	}
	reasons := []string{}
	for _, line := range rev.facts.Lines {
		if line.Commitment != id {
			reasons = append(reasons, ReasonForeignCommitment)
			break
		}
	}
	if r.duplicate(workspace, st, month, rev) {
		reasons = append(reasons, ReasonDuplicateSource)
	}
	recurring, amortized := []string{}, []string{}
	for _, line := range rev.facts.Lines {
		recurring, amortized = append(recurring, line.Recurring), append(amortized, line.Amortized)
	}
	if !same(r.sum(recurring...), r.sum(rev.facts.Recurring)) || !same(r.sum(amortized...), r.sum(rev.facts.Amortized)) {
		reasons = append(reasons, ReasonCostNotPreserved)
	}
	counted := []string{rev.facts.Amortized}
	for other, q := range st.periods {
		if other == month {
			continue
		}
		largest := q.highest().facts.Amortized
		if sealed := q.lastSealed(); sealed != nil && r.sum(largest).Less(r.sum(sealed.facts.Amortized)) {
			largest = sealed.facts.Amortized
		}
		counted = append(counted, largest)
	}
	if r.sum(st.c.Prepaid).Less(r.sum(counted...)) {
		reasons = append(reasons, ReasonPrepaidExceeded)
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
	for _, st := range r.workspaces[workspace].commitments {
		if st.c.Provider != own.c.Provider || st.c.PayerAccount != own.c.PayerAccount {
			continue
		}
		for other, q := range st.periods {
			if st == own && other == month {
				continue
			}
			for _, candidate := range []*revision{q.highest(), q.lastSealed()} {
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
	p := st.periods[month]
	if p == nil {
		return CurrentView{}, ErrNotFound
	}
	top := p.highest()
	state := CurrentNotJudged
	switch top.state {
	case StateSealed:
		state = CurrentSealed
	case StateHeld:
		state = CurrentCorrectionHeld
	}
	return CurrentView{State: state, Revision: top.number, Digest: top.digest}, nil
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
	p := st.periods[month]
	if p == nil {
		return nil, ErrNotFound
	}
	out := make([]RevisionState, 0, len(p.revisions))
	for _, rev := range p.revisions {
		out = append(out, viewRevision(rev))
	}
	return out, nil
}

func (r *Registry) Revision(workspace, id, month string, number int) (Period, error) {
	if !validID(workspace) || !validID(id) || !validMonth(month) || number < 1 || number > 1000 {
		return Period{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return Period{}, err
	}
	p := st.periods[month]
	if p == nil || number > len(p.revisions) {
		return Period{}, ErrNotFound
	}
	rev := p.revisions[number-1]
	if rev != p.highest() || rev.state != StateSealed {
		return Period{}, ErrRejected
	}
	facts := rev.facts
	facts.Lines = append([]Line{}, rev.facts.Lines...)
	return Period{made: true, profile: r.profile, limits: r.limits, commitment: st.c, month: month, revision: number, facts: facts, digest: rev.digest}, nil
}

func (r *Registry) Commitment(workspace, id string) (Commitment, error) {
	if !validID(workspace) || !validID(id) {
		return Commitment{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return Commitment{}, err
	}
	return st.c, nil
}
