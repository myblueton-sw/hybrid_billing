package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestSessionSelectionPreparationIsAuthenticatedAndBounded(t *testing.T) {
	f, _, _, a := queryActors(t)
	got := f.selfPost(a, "sessions/selections", "{}")
	if got.status != http.StatusOK {
		t.Fatalf("preparation status%d, want200", got.status)
	}
	if got.body["state"] != "sealed" || got.body["selection"] == "" || got.body["digest"] == "" {
		t.Fatal("preparation did not seal confirmed targets")
	}
	if got.body["rows"] != nil || got.body["handles"] != nil {
		t.Fatal("preparation downloaded the full target set")
	}
	anonymous := f.selfPost(actor{}, "sessions/selections", "{}")
	if anonymous.status != 401 {
		t.Fatalf("anonymous preparation status%d", anonymous.status)
	}
}

func TestSessionSelectionPreviewUsesFrozenBoundedRows(t *testing.T) {
	f, _, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal(prepared.status)
	}
	ref := prepared.body["selection"].(string)
	got := f.do(http.MethodGet, "/api/v1/me/sessions/selections/"+ref+"?limit=1", "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(csrfHeader, a.csrf) })
	if got.status != 200 {
		t.Fatalf("preview status%d, want200", got.status)
	}
	rows, _ := got.body["rows"].([]any)
	if len(rows) != 1 || got.body["next"] == "" {
		t.Fatal("preview failed to page the immutable target set")
	}
}

func selectionEndBody(prepared reply) string {
	raw, _ := json.Marshal(map[string]any{"selection": prepared.body["selection"], "digest": prepared.body["digest"], "count": countOf(prepared.body["count"])})
	return string(raw)
}

func TestSessionSelectionEndsOnlyFrozenTargetsAndReplaysAfterExpiry(t *testing.T) {
	f, old, middle, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal(prepared.status)
	}
	newcomer := f.signedIn(MemberLogin, pw)
	body := selectionEndBody(prepared)
	ended := f.selfPost(a, "sessions/end", body)
	if ended.status != 202 {
		t.Fatalf("frozen termination status%d, want202", ended.status)
	}
	if countOf(ended.body["recorded_count"]) != 2 || countOf(ended.body["ended_count"]) != 2 || countOf(ended.body["not_included_count"]) != 1 {
		t.Fatalf("frozen operation counts %v", ended.body)
	}
	if f.me(old).status != 401 || f.me(middle).status != 401 || f.me(newcomer).status != 200 {
		t.Fatal("termination widened or omitted its fixed set")
	}
	f.advance(6 * time.Minute)
	f.s.sessions.reauthenticated(verifierOfActor(a), *f.now)
	replay := f.selfPost(a, "sessions/end", body)
	if replay.status != 202 || replay.body["operation"] != ended.body["operation"] || replay.body["replay"] != true || f.me(newcomer).status != 200 {
		t.Fatal("accepted replay expired or reconstructed a target set")
	}
}
