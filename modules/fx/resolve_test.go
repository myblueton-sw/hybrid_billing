package fx

import (
	"testing"
	"time"
)

func reason(t testing.TB, r *Registry, id string, n Need) string {
	t.Helper()
	snap := prepare(t, r, id, n)
	e := snap.Entries[0]
	if e.Kind == EntryHeld {
		return "held:" + e.Reason
	}
	return e.Reason + "/" + e.Policy + "/" + e.SalesQuote
}

func TestSelectionAndPrecedence(t *testing.T) {
	r, _ := open(t)
	usd := key("USD", "KRW", "2026-11-10")
	feed(t, r, usd, 1, 1, "1350")
	n := need("n", "USD", "KRW", "2026-11-10")
	if got := reason(t, r, "s1", n); got != "held:contract_not_stated" {
		t.Fatalf("nothing stated: %s", got)
	}
	policyVersion(t, r, "rev", attrs(LevelContract, "con-1", "KRW", "USD"), dated(1, "2026-01-01", ConstructionInverse))
	policyVersion(t, r, "other-contract", attrs(LevelContract, "con-2", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	policyVersion(t, r, "other-purpose", PolicyAttributes{Level: LevelContract, Key: "con-1", Purpose: "display", Source: "USD", Target: "KRW"}, dated(1, "2026-01-01", ConstructionDirect))
	if got := reason(t, r, "s2", n); got != "held:contract_not_stated" {
		t.Fatalf("a policy of another direction, contract or purpose: %s", got)
	}
	policyVersion(t, r, "con", attrs(LevelContract, "con-1", "USD", "KRW"), VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodDefaultAllowed})
	if got := reason(t, r, "s3", n); got != "held:no_default_policy" {
		t.Fatalf("a default allowed and none: %s", got)
	}
	msp := dated(1, "2026-11-11", ConstructionDirect)
	policyVersion(t, r, "msp", attrs(LevelMSP, "", "USD", "KRW"), msp)
	if got := reason(t, r, "s4", n); got != "held:no_default_policy" {
		t.Fatalf("a default not yet in force: %s", got)
	}
	if got := reason(t, r, "s5", need("n", "USD", "KRW", "2026-11-11")); got != "held:no_rate_recorded" {
		t.Fatalf("a default in force on its first day: %s", got)
	}
	feed(t, r, key("USD", "KRW", "2026-11-11"), 1, 1, "1351")
	if got := reason(t, r, "s6", need("n", "USD", "KRW", "2026-11-11")); got != "msp_default/msp/1351" {
		t.Fatalf("the default: %s", got)
	}
	own := dated(2, "2026-11-11", ConstructionDirect)
	own.Spread = "0.01"
	policyVersion(t, r, "con", attrs(LevelContract, "con-1", "USD", "KRW"), own)
	if got := reason(t, r, "s7", need("n", "USD", "KRW", "2026-11-11")); got != "contract/con/1364.51" {
		t.Fatalf("the contract's own rate: %s", got)
	}
	if got := reason(t, r, "s8", n); got != "held:no_default_policy" {
		t.Fatalf("the default allowed before the contract's own version: %s", got)
	}
	exception := dated(1, "2026-11-11", ConstructionDirect)
	exception.Spread = "0.002"
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), exception)
	if got := reason(t, r, "s9", need("n", "USD", "KRW", "2026-11-11")); got != "customer_exception/cust/1353.7" {
		t.Fatalf("an exception over the contract: %s", got)
	}
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), VersionInput{Version: 2, Effective: "2026-11-12", Author: "author", Method: MethodEnded})
	feed(t, r, key("USD", "KRW", "2026-11-12"), 1, 1, "1352")
	if got := reason(t, r, "s10", need("n", "USD", "KRW", "2026-11-12")); got != "contract/con/1365.52" {
		t.Fatalf("an ended exception: %s", got)
	}
	other := need("n", "USD", "KRW", "2026-11-11")
	other.Customer = "cust-2"
	if got := reason(t, r, "s11", other); got != "contract/con/1364.51" {
		t.Fatalf("another customer: %s", got)
	}
	policyVersion(t, r, "con", attrs(LevelContract, "con-1", "USD", "KRW"), VersionInput{Version: 3, Effective: "2026-11-12", Author: "author", Method: MethodEnded})
	if got := reason(t, r, "s12", need("n", "USD", "KRW", "2026-11-12")); got != "held:contract_not_stated" {
		t.Fatalf("an ended contract: %s", got)
	}
	snap := prepare(t, r, "s13", need("n", "USD", "KRW", "2026-11-12"))
	if e := snap.Entries[0]; e.Level != "" || e.Policy != "" || e.Version != 0 {
		t.Fatalf("an ended contract leaves no selection: %+v", e)
	}
}

