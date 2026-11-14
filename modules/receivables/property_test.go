package receivables

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type modelAlloc struct {
	id, effective, source string
	amount                *big.Rat
	state                 string
	reversal              string
	reversalState         string
}

type modelDoc struct {
	amount *big.Rat
	due    string
	allocs []*modelAlloc
}

func (d *modelDoc) outstanding() *big.Rat {
	out := new(big.Rat).Set(d.amount)
	for _, a := range d.allocs {
		if a.state == StateApproved && a.reversalState != StateApproved {
			out.Sub(out, a.amount)
		}
	}
	return out
}

func rat(text string) *big.Rat {
	v, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return v
}

func TestReceivablesAgainstAModel(t *testing.T) {
	for seed := int64(1); seed <= 300; seed++ {
		zone := []string{"Asia/Seoul", "America/New_York", "Europe/Berlin"}[seed%3]
		seoul, _ := time.LoadLocation(zone)
		rng := rand.New(rand.NewSource(seed))
		l := testLimits()
		l.Allocations, l.Extensions = 12, 6
		r, clock := openWith(t, l)
		clock.set(time.Date(2026, 10, 1+int(seed%20), 0, 0, 0, 0, seoul))
		setup(t, r, "k", zone, 10)
		docs := map[string]*modelDoc{}
		for _, id := range []string{"x", "y"} {
			amount := fmt.Sprintf("%d.%02d", 50+rng.Intn(100), rng.Intn(100))
			at := clock.read().Add(-time.Duration(rng.Intn(48)) * time.Hour)
			v := must(r.Issue("ws", invoice(id, "k", amount, at), "clerk"))
			docs[id] = &modelDoc{amount: rat(amount), due: addDays(at.In(seoul).Format(dateLayout), 10)}
			if v.Snapshot.EffectiveDue != docs[id].due {
				t.Fatalf("seed %d: due %s %s", seed, v.Snapshot.EffectiveDue, docs[id].due)
			}
		}
		var confirmed time.Time
		n := 0
		for step := 0; step < 60; step++ {
			trace := fmt.Sprintf("seed %d step %d", seed, step)
			id := []string{"x", "y"}[rng.Intn(2)]
			d := docs[id]
			today := clock.read().In(seoul).Format(dateLayout)
			switch rng.Intn(8) {
			case 0:
				clock.advance(time.Duration(rng.Intn(5*24)) * time.Hour)
			case 1:
				at := clock.read().Add(-time.Duration(rng.Intn(72)) * time.Hour)
				_, err := r.Confirm("ws", "en", "USD", at, "sync")
				switch {
				case confirmed.IsZero() || at.After(confirmed):
					if err != nil {
						t.Fatalf("%s confirm: %v", trace, err)
					}
					confirmed = at
				case at.Equal(confirmed):
					if err != nil {
						t.Fatalf("%s repeat confirm: %v", trace, err)
					}
				default:
					expect(t, "rejected", err, trace+" backwards confirm")
				}
			case 2, 3:
				n++
				aid := "a" + itoa(n)
				amount := fmt.Sprintf("%d.%d", rng.Intn(40), 1+rng.Intn(9))
				manual := rng.Intn(3) == 0
				a := Allocation{ID: aid, Document: id, Kind: "receipt", Amount: amount, Effective: today, Source: SourceAuthority, Ledger: "erp", Revision: "r"}
				if manual {
					a.Source, a.Ledger, a.Revision, a.Evidence = SourceManual, "", "", digest('a')
				}
				_, err := r.Allocate("ws", a, "sync")
				want := "ok"
				switch {
				case !manual && d.outstanding().Cmp(rat(amount)) < 0:
					want = ReasonExceedsOutstanding
				case len(d.allocs) >= l.Allocations:
					want = "limit"
				}
				if kind(err) != want {
					t.Fatalf("%s allocate: want %s got %s", trace, want, kind(err))
				}
				if err == nil {
					m := &modelAlloc{id: aid, effective: today, source: a.Source, amount: rat(amount), state: StateApproved}
					if manual {
						m.state = StatePending
					}
					d.allocs = append(d.allocs, m)
				}
			case 4:
				for _, m := range d.allocs {
					if m.state != StatePending {
						continue
					}
					_, err := r.ApproveAllocation("ws", m.id, "lead")
					want := "ok"
					if d.outstanding().Cmp(m.amount) < 0 {
						want = ReasonExceedsOutstanding
					}
					if kind(err) != want {
						t.Fatalf("%s approve: want %s got %s", trace, want, kind(err))
					}
					if err == nil {
						m.state = StateApproved
					} else if rng.Intn(2) == 0 {
						must(r.WithdrawAllocation("ws", m.id, "sync"))
						m.state = StateWithdrawn
					}
					break
				}
			case 5:
				for _, m := range d.allocs {
					if m.state != StateApproved || m.reversalState == StateApproved || m.reversalState == StatePending {
						continue
					}
					n++
					vid := "v" + itoa(n)
					v := Reversal{ID: vid, Allocation: m.id, Effective: today, Source: SourceAuthority, Ledger: "erp", Revision: "r"}
					if m.source == SourceManual {
						v.Source, v.Ledger, v.Revision, v.Evidence = SourceManual, "", "", digest('b')
					}
					if rng.Intn(6) == 0 {
						wrong := v
						wrong.ID = "w" + itoa(n)
						if m.source == SourceManual {
							wrong.Source, wrong.Ledger, wrong.Revision, wrong.Evidence = SourceAuthority, "erp", "r", ""
						} else {
							wrong.Source, wrong.Ledger, wrong.Revision, wrong.Evidence = SourceManual, "", "", digest('b')
						}
						expect(t, "rejected", second(r.Reverse("ws", wrong, "sync")), trace+" a reversal from another source")
					}
					_, err := r.Reverse("ws", v, "sync")
					if err != nil {
						if kind(err) == "limit" {
							break
						}
						t.Fatalf("%s reverse: %v", trace, err)
					}
					m.reversal, m.reversalState = vid, StateApproved
					if v.Source == SourceManual {
						m.reversalState = StatePending
					}
					break
				}
			case 6:
				for _, m := range d.allocs {
					if m.reversalState != StatePending {
						continue
					}
					if rng.Intn(2) == 0 {
						must(r.ApproveReversal("ws", m.reversal, "lead"))
						m.reversalState = StateApproved
					} else {
						must(r.WithdrawReversal("ws", m.reversal, "sync"))
						m.reversalState = StateWithdrawn
					}
					break
				}
			case 7:
				n++
				due := addDays(d.due, 1+rng.Intn(20))
				if _, err := r.Extend("ws", Extension{ID: "e" + itoa(n), Document: id, Due: due, Reason: "billing_error"}, "clerk"); err == nil {
					if _, err := r.ApproveExtension("ws", "e"+itoa(n), "lead"); err != nil {
						t.Fatalf("%s extension: %v", trace, err)
					}
					d.due = due
				}
			}
			now := clock.read()
			total := new(big.Rat)
			for _, docID := range []string{"x", "y"} {
				m := docs[docID]
				v := must(r.Status("ws", docID))
				out := m.outstanding()
				if rat(v.Outstanding).Cmp(out) != 0 || out.Sign() < 0 {
					t.Fatalf("%s %s: outstanding %s, model %s", trace, docID, v.Outstanding, out.FloatString(8))
				}
				total.Add(total, out)
				days := daysBetween(m.due, now.In(seoul).Format(dateLayout))
				stale := confirmed.IsZero() || now.Sub(confirmed) > l.Freshness
				want := StatusOverdue
				switch {
				case out.Sign() == 0:
					want = StatusSettled
				case days < 0:
					want = StatusNotDue
				case days == 0:
					want = StatusDueToday
				case stale:
					want = StatusUnknown
				}
				if v.Status != want || v.Days != days || v.Stale != stale || v.Document.Snapshot.EffectiveDue != m.due {
					t.Fatalf("%s %s: %s %d %v, model %s %d %v", trace, docID, v.Status, v.Days, v.Stale, want, days, stale)
				}
			}
			a := must(r.Aging("ws", Filter{}))
			sum := new(big.Rat)
			for _, c := range a.Currencies {
				for _, name := range BucketNames() {
					sum.Add(sum, rat(c.Buckets[name].Amount))
				}
				if rat(c.Total).Cmp(sum) != 0 {
					t.Fatalf("%s: total %s, buckets %s", trace, c.Total, sum.FloatString(8))
				}
			}
			if sum.Cmp(total) != 0 {
				t.Fatalf("%s: aging %s, model %s", trace, sum.FloatString(8), total.FloatString(8))
			}
		}
	}
}
