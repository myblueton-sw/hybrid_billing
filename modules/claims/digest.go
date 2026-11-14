package claims

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
	"strconv"
	"strings"

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

func applicationDigest(workspace string, a Application) string {
	p := newPrinter(applicationLabel)
	p.fields(workspace, a.ID, a.Scope.Contract, a.Scope.Component, a.Scope.Instance, a.Start, a.End, a.Mode, a.Source, a.CapQuantity, a.CapAmount,
		a.Quantity, a.Amount, a.Currency, a.Calculation, a.Replaces, a.CorrectsKind, a.Corrects)
	return p.hex()
}

func eligibilityDigest(workspace, contract string, number int, rev Revision) string {
	p := newPrinter(eligibilityLabel)
	p.fields(workspace, contract, strconv.Itoa(number), rev.From, rev.To, rev.Path, strconv.Itoa(len(rev.Components)))
	for _, c := range rev.Components {
		p.fields(c.ID, c.Kind, strconv.FormatBool(c.Billable), c.Route, c.Reference, strconv.Itoa(len(c.Instances)))
		p.fields(c.Instances...)
	}
	return p.hex()
}

func reversalDigest(workspace string, rev Reversal) string {
	p := newPrinter(reversalLabel)
	p.fields(workspace, rev.ID, rev.Application, rev.Quantity, rev.Amount)
	return p.hex()
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
