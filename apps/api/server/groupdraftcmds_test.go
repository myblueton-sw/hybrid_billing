package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func (f world) draftsRead(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations/view-group-drafts"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) draftMembersRead(a actor, id, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations/view-group-drafts/"+id+"/members"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func lines(ids ...string) string { return strings.Join(ids, "\n") }

func (f world) newDraft(a actor, name string, members ...string) reply {
	f.t.Helper()
	return f.run(a, "view_group.draft", viewGroupDraftsTarget, map[string]string{"name": name, "members": lines(members...)})
}

func draftOf(t testing.TB, got reply) (string, string) {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("draft %+v", got)
	}
	fields := got.body["fields"].(map[string]any)
	return fields["draft"].(string), fields["group"].(string)
}

func (f world) draftRow(a actor, id string) map[string]any {
	f.t.Helper()
	for _, d := range f.draftsRead(a, "").body["drafts"].([]any) {
		if row := d.(map[string]any); row["id"] == id {
			return row
		}
	}
	return nil
}

func (f world) proposeDraft(a actor, id, from, base string) reply {
	f.t.Helper()
	return f.run(a, "view_group.propose", id, map[string]string{"from": from, "base": base})
}

func viewEntries(n int, open bool) []relationships.Entry {
	out := []relationships.Entry{}
	for i := 0; i < n; i++ {
		day := sampleDay(2026, 1+i/14, 1+2*(i%14))
		next := sampleDay(2026, 1+i/14, 2+2*(i%14))
		out = append(out, relationships.Entry{Target: "sample-view-north", Interval: relationships.Between(day, next)})
	}
	if open {
		out = append(out, relationships.Entry{Target: "sample-view-north", Interval: relationships.From(sampleDay(2026, 6, 1))})
	}
	return out
}

