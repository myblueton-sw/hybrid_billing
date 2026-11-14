package exports

import (
	"math/big"
	"strings"
	"unicode/utf8"
)

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func validText(text string) bool {
	return text != "" && len(text) <= maxTextBytes && utf8.ValidString(text)
}

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

var monthNames = map[string]bool{"01": true, "02": true, "03": true, "04": true, "05": true, "06": true, "07": true, "08": true, "09": true, "10": true, "11": true, "12": true}

func validMonth(text string) bool {
	return len(text) == 7 && text[:2] == "20" && text[2] >= '0' && text[2] <= '9' && text[3] >= '0' && text[3] <= '9' && text[4] == '-' && monthNames[text[5:]]
}

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func validZone(zone string) bool {
	if zone == "UTC" {
		return true
	}
	if len(zone) != 6 || zone[0] != '+' && zone[0] != '-' || zone[3] != ':' {
		return false
	}
	for _, index := range []int{1, 2, 4, 5} {
		if zone[index] < '0' || zone[index] > '9' {
			return false
		}
	}
	return zone[1:3] <= "14" && zone[4:] <= "59"
}

func units(text string, scale int) (*big.Int, bool) {
	body := strings.TrimPrefix(text, "-")
	whole, fraction, dotted := strings.Cut(body, ".")
	if whole == "" || len(whole) > maxIntDigits || dotted && (fraction == "" || len(fraction) > scale) {
		return nil, false
	}
	for _, part := range []string{whole, fraction} {
		for index := 0; index < len(part); index++ {
			if part[index] < '0' || part[index] > '9' {
				return nil, false
			}
		}
	}
	if len(whole) > 1 && whole[0] == '0' {
		return nil, false
	}
	value, _ := new(big.Int).SetString(whole+fraction+strings.Repeat("0", scale-len(fraction)), 10)
	if strings.HasPrefix(text, "-") {
		if value.Sign() == 0 {
			return nil, false
		}
		value.Neg(value)
	}
	return value, true
}

func canonical(text string) bool {
	_, ok := units(text, maxScale)
	return ok
}

func show(value *big.Int, scale int) string {
	negative := value.Sign() < 0
	digits := new(big.Int).Abs(value).String()
	for len(digits) <= scale {
		digits = "0" + digits
	}
	out := digits
	if scale > 0 {
		out = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if negative {
		out = "-" + out
	}
	return out
}

func clean(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r == utf8.RuneError, r < 0x20 && r != '\t' && r != '\n' && r != '\r', r == 0x7f, r >= 0xfffe && r <= 0xffff:
			b.WriteRune('�')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func risky(text string) bool {
	if canonical(text) {
		return false
	}
	trimmed := strings.TrimLeft(text, " ")
	if trimmed == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(trimmed)
	switch r {
	case '=', '+', '-', '@', '\t', '\r', '\'', '\uff1d', '\uff0b', '\uff0d', '\uff20':
		return true
	}
	return false
}

func neutral(text string) string {
	if risky(text) {
		return "'" + text
	}
	return text
}

func raw(text string) string {
	if strings.HasPrefix(text, "'") && risky(text[1:]) {
		return text[1:]
	}
	return text
}
