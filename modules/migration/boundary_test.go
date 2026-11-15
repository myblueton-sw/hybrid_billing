package migration

import (
	"strings"
	"testing"
)

func TestPrimitives(t *testing.T) {
	for _, id := range []string{"a", "Z", "z", "9", "0", "A.z_0:9-", strings.Repeat("x", maxIDBytes)} {
		if !validID(id) {
			t.Errorf("%q refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("x", maxIDBytes+1), "a b", " a", "[", "`", "{", "@", "/"} {
		if validID(id) {
			t.Errorf("%q accepted", id)
		}
	}
	for _, text := range []string{"a", strings.Repeat("x", maxTextBytes), " ", "~"} {
		if !validText(text) {
			t.Errorf("%q refused", text)
		}
	}
	for _, text := range []string{"", strings.Repeat("x", maxTextBytes+1), "a\nb", "\x1f", "\x7f", "\u0080", "\u009f", "\xff", "\u0085"} {
		if validText(text) {
			t.Errorf("%q accepted", text)
		}
	}
	for _, text := range []string{"\x01ab", "\x7fab"} {
		if printable(text, 64) {
			t.Errorf("%q printable", text)
		}
	}
	if currency("kRW") || currency("KR1") {
		t.Error("currency")
	}
	if ascending([]string{"a", "a"}, 4) {
		t.Error("duplicate workspaces")
	}
	if !strings.Contains(doc("a", 10).code(), "\x00\x00\x00\x00\x00\x00\x00\x0210") {
		t.Error("decimal number")
	}
}

func TestFormEdges(t *testing.T) {
	f := standard(t)
	for name, m := range map[string]Mapping{
		"source":    {SourceSeries: "a", Workspace: "ws1", Series: "s"},
		"series":    {Source: "src-1", Workspace: "ws1", Series: "s"},
		"workspace": {Source: "src-1", SourceSeries: "a", Series: "s"},
		"target":    {Source: "src-1", SourceSeries: "a", Workspace: "ws1"},
	} {
		expect(t, name, f.try(Change{Kind: ChangeMapping, Mapping: m}), ErrInvalid)
	}
	f.approve(Change{Kind: ChangeBatch, Batch: BatchSpec{Source: "src-1", ID: "full", Count: maxItems, Digest: sum("x")}})
	expect(t, "reject form", f.r.Reject("src-1", "", "x", ann), ErrInvalid)
	expect(t, "reject source form", f.r.Reject("", "full", "x", ann), ErrInvalid)
	if v, err := f.r.HighWater("t1", "", "inv"); err == nil || v != 0 {
		t.Error("high water form")
	}
	if v, err := f.r.HighWater("t9", "ws1", "inv"); err == nil || v != 0 {
		t.Error("high water tenant")
	}
	expect(t, "printed long", second(f.r.PrintedTaken("t1", "ws1", strings.Repeat("x", 65))), ErrInvalid)
	f.must(second(f.r.PrintedTaken("t1", "ws1", strings.Repeat("x", 64))))
}

func TestCapsExactFit(t *testing.T) {
	type setter func(f world, need int)
	cases := map[string]setter{
		"history":        func(f world, need int) { f.r.tenant("t1").history = maxHistory - 3 },
		"history total":  func(f world, need int) { f.r.history = maxHistoryTotal - 3 },
		"keys":           func(f world, need int) { f.r.tenant("t1").keys = maxKeyRefs - 1 },
		"quarantine":     func(f world, need int) { f.r.sources["src-1"].quarantine = make([]Classified, maxQuarantine-1) },
		"series":         func(f world, need int) { fill(f.r.tenant("t1").high, maxSeries-1) },
		"series total":   func(f world, need int) { f.r.series = maxSeriesTotal - 1 },
		"printed":        func(f world, need int) { fillBool(f.r.tenant("t1").printed, maxPrinted-3) },
		"printed total":  func(f world, need int) { f.r.printed = maxPrintedTotal - 3 },
		"delivery":       func(f world, need int) { fillBool(f.r.tenant("t1").delivery, maxDelivery-1) },
		"delivery total": func(f world, need int) { f.r.delivery = maxDeliveryTotal - 1 },
	}
	pub, _, _ := publicDER(t)
	for name, prepare := range cases {
		f := standard(t)
		f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-1", SourceSeries: "LEGACY2", Workspace: "ws1", Series: "inv2"}})
		f.accept("src-1", "t1", docs(doc("a", 1)))
		changed := doc("a", 2)
		changed.Display = "INV-1"
		d := doc("b", 3)
		d.Delivered = true
		fresh := doc("e", 4)
		fresh.SourceSeries = "LEGACY2"
		c := doc("c", 5)
		items := append(docs(changed, d, fresh, c), Item{Key: &KeyReference{SourceID: "k", Kind: KeyPublic, DER: pub}})
		id, v := f.batch("src-1", items, "t1")
		prepare(f, 0)
		if err := f.try(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: id, Effect: v.Effect}}); err != nil {
			t.Errorf("%s exact fit refused: %v", name, err)
		}
	}
	g := standard(t)
	g.accept("src-1", "t1", docs(doc("a", 1)))
	fill(g.r.tenant("t1").high, maxSeries)
	g.accept("src-1", "t1", docs(doc("b", 2)))
	if g.r.series != 1 || g.r.printed != 2 || g.r.delivery != 0 {
		t.Errorf("counters %d %d %d", g.r.series, g.r.printed, g.r.delivery)
	}
	h := standard(t)
	d := doc("a", 1)
	d.Delivered = true
	h.accept("src-1", "t1", docs(d))
	if h.r.delivery != 1 {
		t.Errorf("delivery counter %d", h.r.delivery)
	}
}
