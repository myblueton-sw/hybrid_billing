package perfgate

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"strings"
)

const maxIDBytes = 128

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
