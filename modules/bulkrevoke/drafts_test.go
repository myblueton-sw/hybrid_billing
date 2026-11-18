package bulkrevoke

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

type fixture struct {
	t   testing.TB
	r   *Registry
	now *time.Time
}

func setup(t testing.TB, limits Limits) fixture {
	t.Helper()
	now := start
	r, err := New(limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return fixture{t: t, r: r, now: &now}
}

var defaults = Limits{MaxTargets: 5, Pending: 3, Kept: 2, Lifetime: 30 * time.Minute}

func (f fixture) want(name string, err, want error) {
	f.t.Helper()
	if !errors.Is(err, want) {
		f.t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func (f fixture) add(owner, draft string, users ...string) View {
	f.t.Helper()
	v, err := f.r.Add("w1", owner, draft, users)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func TestNewChecksItsInputs(t *testing.T) {
	for name, l := range map[string]Limits{"no targets": {Pending: 1, Kept: 1, Lifetime: 1}, "no pending": {MaxTargets: 1, Kept: 1, Lifetime: 1},
		"no kept": {MaxTargets: 1, Pending: 1, Lifetime: 1}, "no lifetime": {MaxTargets: 1, Pending: 1, Kept: 1}} {
		if _, err := New(l, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := New(defaults, nil); !errors.Is(err, ErrInvalid) {
		t.Error("no clock")
	}
}

func TestIdentifiers(t *testing.T) {
	for id, want := range map[string]uint64{"bulk-draft-7": 7, "bulk-draft-07": 0, "bulk-draft-": 0, "bulk-draft-x": 0, "bulk-draft-0": 0, "bulk-revocation-7": 0,
		"bulk-draft-99999999999999999999": 0} {
		if n, ok := Number(id, DraftPrefix); n != want || ok != (want > 0) {
			t.Errorf("%s: %d %v", id, n, ok)
		}
	}
	if DraftID(3) != "bulk-draft-3" || ManifestID(3) != "bulk-revocation-3" || !ValidUser("user-1") || ValidUser("bad id") || ValidUser("") ||
		ValidUser(string(make([]byte, 129))) || !SuspendUser.Valid() || Effect("remove").Valid() {
		t.Error("identifiers")
	}
}

func TestDrafts(t *testing.T) {
	f := setup(t, defaults)
	f.want("no workspace", second(f.r.Add("", "ann", "", []string{"u1"})), ErrInvalid)
	f.want("no owner", second(f.r.Add("w1", "", "", []string{"u1"})), ErrInvalid)
	f.want("no people", second(f.r.Add("w1", "ann", "", nil)), ErrInvalid)
	f.want("a malformed ID", second(f.r.Add("w1", "ann", "", []string{"u1", "bad id"})), ErrInvalid)
	if len(f.r.List("w1", "ann")) != 0 {
		t.Fatal("a refused batch created a draft")
	}
	v := f.add("ann", "", "u1", "u2", "u1")
	if v.ID != "bulk-draft-1" || v.Count != 2 || v.Duplicates != 1 || v.State != Draft || !v.ExpiresAt.Equal(start.Add(30*time.Minute)) || v.Owner != "ann" {
		t.Fatalf("draft %+v", v)
	}
	f.want("a second open draft", second(f.r.Add("w1", "ann", "", []string{"u3"})), ErrOpenDraft)
	f.want("an addition by someone else", second(f.r.Add("w1", "bob", v.ID, []string{"u3"})), ErrNotFound)
	f.want("an addition to an unknown draft", second(f.r.Add("w1", "ann", "bulk-draft-9", []string{"u3"})), ErrNotFound)
	f.want("an addition in another workspace", second(f.r.Add("w2", "ann", v.ID, []string{"u3"})), ErrNotFound)
	f.want("a batch past the limit", second(f.r.Add("w1", "ann", v.ID, []string{"u3", "u4", "u5", "u6"})), ErrTooMany)
	if users, _ := f.r.DraftUsers("w1", "ann", v.ID); len(users) != 2 {
		t.Fatalf("a refused batch was partly added %v", users)
	}
	*f.now = start.Add(10 * time.Minute)
	v = f.add("ann", v.ID, "u2", "u3", "u4", "u5")
	if v.Count != 5 || v.Duplicates != 2 || !v.ExpiresAt.Equal(start.Add(40*time.Minute)) {
		t.Fatalf("added %+v", v)
	}
	if users, err := f.r.DraftUsers("w1", "ann", v.ID); err != nil || len(users) != 5 || users[0] != "u1" {
		t.Errorf("users %v %v", users, err)
	}
	if _, err := f.r.DraftUsers("w1", "bob", v.ID); !errors.Is(err, ErrNotFound) {
		t.Error("someone else reads the draft")
	}
	if !f.r.Visible("w1", "ann", v.ID) || f.r.Visible("w1", "bob", v.ID) || f.r.Visible("w1", "ann", "bulk-revocation-1") || f.r.Visible("w1", "ann", "other") {
		t.Error("visibility")
	}
	if _, ok := f.r.Get("w1", "bob", v.ID); ok {
		t.Error("someone else gets the draft")
	}
	f.add("bob", "", "u1")
	f.add("cat", "", "u1")
	f.want("the pending limit", second(f.r.Add("w1", "dan", "", []string{"u1"})), ErrPending)
	pending, open := f.r.Limits("w1", "ann")
	if pending != 3 || !open {
		t.Errorf("limits %d %v", pending, open)
	}
	*f.now = start.Add(40 * time.Minute)
	f.r.Expire()
	if got, _ := f.r.Get("w1", "ann", "bulk-draft-1"); got.State != Expired {
		t.Errorf("expired at the boundary %+v", got)
	}
	pending, open = f.r.Limits("w1", "ann")
	if pending != 0 || open {
		t.Errorf("an expired draft holds a slot %d %v", pending, open)
	}
	v = f.add("ann", "", "u9")
	if v.ID != "bulk-draft-4" {
		t.Errorf("numbers continue %s", v.ID)
	}
	f.want("an addition to an expired draft", second(f.r.Add("w1", "ann", "bulk-draft-1", []string{"u3"})), ErrNotFound)
}

func TestDiscard(t *testing.T) {
	f := setup(t, defaults)
	v := f.add("ann", "", "u1")
	f.want("someone else's draft", second(f.r.Discard("w1", "bob", v.ID)), ErrNotFound)
	got, err := f.r.Discard("w1", "ann", v.ID)
	if err != nil || got.State != Discarded || got.ID != "bulk-draft-1" {
		t.Fatalf("%+v %v", got, err)
	}
	f.want("a discarded draft", second(f.r.Discard("w1", "ann", v.ID)), ErrState)
	f.want("a draft by a list ID", second(f.r.Discard("w1", "ann", ManifestID(1))), ErrNotFound)
	m := prepared(f, "ann", "u1", "u2")
	if _, err := f.r.Discard("w1", "bob", m.ID); err != nil {
		t.Errorf("a prepared list by another holder: %v", err)
	}
	for i := 0; i < 4; i++ {
		d := f.add("ann", "", "u"+strconv.Itoa(i))
		f.r.Discard("w1", "ann", d.ID)
	}
	if n := len(f.r.List("w1", "ann")); n != defaults.Kept {
		t.Errorf("ended lists kept %d", n)
	}
}

func second[T any](_ T, err error) error { return err }
