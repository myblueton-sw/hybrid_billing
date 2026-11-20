package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func (f world) reauth(a actor, password string) reply {
	f.t.Helper()
	return f.selfPost(a, "reauthenticate", `{"password":"`+password+`"}`)
}

func (f world) directPolicy(rev authpolicy.Revision) {
	f.t.Helper()
	f.publishDirect(rev)
}

func refused(t testing.TB, name string, got reply) {
	t.Helper()
	if got.status != 403 || got.body["code"] != CodeReauthentication || got.body["next_action"] != NextReauthenticate || got.body["retry"] != false ||
		got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("%s: %d %s", name, got.status, got.raw)
	}
}

func notRefused(t testing.TB, name string, got reply) {
	t.Helper()
	if got.status == 403 && got.body["code"] == CodeReauthentication || got.status == 0 {
		t.Errorf("%s: %d %s", name, got.status, got.raw)
	}
}

func probeWorld(t *testing.T) (world, actor) {
	f := setup(t, SeedSample).ready()
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.register("peek", reader{f.s}, false)
	f.grant(SampleAdmin)
	f.setPassword(AdminLogin, pw)
	return f, f.signedIn(AdminLogin, pw)
}

func probeCall(f world, a actor, key, request string) reply {
	f.t.Helper()
	var rev string
	f.s.Read(func(r string) { rev = r })
	return f.call("probe", `{"request_id":"`+request+`","target_id":"t1","idempotency_key":"`+key+`","expected_revision":"`+rev+`","body":{"note":"hi"}}`, a.cookie, a.csrf)
}

