package normalize

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func cellText(h *harness, id, field string) string {
	h.t.Helper()
	rows := h.rows(id)
	if len(rows) != 1 {
		h.t.Fatalf("%d normal rows, findings %s", len(rows), h.findings(id))
	}
	cell := rows[0].Cells[field]
	return string(cell.State) + ":" + cell.Text
}

func TestNullNotProvidedAndZeroStayApart(t *testing.T) {
	h := newHarness(t, nil)
	rows := []Row{
		good("zero", "0", "USD", "EffectiveCost", "0.00", "ListCost", "-0"),
		good("null", "1", "USD", "EffectiveCost", null, "ListCost", null),
		without(good("gone", "1", "USD"), "EffectiveCost", "ListCost"),
	}
	view := h.one(fullEntries(), nil, rows...)
	if counts(view) != "3=3+0+0" {
		t.Fatal(counts(view), h.findings("run-1"))
	}
	got := h.rows("run-1")
	want := [][2]OutCell{
		{{State: Value, Text: "0"}, {State: Value, Text: "0"}},
		{{State: Null}, {State: Null}},
		{{State: NotProvided}, {State: NotProvided}},
	}
	for index, pair := range want {
		if got[index].Cells["effective"] != pair[0] || got[index].Cells["list"] != pair[1] || got[index].Cells["note"] != (OutCell{State: Unmapped}) {
			t.Fatalf("row %d: %+v", index, got[index].Cells)
		}
	}
	if !reflect.DeepEqual(view.Sums, map[string]map[string]string{"billed": {"USD": "2"}, "effective": {"USD": "0"}, "list": {"USD": "0"}}) {
		t.Fatalf("%v", view.Sums)
	}
	for _, change := range []struct {
		transform Transform
		power     int
	}{{Negate, 0}, {Scale, 3}, {Scale, -3}} {
		other := newHarness(t, nil)
		entries := fullEntries()
		entries[1].Transform, entries[1].Power = change.transform, change.power
		entries[3].Transform, entries[3].Power = change.transform, change.power
		other.one(entries, nil, good("zero", "1", "USD", "EffectiveCost", "0", "Usage", "0"), good("null", "1", "USD", "EffectiveCost", null), without(good("gone", "1", "USD"), "EffectiveCost"))
		made := other.rows("run-1")
		if made[0].Cells["effective"] != (OutCell{State: Value, Text: "0"}) || made[0].Cells["qty"] != (OutCell{State: Value, Text: "0"}) || made[1].Cells["effective"] != (OutCell{State: Null}) || made[2].Cells["effective"] != (OutCell{State: NotProvided}) {
			t.Fatalf("%v: %+v", change, made)
		}
	}
}

func TestDecimalsAreKeptExactly(t *testing.T) {
	cases := []struct {
		text      string
		transform Transform
		power     int
		want      string
	}{
		{"1.50", Copy, 0, "value:1.5"},
		{"-0", Copy, 0, "value:0"},
		{"-0.0", Negate, 0, "value:0"},
		{"999999999999.9999", Copy, 0, "value:999999999999.9999"},
		{"-999999999999.9999", Negate, 0, "value:999999999999.9999"},
		{"0.0001", Copy, 0, "value:0.0001"},
		{"0.1", Negate, 0, "value:-0.1"},
		{"0.30", Scale, -1, "value:0.03"},
		{"1234500", Scale, -6, "value:1.2345"},
		{"0.0001", Scale, -2, "value:0.000001"},
		{"0.0001", Scale, -3, OutsideProfile},
		{"100", Scale, -12, OutsideProfile},
		{"1000000", Scale, -12, "value:0.000001"},
		{"1.5", Scale, 3, "value:1500"},
		{"0.0001", Scale, 12, "value:100000000"},
		{"99999999999999", Copy, 0, OutsideProfile},
		{"999999999999", Scale, 2, "value:99999999999900"},
		{"999999999999", Scale, 3, OutsideProfile},
		{"0.12345", Copy, 0, OutsideProfile},
		{"1000000000000", Copy, 0, OutsideProfile},
		{"12345678901234567890123456789012345678901234567890xx", Copy, 0, OutsideProfile},
		{"1e3", Copy, 0, NotDecimal},
		{"1,5", Copy, 0, NotDecimal},
		{"abc", Negate, 0, NotDecimal},
		{" 1", Copy, 0, NotDecimal},
		{"", Copy, 0, EmptyText},
	}
	for _, c := range cases {
		for _, source := range []string{"EffectiveCost", "Usage"} {
			h := newHarness(t, nil)
			entries := fullEntries()
			index, field := 1, "effective"
			if source == "Usage" {
				index, field = 3, "qty"
			}
			entries[index].Transform, entries[index].Power = c.transform, c.power
			view := h.one(entries, nil, good("k", "1", "USD", source, c.text))
			if strings.HasPrefix(c.want, "value:") {
				if got := cellText(h, "run-1", field); got != c.want {
					t.Fatalf("%q %s %d: %s, want %s", c.text, c.transform, c.power, got, c.want)
				}
				continue
			}
			if got := h.findings("run-1"); got != "1.1/k/"+field+":"+c.want || counts(view) != "1=0+1+0" {
				t.Fatalf("%q %s %d: %s %s", c.text, c.transform, c.power, got, counts(view))
			}
		}
	}
}

