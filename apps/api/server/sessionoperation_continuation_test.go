package server

import (
	"net/url"
	"testing"
	"time"
)

func TestSessionOperationItemPagesExhaustResultsAndExclusionsWithScopedCursors(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	var other actor
	for i := 0; i < 3; i++ {
		other = f.signedIn(MemberLogin, pw)
	}
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 {
		t.Fatal("acceptance", accepted.status)
	}
	path := "/" + accepted.body["operation"].(string) + "/items"
	firstCursors := map[string]string{}
	for _, view := range []string{"results", "not_included"} {
		seen := map[string]bool{}
		cursor := ""
		want := 2
		if view == "not_included" {
			want = 3
		}
		for page := 0; page < want; page++ {
			query := "?view=" + view + "&limit=1"
			if cursor != "" {
				query += "&after=" + url.QueryEscape(cursor)
			}
			got := f.operationRead(caller, path+query)
			rows, _ := got.body["rows"].([]any)
			if got.status != 200 || len(rows) != 1 || countOf(got.body["count"]) != want {
				t.Fatal("bounded item continuation", view, got.status)
			}
			handle := rows[0].(map[string]any)["handle"].(string)
			if seen[handle] {
				t.Fatal("duplicate continuation row", view)
			}
			seen[handle] = true
			cursor, _ = got.body["next"].(string)
			if page == 0 {
				firstCursors[view] = cursor
			}
		}
		if len(seen) != want || cursor != "" {
			t.Fatal("item pages did not exhaust exactly", view)
		}
	}
	for view, cursor := range firstCursors {
		query := "?view=" + view + "&limit=1&after=" + url.QueryEscape(cursor)
		if got := f.operationRead(other, path+query); got.status != 409 {
			t.Fatal("item cursor crossed caller", got.status)
		}
	}
	f.setPassword(AdminLogin, pw)
	foreign := f.signedIn(AdminLogin, pw)
	if got := f.operationRead(foreign, path+"?view=results&limit=1"); got.status != 404 {
		t.Fatal("foreign owner read accepted rows", got.status)
	}
	f.advance(6 * time.Minute)
	for view, cursor := range firstCursors {
		if got := f.operationRead(caller, path+"?view="+view+"&limit=1&after="+url.QueryEscape(cursor)); got.status != 409 {
			t.Fatal("expired item cursor accepted", got.status)
		}
	}
}
