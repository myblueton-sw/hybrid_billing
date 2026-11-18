package bulkrevoke

import (
	"reflect"
	"testing"
	"time"
)

func TestLatestForEqualsLatestOnTheGivenPeople(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 10, Kept: 5, Lifetime: time.Hour})
	first := prepared(f, "ann", "u1", "u2", "u5")
	f.r.Execute("w1", first.ID, first.Digest, "bob", 1, map[string]string{"u2": HeldChanged})
	second := prepared(f, "cat", "u1", "u3")
	f.r.Execute("w1", second.ID, second.Digest, "bob", 1, nil)
	prepared(f, "dan", "u4")
	all := f.r.Latest("w1")
	for _, ids := range [][]string{nil, {"u1"}, {"u2", "u3"}, {"u3", "u5", "u9", "u1"}, {"u4"}} {
		want := map[string]Latest{}
		for _, id := range ids {
			if l, ok := all[id]; ok {
				want[id] = l
			}
		}
		if got := f.r.LatestFor("w1", ids); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: %+v; want %+v", ids, got, want)
		}
	}
	if got := f.r.LatestFor("w1", []string{"u1"}); got["u1"].Bulk != second.ID {
		t.Errorf("the later job wins %+v", got)
	}
	if len(f.r.LatestFor("w2", []string{"u1", "u3"})) != 0 {
		t.Error("another workspace")
	}
}

func TestLatestIndexFollowsExecutePruneHeaderAndRestore(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 50, Kept: 2, Lifetime: time.Hour})
	people := []string{"u1", "u2", "u3", "u4"}
	compare := func(label string) {
		t.Helper()
		all := f.r.Latest("w1")
		got := f.r.LatestFor("w1", append(people, "u9"))
		if !reflect.DeepEqual(got, all) {
			t.Errorf("%s: %+v; want %+v", label, got, all)
		}
	}
	owners := []string{"ann", "bob", "cat", "dan", "eve"}
	for i, owner := range owners {
		v := prepared(f, owner, people[i%4], people[(i+1)%4])
		f.r.Execute("w1", v.ID, v.Digest, "zed", 1, nil)
		compare("after execute " + owner)
	}
	if len(f.r.LatestFor("w1", people)) == 0 {
		t.Fatal("nothing indexed")
	}
	last := f.r.LatestFor("w1", []string{"u1"})["u1"]
	f.r.Header("w1", last.Bulk, Receipt{Reason: "ledger_full"})
	compare("after a header")
	if f.r.LatestFor("w1", []string{"u1"})["u1"].State == last.State {
		t.Error("an item state change was not read")
	}
	restored, _ := New(Limits{MaxTargets: 5, Pending: 50, Kept: 1, Lifetime: time.Hour}, func() time.Time { return start })
	restored.Restore(9, []View{{Number: 7, Workspace: "w1", Items: []Item{{Target: Target{User: "u1"}, Outcome: OutcomeUnprocessed}}},
		{Number: 8, Workspace: "w1", Items: []Item{{Target: Target{User: "u2"}, Outcome: OutcomeSucceeded}}}})
	if got := restored.LatestFor("w1", []string{"u1", "u2"}); len(got) != 2 || got["u2"].Bulk != ManifestID(8) {
		t.Errorf("restored %+v", got)
	}
}

func TestLatestIndexAfterFinishedJobsAreEvicted(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 50, Kept: 2, Lifetime: time.Hour})
	users := map[string][]string{"ann": {"u1", "u2"}, "bob": {"u2", "u3"}, "cat": {"u3"}, "dan": {"u4"}}
	jobs := []string{}
	for _, owner := range []string{"ann", "bob", "cat", "dan"} {
		v := prepared(f, owner, users[owner]...)
		executed, err := f.r.Execute("w1", v.ID, v.Digest, "zed", 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, executed.ID)
		if err := f.r.Header("w1", executed.ID, Receipt{Sequence: 1}); err != nil {
			t.Fatal(err)
		}
		for i := range executed.Items {
			f.r.Record("w1", executed.ID, []int{i}, []Receipt{{Sequence: uint64(i + 2)}})
			f.r.Cleaning("w1", executed.ID, i)
			f.r.Cleaned("w1", executed.ID, i, nil)
		}
		all := f.r.Latest("w1")
		if got := f.r.LatestFor("w1", []string{"u1", "u2", "u3", "u4", "u9"}); !reflect.DeepEqual(got, all) {
			t.Errorf("after %s: %+v; want %+v", owner, got, all)
		}
	}
	got := f.r.LatestFor("w1", []string{"u1", "u2", "u3", "u4"})
	if _, kept := got["u1"]; kept {
		t.Errorf("u1's only job was evicted but is still indexed: %+v", got["u1"])
	}
	if got["u2"].Bulk != jobs[1] || got["u3"].Bulk != jobs[2] || got["u4"].Bulk != jobs[3] {
		t.Errorf("u2 falls back to bob's job, u3 and u4 keep the latest: %+v", got)
	}
}
