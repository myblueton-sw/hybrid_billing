package tokenmeter

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestConstructionAndRulesForStrings(t *testing.T) {
	clock := &testClock{at: start}
	for index, change := range []func(*Limits){func(l *Limits) { l.Receptions = 0 }, func(l *Limits) { l.Events = 0 }, func(l *Limits) { l.Batch = 0 }, func(l *Limits) { l.Currencies = 0 }, func(l *Limits) { l.Documents = 0 },
		func(l *Limits) { l.Page = 0 }, func(l *Limits) { l.Listed = 0 }, func(l *Limits) { l.MaxPeriod = 0 }} {
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
	from, to := month(10)
	for _, r := range []*Registry{none, {}} {
		for index, err := range []error{third(r.Open(export("r", "s", 10))), second(r.Post("ws1", "r", []Event{event("e", 10)})), second(r.Close("ws1", "r")), r.Void("ws1", "r", ByOperator), second(r.Eligibility("ws1", "r")),
			second(r.Counting("ws1", "tm-1", from, to)), second(r.Reception("ws1", "r")), second(r.Events("ws1", "r", 0, 1)), second(r.Receptions("ws1", "tm-1", "", 1))} {
			if err != ErrInvalid {
				t.Fatalf("operation %d on a registry that was not built: %v", index, err)
			}
		}
	}
	long := strings.Repeat("a", 128)
	for text, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, long + "a": false, "": false, "a b": false, "a/b": false, "hello, world": false, "a@b": false, "a[b": false, "a`b": false, "a{b": false} {
		if validID(text) != want {
			t.Errorf("identifier %q", text)
		}
	}
	for text, want := range map[string]bool{"USD": true, "AZZ": true, "ZAA": true, "US": false, "USDT": false, "uSD": false, "UsD": false, "USd": false, "@SD": false, "U[D": false, "US1": false, "": false} {
		if validCurrency(text) != want {
			t.Errorf("currency %q", text)
		}
	}
	h := newHarness(t, nil)
	h.clock.at = start.Add(time.Hour).In(time.FixedZone("kst", 9*3600))
	h.open(export("r1", "s1", 10))
	h.clock.advance(-30 * time.Minute)
	h.open(export("r2", "s2", 11))
	for _, id := range []string{"r1", "r2"} {
		if view := h.view(id); !view.Opened.Equal(start.Add(time.Hour)) || view.Opened.Location() != time.UTC || !view.Changed.Equal(view.Opened) {
			t.Fatalf("%s: %+v", id, view)
		}
	}
}

