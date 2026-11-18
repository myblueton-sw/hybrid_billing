package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func draftIDs(got reply) string {
	ids := []string{}
	for _, d := range got.body["drafts"].([]any) {
		ids = append(ids, d.(map[string]any)["name"].(string))
	}
	return strings.Join(ids, ",")
}

func TestDraftsReadOrderPagingAndSettings(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Org = OrgSettings{DraftsOpen: 20, Given: map[string]bool{"view_group_drafts_open": true}}
	})
	root := f.as(AdminLogin)
	ids := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d"} {
		ids[name], _ = draftOf(t, f.newDraft(root, name, "sample-acct-01"))
	}
	f.run(root, "view_group.discard_draft", ids["b"], nil)
	planID(t, f.proposeDraft(root, ids["a"], "2026-12-01", "1"))
	page := f.draftsRead(root, "")
	if draftIDs(page) != "d,c,a,b" || page.body["next"] != "" || fmt.Sprint(page.body["open"]) != "3" || fmt.Sprint(page.body["open_limit"]) != "20" || fmt.Sprint(page.body["active_revision"]) != "1" {
		t.Fatalf("page %s", page.raw)
	}
	settings := fmt.Sprint(page.body["settings"])
	for _, want := range []string{"map[default:1000 max:10000 min:100 name:org_filter_scan source:default unit:timelines value:1000]",
		"map[default:16 max:200 min:1 name:view_group_drafts_open source:start_flag unit:drafts value:20]", "name:view_group_draft_members", "name:view_group_drafts_kept", "name:view_group_activated_kept"} {
		if !strings.Contains(settings, want) {
			t.Errorf("settings %s missing %s", settings, want)
		}
	}
	first := f.draftsRead(root, "?limit=2")
	if draftIDs(first) != "d,c" || first.body["next"] != ids["c"] {
		t.Fatalf("first page %s", first.raw)
	}
	second := f.draftsRead(root, "?limit=2&after="+ids["c"])
	if draftIDs(second) != "a,b" || second.body["next"] != "" {
		t.Errorf("second page %s", second.raw)
	}
	for _, q := range []string{"?limit=0", "?limit=51", "?limit=01", "?limit=x", "?after=bad%20id", "?after=vgd-missing"} {
		f.code(t, q, f.draftsRead(root, q), 422, string(commands.InvalidRequest))
	}
	if got := f.draftsRead(root, "?limit=50"); got.status != 200 {
		t.Errorf("limit 50 %s", got.raw)
	}
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	f.code(t, "member", f.draftsRead(member, ""), 403, string(commands.PermissionDenied))
	f.code(t, "member members", f.draftMembersRead(member, ids["a"], ""), 403, string(commands.PermissionDenied))
	f.code(t, "no session", f.draftsRead(actor{}, ""), 401, string(commands.Unauthenticated))
	f.code(t, "no session members", f.draftMembersRead(actor{}, ids["a"], ""), 401, string(commands.Unauthenticated))
}

func TestDraftMembersRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id, _ := draftOf(t, f.newDraft(root, "x", "sample-acct-06", "sample-acct-01", "sample-acct-02"))
	page := f.draftMembersRead(root, id, "?limit=2")
	if fmt.Sprint(page.body["members"]) != "[map[id:sample-acct-01 name:Account 01 (sample)] map[id:sample-acct-02 name:Account 02 (sample)]]" || page.body["next"] != "sample-acct-02" ||
		fmt.Sprint(page.body["member_count"]) != "3" || page.body["draft"] != id || page.body["pruned"] != false {
		t.Fatalf("first %s", page.raw)
	}
	rest := f.draftMembersRead(root, id, "?limit=2&after=sample-acct-02")
	if fmt.Sprint(rest.body["members"]) != "[map[id:sample-acct-06 name:Account 06 (sample)]]" || rest.body["next"] != "" {
		t.Errorf("rest %s", rest.raw)
	}
	if all := f.draftMembersRead(root, id, ""); len(all.body["members"].([]any)) != 3 || all.body["next"] != "" {
		t.Errorf("default %s", all.raw)
	}
	if exact := f.draftMembersRead(root, id, "?limit=3"); exact.body["next"] != "" {
		t.Errorf("exactly the limit %s", exact.raw)
	}
	for _, q := range []string{"?limit=0", "?limit=100", "?after=bad%20id"} {
		f.code(t, q, f.draftMembersRead(root, id, q), 422, string(commands.InvalidRequest))
	}
	if got := f.draftMembersRead(root, id, "?limit=99"); got.status != 200 {
		t.Errorf("limit 99 %s", got.raw)
	}
	f.code(t, "unknown", f.draftMembersRead(root, "vgd-0000000000000000", ""), 404, string(commands.UnsupportedCommand))
	f.s.drafts.put(&groupDraft{workspace: "other-workspace", id: "vgd-elsewhere", members: []string{"x"}})
	f.code(t, "other workspace", f.draftMembersRead(root, "vgd-elsewhere", ""), 404, string(commands.UnsupportedCommand))
	if strings.Contains(f.draftsRead(root, "").raw, "vgd-elsewhere") {
		t.Error("a draft of another workspace is listed")
	}
	if n, _ := f.s.openDrafts("other-workspace"); n != 1 {
		t.Errorf("other workspace count %d", n)
	}
}

