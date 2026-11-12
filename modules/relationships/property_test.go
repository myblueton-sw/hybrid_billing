package relationships

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

const modelDays, modelDepth = 13, 3

type modelEntry struct {
	target   string
	from, to int
}

type modelKey struct {
	kind   Kind
	member string
}

type model map[modelKey][]modelEntry

func (e modelEntry) on(d int) bool { return e.from <= d && (e.to == 0 || d < e.to) }

func (m model) targets(kind Kind, member string, d int) []string {
	targets := []string{}
	for _, entry := range m[modelKey{kind, member}] {
		if entry.on(d) {
			targets = append(targets, entry.target)
		}
	}
	sort.Strings(targets)
	return targets
}

func (m model) members(kind Kind, target string, d int) []string {
	members := []string{}
	for id, entries := range m {
		if id.kind != kind {
			continue
		}
		for _, entry := range entries {
			if entry.target == target && entry.on(d) {
				members = append(members, id.member)
				break
			}
		}
	}
	sort.Strings(members)
	return members
}

func (m model) chain(member string, d int) ([]string, Rule) {
	chain, seen := []string{}, map[string]bool{member: true}
	for current := member; ; {
		parents := m.targets(Hierarchy, current, d)
		if len(parents) == 0 {
			return chain, ""
		}
		if seen[parents[0]] {
			return chain, RuleCycle
		}
		if len(chain) == modelDepth {
			return chain, RuleDepth
		}
		chain, current = append(chain, parents[0]), parents[0]
		seen[current] = true
	}
}

func (m model) violations() []Violation {
	set := map[Violation]bool{}
	for id, entries := range m {
		if len(entries) == 0 {
			continue
		}
		for d := 1; d <= modelDays; d++ {
			switch id.kind {
			case Payer:
				for _, payer := range m.targets(Payer, id.member, d) {
					if len(m.targets(Payer, payer, d)) > 0 {
						set[Violation{Payer, id.member, RulePayerChain}] = true
					}
				}
			case Hierarchy:
				if _, rule := m.chain(id.member, d); rule != "" {
					set[Violation{Hierarchy, id.member, rule}] = true
				}
			}
		}
		if id.kind == ContractPool || id.kind == AllocationPool || id.kind == ViewGroup {
			for _, entry := range entries {
				if len(m[modelKey{id.kind, entry.target}]) > 0 {
					set[Violation{id.kind, entry.target, RuleRoles}] = true
				}
			}
		}
	}
	list := []Violation{}
	for violation := range set {
		list = append(list, violation)
	}
	sort.Slice(list, func(i, j int) bool {
		return fmt.Sprint(list[i].Kind, "\x00", list[i].Member, "\x00", list[i].Rule) < fmt.Sprint(list[j].Kind, "\x00", list[j].Member, "\x00", list[j].Rule)
	})
	return list
}

func (m model) clone() model {
	next := model{}
	for id, entries := range m {
		next[id] = entries
	}
	return next
}

func modelDay(t testing.TB, d int) Day { return nov(t, d) }

func (e modelEntry) entry(t testing.TB) Entry {
	if e.to == 0 {
		return Entry{e.target, From(modelDay(t, e.from))}
	}
	return Entry{e.target, Between(modelDay(t, e.from), modelDay(t, e.to))}
}

func daysOf(t testing.TB, interval Interval) []int {
	days := []int{}
	for d := 1; d <= modelDays; d++ {
		if interval.contains(modelDay(t, d)) {
			days = append(days, d)
		}
	}
	return days
}

