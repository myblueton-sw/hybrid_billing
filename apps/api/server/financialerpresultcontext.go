package server

import (
	"encoding/json"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (a *financialERP) resultPreparationBasis(workspace, key, digest string) (financialDocumentBasis, financialERPOwnership, error) {
	bad := func() (financialDocumentBasis, financialERPOwnership, error) {
		return financialDocumentBasis{}, financialERPOwnership{}, commands.Fail(commands.EvidenceStale)
	}
	owner, ok := a.owners[[2]string{workspace, key}]
	if workspace != a.server.bootWorkspace || !ok || !validPlanHash(digest) || owner.receipt != digest || owner.submission.Key != key || !validRequestID(owner.actor) {
		return bad()
	}
	var basis financialDocumentBasis
	found := false
	for docKey, r := range a.documents {
		if docKey[0] != workspace || r.review.Digest != owner.basis {
			continue
		}
		if found {
			return bad()
		}
		_, b, err := a.governedDocument(workspace, docKey[1], owner.basis)
		if err != nil {
			return bad()
		}
		basis = b
		found = true
	}
	if !found {
		return bad()
	}
	if a.server.chargeProducer == nil {
		return financialDocumentBasis{}, financialERPOwnership{}, commands.Fail(commands.DependencyUnavailable)
	}
	request, binding, err := a.server.financialChargeRecord(workspace, basis.Request)
	if err != nil || request.Digest != basis.RequestDigest || binding.Candidate != basis.Candidate || binding.CandidateDigest != basis.CandidateDigest {
		return bad()
	}
	candidate, err := a.server.chargeProducer.candidates.financialCandidate(workspace, basis.Candidate)
	if err != nil || candidate.Digest != basis.CandidateDigest {
		return bad()
	}
	sub, total, err := documentSubmission(basis, candidate, binding.Subject, key)
	if err != nil || !reflect.DeepEqual(sub, owner.submission) || total != owner.total {
		return bad()
	}
	return basis, owner, nil
}

func (a *financialERP) resultPreparationDiscovery(ctx commands.Context) (map[string]string, error) {
	if err := a.resultPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	if ctx.Target != ctx.Workspace {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	if len(a.owners) > financialDocumentLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	var rows []documentHistoryRow
	for key, owner := range a.owners {
		if key[0] != ctx.Workspace {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if owner.receipt == "" {
			continue
		}
		b, checked, err := a.resultPreparationBasis(key[0], key[1], owner.receipt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, documentHistoryRow{id: key[1], value: map[string]string{"preparation": key[1], "preparation_digest": checked.receipt, "document": b.ID, "document_digest": checked.basis, "connection": checked.submission.Connection, "customer": checked.submission.Customer, "period": checked.submission.Period, "currency": b.Document.Currency, "preparation_actor": checked.actor}})
	}
	fields, err := documentPage(rows, map[string]string{"mode": "synthetic", "workspace": ctx.Workspace, "current_eligibility": "unchecked"}, ctx.Fields["after"], ctx.Fields["snapshot"], ctx.Resolved)
	if err != nil {
		return nil, err
	}
	if err = a.boundedResultFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (a *financialERP) resultContextLocked(ctx commands.Context) (map[string]string, error) {
	if err := a.resultPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	b, owner, err := a.resultPreparationBasis(ctx.Workspace, ctx.Target, ctx.Fields["digest"])
	if err != nil {
		return nil, err
	}
	permission := documentPermissionAdvice(a.resultPermission(ctx.Workspace, ctx.User, gate.ProposeERPResult))
	ready := documentAdviceState{State: "unchecked", Code: permission.Code}
	if permission.State == "allowed" {
		if a.server.held() {
			ready = documentAdviceState{State: "blocked", Code: commands.PrerequisiteFailed}
		} else {
			_, _, _, e := a.resultHistoricalPreparation(ctx.Workspace, financialERPResultPacket{Preparation: ctx.Target, PreparationDigest: owner.receipt, Connection: owner.submission.Connection, Currency: b.Document.Currency, Customer: owner.submission.Customer, Period: owner.submission.Period})
			if e == nil {
				_, e = a.resultNow()
			}
			ready = documentReadinessAdvice(e)
		}
	}
	permissions, err := json.Marshal(map[string]documentAdviceState{"propose": permission})
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	readiness, err := json.Marshal(ready)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	products := map[string]string{}
	for _, m := range b.Mappings {
		if m.Mapping.Kind == "product" {
			products[m.Mapping.Platform] = m.Mapping.External
		}
	}
	if len(b.Lines) == 0 {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	var rows []documentHistoryRow
	for _, line := range owner.submission.Lines {
		product := products[line.Product]
		if product == "" {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		rows = append(rows, documentHistoryRow{id: line.ID, value: map[string]string{"line": line.ID, "expected_product": product, "expected_unit": b.Lines[0].Unit}})
	}
	fields := map[string]string{"mode": "synthetic", "preparation": ctx.Target, "preparation_digest": owner.receipt, "document": b.ID, "document_digest": owner.basis, "connection": owner.submission.Connection, "customer": owner.submission.Customer, "period": owner.submission.Period, "currency": b.Document.Currency, "profile": "synthetic_result_one_to_one_v1", "schema": "hb_erp_result_packet_v1", "permissions": string(permissions), "proposal_readiness": string(readiness), "current_eligibility": "unchecked"}
	fields, err = documentPage(rows, fields, ctx.Fields["after"], ctx.Fields["snapshot"], ctx.Resolved)
	if err != nil {
		return nil, err
	}
	if err = a.boundedResultFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
