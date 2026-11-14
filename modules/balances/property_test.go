package balances

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type model struct {
	opening, increases, applied, appReversed, lotReversed map[string]*big.Rat
	apps                                                  map[string]Application
	targets                                               map[string][]string
	increaseIDs, reversalIDs, appIDs                      []string
	imports                                               map[string]string
	liveOpening                                           bool
}

func newModel() *model {
	m := &model{opening: map[string]*big.Rat{}, increases: map[string]*big.Rat{}, applied: map[string]*big.Rat{}, appReversed: map[string]*big.Rat{},
		lotReversed: map[string]*big.Rat{}, apps: map[string]Application{}, targets: map[string][]string{}, imports: map[string]string{}}
	return m
}

func add(sums map[string]*big.Rat, key string, value *big.Rat) {
	if sums[key] == nil {
		sums[key] = new(big.Rat)
	}
	sums[key].Add(sums[key], value)
}

func ratOr(sums map[string]*big.Rat, key string) *big.Rat {
	if sums[key] == nil {
		return new(big.Rat)
	}
	return sums[key]
}

func amountText(rng *rand.Rand) string {
	return fmt.Sprintf("%d.%02d", rng.Intn(60), rng.Intn(100))
}

func (m *model) room(t *testing.T, r *Registry, document string, amount *big.Rat) *big.Rat {
	room := new(big.Rat).Set(amount)
	active := false
	for _, id := range m.targets[document] {
		v, err := r.Application("ws", id)
		if err != nil {
			t.Fatal(err)
		}
		switch v.State {
		case StateConfirmed:
			room.Sub(room, rat(t, v.Applied))
			room.Add(room, rat(t, v.Reversed))
			active = true
		case StateReserved:
			room.Sub(room, rat(t, v.Applied))
			active = true
		}
	}
	if !active {
		return nil
	}
	return room
}

func TestPropertyLedger(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(1); seed <= 40; seed++ {
		runProperty(t, seed, seen)
	}
	for _, name := range []string{"imported", "import_conflict", "import_rejected", "reason_target_settled", "reason_nothing_available", "reason_insufficient",
		"reason_customer_mismatch", "reason_exceeds_target", "reason_target_mismatch", "reserved", "refused", "confirmed", "released", "application_reversal", "lot_reversal", "expired", "withdrawn_increase", "partial"} {
		if seen[name] < 5 {
			t.Errorf("the property test reached %s %d times: %v", name, seen[name], seen)
		}
	}
}

