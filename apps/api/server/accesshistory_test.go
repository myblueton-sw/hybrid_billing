package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const chromeMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36"

func (f world) history(a actor, query string) reply {
	f.t.Helper()
	f.s.history.Flush()
	path := "/api/v1/me/access-history"
	if query != "" {
		path += "?" + query
	}
	return f.do("GET", path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) signinAgent(login, password, agent string) reply {
	f.t.Helper()
	pre := f.pre()
	return f.do("POST", "/api/v1/auth/login", loginBodyFor(login, password), func(r *http.Request) {
		r.Header.Set(clientHeader, ConsoleClient)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("User-Agent", agent)
		r.AddCookie(pre)
		r.Header.Set(csrfHeader, pre.Value)
	})
}

func historyEvents(t testing.TB, got reply) []map[string]any {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("history read %+v", got)
	}
	out := []map[string]any{}
	for _, e := range got.body["events"].([]any) {
		out = append(out, e.(map[string]any))
	}
	return out
}

func kindList(list []map[string]any) string {
	out := []string{}
	for _, e := range list {
		k := e["kind"].(string)
		if r := e["reason"].(string); r != "" {
			k += ":" + r
		}
		out = append(out, k)
	}
	return strings.Join(out, ",")
}

func historyWorld(t *testing.T) world {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.setPassword(AdminLogin, pw)
	return f
}

func TestHistoryRecordsSignInsAndRefusalsPerPerson(t *testing.T) {
	f := historyWorld(t)
	if got := f.signinAgent(MemberLogin, "wrong password here", chromeMac); got.status != 401 {
		t.Fatalf("wrong %+v", got)
	}
	unknown := f.signinAgent("nobody@example.invalid", "wrong password here", chromeMac)
	if unknown.status != 401 || fmt.Sprint(unknown.body["code"]) != "unauthenticated" {
		t.Fatalf("unknown %+v", unknown)
	}
	first := f.signedIn(MemberLogin, pw)
	f.advance(time.Second)
	second := f.signedIn(MemberLogin, pw)
	admin := f.signedIn(AdminLogin, pw)
	list := historyEvents(t, f.history(second, ""))
	if kindList(list) != "signed_in,signed_in,sign_in_refused:wrong_password" {
		t.Fatalf("member history %s", kindList(list))
	}
	if s := list[0]["session"].(map[string]any); s["current"] != true || s["active"] != true || s["handle"] != handleOfActor(second) {
		t.Errorf("newest row session %v", s)
	}
	if s := list[1]["session"].(map[string]any); s["current"] != false || s["active"] != true || s["handle"] != handleOfActor(first) {
		t.Errorf("older row session %v", s)
	}
	if d := list[2]["device"].(map[string]any); d["browser"] != "chrome" || d["os"] != "macos" || list[2]["session"] != nil || list[2]["workspace"] != nil {
		t.Errorf("refused row %v", list[2])
	}
	if got := kindList(historyEvents(t, f.history(admin, ""))); got != "signed_in" {
		t.Fatalf("admin sees %s", got)
	}
	raw, _ := json.Marshal(f.history(admin, "").body)
	if strings.Contains(string(raw), handleOfActor(first)) || strings.Contains(string(raw), "nobody") {
		t.Fatalf("admin read leaks member data: %s", raw)
	}
}

