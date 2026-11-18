package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) servicesRead(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/services", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) service(a actor, id string) map[string]any {
	f.t.Helper()
	for _, row := range f.servicesRead(a).body["services"].([]any) {
		if m := row.(map[string]any); m["id"] == id {
			return m
		}
	}
	return nil
}

func (f world) newService(a actor, name string) string {
	f.t.Helper()
	got := f.run(a, "service.create", servicesTarget, map[string]string{"name": name, "owner": SampleAdmin})
	if got.status != 200 || got.body["business_state"] != "service_created" {
		f.t.Fatalf("create %s %+v", name, got)
	}
	return servicePrefix + name
}

func (f world) issued(a actor, service string, days string) (string, string, reply) {
	f.t.Helper()
	got := f.run(a, "service.issue_credential", service, map[string]string{"days": days})
	if got.status != 200 {
		f.t.Fatalf("issue %+v", got)
	}
	fields := got.body["fields"].(map[string]any)
	return fields["credential"].(string), fields["secret_ref"].(string), got
}

func (f world) failed(t testing.TB, name string, got reply, code commands.Code, fields ...string) {
	t.Helper()
	if got.body["code"] != string(code) || (len(fields) > 0 && fmt.Sprint(got.body["fields"]) != fmt.Sprint(fields)) {
		t.Errorf("%s: got %d %s, want %s %v", name, got.status, got.raw, code, fields)
	}
}

func (f world) runWith(a actor, revision, command, target string, fields map[string]string) reply {
	f.t.Helper()
	n := keys.Add(1)
	body, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("req-%d", n), "target_id": target, "idempotency_key": fmt.Sprintf("key-%d", n),
		"expected_revision": revision, "rationale": "sample reason", "body": fields})
	return f.call(command, string(body), a.cookie, a.csrf)
}

func secretBytes(t testing.TB, secret string) []byte {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(secret, secretPrefix))
	if !strings.HasPrefix(secret, secretPrefix) || err != nil || len(raw) != tokenBytes {
		t.Fatalf("secret format %q %v", secret, err)
	}
	return raw
}

func TestServicesReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.servicesRead(root)
	if got.status != 200 || len(got.body["services"].([]any)) != 0 || got.body["now"] != stamp(start) || got.body["revision"] != f.s.Revision() {
		t.Fatalf("empty read %+v", got)
	}
	limits := got.body["limits"].(map[string]any)
	if fmt.Sprint(limits["credential_days"], limits["credentials_per_service"], limits["services"]) != "30 10 200" {
		t.Errorf("limits %v", limits)
	}
	owners := fmt.Sprint(got.body["owners"])
	if !strings.Contains(owners, SampleMember) || !strings.Contains(owners, MemberLogin) || len(got.body["owners"].([]any)) != 4 {
		t.Errorf("owners %s", owners)
	}
	f.code(t, "no session", f.servicesRead(actor{}), 401, string(commands.Unauthenticated))
	f.code(t, "member", f.servicesRead(f.as(MemberLogin)), 403, string(commands.PermissionDenied))
	f.s.fault = func(at string) bool { return at == "roster" }
	f.code(t, "roster", f.servicesRead(root), 503, string(commands.DependencyUnavailable))
	f.s.fault = func(at string) bool { return at == "services" }
	f.code(t, "services", f.servicesRead(root), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	svc := f.newService(root, "export")
	f.newService(root, "second")
	f.s.fault = func(at string) bool { return at == "credentials" }
	f.code(t, "credentials", f.servicesRead(root), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	row := f.service(root, svc)
	if row["state"] != "active" || fmt.Sprint(row["version"]) != "1" || row["creator"] != SampleAdmin || row["creator_login"] != AdminLogin ||
		row["owner"] != SampleAdmin || row["owner_login"] != AdminLogin || len(row["credentials"].([]any)) != 0 {
		t.Errorf("row %+v", row)
	}
}

func TestServiceCreateRules(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for _, name := range []string{"", "ab", strings.Repeat("a", 49), "-ab", "ab-", "a--b", "Abc", "a_b", "a b", "é12"} {
		f.failed(t, "name "+name, f.run(root, "service.create", servicesTarget, map[string]string{"name": name, "owner": SampleMember}), commands.InvalidRequest)
	}
	for _, name := range []string{"abc", strings.Repeat("a", 48), "a-b-c", "0x9"} {
		f.newService(root, name)
	}
	f.failed(t, "duplicate", f.run(root, "service.create", servicesTarget, map[string]string{"name": "abc", "owner": SampleMember}), commands.InvalidRequest)
	f.failed(t, "unknown owner", f.run(root, "service.create", servicesTarget, map[string]string{"name": "own", "owner": "ghost"}), commands.InvalidRequest)
	f.failed(t, "service owner", f.run(root, "service.create", servicesTarget, map[string]string{"name": "own", "owner": "svc-abc"}), commands.InvalidRequest)
	self := f.run(root, "service.create", servicesTarget, map[string]string{"name": "mine", "owner": SampleAdmin})
	if self.status != 200 {
		t.Errorf("self-owned %+v", self)
	}
	f.failed(t, "wrong target", f.run(root, "service.create", "elsewhere", map[string]string{"name": "x12", "owner": SampleMember}), commands.PermissionDenied)
	f.failed(t, "member", f.runWith(f.as(MemberLogin), f.revision(root), "service.create", servicesTarget, map[string]string{"name": "x12", "owner": SampleMember}), commands.PermissionDenied)
	rev := f.revision(root)
	f.s.fault = func(at string) bool { return at == "roster" }
	f.failed(t, "roster", f.runWith(root, rev, "service.create", servicesTarget, map[string]string{"name": "x12", "owner": SampleAdmin}), commands.DependencyUnavailable)
	f.s.fault = func(at string) bool { return at == "services" }
	f.failed(t, "services", f.runWith(root, rev, "service.create", servicesTarget, map[string]string{"name": "x12", "owner": SampleAdmin}), commands.DependencyUnavailable)
	f.s.fault = nil
	got := f.run(root, "service.create", servicesTarget, map[string]string{"name": "x12", "owner": SampleMember, "extra": "1"})
	f.failed(t, "extra field", got, commands.InvalidRequest)
}

func TestServiceCap(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	v := root.verifier(f)
	for i := 0; i < maxServices; i++ {
		if err := f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, fmt.Sprintf("svc-cap%03d", i), SampleAdmin); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.s.identity.Workspace().RetireServiceUser(v, SampleWorkspace, "svc-cap000", 1); err != nil {
		t.Fatal(err)
	}
	f.failed(t, "at the cap", f.run(root, "service.create", servicesTarget, map[string]string{"name": "over", "owner": SampleMember}), commands.PrerequisiteFailed)
	if got := f.servicesRead(root); got.status != 200 || len(got.body["services"].([]any)) != maxServices {
		t.Errorf("paged read %d", len(got.body["services"].([]any)))
	}
}

func TestCredentialIssueAndPickup(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewTextHandler(&logs, nil)) })
	root := f.as(AdminLogin)
	svc := f.newService(root, "export")
	for _, days := range []string{"0", "31", "x", ""} {
		f.failed(t, "days "+days, f.run(root, "service.issue_credential", svc, map[string]string{"days": days}), commands.InvalidRequest, "days")
	}
	id, ref, result := f.issued(root, svc, "30")
	if result.body["business_state"] != "credential_issued" || !strings.HasPrefix(id, "cred-") || len(id) != 21 {
		t.Errorf("issue %+v", result)
	}
	second := f.as(AdminLogin)
	f.code(t, "another session", f.collect(second, ref), 403, string(commands.PermissionDenied))
	got := f.collect(root, ref)
	if got.status != 200 || got.header.Get("Cache-Control") != "no-store" || got.body["credential"] != id || got.body["url"] != nil {
		t.Fatalf("collect %+v", got)
	}
	secret := got.body["secret"].(string)
	raw := secretBytes(t, secret)
	f.code(t, "second collect", f.collect(root, ref), 403, string(commands.PermissionDenied))
	answer, err := f.s.identity.Boundary().AccessByCredential(id, verifierOf(raw))
	if err != nil || answer.User() != svc || !answer.NotAfter().Equal(start.Add(30*24*time.Hour)) {
		t.Errorf("credential answer %+v %v", answer, err)
	}
	if _, err := f.s.identity.Boundary().AccessByCredential(id, verifierOf([]byte(secret))); err == nil {
		t.Error("the prefixed text is accepted as the verifier input")
	}
	journal, _ := json.Marshal(f.s.commands.Journal(0, 1000))
	page := f.servicesRead(root)
	for name, text := range map[string]string{"result": result.raw, "journal": string(journal), "logs": logs.String(), "read": page.raw} {
		if strings.Contains(text, secret) || strings.Contains(text, strings.TrimPrefix(secret, secretPrefix)) {
			t.Errorf("secret in %s", name)
		}
	}
	creds := f.service(root, svc)["credentials"].([]any)
	row := creds[0].(map[string]any)
	if len(creds) != 1 || row["id"] != id || row["state"] != "active" || row["usable"] != true || row["not_before"] != stamp(start) || row["not_after"] != stamp(start.Add(30*24*time.Hour)) {
		t.Errorf("credential row %+v", creds)
	}
	_, first, _ := f.issued(root, svc, "1")
	_, kept, _ := f.issued(root, svc, "1")
	if f.collect(root, first).status != 200 || f.collect(root, kept).status != 200 {
		t.Error("an earlier unread pickup was dropped by a later issue")
	}
	revoked, dropped, _ := f.issued(root, svc, "1")
	if got := f.run(root, "service.revoke_credential", revoked, nil); got.status != 200 || got.body["business_state"] != "credential_revoked" {
		t.Fatalf("revoke %+v", got)
	}
	f.code(t, "pickup of a revoked credential", f.collect(root, dropped), 403, string(commands.PermissionDenied))
}

