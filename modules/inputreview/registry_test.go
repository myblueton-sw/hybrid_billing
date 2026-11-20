package inputreview

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	unchecked  bool
	action     string
	evidence   Evidence
	allow      bool
	currentErr error
	calls      int
}

func (f *fixture) Check(scope Scope, actor, action string) error {
	f.calls++
	if !f.allow || f.action != "" && action != f.action || scope != f.evidence.Scope || actor != "operator" || action != ConfirmAction && action != RejectAction {
		return ErrRejected
	}
	return nil
}

func (f *fixture) Compare(workspace, run string) (Evidence, error) {
	if f.currentErr != nil {
		return Evidence{}, f.currentErr
	}
	if !f.unchecked && (workspace != f.evidence.Scope.Workspace || run != f.evidence.Run) {
		return Evidence{}, ErrNotFound
	}
	return f.evidence, nil
}

func setup(t *testing.T) (*Registry, *fixture, Intent) {
	t.Helper()
	f := &fixture{allow: true, evidence: Evidence{Scope: Scope{"ws", "provider", "account", "2026-10"}, Run: "run", Digest: strings.Repeat("a", 64), Method: "independent-v1", Verdict: Match}}
	r, err := NewRegistry(10, f, f, func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Compare("ws", "run", "review"); err != nil {
		t.Fatal(err)
	}
	return r, f, Intent{Workspace: "ws", ID: "review", Actor: "operator", Rationale: "Reviewed complete synthetic evidence", Digest: f.evidence.Digest}
}

func TestConfirmPublishesExactlyOnceAndRetainsHistory(t *testing.T) {
	r, f, in := setup(t)
	got, err := r.Confirm(in)
	if err != nil || got.State != Eligible || got.Decision.ScopeRevision != 1 {
		t.Fatal(got, err)
	}
	publication, err := r.Current(f.evidence.Scope)
	if err != nil || publication.Record != in.ID || publication.Revision != 1 || publication.Digest != in.Digest {
		t.Fatal(publication, err)
	}
	again, err := r.Confirm(in)
	if err != nil || !reflect.DeepEqual(got, again) || f.calls != 2 {
		t.Fatal("repeat", again, err, f.calls)
	}
	got.Decision.Actor = "changed"
	if history, _ := r.Record("ws", "review"); history.Decision.Actor != "operator" {
		t.Fatal("alias", history)
	}
	if _, err := r.Reject(in); !errors.Is(err, ErrConflict) {
		t.Fatal("published rejection", err)
	}
	f.allow = false
	if _, err := r.Confirm(in); !errors.Is(err, ErrRejected) {
		t.Fatal("revoked replay", err)
	}
	if history, _ := r.Record("ws", "review"); history.State != Eligible {
		t.Fatal("history changed")
	}
}

func TestChangedEvidenceAndLaterPublicationRefuseReplay(t *testing.T) {
	r, f, in := setup(t)
	first, err := r.Confirm(in)
	if err != nil {
		t.Fatal(err)
	}
	f.evidence.Digest = strings.Repeat("b", 64)
	if _, err := r.Confirm(in); !errors.Is(err, ErrConflict) {
		t.Fatal("stale evidence", err)
	}
	if _, err := r.Compare("ws", "run", "next"); err != nil {
		t.Fatal(err)
	}
	next := in
	next.ID = "next"
	next.Digest = f.evidence.Digest
	next.ExpectedScope = 1
	if _, err := r.Confirm(next); err != nil {
		t.Fatal(err)
	}
	f.evidence.Digest = in.Digest
	if _, err := r.Confirm(in); !errors.Is(err, ErrConflict) {
		t.Fatal("later publication replaced", err)
	}
	if history, _ := r.Record("ws", "review"); !reflect.DeepEqual(first, history) {
		t.Fatal("history changed")
	}
	if current, _ := r.Current(f.evidence.Scope); current.Record != "next" || current.Revision != 2 {
		t.Fatal(current)
	}
}

func TestBlockedRejectAndImmutableIntent(t *testing.T) {
	r, f, in := setup(t)
	f.evidence.Verdict = Difference
	f.evidence.Findings = []string{"row_difference"}
	f.evidence.Digest = strings.Repeat("c", 64)
	blocked, err := r.Compare("ws", "run", "blocked")
	if err != nil || blocked.State != Blocked {
		t.Fatal(blocked, err)
	}
	blocked.Evidence.Findings[0] = "tampered"
	in.ID = "blocked"
	in.Digest = f.evidence.Digest
	if _, err := r.Confirm(in); !errors.Is(err, ErrRejected) {
		t.Fatal("blocked publication", err)
	}
	rejected, err := r.Reject(in)
	if err != nil || rejected.State != Rejected || rejected.Evidence.Findings[0] != "row_difference" {
		t.Fatal(rejected, err)
	}
	if _, err := r.Current(f.evidence.Scope); !errors.Is(err, ErrNotFound) {
		t.Fatal("rejection published", err)
	}
	if repeat, err := r.Reject(in); err != nil || !reflect.DeepEqual(rejected, repeat) {
		t.Fatal("reject retry", repeat, err)
	}
	in.Rationale = "Different reason"
	if _, err := r.Reject(in); !errors.Is(err, ErrConflict) {
		t.Fatal("intent replaced", err)
	}
}

func TestConcurrentConfirmationHasOneWinner(t *testing.T) {
	r, f, in := setup(t)
	if _, err := r.Compare("ws", "run", "other"); err != nil {
		t.Fatal(err)
	}
	other := in
	other.ID = "other"
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, intent := range []Intent{in, other} {
		wg.Add(1)
		go func(v Intent) { defer wg.Done(); _, err := r.Confirm(v); results <- err }(intent)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
	if current, _ := r.Current(f.evidence.Scope); current.Revision != 1 {
		t.Fatal(current)
	}
}

func TestDependencyFailureCapacityAndOverflowDoNotPublish(t *testing.T) {
	r, f, in := setup(t)
	f.currentErr = errors.New("comparison unavailable")
	if _, err := r.Confirm(in); err != f.currentErr {
		t.Fatal(err)
	}
	if record, _ := r.Record("ws", "review"); record.State != Awaiting || record.Decision != nil {
		t.Fatal(record)
	}
	f.currentErr = nil
	r.limit = 1
	if _, err := r.Compare("ws", "run", "another"); !errors.Is(err, ErrLimit) {
		t.Fatal("capacity", err)
	}
	if _, err := r.Compare("ws", "run", "review"); err != nil {
		t.Fatal("capacity replay", err)
	}
	r.publications[f.evidence.Scope] = Publication{Scope: f.evidence.Scope, Revision: MaxRevision, Record: "prior", Digest: in.Digest}
	in.ExpectedScope = MaxRevision
	if _, err := r.Confirm(in); !errors.Is(err, ErrLimit) {
		t.Fatal("overflow", err)
	}
	if record, _ := r.Record("ws", "review"); record.Decision != nil {
		t.Fatal("partial decision", record)
	}
	if current, _ := r.Current(f.evidence.Scope); current.Record != "prior" {
		t.Fatal("partial publication", current)
	}
}

func TestPortsScopeAndZeroRegistryRefuse(t *testing.T) {
	r, f, in := setup(t)
	var absent *fixture
	if _, err := NewRegistry(1, absent, f, time.Now); err == nil {
		t.Fatal("typed nil authority")
	}
	if _, err := NewRegistry(1, f, absent, time.Now); err == nil {
		t.Fatal("typed nil evidence")
	}
	var zero Registry
	if _, err := zero.Confirm(in); !errors.Is(err, ErrInvalid) {
		t.Fatal("zero registry", err)
	}
	in.Workspace = "foreign"
	if _, err := r.Confirm(in); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign record", err)
	}
	if _, err := r.Record("foreign", "review"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	f.evidence.Scope.Workspace = "foreign"
	if _, err := r.Compare("ws", "run", "foreign"); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign evidence", err)
	}
}

func TestUntrustedEvidenceResultIsValidated(t *testing.T) {
	for _, change := range []func(*Evidence){
		func(e *Evidence) { e.Scope.Workspace = "foreign" },
		func(e *Evidence) { e.Run = "other" },
		func(e *Evidence) { e.Verdict = Match; e.Findings = []string{"difference"} },
	} {
		r, f, in := setup(t)
		f.unchecked = true
		change(&f.evidence)
		if _, err := r.Compare("ws", "run", "bad"); !errors.Is(err, ErrInvalid) {
			t.Fatal("malformed port result accepted", err)
		}
		f.allow = true
		if len(r.records) != 1 || len(r.publications) != 0 {
			t.Fatal("port error mutated records")
		}
		if old, _ := r.Record(in.Workspace, in.ID); old.State != Awaiting {
			t.Fatal(old)
		}
	}
}

func TestRejectAuthorityCannotConfirmAndClockIsMonotonic(t *testing.T) {
	r, f, in := setup(t)
	f.action = RejectAction
	if _, err := r.Confirm(in); !errors.Is(err, ErrRejected) {
		t.Fatal("reject authority confirmed", err)
	}
	initial, _ := r.Record(in.Workspace, in.ID)
	r.clock = func() time.Time { return initial.Compared.Add(-time.Hour) }
	rejected, err := r.Reject(in)
	if err != nil || !rejected.Decision.At.Equal(initial.Compared) {
		t.Fatal("clock rollback", rejected, err)
	}
	r, f, in = setup(t)
	r.clock = func() time.Time { return time.Time{} }
	if _, err := r.Confirm(in); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid clock", err)
	}
	if record, _ := r.Record(in.Workspace, in.ID); record.Decision != nil {
		t.Fatal("invalid clock decision", record)
	}
	if _, err := r.Current(f.evidence.Scope); !errors.Is(err, ErrNotFound) {
		t.Fatal("invalid clock publication", err)
	}
}
