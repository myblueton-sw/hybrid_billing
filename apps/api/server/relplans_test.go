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

func (f world) plansRead(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations/plans", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) proposeChange(a actor, kind, member, action, target, from string) reply {
	f.t.Helper()
	fields := map[string]string{"kind": kind, "member": member, "action": action, "from": from}
	if target != "" {
		fields["target"] = target
	}
	return f.run(a, "relationship.propose", relationshipsTarget, fields)
}

func (f world) planRow(a actor, id string) map[string]any {
	f.t.Helper()
	for _, p := range f.plansRead(a).body["plans"].([]any) {
		if row := p.(map[string]any); row["id"] == id {
			return row
		}
	}
	return nil
}

func planID(t testing.TB, got reply) string {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("propose %+v", got)
	}
	return got.body["fields"].(map[string]any)["plan"].(string)
}

func TestProposeAndActivateMovesAttribution(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	got := f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01")
	id := planID(t, got)
	if got.body["business_state"] != "sealed" {
		t.Fatalf("state %+v", got.body)
	}
	row := f.planRow(root, id)
	change := row["change"].(map[string]any)
	if row["state"] != "sealed" || row["proposer"] != SampleAdmin || row["proposer_login"] != AdminLogin || row["reason"] != "sample reason" ||
		change["target"] != "sample-unit-north-data" || row["member_name"] != "Account 01 (sample)" || row["can_activate"] != false || row["blocked"] != "own_proposal" ||
		fmt.Sprint(row["base"]) != "1" || row["stale"] != false {
		t.Fatalf("row %+v", row)
	}
	changed := row["members"].([]any)
	m := changed[0].(map[string]any)
	if len(changed) != 1 || m["member"] != "sample-acct-01" || fmt.Sprint(m["after"]) != "[sample-unit-north-data sample-unit-north-ops]" ||
		fmt.Sprint(m["differs"]) != "[map[from:2026-12-01 until:]]" {
		t.Errorf("members %+v", changed)
	}
	kinds := row["kinds"].([]any)
	if k := kinds[0].(map[string]any); k["kind"] != "hierarchy" || fmt.Sprint(k["changed"]) != "1" || k["earliest"] != "2026-12-01" {
		t.Errorf("kinds %+v", kinds)
	}
	newer := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-02", "end", "", "2026-12-05"))
	if first := f.plansRead(second).body["plans"].([]any)[0].(map[string]any)["id"]; first != newer {
		t.Errorf("newest first: %v", first)
	}
	f.run(root, "relationship.discard", newer, nil)
	page := f.plansRead(second)
	if page.body["impact"].(map[string]any)["money"] != "not_computed" || fmt.Sprint(page.body["open_plans"]) != "1" || fmt.Sprint(page.body["open_limit"]) != "16" {
		t.Errorf("page %+v", page.body)
	}
	if f.planRow(second, id)["can_activate"] != true {
		t.Error("the second administrator cannot activate")
	}
	f.failed(t, "own activation", f.run(root, "relationship.activate", id, nil), commands.PrerequisiteFailed)
	act := f.run(second, "relationship.activate", id, nil)
	if act.status != 200 || act.body["business_state"] != "activated" || act.body["fields"].(map[string]any)["revision"] != "2" {
		t.Fatalf("activate %+v", act)
	}
	after := members(f.orgRead(root, "?on=2026-12-01"))
	before := members(f.orgRead(root, "?on=2026-11-30"))
	if targetIDs(after["sample-acct-01"]) != "sample-unit-north-data" || targetIDs(before["sample-acct-01"]) != "sample-unit-north-ops" {
		t.Errorf("attribution %v %v", after["sample-acct-01"], before["sample-acct-01"])
	}
	if fmt.Sprint(f.plansRead(root).body["open_plans"]) != "0" {
		t.Error("an activated plan still counts as open")
	}
	if r := f.planRow(root, id); r["state"] != "activated" || fmt.Sprint(r["activated_revision"]) != "2" || r["can_activate"] != false {
		t.Errorf("activated row %+v", r)
	}
	f.failed(t, "activate twice", f.run(root, "relationship.activate", id, nil), commands.InvalidStage)
	f.failed(t, "discard activated", f.run(root, "relationship.discard", id, nil), commands.InvalidStage)
}

func TestProposalRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for name, c := range map[string][5]string{
		"view group with set":       {"view_group", "sample-acct-01", "set", "sample-view-north", "2026-12-01"},
		"hierarchy with join":       {"hierarchy", "sample-acct-01", "join", "sample-le-north", "2026-12-01"},
		"set without target":        {"hierarchy", "sample-acct-01", "set", "", "2026-12-01"},
		"end with target":           {"hierarchy", "sample-acct-01", "end", "sample-le-north", "2026-12-01"},
		"target equals member":      {"hierarchy", "sample-acct-01", "set", "sample-acct-01", "2026-12-01"},
		"bad date":                  {"hierarchy", "sample-acct-01", "set", "sample-unit-north-ops", "2026-12-32"},
		"bad target characters":     {"hierarchy", "sample-acct-01", "set", "bad target", "2026-12-01"},
		"a party as its own parent": {"hierarchy", "sample-unit-north-ops", "set", "sample-unit-north-ops", "2026-12-01"},
		"past date":                 {"hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-11-15"},
		"unknown member":            {"hierarchy", "sample-acct-99", "set", "sample-unit-north-data", "2026-12-01"},
		"unknown hierarchy target":  {"hierarchy", "sample-acct-01", "set", "sample-unit-nowhere", "2026-12-01"},
		"payer to a unit":           {"payer", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"},
		"unknown pool":              {"allocation_pool", "sample-acct-01", "set", "sample-pool-new", "2026-12-01"},
		"unknown view group":        {"view_group", "sample-acct-01", "join", "sample-view-east", "2026-12-01"},
	} {
		field := map[string]string{"view group with set": "action", "hierarchy with join": "action", "set without target": "target", "end with target": "target",
			"target equals member": "target", "bad date": "from", "bad target characters": "target", "a party as its own parent": "target", "past date": "from", "unknown member": "member",
			"unknown hierarchy target": "target", "payer to a unit": "target", "unknown pool": "target", "unknown view group": "target"}[name]
		f.failed(t, name, f.proposeChange(root, c[0], c[1], c[2], c[3], c[4]), commands.InvalidRequest, field)
	}
	f.failed(t, "bad kind", f.proposeChange(root, "other", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"), commands.InvalidRequest)
	if _, err := f.s.orgs.PutParty(organizations.Party{ID: "sample-le-closed", WorkspaceID: SampleWorkspace, Kind: organizations.LegalEntity, State: organizations.Suspended}, 0); err != nil {
		t.Fatal(err)
	}
	f.failed(t, "inactive payer", f.proposeChange(root, "payer", "sample-acct-01", "set", "sample-le-closed", "2026-12-01"), commands.InvalidRequest, "target")
	f.failed(t, "member", f.runWith(f.as(MemberLogin), f.revision(root), "relationship.propose", relationshipsTarget,
		map[string]string{"kind": "hierarchy", "member": "sample-acct-01", "action": "set", "target": "sample-unit-north-data", "from": "2026-12-01"}), commands.PermissionDenied)
	f.failed(t, "wrong target", f.run(root, "relationship.propose", "elsewhere", map[string]string{"kind": "hierarchy", "member": "sample-acct-01", "action": "end", "from": "2026-12-01"}), commands.PermissionDenied)
	f.failed(t, "unknown plan", f.run(root, "relationship.activate", "plan-0000", nil), commands.PermissionDenied)
	if got := f.proposeChange(root, "hierarchy", "sample-acct-01", "end", "", "2026-11-16"); got.status != 200 {
		t.Errorf("today is allowed %+v", got)
	}
	f.code(t, "plans member", f.plansRead(f.as(MemberLogin)), 403, string(commands.PermissionDenied))
	f.code(t, "plans no session", f.plansRead(actor{}), 401, string(commands.Unauthenticated))
}

func TestRejectedPlansAndViewGroups(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	same := f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-ops", "2026-12-01")
	id := planID(t, same)
	row := f.planRow(root, id)
	if same.body["business_state"] != "rejected" || row["state"] != "rejected" || fmt.Sprint(row["violations"]) != "[map[kind: member: rule:no_change]]" || len(row["members"].([]any)) != 0 {
		t.Errorf("no change %+v", row)
	}
	cycle := f.proposeChange(root, "hierarchy", "sample-le-north", "set", "sample-unit-north-ops", "2026-12-01")
	if r := f.planRow(root, planID(t, cycle)); r["state"] != "rejected" || !strings.Contains(fmt.Sprint(r["violations"]), "cycle") {
		t.Errorf("cycle %+v", r)
	}
	join := planID(t, f.proposeChange(root, "view_group", "sample-acct-04", "join", "sample-view-north", "2026-12-01"))
	if act := f.run(second, "relationship.activate", join, nil); act.status != 200 {
		t.Fatalf("join %+v", act)
	}
	views := members(f.orgRead(root, "?on=2026-12-02&kind=view_group"))
	if targetIDs(views["sample-acct-04"]) != "sample-view-north,sample-view-south" {
		t.Errorf("joined %v", views["sample-acct-04"])
	}
	leave := planID(t, f.proposeChange(second, "view_group", "sample-acct-03", "leave", "sample-view-south", "2026-12-01"))
	if act := f.run(root, "relationship.activate", leave, nil); act.status != 200 {
		t.Fatalf("leave %+v", act)
	}
	views = members(f.orgRead(root, "?on=2026-12-02&kind=view_group"))
	if targetIDs(views["sample-acct-03"]) != "sample-view-north" {
		t.Errorf("left %v", views["sample-acct-03"])
	}
	pool := planID(t, f.proposeChange(root, "allocation_pool", "sample-acct-01", "set", "sample-pool-shared", "2026-12-01"))
	if r := f.planRow(root, pool); r["state"] != "sealed" {
		t.Errorf("pool %+v", r)
	}
}

func TestStaleExpiredAndDiscardedPlans(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	a := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	b := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-03", "set", "sample-unit-north-ops", "2026-12-01"))
	if f.run(second, "relationship.activate", a, nil).status != 200 {
		t.Fatal("activate a")
	}
	row := f.planRow(second, b)
	if row["stale"] != true || row["blocked"] != "stale" || row["can_activate"] != false {
		t.Errorf("stale row %+v", row)
	}
	f.failed(t, "stale activation", f.run(second, "relationship.activate", b, nil), commands.InvalidStage)
	c := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-04", "end", "", "2026-11-17"))
	if r := f.planRow(root, b); r["state"] != "discarded" || r["closed"] != "discarded_stale" {
		t.Errorf("auto stale %+v", r)
	}
	f.s.plans.put(&planMeta{workspace: "other-workspace", id: "plan-elsewhere", change: planChange{From: "2026-11-16"}})
	f.advance(48 * time.Hour)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	if r := f.planRow(root, c); r["blocked"] != "effective_date_passed" {
		t.Errorf("expired row %+v", r)
	}
	f.failed(t, "expired activation", f.run(second, "relationship.activate", c, nil), commands.InvalidStage)
	planID(t, f.proposeChange(root, "hierarchy", "sample-acct-05", "end", "", "2026-12-01"))
	if r := f.planRow(root, c); r["state"] != "discarded" || r["closed"] != "discarded_expired" {
		t.Errorf("auto expired %+v", r)
	}
	d := planID(t, f.proposeChange(second, "hierarchy", "sample-acct-06", "end", "", "2026-12-01"))
	if got := f.run(root, "relationship.discard", d, nil); got.status != 200 || got.body["business_state"] != "discarded" {
		t.Fatalf("discard %+v", got)
	}
	f.failed(t, "discard twice", f.run(root, "relationship.discard", d, nil), commands.InvalidStage)
	if r := f.planRow(root, d); r["closed"] != "discarded" || len(r["members"].([]any)) != 0 {
		t.Errorf("discarded row %+v", r)
	}
}

func TestPlanReadsAcrossPagesAndLimits(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	bulk := []relationships.Timeline{}
	for i := 0; i < 120; i++ {
		bulk = append(bulk, relationships.Timeline{Kind: relationships.Payer, Member: fmt.Sprintf("sample-bulk-%03d", i),
			Entries: []relationships.Entry{{Target: "sample-le-north", Interval: relationships.From(sampleDay(2026, 1, 1))}}})
	}
	many := []relationships.Entry{}
	for i := 0; i < RelationshipLimits().EntriesPerTimeline; i++ {
		day := sampleDay(2026, 1+i/14, 1+2*(i%14))
		next := sampleDay(2026, 1+i/14, 2+2*(i%14))
		many = append(many, relationships.Entry{Target: "sample-view-north", Interval: relationships.Between(day, next)})
	}
	bulk = append(bulk, relationships.Timeline{Kind: relationships.ViewGroup, Member: "sample-bulk-full", Entries: many})
	ws := SampleWorkspace
	if err := f.s.rels.Draft(ws, "bulk", 1); err != nil {
		t.Fatal(err)
	}
	if err := f.s.rels.AddChunk(ws, "bulk", 0, bulk); err != nil {
		t.Fatal(err)
	}
	preview, err := f.s.rels.Seal(ws, "bulk", 1)
	if err != nil {
		t.Fatal(err, preview.Violations)
	}
	if _, err := f.s.rels.Activate(ws, "bulk", preview.Digest, 1); err != nil {
		t.Fatal(err)
	}
	f.s.counter++
	got := f.proposeChange(root, "payer", "sample-bulk-119", "set", "sample-le-south", "2026-12-01")
	if planID(t, got) == "" || got.body["business_state"] != "sealed" {
		t.Errorf("member on a later page %+v", got.body)
	}
	if before := fmt.Sprint(f.planRow(root, got.body["fields"].(map[string]any)["plan"].(string))["members"]); !strings.Contains(before, "before:[sample-le-north]") || !strings.Contains(before, "differs:[map[from:2026-12-01 until:]]") {
		t.Errorf("the member's existing timeline on a later page was not read %s", before)
	}
	if found, _ := f.s.knownIn(ws, 2, func(_ relationships.Kind, line relationships.Timeline) bool { return line.Member == "sample-bulk-119" }, relationships.Payer); !found {
		t.Error("known on a later page")
	}
	if entries, _ := f.s.timelineOf(ws, 2, relationships.Payer, "sample-bulk-zzz"); entries != nil {
		t.Error("past the last page")
	}
	f.failed(t, "entry limit", f.proposeChange(root, "view_group", "sample-bulk-full", "join", "sample-view-north", "2026-12-20"), commands.InvalidRequest, "action")
	later := planID(t, f.proposeChange(root, "view_group", "sample-acct-04", "join", "sample-view-north", "2026-12-10"))
	if f.run(second, "relationship.activate", later, nil).status != 200 {
		t.Fatal("activate later join")
	}
	gap := planID(t, f.proposeChange(root, "view_group", "sample-acct-04", "join", "sample-view-north", "2026-12-01"))
	differs := fmt.Sprint(f.planRow(root, gap)["members"])
	if !strings.Contains(differs, "from:2026-12-01 until:2026-12-10") {
		t.Errorf("bounded differs %s", differs)
	}
	if _, _, bad := validChange(planChange{Kind: "hierarchy", Member: "a", Action: "other", From: "2026-12-01"}); bad != "action" {
		t.Error("unknown action accepted")
	}
	if _, _, bad := validChange(planChange{Kind: "hierarchy", Member: "bad member", Action: "end", From: "2026-12-01"}); bad != "member" {
		t.Error("invalid member accepted")
	}
	for _, fault := range []string{"plan-active", "plan-timelines"} {
		f.s.fault = func(at string) bool { return at == fault }
		f.failed(t, fault+" propose", f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"), commands.DependencyUnavailable)
		f.s.fault = nil
	}
	f.s.fault = func(at string) bool { return at == "plan-timelines" }
	if _, err := f.s.timelineOf(ws, 1, relationships.Hierarchy, "sample-acct-01"); err == nil {
		t.Error("timelines fault")
	}
	f.failed(t, "timelines fault on the target", f.proposeChange(root, "allocation_pool", "sample-unit-north-ops", "set", "sample-pool-shared", "2026-12-01"), commands.DependencyUnavailable)
	f.failed(t, "timelines fault on the member", f.proposeChange(root, "hierarchy", "sample-unit-north-ops", "set", "sample-le-south", "2026-12-01"), commands.DependencyUnavailable)
	f.s.fault = func(at string) bool { return at == "plan-draft" }
	f.failed(t, "draft collision", f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"), commands.InternalError)
	f.s.fault = func(at string) bool { return at == "plan-activate" }
	f.failed(t, "activation fault", f.run(second, "relationship.activate", gap, nil), commands.InvalidStage)
	f.s.fault = func(at string) bool { return at == "plan-active" }
	f.failed(t, "active fault on activate", f.run(second, "relationship.activate", gap, nil), commands.DependencyUnavailable)
	f.code(t, "active fault on read", f.plansRead(root), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	saved := f.s.orgs
	f.s.orgs, _ = organizations.NewRegistry(Installation)
	f.failed(t, "organizations unavailable", f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"), commands.DependencyUnavailable)
	f.s.orgs = saved
}

func TestOpenPlanLimitAndFailedProposals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.fault = func(at string) bool { return at == "plan-chunk" }
	for i := 0; i < 17; i++ {
		f.failed(t, "chunk fault", f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"), commands.InternalError)
	}
	f.s.fault = nil
	if got := f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"); got.status != 200 {
		t.Fatalf("after failures %+v", got)
	}
	for i := 0; i < 15; i++ {
		planID(t, f.proposeChange(root, "hierarchy", fmt.Sprintf("sample-acct-0%d", 1+i%6), "end", "", fmt.Sprintf("2026-12-%02d", 2+i)))
	}
	f.failed(t, "open limit", f.proposeChange(root, "hierarchy", "sample-acct-02", "end", "", "2026-12-28"), commands.PrerequisiteFailed)
	if got := f.plansRead(root); fmt.Sprint(got.body["open_plans"]) != "16" || len(got.body["plans"].([]any)) != plansShown-50+16 {
		t.Errorf("open count %v %d", got.body["open_plans"], len(got.body["plans"].([]any)))
	}
}

func TestPlanMetadataCapKeepsOpenPlans(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	open := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	for i := 0; i < closedPlansKept+5; i++ {
		f.s.plans.put(&planMeta{workspace: SampleWorkspace, id: fmt.Sprintf("plan-closed-%03d", i), proposer: SampleAdmin, change: planChange{From: "2026-12-01"}})
	}
	f.s.prunePlans()
	if f.s.plans.get(SampleWorkspace, open) == nil || len(f.s.plans.byKey) != closedPlansKept+1 {
		t.Errorf("kept %d, open kept %v", len(f.s.plans.byKey), f.s.plans.get(SampleWorkspace, open) != nil)
	}
	got := f.plansRead(root)
	ids := fmt.Sprint(got.body["plans"])
	if len(got.body["plans"].([]any)) != plansShown || !strings.Contains(ids, open) {
		t.Errorf("shown %d, open plan shown %v", len(got.body["plans"].([]any)), strings.Contains(ids, open))
	}
	f.s.plans.get(SampleWorkspace, open).proposer = ""
	if r := f.planRow(f.as(Admin2Login), open); r["can_activate"] != false || r["blocked"] != "own_proposal" {
		t.Errorf("empty proposer row %+v", r)
	}
	f.failed(t, "empty proposer", f.run(f.as(Admin2Login), "relationship.activate", open, nil), commands.PrerequisiteFailed)
	h := relationshipCommand{s: f.s, kind: "activate"}
	f.s.plans.put(&planMeta{workspace: "other-workspace", id: open, proposer: SampleAdmin})
	if _, res := h.Resolve("other-workspace", open); res != commands.NotFound {
		t.Errorf("other workspace %v", res)
	}
	if _, err := h.Apply(commands.Context{Target: open, Rationale: strings.Repeat("r", maxRationale+1)}); err.(*commands.Failure).Code != commands.InvalidRequest {
		t.Errorf("long rationale %v", err)
	}
}

func TestPlanErrorPaths(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	saved := f.s.rels
	empty, _ := relationships.NewRegistry(RelationshipLimits(), f.s.clock)
	f.s.rels = empty
	f.failed(t, "activation without the plan", f.run(second, "relationship.activate", id, nil), commands.InvalidStage)
	f.failed(t, "discard without the plan", f.run(root, "relationship.discard", id, nil), commands.InvalidStage)
	f.s.rels = saved
	if _, err := f.s.timelineOf("bad workspace", 0, relationships.Hierarchy, "x"); err == nil {
		t.Error("timeline of an invalid workspace")
	}
	if _, err := f.s.knownIn("bad workspace", 0, func(relationships.Kind, relationships.Timeline) bool { return false }, relationships.Payer); err == nil {
		t.Error("known in an invalid workspace")
	}
	if found, _ := f.s.knownIn(SampleWorkspace, 1, func(relationships.Kind, relationships.Timeline) bool { return false }, relationships.Payer); found {
		t.Error("found nothing")
	}
	if entries, _ := f.s.timelineOf(SampleWorkspace, 1, relationships.Payer, "sample-acct-00"); entries != nil {
		t.Error("a member before the first")
	}
	if entries, _ := f.s.timelineOf(SampleWorkspace, 1, relationships.Payer, "sample-zzz"); entries != nil {
		t.Error("a member after the last")
	}
}
