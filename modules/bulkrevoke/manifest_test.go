package bulkrevoke

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func target(id string) Target {
	return Target{User: id, Login: id + "@example.invalid", Role: "member", Version: 2, Epoch: 1, UserState: "active", MembershipState: "active"}
}

func prepared(f fixture, owner string, users ...string) View {
	f.t.Helper()
	d := f.add(owner, "", users...)
	targets := []Target{}
	for _, u := range users {
		targets = append(targets, target(u))
	}
	v, reused, err := f.r.Prepare("w1", owner, d.ID, SuspendMembership, "r1", targets, nil, Impact{People: len(users), Pending: []PendingStore{{Store: "m03", Count: 1}}})
	if err != nil || reused {
		f.t.Fatalf("prepare %v %v", err, reused)
	}
	return v
}

func TestDigest(t *testing.T) {
	targets := []Target{target("u2"), target("u1")}
	exclusions := []Exclusion{{User: "u4", Reason: ExcludedNotFound}, {User: "u3", Reason: ExcludedSelf}}
	base := Digest(SuspendMembership, Explicit, targets, exclusions)
	if len(base) != 64 || base != Digest(SuspendMembership, Explicit, []Target{target("u1"), target("u2")}, []Exclusion{exclusions[1], exclusions[0]}) {
		t.Fatal("the digest depends on order")
	}
	changes := map[string]func(*Target){
		"user": func(x *Target) { x.User = "u9" }, "login": func(x *Target) { x.Login = "x" }, "role": func(x *Target) { x.Role = "administrator" },
		"until": func(x *Target) { x.Until = start }, "version": func(x *Target) { x.Version++ }, "epoch": func(x *Target) { x.Epoch++ },
		"user state": func(x *Target) { x.UserState = "suspended" }, "membership state": func(x *Target) { x.MembershipState = "suspended" },
		"administrator": func(x *Target) { x.Administrator = true },
	}
	for name, change := range changes {
		copied := append([]Target(nil), targets...)
		change(&copied[0])
		if Digest(SuspendMembership, Explicit, copied, exclusions) == base {
			t.Errorf("the digest ignores the %s", name)
		}
	}
	if Digest(SuspendUser, Explicit, targets, exclusions) == base || Digest(SuspendMembership, "filter", targets, exclusions) == base ||
		Digest(SuspendMembership, Explicit, targets, exclusions[:1]) == base ||
		Digest(SuspendMembership, Explicit, targets, []Exclusion{{User: "u4", Reason: ExcludedSelf}, exclusions[1]}) == base ||
		Digest(SuspendMembership, Explicit, targets, []Exclusion{{User: "u4", Login: "x", Reason: ExcludedNotFound}, exclusions[1]}) == base {
		t.Error("the digest ignores the effect, selection or exclusions")
	}
}

func TestPrepare(t *testing.T) {
	f := setup(t, defaults)
	d := f.add("ann", "", "u1", "u2", "u3")
	targets := []Target{target("u2"), target("u1")}
	exclusions := []Exclusion{{User: "u3", Reason: ExcludedNotFound}}
	_, _, err := f.r.Prepare("w1", "bob", d.ID, SuspendMembership, "r1", targets, exclusions, Impact{})
	f.want("someone else prepares", err, ErrNotFound)
	_, _, err = f.r.Prepare("w1", "ann", d.ID, "remove", "r1", targets, exclusions, Impact{})
	f.want("an unknown effect", err, ErrInvalid)
	_, _, err = f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "", targets, exclusions, Impact{})
	f.want("no revision", err, ErrInvalid)
	_, _, err = f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", targets, nil, Impact{})
	f.want("a missing person", err, ErrInvalid)
	_, _, err = f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", append(targets, target("u3")), exclusions, Impact{})
	f.want("a person twice", err, ErrInvalid)
	_, _, err = f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", targets, []Exclusion{{User: "u9"}}, Impact{})
	f.want("a person outside the draft", err, ErrInvalid)
	_, _, err = f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", nil, []Exclusion{{User: "u1"}, {User: "u2"}, {User: "u3"}}, Impact{})
	f.want("nobody to revoke", err, ErrEmpty)
	*f.now = start.Add(5 * time.Minute)
	v, reused, err := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r7", targets, exclusions, Impact{People: 2, Pending: []PendingStore{{Store: "m02"}}})
	if err != nil || reused || v.ID != "bulk-revocation-1" || v.State != Prepared || v.Targets[0].User != "u1" || v.Revision != "r7" ||
		!v.ExpiresAt.Equal(start.Add(35*time.Minute)) || v.Counts != (Counts{Targets: 2, Excluded: 1}) || v.Digest != Digest(SuspendMembership, Explicit, targets, exclusions) {
		t.Fatalf("prepared %+v %v %v", v, reused, err)
	}
	f.want("additions after freezing", second(f.r.Add("w1", "ann", d.ID, []string{"u4"})), ErrNotFound)
	if _, _, err := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r7", targets, exclusions, Impact{}); !errors.Is(err, ErrNotFound) {
		t.Error("prepared twice")
	}
	again := f.add("ann", "", "u1", "u2", "u3")
	same, reused, err := f.r.Prepare("w1", "ann", again.ID, SuspendMembership, "r8", targets, exclusions, Impact{})
	if err != nil || !reused || same.ID != v.ID {
		t.Fatalf("an equal list %+v %v %v", same, reused, err)
	}
	if _, ok := f.r.Get("w1", "ann", again.ID); ok {
		t.Error("the equal draft is kept")
	}
	other := f.add("ann", "", "u1", "u2", "u3")
	if w, reused, _ := f.r.Prepare("w1", "ann", other.ID, SuspendUser, "r8", targets, exclusions, Impact{}); reused || w.ID == v.ID {
		t.Error("a different effect reused the list")
	}
	page, total, ok := f.r.Targets("w1", "ann", v.ID, 1, 5)
	if !ok || total != 2 || len(page) != 1 || page[0].User != "u2" {
		t.Errorf("targets %+v %d %v", page, total, ok)
	}
	if _, _, ok := f.r.Targets("w1", "bob", v.ID, 0, 5); ok {
		t.Error("someone else reads the targets")
	}
	if _, _, ok := f.r.Targets("w1", "ann", v.ID, -1, 5); ok {
		t.Error("a negative offset")
	}
	if _, _, ok := f.r.Targets("w1", "ann", v.ID, 0, 0); ok {
		t.Error("a zero limit")
	}
	if page, _, ok := f.r.Targets("w1", "ann", v.ID, 9, 5); !ok || len(page) != 0 {
		t.Error("an offset past the end")
	}
}