func TestTimesCurrenciesAndTexts(t *testing.T) {
	cases := []struct{ source, text, field, want string }{
		{"PeriodStart", "2026-10-05T09:00:00+09:00", "start", "value:2026-10-05T00:00:00Z"},
		{"PeriodStart", "2026-10-04T19:30:00.500-04:30", "start", "value:2026-10-05T00:00:00.5Z"},
		{"PeriodStart", "2026-10-05T00:00:00.000000000Z", "start", "value:2026-10-05T00:00:00Z"},
		{"PeriodStart", "2026-10-05T00:00:00.123456789Z", "start", "value:2026-10-05T00:00:00.123456789Z"},
		{"PeriodStart", "2026-10-05T00:00:00.1234567891Z", "start", NotTime},
		{"PeriodStart", "2026-10-05t00:00:00Z", "start", NotTime},
		{"PeriodStart", "2026-10-05T23:59:00+23:59", "start", "value:2026-10-05T00:00:00Z"},
		{"PeriodStart", "2026-10-05T00:00:00+24:00", "start", NotTime},
		{"PeriodStart", "2026-10-05T00:00:00+09:60", "start", NotTime},
		{"PeriodStart", "2026-10-05T00:00:00-31:00", "start", NotTime},
		{"PeriodStart", "0000-01-01T00:00:00+01:00", "start", NotTime},
		{"PeriodStart", "0000-01-01T00:00:00Z", "start", "value:0000-01-01T00:00:00Z"},
		{"PeriodEnd", "9999-12-31T23:59:59-01:00", "end", NotTime},
		{"PeriodEnd", "9999-12-31T23:59:59Z", "end", "value:9999-12-31T23:59:59Z"},
		{"PeriodStart", "2026-10-05T00:00:00z", "start", NotTime},
		{"PeriodStart", "2026-10-05T00:00:60Z", "start", NotTime},
		{"PeriodStart", "2026-10-05 00:00:00Z", "start", NotTime},
		{"PeriodStart", "2026-10-05T00:00:00", "start", NotTime},
		{"PeriodStart", "2026-10-05", "start", NotTime},
		{"PeriodStart", "2026-13-05T00:00:00Z", "start", NotTime},
		{"PeriodStart", "2026-10-05T00:00:00.Z", "start", NotTime},
		{"PeriodStart", "", "start", EmptyText},
		{"BillingCurrency", "KRW", "cur", "value:KRW"},
		{"BillingCurrency", "usd", "cur", NotCurrency},
		{"BillingCurrency", "USDT", "cur", NotCurrency},
		{"BillingCurrency", "U1D", "cur", NotCurrency},
		{"BillingCurrency", "", "cur", EmptyText},
		{"ServiceName", "", "svc", "value:"},
		{"ServiceName", "  Compute Engine, 1.50 ", "svc", "value:  Compute Engine, 1.50 "},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		view := h.one(fullEntries(), func(s *RunSpec) { s.Window = nil }, good("k", "1", "USD", c.source, c.text))
		if strings.HasPrefix(c.want, "value:") {
			if got := cellText(h, "run-1", c.field); got != c.want {
				t.Fatalf("%q: %s, want %s", c.text, got, c.want)
			}
			continue
		}
		if got := h.findings("run-1"); got != "1.1/k/"+c.field+":"+c.want || counts(view) != "1=0+1+0" {
			t.Fatalf("%q: %s", c.text, got)
		}
	}
}

