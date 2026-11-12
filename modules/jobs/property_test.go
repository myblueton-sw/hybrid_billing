package jobs

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

type ledgerTask struct {
	holding    bool
	until      time.Time
	repeatable bool
	grants     []uint64
	cleared    bool
	succeeded  int
	completed  bool
}

type ledgerJob struct {
	workspace, id, kind string
	budget              int
	tasks               map[string]*ledgerTask
	names               []string
	granted, cancelled  bool
	leftQueued          bool
	terminal            *JobView
}

func rule(view JobView, granted, cancelled bool) JobState {
	pending, leased, unknown, succeeded := 0, 0, 0, 0
	for _, shown := range view.Tasks {
		switch shown.State {
		case Pending:
			pending++
		case Leased:
			leased++
		case Unknown:
			unknown++
		case TaskSucceeded:
			succeeded++
		}
	}
	if cancelled {
		if leased+unknown == 0 {
			return Cancelled
		}
		return Waiting
	}
	if succeeded == len(view.Tasks) {
		return Succeeded
	}
	if pending+leased+unknown == 0 {
		return Failed
	}
	if pending+leased == 0 {
		return Waiting
	}
	if !granted {
		return Queued
	}
	return Running
}

func TestRandomHistoriesKeepTheLedger(t *testing.T) {
	random := rand.New(rand.NewSource(180))
	limits := Limits{Jobs: 3, Tasks: 5, Attempts: 3, PerType: 4, PerWorkspace: 2, Batch: 3}
	kinds := []string{"close", "send"}
	reached := map[string]int{}
	for round := 0; round < 300; round++ {
		b := open(t, limits)
		jobs := []*ledgerJob{}
		held := []Lease{}
		highest := uint64(0)
		verify := func(step int) {
			leasedKind, leasedPair := map[string]int{}, map[string]int{}
			for _, known := range jobs {
				view := b.job(known.workspace, known.id)
				if want := rule(view, known.granted, known.cancelled); view.State != want {
					t.Fatalf("round %d step %d: %s/%s is %s; the rule says %s: %s", round, step, known.workspace, known.id, view.State, want, b.shape(known.workspace, known.id))
				}
				if (view.Cancel != "") != known.cancelled {
					t.Fatalf("round %d step %d: the cancellation flag", round, step)
				}
				if view.State != Queued {
					known.leftQueued = true
				} else if known.leftQueued {
					t.Fatalf("round %d step %d: %s/%s is queued again", round, step, known.workspace, known.id)
				}
				log, _ := b.r.Transitions(known.workspace, known.id)
				if len(log) > 2+len(view.Tasks)*(4*known.budget+1) || uint64(len(log)) != view.Revision {
					t.Fatalf("round %d step %d: %d transitions at revision %d", round, step, len(log), view.Revision)
				}
				for _, shown := range view.Tasks {
					entry := known.tasks[shown.ID]
					last := uint64(0)
					if len(entry.grants) > 0 {
						last = entry.grants[len(entry.grants)-1]
					}
					if shown.Attempts != len(entry.grants) || shown.Attempts > known.budget || shown.Token != last {
						t.Fatalf("round %d step %d: task %s has %d attempts and token %d for the grants %v", round, step, shown.ID, shown.Attempts, shown.Token, entry.grants)
					}
					if entry.completed && shown.State != TaskSucceeded {
						t.Fatalf("round %d step %d: task %s is %s after its completion was accepted", round, step, shown.ID, shown.State)
					}
					if entry.succeeded > 1 || (shown.State == TaskSucceeded) != (entry.succeeded == 1) {
						t.Fatalf("round %d step %d: task %s is %s after %d successes", round, step, shown.ID, shown.State, entry.succeeded)
					}
					if shown.State == Leased {
						leasedKind[view.Type]++
						leasedPair[view.WorkspaceID+"/"+view.Type]++
					}
					if final := shown.State == TaskSucceeded || shown.State == TaskFailed || shown.State == TaskCancelled; final && shown.Open {
						t.Fatalf("round %d step %d: the final task %s has an open attempt", round, step, shown.ID)
					}
					if view.State.Terminal() && (shown.State == Pending || shown.State == Leased || shown.State == Unknown || shown.Open) {
						t.Fatalf("round %d step %d: a terminal job holds %s", round, step, b.shape(known.workspace, known.id))
					}
				}
				if view.State.Terminal() {
					view.AsOf, view.Overdue = time.Time{}, 0
					if known.terminal == nil {
						known.terminal = &view
						reached["terminal "+string(view.State)]++
					} else if !reflect.DeepEqual(*known.terminal, view) {
						t.Fatalf("round %d step %d: the terminal job %s/%s changed", round, step, known.workspace, known.id)
					}
				}
			}
			for kind, count := range leasedKind {
				if count > limits.PerType {
					t.Fatalf("round %d step %d: %d leases of type %s", round, step, count, kind)
				}
			}
			for slot, count := range leasedPair {
				if count > limits.PerWorkspace {
					t.Fatalf("round %d step %d: %d leases of %s", round, step, count, slot)
				}
			}
		}
		find := func(ref Ref) (*ledgerJob, *ledgerTask) {
			for _, known := range jobs {
				if known.workspace == ref.WorkspaceID && known.id == ref.JobID {
					return known, known.tasks[ref.TaskID]
				}
			}
			return nil, nil
		}
		for step := 0; step < 120; step++ {
			switch choice := random.Intn(20); {
			case choice < 2 && len(jobs) < 6:
				known := &ledgerJob{workspace: fmt.Sprintf("w%d", 1+random.Intn(2)), id: fmt.Sprintf("j%d", len(jobs)), kind: kinds[random.Intn(2)], budget: 1 + random.Intn(3), tasks: map[string]*ledgerTask{}}
				tasks := []TaskSpec{}
				for index := 1 + random.Intn(4); index > 0; index-- {
					spec := TaskSpec{ID: fmt.Sprintf("t%d", index), Input: "in", Repeatable: random.Intn(2) == 0}
					tasks = append(tasks, spec)
					known.tasks[spec.ID] = &ledgerTask{repeatable: spec.Repeatable}
					known.names = append(known.names, spec.ID)
				}
				policy := Policy{Lease: 10 * time.Minute, Life: 15 * time.Minute, Attempts: known.budget, BackoffBase: time.Minute, BackoffCap: 4 * time.Minute}
				if created, err := b.r.Create(known.workspace, known.id, known.kind, policy, tasks); err == nil && created {
					jobs = append(jobs, known)
				} else if !errors.Is(err, ErrLimit) {
					t.Fatalf("round %d: create %v", round, err)
				}
			case choice < 8:
				before := map[string]bool{}
				for _, known := range jobs {
					for name, entry := range known.tasks {
						before[known.workspace+"/"+known.id+"/"+name] = entry.holding && b.now.Before(entry.until)
					}
				}
				asked := kinds[:1+random.Intn(2)]
				leases, err := b.r.Lease("worker", asked, 1+random.Intn(3))
				if err != nil {
					t.Fatalf("round %d: lease %v", round, err)
				}
				for _, lease := range leases {
					known, entry := find(lease.Ref())
					name := lease.WorkspaceID + "/" + lease.JobID + "/" + lease.TaskID
					if before[name] {
						t.Fatalf("round %d step %d: %s was leased while its lease held", round, step, name)
					}
					before[name] = true
					if lease.Token <= highest {
						t.Fatalf("round %d step %d: token %d after %d", round, step, lease.Token, highest)
					}
					highest = lease.Token
					if len(entry.grants) > 0 && !entry.repeatable && !entry.cleared {
						t.Fatalf("round %d step %d: %s is not repeatable and was granted again without a confirmation that nothing happened", round, step, name)
					}
					if len(entry.grants) > 0 {
						reached["granted again"]++
					}
					if known.cancelled || len(entry.grants) >= known.budget || lease.Attempt != len(entry.grants)+1 {
						t.Fatalf("round %d step %d: a grant after a cancellation or beyond the budget: %+v", round, step, lease)
					}
					entry.grants, entry.cleared, known.granted = append(entry.grants, lease.Token), false, true
					entry.holding, entry.until = true, lease.Expiry
					held = append(held, lease)
				}
				reached["leases"] += len(leases)
			case choice < 14 && len(held) > 0:
				lease := held[random.Intn(len(held))]
				_, entry := find(lease.Ref())
				latest := entry.grants[len(entry.grants)-1] == lease.Token
				var report Report
				var err error
				kind := random.Intn(5)
				switch kind {
				case 0, 1:
					report, err = b.r.Complete(lease.Ref(), "done")
					if err == nil && !report.Replay {
						entry.completed = true
						entry.succeeded++
					}
				case 2:
					permanent := random.Intn(3) == 0
					report, err = b.r.Fail(lease.Ref(), "nothing-sent", permanent)
					if err == nil && !report.Replay && report.State == Pending {
						entry.cleared = true
					}
				case 3:
					report, err = b.r.Unknown(lease.Ref(), "lost")
				default:
					var until time.Time
					if until, err = b.r.Extend(lease.Ref()); err == nil {
						entry.until = until
					}
				}
				if err == nil && kind != 4 && !report.Replay {
					entry.holding = false
				}
				if err == nil {
					reached[fmt.Sprintf("worker write %d accepted", kind)]++
					if !latest {
						t.Fatalf("round %d step %d: a write with the old token %d was accepted", round, step, lease.Token)
					}
					if report.Replay {
						reached["replays"]++
					}
				} else if !errors.Is(err, ErrStale) {
					t.Fatalf("round %d step %d: a worker write ended with %v", round, step, err)
				} else {
					reached["stale"]++
					if !latest {
						reached["stale by an old token"]++
					}
				}
			case choice < 16:
				b.pass(time.Duration(1+random.Intn(9)) * time.Minute)
				if random.Intn(2) == 0 {
					b.r.Sweep()
				}
			case choice < 18 && len(jobs) > 0:
				known := jobs[random.Intn(len(jobs))]
				name := known.names[random.Intn(len(known.names))]
				entry := known.tasks[name]
				if len(entry.grants) == 0 {
					continue
				}
				token := entry.grants[random.Intn(len(entry.grants))]
				resolution := []Resolution{ResolveSucceeded, ResolveFailed, ResolveRetry}[random.Intn(3)]
				report, err := b.r.Resolve(known.workspace, known.id, name, token, resolution)
				if err != nil && !errors.Is(err, ErrStale) && !errors.Is(err, ErrRejected) && !errors.Is(err, ErrConflict) {
					t.Fatalf("round %d step %d: a resolution ended with %v", round, step, err)
				}
				if err == nil && !report.Replay {
					reached["resolved "+string(resolution)]++
					if token != entry.grants[len(entry.grants)-1] {
						t.Fatalf("round %d step %d: a resolution with an old token was accepted", round, step)
					}
					if resolution == ResolveSucceeded {
						entry.succeeded++
					}
					if resolution == ResolveRetry {
						entry.cleared = true
						if known.cancelled {
							t.Fatalf("round %d step %d: a retry after a cancellation", round, step)
						}
					}
				}
			case len(jobs) > 0 && choice == 19:
				known := jobs[random.Intn(len(jobs))]
				if err := b.r.Cancel(known.workspace, known.id, Requested); err == nil {
					if !known.cancelled {
						reached["cancelled"]++
					}
					known.cancelled = true
					for _, entry := range known.tasks {
						entry.holding = false
					}
				} else if !errors.Is(err, ErrRejected) {
					t.Fatalf("round %d step %d: cancel %v", round, step, err)
				}
			}
			verify(step)
		}
	}
	for _, name := range []string{"leases", "granted again", "worker write 0 accepted", "worker write 2 accepted", "worker write 3 accepted", "worker write 4 accepted", "replays", "stale", "stale by an old token",
		"resolved succeeded", "resolved failed", "resolved retry", "cancelled", "terminal succeeded", "terminal failed", "terminal cancelled"} {
		if reached[name] < 20 {
			t.Errorf("%s was reached %d times", name, reached[name])
		}
	}
	t.Logf("%v", reached)
}
