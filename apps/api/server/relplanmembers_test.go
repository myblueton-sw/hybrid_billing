package server

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPlanMembersAreReadByMemberAndKind(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	id, _ := draftOf(t, f.newDraft(root, "Paged group", "sample-acct-01", "sample-acct-02", "sample-acct-06"))
	plan := planID(t, f.proposeDraft(root, id, "2026-12-01", "1"))
	row := f.planRow(root, plan)
	main := row["members"].([]any)
	kinds := row["kinds"].([]any)
	if len(main) != 3 || len(kinds) != 1 {
		t.Fatalf("the plan changes %d members of %d kinds", len(main), len(kinds))
	}
	kind := fmt.Sprint(kinds[0].(map[string]any)["kind"])
	if kinds[0].(map[string]any)["next"] != "" || main[0].(map[string]any)["kind"] != kind {
		t.Fatalf("a list listed whole has no cursor and each row names its kind: %v %v", kinds[0], main[0])
	}
	path := "/api/v1/organizations/plans/members?plan=" + plan + "&kind=" + kind
	whole := f.readAs(root, path)
	if whole.status != 200 || !reflect.DeepEqual(whole.body["members"], row["members"]) || whole.body["next"] != "" || fmt.Sprint(whole.body["total"]) != "3" || whole.body["order"] != "identifier" ||
		whole.body["plan"] != plan || whole.body["kind"] != kind {
		t.Fatalf("the whole list: %s", whole.raw[:min(len(whole.raw), 300)])
	}
	seen, after := []any{}, ""
	for pages := 0; pages < 5; pages++ {
		page := f.readAs(root, path+"&limit=1"+after)
		rows, _ := page.body["members"].([]any)
		if page.status != 200 || len(rows) != 1 || fmt.Sprint(page.body["total"]) != "3" {
			t.Fatalf("page %d: %s", pages, page.raw[:min(len(page.raw), 300)])
		}
		seen = append(seen, rows[0])
		if page.body["next"] == "" {
			break
		}
		if page.body["next"] != rows[0].(map[string]any)["member"] {
			t.Fatalf("the cursor is not the last member returned: %v", page.body["next"])
		}
		after = "&after=" + fmt.Sprint(page.body["next"])
	}
	if !reflect.DeepEqual(seen, main) {
		t.Fatalf("one at a time gives %d rows, not the main read's three in order", len(seen))
	}
	if past := f.readAs(root, path+"&after=zzzz"); past.status != 200 || len(past.body["members"].([]any)) != 0 || past.body["next"] != "" {
		t.Errorf("past the end: %s", past.raw[:min(len(past.raw), 200)])
	}
	if other := f.readAs(root, "/api/v1/organizations/plans/members?plan="+plan+"&kind=payer"); other.status != 200 || len(other.body["members"].([]any)) != 0 || fmt.Sprint(other.body["total"]) != "0" {
		t.Errorf("a kind the plan does not change: %s", other.raw[:min(len(other.raw), 200)])
	}
	base := "/api/v1/organizations/plans/members"
	for _, query := range []string{"", "?plan=" + plan, "?kind=" + kind, "?plan=never-proposed&kind=" + kind, "?plan=" + plan + "&kind=unknown", "?plan=" + plan + "&kind=" + kind + "&limit=0",
		"?plan=" + plan + "&kind=" + kind + "&limit=51", "?plan=" + plan + "&kind=" + kind + "&after=a,b", "?plan=" + plan + "&kind=" + kind + "&after=", "?plan=" + plan + "&kind=" + kind + "&q=a",
		"?plan=" + plan + "&plan=" + plan + "&kind=" + kind} {
		if refused := f.readAs(root, base+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	moved := f.s.plans.get(SampleWorkspace, plan)
	delete(f.s.plans.byKey, planKey(SampleWorkspace, plan))
	moved.workspace = "other-workspace"
	f.s.plans.byKey[planKey("other-workspace", plan)] = moved
	if hidden := f.readAs(root, path); hidden.status != 422 {
		t.Errorf("a plan of another workspace: %d", hidden.status)
	}
}