func TestViewGroupDraftLifecycle(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id, group := draftOf(t, f.newDraft(root, "  Ops reviewers  ", "sample-acct-02", "sample-acct-01"))
	if !strings.HasPrefix(id, "vgd-") || len(id) != 20 || !strings.HasPrefix(group, "group-") || len(group) != 22 {
		t.Fatalf("ids %s %s", id, group)
	}
	row := f.draftRow(root, id)
	if row["name"] != "Ops reviewers" || row["group"] != group || row["group_label"] != group || row["new_group"] != true || row["state"] != "open" ||
		fmt.Sprint(row["member_count"]) != "2" || row["author"] != SampleAdmin || row["author_login"] != AdminLogin || row["plan"] != nil || row["discarded"] != "" ||
		fmt.Sprint(row["members"]) != "[map[id:sample-acct-01 name:Account 01 (sample)] map[id:sample-acct-02 name:Account 02 (sample)]]" {
		t.Fatalf("row %+v", row)
	}
	got := f.run(root, "view_group.edit", id, map[string]string{"name": "Ops review", "add": lines("sample-acct-05", "sample-acct-06"), "remove": "sample-acct-01"})
	if got.status != 200 || got.body["business_state"] != "open" {
		t.Fatalf("edit %+v", got)
	}
	if row := f.draftRow(root, id); row["name"] != "Ops review" || fmt.Sprint(row["member_count"]) != "3" {
		t.Errorf("edited %+v", row)
	}
	prop := f.proposeDraft(root, id, "2026-12-01", "1")
	plan := planID(t, prop)
	if prop.body["business_state"] != "sealed" {
		t.Fatalf("propose %+v", prop.body)
	}
	row = f.draftRow(root, id)
	p := row["plan"].(map[string]any)
	if row["state"] != "proposed" || p["id"] != plan || p["state"] != "sealed" || p["blocked"] != "impact_not_computed" || p["closed"] != "" || p["rule"] != "" {
		t.Errorf("proposed row %+v", row)
	}
	pr := f.planRow(root, plan)
	draft := pr["draft"].(map[string]any)
	if pr["member_name"] != "" || pr["target_name"] != "" || pr["change"].(map[string]any)["target"] != group || pr["change"].(map[string]any)["action"] != "join" ||
		draft["id"] != id || draft["name"] != "Ops review" || draft["new_group"] != true || fmt.Sprint(draft["member_count"]) != "3" || draft["group"] != group {
		t.Errorf("plan row %+v", pr)
	}
	if kinds := pr["kinds"].([]any); len(kinds) != 1 || kinds[0].(map[string]any)["kind"] != "view_group" || fmt.Sprint(kinds[0].(map[string]any)["changed"]) != "3" {
		t.Errorf("kinds %+v", kinds)
	}
	if single := f.planRow(root, planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))); single["draft"] != nil {
		t.Errorf("a single-member plan has a draft block %+v", single["draft"])
	}
	f.failed(t, "edit while proposed", f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-04"}), commands.InvalidStage)
	f.failed(t, "discard while proposed", f.run(root, "view_group.discard_draft", id, nil), commands.InvalidStage)
	f.failed(t, "propose twice", f.proposeDraft(root, id, "2026-12-01", "1"), commands.InvalidStage)
	targets := f.do("GET", "/api/v1/organizations/targets?kind=view_group", "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
	if strings.Contains(targets.raw, group) || strings.Contains(f.orgRead(root, "?kind=view_group").raw, "Ops review") {
		t.Error("an unactivated group or its draft name leaked into a relationship read")
	}
	f.s.plans.impact = computedImpact
	act := f.run(second, "relationship.activate", plan, nil)
	if act.status != 200 || act.body["fields"].(map[string]any)["revision"] != "2" {
		t.Fatalf("activate %+v", act)
	}
	row = f.draftRow(root, id)
	if row["state"] != "activated" || fmt.Sprint(row["activated_revision"]) != "2" || row["group_label"] != "Ops review" {
		t.Errorf("activated row %+v", row)
	}
	union, err := f.s.rels.Union(SampleWorkspace, 2, []string{group}, sampleDay(2026, 12, 1))
	if err != nil || strings.Join(union, ",") != "sample-acct-02,sample-acct-05,sample-acct-06" {
		t.Errorf("union %v %v", union, err)
	}
	if before, _ := f.s.rels.Union(SampleWorkspace, 2, []string{group}, sampleDay(2026, 11, 30)); len(before) != 0 {
		t.Errorf("members before the effective date %v", before)
	}
	m := members(f.orgRead(root, "?kind=view_group&on=2026-12-01"))["sample-acct-05"]
	if m["targets"].([]any)[0].(map[string]any)["name"] != "Ops review" || m["history"].([]any)[0].(map[string]any)["name"] != "Ops review" {
		t.Errorf("the activated label %+v", m)
	}
	listed := f.do("GET", "/api/v1/organizations/targets?kind=view_group&on=2026-12-01&q=ops", "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
	if rows := listed.body["targets"].([]any); len(rows) != 1 || rows[0].(map[string]any)["id"] != group || rows[0].(map[string]any)["name"] != "Ops review" || fmt.Sprint(rows[0].(map[string]any)["members_on"]) != "3" {
		t.Errorf("targets by label %s", listed.raw)
	}
	if pr := f.planRow(root, plan); pr["target_name"] != "Ops review" {
		t.Errorf("plan target name %v", pr["target_name"])
	}
	f.failed(t, "edit after activation", f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-04"}), commands.InvalidStage)
	f.failed(t, "discard after activation", f.run(root, "view_group.discard_draft", id, nil), commands.InvalidStage)
	f.failed(t, "propose after activation", f.proposeDraft(root, id, "2026-12-01", "2"), commands.InvalidStage)
	other, _ := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": group, "members": "sample-acct-04"}))
	if row := f.draftRow(root, other); row["name"] != "" || row["new_group"] != false || row["group_label"] != "Ops review" {
		t.Errorf("a draft on the activated group %+v", row)
	}
	f.s.drafts.put(&groupDraft{workspace: SampleWorkspace, id: "vgd-shadow", group: group, newGroup: true, name: "Shadow"})
	f.s.draftActivated(SampleWorkspace, "vgd-shadow", 3)
	if f.s.label(SampleWorkspace, group) != "Ops review" || f.s.drafts.labels[planKey(SampleWorkspace, group)].revision != 2 {
		t.Error("a newer draft overwrote the label")
	}
}

func TestViewGroupDraftOnExistingGroup(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id, group := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-north", "members": lines("sample-acct-01", "sample-acct-04")}))
	if group != "sample-view-north" {
		t.Fatalf("group %s", group)
	}
	if row := f.draftRow(root, id); row["group_label"] != "North view group (sample)" || row["new_group"] != false || row["name"] != "" {
		t.Errorf("row %+v", row)
	}
	f.failed(t, "rename an existing group", f.run(root, "view_group.edit", id, map[string]string{"name": "Renamed"}), commands.InvalidRequest, "name")
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	f.s.plans.impact = computedImpact
	if got := f.run(second, "relationship.activate", plan, nil); got.status != 200 {
		t.Fatalf("activate %+v", got)
	}
	if _, named := f.s.drafts.labels[planKey(SampleWorkspace, group)]; named {
		t.Error("an existing group gained a label")
	}
	union, _ := f.s.rels.Union(SampleWorkspace, 2, []string{"sample-view-north"}, sampleDay(2026, 12, 1))
	if strings.Join(union, ",") != "sample-acct-01,sample-acct-03,sample-acct-04" {
		t.Errorf("union %v", union)
	}
}

func TestViewGroupDraftRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	many := []string{}
	for i := 0; i < 32; i++ {
		many = append(many, fmt.Sprintf("sample-acct-%02d", i))
	}
	for name, c := range map[string]struct {
		fields map[string]string
		field  string
	}{
		"no name":             {map[string]string{"members": "sample-acct-01"}, "name"},
		"blank name":          {map[string]string{"name": "   ", "members": "sample-acct-01"}, "name"},
		"long name":           {map[string]string{"name": strings.Repeat("가", 81), "members": "sample-acct-01"}, "name"},
		"control in name":     {map[string]string{"name": "a\tb", "members": "sample-acct-01"}, "name"},
		"name with group":     {map[string]string{"name": "x", "group": "sample-view-north", "members": "sample-acct-01"}, "name"},
		"unknown group":       {map[string]string{"group": "sample-view-east", "members": "sample-acct-01"}, "group"},
		"pool as group":       {map[string]string{"group": "sample-pool-shared", "members": "sample-acct-01"}, "group"},
		"32 members":          {map[string]string{"name": "x", "members": lines(many...)}, "members"},
		"repeated member":     {map[string]string{"name": "x", "members": lines("sample-acct-01", "sample-acct-01")}, "members"},
		"blank line":          {map[string]string{"name": "x", "members": lines("sample-acct-01", "")}, "members"},
		"bad characters":      {map[string]string{"name": "x", "members": "sample acct"}, "members"},
		"party member":        {map[string]string{"name": "x", "members": "sample-unit-north-ops"}, "members"},
		"group as member":     {map[string]string{"name": "x", "members": "sample-view-north"}, "members"},
		"pool as member":      {map[string]string{"name": "x", "members": "sample-pool-shared"}, "members"},
		"legal entity member": {map[string]string{"name": "x", "members": "sample-le-north"}, "members"},
		"unknown member":      {map[string]string{"name": "x", "members": "sample-acct-99"}, "members"},
		"group itself":        {map[string]string{"group": "sample-view-south", "members": "sample-view-south"}, "members"},
	} {
		f.failed(t, name, f.run(root, "view_group.draft", viewGroupDraftsTarget, c.fields), commands.InvalidRequest, c.field)
	}
	f.failed(t, "no members", f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"name": "x"}), commands.InvalidRequest, "members")
	if n := len(f.s.drafts.byKey); n != 0 {
		t.Fatalf("refusals created %d drafts", n)
	}
	if _, err := memberLines(lines(many[:31]...), "members"); err != nil {
		t.Errorf("31 members per call %v", err)
	}
	if _, err := memberLines(lines(many...), "members"); err == nil {
		t.Error("32 well-formed members in one call")
	}
	if name, ok := draftName(strings.Repeat("가", 80)); !ok || name != strings.Repeat("가", 80) {
		t.Error("80 characters")
	}
	if _, ok := draftName("a\u0085b"); ok {
		t.Error("a C1 control")
	}
	id, _ := draftOf(t, f.newDraft(root, "x", "sample-acct-01", "sample-acct-02"))
	for name, c := range map[string]struct {
		fields map[string]string
		field  []string
	}{
		"nothing":            {map[string]string{}, nil},
		"blank name":         {map[string]string{"name": " "}, []string{"name"}},
		"add a member":       {map[string]string{"add": "sample-acct-01"}, []string{"add"}},
		"add a party":        {map[string]string{"add": "sample-le-south"}, []string{"add"}},
		"add 32":             {map[string]string{"add": lines(many...)}, []string{"add"}},
		"remove a stranger":  {map[string]string{"remove": "sample-acct-05"}, []string{"remove"}},
		"remove repeated":    {map[string]string{"remove": lines("sample-acct-01", "sample-acct-01")}, []string{"remove"}},
		"remove every one":   {map[string]string{"remove": lines("sample-acct-01", "sample-acct-02")}, []string{"remove"}},
		"bad remove":         {map[string]string{"remove": ""}, []string{"remove"}},
		"add and remove one": {map[string]string{"add": "sample-acct-05", "remove": "sample-acct-05"}, []string{"remove"}},
	} {
		f.failed(t, name, f.run(root, "view_group.edit", id, c.fields), commands.InvalidRequest, c.field...)
	}
	if row := f.draftRow(root, id); fmt.Sprint(row["member_count"]) != "2" || row["name"] != "x" {
		t.Errorf("refused edits changed the draft %+v", row)
	}
	swap := f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-05", "remove": lines("sample-acct-01", "sample-acct-02")})
	if swap.status != 200 || fmt.Sprint(f.draftRow(root, id)["members"]) != "[map[id:sample-acct-05 name:Account 05 (sample)]]" {
		t.Errorf("swap %+v", swap)
	}
	for name, c := range map[string][3]string{
		"bad date":    {"2026-12-32", "1", "from"},
		"past date":   {"2026-11-15", "1", "from"},
		"leading 0":   {"2026-12-01", "01", "base"},
		"negative":    {"2026-12-01", "-1", "base"},
		"2 to the 64": {"2026-12-01", "18446744073709551616", "base"},
	} {
		f.failed(t, name, f.proposeDraft(root, id, c[0], c[1]), commands.InvalidRequest, c[2])
	}
	f.failed(t, "empty base", f.proposeDraft(root, id, "2026-12-01", ""), commands.InvalidRequest, "base")
	f.failed(t, "stale base", f.proposeDraft(root, id, "2026-12-01", "0"), commands.RevisionConflict)
	f.failed(t, "unknown draft", f.run(root, "view_group.edit", "vgd-0000000000000000", map[string]string{"add": "sample-acct-04"}), commands.PermissionDenied)
	if len(f.s.plans.byKey) != 0 {
		t.Error("a refused propose created a plan")
	}
}

