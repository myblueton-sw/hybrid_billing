package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

type actor struct {
	cookie *http.Cookie
	csrf   string
}

var keys atomic.Int64

func (f world) as(login string) actor {
	f.t.Helper()
	cookie, csrf, err := f.s.OpenSession(LoginTriple(login), SampleWorkspace, false)
	if err != nil {
		f.t.Fatal(err)
	}
	return actor{cookie, csrf}
}

func (f world) list(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/users"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) revision(a actor) string {
	f.t.Helper()
	got := f.list(a, "")
	if got.status != 200 {
		f.t.Fatalf("list %+v", got)
	}
	return got.body["revision"].(string)
}

func (f world) row(a actor, user string) map[string]any {
	f.t.Helper()
	got := f.list(a, "")
	for _, u := range got.body["users"].([]any) {
		if row := u.(map[string]any); row["user_id"] == user {
			return row
		}
	}
	return nil
}

func (f world) admin(a actor, command, target, rationale string, fields map[string]string) reply {
	f.t.Helper()
	n := keys.Add(1)
	body, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("req-%d", n), "target_id": target, "idempotency_key": fmt.Sprintf("key-%d", n),
		"expected_revision": f.revision(a), "rationale": rationale, "body": fields})
	return f.call(command, string(body), a.cookie, a.csrf)
}

func (f world) run(a actor, command, target string, fields map[string]string) reply {
	f.t.Helper()
	if fields == nil {
		fields = map[string]string{}
	}
	return f.admin(a, command, target, "sample reason", fields)
}

func (f world) collect(a actor, ref string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/links/"+ref, "", func(r *http.Request) {
		r.AddCookie(a.cookie)
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func linkToken(url string) string { return url[strings.Index(url, "#")+1:] }

func (f world) redeem(tok, password string) reply {
	f.t.Helper()
	return f.auth("redeem", `{"token":"`+tok+`","password":"`+password+`"}`, f.pre())
}

func (f world) signin(login, password string) reply {
	f.t.Helper()
	return f.auth("login", loginBodyFor(login, password), f.pre())
}

func (f world) created(a actor, login, role string) (string, string) {
	f.t.Helper()
	got := f.run(a, "identity.create_user", usersTarget, map[string]string{"login": login, "email": login, "role": role})
	if got.status != 200 {
		f.t.Fatalf("create %+v", got)
	}
	fields := got.body["fields"].(map[string]any)
	link := f.collect(a, fields["link_ref"].(string))
	if link.status != 200 || link.header.Get("Cache-Control") != "no-store" {
		f.t.Fatalf("collect %+v", link)
	}
	return fields["user_id"].(string), linkToken(link.body["url"].(string))
}

func TestUserAdministrationPermissions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	if got := f.list(root, ""); got.status != 200 {
		t.Fatalf("grant holder %+v", got)
	}
	f.code(t, "administrator without grant", f.list(second, ""), 403, string(commands.PermissionDenied))
	f.code(t, "member", f.list(member, ""), 403, string(commands.PermissionDenied))
	f.code(t, "no session", f.list(actor{}, ""), 401, string(commands.Unauthenticated))
	f.code(t, "bad csrf", f.list(actor{root.cookie, "wrong"}, ""), 403, CodeCSRF)
	for _, d := range f.s.userDefinitions() {
		target := SampleMember
		if d.ID == "identity.create_user" {
			target = usersTarget
		}
		body := `{"request_id":"r","target_id":"` + target + `","idempotency_key":"k-` + d.ID + `","expected_revision":"` + f.s.Revision() + `","rationale":"x","body":{}}`
		got := f.call(d.ID, body, second.cookie, second.csrf)
		if got.status != 403 && got.status != 422 {
			t.Errorf("%s for an administrator without grant: %+v", d.ID, got)
		}
		if got.body["code"] == string(commands.InvalidRequest) {
			continue
		}
		f.code(t, d.ID, got, 403, string(commands.PermissionDenied))
	}
	f.code(t, "collect without session", f.do("GET", "/api/v1/identity/links/x", "", nil), 401, string(commands.Unauthenticated))
	f.code(t, "collect bad csrf", f.collect(actor{root.cookie, "wrong"}, "x"), 403, CodeCSRF)
	f.code(t, "collect by a member", f.collect(member, "x"), 403, string(commands.PermissionDenied))
	f.code(t, "bad limit", f.list(root, "?limit=0"), 422, string(commands.InvalidRequest))
	f.code(t, "large limit", f.list(root, "?limit=101"), 422, string(commands.InvalidRequest))
	f.code(t, "bad cursor", f.list(root, "?after=a%20b"), 422, string(commands.InvalidRequest))
}

