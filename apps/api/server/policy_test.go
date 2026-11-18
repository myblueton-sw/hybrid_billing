package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (f world) policyRead(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/policy", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) publishDirect(rev authpolicy.Revision) {
	f.t.Helper()
	base, _ := f.s.trustBase(SampleWorkspace)
	rev.Workspace = SampleWorkspace
	if len(rev.Methods) == 0 {
		rev.Methods, rev.MFA = []string{authpolicy.MethodPassword}, authpolicy.MFANone
	}
	next, _ := nextEpoch(base.Epoch, f.s.generation())
	if _, err := f.s.policy.Publish(authpolicy.Publication{Revision: rev, Epoch: next, Number: base.Policy + 1, Proposer: SampleAdmin, Validator: SampleAdmin2,
		Approver: SampleAdmin2, Activator: SampleAdmin3}); err != nil {
		f.t.Fatal(err)
	}
}

func (f world) applyPolicy(maxAge, step string) {
	f.t.Helper()
	c, err := proposedCandidate(map[string]string{"max_age_minutes": maxAge, "step_up_minutes": step})
	if err != nil {
		f.t.Fatal(err)
	}
	f.publishDirect(revisionFor(SampleWorkspace, c))
}

func TestPolicyReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.policyRead(root)
	if got.status != 200 || got.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("read %+v", got)
	}
	policy := got.body["policy"].(map[string]any)
	epoch := policy["epoch"].(map[string]any)
	if fmt.Sprint(policy["number"]) != "1" || policy["proposer"] != "" || policy["mfa"] != "none" || fmt.Sprint(policy["max_age_seconds"]) != fmt.Sprint(12*3600) ||
		len(policy["step_up"].([]any)) != 0 || epoch["value"] != "1" || fmt.Sprint(epoch["generation"]) != "0" || fmt.Sprint(epoch["number"]) != "1" {
		t.Errorf("policy %+v", policy)
	}
	trust := got.body["trust"].(map[string]any)
	if trust["epoch"] != "1" || trust["origin"] != originSeed || trust["hold"] != nil || fmt.Sprint(trust["policy_number"]) != "1" || len(trust["digest"].(string)) != 64 {
		t.Errorf("trust %+v", trust)
	}
	if got.body["you"] != SampleAdmin || got.body["now"] != stamp(start) || got.body["signed_in_at"] != stamp(start) || got.body["started_at"] != stamp(start) ||
		got.body["act_until"] != stamp(start.Add(stepUpDefault)) || got.body["authenticated_at"] != stamp(start) || fmt.Sprint(got.body["step_up_seconds"]) != "900" ||
		got.body["revision"] != f.s.Revision() || len(got.body["changes"].([]any)) != 0 {
		t.Errorf("page %+v", got.body)
	}
	if _, old := got.body["proposals"]; old {
		t.Error("the proposals list is still served")
	}
	rights := got.body["rights"].(map[string]any)
	holders := got.body["holders"].(map[string]any)
	ledger := got.body["ledger"].(map[string]any)
	if rights["propose"] != true || rights["activate"] != true || fmt.Sprint(holders["qualifying_people"]) != "0" || fmt.Sprint(holders["needed"]) != "3" || fmt.Sprint(holders["members"]) != "3" ||
		holders["enrolment"] != "not_decided" || ledger["state"] != "ok" || fmt.Sprint(ledger["generation"]) != "0" {
		t.Errorf("rights %+v holders %+v ledger %+v", rights, holders, ledger)
	}
	f.code(t, "no session", f.policyRead(actor{}), 401, string(commands.Unauthenticated))
	f.code(t, "bad csrf", f.policyRead(actor{root.cookie, "x"}), 403, CodeCSRF)
	f.setPassword(MemberLogin, pw)
	f.code(t, "member", f.policyRead(f.as(MemberLogin)), 403, string(commands.PermissionDenied))
	none := setup(t, SeedNone).ready()
	tok := none.links()[""]
	none.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, none.pre())
	cookie, csrf, _ := none.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	empty := none.policyRead(actor{cookie, csrf})
	if empty.status != 200 || empty.body["policy"] != nil || empty.body["act_until"] != "" || empty.body["trust"].(map[string]any)["epoch"] != "0" {
		t.Errorf("no policy %+v", empty.body)
	}
}

