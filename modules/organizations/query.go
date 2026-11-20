package organizations

import (
	"errors"
	"math"
)

var ErrQueryUnavailable = errors.New("organization query source is unavailable")

type ContractPartyRefs struct {
	CustomerID string
	SellerID   string
}

func (r *Registry) ContractParties(workspace, id string, revision uint64) (ContractPartyRefs, error) {
	if r == nil {
		return ContractPartyRefs{}, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	history := r.contracts[scoped{workspace, id}]
	if revision == 0 || revision > uint64(len(history)) {
		return ContractPartyRefs{}, ErrNotFound
	}
	row := history[revision-1]
	return ContractPartyRefs{CustomerID: row.CustomerID, SellerID: row.SellerID}, nil
}

func (r *Registry) queryChanged(workspace string) {
	if r.queryVersions[workspace] < math.MaxUint64 {
		r.queryVersions[workspace]++
	}
}

func (r *Registry) QueryVersion(workspace string) (uint64, error) {
	if r == nil || !validID(workspace) {
		return 0, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.workspaces[workspace].version == 0 {
		return 0, ErrNotFound
	}
	version := r.queryVersions[workspace]
	if version == math.MaxUint64 {
		return 0, ErrQueryUnavailable
	}
	return version, nil
}
