package relationships

import (
	"fmt"
	"sort"
)

func (d Day) ISO() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.n/10000, d.n/100%100, d.n%100)
}

func (r *Registry) Timelines(workspaceID string, number uint64, kind Kind, after string, limit int) ([]Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !known(kind) || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.PageSize {
		return nil, ErrInvalid
	}
	found, err := r.at(workspaceID, number)
	if err != nil {
		return nil, err
	}
	members := []string{}
	for k := range found.timelines {
		if k.kind == kind && k.id > after {
			members = append(members, k.id)
		}
	}
	sort.Strings(members)
	if len(members) > limit {
		members = members[:limit]
	}
	out := make([]Timeline, 0, len(members))
	for _, member := range members {
		out = append(out, Timeline{Kind: kind, Member: member, Entries: append([]Entry(nil), found.timelines[key{kind: kind, id: member}]...)})
	}
	return out, nil
}