func TestExecute(t *testing.T) {
	f := setup(t, defaults)
	d := f.add("ann", "", "u1", "u2", "u3", "u4")
	v, _, _ := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", []Target{target("u1"), target("u2"), target("u3")},
		[]Exclusion{{User: "u4", Reason: ExcludedSelf}}, Impact{})
	if _, err := f.r.Begin("w1", "bulk-revocation-9", v.Digest); !errors.Is(err, ErrNotFound) {
		t.Error("an unknown list")
	}
	if _, err := f.r.Begin("w1", v.ID, "x"); !errors.Is(err, ErrDigest) {
		t.Error("a wrong digest")
	}
	if got, err := f.r.Begin("w1", v.ID, v.Digest); err != nil || got.ID != v.ID {
		t.Fatal(err)
	}
	f.want("no executor", second(f.r.Execute("w1", v.ID, v.Digest, "", 1, nil)), ErrInvalid)
	f.want("an unknown list", second(f.r.Execute("w1", "bulk-revocation-9", v.Digest, "bob", 1, nil)), ErrNotFound)
	f.want("a wrong digest", second(f.r.Execute("w1", v.ID, "x", "bob", 1, nil)), ErrDigest)
	got, err := f.r.Execute("w1", v.ID, v.Digest, "bob", 42, map[string]string{"u2": HeldChanged})
	if err != nil || got.State != Executed || got.Executor != "bob" || got.DenyGeneration != 42 || len(got.Items) != 4 ||
		got.Counts != (Counts{Targets: 3, Excluded: 1, Held: 1, Unprocessed: 2}) || got.Items[1].Outcome != OutcomeHeld || got.Items[3].Outcome != OutcomeExcluded {
		t.Fatalf("executed %+v %v", got, err)
	}
	f.want("executed twice", second(f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)), ErrState)
	f.want("begin after execution", second(f.r.Begin("w1", v.ID, v.Digest)), ErrState)
	f.want("a record before the header", f.r.Record("w1", v.ID, []int{0}, []Receipt{{Sequence: 3}}), ErrState)
	f.want("a header of an unknown job", f.r.Header("w1", "bulk-revocation-9", Receipt{}), ErrNotFound)
	if err := f.r.Header("w1", v.ID, Receipt{Sequence: 2, Watermark: "2:x"}); err != nil {
		t.Fatal(err)
	}
	f.want("a second header", f.r.Header("w1", v.ID, Receipt{}), ErrState)
	f.want("mismatched receipts", f.r.Record("w1", v.ID, []int{0, 2}, []Receipt{{}}), ErrState)
	f.want("a held item", f.r.Record("w1", v.ID, []int{1}, []Receipt{{}}), ErrState)
	f.want("an index out of range", f.r.Record("w1", v.ID, []int{7}, []Receipt{{}}), ErrState)
	f.want("cleaning before the record", f.r.Cleaning("w1", v.ID, 0), ErrState)
	if err := f.r.Record("w1", v.ID, []int{0, 2}, []Receipt{{Sequence: 3, Watermark: "3:a"}, {Reason: "commit_unknown"}}); err != nil {
		t.Fatal(err)
	}
	f.want("a second record", f.r.Record("w1", v.ID, []int{0}, []Receipt{{}}), ErrState)
	state, reason := JobState(mustGet(f, v.ID), Cleaning{State: CleaningRunning})
	if state != JobWaiting || reason != ReasonUnconfirmed {
		t.Errorf("job with an unconfirmed item %s %s", state, reason)
	}
	f.want("cleaning an unconfirmed item", f.r.Cleaning("w1", v.ID, 2), ErrState)
	f.want("cleaned before cleaning", f.r.Cleaned("w1", v.ID, 0, nil), ErrState)
	if err := f.r.Cleaning("w1", v.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.r.Cleaning("w1", v.ID, 0); err != nil {
		t.Error("a retried chunk starts cleaning again")
	}
	if err := f.r.Cleaned("w1", v.ID, 0, []Cleanup{{Item: "B2", State: "done"}}); err != nil {
		t.Fatal(err)
	}
	f.want("a failure after completion", f.r.Failed("w1", v.ID, 0, "x"), ErrState)
	f.want("a failure out of range", f.r.Failed("w1", v.ID, 9, "x"), ErrState)
	f.want("cleaned out of range", f.r.Cleaned("w1", v.ID, -1, nil), ErrState)
	f.want("cleaning out of range", f.r.Cleaning("w1", v.ID, 9), ErrState)
	got = mustGet(f, v.ID)
	if got.Counts != (Counts{Targets: 3, Excluded: 1, Held: 1, Succeeded: 1, Unprocessed: 1}) || got.Items[0].State != Complete || got.Items[0].Cleanup[0].Item != "B2" {
		t.Errorf("after cleanup %+v", got.Counts)
	}
	items, total, ok := f.r.Items("w1", "zed", v.ID, 0, 10, OutcomeSucceeded)
	if !ok || total != 1 || items[0].Target.User != "u1" {
		t.Errorf("items %+v %d", items, total)
	}
	if items, total, _ := f.r.Items("w1", "zed", v.ID, 3, 10, ""); total != 4 || len(items) != 1 {
		t.Errorf("paged items %+v %d", items, total)
	}
	if _, _, ok := f.r.Items("w1", "zed", v.ID, -1, 10, ""); ok {
		t.Error("a negative offset")
	}
	if _, _, ok := f.r.Items("w1", "zed", "bulk-draft-1", 0, 10, ""); ok {
		t.Error("items of a draft")
	}
	f.want("a cleanup flag on an unknown job", f.r.Cleanup("w1", "bulk-revocation-9", true), ErrNotFound)
}

