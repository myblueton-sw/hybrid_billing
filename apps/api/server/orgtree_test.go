package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func (f world) treeRead(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations/tree"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func treeRows(got reply) string {
	out := []string{}
	nodes, _ := got.body["nodes"].([]any)
	for _, row := range nodes {
		n := row.(map[string]any)
		out = append(out, fmt.Sprintf("%s/%v/%v/%v", strings.TrimPrefix(n["id"].(string), "sample-"), n["kind"], n["source"], n["children"]))
	}
	return strings.Join(out, ",")
}

func TestTreeReadOfTheSample(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.treeRead(root, "")
	if got.status != 200 || treeRows(got) != "le-north/legal_entity/none/some,le-seller/legal_entity/none/none,le-south/legal_entity/none/some" ||
		got.body["on"] != "2026-11-16" || fmt.Sprint(got.body["relationship_revision"]) != "1" || fmt.Sprint(got.body["active_revision"]) != "1" ||
		got.body["next"] != "" || got.body["scan_stopped"] != false || got.body["registry_empty"] != false || got.body["parent"] != "" || got.body["group"] != "" || got.body["revision"] != f.s.Revision() {
		t.Fatalf("roots %+v", got.body)
	}
	first := got.body["nodes"].([]any)[0].(map[string]any)
	if first["name"] != "North Holdings (sample)" || first["state"] != "active" || len(first) != 6 {
		t.Errorf("a root %+v", first)
	}
	if _, has := got.body["parties"]; has || len(got.body) != 10 {
		t.Errorf("the answer carries more than the tree %+v", got.body)
	}
	for query, want := range map[string]string{
		"?parent=sample-le-north":                                      "unit-north-data/internal_unit/relationship/some,unit-north-ops/internal_unit/relationship/some",
		"?parent=sample-le-seller":                                     "",
		"?parent=sample-unit-north-ops":                                "acct-01/account/relationship/none",
		"?parent=sample-unit-north-ops&on=2026-09-30":                  "acct-01/account/relationship/none,acct-02/account/relationship/none",
		"?parent=sample-unit-south-web&limit=2":                        "acct-04/account/relationship/none,acct-05/account/relationship/none",
		"?parent=sample-unit-south-web&limit=2&after=m:sample-acct-05": "acct-06/account/relationship/none",
		"?parent=sample-unit-south-web&after=m:":                       "acct-04/account/relationship/none,acct-05/account/relationship/none,acct-06/account/relationship/none",
		"?parent=sample-acct-01":                                       "",
		"?parent=sample-nobody":                                        "",
		"?parent=sample-le-north&on=2025-12-31":                        "",
		"?on=2025-12-31":                                               "le-north/legal_entity/none/none,le-seller/legal_entity/none/none,le-south/legal_entity/none/none",
		"?revision=0":                                                  "le-north/legal_entity/none/some,le-seller/legal_entity/none/none,le-south/legal_entity/none/some",
		"?parent=sample-le-north&revision=0":                           "unit-north-data/internal_unit/registry/none,unit-north-ops/internal_unit/registry/none",
		"?parent=sample-unit-north-ops&revision=0":                     "",
		"?limit=1&after=p:sample-le-north":                             "le-seller/legal_entity/none/none",
		"?group=detached":                                              "",
	} {
		if got := f.treeRead(root, query); got.status != 200 || treeRows(got) != want {
			t.Errorf("%s: %d %s", query, got.status, treeRows(got))
		}
	}
	if got := f.treeRead(root, "?parent=sample-unit-south-web&limit=2"); got.body["next"] != "m:sample-acct-05" || got.body["parent"] != "sample-unit-south-web" {
		t.Errorf("a cut page %+v", got.body)
	}
	if got := f.treeRead(root, "?limit=2"); got.body["next"] != "p:sample-le-seller" {
		t.Errorf("a cut page of parties %+v", got.body)
	}
	if got := f.treeRead(root, "?limit=3"); got.body["next"] != "" {
		t.Errorf("a page that ends with the list %+v", got.body)
	}
	for _, query := range []string{"?on=2026-13-01", "?parent=", "?parent=a%20b", "?parent=sample-le-north&group=detached", "?group=other", "?after=sample-le-north", "?after=x:sample-le-north",
		"?after=p:", "?after=p:a%20b", "?after=p:sample-nobody", "?group=detached&after=m:sample-le-north", "?limit=0", "?limit=100", "?limit=01", "?revision=2", "?revision=01", "?kind=hierarchy", "?q=north", "?on=2026-11-16&on=2026-11-17"} {
		if got := f.treeRead(root, query); got.status != 422 {
			t.Errorf("%s: %d %+v", query, got.status, got.body)
		}
	}
	if got := f.treeRead(root, "?provider=aws"); got.status != 501 {
		t.Errorf("a provider filter %d", got.status)
	}
	if got := f.treeRead(f.as(MemberLogin), ""); got.status != 403 {
		t.Errorf("a member %d", got.status)
	}
	if got := f.treeRead(actor{}, ""); got.status != 401 {
		t.Errorf("no session %d", got.status)
	}
	f.s.fault = func(part string) bool { return part == "organizations" }
	if got := f.treeRead(root, ""); got.status != 503 {
		t.Errorf("a fault %d", got.status)
	}
}

