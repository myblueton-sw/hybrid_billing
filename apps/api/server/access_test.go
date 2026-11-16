package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (f world) accessRead(a actor, path string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/access"+path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) grantTo(a actor, subject string, fields map[string]string) reply {
	f.t.Helper()
	body := map[string]string{"subject": subject, "action": "view", "kind": ReaderKind, "fields": ReaderField, "customer": "all", "days": "10"}
	for k, v := range fields {
		body[k] = v
	}
	return f.run(a, "access.grant", grantsTarget, body)
}

func clientGrants(f world) int {
	n := 0
	for _, g := range f.s.admin.Grants() {
		if g.ClientID != "" {
			n++
		}
	}
	return n
}

func TestAccessReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.accessRead(root, "")
	if got.status != 200 {
		t.Fatalf("read %+v", got)
	}
	ids := []string{}
	for _, g := range got.body["grants"].([]any) {
		row := g.(map[string]any)
		ids = append(ids, row["id"].(string))
		if row["action"] == string(gate.AdministerInstallation) {
			t.Error("installation grant listed")
		}
	}
	if strings.Join(ids, ",") != "reader-console,reader-sample-admin,reader-sample-admin-2" {
		t.Errorf("grants %v", ids)
	}
	first := got.body["grants"].([]any)[1].(map[string]any)
	if first["state"] != "active" || first["effective"] != true || first["subject"] != SampleAdmin || first["scope"].(map[string]any)["customer"] != "all" || first["scope"].(map[string]any)["period"] != "any" {
		t.Errorf("grant row %+v", first)
	}
	catalog := got.body["catalog"].([]any)
	if len(catalog) != 1 || catalog[0].(map[string]any)["kind"] != ReaderKind || catalog[0].(map[string]any)["until"] != stamp(start.Add(readerValidity)) {
		t.Errorf("catalog %+v", catalog)
	}
	subjects := fmt.Sprint(got.body["subjects"])
	if !strings.Contains(subjects, SampleMember) || !strings.Contains(subjects, MemberLogin) || len(got.body["subjects"].([]any)) != 3 {
		t.Errorf("subjects %s", subjects)
	}
	f.code(t, "no session", f.accessRead(actor{}, ""), 401, string(commands.Unauthenticated))
	f.code(t, "member", f.accessRead(f.as(MemberLogin), ""), 403, string(commands.PermissionDenied))
}

func TestGrantLetsAMemberReadThroughTheConsole(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.setPassword(MemberLogin, pw)
	root, member := f.as(AdminLogin), f.as(MemberLogin)
	read := func() reply {
		return f.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(member.cookie) })
	}
	f.code(t, "before grant", read(), 403, string(commands.PermissionDenied))
	clients := clientGrants(f)
	g := f.grantTo(root, SampleMember, nil)
	if g.status != 200 || g.body["business_state"] != "granted" {
		t.Fatalf("grant %+v", g)
	}
	if r := read(); r.status != 200 {
		t.Errorf("after grant %+v", r)
	}
	if clientGrants(f) != clients {
		t.Error("a command added a client grant")
	}
	id := g.body["fields"].(map[string]any)["grant"].(string)
	if r := f.run(root, "access.revoke_grant", id, nil); r.status != 200 || r.body["business_state"] != "grant_revoked" {
		t.Fatalf("revoke %+v", r)
	}
	f.code(t, "after revoke", read(), 403, string(commands.PermissionDenied))
	f.code(t, "revoke twice", f.run(root, "access.revoke_grant", id, nil), 422, string(commands.InvalidStage))
}

