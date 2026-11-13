package costpool

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

func profileOf(t testing.TB, whole, result int) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "1", InputIntegerDigits: whole, InputScale: 4, ResultIntegerDigits: result, ResultScale: 6, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Pools: 4, Versions: 12, Assets: 6, Components: 12, Targets: 6, Page: 4, Listed: 2, MaxPeriod: 40 * day, Intermediate: 40}
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
	r, err := NewRegistry(limits, profileOf(t, 12, 14), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func month(number int) (time.Time, time.Time) {
	return time.Date(2026, time.Month(number), 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.Month(number+1), 1, 0, 0, 0, 0, time.UTC)
}

func header(number, monthNumber, supersedes int) Spec {
	from, to := month(monthNumber)
	return Spec{WorkspaceID: "ws1", Pool: "dc-1", Version: number, Currency: "KRW", PeriodStart: from, PeriodEnd: to, Owner: "owner-1", Quantum: "1", Supersedes: supersedes}
}

func cost(id string, kind Kind, amount string) Component {
	return Component{ID: id, Kind: kind, Amount: amount, Evidence: "doc-" + id}
}

func power(id, amount string, grade Grade, monthNumber int) Component {
	from, to := month(monthNumber)
	made := cost(id, PowerCost, amount)
	made.Power = &Power{Grade: grade, Source: "meter-1", PUE: "1.4", From: from, To: to}
	return made
}

func shares(weights ...string) Basis {
	made := Basis{ID: "basis-1", Measure: ReservedCapacity}
	bearers := []Bearer{Organization, Organization, MSP, Unattributed, Organization, MSP}
	sum := 0
	for index, weight := range weights {
		made.Targets = append(made.Targets, Target{ID: fmt.Sprintf("t%d", index+1), Bearer: bearers[index], Weight: weight})
		var value int
		fmt.Sscan(weight, &value)
		sum += value
	}
	made.Capacity = fmt.Sprint(sum)
	return made
}

var approved = Approval{Approver: "approver-1", Reference: "approval-doc-1"}

func (h *harness) open(spec Spec) {
	h.t.Helper()
	if state, err := h.r.Open(spec); err != nil || state != Draft {
		h.t.Fatalf("open %d: %v %v", spec.Version, state, err)
	}
}

func (h *harness) asset(number int, spec Asset) {
	h.t.Helper()
	if state, err := h.r.PutAsset("ws1", "dc-1", number, spec); err != nil || state != Draft {
		h.t.Fatalf("asset %s: %v %v", spec.ID, state, err)
	}
}

func (h *harness) put(number int, specs ...Component) {
	h.t.Helper()
	for _, spec := range specs {
		if state, err := h.r.PutComponent("ws1", "dc-1", number, spec); err != nil || state != Draft {
			h.t.Fatalf("component %s: %v %v", spec.ID, state, err)
		}
	}
}

func (h *harness) basis(number int, spec Basis) {
	h.t.Helper()
	if state, err := h.r.SetBasis("ws1", "dc-1", number, spec); err != nil || state != Draft {
		h.t.Fatalf("basis: %v %v", state, err)
	}
}

func (h *harness) record(number int) {
	h.t.Helper()
	if state, err := h.r.RecordApproval("ws1", "dc-1", number, approved); err != nil || state != ApprovalRecorded {
		h.t.Fatalf("record %d: %v %v", number, state, err)
	}
}

func (h *harness) fill(spec Spec, amount string) {
	h.t.Helper()
	h.open(spec)
	h.put(spec.Version, cost("fee", Facility, amount))
	h.basis(spec.Version, shares("60", "20", "20"))
}

func (h *harness) whole(spec Spec, amount string) {
	h.t.Helper()
	h.fill(spec, amount)
	h.record(spec.Version)
}

func (h *harness) view(number int) View {
	h.t.Helper()
	view, err := h.r.Version("ws1", "dc-1", number)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func (h *harness) standing(number int) string {
	h.t.Helper()
	out, err := h.r.Standing("ws1", "dc-1", number)
	if err != nil {
		h.t.Fatal(err)
	}
	return fmt.Sprintf("%s%v", out.Result, out.Reasons)
}

func (h *harness) counting(monthNumber int) (int, uint64) {
	h.t.Helper()
	from, to := month(monthNumber)
	out, err := h.r.Counting("ws1", "dc-1", from, to)
	if err != nil && err != ErrNotFound {
		h.t.Fatal(err)
	}
	return out.Version, out.Sequence
}

func (h *harness) shares(number int) string {
	h.t.Helper()
	var parts []string
	after := ""
	for {
		page, err := h.r.Shares("ws1", "dc-1", number, after, h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		if len(page) == 0 {
			return strings.Join(parts, " ")
		}
		for _, share := range page {
			idle := ""
			if share.Idle {
				idle = "/idle"
			}
			parts = append(parts, fmt.Sprintf("%s=%s(%s/%s%s)", share.Target, share.Amount, share.Bearer, share.Weight, idle))
		}
		after = page[len(page)-1].Target
	}
}

func (h *harness) assetSums(number int) string {
	h.t.Helper()
	var parts []string
	after := ""
	for {
		page, err := h.r.AssetSums("ws1", "dc-1", number, after, h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		if len(page) == 0 {
			return strings.Join(parts, " ")
		}
		for _, sum := range page {
			parts = append(parts, sum.Asset+"="+sum.Sum)
		}
		after = page[len(page)-1].Asset
	}
}

func flat(figures *Figures) string {
	if figures == nil {
		return "none"
	}
	parts := []string{}
	for kind, sum := range figures.ByKind {
		parts = append(parts, fmt.Sprintf("kind/%s=%s", kind, sum))
	}
	for grade, sum := range figures.ByGrade {
		parts = append(parts, fmt.Sprintf("grade/%s=%s", grade, sum))
	}
	for class, sum := range figures.ByBearer {
		parts = append(parts, fmt.Sprintf("bearer/%s=%s", class, sum))
	}
	sort.Strings(parts)
	out := fmt.Sprintf("total=%s assets=%d included=%d/%d prices=%d/%d", figures.Total, figures.AssetSums, figures.IncludedKnown, figures.IncludedNotKnown, figures.PricesStated, figures.PricesNotStated)
	if figures.HasPoolLevel {
		out += " pool=" + figures.PoolLevel
	}
	if figures.HasIdle {
		out += " idle=" + figures.Idle
	}
	return out + " " + strings.Join(parts, " ")
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for ws, pools := range r.workspaces {
		for name, home := range pools {
			out = append(out, fmt.Sprintf("pool %s/%s sequence=%d versions=%d", ws, name, home.sequence, len(home.versions)))
			for number, held := range home.versions {
				var parts []string
				for id, owned := range held.assets {
					parts = append(parts, "a:"+id+owned.print+fmt.Sprint(owned.component))
				}
				for id, made := range held.components {
					parts = append(parts, "c:"+id+made.print)
				}
				sort.Strings(parts)
				print := ""
				if held.basis != nil {
					print = held.basis.print
				}
				out = append(out, fmt.Sprintf("version %s/%s/%06d %s %s %v %v/%d %+v %s %v %s %s %v %v %s %s", ws, name, number, held.print, held.state, held.recorded, held.superseded, held.supersededBy, held.approval, held.reason,
					parts, print, flat(held.figures), held.shares, held.assetSums, held.opened.Format(time.RFC3339Nano), held.changed.Format(time.RFC3339Nano)))
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
