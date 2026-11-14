package claims

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func (r *Registry) applicationForm(a Application) (Application, bool) {
	out := a
	if !validID(a.ID) || !validID(a.Scope.Contract) || !validID(a.Scope.Component) || !validID(a.Scope.Instance) {
		return Application{}, false
	}
	if !validDate(a.Start) || !validDate(a.End) || a.End <= a.Start || !modes[a.Mode] || !validCurrency(a.Currency) || !validID(a.Calculation) {
		return Application{}, false
	}
	if a.Lifetime < 1 || a.Lifetime > maxLifetime || !optionalID(a.Replaces) || a.Replaces == a.ID || !optionalID(a.Corrects) {
		return Application{}, false
	}
	if (a.CorrectsKind == "") != (a.Corrects == "") || a.CorrectsKind != "" && a.CorrectsKind != CorrectsApplication && a.CorrectsKind != CorrectsReversal {
		return Application{}, false
	}
	if a.CorrectsKind == CorrectsApplication && a.Corrects == a.ID {
		return Application{}, false
	}
	quantity, okQ := r.amount(a.Quantity)
	amount, okA := r.amount(a.Amount)
	if !okQ || !okA {
		return Application{}, false
	}
	out.Quantity, out.Amount = quantity, amount
	if a.Mode == ModeInterval {
		return out, a.Source == "" && a.CapQuantity == "" && a.CapAmount == ""
	}
	capQuantity, okCQ := r.amount(a.CapQuantity)
	capAmount, okCA := r.amount(a.CapAmount)
	if !validID(a.Source) || !okCQ || !okCA {
		return Application{}, false
	}
	out.CapQuantity, out.CapAmount = capQuantity, capAmount
	return out, true
}

func sameRequest(a, b Application) bool { return a == b }

func (r *Registry) reversed(app *application) (exact.Fraction, exact.Fraction) {
	quantity, amount := exact.Whole(0), exact.Whole(0)
	for _, rev := range app.reversals {
		if rev.state == ReversalEffective {
			quantity, amount = quantity.Plus(r.value(rev.r.Quantity)), amount.Plus(r.value(rev.r.Amount))
		}
	}
	return quantity, amount
}

func (r *Registry) whollyReversed(app *application) bool {
	effective := false
	for _, rev := range app.reversals {
		effective = effective || rev.state == ReversalEffective
	}
	quantity, amount := r.reversed(app)
	if !effective {
		return false
	}
	return same(quantity, r.value(app.a.Quantity)) && same(amount, r.value(app.a.Amount))
}

func (r *Registry) stateOf(app *application, now time.Time) string {
	switch {
	case app.state == StateReserved && !now.Before(app.expiry):
		return StateExpired
	case app.state == StateConfirmed && r.whollyReversed(app):
		return StateReversed
	}
	return app.state
}

func (r *Registry) active(app *application, now time.Time) bool {
	state := r.stateOf(app, now)
	return state == StateReserved || state == StateConfirmed
}

func (r *Registry) remaining(app *application) (exact.Fraction, exact.Fraction) {
	quantity, amount := r.reversed(app)
	return r.value(app.a.Quantity).Minus(quantity), r.value(app.a.Amount).Minus(amount)
}

func (r *Registry) scopeReason(s *space, a Application, skip *application, now time.Time) string {
	for _, other := range s.scopes[a.Scope] {
		if other == skip || !r.active(other, now) {
			continue
		}
		if other.a.Mode != a.Mode {
			return ReasonModeMismatch
		}
	}
	for _, other := range s.scopes[a.Scope] {
		if other != skip && r.active(other, now) && other.a.Currency != a.Currency {
			return ReasonCurrencyMismatch
		}
	}
	if a.Mode == ModeInterval {
		for _, other := range s.scopes[a.Scope] {
			if other != skip && r.active(other, now) && other.a.Start < a.End && a.Start < other.a.End {
				return ReasonAlreadyClaimed
			}
		}
		return ""
	}
	src := s.sources[a.Source]
	if src == nil || r.capacityOpen(src, skip, now) {
		if r.value(a.CapQuantity).Less(r.value(a.Quantity)) || r.value(a.CapAmount).Less(r.value(a.Amount)) {
			return ReasonCapacityExceeded
		}
		return ""
	}
	if src.quantity != a.CapQuantity || src.amount != a.CapAmount || src.currency != a.Currency {
		return ReasonCapacityMismatch
	}
	usedQ, usedA := r.value(a.Quantity), r.value(a.Amount)
	for _, other := range src.apps {
		if other == skip || !r.active(other, now) {
			continue
		}
		if other.a.Scope != a.Scope {
			return ReasonSourceInOtherScope
		}
	}
	for _, other := range src.apps {
		if other != skip && r.active(other, now) {
			q, m := r.remaining(other)
			usedQ, usedA = usedQ.Plus(q), usedA.Plus(m)
		}
	}
	if r.value(src.quantity).Less(usedQ) || r.value(src.amount).Less(usedA) {
		return ReasonCapacityExceeded
	}
	return ""
}

