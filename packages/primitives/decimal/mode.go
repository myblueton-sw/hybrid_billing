package decimal

import "math/big"

type Mode int

const (
	modeUnset Mode = iota
	TowardZero
	AwayFromZero
	Floor
	Ceiling
	HalfUp
	HalfEven
)

var modeNames = map[Mode]string{
	TowardZero:   "toward_zero",
	AwayFromZero: "away_from_zero",
	Floor:        "floor",
	Ceiling:      "ceiling",
	HalfUp:       "half_up",
	HalfEven:     "half_even",
}

func ParseMode(name string) (Mode, error) {
	for mode, known := range modeNames {
		if known == name {
			return mode, nil
		}
	}
	return modeUnset, ErrInvalidMode
}

func (m Mode) Valid() bool {
	_, ok := modeNames[m]
	return ok
}

func (m Mode) String() string { return modeNames[m] }

func roundRatio(numerator, denominator *big.Int, mode Mode) *big.Int {
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	sign := big.NewInt(int64(numerator.Sign()))
	twice := new(big.Int).Abs(remainder)
	twice.Lsh(twice, 1)
	half := twice.Cmp(denominator)
	adjust := false
	switch mode {
	case AwayFromZero:
		adjust = remainder.Sign() != 0
	case Floor:
		adjust = remainder.Sign() < 0
	case Ceiling:
		adjust = remainder.Sign() > 0
	case HalfUp:
		adjust = half >= 0
	case HalfEven:
		adjust = half > 0 || (half == 0 && quotient.Bit(0) == 1)
	}
	if adjust {
		quotient.Add(quotient, sign)
	}
	return quotient
}
