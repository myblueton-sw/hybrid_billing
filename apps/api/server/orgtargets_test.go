package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func (f world) targetsRead(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations/targets"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func targetCounts(got reply) string {
	out := []string{}
	for _, row := range got.body["targets"].([]any) {
		t := row.(map[string]any)
		out = append(out, fmt.Sprintf("%s=%v", strings.TrimPrefix(t["id"].(string), "sample-"), t["members_on"]))
	}
	return strings.Join(out, ",")
}

func TestTargetsRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.targetsRead(root, "?kind=view_group")
	if got.status != 200 || targetCounts(got) != "view-north=1,view-south=2" || got.body["kind"] != "view_group" || got.body["on"] != "2026-11-16" ||
		fmt.Sprint(got.body["relationship_revision"]) != "1" || fmt.Sprint(got.body["active_revision"]) != "1" || got.body["next"] != "" || got.body["scan_stopped"] != false || got.body["revision"] != f.s.Revision() {
		t.Fatalf("view groups %+v", got.body)
	}
	if name := got.body["targets"].([]any)[0].(map[string]any)["name"]; name != "North view group (sample)" {
		t.Errorf("name %v", name)
	}
	for query, want := range map[string]string{
		"?kind=hierarchy&on=2026-09-30":             "le-north=2,le-south=1,unit-north-data=1,unit-north-ops=2,unit-south-web=3",
		"?kind=hierarchy&on=2026-10-01":             "le-north=2,le-south=1,unit-north-data=2,unit-north-ops=1,unit-south-web=3",
		"?kind=hierarchy&on=2025-12-31":             "le-north=0,le-south=0,unit-north-data=0,unit-north-ops=0,unit-south-web=0",
		"?kind=hierarchy&q=SOUTH":                   "le-south=1,unit-south-web=3",
		"?kind=hierarchy&q=data%20platform":         "unit-north-data=2",
		"?kind=allocation_pool":                     "pool-shared=2",
		"?kind=contract_pool":                       "",
		"?kind=payer&limit=1":                       "le-north=3",
		"?kind=payer&limit=1&after=sample-le-north": "le-south=3",
		"?kind=view_group&revision=0":               "",
		"?kind=view_group&revision=1":               "view-north=1,view-south=2",
	} {
		if got := f.targetsRead(root, query); got.status != 200 || targetCounts(got) != want {
			t.Errorf("%s: %d %s; want %s", query, got.status, targetCounts(got), want)
		}
	}
	if one := f.targetsRead(root, "?kind=payer&limit=1"); one.body["next"] != "sample-le-north" || one.body["scan_stopped"] != false {
		t.Errorf("next %v", one.body["next"])
	}
	if last := f.targetsRead(root, "?kind=payer&limit=2"); last.body["next"] != "" {
		t.Errorf("an exact page %v", last.body["next"])
	}
	f.s.cfg.Org.FilterScan = 3
	stopped := f.targetsRead(root, "?kind=hierarchy&q=web")
	if targetCounts(stopped) != "" || stopped.body["scan_stopped"] != true || stopped.body["next"] != "sample-unit-north-data" {
		t.Errorf("a stopped scan %+v", stopped.body)
	}
	if rest := f.targetsRead(root, "?kind=hierarchy&q=web&after=sample-unit-north-data"); targetCounts(rest) != "unit-south-web=3" || rest.body["scan_stopped"] != false || rest.body["next"] != "" {
		t.Errorf("the resumed scan %+v", rest.body)
	}
	f.s.cfg.Org.FilterScan = 5
	if exact := f.targetsRead(root, "?kind=hierarchy&q=zzz"); exact.body["scan_stopped"] != false || exact.body["next"] != "" {
		t.Errorf("a budget used up at the end %+v", exact.body)
	}
	f.s.cfg.Org.FilterScan = 1000
	for _, q := range []string{"", "?kind=", "?kind=other", "?kind=payer&limit=0", "?kind=payer&limit=100", "?kind=payer&revision=2", "?kind=payer&on=x", "?kind=payer&after=bad%20id",
		"?kind=payer&q=" + strings.Repeat("q", 65), "?kind=payer&member=sample-acct-01", "?kind=payer&subject=account", "?kind=payer&entity=sample-le-north", "?kind=payer&unattributed=true",
		"?kind=payer&target=sample-le-north"} {
		f.code(t, q, f.targetsRead(root, q), 422, string(commands.InvalidRequest))
	}
	for _, q := range []string{"?kind=payer&provider=aws", "?provider=", "?kind=other&provider=x"} {
		got := f.targetsRead(root, q)
		f.code(t, q, got, 501, string(commands.UnsupportedCapability))
		if got.body["retry"] != false || got.body["next_action"] != "none" {
			t.Errorf("%s: %+v", q, got.body)
		}
	}
	f.code(t, "provider without a session", f.targetsRead(actor{}, "?provider=aws"), 401, string(commands.Unauthenticated))
	f.code(t, "member", f.targetsRead(f.as(MemberLogin), "?kind=payer"), 403, string(commands.PermissionDenied))
	f.code(t, "no session", f.targetsRead(actor{}, "?kind=payer"), 401, string(commands.Unauthenticated))
	f.s.fault = func(at string) bool { return at == "organizations" }
	f.code(t, "fault", f.targetsRead(root, "?kind=payer"), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
}

func TestTargetsAcrossWorkspacesAndFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if _, err := f.s.orgs.PutWorkspace(organizations.Workspace{ID: "other-workspace", Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	day := sampleDay(2026, 11, 16)
	other, err := f.s.readTargets("other-workspace", orgQuery{on: day, kind: relationships.ViewGroup, limit: 50})
	if err != nil || len(other.Targets) != 0 || other.Relationship != 0 {
		t.Errorf("the other workspace %+v %v", other, err)
	}
	for name, q := range map[string]orgQuery{"listing": {on: day, kind: relationships.ViewGroup, limit: 50}, "count": {kind: relationships.ViewGroup, limit: 50}} {
		ws := "bad ws"
		if name == "count" {
			ws = SampleWorkspace
		}
		if rows, next, stopped, err := f.s.targetsPage(ws, 1, q, 10); rows != nil || next != "" || stopped || err == nil {
			t.Errorf("a failed %s %v %q %v %v", name, rows, next, stopped, err)
		}
	}
	if _, err := f.s.readTargets("bad ws", orgQuery{on: day, kind: relationships.ViewGroup, limit: 50, pinned: true, revision: 1}); err == nil || errors.Is(err, errInvalidQuery) {
		t.Errorf("an unreadable active revision with a pin %v", err)
	}
	if _, err := f.s.readTargets("bad ws", orgQuery{on: day, kind: relationships.ViewGroup, limit: 50}); err == nil {
		t.Error("a bad workspace")
	}
	if _, err := f.s.readTargets(SampleWorkspace, orgQuery{kind: relationships.ViewGroup, limit: 50}); !errors.Is(err, relationships.ErrInvalid) {
		t.Errorf("a member count failure %v", err)
	}
	if _, err := f.s.readTargets(SampleWorkspace, orgQuery{on: day, kind: "other", limit: 50}); !errors.Is(err, relationships.ErrInvalid) {
		t.Errorf("a listing failure %v", err)
	}
	if _, err := f.s.readTargets(SampleWorkspace, orgQuery{on: day, kind: relationships.ViewGroup, limit: 50, pinned: true, revision: 2}); !errors.Is(err, errInvalidQuery) {
		t.Errorf("a future revision %v", err)
	}
}

func TestConcurrentFirstReadsOfANewRevision(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	extra := []relationships.Timeline{}
	for i := 0; i < 150; i++ {
		extra = append(extra, openLine(relationships.ViewGroup, fmt.Sprintf("sample-acct-x%03d", i), "sample-view-north"))
	}
	revision := f.activate("many", extra...)
	if _, err := f.s.orgs.PutWorkspace(organizations.Workspace{ID: "other-workspace", Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	got := make([]string, 24)
	for i := range got {
		wait.Add(1)
		go func() {
			defer wait.Done()
			page := f.orgRead(root, "?subject=account&kind=view_group&limit=3&after=sample-acct-x100")
			targets := f.targetsRead(root, "?kind=view_group")
			empty, err := f.s.readOrganizations("other-workspace", orgQuery{on: sampleDay(2026, 11, 16), kind: relationships.ViewGroup, limit: 3, filter: orgFilter{subject: SubjectAccount}})
			got[i] = memberIDs(page) + " " + targetCounts(targets) + " " + fmt.Sprint(len(empty.Members), err == nil, page.body["relationship_revision"])
		}()
	}
	wait.Wait()
	for i, line := range got {
		if line != "acct-x101,acct-x102,acct-x103 view-north=151,view-south=2 0 true "+fmt.Sprint(revision) {
			t.Fatalf("reader %d %s", i, line)
		}
	}
}
