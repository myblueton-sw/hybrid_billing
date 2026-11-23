package server

import (
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestERPFinalPermissionGrantScreens(t *testing.T) {
	output := os.Getenv("HB_FINAL_PERMISSION_QA_OUTPUT")
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
	first, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(first, "erp_final_permission.propose", "decision-screen", financialPermissionFields(f))
	f.ok(t, "decision proposal", p, "pending", "")
	decision := finalPermissionApprove(t, f, second, "erp_final_permission", "decision-screen", p)
	p = f.run(first, "erp_final_request_permission.propose_console", "console-screen", requestPermissionTimes(f))
	f.ok(t, "console proposal", p, "pending", "")
	console := finalPermissionApprove(t, f, second, "erp_final_request_permission", "console-screen", p)
	fields := requestPermissionTimes(f)
	fields["recipient"] = SampleMember
	fields["ceiling"] = field(console, "grant")
	p = f.run(first, "erp_final_request_permission.propose_human", "human-screen", fields)
	f.ok(t, "human proposal", p, "pending", "")
	human := finalPermissionApprove(t, f, second, "erp_final_request_permission", "human-screen", p)
	f.advance(time.Minute)
	count := len(f.s.admin.Grants())
	f.s.origins[frontend] = true
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs("../../../tools/erp-final-permission-grants-qa.mjs")
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--decision", field(decision, "grant"), "--human", field(human, "grant"), "--console", field(console, "grant"), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Env = append(os.Environ(), "HB_PROJECTION_QA_COOKIE="+first.cookie.Name+"="+first.cookie.Value)
	if err := cmd.Run(); err != nil {
		t.Fatal("isolated grant screen assertions failed; inspect non-sensitive evidence", err)
	}
	if len(f.s.admin.Grants()) != count {
		t.Fatal("read-only browser changed grants")
	}
}