func TestGrantValidation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	f.code(t, "self", f.grantTo(root, SampleAdmin, nil), 422, string(commands.PrerequisiteFailed))
	for _, c := range []struct {
		subject string
		change  map[string]string
		field   string
	}{
		{"nobody", nil, "subject"},
		{SampleMember, map[string]string{"action": "issue"}, "action"},
		{SampleMember, map[string]string{"kind": gate.InstallationKind}, "action"},
		{SampleMember, map[string]string{"fields": "other"}, "action"},
		{SampleMember, map[string]string{"fields": ""}, "action"},
		{SampleMember, map[string]string{"fields": "rows,rows"}, "fields"},
		{SampleMember, map[string]string{"fields": "rows,"}, "fields"},
		{SampleMember, map[string]string{"customer": "c1"}, "customer"},
		{SampleMember, map[string]string{"customer": "a b"}, "customer"},
		{SampleMember, map[string]string{"days": "0"}, "days"},
		{SampleMember, map[string]string{"days": "31"}, "days"},
	} {
		got := f.grantTo(root, c.subject, c.change)
		if got.status != 422 || !strings.Contains(got.raw, `"`+c.field+`"`) {
			t.Errorf("%v: %s", c.change, got.raw)
		}
	}
	if got := f.grantTo(root, SampleMember, map[string]string{"days": "30"}); got.status != 200 {
		t.Errorf("exact remaining days %s", got.raw)
	}
	if got := f.grantTo(root, SampleAdmin2, nil); got.status != 200 {
		t.Errorf("the declared cross-grant %s", got.raw)
	}
	f.code(t, "long rationale", f.admin(root, "access.grant", grantsTarget, strings.Repeat("x", maxRationale+1),
		map[string]string{"subject": SampleMember, "action": "view", "kind": ReaderKind, "fields": ReaderField, "customer": "all", "days": "1"}), 422, string(commands.InvalidRequest))
	f.code(t, "wrong target", f.run(root, "access.grant", "elsewhere", map[string]string{"subject": SampleMember, "action": "view", "kind": ReaderKind, "fields": ReaderField, "customer": "all", "days": "1"}),
		403, string(commands.PermissionDenied))
	if _, res := (accessCommand{s: f.s, kind: "grant"}).Resolve("elsewhere", grantsTarget); res != commands.NotFound {
		t.Error("other workspace")
	}
}

