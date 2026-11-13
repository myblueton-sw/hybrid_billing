package formula

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	number  = Type{Kind: KindNumber}
	usd     = Type{Kind: KindMoney, Currency: "USD"}
	krw     = Type{Kind: KindMoney, Currency: "KRW"}
	hour    = Type{Kind: KindQuantity, Unit: "hour"}
	gb      = Type{Kind: KindQuantity, Unit: "gb"}
	usdHour = Type{Kind: KindPrice, Currency: "USD", Unit: "hour"}
	krwHour = Type{Kind: KindPrice, Currency: "KRW", Unit: "hour"}
	usdGB   = Type{Kind: KindPrice, Currency: "USD", Unit: "gb"}
)

func profileOf(t testing.TB, whole, scale, result int) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "7", InputIntegerDigits: whole, InputScale: scale, ResultIntegerDigits: result, ResultScale: scale, IntermediateDigits: 60})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testProfile(t testing.TB) decimal.Profile { return profileOf(t, 12, 6, 14) }

func testLimits() Limits {
	return Limits{Variables: 6, Formulas: 6, Nodes: 15, Depth: 4, Digits: 30}
}

func con(text string, kind Type) Node { return Node{Kind: NodeConstant, Number: text, Type: kind} }

func vr(name string) Node { return Node{Kind: NodeVariable, Name: name} }

func ref(id string) Node { return Node{Kind: NodeReference, Formula: id} }

func op(kind string, left, right Node) Node {
	return Node{Kind: kind, Operands: []Node{left, right}}
}

func decl(name string, kind Type) Variable { return Variable{Name: name, Type: kind} }

func fm(id string, output Type, root Node) Formula {
	return Formula{ID: id, Target: id, Rank: 1, Output: output, Quantum: "0.01", Mode: "half_even", Root: root}
}

func setOf(variables []Variable, formulas ...Formula) Set {
	return Set{Variables: variables, Formulas: formulas}
}

func vars(variables ...Variable) []Variable { return variables }

func codes(found []Finding) string {
	parts := make([]string, 0, len(found))
	for _, finding := range found {
		parts = append(parts, finding.Code+"@"+finding.Formula)
	}
	return strings.Join(parts, " ")
}

func check(t testing.TB, set Set) string {
	t.Helper()
	compiled, found, err := Compile(testProfile(t), testLimits(), set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if (compiled == nil) == (len(found) == 0) {
		t.Fatalf("compiled %v with findings %v", compiled != nil, found)
	}
	return codes(found)
}

func must(t testing.TB, set Set) *Compiled {
	t.Helper()
	compiled, found, err := Compile(testProfile(t), testLimits(), set)
	if err != nil || compiled == nil {
		t.Fatalf("compile: %v %v", err, found)
	}
	return compiled
}

func in(pairs ...string) []Input {
	out := []Input{}
	for index := 0; index+1 < len(pairs); index += 2 {
		out = append(out, Input{Name: pairs[index], Value: pairs[index+1]})
	}
	return out
}

func show(out Outcome) string {
	if out.Reason != "" {
		return out.Formula + ":" + out.Reason
	}
	return out.Formula + "=" + out.Value + "(" + out.Exact + ")"
}

func shown(result Result) string {
	parts := []string{show(result.Outcome)}
	for _, reached := range result.Reached {
		parts = append(parts, show(reached))
	}
	return strings.Join(parts, " ")
}

func value(t testing.TB, set Set, target string, inputs ...string) string {
	t.Helper()
	result, err := must(t, set).Evaluate(target, in(inputs...))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return shown(result)
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

func (c *testClock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

func registryLimits() RegistryLimits {
	return RegistryLimits{Limits: testLimits(), Sets: 3, Revisions: 6, History: 2, Page: 3}
}

func open(t testing.TB) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), registryLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

const digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func plain(amount string) Set {
	return setOf(vars(decl("cost", usd)), fm("price", usd, op(NodeMul, vr("cost"), con(amount, number))))
}

func full(epoch int) Impact {
	return Impact{Manifest: "m-1", ManifestDigest: digestA, Epoch: epoch, Coverage: CoverageFull, Evaluated: 10, Changed: 4}
}

func drafted(t testing.TB, r *Registry, number int, amount string) {
	t.Helper()
	if _, err := r.Draft("ws", "set", number, "author", plain(amount)); err != nil {
		t.Fatalf("draft %d: %v", number, err)
	}
}

func checked(t testing.TB, r *Registry, number int, amount string) {
	t.Helper()
	drafted(t, r, number, amount)
	if view, err := r.RunCheck("ws", "set", number); err != nil || view.State != StateChecked {
		t.Fatalf("check %d: %v %v", number, err, view.State)
	}
}

func impacted(t testing.TB, r *Registry, number int, amount string) string {
	t.Helper()
	checked(t, r, number, amount)
	active, _ := r.Active("ws", "set")
	view, err := r.RecordImpact("ws", "set", number, full(active.Epoch))
	if err != nil {
		t.Fatalf("impact %d: %v", number, err)
	}
	return view.Impact.Subject
}

func approved(t testing.TB, r *Registry, number int, amount string) {
	t.Helper()
	subject := impacted(t, r, number, amount)
	if _, err := r.RecordApproval("ws", "set", number, "approver", "ref-1", subject); err != nil {
		t.Fatalf("approval %d: %v", number, err)
	}
}

func activated(t testing.TB, r *Registry, number int, amount string) {
	t.Helper()
	approved(t, r, number, amount)
	if _, err := r.Activate("ws", "set", number); err != nil {
		t.Fatalf("activate %d: %v", number, err)
	}
}

func state(t testing.TB, r *Registry, number int) string {
	t.Helper()
	view, err := r.Revision("ws", "set", number)
	if err != nil {
		t.Fatalf("revision %d: %v", number, err)
	}
	return view.State
}
