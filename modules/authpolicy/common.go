package authpolicy

import (
	"bytes"
	"encoding/binary"
	"time"
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
	x := c.Revision
	e.text(x.Workspace)
	e.list(x.Methods)
	e.list(x.Issuers)
	e.text(x.MFA)
	e.number(int64(x.MaxAge))
	e.number(int64(len(x.StepUp)))
	for _, s := range x.StepUp {
		e.text(s.Action)
		e.number(int64(s.MaxAge))
		if s.MFA {
			e.number(1)
		} else {
			e.number(0)
		}
	}
	p := c.Plan
	e.text(p.ID)
	e.text(p.Workspace)
	e.list(p.Users)
	e.number(int64(p.Chunk))
	e.text(p.Reason)
	e.text(c.Lift.Plan)
	e.text(c.Lift.Reason)
	return e.out.String()
}

func copyStrings(list []string) []string {
	if list == nil {
		return nil
	}
	return append([]string{}, list...)
}

func copyChange(c Change) Change {
	c.Revision.Methods = copyStrings(c.Revision.Methods)
	c.Revision.Issuers = copyStrings(c.Revision.Issuers)
	if c.Revision.StepUp != nil {
		c.Revision.StepUp = append([]StepUp{}, c.Revision.StepUp...)
	}
	c.Plan.Users = copyStrings(c.Plan.Users)
	return c
}

func ageOK(age time.Duration, high time.Duration) bool { return age >= MinAge && age <= high }
