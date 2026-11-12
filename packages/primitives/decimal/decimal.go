package decimal

import (
	"math/big"
	"strings"
)

type Decimal struct {
	coefficient *big.Int
	scale       int
}

var ten = big.NewInt(10)

func pow10(exponent int) *big.Int {
	return new(big.Int).Exp(ten, big.NewInt(int64(exponent)), nil)
}

func digits(value *big.Int) int {
	if value.Sign() == 0 {
		return 1
	}
	return len(new(big.Int).Abs(value).Text(10))
}

func canonical(coefficient *big.Int, scale int) Decimal {
	value := new(big.Int).Set(coefficient)
	if value.Sign() == 0 {
		return Decimal{coefficient: value, scale: 0}
	}
	remainder := new(big.Int)
	quotient := new(big.Int)
	for scale > 0 {
		quotient.QuoRem(value, ten, remainder)
		if remainder.Sign() != 0 {
			break
		}
		value.Set(quotient)
		scale--
	}
	return Decimal{coefficient: value, scale: scale}
}

func (d Decimal) Valid() bool { return d.coefficient != nil }

func (d Decimal) Sign() int {
	if !d.Valid() {
		return 0
	}
	return d.coefficient.Sign()
}

func (d Decimal) Scale() int { return d.scale }

func (d Decimal) Coefficient() *big.Int {
	if !d.Valid() {
		return nil
	}
	return new(big.Int).Set(d.coefficient)
}

func (d Decimal) Equal(other Decimal) bool {
	return d.Valid() && other.Valid() && d.scale == other.scale && d.coefficient.Cmp(other.coefficient) == 0
}

func (d Decimal) String() string {
	if !d.Valid() {
		return "invalid"
	}
	text := new(big.Int).Abs(d.coefficient).Text(10)
	if d.scale > 0 {
		if len(text) <= d.scale {
			text = strings.Repeat("0", d.scale-len(text)+1) + text
		}
		text = text[:len(text)-d.scale] + "." + text[len(text)-d.scale:]
	}
	if d.coefficient.Sign() < 0 {
		return "-" + text
	}
	return text
}

func integerDigits(d Decimal) int {
	if d.coefficient.Sign() == 0 {
		return 0
	}
	count := digits(d.coefficient) - d.scale
	if count < 0 {
		return 0
	}
	return count
}

func within(p Profile, d Decimal) bool {
	return integerDigits(d) <= p.spec.ResultIntegerDigits && d.scale <= p.spec.ResultScale
}

func operands(p Profile, values ...Decimal) error {
	if !p.valid {
		return ErrInvalidProfile
	}
	for _, value := range values {
		if !value.Valid() {
			return ErrInvalidOperand
		}
		if !within(p, value) {
			return ErrResultBound
		}
	}
	return nil
}

func result(p Profile, coefficient *big.Int, scale int) (Decimal, error) {
	value := canonical(coefficient, scale)
	if !within(p, value) {
		return Decimal{}, ErrResultBound
	}
	return value, nil
}

func New(p Profile, coefficient *big.Int, scale int) (Decimal, error) {
	if !p.valid {
		return Decimal{}, ErrInvalidProfile
	}
	if coefficient == nil {
		return Decimal{}, ErrInvalidOperand
	}
	if scale < 0 || scale > p.spec.ResultScale {
		return Decimal{}, ErrInvalidScale
	}
	if digits(coefficient) > p.spec.IntermediateDigits {
		return Decimal{}, ErrIntermediateBound
	}
	return result(p, coefficient, scale)
}

func Parse(p Profile, text string) (Decimal, error) {
	if !p.valid {
		return Decimal{}, ErrInvalidProfile
	}
	if len(text) > p.spec.InputIntegerDigits+p.spec.InputScale+2 {
		return Decimal{}, ErrInputBound
	}
	body := strings.TrimPrefix(text, "-")
	whole, fraction, pointed := strings.Cut(body, ".")
	if whole == "" || (pointed && fraction == "") || (len(whole) > 1 && whole[0] == '0') {
		return Decimal{}, ErrSyntax
	}
	for _, part := range []string{whole, fraction} {
		for index := 0; index < len(part); index++ {
			if part[index] < '0' || part[index] > '9' {
				return Decimal{}, ErrSyntax
			}
		}
	}
	submitted := len(whole)
	if whole == "0" {
		submitted = 0
	}
	if submitted > p.spec.InputIntegerDigits || len(fraction) > p.spec.InputScale {
		return Decimal{}, ErrInputBound
	}
	coefficient, _ := new(big.Int).SetString(whole+fraction, 10)
	if strings.HasPrefix(text, "-") {
		coefficient.Neg(coefficient)
	}
	return canonical(coefficient, len(fraction)), nil
}

func Cmp(p Profile, a, b Decimal) (int, error) {
	if err := operands(p, a, b); err != nil {
		return 0, err
	}
	left, right := aligned(a, b)
	return left.Cmp(right), nil
}

func aligned(a, b Decimal) (*big.Int, *big.Int) {
	left, right := new(big.Int).Set(a.coefficient), new(big.Int).Set(b.coefficient)
	if a.scale < b.scale {
		left.Mul(left, pow10(b.scale-a.scale))
	} else if b.scale < a.scale {
		right.Mul(right, pow10(a.scale-b.scale))
	}
	return left, right
}

func Neg(p Profile, a Decimal) (Decimal, error) {
	if err := operands(p, a); err != nil {
		return Decimal{}, err
	}
	return result(p, new(big.Int).Neg(a.coefficient), a.scale)
}

