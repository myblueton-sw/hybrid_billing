package server

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
)

func trustKeyFields() (map[string]string, ed25519.PrivateKey) {
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{29}, ed25519.SeedSize))
	var message bytes.Buffer
	for _, value := range []string{"hybrid-billing/license-key/v1", "test-key", "test-issuer"} {
		_ = binary.Write(&message, binary.BigEndian, uint64(len(value)))
		message.WriteString(value)
	}
	return map[string]string{"key_id": "test-key", "issuer": "test-issuer", "public_key": hex.EncodeToString(private.Public().(ed25519.PublicKey)), "proof": hex.EncodeToString(ed25519.Sign(private, message.Bytes())), "not_before": fmt.Sprint(start.Add(-time.Hour).Unix()), "not_after": fmt.Sprint(start.Add(365 * 24 * time.Hour).Unix()), "accept_until": fmt.Sprint(start.Add(366 * 24 * time.Hour).Unix())}, private
}
func (f world) licenseTrustRead(a actor, path string) reply {
	return f.do("GET", "/api/v1/licenses/"+path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}
func licenseProposalTarget(t *testing.T, r reply) string {
	t.Helper()
	requireLicenseOutcome(t, r, licensemode.ProposalPending)
	return r.body["fields"].(map[string]any)["target"].(string)
}
func TestLicenseTrustHTTPDualControlAndRights(t *testing.T) {
	f := licenseWorld(t, &licenseTestAuthority{start, true})
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, private := trustKeyFields()
	target := licenseProposalTarget(t, f.run(root, "licenses.propose_key", licenseTarget, fields))
	f.code(t, "self approval", f.run(root, "licenses.approve_key_change", target, nil), 403, "self_approval_denied")
	f.code(t, "other withdrawal", f.run(second, "licenses.withdraw_key_change", target, nil), 403, "permission_denied")
	read := f.licenseTrustRead(root, "key-changes/1")
	sum := sha256.Sum256(private.Public().(ed25519.PublicKey))
	if read.status != 200 || !strings.Contains(read.raw, hex.EncodeToString(sum[:])) || strings.Contains(read.raw, fields["proof"]) || strings.Contains(read.raw, fields["public_key"]) {
		t.Fatal(read.raw)
	}
	f.must(f.s.license.Close())
	f.s.license = nil
	f.must(f.s.openLicense())
	requireLicenseOutcome(t, f.run(second, "licenses.approve_key_change", target, nil), licensemode.ProposalApplied)
	read = f.licenseTrustRead(root, "keys/test-key")
	if read.status != 200 || read.body["state"] != "configured" {
		t.Fatal(read.raw)
	}
	requireLicenseOutcome(t, f.run(root, "licenses.select_mode", licenseTarget, map[string]string{"mode": "commercial"}), "mode_selected")
	claims := licensemode.Claims{Kind: licensemode.DocCommercial, ID: "api-entitlement", Issuer: "test-issuer", KeyID: "test-key", IssuedAt: start.Unix(), SubjectID: "synthetic-subject", Subject: "Synthetic", Product: licenseProduct, Installation: "test-installation", Scope: []string{"core"}, Agreement: "synthetic-agreement", Channel: licensemode.ChannelDirect, Metric: licensemode.MetricInstallations, Start: start.Unix(), End: start.Add(time.Hour).Unix()}
	requireLicenseOutcome(t, f.run(root, "licenses.register", licenseTarget, signedLicenseFields(claims, private)), "accepted")
	target = licenseProposalTarget(t, f.run(root, "licenses.propose_compromise", licenseTarget, map[string]string{"key_id": "test-key"}))
	requireLicenseOutcome(t, f.run(second, "licenses.approve_key_change", target, nil), licensemode.ProposalApplied)
	read = f.licenseRead(root)
	status := read.body["status"].(map[string]any)
	if status["state"] != "unverified" || status["reason"] != "key_compromised" {
		t.Fatal(read.raw)
	}
	read = f.licenseTrustRead(root, "keys/test-key")
	if read.body["record"].(map[string]any)["compromised"] != true {
		t.Fatal(read.raw)
	}
}
func TestLicenseTrustGuardsAndCanonicalTargets(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	for _, path := range []string{"keys/test-key", "key-changes/1"} {
		f.code(t, "anonymous", f.licenseTrustRead(actor{}, path), 401, "unauthenticated")
		f.code(t, "member", f.licenseTrustRead(member, path), 403, "permission_denied")
		f.code(t, "csrf", f.licenseTrustRead(actor{cookie: root.cookie}, path), 403, CodeCSRF)
	}
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, _ := trustKeyFields()
	f.code(t, "member mutation", f.runWith(member, f.s.Revision(), "licenses.propose_key", licenseTarget, fields), 403, "permission_denied")
	for _, raw := range []string{"0", "01", "-1", "+1", "18446744073709551616", "1.0"} {
		if _, ok := licenseChangeNumber(raw); ok {
			t.Fatal(raw)
		}
		status, code := 403, "permission_denied"
		if raw == "+1" {
			status, code = 422, "invalid_request"
		}
		f.code(t, "bad target", f.run(root, "licenses.approve_key_change", licenseChangePrefix+raw, nil), status, code)
		f.code(t, "bad read", f.licenseTrustRead(root, "key-changes/"+raw), 422, "invalid_request")
	}
	for _, raw := range []string{"1", "9007199254740993", "18446744073709551615"} {
		n, ok := licenseChangeNumber(raw)
		if !ok || licenseChangeTarget(n) != licenseChangePrefix+raw {
			t.Fatal(raw)
		}
	}
	f.code(t, "missing change", f.run(root, "licenses.approve_key_change", licenseChangeTarget(99), nil), 422, "invalid_stage")
	if r := f.licenseTrustRead(root, "key-changes/99"); r.body["state"] != "not_found" {
		t.Fatal(r.raw)
	}
	target := licenseProposalTarget(t, f.run(root, "licenses.propose_key", licenseTarget, fields))
	requireLicenseOutcome(t, f.run(root, "licenses.withdraw_key_change", target, nil), licensemode.ProposalWithdrawn)
	if r := f.licenseTrustRead(root, "key-changes/1"); r.body["state"] != "not_found" {
		t.Fatal(r.raw)
	}
	old := f.s.bootWorkspace
	f.s.bootWorkspace = "other"
	if r := f.licenseTrustRead(root, "keys/test-key"); r.body["state"] != "not_configured" {
		t.Fatal(r.raw)
	}
	f.s.bootWorkspace = old
}
func TestLicenseTrustRejectsAndDoesNotRetainSubmittedSecrets(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, private := trustKeyFields()
	secret := hex.EncodeToString(private)
	fields["public_key"] = secret
	rejected := f.run(root, "licenses.propose_key", licenseTarget, fields)
	requireLicenseOutcome(t, rejected, licensemode.ReasonPrivateKey)
	if strings.Contains(rejected.raw, secret) {
		t.Fatal("submitted material returned")
	}
	fields, _ = trustKeyFields()
	badProof := strings.Repeat("ab", 64)
	fields["proof"] = badProof
	requireLicenseOutcome(t, f.run(root, "licenses.propose_key", licenseTarget, fields), licensemode.ReasonNoProof)
	f.must(filepath.WalkDir(f.s.cfg.Data, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte(secret)) || bytes.Contains(data, []byte(badProof)) {
			t.Errorf("submitted material retained in %s", filepath.Base(path))
		}
		return nil
	}))
}
func TestLicenseTrustRetirementRechecksAuthority(t *testing.T) {
	authority := &licenseTestAuthority{start, true}
	f := licenseWorld(t, authority)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, _ := trustKeyFields()
	target := licenseProposalTarget(t, f.run(root, "licenses.propose_key", licenseTarget, fields))
	requireLicenseOutcome(t, f.run(second, "licenses.approve_key_change", target, nil), licensemode.ProposalApplied)
	retirement := map[string]string{"key_id": "test-key", "at": fmt.Sprint(start.Add(time.Hour).Unix())}
	target = licenseProposalTarget(t, f.run(root, "licenses.propose_retirement", licenseTarget, retirement))
	authority.available = false
	requireLicenseOutcome(t, f.run(second, "licenses.approve_key_change", target, nil), licensemode.OutcomeTimeUnavailable)
	if r := f.licenseTrustRead(root, "keys/test-key"); r.body["record"].(map[string]any)["retired"] != false {
		t.Fatal(r.raw)
	}
	authority.available = true
	authority.now = start.Add(2 * time.Hour)
	requireLicenseOutcome(t, f.run(second, "licenses.approve_key_change", target, nil), licensemode.ReasonRetirePast)
	requireLicenseOutcome(t, f.run(root, "licenses.withdraw_key_change", target, nil), licensemode.ProposalWithdrawn)
	retirement["at"] = fmt.Sprint(start.Add(3 * time.Hour).Unix())
	target = licenseProposalTarget(t, f.run(root, "licenses.propose_retirement", licenseTarget, retirement))
	requireLicenseOutcome(t, f.run(second, "licenses.approve_key_change", target, nil), licensemode.ProposalApplied)
	if r := f.licenseTrustRead(root, "keys/test-key"); r.body["record"].(map[string]any)["retired"] != true {
		t.Fatal(r.raw)
	}
	for _, raw := range []string{"0", "01", "-1", "253402300800"} {
		if _, ok := licenseUnix(raw); ok {
			t.Fatal(raw)
		}
	}
}
func TestLicenseTrustStorageFailureIsUnknown(t *testing.T) {
	f := licenseWorld(t, nil)
	root := f.as(AdminLogin)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, _ := trustKeyFields()
	f.must(os.Remove(filepath.Join(f.s.licenseDir(), "state.json")))
	r := f.run(root, "licenses.propose_key", licenseTarget, fields)
	f.code(t, "storage failure", r, 500, "internal_error")
	if r.body["next_action"] != "inspect_operation" {
		t.Fatal(r.raw)
	}
	f.code(t, "unavailable read", f.licenseTrustRead(root, "key-changes/1"), 503, "dependency_unavailable")
}

func TestLicenseTrustHoldAllowsOnlyWithdrawal(t *testing.T) {
	f := licenseWorld(t, nil)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, _ := trustKeyFields()
	target := licenseProposalTarget(t, f.run(root, "licenses.propose_key", licenseTarget, fields))
	f.s.domain.Lock()
	f.s.enterHold("commit_unknown")
	f.s.domain.Unlock()
	f.code(t, "held approval", f.run(second, "licenses.approve_key_change", target, nil), 503, "dependency_unavailable")
	f.code(t, "held other withdrawal", f.run(second, "licenses.withdraw_key_change", target, nil), 403, "permission_denied")
	requireLicenseOutcome(t, f.run(root, "licenses.withdraw_key_change", target, nil), licensemode.ProposalWithdrawn)
}
