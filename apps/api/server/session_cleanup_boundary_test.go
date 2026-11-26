package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestSessionSettledRetryIgnoresRepeatedCapturedRelease(t *testing.T) {
	f, target, _, caller := queryActors(t)
	verifier := target.verifier(f)
	f.s.fault = func(key string) bool { return key == "self-end-unavailable" }
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if prepared.status != 200 || accepted.status != 202 || accepted.body["state"] != stateRunning {
		t.Fatal("actual pending session cleanup premise")
	}
	retry := f.s.retries[verifier]
	if retry == nil || retry.op.record == nil || retry.op.record.retries != 2 {
		t.Fatal("actual two-target cleanup retry premise")
	}
	record, id := retry.op.record, accepted.body["operation"].(string)
	f.s.fault = nil
	f.advance(2 * time.Second)
	f.s.drainSelfEnds()
	before := f.operationRead(caller, "/"+id)
	units := f.s.sessionRecordUnits
	if len(f.s.retries) != 0 || record.retries != 0 || units == 0 || before.status != 200 || before.body["state"] != stateComplete {
		t.Fatal("actual completed retained cleanup premise")
	}
	for i := 0; i < 2; i++ {
		f.s.releaseSessionRetry(verifier, retry)
	}
	after := f.operationRead(caller, "/"+id)
	if len(f.s.retries) != 0 || record.retries != 0 || f.s.sessionRecordUnits != units || f.s.sessionRecords[record.ref] != record || after.status != 200 || after.body["state"] != stateComplete || !reflect.DeepEqual(before.body["cleanup"], after.body["cleanup"]) || f.me(caller).status != 200 {
		t.Fatal("stale cleanup callback changed settled accounting or recovery")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionSelectionAdmissionClockExpiryReclaimsNewReservationWithoutEndingTargets(t *testing.T) {
	f, first, second, caller := queryActors(t)
	actors := []actor{first, second, caller}
	verifiers := make([]identity.Verifier, 0, len(actors))
	for _, a := range actors {
		verifiers = append(verifiers, a.verifier(f))
	}
	found := capturedAdmissionSession(f, caller)
	answer, code := f.s.access(found)
	if code != "" || !f.s.clock().Before(answer.NotAfter()) {
		t.Fatal("actual pre-expiry admission premise")
	}
	units, operations, counter := f.s.sessionRecordUnits, len(f.s.sessionOperations), f.s.counter
	calls := 0
	f.s.accessHook = func(v identity.Verifier) {
		f.s.accessHook = nil
		if v != found.verifier {
			t.Fatal("expiry hook observed another actual caller")
		}
		calls++
		f.advance(sessionAbsolute + time.Second)
	}
	got := f.selfPost(caller, "sessions/selections", "{}")
	if calls != 1 || f.s.clock().Before(answer.NotAfter()) || got.status != 409 || got.body["code"] != string(commands.RevisionConflict) {
		t.Fatal("post-admission expiry did not refuse the new selection", got.status)
	}
	for _, v := range verifiers {
		if !f.s.sessions.present(v) {
			t.Fatal("expired selection preparation terminated a stored target or caller")
		}
	}
	if f.s.sessionRecordUnits != units || len(f.s.sessionRecords) != 0 || len(f.s.sessionOperations) != operations || f.s.counter != counter {
		t.Fatal("post-admission expiry retained a reservation or produced operation effects")
	}
	assertSessionRecordAccounting(t, f)
	if f.me(caller).status != 401 {
		t.Fatal("actual globally expired caller regained authentication")
	}
}
