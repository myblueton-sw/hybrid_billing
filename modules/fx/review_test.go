package fx

import (
	"strconv"
	"testing"
	"time"
)

func TestRoundingRefusals(t *testing.T) {
	r, _ := open(t)
	zero := dated(1, "2026-01-01", ConstructionDirect)
	zero.QuoteScale, zero.QuoteMode = 0, "floor"
	policyVersion(t, r, "krw", attrs(LevelContract, "con-1", "KRW", "USD"), zero)
	feed(t, r, key("KRW", "USD", "2026-11-10"), 1, 1000, "0.74")
	if got := reason(t, r, "z", need("n", "KRW", "USD", "2026-11-10")); got != "held:rate_rounds_to_zero" {
		t.Fatalf("a quote that rounds to zero: %s", got)
	}
	cross := dated(1, "2026-01-01", ConstructionCross)
	cross.QuoteScale = 0
	policyVersion(t, r, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), cross)
	feed(t, r, key("EUR", "USD", "2026-11-10"), 1, 1, "1.08")
	if got := reason(t, r, "c", need("n", "EUR", "KRW", "2026-11-10")); got != "held:rate_precision_lost" {
		t.Fatalf("a cross refused for precision: %s", got)
	}
	edge := dated(1, "2026-01-01", ConstructionDirect)
	edge.QuoteScale = 0
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), edge)
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1000.1")
	feed(t, r, key("USD", "KRW", "2026-11-09"), 1, 1, "1000.11")
	snap := prepare(t, r, "e1", need("n", "USD", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.Kind != EntryRate || e.Difference != "1/10000" || e.SalesQuote != "1000" {
		t.Fatalf("a difference equal to the tolerance: %+v", e)
	}
	if got := reason(t, r, "e2", need("n", "USD", "KRW", "2026-11-09")); got != "held:rate_precision_lost" {
		t.Fatalf("a difference above the tolerance: %s", got)
	}
	inverse := dated(1, "2026-01-01", ConstructionInverse)
	inverse.QuoteScale, inverse.QuoteMode, inverse.Tolerance = 2, "floor", "0.005"
	policyVersion(t, r, "chf", attrs(LevelContract, "con-1", "CHF", "GBP"), inverse)
	feed(t, r, key("GBP", "CHF", "2026-11-10"), 1, 1, "1.005025")
	snap = prepare(t, r, "i", need("n", "CHF", "GBP", "2026-11-10"))
	if e := snap.Entries[0]; e.Kind != EntryHeld || e.Reason != ReasonRatePrecisionLost || e.Difference != "201/40000" {
		t.Fatalf("the error of an inverse rate counted against the smaller: %+v", e)
	}
}

func TestMonthsOfEveryLengthAndTheTurnOfAYear(t *testing.T) {
	r, _ := open(t)
	cases := []struct {
		contract, month, date string
		least                 int
		want                  string
	}{
		{"feb26", MonthSame, "2026-02-10", 28, "contract/feb26/14.5"},
		{"feb26b", MonthSame, "2026-02-10", 29, "held:not_enough_days"},
		{"feb24", MonthSame, "2024-02-10", 29, "contract/feb24/15"},
		{"apr", MonthSame, "2026-04-10", 30, "contract/apr/15.5"},
		{"dec", MonthPrevious, "2026-01-15", 31, "contract/dec/16"},
	}
	for _, month := range []time.Time{time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)} {
		for day := month; day.Month() == month.Month(); day = day.AddDate(0, 0, 1) {
			feed(t, r, key("USD", "KRW", dateText(day)), 1, 1, strconv.Itoa(day.Day()))
		}
	}
	for _, c := range cases {
		v := pricing(VersionInput{Version: 1, Effective: "2024-01-01", Author: "author", Method: MethodMonthlyAverage, RateSource: "bank", Series: "deal", Construction: ConstructionDirect, Month: c.month, Averaging: Averaging, LeastDays: c.least})
		policyVersion(t, r, c.contract, attrs(LevelContract, c.contract, "USD", "KRW"), v)
		n := need("n", "USD", "KRW", c.date)
		n.Contract = c.contract
		if got := reason(t, r, "m-"+c.contract, n); got != c.want {
			t.Errorf("%s: %s", c.contract, got)
		}
	}
}

