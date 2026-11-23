package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestSessionQueryExclusionTagUsesRetainedAcceptedOtherOperation(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal("preparation", prepared.status)
	}
	held := f.signedIn(MemberLogin, pw)
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 {
		t.Fatal("acceptance", accepted.status)
	}
	if got := f.me(caller); got.body["current_session"].(map[string]any)["not_included_operation"] != "" {
		t.Fatal("current caller incorrectly tagged as excluded")
	}
	page := f.sessionQuery(caller, "?session=other&limit=1")
	rows, _ := page.body["rows"].([]any)
	if page.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["handle"] != handleOfActor(held) {
		t.Fatal("held row missing from bounded query")
	}
	if rows[0].(map[string]any)["not_included_operation"] != accepted.body["operation"] {
		t.Fatal("bounded session row lost confirmed-set exclusion fact")
	}
	detail := f.do(http.MethodGet, "/api/v1/me/sessions/"+handleOfActor(held), "", func(r *http.Request) { r.AddCookie(caller.cookie); r.Header.Set(csrfHeader, caller.csrf) })
	if detail.status != 200 || detail.body["session"].(map[string]any)["not_included_operation"] != accepted.body["operation"] {
		t.Fatal("known-ID detail lost the same scoped exclusion fact")
	}
}

func TestSessionQueryCursorRejectsNoRemovalAcceptanceAndLastOperationEviction(t *testing.T) {
	f, _, _, caller := queryActors(t)
	before := f.sessionQuery(caller, "?session=other&limit=1")
	cursor, _ := before.body["next"].(string)
	if before.status != 200 || cursor == "" {
		t.Fatal("initial cursor missing")
	}
	generation, _ := f.me(caller).body["sessions_generation"].(string)
	accepted := f.selfPost(caller, "sessions/end", `{"session":"others","generation":"`+generation+`","handles":["0000000000000000"]}`)
	if accepted.status != 202 || countOf(accepted.body["count"]) != 0 {
		t.Fatal("no-removal legacy acceptance failed", accepted.status)
	}
	if got := f.sessionQuery(caller, "?session=other&limit=1&after="+url.QueryEscape(cursor)); got.status != 409 {
		t.Fatal("acceptance without session removal did not invalidate query cursor", got.status)
	}
	during := f.sessionQuery(caller, "?session=other&limit=1")
	duringCursor, _ := during.body["next"].(string)
	f.s.domain.Lock()
	record := f.s.sessionOperations[accepted.body["operation"].(string)]
	f.s.forgetSessionRecord(record)
	f.s.domain.Unlock()
	for _, raw := range []string{cursor, duringCursor} {
		if got := f.sessionQuery(caller, "?session=other&limit=1&after="+url.QueryEscape(raw)); got.status != 409 {
			t.Fatal("last-operation eviction reused an older query version", got.status)
		}
	}
}

func TestSessionQueryCursorRejectsRetainedOperationEvictionAfterOwnerIndexRecreation(t *testing.T) {
	f, _, _, caller := queryActors(t)
	generation, _ := f.me(caller).body["sessions_generation"].(string)
	accepted := f.selfPost(caller, "sessions/end", `{"session":"others","generation":"`+generation+`","handles":["0000000000000000"]}`)
	if accepted.status != 202 {
		t.Fatal("initial acceptance", accepted.status)
	}
	f.s.domain.Lock()
	for _, verifier := range f.s.sessions.byLogin(MemberLogin) {
		f.s.sessions.take(verifier)
	}
	f.s.domain.Unlock()
	f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	current := f.signedIn(MemberLogin, pw)
	page := f.sessionQuery(current, "?session=other&limit=1")
	cursor, _ := page.body["next"].(string)
	if page.status != 200 || cursor == "" {
		t.Fatal("recreated owner cursor missing")
	}
	f.s.domain.Lock()
	record := f.s.sessionOperations[accepted.body["operation"].(string)]
	f.s.forgetSessionRecord(record)
	f.s.domain.Unlock()
	if got := f.sessionQuery(current, "?session=other&limit=1&after="+url.QueryEscape(cursor)); got.status != 409 {
		t.Fatal("evicting retained old record missed new owner's cursor", got.status)
	}
}

