package organizations

import "sort"

const PartyPageLimit = 1000

func (r *Registry) putParty(key scoped, record versioned[Party]) {
	ids := r.partyIndex[key.workspace]
	at := sort.SearchStrings(ids, key.id)
	if at == len(ids) || ids[at] != key.id {
		ids = append(ids, "")
		copy(ids[at+1:], ids[at:])
		ids[at] = key.id
		r.partyIndex[key.workspace] = ids
	}
	r.parties[key] = record
	r.queryChanged(key.workspace)
}

func (r *Registry) EligibleParty(workspaceID, id string) (Party, uint64, error) {
	if !validID(workspaceID) || !validID(id) {
		return Party{}, 0, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.writable(workspaceID); err != nil {
		return Party{}, 0, err
	}
	party, err := r.activeParty(workspaceID, id)
	if err != nil {
		return Party{}, 0, err
	}
	return party, r.parties[scoped{workspaceID, id}].version, nil
}

func (r *Registry) PartyPage(workspaceID, after string, limit int) ([]Party, error) {
	if !validID(workspaceID) || after != "" && !validID(after) || limit < 1 || limit > PartyPageLimit {
		return nil, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, exists := r.workspaces[workspaceID]; !exists {
		return nil, ErrNotFound
	}
	ids := r.partyIndex[workspaceID]
	start := sort.Search(len(ids), func(i int) bool { return ids[i] > after })
	rows := make([]Party, 0, min(limit, len(ids)-start))
	for _, id := range ids[start : start+min(limit, len(ids)-start)] {
		rows = append(rows, r.parties[scoped{workspaceID, id}].value)
	}
	return rows, nil
}
