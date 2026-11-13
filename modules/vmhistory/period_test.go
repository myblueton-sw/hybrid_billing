package vmhistory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func half(hours int) time.Time { return at(hours).Add(30 * time.Minute) }

func TestSegmentsAndTheEdgesOfAWindow(t *testing.T) {
	h := newHarness(t, nil)
	h.asset(Asset{ID: "vm", Kind: VM, Created: at(-1), Revision: 1})
	h.counter("c1", "vm", "Wh")
	h.read("c1", rd(3, "30", 1), rd(0, "0", 1), rd(2, "20", 1), rd(1, "10", 1))
	for name, c := range map[string]struct {
		from, to time.Time
		want     string
	}{
		"readings on both edges":           {at(1), at(3), "complete[] 7200/0 total=20 c1=20/7200"},
		"the whole chain":                  {at(0), at(3), "complete[] 10800/0 total=30 c1=30/10800"},
		"one segment":                      {at(2), at(3), "complete[] 3600/0 total=10 c1=10/3600"},
		"a start inside a segment":         {half(1), at(3), "incomplete[boundary_not_read] 3600/1800 total=10 c1=10/3600"},
		"an end inside a segment":          {at(1), half(2), "incomplete[boundary_not_read] 3600/1800 total=10 c1=10/3600"},
		"both edges inside segments":       {half(0), half(2), "incomplete[boundary_not_read] 3600/3600 total=10 c1=10/3600"},
		"a window inside one segment":      {half(1), half(1).Add(time.Minute), "incomplete[boundary_not_read] 0/60 total=- c1=-/0"},
		"a second inside a segment":        {at(1).Add(time.Second), at(1).Add(2 * time.Second), "incomplete[boundary_not_read] 0/1 total=- c1=-/0"},
		"past the last reading":            {at(2), at(5), "incomplete[not_read] 3600/7200 total=10 c1=10/3600"},
		"before the first reading":         {at(-1), at(1), "incomplete[not_read] 3600/3600 total=10 c1=10/3600"},
		"wholly past the last reading":     {at(3), at(4), "incomplete[not_read] 0/3600 total=- c1=-/0"},
		"wholly before the first":          {at(-1), at(0), "incomplete[not_read] 0/3600 total=- c1=-/0"},
		"an edge inside and time not read": {half(2), at(4), "incomplete[boundary_not_read not_read] 0/5400 total=- c1=-/0"},
	} {
		if got := en(h.usage("vm", c.from, c.to)); got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", name, got, c.want)
		}
	}
	h.read("c1", rd(4, "1000", 2), rd(5, "900", 2), rd(7, "950", 2), rd(8, "960", 2))
	h.counter("c2", "vm", "kWh")
	h.read("c2", rd(0, "0", 1), rd(1, "0.001", 1))
	want := "incomplete[counters_overlap counter_reset counter_decrease counter_gap boundary_not_read not_read] 10800/19800 total=- c1=30/10800 c2=1/3600"
	if got := en(h.usage("vm", at(-1), half(7))); got != want {
		t.Fatalf("every reason in its order:\n got %s\nwant %s", got, want)
	}
	for name, c := range map[string]struct {
		from, to time.Time
		want     string
	}{
		"a reset across the start":                                  {half(3), at(4), "incomplete[counter_reset] 0/1800 total=- c1=-/0 c2=-/0"},
		"a reset across the end":                                    {at(2), half(3), "incomplete[counter_reset] 3600/1800 total=10 c1=10/3600 c2=-/0"},
		"a decrease across the start":                               {half(4), at(5), "incomplete[counter_decrease] 0/1800 total=- c1=-/0 c2=-/0"},
		"a decrease wholly inside":                                  {at(4), at(5), "incomplete[counter_decrease] 0/3600 total=- c1=-/0 c2=-/0"},
		"a gap across the start":                                    {half(5), at(8), "incomplete[counter_gap] 3600/5400 total=10 c1=10/3600 c2=-/0"},
		"a gap across the end":                                      {at(5), half(5), "incomplete[counter_gap] 0/1800 total=- c1=-/0 c2=-/0"},
		"a gap wholly inside":                                       {at(5), at(7), "incomplete[counter_gap] 0/7200 total=- c1=-/0 c2=-/0"},
		"a decrease across the end":                                 {at(4), half(4), "incomplete[counter_decrease] 0/1800 total=- c1=-/0 c2=-/0"},
		"a gap across both edges":                                   {half(5), half(6), "incomplete[counter_gap] 0/3600 total=- c1=-/0 c2=-/0"},
		"a reset wholly inside":                                     {at(2), at(4), "incomplete[counter_reset] 3600/3600 total=10 c1=10/3600 c2=-/0"},
		"an edge on a reading between a paired and a reset segment": {at(2), at(3), "complete[] 3600/0 total=10 c1=10/3600 c2=-/0"},
		"two counters over the same hour":                           {at(0), at(1), "incomplete[counters_overlap] 3600/0 total=- c1=10/3600 c2=1/3600"},
		"one of them across the edge":                               {half(0), at(2), "incomplete[counters_overlap boundary_not_read] 3600/1800 total=- c1=10/3600 c2=-/0"},
	} {
		if got := en(h.usage("vm", c.from, c.to)); got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", name, got, c.want)
		}
	}
	if state, err := h.r.VoidCounter("ws1", "vc-1", "c2", WrongReport); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	if got := en(h.usage("vm", at(0), at(1))); got != "complete[] 3600/0 total=10 c1=10/3600" {
		t.Fatalf("a voided counter is in no result: %s", got)
	}
	if _, err := h.r.VoidCounter("ws1", "vc-1", "c1", WrongReport); err != nil {
		t.Fatal(err)
	}
	if got := en(h.usage("vm", at(0), at(1))); got != "incomplete[no_counter] 0/3600 total=-" {
		t.Fatal(got)
	}
}

