package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func fieldsOf(query string) map[string]string {
	values, _ := url.ParseQuery(query)
	out := map[string]string{}
	for key := range values {
		out[key] = values.Get(key)
	}
	return out
}

func (f world) match(a actor, query string) reply {
	f.t.Helper()
	return f.run(a, "bulk_revocation.add_matching", bulkTarget, fieldsOf(query))
}

func (f world) discard(a actor, id string) {
	f.t.Helper()
	if got := f.run(a, "bulk_revocation.discard", id, nil); got.status != 200 {
		f.t.Fatalf("discard %s", got.raw)
	}
}

func (f world) draftUsers(owner, draft string) []string {
	f.t.Helper()
	users, err := f.s.bulk.DraftUsers(SampleWorkspace, owner, draft)
	if err != nil {
		f.t.Fatalf("draft users %v", err)
	}
	return users
}

func (f world) reasons(a actor, id string) map[string]string {
	f.t.Helper()
	out := map[string]string{}
	for _, kind := range []string{"excluded", ""} {
		for after := "0"; after != ""; {
			got := f.bulkRead(a, "/"+id+"/targets?limit=100&kind="+kind+"&after="+after)
			if after == "0" {
				got = f.bulkRead(a, "/"+id+"/targets?limit=100&kind="+kind)
			}
			if got.status != 200 {
				f.t.Fatalf("reasons %s", got.raw)
			}
			key := map[string]string{"excluded": "exclusions", "": "targets"}[kind]
			for _, row := range got.body[key].([]any) {
				x := row.(map[string]any)
				reason, _ := x["reason"].(string)
				if kind == "" {
					reason = "target"
				}
				out[x["user_id"].(string)] = reason
			}
			after, _ = got.body["next"].(string)
		}
	}
	return out
}

func swept(f world, p *population, root actor) (world, *population, actor) {
	f.s.domain.Lock()
	f.s.sweepRoles()
	f.s.domain.Unlock()
	return f, p, root
}

