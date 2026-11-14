package erpledger

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type connection struct {
	view ConnectionView
	caps map[string]bool
}

type mappingKey struct {
	connection, kind, platform string
}

type surveyKey struct {
	connection, customer, period string
}

type SurveyView struct {
	Survey Survey
	Author string
	At     time.Time
}

type space struct {
	connections  map[string]*connection
	mappings     map[mappingKey]*MappingView
	mappingCount int
	anchors      map[string]*Anchor
	anchorAuthor map[string]string
	covering     map[string][]*Anchor
	surveys      map[surveyKey]*SurveyView
	submissions  map[string]*submission
	bySource     map[string][]*submission
	externals    map[string]*submission
	documents    map[string]*document
	observations map[string]*ObservationView
	orphans      []ObservationView
}

type Registry struct {
	mu         sync.Mutex
	numbers    decimal.Profile
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(numbers decimal.Profile, limits Limits, clock func() time.Time) (*Registry, error) {
	if !numbers.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{numbers: numbers, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
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

func (c Connection) valid() bool {
	if !validID(c.ID) || !validID(c.Entity) || !validID(c.System) || c.Transport != TransportAPI && c.Transport != TransportFile && c.Transport != TransportDB ||
		len(c.Capabilities) == 0 || !validCurrency(c.Currency) || c.Scale < 0 || c.Scale > maxScale ||
		c.Interval < minInterval || c.Interval > maxInterval || !validID(c.Owner) {
		return false
	}
	if (c.Transport == TransportFile) != (c.Template != "") || c.Template != "" && !validID(c.Template) {
		return false
	}
	seen := map[string]bool{}
	for _, name := range c.Capabilities {
		if !capabilities[name] || seen[name] {
			return false
		}
		seen[name] = true
	}
	return true
}

func sameConnection(a, b Connection) bool {
	return a.ID == b.ID && a.Entity == b.Entity && a.System == b.System && a.Transport == b.Transport &&
		strings.Join(a.Capabilities, ",") == strings.Join(b.Capabilities, ",") && a.Currency == b.Currency && a.Scale == b.Scale &&
		a.Interval == b.Interval && a.Owner == b.Owner && a.Template == b.Template
}

func (c *connection) copy() ConnectionView {
	v := c.view
	v.Connection.Capabilities = append([]string(nil), v.Connection.Capabilities...)
	return v
}

func (r *Registry) Connect(workspace string, c Connection, actor Actor) (ConnectionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !c.valid() || !actor.valid() {
		return ConnectionView{}, ErrInvalid
	}
	c.Capabilities = append([]string(nil), c.Capabilities...)
	sort.Strings(c.Capabilities)
	s := r.workspaces[workspace]
	if s != nil {
		if seen := s.connections[c.ID]; seen != nil {
			if sameConnection(seen.view.Connection, c) && actor.is(seen.view.Author) {
				return seen.copy(), nil
			}
			return ConnectionView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return ConnectionView{}, err
	}
	if c.Transport == TransportDB {
		return ConnectionView{}, refuse(ReasonDirectDatabase)
	}
	if s == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return ConnectionView{}, ErrLimit
		}
		s = &space{
			connections: map[string]*connection{}, mappings: map[mappingKey]*MappingView{}, anchors: map[string]*Anchor{},
			anchorAuthor: map[string]string{}, covering: map[string][]*Anchor{}, surveys: map[surveyKey]*SurveyView{},
			submissions: map[string]*submission{}, bySource: map[string][]*submission{}, externals: map[string]*submission{},
			documents: map[string]*document{}, observations: map[string]*ObservationView{},
		}
		r.workspaces[workspace] = s
	} else if len(s.connections) >= r.limits.Connections {
		return ConnectionView{}, ErrLimit
	}
	made := &connection{view: ConnectionView{Connection: c, Author: actor.ID, At: now}, caps: map[string]bool{}}
	for _, name := range c.Capabilities {
		made.caps[name] = true
	}
	s.connections[c.ID] = made
	return made.copy(), nil
}

func (r *Registry) VerifyTemplate(workspace, id, evidence string, actor Actor) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(evidence) || !actor.valid() {
		return ConnectionView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil || s.connections[id] == nil {
		return ConnectionView{}, ErrNotFound
	}
	c := s.connections[id]
	if c.view.Verified {
		if actor.is(c.view.Verifier) && c.view.Evidence == evidence {
			return c.copy(), nil
		}
		return ConnectionView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return ConnectionView{}, err
	}
	if c.view.Connection.Transport != TransportFile {
		return ConnectionView{}, refuse(ReasonUnsupported)
	}
	if actor.ID == c.view.Author {
		return ConnectionView{}, refuse(ReasonSelfApproval)
	}
	c.view.Verified, c.view.Evidence, c.view.Verifier = true, evidence, actor.ID
	return c.copy(), nil
}

func (r *Registry) ConnectionOf(workspace, id string) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.connections[id] == nil {
		return ConnectionView{}, ErrNotFound
	}
	return s.connections[id].copy(), nil
}

func (r *Registry) connection(workspace, id string) (*space, *connection) {
	s := r.workspaces[workspace]
	if s == nil {
		return nil, nil
	}
	return s, s.connections[id]
}

func (r *Registry) Map(workspace string, m Mapping, actor Actor) (MappingView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(m.Connection) || !mappingKinds[m.Kind] || !validID(m.Platform) || !validID(m.External) || !actor.valid() {
		return MappingView{}, ErrInvalid
	}
	s, c := r.connection(workspace, m.Connection)
	if c == nil {
		return MappingView{}, refuse(ReasonUnknownConnection)
	}
	k := mappingKey{m.Connection, m.Kind, m.Platform}
	if seen := s.mappings[k]; seen != nil && seen.Mapping == m {
		if actor.is(seen.Author) {
			return *seen, nil
		}
		return MappingView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return MappingView{}, err
	}
	if !c.caps[CommandMapping] {
		return MappingView{}, refuse(ReasonUnsupported)
	}
	if s.mappingCount >= r.limits.Mappings {
		return MappingView{}, ErrLimit
	}
	revision := 1
	if seen := s.mappings[k]; seen != nil {
		revision = seen.Revision + 1
	}
	s.mappings[k] = &MappingView{Mapping: m, Revision: revision, Author: actor.ID, At: now}
	s.mappingCount++
	return *s.mappings[k], nil
}

func (a Anchor) valid() bool {
	if !validID(a.ID) || !validID(a.Connection) || !anchorKinds[a.Kind] || !validID(a.External) || !validID(a.Customer) || len(a.Sources) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, source := range a.Sources {
		if !validID(source) || seen[source] {
			return false
		}
		seen[source] = true
	}
	return true
}

func sameAnchor(a, b Anchor) bool {
	return a.ID == b.ID && a.Connection == b.Connection && a.Kind == b.Kind && a.External == b.External && a.Customer == b.Customer &&
		strings.Join(a.Sources, ",") == strings.Join(b.Sources, ",")
}

func (r *Registry) Anchor(workspace string, a Anchor, actor Actor) (Anchor, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !a.valid() || !actor.valid() {
		return Anchor{}, ErrInvalid
	}
	s, c := r.connection(workspace, a.Connection)
	if c == nil {
		return Anchor{}, refuse(ReasonUnknownConnection)
	}
	if seen := s.anchors[a.ID]; seen != nil {
		if sameAnchor(*seen, a) && actor.is(s.anchorAuthor[a.ID]) {
			return copyAnchor(*seen), nil
		}
		return Anchor{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return Anchor{}, err
	}
	if len(s.anchors) >= r.limits.Anchors {
		return Anchor{}, ErrLimit
	}
	stored := copyAnchor(a)
	s.anchors[a.ID] = &stored
	s.anchorAuthor[a.ID] = actor.ID
	for _, source := range stored.Sources {
		s.covering[source] = append(s.covering[source], &stored)
		for _, purpose := range []string{PurposeInvoice, PurposePayable} {
			if !blocks(purpose, a.Kind) {
				continue
			}
			for _, sub := range s.bySource[sourceKey(purpose, source)] {
				if sub.live() {
					sub.view.Diverged = true
				}
			}
		}
	}
	return copyAnchor(stored), nil
}

func copyAnchor(a Anchor) Anchor {
	a.Sources = append([]string(nil), a.Sources...)
	return a
}

func (r *Registry) Survey(workspace string, v Survey, actor Actor) (SurveyView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(v.Connection) || !validID(v.Customer) || !validMonth(v.Period) || !outcomes[v.Outcome] || !validID(v.Evidence) || !actor.valid() {
		return SurveyView{}, ErrInvalid
	}
	s, c := r.connection(workspace, v.Connection)
	if c == nil {
		return SurveyView{}, refuse(ReasonUnknownConnection)
	}
	k := surveyKey{v.Connection, v.Customer, v.Period}
	if seen := s.surveys[k]; seen != nil && seen.Survey == v && actor.is(seen.Author) {
		return *seen, nil
	}
	if err := actor.refusal(); err != nil {
		return SurveyView{}, err
	}
	if s.surveys[k] == nil && len(s.surveys) >= r.limits.Surveys {
		return SurveyView{}, ErrLimit
	}
	s.surveys[k] = &SurveyView{Survey: v, Author: actor.ID, At: now}
	return *s.surveys[k], nil
}