func mustGet(f fixture, id string) View {
	f.t.Helper()
	v, ok := f.r.Get("w1", "ann", id)
	if !ok {
		f.t.Fatalf("%s is gone", id)
	}
	return v
}

func TestJobStates(t *testing.T) {
	f := setup(t, defaults)
	v := prepared(f, "ann", "u1", "u2")
	v, _ = f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
	f.r.Header("w1", v.ID, Receipt{Sequence: 2})
	f.r.Record("w1", v.ID, []int{0, 1}, []Receipt{{Sequence: 3}, {Sequence: 4}})
	f.r.Cleanup("w1", v.ID, false)
	if s, r := JobState(mustGet(f, v.ID), Cleaning{}); s != JobWaiting || r != ReasonNotStarted {
		t.Errorf("not started %s %s", s, r)
	}
	f.r.Cleanup("w1", v.ID, true)
	for cleaning, want := range map[string][2]string{CleaningQueued: {JobRunning, ""}, CleaningRunning: {JobRunning, ""}, CleaningWaiting: {JobWaiting, "outcome_unknown"},
		CleaningSucceeded: {JobSucceeded, ""}, CleaningFailed: {JobSucceeded, ""}, CleaningNotCreated: {JobRunning, ""}} {
		if s, r := JobState(mustGet(f, v.ID), Cleaning{State: cleaning, Reason: "outcome_unknown"}); s != want[0] || r != want[1] && want[1] != "" {
			t.Errorf("%q: %s %s", cleaning, s, r)
		}
	}
	f.r.Cleaning("w1", v.ID, 0)
	f.r.Failed("w1", v.ID, 0, "attempts_spent")
	f.r.Failed("w1", v.ID, 1, "attempts_spent")
	got := mustGet(f, v.ID)
	if s, _ := JobState(got, Cleaning{}); s != JobSucceeded || got.Counts.Failed != 2 || got.Items[0].Reason != "attempts_spent" || got.Items[0].State != DenyConfirmed {
		t.Errorf("failed %s %+v", s, got.Counts)
	}
	w := prepared(f, "cat", "u1")
	w, _ = f.r.Execute("w1", w.ID, w.Digest, "bob", 1, nil)
	f.r.Header("w1", w.ID, Receipt{Reason: "ledger_isolated"})
	got = mustGet(f, w.ID)
	if got.Items[0].State != Unconfirmed || got.Items[0].Receipt.Reason != "ledger_isolated" || got.Counts.Unprocessed != 1 {
		t.Errorf("unconfirmed header %+v", got.Items[0])
	}
}