func TestFindingsPerRowAndField(t *testing.T) {
	cases := []struct {
		name string
		row  Row
		want string
	}{
		{"required null", good("k", "1", "USD", "ServiceName", null), "svc:required_null"},
		{"required not provided", without(good("k", "1", "USD"), "ServiceName"), "svc:required_not_provided"},
		{"required and invalid gives one finding", good("k", "x", "USD"), "billed:not_decimal"},
		{"optional null and not provided", without(good("k", "1", "USD", "EffectiveCost", null), "ListCost"), ""},
		{"currency missing once for three costs", good("k", "1", "USD", "BillingCurrency", null), "cur:required_null cur:currency_missing"},
		{"currency not provided", without(good("k", "1", "USD"), "BillingCurrency"), "cur:required_not_provided cur:currency_missing"},
		{"an invalid currency is not a missing one", good("k", "1", "usd"), "cur:not_currency"},
		{"no cost, no currency needed beyond the required", without(good("k", "1", "USD", "BilledCost", null, "BillingCurrency", null), "EffectiveCost", "ListCost"), "billed:required_null cur:required_null"},
		{"an invalid cost still needs its currency", without(good("k", "abc", "USD", "BillingCurrency", null), "EffectiveCost", "ListCost"), "billed:not_decimal cur:required_null cur:currency_missing"},
		{"start missing", good("k", "1", "USD", "PeriodStart", null), "start:period_missing"},
		{"end not provided", without(good("k", "1", "USD"), "PeriodEnd"), "end:period_missing"},
		{"both missing", without(good("k", "1", "USD"), "PeriodEnd", "PeriodStart"), "start:period_missing end:period_missing"},
		{"an invalid start is not a missing one", good("k", "1", "USD", "PeriodStart", "soon"), "start:not_time"},
		{"inverted", good("k", "1", "USD", "PeriodStart", "2026-10-07T00:00:00Z"), "start:period_inverted"},
		{"start equal to end", good("k", "1", "USD", "PeriodStart", "2026-10-06T00:00:00Z"), "start:period_inverted"},
		{"start equal to end at another offset", good("k", "1", "USD", "PeriodStart", "2026-10-06T09:00:00+09:00"), "start:period_inverted"},
		{"at the window's start", good("k", "1", "USD", "PeriodStart", "2026-10-01T00:00:00Z"), ""},
		{"before the window", good("k", "1", "USD", "PeriodStart", "2026-09-30T23:59:59.999999999Z"), "start:outside_window"},
		{"at the window's end", good("k", "1", "USD", "PeriodEnd", "2026-11-01T00:00:00Z"), ""},
		{"after the window", good("k", "1", "USD", "PeriodEnd", "2026-11-01T00:00:00.000000001Z"), "start:outside_window"},
		{"straddling both", good("k", "1", "USD", "PeriodStart", "2026-09-01T00:00:00Z", "PeriodEnd", "2026-12-01T00:00:00Z"), "start:outside_window"},
		{"inverted and outside", good("k", "1", "USD", "PeriodStart", "2026-12-01T00:00:00Z", "PeriodEnd", "2026-11-15T00:00:00Z"), "start:period_inverted start:outside_window"},
		{"inverted inside the window", good("k", "1", "USD", "PeriodStart", "2026-10-20T00:00:00Z", "PeriodEnd", "2026-10-10T00:00:00Z"), "start:period_inverted"},
		{"the currency before later fields", good("k", "1", "USD", "BillingCurrency", null, "ServiceName", null), "cur:required_null cur:currency_missing svc:required_null"},
		{"the period before later fields", good("k", "1", "USD", "PeriodStart", null, "PeriodEnd", "2026-12-01T00:00:00Z", "ServiceName", null), "start:period_missing svc:required_null"},
		{"several fields in target order", good("k", "1", "USD", "ServiceName", null, "Usage", "", "EffectiveCost", "1e1", "PeriodEnd", "x"), "effective:not_decimal qty:empty end:not_time svc:required_null"},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		view := h.one(fullEntries(), nil, c.row)
		got := strings.ReplaceAll(h.findings("run-1"), "1.1/k/", "")
		if got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
		wantCounts, total := "1=1+0+0", 0
		if c.want != "" {
			wantCounts, total = "1=0+1+0", len(strings.Fields(c.want))
		}
		if counts(view) != wantCounts || view.FindingsTotal != total {
			t.Fatalf("%s: %s %d", c.name, counts(view), view.FindingsTotal)
		}
	}
	h := newHarness(t, nil)
	view := h.one(fullEntries(), func(s *RunSpec) { s.Window = nil },
		without(good("a", "1", "USD"), "PeriodStart", "PeriodEnd"), good("b", "1", "USD", "PeriodStart", null),
		good("c", "1", "USD", "PeriodStart", "2020-01-01T00:00:00Z", "PeriodEnd", "2030-01-01T00:00:00Z"), good("d", "1", "USD", "PeriodStart", "2026-10-06T00:00:00Z"))
	if got := h.findings("run-1"); got != "1.4/d/start:period_inverted" || counts(view) != "4=3+1+0" {
		t.Fatalf("without a window only the order of the period is checked: %s", got)
	}
	unmapped := newHarness(t, nil)
	entries := fullEntries()
	entries[5], entries[6], entries[1] = missing("start", NotSupported), missing("end", NotSupported), missing("effective", NotSupported)
	view = unmapped.one(entries, func(s *RunSpec) { s.Window = nil }, good("a", "1", "USD", "PeriodStart", "x", "EffectiveCost", "x"))
	if counts(view) != "1=1+0+0" || !reflect.DeepEqual(view.Unused, Listed{Total: 3, Items: []string{"EffectiveCost", "PeriodEnd", "PeriodStart"}}) {
		t.Fatalf("fields that are not mapped are not checked: %s %+v", counts(view), view.Unused)
	}
}

