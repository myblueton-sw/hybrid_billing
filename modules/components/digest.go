package components

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type printer struct{ sum hash.Hash }

func newPrinter(label string) printer {
	p := printer{sum: sha256.New()}
	p.field(label)
	return p
}

func (p printer) field(text string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(text)))
	p.sum.Write(size[:])
	p.sum.Write([]byte(text))
}

func (p printer) fields(texts ...string) {
	for _, text := range texts {
		p.field(text)
	}
}

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }

func pricingDigest(workspace string, p Pricing, actor string) string {
	d := newPrinter(pricingLabel)
	d.fields(workspace, p.Contract, strconv.Itoa(p.Scale), p.Mode, actor)
	return d.hex()
}

func componentDigest(workspace string, c Component, actor string) string {
	p := newPrinter(componentLabel)
	p.fields(workspace, c.ID, c.Contract, c.Currency, c.Kind, c.From, c.To, c.Fee, c.Meter, c.Rate, strconv.Itoa(len(c.Includes)))
	for _, inc := range c.Includes {
		p.fields(inc.Meter, inc.Quantity)
	}
	p.field(actor)
	return p.hex()
}

func incidentDigest(workspace string, i Incident, actor string) string {
	p := newPrinter(incidentLabel)
	p.fields(workspace, i.ID, i.Contract, i.Component, stamp(i.Start), stamp(i.End), i.Evidence, actor)
	return p.hex()
}

func clauseDigest(workspace string, c Clause, actor string) string {
	p := newPrinter(clauseLabel)
	p.fields(workspace, c.ID, c.Contract, c.Component, c.Rate, c.From, c.To, actor)
	return p.hex()
}

func creditDigest(workspace string, c Credit, amount, actor string) string {
	p := newPrinter(creditLabel)
	p.fields(workspace, c.ID, c.Clause, c.PeriodStart, c.PeriodEnd, amount, strconv.Itoa(len(c.Incidents)))
	p.fields(c.Incidents...)
	p.field(actor)
	return p.hex()
}

func show(f exact.Fraction) string {
	text, _ := format(f)
	return text
}

func format(f exact.Fraction) (string, bool) {
	text := f.Text()
	num, den := text, "1"
	if index := strings.IndexByte(text, '/'); index >= 0 {
		num, den = text[:index], text[index+1:]
	}
	value, _ := new(big.Int).SetString(num, 10)
	divisor, _ := new(big.Int).SetString(den, 10)
	rest, twos, fives := new(big.Int).Set(divisor), 0, 0
	two, five, zero := big.NewInt(2), big.NewInt(5), new(big.Int)
	for new(big.Int).Rem(rest, two).Cmp(zero) == 0 {
		rest.Quo(rest, two)
		twos++
	}
	for new(big.Int).Rem(rest, five).Cmp(zero) == 0 {
		rest.Quo(rest, five)
		fives++
	}
	if rest.Cmp(big.NewInt(1)) != 0 {
		return "", false
	}
	scale := twos
	if fives > scale {
		scale = fives
	}
	units := new(big.Int).Mul(value, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	units.Quo(units, divisor)
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
	return out, true
}
