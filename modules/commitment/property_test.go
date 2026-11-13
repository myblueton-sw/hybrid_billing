package commitment

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func ratOf(text string) *big.Rat {
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return value
}

func roundRat(value *big.Rat, scale int, mode string) *big.Rat {
	factor := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	scaled := new(big.Rat).Mul(value, factor)
	floor := new(big.Int).Div(scaled.Num(), scaled.Denom())
	part := new(big.Rat).Sub(scaled, new(big.Rat).SetInt(floor))
	result := floor
	if part.Sign() != 0 {
		half := part.Cmp(big.NewRat(1, 2))
		up := map[string]bool{"ceiling": true, "toward_zero": scaled.Sign() < 0, "away_from_zero": scaled.Sign() > 0,
			"half_up": half > 0 || (half == 0 && scaled.Sign() > 0), "half_even": half > 0 || (half == 0 && floor.Bit(0) == 1)}[mode]
		if up {
			result = new(big.Int).Add(floor, big.NewInt(1))
		}
	}
	return new(big.Rat).Quo(new(big.Rat).SetInt(result), factor)
}

func TestRandomPeriodsAgainstAReference(t *testing.T) {
	random := rand.New(rand.NewSource(2100))
	modes := []string{"toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"}
	amounts := []string{"0", "1", "2.5", "10.005", "33.333", "60", "0.001", "99.99"}
	met := map[string]int{}
	for round := 0; round < 1500; round++ {
		limits := testLimits()
		limits.Lines, limits.Customers = 20, 6
		r, _ := NewRegistry(testProfile(t), limits, func() time.Time { return start })
		payment := []string{PaymentNone, PaymentPartial, PaymentAll}[random.Intn(3)]
		prepaid, recovery := "0", RecoveryCharged
		if payment != PaymentNone {
			prepaid = "100000"
			recovery = []string{RecoveryCharged, RecoveryRecovered}[random.Intn(2)]
		}
		c := commitmentOf("c1", payment, prepaid, recovery)
		c.Quantum = []string{"1", "0.01", "0.001"}[random.Intn(3)]
		r.RecordCommitment("ws", c)
		lines := []Line{}
		recurring, amortized := new(big.Rat), new(big.Rat)
		for index := 0; index < 1+random.Intn(8); index++ {
			rec, am := amounts[random.Intn(len(amounts))], "0"
			if payment != PaymentNone {
				am = amounts[random.Intn(len(amounts))]
			}
			recurring.Add(recurring, ratOf(rec))
			amortized.Add(amortized, ratOf(am))
			source := fmt.Sprintf("s%d", index)
			if random.Intn(3) == 0 {
				lines = append(lines, unused(source, rec, am))
				continue
			}
			account := []string{"acct-a", "acct-b", "acct-c"}[random.Intn(3)]
			lines = append(lines, used(source, account, rec, am, amounts[random.Intn(len(amounts))]))
		}
		if _, err := r.RecordPeriod("ws", "c1", "2026-05", 1, Facts{Recurring: recurring.FloatString(3), Amortized: amortized.FloatString(3), Lines: lines}); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if state, _ := r.Seal("ws", "c1", "2026-05", 1); state.State != StateSealed {
			t.Fatalf("round %d: %+v", round, state)
		}
		p, _ := r.Revision("ws", "c1", "2026-05", 1)
		attribution := Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{Account: "acct-a", Customer: "A"}, {Account: "acct-b", Customer: "B"}, {Account: "acct-c", Customer: "C"}}}
		dropped := random.Intn(10) == 0
		if dropped {
			attribution.Mappings = attribution.Mappings[:2]
		}
		usedBy, listBy := map[string]*big.Rat{}, map[string]*big.Rat{}
		unusedTotal, cUsed := new(big.Rat), false
		for _, line := range lines {
			if line.Kind == LineUnused {
				unusedTotal.Add(unusedTotal, new(big.Rat).Add(ratOf(line.Recurring), ratOf(line.Amortized)))
				continue
			}
			name := map[string]string{"acct-a": "A", "acct-b": "B", "acct-c": "C"}[line.Account]
			cUsed = cUsed || name == "C"
			if usedBy[name] == nil {
				usedBy[name], listBy[name] = new(big.Rat), new(big.Rat)
			}
			usedBy[name].Add(usedBy[name], new(big.Rat).Add(ratOf(line.Recurring), ratOf(line.Amortized)))
			listBy[name].Add(listBy[name], ratOf(line.List))
		}
		bearing := []string{BearMSP, BearCustomer, BearProRataUsed, BearUnattributed}[random.Intn(4)]
		customer := ""
		if bearing == BearCustomer {
			customer = []string{"A", "D"}[random.Intn(2)]
		}
		a, err := Allocate(p, attribution, bear(bearing, customer))
		if err != nil {
			t.Fatalf("round %d allocate: %v", round, err)
		}
		v := a.View()
		met["bearing "+bearing+" hold "+v.Hold]++
		if dropped && cUsed {
			if v.Hold != HoldAccountNotAttributed || len(v.Customers) != 0 {
				t.Fatalf("round %d: an account left out gives %q", round, v.Hold)
			}
			continue
		}
		if v.Hold != "" {
			continue
		}
		basis := new(big.Rat)
		for _, used := range usedBy {
			basis.Add(basis, used)
		}
		borne := new(big.Rat)
		for _, part := range v.Customers {
			if used := usedBy[part.Customer]; used != nil {
				if new(big.Rat).Add(ratOf(part.UsedRecurring), ratOf(part.UsedAmortized)).Cmp(used) != 0 || ratOf(part.List).Cmp(listBy[part.Customer]) != 0 {
					t.Fatalf("round %d %s: used cost not the sum of its lines", round, part.Customer)
				}
			} else if bearing != BearCustomer || part.Customer != customer {
				t.Fatalf("round %d: a customer without lines: %s", round, part.Customer)
			}
			share := new(big.Rat).Add(ratOf(part.UnusedRecurring), ratOf(part.UnusedAmortized))
			borne.Add(borne, share)
			if bearing == BearProRataUsed && basis.Sign() > 0 && usedBy[part.Customer] != nil {
				exactShare := new(big.Rat).Quo(new(big.Rat).Mul(unusedTotal, usedBy[part.Customer]), basis)
				quantum := ratOf(c.Quantum)
				low := new(big.Rat).Sub(exactShare, new(big.Rat).Mul(quantum, big.NewRat(3, 1)))
				high := new(big.Rat).Add(exactShare, new(big.Rat).Mul(quantum, big.NewRat(2, 1)))
				if share.Cmp(low) < 0 || share.Cmp(high) > 0 {
					t.Fatalf("round %d %s: a pro-rata share %s far from %s", round, part.Customer, share.FloatString(4), exactShare.FloatString(4))
				}
			}
		}
		mspPart := new(big.Rat).Add(ratOf(v.MSPRecurring), ratOf(v.MSPAmortized))
		unattributedPart := new(big.Rat).Add(ratOf(v.UnattributedRecurring), ratOf(v.UnattributedAmortized))
		expected := map[string][3]*big.Rat{BearMSP: {unusedTotal, new(big.Rat), new(big.Rat)}, BearUnattributed: {new(big.Rat), unusedTotal, new(big.Rat)}, BearCustomer: {new(big.Rat), new(big.Rat), unusedTotal}}
		if want, ok := expected[bearing]; ok && (mspPart.Cmp(want[0]) != 0 || unattributedPart.Cmp(want[1]) != 0 || borne.Cmp(want[2]) != 0) {
			t.Fatalf("round %d: the bearer under %s", round, bearing)
		}
		total := new(big.Rat)
		for _, text := range []string{v.MSPRecurring, v.MSPAmortized, v.RoundingRecurring, v.RoundingAmortized, v.UnattributedRecurring, v.UnattributedAmortized} {
			total.Add(total, ratOf(text))
		}
		for _, part := range v.Customers {
			for _, text := range []string{part.UsedRecurring, part.UsedAmortized, part.UnusedRecurring, part.UnusedAmortized} {
				total.Add(total, ratOf(text))
			}
			if ratOf(part.UnusedRecurring).Sign() < 0 || ratOf(part.UnusedAmortized).Sign() < 0 {
				t.Fatalf("round %d: a negative share", round)
			}
		}
		if total.Cmp(new(big.Rat).Add(recurring, amortized)) != 0 || ratOf(v.RoundingRecurring).Sign() < 0 || ratOf(v.RoundingAmortized).Sign() < 0 {
			t.Fatalf("round %d: parts %s against cost %s", round, total.FloatString(6), new(big.Rat).Add(recurring, amortized).FloatString(6))
		}
		if ratOf(v.RoundingRecurring).Sign() > 0 {
			met["rounding part"]++
		}
		policies := []SalePolicy{}
		for _, part := range v.Customers {
			if random.Intn(8) == 0 {
				continue
			}
			mode := modes[random.Intn(6)]
			if random.Intn(2) == 0 {
				policies = append(policies, SalePolicy{Customer: part.Customer, ID: "s", Version: "1", Method: MethodCostPlus, Fee: []string{"0", "0.1", "0.25"}[random.Intn(3)], FeeOnUnused: random.Intn(2) == 0, Mode: mode})
			} else {
				policies = append(policies, SalePolicy{Customer: part.Customer, ID: "s", Version: "1", Method: MethodSavingsShare, Share: []string{"0", "0.5", "1"}[random.Intn(3)], Mode: mode})
			}
		}
		s, err := Sell(a, policies)
		if err != nil {
			t.Fatalf("round %d sell: %v", round, err)
		}
		scale := quanta[c.Quantum]
		sum, priced := new(big.Rat), true
		for index, charge := range s.Charges {
			part := v.Customers[index]
			var policy *SalePolicy
			for at := range policies {
				if policies[at].Customer == part.Customer {
					policy = &policies[at]
				}
			}
			usedCost := new(big.Rat).Add(ratOf(part.UsedRecurring), ratOf(part.UsedAmortized))
			unusedCost := new(big.Rat).Add(ratOf(part.UnusedRecurring), ratOf(part.UnusedAmortized))
			if recovery == RecoveryRecovered {
				usedCost, unusedCost = ratOf(part.UsedRecurring), ratOf(part.UnusedRecurring)
			}
			wantHold := ""
			var want *big.Rat
			switch {
			case policy == nil:
				wantHold = HoldNoSalePolicy
			case policy.Method == MethodCostPlus:
				factor := new(big.Rat).Add(big.NewRat(1, 1), ratOf(policy.Fee))
				want = new(big.Rat).Mul(usedCost, factor)
				if policy.FeeOnUnused {
					want.Add(want, new(big.Rat).Mul(unusedCost, factor))
				} else {
					want.Add(want, unusedCost)
				}
			case recovery == RecoveryRecovered:
				wantHold = HoldRecoveredWithSavingsShare
			case part.NegativeSavings > 0:
				wantHold = HoldNegativeSavings
			default:
				want = new(big.Rat).Sub(ratOf(part.List), new(big.Rat).Mul(ratOf(part.Savings), ratOf(policy.Share)))
				want.Add(want, unusedCost)
			}
			if wantHold != "" {
				priced = false
				if charge.Hold != wantHold {
					t.Fatalf("round %d %s: hold %q, the reference %q", round, part.Customer, charge.Hold, wantHold)
				}
				met["hold "+wantHold]++
				continue
			}
			rounded := roundRat(want, scale, policy.Mode)
			if ratOf(charge.Amount).Cmp(rounded) != 0 {
				t.Fatalf("round %d %s: %s, the reference %s", round, part.Customer, charge.Amount, rounded.FloatString(scale))
			}
			sum.Add(sum, rounded)
			met["charge "+policy.Method]++
		}
		compared := new(big.Rat).Add(recurring, amortized)
		if recovery == RecoveryRecovered {
			compared = new(big.Rat).Set(recurring)
		}
		if ratOf(s.Total).Cmp(sum) != 0 || ratOf(s.CostCompared).Cmp(compared) != 0 || s.AllCustomersPriced != priced {
			t.Fatalf("round %d: totals %+v", round, s)
		}
		if priced {
			result := new(big.Rat).Sub(sum, compared)
			if ratOf(s.MSPResult).Cmp(result) != 0 || s.BelowCost != (result.Sign() < 0) {
				t.Fatalf("round %d: result %s", round, s.MSPResult)
			}
			met[fmt.Sprintf("below %v", s.BelowCost)]++
		} else if s.MSPResult != "" || s.BelowCost {
			t.Fatalf("round %d: a result while held", round)
		}
	}
	for _, name := range []string{"rounding part", "hold no_sale_policy", "hold negative_savings", "hold recovered_with_savings_share", "charge cost_plus", "charge savings_share", "below true", "below false",
		"bearing pro_rata_used hold no_basis", "bearing msp hold account_not_attributed", "bearing msp hold ", "bearing customer hold ", "bearing unattributed hold ", "bearing pro_rata_used hold "} {
		if met[name] < 5 {
			t.Errorf("%s met %d times", name, met[name])
		}
	}
}
