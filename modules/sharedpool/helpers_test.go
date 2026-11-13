package sharedpool

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

func profileOf(t testing.TB, whole, scale, intermediate int) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "1", InputIntegerDigits: whole, InputScale: scale, ResultIntegerDigits: whole, ResultScale: scale, IntermediateDigits: intermediate})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Pools: 3, Versions: 6, Runs: 8, Chunks: 4, ChunkWeights: 4, Targets: 8, Page: 4, MaxPeriod: 40 * day, Intermediate: 40}
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
	r, err := NewRegistry(limits, profileOf(t, 12, 4, 40), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func month(number int) (time.Time, time.Time) {
	return time.Date(2026, time.Month(number), 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.Month(number+1), 1, 0, 0, 0, 0, time.UTC)
}

func capped(version, monthNumber int, amount, capacity string) Spec {
	from, to := month(monthNumber)
	spec := Spec{WorkspaceID: "ws1", Pool: "shared-1", Version: version, Currency: "KRW", PeriodStart: from, PeriodEnd: to, Amount: amount, Quantum: "1", Capacity: capacity}
	if capacity != "" {
		spec.ResidualBearer = MSP
	}
	return spec
}

func w(target string, bearer Bearer, weight string) Weight {
	return Weight{Target: target, Bearer: bearer, Weight: weight}
}

var approved = Approval{Approver: "approver-1", Reference: "approval-doc-1"}

func (h *harness) pool(spec Spec) {
	h.t.Helper()
	if err := h.r.OpenPool(spec); err != nil {
		h.t.Fatalf("pool %d: %v", spec.Version, err)
	}
}

func (h *harness) run(version int, id string, purpose Purpose, chunks int) {
	h.t.Helper()
	if state, err := h.r.StartRun("ws1", "shared-1", version, RunSpec{ID: id, Purpose: purpose, Chunks: chunks}); err != nil || state != Staging {
		h.t.Fatalf("run %s: %v %v", id, state, err)
	}
}

func (h *harness) stage(version int, id string, position int, weights ...Weight) {
	h.t.Helper()
	if state, err := h.r.StageChunk("ws1", "shared-1", version, id, position, weights); err != nil || state != Staging {
		h.t.Fatalf("chunk %d of %s: %v %v", position, id, state, err)
	}
}

func (h *harness) seal(version int, id string) State {
	h.t.Helper()
	state, err := h.r.Seal("ws1", "shared-1", version, id)
	if err != nil {
		h.t.Fatalf("seal %s: %v", id, err)
	}
	return state
}

func (h *harness) publish(version int, id string) {
	h.t.Helper()
	view := h.view(version, id)
	approval := Approval{}
	if view.Purpose == Chargeback {
		approval = approved
	}
	if state, err := h.r.Publish("ws1", "shared-1", version, id, approval); err != nil || state != Published {
		h.t.Fatalf("publish %s: %v %v", id, state, err)
	}
}

func (h *harness) whole(version int, id string, purpose Purpose, weights ...Weight) {
	h.t.Helper()
	h.run(version, id, purpose, 1)
	h.stage(version, id, 1, weights...)
	if state := h.seal(version, id); state != Sealed {
		h.t.Fatalf("%s is %s", id, state)
	}
	h.publish(version, id)
}

func (h *harness) view(version int, id string) RunView {
	h.t.Helper()
	view, err := h.r.Run("ws1", "shared-1", version, id)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func shareText(shares []Share) string {
	var parts []string
	for _, share := range shares {
		mark := ""
		if share.Residual {
			mark = "/residual"
		}
		parts = append(parts, fmt.Sprintf("%s=%s(%s/%s%s)", share.Target, share.Amount, share.Bearer, share.Weight, mark))
	}
	return strings.Join(parts, " ")
}

func (h *harness) shares(version int, id string) string {
	h.t.Helper()
	var all []Share
	after := ""
	for {
		page, err := h.r.Shares("ws1", "shared-1", version, id, after, h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		all = append(all, page.Items...)
		if !page.More {
			return shareText(all)
		}
		after = page.Items[len(page.Items)-1].Target
	}
}

func sums(view RunView) string {
	var parts []string
	for class, sum := range view.ByBearer {
		parts = append(parts, fmt.Sprintf("%s=%s", class, sum))
	}
	sort.Strings(parts)
	text := fmt.Sprintf("%s amount=%s denominator=%s", view.State, view.Amount, view.Denominator)
	if view.Residual != "" {
		text += fmt.Sprintf(" residual=%s(%s)", view.Residual, view.ResidualReason)
	}
	return text + " " + strings.Join(parts, " ")
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for ws, pools := range r.workspaces {
		for name, home := range pools {
			var charged []string
			for _, held := range home.charged {
				charged = append(charged, fmt.Sprintf("%d/%s", held.home.spec.Version, held.spec.ID))
			}
			sort.Strings(charged)
			out = append(out, fmt.Sprintf("pool %s/%s sequence=%d versions=%d charged=%v", ws, name, home.sequence, len(home.versions), charged))
			for number, held := range home.versions {
				out = append(out, fmt.Sprintf("version %s/%s/%06d %s runs=%d", ws, name, number, held.print, len(held.runs)))
				for id, made := range held.runs {
					var chunks []string
					for position, part := range made.chunks {
						chunks = append(chunks, fmt.Sprintf("%d:%s", position, part.print))
					}
					sort.Strings(chunks)
					alternative := ""
					if made.alternative != nil {
						alternative = fmt.Sprintf("%+v", *made.alternative)
					}
					out = append(out, fmt.Sprintf("run %s/%s/%06d/%s %+v %s judged=%v published=%v %v targets=%d hold=%s sum=%s den=%s alt=%s %+v %s %s %v res=%s/%s %s %s", ws, name, number, id, made.spec, made.state, made.judged, made.published, chunks,
						len(made.targets), made.hold, made.weightSum, made.denominator, alternative, made.approval, made.reason, shareText(made.shares), made.byBearer, made.residual, made.residualWhy, made.started.Format(time.RFC3339), made.changed.Format(time.RFC3339)))
				}
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

func alter[T any](base T, apply func(*T)) T {
	apply(&base)
	return base
}