func TestHistoryReadParametersAndAdmission(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	for _, q := range []string{"user=" + SampleAdmin, "limit=0", "limit=101", "limit=x", "before=***", "before=" + encodeCursor(1, 2) + "&before=" + encodeCursor(1, 2),
		"from=yesterday", "from=2026-11-17T00:00:00Z&to=2026-11-16T00:00:00Z"} {
		if got := f.history(a, q); got.status != 422 || fmt.Sprint(got.body["code"]) != "invalid_request" {
			t.Errorf("%s: %+v", q, got)
		}
	}
	got := f.history(a, "limit=1&from=2026-11-16T00:00:00Z&to=2026-11-17T00:00:00Z")
	if got.status != 200 || got.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("ok read %+v", got)
	}
	if got := f.history(actor{a.cookie, "bad"}, ""); got.status != 403 {
		t.Errorf("csrf %+v", got)
	}
	if got := f.history(actor{}, ""); got.status != 401 {
		t.Errorf("anonymous %+v", got)
	}
	body := f.history(a, "").body
	for _, k := range []string{"events", "next", "retention_days", "retention_source", "period_since", "effective_days", "available_from", "recording", "notes", "now"} {
		if _, ok := body[k]; !ok {
			t.Errorf("missing %s", k)
		}
	}
	if fmt.Sprint(body["retention_days"]) != "90" || body["retention_source"] != SourceDefault {
		t.Errorf("retention %v %v", body["retention_days"], body["retention_source"])
	}
}

func TestHistoryPagingAndCursorReflectsOnlyOwnRecords(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	for i := 0; i < 5; i++ {
		f.signedIn(AdminLogin, pw)
		f.signinAgent(MemberLogin, "wrong password here", chromeMac)
	}
	seen := []string{}
	query := "limit=2"
	for i := 0; i < 10; i++ {
		got := f.history(a, query)
		for _, e := range historyEvents(t, got) {
			seen = append(seen, e["kind"].(string))
		}
		next, ok := got.body["next"].(string)
		if !ok {
			break
		}
		w, r, valid := decodeCursor(next)
		if !valid || w > 2 || r > 6 {
			t.Fatalf("cursor %q = %d %d", next, w, r)
		}
		query = "limit=2&before=" + next
	}
	if len(seen) != 6 {
		t.Fatalf("paged %v", seen)
	}
}

func TestHistorySelfServiceEventsAndCauses(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	b := f.signedIn(MemberLogin, pw)
	if got := f.reauth(a, "wrong password here"); got.status != 422 {
		t.Fatalf("wrong reauth %+v", got)
	}
	if got := f.reauth(a, pw); got.status != 204 {
		t.Fatalf("reauth %+v", got)
	}
	if got := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(b)+`"}`); got.status != 202 {
		t.Fatalf("end one %+v", got)
	}
	c := f.signedIn(MemberLogin, pw)
	if got := f.selfPost(a, "password", pwBody(pw, newPw)); got.status != 204 {
		t.Fatalf("password %+v", got)
	}
	d := f.signedIn(MemberLogin, newPw)
	f.logout(d)
	list := historyEvents(t, f.history(a, ""))
	want := "signed_out,signed_in,session_ended:password_changed,password_changed,signed_in,session_ended:ended_by_you,password_confirmed,password_confirmation_failed:wrong_password,signed_in,signed_in"
	if kindList(list) != want {
		t.Fatalf("history\n%s\nwant\n%s", kindList(list), want)
	}
	if list[2]["session"].(map[string]any)["handle"] != handleOfActor(c) || list[5]["session"].(map[string]any)["handle"] != handleOfActor(b) {
		t.Errorf("ended handles %v %v", list[2]["session"], list[5]["session"])
	}
}

func TestHistoryEndOthersRecordsOnePerEndedSession(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	b := f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	generation, handles := f.generationOf(a), []string{handleOfActor(b), "00000000000000aa"}
	if got := f.selfPost(a, "sessions/end", endJSON(generation, handles)); got.status != 202 {
		t.Fatalf("end others %+v", got)
	}
	ended := 0
	for _, e := range historyEvents(t, f.history(a, "")) {
		if e["kind"] == accesshistory.KindSessionEnded {
			ended++
			if e["reason"] != accesshistory.CauseByYou || e["session"].(map[string]any)["handle"] != handleOfActor(b) {
				t.Errorf("ended row %v", e)
			}
		}
	}
	if ended != 1 {
		t.Fatalf("ended rows %d", ended)
	}
}

