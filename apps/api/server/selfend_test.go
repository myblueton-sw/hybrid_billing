package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) othersBody(a actor) string {
	f.t.Helper()
	page := f.me(a)
	if page.status != 200 {
		f.t.Fatalf("me %+v", page)
	}
	return othersOf(page.body, nil)
}

func othersOf(page map[string]any, skip map[string]bool) string {
	handles := []string{}
	for _, row := range page["sessions"].([]any) {
		m := row.(map[string]any)
		if m["current"] == false && !skip[m["handle"].(string)] {
			handles = append(handles, m["handle"].(string))
		}
	}
	if len(handles) == 0 {
		handles = append(handles, "0000000000000000")
	}
	return endJSON(fmt.Sprint(page["sessions_generation"]), handles)
}

func endJSON(generation string, handles []string) string {
	raw, _ := json.Marshal(map[string]any{"session": "others", "generation": generation, "handles": handles})
	return string(raw)
}

func verifierOfActor(a actor) identity.Verifier {
	raw, _ := base64.RawURLEncoding.DecodeString(a.cookie.Value)
	return verifierOf(raw)
}

func countOf(v any) int {
	n, _ := strconv.Atoi(fmt.Sprint(v))
	return n
}

func handleOfActor(a actor) string { return handleOf(verifierOfActor(a)) }

