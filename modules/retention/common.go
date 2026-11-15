package retention

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"unicode/utf8"
)

const (
	maxIDBytes   = 128
	maxTextBytes = 200
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

func oneOf(value string, list ...string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func ascending(list []string, low, high int, member func(string) bool) bool {
	if len(list) < low || len(list) > high {
		return false
	}
	for index, item := range list {
		if !member(item) || index > 0 && list[index-1] >= item {
			return false
		}
	}
	return true
}

func contains(list []string, value string) bool {
	low, high := 0, len(list)
	for low < high {
		mid := (low + high) / 2
		if list[mid] < value {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low < len(list) && list[low] == value
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

func (e *encoder) number(value int64) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(value))
	e.out.Write(size[:])
}

func (e *encoder) list(values []string) {
	e.number(int64(len(values)))
	for _, value := range values {
		e.text(value)
	}
}

func encodeChange(c Change) string {
	e := &encoder{}
	e.text(c.Kind)
	e.text(c.Policy.Class)
	e.number(int64(c.Policy.Period))
	e.number(int64(c.Policy.Grace))
	p := c.Plan
	e.text(p.Class)
	e.list(p.Artifacts)
	e.number(int64(p.Chunk))
	e.number(int64(p.Lease))
	e.text(p.Reason)
	e.text(c.Release.Hold)
	e.text(c.Release.Reason)
	return e.out.String()
}

func copyStrings(list []string) []string {
	if list == nil {
		return nil
	}
	return append([]string{}, list...)
}

func copyChange(c Change) Change {
	c.Plan.Artifacts = copyStrings(c.Plan.Artifacts)
	return c
}

func digestIDs(ids []string) string {
	e := &encoder{}
	for _, id := range ids {
		e.text(id)
	}
	sum := sha256.Sum256(e.out.Bytes())
	return hex.EncodeToString(sum[:])
}
