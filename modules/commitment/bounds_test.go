package commitment

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestInputsAtTheirBounds(t *testing.T) {
	long := strings.Repeat("a", 128)
	for id, want := range map[string]bool{long: true, long + "a": false, "": false, "AZaz09._:-": true, " a": false, "a ": false, "a/b": false, "@": false, "[": false, "`": false, "{": false, "/": false, ";": false} {
		if validID(id) != want {
			t.Errorf("identifier %q", id)
		}
	}
	for code, want := range map[string]bool{"AAA": true, "ZZZ": true, "AZA": true, "ZAZ": true, "@AA": false, "[AA": false, "A@A": false, "A[A": false, "AA@": false, "AA[": false, "aAA": false, "AAa": false, "AA": false, "AAAA": false} {
		if validCurrency(code) != want {
			t.Errorf("currency %q", code)
		}
	}
	for month, want := range map[string]bool{"2000-01": true, "2099-12": true, "1999-12": false, "2100-01": false, "2026-00": false, "2026-13": false, "2026-1a": false, "a026-01": false, "2a26-01": false,
		"20a6-01": false, "202a-01": false, "2026-a1": false, "2026/01": false, "2026-011": false, "2026-10": true, "2026-09": true} {
		if validMonth(month) != want {
			t.Errorf("month %q", month)
		}
	}
	clock := func() time.Time { return start }
	if _, err := NewRegistry(testProfile(t), Limits{Commitments: 1, Revisions: 1, Lines: 1, Customers: 1, Digits: 1}, clock); err != nil {
		t.Fatalf("limits of one: %v", err)
	}
	wide, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 70, InputScale: 4, ResultIntegerDigits: 70, ResultScale: 4, IntermediateDigits: 200})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewRegistry(wide, testLimits(), clock)
	for digits, want := range map[int]string{64: "ok", 65: "invalid"} {
		c := commitmentOf("c"+strings.Repeat("x", digits-60), PaymentAll, strings.Repeat("9", digits), RecoveryCharged)
		if _, err := r.RecordCommitment("ws", c); kind(err) != want {
			t.Errorf("a prepaid amount of %d bytes: %v", digits, err)
		}
	}
}

func TestEveryQuantumRoundsCharges(t *testing.T) {
	for quantum, want := range map[string]string{"1": "68", "0.1": "67.9", "0.01": "67.87", "0.001": "67.865", "0.0001": "67.8655"} {
		r := open(t)
		c := commitmentOf("c1", PaymentNone, "0", RecoveryCharged)
		c.Quantum = quantum
		record(t, r, c)
		write(t, r, "c1", "2026-03", 1, "61.6959", "0", used("l1", "acct-a", "61.6959", "0", "100"))
		p := sealed(t, r, "c1", "2026-03", 1)
		a, _ := Allocate(p, accounts(), bear(BearMSP, ""))
		s, err := Sell(a, []SalePolicy{costPlus("A", "0.1", false)})
		if err != nil || s.Charges[0].Amount != want {
			t.Errorf("quantum %s: %v %+v", quantum, err, s.Charges)
		}
	}
	for _, quantum := range []string{"1.1", "0.2", "1.0001", "0.00001"} {
		c := commitmentOf("c1", PaymentNone, "0", RecoveryCharged)
		c.Quantum = quantum
		if _, err := open(t).RecordCommitment("ws", c); kind(err) != "invalid" {
			t.Errorf("quantum %s: %v", quantum, err)
		}
	}
}