func randomTimeline(r *rand.Rand, kind Kind) (string, []modelEntry) {
	pick := func(names ...string) string { return names[r.Intn(len(names))] }
	var member string
	var targets []string
	switch kind {
	case Hierarchy:
		member, targets = pick("u0", "u1", "u2", "u3", "u4", "u5"), []string{"u0", "u1", "u2", "u3", "u4", "u5"}
	case Payer:
		member, targets = pick("a0", "a1", "a2", "a3"), []string{"a0", "a1", "a2", "a3"}
	default:
		member, targets = pick("m0", "m1", "m2", "m3"), []string{"t0", "t1", "t2"}
		if r.Intn(12) == 0 {
			member = pick("t0", "t1", "t2")
		}
		if r.Intn(12) == 0 {
			targets = append(targets, "m0", "m1")
		}
	}
	entries := []modelEntry{}
	if kind == ViewGroup {
		for _, target := range targets {
			if target == member || r.Intn(2) == 0 {
				continue
			}
			from := 1 + r.Intn(10)
			entry := modelEntry{target: target, from: from}
			if r.Intn(2) == 0 {
				entry.to = from + 1 + r.Intn(12-from)
			}
			entries = append(entries, entry)
		}
		return member, entries
	}
	cursor, last := 1+r.Intn(4), ""
	for cursor < 12 && r.Intn(4) != 0 {
		target := targets[r.Intn(len(targets))]
		if target == member || target == last {
			cursor++
			last = ""
			continue
		}
		entry := modelEntry{target: target, from: cursor}
		if r.Intn(4) == 0 {
			entries = append(entries, entry)
			break
		}
		entry.to = cursor + 1 + r.Intn(12-cursor)
		entries = append(entries, entry)
		cursor, last = entry.to, target
		if r.Intn(3) == 0 {
			cursor++
			last = ""
		}
	}
	return member, entries
}

func checkRevision(t *testing.T, f fixture, number uint64, m model, r *rand.Rand, round int, every bool) {
	t.Helper()
	ids := []string{"u0", "u1", "u2", "u3", "u4", "u5", "a0", "a1", "a2", "a3", "m0", "m1", "m2", "m3", "t0", "t1", "t2"}
	total := len(kinds) * len(ids) * modelDays
	for probe := 0; probe < total; probe++ {
		if !every && r.Intn(20) != 0 {
			continue
		}
		kind, id, d := kinds[probe%len(kinds)], ids[probe/len(kinds)%len(ids)], 1+probe/len(kinds)/len(ids)
		on := modelDay(t, d)
		want := m.targets(kind, id, d)
		if got, err := f.TargetsOn("w1", number, kind, id, on); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("round %d revision %d: targets of %s %s on %d = %v %v; model %v", round, number, kind, id, d, got, err, want)
		}
		if got, err := f.MembersOn("w1", number, kind, id, on); err != nil || !reflect.DeepEqual(got, m.members(kind, id, d)) {
			t.Fatalf("round %d revision %d: members of %s %s on %d = %v %v; model %v", round, number, kind, id, d, got, err, m.members(kind, id, d))
		}
		if kind == ViewGroup {
			union := map[string]bool{}
			for _, group := range []string{"t0", "t2", id} {
				for _, member := range m.members(ViewGroup, group, d) {
					union[member] = true
				}
			}
			wantUnion := []string{}
			for member := range union {
				wantUnion = append(wantUnion, member)
			}
			sort.Strings(wantUnion)
			if got, err := f.Union("w1", number, []string{"t0", "t2", id}, on); err != nil || !reflect.DeepEqual(got, wantUnion) {
				t.Fatalf("round %d revision %d: union on %d = %v %v; model %v", round, number, d, got, err, wantUnion)
			}
			continue
		}
		target, has, err := f.TargetOn("w1", number, kind, id, on)
		if err != nil || has != (len(want) == 1) || (has && target != want[0]) {
			t.Fatalf("round %d revision %d: target of %s %s on %d = %q %v %v; model %v", round, number, kind, id, d, target, has, err, want)
		}
		segments, err := f.Segments("w1", number, kind, id, Between(modelDay(t, 1), modelDay(t, modelDays)))
		if err != nil {
			t.Fatal(err)
		}
		covered := 0
		for index, segment := range segments {
			if index > 0 && segments[index-1].Target == segment.Target && segments[index-1].Has == segment.Has {
				t.Fatalf("round %d: segments are not maximal: %+v", round, segments)
			}
			for _, sd := range daysOf(t, segment.Interval) {
				covered++
				answer := m.targets(kind, id, sd)
				if segment.Has != (len(answer) == 1) || (segment.Has && segment.Target != answer[0]) {
					t.Fatalf("round %d revision %d: segment %+v on day %d; model %v", round, number, segment, sd, answer)
				}
			}
		}
		if covered != modelDays-1 {
			t.Fatalf("round %d: segments cover %d days: %+v", round, covered, segments)
		}
		if kind == Hierarchy {
			wantChain, _ := m.chain(id, d)
			if got, err := f.Chain("w1", number, id, on); err != nil || !reflect.DeepEqual(got, wantChain) {
				t.Fatalf("round %d revision %d: chain of %s on %d = %v %v; model %v", round, number, id, d, got, err, wantChain)
			}
		}
	}
}

