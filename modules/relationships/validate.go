package relationships

import "sort"

func canonical(timelines []Timeline, limits Limits) ([]Timeline, error) {
	if len(timelines) > limits.TimelinesPerChunk {
		return nil, ErrLimit
	}
	result := make([]Timeline, 0, len(timelines))
	seen := map[key]struct{}{}
	for _, timeline := range timelines {
		id := key{timeline.Kind, timeline.Member}
		if _, repeated := seen[id]; repeated || !known(timeline.Kind) || !validID(timeline.Member) {
			return nil, ErrInvalid
		}
		seen[id] = struct{}{}
		if len(timeline.Entries) > limits.EntriesPerTimeline {
			return nil, ErrLimit
		}
		entries := append([]Entry(nil), timeline.Entries...)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Interval.from != entries[j].Interval.from {
				return entries[i].Interval.from < entries[j].Interval.from
			}
			return entries[i].Target < entries[j].Target
		})
		ends := map[string]uint32{}
		for index, entry := range entries {
			if !validID(entry.Target) || entry.Target == timeline.Member || !entry.Interval.valid() {
				return nil, ErrInvalid
			}
			if end, same := ends[entry.Target]; same && end >= entry.Interval.from {
				return nil, ErrInvalid
			}
			ends[entry.Target] = entry.Interval.end
			if exclusive(timeline.Kind) && index > 0 && entries[index-1].Interval.end > entry.Interval.from {
				return nil, ErrInvalid
			}
		}
		result = append(result, Timeline{Kind: timeline.Kind, Member: timeline.Member, Entries: entries})
	}
	sort.Slice(result, func(i, j int) bool { return less(result[i], result[j]) })
	return result, nil
}

func less(a, b Timeline) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.Member < b.Member
}

func targetSet(entries []Entry) map[string]struct{} {
	set := map[string]struct{}{}
	for _, entry := range entries {
		set[entry.Target] = struct{}{}
	}
	return set
}

func apply(base *revision, changed map[key][]Entry) *revision {
	next := &revision{timelines: make(map[key][]Entry, len(base.timelines)+len(changed)), members: make(map[key][]string, len(base.members))}
	for id, entries := range base.timelines {
		next.timelines[id] = entries
	}
	for id, members := range base.members {
		next.members[id] = members
	}
	type delta struct{ add, remove map[string]struct{} }
	deltas := map[key]*delta{}
	touch := func(kind Kind, target string) *delta {
		id := key{kind, target}
		if deltas[id] == nil {
			deltas[id] = &delta{add: map[string]struct{}{}, remove: map[string]struct{}{}}
		}
		return deltas[id]
	}
	for id, entries := range changed {
		before, after := targetSet(base.timelines[id]), targetSet(entries)
		for target := range before {
			if _, kept := after[target]; !kept {
				touch(id.kind, target).remove[id.id] = struct{}{}
			}
		}
		for target := range after {
			if _, had := before[target]; !had {
				touch(id.kind, target).add[id.id] = struct{}{}
			}
		}
		if len(entries) == 0 {
			delete(next.timelines, id)
		} else {
			next.timelines[id] = entries
		}
	}
	for id, change := range deltas {
		members := make([]string, 0, len(base.members[id])+len(change.add))
		for _, member := range base.members[id] {
			if _, removed := change.remove[member]; !removed {
				members = append(members, member)
			}
		}
		for member := range change.add {
			members = append(members, member)
		}
		if len(members) == 0 {
			delete(next.members, id)
			continue
		}
		sort.Strings(members)
		next.members[id] = members
	}
	return next
}

func (found *revision) walk(path []string, node string, interval Interval, depth int, rules map[Rule]struct{}) {
	for _, visited := range path {
		if visited == node {
			rules[RuleCycle] = struct{}{}
			return
		}
	}
	if len(path) > depth {
		rules[RuleDepth] = struct{}{}
		return
	}
	path = append(path, node)
	for _, entry := range found.timelines[key{Hierarchy, node}] {
		if narrowed, overlaps := entry.Interval.intersect(interval); overlaps {
			found.walk(path, entry.Target, narrowed, depth, rules)
		}
	}
}

func (found *revision) violations(limits Limits) ([]Violation, bool) {
	set := map[Violation]struct{}{}
	for id := range found.members {
		if disjointRoles(id.kind) {
			if _, member := found.timelines[id]; member {
				set[Violation{Kind: id.kind, Member: id.id, Rule: RuleRoles}] = struct{}{}
			}
		}
	}
	for id, entries := range found.timelines {
		switch id.kind {
		case Payer:
			for _, entry := range entries {
				for _, above := range found.timelines[key{Payer, entry.Target}] {
					if above.Interval.overlaps(entry.Interval) {
						set[Violation{Kind: Payer, Member: id.id, Rule: RulePayerChain}] = struct{}{}
					}
				}
			}
		case Hierarchy:
			rules := map[Rule]struct{}{}
			for _, entry := range entries {
				found.walk([]string{id.id}, entry.Target, entry.Interval, limits.Depth, rules)
			}
			for rule := range rules {
				set[Violation{Kind: Hierarchy, Member: id.id, Rule: rule}] = struct{}{}
			}
		}
	}
	return bounded(set, limits)
}

func bounded(set map[Violation]struct{}, limits Limits) ([]Violation, bool) {
	list := make([]Violation, 0, len(set))
	for violation := range set {
		list = append(list, violation)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Member != b.Member {
			return a.Member < b.Member
		}
		return a.Rule < b.Rule
	})
	if len(list) > limits.Violations {
		return list[:limits.Violations], true
	}
	return list, false
}

func targetsAt(entries []Entry, day uint32) string {
	targets := []string{}
	for _, entry := range entries {
		if entry.Interval.from <= day && day < entry.Interval.end {
			targets = append(targets, entry.Target)
		}
	}
	sort.Strings(targets)
	joined := ""
	for _, target := range targets {
		joined += target + "\x00"
	}
	return joined
}

func differs(before, after []Entry) []Interval {
	set := map[uint32]struct{}{}
	for _, entries := range [][]Entry{before, after} {
		for _, entry := range entries {
			set[entry.Interval.from] = struct{}{}
			set[entry.Interval.end] = struct{}{}
		}
	}
	bounds := make([]uint32, 0, len(set))
	for bound := range set {
		bounds = append(bounds, bound)
	}
	sort.Slice(bounds, func(i, j int) bool { return bounds[i] < bounds[j] })
	result := []Interval{}
	for index := 0; index+1 < len(bounds); index++ {
		if targetsAt(before, bounds[index]) == targetsAt(after, bounds[index]) {
			continue
		}
		if last := len(result) - 1; last >= 0 && result[last].end == bounds[index] {
			result[last].end = bounds[index+1]
			continue
		}
		result = append(result, Interval{from: bounds[index], end: bounds[index+1]})
	}
	return result
}

func spans(entries []Entry) map[string][]Interval {
	result := map[string][]Interval{}
	for _, entry := range entries {
		result[entry.Target] = append(result[entry.Target], entry.Interval)
	}
	return result
}

func sortedTargets(entries []Entry) []string {
	targets := make([]string, 0, len(entries))
	for target := range targetSet(entries) {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}