func TestViewGroupDraftLimits(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Org = OrgSettings{DraftsOpen: 2, DraftMembers: 3} })
	root := f.as(AdminLogin)
	f.failed(t, "member cap", f.newDraft(root, "x", "sample-acct-01", "sample-acct-02", "sample-acct-04", "sample-acct-05"), commands.PrerequisiteFailed)
	first, _ := draftOf(t, f.newDraft(root, "a", "sample-acct-01", "sample-acct-02", "sample-acct-04"))
	f.failed(t, "edit above the cap", f.run(root, "view_group.edit", first, map[string]string{"add": "sample-acct-05"}), commands.PrerequisiteFailed)
	second, _ := draftOf(t, f.newDraft(root, "b", "sample-acct-01"))
	f.failed(t, "open limit", f.newDraft(root, "c", "sample-acct-01"), commands.PrerequisiteFailed)
	plan := planID(t, f.proposeDraft(root, first, "2026-12-01", "1"))
	f.failed(t, "proposed still counts", f.newDraft(root, "c", "sample-acct-01"), commands.PrerequisiteFailed)
	if page := f.draftsRead(root, ""); fmt.Sprint(page.body["open"]) != "2" || fmt.Sprint(page.body["open_limit"]) != "2" {
		t.Errorf("open count %+v", page.body)
	}
	if got := f.run(root, "view_group.discard_draft", second, nil); got.status != 200 || got.body["business_state"] != "discarded" {
		t.Fatalf("discard %+v", got)
	}
	third, _ := draftOf(t, f.newDraft(root, "c", "sample-acct-01"))
	f.run(root, "relationship.discard", plan, nil)
	if row := f.draftRow(root, first); row["state"] != "open" || row["plan"].(map[string]any)["closed"] != "discarded" || row["plan"].(map[string]any)["state"] != "discarded" {
		t.Errorf("a discarded plan reopens the draft %+v", row)
	}
	f.failed(t, "reopened draft counts", f.newDraft(root, "d", "sample-acct-01"), commands.PrerequisiteFailed)
	f.failed(t, "discard twice", f.run(root, "view_group.discard_draft", second, nil), commands.InvalidStage)
	f.run(root, "view_group.discard_draft", third, nil)
	if row := f.draftRow(root, second); row["state"] != "discarded" || row["discarded"] != stamp(start) {
		t.Errorf("discarded row %+v", row)
	}
	again := planID(t, f.proposeDraft(root, first, "2026-12-02", "1"))
	if row := f.draftRow(root, first); row["plan"].(map[string]any)["id"] != again || row["state"] != "proposed" {
		t.Errorf("proposed again %+v", row)
	}
}

