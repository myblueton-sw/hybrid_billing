package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type financialERPResultRow struct {
	ID, Line, Product, Unit, Net, Tax, Quantity string
}
type financialERPResultPacket struct {
	Mode, Profile, Evidence, Preparation, PreparationDigest, Connection, External, Currency, Customer, Period string
	Revision                                                                                                  int
	Rows                                                                                                      []financialERPResultRow
}
type financialERPResultBasis struct {
	Workspace, ID, SourceDigest, PacketDigest, DocumentDigest string
	Packet                                                    financialERPResultPacket
	Preparation                                               erpledger.SubmissionView
	Document                                                  financialDocumentBasis
	Comparison                                                []string
	Total                                                     erpledger.Amounts
}
type financialERPResultRecord struct {
	raw    string
	basis  []byte
	digest string
	review financialDocumentReview
}
type financialERPResultView struct {
	Basis              financialERPResultBasis
	Review             financialDocumentReview
	CurrentEligibility string
	Source             string
	ApplicationReceipt string
	ApplicationState   string
}

func resultSourceDigest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func parseResultPacket(raw string) (financialERPResultPacket, error) {
	var p financialERPResultPacket
	bad := func() (financialERPResultPacket, error) {
		return financialERPResultPacket{}, commands.Fail(commands.InvalidRequest)
	}
	if len(raw) == 0 || len(raw) > pricingPolicyFieldBytes || !pricingJSONUnicode(raw) {
		return bad()
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.UseNumber()
	if !pricingJSONShape(d, reflect.TypeOf(p), 0) {
		return bad()
	}
	if _, err := d.Token(); err != io.EOF {
		return bad()
	}
	if json.Unmarshal([]byte(raw), &p) != nil || p.Mode != "synthetic" || p.Profile != "synthetic_result_one_to_one_v1" || p.Revision < 1 || p.Revision > 2147483647 || len(p.Rows) < 1 || len(p.Rows) > financialDocumentLimit {
		return bad()
	}
	for _, id := range []string{p.Evidence, p.Preparation, p.Connection, p.External, p.Customer} {
		if !validRequestID(id) {
			return bad()
		}
	}
	if !validPlanHash(p.PreparationDigest) || p.Currency == "" || p.Period == "" {
		return bad()
	}
	seen := map[string]bool{}
	for _, row := range p.Rows {
		for _, id := range []string{row.ID, row.Line, row.Product, row.Unit} {
			if !validRequestID(id) {
				return bad()
			}
		}
		if seen[row.ID] {
			return bad()
		}
		seen[row.ID] = true
		if row.Net == "" || row.Tax == "" || row.Quantity == "" {
			return bad()
		}
	}
	return p, nil
}

func (a *financialERP) resultHistoricalPreparation(workspace string, p financialERPResultPacket) (financialDocumentBasis, erpledger.SubmissionView, string, error) {
	fail := func() (financialDocumentBasis, erpledger.SubmissionView, string, error) {
		return financialDocumentBasis{}, erpledger.SubmissionView{}, "", commands.Fail(commands.EvidenceStale)
	}
	owner, ok := a.owners[[2]string{workspace, p.Preparation}]
	if workspace != a.server.bootWorkspace || !ok || owner.receipt == "" || owner.receipt != p.PreparationDigest {
		return fail()
	}
	var b financialDocumentBasis
	found := false
	for key, r := range a.documents {
		if key[0] != workspace || r.review.Digest != owner.basis {
			continue
		}
		if found {
			return fail()
		}
		_, basis, err := a.governedDocument(workspace, key[1], owner.basis)
		if err != nil {
			return fail()
		}
		b = basis
		found = true
	}
	if !found || a.server.chargeProducer == nil {
		return fail()
	}
	request, binding, err := a.server.financialChargeRecord(workspace, b.Request)
	if err != nil || request.Digest != b.RequestDigest || binding.Candidate != b.Candidate || binding.CandidateDigest != b.CandidateDigest {
		return fail()
	}
	c, err := a.server.chargeProducer.candidates.financialCandidate(workspace, b.Candidate)
	if err != nil || c.Digest != b.CandidateDigest {
		return fail()
	}
	sub, total, err := documentSubmission(b, c, binding.Subject, p.Preparation)
	if err != nil || !reflect.DeepEqual(sub, owner.submission) || total != owner.total {
		return fail()
	}
	actual, err := a.ledger.SubmissionOf(workspace, p.Preparation)
	if err != nil {
		return financialDocumentBasis{}, erpledger.SubmissionView{}, "", commands.Fail(commands.DependencyUnavailable)
	}
	if !matchingPreparation(owner, actual) || p.Connection != sub.Connection || p.Currency != b.Document.Currency || p.Customer != sub.Customer || p.Period != sub.Period {
		return fail()
	}
	return b, actual, owner.basis, nil
}

func compareResult(p financialERPResultPacket, b financialDocumentBasis, sub erpledger.SubmissionView) ([]string, erpledger.Amounts, error) {
	profile, err := decimal.NewProfile(candidateProfile())
	if err != nil {
		return nil, erpledger.Amounts{}, err
	}
	zero, _ := decimal.Parse(profile, "0")
	net, tax, quantity := zero, zero, zero
	issues := map[string]bool{}
	seen := map[string]bool{}
	lines := map[string]erpledger.Line{}
	for _, line := range sub.Submission.Lines {
		lines[line.ID] = line
	}
	products := map[string]string{}
	for _, m := range b.Mappings {
		if m.Mapping.Kind == "product" {
			products[m.Mapping.Platform] = m.Mapping.External
		}
	}
	if len(p.Rows) != len(lines) {
		issues["line_coverage"] = true
	}
	for _, row := range p.Rows {
		n, e1 := documentNumber(profile, row.Net, b.Connection.Connection.Scale, false)
		t, e2 := documentNumber(profile, row.Tax, b.Connection.Connection.Scale, false)
		q, e3 := documentNumber(profile, row.Quantity, b.Connection.Connection.Scale, true)
		if e1 != nil || e2 != nil || e3 != nil {
			return nil, erpledger.Amounts{}, commands.Fail(commands.InvalidRequest)
		}
		net, e1 = decimal.Add(profile, net, n)
		tax, e2 = decimal.Add(profile, tax, t)
		quantity, e3 = decimal.Add(profile, quantity, q)
		if e1 != nil || e2 != nil || e3 != nil {
			return nil, erpledger.Amounts{}, commands.Fail(commands.InvalidRequest)
		}
		line, ok := lines[row.Line]
		if !ok || seen[row.Line] {
			issues["line_coverage"] = true
		}
		seen[row.Line] = true
		if !ok {
			continue
		}
		if row.Product != products[line.Product] {
			issues["product_mismatch"] = true
		}
		if len(b.Lines) == 0 || row.Unit != b.Lines[0].Unit {
			issues["unit_mismatch"] = true
		}
		if row.Net != line.Net {
			issues["net_mismatch"] = true
		}
		if row.Tax != line.Tax {
			issues["tax_mismatch"] = true
		}
		if row.Quantity != line.Quantity {
			issues["quantity_mismatch"] = true
		}
	}
	out := []string{}
	for issue := range issues {
		out = append(out, issue)
	}
	sort.Strings(out)
	return out, erpledger.Amounts{Net: documentFixed(net, b.Connection.Connection.Scale), Tax: documentFixed(tax, b.Connection.Connection.Scale), Quantity: documentFixed(quantity, b.Connection.Connection.Scale)}, nil
}

func resultBasisBytes(b financialERPResultBasis) ([]byte, error) {
	raw, err := json.Marshal(b)
	if err != nil || len(raw) > financialDocumentBytes {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	return raw, nil
}

func resultLineAmounts(line erpledger.Line, scale int) erpledger.Amounts {
	profile, err := decimal.NewProfile(candidateProfile())
	if err != nil {
		return erpledger.Amounts{}
	}
	n, e1 := documentNumber(profile, line.Net, scale, false)
	t, e2 := documentNumber(profile, line.Tax, scale, false)
	q, e3 := documentNumber(profile, line.Quantity, scale, true)
	if e1 != nil || e2 != nil || e3 != nil {
		return erpledger.Amounts{}
	}
	return erpledger.Amounts{Net: documentFixed(n, scale), Tax: documentFixed(t, scale), Quantity: documentFixed(q, scale)}
}