func TestCreateUserAndAcceptAtFirstPassword(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	before := f.s.counter
	got := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "New@Example.Invalid", "email": "new@example.invalid", "role": "member"})
	if got.status != 200 || got.body["business_state"] != "user_created" || f.s.counter != before+1 {
		t.Fatalf("create %+v", got)
	}
	fields := got.body["fields"].(map[string]any)
	user, ref := fields["user_id"].(string), fields["link_ref"].(string)
	if strings.Contains(got.raw, "activate#") {
		t.Error("link in the result")
	}
	other := f.as(AdminLogin)
	f.code(t, "pickup by another session", f.collect(other, ref), 403, string(commands.PermissionDenied))
	link := f.collect(root, ref)
	if link.status != 200 || !strings.HasPrefix(link.body["url"].(string), "http://127.0.0.1:5173/activate#") {
		t.Fatalf("pickup %+v", link)
	}
	f.code(t, "second pickup", f.collect(root, ref), 403, string(commands.PermissionDenied))
	row := f.row(root, user)
	if row["login"] != "new@example.invalid" || row["membership"] != nil || row["invitation_expires"] == "" || row["link_purpose"] != "set" || row["password_set"] != false {
		t.Errorf("row before redeem %+v", row)
	}
	if got := f.redeem(linkToken(link.body["url"].(string)), pw); got.status != 204 {
		t.Fatalf("redeem %+v", got)
	}
	row = f.row(root, user)
	m, _ := row["membership"].(map[string]any)
	if m["role"] != "member" || m["state"] != "active" || row["invitation_expires"] != "" || row["link_purpose"] != "" || row["password_set"] != true {
		t.Errorf("row after redeem %+v", row)
	}
	if got := f.signin("new@example.invalid", pw); got.status != 200 {
		t.Errorf("sign-in %+v", got)
	}
	f.code(t, "taken login", f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "new@example.invalid", "email": "x@example.invalid", "role": "member"}),
		422, string(commands.PrerequisiteFailed))
	f.code(t, "seeded login", f.run(root, "identity.create_user", usersTarget, map[string]string{"login": AdminLogin, "email": "x@example.invalid", "role": "member"}),
		422, string(commands.PrerequisiteFailed))
	spaced := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": " spaced@example.invalid", "email": "x@example.invalid", "role": "member"})
	if spaced.status != 422 || !strings.Contains(spaced.raw, `"login"`) {
		t.Errorf("spaced login %+v", spaced)
	}
	bad := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "no", "email": "x@example.invalid", "role": "member"})
	if bad.status != 422 || !strings.Contains(bad.raw, `"login"`) {
		t.Errorf("bad login %+v", bad)
	}
	bad = f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "ok@example.invalid", "email": "nope", "role": "member"})
	if bad.status != 422 || !strings.Contains(bad.raw, `"email"`) {
		t.Errorf("bad email %+v", bad)
	}
	f.code(t, "wrong target", f.run(root, "identity.create_user", "elsewhere", map[string]string{"login": "ok@example.invalid", "email": "ok@example.invalid", "role": "member"}),
		403, string(commands.PermissionDenied))
}

func TestCreateUserLimitsAndPartialFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	create := func(login string) reply {
		return f.run(root, "identity.create_user", usersTarget, map[string]string{"login": login, "email": login, "role": "member"})
	}
	f.s.cfg.Identity.PendingInvitations = 1
	f.code(t, "invitation capacity", create("cap1@example.invalid"), 503, string(commands.DependencyUnavailable))
	f.s.cfg.Identity.PendingInvitations = 1000
	f.s.cfg.Passwords.Links = 3
	f.code(t, "link capacity", create("cap2@example.invalid"), 503, string(commands.DependencyUnavailable))
	f.s.cfg.Passwords.Links = 10000
	for _, step := range []string{"invite", "link"} {
		f.s.fault = func(at string) bool { return at == step }
		before := f.s.counter
		login := step + "@example.invalid"
		got := create(login)
		f.code(t, step+" fault", got, 500, string(commands.InternalError))
		if got.body["operation_id"] == "" || f.s.counter != before+1 {
			t.Errorf("%s fault: %+v counter %d", step, got, f.s.counter-before)
		}
		f.s.fault = nil
		user, _ := f.s.dir.userOf(login)
		if row := f.row(root, user); row == nil || row["login"] != login {
			t.Errorf("%s partial row %+v", step, row)
		}
	}
	user, _ := f.s.dir.userOf("invite@example.invalid")
	if got := f.run(root, "identity.invite", user, map[string]string{"role": "member"}); got.status != 200 {
		t.Errorf("recover invitation %+v", got)
	}
	if got := f.run(root, "identity.issue_set_link", user, nil); got.status != 200 {
		t.Errorf("recover link %+v", got)
	}
}

func TestProtectionRules(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, c := range []struct {
		command string
		fields  map[string]string
	}{{"identity.suspend_user", nil}, {"identity.depart_user", nil}, {"identity.change_role", map[string]string{"role": "member"}},
		{"identity.change_role", map[string]string{"role": "administrator"}},
		{"identity.suspend_membership", nil}, {"identity.remove_membership", nil}, {"identity.issue_reset_link", nil}} {
		f.code(t, "self "+c.command, f.run(root, c.command, SampleAdmin, c.fields), 422, string(commands.PrerequisiteFailed))
	}
	f.setPassword(Admin2Login, pw)
	f.code(t, "reset link for an administrator", f.run(root, "identity.issue_reset_link", SampleAdmin2, nil), 422, string(commands.PrerequisiteFailed))
	if got := f.run(root, "identity.suspend_user", SampleAdmin2, nil); got.status != 200 || got.body["business_state"] != "user_suspended" {
		t.Errorf("suspend another administrator %+v", got)
	}
	if got := f.run(root, "identity.reinstate_user", SampleAdmin2, nil); got.status != 200 {
		t.Errorf("reinstate %+v", got)
	}
	members, _ := f.s.roster(root.verifier(f))
	now := f.s.clock()
	check := func(name, target, user string, want bool) {
		t.Helper()
		err := f.s.protect(&change{ctx: commands.Context{Target: target, User: user}, members: members, now: now})
		if (err == nil) != want {
			t.Errorf("%s: %v", name, err)
		}
	}
	check("a member", SampleMember, SampleAdmin, true)
	check("an administrator while a holder remains", SampleAdmin2, SampleAdmin, true)
	check("the only holder", SampleAdmin, SampleAdmin2, false)
	delete(members, SampleAdmin2)
	check("the last administrator", SampleAdmin, "someone", false)
	plain := map[string]identity.MemberInfo{
		"m":  {User: "m", Role: identity.Member, State: identity.MembershipActive, UserState: identity.Active},
		"sa": {User: "sa", Role: identity.Administrator, State: identity.MembershipSuspended, UserState: identity.Active},
		"su": {User: "su", Role: identity.Administrator, State: identity.MembershipActive, UserState: identity.Suspended},
		"y":  {User: "y", Role: identity.Administrator, State: identity.MembershipActive, UserState: identity.Active},
	}
	for _, target := range []string{"m", "sa", "su"} {
		if err := f.s.protect(&change{ctx: commands.Context{Target: target, User: "z"}, members: plain, now: now}); err != nil {
			t.Errorf("%s is not an active administrator: %v", target, err)
		}
	}
	err := f.s.protect(&change{ctx: commands.Context{Target: "y", User: "z"}, members: plain, now: now})
	if failure, ok := err.(*commands.Failure); !ok || failure.Code != commands.PrerequisiteFailed {
		t.Errorf("only non-holders remain: %v", err)
	}
	f.s.grantInstallation(SampleAdmin2, now)
	f.code(t, "self suspend while another holder remains", f.run(root, "identity.suspend_user", SampleAdmin, nil), 422, string(commands.PrerequisiteFailed))
	f.s.dir.holders[SampleAdmin2] = now
	if row := f.row(root, SampleAdmin2); row["installation_administrator"] != false {
		t.Errorf("expired holder %+v", row)
	}
	f.code(t, "long rationale on a workspace command", f.admin(root, "identity.suspend_membership", SampleAdmin2, strings.Repeat("x", maxRationale+1), map[string]string{}),
		422, string(commands.InvalidRequest))
}

