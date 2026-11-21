package server

import (
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
	f, requester, decider, digest := chargeWorld(t)
	for _, id := range []string{"qa-superseded", "qa-rejected", "qa-current"} {
		p := chargeRequest(f, requester, id, digest)
		f.ok(t, "request", p, approvals.StatePending, "")
		if id == "qa-rejected" {
			f.ok(t, "reject", f.run(decider, "financial_charge.reject", id, map[string]string{"digest": field(p, "digest"), "reason": "other"}), approvals.StateRejected, "")
		}
	}
	f.setPassword(AdminLogin, pw)
	reader := f.signedIn(AdminLogin, pw)
	f.s.domain.Lock()
	f.s.origins[frontend] = true
	f.s.domain.Unlock()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs(filepath.Join("../../../tools", runner))
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Args = append(cmd.Args, extra...)
	cmd.Env = append(os.Environ(), "HB_PROJECTION_QA_COOKIE="+reader.cookie.Name+"="+reader.cookie.Value, "HB_PROJECTION_QA_PASSWORD="+pw)
	if err := cmd.Run(); err != nil {
		t.Fatal("isolated browser assertions failed; inspect non-sensitive evidence", err)
	}
}