func (f world) logout(a actor) reply {
	f.t.Helper()
	return f.do("POST", "/api/v1/auth/logout", "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(csrfHeader, a.csrf) })
}

func resultsOf(body map[string]any) map[string]string {
	out := map[string]string{}
	for _, row := range body["sessions"].([]any) {
		m := row.(map[string]any)
		out[m["handle"].(string)] = m["result"].(string)
	}
	return out
}

func notIncludedOf(body map[string]any) []string {
	out := []string{}
	for _, row := range body["not_included"].([]any) {
		out = append(out, row.(map[string]any)["handle"].(string))
	}
	sort.Strings(out)
	return out
}

func c3Of(body map[string]any, handle string) string {
	for _, row := range body["sessions"].([]any) {
		m := row.(map[string]any)
		if m["handle"] != handle {
			continue
		}
		for _, item := range m["items"].([]any) {
			if it := item.(map[string]any); it["item"] == "C3" {
				return fmt.Sprint(it["state"], "/", it["reason"])
			}
		}
	}
	return ""
}

func memberWorld(t testing.TB) world {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.setPassword(AdminLogin, pw)
	return f
}

func (f world) generationOf(a actor) string {
	f.t.Helper()
	return fmt.Sprint(f.me(a).body["sessions_generation"])
}

func TestSelfEndDecoderTable(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	gen := f.generationOf(a)
	hb := handleOfActor(b)
	many := []string{}
	for i := 0; i < selfEndHandles+1; i++ {
		many = append(many, fmt.Sprintf("%016x", i+1))
	}
	cases := map[string]struct {
		body  string
		field string
	}{
		"manifest-free":       {`{"session":"others"}`, "generation"},
		"no handles":          {`{"session":"others","generation":"` + gen + `"}`, "handles"},
		"empty handles":       {`{"session":"others","generation":"` + gen + `","handles":[]}`, "handles"},
		"short generation":    {`{"session":"others","generation":"abc","handles":["` + hb + `"]}`, "generation"},
		"upper generation":    {`{"session":"others","generation":"` + strings.ToUpper(gen) + `","handles":["` + hb + `"]}`, "generation"},
		"bad handle":          {`{"session":"others","generation":"` + gen + `","handles":["xyz"]}`, "handles"},
		"duplicate":           {`{"session":"others","generation":"` + gen + `","handles":["` + hb + `","` + hb + `"]}`, "handles"},
		"too many":            {endJSON(gen, many), "handles"},
		"current":             {endJSON(gen, []string{handleOfActor(a)}), "handles"},
		"unknown key":         {`{"session":"others","generation":"` + gen + `","handles":["` + hb + `"],"x":1}`, ""},
		"one with generation": {`{"session":"` + hb + `","generation":"` + gen + `"}`, "session"},
		"trailing":            {endJSON(gen, []string{hb}) + `{}`, ""},
		"not json":            {`nope`, ""},
	}
	for name, c := range cases {
		before := f.s.counter
		got := f.selfPost(a, "sessions/end", c.body)
		f.code(t, name, got, 422, string(commands.InvalidRequest))
		if c.field != "" && fmt.Sprint(got.body["fields"]) != "["+c.field+"]" {
			t.Errorf("%s fields %v", name, got.body["fields"])
		}
		if f.s.counter != before || f.me(b).status != 200 || len(f.s.selfEnds) != 0 {
			t.Fatalf("%s changed state", name)
		}
	}
	f.code(t, "csrf", f.selfPost(actor{a.cookie, "wrong"}, "sessions/end", endJSON(gen, []string{hb})), 403, CodeCSRF)
	big := `{"session":"others","generation":"` + gen + `","handles":["` + strings.Repeat("a", 2<<20) + `"]}`
	if got := f.selfPost(a, "sessions/end", big); got.status != 413 {
		t.Errorf("oversize %d", got.status)
	}
}

func TestSelfEndHoldReportsNewcomersAndGone(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	c := f.as(MemberLogin)
	other := f.as(AdminLogin)
	listed := f.generationOf(a)
	d := f.as(MemberLogin)
	if got := f.logout(c); got.status != 204 {
		t.Fatalf("logout %+v", got)
	}
	foreign := handleOfActor(other)
	got := f.selfPost(a, "sessions/end", endJSON(listed, []string{handleOfActor(b), handleOfActor(c), foreign}))
	if got.status != 202 {
		t.Fatalf("%+v", got)
	}
	want := map[string]string{handleOfActor(b): resultEnded, handleOfActor(c): resultGone, foreign: resultGone}
	if fmt.Sprint(resultsOf(got.body)) != fmt.Sprint(want) {
		t.Errorf("results %v", resultsOf(got.body))
	}
	if fmt.Sprint(notIncludedOf(got.body)) != fmt.Sprint([]string{handleOfActor(d)}) {
		t.Errorf("not included %v", notIncludedOf(got.body))
	}
	if got.body["generation_changed"] != true || got.body["list_generation"] != listed || got.body["current_generation"] == listed || fmt.Sprint(got.body["count"]) != "1" {
		t.Errorf("generations %+v", got.body)
	}
	if got.body["current_generation"] != f.generationOf(a) {
		t.Error("current generation is not the live token")
	}
	f.code(t, "b ended", f.me(b), 401, string(commands.Unauthenticated))
	if f.me(d).status != 200 || f.me(other).status != 200 || f.me(a).status != 200 {
		t.Error("a session outside the confirmed list ended")
	}
	cleanup := got.body["cleanup"].(map[string]any)
	if fmt.Sprint(cleanup["done"], cleanup["already_ended"], cleanup["failed"], cleanup["pending"]) != "1 0 0 0" {
		t.Errorf("cleanup %v", cleanup)
	}
	ends := f.me(a).body["session_ends"].([]any)
	if len(ends) != 1 || ends[0].(map[string]any)["operation"] != got.body["operation"] {
		t.Errorf("read ends %v", ends)
	}
}

func TestSelfEndUnchangedListIsNotFlagged(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	got := f.selfPost(a, "sessions/end", f.othersBody(a))
	if got.status != 202 || got.body["generation_changed"] != false || len(got.body["not_included"].([]any)) != 0 || resultsOf(got.body)[handleOfActor(b)] != resultEnded {
		t.Fatalf("%+v", got)
	}
}

func TestSelfEndReplay(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	f.as(MemberLogin)
	body := f.othersBody(a)
	first := f.selfPost(a, "sessions/end", body)
	newcomer := f.as(MemberLogin)
	before := f.s.counter
	again := f.selfPost(a, "sessions/end", body)
	if again.status != 202 || again.body["replay"] != true || again.body["operation"] != first.body["operation"] || f.s.counter != before {
		t.Fatalf("replay %+v", again)
	}
	if f.me(newcomer).status != 200 || len(f.s.selfEnds[SampleMember]) != 1 {
		t.Error("a replay ended a session or created an operation")
	}
	f.advance(stepUpDefault + time.Nanosecond)
	f.code(t, "replay without step-up", f.selfPost(a, "sessions/end", body), 403, CodeReauthentication)
}

func TestSelfEndOlderRepeatEndsNothingNew(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	body := f.othersBody(a)
	f.selfPost(a, "sessions/end", body)
	f.selfPost(a, "sessions/end", endJSON(f.generationOf(a), []string{"00000000000000aa"}))
	newcomer := f.as(MemberLogin)
	got := f.selfPost(a, "sessions/end", body)
	if got.status != 202 || got.body["replay"] != false || resultsOf(got.body)[handleOfActor(b)] != resultGone || fmt.Sprint(got.body["count"]) != "0" {
		t.Fatalf("%+v", got)
	}
	if f.me(newcomer).status != 200 || fmt.Sprint(notIncludedOf(got.body)) != fmt.Sprint([]string{handleOfActor(newcomer)}) {
		t.Error("the set was extended")
	}
	if got.body["state"] != stateComplete {
		t.Errorf("state %v", got.body["state"])
	}
}

func TestSelfEndFalsePathChangesNothing(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	body := f.othersBody(a)
	f.s.selfEndHook = func() { f.s.sessions.drop(verifierOfActor(a)) }
	before, gen := f.s.counter, f.s.sessions.generation(MemberLogin)
	got := f.selfPost(a, "sessions/end", body)
	f.s.selfEndHook = nil
	f.code(t, "own session gone", got, 401, string(commands.Unauthenticated))
	if f.s.counter != before || len(f.s.selfEnds) != 0 || f.me(b).status != 200 {
		t.Error("the false path changed state")
	}
	if f.s.sessions.generation(MemberLogin) <= gen || len(f.s.retries) != 0 {
		t.Error("the false path removed or denied a session")
	}
}

func TestSelfEndKindOneHasNoHeldList(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	f.as(MemberLogin)
	got := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(b)+`"}`)
	if got.status != 202 || got.body["kind"] != kindOne || len(got.body["not_included"].([]any)) != 0 || got.body["generation_changed"] != false || got.body["list_generation"] != "" {
		t.Fatalf("%+v", got)
	}
}