func TestDuplicateKeysQuarantineEveryRowThatHasThem(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(fullEntries())
	h.open(spec("r1", 3, 9))
	bad := good("b", "x", "USD")
	h.check("r1", 1, good("a", "1", "USD"), bad, good("c", "1", "USD"), good("c", "1", "USD"))
	if view := h.view("r1"); counts(view) != "4=1+1+2" {
		t.Fatal(counts(view))
	}
	h.check("r1", 3, good("d", "4", "USD"), good("b", "2", "USD"), Row{Key: RowKey{Kind: LocatorKeys, Document: "doc1", Key: "a"}, Cells: good("a", "8", "USD").Cells})
	if view := h.view("r1"); counts(view) != "7=3+0+4" {
		t.Fatal(counts(view))
	}
	h.check("r1", 2, good("c", "1", "USD"), good("e", "y", "EUR"))
	view := h.view("r1")
	if counts(view) != "9=3+1+5" || view.State != Open || view.Format != nil || view.Billing != nil || view.Sums != nil {
		t.Fatalf("%s %+v", counts(view), view)
	}
	view = h.close("r1")
	if counts(view) != "9=3+1+5" || view.FindingsTotal != 1 || h.findings("r1") != "2.2/e/billed:not_decimal" {
		t.Fatalf("%s %s", counts(view), h.findings("r1"))
	}
	if verdicts(view) != "nonconformant[rows_in_error rows_quarantined] not_ready[rows_in_error rows_quarantined totals_not_expected]" {
		t.Fatal(verdicts(view))
	}
	apart, err := h.r.Quarantined("ws1", "r1", 0, 100)
	want := []QuarantinedRow{{1, 2, ev("b")}, {1, 3, ev("c")}, {1, 4, ev("c")}, {2, 1, ev("c")}, {3, 2, ev("b")}}
	if err != nil || !reflect.DeepEqual(apart, want) {
		t.Fatalf("%+v %v", apart, err)
	}
	rows := h.rows("r1")
	if len(rows) != 3 || rows[0].Key != ev("a") || rows[1].Key != ev("d") || rows[1].Chunk != 3 || rows[1].Position != 1 || rows[2].Key.Document != "doc1" {
		t.Fatalf("%+v", rows)
	}
	if !reflect.DeepEqual(view.Sums["billed"], map[string]string{"USD": "13"}) {
		t.Fatalf("a locator and an event key of the same text are two rows: %v", view.Sums)
	}
	keys := newHarness(t, nil)
	keys.declare(fullEntries())
	keys.open(spec("r1", 1, 4))
	locator := func(document, key string) Row {
		return Row{Key: RowKey{Kind: LocatorKeys, Document: document, Key: key}, Cells: good(key, "1", "USD").Cells}
	}
	keys.check("r1", 1, locator("d1", "r1"), locator("d2", "r1"), locator("d1", "r2"), good("r1", "1", "USD"))
	if view := keys.view("r1"); counts(view) != "4=4+0+0" {
		t.Fatal(counts(view))
	}
}

func TestMalformedChunksAreRefusedWhole(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.ChunkRows = 3; l.Cells = 9; l.Text = 20 })
	h.declare(fullEntries())
	h.open(spec("r1", 2, 4))
	long := strings.Repeat("x", 21)
	nine := good("k", "1", "USD", "Extra", "1")
	cases := []struct {
		name   string
		ws, id string
		chunk  int
		rows   []Row
		want   error
	}{
		{"chunk zero", "ws1", "r1", 0, []Row{good("k", "1", "USD")}, ErrInvalid},
		{"chunk negative", "ws1", "r1", -1, []Row{good("k", "1", "USD")}, ErrInvalid},
		{"chunk over the count", "ws1", "r1", 3, []Row{good("k", "1", "USD")}, ErrInvalid},
		{"no row", "ws1", "r1", 1, nil, ErrInvalid},
		{"too many rows", "ws1", "r1", 1, []Row{good("a", "1", "USD"), good("b", "1", "USD"), good("c", "1", "USD"), good("d", "1", "USD")}, ErrInvalid},
		{"key", "ws1", "r1", 1, []Row{good("a b", "1", "USD")}, ErrInvalid},
		{"key empty", "ws1", "r1", 1, []Row{good("", "1", "USD")}, ErrInvalid},
		{"key kind", "ws1", "r1", 1, []Row{{Key: RowKey{Kind: "row", Key: "a"}, Cells: good("a", "1", "USD").Cells}}, ErrInvalid},
		{"event key with a document", "ws1", "r1", 1, []Row{{Key: RowKey{Kind: EventKeys, Document: "d1", Key: "a"}, Cells: good("a", "1", "USD").Cells}}, ErrInvalid},
		{"locator without a document", "ws1", "r1", 1, []Row{{Key: RowKey{Kind: LocatorKeys, Key: "a"}, Cells: good("a", "1", "USD").Cells}}, ErrInvalid},
		{"locator with a malformed document", "ws1", "r1", 1, []Row{{Key: RowKey{Kind: LocatorKeys, Document: "d 1", Key: "a"}, Cells: good("a", "1", "USD").Cells}}, ErrInvalid},
		{"locator with a malformed key", "ws1", "r1", 1, []Row{{Key: RowKey{Kind: LocatorKeys, Document: "d1", Key: ""}, Cells: good("a", "1", "USD").Cells}}, ErrInvalid},
		{"source field name", "ws1", "r1", 1, []Row{good("a", "1", "USD", "Service Name", "x")}, ErrInvalid},
		{"text too long", "ws1", "r1", 1, []Row{good("a", "1", "USD", "ServiceName", long)}, ErrInvalid},
		{"unused text too long", "ws1", "r1", 1, []Row{without(good("a", "1", "USD", "Extra", long), "ListCost")}, ErrInvalid},
		{"too many cells", "ws1", "r1", 1, []Row{without(good("a", "1", "USD", "Extra", "1", "More", "2"))}, ErrInvalid},
		{"null with a text", "ws1", "r1", 1, []Row{{Key: ev("a"), Cells: map[string]Cell{"BilledCost": {Null: true, Text: "1"}}}}, ErrInvalid},
		{"second row malformed", "ws1", "r1", 1, []Row{good("a", "1", "USD"), good("b c", "1", "USD")}, ErrInvalid},
		{"workspace", "", "r1", 1, []Row{good("k", "1", "USD")}, ErrInvalid},
		{"run id", "ws1", "", 1, []Row{good("k", "1", "USD")}, ErrInvalid},
		{"missing run", "ws1", "r9", 1, []Row{good("k", "1", "USD")}, ErrNotFound},
		{"another workspace", "ws2", "r1", 1, []Row{good("k", "1", "USD")}, ErrNotFound},
	}
	for _, c := range cases {
		before := h.r.dump()
		_, err := h.r.Check(c.ws, c.id, c.chunk, c.rows)
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	h.check("r1", 1, good("a", "1", "USD", "ServiceName", strings.Repeat("x", 20)), nine, good("c", "1", "USD"))
	h.check("r1", 2, Row{Key: ev("d"), Cells: map[string]Cell{}})
	view := h.close("r1")
	if counts(view) != "4=3+1+0" || view.FindingsTotal != 6 {
		t.Fatalf("%s %s", counts(view), h.findings("r1"))
	}
}

