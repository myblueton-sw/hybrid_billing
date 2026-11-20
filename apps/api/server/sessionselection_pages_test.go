package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func (f world) selectionRead(a actor, path string) reply {
	return f.do(http.MethodGet, "/api/v1/me/sessions/selections/"+path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func TestSessionSelectionPreviewContinuationPreservesFrozenRowsAndExpiry(t *testing.T) {
	f, old, middle, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	ref := prepared.body["selection"].(string)
	first := f.selectionRead(a, ref+"?limit=1")
	if first.status != 200 || first.body["next"] == "" {
		t.Fatal("missing preview continuation")
	}
	cursor := url.QueryEscape(first.body["next"].(string))
	if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
		t.Fatal(err)
	}
	second := f.selectionRead(a, ref+"?limit=1&after="+cursor)
	if second.status != 200 || second.body["next"] != "" {
		t.Fatal("immutable preview failed after later arrival")
	}
	seen := map[string]bool{}
	for _, page := range []reply{first, second} {
		for _, row := range page.body["rows"].([]any) {
			handle := row.(map[string]any)["handle"].(string)
			if seen[handle] {
				t.Fatal("preview duplicate")
			}
			seen[handle] = true
		}
	}
	if !seen[handleOfActor(old)] || !seen[handleOfActor(middle)] || len(seen) != 2 {
		t.Fatal("preview set widened or omitted targets")
	}
	for _, query := range []string{"?limit=2&after=" + cursor, "?limit=1&after=forged"} {
		if got := f.selectionRead(a, ref+query); got.status != 409 {
			t.Fatal("preview accepted incompatible cursor")
		}
	}
	f.advance(301 * time.Second)
	f.s.sessions.reauthenticated(verifierOfActor(a), *f.now)
	if got := f.selectionRead(a, ref); got.status != 404 || f.s.sessionRecordUnits != 0 {
		t.Fatal("expired preview retained or disclosed record")
	}
}

func TestSessionSelectionReadRoutesRejectMalformedAndUnauthorizedQueries(t *testing.T) {
	f, _, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	ref := prepared.body["selection"].(string)
	accepted := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	id := accepted.body["operation"].(string)
	for _, query := range []string{"?unknown=x", "?limit=0", "?limit=101", "?limit=no", "?limit=1&limit=2", "?after=", "?after=%zz", "?q=" + strings.Repeat("x", 8193)} {
		for _, got := range []reply{f.selectionRead(a, ref+query), f.operationRead(a, query), f.operationRead(a, "/"+id+"/items"+query)} {
			if got.status != 422 {
				t.Fatalf("malformed query status%d", got.status)
			}
		}
	}
	for _, query := range []string{"?view=other", "?view=results&view=not_included"} {
		if got := f.operationRead(a, "/"+id+"/items"+query); got.status != 422 {
			t.Fatal("invalid detail view accepted")
		}
	}
	if got := f.operationRead(a, "/"+id+"?limit=1"); got.status != 422 {
		t.Fatal("summary ignored unsupported query")
	}
	for _, got := range []reply{f.selectionRead(actor{}, ref), f.operationRead(actor{}, ""), f.operationRead(actor{}, "/"+id), f.operationRead(actor{}, "/"+id+"/items")} {
		if got.status != 401 {
			t.Fatal("anonymous operation read")
		}
	}
	for _, got := range []reply{f.selectionRead(a, "missing"), f.operationRead(a, "/missing"), f.operationRead(a, "/missing/items")} {
		if got.status != 404 {
			t.Fatal("missing operation disclosed")
		}
	}
	f.setPassword(AdminLogin, pw)
	foreign := f.signedIn(AdminLogin, pw)
	if got := f.operationRead(foreign, "/"+id+"/items"); got.status != 404 {
		t.Fatal("foreign results disclosed")
	}
}
