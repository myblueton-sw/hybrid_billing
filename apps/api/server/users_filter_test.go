package server

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

type population struct {
	f      world
	admin  identity.Verifier
	record passwords.Record
}

func newPopulation(f world) *population {
	f.t.Helper()
	admin := verifierOf(random(tokenBytes))
	if err := f.s.identity.Boundary().OpenSession(AdminTriple(), admin, sessionIdle, sessionAbsolute); err != nil {
		f.t.Fatal(err)
	}
	record, err := f.s.passwords.Hash(LoginKey("record@example.invalid"), pw)
	if err != nil {
		f.t.Fatal(err)
	}
	return &population{f: f, admin: admin, record: record}
}

func (p *population) must(err error) {
	p.f.t.Helper()
	if err != nil {
		p.f.t.Fatal(err)
	}
}

func (p *population) human(id, login string) {
	p.f.t.Helper()
	p.must(p.f.s.identity.Installation().CreateUser(seedOperator, id, LoginTriple(login), login))
	p.f.s.dir.add(id, login, "contact-"+login)
}

func (p *population) join(id, login string, role identity.Role) {
	p.f.t.Helper()
	now := p.f.s.clock()
	invitation := "inv-" + id
	p.must(p.f.s.identity.Workspace().Invite(p.admin, SampleWorkspace, invitation, role, verifierOf([]byte(invitation)), identity.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}, id))
	v := verifierOf(random(tokenBytes))
	p.must(p.f.s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute))
	p.must(p.f.s.identity.Workspace().Accept(SampleWorkspace, invitation, verifierOf([]byte(invitation)), v))
	p.must(p.f.s.identity.Boundary().EndSession(v))
}

func (p *population) member(id, login string, role identity.Role) {
	p.f.t.Helper()
	p.human(id, login)
	p.join(id, login, role)
}

func (p *population) person(id string) identity.Person {
	p.f.t.Helper()
	found, err := p.f.s.identity.Workspace().People(p.admin, SampleWorkspace, []string{id})
	p.must(err)
	return found[0]
}

func (p *population) until(id string, at time.Time) {
	p.f.t.Helper()
	p.must(p.f.s.identity.Workspace().SetRolePeriod(p.admin, SampleWorkspace, id, at, p.person(id).Member.Version, p.f.s.clock()))
}

func (p *population) suspendMembership(id string) {
	p.f.t.Helper()
	p.must(p.f.s.identity.Workspace().SuspendMembership(p.admin, SampleWorkspace, id, p.person(id).Member.Version))
}

func (p *population) removeMembership(id string) {
	p.f.t.Helper()
	p.must(p.f.s.identity.Workspace().RemoveMembership(p.admin, SampleWorkspace, id, p.person(id).Member.Version))
}

func (p *population) suspendUser(id string) {
	p.f.t.Helper()
	p.must(p.f.s.identity.Installation().SuspendUser(seedOperator, id, p.person(id).User.Epoch))
}

func (p *population) depart(id string) {
	p.f.t.Helper()
	p.must(p.f.s.identity.Installation().DepartUser(seedOperator, id, p.person(id).User.Epoch))
	if login, ok := p.f.s.dir.loginOf(id); ok {
		p.f.s.dir.dropInvite(login)
		p.f.s.dir.dropLink(login)
	}
	p.f.s.dir.dropHolder(id)
	p.f.s.dir.depart(id)
}

func (p *population) password(login string) {
	p.f.t.Helper()
	p.must(p.f.s.passwords.Put(LoginKey(login), p.record))
}

func (p *population) link(login, purpose string) {
	p.f.t.Helper()
	_, err := p.f.s.recordLink(login, purpose, p.f.s.clock())
	p.must(err)
}

func (p *population) invite(id, login string) {
	p.f.t.Helper()
	p.must(p.f.s.inviteUser(p.admin, login, id, identity.Member, p.f.s.clock(), time.Time{}))
}

func (p *population) holder(id string) {
	p.f.s.dir.holders[id] = p.f.s.clock().Add(30 * 24 * time.Hour)
}

func (p *population) refresh() {
	p.f.t.Helper()
	_ = p.f.s.identity.Boundary().EndSession(p.admin)
	p.admin = verifierOf(random(tokenBytes))
	p.must(p.f.s.identity.Boundary().OpenSession(AdminTriple(), p.admin, sessionIdle, sessionAbsolute))
}

