package server

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

func reqRows(got reply) []map[string]any {
	out := []map[string]any{}
	rows, _ := got.body["requirements"].([]any)
	for _, item := range rows {
		out = append(out, item.(map[string]any))
	}
	return out
}

func reqTargets(rows []map[string]any) string {
	out := []string{}
	for _, row := range rows {
		out = append(out, strings.TrimPrefix(fmt.Sprint(row["target"]), traceRequirementTarget))
	}
	return strings.Join(out, ",")
}

func (f world) requirementsRead(a actor, query string) reply {
	f.t.Helper()
	return f.readAs(a, "/api/v1/release-evidence/requirements"+query)
}

func TestRequirementsAreReadInRecordedOrderBeyondTheMainRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for i := 0; i < 70; i++ {
		f.evAddRequirement(fmt.Sprintf("extra-%03d", i), false, i%2 == 0, releasetrace.TestRef{ID: fmt.Sprintf("t-extra-%03d", i)})
	}
	total := len(f.s.ev.requirements)
	first := f.evidenceRead(root)
	section := evMap(first.body, "requirements")
	mainRows := evRows(evMap(section, "requirements"))
	if first.status != 200 || section["next"] != "50" || len(mainRows) != evidenceShown || mainRows[0]["target"] != "trace-requirement-1" || mainRows[0]["state"] != "tested" {
		t.Fatalf("the main read keeps its order and says where it stopped: next %v rows %d first %v %v", section["next"], len(mainRows), mainRows[0]["target"], mainRows[0]["state"])
	}
	all, next, pages := append([]map[string]any{}, mainRows...), "50", 0
	for next != "" {
		got := f.requirementsRead(root, "?limit=20&after="+next)
		if got.status != 200 || whole(got.body["total"]) != len(f.s.ev.requirements) || whole(got.body["matching"]) != len(f.s.ev.requirements) || got.body["order"] != "recorded" ||
			got.body["revision"] == "" || got.body["server_time"] == "" {
			t.Fatalf("after %s: %s", next, got.raw)
		}
		all = append(all, reqRows(got)...)
		next = fmt.Sprint(got.body["next"])
		if pages++; pages == 1 {
			f.evAddRequirement("added-meanwhile", false, false, releasetrace.TestRef{ID: "t-added-meanwhile"})
		}
	}
	if len(all) != total+1 || all[len(all)-1]["id"] != "added-meanwhile" {
		t.Fatalf("the pages hold %d of %d", len(all), total+1)
	}
	for index, row := range all {
		if row["target"] != requirementTarget(index+1) {
			t.Fatalf("row %d is %v", index, row["target"])
		}
	}
	again := reqRows(f.requirementsRead(root, "?limit=50"))
	fresh := evRows(evMap(f.evidenceRead(root).body, "requirements", "requirements"))
	if !reflect.DeepEqual(again, fresh) {
		t.Error("the first page of the further read differs from the rows of the main read")
	}
	if below := f.requirementsRead(root, fmt.Sprintf("?after=%d", total+1)); below.status != 200 || len(reqRows(below)) != 0 || below.body["next"] != "" {
		t.Errorf("after the last requirement: %s", below.raw)
	}
}