func runProperty(t *testing.T, seed int64, seen map[string]int) {
	rng := rand.New(rand.NewSource(seed))
	limits := Limits{Accounts: 10, Openings: 5, Increases: 200, Applications: 1000, Targets: 100, Reversals: 50, Workspace: 1000}
	r, clock := openWith(t, limits)
	accounts := []Account{cash("cash", "cust-1", "USD"), credit("credit", "cust-1", "USD"), cash("cash2", "cust-2", "USD")}
	for _, a := range accounts {
		opened(t, r, a)
	}
	m := newModel()
	for index, a := range accounts {
		if rng.Intn(2) == 0 {
			continue
		}
		o := Opening{ID: "o-" + a.ID, Account: a.ID, PriorLedger: "legacy", AsOf: "2026-10-31", Amount: amountText(rng), Evidence: evidence, MigrationKey: fmt.Sprintf("m%d", index), IncludedFrom: "2026-01-01"}
		if rng.Intn(3) == 0 {
			o.ExpiresOn = "2026-11-25"
		}
		if _, err := r.RecordOpening("ws", o, "clerk"); err != nil {
			t.Fatal(err)
		}
		switch rng.Intn(3) {
		case 0:
			r.WithdrawOpening("ws", o.ID, "clerk")
		default:
			if _, err := r.ApproveOpening("ws", o.ID, "lead"); err != nil {
				t.Fatal(err)
			}
			add(m.opening, a.ID, rat(t, o.Amount))
			m.liveOpening = true
		}
	}
	documents := map[string]string{"d1": "40", "d2": "25.5", "d3": "80", "d4": "10.01"}
	names := []string{"d1", "d2", "d3", "d4"}
	for step := 0; step < 250; step++ {
		a := accounts[rng.Intn(len(accounts))]
		switch op := rng.Intn(9); op {
		case 0, 1:
			id := fmt.Sprintf("i%d-%d", seed, step)
			inc := Increase{ID: id, Account: a.ID, Kind: IncreaseCashReceipt, Amount: amountText(rng), Reference: "ref-" + id}
			if a.Kind == KindCredit {
				inc.Kind = []string{IncreaseCreditDocument, IncreasePromotion, IncreaseSLACredit}[rng.Intn(3)]
			}
			if inc.Kind == IncreasePromotion || rng.Intn(3) == 0 {
				inc.ExpiresOn = clock.read().Add(time.Duration(rng.Intn(5)) * 24 * time.Hour).Format("2006-01-02")
			}
			if rng.Intn(4) == 0 {
				inc.Import = Import{Ledger: "legacy", Transaction: fmt.Sprintf("t%d", rng.Intn(20)), Occurred: []string{"2026-10-15", "2026-11-05"}[rng.Intn(2)]}
			}
			want := ""
			if inc.Import.Occurred != "" && m.imports[inc.Import.Transaction] != "" {
				want = "conflict"
			} else if inc.Import.Occurred != "" && m.liveOpening && inc.Import.Occurred <= "2026-10-31" {
				want = "rejected"
			}
			_, err := r.RecordIncrease("ws", inc, "clerk")
			if want != "" && kind(err) != want || want == "" && err != nil && kind(err) != "rejected" && kind(err) != "limit" {
				t.Fatalf("seed %d step %d: record %+v: %v, want %s", seed, step, inc, err, want)
			}
			if err == nil {
				m.increaseIDs = append(m.increaseIDs, id)
				if inc.Import.Occurred != "" {
					m.imports[inc.Import.Transaction] = id
					seen["imported"]++
				}
			} else if want != "" {
				seen["import_"+want]++
			}
		case 2:
			if len(m.increaseIDs) == 0 {
				continue
			}
			id := m.increaseIDs[rng.Intn(len(m.increaseIDs))]
			before, _ := r.Increase("ws", id)
			if rng.Intn(4) == 0 {
				if v, err := r.WithdrawIncrease("ws", id, "clerk"); err == nil {
					seen["withdrawn_increase"]++
					if m.imports[v.Increase.Import.Transaction] == id {
						delete(m.imports, v.Increase.Import.Transaction)
					}
				}
				continue
			}
			if v, err := r.ApproveIncrease("ws", id, "lead"); err == nil && before.State == StatePending {
				add(m.increases, v.Increase.Account, rat(t, v.Increase.Amount))
			}
		case 3, 4:
			document := names[rng.Intn(len(names))]
			amount := documents[document]
			if rng.Intn(8) == 0 {
				amount = "41"
			}
			id := fmt.Sprintf("a%d-%d", seed, step)
			app := Application{ID: id, Account: a.ID, Target: Target{Document: document, Line: "1", Amount: amount}, Requested: amountText(rng), Partial: rng.Intn(2) == 0,
				Lifetime: 600 + rng.Intn(7200)}
			if app.Requested == "0.00" {
				continue
			}
			st, _ := r.Statement("ws", a.ID)
			room := m.room(t, r, document, rat(t, amount))
			expected := m.reason(t, r, app, room, accounts, rat(t, st.Available))
			d, err := r.Reserve("ws", app, "runner")
			if err != nil {
				t.Fatalf("seed %d step %d: reserve: %v", seed, step, err)
			}
			if d.Reason != expected {
				t.Fatalf("seed %d step %d: reason %q, want %q for %+v", seed, step, d.Reason, expected, app)
			}
			if d.Reason != "" {
				seen["reason_"+d.Reason]++
				seen["refused"]++
				continue
			}
			seen["reserved"]++
			if rat(t, d.Application.Applied).Cmp(rat(t, app.Requested)) < 0 {
				seen["partial"]++
			}
			want := rat(t, app.Requested)
			if room == nil {
				room = rat(t, amount)
			}
			if room.Cmp(want) < 0 {
				want = room
			}
			if available := rat(t, st.Available); available.Cmp(want) < 0 {
				want = available
			}
			if rat(t, d.Application.Applied).Cmp(want) != 0 || !app.Partial && rat(t, d.Application.Applied).Cmp(rat(t, app.Requested)) != 0 {
				t.Fatalf("seed %d step %d: applied %s, want %s (%+v)", seed, step, d.Application.Applied, want.RatString(), st)
			}
			m.apps[id] = app
			m.appIDs = append(m.appIDs, id)
			m.targets[document] = append(m.targets[document], id)
		case 5:
			if len(m.appIDs) == 0 {
				continue
			}
			id := m.appIDs[rng.Intn(len(m.appIDs))]
			before, _ := r.Application("ws", id)
			if v, err := r.Confirm("ws", id, "runner"); err == nil && before.State == StateReserved {
				add(m.applied, m.apps[id].Account, rat(t, v.Applied))
				seen["confirmed"]++
			} else if err != nil && before.State == StateReserved {
				t.Fatalf("seed %d step %d: a reservation not confirmed: %v", seed, step, err)
			}
		case 6:
			if len(m.appIDs) > 0 {
				if v, err := r.Release("ws", m.appIDs[rng.Intn(len(m.appIDs))], "runner"); err == nil && v.State == StateReleased {
					seen["released"]++
				}
			}
		case 7:
			id := fmt.Sprintf("v%d-%d", seed, step)
			rev := Reversal{ID: id, Kind: ReverseApplication, Amount: amountText(rng)}
			if rng.Intn(2) == 0 && len(m.increaseIDs) > 0 {
				rev.Kind, rev.Record = ReverseIncrease, m.increaseIDs[rng.Intn(len(m.increaseIDs))]
			} else if len(m.appIDs) > 0 {
				rev.Record = m.appIDs[rng.Intn(len(m.appIDs))]
			}
			if rev.Record == "" || rev.Amount == "0.00" {
				continue
			}
			allowed := m.reversible(t, r, rev, clock.read())
			if _, err := r.Reverse("ws", rev, "clerk"); err == nil {
				m.reversalIDs = append(m.reversalIDs, id)
				if !allowed {
					t.Fatalf("seed %d step %d: an excessive reversal recorded: %+v", seed, step, rev)
				}
			} else if kind(err) == "rejected" && allowed {
				t.Fatalf("seed %d step %d: a reversal within its limit rejected: %+v", seed, step, rev)
			}
			if len(m.reversalIDs) > 0 {
				pick := m.reversalIDs[rng.Intn(len(m.reversalIDs))]
				before, _ := r.Reversal("ws", pick)
				if v, err := r.ApproveReversal("ws", pick, "lead"); err == nil && before.State == StatePending {
					owner := ""
					if v.Reversal.Kind == ReverseApplication {
						owner = m.apps[v.Reversal.Record].Account
						add(m.appReversed, owner, rat(t, v.Reversal.Amount))
						seen["application_reversal"]++
					} else {
						inc, _ := r.Increase("ws", v.Reversal.Record)
						add(m.lotReversed, inc.Increase.Account, rat(t, v.Reversal.Amount))
						seen["lot_reversal"]++
					}
				}
			}
		case 8:
			clock.advance(time.Duration(rng.Intn(36*60)) * time.Minute)
		}
		for _, acct := range accounts {
			st := identity(t, r, "ws", acct.ID)
			m.lotsAgree(t, r, acct.ID, clock.read(), seed, step)
			if st.Expired != "0" {
				seen["expired"]++
			}
			for name, pair := range map[string][2]*big.Rat{
				"opening":     {rat(t, st.Opening), ratOr(m.opening, acct.ID)},
				"increases":   {rat(t, st.Increases), ratOr(m.increases, acct.ID)},
				"applied":     {rat(t, st.Applications), ratOr(m.applied, acct.ID)},
				"appReversed": {rat(t, st.ApplicationReversals), ratOr(m.appReversed, acct.ID)},
				"lotReversed": {rat(t, st.LotReversals), ratOr(m.lotReversed, acct.ID)},
			} {
				if pair[0].Cmp(pair[1]) != 0 {
					t.Fatalf("seed %d step %d %s: %s is %s, the model has %s", seed, step, acct.ID, name, pair[0].RatString(), pair[1].RatString())
				}
			}
		}
		for document, ids := range m.targets {
			used := new(big.Rat)
			fixed := ""
			for _, id := range ids {
				v, _ := r.Application("ws", id)
				if rat(t, v.Reversed).Cmp(rat(t, v.Applied)) > 0 || rat(t, v.Applied).Cmp(rat(t, v.Application.Requested)) > 0 {
					t.Fatalf("seed %d step %d: %+v", seed, step, v)
				}
				if v.State == StateReserved || v.State == StateConfirmed {
					used.Add(used, rat(t, v.Applied))
					used.Sub(used, rat(t, v.Reversed))
					if fixed != "" && fixed != v.Application.Target.Amount && v.State == StateConfirmed {
						t.Fatalf("seed %d step %d: two amounts for %s", seed, step, document)
					}
					fixed = v.Application.Target.Amount
				}
			}
			if fixed != "" && used.Cmp(rat(t, fixed)) > 0 {
				t.Fatalf("seed %d step %d: %s over its amount: %s", seed, step, document, used.RatString())
			}
		}
	}
}

func (m *model) reason(t *testing.T, r *Registry, app Application, room *big.Rat, accounts []Account, available *big.Rat) string {
	customers := map[string]string{}
	for _, a := range accounts {
		customers[a.ID] = a.Customer
	}
	requested := rat(t, app.Requested)
	if room != nil {
		for _, id := range m.targets[app.Target.Document] {
			v, _ := r.Application("ws", id)
			if v.State != StateReserved && v.State != StateConfirmed {
				continue
			}
			if customers[m.apps[id].Account] != customers[app.Account] {
				return ReasonCustomerMismatch
			}
			if v.Application.Target.Amount != app.Target.Amount {
				return ReasonTargetMismatch
			}
		}
	} else {
		room = rat(t, app.Target.Amount)
	}
	switch {
	case room.Sign() <= 0:
		return ReasonTargetSettled
	case !app.Partial && room.Cmp(requested) < 0:
		return ReasonExceedsTarget
	case available.Sign() <= 0:
		return ReasonNothingAvailable
	case !app.Partial && available.Cmp(requested) < 0:
		return ReasonInsufficient
	}
	return ""
}

func (m *model) reversible(t *testing.T, r *Registry, rev Reversal, now time.Time) bool {
	amount := rat(t, rev.Amount)
	if rev.Kind == ReverseApplication {
		v, _ := r.Application("ws", rev.Record)
		return v.State == StateConfirmed && new(big.Rat).Sub(rat(t, v.Applied), rat(t, v.Reversed)).Cmp(amount) >= 0
	}
	inc, _ := r.Increase("ws", rev.Record)
	if inc.State != StateApproved {
		return false
	}
	lots, _ := r.Lots("ws", inc.Increase.Account)
	for _, l := range lots {
		if l.Record == rev.Record {
			return rat(t, l.Available).Cmp(amount) >= 0 && (inc.Increase.ExpiresOn == "" || now.Before(dayAfter(t, inc.Increase.ExpiresOn)))
		}
	}
	return false
}

func dayAfter(t *testing.T, date string) time.Time {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.AddDate(0, 0, 1)
}

func (m *model) lotsAgree(t *testing.T, r *Registry, account string, now time.Time, seed int64, step int) {
	lots, _ := r.Lots("ws", account)
	held := new(big.Rat)
	for _, l := range lots {
		expires := ""
		if l.Source == ReverseIncrease {
			v, _ := r.Increase("ws", l.Record)
			expires = v.Increase.ExpiresOn
		} else {
			v, _ := r.Opening("ws", l.Record)
			expires = v.Opening.ExpiresOn
		}
		over := expires != "" && !now.Before(dayAfter(t, expires))
		if over && (l.Remainder != "0" || l.Available != "0" || l.Held != "0") || !over && l.Expired != "0" {
			t.Fatalf("seed %d step %d: lot %+v at %v", seed, step, l, now)
		}
		held.Add(held, rat(t, l.Held))
	}
	want := new(big.Rat)
	for _, id := range m.appIDs {
		if m.apps[id].Account != account {
			continue
		}
		if v, _ := r.Application("ws", id); v.State == StateReserved {
			want.Add(want, rat(t, v.Applied))
		}
	}
	if held.Cmp(want) != 0 {
		t.Fatalf("seed %d step %d: %s holds %s, the reservations %s", seed, step, account, held.RatString(), want.RatString())
	}
}
