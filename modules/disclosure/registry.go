package disclosure

import (
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	stateProposed   = "proposed"
	stateActive     = "active"
	stateSuperseded = "superseded"
	stateSuspended  = "suspended"
	stateReplaced   = "replaced"
	stateRevoked    = "revoked"
)

type policy struct {
	revisions []*PolicyView
	pending   *PolicyView
	active    *PolicyView
}

type grant struct {
	revisions []*GrantView
	pending   *GrantView
	active    *GrantView
	revoked   bool
	revoker   string
}

type space struct {
	fields       map[string]*FieldView
	combinations map[string]*CombinationView
	pendingCombo map[string]*CombinationView
	policies     map[string]*policy
	grants       map[string]*grant
	projections  map[string]*Audit
	refs         map[string]string
	packages     map[string]*PackageView
	links        map[string]*LinkView
	linkCount    int
}

type linkRef struct {
	workspace string
	link      *LinkView
}

type Registry struct {
	mu         sync.Mutex
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
	tokens     map[string]linkRef
	recipients map[string]bool
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Keys == nil || config.Random == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}, tokens: map[string]linkRef{}, recipients: map[string]bool{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Fields, maxFields}, {l.Policies, maxPolicies}, {l.Recipients, maxRecipients},
		{l.Projections, maxProjections}, {l.Packages, maxPackages}, {l.Links, maxLinks}, {l.Rows, maxRows}, {l.Combinations, maxCombinations}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return true
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

