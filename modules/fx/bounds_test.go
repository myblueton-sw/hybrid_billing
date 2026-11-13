package fx

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestIdentifiersCurrenciesAndDatesAtTheirBounds(t *testing.T) {
	long := strings.Repeat("a", 128)
	cases := map[string]bool{
		long: true, long + "a": false, "": false, "AZaz09._:-": true, " a": false, "a ": false, "a/b": false, "a@b": false, "[": false, "`": false, "{": false, "@": false, "/": false, ";": false,
	}
	for id, want := range cases {
		if validID(id) != want {
			t.Errorf("identifier %q: %v", id, !want)
		}
	}
	for code, want := range map[string]bool{"USD": true, "AZZ": true, "ZAA": true, "usd": false, "US": false, "USDX": false, "U1D": false, "@SD": false, "[SD": false, "U@D": false, "US[": false} {
		if validCurrency(code) != want {
			t.Errorf("currency %q: %v", code, !want)
		}
	}
	for date, want := range map[string]bool{"2000-01-01": true, "2099-12-31": true, "1999-12-31": false, "2100-01-01": false, "2024-02-29": true, "2026-02-29": false,
		"2026-00-10": false, "2026-13-10": false, "2026-04-31": false, "2026-04-00": false, "2026-04-3a": false, "2026/04/30": false, "2026-04-301": false, "202a-04-30": false} {
		if _, ok := parseDate(date); ok != want {
			t.Errorf("date %q: %v", date, ok)
		}
	}
}

func TestUnitsQuantaAndScalesAtTheirBounds(t *testing.T) {
	r, _ := open(t)
	for index, unit := range []int{1, 10, 100, 1000, 10000} {
		in := ObservationInput{Key: key("USD", "KRW", "2026-11-0"+string(rune('1'+index))), Revision: 1, Unit: unit, Value: "1", Origin: OriginFeed, Recorder: "loader"}
		if _, err := r.RecordObservation("ws", in); err != nil {
			t.Errorf("unit %d: %v", unit, err)
		}
	}
	for _, unit := range []int{0, 2, 11, 99, 1001, 10001, 100000} {
		in := ObservationInput{Key: key("USD", "KRW", "2026-11-09"), Revision: 1, Unit: unit, Value: "1", Origin: OriginFeed, Recorder: "loader"}
		if _, err := r.RecordObservation("ws", in); kind(err) != "invalid" {
			t.Errorf("unit %d: %v", unit, err)
		}
	}
	want := map[string]string{"1": "1235", "0.1": "1234.6", "0.01": "1234.57", "0.001": "1234.568", "0.0001": "1234.5679"}
	feed(t, r, key("EUR", "KRW", "2026-11-10"), 1, 1, "1234.56789")
	index := 0
	for quantum, amount := range want {
		v := dated(1, "2026-01-01", ConstructionDirect)
		v.QuoteScale, v.Quantum, v.Tolerance = 5, quantum, "0.0001"
		id := "q" + string(rune('a'+index))
		policyVersion(t, r, id, attrs(LevelContract, "con-"+id, "EUR", "KRW"), v)
		n := need("n", "EUR", "KRW", "2026-11-10")
		n.Contract = "con-" + id
		prepare(t, r, "run-"+id, n)
		if got := convert(t, r, "run-"+id, line("a", "n", "1", "EUR", StateSource)); got.Lines[0].Amount != amount {
			t.Errorf("quantum %s: %s", quantum, got.Lines[0].Amount)
		}
		index++
	}
	for _, quantum := range []string{"1.1", "0.2", "1.0001", "0.00001", "10"} {
		v := dated(1, "2026-01-01", ConstructionDirect)
		v.Quantum = quantum
		if _, err := r.RecordPolicyVersion("ws", "bad", attrs(LevelContract, "con-x", "EUR", "KRW"), v); kind(err) != "invalid" {
			t.Errorf("quantum %s: %v", quantum, err)
		}
	}
	wide, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 70, InputScale: 10, ResultIntegerDigits: 70, ResultScale: 10, IntermediateDigits: 200})
	if err != nil {
		t.Fatal(err)
	}
	w, _ := NewRegistry(wide, testLimits(), func() time.Time { return start })
	for scale, want := range map[int]string{8: "ok", 9: "invalid"} {
		v := dated(1, "2026-01-01", ConstructionDirect)
		v.QuoteScale = scale
		if _, err := w.RecordPolicyVersion("ws", "s"+string(rune('0'+scale)), attrs(LevelContract, "con-"+string(rune('0'+scale)), "EUR", "KRW"), v); kind(err) != want {
			t.Errorf("quote scale %d: %v", scale, err)
		}
	}
	for digits, want := range map[int]string{64: "ok", 65: "invalid"} {
		in := ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 1, Unit: 1, Value: strings.Repeat("9", digits-11) + "." + strings.Repeat("9", 10), Origin: OriginFeed, Recorder: "loader"}
		if digits == 65 {
			in.Key.Date = "2026-11-11"
		}
		if _, err := w.RecordObservation("ws", in); kind(err) != want {
			t.Errorf("a value of %d bytes: %v", digits, err)
		}
	}
	for offset, want := range map[int]string{-720: "ok", 840: "ok", -721: "invalid", 841: "invalid"} {
		n := need("n", "KRW", "KRW", "2026-11-10")
		n.Offset = offset
		if _, err := r.Prepare("ws", "off"+strconv.Itoa(offset+1000), []Need{n}); kind(err) != want {
			t.Errorf("offset %d: %v", offset, err)
		}
	}
}