func (a actor) verifier(f world) identity.Verifier {
	found, _ := f.s.lookup(func() *http.Request {
		r, _ := http.NewRequest("GET", "/", nil)
		r.AddCookie(a.cookie)
		return r
	}())
	return found.verifier
}

func TestResetLinksAndPromotion(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.code(t, "reset before a password", f.run(root, "identity.issue_reset_link", SampleMember, nil), 422, string(commands.InvalidStage))
	f.setPassword(MemberLogin, pw)
	f.code(t, "set link after a password", f.run(root, "identity.issue_set_link", SampleMember, nil), 422, string(commands.InvalidStage))
	member := f.as(MemberLogin)
	got := f.run(root, "identity.issue_reset_link", SampleMember, nil)
	if got.status != 200 {
		t.Fatalf("reset link %+v", got)
	}
	f.code(t, "promotion with an outstanding link", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), 422,
		string(commands.PrerequisiteFailed))
	link := f.collect(root, got.body["fields"].(map[string]any)["link_ref"].(string))
	f.code(t, "promotion with a used pickup but live link", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), 422,
		string(commands.PrerequisiteFailed))
	generation := f.s.dir.generations[MemberLogin]
	f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "bystander@example.invalid", "email": "bystander@example.invalid", "role": "member"})
	f.s.dir.pickups["member-session"] = pickup{session: member.verifier(f), target: "other", expires: f.s.clock().Add(time.Minute)}
	if got := f.redeem(linkToken(link.body["url"].(string)), "another sample password"); got.status != 204 {
		t.Fatalf("reset redeem %+v", got)
	}
	if _, ok := f.s.sessions.get(member.verifier(f)); ok || len(f.s.sessions.byLogin(MemberLogin)) != 0 {
		t.Error("session kept after reset")
	}
	if f.s.dir.generations[MemberLogin] != generation+1 {
		t.Error("generation not bumped")
	}
	if _, kept := f.s.dir.pickups["member-session"]; kept || len(f.s.dir.pickups) != 1 {
		t.Errorf("pickups after reset %d", len(f.s.dir.pickups))
	}
	if _, _, _, ok := f.s.signIn(MemberLogin, SampleWorkspace, false, generation); ok {
		t.Error("a sign-in verified before the reset opened a session")
	}
	if got := f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}); got.status != 200 || got.body["business_state"] != "role_administrator" {
		t.Errorf("promotion after the link was used %+v", got)
	}
	if got := f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "member"}); got.status != 200 {
		t.Errorf("demotion %+v", got)
	}
	got = f.run(root, "identity.issue_reset_link", SampleMember, nil)
	link = f.collect(root, got.body["fields"].(map[string]any)["link_ref"].(string))
	v := verifierOf(random(tokenBytes))
	f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute)
	members, _ := f.s.roster(v)
	if err := f.s.identity.Workspace().ChangeRole(v, SampleWorkspace, SampleMember, identity.Administrator, members[SampleMember].Version); err != nil {
		t.Fatal(err)
	}
	f.code(t, "reset of a user who became an administrator", f.redeem(linkToken(link.body["url"].(string)), "third sample password"), 422, CodeLinkInvalid)
}

