package sandbox

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	ann    = Actor{ID: "ann", Kind: KindHuman}
	bob    = Actor{ID: "bob", Kind: KindHuman}
	cat    = Actor{ID: "cat", Kind: KindHuman}
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	tool   = Actor{ID: "reset-tool", Kind: KindSystem}
	prod   = Environment{ID: "prod", Mode: ModeProduction, Database: "billing", KeyRing: "ring", EvidencePath: "/srv/evidence", NumberPrefix: "INV-"}
	demo   = Environment{ID: "demo", Mode: ModeDemo, Database: "demo_billing", KeyRing: "demo_ring", EvidencePath: "/srv/demo/evidence", NumberPrefix: "DEMO-"}
	demo2  = Environment{ID: "demo2", Mode: ModeDemo, Database: "demo2_billing", KeyRing: "demo2_ring", EvidencePath: "/srv/demo2", NumberPrefix: "DEMOB-"}
	erpDst = Destination{ID: "erp-mock", Environment: "demo", Kind: DestinationERP, Host: "erp.mock.test", Port: 8069}
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

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) approve(c Change) ProposalView {
	f.t.Helper()
	p, err := f.r.Propose(c, ann)
	f.must(err)
	v, err := f.r.Approve(p.Number, bob)
	f.must(err)
	return v
}

func (f world) try(c Change) error {
	p, err := f.r.Propose(c, ann)
	if err != nil {
		return err
	}
	_, err = f.r.Approve(p.Number, bob)
	return err
}

func env(e Environment) Change { return Change{Kind: ChangeEnvironment, Environment: e} }

func dst(d Destination) Change { return Change{Kind: ChangeDestination, Destination: d} }

func schema() Schema {
	return Schema{ID: "erp-demo", Fields: []FieldSpec{
		{Key: "batch_size", Kind: FieldInteger},
		{Key: "dry_run", Kind: FieldFlag},
		{Key: "erp", Kind: FieldDestination, Values: []string{DestinationERP}},
		{Key: "mode", Kind: FieldEnum, Values: []string{"file", "odoo-test"}},
	}}
}

func standard(t testing.TB) world {
	f := setup(t)
	f.approve(env(prod))
	f.approve(env(demo))
	f.approve(env(demo2))
	f.approve(dst(erpDst))
	f.approve(Change{Kind: ChangeSchema, Schema: schema()})
	return f
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