func TestViewGroupDraftEntriesPerTimeline(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	full := RelationshipLimits().EntriesPerTimeline
	f.activate("long-lines",
		relationships.Timeline{Kind: relationships.ViewGroup, Member: "sample-acct-full", Entries: viewEntries(full, false)},
		relationships.Timeline{Kind: relationships.ViewGroup, Member: "sample-acct-in", Entries: viewEntries(full-1, true)})
	f.failed(t, "full outside the group", f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-north", "members": "sample-acct-full"}), commands.InvalidRequest, "members")
	id, _ := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-north", "members": lines("sample-acct-in", "sample-acct-01")}))
	f.failed(t, "add full", f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-full"}), commands.InvalidRequest, "add")
	if got := f.proposeDraft(root, id, "2026-12-01", "2"); got.status != 200 {
		t.Errorf("a full member already in the group is accepted %+v", got)
	}
	other, _ := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-south", "members": "sample-acct-01"}))
	f.activate("fill", relationships.Timeline{Kind: relationships.ViewGroup, Member: "sample-acct-01", Entries: viewEntries(full, false)})
	f.failed(t, "full at propose", f.proposeDraft(root, other, "2026-12-01", "3"), commands.InvalidRequest)
}

func TestViewGroupDraftNoChangeAndPlanLimit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id, _ := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-south", "members": lines("sample-acct-03", "sample-acct-04")}))
	got := f.proposeDraft(root, id, "2026-12-01", "1")
	if planID(t, got) == "" || got.body["business_state"] != "rejected" {
		t.Fatalf("no change %+v", got.body)
	}
	if row := f.draftRow(root, id); row["state"] != "open" || row["plan"].(map[string]any)["rule"] != "no_change" || row["plan"].(map[string]any)["state"] != "rejected" {
		t.Errorf("rejected row %+v", row)
	}
	f.run(root, "view_group.discard_draft", id, nil)
	small := RelationshipLimits()
	small.EntriesPerPlan, small.TimelinesPerChunk = 3, 2
	f.s.relLimits = small
	reseed := func() {
		f.s.rels, _ = relationships.NewRegistry(small, f.s.clock)
		f.activate("seed-a", openLine(relationships.Payer, "sample-acct-01", "sample-le-north"), openLine(relationships.Payer, "sample-acct-02", "sample-le-north"))
		f.activate("seed-b", openLine(relationships.Payer, "sample-acct-05", "sample-le-south"), openLine(relationships.Payer, "sample-acct-06", "sample-le-south"))
	}
	reseed()
	big, _ := draftOf(t, f.newDraft(root, "big", "sample-acct-01", "sample-acct-02", "sample-acct-05", "sample-acct-06"))
	rejected := f.proposeDraft(root, big, "2026-12-01", "2")
	if rejected.body["business_state"] != "rejected" {
		t.Fatalf("plan limit %+v", rejected.body)
	}
	if row := f.draftRow(root, big); row["state"] != "open" || row["plan"].(map[string]any)["rule"] != "plan_limit" {
		t.Errorf("plan limit row %+v", row)
	}
	small.EntriesPerPlan = 100
	f.s.drafts.get(SampleWorkspace, big).plans = nil
	reseed()
	chunked := f.proposeDraft(root, big, "2026-12-01", "2")
	plan := planID(t, chunked)
	if chunked.body["business_state"] != "sealed" {
		t.Fatalf("two chunks %+v", chunked.body)
	}
	if info, _ := f.s.rels.Plan(SampleWorkspace, plan); info.Kinds[relationships.ViewGroup].Changed != 4 {
		t.Errorf("chunked plan %+v", info)
	}
}

func TestViewGroupDraftFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id, _ := draftOf(t, f.newDraft(root, "x", "sample-acct-01"))
	for _, step := range []string{"draft-parties", "draft-roles", "plan-active"} {
		f.s.fault = func(s string) bool { return s == step }
		f.failed(t, step+" draft", f.newDraft(root, "y", "sample-acct-02"), commands.DependencyUnavailable)
		f.failed(t, step+" edit", f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-02"}), commands.DependencyUnavailable)
		f.failed(t, step+" propose", f.proposeDraft(root, id, "2026-12-01", "1"), commands.DependencyUnavailable)
	}
	if err := f.s.groupUsable(f.s.drafts.get(SampleWorkspace, id), 2); err == nil {
		t.Error("an unreadable revision")
	}
	for _, step := range []string{"plan-draft", "plan-chunk"} {
		f.s.fault = func(s string) bool { return s == step }
		f.failed(t, step, f.proposeDraft(root, id, "2026-12-01", "1"), commands.InternalError)
	}
	f.s.fault = nil
	if row := f.draftRow(root, id); row["state"] != "open" || row["plan"] != nil {
		t.Errorf("a failed propose left a plan %+v", row)
	}
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	f.s.fault = func(s string) bool { return s == "draft-plan" }
	f.failed(t, "state unreadable on edit", f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-02"}), commands.DependencyUnavailable)
	f.failed(t, "state unreadable on draft", f.newDraft(root, "y", "sample-acct-02"), commands.DependencyUnavailable)
	f.code(t, "drafts read", f.draftsRead(root, ""), 503, string(commands.DependencyUnavailable))
	f.s.fault = func(s string) bool { return s == "plan-active" }
	f.code(t, "drafts read active", f.draftsRead(root, ""), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	if plan == "" {
		t.Fatal("plan")
	}
}

func TestViewGroupDraftNewGroupCollision(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, taken := range []string{"sample-view-north", "sample-acct-05", "sample-unit-north-ops"} {
		f.s.drafts.groupID = func() string { return taken }
		id, group := draftOf(t, f.newDraft(root, "x", "sample-acct-01"))
		if group != taken {
			t.Fatalf("seam %s", group)
		}
		f.failed(t, taken, f.proposeDraft(root, id, "2026-12-01", "1"), commands.InvalidStage)
		f.run(root, "view_group.discard_draft", id, nil)
	}
	f.s.drafts.groupID = func() string { return "sample-acct-01" }
	f.failed(t, "member equals the new group", f.newDraft(root, "x", "sample-acct-01"), commands.InvalidRequest, "members")
	id, _ := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-north", "members": "sample-acct-01"}))
	f.activate("drop-north", relationships.Timeline{Kind: relationships.ViewGroup, Member: "sample-acct-03", Entries: []relationships.Entry{{Target: "sample-view-south", Interval: relationships.From(sampleDay(2026, 1, 1))}}})
	f.failed(t, "existing group gone", f.proposeDraft(root, id, "2026-12-01", "2"), commands.InvalidRequest)
}

func TestViewGroupProposeOpenPlanLimitAndStale(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id, _ := draftOf(t, f.newDraft(root, "x", "sample-acct-01"))
	for i := 0; i < RelationshipLimits().OpenPlans; i++ {
		planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-11-17"))
	}
	f.failed(t, "open plans", f.proposeDraft(root, id, "2026-12-01", "1"), commands.PrerequisiteFailed)
	f.advance(2 * 24 * time.Hour)
	root = f.as(AdminLogin)
	if got := f.proposeDraft(root, id, "2026-12-20", "1"); got.status != 200 {
		t.Errorf("expired plans are discarded first %+v", got)
	}
}

func TestJoinRolesBeyondTheSampleShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	if _, err := f.s.orgs.PutParty(organizations.Party{ID: "sample-unit-spare", WorkspaceID: SampleWorkspace, Kind: organizations.InternalUnit, ParentID: "sample-le-north", State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	f.activate("roles", openLine(relationships.Hierarchy, "sample-unit-spare", "sample-le-north"), openLine(relationships.Payer, "sample-cp-1", "sample-le-north"),
		openLine(relationships.ContractPool, "sample-acct-05", "sample-cp-1"))
	f.failed(t, "a party with a timeline that is no target", f.newDraft(root, "x", "sample-unit-spare"), commands.InvalidRequest, "members")
	f.failed(t, "a contract pool target with its own timeline", f.newDraft(root, "x", "sample-cp-1"), commands.InvalidRequest, "members")
	if got := f.newDraft(root, "x", "sample-acct-05"); got.status != 200 {
		t.Errorf("a pool member is a source account %+v", got)
	}
}

func TestDraftEditorsCannotActivate(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.plans.impact = computedImpact
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id, _ := draftOf(t, f.newDraft(root, "Shared", "sample-acct-01"))
	if got := f.run(second, "view_group.edit", id, map[string]string{"add": "sample-acct-02"}); got.status != 200 {
		t.Fatalf("second edits %+v", got)
	}
	f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-05"})
	if g := f.s.drafts.get(SampleWorkspace, id); strings.Join(g.editors, ",") != SampleAdmin+","+SampleAdmin2 {
		t.Errorf("editors %v", g.editors)
	}
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	if row := f.planRow(second, plan); row["blocked"] != "own_draft" || row["can_activate"] != false {
		t.Errorf("an editor is offered activation %+v", row)
	}
	if row := f.planRow(root, plan); row["blocked"] != "own_proposal" {
		t.Errorf("the proposer keeps own_proposal %+v", row["blocked"])
	}
	before := f.activeRelationships(t)
	f.failed(t, "an editor activates", f.run(second, "relationship.activate", plan, nil), commands.PrerequisiteFailed)
	f.failed(t, "the proposer activates", f.run(root, "relationship.activate", plan, nil), commands.PrerequisiteFailed)
	if f.activeRelationships(t) != before {
		t.Error("a refused activation moved the revision")
	}
	other, _ := draftOf(t, f.newDraft(root, "Solo", "sample-acct-06"))
	solo := planID(t, f.proposeDraft(root, other, "2026-12-02", "1"))
	f.run(root, "relationship.discard", plan, nil)
	if got := f.run(second, "relationship.activate", solo, nil); got.status != 200 {
		t.Errorf("an administrator who never edited the draft activates %+v", got)
	}
}

func TestDraftAuthorCannotActivateAnotherProposal(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.plans.impact = computedImpact
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id, _ := draftOf(t, f.newDraft(second, "By second", "sample-acct-01"))
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	f.failed(t, "the author activates", f.run(second, "relationship.activate", plan, nil), commands.PrerequisiteFailed)
	if m := f.s.plans.get(SampleWorkspace, plan); strings.Join(m.editors, ",") != SampleAdmin2 || m.snapshot.Name != "By second" || m.snapshot.MemberCount != 1 {
		t.Errorf("plan record %+v %+v", m.editors, m.snapshot)
	}
}

func TestRequiredDraftFields(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.failed(t, "no fields at all", f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{}), commands.InvalidRequest, "members")
	id, _ := draftOf(t, f.newDraft(root, "x", "sample-acct-01"))
	planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	f.failed(t, "propose without base on a proposed draft", f.run(root, "view_group.propose", id, map[string]string{"from": "2026-12-01"}), commands.InvalidRequest, "base")
}

func TestDraftCommandResolve(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, _ := draftOf(t, f.newDraft(f.as(AdminLogin), "x", "sample-acct-01"))
	for _, c := range []struct {
		kind, ws, target string
		want             commands.Resolution
	}{
		{"draft", SampleWorkspace, viewGroupDraftsTarget, commands.Found},
		{"draft", SampleWorkspace, id, commands.NotFound},
		{"edit", SampleWorkspace, id, commands.Found},
		{"edit", SampleWorkspace, viewGroupDraftsTarget, commands.NotFound},
		{"edit", "other-workspace", id, commands.NotFound},
		{"draft", "other-workspace", viewGroupDraftsTarget, commands.NotFound},
	} {
		if _, got := (draftCommand{s: f.s, kind: c.kind}).Resolve(c.ws, c.target); got != c.want {
			t.Errorf("%+v got %v", c, got)
		}
	}
	if _, err := (draftCommand{s: f.s, kind: "discard"}).Apply(commands.Context{Target: id, Rationale: strings.Repeat("r", maxRationale+1)}); err == nil || err.(*commands.Failure).Code != commands.InvalidRequest {
		t.Errorf("long rationale %v", err)
	}
	if _, err := (draftCommand{s: f.s, kind: "propose"}).Apply(commands.Context{Target: id, Fields: map[string]string{"from": "2026-12-01", "base": "01"}}); err == nil || fmt.Sprint(err.(*commands.Failure).Fields) != "[base]" {
		t.Errorf("a non-canonical base reaching the handler %v", err)
	}
	for i := 0; i < closedPlansKept+1; i++ {
		f.s.plans.put(&planMeta{workspace: SampleWorkspace, id: fmt.Sprintf("plan-gone-%03d", i)})
	}
	if got, err := (draftCommand{s: f.s, kind: "discard"}).Apply(commands.Context{Target: id, Rationale: "r"}); err != nil || got.BusinessState != "discarded" || len(f.s.plans.byKey) != closedPlansKept {
		t.Errorf("draft commands prune closed plan records %v %d", err, len(f.s.plans.byKey))
	}
}
