package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func queryActors(t *testing.T) (world, actor, actor, actor) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	old := f.signedIn(MemberLogin, pw)
	f.advance(time.Second)
	middle := f.signedIn(MemberLogin, pw)
	f.advance(time.Second)
	current := f.signedIn(MemberLogin, pw)
	return f, old, middle, current
}

func queryCursor(t *testing.T, f world, a actor) string {
	t.Helper()
	got := f.sessionQuery(a, "?session=other&limit=1")
	next, _ := got.body["next"].(string)
	if got.status != 200 || next == "" {
		t.Fatalf("cannot obtain bounded continuation: status %d", got.status)
	}
	return next
}

func TestSessionQueryRejectsMalformedQueryEncoding(t *testing.T) {
	f, _, _, a := queryActors(t)
	for _, query := range []string{"?session=other;ignored=x", "?session=%ZZ"} {
		got := f.sessionQuery(a, query)
		if got.status != http.StatusUnprocessableEntity || got.body["rows"] != nil {
			t.Errorf("malformed query status %d disclosed=%t", got.status, got.body["rows"] != nil)
		}
	}
}

func TestSessionQueryStrictQueryValues(t *testing.T) {
	f, _, _, a := queryActors(t)
	for _, query := range []string{"?session=invalid", "?limit=0", "?limit=101", "?limit=1&limit=2", "?q=", "?q=%FF", "?q=%00", "?since=tomorrow", "?until=2026-11-16T00%3A00%3A00Z&since=2026-11-17T00%3A00%3A00Z", "?tenant=another", "?workspace="} {
		got := f.sessionQuery(a, query)
		if got.status != 422 || got.body["rows"] != nil {
			t.Errorf("invalid query %q status %d", query, got.status)
		}
	}
}

func TestSessionQueryCursorBindsCallerAndNormalizedQuery(t *testing.T) {
	f, old, _, a := queryActors(t)
	next := queryCursor(t, f, a)
	for _, test := range []struct {
		a     actor
		query string
	}{{old, "?session=other&limit=1"}, {a, "?session=all&limit=1"}, {a, "?session=other&limit=2"}, {a, "?session=other&limit=1&q=not-present"}} {
		got := f.sessionQuery(test.a, test.query+"&after="+url.QueryEscape(next))
		if got.status != 409 || got.body["rows"] != nil {
			t.Errorf("mismatched continuation status %d", got.status)
		}
	}
}

func TestSessionQueryCursorIntegrityAndProcessBoundary(t *testing.T) {
	f, _, _, a := queryActors(t)
	next := queryCursor(t, f, a)
	parts := strings.Split(next, ".")
	body, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		t.Fatal("invalid cursor fixture")
	}
	payload["matched"] = 9000
	changed, _ := json.Marshal(payload)
	tampered := base64.RawURLEncoding.EncodeToString(changed) + "." + parts[1]
	for _, token := range []string{tampered, "bad", "a.b.c"} {
		if got := f.sessionQuery(a, "?session=other&limit=1&after="+url.QueryEscape(token)); got.status != 409 {
			t.Errorf("invalid continuation status %d", got.status)
		}
	}
	f.s.sessionCursorKey = random(32)
	if got := f.sessionQuery(a, "?session=other&limit=1&after="+url.QueryEscape(next)); got.status != 409 {
		t.Errorf("old-process cursor status %d", got.status)
	}
}

func TestSessionQueryCursorExpiresAndDetectsIndexChanges(t *testing.T) {
	for _, change := range []string{"expiry", "reauthentication", "denial", "domain"} {
		t.Run(change, func(t *testing.T) {
			f, old, _, a := queryActors(t)
			next := queryCursor(t, f, a)
			switch change {
			case "expiry":
				f.advance(301 * time.Second)
			case "reauthentication":
				f.s.sessions.reauthenticated(verifierOfActor(old), *f.now)
			case "denial":
				f.s.sessions.deny(verifierOfActor(old), MemberLogin, *f.now)
			case "domain":
				f.s.counter++
			}
			got := f.sessionQuery(a, "?session=other&limit=1&after="+url.QueryEscape(next))
			if got.status != 409 || got.body["rows"] != nil {
				t.Fatalf("invalidated continuation status %d", got.status)
			}
		})
	}
}

func TestSessionQueryDoesNotRefreshOtherIdleDeadlines(t *testing.T) {
	f, old, _, a := queryActors(t)
	_, before, err := f.s.identity.Boundary().Peek(verifierOfActor(old), SampleWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	f.advance(5 * time.Minute)
	if got := f.sessionQuery(a, "?session=other&limit=1&q="+handleOfActor(old)); got.status != 200 {
		t.Fatal(got.status)
	}
	_, after, err := f.s.identity.Boundary().Peek(verifierOfActor(old), SampleWorkspace)
	if err != nil || !after.Equal(before) {
		t.Fatal("viewing other sessions extended their idle deadline")
	}
}

func TestSessionQueryIdleBoundaryExpiresContinuation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.advance(sessionIdle - time.Second)
	a := f.signedIn(MemberLogin, pw)
	next := queryCursor(t, f, a)
	f.advance(2 * time.Second)
	if got := f.sessionQuery(a, "?session=other&limit=1&after="+url.QueryEscape(next)); got.status != 409 {
		t.Fatalf("target idle boundary status %d", got.status)
	}
}

