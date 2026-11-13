package normalize

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestConstruction(t *testing.T) {
	clock := &testClock{at: start}
	changes := []func(*Limits){
		func(l *Limits) { l.Targets = 0 }, func(l *Limits) { l.Fields = 0 }, func(l *Limits) { l.Mappings = 0 }, func(l *Limits) { l.Revisions = 0 }, func(l *Limits) { l.Runs = 0 },
		func(l *Limits) { l.Chunks = 0 }, func(l *Limits) { l.ChunkRows = 0 }, func(l *Limits) { l.Rows = 0 }, func(l *Limits) { l.Cells = 0 }, func(l *Limits) { l.Text = 0 },
		func(l *Limits) { l.Findings = 0 }, func(l *Limits) { l.Unused = 0 }, func(l *Limits) { l.Page = 0 },
	}
	for index, change := range changes {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(limits, testProfile(t), clock.read); err != ErrInvalid {
			t.Fatalf("limit %d: %v", index, err)
		}
	}
	if _, err := NewRegistry(testLimits(), decimal.Profile{}, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testLimits(), testProfile(t), nil); err != ErrInvalid {
		t.Fatal(err)
	}
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		if err := r.DeclareTarget(testTarget()); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.DeclareMapping(mapping, 0, testEntries()); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Revision(mapping, 1); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Latest(mapping); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Open(spec("r", 1, 1)); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Check("ws1", "r", 1, []Row{good("k", "1", "USD")}); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, _, err := r.Close("ws1", "r"); err != ErrInvalid {
			t.Fatal(err)
		}
		if err := r.Abandon("ws1", "r", Superseded); err != ErrInvalid {
			t.Fatal(err)
		}
		if err := r.Discard("ws1", "r", WrongInput); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Run("ws1", "r"); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Rows("ws1", "r", 0, 1); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Findings("ws1", "r", 0, 1); err != ErrInvalid {
			t.Fatal(err)
		}
		if _, err := r.Quarantined("ws1", "r", 0, 1); err != ErrInvalid {
			t.Fatal(err)
		}
	}
}

