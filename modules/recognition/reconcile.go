package recognition

import (
	"math/big"
	"strconv"
)

func (r *Registry) quarterForm(q Quarter) bool {
	if !validID(q.ID) || len(q.Months) < 1 || len(q.Months) > maxMonths || !validCurrency(q.Currency) || !bases[q.Basis] || !taxBases[q.TaxBasis] ||
		!r.number(q.Billed.Net) || !r.number(q.Billed.Tax) {
		return false
	}
	for index, month := range q.Months {
		if !validMonth(month) || index > 0 && month <= q.Months[index-1] {
			return false
		}
	}
	return true
}

type tally struct {
	view      MonthView
	net, tax  *big.Int
	ready     bool
	final     bool
	evidence  []string
	pending   []string
	unposted  []string
	termNet   *big.Int
	termTax   *big.Int
	provision int
}

func (s *space) month(k monthKey, scale int) tally {
	t := tally{net: new(big.Int), tax: new(big.Int), termNet: new(big.Int), termTax: new(big.Int)}
	live := s.liveInitial(k)
	t.ready = live != nil && live.view.State == BatchApproved
	t.final = live != nil && !live.view.Batch.Provisional
	for _, b := range s.byMonth[k] {
		switch {
		case b.view.State == BatchOpen:
			t.view.Open++
		case b.view.State != BatchApproved:
		case b.posted():
			t.net.Add(t.net, b.net)
			t.tax.Add(t.tax, b.tax)
			t.evidence = append(t.evidence, b.view.Batch.ID, b.view.Digest, b.external())
			if b.view.Batch.Provisional && b.view.Batch.Kind != BatchReversal && !b.reversed() {
				t.provision++
			}
		case !b.carrying():
			t.view.Unposted++
		}
	}
	for _, a := range s.adjByMonth[k] {
		carrier, posted := "", false
		if a.carrier != nil {
			carrier, posted = a.carrier.view.Batch.ID, a.carrier.posted()
		}
		t.evidence = append(t.evidence, a.view.Adjustment.ID, a.view.State, carrier, flag(posted), a.view.Digest)
		switch {
		case a.view.State == AdjustmentPending:
			t.pending = append(t.pending, a.view.Adjustment.ID)
		case a.view.State == AdjustmentApproved && !posted:
			t.termNet.Add(t.termNet, a.net)
			t.termTax.Add(t.termTax, a.tax)
			t.unposted = append(t.unposted, a.view.Adjustment.ID)
		}
	}
	t.view.Month, t.view.Live, t.view.Provisional, t.view.Final = k.month, t.ready, t.provision, t.final
	t.view.Net, t.view.Tax = show(t.net, scale), show(t.tax, scale)
	return t
}

func (r *Registry) reconcile(workspace string, s *space, p *profile, q Quarter) (Reconciliation, error) {
	pr := p.view.Profile
	if q.Currency != pr.Currency || q.Basis != pr.Basis || q.TaxBasis != pr.TaxBasis {
		return Reconciliation{}, refuse(ReasonBasisMismatch)
	}
	billedNet, ok := r.units(q.Billed.Net, pr.Scale)
	billedTax, fine := r.units(q.Billed.Tax, pr.Scale)
	if !ok || !fine {
		return Reconciliation{}, ErrInvalid
	}
	q.Months = append([]string(nil), q.Months...)
	out := Reconciliation{Profile: pr.ID, Quarter: q}
	net, tax, termNet, termTax := new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	collecting, unposted, mixed := false, false, false
	texts := []string{reconcileLabel, workspace, pr.ID, pr.Entity, pr.Ledger, pr.Contract, pr.Basis, pr.TaxBasis, pr.Currency, strconv.Itoa(pr.Scale), flag(pr.Provisional), pr.Method,
		q.ID, joined(q.Months), q.Currency, q.Basis, q.TaxBasis, q.Billed.Net, q.Billed.Tax, flag(q.Final)}
	for _, month := range q.Months {
		t := s.month(monthKey{pr.ID, month}, pr.Scale)
		out.Months = append(out.Months, t.view)
		net.Add(net, t.net)
		tax.Add(tax, t.tax)
		termNet.Add(termNet, t.termNet)
		termTax.Add(termTax, t.termTax)
		out.Pending = append(out.Pending, t.pending...)
		out.Unposted = append(out.Unposted, t.unposted...)
		collecting = collecting || !t.ready || t.view.Open > 0
		unposted = unposted || t.view.Unposted > 0
		mixed = mixed || q.Final && t.provision > 0 || !q.Final && t.final
		texts = append(texts, month, t.view.Net, t.view.Tax, flag(t.ready), strconv.Itoa(t.view.Open), strconv.Itoa(t.view.Unposted), strconv.Itoa(t.provision), flag(t.final))
		texts = append(texts, t.evidence...)
	}
	baseNet, baseTax := new(big.Int).Add(net, termNet), new(big.Int).Add(tax, termTax)
	baseGross := new(big.Int).Add(baseNet, baseTax)
	diffNet, diffTax := new(big.Int).Sub(billedNet, baseNet), new(big.Int).Sub(billedTax, baseTax)
	diffGross := new(big.Int).Add(diffNet, diffTax)
	for _, value := range []*big.Int{net, tax, termNet, termTax, baseNet, baseTax, baseGross, diffNet, diffTax, diffGross} {
		if !r.bounded(value, pr.Scale) {
			return Reconciliation{}, ErrLimit
		}
	}
	out.Recognised = Amounts{Net: show(net, pr.Scale), Tax: show(tax, pr.Scale)}
	out.Term = Amounts{Net: show(termNet, pr.Scale), Tax: show(termTax, pr.Scale)}
	out.Base = Amounts{Net: show(baseNet, pr.Scale), Tax: show(baseTax, pr.Scale)}
	out.BaseGross = show(baseGross, pr.Scale)
	out.Difference = Amounts{Net: show(diffNet, pr.Scale), Tax: show(diffTax, pr.Scale)}
	out.Gross = show(diffGross, pr.Scale)
	switch {
	case collecting:
		out.State = StateCollecting
	case unposted:
		out.State = StatePostingUnconfirmed
	case mixed:
		out.State = StateMixedBasis
	case len(out.Pending) > 0:
		out.State = StateAdjustmentPending
	case diffNet.Sign() != 0 || diffTax.Sign() != 0:
		out.State = StateAmountDifference
	case len(out.Unposted) > 0:
		out.State = StateAdjustmentPosting
	default:
		out.State = StateMatched
	}
	texts = append(texts, out.Recognised.Net, out.Recognised.Tax, out.Term.Net, out.Term.Tax, out.Base.Net, out.Base.Tax, out.BaseGross,
		out.Difference.Net, out.Difference.Tax, out.Gross, joined(out.Pending), joined(out.Unposted), out.State)
	out.Digest = hashOf(texts...)
	return out, nil
}