func TestInvitationRecovery(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	user, tok := f.created(root, "late@example.invalid", "administrator")
	if got := f.run(root, "identity.revoke_invitation", user, nil); got.status != 200 {
		t.Fatalf("revoke %+v", got)
	}
	f.code(t, "revoke twice", f.run(root, "identity.revoke_invitation", user, nil), 422, string(commands.InvalidStage))
	if got := f.redeem(tok, pw); got.status != 204 {
		t.Fatalf("redeem %+v", got)
	}
	if row := f.row(root, user); row["membership"] != nil || row["login"] != "late@example.invalid" {
		t.Errorf("revoked row %+v", row)
	}
	f.code(t, "sign-in without membership", f.signin("late@example.invalid", pw), 401, string(commands.Unauthenticated))
	if got := f.run(root, "identity.invite", user, map[string]string{"role": "member"}); got.status != 200 || got.body["business_state"] != "invited" {
		t.Fatalf("invite %+v", got)
	}
	f.code(t, "invite twice", f.run(root, "identity.invite", user, map[string]string{"role": "member"}), 422, string(commands.InvalidStage))
	if got := f.signin("late@example.invalid", pw); got.status != 200 || got.body["role"] != "member" {
		t.Errorf("sign-in accepts the invitation %+v", got)
	}
	f.code(t, "invite a member", f.run(root, "identity.invite", user, map[string]string{"role": "member"}), 422, string(commands.InvalidStage))
	if got := f.run(root, "identity.suspend_membership", user, nil); got.status != 200 {
		t.Errorf("suspend membership %+v", got)
	}
	f.code(t, "invite a suspended member", f.run(root, "identity.invite", user, map[string]string{"role": "member"}), 422, string(commands.InvalidStage))
	f.code(t, "suspend twice", f.run(root, "identity.suspend_membership", user, nil), 422, string(commands.InvalidStage))
	if got := f.run(root, "identity.reinstate_membership", user, nil); got.status != 200 {
		t.Errorf("reinstate membership %+v", got)
	}
	if got := f.run(root, "identity.remove_membership", user, nil); got.status != 200 {
		t.Errorf("remove membership %+v", got)
	}
	f.code(t, "role without membership", f.run(root, "identity.change_role", user, map[string]string{"role": "member"}), 422, string(commands.InvalidStage))
	f.code(t, "reinstate without membership", f.run(root, "identity.reinstate_membership", user, nil), 422, string(commands.InvalidStage))
	other, _ := f.created(root, "changed@example.invalid", "member")
	if got := f.run(root, "identity.suspend_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatal(got)
	}
	f.run(root, "identity.reinstate_user", SampleAdmin2, nil)
	if got := f.run(root, "identity.issue_set_link", other, nil); got.status != 200 {
		t.Fatal(got)
	}
	f.s.identity.Installation().SuspendUser(identity.Operator{Reference: "t", Reason: "t"}, other, 1)
	f.s.identity.Installation().ReinstateUser(identity.Operator{Reference: "t", Reason: "t"}, other, 2)
	f.code(t, "invite with a pending invitation", f.run(root, "identity.invite", other, map[string]string{"role": "member"}), 422, string(commands.InvalidStage))
}