func TestWithdrawalsHold(t *testing.T) {
	r, _ := open(t)
	for _, day := range []string{"2026-06-01", "2026-07-01", "2026-11-10"} {
		feed(t, r, key("USD", "KRW", day), 1, 1, "1300")
	}
	policyVersion(t, r, "msp", attrs(LevelMSP, "", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	policyVersion(t, r, "con", attrs(LevelContract, "con-1", "USD", "KRW"), VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodDefaultAllowed})
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), dated(2, "2026-06-01", ConstructionDirect))
	if _, err := r.WithdrawPolicyVersion("ws", "cust", 2, "manager", "mistake"); err != nil {
		t.Fatal(err)
	}
	if got := reason(t, r, "w1", need("n", "USD", "KRW", "2026-07-01")); got != "held:withdrawn_in_force" {
		t.Fatalf("a withdrawn version in force: %s", got)
	}
	if got := reason(t, r, "w2", need("n", "USD", "KRW", "2026-05-31")); got != "held:no_rate_recorded" {
		t.Fatalf("the version before it: %s", got)
	}
	early := dated(3, "2026-05-31", ConstructionDirect)
	if _, err := r.RecordPolicyVersion("ws", "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), early); kind(err) != "rejected" {
		t.Fatalf("an effective date before a withdrawn version: %v", err)
	}
	same := dated(3, "2026-06-01", ConstructionDirect)
	same.Spread = "0.01"
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), same)
	if got := reason(t, r, "w3", need("n", "USD", "KRW", "2026-07-01")); got != "customer_exception/cust/1313" {
		t.Fatalf("a later version on the withdrawn one's date: %s", got)
	}
	if got := reason(t, r, "w4", need("n", "USD", "KRW", "2026-06-01")); got != "customer_exception/cust/1313" {
		t.Fatalf("the later version on its first day: %s", got)
	}
	if _, err := r.WithdrawPolicyVersion("ws", "con", 1, "manager", "replaced"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.WithdrawPolicyVersion("ws", "cust", 3, "manager", "replaced"); err != nil {
		t.Fatal(err)
	}
	if got := reason(t, r, "w5", need("n", "USD", "KRW", "2026-11-10")); got != "held:withdrawn_in_force" {
		t.Fatalf("an exception withdrawn: %s", got)
	}
	policyVersion(t, r, "cust", attrs(LevelCustomer, "cust-1", "USD", "KRW"), VersionInput{Version: 4, Effective: "2026-07-01", Author: "author", Method: MethodEnded})
	if got := reason(t, r, "w6", need("n", "USD", "KRW", "2026-11-10")); got != "held:withdrawn_in_force" {
		t.Fatalf("a withdrawn contract: %s", got)
	}
	if _, err := r.WithdrawPolicyVersion("ws", "msp", 1, "manager", "mistake"); err != nil {
		t.Fatal(err)
	}
	policyVersion(t, r, "con", attrs(LevelContract, "con-1", "USD", "KRW"), VersionInput{Version: 2, Effective: "2026-07-01", Author: "author", Method: MethodDefaultAllowed})
	if got := reason(t, r, "w7", need("n", "USD", "KRW", "2026-11-10")); got != "held:withdrawn_in_force" {
		t.Fatalf("a withdrawn default: %s", got)
	}
	if got := reason(t, r, "w8", need("n", "USD", "KRW", "2025-12-31")); got != "held:contract_not_stated" {
		t.Fatalf("before every version: %s", got)
	}
}

