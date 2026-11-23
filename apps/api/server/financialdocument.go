package server

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const financialDocumentLimit = 100
const financialDocumentBytes = 128 << 10

type financialDocumentLine struct {
	Line, Rule, Kind, Product, Quantity, Unit, TaxCode, Tax, Evidence string
}

type financialDocumentBasis struct {
	ID, Mode, Profile, Evidence, Workspace, Request, RequestDigest, Candidate, CandidateDigest, Path string
	Document                                                                                         organizations.DocumentKey
	Connection                                                                                       erpledger.ConnectionView
	Mappings                                                                                         []erpledger.MappingView
	Survey                                                                                           erpledger.SurveyView
	ExpectedNet, ExpectedTax                                                                         string
	Lines                                                                                            []financialDocumentLine
}

func documentBytes(b financialDocumentBasis) ([]byte, error) {
	if len(b.Lines) < 1 || len(b.Lines) > financialDocumentLimit || len(b.Mappings) > 2*financialDocumentLimit+1 || len(b.Connection.Connection.Capabilities) > 20 {
		return nil, errCandidateLimit
	}
	remaining := financialDocumentBytes
	if !candidateText(reflect.ValueOf(b), &remaining) {
		return nil, errCandidateLimit
	}
	raw, err := json.Marshal(b)
	if err != nil || len(raw) > financialDocumentBytes {
		return nil, errCandidateLimit
	}
	return raw, nil
}

func documentNumber(profile decimal.Profile, text string, scale int, positive bool) (decimal.Decimal, error) {
	if len(text) > 64 {
		return decimal.Decimal{}, errCandidateInvalid
	}
	n, err := decimal.Parse(profile, text)
	if err != nil || !n.Valid() || n.String() != text || n.Scale() > scale || n.Sign() < 0 || positive && n.Sign() == 0 {
		return decimal.Decimal{}, errCandidateInvalid
	}
	return n, nil
}

func documentSubmission(b financialDocumentBasis, c chargeCandidate, subject approvals.Subject, key string) (erpledger.Submission, erpledger.Amounts, error) {
	bad := func() (erpledger.Submission, erpledger.Amounts, error) {
		return erpledger.Submission{}, erpledger.Amounts{}, errCandidateInvalid
	}
	if _, err := documentBytes(b); err != nil {
		return bad()
	}
	for _, id := range []string{b.ID, b.Evidence, b.Workspace, b.Request, b.Candidate, key, b.Connection.Connection.ID, b.Connection.Author, b.Survey.Author, b.Survey.Survey.Evidence} {
		if !validRequestID(id) {
			return bad()
		}
	}
	in := c.Envelope.Input
	origin := in.Normalized
	conn := b.Connection.Connection
	if b.Mode != "synthetic" || b.Profile != "synthetic_individual_net_v1" || b.Path != erpledger.PathDirect || b.Workspace != in.Workspace || b.Candidate != in.ID || b.CandidateDigest != c.Digest || b.Request != subject.ID || subject.Digests["result"] != c.Digest || b.Document != origin.Document || b.Document.WorkspaceID != b.Workspace || b.Document.Path != organizations.ExternalInvoice || b.Document.Currency != subject.Currency || conn.Currency != subject.Currency || conn.Entity != b.Document.IssuerLegalEntityID || conn.Scale < 0 || conn.Scale > 4 || b.Connection.At.IsZero() {
		return bad()
	}
	if b.Survey.Survey.Connection != conn.ID || b.Survey.Survey.Customer != in.Customer || b.Survey.Survey.Period != origin.Binding.Period || b.Survey.Survey.Outcome != erpledger.SurveyComplete || b.Survey.At.IsZero() || len(in.Lines) != len(b.Lines) {
		return bad()
	}
	charges, sources, err := documentCandidateShape(c, subject, b.Connection)
	if err != nil {
		return bad()
	}
	profile, err := decimal.NewProfile(candidateProfile())
	if err != nil {
		return bad()
	}
	zero, _ := decimal.Parse(profile, "0")
	net, tax, quantity := zero, zero, zero
	sub := erpledger.Submission{Key: key, Connection: conn.ID, Purpose: erpledger.PurposeInvoice, Customer: in.Customer, Period: origin.Binding.Period, Path: b.Path}
	seen := map[string]bool{}
	unit := b.Lines[0].Unit
	for _, line := range b.Lines {
		charge, ok := charges[line.Line]
		if !ok || seen[line.Line] || charge.Rule != line.Rule || charge.Kind != line.Kind || line.Unit != unit {
			return bad()
		}
		for _, id := range []string{line.Product, line.Unit, line.TaxCode, line.Evidence} {
			if !validRequestID(id) {
				return bad()
			}
		}
		q, e1 := documentNumber(profile, line.Quantity, conn.Scale, true)
		t, e2 := documentNumber(profile, line.Tax, conn.Scale, false)
		amount, e3 := documentNumber(profile, charge.Amount, conn.Scale, true)
		if e1 != nil || e2 != nil || e3 != nil {
			return bad()
		}
		net, e1 = decimal.Add(profile, net, amount)
		tax, e2 = decimal.Add(profile, tax, t)
		quantity, e3 = decimal.Add(profile, quantity, q)
		if e1 != nil || e2 != nil || e3 != nil {
			return bad()
		}
		source := sources[line.Line]
		seen[line.Line] = true
		sub.Lines = append(sub.Lines, erpledger.Line{ID: "line-" + source, Source: "source-" + source, Product: line.Product, TaxCode: line.TaxCode, Quantity: line.Quantity, Net: charge.Amount, Tax: line.Tax})
	}
	if net.String() != subject.Amount || b.ExpectedNet != net.String() || b.ExpectedTax != tax.String() {
		return bad()
	}
	required := map[string]bool{"customer\x00" + in.Customer: true}
	for _, line := range sub.Lines {
		required["product\x00"+line.Product] = true
		required["tax\x00"+line.TaxCode] = true
	}
	if len(b.Mappings) != len(required) {
		return bad()
	}
	for _, mapping := range b.Mappings {
		id := mapping.Mapping.Kind + "\x00" + mapping.Mapping.Platform
		if !required[id] || mapping.Mapping.Connection != conn.ID || !validRequestID(mapping.Mapping.External) || !validRequestID(mapping.Author) || mapping.Revision < 1 || mapping.At.IsZero() {
			return bad()
		}
		delete(required, id)
	}
	if len(required) != 0 {
		return bad()
	}
	return sub, erpledger.Amounts{Net: documentFixed(net, conn.Scale), Tax: documentFixed(tax, conn.Scale), Quantity: documentFixed(quantity, conn.Scale)}, nil
}

