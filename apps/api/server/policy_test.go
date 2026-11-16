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
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
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

func (f world) propose(a actor, maxAge, step string) reply {
	f.t.Helper()
	return f.run(a, "policy.propose", policyTarget, map[string]string{"max_age_minutes": maxAge, "step_up_minutes": step})
}

func (f world) proposals(a actor) []map[string]any {
	f.t.Helper()
	out := []map[string]any{}
	for _, p := range f.policyRead(a).body["proposals"].([]any) {
		out = append(out, p.(map[string]any))
	}
	return out
}

func (f world) applyPolicy(maxAge, step string) {
	f.t.Helper()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	got := f.propose(root, maxAge, step)
	if got.status != 200 {
		f.t.Fatalf("propose %+v", got)
	}
	target := got.body["fields"].(map[string]any)["proposal"].(string)
	if got := f.run(second, "policy.approve", target, nil); got.status != 200 {
		f.t.Fatalf("approve %+v", got)
	}
}

func TestPolicyReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.policyRead(root)
	if got.status != 200 || got.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("read %+v", got)
	}
	policy := got.body["policy"].(map[string]any)
	if fmt.Sprint(policy["number"]) != fmt.Sprint(1) || policy["proposer"] != SampleAdmin || policy["approver"] != SampleAdmin2 || policy["mfa"] != "none" ||
		fmt.Sprint(policy["max_age_seconds"]) != fmt.Sprint(12*3600) || len(policy["step_up"].([]any)) != 0 {
		t.Errorf("policy %+v", policy)
	}
	list := f.proposals(root)
	if len(list) != 1 || list[0]["state"] != "applied" || list[0]["reason"] != "Sample data" || list[0]["target"] != "proposal-1" {
		t.Errorf("proposals %+v", list)
	}
	if got.body["you"] != SampleAdmin || got.body["now"] != stamp(start) || got.body["signed_in_at"] != stamp(start) ||
		got.body["act_until"] != stamp(start.Add(12*time.Hour)) || got.body["revision"] != f.s.Revision() {
		t.Errorf("page %+v", got.body)
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
	if empty.status != 200 || empty.body["policy"] != nil || empty.body["act_until"] != "" {
		t.Errorf("no policy %+v", empty.body)
	}
}

func TestProposeApproveWithdraw(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	first := f.admin(root, "policy.propose", policyTarget, "Shorter sign-in", map[string]string{"max_age_minutes": "60", "step_up_minutes": "10"})
	if first.status != 200 || first.body["business_state"] != "proposed" {
		t.Fatalf("propose %+v", first)
	}
	a := first.body["fields"].(map[string]any)["proposal"].(string)
	b := f.propose(root, "120", "0").body["fields"].(map[string]any)["proposal"].(string)
	mine := f.proposals(root)
	if len(mine) != 3 || mine[0]["target"] != b || mine[1]["target"] != a || mine[1]["reason"] != "Shorter sign-in" || mine[0]["can_withdraw"] != true || mine[0]["can_approve"] != false {
		t.Errorf("proposer view %+v", mine)
	}
	if mine[1]["act_until"] != stamp(start.Add(10*time.Minute)) {
		t.Errorf("act until %+v", mine[1]["act_until"])
	}
	theirs := f.proposals(second)
	if theirs[0]["can_approve"] != true || theirs[0]["can_withdraw"] != false {
		t.Errorf("other view %+v", theirs[0])
	}
	f.code(t, "self approval", f.run(root, "policy.approve", a, nil), 422, string(commands.PrerequisiteFailed))
	f.code(t, "withdraw by another", f.run(second, "policy.withdraw", a, nil), 422, string(commands.PrerequisiteFailed))
	before := f.s.counter
	if got := f.run(second, "policy.approve", a, nil); got.status != 200 || got.body["business_state"] != "applied" || f.s.counter <= before {
		t.Fatalf("approve %+v", got)
	}
	page := f.policyRead(root).body
	policy := page["policy"].(map[string]any)
	if fmt.Sprint(policy["number"]) != fmt.Sprint(2) || fmt.Sprint(policy["max_age_seconds"]) != fmt.Sprint(3600) || fmt.Sprint(policy["step_up"].([]any)[0].(map[string]any)["max_age_seconds"]) != fmt.Sprint(600) {
		t.Errorf("applied policy %+v", policy)
	}
	for _, p := range f.proposals(root) {
		if p["target"] == b {
			t.Error("discarded proposal still listed")
		}
	}
	f.code(t, "approve a discarded proposal", f.run(second, "policy.approve", b, nil), 403, string(commands.PermissionDenied))
	f.code(t, "approve an applied proposal", f.run(root, "policy.approve", a, nil), 422, string(commands.InvalidStage))
	f.code(t, "approve an applied proposal again by its approver", f.run(second, "policy.approve", a, nil), 422, string(commands.InvalidStage))
	dup := f.propose(root, "200", "0")
	if dup.status != 200 {
		t.Fatal(dup.raw)
	}
	f.code(t, "an identical proposal by the same proposer", f.propose(root, "200", "0"), 422, string(commands.InvalidStage))
	c := f.propose(root, "90", "10").body["fields"].(map[string]any)["proposal"].(string)
	if got := f.run(root, "policy.withdraw", c, nil); got.status != 200 || got.body["business_state"] != "withdrawn" {
		t.Errorf("withdraw %+v", got)
	}
	wide := f.propose(root, "100", "0").body["fields"].(map[string]any)["proposal"].(string)
	for _, p := range f.proposals(root) {
		if p["target"] == wide && p["act_until"] != stamp(start.Add(10*time.Minute)) {
			t.Errorf("a proposal is limited by the current policy window %+v", p)
		}
	}
	if f.s.recorded(mustNumber(t, c)) {
		t.Error("withdrawn record kept after the next change")
	}
}

