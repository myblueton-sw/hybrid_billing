package server

import (
	"fmt"
	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"net/url"
	"strings"
	"testing"
	"time"
)

func listIDs(got reply) []string {
	out := []string{}
	for _, raw := range got.body["rows"].([]any) {
		out = append(out, fmt.Sprint(raw.(map[string]any)["id"]))
	}
	return out
}

func wholeIDs(got reply, field string, keep func(map[string]any) bool) []string {
	out := []string{}
	for _, raw := range got.body[field].([]any) {
		if row := raw.(map[string]any); keep == nil || keep(row) {
			out = append(out, fmt.Sprint(row["id"]))
		}
	}
	return out
}

func walkList(t *testing.T, f world, a actor, path, query string) []string {
	t.Helper()
	out := []string{}
	next := ""
	for pages := 0; pages < 100; pages++ {
		at := path + "?limit=1" + query
		if next != "" {
			at += "&after=" + url.QueryEscape(next)
		}
		got := f.readAs(a, at)
		if got.status != 200 {
			t.Fatalf("page %d: %s", pages, got.raw)
		}
		out = append(out, listIDs(got)...)
		if next, _ = got.body["next"].(string); next == "" {
			return out
		}
	}
	t.Fatal("the walk did not end")
	return nil
}

const requestsPath, revocationsPath = "/api/v1/identity/access/requests", "/api/v1/identity/access/revocations"

func TestAccessRequestsReadEqualsTheWholeReadsRequests(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, f.grantVia(root, second, SampleMember, nil)); got.status != 202 {
		t.Fatalf("revoke %s", got.raw)
	}
	if got := f.run(root, "access.request_revoke", f.grantVia(root, second, SampleMember, nil), nil); got.status != 200 {
		t.Fatalf("ask to revoke %s", got.raw)
	}
	whole := f.accessRead(root, "")
	pending := wholeIDs(whole, "requests", func(r map[string]any) bool { return r["state"] == "pending" })
	ended := wholeIDs(whole, "requests", func(r map[string]any) bool { return r["state"] != "pending" })
	if len(pending) < 1 || len(ended) < 3 {
		t.Fatalf("fixture: %d pending, %d ended", len(pending), len(ended))
	}
	if got := f.readAs(root, requestsPath+"?state=pending&limit=50"); strings.Join(listIDs(got), " ") != strings.Join(pending, " ") || got.body["order"] != requestsOrder || got.body["next"] != "" {
		t.Errorf("pending: %v want %v", listIDs(got), pending)
	}
	if got := f.readAs(root, requestsPath+"?state=ended&limit=50"); strings.Join(listIDs(got), " ") != strings.Join(ended, " ") {
		t.Errorf("ended: %v want %v", listIDs(got), ended)
	}
	all := f.readAs(root, requestsPath+"?limit=50")
	if part := f.readAs(root, requestsPath+"?limit=1"); part.body["next"] == "" || part.body["count"].(map[string]any)["certainty"] != "lower_bound" || all.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Errorf("a part of the list states its count as %v, the whole as %v", part.body["count"], all.body["count"])
	}
	if len(listIDs(all)) != len(pending)+len(ended) || all.body["count"].(map[string]any)["value"] != fmt.Sprint(len(pending)+len(ended)) {
		t.Errorf("all: %v", listIDs(all))
	}
	if walked := walkList(t, f, root, requestsPath, ""); strings.Join(walked, " ") != strings.Join(listIDs(all), " ") {
		t.Errorf("one at a time: %v want %v", walked, listIDs(all))
	}
	rows := map[string]map[string]any{}
	for _, raw := range whole.body["requests"].([]any) {
		rows[fmt.Sprint(raw.(map[string]any)["id"])] = raw.(map[string]any)
	}
	for _, raw := range all.body["rows"].([]any) {
		row := raw.(map[string]any)
		want := rows[fmt.Sprint(row["id"])]
		for key, value := range want {
			if fmt.Sprint(row[key]) != fmt.Sprint(value) {
				t.Errorf("%v %s: %v want %v", row["id"], key, row[key], value)
			}
		}
		if row["proposer_login"] == "" || row["subject_login"] == "" {
			t.Errorf("%v has no sign-in names: %v", row["id"], row)
		}
	}
	other := f.readAs(second, requestsPath+"?state=pending&limit=50")
	mine := f.accessRead(second, "")
	for _, raw := range other.body["rows"].([]any) {
		row := raw.(map[string]any)
		for _, w := range mine.body["requests"].([]any) {
			if want := w.(map[string]any); want["id"] == row["id"] && (want["can_approve"] != row["can_approve"] || want["can_withdraw"] != row["can_withdraw"] || want["can_reject"] != row["can_reject"]) {
				t.Errorf("%v: eligibility differs from the whole read for the second administrator", row["id"])
			}
		}
	}
	if got := f.readAs(root, requestsPath+"?limit=50&q="+strings.ToUpper(pending[0])); len(listIDs(got)) != 1 || listIDs(got)[0] != pending[0] {
		t.Errorf("search by identifier: %v", listIDs(got))
	}
	shown := f.s.cfg.Access.EndedShown
	f.s.cfg.Access.EndedShown = 0
	if got := f.readAs(root, requestsPath+"?limit=50"); strings.Join(listIDs(got), " ") != strings.Join(pending, " ") {
		t.Errorf("with no ended requests shown: %v want %v", listIDs(got), pending)
	}
	f.s.cfg.Access.EndedShown = shown
}