func TestNegativeAmountsUnderFloorAndCeiling(t *testing.T) {
	r, _ := open(t)
	for _, mode := range []string{"floor", "ceiling"} {
		v := dated(1, "2026-01-01", ConstructionDirect)
		v.QuoteScale, v.AmountMode = 1, mode
		policyVersion(t, r, mode, attrs(LevelContract, "con-"+mode, "USD", "KRW"), v)
	}
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1300.3")
	for mode, want := range map[string]string{
		"floor":   "a=1950 KRW fee 0 total 1950 quoted 1950; b=-1951 KRW fee 0 total -1951 quoted -1951",
		"ceiling": "a=1951 KRW fee 0 total 1951 quoted 1951; b=-1950 KRW fee 0 total -1950 quoted -1950",
	} {
		n := need("n", "USD", "KRW", "2026-11-10")
		n.Contract = "con-" + mode
		prepare(t, r, "run-"+mode, n)
		if got := shown(convert(t, r, "run-"+mode, line("a", "n", "1.5", "USD", StateSource), line("b", "n", "-1.5", "USD", StateSource))); got != want {
			t.Errorf("%s: %s", mode, got)
		}
	}
}

func TestContentDigestsAcrossPreparationTimes(t *testing.T) {
	r, clock := open(t)
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	first := prepare(t, r, "a", need("n", "USD", "KRW", "2026-11-10"))
	clock.set(start.Add(5 * time.Hour))
	second := prepare(t, r, "b", need("n", "USD", "KRW", "2026-11-10"))
	if first.ContentDigest != second.ContentDigest || first.Digest == second.Digest || !second.PreparedAt.Equal(start.Add(5*time.Hour)) || !first.PreparedAt.Equal(start) {
		t.Fatalf("content across times: %+v %+v", first, second)
	}
}

func TestVoidsAndTiesInAverages(t *testing.T) {
	r, _ := open(t)
	v := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodMonthlyAverage, RateSource: "bank", Series: "deal", Construction: ConstructionDirect, Month: MonthPrevious, Averaging: Averaging, LeastDays: 2})
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), v)
	feed(t, r, key("USD", "KRW", "2026-10-01"), 1, 1, "1300")
	feed(t, r, key("USD", "KRW", "2026-10-02"), 1, 1, "1400")
	feed(t, r, key("USD", "KRW", "2026-10-03"), 1, 1, "1500")
	r.VoidObservation("ws", key("USD", "KRW", "2026-10-02"), 1, "clerk", "doc", "duplicate")
	if got := reason(t, r, "v1", need("n", "USD", "KRW", "2026-11-10")); got != "held:awaiting_approval" {
		t.Fatalf("a void awaiting approval in a month: %s", got)
	}
	r.RecordVoidApproval("ws", key("USD", "KRW", "2026-10-02"), 1, "boss", "ok")
	snap := prepare(t, r, "v2", need("n", "USD", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.SalesQuote != "1400" || e.VoidsPassed != 1 || len(e.Observations) != 2 || e.Observations[1].Key.Date != "2026-10-03" {
		t.Fatalf("an approved void left out of an average: %+v", e)
	}
	d := dated(1, "2026-01-01", ConstructionDirect)
	d.Rule, d.MaxDays = RulePreviousRecorded, 3
	policyVersion(t, r, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), d)
	feed(t, r, key("EUR", "KRW", "2026-11-12"), 1, 1, "1450")
	manual(t, r, key("EUR", "KRW", "2026-11-13"), 1, 1, "1460")
	if got := reason(t, r, "v3", need("n", "EUR", "KRW", "2026-11-13")); got != "held:date_not_reached" {
		t.Fatalf("today with a value awaiting approval: %s", got)
	}
	held := prepare(t, r, "v4", need("n", "USD", "KRW", "2026-12-10"))
	if e := held.Entries[0]; len(e.Observations) != 0 {
		t.Fatalf("a held entry lists observations: %+v", e)
	}
}

func TestDigitsInAnAverage(t *testing.T) {
	limits := testLimits()
	limits.Digits = 6
	r, err := NewRegistry(testProfile(t), limits, func() time.Time { return start })
	if err != nil {
		t.Fatal(err)
	}
	v := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodMonthlyAverage, RateSource: "bank", Series: "deal", Construction: ConstructionDirect, Month: MonthPrevious, Averaging: Averaging, LeastDays: 1})
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), v)
	feed(t, r, key("USD", "KRW", "2026-10-01"), 1, 1, "1.2345678")
	if _, err := r.Prepare("ws", "a", []Need{need("n", "USD", "KRW", "2026-11-10")}); kind(err) != "limit" {
		t.Fatalf("digits in an average: %v", err)
	}
}