func userIDs(got reply) []string {
	out := []string{}
	users, _ := got.body["users"].([]any)
	for _, u := range users {
		out = append(out, u.(map[string]any)["user_id"].(string))
	}
	return out
}

func (f world) walk(a actor, query string, from ...string) ([]string, []string) {
	f.t.Helper()
	ids, cursors := []string{}, []string{}
	after := strings.Join(from, "")
	for {
		q := query
		if after != "" {
			q += "&after=" + url.QueryEscape(after)
		}
		got := f.list(a, "?"+strings.TrimPrefix(q, "&"))
		if got.status != 200 {
			f.t.Fatalf("walk %s: %+v", q, got)
		}
		ids = append(ids, userIDs(got)...)
		after = got.body["next"].(string)
		if after == "" {
			return ids, cursors
		}
		cursors = append(cursors, after)
	}
}

func filteredWorld(t *testing.T) (world, *population, actor) {
	f := setupWith(t, func(c *Config) { c.Passwords.Links = 1000 })
	p := newPopulation(f)
	now := f.s.clock()
	p.member("u-ann", "ann@example.invalid", identity.Member)
	p.password("ann@example.invalid")
	p.member("u-bob", "bob@example.invalid", identity.Administrator)
	p.until("u-bob", now.Add(5*24*time.Hour))
	p.member("u-cat", "cat@example.invalid", identity.Member)
	p.until("u-cat", now.Add(40*24*time.Hour))
	p.member("u-dan", "dan@example.invalid", identity.Member)
	p.until("u-dan", now.Add(time.Hour))
	p.member("u-eve", "eve@example.invalid", identity.Member)
	p.suspendMembership("u-eve")
	p.member("u-fay", "fay@example.invalid", identity.Member)
	p.removeMembership("u-fay")
	p.human("u-gus", "gus@example.invalid")
	p.invite("u-gus", "gus@example.invalid")
	p.link("gus@example.invalid", passwords.PurposeSet)
	p.member("u-hal", "hal@example.invalid", identity.Member)
	p.password("hal@example.invalid")
	p.link("hal@example.invalid", passwords.PurposeReset)
	p.member("u-ivy", "ivy@example.invalid", identity.Member)
	p.suspendUser("u-ivy")
	p.member("u-jon", "jon@example.invalid", identity.Member)
	p.depart("u-jon")
	p.human("u-kim", "kim@example.invalid")
	p.holder("u-kim")
	f.advance(2 * time.Hour)
	p.refresh()
	return f, p, f.as(AdminLogin)
}

func TestUsersFiltersAloneAndCombined(t *testing.T) {
	f, _, root := filteredWorld(t)
	cases := map[string]string{
		"status=suspended":                      "u-ivy",
		"status=departed":                       "u-jon",
		"membership=suspended":                  "u-eve",
		"membership=ended":                      "u-dan",
		"membership=none":                       "u-fay,u-gus,u-kim,sample-member,u-jon",
		"role=administrator":                    "sample-admin-2,sample-admin-3,sample-admin,u-bob",
		"role=member&membership=active":         "u-ann,u-cat,u-hal,u-ivy",
		"period=ends_soon":                      "u-bob",
		"period=bounded":                        "u-cat",
		"period=ended":                          "u-dan",
		"signin=reset_link":                     "u-hal",
		"signin=set_link":                       "sample-admin-2,sample-admin-3,sample-admin,u-gus,sample-member",
		"signin=password_set&role=member":       "u-ann,u-hal",
		"invitation=pending":                    "sample-admin-2,sample-admin-3,sample-admin,u-gus,sample-member",
		"holder=true":                           "sample-admin-2,sample-admin-3,sample-admin,u-kim",
		"holder=true&membership=none":           "u-kim",
		"q=EVE":                                 "u-eve",
		"q=contact-fay":                         "u-fay",
		"q=u-j":                                 "u-jon",
		"q=example&status=departed":             "",
		"q=JON&status=departed":                 "u-jon",
		"q=nobody-here":                         "",
		"status=active&role=member&period=open": "u-ann,u-eve,u-hal",
	}
	for query, want := range cases {
		ids, _ := f.walk(root, query+"&limit=2")
		if strings.Join(ids, ",") != want {
			t.Errorf("%s: %v; want %s", query, ids, want)
		}
	}
	all, _ := f.walk(root, "limit=3")
	if strings.Join(all, ",") != "sample-admin-2,sample-admin-3,sample-admin,u-ann,u-bob,u-cat,u-dan,u-eve,u-fay,u-gus,u-hal,u-ivy,u-kim,sample-member,u-jon" {
		t.Errorf("unfiltered order %v", all)
	}
	got := f.list(root, "?status=departed")
	row := got.body["users"].([]any)[0].(map[string]any)
	if row["login"] != "" || row["email"] != "" || row["membership"] != nil || row["password_set"] != false {
		t.Errorf("departed row %+v", row)
	}
	echo := f.list(root, "?q=+EVE+&role=member&holder=true").body["filters"].(map[string]any)
	if echo["q"] != "EVE" || echo["role"] != "member" || echo["holder"] != "true" || echo["status"] != "" {
		t.Errorf("echo %+v", echo)
	}
}

