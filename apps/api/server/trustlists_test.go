package server

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

const trustChangesPath = "/api/v1/identity/policy/changes"

func (f world) trustMixture(t *testing.T) admins {
	t.Helper()
	p := f.trustAdmins()
	cancelled := changeOf(t, f.trustPropose(p.a, "50", "0"))
	if got := f.trustEnd(p.a, "trust.cancel", cancelled); got.status != 200 {
		t.Fatalf("cancel %s", got.raw)
	}
	rejected := changeOf(t, f.trustPropose(p.a, "51", "0"))
	if got := f.trustEnd(p.b, "trust.reject", rejected); got.status != 200 {
		t.Fatalf("reject %s", got.raw)
	}
	changeOf(t, f.trustPropose(p.b, "52", "0"))
	if got := f.trustActivate(p.c, f.approvedChange(p, "60", "10")); got.status != 200 {
		t.Fatalf("activate %s", got.raw)
	}
	p = admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
	changeOf(t, f.trustPropose(p.a, "53", "0"))
	validated := changeOf(t, f.trustPropose(p.b, "54", "0"))
	if got := f.trustValidate(p.c, validated); got.status != 200 {
		t.Fatalf("validate %s", got.raw)
	}
	f.approvedChange(p, "55", "0")
	return p
}

func wholeChanges(t *testing.T, f world, a actor) []map[string]any {
	t.Helper()
	got := f.policyRead(a)
	if got.status != 200 {
		t.Fatalf("policy read %s", got.raw)
	}
	out := []map[string]any{}
	for _, raw := range got.body["changes"].([]any) {
		out = append(out, raw.(map[string]any))
	}
	return out
}

func listedOf(t *testing.T, whole map[string]any) map[string]any {
	t.Helper()
	state := fmt.Sprint(whole["state"])
	want := map[string]any{"validation_result": "", "evidence_current": evidenceNotApplicable, "invalidation_state": "", "decide_in_detail": trustchange.Open(state)}
	for _, name := range []string{"id", "number", "state", "change_revision", "old_epoch", "old_policy", "candidate", "proposer", "validator", "approver", "activator", "superseded_by", "earlier_run", "run_started_at", "act_until"} {
		want[name] = whole[name]
	}
	if v, ok := whole["validation"].(map[string]any); ok {
		want["validation_result"] = v["result"]
		if trustchange.Open(state) {
			want["evidence_current"] = evidenceNotEvaluated
		}
	}
	if v, ok := whole["invalidation"].(map[string]any); ok {
		want["invalidation_state"] = v["state"]
	}
	can := whole["can"].(map[string]any)
	want["can"] = map[string]any{"revise": can["revise"], "validate": can["validate"], "reject": can["reject"], "cancel": can["cancel"]}
	why := map[string]any{}
	for name, reason := range whole["why_not"].(map[string]any) {
		if reason == "evidence_stale" {
			reason = ""
		}
		why[name] = reason
	}
	want["why_not"] = why
	return want
}

