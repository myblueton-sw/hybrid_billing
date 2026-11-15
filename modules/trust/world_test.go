package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	limits = Limits{Workspaces: 3, Connections: 4, Retired: 8, Endpoints: 3, Keys: 2, Destinations: 4, Events: 4, Items: 3, Quarantine: 3, Allowlist: 2,
		Redirects: 2, Body: 64, Skew: time.Minute, KeyLife: 30 * 24 * time.Hour}
	ann     = Actor{ID: "ann", Kind: KindHuman}
	bob     = Actor{ID: "bob", Kind: KindHuman}
	cat     = Actor{ID: "cat", Kind: KindHuman}
	bot     = Actor{ID: "bot", Kind: KindAssistant}
	sweeper = Actor{ID: "sweeper", Kind: KindSystem}
)

type world struct {
	t   testing.TB
	now *time.Time
	r   *Registry
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	r, err := NewRegistry(l, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, now: &now, r: r}
}

func setup(t testing.TB) world { return setupWith(t, limits) }

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) advance(d time.Duration) { *f.now = f.now.Add(d) }

func print(label string) Fingerprint {
	return Fingerprint(sha256.Sum256([]byte("synthetic-certificate-" + label)))
}

func signer(label string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte("synthetic-signing-seed-" + label))
	return ed25519.NewKeyFromSeed(seed[:])
}

func public(label string) []byte { return signer(label).Public().(ed25519.PublicKey) }

func (f world) connect(workspace, id string, prints ...Fingerprint) Admission {
	f.t.Helper()
	_, err := f.r.ProposeConnection(workspace, Connection{ID: id, Fingerprints: prints, Endpoints: []string{"usage"}}, ann)
	f.must(err)
	view, err := f.approveConnection(workspace, id, bob)
	f.must(err)
	return Admission{Workspace: workspace, Connection: id, Revision: view.Revision}
}

func (f world) key(workspace, id, version, label string, from, to time.Time) {
	f.t.Helper()
	_, err := f.r.AddKey(workspace, id, Key{Version: version, Public: public(label), NotBefore: from, NotAfter: to}, ann)
	f.must(err)
	_, err = f.r.ApproveKey(workspace, id, version, bob)
	f.must(err)
}

func sign(a Admission, label, version string, signed time.Time, body string, ids ...string) Push {
	p := Push{EventIDs: ids, KeyVersion: version, Signed: signed, Body: []byte(body)}
	p.Signature = ed25519.Sign(signer(label), Message(a.Workspace, a.Connection, p))
	return p
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

func errorsAs(err error, target *Refusal) bool { return errors.As(err, target) }

func (f world) approveConnection(workspace, id string, actor Actor) (ConnectionView, error) {
	proposal := 1
	if view, err := f.r.ConnectionOf(workspace, id); err == nil && view.Proposal > 0 {
		proposal = view.Proposal
	}
	return f.r.ApproveConnection(workspace, id, proposal, actor)
}