func TestUsersMembershipValuesPartitionEveryRow(t *testing.T) {
	f, _, root := filteredWorld(t)
	all, _ := f.walk(root, "limit=100")
	seen := map[string]int{}
	for _, value := range userFilterValues["membership"] {
		ids, _ := f.walk(root, "membership="+value)
		for _, id := range ids {
			seen[id]++
		}
	}
	for _, id := range all {
		if seen[id] != 1 {
			t.Errorf("%s matched %d membership values", id, seen[id])
		}
	}
	if len(seen) != len(all) {
		t.Errorf("%d rows matched, %d listed", len(seen), len(all))
	}
}

func TestUsersSearchFoldsCase(t *testing.T) {
	f := setupWith(t, func(*Config) {})
	p := newPopulation(f)
	p.human("u-ko", "김민수@example.invalid")
	p.human("u-ar", "سارة@example.invalid")
	p.human("u-de", "ÄRGER@example.invalid")
	p.human("u-el", "ΣΟΦΙΑ@example.invalid")
	p.human("u-kelvin", "Kelvin@example.invalid")
	root := f.as(AdminLogin)
	for q, want := range map[string]string{"민수": "u-ko", "سارة": "u-ar", "ärger": "u-de", "Ärger": "u-de", "σοφια": "u-el", "ςοφια": "u-el", "KELVIN": "u-kelvin",
		"kelvin": "u-kelvin", "U-KO": "u-ko"} {
		if ids, _ := f.walk(root, "q="+url.QueryEscape(q)); strings.Join(ids, ",") != want {
			t.Errorf("q=%s: %v; want %s", q, ids, want)
		}
	}
	long := strings.Repeat("가", 21) + "a"
	if len(long) != 64 {
		t.Fatal(len(long))
	}
	if got := f.list(root, "?q="+url.QueryEscape(long)); got.status != 200 {
		t.Errorf("64 bytes %+v", got)
	}
	f.code(t, "65 bytes", f.list(root, "?q="+url.QueryEscape(long+"b")), 422, string(commands.InvalidRequest))
	f.code(t, "spaces only", f.list(root, "?q=+++"), 422, string(commands.InvalidRequest))
	f.code(t, "invalid text", f.list(root, "?q=%ff"), 422, string(commands.InvalidRequest))
}

func TestUsersReadRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, query := range []string{"?workspace=" + SampleWorkspace, "?workspace=elsewhere", "?workspace=bad%20id", "?status=gone", "?colour=red", "?status=active&status=suspended",
		"?status=", "?after=", "?q=", "?limit=0", "?limit=101", "?limit=x", "?after=sample-admin", "?after=x:sample-admin", "?after=a:bad%20id", "?holder=false",
		"?invitation=sent", "?membership=removed", "?role=service", "?period=soon", "?signin=sso", "?method=", "?method=saml&status=gone"} {
		got := f.list(root, query)
		f.code(t, query, got, 422, string(commands.InvalidRequest))
		if got.body["next_action"] != string(commands.NextCorrect) || got.body["fields"] != nil {
			t.Errorf("%s: %+v", query, got.body)
		}
	}
	for _, query := range []string{"?method=saml", "?method=password&status=active"} {
		got := f.list(root, query)
		f.code(t, query, got, 501, string(commands.UnsupportedCapability))
		if got.body["next_action"] != string(commands.NextNone) || got.body["retry"] != false {
			t.Errorf("%s: %+v", query, got.body)
		}
	}
	f.code(t, "no session", f.list(actor{}, ""), 401, string(commands.Unauthenticated))
	f.code(t, "no csrf", f.list(actor{cookie: root.cookie}, ""), 403, CodeCSRF)
	f.setPassword(MemberLogin, pw)
	f.code(t, "member", f.list(f.as(MemberLogin), ""), 403, string(commands.PermissionDenied))
	f.s.bootWorkspace = "elsewhere"
	f.code(t, "another workspace source", f.list(root, ""), 403, string(commands.PermissionDenied))
}

