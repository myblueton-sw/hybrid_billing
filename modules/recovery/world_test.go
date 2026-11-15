package recovery

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	erp   = Actor{ID: "w-erp", Kind: KindSystem}
	iss   = Actor{ID: "w-iss", Kind: KindSystem}
	ver   = Actor{ID: "v-1", Kind: KindSystem}
)

type world struct {
	t       testing.TB
	r       *Registry
	now     *time.Time
	clockMu *sync.Mutex
}

func sum(text string) string {
	s := sha256.Sum256([]byte(text))
	return hex.EncodeToString(s[:])
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
	return world{t: t, r: r, now: &now, clockMu: mu}
}

func (f world) advance(d time.Duration) {
	f.clockMu.Lock()
	*f.now = f.now.Add(d)
	f.clockMu.Unlock()
}

func (f world) at() time.Time {
	f.clockMu.Lock()
	defer f.clockMu.Unlock()
	return *f.now
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

func (f world) installation(id string, prefix string) {
	f.t.Helper()
	f.approve(Change{Kind: ChangeInstallation, Installation: Installation{ID: id, Tenant: "t-" + id, Skew: time.Minute}})
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: id, ID: prefix + "w-erp", Kind: WriterERP, MaxCall: time.Minute}})
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: id, ID: prefix + "w-iss", Kind: WriterIssuance, MaxCall: 2 * time.Minute}})
	f.approve(Change{Kind: ChangeVerifier, Verifier: Verifier{Installation: id, ID: prefix + "v-1"}})
	f.approve(Change{Kind: ChangeExternal, External: External{Installation: id, ID: "erp-x", Kind: WriterERP}})
	f.approve(Change{Kind: ChangeExternal, External: External{Installation: id, ID: "iss-x", Kind: WriterIssuance}})
}

func standard(t testing.TB) world {
	f := setup(t)
	f.installation("i1", "")
	f.installation("i2", "b-")
	return f
}

func manifest() []Inventory {
	out := []Inventory{}
	for _, d := range domains {
		out = append(out, Inventory{Domain: d, Items: 10, Chunks: []Chunk{{Start: 0, End: 4, Checksum: sum(d + "a")}, {Start: 4, End: 10, Checksum: sum(d + "b")}}})
	}
	return out
}

func (f world) point(id string, asOf time.Time) RestorePoint {
	gen := f.r.installations[id].generation
	m := manifest()
	return RestorePoint{Installation: id, Generation: gen, Backup: sum("backup"), ManifestDigest: manifestDigest(m), AsOf: asOf, Manifest: m}
}

func (f world) begin(id string, asOf time.Time) {
	f.t.Helper()
	f.approve(Change{Kind: ChangeBegin, Begin: f.point(id, asOf)})
}

func (f world) verifyAll(id string, actor Actor) {
	f.t.Helper()
	for _, inv := range manifest() {
		st, err := f.r.Report(id, inv, actor)
		f.must(err)
		if st.Status != DomainVerified {
			f.t.Fatalf("not verified %+v", st)
		}
	}
}

func (f world) reconcileAll(id string, actor Actor, from, through time.Time) {
	f.t.Helper()
	for _, x := range []string{"erp-x", "iss-x"} {
		f.must(second(f.r.Reconcile(id, x, Page{Number: 1, From: from, Through: through, Final: true, Rotated: true}, actor)))
	}
}

func (f world) unlock(id string) error {
	s := f.r.installations[id]
	lost := lostDigest(nil)
	if s.current != nil {
		lost = lostDigest(s.current.lost)
	}
	return f.try(Change{Kind: ChangeUnlock, Unlock: Unlock{Installation: id, Generation: s.generation, Lost: lost}})
}

func (f world) permit(id string, actor Actor) Token {
	f.t.Helper()
	tok, err := f.r.Permit(id, actor.ID, actor)
	f.must(err)
	return tok
}

func target(key string) Target { return Target{Workspace: "ws", Key: key} }

func numbered(key string, n int64) Target {
	return Target{Workspace: "ws", Key: key, Series: "inv", Number: n}
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

func itoa(n int) string { return strconv.Itoa(n) }