func TestEveryRegisteredMutatingCommandRequiresStepUp(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	ids := []string{}
	for id, mutating := range f.s.mutating {
		if !mutating {
			if f.s.stepUp(id) {
				t.Errorf("read-only command %s requires mutation step-up", id)
			}
			continue
		}
		if exempt[id] {
			t.Errorf("%s is not covered by step-up", id)
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	mutating := 0
	families := map[string]int{}
	for _, entry := range f.s.commands.Catalogue() {
		if entry.Mutating {
			mutating++
			families[strings.SplitN(entry.ID, ".", 2)[0]]++
		}
	}
	if len(ids) != mutating || len(ids) < 63 || len(exempt) != 0 {
		t.Fatalf("registry %d catalogue %d exempt %d", len(ids), mutating, len(exempt))
	}
	for _, family := range []string{"access", "contract", "financial_authority", "financial_permission", "financial_policy", "financial_request_permission", "identity", "input_authority", "pricing_authority", "pricing_policy", "installation", "integrations", "trust", "recovery", "relationship", "retention", "service"} {
		if families[family] == 0 {
			t.Errorf("family %s has no mutating command", family)
		}
	}
	t.Logf("%d step-up commands %v", len(ids), families)
	call := func(a actor, id string, n int) reply {
		return f.call(id, fmt.Sprintf(`{"request_id":"cover-%d","target_id":"cover-missing","idempotency_key":"cover-%s-%d","expected_revision":"r0","rationale":"x","body":{}}`, n, id, n),
			a.cookie, a.csrf)
	}
	old := f.signedIn(AdminLogin, pw)
	exact := f.signedIn(AdminLogin, pw)
	f.advance(stepUpDefault)
	for _, id := range ids {
		notRefused(t, "exactly at the window "+id, call(exact, id, 1))
	}
	f.advance(time.Nanosecond)
	before, journal := f.s.counter, len(f.s.commands.Journal(0, 1<<20))
	for _, id := range ids {
		refused(t, "past the window "+id, call(old, id, 2))
	}
	if f.s.counter != before || len(f.s.commands.Journal(0, 1<<20)) != journal {
		t.Error("refusals were recorded")
	}
	if got := f.reauth(old, pw); got.status != 204 {
		t.Fatalf("reauthenticate %+v", got)
	}
	for _, id := range ids {
		notRefused(t, "after reauthentication "+id, call(old, id, 3))
	}
	f.register("synthetic.later", &probe{s: f.s, value: f.value}, true)
	f.register("synthetic.read", reader{f.s}, false)
	f.grant(SampleAdmin)
	f.advance(stepUpDefault + time.Nanosecond)
	refused(t, "a mutating command registered later", call(old, "synthetic.later", 4))
	notRefused(t, "a reading command", f.call("synthetic.read", `{"request_id":"r","target_id":"t1","body":{}}`, old.cookie, old.csrf))
	f.code(t, "an unregistered command", call(old, "synthetic.unknown", 5), 404, string(commands.UnsupportedCommand))
	refused(t, "a contract command", call(old, "contract.suspend", 6))
}

func TestPostRoutesAreClassified(t *testing.T) {
	data, err := os.ReadFile("boundary.go")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, m := range regexp.MustCompile(`"(POST|PUT|PATCH|DELETE) (/[^"]+)"`).FindAllStringSubmatch(string(data), -1) {
		found[m[2]] = true
	}
	listed := map[string]bool{}
	for _, r := range postRoutes {
		listed[r.path] = r.stepUp
		if !found[r.path] {
			t.Errorf("listed route %s is not registered", r.path)
		}
	}
	for path := range found {
		if _, ok := listed[path]; !ok {
			t.Errorf("POST route %s is not classified", path)
		}
	}
	want := map[string]bool{"/api/v1/commands/{command}": true, "/api/v1/me/sessions/end": true}
	for path, step := range listed {
		if step != want[path] {
			t.Errorf("route %s classified %v", path, step)
		}
	}
	if len(found) != 8 || len(postRoutes) != 8 {
		t.Errorf("routes %d listed %d", len(found), len(postRoutes))
	}
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.advance(stepUpDefault + time.Nanosecond)
	before := f.s.counter
	got := f.selfPost(a, "sessions/end", `{"session":"others"}`)
	refused(t, "sessions/end", got)
	if got.body["request_id"] != got.header.Get("X-Request-Id") || got.body["request_id"] == "" || f.s.counter != before || len(f.me(a).body["sessions"].([]any)) != 2 {
		t.Errorf("sessions/end refusal %+v", got)
	}
	if f.reauth(a, pw).status != 204 {
		t.Fatal("reauthenticate")
	}
	if got := f.selfPost(a, "sessions/end", f.othersBody(a)); got.status != 202 || fmt.Sprint(got.body["count"]) != "1" {
		t.Errorf("after reauthentication %+v", got)
	}
	if got := f.selfPost(a, "password", pwBody(pw, newPw)); got.status != 204 {
		t.Errorf("password change is not step-up %+v", got)
	}
}

func TestRefusalContract(t *testing.T) {
	f, a := probeWorld(t)
	f.setPassword(MemberLogin, pw)
	member := f.signedIn(MemberLogin, pw)
	f.advance(stepUpDefault + time.Nanosecond)
	body := func(target, request string) string {
		return `{"request_id":"` + request + `","target_id":"` + target + `","idempotency_key":"k1","expected_revision":"r2","body":{"note":"hi"}}`
	}
	valid := f.call("probe", body("t1", "req-echo"), a.cookie, a.csrf)
	refused(t, "valid", valid)
	if valid.body["request_id"] != "req-echo" || valid.header.Get("X-Request-Id") != "req-echo" || len(valid.body) != 4 {
		t.Errorf("valid request id %s", valid.raw)
	}
	for _, bad := range []string{"", "has space", strings.Repeat("a", 129), "ü"} {
		got := f.call("probe", body("t1", bad), a.cookie, a.csrf)
		refused(t, "invalid "+bad, got)
		if got.body["request_id"] != commands.Marker {
			t.Errorf("invalid request id %q %s", bad, got.raw)
		}
	}
	missing := f.call("probe", body("missing", "req-echo"), a.cookie, a.csrf)
	other := f.call("probe", body("t1", "req-echo"), member.cookie, member.csrf)
	if missing.raw != valid.raw || other.raw != valid.raw {
		t.Errorf("refusals differ: %s %s %s", valid.raw, missing.raw, other.raw)
	}
	if strings.Contains(valid.raw, "t1") || strings.Contains(valid.raw, "fields") {
		t.Error("refusal reveals the target")
	}
}

func TestRequestIDParity(t *testing.T) {
	f := setup(t, SeedSample).ready()
	samples := []string{"", "a", "A-z_0.9:x", strings.Repeat("a", 128), strings.Repeat("a", 129), "a b", "a/b", "ü", "a\x00", "a,b", "\"", "a+b"}
	for b := 0; b < 256; b++ {
		samples = append(samples, "x"+string(rune(b)), string([]byte{byte(b)}))
	}
	for _, id := range samples {
		if got := f.s.commands.Execute(commands.Call{Request: id}).Request; got != echoRequest(id) {
			t.Errorf("%q: module %q apps/api %q", id, got, echoRequest(id))
		}
	}
}

func TestRefusalRecordsNothing(t *testing.T) {
	f, a := probeWorld(t)
	f.advance(stepUpDefault + time.Nanosecond)
	before, journal, applied := f.s.counter, len(f.s.commands.Journal(0, 1<<20)), *f.value
	for i := 0; i < 80; i++ {
		refused(t, "refusal", probeCall(f, a, "k1", "req-1"))
	}
	if f.s.counter != before || len(f.s.commands.Journal(0, 1<<20)) != journal || *f.value != applied {
		t.Fatalf("refusals recorded counter %d journal %d", f.s.counter, len(f.s.commands.Journal(0, 1<<20)))
	}
	if f.reauth(a, pw).status != 204 {
		t.Fatal("reauthenticate")
	}
	if f.s.counter != before {
		t.Error("reauthentication bumped the counter")
	}
	got := probeCall(f, a, "k1", "req-1")
	if got.status != 200 || got.body["replay"] != false || f.s.counter != before+1 || *f.value != applied+1 {
		t.Errorf("resend after reauthentication %+v", got)
	}
}

func TestReplayNeedsStepUp(t *testing.T) {
	f, a := probeWorld(t)
	rev := f.s.Revision()
	first := f.call("probe", mutation("k1", rev), a.cookie, a.csrf)
	if first.status != 200 || first.body["replay"] != false {
		t.Fatalf("first %+v", first)
	}
	f.advance(stepUpDefault + time.Nanosecond)
	refused(t, "replay", f.call("probe", mutation("k1", rev), a.cookie, a.csrf))
	if f.reauth(a, pw).status != 204 {
		t.Fatal("reauthenticate")
	}
	if again := f.call("probe", mutation("k1", rev), a.cookie, a.csrf); again.status != 200 || again.body["replay"] != true || *f.value != 1 {
		t.Errorf("replay after reauthentication %+v", again)
	}
	f.call("probe", mutation("k2", f.s.Revision()), a.cookie, a.csrf)
	f.code(t, "stale resend", f.call("probe", mutation("k3", rev), a.cookie, a.csrf), 409, string(commands.RevisionConflict))
}

func TestStepUpWindows(t *testing.T) {
	cases := []struct {
		name   string
		rule   []authpolicy.StepUp
		window time.Duration
	}{
		{"no rule", nil, stepUpDefault},
		{"shorter sensitive rule", []authpolicy.StepUp{{Action: ActionSensitive, MaxAge: 10 * time.Minute}}, 10 * time.Minute},
		{"longer sensitive rule", []authpolicy.StepUp{{Action: ActionSensitive, MaxAge: 40 * time.Minute}}, stepUpDefault},
		{"policy rule only", []authpolicy.StepUp{{Action: authpolicy.ActionPolicyChange, MaxAge: 5 * time.Minute}}, stepUpDefault},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := probeWorld(t)
			f.directPolicy(authpolicy.Revision{MaxAge: time.Hour, StepUp: c.rule})
			a := f.signedIn(AdminLogin, pw)
			f.advance(c.window)
			if got := probeCall(f, a, "k1", "req-1"); got.status != 200 {
				t.Errorf("at the window %+v", got)
			}
			f.advance(time.Nanosecond)
			refused(t, "past the window", probeCall(f, a, "k2", "req-2"))
		})
	}
	for _, c := range []struct {
		step   string
		window time.Duration
	}{{"10", 10 * time.Minute}, {"40", stepUpDefault}, {"0", stepUpDefault}} {
		t.Run("policy.change "+c.step, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.applyPolicy("60", c.step)
			f.setPassword(AdminLogin, pw)
			a := f.signedIn(AdminLogin, pw)
			f.advance(c.window)
			if got := f.trustPropose(a, "60", "5"); got.status != 200 {
				t.Errorf("at the window %+v", got)
			}
			f.advance(time.Nanosecond)
			refused(t, "past the window", f.trustPropose(a, "61", "5"))
			page := f.policyRead(a).body
			if fmt.Sprint(page["step_up_seconds"]) != fmt.Sprint(int64(c.window/time.Second)) || page["act_until"] != stamp(start.Add(c.window)) {
				t.Errorf("page window %+v", page)
			}
		})
	}
}

