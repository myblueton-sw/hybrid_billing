package trust

import (
	"sort"
	"sync"
	"time"
)

type connection struct {
	active   Connection
	revision int
	state    string
	pending  *Connection
	proposal int
	proposer string
	approver string
	keys     map[string]*key
	seen     map[string]seen
}

type seen struct {
	digest  [32]byte
	expires time.Time
}

type key struct {
	view    KeyView
	retired uint64
}

type owner struct {
	workspace  string
	connection string
}

type space struct {
	connections  map[string]*connection
	destinations map[string]*destination
	quarantine   []Entry
	dropped      int
	counts       map[string]int
}

type Registry struct {
	mu          sync.Mutex
	limits      Limits
	clock       func() time.Time
	now         time.Time
	workspaces  map[string]*space
	owners      map[Fingerprint]owner
	retired     map[Fingerprint]bool
	refusals    map[string]int
	retiredKeys uint64
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, workspaces: map[string]*space{}, owners: map[Fingerprint]owner{}, retired: map[Fingerprint]bool{},
		refusals: map[string]int{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Connections, maxConnections}, {l.Retired, maxRetired}, {l.Endpoints, maxEndpoints}, {l.Keys, maxKeys},
		{l.Destinations, maxDestinations}, {l.Events, maxEvents}, {l.Items, maxItems}, {l.Quarantine, maxQuarantine}, {l.Allowlist, maxAllowlist},
		{l.Redirects, maxRedirects}, {l.Body, maxBody}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return l.Items <= l.Events && l.Skew >= minSkew && l.Skew <= maxSkew && l.KeyLife >= minKeyLife && l.KeyLife <= maxKeyLife
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

func (r *Registry) configure(workspace string, actor Actor, ids ...string) error {
	if !validID(workspace) || !actor.valid() {
		return ErrInvalid
	}
	for _, id := range ids {
		if !validID(id) {
			return ErrInvalid
		}
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
	s := &space{connections: map[string]*connection{}, destinations: map[string]*destination{}, counts: map[string]int{}}
	r.workspaces[workspace] = s
	return s, nil
}

func (r *Registry) connectionForm(c Connection) bool {
	if !validID(c.ID) || len(c.Fingerprints) < 1 || len(c.Fingerprints) > maxFingerprints || len(c.Endpoints) < 1 || len(c.Endpoints) > r.limits.Endpoints {
		return false
	}
	prints := map[Fingerprint]bool{}
	for _, print := range c.Fingerprints {
		if print == (Fingerprint{}) || prints[print] {
			return false
		}
		prints[print] = true
	}
	ends := map[string]bool{}
	for _, endpoint := range c.Endpoints {
		if !validID(endpoint) || ends[endpoint] {
			return false
		}
		ends[endpoint] = true
	}
	return true
}

func copyConnection(c Connection) Connection {
	c.Fingerprints = append([]Fingerprint(nil), c.Fingerprints...)
	c.Endpoints = append([]string(nil), c.Endpoints...)
	sort.Slice(c.Fingerprints, func(i, j int) bool { return string(c.Fingerprints[i][:]) < string(c.Fingerprints[j][:]) })
	sort.Strings(c.Endpoints)
	return c
}

func sameConnection(a, b Connection) bool {
	a, b = copyConnection(a), copyConnection(b)
	if len(a.Fingerprints) != len(b.Fingerprints) || len(a.Endpoints) != len(b.Endpoints) {
		return false
	}
	for index := range a.Fingerprints {
		if a.Fingerprints[index] != b.Fingerprints[index] {
			return false
		}
	}
	for index := range a.Endpoints {
		if a.Endpoints[index] != b.Endpoints[index] {
			return false
		}
	}
	return true
}

func viewConnection(c *connection) ConnectionView {
	v := ConnectionView{Active: copyConnection(c.active), Revision: c.revision, Proposal: c.proposal, State: c.state, Proposer: c.proposer, Approver: c.approver}
	if c.pending != nil {
		pending := copyConnection(*c.pending)
		v.Pending = &pending
	}
	return v
}

func (r *Registry) usable(workspace string, c Connection) error {
	for _, print := range c.Fingerprints {
		if r.retired[print] {
			return refuse(ReasonFingerprintRetired)
		}
		if found, ok := r.owners[print]; ok && found != (owner{workspace, c.ID}) {
			return refuse(ReasonFingerprintInUse)
		}
	}
	return nil
}

func (r *Registry) ProposeConnection(workspace string, c Connection, actor Actor) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor); err != nil || !r.connectionForm(c) {
		return ConnectionView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ConnectionView{}, err
	}
	var found *connection
	if s := r.workspaces[workspace]; s != nil {
		found = s.connections[c.ID]
	}
	if found != nil {
		switch {
		case found.state == StateRevoked:
			return ConnectionView{}, refuse(ReasonRevoked)
		case found.pending != nil && found.proposer == actor.ID && sameConnection(*found.pending, c):
			return viewConnection(found), nil
		case found.pending != nil:
			return ConnectionView{}, ErrConflict
		case sameConnection(found.active, c):
			return viewConnection(found), nil
		}
	}
	if err := r.usable(workspace, c); err != nil {
		return ConnectionView{}, err
	}
	s, err := r.open(workspace)
	if err != nil {
		return ConnectionView{}, err
	}
	if found == nil {
		if len(s.connections) >= r.limits.Connections {
			return ConnectionView{}, ErrLimit
		}
		found = &connection{state: StateProposed, active: Connection{ID: c.ID}, keys: map[string]*key{}, seen: map[string]seen{}}
		s.connections[c.ID] = found
	}
	pending := copyConnection(c)
	found.pending, found.proposal, found.proposer, found.approver = &pending, found.proposal+1, actor.ID, ""
	return viewConnection(found), nil
}

