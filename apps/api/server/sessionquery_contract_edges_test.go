package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSessionQueryExplicitWorkspaceAndExclusiveTimeWindow(t *testing.T) {
	f, old, middle, current := queryActors(t)
	since := start.Add(time.Second).UTC().Format(time.RFC3339Nano)
	until := start.Add(2 * time.Second).UTC().Format(time.RFC3339Nano)
	for _, selected := range []string{"all", "other", "current"} {
		values := url.Values{"workspace": {SampleWorkspace}, "session": {selected}, "since": {since}, "until": {until}, "limit": {"1"}}
		want := 1
		if selected == "current" {
			want = 0
		}
		seen := []string{}
		complete := false
		for page := 0; page < 5; page++ {
			got := f.sessionQuery(current, "?"+values.Encode())
			if got.status != 200 || got.body["query"].(map[string]any)["workspace"] != SampleWorkspace {
				t.Fatal("matching authenticated workspace query was not retained", got.status)
			}
			rows := got.body["rows"].([]any)
			if len(rows) > 1 {
				t.Fatal("one-row limit exceeded")
			}
			for _, row := range rows {
				seen = append(seen, row.(map[string]any)["handle"].(string))
			}
			count := got.body["count"].(map[string]any)
			if countOf(count["value"]) != len(seen) {
				t.Fatal("cumulative matches differ from traversed set")
			}
			next := got.body["next"].(string)
			if next == "" {
				if len(seen) != want || count["certainty"] != "exact" {
					t.Fatal("exclusive time window changed final matching set", selected)
				}
				complete = true
				break
			}
			if count["certainty"] != "lower_bound" || next == values.Get("after") {
				t.Fatal("unfinished bounded query has invalid certainty or position")
			}
			values.Set("after", next)
		}
		if !complete {
			t.Fatal("bounded time-window traversal did not terminate")
		}
		if want == 1 && seen[0] != handleOfActor(middle) {
			t.Fatal("time window included a boundary-excluded session")
		}
	}
	oldSession, oldKnown := f.s.sessions.get(verifierOfActor(old))
	currentSession, currentKnown := f.s.sessions.get(verifierOfActor(current))
	if !oldKnown || !currentKnown {
		t.Fatal("real session matcher premises missing")
	}
	if (sessionQuery{Session: "current"}).matches(oldSession, verifierOfActor(current)) || (sessionQuery{Session: "other"}).matches(currentSession, verifierOfActor(current)) {
		t.Fatal("pure matcher ignored current/other exclusion")
	}
}

func TestSessionQueryMalformedBoundsAndKnownQueryHaveNoEffects(t *testing.T) {
	f, old, middle, current := queryActors(t)
	before, records, units := f.s.Revision(), len(f.s.sessionRecords), f.s.sessionRecordUnits
	for _, query := range []string{"?since=" + strings.Repeat("2", 41), "?until=" + strings.Repeat("2", 41), "?after=" + strings.Repeat("x", 4097)} {
		got := f.sessionQuery(current, query)
		if got.status != 422 || got.body["rows"] != nil || got.body["count"] != nil {
			t.Fatal("malformed bounds disclosed session data", got.status)
		}
	}
	for _, suffix := range []string{"?unexpected=1", "?q=%ZZ"} {
		got := f.do(http.MethodGet, "/api/v1/me/sessions/"+handleOfActor(old)+suffix, "", func(r *http.Request) {
			r.AddCookie(current.cookie)
			r.Header.Set(csrfHeader, current.csrf)
		})
		if got.status != 422 || got.body["session"] != nil || got.body["rows"] != nil {
			t.Fatal("invalid known-session query disclosed metadata", got.status)
		}
	}
	if f.s.Revision() != before || len(f.s.sessionRecords) != records || f.s.sessionRecordUnits != units || f.me(old).status != 200 || f.me(middle).status != 200 {
		t.Fatal("refused reads changed operations, accounting or peer authority")
	}
}
