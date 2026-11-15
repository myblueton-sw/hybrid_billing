package simulation

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"
)

const (
	maxIDBytes  = 128
	maxURLBytes = 200
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

func validHex(text string) bool {
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

type encoder struct {
	out bytes.Buffer
}

func (e *encoder) text(value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	e.out.Write(size[:])
	e.out.WriteString(value)
}

func (e *encoder) number(value int64) { e.text(strconv.FormatInt(value, 10)) }

func (e *encoder) count(value int) { e.number(int64(value)) }

func digits(text string, limit int) bool {
	if text == "" || len(text) > limit || len(text) > 1 && text[0] == '0' {
		return false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return true
}

func semver(text string) bool {
	parts := strings.Split(text, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if !digits(part, 9) {
			return false
		}
	}
	return true
}

func semverLess(a, b string) bool {
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for index := range x {
		if len(x[index]) != len(y[index]) {
			return len(x[index]) < len(y[index])
		}
		if x[index] != y[index] {
			return x[index] < y[index]
		}
	}
	return false
}

func validSeed(text string) bool {
	if !digits(text, 20) {
		return false
	}
	_, err := strconv.ParseUint(text, 10, 64)
	return err == nil
}

func validDecimal(text string) bool {
	if len(text) > maxDecimalBytes || text == "-0" {
		return false
	}
	body := strings.TrimPrefix(text, "-")
	whole, fraction, dotted := strings.Cut(body, ".")
	if !digits(whole, maxDecimalBytes) {
		return false
	}
	if !dotted {
		return true
	}
	if fraction == "" || fraction[len(fraction)-1] == '0' {
		return false
	}
	for index := 0; index < len(fraction); index++ {
		if fraction[index] < '0' || fraction[index] > '9' {
			return false
		}
	}
	return true
}

func validCurrency(text string) bool {
	if len(text) != 3 {
		return false
	}
	for index := 0; index < 3; index++ {
		if text[index] < 'A' || text[index] > 'Z' {
			return false
		}
	}
	return true
}

func validStatus(text string) bool {
	return validID(text) && text[0] >= 'a'
}

func validPeriod(text string) bool {
	if len(text) != 7 || text[4] != '-' || !digits(text[:4], 4) || text[6] < '0' || text[6] > '9' {
		return false
	}
	month := text[5:]
	return month >= "01" && month <= "12"
}

func validZone(text string) bool {
	if text == "" || len(text) > maxZoneBytes {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '/' || c == '_' || c == '+' || c == '-') {
			return false
		}
	}
	return true
}

func validSource(text string) bool {
	rest, ok := strings.CutPrefix(text, "https://")
	if !ok || rest == "" || len(text) > maxURLBytes || strings.ContainsAny(text, "@?#") {
		return false
	}
	for index := 0; index < len(rest); index++ {
		if rest[index] <= 0x20 || rest[index] >= 0x7f {
			return false
		}
	}
	return true
}

func ascending(values []string, low, high int, valid func(string) bool) bool {
	if len(values) < low || len(values) > high {
		return false
	}
	for index, value := range values {
		if !valid(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
