package relationships

import (
	"errors"
	"reflect"
	"testing"
)

func TestMembersOnAfterWalksOneTargetInOrder(t *testing.T) {
	f := setup(t)
	nov1, nov10 := nov(t, 1), nov(t, 10)
	revision := f.publish("p1",
		line(Hierarchy, "a1", Entry{Target: "unit", Interval: From(nov1)}),
		line(Hierarchy, "a2", Entry{Target: "unit", Interval: Between(nov1, nov10)}, Entry{Target: "other", Interval: From(nov10)}),
		line(Hierarchy, "a3", Entry{Target: "unit", Interval: From(nov10)}),
		line(Payer, "a4", Entry{Target: "unit", Interval: From(nov1)}))
	for name, c := range map[string]struct {
		on            Day
		cursor        string
		limit, budget int
		want          MemberScan
	}{
		"all on a later day":      {nov(t, 20), "", 5, 9, MemberScan{[]string{"a1", "a3"}, "a3", 3, true}},
		"all on an earlier day":   {nov(t, 5), "", 5, 9, MemberScan{[]string{"a1", "a2"}, "a3", 3, true}},
		"stopped by the limit":    {nov(t, 20), "", 1, 9, MemberScan{[]string{"a1"}, "a1", 1, false}},
		"continued":               {nov(t, 20), "a1", 1, 9, MemberScan{[]string{"a3"}, "a3", 2, true}},
		"stopped by the budget":   {nov(t, 20), "a1", 1, 1, MemberScan{[]string{}, "a2", 1, false}},
		"no budget":               {nov(t, 20), "", 1, 0, MemberScan{[]string{}, "", 0, false}},
		"past the end":            {nov(t, 20), "a3", 1, 9, MemberScan{[]string{}, "a3", 0, true}},
		"before anyone belonged":  {day(t, 2026, 10, 1), "", 5, 9, MemberScan{[]string{}, "a3", 3, true}},
		"limit met at the end":    {nov(t, 5), "a1", 1, 9, MemberScan{[]string{"a2"}, "a2", 1, false}},
		"budget met at the end":   {nov(t, 20), "a2", 5, 1, MemberScan{[]string{"a3"}, "a3", 1, true}},
		"a cursor between people": {nov(t, 20), "a2x", 5, 9, MemberScan{[]string{"a3"}, "a3", 1, true}},
	} {
		got, err := f.MembersOnAfter("w1", revision, Hierarchy, "unit", c.on, c.cursor, c.limit, c.budget)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v %v; want %+v", name, got, err, c.want)
		}
	}
	if got, err := f.MembersOnAfter("w1", revision, Hierarchy, "nobody", nov(t, 20), "", 1, 9); err != nil || len(got.Members) != 0 || !got.Whole || got.Examined != 0 {
		t.Errorf("a target nobody has %+v %v", got, err)
	}
	if got, err := f.MembersOnAfter("w1", 0, Hierarchy, "unit", nov(t, 20), "", 1, 9); err != nil || len(got.Members) != 0 || !got.Whole {
		t.Errorf("the empty revision %+v %v", got, err)
	}
	if got, err := f.MembersOnAfter("w2", 0, Hierarchy, "unit", nov(t, 20), "", 5, 9); err != nil || len(got.Members) != 0 {
		t.Errorf("another workspace %+v %v", got, err)
	}
	for name, call := range map[string]func() (MemberScan, error){
		"an unknown kind": func() (MemberScan, error) { return f.MembersOnAfter("w1", revision, "other", "unit", nov1, "", 1, 1) },
		"a bad target":    func() (MemberScan, error) { return f.MembersOnAfter("w1", revision, Hierarchy, "a b", nov1, "", 1, 1) },
		"no day": func() (MemberScan, error) {
			return f.MembersOnAfter("w1", revision, Hierarchy, "unit", Day{}, "", 1, 1)
		},
		"a bad cursor": func() (MemberScan, error) {
			return f.MembersOnAfter("w1", revision, Hierarchy, "unit", nov1, "a b", 1, 1)
		},
		"no limit": func() (MemberScan, error) { return f.MembersOnAfter("w1", revision, Hierarchy, "unit", nov1, "", 0, 1) },
		"a negative budget": func() (MemberScan, error) {
			return f.MembersOnAfter("w1", revision, Hierarchy, "unit", nov1, "", 1, -1)
		},
	} {
		if got, err := call(); !errors.Is(err, ErrInvalid) || got.Members != nil {
			t.Errorf("%s: %+v %v", name, got, err)
		}
	}
	if _, err := f.MembersOnAfter("w1", revision+1, Hierarchy, "unit", nov1, "", 1, 1); err == nil {
		t.Error("a revision that does not exist")
	}
}
