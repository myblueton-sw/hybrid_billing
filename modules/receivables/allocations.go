package receivables

import "github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"

func sourceForm(source, ledger, revision, evidence string) bool {
	switch source {
	case SourceAuthority:
		return validID(ledger) && validID(revision) && (evidence == "" || validDigest(evidence))
	case SourceManual:
		return ledger == "" && revision == "" && validDigest(evidence)
	}
	return false
}

func allocationForm(a Allocation) bool {
	return validID(a.ID) && validID(a.Document) && allocationKinds[a.Kind] && validDate(a.Effective) && sourceForm(a.Source, a.Ledger, a.Revision, a.Evidence)
}

func (a *allocation) effective() bool {
	return a.view.State == StateApproved && (a.reversal == nil || a.reversal.view.State != StateApproved)
}

func (d *document) outstanding() exact.Fraction {
	total := d.amount
	for _, a := range d.allocations {
		if a.effective() {
			total = total.Minus(a.amount)
		}
	}
	return total
}

func (r *Registry) Allocate(workspace string, a Allocation, actor string) (AllocationView, error) {
	if !validID(workspace) || !validID(actor) || !allocationForm(a) {
		return AllocationView{}, ErrInvalid
	}
	amount, printed, ok := r.amount(a.Amount)
	if !ok {
		return AllocationView{}, ErrInvalid
	}
	a.Amount = printed
	now := r.enter()
	defer r.leave()
	d := r.find(workspace, a.Document)
	if d == nil {
		return AllocationView{}, ErrNotFound
	}
	s := r.workspaces[workspace]
	if old := s.allocations[a.ID]; old != nil {
		if old.view.Allocation != a || old.view.Author != actor {
			return AllocationView{}, ErrConflict
		}
		return old.view, nil
	}
	if d.view.Receivable.Obligation != ObligationPayable {
		return AllocationView{}, ErrRejected
	}
	if a.Effective > today(d, now) {
		return AllocationView{}, refuse(ReasonFutureEffective)
	}
	if a.Source == SourceAuthority && d.outstanding().Less(amount) {
		return AllocationView{}, refuse(ReasonExceedsOutstanding)
	}
	if len(d.allocations) >= r.limits.Allocations {
		return AllocationView{}, ErrLimit
	}
	made := &allocation{doc: d, amount: amount, view: AllocationView{Allocation: a, State: StatePending, Author: actor, Arrived: now, Digest: allocationDigest(workspace, a, actor)}}
	if a.Source == SourceAuthority {
		made.view.State, made.view.ApprovedAt = StateApproved, now
	}
	s.allocations[a.ID] = made
	d.allocations = append(d.allocations, made)
	return made.view, nil
}

func (r *Registry) ApproveAllocation(workspace, id, actor string) (AllocationView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return AllocationView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.allocations[id] == nil {
		return AllocationView{}, ErrNotFound
	}
	a := s.allocations[id]
	if a.view.State == StateApproved && a.view.Approver == actor {
		return a.view, nil
	}
	if a.view.State != StatePending || a.view.Author == actor {
		return AllocationView{}, ErrRejected
	}
	if a.doc.outstanding().Less(a.amount) {
		return AllocationView{}, refuse(ReasonExceedsOutstanding)
	}
	a.view.State, a.view.Approver, a.view.ApprovedAt = StateApproved, actor, now
	return a.view, nil
}

func (r *Registry) WithdrawAllocation(workspace, id, actor string) (AllocationView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return AllocationView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.allocations[id] == nil {
		return AllocationView{}, ErrNotFound
	}
	a := s.allocations[id]
	if a.view.State == StateWithdrawn && a.view.WithdrawnBy == actor {
		return a.view, nil
	}
	if a.view.State != StatePending {
		return AllocationView{}, ErrRejected
	}
	a.view.State, a.view.WithdrawnBy = StateWithdrawn, actor
	return a.view, nil
}

func reversalForm(v Reversal) bool {
	return validID(v.ID) && validID(v.Allocation) && validDate(v.Effective) && sourceForm(v.Source, v.Ledger, v.Revision, v.Evidence)
}

func (r *Registry) Reverse(workspace string, v Reversal, actor string) (ReversalView, error) {
	if !validID(workspace) || !validID(actor) || !reversalForm(v) {
		return ReversalView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.allocations[v.Allocation] == nil {
		return ReversalView{}, ErrNotFound
	}
	a := s.allocations[v.Allocation]
	if old := s.reversals[v.ID]; old != nil {
		if old.view.Reversal != v || old.view.Author != actor {
			return ReversalView{}, ErrConflict
		}
		return old.view, nil
	}
	if a.view.State != StateApproved || a.reversal != nil && a.reversal.view.State != StateWithdrawn || v.Effective < a.view.Allocation.Effective || v.Source != a.view.Allocation.Source {
		return ReversalView{}, ErrRejected
	}
	if v.Effective > today(a.doc, now) {
		return ReversalView{}, refuse(ReasonFutureEffective)
	}
	if len(a.doc.reversals) >= r.limits.Allocations {
		return ReversalView{}, ErrLimit
	}
	made := &reversal{alloc: a, view: ReversalView{Reversal: v, State: StatePending, Author: actor, Arrived: now, Digest: reversalDigest(workspace, v, actor)}}
	if v.Source == SourceAuthority {
		made.view.State, made.view.ApprovedAt = StateApproved, now
	}
	s.reversals[v.ID] = made
	a.reversal = made
	a.view.Reversal = v.ID
	a.doc.reversals = append(a.doc.reversals, made)
	return made.view, nil
}

func (r *Registry) ApproveReversal(workspace, id, actor string) (ReversalView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ReversalView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.reversals[id] == nil {
		return ReversalView{}, ErrNotFound
	}
	v := s.reversals[id]
	if v.view.State == StateApproved && v.view.Approver == actor {
		return v.view, nil
	}
	if v.view.State != StatePending || v.view.Author == actor {
		return ReversalView{}, ErrRejected
	}
	v.view.State, v.view.Approver, v.view.ApprovedAt = StateApproved, actor, now
	return v.view, nil
}

func (r *Registry) WithdrawReversal(workspace, id, actor string) (ReversalView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ReversalView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.reversals[id] == nil {
		return ReversalView{}, ErrNotFound
	}
	v := s.reversals[id]
	if v.view.State == StateWithdrawn && v.view.WithdrawnBy == actor {
		return v.view, nil
	}
	if v.view.State != StatePending {
		return ReversalView{}, ErrRejected
	}
	v.view.State, v.view.WithdrawnBy = StateWithdrawn, actor
	v.alloc.view.Reversal = ""
	return v.view, nil
}