func TestRequirementsSearchStateAndRetired(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	want := map[string][]string{}
	for index, k := range f.s.ev.requirements {
		v, err := f.s.trace.Requirement(k, f.s.ev.candidate.Build)
		if err != nil {
			t.Fatal(err)
		}
		number := fmt.Sprint(index + 1)
		want["state="+v.State] = append(want["state="+v.State], number)
		retired := "retired=no"
		if v.Retired {
			retired = "retired=yes"
		}
		want[retired] = append(want[retired], number)
		want["state="+v.State+"&"+retired] = append(want["state="+v.State+"&"+retired], number)
	}
	for _, state := range requirementStates {
		for _, query := range []string{"state=" + state, "state=" + state + "&retired=no", "state=" + state + "&retired=yes"} {
			got := f.requirementsRead(root, "?"+query)
			if got.status != 200 || reqTargets(reqRows(got)) != strings.Join(want[query], ",") || whole(got.body["matching"]) != len(want[query]) || whole(got.body["total"]) != len(f.s.ev.requirements) {
				t.Errorf("%s: %s want %v", query, reqTargets(reqRows(got)), want[query])
			}
			for _, row := range reqRows(got) {
				if row["state"] != state {
					t.Errorf("%s lists a row in state %v", query, row["state"])
				}
			}
		}
	}
	if len(want["state=tested"]) == 0 || len(want["state=designed"]) == 0 || len(want["retired=no"]) != len(f.s.ev.requirements)-len(want["retired=yes"]) {
		t.Fatalf("the sample does not exercise the filters: %v", want)
	}
	for query, targets := range map[string]string{"q=SAMPLE-APPROVAL": "2", "q=erp": "4", "q=sample%2Fsample-recovery": "3", "q=no-such-thing": ""} {
		got := f.requirementsRead(root, "?"+query)
		if got.status != 200 || reqTargets(reqRows(got)) != targets {
			t.Errorf("%s: %s %s", query, reqTargets(reqRows(got)), got.raw[:min(len(got.raw), 200)])
		}
	}
	paged := f.requirementsRead(root, "?retired=no&limit=2")
	rest := f.requirementsRead(root, "?retired=no&limit=50&after="+fmt.Sprint(paged.body["next"]))
	if joined := reqTargets(append(reqRows(paged), reqRows(rest)...)); joined != strings.Join(want["retired=no"], ",") || whole(paged.body["matching"]) != len(want["retired=no"]) ||
		whole(rest.body["matching"]) != len(want["retired=no"]) || rest.body["next"] != "" {
		t.Errorf("a filtered list in pages: %s", joined)
	}
}

func TestRequirementsByTargetAndRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.requirementsRead(root, "?target=trace-requirement-5,trace-requirement-2,trace-requirement-999")
	if got.status != 200 || reqTargets(reqRows(got)) != "5,2" || whole(got.body["matching"]) != 2 || got.body["next"] != "" {
		t.Fatalf("by target: %s", got.raw)
	}
	main := evRows(evMap(f.evidenceRead(root).body, "requirements", "requirements"))
	if !reflect.DeepEqual(reqRows(got)[1], main[1]) || !reflect.DeepEqual(reqRows(got)[0], main[4]) {
		t.Error("a row read by target differs from the row of the main read")
	}
	f.code(t, "member", f.requirementsRead(f.as(MemberLogin), ""), 403, "permission_denied")
	f.code(t, "anonymous", f.requirementsRead(actor{}, ""), 401, "unauthenticated")
	same := func(name string, a actor) {
		t.Helper()
		theirs := evRows(evMap(f.evidenceRead(a).body, "requirements", "requirements"))
		mine := f.requirementsRead(a, "?target=trace-requirement-6,trace-requirement-2")
		if mine.status != 200 || len(theirs) != 6 || !reflect.DeepEqual(reqRows(mine)[0], theirs[5]) || !reflect.DeepEqual(reqRows(mine)[1], theirs[1]) {
			t.Errorf("%s: the rows read by target differ from the rows of the main read", name)
		}
	}
	f.setPassword(Admin2Login, pw)
	same("a second administrator", f.as(Admin2Login))
	many := []string{}
	for n := 1; n <= requirementTargets+1; n++ {
		many = append(many, requirementTarget(n))
	}
	for _, query := range []string{"?target=", "?target=trace-requirement-0", "?target=trace-requirement-01", "?target=requirement-1", "?target=trace-requirement-1,trace-requirement-1",
		"?target=trace-requirement-1,", "?target=" + strings.Join(many, ","), "?target=trace-requirement-1&limit=5", "?target=trace-requirement-1&q=a", "?target=trace-requirement-1&target=trace-requirement-2",
		"?after=0", "?after=x", "?after=01", "?limit=0", "?limit=51", "?limit=5x", "?q=", "?q=%20", "?state=retired", "?state=", "?retired=maybe", "?retired=", "?order=newest", "?limit=5&limit=6"} {
		if refused := f.requirementsRead(root, query); refused.status != 422 {
			t.Errorf("%s: %d", query, refused.status)
		}
	}
	if twenty := f.requirementsRead(root, "?target="+strings.Join(many[:requirementTargets], ",")); twenty.status != 200 || len(reqRows(twenty)) != len(f.s.ev.requirements) {
		t.Errorf("twenty targets: %d", twenty.status)
	}
	for _, path := range []string{"/api/v1/installation/proposals", "/api/v1/integrations/proposals", "/api/v1/recovery/proposals", "/api/v1/organizations/plans/closed"} {
		for _, query := range []string{"?state=tested", "?retired=no", "?target=trace-requirement-1"} {
			if refused := f.readAs(root, path+query); refused.status != 422 {
				t.Errorf("%s%s: %d", path, query, refused.status)
			}
		}
	}
	f.s.fault = func(at string) bool { return at == "release-evidence-read" }
	if failed := f.requirementsRead(root, ""); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.ev.requirements = append(f.s.ev.requirements, evKey("never-recorded"))
	if failed := f.requirementsRead(root, "?q=never"); failed.status != 503 {
		t.Errorf("an unknown stored requirement under a search: %d", failed.status)
	}
	if failed := f.requirementsRead(root, "?target="+requirementTarget(len(f.s.ev.requirements))); failed.status != 503 {
		t.Errorf("an unknown stored requirement by target: %d", failed.status)
	}
}

