package webhooks

import (
	"crypto/sha256"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/events"
)

var (
	start   = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	limits  = Limits{Workspaces: 2, Subscriptions: 3, Types: 2, Lease: time.Minute, Backoff: time.Second, MaxBackoff: time.Minute, Attempts: 3, MaxAge: time.Hour, Dead: 2, Batch: 5, Work: 2}
	ann     = Actor{ID: "ann", Kind: KindHuman}
	bob     = Actor{ID: "bob", Kind: KindHuman}
	bot     = Actor{ID: "bot", Kind: KindAssistant}
	relay   = Actor{ID: "relay", Kind: KindSystem}
	boom    = errors.New("synthetic outage")
	invoice = events.Type{Name: "invoice.issued", SchemaVersion: "v1", Fields: []string{"invoice_id"}}
)

type outbox struct {
	r        *events.Registry
	boundErr error
	readErr  error
	onRead   func()
	mu       sync.Mutex
	reads    int
}

func (o *outbox) Read(workspace string, after uint64, limit int) (events.Page, error) {
	o.mu.Lock()
	o.reads++
	o.mu.Unlock()
	if o.onRead != nil {
		o.onRead()
	}
	if o.readErr != nil {
		return events.Page{}, o.readErr
	}
	return o.r.Read(workspace, after, limit, events.Actor{ID: "relay", Kind: events.KindSystem})
}

func (o *outbox) Bounds(workspace string) (uint64, uint64, error) {
	if o.boundErr != nil {
		return 0, 0, o.boundErr
	}
	return o.r.Bounds(workspace, events.Actor{ID: "relay", Kind: events.KindSystem})
}

type fakes struct {
	mu          sync.Mutex
	states      map[string]string
	stateErr    error
	subscribe   bool
	subErr      error
	allowed     bool
	allowErr    error
	version     string
	versionErr  error
	signErr     error
	hold        chan struct{}
	entered     chan struct{}
	redirectTo  string
	redirectErr error
	onState     func()
}

func (f *fakes) State(workspace, destination string) (string, error) {
	f.mu.Lock()
	hook := f.onState
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.states[destination], f.stateErr
}

func (f *fakes) Subscribe(string, string, []string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subscribe, f.subErr
}

func (f *fakes) Allowed(string, string, events.Stored) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.allowed, f.allowErr
}

func (f *fakes) Current(string, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.version, f.versionErr
}

func (f *fakes) Sign(workspace, subscription, version string, message []byte) ([]byte, error) {
	f.mu.Lock()
	hold, entered, err := f.hold, f.entered, f.signErr
	f.mu.Unlock()
	if hold != nil {
		entered <- struct{}{}
		<-hold
	}
	sum := sha256.Sum256(append([]byte(version+"|"), message...))
	return sum[:], err
}

func (f *fakes) Check(workspace, destination, current string, hop int, location string) (string, string, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.redirectTo, location, hop + 1, f.redirectErr
}

func (f *fakes) set(change func(*fakes)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(f)
}

type world struct {
	t   testing.TB
	now *time.Time
	ev  *events.Registry
	out *outbox
	fk  *fakes
	r   *Registry
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	clock := func() time.Time { return now }
	ev, err := events.NewRegistry(events.Limits{Workspaces: 2, Types: 2, Events: 1000, Retention: time.Hour, Page: 100, Sources: 1, Seen: 1, InboxRetention: time.Hour,
		Revisions: 1, Reservation: time.Minute}, clock)
	if err != nil {
		t.Fatal(err)
	}
	out := &outbox{r: ev}
	fk := &fakes{states: map[string]string{"d1": DestinationActive, "d2": DestinationActive}, subscribe: true, allowed: true, version: "k1"}
	r, err := NewRegistry(Config{Outbox: out, Destinations: fk, Permits: fk, Signer: fk, Redirects: fk}, l, clock)
	if err != nil {
		t.Fatal(err)
	}
	f := world{t: t, now: &now, ev: ev, out: out, fk: fk, r: r}
	_, err = ev.ProposeType("w1", invoice, events.Actor{ID: "ann", Kind: events.KindHuman})
	f.must(err)
	_, err = ev.ApproveType("w1", invoice.Name, invoice.SchemaVersion, events.Actor{ID: "bob", Kind: events.KindHuman})
	f.must(err)
	_, err = ev.ProposeType("w1", events.Type{Name: "other", SchemaVersion: "v1"}, events.Actor{ID: "ann", Kind: events.KindHuman})
	f.must(err)
	_, err = ev.ApproveType("w1", "other", "v1", events.Actor{ID: "bob", Kind: events.KindHuman})
	f.must(err)
	return f
}

func setup(t testing.TB) world { return setupWith(t, limits) }

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) advance(d time.Duration) { *f.now = f.now.Add(d) }

func (f world) publish(id, kind string) {
	f.t.Helper()
	e := events.Event{ID: id, Source: "urn:billing", Type: kind, SchemaVersion: "v1"}
	if kind == invoice.Name {
		e.Data = map[string]string{"invoice_id": id}
	}
	_, err := f.ev.Publish("w1", e, events.Actor{ID: "relay", Kind: events.KindSystem})
	f.must(err)
}

func (f world) subscribe(id, destination string, types ...string) {
	f.t.Helper()
	if len(types) == 0 {
		types = []string{invoice.Name}
	}
	_, err := f.r.ProposeSubscription("w1", Subscription{ID: id, Subscriber: "acme", Destination: destination, Types: types}, ann)
	f.must(err)
	_, err = f.r.ApproveSubscription("w1", id, bob)
	f.must(err)
}

func (f world) dispatch() []Lease {
	f.t.Helper()
	leases, err := f.r.Dispatch("w1", limits.Batch, relay)
	f.must(err)
	return leases
}

func (f world) confirm(l Lease) Signed {
	f.t.Helper()
	signed, err := f.r.Confirm("w1", l.Delivery, l.Token, relay)
	f.must(err)
	return signed
}

func (f world) report(l Lease, status string, retry time.Duration) DeliveryView {
	f.t.Helper()
	view, err := f.r.Report("w1", l.Delivery, l.Token, Outcome{Status: status, RetryAfter: retry}, relay)
	f.must(err)
	return view
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

func third[T, U any](_ T, _ U, err error) error { return err }
