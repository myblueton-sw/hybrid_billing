package server

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

func TestEvidenceReplacementsAreSearchedBeyondTheTwentyOfTheMainRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.ev.requirements)
	for i := 0; i < 60; i++ {
		f.evAddRequirement(fmt.Sprintf("later-%03d", i), false, false, releasetrace.TestRef{ID: fmt.Sprintf("t-later-%03d", i)})
	}
	subject := seeded + 1
	target := requirementTarget(subject)
	row := reqRows(f.requirementsRead(root, "?target="+target))[0]
	list, _ := row["replacements"].(map[string]any)
	main, _ := list["rows"].([]any)
	total := whole(list["total"])
	if len(main) != evidenceReplacements || total != 59 || list["more"] != true {
		t.Fatalf("the main read lists the first twenty of %d candidates: %d rows, more %v", total, len(main), list["more"])
	}
	path := "/api/v1/release-evidence/replacements?requirement=" + target
	first := f.readAs(root, path)
	if rows, _ := first.body["replacements"].([]any); first.status != 200 || !reflect.DeepEqual(rows, main) || whole(first.body["matching"]) != total || first.body["more"] != true ||
		first.body["order"] != "recorded" || first.body["requirement"] != target || first.body["q"] != "" {
		t.Fatalf("without a term the read gives the main read's twenty and the number of all: %s", first.raw[:min(len(first.raw), 300)])
	}
	found := f.readAs(root, path+"&q=LATER-05")
	rows, _ := found.body["replacements"].([]any)
	if found.status != 200 || len(rows) != 10 || whole(found.body["matching"]) != 10 || found.body["more"] != false || rows[0].(map[string]any)["label"] == main[0].(map[string]any)["label"] {
		t.Fatalf("a search reaches candidates beyond the twentieth: %s", found.raw[:min(len(found.raw), 300)])
	}
	beyond := rows[9].(map[string]any)
	for _, raw := range main {
		if raw.(map[string]any)["target"] == beyond["target"] {
			t.Fatal("the searched candidate is among the main read's twenty")
		}
	}
	if cut := f.readAs(root, path+"&q=later&limit=5"); len(cut.body["replacements"].([]any)) != 5 || whole(cut.body["matching"]) != total || cut.body["more"] != true {
		t.Errorf("a limit below the number that match: %s", cut.raw[:min(len(cut.raw), 200)])
	}
	one := f.readAs(root, path+"&id="+fmt.Sprint(beyond["target"]))
	if got, _ := one.body["replacements"].([]any); one.status != 200 || len(got) != 1 || !reflect.DeepEqual(got[0], rows[9]) || whole(one.body["matching"]) != 1 {
		t.Errorf("one candidate by its target: %s", one.raw[:min(len(one.raw), 300)])
	}
	for _, id := range []string{target, requirementTarget(1), "trace-requirement-99999", "not-a-target"} {
		if none := f.readAs(root, path+"&id="+id); none.status != 200 || len(none.body["replacements"].([]any)) != 0 || whole(none.body["matching"]) != 0 {
			t.Errorf("%s is offered as a replacement: %s", id, none.raw[:min(len(none.raw), 200)])
		}
	}
	if none := f.readAs(root, path+"&q=no-such-requirement"); whole(none.body["matching"]) != 0 || none.body["more"] != false {
		t.Errorf("a term nothing matches: %s", none.raw[:min(len(none.raw), 200)])
	}
	accepted := f.run(root, "release_evidence.retire_requirement", target, map[string]string{"reason": "replaced", "replacement": fmt.Sprint(beyond["target"])})
	if accepted.status != 200 {
		t.Fatalf("the command refuses a candidate the read offers: %s", accepted.raw[:min(len(accepted.raw), 300)])
	}
	if taken := f.readAs(root, "/api/v1/release-evidence/replacements?requirement="+requirementTarget(subject+1)+"&id="+fmt.Sprint(beyond["target"])); len(taken.body["replacements"].([]any)) != 0 {
		t.Error("a requirement that is already a pending replacement is offered to another retirement")
	}
	base := "/api/v1/release-evidence/replacements"
	for _, query := range []string{"", "?requirement=", "?requirement=trace-requirement-0", "?requirement=trace-requirement-99999", "?requirement=trace-requirement-01", "?requirement=not-a-target",
		"?q=later", "?requirement=" + target + "&q=", "?requirement=" + target + "&q=" + strings.Repeat("a", 65), "?requirement=" + target + "&limit=0", "?requirement=" + target + "&limit=21",
		"?requirement=" + target + "&id=", "?requirement=" + target + "&id=a&q=b", "?requirement=" + target + "&id=a&limit=5", "?requirement=" + target + "&requirement=" + target,
		"?requirement=" + target + "&after=1"} {
		if refused := f.readAs(root, base+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "release-evidence-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = func(at string) bool { return at == "roster" }
	if without := f.readAs(root, path); without.status != 200 {
		t.Errorf("the read needs the roster: %d", without.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, path); out.status != 422 {
		t.Errorf("another workspace: %d", out.status)
	}
}
