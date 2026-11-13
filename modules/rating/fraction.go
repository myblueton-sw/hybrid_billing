package rating

import (
	"math/big"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
	bigTwo  = big.NewInt(2)
	bigTen  = big.NewInt(10)
)

var modes = map[string]bool{"toward_zero": true, "away_from_zero": true, "floor": true, "ceiling": true, "half_up": true, "half_even": true}

type fraction struct {
	num *big.Int
	den *big.Int
}

func power(scale int) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(scale)), nil)
}

func whole(value int64) fraction {
	return fraction{num: big.NewInt(value), den: big.NewInt(1)}
}

func of(value decimal.Decimal) fraction {
	return fraction{num: value.Coefficient(), den: power(value.Scale())}
}

func (f fraction) plus(other fraction) fraction {
	num := new(big.Int).Mul(f.num, other.den)
	num.Add(num, new(big.Int).Mul(other.num, f.den))
	return fraction{num: num, den: new(big.Int).Mul(f.den, other.den)}
}

func (f fraction) minus(other fraction) fraction {
	return f.plus(fraction{num: new(big.Int).Neg(other.num), den: other.den})
}

func (f fraction) times(other fraction) fraction {
	return fraction{num: new(big.Int).Mul(f.num, other.num), den: new(big.Int).Mul(f.den, other.den)}
}

func (f fraction) over(other fraction) fraction {
	return fraction{num: new(big.Int).Mul(f.num, other.den), den: new(big.Int).Mul(f.den, other.num)}
}

func (f fraction) sign() int { return f.num.Sign() }

func (f fraction) less(other fraction) bool {
	return new(big.Int).Mul(f.num, other.den).Cmp(new(big.Int).Mul(other.num, f.den)) < 0
}

func (f fraction) text() string {
	divisor := new(big.Int).GCD(nil, nil, new(big.Int).Abs(f.num), f.den)
	num, den := new(big.Int).Quo(f.num, divisor), new(big.Int).Quo(f.den, divisor)
	if den.Cmp(bigOne) == 0 {
		return num.String()
	}
	return num.String() + "/" + den.String()
}

func round(num, den *big.Int, mode string) *big.Int {
	quotient, remainder := new(big.Int).QuoRem(num, den, new(big.Int))
	away := big.NewInt(int64(remainder.Sign()))
	twice := new(big.Int).Mul(new(big.Int).Abs(remainder), bigTwo)
	half := twice.Cmp(den)
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
	return quotient
}

func (f fraction) quanta(scale int, mode string) *big.Int {
	return round(new(big.Int).Mul(f.num, power(scale)), f.den, mode)
}
