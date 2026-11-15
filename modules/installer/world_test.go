package installer

import (
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
	limits = Limits{Keys: 2, KeyLife: 3650 * 24 * time.Hour, Releases: 6, Sites: 4, Destinations: 3, Reports: 3, Runs: 3,
		ObservationLife: 10 * time.Minute, BootstrapLife: time.Hour, Proposals: 8}
	ann   = Actor{ID: "ann", Kind: KindHuman}
	bob   = Actor{ID: "bob", Kind: KindHuman}
	cat   = Actor{ID: "cat", Kind: KindHuman}
	bot   = Actor{ID: "bot", Kind: KindAssistant}
	p1    = Actor{ID: "probe-1", Kind: KindSystem}
	p2    = Actor{ID: "probe-2", Kind: KindSystem}
	linux = Platform{OS: "linux", Arch: "amd64"}
	boom  = errors.New("synthetic outage")
)

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
	random  *entropy
	clockMu *sync.Mutex
}

func private(name string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte("synthetic-release-seed-" + name))
	return ed25519.NewKeyFromSeed(seed[:])
}

func public(name string) []byte { return private(name).Public().(ed25519.PublicKey) }

func sum(text string) string {
	s := sha256.Sum256([]byte(text))
	return hex.EncodeToString(s[:])
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	e := &entropy{}
	r, err := NewRegistry(Config{Random: e}, l, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now, random: e, clockMu: mu}
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

func keyChange(version string) Change {
	return Change{Kind: ChangeKey, Key: ReleaseKey{Version: version, Public: public(version), NotBefore: start, NotAfter: start.Add(365 * 24 * time.Hour),
		Proof: ed25519.Sign(private(version), challenge(version))}}
}

func fixture(version string) Release {
	return Release{Version: version, Key: "k1", Platforms: []Platform{{OS: "freebsd", Arch: "amd64"}, linux, {OS: "windows", Arch: "amd64"}},
		Artifacts: []Artifact{{Name: "api", Kind: ArtifactContainer, Digest: sum("api" + version)}, {Name: "worker", Kind: ArtifactArchive, Digest: sum("worker" + version)}},
		Services: []Service{
			{Name: "api", Artifact: "api", Ports: []Port{{Number: 8443, Protocol: ProtocolTCP}}, Volumes: []string{"config", "ledger"}},
			{Name: "worker", Artifact: "worker", Ports: []Port{{Number: 9000, Protocol: ProtocolTCP}, {Number: 9000, Protocol: ProtocolUDP}}, Volumes: []string{"ledger"}},
		},
		Volumes: []Volume{{Name: "config"}, {Name: "ledger", Data: true}}}
}

func signed(rel Release) Change {
	return Change{Kind: ChangeRelease, Release: rel, Signature: ed25519.Sign(private(rel.Key), rel.encode())}
}

func (f world) site(id, probe string) {
	f.t.Helper()
	f.approve(Change{Kind: ChangeSite, Site: Site{ID: id, Tenant: "tenant-" + id, Platform: linux, Probe: probe}})
}

func standard(t testing.TB) world {
	f := setup(t)
	f.approve(keyChange("k1"))
	f.approve(signed(fixture("1.0.0")))
	f.site("s1", "probe-1")
	f.site("s2", "probe-2")
	return f
}

func empty() Observation {
	return Observation{Platform: linux, Complete: true}
}

func (f world) observe(site string, probe Actor, o Observation) {
	f.t.Helper()
	f.must(second(f.r.Observe(site, o, probe)))
}

func (f world) preview(site, version string) PreviewView {
	f.t.Helper()
	v, err := f.r.Preview(site, version)
	f.must(err)
	return v
}

func (f world) accept(site, version string) ProposalView {
	f.t.Helper()
	v := f.preview(site, version)
	return f.approve(Change{Kind: ChangePlan, Acceptance: Acceptance{Site: site, Version: version, Hash: v.Hash, Admin: "cat"}})
}

func (f world) started(site string, probe Actor) RunView {
	f.t.Helper()
	f.observe(site, probe, empty())
	p := f.accept(site, "1.0.0")
	run, err := f.r.StartRun(site, p.Number, ann)
	f.must(err)
	return run
}

func (f world) finish(site, id string, probe Actor) RunView {
	f.t.Helper()
	run, err := f.r.Run(site, id)
	f.must(err)
	rel := f.r.releases[run.Version].release
	digests := map[string]string{}
	for _, a := range rel.Artifacts {
		digests["artifact:"+a.Name] = a.Digest
	}
	for _, step := range run.Steps[run.Done : len(run.Steps)-1] {
		d := digests[step]
		if d == "" {
			d = sum(step)
		}
		run, err = f.r.Complete(site, id, step, Evidence{Digest: d}, probe)
		f.must(err)
	}
	return run
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
