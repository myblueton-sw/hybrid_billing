package connectors

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

type capKey struct {
	provider  string
	version   string
	operation Operation
	kind      Kind
	condition Condition
}

type capability struct {
	stated   Capability
	revision uint64
	at       time.Time
}

type part struct {
	stated     Part
	verified   bool
	mismatches int
	last       Part
}

type page struct {
	next     string
	rows     int64
	checksum string
}

type declared struct {
	pages int
	rows  int64
}

type collection struct {
	spec          Spec
	state         State
	capability    uint64
	manifest      bool
	origin        Origin
	parts         map[string]*part
	pages         map[string]page
	chain         []string
	sums          map[string]bool
	waiting       string
	ended         bool
	total         *declared
	losses        []LossReason
	resumed       uint64
	abandon       AbandonReason
	retire        RetireReason
	throttles     int
	lastWait      time.Duration
	totals        Totals
	digest        string
	empty         bool
	completion    uint64
	started       time.Time
	changed       time.Time
	contradiction string
	subject       *subject
}

type subject struct {
	known     uint64
	effective *collection
	pointer   uint64
	completed int
	moved     time.Time
	members   []*collection
}

type space struct {
	collections map[string]*collection
	subjects    map[Subject]*subject
	bindings    map[string]string
}

type Registry struct {
	mu           sync.Mutex
	built        bool
	limits       Limits
	clock        func() time.Time
	last         time.Time
	capabilities map[capKey]*capability
	spaces       map[string]*space
	completions  uint64
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, clock: clock, capabilities: map[capKey]*capability{}, spaces: map[string]*space{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock(); read.After(r.last) {
		r.last = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func keyOf(c Capability) capKey {
	return capKey{c.Provider, c.Version, c.Operation, c.Kind, c.Condition}
}

func (r *Registry) Declare(stated Capability) (uint64, error) {
	defer r.leave()
	if !r.enter() || !stated.valid() {
		return 0, ErrInvalid
	}
	held := r.capabilities[keyOf(stated)]
	switch {
	case held == nil && len(r.capabilities) >= r.limits.Capabilities:
		return 0, ErrLimit
	case held == nil:
		r.capabilities[keyOf(stated)] = &capability{stated: stated, revision: 1, at: r.last}
		return 1, nil
	case held.stated != stated:
		held.stated, held.at = stated, r.last
		held.revision++
	}
	return held.revision, nil
}

func (r *Registry) permit(key capKey) Permission {
	held := r.capabilities[key]
	if held == nil {
		return Permission{Answer: Unknown}
	}
	answer := map[Support]Answer{Verified: Allowed, Limited: AnswerLimited, Unsupported: AnswerUnsupported}[held.stated.State]
	return Permission{Answer: answer, Reason: held.stated.Reason, Alternative: held.stated.Alternative, Revision: held.revision}
}

func (r *Registry) Permit(provider, version string, operation Operation, kind Kind, condition Condition) (Permission, error) {
	defer r.leave()
	if !r.enter() || !validID(provider) || !validID(version) || !paired(operation, kind) || !condition.valid() {
		return Permission{}, ErrInvalid
	}
	return r.permit(capKey{provider, version, operation, kind, condition}), nil
}

func (c *collection) key() capKey {
	return capKey{c.spec.Provider, c.spec.Version, c.spec.Subject.Operation, c.spec.Subject.Kind, c.spec.Condition}
}

func (r *Registry) permitted(c *collection) bool {
	answer := r.permit(c.key())
	return answer.Answer == Allowed && answer.Revision == c.capability
}

func (c *collection) become(state State, at time.Time) {
	c.state, c.changed = state, at
}

func (c *collection) contradict(what string, at time.Time) State {
	c.contradiction = what
	c.become(Inconsistent, at)
	return c.state
}

func (s *subject) raise(revision uint64, at time.Time) {
	if revision <= s.known {
		return
	}
	s.known = revision
	for _, member := range s.members {
		if member.state.running() && member.spec.Revision < revision {
			member.become(Superseded, at)
		}
	}
}

func (r *Registry) Start(spec Spec) error {
	defer r.leave()
	if !r.enter() || !spec.valid() {
		return ErrInvalid
	}
	held := r.spaces[spec.Subject.WorkspaceID]
	if held != nil {
		if existing := held.collections[spec.ID]; existing != nil {
			if existing.spec == spec {
				return nil
			}
			return ErrConflict
		}
	}
	made := &collection{spec: spec, state: Collecting, parts: map[string]*part{}, pages: map[string]page{}, sums: map[string]bool{}, waiting: spec.Cursor, started: r.last, changed: r.last}
	answer := r.permit(made.key())
	if answer.Answer != Allowed {
		return ErrRejected
	}
	made.capability = answer.Revision
	var topic *subject
	if held != nil {
		if bound, known := held.bindings[spec.Subject.ConnectionID]; known && bound != spec.Provider {
			return ErrConflict
		}
		topic = held.subjects[spec.Subject]
		if topic != nil && spec.Revision < topic.known {
			return ErrRejected
		}
		if len(held.collections) >= r.limits.Collections {
			return ErrLimit
		}
	}
	if held == nil {
		held = &space{collections: map[string]*collection{}, subjects: map[Subject]*subject{}, bindings: map[string]string{}}
		r.spaces[spec.Subject.WorkspaceID] = held
	}
	if topic == nil {
		topic = &subject{}
		held.subjects[spec.Subject] = topic
	}
	held.bindings[spec.Subject.ConnectionID] = spec.Provider
	topic.raise(spec.Revision, r.last)
	made.subject = topic
	topic.members = append(topic.members, made)
	held.collections[spec.ID] = made
	return nil
}

func (r *Registry) find(workspace, id string) (*collection, error) {
	if !r.enter() || !validID(workspace) || !validID(id) {
		return nil, ErrInvalid
	}
	held := r.spaces[workspace]
	if held == nil || held.collections[id] == nil {
		return nil, ErrNotFound
	}
	return held.collections[id], nil
}

func (r *Registry) validPart(given Part) bool {
	return validID(given.ID) && validChecksum(given.Checksum) && given.Bytes >= 0 && given.Bytes <= r.limits.MaxBytes && given.Rows >= 0 && given.Rows <= r.limits.MaxRows
}

func (r *Registry) DeclareManifest(workspace, id string, origin Origin, parts []Part) (State, error) {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return "", err
	}
	if origin != SourceListed && origin != AdapterDerived {
		return "", ErrInvalid
	}
	listed := map[string]Part{}
	for _, given := range parts {
		if _, twice := listed[given.ID]; twice || !r.validPart(given) {
			return "", ErrInvalid
		}
		listed[given.ID] = given
	}
	if found.spec.Shape != Files {
		return "", ErrRejected
	}
	if found.manifest {
		same := origin == found.origin && len(listed) == len(found.parts)
		for name, given := range listed {
			if held := found.parts[name]; held == nil || held.stated != given {
				same = false
			}
		}
		switch {
		case same:
			return found.state, nil
		case !found.state.running():
			return "", ErrRejected
		}
		return found.contradict("manifest", r.last), nil
	}
	if found.state != Collecting {
		return "", ErrRejected
	}
	if len(parts) > r.limits.Parts {
		return "", ErrLimit
	}
	found.manifest, found.origin = true, origin
	for name, given := range listed {
		found.parts[name] = &part{stated: given}
	}
	return found.state, nil
}

func (r *Registry) ReceivePart(workspace, id string, given Part) (State, error) {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return "", err
	}
	if !r.validPart(given) {
		return "", ErrInvalid
	}
	held := found.parts[given.ID]
	if found.spec.Shape != Files || held == nil {
		return "", ErrRejected
	}
	matches := held.stated == given
	switch {
	case matches && held.verified:
		return found.state, nil
	case held.verified && found.state.running():
		return found.contradict("part:"+given.ID, r.last), nil
	case found.state != Collecting:
		return "", ErrRejected
	case matches:
		held.verified = true
	default:
		held.last = given
		if held.mismatches < counterCap {
			held.mismatches++
		}
	}
	return found.state, nil
}

func (c *collection) rows() int64 {
	sum := int64(0)
	for _, received := range c.pages {
		sum += received.rows
	}
	return sum
}

func (c *collection) contradicted() bool {
	return c.total != nil && (len(c.chain) > c.total.pages || (c.ended && (len(c.chain) != c.total.pages || c.rows() != c.total.rows)))
}

func (r *Registry) DeclareTotal(workspace, id string, pages int, rows int64) (State, error) {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return "", err
	}
	if pages < 1 || rows < 0 {
		return "", ErrInvalid
	}
	if found.spec.Shape != Pages {
		return "", ErrRejected
	}
	if found.total != nil && *found.total == (declared{pages, rows}) {
		return found.state, nil
	}
	if found.total != nil && found.state.running() {
		return found.contradict("total", r.last), nil
	}
	if found.state != Collecting || found.total != nil {
		return "", ErrRejected
	}
	found.total = &declared{pages, rows}
	if found.contradicted() {
		return found.contradict("totals", r.last), nil
	}
	return found.state, nil
}

