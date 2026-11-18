package relationships

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
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

func (f fixture) publishIn(workspaceID, plan string, timelines ...Timeline) uint64 {
	f.t.Helper()
	active, _ := f.Active(workspaceID)
	if err := f.Draft(workspaceID, plan, active); err != nil {
		f.t.Fatal(err)
	}
	for start, index := 0, 0; start < len(timelines); start, index = start+f.limits.TimelinesPerChunk, index+1 {
		if err := f.AddChunk(workspaceID, plan, index, timelines[start:min(len(timelines), start+f.limits.TimelinesPerChunk)]); err != nil {
			f.t.Fatal(err)
		}
	}
	preview, err := f.Seal(workspaceID, plan, (len(timelines)+f.limits.TimelinesPerChunk-1)/f.limits.TimelinesPerChunk)
	if err != nil {
		f.t.Fatalf("seal %v %+v", err, preview.Violations)
	}
	number, err := f.Activate(workspaceID, plan, preview.Digest, active)
	if err != nil {
		f.t.Fatal(err)
	}
	return number
}

func TestMembersTargetsAndExactLookups(t *testing.T) {
	f := setup(t)
	open := func(target string) Entry { return Entry{Target: target, Interval: From(nov(t, 1))} }
	first := f.publishIn("w1", "p1",
		line(ViewGroup, "acct-c", open("g-2"), open("g-1")),
		line(ViewGroup, "acct-a", open("g-2")),
		line(Payer, "acct-b", open("le-1")),
		line(Hierarchy, "acct-a", open("unit-1")),
		line(Hierarchy, "unit-1", open("le-1")))
	f.publishIn("w2", "q1", line(ViewGroup, "acct-z", open("g-9")), line(Payer, "acct-y", open("le-9")))
	pages := func(read func(string) ([]string, error)) []string {
		out, cursor := []string{}, ""
		for {
			page, err := read(cursor)
			if err != nil {
				t.Fatal(err)
			}
			if len(page) == 0 {
				return out
			}
			out, cursor = append(out, page...), page[len(page)-1]
		}
	}
	if got := pages(func(c string) ([]string, error) { return f.Members("w1", first, c, 2) }); !reflect.DeepEqual(got, []string{"acct-a", "acct-b", "acct-c", "unit-1"}) {
		t.Errorf("members %v", got)
	}
	if got := pages(func(c string) ([]string, error) { return f.Targets("w1", first, ViewGroup, c, 1) }); !reflect.DeepEqual(got, []string{"g-1", "g-2"}) {
		t.Errorf("targets %v", got)
	}
	if got, _ := f.Targets("w1", first, Hierarchy, "le-1", 2); !reflect.DeepEqual(got, []string{"unit-1"}) {
		t.Errorf("targets after a cursor %v", got)
	}
	if got, _ := f.Members("w1", first, "acct-b0", 1); !reflect.DeepEqual(got, []string{"acct-c"}) {
		t.Errorf("a cursor between members %v", got)
	}
	if got, _ := f.Targets("w1", first, ContractPool, "", 2); len(got) != 0 || got == nil {
		t.Errorf("a kind without targets %#v", got)
	}
	if got, _ := f.Members("w2", 1, "", 2); !reflect.DeepEqual(got, []string{"acct-y", "acct-z"}) {
		t.Errorf("other workspace members %v", got)
	}
	if got, _ := f.Targets("w2", 1, ViewGroup, "", 2); !reflect.DeepEqual(got, []string{"g-9"}) {
		t.Errorf("other workspace targets %v", got)
	}
	if got, _ := f.Members("w1", 0, "", 2); len(got) != 0 {
		t.Errorf("revision 0 %v", got)
	}
	if got, _ := f.Members("w9", 0, "", 2); len(got) != 0 {
		t.Errorf("an unknown workspace at revision 0 %v", got)
	}
	page, _ := f.Members("w1", first, "", 2)
	page[0] = "changed"
	if again, _ := f.Members("w1", first, "", 2); again[0] != "acct-a" {
		t.Error("the member index is shared with the caller")
	}
	targets, _ := f.Targets("w1", first, ViewGroup, "", 2)
	targets[0] = "changed"
	if again, _ := f.Targets("w1", first, ViewGroup, "", 2); again[0] != "g-1" {
		t.Error("the target index is shared with the caller")
	}
	one, has, err := f.TimelineOf("w1", first, ViewGroup, "acct-c")
	if err != nil || !has || one.Kind != ViewGroup || one.Member != "acct-c" || len(one.Entries) != 2 {
		t.Fatalf("exact lookup %+v %v %v", one, has, err)
	}
	one.Entries[0].Target = "changed"
	if again, _, _ := f.TimelineOf("w1", first, ViewGroup, "acct-c"); again.Entries[0].Target == "changed" {
		t.Error("entries are shared with the registry")
	}
	if none, has, err := f.TimelineOf("w1", first, Payer, "acct-a"); err != nil || has || none.Member != "" || none.Entries != nil {
		t.Errorf("a member without that kind %+v %v %v", none, has, err)
	}
	if _, has, _ := f.TimelineOf("w2", 1, ViewGroup, "acct-c"); has {
		t.Error("a member of another workspace")
	}
	for name, want := range map[string]bool{"g-1": true, "g-9": false, "acct-a": false} {
		if got, err := f.IsTarget("w1", first, ViewGroup, name); err != nil || got != want {
			t.Errorf("is target %s %v %v", name, got, err)
		}
	}
	if got, _ := f.IsTarget("w1", first, Hierarchy, "zz"); got {
		t.Error("an ID after every target")
	}
	if got, _ := f.IsTarget("w2", 1, ViewGroup, "g-9"); !got {
		t.Error("the other workspace target")
	}
	if got, err := f.IsTarget("w9", 0, ViewGroup, "g-1"); err != nil || got {
		t.Errorf("an unknown workspace at revision 0 %v %v", got, err)
	}
	for name, err := range map[string]error{
		"targets of an unknown kind":   second(f.Targets("w1", first, Kind("other"), "", 1)),
		"targets with a zero limit":    second(f.Targets("w1", first, ViewGroup, "", 0)),
		"targets above the page":       second(f.Targets("w1", first, ViewGroup, "", 3)),
		"targets with a bad cursor":    second(f.Targets("w1", first, ViewGroup, "bad cursor", 1)),
		"targets of a bad workspace":   second(f.Targets("", first, ViewGroup, "", 1)),
		"members with a zero limit":    second(f.Members("w1", first, "", 0)),
		"members above the page":       second(f.Members("w1", first, "", 3)),
		"members with a bad cursor":    second(f.Members("w1", first, "bad cursor", 1)),
		"members of a bad workspace":   second(f.Members("", first, "", 1)),
		"a lookup of an unknown kind":  third(f.TimelineOf("w1", first, Kind("other"), "acct-a")),
		"a lookup of a bad member":     third(f.TimelineOf("w1", first, ViewGroup, "bad member")),
		"a lookup in a bad workspace":  third(f.TimelineOf("", first, ViewGroup, "acct-a")),
		"is target of an unknown kind": notTarget(f.IsTarget("w1", first, Kind("other"), "g-1")),
		"is target of a bad ID":        notTarget(f.IsTarget("w1", first, ViewGroup, "")),
		"is target in a bad workspace": notTarget(f.IsTarget("", first, ViewGroup, "g-1")),
	} {
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, err := range map[string]error{
		"targets of a future revision":    second(f.Targets("w1", first+1, ViewGroup, "", 1)),
		"members of a future revision":    second(f.Members("w1", first+1, "", 1)),
		"members of an unknown workspace": second(f.Members("w9", 1, "", 1)),
		"a lookup in a future revision":   third(f.TimelineOf("w1", first+1, ViewGroup, "acct-a")),
		"is target in a future revision":  notTarget(f.IsTarget("w1", first+1, ViewGroup, "g-1")),
	} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func notTarget(is bool, err error) error {
	if is {
		return errors.New("a failed check reported a target")
	}
	return err
}

func third[A any](_ A, has bool, err error) error {
	if has {
		return errors.New("a failed lookup reported a timeline")
	}
	return err
}

func TestIndexIsBuiltOncePerRevisionUnderConcurrentReaders(t *testing.T) {
	f := setupWith(t, Limits{Depth: 3, EntriesPerTimeline: 6, TimelinesPerChunk: 100, ChunksPerPlan: 5, TimelinesPerPlan: 500, EntriesPerPlan: 500, OpenPlans: 3, Violations: 4, PageSize: 100})
	lines := []Timeline{}
	for i := 0; i < 300; i++ {
		lines = append(lines, line(ViewGroup, fmt.Sprintf("acct-%03d", i), Entry{Target: fmt.Sprintf("g-%d", i%7), Interval: From(nov(t, 1))}))
	}
	first := f.publishIn("w1", "p1", lines...)
	var wait sync.WaitGroup
	results := make([][]string, 32)
	for reader := range results {
		wait.Add(1)
		go func() {
			defer wait.Done()
			members, _ := f.Members("w1", first, "acct-100", 3)
			targets, _ := f.Targets("w1", first, ViewGroup, "", 100)
			timelines, _ := f.Timelines("w1", first, ViewGroup, "acct-298", 5)
			empty, _ := f.Members("w9", 0, "", 5)
			fresh, _ := f.Targets("w1", 0, ViewGroup, "", 5)
			results[reader] = append(append(members, targets...), fmt.Sprint(len(timelines), len(empty), len(fresh)))
		}()
	}
	wait.Wait()
	want := []string{"acct-101", "acct-102", "acct-103", "g-0", "g-1", "g-2", "g-3", "g-4", "g-5", "g-6", "1 0 0"}
	for reader, got := range results {
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("reader %d %v", reader, got)
		}
	}
	later := f.publishIn("w1", "p2", line(ViewGroup, "acct-999", Entry{Target: "g-new", Interval: From(nov(t, 2))}))
	if got, _ := f.Targets("w1", later, ViewGroup, "g-6", 5); !reflect.DeepEqual(got, []string{"g-new"}) {
		t.Errorf("a later revision builds its own index %v", got)
	}
	if got, _ := f.Members("w1", later, "acct-299", 5); !reflect.DeepEqual(got, []string{"acct-999"}) {
		t.Errorf("a later revision lists its new member %v", got)
	}
	if got, _ := f.Targets("w1", first, ViewGroup, "g-6", 5); len(got) != 0 {
		t.Errorf("the pinned revision keeps its index %v", got)
	}
}

func benchmarkRegistry(b *testing.B, n int) (fixture, uint64) {
	f := setupWith(b, Limits{Depth: 3, EntriesPerTimeline: 6, TimelinesPerChunk: 1000, ChunksPerPlan: 100, TimelinesPerPlan: 100000, EntriesPerPlan: 100000, OpenPlans: 3, Violations: 4, PageSize: 100})
	lines := make([]Timeline, 0, n)
	for i := 0; i < n; i++ {
		lines = append(lines, line(ViewGroup, fmt.Sprintf("acct-%06d", i), Entry{Target: fmt.Sprintf("g-%05d", i%(n/10)), Interval: From(nov(b, 1))}))
	}
	return f, f.publishIn("w1", "bench", lines...)
}

func benchmarkPages(b *testing.B, read func(cursor string) ([]string, error)) {
	if _, err := read(""); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	cursor := ""
	for i := 0; i < b.N; i++ {
		page, err := read(cursor)
		if err != nil {
			b.Fatal(err)
		}
		cursor = ""
		if len(page) == 100 {
			cursor = page[len(page)-1]
		}
	}
}

func BenchmarkMembersPage(b *testing.B) {
	for _, n := range []int{10000, 100000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f, number := benchmarkRegistry(b, n)
			benchmarkPages(b, func(cursor string) ([]string, error) { return f.Members("w1", number, cursor, 100) })
		})
	}
}

func BenchmarkTargetsPage(b *testing.B) {
	for _, n := range []int{10000, 100000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f, number := benchmarkRegistry(b, n)
			benchmarkPages(b, func(cursor string) ([]string, error) { return f.Targets("w1", number, ViewGroup, cursor, 100) })
		})
	}
}

func TestReadsOfAMissingRevisionOrPlan(t *testing.T) {
	f := setup(t)
	d := nov(t, 1)
	for name, err := range map[string]error{
		"segments":   second(f.Segments("w9", 1, Payer, "acct-a", Between(d, nov(t, 2)))),
		"targets on": second(f.TargetsOn("w9", 1, ViewGroup, "acct-a", d)),
		"members on": second(f.MembersOn("w9", 1, ViewGroup, "g-1", d)),
		"chain":      second(f.Chain("w9", 1, "acct-a", d)),
		"union":      second(f.Union("w9", 1, []string{"g-1"}, d)),
	} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := f.Plan("w1", "bad plan"); !errors.Is(err, ErrInvalid) {
		t.Errorf("a bad plan ID %v", err)
	}
}