func mustNumber(t testing.TB, target string) uint64 {
	t.Helper()
	n := proposalNumber(target)
	if n == 0 {
		t.Fatalf("target %s", target)
	}
	return n
}

func TestProposeValidation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, c := range []struct{ age, step, field string }{
		{"-1", "0", "max_age_minutes"}, {"0", "0", "max_age_minutes"}, {"4", "0", "max_age_minutes"}, {"721", "0", "max_age_minutes"},
		{"99999999999999999999", "0", "max_age_minutes"}, {"60", "4", "step_up_minutes"}, {"60", "61", "step_up_minutes"}, {"60", "-5", "step_up_minutes"},
	} {
		got := f.propose(root, c.age, c.step)
		if got.status != 422 || !strings.Contains(got.raw, c.field) {
			t.Errorf("%s/%s: %+v", c.age, c.step, got.raw)
		}
	}
	for _, c := range [][2]string{{"5", "0"}, {"720", "720"}, {"60", "60"}, {"60", "5"}} {
		if got := f.propose(root, c[0], c[1]); got.status != 200 {
			t.Errorf("%v: %+v", c, got.raw)
		}
	}
	for _, raw := range []string{"007", "1a", "", "12345"} {
		if _, ok := minutes(raw, 0, 720); ok {
			t.Errorf("minutes %q", raw)
		}
	}
	for _, target := range []string{"1", "12", "proposal-", "proposal-01", "proposal-x", "proposal-99", "policy-1", "proposal-99999999999999999999"} {
		f.code(t, target, f.run(root, "policy.withdraw", target, nil), 403, string(commands.PermissionDenied))
	}
	stray, err := f.s.policy.Propose(authpolicy.Change{Kind: authpolicy.ChangeRevision, Revision: authpolicy.Revision{Workspace: SampleWorkspace, Methods: []string{authpolicy.MethodPassword},
		MFA: authpolicy.MFANone, MaxAge: time.Hour}}, authpolicy.Principal{User: SampleAdmin, Workspace: SampleWorkspace, Role: authpolicy.RoleAdministrator, Kind: authpolicy.KindHuman},
		authpolicy.Authentication{Method: authpolicy.MethodPassword, At: f.s.clock()})
	if err != nil {
		t.Fatal(err)
	}
	f.code(t, "a proposal apps/api did not record", f.run(root, "policy.withdraw", fmt.Sprintf("proposal-%d", stray.Number), nil), 403, string(commands.PermissionDenied))
	f.code(t, "propose with a wrong target", f.run(root, "policy.propose", "elsewhere", map[string]string{"max_age_minutes": "60", "step_up_minutes": "0"}), 403, string(commands.PermissionDenied))
	f.code(t, "long rationale", f.admin(root, "policy.propose", policyTarget, strings.Repeat("x", maxRationale+1), map[string]string{"max_age_minutes": "60", "step_up_minutes": "0"}),
		422, string(commands.InvalidRequest))
	if _, res := (policyCommand{s: f.s, kind: "propose"}).Resolve("elsewhere", policyTarget); res != commands.NotFound {
		t.Error("other workspace")
	}
}

