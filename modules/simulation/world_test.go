package simulation

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
	start  = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	ann    = Actor{ID: "ann", Kind: KindHuman}
	bob    = Actor{ID: "bob", Kind: KindHuman}
	cat    = Actor{ID: "cat", Kind: KindHuman}
	dan    = Actor{ID: "dan", Kind: KindHuman}
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	runner = Actor{ID: "runner-1", Kind: KindSystem}
	other  = Actor{ID: "runner-2", Kind: KindSystem}
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

func (f world) advance(d time.Duration) {
	f.mu.Lock()
	*f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) approveBy(c Change, proposer, approver Actor) ProposalView {
	f.t.Helper()
	p, err := f.r.Propose(c, proposer)
	f.must(err)
	v, err := f.r.Approve(p.Number, approver)
	f.must(err)
	return v
}

func (f world) approve(c Change) ProposalView { return f.approveBy(c, ann, bob) }

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

func public(id string) Dataset {
	return Dataset{ID: id, Version: "1.0.0", Origin: OriginPublic, Digest: sum("bytes-" + id), Source: "https://example.org/focus/sample.csv", License: "CC-BY-4.0",
		Notice: sum("notice"), Coverage: []string{"aws", "focus-1.2"}}
}

func converted(id, parent string) Dataset {
	return Dataset{ID: id, Version: "1.0.0", Origin: OriginConverted, Digest: sum("bytes-" + id), License: "CC-BY-4.0", Notice: sum("notice"), Parent: parent,
		Tool: "focus-converter", ToolVersion: "0.4.1", Coverage: []string{"aws"}}
}

func synthetic(id string) Dataset {
	return Dataset{ID: id, Version: "1.0.0", Origin: OriginSynthetic, Digest: sum("bytes-" + id), Tool: "generator", ToolVersion: "2.1.0", Seed: "42",
		Coverage: []string{"commitments"}}
}

func billing(id string) Scenario {
	return Scenario{ID: id, FixtureVersion: "1.0.0", Purpose: PurposeBilling, Topology: TopologyMSP, Deployment: DeploymentMixed, Seed: "7", Start: start,
		Zone: "Asia/Seoul", Relations: "rel-1", Policy: "pol-1", FX: "fx-1", Datasets: []string{"syn"}, Events: sum("events-" + id), EventCount: 100,
		Profile: "p1", Covers: []string{EventOverage, EventPartialCSVError, EventRefund}}
}

func format(id string) Scenario {
	return Scenario{ID: id, FixtureVersion: "1.0.0", Purpose: PurposeFormat, Topology: TopologySingle, Deployment: DeploymentCloud, Seed: "1", Start: start,
		Zone: "UTC", Relations: "rel-1", Policy: "pol-1", FX: "fx-1", Datasets: []string{"pub"}, Events: sum("events-" + id), EventCount: 3}
}

func amount(name, value, currency string) Check {
	return Check{Name: name, Kind: CheckAmount, Value: value, Currency: currency}
}

func expectation(scenario string, version int, checks ...Check) Expectation {
	return Expectation{Scenario: scenario, Version: version, Method: MethodManual, Evidence: sum("sheet-" + scenario + strconv.Itoa(version)), Checks: checks}
}

func standardChecks() []Check {
	return []Check{amount("krw-total", "1000.5", "KRW"), {Name: "lines", Kind: CheckCount, Value: "12"}, amount("usd-total", "-3.25", "USD")}
}

func results(checks []Check) []Result {
	out := []Result{}
	for _, c := range checks {
		out = append(out, Result{Name: c.Name, Value: c.Value, Currency: c.Currency})
	}
	return out
}

func standard(t testing.TB) world {
	f := setup(t)
	for _, d := range []Dataset{public("pub"), converted("conv", "pub"), synthetic("syn")} {
		f.approve(Change{Kind: ChangeDataset, Dataset: d})
	}
	f.approve(Change{Kind: ChangeProfile, Profile: Profile{ID: "p1"}})
	f.approve(Change{Kind: ChangeScenario, Scenario: billing("s1")})
	f.approve(Change{Kind: ChangeScenario, Scenario: format("f1")})
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: runner.ID}}, cat, dan)
	return f
}

func (f world) expect(scenario string, checks ...Check) {
	f.t.Helper()
	version := len(f.r.scenarios[scenario].expectations) + 1
	f.approve(Change{Kind: ChangeExpectation, Expectation: expectation(scenario, version, checks...)})
}

func (f world) record(scenario, engine, rules, output string, list []Result) RunView {
	f.t.Helper()
	v, err := f.r.Record(Run{Scenario: scenario, Engine: engine, Rules: sum(rules), Output: sum(output), Results: list}, runner)
	f.must(err)
	return v
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