func (r *Registry) ReceivePage(workspace, id, cursor, next string, rows int64, checksum string) (State, error) {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return "", err
	}
	if !validID(cursor) || (next != "" && !validID(next)) || !validChecksum(checksum) || rows < 0 || rows > r.limits.MaxRows {
		return "", ErrInvalid
	}
	if found.spec.Shape != Pages {
		return "", ErrRejected
	}
	given := page{next: next, rows: rows, checksum: checksum}
	if seen, received := found.pages[cursor]; received {
		switch {
		case seen == given:
			return found.state, nil
		case !found.state.running():
			return "", ErrRejected
		}
		return found.contradict("page:"+cursor, r.last), nil
	}
	if found.state != Collecting || cursor != found.waiting {
		return "", ErrRejected
	}
	if len(found.chain) >= r.limits.Pages {
		return "", ErrLimit
	}
	_, loops := found.pages[next]
	if next == cursor || loops {
		return found.contradict("loop:"+cursor, r.last), nil
	}
	if rows > 0 && found.sums[checksum] {
		return found.contradict("content:"+cursor, r.last), nil
	}
	found.pages[cursor] = given
	found.chain = append(found.chain, cursor)
	found.sums[checksum] = found.sums[checksum] || rows > 0
	found.waiting, found.ended = next, next == ""
	if found.contradicted() {
		return found.contradict("totals", r.last), nil
	}
	return found.state, nil
}

