package commitment

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type Period struct {
	made       bool
	profile    decimal.Profile
	limits     Limits
	commitment Commitment
	month      string
	revision   int
	facts      Facts
	digest     string
}

type PeriodView struct {
	Commitment Commitment
	Month      string
	Revision   int
	Facts      Facts
	Digest     string
}

func (p Period) View() PeriodView {
	facts := p.facts
	facts.Lines = append([]Line{}, p.facts.Lines...)
	return PeriodView{Commitment: p.commitment, Month: p.month, Revision: p.revision, Facts: facts, Digest: p.digest}
}

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

func (p printer) number(value int) { p.field(strconv.Itoa(value)) }

func (p printer) fields(texts ...string) {
	for _, text := range texts {
		p.field(text)
	}
}

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func (p printer) commitment(c Commitment) {
	p.fields(c.ID, c.Provider, c.PayerAccount, c.Kind, c.Currency, c.First, c.Last, c.Payment, c.Prepaid, c.Quantum, c.Recovery)
}

func periodDigest(workspace string, c Commitment, month string, number int, facts Facts) string {
	p := newPrinter(periodLabel)
	p.field(workspace)
	p.commitment(c)
	p.field(month)
	p.number(number)
	p.fields(facts.Recurring, facts.Amortized)
	p.number(len(facts.Lines))
	for _, line := range facts.Lines {
		p.fields(line.Source, line.Kind, line.Commitment, line.Account, line.Recurring, line.Amortized, line.List)
	}
	return p.hex()
}
