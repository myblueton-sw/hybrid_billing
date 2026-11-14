package receivables

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
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

func termDigest(workspace string, t Term, actor string) string {
	p := newPrinter(termLabel)
	p.fields(workspace, t.ID, itoa(t.Revision), itoa(t.NetDays), t.Mode, actor)
	return p.hex()
}

func bindingDigest(workspace string, b Binding, actor string) string {
	p := newPrinter(bindingLabel)
	p.fields(workspace, b.ID, b.Contract, b.Term, itoa(b.Revision), b.Zone, b.From, b.To, b.Reason, actor)
	return p.hex()
}

func documentDigest(workspace string, v Receivable, s Snapshot, actor string) string {
	p := newPrinter(documentLabel)
	p.fields(workspace, v.Document, v.Customer, v.Entity, v.Currency, v.Contract, v.Amount, stamp(v.IssuedAt), v.Obligation, v.Replaces, v.DueBasis, actor)
	p.fields(s.Binding, s.Term, itoa(s.Revision), itoa(s.NetDays), s.Mode, s.Zone, s.IssueDate, s.OriginalDue, s.EffectiveDue)
	return p.hex()
}

func extensionDigest(workspace string, e Extension, actor string) string {
	p := newPrinter(extensionLabel)
	p.fields(workspace, e.ID, e.Document, e.Due, e.Reason, actor)
	return p.hex()
}

func allocationDigest(workspace string, a Allocation, actor string) string {
	p := newPrinter(movementLabel)
	p.fields(workspace, a.ID, a.Document, a.Kind, a.Amount, a.Effective, a.Source, a.Ledger, a.Revision, a.Evidence, actor)
	return p.hex()
}

func reversalDigest(workspace string, v Reversal, actor string) string {
	p := newPrinter(reversalLabel)
	p.fields(workspace, v.ID, v.Allocation, v.Effective, v.Source, v.Ledger, v.Revision, v.Evidence, actor)
	return p.hex()
}

func holdDigest(workspace string, h Hold, actor string) string {
	p := newPrinter(holdLabel)
	p.fields(workspace, h.ID, h.Document, h.Kind, h.Reference, actor)
	return p.hex()
}

func (r *Registry) print(f exact.Fraction) string {
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
