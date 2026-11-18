package server

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
)

func (f world) lastDurableEpoch() uint64 {
	last := uint64(0)
	for _, rec := range f.s.ledger.Verified() {
		if t := rec.Decision.Trust; t != nil && rec.Decision.Workspace == SampleWorkspace {
			last = t.NewEpoch
		}
	}
	return last
}

func TestServerTrustProperties(t *testing.T) {
	for seed := uint64(1); seed <= 6; seed++ {
		rng := rand.New(rand.NewPCG(seed, 7))
		f := setup(t, SeedSample).ready()
		f.trustAdmins()
		pool := []actor{}
		published := f.s.currentEpoch(SampleWorkspace)
		for step := 0; step < 10; step++ {
			switch rng.IntN(6) {
			case 0, 1:
				p := admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
				pool = append(pool, p.a)
				current, _ := f.s.policy.Policy(SampleWorkspace)
				ages := []string{}
				for _, age := range []string{"30", "60", "90", "120"} {
					if age != fmt.Sprint(int(current.Revision.MaxAge.Minutes())) {
						ages = append(ages, age)
					}
				}
				id := f.approvedChange(p, ages[rng.IntN(len(ages))], "0")
				fault := ""
				if rng.IntN(3) == 0 {
					fault = []string{"ledger-sync", "trust-publish"}[rng.IntN(2)]
				}
				f.s.fault = func(at string) bool { return at == fault }
				got := f.trustActivate(p.c, id)
				f.s.fault = nil
				switch got.status {
				case 200:
					published = f.s.currentEpoch(SampleWorkspace)
				case 202:
					if f.s.currentEpoch(SampleWorkspace) != published {
						t.Fatalf("seed %d: an unconfirmed activation changed the epoch", seed)
					}
					f = f.restart(nil)
					f.trustAdmins()
					pool = nil
					published = f.s.currentEpoch(SampleWorkspace)
				default:
					t.Fatalf("seed %d: activation %s", seed, got.raw)
				}
			case 2:
				f = f.restart(nil)
				f.trustAdmins()
				pool = nil
				if f.s.currentEpoch(SampleWorkspace) != published {
					t.Fatalf("seed %d: a restart moved the epoch from %d to %d", seed, published, f.s.currentEpoch(SampleWorkspace))
				}
			case 3:
				pool = append(pool, f.signedIn(AdminLogin, pw))
			default:
				for _, a := range pool {
					found, live := f.s.sessions.get(a.verifier(f))
					got := f.policyRead(a)
					if live && found.trustEpoch < f.s.currentEpoch(SampleWorkspace) && got.status != 401 {
						t.Fatalf("seed %d: an old-epoch session was admitted", seed)
					}
					if !live && got.status != 401 {
						t.Fatalf("seed %d: an ended session was admitted", seed)
					}
				}
			}
			if st := f.s.ledger.Status(); st.State != decisionledger.StateIsolated && f.s.currentEpoch(SampleWorkspace) < f.lastDurableEpoch() {
				t.Fatalf("seed %d: the published epoch %d is below the durable epoch %d", seed, f.s.currentEpoch(SampleWorkspace), f.lastDurableEpoch())
			}
		}
	}
}
