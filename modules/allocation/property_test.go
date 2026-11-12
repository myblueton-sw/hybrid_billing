package allocation

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const propertyCases = 3000

type drawn struct {
	total, quantum string
	mode           decimal.Mode
	ids, weights   []string
}

func draw(r *rand.Rand) drawn {
	quanta := []string{"0.01", "0.05", "0.25", "1", "3", "0.001"}
	modes := []decimal.Mode{decimal.TowardZero, decimal.AwayFromZero, decimal.Floor, decimal.Ceiling, decimal.HalfUp, decimal.HalfEven}
	d := drawn{quantum: quanta[r.IntN(len(quanta))], mode: modes[r.IntN(len(modes))]}
	switch r.IntN(6) {
	case 0:
		d.total = "0"
	case 1:
		d.total = strconv.Itoa(r.IntN(7)) + ".0" + strconv.Itoa(r.IntN(10))
	default:
		d.total = strconv.FormatInt(r.Int64N(1_000_000), 10) + "." + strconv.Itoa(1000 + r.IntN(9000))[1:]
	}
	if r.IntN(3) == 0 && d.total != "0" {
		d.total = "-" + d.total
	}
	count := 1 + r.IntN(9)
	tied := r.IntN(3) == 0
	shared := strconv.Itoa(1 + r.IntN(9))
	for index := 0; index < count; index++ {
		d.ids = append(d.ids, "t"+strconv.Itoa(r.IntN(1000))+"-"+strconv.Itoa(index))
		switch {
		case tied:
			d.weights = append(d.weights, shared)
		case r.IntN(6) == 0:
			d.weights = append(d.weights, "0")
		default:
			d.weights = append(d.weights, strconv.Itoa(r.IntN(500))+"."+strconv.Itoa(r.IntN(100)))
		}
	}
	return d
}

func build(t testing.TB, p decimal.Profile, d drawn, order []int, factor string) Request {
	t.Helper()
	list := make([]Target, len(order))
	for position, index := range order {
		weight := value(t, p, d.weights[index])
		if factor != "" {
			scaled, err := decimal.Mul(p, weight, value(t, p, factor))
			if err != nil {
				t.Fatal(err)
			}
			weight = scaled
		}
		list[position] = Target{ID: d.ids[index], Weight: weight}
	}
	return request(t, p, d.total, d.quantum, d.mode, list)
}

func oracle(t testing.TB, text string) *big.Rat {
	t.Helper()
	parsed, ok := new(big.Rat).SetString(text)
	if !ok {
		t.Fatalf("oracle cannot read %q", text)
	}
	return parsed
}

func fraction(x *big.Rat) *big.Rat {
	whole := new(big.Int).Quo(x.Num(), x.Denom())
	return new(big.Rat).Sub(x, new(big.Rat).SetInt(whole))
}

