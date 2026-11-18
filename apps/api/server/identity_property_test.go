package server

import (
	"bytes"
	"math/rand/v2"
	"testing"
	"time"
)

func (f world) anchorCount() int {
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	return len(f.s.anchors(f.s.operatorMembers(), f.s.clock()))
}

func TestPropertyAnchorsRecordsAndRestarts(t *testing.T) {
	if testing.Short() {
		t.Skip("property run")
	}
	for seed := uint64(1); seed <= 12; seed++ {
		r := rand.New(rand.NewPCG(299, seed))
		f := setup(t, SeedSample).ready()
		targets := []string{SampleAdmin2, SampleAdmin3}
		for step := 0; step < 40; step++ {
			f.sweep()
			root := f.as(AdminLogin)
			target := targets[r.IntN(2)]
			anchors := f.anchorCount()
			before, _ := readLedger(f)
			var got reply
			switch r.IntN(6) {
			case 0:
				got = f.period(root, target, f.at(time.Duration(1+r.IntN(90))*time.Minute))
			case 1:
				got = f.period(root, target, "")
			case 2:
				staged := f.stage(root, "", target)
				prepared := f.prepare(root, field(staged, "draft"), "suspend_membership")
				if prepared.status != 200 {
					f.run(root, "bulk_revocation.discard", field(staged, "draft"), nil)
					continue
				}
				got = f.execute(root, field(prepared, "bulk"), field(prepared, "digest"))
				f.s.drainCleanup()
			case 3:
				got = f.run(root, "identity.reinstate_membership", target, nil)
			case 4:
				f.advance(time.Duration(r.IntN(40)) * time.Minute)
				continue
			default:
				effective := map[string]bool{}
				for _, user := range targets {
					effective[user] = f.membership(user).Effective
				}
				f = f.restart(nil)
				for _, user := range targets {
					if f.membership(user).Effective && !effective[user] {
						t.Fatalf("seed %d step %d: %s is less restricted after a restart", seed, step, user)
					}
				}
				continue
			}
			after, _ := readLedger(f)
			if got.status >= 400 && !bytes.Equal(before, after) {
				t.Fatalf("seed %d step %d: a refusal wrote a record %s", seed, step, got.raw)
			}
			if anchors > 0 && f.anchorCount() == 0 {
				t.Fatalf("seed %d step %d: the anchors became empty %s", seed, step, got.raw)
			}
		}
	}
}