func TestTrustChangesReadEqualsThePolicyReadsChanges(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustMixture(t)
	seen := map[string]int{}
	for name, a := range map[string]actor{"proposer": p.a, "validator": p.b, "third": p.c} {
		whole := wholeChanges(t, f, a)
		if len(whole) != 7 {
			t.Fatalf("fixture has %d changes", len(whole))
		}
		got := f.readAs(a, trustChangesPath+"?limit=50")
		rows := got.body["rows"].([]any)
		if got.status != 200 || len(rows) != len(whole) || got.body["order"] != trustChangesOrder || got.body["next"] != "" {
			t.Fatalf("%s: %s", name, got.raw)
		}
		acts := 0
		for i, raw := range rows {
			want := listedOf(t, whole[i])
			if !reflect.DeepEqual(raw, want) {
				t.Errorf("%s row %d:\n got %v\nwant %v", name, i, raw, want)
			}
			seen[fmt.Sprint(whole[i]["state"])]++
			for _, allowed := range want["can"].(map[string]any) {
				if allowed == true {
					acts++
				}
			}
			detail := f.readAs(a, trustChangesPath+"/"+fmt.Sprint(whole[i]["id"]))
			if detail.status != 200 || !reflect.DeepEqual(detail.body["change"], any(whole[i])) || detail.body["revision"] != f.s.Revision() || detail.body["observed_at"] == "" {
				t.Errorf("%s detail %d: %s", name, i, detail.raw)
			}
		}
		if acts == 0 {
			t.Errorf("%s may do nothing in the fixture", name)
		}
		for filter, states := range map[string][]string{"open": trustChangeGroups["open"], "closed": trustChangeGroups["closed"], "draft": {"draft"}, "validated": {"validated"},
			"independently_approved": {"independently_approved"}, "activated": {"activated"}, "rejected": {"rejected"}, "cancelled": {"cancelled"}, "superseded": {"superseded"}, "activation_unknown": {"activation_unknown"}} {
			want := []string{}
			for _, row := range whole {
				if strings.Contains(" "+strings.Join(states, " ")+" ", " "+fmt.Sprint(row["state"])+" ") {
					want = append(want, fmt.Sprint(row["id"]))
				}
			}
			page := f.readAs(a, trustChangesPath+"?limit=50&state="+filter)
			count := page.body["count"].(map[string]any)
			if strings.Join(listIDs(page), " ") != strings.Join(want, " ") || count["value"] != fmt.Sprint(len(want)) || count["certainty"] != "exact" {
				t.Errorf("%s state %s: %v; want %v", name, filter, listIDs(page), want)
			}
			if walked := walkList(t, f, a, trustChangesPath, "&state="+filter); strings.Join(walked, " ") != strings.Join(want, " ") {
				t.Errorf("%s state %s by one: %v; want %v", name, filter, walked, want)
			}
		}
	}
	for _, state := range []string{"draft", "validated", "independently_approved", "activated", "rejected", "cancelled", "superseded"} {
		if seen[state] == 0 {
			t.Errorf("the fixture has no %s change", state)
		}
	}
	whole := wholeChanges(t, f, p.a)
	one := fmt.Sprint(whole[3]["id"])
	if got := f.readAs(p.a, trustChangesPath+"?q="+url.QueryEscape(strings.ToUpper(one))); strings.Join(listIDs(got), " ") != one {
		t.Errorf("search for one identifier: %s", got.raw)
	}
	if got := f.readAs(p.a, trustChangesPath+"?q=trust-&limit=2"); len(listIDs(got)) != 2 || got.body["next"] == "" || got.body["count"].(map[string]any)["value"] != "7" ||
		got.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Errorf("a first page of two: %s", got.raw)
	}
	if got := f.readAs(p.a, trustChangesPath+"?q=nothing"); got.status != 200 || len(listIDs(got)) != 0 || got.body["count"].(map[string]any)["value"] != "0" {
		t.Errorf("a search that matches nothing: %s", got.raw)
	}
}

func TestAListedChangeLeavesEvidenceToTheDetail(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.b, "90", "0"))
	if got := f.trustValidate(p.c, id); got.status != 200 {
		t.Fatal(got.raw)
	}
	fresh := f.readAs(p.a, trustChangesPath+"/"+id).body["change"].(map[string]any)
	if fresh["why_not"].(map[string]any)["approve"] != "" || fresh["can"].(map[string]any)["approve"] != true || fresh["impact"] == nil ||
		fresh["validation"].(map[string]any)["evidence_current"] != true {
		t.Fatalf("a validated change with current evidence %+v", fresh)
	}
	g, _ := f.s.grantView("trust-trust_validate-console")
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	row := f.readAs(p.a, trustChangesPath).body["rows"].([]any)[0].(map[string]any)
	if row["id"] != id || row["why_not"].(map[string]any)["approve"] != "" || row["evidence_current"] != evidenceNotEvaluated || row["decide_in_detail"] != true || row["validation_result"] != trustchange.ResultValidated {
		t.Errorf("the listed row %+v", row)
	}
	if _, answered := row["can"].(map[string]any)["approve"]; answered {
		t.Error("the listed row answers whether the caller may approve")
	}
	if _, carried := row["impact"]; carried {
		t.Error("the listed row carries the impact")
	}
	detail := f.readAs(p.a, trustChangesPath+"/"+id).body["change"].(map[string]any)
	if detail["why_not"].(map[string]any)["approve"] != "evidence_stale" || detail["can"].(map[string]any)["approve"] != false || detail["validation"].(map[string]any)["evidence_current"] != false {
		t.Errorf("the detail %+v", detail)
	}
	own := f.readAs(p.b, trustChangesPath).body["rows"].([]any)[0].(map[string]any)
	if own["why_not"].(map[string]any)["approve"] != "self" || own["can"].(map[string]any)["cancel"] != true || own["can"].(map[string]any)["reject"] != false {
		t.Errorf("the proposer's row %+v", own)
	}
}

