package tokenmeter

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

const day = 24 * time.Hour

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

func testLimits() Limits {
	return Limits{Receptions: 200, Events: 200000, Batch: 1000, Currencies: 3, Documents: 4, Page: 5, Listed: 3, MaxPeriod: 40 * day}
}

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
	r, err := NewRegistry(limits, testProfile(t), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func month(number int) (time.Time, time.Time) {
	return time.Date(2026, time.Month(number), 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.Month(number+1), 1, 0, 0, 0, 0, time.UTC)
}

func export(id, snapshot string, monthNumber int) Spec {
	from, to := month(monthNumber)
	return Spec{WorkspaceID: "ws1", ID: id, Source: "tm-1", Snapshot: snapshot, Schema: "v2", Connection: "conn-1", PeriodStart: from, PeriodEnd: to, Revision: 1, Produced: SnapshotExport, Numeric: Decimal, Unit: UnitToken,
		Events: 3, Tokens: map[Kind]string{Input: "300", Output: "30", CacheRead: "0", CacheWrite: "0"}, Costs: map[string]string{"USD": "3"}, Declaration: SourceManifest}
}

func event(id string, monthNumber int) Event {
	from, _ := month(monthNumber)
	return Event{ID: id, Request: "req-" + id, Attempt: "1", Time: from.Add(time.Hour), Consumer: "user-1", Provider: "prov-a", Model: "model-x", Version: "v1", Endpoint: "chat", Outcome: Succeeded,
		Input: "100", Output: "10", CacheRead: "0", CacheWrite: "0", Usage: ProviderReported, Cache: Separate, Cost: &Cost{Amount: "1", Currency: "USD", Grade: Invoiced, Origin: Provider, Document: "inv-1"}}
}

func priced(id string, monthNumber int, amount string, grade Grade, origin Origin) Event {
	made := event(id, monthNumber)
	made.Cost = &Cost{Amount: amount, Currency: "USD", Grade: grade, Origin: origin}
	if grade == Invoiced {
		made.Cost.Document = "inv-1"
	}
	return made
}

func (h *harness) open(spec Spec) {
	h.t.Helper()
	if state, occupant, err := h.r.Open(spec); err != nil || state != Open || occupant != "" {
		h.t.Fatalf("open %s: %v %q %v", spec.ID, state, occupant, err)
	}
}

func (h *harness) post(id string, events ...Event) {
	h.t.Helper()
	if state, err := h.r.Post("ws1", id, events); err != nil || state != Open {
		h.t.Fatalf("post %s: %v %v", id, state, err)
	}
}

func (h *harness) close(id string) *Figures {
	h.t.Helper()
	if state, err := h.r.Close("ws1", id); err != nil || state != Closed {
		h.t.Fatalf("close %s: %v %v", id, state, err)
	}
	return h.view(id).Figures
}

func (h *harness) full(spec Spec) {
	h.t.Helper()
	h.open(spec)
	number := int(spec.PeriodStart.Month())
	h.post(spec.ID, event(spec.ID+"-e1", number), event(spec.ID+"-e2", number), event(spec.ID+"-e3", number))
	h.close(spec.ID)
}

func (h *harness) one(change func(*Spec), events ...Event) *Figures {
	h.t.Helper()
	spec := export("r1", "snap-1", 10)
	spec.Events = len(events)
	if change != nil {
		change(&spec)
	}
	h.open(spec)
	if len(events) > 0 {
		h.post("r1", events...)
	}
	return h.close("r1")
}

func (h *harness) verdict(id string) string {
	h.t.Helper()
	out, err := h.r.Eligibility("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return show(out)
}

func show(out Verdict) string {
	text := fmt.Sprintf("%s%v", out.Result, out.Reasons)
	if out.Conflicts.Total > 0 {
		text += fmt.Sprintf(" with %v of %d", out.Conflicts.Items, out.Conflicts.Total)
	}
	if out.Shared.Total > 0 {
		text += fmt.Sprintf(" sharing %v of %d", out.Shared.Items, out.Shared.Total)
	}
	return text
}

func (h *harness) view(id string) View {
	h.t.Helper()
	view, err := h.r.Reception("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func (h *harness) counting(monthNumber int) string {
	h.t.Helper()
	from, to := month(monthNumber)
	out, err := h.r.Counting("ws1", "tm-1", from, to)
	if err != nil && err != ErrNotFound {
		h.t.Fatal(err)
	}
	return out.Reception
}

func (h *harness) events(id string) []EventView {
	h.t.Helper()
	var out []EventView
	for {
		page, err := h.r.Events("ws1", id, len(out), h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		if len(page) == 0 {
			return out
		}
		out = append(out, page...)
	}
}

func flat(figures *Figures) string {
	var parts []string
	for usage, byKind := range figures.Tokens {
		for kind, sum := range byKind {
			parts = append(parts, fmt.Sprintf("tokens/%s/%s=%s", usage, kind, sum))
		}
	}
	for kind, count := range figures.Unknown {
		parts = append(parts, fmt.Sprintf("unknown/%s=%d", kind, count))
	}
	for class, byCurrency := range figures.Costs {
		for currency, sum := range byCurrency {
			parts = append(parts, fmt.Sprintf("cost/%s/%s=%s", class, currency, sum))
		}
	}
	for document, entry := range figures.Documents {
		for currency, sum := range entry.Sums {
			parts = append(parts, fmt.Sprintf("doc/%s/%s=%s", document, currency, sum))
		}
		parts = append(parts, fmt.Sprintf("doc/%s=%d", document, entry.Events))
	}
	for outcome, count := range figures.Outcomes {
		parts = append(parts, fmt.Sprintf("outcome/%s=%d", outcome, count))
	}
	sort.Strings(parts)
	return fmt.Sprintf("error=%d net=%s/%d %s", figures.InError, figures.NetInput, figures.NetUnknown, strings.Join(parts, " "))
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for id, ws := range r.workspaces {
		for name, held := range ws.receptions {
			var events []string
			for _, made := range held.events {
				events = append(events, made.print+fmt.Sprint(made.findings, made.class))
			}
			sort.Strings(events)
			figures := ""
			if held.figures != nil {
				figures = flat(held.figures)
			}
			shared := 0
			for _, count := range held.shared {
				shared += count
			}
			out = append(out, fmt.Sprintf("reception %s/%s %s %s/%s %s %v %v %s %v %v shared=%d cur=%d doc=%d %s %s %s", id, name, held.print, held.spec.Schema, held.spec.Connection, held.state, held.wasClosed, events, figures,
				held.differ, held.outside, shared, len(held.currencies), len(held.documents), held.reason, held.opened.Format(time.RFC3339Nano), held.changed.Format(time.RFC3339Nano)))
		}
		for name, origin := range ws.sources {
			out = append(out, fmt.Sprintf("source %s/%s sequence=%d receptions=%d exports=%d identities=%d", id, name, origin.sequence, len(origin.receptions), len(origin.exports), len(origin.holders)))
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

func second[T any](_ T, err error) error { return err }

func third[A, B any](_ A, _ B, err error) error { return err }
