package auth

import (
	"crypto/rand"
	"math"
	"sync"
	"time"
)

type versioned[T any] struct {
	value   T
	version uint64
	closed  bool
}

type Registry struct {
	mu           sync.RWMutex
	installation string
	instance     [16]byte
	clock        func() time.Time
	kinds        map[string]struct{}
	fields       map[string]FieldClass
	available    bool
	generation   uint64
	subjects     map[string]versioned[Subject]
	clients      map[string]versioned[Client]
	grants       map[string]versioned[Grant]
	delegations  map[string]versioned[Delegation]
	sessions     map[string]versioned[SupportSession]
	source       SubjectSource
	grantJoined  map[string]uint64
	chainJoined  map[string][2]uint64
	actorJoined  map[string]uint64
}

func NewRegistryWithSubjects(installationID string, catalog Catalog, clock func() time.Time, source SubjectSource) (*Registry, error) {
	if source == nil {
		return nil, ErrInvalid
	}
	r, err := NewRegistry(installationID, catalog, clock)
	if err != nil {
		return nil, err
	}
	r.source = source
	return r, nil
}

func (r *Registry) subject(id string) (Subject, bool) {
	if r.source != nil {
		return r.source.Subject(id)
	}
	found := r.subjects[id]
	return found.value, found.version != 0
}

func (r *Registry) member(workspaceID, id string) (uint64, bool) {
	if r.source == nil {
		return 0, true
	}
	return r.source.Member(workspaceID, id)
}

func (r *Registry) memberWith(workspaceID, id string, joined uint64) bool {
	current, active := r.member(workspaceID, id)
	return active && current == joined
}

func (r *Registry) subjectsGeneration() uint64 {
	if r.source == nil {
		return 0
	}
	return r.source.Generation()
}

func NewRegistry(installationID string, catalog Catalog, clock func() time.Time) (*Registry, error) {
	if !validID(installationID) || clock == nil || len(catalog.Kinds) == 0 {
		return nil, ErrInvalid
	}
	r := &Registry{
		installation: installationID, clock: clock, available: true, generation: 1,
		kinds: make(map[string]struct{}), fields: make(map[string]FieldClass),
		subjects: make(map[string]versioned[Subject]), clients: make(map[string]versioned[Client]),
		grants: make(map[string]versioned[Grant]), delegations: make(map[string]versioned[Delegation]),
		sessions:    make(map[string]versioned[SupportSession]),
		grantJoined: make(map[string]uint64), chainJoined: make(map[string][2]uint64), actorJoined: make(map[string]uint64),
	}
	for _, kind := range catalog.Kinds {
		if _, repeated := r.kinds[kind]; repeated || !validID(kind) {
			return nil, ErrInvalid
		}
		r.kinds[kind] = struct{}{}
	}
	for field, class := range catalog.Fields {
		if !validID(field) || (class != General && class != Cost) {
			return nil, ErrInvalid
		}
		r.fields[field] = class
	}
	if _, err := rand.Read(r.instance[:]); err != nil {
		return nil, ErrUnavailable
	}
	return r, nil
}

func next(current, expected uint64) (uint64, error) {
	if current != expected || current == math.MaxUint64 {
		return 0, ErrConflict
	}
	return current + 1, nil
}

func (r *Registry) changed() { r.generation++ }

func (r *Registry) SetAvailable(available bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.clock == nil || r.available == available {
		return
	}
	r.available = available
	r.changed()
}

func (r *Registry) PutSubject(subject Subject, expected uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.subjects == nil {
		return 0, ErrUnavailable
	}
	if r.source != nil {
		return 0, ErrInvalid
	}
	if !validID(subject.ID) || (subject.Kind != Human && subject.Kind != Service) ||
		(subject.State != Active && subject.State != Suspended) || subject.Epoch == 0 {
		return 0, ErrInvalid
	}
	old := r.subjects[subject.ID]
	version, err := next(old.version, expected)
	if err != nil {
		return 0, err
	}
	if old.version != 0 && (old.value.Kind != subject.Kind || subject.Epoch < old.value.Epoch ||
		(old.value.State != Active && subject.State == Active && subject.Epoch <= old.value.Epoch)) {
		return 0, ErrInvalid
	}
	r.subjects[subject.ID] = versioned[Subject]{value: subject, version: version}
	r.changed()
	return version, nil
}

func (r *Registry) PutClient(client Client, expected uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.clients == nil {
		return 0, ErrUnavailable
	}
	if !validID(client.ID) || (client.State != Active && client.State != Suspended) {
		return 0, ErrInvalid
	}
	version, err := next(r.clients[client.ID].version, expected)
	if err != nil {
		return 0, err
	}
	r.clients[client.ID] = versioned[Client]{value: client, version: version}
	r.changed()
	return version, nil
}

func (r *Registry) activeHuman(id string) bool {
	subject, known := r.subject(id)
	return known && subject.State == Active && subject.Kind == Human
}

