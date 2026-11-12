package intake

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var ten = big.NewInt(10)

type exact struct {
	coefficient *big.Int
	scale       int
}

func newExact() *exact { return &exact{coefficient: new(big.Int)} }

func (e *exact) add(d decimal.Decimal) {
	added := d.Coefficient()
	for e.scale < d.Scale() {
		e.coefficient.Mul(e.coefficient, ten)
		e.scale++
	}
	for scale := d.Scale(); scale < e.scale; scale++ {
		added.Mul(added, ten)
	}
	e.coefficient.Add(e.coefficient, added)
}

func (e *exact) reduced() (*big.Int, int) {
	value := new(big.Int).Set(e.coefficient)
	scale := e.scale
	quotient, remainder := new(big.Int), new(big.Int)
	for scale > 0 {
		quotient.QuoRem(value, ten, remainder)
		if remainder.Sign() != 0 {
			break
		}
		value.Set(quotient)
		scale--
	}
	return value, scale
}

func (e *exact) equals(d decimal.Decimal) bool {
	value, scale := e.reduced()
	return scale == d.Scale() && value.Cmp(d.Coefficient()) == 0
}

func (e *exact) text() string {
	value, scale := e.reduced()
	digits := new(big.Int).Abs(value).Text(10)
	if scale > 0 {
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		digits = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if value.Sign() < 0 {
		return "-" + digits
	}
	return digits
}

func texts(sums map[string]*exact) map[string]string {
	out := make(map[string]string, len(sums))
	for currency, sum := range sums {
		out[currency] = sum.text()
	}
	return out
}

func put(h hash.Hash, field string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(field)))
	h.Write(size[:])
	h.Write([]byte(field))
}

type row struct {
	amount   decimal.Decimal
	currency string
	digest   string
}

func (a row) same(b row) bool {
	return a.currency == b.currency && a.digest == b.digest && a.amount.Equal(b.amount)
}

func recordDigest(rows map[string]row) string {
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, key := range keys {
		held := rows[key]
		put(h, key)
		put(h, held.amount.String())
		put(h, held.currency)
		put(h, held.digest)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func chunkDigest(keys []string, rows []row) string {
	h := sha256.New()
	for index, key := range keys {
		put(h, key)
		put(h, rows[index].amount.String())
		put(h, rows[index].currency)
		put(h, rows[index].digest)
	}
	return hex.EncodeToString(h.Sum(nil))
}
