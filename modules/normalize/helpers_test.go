package normalize

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
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "7", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 6, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Targets: 20, Fields: 12, Mappings: 20, Revisions: 20, Runs: 200, Chunks: 200, ChunkRows: 1000, Rows: 1000000, Cells: 16, Text: 64, Findings: 1000, Unused: 3, Page: 100}
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

func testTarget() Target {
	return Target{ID: "t1", CostCurrency: "cur", PeriodStart: "start", PeriodEnd: "end", Fields: []Field{
		{Name: "billed", Kind: Cost, Cost: Billed, Format: true, Billing: true},
		{Name: "effective", Kind: Cost, Cost: Effective},
		{Name: "list", Kind: Cost, Cost: List},
		{Name: "qty", Kind: Quantity, Format: true},
		{Name: "cur", Kind: Currency, Format: true, Billing: true},
		{Name: "start", Kind: Time},
		{Name: "end", Kind: Time},
		{Name: "svc", Kind: Text, Format: true},
		{Name: "note", Kind: Text},
	}}
}

var mapping = MappingKey{Provider: "aws", SourceVersion: "focus-1.0", Target: "t1"}

func from(field, source string, kind Kind, cost CostKind) Entry {
	return Entry{Field: field, Source: source, SourceKind: kind, SourceCost: cost, Transform: Copy}
}

func missing(field string, reason MissingReason) Entry {
	return Entry{Field: field, Missing: reason}
}

func testEntries() []Entry {
	return []Entry{
		from("billed", "BilledCost", Cost, Billed),
		from("effective", "EffectiveCost", Cost, Effective),
		missing("list", NotSupported),
		from("qty", "Usage", Quantity, ""),
		from("cur", "BillingCurrency", Currency, ""),
		from("start", "PeriodStart", Time, ""),
		from("end", "PeriodEnd", Time, ""),
		from("svc", "ServiceName", Text, ""),
		missing("note", NotApplicable),
	}
}

func fullEntries() []Entry {
	entries := testEntries()
	entries[2] = from("list", "ListCost", Cost, List)
	return entries
}

func (h *harness) declare(entries []Entry) uint64 {
	h.t.Helper()
	if h.r.targets["t1"] == nil {
		if err := h.r.DeclareTarget(testTarget()); err != nil {
			h.t.Fatal(err)
		}
	}
	latest := uint64(len(h.r.mappings[mapping]))
	number, err := h.r.DeclareMapping(mapping, latest, entries)
	if err != nil {
		h.t.Fatal(err)
	}
	return number
}

var window = &Window{From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}

func spec(id string, chunks, rows int) RunSpec {
	return RunSpec{WorkspaceID: "ws1", ID: id, Mapping: mapping, Revision: 1, Input: Input{Scope: "payer-1", Period: "2026-10", Manifest: 1}, Chunks: chunks, Rows: rows, Window: window}
}

func ev(key string) RowKey { return RowKey{Kind: EventKeys, Key: key} }

const null = "<null>"

func cells(pairs ...string) map[string]Cell {
	out := map[string]Cell{}
	for index := 0; index+1 < len(pairs); index += 2 {
		if pairs[index+1] == null {
			out[pairs[index]] = Cell{Null: true}
		} else {
			out[pairs[index]] = Cell{Text: pairs[index+1]}
		}
	}
	return out
}

func good(key, billed, currency string, more ...string) Row {
	made := cells("BilledCost", billed, "EffectiveCost", "1", "ListCost", "1", "Usage", "3", "BillingCurrency", currency, "PeriodStart", "2026-10-05T00:00:00Z", "PeriodEnd", "2026-10-06T00:00:00Z", "ServiceName", "compute")
	for name, cell := range cells(more...) {
		made[name] = cell
	}
	return Row{Key: ev(key), Cells: made}
}

func without(row Row, names ...string) Row {
	made := map[string]Cell{}
	for name, cell := range row.Cells {
		made[name] = cell
	}
	for _, name := range names {
		delete(made, name)
	}
	return Row{Key: row.Key, Cells: made}
}

