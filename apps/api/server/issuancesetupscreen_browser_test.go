package server

import (
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestIssuanceSetupActualScreenBrowser(t *testing.T) {
	output := os.Getenv("HB_ISSUANCE_SETUP_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated browser configuration required")
	}
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	j := confirmJourneyConfiguration(t, f)
	c := j.configuration
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
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"), "--contract", c.Spec.Contract, "--on", "2026-11-16")
	cmd.Env = os.Environ()
	for name, login := range map[string]string{"FIRST": AdminLogin, "SECOND": Admin2Login, "THIRD": Admin3Login, "MEMBER": MemberLogin} {
		who := f.as(login)
		cmd.Env = append(cmd.Env, "HB_ISSUANCE_SETUP_BROWSER_"+name+"="+who.cookie.Name+"="+who.cookie.Value)
	}
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual issuance setup browser failed; inspect isolated evidence: %v\n%s", err, raw)
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
}
