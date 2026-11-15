package retention

import (
	"math/rand"
	"testing"
	"time"
)

type mArt struct {
	anchorDays int
	stores     []string
	results    map[string]string
	deleted    bool
	damaged    bool
	held       bool
	requiredBy int
	outcome    string
	owned      bool
}

func TestProperty(t *testing.T) {
	for seed := int64(1); seed <= 120; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := setup(t)
		f.policy(ClassSource, 10, 0)
		period := 10
		arts := map[string]*mArt{}
		names := []string{}
		for i := 0; i < 8; i++ {
			id := "a" + itoa(i)
			stores := []string{"db"}
			if rng.Intn(2) == 0 {
				stores = []string{"db", "obj"}
			}
			age := rng.Intn(20)
			f.put(id, ClassSource, age, one, stores...)
			arts[id] = &mArt{anchorDays: age, stores: stores, results: map[string]string{}}
			names = append(names, id)
		}
		f.put("ev", ClassFinal, 0, one)
		elapsed := 0
		due := func(a *mArt) bool { return a.anchorDays+elapsed >= period }
		exclusion := func(a *mArt) string {
			switch {
			case a.held:
				return OutcomeHeld
			case a.requiredBy > 0:
				return OutcomeReferenced
			case !due(a):
				return OutcomeNotDue
			}
			return ""
		}
		holds, tombs := 0, 0
		var runID string
		var manifest []string
		chunk := 0
		next := 0
		var token uint64
		claimedChunk := -1
		for step := 0; step < 50; step++ {
			switch op := rng.Intn(9); {
			case op == 0:
				id := names[rng.Intn(len(names))]
				holds++
				f.must(second(f.r.PlaceHold(Hold{ID: "h" + itoa(holds), Owner: "o", Reason: "r", Artifacts: []string{id}}, cat)))
				arts[id].held = true
			case op == 1:
				id := names[rng.Intn(len(names))]
				a := arts[id]
				err := f.r.Refer(Reference{From: "ev", To: id, Kind: Requires}, ingest)
				busy := a.deleted || a.damaged || a.outcome == OutcomeDeleting || a.outcome == OutcomeFailed
				if busy {
					if err == nil && a.requiredBy > 0 {
						continue
					}
					reason(t, "busy ref", err, ReasonDeleting)
					continue
				}
				f.must(err)
				a.requiredBy = 1
			case op == 2:
				d := rng.Intn(3)
				f.advance(time.Duration(d) * day)
				elapsed += d
			case op == 3 && runID == "":
				manifest = nil
				for _, id := range names {
					if a := arts[id]; !a.deleted && !a.owned && rng.Intn(2) == 0 {
						manifest = append(manifest, id)
					}
				}
				if len(manifest) == 0 {
					continue
				}
				chunk = 1 + rng.Intn(3)
				runID = f.plan(ClassSource, chunk, manifest...)
				next, claimedChunk = 0, -1
				for _, id := range manifest {
					a := arts[id]
					if out := exclusion(a); out != "" {
						a.outcome, a.owned = out, false
						continue
					}
					a.outcome, a.owned = OutcomePending, true
					for s, v := range a.results {
						if v == StoreFailed {
							delete(a.results, s)
						}
					}
				}
			case op >= 4 && runID != "":
				span := manifest[min(next*chunk, len(manifest)):min((next+1)*chunk, len(manifest))]
				if claimedChunk != next {
					if next*chunk >= len(manifest) {
						runID = ""
						continue
					}
					c, err := f.r.Claim(runID, next, worker)
					f.must(err)
					token = c.Token
					claimedChunk = next
					want := 0
					for _, id := range span {
						a := arts[id]
						if a.outcome != OutcomePending {
							continue
						}
						if out := exclusion(a); out != "" {
							a.outcome, a.owned = out, false
							continue
						}
						a.outcome = OutcomeDeleting
						want++
					}
					if len(c.Deleting) != want {
						t.Fatalf("seed %d claim %v want %d", seed, c.Deleting, want)
					}
				} else {
					store := []string{"db", "obj"}[rng.Intn(2)]
					var del, fail []string
					for _, id := range span {
						a := arts[id]
						if a.outcome != OutcomeDeleting || !contains(a.stores, store) || a.results[store] != "" {
							continue
						}
						if rng.Intn(4) == 0 {
							fail = append(fail, id)
						} else {
							del = append(del, id)
						}
					}
					if len(del)+len(fail) == 0 {
						allDone := true
						for _, id := range span {
							if arts[id].outcome == OutcomeDeleting {
								allDone = false
							}
						}
						if allDone {
							next++
						}
						continue
					}
					f.must(f.r.Report(runID, next, token, store, del, fail, worker))
					for _, id := range del {
						arts[id].results[store], arts[id].damaged = StoreDeleted, true
					}
					for _, id := range fail {
						arts[id].results[store] = StoreFailed
					}
					for _, id := range append(del, fail...) {
						a := arts[id]
						all, reported := true, true
						for _, s := range a.stores {
							all = all && a.results[s] == StoreDeleted
							reported = reported && a.results[s] != ""
						}
						switch {
						case all:
							a.deleted, a.outcome, a.owned = true, OutcomeDeleted, false
							tombs++
						case reported:
							a.outcome = OutcomeFailed
						}
					}
				}
			}
			for _, id := range names {
				a := arts[id]
				want := AccessOK
				switch {
				case a.deleted:
					want = AccessDeleted
				case a.damaged || a.owned:
					want = AccessBlocked
				}
				if got := f.access(id); got != want {
					t.Fatalf("seed %d step %d %s access %s want %s (%+v)", seed, step, id, got, want, a)
				}
			}
			if f.r.Watermark() != uint64(tombs) {
				t.Fatalf("seed %d tombstones %d want %d", seed, f.r.Watermark(), tombs)
			}
			if runID != "" {
				v := f.run(runID)
				for _, id := range manifest {
					if v.Outcomes[id] != arts[id].outcome {
						t.Fatalf("seed %d step %d %s outcome %s want %s", seed, step, id, v.Outcomes[id], arts[id].outcome)
					}
				}
			}
		}
	}
}
