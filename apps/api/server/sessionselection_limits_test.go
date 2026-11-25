package server

import (
	"testing"
)

func TestSessionSelectionCapacityFailsBeforeAnyEffect(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 100
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	var b actor
	for i := 0; i < 64; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		b = actor{cookie, csrf}
	}
	first := f.selfPost(a, "sessions/selections", "{}")
	second := f.selfPost(b, "sessions/selections", "{}")
	if first.status != 200 || second.status != 200 {
		t.Fatal("preparation failed")
	}
	if f.s.sessionRecordUnits != 130 {
		t.Fatalf("frozen records charge%d,want130", f.s.sessionRecordUnits)
	}
	before := f.s.counter
	blocked := f.selfPost(a, "sessions/end", selectionEndBody(first))
	if blocked.status != 429 || f.s.counter != before || len(f.s.selfEnds) != 0 || f.me(b).status != 200 || f.s.sessionRecordUnits != 130 {
		t.Fatal("capacity failure changed targets or reservation")
	}
	f.s.forgetSessionRecord(f.s.sessionRecords[second.body["selection"].(string)])
	accepted := f.selfPost(a, "sessions/end", selectionEndBody(first))
	if accepted.status != 202 || countOf(accepted.body["ended_count"]) != 64 || f.s.sessionRecordUnits != 129 {
		t.Fatalf("accepted reservation not reconciled:status%d units%d", accepted.status, f.s.sessionRecordUnits)
	}
}

func TestSessionSelectionForeignCallerCannotSubmitOrPreview(t *testing.T) {
	f, other, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal(prepared.status)
	}
	got := f.selfPost(other, "sessions/end", selectionEndBody(prepared))
	if got.status != 404 || f.me(a).status != 200 {
		t.Fatal("another caller used the frozen selection")
	}
}
