package server

import (
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDocumentEvidenceImpactRenderedBrowser(t *testing.T) {
	output := os.Getenv("HB_DOCUMENT_IMPACT_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated browser configuration required")
	}
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	w, _ := publicDocument(t)
	f := w.f
	for _, id := range []string{"confirmed-document", "revoked-document"} {
		p := f.run(w.actors[Admin2Login], "document_evidence.propose", id, map[string]string{"packet": packetJSON(t, packetFromBasis(w.basis))})
		f.ok(t, "proposal", p, "pending", "")
		command, state := "document_evidence.confirm", "confirmed"
		if id == "revoked-document" {
			command, state = "document_evidence.revoke", "revoked"
		}
		f.ok(t, state, f.run(w.actors[Admin3Login], command, id, map[string]string{"digest": field(p, "digest")}), state, "")
	}
	counts, err := documentEvidenceTally(f.s)
	f.must(err)
	if counts[SampleAdmin2] != 1 {
		t.Fatal("terminal packets counted", counts)
	}
	for _, id := range []string{"impact-permission-one", "impact-permission-two"} {
		fields := documentPermissionTimes(f)
		fields["capability"] = "prepare"
		f.ok(t, "pending permission", f.run(w.actors[Admin2Login], "document_authority.propose_console", id, fields), "pending", "")
	}
	f.setPassword(AdminLogin, pw)
	reader := f.signedIn(AdminLogin, pw)
	staged := f.stage(reader, "", SampleAdmin2)
	f.ok(t, "stage", staged, "draft", "")
	prepared := f.prepare(reader, field(staged, "draft"), "suspend_membership")
	f.ok(t, "preview only", prepared, "prepared", "")
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
	script, err := filepath.Abs("../../../tools/document-impact-qa.mjs")
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--bulk", field(prepared, "bulk"), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Env = append(os.Environ(), "HB_PROJECTION_QA_COOKIE="+reader.cookie.Name+"="+reader.cookie.Value)
	if err := cmd.Run(); err != nil {
		t.Fatal("isolated browser assertions failed; inspect non-sensitive evidence", err)
	}
	counts, err = documentEvidenceTally(f.s)
	f.must(err)
	if counts[SampleAdmin2] != 1 || len(f.s.documentERP.owners) != 0 {
		t.Fatal("browser changed financial state")
	}
}
