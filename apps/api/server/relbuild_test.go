package server

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func d(n int) relationships.Day { return sampleDay(2026, 1+(n-1)/28, 1+(n-1)%28) }

func show(entries []relationships.Entry) string {
	parts := []string{}
	for _, e := range entries {
		end, bounded := e.Interval.End()
		until := "open"
		if bounded {
			until = end.ISO()
		}
		parts = append(parts, e.Target+":"+e.Interval.Start().ISO()+"/"+until)
	}
	return strings.Join(parts, " ")
}

func between(t string, a, b int) relationships.Entry {
	return relationships.Entry{Target: t, Interval: relationships.Between(d(a), d(b))}
}
func open(t string, a int) relationships.Entry {
	return relationships.Entry{Target: t, Interval: relationships.From(d(a))}
}

func TestBuildTimelineActions(t *testing.T) {
	base := []relationships.Entry{between("a", 1, 10), open("b", 10)}
	cases := []struct {
		name, action, target string
		from                 int
		want                 []relationships.Entry
	}{
		{"set a new target cuts the open entry", "set", "c", 20, []relationships.Entry{between("a", 1, 10), between("b", 10, 20), open("c", 20)}},
		{"set the current target merges back", "set", "b", 20, base},
		{"set at the start of an entry drops it", "set", "c", 10, []relationships.Entry{between("a", 1, 10), open("c", 10)}},
		{"set to an earlier target adjacent to its end merges", "set", "a", 10, []relationships.Entry{open("a", 1)}},
		{"end cuts", "end", "", 20, []relationships.Entry{between("a", 1, 10), between("b", 10, 20)}},
		{"end before everything empties", "end", "", 1, []relationships.Entry{}},
		{"end after a closed entry keeps it", "end", "", 5, []relationships.Entry{between("a", 1, 5)}},
	}
	for _, c := range cases {
		if got := buildTimeline(base, c.action, c.target, d(c.from)); show(got) != show(c.want) {
			t.Errorf("%s: %s, want %s", c.name, show(got), show(c.want))
		}
	}
	groups := []relationships.Entry{open("g1", 1), between("g2", 5, 8), open("g2", 20)}
	for name, c := range map[string]struct {
		action, target string
		from           int
		want           string
	}{
		"join a new group":                         {"join", "g3", 10, show([]relationships.Entry{open("g1", 1), between("g2", 5, 8), open("g3", 10), open("g2", 20)})},
		"join before a later entry merges into it": {"join", "g2", 10, show([]relationships.Entry{open("g1", 1), between("g2", 5, 8), open("g2", 10)})},
		"join adjacent to a closed entry":          {"join", "g2", 8, show([]relationships.Entry{open("g1", 1), open("g2", 5)})},
		"join a covered group changes nothing":     {"join", "g1", 10, show(groups)},
		"leave cuts only that group":               {"leave", "g1", 10, show([]relationships.Entry{between("g1", 1, 10), between("g2", 5, 8), open("g2", 20)})},
		"leave drops later entries of it":          {"leave", "g2", 10, show([]relationships.Entry{open("g1", 1), between("g2", 5, 8)})},
		"unknown action keeps entries":             {"other", "x", 10, show(groups)},
	} {
		if got := show(buildTimeline(groups, c.action, c.target, d(c.from))); got != c.want {
			t.Errorf("%s: %s, want %s", name, got, c.want)
		}
	}
	asEntries := func(spans []span) []relationships.Entry {
		out := []relationships.Entry{}
		for _, s := range spans {
			out = append(out, s.entry())
		}
		return out
	}
	if got := show(asEntries(merge([]span{{target: "a", start: d(1), end: d(9)}, {target: "a", start: d(3), end: d(5)}, {target: "a", start: d(4), open: true}}))); got != "a:2026-01-01/open" {
		t.Errorf("merge %s", got)
	}
	if got := show(asEntries(merge([]span{{target: "a", start: d(1), end: d(9)}, {target: "a", start: d(3), end: d(5)}}))); got != "a:2026-01-01/2026-01-09" {
		t.Errorf("merge keeps the longer end %s", got)
	}
	if got := show(asEntries(merge([]span{{target: "b", start: d(4), end: d(6)}, {target: "a", start: d(5), open: true}, {target: "a", start: d(1), end: d(5)}}))); got != "a:2026-01-01/open b:2026-01-04/2026-01-06" {
		t.Errorf("merge sorts before joining %s", got)
	}
	if !spanCovers(span{target: "a", start: d(3), open: true}, d(40)) || spanCovers(span{target: "a", start: d(3), open: true}, d(2)) || spanCovers(span{target: "a", start: d(3), end: d(5)}, d(5)) {
		t.Error("covers")
	}
	if got := show(asEntries(merge([]span{{target: "a", start: d(1), end: d(9)}, {target: "a", start: d(3), end: d(12)}}))); got != "a:2026-01-01/2026-01-12" {
		t.Errorf("merge extends %s", got)
	}
}

func TestBuildTimelineProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	targets := []string{"t1", "t2", "t3"}
	for round := 0; round < 400; round++ {
		base := []relationships.Entry{}
		day := 1 + rng.Intn(5)
		for day < 60 && len(base) < 4 {
			length := 1 + rng.Intn(10)
			target := targets[rng.Intn(len(targets))]
			if len(base) > 0 && base[len(base)-1].Target == target {
				target = target + "x"
			}
			if rng.Intn(4) == 0 || day+length >= 60 {
				base = append(base, open(target, day))
				break
			}
			base = append(base, between(target, day, day+length))
			day += length + rng.Intn(3)
		}
		action := []string{"set", "end", "join", "leave"}[rng.Intn(4)]
		target := targets[rng.Intn(len(targets))]
		from := 1 + rng.Intn(70)
		if action == "set" || action == "end" {
			trimmed := []relationships.Entry{}
			for _, e := range base {
				if len(trimmed) == 0 || !spanCovers(spanOf(trimmed[len(trimmed)-1]), e.Interval.Start()) {
					trimmed = append(trimmed, e)
				}
			}
			base = trimmed
		}
		got := buildTimeline(base, action, target, d(from))
		kind := relationships.ViewGroup
		if action == "set" || action == "end" {
			kind = relationships.Hierarchy
		}
		check, _ := relationships.NewRegistry(RelationshipLimits(), time.Now)
		if err := check.Draft("w", "p", 0); err != nil {
			t.Fatal(err)
		}
		if len(got) > 0 {
			if err := check.AddChunk("w", "p", 0, []relationships.Timeline{{Kind: kind, Member: "m", Entries: got}}); err != nil {
				t.Fatalf("round %d: %s %s from %d: the module refuses %s: %v", round, action, target, from, show(got), err)
			}
		}
		for n := from; n < from+20; n++ {
			on := targetsOn(got, d(n))
			switch action {
			case "set":
				if on != "["+target+"]" {
					t.Fatalf("round %d: set %s from %d: day %d has %s", round, target, from, n, on)
				}
			case "end":
				if on != "[]" {
					t.Fatalf("round %d: end from %d: day %d has %s", round, from, n, on)
				}
			case "join":
				if !strings.Contains(on, target) {
					t.Fatalf("round %d: join %s from %d: day %d has %s", round, target, from, n, on)
				}
			case "leave":
				if strings.Contains(on, target+" ") || strings.HasSuffix(strings.TrimSuffix(on, "]"), target) && !strings.HasSuffix(strings.TrimSuffix(on, "]"), target+"x") {
					t.Fatalf("round %d: leave %s from %d: day %d has %s", round, target, from, n, on)
				}
			}
		}
		for i := 1; i < len(got); i++ {
			prev, cur := spanOf(got[i-1]), spanOf(got[i])
			if cur.start.Before(prev.start) {
				t.Fatalf("round %d: unsorted %s", round, show(got))
			}
		}
		seen := map[string]span{}
		for _, e := range got {
			s := spanOf(e)
			if p, ok := seen[s.target]; ok && (p.open || !p.end.Before(s.start)) {
				t.Fatalf("round %d: %s %s from %d: same target touches %s", round, action, target, from, show(got))
			}
			seen[s.target] = s
		}
		for n := 1; n < from; n++ {
			if a, b := targetsOn(base, d(n)), targetsOn(got, d(n)); a != b {
				t.Fatalf("round %d: %s %s from %d changed day %d: %s -> %s (%s => %s)", round, action, target, from, n, a, b, show(base), show(got))
			}
		}
	}
}

func targetsOn(entries []relationships.Entry, day relationships.Day) string {
	out := []string{}
	for _, e := range entries {
		if spanCovers(spanOf(e), day) {
			out = append(out, e.Target)
		}
	}
	return fmt.Sprint(out)
}
