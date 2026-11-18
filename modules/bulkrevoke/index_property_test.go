package bulkrevoke

import (
	"math/rand/v2"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type indexWorld struct {
	t     *testing.T
	f     fixture
	r     *rand.Rand
	pool  []string
	ready []View
}

func (w *indexWorld) people() []string {
	out := []string{}
	for _, u := range w.pool {
		if w.r.IntN(3) == 0 {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		out = append(out, w.pool[w.r.IntN(len(w.pool))])
	}
	return out
}

func (w *indexWorld) prepare(owner string) {
	users := w.people()
	d, err := w.f.r.Add("w1", owner, "", users)
	if err != nil {
		return
	}
	targets, exclusions := []Target{}, []Exclusion{}
	for _, u := range users {
		if w.r.IntN(4) == 0 {
			exclusions = append(exclusions, Exclusion{User: u, Reason: ExcludedNotFound})
			continue
		}
		targets = append(targets, target(u))
	}
	if len(targets) == 0 {
		_, _ = w.f.r.Discard("w1", owner, d.ID)
		return
	}
	v, _, err := w.f.r.Prepare("w1", owner, d.ID, SuspendMembership, "r1", targets, exclusions, Impact{})
	if err != nil {
		w.t.Fatal(err)
	}
	w.ready = append(w.ready, v)
}

func (w *indexWorld) execute() {
	if len(w.ready) == 0 {
		return
	}
	i := w.r.IntN(len(w.ready))
	v := w.ready[i]
	w.ready = append(w.ready[:i], w.ready[i+1:]...)
	held := map[string]string{}
	for _, x := range v.Targets {
		if w.r.IntN(5) == 0 {
			held[x.User] = HeldChanged
		}
	}
	if _, err := w.f.r.Execute("w1", v.ID, v.Digest, "zed", 1, held); err != nil {
		return
	}
	if w.r.IntN(3) == 0 {
		return
	}
	w.finish(v.ID)
}

func (w *indexWorld) finish(id string) {
	got, ok := w.f.r.Get("w1", "ann", id)
	if !ok || got.State != Executed {
		return
	}
	if !got.Headed {
		_ = w.f.r.Header("w1", id, Receipt{Sequence: 1})
	}
	for _, item := range got.Items {
		if item.Outcome != OutcomeUnprocessed {
			continue
		}
		_ = w.f.r.Record("w1", id, []int{item.Index}, []Receipt{{Sequence: 2}})
		_ = w.f.r.Cleaning("w1", id, item.Index)
		if w.r.IntN(4) == 0 {
			_ = w.f.r.Failed("w1", id, item.Index, "x")
			continue
		}
		_ = w.f.r.Cleaned("w1", id, item.Index, nil)
	}
}

func (w *indexWorld) restore() {
	listed := w.f.r.List("w1", "ann")
	jobs := []View{}
	for _, row := range listed {
		if row.State == Executed && w.r.IntN(2) == 0 {
			full, _ := w.f.r.Get("w1", "ann", row.ID)
			if w.r.IntN(2) == 0 && len(full.Items) > 0 {
				full.Items[0].Outcome = OutcomeHeld
			}
			jobs = append(jobs, full)
		}
	}
	if w.r.IntN(2) == 0 {
		items := []Item{}
		for i, u := range w.people() {
			items = append(items, Item{Index: i, Target: target(u), Outcome: OutcomeSucceeded, State: Complete})
		}
		jobs = append(jobs, View{Number: uint64(1 + w.r.IntN(60)), Workspace: "w1", Items: items})
	}
	w.f.r.Restore(0, jobs)
}

func (w *indexWorld) check(step string) {
	w.t.Helper()
	for _, workspace := range []string{"w1", "w2"} {
		want := w.f.r.Latest(workspace)
		got := w.f.r.LatestFor(workspace, w.pool)
		if len(want) == 0 && len(got) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			w.t.Fatalf("%s %s: index %v, oracle %v", step, workspace, got, want)
		}
	}
}

func TestIncrementalIndexEqualsTheFullScan(t *testing.T) {
	for seed := uint64(0); seed < 60; seed++ {
		w := &indexWorld{t: t, f: setup(t, Limits{MaxTargets: 8, MaxMatching: 8, Pending: 6, Kept: 1 + int(seed%4), Lifetime: time.Hour}), r: rand.New(rand.NewPCG(seed, 323))}
		for i := 0; i < 8; i++ {
			w.pool = append(w.pool, "u"+strconv.Itoa(i))
		}
		owners := []string{"ann", "bob", "cat", "dan", "eve"}
		for step := 0; step < 120; step++ {
			switch w.r.IntN(10) {
			case 0, 1, 2:
				w.prepare(owners[w.r.IntN(len(owners))])
			case 3, 4, 5:
				w.execute()
			case 6:
				for _, row := range w.f.r.List("w1", "ann") {
					if row.State == Executed {
						w.finish(row.ID)
						break
					}
				}
			case 7:
				w.restore()
			case 8:
				if len(w.ready) > 0 {
					_, _ = w.f.r.Discard("w1", w.ready[0].Owner, w.ready[0].ID)
					w.ready = w.ready[1:]
				}
			default:
				*w.f.now = w.f.now.Add(time.Duration(w.r.IntN(40)) * time.Minute)
				w.f.r.Expire()
				w.f.r.Restore(0, nil)
			}
			w.check("seed " + strconv.FormatUint(seed, 10) + " step " + strconv.Itoa(step))
		}
	}
}

func TestIndexWorkIsTheNewAndEvictedJobsOnly(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 50, MaxMatching: 50, Pending: 5, Kept: 2, Lifetime: time.Hour})
	run := func(n int) View {
		users := []string{}
		for i := 0; i < n; i++ {
			users = append(users, "u"+strconv.Itoa(i))
		}
		v := prepared(f, "ann", users...)
		before := f.r.visited
		v, err := f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		f.r.visited -= before
		return v
	}
	finish := func(v View) {
		_ = f.r.Header("w1", v.ID, Receipt{Sequence: 1})
		for _, item := range v.Items {
			_ = f.r.Record("w1", v.ID, []int{item.Index}, []Receipt{{Sequence: 2}})
			_ = f.r.Cleaning("w1", v.ID, item.Index)
			_ = f.r.Cleaned("w1", v.ID, item.Index, nil)
		}
	}
	sizes := []int{40, 30, 20, 10}
	jobs := []View{}
	for i, n := range sizes {
		f.r.visited = 0
		jobs = append(jobs, run(n))
		want := n
		if i >= 3 {
			want += sizes[i-3]
		}
		if f.r.visited != want {
			t.Errorf("job %d: %d items visited, want %d", i, f.r.visited, want)
		}
		if people := len(f.r.refs["w1"]); i >= 3 && people != sizes[i-2] {
			t.Errorf("job %d: %d people indexed after the eviction, want %d", i, people, sizes[i-2])
		}
		finish(jobs[i])
	}
}

func TestIndexKeepsTheFirstItemOfAPersonInAJob(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, MaxMatching: 5, Pending: 5, Kept: 5, Lifetime: time.Hour})
	items := []Item{{Index: 0, Target: target("u1"), Outcome: OutcomeSucceeded, State: "first"}, {Index: 1, Target: target("u1"), Outcome: OutcomeSucceeded, State: "second"},
		{Index: 2, Target: Target{}, Outcome: OutcomeSucceeded}}
	f.r.Restore(0, []View{{Number: 4, Workspace: "w1", Items: items}})
	if got := f.r.LatestFor("w1", []string{"u1", ""}); !reflect.DeepEqual(got, f.r.Latest("w1")) || got["u1"].State != "first" {
		t.Errorf("latest %v", got)
	}
	f.r.Restore(0, []View{{Number: 4, Workspace: "w1", Items: items[1:2]}})
	if got := f.r.LatestFor("w1", []string{"u1"}); got["u1"].State != "second" || len(f.r.refs["w1"]["u1"]) != 1 {
		t.Errorf("after a repeated restore %v %v", got, f.r.refs["w1"])
	}
}