func jsonOf(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func code(got reply) string {
	value, _ := got.body["code"].(string)
	return value
}

func TestAddMatchingFixesTheListsSet(t *testing.T) {
	f, _, root := swept(filteredWorld(t))
	for _, query := range []string{"", "membership=active", "role=member", "q=ANN", "signin=password_set", "status=departed", "membership=none", "period=ends_soon",
		"holder=true", "invitation=pending", "q=u-&status=active&role=member"} {
		got := f.match(root, query)
		if got.status != 200 || got.body["business_state"] != "draft" {
			t.Fatalf("%q: %s", query, got.raw)
		}
		draft := field(got, "draft")
		want, _ := f.walk(root, query)
		slices.Sort(want)
		if users := f.draftUsers(SampleAdmin, draft); !slices.Equal(users, want) || field(got, "matched") != fmt.Sprint(len(want)) || field(got, "count") != fmt.Sprint(len(want)) {
			t.Errorf("%q: fixed %v, list %v, fields %s", query, users, want, got.raw)
		}
		for key, value := range fieldsOf(query) {
			if field(got, "filter."+key) != value {
				t.Errorf("%q: filter.%s %s", query, key, got.raw)
			}
		}
		if field(got, "fixed_at") != stamp(f.s.clock()) || field(got, "fixed_revision") == "" {
			t.Errorf("%q: fixed facts %s", query, got.raw)
		}
		f.discard(root, draft)
	}
}

func TestAddMatchingRefusals(t *testing.T) {
	f, _, root := swept(filteredWorld(t))
	if got := f.match(root, "q=nobody-at-all"); code(got) != "prerequisite_failed" {
		t.Errorf("nobody matches %s", got.raw)
	}
	if got := f.match(root, "method=password"); code(got) != "unsupported_capability" {
		t.Errorf("method %s", got.raw)
	}
	f.fieldRefused(t, "workspace", f.run(root, "bulk_revocation.add_matching", bulkTarget, map[string]string{"workspace": SampleWorkspace}), "workspace")
	f.fieldRefused(t, "empty search", f.run(root, "bulk_revocation.add_matching", bulkTarget, map[string]string{"q": ""}), "q")
	f.fieldRefused(t, "blank search", f.run(root, "bulk_revocation.add_matching", bulkTarget, map[string]string{"q": "   "}), "q")
	f.fieldRefused(t, "long search", f.run(root, "bulk_revocation.add_matching", bulkTarget, map[string]string{"q": strings.Repeat("a", 65)}), "q")
	f.fieldRefused(t, "unknown status", f.run(root, "bulk_revocation.add_matching", bulkTarget, map[string]string{"status": "gone"}), "status")
	f.fieldRefused(t, "after", f.run(root, "bulk_revocation.add_matching", bulkTarget, map[string]string{"after": "a:u-ann"}), "after")
	if got := f.match(f.as(Admin3Login), "role=member"); code(got) != "permission_denied" {
		t.Errorf("without the bulk right %s", got.raw)
	}
	if got := f.call("bulk_revocation.add_matching", `{"request_id":"r-x","target_id":"bulk-revocations","idempotency_key":"k-x","expected_revision":"r1","rationale":"x","body":{"role":"member"}}`, nil, ""); got.status != 401 {
		t.Errorf("no session %s", got.raw)
	}
	staged := f.stage(root, "", "u-ann")
	if got := f.match(root, "role=member"); code(got) != "prerequisite_failed" {
		t.Errorf("with an open explicit draft %s", got.raw)
	}
	f.discard(root, field(staged, "draft"))
	fixed := f.match(root, "role=member")
	draft := field(fixed, "draft")
	if got := f.stage(root, draft, "u-kim"); code(got) != "invalid_stage" {
		t.Errorf("adding to a filter draft %s", got.raw)
	}
	if got := f.match(root, "role=administrator"); code(got) != "prerequisite_failed" {
		t.Errorf("a second filter draft %s", got.raw)
	}
	other := f.as(Admin2Login)
	unknown := f.prepare(other, "bulk-draft-99", "suspend_membership")
	for name, got := range map[string]reply{"prepare": f.prepare(other, draft, "suspend_membership"), "discard": f.run(other, "bulk_revocation.discard", draft, nil)} {
		if got.status == 200 || code(got) != code(unknown) {
			t.Errorf("another holder's %s of a filter draft %s (unknown draft %s)", name, got.raw, unknown.raw)
		}
	}
	if got := f.bulkRead(other, "/"+draft+"/targets?kind=excluded"); got.status == 200 {
		t.Errorf("another holder reads the draft's exclusions %s", got.raw)
	}
	if _, err := f.s.bulk.DraftView(SampleWorkspace, SampleAdmin, draft); err != nil {
		t.Errorf("the draft changed %v", err)
	}
}

func TestAddMatchingLimits(t *testing.T) {
	f, _, root := swept(filteredWorld(t))
	members, _ := f.walk(root, "role=member")
	n := len(members)
	limited := func(limit, pending int) (world, actor) {
		g, _, a := swept(filteredWorldWith(t, func(c *Config) {
			c.Access = DefaultAccessSettings()
			c.Access.BulkMaxTargets, c.Access.BulkMaxMatching, c.Access.BulkPending = 1, limit, pending
		}))
		return g, a
	}
	g, a := limited(n, 20)
	if got := g.match(a, "role=member"); got.status != 200 || field(got, "matched") != fmt.Sprint(n) {
		t.Errorf("exactly the limit %s", got.raw)
	}
	g, a = limited(n-1, 20)
	if got := g.match(a, "role=member"); code(got) != "prerequisite_failed" {
		t.Errorf("one over the limit %s", got.raw)
	}
	if got := g.match(a, "role=administrator"); got.status != 200 {
		t.Errorf("under the limit %s", got.raw)
	}
	g, a = limited(n, 1)
	g.stage(g.as(Admin2Login), "", "u-ann")
	if got := g.match(a, "role=member"); code(got) != "prerequisite_failed" {
		t.Errorf("at the pending limit %s", got.raw)
	}
}

func TestFilterListPrepareRechecksTheFilter(t *testing.T) {
	f, p, root := swept(filteredWorld(t))
	fixed := f.match(root, "membership=active&role=member")
	draft := field(fixed, "draft")
	before := f.draftUsers(SampleAdmin, draft)
	for _, id := range []string{"u-ann", "u-cat", "u-hal"} {
		if !slices.Contains(before, id) {
			t.Fatalf("%s is not in the fixed list %v", id, before)
		}
	}
	now := f.s.clock()
	p.suspendMembership("u-cat")
	p.until("u-ann", now.Add(time.Minute))
	p.until("u-hal", now.Add(300*24*time.Hour))
	p.member("u-new", "new@example.invalid", identity.Member)
	f.advance(2 * time.Minute)
	p.refresh()
	swept(f, p, root)
	got := f.prepare(root, draft, "suspend_membership")
	if got.status != 200 {
		t.Fatalf("prepare %s", got.raw)
	}
	id := field(got, "bulk")
	reasons := f.reasons(root, id)
	if reasons["u-cat"] != bulkrevoke.ExcludedNoLongerMatches || reasons["u-ann"] != bulkrevoke.ExcludedNoLongerMatches || reasons["u-hal"] != "target" {
		t.Errorf("reasons %v", reasons)
	}
	if _, listed := reasons["u-new"]; listed || len(reasons) != len(before) {
		t.Errorf("the list widened %v", reasons)
	}
	detail := f.bulkRead(root, "/"+id)
	job := detail.body["job"].(map[string]any)
	counts := job["counts"].(map[string]any)
	if detail.body["exclusions"].(map[string]any)["no_longer_matches"] != 2.0 || whole(counts["targets"])+whole(counts["excluded"]) != len(before) ||
		job["selection"] != "filter" || whole(job["matched"]) != len(before) || job["fixed_at"] != field(fixed, "fixed_at") || job["fixed_revision"] != field(fixed, "fixed_revision") {
		t.Errorf("detail %s", detail.raw)
	}
	filter := job["filter"].(map[string]any)
	if filter["membership"] != "active" || filter["role"] != "member" || filter["q"] != "" || job["search_masked"] != false {
		t.Errorf("filter %v", filter)
	}
}

func TestFilterListClassificationOrder(t *testing.T) {
	f, p, root := swept(filteredWorld(t))
	fixed := f.match(root, "status=active&membership=none")
	draft := field(fixed, "draft")
	if users := f.draftUsers(SampleAdmin, draft); !slices.Contains(users, "u-gus") || !slices.Contains(users, "u-kim") {
		t.Fatalf("fixed %v", users)
	}
	p.suspendUser("u-kim")
	if got := f.prepare(root, draft, "suspend_user"); code(got) != "invalid_stage" {
		t.Fatalf("nothing left to revoke %s", got.raw)
	}
	if _, err := f.s.bulk.DraftView(SampleWorkspace, SampleAdmin, draft); err != nil {
		t.Errorf("the draft is gone %v", err)
	}
	f.discard(root, draft)
	fixed = f.match(root, "q=u-jon")
	if got := f.prepare(root, field(fixed, "draft"), "suspend_membership"); code(got) != "invalid_stage" {
		t.Errorf("a departed person only %s", got.raw)
	}
	f.discard(root, field(fixed, "draft"))
	fixed = f.match(root, "status=active&q=example")
	draft = field(fixed, "draft")
	p.depart("u-cat")
	p.suspendUser("u-ann")
	p.suspendUser("u-fay")
	got := f.prepare(root, draft, "suspend_user")
	if got.status != 200 {
		t.Fatalf("prepare %s", got.raw)
	}
	reasons := f.reasons(root, field(got, "bulk"))
	if reasons[SampleAdmin] != bulkrevoke.ExcludedSelf || reasons["u-cat"] != bulkrevoke.ExcludedNotFound || reasons["u-ann"] != bulkrevoke.ExcludedNoLongerMatches ||
		reasons["u-fay"] != bulkrevoke.ExcludedNotFound || reasons["u-gus"] != bulkrevoke.ExcludedNotFound || reasons["u-hal"] != "target" {
		t.Errorf("reasons %v", reasons)
	}
}

func TestFilterListThatStrandsTheAnchorsIsRefused(t *testing.T) {
	f, p, root := swept(filteredWorld(t))
	p.until(SampleAdmin, f.s.clock().Add(30*24*time.Hour))
	f.advance(time.Second)
	fixed := f.match(root, "role=administrator")
	draft := field(fixed, "draft")
	if got := f.prepare(root, draft, "suspend_membership"); code(got) != "prerequisite_failed" {
		t.Fatalf("stranding the anchors %s", got.raw)
	}
	if v, err := f.s.bulk.DraftView(SampleWorkspace, SampleAdmin, draft); err != nil || v.State != bulkrevoke.Draft {
		t.Errorf("the draft froze %+v %v", v, err)
	}
}

func TestFilterListSearchIsShownToThePreparerOnly(t *testing.T) {
	var logs bytes.Buffer
	f, _, root := swept(filteredWorldWith(t, func(c *Config) { c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) }))
	fixed := f.match(root, "q=U-AN&role=member")
	got := f.prepare(root, field(fixed, "draft"), "suspend_membership")
	id := field(got, "bulk")
	executed := f.execute(root, id, field(got, "digest"))
	if executed.status != 202 {
		t.Fatalf("execute %s", executed.raw)
	}
	if !strings.Contains(logs.String(), `"msg":"bulk_revocation_matched"`) || !strings.Contains(logs.String(), `"filters":"q,role"`) || strings.Contains(logs.String(), "U-AN") ||
		strings.Contains(logs.String(), "ann@example.invalid") {
		t.Errorf("logs %s", logs.String())
	}
	if job := f.job(root, id); job["filter"].(map[string]any)["q"] != "U-AN" || job["search_masked"] != false {
		t.Errorf("the preparer %+v", job)
	}
	for _, login := range []string{Admin2Login, Admin3Login} {
		a := f.as(login)
		list := f.bulkRead(a, "")
		detail := f.bulkRead(a, "/"+id)
		items := f.bulkRead(a, "/"+id+"/items")
		if list.status != 200 || detail.status != 200 || items.status != 200 {
			t.Fatalf("%s reads %d %d %d", login, list.status, detail.status, items.status)
		}
		for name, raw := range map[string]string{"list": list.raw, "detail": detail.raw, "items": items.raw} {
			if strings.Contains(raw, "U-AN") {
				t.Errorf("%s %s shows the search text", login, name)
			}
		}
		job := detail.body["job"].(map[string]any)
		if job["search_masked"] != true || job["filter"].(map[string]any)["role"] != "member" || job["filter"].(map[string]any)["q"] != "" {
			t.Errorf("%s detail %+v", login, job)
		}
		row := list.body["jobs"].([]any)[0].(map[string]any)
		if row["id"] != id || row["search_masked"] != true {
			t.Errorf("%s row %+v", login, row)
		}
	}
}

