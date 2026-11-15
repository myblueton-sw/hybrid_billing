package licensemode

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	start = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	ann   = Actor{ID: "ann", Kind: KindHuman}
	bob   = Actor{ID: "bob", Kind: KindHuman}
	cat   = Actor{ID: "cat", Kind: KindHuman}
	bot   = Actor{ID: "bot", Kind: KindAssistant}
	sys   = Actor{ID: "sys", Kind: KindSystem}
)

type authority struct {
	mu  sync.Mutex
	now time.Time
	ok  bool
}

func (a *authority) Now() (time.Time, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.now, a.ok
}

func (a *authority) set(t time.Time, ok bool) {
	a.mu.Lock()
	a.now, a.ok = t, ok
	a.mu.Unlock()
}

type world struct {
	t     testing.TB
	r     *Registry
	auth  *authority
	local *time.Time
	mu    *sync.Mutex
	keys  map[string]ed25519.PrivateKey
}

func setup(t testing.TB) world {
	t.Helper()
	local := start
	mu := &sync.Mutex{}
	auth := &authority{now: start, ok: true}
	r, err := NewRegistry(Config{Authority: auth}, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return local
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, auth: auth, local: &local, mu: mu, keys: map[string]ed25519.PrivateKey{}}
}

func (f world) advance(d time.Duration) {
	f.mu.Lock()
	*f.local = f.local.Add(d)
	now := *f.local
	f.mu.Unlock()
	f.auth.set(now, true)
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

func seed(name string) ed25519.PrivateKey {
	s := make([]byte, ed25519.SeedSize)
	copy(s, name)
	return ed25519.NewKeyFromSeed(s)
}

func keySpec(f world, id, issuer string) Key {
	priv := seed(id)
	f.keys[id] = priv
	pub := priv.Public().(ed25519.PublicKey)
	return Key{ID: id, Issuer: issuer, Public: hex.EncodeToString(pub), NotBefore: start.Add(-24 * time.Hour).Unix(), NotAfter: start.Add(365 * 24 * time.Hour).Unix(),
		AcceptUntil: start.Add(2 * 365 * 24 * time.Hour).Unix(), Proof: hex.EncodeToString(ed25519.Sign(priv, proofMessage(id, issuer)))}
}

func (f world) sign(c Claims) Signed {
	body := Encode(c)
	return Signed{Body: body, Signature: ed25519.Sign(f.keys[c.KeyID], signedMessage(body))}
}

func evaluation(id string) Claims {
	return Claims{Kind: DocEvaluation, ID: id, Issuer: "owner", KeyID: "k1", IssuedAt: start.Add(-time.Hour).Unix(), SubjectID: "acme", Subject: "Acme Ltd",
		Product: "hybrid-billing", Installation: "i1", Scope: []string{"core"}, Terms: "eval-terms-v1"}
}

func commercial(id, agreement string, from, to time.Time) Claims {
	return Claims{Kind: DocCommercial, ID: id, Issuer: "owner", KeyID: "k1", IssuedAt: start.Add(-time.Hour).Unix(), SubjectID: "acme", Subject: "Acme Ltd",
		Agreement: agreement, Product: "hybrid-billing", Installation: "i1", Scope: []string{"core", "erp"}, Channel: ChannelDirect, Metric: MetricInstallations,
		Start: from.Unix(), End: to.Unix()}
}

func standard(t testing.TB) world {
	f := setup(t)
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "k1", "owner")})
	f.must(f.r.Install("i1", "hybrid-billing", ann))
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