func TestNumbersOfRevisionsAndVersionsAtTheirBounds(t *testing.T) {
	r, _ := open(t)
	k := key("USD", "KRW", "2026-11-10")
	for number, want := range map[int]string{1000: "not_found", 1001: "invalid"} {
		if _, err := r.RecordObservation("ws", ObservationInput{Key: k, Revision: number, Unit: 1, Value: "1", Origin: OriginFeed, Recorder: "loader"}); kind(err) != want {
			t.Errorf("record %d: %v", number, err)
		}
		if _, err := r.RecordObservationApproval("ws", k, number, "boss", "ok"); kind(err) != want {
			t.Errorf("approve %d: %v", number, err)
		}
		if _, err := r.VoidObservation("ws", k, number, "boss", "doc", "mistake"); kind(err) != want {
			t.Errorf("void %d: %v", number, err)
		}
		if _, err := r.RecordVoidApproval("ws", k, number, "boss", "ok"); kind(err) != want {
			t.Errorf("void approval %d: %v", number, err)
		}
	}
	for number, want := range map[int]string{10000: "not_found", 10001: "invalid"} {
		v := dated(number, "2026-01-01", ConstructionDirect)
		if _, err := r.RecordPolicyVersion("ws", "p", attrs(LevelContract, "con-1", "USD", "KRW"), v); kind(err) != want {
			t.Errorf("version %d: %v", number, err)
		}
		if _, err := r.RecordPolicyApproval("ws", "p", number, "boss", "ok"); kind(err) != want {
			t.Errorf("approve version %d: %v", number, err)
		}
		if _, err := r.WithdrawPolicyVersion("ws", "p", number, "boss", "mistake"); kind(err) != want {
			t.Errorf("withdraw version %d: %v", number, err)
		}
	}
	for _, date := range []string{"2026-04/30", "2026/04-30", "2026-04-30 "} {
		if _, ok := parseDate(date); ok {
			t.Errorf("date %q accepted", date)
		}
	}
}

