package approvals

import "sort"

func (r *Registry) Open(workspace string, admins []string, actor Actor) ([]string, error) {
	if !validID(workspace) || !validActor(actor) || len(admins) < 2 || len(admins) > r.limits.Admins {
		return nil, ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range admins {
		if !validID(id) || seen[id] {
			return nil, ErrInvalid
		}
		seen[id] = true
	}
	r.enter()
	defer r.leave()
	if actor.Kind != KindHuman {
		return nil, refuse(ReasonAINotAllowed)
	}
	if s := r.workspaces[workspace]; s != nil {
		return nil, ErrConflict
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	r.workspaces[workspace] = &space{admins: seen, proposals: map[string]*ProposalView{}, policies: map[string]Policy{}, authorities: map[string]*authority{},
		requests: map[string]*request{}, targets: map[[2]string][]*request{}, executions: map[string]*execution{}, keys: map[[2]string]string{}}
	return adminList(seen), nil
}

func adminList(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) Admins(workspace string) ([]string, error) {
	if !validID(workspace) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	return adminList(s.admins), nil
}

func policyForm(p Policy) bool {
	if !stages[p.Stage] || p.Revision < 1 || p.Approvals < 1 || p.Approvals > 3 || p.Validity < minValidity || p.Validity > maxValidity || len(p.Required) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, k := range p.Required {
		if !digestKinds[k] || seen[k] {
			return false
		}
		seen[k] = true
	}
	return seen["result"]
}

func (r *Registry) authorityForm(a Authority) bool {
	if !validID(a.ID) || !validID(a.Approver) || !stages[a.Stage] || !validCurrency(a.Currency) || !validDate(a.From) || a.To != "" && (!validDate(a.To) || a.To < a.From) {
		return false
	}
	_, _, ok := r.number(a.Limit, false)
	return ok
}

func (r *Registry) proposalForm(p Proposal) (Proposal, bool) {
	if !validID(p.ID) {
		return p, false
	}
	empty := Policy{}
	switch p.Kind {
	case ProposePolicy:
		ok := policyForm(p.Policy) && p.Authority == (Authority{}) && p.Admin == ""
		p.Policy.Required = append([]string(nil), p.Policy.Required...)
		sort.Strings(p.Policy.Required)
		return p, ok
	case ProposeGrant:
		if !r.authorityForm(p.Authority) || !samePolicy(p.Policy, empty) || p.Admin != "" {
			return p, false
		}
		_, printed, _ := r.number(p.Authority.Limit, false)
		p.Authority.Limit = printed
		return p, true
	case ProposeAddAdmin, ProposeRemoveAdmin:
		return p, validID(p.Admin) && samePolicy(p.Policy, empty) && p.Authority == (Authority{})
	}
	return p, false
}

func samePolicy(a, b Policy) bool {
	if a.Stage != b.Stage || a.Revision != b.Revision || a.Approvals != b.Approvals || a.Validity != b.Validity || len(a.Required) != len(b.Required) {
		return false
	}
	for index := range a.Required {
		if a.Required[index] != b.Required[index] {
			return false
		}
	}
	return true
}

func sameProposal(a, b Proposal) bool {
	return a.ID == b.ID && a.Kind == b.Kind && samePolicy(a.Policy, b.Policy) && a.Authority == b.Authority && a.Admin == b.Admin
}

func viewProposal(p *ProposalView) ProposalView {
	v := *p
	v.Proposal.Policy.Required = append([]string(nil), p.Proposal.Policy.Required...)
	return v
}

func (r *Registry) Propose(workspace string, p Proposal, actor Actor) (ProposalView, error) {
	if !validID(workspace) || !validActor(actor) {
		return ProposalView{}, ErrInvalid
	}
	p, ok := r.proposalForm(p)
	if !ok {
		return ProposalView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return ProposalView{}, ErrNotFound
	}
	if old := s.proposals[p.ID]; old != nil {
		if !sameProposal(old.Proposal, p) || old.Proposer != actor.ID {
			return ProposalView{}, ErrConflict
		}
		return viewProposal(old), nil
	}
	if actor.Kind != KindHuman {
		return ProposalView{}, refuse(ReasonAINotAllowed)
	}
	if !s.admins[actor.ID] {
		return ProposalView{}, refuse(ReasonNotAdmin)
	}
	if p.Kind == ProposeGrant && p.Authority.Approver == actor.ID {
		return ProposalView{}, refuse(ReasonGranteeActs)
	}
	if len(s.proposals) >= r.limits.Proposals {
		return ProposalView{}, ErrLimit
	}
	v := &ProposalView{Proposal: p, State: StatePending, Proposer: actor.ID, Written: now, Digest: proposalDigest(workspace, p, actor.ID)}
	s.proposals[p.ID] = v
	return viewProposal(v), nil
}

func (r *Registry) Confirm(workspace, id string, actor Actor) (ProposalView, error) {
	if !validID(workspace) || !validID(id) || !validActor(actor) {
		return ProposalView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.proposals[id] == nil {
		return ProposalView{}, ErrNotFound
	}
	v := s.proposals[id]
	if v.State == StateConfirmed {
		if v.Confirmer == actor.ID {
			return viewProposal(v), nil
		}
		return ProposalView{}, ErrRejected
	}
	if actor.Kind != KindHuman {
		return ProposalView{}, refuse(ReasonAINotAllowed)
	}
	if !s.admins[actor.ID] || actor.ID == v.Proposer {
		return ProposalView{}, refuse(ReasonNotAdmin)
	}
	if !s.admins[v.Proposer] {
		return ProposalView{}, refuse(ReasonStaleProposal)
	}
	p := v.Proposal
	switch p.Kind {
	case ProposePolicy:
		if p.Policy.Revision != s.policies[p.Policy.Stage].Revision+1 {
			return ProposalView{}, refuse(ReasonStaleProposal)
		}
	case ProposeGrant:
		if p.Authority.Approver == actor.ID {
			return ProposalView{}, refuse(ReasonGranteeActs)
		}
		if s.authorities[p.Authority.ID] != nil {
			return ProposalView{}, ErrConflict
		}
		if len(s.authorities) >= r.limits.Authorities {
			return ProposalView{}, ErrLimit
		}
	case ProposeAddAdmin:
		if s.admins[p.Admin] {
			return ProposalView{}, refuse(ReasonStaleProposal)
		}
		if len(s.admins) >= r.limits.Admins {
			return ProposalView{}, ErrLimit
		}
	case ProposeRemoveAdmin:
		if !s.admins[p.Admin] {
			return ProposalView{}, refuse(ReasonStaleProposal)
		}
		if len(s.admins) <= 2 {
			return ProposalView{}, refuse(ReasonTooFewAdmins)
		}
	}
	switch p.Kind {
	case ProposePolicy:
		s.policies[p.Policy.Stage] = p.Policy
		for _, req := range s.requests {
			if req.view.Subject.Stage == p.Policy.Stage && (req.view.State == StatePending || req.view.State == StateApproved) {
				req.end(StateSuperseded, actor.ID, now)
			}
		}
	case ProposeGrant:
		limit, _, _ := r.number(p.Authority.Limit, false)
		s.authorities[p.Authority.ID] = &authority{limit: limit, view: AuthorityView{Authority: p.Authority, Proposer: v.Proposer, Confirmer: actor.ID, Granted: now}}
	case ProposeAddAdmin:
		s.admins[p.Admin] = true
	case ProposeRemoveAdmin:
		delete(s.admins, p.Admin)
	}
	v.State, v.Confirmer, v.Confirmed = StateConfirmed, actor.ID, now
	return viewProposal(v), nil
}

func (r *Registry) Revoke(workspace, id string, actor Actor) (AuthorityView, error) {
	if !validID(workspace) || !validID(id) || !validActor(actor) {
		return AuthorityView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.authorities[id] == nil {
		return AuthorityView{}, ErrNotFound
	}
	a := s.authorities[id]
	if !a.view.Revoked.IsZero() {
		if a.view.RevokedBy == actor.ID {
			return a.view, nil
		}
		return AuthorityView{}, ErrRejected
	}
	if actor.Kind != KindHuman {
		return AuthorityView{}, refuse(ReasonAINotAllowed)
	}
	if !s.admins[actor.ID] && actor.ID != a.view.Authority.Approver {
		return AuthorityView{}, refuse(ReasonNotAdmin)
	}
	a.view.RevokedBy, a.view.Revoked = actor.ID, now
	return a.view, nil
}

func (r *Registry) AuthorityOf(workspace, id string) (AuthorityView, error) {
	if !validID(workspace) || !validID(id) {
		return AuthorityView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.authorities[id] == nil {
		return AuthorityView{}, ErrNotFound
	}
	return s.authorities[id].view, nil
}

func (r *Registry) PolicyOf(workspace, stage string) (Policy, error) {
	if !validID(workspace) || !stages[stage] {
		return Policy{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return Policy{}, ErrNotFound
	}
	p, ok := s.policies[stage]
	if !ok {
		return Policy{}, refuse(ReasonNoPolicy)
	}
	p.Required = append([]string(nil), p.Required...)
	return p, nil
}