func (r *Registry) Reconcile(workspace, profileID string, q Quarter) (Reconciliation, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(profileID) || !r.quarterForm(q) {
		return Reconciliation{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.profiles[profileID] == nil {
		return Reconciliation{}, ErrNotFound
	}
	return r.reconcile(workspace, s, s.profiles[profileID], q)
}

func copyClearance(c *ClearanceView) ClearanceView {
	v := *c
	v.Quarter.Months = append([]string(nil), v.Quarter.Months...)
	return v
}

func (r *Registry) Clear(workspace, profileID string, q Quarter, digest string, actor Actor) (ClearanceView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(profileID) || !r.quarterForm(q) || !validDigest(digest) || !actor.valid() {
		return ClearanceView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.profiles[profileID] == nil {
		return ClearanceView{}, ErrNotFound
	}
	p := s.profiles[profileID]
	if seen := s.clearances[profileID][q.ID]; seen != nil && seen.Digest == digest && actor.is(seen.Actor) && sameQuarter(seen.Quarter, q) {
		return copyClearance(seen), nil
	}
	if err := actor.refusal(); err != nil {
		return ClearanceView{}, err
	}
	if p.view.Profile.Method == MethodNone {
		return ClearanceView{}, refuse(ReasonNoMethod)
	}
	if s.clearances[profileID][q.ID] != nil {
		return ClearanceView{}, refuse(ReasonMonthAlreadyCleared)
	}
	for _, month := range q.Months {
		if s.cleared[monthKey{profileID, month}] != "" {
			return ClearanceView{}, refuse(ReasonMonthAlreadyCleared)
		}
	}
	result, err := r.reconcile(workspace, s, p, q)
	if err != nil {
		return ClearanceView{}, err
	}
	if result.State != StateMatched {
		return ClearanceView{}, refuse(ReasonNotMatched)
	}
	if result.Digest != digest {
		return ClearanceView{}, refuse(ReasonDigestChanged)
	}
	q.Months = append([]string(nil), q.Months...)
	made := &ClearanceView{Quarter: q, Method: p.view.Profile.Method, Digest: digest, Actor: actor.ID, At: now}
	if s.clearances[profileID] == nil {
		s.clearances[profileID] = map[string]*ClearanceView{}
	}
	s.clearances[profileID][q.ID] = made
	for _, month := range q.Months {
		s.cleared[monthKey{profileID, month}] = q.ID
	}
	return copyClearance(made), nil
}

func (r *Registry) ClearanceOf(workspace, profileID, quarter string) (ClearanceView, error) {
	r.enter()
	defer r.leave()
	s := r.workspace(workspace)
	if s == nil || s.clearances[profileID][quarter] == nil {
		return ClearanceView{}, ErrNotFound
	}
	return copyClearance(s.clearances[profileID][quarter]), nil
}

func sameQuarter(a, b Quarter) bool {
	return a.ID == b.ID && joined(a.Months) == joined(b.Months) && a.Currency == b.Currency && a.Basis == b.Basis &&
		a.TaxBasis == b.TaxBasis && a.Billed == b.Billed && a.Final == b.Final
}
