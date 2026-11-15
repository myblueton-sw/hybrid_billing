package migration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	start = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	ann   = Actor{ID: "ann", Kind: KindHuman}
	bob   = Actor{ID: "bob", Kind: KindHuman}
	cat   = Actor{ID: "cat", Kind: KindHuman}
	bot   = Actor{ID: "bot", Kind: KindAssistant}
	imp   = Actor{ID: "imp-1", Kind: KindSystem}
	imp2  = Actor{ID: "imp-2", Kind: KindSystem}
)

type evidence struct {
	mu      sync.Mutex
	missing map[string]bool
	err     error
	during  func()
}

func (e *evidence) Present(tenant, d string) (bool, error) {
	e.mu.Lock()
	missing, err, during := e.missing[d], e.err, e.during
	e.mu.Unlock()
	if during != nil {
		during()
	}
	return !missing, err
}

type world struct {
	t       testing.TB
	r       *Registry
	ev      *evidence
	now     *time.Time
	clockMu *sync.Mutex
}

func setup(t testing.TB) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	ev := &evidence{missing: map[string]bool{}}
	r, err := NewRegistry(Config{Evidence: ev}, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, ev: ev, now: &now, clockMu: mu}
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

func sum(text string) string {
	s := sha256.Sum256([]byte(text))
	return hex.EncodeToString(s[:])
}

func standard(t testing.TB) world {
	f := setup(t)
	f.approve(Change{Kind: ChangeSource, Source: Source{ID: "src-1", Tenant: "t1", Importer: "imp-1", Workspaces: []string{"ws1", "ws2"}}})
	f.approve(Change{Kind: ChangeSource, Source: Source{ID: "src-2", Tenant: "t2", Importer: "imp-2", Workspaces: []string{"ws9"}}})
	f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "LEGACY", Workspace: "ws1", Series: "inv"}})
	f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-2", SourceSeries: "LEGACY", Workspace: "ws9", Series: "inv"}})
	return f
}

func doc(id string, number int64) Document {
	return Document{SourceID: id, Kind: DocumentInvoice, Workspace: "ws1", SourceSeries: "LEGACY", Number: number, Display: "INV-" + strconv.FormatInt(number, 10),
		IssuedAt: start.Add(-time.Hour), Currency: "KRW", Total: 1000 * number, Content: sum("render-" + id)}
}

func (f world) batch(source string, items []Item, tenant string) (string, BatchView) {
	f.t.Helper()
	s := f.r.sources[source]
	id := "b" + strconv.Itoa(len(s.order)+1)
	b := &batch{spec: BatchSpec{Source: source, ID: id, Count: len(items)}, codes: map[string]string{}, items: map[string]Item{}}
	for _, it := range items {
		if it.Document != nil {
			d := *it.Document
			d.IssuedAt = d.IssuedAt.UTC()
			b.codes[d.SourceID] = d.code()
		} else {
			k, err := keyView(*it.Key)
			f.must(err)
			b.codes[k.SourceID] = keyCode(k)
		}
	}
	f.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: source, ID: id, Count: len(items), Digest: batchDigest(tenant, source, b)}})
	actor := Actor{ID: s.view.Importer, Kind: KindSystem}
	for _, it := range items {
		f.must(f.r.Add(source, id, it, actor))
	}
	v, err := f.r.Seal(source, id, actor)
	f.must(err)
	return id, v
}

func (f world) accept(source, tenant string, items []Item) BatchView {
	f.t.Helper()
	id, v := f.batch(source, items, tenant)
	if v.State != BatchVerified {
		f.t.Fatalf("not verified %+v", v)
	}
	f.approve(Change{Kind: ChangeAccept, Accept: Accept{Source: source, Batch: id, Effect: v.Effect}})
	out, err := f.r.Batch(tenant, source, id)
	f.must(err)
	return out
}

func docs(list ...Document) []Item {
	out := []Item{}
	for index := range list {
		d := list[index]
		out = append(out, Item{Document: &d})
	}
	return out
}

func publicDER(t testing.TB) ([]byte, []byte, []byte) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	priv, _ := x509.MarshalPKCS8PrivateKey(key)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "legacy"}, NotBefore: start, NotAfter: start.Add(time.Hour)}
	cert, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv, cert
}

func sumBytes(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func otherPrivate(t testing.TB) ([]byte, []byte) {
	rk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	ek, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ec, _ := x509.MarshalECPrivateKey(ek)
	return x509.MarshalPKCS1PrivateKey(rk), ec
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