func TestResetsVoidsAndSeveralCounters(t *testing.T) {
	h := newHarness(t, nil)
	h.asset(machine("vm"))
	h.counter("m", "vm", "Wh")
	h.read("m", rd(2, "50010", 2), rd(0, "1000", 1))
	if got := en(h.usage("vm", at(0), at(2))); got != "incomplete[counter_reset] 0/7200 total=- m=-/0" {
		t.Fatalf("a later reading reported before the one between: %s", got)
	}
	h.read("m", rd(1, "50000", 2))
	if got := en(h.usage("vm", at(0), at(2))); got != "incomplete[counter_reset] 3600/3600 total=10 m=10/3600" {
		t.Fatal(got)
	}
	if _, err := h.r.VoidReading("ws1", "vc-1", "m", at(1), 1, WrongReport); err != nil {
		t.Fatal(err)
	}
	if got := en(h.usage("vm", at(0), at(2))); got != "incomplete[counter_reset] 0/7200 total=- m=-/0" {
		t.Fatalf("voiding the first reading of a new generation does not join the two runs: %s", got)
	}
	h.read("m", rd(3, "50020", 2), rd(4, "50030", 2), rd(5, "50040", 2))
	if _, err := h.r.VoidReading("ws1", "vc-1", "m", at(4), 1, WrongReport); err != nil {
		t.Fatal(err)
	}
	if got := en(h.usage("vm", at(3), at(5))); got != "incomplete[counter_gap] 0/7200 total=- m=-/0" {
		t.Fatalf("voiding a reading makes its neighbours a segment of their own distance: %s", got)
	}
	h.read("m", Reading{At: at(4), Value: "50025", Generation: 2, Revision: 2})
	if got := en(h.usage("vm", at(3), at(5))); got != "complete[] 7200/0 total=20 m=20/7200" {
		t.Fatalf("a reading made live again: %s", got)
	}
	h.read("m", rd(6, "50040", 2), Reading{At: at(6).Add(3600 * time.Second), Value: "50041", Generation: 2, Revision: 1}, Reading{At: at(8).Add(time.Second), Value: "50042", Generation: 2, Revision: 1})
	if got := en(h.usage("vm", at(5), at(8).Add(time.Second))); got != "incomplete[counter_gap] 7200/3601 total=1 m=1/7200" {
		t.Fatalf("the longest distance pairs and a second more does not: %s", got)
	}
	seq := newHarness(t, nil)
	seq.asset(machine("vm"))
	seq.counter("a", "vm", "Wh")
	seq.counter("b", "vm", "Wh")
	seq.read("a", rd(0, "100", 1), rd(1, "110", 1), rd(2, "125", 1))
	seq.read("b", rd(2, "7", 1), rd(3, "8", 1), rd(4, "8.5", 1))
	if got := en(seq.usage("vm", at(0), at(4))); got != "complete[] 14400/0 total=26.5 a=25/7200 b=1.5/7200" {
		t.Fatalf("a meter replaced at one instant, as two counters that touch: %s", got)
	}
	if got := en(seq.usage("vm", at(2), at(4))); got != "complete[] 7200/0 total=1.5 a=-/0 b=1.5/7200" {
		t.Fatalf("a counter without a segment in the window has no energy, not zero: %s", got)
	}
	gap := newHarness(t, nil)
	gap.asset(machine("vm"))
	gap.counter("a", "vm", "Wh")
	gap.counter("b", "vm", "Wh")
	gap.read("a", rd(0, "100", 1), rd(1, "110", 1), rd(2, "125", 1))
	gap.read("b", Reading{At: at(2).Add(time.Second), Value: "7", Generation: 1, Revision: 1}, rd(3, "8", 1))
	if got := en(gap.usage("vm", at(0), at(3))); got != "incomplete[not_read] 10799/1 total=26 a=25/7200 b=1/3599" {
		t.Fatalf("a second between two meters is not read: %s", got)
	}
	beside := newHarness(t, nil)
	beside.asset(machine("vm"))
	beside.counter("a", "vm", "Wh")
	beside.counter("b", "vm", "Wh")
	beside.read("a", rd(0, "100", 1), rd(1, "110", 1), rd(2, "125", 1))
	beside.read("b", rd(0, "1", 1), rd(2, "2", 1))
	if got := en(beside.usage("vm", at(0), at(2))); got != "incomplete[counter_gap] 7200/0 total=25 a=25/7200 b=-/0" {
		t.Fatalf("a second counter with a gap beside one that covers the window: %s", got)
	}
	beside.read("b", rd(1, "1.5", 1))
	if got := en(beside.usage("vm", at(0), at(2))); got != "incomplete[counters_overlap] 7200/0 total=- a=25/7200 b=1/7200" {
		t.Fatal(got)
	}
	three := newHarness(t, nil)
	three.asset(machine("vm"))
	for _, id := range []string{"a", "b", "c"} {
		three.counter(id, "vm", "Wh")
	}
	three.read("a", rd(0, "1", 1), rd(1, "2", 1))
	three.read("b", rd(1, "1", 1), rd(2, "2", 1))
	three.read("c", rd(2, "1", 1), rd(3, "2", 1), rd(4, "3", 1))
	if got := en(three.usage("vm", at(0), at(4))); got != "complete[] 14400/0 total=4 a=1/3600 b=1/3600 c=2/7200" {
		t.Fatal(got)
	}
	three.read("a", rd(4, "9", 2), rd(3, "8", 2))
	if got := en(three.usage("vm", at(0), at(4))); got != "incomplete[counters_overlap counter_reset] 14400/0 total=- a=2/7200 b=1/3600 c=2/7200" {
		t.Fatalf("an overlap found behind a counter that ended earlier: %s", got)
	}
}