func (r *Registry) Throttled(workspace, id string, wait time.Duration) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	if wait < 0 || wait > r.limits.MaxWait {
		return ErrInvalid
	}
	if found.state != Collecting {
		return ErrRejected
	}
	if found.throttles < counterCap {
		found.throttles++
	}
	found.lastWait = wait
	return nil
}

func (r *Registry) AccessLost(workspace, id string, interruption uint64, reason LossReason) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	if interruption == 0 || !reason.known() {
		return ErrInvalid
	}
	current := uint64(len(found.losses))
	switch {
	case interruption <= current && found.losses[interruption-1] == reason:
		return nil
	case interruption != current+1:
		return ErrConflict
	case found.state != Collecting:
		return ErrRejected
	}
	found.losses = append(found.losses, reason)
	found.become(Interrupted, r.last)
	return nil
}

func (r *Registry) Resume(workspace, id string, interruption uint64) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	current := uint64(len(found.losses))
	if interruption == 0 {
		return ErrInvalid
	}
	switch {
	case interruption > current:
		return ErrRejected
	case interruption <= found.resumed:
		return nil
	case found.state != Interrupted || interruption < current || !r.permitted(found):
		return ErrRejected
	}
	found.resumed = interruption
	found.become(Collecting, r.last)
	return nil
}

func (r *Registry) SourceChanged(workspace, id string, revision uint64) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	if revision <= found.spec.Revision {
		return ErrRejected
	}
	found.subject.raise(revision, r.last)
	return nil
}

func (r *Registry) Abandon(workspace, id string, reason AbandonReason) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	if !reason.known() {
		return ErrInvalid
	}
	switch {
	case found.state == Abandoned && found.abandon == reason:
		return nil
	case found.state == Abandoned:
		return ErrConflict
	case !found.state.running():
		return ErrRejected
	}
	found.abandon = reason
	found.become(Abandoned, r.last)
	return nil
}

func (c *collection) gaps(permitted bool) []Gap {
	missing := []Gap{}
	if !permitted {
		missing = append(missing, Gap{What: GapCapability})
	}
	if c.spec.Shape == Pages {
		if !c.ended {
			missing = append(missing, Gap{What: GapChain})
		}
		if c.contradicted() {
			missing = append(missing, Gap{What: GapTotals})
		}
		return missing
	}
	if !c.manifest {
		return append(missing, Gap{What: GapManifest})
	}
	for _, name := range c.names() {
		switch held := c.parts[name]; {
		case held.verified:
		case held.mismatches > 0:
			missing = append(missing, Gap{What: GapMismatched, Part: name})
		default:
			missing = append(missing, Gap{What: GapMissing, Part: name})
		}
	}
	return missing
}

func (c *collection) names() []string {
	names := make([]string, 0, len(c.parts))
	for name := range c.parts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *collection) seal() {
	hash := sha256.New()
	if c.spec.Shape == Files {
		for _, name := range c.names() {
			stated := c.parts[name].stated
			fmt.Fprintf(hash, "%d:%s|%d|%d|%s\n", len(name), name, stated.Bytes, stated.Rows, stated.Checksum)
			c.totals.Units++
			c.totals.Bytes += stated.Bytes
			c.totals.Rows += stated.Rows
		}
	} else {
		for _, cursor := range c.chain {
			received := c.pages[cursor]
			fmt.Fprintf(hash, "%d|%s\n", received.rows, received.checksum)
			c.totals.Units++
			c.totals.Rows += received.rows
		}
	}
	c.digest, c.empty = hex.EncodeToString(hash.Sum(nil)), c.totals.Rows == 0
}