func TestStepUpAndApproverLockout(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.applyPolicy("60", "10")
	root := f.as(AdminLogin)
	f.advance(11 * time.Minute)
	f.code(t, "step-up required", f.propose(root, "60", "10"), 422, string(commands.PrerequisiteFailed))
	page := f.policyRead(root).body
	if page["now"].(string) <= page["act_until"].(string) {
		t.Errorf("act until not passed %+v", page)
	}
	root = f.as(AdminLogin)
	got := f.propose(root, "5", "0")
	if got.status != 200 {
		t.Fatalf("fresh sign-in %+v", got)
	}
	target := got.body["fields"].(map[string]any)["proposal"].(string)
	second := f.as(Admin2Login)
	f.advance(6 * time.Minute)
	f.code(t, "approver locked out", f.run(second, "policy.approve", target, nil), 422, string(commands.PrerequisiteFailed))
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
	f.s.sessions.byID[v].created = f.s.clock().Add(time.Hour)
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

func TestPolicyActorAndErrors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if _, err := (policyCommand{s: f.s, kind: "propose"}).Apply(commands.Context{Fields: map[string]string{"max_age_minutes": "60", "step_up_minutes": "0"}}); err.(*commands.Failure).Code != commands.PermissionDenied {
		t.Errorf("missing session: %v", err)
	}
	f.s.caller = f.as(AdminLogin).verifier(f)
	f.s.identity.Boundary().EndSession(f.s.caller)
	if _, _, ok := f.s.actorOf(); ok {
		t.Error("ended identity session")
	}
	kept := f.as(AdminLogin).verifier(f)
	f.s.sessions.drop(kept)
	f.s.caller = kept
	if _, err := (policyCommand{s: f.s, kind: "propose"}).Apply(commands.Context{Fields: map[string]string{"max_age_minutes": "60", "step_up_minutes": "0"}}); err.(*commands.Failure).Code != commands.PermissionDenied {
		t.Errorf("identity session without a table entry: %v", err)
	}
	f.s.caller = identity.Verifier{}
	other := authpolicy.Revision{MaxAge: time.Hour, StepUp: []authpolicy.StepUp{{Action: authpolicy.ActionPolicyChange, MaxAge: 30 * time.Minute}, {Action: "zz.other", MaxAge: time.Minute}}}
	if window(other) != 30*time.Minute {
		t.Errorf("window %v", window(other))
	}
	for _, c := range []struct {
		err  error
		code commands.Code
	}{{authpolicy.Refusal{Reason: authpolicy.ReasonNotPending}, commands.InvalidStage}, {authpolicy.ErrNotFound, commands.InvalidStage},
		{authpolicy.Refusal{Reason: authpolicy.ReasonSelfApproval}, commands.PrerequisiteFailed}, {authpolicy.ErrLimit, commands.DependencyUnavailable},
		{authpolicy.ErrInvalid, commands.InvalidRequest}} {
		if got := mapPolicy(c.err).(*commands.Failure).Code; got != c.code {
			t.Errorf("%v: %s", c.err, got)
		}
	}
	f.s.records = nil
	for i := 0; i < maxRecords+50; i++ {
		f.s.records = append(f.s.records, record{number: 1, reason: fmt.Sprint(i)})
	}
	f.s.pruneRecords()
	if len(f.s.records) != maxRecords || f.s.records[0].reason != "50" {
		t.Errorf("cap %d %s", len(f.s.records), f.s.records[0].reason)
	}
	f.s.recordProposal(1, "again")
	if len(f.s.records) != maxRecords {
		t.Error("duplicate record")
	}
	page := f.policyRead(f.as(AdminLogin))
	if len(page.body["proposals"].([]any)) != appliedShown {
		t.Errorf("applied shown %d", len(page.body["proposals"].([]any)))
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
	if !strings.Contains(string(raw), `"required":["max_age_minutes","step_up_minutes"]`) || !strings.Contains(string(raw), "policy.approve") {
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
	f.s.records = append(f.s.records, record{number: 999, reason: "dead"})
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
		body := fmt.Sprintf(`{"request_id":"c%d","target_id":"policy","idempotency_key":"c%d","expected_revision":"%s","rationale":"x","body":{"max_age_minutes":"%d","step_up_minutes":"0"}}`, i, i, f.s.Revision(), 60+i)
		f.call("policy.propose", body, root.cookie, root.csrf)
	}
	wg.Wait()
	if f.s.recorded(999) {
		t.Error("dead record kept after commands")
	}
	f.s.records = append(f.s.records, record{number: 998, reason: "dead"})
	for _, p := range f.proposals(root) {
		if p["number"] == nil || fmt.Sprint(p["number"]) == "0" || fmt.Sprint(p["number"]) == "998" {
			t.Errorf("dead record listed %+v", p)
		}
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
