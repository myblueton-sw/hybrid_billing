package events

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"
	"time"
)

type typeKey struct {
	name    string
	version string
}

type eventKey struct {
	source string
	id     string
}

type space struct {
	types        map[typeKey]*TypeView
	events       []Stored
	first        int
	floor        uint64
	sequence     uint64
	last         time.Time
	published    map[eventKey]uint64
	bindings     map[string]*Binding
	seen         map[eventKey]seenEvent
	queue        []eventKey
	queued       int
	revisions    map[subjectKey]revision
	reservations map[string]*reservation
	pending      map[eventKey]string
	subjects     map[subjectKey]string
	expiring     []*reservation
	expired      int
	fresh        int
}

type Registry struct {
	mu         sync.Mutex
	limits     Limits
	clock      func() time.Time
	now        time.Time
	tokens     uint64
	workspaces map[string]*space
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Types, maxTypes}, {l.Events, maxEvents}, {l.Page, maxPage}, {l.Sources, maxSources},
		{l.Seen, maxSeen}, {l.Revisions, maxRevisions}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return l.Retention >= minRetention && l.Retention <= maxRetention && l.InboxRetention >= minRetention && l.InboxRetention <= maxRetention &&
		l.Reservation >= minReservation && l.Reservation <= maxReservation
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

func validField(name string) bool {
	if name == "" || len(name) > maxFieldName || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for index := 1; index < len(name); index++ {
		c := name[index]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

func validSource(source string) bool {
	if source == "" || len(source) > maxSourceBytes {
		return false
	}
	for index := 0; index < len(source); index++ {
		if c := source[index]; c <= ' ' || c >= 0x7f || c == '@' || c == '\\' {
			return false
		}
	}
	return true
}

func parseRevision(text string) (uint64, bool) {
	if text == "" || len(text) > maxRevisionDigits || text[0] == '0' && len(text) > 1 {
		return 0, false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return 0, false
		}
	}
	value, err := strconv.ParseUint(text, 10, 64)
	return value, err == nil
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) human() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (a Actor) system() error {
	if a.Kind != KindSystem {
		return refuse(ReasonNotSystem)
	}
	return nil
}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{types: map[typeKey]*TypeView{}, published: map[eventKey]uint64{}, bindings: map[string]*Binding{}, seen: map[eventKey]seenEvent{},
		revisions: map[subjectKey]revision{}, reservations: map[string]*reservation{}, pending: map[eventKey]string{}, subjects: map[subjectKey]string{}}
	r.workspaces[workspace] = s
	return s, nil
}

func typeForm(t Type) bool {
	if !validID(t.Name) || !validID(t.SchemaVersion) || len(t.Fields) > maxFields {
		return false
	}
	seen := map[string]bool{}
	for _, field := range t.Fields {
		if !validField(field) || seen[field] {
			return false
		}
		seen[field] = true
	}
	return true
}

func copyType(t Type) Type {
	t.Fields = append([]string(nil), t.Fields...)
	sort.Strings(t.Fields)
	return t
}

func sameType(a, b Type) bool {
	a, b = copyType(a), copyType(b)
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for index := range a.Fields {
		if a.Fields[index] != b.Fields[index] {
			return false
		}
	}
	return true
}

func viewType(v *TypeView) TypeView {
	out := *v
	out.Type = copyType(v.Type)
	return out
}

func (r *Registry) ProposeType(workspace string, t Type, actor Actor) (TypeView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !typeForm(t) {
		return TypeView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return TypeView{}, err
	}
	key := typeKey{t.Name, t.SchemaVersion}
	if s := r.workspaces[workspace]; s != nil && s.types[key] != nil {
		found := s.types[key]
		if found.State == StateProposed && found.Proposer == actor.ID && sameType(found.Type, t) {
			return viewType(found), nil
		}
		return TypeView{}, ErrConflict
	}
	s, err := r.open(workspace)
	if err != nil {
		return TypeView{}, err
	}
	if len(s.types) >= r.limits.Types {
		return TypeView{}, ErrLimit
	}
	made := &TypeView{Type: copyType(t), State: StateProposed, Proposer: actor.ID}
	s.types[key] = made
	return viewType(made), nil
}