func (h *harness) open(s RunSpec) {
	h.t.Helper()
	if state, err := h.r.Open(s); err != nil || state != Open {
		h.t.Fatalf("open %s: %v %v", s.ID, state, err)
	}
}

func (h *harness) check(id string, chunk int, rows ...Row) {
	h.t.Helper()
	if state, err := h.r.Check("ws1", id, chunk, rows); err != nil || state != Open {
		h.t.Fatalf("check %s chunk %d: %v %v", id, chunk, state, err)
	}
}

func (h *harness) close(id string) RunView {
	h.t.Helper()
	if state, reasons, err := h.r.Close("ws1", id); err != nil || state != Closed {
		h.t.Fatalf("close %s: %v %v %v", id, state, reasons, err)
	}
	return h.view(id)
}

func (h *harness) view(id string) RunView {
	h.t.Helper()
	view, err := h.r.Run("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func (h *harness) one(entries []Entry, change func(*RunSpec), rows ...Row) RunView {
	h.t.Helper()
	h.declare(entries)
	id := fmt.Sprintf("run-%d", len(h.r.workspaces["ws1"].safeRuns())+1)
	s := spec(id, 1, len(rows))
	s.Revision = uint64(len(h.r.mappings[mapping]))
	if change != nil {
		change(&s)
	}
	h.open(s)
	h.check(id, 1, rows...)
	return h.close(id)
}

func (w *workspace) safeRuns() map[string]*run {
	if w == nil {
		return nil
	}
	return w.runs
}

func (h *harness) findings(id string) string {
	h.t.Helper()
	var parts []string
	for {
		page, err := h.r.Findings("ws1", id, len(parts), h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		if len(page) == 0 {
			return strings.Join(parts, " ")
		}
		for _, finding := range page {
			parts = append(parts, fmt.Sprintf("%d.%d/%s/%s:%s", finding.Chunk, finding.Position, finding.Key.Key, finding.Field, finding.Code))
		}
	}
}

func (h *harness) rows(id string) []OutRow {
	h.t.Helper()
	var out []OutRow
	for {
		page, err := h.r.Rows("ws1", id, len(out), h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		if len(page.Rows) == 0 {
			return out
		}
		out = append(out, page.Rows...)
	}
}

func verdicts(view RunView) string {
	if view.Format == nil || view.Billing == nil {
		return "none"
	}
	return fmt.Sprintf("%s%v %s%v", view.Format.Result, view.Format.Reasons, view.Billing.Result, view.Billing.Reasons)
}

func counts(view RunView) string {
	return fmt.Sprintf("%d=%d+%d+%d", view.Checked, view.Normal, view.Errors, view.Quarantined)
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var lines []string
	for id, held := range r.targets {
		lines = append(lines, "target "+id+" "+held.print)
	}
	for key, held := range r.mappings {
		for _, revision := range held {
			lines = append(lines, fmt.Sprintf("mapping %v %d %s %s", key, revision.number, revision.print, revision.at.Format(time.RFC3339Nano)))
		}
	}
	for id, ws := range r.workspaces {
		lines = append(lines, fmt.Sprintf("ws %s rows=%d", id, ws.rows))
		for runID, held := range ws.runs {
			var chunks []string
			for number, sum := range held.chunks {
				chunks = append(chunks, fmt.Sprintf("%05d:%s", number, sum))
			}
			sort.Strings(chunks)
			lines = append(lines, fmt.Sprintf("run %s/%s %s %s %v %d %v held=%d unused=%d %v %v %v rows=%d findings=%d/%d apart=%d %v %s %s %d %s %s", id, runID, held.print, held.state, chunks, held.checked, held.counts, held.held, len(held.unused),
				held.sums, held.format, held.billing, len(held.rows), len(held.findings), held.findingsTotal, len(held.apart), held.names, held.abandon, held.discard, held.contradiction, held.opened.Format(time.RFC3339Nano), held.changed.Format(time.RFC3339Nano)))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
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
