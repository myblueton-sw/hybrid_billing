package server

import (
	"reflect"
	"testing"
)

func TestSessionMalformedReservationChargesPreserveEvictableRecoveryAndAccounting(t *testing.T) {
	f, target, peer, caller := queryActors(t)
	ids := []string{}
	for _, a := range []actor{target, peer} {
		prepared := f.selfPost(caller, "sessions/selections", `{"session":"`+handleOfActor(a)+`"}`)
		if prepared.status != 200 {
			t.Fatal("actual single-target selection premise", prepared.status)
		}
		accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
		if accepted.status != 202 {
			t.Fatal("actual retained operation premise", accepted.status)
		}
		ids = append(ids, accepted.body["operation"].(string))
	}
	records, charges := map[string]*sessionRecord{}, map[string]int{}
	originals := map[string]reply{}
	for _, id := range ids {
		record := f.s.sessionOperations[id]
		if record == nil || record.op == nil || record.state != "accepted" || record.retries != 0 || !record.indexed {
			t.Fatal("genuine settled evictable record premise")
		}
		records[id], charges[id] = record, record.units
		originals[id] = f.operationRead(caller, "/"+id)
		if originals[id].status != 200 || originals[id].body["operation"] != id || countOf(originals[id].body["recorded_count"]) != 1 || countOf(originals[id].body["ended_count"]) != 1 {
			t.Fatal("nonempty original known-ID recovery premise")
		}
	}
	units, count := f.s.sessionRecordUnits, len(f.s.sessionRecords)
	if units <= 0 || count != 2 || records[ids[0]] == records[ids[1]] {
		t.Fatal("independent nonempty retained ledger premise")
	}
	maximum := int(^uint(0) >> 1)
	for _, invalid := range []int{-1, f.s.cfg.Access.SessionRecordBudget + 1, maximum, -maximum - 1} {
		func() {
			f.s.domain.Lock()
			defer f.s.domain.Unlock()
			if f.s.reserveSessionRecord(records[ids[0]], invalid) {
				t.Fatal("malformed reservation charge was accepted")
			}
			if f.s.sessionRecordUnits != units || f.s.sessionRecordUnits < 0 || len(f.s.sessionRecords) != count || len(f.s.sessionOperations) != 2 {
				t.Fatal("malformed reservation charge changed global accounting or recovery")
			}
			for _, id := range ids {
				record := records[id]
				if f.s.sessionRecords[record.ref] != record || f.s.sessionOperations[id] != record || record.units != charges[id] || record.state != "accepted" || record.retries != 0 {
					t.Fatal("malformed reservation charge evicted or altered a retained record")
				}
			}
			assertSessionRecordAccounting(t, f)
		}()
		for _, id := range ids {
			current := f.operationRead(caller, "/"+id)
			if current.status != 200 || current.body["operation"] != id || current.body["state"] != originals[id].body["state"] || current.body["digest"] != originals[id].body["digest"] || countOf(current.body["ended_count"]) != 1 || !reflect.DeepEqual(current.body["cleanup"], originals[id].body["cleanup"]) {
				t.Fatal("malformed reservation charge lost original operation recovery")
			}
		}
		if f.me(caller).status != 200 {
			t.Fatal("malformed reservation charge ended the surviving caller")
		}
	}
}
