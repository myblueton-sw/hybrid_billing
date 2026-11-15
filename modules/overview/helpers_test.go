package overview

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func reason(err error) string {
	var refusal Refusal
	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "other: " + err.Error()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func second[T any](_ T, err error) error { return err }

func expect(t *testing.T, label string, err error, want string) {
	t.Helper()
	if got := reason(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type scope struct {
	mu      sync.Mutex
	viewers map[string]Visibility
	fail    bool
}

func (s *scope) Visible(workspace, viewer string) (Visibility, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.viewers[viewer]
	if s.fail || !ok {
		return Visibility{}, errors.New("no grant")
	}
	return v, nil
}

func (s *scope) set(viewer string, v Visibility) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.viewers[viewer] = v
}

type counter struct {
	mu   sync.Mutex
	n    byte
	fail bool
}

func (c *counter) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return 0, errors.New("no entropy")
	}
	for index := range b {
		c.n++
		b[index] = c.n
	}
	return len(b), nil
}

type fixture struct {
	r      *Registry
	clock  *testClock
	scope  *scope
	random *counter
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Subjects: 200000, Facts: 400000, Filters: 4, Page: 100, Stale: time.Hour}
}

func open(t testing.TB, limits Limits) fixture {
	t.Helper()
	clock := &testClock{now: start}
	sc := &scope{viewers: map[string]Visibility{"admin": {All: true, Cost: true}, "job": {All: true, Cost: true}}}
	random := &counter{}
	r, err := NewRegistry(Config{Scope: sc, Random: random}, limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{r: r, clock: clock, scope: sc, random: random}
}

func subjectOf(customer, contract string) Subject {
	return Subject{Customer: customer, Contract: contract, Entity: "kr", Period: "2026-10", Currency: "KRW"}
}

func fact(id string, s Subject, source string, revision int, stage string, amounts map[string]string, occurred time.Time) Fact {
	return Fact{ID: id, Subject: s, Source: Source{ID: source, Revision: revision}, Stage: stage, Amounts: amounts, Occurred: occurred}
}

func (f fixture) ingest(t testing.TB, x Fact) Entry {
	t.Helper()
	return must(f.r.Ingest("ws", x, system("feed")))
}

func (f fixture) all(t testing.TB, viewer string, q Query) []Row {
	t.Helper()
	var rows []Row
	q.Limit = 100
	for {
		p := must(f.r.Query("ws", q, human(viewer)))
		rows = append(rows, p.Rows...)
		if p.Next == "" {
			return rows
		}
		q.Cursor = p.Next
	}
}
