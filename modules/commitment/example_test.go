package commitment

import "testing"

func sell(t testing.TB, p Period, policy AllocationPolicy, policies ...SalePolicy) string {
	t.Helper()
	a, err := Allocate(p, accounts(), policy)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	s, err := Sell(a, policies)
	if err != nil {
		t.Fatalf("sell: %v", err)
	}
	return charges(s)
}

func TestTheSourceExample(t *testing.T) {
	_, p := example(t)
	cases := []struct {
		name     string
		policy   AllocationPolicy
		policies []SalePolicy
		want     string
	}{
		{"cost plus 10% with the MSP bearing unused cost", bear(BearMSP, ""), []SalePolicy{costPlus("A", "0.1", false), costPlus("B", "0.1", false)},
			"A=66 B=27.5 | total 93.5 cost 100 recovered 0 result -6.5 below true priced true"},
		{"alternative 1, A bearing unused cost without a fee on it", bear(BearCustomer, "A"), []SalePolicy{costPlus("A", "0.1", false), costPlus("B", "0.1", false)},
			"A=81 B=27.5 | total 108.5 cost 100 recovered 0 result 8.5 below false priced true"},
		{"alternative 1 with the fee on unused cost", bear(BearCustomer, "A"), []SalePolicy{costPlus("A", "0.1", true), costPlus("B", "0.1", true)},
			"A=82.5 B=27.5 | total 110 cost 100 recovered 0 result 10 below false priced true"},
		{"alternative 2, half the savings passed on with the MSP bearing unused cost", bear(BearMSP, ""), []SalePolicy{share("A", "0.5"), share("B", "0.5")},
			"A=80 B=37.5 | total 117.5 cost 100 recovered 0 result 17.5 below false priced true"},
		{"a customer without a policy", bear(BearMSP, ""), []SalePolicy{costPlus("A", "0.1", false)},
			"A=66 B:no_sale_policy | total 66 cost 100 recovered 0 result  below false priced false"},
	}
	for _, c := range cases {
		if got := sell(t, p, c.policy, c.policies...); got != c.want {
			t.Errorf("%s: %s", c.name, got)
		}
	}
}

func TestAPrepaymentIsNotChargedAgain(t *testing.T) {
	for _, c := range []struct {
		recovery string
		want     string
	}{
		{RecoveryRecovered, "A=0 | total 0 cost 0 recovered 100 result 0 below false priced true"},
		{RecoveryCharged, "A=110 | total 110 cost 100 recovered 0 result 10 below false priced true"},
	} {
		r := open(t)
		record(t, r, commitmentOf("c1", PaymentAll, "1200", c.recovery))
		write(t, r, "c1", "2026-03", 1, "0", "100", used("l1", "acct-a", "0", "100", "150"))
		p := sealed(t, r, "c1", "2026-03", 1)
		if got := sell(t, p, bear(BearMSP, ""), costPlus("A", "0.1", false)); got != c.want {
			t.Errorf("%s: %s", c.recovery, got)
		}
		a, _ := Allocate(p, accounts(), bear(BearMSP, ""))
		s, err := Sell(a, []SalePolicy{share("A", "0.5")})
		if c.recovery == RecoveryRecovered && (err != nil || s.Charges[0].Hold != HoldRecoveredWithSavingsShare || s.MSPResult != "") {
			t.Errorf("savings shared after a recovered purchase: %v %+v", err, s)
		}
	}
}