func TestTargets(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Target)
		want   error
	}{
		{"id", func(x *Target) { x.ID = "a b" }, ErrInvalid},
		{"no field", func(x *Target) { x.Fields = nil; x.CostCurrency, x.PeriodStart, x.PeriodEnd = "", "", "" }, ErrInvalid},
		{"field name", func(x *Target) { x.Fields[8].Name = "" }, ErrInvalid},
		{"field twice", func(x *Target) { x.Fields[8].Name = "svc" }, ErrInvalid},
		{"kind", func(x *Target) { x.Fields[8].Kind = "number" }, ErrInvalid},
		{"cost without kind", func(x *Target) { x.Fields[1].Cost = "" }, ErrInvalid},
		{"sale cost", func(x *Target) { x.Fields[1].Cost = Sale }, ErrInvalid},
		{"unknown cost kind", func(x *Target) { x.Fields[1].Cost = "amortized" }, ErrInvalid},
		{"cost kind on a quantity", func(x *Target) { x.Fields[3].Cost = Billed }, ErrInvalid},
		{"sale on a text", func(x *Target) { x.Fields[8].Cost = Sale }, ErrInvalid},
		{"costs without currency", func(x *Target) { x.CostCurrency = "" }, ErrInvalid},
		{"currency unknown", func(x *Target) { x.CostCurrency = "money" }, ErrInvalid},
		{"currency of another kind", func(x *Target) { x.CostCurrency = "svc" }, ErrInvalid},
		{"start only", func(x *Target) { x.PeriodEnd = "" }, ErrInvalid},
		{"end only", func(x *Target) { x.PeriodStart = "" }, ErrInvalid},
		{"start unknown", func(x *Target) { x.PeriodStart = "begin" }, ErrInvalid},
		{"end of another kind", func(x *Target) { x.PeriodEnd = "svc" }, ErrInvalid},
		{"start is end", func(x *Target) { x.PeriodEnd = "start" }, ErrInvalid},
		{"too many fields", func(x *Target) {
			for _, name := range []string{"a", "b", "c", "d"} {
				x.Fields = append(x.Fields, Field{Name: name, Kind: Text})
			}
		}, ErrLimit},
		{"at the field limit", func(x *Target) {
			for _, name := range []string{"a", "b", "c"} {
				x.Fields = append(x.Fields, Field{Name: name, Kind: Text})
			}
		}, nil},
		{"no period", func(x *Target) { x.PeriodStart, x.PeriodEnd = "", "" }, nil},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		stated := testTarget()
		c.change(&stated)
		before := h.r.dump()
		err := h.r.DeclareTarget(stated)
		if c.want == nil {
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			continue
		}
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	h := newHarness(t, func(l *Limits) { l.Targets = 2 })
	plain := Target{ID: "plain", Fields: []Field{{Name: "a", Kind: Text}}}
	for _, bad := range []Target{{ID: "plain", Fields: []Field{{Name: "a", Kind: Text}}, CostCurrency: "a"}, {ID: "plain", Fields: []Field{{Name: "a", Kind: Currency}}, CostCurrency: "a"}} {
		wantErr(t, h.r.DeclareTarget(bad), ErrInvalid, "a currency of costs without costs")
	}
	if err := h.r.DeclareTarget(plain); err != nil {
		t.Fatal(err)
	}
	stated := testTarget()
	if err := h.r.DeclareTarget(stated); err != nil {
		t.Fatal(err)
	}
	stated.Fields[0].Kind, stated.Fields[0].Cost = Text, ""
	if _, err := h.r.DeclareMapping(mapping, 0, testEntries()); err != nil {
		t.Fatalf("the registry kept the caller's fields: %v", err)
	}
	h.clock.advance(time.Minute)
	before := h.r.dump()
	if err := h.r.DeclareTarget(testTarget()); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Target){
		"format flag":  func(x *Target) { x.Fields[1].Format = true },
		"billing flag": func(x *Target) { x.Fields[1].Billing = true },
		"cost kind":    func(x *Target) { x.Fields[1].Cost = List },
		"kind":         func(x *Target) { x.Fields[8].Kind = Time },
		"name":         func(x *Target) { x.Fields[8].Name = "remark" },
		"order":        func(x *Target) { x.Fields[7], x.Fields[8] = x.Fields[8], x.Fields[7] },
		"period":       func(x *Target) { x.PeriodStart, x.PeriodEnd = "end", "start" },
		"no period":    func(x *Target) { x.PeriodStart, x.PeriodEnd = "", "" },
		"one fewer":    func(x *Target) { x.Fields = x.Fields[:8] },
	} {
		other := testTarget()
		change(&other)
		wantErr(t, h.r.DeclareTarget(other), ErrConflict, name)
	}
	wantErr(t, h.r.DeclareTarget(Target{ID: "third", Fields: []Field{{Name: "a", Kind: Text}}}), ErrLimit, "target limit")
	h.unchanged(before, "target repeats and refusals")
}

func TestMappingRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func([]Entry) []Entry
		want   error
	}{
		{"an entry fewer", func(e []Entry) []Entry { return e[:8] }, ErrInvalid},
		{"an entry more", func(e []Entry) []Entry { return append(e, missing("extra", NotSupported)) }, ErrInvalid},
		{"a field twice", func(e []Entry) []Entry { e[8] = from("svc", "Other", Text, ""); return e }, ErrInvalid},
		{"an unknown field", func(e []Entry) []Entry { e[8] = missing("remark", NotApplicable); return e }, ErrInvalid},
		{"source name", func(e []Entry) []Entry { e[7].Source = "Service Name"; return e }, ErrInvalid},
		{"source kind", func(e []Entry) []Entry { e[7].SourceKind = "string"; return e }, ErrInvalid},
		{"cost without a cost kind", func(e []Entry) []Entry { e[0].SourceCost = ""; return e }, ErrInvalid},
		{"unknown cost kind", func(e []Entry) []Entry { e[0].SourceCost = "amortized"; return e }, ErrInvalid},
		{"cost kind on a text", func(e []Entry) []Entry { e[7].SourceCost = Billed; return e }, ErrInvalid},
		{"transform", func(e []Entry) []Entry { e[0].Transform = "round"; return e }, ErrInvalid},
		{"no transform", func(e []Entry) []Entry { e[0].Transform = ""; return e }, ErrInvalid},
		{"power on copy", func(e []Entry) []Entry { e[0].Power = 2; return e }, ErrInvalid},
		{"power on negate", func(e []Entry) []Entry { e[0].Transform, e[0].Power = Negate, 1; return e }, ErrInvalid},
		{"scale by nothing", func(e []Entry) []Entry { e[0].Transform = Scale; return e }, ErrInvalid},
		{"scale too far", func(e []Entry) []Entry { e[0].Transform, e[0].Power = Scale, 13; return e }, ErrInvalid},
		{"scale too far down", func(e []Entry) []Entry { e[0].Transform, e[0].Power = Scale, -13; return e }, ErrInvalid},
		{"missing reason", func(e []Entry) []Entry { e[2].Missing = "later"; return e }, ErrInvalid},
		{"missing with a source", func(e []Entry) []Entry { e[2].Source = "ListCost"; return e }, ErrInvalid},
		{"missing with a kind", func(e []Entry) []Entry { e[2].SourceKind = Cost; return e }, ErrInvalid},
		{"missing with a cost kind", func(e []Entry) []Entry { e[2].SourceCost = List; return e }, ErrInvalid},
		{"missing with a transform", func(e []Entry) []Entry { e[2].Transform = Copy; return e }, ErrInvalid},
		{"missing with a power", func(e []Entry) []Entry { e[2].Power = 1; return e }, ErrInvalid},
		{"a sale amount as billed", func(e []Entry) []Entry { e[0].SourceCost = Sale; return e }, ErrRejected},
		{"an effective cost as billed", func(e []Entry) []Entry { e[0].SourceCost = Effective; return e }, ErrRejected},
		{"a quantity as a cost", func(e []Entry) []Entry { e[0].SourceKind, e[0].SourceCost = Quantity, ""; return e }, ErrRejected},
		{"a cost as a quantity", func(e []Entry) []Entry { e[3].SourceKind, e[3].SourceCost = Cost, Billed; return e }, ErrRejected},
		{"a text as a time", func(e []Entry) []Entry { e[5].SourceKind = Text; return e }, ErrRejected},
		{"one source with two cost kinds", func(e []Entry) []Entry { e[1].Source = "BilledCost"; return e }, ErrRejected},
		{"one source with two kinds", func(e []Entry) []Entry { e[3].Source = "ServiceName"; return e }, ErrRejected},
		{"negate a text", func(e []Entry) []Entry { e[7].Transform = Negate; return e }, ErrRejected},
		{"scale a time", func(e []Entry) []Entry { e[5].Transform, e[5].Power = Scale, 3; return e }, ErrRejected},
		{"negate a currency", func(e []Entry) []Entry { e[4].Transform = Negate; return e }, ErrRejected},
		{"not applicable but required for format", func(e []Entry) []Entry { e[7] = missing("svc", NotApplicable); return e }, ErrRejected},
		{"not applicable but required for billing", func(e []Entry) []Entry {
			e[0], e[1], e[4] = missing("billed", NotSupported), missing("effective", NotSupported), missing("cur", NotApplicable)
			return e
		}, ErrRejected},
		{"not applicable on a cost that only billing requires", func(e []Entry) []Entry { e[2] = missing("list", NotApplicable); return e }, ErrRejected},
		{"a cost without its currency", func(e []Entry) []Entry { e[4] = missing("cur", NotSupported); return e }, ErrRejected},
		{"an optional cost without its currency", func(e []Entry) []Entry {
			e[0], e[4] = missing("billed", NotSupported), missing("cur", NotSupported)
			return e
		}, ErrRejected},
		{"start without end", func(e []Entry) []Entry { e[6] = missing("end", NotSupported); return e }, ErrRejected},
		{"end without start", func(e []Entry) []Entry { e[5] = missing("start", NotApplicable); return e }, ErrRejected},
		{"nothing mapped", func(e []Entry) []Entry {
			for index, name := range []string{"billed", "effective", "list", "qty", "cur", "start", "end", "svc", "note"} {
				e[index] = missing(name, NotSupported)
			}
			return e
		}, ErrRejected},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		stated := testTarget()
		stated.Fields[2].Billing = true
		if err := h.r.DeclareTarget(stated); err != nil {
			t.Fatal(err)
		}
		before := h.r.dump()
		_, err := h.r.DeclareMapping(mapping, 0, c.change(testEntries()))
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	h := newHarness(t, nil)
	_, err := h.r.DeclareMapping(mapping, 0, testEntries())
	wantErr(t, err, ErrNotFound, "no target")
	for _, key := range []MappingKey{{SourceVersion: "v", Target: "t1"}, {Provider: "aws", Target: "t1"}, {Provider: "aws", SourceVersion: "v"}, {Provider: "a b", SourceVersion: "v", Target: "t1"}} {
		_, err = h.r.DeclareMapping(key, 0, testEntries())
		wantErr(t, err, ErrInvalid, "mapping key")
		_, err = h.r.Latest(key)
		wantErr(t, err, ErrInvalid, "mapping key")
		_, err = h.r.Revision(key, 1)
		wantErr(t, err, ErrInvalid, "mapping key")
	}
}

