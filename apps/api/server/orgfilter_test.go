package server

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func memberIDs(got reply) string {
	ids := []string{}
	for _, m := range got.body["members"].([]any) {
		ids = append(ids, strings.TrimPrefix(m.(map[string]any)["member"].(string), "sample-"))
	}
	return strings.Join(ids, ",")
}

func (f world) activate(plan string, lines ...relationships.Timeline) uint64 {
	f.t.Helper()
	active, _ := f.s.rels.Active(SampleWorkspace)
	if err := f.s.rels.Draft(SampleWorkspace, plan, active); err != nil {
		f.t.Fatal(err)
	}
	if err := f.s.rels.AddChunk(SampleWorkspace, plan, 0, lines); err != nil {
		f.t.Fatal(err)
	}
	preview, err := f.s.rels.Seal(SampleWorkspace, plan, 1)
	if err != nil {
		f.t.Fatalf("seal %v %+v", err, preview.Violations)
	}
	number, err := f.s.rels.Activate(SampleWorkspace, plan, preview.Digest, active)
	if err != nil {
		f.t.Fatal(err)
	}
	return number
}

func openLine(kind relationships.Kind, member, target string) relationships.Timeline {
	return relationships.Timeline{Kind: kind, Member: member, Entries: []relationships.Entry{{Target: target, Interval: relationships.From(sampleDay(2026, 1, 1))}}}
}

func TestOrganizationFiltersOneByOne(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	want := map[string]string{
		"?entity=sample-le-north":                                 "acct-01,acct-02,acct-03,unit-north-data,unit-north-ops",
		"?entity=sample-le-north&on=2026-09-30":                   "acct-01,acct-02,acct-03,unit-north-data,unit-north-ops",
		"?entity=sample-le-north&subject=account":                 "acct-01,acct-02,acct-03",
		"?entity=sample-le-north&kind=payer":                      "acct-01,acct-02,acct-03",
		"?entity=sample-le-south&subject=party":                   "unit-south-web",
		"?entity=sample-le-seller":                                "",
		"?entity=sample-le-north&on=2025-12-31":                   "",
		"?subject=party":                                          "unit-north-data,unit-north-ops,unit-south-web",
		"?subject=party&kind=payer":                               "",
		"?subject=account":                                        "acct-01,acct-02,acct-03,acct-04,acct-05,acct-06",
		"?subject=any":                                            "acct-01,acct-02,acct-03,acct-04,acct-05,acct-06,unit-north-data,unit-north-ops,unit-south-web",
		"?unattributed=true":                                      "",
		"?unattributed=true&on=2025-12-31&subject=party":          "unit-north-data,unit-north-ops,unit-south-web",
		"?unattributed=true&kind=view_group&subject=account":      "acct-01,acct-02,acct-05,acct-06",
		"?unattributed=true&kind=view_group":                      "",
		"?unattributed=true&kind=allocation_pool&subject=account": "acct-01,acct-02,acct-04,acct-05",
		"?unattributed=true&kind=payer&subject=account":           "",
		"?unattributed=true&kind=contract_pool&subject=account":   "acct-01,acct-02,acct-03,acct-04,acct-05,acct-06",
		"?q=ACCT-0":            "acct-01,acct-02,acct-03,acct-04,acct-05,acct-06",
		"?q=north":             "unit-north-data,unit-north-ops",
		"?q=Data%20Platform":   "unit-north-data",
		"?q=account%2005":      "acct-05",
		"?q=nothing-like-this": "",
		"?kind=view_group&target=sample-view-south":                   "acct-03,acct-04",
		"?kind=view_group&target=sample-view-north":                   "acct-03",
		"?kind=view_group&target=sample-view-none":                    "",
		"?target=sample-unit-north-ops&on=2026-09-30":                 "acct-01,acct-02",
		"?target=sample-unit-north-ops&on=2026-10-01":                 "acct-01",
		"?target=sample-unit-north-data&on=2026-10-01":                "acct-02,acct-03",
		"?kind=hierarchy&member=sample-acct-03&member=sample-acct-01": "acct-01,acct-03",
	}
	for query, ids := range want {
		got := f.orgRead(root, query)
		if got.status != 200 || memberIDs(got) != ids {
			t.Errorf("%s: %d %s; want %s", query, got.status, memberIDs(got), ids)
		}
	}
	got := f.orgRead(root, "?entity=sample-le-north&subject=account&unattributed=true&q=acct&kind=view_group&on=2026-11-16")
	echo := got.body["filters"].(map[string]any)
	if memberIDs(got) != "acct-01,acct-02" || echo["subject"] != "account" || echo["entity"] != "sample-le-north" || echo["unattributed"] != true || echo["q"] != "acct" ||
		echo["member_count"] != 0.0 || echo["target"] != "" || got.body["provider_filter"] != "not_connected" || got.body["scan_stopped"] != false || got.body["scan_budget"] != 1000.0 {
		t.Errorf("combined %s %+v", memberIDs(got), got.body)
	}
	plain := f.orgRead(root, "")
	if e := plain.body["filters"].(map[string]any); e["subject"] != "any" || e["unattributed"] != false || plain.body["scan_stopped"] != false {
		t.Errorf("echo without filters %+v", e)
	}
	acct := members(f.orgRead(root, "?subject=account&kind=view_group"))
	if targetIDs(acct["sample-acct-03"]) != "sample-view-north,sample-view-south" || targetIDs(acct["sample-acct-01"]) != "" || len(acct["sample-acct-01"]["history"].([]any)) != 0 {
		t.Errorf("account rows without a view group timeline %+v", acct)
	}
}