func (r *Registry) validTerms(terms Terms) bool {
	if _, known := actions[terms.Action]; !known {
		return false
	}
	if _, known := r.kinds[terms.Kind]; !known {
		return false
	}
	if !validID(terms.Scope.WorkspaceID) || !terms.Scope.Customer.valid() || !terms.Scope.LegalEntity.valid() ||
		!terms.Scope.Contract.valid() || !terms.Period.valid() {
		return false
	}
	return r.validFields(terms.Action, terms.Fields)
}

func (r *Registry) validFields(action Action, fields []string) bool {
	if !fieldAction(action) {
		return len(fields) == 0
	}
	if len(fields) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		class, known := r.fields[field]
		if _, repeated := seen[field]; repeated || !known {
			return false
		}
		if (action == View && class != General) || (action == ViewCost && class != Cost) {
			return false
		}
		seen[field] = struct{}{}
	}
	return true
}

func copyTerms(terms Terms) Terms {
	terms.Fields = append([]string(nil), terms.Fields...)
	return terms
}

func (r *Registry) AddGrant(grant Grant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	grant.Terms = copyTerms(grant.Terms)
	if !validID(grant.ID) || !r.validTerms(grant.Terms) || !grant.Validity.valid() || (grant.SubjectID == "") == (grant.ClientID == "") {
		return ErrInvalid
	}
	generation := r.subjectsGeneration()
	joined := uint64(0)
	if grant.SubjectID != "" {
		subject, known := r.subject(grant.SubjectID)
		if !known || subject.State != Active || (humanOnly(grant.Terms.Action) && subject.Kind != Human) {
			return ErrInvalid
		}
		active := false
		if joined, active = r.member(grant.Terms.Scope.WorkspaceID, grant.SubjectID); !active {
			return ErrInvalid
		}
	} else {
		client := r.clients[grant.ClientID]
		if client.version == 0 || client.value.State != Active || humanOnly(grant.Terms.Action) {
			return ErrInvalid
		}
	}
	if r.subjectsGeneration() != generation {
		return ErrChanged
	}
	if r.grants[grant.ID].version != 0 {
		return ErrConflict
	}
	r.grants[grant.ID] = versioned[Grant]{value: grant, version: 1}
	r.grantJoined[grant.ID] = joined
	r.changed()
	return nil
}

func (r *Registry) RevokeGrant(id string, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := r.grants[id]
	if found.version == 0 {
		return ErrNotFound
	}
	if found.closed {
		return ErrConflict
	}
	version, err := next(found.version, expected)
	if err != nil {
		return err
	}
	found.version, found.closed = version, true
	r.grants[id] = found
	r.changed()
	return nil
}

func contains(fields []string, field string) bool {
	for _, candidate := range fields {
		if candidate == field {
			return true
		}
	}
	return false
}

func (r *Registry) backedDirectly(delegatorID string, terms Terms, now time.Time) bool {
	fields := terms.Fields
	if !fieldAction(terms.Action) {
		fields = []string{""}
	}
	for _, field := range fields {
		backed := false
		for id, grant := range r.grants {
			g := grant.value
			if !grant.closed && g.SubjectID == delegatorID && g.Validity.at(now) && g.Terms.Action == terms.Action &&
				g.Terms.Kind == terms.Kind && terms.Scope.within(g.Terms.Scope) && terms.Period.within(g.Terms.Period) &&
				(field == "" || contains(g.Terms.Fields, field)) && r.memberWith(g.Terms.Scope.WorkspaceID, delegatorID, r.grantJoined[id]) {
				backed = true
				break
			}
		}
		if !backed {
			return false
		}
	}
	return true
}

func (r *Registry) holdsInWorkspace(subjectID, workspaceID string, now time.Time) bool {
	for id, grant := range r.grants {
		if !grant.closed && grant.value.SubjectID == subjectID && grant.value.Terms.Scope.WorkspaceID == workspaceID && grant.value.Validity.at(now) &&
			r.memberWith(workspaceID, subjectID, r.grantJoined[id]) {
			return true
		}
	}
	return false
}

func (r *Registry) chainSubjects(delegation Delegation) (map[string]struct{}, bool) {
	subjects := map[string]struct{}{delegation.DelegateID: {}}
	length := 1
	for current := delegation; ; length++ {
		if _, repeated := subjects[current.DelegatorID]; repeated || length > maxChain {
			return nil, false
		}
		subjects[current.DelegatorID] = struct{}{}
		if current.ParentID == "" {
			return subjects, true
		}
		parent := r.delegations[current.ParentID]
		if parent.version == 0 || parent.value.DelegateID != current.DelegatorID {
			return nil, false
		}
		current = parent.value
	}
}