func TestProposeValidation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, c := range []struct{ age, step, field string }{
		{"-1", "0", "max_age_minutes"}, {"0", "0", "max_age_minutes"}, {"4", "0", "max_age_minutes"}, {"721", "0", "max_age_minutes"},
		{"99999999999999999999", "0", "max_age_minutes"}, {"60", "4", "step_up_minutes"}, {"60", "61", "step_up_minutes"}, {"60", "-5", "step_up_minutes"},
	} {
		got := f.trustPropose(root, c.age, c.step)
		if got.status != 422 || !strings.Contains(got.raw, c.field) {
			t.Errorf("%s/%s: %+v", c.age, c.step, got.raw)
		}
	}
	for _, c := range [][2]string{{"5", "0"}, {"719", "719"}, {"60", "60"}, {"60", "5"}} {
		if got := f.trustPropose(root, c[0], c[1]); got.status != 200 {
			t.Errorf("%v: %+v", c, got.raw)
		}
	}
	f.code(t, "the current configuration", f.trustPropose(root, "720", "0"), 422, string(commands.InvalidRequest))
	for _, raw := range []string{"007", "1a", "", "12345"} {
		if _, ok := minutes(raw, 0, 720); ok {
			t.Errorf("minutes %q", raw)
		}
	}
	for _, target := range []string{"1", "trust-", "trust-x-1", "proposal-1", "trust-00000000-99", "trust-99999999999999999999"} {
		f.code(t, target, f.run(root, "trust.cancel", target, map[string]string{"expected_change_revision": "1"}), 403, string(commands.PermissionDenied))
	}
	f.code(t, "propose with a wrong target", f.run(root, "trust.propose", "elsewhere", f.proposeFields("60", "0")), 403, string(commands.PermissionDenied))
	f.code(t, "long rationale", f.admin(root, "trust.propose", trustTarget, strings.Repeat("x", maxRationale+1), f.proposeFields("60", "0")), 422, string(commands.InvalidRequest))
	if got := f.admin(root, "trust.propose", trustTarget, strings.Repeat("x", maxRationale), f.proposeFields("61", "0")); got.status != 200 {
		t.Errorf("a rationale at the limit %s", got.raw)
	}
	if _, res := (trustCommand{s: f.s, kind: "propose"}).Resolve("elsewhere", trustTarget); res != commands.NotFound {
		t.Error("other workspace")
	}
	for _, id := range []string{"policy.propose", "policy.approve", "policy.withdraw"} {
		f.code(t, id, f.run(root, id, "policy", nil), 404, string(commands.UnsupportedCommand))
	}
}

func TestMaximumAgeEndsSessionsOnEveryPath(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.applyPolicy("5", "0")
	root := f.as(AdminLogin)
	old := []actor{f.as(AdminLogin), f.as(AdminLogin), f.as(AdminLogin), f.as(AdminLogin), f.as(AdminLogin)}
	ref := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "age@example.invalid", "email": "age@example.invalid", "role": "member"}).body["fields"].(map[string]any)["link_ref"].(string)
	f.advance(6 * time.Minute)
	paths := map[string]reply{
		"current": f.do("GET", "/api/v1/session", "", func(r *http.Request) { r.AddCookie(old[0].cookie); r.Header.Set(clientHeader, ConsoleClient) }),
		"reader":  f.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(old[1].cookie) }),
		"command": f.call("identity.suspend_user", `{"request_id":"r","target_id":"`+SampleMember+`","idempotency_key":"k","expected_revision":"r1","rationale":"x","body":{}}`, old[2].cookie, old[2].csrf),
		"users":   f.list(old[3], ""),
		"collect": f.collect(root, ref),
	}
	for name, r := range paths {
		f.code(t, name, r, 401, string(commands.Unauthenticated))
	}
	for i, a := range old[:4] {
		if _, ok := f.s.sessions.get(a.verifier(f)); ok {
			t.Errorf("session %d kept", i)
		}
	}
	if got := f.policyRead(f.as(AdminLogin)); got.status != 200 {
		t.Errorf("fresh session %+v", got)
	}
	future := f.as(AdminLogin)
	v := future.verifier(f)
	f.s.sessions.mu.Lock()
	f.s.sessions.byID[v].authenticated = f.s.clock().Add(time.Hour)
	f.s.sessions.mu.Unlock()
	f.code(t, "future sign-in time", f.policyRead(future), 401, string(commands.Unauthenticated))
}

func TestClockRegressionRefusesSignIn(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.applyPolicy("5", "0")
	f.mu.Lock()
	*f.now = f.now.Add(-10 * time.Minute)
	f.mu.Unlock()
	if _, _, _, ok := f.s.signIn(AdminLogin, SampleWorkspace, false, 0); ok {
		t.Error("sign-in after a clock regression larger than the maximum age")
	}
}