func TestDraftLabelsAndPruning(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Org = OrgSettings{DraftsKept: 50, ActivatedKept: 100} })
	f.s.plans.impact = computedImpact
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id, group := draftOf(t, f.newDraft(root, "Kept label", "sample-acct-01"))
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	if got := f.run(second, "relationship.activate", plan, nil); got.status != 200 {
		t.Fatalf("activate %+v", got)
	}
	for i := 0; i < 100; i++ {
		f.s.drafts.put(&groupDraft{workspace: SampleWorkspace, id: fmt.Sprintf("vgd-act-%03d", i), group: fmt.Sprintf("group-%03d", i), newGroup: true, name: "n", members: []string{"m"}, count: 1, plans: []string{plan}})
		f.s.draftActivated(SampleWorkspace, fmt.Sprintf("vgd-act-%03d", i), 9)
	}
	g := f.s.drafts.get(SampleWorkspace, id)
	if !g.pruned || g.members != nil || g.name != "" || g.count != 1 {
		t.Fatalf("the oldest activated draft keeps its body %+v", g)
	}
	if f.s.drafts.get(SampleWorkspace, "vgd-act-000").pruned || f.s.label(SampleWorkspace, group) != "Kept label" {
		t.Error("label or newer body lost")
	}
	if row := f.planRow(root, plan); row["draft"].(map[string]any)["name"] != "Kept label" || fmt.Sprint(row["draft"].(map[string]any)["member_count"]) != "1" {
		t.Errorf("the plan keeps its draft snapshot after pruning %+v", row["draft"])
	}
	if row := f.s.draftRow(g, draftActivated, 2, "", start); !row.Pruned || row.GroupLabel != "Kept label" || len(row.Members) != 0 || row.Name != "" || row.MemberCount != 1 {
		t.Errorf("pruned row %+v", row)
	}
	if page := f.draftMembersRead(root, id, ""); page.body["pruned"] != true || len(page.body["members"].([]any)) != 0 {
		t.Errorf("pruned members %s", page.raw)
	}
	for i := 0; i < 51; i++ {
		f.advance(2 * time.Second)
		d, _ := draftOf(t, f.newDraft(root, fmt.Sprintf("d%02d", i), "sample-acct-02"))
		f.run(root, "view_group.discard_draft", d, nil)
		if i == 0 {
			defer func(first string) {
				if f.s.drafts.get(SampleWorkspace, first) != nil {
					t.Error("the oldest discarded draft was kept")
				}
			}(d)
		}
	}
	discarded := 0
	for _, d := range f.s.drafts.byKey {
		if !d.discarded.IsZero() {
			discarded++
		}
	}
	if discarded != 50 {
		t.Errorf("discarded kept %d", discarded)
	}
}

