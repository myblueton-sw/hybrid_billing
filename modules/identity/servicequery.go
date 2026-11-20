package identity

import "sort"

func removeSorted(ids []string, id string) []string {
	at := sort.SearchStrings(ids, id)
	if at == len(ids) || ids[at] != id {
		return ids
	}
	copy(ids[at:], ids[at+1:])
	ids[len(ids)-1] = ""
	return ids[:len(ids)-1]
}

func (r *Registry) putCredential(id string, found *credential) {
	if r.credentials[id] != nil {
		r.removeCredential(id)
	}
	r.credentials[id] = found
	r.credentialIndex[found.user] = insertSorted(r.credentialIndex[found.user], id)
}

func (r *Registry) removeCredential(id string) {
	found := r.credentials[id]
	if found == nil {
		return
	}
	r.credentialIndex[found.user] = removeSorted(r.credentialIndex[found.user], id)
	if len(r.credentialIndex[found.user]) == 0 {
		delete(r.credentialIndex, found.user)
	}
	delete(r.credentials, id)
}

func serviceInfo(id string, found *user) ServiceInfo {
	return ServiceInfo{ID: id, State: found.state, Epoch: found.epoch, Creator: found.creator, Owner: found.owner}
}

func (h *Workspace) ServiceUser(actor Verifier, workspace, serviceID string) (ServiceInfo, error) {
	r, ok := h.open()
	if !ok {
		return ServiceInfo{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	found, err := r.service(actor, workspace, serviceID)
	if err != nil {
		return ServiceInfo{}, err
	}
	return serviceInfo(serviceID, found), nil
}

func (h *Workspace) ServiceUsers(actor Verifier, workspace, after string, limit int) ([]ServiceInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !r.pageable(after, limit) {
		return nil, ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return nil, err
	}
	ids := r.serviceIndex[workspace]
	start := beyond(ids, after)
	list := []ServiceInfo{}
	for _, id := range ids[start : start+min(limit, len(ids)-start)] {
		list = append(list, serviceInfo(id, r.users[id]))
	}
	return list, nil
}

func (r *Registry) credentialInfos(ids []string) []CredentialInfo {
	list := make([]CredentialInfo, 0, len(ids))
	for _, id := range ids {
		found := r.credentials[id]
		list = append(list, CredentialInfo{ID: id, Validity: found.validity, Revoked: found.revoked})
	}
	return list
}

func (h *Workspace) CredentialPage(actor Verifier, workspace, serviceID, after string, limit int) ([]CredentialInfo, error) {
	r, ok := h.open()
	if !ok {
		return nil, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !r.pageable(after, limit) {
		return nil, ErrInvalid
	}
	if _, err := r.service(actor, workspace, serviceID); err != nil {
		return nil, err
	}
	ids := r.credentialIndex[serviceID]
	start := beyond(ids, after)
	return r.credentialInfos(ids[start : start+min(limit, len(ids)-start)]), nil
}