func TestValidateUsesTheCandidateWindow(t *testing.T) {
	for _, c := range []struct{ name, age, step string }{{"candidate rule", "60", "5"}, {"candidate maximum age", "5", "0"}} {
		t.Run(c.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(Admin2Login, pw)
			id := changeOf(t, f.trustPropose(f.as(AdminLogin), c.age, c.step))
			second := f.signedIn(Admin2Login, pw)
			f.advance(5 * time.Minute)
			if f.reauth(second, pw).status != 204 {
				t.Fatal("reauthenticate")
			}
			for _, row := range f.policyRead(second).body["changes"].([]any) {
				if r := row.(map[string]any); r["id"] == id && r["act_until"] != stamp(start.Add(10*time.Minute)) {
					t.Errorf("row act until %+v", r)
				}
			}
			f.advance(10 * time.Minute)
			before := f.s.counter
			refused(t, "dead end becomes a prompt", f.trustValidate(second, id))
			if f.s.counter != before || f.reauth(second, pw).status != 204 {
				t.Fatal("reauthenticate")
			}
			if got := f.trustValidate(second, id); got.status != 200 {
				t.Errorf("validation after reauthentication %+v", got)
			}
			if limit, _ := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, "trust.approve", id); limit != stepUpDefault {
				t.Errorf("only validation takes the candidate window %v", limit)
			}
		})
	}
	f := setup(t, SeedSample).ready()
	id := changeOf(t, f.trustPropose(f.as(AdminLogin), "5", "0"))
	for _, target := range []string{"trust-x", "trust-00000000-9", ""} {
		if limit, mfa := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, validateCommand, target); limit != stepUpDefault || mfa {
			t.Errorf("target %q %v %v", target, limit, mfa)
		}
	}
	if limit, _ := f.s.stepUpTerms("elsewhere", SampleAdmin2, validateCommand, id); limit != stepUpDefault {
		t.Errorf("a change of another workspace contributes nothing %v", limit)
	}
	if limit, _ := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, validateCommand, id); limit != 5*time.Minute {
		t.Errorf("the candidate window %v", limit)
	}
	empty, _ := newPolicy(f)
	f.s.policy = empty
	if limit, mfa := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, "identity.suspend_user", ""); limit != stepUpDefault || mfa {
		t.Errorf("no policy %v %v", limit, mfa)
	}
	f.s.trust = nil
	if limit, _ := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, validateCommand, id); limit != stepUpDefault {
		t.Errorf("no registry %v", limit)
	}
}

