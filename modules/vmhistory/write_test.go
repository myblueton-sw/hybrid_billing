package vmhistory

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestConstructionAndForms(t *testing.T) {
	clock := func() time.Time { return start }
	for name, change := range map[string]func(*Limits){
		"sources": func(l *Limits) { l.Sources = 0 }, "assets": func(l *Limits) { l.Assets = 0 }, "spans": func(l *Limits) { l.Spans = 0 }, "counters": func(l *Limits) { l.Counters = 0 }, "readings": func(l *Limits) { l.Readings = 0 },
		"batch": func(l *Limits) { l.Batch = 0 }, "page": func(l *Limits) { l.Page = 0 }, "period": func(l *Limits) { l.MaxPeriod = 0 }, "gap": func(l *Limits) { l.MaxGap = 0 },
		"a period beyond a year": func(l *Limits) { l.MaxPeriod = 366*day + time.Second }, "a distance beyond the longest period": func(l *Limits) { l.MaxGap = 40*86400 + 1 },
	} {
		limits := testLimits()
		change(&limits)
		_, err := NewRegistry(limits, clock)
		wantErr(t, err, ErrInvalid, name)
	}
	_, err := NewRegistry(testLimits(), nil)
	wantErr(t, err, ErrInvalid, "no clock")
	widest := testLimits()
	widest.MaxPeriod, widest.MaxGap = 366*day, 366*86400
	if _, err := NewRegistry(widest, clock); err != nil {
		t.Fatalf("the widest limits: %v", err)
	}
	for id, want := range map[string]bool{"a": true, "A.z_0:9-": true, strings.Repeat("a", 128): true, strings.Repeat("a", 129): false, "": false, "a b": false, "a/b": false, "kim@example": false} {
		if validID(id) != want {
			t.Errorf("identifier %q", id)
		}
	}
	for text, want := range map[string]string{"0": "0", "7": "7", "10": "10", "123456789012345678": "123456789012345678", "1234567890123456789": "", "": "", "01": "", "00": "", "-1": "", "+1": "", "1.0": "", " 1": "", "1 ": "", "1e3": "", "٣": ""} {
		value, formed := whole(text, 18)
		if formed != (want != "") || (formed && value.String() != want) {
			t.Errorf("whole number %q: %v %v", text, value, formed)
		}
	}
	for text, want := range map[string]string{"0": "0", "1": "1", "1.5": "1.5", "1.50": "1.5", "0.000001": "0.000001", "123456789012345.999999": "123456789012345.999999", "10.010": "10.01", "5.000000": "5",
		"1234567890123456": "", "0.0000001": "", "1.": "", ".5": "", "": "", "-1": "", "1e3": "", "01.5": "", "1,5": "", "1.5 ": "", "1..5": "", "1.x": ""} {
		value, formed := microOf(text, energyUnits["Wh"])
		if formed != (want != "") || (formed && wattHours(value) != want) {
			t.Errorf("value %q: %v %v", text, value, formed)
		}
	}
	for unit, want := range map[string]string{"B": "999999999999999999", "kB": "999999999999999999000", "MB": "999999999999999999000000", "GB": "999999999999999999000000000", "TB": "999999999999999999000000000000",
		"KiB": "1023999999999999998976", "MiB": "1048575999999999998951424", "GiB": "1073741823999999998926258176", "TiB": "1099511627775999998900488372224"} {
		bytes, formed := bytesOf(Size{Value: "999999999999999999", Unit: unit})
		if !formed || bytes.String() != want {
			t.Errorf("%s: %v %v", unit, bytes, formed)
		}
	}
	for _, size := range []Size{{Value: "1"}, {Unit: "GB"}, {Value: "1.5", Unit: "GB"}, {Value: "1234567890123456789", Unit: "B"}, {Value: "-1", Unit: "B"}, {Value: "01", Unit: "B"}} {
		if _, formed := bytesOf(size); formed {
			t.Errorf("size %+v", size)
		}
	}
	if bytes, formed := bytesOf(Size{}); !formed || bytes != nil {
		t.Error("a size that is not stated")
	}
	if got := mustMicro(t, "2.5", "MWh"); got != "2500000" {
		t.Error(got)
	}
	if got := mustMicro(t, "0.0015", "kWh"); got != "1.5" {
		t.Error(got)
	}
}

func mustMicro(t *testing.T, text, unit string) string {
	t.Helper()
	value, formed := microOf(text, energyUnits[unit])
	if !formed {
		t.Fatal(text)
	}
	return wattHours(value)
}

