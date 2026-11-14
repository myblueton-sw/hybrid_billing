package components

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func ratOf(text string) *big.Rat {
	v, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return v
}

func roundModel(v *big.Rat, scale int, mode string) *big.Rat {
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	scaled := new(big.Rat).Mul(v, new(big.Rat).SetInt(factor))
	floor := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	rest := new(big.Rat).Sub(scaled, new(big.Rat).SetInt(floor))
	up := false
	switch mode {
	case "ceiling", "away_from_zero":
		up = rest.Sign() > 0
	case "half_up":
		up = rest.Cmp(big.NewRat(1, 2)) >= 0
	case "half_even":
		up = rest.Cmp(big.NewRat(1, 2)) > 0 || rest.Cmp(big.NewRat(1, 2)) == 0 && floor.Bit(0) == 1
	}
	if up {
		floor.Add(floor, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(floor, factor)
}

func TestStatementsAgainstAModel(t *testing.T) {
	meters := []string{"m0", "m1", "m2", "m3"}
	for seed := int64(1); seed <= 400; seed++ {
		rng := rand.New(rand.NewSource(seed))
		l := testLimits()
		l.Components, l.Inclusions = 20, 4
		r, _ := openWith(t, l)
		scale := rng.Intn(5)
		mode := []string{"toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"}[rng.Intn(6)]
		must(r.SetPricing("ws", Pricing{Contract: "k", Scale: scale, Mode: mode}, "admin"))
		fees := []*big.Rat{}
		expected := new(big.Rat)
		included := map[string]*big.Rat{}
		all := map[string]bool{}
		owner := map[string]bool{}
		rates := map[string]*big.Rat{}
		for i := 0; i < 1+rng.Intn(3); i++ {
			fee := fmt.Sprintf("%d", 1+rng.Intn(5000))
			if scale > 0 {
				fee += fmt.Sprintf(".%0*d", scale, rng.Intn(10))
			}
			fees = append(fees, ratOf(fee))
			var inc []Inclusion
			for _, m := range meters {
				if owner[m] || rng.Intn(3) != 0 {
					continue
				}
				owner[m] = true
				if rng.Intn(3) == 0 {
					inc = append(inc, Inclusion{Meter: m, Quantity: All})
					all[m] = true
				} else {
					q := fmt.Sprintf("%d.%d", 1+rng.Intn(500), rng.Intn(10))
					inc = append(inc, Inclusion{Meter: m, Quantity: q})
					included[m] = ratOf(q)
				}
			}
			approved(t, r, rack(fmt.Sprintf("f%d", i), fee, inc...))
			expected.Add(expected, ratOf(fee))
		}
		for _, m := range meters {
			if rng.Intn(2) == 0 {
				rate := fmt.Sprintf("0.%03d", rng.Intn(1000))
				approved(t, r, usage("u"+m, m, rate))
				rates[m] = ratOf(rate)
			}
		}
		var readings []Reading
		unpriced := new(big.Rat)
		for _, m := range meters {
			if rates[m] == nil && rng.Intn(2) == 0 {
				continue
			}
			q := fmt.Sprintf("%d.%03d", rng.Intn(1000), rng.Intn(1000))
			readings = append(readings, Reading{Meter: m, Quantity: q})
			measured := ratOf(q)
			inc := new(big.Rat)
			if included[m] != nil {
				inc = included[m]
			}
			billable := new(big.Rat)
			if !all[m] && measured.Cmp(inc) > 0 {
				billable.Sub(measured, inc)
			}
			if rates[m] != nil {
				expected.Add(expected, roundModel(new(big.Rat).Mul(billable, rates[m]), scale, mode))
			} else {
				unpriced.Add(unpriced, billable)
			}
		}
		credits := map[string]*big.Rat{}
		for i := range fees {
			if rng.Intn(2) == 0 {
				continue
			}
			comp := fmt.Sprintf("f%d", i)
			rate := fmt.Sprintf("0.%d", 1+rng.Intn(9))
			must(r.AddClause("ws", Clause{ID: "s" + comp, Contract: "k", Component: comp, Rate: rate, From: "2026-01-01"}, "legal"))
			must(r.ApproveClause("ws", "s"+comp, "lead"))
			must(r.RecordIncident("ws", Incident{ID: "i" + comp, Contract: "k", Component: comp, Start: start.Add(-48 * time.Hour), End: start.Add(-47 * time.Hour), Evidence: digest('a')}, "noc"))
			want := roundModel(new(big.Rat).Mul(fees[i], ratOf(rate)), scale, mode)
			v, err := r.ProposeCredit("ws", Credit{ID: "c" + comp, Clause: "s" + comp, Incidents: []string{"i" + comp}, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"}, "clerk")
			if want.Sign() == 0 {
				expect(t, ReasonZeroCredit, err, "zero credit")
				continue
			}
			if err != nil || ratOf(v.Amount).Cmp(want) != 0 {
				t.Fatalf("seed %d credit %s: %v %v", seed, v.Amount, want.FloatString(6), err)
			}
			switch rng.Intn(3) {
			case 0:
				must(r.WithdrawCredit("ws", "c"+comp, "clerk"))
			case 1:
				must(r.ApproveCredit("ws", "c"+comp, "ops"))
				credits["c"+comp] = want
			default:
				must(r.ApproveCredit("ws", "c"+comp, "ops"))
				must(r.LinkCredit("ws", "c"+comp, Link{Document: "d" + comp, Increase: "n" + comp}, "ar"))
				credits["c"+comp] = want
			}
		}
		rng.Shuffle(len(readings), func(a, b int) { readings[a], readings[b] = readings[b], readings[a] })
		s := must(r.Statement("ws", november(readings...)))
		if ratOf(s.Charges).Cmp(expected) != 0 || s.Net != s.Charges {
			t.Fatalf("seed %d: charges %s, model %s", seed, s.Charges, expected.FloatString(4))
		}
		if len(s.Credits) != len(credits) {
			t.Fatalf("seed %d: %d credits, model %d", seed, len(s.Credits), len(credits))
		}
		for _, c := range s.Credits {
			if credits[c.Credit] == nil || ratOf(c.Amount).Cmp(credits[c.Credit]) != 0 {
				t.Fatalf("seed %d: credit %+v", seed, c)
			}
		}
		sum := new(big.Rat)
		for _, line := range s.Lines {
			sum.Add(sum, ratOf(line.Amount))
		}
		if sum.Cmp(expected) != 0 {
			t.Fatalf("seed %d: lines %s", seed, sum.FloatString(4))
		}
		excess := new(big.Rat)
		for _, u := range s.Unpriced {
			excess.Add(excess, ratOf(u.Excess))
		}
		if excess.Cmp(unpriced) != 0 {
			t.Fatalf("seed %d: unpriced %s, model %s", seed, excess.FloatString(4), unpriced.FloatString(4))
		}
	}
}