func TestSelfEndTokenIsOpaqueAndPerLogin(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	first := f.generationOf(a)
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(first) {
		t.Fatalf("token %q", first)
	}
	other := f.as(AdminLogin)
	f.as(AdminLogin)
	f.selfPost(other, "sessions/end", f.othersBody(other))
	f.logout(other)
	if f.generationOf(a) != first {
		t.Error("another login's activity changed the caller's token")
	}
	b := f.as(MemberLogin)
	second := f.generationOf(a)
	f.logout(b)
	third := f.generationOf(a)
	if second == first || third == second || third == first {
		t.Errorf("token did not change: %s %s %s", first, second, third)
	}
	f.reauth(a, pw)
	if f.generationOf(a) != third {
		t.Error("a reauthentication changed the token")
	}
}

func TestSessionTokensNeverRepeat(t *testing.T) {
	store := newSessions()
	seen := map[string]bool{}
	last := uint64(0)
	now := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 10000; i++ {
		v := verifierOf(random(tokenBytes))
		if _, err := store.put(v, "w", "x@example.invalid", now, 1); err != nil {
			t.Fatal(err)
		}
		snap := store.snapshot("x@example.invalid")
		if seen[snap.token] || snap.counter <= last {
			t.Fatalf("repeat at %d", i)
		}
		seen[snap.token], last = true, snap.counter
		store.drop(v)
		if store.snapshot("x@example.invalid").token != "" || store.generation("x@example.invalid") != 0 {
			t.Fatal("an empty login kept its generation")
		}
	}
	v := verifierOf(random(tokenBytes))
	_, _ = store.put(v, "w", "y@example.invalid", now, 1)
	before := store.generation("y@example.invalid")
	store.prune(now.Add(time.Second))
	if store.generation("y@example.invalid") != 0 || before == 0 {
		t.Error("prune did not change the generation")
	}
}

func TestSessionListOrder(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	f.advance(time.Second)
	b := f.as(MemberLogin)
	c := f.as(MemberLogin)
	f.advance(time.Second)
	d := f.as(MemberLogin)
	same := []string{handleOfActor(b), handleOfActor(c)}
	sort.Strings(same)
	want := []string{handleOfActor(a), handleOfActor(d), same[0], same[1]}
	got := []string{}
	for _, row := range f.me(a).body["sessions"].([]any) {
		got = append(got, row.(map[string]any)["handle"].(string))
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("order %v want %v", got, want)
	}
	end := f.selfPost(a, "sessions/end", f.othersBody(a))
	rows := []string{}
	for _, row := range end.body["sessions"].([]any) {
		rows = append(rows, row.(map[string]any)["handle"].(string))
	}
	if fmt.Sprint(rows) != fmt.Sprint(want[1:]) {
		t.Errorf("operation order %v want %v", rows, want[1:])
	}
}