func (r *Registry) capacityOpen(src *source, skip *application, now time.Time) bool {
	for _, other := range src.apps {
		if other != skip && (other.state == StateConfirmed || r.active(other, now)) {
			return false
		}
	}
	return true
}

func (r *Registry) view(app *application, now time.Time) ApplicationView {
	quantity, amount := r.reversed(app)
	q, _ := format(quantity)
	m, _ := format(amount)
	return ApplicationView{Application: app.a, State: r.stateOf(app, now), Expiry: app.expiry, ReversedQuantity: q, ReversedAmount: m,
		ReservedBy: app.reservedBy, ReservedAt: app.reservedAt, ConfirmedBy: app.confirmedBy, ConfirmedAt: app.confirmedAt,
		ReleasedBy: app.releasedBy, ReleasedAt: app.releasedAt, ReplacedBy: app.replacedBy, ReplacedAt: app.replacedAt, Digest: app.digest}
}

func (r *Registry) Reserve(workspace string, a Application, actor string) (Decision, error) {
	canonical, formed := r.applicationForm(a)
	if !validID(workspace) || !validID(actor) || !formed {
		return Decision{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		s = emptySpace()
	}
	var target *application
	if canonical.Replaces != "" {
		if target = s.applications[canonical.Replaces]; target == nil {
			return Decision{}, ErrNotFound
		}
	}
	if canonical.CorrectsKind == CorrectsApplication && s.applications[canonical.Corrects] == nil ||
		canonical.CorrectsKind == CorrectsReversal && s.reversals[canonical.Corrects] == nil {
		return Decision{}, ErrNotFound
	}
	if existing := s.applications[canonical.ID]; existing != nil {
		if !sameRequest(existing.a, canonical) {
			return Decision{}, ErrConflict
		}
		return Decision{Application: r.view(existing, now)}, nil
	}
	if target != nil && (target.a.Scope != canonical.Scope || r.stateOf(target, now) != StateReserved) {
		return Decision{}, ErrRejected
	}
	switch canonical.CorrectsKind {
	case CorrectsApplication:
		corrected := s.applications[canonical.Corrects]
		state := r.stateOf(corrected, now)
		if corrected.a.Scope != canonical.Scope || state != StateConfirmed && state != StateReversed {
			return Decision{}, ErrRejected
		}
	case CorrectsReversal:
		rev := s.reversals[canonical.Corrects]
		if rev.state != ReversalEffective || s.applications[rev.r.Application].a.Scope != canonical.Scope {
			return Decision{}, ErrRejected
		}
	}
	if reason := eligibilityReason(s.contracts[canonical.Scope.Contract], canonical); reason != "" {
		return Decision{Reason: reason}, nil
	}
	if reason := r.scopeReason(s, canonical, target, now); reason != "" {
		return Decision{Reason: reason}, nil
	}
	newSource := canonical.Mode == ModeSource && s.sources[canonical.Source] == nil
	if len(s.applications) >= r.limits.Applications || newSource && len(s.sources) >= r.limits.Sources {
		return Decision{}, ErrLimit
	}
	s = r.space(workspace)
	app := &application{a: canonical, state: StateReserved, expiry: now.Add(time.Duration(canonical.Lifetime) * time.Second), reservedBy: actor, reservedAt: now,
		digest: applicationDigest(workspace, canonical)}
	s.applications[canonical.ID] = app
	s.scopes[canonical.Scope] = append(s.scopes[canonical.Scope], app)
	if canonical.Mode == ModeSource {
		if newSource {
			s.sources[canonical.Source] = &source{}
		}
		if src := s.sources[canonical.Source]; r.capacityOpen(src, target, now) {
			src.quantity, src.amount, src.currency = canonical.CapQuantity, canonical.CapAmount, canonical.Currency
		}
		s.sources[canonical.Source].apps = append(s.sources[canonical.Source].apps, app)
	}
	if target != nil {
		target.state, target.replacedBy, target.replacedAt = StateReplaced, canonical.ID, now
	}
	return Decision{Application: r.view(app, now)}, nil
}