func TestHistoryAttemptLimitRecordsFailureAndEnd(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	watcher := f.signedIn(MemberLogin, pw)
	for i := 0; i < selfAttempts; i++ {
		f.reauth(a, "wrong password here")
	}
	if got := f.reauth(a, "wrong password here"); got.status != 401 {
		t.Fatalf("sixth %+v", got)
	}
	got := kindList(historyEvents(t, f.history(watcher, "")))
	if !strings.HasPrefix(got, "session_ended:attempt_limit,password_confirmation_failed:attempt_limit,") {
		t.Fatalf("history %s", got)
	}
}

func TestHistoryAccessCausesByRefusalValue(t *testing.T) {
	cases := map[identity.Reason]string{identity.ReasonSession: "", identity.ReasonUser: accesshistory.CauseAdministrator,
		identity.ReasonOperatorBlock: accesshistory.CauseAdministrator, identity.ReasonMembership: accesshistory.CauseAccessChanged,
		identity.ReasonRolePeriod: accesshistory.CauseRolePeriod, identity.ReasonState: accesshistory.CauseAccessChanged, identity.ReasonUnknown: accesshistory.CauseAccessChanged}
	for reason, want := range cases {
		if got := identityCause(reason); got != want {
			t.Errorf("%s = %q want %q", reason, got, want)
		}
	}
	policy := map[string]string{authpolicy.ReasonStale: "", authpolicy.ReasonBlocked: accesshistory.CauseAccessChanged, authpolicy.ReasonMFARequired: accesshistory.CauseAccessChanged,
		authpolicy.ReasonMethodNotAllowed: accesshistory.CauseAccessChanged, authpolicy.ReasonFutureAuthentication: accesshistory.CauseAccessChanged}
	for reason, want := range policy {
		if got := policyCause(reason); got != want {
			t.Errorf("%s = %q want %q", reason, got, want)
		}
	}
}

func TestHistoryExpiryRecordsNothing(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	watcher := f.signedIn(MemberLogin, pw)
	f.advance(sessionIdle + time.Minute)
	if got := f.me(a); got.status != 401 {
		t.Fatalf("expired %+v", got)
	}
	_ = watcher
	b := f.signedIn(MemberLogin, pw)
	if got := kindList(historyEvents(t, f.history(b, ""))); strings.Contains(got, "session_ended") {
		t.Fatalf("expiry recorded: %s", got)
	}
}

func TestHistorySurvivesRestartAndOldSessionsAreNotActive(t *testing.T) {
	f := historyWorld(t)
	old := f.signedIn(MemberLogin, pw)
	f.advance(time.Minute)
	g := f.restart(nil)
	g.setPassword(MemberLogin, pw)
	a := g.signedIn(MemberLogin, pw)
	list := historyEvents(t, g.history(a, ""))
	if kindList(list) != "signed_in,signed_in" {
		t.Fatalf("after restart %s", kindList(list))
	}
	if s := list[1]["session"].(map[string]any); s["handle"] != handleOfActor(old) || s["active"] != false || s["current"] != false {
		t.Errorf("old session %v", s)
	}
	for _, n := range g.history(a, "").body["notes"].([]any) {
		t.Errorf("clean restart note %v", n)
	}
}

func TestHistoryIncarnationHidesEarlierRecordsAndFailsClosed(t *testing.T) {
	f := historyWorld(t)
	root := f.signedIn(AdminLogin, pw)
	f.s.fault = func(at string) bool { return at == "incarnation" }
	got := f.run(root, "identity.create_user", usersTarget, map[string]string{"login": "late@example.invalid", "email": "late@example.invalid", "role": "member"})
	if got.status != 503 || f.known("late@example.invalid") {
		t.Fatalf("create with failed marker %+v", got)
	}
	f.s.fault = nil
	user, link := f.created(root, "late@example.invalid", "member")
	if got := f.redeem(link, pw); got.status != 204 {
		t.Fatalf("redeem %+v", got)
	}
	a := f.signedIn("late@example.invalid", pw)
	if got := kindList(historyEvents(t, f.history(a, ""))); got != "signed_in" {
		t.Fatalf("new person %s", got)
	}
	if f.s.history == nil || user == "" {
		t.Fatal("no history")
	}
	f.s.history.Offer(accesshistory.Record{Kind: accesshistory.KindSignedIn, User: user, Workspace: SampleWorkspace, Handle: "0123456789abcdef", Browser: "chrome", OS: "macos"})
	f.s.history.Flush()
	if err := f.s.history.Incarnation(user); err != nil {
		t.Fatal(err)
	}
	if got := historyEvents(t, f.history(a, "")); len(got) != 0 {
		t.Fatalf("earlier incarnation still visible: %s", kindList(got))
	}
}

