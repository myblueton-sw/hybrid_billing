package server

import (
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDocumentActualTypeScriptClient(t *testing.T) {
	output := os.Getenv("HB_DOCUMENT_CLIENT_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated client configuration required")
	}
	w, _ := publicDocument(t)
	f := w.f
	f.s.origins["http://127.0.0.1:8896"] = true
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs("../../../tools/document-client-qa.mts")
	f.must(err)
	runner, err := filepath.Abs("../../console/node_modules/tsx/dist/cli.mjs")
	f.must(err)
	proposer, confirmer := w.actors[Admin2Login], w.actors[Admin3Login]
	cmd := exec.Command("node", runner, script, "--api", server.URL, "--output", output, "--request", w.basis.Request, "--digest", w.basis.RequestDigest)
	cmd.Env = append(os.Environ(), "HB_DOCUMENT_CLIENT_PROPOSER="+proposer.cookie.Name+"="+proposer.cookie.Value, "HB_DOCUMENT_CLIENT_CONFIRMER="+confirmer.cookie.Name+"="+confirmer.cookie.Value)
	if err = cmd.Run(); err != nil {
		t.Fatal("actual client failed; inspect non-sensitive evidence", err)
	}
	actual, err := w.a.ledger.SubmissionOf(SampleWorkspace, "client-preparation")
	f.must(err)
	if actual.Total.Net != "15.00" || actual.Total.Tax != "1.00" || actual.Total.Quantity != "5.00" || len(actual.Commands) != 0 || len(w.a.owners) != 1 {
		t.Fatal("unexpected client effects")
	}
	if w.a.documents[[2]string{SampleWorkspace, "client-document"}].review.State != "revoked" {
		t.Fatal("client revocation absent")
	}
}
