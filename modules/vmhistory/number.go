package vmhistory

import (
	"math/big"
	"strings"
)

var (
	sizeUnits = map[string]*big.Int{
		"B":   big.NewInt(1),
		"kB":  big.NewInt(1000),
		"MB":  big.NewInt(1000 * 1000),
		"GB":  big.NewInt(1000 * 1000 * 1000),
		"TB":  big.NewInt(1000 * 1000 * 1000 * 1000),
		"KiB": big.NewInt(1024),
		"MiB": big.NewInt(1024 * 1024),
		"GiB": big.NewInt(1024 * 1024 * 1024),
		"TiB": big.NewInt(1024 * 1024 * 1024 * 1024),
	}
	energyUnits = map[string]*big.Int{
		"Wh":  big.NewInt(1),
		"kWh": big.NewInt(1000),
		"MWh": big.NewInt(1000 * 1000),
	}
	million = big.NewInt(1000 * 1000)
)

func whole(text string, most int) (*big.Int, bool) {
	if text == "" || len(text) > most || (len(text) > 1 && text[0] == '0') {
		return nil, false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return nil, false
		}
	}
	value, _ := new(big.Int).SetString(text, 10)
	return value, true
}

func bytesOf(size Size) (*big.Int, bool) {
	if size.Value == "" && size.Unit == "" {
		return nil, true
	}
	value, formed := whole(size.Value, maxSizeDigits)
	factor := sizeUnits[size.Unit]
	if !formed || factor == nil {
		return nil, false
	}
	return value.Mul(value, factor), true
}

func microOf(text string, factor *big.Int) (*big.Int, bool) {
	before, after, pointed := strings.Cut(text, ".")
	value, formed := whole(before, maxWholePart)
	if !formed || (pointed && (after == "" || len(after) > maxFraction)) {
		return nil, false
	}
	fraction := new(big.Int)
	for index := 0; index < len(after); index++ {
		if after[index] < '0' || after[index] > '9' {
			return nil, false
		}
	}
	if after != "" {
		fraction.SetString(after+strings.Repeat("0", maxFraction-len(after)), 10)
	}
	value.Mul(value, million).Add(value, fraction)
	return value.Mul(value, factor), true
}

func wattHours(micro *big.Int) string {
	quotient, remainder := new(big.Int).QuoRem(micro, million, new(big.Int))
	if remainder.Sign() == 0 {
		return quotient.String()
	}
	digits := remainder.String()
	digits = strings.Repeat("0", maxFraction-len(digits)) + digits
	return quotient.String() + "." + strings.TrimRight(digits, "0")
}