func TestCredentialLimitAndOrder(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "limit")
	ids := []string{}
	for i := 0; i < 10; i++ {
		id, _, _ := f.issued(root, svc, fmt.Sprint(1+i%2))
		ids = append(ids, id)
	}
	f.failed(t, "at the limit", f.run(root, "service.issue_credential", svc, map[string]string{"days": "1"}), commands.PrerequisiteFailed)
	f.advance(24 * time.Hour)
	root = f.as(AdminLogin)
	if _, _, got := f.issued(root, svc, "3"); got.status != 200 {
		t.Error("expired credentials still count")
	}
	f.run(root, "service.revoke_credential", ids[1], nil)
	rows := f.service(root, svc)["credentials"].([]any)
	states := []string{}
	for _, r := range rows {
		m := r.(map[string]any)
		states = append(states, m["state"].(string))
	}
	if strings.Join(states, ",") != "active,active,active,active,active,revoked,expired,expired,expired,expired,expired" {
		t.Errorf("order %v", states)
	}
	f.failed(t, "revoke twice", f.run(root, "service.revoke_credential", ids[1], nil), commands.InvalidStage)
	if got := f.run(root, "service.revoke_credential", ids[0], nil); got.status != 200 {
		t.Errorf("revoking an expired credential %+v", got)
	}
	f.failed(t, "unknown credential", f.run(root, "service.revoke_credential", "cred-ghost", nil), commands.PermissionDenied)
	f.s.fault = func(at string) bool { return at == "credentials" }
	f.failed(t, "credentials unavailable", f.run(root, "service.revoke_credential", ids[2], nil), commands.DependencyUnavailable)
	f.s.fault = nil
}

func TestServiceStatesAndRetire(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	svc := f.newService(root, "states")
	id, ref, _ := f.issued(root, svc, "5")
	f.failed(t, "reinstate active", f.run(root, "service.reinstate", svc, nil), commands.InvalidStage)
	if got := f.run(root, "service.suspend", svc, nil); got.body["business_state"] != "service_suspended" {
		t.Fatalf("suspend %+v", got)
	}
	f.code(t, "pickup after suspend", f.collect(root, ref), 403, string(commands.PermissionDenied))
	row := f.service(root, svc)
	cred := row["credentials"].([]any)[0].(map[string]any)
	if row["state"] != "suspended" || cred["state"] != "active" || cred["usable"] != false {
		t.Errorf("suspended %+v", row)
	}
	f.failed(t, "issue while suspended", f.run(root, "service.issue_credential", svc, map[string]string{"days": "1"}), commands.PrerequisiteFailed)
	f.failed(t, "suspend twice", f.run(root, "service.suspend", svc, nil), commands.InvalidStage)
	if got := f.run(root, "service.reinstate", svc, nil); got.body["business_state"] != "service_active" {
		t.Fatalf("reinstate %+v", got)
	}
	f.run(root, "service.revoke_credential", id, nil)
	live, pending, _ := f.issued(root, svc, "5")
	if got := f.run(root, "service.retire", svc, nil); got.body["business_state"] != "service_retired" {
		t.Fatalf("retire %+v", got)
	}
	f.code(t, "pickup after retire", f.collect(root, pending), 403, string(commands.PermissionDenied))
	for _, c := range f.service(root, svc)["credentials"].([]any) {
		if c.(map[string]any)["state"] != "revoked" {
			t.Errorf("credential after retire %+v", c)
		}
	}
	if _, err := f.s.identity.Boundary().AccessByCredential(live, identity.Verifier{1}); err == nil {
		t.Error("retired credential answers")
	}
	f.failed(t, "retire twice", f.run(root, "service.retire", svc, nil), commands.InvalidStage)
	f.failed(t, "issue after retire", f.run(root, "service.issue_credential", svc, map[string]string{"days": "1"}), commands.InvalidStage)
	f.failed(t, "member", f.runWith(f.as(MemberLogin), f.revision(root), "service.suspend", svc, map[string]string{}), commands.PermissionDenied)
	f.failed(t, "human target", f.run(root, "service.suspend", SampleMember, nil), commands.PermissionDenied)
}

func TestPartialRetireAndFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "partial")
	id, _, _ := f.issued(root, svc, "5")
	f.s.fault = func(at string) bool { return at == "retire" }
	f.failed(t, "partial", f.run(root, "service.retire", svc, nil), commands.InternalError)
	f.s.fault = nil
	if row := f.service(root, svc); row["state"] != "retired" || row["credentials"].([]any)[0].(map[string]any)["id"] != id {
		t.Errorf("after partial %+v", row)
	}
	other := f.newService(root, "faulty")
	f.issued(root, other, "5")
	f.s.fault = func(at string) bool { return at == "credentials" }
	f.failed(t, "credentials at issue", f.run(root, "service.issue_credential", other, map[string]string{"days": "1"}), commands.DependencyUnavailable)
	f.failed(t, "credentials at retire", f.run(root, "service.retire", other, nil), commands.InternalError)
	f.s.fault = func(at string) bool { return at == "services" }
	f.failed(t, "services at resolve", f.run(root, "service.suspend", other, nil), commands.DependencyUnavailable)
	f.s.fault = nil
	f.s.caller = root.verifier(f)
	defer func() { f.s.caller = identity.Verifier{} }()
	h := serviceCommand{s: f.s, kind: "suspend"}
	if _, res := h.Resolve("elsewhere", other); res != commands.NotFound {
		t.Errorf("other workspace %v", res)
	}
	if _, err := h.Apply(commands.Context{Target: other, Rationale: strings.Repeat("r", maxRationale+1)}); err.(*commands.Failure).Code != commands.InvalidRequest {
		t.Errorf("long rationale %v", err)
	}
	f.s.fault = func(at string) bool { return at == "services" }
	if _, err := h.Apply(commands.Context{Target: other}); err.(*commands.Failure).Code != commands.DependencyUnavailable {
		t.Errorf("services at apply %v", err)
	}
	f.s.fault = nil
	if _, err := (serviceCommand{s: f.s, kind: "revoke_credential"}).Apply(commands.Context{Target: "cred-ghost"}); err.(*commands.Failure).Code != commands.InvalidStage {
		t.Errorf("vanished credential %v", err)
	}
	if _, err := (serviceCommand{s: f.s, kind: "create"}).Apply(commands.Context{Fields: map[string]string{"name": "partial", "owner": SampleAdmin}}); err.(*commands.Failure).Fields[0] != "name" {
		t.Errorf("conflict %v", err)
	}
	saved := f.s.caller
	f.s.caller = identity.Verifier{9}
	if _, err := (serviceCommand{s: f.s, kind: "create"}).Apply(commands.Context{Fields: map[string]string{"name": "lost", "owner": SampleMember}}); err.(*commands.Failure).Code != commands.PermissionDenied {
		t.Errorf("create without session %v", err)
	}
	f.s.caller = saved
}

func TestServiceStatesAndNames(t *testing.T) {
	for state, want := range map[identity.UserState]string{identity.Active: "active", identity.Suspended: "suspended", identity.Departed: "retired"} {
		if serviceState(state) != want {
			t.Errorf("%s", state)
		}
	}
	if (&Server{dir: newDirectory()}).loginOf("ghost") != "" {
		t.Error("unknown login")
	}
}

func TestRetireWithManyEndedCredentials(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "crowded")
	for round := 0; round < 3; round++ {
		if round > 0 {
			f.advance(24 * time.Hour)
			root = f.as(AdminLogin)
		}
		for i := 0; i < 10; i++ {
			f.issued(root, svc, "1")
		}
	}
	if got := f.run(root, "service.retire", svc, nil); got.status != 200 || got.body["business_state"] != "service_retired" {
		t.Fatalf("retire %+v", got)
	}
	rows := f.service(root, svc)["credentials"].([]any)
	revoked := 0
	for _, r := range rows {
		if r.(map[string]any)["state"] == "revoked" {
			revoked++
		}
	}
	if len(rows) != identity.EndedCredentialsKept || revoked != 10 {
		t.Errorf("after retire %d rows, %d revoked", len(rows), revoked)
	}
}