func (r *Registry) Complete(workspace, id string) ([]Gap, error) {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	if found.state == Completed || found.state == Retired {
		return nil, nil
	}
	if found.state != Collecting {
		return nil, ErrRejected
	}
	if missing := found.gaps(r.permitted(found)); len(missing) > 0 {
		return missing, ErrRejected
	}
	found.seal()
	r.completions++
	found.completion = r.completions
	found.become(Completed, r.last)
	topic := found.subject
	if topic.completed == 0 {
		topic.effective, topic.moved = found, r.last
		topic.pointer++
	}
	topic.completed++
	return nil, nil
}

func (r *Registry) Adopt(workspace, id string, pointer uint64) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	topic := found.subject
	switch {
	case found.state != Completed:
		return ErrRejected
	case pointer != topic.pointer:
		return ErrConflict
	case topic.effective == found:
		return nil
	case topic.effective != nil && found.spec.Revision < topic.effective.spec.Revision:
		return ErrRejected
	}
	topic.effective, topic.moved = found, r.last
	topic.pointer++
	return nil
}

func (r *Registry) Retire(workspace, id string, reason RetireReason) error {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return err
	}
	if !reason.known() {
		return ErrInvalid
	}
	switch {
	case found.state == Retired && found.retire == reason:
		return nil
	case found.state == Retired:
		return ErrConflict
	case found.state != Completed:
		return ErrRejected
	}
	found.retire = reason
	found.become(Retired, r.last)
	if topic := found.subject; topic.effective == found {
		topic.effective, topic.moved = nil, r.last
		topic.pointer++
	}
	return nil
}

func (c *collection) view() CollectionView {
	shown := CollectionView{Spec: c.spec, State: c.state, Capability: c.capability, Origin: c.origin, Declared: c.manifest, Pages: len(c.chain), Waiting: c.waiting, Ended: c.ended,
		Interruption: uint64(len(c.losses)), AbandonReason: c.abandon, RetireReason: c.retire, Throttles: c.throttles, LastWait: c.lastWait, Totals: c.totals, Digest: c.digest,
		Empty: c.empty, Completion: c.completion, Contradiction: c.contradiction, Started: c.started, Changed: c.changed}
	if len(c.losses) > 0 {
		shown.LossReason = c.losses[len(c.losses)-1]
	}
	for _, name := range c.names() {
		held := c.parts[name]
		shown.Parts = append(shown.Parts, PartView{Part: held.stated, Verified: held.verified, Mismatches: held.mismatches, Last: held.last})
	}
	return shown
}

func (r *Registry) Collection(workspace, id string) (CollectionView, error) {
	defer r.leave()
	found, err := r.find(workspace, id)
	if err != nil {
		return CollectionView{}, err
	}
	return found.view(), nil
}

func (r *Registry) Effective(about Subject) (EffectiveView, error) {
	defer r.leave()
	if !r.enter() || !about.valid() {
		return EffectiveView{}, ErrInvalid
	}
	held := r.spaces[about.WorkspaceID]
	if held == nil || held.subjects[about] == nil {
		return EffectiveView{}, ErrNotFound
	}
	topic := held.subjects[about]
	shown := EffectiveView{Pointer: topic.pointer, Moved: topic.moved, Known: topic.known, Others: []Other{}}
	if topic.effective != nil {
		view := topic.effective.view()
		shown.Collection, shown.Stale, shown.Present = &view, topic.known > topic.effective.spec.Revision, r.permit(topic.effective.key())
	}
	for _, member := range topic.members {
		if member == topic.effective || (member.state != Completed && member.state != Retired) {
			continue
		}
		other := Other{ID: member.spec.ID, Revision: member.spec.Revision, Digest: member.digest, Empty: member.empty, Origin: member.origin, RetireReason: member.retire}
		switch current := topic.effective; {
		case member.state == Retired:
			other.Standing = IsRetired
		case current == nil:
			other.Standing = Candidate
		case member.spec.Revision > current.spec.Revision:
			other.Standing = Newer
		case member.spec.Revision < current.spec.Revision:
			other.Standing = Older
		case member.digest == current.digest:
			other.Standing = Duplicate
		default:
			other.Standing = Conflicting
		}
		shown.Others = append(shown.Others, other)
	}
	return shown, nil
}