func documentFixed(value decimal.Decimal, scale int) string {
	whole, fraction, _ := strings.Cut(value.String(), ".")
	if scale == 0 {
		return whole
	}
	return whole + "." + fraction + strings.Repeat("0", scale-len(fraction))
}

func documentCandidateShape(c chargeCandidate, subject approvals.Subject, connection erpledger.ConnectionView) (map[string]rating.Charge, map[string]string, error) {
	bad := func() (map[string]rating.Charge, map[string]string, error) { return nil, nil, errCandidateInvalid }
	in, result := c.Envelope.Input, c.Envelope.Result
	origin, conn := in.Normalized, connection.Connection
	if subject.Digests["result"] != c.Digest || origin.Document.WorkspaceID != in.Workspace || origin.Document.Path != organizations.ExternalInvoice || origin.Document.Currency != subject.Currency || conn.Currency != subject.Currency || conn.Entity != origin.Document.IssuerLegalEntityID || conn.Scale < 0 || conn.Scale > 4 || connection.At.IsZero() {
		return bad()
	}
	from, e1 := time.Parse(time.DateOnly, in.From)
	to, e2 := time.Parse(time.DateOnly, in.To)
	if e1 != nil || e2 != nil || from.Day() != 1 || !to.AddDate(0, 0, 1).Equal(from.AddDate(0, 1, 0)) || origin.Binding.Period != from.Format("2006-01") {
		return bad()
	}
	n := len(in.Lines)
	if n < 1 || n > financialDocumentLimit || n != len(result.Charges) || n != len(origin.Rows) || len(result.Excluded) != 0 || result.Total != subject.Amount || result.Currency != subject.Currency {
		return bad()
	}
	input := map[string]bool{}
	rows := map[string]normalize.RowKey{}
	charges := map[string]rating.Charge{}
	sources := map[string]string{}
	seen := map[string]bool{}
	for _, line := range in.Lines {
		if !validRequestID(line.ID) || input[line.ID] || line.Currency != subject.Currency {
			return bad()
		}
		input[line.ID] = true
	}
	for _, row := range origin.Rows {
		if !input[row.Line] || rows[row.Line].Kind != "" || !validRequestID(row.Key.Key) || row.Key.Kind != normalize.EventKeys && row.Key.Kind != normalize.LocatorKeys || row.Key.Kind == normalize.EventKeys && row.Key.Document != "" || row.Key.Kind == normalize.LocatorKeys && !validRequestID(row.Key.Document) {
			return bad()
		}
		rows[row.Line] = row.Key
		source, err := normalizedDigest(struct {
			Schema, Workspace, Target string
			Row                       normalize.RowKey
		}{"hb_erp_economic_source_v1", in.Workspace, subject.Target, row.Key})
		if err != nil || seen[source] {
			return bad()
		}
		seen[source], sources[row.Line] = true, source
	}
	profile, err := decimal.NewProfile(candidateProfile())
	if err != nil {
		return bad()
	}
	total, _ := decimal.Parse(profile, "0")
	for _, charge := range result.Charges {
		if charge.Kind != rating.Rated || !input[charge.Line] || charges[charge.Line].Line != "" {
			return bad()
		}
		amount, err := documentNumber(profile, charge.Amount, conn.Scale, true)
		if err != nil {
			return bad()
		}
		total, err = decimal.Add(profile, total, amount)
		if err != nil {
			return bad()
		}
		charges[charge.Line] = charge
	}
	if total.String() != subject.Amount {
		return bad()
	}
	return charges, sources, nil
}
