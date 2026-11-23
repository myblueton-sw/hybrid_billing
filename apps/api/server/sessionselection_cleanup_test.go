package server

import (
	"testing"
	"time"
)

func TestSessionSelectionActiveCleanupSurvivesRetentionAndReleasesExactly(t *testing.T) {
	f, _, _, a := queryActors(t)
	f.s.fault = func(key string) bool { return key == "self-end-unavailable" }
	prepared := f.selfPost(a, "sessions/selections", "{}")
	ended := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if ended.status != 202 || ended.body["state"] != stateRunning {
		t.Fatal("cleanup uncertainty was hidden")
	}
	reference := prepared.body["selection"].(string)
	record := f.s.sessionRecords[reference]
	if record.retries != 2 || f.s.sessionRecordUnits != 7 {
		t.Fatal("initial retry reservation differs")
	}
	for i := 0; i < 6; i++ {
		cookie, csrf, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false)
		if err != nil {
			t.Fatal(err)
		}
		peer := actor{cookie, csrf}
		if got := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(peer)+`"}`); got.status != 202 {
			t.Fatal(got.status)
		}
	}
	if len(f.s.selfEnds[SampleMember]) != selfEndsKept || f.s.sessionRecords[reference] != record || record.retries != 2 {
		t.Fatal("active older cleanup was discarded")
	}
	if f.s.reserveSessionRecord(nil, f.s.cfg.Access.SessionRecordBudget) {
		t.Fatal("active records were evicted for admission")
	}
	assertSessionRecordAccounting(t, f)
	f.s.fault = nil
	f.advance(2 * time.Second)
	f.s.drainSelfEnds()
	if len(f.s.retries) != 0 || f.s.sessionRecords[reference] != nil || len(f.s.sessionRecords) != selfEndsKept || f.s.sessionRecordUnits != 15 {
		t.Fatalf("cleanup did not release exactly: records%d units%d", len(f.s.sessionRecords), f.s.sessionRecordUnits)
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionFinalFailureChargeReleasesOnDenialSweep(t *testing.T) {
	f, _, _, a := queryActors(t)
	f.s.fault = func(key string) bool { return key == "self-end-unexpected" }
	prepared := f.selfPost(a, "sessions/selections", "{}")
	ended := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if ended.status != 202 || ended.body["state"] != stateIncomplete || f.s.sessionRecordUnits != 7 {
		t.Fatal("final failed cleanup not retained")
	}
	f.advance(sessionAbsolute)
	f.s.drainSelfEnds()
	if len(f.s.retries) != 0 || f.s.sessionRecordUnits != 5 {
		t.Fatal("swept retry budget not released")
	}
	assertSessionRecordAccounting(t, f)
}