func TestWindowsOrganisationsAndNumbers(t *testing.T) {
	h := newHarness(t, nil)
	h.asset(Asset{ID: "vm", Kind: VM, Created: at(1), Deleted: at(5), Revision: 1})
	h.span(sized("b", "vm", 2, 3, "2", "1 KiB", "", "org-b"))
	h.span(sized("a", "vm", 3, 4, "", "2 KiB", "1 kB", "org-a"))
	h.span(sized("n", "vm", 1, 2, "1", "", "", ""))
	h.span(sized("a2", "vm", 4, 5, "0", "0 B", "3 kB", "org-a"))
	out := h.usage("vm", at(0), at(10))
	if !out.WindowStart.Equal(at(1)) || !out.WindowEnd.Equal(at(5)) || out.Window != 14400 || out.Asset != "vm" || out.WindowStart.Location() != time.UTC {
		t.Fatalf("a window cut by creation and deletion: %+v", out)
	}
	want := "incomplete[cpu_not_stated memory_not_stated disk_not_stated] 14400/0 org-a:7200 cpu=0/3600 mem=7372800/0 disk=14400000/0 org-b:3600 cpu=7200/0 mem=3686400/0 disk=-/3600 none=-:3600 cpu=3600/0 mem=-/3600 disk=-/3600"
	if got := cfg(out); got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if zoned := h.usage("vm", at(2).In(time.FixedZone("kst", 9*3600)), at(3).In(time.FixedZone("kst", 9*3600))); zoned.WindowStart.Location() != time.UTC || zoned.WindowEnd.Location() != time.UTC || zoned.Window != 3600 {
		t.Fatalf("a window is returned in UTC: %+v", zoned)
	}
	if got := cfg(h.usage("vm", half(2), half(3))); got != "incomplete[cpu_not_stated disk_not_stated] 3600/0 org-a:1800 cpu=-/1800 mem=3686400/0 disk=1800000/0 org-b:1800 cpu=3600/0 mem=1843200/0 disk=-/1800" {
		t.Fatalf("spans cut to the window: %s", got)
	}
	if _, err := h.r.VoidSpan("ws1", "vc-1", "a2", ByOperator); err != nil {
		t.Fatal(err)
	}
	if got := cfg(h.usage("vm", at(1), at(5))); !strings.HasPrefix(got, "incomplete[not_covered cpu_not_stated memory_not_stated disk_not_stated] 10800/3600 org-a:3600") || !strings.Contains(got, "none=") {
		t.Fatalf("every reason of configuration in its order: %s", got)
	}
	h.span(sized("a3", "vm", 4, 5, "0", "0 B", "3 kB", "org-a"))
	if got := cfg(h.usage("vm", at(4), at(5))); got != "complete[] 3600/0 org-a:3600 cpu=0/0 mem=0/0 disk=10800000/0" {
		t.Fatalf("zero is a stated value: %s", got)
	}
	for _, c := range [][2]int{{5, 6}, {-3, 1}, {8, 9}} {
		empty := h.usage("vm", at(c[0]), at(c[1]))
		if cfg(empty) != "not_applicable[] 0/0" || en(empty) != "not_applicable[] 0/0 total=-" || empty.Window != 0 || !empty.WindowStart.IsZero() || empty.Configuration.Entries != nil || empty.Energy.Counters != nil {
			t.Fatalf("an empty window: %s %s", cfg(empty), en(empty))
		}
	}
	if out := h.usage("vm", at(4).Add(59*time.Minute+59*time.Second), at(9)); out.Window != 1 || cfg(out) != "complete[] 1/0 org-a:1 cpu=0/0 mem=0/0 disk=3000/0" {
		t.Fatalf("the last second of a life: %s", cfg(out))
	}
	for name, period := range map[string][2]time.Time{"backwards": {at(2), at(1)}, "empty": {at(1), at(1)}, "a start with a fraction": {at(1).Add(time.Millisecond), at(2)}, "an end with a fraction": {at(1), at(2).Add(time.Millisecond)},
		"too long": {at(0), at(0).Add(40*day + time.Second)}, "none": {}} {
		_, err := h.r.Usage("ws1", "vc-1", "vm", period[0], period[1])
		wantErr(t, err, ErrInvalid, name)
	}
	if _, err := h.r.Usage("ws1", "vc-1", "vm", at(0), at(0).Add(40*day)); err != nil {
		t.Fatalf("the longest period: %v", err)
	}
	for _, c := range []struct {
		ws, source, id string
		want           error
	}{{"ws2", "vc-1", "vm", ErrNotFound}, {"ws1", "vc-2", "vm", ErrNotFound}, {"ws1", "vc-1", "vm-9", ErrNotFound}, {"", "vc-1", "vm", ErrInvalid}, {"ws1", "", "vm", ErrInvalid}, {"ws1", "vc-1", "", ErrInvalid}} {
		_, err := h.r.Usage(c.ws, c.source, c.id, at(0), at(1))
		wantErr(t, err, c.want, "usage")
	}
	wide := newHarness(t, nil)
	wide.asset(Asset{ID: "vm", Kind: VM, Created: at(1), Revision: 1})
	wide.span(sized("s", "vm", 1, 3, "1", "1 B", "1 B", ""))
	wide.counter("c", "vm", "Wh")
	wide.read("c", rd(1, "1", 1), rd(2, "2", 1), rd(3, "3", 1))
	if out := wide.usage("vm", at(0), at(3)); out.Configuration.Result != Complete || out.Energy.Result != Complete || out.Window != 7200 {
		t.Fatalf("%s %s", cfg(out), en(out))
	}
	wide.asset(Asset{ID: "vm", Kind: VM, Created: at(0), Revision: 2})
	if out := wide.usage("vm", at(0), at(3)); cfg(out) != "incomplete[not_covered] 7200/3600 none=-:7200 cpu=7200/0 mem=7200/0 disk=7200/0" || en(out) != "incomplete[not_read] 7200/3600 total=2 c=2/7200" {
		t.Fatalf("a revision that widens a life makes a complete result incomplete: %s %s", cfg(out), en(out))
	}
	live := newHarness(t, nil)
	live.asset(Asset{ID: "vm", Kind: VM, Created: start.Add(-2 * time.Hour), Revision: 1})
	live.span(Span{ID: "s", Asset: "vm", From: start.Add(-2 * time.Hour), To: start, CPUs: "1", Memory: Size{Value: "1", Unit: "B"}, Disk: Size{Value: "1", Unit: "B"}})
	if got := cfg(live.usage("vm", start.Add(-time.Hour), start.Add(time.Hour))); got != "incomplete[not_covered] 3600/3600 none=-:3600 cpu=3600/0 mem=3600/0 disk=3600/0" {
		t.Fatalf("a period that reaches past the registry's time: %s", got)
	}
	live.clock.advance(time.Hour)
	if got := live.usage("vm", start, start.Add(time.Hour)); got.Sequence != 2 || got.Change != 2 {
		t.Fatal(got.Sequence, got.Change)
	}
	live.clock.advance(-3 * time.Hour)
	live.span(Span{ID: "s2", Asset: "vm", From: start, To: start.Add(time.Hour)})
	_, err := live.r.PutSpan("ws1", "vc-1", Span{ID: "s3", Asset: "vm", From: start.Add(time.Hour), To: start.Add(time.Hour + time.Second)})
	wantErr(t, err, ErrRejected, "the registry's time is the highest reading of its clock and no later")
}

