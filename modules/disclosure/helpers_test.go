package disclosure

import (
	"bytes"
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

func kind(err error) string {
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
	case errors.Is(err, ErrRejected):
		return "rejected"
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

func third[A, B any](_ A, _ B, err error) error { return err }

func expect(t *testing.T, label string, err error, want string) {
	t.Helper()
	if got := kind(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type counter struct{ n uint64 }

func (c *counter) Read(p []byte) (int, error) {
	for i := range p {
		if i%8 == 0 {
			c.n++
		}
		p[i] = byte(c.n >> (8 * uint(i%8)))
	}
	return len(p), nil
}

type repeating struct{}

func (repeating) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 7
	}
	return len(p), nil
}

func keys(workspace string) ([]byte, error) {
	if workspace == "nokey" {
		return nil, errors.New("unavailable")
	}
	return []byte("test-key-" + workspace), nil
}

type fixture struct {
	r     *Registry
	clock *testClock
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Fields: 12, Policies: 3, Recipients: 4, Projections: 20, Packages: 20, Links: 20, Rows: 10, Combinations: 4}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(Config{Keys: keys, Random: &counter{}}, testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r: r, clock: clock}
	for _, field := range []Field{
		{Name: "customer", Class: ClassCustomer}, {Name: "service", Class: ClassPublic}, {Name: "amount", Class: ClassCustomer},
		{Name: "source", Class: ClassCustomer, Locator: true}, {Name: "cost", Class: ClassInternal}, {Name: "margin", Class: ClassSecret},
		{Name: "region", Class: ClassCustomer},
	} {
		must(r.Classify("w1", field, human("steward")))
	}
	f.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer", "service", "amount", "source"}, RolePartner: {"customer", "service"}, RoleAuditor: {"customer", "service", "amount", "cost"}}})
	f.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	return f
}

func (f fixture) policy(t *testing.T, p Policy) PolicyView {
	t.Helper()
	v := must(f.r.Propose("w1", p, human("author")))
	return must(f.r.Confirm("w1", p.ID, v.Revision, human("approver")))
}

func (f fixture) grant(t *testing.T, rec Recipient) GrantView {
	t.Helper()
	v := must(f.r.Grant("w1", rec, human("author")))
	return must(f.r.ConfirmGrant("w1", rec.ID, v.Revision, human("approver")))
}

func rows() []Row {
	return []Row{
		{"customer": "acme", "service": "compute", "amount": "100.00", "source": "s3://bucket/acme/2026-10.csv", "cost": "7777.77", "margin": "3333.33", "newfield": "x"},
		{"customer": "beta", "service": "compute", "amount": "999.00", "source": "s3://bucket/beta/2026-10.csv", "cost": "1", "margin": "2"},
		{"customer": "acme", "service": "storage", "amount": "=HYPERLINK(\"x\")", "source": "", "cost": "1", "margin": "2"},
	}
}

func contains(b []byte, s string) bool { return bytes.Contains(b, []byte(s)) }