func TestLimitsInPreparationAndConversion(t *testing.T) {
	limits := testLimits()
	limits.Digits = 6
	r, _ := NewRegistry(testProfile(t), limits, func() time.Time { return start })
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	feed(t, r, key("USD", "KRW", "2026-11-09"), 1, 1, "1234.56")
	prepare(t, r, "run", need("n", "USD", "KRW", "2026-11-10"))
	if snap := prepare(t, r, "six", need("n", "USD", "KRW", "2026-11-09")); snap.Entries[0].SalesRate != "30864/25" {
		t.Fatalf("a rate of six digits at a limit of six: %+v", snap.Entries[0])
	}
	if got := shown(convert(t, r, "run", line("a", "n", "0.1234", "USD", StateSource))); got != "a=167 KRW fee 0 total 167 quoted 167" {
		t.Fatalf("an exact amount of six digits: %s", got)
	}
	if _, err := r.Convert("ws", "run", []Line{line("a", "n", "1234.5678", "USD", StateSource)}); kind(err) != "limit" {
		t.Fatalf("an exact amount beyond the digits: %v", err)
	}
	wide, _ := NewRegistry(testProfile(t), testLimits(), func() time.Time { return start })
	cross := dated(1, "2026-01-01", ConstructionCross)
	cross.Tolerance = "0.01"
	policyVersion(t, wide, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), cross)
	feed(t, wide, key("EUR", "USD", "2026-11-10"), 1, 1, "99999999999999")
	feed(t, wide, key("KRW", "USD", "2026-11-10"), 1, 1, "0.00000001")
	if _, err := wide.Prepare("ws", "big", []Need{need("n", "EUR", "KRW", "2026-11-10")}); kind(err) != "limit" {
		t.Fatalf("a quote the profile cannot hold: %v", err)
	}
	fee := dated(1, "2026-01-01", ConstructionDirect)
	fee.Fee = "0.5"
	policyVersion(t, wide, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), fee)
	feed(t, wide, key("USD", "KRW", "2026-11-10"), 1, 1, "100")
	prepare(t, wide, "fee", need("n", "USD", "KRW", "2026-11-10"))
	if got := shown(convert(t, wide, "fee", line("a", "n", "9999999999999", "USD", StateSource))); got != "a=999999999999900 KRW fee 499999999999950 total 1499999999999850 quoted 999999999999900" {
		t.Fatalf("a total of sixteen digits: %s", got)
	}
	if _, err := wide.Convert("ws", "fee", []Line{line("a", "n", "99999999999999", "USD", StateSource)}); kind(err) != "limit" {
		t.Fatalf("a total the profile cannot hold: %v", err)
	}
	eight := []Line{}
	for index := 0; index < 8; index++ {
		eight = append(eight, line("l"+strconv.Itoa(index), "n", "1", "USD", StateSource))
	}
	if out, err := wide.Convert("ws", "fee", eight); err != nil || len(out.Lines) != 8 {
		t.Fatalf("lines at their limit: %v", err)
	}
	if out := convert(t, wide, "fee"); out.Held != 0 || len(out.Lines) != 0 {
		t.Fatal("no lines")
	}
}

func TestHeldCountsAndSelectionsLeftEmpty(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodEnded})
	snap := prepare(t, r, "h", need("a", "USD", "KRW", "2026-11-10"), need("b", "KRW", "KRW", "2026-11-10"), need("c", "EUR", "KRW", "2026-11-10"))
	if snap.Held != 2 {
		t.Fatalf("held entries: %d", snap.Held)
	}
	if e := snap.Entries[0]; e.Reason != ReasonContractNotStated || e.Level != "" || e.Policy != "" || e.Version != 0 {
		t.Fatalf("an ended exception and no contract leave no selection: %+v", e)
	}
	listed, _ := r.Snapshot("ws", "h")
	if listed.Held != 2 {
		t.Fatal("held count read back")
	}
}

func TestExactlyAtTheLimitOfDigits(t *testing.T) {
	limits := testLimits()
	limits.Digits = 6
	r, _ := NewRegistry(testProfile(t), limits, func() time.Time { return start })
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-08"), 1, 1, "123456")
	if snap := prepare(t, r, "six", need("n", "USD", "KRW", "2026-11-08")); snap.Entries[0].SalesRate != "123456" {
		t.Fatalf("a quote of six digits: %+v", snap.Entries[0])
	}
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	prepare(t, r, "run", need("n", "USD", "KRW", "2026-11-10"))
	if got := shown(convert(t, r, "run", line("a", "n", "12.3456", "USD", StateSource))); got != "a=16667 KRW fee 0 total 16667 quoted 16667" {
		t.Fatalf("an exact amount of six digits: %s", got)
	}
}

func TestADefaultAllowedWithoutADefaultLeavesNoSelection(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "con", attrs(LevelContract, "con-1", "USD", "KRW"), VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodDefaultAllowed})
	snap := prepare(t, r, "d", need("n", "USD", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.Reason != ReasonNoDefaultPolicy || e.Level != "" || e.Policy != "" || e.Version != 0 || e.AllowingPolicy != "con" || e.AllowingVersion != 1 {
		t.Fatalf("a default allowed and none: %+v", e)
	}
}