func TestAmountTexts(t *testing.T) {
	for _, text := range []string{"", "+1", ".5", "5.", "1e", "1e+", "1e123", "NaN", "Infinity", "01", "-01", "1,5", " 1", "1 ", "0x10", "--1", "1.2.3", "1e1.5", "-", "-.5", "00", strings.Repeat("1", 41), "1." + strings.Repeat("0", 39)} {
		if _, _, formed := parseAmount(text); formed {
			t.Errorf("%q is taken as an amount", text)
		}
	}
	cases := []struct {
		text   string
		value  string
		digits int
	}{
		{"0", "0", 1}, {"-0", "0", 1}, {"0.000", "0", 1}, {"0e5", "0", 1}, {"1", "1", 1}, {"-1.50", "-1.5", 2}, {"1.50", "1.5", 2}, {"100000", "100000", 1}, {"1e16", "10000000000000000", 1}, {"1E5", "100000", 1}, {"1e05", "100000", 1},
		{"1e-05", "0.00001", 1}, {"0.00012", "0.00012", 2}, {"12.5e+1", "125", 3}, {"12.5e-3", "0.0125", 3}, {"-7.25E-2", "-0.0725", 3}, {"123456789012345", "123456789012345", 15}, {"0.1234567890123456", "0.1234567890123456", 16},
		{"0.30000000000000004", "0.30000000000000004", 17}, {"1.5e99", "15" + strings.Repeat("0", 98), 2}, {"1e-99", "0." + strings.Repeat("0", 98) + "1", 1}, {strings.Repeat("9", 40), strings.Repeat("9", 40), 40}, {"1." + strings.Repeat("0", 38), "1", 1},
	}
	for _, c := range cases {
		value, digits, formed := parseAmount(c.text)
		if !formed || value.text() != c.value || digits != c.digits || value.scale < 0 {
			t.Errorf("%q: %v %d %v, want %s with %d digits", c.text, value, digits, formed, c.value, c.digits)
		}
	}
	for text, want := range map[string]bool{"0": true, "7": true, "123456789012345678": true, "1234567890123456789": false, "07": false, "+7": false, "-1": false, "1.0": false, "": false, "1e3": false} {
		if _, formed := parseWhole(text, maxCountDigits); formed != want {
			t.Errorf("whole number %q", text)
		}
	}
	if _, formed := parseWhole(strings.Repeat("9", 30), maxTotalDigits); !formed {
		t.Fatal("thirty digits")
	}
	if _, formed := parseWhole(strings.Repeat("9", 31), maxTotalDigits); formed {
		t.Fatal("thirty-one digits")
	}
	random := rand.New(rand.NewSource(19600))
	tenth := big.NewInt(10)
	profile, held := testProfile(t), map[bool]int{}
	for round := 0; round < 20000; round++ {
		text := ""
		if random.Intn(3) == 0 {
			text = "-"
		}
		if random.Intn(4) == 0 {
			text += "0"
		} else {
			text += fmt.Sprint(1 + random.Intn(9))
			for count := random.Intn(18); count > 0; count-- {
				text += fmt.Sprint(random.Intn(10) * random.Intn(2))
			}
		}
		if random.Intn(2) == 0 {
			text += "."
			for count := 1 + random.Intn(12); count > 0; count-- {
				text += fmt.Sprint(random.Intn(10) * random.Intn(2))
			}
		}
		if random.Intn(3) == 0 {
			text += []string{"e", "E"}[random.Intn(2)] + []string{"", "+", "-"}[random.Intn(3)] + fmt.Sprint(random.Intn(25))
		}
		value, digits, formed := parseAmount(text)
		if !formed {
			t.Fatalf("%q refused", text)
		}
		exact, parsed := new(big.Rat).SetString(text)
		if !parsed {
			t.Fatalf("the oracle refused %q", text)
		}
		mine := new(big.Rat).SetFrac(value.coefficient, new(big.Int).Exp(tenth, big.NewInt(int64(value.scale)), nil))
		if mine.Cmp(exact) != 0 || value.scale < 0 {
			t.Fatalf("%q: %v, oracle %v", text, mine, exact)
		}
		if value.fits(profile) != ratFits(exact) {
			t.Fatalf("%q: inside the profile %v, oracle %v", text, value.fits(profile), ratFits(exact))
		}
		held[ratFits(exact)]++
		printed, readable := new(big.Rat).SetString(value.text())
		if !readable || printed.Cmp(exact) != 0 || strings.ContainsAny(value.text(), "eE") {
			t.Fatalf("%q printed as %q", text, value.text())
		}
		want := 1
		if exact.Sign() != 0 {
			whole := new(big.Rat).Set(exact)
			for !whole.IsInt() {
				whole.Mul(whole, big.NewRat(10, 1))
			}
			coefficient := new(big.Int).Abs(whole.Num())
			for new(big.Int).Mod(coefficient, tenth).Sign() == 0 {
				coefficient.Div(coefficient, tenth)
			}
			want = len(coefficient.String())
		}
		if digits != want {
			t.Fatalf("%q: %d significant digits, oracle %d", text, digits, want)
		}
	}
	if held[true] < 1000 || held[false] < 1000 {
		t.Fatalf("texts inside and outside the profile: %v", held)
	}
}