func Add(p Profile, a, b Decimal) (Decimal, error) {
	if err := operands(p, a, b); err != nil {
		return Decimal{}, err
	}
	left, right := aligned(a, b)
	return result(p, left.Add(left, right), max(a.scale, b.scale))
}

func Sub(p Profile, a, b Decimal) (Decimal, error) {
	if err := operands(p, a, b); err != nil {
		return Decimal{}, err
	}
	left, right := aligned(a, b)
	return result(p, left.Sub(left, right), max(a.scale, b.scale))
}

func Mul(p Profile, a, b Decimal) (Decimal, error) {
	if err := operands(p, a, b); err != nil {
		return Decimal{}, err
	}
	if digits(a.coefficient)+digits(b.coefficient) > p.spec.IntermediateDigits {
		return Decimal{}, ErrIntermediateBound
	}
	product := canonical(new(big.Int).Mul(a.coefficient, b.coefficient), a.scale+b.scale)
	if !within(p, product) {
		return Decimal{}, ErrResultBound
	}
	return product, nil
}

func ratio(p Profile, numerator, denominator Decimal, shift int) (*big.Int, *big.Int, error) {
	if denominator.coefficient.Sign() == 0 {
		return nil, nil, ErrDivisionByZero
	}
	if digits(numerator.coefficient)+denominator.scale+shift > p.spec.IntermediateDigits ||
		digits(denominator.coefficient)+numerator.scale > p.spec.IntermediateDigits {
		return nil, nil, ErrIntermediateBound
	}
	top := new(big.Int).Mul(numerator.coefficient, pow10(denominator.scale+shift))
	bottom := new(big.Int).Mul(denominator.coefficient, pow10(numerator.scale))
	if bottom.Sign() < 0 {
		top.Neg(top)
		bottom.Neg(bottom)
	}
	return top, bottom, nil
}

func Quo(p Profile, numerator, denominator Decimal) (Decimal, error) {
	if err := operands(p, numerator, denominator); err != nil {
		return Decimal{}, err
	}
	top, bottom, err := ratio(p, numerator, denominator, 0)
	if err != nil {
		return Decimal{}, err
	}
	common := new(big.Int).GCD(nil, nil, new(big.Int).Abs(top), bottom)
	if common.Sign() != 0 {
		top.Quo(top, common)
		bottom.Quo(bottom, common)
	}
	twos, fives := 0, 0
	rest := new(big.Int).Set(bottom)
	remainder := new(big.Int)
	quotient := new(big.Int)
	for _, factor := range []struct {
		value *big.Int
		count *int
	}{{big.NewInt(2), &twos}, {big.NewInt(5), &fives}} {
		for *factor.count <= p.spec.ResultScale {
			quotient.QuoRem(rest, factor.value, remainder)
			if remainder.Sign() != 0 {
				break
			}
			rest.Set(quotient)
			*factor.count++
		}
	}
	scale := max(twos, fives)
	if rest.Cmp(big.NewInt(1)) != 0 || scale > p.spec.ResultScale {
		return Decimal{}, ErrInexactDivision
	}
	if digits(top)+scale > p.spec.IntermediateDigits {
		return Decimal{}, ErrIntermediateBound
	}
	top.Mul(top, pow10(scale))
	return result(p, top.Quo(top, bottom), scale)
}

func DivideRound(p Profile, numerator, denominator Decimal, scale int, mode Mode) (Decimal, error) {
	if err := operands(p, numerator, denominator); err != nil {
		return Decimal{}, err
	}
	if !mode.Valid() {
		return Decimal{}, ErrInvalidMode
	}
	if scale < 0 || scale > p.spec.ResultScale {
		return Decimal{}, ErrInvalidScale
	}
	top, bottom, err := ratio(p, numerator, denominator, scale)
	if err != nil {
		return Decimal{}, err
	}
	return result(p, roundRatio(top, bottom, mode), scale)
}

func Round(p Profile, value Decimal, scale int, mode Mode) (Decimal, error) {
	if err := operands(p, value); err != nil {
		return Decimal{}, err
	}
	if !mode.Valid() {
		return Decimal{}, ErrInvalidMode
	}
	if scale < 0 || scale > p.spec.ResultScale {
		return Decimal{}, ErrInvalidScale
	}
	if scale >= value.scale {
		return value, nil
	}
	return result(p, roundRatio(value.coefficient, pow10(value.scale-scale), mode), scale)
}

func Quanta(p Profile, value, quantum Decimal, mode Mode) (*big.Int, error) {
	if err := operands(p, value, quantum); err != nil {
		return nil, err
	}
	if !mode.Valid() {
		return nil, ErrInvalidMode
	}
	if quantum.coefficient.Sign() <= 0 {
		return nil, ErrInvalidQuantum
	}
	top, bottom, err := ratio(p, value, quantum, 0)
	if err != nil {
		return nil, err
	}
	return roundRatio(top, bottom, mode), nil
}

func RoundToQuantum(p Profile, value, quantum Decimal, mode Mode) (Decimal, error) {
	count, err := Quanta(p, value, quantum, mode)
	if err != nil {
		return Decimal{}, err
	}
	if digits(count)+digits(quantum.coefficient) > p.spec.IntermediateDigits {
		return Decimal{}, ErrIntermediateBound
	}
	return result(p, count.Mul(count, quantum.coefficient), quantum.scale)
}
