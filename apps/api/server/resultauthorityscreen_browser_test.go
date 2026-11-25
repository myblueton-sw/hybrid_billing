package server

import (
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
)

func TestResultAuthorityActualScreenBrowser(t *testing.T) {
	output := os.Getenv("HB_RESULT_AUTHORITY_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated authority browser configuration required")
	}
	for _, locale := range []string{"en", "ko", "ar"} {
		for _, width := range []int{1440, 1024, 390} {
			t.Run(locale+"-"+strconv.Itoa(width), func(t *testing.T) {
				resultAuthorityActualScreenBrowser(t, filepath.Join(output, locale+"-"+strconv.Itoa(width)), locale, width)
			})
		}
	}
}
func resultAuthorityActualScreenBrowser(t *testing.T, output, locale string, width int) {
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	f.advance(500 * time.Millisecond)
	before := len(f.s.admin.Grants())
	grantWrites := 0
	f.s.fault = func(at string) bool {
		if at != "result-authority-after-grant" {
			return false
		}
		grantWrites++
		return grantWrites == 2
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
	script, err := filepath.Abs("../../../tools/result-authority-live-qa.mjs")
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Env = append(os.Environ(), "HB_AUTHORITY_LOCALE="+locale, "HB_AUTHORITY_WIDTH="+strconv.Itoa(width), "HB_AUTHORITY_PROPOSER="+root.cookie.Name+"="+root.cookie.Value, "HB_AUTHORITY_APPROVER="+second.cookie.Name+"="+second.cookie.Value, "HB_AUTHORITY_FROM="+f.s.clock().Format(time.RFC3339Nano), "HB_AUTHORITY_UNTIL="+f.s.clock().Add(12*time.Hour).Format(time.RFC3339Nano))
	if err = cmd.Run(); err != nil {
		t.Fatal("actual authority browser failed; inspect isolated evidence", err)
	}
	if grantWrites != 2 || len(f.s.admin.Grants()) != before+2 {
		t.Fatal("unexpected granted effects")
	}
	f.must(inputAuthorityAs(f, member, func() error {
		request := gate.Request{Action: gate.ApplyERPResult, Kind: resultauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, request).Allowed {
			t.Fatal("provisioned result capability unavailable")
		}
		request.Action = gate.Approve
		if f.s.gate.CheckApproval(f.s.answer, SampleAdmin, request).Allowed {
			t.Fatal("provisioning enabled financial approval")
		}
		return nil
	}))
}
