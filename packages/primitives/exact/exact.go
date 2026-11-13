package exact

import (
	"math/big"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	one = big.NewInt(1)
	two = big.NewInt(2)
	ten = big.NewInt(10)
)

var modes = map[string]bool{"toward_zero": true, "away_from_zero": true, "floor": true, "ceiling": true, "half_up": true, "half_even": true}

type Fraction struct {
	num *big.Int
	den *big.Int
}

func ValidMode(name string) bool { return modes[name] }

func power(scale int) *big.Int {
	return new(big.Int).Exp(ten, big.NewInt(int64(scale)), nil)
}

func lowest(num, den *big.Int) Fraction {
	if den.Sign() < 0 {
		num, den = new(big.Int).Neg(num), new(big.Int).Neg(den)
	}
	divisor := new(big.Int).GCD(nil, nil, new(big.Int).Abs(num), den)
	return Fraction{num: new(big.Int).Quo(num, divisor), den: new(big.Int).Quo(den, divisor)}
}

func Whole(value int64) Fraction {
	return Fraction{num: big.NewInt(value), den: big.NewInt(1)}
}

func Of(value decimal.Decimal) (Fraction, bool) {
	if !value.Valid() {
		return Fraction{}, false
	}
	return lowest(value.Coefficient(), power(value.Scale())), true
}

func Units(units *big.Int, scale int) (Fraction, bool) {
	if units == nil || scale < 0 {
		return Fraction{}, false
	}
	return lowest(units, power(scale)), true
}

func (f Fraction) Valid() bool { return f.num != nil && f.den != nil }

func (f Fraction) Plus(other Fraction) Fraction {
	num := new(big.Int).Mul(f.num, other.den)
	num.Add(num, new(big.Int).Mul(other.num, f.den))
	return lowest(num, new(big.Int).Mul(f.den, other.den))
}

func (f Fraction) Minus(other Fraction) Fraction {
	return f.Plus(Fraction{num: new(big.Int).Neg(other.num), den: other.den})
}

func (f Fraction) Times(other Fraction) Fraction {
	return lowest(new(big.Int).Mul(f.num, other.num), new(big.Int).Mul(f.den, other.den))
}

func (f Fraction) Over(other Fraction) (Fraction, bool) {
	if other.num.Sign() == 0 {
		return Fraction{}, false
	}
	return lowest(new(big.Int).Mul(f.num, other.den), new(big.Int).Mul(f.den, other.num)), true
}

func (f Fraction) Sign() int { return f.num.Sign() }

func (f Fraction) Less(other Fraction) bool {
	return new(big.Int).Mul(f.num, other.den).Cmp(new(big.Int).Mul(other.num, f.den)) < 0
}

func (f Fraction) Digits() int {
	num, den := len(new(big.Int).Abs(f.num).String()), len(f.den.String())
	if den > num {
		return den
	}
	return num
}

func (f Fraction) Text() string {
	if f.den.Cmp(one) == 0 {
		return f.num.String()
	}
	return f.num.String() + "/" + f.den.String()
}

func (f Fraction) Quanta(scale int, mode string) (*big.Int, bool) {
	if scale < 0 || !modes[mode] {
		return nil, false
	}
	quotient, remainder := new(big.Int).QuoRem(new(big.Int).Mul(f.num, power(scale)), f.den, new(big.Int))
	away := big.NewInt(int64(remainder.Sign()))
	half := new(big.Int).Mul(new(big.Int).Abs(remainder), two).Cmp(f.den)
	up := false
	switch mode {
	case "away_from_zero":
		up = true
	case "floor":
		up = remainder.Sign() < 0
	case "ceiling":
		up = remainder.Sign() > 0
	case "half_up":
		up = half >= 0
	case "half_even":
		up = half > 0 || (half == 0 && quotient.Bit(0) == 1)
	}
	if up {
		quotient.Add(quotient, away)
	}
	return quotient, true
}
