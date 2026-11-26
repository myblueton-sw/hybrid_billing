package server

import "testing"

func TestSessionRecordPublicationRemovalAndForgetPreserveUnrelatedAccounting(t *testing.T) {
	f, target, peer, caller := queryActors(t)
	makeOperation := func(a actor) string {
		t.Helper()
		prepared := f.selfPost(caller, "sessions/selections", `{"session":"`+handleOfActor(a)+`"}`)
		if prepared.status != 200 {
			t.Fatal("actual frozen single-target producer premise", prepared.status)
		}
		accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
		if accepted.status != 202 {
			t.Fatal("actual accepted operation premise", accepted.status)
		}
		return accepted.body["operation"].(string)
	}
	first, second := makeOperation(target), makeOperation(peer)
	record := f.s.sessionOperations[first]
	if record == nil || record.state != "accepted" || record.retries != 0 || !record.indexed {
		t.Fatal("settled registered record premise")
	}
	owner := record.owner
	units, generation := f.s.sessionRecordUnits, f.s.sessionOperationVersions[owner]
	for i := 0; i < 2; i++ {
		f.s.publishSessionOperation(record)
	}
	page := f.operationRead(caller, "")
	if page.status != 200 || countOf(page.body["accepted_count"]) != 2 || len(page.body["rows"].([]any)) != 2 || f.s.sessionOperationVersions[owner] != generation {
		t.Fatal("republication duplicated a retained operation or changed generation")
	}
	impostor := *record
	f.s.forgetSessionRecord(&impostor)
	if f.s.sessionRecords[record.ref] != record || f.s.sessionRecordUnits != units || f.operationRead(caller, "/"+first).status != 200 {
		t.Fatal("same-reference local copy removed the registered record")
	}
	f.s.removeSessionOperationIndex(record, false)
	removedGeneration := f.s.sessionOperationVersions[owner]
	if removedGeneration == generation {
		t.Fatal("first index removal did not invalidate operation generation")
	}
	f.s.removeSessionOperationIndex(record, false)
	page = f.operationRead(caller, "")
	if page.status != 200 || countOf(page.body["accepted_count"]) != 1 || len(page.body["rows"].([]any)) != 1 || page.body["rows"].([]any)[0].(map[string]any)["operation"] != second || f.s.sessionOperationVersions[owner] != removedGeneration {
		t.Fatal("index removal was not idempotent or removed another operation")
	}
	if f.operationRead(caller, "/"+first).status != 200 || f.s.sessionRecordUnits != units {
		t.Fatal("index-only removal deleted known-ID recovery or released record units")
	}
	remaining := units - record.units
	f.s.forgetSessionRecord(record)
	f.s.forgetSessionRecord(record)
	if f.s.sessionRecordUnits != remaining || remaining == 0 || f.operationRead(caller, "/"+first).status != 404 || f.operationRead(caller, "/"+second).status != 200 || f.s.sessionOperationVersions[owner] != removedGeneration {
		t.Fatal("repeated forgetting changed unrelated recovery or accounting")
	}
	if f.me(caller).status != 200 {
		t.Fatal("record cleanup terminated the surviving caller")
	}
	assertSessionRecordAccounting(t, f)
}

func TestSessionOthersPublicationAndRemovalPreserveHeldSessionTag(t *testing.T) {
	f, _, _, caller := queryActors(t)
	prepared := f.selfPost(caller, "sessions/selections", "{}")
	if prepared.status != 200 {
		t.Fatal("actual others selection premise", prepared.status)
	}
	held := f.signedIn(MemberLogin, pw)
	accepted := f.selfPost(caller, "sessions/end", selectionEndBody(prepared))
	if accepted.status != 202 || countOf(accepted.body["not_included_count"]) != 1 {
		t.Fatal("actual held-session acceptance premise", accepted.status)
	}
	id := accepted.body["operation"].(string)
	record := f.s.sessionOperations[id]
	if record == nil || record.kind != kindOthers || record.retries != 0 {
		t.Fatal("settled others record premise")
	}
	captured := capturedAdmissionSession(f, held)
	generation, units := f.s.sessionOperationVersions[record.owner], f.s.sessionRecordUnits
	f.s.publishSessionOperation(record)
	if f.s.taggedSessionView(captured, caller.verifier(f)).NotIncludedOperation != id || record.olderOthers != nil || record.newerOthers != nil || f.s.sessionOperationVersions[record.owner] != generation {
		t.Fatal("republication changed the held tag or introduced a self-linked operation")
	}
	f.s.removeSessionOperationIndex(record, false)
	generation = f.s.sessionOperationSequence
	f.s.removeSessionOperationIndex(record, false)
	if f.s.taggedSessionView(captured, caller.verifier(f)).NotIncludedOperation != "" || f.s.sessionOperationSequence != generation || f.s.sessionRecordUnits != units || f.operationRead(caller, "/"+id).status != 200 {
		t.Fatal("others index cleanup retained a tag or removed known recovery")
	}
	f.s.forgetSessionRecord(record)
	f.s.forgetSessionRecord(record)
	if f.s.sessionRecordUnits != 0 || f.operationRead(caller, "/"+id).status != 404 || f.me(caller).status != 200 || f.me(held).status != 200 {
		t.Fatal("others record cleanup changed surviving sessions or exact accounting")
	}
	assertSessionRecordAccounting(t, f)
}
