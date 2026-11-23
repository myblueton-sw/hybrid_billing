package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSessionSelectionResumeRejectsForeignCallerAndInvalidCursor(t *testing.T) {
	f, other, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	body, _ := json.Marshal(map[string]any{"selection": prepared.body["selection"], "after": "invalid-cursor"})
	if got := f.selfPost(other, "sessions/selections", string(body)); got.status != 404 {
		t.Fatal("foreign caller resumed a selection", got.status)
	}
	if got := f.selfPost(caller, "sessions/selections", string(body)); got.status != 409 {
		t.Fatal("invalid continuation was not rejected", got.status)
	}
	if got := f.selfPost(actor{}, "sessions/selections", string(body)); got.status != 401 {
		t.Fatal("anonymous selection resume", got.status)
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionPendingCapacityRefusesBeforeChangingRecordsOrSessions(t *testing.T) {
	f, _, _, caller := queryActors(t)
	for i := 0; i < 2; i++ {
		if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 200 {
			t.Fatal("initial pending view", got.status)
		}
		f.signedIn(MemberLogin, pw)
	}
	units, records, revision := f.s.sessionRecordUnits, len(f.s.sessionRecords), f.s.Revision()
	if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 429 {
		t.Fatal("pending cap not enforced", got.status)
	}
	if f.s.sessionRecordUnits != units || len(f.s.sessionRecords) != records || f.s.Revision() != revision || len(f.s.sessionOperations) != 0 {
		t.Fatal("pending refusal changed capacity or effects")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionTargetQuotaRollbackKeepsEverySession(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 200
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	caller := f.signedIn(MemberLogin, pw)
	var last actor
	for i := 0; i < 65; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		last = actor{cookie, csrf}
	}
	revision := f.s.Revision()
	if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 429 {
		t.Fatal("target quota not enforced", got.status)
	}
	if f.s.sessionRecordUnits != 0 || len(f.s.sessionRecords) != 0 || len(f.s.sessionOperations) != 0 || f.s.Revision() != revision || f.me(last).status != 200 {
		t.Fatal("failed preparation retained capacity or changed a target")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionGlobalBuildQuotaRollsBackOnlyRefusedView(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 200
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	actors := []actor{f.signedIn(MemberLogin, pw)}
	for i := 0; i < 64; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		if i < 3 {
			actors = append(actors, actor{cookie, csrf})
		}
	}
	for _, caller := range actors[:3] {
		if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 200 {
			t.Fatal("charged view setup", got.status)
		}
	}
	units, records, revision := f.s.sessionRecordUnits, len(f.s.sessionRecords), f.s.Revision()
	if units != 195 || records != 3 {
		t.Fatal("unexpected charged baseline", units, records)
	}
	if got := f.selfPost(actors[3], "sessions/selections", "{}"); got.status != 429 {
		t.Fatal("global build quota not enforced", got.status)
	}
	if f.s.sessionRecordUnits != units || len(f.s.sessionRecords) != records || f.s.Revision() != revision || len(f.s.sessionOperations) != 0 {
		t.Fatal("failed build damaged retained views or effects")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionPartialTargetQuotaRollsBackFrozenProgress(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 200
		c.Access = DefaultAccessSettings()
		c.Access.SessionQueryScan = 100
		c.Access.SessionSelectionTargets = 128
		c.Access.SessionRecordBudget = 512
	})
	f.setPassword(MemberLogin, pw)
	caller := f.signedIn(MemberLogin, pw)
	for i := 0; i < 130; i++ {
		if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
			t.Fatal(err)
		}
	}
	first := f.selfPost(caller, "sessions/selections", "{}")
	if first.status != 200 || first.body["state"] != "building" || f.s.sessionRecordUnits < 100 {
		t.Fatal("partial preparation did not charge progress", first.status)
	}
	body, _ := json.Marshal(map[string]any{"selection": first.body["selection"], "after": first.body["after"]})
	if got := f.selfPost(caller, "sessions/selections", string(body)); got.status != 429 {
		t.Fatal("partial target limit not enforced", got.status)
	}
	if f.s.sessionRecordUnits != 0 || len(f.s.sessionRecords) != 0 || len(f.s.sessionOperations) != 0 {
		t.Fatal("partial target quota left frozen charges")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionPartialGlobalQuotaPreservesOtherChargedViews(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 500
		c.Access = DefaultAccessSettings()
		c.Access.SessionQueryScan = 100
		c.Access.SessionSelectionTargets = 200
		c.Access.SessionRecordBudget = 768
	})
	f.setPassword(MemberLogin, pw)
	actors := []actor{f.signedIn(MemberLogin, pw)}
	for i := 0; i < 200; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		if i < 6 {
			actors = append(actors, actor{cookie, csrf})
		}
	}
	var first reply
	for i, caller := range actors {
		got := f.selfPost(caller, "sessions/selections", "{}")
		if got.status != 200 || got.body["state"] != "building" {
			t.Fatal("partial capacity setup", got.status)
		}
		if i == 0 {
			first = got
		}
	}
	record := f.s.sessionRecords[first.body["selection"].(string)]
	expected := f.s.sessionRecordUnits - record.units
	body, _ := json.Marshal(map[string]any{"selection": first.body["selection"], "after": first.body["after"]})
	if got := f.selfPost(actors[0], "sessions/selections", string(body)); got.status != 429 {
		t.Fatal("partial global quota not enforced", got.status)
	}
	if f.s.sessionRecordUnits != expected || len(f.s.sessionRecords) != 6 || len(f.s.sessionOperations) != 0 {
		t.Fatal("partial global quota damaged other views")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionPreparationRejectsQueryAndMalformedContinuationFields(t *testing.T) {
	f, _, _, caller := queryActors(t)
	for _, body := range []string{`{"selection":"s","after":null}`, `{"selection":"s","after":true}`, `{"selection":"s","after":""}`, `{"selection":"s","after":"` + strings.Repeat("x", 4097) + `"}`} {
		if got := f.selfPost(caller, "sessions/selections", body); got.status != 422 {
			t.Fatal("malformed continuation accepted", got.status)
		}
	}
	if got := f.selfPost(caller, "sessions/selections?unknown=1", "{}"); got.status != 422 {
		t.Fatal("preparation accepted query parameters", got.status)
	}
	if got := f.selfPost(caller, "sessions/selections", `{"after":"`+strings.Repeat("x", 8193)+`"}`); got.status != 413 {
		t.Fatal("preparation body limit missing", got.status)
	}
	if len(f.s.sessionRecords) != 0 || f.s.sessionRecordUnits != 0 {
		t.Fatal("invalid input allocated preparation state")
	}
}

func TestSessionSelectionCannotCreateRecordAtGlobalCapacity(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 100
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	actors := []actor{f.signedIn(MemberLogin, pw)}
	for i := 0; i < 63; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		actors = append(actors, actor{cookie, csrf})
	}
	for _, caller := range actors[:4] {
		if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 200 {
			t.Fatal("capacity setup", got.status)
		}
	}
	if f.s.sessionRecordUnits != 256 {
		t.Fatal("expected full capacity")
	}
	if got := f.selfPost(actors[4], "sessions/selections", "{}"); got.status != 429 {
		t.Fatal("record capacity not enforced", got.status)
	}
	if f.s.sessionRecordUnits != 256 || len(f.s.sessionRecords) != 4 || len(f.s.sessionOperations) != 0 {
		t.Fatal("record creation failure changed retained views")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionSingleTargetRollsBackAtLastGlobalUnit(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 100
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	actors := []actor{f.signedIn(MemberLogin, pw)}
	for i := 0; i < 62; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		actors = append(actors, actor{cookie, csrf})
	}
	if got := f.selfPost(actors[0], "sessions/selections", "{}"); got.status != 200 {
		t.Fatal(got.status)
	}
	f.signedIn(MemberLogin, pw)
	for _, caller := range actors[1:4] {
		if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 200 {
			t.Fatal(got.status)
		}
	}
	if f.s.sessionRecordUnits != 255 {
		t.Fatal("expected one remaining capacity unit", f.s.sessionRecordUnits)
	}
	if got := f.selfPost(actors[4], "sessions/selections", `{"session":"`+handleOfActor(actors[0])+`"}`); got.status != 429 {
		t.Fatal("single target capacity failure missing", got.status)
	}
	if f.s.sessionRecordUnits != 255 || len(f.s.sessionRecords) != 4 || len(f.s.sessionOperations) != 0 {
		t.Fatal("single-target failure leaked its record charge")
	}
	assertSessionRecordAccounting(t, f)
}
