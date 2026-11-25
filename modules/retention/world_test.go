package retention

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	ann    = Actor{ID: "ann", Kind: KindHuman}
	bob    = Actor{ID: "bob", Kind: KindHuman}
	cat    = Actor{ID: "cat", Kind: KindHuman}
	dan    = Actor{ID: "dan", Kind: KindHuman}
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	ingest = Actor{ID: "ingest", Kind: KindSystem}
	worker = Actor{ID: "worker", Kind: KindSystem}
)

type world struct {
	t   testing.TB
	r   *Registry
	now *time.Time
	mu  *sync.Mutex
}

func setup(t testing.TB) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	r, err := NewRegistry(func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now, mu: mu}
}

func (f world) advance(d time.Duration) {
	f.mu.Lock()
	*f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f world) at() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.now
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) put(id, class string, ageDays int, tenants []string, stores ...string) {
	f.t.Helper()
	if len(stores) == 0 {
		stores = []string{"db"}
	}
	f.must(f.r.Register(Artifact{ID: id, Class: class, Tenants: tenants, Anchor: f.at().Add(-time.Duration(ageDays) * day), Stores: stores}, ingest))
}

func (f world) approve(c Change) ProposalView {
	f.t.Helper()
	p, err := f.r.Propose(c, ann)
	f.must(err)
	v, err := f.r.Approve(p.Number, bob, p.Change.Impact)
	f.must(err)
	return v
}

func (f world) by(c Change, proposer, approver Actor) {
	f.t.Helper()
	p, err := f.r.Propose(c, proposer)
	f.must(err)
	f.must(second(f.r.Approve(p.Number, approver, p.Change.Impact)))
}

func (f world) try(c Change) error {
	p, err := f.r.Propose(c, ann)
	if err != nil {
		return err
	}
	_, err = f.r.Approve(p.Number, bob, p.Change.Impact)
	return err
}

func (f world) policy(class string, period, grace int) {
	f.t.Helper()
	f.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: class, Period: period, Grace: grace}})
}

func (f world) plan(class string, chunk int, ids ...string) string {
	f.t.Helper()
	f.approve(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: class, Artifacts: ids, Chunk: chunk, Lease: time.Hour, Reason: "expired"}})
	id := "run/" + itoa(int(f.r.runSeq))
	f.must(second(f.r.Run(id)))
	return id
}

func (f world) access(id string) string {
	f.t.Helper()
	v, err := f.r.Access(id)
	f.must(err)
	return v
}

func (f world) run(id string) RunView {
	f.t.Helper()
	v, err := f.r.Run(id)
	f.must(err)
	if len(v.Counts) != 9 {
		f.t.Fatalf("count keys %v", v.Counts)
	}
	sum := 0
	for _, n := range v.Counts {
		sum += n
	}
	if sum != len(v.Plan.Artifacts) {
		f.t.Fatalf("counts %v do not add to %d", v.Counts, len(v.Plan.Artifacts))
	}
	for s, c := range v.Stores {
		if c.Target != c.Deleted+c.Failed+c.Pending {
			f.t.Fatalf("store %s %+v", s, c)
		}
	}
	return v
}

func expect(t testing.TB, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || want == nil && err != nil {
		t.Errorf("%s: got %v, want %v", name, err, want)
	}
}

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var refusal Refusal
	if !errors.As(err, &refusal) || refusal.Reason != want {
		t.Errorf("%s: got %v, want %s", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}

var one = []string{"t1"}
