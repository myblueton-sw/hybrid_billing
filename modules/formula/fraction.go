package formula

import (
	"math/big"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	bigOne = big.NewInt(1)
	bigTwo = big.NewInt(2)
	bigTen = big.NewInt(10)
)

var modes = map[string]bool{"toward_zero": true, "away_from_zero": true, "floor": true, "ceiling": true, "half_up": true, "half_even": true}

var quanta = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}

type fraction struct {
	num *big.Int
	den *big.Int
}

func power(scale int) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(scale)), nil)
}

func lowest(num, den *big.Int) fraction {
	if den.Sign() < 0 {
		num, den = new(big.Int).Neg(num), new(big.Int).Neg(den)
	}
	divisor := new(big.Int).GCD(nil, nil, new(big.Int).Abs(num), den)
	return fraction{num: new(big.Int).Quo(num, divisor), den: new(big.Int).Quo(den, divisor)}
}

func whole(value int64) fraction {
	return fraction{num: big.NewInt(value), den: big.NewInt(1)}
}

func of(value decimal.Decimal) fraction {
	return lowest(value.Coefficient(), power(value.Scale()))
}

func (f fraction) plus(other fraction) fraction {
	num := new(big.Int).Mul(f.num, other.den)
	num.Add(num, new(big.Int).Mul(other.num, f.den))
	return lowest(num, new(big.Int).Mul(f.den, other.den))
}

func (f fraction) minus(other fraction) fraction {
	return f.plus(fraction{num: new(big.Int).Neg(other.num), den: other.den})
}

func (f fraction) times(other fraction) fraction {
	return lowest(new(big.Int).Mul(f.num, other.num), new(big.Int).Mul(f.den, other.den))
}

func (f fraction) over(other fraction) fraction {
	return lowest(new(big.Int).Mul(f.num, other.den), new(big.Int).Mul(f.den, other.num))
}

func (f fraction) sign() int { return f.num.Sign() }

func (f fraction) less(other fraction) bool {
	return new(big.Int).Mul(f.num, other.den).Cmp(new(big.Int).Mul(other.num, f.den)) < 0
}

func (f fraction) digits() int {
	num, den := len(new(big.Int).Abs(f.num).String()), len(f.den.String())
	if den > num {
		return den
	}
	return num
}

func (f fraction) text() string {
	if f.den.Cmp(bigOne) == 0 {
		return f.num.String()
	}
	return f.num.String() + "/" + f.den.String()
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
