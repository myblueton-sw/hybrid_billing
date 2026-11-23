package server

import (
	"net/http"
	"testing"
	"time"
)

func TestSessionSelectionOneUsesFrozenReferenceAndOwnReplay(t *testing.T) {
	f, target, peer, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", `{"session":"`+handleOfActor(target)+`"}`)
	if prepared.status != 200 {
		t.Fatalf("one-session preparation status%d,want200", prepared.status)
	}
	if prepared.body["kind"] != kindOne || prepared.body["state"] != "sealed" || countOf(prepared.body["count"]) != 1 {
		t.Fatal("single target was not sealed distinctly")
	}
	same := f.selfPost(a, "sessions/selections", `{"session":"`+handleOfActor(target)+`"}`)
	if same.body["selection"] != prepared.body["selection"] || len(f.s.sessionRecords) != 1 {
		t.Fatal("same single-target preparation allocated twice")
	}
	preview := f.selectionRead(a, prepared.body["selection"].(string))
	if preview.status != 200 || len(preview.body["rows"].([]any)) != 1 || preview.body["rows"].([]any)[0].(map[string]any)["handle"] != handleOfActor(target) {
		t.Fatal("single preview widened targets")
	}
	newcomer := f.signedIn(MemberLogin, pw)
	body := selectionEndBody(prepared)
	accepted := f.selfPost(a, "sessions/end", body)
	if accepted.status != 202 || accepted.body["kind"] != kindOne || countOf(accepted.body["ended_count"]) != 1 || countOf(accepted.body["not_included_count"]) != 0 {
		t.Fatal("single reference applied as all others")
	}
	if f.me(target).status != 401 || f.me(peer).status != 200 || f.me(newcomer).status != 200 {
		t.Fatal("single target application widened or omitted effects")
	}
	f.advance(6 * time.Minute)
	f.s.sessions.reauthenticated(verifierOfActor(a), *f.now)
	replay := f.selfPost(a, "sessions/end", body)
	if replay.status != 202 || replay.body["operation"] != accepted.body["operation"] || replay.body["replay"] != true {
		t.Fatal("single reference replay lost operation identity")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionOneRejectsCurrentMissingForeignAndMixedBodies(t *testing.T) {
	f, target, _, a := queryActors(t)
	for _, body := range []string{`{"session":"` + handleOfActor(a) + `"}`, `{"session":null}`, `{"session":"others"}`, `{"session":"` + handleOfActor(target) + `","after":"x"}`, `{"session":"` + handleOfActor(target) + `","selection":"x"}`} {
		if got := f.selfPost(a, "sessions/selections", body); got.status != 422 {
			t.Fatal("invalid single preparation accepted", got.status)
		}
	}
	f.setPassword(AdminLogin, pw)
	other := f.signedIn(AdminLogin, pw)
	for _, handle := range []string{"0000000000000000", handleOfActor(other)} {
		if got := f.selfPost(a, "sessions/selections", `{"session":"`+handle+`"}`); got.status != http.StatusNotFound {
			t.Fatal("missing/foreign target disclosure", got.status)
		}
	}
	one := f.selfPost(a, "sessions/selections", `{"session":"`+handleOfActor(target)+`"}`)
	all := f.selfPost(a, "sessions/selections", "{}")
	if one.status != 200 || all.status != 200 || one.body["selection"] == all.body["selection"] || one.body["kind"] != kindOne || all.body["kind"] != kindOthers {
		t.Fatal("preparation reuse crossed kind or target")
	}
}
