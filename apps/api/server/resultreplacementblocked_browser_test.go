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
	"time"
)

func TestResultReplacementBlockedBrowser(t *testing.T) {
	output := os.Getenv("HB_RESULT_REPLACEMENT_BLOCKED_OUTPUT")
	if output == "" {
		t.Skip("explicit isolated blocked replacement configuration required")
	}
	frontend := os.Getenv("HB_PROJECTION_QA_URL")
	u, err := url.Parse(frontend)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		t.Fatal("loopback frontend required")
	}
	w, p := resultWorld(t)
	f := w.f
	f.s.documentERP = w.a
	first := confirmResult(t, w, proposeResult(t, w, p, "unknown-predecessor"))
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted predecessor") }
	_, err = applyResult(w, first)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	w.a.afterResultWrite = nil
	p.Revision = 2
	confirmResult(t, w, proposeResult(t, w, p, "blocked-successor"))
	f.s.origins[frontend] = true
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	f.must(err)
	f.must(f.s.Bind(listener.Addr()))
	server := httptest.NewUnstartedServer(f.h)
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	defer server.Close()
	script, err := filepath.Abs("../../../tools/result-replacement-blocked-qa.mjs")
	f.must(err)
	actor := w.actors[Admin2Login]
	for _, locale := range []string{"en", "ko", "ar"} {
		for _, width := range []int{1440, 390} {
			f.advance(time.Minute)
			name := locale + "-" + strconv.Itoa(width)
			cmd := exec.Command("node", script, "--url", frontend, "--api", server.URL, "--output", filepath.Join(output, name), "--playwright-module", os.Getenv("HB_PROJECTION_QA_PLAYWRIGHT"), "--chromium", os.Getenv("HB_PROJECTION_QA_CHROMIUM"))
			cmd.Env = append(os.Environ(), "HB_RESULT_BLOCKED_COOKIE="+actor.cookie.Name+"="+actor.cookie.Value, "HB_RESULT_BLOCKED_LOCALE="+locale, "HB_RESULT_BLOCKED_WIDTH="+strconv.Itoa(width))
			if err = cmd.Run(); err != nil {
				t.Fatal("blocked replacement browser failed; inspect isolated evidence", name, err)
			}
		}
	}
	if len(w.a.resultApplications) != 1 || w.a.resultApplications[[2]string{SampleWorkspace, "unknown-predecessor"}].receipt != "" {
		t.Fatal("read-only browser changed unknown ownership")
	}
}