func TestRequirementsWithoutACandidateAndOutsideTheNewestBuilds(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	candidate := f.s.ev.candidate
	f.s.ev.candidate = nil
	if got := f.requirementsRead(root, "?state=tested"); got.status != 422 {
		t.Errorf("a state filter without a candidate: %d", got.status)
	}
	bare := f.requirementsRead(root, "?retired=no&q=sample")
	if bare.status != 200 || len(reqRows(bare)) == 0 || reqRows(bare)[0]["state"] != nil {
		t.Errorf("without a candidate the rows carry no state: %s", bare.raw[:min(len(bare.raw), 300)])
	}
	f.s.ev.candidate = candidate
	for n := 0; n < evidenceStatesShown+2; n++ {
		f.s.ev.builds = append(f.s.ev.builds, evidenceBuildRecord{id: fmt.Sprintf("later-%02d", n), commit: sampleHash(fmt.Sprint("later", n))})
	}
	got := f.requirementsRead(root, "?state=tested")
	if got.status != 200 || len(reqRows(got)) == 0 {
		t.Fatalf("with later builds: %s", got.raw[:min(len(got.raw), 300)])
	}
	for _, row := range reqRows(got) {
		states, _ := row["states"].(map[string]any)
		if _, listed := states[candidate.Build]; listed || row["state"] != releasetrace.StateTested {
			t.Errorf("the state at the candidate is given although the candidate is not among the builds listed: %v %v", row["state"], states)
		}
	}
}

func TestRequirementsAtTheModuleLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("walks ten thousand requirements")
	}
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for i := len(f.s.ev.requirements); i < 10000; i++ {
		f.evAddRequirement(fmt.Sprintf("bulk-%05d", i), false, false, releasetrace.TestRef{ID: fmt.Sprintf("t-bulk-%05d", i)})
	}
	started := time.Now()
	got := f.requirementsRead(root, "?q=bulk-09999")
	searched := time.Since(started)
	if got.status != 200 || reqTargets(reqRows(got)) != "10000" || whole(got.body["matching"]) != 1 || whole(got.body["total"]) != 10000 {
		t.Fatalf("a search at the limit: %d %s", got.status, reqTargets(reqRows(got)))
	}
	started = time.Now()
	page := f.requirementsRead(root, "?after=9990&limit=10")
	paged := time.Since(started)
	if page.status != 200 || len(reqRows(page)) != 10 || page.body["next"] != "" {
		t.Fatalf("the last page at the limit: %d", page.status)
	}
	t.Logf("10,000 requirements: a search walking all of them took %s; a page of ten rows took %s", searched, paged)
}

func TestRequirementRowsForAReaderWithoutAnEligibleApprover(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withoutThirdAdmin()
	root := f.as(AdminLogin)
	before := fmt.Sprint(reqRows(f.requirementsRead(root, "?target=trace-requirement-2"))[0]["retire_blocked"])
	if got := f.run(root, "identity.suspend_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	alone := reqRows(f.requirementsRead(root, "?target=trace-requirement-2"))[0]
	if before == "no_eligible_approver" || alone["retire_blocked"] != "no_eligible_approver" || alone["can_retire"] != false {
		t.Errorf("before %s, then %v", before, alone["retire_blocked"])
	}
	theirs := evRows(evMap(f.evidenceRead(root).body, "requirements", "requirements"))
	mine := reqRows(f.requirementsRead(root, "?target=trace-requirement-6,trace-requirement-2"))
	if len(theirs) != 6 || !reflect.DeepEqual(mine[0], theirs[5]) || !reflect.DeepEqual(mine[1], theirs[1]) {
		t.Error("the rows read by target differ from the rows of the main read")
	}
}