func (r *Registry) ApproveType(workspace, name, version string, actor Actor) (TypeView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(name) || !validID(version) || !actor.valid() {
		return TypeView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return TypeView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.types[typeKey{name, version}] == nil {
		return TypeView{}, ErrNotFound
	}
	t := s.types[typeKey{name, version}]
	switch {
	case t.State == StateActive && t.Approver == actor.ID:
		return viewType(t), nil
	case t.State != StateProposed:
		return TypeView{}, refuse(ReasonNotPending)
	case t.Proposer == actor.ID:
		return TypeView{}, refuse(ReasonSelfApproval)
	}
	t.State, t.Approver = StateActive, actor.ID
	return viewType(t), nil
}

func eventForm(e Event) bool {
	if !validID(e.ID) || !validSource(e.Source) || !validID(e.Type) || !validID(e.SchemaVersion) || e.Subject != "" && !validID(e.Subject) || len(e.Data) > maxFields {
		return false
	}
	if e.ObjectRevision != "" {
		if _, ok := parseRevision(e.ObjectRevision); !ok {
			return false
		}
	}
	for name, value := range e.Data {
		if !validField(name) || !validID(value) {
			return false
		}
	}
	return true
}

func copyEvent(e Event) Event {
	data := make(map[string]string, len(e.Data))
	for name, value := range e.Data {
		data[name] = value
	}
	e.Data = data
	return e
}

func sameEvent(a, b Event) bool {
	if a.ID != b.ID || a.Source != b.Source || a.Type != b.Type || a.SchemaVersion != b.SchemaVersion || a.Subject != b.Subject ||
		a.ObjectRevision != b.ObjectRevision || len(a.Data) != len(b.Data) {
		return false
	}
	for name, value := range a.Data {
		if other, ok := b.Data[name]; !ok || other != value {
			return false
		}
	}
	return true
}

func viewStored(e Stored) Stored {
	e.Event = copyEvent(e.Event)
	return e
}

func (r *Registry) sweep(s *space, now time.Time) {
	for s.first < len(s.events) && !now.Before(s.events[s.first].Time.Add(r.limits.Retention)) {
		gone := s.events[s.first]
		delete(s.published, eventKey{gone.Event.Source, gone.Event.ID})
		s.floor = gone.Sequence
		s.events[s.first] = Stored{}
		s.first++
	}
	if s.first > 0 && s.first*2 >= len(s.events) {
		s.events = append([]Stored(nil), s.events[s.first:]...)
		s.first = 0
	}
}

func (r *Registry) Publish(workspace string, e Event, actor Actor) (Stored, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !eventForm(e) {
		return Stored{}, ErrInvalid
	}
	if err := actor.system(); err != nil {
		return Stored{}, err
	}
	s := r.workspaces[workspace]
	if s == nil {
		return Stored{}, refuse(ReasonUnknownType)
	}
	r.sweep(s, now)
	if sequence, ok := s.published[eventKey{e.Source, e.ID}]; ok {
		stored := s.events[s.first+int(sequence-s.floor-1)]
		if sameEvent(stored.Event, e) {
			return viewStored(stored), nil
		}
		return Stored{}, ErrConflict
	}
	t := s.types[typeKey{e.Type, e.SchemaVersion}]
	if t == nil || t.State != StateActive {
		return Stored{}, refuse(ReasonUnknownType)
	}
	allowed := map[string]bool{}
	for _, field := range t.Type.Fields {
		allowed[field] = true
	}
	for name := range e.Data {
		if !allowed[name] {
			return Stored{}, refuse(ReasonUnknownField)
		}
	}
	if len(s.events)-s.first >= r.limits.Events {
		return Stored{}, ErrLimit
	}
	at := now
	if at.Before(s.last) {
		at = s.last
	}
	s.last = at
	s.sequence++
	stored := Stored{Event: copyEvent(e), Sequence: s.sequence, Time: at, SpecVersion: specVersion}
	s.events = append(s.events, stored)
	s.published[eventKey{e.Source, e.ID}] = s.sequence
	return viewStored(stored), nil
}

func (r *Registry) Read(workspace string, after uint64, limit int, actor Actor) (Page, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || limit < 1 || !actor.valid() {
		return Page{}, ErrInvalid
	}
	if err := actor.system(); err != nil {
		return Page{}, err
	}
	s := r.workspaces[workspace]
	if s == nil {
		if after != 0 {
			return Page{}, ErrInvalid
		}
		return Page{Events: []Stored{}}, nil
	}
	r.sweep(s, now)
	switch {
	case after > s.sequence:
		return Page{}, ErrInvalid
	case after < s.floor:
		return Page{}, Expired{Resync: Resync{Floor: s.floor, Head: s.sequence}}
	}
	start := s.first + int(after-s.floor)
	end := min(len(s.events), start+min(limit, r.limits.Page))
	out := make([]Stored, 0, end-start)
	for _, e := range s.events[start:end] {
		out = append(out, viewStored(e))
	}
	return Page{Events: out, Head: s.sequence}, nil
}

func (r *Registry) Bounds(workspace string, actor Actor) (uint64, uint64, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() {
		return 0, 0, ErrInvalid
	}
	if err := actor.system(); err != nil {
		return 0, 0, err
	}
	s := r.workspaces[workspace]
	if s == nil {
		return 0, 0, nil
	}
	r.sweep(s, now)
	return s.floor, s.sequence, nil
}

func Body(e Stored) []byte {
	document := map[string]any{"id": e.Event.ID, "source": e.Event.Source, "type": e.Event.Type, "specversion": e.SpecVersion,
		"time": e.Time.UTC().Format(time.RFC3339Nano), "schemaversion": e.Event.SchemaVersion, "data": copyEvent(e.Event).Data}
	if e.Event.Subject != "" {
		document["subject"] = e.Event.Subject
	}
	if e.Event.ObjectRevision != "" {
		document["objectrevision"] = e.Event.ObjectRevision
	}
	out, _ := json.Marshal(document)
	return out
}
