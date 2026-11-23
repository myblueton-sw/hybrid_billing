package server

import (
	"net/http"
	"testing"
)

func (f world) operationRead(a actor, path string) reply {
	return f.do(http.MethodGet, "/api/v1/me/sessions/operations"+path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func TestSessionSelectionOperationReadScopesOwnerAndAllowsOwnRecovery(t *testing.T) {
	f, _, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	other := f.signedIn(MemberLogin, pw)
	accepted := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 {
		t.Fatal(accepted.status)
	}
	id := accepted.body["operation"].(string)
	own := f.operationRead(other, "/"+id)
	if own.status != 200 || countOf(own.body["recorded_count"]) != 2 {
		t.Fatalf("own operation recovery status%d", own.status)
	}
	f.setPassword(AdminLogin, pw)
	foreign := f.signedIn(AdminLogin, pw)
	hidden := f.operationRead(foreign, "/"+id)
	if hidden.status != 404 || hidden.body["operation"] != nil {
		t.Fatal("foreign operation disclosed")
	}
}

func TestSessionSelectionOperationListIsBounded(t *testing.T) {
	f, _, _, a := queryActors(t)
	for i := 0; i < 4; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		peer := actor{cookie, csrf}
		if got := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(peer)+`"}`); got.status != 202 {
			t.Fatal(got.status)
		}
	}
	got := f.operationRead(a, "?limit=1")
	if got.status != 200 {
		t.Fatalf("operation list status%d,want200", got.status)
	}
	if rows := got.body["rows"].([]any); len(rows) != 1 || got.body["next"] == "" || countOf(got.body["accepted_count"]) != 4 {
		t.Fatal("operation list was unbounded or incomplete")
	}
}

func TestSessionSelectionOperationRowsAndHeldAreBounded(t *testing.T) {
	f, a := selectionPopulation(t, 170)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	for n := 0; prepared.status == 200 && prepared.body["state"] == "building" && n < 5; n++ {
		prepared = f.selfPost(a, "sessions/selections", selectionResumeBody(prepared))
	}
	if prepared.status != 200 || prepared.body["state"] != "sealed" {
		t.Fatal("large selection did not finish preparation")
	}
	for i := 0; i < 120; i++ {
		if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
			t.Fatal(err)
		}
	}
	accepted := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 || len(accepted.raw) > 2500 || accepted.body["sessions"] != nil {
		t.Fatal("acceptance returned full results or failed")
	}
	id := accepted.body["operation"].(string)
	for _, view := range []string{"results", "not_included"} {
		page := f.operationRead(a, "/"+id+"/items?view="+view+"&limit=17")
		if page.status != 200 {
			t.Fatalf("operation rows status%d,want200", page.status)
		}
		if rows := page.body["rows"].([]any); len(rows) != 17 || page.body["next"] == "" || len(page.raw) > 30000 {
			t.Fatal("operation rows were unbounded")
		}
	}
}