func TestOpenAPISelfPaths(t *testing.T) {
	f := memberWorld(t)
	raw, err := f.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	self := doc["x-self"].(map[string]any)
	post := self["/me/sessions/end"].(map[string]any)["post"].(map[string]any)
	if post["x-step-up"] != true || post["responses"].(map[string]any)["202"] == nil {
		t.Errorf("post %v", post)
	}
	read := self["/me"].(map[string]any)["get"].(map[string]any)
	want := map[string]map[string]string{
		"post": {"404": "unsupported_command|[unsupported_command]|none", "409": "revision_conflict|[revision_conflict]|inspect_current_state", "429": "rate_limited|[rate_limited]|retry_later", "401": "unauthenticated|[unauthenticated]|obtain_new_access_answer", "403": "reauthentication_required|[csrf_refused reauthentication_required]|reauthenticate",
			"413": "body_too_large|[body_too_large]|none", "422": "invalid_request|[invalid_request prerequisite_failed]|correct_request",
			"500": "internal_error|[internal_error]|none", "503": "authority_unavailable|[authority_unavailable]|retry_later"},
		"get": {"401": "unauthenticated|[unauthenticated]|obtain_new_access_answer", "403": "csrf_refused|[csrf_refused]|none",
			"500": "internal_error|[internal_error]|none", "503": "authority_unavailable|[authority_unavailable]|retry_later"},
	}
	for method, op := range map[string]map[string]any{"post": post, "get": read} {
		responses := op["responses"].(map[string]any)
		for status, line := range want[method] {
			entry, ok := responses[status].(map[string]any)
			if !ok {
				t.Errorf("%s %s missing", method, status)
				continue
			}
			if got := fmt.Sprint(entry["x-code"], "|", entry["x-codes"], "|", entry["x-next-action"]); got != line {
				t.Errorf("%s %s = %s want %s", method, status, got, line)
			}
		}
		for status, raw := range responses {
			entry, _ := raw.(map[string]any)
			if status == "202" || status == "200" {
				continue
			}
			actions := fmt.Sprint(entry["x-next-actions"])
			for _, code := range entry["x-codes"].([]any) {
				if !strings.Contains(actions, fmt.Sprint(code)+":") {
					t.Errorf("%s %s: no next action for %v in %s", method, status, code, actions)
				}
			}
		}
		if method == "post" {
			actions := responses["403"].(map[string]any)["x-next-actions"].(map[string]any)
			if actions[CodeCSRF] != "none" || actions[CodeReauthentication] != "reauthenticate" {
				t.Errorf("403 next actions %v", actions)
			}
			if actions := responses["422"].(map[string]any)["x-next-actions"].(map[string]any); actions["prerequisite_failed"] != "correct_request" {
				t.Errorf("422 next actions %v", actions)
			}
		}
		if len(responses) != len(want[method])+1 {
			t.Errorf("%s has %d responses", method, len(responses))
		}
	}
	props := read["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)
	if props["sessions_generation"] == nil || props["session_ends"] == nil {
		t.Errorf("read %v", props)
	}
	schema := selfEndSchema()["properties"].(map[string]any)
	view, _ := json.Marshal(selfEndView{})
	var fields map[string]any
	_ = json.Unmarshal(view, &fields)
	for name := range fields {
		if schema[name] == nil {
			t.Errorf("answer field %s missing from the schema", name)
		}
	}
	if len(schema) != len(fields) {
		t.Errorf("schema %d fields, answer %d", len(schema), len(fields))
	}
}

func TestSelfEndSameHandlesUnderANewGenerationIsNotAReplay(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	hb := handleOfActor(b)
	first := f.selfPost(a, "sessions/end", endJSON(f.generationOf(a), []string{hb}))
	f.as(MemberLogin)
	got := f.selfPost(a, "sessions/end", endJSON(f.generationOf(a), []string{hb}))
	if got.status != 202 || got.body["replay"] != false || got.body["operation"] == first.body["operation"] || resultsOf(got.body)[hb] != resultGone {
		t.Fatalf("%+v", got)
	}
}