func TestMFARulesFailClosed(t *testing.T) {
	for _, c := range []struct {
		name string
		rule authpolicy.StepUp
		run  func(f world, a actor) reply
	}{
		{"policy", authpolicy.StepUp{Action: authpolicy.ActionPolicyChange, MaxAge: 30 * time.Minute, MFA: true}, func(f world, a actor) reply { return f.trustPropose(a, "60", "0") }},
		{"suspend", authpolicy.StepUp{Action: ActionSensitive, MaxAge: 30 * time.Minute, MFA: true}, func(f world, a actor) reply {
			return f.run(a, "identity.suspend_user", SampleMember, nil)
		}},
		{"sessions/end", authpolicy.StepUp{Action: ActionSensitive, MaxAge: 30 * time.Minute, MFA: true}, func(f world, a actor) reply {
			return f.selfPost(a, "sessions/end", `{"session":"others"}`)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(AdminLogin, pw)
			f.directPolicy(authpolicy.Revision{MaxAge: time.Hour, StepUp: []authpolicy.StepUp{c.rule}})
			a := f.signedIn(AdminLogin, pw)
			before := f.s.counter
			got := c.run(f, a)
			f.code(t, "fresh session", got, 422, string(commands.PrerequisiteFailed))
			if got.body["next_action"] != string(commands.NextCorrect) || f.s.counter != before {
				t.Errorf("mfa refusal %+v", got)
			}
			if f.reauth(a, pw).status != 204 {
				t.Fatal("reauthenticate")
			}
			f.code(t, "after reauthentication", c.run(f, a), 422, string(commands.PrerequisiteFailed))
		})
	}
}

func TestStepUpCheckWithoutSession(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if code := f.s.stepUpCheck(identity.Verifier{9}, SampleAdmin, "identity.suspend_user", ""); code != commands.Unauthenticated {
		t.Errorf("missing session %s", code)
	}
	result := stepUpRefusal(commands.Unauthenticated, "r")
	if result.Err.Status != 401 || result.Err.Next != commands.NextAnswer || result.Request != "r" {
		t.Errorf("refusal %+v", result.Err)
	}
	a := f.as(AdminLogin)
	v := a.verifier(f)
	f.s.sessions.mu.Lock()
	f.s.sessions.byID[v].authenticated = f.s.clock().Add(time.Nanosecond)
	f.s.sessions.mu.Unlock()
	if code := f.s.stepUpCheck(v, SampleAdmin, "identity.suspend_user", ""); code != CodeReauthentication {
		t.Errorf("future authentication time %s", code)
	}
	if stepUpAction("trust.cancel") != authpolicy.ActionPolicyChange || stepUpAction("policy.withdraw") != ActionSensitive || stepUpAction("sessions/end") != ActionSensitive || stepUpAction("policyx") != ActionSensitive {
		t.Error("action names")
	}
}

