package identity

import (
	"crypto/rand"
	"sort"
	"sync"
	"time"
)

type user struct {
	kind          Kind
	state         UserState
	epoch         uint64
	workspace     string
	creator       string
	owner         string
	operatorBlock bool
	linked        int
	links         uint64
}

type identityRecord struct {
	user   string
	email  string
	linked bool
	order  uint64
}

type bootstrap struct {
	armed     bool
	consumed  bool
	locked    bool
	verifier  Verifier
	validity  Validity
	workspace string
	attempts  int
}

type invitationState int

const (
	invitationPending invitationState = iota
	invitationAccepted
	invitationRevoked
	invitationLocked
)

type invitation struct {
	workspace      string
	role           Role
	roleUntil      time.Time
	verifier       Verifier
	validity       Validity
	inviter        string
	inviterEpoch   uint64
	inviterVersion uint64
	invitee        string
	attempts       int
	state          invitationState
}

type membership struct {
	role         Role
	state        MembershipState
	version      uint64
	joined       uint64
	from         time.Time
	until        time.Time
	suspendedFor SuspendReason
}

func (r *Registry) effectiveNow(m *membership) bool {
	return m != nil && m.state == MembershipActive && (m.until.IsZero() || r.clock().Before(m.until))
}

func effective(m *membership, now time.Time) bool {
	return m != nil && m.state == MembershipActive && (m.until.IsZero() || now.Before(m.until))
}

type session struct {
	serial     uint64
	user       string
	epoch      uint64
	identity   Triple
	opened     time.Time
	idle       time.Duration
	idleExpiry time.Time
	absolute   time.Time
}

type credential struct {
	serial   uint64
	user     string
	verifier Verifier
	validity Validity
	revoked  bool
}

type Registry struct {
	mu             sync.Mutex
	built          bool
	installation   string
	instance       [16]byte
	limits         Limits
	clock          func() time.Time
	users          map[string]*user
	identities     map[Triple]*identityRecord
	boot           bootstrap
	invitations    map[string]*invitation
	memberships    map[string]map[string]*membership
	sessions       map[Verifier]*session
	serials        map[uint64]*session
	byUser         map[string]map[Verifier]struct{}
	serial         uint64
	credentials    map[string]*credential
	generation     uint64
	userIndex      []string
	memberIndex    map[string][]string
	usersVersion   uint64
	membersVersion map[string]uint64
}

type Installation struct{ r *Registry }

type Boundary struct{ r *Registry }

type Workspace struct{ r *Registry }

type Subjects struct{ r *Registry }

func New(installation string, limits Limits, clock func() time.Time) (*Registry, error) {
	if !validID(installation) || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	r := &Registry{built: true, installation: installation, limits: limits, clock: clock,
		users: map[string]*user{}, identities: map[Triple]*identityRecord{}, invitations: map[string]*invitation{},
		memberships: map[string]map[string]*membership{}, sessions: map[Verifier]*session{}, serials: map[uint64]*session{},
		byUser: map[string]map[Verifier]struct{}{}, credentials: map[string]*credential{},
		memberIndex: map[string][]string{}, membersVersion: map[string]uint64{}}
	if _, err := rand.Read(r.instance[:]); err != nil {
		return nil, ErrUnavailable
	}
	return r, nil
}

func (r *Registry) Installation() *Installation { return &Installation{r: r} }

func (r *Registry) Boundary() *Boundary { return &Boundary{r: r} }

func (r *Registry) Workspace() *Workspace { return &Workspace{r: r} }

func (r *Registry) Subjects() *Subjects { return &Subjects{r: r} }

func (r *Registry) InstallationID() string {
	if r == nil {
		return ""
	}
	return r.installation
}

func (r *Registry) Clock() func() time.Time {
	if r == nil {
		return nil
	}
	return r.clock
}

func (h *Subjects) open() (*Registry, bool) {
	if h == nil {
		return nil, false
	}
	return h.r, h.r.lock()
}

func (h *Subjects) Subject(userID string) (Kind, UserState, uint64, bool) {
	r, ok := h.open()
	if !ok {
		return "", "", 0, false
	}
	defer r.mu.Unlock()
	found := r.users[userID]
	if found == nil {
		return "", "", 0, false
	}
	return found.kind, found.state, found.epoch, true
}

func (h *Subjects) Member(workspace, userID string) (uint64, bool) {
	r, ok := h.open()
	if !ok {
		return 0, false
	}
	defer r.mu.Unlock()
	found := r.users[userID]
	if found == nil || found.state != Active {
		return 0, false
	}
	if found.kind == Service {
		return 1, found.workspace == workspace
	}
	own := r.member(workspace, userID)
	if !r.effectiveNow(own) {
		return 0, false
	}
	return own.joined, true
}

func (h *Subjects) Service(userID string) (string, string, bool) {
	r, ok := h.open()
	if !ok {
		return "", "", false
	}
	defer r.mu.Unlock()
	found := r.users[userID]
	if found == nil || found.kind != Service {
		return "", "", false
	}
	return found.creator, found.owner, true
}

func (h *Subjects) Generation() uint64 {
	r, ok := h.open()
	if !ok {
		return 0
	}
	defer r.mu.Unlock()
	return r.generation
}

func (r *Registry) lock() bool {
	if r == nil || !r.built {
		return false
	}
	r.mu.Lock()
	return true
}

func (h *Installation) open() (*Registry, bool) {
	if h == nil {
		return nil, false
	}
	return h.r, h.r.lock()
}

