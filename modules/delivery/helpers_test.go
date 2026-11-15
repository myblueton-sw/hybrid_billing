package delivery

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/disclosure"
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

type packages struct {
	mu         sync.Mutex
	items      map[string]disclosure.Validity
	recipients map[string]bool
	fail       bool
}

func (p *packages) Validity(workspace, pkg string) (disclosure.Validity, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.items[workspace+"|"+pkg]
	if p.fail || !ok {
		return disclosure.Validity{}, disclosure.ErrNotFound
	}
	return v, nil
}

func (p *packages) Recipient(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.recipients[id]
}

func (p *packages) put(workspace string, v disclosure.Validity) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[workspace+"|"+v.Package] = v
}

func (p *packages) lapse(workspace, pkg, why string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v := p.items[workspace+"|"+pkg]
	v.Live, v.Reason = false, why
	p.items[workspace+"|"+pkg] = v
}

type counter struct {
	mu    sync.Mutex
	n     byte
	fail  bool
	stuck bool
}

func (c *counter) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return 0, errors.New("no entropy")
	}
	if !c.stuck {
		c.n++
	}
	for index := range b {
		b[index] = c.n
	}
	return len(b), nil
}

type fixture struct {
	r      *Registry
	clock  *testClock
	pkgs   *packages
	random *counter
}

func testLimits() Limits {
	return Limits{Workspaces: 2, Channels: 8, Deliveries: 20, Queue: 10, Attempts: 3, Lease: 10 * time.Minute}
}

func open(t testing.TB, limits Limits) fixture {
	t.Helper()
	clock := &testClock{now: start}
	pkgs := &packages{items: map[string]disclosure.Validity{}, recipients: map[string]bool{"acme": true, "beta": true}}
	random := &counter{}
	r, err := NewRegistry(Config{Packages: pkgs, Random: random}, limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{r: r, clock: clock, pkgs: pkgs, random: random}
}

func pkg(id, recipient string) disclosure.Validity {
	return disclosure.Validity{Package: id, Recipient: recipient, FileDigest: "digest-" + id, Policy: "std", Revision: 1, Grant: 1, Live: true}
}

func proven() Diagnosis {
	return Diagnosis{FieldControl: true, AttachmentControl: true, Evidence: Evidence{Kind: EvidenceChannelTest, Reference: "test-run-1"}}
}

func standard(t testing.TB) fixture {
	t.Helper()
	f := open(t, testLimits())
	must(f.r.Register("ws", Channel{ID: "mail-acme", Kind: ChannelMail, Recipient: "acme", Target: "mailbox-1"}, human("ops")))
	must(f.r.Register("ws", Channel{ID: "portal-acme", Kind: ChannelPortal, Recipient: "acme", Target: "portal-1"}, human("ops")))
	must(f.r.Register("ws", Channel{ID: "erp-acme", Kind: ChannelERP, Recipient: "acme", Target: "erp-1"}, human("ops")))
	must(f.r.Register("ws", Channel{ID: "mail-beta", Kind: ChannelMail, Recipient: "beta", Target: "mailbox-2"}, human("ops")))
	d := must(f.r.Diagnose("ws", "mail-acme", proven(), human("tester")))
	must(f.r.Open("ws", "mail-acme", d.Revision, human("lead")))
	f.pkgs.put("ws", pkg("p1", "acme"))
	f.pkgs.put("ws", pkg("p2", "acme"))
	f.pkgs.put("ws", pkg("p3", "beta"))
	return f
}