func TestFixedRates(t *testing.T) {
	r, _ := open(t)
	fixed := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodFixed, Construction: ConstructionDirect, Unit: 1, Value: "1300.00", Evidence: "contract-annex"})
	policyVersion(t, r, "fix", attrs(LevelContract, "con-1", "USD", "KRW"), fixed)
	inverse := pricing(VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodFixed, Construction: ConstructionInverse, Unit: 1, Value: "1350", Evidence: "contract-annex"})
	policyVersion(t, r, "inv", attrs(LevelContract, "con-1", "KRW", "USD"), inverse)
	if got := reason(t, r, "f1", need("n", "USD", "KRW", "2026-11-10")); got != "held:awaiting_approval" {
		t.Fatalf("a fixed rate awaiting approval: %s", got)
	}
	if _, err := r.RecordPolicyApproval("ws", "fix", 1, "author", "ok-1"); kind(err) != "rejected" {
		t.Fatalf("approved by its author: %v", err)
	}
	for _, id := range []string{"fix", "inv"} {
		if _, err := r.RecordPolicyApproval("ws", id, 1, "manager", "ok-1"); err != nil {
			t.Fatal(err)
		}
	}
	snap := prepare(t, r, "f2", need("n", "USD", "KRW", "2026-11-10"), need("m", "KRW", "USD", "2026-11-10"))
	if got := entryOf(snap); got != "n:contract/contract/fix quote 1300 rate 1300; m:contract/contract/inv quote 1350 rate 1/1350" {
		t.Fatalf("fixed rates: %s", got)
	}
	if e := snap.Entries[0]; e.FixedValue != "1300" || e.FixedEvidence != "contract-annex" || e.FixedReference != "ok-1" || e.FixedUnit != 1 || len(e.Observations) != 0 {
		t.Fatalf("the fixed rate kept: %+v", e)
	}
	if got := shown(convert(t, r, "f2", line("a", "m", "1350000", "KRW", StateSource))); got != "a=1000 USD fee 0 total 1000 quoted 1000" {
		t.Fatalf("an inverse fixed rate: %s", got)
	}
}

