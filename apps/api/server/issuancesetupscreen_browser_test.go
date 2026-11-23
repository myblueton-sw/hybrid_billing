package server

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"

	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestIssuanceSetupActualScreenBrowser(t *testing.T) {
	output := os.Getenv("HB_ISSUANCE_SETUP_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated browser configuration required")
	}
	for _, lang := range []string{"en", "ko", "ar"} {
		t.Run(lang, func(t *testing.T) { issuanceSetupActualScreenBrowser(t, filepath.Join(output, lang), lang) })
	}
}

func issuanceSetupActualScreenBrowser(t *testing.T, output, lang string) {
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	f := setupWith(t, func(c *Config) {
		c.Fields = map[string]gate.FieldClass{"state": gate.General}
		c.Commands = CommandLimits()
		c.Commands.Burst = 10 * c.Commands.Burst
	})
	f.setPassword(MemberLogin, pw)
	j := confirmJourneyConfiguration(t, f)
	c := j.configuration
	requester, verifier := activateScreenAdministrator(f, "screen-requester"), activateScreenAdministrator(f, "screen-verifier")
	for _, right := range []struct{ namespace, capability, recipient string }{
		{"issuance_protected_authority", protectedauthority.Inspect, SampleAdmin3},
		{"issuance_protected_authority", protectedauthority.RegisterChannel, SampleAdmin3},
		{"issuance_protected_authority", protectedauthority.AdmitChannel, SampleAdmin},
		{"issuance_term_authority", termsauthority.Inspect, SampleAdmin3},
		{"issuance_term_authority", termsauthority.Inspect, SampleAdmin},
		{"issuance_term_authority", termsauthority.RecordTerm, SampleAdmin3},
		{"issuance_term_authority", termsauthority.ApproveTerm, SampleAdmin},
	} {
		j.grant(t, f, right.namespace, right.capability, right.recipient)
	}
	a := f.s.financialIssuance
	if a.protectedRules != nil || len(a.channels) != 0 || len(a.termIntents) != 0 || len(a.attestations) != 0 {
		t.Fatal("the screen run starts with issuance records")
	}
	f.s.origins[frontend] = true
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs("../../../tools/o03-setup-live-qa.mjs")
	f.must(err)
	browse := func(phase string, people map[string]string, extra ...string) {
		t.Helper()
		arguments := append([]string{script, "--phase", phase, "--lang", lang, "--url", frontend, "--api", server.URL, "--output", filepath.Join(output, phase), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"), "--contract", c.Spec.Contract, "--on", "2026-11-16"}, extra...)
		cmd := exec.Command("node", arguments...)
		cmd.Env = os.Environ()
		for name, login := range people {
			who := f.as(login)
			cmd.Env = append(cmd.Env, "HB_ISSUANCE_SETUP_BROWSER_"+name+"="+who.cookie.Name+"="+who.cookie.Value)
		}
		if raw, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("actual issuance setup browser failed in phase %s; inspect isolated evidence: %v\n%s", phase, err, raw)
		}
	}
	interrupted := false
	f.s.fault = func(at string) bool {
		if at == "issuance-channel-after-admit" && !interrupted {
			interrupted = true
			return true
		}
		return false
	}
	browse("setup", map[string]string{"FIRST": AdminLogin, "SECOND": Admin2Login, "THIRD": Admin3Login, "MEMBER": MemberLogin})
	f.s.fault = nil
	if !interrupted {
		t.Fatal("the channel acceptance was never interrupted")
	}
	rules := a.protectedRules
	if rules == nil || rules.State != "confirmed" || rules.Proposer != SampleAdmin || rules.Confirmer != SampleAdmin2 {
		t.Fatal("rules were not proposed and confirmed by the two browser sessions")
	}
	if len(a.channels) != 1 || len(a.termIntents) != 1 || len(a.attestations) != 2 {
		t.Fatalf("unexpected records: channels %d terms %d statements %d", len(a.channels), len(a.termIntents), len(a.attestations))
	}
	for _, channel := range a.channels {
		if channel.State != "admitted" || channel.Registrar != SampleAdmin3 || channel.Admitter != SampleAdmin || channel.AdmissionAttestation == "" {
			t.Fatal("the channel was not registered and accepted by the two browser sessions")
		}
	}
	for _, term := range a.termIntents {
		if term.State != "approved" || term.Author != SampleAdmin3 || term.Approver != SampleAdmin || term.Spec.NetDays != 45 {
			t.Fatal("the term was not recorded and approved by the two browser sessions")
		}
	}

	f.advance(24 * time.Hour)
	f.must(f.s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}))
	for _, right := range []struct{ namespace, capability, recipient string }{
		{"issuance_protected_authority", protectedauthority.Inspect, "screen-requester"},
		{"issuance_protected_authority", protectedauthority.Inspect, "screen-verifier"},
		{"issuance_protected_authority", protectedauthority.Inspect, SampleAdmin2},
		{"issuance_protected_authority", protectedauthority.Inspect, SampleAdmin3},
		{"issuance_protected_authority", protectedauthority.Inspect, SampleAdmin},
		{"issuance_protected_authority", protectedauthority.RegisterChannel, SampleAdmin3},
		{"issuance_protected_authority", protectedauthority.AdmitChannel, SampleAdmin},
		{"issuance_protected_authority", protectedauthority.RequestChange, "screen-requester"},
		{"issuance_protected_authority", protectedauthority.VerifyChange, "screen-verifier"},
		{"issuance_protected_authority", protectedauthority.ApproveChange, SampleAdmin2},
		{"issuance_term_authority", termsauthority.Inspect, SampleAdmin3},
		{"issuance_term_authority", termsauthority.Inspect, SampleAdmin},
		{"issuance_term_authority", termsauthority.RecordTerm, SampleAdmin3},
		{"issuance_term_authority", termsauthority.ApproveTerm, SampleAdmin},
		{"issuance_term_authority", termsauthority.BindTerm, SampleAdmin3},
		{"issuance_term_authority", termsauthority.ApproveBinding, SampleAdmin},
	} {
		j.grant(t, f, right.namespace, right.capability, right.recipient)
	}
	f.ok(t, "unused statement", attestRun(f, f.as(Admin3Login), "attest_channel_registration", c, map[string]string{"attestation": "screen-unused-statement", "target": "screen-unused-channel", "statement": "Synthetic statement that no step will use."}), "recorded", "")
	f.must(f.s.Bind(listener.Addr()))
	browse("changes", map[string]string{"FIRST": AdminLogin, "SECOND": Admin2Login, "THIRD": Admin3Login, "REQUESTER": requester, "VERIFIER": verifier}, "--effective", f.s.clock().Add(2*time.Hour).UTC().Format("2006-01-02T15:04"))
	if len(a.changes) != 3 || len(a.bindingIntents) != 1 || len(a.attestations) != 9 {
		t.Fatalf("unexpected records: changes %d bindings %d statements %d", len(a.changes), len(a.bindingIntents), len(a.attestations))
	}
	states := map[string]int{}
	for _, change := range a.changes {
		v := change.Module
		if v.Requester != "screen-requester" || v.Verification.Verifier != "screen-verifier" || change.Spec.Field != protected.FieldPaymentAccount {
			t.Fatalf("a change was not requested and verified by the browser sessions: %+v", v)
		}
		states[v.State]++
	}
	if states[protected.StateRejected] != 2 || states[protected.StateScheduled] != 1 {
		t.Fatalf("expected two rejected and one approved change: %v", states)
	}
	ended := map[string]int{}
	for _, change := range a.changes {
		if change.Module.State == protected.StateRejected {
			ended[change.Module.EndedBy+" "+change.Module.Verification.Verification.Outcome]++
		}
	}
	if ended["screen-verifier "+protected.OutcomeDenied] != 1 || ended[SampleAdmin2+" "+protected.OutcomeConfirmed] != 1 {
		t.Fatalf("expected one change ended by the verifier's denial and one by the approver's rejection: %v", ended)
	}
	unused := a.attestations["screen-unused-statement"]
	if attestationState(unused) != "withdrawn" || unused.Actor != SampleAdmin3 {
		t.Fatal("the unused statement was not withdrawn by its author: " + attestationState(unused))
	}
	for id, statement := range a.attestations {
		if id != "screen-unused-statement" && attestationState(statement) != "consumed" {
			t.Fatal("a statement recorded through the screen is not used by its step: " + id)
		}
	}
	for _, binding := range a.bindingIntents {
		if binding.State != "approved" {
			t.Fatal("applying the terms was not approved: " + binding.State)
		}
	}
}

func activateScreenAdministrator(f world, id string) string {
	login := id + "@example.invalid"
	newPopulation(f).member(id, login, identity.Administrator)
	f.must(f.s.grantInstallation(id, f.s.clock()))
	token, err := f.s.recordLink(login, passwords.PurposeSet, f.s.clock())
	f.must(err)
	body, err := json.Marshal(map[string]string{"token": token, "password": pw})
	f.must(err)
	if reply := f.auth("redeem", string(body), f.pre()); reply.status != 204 {
		f.t.Fatalf("administrator activation: status %d", reply.status)
	}
	return login
}