func TestOrganizationFiltersCombined(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for query, ids := range map[string]string{
		"?subject=account&entity=sample-le-south&kind=allocation_pool&unattributed=true": "acct-04,acct-05",
		"?subject=account&entity=sample-le-south&kind=allocation_pool":                   "acct-04,acct-05,acct-06",
		"?kind=view_group&target=sample-view-south&q=04":                                 "acct-04",
		"?kind=view_group&target=sample-view-south&entity=sample-le-north":               "acct-03",
		"?kind=view_group&target=sample-view-south&unattributed=true":                    "",
		"?subject=party&entity=sample-le-north&q=ops":                                    "unit-north-ops",
		"?member=sample-acct-02&entity=sample-le-north&on=2026-09-30":                    "acct-02",
		"?member=sample-acct-02&target=sample-unit-north-ops&on=2026-10-01":              "",
		"?member=sample-acct-01&member=sample-unit-north-ops&subject=account":            "acct-01",
		"?member=sample-acct-01&subject=account&kind=view_group":                         "acct-01",
		"?member=sample-acct-01&kind=view_group":                                         "",
		"?member=sample-view-north&subject=account":                                      "",
		"?member=sample-unit-north-ops&subject=party":                                    "unit-north-ops",
		"?member=sample-acct-01&subject=party":                                           "",
		"?member=nobody&member=sample-acct-04&member=sample-acct-04":                     "acct-04",
	} {
		if got := f.orgRead(root, query); got.status != 200 || memberIDs(got) != ids || got.body["next"] != "" || got.body["scan_stopped"] != false {
			t.Errorf("%s: %d %s; want %s", query, got.status, memberIDs(got), ids)
		}
	}
	exact := members(f.orgRead(root, "?member=sample-acct-03&member=sample-acct-01&subject=account&on=2026-11-16"))
	if targetIDs(exact["sample-acct-03"]) != "sample-unit-north-data" || targetIDs(exact["sample-acct-01"]) != "sample-unit-north-ops" || len(exact["sample-acct-03"]["history"].([]any)) != 1 {
		t.Errorf("an exact account lookup carries the requested kind %+v", exact)
	}
	if got := f.orgRead(root, "?member=nobody&member=sample-acct-04&member=sample-acct-04"); got.body["filters"].(map[string]any)["member_count"] != 2.0 {
		t.Errorf("member count %+v", got.body["filters"])
	}
	big := []string{}
	for i := 0; i < orgMemberCap; i++ {
		big = append(big, "member="+fmt.Sprintf("%03d", i)+strings.Repeat("m", 125))
	}
	query := "?" + strings.Join(big, "&")
	if len(query) > 8000 {
		t.Errorf("fifty maximal IDs take %d bytes", len(query))
	}
	if got := f.orgRead(root, query); got.status != 200 || memberIDs(got) != "" || got.body["filters"].(map[string]any)["member_count"] != 50.0 {
		t.Errorf("fifty maximal IDs %d %s", got.status, got.raw)
	}
	for _, q := range []string{query + "&member=x", "?member=sample-acct-01&after=sample-acct-00", "?member=sample-acct-01&limit=5", "?member=sample-acct-01&limit=", "?member=", "?member=bad%20id", "?member=" + strings.Repeat("m", 129),
		"?subject=accounts", "?unattributed=yes", "?unattributed=false", "?entity=bad%20id", "?target=bad%20id", "?q=" + strings.Repeat("q", 65), "?q=%ff",
		"?entity=sample-unit-north-ops", "?entity=sample-acct-01", "?entity=nowhere", "?entity=other-le"} {
		f.code(t, q[:min(len(q), 40)], f.orgRead(root, q), 422, string(commands.InvalidRequest))
	}
	if got := f.orgRead(root, "?subject=&q="); got.status != 200 {
		t.Errorf("empty values are absent %d", got.status)
	}
	if got := f.orgRead(root, "?q="+strings.Repeat("q", 64)); got.status != 200 {
		t.Errorf("a 64-byte search %d", got.status)
	}
	for _, q := range []string{"?provider=aws", "?provider=", "?provider=aws&kind=other"} {
		got := f.orgRead(root, q)
		f.code(t, q, got, 501, string(commands.UnsupportedCapability))
		if got.body["retry"] != false || got.body["next_action"] != "none" {
			t.Errorf("%s: %+v", q, got.body)
		}
	}
	f.code(t, "provider without a session", f.orgRead(actor{}, "?provider=aws"), 401, string(commands.Unauthenticated))
}

