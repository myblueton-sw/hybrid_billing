package bulkrevoke

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIdentifierBoundaries(t *testing.T) {
	if !ValidUser("AZaz09._:-") || !ValidUser(strings.Repeat("a", 128)) || ValidUser(strings.Repeat("a", 129)) {
		t.Error("the identifier boundaries")
	}
	for _, c := range []string{"@", "[", "`", "{", "/", ";", " "} {
		if ValidUser("a" + c) {
			t.Errorf("%q is accepted", c)
		}
	}
}

func TestErrorsNeverClaimReuse(t *testing.T) {
	f := setup(t, defaults)
	d := f.add("ann", "", "u1", "u2")
	for name, call := range map[string]func() (View, bool, error){
		"someone else": func() (View, bool, error) {
			return f.r.Prepare("w1", "bob", d.ID, SuspendMembership, "r1", nil, nil, Impact{})
		},
		"a bad effect": func() (View, bool, error) { return f.r.Prepare("w1", "ann", d.ID, "x", "r1", nil, nil, Impact{}) },
		"a missing user": func() (View, bool, error) {
			return f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", []Target{target("u1")}, nil, Impact{})
		},
		"a stranger": func() (View, bool, error) {
			return f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", []Target{target("u1")}, []Exclusion{{User: "u9"}}, Impact{})
		},
		"nobody": func() (View, bool, error) {
			return f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", nil, []Exclusion{{User: "u1"}, {User: "u2"}}, Impact{})
		},
	} {
		if _, reused, err := call(); err == nil || reused {
			t.Errorf("%s: %v %v", name, reused, err)
		}
	}
}

func TestIndexAndPageBoundaries(t *testing.T) {
	f := setup(t, defaults)
	v := prepared(f, "ann", "u1", "u2")
	if page, _, ok := f.r.Targets("w1", "ann", v.ID, 0, 1); !ok || len(page) != 1 || page[0].User != "u1" {
		t.Errorf("the first target page %+v", page)
	}
	v, _ = f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
	if items, _, ok := f.r.Items("w1", "x", v.ID, 0, 1, ""); !ok || len(items) != 1 {
		t.Error("an item page of one")
	}
	f.r.Header("w1", v.ID, Receipt{Sequence: 5})
	f.want("an index at the end", f.r.Record("w1", v.ID, []int{2}, []Receipt{{Sequence: 6}}), ErrState)
	if err := f.r.Record("w1", v.ID, []int{0, 1}, []Receipt{{Sequence: 5}, {Sequence: 6}}); err != nil {
		t.Fatal(err)
	}
	got := mustGet(f, v.ID)
	if got.Items[0].State != Unconfirmed || got.Items[0].Receipt.Reason != "commit_failed" || got.Items[1].State != DenyConfirmed || got.Items[1].Receipt.Reason != "" {
		t.Errorf("a receipt that does not advance %+v", got.Items[:2])
	}
	if err := f.r.Cleanup("w1", v.ID, true); err != nil {
		t.Errorf("cleanup flag %v", err)
	}
	for name, err := range map[string]error{"cleaning at the end": f.r.Cleaning("w1", v.ID, 2), "a failure at the end": f.r.Failed("w1", v.ID, 2, "x"),
		"cleaned at the end": f.r.Cleaned("w1", v.ID, 2, nil), "cleaned while confirmed": f.r.Cleaned("w1", v.ID, 1, nil), "cleaning an unconfirmed item": f.r.Cleaning("w1", v.ID, 0),
		"a failure of an unconfirmed item": f.r.Failed("w1", v.ID, 0, "x")} {
		if !errors.Is(err, ErrState) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOrderPruningAndReplayedJobs(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 10, Kept: 1, Lifetime: time.Hour})
	d := f.add("ann", "", "u1")
	v := prepared(f, "bob", "u2")
	f.r.Execute("w1", v.ID, v.Digest, "cat", 1, nil)
	rows := f.r.List("w1", "ann")
	if len(rows) != 2 || rows[0].ID != d.ID || rows[1].ID != v.ID {
		t.Errorf("a draft before a later executed job %+v", rows)
	}
	for i := 0; i < 3; i++ {
		x := f.add("dan", "", "u3")
		f.r.Discard("w1", "dan", x.ID)
	}
	p := prepared(f, "eve", "u4")
	ended := 0
	for _, row := range f.r.List("w1", "dan") {
		if row.State == Discarded {
			ended++
		}
	}
	if _, ok := f.r.Get("w1", "eve", p.ID); ended != 1 || !ok {
		t.Errorf("ended kept %d; prepared kept %v", ended, ok)
	}
	f.r.Restore(0, []View{{Number: 50, Workspace: "w1", Digest: "d", ReplayMissing: 1}, {Number: 51, Workspace: "w1", Digest: "d"}, {Number: 52, Workspace: "w1", Digest: "d"}})
	if _, ok := f.r.Get("w1", "x", ManifestID(50)); !ok {
		t.Error("a replayed job with unrecorded items was pruned")
	}
}

func TestExclusionPages(t *testing.T) {
	f := setup(t, defaults)
	d := f.add("ann", "", "u1", "u2", "u3")
	v, _, _ := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", []Target{target("u1")}, []Exclusion{{User: "u3", Reason: ExcludedSelf}, {User: "u2", Reason: ExcludedNotFound}}, Impact{})
	page, total, ok := f.r.Exclusions("w1", "ann", v.ID, 1, 1)
	if !ok || total != 2 || len(page) != 1 || page[0].User != "u3" {
		t.Errorf("exclusions %+v %d %v", page, total, ok)
	}
	for name, ok := range map[string]bool{"someone else": second3(f.r.Exclusions("w1", "bob", v.ID, 0, 1)), "a draft": second3(f.r.Exclusions("w1", "ann", f.add("ann", "", "u9").ID, 0, 1)),
		"a negative offset": second3(f.r.Exclusions("w1", "ann", v.ID, -1, 1)), "no limit": second3(f.r.Exclusions("w1", "ann", v.ID, 0, 0))} {
		if ok {
			t.Errorf("%s reads exclusions", name)
		}
	}
	if page, _, ok := f.r.Exclusions("w1", "ann", v.ID, 0, 1); !ok || len(page) != 1 {
		t.Error("the first exclusion page of one")
	}
}

func second3[A any](_ A, _ int, ok bool) bool { return ok }