func TestSessionQueryReportsIdleExpiryTruthfully(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	old := f.signedIn(MemberLogin, pw)
	f.advance(sessionIdle - time.Second)
	a := f.signedIn(MemberLogin, pw)
	f.advance(2 * time.Second)
	got := f.sessionQuery(a, "?q="+handleOfActor(old))
	if got.status != 200 {
		t.Fatal(got.status)
	}
	rows, _ := got.body["rows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["state"] != "expired" || rows[0].(map[string]any)["can_end"] != false {
		t.Fatalf("idle-expired row %v", got.body["rows"])
	}
}

func TestSessionQueryPinsCurrentWithoutRepeatingAndKeepsStableTies(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	query := "?limit=1"
	seen := map[string]bool{}
	last := ""
	for n := 0; n < 6; n++ {
		got := f.sessionQuery(a, query)
		if got.status != 200 {
			t.Fatal(got.status)
		}
		rows := got.body["rows"].([]any)
		for _, raw := range rows {
			row := raw.(map[string]any)
			handle := row["handle"].(string)
			if seen[handle] {
				t.Fatal("cursor repeated a row")
			}
			seen[handle] = true
			if n == 0 && row["current"] != true {
				t.Fatal("current session was not first")
			}
			if n > 0 {
				if last != "" && handle < last {
					t.Fatal("equal-time handles not stably ordered")
				}
				last = handle
			}
		}
		next, _ := got.body["next"].(string)
		if next == "" {
			if got.body["count"].(map[string]any)["certainty"] != "exact" {
				t.Fatal("complete query count not exact")
			}
			break
		}
		query = "?limit=1&after=" + url.QueryEscape(next)
	}
	if len(seen) != 3 {
		t.Fatalf("saw %d sessions, want3", len(seen))
	}
}

func TestSessionQueryPreservesFractionalTimeBounds(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.advance(800 * time.Millisecond)
	f.signedIn(MemberLogin, pw)
	f.advance(400 * time.Millisecond)
	a := f.signedIn(MemberLogin, pw)
	bound := start.Add(900 * time.Millisecond).Format(time.RFC3339Nano)
	got := f.sessionQuery(a, "?session=other&since="+url.QueryEscape(bound))
	if got.status != 200 {
		t.Fatal(got.status)
	}
	if rows := got.body["rows"].([]any); len(rows) != 0 {
		t.Fatalf("fractional lower bound widened to %d rows", len(rows))
	}
	if got.body["query"].(map[string]any)["since"] != bound {
		t.Fatal("normalized bound lost precision")
	}
}

func TestSessionQueryPolicyEqualityTransitionInvalidatesCursor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.applyPolicy("5", "5")
	f.setPassword(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	f.advance(5 * time.Minute)
	a := f.signedIn(MemberLogin, pw)
	next := queryCursor(t, f, a)
	f.advance(time.Nanosecond)
	if got := f.sessionQuery(a, "?session=other&limit=1&after="+url.QueryEscape(next)); got.status != 409 {
		t.Fatalf("policy equality transition status %d", got.status)
	}
}

func TestSessionQueryKnownIDIndependentOfDisplayedPage(t *testing.T) {
	f, old, _, a := queryActors(t)
	page := f.sessionQuery(a, "?session=current&limit=1")
	if page.status != 200 {
		t.Fatal(page.status)
	}
	path := "/api/v1/me/sessions/" + handleOfActor(old)
	got := f.do(http.MethodGet, path, "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(csrfHeader, a.csrf) })
	if got.status != 200 {
		t.Fatalf("known-ID detail status %d, want200", got.status)
	}
	row, _ := got.body["session"].(map[string]any)
	if row["handle"] != handleOfActor(old) || row["can_end"] != true {
		t.Fatalf("known-ID eligibility %v", row)
	}
}

func TestSessionQueryKnownIDRequiresAdmissionAndHidesForeignTargets(t *testing.T) {
	f, _, _, a := queryActors(t)
	f.setPassword(AdminLogin, pw)
	foreign := f.signedIn(AdminLogin, pw)
	path := "/api/v1/me/sessions/" + handleOfActor(foreign)
	missing := f.do(http.MethodGet, path, "", nil)
	if missing.status != 401 {
		t.Errorf("anonymous known-ID status %d", missing.status)
	}
	hidden := f.do(http.MethodGet, path, "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(csrfHeader, a.csrf) })
	if hidden.status != 404 || hidden.body["session"] != nil {
		t.Errorf("foreign known-ID status %d", hidden.status)
	}
}
