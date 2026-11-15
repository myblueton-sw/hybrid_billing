package support

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

const (
	maxIDBytes   = 128
	maxComponent = 6
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
