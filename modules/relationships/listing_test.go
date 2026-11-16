package relationships

import (
	"errors"
	"reflect"
	"testing"
)

func TestDayISO(t *testing.T) {
	if got := day(t, 2026, 3, 7).ISO(); got != "2026-03-07" {
		t.Errorf("iso %s", got)
	}
	if got := day(t, 1, 12, 31).ISO(); got != "0001-12-31" {
		t.Errorf("iso %s", got)
	}
}

func TestTimelinesListing(t *testing.T) {
	f := setup(t)
	if got, err := f.Timelines("w1", 0, Payer, "", 2); err != nil || len(got) != 0 {
		t.Fatalf("empty workspace %v %v", got, err)
	}
	first := f.publish("p1",
		line(Payer, "acct-b", Entry{Target: "le-1", Interval: From(nov(t, 1))}),
		line(Payer, "acct-a", Entry{Target: "le-1", Interval: Between(nov(t, 1), nov(t, 10))}, Entry{Target: "le-2", Interval: From(nov(t, 10))}),
		line(Payer, "acct-c", Entry{Target: "le-2", Interval: From(nov(t, 1))}),
		line(Hierarchy, "acct-a", Entry{Target: "unit-1", Interval: From(nov(t, 1))}))
	page, err := f.Timelines("w1", first, Payer, "", 2)
	if err != nil || len(page) != 2 || page[0].Member != "acct-a" || page[1].Member != "acct-b" || page[0].Kind != Payer {
		t.Fatalf("first page %+v %v", page, err)
	}
	if len(page[0].Entries) != 2 || page[0].Entries[1].Target != "le-2" {
		t.Errorf("entries %+v", page[0].Entries)
	}
	page[0].Entries[0].Target = "changed"
	again, _ := f.Timelines("w1", first, Payer, "", 1)
	if again[0].Entries[0].Target != "le-1" {
		t.Error("entries are shared with the registry")
	}
	rest, err := f.Timelines("w1", first, Payer, "acct-b", 2)
	if err != nil || len(rest) != 1 || rest[0].Member != "acct-c" {
		t.Errorf("second page %+v %v", rest, err)
	}
	if tree, _ := f.Timelines("w1", first, Hierarchy, "", 2); len(tree) != 1 || tree[0].Member != "acct-a" {
		t.Errorf("other kind %+v", tree)
	}
	if before, _ := f.Timelines("w1", 0, Payer, "", 2); len(before) != 0 {
		t.Errorf("revision 0 %+v", before)
	}
	for name, err := range map[string]error{
		"an unknown kind":    second(f.Timelines("w1", first, Kind("other"), "", 1)),
		"a zero limit":       second(f.Timelines("w1", first, Payer, "", 0)),
		"a limit above page": second(f.Timelines("w1", first, Payer, "", 3)),
		"a bad cursor":       second(f.Timelines("w1", first, Payer, "bad cursor", 1)),
		"a bad workspace":    second(f.Timelines("", first, Payer, "", 1)),
	} {
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, err := range map[string]error{
		"a future revision":                  second(f.Timelines("w1", first+1, Payer, "", 1)),
		"a revision of an unknown workspace": second(f.Timelines("w9", 1, Payer, "", 1)),
	} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := f.Draft("w2", "q1", 0); err != nil {
		t.Fatal(err)
	}
	if err := f.AddChunk("w2", "q1", 0, []Timeline{line(Payer, "acct-z", Entry{Target: "le-9", Interval: From(nov(t, 1))})}); err != nil {
		t.Fatal(err)
	}
	preview, err := f.Seal("w2", "q1", 1)
	if err != nil {
		t.Fatal(err)
	}
	number, err := f.Activate("w2", "q1", preview.Digest, 0)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := f.Timelines("w2", number, Payer, "", 2)
	mine, _ := f.Timelines("w1", first, Payer, "", 2)
	if len(other) != 1 || other[0].Member != "acct-z" || reflect.DeepEqual(other, mine[:1]) {
		t.Errorf("workspaces mixed %+v", other)
	}
}

func second[A any](_ A, err error) error { return err }