func TestInviterChangeLosesInvitationAtSignIn(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(Admin2Login, pw)
	f.s.grantInstallation(SampleAdmin2, f.s.clock())
	second := f.as(Admin2Login)
	user, tok := f.created(second, "lost@example.invalid", "member")
	root := f.as(AdminLogin)
	if got := f.run(root, "identity.suspend_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatal(got)
	}
	f.run(root, "identity.reinstate_user", SampleAdmin2, nil)
	if got := f.redeem(tok, pw); got.status != 204 {
		t.Fatal(got)
	}
	if row := f.row(root, user); row["membership"] != nil || row["invitation_expires"] != "" {
		t.Errorf("lost invitation row %+v", row)
	}
	before := f.s.counter
	if _, _, _, ok := f.s.signIn("lost@example.invalid", SampleWorkspace, false, 0); ok || f.s.counter != before {
		t.Error("sign-in without invitation")
	}
	f.run(root, "identity.invite", user, map[string]string{"role": "member"})
	f.s.policy, _ = newPolicy(f)
	before = f.s.counter
	if _, _, _, ok := f.s.signIn("lost@example.invalid", SampleWorkspace, false, 0); ok || f.s.counter != before+1 {
		t.Error("member without policy")
	}
	if row := f.row(root, user); row["membership"] == nil {
		t.Error("acceptance did not persist when the policy refused")
	}
}

func TestDepartAndLifecycle(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	f.run(root, "identity.issue_reset_link", SampleMember, nil)
	if got := f.run(root, "identity.depart_user", SampleMember, nil); got.status != 200 || got.body["business_state"] != "user_departed" {
		t.Fatalf("depart %+v", got)
	}
	row := f.row(root, SampleMember)
	if row["state"] != "departed" || row["login"] != "" || row["membership"] != nil {
		t.Errorf("departed row %+v", row)
	}
	if f.s.passwords.Has(LoginKey(MemberLogin)) || len(f.s.dir.pickups) != 0 {
		t.Error("password or pickup kept")
	}
	if _, ok := f.s.sessions.get(member.verifier(f)); ok {
		t.Error("session kept")
	}
	f.code(t, "recreate a departed login", f.run(root, "identity.create_user", usersTarget, map[string]string{"login": MemberLogin, "email": MemberLogin, "role": "member"}),
		422, string(commands.PrerequisiteFailed))
	f.code(t, "depart twice", f.run(root, "identity.depart_user", SampleMember, nil), 422, string(commands.InvalidStage))
	f.code(t, "set link for departed", f.run(root, "identity.issue_set_link", SampleMember, nil), 422, string(commands.InvalidStage))
	f.code(t, "invite departed", f.run(root, "identity.invite", SampleMember, map[string]string{"role": "member"}), 422, string(commands.InvalidStage))
	f.code(t, "reinstate active", f.run(root, "identity.reinstate_user", SampleAdmin2, nil), 422, string(commands.InvalidStage))
	f.code(t, "unknown user", f.run(root, "identity.suspend_user", "nobody", nil), 403, string(commands.PermissionDenied))
}