func TestWindow(t *testing.T) {
	other := authpolicy.Revision{MaxAge: time.Hour, StepUp: []authpolicy.StepUp{{Action: authpolicy.ActionPolicyChange, MaxAge: 30 * time.Minute}, {Action: "zz.other", MaxAge: time.Minute}}}
	if window(other) != 30*time.Minute {
		t.Errorf("window %v", window(other))
	}
}

func TestSecondHolderTrustBoundaryIsPinned(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	second := f.as(Admin2Login)
	if got := f.run(second, "identity.suspend_user", SampleAdmin, nil); got.status != 200 {
		t.Errorf("the second holder can suspend the first %+v", got)
	}
	root := f.as(Admin2Login)
	f.code(t, "self", f.run(root, "identity.suspend_user", SampleAdmin2, nil), 422, string(commands.PrerequisiteFailed))
	doc := map[string]any{}
	_ = json.Unmarshal(f.mustOpenAPI(), &doc)
	raw, _ := json.Marshal(doc)
	if !strings.Contains(string(raw), `"required":["expected_epoch","expected_policy","max_age_minutes","step_up_minutes"]`) || !strings.Contains(string(raw), "trust.activate") ||
		strings.Contains(string(raw), "policy.approve") {
		t.Error("openapi")
	}
}

func TestOnlySignInOpensSessions(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(".", name))
		calls := strings.Count(string(data), "s.openSession(")
		if (name == "signin.go" && calls != 1) || (name == "session.go" && calls != 1) || (name != "signin.go" && name != "session.go" && calls != 0) {
			t.Errorf("%s opens sessions %d times", name, calls)
		}
	}
}

func TestConcurrentPolicyReadsAndCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				f.policyRead(root)
			}
		}()
	}
	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"request_id":"c%d","target_id":"trust","idempotency_key":"c%d","expected_revision":"%s","rationale":"x","body":{"max_age_minutes":"%d","step_up_minutes":"0","expected_epoch":"1","expected_policy":"1"}}`, i, i, f.s.Revision(), 60+i)
		f.call("trust.propose", body, root.cookie, root.csrf)
	}
	wg.Wait()
	if n := len(f.policyRead(root).body["changes"].([]any)); n != 3 {
		t.Errorf("changes %d", n)
	}
}

func TestTakeoverRouteRemainsDeclared(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(Admin2Login, pw)
	root := f.as(AdminLogin)
	if got := f.run(root, "identity.change_role", SampleAdmin2, map[string]string{"role": "member"}); got.status != 200 {
		t.Fatalf("demote %+v", got)
	}
	got := f.run(root, "identity.issue_reset_link", SampleAdmin2, nil)
	link := f.collect(root, got.body["fields"].(map[string]any)["link_ref"].(string))
	if r := f.redeem(linkToken(link.body["url"].(string)), "a password chosen by the other holder"); r.status != 204 {
		t.Fatalf("redeem %+v", r)
	}
	if got := f.run(root, "identity.change_role", SampleAdmin2, map[string]string{"role": "administrator"}); got.status != 200 {
		t.Fatalf("promote %+v", got)
	}
	cookie, csrf, _ := f.s.openFor(LoginTriple(Admin2Login), SampleWorkspace, false)
	if got := f.policyRead(actor{cookie, csrf}); got.status != 200 {
		t.Errorf("the declared takeover route reaches the second holder's policy access %+v", got)
	}
}

func TestConcurrentStaleRequests(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.applyPolicy("5", "0")
	olds := []actor{}
	for i := 0; i < 10; i++ {
		olds = append(olds, f.as(AdminLogin))
	}
	f.advance(6 * time.Minute)
	fresh := f.as(AdminLogin)
	var wg sync.WaitGroup
	for i, a := range olds {
		wg.Add(1)
		go func(i int, a actor) {
			defer wg.Done()
			switch i % 5 {
			case 0:
				f.do("GET", "/api/v1/session", "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(clientHeader, ConsoleClient) })
			case 1:
				f.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(a.cookie) })
			case 2:
				f.call("identity.suspend_user", `{"request_id":"r","target_id":"x","idempotency_key":"k","expected_revision":"r1","rationale":"x","body":{}}`, a.cookie, a.csrf)
			case 3:
				f.list(a, "")
			default:
				f.collect(a, "missing")
			}
		}(i, a)
	}
	wg.Add(1)
	go func() { defer wg.Done(); f.policyRead(fresh) }()
	wg.Wait()
	for _, a := range olds {
		if _, ok := f.s.sessions.get(a.verifier(f)); ok {
			t.Error("stale session kept")
		}
	}
}