func TestTrustChangeReadsRefuse(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustMixture(t)
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	known := listIDs(f.readAs(p.a, trustChangesPath))[0]
	for _, path := range []string{trustChangesPath, trustChangesPath + "/" + known} {
		f.code(t, "no session "+path, f.readAs(actor{}, path), 401, string(commands.Unauthenticated))
		f.code(t, "bad csrf "+path, f.readAs(actor{p.a.cookie, "x"}, path), 403, CodeCSRF)
		f.code(t, "member "+path, f.readAs(member, path), 403, string(commands.PermissionDenied))
	}
	for _, query := range []string{"state=pending", "state=", "state=open&state=closed", "limit=0", "limit=01", "limit=100000", "q=", "q=%00", "q=" + strings.Repeat("a", 4000), "after=", "after=x", "sort=number", "workspace=other"} {
		want := 422
		if query == "after=x" {
			want = 409
		}
		if got := f.readAs(p.a, trustChangesPath+"?"+query); got.status != want {
			t.Errorf("%s: %d %s", query, got.status, got.raw)
		}
	}
	unknown := f.readAs(p.a, trustChangesPath+"/trust-0a1b2c3d-999")
	f.code(t, "an unknown change", unknown, 422, string(commands.InvalidRequest))
	malformed := f.readAs(p.a, trustChangesPath+"/"+url.PathEscape("a b"))
	if malformed.status != unknown.status || malformed.body["code"] != unknown.body["code"] {
		t.Errorf("a malformed identifier answers %s, an unknown one %s", malformed.raw, unknown.raw)
	}
	f.code(t, "a detail with a query", f.readAs(p.a, trustChangesPath+"/"+known+"?x=1"), 422, string(commands.InvalidRequest))

	first := f.readAs(p.a, trustChangesPath+"?limit=2")
	next := url.QueryEscape(fmt.Sprint(first.body["next"]))
	if first.status != 200 || next == "" {
		t.Fatalf("first page %s", first.raw)
	}
	second := f.readAs(p.a, trustChangesPath+"?limit=2&after="+next)
	if second.status != 200 || len(listIDs(second)) != 2 || listIDs(second)[0] == listIDs(first)[1] || second.body["reference_at"] != first.body["reference_at"] {
		t.Errorf("second page %s", second.raw)
	}
	f.code(t, "another filter", f.readAs(p.a, trustChangesPath+"?limit=2&state=open&after="+next), 409, string(commands.RevisionConflict))
	f.code(t, "another limit", f.readAs(p.a, trustChangesPath+"?limit=3&after="+next), 409, string(commands.RevisionConflict))
	f.code(t, "another session", f.readAs(p.b, trustChangesPath+"?limit=2&after="+next), 409, string(commands.RevisionConflict))
	f.code(t, "another list", f.readAs(p.a, requestsPath+"?limit=2&after="+next), 409, string(commands.RevisionConflict))
	changeOf(t, f.trustPropose(p.c, "70", "0"))
	f.code(t, "after a change was recorded", f.readAs(p.a, trustChangesPath+"?limit=2&after="+next), 409, string(commands.RevisionConflict))
	if again := f.readAs(p.a, trustChangesPath+"?limit=2"); again.status != 200 || again.body["count"].(map[string]any)["value"] != "8" {
		t.Errorf("a fresh read %s", again.raw)
	}
}

func TestThePolicyReadNamesAWaitingActivationAndEarlierRuns(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	if got := f.policyRead(p.a); got.body["waiting_activation"] != "" || got.body["earlier_run_kept"] != false {
		t.Errorf("an empty registry %s", got.raw)
	}
	changeOf(t, f.trustPropose(p.a, "50", "0"))
	if got := f.policyRead(p.a); got.body["waiting_activation"] != "" || got.body["earlier_run_kept"] != false {
		t.Errorf("a draft %s", got.raw)
	}
	g := setup(t, SeedSample).ready()
	id, _ := g.unknownActivation(t, "ledger-sync")
	a := g.signedIn(AdminLogin, pw)
	if got := g.policyRead(a); got.body["waiting_activation"] != id {
		t.Errorf("a waiting activation %v; want %s", got.body["waiting_activation"], id)
	}
	if got := g.readAs(a, trustChangesPath+"?state=open"); strings.Join(listIDs(got), " ") != id || got.body["rows"].([]any)[0].(map[string]any)["decide_in_detail"] != false {
		t.Errorf("the open group with a waiting activation %s", got.raw)
	}
	if err := g.s.trust.Restore([]trustchange.View{{ID: "trust-11111111-1", Workspace: SampleWorkspace, State: trustchange.StateActivated}}); err != nil {
		t.Fatal(err)
	}
	if got := g.policyRead(a); got.body["earlier_run_kept"] != true {
		t.Errorf("an earlier run's change %s", got.raw)
	}
	whole := wholeChanges(t, g, a)
	listed := g.readAs(a, trustChangesPath).body["rows"].([]any)
	if len(whole) != 2 || len(listed) != 2 || whole[0]["earlier_run"] != true || whole[1]["state"] != trustchange.StateUnknown || whole[1]["why_not"].(map[string]any)["activate"] != "trust_unavailable" {
		t.Fatalf("fixture %+v", whole)
	}
	for i, row := range whole {
		if want := listedOf(t, row); !reflect.DeepEqual(listed[i], any(want)) {
			t.Errorf("row %d on hold:\n got %v\nwant %v", i, listed[i], want)
		}
		if detail := g.readAs(a, trustChangesPath+"/"+fmt.Sprint(row["id"])); !reflect.DeepEqual(detail.body["change"], any(row)) {
			t.Errorf("detail %d on hold: %s", i, detail.raw)
		}
	}
}
