package approvals

func (r *Registry) CheckAuthorityCoverage(workspace, authorityID, approver, stage, currency, amount string) error {
	if !validID(workspace) || !validID(authorityID) || !validID(approver) || !stages[stage] || !validCurrency(currency) {
		return ErrInvalid
	}
	value, _, ok := r.number(amount, true)
	if !ok {
		return ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.authorities[authorityID] == nil {
		return ErrNotFound
	}
	a := s.active(authorityID, now)
	if a == nil || a.view.Authority.Approver != approver || a.view.Authority.Stage != stage || a.view.Authority.Currency != currency || a.limit.Less(absolute(value)) {
		return refuse(ReasonNoAuthority)
	}
	return nil
}