func TestUsersCursorRefusals(t *testing.T) {
	f, p, root := filteredWorld(t)
	v := p.admin
	if err := f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, "a-service", SampleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := f.s.identity.Installation().CreateUser(seedOperator, "u-nodir", LoginTriple("nodir@example.invalid"), "nodir@example.invalid"); err != nil {
		t.Fatal(err)
	}
	for _, after := range []string{"a:ghost", "a:a-service", "a:u-nodir", "d:ghost", "d:u-ann", "d:a-service", "d:zzzz"} {
		got := f.list(root, "?after="+after)
		f.code(t, after, got, 422, string(commands.InvalidRequest))
		if fields, _ := got.body["fields"].([]any); len(fields) != 1 || fields[0] != "after" {
			t.Errorf("%s fields %+v", after, got.body)
		}
	}
	if got := f.list(root, "?after=d:u-jon"); got.status != 200 || len(userIDs(got)) != 0 || got.body["next"] != "" {
		t.Errorf("after the last departed %+v", got.body)
	}
	if got := f.list(root, "?after=a:sample-member"); strings.Join(userIDs(got), ",") != "u-jon" {
		t.Errorf("after the last login %v", userIDs(got))
	}
	first := f.list(root, "?limit=4")
	cursor := first.body["next"].(string)
	if cursor != "a:u-ann" {
		t.Fatalf("cursor %s", cursor)
	}
	p.depart("u-ann")
	got := f.list(root, "?limit=4&after="+cursor)
	f.code(t, "cursor person departed", got, 409, string(commands.RevisionConflict))
	if fields, _ := got.body["fields"].([]any); len(fields) != 1 || fields[0] != "after" || got.body["next_action"] != string(commands.NextInspect) || got.body["retry"] != false {
		t.Errorf("restart answer %+v", got.body)
	}
	bulk := f.do("GET", "/api/v1/identity/bulk-revocations?after=a%20b", "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
	f.code(t, "bulk list cursor", bulk, 422, string(commands.InvalidRequest))
	for limit, status := range map[string]int{"100": 200, "101": 422, "0": 422} {
		got := f.do("GET", "/api/v1/identity/bulk-revocations?limit="+limit, "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
		if got.status != status {
			t.Errorf("bulk list limit %s: %d", limit, got.status)
		}
	}
	f.s.dir.add("ghost", "aaa-ghost@example.invalid", "ghost")
	f.s.dir.add("a-service", "aaa-service@example.invalid", "service")
	if got := f.list(root, "?limit=1"); strings.Join(userIDs(got), ",") != "sample-admin-2" || got.body["next"] != "a:sample-admin-2" {
		t.Errorf("entries without a human %v %v", userIDs(got), got.body["next"])
	}
	f.s.dir.remove("ghost")
	f.s.dir.remove("a-service")
	f.s.fault = func(at string) bool { return at == "users" }
	f.code(t, "lookup fault", f.list(root, "?after=a:u-ann"), 503, string(commands.DependencyUnavailable))
	f.s.fault = func(at string) bool { return at == "people" }
	f.code(t, "people fault", f.list(root, ""), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
}

func TestUsersScanBudget(t *testing.T) {
	f, _, root := filteredWorld(t)
	total := len(f.s.dir.order) + len(f.s.dir.departed)
	f.s.cfg.Access.UserFilterScan = total
	got := f.list(root, "?status=departed")
	if strings.Join(userIDs(got), ",") != "u-jon" || got.body["scan_stopped"] != false || got.body["next"] != "" || fmt.Sprint(got.body["scan_budget"]) != fmt.Sprint(total) {
		t.Errorf("budget used exactly at the end %+v", got.body)
	}
	f.s.cfg.Access.UserFilterScan = total - 1
	got = f.list(root, "?status=departed")
	if len(userIDs(got)) != 0 || got.body["scan_stopped"] != true || got.body["next"] != "a:sample-member" || got.body["next_login"] != MemberLogin {
		t.Errorf("budget ends before the only match %+v", got.body)
	}
	rest := f.list(root, "?status=departed&after=a:sample-member")
	if strings.Join(userIDs(rest), ",") != "u-jon" || rest.body["next"] != "" || rest.body["scan_stopped"] != false || rest.body["next_login"] != "" {
		t.Errorf("resumed %+v", rest.body)
	}
	f.s.cfg.Access.UserFilterScan = 3
	ids, cursors := f.walk(root, "membership=none")
	if strings.Join(ids, ",") != "u-fay,u-gus,u-kim,sample-member,u-jon" {
		t.Errorf("small budget walk %v", ids)
	}
	if len(cursors) < 4 {
		t.Errorf("cursors %v", cursors)
	}
	f.s.cfg.Access.UserFilterScan = len(f.s.dir.order)
	got = f.list(root, "?q=nobody")
	if got.body["scan_stopped"] != true || got.body["next"] != "a:sample-member" {
		t.Errorf("budget ends at the last login %+v", got.body)
	}
	if more := f.list(root, "?q=nobody&after=a:sample-member"); more.body["scan_stopped"] != false || more.body["next"] != "" {
		t.Errorf("departed segment after a stop %+v", more.body)
	}
	f.s.cfg.Access.UserFilterScan = 1000
	limited := f.list(root, "?limit=14")
	if len(userIDs(limited)) != 14 || limited.body["next"] != "a:sample-member" || limited.body["scan_stopped"] != false {
		t.Errorf("limit reached at the last login %+v", limited.body["next"])
	}
	last := f.list(root, "?limit=15")
	if len(userIDs(last)) != 15 || last.body["next"] != "" {
		t.Errorf("limit reached at the end %+v", last.body["next"])
	}
	fresh := setup(t, SeedSample).ready()
	logins := len(fresh.s.dir.order)
	if got := fresh.list(fresh.as(AdminLogin), fmt.Sprintf("?limit=%d", logins)); len(userIDs(got)) != logins || got.body["next"] != "" {
		t.Errorf("limit reached at the last login with nobody departed %+v", got.body["next"])
	}
}

func TestUsersPagingAcrossChanges(t *testing.T) {
	f, p, root := filteredWorld(t)
	first := f.list(root, "?limit=6")
	version := first.body["list_version"].(string)
	if first.body["order"] != usersOrder || fmt.Sprint(first.body["warning_days"]) != "14" {
		t.Errorf("page facts %+v %+v", first.body["order"], first.body["warning_days"])
	}
	methods := first.body["auth_methods"].(map[string]any)
	if fmt.Sprint(methods["connected"]) != "[password]" || fmt.Sprint(methods["not_connected"]) != "[saml oidc ldap]" {
		t.Errorf("methods %+v", methods)
	}
	f.advance(48 * time.Hour)
	p.refresh()
	root = f.as(AdminLogin)
	if again := f.list(root, "?limit=6"); again.body["list_version"] != version {
		t.Errorf("time alone changed the version %v %v", again.body["list_version"], version)
	}
	p.human("u-aaa", "aaa@example.invalid")
	p.human("u-zed", "zed@example.invalid")
	p.depart("u-hal")
	rest, _ := f.walk(root, "limit=4", first.body["next"].(string))
	if strings.Join(rest, ",") != "u-dan,u-eve,u-fay,u-gus,u-ivy,u-kim,sample-member,u-zed,u-hal,u-jon" {
		t.Errorf("after changes %v", rest)
	}
	changed := f.list(root, "?limit=6").body["list_version"]
	if changed == version {
		t.Error("the version did not change")
	}
	before := f.s.dir.version
	f.s.dir.dropInvite("nobody@example.invalid")
	f.s.dir.dropLink("nobody@example.invalid")
	f.s.dir.dropHolder("nobody")
	if f.s.dir.version != before {
		t.Error("an empty drop changed the version")
	}
	f.s.dir.dropInvite("gus@example.invalid")
	f.s.dir.dropLink("gus@example.invalid")
	f.s.dir.dropHolder("u-kim")
	if f.s.dir.version != before+3 {
		t.Errorf("drops %d", f.s.dir.version-before)
	}
}

func TestUsersPeriodEndingBetweenPages(t *testing.T) {
	f := setupWith(t, func(*Config) {})
	p := newPopulation(f)
	p.member("u-a", "a1@example.invalid", identity.Member)
	p.member("u-b", "b1@example.invalid", identity.Member)
	p.until("u-b", f.s.clock().Add(time.Hour))
	root := f.as(AdminLogin)
	first := f.list(root, "?membership=active&limit=1")
	if strings.Join(userIDs(first), ",") != "u-a" {
		t.Fatalf("first %v", userIDs(first))
	}
	f.advance(time.Hour)
	root = f.as(AdminLogin)
	second := f.list(root, "?membership=active&limit=10&after="+first.body["next"].(string))
	if strings.Contains(strings.Join(userIDs(second), ","), "u-b") {
		t.Errorf("an ended period still active %v", userIDs(second))
	}
	if ended, _ := f.walk(root, "membership=ended"); strings.Join(ended, ",") != "u-b" {
		t.Errorf("ended %v", ended)
	}
	if second.body["server_time"] == first.body["server_time"] {
		t.Error("server time")
	}
}

func TestDepartingWithoutADirectoryEntry(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	if err := f.s.identity.Installation().CreateUser(seedOperator, "u-lone", LoginTriple("lone@example.invalid"), "lone@example.invalid"); err != nil {
		t.Fatal(err)
	}
	if got := f.run(root, "identity.depart_user", "u-lone", nil); got.status != 200 {
		t.Fatalf("depart %+v", got)
	}
	if ids, _ := f.walk(root, "status=departed"); strings.Join(ids, ",") != "u-lone" {
		t.Errorf("departed segment %v", ids)
	}
	if got := f.run(root, "identity.depart_user", SampleMember, nil); got.status != 200 {
		t.Fatalf("depart %+v", got)
	}
	if !sort.StringsAreSorted(f.s.dir.departed) || len(f.s.dir.departed) != 2 || f.s.dir.people[SampleMember] != (person{}) {
		t.Errorf("departed %v", f.s.dir.departed)
	}
	if _, ok := f.s.dir.byLogin[MemberLogin]; ok {
		t.Error("login kept")
	}
	if !f.s.dir.taken[MemberLogin] {
		t.Error("a departed login became free")
	}
}

func TestDirectoryIndexes(t *testing.T) {
	d := newDirectory()
	d.add("u3", "b@x", "b@x")
	if d.version != 1 {
		t.Errorf("an add does not move the version: %d", d.version)
	}
	d.invite("b@x", invite{id: "i"})
	d.issued("b@x", outstanding{purpose: "set"})
	if d.version != 3 {
		t.Errorf("an invitation and a link move the version to %d", d.version)
	}
	d.dropInvite("b@x")
	d.dropLink("b@x")
	d.grant("u3", time.Now())
	d.dropHolder("u3")
	if _, ok := d.invites["b@x"]; ok || d.version != 7 {
		t.Errorf("drops %d %v", d.version, d.invites)
	}
	if _, ok := d.links["b@x"]; ok {
		t.Error("link kept")
	}
	if _, ok := d.holders["u3"]; ok {
		t.Error("holder kept")
	}
	d.add("u1", "c@x", "c@x")
	d.add("u2", "a@x", "a@x")
	d.add("u4", "b@x", "b@x")
	d.add("u1", "c@x", "other@x")
	if strings.Join(d.order, ",") != "u2,u3,u4,u1" || d.people["u1"].email != "other@x" || d.byLogin["b@x"] != "u4" {
		t.Errorf("order %v %+v %v", d.order, d.people, d.byLogin)
	}
	version := d.version
	d.remove("ghost")
	if d.version != version {
		t.Error("removing nobody")
	}
	d.depart("u3")
	d.depart("u0")
	d.depart("u0")
	if strings.Join(d.order, ",") != "u2,u4,u1" || strings.Join(d.departed, ",") != "u0,u3" || len(d.people) != 3 {
		t.Errorf("after departures %v %v", d.order, d.departed)
	}
	if user, ok := d.userOf("a@x"); !ok || user != "u2" {
		t.Error("user of a login")
	}
	if d.position("u4", "b@x") != 1 {
		t.Error("position")
	}
	if d.version != version+3 {
		t.Errorf("versions %d", d.version-version)
	}
}

func TestDepartureClearsTheDirectoryEntry(t *testing.T) {
	f, _, root := filteredWorld(t)
	for _, c := range []struct{ id, login string }{{"u-gus", "gus@example.invalid"}, {"u-hal", "hal@example.invalid"}, {"u-kim", "kim@example.invalid"}} {
		got := f.run(root, "identity.depart_user", c.id, nil)
		if got.status == 409 {
			got = f.run(root, "identity.depart_user", c.id, nil)
		}
		if got.status != 200 {
			t.Fatalf("depart %s %+v", c.id, got)
		}
		_, invited := f.s.dir.invites[c.login]
		_, linked := f.s.dir.links[c.login]
		_, holds := f.s.dir.holders[c.id]
		_, person := f.s.dir.people[c.id]
		if invited || linked || holds || person || f.s.passwords.Has(LoginKey(c.login)) {
			t.Errorf("%s kept: invitation %v link %v holder %v person %v", c.id, invited, linked, holds, person)
		}
	}
}

func TestDirectoryMatchesIdentityHumans(t *testing.T) {
	f, p, root := filteredWorld(t)
	p.depart("u-bob")
	f.run(root, "identity.depart_user", "u-cat", nil)
	humans, departed := []string{}, []string{}
	for after := ""; ; {
		page, err := f.s.identity.Installation().Users(after, 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range page {
			switch {
			case u.Kind != identity.Human:
			case u.State == identity.Departed:
				departed = append(departed, u.ID)
			default:
				humans = append(humans, u.ID)
			}
		}
		if len(page) < 100 {
			break
		}
		after = page[len(page)-1].ID
	}
	listed := append([]string{}, f.s.dir.order...)
	sort.Strings(listed)
	if strings.Join(listed, ",") != strings.Join(humans, ",") || strings.Join(f.s.dir.departed, ",") != strings.Join(departed, ",") {
		t.Errorf("directory %v %v; identity %v %v", listed, f.s.dir.departed, humans, departed)
	}
	if len(f.s.dir.people) != len(f.s.dir.order) || len(f.s.dir.byLogin) != len(f.s.dir.order) {
		t.Error("index sizes")
	}
}

func TestFindUserEqualsAPagedScan(t *testing.T) {
	f, p, _ := filteredWorld(t)
	if err := f.s.identity.Workspace().CreateServiceUser(p.admin, SampleWorkspace, "a-service", SampleAdmin); err != nil {
		t.Fatal(err)
	}
	all, err := f.s.identity.Installation().Users("", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range append(all, identity.UserInfo{ID: "ghost"}) {
		got, ok, err := f.s.findUser(u.ID)
		want := u.Kind == identity.Human
		if err != nil || ok != want || (u.Kind != "" && got != u) {
			t.Errorf("%s: %+v %v %v", u.ID, got, ok, err)
		}
	}
}

func TestInvitationsFilterEqualsTheConsoleRule(t *testing.T) {
	f, _, root := filteredWorld(t)
	all := f.list(root, "?limit=100").body["users"].([]any)
	want := []string{}
	for _, u := range all {
		row := u.(map[string]any)
		if row["invitation_expires"] != "" || row["link_purpose"] == "set" {
			want = append(want, row["user_id"].(string))
		}
	}
	if got, _ := f.walk(root, "invitation=pending"); strings.Join(got, ",") != strings.Join(want, ",") || len(want) == 0 {
		t.Errorf("invitations %v; want %v", got, want)
	}
}

func TestUsersReadTopLevelKeys(t *testing.T) {
	f := setup(t, SeedSample).ready()
	got := f.list(f.as(AdminLogin), "")
	allowed := map[string]bool{"revision": true, "next": true, "next_login": true, "users": true, "server_time": true, "wall_time": true, "clock_behind": true, "clock_ahead": true,
		"clock_base": true, "wall_lead_beyond_tolerance": true, "recovery_state": true, "settings": true, "ledger_state": true, "can_bulk_revoke": true, "filters": true,
		"scan_stopped": true, "scan_budget": true, "order": true, "list_version": true, "auth_methods": true, "warning_days": true}
	for key := range got.body {
		if !allowed[key] {
			t.Errorf("unexpected key %s", key)
		}
	}
	if len(got.body) != len(allowed) {
		t.Errorf("keys %d", len(got.body))
	}
}

func TestUsersReadIsRecordedInTheDocument(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := f.do("GET", "/api/v1/openapi.json", "", nil)
	read := doc.body["x-reads"].(map[string]any)["/identity/users"].(map[string]any)["get"].(map[string]any)
	responses := read["responses"].(map[string]any)
	conflict := responses["409"].(map[string]any)
	if conflict["x-code"] != string(commands.RevisionConflict) || conflict["x-next-action"] != string(commands.NextInspect) || fmt.Sprint(conflict["x-fields"]) != "[after]" {
		t.Errorf("409 %+v", conflict)
	}
	if fmt.Sprint(responses["422"].(map[string]any)["x-fields"]) != "[after]" || responses["422"].(map[string]any)["x-next-action"] != string(commands.NextCorrect) {
		t.Errorf("422 %+v", responses["422"])
	}
	for status, code := range map[string]commands.Code{"401": commands.Unauthenticated, "403": commands.PermissionDenied, "501": commands.UnsupportedCapability,
		"503": commands.DependencyUnavailable} {
		got := responses[status].(map[string]any)
		_, fields := got["x-fields"]
		if got["x-code"] != string(code) || fields {
			t.Errorf("%s %+v", status, got)
		}
	}
	for status, want := range map[string]string{"401": "[unauthenticated]", "403": "[csrf_refused permission_denied]", "409": "[revision_conflict]", "422": "[invalid_request]",
		"501": "[unsupported_capability]", "503": "[authority_unavailable dependency_unavailable]"} {
		if got := fmt.Sprint(responses[status].(map[string]any)["x-codes"]); got != want {
			t.Errorf("%s codes %s; want %s", status, got, want)
		}
	}
	if responses["422"].(map[string]any)["x-fields-when"] != "cursor" {
		t.Errorf("422 field condition %+v", responses["422"])
	}
	if responses["401"].(map[string]any)["x-next-action"] != string(commands.NextAnswer) || responses["503"].(map[string]any)["x-next-action"] != string(commands.NextRetry) ||
		responses["501"].(map[string]any)["x-next-action"] != string(commands.NextNone) {
		t.Errorf("next actions %+v", responses)
	}
	refused := read["x-refused-parameters"].(map[string]any)
	if refused["method"] != string(commands.UnsupportedCapability) || refused["workspace"] != string(commands.InvalidRequest) || read["x-order"] != usersOrder {
		t.Errorf("read %+v", read)
	}
	if len(read["parameters"].([]any)) != 10 {
		t.Errorf("parameters %d", len(read["parameters"].([]any)))
	}
}

func TestUserFilterScanSetting(t *testing.T) {
	for _, value := range []int{99, 10001} {
		_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: AccessSettings{UserFilterScan: value}})
		if err == nil || !strings.HasSuffix(err.Error(), "user_filter_scan") {
			t.Errorf("%d: %v", value, err)
		}
	}
	for _, value := range []int{100, 10000} {
		if _, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: AccessSettings{UserFilterScan: value}}); err != nil {
			t.Errorf("%d: %v", value, err)
		}
	}
	if (AccessSettings{}).withDefaults().UserFilterScan != 1000 || DefaultAccessSettings().UserFilterScan != 1000 || len(UserSettingList()) != 1 {
		t.Error("default")
	}
}

func TestTwoInstallationsNeverMix(t *testing.T) {
	one, two := setupWith(t, func(*Config) {}), setupWith(t, func(*Config) {})
	newPopulation(one).human("u-one", "one@example.invalid")
	other := newPopulation(two)
	other.human("u-two", "two@example.invalid")
	if _, err := one.s.usersPage(other.admin, usersQuery{limit: 50}); err == nil {
		t.Error("a session of another installation read this installation")
	}
	ids, _ := one.walk(one.as(AdminLogin), "limit=100")
	if strings.Contains(strings.Join(ids, ","), "u-two") || !strings.Contains(strings.Join(ids, ","), "u-one") {
		t.Errorf("rows %v", ids)
	}
}
