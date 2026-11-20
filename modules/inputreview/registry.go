package inputreview

import (
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Registry struct {
	built        bool
	mu           sync.Mutex
	authority    Authority
	current      CurrentEvidence
	clock        func() time.Time
	limit        int
	now          time.Time
	records      map[[2]string]Record
	publications map[Scope]Publication
}

func missing(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

func NewRegistry(limit int, authority Authority, current CurrentEvidence, clock func() time.Time) (*Registry, error) {
	if limit < 1 || limit > 10000 || missing(authority) || missing(current) || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limit: limit, authority: authority, current: current, clock: clock, records: make(map[[2]string]Record), publications: make(map[Scope]Publication)}, nil
}

func validID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, c := range []byte(value) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._:-", rune(c))) {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range []byte(value) {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validScope(s Scope) bool {
	return validID(s.Workspace) && validID(s.Provider) && validID(s.Account) && validID(s.Period)
}

func validEvidence(e Evidence, workspace, run string) bool {
	if !validScope(e.Scope) || e.Scope.Workspace != workspace || e.Run != run || !validID(e.Run) || !validID(e.Method) || !validDigest(e.Digest) || len(e.Findings) > 1000 {
		return false
	}
	if e.Verdict != Match && e.Verdict != Difference || (e.Verdict == Match) != (len(e.Findings) == 0) {
		return false
	}
	for _, finding := range e.Findings {
		if !validID(finding) {
			return false
		}
	}
	return true
}

func clone(record Record) Record {
	record.Evidence.Findings = append([]string(nil), record.Evidence.Findings...)
	if record.Decision != nil {
		decision := *record.Decision
		record.Decision = &decision
	}
	return record
}

func sameEvidence(a, b Evidence) bool {
	if a.Scope != b.Scope || a.Run != b.Run || a.Digest != b.Digest || a.Method != b.Method || a.Verdict != b.Verdict || len(a.Findings) != len(b.Findings) {
		return false
	}
	for i := range a.Findings {
		if a.Findings[i] != b.Findings[i] {
			return false
		}
	}
	return true
}

func (r *Registry) instant() (time.Time, error) {
	now := r.clock().UTC().Round(0)
	if now.IsZero() || now.Year() < 1 || now.Year() > 9999 {
		return time.Time{}, ErrInvalid
	}
	if now.After(r.now) {
		r.now = now
	}
	return r.now, nil
}

func (r *Registry) Compare(workspace, run, id string) (Record, error) {
	if r == nil || !r.built || !validID(workspace) || !validID(run) || !validID(id) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	evidence, err := r.current.Compare(workspace, run)
	if err != nil {
		return Record{}, err
	}
	if !validEvidence(evidence, workspace, run) {
		return Record{}, ErrInvalid
	}
	key := [2]string{workspace, id}
	if prior, exists := r.records[key]; exists {
		if !sameEvidence(prior.Evidence, evidence) {
			return Record{}, ErrConflict
		}
		return clone(prior), nil
	}
	if len(r.records) >= r.limit {
		return Record{}, ErrLimit
	}
	at, err := r.instant()
	if err != nil {
		return Record{}, err
	}
	state := Blocked
	if evidence.Verdict == Match {
		state = Awaiting
	}
	record := clone(Record{ID: id, Evidence: evidence, State: state, Compared: at})
	r.records[key] = record
	return clone(record), nil
}

func validIntent(in Intent) bool {
	if !validID(in.Workspace) || !validID(in.ID) || !validID(in.Actor) || !validDigest(in.Digest) || in.ExpectedScope > MaxRevision || len(in.Rationale) < 1 || len(in.Rationale) > 1024 || !utf8.ValidString(in.Rationale) || strings.TrimSpace(in.Rationale) == "" {
		return false
	}
	for _, c := range in.Rationale {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}

func sameIntent(decision *Decision, in Intent) bool {
	return decision != nil && decision.Actor == in.Actor && decision.Rationale == in.Rationale && decision.Digest == in.Digest && decision.ExpectedScope == in.ExpectedScope
}

func (r *Registry) decide(in Intent, confirm bool) (Record, error) {
	if r == nil || !r.built || !validIntent(in) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := [2]string{in.Workspace, in.ID}
	record, exists := r.records[key]
	if !exists {
		return Record{}, ErrNotFound
	}
	action, state := RejectAction, Rejected
	if confirm {
		action, state = ConfirmAction, Eligible
	}
	if err := r.authority.Check(record.Evidence.Scope, in.Actor, action); err != nil {
		return Record{}, err
	}
	fresh, err := r.current.Compare(in.Workspace, record.Evidence.Run)
	if err != nil {
		return Record{}, err
	}
	if !validEvidence(fresh, in.Workspace, record.Evidence.Run) {
		return Record{}, ErrInvalid
	}
	if in.Digest != record.Evidence.Digest || !sameEvidence(record.Evidence, fresh) {
		return Record{}, ErrConflict
	}
	publication := r.publications[record.Evidence.Scope]
	if record.Decision != nil {
		if record.State != state || !sameIntent(record.Decision, in) {
			return Record{}, ErrConflict
		}
		if confirm && (publication.Record != record.ID || publication.Revision != record.Decision.ScopeRevision || publication.Digest != in.Digest) {
			return Record{}, ErrConflict
		}
		return clone(record), nil
	}
	if record.State != Awaiting && record.State != Blocked {
		return Record{}, ErrRejected
	}
	if publication.Revision != in.ExpectedScope {
		return Record{}, ErrConflict
	}
	if confirm && (record.State != Awaiting || fresh.Verdict != Match) {
		return Record{}, ErrRejected
	}
	if confirm && publication.Revision == MaxRevision {
		return Record{}, ErrLimit
	}
	at, err := r.instant()
	if err != nil {
		return Record{}, err
	}
	revision := publication.Revision
	if confirm {
		revision++
	}
	record.State = state
	record.Decision = &Decision{Actor: in.Actor, Rationale: in.Rationale, Digest: in.Digest, ExpectedScope: in.ExpectedScope, ScopeRevision: revision, At: at}
	if confirm {
		r.publications[record.Evidence.Scope] = Publication{Scope: record.Evidence.Scope, Revision: revision, Record: record.ID, Digest: in.Digest}
	}
	r.records[key] = record
	return clone(record), nil
}

func (r *Registry) Confirm(in Intent) (Record, error) { return r.decide(in, true) }
func (r *Registry) Reject(in Intent) (Record, error)  { return r.decide(in, false) }

func (r *Registry) Record(workspace, id string) (Record, error) {
	if r == nil || !r.built || !validID(workspace) || !validID(id) {
		return Record{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, exists := r.records[[2]string{workspace, id}]
	if !exists {
		return Record{}, ErrNotFound
	}
	return clone(record), nil
}

func (r *Registry) Current(scope Scope) (Publication, error) {
	if r == nil || !r.built || !validScope(scope) {
		return Publication{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	publication, exists := r.publications[scope]
	if !exists {
		return Publication{}, ErrNotFound
	}
	return publication, nil
}