func TestDatedSearch(t *testing.T) {
	r, clock := open(t)
	v := dated(1, "2026-01-01", ConstructionDirect)
	v.Rule, v.MaxDays = RulePreviousRecorded, 3
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), v)
	cross := dated(1, "2026-01-01", ConstructionCross)
	cross.Rule, cross.MaxDays = RulePreviousRecorded, 3
	policyVersion(t, r, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), cross)
	hold := dated(1, "2026-01-01", ConstructionDirect)
	policyVersion(t, r, "jpy", attrs(LevelContract, "con-1", "JPY", "KRW"), hold)
	feed(t, r, key("USD", "KRW", "2026-11-07"), 1, 1, "1300")
	snap := prepare(t, r, "d1", need("n", "USD", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.SalesQuote != "1300" || e.DateUsed != "2026-11-07" || e.DaysElapsed != 3 || e.Observations[0].Key.Date != "2026-11-07" {
		t.Fatalf("three days back: %+v", e)
	}
	if got := reason(t, r, "d2", need("n", "USD", "KRW", "2026-11-11")); got != "held:no_rate_recorded" {
		t.Fatalf("four days back: %s", got)
	}
	feed(t, r, key("JPY", "KRW", "2026-11-09"), 1, 100, "905")
	if got := reason(t, r, "d3", need("n", "JPY", "KRW", "2026-11-10")); got != "held:no_rate_recorded" {
		t.Fatalf("a rule that holds: %s", got)
	}
	feed(t, r, key("EUR", "USD", "2026-11-10"), 1, 1, "1.08")
	feed(t, r, key("KRW", "USD", "2026-11-09"), 1, 1000, "0.74")
	if got := reason(t, r, "d4", need("n", "EUR", "KRW", "2026-11-10")); got != "held:legs_not_on_one_date" {
		t.Fatalf("legs on different dates: %s", got)
	}
	feed(t, r, key("EUR", "USD", "2026-11-08"), 1, 1, "1.08")
	feed(t, r, key("KRW", "USD", "2026-11-08"), 1, 1000, "0.74")
	snap = prepare(t, r, "d5", need("n", "EUR", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.QuotedRate != "54000/37" || e.SalesQuote != "1459.46" || e.SalesRate != "72973/50" || e.DaysElapsed != 2 || len(e.Observations) != 2 || e.Unit != 1 {
		t.Fatalf("a cross on one date: %+v", e)
	}
	feed(t, r, key("USD", "KRW", "2026-11-13"), 1, 1, "1310")
	feed(t, r, key("USD", "KRW", "2026-11-12"), 1, 1, "1305")
	if got := reason(t, r, "d6", need("n", "USD", "KRW", "2026-11-13")); got != "contract/usd/1310" {
		t.Fatalf("today with a rate: %s", got)
	}
	feed(t, r, key("EUR", "USD", "2026-11-12"), 1, 1, "1.1")
	feed(t, r, key("KRW", "USD", "2026-11-12"), 1, 1000, "0.75")
	if got := reason(t, r, "d7", need("n", "EUR", "KRW", "2026-11-13")); got != "held:date_not_reached" {
		t.Fatalf("today without a rate falls back to nothing: %s", got)
	}
	if got := reason(t, r, "d8", need("n", "USD", "KRW", "2026-11-14")); got != "held:date_not_reached" {
		t.Fatalf("tomorrow: %s", got)
	}
	behind := need("n", "USD", "KRW", "2026-11-13")
	behind.Offset = -300
	if got := reason(t, r, "d9", behind); got != "held:date_not_reached" {
		t.Fatalf("a calendar behind UTC: %s", got)
	}
	ahead := need("n", "USD", "KRW", "2026-11-14")
	ahead.Offset = 540
	clock.set(time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC))
	feed(t, r, key("USD", "KRW", "2026-11-14"), 1, 1, "1320")
	if got := reason(t, r, "d10", ahead); got != "contract/usd/1320" {
		t.Fatalf("a calendar ahead of UTC: %s", got)
	}
	clock.set(start)
	if got := reason(t, r, "d11", ahead); got != "contract/usd/1320" {
		t.Fatalf("a clock that went back keeps its highest reading: %s", got)
	}
}

func TestApprovalsAndVoidsInTheSearch(t *testing.T) {
	r, _ := open(t)
	v := dated(1, "2026-01-01", ConstructionDirect)
	v.Rule, v.MaxDays = RulePreviousRecorded, 3
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), v)
	day := key("USD", "KRW", "2026-11-10")
	feed(t, r, key("USD", "KRW", "2026-11-09"), 1, 1, "1290")
	manual(t, r, day, 1, 1, "1300")
	if got := reason(t, r, "a1", need("n", "USD", "KRW", "2026-11-10")); got != "held:awaiting_approval" {
		t.Fatalf("a manual rate awaiting approval: %s", got)
	}
	if _, err := r.RecordObservationApproval("ws", day, 1, "manager", "ok-7"); err != nil {
		t.Fatal(err)
	}
	snap := prepare(t, r, "a2", need("n", "USD", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.SalesQuote != "1300" || e.Observations[0].Reference != "ok-7" || e.Observations[0].Origin != OriginManual || e.Observations[0].Evidence != "doc-1" {
		t.Fatalf("an approved manual rate: %+v", e)
	}
	feed(t, r, day, 2, 1, "1310")
	if got := reason(t, r, "a3", need("n", "USD", "KRW", "2026-11-10")); got != "contract/usd/1310" {
		t.Fatalf("a correction: %s", got)
	}
	if _, err := r.VoidObservation("ws", day, 2, "clerk", "doc-2", "mistake"); err != nil {
		t.Fatal(err)
	}
	if got := reason(t, r, "a4", need("n", "USD", "KRW", "2026-11-10")); got != "held:awaiting_approval" {
		t.Fatalf("a void awaiting approval: %s", got)
	}
	if _, err := r.RecordVoidApproval("ws", day, 2, "clerk", "ok-8"); kind(err) != "rejected" {
		t.Fatalf("a void approved by its voider: %v", err)
	}
	if _, err := r.RecordVoidApproval("ws", day, 2, "manager", "ok-8"); err != nil {
		t.Fatal(err)
	}
	snap = prepare(t, r, "a5", need("n", "USD", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.SalesQuote != "1290" || e.DateUsed != "2026-11-09" || e.VoidsPassed != 1 {
		t.Fatalf("an approved void passed over, the older revision not coming back: %+v", e)
	}
	held := dated(1, "2026-01-01", ConstructionDirect)
	policyVersion(t, r, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), held)
	eur := key("EUR", "KRW", "2026-11-10")
	feed(t, r, eur, 1, 1, "1450")
	if _, err := r.VoidObservation("ws", eur, 1, "clerk", "doc-3", "source_withdrew"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordVoidApproval("ws", eur, 1, "manager", "ok-9"); err != nil {
		t.Fatal(err)
	}
	snap = prepare(t, r, "a6", need("n", "EUR", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.Kind != EntryHeld || e.Reason != ReasonNoRateRecorded || e.VoidsPassed != 1 {
		t.Fatalf("a void under a rule that holds: %+v", e)
	}
	feed(t, r, key("USD", "KRW", "2026-11-08"), 1, 1, "1280")
	manual(t, r, key("USD", "KRW", "2026-11-09"), 2, 1, "1295")
	if got := reason(t, r, "a7", need("n", "USD", "KRW", "2026-11-10")); got != "held:awaiting_approval" {
		t.Fatalf("a correction awaiting approval ends the search: %s", got)
	}
}

func TestMonthlyAverages(t *testing.T) {
	r, clock := open(t)
	v := VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodMonthlyAverage, RateSource: "bank", Series: "deal", Construction: ConstructionDirect, Month: MonthPrevious, Averaging: Averaging, LeastDays: 3}
	policyVersion(t, r, "jpy", attrs(LevelContract, "con-1", "JPY", "KRW"), pricing(v))
	inverse := v
	inverse.Construction, inverse.LeastDays = ConstructionInverse, 2
	policyVersion(t, r, "krw", attrs(LevelContract, "con-1", "KRW", "USD"), pricing(inverse))
	same := v
	same.Month, same.LeastDays = MonthSame, 1
	policyVersion(t, r, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), pricing(same))
	feed(t, r, key("JPY", "KRW", "2026-10-01"), 1, 100, "905.25")
	feed(t, r, key("JPY", "KRW", "2026-10-02"), 1, 1, "9.0525")
	if got := reason(t, r, "m1", need("n", "JPY", "KRW", "2026-11-10")); got != "held:not_enough_days" {
		t.Fatalf("two days of three: %s", got)
	}
	feed(t, r, key("JPY", "KRW", "2026-10-31"), 1, 1, "9.10")
	snap := prepare(t, r, "m2", need("n", "JPY", "KRW", "2026-11-10"))
	if e := snap.Entries[0]; e.QuotedRate != "5441/6" || e.Unit != 100 || e.SalesQuote != "906.83" || len(e.Observations) != 3 || e.Averaging != Averaging {
		t.Fatalf("mixed units: %+v", e)
	}
	feed(t, r, key("USD", "KRW", "2026-10-05"), 1, 1, "1300")
	feed(t, r, key("USD", "KRW", "2026-10-06"), 1, 1, "1400")
	snap = prepare(t, r, "m3", need("n", "KRW", "USD", "2026-11-10"))
	if e := snap.Entries[0]; e.QuotedRate != "1350" || e.SalesRate != "1/1350" {
		t.Fatalf("an inverse average is one over the mean of quotes: %+v", e)
	}
	manual(t, r, key("JPY", "KRW", "2026-10-15"), 1, 1, "9")
	if got := reason(t, r, "m4", need("n", "JPY", "KRW", "2026-11-10")); got != "held:awaiting_approval" {
		t.Fatalf("a day awaiting approval: %s", got)
	}
	if got := reason(t, r, "m5", need("n", "EUR", "KRW", "2026-11-10")); got != "held:month_not_ended" {
		t.Fatalf("the current month: %s", got)
	}
	feed(t, r, key("EUR", "KRW", "2026-10-31"), 1, 1, "1450")
	clock.set(time.Date(2026, 11, 30, 15, 0, 0, 0, time.UTC))
	kst := need("n", "EUR", "KRW", "2026-10-20")
	kst.Offset = 540
	if got := reason(t, r, "m6", kst); got != "contract/eur/1450" {
		t.Fatalf("a month ended under a calendar ahead of UTC: %s", got)
	}
	clock2 := &testClock{now: time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)}
	r2, err := NewRegistry(testProfile(t), testLimits(), clock2.read)
	if err != nil {
		t.Fatal(err)
	}
	policyVersion(t, r2, "eur", attrs(LevelContract, "con-1", "EUR", "KRW"), pricing(same))
	feed(t, r2, key("EUR", "KRW", "2026-10-31"), 1, 1, "1450")
	if got := reason(t, r2, "m7", need("n", "EUR", "KRW", "2026-10-20")); got != "held:month_not_ended" {
		t.Fatalf("the last day of the month in UTC: %s", got)
	}
	if got := reason(t, r2, "m8", kst); got != "contract/eur/1450" {
		t.Fatalf("the first hours of the next month in KST: %s", got)
	}
}