func TestChunkRepeatsAndContradictions(t *testing.T) {
	base := []Row{good("a", "1.50", "USD"), good("b", "2", "USD", "EffectiveCost", null)}
	variants := []struct {
		name string
		rows []Row
		same bool
	}{
		{"identical", []Row{good("a", "1.50", "USD"), good("b", "2", "USD", "EffectiveCost", null)}, true},
		{"another spelling of an amount", []Row{good("a", "1.5", "USD"), base[1]}, false},
		{"another order", []Row{base[1], base[0]}, false},
		{"an unused cell more", []Row{good("a", "1.50", "USD", "Extra", "1"), base[1]}, false},
		{"null against not provided", []Row{base[0], without(base[1], "EffectiveCost")}, false},
		{"null against an empty text", []Row{base[0], good("b", "2", "USD", "EffectiveCost", "")}, false},
		{"a row fewer", []Row{base[0]}, false},
		{"another key", []Row{good("z", "1.50", "USD"), base[1]}, false},
		{"a locator of the same text", []Row{{Key: RowKey{Kind: LocatorKeys, Document: "a", Key: "a"}, Cells: base[0].Cells}, base[1]}, false},
		{"a cell moved to another name", []Row{without(good("a", "1.50", "USD", "Usage2", "3"), "Usage"), base[1]}, false},
	}
	for _, v := range variants {
		h := newHarness(t, nil)
		h.declare(fullEntries())
		h.open(spec("r1", 2, 3))
		h.check("r1", 1, base...)
		h.clock.advance(time.Minute)
		before := h.r.dump()
		state, err := h.r.Check("ws1", "r1", 1, v.rows)
		if v.same {
			if err != nil || state != Open {
				t.Fatal(v.name, state, err)
			}
			h.unchanged(before, v.name)
			continue
		}
		if err != nil || state != Inconsistent {
			t.Fatalf("%s: %v %v", v.name, state, err)
		}
		view := h.view("r1")
		if view.State != Inconsistent || view.Contradiction != 1 || !view.Changed.Equal(start.Add(time.Minute)) || !view.Opened.Equal(start) || h.r.workspaces["ws1"].rows != 0 || counts(view) != "2=2+0+0" {
			t.Fatalf("%s: %+v", v.name, view)
		}
		before = h.r.dump()
		if state, err := h.r.Check("ws1", "r1", 1, base); err != nil || state != Inconsistent {
			t.Fatal(v.name, state, err)
		}
		_, err = h.r.Check("ws1", "r1", 1, v.rows)
		wantErr(t, err, ErrRejected, v.name+" sent again")
		_, err = h.r.Check("ws1", "r1", 2, []Row{good("c", "1", "USD")})
		wantErr(t, err, ErrRejected, v.name+" new chunk")
		_, _, err = h.r.Close("ws1", "r1")
		wantErr(t, err, ErrRejected, v.name+" close")
		wantErr(t, h.r.Abandon("ws1", "r1", Superseded), ErrRejected, v.name+" abandon")
		wantErr(t, h.r.Discard("ws1", "r1", WrongInput), ErrRejected, v.name+" discard")
		for _, read := range []func() error{
			func() error { _, err := h.r.Rows("ws1", "r1", 0, 1); return err },
			func() error { _, err := h.r.Findings("ws1", "r1", 0, 1); return err },
			func() error { _, err := h.r.Quarantined("ws1", "r1", 0, 1); return err },
		} {
			wantErr(t, read(), ErrRejected, v.name+" read")
		}
		if state, err := h.r.Open(spec("r1", 2, 3)); err != nil || state != Inconsistent {
			t.Fatal(v.name, state, err)
		}
		h.unchanged(before, v.name)
	}
}

