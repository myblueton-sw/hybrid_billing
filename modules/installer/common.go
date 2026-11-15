package installer

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"math/big"
	"strconv"
	"unicode/utf8"
)

const (
	maxIDBytes   = 128
	maxTextBytes = 200
	maxComponent = 6
)

var (
	fieldPrime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	smallOrder = []string{
		"0100000000000000000000000000000000000000000000000000000000000000",
		"0100000000000000000000000000000000000000000000000000000000000080",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"0000000000000000000000000000000000000000000000000000000000000080",
		"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05",
		"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85",
		"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a",
		"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa",
	}
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
	if text == "" || len(text) > maxTextBytes || !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 {
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

func semver(text string) ([3]int, bool) {
	var out [3]int
	parts := bytes.Split([]byte(text), []byte("."))
	if len(parts) != 3 {
		return out, false
	}
	for index, part := range parts {
		if len(part) == 0 || len(part) > maxComponent || len(part) > 1 && part[0] == '0' {
			return out, false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return out, false
			}
		}
		out[index], _ = strconv.Atoi(string(part))
	}
	return out, true
}

func compare(a, b string) int {
	x, _ := semver(a)
	y, _ := semver(b)
	for index := range x {
		switch {
		case x[index] < y[index]:
			return -1
		case x[index] > y[index]:
			return 1
		}
	}
	return 0
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

func hexBytes(text string) []byte {
	out := make([]byte, len(text)/2)
	for index := range out {
		out[index] = nibble(text[2*index])<<4 | nibble(text[2*index+1])
	}
	return out
}

func nibble(c byte) byte {
	if c >= 'a' {
		return c - 'a' + 10
	}
	return c - '0'
}

func weak(public []byte) bool {
	if len(public) != ed25519.PublicKeySize {
		return true
	}
	for _, known := range smallOrder {
		if bytes.Equal(public, hexBytes(known)) {
			return true
		}
	}
	reversed := make([]byte, len(public))
	for index, b := range public {
		reversed[len(public)-1-index] = b
	}
	reversed[0] &= 0x7f
	return new(big.Int).SetBytes(reversed).Cmp(fieldPrime) >= 0
}