type treeWorld struct {
	id      string
	kinds   map[string]organizations.PartyKind
	parents map[string]string
	lines   map[string][]spanned
}

type treeRef struct {
	parent   map[string]string
	held     map[string]bool
	party    map[string]bool
	nodes    []string
	children map[string][]string
	roots    []string
	cycles   map[string]bool
}

func (w treeWorld) reference(day string) treeRef {
	ref := treeRef{parent: map[string]string{}, held: map[string]bool{}, party: map[string]bool{}, children: map[string][]string{}, cycles: map[string]bool{}}
	rank := map[string]int{}
	add := func(id string, r int) {
		if _, has := rank[id]; !has {
			rank[id] = r
			ref.nodes = append(ref.nodes, id)
		}
	}
	for id, kind := range w.kinds {
		ref.party[id] = true
		add(id, rankOf(kind))
	}
	members := []string{}
	for member := range w.lines {
		members = append(members, member)
	}
	sort.Strings(members)
	for _, member := range members {
		add(member, 2)
		on := []string{}
		for _, s := range w.lines[member] {
			if s.from <= day && (s.to == "" || day < s.to) {
				on = append(on, s.target)
			}
		}
		if len(on) == 0 {
			ref.held[member] = true
			continue
		}
		ref.parent[member] = on[0]
		add(on[0], 3)
	}
	for id := range w.kinds {
		if _, member := w.lines[id]; !member && w.parents[id] != "" {
			ref.parent[id] = w.parents[id]
		}
	}
	less := func(list []string) {
		sort.Slice(list, func(a, b int) bool {
			ra, rb := min(rank[list[a]], 2), min(rank[list[b]], 2)
			return ra < rb || (ra == rb && list[a] < list[b])
		})
	}
	for child, parent := range ref.parent {
		ref.children[parent] = append(ref.children[parent], child)
	}
	for _, list := range ref.children {
		less(list)
	}
	seen := map[string]bool{}
	var walk func(string)
	walk = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		for _, child := range ref.children[id] {
			walk(child)
		}
	}
	for _, id := range ref.nodes {
		if _, has := ref.parent[id]; !has && !ref.held[id] {
			ref.roots = append(ref.roots, id)
			walk(id)
		}
	}
	less(ref.roots)
	for id := range ref.held {
		for _, child := range ref.children[id] {
			walk(child)
		}
	}
	for _, id := range ref.nodes {
		if !seen[id] && !ref.held[id] {
			ref.cycles[id] = true
		}
	}
	return ref
}

