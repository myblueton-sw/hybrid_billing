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
)

func TestResultAuthorityImpactRenderedBrowser(t *testing.T) {
	output := os.Getenv("HB_RESULT_IMPACT_QA_OUTPUT")
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
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	for _, id := range []string{"result-pending-one", "result-pending-two", "result-applied"} {
		p := f.run(second, "result_authority.propose_console", id, resultPermissionTimes(f))
		f.ok(t, "result proposal", p, "pending", "")
		if id == "result-applied" {
			f.ok(t, "applied", resultPermissionApprove(f, root, id, p), "applied", "")
		}
	}
	for _, id := range []string{"document-pending-one", "document-pending-two"} {
		f.ok(t, "document proposal", f.run(second, "document_authority.propose_console", id, documentPermissionTimes(f)), "pending", "")
	}
	counts, err := resultAuthorityTally(f.s)
	f.must(err)
	if counts[SampleAdmin2] != 2 {
		t.Fatal("terminal grant counted")
	}
	grants := []map[string]string{}
	for _, capability := range []string{"propose", "confirm", "revoke", "apply", "inspect"} {
		fields := resultPermissionTimes(f)
		fields["capability"] = capability
		cp := f.run(root, "result_authority.propose_console", "display-console-"+capability, fields)
		f.ok(t, "console proposal", cp, "pending", "")
		ceiling := resultPermissionApprove(f, second, "display-console-"+capability, cp)
		f.ok(t, "console grant", ceiling, "applied", "")
		fields["recipient"], fields["ceiling"] = SampleMember, field(ceiling, "grant")
		hp := f.run(root, "result_authority.propose_human", "display-human-"+capability, fields)
		f.ok(t, "human proposal", hp, "pending", "")
		human := resultPermissionApprove(f, second, "display-human-"+capability, hp)
		f.ok(t, "human grant", human, "applied", "")
		grants = append(grants, map[string]string{"capability": capability, "human": field(human, "grant"), "console": field(ceiling, "grant")})
	}
	grantJSON, err := json.Marshal(grants)
	f.must(err)
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
	script, err := filepath.Abs("../../../tools/result-impact-qa.mjs")
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--bulk", field(prepared, "bulk"), "--grants", string(grantJSON), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Env = append(os.Environ(), "HB_PROJECTION_QA_COOKIE="+reader.cookie.Name+"="+reader.cookie.Value)
	if err := cmd.Run(); err != nil {
		t.Fatal("isolated browser assertions failed; inspect non-sensitive evidence", err)
	}
	counts, err = resultAuthorityTally(f.s)
	f.must(err)
	if counts[SampleAdmin2] != 2 || len(f.s.admin.Grants()) != beforeGrants || len(f.s.documentERP.owners) != 0 {
		t.Fatal("browser changed grant or ERP state")
	}
}
