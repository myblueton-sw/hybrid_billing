package server

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
)

type licenseTestAuthority struct {
	now       time.Time
	available bool
}

func (a *licenseTestAuthority) Now() (time.Time, bool) { return a.now, a.available }
func licenseWorld(t *testing.T, authority licensemode.Authority) world {
	f := setup(t, SeedSample)
	f.s.cfg.License = LicenseSettings{Installation: "test-installation", Authority: authority}
	f = f.ready()
	t.Cleanup(func() { _ = f.s.Close() })
	return f
}
func (f world) licenseRead(a actor) reply {
	return f.do("GET", "/api/v1/licenses", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}
func requireLicenseOutcome(t *testing.T, r reply, want string) {
	t.Helper()
	if r.status != 200 || r.body["fields"].(map[string]any)["outcome"] != want {
		t.Fatalf("want %s: %s", want, r.raw)
	}
}
func TestLicenseAPIBindingAuthorizationAndRestart(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	f.code(t, "anonymous", f.licenseRead(actor{}), 401, "unauthenticated")
	f.code(t, "csrf", f.licenseRead(actor{cookie: root.cookie}), 403, CodeCSRF)
	f.setPassword(MemberLogin, pw)
	f.code(t, "member", f.licenseRead(f.as(MemberLogin)), 403, "permission_denied")
	read := f.licenseRead(root)
	if read.status != 200 || read.body["state"] != "uninitialized" || read.body["time_authority"] != "not_connected" {
		t.Fatal(read.raw)
	}
	if _, err := os.Stat(f.s.licenseDir()); !os.IsNotExist(err) {
		t.Fatal("read initialized state")
	}
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "already_initialized")
	requireLicenseOutcome(t, f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "evaluation"}), "mode_selected")
	read = f.licenseRead(root)
	status := read.body["status"].(map[string]any)
	if status["state"] != "unverified" || status["reason"] != "time_unavailable" || status["evaluation"] != nil || status["last_verified_at"] != "" {
		t.Fatal(read.raw)
	}
	requireLicenseOutcome(t, f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "noncommercial"}), "mode_selected")
	if err := f.s.license.Close(); err != nil {
		t.Fatal(err)
	}
	f.s.license = nil
	if err := f.s.openLicense(); err != nil {
		t.Fatal(err)
	}
	if f.licenseRead(root).body["status"].(map[string]any)["state"] != "noncommercial" {
		t.Fatal("mode lost on open")
	}
	if target, res := (licenseHandler{f.s, "initialize"}).Resolve("other-workspace", licenseTarget); res != 0 || target.Revision != "" {
		t.Fatal("foreign workspace resolved")
	}
	if _, res := (licenseHandler{f.s, "initialize"}).Resolve(f.s.bootWorkspace, "other-target"); res != 0 {
		t.Fatal("foreign target resolved")
	}
}
func TestLicenseMissingConfigurationAndBadBinding(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	read := f.licenseRead(root)
	if read.body["state"] != stateNotConfigured || read.body["installation"] != nil || read.body["status"] != nil {
		t.Fatal(read.raw)
	}
	for _, id := range []string{"../bad", "has space", strings.Repeat("a", 129)} {
		cfg := f.s.cfg
		cfg.License.Installation = id
		if _, err := New(cfg); err == nil {
			t.Fatal("invalid id")
		}
	}
	configured := licenseWorld(t, nil)
	a := configured.as(AdminLogin)
	requireLicenseOutcome(t, configured.run(a, "licenses.initialize", licenseTarget, nil), "initialized")
	if err := configured.s.license.Close(); err != nil {
		t.Fatal(err)
	}
	configured.s.license = nil
	configured.s.cfg.License.Installation = "wrong-installation"
	if err := configured.s.openLicense(); err == nil {
		t.Fatal("wrong bound store opened")
	}
	configured.s.cfg.License.Installation = "test-installation"
	if err := os.WriteFile(filepath.Join(configured.s.licenseDir(), "state.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := configured.s.openLicense(); err == nil {
		t.Fatal("corrupt store accepted")
	}
}
func licenseTestKey(t *testing.T, f world) ed25519.PrivateKey {
	t.Helper()
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{17}, ed25519.SeedSize))
	var message bytes.Buffer
	for _, value := range []string{"hybrid-billing/license-key/v1", "test-key", "test-issuer"} {
		_ = binary.Write(&message, binary.BigEndian, uint64(len(value)))
		message.WriteString(value)
	}
	key := licensemode.Key{ID: "test-key", Issuer: "test-issuer", Public: hex.EncodeToString(private.Public().(ed25519.PublicKey)), Proof: hex.EncodeToString(ed25519.Sign(private, message.Bytes())), NotBefore: start.Add(-time.Hour).Unix(), NotAfter: start.Add(365 * 24 * time.Hour).Unix(), AcceptUntil: start.Add(366 * 24 * time.Hour).Unix()}
	p, err := f.s.license.Propose(licensemode.Change{Kind: licensemode.ChangeKey, Key: key}, licensemode.Actor{ID: SampleAdmin, Kind: licensemode.KindHuman})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.license.Approve(p.Number, licensemode.Actor{ID: SampleAdmin2, Kind: licensemode.KindHuman})
	if err != nil {
		t.Fatal(err)
	}
	return private
}
func signedLicenseFields(c licensemode.Claims, private ed25519.PrivateKey) map[string]string {
	body := licensemode.Encode(c)
	signature := ed25519.Sign(private, append([]byte("hybrid-billing/license/v1"), body...))
	encoded := base64.StdEncoding.EncodeToString(body)
	fields := map[string]string{"signature": base64.StdEncoding.EncodeToString(signature)}
	if len(encoded) > 4096 {
		fields["body"], fields["body_tail"] = encoded[:4096], encoded[4096:]
	} else {
		fields["body"] = encoded
	}
	return fields
}
func TestLicenseSignedAPIAndAuthorityOutcomes(t *testing.T) {
	authority := &licenseTestAuthority{start, true}
	f := licenseWorld(t, authority)
	root := f.as(AdminLogin)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	private := licenseTestKey(t, f)
	claims := licensemode.Claims{Kind: licensemode.DocEvaluation, ID: "grant-1", Issuer: "test-issuer", KeyID: "test-key", IssuedAt: start.Unix(), SubjectID: "synthetic-subject", Subject: "Synthetic", Product: licenseProduct, Installation: "test-installation", Scope: []string{"core"}, Terms: "evaluation-v1"}
	requireLicenseOutcome(t, f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "evaluation"}), "mode_selected")
	fields := signedLicenseFields(claims, private)
	fields["terms"] = "wrong"
	requireLicenseOutcome(t, f.run(root, "licenses.activate", licenseTarget, fields), "terms_mismatch")
	fields["terms"] = "evaluation-v1"
	requireLicenseOutcome(t, f.run(root, "licenses.activate", licenseTarget, fields), "activation_recorded")
	read := f.licenseRead(root)
	status := read.body["status"].(map[string]any)
	if status["state"] != "evaluation_active" || status["remaining_seconds"] != float64(336*3600) {
		t.Fatal(read.raw)
	}
	requireLicenseOutcome(t, f.run(root, "licenses.activate", licenseTarget, fields), "activation_recorded")
	authority.available = false
	read = f.licenseRead(root)
	status = read.body["status"].(map[string]any)
	if status["state"] != "unverified" || status["reason"] != "time_unavailable" || status["last_verified"] != "evaluation_active" {
		t.Fatal(read.raw)
	}
	authority.available = true
	requireLicenseOutcome(t, f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "commercial"}), "mode_selected")
	claims.Kind, claims.ID, claims.Terms, claims.Agreement, claims.Channel, claims.Metric = licensemode.DocCommercial, "entitlement-1", "", "test-agreement", licensemode.ChannelDirect, licensemode.MetricInstallations
	claims.Start, claims.End = start.Unix(), start.Add(time.Hour).Unix()
	fields = signedLicenseFields(claims, private)
	requireLicenseOutcome(t, f.run(root, "licenses.register", licenseTarget, fields), "accepted")
	requireLicenseOutcome(t, f.run(root, "licenses.register", licenseTarget, fields), "repeat")
	if strings.Contains(f.licenseRead(root).raw, "test-key") {
		t.Fatal("key metadata leaked")
	}
	claims.KeyID = "unknown"
	requireLicenseOutcome(t, f.run(root, "licenses.register", licenseTarget, signedLicenseFields(claims, private)), "unverifiable")
	claims.KeyID = "test-key"
	claims.Installation = "other"
	requireLicenseOutcome(t, f.run(root, "licenses.register", licenseTarget, signedLicenseFields(claims, private)), "scope_mismatch")
}
func TestLicenseStorageFailureUsesUnknownOutcome(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	if err := os.Remove(filepath.Join(f.s.licenseDir(), "state.json")); err != nil {
		t.Fatal(err)
	}
	got := f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "noncommercial"})
	f.code(t, "unknown", got, 500, "internal_error")
	if got.body["next_action"] != "inspect_operation" || (got.body["operation_id"] == nil || fmt.Sprint(got.body["operation_id"]) == "") {
		t.Fatal(got.raw)
	}
	f.code(t, "poisoned read", f.licenseRead(root), 503, "dependency_unavailable")
}
func TestLicenseSignedInputBounds(t *testing.T) {
	signature := base64.StdEncoding.EncodeToString(make([]byte, 64))
	encoded := base64.StdEncoding.EncodeToString(make([]byte, 4096))
	if _, err := decodeLicense(map[string]string{"body": encoded[:4096], "body_tail": encoded[4096:], "signature": signature}); err != nil {
		t.Fatal(err)
	}
	for _, fields := range []map[string]string{
		{"body": "YQ==\n", "signature": signature},
		{"body": "YQ==", "body_tail": "Yg==", "signature": signature},
		{"body": "YQ==", "signature": signature[:87]},
		{"body": encoded[:4096], "body_tail": encoded[4096:] + "AAAA", "signature": signature},
	} {
		if _, err := decodeLicense(fields); err == nil {
			t.Fatal("accepted malformed signed input")
		}
	}
}

