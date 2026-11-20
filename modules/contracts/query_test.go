package contracts

import (
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"
)

func assertCurrentPositionOracle(t testing.TB, b book) {
	t.Helper()
	seq := b.sequence()
	for _, date := range []string{"2026-05-31", "2026-06-01", "2026-06-04", "2026-06-05", "2026-06-11", "2026-06-12", "2026-06-19", "2026-06-20", "2026-07-01", "2026-07-02"} {
		want, err := b.r.StateOn(b.space, b.id, seq, day(t, date))
		if err != nil {
			t.Fatal(err)
		}
		got, at, err := b.r.CurrentPosition(b.space, b.id, day(t, date))
		if err != nil || got != want || at != seq {
			t.Fatal("current position differs from independent historical evaluation", date, got, want, at, seq, err)
		}
	}
}

func TestCurrentPositionMatchesHistoricalOracleAcrossTimelineCorrections(t *testing.T) {
	b := open(t, testLimits)
	initial := evs(t, "a start 2026-06-01 1", "b suspend 2026-06-11")
	out, err := b.r.Amend(b.space, b.id, 0, "initial", nil, initial, "")
	b.want("initial", err, nil)
	assertCurrentPositionOracle(t, b)
	before, _ := b.r.QueryVersion(b.space)
	replay, err := b.r.Amend(b.space, b.id, 0, "initial", nil, initial, "")
	if err != nil || !replay.Replay || replay.Sequence != out.Sequence {
		t.Fatal("timeline replay changed outcome", replay, err)
	}
	after, _ := b.r.QueryVersion(b.space)
	if before != after {
		t.Fatal("replay advanced query source")
	}
	b.want("backdated revision", b.amend("", nil, "c change 2026-06-05 2"), nil)
	assertCurrentPositionOracle(t, b)
	b.want("correct suspension and add resume", b.amend("", []string{"b"}, "d suspend 2026-06-12", "e resume 2026-06-20"), nil)
	assertCurrentPositionOracle(t, b)
	expected := map[string]Position{"2026-05-31": {NotStarted, 0}, "2026-06-05": {Active, 2}, "2026-06-11": {Active, 2}, "2026-06-12": {Suspended, 2}, "2026-06-20": {Active, 2}}
	for date, want := range expected {
		got, _, err := b.r.CurrentPosition(b.space, b.id, day(t, date))
		if err != nil || got != want {
			t.Fatal("canonical date boundary", date, got, want, err)
		}
	}
	before, _ = b.r.QueryVersion(b.space)
	b.want("invalid same-day event", b.amend("", nil, "bad renew 2026-06-12"), ErrRejected)
	after, _ = b.r.QueryVersion(b.space)
	if before != after {
		t.Fatal("rejected timeline advanced query source")
	}
	assertCurrentPositionOracle(t, b)
	b.want("future end", b.amend("", nil, "f end 2026-07-01"), nil)
	assertCurrentPositionOracle(t, b)
	for _, step := range []func() error{func() error { return b.secured("2026-07-01") }, func() error { return b.settle("2026-07-02") }, func() error { return b.reopen(LateSource) }} {
		previous, _ := b.r.QueryVersion(b.space)
		b.want("settlement write", step(), nil)
		current, _ := b.r.QueryVersion(b.space)
		if current != previous+1 {
			t.Fatal("successful shared commit omitted query generation")
		}
		assertCurrentPositionOracle(t, b)
	}
	b.want("withdraw final event", b.amend(ContractContinues, []string{"f"}), nil)
	assertCurrentPositionOracle(t, b)
}

func TestCurrentPositionScopeErrorsAndQueryExhaustionAreExplicit(t *testing.T) {
	b := open(t, testLimits)
	for _, args := range [][2]string{{"w1", "missing"}, {"w2", "c1"}} {
		if _, _, err := b.r.CurrentPosition(args[0], args[1], day(t, "2026-06-01")); !errors.Is(err, ErrNotFound) {
			t.Fatal("foreign/missing contract resolved", args, err)
		}
	}
	if _, _, err := b.r.CurrentPosition("w1", "c1", Day{}); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid reference day", err)
	}
	var absent *Registry
	if _, _, err := absent.CurrentPosition("w1", "c1", day(t, "2026-06-01")); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil current position", err)
	}
	b.want("start", b.amend("", nil, "a start 2026-06-01 1"), nil)
	otherBefore, _ := b.r.QueryVersion("w2")
	before, _ := b.r.QueryVersion("w1")
	if _, err := b.r.QueryVersion("bad workspace"); !errors.Is(err, ErrInvalid) {
		t.Fatal("malformed query workspace", err)
	}
	b.want("renew", b.amend("", nil, "b renew 2026-06-02"), nil)
	otherAfter, _ := b.r.QueryVersion("w2")
	after, _ := b.r.QueryVersion("w1")
	if otherBefore != otherAfter || after != before+1 {
		t.Fatal("query generation crossed workspace boundary")
	}
	b.r.mu.Lock()
	b.r.queryVersions["w1"] = math.MaxUint64 - 1
	b.r.mu.Unlock()
	b.want("write at query exhaustion", b.amend("", nil, "c change 2026-06-03 2"), nil)
	if _, err := b.r.QueryVersion("w1"); !errors.Is(err, ErrQueryUnavailable) {
		t.Fatal("exhausted generation reused a query basis", err)
	}
	b.want("domain still works after query exhaustion", b.amend("", nil, "d renew 2026-06-04"), nil)
	if _, err := b.r.QueryVersion("w1"); !errors.Is(err, ErrQueryUnavailable) {
		t.Fatal("query exhaustion was not sticky", err)
	}
	b.want("withdraw complete timeline", b.amend("", []string{"a", "b", "c", "d"}), nil)
	assertCurrentPositionOracle(t, b)
}

func TestCurrentPositionConcurrentReadsPreserveAcceptedTimeline(t *testing.T) {
	b := open(t, testLimits)
	b.want("start", b.amend("", nil, "a start 2026-06-01 1"), nil)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				got, seq, err := b.r.CurrentPosition("w1", "c1", day(t, "2026-06-15"))
				if err != nil || seq < 1 || !reflect.DeepEqual(got, Position{Active, 1}) {
					t.Error("concurrent current read", got, seq, err)
					return
				}
			}
		}()
	}
	for _, text := range []string{"b renew 2026-07-01", "c renew 2026-08-01", "d renew 2026-09-01"} {
		b.want("concurrent amendment", b.amend("", nil, text), nil)
	}
	wg.Wait()
}
