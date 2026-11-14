package erpledger

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func refuse(reason string) error { return Refusal{Reason: reason} }

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

var monthNames = map[string]bool{"01": true, "02": true, "03": true, "04": true, "05": true, "06": true, "07": true, "08": true, "09": true, "10": true, "11": true, "12": true}

func validMonth(text string) bool {
	if len(text) != 7 || text[:2] != "20" || text[4] != '-' || !monthNames[text[5:]] {
		return false
	}
	return text[2] >= '0' && text[2] <= '9' && text[3] >= '0' && text[3] <= '9'
}

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (a Actor) refusal() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (r *Registry) number(text string) bool {
	if len(text) > maxNumberBytes {
		return false
	}
	_, err := decimal.Parse(r.numbers, text)
	return err == nil
}

func (r *Registry) units(text string, scale int) (*big.Int, bool) {
	value, _ := decimal.Parse(r.numbers, text)
	if value.Scale() > scale {
		return nil, false
	}
	out := new(big.Int).Mul(value.Coefficient(), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale-value.Scale())), nil))
	return out, true
}

func (r *Registry) bounded(units *big.Int, scale int) bool {
	_, err := decimal.New(r.numbers, units, scale)
	return err == nil
}

func show(units *big.Int, scale int) string {
	negative := units.Sign() < 0
	digits := new(big.Int).Abs(units).String()
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

func hashOf(texts ...string) string {
	sum := sha256.New()
	for _, text := range texts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		sum.Write(size[:])
		sum.Write([]byte(text))
	}
	return hex.EncodeToString(sum.Sum(nil))
}
