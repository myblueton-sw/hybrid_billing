package balances

import (
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func (r *Registry) applicationForm(a Application) (Application, bool) {
	amount, okA := r.amount(a.Target.Amount, true)
	requested, okR := r.amount(a.Requested, true)
	if !okA || !okR || !validID(a.ID) || !validID(a.Account) || !validID(a.Target.Document) || !validID(a.Target.Line) {
		return Application{}, false
	}
	if a.Lifetime < 1 || a.Lifetime > maxLifetime || !validImport(a.Import) || a.Import != (Import{}) && a.Partial {
		return Application{}, false
	}
	a.Target.Amount, a.Requested = amount, requested
	return a, true
}

func stateOf(app *application, now time.Time) string {
	if app.state == StateReserved && !now.Before(app.expiry) {
		return StateExpired
	}
	return app.state
}

func (r *Registry) reversedOf(app *application) exact.Fraction {
	total := exact.Whole(0)
	for _, rev := range app.reversals {
		if rev.state == StateEffective {
			total = total.Plus(r.value(rev.r.Amount))
		}
	}
	return total
}

func (r *Registry) view(app *application, now time.Time) ApplicationView {
	applied, _ := format(app.applied)
	reversed, _ := format(r.reversedOf(app))
	return ApplicationView{Application: app.a, State: stateOf(app, now), Applied: applied, Reversed: reversed, Expiry: app.expiry,
		ReservedBy: app.reservedBy, ReservedAt: app.reservedAt, ConfirmedBy: app.confirmedBy, ConfirmedAt: app.confirmedAt,
		ReleasedBy: app.releasedBy, ReleasedAt: app.releasedAt, Digest: app.digest}
}

func (t *target) locked(now time.Time) bool {
	for _, app := range t.apps {
		if state := stateOf(app, now); state == StateConfirmed || state == StateReserved {
			return true
		}
	}
	return false
}

func (r *Registry) room(t *target, now time.Time) exact.Fraction {
	room := r.value(t.amount)
	for _, app := range t.apps {
		switch stateOf(app, now) {
		case StateConfirmed:
			room = room.Minus(app.applied).Plus(r.reversedOf(app))
		case StateReserved:
			room = room.Minus(app.applied)
		}
	}
	return room
}

func (r *Registry) refusal(acct *account, t *target, a Application, now time.Time) (string, exact.Fraction) {
	none := exact.Whole(0)
	if acct.a.Authority == AuthorityExternal {
		return ReasonExternalAuthority, none
	}
	if acct.pending() {
		return ReasonOpeningPending, none
	}
	room := r.value(a.Target.Amount)
	if t != nil && t.locked(now) {
		switch {
		case t.customer != acct.a.Customer || t.entity != acct.a.Entity:
			return ReasonCustomerMismatch, none
		case t.currency != acct.a.Currency:
			return ReasonCurrencyMismatch, none
		case t.amount != a.Target.Amount:
			return ReasonTargetMismatch, none
		}
		room = r.room(t, now)
	}
	requested := r.value(a.Requested)
	if room.Sign() <= 0 {
		return ReasonTargetSettled, none
	}
	if !a.Partial && room.Less(requested) {
		return ReasonExceedsTarget, none
	}
	available := r.available(acct, now)
	if available.Sign() <= 0 {
		return ReasonNothingAvailable, none
	}
	if !a.Partial && available.Less(requested) {
		return ReasonInsufficient, none
	}
	return "", least(requested, room, available)
}

func (r *Registry) allocate(acct *account, applied exact.Fraction, now time.Time) []*hold {
	lots := []*lot{}
	for _, l := range acct.lots {
		if !l.expired(now) {
			lots = append(lots, l)
		}
	}
	sort.SliceStable(lots, func(a, b int) bool {
		x, y := lots[a], lots[b]
		if x.expiry.IsZero() != y.expiry.IsZero() {
			return y.expiry.IsZero()
		}
		if !x.expiry.Equal(y.expiry) {
			return x.expiry.Before(y.expiry)
		}
		return x.number < y.number
	})
	holds := []*hold{}
	left := applied
	for _, l := range lots {
		if left.Sign() <= 0 {
			break
		}
		free := l.remainder.Minus(r.held(acct, l, now))
		if free.Sign() <= 0 {
			continue
		}
		take := least(free, left)
		holds = append(holds, &hold{lot: l, amount: take, returned: exact.Whole(0)})
		left = left.Minus(take)
	}
	return holds
}

func (r *Registry) Reserve(workspace string, a Application, actor string) (Decision, error) {
	canonical, formed := r.applicationForm(a)
	if !validID(workspace) || !validID(actor) || !formed {
		return Decision{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	acct := r.lookup(workspace, canonical.Account)
	if acct == nil {
		return Decision{}, ErrNotFound
	}
	s := r.workspaces[workspace]
	if existing := s.applications[canonical.ID]; existing != nil {
		if existing.a != canonical {
			return Decision{}, ErrConflict
		}
		return Decision{Application: r.view(existing, now)}, nil
	}
	if r.imported(s, canonical.Import, now) {
		return Decision{}, ErrConflict
	}
	if err := r.importRefused(s, acct, canonical.Import, now); err != nil {
		return Decision{}, err
	}
	acct.prune(now)
	key := [2]string{canonical.Target.Document, canonical.Target.Line}
	t := s.targets[key]
	reason, applied := r.refusal(acct, t, canonical, now)
	if reason != "" {
		return Decision{Reason: reason}, nil
	}
	if len(s.applications) >= r.limits.Applications || t == nil && len(s.targets) >= r.limits.Targets {
		return Decision{}, ErrLimit
	}
	if t == nil {
		t = &target{}
		s.targets[key] = t
	}
	if !t.locked(now) {
		t.customer, t.entity, t.currency, t.amount = acct.a.Customer, acct.a.Entity, acct.a.Currency, canonical.Target.Amount
	}
	app := &application{a: canonical, acct: acct, tgt: t, state: StateReserved, applied: applied, digest: applicationDigest(workspace, canonical),
		reservedBy: actor, reservedAt: now, expiry: now.Add(time.Duration(canonical.Lifetime) * time.Second)}
	app.holds = r.allocate(acct, applied, now)
	for _, h := range app.holds {
		if !h.lot.expiry.IsZero() && h.lot.expiry.Before(app.expiry) {
			app.expiry = h.lot.expiry
		}
	}
	t.apps = append(t.apps, app)
	acct.apps = append(acct.apps, app)
	acct.live = append(acct.live, app)
	s.applications[canonical.ID] = app
	if canonical.Import != (Import{}) {
		s.importedApps[importKey(canonical.Import)] = app
	}
	return Decision{Application: r.view(app, now)}, nil
}

func (r *Registry) find(workspace, id string) *application {
	s := r.workspaces[workspace]
	if s == nil {
		return nil
	}
	return s.applications[id]
}

func (r *Registry) Confirm(workspace, id, actor string) (ApplicationView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ApplicationView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	app := r.find(workspace, id)
	if app == nil {
		return ApplicationView{}, ErrNotFound
	}
	switch stateOf(app, now) {
	case StateConfirmed:
		return r.view(app, now), nil
	case StateReserved:
	default:
		return ApplicationView{}, ErrRejected
	}
	for _, h := range app.holds {
		h.lot.remainder = h.lot.remainder.Minus(h.amount)
	}
	app.state, app.confirmedBy, app.confirmedAt = StateConfirmed, actor, now
	return r.view(app, now), nil
}

func (r *Registry) Release(workspace, id, actor string) (ApplicationView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ApplicationView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	app := r.find(workspace, id)
	if app == nil {
		return ApplicationView{}, ErrNotFound
	}
	switch stateOf(app, now) {
	case StateReleased:
		return r.view(app, now), nil
	case StateReserved:
	default:
		return ApplicationView{}, ErrRejected
	}
	app.state, app.releasedBy, app.releasedAt = StateReleased, actor, now
	return r.view(app, now), nil
}

func (r *Registry) Application(workspace, id string) (ApplicationView, error) {
	if !validID(workspace) || !validID(id) {
		return ApplicationView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	app := r.find(workspace, id)
	if app == nil {
		return ApplicationView{}, ErrNotFound
	}
	return r.view(app, now), nil
}
