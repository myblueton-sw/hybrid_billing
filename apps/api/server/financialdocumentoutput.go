package server

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (a *financialERP) boundedDocumentFields(fields map[string]string) error {
	limits := a.server.cfg.Commands
	if limits.Fields < 17 || limits.Value < 2000 {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if len(fields) > limits.Fields {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	for _, value := range fields {
		if len(value) > min(4096, limits.Value) {
			return commands.Fail(commands.PrerequisiteFailed)
		}
	}
	return nil
}

func documentReceipt(v financialDocumentView) map[string]string {
	return map[string]string{"mode": "synthetic", "document": v.Basis.ID, "digest": v.Review.Digest, "recorded_state": v.Review.State, "receipt": "historical_decision"}
}

func (a *financialERP) documentFields(v financialDocumentView) (map[string]string, error) {
	raw, err := documentBytes(v.Basis)
	if err != nil {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	f := documentReceipt(v)
	f["receipt"], f["basis"], f["basis_digest"], f["current_eligibility"] = "historical_evidence", string(raw), v.BasisDigest, "unchecked"
	f["proposed_by"], f["proposal_reason"], f["proposed_at"] = v.Review.Proposer, v.Review.Reason, v.Review.At.Format(time.RFC3339Nano)
	for _, event := range []struct {
		prefix string
		value  financialDocumentDecision
	}{{"confirmation", v.Review.Confirmation}, {"revocation", v.Review.Revocation}} {
		if event.value.Actor != "" {
			f[event.prefix+"_by"], f[event.prefix+"_reason"], f[event.prefix+"_at"] = event.value.Actor, event.value.Reason, event.value.At.Format(time.RFC3339Nano)
		}
	}
	return f, a.boundedDocumentFields(f)
}

func (a *financialERP) admitPublicDocument(b financialDocumentBasis) error {
	event := financialDocumentDecision{Actor: strings.Repeat("a", 128), Reason: strings.Repeat("r", 2000), At: time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)}
	v := financialDocumentView{Basis: b, BasisDigest: strings.Repeat("d", 64), Review: financialDocumentReview{Digest: strings.Repeat("d", 64), Proposer: event.Actor, Reason: event.Reason, At: event.At, State: "confirmed", Confirmation: event, Revocation: event}}
	_, err := a.documentFields(v)
	return err
}

func (a *financialERP) preparationFields(key string, owner financialERPOwnership, observation, observed string) (map[string]string, error) {
	f := map[string]string{"mode": "synthetic", "preparation": key, "document_digest": owner.basis, "recorded_receipt": owner.receipt, "observed_digest": observed, "ledger_observation": observation, "current_eligibility": "unchecked", "actor": owner.actor}
	for name, value := range map[string]any{"submission": owner.submission, "total": owner.total, "mappings": owner.mappings} {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, commands.Fail(commands.PrerequisiteFailed)
		}
		f[name] = string(raw)
	}
	return f, a.boundedDocumentFields(f)
}

func (a *financialERP) admitPublicPreparation(key string, owner financialERPOwnership) error {
	owner.receipt = strings.Repeat("d", 64)
	_, err := a.preparationFields(key, owner, "matching_prepared", owner.receipt)
	return err
}

func (a *financialERP) inspectPreparationLocked(workspace, key, actor string) (map[string]string, error) {
	if err := a.documentPermission(workspace, actor, gate.ViewCost); err != nil {
		return nil, err
	}
	owner, ok := a.owners[[2]string{workspace, key}]
	if !ok {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	actual, err := a.ledger.SubmissionOf(workspace, key)
	observation, digest := "unavailable", ""
	if errors.Is(err, erpledger.ErrNotFound) {
		observation = "missing"
	} else if err == nil {
		observation = "conflict"
		if matchingPreparation(owner, actual) {
			observation, digest = "matching_prepared", actual.Digest
		}
	}
	return a.preparationFields(key, owner, observation, digest)
}