func TestFormsJudgedOneByOne(t *testing.T) {
	r, _ := open(t)
	k := key("USD", "KRW", "2026-11-10")
	feed(t, r, k, 1, 1, "1350")
	bad := key("USD", "USD", "2026-11-10")
	if _, err := r.VoidObservation("", k, 1, "clerk", "doc", "mistake"); kind(err) != "invalid" {
		t.Errorf("void workspace: %v", err)
	}
	if _, err := r.VoidObservation("ws", bad, 1, "clerk", "doc", "mistake"); kind(err) != "invalid" {
		t.Errorf("void key: %v", err)
	}
	if _, err := r.RecordVoidApproval("", k, 1, "boss", "ok"); kind(err) != "invalid" {
		t.Errorf("void approval workspace: %v", err)
	}
	if _, err := r.RecordVoidApproval("ws", bad, 1, "boss", "ok"); kind(err) != "invalid" {
		t.Errorf("void approval key: %v", err)
	}
	fixed := func(apply func(*VersionInput)) VersionInput {
		in := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodFixed, Construction: ConstructionDirect, Unit: 1, Value: "1300", Evidence: "doc"})
		apply(&in)
		return in
	}
	for name, in := range map[string]VersionInput{
		"series": fixed(func(v *VersionInput) { v.Series = "deal" }), "pivot": fixed(func(v *VersionInput) { v.Pivot = "EUR" }),
		"days": fixed(func(v *VersionInput) { v.MaxDays = 1 }), "averaging": fixed(func(v *VersionInput) { v.Averaging = Averaging }),
		"least days": fixed(func(v *VersionInput) { v.LeastDays = 1 }),
	} {
		if _, err := r.RecordPolicyVersion("ws", "f", attrs(LevelContract, "con-1", "EUR", "KRW"), in); kind(err) != "invalid" {
			t.Errorf("fixed with %s: %v", name, err)
		}
	}
}

func TestMoreBoundsAndRepeats(t *testing.T) {
	r, _ := open(t)
	v := dated(1, "2026-01-01", ConstructionDirect)
	v.Rule, v.MaxDays = RulePreviousRecorded, 31
	if _, err := r.RecordPolicyVersion("ws", "a", attrs(LevelContract, "con-1", "USD", "KRW"), v); err != nil {
		t.Fatalf("thirty-one days back: %v", err)
	}
	m := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodMonthlyAverage, RateSource: "bank", Series: "deal", Construction: ConstructionDirect, Month: MonthSame, Averaging: Averaging, LeastDays: 31})
	if _, err := r.RecordPolicyVersion("ws", "b", attrs(LevelContract, "con-1", "EUR", "KRW"), m); err != nil {
		t.Fatalf("thirty-one least days: %v", err)
	}
	second := dated(2, "2026-02-01", ConstructionDirect)
	second.Rule, second.MaxDays = RulePreviousRecorded, 2
	if _, err := r.RecordPolicyVersion("ws", "a", attrs(LevelContract, "con-1", "USD", "KRW"), second); err != nil {
		t.Fatal(err)
	}
	if view, err := r.RecordPolicyVersion("ws", "a", attrs(LevelContract, "con-1", "USD", "KRW"), second); err != nil || view.Version != 2 || view.MaxDays != 2 {
		t.Fatalf("a repeat returns its own version: %v %+v", err, view)
	}
	if view, err := r.RecordObservation("ws", ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 1, Unit: 1, Value: "1", Origin: OriginFeed, Recorder: "loader"}); err != nil || view.Revision != 1 {
		t.Fatal(err)
	}
	r.RecordObservation("ws", ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 2, Unit: 1, Value: "2", Origin: OriginFeed, Recorder: "loader"})
	if view, err := r.RecordObservation("ws", ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 2, Unit: 1, Value: "2", Origin: OriginFeed, Recorder: "loader"}); err != nil || view.Revision != 2 || view.Value != "2" {
		t.Fatalf("a repeat returns its own revision: %v %+v", err, view)
	}
	if _, err := r.WithdrawPolicyVersion("", "a", 1, "boss", "mistake"); kind(err) != "invalid" {
		t.Errorf("withdraw workspace: %v", err)
	}
	if _, err := r.WithdrawPolicyVersion("ws", "", 1, "boss", "mistake"); kind(err) != "invalid" {
		t.Errorf("withdraw id: %v", err)
	}
	if _, err := r.RecordPolicyApproval("", "a", 1, "boss", "ok"); kind(err) != "invalid" {
		t.Errorf("approve workspace: %v", err)
	}
}