func TestCommandEnvelope(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	stale := `{"request_id":"r1","target_id":"` + SampleAdmin2 + `","idempotency_key":"k1","expected_revision":"r1","rationale":"x","body":{}}`
	f.code(t, "stale revision", f.call("identity.suspend_user", stale, root.cookie, root.csrf), 409, string(commands.RevisionConflict))
	f.code(t, "long rationale", f.admin(root, "identity.suspend_user", SampleAdmin2, strings.Repeat("x", maxRationale+1), map[string]string{}), 422,
		string(commands.InvalidRequest))
	f.code(t, "no rationale", f.admin(root, "identity.suspend_user", SampleAdmin2, "", map[string]string{}), 422, string(commands.InvalidRequest))
	body, _ := json.Marshal(map[string]any{"request_id": "r9", "target_id": usersTarget, "idempotency_key": "k9", "expected_revision": f.revision(root),
		"rationale": "x", "body": map[string]string{"login": "replay@example.invalid", "email": "replay@example.invalid", "role": "member"}})
	first := f.call("identity.create_user", string(body), root.cookie, root.csrf)
	again := f.call("identity.create_user", string(body), root.cookie, root.csrf)
	ref := first.body["fields"].(map[string]any)["link_ref"]
	if again.body["fields"].(map[string]any)["link_ref"] != ref || again.status != 200 {
		t.Fatalf("replay %+v", again)
	}
	f.collect(root, ref.(string))
	f.code(t, "replayed ref used", f.collect(root, ref.(string)), 403, string(commands.PermissionDenied))
	if f.s.caller != (identity.Verifier{}) {
		t.Error("caller kept after a call")
	}
	for _, record := range f.s.commands.Journal(0, 1000) {
		raw, _ := json.Marshal(record)
		if strings.Contains(string(raw), "activate#") {
			t.Error("link in the journal")
		}
	}
	for _, d := range f.s.userDefinitions() {
		if !strings.Contains(string(f.mustOpenAPI()), d.ID) {
			t.Errorf("openapi lacks %s", d.ID)
		}
	}
}

func (f world) mustOpenAPI() []byte {
	f.t.Helper()
	got := f.do("GET", "/api/v1/openapi.json", "", nil)
	return []byte(got.raw)
}

func TestPickupLifecycle(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	ref := func() string {
		got := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": fmt.Sprintf("p%d@example.invalid", keys.Load()), "email": "p@example.invalid", "role": "member"})
		return got.body["fields"].(map[string]any)["link_ref"].(string)
	}
	expired := ref()
	f.advance(pickupValidity)
	f.code(t, "expired pickup", f.collect(root, expired), 403, string(commands.PermissionDenied))
	if _, ok := f.s.dir.pickups[expired]; ok {
		t.Error("expired pickup kept")
	}
	kept := ref()
	f.do("POST", "/api/v1/auth/logout", "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
	if _, ok := f.s.dir.pickups[kept]; ok {
		t.Error("pickup kept after sign-out")
	}
}

func TestUsersReadPaging(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	v := verifierOf(random(tokenBytes))
	f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute)
	if err := f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, "a-service", SampleAdmin); err != nil {
		t.Fatal(err)
	}
	got := f.list(root, "?limit=1")
	if got.status != 200 || got.body["revision"] != f.s.Revision() || len(got.body["users"].([]any)) != 1 || got.body["next"] == "" {
		t.Fatalf("first page %+v", got)
	}
	seen := []string{}
	after := ""
	for {
		page := f.list(root, "?limit=1&after="+after)
		for _, u := range page.body["users"].([]any) {
			row := u.(map[string]any)
			seen = append(seen, row["user_id"].(string))
			allowed := map[string]bool{"user_id": true, "login": true, "email": true, "state": true, "epoch": true, "membership": true, "password_set": true,
				"invitation_expires": true, "link_purpose": true, "link_expires": true, "installation_administrator": true}
			for key := range row {
				if !allowed[key] {
					t.Errorf("unexpected key %s", key)
				}
			}
		}
		after = page.body["next"].(string)
		if after == "" {
			break
		}
	}
	if strings.Join(seen, ",") != "sample-admin,sample-admin-2,sample-member" {
		t.Errorf("pages %v", seen)
	}
	if row := f.row(root, SampleAdmin); row["installation_administrator"] != true || row["email"] != AdminLogin {
		t.Errorf("holder row %+v", row)
	}
}

