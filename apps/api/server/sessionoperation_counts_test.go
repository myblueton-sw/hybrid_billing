package server

import (
	"testing"
	"time"
)

func assertOperationCounts(t *testing.T, record *sessionRecord) {
	t.Helper()
	ended, gone := 0, 0
	counts := selfCounts{}
	for _, row := range record.op.rows {
		if row.Result == resultGone {
			gone++
			continue
		}
		ended++
		switch row.Items[2].State {
		case itemDone:
			counts.Done++
		case itemCleared:
			counts.AlreadyEnded++
		case itemFailed:
			counts.Failed++
		default:
			counts.Pending++
		}
	}
	summary := operationSummary(record, false)
	if summary.Ended == nil || *summary.Ended != ended || *summary.Gone != gone || *summary.Cleanup != counts || *summary.NotIncluded != len(record.op.notIncluded) {
		t.Fatal("summary differs from independently counted result rows")
	}
}

func TestSessionOperationSummaryDoesNotTraverseStoredResultRows(t *testing.T) {
	f, _, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if got := f.selfPost(a, "sessions/end", selectionEndBody(prepared)); got.status != 202 {
		t.Fatal(got.status)
	}
	record := f.s.sessionRecords[prepared.body["selection"].(string)]
	assertOperationCounts(t, record)
	before := operationSummary(record, false)
	rows := record.op.rows
	record.op.rows = nil
	defer func() { record.op.rows = rows }()
	after := operationSummary(record, false)
	if *after.Ended != *before.Ended || *after.Cleanup != *before.Cleanup {
		t.Fatal("summary still depends on traversing stored result rows")
	}
}

func TestSessionOperationCountsFollowCleanupTransitions(t *testing.T) {
	for _, fault := range []string{"self-end-unavailable", "self-end-unexpected"} {
		t.Run(fault, func(t *testing.T) {
			f, _, _, a := queryActors(t)
			f.s.fault = func(key string) bool { return key == fault }
			prepared := f.selfPost(a, "sessions/selections", "{}")
			if got := f.selfPost(a, "sessions/end", selectionEndBody(prepared)); got.status != 202 {
				t.Fatal(got.status)
			}
			record := f.s.sessionRecords[prepared.body["selection"].(string)]
			assertOperationCounts(t, record)
			f.s.fault = nil
			f.advance(2 * time.Second)
			if fault == "self-end-unexpected" {
				for v := range f.s.retries {
					f.s.settleEnded(v, f.s.retryModuleEnd(v))
				}
			} else {
				f.s.drainSelfEnds()
			}
			assertOperationCounts(t, record)
		})
	}
}

func TestSessionOperationCountsIncludeGoneAndDenialCapacityFailure(t *testing.T) {
	f, _, peer, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if got := f.selfPost(a, "sessions/end", `{"session":"`+handleOfActor(peer)+`"}`); got.status != 202 {
		t.Fatal(got.status)
	}
	for len(f.s.sessions.denied) < maxSessions {
		f.s.sessions.denied[verifierOf(random(tokenBytes))] = denial{login: "filler@example.invalid", at: f.s.clock()}
	}
	f.s.fault = func(key string) bool { return key == "self-end-unavailable" }
	if got := f.selfPost(a, "sessions/end", selectionEndBody(prepared)); got.status != 202 {
		t.Fatal(got.status)
	}
	record := f.s.sessionRecords[prepared.body["selection"].(string)]
	assertOperationCounts(t, record)
	summary := operationSummary(record, false)
	if *summary.Gone != 1 || *summary.Ended != 1 || summary.Cleanup.Failed != 1 || summary.Cleanup.Pending != 0 {
		t.Fatal("gone or denial-capacity failure counts differ")
	}
}