func TestOrganizationEntityOfAnotherWorkspaceIsRefused(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if _, err := f.s.orgs.PutWorkspace(organizations.Workspace{ID: "other-workspace", Profile: organizations.ProfileGroup, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.orgs.PutParty(organizations.Party{ID: "other-le", WorkspaceID: "other-workspace", Kind: organizations.LegalEntity, State: organizations.Active}, 0); err != nil {
		t.Fatal(err)
	}
	f.code(t, "another workspace", f.orgRead(f.as(AdminLogin), "?entity=other-le"), 422, string(commands.InvalidRequest))
	other, err := f.s.readOrganizations("other-workspace", orgQuery{on: sampleDay(2026, 11, 16), kind: relationships.Hierarchy, limit: 50, filter: orgFilter{subject: SubjectAccount, entity: "other-le"}})
	if err != nil || len(other.Members) != 0 || len(other.Parties) != 1 {
		t.Errorf("the other workspace sees only its own %+v %v", other, err)
	}
	if _, err := f.s.readOrganizations("other-workspace", orgQuery{on: sampleDay(2026, 11, 16), kind: relationships.Hierarchy, limit: 50, filter: orgFilter{entity: "sample-le-north"}}); !errors.Is(err, errInvalidQuery) {
		t.Errorf("a sample entity from the other workspace %v", err)
	}
}

func TestSuspendedEntityAndAccountSet(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	party, revision, err := f.s.orgs.Party(SampleWorkspace, "sample-le-south")
	if err != nil {
		t.Fatal(err)
	}
	party.State = organizations.Suspended
	if _, err := f.s.orgs.PutParty(party, revision); err != nil {
		t.Fatal(err)
	}
	if got := f.orgRead(root, "?entity=sample-le-south&subject=account"); got.status != 200 || memberIDs(got) != "acct-04,acct-05,acct-06" {
		t.Errorf("a suspended entity %d %s", got.status, memberIDs(got))
	}
	f.activate("payer-only", openLine(relationships.Payer, "sample-acct-07", "sample-le-north"))
	if got := f.orgRead(root, "?subject=account&kind=hierarchy&unattributed=true"); memberIDs(got) != "acct-07" {
		t.Errorf("a payer-only account is unattributed in the hierarchy %s", memberIDs(got))
	}
	rows := members(f.orgRead(root, "?subject=account"))
	if _, has := rows["sample-acct-07"]; !has || len(rows) != 7 || targetIDs(rows["sample-acct-07"]) != "" {
		t.Errorf("the account set %v", rows)
	}
	if got := f.orgRead(root, "?kind=hierarchy"); strings.Contains(memberIDs(got), "acct-07") {
		t.Errorf("subject any keeps the timelines of the kind %s", memberIDs(got))
	}
}

func TestScanBudget(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.cfg.Org.FilterScan = 9
	if got := f.orgRead(root, "?unattributed=true"); got.status != 200 || memberIDs(got) != "" || got.body["scan_stopped"] != false || got.body["next"] != "" || got.body["scan_budget"] != 9.0 {
		t.Errorf("a budget used up exactly at the end %+v", got.body)
	}
	f.s.cfg.Org.FilterScan = 8
	stopped := f.orgRead(root, "?unattributed=true")
	if memberIDs(stopped) != "" || stopped.body["scan_stopped"] != true || stopped.body["next"] != "sample-unit-north-ops" {
		t.Errorf("a budget used up before any match %+v", stopped.body)
	}
	if rest := f.orgRead(root, "?unattributed=true&after=sample-unit-north-ops"); rest.body["scan_stopped"] != false || rest.body["next"] != "" {
		t.Errorf("the resumed scan %+v", rest.body)
	}
	f.s.cfg.Org.FilterScan = 2
	if got := f.orgRead(root, "?limit=2"); memberIDs(got) != "acct-01,acct-02" || got.body["scan_stopped"] != true || got.body["next"] != "sample-acct-02" {
		t.Errorf("a full page with the look-ahead out of budget %+v", got.body)
	}
	if got := f.orgRead(root, "?limit=1"); memberIDs(got) != "acct-01" || got.body["scan_stopped"] != false || got.body["next"] != "sample-acct-01" {
		t.Errorf("a page with its look-ahead inside the budget %+v", got.body)
	}
	for _, budget := range []int{1, 2, 3, 5, 9, 1000} {
		for _, query := range []string{"?q=0&limit=1", "?entity=sample-le-south&limit=2", "?subject=account&kind=view_group&unattributed=true&limit=1", "?kind=view_group&target=sample-view-south&limit=1", "?limit=3"} {
			f.s.cfg.Org.FilterScan = 1000
			whole := memberIDs(f.orgRead(root, strings.Replace(query, "limit=", "limit=9", 1)))
			f.s.cfg.Org.FilterScan = budget
			seen, cursor := []string{}, ""
			for i := 0; i < 20; i++ {
				got := f.orgRead(root, query+cursor)
				if ids := memberIDs(got); ids != "" {
					seen = append(seen, ids)
				}
				if got.body["next"] == "" {
					break
				}
				cursor = "&after=" + url.QueryEscape(got.body["next"].(string))
			}
			if strings.Join(seen, ",") != whole {
				t.Errorf("budget %d %s: %v; want %s", budget, query, seen, whole)
			}
		}
	}
}

func TestPinnedPagesAcrossAnActivation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	first := f.orgRead(root, "?limit=5&revision=1")
	f.activate("moved", openLine(relationships.Hierarchy, "sample-acct-07", "sample-unit-north-ops"))
	second := f.orgRead(root, "?limit=5&revision=1&after="+first.body["next"].(string))
	if memberIDs(first)+","+memberIDs(second) != "acct-01,acct-02,acct-03,acct-04,acct-05,acct-06,unit-north-data,unit-north-ops,unit-south-web" ||
		fmt.Sprint(second.body["relationship_revision"]) != "1" || fmt.Sprint(second.body["active_revision"]) != "2" || second.body["next"] != "" {
		t.Errorf("pinned pages %s %s %+v", memberIDs(first), memberIDs(second), second.body)
	}
	if now := f.orgRead(root, "?target=sample-unit-north-ops&subject=account"); memberIDs(now) != "acct-01,acct-07" || fmt.Sprint(now.body["relationship_revision"]) != "2" {
		t.Errorf("the new revision %s", memberIDs(now))
	}
	if old := f.orgRead(root, "?target=sample-unit-north-ops&revision=1"); memberIDs(old) != "acct-01" {
		t.Errorf("the pinned revision %s", memberIDs(old))
	}
}

func TestOrganizationScanFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	day := sampleDay(2026, 11, 16)
	base := orgQuery{on: day, kind: relationships.Hierarchy, limit: 5}
	scan := func(ws string, change func(*orgQuery)) error {
		q := base
		change(&q)
		_, err := (&orgScan{s: f.s, ws: ws, number: 1, q: q, parties: map[string]bool{}}).page(10)
		return err
	}
	for name, err := range map[string]error{
		"listing":              scan("bad ws", func(*orgQuery) {}),
		"account set":          scan("bad ws", func(q *orgQuery) { q.filter.subject = SubjectAccount }),
		"target":               scan("bad ws", func(q *orgQuery) { q.filter.target = "sample-view-south" }),
		"exact lookup":         scan("bad ws", func(q *orgQuery) { q.filter.members = []string{"sample-acct-01"} }),
		"exact account lookup": scan("bad ws", func(q *orgQuery) { q.filter.members, q.filter.subject = []string{"sample-acct-01"}, SubjectAccount }),
		"account timeline":     scan(SampleWorkspace, func(q *orgQuery) { q.kind, q.filter.subject = "other", SubjectAccount }),
		"chain on a listing":   scan(SampleWorkspace, func(q *orgQuery) { q.on, q.filter.entity = relationships.Day{}, "sample-le-north" }),
		"chain on an exact read": scan(SampleWorkspace, func(q *orgQuery) {
			q.on, q.filter.entity, q.filter.members = relationships.Day{}, "sample-le-north", []string{"sample-acct-01"}
		}),
	} {
		if !errors.Is(err, relationships.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, subject := range []string{SubjectAny, SubjectAccount} {
		if line, has, err := (&orgScan{s: f.s, ws: "bad ws", number: 1, q: orgQuery{kind: relationships.Hierarchy, filter: orgFilter{subject: subject}}}).lookup("sample-acct-01"); has || err == nil || line.Member != "sample-acct-01" {
			t.Errorf("a failed %s lookup %+v %v %v", subject, line, has, err)
		}
	}
	for _, filter := range []orgFilter{{target: "g"}, {subject: SubjectAccount}} {
		if lines, ended, err := (&orgScan{s: f.s, ws: "bad ws", number: 1, q: orgQuery{on: day, kind: relationships.ViewGroup, filter: filter}}).chunk(""); lines != nil || ended || err == nil {
			t.Errorf("a failed chunk %+v: %v %v %v", filter, lines, ended, err)
		}
	}
	if _, err := f.s.readOrganizations(SampleWorkspace, orgQuery{kind: relationships.Hierarchy, limit: 5, filter: orgFilter{entity: "sample-le-north"}}); err == nil || errors.Is(err, errInvalidQuery) {
		t.Errorf("a scan failure %v", err)
	}
	f.h = f.s.Handler()
	f.s.fault = func(at string) bool { return at == "organizations" }
	f.code(t, "fault with filters", f.orgRead(f.as(AdminLogin), "?subject=account&q=a"), 503, string(commands.DependencyUnavailable))
}

func TestScansAcrossSeveralChunks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	extra := []relationships.Timeline{}
	for i := 0; i < 130; i++ {
		extra = append(extra, openLine(relationships.Hierarchy, fmt.Sprintf("sample-acct-x%03d", i), "sample-unit-north-ops"),
			openLine(relationships.ViewGroup, fmt.Sprintf("sample-acct-x%03d", i), fmt.Sprintf("sample-view-x%03d", i)))
	}
	f.activate("many", extra...)
	groups := f.targetsRead(root, "?kind=view_group&q=view-x12")
	if targetCounts(groups) != "view-x120=1,view-x121=1,view-x122=1,view-x123=1,view-x124=1,view-x125=1,view-x126=1,view-x127=1,view-x128=1,view-x129=1" || groups.body["next"] != "" || groups.body["scan_stopped"] != false {
		t.Errorf("targets across chunks %s %+v", targetCounts(groups), groups.body["next"])
	}
	if page := f.targetsRead(root, "?kind=view_group&limit=99&after=sample-view-x020"); len(page.body["targets"].([]any)) != 99 || page.body["next"] != "sample-view-x119" {
		t.Errorf("a full targets page across chunks %d %v", len(page.body["targets"].([]any)), page.body["next"])
	}
	want := "acct-x120,acct-x121,acct-x122,acct-x123,acct-x124,acct-x125,acct-x126,acct-x127,acct-x128,acct-x129"
	for _, query := range []string{"?q=x12", "?q=x12&subject=account", "?q=x12&target=sample-unit-north-ops&subject=account"} {
		if got := f.orgRead(root, query); memberIDs(got) != want || got.body["next"] != "" || got.body["scan_stopped"] != false {
			t.Errorf("%s: %s %+v", query, memberIDs(got), got.body["next"])
		}
	}
	if got := f.orgRead(root, "?subject=account&limit=99&after=sample-acct-x020"); len(got.body["members"].([]any)) != 99 || got.body["next"] != "sample-acct-x119" {
		t.Errorf("a full page across chunks %d %v", len(got.body["members"].([]any)), got.body["next"])
	}
	if lines, ended, err := (&orgScan{s: f.s, ws: SampleWorkspace, number: 2, q: orgQuery{kind: "other", filter: orgFilter{subject: SubjectAccount}}}).chunk(""); lines != nil || ended || err == nil {
		t.Errorf("a failed timeline in a chunk %v %v %v", lines, ended, err)
	}
}
