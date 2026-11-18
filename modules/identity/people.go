package identity

import "time"

const PeopleLimit = 256

type Person struct {
	User   UserInfo
	Member *MemberInfo
}

func (r *Registry) memberInfo(userID string, found *membership, now time.Time) MemberInfo {
	return MemberInfo{User: userID, Role: found.role, State: found.state, Version: found.version, UserState: r.users[userID].state, From: found.from,
		Until: found.until, SuspendedFor: found.suspendedFor, Effective: effective(found, now)}
}

func userInfo(id string, found *user) UserInfo {
	return UserInfo{ID: id, Kind: found.kind, State: found.state, Epoch: found.epoch, Workspace: found.workspace}
}

func (h *Workspace) People(actor Verifier, workspace string, ids []string) ([]Person, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || len(ids) > PeopleLimit {
		return nil, ErrInvalid
	}
	for _, id := range ids {
		if !validID(id) {
			return nil, ErrInvalid
		}
	}
	now := r.clock()
	if _, err := r.administrator(actor, workspace, now); err != nil {
		return nil, err
	}
	out := make([]Person, 0, len(ids))
	for _, id := range ids {
		found := r.users[id]
		if found == nil {
			continue
		}
		person := Person{User: userInfo(id, found)}
		if own := r.member(workspace, id); own != nil {
			info := r.memberInfo(id, own, now)
			person.Member = &info
		}
		out = append(out, person)
	}
	return out, nil
}

func (h *Installation) User(id string) (UserInfo, bool, error) {
	r, ok := h.open()
	if !ok {
		return UserInfo{}, false, ErrUnavailable
	}
	defer r.mu.Unlock()
	found := r.users[id]
	if found == nil {
		return UserInfo{}, false, nil
	}
	return userInfo(id, found), true, nil
}
