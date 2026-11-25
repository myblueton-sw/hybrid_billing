package server

import (
	"net/url"
	"testing"
	"time"
)

func TestSessionOperationPagesExhaustWithoutDuplicatesAndBindCaller(t *testing.T) {
	f, _, _, a := queryActors(t)
	for i := 0; i < 4; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		if got := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(actor{cookie, csrf})+`"}`); got.status != 202 {
			t.Fatal(got.status)
		}
	}
	seen := map[string]bool{}
	after := ""
	for i := 0; i < 5; i++ {
		page := f.operationRead(a, "?limit=1&after="+url.QueryEscape(after))
		if after == "" {
			page = f.operationRead(a, "?limit=1")
		}
		if page.status != 200 {
			t.Fatal(page.status)
		}
		rows := page.body["rows"].([]any)
		if len(rows) != 1 {
			t.Fatal("missing bounded operation row")
		}
		id := rows[0].(map[string]any)["operation"].(string)
		if seen[id] {
			t.Fatal("duplicate operation page")
		}
		seen[id] = true
		after = page.body["next"].(string)
		if after == "" {
			break
		}
		if i == 0 {
			other := f.signedIn(MemberLogin, pw)
			if got := f.operationRead(other, "?limit=1&after="+url.QueryEscape(after)); got.status != 409 {
				t.Fatal("cursor crossed caller")
			}
			if got := f.operationRead(a, "?limit=2&after="+url.QueryEscape(after)); got.status != 409 {
				t.Fatal("cursor crossed page size")
			}
		}
	}
	if len(seen) != 4 {
		t.Fatal("operation paging omitted records")
	}
}

func TestSessionOperationItemCursorInvalidatesOnCleanupAndView(t *testing.T) {
	f, _, _, a := queryActors(t)
	f.s.fault = func(key string) bool { return key == "self-end-unavailable" }
	prepared := f.selfPost(a, "sessions/selections", "{}")
	accepted := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 {
		t.Fatal(accepted.status)
	}
	path := "/" + accepted.body["operation"].(string) + "/items"
	page := f.operationRead(a, path+"?limit=1")
	if page.status != 200 || page.body["next"] == "" {
		t.Fatal("missing result cursor")
	}
	after := url.QueryEscape(page.body["next"].(string))
	if got := f.operationRead(a, path+"?view=not_included&limit=1&after="+after); got.status != 409 {
		t.Fatal("cursor crossed view")
	}
	if got := f.operationRead(a, path+"?limit=2&after="+after); got.status != 409 {
		t.Fatal("result cursor crossed page size")
	}
	f.s.fault = nil
	f.advance(2 * time.Second)
	f.s.drainSelfEnds()
	if got := f.operationRead(a, path+"?limit=1&after="+after); got.status != 409 {
		t.Fatal("cleanup did not invalidate result cursor")
	}
}

func TestSessionOperationUnconfirmedIntentHasUnknownOutcomes(t *testing.T) {
	f, _, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	record := f.s.sessionRecords[prepared.body["selection"].(string)]
	op := f.s.startSessionOperation(record, kindOthers)
	record.state = "applying"
	f.s.sessionOperations[op.id] = record
	page := f.operationRead(a, "/"+op.id)
	if page.status != 200 || page.body["state"] != "unconfirmed" {
		t.Fatal("unconfirmed intent missing")
	}
	for _, key := range []string{"ended_count", "gone_count", "not_included_count", "cleanup", "blocked_at", "current_generation", "generation_changed"} {
		if value, exists := page.body[key]; !exists || value != nil {
			t.Fatal("unconfirmed outcome was asserted", key)
		}
	}
	if got := f.operationRead(a, "/"+op.id+"/items"); got.status != 503 {
		t.Fatal("unconfirmed items were presented")
	}
	other := f.signedIn(MemberLogin, pw)
	if got := f.operationRead(other, "/"+op.id); got.status != 404 {
		t.Fatal("unconfirmed intent crossed caller")
	}
	list := f.operationRead(a, "")
	if list.status != 200 || countOf(list.body["accepted_count"]) != 0 {
		t.Fatal("unconfirmed intent counted as accepted")
	}
}