func refuse(reason string) error { return Refusal{Reason: reason} }

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func validValue(text string) bool {
	return len(text) <= maxValueBytes && utf8.ValidString(text)
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (a Actor) human() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (a Actor) notAssistant() error {
	if a.Kind == KindAssistant {
		return refuse(ReasonAINotAllowed)
	}
	return nil
}

func (r *Registry) staff(actor Actor) error {
	if r.recipients[actor.ID] {
		return refuse(ReasonRecipientActor)
	}
	return nil
}

func (r *Registry) space(workspace string, create bool) (*space, error) {
	s := r.workspaces[workspace]
	if s == nil && create {
		if len(r.workspaces) >= r.limits.Workspaces {
			return nil, ErrLimit
		}
		s = &space{fields: map[string]*FieldView{}, combinations: map[string]*CombinationView{}, pendingCombo: map[string]*CombinationView{},
			policies: map[string]*policy{}, grants: map[string]*grant{}, projections: map[string]*Audit{}, refs: map[string]string{},
			packages: map[string]*PackageView{}, links: map[string]*LinkView{}}
		r.workspaces[workspace] = s
	}
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *Registry) Classify(workspace string, f Field, actor Actor) (FieldView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(f.Name) || !classes[f.Class] || !actor.valid() {
		return FieldView{}, ErrInvalid
	}
	if s := r.workspaces[workspace]; s != nil {
		if seen := s.fields[f.Name]; seen != nil {
			if seen.Field == f && actor.is(seen.Author) {
				return *seen, nil
			}
			return FieldView{}, ErrConflict
		}
	}
	if err := actor.human(); err != nil {
		return FieldView{}, err
	}
	if err := r.staff(actor); err != nil {
		return FieldView{}, err
	}
	if f.Locator && f.Class != ClassPublic && f.Class != ClassCustomer {
		return FieldView{}, refuse(ReasonLocatorClass)
	}
	s, err := r.space(workspace, true)
	if err != nil {
		return FieldView{}, err
	}
	if len(s.fields) >= r.limits.Fields {
		return FieldView{}, ErrLimit
	}
	view := &FieldView{Field: f, Author: actor.ID, At: now}
	s.fields[f.Name] = view
	return *view, nil
}

func sortedUnique(list []string, min, max int) ([]string, bool) {
	if len(list) < min || len(list) > max {
		return nil, false
	}
	out := append([]string(nil), list...)
	sort.Strings(out)
	for i, v := range out {
		if !validID(v) || i > 0 && out[i-1] == v {
			return nil, false
		}
	}
	return out, true
}

func copyCombination(v CombinationView) CombinationView {
	v.Combination.Fields = append([]string(nil), v.Combination.Fields...)
	return v
}

func (r *Registry) ProposeCombination(workspace string, c Combination, actor Actor) (CombinationView, error) {
	now := r.enter()
	defer r.leave()
	fields, ok := sortedUnique(c.Fields, minCombination, maxCombination)
	if !validID(workspace) || !validID(c.ID) || !ok || !actor.valid() {
		return CombinationView{}, ErrInvalid
	}
	c.Fields = fields
	s, err := r.space(workspace, false)
	if err != nil {
		return CombinationView{}, err
	}
	if seen := s.combinations[c.ID]; seen != nil {
		return CombinationView{}, ErrConflict
	}
	if seen := s.pendingCombo[c.ID]; seen != nil {
		if strings.Join(seen.Combination.Fields, ",") == strings.Join(fields, ",") && actor.is(seen.Proposer) {
			return copyCombination(*seen), nil
		}
		return CombinationView{}, ErrConflict
	}
	if err := actor.human(); err != nil {
		return CombinationView{}, err
	}
	if err := r.staff(actor); err != nil {
		return CombinationView{}, err
	}
	for _, name := range fields {
		if s.fields[name] == nil {
			return CombinationView{}, refuse(ReasonUnclassifiedField)
		}
	}
	if len(s.pendingCombo)+len(s.combinations) >= r.limits.Combinations {
		return CombinationView{}, ErrLimit
	}
	view := &CombinationView{Combination: c, Proposer: actor.ID, At: now}
	s.pendingCombo[c.ID] = view
	return copyCombination(*view), nil
}

func (r *Registry) ConfirmCombination(workspace, id string, actor Actor) (CombinationView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return CombinationView{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return CombinationView{}, err
	}
	if seen := s.combinations[id]; seen != nil {
		if actor.is(seen.Confirmer) {
			return copyCombination(*seen), nil
		}
		return CombinationView{}, ErrConflict
	}
	pending := s.pendingCombo[id]
	if pending == nil {
		return CombinationView{}, ErrNotFound
	}
	if err := actor.human(); err != nil {
		return CombinationView{}, err
	}
	if err := r.staff(actor); err != nil {
		return CombinationView{}, err
	}
	if actor.ID == pending.Proposer {
		return CombinationView{}, refuse(ReasonSelfApproval)
	}
	pending.Confirmer, pending.At = actor.ID, now
	s.combinations[id] = pending
	delete(s.pendingCombo, id)
	for _, p := range s.policies {
		if p.active != nil && s.violates(p.active.Policy) != "" {
			p.active.State = stateSuspended
		}
	}
	return copyCombination(*pending), nil
}

func (s *space) violates(p Policy) string {
	for _, role := range []string{RoleCustomer, RolePartner, RoleAuditor} {
		set := map[string]bool{}
		for _, name := range p.Roles[role] {
			set[name] = true
		}
		for _, c := range s.combinations {
			all := true
			for _, name := range c.Combination.Fields {
				all = all && set[name]
			}
			if all {
				return ReasonInferenceConflict
			}
		}
	}
	return ""
}

func (s *space) check(p Policy) error {
	for _, role := range []string{RoleCustomer, RolePartner, RoleAuditor} {
		for _, name := range p.Roles[role] {
			f := s.fields[name]
			if f == nil {
				return refuse(ReasonUnclassifiedField)
			}
			if !allowed[role][f.Field.Class] {
				return refuse(ReasonClassNotAllowed)
			}
		}
	}
	if reason := s.violates(p); reason != "" {
		return refuse(reason)
	}
	return nil
}

func copyPolicy(v PolicyView) PolicyView {
	roles := map[string][]string{}
	for role, list := range v.Policy.Roles {
		roles[role] = append([]string(nil), list...)
	}
	v.Policy.Roles = roles
	return v
}

func samePolicy(a, b Policy) bool {
	if len(a.Roles) != len(b.Roles) {
		return false
	}
	for role, list := range a.Roles {
		if strings.Join(list, ",") != strings.Join(b.Roles[role], ",") {
			return false
		}
	}
	return true
}

func (r *Registry) Propose(workspace string, p Policy, actor Actor) (PolicyView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(p.ID) || len(p.Roles) == 0 || !actor.valid() {
		return PolicyView{}, ErrInvalid
	}
	roleLists := map[string][]string{}
	for role, list := range p.Roles {
		sorted, ok := sortedUnique(list, 1, maxFields)
		if !roles[role] || !ok {
			return PolicyView{}, ErrInvalid
		}
		roleLists[role] = sorted
	}
	p.Roles = roleLists
	s, err := r.space(workspace, false)
	if err != nil {
		return PolicyView{}, err
	}
	pol := s.policies[p.ID]
	if pol != nil && pol.pending != nil && samePolicy(pol.pending.Policy, p) && actor.is(pol.pending.Proposer) {
		return copyPolicy(*pol.pending), nil
	}
	if err := actor.human(); err != nil {
		return PolicyView{}, err
	}
	if err := r.staff(actor); err != nil {
		return PolicyView{}, err
	}
	if err := s.check(p); err != nil {
		return PolicyView{}, err
	}
	if pol == nil {
		if len(s.policies) >= r.limits.Policies {
			return PolicyView{}, ErrLimit
		}
		pol = &policy{}
		s.policies[p.ID] = pol
	}
	if pol.pending != nil {
		pol.pending.State = stateReplaced
	}
	view := &PolicyView{Policy: p, Revision: len(pol.revisions) + 1, State: stateProposed, Proposer: actor.ID, At: now}
	pol.revisions = append(pol.revisions, view)
	pol.pending = view
	return copyPolicy(*view), nil
}

func (r *Registry) Confirm(workspace, id string, revision int, actor Actor) (PolicyView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || revision < 1 || !actor.valid() {
		return PolicyView{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return PolicyView{}, err
	}
	pol := s.policies[id]
	if pol == nil {
		return PolicyView{}, ErrNotFound
	}
	if pol.active != nil && pol.active.Revision == revision && actor.is(pol.active.Confirmer) {
		return copyPolicy(*pol.active), nil
	}
	if err := actor.human(); err != nil {
		return PolicyView{}, err
	}
	if err := r.staff(actor); err != nil {
		return PolicyView{}, err
	}
	if pol.pending == nil || pol.pending.Revision != revision {
		return PolicyView{}, refuse(ReasonStaleProposal)
	}
	if actor.ID == pol.pending.Proposer {
		return PolicyView{}, refuse(ReasonSelfApproval)
	}
	if err := s.check(pol.pending.Policy); err != nil {
		return PolicyView{}, err
	}
	if pol.active != nil {
		pol.active.State = stateSuperseded
	}
	pol.pending.State, pol.pending.Confirmer, pol.pending.At = stateActive, actor.ID, now
	pol.active, pol.pending = pol.pending, nil
	return copyPolicy(*pol.active), nil
}

func (r *Registry) PolicyOf(workspace, id string, revision int) (PolicyView, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.policies[id] == nil || revision < 1 || revision > len(s.policies[id].revisions) {
		return PolicyView{}, ErrNotFound
	}
	return copyPolicy(*s.policies[id].revisions[revision-1]), nil
}

func copyGrant(v GrantView) GrantView {
	v.Recipient.Customers = append([]string(nil), v.Recipient.Customers...)
	return v
}

func (r *Registry) Grant(workspace string, rec Recipient, actor Actor) (GrantView, error) {
	now := r.enter()
	defer r.leave()
	customers, ok := sortedUnique(rec.Customers, 1, 1000)
	if !validID(workspace) || !validID(rec.ID) || !roles[rec.Role] || !ok || !validID(rec.Policy) || !actor.valid() {
		return GrantView{}, ErrInvalid
	}
	rec.Customers = customers
	s, err := r.space(workspace, false)
	if err != nil {
		return GrantView{}, err
	}
	g := s.grants[rec.ID]
	if g != nil && g.pending != nil && g.pending.Recipient.Role == rec.Role && g.pending.Recipient.Policy == rec.Policy && strings.Join(g.pending.Recipient.Customers, ",") == strings.Join(customers, ",") && actor.is(g.pending.Proposer) {
		return copyGrant(*g.pending), nil
	}
	if err := actor.human(); err != nil {
		return GrantView{}, err
	}
	if err := r.staff(actor); err != nil {
		return GrantView{}, err
	}
	if g != nil && g.revoked {
		return GrantView{}, refuse(ReasonRecipientRevoked)
	}
	if rec.ID == actor.ID {
		return GrantView{}, refuse(ReasonSelfApproval)
	}
	if g == nil {
		if len(s.grants) >= r.limits.Recipients {
			return GrantView{}, ErrLimit
		}
		g = &grant{}
		s.grants[rec.ID] = g
		r.recipients[rec.ID] = true
	}
	if g.pending != nil {
		g.pending.State = stateReplaced
	}
	view := &GrantView{Recipient: rec, Revision: len(g.revisions) + 1, State: stateProposed, Proposer: actor.ID, At: now}
	g.revisions = append(g.revisions, view)
	g.pending = view
	return copyGrant(*view), nil
}

func (r *Registry) ConfirmGrant(workspace, id string, revision int, actor Actor) (GrantView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || revision < 1 || !actor.valid() {
		return GrantView{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return GrantView{}, err
	}
	g := s.grants[id]
	if g == nil {
		return GrantView{}, ErrNotFound
	}
	if g.active != nil && g.active.Revision == revision && actor.is(g.active.Confirmer) {
		return copyGrant(*g.active), nil
	}
	if err := actor.human(); err != nil {
		return GrantView{}, err
	}
	if err := r.staff(actor); err != nil {
		return GrantView{}, err
	}
	if g.revoked {
		return GrantView{}, refuse(ReasonRecipientRevoked)
	}
	if g.pending == nil || g.pending.Revision != revision {
		return GrantView{}, refuse(ReasonStaleProposal)
	}
	if actor.ID == g.pending.Proposer || actor.ID == id {
		return GrantView{}, refuse(ReasonSelfApproval)
	}
	if g.active != nil {
		g.active.State = stateSuperseded
	}
	g.pending.State, g.pending.Confirmer, g.pending.At = stateActive, actor.ID, now
	g.active, g.pending = g.pending, nil
	return copyGrant(*g.active), nil
}

func (r *Registry) RevokeRecipient(workspace, id string, actor Actor) (GrantView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return GrantView{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return GrantView{}, err
	}
	g := s.grants[id]
	if g == nil {
		return GrantView{}, ErrNotFound
	}
	last := g.revisions[len(g.revisions)-1]
	if g.revoked {
		if actor.is(g.revoker) {
			return copyGrant(*last), nil
		}
		return GrantView{}, refuse(ReasonRecipientRevoked)
	}
	if err := actor.human(); err != nil {
		return GrantView{}, err
	}
	if err := r.staff(actor); err != nil {
		return GrantView{}, err
	}
	g.revoked, g.revoker = true, actor.ID
	for _, v := range g.revisions {
		if v.State == stateActive || v.State == stateProposed {
			v.State = stateRevoked
		}
	}
	last.RevokedBy, last.RevokedAt = actor.ID, now
	g.pending = nil
	return copyGrant(*last), nil
}
