package decimal

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const propertyCases = 4000

func source() *rand.Rand { return rand.New(rand.NewPCG(20261112, 167)) }

func drawText(r *rand.Rand) string {
	edges := []string{"0", "0.00000001", "-0.00000001", "999999999999.99999999", "-999999999999.99999999", "1", "-1", "0.5", "2.5", "-2.5", "0.125"}
	if r.IntN(8) == 0 {
		return edges[r.IntN(len(edges))]
	}
	whole := strconv.FormatInt(r.Int64N(1_000_000_000_000), 10)
	if r.IntN(3) == 0 {
		whole = strconv.Itoa(r.IntN(100))
	}
	text := whole
	if scale := r.IntN(9); scale > 0 {
		fraction := strconv.FormatInt(r.Int64N(100_000_000), 10)
		fraction = strings.Repeat("0", 8-len(fraction)) + fraction
		text += "." + fraction[:scale]
	}
	if r.IntN(3) == 0 {
		text = "-" + text
	}
	return text
}

func rat(t testing.TB, text string) *big.Rat {
	t.Helper()
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		t.Fatalf("oracle cannot read %q", text)
	}
	return value
}

func representable(value *big.Rat) bool {
	limit := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	if new(big.Rat).Abs(value).Cmp(limit) >= 0 {
		return false
	}
	shifted := new(big.Rat).Mul(value, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)))
	return shifted.IsInt()
}

func TestExactOperationsAgreeWithRationalArithmetic(t *testing.T) {
	p := synthetic(t)
	r := source()
	accepted, rejected := 0, 0
	operations := []struct {
		name   string
		run    func(Profile, Decimal, Decimal) (Decimal, error)
		oracle func(a, b *big.Rat) *big.Rat
	}{
		{"add", Add, func(a, b *big.Rat) *big.Rat { return new(big.Rat).Add(a, b) }},
		{"sub", Sub, func(a, b *big.Rat) *big.Rat { return new(big.Rat).Sub(a, b) }},
		{"mul", Mul, func(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }},
		{"quo", Quo, func(a, b *big.Rat) *big.Rat {
			if b.Sign() == 0 {
				return nil
			}
			return new(big.Rat).Quo(a, b)
		}},
	}
	for index := 0; index < propertyCases; index++ {
		left, right := drawText(r), drawText(r)
		operation := operations[index%len(operations)]
		got, err := operation.run(p, parse(t, p, left), parse(t, p, right))
		want := operation.oracle(rat(t, left), rat(t, right))
		switch {
		case err == nil:
			accepted++
			if want == nil || rat(t, got.String()).Cmp(want) != 0 || !representable(want) {
				t.Fatalf("%s(%s, %s) = %s; oracle %v", operation.name, left, right, got, want)
			}
			if strings.Contains(got.String(), ".") && strings.HasSuffix(got.String(), "0") || got.String() == "-0" {
				t.Fatalf("%s(%s, %s) = %s is not canonical", operation.name, left, right, got)
			}
		case errors.Is(err, ErrDivisionByZero):
			rejected++
			if want != nil {
				t.Fatalf("%s(%s, %s) reported division by zero", operation.name, left, right)
			}
		case errors.Is(err, ErrResultBound), errors.Is(err, ErrInexactDivision):
			rejected++
			if want != nil && representable(want) {
				t.Fatalf("%s(%s, %s) rejected a representable result %s: %v", operation.name, left, right, want.FloatString(14), err)
			}
		default:
			t.Fatalf("%s(%s, %s) unexpected error %v", operation.name, left, right, err)
		}
	}
	if accepted < propertyCases/4 || rejected == 0 {
		t.Fatalf("generator coverage too thin: accepted %d rejected %d", accepted, rejected)
	}
	t.Logf("seed 20261112/167 cases %d accepted %d rejected %d", propertyCases, accepted, rejected)
}

func checkRounded(t *testing.T, label string, exact, rounded, unit *big.Rat, mode Mode) (tie bool) {
	t.Helper()
	if steps := new(big.Rat).Quo(rounded, unit); !steps.IsInt() {
		t.Fatalf("%s: %s is not a multiple of the unit", label, rounded.FloatString(14))
	}
	low, high := new(big.Rat).Sub(rounded, unit), new(big.Rat).Add(rounded, unit)
	distance := new(big.Rat).Abs(new(big.Rat).Sub(exact, rounded))
	half := new(big.Rat).Quo(unit, big.NewRat(2, 1))
	absoluteExact, absoluteRounded := new(big.Rat).Abs(exact), new(big.Rat).Abs(rounded)
	ok := false
	switch mode {
	case Floor:
		ok = rounded.Cmp(exact) <= 0 && exact.Cmp(high) < 0
	case Ceiling:
		ok = low.Cmp(exact) < 0 && exact.Cmp(rounded) <= 0
	case TowardZero:
		ok = absoluteRounded.Cmp(absoluteExact) <= 0 && distance.Cmp(unit) < 0 && rounded.Sign()*exact.Sign() >= 0
	case AwayFromZero:
		ok = absoluteRounded.Cmp(absoluteExact) >= 0 && distance.Cmp(unit) < 0 && rounded.Sign()*exact.Sign() >= 0
	case HalfUp:
		tie = distance.Cmp(half) == 0
		ok = distance.Cmp(half) < 0 || (tie && absoluteRounded.Cmp(absoluteExact) > 0)
	case HalfEven:
		tie = distance.Cmp(half) == 0
		steps := new(big.Rat).Quo(rounded, unit)
		ok = distance.Cmp(half) < 0 || (tie && steps.Num().Bit(0) == 0)
	}
	if !ok {
		t.Fatalf("%s: %s does not satisfy %s for exact %s", label, rounded.FloatString(14), mode, exact.FloatString(14))
	}
	return tie
}

