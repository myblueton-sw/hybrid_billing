package closing

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type modelCounts struct {
	targets, done, waived, held, open int
}

func TestProgressMatchesAModelAtEverySequence(t *testing.T) {
	names := []string{StageCollect, StageValidate, StageIssue, StageSettle}
	kinds := map[string]Evidence{StageCollect: evidence(EvidenceFile), StageValidate: evidence(EvidenceInternal), StageIssue: evidence(EvidenceERP), StageSettle: evidence(EvidenceInternal)}
	stats := map[string]int{}
	for seed := int64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := open(t, Limits{Workspaces: 1, Calendars: 2, Cycles: 2, Targets: 100, History: 5000, Claims: 10, Lead: 24 * time.Hour})
		f.cycle(t, "oct")
		state := map[[2]string]string{}
		pending := map[[2]string]bool{}
		var targets []string
		snapshots := map[int]map[string]modelCounts{}
		record := func() {
			seq := must(f.r.CycleOf("ws", "oct")).Sequence
			snap := map[string]modelCounts{}
			for _, st := range names {
				var c modelCounts
				for _, target := range targets {
					c.targets++
					switch state[[2]string{target, st}] {
					case ItemDone:
						c.done++
					case ItemWaived:
						c.waived++
					case ItemHeld:
						c.held++
					default:
						c.open++
					}
				}
				snap[st] = c
			}
			snapshots[seq] = snap
		}
		record()
		next := 0
		for step := 0; step < 150; step++ {
			if len(targets) == 0 || rng.Intn(8) == 0 {
				next++
				target := "t" + strconv.Itoa(next)
				must(f.r.AddTargets("ws", "oct", Addition{Targets: []string{target}, Reason: "r"}, human("planner")))
				targets = append(targets, target)
				for _, st := range names {
					state[[2]string{target, st}] = ItemOpen
				}
				record()
				continue
			}
			target := targets[rng.Intn(len(targets))]
			index := rng.Intn(len(names))
			st := names[index]
			key := [2]string{target, st}
			var err error
			switch rng.Intn(7) {
			case 0, 1:
				actor := system("feed")
				if st == StageValidate || st == StageSettle {
					actor = human("ana")
				}
				_, err = f.r.Complete("ws", "oct", target, st, kinds[st], actor)
				ready := true
				for _, earlier := range names[:index] {
					if s := state[[2]string{target, earlier}]; s != ItemDone && s != ItemWaived {
						ready = false
					}
				}
				want := "ok"
				switch {
				case state[key] == ItemDone:
					want = "ok"
				case pending[key]:
					want = RefusalWaiverPending
				case state[key] != ItemOpen:
					want = RefusalState
				case !ready:
					want = RefusalStageOrder
				}
				if reason(err) != want {
					t.Fatalf("seed %d: complete %v model %s", seed, err, want)
				}
				if want == "ok" && state[key] != ItemDone {
					state[key] = ItemDone
					stats["done"]++
					record()
				}
			case 2:
				_, err = f.r.Reopen("ws", "oct", target, st, "r", human("ops"))
				if (state[key] == ItemDone) != (err == nil) {
					t.Fatalf("seed %d: reopen %v", seed, err)
				}
				if err == nil {
					for _, later := range names[index:] {
						if state[[2]string{target, later}] == ItemDone {
							state[[2]string{target, later}] = ItemOpen
						}
					}
					stats["reopened"]++
					record()
				}
			case 3:
				_, err = f.r.Hold("ws", "oct", target, st, "r", human("ops"))
				if (state[key] == ItemOpen && !pending[key]) != (err == nil) {
					t.Fatalf("seed %d: hold %v", seed, err)
				}
				if err == nil {
					state[key] = ItemHeld
					record()
				}
			case 4:
				_, err = f.r.Release("ws", "oct", target, st, human("ops"))
				if (state[key] == ItemHeld && !pending[key]) != (err == nil) {
					t.Fatalf("seed %d: release %v", seed, err)
				}
				if err == nil {
					state[key] = ItemOpen
					record()
				}
			case 5:
				_, err = f.r.RequestWaiver("ws", "oct", target, st, Waiver{Reason: "r", Evidence: "e"}, human("ops"))
				ok := !pending[key] && (state[key] == ItemOpen || state[key] == ItemHeld)
				if pending[key] {
					ok = true
				}
				if ok != (err == nil) {
					t.Fatalf("seed %d: waiver %v", seed, err)
				}
				if err == nil && !pending[key] {
					pending[key] = true
					record()
				}
			case 6:
				_, err = f.r.ApproveWaiver("ws", "oct", target, st, human("lead"))
				if (pending[key] || state[key] == ItemWaived) != (err == nil) {
					t.Fatalf("seed %d: approve %v", seed, err)
				}
				if err == nil && pending[key] {
					pending[key] = false
					state[key] = ItemWaived
					stats["waived"]++
					record()
				}
			}
		}
		for seq, snap := range snapshots {
			for _, st := range names {
				p := must(f.r.Progress("ws", "oct", st, AsOf{Sequence: seq}))
				w := snap[st]
				if p.Targets != w.targets || p.Done != w.done || p.Waived != w.waived || p.Held != w.held || p.Open != w.open || p.Denominator != w.targets-w.waived {
					t.Fatalf("seed %d seq %d %s: %+v model %+v", seed, seq, st, p, w)
				}
			}
		}
	}
	if stats["done"] < 100 || stats["reopened"] < 20 || stats["waived"] < 20 {
		t.Fatalf("weak exploration: %v", stats)
	}
}
