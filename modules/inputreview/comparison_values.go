package inputreview

import (
	"math/big"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type totalKey struct{ field, basis, unit string }
type comparisonTotal struct {
	expected, observed       *big.Rat
	hasExpected, hasObserved bool
}

func currencyCode(text string) bool {
	return len(text) == 3 && text[0] >= 'A' && text[0] <= 'Z' && text[1] >= 'A' && text[1] <= 'Z' && text[2] >= 'A' && text[2] <= 'Z'
}

func exactTotal(profile decimal.Profile, value *big.Rat) (string, error) {
	denominator := new(big.Int).Set(value.Denom())
	scale := 0
	for denominator.Cmp(big.NewInt(1)) != 0 && scale <= 80 {
		gcd := new(big.Int).GCD(nil, nil, denominator, big.NewInt(10))
		if gcd.Cmp(big.NewInt(1)) == 0 {
			return "", ErrRejected
		}
		denominator.Quo(denominator, gcd)
		scale++
	}
	if scale > 80 {
		return "", ErrLimit
	}
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	coefficient := new(big.Int).Mul(value.Num(), power)
	coefficient.Quo(coefficient, value.Denom())
	out, err := decimal.New(profile, coefficient, scale)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func resultNumber(profile decimal.Profile, text string) (decimal.Decimal, error) {
	if len(text) < 1 || len(text) > 163 {
		return decimal.Decimal{}, ErrInvalid
	}
	body := strings.TrimPrefix(text, "-")
	whole, fraction, point := strings.Cut(body, ".")
	if whole == "" || point && fraction == "" || len(whole) > 1 && whole[0] == '0' {
		return decimal.Decimal{}, ErrInvalid
	}
	for _, c := range whole + fraction {
		if c < '0' || c > '9' {
			return decimal.Decimal{}, ErrInvalid
		}
	}
	coefficient, ok := new(big.Int).SetString(whole+fraction, 10)
	if !ok {
		return decimal.Decimal{}, ErrInvalid
	}
	if strings.HasPrefix(text, "-") {
		coefficient.Neg(coefficient)
	}
	return decimal.New(profile, coefficient, len(fraction))
}

func accumulate(profile decimal.Profile, totals map[totalKey]*comparisonTotal, key totalKey, text string, observed bool) error {
	value, err := resultNumber(profile, text)
	if err != nil {
		return err
	}
	rat, ok := new(big.Rat).SetString(value.String())
	if !ok {
		return ErrInvalid
	}
	if totals[key] == nil {
		totals[key] = &comparisonTotal{expected: new(big.Rat), observed: new(big.Rat)}
	}
	if observed {
		totals[key].hasObserved = true
		totals[key].observed.Add(totals[key].observed, rat)
	} else {
		totals[key].hasExpected = true
		totals[key].expected.Add(totals[key].expected, rat)
	}
	return nil
}

var sourceTime = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

func expectedCell(profile decimal.Profile, field normalize.Field, entry normalize.Entry, cells map[string]normalize.Cell) (normalize.OutCell, error) {
	if entry.Missing != "" {
		return normalize.OutCell{State: normalize.Unmapped}, nil
	}
	cell, exists := cells[entry.Source]
	if !exists {
		return normalize.OutCell{State: normalize.NotProvided}, nil
	}
	if cell.Null {
		return normalize.OutCell{State: normalize.Null}, nil
	}
	text := cell.Text
	switch field.Kind {
	case normalize.Cost, normalize.Quantity:
		value, err := decimal.Parse(profile, text)
		if err != nil {
			return normalize.OutCell{}, err
		}
		rat, ok := new(big.Rat).SetString(value.String())
		if !ok {
			return normalize.OutCell{}, ErrInvalid
		}
		if entry.Transform == normalize.Negate {
			rat.Neg(rat)
		}
		if entry.Transform == normalize.Scale {
			power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(entry.Power))), nil)
			factor := new(big.Rat).SetInt(power)
			if entry.Power < 0 {
				rat.Quo(rat, factor)
			} else {
				rat.Mul(rat, factor)
			}
		}
		text, err = exactTotal(profile, rat)
		if err != nil {
			return normalize.OutCell{}, err
		}
	case normalize.Currency:
		if !currencyCode(text) {
			return normalize.OutCell{}, ErrInvalid
		}
	case normalize.Time:
		if !sourceTime.MatchString(text) {
			return normalize.OutCell{}, ErrInvalid
		}
		at, err := time.Parse(time.RFC3339Nano, text)
		if err != nil || at.UTC().Year() < 0 || at.UTC().Year() > 9999 {
			return normalize.OutCell{}, ErrInvalid
		}
		text = at.UTC().Format(time.RFC3339Nano)
	case normalize.Text:
	default:
		return normalize.OutCell{}, ErrRejected
	}
	return normalize.OutCell{State: normalize.Value, Text: text}, nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func boundedText(value reflect.Value, remaining *int) bool {
	if value.Type() == reflect.TypeOf(time.Time{}) {
		return true
	}
	switch value.Kind() {
	case reflect.String:
		text := value.String()
		*remaining -= len(text)
		return *remaining >= 0 && len(text) <= 4096 && utf8.ValidString(text)
	case reflect.Pointer:
		return value.IsNil() || boundedText(value.Elem(), remaining)
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if !boundedText(value.Field(i), remaining) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if !boundedText(value.Index(i), remaining) {
				return false
			}
		}
	case reflect.Map:
		iterator := value.MapRange()
		for iterator.Next() {
			if !boundedText(iterator.Key(), remaining) || !boundedText(iterator.Value(), remaining) {
				return false
			}
		}
	}
	return true
}