func (ref treeRef) onCycle(id string) bool {
	for cur, steps := ref.parent[id], 0; cur != "" && steps <= len(ref.nodes); cur, steps = ref.parent[cur], steps+1 {
		if cur == id {
			return true
		}
	}
	return false
}

func buildTree(t *testing.T, s *Server, w treeWorld) bool {
	t.Helper()
	if _, err := s.orgs.PutWorkspace(organizations.Workspace{ID: w.id, Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for id := range w.kinds {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool {
		return rankOf(w.kinds[ids[a]]) < rankOf(w.kinds[ids[b]]) || (w.kinds[ids[a]] == w.kinds[ids[b]] && ids[a] < ids[b])
	})
	for _, id := range ids {
		if _, err := s.orgs.PutParty(organizations.Party{ID: id, WorkspaceID: w.id, Kind: w.kinds[id], ParentID: w.parents[id], State: organizations.Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	day := func(raw string) relationships.Day {
		d, _ := parseDay(raw)
		return d
	}
	lines := []relationships.Timeline{}
	for member, spans := range w.lines {
		entries := []relationships.Entry{}
		for _, sp := range spans {
			interval := relationships.From(day(sp.from))
			if sp.to != "" {
				interval = relationships.Between(day(sp.from), day(sp.to))
			}
			entries = append(entries, relationships.Entry{Target: sp.target, Interval: interval})
		}
		lines = append(lines, relationships.Timeline{Kind: relationships.Hierarchy, Member: member, Entries: entries})
	}
	if len(lines) == 0 {
		return true
	}
	if err := s.rels.Draft(w.id, "tree", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.rels.AddChunk(w.id, "tree", 0, lines); err != nil {
		return false
	}
	preview, err := s.rels.Seal(w.id, "tree", 1)
	if err != nil || len(preview.Violations) > 0 {
		return false
	}
	if _, err := s.rels.Activate(w.id, "tree", preview.Digest, 0); err != nil {
		t.Fatal(err)
	}
	return true
}

func (f world) treeList(ws, day, parent, group string, limit int) ([]treeNode, int) {
	f.t.Helper()
	on, _ := parseDay(day)
	out, pages := []treeNode{}, 0
	q := treeQuery{on: on, parent: parent, group: group, limit: limit, phase: phaseParties}
	for {
		page, err := f.s.readTree(ws, q)
		if err != nil {
			f.t.Fatalf("%s %s parent %q group %q after %s:%s: %v", ws, day, parent, group, q.phase, q.after, err)
		}
		if pages++; pages > 5000 {
			f.t.Fatalf("%s parent %q does not end", ws, parent)
		}
		if (page.ScanStopped && page.Next == "") || (!page.ScanStopped && page.Next != "" && len(page.Nodes) < limit) || len(page.Nodes) > limit {
			f.t.Fatalf("%s parent %q: stopped %v with %d of %d and next %q", ws, parent, page.ScanStopped, len(page.Nodes), limit, page.Next)
		}
		out = append(out, page.Nodes...)
		if page.Next == "" {
			return out, pages
		}
		q.phase, q.after, _ = strings.Cut(page.Next, ":")
	}
}

func idsOf(nodes []treeNode) string {
	out := []string{}
	for _, n := range nodes {
		out = append(out, n.ID)
	}
	return strings.Join(out, ",")
}

func (f world) checkTree(w treeWorld, day string, limit int, exact bool, undecided ...string) (int, int) {
	f.t.Helper()
	ref := w.reference(day)
	where := fmt.Sprintf("%s on %s limit %d", w.id, day, limit)
	listed, _ := f.treeList(w.id, day, "", "", limit)
	roots := []treeNode{}
	for _, n := range listed {
		if !exact && n.Children == childrenUnchecked && slices.Contains(undecided, n.ID) && !slices.Contains(ref.roots, n.ID) {
			continue
		}
		roots = append(roots, n)
	}
	if idsOf(roots) != strings.Join(ref.roots, ",") {
		f.t.Fatalf("%s: roots %s, want %s", where, idsOf(roots), strings.Join(ref.roots, ","))
	}
	check := func(n treeNode, root bool) {
		kind := "account"
		if root {
			kind = "unknown"
		}
		state := ""
		if ref.party[n.ID] {
			kind, state = string(w.kinds[n.ID]), "active"
		}
		want := childrenNone
		if len(ref.children[n.ID]) > 0 {
			want = childrenSome
		}
		if n.Kind != kind || n.State != state || (exact && n.Children != want) || (!exact && n.Children != want && n.Children != childrenUnchecked) {
			f.t.Fatalf("%s: node %+v, want kind %s state %q children %s", where, n, kind, state, want)
		}
	}
	for _, n := range roots {
		check(n, true)
	}
	for _, id := range append(append([]string{}, ref.nodes...), "nobody") {
		got, _ := f.treeList(w.id, day, id, "", limit)
		if idsOf(got) != strings.Join(ref.children[id], ",") {
			f.t.Fatalf("%s: children of %s are %s, want %s", where, id, idsOf(got), strings.Join(ref.children[id], ","))
		}
		for _, n := range got {
			check(n, false)
		}
	}
	detached, _ := f.treeList(w.id, day, "", groupDetached, limit)
	want := []string{}
	for _, id := range ref.nodes {
		if w.kinds[id] == organizations.LegalEntity && ref.onCycle(id) {
			want = append(want, id)
		}
	}
	sort.Strings(want)
	if idsOf(detached) != strings.Join(want, ",") {
		f.t.Fatalf("%s: detached %s, want %s", where, idsOf(detached), strings.Join(want, ","))
	}
	reached := map[string]bool{}
	var open func(string)
	open = func(id string) {
		if reached[id] {
			return
		}
		reached[id] = true
		for _, child := range ref.children[id] {
			open(child)
		}
	}
	for _, n := range detached {
		open(n.ID)
	}
	for id := range ref.cycles {
		if !reached[id] {
			f.t.Fatalf("%s: %s is in no tree and under no detached row %s", where, id, idsOf(detached))
		}
	}
	for id := range reached {
		if !ref.cycles[id] {
			f.t.Fatalf("%s: %s is under a detached row and also in a tree", where, id)
		}
	}
	return len(detached), len(ref.cycles)
}

func randomTree(r *rand.Rand, id string) treeWorld {
	w := treeWorld{id: id, kinds: map[string]organizations.PartyKind{}, parents: map[string]string{}, lines: map[string][]spanned{}}
	starts, ends := []string{"2026-01-01", "2026-05-01"}, []string{"", "2026-07-01"}
	span := func(target string) []spanned {
		return []spanned{{target: target, from: starts[r.Intn(2)], to: ends[r.Intn(2)]}}
	}
	entities, units, accounts := []string{}, []string{}, []string{}
	for i := 0; i < 1+r.Intn(3); i++ {
		entities = append(entities, fmt.Sprintf("le-%d", i))
		w.kinds[entities[i]] = organizations.LegalEntity
	}
	for i := 0; i < r.Intn(5); i++ {
		unit := fmt.Sprintf("unit-%d", i)
		units, w.kinds[unit], w.parents[unit] = append(units, unit), organizations.InternalUnit, entities[r.Intn(len(entities))]
	}
	for i := 0; i < r.Intn(12); i++ {
		accounts = append(accounts, fmt.Sprintf("acct-%02d", i))
	}
	anyOf := func(lists ...[]string) string {
		all := []string{"ghost-1", "ghost-2"}
		for _, l := range lists {
			all = append(all, l...)
		}
		return all[r.Intn(len(all))]
	}
	for _, unit := range units {
		if r.Intn(3) == 0 {
			w.lines[unit] = span(anyOf(entities, units))
		}
	}
	for _, le := range entities {
		if r.Intn(2) == 0 && len(units) > 0 {
			w.lines[le] = span(units[r.Intn(len(units))])
		}
	}
	for _, account := range accounts {
		if r.Intn(5) > 0 {
			w.lines[account] = span(anyOf(entities, units, accounts[:1]))
		}
	}
	for member, spans := range w.lines {
		if spans[0].target == member {
			delete(w.lines, member)
		}
	}
	return w
}

func TestTreePagesMatchTheTreeBuiltFromEveryMember(t *testing.T) {
	f := setup(t, SeedNone).ready()
	r := rand.New(rand.NewSource(320))
	built, withCycle, hanging := 0, 0, 0
	for round := 0; round < 400; round++ {
		w := randomTree(r, fmt.Sprintf("tree-%03d", round))
		if !buildTree(t, f.s, w) {
			continue
		}
		built++
		for _, day := range []string{"2025-12-31", "2026-03-01", "2026-06-01", "2026-09-01"} {
			entities, nodes := f.checkTree(w, day, 1+r.Intn(4), true)
			if entities > 0 {
				withCycle++
			}
			if nodes > entities {
				hanging++
			}
		}
	}
	if built < 100 || withCycle < 20 || hanging < 10 {
		t.Fatalf("too little was compared: %d revisions, %d days with a cycle, %d with nodes hanging below one", built, withCycle, hanging)
	}
}

func TestTreeCycleThroughSeveralLegalEntities(t *testing.T) {
	f := setup(t, SeedNone).ready()
	w := treeWorld{id: "ring", kinds: map[string]organizations.PartyKind{}, parents: map[string]string{}, lines: map[string][]spanned{}}
	const entities, stretch = 6, 30
	for i := 0; i < entities; i++ {
		le, unit := fmt.Sprintf("le-%d", i), fmt.Sprintf("unit-%d", i)
		w.kinds[le], w.kinds[unit], w.parents[unit] = organizations.LegalEntity, organizations.InternalUnit, fmt.Sprintf("le-%d", (i+1)%entities)
		previous := le
		for step := 0; step < stretch; step++ {
			next := fmt.Sprintf("mid-%d-%02d", i, step)
			w.lines[previous] = []spanned{{target: next, from: "2026-01-01"}}
			previous = next
		}
		w.lines[previous] = []spanned{{target: unit, from: "2026-01-01"}}
		w.lines[fmt.Sprintf("acct-%d", i)] = []spanned{{target: unit, from: "2026-01-01"}}
	}
	w.kinds["le-free"] = organizations.LegalEntity
	if !buildTree(t, f.s, w) {
		t.Fatal("the ring was refused")
	}
	if entities*(stretch+2) <= RelationshipLimits().Depth+2 {
		t.Fatal("the ring is not longer than one stretch may be")
	}
	if on, all := f.checkTree(w, "2026-06-01", 4, true); on != entities || all != entities*(stretch+3) {
		t.Fatalf("the ring has %d legal entities of %d nodes", on, all)
	}
	if roots, _ := f.treeList("ring", "2026-06-01", "", "", 4); idsOf(roots) != "le-free" {
		t.Fatalf("roots beside the ring %s", idsOf(roots))
	}
	if on, all := f.checkTree(w, "2025-06-01", 4, true); on != 0 || all != 0 {
		t.Fatalf("before the ring exists %d %d", on, all)
	}
}

func TestTreeStaysWithinItsScanBudget(t *testing.T) {
	f := setup(t, SeedNone).ready()
	w := treeWorld{id: "wide", kinds: map[string]organizations.PartyKind{"le-0": organizations.LegalEntity, "unit-0": organizations.InternalUnit}, parents: map[string]string{"unit-0": "le-0"}, lines: map[string][]spanned{}}
	for i := 0; i < 60; i++ {
		to := "2026-03-01"
		if i%7 == 0 {
			to = ""
		}
		w.lines[fmt.Sprintf("acct-%02d", i)] = []spanned{{target: "unit-0", from: "2026-01-01", to: to}}
		w.lines[fmt.Sprintf("old-%02d", i)] = []spanned{{target: fmt.Sprintf("ghost-%02d", i%9), from: "2026-01-01", to: to}}
	}
	for i := 0; i < 20; i++ {
		w.lines[fmt.Sprintf("gone-%02d", i)] = []spanned{{target: "ghost-00x", from: "2026-01-01", to: "2026-03-01"}}
	}
	if !buildTree(t, f.s, w) {
		t.Fatal("refused")
	}
	whole := f.s.cfg.Org.FilterScan
	defer func() { f.s.cfg.Org.FilterScan = whole }()
	for _, budget := range []int{1, 2, 4, 5, 13, whole} {
		f.s.cfg.Org.FilterScan = budget
		for _, day := range []string{"2026-02-01", "2026-06-01"} {
			f.checkTree(w, day, 4, budget == whole, "ghost-00x")
		}
		roots, _ := f.treeList("wide", "2026-06-01", "", "", 4)
		if at := slices.IndexFunc(roots, func(n treeNode) bool { return n.ID == "ghost-00x" }); (at >= 0) != (budget != whole) || (at >= 0 && (roots[at].Children != childrenUnchecked || roots[at].Kind != "unknown" || at == len(roots)-1)) {
			t.Errorf("budget %d: the target nobody belongs to any more is at %d of %s", budget, at, idsOf(roots))
		}
		_, pages := f.treeList("wide", "2026-06-01", "unit-0", "", 4)
		if budget < 5 && pages < 60/budget/2 {
			t.Errorf("budget %d read the children in %d pages", budget, pages)
		}
		if budget == whole && pages != 3 {
			t.Errorf("the whole budget read nine children by four in %d pages", pages)
		}
	}
	f.s.cfg.Org.FilterScan = 2
	on, _ := parseDay("2026-06-01")
	page, err := f.s.readTree("wide", treeQuery{on: on, parent: "le-0", limit: 4, phase: phaseParties})
	if err != nil || idsOf(page.Nodes) != "unit-0" || page.Nodes[0].Children != childrenSome || page.ScanStopped {
		t.Errorf("a child found within the budget %+v %v", page, err)
	}
	page, err = f.s.readTree("wide", treeQuery{on: on, parent: "unit-0", limit: 4, phase: phaseParties})
	if err != nil || idsOf(page.Nodes) != "acct-00" || page.Nodes[0].Children != childrenUnchecked || !page.ScanStopped || page.Next != "m:acct-01" {
		t.Errorf("a page cut by the budget %+v %v", page, err)
	}
}

func TestTreeAcrossWorkspacesAndFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if _, err := f.s.orgs.PutWorkspace(organizations.Workspace{ID: "other-workspace", Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	day := sampleDay(2026, 11, 16)
	for _, q := range []treeQuery{{}, {parent: "sample-le-north"}, {parent: "sample-unit-north-ops"}, {group: groupDetached}} {
		q.on, q.limit, q.phase = day, 50, phaseParties
		other, err := f.s.readTree("other-workspace", q)
		if err != nil || len(other.Nodes) != 0 || other.Relationship != 0 || !other.RegistryEmpty || other.Next != "" {
			t.Errorf("the other workspace %+v %v", other, err)
		}
	}
	if _, err := f.s.orgs.PutParty(organizations.Party{ID: "sample-le-north", WorkspaceID: "other-workspace", Kind: organizations.LegalEntity, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	other, err := f.s.readTree("other-workspace", treeQuery{on: day, limit: 50, phase: phaseParties})
	if err != nil || idsOf(other.Nodes) != "sample-le-north" || other.Nodes[0].Children != childrenNone || other.RegistryEmpty {
		t.Errorf("the same identifier in the other workspace %+v %v", other, err)
	}
	if _, err := f.s.readTree("bad ws", treeQuery{on: day, limit: 50, phase: phaseParties}); err == nil {
		t.Error("a bad workspace")
	}
	if _, err := f.s.readTree(SampleWorkspace, treeQuery{limit: 50, phase: phaseParties}); err == nil {
		t.Error("no day")
	}
}
