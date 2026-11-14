package marketplace

func (r *Registry) settlementForm(f SettlementFacts) (SettlementFacts, bool) {
	out := f
	texts := []*string{&out.Gross, &out.Fee, &out.Refund, &out.Tax, &out.Hold, &out.Net}
	for _, text := range texts {
		canonical, ok := r.amount(*text)
		if !ok {
			return SettlementFacts{}, false
		}
		*text = canonical
	}
	if rank[f.State] == 0 || !validCurrency(f.Currency) {
		return SettlementFacts{}, false
	}
	if (f.State == SettlementExpected) != (f.Report == "") || !optionalID(f.Report) {
		return SettlementFacts{}, false
	}
	if f.State != SettlementPaid {
		return out, f.Payout == "" && f.Received == ""
	}
	received, ok := r.amount(f.Received)
	if !validID(f.Payout) || !ok {
		return SettlementFacts{}, false
	}
	out.Received = received
	return out, true
}

func (r *Registry) judgeSettlement(f SettlementFacts) (string, []string) {
	reasons := []string{}
	if !same(r.sum(f.Gross), r.sum(f.Fee, f.Refund, f.Tax, f.Hold, f.Net)) {
		reasons = append(reasons, ReasonNetNotPreserved)
	}
	if f.State == SettlementPaid && !same(r.sum(f.Received), r.sum(f.Net)) {
		reasons = append(reasons, ReasonReceivedDiffers)
	}
	if len(reasons) > 0 {
		return JudgedHeld, reasons
	}
	return JudgedConsistent, reasons
}

func (r *Registry) payoutReused(workspace string, own *stored, id, payout string) bool {
	for _, st := range r.workspaces[workspace].transactions {
		if st.t.Provider != own.t.Provider {
			continue
		}
		for other, s := range st.settlements {
			if st == own && other == id {
				continue
			}
			top := s.revisions[len(s.revisions)-1].facts
			if top.State == SettlementPaid && top.Payout == payout {
				return true
			}
		}
	}
	return false
}

func viewSettlement(transaction, id string, number int, rev *settlementRevision) SettlementView {
	return SettlementView{Transaction: transaction, Settlement: id, Revision: number, Facts: rev.facts, Judged: rev.judged, Reasons: append([]string{}, rev.reasons...),
		Received: rev.facts.State == SettlementPaid && rev.judged == JudgedConsistent, Digest: rev.digest, WrittenAt: rev.writtenAt}
}

func (r *Registry) RecordSettlement(workspace, id, settlementID string, number int, facts SettlementFacts) (SettlementView, error) {
	canonical, formed := r.settlementForm(facts)
	if !validID(workspace) || !validID(id) || !validID(settlementID) || number < 1 || number > 1000 || !formed {
		return SettlementView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return SettlementView{}, err
	}
	s := st.settlements[settlementID]
	count := 0
	if s != nil {
		count = len(s.revisions)
	}
	if number <= count {
		existing := s.revisions[number-1]
		if existing.facts != canonical {
			return SettlementView{}, ErrConflict
		}
		return viewSettlement(id, settlementID, number, existing), nil
	}
	if st.t.Role == RoleBuyer || canonical.Currency != st.t.Term.Currency || number > count+1 {
		return SettlementView{}, ErrRejected
	}
	if count > 0 && rank[canonical.State] < rank[s.revisions[count-1].facts.State] {
		return SettlementView{}, ErrRejected
	}
	if count >= r.limits.Revisions || s == nil && len(st.settlements) >= r.limits.Settlements {
		return SettlementView{}, ErrLimit
	}
	if s == nil {
		s = &settlement{}
		st.settlements[settlementID] = s
	}
	judged, reasons := r.judgeSettlement(canonical)
	if canonical.State == SettlementPaid && r.payoutReused(workspace, st, settlementID, canonical.Payout) {
		judged, reasons = JudgedHeld, append(reasons, ReasonPayoutReused)
	}
	rev := &settlementRevision{facts: canonical, judged: judged, reasons: reasons, writtenAt: now}
	rev.digest = settlementDigest(workspace, st.t, settlementID, number, canonical)
	s.revisions = append(s.revisions, rev)
	return viewSettlement(id, settlementID, number, rev), nil
}

func (r *Registry) Settlement(workspace, id, settlementID string) (SettlementView, error) {
	if !validID(workspace) || !validID(id) || !validID(settlementID) {
		return SettlementView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	st, err := r.find(workspace, id)
	if err != nil {
		return SettlementView{}, err
	}
	s := st.settlements[settlementID]
	if s == nil {
		return SettlementView{}, ErrNotFound
	}
	return viewSettlement(id, settlementID, len(s.revisions), s.revisions[len(s.revisions)-1]), nil
}
