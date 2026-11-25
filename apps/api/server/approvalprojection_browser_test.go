package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func TestApprovalProjectionRenderedBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "approval-projection-qa.mjs")
}

func TestFinancialDecisionCoreBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "financial-decision-core-qa.mjs")
}

func TestFinancialDecisionScreenBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "financial-decision-screen-qa.mjs")
}

func TestFinancialDecisionRejectionScreenBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "financial-decision-screen-qa.mjs", "--flow", "reject")
}

func TestFinancialFinalRequestScreenBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "financial-final-request-screen-qa.mjs", "--family", "erp_final", "--requester", "true")
}

func TestFinancialFinalDecisionScreenBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "financial-decision-screen-qa.mjs", "--family", "erp_final")
}

func TestFinancialFinalRejectionScreenBrowser(t *testing.T) {
	runApprovalProjectionBrowser(t, "financial-decision-screen-qa.mjs", "--family", "erp_final", "--flow", "reject")
}

func runApprovalProjectionBrowser(t *testing.T, runner string, extra ...string) {
	t.Helper()
	output := os.Getenv("HB_PROJECTION_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated browser configuration required")
	}
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	var f world
	var requester, decider actor
	var digest string
	family := "financial_charge"
	if slices.Contains(extra, "erp_final") {
		f, requester, decider, digest = finalCommandWorld(t)
		family = "erp_final"
	} else {
		f, requester, decider, digest = chargeWorld(t)
	}
	for _, id := range []string{"qa-superseded", "qa-rejected", "qa-current"} {
		var p reply
		if family == "erp_final" {
			p = finalRequest(f, requester, id, digest)
		} else {
			p = chargeRequest(f, requester, id, digest)
		}
		f.ok(t, "request", p, approvals.StatePending, "")
		if id == "qa-rejected" {
			f.ok(t, "reject", f.run(decider, family+".reject", id, map[string]string{"digest": field(p, "digest"), "reason": "other"}), approvals.StateRejected, "")
		}
	}
	login := AdminLogin
	if slices.Contains(extra, "--requester") {
		login = Admin3Login
	}
	f.setPassword(login, pw)
	reader := f.signedIn(login, pw)
	f.s.domain.Lock()
	f.s.origins[frontend] = true
	f.s.domain.Unlock()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	handler := f.h
	if slices.Contains(extra, "--requester") {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.advance(time.Second); f.h.ServeHTTP(w, r) })
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs(filepath.Join("../../../tools", runner))
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Args = append(cmd.Args, extra...)
	cmd.Env = append(os.Environ(), "HB_PROJECTION_QA_COOKIE="+reader.cookie.Name+"="+reader.cookie.Value, "HB_PROJECTION_QA_PASSWORD="+pw)
	initialFinals := len(f.s.financialFinals)
	if err := cmd.Run(); err != nil {
		t.Fatal("isolated browser assertions failed; inspect non-sensitive evidence", err)
	}
	if slices.Contains(extra, "--requester") {
		record, binding, err := f.s.financialFinalRecord(SampleWorkspace, "qa-browser-final")
		f.must(err)
		if len(f.s.financialFinals) != initialFinals+1 || record.State != approvals.StatePending || len(record.Decisions) != 0 || record.Subject.Amount != "16" || binding.Requester != SampleAdmin3 {
			t.Fatal("request journey must create exactly one pending gross request without a decision")
		}
	}
}