func TestAllocationIsCharacterizedByTheContract(t *testing.T) {
	p := synthetic(t)
	r := rand.New(rand.NewPCG(20261112, 1670))
	accepted, zeroWeightTotals, ties, negatives, zeroWeights, rescaled := 0, 0, 0, 0, 0, 0
	for index := 0; index < propertyCases; index++ {
		d := draw(r)
		identity := make([]int, len(d.ids))
		for position := range identity {
			identity[position] = position
		}
		result, err := LargestRemainderV1(build(t, p, d, identity, ""))
		sum := new(big.Rat)
		for _, weight := range d.weights {
			sum.Add(sum, oracle(t, weight))
		}
		if sum.Sign() == 0 {
			zeroWeightTotals++
			if !errors.Is(err, ErrZeroTotalWeight) {
				t.Fatalf("%+v: zero total weight error = %v", d, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%+v: %v", d, err)
		}
		accepted++
		quantum, original := oracle(t, d.quantum), oracle(t, d.total)
		quantized := oracle(t, result.Quantized.String())
		units := new(big.Rat).Quo(quantized, quantum)
		if !units.IsInt() || new(big.Rat).Abs(new(big.Rat).Sub(original, quantized)).Cmp(quantum) >= 0 {
			t.Fatalf("%+v: quantized %s is not within one quantum of the total", d, result.Quantized)
		}
		if new(big.Rat).Add(quantized, oracle(t, result.RoundingComponent.String())).Cmp(original) != 0 {
			t.Fatalf("%+v: original is not quantized plus component", d)
		}
		if oracle(t, result.Denominator.String()).Cmp(sum) != 0 {
			t.Fatalf("%+v: denominator %s; oracle %s", d, result.Denominator, sum.FloatString(6))
		}
		sign := big.NewRat(1, 1)
		if units.Sign() < 0 {
			sign = big.NewRat(-1, 1)
			negatives++
		}
		magnitude := new(big.Rat).Abs(units)
		weightOf := map[string]*big.Rat{}
		for position, id := range d.ids {
			weightOf[id] = oracle(t, d.weights[position])
		}
		total := new(big.Rat)
		type line struct {
			id     string
			extra  bool
			remain *big.Rat
		}
		lines := make([]line, 0, len(result.Shares))
		previous := ""
		for _, share := range result.Shares {
			if previous != "" && share.ID <= previous {
				t.Fatalf("%+v: shares are not in ascending identifier order", d)
			}
			previous = share.ID
			amount := oracle(t, share.Amount.String())
			total.Add(total, amount)
			count := new(big.Rat).Quo(new(big.Rat).Mul(amount, sign), quantum)
			exact := new(big.Rat).Quo(new(big.Rat).Mul(magnitude, weightOf[share.ID]), sum)
			floor := new(big.Rat).Sub(exact, fraction(exact))
			over := new(big.Rat).Sub(count, floor)
			if !count.IsInt() || (over.Sign() != 0 && over.Cmp(big.NewRat(1, 1)) != 0) {
				t.Fatalf("%+v: share %s is not the floor or the floor plus one quantum", d, share.ID)
			}
			if weightOf[share.ID].Sign() == 0 {
				zeroWeights++
				if amount.Sign() != 0 {
					t.Fatalf("%+v: zero weight %s received %s", d, share.ID, share.Amount)
				}
			}
			lines = append(lines, line{share.ID, over.Sign() != 0, fraction(exact)})
		}
		if total.Cmp(quantized) != 0 || len(result.Shares) != len(d.ids) {
			t.Fatalf("%+v: shares sum to %s; quantized %s", d, total.FloatString(6), result.Quantized)
		}
		for _, winner := range lines {
			for _, loser := range lines {
				if !winner.extra || loser.extra {
					continue
				}
				order := winner.remain.Cmp(loser.remain)
				if order == 0 {
					ties++
				}
				if order < 0 || (order == 0 && winner.id > loser.id) {
					t.Fatalf("%+v: %s received a quantum ahead of %s", d, winner.id, loser.id)
				}
			}
		}
		permuted := r.Perm(len(d.ids))
		again, err := LargestRemainderV1(build(t, p, d, permuted, ""))
		if err != nil || amounts(again) != amounts(result) {
			t.Fatalf("%+v: permuted input gives %s, %v; want %s", d, amounts(again), err, amounts(result))
		}
		factor := []string{"2", "7", "0.5", "12.5"}[r.IntN(4)]
		scaled, err := LargestRemainderV1(build(t, p, d, identity, factor))
		if err != nil || amounts(scaled) != amounts(result) {
			t.Fatalf("%+v: weights times %s give %s, %v; want %s", d, factor, amounts(scaled), err, amounts(result))
		}
		rescaled++
		mirror := map[decimal.Mode]decimal.Mode{decimal.Floor: decimal.Ceiling, decimal.Ceiling: decimal.Floor}
		opposite := d
		opposite.total = strings.TrimPrefix("-"+d.total, "--")
		if other, ok := mirror[d.mode]; ok {
			opposite.mode = other
		}
		inverse, err := LargestRemainderV1(build(t, p, opposite, identity, ""))
		if err != nil {
			t.Fatalf("%+v: sign inverse: %v", d, err)
		}
		for position, share := range inverse.Shares {
			if new(big.Rat).Add(oracle(t, share.Amount.String()), oracle(t, result.Shares[position].Amount.String())).Sign() != 0 {
				t.Fatalf("%+v: inverse total does not give inverse shares: %s against %s", d, amounts(inverse), amounts(result))
			}
		}
	}
	if accepted < propertyCases/2 || zeroWeightTotals == 0 || ties == 0 || negatives == 0 || zeroWeights == 0 || rescaled == 0 {
		t.Fatalf("generator coverage too thin: accepted %d zero-weight pools %d ties %d negatives %d zero weights %d rescaled %d", accepted, zeroWeightTotals, ties, negatives, zeroWeights, rescaled)
	}
	t.Logf("seed 20261112/1670 cases %d accepted %d zero-weight pools %d tie comparisons %d negative totals %d zero weights %d", propertyCases, accepted, zeroWeightTotals, ties, negatives, zeroWeights)
}
