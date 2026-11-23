package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDocumentActualScreenBrowser(t *testing.T) {
	output := os.Getenv("HB_DOCUMENT_SCREEN_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated client configuration required")
	}
	w := explicitDocumentWorld(t)
	w.a = w.f.s.documentERP
	if w.a == nil || len(w.a.documents) != 0 || len(w.a.owners) != 0 {
		t.Fatal("unexpected seeded document effects")
	}
	f := w.f
	mode := os.Getenv("HB_DOCUMENT_SCREEN_QA_MODE")
	if mode == "reauth" {
		f.setPassword(Admin2Login, pw)
		w.actors[Admin2Login] = f.signedIn(Admin2Login, pw)
		w.a.afterPrepare = func() error { f.advance(16 * time.Minute); return nil }
	}
	if mode == "revoke" {
		w.a.afterPrepare = func() error { f.s.enterHold("synthetic browser hold"); return nil }
	}
	if mode == "permission" {
		w.a.afterPrepare = func() error {
			for _, grant := range f.s.admin.Grants() {
				if grant.SubjectID == SampleAdmin2 && grant.Terms.Action == gate.PrepareERPRecord {
					if err := f.s.admin.RevokeGrant(grant.ID, grant.Version); err != nil {
						return err
					}
				}
			}
			return errors.New("synthetic permission-loss uncertainty")
		}
	}
	fresh := w.actors[Admin2Login]
	if mode == "recovery" {
		fresh = f.as(Admin2Login)
		lost := false
		w.a.afterPrepare = func() error {
			if !lost {
				lost = true
				return errors.New("synthetic receipt interruption")
			}
			return nil
		}
	}
	f.s.origins[os.Getenv("HB_PROJECTION_QA_URL")] = true
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs("../../../tools/document-screen-qa.mjs")
	f.must(err)
	proposer, confirmer := w.actors[Admin2Login], w.actors[Admin3Login]
	cmd := exec.Command("node", script, "--mode", mode, "--url", os.Getenv("HB_PROJECTION_QA_URL"), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"), "--api", server.URL, "--output", output, "--request", w.basis.Request, "--digest", w.basis.RequestDigest)
	cmd.Env = append(os.Environ(), "HB_DOCUMENT_BROWSER_PASSWORD="+pw, "HB_DOCUMENT_BROWSER_FRESH="+fresh.cookie.Name+"="+fresh.cookie.Value, "HB_DOCUMENT_CLIENT_PROPOSER="+proposer.cookie.Name+"="+proposer.cookie.Value, "HB_DOCUMENT_CLIENT_CONFIRMER="+confirmer.cookie.Name+"="+confirmer.cookie.Value)
	if err = cmd.Run(); err != nil {
		t.Fatal("actual client failed; inspect non-sensitive evidence", err)
	}
	if len(w.a.owners) != 1 || len(w.a.documents) != 1 {
		t.Fatal("unexpected document or preparation count")
	}
	for key := range w.a.owners {
		actual, err := w.a.ledger.SubmissionOf(SampleWorkspace, key[1])
		f.must(err)
		if key[0] != SampleWorkspace || actual.Total.Net != "15.00" || actual.Total.Tax != "1.00" || actual.Total.Quantity != "5.00" || len(actual.Commands) != 0 {
			t.Fatal("unexpected screen financial effects")
		}
	}
	expected := "confirmed"
	if mode == "revoke" {
		expected = "revoked"
	}
	for key, document := range w.a.documents {
		if key[0] != SampleWorkspace || document.review.State != expected {
			t.Fatal("independent screen confirmation absent")
		}
	}
}