func TestHistoryInviteLinkRetriesAfterMarkerFailure(t *testing.T) {
	f := historyWorld(t)
	root := f.signedIn(AdminLogin, pw)
	_, link := f.created(root, "invited@example.invalid", "member")
	f.s.fault = func(at string) bool { return at == "incarnation" }
	if got := f.redeem(link, pw); got.status == 204 {
		t.Fatalf("redeem with failed marker %+v", got)
	}
	if got := f.signin("invited@example.invalid", pw); got.status != 401 {
		t.Fatalf("password committed despite the failed marker %+v", got)
	}
	f.s.fault = nil
	if got := f.redeem(link, pw); got.status != 204 {
		t.Fatalf("same invite link after the failure %+v", got)
	}
}

func TestHistoryBootstrapActivationRetriesAfterMarkerFailure(t *testing.T) {
	f := setup(t, SeedNone).ready()
	f.s.fault = func(at string) bool { return at == "incarnation" }
	if got := f.auth("redeem", activationBody(f.links()[""], "first@example.invalid"), f.pre()); got.status == 204 {
		t.Fatalf("activation with failed marker %+v", got)
	}
	if f.known("first@example.invalid") {
		t.Fatal("bootstrapped despite the failed marker")
	}
	f.s.fault = nil
	if got := f.auth("redeem", activationBody(f.links()[""], "first@example.invalid"), f.pre()); got.status != 204 {
		t.Fatalf("same activation link after the failure %+v", got)
	}
}

func TestHistoryFilesHoldNoSecrets(t *testing.T) {
	f := historyWorld(t)
	a := f.signedIn(MemberLogin, pw)
	f.signinAgent(MemberLogin, "wrong password here", chromeMac)
	f.reauth(a, pw)
	f.history(a, "")
	f.s.Close()
	dir := filepath.Join(f.s.cfg.Data, accesshistory.DirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	needles := []string{pw, "wrong password here", MemberLogin, AdminLogin, a.csrf, a.cookie.Value, "Mozilla", "AppleWebKit", "127.0.0.1"}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range needles {
			if n != "" && strings.Contains(string(raw), n) {
				t.Errorf("%s contains %q", e.Name(), n)
			}
		}
	}
}

func TestOpenAPIAccessHistoryEntry(t *testing.T) {
	f := memberWorld(t)
	raw, err := f.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	op := doc["x-self"].(map[string]any)["/me/access-history"].(map[string]any)["get"].(map[string]any)
	if op["operationId"] != "me.access_history.read" {
		t.Fatalf("op %v", op)
	}
	responses := op["responses"].(map[string]any)
	for status, code := range map[string]string{"401": "unauthenticated", "403": "csrf_refused", "422": "invalid_request", "500": "internal_error", "503": "authority_unavailable"} {
		if responses[status].(map[string]any)["x-code"] != code {
			t.Errorf("%s %v", status, responses[status])
		}
	}
	props := responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)
	view, _ := json.Marshal(historyPage{})
	var fields map[string]any
	_ = json.Unmarshal(view, &fields)
	for name := range fields {
		if props[name] == nil {
			t.Errorf("answer field %s missing from the schema", name)
		}
	}
	if len(props) != len(fields) {
		t.Errorf("schema %d fields, answer %d", len(props), len(fields))
	}
}
