package server

import (
	"errors"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func TestResultEvidenceImpactRenderedBrowser(t *testing.T) {
	output := os.Getenv("HB_RESULT_EVIDENCE_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated browser configuration required")
	}
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	w, p, digest := publicResult(t)
	f := w.f
	pending := cloneDocumentValue(t, p)
	pending.External = "pending-impact"
	proposeResult(t, w, pending, "pending-impact")
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	f.code(t, "unknown result", f.run(w.actors[Admin2Login], "erp_result.apply", "public-result", map[string]string{"digest": digest}), 500, "internal_error")
	w.a.afterResultWrite = nil
	for prefix, stage := range map[string]string{"financial_policy": approvals.StageCharge, "erp_final_policy": approvals.StageERPFinal} {
		policy, err := f.s.approvals.PolicyOf(SampleWorkspace, stage)
		f.must(err)
		proposal := f.run(w.actors[Admin2Login], prefix+".propose", prefix+"-impact", financialFields(strconv.Itoa(policy.Revision+1)))
		f.ok(t, "pending policy impact", proposal, "pending", "")
	}
	chargePolicy, err := f.s.approvals.PolicyOf(SampleWorkspace, approvals.StageCharge)
	f.must(err)
	otherCharge := f.run(w.actors[Admin2Login], "financial_policy.propose", "second-charge-impact", financialFields(strconv.Itoa(chargePolicy.Revision+1)))
	f.ok(t, "second charge policy impact", otherCharge, "pending", "")
	finalLimit := f.run(w.actors[Admin2Login], "erp_final_authority.propose", "final-limit-impact", financialAuthorityFields(f))
	f.ok(t, "final amount limit impact", finalLimit, "pending", "")
	beforeGrants := len(f.s.admin.Grants())
	f.setPassword(AdminLogin, pw)
	reader := f.signedIn(AdminLogin, pw)
	staged := f.stage(reader, "", SampleAdmin2)
	f.ok(t, "stage", staged, "draft", "")
	prepared := f.prepare(reader, field(staged, "draft"), "suspend_membership")
	f.ok(t, "preview only", prepared, "prepared", "")
	f.s.origins[frontend] = true
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs("../../../tools/result-evidence-impact-qa.mjs")
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--bulk", field(prepared, "bulk"), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Env = append(os.Environ(), "HB_PROJECTION_QA_COOKIE="+reader.cookie.Name+"="+reader.cookie.Value)
	if err := cmd.Run(); err != nil {
		t.Fatal("isolated browser assertions failed; inspect non-sensitive evidence", err)
	}
	pendingCount, unknownCount, err := resultEvidenceCounts(f.s)
	f.must(err)
	if pendingCount[SampleAdmin2] != 1 || unknownCount[SampleAdmin2] != 1 || len(f.s.admin.Grants()) != beforeGrants {
		t.Fatal("browser changed result state")
	}
}