func TestRevokeRules(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for _, id := range []string{"installation-" + SampleAdmin2, "reader-console", "reader-" + SampleAdmin} {
		f.code(t, id, f.run(root, "access.revoke_grant", id, nil), 422, string(commands.PrerequisiteFailed))
	}
	f.code(t, "unknown grant", f.run(root, "access.revoke_grant", "missing", nil), 403, string(commands.PermissionDenied))
	if got := f.run(root, "access.revoke_grant", "reader-"+SampleAdmin2, nil); got.status != 200 {
		t.Errorf("revoke another holder's reader grant %s", got.raw)
	}
	f.grantTo(root, SampleMember, nil)
	now := f.s.clock()
	terms := gate.Terms{Action: gate.View, Kind: ReaderKind, Fields: []string{ReaderField}, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	f.s.grantInstallation(SampleAdmin, now)
	if err := f.s.admin.AddDelegation(gate.Delegation{ID: "del-1", DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	read := f.accessRead(root, "")
	if !strings.Contains(read.raw, `"delegator":"sample-admin"`) {
		t.Errorf("delegation listed %s", read.raw)
	}
	if got := f.run(root, "access.revoke_delegation", "del-1", nil); got.status != 200 || got.body["business_state"] != "delegation_revoked" {
		t.Errorf("revoke delegation %s", got.raw)
	}
	f.code(t, "revoke delegation twice", f.run(root, "access.revoke_delegation", "del-1", nil), 422, string(commands.InvalidStage))
	f.code(t, "unknown delegation", f.run(root, "access.revoke_delegation", "missing", nil), 403, string(commands.PermissionDenied))
}

func TestAccessPreview(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	before := f.accessRead(root, "/preview?subject="+SampleMember+"&kind="+ReaderKind)
	if before.status != 200 {
		t.Fatalf("preview %s", before.raw)
	}
	decisions := before.body["decisions"].([]any)
	pick := func(r reply, field string) map[string]any {
		for _, d := range r.body["decisions"].([]any) {
			if row := d.(map[string]any); row["action"] == "view" && row["field"] == field {
				return row
			}
		}
		return nil
	}
	if view := pick(before, ReaderField); view["allowed"] != false || view["reason"] != string(gate.ReasonNoGrant) {
		t.Errorf("before %+v", view)
	}
	for _, d := range decisions {
		if d.(map[string]any)["action"] == "view_cost" {
			t.Error("view_cost listed without cost fields")
		}
	}
	f.grantTo(root, SampleMember, nil)
	after := f.accessRead(root, "/preview?subject="+SampleMember+"&kind="+ReaderKind+"&customer=c1")
	if pick(after, ReaderField)["allowed"] != true || pick(after, "state")["allowed"] != false {
		t.Errorf("after %s", after.raw)
	}
	for _, q := range []string{"?subject=nobody&kind=" + ReaderKind, "?subject=" + SampleMember + "&kind=" + gate.InstallationKind, "?subject=" + SampleMember + "&kind=" + ReaderKind + "&customer=a%20b"} {
		f.code(t, q, f.accessRead(root, "/preview"+q), 422, string(commands.InvalidRequest))
	}
	cost := setupWith(t, func(c *Config) {
		c.Kinds = []string{"usage"}
		c.Fields = map[string]gate.FieldClass{"unit_cost": gate.Cost}
	})
	cost.setPassword(MemberLogin, pw)
	got := cost.accessRead(cost.as(AdminLogin), "/preview?subject="+SampleMember+"&kind=usage")
	if !strings.Contains(got.raw, `"action":"view_cost"`) {
		t.Errorf("cost preview %s", got.raw)
	}
}

func TestEffectiveAfterMembershipChange(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	id := f.grantTo(root, SampleMember, nil).body["fields"].(map[string]any)["grant"].(string)
	f.run(root, "identity.remove_membership", SampleMember, nil)
	f.run(root, "identity.invite", SampleMember, map[string]string{"role": "member"})
	f.signin(MemberLogin, pw)
	for _, g := range f.accessRead(root, "").body["grants"].([]any) {
		row := g.(map[string]any)
		if row["id"] == id && (row["state"] != "active" || row["effective"] != false) {
			t.Errorf("grant after membership change %+v", row)
		}
	}
}

func TestAccessStatesAndErrors(t *testing.T) {
	now := start
	v := gate.Validity{NotBefore: now.Add(time.Hour), NotAfter: now.Add(2 * time.Hour)}
	for want, state := range map[string]string{"revoked": stateOf(true, v, now), "pending": stateOf(false, v, now), "expired": stateOf(false, v, now.Add(3*time.Hour)), "active": stateOf(false, v, now.Add(90*time.Minute))} {
		if want != state {
			t.Errorf("%s: %s", want, state)
		}
	}
	from, _ := gate.NewMonth(2026, 1)
	to, _ := gate.NewMonth(2026, 3)
	if got := scopeOf(gate.Terms{Scope: gate.Scope{Customer: gate.One("c1"), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.Months(from, to)}); got.Customer != "c1" || got.Period == "any" {
		t.Errorf("scope %+v", got)
	}
	for _, c := range []struct {
		err  error
		code commands.Code
	}{{gate.ErrConflict, commands.InvalidStage}, {gate.ErrNotFound, commands.InvalidStage}, {gate.ErrChanged, commands.PrerequisiteFailed}, {gate.ErrNotBacked, commands.PrerequisiteFailed},
		{gate.ErrUnavailable, commands.DependencyUnavailable}, {gate.ErrInvalid, commands.InvalidRequest}} {
		if got := mapAccess(c.err).(*commands.Failure).Code; got != c.code {
			t.Errorf("%v: %s", c.err, got)
		}
	}
	if f, ok := splitFields(""); f != nil || !ok {
		t.Error("empty fields")
	}
	if !covers([]string{"a", "b"}, []string{"b"}) || covers([]string{"a"}, []string{"b"}) {
		t.Error("field helpers")
	}
	f := setup(t, SeedSample).ready()
	if !strings.Contains(string(f.mustOpenAPI()), "access.grant") {
		t.Error("openapi")
	}
}

func TestConcurrentAccessReadsAndGrants(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if a, b := f.accessRead(root, ""), f.accessRead(root, "/preview?subject="+SampleMember+"&kind="+ReaderKind); a.status != 200 || b.status != 200 {
					t.Errorf("concurrent read %d %d", a.status, b.status)
				}
			}
		}()
	}
	for i := 0; i < 3; i++ {
		f.advance(time.Second)
		body := fmt.Sprintf(`{"request_id":"g%d","target_id":"grants","idempotency_key":"g%d","expected_revision":"%s","rationale":"x","body":{"subject":"%s","action":"view","kind":"%s","fields":"%s","customer":"all","days":"5"}}`,
			i, i, f.s.Revision(), SampleMember, ReaderKind, ReaderField)
		f.call("access.grant", body, root.cookie, root.csrf)
	}
	wg.Wait()
}

func TestAccessFailurePaths(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Kinds = []string{"usage"}
		c.Fields = map[string]gate.FieldClass{"state": gate.General}
	})
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	f.code(t, "preview without session", f.accessRead(actor{}, "/preview?subject=x&kind=usage"), 401, string(commands.Unauthenticated))
	terms := gate.Terms{Action: gate.View, Kind: "usage", Fields: []string{"state"}, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	if err := f.s.admin.AddGrant(gate.Grant{ID: "usage-console", ClientID: ConsoleClient, Terms: terms, Validity: gate.Validity{NotBefore: start, NotAfter: start.Add(60 * 24 * time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	usage := map[string]string{"kind": "usage", "fields": "state", "customer": "c1", "days": "40"}
	if got := f.grantTo(root, SampleMember, usage); got.status != 200 {
		t.Errorf("customer-scoped grant on a kind with a ceiling %s", got.raw)
	}
	f.s.fault = func(at string) bool { return at == "access" }
	f.code(t, "module refusal", f.grantTo(root, SampleMember, usage), 422, string(commands.PrerequisiteFailed))
	f.s.fault = func(at string) bool { return at == "roster" }
	f.code(t, "read", f.accessRead(root, ""), 503, string(commands.DependencyUnavailable))
	f.code(t, "preview", f.accessRead(root, "/preview?subject="+SampleMember+"&kind=usage"), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	saved := f.s.admin
	f.s.admin = &gate.Administration{}
	for _, kind := range []string{"revoke_grant", "revoke_delegation"} {
		if _, err := (accessCommand{s: f.s, kind: kind}).Apply(commands.Context{Target: "x", User: "y"}); err.(*commands.Failure).Code != commands.DependencyUnavailable {
			t.Errorf("%s: %v", kind, err)
		}
	}
	f.s.admin = saved
	f.s.caller = root.verifier(f)
	f.s.fault = func(at string) bool { return at == "roster" }
	if _, err := (accessCommand{s: f.s, kind: "grant"}).Apply(commands.Context{User: SampleAdmin, Fields: map[string]string{"subject": SampleMember}}); err.(*commands.Failure).Code != commands.DependencyUnavailable {
		t.Errorf("grant without roster: %v", err)
	}
	f.s.fault = nil
	f.s.caller = [32]byte{}
}

func TestCeilingNeedsFullScopeAndGranteesNeedActiveState(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Kinds = []string{"usage"}
		c.Fields = map[string]gate.FieldClass{"state": gate.General}
	})
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	from, _ := gate.NewMonth(2026, 1)
	to, _ := gate.NewMonth(2026, 12)
	full := gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}
	narrow := []func(*gate.Terms){
		func(t *gate.Terms) { t.Scope.Customer = gate.One("c1") },
		func(t *gate.Terms) { t.Scope.LegalEntity = gate.One("e1") },
		func(t *gate.Terms) { t.Scope.Contract = gate.One("k1") },
		func(t *gate.Terms) { t.Period = gate.Months(from, to) },
	}
	for i, change := range narrow {
		terms := gate.Terms{Action: gate.View, Kind: "usage", Fields: []string{"state"}, Period: gate.AnyPeriod(), Scope: full}
		change(&terms)
		if err := f.s.admin.AddGrant(gate.Grant{ID: fmt.Sprintf("narrow-%d", i), ClientID: ConsoleClient, Terms: terms, Validity: gate.Validity{NotBefore: start, NotAfter: start.Add(48 * time.Hour)}}); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(fmt.Sprint(f.accessRead(root, "").body["catalog"]), "usage") {
		t.Error("a narrow client grant counted as ceiling")
	}
	f.code(t, "grant under a narrow ceiling", f.grantTo(root, SampleMember, map[string]string{"kind": "usage", "fields": "state", "days": "1"}), 422, string(commands.InvalidRequest))
	f.run(root, "identity.suspend_user", SampleMember, nil)
	if strings.Contains(fmt.Sprint(f.accessRead(root, "").body["subjects"]), SampleMember) {
		t.Error("a suspended user is a grantee")
	}
	f.run(root, "identity.reinstate_user", SampleMember, nil)
	f.run(root, "identity.suspend_membership", SampleMember, nil)
	if strings.Contains(fmt.Sprint(f.accessRead(root, "").body["subjects"]), SampleMember) {
		t.Error("a suspended membership is a grantee")
	}
	doc := map[string]any{}
	_ = json.Unmarshal(f.mustOpenAPI(), &doc)
	raw, _ := json.Marshal(doc)
	if !strings.Contains(string(raw), `"required":["action","customer","days","kind","subject"]`) {
		t.Error("access.grant required fields")
	}
}
