package balances

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
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

func accountDigest(workspace string, a Account) string {
	p := newPrinter(accountLabel)
	p.fields(workspace, a.ID, a.Customer, a.Entity, a.Currency, a.Kind, a.Authority, a.Ledger)
	return p.hex()
}

func openingDigest(workspace string, o Opening) string {
	p := newPrinter(openingLabel)
	p.fields(workspace, o.ID, o.Account, o.PriorLedger, o.AsOf, o.Amount, o.Evidence, o.MigrationKey, o.IncludedFrom, o.ExpiresOn)
	return p.hex()
}

func increaseDigest(workspace string, i Increase) string {
	p := newPrinter(increaseLabel)
	p.fields(workspace, i.ID, i.Account, i.Kind, i.Amount, i.Reference, i.ExpiresOn, i.Import.Ledger, i.Import.Transaction, i.Import.Occurred)
	return p.hex()
}

func applicationDigest(workspace string, a Application) string {
	p := newPrinter(applicationLabel)
	partial := "false"
	if a.Partial {
		partial = "true"
	}
	p.fields(workspace, a.ID, a.Account, a.Target.Document, a.Target.Line, a.Target.Amount, a.Requested, partial, a.Import.Ledger, a.Import.Transaction, a.Import.Occurred)
	return p.hex()
}

func reversalDigest(workspace string, rev Reversal) string {
	p := newPrinter(reversalLabel)
	p.fields(workspace, rev.ID, rev.Kind, rev.Record, rev.Amount)
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
