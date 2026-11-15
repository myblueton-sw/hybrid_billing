package plugins

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	limits = Limits{Publishers: 3, Keys: 2, KeyLife: 3650 * 24 * time.Hour, Plugins: 4, Versions: 8, Workspaces: 4, Installations: 3, Tokens: 3,
		Fingerprints: 4, TokenLife: time.Hour, Endpoints: 2, Proposals: 8}
	ann    = Actor{ID: "ann", Kind: KindHuman}
	bob    = Actor{ID: "bob", Kind: KindHuman}
	cat    = Actor{ID: "cat", Kind: KindHuman}
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	run1   = Actor{ID: "run-1", Kind: KindSystem}
	run2   = Actor{ID: "run-2", Kind: KindSystem}
	runV   = Actor{ID: "run-v", Kind: KindSystem}
	boom   = errors.New("synthetic outage")
	events = Capability{Kind: CapabilityReadEvents, Target: "invoice.issued"}
	call   = Capability{Kind: CapabilityCallCommand, Target: "invoice.void"}
	screen = Capability{Kind: CapabilityServeScreen, Target: "overview"}
)

type destinations struct {
	mu     sync.Mutex
	active bool
	err    error
	during func()
	calls  int
}

func (d *destinations) Active(workspace, destination string) (bool, error) {
	d.mu.Lock()
	active, err, during := d.active, d.err, d.during
	d.calls++
	d.mu.Unlock()
	if during != nil {
		during()
	}
	return active, err
}

type entropy struct {
	err     error
	counter uint64
	repeat  bool
}

func (e *entropy) Read(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	if !e.repeat {
		e.counter++
	}
	for index := range p {
		p[index] = byte(index)
	}
	binary.BigEndian.PutUint64(p, e.counter)
	return len(p), nil
}

type world struct {
	t       testing.TB
	r       *Registry
	now     *time.Time
	dest    *destinations
	random  *entropy
	clockMu *sync.Mutex
}

func private(name string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte("synthetic-plugin-seed-" + name))
	return ed25519.NewKeyFromSeed(seed[:])
}

func public(name string) []byte { return private(name).Public().(ed25519.PublicKey) }

func fingerprint(name string) string {
	sum := sha256.Sum256([]byte("synthetic-certificate-" + name))
	return hex.EncodeToString(sum[:])
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	d := &destinations{active: true}
	e := &entropy{}
	r, err := NewRegistry(Config{Core: "2.4.0", Destinations: d, Random: e}, l, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now, dest: d, random: e, clockMu: mu}
}

func setup(t testing.TB) world { return setupWith(t, limits) }

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

func keyChange(publisher, version string) Change {
	return Change{Kind: ChangeKey, Key: PublisherKey{Publisher: publisher, Version: version, Public: public(publisher + version), NotBefore: start,
		NotAfter: start.Add(365 * 24 * time.Hour), Proof: ed25519.Sign(private(publisher+version), challenge(publisher, version))}}
}

func (f world) publisher(id string) {
	f.t.Helper()
	f.approve(Change{Kind: ChangePublisher, Publisher: Publisher{ID: id, Name: "Synthetic " + id}})
	f.approve(keyChange(id, "k1"))
}

func manifest(plugin, number string) Manifest {
	sum := sha256.Sum256([]byte("synthetic-package-" + plugin + number))
	return Manifest{Plugin: plugin, Version: number, Publisher: "acme", Key: "k1", Digest: hex.EncodeToString(sum[:]),
		Core: CoreRange{Min: "2.0.0", Max: "2.9.9"}, Capabilities: []Capability{call, events, screen}, Egress: []string{"crm"},
		Endpoints: []string{"api"}, Limits: RuntimeLimits{Timeout: time.Second, MemoryMiB: 64, Concurrency: 2},
		Storage: Storage{Namespace: plugin + "-data", Owner: "acme", RetentionDays: 30, Deletion: RetainForEvidence, Migration: MigrationAdditive},
		Screens: []Screen{{ID: "overview", Component: ComponentTable, Label: "Overview <b>", Fields: []Field{
			{Name: "total", Label: "Total", Source: plugin + "-data"}, {Name: "event", Label: "Event", Source: "invoice.issued"}}}}}
}

func signed(m Manifest) Change {
	return Change{Kind: ChangeVersion, Manifest: m, Signature: ed25519.Sign(private(m.Publisher+m.Key), m.encode())}
}

func (f world) version(m Manifest) {
	f.t.Helper()
	f.approve(signed(m))
}

func (f world) workspace(id, runtime string, verification bool) {
	f.t.Helper()
	f.approve(Change{Kind: ChangeWorkspace, Workspace: Workspace{ID: id, Runtime: runtime, Verification: verification}})
}

func at(kind, workspace, plugin, number string) Change {
	return Change{Kind: kind, Workspace: Workspace{ID: workspace}, Plugin: plugin, Version: number}
}

func (f world) install(workspace, plugin, number string) {
	f.t.Helper()
	f.approve(at(ChangeInstall, workspace, plugin, number))
}

func (f world) enable(workspace, plugin string) {
	f.t.Helper()
	f.approve(at(ChangeEnable, workspace, plugin, ""))
}

func standard(t testing.TB) world {
	f := setup(t)
	f.publisher("acme")
	f.version(manifest("ledger", "1.0.0"))
	f.version(manifest("ledger", "1.1.0"))
	f.workspace("w1", "run-1", false)
	f.workspace("w2", "run-2", false)
	f.workspace("v1", "run-v", true)
	return f
}

func running(t testing.TB) world {
	f := standard(t)
	for _, w := range []string{"w1", "w2", "v1"} {
		f.install(w, "ledger", "1.1.0")
		f.enable(w, "ledger")
	}
	return f
}

func (f world) state(workspace, plugin string) InstallationView {
	f.t.Helper()
	v, err := f.r.Installation(workspace, plugin)
	f.must(err)
	return v
}

func (f world) token(workspace string, actor Actor, scopes ...Capability) string {
	f.t.Helper()
	text, _, err := f.r.IssueToken(workspace, "ledger", scopes, time.Minute, actor)
	f.must(err)
	return text
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

func hexString(b []byte) string { return hex.EncodeToString(b) }

const tokenAnswer = "a172705cb7c0238803258ac4b650b43c4c6cedf6a175a70ac57e707211f951fb"

func errorsIs(err, target error) bool { return errors.Is(err, target) }

func same(a, b []byte) bool { return bytes.Equal(a, b) }
