package marketplace

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
)

func randomAmount(rng *rand.Rand) string {
	units := rng.Int63n(10000000)
	scale := rng.Intn(5)
	text := new(big.Rat).SetFrac(big.NewInt(units), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)).FloatString(scale)
	return text
}

func randomBenefit(rng *rand.Rand) Benefit {
	switch rng.Intn(3) {
	case 0:
		return applied(randomAmount(rng))
	case 1:
		return notApplied()
	}
	return unknown()
}

func rat(text string) *big.Rat {
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return value
}

func referenceCharge(tx Transaction, facts Facts, p PolicyTerms) (string, map[string]*big.Rat, *big.Rat, *big.Rat) {
	if tx.Role == RoleSeller {
		return HoldSellerTransaction, nil, nil, nil
	}
	if tx.Collector != CollectorMSP {
		return HoldCollectedElsewhere, nil, nil, nil
	}
	rate := rat(p.Rate)
	scope := map[string]bool{}
	for _, k := range p.Scope {
		scope[k] = true
	}
	private := tx.Type == TypePrivate || tx.Type == TypeChannelPrivate
	marked := func(l Line) bool { return l.Discount.State == Applied || private && l.Kind == KindSoftware }
	if rate.Sign() > 0 {
		for _, l := range facts.Lines {
			if scope[l.Kind] && !marked(l) && l.Discount.State == Unknown {
				return HoldSourceUnknown, nil, nil, nil
			}
		}
	}
	quantum := rat(p.Quantum)
	out := map[string]*big.Rat{}
	total, charged := new(big.Rat), new(big.Rat)
	for _, l := range facts.Lines {
		value := rat(l.Charged)
		discount := new(big.Rat)
		if scope[l.Kind] && (marked(l) && p.SourceDiscounted == Included || !marked(l) && l.Discount.State == NotApplied) {
			raw := new(big.Rat).Mul(value, rate)
			steps := new(big.Rat).Quo(raw, quantum)
			whole := new(big.Int).Quo(steps.Num(), steps.Denom())
			discount = new(big.Rat).Mul(new(big.Rat).SetInt(whole), quantum)
		}
		amount := new(big.Rat).Sub(value, discount)
		out[l.Source] = amount
		total.Add(total, amount)
		charged.Add(charged, value)
	}
	return "", out, total, charged
}

func TestRandomChargesAgainstAReference(t *testing.T) {
	rng := rand.New(rand.NewSource(212))
	roles := []string{RoleBuyer, RoleReseller, RoleSeller}
	offers := []string{TypePublic, TypePrivate}
	kinds := []string{KindSoftware, KindInfrastructure, KindOther}
	seen := map[string]int{}
	for round := 0; round < 3000; round++ {
		role, offer := roles[rng.Intn(3)], offers[rng.Intn(2)]
		collector := CollectorMSP
		if role != RoleBuyer && rng.Intn(4) == 0 {
			collector = CollectorMarketplace
		}
		if role == RoleReseller && rng.Intn(3) == 0 {
			offer = TypeChannelPrivate
		}
		tx := deal("tx", role, ChannelMarketplace, offer, collector)
		facts := Facts{}
		for index := 0; index < 1+rng.Intn(5); index++ {
			l := line(fmt.Sprintf("l%d", index), kinds[rng.Intn(3)], randomAmount(rng), randomBenefit(rng))
			facts.Lines = append(facts.Lines, l)
		}
		if rng.Intn(2) == 0 {
			facts.Recognitions = []Recognition{{Program: "p", State: Unknown}}
		}
		rate := []string{"0", "0.1", "0.2", "0.333", "0.99999999", "0.05"}[rng.Intn(6)]
		source := Excluded
		if rate != "0" && rng.Intn(2) == 0 {
			source = Included
		}
		scope := []string{}
		for _, k := range kinds {
			if rng.Intn(2) == 0 {
				scope = append(scope, k)
			}
		}
		if len(scope) == 0 {
			scope = []string{KindSoftware}
		}
		p := terms("pol", "1", "tx", rate, scope, source)
		p.Quantum = []string{"1", "0.1", "0.01", "0.001", "0.0001"}[rng.Intn(5)]
		r := open(t)
		record(t, r, tx)
		write(t, r, "tx", "2026-03", 1, facts)
		approve(t, r, p)
		v := compute(t, sealed(t, r, "tx", "2026-03", 1), policyFor(t, r, "tx", "2026-03"))
		hold, want, total, charged := referenceCharge(tx, facts, p)
		seen[hold]++
		if v.Hold != hold {
			t.Fatalf("round %d hold %q want %q", round, v.Hold, hold)
		}
		if hold != "" {
			continue
		}
		if rat(v.Total).Cmp(total) != 0 || rat(v.ChargedTotal).Cmp(charged) != 0 || rat(v.Margin).Cmp(new(big.Rat).Sub(total, charged)) != 0 {
			t.Fatalf("round %d totals %s %s %s want %s %s", round, v.Total, v.ChargedTotal, v.Margin, total.RatString(), charged.RatString())
		}
		sum := new(big.Rat)
		for _, l := range v.Lines {
			amount := rat(l.Amount)
			if amount.Cmp(want[l.Source]) != 0 || amount.Sign() < 0 || amount.Cmp(rat(l.Charged)) > 0 {
				t.Fatalf("round %d line %+v want %s", round, l, want[l.Source].RatString())
			}
			if new(big.Rat).Add(amount, rat(l.Discount)).Cmp(rat(l.Charged)) != 0 {
				t.Fatalf("round %d discount %+v", round, l)
			}
			sum.Add(sum, amount)
		}
		if sum.Cmp(total) != 0 || total.Cmp(charged) > 0 {
			t.Fatalf("round %d sum", round)
		}
		if len(facts.Recognitions) > 0 {
			without := Facts{Lines: facts.Lines}
			q := open(t)
			record(t, q, tx)
			write(t, q, "tx", "2026-03", 1, without)
			approve(t, q, p)
			w := compute(t, sealed(t, q, "tx", "2026-03", 1), policyFor(t, q, "tx", "2026-03"))
			if w.Total != v.Total || amounts(w) != amounts(v) {
				t.Fatalf("round %d a recognition changed an amount", round)
			}
		}
	}
	for _, hold := range []string{"", HoldSellerTransaction, HoldCollectedElsewhere, HoldSourceUnknown} {
		if seen[hold] < 50 {
			t.Errorf("hold %q reached %d times", hold, seen[hold])
		}
	}
}