func TestAccessRevocationsReadEqualsTheWholeReadsRevocations(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	for range 2 {
		if got := f.revokeVia(root, second, f.grantVia(root, second, SampleMember, nil)); got.status != 202 {
			t.Fatalf("revoke %s", got.raw)
		}
	}
	whole := f.accessRead(root, "")
	want := wholeIDs(whole, "revocations", nil)
	if len(want) < 2 {
		t.Fatalf("fixture: %d revocations", len(want))
	}
	all := f.readAs(root, revocationsPath+"?limit=50")
	got := listIDs(all)
	if len(got) != len(want) || all.body["order"] != revocationsOrder {
		t.Fatalf("revocations: %v want the same set as %v", got, want)
	}
	seen := map[string]map[string]any{}
	for _, raw := range whole.body["revocations"].([]any) {
		seen[fmt.Sprint(raw.(map[string]any)["id"])] = raw.(map[string]any)
	}
	previous := ""
	for _, raw := range all.body["rows"].([]any) {
		row := raw.(map[string]any)
		if fmt.Sprint(row) != fmt.Sprint(seen[fmt.Sprint(row["id"])]) {
			t.Errorf("%v differs from the whole read", row["id"])
		}
		if at := fmt.Sprint(row["recorded_at"]); previous != "" && at > previous {
			t.Errorf("not newest first: %s after %s", at, previous)
		} else {
			previous = at
		}
	}
	for _, raw := range all.body["rows"].([]any) {
		row := raw.(map[string]any)
		if row["target_subject"] != SampleMember || row["target_subject_login"] != MemberLogin || row["target_action"] != "view" || row["target_resource_kind"] != ReaderKind ||
			fmt.Sprint(row["target_fields"]) != "["+ReaderField+"]" || row["target_kind"] != "grant" || row["target_present"] != true {
			t.Errorf("%v: the target's facts: %v", row["id"], row)
		}
	}
	f.manyPermissions()
	if got := f.run(root, "access.revoke_delegation", "qd-02", nil); got.status != 202 && got.status != 200 {
		t.Fatalf("revoke a delegation %s", got.raw)
	}
	newest := f.readAs(root, revocationsPath+"?limit=1").body["rows"].([]any)[0].(map[string]any)
	if newest["target_kind"] != "delegation" || newest["target"] != "qd-02" || newest["target_subject"] != SampleAdmin2 || newest["target_subject_login"] != Admin2Login ||
		newest["target_action"] != "view" || newest["target_resource_kind"] != ReaderKind || fmt.Sprint(newest["target_fields"]) != "["+ReaderField+"]" {
		t.Errorf("a revoked delegation's facts name its delegate: %v", newest)
	}
	for _, raw := range f.accessRead(root, "").body["revocations"].([]any) {
		if row := raw.(map[string]any); row["id"] == newest["id"] && fmt.Sprint(row) != fmt.Sprint(newest) {
			t.Error("the whole read's row for the delegation differs")
		}
	}
	for _, kind := range []string{accesschange.TargetGrant, accesschange.TargetDelegation} {
		gone := f.s.revocationRow(accesschange.RevocationView{ID: "revocation-gone", Target: accesschange.Target{Kind: kind, ID: "no-longer-here"}, History: []accesschange.Transition{{State: "received"}}})
		if gone.TargetSubject != "" || gone.TargetLogin != "" || gone.TargetAction != "" || gone.TargetResource != "" || gone.TargetFields == nil || len(gone.TargetFields) != 0 {
			t.Errorf("a %s that is no longer present: %+v", kind, gone)
		}
	}
	got = listIDs(f.readAs(root, revocationsPath+"?limit=50"))
	if walked := walkList(t, f, root, revocationsPath, ""); strings.Join(walked, " ") != strings.Join(got, " ") {
		t.Errorf("one at a time: %v want %v", walked, got)
	}
	oldest := got[len(got)-1]
	target := fmt.Sprint(seen[oldest]["target"])
	if found := f.readAs(root, revocationsPath+"?limit=50&q="+target); len(listIDs(found)) != 1 || listIDs(found)[0] != oldest {
		t.Errorf("search by target %q: %v", target, listIDs(found))
	}
}

