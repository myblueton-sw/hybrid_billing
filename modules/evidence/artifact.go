package evidence

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type instance struct {
	custody Custody
	lost    bool
	reason  LossReason
	order   uint64
}

type artifact struct {
	spec       ArtifactSpec
	print      string
	state      ArtifactState
	reason     DeleteReason
	instances  []*instance
	bundles    map[string]*bundle
	registered time.Time
	changed    time.Time
}

type owner struct {
	workspace string
	artifact  string
}

type workspace struct {
	artifacts map[string]*artifact
	bundles   map[string]*bundle
	holds     map[string]*hold
	gaps      map[string]*gap
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	clock      func() time.Time
	now        time.Time
	losses     uint64
	locations  map[string]owner
	restricted map[string]string
	workspaces map[string]*workspace
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, clock: clock, locations: map[string]owner{}, restricted: map[string]string{}, workspaces: map[string]*workspace{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock().UTC().Round(0); read.After(r.now) {
		r.now = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func join(fields ...string) string {
	var out strings.Builder
	for _, field := range fields {
		out.WriteString(strconv.Itoa(len(field)))
		out.WriteByte(':')
		out.WriteString(field)
	}
	return out.String()
}

func stamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339Nano)
}

func (r *Registry) space(id string) *workspace {
	ws := r.workspaces[id]
	if ws == nil {
		ws = &workspace{artifacts: map[string]*artifact{}, bundles: map[string]*bundle{}, holds: map[string]*hold{}, gaps: map[string]*gap{}}
		r.workspaces[id] = ws
	}
	return ws
}

func (r *Registry) artifact(workspaceID, id string) *artifact {
	if ws := r.workspaces[workspaceID]; ws != nil {
		return ws.artifacts[id]
	}
	return nil
}

func (r *Registry) taken(c Custody, workspaceID, id string) bool {
	held, known := r.locations[c.Location]
	return c.Kind == Product && known && held != owner{workspaceID, id}
}

func (r *Registry) listed(items []string) Listed {
	sort.Strings(items)
	shown := items
	if len(shown) > r.limits.Listed {
		shown = shown[:r.limits.Listed]
	}
	return Listed{Total: len(items), Items: shown}
}

func (a *artifact) heldUntil(now, until time.Time) bool {
	if a.state == Deleted {
		return false
	}
	if until.Before(now) {
		until = now
	}
	for _, held := range a.instances {
		if held.lost {
			continue
		}
		if held.custody.Kind == Product || !held.custody.VerifiedUntil.IsZero() && !held.custody.VerifiedUntil.Before(until) {
			return true
		}
	}
	return false
}

func (a *artifact) copied() bool {
	for _, held := range a.instances {
		if !held.lost && held.custody.Kind == Product {
			return true
		}
	}
	return false
}

func (a *artifact) cause() (string, LossReason) {
	var order uint64
	var reason LossReason
	for _, held := range a.instances {
		if !held.lost {
			return CauseUnverified, ""
		}
		if held.order > order {
			order, reason = held.order, held.reason
		}
	}
	return CauseLost, reason
}

func (r *Registry) Register(spec ArtifactSpec) (ArtifactState, error) {
	defer r.leave()
	if !r.enter() || !validID(spec.WorkspaceID) || !validID(spec.ID) || spec.Kind.rank() < 0 || !validID(spec.Version) || !validDigest(spec.Digest) || spec.Size <= 0 || !spec.Custody.valid() {
		return "", ErrInvalid
	}
	print := join(spec.ID, string(spec.Kind), spec.Version, spec.Digest, strconv.FormatInt(spec.Size, 10), string(spec.Custody.Kind), spec.Custody.Location, spec.Custody.Receipt, stamp(spec.Custody.VerifiedUntil), spec.Custody.Evidence)
	if held := r.artifact(spec.WorkspaceID, spec.ID); held != nil {
		if held.print != print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if r.taken(spec.Custody, spec.WorkspaceID, spec.ID) {
		return "", ErrConflict
	}
	if ws := r.workspaces[spec.WorkspaceID]; ws != nil && len(ws.artifacts) >= r.limits.Artifacts {
		return "", ErrLimit
	}
	spec.Custody.VerifiedUntil = spec.Custody.VerifiedUntil.UTC()
	made := &artifact{spec: spec, print: print, state: Registered, instances: []*instance{{custody: spec.Custody}}, bundles: map[string]*bundle{}, registered: r.now, changed: r.now}
	if spec.Custody.Kind == Product {
		r.locations[spec.Custody.Location] = owner{spec.WorkspaceID, spec.ID}
	}
	r.space(spec.WorkspaceID).artifacts[spec.ID] = made
	return made.state, nil
}

func (a *artifact) find(kind CustodyKind, location string) *instance {
	for _, held := range a.instances {
		if held.custody.Kind == kind && held.custody.Location == location {
			return held
		}
	}
	return nil
}

func (r *Registry) AddCustody(workspaceID, id string, custody Custody) (time.Time, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !custody.valid() {
		return time.Time{}, ErrInvalid
	}
	held := r.artifact(workspaceID, id)
	if held == nil {
		return time.Time{}, ErrNotFound
	}
	if held.state == Deleted {
		return time.Time{}, ErrRejected
	}
	custody.VerifiedUntil = custody.VerifiedUntil.UTC()
	if stored := held.find(custody.Kind, custody.Location); stored != nil {
		switch {
		case stored.lost:
			return time.Time{}, ErrRejected
		case custody.Kind == Product && stored.custody.Receipt != custody.Receipt:
			return time.Time{}, ErrConflict
		case custody.VerifiedUntil.After(stored.custody.VerifiedUntil):
			stored.custody.VerifiedUntil, stored.custody.Evidence = custody.VerifiedUntil, custody.Evidence
			held.changed = r.now
		}
		return stored.custody.VerifiedUntil, nil
	}
	if r.taken(custody, workspaceID, id) {
		return time.Time{}, ErrConflict
	}
	if len(held.instances) >= r.limits.Instances {
		return time.Time{}, ErrLimit
	}
	if custody.Kind == Product {
		r.locations[custody.Location] = owner{workspaceID, id}
	}
	held.instances = append(held.instances, &instance{custody: custody})
	held.changed = r.now
	return custody.VerifiedUntil, nil
}

func (r *Registry) ReportLoss(workspaceID, id string, kind CustodyKind, location string, reason LossReason) (Listed, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || (kind != Product && kind != Remote) || !validID(location) || !reason.known() {
		return Listed{}, ErrInvalid
	}
	held := r.artifact(workspaceID, id)
	if held == nil {
		return Listed{}, ErrNotFound
	}
	if held.state == Deleted {
		return Listed{}, ErrRejected
	}
	stored := held.find(kind, location)
	switch {
	case stored == nil:
		return Listed{}, ErrNotFound
	case stored.lost && stored.reason != reason:
		return Listed{}, ErrConflict
	case !stored.lost && reason == VerificationWithdrawn && kind == Product:
		return Listed{}, ErrRejected
	case !stored.lost:
		r.losses++
		stored.lost, stored.reason, stored.order = true, reason, r.losses
		held.changed = r.now
	}
	var exposed []string
	for name, dependant := range held.bundles {
		if !held.heldUntil(r.now, r.now) || r.risky(dependant, held) {
			exposed = append(exposed, name)
		}
	}
	return r.listed(exposed), nil
}

func (r *Registry) DeleteArtifact(workspaceID, id string, reason DeleteReason) (Listed, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return Listed{}, ErrInvalid
	}
	held := r.artifact(workspaceID, id)
	if held == nil {
		return Listed{}, ErrNotFound
	}
	if held.state == Deleted {
		if held.reason != reason {
			return Listed{}, ErrConflict
		}
		return Listed{}, nil
	}
	if len(held.bundles) > 0 {
		names := make([]string, 0, len(held.bundles))
		for name := range held.bundles {
			names = append(names, name)
		}
		return r.listed(names), ErrRejected
	}
	held.state, held.reason, held.changed = Deleted, reason, r.now
	return Listed{}, nil
}

func (r *Registry) Restrict(kind Kind, version, reason string) error {
	defer r.leave()
	if !r.enter() || (kind != Engine && kind != Module) || !validID(version) || !validID(reason) {
		return ErrInvalid
	}
	key := join(string(kind), version)
	if stored, known := r.restricted[key]; known {
		if stored != reason {
			return ErrConflict
		}
		return nil
	}
	if len(r.restricted) >= r.limits.Restricted {
		return ErrLimit
	}
	r.restricted[key] = reason
	return nil
}

func (r *Registry) barred(a *artifact) bool {
	_, known := r.restricted[join(string(a.spec.Kind), a.spec.Version)]
	return known
}

func (r *Registry) Artifact(workspaceID, id string) (ArtifactView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return ArtifactView{}, ErrInvalid
	}
	held := r.artifact(workspaceID, id)
	if held == nil {
		return ArtifactView{}, ErrNotFound
	}
	view := ArtifactView{Spec: held.spec, State: held.state, DeleteReason: held.reason, Restricted: r.barred(held), Bundles: len(held.bundles), Registered: held.registered, Changed: held.changed}
	for _, stored := range held.instances {
		view.Instances = append(view.Instances, InstanceView{Custody: stored.custody, Lost: stored.lost, Reason: stored.reason})
	}
	return view, nil
}
