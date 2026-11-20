package approvals

import "time"

func (r *Registry) AuthorityAt(workspace, id string) (AuthorityView, time.Time, error) {
	if !validID(workspace) || !validID(id) {
		return AuthorityView{}, time.Time{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.authorities[id] == nil {
		return AuthorityView{}, time.Time{}, ErrNotFound
	}
	return s.authorities[id].view, now, nil
}
