package relationships

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"slices"
	"sort"
)

const manifestTag = "hybrid-billing/relationships/manifest/v1"

func writeInt(h hash.Hash, value uint64) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], value)
	h.Write(buffer[:])
}

func writeString(h hash.Hash, value string) {
	writeInt(h, uint64(len(value)))
	h.Write([]byte(value))
}

func writeTimelines(h hash.Hash, timelines []Timeline) {
	writeInt(h, uint64(len(timelines)))
	for _, timeline := range timelines {
		writeString(h, string(timeline.Kind))
		writeString(h, timeline.Member)
		writeInt(h, uint64(len(timeline.Entries)))
		for _, entry := range timeline.Entries {
			writeString(h, entry.Target)
			h.Write([]byte(entry.Interval.Start().String()))
			if end, bounded := entry.Interval.End(); bounded {
				h.Write([]byte{1})
				h.Write([]byte(end.String()))
			} else {
				h.Write([]byte{0})
			}
		}
	}
}

func sum(h hash.Hash) [32]byte {
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

func manifestDigest(workspaceID string, base *revision, timelines []Timeline) [32]byte {
	h := sha256.New()
	writeString(h, manifestTag)
	writeString(h, workspaceID)
	writeInt(h, base.info.Number)
	h.Write(base.info.Digest[:])
	writeTimelines(h, timelines)
	return sum(h)
}

func (r *Registry) find(workspaceID, planID string) (*workspace, *plan, error) {
	if !validID(workspaceID) || !validID(planID) {
		return nil, nil, ErrInvalid
	}
	space := r.workspaces[workspaceID]
	if space == nil || space.plans[planID] == nil {
		return nil, nil, ErrNotFound
	}
	return space, space.plans[planID], nil
}

func (r *Registry) Draft(workspaceID, planID string, base uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(workspaceID) || !validID(planID) {
		return ErrInvalid
	}
	if existing := r.workspaces[workspaceID]; existing == nil {
		if base != 0 {
			return ErrConflict
		}
	} else if existing.plans[planID] != nil || base != existing.active {
		return ErrConflict
	} else if existing.open >= r.limits.OpenPlans {
		return ErrLimit
	}
	space := r.space(workspaceID)
	space.plans[planID] = &plan{state: PlanDraft, base: base, chunks: map[int]chunk{}, named: map[key]int{}}
	space.open++
	return nil
}

func (r *Registry) AddChunk(workspaceID, planID string, index int, timelines []Timeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, found, err := r.find(workspaceID, planID)
	if err != nil {
		return err
	}
	if found.state != PlanDraft {
		return ErrState
	}
	if index < 0 {
		return ErrInvalid
	}
	if index >= r.limits.ChunksPerPlan {
		return ErrLimit
	}
	stored, err := canonical(timelines, r.limits)
	if err != nil {
		return err
	}
	h := sha256.New()
	writeTimelines(h, stored)
	digest := sum(h)
	if existing, has := found.chunks[index]; has {
		if existing.digest != digest {
			return ErrConflict
		}
		return nil
	}
	for _, timeline := range stored {
		if _, named := found.named[key{timeline.Kind, timeline.Member}]; named {
			return ErrConflict
		}
	}
	for _, timeline := range stored {
		found.named[key{timeline.Kind, timeline.Member}] = index
	}
	found.chunks[index] = chunk{digest: digest, timelines: stored}
	return nil
}

func (space *workspace) close(found *plan, state PlanState) {
	found.state = state
	found.chunks, found.named, found.changed = nil, nil, nil
	space.open--
}

func reject(space *workspace, found *plan, violations []Violation, truncated bool) (Preview, error) {
	space.close(found, PlanRejected)
	found.violations, found.truncated = violations, truncated
	return Preview{Violations: slices.Clone(violations), Truncated: truncated}, ErrRejected
}

func (r *Registry) Seal(workspaceID, planID string, chunks int) (Preview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	space, found, err := r.find(workspaceID, planID)
	if err != nil {
		return Preview{}, err
	}
	if found.state != PlanDraft {
		return Preview{}, ErrState
	}
	if chunks < 0 {
		return Preview{}, ErrInvalid
	}
	if chunks > r.limits.ChunksPerPlan {
		return Preview{}, ErrLimit
	}
	if len(found.chunks) != chunks {
		return Preview{}, ErrIncomplete
	}
	for index := range found.chunks {
		if index >= chunks {
			return Preview{}, ErrIncomplete
		}
	}
	if found.base != space.active {
		return Preview{}, ErrConflict
	}
	base := space.revisions[space.active]
	manifest := []Timeline{}
	entries := 0
	for _, part := range found.chunks {
		manifest = append(manifest, part.timelines...)
		for _, timeline := range part.timelines {
			entries += len(timeline.Entries)
		}
	}
	if len(manifest) > r.limits.TimelinesPerPlan || entries > r.limits.EntriesPerPlan {
		return reject(space, found, []Violation{{Rule: RulePlanLimit}}, false)
	}
	sort.Slice(manifest, func(i, j int) bool { return less(manifest[i], manifest[j]) })
	changed := map[key][]Entry{}
	for _, timeline := range manifest {
		id := key{timeline.Kind, timeline.Member}
		if !slices.Equal(base.timelines[id], timeline.Entries) {
			changed[id] = timeline.Entries
		}
	}
	if len(changed) == 0 {
		return reject(space, found, []Violation{{Rule: RuleNoChange}}, false)
	}
	candidate := apply(base, changed)
	if violations, truncated := candidate.violations(r.limits); len(violations) > 0 {
		return reject(space, found, violations, truncated)
	}
	found.digest = manifestDigest(workspaceID, base, manifest)
	found.kinds, found.members, found.targets = map[Kind]KindSummary{}, map[Kind][]MemberChange{}, map[Kind][]string{}
	affected := map[key]struct{}{}
	for id, after := range changed {
		before := base.timelines[id]
		change := MemberChange{Member: id.id, Differs: differs(before, after), Before: sortedTargets(before), After: sortedTargets(after)}
		found.members[id.kind] = append(found.members[id.kind], change)
		summary := found.kinds[id.kind]
		summary.Changed++
		summary.EntriesBefore += len(before)
		summary.EntriesAfter += len(after)
		if len(change.Differs) > 0 && (!summary.HasEarliest || change.Differs[0].from < summary.Earliest.n) {
			summary.Earliest, summary.HasEarliest = Day{n: change.Differs[0].from}, true
		}
		found.kinds[id.kind] = summary
		was, is := spans(before), spans(after)
		for target, intervals := range was {
			if !slices.Equal(intervals, is[target]) {
				affected[key{id.kind, target}] = struct{}{}
			}
		}
		for target := range is {
			if _, had := was[target]; !had {
				affected[key{id.kind, target}] = struct{}{}
			}
		}
	}
	for kind, members := range found.members {
		sort.Slice(members, func(i, j int) bool { return members[i].Member < members[j].Member })
		found.members[kind] = members
	}
	for id := range affected {
		found.targets[id.kind] = append(found.targets[id.kind], id.id)
	}
	for kind := range found.targets {
		sort.Strings(found.targets[kind])
	}
	found.state, found.changed, found.chunks, found.named = PlanSealed, changed, nil, nil
	return Preview{Digest: found.digest, Kinds: copyKinds(found.kinds)}, nil
}

func copyKinds(kinds map[Kind]KindSummary) map[Kind]KindSummary {
	result := make(map[Kind]KindSummary, len(kinds))
	for kind, summary := range kinds {
		result[kind] = summary
	}
	return result
}

func (r *Registry) Activate(workspaceID, planID string, digest [32]byte, expected uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	space, found, err := r.find(workspaceID, planID)
	if err != nil {
		return 0, err
	}
	if found.state != PlanSealed {
		return 0, ErrState
	}
	if digest != found.digest || expected != found.base || found.base != space.active {
		return 0, ErrConflict
	}
	next := apply(space.revisions[space.active], found.changed)
	next.info = RevisionInfo{Number: space.active + 1, Recorded: r.clock(), Plan: planID, Digest: found.digest}
	space.revisions = append(space.revisions, next)
	space.active = next.info.Number
	space.close(found, PlanActivated)
	found.revision = next.info.Number
	return next.info.Number, nil
}

func (r *Registry) Discard(workspaceID, planID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	space, found, err := r.find(workspaceID, planID)
	if err != nil {
		return err
	}
	if found.state != PlanDraft && found.state != PlanSealed {
		return ErrState
	}
	space.close(found, PlanDiscarded)
	found.kinds, found.members, found.targets, found.digest = nil, nil, nil, [32]byte{}
	return nil
}

func (r *Registry) Plan(workspaceID, planID string) (PlanInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, found, err := r.find(workspaceID, planID)
	if err != nil {
		return PlanInfo{}, err
	}
	info := PlanInfo{State: found.state, Base: found.base, Chunks: []int{}, Digest: found.digest, Kinds: copyKinds(found.kinds),
		Revision: found.revision, Violations: slices.Clone(found.violations), Truncated: found.truncated}
	for index := range found.chunks {
		info.Chunks = append(info.Chunks, index)
	}
	sort.Ints(info.Chunks)
	return info, nil
}

func (r *Registry) page(workspaceID, planID string, kind Kind, after string, limit int) (*plan, error) {
	_, found, err := r.find(workspaceID, planID)
	if err != nil {
		return nil, err
	}
	if !known(kind) || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.PageSize {
		return nil, ErrInvalid
	}
	if found.state != PlanSealed && found.state != PlanActivated {
		return nil, ErrState
	}
	return found, nil
}

func (r *Registry) ChangedMembers(workspaceID, planID string, kind Kind, after string, limit int) ([]MemberChange, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.page(workspaceID, planID, kind, after, limit)
	if err != nil {
		return nil, err
	}
	all := found.members[kind]
	start := sort.Search(len(all), func(i int) bool { return all[i].Member > after })
	result := []MemberChange{}
	for _, change := range all[start:min(len(all), start+limit)] {
		result = append(result, MemberChange{Member: change.Member, Differs: slices.Clone(change.Differs), Before: slices.Clone(change.Before), After: slices.Clone(change.After)})
	}
	return result, nil
}

func (r *Registry) ChangedTargets(workspaceID, planID string, kind Kind, after string, limit int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.page(workspaceID, planID, kind, after, limit)
	if err != nil {
		return nil, err
	}
	all := found.targets[kind]
	start := sort.Search(len(all), func(i int) bool { return all[i] > after })
	return append([]string{}, all[start:min(len(all), start+limit)]...), nil
}
