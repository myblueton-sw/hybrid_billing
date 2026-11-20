package pricingpolicy

import (
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

func nilPort(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func New(limit int, authority Authority, targets Targets, clock func() time.Time) (*Registry, error) {
	if limit < 1 || limit > 1000 || nilPort(authority) || nilPort(targets) || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{limit: limit, authority: authority, targets: targets, clock: clock, records: make(map[[2]string]Record)}, nil
}
func (r *Registry) now() (time.Time, error) {
	at := r.clock()
	if !instant(at) || !r.last.IsZero() && at.Before(r.last) {
		return time.Time{}, ErrRejected
	}
	r.last = at.UTC()
	return r.last, nil
}
func (r *Registry) current(p Proposal) error {
	target, err := r.targets.Current(p.Spec.Scope, p.Spec.From, p.Spec.To)
	if err != nil {
		return err
	}
	if !validTarget(target) || target != p.Target || target.Currency != p.Spec.Policy.Currency || target.Path != p.Spec.Policy.Path {
		return ErrConflict
	}
	return nil
}
func (r *Registry) Propose(id, actor, reason string, spec Spec) (Record, error) {
	if !validID(id) || !validID(actor) || !validReason(reason) || !form(spec) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.authority.Check(spec.Scope, actor, ProposeAction); err != nil {
		return Record{}, err
	}
	target, err := r.targets.Current(spec.Scope, spec.From, spec.To)
	if err != nil {
		return Record{}, err
	}
	if !validTarget(target) || target.Currency != spec.Policy.Currency || target.Path != spec.Policy.Path {
		return Record{}, ErrConflict
	}
	record := clone(Record{Proposal: Proposal{Schema: Schema, Algorithm: rating.Algorithm, ID: id, Proposer: actor, Rationale: reason, Spec: spec, Target: target}, State: Pending})
	record.Digest, err = digest(record.Proposal)
	if err != nil {
		return Record{}, err
	}
	if err = compile(record.Proposal.Spec); err != nil {
		return Record{}, err
	}
	at, err := r.now()
	if err != nil {
		return Record{}, err
	}
	key := [2]string{spec.Scope.Workspace, id}
	if prior, ok := r.records[key]; ok {
		if prior.Digest != record.Digest {
			return Record{}, ErrConflict
		}
		return clone(prior), nil
	}
	if len(r.records) >= r.limit {
		return Record{}, ErrLimit
	}
	for _, prior := range r.records {
		old := prior.Proposal.Spec
		if old.Scope.Workspace == spec.Scope.Workspace && old.Policy.ID == spec.Policy.ID && old.Policy.Version == spec.Policy.Version &&
			(!reflect.DeepEqual(old.Policy, record.Proposal.Spec.Policy) || old.Profile != spec.Profile || old.Limits != spec.Limits) {
			return Record{}, ErrConflict
		}
	}
	record.ProposedAt = at
	r.records[key] = record
	return clone(record), nil
}
func (r *Registry) Approve(workspace, id, want, actor, reason string) (Record, error) {
	if !validID(workspace) || !validID(id) || !validDigest(want) || !validID(actor) || !validReason(reason) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := [2]string{workspace, id}
	record, ok := r.records[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	p := record.Proposal
	if actor == p.Proposer {
		return Record{}, ErrRejected
	}
	if err := r.authority.Check(p.Spec.Scope, actor, ApproveAction); err != nil {
		return Record{}, err
	}
	if err := r.authority.Check(p.Spec.Scope, p.Proposer, ProposeAction); err != nil {
		return Record{}, err
	}
	if record.Digest != want {
		return Record{}, ErrConflict
	}
	if err := r.current(p); err != nil {
		return Record{}, err
	}
	at, err := r.now()
	if err != nil {
		return Record{}, err
	}
	if record.State == Approved {
		if record.Approval.Actor != actor || record.Approval.Rationale != reason {
			return Record{}, ErrConflict
		}
		return clone(record), nil
	}
	if record.State != Pending {
		return Record{}, ErrRejected
	}
	for _, other := range r.records {
		s := other.Proposal.Spec
		if other.State == Approved && s.Scope.Workspace == p.Spec.Scope.Workspace && s.Scope.Contract == p.Spec.Scope.Contract && s.From.Before(p.Spec.To) && p.Spec.From.Before(s.To) {
			return Record{}, ErrConflict
		}
	}
	record.State = Approved
	record.Approval = &Decision{Actor: actor, Rationale: reason, At: at}
	r.records[key] = record
	return clone(record), nil
}
func (r *Registry) Revoke(workspace, id, want, actor, reason string) (Record, error) {
	if !validID(workspace) || !validID(id) || !validDigest(want) || !validID(actor) || !validReason(reason) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := [2]string{workspace, id}
	record, ok := r.records[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	if err := r.authority.Check(record.Proposal.Spec.Scope, actor, RevokeAction); err != nil {
		return Record{}, err
	}
	if record.Digest != want {
		return Record{}, ErrConflict
	}
	at, err := r.now()
	if err != nil {
		return Record{}, err
	}
	if record.State == Revoked {
		if record.Revocation.Actor != actor || record.Revocation.Rationale != reason {
			return Record{}, ErrConflict
		}
		return clone(record), nil
	}
	if record.State != Approved {
		return Record{}, ErrRejected
	}
	record.State = Revoked
	record.Revocation = &Decision{Actor: actor, Rationale: reason, At: at}
	r.records[key] = record
	return clone(record), nil
}
func (r *Registry) Record(workspace, id string) (Record, error) {
	if !validID(workspace) || !validID(id) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[[2]string{workspace, id}]
	if !ok {
		return Record{}, ErrNotFound
	}
	return clone(record), nil
}
func (r *Registry) Resolve(scope Scope, id, want, actor string, from, to time.Time) (Record, error) {
	if !validScope(scope) || !validID(id) || !validDigest(want) || !validID(actor) || !interval(from, to) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.authority.Check(scope, actor, UseAction); err != nil {
		return Record{}, err
	}
	record, ok := r.records[[2]string{scope.Workspace, id}]
	if !ok {
		return Record{}, ErrNotFound
	}
	p := record.Proposal
	if record.Digest != want || p.Spec.Scope != scope {
		return Record{}, ErrConflict
	}
	if record.State != Approved || from.Before(p.Spec.From) || to.After(p.Spec.To) {
		return Record{}, ErrRejected
	}
	if err := r.current(p); err != nil {
		return Record{}, err
	}
	if _, err := r.now(); err != nil {
		return Record{}, err
	}
	return clone(record), nil
}
