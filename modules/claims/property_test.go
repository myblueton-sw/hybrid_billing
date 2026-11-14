package claims

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type refReversal struct {
	id        string
	quantity  *big.Rat
	amount    *big.Rat
	effective bool
	author    string
}

type refApp struct {
	a         Application
	quantity  *big.Rat
	amount    *big.Rat
	state     string
	expiry    time.Time
	reversals []*refReversal
	confirmed bool
}

type refSource struct {
	quantity, amount *big.Rat
	currency         string
	apps             []*refApp
}

type model struct {
	now     time.Time
	apps    map[string]*refApp
	order   []*refApp
	sources map[string]*refSource
	revs    map[string]*refReversal
	revApp  map[string]*refApp
}

func ratOf(text string) *big.Rat {
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return value
}

func (m *model) reversed(app *refApp) (*big.Rat, *big.Rat) {
	q, a := new(big.Rat), new(big.Rat)
	for _, rev := range app.reversals {
		if rev.effective {
			q.Add(q, rev.quantity)
			a.Add(a, rev.amount)
		}
	}
	return q, a
}

func (m *model) stateOf(app *refApp) string {
	if app.state == StateReserved && !m.now.Before(app.expiry) {
		return StateExpired
	}
	if app.state == StateConfirmed && m.effectiveCount(app) > 0 {
		q, a := m.reversed(app)
		if q.Cmp(app.quantity) == 0 && a.Cmp(app.amount) == 0 {
			return StateReversed
		}
	}
	return app.state
}

func (m *model) effectiveCount(app *refApp) int {
	count := 0
	for _, rev := range app.reversals {
		if rev.effective {
			count++
		}
	}
	return count
}

func (m *model) active(app *refApp) bool {
	s := m.stateOf(app)
	return s == StateReserved || s == StateConfirmed
}

func eligibilityOf(a Application) string {
	if a.Start < "2026-01-01" || a.End > "2027-01-01" {
		return ReasonNoEligibility
	}
	switch a.Scope.Component {
	case "fee", "usage", "ext", "unk":
	default:
		return ReasonNotEligible
	}
	declared := map[string][]string{"fee": {"fee-1", "fee-2"}, "usage": {"main", "alt"}, "ext": {"e"}, "unk": {"u"}}
	found := false
	for _, instance := range declared[a.Scope.Component] {
		found = found || instance == a.Scope.Instance
	}
	switch {
	case !found:
		return ReasonUnknownInstance
	case a.Scope.Component == "ext":
		return ReasonRoutedExternally
	case a.Scope.Component == "unk":
		return ReasonRouteUnknown
	}
	return ""
}

func (m *model) capacityOpen(src *refSource, skip *refApp) bool {
	for _, other := range src.apps {
		if other != skip && (other.confirmed || m.active(other)) {
			return false
		}
	}
	return true
}

func (m *model) reason(a Application, skip *refApp) string {
	if reason := eligibilityOf(a); reason != "" {
		return reason
	}
	scope := []*refApp{}
	for _, other := range m.order {
		if other.a.Scope == a.Scope && other != skip && m.active(other) {
			scope = append(scope, other)
		}
	}
	for _, other := range scope {
		if other.a.Mode != a.Mode {
			return ReasonModeMismatch
		}
	}
	for _, other := range scope {
		if other.a.Currency != a.Currency {
			return ReasonCurrencyMismatch
		}
	}
	if a.Mode == ModeInterval {
		for _, other := range scope {
			if other.a.Start < a.End && a.Start < other.a.End {
				return ReasonAlreadyClaimed
			}
		}
		return ""
	}
	q, am := ratOf(a.Quantity), ratOf(a.Amount)
	src := m.sources[a.Source]
	if src == nil || m.capacityOpen(src, skip) {
		if ratOf(a.CapQuantity).Cmp(q) < 0 || ratOf(a.CapAmount).Cmp(am) < 0 {
			return ReasonCapacityExceeded
		}
		return ""
	}
	if src.quantity.Cmp(ratOf(a.CapQuantity)) != 0 || src.amount.Cmp(ratOf(a.CapAmount)) != 0 || src.currency != a.Currency {
		return ReasonCapacityMismatch
	}
	for _, other := range src.apps {
		if other != skip && m.active(other) && other.a.Scope != a.Scope {
			return ReasonSourceInOtherScope
		}
	}
	for _, other := range src.apps {
		if other != skip && m.active(other) {
			rq, ra := m.reversed(other)
			q.Add(q, new(big.Rat).Sub(other.quantity, rq))
			am.Add(am, new(big.Rat).Sub(other.amount, ra))
		}
	}
	if src.quantity.Cmp(q) < 0 || src.amount.Cmp(am) < 0 {
		return ReasonCapacityExceeded
	}
	return ""
}