func (h *Boundary) open() (*Registry, bool) {
	if h == nil {
		return nil, false
	}
	return h.r, h.r.lock()
}

func (h *Workspace) open() (*Registry, bool) {
	if h == nil {
		return nil, false
	}
	return h.r, h.r.lock()
}

func (r *Registry) activeHuman(id string) bool {
	found := r.users[id]
	return found != nil && found.kind == Human && found.state == Active
}

func (r *Registry) current(found *session, now time.Time) bool {
	if found == nil || now.Before(found.opened) || !now.Before(found.idleExpiry) || !now.Before(found.absolute) {
		return false
	}
	linked := r.identities[found.identity]
	owner := r.users[found.user]
	return linked != nil && linked.linked && linked.user == found.user && owner != nil && owner.kind == Human && owner.state == Active && owner.epoch == found.epoch
}

func (r *Registry) acting(verifier Verifier, now time.Time) (*session, error) {
	found := r.sessions[verifier]
	if !verifier.valid() || !r.current(found, now) {
		return nil, refused(ReasonSession)
	}
	return found, nil
}

func (r *Registry) member(workspace, userID string) *membership {
	found := r.memberships[workspace][userID]
	if found == nil || found.state == MembershipRemoved {
		return nil
	}
	return found
}

func (r *Registry) activeAdministrator(workspace, userID string, now time.Time) bool {
	found := r.member(workspace, userID)
	return effective(found, now) && found.role == Administrator && r.activeHuman(userID)
}

func (r *Registry) administrator(verifier Verifier, workspace string, now time.Time) (*session, error) {
	found, err := r.acting(verifier, now)
	if err != nil {
		return nil, err
	}
	if !r.activeAdministrator(workspace, found.user, now) {
		return nil, refused(ReasonNotAdministrator)
	}
	return found, nil
}

func (r *Registry) administrators(workspace string, now time.Time) int {
	count := 0
	for userID := range r.memberships[workspace] {
		if r.activeAdministrator(workspace, userID, now) {
			count++
		}
	}
	return count
}

func (r *Registry) serviceUsers(workspace string) bool {
	for _, found := range r.users {
		if found.kind == Service && found.workspace == workspace && found.state != Departed {
			return true
		}
	}
	return false
}

func (r *Registry) strands(workspace, target string, after *membership, now time.Time) bool {
	before := r.administrators(workspace, now)
	if before == 0 || after != nil && after.role == Administrator && effective(after, now) && r.activeHuman(target) {
		return false
	}
	remaining := before
	if r.activeAdministrator(workspace, target, now) {
		remaining--
	}
	if remaining > 0 {
		return false
	}
	if after != nil && after.state != MembershipRemoved {
		return true
	}
	return r.others(workspace, target)
}

func (r *Registry) others(workspace, target string) bool {
	for userID := range r.memberships[workspace] {
		if userID != target && r.member(workspace, userID) != nil {
			return true
		}
	}
	return r.serviceUsers(workspace)
}

func (r *Registry) onlySeat(workspace, target string, now time.Time) bool {
	own := r.member(workspace, target)
	if own == nil || own.role != Administrator {
		return false
	}
	for userID := range r.memberships[workspace] {
		other := r.member(workspace, userID)
		if userID != target && effective(other, now) && other.role == Administrator && r.users[userID] != nil && r.users[userID].state != Departed {
			return false
		}
	}
	return r.others(workspace, target)
}

func (r *Registry) withoutAdministrator(now time.Time) []string {
	set := map[string]struct{}{}
	for workspace, members := range r.memberships {
		for userID := range members {
			if r.member(workspace, userID) != nil {
				set[workspace] = struct{}{}
				break
			}
		}
	}
	for _, found := range r.users {
		if found.kind == Service && found.state != Departed {
			set[found.workspace] = struct{}{}
		}
	}
	list := []string{}
	for workspace := range set {
		if r.administrators(workspace, now) == 0 {
			list = append(list, workspace)
		}
	}
	sort.Strings(list)
	return list
}

func (r *Registry) drop(userID string) {
	for held := range r.byUser[userID] {
		delete(r.serials, r.sessions[held].serial)
		delete(r.sessions, held)
	}
	delete(r.byUser, userID)
}

func beyond(ids []string, after string) int {
	return sort.Search(len(ids), func(i int) bool { return ids[i] > after })
}

func page(ids []string, after string, limit int) []string {
	sort.Strings(ids)
	start := beyond(ids, after)
	return ids[start : start+min(limit, len(ids)-start)]
}

func insertSorted(ids []string, id string) []string {
	at := sort.SearchStrings(ids, id)
	if at < len(ids) && ids[at] == id {
		return ids
	}
	ids = append(ids, "")
	copy(ids[at+1:], ids[at:])
	ids[at] = id
	return ids
}

func (r *Registry) putUser(id string, found *user) {
	r.users[id] = found
	r.userIndex = insertSorted(r.userIndex, id)
	r.usersVersion++
}

func (r *Registry) putMembership(workspace, id string, found *membership) {
	if r.memberships[workspace] == nil {
		r.memberships[workspace] = map[string]*membership{}
	}
	r.memberships[workspace][id] = found
	r.memberIndex[workspace] = insertSorted(r.memberIndex[workspace], id)
	r.membersVersion[workspace]++
}

func (r *Registry) pageable(after string, limit int) bool {
	return (after == "" || validID(after)) && limit >= 1 && limit <= r.limits.PageSize
}