func TestIdentifierRules(t *testing.T) {
	long := ""
	for len(long) < 128 {
		long += "a"
	}
	for text, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, long + "a": false, "": false, "a b": false, "a/b": false, "a@b": false, "a[b": false, "a`b": false, "a{b": false, "a,b": false} {
		if validID(text) != want {
			t.Errorf("identifier %q", text)
		}
	}
	for text, want := range map[string]bool{"USD": true, "AZZ": true, "ZAA": true, "US": false, "USDT": false, "uSD": false, "UsD": false, "USd": false, "@SD": false, "U[D": false, "US1": false, "": false} {
		if validCurrency(text) != want {
			t.Errorf("currency %q", text)
		}
	}
}

func TestMappingRevisionsAndTheirSupport(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Revisions = 4; l.Mappings = 2 })
	if err := h.r.DeclareTarget(testTarget()); err != nil {
		t.Fatal(err)
	}
	_, err := h.r.Latest(mapping)
	wantErr(t, err, ErrNotFound, "no revision")
	_, err = h.r.DeclareMapping(mapping, 1, testEntries())
	wantErr(t, err, ErrConflict, "expected ahead")
	shuffled := testEntries()
	shuffled[0], shuffled[8] = shuffled[8], shuffled[0]
	if number, err := h.r.DeclareMapping(mapping, 0, shuffled); err != nil || number != 1 {
		t.Fatal(number, err)
	}
	h.clock.advance(time.Minute)
	before := h.r.dump()
	for _, expected := range []uint64{0, 1, 9} {
		if number, err := h.r.DeclareMapping(mapping, expected, testEntries()); err != nil || number != 1 {
			t.Fatal(number, err)
		}
	}
	h.unchanged(before, "a repeated declaration")
	first, err := h.r.Latest(mapping)
	want := RevisionView{Key: mapping, Number: 1, Entries: testEntries(), Partial: []string{"list"}, NotApplicable: []string{"note"}, BillingPossible: true, At: start}
	if err != nil || !reflect.DeepEqual(first, want) {
		t.Fatalf("%+v %v", first, err)
	}
	first.Entries[0].Source = "changed"
	first.Partial[0] = "changed"
	if again, _ := h.r.Revision(mapping, 1); !reflect.DeepEqual(again, want) {
		t.Fatal("view shares state")
	}
	for _, expected := range []uint64{0, 2} {
		_, err = h.r.DeclareMapping(mapping, expected, fullEntries())
		wantErr(t, err, ErrConflict, "stale or early expected number")
	}
	h.unchanged(before, "expected number")
	if number, err := h.r.DeclareMapping(mapping, 1, fullEntries()); err != nil || number != 2 {
		t.Fatal(number, err)
	}
	second, _ := h.r.Latest(mapping)
	if !second.Complete || second.Partial != nil || !second.BillingPossible || !second.At.Equal(start.Add(time.Minute)) || second.Number != 2 {
		t.Fatalf("%+v", second)
	}
	optional := fullEntries()
	optional[1] = missing("effective", NotProvidedBySource)
	if number, err := h.r.DeclareMapping(mapping, 2, optional); err != nil || number != 3 {
		t.Fatal(number, err)
	}
	if third, _ := h.r.Latest(mapping); third.Complete || !reflect.DeepEqual(third.Partial, []string{"effective"}) || !third.BillingPossible {
		t.Fatalf("a field that is not required and not supported must make the mapping partial: %+v", third)
	}
	gap := testEntries()
	gap[0] = missing("billed", NotSupported)
	scaled := gap
	scaled[1].Transform, scaled[1].Power = Scale, -12
	if number, err := h.r.DeclareMapping(mapping, 3, scaled); err != nil || number != 4 {
		t.Fatal(number, err)
	}
	if fourth, _ := h.r.Latest(mapping); fourth.Complete || fourth.BillingPossible || fourth.NoRequiredCost || !reflect.DeepEqual(fourth.BillingGaps, []string{"billed"}) || !reflect.DeepEqual(fourth.Partial, []string{"billed", "list"}) {
		t.Fatalf("%+v", fourth)
	}
	before = h.r.dump()
	_, err = h.r.DeclareMapping(mapping, 4, fullEntries())
	wantErr(t, err, ErrLimit, "revision limit")
	if number, err := h.r.DeclareMapping(mapping, 0, scaled); err != nil || number != 4 {
		t.Fatal(number, err)
	}
	_, err = h.r.Revision(mapping, 5)
	wantErr(t, err, ErrNotFound, "revision ahead")
	_, err = h.r.Revision(mapping, 0)
	wantErr(t, err, ErrInvalid, "revision zero")
	if old, _ := h.r.Revision(mapping, 1); !reflect.DeepEqual(old, want) {
		t.Fatal("a revision changed")
	}
	other := mapping
	other.SourceVersion = "focus-1.2"
	if number, err := h.r.DeclareMapping(other, 0, fullEntries()); err != nil || number != 1 {
		t.Fatal(number, err)
	}
	third := mapping
	third.Provider = "gcp"
	before = h.r.dump()
	_, err = h.r.DeclareMapping(third, 0, fullEntries())
	wantErr(t, err, ErrLimit, "mapping limit")
	h.unchanged(before, "limits")
	if number, err := h.r.DeclareMapping(other, 1, testEntries()); err != nil || number != 2 {
		t.Fatalf("a new revision of an existing mapping is not a new mapping: %v %v", number, err)
	}

	costless := newHarness(t, nil)
	stated := testTarget()
	stated.ID, stated.Fields[0].Billing = "t1", false
	if err := costless.r.DeclareTarget(stated); err != nil {
		t.Fatal(err)
	}
	if _, err := costless.r.DeclareMapping(mapping, 0, fullEntries()); err != nil {
		t.Fatal(err)
	}
	if view, _ := costless.r.Latest(mapping); !view.Complete || view.BillingPossible || !view.NoRequiredCost || view.BillingGaps != nil {
		t.Fatalf("readiness needs a required cost: %+v", view)
	}
	same := fullEntries()
	same[1].Source, same[1].SourceCost = "BilledCost", Billed
	twice := newHarness(t, nil)
	both := testTarget()
	both.Fields[1].Cost = Billed
	if err := twice.r.DeclareTarget(both); err != nil {
		t.Fatal(err)
	}
	if _, err := twice.r.DeclareMapping(mapping, 0, same); err != nil {
		t.Fatalf("one source may feed two fields of one kind: %v", err)
	}
}