func TestRoundingSatisfiesItsDefinition(t *testing.T) {
	p := synthetic(t)
	r := source()
	modes := []Mode{TowardZero, AwayFromZero, Floor, Ceiling, HalfUp, HalfEven}
	quanta := []string{"0.01", "0.05", "0.25", "3", "0.125", "1"}
	ties, negatives, divisions := 0, 0, 0
	for index := 0; index < propertyCases; index++ {
		text, mode := drawText(r), modes[r.IntN(len(modes))]
		value := parse(t, p, text)
		if strings.HasPrefix(text, "-") {
			negatives++
		}
		switch index % 3 {
		case 0:
			scale := r.IntN(5)
			got, err := Round(p, value, scale, mode)
			if err != nil {
				t.Fatalf("Round(%s, %d, %s): %v", text, scale, mode, err)
			}
			unit := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
			if checkRounded(t, "round "+text, rat(t, text), rat(t, got.String()), unit, mode) {
				ties++
			}
		case 1:
			quantum := quanta[r.IntN(len(quanta))]
			got, err := RoundToQuantum(p, value, parse(t, p, quantum), mode)
			if err != nil {
				t.Fatalf("RoundToQuantum(%s, %s, %s): %v", text, quantum, mode, err)
			}
			if checkRounded(t, "quantum "+text+"@"+quantum, rat(t, text), rat(t, got.String()), rat(t, quantum), mode) {
				ties++
			}
		default:
			divisor := drawText(r)
			if rat(t, divisor).Sign() == 0 {
				continue
			}
			scale := r.IntN(7)
			got, err := DivideRound(p, value, parse(t, p, divisor), scale, mode)
			exact := new(big.Rat).Quo(rat(t, text), rat(t, divisor))
			if errors.Is(err, ErrResultBound) {
				if new(big.Rat).Abs(exact).Cmp(new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))) < 0 {
					limit := new(big.Rat).Sub(new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)), big.NewRat(1, 1))
					if new(big.Rat).Abs(exact).Cmp(limit) < 0 {
						t.Fatalf("DivideRound(%s, %s) rejected %s", text, divisor, exact.FloatString(14))
					}
				}
				continue
			}
			if err != nil {
				t.Fatalf("DivideRound(%s, %s, %d, %s): %v", text, divisor, scale, mode, err)
			}
			divisions++
			unit := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
			if checkRounded(t, "divide "+text+"/"+divisor, exact, rat(t, got.String()), unit, mode) {
				ties++
			}
		}
	}
	if ties == 0 || negatives == 0 || divisions == 0 {
		t.Fatalf("generator coverage too thin: ties %d negatives %d divisions %d", ties, negatives, divisions)
	}
	t.Logf("seed 20261112/167 cases %d ties %d negatives %d divisions %d", propertyCases, ties, negatives, divisions)
}

var grammar = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

func FuzzParse(f *testing.F) {
	for _, seed := range []string{"0", "-0", "0.10", "00.1", "1e3", "999999999999.99999999", "1234567890123", "0.123456789", "-12.50", ".5", "5.", "+1", "NaN", "１", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		p := synthetic(t)
		got, err := Parse(p, text)
		wellFormed := grammar.MatchString(text)
		inBounds := false
		if wellFormed {
			body := strings.TrimPrefix(text, "-")
			whole, fraction, _ := strings.Cut(body, ".")
			count := len(whole)
			if whole == "0" {
				count = 0
			}
			inBounds = count <= 12 && len(fraction) <= 8
		}
		if (err == nil) != (wellFormed && inBounds) {
			t.Fatalf("Parse(%q) error %v; grammar %v bounds %v", text, err, wellFormed, inBounds)
		}
		if err != nil {
			if got.Valid() {
				t.Fatalf("Parse(%q) returned a value with an error", text)
			}
			return
		}
		want, _ := new(big.Rat).SetString(text)
		back, ok := new(big.Rat).SetString(got.String())
		if !ok || back.Cmp(want) != 0 {
			t.Fatalf("Parse(%q) = %s; oracle %s", text, got, want)
		}
		printed := got.String()
		if printed == "-0" || (strings.Contains(printed, ".") && strings.HasSuffix(printed, "0")) || !grammar.MatchString(printed) {
			t.Fatalf("Parse(%q) printed non-canonical %q", text, printed)
		}
	})
}