func (r *Registry) find(workspace, id string) (*space, *connection, error) {
	s := r.workspaces[workspace]
	if s == nil || s.connections[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.connections[id], nil
}

func (r *Registry) ApproveConnection(workspace, id string, proposal int, actor Actor) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, id); err != nil || proposal < 1 {
		return ConnectionView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ConnectionView{}, err
	}
	_, c, err := r.find(workspace, id)
	if err != nil {
		return ConnectionView{}, err
	}
	switch {
	case c.proposal != proposal:
		return ConnectionView{}, refuse(ReasonProposalChanged)
	case c.pending == nil && c.state == StateActive && c.approver == actor.ID:
		return viewConnection(c), nil
	case c.pending == nil:
		return ConnectionView{}, refuse(ReasonNotPending)
	case c.proposer == actor.ID:
		return ConnectionView{}, refuse(ReasonSelfApproval)
	}
	if err := r.usable(workspace, *c.pending); err != nil {
		return ConnectionView{}, err
	}
	kept := map[Fingerprint]bool{}
	for _, print := range c.pending.Fingerprints {
		kept[print] = true
	}
	var dropped []Fingerprint
	for _, print := range c.active.Fingerprints {
		if !kept[print] {
			dropped = append(dropped, print)
		}
	}
	added := 0
	for _, print := range c.pending.Fingerprints {
		if _, owned := r.owners[print]; !owned {
			added++
		}
	}
	if len(r.retired)+len(r.owners)+added > r.limits.Retired {
		return ConnectionView{}, ErrLimit
	}
	r.retire(dropped)
	for _, print := range c.pending.Fingerprints {
		r.owners[print] = owner{workspace, id}
	}
	c.active, c.pending, c.revision, c.state, c.approver = *c.pending, nil, c.revision+1, StateActive, actor.ID
	return viewConnection(c), nil
}

func (r *Registry) retire(prints []Fingerprint) {
	for _, print := range prints {
		delete(r.owners, print)
		r.retired[print] = true
	}
}

func (r *Registry) RevokeConnection(workspace, id string, actor Actor) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, id); err != nil {
		return ConnectionView{}, err
	}
	if err := actor.human(); err != nil {
		return ConnectionView{}, err
	}
	_, c, err := r.find(workspace, id)
	if err != nil {
		return ConnectionView{}, err
	}
	if c.state == StateRevoked {
		return viewConnection(c), nil
	}
	if c.state == StateActive {
		r.retire(c.active.Fingerprints)
	}
	c.state, c.pending = StateRevoked, nil
	c.seen = map[string]seen{}
	return viewConnection(c), nil
}

func (r *Registry) WithdrawConnection(workspace, id string, actor Actor) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, id); err != nil {
		return ConnectionView{}, err
	}
	if err := actor.human(); err != nil {
		return ConnectionView{}, err
	}
	_, c, err := r.find(workspace, id)
	if err != nil {
		return ConnectionView{}, err
	}
	c.pending = nil
	return viewConnection(c), nil
}

func (r *Registry) ConnectionOf(workspace, id string) (ConnectionView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return ConnectionView{}, ErrInvalid
	}
	_, c, err := r.find(workspace, id)
	if err != nil {
		return ConnectionView{}, err
	}
	return viewConnection(c), nil
}

func (r *Registry) Admit(p Presented) (Admission, error) {
	r.enter()
	defer r.leave()
	if p.Fingerprint == (Fingerprint{}) || !validID(p.Workspace) || !validID(p.Endpoint) {
		return Admission{}, ErrInvalid
	}
	found, ok := r.owners[p.Fingerprint]
	reason := ""
	switch {
	case !ok:
		reason = ReasonUnknownCertificate
	case found.workspace != p.Workspace:
		reason = ReasonWorkspaceMismatch
	}
	if reason == "" {
		c := r.workspaces[found.workspace].connections[found.connection]
		registered := false
		for _, endpoint := range c.active.Endpoints {
			registered = registered || endpoint == p.Endpoint
		}
		if registered {
			return Admission{Workspace: found.workspace, Connection: found.connection, Revision: c.revision}, nil
		}
		reason = ReasonEndpointNotRegistered
	}
	r.refusals[reason]++
	return Admission{}, refuse(reason)
}

func (r *Registry) AdmissionRefusals() map[string]int {
	r.enter()
	defer r.leave()
	out := map[string]int{}
	for reason, count := range r.refusals {
		out[reason] = count
	}
	return out
}