func TestHeaderRefusals(t *testing.T) {
	from, to := month(10)
	cases := []struct {
		name string
		set  func(*Spec)
		want error
	}{
		{"workspace", func(s *Spec) { s.WorkspaceID = "" }, ErrInvalid},
		{"id", func(s *Spec) { s.ID = "r 1" }, ErrInvalid},
		{"source", func(s *Spec) { s.Source = "" }, ErrInvalid},
		{"snapshot", func(s *Spec) { s.Snapshot = "" }, ErrInvalid},
		{"schema", func(s *Spec) { s.Schema = "" }, ErrInvalid},
		{"connection", func(s *Spec) { s.Connection = "" }, ErrInvalid},
		{"a period that runs backwards", func(s *Spec) { s.PeriodStart, s.PeriodEnd = to, from }, ErrInvalid},
		{"a period of no length", func(s *Spec) { s.PeriodEnd = from }, ErrInvalid},
		{"a start that is not a whole second", func(s *Spec) { s.PeriodStart = from.Add(time.Millisecond) }, ErrInvalid},
		{"an end that is not a whole second", func(s *Spec) { s.PeriodEnd = to.Add(-time.Nanosecond) }, ErrInvalid},
		{"a period beyond the longest", func(s *Spec) { s.PeriodEnd = from.Add(40*day + time.Second) }, ErrInvalid},
		{"revision zero", func(s *Spec) { s.Revision = 0 }, ErrInvalid},
		{"revision too large", func(s *Spec) { s.Revision = 1000000 }, ErrInvalid},
		{"how it was produced", func(s *Spec) { s.Produced = "scraped" }, ErrInvalid},
		{"the kind of numeric", func(s *Spec) { s.Numeric = "" }, ErrInvalid},
		{"another unit", func(s *Spec) { s.Unit = "kilotoken" }, ErrInvalid},
		{"no unit", func(s *Spec) { s.Unit = "" }, ErrInvalid},
		{"no event declared", func(s *Spec) { s.Events = 0 }, ErrInvalid},
		{"declaration", func(s *Spec) { s.Declaration = "" }, ErrInvalid},
		{"a token total that is no whole number", func(s *Spec) { s.Tokens[Input] = "3.0" }, ErrInvalid},
		{"a token total with a leading zero", func(s *Spec) { s.Tokens[Output] = "030" }, ErrInvalid},
		{"a token total of thirty-one digits", func(s *Spec) { s.Tokens[Output] = strings.Repeat("9", 31) }, ErrInvalid},
		{"a token total given as empty", func(s *Spec) { s.Tokens[CacheWrite] = "" }, ErrInvalid},
		{"an unknown token kind", func(s *Spec) { s.Tokens["reasoning"] = "1" }, ErrInvalid},
		{"a malformed currency", func(s *Spec) { s.Costs = map[string]string{"usd": "1"} }, ErrInvalid},
		{"a cost total that is no amount", func(s *Spec) { s.Costs = map[string]string{"USD": "NaN"} }, ErrInvalid},
		{"more events than a reception may have", func(s *Spec) { s.Events = 200001 }, ErrLimit},
		{"more currencies than a reception may have", func(s *Spec) { s.Costs = map[string]string{"USD": "1", "EUR": "1", "KRW": "1", "JPY": "1"} }, ErrLimit},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := export("r1", "s1", 10)
		c.set(&spec)
		_, _, err := h.r.Open(spec)
		wantErr(t, err, c.want, c.name)
		if h.r.dump() != "" {
			t.Fatalf("%s wrote something", c.name)
		}
	}
	h := newHarness(t, func(l *Limits) { l.Receptions = 2 })
	edge := export("r1", "s1", 10)
	edge.PeriodEnd, edge.Events, edge.Revision = from.Add(40*day), 200000, 999999
	edge.Tokens = map[Kind]string{Input: strings.Repeat("9", 30)}
	edge.Costs = map[string]string{"USD": "-1.5e20", "EUR": "0", "KRW": "1E5"}
	edge.Produced, edge.Numeric, edge.Declaration = ListInterface, Float, Derived
	h.open(edge)
	bare := export("r2", "s2", 12)
	bare.Tokens, bare.Costs = nil, nil
	h.open(bare)
	before := h.r.dump()
	_, _, err := h.r.Open(export("r3", "s3", 6))
	wantErr(t, err, ErrLimit, "reception limit")
	h.open(edge)
	h.unchanged(before, "the reception limit and a repeat at it")
	if view := h.view("r1"); view.Spec.Costs["USD"] != "-1.5e20" || view.Spec.Tokens[Input] != strings.Repeat("9", 30) || len(view.Spec.Tokens) != 1 {
		t.Fatalf("texts are kept as they were given: %+v", view.Spec)
	}
}

