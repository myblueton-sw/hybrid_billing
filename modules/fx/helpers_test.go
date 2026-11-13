package fx

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "fx-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Keys: 200, Revisions: 3, Policies: 12, Versions: 4, Needs: 8, Snapshots: 20, Lines: 8, Digits: 60, Page: 3}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

var start = time.Date(2026, 11, 13, 3, 0, 0, 0, time.UTC)

func open(t testing.TB) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

func kind(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return fmt.Sprintf("other:%v", err)
}

func key(base, quote, date string) Key {
	return Key{Source: "bank", Series: "deal", Base: base, Quote: quote, Date: date}
}

func feed(t testing.TB, r *Registry, k Key, revision, unit int, value string) {
	t.Helper()
	if _, err := r.RecordObservation("ws", ObservationInput{Key: k, Revision: revision, Unit: unit, Value: value, Origin: OriginFeed, Recorder: "loader"}); err != nil {
		t.Fatalf("feed %v %d: %v", k, revision, err)
	}
}

func manual(t testing.TB, r *Registry, k Key, revision, unit int, value string) {
	t.Helper()
	if _, err := r.RecordObservation("ws", ObservationInput{Key: k, Revision: revision, Unit: unit, Value: value, Origin: OriginManual, Evidence: "doc-1", Recorder: "clerk"}); err != nil {
		t.Fatalf("manual %v %d: %v", k, revision, err)
	}
}

func pricing(v VersionInput) VersionInput {
	v.Spread, v.Fee, v.QuoteScale, v.QuoteMode, v.Tolerance, v.Quantum, v.AmountMode, v.Order = "0", "0", 2, "half_even", "0.0001", "1", "half_even", Order
	return v
}

func dated(version int, effective, construction string) VersionInput {
	v := VersionInput{Version: version, Effective: effective, Author: "author", Method: MethodDated, RateSource: "bank", Series: "deal", Construction: construction, Rule: RuleHold}
	if construction == ConstructionCross {
		v.Pivot = "USD"
	}
	return pricing(v)
}

func attrs(level, keyed, source, target string) PolicyAttributes {
	return PolicyAttributes{Level: level, Key: keyed, Purpose: "billing", Source: source, Target: target}
}

func policyVersion(t testing.TB, r *Registry, id string, a PolicyAttributes, v VersionInput) {
	t.Helper()
	if _, err := r.RecordPolicyVersion("ws", id, a, v); err != nil {
		t.Fatalf("policy %s v%d: %v", id, v.Version, err)
	}
}

func need(id, source, target, date string) Need {
	return Need{ID: id, Purpose: "billing", Source: source, Target: target, Customer: "cust-1", Contract: "con-1", Date: date}
}

func prepare(t testing.TB, r *Registry, id string, needs ...Need) Snapshot {
	t.Helper()
	snap, err := r.Prepare("ws", id, needs)
	if err != nil {
		t.Fatalf("prepare %s: %v", id, err)
	}
	return snap
}

func line(id, needID, amount, currency, state string) Line {
	return Line{ID: id, Need: needID, Amount: amount, Currency: currency, State: state}
}

func convert(t testing.TB, r *Registry, id string, lines ...Line) Result {
	t.Helper()
	out, err := r.Convert("ws", id, lines)
	if err != nil {
		t.Fatalf("convert %s: %v", id, err)
	}
	return out
}

func shown(out Result) string {
	parts := []string{}
	for _, l := range out.Lines {
		switch l.Kind {
		case ResultConverted:
			parts = append(parts, fmt.Sprintf("%s=%s %s fee %s total %s quoted %s", l.Line, l.Amount, l.Currency, l.Fee, l.Total, l.AtQuoted))
		case ResultHeld:
			parts = append(parts, l.Line+":held:"+l.Reason)
		default:
			parts = append(parts, l.Line+":"+l.Kind+":"+l.Amount+" "+l.Currency)
		}
	}
	return strings.Join(parts, "; ")
}

func entryOf(snap Snapshot) string {
	parts := []string{}
	for _, e := range snap.Entries {
		if e.Kind == EntryRate {
			parts = append(parts, fmt.Sprintf("%s:%s/%s/%s quote %s rate %s", e.Need.ID, e.Reason, e.Level, e.Policy, e.SalesQuote, e.SalesRate))
		} else {
			parts = append(parts, e.Need.ID+":"+e.Kind+":"+e.Reason)
		}
	}
	return strings.Join(parts, "; ")
}

func readSource(file string) (string, error) {
	body, err := os.ReadFile(file)
	return string(body), err
}
