package server

import (
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type financialDocumentPacketLine struct {
	Line, Product, Quantity, Unit, TaxCode, Tax, Evidence string
}

type financialDocumentPacket struct {
	Mode, Profile, Evidence, Request, RequestDigest, Connection, ExpectedNet, ExpectedTax string
	Lines                                                                                 []financialDocumentPacketLine
}

func parseDocumentPacket(raw string) (financialDocumentPacket, error) {
	var p financialDocumentPacket
	bad := func() (financialDocumentPacket, error) {
		return financialDocumentPacket{}, commands.Fail(commands.InvalidRequest)
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
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return bad()
	}
	if p.Mode != "synthetic" || p.Profile != "synthetic_individual_net_v1" || len(p.Lines) < 1 || len(p.Lines) > financialDocumentLimit {
		return bad()
	}
	for _, text := range []string{p.Evidence, p.Request, p.RequestDigest, p.Connection, p.ExpectedNet, p.ExpectedTax} {
		if strings.TrimSpace(text) == "" {
			return bad()
		}
	}
	seen := map[string]bool{}
	for _, line := range p.Lines {
		if seen[line.Line] {
			return bad()
		}
		seen[line.Line] = true
		for _, text := range []string{line.Line, line.Product, line.Quantity, line.Unit, line.TaxCode, line.Tax, line.Evidence} {
			if strings.TrimSpace(text) == "" {
				return bad()
			}
		}
	}
	return p, nil
}

func (a *financialERP) buildDocumentPacketLocked(workspace, id, actor, raw string) (financialDocumentBasis, error) {
	bad := func(err error) (financialDocumentBasis, error) { return financialDocumentBasis{}, err }
	if err := a.documentPermission(workspace, actor, gate.ProposeDocumentEvidence); err != nil {
		return bad(err)
	}
	if !validRequestID(id) {
		return bad(commands.Fail(commands.InvalidRequest))
	}
	p, err := parseDocumentPacket(raw)
	if err != nil {
		return bad(err)
	}
	if a.server.held() {
		return bad(commands.Fail(commands.PrerequisiteFailed))
	}
	if _, err = a.server.currentFinancialChargeLocked(workspace, p.Request, p.RequestDigest, actor); err != nil {
		return bad(err)
	}
	_, binding, err := a.server.financialChargeRecord(workspace, p.Request)
	if err != nil {
		return bad(financialChargeEvidenceError(err))
	}
	if a.server.chargeProducer == nil {
		return bad(commands.Fail(commands.UnsupportedCapability))
	}
	candidate, err := a.server.chargeProducer.candidates.financialCandidate(workspace, binding.Candidate)
	if err != nil {
		return bad(financialChargeEvidenceError(err))
	}
	if candidate.Digest != binding.CandidateDigest {
		return bad(commands.Fail(commands.EvidenceStale))
	}
	b := financialDocumentBasis{ID: id, Mode: p.Mode, Profile: p.Profile, Evidence: p.Evidence, Workspace: workspace, Request: p.Request, RequestDigest: p.RequestDigest, Candidate: binding.Candidate, CandidateDigest: binding.CandidateDigest, Path: erpledger.PathDirect, Document: candidate.Envelope.Input.Normalized.Document, ExpectedNet: p.ExpectedNet, ExpectedTax: p.ExpectedTax}
	b.Connection, err = a.ledger.ConnectionOf(workspace, p.Connection)
	if err != nil {
		return bad(commands.Fail(commands.DependencyUnavailable))
	}
	keys := map[[2]string]bool{{"customer", candidate.Envelope.Input.Customer}: true}
	for _, line := range p.Lines {
		derived := financialDocumentLine{Line: line.Line, Product: line.Product, Quantity: line.Quantity, Unit: line.Unit, TaxCode: line.TaxCode, Tax: line.Tax, Evidence: line.Evidence}
		for _, charge := range candidate.Envelope.Result.Charges {
			if charge.Line == line.Line {
				derived.Rule, derived.Kind = charge.Rule, charge.Kind
			}
		}
		b.Lines = append(b.Lines, derived)
		keys[[2]string{"product", line.Product}], keys[[2]string{"tax", line.TaxCode}] = true, true
	}
	ordered := make([][2]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i][0] != ordered[j][0] {
			return ordered[i][0] < ordered[j][0]
		}
		return ordered[i][1] < ordered[j][1]
	})
	for _, key := range ordered {
		mapping, err := a.ledger.MappingOf(workspace, p.Connection, key[0], key[1])
		if err != nil {
			return bad(commands.Fail(commands.DependencyUnavailable))
		}
		b.Mappings = append(b.Mappings, mapping)
	}
	b.Survey, err = a.ledger.SurveyOf(workspace, p.Connection, candidate.Envelope.Input.Customer, candidate.Envelope.Input.Normalized.Binding.Period)
	if err != nil {
		return bad(commands.Fail(commands.DependencyUnavailable))
	}
	if _, err = a.validateDocumentLocked(b); err != nil {
		return bad(financialChargeEvidenceError(err))
	}
	return b, nil
}
