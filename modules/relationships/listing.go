package relationships

import (
	"fmt"
	"sort"
	"sync"
)

func (d Day) ISO() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.n/10000, d.n/100%100, d.n%100)
}

type index struct {
	once    sync.Once
	members map[Kind][]string
	targets map[Kind][]string
	all     []string
}

func (found *revision) indexed() *index {
	found.index.once.Do(func() {
		members, targets, seen := map[Kind][]string{}, map[Kind][]string{}, map[string]struct{}{}
		for id := range found.timelines {
			members[id.kind] = append(members[id.kind], id.id)
			seen[id.id] = struct{}{}
		}
		for id := range found.members {
			targets[id.kind] = append(targets[id.kind], id.id)
		}
		all := make([]string, 0, len(seen))
		for id := range seen {
			all = append(all, id)
		}
		for _, list := range []map[Kind][]string{members, targets} {
			for _, ids := range list {
				sort.Strings(ids)
			}
		}
		sort.Strings(all)
		found.index.members, found.index.targets, found.index.all = members, targets, all
	})
	return &found.index
}

func after(sorted []string, cursor string, limit int) []string {
	start := sort.Search(len(sorted), func(i int) bool { return sorted[i] > cursor })
	return append([]string{}, sorted[start:min(len(sorted), start+limit)]...)
}

func (r *Registry) listed(workspaceID string, number uint64, cursor string, limit int) (*revision, error) {
	if (cursor != "" && !validID(cursor)) || limit < 1 || limit > r.limits.PageSize {
		return nil, ErrInvalid
	}
	return r.at(workspaceID, number)
}

func (r *Registry) Timelines(workspaceID string, number uint64, kind Kind, cursor string, limit int) ([]Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !known(kind) {
		return nil, ErrInvalid
	}
	found, err := r.listed(workspaceID, number, cursor, limit)
	if err != nil {
		return nil, err
	}
	members := after(found.indexed().members[kind], cursor, limit)
	out := make([]Timeline, 0, len(members))
	for _, member := range members {
		out = append(out, Timeline{Kind: kind, Member: member, Entries: append([]Entry(nil), found.timelines[key{kind: kind, id: member}]...)})
	}
	return out, nil
}

func (r *Registry) Targets(workspaceID string, number uint64, kind Kind, cursor string, limit int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !known(kind) {
		return nil, ErrInvalid
	}
	found, err := r.listed(workspaceID, number, cursor, limit)
	if err != nil {
		return nil, err
	}
	return after(found.indexed().targets[kind], cursor, limit), nil
}

func (r *Registry) Members(workspaceID string, number uint64, cursor string, limit int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found, err := r.listed(workspaceID, number, cursor, limit)
	if err != nil {
		return nil, err
	}
	return after(found.indexed().all, cursor, limit), nil
}

func (r *Registry) TimelineOf(workspaceID string, number uint64, kind Kind, member string) (Timeline, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !known(kind) || !validID(member) {
		return Timeline{}, false, ErrInvalid
	}
	found, err := r.at(workspaceID, number)
	if err != nil {
		return Timeline{}, false, err
	}
	entries, has := found.timelines[key{kind, member}]
	if !has {
		return Timeline{}, false, nil
	}
	return Timeline{Kind: kind, Member: member, Entries: append([]Entry(nil), entries...)}, true, nil
}

func (r *Registry) IsTarget(workspaceID string, number uint64, kind Kind, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !known(kind) || !validID(id) {
		return false, ErrInvalid
	}
	found, err := r.at(workspaceID, number)
	if err != nil {
		return false, err
	}
	targets := found.indexed().targets[kind]
	at := sort.SearchStrings(targets, id)
	return at < len(targets) && targets[at] == id, nil
}
