package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestRetentionOwnersAreTheListThePlaceHoldCommandChecks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withMember()
	root := f.as(AdminLogin)
	path := "/api/v1/retention/owners"
	listed := func(answer reply) []string {
		out := []string{}
		for _, raw := range answer.body["owners"].([]any) {
			row := raw.(map[string]any)
			out = append(out, fmt.Sprint(row["user_id"], " ", row["login"]))
		}
		return out
	}
	want := func() []string {
		access := f.accessRead(root, "")
		if access.status != 200 {
			t.Fatalf("the access read: %d", access.status)
		}
		out := []string{}
		for _, raw := range access.body["subjects"].([]any) {
			row := raw.(map[string]any)
			out = append(out, fmt.Sprint(row["user_id"], " ", row["login"]))
		}
		return out
	}
	all := f.readAs(root, path)
	everyone := want()
	if all.status != 200 || strings.Join(listed(all), "|") != strings.Join(everyone, "|") || fmt.Sprint(all.body["matching"]) != fmt.Sprint(len(everyone)) || all.body["more"] != false ||
		all.body["order"] != "identifier" || all.body["q"] != "" || len(everyone) < 3 {
		t.Fatalf("without a term the read lists the command's candidates in identifier order: %s", all.raw[:min(len(all.raw), 400)])
	}
	first := f.readAs(root, path+"?limit=1")
	if got := listed(first); len(got) != 1 || got[0] != everyone[0] || fmt.Sprint(first.body["matching"]) != fmt.Sprint(len(everyone)) || first.body["more"] != true {
		t.Errorf("a limit below the number that match: %s", first.raw[:min(len(first.raw), 300)])
	}
	if exact := f.readAs(root, path+"?limit="+fmt.Sprint(len(everyone))); len(listed(exact)) != len(everyone) || exact.body["more"] != false {
		t.Errorf("a limit equal to the number that match says more: %v", exact.body["more"])
	}
	byLogin := f.readAs(root, path+"?q="+strings.ToUpper(MemberLogin[:6]))
	if got := listed(byLogin); byLogin.status != 200 || len(got) == 0 || !strings.Contains(strings.Join(got, "|"), SampleMember+" "+MemberLogin) || byLogin.body["q"] != strings.ToUpper(MemberLogin[:6]) {
		t.Errorf("a search by sign-in name without regard to case: %s", byLogin.raw[:min(len(byLogin.raw), 300)])
	}
	if byID := f.readAs(root, path+"?q="+SampleMember); len(listed(byID)) == 0 || fmt.Sprint(byID.body["matching"]) == "0" {
		t.Errorf("a search by user identifier: %s", byID.raw[:min(len(byID.raw), 300)])
	}
	if none := f.readAs(root, path+"?q=no-such-person"); none.status != 200 || len(listed(none)) != 0 || fmt.Sprint(none.body["matching"]) != "0" || none.body["more"] != false {
		t.Errorf("a term nobody matches: %s", none.raw[:min(len(none.raw), 200)])
	}
	one := f.readAs(root, path+"?id="+SampleMember)
	if got := listed(one); len(got) != 1 || got[0] != SampleMember+" "+MemberLogin || fmt.Sprint(one.body["matching"]) != "1" {
		t.Errorf("one person by identifier: %s", one.raw[:min(len(one.raw), 300)])
	}
	unknown := f.readAs(root, path+"?id=never-a-member")
	if unknown.status != 200 || len(listed(unknown)) != 0 {
		t.Errorf("an unknown identifier: %s", unknown.raw[:min(len(unknown.raw), 200)])
	}
	for _, query := range []string{"?q=", "?q=" + strings.Repeat("a", 65), "?limit=0", "?limit=21", "?limit=x", "?id=", "?id=a,b", "?id=" + SampleMember + "&q=a", "?id=" + SampleMember + "&limit=5",
		"?id=" + SampleMember + "&id=" + SampleMember, "?q=a&q=b", "?after=1", "?state=x"} {
		if refused := f.readAs(root, path+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.ok(t, "suspend", f.run(root, "identity.suspend_user", SampleMember, nil), "user_suspended", "")
	after := want()
	gone := f.readAs(root, path+"?id="+SampleMember)
	bySearch := f.readAs(root, path+"?q="+SampleMember)
	rest := f.readAs(root, path)
	if len(after) != len(everyone)-1 || len(listed(gone)) != 0 || gone.raw[strings.Index(gone.raw, `"owners"`):] != unknown.raw[strings.Index(unknown.raw, `"owners"`):] ||
		strings.Contains(bySearch.raw, MemberLogin) || strings.Join(listed(rest), "|") != strings.Join(after, "|") {
		t.Errorf("a suspended member is offered, or answers differently from an unknown one: %s / %s", gone.raw[:min(len(gone.raw), 200)], bySearch.raw[:min(len(bySearch.raw), 200)])
	}
	if placed := f.run(root, "retention.place_hold", retentionTenantTarget+SampleWorkspace, map[string]string{"owner": SampleMember, "reason": "x"}); placed.status == 200 {
		t.Error("the command accepts an owner the read no longer offers")
	}
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = func(at string) bool { return at == "roster" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a roster that cannot be read: %d", failed.status)
	}
	reads := 0
	f.s.fault = func(at string) bool {
		if at == "roster" {
			reads++
		}
		return false
	}
	f.s.bootWorkspace = "elsewhere"
	for _, query := range []string{"", "?q=admin", "?id=" + SampleAdmin} {
		if out := f.readAs(root, path+query); out.status != 200 || len(out.body["owners"].([]any)) != 0 || fmt.Sprint(out.body["matching"]) != "0" || reads != 0 {
			t.Errorf("another workspace %q: %d, %d roster reads", query, out.status, reads)
		}
	}
}
