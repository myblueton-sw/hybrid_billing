package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
)

func TestResultActualScreenBrowser(t *testing.T) {
	output := os.Getenv("HB_RESULT_SCREEN_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated browser configuration required")
	}
	for _, locale := range []string{"en", "ko", "ar"} {
		for _, width := range []int{1440, 1024, 390} {
			t.Run(locale+"-"+strconv.Itoa(width), func(t *testing.T) {
				resultActualScreenBrowser(t, filepath.Join(output, locale+"-"+strconv.Itoa(width)), locale, width, false)
			})
		}
	}
}

func TestResultReplacementScreenBrowser(t *testing.T) {
	output := os.Getenv("HB_RESULT_REPLACEMENT_QA_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated replacement browser configuration required")
	}
	for _, locale := range []string{"en", "ko", "ar"} {
		for _, width := range []int{1440, 1024, 390} {
			t.Run(locale+"-"+strconv.Itoa(width), func(t *testing.T) {
				resultActualScreenBrowser(t, filepath.Join(output, locale+"-"+strconv.Itoa(width)), locale, width, true)
			})
		}
	}
}

func resultActualScreenBrowser(t *testing.T, output, locale string, width int, replacement bool) {
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	w, packet := resultWorld(t)
	f := w.f
	f.s.documentERP = w.a
	if replacement {
		previous := confirmResult(t, w, proposeResult(t, w, packet, "prior-recorded-result"))
		_, err := applyResult(w, previous)
		f.must(err)
		packet.Revision = 2
		for i := range packet.Rows {
			packet.Rows[i].ID = "replacement-" + packet.Rows[i].ID
		}
	}
	f.advance(time.Minute)
	interrupted := false
	w.a.afterResultWrite = func(string) error {
		if !interrupted {
			interrupted = true
			return errors.New("synthetic browser result interruption")
		}
		return nil
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
	script, err := filepath.Abs("../../../tools/result-screen-live-qa.mjs")
	f.must(err)
	proposer, confirmer := w.actors[Admin2Login], w.actors[Admin3Login]
	raw, err := json.Marshal(packet)
	f.must(err)
	cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", output, "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
	cmd.Env = append(os.Environ(), "HB_RESULT_BROWSER_LOCALE="+locale, "HB_RESULT_BROWSER_WIDTH="+strconv.Itoa(width), "HB_RESULT_BROWSER_PACKET="+string(raw), "HB_RESULT_BROWSER_PROPOSER="+proposer.cookie.Name+"="+proposer.cookie.Value, "HB_RESULT_BROWSER_CONFIRMER="+confirmer.cookie.Name+"="+confirmer.cookie.Value)
	if replacement {
		cmd.Env = append(cmd.Env, "HB_RESULT_BROWSER_REPLACEMENT=prior-recorded-result")
	}
	if err = cmd.Run(); err != nil {
		t.Fatal("actual result browser failed; inspect isolated evidence", err)
	}
	expected := 1
	if replacement {
		expected = 2
	}
	if !interrupted || len(w.a.results) != expected || len(w.a.resultApplications) != expected {
		t.Fatal("unexpected result effects")
	}
	for key, result := range w.a.results {
		app := w.a.resultApplications[key]
		if result.review.State != "confirmed" || result.review.Proposer != SampleAdmin2 || result.review.Confirmation.Actor != SampleAdmin3 || app.actor != SampleAdmin2 || !validPlanHash(app.receipt) {
			t.Fatal("missing independent confirmation or recovered receipt")
		}
	}
	snapshot, err := w.a.ledger.ResultOf(SampleWorkspace, packet.Preparation)
	f.must(err)
	if !snapshot.Reconciliation.Matched || len(snapshot.Documents) != 1 || len(snapshot.Documents[0].Links) != 2*expected || snapshot.Documents[0].Total != (erpledger.Amounts{Net: "15.00", Tax: "1.00", Quantity: "5.00"}) || snapshot.Submission.State != erpledger.StatePrepared || len(snapshot.Submission.Commands) != 0 {
		t.Fatal("unexpected external command or document count")
	}
	if replacement {
		if snapshot.Documents[0].Document.Revision != 2 {
			t.Fatal("replacement revision lost")
		}
		for i, link := range snapshot.Documents[0].Links {
			if link.Stale != (i < 2) {
				t.Fatal("replacement stale history lost")
			}
		}
	}

}