func TestOpenAPIStepUp(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.register("peek", reader{f.s}, false)
	got := f.do("GET", "/api/v1/openapi.json", "", nil)
	var doc map[string]any
	if err := json.Unmarshal([]byte(got.raw), &doc); err != nil {
		t.Fatal(err)
	}
	props := doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"].(map[string]any)["properties"].(map[string]any)
	if !strings.Contains(fmt.Sprint(props["next_action"].(map[string]any)["enum"]), NextReauthenticate) || !strings.Contains(fmt.Sprint(props["code"].(map[string]any)["enum"]), CodeReauthentication) {
		t.Error("enums")
	}
	listed := false
	for _, e := range doc["x-errors"].([]any) {
		row := e.(map[string]any)
		if row["code"] == CodeReauthentication {
			listed = fmt.Sprint(row["status"]) == "403" && row["next_action"] == NextReauthenticate && row["retry"] == false
		}
		if row["code"] == CodeCSRF && row["next_action"] != string(commands.NextNone) {
			t.Error("other boundary codes keep their next action")
		}
	}
	if !listed {
		t.Error("x-errors")
	}
	marked := 0
	for path, item := range doc["paths"].(map[string]any) {
		post := item.(map[string]any)["post"].(map[string]any)
		id := strings.TrimPrefix(path, "/commands/")
		if (post["x-step-up"] == true) != f.s.stepUp(id) {
			t.Errorf("%s x-step-up %v", path, post["x-step-up"])
		}
		if post["x-step-up"] == true {
			marked++
		}
	}
	if marked < 64 || strings.Contains(got.raw, pw) {
		t.Errorf("marked %d", marked)
	}
	if doc["paths"].(map[string]any)["/commands/peek"].(map[string]any)["post"].(map[string]any)["x-step-up"] != nil {
		t.Error("reading command marked")
	}
}

func TestStepUpLogLine(t *testing.T) {
	logs := &syncWriter{}
	f := setup(t, SeedSample).ready()
	f.s.cfg.Log = captureLog(logs)
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	f.setPassword(AdminLogin, pw)
	a := f.signedIn(AdminLogin, pw)
	f.advance(stepUpDefault + time.Nanosecond)
	probeCall(f, a, "k1", "req-1")
	f.selfPost(a, "sessions/end", `{"session":"others"}`)
	lines := logLines(t, logs.String(), "step_up_required")
	if len(lines) != 2 {
		t.Fatalf("lines %v", lines)
	}
	for _, line := range lines {
		keys := []string{}
		for k := range line {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		joined := strings.Join(keys, ",")
		if joined != "command,level,msg,time,user" && joined != "level,msg,route,time,user" || line["user"] != SampleAdmin {
			t.Errorf("log line %v", line)
		}
	}
}

func TestStepUpGrantsNoPermission(t *testing.T) {
	f, _ := probeWorld(t)
	f.setPassword(MemberLogin, pw)
	member := f.signedIn(MemberLogin, pw)
	f.advance(stepUpDefault + time.Nanosecond)
	refused(t, "member asked first", probeCall(f, member, "k1", "req-1"))
	if f.reauth(member, pw).status != 204 {
		t.Fatal("reauthenticate")
	}
	f.code(t, "then refused by the gate", probeCall(f, member, "k1", "req-1"), 403, string(commands.PermissionDenied))
}

type syncWriter struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func captureLog(w io.Writer) *slog.Logger { return slog.New(slog.NewJSONHandler(w, nil)) }

func logLines(t testing.TB, raw, msg string) []map[string]any {
	t.Helper()
	out := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["msg"] == msg {
			out = append(out, m)
		}
	}
	return out
}

func TestProposalLimitStillMapped(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for i := 0; i < trustchange.MaxOpenPerProposer; i++ {
		if got := f.trustPropose(root, fmt.Sprint(10+i), "0"); got.status != 200 {
			t.Fatalf("propose %d %+v", i, got)
		}
	}
	f.code(t, "pending limit", f.trustPropose(root, "40", "0"), 503, string(commands.DependencyUnavailable))
}

func TestCandidateRuleOnlyAppliesToValidate(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	root := f.signedIn(AdminLogin, pw)
	id := changeOf(t, f.trustPropose(root, "60", "5"))
	f.advance(10 * time.Minute)
	if got := f.trustEnd(root, "trust.cancel", id); got.status != 200 || got.body["business_state"] != "cancelled" {
		t.Errorf("cancel judged by the candidate's 5-minute rule: %+v", got)
	}
}
