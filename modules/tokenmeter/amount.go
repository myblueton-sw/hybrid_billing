package tokenmeter

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	ten        = big.NewInt(10)
	amountForm = regexp.MustCompile(`^(-?)(0|[1-9][0-9]*)(?:\.([0-9]+))?(?:[eE]([+-]?[0-9]{1,2}))?$`)
	wholeForm  = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

const (
	maxAmountBytes = 40
	maxCountDigits = 18
	maxTotalDigits = 30
	floatDigits    = 15
)

type number struct {
	coefficient *big.Int
	scale       int
}

func zero() *number { return &number{coefficient: new(big.Int)} }

func (n *number) add(other *number) {
	added := new(big.Int).Set(other.coefficient)
	for n.scale < other.scale {
		n.coefficient.Mul(n.coefficient, ten)
		n.scale++
	}
	for scale := other.scale; scale < n.scale; scale++ {
		added.Mul(added, ten)
	}
	n.coefficient.Add(n.coefficient, added)
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

func (n *number) equals(other *number) bool {
	value, scale := n.reduced()
	against, places := other.reduced()
	return scale == places && value.Cmp(against) == 0
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

func parseAmount(text string) (*number, int, bool) {
	if len(text) > maxAmountBytes {
		return nil, 0, false
	}
	parts := amountForm.FindStringSubmatch(text)
	if parts == nil {
		return nil, 0, false
	}
	digits := parts[2] + parts[3]
	coefficient, _ := new(big.Int).SetString(digits, 10)
	scale := len(parts[3])
	if parts[4] != "" {
		exponent, _ := strconv.Atoi(parts[4])
		scale -= exponent
	}
	for scale < 0 {
		coefficient.Mul(coefficient, ten)
		scale++
	}
	if parts[1] == "-" {
		coefficient.Neg(coefficient)
	}
	significant := strings.TrimRight(strings.TrimLeft(digits, "0"), "0")
	count := len(significant)
	if count == 0 {
		count = 1
	}
	return &number{coefficient: coefficient, scale: scale}, count, true
}

func parseWhole(text string, most int) (*big.Int, bool) {
	if len(text) > most || !wholeForm.MatchString(text) {
		return nil, false
	}
	value, _ := new(big.Int).SetString(text, 10)
	return value, true
}
