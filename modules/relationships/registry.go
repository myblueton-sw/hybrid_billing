package relationships

import (
	"sort"
	"sync"
	"time"
)

type key struct {
	kind Kind
	id   string
}

type revision struct {
	info      RevisionInfo
	timelines map[key][]Entry
	members   map[key][]string
}

type chunk struct {
	digest    [32]byte
	timelines []Timeline
}

type plan struct {
	state      PlanState
	base       uint64
	chunks     map[int]chunk
	named      map[key]int
	digest     [32]byte
	kinds      map[Kind]KindSummary
	changed    map[key][]Entry
	members    map[Kind][]MemberChange
	targets    map[Kind][]string
	revision   uint64
	violations []Violation
	truncated  bool
}

type workspace struct {
	revisions []*revision
	active    uint64
	plans     map[string]*plan
	open      int
}

type Registry struct {
	mu         sync.RWMutex
	limits     Limits
	clock      func() time.Time
	workspaces map[string]*workspace
}

var empty = &revision{timelines: map[key][]Entry{}, members: map[key][]string{}}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, workspaces: map[string]*workspace{}}, nil
}

func (r *Registry) space(id string) *workspace {
	found := r.workspaces[id]
	if found == nil {
		found = &workspace{revisions: []*revision{empty}, plans: map[string]*plan{}}
		r.workspaces[id] = found
	}
	return found
}

func (r *Registry) at(workspaceID string, number uint64) (*revision, error) {
	if !validID(workspaceID) {
		return nil, ErrInvalid
	}
	found := r.workspaces[workspaceID]
	if found == nil {
		if number == 0 {
			return empty, nil
		}
		return nil, ErrNotFound
	}
	if number >= uint64(len(found.revisions)) {
		return nil, ErrNotFound
	}
	return found.revisions[number], nil
}

func (r *Registry) Active(workspaceID string) (uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !validID(workspaceID) {
		return 0, ErrInvalid
	}
	if found := r.workspaces[workspaceID]; found != nil {
		return found.active, nil
	}
	return 0, nil
}

func (r *Registry) Revision(workspaceID string, number uint64) (RevisionInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return RevisionInfo{}, err
	}
	return found.info, nil
}

func targetOn(entries []Entry, day Day) (string, bool) {
	for _, entry := range entries {
		if entry.Interval.contains(day) {
			return entry.Target, true
		}
	}
	return "", false
}

func (r *Registry) TargetOn(workspaceID string, number uint64, kind Kind, member string, day Day) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return "", false, err
	}
	if !known(kind) || !exclusive(kind) || !validID(member) || !day.valid() {
		return "", false, ErrInvalid
	}
	target, has := targetOn(found.timelines[key{kind, member}], day)
	return target, has, nil
}

func (r *Registry) Segments(workspaceID string, number uint64, kind Kind, member string, interval Interval) ([]Segment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return nil, err
	}
	if !known(kind) || !exclusive(kind) || !validID(member) || !interval.valid() || !interval.bounded() {
		return nil, ErrInvalid
	}
	segments := []Segment{}
	cursor := interval.from
	for _, entry := range found.timelines[key{kind, member}] {
		piece, overlaps := entry.Interval.intersect(interval)
		if !overlaps {
			continue
		}
		if cursor < piece.from {
			segments = append(segments, Segment{Interval: Interval{from: cursor, end: piece.from}})
		}
		segments = append(segments, Segment{Interval: piece, Target: entry.Target, Has: true})
		cursor = piece.end
	}
	if cursor < interval.end {
		segments = append(segments, Segment{Interval: Interval{from: cursor, end: interval.end}})
	}
	return segments, nil
}

func (r *Registry) TargetsOn(workspaceID string, number uint64, kind Kind, member string, day Day) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return nil, err
	}
	if !known(kind) || !validID(member) || !day.valid() {
		return nil, ErrInvalid
	}
	targets := []string{}
	for _, entry := range found.timelines[key{kind, member}] {
		if entry.Interval.contains(day) {
			targets = append(targets, entry.Target)
		}
	}
	sort.Strings(targets)
	return targets, nil
}

func (found *revision) membersOn(kind Kind, target string, day Day) []string {
	members := []string{}
	for _, member := range found.members[key{kind, target}] {
		for _, entry := range found.timelines[key{kind, member}] {
			if entry.Target == target && entry.Interval.contains(day) {
				members = append(members, member)
				break
			}
		}
	}
	return members
}

func (r *Registry) MembersOn(workspaceID string, number uint64, kind Kind, target string, day Day) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return nil, err
	}
	if !known(kind) || !validID(target) || !day.valid() {
		return nil, ErrInvalid
	}
	return found.membersOn(kind, target, day), nil
}

func (r *Registry) Chain(workspaceID string, number uint64, member string, day Day) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return nil, err
	}
	if !validID(member) || !day.valid() {
		return nil, ErrInvalid
	}
	chain := []string{}
	for current := member; len(chain) <= MaxDepth; {
		parent, has := targetOn(found.timelines[key{Hierarchy, current}], day)
		if !has {
			break
		}
		chain = append(chain, parent)
		current = parent
	}
	return chain, nil
}

func (r *Registry) Union(workspaceID string, number uint64, groups []string, day Day) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.at(workspaceID, number)
	if err != nil {
		return nil, err
	}
	if !day.valid() {
		return nil, ErrInvalid
	}
	seen := map[string]struct{}{}
	for _, group := range groups {
		if !validID(group) {
			return nil, ErrInvalid
		}
		for _, member := range found.membersOn(ViewGroup, group, day) {
			seen[member] = struct{}{}
		}
	}
	union := make([]string, 0, len(seen))
	for member := range seen {
		union = append(union, member)
	}
	sort.Strings(union)
	return union, nil
}
