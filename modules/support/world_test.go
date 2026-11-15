package support

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	limits = Limits{Sites: 3, Destinations: 2, Bundles: 3, SiteBytes: 1 << 20, Bytes: 2 << 20, BundleLife: time.Hour, Proposals: 8}
	ann    = Actor{ID: "ann", Kind: KindHuman}
	bob    = Actor{ID: "bob", Kind: KindHuman}
	cat    = Actor{ID: "cat", Kind: KindHuman}
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	agent  = Actor{ID: "collector", Kind: KindSystem}
)

type world struct {
	t       testing.TB
	r       *Registry
	now     *time.Time
	clockMu *sync.Mutex
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	r, err := NewRegistry(l, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now, clockMu: mu}
}

func (f world) advance(d time.Duration) {
	f.clockMu.Lock()
	*f.now = f.now.Add(d)
	f.clockMu.Unlock()
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
	p, err = f.r.Approve(p.Number, bob)
	f.must(err)
	return p
}

func (f world) try(c Change) error {
	p, err := f.r.Propose(c, ann)
	if err != nil {
		return err
	}
	_, err = f.r.Approve(p.Number, bob)
	return err
}

func logSchema() Schema {
	return Schema{Source: SourceLog, Fields: []FieldSpec{
		{Key: "client", Class: ClassAddress},
		{Key: "contact", Class: ClassEmail},
		{Key: "elapsed", Class: ClassDuration},
		{Key: "event", Class: ClassEnum, Values: []string{"export.failed", "import.done", "login.failed"}},
		{Key: "node", Class: ClassHost},
		{Key: "operator", Class: ClassUser},
		{Key: "retries", Class: ClassInteger},
		{Key: "subject", Class: ClassIdentifier},
		{Key: "when", Class: ClassTimestamp},
	}}
}

func standard(t testing.TB) world {
	f := setupWith(t, limits)
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: "s1", Tenant: "t1"}})
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: "s2", Tenant: "t2"}})
	f.approve(Change{Kind: ChangeSchema, Schema: logSchema()})
	f.approve(Change{Kind: ChangeSchema, Schema: Schema{Source: SourceVersion, Fields: []FieldSpec{{Key: "product", Class: ClassVersion}}}})
	f.approve(Change{Kind: ChangeDestination, Destination: Destination{ID: "vendor", Host: "support.vendor.example", Port: 443}})
	return f
}

func sample() []Item {
	return []Item{
		{Source: SourceLog, Fields: []Field{{Key: "event", Text: "login.failed"}, {Key: "operator", Text: "Jane.Doe@corp"}, {Key: "client", Text: "10.1.2.3"},
			{Key: "when", Time: start}, {Key: "elapsed", Duration: 1500 * time.Millisecond}, {Key: "retries", Int: 3}}},
		{Source: SourceLog, Fields: []Field{{Key: "event", Text: "export.failed"}, {Key: "operator", Text: " jane.doe@CORP "}, {Key: "node", Text: "db01.acme.internal"},
			{Key: "subject", Int: 4111111111111111}, {Key: "contact", Text: "ops@acme.example"}}},
		{Source: SourceVersion, Fields: []Field{{Key: "product", Text: "2.4.0"}}},
	}
}

func (f world) bundle(site string) BundleView {
	f.t.Helper()
	b, err := f.r.Create(site, sample(), ann)
	f.must(err)
	return b
}

func sum(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

func expect(t testing.TB, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || want == nil && err != nil {
		t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var refusal Refusal
	if !errors.As(err, &refusal) || refusal.Reason != want {
		t.Errorf("%s: %v; want %s", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }
