package relationships

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestHierarchyCyclesAndDepth(t *testing.T) {
	a, b, c := nov(t, 1), nov(t, 10), nov(t, 20)
	open := func(member, parent string) Timeline { return line(Hierarchy, member, Entry{parent, From(a)}) }
	cycle := func(members ...string) []Violation {
		list := []Violation{}
		for _, member := range members {
			list = append(list, Violation{Hierarchy, member, RuleCycle})
		}
		return list
	}
	refusedCases := []struct {
		name      string
		timelines []Timeline
		want      []Violation
	}{
		{"a direct cycle", []Timeline{open("a", "b"), open("b", "a")}, cycle("a", "b")},
		{"an indirect cycle", []Timeline{open("a", "b"), open("b", "c"), open("c", "a")}, cycle("a", "b", "c")},
		{"a cycle that excludes the start member", []Timeline{open("x", "b"), open("b", "c"), open("c", "b")}, cycle("b", "c", "x")},
		{"a cycle on some days only", []Timeline{line(Hierarchy, "a", Entry{"b", Between(a, c)}), line(Hierarchy, "b", Entry{"a", From(b)})}, cycle("a", "b")},
		{"a cycle of one day", []Timeline{line(Hierarchy, "a", Entry{"b", Between(a, nov(t, 11))}), line(Hierarchy, "b", Entry{"a", From(b)})}, cycle("a", "b")},
		{"a chain of four ancestors", []Timeline{open("u0", "u1"), open("u1", "u2"), open("u2", "u3"), open("u3", "u4")}, []Violation{{Hierarchy, "u0", RuleDepth}}},
		{"a chain too deep on some days only", []Timeline{line(Hierarchy, "u0", Entry{"u1", Between(a, c)}), open("u1", "u2"), open("u2", "u3"), line(Hierarchy, "u3", Entry{"u4", From(b)})}, []Violation{{Hierarchy, "u0", RuleDepth}}},
	}
	for _, c := range refusedCases {
		f := setup(t)
		f.refused(c.name, "p", c.want, c.timelines...)
	}
	accepted := map[string][]Timeline{
		"opposite links on touching intervals":   {line(Hierarchy, "a", Entry{"b", Between(a, b)}), line(Hierarchy, "b", Entry{"a", From(b)})},
		"a chain of exactly three ancestors":     {open("u1", "u2"), open("u2", "u3"), open("u3", "u4")},
		"a deep chain whose links never coexist": {line(Hierarchy, "u0", Entry{"u1", Between(a, b)}), open("u1", "u2"), open("u2", "u3"), line(Hierarchy, "u3", Entry{"u4", From(b)})},
		"two parents on disjoint days":           {line(Hierarchy, "a", Entry{"b", Between(a, b)}, Entry{"c", From(b)}), open("b", "c")},
	}
	for name, timelines := range accepted {
		f := setup(t)
		if _, err := f.seal("p", timelines...); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	wide := testLimits
	wide.Violations = 10
	f := setupWith(t, wide)
	ring := []Timeline{open("r1", "r2"), open("r2", "r3"), open("r3", "r4")}
	f.publish("p1", ring...)
	f.refused("a ring of five closed by a later plan is reported as depth", "p2",
		[]Violation{{Hierarchy, "r1", RuleDepth}, {Hierarchy, "r2", RuleDepth}, {Hierarchy, "r3", RuleDepth}, {Hierarchy, "r4", RuleDepth}, {Hierarchy, "r5", RuleDepth}},
		open("r4", "r5"), open("r5", "r1"))
	f.refused("a violation names a member that the plan did not touch", "p3",
		[]Violation{{Hierarchy, "r1", RuleDepth}}, open("r4", "top"))
	busy := line(Hierarchy, "dept")
	for index, parent := range []string{"d1", "d2", "d1", "d2", "d1", "d2"} {
		busy.Entries = append(busy.Entries, Entry{parent, Between(nov(t, 1+4*index), nov(t, 5+4*index))})
	}
	f = setup(t)
	f.publish("p", busy, open("team", "dept"), open("d2", "top"))
	for on, want := range map[Day][]string{nov(t, 2): {"dept", "d1"}, nov(t, 6): {"dept", "d2", "top"}, nov(t, 24): {"dept", "d2", "top"}, nov(t, 25): {"dept"}} {
		if chain, _ := f.Chain("w1", 1, "team", on); !reflect.DeepEqual(chain, want) {
			t.Errorf("chain under a parent with many entries on %s: %v; want %v", on, chain, want)
		}
	}
	f.refused("a link above a parent with many entries that is too deep on its days only", "deep",
		[]Violation{{Hierarchy, "team", RuleDepth}}, open("top", "group"))
}

func TestPayerChainsOfOne(t *testing.T) {
	a, b := nov(t, 1), nov(t, 10)
	f := setup(t)
	if _, err := f.seal("p", line(Payer, "acct", Entry{"mid", Between(a, b)}), line(Payer, "mid", Entry{"top", From(b)})); err != nil {
		t.Errorf("a payer that gets a payer on the day the first link ends: %v", err)
	}
	f = setup(t)
	f.refused("a payer with a payer for one overlapping day", "p", []Violation{{Payer, "acct", RulePayerChain}},
		line(Payer, "acct", Entry{"mid", Between(a, b)}), line(Payer, "mid", Entry{"top", From(nov(t, 9))}))
	f = setup(t)
	f.refused("mutual payers", "p", []Violation{{Payer, "x", RulePayerChain}, {Payer, "y", RulePayerChain}},
		line(Payer, "x", Entry{"y", From(a)}), line(Payer, "y", Entry{"x", From(a)}))
	f = setup(t)
	f.publish("p1", line(Payer, "acct", Entry{"mid", From(a)}))
	f.refused("a later plan gives an existing payer a payer", "p2", []Violation{{Payer, "acct", RulePayerChain}}, line(Payer, "mid", Entry{"top", From(b)}))
	f.publish("p3", line(Payer, "acct", Entry{"mid", Between(a, b)}), line(Payer, "mid", Entry{"top", From(b)}))
	if got := f.target(2, Payer, "mid", nov(t, 12)); got != "top" {
		t.Errorf("payer of the former payer: %q", got)
	}
}

func TestMembersAndTargetsAreDisjointInThreeKinds(t *testing.T) {
	a := nov(t, 1)
	for _, kind := range []Kind{ContractPool, AllocationPool, ViewGroup} {
		f := setup(t)
		f.refused(string(kind)+": a target that is also a member, on other days", "p", []Violation{{kind, "pool", RuleRoles}},
			line(kind, "m", Entry{"pool", Between(a, nov(t, 5))}), line(kind, "pool", Entry{"outer", From(nov(t, 20))}))
		f = setup(t)
		f.publish("p1", line(kind, "m", Entry{"pool", From(a)}))
		f.refused(string(kind)+": a later plan makes an existing target a member", "p2", []Violation{{kind, "pool", RuleRoles}}, line(kind, "pool", Entry{"outer", From(a)}))
		f.refused(string(kind)+": a later plan makes an existing member a target", "p3", []Violation{{kind, "m", RuleRoles}}, line(kind, "other", Entry{"m", From(a)}))
	}
	f := setup(t)
	if _, err := f.seal("p",
		line(ContractPool, "x", Entry{"y", From(a)}), line(AllocationPool, "y", Entry{"x", From(a)}),
		line(ViewGroup, "y", Entry{"x", From(a)}), line(Hierarchy, "x", Entry{"y", From(a)})); err != nil {
		t.Errorf("one identifier with different roles in different kinds: %v", err)
	}
	f = setup(t)
	if _, err := f.seal("p", line(Hierarchy, "team", Entry{"dept", From(a)}), line(Hierarchy, "dept", Entry{"division", From(a)})); err != nil {
		t.Errorf("a unit that is both child and parent: %v", err)
	}
}

func TestViolationsAreSortedAndBounded(t *testing.T) {
	a := nov(t, 1)
	limits := testLimits
	limits.TimelinesPerChunk, limits.TimelinesPerPlan = 8, 8
	f := setupWith(t, limits)
	preview, err := f.seal("p",
		line(Payer, "y", Entry{"x", From(a)}), line(Payer, "x", Entry{"y", From(a)}),
		line(Hierarchy, "b", Entry{"a", From(a)}), line(Hierarchy, "a", Entry{"b", From(a)}),
		line(ContractPool, "m", Entry{"pool", From(a)}), line(ContractPool, "pool", Entry{"outer", From(a)}))
	want := []Violation{{ContractPool, "pool", RuleRoles}, {Hierarchy, "a", RuleCycle}, {Hierarchy, "b", RuleCycle}, {Payer, "x", RulePayerChain}}
	if !errors.Is(err, ErrRejected) || !reflect.DeepEqual(preview.Violations, want) || !preview.Truncated {
		t.Errorf("violations %+v truncated %v error %v", preview.Violations, preview.Truncated, err)
	}
	if info, _ := f.Plan("w1", "p"); !info.Truncated || len(info.Violations) != 4 {
		t.Errorf("the stored violations %+v", info)
	}
	f = setupWith(t, limits)
	preview, _ = f.seal("p", line(Payer, "y", Entry{"x", From(a)}), line(Payer, "x", Entry{"y", From(a)}),
		line(Hierarchy, "b", Entry{"a", From(a)}), line(Hierarchy, "a", Entry{"b", From(a)}))
	if preview.Truncated || len(preview.Violations) != 4 {
		t.Errorf("violations exactly at the bound %+v truncated %v", preview.Violations, preview.Truncated)
	}
}

func TestActivationIsAtomicForReaders(t *testing.T) {
	limits := Limits{Depth: 3, EntriesPerTimeline: 4, TimelinesPerChunk: 64, ChunksPerPlan: 4, TimelinesPerPlan: 64, EntriesPerPlan: 256, OpenPlans: 4, Violations: 8, PageSize: 8}
	f := setupWith(t, limits)
	const members, rounds, most = 40, 60, 5000
	var superseded atomic.Int64
	manifest := func(pool string) []Timeline {
		timelines := []Timeline{}
		for index := 0; index < members; index++ {
			timelines = append(timelines, line(AllocationPool, fmt.Sprintf("m%02d", index), Entry{pool, From(nov(t, 1))}))
		}
		return timelines
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	seen := make([]map[uint64]int, 6)
	for reader := range seen {
		seen[reader] = map[uint64]int{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				number, _ := f.Active("w1")
				inA, errA := f.MembersOn("w1", number, AllocationPool, "poolA", nov(t, 5))
				inB, errB := f.MembersOn("w1", number, AllocationPool, "poolB", nov(t, 5))
				wantA, wantB := 0, 0
				if number%2 == 1 {
					wantA = members
				} else if number > 0 {
					wantB = members
				}
				if errA != nil || errB != nil || len(inA) != wantA || len(inB) != wantB {
					t.Errorf("revision %d read as %d and %d members (%v %v)", number, len(inA), len(inB), errA, errB)
					return
				}
				seen[reader][number]++
				if later, _ := f.Active("w1"); later != number {
					superseded.Add(1)
				}
			}
		}()
	}
	for round := 1; round <= rounds || (superseded.Load() == 0 && round <= most); round++ {
		pool := "poolA"
		if round%2 == 0 {
			pool = "poolB"
		}
		if got := f.publish(fmt.Sprintf("p%d", round), manifest(pool)...); got != uint64(round) {
			t.Fatalf("revision %d in round %d", got, round)
		}
	}
	close(stop)
	wg.Wait()
	distinct := map[uint64]struct{}{}
	for _, reader := range seen {
		for number := range reader {
			distinct[number] = struct{}{}
		}
	}
	if len(distinct) < 2 || superseded.Load() == 0 {
		t.Errorf("readers observed %d distinct revisions and %d reads that an activation overtook", len(distinct), superseded.Load())
	}
	base, _ := f.Active("w1")
	digests := map[string][32]byte{}
	for _, plan := range []string{"left", "right"} {
		preview, err := func() (Preview, error) {
			if err := f.Draft("w1", plan, base); err != nil {
				return Preview{}, err
			}
			if err := f.AddChunk("w1", plan, 0, []Timeline{line(Payer, plan, Entry{"master", From(nov(t, 1))})}); err != nil {
				return Preview{}, err
			}
			return f.Seal("w1", plan, 1)
		}()
		if err != nil {
			t.Fatal(err)
		}
		digests[plan] = preview.Digest
	}
	results := make(chan error, 2)
	for plan, digest := range digests {
		go func() {
			_, err := f.Activate("w1", plan, digest, base)
			results <- err
		}()
	}
	first, second := <-results, <-results
	if (first == nil) == (second == nil) || (first != nil && !errors.Is(first, ErrConflict)) || (second != nil && !errors.Is(second, ErrConflict)) {
		t.Errorf("two plans on one base: %v and %v", first, second)
	}
	if active, _ := f.Active("w1"); active != base+1 {
		t.Errorf("active revision %d after two competing activations on %d", active, base)
	}
}

func TestScaleOfOneHundredThousandTimelines(t *testing.T) {
	if testing.Short() {
		t.Skip("scale")
	}
	const total, perChunk = 100000, 5000
	limits := Limits{Depth: 4, EntriesPerTimeline: 4, TimelinesPerChunk: perChunk, ChunksPerPlan: total / perChunk, TimelinesPerPlan: total, EntriesPerPlan: 2 * total, OpenPlans: 2, Violations: 10, PageSize: 100}
	f := setupWith(t, limits)
	build := func(plan string, moved int) {
		active, _ := f.Active("w1")
		if err := f.Draft("w1", plan, active); err != nil {
			t.Fatal(err)
		}
		for index := 0; index < total/perChunk; index++ {
			timelines := make([]Timeline, 0, perChunk)
			for member := index * perChunk; member < (index+1)*perChunk; member++ {
				pool := fmt.Sprintf("pool%03d", member%500)
				entries := []Entry{{pool, From(nov(t, 1))}}
				if member < moved {
					entries = []Entry{{pool, Between(nov(t, 1), nov(t, 15))}, {fmt.Sprintf("pool%03d", (member+1)%500), From(nov(t, 15))}}
				}
				timelines = append(timelines, line(AllocationPool, fmt.Sprintf("acct%06d", member), entries...))
			}
			if err := f.AddChunk("w1", plan, index, timelines); err != nil {
				t.Fatal(err)
			}
		}
		preview, err := f.Seal("w1", plan, total/perChunk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Activate("w1", plan, preview.Digest, active); err != nil {
			t.Fatal(err)
		}
	}
	build("load", 0)
	build("move", total)
	if members, _ := f.MembersOn("w1", 1, AllocationPool, "pool007", nov(t, 20)); len(members) != total/500 {
		t.Errorf("pool size at the first revision %d", len(members))
	}
	if got := f.target(2, AllocationPool, "acct000007", nov(t, 20)); got != "pool008" {
		t.Errorf("a moved account at the second revision: %q", got)
	}
	if got := f.target(1, AllocationPool, "acct000007", nov(t, 20)); got != "pool007" {
		t.Errorf("a moved account at the first revision: %q", got)
	}
	info, _ := f.Plan("w1", "move")
	if summary := info.Kinds[AllocationPool]; summary.Changed != total || summary.EntriesAfter != 2*total || summary.Earliest != nov(t, 15) {
		t.Errorf("summary of the move %+v", summary)
	}
}
