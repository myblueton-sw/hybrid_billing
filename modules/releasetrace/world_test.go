package releasetrace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	start  = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	ann    = Actor{ID: "ann", Kind: KindHuman}
	bob    = Actor{ID: "bob", Kind: KindHuman}
	cat    = Actor{ID: "cat", Kind: KindHuman}
	dan    = Actor{ID: "dan", Kind: KindHuman}
	eve    = Actor{ID: "eve", Kind: KindHuman}
	fay    = Actor{ID: "fay", Kind: KindHuman}
	gus    = Actor{ID: "gus", Kind: KindHuman}
	hal    = Actor{ID: "hal", Kind: KindHuman}
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	runner = Actor{ID: "runner", Kind: KindSystem}
)

type world struct {
	t   testing.TB
	r   *Registry
	now *time.Time
	mu  *sync.Mutex
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
	return world{t: t, r: r, now: &now, mu: mu}
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) by(c Change, proposer, approver Actor) ProposalView {
	f.t.Helper()
	p, err := f.r.Propose(c, proposer)
	f.must(err)
	v, err := f.r.Approve(p.Number, approver)
	f.must(err)
	return v
}

func (f world) approve(c Change) ProposalView { return f.by(c, ann, bob) }

func (f world) tryBy(c Change, proposer, approver Actor) error {
	p, err := f.r.Propose(c, proposer)
	if err != nil {
		return err
	}
	_, err = f.r.Approve(p.Number, approver)
	return err
}

func (f world) try(c Change) error { return f.tryBy(c, ann, bob) }

func sum(text string) string {
	s := sha256.Sum256([]byte(text))
	return hex.EncodeToString(s[:])
}

func req(id string, common bool, tests ...TestRef) Requirement {
	return Requirement{Namespace: "v1", ID: id, Source: "ch-" + id, Owner: "pm", Interfaces: []string{"api-" + id}, Tests: tests, Common: common, Design: sum("design-" + id)}
}

func allCategories(prefix string) []TestRef {
	out := []TestRef{}
	for _, c := range Categories {
		out = append(out, TestRef{ID: prefix + "-" + c, Category: c})
	}
	return out
}

func k(id string) Key { return Key{Namespace: "v1", ID: id} }

func (f world) pass(namespace, test, build string, passed bool) {
	f.t.Helper()
	f.must(second(f.r.Record(Result{Namespace: namespace, Test: test, Build: build, Passed: passed, Evidence: sum(test + build)}, runner)))
}

func standard(t testing.TB) world {
	f := setup(t)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("R1", true, allCategories("t")...)})
	f.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c1")[:40]}})
	f.approve(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b1")[:40]}})
	f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: runner.ID}}, cat, dan)
	for _, tr := range allCategories("t") {
		f.pass("v1", tr.ID, "1.0.0", true)
	}
	f.approve(Change{Kind: ChangePath, Path: Path{ID: "p1", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceXLSX, Template: "std-xlsx",
		Requirements: []Key{k("R1")}}})
	return f
}

func (f world) reconcile(path, period string, matched bool) {
	f.t.Helper()
	d := 0
	if !matched {
		d = 1
	}
	f.by(Change{Kind: ChangeReconcile, Reconcile: Reconcile{Path: path, Period: period, Evidence: sum(path + period), Matched: matched, Differences: d}}, eve, fay)
}

func (f world) activate(path, build string) error {
	return f.tryBy(Change{Kind: ChangeActivate, Activate: Activate{Path: path, Build: build, Proof: sum("proof")}}, gus, hal)
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
