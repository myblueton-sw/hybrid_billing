package approvals

import "sort"

func (r *Registry) ProposalOf(workspace, id string) (ProposalView, error) {
	if !validID(workspace) || !validID(id) {
		return ProposalView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.proposals[id] == nil {
		return ProposalView{}, ErrNotFound
	}
	return viewProposal(s.proposals[id]), nil
}

func (r *Registry) Proposals(workspace string) ([]ProposalView, error) {
	if !validID(workspace) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	out := make([]ProposalView, 0, len(s.proposals))
	for _, p := range s.proposals {
		out = append(out, viewProposal(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Proposal.ID < out[j].Proposal.ID })
	return out, nil
}

func (r *Registry) PolicyAt(workspace, stage string, revision int) (Policy, error) {
	if !validID(workspace) || !stages[stage] || revision < 1 {
		return Policy{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return Policy{}, ErrNotFound
	}
	var found *Policy
	for _, p := range s.proposals {
		if p.State != StateConfirmed || p.Proposal.Kind != ProposePolicy || p.Proposal.Policy.Stage != stage || p.Proposal.Policy.Revision != revision {
			continue
		}
		policy := p.Proposal.Policy
		if found != nil && !samePolicy(*found, policy) {
			return Policy{}, ErrConflict
		}
		found = &policy
	}
	if found == nil {
		return Policy{}, ErrNotFound
	}
	out := *found
	out.Required = append([]string(nil), out.Required...)
	return out, nil
}