func checkPreview(t *testing.T, f fixture, plan string, before, after model, round int) {
	t.Helper()
	info, err := f.Plan("w1", plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range kinds {
		changedMembers, targets, earliest := []string{}, map[string]bool{}, 0
		daysByMember := map[string][]int{}
		ids := map[string]bool{}
		for id := range before {
			ids[id.member] = id.kind == kind || ids[id.member]
		}
		for id := range after {
			ids[id.member] = id.kind == kind || ids[id.member]
		}
		for member, relevant := range ids {
			if !relevant || reflect.DeepEqual(before[modelKey{kind, member}], after[modelKey{kind, member}]) {
				continue
			}
			changedMembers = append(changedMembers, member)
			for d := 1; d <= modelDays; d++ {
				was, is := before.targets(kind, member, d), after.targets(kind, member, d)
				if reflect.DeepEqual(was, is) {
					continue
				}
				daysByMember[member] = append(daysByMember[member], d)
				if earliest == 0 || d < earliest {
					earliest = d
				}
				for _, target := range append(append([]string{}, was...), is...) {
					inWas, inIs := sort.SearchStrings(was, target) < len(was) && was[sort.SearchStrings(was, target)] == target, sort.SearchStrings(is, target) < len(is) && is[sort.SearchStrings(is, target)] == target
					if inWas != inIs {
						targets[target] = true
					}
				}
			}
		}
		sort.Strings(changedMembers)
		summary := info.Kinds[kind]
		if summary.Changed != len(changedMembers) || summary.HasEarliest != (earliest != 0) || (earliest != 0 && summary.Earliest != modelDay(t, earliest)) {
			t.Fatalf("round %d: summary of %s %+v; model %d changed, earliest %d", round, kind, summary, len(changedMembers), earliest)
		}
		got, after := []MemberChange{}, ""
		for {
			page, err := f.ChangedMembers("w1", plan, kind, after, 3)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, page...)
			if len(page) < 3 {
				break
			}
			after = page[len(page)-1].Member
		}
		if len(got) != len(changedMembers) {
			t.Fatalf("round %d: %d changed members of %s; model %v", round, len(got), kind, changedMembers)
		}
		for index, change := range got {
			days := []int{}
			for position, interval := range change.Differs {
				if position > 0 && change.Differs[position-1].end >= interval.from {
					t.Fatalf("round %d: difference intervals are not maximal: %+v", round, change.Differs)
				}
				days = append(days, daysOf(t, interval)...)
			}
			if change.Member != changedMembers[index] || !reflect.DeepEqual(days, append([]int{}, daysByMember[change.Member]...)) {
				t.Fatalf("round %d: change %+v covers %v; model %s %v", round, change, days, changedMembers[index], daysByMember[changedMembers[index]])
			}
		}
		wantTargets := []string{}
		for target := range targets {
			wantTargets = append(wantTargets, target)
		}
		sort.Strings(wantTargets)
		gotTargets, cursor := []string{}, ""
		for {
			page, err := f.ChangedTargets("w1", plan, kind, cursor, 3)
			if err != nil {
				t.Fatal(err)
			}
			gotTargets = append(gotTargets, page...)
			if len(page) < 3 {
				break
			}
			cursor = page[len(page)-1]
		}
		if !reflect.DeepEqual(gotTargets, wantTargets) {
			t.Fatalf("round %d: changed targets of %s %v; model %v", round, kind, gotTargets, wantTargets)
		}
	}
}

