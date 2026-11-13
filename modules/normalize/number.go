package normalize

import (
	"errors"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var ten = big.NewInt(10)

type number struct {
	coefficient *big.Int
	scale       int
}

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

func parseNumber(profile decimal.Profile, text string) (*number, string) {
	parsed, err := decimal.Parse(profile, text)
	switch {
	case errors.Is(err, decimal.ErrSyntax):
		return nil, NotDecimal
	case err != nil:
		return nil, OutsideProfile
	}
	return &number{coefficient: parsed.Coefficient(), scale: parsed.Scale()}, ""
}

func (n *number) apply(entry Entry) {
	switch entry.Transform {
	case Negate:
		n.coefficient.Neg(n.coefficient)
	case Scale:
		n.scale -= entry.Power
		for n.scale < 0 {
			n.coefficient.Mul(n.coefficient, ten)
			n.scale++
		}
	}
}

var timeShape = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

func parseTime(text string) (time.Time, bool) {
	if !timeShape.MatchString(text) {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, false
	}
	parsed = parsed.UTC()
	if parsed.Year() < 0 || parsed.Year() > 9999 {
		return time.Time{}, false
	}
	return parsed, true
}
