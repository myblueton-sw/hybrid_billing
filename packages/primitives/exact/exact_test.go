package exact

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func profile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 9, InputScale: 4, ResultIntegerDigits: 18, ResultScale: 6, IntermediateDigits: 60})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func of(t testing.TB, text string) Fraction {
	t.Helper()
	value, err := decimal.Parse(profile(t), text)
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	made, ok := Of(value)
	if !ok {
		t.Fatalf("%s: not a fraction", text)
	}
	return made
}

func over(t testing.TB, top, bottom string) Fraction {
	t.Helper()
	made, ok := of(t, top).Over(of(t, bottom))
	if !ok {
		t.Fatalf("%s over %s", top, bottom)
	}
	return made
}

func TestLiteralArithmetic(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"a whole number", Whole(-7).Text(), "-7"},
		{"a decimal", of(t, "2.50").Text(), "5/2"},
		{"zero", of(t, "0.000").Text(), "0"},
		{"a sum", of(t, "0.5").Plus(over(t, "1", "3")).Text(), "5/6"},
		{"a sum that reduces", over(t, "1", "6").Plus(over(t, "1", "3")).Text(), "1/2"},
		{"a difference", of(t, "0.5").Minus(over(t, "2", "3")).Text(), "-1/6"},
		{"a product", over(t, "2", "3").Times(over(t, "9", "4")).Text(), "3/2"},
		{"a product below zero", over(t, "-2", "3").Times(of(t, "3")).Text(), "-2"},
		{"a quotient", over(t, "10", "4").Text(), "5/2"},
		{"a quotient by a negative", over(t, "10", "-4").Text(), "-5/2"},
		{"a quotient of two negatives", over(t, "-7", "-2").Text(), "7/2"},
		{"zero by a negative", over(t, "0", "-3").Text(), "0"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %s", c.name, c.got)
		}
	}
	if _, ok := of(t, "1").Over(of(t, "0")); ok {
		t.Fatal("a divisor of zero gave a value")
	}
	if _, ok := Of(decimal.Decimal{}); ok {
		t.Fatal("an invalid decimal gave a fraction")
	}
	if (Fraction{}).Valid() || !Whole(0).Valid() {
		t.Fatal("validity")
	}
	units, ok := Units(big.NewInt(-1250), 3)
	if !ok || units.Text() != "-5/4" {
		t.Fatalf("units: %v %s", ok, units.Text())
	}
	if _, ok := Units(nil, 1); ok {
		t.Fatal("no units gave a fraction")
	}
	if _, ok := Units(big.NewInt(1), -1); ok {
		t.Fatal("a scale below zero gave a fraction")
	}
	if whole, ok := Units(big.NewInt(12), 0); !ok || whole.Text() != "12" {
		t.Fatal("units at scale zero")
	}
}

func TestComparisonSignAndDigits(t *testing.T) {
	half, third := over(t, "1", "2"), over(t, "1", "3")
	minus := over(t, "1", "-2")
	if !third.Less(half) || half.Less(third) || half.Less(half) || !minus.Less(third) || third.Less(minus) || !minus.Less(Whole(0)) {
		t.Fatal("order")
	}
	if half.Sign() != 1 || minus.Sign() != -1 || Whole(0).Sign() != 0 {
		t.Fatal("sign")
	}
	cases := map[string]int{"0": 1, "7": 1, "-7": 1, "-123": 3, "0.001": 4, "123456.789": 9, "99.5": 3}
	for text, want := range cases {
		if got := of(t, text).Digits(); got != want {
			t.Errorf("digits of %s: %d", text, got)
		}
	}
}

func TestOperandsAreNotChanged(t *testing.T) {
	a, b := over(t, "-7", "3"), over(t, "5", "-4")
	before := a.Text() + " " + b.Text()
	a.Plus(b)
	a.Minus(b)
	a.Times(b)
	a.Over(b)
	a.Less(b)
	a.Digits()
	a.Quanta(3, "half_even")
	b.Quanta(0, "away_from_zero")
	if after := a.Text() + " " + b.Text(); after != before || before != "-7/3 -5/4" {
		t.Fatalf("operands: %s then %s", before, after)
	}
}

func TestLiteralRounding(t *testing.T) {
	cases := []struct {
		top, bottom string
		want        map[string]int64
	}{
		{"10", "-4", map[string]int64{"toward_zero": -2, "away_from_zero": -3, "floor": -3, "ceiling": -2, "half_up": -3, "half_even": -2}},
		{"7", "-2", map[string]int64{"toward_zero": -3, "away_from_zero": -4, "floor": -4, "ceiling": -3, "half_up": -4, "half_even": -4}},
		{"7", "2", map[string]int64{"toward_zero": 3, "away_from_zero": 4, "floor": 3, "ceiling": 4, "half_up": 4, "half_even": 4}},
		{"5", "2", map[string]int64{"toward_zero": 2, "away_from_zero": 3, "floor": 2, "ceiling": 3, "half_up": 3, "half_even": 2}},
		{"7", "3", map[string]int64{"toward_zero": 2, "away_from_zero": 3, "floor": 2, "ceiling": 3, "half_up": 2, "half_even": 2}},
		{"8", "3", map[string]int64{"toward_zero": 2, "away_from_zero": 3, "floor": 2, "ceiling": 3, "half_up": 3, "half_even": 3}},
		{"-8", "3", map[string]int64{"toward_zero": -2, "away_from_zero": -3, "floor": -3, "ceiling": -2, "half_up": -3, "half_even": -3}},
		{"6", "3", map[string]int64{"toward_zero": 2, "away_from_zero": 2, "floor": 2, "ceiling": 2, "half_up": 2, "half_even": 2}},
	}
	for _, c := range cases {
		if len(c.want) != len(modes) {
			t.Fatalf("modes of %s/%s", c.top, c.bottom)
		}
		for mode, want := range c.want {
			got, ok := over(t, c.top, c.bottom).Quanta(0, mode)
			if !ok || got.Int64() != want {
				t.Errorf("%s/%s under %s: %v", c.top, c.bottom, mode, got)
			}
		}
	}
	scales := map[int]int64{0: 1235, 1: 12346, 2: 123457, 3: 1234568, 4: 12345678}
	for scale, want := range scales {
		got, ok := of(t, "1234.5678").Plus(over(t, "5", "100000")).Quanta(scale, "half_even")
		if !ok || got.Int64() != want {
			t.Errorf("scale %d: %v", scale, got)
		}
	}
	for _, mode := range []string{"", "half_down", "HALF_UP", "round"} {
		if _, ok := Whole(1).Quanta(0, mode); ok || ValidMode(mode) {
			t.Errorf("mode %q accepted", mode)
		}
	}
	if _, ok := Whole(1).Quanta(-1, "floor"); ok {
		t.Fatal("a scale below zero accepted")
	}
	if len(modes) != 6 || !ValidMode("half_even") {
		t.Fatal("modes")
	}
}