func TestChunkIdentityIsWhatWasSent(t *testing.T) {
	pairs := []struct {
		name          string
		first, second Row
	}{
		{"fields are kept apart by their length", Row{Key: ev("a"), Cells: cells("A", "textZ")}, Row{Key: ev("a"), Cells: cells("Atext", "Z")}},
		{"the document of a locator", Row{Key: RowKey{Kind: LocatorKeys, Document: "d1", Key: "r1"}, Cells: cells("A", "1")}, Row{Key: RowKey{Kind: LocatorKeys, Document: "d2", Key: "r1"}, Cells: cells("A", "1")}},
		{"the key and a cell name", Row{Key: ev("ab"), Cells: cells("c", "1")}, Row{Key: ev("a"), Cells: cells("bc", "1")}},
	}
	for _, pair := range pairs {
		h := newHarness(t, nil)
		h.declare(fullEntries())
		h.open(spec("r1", 1, 1))
		h.check("r1", 1, pair.first)
		if state, err := h.r.Check("ws1", "r1", 1, []Row{pair.second}); err != nil || state != Inconsistent {
			t.Fatalf("%s: %v %v", pair.name, state, err)
		}
	}
	h := newHarness(t, nil)
	h.declare(fullEntries())
	h.open(spec("r1", 1, 1))
	h.check("r1", 1, good("a", "1", "USD"))
	if err := h.r.Abandon("ws1", "r1", Superseded); err != nil {
		t.Fatal(err)
	}
	before := h.r.dump()
	state, reasons, err := h.r.Close("ws1", "r1")
	if err != ErrRejected || state != "" || reasons != nil {
		t.Fatalf("an abandoned run with every chunk checked must not close: %v %v %v", state, reasons, err)
	}
	h.unchanged(before, "closing an abandoned run")
}

func TestTargetWithoutCosts(t *testing.T) {
	h := newHarness(t, nil)
	plain := Target{ID: "plain", Fields: []Field{{Name: "qty", Kind: Quantity, Format: true}, {Name: "svc", Kind: Text}}}
	if err := h.r.DeclareTarget(plain); err != nil {
		t.Fatal(err)
	}
	key := MappingKey{Provider: "aws", SourceVersion: "usage-1", Target: "plain"}
	if _, err := h.r.DeclareMapping(key, 0, []Entry{from("qty", "Usage", Quantity, ""), from("svc", "ServiceName", Text, "")}); err != nil {
		t.Fatal(err)
	}
	s := spec("r1", 1, 2)
	s.Mapping, s.Window = key, nil
	h.open(s)
	h.check("r1", 1, good("a", "1", "USD"), good("b", "1", "USD", "Usage", "2.5"))
	view := h.close("r1")
	if verdicts(view) != "conformant[] not_ready[mapping_gap]" || len(view.Sums) != 0 || counts(view) != "2=2+0+0" {
		t.Fatalf("%s %v", verdicts(view), view.Sums)
	}
}

func TestCloseNeedsEveryChunkAndTheDeclaredRows(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(fullEntries())
	h.open(spec("r1", 2, 3))
	refused := func(want ...string) {
		t.Helper()
		before := h.r.dump()
		state, reasons, err := h.r.Close("ws1", "r1")
		if err != ErrRejected || state != "" || !reflect.DeepEqual(reasons, want) {
			t.Fatal(state, reasons, err)
		}
		h.unchanged(before, "refused close")
	}
	refused(ChunksMissing, RowCountDiffers)
	h.check("r1", 2, good("a", "1", "USD"), good("b", "1", "USD"), good("c", "1", "USD"))
	refused(ChunksMissing)
	for _, read := range []func() error{
		func() error { _, err := h.r.Rows("ws1", "r1", 0, 1); return err },
		func() error { _, err := h.r.Findings("ws1", "r1", 0, 1); return err },
		func() error { _, err := h.r.Quarantined("ws1", "r1", 0, 1); return err },
	} {
		wantErr(t, read(), ErrRejected, "reading an open run")
	}
	h.check("r1", 1, good("d", "1", "USD"))
	refused(RowCountDiffers)
	if err := h.r.Abandon("ws1", "r1", TransferFailed); err != nil {
		t.Fatal(err)
	}
	h.open(spec("r2", 2, 3))
	h.check("r2", 1, good("a", "1", "USD"))
	h.check("r2", 2, good("b", "1", "USD"))
	if _, reasons, err := h.r.Close("ws1", "r2"); err != ErrRejected || !reflect.DeepEqual(reasons, []string{RowCountDiffers}) {
		t.Fatal(reasons, err)
	}
	h.open(spec("r3", 2, 2))
	h.check("r3", 1, good("a", "1", "USD"))
	h.check("r3", 2, good("b", "1", "USD"))
	h.clock.advance(time.Minute)
	view := h.close("r3")
	if !view.Changed.Equal(start.Add(time.Minute)) || !view.Opened.Equal(start) || view.ProfileID != "synthetic" || view.ProfileRevision != "7" {
		t.Fatalf("%+v", view)
	}
	h.clock.advance(time.Minute)
	before := h.r.dump()
	if state, reasons, err := h.r.Close("ws1", "r3"); err != nil || state != Closed || reasons != nil {
		t.Fatal(state, reasons, err)
	}
	if state, err := h.r.Check("ws1", "r3", 2, []Row{good("b", "1", "USD")}); err != nil || state != Closed {
		t.Fatal(state, err)
	}
	_, err := h.r.Check("ws1", "r3", 2, []Row{good("z", "1", "USD")})
	wantErr(t, err, ErrRejected, "another chunk after closing")
	wantErr(t, h.r.Abandon("ws1", "r3", Superseded), ErrRejected, "abandoning a closed run")
	h.unchanged(before, "after closing")
	for _, c := range []struct{ ws, id string }{{"ws2", "r3"}, {"ws1", "r9"}} {
		_, _, err := h.r.Close(c.ws, c.id)
		wantErr(t, err, ErrNotFound, c.id)
	}
	for _, c := range []struct{ ws, id string }{{"", "r3"}, {"ws1", ""}} {
		_, _, err := h.r.Close(c.ws, c.id)
		wantErr(t, err, ErrInvalid, c.id)
	}
}