func TestPruningKeepsUnfinishedJobs(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 10, Kept: 2, Lifetime: time.Hour})
	ids := []string{}
	for i := 0; i < 5; i++ {
		v := prepared(f, "o"+strconv.Itoa(i), "u1")
		v, _ = f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
		f.r.Header("w1", v.ID, Receipt{Sequence: 1})
		f.r.Record("w1", v.ID, []int{0}, []Receipt{{Sequence: 2}})
		if i != 1 {
			f.r.Cleaning("w1", v.ID, 0)
			f.r.Cleaned("w1", v.ID, 0, nil)
		}
		ids = append(ids, v.ID)
	}
	v := prepared(f, "o9", "u1")
	f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
	kept := map[string]bool{}
	for _, row := range f.r.List("w1", "x") {
		kept[row.ID] = true
	}
	if !kept[ids[1]] || !kept[ids[4]] || !kept[ids[3]] || kept[ids[0]] || kept[ids[2]] {
		t.Errorf("kept %v", kept)
	}
}

func TestListOrderAndRestore(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 10, Kept: 5, Lifetime: time.Hour})
	done := prepared(f, "ann", "u1")
	f.r.Execute("w1", done.ID, done.Digest, "bob", 1, nil)
	p := prepared(f, "bob", "u1")
	d := f.add("ann", "", "u2")
	x := f.add("bob", "", "u3")
	f.r.Discard("w1", "bob", x.ID)
	got := []string{}
	for _, v := range f.r.List("w1", "ann") {
		got = append(got, v.ID)
	}
	if !reflect.DeepEqual(got, []string{d.ID, p.ID, done.ID}) || len(f.r.List("w1", "bob")) != 3 {
		t.Errorf("order %v", got)
	}
	if f.r.Highest() != 4 {
		t.Errorf("highest %d", f.r.Highest())
	}
	r := setup(t, defaults)
	r.r.Restore(7, []View{{Number: 5, Workspace: "w1", Effect: SuspendUser, Digest: "d", Items: []Item{{Outcome: OutcomeUnprocessed, Reason: RecordNotConfirmed}, {Outcome: OutcomeSucceeded}}}})
	v, ok := r.r.Get("w1", "x", "bulk-revocation-5")
	if !ok || v.Origin != OriginReplay || v.State != Executed || v.Counts != (Counts{Targets: 2, Succeeded: 1, Unprocessed: 1}) || r.r.Highest() != 7 {
		t.Fatalf("restored %+v", v)
	}
	r.r.Restore(3, []View{{Number: 9, Workspace: "w1", Digest: "d", ReplayHeld: 2, ReplayExcluded: 1, ReplayMissing: 3}})
	missing, _ := r.r.Get("w1", "x", "bulk-revocation-9")
	if missing.Counts != (Counts{Targets: 5, Held: 2, Excluded: 1, Unprocessed: 3}) {
		t.Errorf("replayed counts %+v", missing.Counts)
	}
	if s, reason := JobState(missing, Cleaning{}); s != JobWaiting || reason != ReasonUnconfirmed {
		t.Errorf("a job with unrecorded items %s %s", s, reason)
	}
	if r.r.Highest() != 9 || r.add("ann", "", "u1").ID != "bulk-draft-10" {
		t.Error("numbers after a restore")
	}
}

func TestLatestRevocationPerPerson(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, Pending: 10, Kept: 5, Lifetime: time.Hour})
	first := prepared(f, "ann", "u1", "u2")
	f.r.Execute("w1", first.ID, first.Digest, "bob", 1, map[string]string{"u2": HeldChanged})
	second := prepared(f, "cat", "u1", "u3")
	f.r.Execute("w1", second.ID, second.Digest, "bob", 1, nil)
	f.r.Header("w1", second.ID, Receipt{Reason: "ledger_full"})
	prepared(f, "dan", "u4")
	got := f.r.Latest("w1")
	if len(got) != 2 || got["u1"] != (Latest{Bulk: second.ID, State: Unconfirmed}) || got["u3"].Bulk != second.ID {
		t.Errorf("latest %+v", got)
	}
	if len(f.r.Latest("w2")) != 0 {
		t.Error("another workspace")
	}
}
