package thirdparty

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

type world struct {
	t testing.TB
	r *Registry
}

func setup(t testing.TB) world {
	t.Helper()
	r, err := NewRegistry(func() time.Time { return start })
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r}
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

func sum(text string) string {
	s := sha256.Sum256([]byte(text))
	return hex.EncodeToString(s[:])
}

func comp(id, name, version string) Component {
	return Component{ID: id, Name: name, Version: version, Commit: sum("commit-" + id)[:40], Source: "https://example.org/" + name, License: "MIT",
		LicenseText: sum("license-" + id), Notice: sum("notice-" + id), SBOM: sum("sbom-" + id), Artifact: sum("artifact-" + id), Offline: true,
		Telemetry: TelemetryNone, Modification: ModUnmodified, Distribution: DistBundled, Tests: sum("tests-" + id), Passed: true}
}

func (f world) adopted(c Component) {
	f.t.Helper()
	f.approve(Change{Kind: ChangeComponent, Component: c})
	f.approve(Change{Kind: ChangeAdopt, Decision: Decision{Component: c.ID}})
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}
