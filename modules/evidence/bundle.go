package evidence

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"sort"
	"strconv"
	"time"
)

type entry struct {
	role     Kind
	artifact *artifact
}

type bundle struct {
	spec      BundleSpec
	print     string
	state     BundleState
	reason    PurgeReason
	retained  time.Time
	counts    map[Kind]int
	digest    string
	gaps      []string
	holds     map[string]bool
	entries   []entry
	confirmed time.Time
	changed   time.Time
}

func (r *Registry) bundle(workspaceID, id string) *bundle {
	if ws := r.workspaces[workspaceID]; ws != nil {
		return ws.bundles[id]
	}
	return nil
}

func (b *bundle) protected(now time.Time) bool {
	return b.state == Confirmed && (now.Before(b.retained) || len(b.holds) > 0)
}

func (r *Registry) risky(b *bundle, a *artifact) bool {
	if !a.heldUntil(r.now, r.now) {
		return false
	}
	if len(b.holds) > 0 {
		return !a.copied()
	}
	return !a.heldUntil(r.now, b.retained)
}

func (r *Registry) exposure(b *bundle) (int, int) {
	broken, risk := 0, 0
	for _, item := range b.entries {
		switch {
		case !item.artifact.heldUntil(r.now, r.now):
			broken++
		case r.risky(b, item.artifact):
			risk++
		}
	}
	return broken, risk
}

func put(h hash.Hash, field string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(field)))
	h.Write(size[:])
	h.Write([]byte(field))
}

func flag(set bool, yes, no string) string {
	if set {
		return yes
	}
	return no
}

func ordered(items []Item) []Item {
	sorted := append([]Item(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Role != sorted[j].Role {
			return sorted[i].Role.rank() < sorted[j].Role.rank()
		}
		return sorted[i].Artifact < sorted[j].Artifact
	})
	return sorted
}

func (r *Registry) Confirm(spec BundleSpec) (Confirmation, error) {
	defer r.leave()
	if !r.enter() || !validID(spec.WorkspaceID) || !validID(spec.ID) || !validID(spec.Customer) || !validID(spec.Period) || spec.RetainUntil.IsZero() || len(spec.Items) == 0 {
		return Confirmation{}, ErrInvalid
	}
	for _, item := range spec.Items {
		if item.Role.rank() < 0 || !validID(item.Artifact) {
			return Confirmation{}, ErrInvalid
		}
	}
	if len(spec.Items) > r.limits.Items {
		return Confirmation{}, ErrLimit
	}
	items := ordered(spec.Items)
	prints := []string{spec.ID, spec.Customer, spec.Period, stamp(spec.RetainUntil), strconv.FormatBool(spec.FX), strconv.FormatBool(spec.Modules)}
	for _, item := range items {
		prints = append(prints, string(item.Role), item.Artifact)
	}
	print := join(prints...)
	if held := r.bundle(spec.WorkspaceID, spec.ID); held != nil {
		if held.print != print {
			return Confirmation{}, ErrConflict
		}
		return Confirmation{State: held.state, Gaps: r.listed(append([]string(nil), held.gaps...))}, nil
	}
	byCode := map[string][]Reason{}
	note := func(code string, role Kind, id string) {
		byCode[code] = append(byCode[code], Reason{Code: code, Role: role, Artifact: id})
	}
	counts := map[Kind]int{}
	named := map[string]int{}
	entries := make([]entry, 0, len(items))
	for _, item := range items {
		counts[item.Role]++
		named[item.Artifact]++
		held := r.artifact(spec.WorkspaceID, item.Artifact)
		switch {
		case held == nil:
			note(UnknownArtifact, item.Role, item.Artifact)
		case held.state == Deleted:
			note(DeletedArtifact, item.Role, item.Artifact)
		case held.spec.Kind != item.Role:
			note(WrongKind, item.Role, item.Artifact)
		case !held.heldUntil(r.now, spec.RetainUntil):
			note(NotHeld, item.Role, item.Artifact)
		}
		entries = append(entries, entry{item.Role, held})
	}
	twice := make([]string, 0, len(named))
	for id, times := range named {
		if times > 1 {
			twice = append(twice, id)
		}
	}
	sort.Strings(twice)
	for _, id := range twice {
		note(NamedTwice, "", id)
	}
	for _, role := range required {
		if counts[role] == 0 {
			note(RoleMissing, role, "")
		}
	}
	if spec.FX != (counts[FX] > 0) {
		note(FXMismatch, FX, "")
	}
	if spec.Modules != (counts[Module] > 0) {
		note(ModuleMismatch, Module, "")
	}
	if !spec.RetainUntil.After(r.now) {
		note(RetentionNotAhead, "", "")
	}
	var reasons []Reason
	for _, code := range []string{UnknownArtifact, DeletedArtifact, WrongKind, NamedTwice, RoleMissing, FXMismatch, ModuleMismatch, RetentionNotAhead, NotHeld} {
		reasons = append(reasons, byCode[code]...)
	}
	if len(reasons) > 0 {
		return Confirmation{Reasons: reasons}, ErrRejected
	}
	ws := r.workspaces[spec.WorkspaceID]
	if len(ws.bundles) >= r.limits.Bundles {
		return Confirmation{}, ErrLimit
	}
	h := sha256.New()
	for _, field := range []string{"evidence-bundle-v1", spec.WorkspaceID, spec.ID, spec.Customer, spec.Period, flag(spec.FX, "fx", "no_fx"), flag(spec.Modules, "modules", "no_modules")} {
		put(h, field)
	}
	for _, item := range entries {
		put(h, string(item.role))
		put(h, item.artifact.spec.ID)
		put(h, item.artifact.spec.Digest)
	}
	var open []string
	for id, known := range ws.gaps {
		if known.state == GapOpen && known.spec.Customer == spec.Customer && known.spec.Period == spec.Period {
			open = append(open, id)
		}
	}
	sort.Strings(open)
	stored := spec
	stored.Items, stored.RetainUntil = items, spec.RetainUntil.UTC()
	made := &bundle{spec: stored, print: print, state: Confirmed, retained: stored.RetainUntil, counts: counts, digest: hex.EncodeToString(h.Sum(nil)), gaps: open,
		holds: map[string]bool{}, entries: entries, confirmed: r.now, changed: r.now}
	ws.bundles[spec.ID] = made
	for _, item := range entries {
		item.artifact.bundles[spec.ID] = made
	}
	return Confirmation{State: Confirmed, Gaps: r.listed(append([]string(nil), open...))}, nil
}

func (r *Registry) Extend(workspaceID, id string, until time.Time) (Extension, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || until.IsZero() {
		return Extension{}, ErrInvalid
	}
	held := r.bundle(workspaceID, id)
	if held == nil {
		return Extension{}, ErrNotFound
	}
	if held.state == Purged {
		return Extension{}, ErrRejected
	}
	if until.After(held.retained) {
		held.retained, held.changed = until.UTC(), r.now
	}
	broken, risk := r.exposure(held)
	return Extension{Until: held.retained, NotHeld: broken, AtRisk: risk}, nil
}

func (r *Registry) Purge(workspaceID, id string, reason PurgeReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	held := r.bundle(workspaceID, id)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Purged {
		if held.reason != reason {
			return ErrConflict
		}
		return nil
	}
	if held.protected(r.now) {
		return ErrRejected
	}
	for _, item := range held.entries {
		delete(item.artifact.bundles, id)
	}
	held.state, held.reason, held.changed, held.entries = Purged, reason, r.now, nil
	held.spec.Items = nil
	return nil
}