func TestAccessListsRefusalsAdmissionAndCursors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, f.grantVia(root, second, SampleMember, nil)); got.status != 202 {
		t.Fatalf("revoke %s", got.raw)
	}
	for _, path := range []string{requestsPath, revocationsPath} {
		for _, query := range []string{"?unknown=1", "?q=", "?q=%01", "?limit=0", "?limit=x", "?after=", "?q=a&q=b", "?state=nonsense", "?subject=a"} {
			if got := f.readAs(root, path+query); got.status != 422 {
				t.Errorf("%s%s: %d", path, query, got.status)
			}
		}
		f.code(t, "a member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
		f.code(t, "no session", f.readAs(actor{}, path), 401, "unauthenticated")
	}
	if got := f.readAs(root, revocationsPath+"?state=pending"); got.status != 422 {
		t.Errorf("revocations take no state: %d", got.status)
	}
	conflict := func(name string, got reply) {
		t.Helper()
		if got.status != 409 || got.body["code"] != "revision_conflict" {
			t.Errorf("%s: %d %s", name, got.status, got.raw[:min(len(got.raw), 140)])
		}
	}
	first := f.readAs(root, requestsPath+"?limit=1")
	token := url.QueryEscape(first.body["next"].(string))
	if got := f.readAs(root, requestsPath+"?limit=1&after="+token); got.status != 200 {
		t.Fatalf("the cursor with its own query: %s", got.raw)
	}
	conflict("another filter", f.readAs(root, requestsPath+"?limit=1&state=pending&after="+token))
	conflict("another administrator", f.readAs(second, requestsPath+"?limit=1&after="+token))
	conflict("the revocations list", f.readAs(root, revocationsPath+"?limit=1&after="+token))
	conflict("a subjects cursor", f.readAs(root, requestsPath+"?limit=1&after="+url.QueryEscape(f.subjects(root, "?limit=1").body["next"].(string))))
	if got := f.requestGrant(root, SampleMember, nil); got.status != 200 {
		t.Fatalf("propose %s", got.raw)
	}
	conflict("after a new request", f.readAs(root, requestsPath+"?limit=1&after="+token))
	fresh := url.QueryEscape(f.readAs(root, requestsPath+"?limit=1").body["next"].(string))
	if got := f.run(root, "identity.suspend_membership", SampleAdmin2, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	conflict("after a membership changed", f.readAs(root, requestsPath+"?limit=1&after="+fresh))
	again := url.QueryEscape(f.readAs(root, requestsPath+"?limit=1").body["next"].(string))
	f.advance(time.Duration(f.s.cfg.Access.ServiceQuerySeconds+1) * time.Second)
	conflict("after the cursor's lifetime", f.readAs(f.as(AdminLogin), requestsPath+"?limit=1&after="+again))
	f.s.fault = func(at string) bool { return at == "access-query" }
	for _, path := range []string{requestsPath, revocationsPath} {
		if got := f.readAs(f.as(AdminLogin), path); got.status != 503 {
			t.Errorf("%s under a fault: %d", path, got.status)
		}
	}
}
