package reconcile

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var start = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

type testClock struct{ at time.Time }

func (c *testClock) read() time.Time { return c.at }

func (c *testClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 6, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits { return Limits{Runs: 100, Parts: 1000, Lines: 100, Page: 5, Listed: 3} }

var testRounding = map[string]string{"USD": "0.05", "KRW": "1"}

type harness struct {
	t     testing.TB
	r     *Registry
	clock *testClock
}

func newHarness(t testing.TB, change func(*Limits)) *harness {
	t.Helper()
	limits := testLimits()
	if change != nil {
		change(&limits)
	}
	clock := &testClock{at: start}
	r, err := NewRegistry(limits, testRounding, testProfile(t), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func doc(id string, revision int, currency, total string, parts int) RunSpec {
	return RunSpec{WorkspaceID: "ws1", ID: id, Basis: ProviderDocument, Issuer: "aws", Account: "acct-1", Document: "inv-100", Revision: revision, Currency: currency, Total: total, Input: "manifest:7", Parts: parts, Rounding: "0.05"}
}

func priced(id string) RunSpec {
	return RunSpec{WorkspaceID: "ws1", ID: id, Basis: IndependentPrice, Customer: "cust-1", Period: "2026-10", Charge: "support-plan", QuantityComplete: true, QuantityEvidence: "usage:1", PriceEvidence: "pricebook:3"}
}

func pooled(id string) RunSpec {
	return RunSpec{WorkspaceID: "ws1", ID: id, Basis: InternalCost, Customer: "cust-1", Period: "2026-10", Pool: "dc-seoul", Cost: "1200.50", Currency: "KRW", PoolEvidence: "ledger:9", AllocationEvidence: "alloc:2"}
}

func bill(id, amount string) Part {
	return Part{ID: id, Amount: amount, Currency: "USD", Kind: Billed, Source: "run:" + id}
}

func explain(id string, category Category, amount, item string) Line {
	made := Line{ID: id, Category: category, Amount: amount, Currency: "USD", Evidence: "doc:" + id}
	if category != Rounding {
		made.ItemType, made.Item = DocumentLine, item
	}
	return made
}

func (h *harness) open(spec RunSpec) {
	h.t.Helper()
	want := Open
	if spec.Basis != ProviderDocument {
		want = Closed
	}
	if state, err := h.r.Open(spec); err != nil || state != want {
		h.t.Fatalf("open %s: %v %v", spec.ID, state, err)
	}
}

func (h *harness) post(id string, parts ...Part) {
	h.t.Helper()
	for _, part := range parts {
		if state, err := h.r.Post("ws1", id, part); err != nil || state != Open {
			h.t.Fatalf("post %s/%s: %v %v", id, part.ID, state, err)
		}
	}
}

func (h *harness) explain(id string, lines ...Line) {
	h.t.Helper()
	for _, line := range lines {
		if state, err := h.r.Explain("ws1", id, line); err != nil || state != Open {
			h.t.Fatalf("explain %s/%s: %v %v", id, line.ID, state, err)
		}
	}
}

func (h *harness) close(id string) string {
	h.t.Helper()
	state, verdict, err := h.r.Close("ws1", id)
	if err != nil || state != Closed {
		h.t.Fatalf("close %s: %v %v %v", id, state, verdict, err)
	}
	return verdict
}

func (h *harness) view(id string) RunView {
	h.t.Helper()
	view, err := h.r.Run("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func (h *harness) ready(id string) string {
	h.t.Helper()
	out, err := h.r.Readiness("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return fmt.Sprintf("%s%v %s", out.Verdict, out.Reasons, out.Cost)
}

func sums(view RunView) string {
	var parts []string
	for _, sum := range view.Categories {
		parts = append(parts, fmt.Sprintf("%s:%d:%s", sum.Category, sum.Lines, sum.Sum))
	}
	return fmt.Sprintf("internal=%s explained=%s residual=%s %s [%s]", view.Internal, view.Explained, view.Residual, view.Verdict, strings.Join(parts, " "))
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for id, ws := range r.workspaces {
		for name, held := range ws.runs {
			var parts, lines []string
			for _, made := range held.parts {
				parts = append(parts, made.print)
			}
			for _, made := range held.lines {
				lines = append(lines, made.print)
			}
			sort.Strings(parts)
			sort.Strings(lines)
			residual := ""
			if held.residual != nil {
				residual = held.internal.text() + "/" + held.explained.text() + "/" + held.residual.text()
			}
			out = append(out, fmt.Sprintf("run %s/%s %s %s %v %v %v %v %s %s %s %s %s", id, name, held.print, held.state, held.wasClosed, parts, lines, held.rounded, residual, held.verdict, held.reason,
				held.opened.Format(time.RFC3339Nano), held.changed.Format(time.RFC3339Nano)))
		}
		for key, held := range ws.subjects {
			out = append(out, fmt.Sprintf("subject %s/%s %s", id, key, held.spec.ID))
		}
		for key, revision := range ws.highest {
			out = append(out, fmt.Sprintf("highest %s/%s %d %d", id, key, revision, len(ws.byDoc[key])))
		}
		for key, held := range ws.items {
			out = append(out, fmt.Sprintf("item %s/%s %s", id, key, held.spec.ID))
		}
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}

func (h *harness) unchanged(before, what string) {
	h.t.Helper()
	if after := h.r.dump(); after != before {
		h.t.Fatalf("%s wrote:\n%s\n---\n%s", what, before, after)
	}
}

func wantErr(t testing.TB, err, want error, what string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("%s: got %v, want %v", what, err, want)
	}
}
