package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func selectionResumeBody(prepared reply) string {
	raw, _ := json.Marshal(map[string]any{"selection": prepared.body["selection"], "after": prepared.body["after"]})
	return string(raw)
}

func selectionPopulation(t *testing.T, total int) (world, actor) {
	t.Helper()
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 500
		c.Access = DefaultAccessSettings()
		c.Access.SessionQueryScan = 100
	})
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	for i := 0; i < total; i++ {
		if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
			t.Fatal(err)
		}
	}
	return f, a
}

func assertSessionRecordAccounting(t *testing.T, f world) {
	t.Helper()
	units := 0
	for _, record := range f.s.sessionRecords {
		units += record.units
		actual := 1 + len(record.targets)
		if record.op != nil {
			actual += len(record.op.rows) + len(record.op.notIncluded) + record.retries
		}
		if record.state != "applying" && actual != record.units {
			t.Fatalf("record charged%d,actual%d", record.units, actual)
		}
		retries := 0
		for _, retry := range f.s.retries {
			if retry.op.record == record {
				retries++
			}
		}
		if retries != record.retries {
			t.Fatal("retry charges differ from outstanding records")
		}
	}
	if units != f.s.sessionRecordUnits || units > f.s.cfg.Access.SessionRecordBudget {
		t.Fatalf("global reference accounting%d, recorded%d", f.s.sessionRecordUnits, units)
	}
}

func TestSessionSelectionBuildsAndReplaysContinuationWithoutDuplicateTargets(t *testing.T) {
	f, a := selectionPopulation(t, 350)
	first := f.selfPost(a, "sessions/selections", "{}")
	if first.status != 200 || first.body["state"] != "building" || first.body["count"] != nil || first.body["after"] == "" {
		t.Fatal("partial build claimed a complete set")
	}
	ref := first.body["selection"]
	second := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	if second.status != 200 || second.body["state"] != "building" {
		t.Fatal("bounded build did not advance")
	}
	repeated := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	if repeated.status != 200 || repeated.body["observed"] != second.body["observed"] || repeated.body["after"] != second.body["after"] {
		t.Fatal("replayed build appended targets")
	}
	current := second
	for n := 0; n < 5 && current.body["state"] == "building"; n++ {
		current = f.selfPost(a, "sessions/selections", selectionResumeBody(current))
		if current.status != 200 {
			t.Fatal(current.status)
		}
	}
	if current.body["state"] != "sealed" || countOf(current.body["count"]) != 350 {
		t.Fatal("build omitted or duplicated targets")
	}
	same := f.selfPost(a, "sessions/selections", "{}")
	if same.status != 200 || same.body["selection"] != ref || len(f.s.sessionRecords) != 1 {
		t.Fatal("identical preparation allocated a second record")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionBuildingRejectsChangedSource(t *testing.T) {
	f, a := selectionPopulation(t, 150)
	first := f.selfPost(a, "sessions/selections", "{}")
	if first.status != 200 {
		t.Fatal(first.status)
	}
	if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
		t.Fatal(err)
	}
	changed := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	if changed.status != 409 || len(f.s.selfEnds) != 0 || f.s.sessionRecordUnits != 0 {
		t.Fatal("changed build continued or applied effects")
	}
}

func TestSessionSelectionExpiredBeforeAcceptanceNeverApplies(t *testing.T) {
	f, old, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal(prepared.status)
	}
	f.advance(301 * time.Second)
	f.s.sessions.reauthenticated(verifierOfActor(a), *f.now)
	expired := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if expired.status != 409 || f.me(old).status != 200 || f.s.sessionRecordUnits != 0 || len(f.s.selfEnds) != 0 {
		t.Fatal("expired selection applied or retained a target set")
	}
}

func TestSessionSelectionStrictBodyAndForeignPreview(t *testing.T) {
	f, other, _, a := queryActors(t)
	for _, body := range []string{`null`, `[]`, `{"unknown":true}`, `{"selection":null}`, `{"after":"x"}`, `{"selection":"x","after":"a","after":"b"}`} {
		if got := f.selfPost(a, "sessions/selections", body); got.status != 422 {
			t.Errorf("invalid preparation status%d", got.status)
		}
	}
	prepared := f.selfPost(a, "sessions/selections", "{}")
	path := "/api/v1/me/sessions/selections/" + prepared.body["selection"].(string)
	foreign := f.do(http.MethodGet, path, "", func(r *http.Request) { r.AddCookie(other.cookie); r.Header.Set(csrfHeader, other.csrf) })
	if foreign.status != 404 || foreign.body["rows"] != nil {
		t.Fatal("foreign preview disclosed targets")
	}
	var values map[string]any
	_ = json.Unmarshal([]byte(selectionEndBody(prepared)), &values)
	values["generation"] = nil
	mixed, _ := json.Marshal(values)
	if got := f.selfPost(a, "sessions/end", string(mixed)); got.status != 422 || f.me(other).status != 200 {
		t.Fatal("mixed null variant was accepted")
	}
}