func TestAssets(t *testing.T) {
	h := newHarness(t, nil)
	good := machine("vm-1")
	for name, apply := range map[string]func(*Asset){
		"identifier": func(a *Asset) { a.ID = "vm 1" }, "kind": func(a *Asset) { a.Kind = "cluster" }, "no kind": func(a *Asset) { a.Kind = "" }, "revision zero": func(a *Asset) { a.Revision = 0 },
		"a revision above the maximum": func(a *Asset) { a.Revision = 1000000 }, "no creation": func(a *Asset) { a.Created = time.Time{} }, "a creation with a fraction": func(a *Asset) { a.Created = a.Created.Add(time.Millisecond) },
		"a deletion with a fraction": func(a *Asset) { a.Deleted = a.Created.Add(time.Hour + time.Millisecond) }, "a deletion at the creation": func(a *Asset) { a.Deleted = a.Created },
		"a deletion before the creation": func(a *Asset) { a.Deleted = a.Created.Add(-time.Second) },
	} {
		wantErr(t, h.r.PutAsset("ws1", "vc-1", alter(good, apply)), ErrInvalid, name)
	}
	wantErr(t, h.r.PutAsset("", "vc-1", good), ErrInvalid, "workspace")
	wantErr(t, h.r.PutAsset("ws1", "", good), ErrInvalid, "source")
	wantErr(t, h.r.PutAsset("ws1", "vc-1", alter(good, func(a *Asset) { a.Created = start.Add(time.Second) })), ErrRejected, "a creation after the registry's time")
	wantErr(t, h.r.PutAsset("ws1", "vc-1", alter(good, func(a *Asset) { a.Deleted = start.Add(time.Second) })), ErrRejected, "a deletion after the registry's time")
	if h.r.dump() != "" {
		t.Fatal("a refused first asset created its source")
	}
	if _, err := h.r.Assets("ws1", "vc-1", "", 1); err != ErrNotFound {
		t.Fatal(err)
	}
	h.asset(alter(good, func(a *Asset) {
		a.Created, a.Deleted, a.Revision = start.Add(-time.Hour).In(time.FixedZone("kst", 9*3600)), start.In(time.FixedZone("kst", 9*3600)), 999999
	}))
	if page, err := h.r.Assets("ws1", "vc-1", "", 1); err != nil || page.Items[0].Created.Location() != time.UTC || page.Items[0].Deleted.Location() != time.UTC || !page.Items[0].Deleted.Equal(start) {
		t.Fatalf("a life is kept in UTC: %+v %v", page, err)
	}
	h = newHarness(t, nil)
	h.asset(good)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.asset(good)
	for name, apply := range map[string]func(*Asset){"kind": func(a *Asset) { a.Kind = Host }, "creation": func(a *Asset) { a.Created = a.Created.Add(time.Second) }, "deletion": func(a *Asset) { a.Deleted = start }} {
		wantErr(t, h.r.PutAsset("ws1", "vc-1", alter(good, apply)), ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	h.span(sized("s1", "vm-1", 0, 24, "2", "", "", ""))
	h.counter("c1", "vm-1", "kWh")
	h.read("c1", rd(-24, "1", 1), rd(48, "2", 1))
	h.counter("c2", "vm-1", "kWh")
	h.read("c2", rd(72, "1", 1))
	revise := func(revision int, created, deleted int) error {
		spec := Asset{ID: "vm-1", Kind: Host, Created: at(created), Revision: revision}
		if deleted != 0 {
			spec.Deleted = at(deleted)
		}
		return h.r.PutAsset("ws1", "vc-1", spec)
	}
	before = h.r.dump()
	for name, c := range map[string][2]int{"a creation after the first span": {1, 0}, "a creation after the first reading": {-23, 0}, "a deletion before the end of the last span": {-240, 23},
		"a deletion before the last reading of one counter": {-240, 71}, "a deletion before the last reading of another": {-240, 47}} {
		wantErr(t, revise(2, c[0], c[1]), ErrRejected, name)
	}
	h.unchanged(before, "a refused revision")
	if _, err := h.r.VoidCounter("ws1", "vc-1", "c2", WrongReport); err != nil {
		t.Fatal(err)
	}
	if err := revise(2, -24, 48); err != nil {
		t.Fatalf("a reading on the deletion and one on the creation lie within the life, and a voided counter holds nothing: %v", err)
	}
	if page, err := h.r.Assets("ws1", "vc-1", "", 4); err != nil || len(page.Items) != 1 || page.Items[0] != (Asset{ID: "vm-1", Kind: Host, Created: at(-24), Deleted: at(48), Revision: 2}) || page.Sequence != 8 {
		t.Fatalf("a higher revision replaces kind and lifetime: %+v %v", page, err)
	}
	wantErr(t, h.r.PutAsset("ws1", "vc-1", good), ErrRejected, "a stale revision, also an identical old retry")
	wantErr(t, revise(2, -24, 49), ErrConflict, "the stored revision with another lifetime")
	wantErr(t, revise(3, -24, 24*50), ErrRejected, "a revised deletion in the future")
	if err := revise(5, -24, 0); err != nil {
		t.Fatalf("a revision need not follow the last, and may take a deletion back: %v", err)
	}
	wantErr(t, revise(4, -24, 0), ErrRejected, "a revision below the stored one")
	limited := newHarness(t, func(l *Limits) { l.Sources, l.Assets = 2, 2 })
	limited.asset(machine("a"))
	limited.asset(machine("b"))
	wantErr(t, limited.r.PutAsset("ws1", "vc-1", machine("c")), ErrLimit, "a third asset")
	if err := limited.r.PutAsset("ws1", "vc-2", machine("a")); err != nil {
		t.Fatal(err)
	}
	before = limited.r.dump()
	wantErr(t, limited.r.PutAsset("ws1", "vc-3", machine("a")), ErrLimit, "a third source")
	limited.unchanged(before, "limits")
	if err := limited.r.PutAsset("ws2", "vc-3", machine("a")); err != nil {
		t.Fatalf("limits are per workspace: %v", err)
	}
	limited.asset(alter(machine("b"), func(a *Asset) { a.Revision = 2; a.Kind = Host }))
}

func TestSpans(t *testing.T) {
	h := newHarness(t, nil)
	h.asset(Asset{ID: "vm-1", Kind: VM, Created: at(0), Deleted: at(100), Revision: 1})
	good := sized("s1", "vm-1", 10, 20, "4", "8 GiB", "100 GB", "org-a")
	for name, apply := range map[string]func(*Span){
		"identifier": func(s *Span) { s.ID = "" }, "asset": func(s *Span) { s.Asset = "vm 1" }, "an organisation with a space": func(s *Span) { s.Organization = "org a" }, "no start": func(s *Span) { s.From = time.Time{} },
		"no end": func(s *Span) { s.To = time.Time{} }, "a start with a fraction": func(s *Span) { s.From = s.From.Add(time.Millisecond) }, "an end with a fraction": func(s *Span) { s.To = s.To.Add(time.Millisecond) },
		"an end at the start": func(s *Span) { s.To = s.From }, "an end before the start": func(s *Span) { s.To = s.From.Add(-time.Second) }, "CPUs that are no number": func(s *Span) { s.CPUs = "four" },
		"CPUs with a leading zero": func(s *Span) { s.CPUs = "04" }, "negative CPUs": func(s *Span) { s.CPUs = "-1" }, "more CPUs than the bound": func(s *Span) { s.CPUs = "1000001" }, "a memory value without a unit": func(s *Span) { s.Memory.Unit = "" },
		"a memory unit without a value": func(s *Span) { s.Memory.Value = "" }, "a disk value without a unit": func(s *Span) { s.Disk.Unit = "" }, "a disk unit without a value": func(s *Span) { s.Disk.Value = "" },
		"a disk of a fraction": func(s *Span) { s.Disk.Value = "1.5" }, "a disk in an unknown unit": func(s *Span) { s.Disk.Unit = "gb" }, "a memory of nineteen digits": func(s *Span) { s.Memory.Value = "1234567890123456789" },
	} {
		_, err := h.r.PutSpan("ws1", "vc-1", alter(good, apply))
		wantErr(t, err, ErrInvalid, name)
		_, err = h.r.PutSpan("ws1", "vc-9", alter(good, apply))
		wantErr(t, err, ErrInvalid, name+" before anything is looked up")
	}
	for _, c := range []struct{ ws, source, asset string }{{"ws2", "vc-1", "vm-1"}, {"ws1", "vc-2", "vm-1"}, {"ws1", "vc-1", "vm-2"}} {
		_, err := h.r.PutSpan(c.ws, c.source, alter(good, func(s *Span) { s.Asset = c.asset }))
		wantErr(t, err, ErrNotFound, "a span for what does not exist")
	}
	h.span(alter(good, func(s *Span) { s.From = s.From.In(time.FixedZone("kst", 9*3600)) }))
	if state, err := h.r.PutSpan("ws1", "vc-1", alter(sized("most", "vm-1", 30, 31, "1000000", "", "", ""), func(s *Span) {})); err != nil || state != Live {
		t.Fatalf("the highest number of CPUs: %v %v", state, err)
	}
	if _, err := h.r.VoidSpan("ws1", "vc-1", "most", ByOperator); err != nil {
		t.Fatal(err)
	}
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.span(good)
	h.unchanged(before, "a repeated span")
	_, err := h.r.PutSpan("ws1", "vc-1", alter(good, func(s *Span) { s.Asset = "vm-9" }))
	wantErr(t, err, ErrNotFound, "a known span for an unknown asset")
	h.asset(Asset{ID: "vm-2", Kind: VM, Created: at(0), Revision: 1})
	before = h.r.dump()
	for name, apply := range map[string]func(*Span){
		"the asset": func(s *Span) { s.Asset = "vm-2" }, "start": func(s *Span) { s.From = at(11) }, "end": func(s *Span) { s.To = at(19) }, "CPUs": func(s *Span) { s.CPUs = "8" }, "CPUs not stated": func(s *Span) { s.CPUs = "" },
		"memory": func(s *Span) { s.Memory.Value = "16" }, "the unit of memory": func(s *Span) { s.Memory.Unit = "GB" }, "the same memory in another unit": func(s *Span) { s.Memory = Size{Value: "8192", Unit: "MiB"} },
		"disk": func(s *Span) { s.Disk.Value = "200" }, "the unit of disk": func(s *Span) { s.Disk.Unit = "GiB" }, "no disk": func(s *Span) { s.Disk = Size{} }, "organisation": func(s *Span) { s.Organization = "org-b" }, "no organisation": func(s *Span) { s.Organization = "" },
		"memory and disk exchanged": func(s *Span) { s.Memory, s.Disk = s.Disk, s.Memory },
	} {
		_, err := h.r.PutSpan("ws1", "vc-1", alter(good, apply))
		wantErr(t, err, ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	next := func(id string, from, to int) error {
		_, err := h.r.PutSpan("ws1", "vc-1", sized(id, "vm-1", from, to, "1", "", "", ""))
		return err
	}
	for name, c := range map[string][2]int{"before the creation": {-1, 5}, "past the deletion": {95, 101}, "inside another": {12, 15}, "around another": {5, 25}, "over the start of another": {5, 11}, "over the end of another": {19, 25},
		"exactly another": {10, 20}} {
		wantErr(t, next("x", c[0], c[1]), ErrRejected, name)
	}
	h.asset(Asset{ID: "vm-3", Kind: VM, Created: at(0), Revision: 1})
	_, err = h.r.PutSpan("ws1", "vc-1", Span{ID: "future", Asset: "vm-3", From: start.Add(-time.Hour), To: start.Add(time.Hour + time.Second)})
	wantErr(t, err, ErrRejected, "an end after the registry's time")
	h.span(Span{ID: "now", Asset: "vm-3", From: start.Add(-time.Hour), To: start.Add(time.Hour)})
	h.counter("edge", "vm-3", "Wh")
	_, err = h.r.PutReadings("ws1", "vc-1", "edge", []Reading{{At: start.Add(time.Hour + time.Second), Value: "1", Generation: 1, Revision: 1}})
	wantErr(t, err, ErrRejected, "a reading one second after the registry's time")
	h.read("edge", Reading{At: start.Add(time.Hour), Value: "1", Generation: 1, Revision: 1})
	for id, c := range map[string][2]int{"touching-before": {0, 10}, "touching-after": {20, 30}, "last": {99, 100}} {
		if err := next(id, c[0], c[1]); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	page, err := h.r.Spans("ws1", "vc-1", "vm-1", time.Time{}, 4)
	if err != nil || len(page.Items) != 4 || page.More || page.Items[0].ID != "touching-before" || page.Items[1] != alter(good, func(s *Span) {}) || page.Items[3].ID != "last" || page.Items[1].From.Location() != time.UTC {
		t.Fatalf("%+v %v", page, err)
	}
	for _, reason := range []VoidReason{"", "mistake"} {
		_, err := h.r.VoidSpan("ws1", "vc-1", "s1", reason)
		wantErr(t, err, ErrInvalid, "a reason")
	}
	for _, c := range []struct{ ws, source, id string }{{"ws2", "vc-1", "s1"}, {"ws1", "vc-2", "s1"}, {"ws1", "vc-1", "s9"}} {
		_, err := h.r.VoidSpan(c.ws, c.source, c.id, ByOperator)
		wantErr(t, err, ErrNotFound, "a void of what does not exist")
	}
	before = h.r.dump()
	if state, err := h.r.VoidSpan("ws1", "vc-1", "s1", WrongReport); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	if h.r.dump() == before {
		t.Fatal("a void wrote nothing")
	}
	before = h.r.dump()
	if state, err := h.r.VoidSpan("ws1", "vc-1", "s1", WrongReport); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	_, err = h.r.VoidSpan("ws1", "vc-1", "s1", ByOperator)
	wantErr(t, err, ErrConflict, "a second reason")
	if state, err := h.r.PutSpan("ws1", "vc-1", good); err != nil || state != Voided {
		t.Fatalf("a repeat of a voided span says so: %v %v", state, err)
	}
	_, err = h.r.PutSpan("ws1", "vc-1", alter(good, func(s *Span) { s.CPUs = "8" }))
	wantErr(t, err, ErrConflict, "a voided identifier stays taken")
	h.unchanged(before, "a voided span")
	if err := next("again", 12, 18); err != nil {
		t.Fatalf("the time of a voided span is free: %v", err)
	}
	page, err = h.r.Spans("ws1", "vc-1", "vm-1", at(0), 2)
	if err != nil || len(page.Items) != 2 || !page.More || page.Items[0].ID != "again" || page.Items[1].ID != "touching-after" {
		t.Fatalf("voided spans are not listed and the position is exclusive: %+v %v", page, err)
	}
	limited := newHarness(t, func(l *Limits) { l.Spans = 2 })
	limited.asset(machine("vm-1"))
	limited.span(sized("a", "vm-1", 0, 1, "", "", "", ""))
	if _, err := limited.r.VoidSpan("ws1", "vc-1", "a", ByOperator); err != nil {
		t.Fatal(err)
	}
	limited.span(sized("b", "vm-1", 0, 1, "", "", "", ""))
	_, err = limited.r.PutSpan("ws1", "vc-1", sized("c", "vm-1", 1, 2, "", "", "", ""))
	wantErr(t, err, ErrLimit, "a voided span counts towards the limit")
	_, err = limited.r.PutSpan("ws1", "vc-1", sized("c", "vm-1", 0, 2, "", "", "", ""))
	wantErr(t, err, ErrRejected, "rejected before no room")
	limited.asset(machine("vm-2"))
	limited.span(sized("d", "vm-2", 0, 1, "", "", "", ""))
}

func TestCountersAndReadings(t *testing.T) {
	h := newHarness(t, nil)
	h.asset(Asset{ID: "vm-1", Kind: VM, Created: at(0), Deleted: at(100), Revision: 1})
	good := Counter{ID: "c1", Asset: "vm-1", Unit: "kWh", MaxGap: 3600}
	for name, apply := range map[string]func(*Counter){"identifier": func(c *Counter) { c.ID = "" }, "asset": func(c *Counter) { c.Asset = "" }, "a distance of zero": func(c *Counter) { c.MaxGap = 0 },
		"a negative distance": func(c *Counter) { c.MaxGap = -1 }, "a distance above the limit": func(c *Counter) { c.MaxGap = 7201 }} {
		_, err := h.r.PutCounter("ws1", "vc-1", alter(good, apply))
		wantErr(t, err, ErrInvalid, name)
	}
	for _, c := range []struct{ ws, source, asset string }{{"ws2", "vc-1", "vm-1"}, {"ws1", "vc-2", "vm-1"}, {"ws1", "vc-1", "vm-2"}} {
		_, err := h.r.PutCounter(c.ws, c.source, alter(good, func(k *Counter) { k.Asset = c.asset }))
		wantErr(t, err, ErrNotFound, "a counter for what does not exist")
	}
	if state, err := h.r.PutCounter("ws1", "vc-1", alter(good, func(c *Counter) { c.MaxGap = 7200 })); err != nil || state != Live {
		t.Fatal(state, err)
	}
	h = newHarness(t, nil)
	h.asset(Asset{ID: "vm-1", Kind: VM, Created: at(0), Deleted: at(100), Revision: 1})
	h.asset(machine("vm-2"))
	h.counter("c1", "vm-1", "kWh")
	before := h.r.dump()
	h.counter("c1", "vm-1", "kWh")
	for name, apply := range map[string]func(*Counter){"asset": func(c *Counter) { c.Asset = "vm-2" }, "unit": func(c *Counter) { c.Unit = "Wh" }, "distance": func(c *Counter) { c.MaxGap = 60 }} {
		_, err := h.r.PutCounter("ws1", "vc-1", alter(good, apply))
		wantErr(t, err, ErrConflict, name)
	}
	_, err := h.r.PutCounter("ws1", "vc-1", alter(good, func(c *Counter) { c.Asset = "vm-9" }))
	wantErr(t, err, ErrNotFound, "a known counter for an unknown asset")
	h.unchanged(before, "repeats and conflicts")
	put := func(readings ...Reading) error {
		_, err := h.r.PutReadings("ws1", "vc-1", "c1", readings)
		return err
	}
	for name, batch := range map[string][]Reading{
		"an empty batch": nil, "no instant": {{Value: "1", Generation: 1, Revision: 1}}, "an instant with a fraction": {alter(rd(1, "1", 1), func(r *Reading) { r.At = r.At.Add(time.Millisecond) })},
		"an instant twice": {rd(1, "1", 1), rd(2, "1", 1), rd(1, "2", 1)}, "generation zero": {rd(1, "1", 0)}, "a generation above the maximum": {rd(1, "1", 1000000)},
		"revision zero": {alter(rd(1, "1", 1), func(r *Reading) { r.Revision = 0 })}, "a revision above the maximum": {alter(rd(1, "1", 1), func(r *Reading) { r.Revision = 1000000 })},
		"a negative value": {rd(1, "-1", 1)}, "no value": {rd(1, "", 1)}, "a value with an exponent": {rd(1, "1e3", 1)}, "a value of seven places": {rd(1, "0.0000001", 1)}, "a malformed reading after a good one": {rd(1, "1", 1), rd(2, "x", 1)},
	} {
		wantErr(t, put(batch...), ErrInvalid, name)
		_, err := h.r.PutReadings("ws1", "vc-1", "c9", batch)
		wantErr(t, err, ErrInvalid, name+" before the counter is looked up")
	}
	eleven := make([]Reading, 11)
	for index := range eleven {
		eleven[index] = rd(index+1, "1", 1)
	}
	wantErr(t, put(eleven...), ErrLimit, "a batch above the limit")
	for _, c := range []struct{ ws, source, id string }{{"ws2", "vc-1", "c1"}, {"ws1", "vc-2", "c1"}, {"ws1", "vc-1", "c9"}} {
		_, err := h.r.PutReadings(c.ws, c.source, c.id, eleven)
		wantErr(t, err, ErrNotFound, "an unknown counter comes before the size of the batch")
	}
	for name, batch := range map[string][]Reading{
		"before the creation": {rd(-1, "1", 1)}, "after the deletion": {{At: at(100).Add(time.Second), Value: "1", Generation: 1, Revision: 1}}, "one of two outside": {rd(1, "1", 1), rd(101, "2", 1)},
		"generations falling within a batch": {rd(1, "1", 2), rd(2, "2", 1)}, "generations falling in a batch given in another order": {rd(2, "2", 1), rd(1, "1", 2)},
	} {
		before := h.r.dump()
		wantErr(t, put(batch...), ErrRejected, name)
		h.unchanged(before, name)
	}
	states, err := h.r.PutReadings("ws1", "vc-1", "c1", []Reading{rd(100, "9", 3), rd(0, "1", 1), alter(rd(50, "5.50", 2), func(r *Reading) { r.At = r.At.In(time.FixedZone("kst", 9*3600)) })})
	if err != nil || fmt.Sprint(states) != "[live live live]" {
		t.Fatalf("readings on the creation and on the deletion, in any order: %v %v", states, err)
	}
	h.clock.advance(time.Hour)
	before = h.r.dump()
	states, err = h.r.PutReadings("ws1", "vc-1", "c1", []Reading{rd(50, "5.50", 2), rd(0, "1", 1)})
	if err != nil || fmt.Sprint(states) != "[live live]" {
		t.Fatal(states, err)
	}
	for name, reading := range map[string]Reading{"the value": rd(50, "6", 2), "the spelling of the value": rd(50, "5.5", 2), "the generation": rd(50, "5.50", 3)} {
		wantErr(t, put(reading), ErrConflict, name)
		wantErr(t, put(rd(-5, "1", 1), reading), ErrConflict, name+": a conflict comes before a rejection of another reading")
		wantErr(t, put(rd(20, "1", 3), reading), ErrConflict, name+": and before the order of generations")
	}
	for name, batch := range map[string][]Reading{
		"a generation below an earlier one": {rd(60, "1", 1)}, "a generation above a later one": {rd(40, "1", 3)}, "a generation above a later one at the start": {rd(1, "1", 2), rd(2, "1", 3)},
		"a new reading beside a repeat": {rd(50, "5.50", 2), rd(60, "1", 1)},
	} {
		wantErr(t, put(batch...), ErrRejected, name)
	}
	h.unchanged(before, "repeats, conflicts and rejections")
	if err := put(rd(40, "4", 1), rd(45, "4.5", 2), rd(60, "6", 2), rd(70, "7", 3)); err != nil {
		t.Fatal(err)
	}
	page, err := h.r.Readings("ws1", "vc-1", "c1", time.Time{}, 4)
	if err != nil || len(page.Items) != 4 || !page.More || page.Items[0] != rd(0, "1", 1) || page.Items[3] != rd(50, "5.50", 2) || page.Items[3].At.Location() != time.UTC {
		t.Fatalf("%+v %v", page, err)
	}
	page, err = h.r.Readings("ws1", "vc-1", "c1", at(50), 4)
	if err != nil || len(page.Items) != 3 || page.More || page.Items[0] != rd(60, "6", 2) || page.Items[2] != rd(100, "9", 3) {
		t.Fatalf("%+v %v", page, err)
	}
	revised := func(hours int, value string, generation, revision int) Reading {
		return alter(rd(hours, value, generation), func(r *Reading) { r.Revision = revision })
	}
	before = h.r.dump()
	wantErr(t, put(revised(50, "5", 1, 2)), ErrRejected, "a revision whose generation falls below an earlier reading")
	wantErr(t, put(revised(50, "5", 3, 2)), ErrRejected, "a revision whose generation rises above a later reading")
	h.unchanged(before, "a refused revision")
	if err := put(revised(50, "5.75", 2, 3)); err != nil {
		t.Fatal(err)
	}
	wantErr(t, put(revised(50, "5.75", 2, 2)), ErrRejected, "a stale revision")
	wantErr(t, put(rd(50, "5.50", 2)), ErrRejected, "the first revision again")
	wantErr(t, put(revised(50, "5.8", 2, 3)), ErrConflict, "the stored revision with another value")
	wantErr(t, put(revised(50, "5.75", 2, 2), revised(60, "9", 2, 1)), ErrConflict, "a conflict beside a stale revision")
	if err := put(revised(45, "9", 1, 2), revised(47, "9.5", 1, 1), revised(50, "5", 2, 4)); err != nil {
		t.Fatalf("the order of generations is judged on what the batch leaves: %v", err)
	}
	void := func(hours, revision int, reason VoidReason) (State, error) {
		return h.r.VoidReading("ws1", "vc-1", "c1", at(hours), revision, reason)
	}
	for name, call := range map[string]func() (State, error){
		"no reason":     func() (State, error) { return void(50, 4, "") },
		"revision zero": func() (State, error) { return void(50, 0, ByOperator) },
		"an instant with a fraction": func() (State, error) {
			return h.r.VoidReading("ws1", "vc-1", "c1", at(50).Add(time.Millisecond), 4, ByOperator)
		},
	} {
		_, err := call()
		wantErr(t, err, ErrInvalid, name)
	}
	_, err = void(51, 1, ByOperator)
	wantErr(t, err, ErrNotFound, "an instant without a reading")
	_, err = h.r.VoidReading("ws1", "vc-1", "c9", at(50), 4, ByOperator)
	wantErr(t, err, ErrNotFound, "a counter that does not exist")
	_, err = h.r.VoidReading("ws2", "vc-1", "c1", at(50), 4, ByOperator)
	wantErr(t, err, ErrNotFound, "another workspace")
	before = h.r.dump()
	_, err = void(50, 3, ByOperator)
	wantErr(t, err, ErrRejected, "a late retry of a void of an earlier revision")
	_, err = void(50, 5, ByOperator)
	wantErr(t, err, ErrRejected, "a revision that is not stored yet")
	h.unchanged(before, "a refused void")
	if state, err := void(50, 4, WrongReport); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	before = h.r.dump()
	if state, err := void(50, 4, WrongReport); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	_, err = void(50, 4, ByOperator)
	wantErr(t, err, ErrConflict, "a second reason")
	states, err = h.r.PutReadings("ws1", "vc-1", "c1", []Reading{revised(50, "5", 2, 4), rd(0, "1", 1)})
	if err != nil || fmt.Sprint(states) != "[voided live]" {
		t.Fatalf("a repeat of a voided reading says so: %v %v", states, err)
	}
	wantErr(t, put(revised(50, "5", 2, 3)), ErrRejected, "a stale revision of a voided reading")
	h.unchanged(before, "a voided reading")
	page, err = h.r.Readings("ws1", "vc-1", "c1", at(47), 1)
	if err != nil || len(page.Items) != 1 || page.Items[0] != rd(60, "6", 2) {
		t.Fatalf("a voided reading is not listed: %+v %v", page, err)
	}
	if err := put(revised(50, "5.25", 2, 5)); err != nil {
		t.Fatalf("a higher revision makes a voided reading live again: %v", err)
	}
	if state, err := void(50, 5, ByOperator); err != nil || state != Voided {
		t.Fatalf("a reading made live again can be voided for any reason: %v %v", state, err)
	}
	if err := put(revised(50, "5.25", 3, 6)); err != ErrRejected {
		t.Fatalf("a revival is held to the order of generations: %v", err)
	}
	if state, err := h.r.VoidCounter("ws1", "vc-1", "c1", "unknown"); err != ErrInvalid || state != "" {
		t.Fatal(state, err)
	}
	_, err = h.r.VoidCounter("ws1", "vc-1", "c9", ByOperator)
	wantErr(t, err, ErrNotFound, "a void of a counter that does not exist")
	if state, err := h.r.VoidCounter("ws1", "vc-1", "c1", WithdrawnAtSource); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	before = h.r.dump()
	if state, err := h.r.VoidCounter("ws1", "vc-1", "c1", WithdrawnAtSource); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	_, err = h.r.VoidCounter("ws1", "vc-1", "c1", ByOperator)
	wantErr(t, err, ErrConflict, "a second reason for a counter")
	if state, err := h.r.PutCounter("ws1", "vc-1", good); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	states, err = h.r.PutReadings("ws1", "vc-1", "c1", []Reading{rd(0, "1", 1), revised(50, "5.25", 2, 5)})
	if err != nil || fmt.Sprint(states) != "[live voided]" {
		t.Fatalf("readings of a voided counter keep their own state: %v %v", states, err)
	}
	wantErr(t, put(rd(80, "8", 3)), ErrRejected, "a new reading for a voided counter")
	wantErr(t, put(revised(0, "1", 1, 2)), ErrRejected, "a revision for a voided counter")
	_, err = void(0, 1, ByOperator)
	wantErr(t, err, ErrRejected, "a void of a reading of a voided counter")
	if state, err := void(50, 5, ByOperator); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	_, err = h.r.Readings("ws1", "vc-1", "c1", time.Time{}, 1)
	wantErr(t, err, ErrRejected, "the readings of a voided counter")
	h.unchanged(before, "a voided counter")
	counters, err := h.r.Counters("ws1", "vc-1", "vm-1", "", 4)
	if err != nil || len(counters.Items) != 1 || counters.Items[0] != (CounterView{Counter: good, State: Voided}) {
		t.Fatalf("%+v %v", counters, err)
	}
	joined := newHarness(t, nil)
	joined.asset(machine("vm-1"))
	joined.counter("c", "vm-1", "Wh")
	joined.read("c", rd(0, "1", 11))
	_, err = joined.r.PutReadings("ws1", "vc-1", "c", []Reading{rd(0, "11", 1)})
	wantErr(t, err, ErrConflict, "a digit that moved from the generation to the value")
	limited := newHarness(t, func(l *Limits) { l.Counters, l.Readings, l.Batch = 2, 3, 3 })
	limited.asset(machine("vm-1"))
	limited.counter("a", "vm-1", "Wh")
	limited.counter("b", "vm-1", "Wh")
	_, err = limited.r.PutCounter("ws1", "vc-1", Counter{ID: "c", Asset: "vm-1", Unit: "Wh", MaxGap: 1})
	wantErr(t, err, ErrLimit, "a third counter")
	limited.read("a", rd(0, "1", 1), rd(1, "2", 1))
	if _, err := limited.r.VoidReading("ws1", "vc-1", "a", at(1), 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	limited.read("a", rd(2, "3", 1), alter(rd(1, "2", 1), func(r *Reading) { r.Revision = 2 }), rd(0, "1", 1))
	_, err = limited.r.PutReadings("ws1", "vc-1", "a", []Reading{rd(3, "4", 1)})
	wantErr(t, err, ErrLimit, "a fourth reading")
	_, err = limited.r.PutReadings("ws1", "vc-1", "a", []Reading{rd(3, "4", 1), rd(-999, "4", 1)})
	wantErr(t, err, ErrRejected, "rejected before no room")
	limited.read("a", alter(rd(2, "3.5", 1), func(r *Reading) { r.Revision = 2 }))
}