func TestSessionQueryExclusionTagIgnoresNewerOneOperationsAndFallsBackAfterEviction(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	held := f.signedIn(MemberLogin, pw)
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 {
		t.Fatal("original acceptance", accepted.status)
	}
	first := accepted.body["operation"].(string)
	check := func(want string) {
		t.Helper()
		got := f.sessionQuery(caller, "?session=other&q="+handleOfActor(held)+"&limit=1")
		rows, _ := got.body["rows"].([]any)
		if got.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["not_included_operation"] != want {
			t.Fatal("wrong latest retained others operation", got.status)
		}
	}
	for i := 0; i < 3; i++ {
		target := f.signedIn(MemberLogin, pw)
		one := f.selfPost(caller, "sessions/end", `{"session":"`+handleOfActor(target)+`"}`)
		if one.status != 202 {
			t.Fatal("one acceptance", one.status)
		}
	}
	check(first)
	generation, _ := f.me(caller).body["sessions_generation"].(string)
	newer := f.selfPost(caller, "sessions/end", `{"session":"others","generation":"`+generation+`","handles":["0000000000000001"]}`)
	if newer.status != 202 {
		t.Fatal("new others acceptance", newer.status)
	}
	check(newer.body["operation"].(string))
	f.s.domain.Lock()
	record := f.s.sessionOperations[newer.body["operation"].(string)]
	f.s.forgetSessionRecord(record)
	linked := record.olderOthers != nil || record.newerOthers != nil
	f.s.domain.Unlock()
	if linked {
		t.Fatal("forgotten record retained subset links")
	}
	check(first)
	f.s.domain.Lock()
	f.s.forgetSessionRecord(f.s.sessionOperations[first])
	f.s.domain.Unlock()
	check("")
	assertSessionRecordAccounting(t, f)
}

func TestSessionQueryExclusionTagRetainsPendingOtherOperationAcrossManyNewerOneOperations(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	held := f.signedIn(MemberLogin, pw)
	f.s.fault = func(key string) bool { return key == "self-end-unavailable" }
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	f.s.fault = nil
	if accepted.status != 202 {
		t.Fatal("pending others acceptance", accepted.status)
	}
	for i := 0; i < 12; i++ {
		target := f.signedIn(MemberLogin, pw)
		if one := f.selfPost(caller, "sessions/end", `{"session":"`+handleOfActor(target)+`"}`); one.status != 202 {
			t.Fatal("newer one acceptance", one.status)
		}
	}
	got := f.sessionQuery(caller, "?session=other&q="+handleOfActor(held)+"&limit=1")
	rows, _ := got.body["rows"].([]any)
	if got.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["not_included_operation"] != accepted.body["operation"] {
		t.Fatal("newer one operations replaced retained pending others fact")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionQueryCursorRejectsOwnRecordEvictionByAnotherOwnersLockedReservation(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 500
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	member := f.signedIn(MemberLogin, pw)
	first := ""
	for round := 0; round < 5; round++ {
		for i := 0; i < 20; i++ {
			if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
				t.Fatal(err)
			}
		}
		prep := f.selfPost(member, "sessions/selections", "{}")
		accepted := f.selfPost(member, "sessions/end", selectionEndBody(prep))
		if accepted.status != 202 {
			t.Fatal("member capacity setup", accepted.status)
		}
		if first == "" {
			first = accepted.body["operation"].(string)
		}
	}
	f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.setPassword(AdminLogin, pw)
	admin := f.signedIn(AdminLogin, pw)
	for i := 0; i < 20; i++ {
		if _, _, err := f.s.openFor(LoginTriple(AdminLogin), SampleWorkspace, false); err != nil {
			t.Fatal(err)
		}
	}
	generation, _ := f.me(admin).body["sessions_generation"].(string)
	handles := make([]string, 20)
	for i := range handles {
		handles[i] = fmt.Sprintf("%016x", i)
	}
	body, err := json.Marshal(map[string]any{"session": "others", "generation": generation, "handles": handles})
	if err != nil {
		t.Fatal(err)
	}
	page := f.sessionQuery(member, "?session=other&limit=1")
	cursor, _ := page.body["next"].(string)
	if page.status != 200 || cursor == "" {
		t.Fatal("member cursor missing")
	}
	revision := f.s.Revision()
	accepted := f.selfPost(admin, "sessions/end", string(body))
	if accepted.status != 202 {
		t.Fatal("admin nested capacity reservation", accepted.status)
	}
	if _, present := accepted.body["count"]; !present || countOf(accepted.body["count"]) != 0 {
		t.Fatal("fixture unexpectedly removed live admin sessions")
	}
	if f.s.Revision() != revision {
		t.Fatal("global revision changed and could mask the owner operation fence")
	}
	if f.s.sessionOperations[first] != nil {
		t.Fatal("foreign reservation did not exercise own-record eviction")
	}
	if got := f.sessionQuery(member, "?session=other&limit=1&after="+url.QueryEscape(cursor)); got.status != 409 {
		t.Fatal("nested eviction missed affected owner's cursor", got.status)
	}
	assertSessionRecordAccounting(t, f)
}