func TestLicenseCommandGuardsReplayAndSchema(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	revision := f.revision(root)
	f.code(t, "member command", f.runWith(member, revision, "licenses.initialize", licenseTarget, map[string]string{}), 403, "permission_denied")
	f.code(t, "foreign target", f.runWith(root, revision, "licenses.initialize", "wrong", map[string]string{}), 403, "permission_denied")
	body := `{"request_id":"license-replay","target_id":"license-installation","idempotency_key":"license-init-replay","expected_revision":"` + revision + `","rationale":"synthetic","body":{}}`
	f.code(t, "command csrf", f.call("licenses.initialize", body, root.cookie, "wrong"), 403, CodeCSRF)
	first := f.call("licenses.initialize", body, root.cookie, root.csrf)
	requireLicenseOutcome(t, first, "initialized")
	again := f.call("licenses.initialize", body, root.cookie, root.csrf)
	if again.status != 200 || again.body["replay"] != true {
		t.Fatal(again.raw)
	}
	f.code(t, "stale", f.runWith(root, revision, "licenses.select_mode", licenseTarget, map[string]string{"mode": "noncommercial"}), 409, "revision_conflict")
	oldWorkspace := f.s.bootWorkspace
	f.s.bootWorkspace = "other"
	read := f.licenseRead(root)
	if read.status != 200 || read.body["state"] != "not_configured" || read.body["installation"] != nil {
		t.Fatal(read.raw)
	}
	f.code(t, "foreign workspace", f.runWith(root, f.s.Revision(), "licenses.select_mode", licenseTarget, map[string]string{"mode": "noncommercial"}), 403, "permission_denied")
	f.s.bootWorkspace = oldWorkspace
	doc, err := f.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"/licenses"`, `"licenses.initialize"`, `"licenses.select_mode"`, `"licenses.register"`, `"licenses.activate"`} {
		if !bytes.Contains(doc, []byte(needle)) {
			t.Fatal("missing API contract", needle)
		}
	}
	current := f.s.Revision()
	f.advance(16 * time.Minute)
	f.code(t, "step up", f.runWith(root, current, "licenses.select_mode", licenseTarget, map[string]string{"mode": "noncommercial"}), 403, CodeReauthentication)
}

func TestLicenseNonhumanCannotInitialize(t *testing.T) {
	f := licenseWorld(t, nil)
	outcome, err := (licenseHandler{f.s, "initialize"}).Apply(commands.Context{User: "nonexistent-worker", Rationale: "synthetic"})
	if err == nil || outcome.BusinessState != "" || f.s.license != nil {
		t.Fatal("nonhuman initialized license state")
	}
	if _, err = os.Stat(f.s.licenseDir()); !os.IsNotExist(err) {
		t.Fatal("unauthorized state created")
	}
}

func TestLicenseFullServerRestartPreservesIdentityAndMode(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	requireLicenseOutcome(t, f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "noncommercial"}), "mode_selected")
	cfg := f.s.cfg
	if err := f.s.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	if err = next.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		t.Fatal(err)
	}
	next.manual = true
	if err = next.Start(); err != nil {
		t.Fatal(err)
	}
	g := world{t: t, s: next, h: next.Handler(), now: f.now, mu: f.mu, value: f.value}
	read := g.licenseRead(g.as(AdminLogin))
	if read.status != 200 || read.body["installation"] != "test-installation" || read.body["status"].(map[string]any)["state"] != "noncommercial" {
		t.Fatal(read.raw)
	}
	requireLicenseOutcome(t, g.run(g.as(AdminLogin), "licenses.initialize", licenseTarget, nil), "already_initialized")
	cfg.License.Installation = "other-installation"
	if err = next.Close(); err != nil {
		t.Fatal(err)
	}
	wrong, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer wrong.Close()
	if err = wrong.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		t.Fatal(err)
	}
	wrong.manual = true
	if err = wrong.Start(); err == nil {
		t.Fatal("wrong installation started")
	}
	if _, err = os.Stat(filepath.Join(f.s.licenseDir(), "owner")); !os.IsNotExist(err) {
		t.Fatal("failed open leaked ownership")
	}
}
