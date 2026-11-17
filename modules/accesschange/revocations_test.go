package accesschange

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeLedger struct {
	seq     uint64
	err     error
	repeat  bool
	entries []Entry
}

func (l *fakeLedger) Commit(e Entry) (Receipt, error) {
	l.entries = append(l.entries, e)
	if l.err != nil {
		return Receipt{}, l.err
	}
	if !l.repeat {
		l.seq++
	}
	return Receipt{Sequence: l.seq, Watermark: "mark-" + string(rune('a'+l.seq))}, nil
}

type reasoned struct{ reason string }

func (r reasoned) Error() string        { return r.reason }
func (r reasoned) CommitReason() string { return r.reason }

func target(id string) Revocation {
	return Revocation{Workspace: "w1", Target: Target{Kind: TargetGrant, ID: id, Version: 1}, RequestedBy: "alice", ApprovedBy: "bob", Request: "access-request-1", RequestDigest: "d"}
}

type effects struct {
	denied  int
	cleaned int
	ledger  *fakeLedger
	denyErr error
}

func (e *effects) build() Effects {
	return Effects{
		Deny: func() (Denied, error) {
			if e.denyErr != nil {
				return Denied{}, e.denyErr
			}
			e.denied++
			return Denied{Generation: 9, Ended: []string{"del-1"}}, nil
		},
		Ledger: e.ledger,
		Cleanup: func(d Denied) []Item {
			e.cleaned++
			return []Item{{Item: "C1", State: ItemDone, Reason: "deny"}, {Item: "C9", State: ItemDone, Refs: d.Ended}}
		},
		Planned: []Item{{Item: "C1", State: ItemDone}, {Item: "C5", State: ItemOutside}},
	}
}

func states(v RevocationView) string {
	out := []string{}
	for _, h := range v.History {
		out = append(out, h.State)
	}
	return strings.Join(out, ",")
}

func TestConfirmedRevocation(t *testing.T) {
	w := setup(t)
	e := &effects{ledger: &fakeLedger{seq: 4}}
	w.r.Restore(nil, Receipt{Sequence: 4, Watermark: "mark-e"}, 0, 0)
	v, err := w.r.Revoke(target("g1"), e.build())
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "revocation-1" || v.State != RevocationComplete || states(v) != "received,deny_confirmed,cleanup_running,revocation_complete" || v.Receipt.Sequence != 5 ||
		v.DenyGeneration != 9 || strings.Join(v.Ended, ",") != "del-1" || e.cleaned != 1 || v.Origin != OriginProcess || !v.TargetPresent || v.CompletedAt.IsZero() {
		t.Fatalf("view %+v", v)
	}
	entry := e.ledger.entries[0]
	if entry.Revocation != v.ID || entry.DenyGeneration != 9 || entry.Request != "access-request-1" || entry.Target.ID != "g1" || entry.ApprovedBy != "bob" {
		t.Errorf("entry %+v", entry)
	}
	if job, reason := JobState(v.State); job != JobSucceeded || reason != "" {
		t.Error("job state")
	}
	listed := w.r.Revocations("w1")
	listed[0].Ended[0], listed[0].Cleanup[1].Refs[0], listed[0].History[0].State = "x", "x", "x"
	if again := w.r.Revocations("w1")[0]; again.Ended[0] != "del-1" || again.Cleanup[1].Refs[0] != "del-1" || again.History[0].State != RevocationReceived {
		t.Error("views alias the registry")
	}
	if len(w.r.Revocations("w2")) != 0 {
		t.Error("other workspace")
	}
}

func TestUnconfirmedRevocations(t *testing.T) {
	for name, c := range map[string]struct {
		ledger *fakeLedger
		reason string
	}{
		"failed":        {&fakeLedger{err: errors.New("plain")}, CommitFailed},
		"unknown":       {&fakeLedger{err: reasoned{CommitUnknown}}, CommitUnknown},
		"isolated":      {&fakeLedger{err: reasoned{"ledger_isolated"}}, "ledger_isolated"},
		"non-advancing": {&fakeLedger{repeat: true}, CommitUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			w := setup(t)
			e := &effects{ledger: c.ledger}
			v, err := w.r.Revoke(target("g1"), e.build())
			if err != nil || v.State != RevocationUnconfirmed || v.Reason != c.reason || states(v) != "received,decision_unconfirmed" || e.denied != 1 || e.cleaned != 0 {
				t.Fatalf("%+v %v", v, err)
			}
			if len(v.Cleanup) != 2 || v.Cleanup[0].State != ItemNotStarted || v.Cleanup[1].Item != "C5" || v.Receipt != (Receipt{}) {
				t.Errorf("planned scope %+v", v.Cleanup)
			}
			if job, reason := JobState(v.State); job != JobWaiting || reason != RevocationUnconfirmed {
				t.Error("job state")
			}
		})
	}
	for _, s := range []string{RevocationReceived, RevocationConfirmed, RevocationCleaning} {
		if job, _ := JobState(s); job != JobRunning {
			t.Errorf("%s", s)
		}
	}
}