func TestRosterRowsEqualPeopleRows(t *testing.T) {
	f, p, _ := filteredWorld(t)
	p.member("u-old", "old@example.invalid", identity.Administrator)
	p.until("u-old", f.s.clock().Add(time.Minute))
	f.advance(time.Hour)
	p.refresh()
	f.s.caller = p.admin
	r, err := f.s.everyone()
	if err != nil {
		t.Fatal(err)
	}
	now := f.s.clock()
	checked := 0
	for id, u := range r.users {
		if u.Kind != identity.Human {
			continue
		}
		found, err := f.s.identity.Workspace().People(p.admin, SampleWorkspace, []string{id})
		p.must(err)
		var member *identity.MemberInfo
		if m, ok := r.members[id]; ok {
			member = &m
		}
		if (member == nil) != (found[0].Member == nil) || member != nil && *member != *found[0].Member {
			t.Fatalf("%s: roster member %+v, people member %+v", id, member, found[0].Member)
		}
		fromRoster, fromPeople := f.s.row(u, member, now), f.s.row(found[0].User, found[0].Member, now)
		if a, b := jsonOf(fromRoster), jsonOf(fromPeople); a != b {
			t.Fatalf("%s rows differ", id)
		}
		for key, values := range userFilterValues {
			for _, value := range values {
				m := newMatcher(userFilter{}).withField(key, value)
				if m.matches(fromRoster, member, now) != m.matches(fromPeople, found[0].Member, now) {
					t.Errorf("%s %s=%s differs", id, key, value)
				}
			}
		}
		draft := bulkrevoke.View{Selection: bulkrevoke.Filter, Filter: []bulkrevoke.FilterTerm{{Key: "membership", Value: membershipNone}}}
		if f.s.stillMatches(draft, r, now)(id) != (found[0].Member == nil) {
			t.Errorf("%s: membership none", id)
		}
		checked++
	}
	for _, id := range []string{"u-fay", "u-gus", "u-kim", "u-jon", "u-eve", "u-dan", "u-old", "u-ivy"} {
		if _, ok := r.users[id]; !ok {
			t.Errorf("%s is not in the roster", id)
		}
	}
	if checked < 15 || f.s.stillMatches(bulkrevoke.View{Selection: bulkrevoke.Explicit}, r, now) != nil {
		t.Errorf("checked %d", checked)
	}
}

