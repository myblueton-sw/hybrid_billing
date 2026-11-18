package server

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

type spanned struct {
	target   string
	from, to string
}

type randomWorkspace struct {
	id      string
	parties map[string]string
	lines   map[relationships.Kind]map[string][]spanned
}

func (w randomWorkspace) on(kind relationships.Kind, member, day string) []string {
	out := []string{}
	for _, s := range w.lines[kind][member] {
		if s.from <= day && (s.to == "" || day < s.to) {
			out = append(out, s.target)
		}
	}
	return out
}

func (w randomWorkspace) chain(member, day string) []string {
	out := []string{}
	for current := member; ; {
		above := w.on(relationships.Hierarchy, current, day)
		if len(above) == 0 {
			return out
		}
		out = append(out, above[0])
		current = above[0]
	}
}

func (w randomWorkspace) expected(kind relationships.Kind, day string, f orgFilter) []string {
	set := map[string]bool{}
	for k, byMember := range w.lines {
		for member := range byMember {
			if k == kind || (f.subject == SubjectAccount && w.parties[member] == "") {
				set[member] = true
			}
		}
	}
	out := []string{}
	for member := range set {
		_, timeline := w.lines[kind][member]
		party := w.parties[member] != ""
		on := w.on(kind, member, day)
		switch {
		case f.subject == SubjectAccount && party, f.subject == SubjectParty && !party, f.subject != SubjectAccount && !timeline:
		case f.unattributed && len(on) > 0, f.target != "" && !slices.Contains(on, f.target):
		case f.q != "" && !strings.Contains(member, strings.ToLower(f.q)):
		case f.entity != "" && member != f.entity && !slices.Contains(w.chain(member, day), f.entity):
		default:
			out = append(out, member)
		}
	}
	sort.Strings(out)
	return out
}

func buildRandom(t *testing.T, s *Server, r *rand.Rand, id string) randomWorkspace {
	w := randomWorkspace{id: id, parties: map[string]string{}, lines: map[relationships.Kind]map[string][]spanned{}}
	if _, err := s.orgs.PutWorkspace(organizations.Workspace{ID: id, Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	add := func(kind relationships.Kind, member string, spans ...spanned) {
		if w.lines[kind] == nil {
			w.lines[kind] = map[string][]spanned{}
		}
		w.lines[kind][member] = spans
	}
	starts, ends := []string{"2026-01-01", "2026-05-01"}, []string{"", "2026-07-01"}
	span := func(target string) spanned {
		return spanned{target: target, from: starts[r.Intn(2)], to: ends[r.Intn(2)]}
	}
	entities := []string{"le-a", "le-b"}
	for _, le := range entities {
		w.parties[le] = string(organizations.LegalEntity)
	}
	units := []string{}
	for i := 0; i < 1+r.Intn(3); i++ {
		unit := fmt.Sprintf("unit-%d", i)
		units, w.parties[unit] = append(units, unit), string(organizations.InternalUnit)
		add(relationships.Hierarchy, unit, spanned{target: entities[r.Intn(2)], from: "2026-01-01"})
	}
	for i := 0; i < 2+r.Intn(14); i++ {
		account := fmt.Sprintf("acct-%02d", r.Intn(40))
		if r.Intn(3) > 0 {
			add(relationships.Hierarchy, account, span(units[r.Intn(len(units))]))
		}
		if r.Intn(2) == 0 {
			add(relationships.Payer, account, span(entities[r.Intn(2)]))
		}
		if r.Intn(2) == 0 {
			spans := []spanned{{target: "g-1", from: "2026-01-01"}}
			if r.Intn(2) == 0 {
				spans = append(spans, span("g-2"))
			}
			add(relationships.ViewGroup, account, spans...)
		}
	}
	names := []string{}
	for party := range w.parties {
		names = append(names, party)
	}
	sort.Strings(names)
	for _, party := range names {
		kind, parent := w.parties[party], ""
		if kind == string(organizations.InternalUnit) {
			parent = w.lines[relationships.Hierarchy][party][0].target
		}
		if _, err := s.orgs.PutParty(organizations.Party{ID: party, WorkspaceID: id, Kind: organizations.PartyKind(kind), ParentID: parent, State: organizations.Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	day := func(raw string) relationships.Day {
		d, _ := parseDay(raw)
		return d
	}
	lines := []relationships.Timeline{}
	for kind, byMember := range w.lines {
		for member, spans := range byMember {
			entries := []relationships.Entry{}
			for _, sp := range spans {
				interval := relationships.From(day(sp.from))
				if sp.to != "" {
					interval = relationships.Between(day(sp.from), day(sp.to))
				}
				entries = append(entries, relationships.Entry{Target: sp.target, Interval: interval})
			}
			lines = append(lines, relationships.Timeline{Kind: kind, Member: member, Entries: entries})
		}
	}
	if len(lines) == 0 {
		return w
	}
	if err := s.rels.Draft(id, "random", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.rels.AddChunk(id, "random", 0, lines); err != nil {
		t.Fatal(err)
	}
	preview, err := s.rels.Seal(id, "random", 1)
	if err != nil {
		t.Fatalf("seal %v %+v", err, preview.Violations)
	}
	if _, err := s.rels.Activate(id, "random", preview.Digest, 0); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestFilteredPagesMatchABruteForceFilter(t *testing.T) {
	f := setup(t, SeedNone).ready()
	r := rand.New(rand.NewSource(300))
	kinds := []relationships.Kind{relationships.Hierarchy, relationships.Payer, relationships.ViewGroup, relationships.ContractPool}
	days := []string{"2025-12-31", "2026-03-01", "2026-06-01", "2026-09-01"}
	checked := 0
	for round := 0; round < 120; round++ {
		w := buildRandom(t, f.s, r, fmt.Sprintf("random-%02d", round))
		for trial := 0; trial < 25; trial++ {
			on := days[r.Intn(len(days))]
			filter := orgFilter{subject: []string{SubjectAny, SubjectAccount, SubjectParty}[r.Intn(3)], unattributed: r.Intn(3) == 0,
				q: []string{"", "", "ACCT-0", "1", "unit"}[r.Intn(5)], entity: []string{"", "", "le-a", "le-b"}[r.Intn(4)], target: []string{"", "", "", "g-1", "g-2", "unit-0", "le-a"}[r.Intn(7)]}
			kind := kinds[r.Intn(len(kinds))]
			want := w.expected(kind, on, filter)
			f.s.cfg.Org.FilterScan = 1 + r.Intn(12)
			limit := 1 + r.Intn(4)
			got, cursor := []string{}, ""
			for pages := 0; ; pages++ {
				day, _ := parseDay(on)
				page, err := f.s.readOrganizations(w.id, orgQuery{on: day, kind: kind, after: cursor, limit: limit, filter: filter})
				if err != nil || pages > 200 || len(page.Members) > limit || (page.ScanStopped && page.Next == "") {
					t.Fatalf("round %d trial %d: %v %+v", round, trial, err, page)
				}
				for _, m := range page.Members {
					got = append(got, m.Member)
				}
				if page.Next == "" {
					break
				}
				cursor = page.Next
			}
			if !slices.Equal(got, want) {
				t.Fatalf("round %d trial %d kind %s on %s filter %+v budget %d limit %d: %v; want %v", round, trial, kind, on, filter, f.s.cfg.Org.FilterScan, limit, got, want)
			}
			if len(want) > 0 {
				checked++
			}
		}
	}
	if checked < 300 {
		t.Errorf("only %d trials had matching rows", checked)
	}
}
