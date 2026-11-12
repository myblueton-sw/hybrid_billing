package relationships

import (
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	*Registry
	t   testing.TB
	now *time.Time
}

var testLimits = Limits{Depth: 3, EntriesPerTimeline: 6, TimelinesPerChunk: 4, ChunksPerPlan: 5, TimelinesPerPlan: 8, EntriesPerPlan: 20, OpenPlans: 3, Violations: 4, PageSize: 2}

func setupWith(t testing.TB, limits Limits) fixture {
	t.Helper()
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	r, err := NewRegistry(limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return fixture{r, t, &now}
}

func setup(t testing.TB) fixture { return setupWith(t, testLimits) }

func day(t testing.TB, year, month, number int) Day {
	t.Helper()
	d, err := NewDay(year, month, number)
	if err != nil {
		t.Fatalf("day %d-%d-%d: %v", year, month, number, err)
	}
	return d
}

func nov(t testing.TB, number int) Day { return day(t, 2026, 11, number) }

func line(kind Kind, member string, entries ...Entry) Timeline {
	return Timeline{Kind: kind, Member: member, Entries: entries}
}

func (f fixture) seal(plan string, timelines ...Timeline) (Preview, error) {
	f.t.Helper()
	active, err := f.Active("w1")
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.Draft("w1", plan, active); err != nil {
		f.t.Fatalf("draft %s: %v", plan, err)
	}
	if err := f.AddChunk("w1", plan, 0, timelines); err != nil {
		f.t.Fatalf("chunk of %s: %v", plan, err)
	}
	return f.Seal("w1", plan, 1)
}

func (f fixture) publish(plan string, timelines ...Timeline) uint64 {
	f.t.Helper()
	preview, err := f.seal(plan, timelines...)
	if err != nil {
		f.t.Fatalf("seal %s: %v %+v", plan, err, preview.Violations)
	}
	active, _ := f.Active("w1")
	number, err := f.Activate("w1", plan, preview.Digest, active)
	if err != nil {
		f.t.Fatalf("activate %s: %v", plan, err)
	}
	return number
}

func (f fixture) refused(name, plan string, want []Violation, timelines ...Timeline) {
	f.t.Helper()
	preview, err := f.seal(plan, timelines...)
	if !errors.Is(err, ErrRejected) {
		f.t.Errorf("%s: error %v; want rejection", name, err)
		return
	}
	if !reflect.DeepEqual(preview.Violations, want) {
		f.t.Errorf("%s: violations %+v; want %+v", name, preview.Violations, want)
	}
	if preview.Truncated {
		f.t.Errorf("%s: the violation list is truncated", name)
	}
	if info, err := f.Plan("w1", plan); err != nil || info.State != PlanRejected || !reflect.DeepEqual(info.Violations, want) || info.Truncated {
		f.t.Errorf("%s: plan read-back %+v %v", name, info, err)
	}
}

func (f fixture) target(revision uint64, kind Kind, member string, on Day) string {
	f.t.Helper()
	target, has, err := f.TargetOn("w1", revision, kind, member, on)
	if err != nil {
		f.t.Fatalf("target of %s: %v", member, err)
	}
	if has != (target != "") {
		f.t.Fatalf("target %q with flag %v", target, has)
	}
	return target
}

func TestDaysAndIntervals(t *testing.T) {
	for name, date := range map[string][3]int{
		"30 February": {2026, 2, 30}, "29 February of a common year": {2026, 2, 29}, "month 13": {2026, 13, 1},
		"day 0": {2026, 1, 0}, "day 32": {2026, 1, 32}, "year 0": {0, 1, 1}, "year 10000": {10000, 1, 1}, "31 April": {2026, 4, 31},
	} {
		if _, err := NewDay(date[0], date[1], date[2]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if got := day(t, 2028, 2, 29).String(); got != "20280229" {
		t.Errorf("leap day %s", got)
	}
	if got := day(t, 1, 1, 1).String(); got != "00010101" {
		t.Errorf("first day %s", got)
	}
	if !nov(t, 1).Before(nov(t, 2)) || nov(t, 2).Before(nov(t, 2)) || !day(t, 2026, 10, 31).Before(nov(t, 1)) {
		t.Errorf("day order")
	}
	f := setup(t)
	if err := f.Draft("w1", "p", 0); err != nil {
		t.Fatal(err)
	}
	for name, interval := range map[string]Interval{
		"zero value": {}, "empty": Between(nov(t, 5), nov(t, 5)), "inverted": Between(nov(t, 6), nov(t, 5)),
		"zero start": Between(Day{}, nov(t, 5)), "zero end": Between(nov(t, 5), Day{}), "open from zero": From(Day{}),
	} {
		err := f.AddChunk("w1", "p", 0, []Timeline{line(ContractPool, "m", Entry{"pool", interval})})
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s interval: error %v", name, err)
		}
	}
	start, end := Between(nov(t, 3), nov(t, 9)).Start(), Day{}
	if start != nov(t, 3) {
		t.Errorf("start %v", start)
	}
	if end, bounded := Between(nov(t, 3), nov(t, 9)).End(); !bounded || end != nov(t, 9) {
		t.Errorf("end %v %v", end, bounded)
	}
	if got, bounded := From(nov(t, 3)).End(); bounded || got != end {
		t.Errorf("an open interval reports an end: %v %v", got, bounded)
	}
}

func TestIdentifierRule(t *testing.T) {
	for id, want := range map[string]bool{
		"a": true, "A.b_c:d-9": true, strings.Repeat("x", 128): true, strings.Repeat("x", 129): false,
		"": false, "a b": false, "a/b": false, "한": false, "a\x00": false, "a*": false,
	} {
		if validID(id) != want {
			t.Errorf("identifier %q valid = %v", id, !want)
		}
	}
	f := setup(t)
	if err := f.Draft("bad id", "p", 0); !errors.Is(err, ErrInvalid) {
		t.Errorf("workspace identifier error = %v", err)
	}
	if err := f.Draft("w1", "bad id", 0); !errors.Is(err, ErrInvalid) {
		t.Errorf("plan identifier error = %v", err)
	}
}

func TestLimitsAreRequired(t *testing.T) {
	fields := reflect.TypeOf(testLimits).NumField()
	for index := 0; index < fields; index++ {
		limits := testLimits
		reflect.ValueOf(&limits).Elem().Field(index).SetInt(0)
		if _, err := NewRegistry(limits, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("limit %s zero: error %v", reflect.TypeOf(limits).Field(index).Name, err)
		}
	}
	deep := testLimits
	deep.Depth = MaxDepth
	if _, err := NewRegistry(deep, time.Now); err != nil {
		t.Errorf("depth at the package maximum: %v", err)
	}
	deep.Depth = MaxDepth + 1
	if _, err := NewRegistry(deep, time.Now); !errors.Is(err, ErrInvalid) {
		t.Errorf("depth above the package maximum error = %v", err)
	}
	if _, err := NewRegistry(testLimits, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("missing clock error = %v", err)
	}
}

func TestTimelineRulesAtAddChunk(t *testing.T) {
	f := setup(t)
	if err := f.Draft("w1", "p", 0); err != nil {
		t.Fatal(err)
	}
	a, b, c := nov(t, 1), nov(t, 10), nov(t, 20)
	cases := map[string]struct {
		timeline Timeline
		want     error
	}{
		"exclusive kind, two targets overlapping":       {line(ContractPool, "m", Entry{"p1", Between(a, c)}, Entry{"p2", Between(b, c)}), ErrInvalid},
		"exclusive kind, overlap of one day":            {line(Payer, "m", Entry{"p1", Between(a, nov(t, 11))}, Entry{"p2", From(b)}), ErrInvalid},
		"exclusive kind, open entry before a later one": {line(Hierarchy, "m", Entry{"p1", From(a)}, Entry{"p2", From(b)}), ErrInvalid},
		"exclusive kind, different targets touching":    {line(ContractPool, "m", Entry{"p1", Between(a, b)}, Entry{"p2", From(b)}), nil},
		"exclusive kind, same target touching":          {line(AllocationPool, "m", Entry{"p1", Between(a, b)}, Entry{"p1", From(b)}), ErrInvalid},
		"exclusive kind, same target with a gap":        {line(AllocationPool, "m2", Entry{"p1", Between(a, b)}, Entry{"p1", From(c)}), nil},
		"view group, two groups overlapping":            {line(ViewGroup, "m", Entry{"g1", Between(a, c)}, Entry{"g2", From(b)}), nil},
		"view group, same group overlapping":            {line(ViewGroup, "m3", Entry{"g1", Between(a, c)}, Entry{"g1", From(b)}), ErrInvalid},
		"view group, same group touching":               {line(ViewGroup, "m3", Entry{"g1", Between(a, b)}, Entry{"g2", From(a)}, Entry{"g1", From(b)}), ErrInvalid},
		"view group, same group nested":                 {line(ViewGroup, "m3", Entry{"g1", From(a)}, Entry{"g2", From(a)}, Entry{"g1", Between(b, c)}), ErrInvalid},
		"member equal to target":                        {line(Hierarchy, "m", Entry{"m", From(a)}), ErrInvalid},
		"unknown kind":                                  {line("cost_center", "m", Entry{"x", From(a)}), ErrInvalid},
		"invalid member":                                {line(Hierarchy, "m m", Entry{"x", From(a)}), ErrInvalid},
		"invalid target":                                {line(Hierarchy, "m", Entry{"", From(a)}), ErrInvalid},
		"entries above the limit":                       {line(ViewGroup, "m4", Entry{"g1", From(a)}, Entry{"g2", From(a)}, Entry{"g3", From(a)}, Entry{"g4", From(a)}, Entry{"g5", From(a)}, Entry{"g6", From(a)}, Entry{"g7", From(a)}), ErrLimit},
	}
	index := 0
	for name, c := range cases {
		err := f.AddChunk("w1", "p", index%5, []Timeline{c.timeline})
		if c.want == nil && err != nil || c.want != nil && !errors.Is(err, c.want) {
			t.Errorf("%s: error %v; want %v", name, err, c.want)
		}
		if err == nil {
			index++
		}
	}
	twice := []Timeline{line(Payer, "x", Entry{"y", From(a)}), line(Payer, "x")}
	if err := f.AddChunk("w1", "p", 4, twice); !errors.Is(err, ErrInvalid) {
		t.Errorf("a member twice inside a chunk: error %v", err)
	}
	many := []Timeline{line(Payer, "a1"), line(Payer, "a2"), line(Payer, "a3"), line(Payer, "a4"), line(Payer, "a5")}
	if err := f.AddChunk("w1", "p", 4, many); !errors.Is(err, ErrLimit) {
		t.Errorf("timelines above the chunk limit: error %v", err)
	}
	replaceable := []Timeline{line(Payer, "last", Entry{"p1", From(a)}, Entry{"p2", From(b)})}
	if err := f.AddChunk("w1", "p", 4, replaceable); !errors.Is(err, ErrInvalid) {
		t.Fatalf("a chunk with an overlap at the last index: %v", err)
	}
	replaceable[0].Entries[0].Interval = Between(a, b)
	if err := f.AddChunk("w1", "p", 4, replaceable); err != nil {
		t.Errorf("the corrected chunk at the index that was refused: %v", err)
	}
	if err := f.AddChunk("w1", "p", 5, nil); !errors.Is(err, ErrLimit) {
		t.Errorf("chunk index at the limit: error %v", err)
	}
	if err := f.AddChunk("w1", "p", -1, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("negative chunk index: error %v", err)
	}
	if info, err := f.Plan("w1", "p"); err != nil || info.State != PlanDraft {
		t.Errorf("refused chunks leave the plan in draft: %+v %v", info, err)
	}
}

func TestMidMonthMoveSegmentsAndGaps(t *testing.T) {
	f := setup(t)
	first := f.publish("p1", line(ContractPool, "acct", Entry{"poolA", Between(nov(t, 1), nov(t, 15))}, Entry{"poolB", From(nov(t, 15))}),
		line(ContractPool, "late", Entry{"poolA", Between(nov(t, 10), nov(t, 20))}))
	if first != 1 {
		t.Fatalf("first revision %d", first)
	}
	for on, want := range map[Day]string{day(t, 2026, 10, 31): "", nov(t, 1): "poolA", nov(t, 14): "poolA", nov(t, 15): "poolB", nov(t, 16): "poolB", day(t, 2030, 1, 1): "poolB"} {
		if got := f.target(1, ContractPool, "acct", on); got != want {
			t.Errorf("target on %s = %q; want %q", on, got, want)
		}
	}
	month := Between(nov(t, 1), day(t, 2026, 12, 1))
	segments, err := f.Segments("w1", 1, ContractPool, "acct", month)
	want := []Segment{{Between(nov(t, 1), nov(t, 15)), "poolA", true}, {Between(nov(t, 15), day(t, 2026, 12, 1)), "poolB", true}}
	if err != nil || !reflect.DeepEqual(segments, want) {
		t.Errorf("segments of the move month %+v %v", segments, err)
	}
	segments, err = f.Segments("w1", 1, ContractPool, "late", month)
	want = []Segment{{Between(nov(t, 1), nov(t, 10)), "", false}, {Between(nov(t, 10), nov(t, 20)), "poolA", true}, {Between(nov(t, 20), day(t, 2026, 12, 1)), "", false}}
	if err != nil || !reflect.DeepEqual(segments, want) {
		t.Errorf("segments with gaps %+v %v", segments, err)
	}
	segments, err = f.Segments("w1", 1, ContractPool, "nobody", month)
	if err != nil || !reflect.DeepEqual(segments, []Segment{{month, "", false}}) {
		t.Errorf("segments of an unknown member %+v %v", segments, err)
	}
	segments, err = f.Segments("w1", 1, ContractPool, "acct", Between(nov(t, 20), nov(t, 25)))
	if err != nil || !reflect.DeepEqual(segments, []Segment{{Between(nov(t, 20), nov(t, 25)), "poolB", true}}) {
		t.Errorf("an open-ended entry clipped to the query %+v %v", segments, err)
	}
	if _, err := f.Segments("w1", 1, ContractPool, "acct", From(nov(t, 1))); !errors.Is(err, ErrInvalid) {
		t.Errorf("open-ended query error = %v", err)
	}
	if _, err := f.Segments("w1", 1, ContractPool, "acct", Interval{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("zero query interval error = %v", err)
	}
	members, err := f.MembersOn("w1", 1, ContractPool, "poolA", nov(t, 12))
	if err != nil || !reflect.DeepEqual(members, []string{"acct", "late"}) {
		t.Errorf("members of pool A on the 12th %v %v", members, err)
	}
	members, _ = f.MembersOn("w1", 1, ContractPool, "poolA", nov(t, 15))
	if !reflect.DeepEqual(members, []string{"late"}) {
		t.Errorf("members of pool A on the move day %v", members)
	}
	members, _ = f.MembersOn("w1", 1, ContractPool, "poolA", nov(t, 20))
	if len(members) != 0 || members == nil {
		t.Errorf("members of pool A after everyone left %v", members)
	}
	if other, _ := f.MembersOn("w1", 1, AllocationPool, "poolA", nov(t, 12)); len(other) != 0 {
		t.Errorf("another kind reads this kind's members %v", other)
	}
}

func TestRetroactiveChangeKeepsPinnedAnswers(t *testing.T) {
	f := setup(t)
	f.publish("p1", line(ContractPool, "acct", Entry{"poolA", Between(nov(t, 1), nov(t, 15))}, Entry{"poolB", From(nov(t, 15))}),
		line(ContractPool, "other", Entry{"poolA", From(nov(t, 1))}))
	preview, err := f.seal("p2", line(ContractPool, "acct", Entry{"poolA", Between(nov(t, 1), nov(t, 8))}, Entry{"poolB", From(nov(t, 8))}),
		line(ContractPool, "other", Entry{"poolA", From(nov(t, 1))}))
	if err != nil {
		t.Fatal(err)
	}
	want := KindSummary{Changed: 1, EntriesBefore: 2, EntriesAfter: 2, Earliest: nov(t, 8), HasEarliest: true}
	if len(preview.Kinds) != 1 || preview.Kinds[ContractPool] != want {
		t.Errorf("preview %+v", preview.Kinds)
	}
	changes, err := f.ChangedMembers("w1", "p2", ContractPool, "", 2)
	wantChanges := []MemberChange{{Member: "acct", Differs: []Interval{Between(nov(t, 8), nov(t, 15))}, Before: []string{"poolA", "poolB"}, After: []string{"poolA", "poolB"}}}
	if err != nil || !reflect.DeepEqual(changes, wantChanges) {
		t.Errorf("changed members %+v %v", changes, err)
	}
	targets, err := f.ChangedTargets("w1", "p2", ContractPool, "", 2)
	if err != nil || !reflect.DeepEqual(targets, []string{"poolA", "poolB"}) {
		t.Errorf("changed targets %v %v", targets, err)
	}
	if got := f.target(1, ContractPool, "acct", nov(t, 10)); got != "poolA" {
		t.Errorf("a sealed plan changed the active answer: %q", got)
	}
	*f.now = f.now.Add(time.Hour)
	second, err := f.Activate("w1", "p2", preview.Digest, 1)
	if err != nil || second != 2 {
		t.Fatalf("activation %d %v", second, err)
	}
	if got := f.target(1, ContractPool, "acct", nov(t, 10)); got != "poolA" {
		t.Errorf("the pinned revision changed its answer: %q", got)
	}
	if got := f.target(2, ContractPool, "acct", nov(t, 10)); got != "poolB" {
		t.Errorf("the new revision answers %q", got)
	}
	old, _ := f.Revision("w1", 1)
	current, _ := f.Revision("w1", 2)
	if old.Plan != "p1" || current.Plan != "p2" || current.Digest != preview.Digest || !current.Recorded.Equal(old.Recorded.Add(time.Hour)) || current.Number != 2 {
		t.Errorf("revision records %+v %+v", old, current)
	}
	if info, err := f.Plan("w1", "p2"); err != nil || info.State != PlanActivated || info.Revision != 2 || info.Digest != preview.Digest || info.Kinds[ContractPool] != want {
		t.Errorf("plan read-back after activation %+v %v", info, err)
	}
	if changes, err := f.ChangedMembers("w1", "p2", ContractPool, "", 2); err != nil || len(changes) != 1 {
		t.Errorf("changed members are readable after activation: %v %v", changes, err)
	}
}

func TestViewGroupUnionAndRelease(t *testing.T) {
	f := setup(t)
	f.publish("p1",
		line(ViewGroup, "src1", Entry{"g1", From(nov(t, 1))}, Entry{"g2", From(nov(t, 1))}),
		line(ViewGroup, "src2", Entry{"g1", From(nov(t, 1))}),
		line(ViewGroup, "alloc1", Entry{"g2", Between(nov(t, 1), nov(t, 20))}))
	union, err := f.Union("w1", 1, []string{"g1", "g2", "g1"}, nov(t, 5))
	if err != nil || !reflect.DeepEqual(union, []string{"alloc1", "src1", "src2"}) {
		t.Errorf("union %v %v", union, err)
	}
	targets, _ := f.TargetsOn("w1", 1, ViewGroup, "src1", nov(t, 5))
	if !reflect.DeepEqual(targets, []string{"g1", "g2"}) {
		t.Errorf("groups of a member %v", targets)
	}
	f.publish("p2", line(ViewGroup, "src1", Entry{"g1", Between(nov(t, 1), nov(t, 10))}))
	for name, c := range map[string]struct {
		revision uint64
		groups   []string
		on       Day
		want     []string
	}{
		"before the release at the new revision": {2, []string{"g1"}, nov(t, 9), []string{"src1", "src2"}},
		"on the release day":                     {2, []string{"g1"}, nov(t, 10), []string{"src2"}},
		"the other group lost the member at all": {2, []string{"g2"}, nov(t, 5), []string{"alloc1"}},
		"the old revision still has the member":  {1, []string{"g2"}, nov(t, 5), []string{"alloc1", "src1"}},
		"no group":                               {2, nil, nov(t, 5), []string{}},
		"an unknown group":                       {2, []string{"ghost"}, nov(t, 5), []string{}},
	} {
		union, err := f.Union("w1", c.revision, c.groups, c.on)
		if err != nil || !reflect.DeepEqual(union, c.want) {
			t.Errorf("%s: %v %v", name, union, err)
		}
	}
	f.publish("p3", line(ViewGroup, "src2"))
	if members, _ := f.MembersOn("w1", 3, ViewGroup, "g1", nov(t, 5)); !reflect.DeepEqual(members, []string{"src1"}) {
		t.Errorf("members after an empty timeline removed one: %v", members)
	}
	f.publish("p4", line(ViewGroup, "g9", Entry{"src2", From(nov(t, 1))}))
	if members, _ := f.MembersOn("w1", 4, ViewGroup, "src2", nov(t, 5)); !reflect.DeepEqual(members, []string{"g9"}) {
		t.Errorf("a removed member can become a group: %v", members)
	}
}

func TestConcurrentPlansAndActivationChecks(t *testing.T) {
	f := setup(t)
	one := []Timeline{line(Payer, "a", Entry{"p", From(nov(t, 1))})}
	two := []Timeline{line(Payer, "b", Entry{"p", From(nov(t, 1))})}
	for _, plan := range []string{"first", "second", "third"} {
		if err := f.Draft("w1", plan, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Draft("w1", "fourth", 0); !errors.Is(err, ErrLimit) {
		t.Errorf("open plans above the limit: error %v", err)
	}
	for plan, timelines := range map[string][]Timeline{"first": one, "second": two} {
		if err := f.AddChunk("w1", plan, 0, timelines); err != nil {
			t.Fatal(err)
		}
	}
	first, err := f.Seal("w1", "first", 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.Seal("w1", "second", 1)
	if err != nil {
		t.Fatal(err)
	}
	var wrong [32]byte
	for name, c := range map[string]struct {
		digest   [32]byte
		expected uint64
	}{"a wrong digest": {wrong, 0}, "another plan's digest": {second.Digest, 0}, "a wrong expected revision": {first.Digest, 1}} {
		if _, err := f.Activate("w1", "first", c.digest, c.expected); !errors.Is(err, ErrConflict) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if active, _ := f.Active("w1"); active != 0 {
		t.Fatalf("failed activations changed the active revision to %d", active)
	}
	if number, err := f.Activate("w1", "first", first.Digest, 0); err != nil || number != 1 {
		t.Fatalf("activation %d %v", number, err)
	}
	if _, err := f.Activate("w1", "second", second.Digest, 0); !errors.Is(err, ErrConflict) {
		t.Errorf("the stale plan activated: %v", err)
	}
	if _, err := f.Activate("w1", "second", second.Digest, 1); !errors.Is(err, ErrConflict) {
		t.Errorf("the stale plan activated with the new revision number: %v", err)
	}
	if got := f.target(1, Payer, "b", nov(t, 5)); got != "" {
		t.Errorf("the stale plan changed something: %q", got)
	}
	if _, err := f.Activate("w1", "first", first.Digest, 0); !errors.Is(err, ErrState) {
		t.Errorf("activating twice: %v", err)
	}
	if err := f.AddChunk("w1", "third", 0, two); err != nil {
		t.Errorf("a draft on a stale base still accepts chunks: %v", err)
	}
	if _, err := f.Seal("w1", "third", 1); !errors.Is(err, ErrConflict) {
		t.Errorf("sealing on a stale base: %v", err)
	}
	if info, _ := f.Plan("w1", "third"); info.State != PlanDraft || !reflect.DeepEqual(info.Chunks, []int{0}) {
		t.Errorf("a stale seal changed the plan: %+v", info)
	}
	if err := f.Draft("w1", "filler", 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Draft("w1", "fourth", 1); !errors.Is(err, ErrLimit) {
		t.Errorf("stale plans still count as open: %v", err)
	}
	for _, plan := range []string{"second", "third"} {
		if err := f.Discard("w1", plan); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Draft("w1", "fourth", 0); !errors.Is(err, ErrConflict) {
		t.Errorf("a draft on an old base: %v", err)
	}
	if err := f.Draft("w1", "fourth", 1); err != nil {
		t.Fatal(err)
	}
	if err := f.AddChunk("w1", "fourth", 0, two); err != nil {
		t.Fatal(err)
	}
	fourth, err := f.Seal("w1", "fourth", 1)
	if err != nil {
		t.Fatal(err)
	}
	if fourth.Digest == second.Digest {
		t.Errorf("the same timelines on another base have the same digest")
	}
	if number, err := f.Activate("w1", "fourth", fourth.Digest, 1); err != nil || number != 2 {
		t.Errorf("revision numbers have no gaps after failures: %d %v", number, err)
	}
}

func TestPlanLifeCycle(t *testing.T) {
	f := setup(t)
	a := nov(t, 1)
	chunk0 := []Timeline{line(Payer, "a", Entry{"p", From(a)}), line(Payer, "b", Entry{"p", From(a)})}
	reordered := []Timeline{line(Payer, "b", Entry{"p", From(a)}), line(Payer, "a", Entry{"p", From(a)})}
	chunk2 := []Timeline{line(ViewGroup, "a", Entry{"g2", From(a)}, Entry{"g1", From(a)})}
	if err := f.Draft("w1", "p", 0); err != nil {
		t.Fatal(err)
	}
	if err := f.Draft("w1", "p", 0); !errors.Is(err, ErrConflict) {
		t.Errorf("drafting an existing plan: %v", err)
	}
	if err := f.AddChunk("w1", "p", 0, chunk0); err != nil {
		t.Fatal(err)
	}
	if err := f.AddChunk("w1", "p", 0, reordered); err != nil {
		t.Errorf("the same chunk in another order: %v", err)
	}
	if err := f.AddChunk("w1", "p", 0, chunk2); !errors.Is(err, ErrConflict) {
		t.Errorf("an index with different content: %v", err)
	}
	if err := f.AddChunk("w1", "p", 1, []Timeline{line(Payer, "a")}); !errors.Is(err, ErrConflict) {
		t.Errorf("a member already named in another chunk: %v", err)
	}
	if err := f.AddChunk("w1", "p", 2, chunk2); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"a missing chunk": 3, "a stored index at the count": 2, "a stored index above the count": 1} {
		if _, err := f.Seal("w1", "p", count); !errors.Is(err, ErrIncomplete) {
			t.Errorf("seal with %s: %v", name, err)
		}
	}
	if _, err := f.Seal("w1", "p", 6); !errors.Is(err, ErrLimit) {
		t.Errorf("a chunk count above the limit: %v", err)
	}
	if _, err := f.Seal("w1", "p", -1); !errors.Is(err, ErrInvalid) {
		t.Errorf("a negative chunk count: %v", err)
	}
	info, err := f.Plan("w1", "p")
	if err != nil || info.State != PlanDraft || !reflect.DeepEqual(info.Chunks, []int{0, 2}) || info.Base != 0 {
		t.Fatalf("plan read-back in draft %+v %v", info, err)
	}
	if _, err := f.ChangedMembers("w1", "p", Payer, "", 1); !errors.Is(err, ErrState) {
		t.Errorf("changed members of a draft: %v", err)
	}
	if err := f.AddChunk("w1", "p", 1, []Timeline{line(Hierarchy, "u", Entry{"root", From(a)})}); err != nil {
		t.Fatal(err)
	}
	preview, err := f.Seal("w1", "p", 3)
	if err != nil {
		t.Fatalf("seal after the missing chunk arrived: %v", err)
	}
	if len(preview.Kinds) != 3 || preview.Kinds[Payer].Changed != 2 || preview.Kinds[ViewGroup].EntriesAfter != 2 {
		t.Errorf("preview %+v", preview.Kinds)
	}
	for name, err := range map[string]error{
		"a chunk after seal": f.AddChunk("w1", "p", 3, nil),
		"sealing twice":      func() error { _, err := f.Seal("w1", "p", 3); return err }(),
	} {
		if !errors.Is(err, ErrState) {
			t.Errorf("%s: %v", name, err)
		}
	}
	sealed, _ := f.Plan("w1", "p")
	if sealed.State != PlanSealed || sealed.Digest != preview.Digest || len(sealed.Chunks) != 0 {
		t.Errorf("plan read-back when sealed %+v", sealed)
	}
	page, err := f.ChangedMembers("w1", "p", Payer, "", 1)
	if err != nil || len(page) != 1 || page[0].Member != "a" {
		t.Errorf("first page %+v %v", page, err)
	}
	page, _ = f.ChangedMembers("w1", "p", Payer, "a", 2)
	if len(page) != 1 || page[0].Member != "b" || !reflect.DeepEqual(page[0].Differs, []Interval{From(a)}) || len(page[0].Before) != 0 || !reflect.DeepEqual(page[0].After, []string{"p"}) {
		t.Errorf("second page %+v", page)
	}
	if page, _ = f.ChangedMembers("w1", "p", Payer, "b", 2); len(page) != 0 {
		t.Errorf("page after the last member %+v", page)
	}
	for name, limit := range map[string]int{"zero": 0, "above the page size": 3} {
		if _, err := f.ChangedTargets("w1", "p", Payer, "", limit); !errors.Is(err, ErrInvalid) {
			t.Errorf("page limit %s: %v", name, err)
		}
	}
	if groups, _ := f.ChangedTargets("w1", "p", ViewGroup, "g1", 2); !reflect.DeepEqual(groups, []string{"g2"}) {
		t.Errorf("targets after g1: %v", groups)
	}
	if err := f.Discard("w1", "p"); err != nil {
		t.Fatal(err)
	}
	for name, err := range map[string]error{
		"discarding twice":             f.Discard("w1", "p"),
		"a chunk for a discarded plan": f.AddChunk("w1", "p", 3, nil),
		"sealing a discarded plan":     func() error { _, err := f.Seal("w1", "p", 3); return err }(),
		"activating a discarded plan":  func() error { _, err := f.Activate("w1", "p", preview.Digest, 0); return err }(),
		"changes of a discarded plan":  func() error { _, err := f.ChangedTargets("w1", "p", Payer, "", 1); return err }(),
	} {
		if !errors.Is(err, ErrState) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if info, _ := f.Plan("w1", "p"); info.State != PlanDiscarded || info.Digest != ([32]byte{}) || len(info.Kinds) != 0 {
		t.Errorf("a discarded plan keeps only its state: %+v", info)
	}
	if err := f.Draft("w1", "p", 0); !errors.Is(err, ErrConflict) {
		t.Errorf("reusing a discarded identifier: %v", err)
	}
	f.refused("a plan without any change", "empty", []Violation{{Rule: RuleNoChange}})
	if err := f.Draft("w1", "empty", 0); !errors.Is(err, ErrConflict) {
		t.Errorf("reusing a rejected identifier: %v", err)
	}
	for name, err := range map[string]error{
		"sealing a rejected plan":     func() error { _, err := f.Seal("w1", "empty", 0); return err }(),
		"activating a rejected plan":  func() error { _, err := f.Activate("w1", "empty", [32]byte{}, 0); return err }(),
		"discarding a rejected plan":  f.Discard("w1", "empty"),
		"a chunk for a rejected plan": f.AddChunk("w1", "empty", 0, nil),
		"changes of a rejected plan":  func() error { _, err := f.ChangedMembers("w1", "empty", Payer, "", 1); return err }(),
	} {
		if !errors.Is(err, ErrState) {
			t.Errorf("%s: %v", name, err)
		}
	}
	f.publish("done", line(Payer, "a", Entry{"p", From(a)}))
	if err := f.Draft("w1", "done", 1); !errors.Is(err, ErrConflict) {
		t.Errorf("reusing an activated identifier: %v", err)
	}
	for name, err := range map[string]error{
		"discarding an activated plan":  f.Discard("w1", "done"),
		"a chunk for an activated plan": f.AddChunk("w1", "done", 1, nil),
		"sealing an activated plan":     func() error { _, err := f.Seal("w1", "done", 1); return err }(),
	} {
		if !errors.Is(err, ErrState) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := f.Draft("w9", "p", 1); !errors.Is(err, ErrConflict) {
		t.Errorf("a draft on a revision of a workspace that has none: %v", err)
	}
	if _, err := f.Revision("w9", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("a revision of a workspace that has none: %v", err)
	}
	if _, err := f.Plan("w9", "p"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a refused draft left a plan behind: %v", err)
	}
	f.refused("a plan that repeats the active state", "same", []Violation{{Rule: RuleNoChange}}, line(Payer, "a", Entry{"p", From(a)}), line(Payer, "zz"))
	for _, err := range []error{f.Discard("w1", "ghost"), f.AddChunk("w1", "ghost", 0, nil), func() error { _, err := f.Plan("w2", "done"); return err }()} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("an unknown plan: %v", err)
		}
	}
}

func TestPlanLimitsAtSeal(t *testing.T) {
	limits := testLimits
	limits.TimelinesPerPlan, limits.EntriesPerPlan = 3, 4
	a := nov(t, 1)
	build := func(t *testing.T, chunks ...[]Timeline) (fixture, error) {
		f := setupWith(t, limits)
		if err := f.Draft("w1", "p", 0); err != nil {
			t.Fatal(err)
		}
		for index, timelines := range chunks {
			if err := f.AddChunk("w1", "p", index, timelines); err != nil {
				t.Fatal(err)
			}
		}
		_, err := f.Seal("w1", "p", len(chunks))
		return f, err
	}
	three := []Timeline{line(Payer, "a", Entry{"p", From(a)}), line(Payer, "b", Entry{"p", From(a)}), line(Payer, "c", Entry{"p", From(a)})}
	if _, err := build(t, three); err != nil {
		t.Errorf("timelines at the plan limit: %v", err)
	}
	f, err := build(t, three, []Timeline{line(Payer, "d")})
	if info, _ := f.Plan("w1", "p"); !errors.Is(err, ErrRejected) || !reflect.DeepEqual(info.Violations, []Violation{{Rule: RulePlanLimit}}) {
		t.Errorf("timelines above the plan limit: %v %+v", err, info)
	}
	groups := func(count int) Timeline {
		entries := []Entry{}
		for _, name := range []string{"g1", "g2", "g3", "g4", "g5"}[:count] {
			entries = append(entries, Entry{name, From(a)})
		}
		return line(ViewGroup, "m", entries...)
	}
	if _, err := build(t, []Timeline{groups(4)}); err != nil {
		t.Errorf("entries at the plan limit: %v", err)
	}
	if _, err := build(t, []Timeline{groups(5)}); !errors.Is(err, ErrRejected) {
		t.Errorf("entries above the plan limit: %v", err)
	}
}

func TestDigest(t *testing.T) {
	a, b := nov(t, 1), nov(t, 15)
	manifest := []Timeline{
		line(ContractPool, "acct", Entry{"poolB", From(b)}, Entry{"poolA", Between(a, b)}),
		line(ViewGroup, "acct", Entry{"g2", From(a)}, Entry{"g1", From(a)}),
		line(Payer, "acct", Entry{"master", From(a)}),
	}
	seal := func(t *testing.T, workspace string, chunks ...[]Timeline) [32]byte {
		f := setup(t)
		if err := f.Draft(workspace, "p", 0); err != nil {
			t.Fatal(err)
		}
		for index, timelines := range chunks {
			if err := f.AddChunk(workspace, "p", index, timelines); err != nil {
				t.Fatal(err)
			}
		}
		preview, err := f.Seal(workspace, "p", len(chunks))
		if err != nil {
			t.Fatal(err)
		}
		return preview.Digest
	}
	whole := seal(t, "w1", manifest)
	const golden = "5713a727f1fd70f0338b3fe9deb461c79224fddeb03e8f0734783e67a31d4ef1"
	if got := hex.EncodeToString(whole[:]); got != golden {
		t.Errorf("digest %s; want %s", got, golden)
	}
	sorted := []Timeline{
		line(ContractPool, "acct", Entry{"poolA", Between(a, b)}, Entry{"poolB", From(b)}),
		line(ViewGroup, "acct", Entry{"g1", From(a)}, Entry{"g2", From(a)}),
	}
	if split := seal(t, "w1", []Timeline{manifest[2]}, sorted); split != whole {
		t.Errorf("another chunking and input order changes the digest")
	}
	if other := seal(t, "w2", manifest); other == whole {
		t.Errorf("another workspace has the same digest")
	}
	onBase := func(first Timeline) [32]byte {
		f := setup(t)
		f.publish("p1", first)
		preview, err := f.seal("p2", manifest...)
		if err != nil {
			t.Fatal(err)
		}
		return preview.Digest
	}
	if onBase(line(Hierarchy, "u", Entry{"v", From(a)})) == onBase(line(Hierarchy, "u", Entry{"w", From(a)})) {
		t.Errorf("the same manifest on two different revisions with the same number has the same digest")
	}
	different := map[string][]Timeline{
		"a bounded end instead of an open one": {line(ContractPool, "acct", Entry{"poolA", Between(a, b)}, Entry{"poolB", Between(b, nov(t, 30))}), manifest[1], manifest[2]},
		"another start day":                    {line(ContractPool, "acct", Entry{"poolA", Between(nov(t, 2), b)}, Entry{"poolB", From(b)}), manifest[1], manifest[2]},
		"another target":                       {manifest[0], manifest[1], line(Payer, "acct", Entry{"master2", From(a)})},
		"another kind":                         {manifest[0], manifest[1], line(Hierarchy, "acct", Entry{"master", From(a)})},
		"one timeline fewer":                   {manifest[0], manifest[1]},
		"an added empty timeline":              {manifest[0], manifest[1], manifest[2], line(Payer, "ghost")},
	}
	for name, timelines := range different {
		if got := seal(t, "w1", timelines); got == whole {
			t.Errorf("%s has the same digest", name)
		}
	}
	left := seal(t, "w1", []Timeline{line(Payer, "ab", Entry{"c", From(a)})})
	right := seal(t, "w1", []Timeline{line(Payer, "a", Entry{"bc", From(a)})})
	if left == right {
		t.Errorf("a boundary shift between member and target keeps the digest")
	}
}

func TestInputAndOutputAreCopies(t *testing.T) {
	f := setup(t)
	a := nov(t, 1)
	entries := []Entry{{"poolA", From(a)}}
	timelines := []Timeline{line(ContractPool, "acct", entries...), line(ViewGroup, "acct", Entry{"g1", From(a)})}
	if err := f.Draft("w1", "p", 0); err != nil {
		t.Fatal(err)
	}
	if err := f.AddChunk("w1", "p", 0, timelines); err != nil {
		t.Fatal(err)
	}
	entries[0].Target = "poolX"
	timelines[1].Member = "other"
	preview, err := f.Seal("w1", "p", 1)
	if err != nil {
		t.Fatal(err)
	}
	timelines[0].Entries[0].Interval = From(nov(t, 9))
	if _, err := f.Activate("w1", "p", preview.Digest, 0); err != nil {
		t.Fatal(err)
	}
	if got := f.target(1, ContractPool, "acct", nov(t, 2)); got != "poolA" {
		t.Errorf("the caller's later change reached the revision: %q", got)
	}
	if members, _ := f.MembersOn("w1", 1, ViewGroup, "g1", nov(t, 2)); !reflect.DeepEqual(members, []string{"acct"}) {
		t.Errorf("the caller's later rename reached the revision: %v", members)
	}
	members, _ := f.MembersOn("w1", 1, ContractPool, "poolA", nov(t, 2))
	members[0] = "tampered"
	changes, _ := f.ChangedMembers("w1", "p", ContractPool, "", 1)
	changes[0].After[0], changes[0].Differs[0] = "tampered", Interval{}
	info, _ := f.Plan("w1", "p")
	info.Kinds[ContractPool] = KindSummary{}
	preview.Kinds[ViewGroup] = KindSummary{}
	if again, _ := f.MembersOn("w1", 1, ContractPool, "poolA", nov(t, 2)); !reflect.DeepEqual(again, []string{"acct"}) {
		t.Errorf("a returned member list aliases the revision: %v", again)
	}
	if again, _ := f.ChangedMembers("w1", "p", ContractPool, "", 1); again[0].After[0] != "poolA" || again[0].Differs[0] != From(a) {
		t.Errorf("a returned change aliases the plan: %+v", again)
	}
	if again, _ := f.Plan("w1", "p"); again.Kinds[ContractPool].Changed != 1 || again.Kinds[ViewGroup].Changed != 1 {
		t.Errorf("a returned summary aliases the plan: %+v", again.Kinds)
	}
	pools, _ := f.ChangedTargets("w1", "p", ContractPool, "", 1)
	pools[0] = "tampered"
	if again, _ := f.ChangedTargets("w1", "p", ContractPool, "", 1); !reflect.DeepEqual(again, []string{"poolA"}) {
		t.Errorf("a returned target list aliases the plan: %v", again)
	}
	rejected, err := f.seal("bad", line(Payer, "x", Entry{"y", From(a)}), line(Payer, "y", Entry{"x", From(a)}))
	if !errors.Is(err, ErrRejected) || len(rejected.Violations) != 2 {
		t.Fatalf("rejection %+v %v", rejected, err)
	}
	rejected.Violations[0].Member = "tampered"
	stored, _ := f.Plan("w1", "bad")
	if stored.Violations[0].Member != "x" {
		t.Errorf("returned violations alias the plan: %+v", stored.Violations)
	}
	stored.Violations[1].Member = "tampered"
	if again, _ := f.Plan("w1", "bad"); again.Violations[1].Member != "y" {
		t.Errorf("read-back violations alias the plan: %+v", again.Violations)
	}
}

func TestQueriesRejectWhatDoesNotSuitThem(t *testing.T) {
	f := setup(t)
	a := nov(t, 1)
	f.publish("p", line(Hierarchy, "team", Entry{"dept", From(a)}), line(Hierarchy, "dept", Entry{"division", From(nov(t, 10))}), line(ViewGroup, "team", Entry{"g", From(a)}))
	chain, err := f.Chain("w1", 1, "team", nov(t, 12))
	if err != nil || !reflect.DeepEqual(chain, []string{"dept", "division"}) {
		t.Errorf("chain %v %v", chain, err)
	}
	if chain, _ := f.Chain("w1", 1, "team", nov(t, 5)); !reflect.DeepEqual(chain, []string{"dept"}) {
		t.Errorf("chain before the upper link exists %v", chain)
	}
	if chain, _ := f.Chain("w1", 1, "division", nov(t, 12)); len(chain) != 0 || chain == nil {
		t.Errorf("chain of a root %v", chain)
	}
	if chain, _ := f.Chain("w1", 0, "team", nov(t, 12)); len(chain) != 0 {
		t.Errorf("chain at revision 0 %v", chain)
	}
	invalid := map[string]error{
		"target of a view group":   func() error { _, _, err := f.TargetOn("w1", 1, ViewGroup, "team", a); return err }(),
		"segments of a view group": func() error { _, err := f.Segments("w1", 1, ViewGroup, "team", Between(a, nov(t, 5))); return err }(),
		"an unknown kind":          func() error { _, err := f.TargetsOn("w1", 1, "cost_center", "team", a); return err }(),
		"members of unknown kind":  func() error { _, err := f.MembersOn("w1", 1, "cost_center", "g", a); return err }(),
		"a zero day":               func() error { _, _, err := f.TargetOn("w1", 1, Hierarchy, "team", Day{}); return err }(),
		"an invalid member":        func() error { _, err := f.Chain("w1", 1, "", a); return err }(),
		"an invalid group":         func() error { _, err := f.Union("w1", 1, []string{"g", ""}, a); return err }(),
		"an invalid workspace":     func() error { _, err := f.Active(""); return err }(),
		"segments, invalid member": func() error { _, err := f.Segments("w1", 1, Hierarchy, "", Between(a, nov(t, 5))); return err }(),
		"targets, invalid member":  func() error { _, err := f.TargetsOn("w1", 1, Hierarchy, "a b", a); return err }(),
		"targets, zero day":        func() error { _, err := f.TargetsOn("w1", 1, Hierarchy, "team", Day{}); return err }(),
		"members, invalid target":  func() error { _, err := f.MembersOn("w1", 1, Hierarchy, "", a); return err }(),
		"members, zero day":        func() error { _, err := f.MembersOn("w1", 1, Hierarchy, "dept", Day{}); return err }(),
		"chain, zero day":          func() error { _, err := f.Chain("w1", 1, "team", Day{}); return err }(),
		"union, zero day":          func() error { _, err := f.Union("w1", 1, []string{"g"}, Day{}); return err }(),
		"target, invalid member":   func() error { _, _, err := f.TargetOn("w1", 1, Hierarchy, "a/b", a); return err }(),
		"page, invalid cursor":     func() error { _, err := f.ChangedMembers("w1", "p", Hierarchy, "a b", 1); return err }(),
		"page, unknown kind":       func() error { _, err := f.ChangedTargets("w1", "p", "cost_center", "", 1); return err }(),
	}
	for name, err := range invalid {
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, _, err := f.TargetOn("w1", 2, Hierarchy, "team", a); !errors.Is(err, ErrNotFound) {
		t.Errorf("a revision above the latest: %v", err)
	}
	if _, err := f.Revision("w1", 2); !errors.Is(err, ErrNotFound) {
		t.Errorf("a revision record above the latest: %v", err)
	}
	zero, err := f.Revision("w1", 0)
	if err != nil || zero != (RevisionInfo{}) {
		t.Errorf("revision 0 %+v %v", zero, err)
	}
	if got := f.target(0, Hierarchy, "team", nov(t, 12)); got != "" {
		t.Errorf("revision 0 answers %q", got)
	}
}

func TestWorkspacesAreSeparate(t *testing.T) {
	f := setup(t)
	a := nov(t, 1)
	f.publish("p", line(ContractPool, "acct", Entry{"poolA", From(a)}))
	if active, err := f.Active("w2"); err != nil || active != 0 {
		t.Errorf("an untouched workspace is at revision %d %v", active, err)
	}
	if target, has, err := f.TargetOn("w2", 0, ContractPool, "acct", nov(t, 5)); err != nil || has || target != "" {
		t.Errorf("another workspace reads this one's member: %q %v", target, err)
	}
	if _, _, err := f.TargetOn("w2", 1, ContractPool, "acct", nov(t, 5)); !errors.Is(err, ErrNotFound) {
		t.Errorf("another workspace reads this one's revision: %v", err)
	}
	for name, err := range map[string]error{
		"plan read":       func() error { _, err := f.Plan("w2", "p"); return err }(),
		"changed members": func() error { _, err := f.ChangedMembers("w2", "p", ContractPool, "", 1); return err }(),
		"activation":      func() error { _, err := f.Activate("w2", "p", [32]byte{}, 0); return err }(),
		"chunk":           f.AddChunk("w2", "p", 0, nil),
		"seal":            func() error { _, err := f.Seal("w2", "p", 0); return err }(),
		"discard":         f.Discard("w2", "p"),
	} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s of another workspace's plan: %v", name, err)
		}
	}
	if err := f.Draft("w2", "p", 0); err != nil {
		t.Errorf("the same plan identifier in another workspace: %v", err)
	}
	if err := f.AddChunk("w2", "p", 0, []Timeline{line(ContractPool, "acct", Entry{"poolZ", From(a)})}); err != nil {
		t.Fatal(err)
	}
	preview, err := f.Seal("w2", "p", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Activate("w2", "p", preview.Digest, 0); err != nil {
		t.Fatal(err)
	}
	if got := f.target(1, ContractPool, "acct", nov(t, 5)); got != "poolA" {
		t.Errorf("activation in another workspace changed this one: %q", got)
	}
	if target, _, _ := f.TargetOn("w2", 1, ContractPool, "acct", nov(t, 5)); target != "poolZ" {
		t.Errorf("the other workspace answers %q", target)
	}
	if members, _ := f.MembersOn("w2", 1, ContractPool, "poolA", nov(t, 5)); len(members) != 0 {
		t.Errorf("the other workspace lists this one's pool members: %v", members)
	}
}
