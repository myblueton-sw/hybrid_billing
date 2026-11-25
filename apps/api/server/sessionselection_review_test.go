package server

import (
	"testing"
	"time"
)

func TestSessionSelectionLegacyReplayRequiresOriginalRecordBinding(t *testing.T) {
	f, old, other, a := queryActors(t)
	body := `{"session":"` + handleOfActor(old) + `"}`
	first := f.selfPost(a, "sessions/end", body)
	if first.status != 202 {
		t.Fatal(first.status)
	}
	foreign := f.selfPost(other, "sessions/end", body)
	if foreign.status != 404 || foreign.body["operation"] != nil {
		t.Fatal("legacy replay disclosed another caller's operation")
	}
	own := f.selfPost(a, "sessions/end", body)
	if own.status != 202 || own.body["operation"] != first.body["operation"] || own.body["replay"] != true {
		t.Fatal("own replay stopped working")
	}
}

func TestSessionSelectionExpiredReservationsCannotBlockLegacyEnd(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 100
		c.Access = DefaultAccessSettings()
		c.Access.SessionSelectionTargets = 64
		c.Access.SessionRecordBudget = 256
	})
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	peers := []actor{}
	for i := 0; i < 63; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		peers = append(peers, actor{cookie, csrf})
	}
	for _, caller := range []actor{a, peers[1], peers[2], peers[3]} {
		if got := f.selfPost(caller, "sessions/selections", "{}"); got.status != 200 {
			t.Fatal(got.status)
		}
	}
	if f.s.sessionRecordUnits != 256 {
		t.Fatal("saturation fixture did not fill valid budget")
	}
	f.advance(301 * time.Second)
	f.s.sessions.reauthenticated(verifierOfActor(a), *f.now)
	ended := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(peers[0])+`"}`)
	if ended.status != 202 || f.s.sessionRecordUnits != 3 {
		t.Fatalf("expired reservations blocked admission:status%d units%d", ended.status, f.s.sessionRecordUnits)
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionDuplicateBuildRetryRejectsChangedSnapshot(t *testing.T) {
	f, a := selectionPopulation(t, 350)
	first := f.selfPost(a, "sessions/selections", "{}")
	if first.status != 200 {
		t.Fatal(first.status)
	}
	second := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	if second.status != 200 || second.body["state"] != "building" {
		t.Fatal("fixture completed too early")
	}
	if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
		t.Fatal(err)
	}
	duplicate := f.selfPost(a, "sessions/selections", selectionResumeBody(first))
	if duplicate.status != 409 || len(f.s.sessionRecords) != 0 || len(f.s.selfEnds) != 0 {
		t.Fatal("duplicate shortcut accepted a stale building snapshot")
	}
}

func TestSessionSelectionExpiryAtApplicationBoundaryHasNoEffect(t *testing.T) {
	f, old, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	f.s.selfEndHook = func() { f.advance(301 * time.Second) }
	got := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	f.s.selfEndHook = nil
	if got.status != 409 || f.me(old).status != 200 || len(f.s.selfEnds) != 0 {
		t.Fatal("selection expired before application but still changed targets")
	}
}

func TestSessionSelectionPreparationDoesNotBypassTerminationStepUp(t *testing.T) {
	f, _, _, a := queryActors(t)
	f.advance(stepUpDefault + time.Nanosecond)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if prepared.status != 200 || prepared.body["state"] != "sealed" {
		t.Fatal("read-only preparation unexpectedly required recent authentication")
	}
	before := f.s.counter
	got := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if got.status != 403 || got.body["code"] != CodeReauthentication || f.s.counter != before {
		t.Fatal("prepared snapshot bypassed actual termination step-up")
	}
	if f.reauth(a, pw).status != 204 {
		t.Fatal("reauthentication failed")
	}
	if got := f.selfPost(a, "sessions/end", selectionEndBody(prepared)); got.status != 202 {
		t.Fatal(got.status)
	}
}