func TestMapIdentityErrors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	member := verifierOf(random(tokenBytes))
	f.setPassword(MemberLogin, pw)
	f.s.identity.Boundary().OpenSession(LoginTriple(MemberLogin), member, time.Minute, time.Minute)
	_, notAdmin := f.s.identity.Workspace().Members(member, SampleWorkspace, "", 10)
	_, session := f.s.identity.Workspace().Members(verifierOf(random(tokenBytes)), SampleWorkspace, "", 10)
	_, unknown := f.s.identity.Installation().Users("", 0)
	for _, c := range []struct {
		err  error
		code commands.Code
	}{{identity.ErrLastAdministrator, commands.PrerequisiteFailed}, {identity.ErrLimit, commands.DependencyUnavailable},
		{identity.ErrUnavailable, commands.DependencyUnavailable}, {identity.ErrInvalid, commands.InvalidRequest}, {identity.ErrConflict, commands.InvalidStage},
		{notAdmin, commands.PermissionDenied}, {session, commands.PermissionDenied}, {unknown, commands.InvalidRequest}} {
		got := mapIdentity(c.err).(*commands.Failure)
		if got.Code != c.code {
			t.Errorf("%v: %s want %s", c.err, got.Code, c.code)
		}
	}
}

func TestSeedNoneActivationGrantsAdministration(t *testing.T) {
	f := setup(t, SeedNone).ready()
	tok := f.links()[""]
	if got := f.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, f.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := f.s.OpenSession(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	got := f.list(actor{cookie, csrf}, "")
	if got.status != 200 || len(got.body["users"].([]any)) != 1 {
		t.Fatalf("read after activation %+v", got)
	}
	if !f.s.dir.holds(EmptyAdmin, f.s.clock()) || !f.s.dir.taken["first@example.invalid"] {
		t.Error("activation directory")
	}
}

func TestConcurrentReadsAndCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	done := make(chan bool)
	for i := 0; i < 4; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				f.list(root, "")
			}
			done <- true
		}()
	}
	for i := 0; i < 5; i++ {
		f.run(root, "identity.suspend_user", SampleAdmin2, nil)
		f.run(root, "identity.reinstate_user", SampleAdmin2, nil)
	}
	for i := 0; i < 4; i++ {
		<-done
	}
	_ = passwords.PurposeSet
}

func (d *directory) userOf(login string) (string, bool) {
	for user, found := range d.people {
		if found.login == login {
			return user, true
		}
	}
	return "", false
}

func TestLinkAndInvitationExpiry(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	first, _ := f.created(root, "first@example.invalid", "member")
	got := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "second@example.invalid", "email": "second@example.invalid", "role": "member"})
	keep := got.body["fields"].(map[string]any)["link_ref"].(string)
	if got := f.run(root, "identity.issue_set_link", first, nil); got.status != 200 {
		t.Fatal(got)
	}
	if _, kept := f.s.dir.pickups[keep]; !kept {
		t.Error("another user's pickup dropped")
	}
	if row := f.row(root, first); row["link_purpose"] != "set" {
		t.Errorf("set link purpose %+v", row)
	}
	missing := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "norole@example.invalid", "email": "norole@example.invalid"})
	f.code(t, "missing role", missing, 422, string(commands.InvalidRequest))
	f.s.identity.Installation().CreateUser(seedOperator, "outsider", LoginTriple("outsider@example.invalid"), "outsider@example.invalid")
	f.code(t, "set link outside the directory", f.run(root, "identity.issue_set_link", "outsider", nil), 422, string(commands.InvalidStage))
	f.code(t, "reset link outside the directory", f.run(root, "identity.issue_reset_link", "outsider", nil), 422, string(commands.InvalidStage))
	f.advance(invitationValidity + time.Minute)
	root = f.as(AdminLogin)
	f.s.dir.holders[SampleAdmin] = f.s.clock().Add(time.Hour)
	row := f.list(root, "")
	for _, u := range row.body["users"].([]any) {
		r := u.(map[string]any)
		if r["user_id"] == first && (r["invitation_expires"] != "" || r["link_purpose"] != "") {
			t.Errorf("expired entries shown %+v", r)
		}
	}
}
