package server

import (
	"testing"
	"time"
)

func TestSessionSelectionCapacityEvictsSettledRecordsAndReleasesAllIndexes(t *testing.T) {
	assertSessionSelectionPressure(t, false)
}

func TestSessionSelectionChronologicalPressureEvictsOldestSettledOperation(t *testing.T) {
	assertSessionSelectionPressure(t, true)
}

func assertSessionSelectionPressure(t *testing.T, advance bool) {
	t.Helper()
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 500
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	firstRef, firstOperation := "", ""
	for round := 0; round < 6; round++ {
		if advance {
			f.advance(time.Second)
		}
		for i := 0; i < 20; i++ {
			if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
				t.Fatal(err)
			}
		}
		prepared := f.selfPost(a, "sessions/selections", "{}")
		if prepared.status != 200 {
			t.Fatal(prepared.status)
		}
		accepted := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
		if accepted.status != 202 {
			t.Fatal(accepted.status)
		}
		if round == 0 {
			firstRef = prepared.body["selection"].(string)
			firstOperation = accepted.body["operation"].(string)
		}
		assertSessionRecordAccounting(t, f)
	}
	if f.s.sessionRecordUnits != 205 || len(f.s.sessionRecords) != 5 || len(f.s.sessionOperations) != 5 {
		t.Fatal("settled retention did not release exact references")
	}
	if got := f.selectionRead(a, firstRef); got.status != 404 {
		t.Fatal("evicted selection could be reconstructed")
	}
	if got := f.operationRead(a, "/"+firstOperation); got.status != 404 {
		t.Fatal("evicted operation remained indexed")
	}
	list := f.operationRead(a, "?limit=100")
	if list.status != 200 || countOf(list.body["accepted_count"]) != 5 || len(list.body["rows"].([]any)) != 5 {
		t.Fatal("operation index retained evicted rows")
	}
}
