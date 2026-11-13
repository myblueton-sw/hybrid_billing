package hawkeye

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
	return Limits{Receptions: 200, Lines: 3000, Currencies: 3, Page: 5, Listed: 3, MaxPeriod: 40 * day}
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
	r, err := NewRegistry(limits, 9500, testProfile(t), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func month(number int) (time.Time, time.Time) {
	return time.Date(2026, time.Month(number), 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.Month(number+1), 1, 0, 0, 0, 0, time.UTC)
}

func stmt(id, statement string, monthNumber int) Spec {
	from, to := month(monthNumber)
	return Spec{WorkspaceID: "ws1", ID: id, Source: "hawk-1", Statement: statement, CostOrigin: "gpu-power", Schema: "v3", Connection: "conn-1", PeriodStart: from, PeriodEnd: to, Revision: 1,
		Status: Final, Tier: Measured, Coverage: 9800, Lines: 3, Totals: map[string]string{"KRW": "300"}, Declaration: SourceHeader}
}

func superseding(id, statement string, monthNumber int, earlier string) Spec {
	spec := stmt(id, statement, monthNumber)
	spec.Supersedes, spec.CorrectionReason, spec.Revision = earlier, "recount", 2
	return spec
}

func team(name, cost string) Line {
	return Line{Team: name, Cost: cost, Currency: "KRW", Measured: true}
}

func (h *harness) open(spec Spec) {
	h.t.Helper()
	if state, occupant, err := h.r.Open(spec); err != nil || state != Open || occupant != "" {
		h.t.Fatalf("open %s: %v %q %v", spec.ID, state, occupant, err)
	}
}

func (h *harness) post(id string, lines ...Line) {
	h.t.Helper()
	for _, line := range lines {
		if state, err := h.r.Post("ws1", id, line); err != nil || state != Open {
			h.t.Fatalf("post %s/%s: %v %v", id, line.Team, state, err)
		}
	}
}

func (h *harness) close(id string) {
	h.t.Helper()
	if state, err := h.r.Close("ws1", id); err != nil || state != Closed {
		h.t.Fatalf("close %s: %v %v", id, state, err)
	}
}

func (h *harness) full(spec Spec) {
	h.t.Helper()
	h.open(spec)
	h.post(spec.ID, team("team-a", "100"), team("team-b", "150.5"), team("team-c", "49.50"))
	h.close(spec.ID)
}

func (h *harness) verdict(id string) string {
	h.t.Helper()
	out, err := h.r.Eligibility("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	text := fmt.Sprintf("%s%v", out.Result, out.Reasons)
	if out.Conflicts.Total > 0 {
		text += fmt.Sprintf(" with %v of %d", out.Conflicts.Items, out.Conflicts.Total)
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
	out, err := h.r.Counting("ws1", "hawk-1", from, to)
	if err != nil && err != ErrNotFound {
		h.t.Fatal(err)
	}
	return out.Reception
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for id, ws := range r.workspaces {
		for name, held := range ws.receptions {
			var lines []string
			for _, line := range held.lines {
				lines = append(lines, line.print)
			}
			sort.Strings(lines)
			sums := map[string]string{}
			for currency, sum := range held.sums {
				sums[currency] = sum.text()
			}
			out = append(out, fmt.Sprintf("reception %s/%s %s %s/%s %s %v %v %v %v %s %s %s", id, name, held.print, held.spec.Schema, held.spec.Connection, held.state, held.wasClosed, lines, sums, held.unmeasured, held.reason,
				held.opened.Format(time.RFC3339Nano), held.changed.Format(time.RFC3339Nano)))
		}
		for name, origin := range ws.sources {
			out = append(out, fmt.Sprintf("source %s/%s sequence=%d receptions=%d", id, name, origin.sequence, len(origin.receptions)))
			for statementID, known := range origin.statements {
				live := ""
				if known.live != nil {
					live = known.live.spec.ID
				}
				out = append(out, fmt.Sprintf("statement %s/%s/%s supersedes=%s by=%s live=%s", id, name, statementID, known.supersedes, known.supersededBy, live))
			}
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
