package marketplace

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"
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

func (p printer) number(value int) { p.field(strconv.Itoa(value)) }

func (p printer) fields(texts ...string) {
	for _, text := range texts {
		p.field(text)
	}
}

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func (p printer) transaction(t Transaction) {
	p.fields(t.ID, t.Provider, t.Agreement, t.Role, t.Channel, t.Type, t.Collector)
	q := t.Participants
	p.fields(q.SoftwareProvider, q.Seller, q.Reseller, q.Issuer, q.Payee)
	ref := t.References
	p.fields(ref.Product, ref.Offer, ref.Subscription, ref.Order)
	p.fields(t.Term.Accepted, t.Term.Start, t.Term.End, t.Term.Currency)
}

func statementDigest(workspace string, t Transaction, month string, number int, facts Facts) string {
	p := newPrinter(statementLabel)
	p.field(workspace)
	p.transaction(t)
	p.field(month)
	p.number(number)
	p.number(len(facts.Lines))
	for _, line := range facts.Lines {
		p.fields(line.Source, line.Kind, line.Account, line.Charged, line.List, line.Discount.State, line.Discount.Amount, line.Discount.Evidence)
	}
	p.number(len(facts.Recognitions))
	for _, rec := range facts.Recognitions {
		p.fields(rec.Program, rec.State, rec.Amount, rec.Evidence)
	}
	return p.hex()
}

func settlementDigest(workspace string, t Transaction, id string, number int, f SettlementFacts) string {
	p := newPrinter(settlementLbl)
	p.fields(workspace, t.ID, t.Provider, t.Agreement, id)
	p.number(number)
	p.fields(f.State, f.Currency, f.Gross, f.Fee, f.Refund, f.Tax, f.Hold, f.Net, f.Report, f.Payout, f.Received)
	return p.hex()
}

func policyDigest(workspace string, terms PolicyTerms) string {
	p := newPrinter(policyLabel)
	p.fields(workspace, terms.ID, terms.Version, terms.Customer, terms.Transaction, terms.First, terms.Last, terms.Rate)
	p.number(len(terms.Scope))
	p.fields(terms.Scope...)
	p.fields(terms.SourceDiscounted, terms.Evidence, terms.SourceBasis, terms.Quantum)
	return p.hex()
}

func chargeDigest(view ChargeView) string {
	p := newPrinter(ChargeLabel)
	p.fields(view.ProfileID, view.ProfileRev, view.Transaction, view.Month)
	p.number(view.Revision)
	p.fields(view.StatementHash, view.PolicyID, view.PolicyVersion, view.PolicyDigest, view.PolicyFirst, view.PolicyLast, view.Customer, view.Hold)
	p.number(len(view.Lines))
	for _, line := range view.Lines {
		p.fields(line.Source, line.Kind, line.Charged, line.Discount, line.Amount)
	}
	p.number(len(view.Recognitions))
	for _, rec := range view.Recognitions {
		p.fields(rec.Program, rec.State, rec.Amount, rec.Evidence)
	}
	p.fields(view.Total, view.ChargedTotal, view.Margin)
	return p.hex()
}