func TestArithmeticAgainstAReference(t *testing.T) {
	random := rand.New(rand.NewSource(2080))
	text := func() string {
		return fmt.Sprintf("%d.%04d", random.Intn(4000)-2000, random.Intn(10000))
	}
	rat := func(value string) *big.Rat {
		made, _ := new(big.Rat).SetString(value)
		return made
	}
	show := func(value *big.Rat) string {
		if value.IsInt() {
			return value.Num().String()
		}
		return value.Num().String() + "/" + value.Denom().String()
	}
	zero := 0
	for round := 0; round < 20000; round++ {
		a, b, c := text(), text(), text()
		if random.Intn(20) == 0 {
			b = "0"
		}
		left, right, third := of(t, a), of(t, b), of(t, c)
		want := new(big.Rat).Mul(new(big.Rat).Add(rat(a), rat(b)), rat(c))
		got := left.Plus(right).Times(third)
		if got.Text() != show(want) {
			t.Fatalf("(%s+%s)*%s: %s, the reference %s", a, b, c, got.Text(), show(want))
		}
		want = new(big.Rat).Sub(rat(a), rat(c))
		got = left.Minus(third)
		if rat(b).Sign() == 0 {
			zero++
			if _, ok := got.Over(right); ok {
				t.Fatalf("%s over zero", got.Text())
			}
			continue
		}
		want.Quo(want, rat(b))
		got, _ = got.Over(right)
		if got.Text() != show(want) || got.Sign() != want.Sign() || got.Less(third) != (want.Cmp(rat(c)) < 0) {
			t.Fatalf("(%s-%s)/%s: %s, the reference %s", a, c, b, got.Text(), show(want))
		}
		digits := len(new(big.Int).Abs(want.Num()).String())
		if other := len(want.Denom().String()); other > digits {
			digits = other
		}
		if got.Digits() != digits {
			t.Fatalf("digits of %s: %d", got.Text(), got.Digits())
		}
	}
	if zero < 500 {
		t.Fatalf("only %d divisors of zero", zero)
	}
}

func TestRoundingAgreesWithThePrimitive(t *testing.T) {
	p := profile(t)
	random := rand.New(rand.NewSource(2081))
	ties, negative := 0, 0
	for round := 0; round < 20000; round++ {
		numerator := fmt.Sprintf("%d.%d", random.Intn(2000)-1000, random.Intn(4)*25)
		denominator := fmt.Sprintf("%d.%d", 1+random.Intn(40), random.Intn(2)*5)
		if random.Intn(4) == 0 {
			denominator = []string{"2", "4", "8", "0.5"}[random.Intn(4)]
		}
		if random.Intn(3) == 0 {
			denominator = "-" + denominator
			negative++
		}
		top, _ := decimal.Parse(p, numerator)
		bottom, _ := decimal.Parse(p, denominator)
		value := over(t, numerator, denominator)
		for scale := 0; scale <= 4; scale++ {
			for name := range modes {
				mode, err := decimal.ParseMode(name)
				if err != nil {
					t.Fatal(err)
				}
				want, err := decimal.DivideRound(p, top, bottom, scale, mode)
				if err != nil {
					t.Fatal(err)
				}
				units, ok := value.Quanta(scale, name)
				if !ok {
					t.Fatal("no rounding")
				}
				got, err := decimal.New(p, units, scale)
				if err != nil || !got.Equal(want) {
					t.Fatalf("%s over %s at scale %d under %s: %s, the primitive %s (%v)", numerator, denominator, scale, name, got, want, err)
				}
			}
			scaled := new(big.Int).Mul(value.num, power(scale))
			if new(big.Int).Rem(new(big.Int).Mul(scaled, two), value.den).Sign() == 0 && new(big.Int).Rem(scaled, value.den).Sign() != 0 {
				ties++
			}
		}
	}
	if ties < 2000 || negative < 5000 {
		t.Fatalf("only %d ties and %d negative divisors", ties, negative)
	}
}

func TestTheSourceUsesNoFloatingPointAndHasNoComments(t *testing.T) {
	body, err := os.ReadFile("exact.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, word := range []string{"float", "Float", "big.Rat", "//", "/*"} {
		if strings.Contains(string(body), word) {
			t.Errorf("exact.go contains %q", word)
		}
	}
}