func TestVerdictsAreJudgedSeparately(t *testing.T) {
	expect := func(sums ...string) func(*RunSpec) {
		return func(s *RunSpec) {
			s.Expected = map[string]map[string]string{"billed": {}}
			for index := 0; index+1 < len(sums); index += 2 {
				s.Expected["billed"][sums[index]] = sums[index+1]
			}
		}
	}
	gap := fullEntries()
	gap[0] = missing("billed", NotSupported)
	lackingTarget := func(h *harness) {
		stated := testTarget()
		stated.Fields[1].Billing = true
		if err := h.r.DeclareTarget(stated); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name    string
		prepare func(*harness)
		entries []Entry
		change  func(*RunSpec)
		rows    []Row
		want    string
	}{
		{"conformant and ready", nil, fullEntries(), expect("USD", "3.5"), []Row{good("a", "1.5", "USD"), good("b", "2", "USD")}, "conformant[] ready[]"},
		{"partial and ready", nil, testEntries(), expect("USD", "3.50"), []Row{good("a", "1.5", "USD"), good("b", "2", "USD")}, "partial[mapping_partial] ready[]"},
		{"conformant and not ready: totals not expected", nil, fullEntries(), nil, []Row{good("a", "1.5", "USD")}, "conformant[] not_ready[totals_not_expected]"},
		{"conformant and not ready: totals differ", nil, fullEntries(), expect("USD", "3.4999"), []Row{good("a", "1.5", "USD"), good("b", "2", "USD")}, "conformant[] not_ready[totals_differ]"},
		{"totals differ by a power of ten", nil, fullEntries(), expect("USD", "35"), []Row{good("a", "1.5", "USD"), good("b", "2", "USD")}, "conformant[] not_ready[totals_differ]"},
		{"an unexpected currency", nil, fullEntries(), expect("USD", "1.5"), []Row{good("a", "1.5", "USD"), good("b", "0", "EUR")}, "conformant[] not_ready[totals_differ]"},
		{"an expected currency without a value", nil, fullEntries(), expect("USD", "1.5", "EUR", "0"), []Row{good("a", "1.5", "USD")}, "conformant[] not_ready[totals_differ]"},
		{"an expected currency in place of another", nil, fullEntries(), expect("EUR", "1.5"), []Row{good("a", "1.5", "USD")}, "conformant[] not_ready[totals_differ]"},
		{"two currencies met, one of them zero", nil, fullEntries(), expect("USD", "1.5", "EUR", "0"), []Row{good("a", "1.5", "USD"), good("b", "2", "EUR"), good("c", "-2", "EUR")}, "conformant[] ready[]"},
		{"window not declared", nil, fullEntries(), func(s *RunSpec) { expect("USD", "1")(s); s.Window = nil }, []Row{good("a", "1", "USD")}, "conformant[] not_ready[window_not_declared]"},
		{"partial and not ready: mapping gap", nil, gap, nil, []Row{good("a", "1", "USD")}, "partial[mapping_partial] not_ready[mapping_gap]"},
		{"nonconformant and not ready: rows in error", nil, fullEntries(), expect("USD", "1"), []Row{good("a", "1", "USD"), good("b", "x", "USD")}, "nonconformant[rows_in_error] not_ready[rows_in_error]"},
		{"errors beside a partial mapping", nil, testEntries(), expect("USD", "1"), []Row{good("a", "1", "USD"), good("b", "x", "USD")}, "nonconformant[mapping_partial rows_in_error] not_ready[rows_in_error]"},
		{"rows quarantined", nil, fullEntries(), expect(), []Row{good("a", "1", "USD"), good("a", "1", "USD")}, "nonconformant[rows_quarantined] not_ready[rows_quarantined]"},
		{"expected sums given with no currency, and no value", nil, fullEntries(), func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {}, "effective": {}} },
			[]Row{good("a", "x", "USD"), good("b", "y", "USD")}, "nonconformant[rows_in_error] not_ready[rows_in_error]"},
		{"expected sums given with no currency, and a value", nil, fullEntries(), expect(), []Row{good("a", "0", "USD")}, "conformant[] not_ready[totals_differ]"},
		{"sums of a field that is not required are compared when given", nil, fullEntries(), func(s *RunSpec) {
			s.Expected = map[string]map[string]string{"billed": {"USD": "1"}, "list": {"USD": "2"}}
		},
			[]Row{good("a", "1", "USD")}, "conformant[] not_ready[totals_differ]"},
		{"a field that is not required needs no expected sum", nil, fullEntries(), expect("USD", "1"), []Row{good("a", "1", "USD", "EffectiveCost", "77")}, "conformant[] ready[]"},
		{"a null cost counts for no currency", nil, fullEntries(), func(s *RunSpec) {
			s.Expected = map[string]map[string]string{"billed": {"USD": "1", "EUR": "1"}, "list": {"USD": "1"}}
		},
			[]Row{good("a", "1", "USD"), good("b", "1", "EUR", "ListCost", null)}, "conformant[] ready[]"},
		{"rows lacking", lackingTarget, fullEntries(), func(s *RunSpec) {
			s.Expected = map[string]map[string]string{"billed": {"USD": "2"}, "effective": {"USD": "1"}}
		},
			[]Row{good("a", "1", "USD"), good("b", "1", "USD", "EffectiveCost", null)}, "conformant[] not_ready[rows_lacking]"},
		{"every reason at once", lackingTarget, testEntries(), func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"USD": "9"}}; s.Window = nil },
			[]Row{good("a", "1", "USD"), without(good("b", "1", "USD"), "EffectiveCost"), good("c", "x", "USD"), good("d", "1", "USD"), good("d", "1", "USD")},
			"nonconformant[mapping_partial rows_in_error rows_quarantined] not_ready[rows_in_error rows_quarantined rows_lacking totals_not_expected totals_differ window_not_declared]"},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		h := newHarness(t, nil)
		if c.prepare != nil {
			c.prepare(h)
		}
		view := h.one(c.entries, c.change, c.rows...)
		if got := verdicts(view); got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s\n%s %v", c.name, got, c.want, h.findings("run-1"), view.Sums)
		}
		seen[view.Format.Result+"+"+view.Billing.Result] = true
		page, err := h.r.Rows("ws1", "run-1", 0, 1)
		if err != nil || page.Format.Result != view.Format.Result || page.Billing.Result != view.Billing.Result || !reflect.DeepEqual(page.Billing.Reasons, view.Billing.Reasons) || !reflect.DeepEqual(page.Format.Reasons, view.Format.Reasons) {
			t.Fatalf("%s: a page of rows must carry both verdicts: %+v %v", c.name, page, err)
		}
		far, _ := h.r.Rows("ws1", "run-1", 99, 1)
		if far.Billing.Result != view.Billing.Result || far.Rows != nil {
			t.Fatalf("%s: %+v", c.name, far)
		}
	}
	want := map[string]bool{"conformant+ready": true, "conformant+not_ready": true, "partial+ready": true, "partial+not_ready": true, "nonconformant+not_ready": true}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("combinations of verdicts: %v", seen)
	}
	h := newHarness(t, nil)
	stated := testTarget()
	stated.Fields[1].Billing = true
	if err := h.r.DeclareTarget(stated); err != nil {
		t.Fatal(err)
	}
	h.one(fullEntries(), nil, good("a", "1", "USD"), without(good("b", "1", "USD", "EffectiveCost", null)), without(good("c", "1", "USD"), "EffectiveCost"), good("d", "1", "USD", "EffectiveCost", "0"))
	rows := h.rows("run-1")
	if rows[0].Lacking != nil || !reflect.DeepEqual(rows[1].Lacking, []string{"effective"}) || !reflect.DeepEqual(rows[2].Lacking, []string{"effective"}) || rows[3].Lacking != nil {
		t.Fatalf("%+v", rows)
	}
}

