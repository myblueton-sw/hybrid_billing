package perfgate

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
	bot    = Actor{ID: "bot", Kind: KindAssistant}
	runner = Actor{ID: "runner-1", Kind: KindSystem}
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

func axes(gate string) Axes {
	a := Axes{Customers: 10000, Accounts: 50000, SharedPools: 20, SkewPermille: 900, FanOut: 8, ConcurrentCloses: 4, PeakWindowMinutes: 60, RetentionMonths: 13}
	switch gate {
	case GateE2E:
		a.RowsRun = 100000
	case GateDailyInflow:
		a.RowsDay = 1000001
	case GateMonthlyClose:
		a.RowsMonth = 30000000
	case GateInterruptResume:
		a.RowsRun, a.FailureInjections = 100000, 1
	case GateCatchUp:
		a.RowsDay, a.BacklogRows = 1000001, 1000000
	}
	return a
}

func prof(id, gate string) Profile {
	return Profile{ID: id, Gate: gate, Axes: axes(gate), Dataset: sum("data-" + id), Expected: sum("answer-" + id)}
}

func hardware(id string) Hardware {
	return Hardware{ID: id, CPUs: 16, MemoryMiB: 65536, DiskGiB: 1000, Storage: StorageSSD, Spec: sum("spec-" + id)}
}

func limits(gate string) []Limit {
	out := []Limit{{MetricCompletion, 3600000}, {MetricMaxRSS, 8 << 30}, {MetricP95, 500}, {MetricP99, 1000}}
	if gate == GateInterruptResume {
		out = append(out, Limit{MetricResume, 60000})
	}
	return append(out, Limit{MetricThroughput, 12})
}

func measures(gate string) []Measure {
	out := []Measure{{MetricCompletion, 3000000}, {MetricDBIO, 1 << 30}, {MetricLockWait, 10}, {MetricMaxRSS, 4 << 30}, {MetricP95, 400}, {MetricP99, 900}}
	if gate == GateInterruptResume {
		out = append(out, Measure{MetricResume, 30000})
	}
	return append(out, Measure{MetricThroughput, 20}, Measure{MetricWAL, 1 << 28})
}

func runOf(p Profile, build string) Run {
	x := Run{Profile: p.ID, Hardware: "hw1", Build: build, Dataset: p.Dataset, Output: p.Expected, DurationS: 3600, Metrics: measures(p.Gate)}
	a := p.Axes
	switch p.Gate {
	case GateE2E, GateInterruptResume:
		x.Rows = a.RowsRun
		x.Interrupted = p.Gate == GateInterruptResume
	case GateMonthlyClose:
		x.Rows = a.RowsMonth
	case GateDailyInflow:
		x.Rows, x.DurationS = a.RowsDay, 86400
	case GateCatchUp:
		x.Rows, x.DurationS = 2*a.RowsDay, 86400
	}
	return x
}

func standard(t testing.TB) world {
	f := setup(t)
	f.approve(Change{Kind: ChangeHardware, Hardware: hardware("hw1")})
	f.approve(Change{Kind: ChangeHardware, Hardware: hardware("hw2")})
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: runner.ID}}, cat, dan)
	return f
}

func (f world) gate(id, gate string, approved bool) Profile {
	f.t.Helper()
	p := prof(id, gate)
	f.approve(Change{Kind: ChangeProfile, Profile: p})
	f.approve(Change{Kind: ChangeBudget, Budget: Budget{ID: "b-" + id, Profile: id, Hardware: "hw1", Limits: limits(gate)}})
	if approved {
		f.approveBy(Change{Kind: ChangeApproval, Approval: Approval{Budget: "b-" + id, Distribution: sum("dist-" + id)}}, ann, eve)
	}
	return p
}

func (f world) record(x Run) RunView {
	f.t.Helper()
	v, err := f.r.Record(x, runner)
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