func randomApplication(rng *rand.Rand, index int, ids []string) Application {
	months := []string{"2025-12-01", "2026-01-01", "2026-02-01", "2026-03-01", "2026-04-01", "2026-05-01", "2026-12-01", "2027-01-01", "2027-02-01"}
	first := rng.Intn(len(months) - 1)
	last := first + 1 + rng.Intn(2)
	if last >= len(months) {
		last = len(months) - 1
	}
	components := []string{"fee", "fee", "usage", "usage", "ext", "unk", "none"}
	component := components[rng.Intn(len(components))]
	instances := map[string][]string{"fee": {"fee-1", "fee-2", "fee-3"}, "usage": {"main", "alt"}, "ext": {"e"}, "unk": {"u"}, "none": {"n"}}
	options := instances[component]
	a := Application{ID: fmt.Sprintf("a%d", index), Scope: Scope{"c1", component, options[rng.Intn(len(options))]}, Start: months[first], End: months[last],
		Quantity: fmt.Sprint(rng.Intn(9)), Amount: fmt.Sprint(rng.Intn(9)), Currency: []string{"USD", "USD", "USD", "EUR"}[rng.Intn(4)],
		Calculation: "calc", Lifetime: 1 + rng.Intn(120)}
	if rng.Intn(2) == 0 {
		a.Mode = ModeInterval
	} else {
		a.Mode = ModeSource
		a.Source = []string{"s1", "s2", "s3"}[rng.Intn(3)]
		a.CapQuantity, a.CapAmount = []string{"10", "10", "6"}[rng.Intn(3)], []string{"10", "10", "7"}[rng.Intn(3)]
	}
	if len(ids) > 0 && rng.Intn(5) == 0 {
		a.Replaces = ids[rng.Intn(len(ids))]
	}
	return a
}