func (r *Registry) AddDelegation(delegation Delegation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	terms := copyTerms(delegation.Terms)
	generation := r.subjectsGeneration()
	if !validID(delegation.ID) || !r.validTerms(terms) || !delegation.Validity.valid() || humanOnly(terms.Action) ||
		delegation.DelegatorID == delegation.DelegateID || !r.activeHuman(delegation.DelegatorID) || !r.activeHuman(delegation.DelegateID) {
		return ErrInvalid
	}
	delegatorJoined, delegatorActive := r.member(terms.Scope.WorkspaceID, delegation.DelegatorID)
	delegateJoined, delegateActive := r.member(terms.Scope.WorkspaceID, delegation.DelegateID)
	if !delegatorActive || !delegateActive {
		return ErrInvalid
	}
	now := r.clock()
	if delegation.ParentID == "" {
		if !r.backedDirectly(delegation.DelegatorID, terms, now) {
			return ErrNotBacked
		}
	} else {
		parent := r.delegations[delegation.ParentID]
		p := parent.value
		if parent.version == 0 || parent.closed || !p.Validity.at(now) || p.DelegateID != delegation.DelegatorID || !p.AllowRedelegation ||
			p.Terms.Action != terms.Action || p.Terms.Kind != terms.Kind || !terms.Scope.within(p.Terms.Scope) ||
			!terms.Period.within(p.Terms.Period) || !delegation.Validity.within(p.Validity) {
			return ErrNotBacked
		}
		for _, field := range terms.Fields {
			if !contains(p.Terms.Fields, field) {
				return ErrNotBacked
			}
		}
	}
	if _, ok := r.chainSubjects(delegation); !ok {
		return ErrInvalid
	}
	if !r.holdsInWorkspace(delegation.DelegateID, terms.Scope.WorkspaceID, now) {
		return ErrInvalid
	}
	if r.subjectsGeneration() != generation {
		return ErrChanged
	}
	if r.delegations[delegation.ID].version != 0 {
		return ErrConflict
	}
	delegation.Terms = terms
	r.delegations[delegation.ID] = versioned[Delegation]{value: delegation, version: 1}
	r.chainJoined[delegation.ID] = [2]uint64{delegatorJoined, delegateJoined}
	r.changed()
	return nil
}

func (r *Registry) RevokeDelegation(id string, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := r.delegations[id]
	if found.version == 0 {
		return ErrNotFound
	}
	if found.closed {
		return ErrConflict
	}
	version, err := next(found.version, expected)
	if err != nil {
		return err
	}
	found.version, found.closed = version, true
	r.delegations[id] = found
	r.changed()
	return nil
}

func (r *Registry) delegatesFor(actorID, approverID string, now time.Time) bool {
	for _, delegation := range r.delegations {
		if delegation.closed || delegation.value.DelegateID != approverID || !delegation.value.Validity.at(now) {
			continue
		}
		for current := delegation; ; {
			if current.value.DelegatorID == actorID {
				return true
			}
			if current.value.ParentID == "" {
				break
			}
			current = r.delegations[current.value.ParentID]
			if current.version == 0 || current.closed || !current.value.Validity.at(now) {
				break
			}
		}
	}
	return false
}

func (r *Registry) AddSupportSession(session SupportSession, approver Principal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.available {
		return ErrUnavailable
	}
	generation := r.subjectsGeneration()
	session.Fields = append([]string(nil), session.Fields...)
	if _, known := r.kinds[session.Kind]; !known || !validID(session.ID) || !validID(session.WorkspaceID) ||
		!validID(session.CustomerID) || session.Reason == "" || !session.Validity.valid() ||
		!r.validFields(View, session.Fields) || !validID(session.ActorID) {
		return ErrInvalid
	}
	now := r.clock()
	subject, _ := r.subject(approver.SubjectID)
	if approver.Installation != r.installation || approver.ClientID != "" || approver.SessionID != "" ||
		!r.activeHuman(approver.SubjectID) || subject.Epoch != approver.Epoch ||
		approver.SubjectID == session.ActorID || r.delegatesFor(session.ActorID, approver.SubjectID, now) {
		return ErrDenied
	}
	resource := Resource{WorkspaceID: session.WorkspaceID, CustomerID: session.CustomerID}
	if !r.direct(approver.SubjectID, "", ApproveSupportSession, session.Kind, resource, "", now) {
		return ErrDenied
	}
	actorJoined, actorActive := r.member(session.WorkspaceID, session.ActorID)
	if !r.activeHuman(session.ActorID) || !actorActive {
		return ErrInvalid
	}
	if r.subjectsGeneration() != generation {
		return ErrChanged
	}
	if r.sessions[session.ID].version != 0 {
		return ErrConflict
	}
	session.ApproverID = approver.SubjectID
	r.sessions[session.ID] = versioned[SupportSession]{value: session, version: 1}
	r.actorJoined[session.ID] = actorJoined
	r.changed()
	return nil
}

func (r *Registry) EndSupportSession(id string, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	found := r.sessions[id]
	if found.version == 0 {
		return ErrNotFound
	}
	if found.closed {
		return ErrConflict
	}
	version, err := next(found.version, expected)
	if err != nil {
		return err
	}
	found.version, found.closed = version, true
	r.sessions[id] = found
	r.changed()
	return nil
}
