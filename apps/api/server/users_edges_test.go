package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

func setupWith(t testing.TB, change func(*Config)) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	cfg := Config{Data: t.TempDir() + "/data", Seed: SeedSample, Passwords: fastPasswords(), Identity: IdentityLimits(), Clock: func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}}
	change(&cfg)
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		t.Fatal(err)
	}
	s.manual = true
	value := 0
	return world{t: t, s: s, now: &now, mu: mu, value: &value}.ready()
}

var seedOperator = identity.Operator{Reference: "test", Reason: "test"}

func TestIdentityUnavailablePaths(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	revision := f.revision(root)
	f.s.fault = func(at string) bool { return at == "users" }
	h := userCommand{s: f.s}
	if _, res := h.Resolve(SampleWorkspace, SampleMember); res != commands.Unavailable {
		t.Errorf("resolve %v", res)
	}
	if _, res := h.Resolve("elsewhere", SampleMember); res != commands.NotFound {
		t.Errorf("other workspace %v", res)
	}
	f.s.caller = root.verifier(f)
	if _, err := h.Apply(commands.Context{Target: SampleMember}); err.(*commands.Failure).Code != commands.DependencyUnavailable {
		t.Errorf("apply %v", err)
	}
	f.s.caller = identity.Verifier{}
	f.code(t, "read", f.list(root, ""), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	if _, err := f.s.roster(verifierOf(random(tokenBytes))); err == nil {
		t.Error("roster for a stale actor")
	}
	if _, err := f.s.usersPage(verifierOf(random(tokenBytes)), "", 10); err == nil {
		t.Error("page for a stale actor")
	}
	if err := f.s.registerUsers(); err == nil {
		t.Error("registered twice")
	}
	_ = revision
}

func TestManyUsersPage(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	admin := root.verifier(f)
	now := f.s.clock()
	for i := 0; i < 105; i++ {
		id := fmt.Sprintf("bulk-%03d", i)
		login := id + "@example.invalid"
		f.s.identity.Installation().CreateUser(seedOperator, id, LoginTriple(login), login)
		f.s.identity.Workspace().Invite(admin, SampleWorkspace, "inv-"+id, identity.Member, verifierOf([]byte(id)), identity.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}, id)
		v := verifierOf(random(tokenBytes))
		f.s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute)
		if err := f.s.identity.Workspace().Accept(SampleWorkspace, "inv-"+id, verifierOf([]byte(id)), v); err != nil {
			t.Fatal(err)
		}
		f.s.identity.Boundary().EndSession(v)
	}
	got := f.list(root, "?limit=100")
	if len(got.body["users"].([]any)) != 100 || got.body["next"] == "" {
		t.Fatalf("first page %d %v", len(got.body["users"].([]any)), got.body["next"])
	}
	rest := f.list(root, "?limit=100&after="+got.body["next"].(string))
	if len(rest.body["users"].([]any)) != 9 || rest.body["next"] != "" {
		t.Errorf("second page %d", len(rest.body["users"].([]any)))
	}
	if row := f.row(root, "sample-member"); row != nil {
		if _, ok, _ := f.s.findUser("sample-member"); !ok {
			t.Error("find beyond the first page")
		}
	}
	if got := f.run(root, "identity.change_role", "sample-member", map[string]string{"role": "administrator"}); got.status != 200 {
		t.Errorf("role change for a member beyond the first page %+v", got)
	}
	if got := f.run(root, "identity.suspend_user", "sample-member", nil); got.status != 200 {
		t.Errorf("command on a user beyond the first page %+v", got)
	}
}

func TestLimitsFromIdentityAndPasswords(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Identity.PendingInvitations = 2; c.Passwords.Links = 5 })
	root := f.as(AdminLogin)
	user, _ := f.created(root, "one@example.invalid", "member")
	f.run(root, "identity.revoke_invitation", user, nil)
	other := "lonely"
	f.s.identity.Installation().CreateUser(seedOperator, other, LoginTriple("lonely@example.invalid"), "lonely@example.invalid")
	f.s.dir.add(other, "lonely@example.invalid", "lonely@example.invalid")
	v := verifierOf(random(tokenBytes))
	f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute)
	f.s.identity.Workspace().Invite(v, SampleWorkspace, "fill", identity.Member, v, identity.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}, SampleAdmin2)
	f.code(t, "identity invitation limit", f.run(root, "identity.invite", other, map[string]string{"role": "member"}), 503, string(commands.DependencyUnavailable))
	f.setPassword(MemberLogin, pw)
	for i := 0; i < 10; i++ {
		f.s.passwords.Issue(LoginKey(fmt.Sprintf("x%d@example.invalid", i)), passwords.PurposeSet, time.Hour)
	}
	f.s.dir.pickups["unrelated"] = pickup{target: "nobody", expires: f.s.clock().Add(time.Minute)}
	f.s.dir.pickups["mine"] = pickup{target: other, expires: f.s.clock().Add(time.Minute)}
	f.code(t, "set link limit", f.run(root, "identity.issue_set_link", other, nil), 500, string(commands.InternalError))
	_, unrelated := f.s.dir.pickups["unrelated"]
	_, mine := f.s.dir.pickups["mine"]
	if !unrelated || mine {
		t.Errorf("pickups after a failed reissue: unrelated %v target %v", unrelated, mine)
	}
	f.code(t, "reset link limit", f.run(root, "identity.issue_reset_link", SampleMember, nil), 500, string(commands.InternalError))
	if _, live := f.s.dir.link(MemberLogin, f.s.clock()); live {
		t.Error("a failed reissue left a live link record")
	}
	if err := f.s.inviteUser(verifierOf(random(tokenBytes)), "x", other, identity.Member, f.s.clock(), time.Time{}); err == nil {
		t.Error("invite with a stale actor")
	}
}

func TestIdentityStateRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(Admin2Login, pw)
	root := f.as(AdminLogin)
	f.s.identity.Installation().CreateUser(seedOperator, "ghost", LoginTriple("ghost@example.invalid"), "ghost@example.invalid")
	f.code(t, "login linked outside the directory", f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "ghost@example.invalid", "email": "g@example.invalid", "role": "member"}),
		422, string(commands.PrerequisiteFailed))
	f.s.identity.Installation().DepartUser(seedOperator, "ghost", 1)
	f.code(t, "departed login outside the directory", f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "ghost@example.invalid", "email": "g@example.invalid", "role": "member"}),
		422, string(commands.InvalidStage))
	f.run(root, "identity.suspend_user", SampleAdmin2, nil)
	f.code(t, "suspend twice", f.run(root, "identity.suspend_user", SampleAdmin2, nil), 422, string(commands.InvalidStage))
	f.code(t, "same role", f.run(root, "identity.change_role", SampleAdmin2, map[string]string{"role": "administrator"}), 422, string(commands.InvalidStage))
	user, _ := f.created(root, "revoked@example.invalid", "member")
	inv := f.s.dir.invites["revoked@example.invalid"]
	f.s.identity.Workspace().RevokeInvitation(root.verifier(f), SampleWorkspace, inv.id)
	f.code(t, "invitation revoked elsewhere", f.run(root, "identity.revoke_invitation", user, nil), 422, string(commands.InvalidStage))
}

func TestResetEdges(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.identity.Installation().CreateUser(seedOperator, "outsider", LoginTriple("outsider@example.invalid"), "outsider@example.invalid")
	rec, _ := f.s.passwords.Hash(LoginKey("outsider@example.invalid"), pw)
	f.s.passwords.Put(LoginKey("outsider@example.invalid"), rec)
	tok, _ := f.s.passwords.Issue(LoginKey("outsider@example.invalid"), passwords.PurposeReset, time.Hour)
	f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "left@example.invalid", "email": "left@example.invalid", "role": "member"})
	if got := f.redeem(tok, "a new sample password"); got.status != 204 {
		t.Errorf("reset outside the directory %+v", got)
	}
	f.setPassword(MemberLogin, pw)
	got := f.run(root, "identity.issue_reset_link", SampleMember, nil)
	link := f.collect(root, got.body["fields"].(map[string]any)["link_ref"].(string))
	for i := 0; i < IdentityLimits().SessionsPerUser; i++ {
		f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
	}
	if got := f.redeem(linkToken(link.body["url"].(string)), "another sample password"); got.status != 204 {
		t.Errorf("reset while sessions are at the limit %+v", got)
	}
	if len(f.s.sessions.byLogin(MemberLogin)) != 0 {
		t.Error("sessions kept after reset")
	}
	got = f.run(root, "identity.issue_reset_link", SampleMember, nil)
	link = f.collect(root, got.body["fields"].(map[string]any)["link_ref"].(string))
	f.run(root, "identity.suspend_user", SampleMember, nil)
	f.code(t, "reset link for a suspended user", f.run(root, "identity.issue_reset_link", SampleMember, nil), 422, string(commands.InvalidStage))
	f.code(t, "redeem a reset for a user suspended after issue", f.redeem(linkToken(link.body["url"].(string)), "yet another sample password"), 422, CodeLinkInvalid)
	f.run(root, "identity.reinstate_user", SampleMember, nil)
	for i := 0; i < IdentityLimits().SessionsPerUser; i++ {
		f.s.identity.Boundary().OpenSession(LoginTriple(MemberLogin), verifierOf(random(tokenBytes)), time.Minute, time.Minute)
	}
	f.code(t, "redeem when only sessions outside the table fill the limit", f.redeem(linkToken(link.body["url"].(string)), "yet another sample password"), 422, CodeLinkInvalid)
	f.s.fault = func(at string) bool { return at == "grant" }
	if err := f.s.grantInstallation(SampleAdmin2, f.s.clock()); err == nil {
		t.Error("grant fault")
	}
	f.s.fault = nil
}

func TestReadAdmissionFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.must(f.s.admin.SetAvailable(false))
	f.code(t, "authority unavailable", f.list(root, ""), 503, string(commands.AuthorityUnavailable))
	f.must(f.s.admin.SetAvailable(true))
	f.s.identity.Boundary().EndSession(root.verifier(f))
	f.code(t, "ended identity session", f.list(root, ""), 401, string(commands.Unauthenticated))
	_ = gate.AdministerInstallation
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func TestPruneDuringCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "prune@example.invalid", "email": "prune@example.invalid", "role": "member"})
	f.advance(setLinkValidity + time.Minute)
	root = f.as(AdminLogin)
	f.run(root, "identity.suspend_user", SampleAdmin2, nil)
	if len(f.s.dir.pickups) != 0 || len(f.s.dir.links) != 0 {
		t.Errorf("pruned %d %d", len(f.s.dir.pickups), len(f.s.dir.links))
	}
}

func TestOperatorAndContract(t *testing.T) {
	got := operator(&change{ctx: commands.Context{User: "the-actor", Rationale: "the reason", Fields: map[string]string{"operator": "forged"}}})
	if got != (identity.Operator{Reference: "the-actor", Reason: "the reason"}) {
		t.Errorf("operator %+v", got)
	}
	f := setup(t, SeedSample).ready()
	doc := map[string]any{}
	if err := json.Unmarshal(f.mustOpenAPI(), &doc); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(doc)
	if !strings.Contains(string(raw), `"required":["email","login","role"]`) {
		t.Error("create_user required fields")
	}
	f.setPassword(AdminLogin, pw)
	root := f.as(AdminLogin)
	f.code(t, "self promotion", f.run(root, "identity.change_role", SampleAdmin, map[string]string{"role": "administrator"}), 422, string(commands.PrerequisiteFailed))
	f.setPassword(Admin2Login, pw)
	if got := f.run(root, "identity.depart_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatal(got)
	}
	if f.s.dir.holds(SampleAdmin2, f.s.clock()) {
		t.Error("departed holder kept")
	}
}

func TestActivationGrantFailureIsLogged(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Seed = SeedNone; c.Log = slog.New(slog.NewTextHandler(&logs, nil)) })
	f.s.fault = func(at string) bool { return at == "grant" }
	tok := f.links()[""]
	if got := f.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, f.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	if !strings.Contains(logs.String(), "installation grant failed at activation") || f.s.dir.holds(EmptyAdmin, f.s.clock()) {
		t.Errorf("logs %q", logs.String())
	}
}

func TestTokensNeverLeave(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewTextHandler(&logs, nil)) })
	root := f.as(AdminLogin)
	got := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "leak@example.invalid", "email": "leak@example.invalid", "role": "member"})
	link := f.collect(root, got.body["fields"].(map[string]any)["link_ref"].(string))
	tok := linkToken(link.body["url"].(string))
	journal, _ := json.Marshal(f.s.commands.Journal(0, 1000))
	page := f.list(root, "")
	for name, text := range map[string]string{"result": got.raw, "journal": string(journal), "logs": logs.String(), "read": page.raw} {
		if strings.Contains(text, tok) || strings.Contains(text, "activate#") {
			t.Errorf("token in %s", name)
		}
	}
}

func TestConcurrentRedeemAndRevoke(t *testing.T) {
	for round := 0; round < 5; round++ {
		f := setup(t, SeedSample).ready()
		root := f.as(AdminLogin)
		user, tok := f.created(root, fmt.Sprintf("race%d@example.invalid", round), "member")
		pre := f.pre()
		revision := f.revision(root)
		body := func(key string) string {
			return `{"request_id":"r-` + key + `","target_id":"` + user + `","idempotency_key":"` + key + `","expected_revision":"` + revision + `","rationale":"x","body":{}}`
		}
		var wg sync.WaitGroup
		wg.Add(4)
		go func() { defer wg.Done(); f.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`"}`, pre) }()
		go func() { defer wg.Done(); f.call("identity.revoke_invitation", body("revoke"), root.cookie, root.csrf) }()
		go func() { defer wg.Done(); f.call("identity.issue_set_link", body("reissue"), root.cookie, root.csrf) }()
		go func() { defer wg.Done(); f.list(root, "") }()
		wg.Wait()
		row := f.row(root, user)
		if row["password_set"] == true && row["link_purpose"] == "set" {
			t.Errorf("round %d: a live set link after the password was set %+v", round, row)
		}
		if row["membership"] != nil && row["invitation_expires"] != "" {
			t.Errorf("round %d: a member with a pending invitation %+v", round, row)
		}
	}
}

func TestStaleGenerationOpensNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	user, tok := f.created(root, "stale@example.invalid", "member")
	f.redeem(tok, pw)
	f.run(root, "identity.remove_membership", user, nil)
	f.run(root, "identity.invite", user, map[string]string{"role": "member"})
	f.s.dir.generations["stale@example.invalid"]++
	before := len(f.s.sessions.byLogin("stale@example.invalid"))
	if _, _, _, ok := f.s.signIn("stale@example.invalid", SampleWorkspace, false, 0); ok {
		t.Error("stale sign-in")
	}
	if len(f.s.sessions.byLogin("stale@example.invalid")) != before || f.row(root, user)["invitation_expires"] == "" {
		t.Error("stale sign-in opened a session or accepted the invitation")
	}
}