func TestRandomSequencesAgainstAReference(t *testing.T) {
	rng := rand.New(rand.NewSource(214))
	reached := map[string]int{}
	for round := 0; round < 400; round++ {
		clock := &testClock{now: start}
		limits := Limits{Contracts: 4, Revisions: 3, Components: 6, Applications: 1000, Sources: 1000, Reversals: 1000, Batch: 10}
		r, _ := NewRegistry(testProfile(t), limits, clock.read)
		rev := Revision{From: "2026-01-01", To: "2027-01-01", Path: PathResale, Components: []Component{
			comp("fee", KindMSPService, true, RoutePlatform, "fee-1", "fee-2"), comp("usage", KindCSPUsage, true, RoutePlatform, "main", "alt"),
			comp("ext", KindOther, true, RouteExternal, "e"), comp("unk", KindOther, true, RouteUnknown, "u"), comp("off", KindOther, false, RoutePlatform, "o")}}
		eligible(t, r, "c1", 1, rev)
		m := &model{now: start, apps: map[string]*refApp{}, sources: map[string]*refSource{}, revs: map[string]*refReversal{}, revApp: map[string]*refApp{}}
		ids := []string{}
		for step := 0; step < 120; step++ {
			switch op := rng.Intn(10); {
			case op < 5:
				a := randomApplication(rng, len(ids), ids)
				d, err := r.Reserve("ws", a, "runner")
				var skip *refApp
				wantErr := "ok"
				if a.Replaces != "" {
					skip = m.apps[a.Replaces]
					if skip.a.Scope != a.Scope || m.stateOf(skip) != StateReserved {
						wantErr = "rejected"
					}
				}
				if kind(err) != wantErr {
					t.Fatalf("round %d step %d: %v want %s", round, step, err, wantErr)
				}
				if err != nil {
					reached["error"]++
					continue
				}
				want := m.reason(a, skip)
				if d.Reason != want {
					t.Fatalf("round %d step %d %+v: %q want %q", round, step, a, d.Reason, want)
				}
				reached[want]++
				if want != "" {
					continue
				}
				app := &refApp{a: d.Application.Application, quantity: ratOf(a.Quantity), amount: ratOf(a.Amount), state: StateReserved,
					expiry: m.now.Add(time.Duration(a.Lifetime) * time.Second)}
				m.apps[a.ID], m.order = app, append(m.order, app)
				ids = append(ids, a.ID)
				if a.Mode == ModeSource {
					src := m.sources[a.Source]
					if src == nil {
						src = &refSource{}
						m.sources[a.Source] = src
					}
					if m.capacityOpen(src, skip) {
						src.quantity, src.amount, src.currency = ratOf(a.CapQuantity), ratOf(a.CapAmount), a.Currency
					}
					src.apps = append(src.apps, app)
				}
				if skip != nil {
					skip.state = StateReplaced
				}
			case op < 7 && len(ids) > 0:
				batch := []string{}
				for _, index := range rng.Perm(len(ids))[:1+rng.Intn(min(3, len(ids)))] {
					batch = append(batch, ids[index])
				}
				_, _, err := r.Confirm("ws", batch, "runner")
				want := "ok"
				for _, id := range batch {
					switch m.stateOf(m.apps[id]) {
					case StateReserved, StateConfirmed, StateReversed:
					default:
						want = "rejected"
					}
				}
				if kind(err) != want {
					t.Fatalf("round %d step %d confirm %v: %v want %s", round, step, batch, err, want)
				}
				if err == nil {
					reached["confirm"]++
					for _, id := range batch {
						if m.apps[id].state == StateReserved {
							m.apps[id].state, m.apps[id].confirmed = StateConfirmed, true
						}
					}
				}
			case op < 8 && len(ids) > 0:
				id := ids[rng.Intn(len(ids))]
				_, err := r.Release("ws", id, "runner")
				state := m.stateOf(m.apps[id])
				want := "rejected"
				if state == StateReserved || state == StateReleased {
					want = "ok"
				}
				if kind(err) != want {
					t.Fatalf("round %d step %d release %s: %v want %s", round, step, id, err, want)
				}
				if err == nil {
					m.apps[id].state = StateReleased
				}
			case op < 9 && len(ids) > 0:
				app := m.apps[ids[rng.Intn(len(ids))]]
				q, a := app.a.Quantity, app.a.Amount
				if app.a.Mode == ModeSource && rng.Intn(2) == 0 {
					q, a = fmt.Sprint(rng.Intn(4)), fmt.Sprint(rng.Intn(4))
				}
				id := fmt.Sprintf("r%d", len(m.revs))
				_, err := r.Reverse("ws", Reversal{ID: id, Application: app.a.ID, Quantity: q, Amount: a}, "clerk")
				fits := func() bool {
					if app.a.Mode == ModeInterval {
						return ratOf(q).Cmp(app.quantity) == 0 && ratOf(a).Cmp(app.amount) == 0
					}
					rq, ra := m.reversed(app)
					rq.Add(rq, ratOf(q))
					ra.Add(ra, ratOf(a))
					return rq.Cmp(app.quantity) <= 0 && ra.Cmp(app.amount) <= 0
				}
				want := "rejected"
				if m.stateOf(app) == StateConfirmed && fits() {
					want = "ok"
				}
				if kind(err) != want {
					t.Fatalf("round %d step %d reverse: %v want %s", round, step, err, want)
				}
				if err != nil {
					continue
				}
				rev := &refReversal{id: id, quantity: ratOf(q), amount: ratOf(a), author: "clerk"}
				m.revs[id], m.revApp[id] = rev, app
				app.reversals = append(app.reversals, rev)
				if rng.Intn(3) > 0 {
					_, err := r.ApproveReversal("ws", id, "lead")
					want := "rejected"
					if m.stateOf(app) == StateConfirmed && fits() {
						want = "ok"
					}
					if kind(err) != want {
						t.Fatalf("round %d step %d approve: %v want %s", round, step, err, want)
					}
					if err == nil {
						rev.effective = true
						reached["reversal"]++
					}
				}
			default:
				d := time.Duration(rng.Intn(90)) * time.Second
				clock.advance(d)
				m.now = m.now.Add(d)
			}
			for _, id := range ids {
				v, _ := r.Application("ws", id)
				if v.State != m.stateOf(m.apps[id]) {
					t.Fatalf("round %d step %d %s: %s want %s", round, step, id, v.State, m.stateOf(m.apps[id]))
				}
			}
			for i, x := range m.order {
				for _, y := range m.order[i+1:] {
					if x.a.Scope == y.a.Scope && x.a.Mode == ModeInterval && y.a.Mode == ModeInterval && m.active(x) && m.active(y) && x.a.Start < y.a.End && y.a.Start < x.a.End {
						t.Fatalf("overlap %s %s", x.a.ID, y.a.ID)
					}
				}
			}
			for name, src := range m.sources {
				used := new(big.Rat)
				for _, app := range src.apps {
					if m.active(app) {
						rq, _ := m.reversed(app)
						used.Add(used, new(big.Rat).Sub(app.quantity, rq))
					}
				}
				if src.quantity != nil && used.Cmp(src.quantity) > 0 {
					t.Fatalf("source %s over capacity", name)
				}
			}
			for _, app := range m.order {
				if !app.confirmed {
					continue
				}
				v, _ := r.Application("ws", app.a.ID)
				effective, q, a := 0, new(big.Rat), new(big.Rat)
				list, _ := r.Reversals("ws", app.a.ID)
				for _, rev := range list {
					if rev.State == ReversalEffective {
						effective++
						q.Add(q, ratOf(rev.Reversal.Quantity))
						a.Add(a, ratOf(rev.Reversal.Amount))
					}
				}
				inactive := v.State != StateConfirmed
				if inactive != (effective > 0 && q.Cmp(ratOf(app.a.Quantity)) == 0 && a.Cmp(ratOf(app.a.Amount)) == 0) {
					t.Fatalf("round %d step %d: %s is %s with %d effective reversals", round, step, app.a.ID, v.State, effective)
				}
			}
		}
	}
	for _, key := range []string{"", ReasonNoEligibility, ReasonNotEligible, ReasonUnknownInstance, ReasonRoutedExternally, ReasonRouteUnknown, ReasonModeMismatch,
		ReasonCurrencyMismatch, ReasonAlreadyClaimed, ReasonCapacityMismatch, ReasonSourceInOtherScope, ReasonCapacityExceeded, "error", "confirm", "reversal"} {
		if reached[key] < 5 {
			t.Errorf("%q reached %d times", key, reached[key])
		}
	}
}
