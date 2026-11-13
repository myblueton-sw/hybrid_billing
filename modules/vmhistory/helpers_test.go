package vmhistory

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

const day = 24 * time.Hour

type testClock struct{ at time.Time }

func (c *testClock) read() time.Time { return c.at }

func (c *testClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func testLimits() Limits {
	return Limits{Sources: 3, Assets: 8, Spans: 12, Counters: 4, Readings: 40, Batch: 10, Page: 4, MaxPeriod: 40 * day, MaxGap: 7200}
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
	r, err := NewRegistry(limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func month(number int) (time.Time, time.Time) {
	return time.Date(2026, time.Month(number), 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.Month(number+1), 1, 0, 0, 0, 0, time.UTC)
}

var october, november = month(10)

func at(hours int) time.Time { return october.Add(time.Duration(hours) * time.Hour) }

func machine(id string) Asset {
	return Asset{ID: id, Kind: VM, Created: october.Add(-10 * day), Revision: 1}
}

func sized(id, asset string, from, to int, cpus, memory, disk, organization string) Span {
	parse := func(text string) Size {
		if text == "" {
			return Size{}
		}
		parts := strings.Fields(text)
		return Size{Value: parts[0], Unit: parts[1]}
	}
	return Span{ID: id, Asset: asset, From: at(from), To: at(to), CPUs: cpus, Memory: parse(memory), Disk: parse(disk), Organization: organization}
}

func rd(hours int, value string, generation int) Reading {
	return Reading{At: at(hours), Value: value, Generation: generation, Revision: 1}
}

func (h *harness) asset(spec Asset) {
	h.t.Helper()
	if err := h.r.PutAsset("ws1", "vc-1", spec); err != nil {
		h.t.Fatalf("asset %s: %v", spec.ID, err)
	}
}

func (h *harness) span(spec Span) {
	h.t.Helper()
	if state, err := h.r.PutSpan("ws1", "vc-1", spec); err != nil || state != Live {
		h.t.Fatalf("span %s: %v %v", spec.ID, state, err)
	}
}

func (h *harness) counter(id, asset, unit string) {
	h.t.Helper()
	if state, err := h.r.PutCounter("ws1", "vc-1", Counter{ID: id, Asset: asset, Unit: unit, MaxGap: 3600}); err != nil || state != Live {
		h.t.Fatalf("counter %s: %v %v", id, state, err)
	}
}

func (h *harness) read(id string, readings ...Reading) {
	h.t.Helper()
	if _, err := h.r.PutReadings("ws1", "vc-1", id, readings); err != nil {
		h.t.Fatalf("readings of %s: %v", id, err)
	}
}

func (h *harness) usage(id string, from, to time.Time) Result {
	h.t.Helper()
	out, err := h.r.Usage("ws1", "vc-1", id, from, to)
	if err != nil {
		h.t.Fatal(err)
	}
	return out
}

func absent(text string) string {
	if text == "" {
		return "-"
	}
	return text
}

func entryText(e Entry) string {
	return fmt.Sprintf("%s:%d cpu=%s/%d mem=%s/%d disk=%s/%d", absent(e.Organization), e.Seconds, absent(e.CPU.Integral), e.CPU.NotStatedSeconds, absent(e.Memory.Integral), e.Memory.NotStatedSeconds, absent(e.Disk.Integral), e.Disk.NotStatedSeconds)
}

func cfg(out Result) string {
	text := fmt.Sprintf("%s%v %d/%d", out.Configuration.Result, out.Configuration.Reasons, out.Configuration.Covered, out.Configuration.Uncovered)
	for _, entry := range out.Configuration.Entries {
		text += " " + entryText(entry)
	}
	if out.Configuration.None != nil {
		text += " none=" + entryText(*out.Configuration.None)
	}
	return text
}

func en(out Result) string {
	text := fmt.Sprintf("%s%v %d/%d total=%s", out.Energy.Result, out.Energy.Reasons, out.Energy.Covered, out.Energy.Uncovered, absent(out.Energy.Total))
	for _, entry := range out.Energy.Counters {
		text += fmt.Sprintf(" %s=%s/%d", entry.Counter, absent(entry.Energy), entry.Seconds)
	}
	return text
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for ws, sources := range r.workspaces {
		for name, home := range sources {
			out = append(out, fmt.Sprintf("source %s/%s sequence=%d assets=%d spans=%d counters=%d", ws, name, home.sequence, len(home.assets), len(home.spans), len(home.counters)))
			for id, owner := range home.assets {
				var live []string
				for _, held := range owner.live {
					live = append(live, held.spec.ID)
				}
				out = append(out, fmt.Sprintf("asset %s/%s/%s r%d %s spans=%d live=%v counters=%d change=%d", ws, name, id, owner.spec.Revision, owner.print, owner.spans, live, len(owner.counters), owner.change))
			}
			for id, held := range home.spans {
				out = append(out, fmt.Sprintf("span %s/%s/%s %s %s %s", ws, name, id, held.print, held.state, held.reason))
			}
			for id, meter := range home.counters {
				var parts []string
				for _, held := range meter.readings {
					parts = append(parts, fmt.Sprintf("%d r%d %s %s %s %s", held.spec.At.Unix(), held.spec.Revision, held.print, held.micro, held.state, held.reason))
				}
				sort.Strings(parts)
				var live []int64
				for _, held := range meter.live {
					live = append(live, held.spec.At.Unix())
				}
				out = append(out, fmt.Sprintf("counter %s/%s/%s %s %s %s %v live=%v", ws, name, id, meter.print, meter.state, meter.reason, parts, live))
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
