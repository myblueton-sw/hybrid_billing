package server

import (
	"encoding/json"
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (a *financialERP) documentContextERP(workspace string, c chargeCandidate) (erpledger.ConnectionView, error) {
	connection, err := a.ledger.ConnectionOf(workspace, syntheticDocumentConnection)
	if err != nil {
		return connection, commands.Fail(commands.DependencyUnavailable)
	}
	conn := connection.Connection
	if conn.ID != syntheticDocumentConnection || !validRequestID(connection.Author) || !slices.Contains(conn.Capabilities, erpledger.CommandPrepare) {
		return connection, commands.Fail(commands.PrerequisiteFailed)
	}
	for _, key := range [][2]string{{"customer", c.Envelope.Input.Customer}, {"product", syntheticDocumentProduct}, {"tax", syntheticDocumentTax}} {
		mapping, err := a.ledger.MappingOf(workspace, conn.ID, key[0], key[1])
		if err != nil {
			return connection, commands.Fail(commands.DependencyUnavailable)
		}
		if mapping.Mapping.Connection != conn.ID || mapping.Mapping.Kind != key[0] || mapping.Mapping.Platform != key[1] || !validRequestID(mapping.Mapping.External) || !validRequestID(mapping.Author) || mapping.Revision < 1 || mapping.At.IsZero() {
			return connection, commands.Fail(commands.EvidenceStale)
		}
	}
	survey, err := a.ledger.SurveyOf(workspace, conn.ID, c.Envelope.Input.Customer, c.Envelope.Input.Normalized.Binding.Period)
	if err != nil {
		return connection, commands.Fail(commands.DependencyUnavailable)
	}
	if survey.Survey.Connection != conn.ID || survey.Survey.Customer != c.Envelope.Input.Customer || survey.Survey.Period != c.Envelope.Input.Normalized.Binding.Period || survey.Survey.Outcome != erpledger.SurveyComplete || !validRequestID(survey.Author) || !validRequestID(survey.Survey.Evidence) || survey.At.IsZero() {
		return connection, commands.Fail(commands.EvidenceStale)
	}
	return connection, nil
}

func (a *financialERP) documentContextLocked(ctx commands.Context) (map[string]string, error) {
	if err := a.documentPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	subject, err := a.server.currentFinancialChargeLocked(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User)
	if err != nil {
		return nil, err
	}
	_, binding, err := a.server.financialChargeRecord(ctx.Workspace, ctx.Target)
	if err != nil {
		return nil, financialChargeEvidenceError(err)
	}
	if a.server.chargeProducer == nil {
		return nil, commands.Fail(commands.UnsupportedCapability)
	}
	candidate, err := a.server.chargeProducer.candidates.financialCandidate(ctx.Workspace, binding.Candidate)
	if err != nil {
		return nil, financialChargeEvidenceError(err)
	}
	if candidate.Digest != binding.CandidateDigest || candidate.Envelope.Input.ID != binding.Candidate || candidate.Envelope.Input.Workspace != ctx.Workspace {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	connection, err := a.documentContextERP(ctx.Workspace, candidate)
	if err != nil {
		return nil, err
	}
	charges, _, err := documentCandidateShape(candidate, subject, connection)
	if err != nil {
		return nil, financialChargeEvidenceError(err)
	}
	in := candidate.Envelope.Input
	header, err := json.Marshal(map[string]any{"candidate": binding.Candidate, "candidate_digest": binding.CandidateDigest, "profile": "synthetic_individual_net_v1", "currency": subject.Currency, "total": subject.Amount, "customer": in.Customer, "period": in.Normalized.Binding.Period, "organization": in.Normalized.Document, "erp": map[string]any{"connection": connection.Connection.ID, "product": syntheticDocumentProduct, "tax_code": syntheticDocumentTax, "scale": connection.Connection.Scale}})
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	var rows []documentHistoryRow
	for _, charge := range charges {
		rows = append(rows, documentHistoryRow{id: charge.Line, value: map[string]string{"line": charge.Line, "rule": charge.Rule, "kind": charge.Kind, "amount": charge.Amount}})
	}
	permission, err := json.Marshal(documentPermissionAdvice(a.documentPermission(ctx.Workspace, ctx.User, gate.ProposeDocumentEvidence)))
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	fields, err := documentPage(rows, map[string]string{"mode": "synthetic", "request": ctx.Target, "request_digest": ctx.Fields["digest"], "header": string(header), "proposal_permission": string(permission)}, ctx.Fields["after"], ctx.Fields["snapshot"], ctx.Resolved)
	if err != nil {
		return nil, err
	}
	if err = a.boundedDocumentFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