func TestSumsOutsideTheProfile(t *testing.T) {
	h := newHarness(t, nil)
	var rows []Row
	for index := 0; index < 101; index++ {
		rows = append(rows, good(fmt.Sprintf("k%03d", index), "999999999999", "USD", "EffectiveCost", "1", "ListCost", "0.0001"))
	}
	view := h.one(fullEntries(), func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"USD": "1"}} }, rows...)
	if verdicts(view) != "conformant[] not_ready[totals_differ sum_outside_profile]" || view.Sums["billed"]["USD"] != "100999999999899" || view.Sums["list"]["USD"] != "0.0101" || view.Sums["effective"]["USD"] != "101" {
		t.Fatalf("%s %v", verdicts(view), view.Sums)
	}
	alone := newHarness(t, nil)
	for index := range rows {
		rows[index] = good(fmt.Sprintf("k%03d", index), "1", "USD", "EffectiveCost", "999999999999")
	}
	view = alone.one(fullEntries(), func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"USD": "101"}} }, rows...)
	if verdicts(view) != "conformant[] not_ready[sum_outside_profile]" {
		t.Fatal(verdicts(view))
	}
	inside := newHarness(t, nil)
	view = inside.one(fullEntries(), func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"USD": "0"}} }, good("a", "999999999999", "USD"), good("b", "-999999999999", "USD"))
	if verdicts(view) != "conformant[] ready[]" {
		t.Fatal(verdicts(view))
	}
}