func TestServiceOrderingAndLookups(t *testing.T) {
	at := func(h int) time.Time { return start.Add(time.Duration(h) * time.Hour) }
	list := []identity.CredentialInfo{
		{ID: "a", Validity: identity.Validity{NotBefore: at(-9), NotAfter: at(-1)}},
		{ID: "b", Validity: identity.Validity{NotBefore: at(0), NotAfter: at(5)}},
		{ID: "c", Validity: identity.Validity{NotBefore: at(0), NotAfter: at(9)}, Revoked: true},
		{ID: "d", Validity: identity.Validity{NotBefore: at(0), NotAfter: at(3)}},
	}
	orderCredentials(list, start)
	ids := ""
	for _, c := range list {
		ids += c.ID
	}
	if ids != "bdca" {
		t.Errorf("order %s", ids)
	}

	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "lookup")
	f.newService(root, "lookup2")
	if _, err := f.s.serviceUsers(identity.Verifier{7}); err == nil {
		t.Error("service users without a session")
	}
	f.s.caller = identity.Verifier{7}
	if _, res := (serviceCommand{s: f.s, kind: "suspend"}).Resolve(SampleWorkspace, svc); res != commands.NotFound {
		t.Errorf("resolve without a session %v", res)
	}
	f.s.caller = identity.Verifier{}
	calls := 0
	f.s.fault = func(at string) bool {
		if at == "credentials" {
			calls++
			return calls == 1
		}
		return false
	}
	f.code(t, "first credentials read fails", f.servicesRead(root), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	f.failed(t, "no owner", f.run(root, "service.create", servicesTarget, map[string]string{"name": "nobody"}), commands.InvalidRequest, "owner")
}

func TestServiceReplayRecoveryAndStalePickups(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "replay")
	stale := `{"request_id":"r1","target_id":"` + svc + `","idempotency_key":"k1","expected_revision":"r1","rationale":"x","body":{"days":"1"}}`
	f.code(t, "stale revision", f.call("service.issue_credential", stale, root.cookie, root.csrf), 409, string(commands.RevisionConflict))
	body, _ := json.Marshal(map[string]any{"request_id": "r9", "target_id": svc, "idempotency_key": "k9", "expected_revision": f.revision(root),
		"rationale": "x", "body": map[string]string{"days": "1"}})
	first := f.call("service.issue_credential", string(body), root.cookie, root.csrf)
	again := f.call("service.issue_credential", string(body), root.cookie, root.csrf)
	if again.status != 200 || fmt.Sprint(again.body["fields"]) != fmt.Sprint(first.body["fields"]) {
		t.Fatalf("replay %+v", again)
	}
	if n := len(f.service(root, svc)["credentials"].([]any)); n != 1 {
		t.Errorf("replay issued %d credentials", n)
	}
	ref := first.body["fields"].(map[string]any)["secret_ref"].(string)
	f.collect(root, ref)
	f.code(t, "replayed ref used", f.collect(root, ref), 403, string(commands.PermissionDenied))

	_, unread, _ := f.issued(root, svc, "1")
	f.advance(pickupValidity)
	root = f.as(AdminLogin)
	f.newService(root, "later")
	f.s.domain.Lock()
	_, kept := f.s.dir.pickups[unread]
	f.s.domain.Unlock()
	if kept {
		t.Error("an expired secret pickup stayed in memory after a service command")
	}

	other := f.newService(root, "recover")
	id, _, _ := f.issued(root, other, "5")
	f.s.fault = func(at string) bool { return at == "retire" }
	f.run(root, "service.retire", other, nil)
	f.s.fault = nil
	if row := f.service(root, other); row["state"] != "retired" {
		t.Fatalf("retire %+v", row)
	}
	f.failed(t, "revoking a credential the retire already revoked", f.run(root, "service.revoke_credential", id, nil), commands.InvalidStage)
}

func TestRetireRecoversFromAnUnrevokedCredential(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "unrevoked")
	id, _, _ := f.issued(root, svc, "5")
	f.s.domain.Lock()
	f.s.caller = root.verifier(f)
	if err := f.s.identity.Workspace().RetireServiceUser(f.s.caller, SampleWorkspace, svc, 1); err != nil {
		t.Fatal(err)
	}
	f.s.caller = identity.Verifier{}
	f.s.domain.Unlock()
	if got := f.run(root, "service.revoke_credential", id, nil); got.status != 200 {
		t.Errorf("recovery revoke %+v", got)
	}
	if row := f.service(root, svc); row["credentials"].([]any)[0].(map[string]any)["state"] != "revoked" {
		t.Errorf("after recovery %+v", row)
	}
}
