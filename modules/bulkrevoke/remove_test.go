package bulkrevoke

import (
	"testing"
	"time"
)

func TestRemoveMarksPeopleOfAFilterList(t *testing.T) {
	f := setup(t, defaults)
	terms := []FilterTerm{{Key: "role", Value: "member"}}
	d, err := f.r.AddMatching("w1", "ann", terms, []string{"u1", "u2", "u3", "ann"}, start, "r7")
	if err != nil {
		t.Fatal(err)
	}
	*f.now = start.Add(10 * time.Minute)
	v, got, err := f.r.Remove("w1", "ann", d.ID, []string{"u2", "ghost", "ann", "u2"})
	if err != nil || got != (Removal{Removed: 1, Already: 1, Ignored: 2}) || v.Chosen != 1 || v.Matched != 4 || v.Count != 4 || !v.ExpiresAt.Equal(start.Add(40*time.Minute)) {
		t.Fatalf("first removal %+v %+v %v", v, got, err)
	}
	*f.now = start.Add(20 * time.Minute)
	v, got, err = f.r.Remove("w1", "ann", d.ID, []string{"u2", "ghost"})
	if err != nil || got != (Removal{Already: 1, Ignored: 1}) || v.Chosen != 1 || !v.ExpiresAt.Equal(start.Add(40*time.Minute)) || !v.Updated.Equal(start.Add(10*time.Minute)) {
		t.Fatalf("a removal that marks nobody keeps the lifetime %+v %+v %v", v, got, err)
	}
	users, _ := f.r.DraftUsers("w1", "ann", d.ID)
	chosen, err := f.r.DraftChosen("w1", "ann", d.ID)
	if len(users) != 4 || err != nil || len(chosen) != 1 || !chosen["u2"] {
		t.Fatalf("the list keeps everyone and knows who was removed: %v %v %v", users, chosen, err)
	}
	if listed := f.r.List("w1", "ann"); len(listed) != 1 || listed[0].Chosen != 1 {
		t.Errorf("the listing %+v", listed)
	}
	f.want("another person's draft", third(f.r.Remove("w1", "bob", d.ID, []string{"u1"})), ErrNotFound)
	f.want("another workspace", third(f.r.Remove("w2", "ann", d.ID, []string{"u1"})), ErrNotFound)
	f.want("no identifiers", third(f.r.Remove("w1", "ann", d.ID, nil)), ErrInvalid)
	f.want("a malformed identifier", third(f.r.Remove("w1", "ann", d.ID, []string{"u1", "a b"})), ErrInvalid)
	if c, _ := f.r.DraftChosen("w1", "ann", d.ID); len(c) != 1 {
		t.Errorf("a refused request marked someone %v", c)
	}
	f.want("chosen people of another person's draft", second(f.r.DraftChosen("w1", "bob", d.ID)), ErrNotFound)

	both := []Exclusion{{User: "ann", Login: "ann@example.invalid", Reason: ExcludedSelf}, {User: "u2", Login: "u2@example.invalid", Reason: ExcludedChosen}}
	_, _, err = f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", []Target{target("u1"), target("u2"), target("u3")}, both[:1], Impact{})
	f.want("a removed person among the targets", err, ErrInvalid)
	plain := ListDigest(d, SuspendMembership, []Target{target("u1"), target("u2"), target("u3")}, both[:1])
	p, _, err := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", []Target{target("u1"), target("u3")}, both, Impact{})
	if err != nil || p.Digest == plain || p.Counts.Targets != 2 || p.Counts.Excluded != 2 || p.Chosen != 1 || p.Matched != 4 {
		t.Fatalf("prepared %+v %v", p, err)
	}
	f.want("a prepared list by its own identifier", third(f.r.Remove("w1", "ann", p.ID, []string{"u1"})), ErrState)
	f.want("a prepared list by its draft identifier", third(f.r.Remove("w1", "ann", d.ID, []string{"u1"})), ErrNotFound)
	f.want("a prepared list of another person", third(f.r.Remove("w1", "bob", p.ID, []string{"u1"})), ErrNotFound)
}

func TestRemoveIsForFilterListsOnly(t *testing.T) {
	f := setup(t, defaults)
	d := f.add("ann", "", "u1", "u2")
	f.want("a draft of chosen people", third(f.r.Remove("w1", "ann", d.ID, []string{"u1"})), ErrState)
	if chosen, err := f.r.DraftChosen("w1", "ann", d.ID); err != nil || len(chosen) != 0 {
		t.Errorf("nobody is removed from a draft of chosen people %v %v", chosen, err)
	}
	again := f.add("ann", d.ID, "u1", "u3")
	if again.Count != 3 || again.Duplicates != 1 {
		t.Errorf("adding to it still works %+v", again)
	}
	*f.now = start.Add(time.Hour)
	g, err := f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "role", Value: "member"}}, []string{"u1"}, start, "r7")
	if err != nil {
		t.Fatal(err)
	}
	*f.now = start.Add(3 * time.Hour)
	f.want("an expired draft", third(f.r.Remove("w1", "ann", g.ID, []string{"u1"})), ErrNotFound)
}

func third[A, B any](_ A, _ B, err error) error { return err }