func TestMatchingSettings(t *testing.T) {
	for name, change := range map[string]func(*AccessSettings){
		"bulk_revocation_max_matching": func(s *AccessSettings) { s.BulkMaxTargets, s.BulkMaxMatching = 600, 599 },
		"bulk_revocation_retained_people_max": func(s *AccessSettings) {
			s.BulkKept, s.BulkPending, s.BulkMaxTargets, s.BulkMaxMatching = 1000, 200, 5000, 5000
		},
		"user_count_scan": func(s *AccessSettings) { s.UserCountScan = 999 },
	} {
		s := DefaultAccessSettings()
		change(&s)
		_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: s})
		if !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	s := DefaultAccessSettings()
	s.BulkKept, s.BulkPending, s.BulkMaxTargets, s.BulkMaxMatching = 1000, 200, 5000, 5000
	_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: s})
	if err == nil || err.Error() != "configuration is invalid: bulk_revocation_retained_people_max 1100000 is below (2 × bulk_revocation_jobs_kept 1000 + bulk_revocation_pending 200) × bulk_revocation_max_matching 5000 = 11000000" {
		t.Errorf("the refusal %v", err)
	}
	if d := DefaultAccessSettings(); d.RetainedPeople() != 1100000 || d.BulkRetainedMax != 1100000 || d.BulkMaxMatching != 5000 || d.UserCountScan != 20000 {
		t.Errorf("defaults %+v", d)
	}
	s = DefaultAccessSettings()
	s.BulkKept, s.BulkPending, s.BulkMaxMatching, s.BulkRetainedMax = 100, 20, 5000, 1100000
	if _, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: s}); err != nil {
		t.Errorf("exactly the bound %v", err)
	}
	if SettingNote("bulk_revocation_retained_people_max") == "" || SettingNote("user_count_scan") != "" {
		t.Error("notes")
	}
	f, _, root := swept(filteredWorld(t))
	body := `{"request_id":"r-f","target_id":"bulk-revocations","idempotency_key":"k-f","expected_revision":"` + f.revision(root) + `","rationale":"x","body":{"role":"member"}}`
	f.s.fault = func(at string) bool { return at == "people" }
	if got := f.call("bulk_revocation.add_matching", body, root.cookie, root.csrf); code(got) != "dependency_unavailable" {
		t.Errorf("people fault %s", got.raw)
	}
}

