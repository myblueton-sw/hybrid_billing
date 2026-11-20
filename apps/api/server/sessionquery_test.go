package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func (f world) sessionQuery(a actor, query string) reply {
	f.t.Helper()
	return f.do(http.MethodGet, "/api/v1/me/sessions"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func TestSessionQueryPagesOwnOtherSessionsWithoutSecrets(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	older := f.signedIn(MemberLogin, pw)
	f.advance(time.Second)
	newer := f.signedIn(MemberLogin, pw)
	f.advance(time.Second)
	current := f.signedIn(MemberLogin, pw)
	first := f.sessionQuery(current, "?session=other&limit=1")
	if first.status != http.StatusOK {
		t.Fatalf("first query status %d, want200", first.status)
	}
	rows, ok := first.body["rows"].([]any)
	if !ok || len(rows) != 1 || rows[0].(map[string]any)["handle"] != handleOfActor(newer) {
		t.Fatalf("first bounded rows %v", first.body["rows"])
	}
	next, ok := first.body["next"].(string)
	if !ok || next == "" {
		t.Fatal("continuation missing for another matching session")
	}
	second := f.sessionQuery(current, "?session=other&limit=1&after="+url.QueryEscape(next))
	if second.status != http.StatusOK {
		t.Fatalf("second query status %d, want200", second.status)
	}
	rows, ok = second.body["rows"].([]any)
	if !ok || len(rows) != 1 || rows[0].(map[string]any)["handle"] != handleOfActor(older) {
		t.Fatalf("second bounded rows %v", second.body["rows"])
	}
	for _, value := range []string{older.cookie.Value, newer.cookie.Value, current.cookie.Value, older.csrf, newer.csrf, current.csrf} {
		if strings.Contains(first.raw, value) || strings.Contains(second.raw, value) {
			t.Fatal("session query exposed authentication material")
		}
	}
}

func TestSessionQueryRequiresCurrentAuthentication(t *testing.T) {
	f := setup(t, SeedSample).ready()
	got := f.sessionQuery(actor{}, "?limit=1")
	if got.status != http.StatusUnauthorized {
		t.Fatalf("anonymous status %d, want401", got.status)
	}
}

func TestSessionQueryRejectsForeignWorkspaceBeforeDisclosure(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	got := f.sessionQuery(a, "?workspace=foreign-workspace&limit=1")
	if got.status != http.StatusUnprocessableEntity {
		t.Fatalf("foreign workspace status %d, want422", got.status)
	}
	if got.body["rows"] != nil || strings.Contains(got.raw, "\"count\"") {
		t.Fatal("rejected workspace disclosed list data")
	}
}