func TestDraftStateAfterPlanMetaPruned(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id, _ := draftOf(t, f.run(root, "view_group.draft", viewGroupDraftsTarget, map[string]string{"group": "sample-view-south", "members": "sample-acct-04"}))
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	for i := 0; i < closedPlansKept+1; i++ {
		f.advance(2 * time.Second)
		f.run(root, "relationship.discard", planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01")), nil)
	}
	if f.s.plans.get(SampleWorkspace, plan) != nil {
		t.Fatal("the rejected plan record was kept")
	}
	row := f.draftRow(root, id)
	if row["state"] != "open" || row["plan"].(map[string]any)["state"] != "rejected" || row["plan"].(map[string]any)["closed"] != "" || row["plan"].(map[string]any)["rule"] != "no_change" {
		t.Errorf("state from the registry %+v", row)
	}
}

func TestDraftsGrantNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	snapshot := func() string {
		access := f.accessRead(root, "")
		preview := f.accessRead(root, "/preview?subject="+SampleMember+"&kind=installation")
		return fmt.Sprint(access.body["grants"], access.body["delegations"], preview.body["decisions"], f.orgRead(member, "").status, f.draftsRead(member, "").status)
	}
	before := snapshot()
	id, _ := draftOf(t, f.newDraft(root, "Grants nothing", "sample-acct-01", "sample-acct-02"))
	f.run(root, "view_group.edit", id, map[string]string{"add": "sample-acct-05"})
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	f.s.plans.impact = computedImpact
	if got := f.run(second, "relationship.activate", plan, nil); got.status != 200 {
		t.Fatalf("activate %+v", got)
	}
	if after := snapshot(); after != before || !strings.Contains(before, "403 403") {
		t.Errorf("grants or gate decisions changed\n%s\n%s", before, after)
	}
}

func TestDraftHelpers(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.drafts.labels[planKey(SampleWorkspace, "group-x")] = groupLabel{name: "Finance Team"}
	if !f.s.labelMatches(SampleWorkspace, "group-x", "finance") || f.s.labelMatches(SampleWorkspace, "group-x", "ops") || f.s.labelMatches(SampleWorkspace, "group-y", "") {
		t.Error("label search")
	}
	if f.s.label(SampleWorkspace, "sample-acct-01") != "Account 01 (sample)" || f.s.label("other-workspace", "group-x") != "" {
		t.Error("label lookup")
	}
	if got := f.s.labelled(SampleWorkspace, []namedRow{{ID: "group-x"}}); got[0].Name != "Finance Team" {
		t.Errorf("labelled %+v", got)
	}
	if got := f.s.labelledHistory(SampleWorkspace, []historyRow{{Target: "group-x"}}); got[0].Name != "Finance Team" {
		t.Errorf("labelled history %+v", got)
	}
	if store := newDraftStore(); !strings.HasPrefix(store.groupID(), "group-") || store.groupID() == store.groupID() {
		t.Error("group IDs")
	}
	g := &groupDraft{workspace: SampleWorkspace, plans: []string{"plan-missing"}}
	if _, err := f.s.draftState(g); !errors.Is(err, relationships.ErrNotFound) {
		t.Errorf("unknown plan %v", err)
	}
	if n, err := f.s.openDrafts(SampleWorkspace); n != 0 || err != nil {
		t.Errorf("empty %d %v", n, err)
	}
	f.s.drafts.put(g)
	if _, err := f.s.openDrafts(SampleWorkspace); err == nil {
		t.Error("an unreadable plan in the open count")
	}
}

func TestTargetsLabelWhilePaging(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.plans.impact = computedImpact
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	read := func(q string) reply {
		return f.do("GET", "/api/v1/organizations/targets?kind=view_group&on=2026-12-01"+q, "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
	}
	first := read("&q=view&limit=1")
	if rows := first.body["targets"].([]any); len(rows) != 1 || first.body["next"] != "sample-view-north" {
		t.Fatalf("first %s", first.raw)
	}
	f.s.drafts.groupID = func() string { return "sample-view-west" }
	id, _ := draftOf(t, f.newDraft(root, "Another view", "sample-acct-01"))
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	f.run(second, "relationship.activate", plan, nil)
	rest := read("&q=view&limit=5&revision=1&after=sample-view-north")
	if fmt.Sprint(rest.body["targets"]) != "[map[id:sample-view-south members_on:2 name:South view group (sample)]]" {
		t.Errorf("pinned rest %s", rest.raw)
	}
	now := read("&q=another")
	if fmt.Sprint(now.body["targets"]) != "[map[id:sample-view-west members_on:1 name:Another view]]" {
		t.Errorf("by label %s", now.raw)
	}
}