func TestDenyFailureCreatesNoRecord(t *testing.T) {
	w := setup(t)
	boom := errors.New("deny")
	e := &effects{ledger: &fakeLedger{}, denyErr: boom}
	if _, err := w.r.Revoke(target("g1"), e.build()); err != boom {
		t.Errorf("deny error %v", err)
	}
	if len(w.r.Revocations("w1")) != 0 || len(e.ledger.entries) != 0 {
		t.Error("a failed deny left a record or a ledger entry")
	}
	v := w.propose(proposal(revoke("g1"), alice))
	_, err := w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-1", Allowed: true}, func(a Applier) (string, error) {
		rv, err := a.Revoke(target("g1"), e.build())
		return rv.ID, err
	})
	if err != boom {
		t.Errorf("approve %v", err)
	}
	if got, _ := w.r.Request(v.Number, "w1"); got.State != StatePending {
		t.Error("the request did not stay pending")
	}
	e.denyErr = nil
	got, err := w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-1", Allowed: true}, func(a Applier) (string, error) {
		rv, err := a.Revoke(target("g1"), e.build())
		return rv.ID, err
	})
	if err != nil || got.Result != "revocation-1" || got.State != StateApplied {
		t.Errorf("approve after a fixed deny %+v %v", got, err)
	}
}

func TestRevocationInputs(t *testing.T) {
	w := setup(t)
	e := &effects{ledger: &fakeLedger{}}
	full := e.build()
	for name, c := range map[string]struct {
		in Revocation
		ef Effects
	}{
		"workspace": {Revocation{Target: Target{Kind: TargetGrant, ID: "g"}, RequestedBy: "a"}, full},
		"target":    {Revocation{Workspace: "w", Target: Target{Kind: TargetGrant}, RequestedBy: "a"}, full},
		"kind":      {Revocation{Workspace: "w", Target: Target{Kind: "x", ID: "g"}, RequestedBy: "a"}, full},
		"requester": {Revocation{Workspace: "w", Target: Target{Kind: TargetDelegation, ID: "g"}}, full},
		"deny":      {target("g"), Effects{Ledger: full.Ledger, Cleanup: full.Cleanup}},
		"ledger":    {target("g"), Effects{Deny: full.Deny, Cleanup: full.Cleanup}},
		"cleanup":   {target("g"), Effects{Deny: full.Deny, Ledger: full.Ledger}},
	} {
		if _, err := w.r.Revoke(c.in, c.ef); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if e.denied != 0 {
		t.Error("invalid input reached the deny")
	}
}

func TestUnconfirmedRecordsAreNeverPruned(t *testing.T) {
	w := setup(t)
	bad := &effects{ledger: &fakeLedger{err: errors.New("x")}}
	for i := 0; i < 5; i++ {
		if v, err := w.r.Revoke(target("u"), bad.build()); err != nil || v.State != RevocationUnconfirmed {
			t.Fatal(err)
		}
	}
	good := &effects{ledger: &fakeLedger{}}
	for i := 0; i < 4; i++ {
		if v, err := w.r.Revoke(target("c"), good.build()); err != nil || v.State != RevocationComplete {
			t.Fatalf("a revocation over the record limit %v", err)
		}
	}
	got := w.r.Revocations("w1")
	unconfirmed := 0
	for _, v := range got {
		if v.State == RevocationUnconfirmed {
			unconfirmed++
		}
	}
	if unconfirmed != 5 || len(got) != 5 {
		t.Errorf("kept %d unconfirmed %d", len(got), unconfirmed)
	}
}

func TestRestore(t *testing.T) {
	w := setup(t)
	w.propose(proposal(add("carol"), alice))
	at := start.Add(-time.Hour)
	records := []Restored{
		{ID: "revocation-7", RecordedAt: at, Revocation: target("g7"), Generation: 3, Ended: []string{"d1"}, Receipt: Receipt{Sequence: 2, Watermark: "m2"}, Replay: ReplayReapplied, Present: true,
			Cleanup: []Item{{Item: "C1", State: ItemDone}}},
		{ID: "revocation-9", RecordedAt: at.Add(time.Minute), Revocation: target("g9"), Replay: ReplayAbsent},
	}
	w.r.Restore(records, Receipt{Sequence: 3, Watermark: "m3"}, 12, 9)
	got := w.r.Revocations("w1")
	if len(got) != 2 || got[0].ID != "revocation-9" || got[1].Origin != OriginReplay || got[1].State != RevocationComplete || got[1].Replay != ReplayReapplied ||
		!got[1].TargetPresent || got[0].TargetPresent || !got[1].RecordedAt.Equal(at) || got[1].Receipt.Watermark != "m2" || states(got[1]) != RevocationComplete {
		t.Fatalf("restored %+v", got)
	}
	next := w.propose(proposal(add("dave"), alice))
	if next.ID != "access-request-13" {
		t.Errorf("request counter %s", next.ID)
	}
	e := &effects{ledger: &fakeLedger{seq: 3}}
	v, _ := w.r.Revoke(target("g1"), e.build())
	if v.ID != "revocation-10" || v.State != RevocationComplete {
		t.Errorf("revocation counter %+v", v)
	}
	w.r.Restore(nil, Receipt{Sequence: 4, Watermark: "mark-d"}, 1, 1)
	if v, _ := w.r.Revoke(target("g2"), e.build()); v.ID != "revocation-11" || v.State != RevocationComplete {
		t.Errorf("counters never go back %+v", v)
	}
	stale := &effects{ledger: &fakeLedger{seq: 1}}
	if v, _ := w.r.Revoke(target("g3"), stale.build()); v.State != RevocationUnconfirmed || v.Reason != CommitUnknown {
		t.Errorf("a receipt below the last one seen %+v", v)
	}
	if !reflect.DeepEqual(copyItems(nil), []Item{}) {
		t.Error("copy of no items")
	}
}
