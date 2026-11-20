package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestSessionRegisteredSelectionMismatchPreservesTargetsAndReservation(t *testing.T) {
	f, first, second, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 || countOf(prepared.body["count"]) != 2 {
		t.Fatal("actual sealed selection premise")
	}
	units := f.s.sessionRecordUnits
	for _, field := range []string{"digest", "count"} {
		body := map[string]any{"selection": prepared.body["selection"], "digest": prepared.body["digest"], "count": 2}
		if field == "digest" {
			body[field] = strings.Repeat("0", 64)
			if body[field] == prepared.body["digest"] {
				t.Fatal("wrong digest premise")
			}
		} else {
			body[field] = 1
		}
		raw, _ := json.Marshal(body)
		got := f.selfPost(caller, "sessions/end", string(raw))
		if got.status != 422 || got.body["code"] != string(commands.InvalidRequest) || f.s.sessionRecordUnits != units || len(f.s.sessionOperations) != 0 || f.s.sessionRecords[prepared.body["selection"].(string)] == nil {
			t.Fatal("registered selection mismatch changed accepted effects or reservation", field, got.status)
		}
	}
	for _, a := range []actor{first, second, caller} {
		if f.me(a).status != 200 {
			t.Fatal("mismatch ended an actual session")
		}
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionCapturedCallerLogoutRefusesSelectionApplication(t *testing.T) {
	f, first, second, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal("actual sealed selection premise")
	}
	found := capturedAdmissionSession(f, caller)
	answer, code := f.s.access(found)
	if code != "" {
		t.Fatal("actual admitted caller premise")
	}
	var body endBody
	if json.Unmarshal([]byte(selectionEndBody(prepared)), &body) != nil {
		t.Fatal("actual selection body decode")
	}
	record := f.s.sessionRecords[*body.Selection]
	units := f.s.sessionRecordUnits
	if f.logout(caller).status != 204 {
		t.Fatal("actual logout premise")
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/sessions/end", nil)
	f.s.acceptSessionSelection(w, r, found, answer, body)
	if w.Code != 401 || record.op != nil || record.state != "sealed" || len(f.s.sessionOperations) != 0 || f.s.sessionRecordUnits != units || f.me(first).status != 200 || f.me(second).status != 200 || f.me(caller).status != 401 {
		t.Fatal("captured ended caller applied or damaged the retained selection")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionPreviewAndBodyParserRejectPublicMalformedInputsWithoutEffects(t *testing.T) {
	f, first, second, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal("actual preview premise")
	}
	units := f.s.sessionRecordUnits
	ref := prepared.body["selection"].(string)
	got := f.selectionRead(caller, ref+"?after="+strings.Repeat("x", 4097))
	if got.status != 422 || got.body["rows"] != nil {
		t.Fatal("oversized public preview position was admitted", got.status)
	}
	for _, body := range []string{`{"\x":"others"}`, `{"session":"others","\u`} {
		if got := f.selfPost(caller, "sessions/end", body); got.status != 422 {
			t.Fatal("malformed public object key was admitted", got.status)
		}
	}
	if f.s.sessionRecordUnits != units || len(f.s.sessionOperations) != 0 || f.me(first).status != 200 || f.me(second).status != 200 || f.me(caller).status != 200 {
		t.Fatal("parser refusal changed actual sessions or reservations")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionPendingCleanupCannotBeForgottenBeforeActualCompletion(t *testing.T) {
	f, _, _, caller := queryActors(t)
	f.s.fault = func(key string) bool { return key == "self-end-unavailable" }
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if prepared.status != 200 || accepted.status != 202 || accepted.body["state"] != stateRunning {
		t.Fatal("actual pending cleanup premise")
	}
	record := f.s.sessionOperations[accepted.body["operation"].(string)]
	units, retries := f.s.sessionRecordUnits, record.retries
	if retries == 0 {
		t.Fatal("actual cleanup retry premise")
	}
	f.s.forgetSessionRecord(record)
	if f.s.sessionRecords[record.ref] != record || f.s.sessionRecordUnits != units || record.retries != retries {
		t.Fatal("pending cleanup was discarded before completion")
	}
	f.s.fault = nil
	f.advance(2 * time.Second)
	f.s.drainSelfEnds()
	if record.retries != 0 || f.s.sessionRecordUnits >= units {
		t.Fatal("actual cleanup did not release retry reservations")
	}
	f.s.forgetSessionRecord(record)
	if f.s.sessionRecordUnits != 0 || f.s.sessionRecords[record.ref] != nil || f.me(caller).status != 200 {
		t.Fatal("settled cleanup failed to release exactly")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSignerUnknownPositionsRefuseWithoutPartialPages(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal("actual sealed selection premise")
	}
	ref := prepared.body["selection"].(string)
	paths := []struct {
		page reply
		read func(string) reply
	}{
		{f.sessionQuery(caller, "?session=other&limit=1"), func(after string) reply { return f.sessionQuery(caller, "?session=other&limit=1&after="+after) }},
		{f.selectionRead(caller, ref+"?limit=1"), func(after string) reply { return f.selectionRead(caller, ref+"?limit=1&after="+after) }},
	}
	for _, path := range paths {
		raw, _ := path.page.body["next"].(string)
		cursor, ok := f.s.openSessionCursor(raw)
		if path.page.status != 200 || raw == "" || !ok {
			t.Fatal("actual signed continuation premise")
		}
		cursor.After = "0000000000000000"
		got := path.read(url.QueryEscape(f.s.sealSessionCursor(cursor)))
		assertServiceQueryFailurePayload(t, got, 409, commands.RevisionConflict)
	}
	if len(f.s.sessionOperations) != 0 || f.me(caller).status != 200 {
		t.Fatal("unknown signer position produced effects")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionOperationSignerUnknownResultAndHeldPositionsRefuse(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 || countOf(prepared.body["count"]) != 2 {
		t.Fatal("actual frozen two-target premise")
	}
	held := []actor{f.signedIn(MemberLogin, pw), f.signedIn(MemberLogin, pw)}
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 || countOf(accepted.body["ended_count"]) != 2 || countOf(accepted.body["not_included_count"]) != 2 {
		t.Fatal("actual result and held populations premise")
	}
	id := accepted.body["operation"].(string)
	for _, view := range []string{"results", "not_included"} {
		path := "/" + id + "/items?view=" + view + "&limit=1"
		page := f.operationRead(caller, path)
		raw, _ := page.body["next"].(string)
		cursor, ok := f.s.openSessionCursor(raw)
		if page.status != 200 || raw == "" || !ok || len(page.body["rows"].([]any)) != 1 {
			t.Fatal("actual operation signed continuation premise", view)
		}
		cursor.After = "0000000000000000"
		got := f.operationRead(caller, path+"&after="+url.QueryEscape(f.s.sealSessionCursor(cursor)))
		assertServiceQueryFailurePayload(t, got, 409, commands.RevisionConflict)
		if f.operationRead(caller, "/"+id).status != 200 {
			t.Fatal("unknown signer position erased accepted recovery")
		}
	}
	for _, a := range append(held, caller) {
		if f.me(a).status != 200 {
			t.Fatal("unknown operation position terminated a surviving session")
		}
	}
	assertSessionRecordAccounting(t, f)
}
