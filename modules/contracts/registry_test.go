package contracts

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

var testLimits = Limits{Contracts: 3, Writes: 60, Reserve: 2, Batch: 6}

type book struct {
	t      testing.TB
	r      *Registry
	space  string
	id     string
	writes *int
}

func open(t testing.TB, limits Limits) book {
	t.Helper()
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	registry, err := NewRegistry(limits, func() time.Time {
		now = now.Add(time.Minute)
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	return book{t: t, r: registry, space: "w1", id: "c1", writes: &count}
}

func (b book) sequence() uint64 {
	current, err := b.r.Sequence(b.space, b.id)
	if err != nil {
		return 0
	}
	return current
}

func (b book) next() string {
	*b.writes++
	return fmt.Sprintf("write-%d", *b.writes)
}

func (b book) check() {
	b.t.Helper()
	current := b.sequence()
	if current == 0 {
		return
	}
	events, _ := b.r.Lifecycle(b.space, b.id, current)
	view, _ := b.r.Settlement(b.space, b.id, current)
	far, _ := NewDay(9999, 12, 31)
	position, _ := b.r.StateOn(b.space, b.id, current, far)
	if final := position.State == Ended || len(events) == 0; final != (view.State != NoSettlement) {
		b.t.Errorf("at %d the settlement is %s for a lifecycle that is final=%v", current, view.State, final)
	}
	if !validLifecycle(events) {
		b.t.Errorf("at %d the lifecycle that holds is not valid", current)
	}
}

func (b book) want(name string, err, target error) {
	b.t.Helper()
	if !errors.Is(err, target) && !(err == nil && target == nil) {
		b.t.Errorf("%s: %v; want %v", name, err, target)
	}
	b.check()
}

func (b book) amend(cancel CancelReason, withdrawals []string, events ...string) error {
	b.t.Helper()
	_, err := b.r.Amend(b.space, b.id, b.sequence(), b.next(), withdrawals, evs(b.t, events...), cancel)
	return err
}

func (b book) secured(until string, resolves ...string) error {
	_, err := b.r.RecordEvidence(b.space, b.id, b.sequence(), b.next(), Evidence{Kind: Secured, Reference: "evidence-1", CoveredUntil: day(b.t, until), Resolves: resolves})
	return err
}

func (b book) missing(write string, reason GapReason) error {
	_, err := b.r.RecordEvidence(b.space, b.id, b.sequence(), write, Evidence{Kind: Missing, Reason: reason, OwnerID: "owner-7", FollowUp: "task-9"})
	return err
}

func (b book) settle(asOf string) error {
	_, err := b.r.Settle(b.space, b.id, b.sequence(), b.next(), day(b.t, asOf))
	return err
}

func (b book) reopen(reason ReopenReason) error {
	_, err := b.r.Reopen(b.space, b.id, b.sequence(), b.next(), reason)
	return err
}

func (b book) state() SettlementState {
	view, err := b.r.Settlement(b.space, b.id, b.sequence())
	if err != nil {
		b.t.Fatal(err)
	}
	return view.State
}

func (b book) life(at uint64) string {
	events, err := b.r.Lifecycle(b.space, b.id, at)
	if err != nil {
		return err.Error()
	}
	text := ""
	for _, event := range events {
		text += fmt.Sprintf("%s %s %s %d; ", event.ID, event.Kind, event.From, event.Revision)
	}
	return text
}

func TestBatches(t *testing.T) {
	b := open(t, testLimits)
	if _, err := b.r.Lifecycle("w1", "c1", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("a contract without writes: %v", err)
	}
	b.want("the first batch", b.amend("", nil, "a start 2026-06-01 1"), nil)
	b.want("a suspension", b.amend("", nil, "b suspend 2026-06-11"), nil)
	b.want("a backdated change", b.amend("", nil, "c change 2026-06-05 2"), nil)
	b.want("a withdrawal with its replacement", b.amend("", []string{"b"}, "d suspend 2026-06-12", "e resume 2026-06-20"), nil)
	history := map[uint64]string{
		0: "",
		1: "a start 2026-06-01 1; ",
		2: "a start 2026-06-01 1; b suspend 2026-06-11 0; ",
		3: "a start 2026-06-01 1; c change 2026-06-05 2; b suspend 2026-06-11 0; ",
		4: "a start 2026-06-01 1; c change 2026-06-05 2; d suspend 2026-06-12 0; e resume 2026-06-20 0; ",
		5: ErrNotFound.Error(),
	}
	for at, want := range history {
		if got := b.life(at); got != want {
			t.Errorf("the lifecycle at %d: %q; want %q", at, got, want)
		}
	}
	for name, c := range map[string]struct {
		at   uint64
		on   string
		want Position
	}{
		"before anything was known":            {0, "2026-06-15", Position{NotStarted, 0}},
		"the eleventh as first known":          {2, "2026-06-11", Position{Suspended, 1}},
		"the eleventh after the correction":    {4, "2026-06-11", Position{Active, 2}},
		"the fifth before the backdated event": {2, "2026-06-05", Position{Active, 1}},
		"the fifth after it":                   {3, "2026-06-05", Position{Active, 2}},
		"the twelfth now":                      {4, "2026-06-12", Position{Suspended, 2}},
	} {
		got, err := b.r.StateOn("w1", "c1", c.at, day(t, c.on))
		if err != nil || got != c.want {
			t.Errorf("%s: %+v %v", name, got, err)
		}
	}
	if _, err := b.r.StateOn("w1", "c1", 4, Day{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a state without a day: %v", err)
	}
	if _, err := b.r.StateOn("w1", "c1", 5, day(t, "2026-06-01")); !errors.Is(err, ErrNotFound) {
		t.Errorf("a state at a knowledge point that does not exist yet: %v", err)
	}
	segments, err := b.r.SegmentsAt("w1", "c1", 2, span(t, "2026-06-01", "2026-07-01"))
	if err != nil || shape(segments) != "[2026-06-01,2026-06-11) active 1; [2026-06-11,2026-07-01) suspended 1" {
		t.Errorf("segments as first known: %s %v", shape(segments), err)
	}
	segments, err = b.r.SegmentsAt("w1", "c1", 4, span(t, "2026-06-01", "2026-07-01"))
	if err != nil || shape(segments) != "[2026-06-01,2026-06-05) active 1; [2026-06-05,2026-06-12) active 2; [2026-06-12,2026-06-20) suspended 2; [2026-06-20,2026-07-01) active 2" {
		t.Errorf("segments now: %s %v", shape(segments), err)
	}
	if _, err := b.r.SegmentsAt("w1", "c1", 9, span(t, "2026-06-01", "2026-07-01")); !errors.Is(err, ErrNotFound) {
		t.Errorf("segments at an unknown point: %v", err)
	}
	before, _ := b.r.Records("w1", "c1")
	refused := map[string]struct {
		err         error
		expected    uint64
		write       string
		withdrawals []string
		events      []string
		cancel      CancelReason
	}{
		"an empty batch":                           {ErrInvalid, 4, "x", nil, nil, ""},
		"an event that is malformed":               {ErrInvalid, 4, "x", nil, []string{"f change 2026-06-25"}, ""},
		"an unknown cancellation reason":           {ErrInvalid, 4, "x", nil, []string{"f renew 2026-06-25"}, "because"},
		"a write identifier that is text":          {ErrInvalid, 4, "x y", nil, []string{"f renew 2026-06-25"}, ""},
		"a withdrawal that is text":                {ErrInvalid, 4, "x", []string{"b c"}, nil, ""},
		"a stale expected sequence":                {ErrConflict, 3, "x", nil, []string{"f renew 2026-06-25"}, ""},
		"an expected sequence ahead":               {ErrConflict, 5, "x", nil, []string{"f renew 2026-06-25"}, ""},
		"an event identifier twice in the batch":   {ErrConflict, 4, "x", nil, []string{"f renew 2026-06-25", "f renew 2026-06-26"}, ""},
		"an event identifier that exists":          {ErrConflict, 4, "x", nil, []string{"a renew 2026-06-25"}, ""},
		"the identifier of a withdrawn event":      {ErrConflict, 4, "x", nil, []string{"b renew 2026-06-25"}, ""},
		"a withdrawal of an unknown event":         {ErrRejected, 4, "x", []string{"z"}, nil, ""},
		"a withdrawal of a withdrawn event":        {ErrRejected, 4, "x", []string{"b"}, nil, ""},
		"the same withdrawal twice":                {ErrRejected, 4, "x", []string{"e", "e"}, nil, ""},
		"a withdrawal of an event of the batch":    {ErrRejected, 4, "x", []string{"f"}, []string{"f renew 2026-06-25"}, ""},
		"a batch that breaks the lifecycle":        {ErrRejected, 4, "x", nil, []string{"f resume 2026-06-25"}, ""},
		"a withdrawal that breaks the lifecycle":   {ErrRejected, 4, "x", []string{"d"}, nil, ""},
		"a second event on a day":                  {ErrRejected, 4, "x", nil, []string{"f renew 2026-06-20"}, ""},
		"a cancellation where none is due":         {ErrRejected, 4, "x", nil, []string{"f renew 2026-06-25"}, RecordedInError},
		"a cancellation on a batch that ends":      {ErrRejected, 4, "x", nil, []string{"f end 2026-06-25"}, ContractContinues},
		"more records than a batch may hold":       {ErrLimit, 4, "x", []string{"e"}, []string{"f resume 2026-06-21", "g renew 2026-06-22", "h renew 2026-06-23", "i renew 2026-06-24", "j renew 2026-06-25", "k renew 2026-06-26"}, ""},
		"a valid event beside one that is refused": {ErrInvalid, 4, "x", nil, []string{"f renew 2026-06-25", "g start 2026-06-26"}, ""},
	}
	for name, c := range refused {
		_, err := b.r.Amend("w1", "c1", c.expected, c.write, c.withdrawals, evs(t, c.events...), c.cancel)
		b.want(name, err, c.err)
	}
	for name, ids := range map[string][2]string{"a workspace that is text": {"w 1", "c1"}, "no workspace": {"", "c1"}, "a contract that is text": {"w1", "c/1"}} {
		if _, err := b.r.Amend(ids[0], ids[1], 0, "x", nil, evs(t, "a start 2026-06-01 1"), ""); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
		if _, err := b.r.Lifecycle(ids[0], ids[1], 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s, read: %v", name, err)
		}
	}
	after, _ := b.r.Records("w1", "c1")
	if !reflect.DeepEqual(before, after) || b.sequence() != 4 || b.life(4) != history[4] {
		t.Errorf("a refused batch wrote something")
	}
	if _, err := b.r.Amend("w1", "c1", 4, "x", nil, evs(t, "f renew 2026-06-25"), ""); err != nil {
		t.Errorf("a refused write consumed its identifier: %v", err)
	}
	records, _ := b.r.Records("w1", "c1")
	for index, record := range records {
		if record.Sequence != uint64(index)+1 || record.Kind != WriteBatch || record.State != NoSettlement || (index > 0 && !records[index-1].At.Before(record.At)) {
			t.Errorf("record %d: %+v", index, record)
		}
	}
	if len(records) != 5 || !reflect.DeepEqual(records[3].Withdrawals, []string{"b"}) || len(records[3].Events) != 2 || records[3].Events[0].ID != "d" {
		t.Errorf("the batch record: %+v", records[3])
	}
	records[3].Events[0].ID, records[3].Withdrawals[0] = "tampered", "tampered"
	again, _ := b.r.Records("w1", "c1")
	if again[3].Events[0].ID != "d" || again[3].Withdrawals[0] != "b" {
		t.Errorf("the log aliases what it returns")
	}
	given := evs(t, "g renew 2026-06-27")
	if _, err := b.r.Amend("w1", "c1", 5, "y", nil, given, ""); err != nil {
		t.Fatal(err)
	}
	given[0].ID = "tampered"
	if stored, _ := b.r.Records("w1", "c1"); stored[5].Events[0].ID != "g" || b.life(6) != history[4]+"f renew 2026-06-25 0; g renew 2026-06-27 0; " {
		t.Errorf("the registry aliases the caller's events")
	}

	other := b
	other.space = "w2"
	if _, err := other.r.Lifecycle("w2", "c1", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("the same contract identifier in another workspace is found: %v", err)
	}
	other.want("the same identifier in another workspace", other.amend("", nil, "a start 2027-01-01 7"), nil)
	if other.sequence() != 1 || other.life(1) != "a start 2027-01-01 7; " || b.sequence() != 6 {
		t.Errorf("two workspaces share a contract")
	}
	if _, err := b.r.Records("w3", "c1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("records of an unknown workspace: %v", err)
	}
	if _, err := b.r.Settlement("w3", "c1", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("settlement of an unknown workspace: %v", err)
	}
}

func TestRepeatedWrites(t *testing.T) {
	b := open(t, testLimits)
	start := evs(t, "a start 2026-06-01 1")
	first, err := b.r.Amend("w1", "c1", 0, "w-start", nil, start, "")
	if err != nil || first != (Outcome{Sequence: 1}) {
		t.Fatalf("%+v %v", first, err)
	}
	again, err := b.r.Amend("w1", "c1", 0, "w-start", nil, start, "")
	if err != nil || again != (Outcome{Sequence: 1, Replay: true}) || b.sequence() != 1 {
		t.Errorf("the same write again: %+v %v", again, err)
	}
	end := evs(t, "b end 2026-06-11")
	if _, err := b.r.Amend("w1", "c1", 1, "w-end", nil, end, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := b.r.Amend("w1", "c1", 2, "w-continue", []string{"b"}, nil, ContractContinues); err != nil {
		t.Fatal(err)
	}
	replay, err := b.r.Amend("w1", "c1", 1, "w-end", nil, end, "")
	if err != nil || replay != (Outcome{Sequence: 2, Replay: true}) || b.sequence() != 3 || b.state() != NoSettlement {
		t.Errorf("a replay after the state moved on: %+v %v", replay, err)
	}
	if replay, err := b.r.Amend("w1", "c1", 99, "w-end", nil, end, ""); err != nil || !replay.Replay {
		t.Errorf("a replay is evaluated before the expected sequence: %+v %v", replay, err)
	}
	for name, change := range map[string]func() (Outcome, error){
		"another event":       func() (Outcome, error) { return b.r.Amend("w1", "c1", 3, "w-end", nil, evs(t, "b end 2026-06-12"), "") },
		"an added withdrawal": func() (Outcome, error) { return b.r.Amend("w1", "c1", 3, "w-end", []string{"a"}, end, "") },
		"an added reason":     func() (Outcome, error) { return b.r.Amend("w1", "c1", 3, "w-end", nil, end, RecordedInError) },
		"another reason": func() (Outcome, error) {
			return b.r.Amend("w1", "c1", 3, "w-continue", []string{"b"}, nil, RecordedInError)
		},
		"another kind, settle": func() (Outcome, error) { return b.r.Settle("w1", "c1", 3, "w-end", day(t, "2026-07-01")) },
		"another kind, reopen": func() (Outcome, error) { return b.r.Reopen("w1", "c1", 3, "w-end", LateSource) },
		"another kind, evidence": func() (Outcome, error) {
			return b.r.RecordEvidence("w1", "c1", 3, "w-end", Evidence{Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f"})
		},
	} {
		if _, err := change(); !errors.Is(err, ErrConflict) {
			t.Errorf("%s under a stored identifier: %v", name, err)
		}
	}
	two := evs(t, "c renew 2026-06-20", "d renew 2026-06-21")
	if _, err := b.r.Amend("w1", "c1", 3, "w-two", nil, two, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := b.r.Amend("w1", "c1", 3, "w-two", nil, []Event{two[1], two[0]}, ""); !errors.Is(err, ErrConflict) {
		t.Errorf("the same events in another order are other content: %v", err)
	}
	if replay, err := b.r.Amend("w1", "c1", 3, "w-two", []string{}, two, ""); err != nil || !replay.Replay {
		t.Errorf("an empty and an absent list of withdrawals differ: %v", err)
	}
	if b.sequence() != 4 {
		t.Errorf("a replay or a conflict wrote: sequence %d", b.sequence())
	}
	if _, err := b.r.Amend("w1", "c1", 4, "w-gap", nil, evs(t, "e end 2026-07-01"), ""); err != nil {
		t.Fatal(err)
	}
	gap := Evidence{Kind: Missing, Reason: SourceUnavailable, OwnerID: "owner-7", FollowUp: "task-9"}
	if _, err := b.r.RecordEvidence("w1", "c1", 5, "w-missing", gap); err != nil {
		t.Fatal(err)
	}
	if replay, err := b.r.RecordEvidence("w1", "c1", 5, "w-missing", gap); err != nil || replay != (Outcome{Sequence: 6, Replay: true}) {
		t.Errorf("an evidence record again: %+v %v", replay, err)
	}
	gap.OwnerID = "owner-8"
	if _, err := b.r.RecordEvidence("w1", "c1", 6, "w-missing", gap); !errors.Is(err, ErrConflict) {
		t.Errorf("an evidence record with another owner: %v", err)
	}
	proof := Evidence{Kind: Secured, Reference: "evidence-1", CoveredUntil: day(t, "2026-07-01"), Resolves: []string{"w-missing"}}
	if _, err := b.r.RecordEvidence("w1", "c1", 6, "w-proof", proof); err != nil {
		t.Fatal(err)
	}
	if replay, err := b.r.RecordEvidence("w1", "c1", 6, "w-proof", proof); err != nil || !replay.Replay || replay.Sequence != 7 {
		t.Errorf("a secured record again, although its gap is now resolved: %+v %v", replay, err)
	}
	if _, err := b.r.Settle("w1", "c1", 7, "w-settle", day(t, "2026-07-02")); err != nil {
		t.Fatal(err)
	}
	if replay, err := b.r.Settle("w1", "c1", 7, "w-settle", day(t, "2026-07-02")); err != nil || !replay.Replay || replay.Sequence != 8 {
		t.Errorf("a settlement again: %+v %v", replay, err)
	}
	if _, err := b.r.Settle("w1", "c1", 8, "w-settle", day(t, "2026-07-03")); !errors.Is(err, ErrConflict) {
		t.Errorf("a settlement as of another day: %v", err)
	}
	if _, err := b.r.Reopen("w1", "c1", 8, "w-reopen", LateCredit); err != nil {
		t.Fatal(err)
	}
	if replay, err := b.r.Reopen("w1", "c1", 8, "w-reopen", LateCredit); err != nil || !replay.Replay || replay.Sequence != 9 {
		t.Errorf("a reopening again: %+v %v", replay, err)
	}
	if _, err := b.r.Reopen("w1", "c1", 9, "w-reopen", LateSource); !errors.Is(err, ErrConflict) {
		t.Errorf("a reopening for another reason: %v", err)
	}
	if b.sequence() != 9 {
		t.Errorf("sequence %d", b.sequence())
	}
	b.check()
}

func TestSettlementIsStoredApartAndGuarded(t *testing.T) {
	b := open(t, testLimits)
	if _, err := b.r.RecordEvidence("w1", "c1", 0, "g0", Evidence{Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("evidence on an unknown contract: %v", err)
	}
	if _, err := b.r.Settle("w1", "c1", 0, "s0", day(t, "2026-07-01")); !errors.Is(err, ErrNotFound) {
		t.Errorf("settling an unknown contract: %v", err)
	}
	if _, err := b.r.Reopen("w1", "c1", 0, "r0", LateSource); !errors.Is(err, ErrNotFound) {
		t.Errorf("reopening an unknown contract: %v", err)
	}
	if _, err := b.r.Lifecycle("w1", "c1", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("a refused write created the contract: %v", err)
	}
	b.want("a start", b.amend("", nil, "a start 2026-06-01 1"), nil)
	if b.state() != NoSettlement {
		t.Errorf("a running contract has the settlement %s", b.state())
	}
	b.want("secured evidence on a running contract", b.secured("2026-07-01"), ErrState)
	b.want("settling a running contract", b.settle("2026-07-01"), ErrState)
	b.want("reopening a running contract", b.reopen(LateSource), ErrState)
	b.want("an emergency withdrawal on a running contract is recorded", b.missing("gap-emergency", AccessWithdrawnEmergency), nil)
	if b.state() != NoSettlement {
		t.Errorf("a gap record changed the settlement to %s", b.state())
	}
	b.want("an end", b.amend("", nil, "b end 2026-07-01"), nil)
	if b.state() != PendingFinal {
		t.Errorf("an end left the settlement %s", b.state())
	}
	records, _ := b.r.Records("w1", "c1")
	if last := records[len(records)-1]; last.Kind != WriteBatch || last.State != PendingFinal {
		t.Errorf("the end and the pending final settlement are not one write: %+v", last)
	}
	b.want("settling without evidence", b.settle("2026-07-02"), ErrRejected)
	b.want("the last collection fails", b.missing("gap-last", SourceUnavailable), nil)
	b.want("settling with only gaps", b.settle("2026-07-02"), ErrRejected)
	b.want("evidence that resolves an unknown gap", b.secured("2026-07-01", "gap-none"), ErrRejected)
	b.want("evidence that names a write that is not a gap", b.secured("2026-07-01", "write-1"), ErrRejected)
	b.want("evidence that resolves one gap", b.secured("2026-07-01", "gap-last"), nil)
	proof := fmt.Sprintf("write-%d", *b.writes)
	b.want("settling while the earlier gap is open", b.settle("2026-07-02"), ErrRejected)
	b.want("evidence that resolves a resolved gap", b.secured("2026-07-01", "gap-last"), ErrRejected)
	b.want("evidence that names an open gap and an unknown one", b.secured("2026-07-01", "gap-emergency", "gap-none"), ErrRejected)
	if view, _ := b.r.Settlement("w1", "c1", b.sequence()); view.Evidence[0].Write != "gap-emergency" || view.Evidence[0].Resolved {
		t.Errorf("a refused secured record resolved a gap: %+v", view.Evidence[0])
	}
	b.want("evidence that names a secured record as a gap", b.secured("2026-07-01", proof), ErrRejected)
	b.want("evidence secured before the last served day, resolving the earlier gap", b.secured("2026-06-30", "gap-emergency"), nil)
	b.want("settling on evidence that stops short", b.settle("2026-07-02"), ErrRejected)
	b.want("evidence through the end", b.secured("2026-07-01"), nil)
	b.want("settling before the end day", b.settle("2026-06-30"), ErrRejected)
	b.want("settling on the end day", b.settle("2026-07-01"), nil)
	if b.state() != Settled {
		t.Errorf("state %s", b.state())
	}
	settledAt := b.sequence()
	b.want("a batch while settled", b.amend("", []string{"b"}, "c end 2026-07-05"), ErrState)
	b.want("an addition while settled", b.amend(RecordedInError, []string{"b"}), ErrState)
	b.want("secured evidence while settled", b.secured("2026-07-01"), ErrState)
	b.want("settling twice", b.settle("2026-07-02"), ErrState)
	b.want("a source received after the end is recorded while settled", b.missing("gap-late", LateSourceReceived), nil)
	if b.state() != Settled {
		t.Errorf("a gap record alone reopened the settlement")
	}
	if _, err := b.r.Reopen("w1", "c1", b.sequence(), "r-bad", "because"); !errors.Is(err, ErrInvalid) {
		t.Errorf("an unknown reopening reason: %v", err)
	}
	b.want("a reopening", b.reopen(LateSource), nil)
	b.want("reopening twice", b.reopen(LateSource), ErrState)
	b.want("settling on the evidence from before the reopening", b.settle("2026-07-02"), ErrRejected)
	b.want("new evidence that does not name the late source", b.secured("2026-07-01"), nil)
	b.want("settling while the late source is open", b.settle("2026-07-02"), ErrRejected)
	b.want("new evidence for the late source", b.secured("2026-07-01", "gap-late"), nil)
	b.want("the end moved earlier while pending", b.amend("", []string{"b"}, "m end 2026-06-28"), nil)
	b.want("settling on evidence that covers the new end but is older than the batch", b.settle("2026-07-10"), ErrRejected)
	b.want("the end moved while pending", b.amend("", []string{"m"}, "c end 2026-07-06"), nil)
	if b.state() != PendingFinal {
		t.Errorf("moving the end changed the settlement to %s", b.state())
	}
	b.want("settling on evidence older than the last batch", b.settle("2026-07-10"), ErrRejected)
	b.want("evidence that covers the old end only", b.secured("2026-07-01"), nil)
	b.want("settling on it", b.settle("2026-07-10"), ErrRejected)
	b.want("evidence through the new end", b.secured("2026-07-06"), nil)
	b.want("settling again", b.settle("2026-07-06"), nil)

	view, _ := b.r.Settlement("w1", "c1", b.sequence())
	if view.State != Settled || view.Reopenings != 1 || len(view.Evidence) != 10 {
		t.Errorf("the settlement view: %+v", view)
	}
	for _, kept := range view.Evidence {
		if kept.Evidence.Kind == Missing && (!kept.Resolved || kept.Evidence.OwnerID != "owner-7" || kept.Evidence.FollowUp != "task-9") {
			t.Errorf("a gap lost its owner, its follow-up or its resolution: %+v", kept)
		}
		if kept.Evidence.Kind == Secured && kept.Resolved {
			t.Errorf("a secured record is marked resolved")
		}
	}
	earlier, _ := b.r.Settlement("w1", "c1", settledAt)
	if earlier.State != Settled || earlier.Reopenings != 0 || len(earlier.Evidence) != 5 {
		t.Errorf("the settlement as first settled: %+v", earlier)
	}
	atGap, _ := b.r.Settlement("w1", "c1", 4)
	if atGap.State != PendingFinal || len(atGap.Evidence) != 2 || atGap.Evidence[0].Resolved || atGap.Evidence[1].Resolved || atGap.Evidence[1].Write != "gap-last" ||
		atGap.Evidence[1].Evidence.Reason != SourceUnavailable || atGap.Evidence[1].Sequence != 4 {
		t.Errorf("the settlement when the last collection had failed: %+v", atGap)
	}
	running, _ := b.r.Settlement("w1", "c1", 2)
	if running.State != NoSettlement || len(running.Evidence) != 1 || running.Evidence[0].Evidence.Reason != AccessWithdrawnEmergency {
		t.Errorf("the settlement of the running contract with its gap: %+v", running)
	}
	none, _ := b.r.Settlement("w1", "c1", 0)
	if none.State != NoSettlement || len(none.Evidence) != 0 {
		t.Errorf("the settlement before anything: %+v", none)
	}
	if _, err := b.r.Settlement("w1", "c1", b.sequence()+1); !errors.Is(err, ErrNotFound) {
		t.Errorf("a settlement at an unknown point: %v", err)
	}
	view.Evidence[1].Evidence.Resolves = append(view.Evidence[1].Evidence.Resolves, "tampered")
	if fresh, _ := b.r.Settlement("w1", "c1", b.sequence()); len(fresh.Evidence[2].Evidence.Resolves) != 1 || fresh.Evidence[2].Evidence.Resolves[0] != "gap-last" {
		t.Errorf("the evidence view: %+v", fresh.Evidence[2])
	}
}

func TestAMistakenEndAndAVoidedContract(t *testing.T) {
	b := open(t, testLimits)
	b.want("a start", b.amend("", nil, "a start 2026-06-01 1"), nil)
	b.want("an end recorded by mistake", b.amend("", nil, "b end 2026-06-11"), nil)
	b.want("evidence for it", b.secured("2026-06-11"), nil)
	b.want("removing the end without a reason", b.amend("", []string{"b"}), ErrRejected)
	b.want("a cancellation although the contract stays ended", b.amend(RecordedInError, []string{"b"}, "c end 2026-06-12"), ErrRejected)
	b.want("removing the end with a reason", b.amend(RecordedInError, []string{"b"}), nil)
	if b.state() != NoSettlement {
		t.Errorf("the cancelled final settlement is %s", b.state())
	}
	records, _ := b.r.Records("w1", "c1")
	if last := records[len(records)-1]; last.Cancel != RecordedInError || last.State != NoSettlement {
		t.Errorf("the cancellation is not recorded: %+v", last)
	}
	b.want("a cancellation where the state is none", b.amend(ContractContinues, nil, "d renew 2026-06-20"), ErrRejected)
	b.want("a new end", b.amend("", nil, "e end 2026-07-01"), nil)
	b.want("settling the new end on the evidence of the cancelled one", b.settle("2026-07-02"), ErrRejected)
	b.want("new evidence", b.secured("2026-07-01"), nil)
	b.want("settling", b.settle("2026-07-02"), nil)
	b.want("a reopening for a correction", b.reopen(Correcting), nil)
	b.want("settling at once on the evidence from before the reopening, with no gap open", b.settle("2026-07-02"), ErrRejected)
	b.want("evidence after the reopening", b.secured("2026-07-01"), nil)
	b.want("settling on it", b.settle("2026-07-02"), nil)
	b.want("a second reopening", b.reopen(Correcting), nil)
	b.want("the end withdrawn after the reopening", b.amend(ContractContinues, []string{"e"}), nil)
	if view, _ := b.r.Settlement("w1", "c1", b.sequence()); view.State != NoSettlement || len(view.Evidence) != 3 || view.Reopenings != 2 {
		t.Errorf("after the cancellation: %+v", view)
	}

	v := open(t, testLimits)
	v.want("a start", v.amend("", nil, "a start 2026-06-01 1"), nil)
	v.want("a gap on the running contract", v.missing("gap-1", ProviderRefused), nil)
	v.want("the start withdrawn", v.amend("", []string{"a"}), nil)
	if v.state() != PendingFinal || v.life(v.sequence()) != "" {
		t.Errorf("a voided contract: %s %q", v.state(), v.life(v.sequence()))
	}
	if position, _ := v.r.StateOn("w1", "c1", v.sequence(), day(t, "2026-06-15")); position != (Position{NotStarted, 0}) {
		t.Errorf("the state of a voided contract: %+v", position)
	}
	v.want("settling a voided contract with an open gap", v.settle("2026-06-02"), ErrRejected)
	v.want("a new start without a cancellation", v.amend("", nil, "s start 2026-08-01 1"), ErrRejected)
	v.want("the gap resolved", v.secured("2026-06-01", "gap-1"), nil)
	v.want("settling a voided contract", v.settle("2026-01-01"), nil)
	v.want("a reopening", v.reopen(LateCredit), nil)
	v.want("settling a voided contract again without new evidence", v.settle("2026-01-01"), nil)
	v.want("a second reopening", v.reopen(Correcting), nil)
	v.want("a new start with a cancellation", v.amend(RecordedInError, nil, "s start 2026-08-01 1"), nil)
	if v.state() != NoSettlement || v.life(v.sequence()) != "s start 2026-08-01 1; " {
		t.Errorf("a voided contract started again: %s %q", v.state(), v.life(v.sequence()))
	}
	both := open(t, testLimits)
	both.want("a start and an end in one batch", both.amend("", nil, "a start 2026-06-01 1", "b end 2026-06-11"), nil)
	if both.state() != PendingFinal {
		t.Errorf("state %s", both.state())
	}
	both.want("everything withdrawn while pending", both.amend("", []string{"a", "b"}), nil)
	if both.state() != PendingFinal {
		t.Errorf("an ended contract that became voided left the pending final settlement: %s", both.state())
	}
}

func TestEvidenceShapes(t *testing.T) {
	b := open(t, testLimits)
	b.want("a start and an end", b.amend("", nil, "a start 2026-06-01 1", "b end 2026-06-11"), nil)
	until := day(t, "2026-06-11")
	for name, evidence := range map[string]Evidence{
		"no kind":                           {},
		"an unknown kind":                   {Kind: "partial", Reference: "e", CoveredUntil: until},
		"secured without a reference":       {Kind: Secured, CoveredUntil: until},
		"secured with text for a reference": {Kind: Secured, Reference: "see the mail", CoveredUntil: until},
		"secured without a day":             {Kind: Secured, Reference: "e"},
		"secured with a reason":             {Kind: Secured, Reference: "e", CoveredUntil: until, Reason: NotAttempted},
		"secured with an owner":             {Kind: Secured, Reference: "e", CoveredUntil: until, OwnerID: "o"},
		"secured with a follow-up":          {Kind: Secured, Reference: "e", CoveredUntil: until, FollowUp: "f"},
		"secured naming a gap twice":        {Kind: Secured, Reference: "e", CoveredUntil: until, Resolves: []string{"g", "g"}},
		"secured naming text":               {Kind: Secured, Reference: "e", CoveredUntil: until, Resolves: []string{"the gap"}},
		"missing without a reason":          {Kind: Missing, OwnerID: "o", FollowUp: "f"},
		"missing with a reason of its own":  {Kind: Missing, Reason: "lost", OwnerID: "o", FollowUp: "f"},
		"missing without an owner":          {Kind: Missing, Reason: NotAttempted, FollowUp: "f"},
		"missing with a name for an owner":  {Kind: Missing, Reason: NotAttempted, OwnerID: "Kim Min-su", FollowUp: "f"},
		"missing without a follow-up":       {Kind: Missing, Reason: NotAttempted, OwnerID: "o"},
		"missing with a reference":          {Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f", Reference: "e"},
		"missing with a day":                {Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f", CoveredUntil: until},
		"missing that resolves":             {Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f", Resolves: []string{"g"}},
	} {
		_, err := b.r.RecordEvidence("w1", "c1", 1, "e-1", evidence)
		b.want(name, err, ErrInvalid)
	}
	for _, reason := range []GapReason{SourceUnavailable, AccessWithdrawnEmergency, ProviderRefused, NotAttempted, LateSourceReceived} {
		b.want("the reason "+string(reason), b.missing("gap-"+string(reason), reason), nil)
	}
	if _, err := b.r.Settle("w1", "c1", b.sequence(), "s-1", Day{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("settling without a day: %v", err)
	}
	if _, err := b.r.RecordEvidence("w1", "c1", 0, "e-2", Evidence{Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f"}); !errors.Is(err, ErrConflict) {
		t.Errorf("evidence with a stale expected sequence: %v", err)
	}
	if _, err := b.r.Settle("w1", "c1", 0, "s-2", until); !errors.Is(err, ErrConflict) {
		t.Errorf("settling with a stale expected sequence: %v", err)
	}
	if _, err := b.r.Reopen("w1", "c1", 0, "r-2", LateSource); !errors.Is(err, ErrConflict) {
		t.Errorf("reopening with a stale expected sequence: %v", err)
	}
}

func TestLimitsAndTheReserve(t *testing.T) {
	for name, limits := range map[string]Limits{"contracts": {0, 1, 1, 1}, "writes": {1, 0, 1, 1}, "reserve": {1, 1, 0, 1}, "batch": {1, 1, 1, 0}} {
		if _, err := NewRegistry(limits, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("no limit on %s: %v", name, err)
		}
	}
	if _, err := NewRegistry(testLimits, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("no clock: %v", err)
	}
	b := open(t, Limits{Contracts: 2, Writes: 3, Reserve: 2, Batch: 4})
	b.want("write 1", b.amend("", nil, "a start 2026-06-01 1"), nil)
	b.want("write 2", b.amend("", nil, "b suspend 2026-06-05"), nil)
	b.want("write 3", b.amend("", nil, "c resume 2026-06-08"), nil)
	b.want("an ordinary batch at the limit", b.amend("", nil, "d renew 2026-06-09"), ErrLimit)
	b.want("a batch that resumes billing at the limit", b.amend("", []string{"b", "c"}), ErrLimit)
	b.want("a suspension that carries a withdrawal", b.amend("", []string{"c"}, "d end 2026-06-09"), ErrLimit)
	b.want("an end beside another event", b.amend("", nil, "d renew 2026-06-09", "e end 2026-06-10"), ErrLimit)
	b.want("an end from the reserve", b.amend("", nil, "d end 2026-06-09"), nil)
	if records, _ := b.r.Records("w1", "c1"); len(records) != 4 || records[3].State != PendingFinal || b.state() != PendingFinal {
		t.Errorf("an end written from the reserve did not open the final settlement in the same write")
	}
	b.want("a gap record from the reserve", b.missing("gap-1", SourceUnavailable), nil)
	b.want("a secured record that names the gap, at the limit", b.secured("2026-06-09", "gap-1"), ErrLimit)
	b.want("a gap record when the reserve is used up", b.missing("gap-2", SourceUnavailable), ErrLimit)
	if view, _ := b.r.Settlement("w1", "c1", 5); len(view.Evidence) != 1 || view.Evidence[0].Resolved {
		t.Errorf("a secured record refused at the limit resolved its gap: %+v", view.Evidence)
	}
	u := open(t, Limits{Contracts: 2, Writes: 2, Reserve: 2, Batch: 4})
	u.want("write 1", u.amend("", nil, "a start 2026-06-01 1"), nil)
	u.want("write 2", u.amend("", nil, "b renew 2026-06-05"), nil)
	u.want("a suspension from the reserve", u.amend("", nil, "c suspend 2026-06-09"), nil)
	u.want("an end from the reserve", u.amend("", nil, "d end 2026-06-10"), nil)
	u.want("an end when the reserve is used up", u.amend("", []string{"d"}, "e end 2026-06-11"), ErrLimit)
	u.want("a gap record when the reserve is used up", u.missing("gap-1", SourceUnavailable), ErrLimit)
	if b.sequence() != 5 {
		t.Errorf("sequence %d", b.sequence())
	}

	e := open(t, Limits{Contracts: 2, Writes: 4, Reserve: 2, Batch: 4})
	e.want("write 1", e.amend("", nil, "a start 2026-06-01 1", "b end 2026-06-11"), nil)
	e.want("write 2", e.secured("2026-06-11"), nil)
	e.want("write 3", e.settle("2026-06-11"), nil)
	e.want("write 4", e.reopen(LateSource), nil)
	e.want("secured evidence at the limit", e.secured("2026-06-11"), ErrLimit)
	e.want("the cancellation of an end at the limit", e.amend(RecordedInError, []string{"b"}), ErrLimit)
	e.want("an end and a suspension from the reserve", e.amend("", []string{"b"}, "c end 2026-06-12"), ErrLimit)
	s := open(t, Limits{Contracts: 2, Writes: 3, Reserve: 1, Batch: 4})
	s.want("write 1", s.amend("", nil, "a start 2026-06-01 1", "b end 2026-06-11"), nil)
	s.want("write 2", s.secured("2026-06-11"), nil)
	s.want("write 3", s.settle("2026-06-11"), nil)
	s.want("a reopening from the reserve", s.reopen(LateSource), nil)
	s.want("a second reserve write", s.missing("gap-1", NotAttempted), ErrLimit)
	t3 := open(t, Limits{Contracts: 2, Writes: 2, Reserve: 1, Batch: 4})
	t3.want("write 1", t3.amend("", nil, "a start 2026-06-01 1", "b end 2026-06-11"), nil)
	t3.want("write 2", t3.secured("2026-06-11"), nil)
	t3.want("settling at the limit", t3.settle("2026-06-11"), ErrLimit)

	c := open(t, Limits{Contracts: 2, Writes: 5, Reserve: 1, Batch: 4})
	for _, id := range []string{"c1", "c2"} {
		c.id = id
		c.want("contract "+id, c.amend("", nil, "a start 2026-06-01 1"), nil)
	}
	c.id = "c3"
	c.want("a third contract", c.amend("", nil, "a start 2026-06-01 1"), ErrLimit)
	c.want("a refused batch for a new contract", c.amend("", nil, "a end 2026-06-01"), ErrRejected)
	if _, err := c.r.Sequence("w1", "c3"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a refused first batch created the contract: %v", err)
	}
	c.space = "w2"
	c.want("the same contract in another workspace", c.amend("", nil, "a start 2026-06-01 1"), nil)
	c.space, c.id = "w1", "c1"
	c.want("a further write of an existing contract", c.amend("", nil, "b renew 2026-06-05"), nil)
}

func TestOneWriteForOneExpectedSequence(t *testing.T) {
	b := open(t, testLimits)
	b.want("a start and an end", b.amend("", nil, "a start 2026-06-01 1", "b end 2026-06-11"), nil)
	b.want("evidence", b.secured("2026-06-11"), nil)
	var group sync.WaitGroup
	results := make(chan error, 32)
	asOf := day(t, "2026-07-01")
	for index := 0; index < 16; index++ {
		group.Add(2)
		moved := evs(t, fmt.Sprintf("end-%d end 2026-06-%d", index, 12+index))
		go func() {
			defer group.Done()
			_, err := b.r.Amend("w1", "c1", 2, fmt.Sprintf("move-%d", index), []string{"b"}, moved, "")
			results <- err
		}()
		go func() {
			defer group.Done()
			_, err := b.r.Settle("w1", "c1", 2, fmt.Sprintf("settle-%d", index), asOf)
			results <- err
		}()
	}
	group.Wait()
	close(results)
	written := 0
	for err := range results {
		switch {
		case err == nil:
			written++
		case !errors.Is(err, ErrConflict):
			t.Errorf("a losing write ended with %v", err)
		}
	}
	if written != 1 || b.sequence() != 3 {
		t.Errorf("%d writes for one expected sequence; sequence %d", written, b.sequence())
	}
	b.check()
	view, _ := b.r.Settlement("w1", "c1", 3)
	events, _ := b.r.Lifecycle("w1", "c1", 3)
	if (view.State == Settled) != (events[1].ID == "b") {
		t.Errorf("a settlement was applied to a lifecycle its caller did not see: %s with the end %s", view.State, events[1].ID)
	}
}

func TestARegistryThatIsNotBuilt(t *testing.T) {
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		if _, err := r.Amend("w1", "c1", 0, "x", nil, evs(t, "a start 2026-06-01 1"), ""); !errors.Is(err, ErrInvalid) {
			t.Errorf("amend %v", err)
		}
		if _, err := r.RecordEvidence("w1", "c1", 0, "x", Evidence{Kind: Missing, Reason: NotAttempted, OwnerID: "o", FollowUp: "f"}); !errors.Is(err, ErrInvalid) {
			t.Errorf("evidence %v", err)
		}
		if _, err := r.Settle("w1", "c1", 0, "x", day(t, "2026-06-01")); !errors.Is(err, ErrInvalid) {
			t.Errorf("settle %v", err)
		}
		if _, err := r.Reopen("w1", "c1", 0, "x", LateSource); !errors.Is(err, ErrInvalid) {
			t.Errorf("reopen %v", err)
		}
		if _, err := r.Sequence("w1", "c1"); !errors.Is(err, ErrInvalid) {
			t.Errorf("sequence %v", err)
		}
		if _, err := r.Lifecycle("w1", "c1", 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("lifecycle %v", err)
		}
		if _, err := r.StateOn("w1", "c1", 0, day(t, "2026-06-01")); !errors.Is(err, ErrInvalid) {
			t.Errorf("state %v", err)
		}
		if _, err := r.SegmentsAt("w1", "c1", 0, span(t, "2026-06-01", "2026-07-01")); !errors.Is(err, ErrInvalid) {
			t.Errorf("segments %v", err)
		}
		if _, err := r.Settlement("w1", "c1", 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("settlement %v", err)
		}
		if _, err := r.Records("w1", "c1"); !errors.Is(err, ErrInvalid) {
			t.Errorf("records %v", err)
		}
	}
	for id, valid := range map[string]bool{"a": true, "A.b_c:d-9": true, "": false, "a b": false, "a/b": false} {
		if validID(id) != valid {
			t.Errorf("identifier %q", id)
		}
	}
	long := make([]byte, 129)
	for index := range long {
		long[index] = 'x'
	}
	if validID(string(long)) || !validID(string(long[:128])) {
		t.Errorf("the length bound of an identifier")
	}
}