func TestRegistryBounds(t *testing.T) {
	r := open(t)
	one := commitmentOf("c1", PaymentNone, "0", RecoveryCharged)
	one.First, one.Last = "2026-12", "2026-12"
	if _, err := r.RecordCommitment("ws", one); err != nil {
		t.Fatalf("one month: %v", err)
	}
	bad := commitmentOf("c2", PaymentNone, "0", RecoveryCharged)
	bad.First = "2026-1"
	if _, err := r.RecordCommitment("ws", bad); kind(err) != "invalid" {
		t.Fatalf("a first month outside its form: %v", err)
	}
	if _, err := r.RecordPeriod("ws", "c1", "2026-12", 1, Facts{Recurring: "1", Amortized: "0", Lines: []Line{used("a", "acct-a", "1", "0", "1")}}); err != nil {
		t.Fatalf("the last month: %v", err)
	}
	for name, facts := range map[string]Facts{
		"recurring alone": {Recurring: "2", Amortized: "0", Lines: []Line{used("a", "acct-a", "1", "0", "1")}},
		"amortized alone": {Recurring: "1", Amortized: "1", Lines: []Line{used("a", "acct-a", "1", "0", "1")}},
		"lines alone":     {Recurring: "1", Amortized: "0", Lines: []Line{used("a", "acct-a", "1", "0", "1"), used("b", "acct-a", "0", "0", "0")}},
	} {
		if _, err := r.RecordPeriod("ws", "c1", "2026-12", 1, facts); kind(err) != "conflict" {
			t.Errorf("other facts, %s: %v", name, err)
		}
	}
	badLine := used("a", "acct-a", "x", "0", "1")
	if _, err := r.RecordPeriod("ws", "c1", "2026-12", 2, Facts{Recurring: "1", Amortized: "0", Lines: []Line{badLine}}); kind(err) != "invalid" {
		t.Fatalf("a line's recurring cost alone: %v", err)
	}
	eight := Facts{Recurring: "0", Amortized: "0"}
	for index := 0; index < 8; index++ {
		eight.Lines = append(eight.Lines, unused(string(rune('a'+index)), "0", "0"))
	}
	if _, err := r.RecordPeriod("ws", "c1", "2026-12", 2, eight); err != nil {
		t.Fatalf("eight lines: %v", err)
	}
	for number, want := range map[int]string{1000: "not_found", 1001: "invalid", 0: "invalid"} {
		if _, err := r.Seal("ws", "c1", "2026-12", number); kind(err) != want {
			t.Errorf("seal %d: %v", number, err)
		}
		if _, err := r.Revision("ws", "c1", "2026-12", number); kind(err) != want {
			t.Errorf("revision %d: %v", number, err)
		}
	}
	if _, err := r.RecordPeriod("ws", "c1", "2026-12", 1000, eight); kind(err) != "rejected" {
		t.Fatalf("revision 1000: %v", err)
	}
}

func TestAllocationInputForms(t *testing.T) {
	r, p := example(t)
	if _, err := r.Commitment("", "c1"); kind(err) != "invalid" {
		t.Fatal("commitment workspace")
	}
	if _, err := r.Commitment("ws", ""); kind(err) != "invalid" {
		t.Fatal("commitment identifier")
	}
	attr := func(mappings ...Mapping) Attribution {
		return Attribution{ID: "attr", Version: "1", Mappings: mappings}
	}
	cases := map[string]struct {
		attribution Attribution
		policy      AllocationPolicy
	}{
		"account":               {attr(Mapping{"", "A"}), bear(BearMSP, "")},
		"customer":              {attr(Mapping{"acct-a", ""}), bear(BearMSP, "")},
		"account twice":         {attr(Mapping{"acct-a", "A"}, Mapping{"acct-a", "B"}), bear(BearMSP, "")},
		"attribution id":        {Attribution{ID: "", Version: "1"}, bear(BearMSP, "")},
		"attribution version":   {Attribution{ID: "attr", Version: ""}, bear(BearMSP, "")},
		"policy id":             {accounts(), AllocationPolicy{ID: "", Version: "1", Bearing: BearMSP}},
		"policy version":        {accounts(), AllocationPolicy{ID: "alloc", Version: "", Bearing: BearMSP}},
		"bearing":               {accounts(), bear("shared", "")},
		"customer of another":   {accounts(), bear(BearMSP, "A")},
		"customer missing":      {accounts(), bear(BearCustomer, "")},
		"customer outside form": {accounts(), bear(BearCustomer, "a b")},
	}
	for name, c := range cases {
		if _, err := Allocate(p, c.attribution, c.policy); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	four := attr(Mapping{"acct-a", "A"}, Mapping{"acct-b", "B"}, Mapping{"acct-c", "C"}, Mapping{"acct-d", "D"})
	if _, err := Allocate(p, four, bear(BearMSP, "")); err != nil {
		t.Fatalf("mappings at their limit: %v", err)
	}
}

func TestAmortizedPartsInSavingsAndBasis(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentPartial, "100", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "5", "10", used("l1", "acct-a", "5", "5", "8"), unused("l2", "0", "5"))
	p := sealed(t, r, "c1", "2026-01", 1)
	a, err := Allocate(p, accounts(), bear(BearProRataUsed, ""))
	if err != nil || a.View().Hold != "" || a.View().Customers[0].NegativeSavings != 1 || a.View().Customers[0].UnusedAmortized != "5" {
		t.Fatalf("an amortized part in savings and in the basis: %v %+v", err, a.View())
	}
	write(t, r, "c1", "2026-02", 1, "0", "5", used("l3", "acct-a", "0", "0", "1"), unused("l4", "0", "5"))
	q := sealed(t, r, "c1", "2026-02", 1)
	if a, _ := Allocate(q, accounts(), bear(BearProRataUsed, "")); a.View().Hold != HoldNoBasis {
		t.Fatalf("an amortized unused part without basis: %+v", a.View())
	}
}