func TestHeaderRepeatsAndOneReceptionPerExport(t *testing.T) {
	h := newHarness(t, nil)
	spec := export("r1", "s1", 10)
	h.open(spec)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	again := export("r1", "s1", 10)
	again.Schema, again.Connection = "v3", "conn-2"
	again.PeriodStart = again.PeriodStart.In(time.FixedZone("kst", 9*3600))
	if state, occupant, err := h.r.Open(again); err != nil || state != Open || occupant != "" {
		t.Fatalf("the same reception through another connection and schema version: %v %q %v", state, occupant, err)
	}
	if view := h.view("r1"); view.Spec.Schema != "v2" || view.Spec.Connection != "conn-1" || view.Spec.PeriodStart.Location() != time.UTC {
		t.Fatalf("%+v", view.Spec)
	}
	from, _ := month(10)
	for name, change := range map[string]func(*Spec){
		"source":                        func(s *Spec) { s.Source = "tm-2" },
		"snapshot":                      func(s *Spec) { s.Snapshot = "s9" },
		"start":                         func(s *Spec) { s.PeriodStart = from.Add(time.Second) },
		"end":                           func(s *Spec) { s.PeriodEnd = s.PeriodEnd.Add(-time.Second) },
		"revision":                      func(s *Spec) { s.Revision = 2 },
		"produced":                      func(s *Spec) { s.Produced = ListInterface },
		"numeric":                       func(s *Spec) { s.Numeric = Float },
		"events":                        func(s *Spec) { s.Events = 4 },
		"declaration":                   func(s *Spec) { s.Declaration = Derived },
		"a token total":                 func(s *Spec) { s.Tokens[Input] = "301" },
		"a token total of another kind": func(s *Spec) { s.Tokens[Input], s.Tokens[Output] = "30", "300" },
		"a token total not stated":      func(s *Spec) { delete(s.Tokens, CacheWrite) },
		"a cost total":                  func(s *Spec) { s.Costs["USD"] = "4" },
		"the spelling of a cost total":  func(s *Spec) { s.Costs["USD"] = "3.0" },
		"a currency":                    func(s *Spec) { s.Costs = map[string]string{"EUR": "3"} },
		"a currency more":               func(s *Spec) { s.Costs["EUR"] = "0" },
		"no cost total":                 func(s *Spec) { s.Costs = nil },
	} {
		other := export("r1", "s1", 10)
		change(&other)
		_, _, err := h.r.Open(other)
		wantErr(t, err, ErrConflict, name)
	}
	zeroed := export("z1", "sz", 9)
	zeroed.Costs = map[string]string{"USD": "0"}
	h.open(zeroed)
	before2 := h.r.dump()
	negative := export("z1", "sz", 9)
	negative.Costs = map[string]string{"USD": "-0"}
	_, _, err := h.r.Open(negative)
	wantErr(t, err, ErrConflict, "a negative zero against a zero")
	h.unchanged(before2, "texts compared as texts")
	if err := h.r.Void("ws1", "z1", WrongReport); err != nil {
		t.Fatal(err)
	}
	partly := export("p1", "sp", 9)
	partly.Tokens = map[Kind]string{Input: "300"}
	h.open(partly)
	moved := export("p1", "sp", 9)
	moved.Tokens = map[Kind]string{Output: "300"}
	_, _, err = h.r.Open(moved)
	wantErr(t, err, ErrConflict, "the one stated total under another kind")
	before = h.r.dump()
	state, occupant, err := h.r.Open(export("r9", "s1", 10))
	if err != ErrConflict || state != "" || occupant != "r1" {
		t.Fatalf("a second reception of an export: %v %q %v", state, occupant, err)
	}
	h.unchanged(before, "repeats and conflicts")
	far := export("r1", "s1", 10)
	far.WorkspaceID = "ws2"
	if state, _, err := h.r.Open(far); err != nil || state != Open {
		t.Fatalf("the same identifiers in another workspace: %v %v", state, err)
	}
	other := export("r5", "s1", 10)
	other.Source = "tm-2"
	h.open(other)
	if err := h.r.Void("ws1", "r1", WrongReport); err != nil {
		t.Fatal(err)
	}
	corrected := export("r1b", "s1", 10)
	corrected.Events = 7
	h.open(corrected)
	before = h.r.dump()
	if state, occupant, err := h.r.Open(spec); err != nil || state != Voided || occupant != "" {
		t.Fatalf("an identical open of a voided reception whose export another reception took: %v %q %v", state, occupant, err)
	}
	h.unchanged(before, "a repeat of a voided reception")
	if view := h.view("r1b"); view.Spec.Events != 7 || !view.Opened.Equal(start.Add(time.Hour)) {
		t.Fatalf("%+v", view)
	}
}
