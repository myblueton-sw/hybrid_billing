package identity

import (
	"math"
	"sort"
	"time"
)

type credentialExpiry struct {
	id    string
	until time.Time
}

type ServiceControlInfo struct {
	OperatorBlocked       bool
	LiveCredentials       int
	CredentialLimit       int
	CanAllocateCredential bool
	CanTransition         bool
}

func expiryBefore(a, b credentialExpiry) bool {
	return a.until.Before(b.until) || a.until.Equal(b.until) && a.id < b.id
}

func (r *Registry) indexCredentialExpiry(id string, found *credential) {
	if found.revoked {
		return
	}
	point := credentialExpiry{id, found.validity.NotAfter}
	rows := r.credentialExpiry[found.user]
	at := sort.Search(len(rows), func(i int) bool { return !expiryBefore(rows[i], point) })
	rows = append(rows, credentialExpiry{})
	copy(rows[at+1:], rows[at:])
	rows[at] = point
	r.credentialExpiry[found.user] = rows
}

func (r *Registry) unindexCredentialExpiry(id string, found *credential) {
	point := credentialExpiry{id, found.validity.NotAfter}
	rows := r.credentialExpiry[found.user]
	at := sort.Search(len(rows), func(i int) bool { return !expiryBefore(rows[i], point) })
	if at == len(rows) || rows[at].id != id || !rows[at].until.Equal(point.until) {
		return
	}
	copy(rows[at:], rows[at+1:])
	rows[len(rows)-1] = credentialExpiry{}
	rows = rows[:len(rows)-1]
	if len(rows) == 0 {
		delete(r.credentialExpiry, found.user)
	} else {
		r.credentialExpiry[found.user] = rows
	}
}

func (r *Registry) liveCredentialCount(service string, at time.Time) int {
	rows := r.credentialExpiry[service]
	return len(rows) - sort.Search(len(rows), func(i int) bool { return at.Before(rows[i].until) })
}

func (h *Workspace) ServiceControls(actor Verifier, workspace, service string) (ServiceControlInfo, error) {
	r, ok := h.open()
	if !ok {
		return ServiceControlInfo{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	found, err := r.service(actor, workspace, service)
	if err != nil {
		return ServiceControlInfo{}, err
	}
	return ServiceControlInfo{OperatorBlocked: found.operatorBlock, LiveCredentials: r.liveCredentialCount(service, r.clock()), CredentialLimit: r.limits.CredentialsPerServiceUser, CanAllocateCredential: r.serial < math.MaxUint64, CanTransition: found.epoch < math.MaxUint64}, nil
}