func TestDetailNamesNoLongerMatchingOnlyForFilterLists(t *testing.T) {
	f, _, root := swept(filteredWorld(t))
	fixed := f.match(root, "role=administrator&status=active")
	filter := f.prepare(root, field(fixed, "draft"), "suspend_membership")
	explicitID, _ := f.prepared(t, f.as(Admin2Login), "suspend_membership", "u-ann")
	got := f.bulkRead(root, "/"+field(filter, "bulk"))
	if value, ok := got.body["exclusions"].(map[string]any)["no_longer_matches"]; !ok || value != 0.0 {
		t.Errorf("a filter list without drift %s", got.raw)
	}
	got = f.bulkRead(root, "/"+explicitID)
	if _, ok := got.body["exclusions"].(map[string]any)["no_longer_matches"]; ok {
		t.Errorf("an explicit list %s", got.raw)
	}
}

func TestPeopleFailureInTheLoginSegmentFailsTheWholeWalk(t *testing.T) {
	for _, once := range []bool{false, true} {
		f := setup(t, SeedSample).ready()
		root := f.as(AdminLogin)
		if len(f.s.dir.departed) != 0 || len(f.s.dir.order) == 0 {
			t.Fatalf("the seed has departed people %v", f.s.dir.departed)
		}
		fault := func() {
			fired := false
			f.s.fault = func(at string) bool {
				if at != "people" || once && fired {
					return false
				}
				fired = true
				return true
			}
		}
		body := `{"request_id":"r-p","target_id":"bulk-revocations","idempotency_key":"k-p","expected_revision":"` + f.revision(root) + `","rationale":"x","body":{}}`
		fault()
		if got := f.count(root, "?role=member"); code(got) != "dependency_unavailable" {
			t.Errorf("once %v: count %s", once, got.raw)
		}
		fault()
		if got := f.call("bulk_revocation.add_matching", body, root.cookie, root.csrf); code(got) != "dependency_unavailable" {
			t.Errorf("once %v: add_matching %s", once, got.raw)
		}
		f.s.fault = nil
		if pending, open := f.s.bulk.Limits(SampleWorkspace, SampleAdmin); pending != 0 || open {
			t.Errorf("once %v: a draft was created (%d pending, open %v)", once, pending, open)
		}
	}
}