func TestSaleInputsAtTheirBounds(t *testing.T) {
	_, p := example(t)
	a, _ := Allocate(p, accounts(), bear(BearCustomer, "C"))
	for _, text := range []string{"1.", ".5", "0.5.5", "01", "0.0a", "", "00.5", "0..5"} {
		if _, err := Sell(a, []SalePolicy{share("A", text)}); kind(err) != "invalid" {
			t.Errorf("share %q: %v", text, err)
		}
	}
	if _, err := Sell(a, []SalePolicy{costPlus("A", "", false)}); kind(err) != "invalid" {
		t.Error("an empty fee")
	}
	for name, policy := range map[string]SalePolicy{
		"id":      {Customer: "A", ID: "", Version: "1", Method: MethodCostPlus, Fee: "0", Mode: "half_even"},
		"version": {Customer: "A", ID: "s", Version: "", Method: MethodCostPlus, Fee: "0", Mode: "half_even"},
	} {
		if _, err := Sell(a, []SalePolicy{policy}); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	four := []SalePolicy{costPlus("A", "0", false), costPlus("B", "0", false), costPlus("C", "0", false)}
	if _, err := Sell(a, four); err != nil {
		t.Fatalf("three policies: %v", err)
	}
	limits := testLimits()
	limits.Digits = 4
	r, _ := NewRegistry(testProfile(t), limits, func() time.Time { return start })
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-03", 1, "100", "0", used("l1", "acct-a", "60", "0", "100"), used("l2", "acct-b", "25", "0", "50"), unused("l3", "15", "0"))
	m := sealed(t, r, "c1", "2026-03", 1)
	b, _ := Allocate(m, accounts(), bear(BearMSP, ""))
	if s, err := Sell(b, []SalePolicy{costPlus("A", "0.123", false), costPlus("B", "0", false)}); err != nil || s.Charges[0].Exact != "67.38" {
		t.Fatalf("an exact charge of four digits: %v %+v", err, s.Charges)
	}
}

func TestFourCustomers(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "4", "0", used("1", "acct-a", "1", "0", "1"), used("2", "acct-b", "1", "0", "1"), used("3", "acct-c", "1", "0", "1"), used("4", "acct-d", "1", "0", "1"))
	q := sealed(t, r, "c1", "2026-01", 1)
	attribution := Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{"acct-a", "A"}, {"acct-b", "B"}, {"acct-c", "C"}, {"acct-d", "D"}}}
	a, err := Allocate(q, attribution, bear(BearMSP, ""))
	if err != nil || len(a.View().Customers) != 4 {
		t.Fatalf("four customers: %v", err)
	}
	if _, err := Sell(a, []SalePolicy{costPlus("A", "0", false), costPlus("B", "0", false), costPlus("C", "0", false), costPlus("D", "0", false)}); err != nil {
		t.Fatalf("four policies: %v", err)
	}
}
