package accesschange

import (
	"errors"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"
)

func TestPropertyRequestInvariants(t *testing.T) {
	r := rand.New(rand.NewPCG(298, 3))
	people := []string{"a", "b", "c", "d"}
	for round := 0; round < 60; round++ {
		w := setup(t)
		applies := map[uint64]int{}
		seen := map[uint64]string{}
		for step := 0; step < 80; step++ {
			who := Actor{User: people[r.IntN(4)], Human: r.IntN(10) > 0}
			switch r.IntN(6) {
			case 0:
				p := proposal(add(people[r.IntN(4)]), who)
				p.Allowed = r.IntN(8) > 0
				w.r.Propose(p)
			case 1:
				p := proposal(revoke("g"+strconv.Itoa(r.IntN(3))), who)
				p.Holder = people[r.IntN(4)]
				w.r.Propose(p)
			case 2:
				n := uint64(1 + r.IntN(8))
				v, ok := w.r.Request(n, "w1")
				basis := "basis-1"
				if r.IntN(5) == 0 {
					basis = "basis-2"
				}
				_, err := w.r.Approve(n, "w1", who, v.Digest, Standing{Basis: basis, Allowed: r.IntN(6) > 0}, func(Applier) (string, error) {
					if r.IntN(6) == 0 {
						return "", errors.New("apply failed")
					}
					applies[n]++
					return "ref", nil
				})
				if err == nil && (!ok || who.User == v.Proposer || (v.Change.Kind == ChangeAdd && who.User == v.Change.Subject) || !who.Human) {
					t.Fatalf("approved by %s %+v", who.User, v)
				}
			case 3:
				n := uint64(1 + r.IntN(8))
				v, _ := w.r.Request(n, "w1")
				if _, err := w.r.Reject(n, "w1", who, r.IntN(6) > 0, "x"); err == nil && v.Change.Kind == ChangeRevoke && who.User == v.Holder {
					t.Fatalf("holder rejected %+v", v)
				}
			case 4:
				w.r.Withdraw(uint64(1+r.IntN(8)), "w1", who)
			default:
				w.advance(time.Duration(r.IntN(48)) * time.Hour)
			}
			pending := map[string]int{}
			for _, v := range w.r.Requests("w1") {
				if prior, ok := seen[v.Number]; ok && prior != StatePending && prior != v.State {
					t.Fatalf("request %d moved from %s to %s", v.Number, prior, v.State)
				}
				seen[v.Number] = v.State
				if (v.State == StateApplied) != (applies[v.Number] == 1) || applies[v.Number] > 1 {
					t.Fatalf("request %d state %s applies %d", v.Number, v.State, applies[v.Number])
				}
				if v.State == StatePending {
					pending[w.r.requests[v.Number].key]++
					if pending[w.r.requests[v.Number].key] > 1 {
						t.Fatalf("two pending requests for one target")
					}
				}
			}
		}
	}
}

func TestPropertyNoConfirmationWithoutAnAdvancingReceipt(t *testing.T) {
	r := rand.New(rand.NewPCG(298, 4))
	for round := 0; round < 40; round++ {
		w := setup(t)
		ledger := &fakeLedger{}
		var last uint64
		for step := 0; step < 30; step++ {
			ledger.err, ledger.repeat = nil, r.IntN(4) == 0
			if r.IntN(4) == 0 {
				ledger.err = reasoned{"ledger_stopped"}
			}
			before := ledger.seq
			e := &effects{ledger: ledger}
			v, err := w.r.Revoke(target("g"+strconv.Itoa(step)), e.build())
			if err != nil {
				t.Fatal(err)
			}
			advancing := ledger.err == nil && ledger.seq > before && ledger.seq > last
			if (v.State == RevocationComplete) != advancing {
				t.Fatalf("state %s advancing %v", v.State, advancing)
			}
			if advancing {
				last = ledger.seq
			}
		}
	}
}
