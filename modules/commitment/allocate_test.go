package commitment

import "testing"

func TestBearingPoliciesAndTheRoundingPart(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentPartial, "12", RecoveryCharged))
	write(t, r, "c1", "2026-03", 1, "100.005", "0.007", used("l1", "acct-a", "60", "0", "100"), used("l2", "acct-b", "25", "0", "50"), unused("l3", "15.005", "0.007"))
	p := sealed(t, r, "c1", "2026-03", 1)
	a, err := Allocate(p, accounts(), bear(BearProRataUsed, ""))
	if err != nil {
		t.Fatal(err)
	}
	v := a.View()
	if v.Customers[0].UnusedRecurring != "10.59" || v.Customers[1].UnusedRecurring != "4.41" || v.RoundingRecurring != "0.005" ||
		v.Customers[0].UnusedAmortized != "0" || v.RoundingAmortized != "0.007" || v.MSPRecurring != "0" {
		t.Fatalf("pro rata: %+v", v)
	}
	for bearing, want := range map[string][3]string{BearMSP: {"15.005", "0", "0"}, BearUnattributed: {"0", "15.005", "0"}, BearCustomer: {"0", "0", "15.005"}} {
		customer := ""
		if bearing == BearCustomer {
			customer = "C"
		}
		a, err := Allocate(p, accounts(), bear(bearing, customer))
		if err != nil {
			t.Fatal(err)
		}
		v := a.View()
		got := [3]string{v.MSPRecurring, v.UnattributedRecurring, "0"}
		if bearing == BearCustomer {
			got[2] = v.Customers[2].UnusedRecurring
			if v.Customers[2].Customer != "C" || v.Customers[2].UsedRecurring != "0" {
				t.Fatalf("a customer bearing unused cost only: %+v", v.Customers)
			}
		}
		if got != want {
			t.Errorf("%s: %v", bearing, got)
		}
	}
}

func TestAllocationHolds(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "15", "0", used("l1", "acct-x", "0", "0", "5"), unused("l2", "15", "0"))
	p := sealed(t, r, "c1", "2026-01", 1)
	a, _ := Allocate(p, accounts(), bear(BearProRataUsed, ""))
	if a.View().Hold != HoldAccountNotAttributed {
		t.Fatalf("an account not attributed before no basis: %s", a.View().Hold)
	}
	if _, err := Sell(a, nil); kind(err) != "invalid" {
		t.Fatalf("a held allocation sold: %v", err)
	}
	mapped := Attribution{ID: "attr", Version: "2", Mappings: []Mapping{{Account: "acct-x", Customer: "X"}, {Account: "acct-unused", Customer: "Y"}}}
	a, _ = Allocate(p, mapped, bear(BearProRataUsed, ""))
	if a.View().Hold != HoldNoBasis {
		t.Fatalf("utilisation zero: %s", a.View().Hold)
	}
	write(t, r, "c1", "2026-02", 1, "0", "0", used("l3", "acct-x", "0", "0", "0"))
	q := sealed(t, r, "c1", "2026-02", 1)
	if a, err := Allocate(q, mapped, bear(BearProRataUsed, "")); err != nil || a.View().Hold != "" || len(a.View().Customers) != 1 || a.View().Customers[0].Customer != "X" {
		t.Fatalf("nothing to split: %v", err)
	}
	if _, err := Allocate(Period{}, accounts(), bear(BearMSP, "")); kind(err) != "invalid" {
		t.Fatal("a zero period")
	}
	if _, err := Sell(Allocation{}, nil); kind(err) != "invalid" {
		t.Fatal("a zero allocation")
	}
}

func TestNegativeSavings(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "70", "0", used("l1", "acct-a", "60", "0", "50"), used("l2", "acct-b", "10", "0", "20"))
	p := sealed(t, r, "c1", "2026-01", 1)
	a, _ := Allocate(p, accounts(), bear(BearMSP, ""))
	if v := a.View(); v.Customers[0].Savings != "-10" || v.Customers[0].NegativeSavings != 1 || v.Customers[1].NegativeSavings != 0 {
		t.Fatalf("negative savings: %+v", v.Customers)
	}
	s, _ := Sell(a, []SalePolicy{share("A", "0.5"), share("B", "0.5")})
	if got := charges(s); got != "A:negative_savings B=15 | total 15 cost 70 recovered 0 result  below false priced false" {
		t.Fatalf("savings share: %s", got)
	}
	s, _ = Sell(a, []SalePolicy{costPlus("A", "0", false), costPlus("B", "0", false)})
	if got := charges(s); got != "A=60 B=10 | total 70 cost 70 recovered 0 result 0 below false priced true" {
		t.Fatalf("cost plus: %s", got)
	}
}
