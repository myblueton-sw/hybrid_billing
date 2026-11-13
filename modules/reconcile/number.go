package reconcile

import (
	"math/big"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var ten = big.NewInt(10)

type number struct {
	coefficient *big.Int
	scale       int
}

func zero() *number { return &number{coefficient: new(big.Int)} }

func parse(profile decimal.Profile, text string) (*number, bool) {
	parsed, err := decimal.Parse(profile, text)
	if err != nil {
		return nil, false
	}
	return &number{coefficient: parsed.Coefficient(), scale: parsed.Scale()}, true
}

func (n *number) add(other *number, sign int64) {
	added := new(big.Int).Mul(other.coefficient, big.NewInt(sign))
	for n.scale < other.scale {
		n.coefficient.Mul(n.coefficient, ten)
		n.scale++
	}
	for scale := other.scale; scale < n.scale; scale++ {
		added.Mul(added, ten)
	}
	n.coefficient.Add(n.coefficient, added)
}

func (n *number) copy() *number {
	return &number{coefficient: new(big.Int).Set(n.coefficient), scale: n.scale}
}

func (n *number) sign() int { return n.coefficient.Sign() }

func (n *number) exceeds(bound *number) bool {
	size := n.copy()
	size.coefficient.Abs(size.coefficient)
	size.add(bound, -1)
	return size.sign() > 0
}

func (n *number) reduced() (*big.Int, int) {
	value := new(big.Int).Set(n.coefficient)
	scale := n.scale
	quotient, remainder := new(big.Int), new(big.Int)
	for scale > 0 {
		quotient.QuoRem(value, ten, remainder)
		if remainder.Sign() != 0 {
			break
		}
		value.Set(quotient)
		scale--
	}
	return value, scale
}

func (n *number) text() string {
	value, scale := n.reduced()
	digits := new(big.Int).Abs(value).Text(10)
	if scale > 0 {
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		digits = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if value.Sign() < 0 {
		return "-" + digits
	}
	return digits
}

func (n *number) fits(profile decimal.Profile) bool {
	value, scale := n.reduced()
	_, err := decimal.New(profile, value, scale)
	return err == nil
}