func TestNumbersPagesAndTheBoundaryOfAWorkspace(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Counters = 5 })
	numbers := func(asset string) string {
		out := h.usage(asset, at(0), at(1))
		return fmt.Sprintf("%d/%d", out.Sequence, out.Change)
	}
	h.asset(machine("a"))
	h.asset(machine("b"))
	if numbers("a") != "2/1" || numbers("b") != "2/2" {
		t.Fatal(numbers("a"), numbers("b"))
	}
	steps := []struct {
		name string
		call func() error
		want string
	}{
		{"a span", func() error { return second(h.r.PutSpan("ws1", "vc-1", sized("s", "a", 0, 1, "1", "", "", ""))) }, "3/3"},
		{"a counter", func() error {
			return second(h.r.PutCounter("ws1", "vc-1", Counter{ID: "c", Asset: "a", Unit: "Wh", MaxGap: 3600}))
		}, "4/4"},
		{"a batch of three readings", func() error {
			return second(h.r.PutReadings("ws1", "vc-1", "c", []Reading{rd(0, "1", 1), rd(1, "2", 1), rd(2, "3", 1)}))
		}, "5/5"},
		{"a batch of repeats", func() error {
			return second(h.r.PutReadings("ws1", "vc-1", "c", []Reading{rd(0, "1", 1), rd(1, "2", 1)}))
		}, "5/5"},
		{"a repeated span", func() error { return second(h.r.PutSpan("ws1", "vc-1", sized("s", "a", 0, 1, "1", "", "", ""))) }, "5/5"},
		{"a repeated counter", func() error {
			return second(h.r.PutCounter("ws1", "vc-1", Counter{ID: "c", Asset: "a", Unit: "Wh", MaxGap: 3600}))
		}, "5/5"},
		{"a repeated asset", func() error { return h.r.PutAsset("ws1", "vc-1", machine("a")) }, "5/5"},
		{"a reading voided", func() error { return second(h.r.VoidReading("ws1", "vc-1", "c", at(2), 1, ByOperator)) }, "6/6"},
		{"the void repeated", func() error { return second(h.r.VoidReading("ws1", "vc-1", "c", at(2), 1, ByOperator)) }, "6/6"},
		{"a reading replaced", func() error {
			return second(h.r.PutReadings("ws1", "vc-1", "c", []Reading{{At: at(1), Value: "2.5", Generation: 1, Revision: 2}}))
		}, "7/7"},
		{"a span voided", func() error { return second(h.r.VoidSpan("ws1", "vc-1", "s", ByOperator)) }, "8/8"},
		{"the void of the span repeated", func() error { return second(h.r.VoidSpan("ws1", "vc-1", "s", ByOperator)) }, "8/8"},
		{"a write for another asset", func() error { return second(h.r.PutSpan("ws1", "vc-1", sized("t", "b", 0, 1, "1", "", "", ""))) }, "9/8"},
		{"a counter voided", func() error { return second(h.r.VoidCounter("ws1", "vc-1", "c", ByOperator)) }, "10/10"},
		{"the void of the counter repeated", func() error { return second(h.r.VoidCounter("ws1", "vc-1", "c", ByOperator)) }, "10/10"},
		{"a revision", func() error {
			return h.r.PutAsset("ws1", "vc-1", alter(machine("a"), func(a *Asset) { a.Revision = 2 }))
		}, "11/11"},
	}
	for _, step := range steps {
		if err := step.call(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if got := numbers("a"); got != step.want {
			t.Fatalf("after %s: %s, want %s", step.name, got, step.want)
		}
	}
	for name, call := range map[string]func() error{
		"a refused asset": func() error { return h.r.PutAsset("ws1", "vc-1", machine("a")) },
		"a refused span":  func() error { return second(h.r.PutSpan("ws1", "vc-1", sized("t", "a", 0, 1, "1", "", "", ""))) },
		"a refused counter": func() error {
			return second(h.r.PutCounter("ws1", "vc-1", Counter{ID: "c", Asset: "a", Unit: "kWh", MaxGap: 3600}))
		},
		"a refused batch": func() error { return second(h.r.PutReadings("ws1", "vc-1", "c", []Reading{rd(5, "9", 1)})) },
		"a refused void":  func() error { return second(h.r.VoidSpan("ws1", "vc-1", "s", WrongReport)) },
	} {
		if call() == nil || numbers("a") != "11/11" {
			t.Fatalf("%s moved a number or succeeded", name)
		}
	}
	if numbers("b") != "11/9" {
		t.Fatalf("the change number of an asset moves with its own facts only: %s", numbers("b"))
	}
	spans, err := h.r.Spans("ws1", "vc-1", "b", time.Time{}, 1)
	if err != nil || spans.Sequence != 11 || spans.Change != 9 || len(spans.Items) != 1 || spans.More {
		t.Fatalf("%+v %v", spans, err)
	}
	counters, err := h.r.Counters("ws1", "vc-1", "a", "", 1)
	if err != nil || counters.Sequence != 11 || counters.Change != 11 {
		t.Fatalf("%+v %v", counters, err)
	}
	h.counter("r", "b", "Wh")
	h.read("r", rd(0, "1", 1))
	readings, err := h.r.Readings("ws1", "vc-1", "r", time.Time{}, 1)
	if err != nil || readings.Sequence != 13 || readings.Change != 13 || readings.More {
		t.Fatalf("%+v %v", readings, err)
	}
	for _, id := range []string{"m", "c", "x", "k"} {
		h.asset(machine(id))
		h.counter("counter-"+id, "b", "Wh")
	}
	for _, c := range []struct {
		after string
		limit int
		want  string
		more  bool
	}{{"", 4, "a b c k", true}, {"k", 4, "m x", false}, {"a", 2, "b c", true}, {"bb", 1, "c", true}, {"x", 4, "", false}, {"zz", 1, "", false}, {"m", 1, "x", false}} {
		page, err := h.r.Assets("ws1", "vc-1", c.after, c.limit)
		var ids []string
		for _, item := range page.Items {
			ids = append(ids, item.ID)
		}
		if err != nil || strings.Join(ids, " ") != c.want || page.More != c.more || page.Sequence != 21 {
			t.Fatalf("after %q: %v more=%v %v", c.after, ids, page.More, err)
		}
	}
	for _, c := range []struct {
		after string
		limit int
		want  string
		more  bool
	}{{"", 4, "counter-c counter-k counter-m counter-x", true}, {"counter-x", 4, "r", false}, {"counter-k", 1, "counter-m", true}, {"r", 2, "", false}} {
		page, err := h.r.Counters("ws1", "vc-1", "b", c.after, c.limit)
		var ids []string
		for _, item := range page.Items {
			ids = append(ids, item.Counter.ID)
		}
		if err != nil || strings.Join(ids, " ") != c.want || page.More != c.more {
			t.Fatalf("counters after %q: %v more=%v %v", c.after, ids, page.More, err)
		}
	}
	_, err = h.r.PutCounter("ws1", "vc-1", Counter{ID: "sixth", Asset: "b", Unit: "Wh", MaxGap: 1})
	wantErr(t, err, ErrLimit, "a sixth counter of an asset")
	for _, limit := range []int{0, 5, -1} {
		wantErr(t, second(h.r.Assets("ws1", "vc-1", "", limit)), ErrInvalid, "a page of assets")
		wantErr(t, second(h.r.Spans("ws1", "vc-1", "b", time.Time{}, limit)), ErrInvalid, "a page of spans")
		wantErr(t, second(h.r.Counters("ws1", "vc-1", "b", "", limit)), ErrInvalid, "a page of counters")
		wantErr(t, second(h.r.Readings("ws1", "vc-1", "r", time.Time{}, limit)), ErrInvalid, "a page of readings")
	}
	wantErr(t, second(h.r.Assets("ws1", "vc-1", "a b", 1)), ErrInvalid, "a malformed position")
	wantErr(t, second(h.r.Counters("ws1", "vc-1", "b", "a b", 1)), ErrInvalid, "a malformed position")
	wantErr(t, second(h.r.Spans("ws1", "vc-1", "b", at(0).Add(time.Millisecond), 1)), ErrInvalid, "a position with a fraction")
	wantErr(t, second(h.r.Readings("ws1", "vc-1", "r", at(0).Add(time.Millisecond), 1)), ErrInvalid, "a position with a fraction")
	for _, c := range []struct{ ws, source string }{{"ws2", "vc-1"}, {"ws1", "vc-9"}} {
		wantErr(t, second(h.r.Assets(c.ws, c.source, "", 1)), ErrNotFound, "assets")
		wantErr(t, second(h.r.Spans(c.ws, c.source, "b", time.Time{}, 1)), ErrNotFound, "spans")
		wantErr(t, second(h.r.Counters(c.ws, c.source, "b", "", 1)), ErrNotFound, "counters")
		wantErr(t, second(h.r.Readings(c.ws, c.source, "r", time.Time{}, 1)), ErrNotFound, "readings")
	}
	wantErr(t, second(h.r.Spans("ws1", "vc-1", "zz", at(0).Add(time.Millisecond), 1)), ErrInvalid, "a malformed position comes before the asset is looked up")
	wantErr(t, second(h.r.Counters("ws1", "vc-1", "zz", "a b", 1)), ErrInvalid, "a malformed position comes before the asset is looked up")
	wantErr(t, second(h.r.Spans("ws1", "vc-1", "zz", time.Time{}, 1)), ErrNotFound, "spans of an unknown asset")
	wantErr(t, second(h.r.Counters("ws1", "vc-1", "zz", "", 1)), ErrNotFound, "counters of an unknown asset")
	wantErr(t, second(h.r.Readings("ws1", "vc-1", "zz", time.Time{}, 1)), ErrNotFound, "readings of an unknown counter")
	alone := h.r.dump()
	if err := h.r.PutAsset("ws2", "vc-1", machine("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.PutSpan("ws2", "vc-1", sized("t", "a", 0, 1, "9", "", "", "")); err != nil {
		t.Fatalf("the same identifiers in another workspace are free: %v", err)
	}
	if _, err := h.r.PutCounter("ws2", "vc-1", Counter{ID: "r", Asset: "a", Unit: "MWh", MaxGap: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.VoidSpan("ws2", "vc-1", "t", ByOperator); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(alone, "\n") {
		if !strings.Contains(h.r.dump(), line) {
			t.Fatalf("another workspace changed: %s", line)
		}
	}
	if err := h.r.PutAsset("ws1", "vc-2", machine("a")); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.PutSpan("ws1", "vc-2", sized("t", "a", 0, 1, "9", "", "", "")); err != nil {
		t.Fatalf("the same identifiers in another source are free: %v", err)
	}
	if got := cfg(h.usage("b", at(0), at(1))); got != "incomplete[memory_not_stated disk_not_stated] 3600/0 none=-:3600 cpu=3600/0 mem=-/3600 disk=-/3600" {
		t.Fatal(got)
	}
	var unbuilt Registry
	var missing *Registry
	for _, r := range []*Registry{&unbuilt, missing} {
		wantErr(t, r.PutAsset("ws1", "vc-1", machine("a")), ErrInvalid, "asset")
		wantErr(t, second(r.PutSpan("ws1", "vc-1", sized("t", "a", 0, 1, "9", "", "", ""))), ErrInvalid, "span")
		wantErr(t, second(r.VoidSpan("ws1", "vc-1", "t", ByOperator)), ErrInvalid, "void span")
		wantErr(t, second(r.PutCounter("ws1", "vc-1", Counter{ID: "r", Asset: "a", Unit: "MWh", MaxGap: 1})), ErrInvalid, "counter")
		wantErr(t, second(r.VoidCounter("ws1", "vc-1", "r", ByOperator)), ErrInvalid, "void counter")
		wantErr(t, second(r.PutReadings("ws1", "vc-1", "r", []Reading{rd(0, "1", 1)})), ErrInvalid, "readings")
		wantErr(t, second(r.VoidReading("ws1", "vc-1", "r", at(0), 1, ByOperator)), ErrInvalid, "void reading")
		wantErr(t, second(r.Usage("ws1", "vc-1", "a", at(0), at(1))), ErrInvalid, "usage")
		wantErr(t, second(r.Assets("ws1", "vc-1", "", 1)), ErrInvalid, "assets")
		wantErr(t, second(r.Spans("ws1", "vc-1", "a", time.Time{}, 1)), ErrInvalid, "spans")
		wantErr(t, second(r.Counters("ws1", "vc-1", "a", "", 1)), ErrInvalid, "counters")
		wantErr(t, second(r.Readings("ws1", "vc-1", "r", time.Time{}, 1)), ErrInvalid, "readings page")
	}
}

func TestRefusalsAreFixedValuesAndTheSourceHasNoPrice(t *testing.T) {
	h := newHarness(t, nil)
	secret := "customer-9f3a"
	err := h.r.PutAsset("ws1", secret, alter(machine(secret), func(a *Asset) { a.Kind = AssetKind(secret) }))
	if err != ErrInvalid {
		t.Fatal(err)
	}
	for _, kind := range []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit} {
		if strings.ContainsAny(kind.Error(), "%0123456789") || strings.Contains(kind.Error(), secret) {
			t.Fatalf("%q", kind)
		}
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := regexp.MustCompile(`float32|float64|"math"|math\.|ParseFloat|big\.Float|big\.Rat|time\.Now|rand\.|(?i)\b(utili[sz]ation|rate|rates|price|prices|usage percent)\b`)
	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for number, line := range strings.Split(string(content), "\n") {
			if found := banned.FindString(line); found != "" {
				t.Errorf("%s:%d uses %s", name, number+1, found)
			}
			if strings.Contains(line, "//") || strings.Contains(line, "/*") {
				t.Errorf("%s:%d has a comment", name, number+1)
			}
		}
	}
	if scanned != 5 {
		t.Fatalf("scanned %d source files", scanned)
	}
}

func TestAnAssetAtItsLimits(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Spans, l.Readings, l.Batch = 2000, 5761, 1000 })
	created := start.Add(-40 * day)
	h.asset(Asset{ID: "vm", Kind: VM, Created: created, Revision: 1})
	for index := 0; index < 2000; index++ {
		from := created.Add(time.Duration(index) * 1728 * time.Second)
		h.span(Span{ID: fmt.Sprintf("s%04d", index), Asset: "vm", From: from, To: from.Add(1728 * time.Second), CPUs: fmt.Sprint(index%4 + 1), Memory: Size{Value: "1", Unit: "GiB"}, Organization: fmt.Sprintf("org-%d", index%3)})
	}
	h.counter("c", "vm", "kWh")
	for first := 0; first < 5761; first += 1000 {
		var batch []Reading
		for index := first; index < first+1000 && index < 5761; index++ {
			batch = append(batch, Reading{At: created.Add(time.Duration(index) * 600 * time.Second), Value: fmt.Sprintf("%d.%03d", index/1000, index%1000), Generation: 1, Revision: 1})
		}
		h.read("c", batch...)
	}
	out := h.usage("vm", created, start)
	if got := en(out); got != "complete[] 3456000/0 total=5760 c=5760/3456000" {
		t.Fatal(got)
	}
	_, err := h.r.PutReadings("ws1", "vc-1", "c", []Reading{{At: start.Add(-time.Second), Value: "9", Generation: 1, Revision: 1}})
	wantErr(t, err, ErrLimit, "the counter holds its limit of readings")
	_, err = h.r.PutSpan("ws1", "vc-1", Span{ID: "one-more", Asset: "vm", From: created, To: created.Add(time.Second)})
	wantErr(t, err, ErrRejected, "the asset holds its limit of spans and the time is taken")
	configuration := out.Configuration
	if configuration.Result != Incomplete || fmt.Sprint(configuration.Reasons) != "[disk_not_stated]" || configuration.Covered != 3456000 || len(configuration.Entries) != 3 || configuration.None != nil {
		t.Fatalf("%+v", configuration)
	}
	cpu, seconds := int64(0), int64(0)
	for index, entry := range configuration.Entries {
		if entry.Organization != fmt.Sprintf("org-%d", index) || entry.Disk.Integral != "" || entry.Disk.NotStatedSeconds != entry.Seconds {
			t.Fatalf("%+v", entry)
		}
		var value int64
		fmt.Sscan(entry.CPU.Integral, &value)
		cpu += value
		seconds += entry.Seconds
	}
	if seconds != 3456000 || cpu != 500*1728*(1+2+3+4) {
		t.Fatal(seconds, cpu)
	}
	if got := h.usage("vm", created.Add(1728*time.Second*1000), created.Add(1728*time.Second*1001)); cfg(got) != "incomplete[disk_not_stated] 1728/0 org-1:1728 cpu=1728/0 mem=1855425871872/0 disk=-/1728" || got.Energy.Result != Incomplete {
		t.Fatal(cfg(got), en(got))
	}
}