func TestPlansAgreeWithABruteForceModel(t *testing.T) {
	limits := Limits{Depth: modelDepth, EntriesPerTimeline: 16, TimelinesPerChunk: 8, ChunksPerPlan: 4, TimelinesPerPlan: 16, EntriesPerPlan: 200, OpenPlans: 2, Violations: 100, PageSize: 3}
	r := rand.New(rand.NewSource(170))
	counts := map[Rule]int{}
	accepted := 0
	for world := 0; world < 60; world++ {
		f := setupWith(t, limits)
		current := model{}
		history := []model{current}
		for step := 0; step < 25; step++ {
			round := world*100 + step
			candidate := current.clone()
			timelines := []Timeline{}
			named := map[modelKey]bool{}
			for count := 1 + r.Intn(4); count > 0; count-- {
				kind := kinds[r.Intn(len(kinds))]
				member, entries := randomTimeline(r, kind)
				id := modelKey{kind, member}
				if named[id] {
					continue
				}
				named[id] = true
				timeline := Timeline{Kind: kind, Member: member}
				for _, entry := range entries {
					timeline.Entries = append(timeline.Entries, entry.entry(t))
				}
				r.Shuffle(len(timeline.Entries), func(i, j int) { timeline.Entries[i], timeline.Entries[j] = timeline.Entries[j], timeline.Entries[i] })
				timelines = append(timelines, timeline)
				if len(entries) == 0 {
					delete(candidate, id)
				} else {
					candidate[id] = entries
				}
			}
			plan := fmt.Sprintf("p%d", step)
			active := uint64(len(history) - 1)
			if err := f.Draft("w1", plan, active); err != nil {
				t.Fatalf("round %d: draft %v", round, err)
			}
			half := len(timelines) / 2
			for index, part := range [][]Timeline{timelines[half:], timelines[:half]} {
				if err := f.AddChunk("w1", plan, index, part); err != nil {
					t.Fatalf("round %d: chunk %v for %+v", round, err, part)
				}
			}
			preview, err := f.Seal("w1", plan, 2)
			want := candidate.violations()
			if reflect.DeepEqual(candidate, current) {
				want = []Violation{{Rule: RuleNoChange}}
			}
			if len(want) > 0 {
				if !errors.Is(err, ErrRejected) || !reflect.DeepEqual(preview.Violations, want) {
					t.Fatalf("round %d: seal %v with %+v; model %+v\ncandidate %+v", round, err, preview.Violations, want, candidate)
				}
				for _, violation := range want {
					counts[violation.Rule]++
				}
				continue
			}
			if err != nil {
				t.Fatalf("round %d: seal refused a candidate the model accepts: %v %+v\ncandidate %+v", round, err, preview.Violations, candidate)
			}
			checkPreview(t, f, plan, current, candidate, round)
			number, err := f.Activate("w1", plan, preview.Digest, active)
			if err != nil || number != active+1 {
				t.Fatalf("round %d: activation %d %v", round, number, err)
			}
			accepted++
			current = candidate
			history = append(history, current)
			checkRevision(t, f, number, current, r, round, step%5 == 0)
		}
		for number, past := range history {
			checkRevision(t, f, uint64(number), past, r, world*100+99, false)
		}
	}
	for _, rule := range []Rule{RuleCycle, RuleDepth, RulePayerChain, RuleRoles, RuleNoChange} {
		if counts[rule] < 15 {
			t.Errorf("the generator reached %s only %d times", rule, counts[rule])
		}
	}
	if accepted < 300 {
		t.Errorf("only %d plans were accepted", accepted)
	}
	t.Logf("%d accepted; rejected by rule %v", accepted, counts)
}
