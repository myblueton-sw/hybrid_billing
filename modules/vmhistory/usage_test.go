package vmhistory

import (
	"testing"
	"time"
)

func TestTheCasesOfTheAcceptance(t *testing.T) {
	h := newHarness(t, nil)
	h.asset(machine("vm-1"))
	h.span(sized("s1", "vm-1", 0, 15*24, "4", "8 GiB", "100 GB", "org-a"))
	if got, want := cfg(h.usage("vm-1", october, november)), "incomplete[not_covered] 1296000/1382400 org-a:1296000 cpu=5184000/0 mem=11132555231232000/0 disk=129600000000000000/0"; got != want {
		t.Fatalf("a part of a month without a span is uncovered and not filled:\n got %s\nwant %s", got, want)
	}
	h.span(sized("s2", "vm-1", 15*24, 31*24, "8", "16 GiB", "100 GB", "org-a"))
	if got, want := cfg(h.usage("vm-1", october, november)), "complete[] 2678400/0 org-a:2678400 cpu=16243200/0 mem=34882006391193600/0 disk=267840000000000000/0"; got != want {
		t.Fatalf("a size changed within a month:\n got %s\nwant %s", got, want)
	}
	if got := en(h.usage("vm-1", october, november)); got != "incomplete[no_counter] 0/2678400 total=-" {
		t.Fatalf("no meter is not zero energy: %s", got)
	}
	h.asset(machine("vm-2"))
	empty := h.usage("vm-2", october, november)
	if cfg(empty) != "incomplete[not_covered] 0/2678400" || empty.Configuration.Entries != nil || empty.Configuration.None != nil || empty.Window != 2678400 {
		t.Fatalf("an asset without history: %s", cfg(empty))
	}
	h.span(sized("g1", "vm-2", 0, 1, "1", "1 GB", "1 GiB", ""))
	h.span(sized("g2", "vm-2", 1, 2, "1", "1 GiB", "1 GB", ""))
	if got := cfg(h.usage("vm-2", at(0), at(1))); got != "complete[] 3600/0 none=-:3600 cpu=3600/0 mem=3600000000000/0 disk=3865470566400/0" {
		t.Fatalf("GB is not GiB: %s", got)
	}
	if got := cfg(h.usage("vm-2", at(1), at(2))); got != "complete[] 3600/0 none=-:3600 cpu=3600/0 mem=3865470566400/0 disk=3600000000000/0" {
		t.Fatal(got)
	}
	for _, unit := range []string{"Gb", "gb", "KB", "GiB ", " GB", "gib", "G", "", "kb", "Kib", "PB"} {
		_, err := h.r.PutSpan("ws1", "vc-1", alter(sized("u", "vm-2", 2, 3, "1", "1 GB", "1 GB", ""), func(s *Span) { s.Memory.Unit = unit }))
		wantErr(t, err, ErrInvalid, "the unit "+unit)
	}
	for _, unit := range []string{"kW", "W", "MW", "kwh", "KWh", "J", "", "kWh "} {
		_, err := h.r.PutCounter("ws1", "vc-1", Counter{ID: "c", Asset: "vm-2", Unit: unit, MaxGap: 3600})
		wantErr(t, err, ErrInvalid, "a counter in "+unit)
	}
	h.counter("wh", "vm-2", "Wh")
	h.counter("kwh", "vm-1", "kWh")
	h.asset(machine("vm-3"))
	h.counter("mwh", "vm-3", "MWh")
	h.read("wh", rd(0, "1000", 1), rd(1, "1250.5", 1))
	h.read("kwh", rd(0, "1000", 1), rd(1, "1250.5", 1))
	h.read("mwh", rd(0, "0.000001", 1), rd(1, "0.000003", 1))
	for asset, want := range map[string]string{
		"vm-2": "complete[] 3600/0 total=250.5 wh=250.5/3600",
		"vm-1": "complete[] 3600/0 total=250500 kwh=250500/3600",
		"vm-3": "complete[] 3600/0 total=2 mwh=2/3600",
	} {
		if got := en(h.usage(asset, at(0), at(1))); got != want {
			t.Fatalf("%s: %s", asset, got)
		}
	}
	h.read("wh", rd(2, "1300", 1), rd(3, "20", 2), rd(4, "30", 2), rd(5, "25", 2), rd(6, "40", 2), rd(8, "50", 2), rd(9, "50", 2))
	for name, c := range map[string]struct {
		from, to int
		want     string
	}{
		"a reset by generation":     {2, 4, "incomplete[counter_reset] 3600/3600 total=10 wh=10/3600"},
		"a decrease":                {4, 6, "incomplete[counter_decrease] 3600/3600 total=15 wh=15/3600"},
		"a gap":                     {6, 9, "incomplete[counter_gap] 3600/7200 total=0 wh=0/3600"},
		"a meter that stood still":  {8, 9, "complete[] 3600/0 total=0 wh=0/3600"},
		"nothing but a reset":       {2, 3, "incomplete[counter_reset] 0/3600 total=- wh=-/0"},
		"before the first reading":  {-2, 1, "incomplete[not_read] 3600/7200 total=250.5 wh=250.5/3600"},
		"a window inside a segment": {6, 7, "incomplete[counter_gap] 0/3600 total=- wh=-/0"},
	} {
		if got := en(h.usage("vm-2", at(c.from), at(c.to))); got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", name, got, c.want)
		}
	}
	shorter, longer := time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 4, 0, 0, 0, time.UTC)
	h.asset(Asset{ID: "vm-4", Kind: VM, Created: shorter.Add(-day), Revision: 1})
	h.span(Span{ID: "dst", Asset: "vm-4", From: shorter.Add(-day), To: longer.Add(day), CPUs: "2"})
	if got := h.usage("vm-4", shorter, longer); got.Window != 743*3600 || got.Configuration.None.CPU.Integral != "5349600" {
		t.Fatalf("a month of a zone that lost an hour is taken as its two instants: %d %+v", got.Window, got.Configuration.None)
	}
	fall, back := time.Date(2025, 11, 1, 4, 0, 0, 0, time.UTC), time.Date(2025, 12, 1, 5, 0, 0, 0, time.UTC)
	h.asset(Asset{ID: "vm-5", Kind: Host, Created: fall, Revision: 1})
	h.span(Span{ID: "dst-2", Asset: "vm-5", From: fall, To: back, CPUs: "1"})
	if got := h.usage("vm-5", fall.In(time.FixedZone("edt", -4*3600)), back.In(time.FixedZone("est", -5*3600))); got.Window != 721*3600 || cfg(got) != "incomplete[memory_not_stated disk_not_stated] 2595600/0 none=-:2595600 cpu=2595600/0 mem=-/2595600 disk=-/2595600" {
		t.Fatalf("a month that gained an hour: %s", cfg(got))
	}
}
