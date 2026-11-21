package server

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (a *financialERP) boundedResultFields(fields map[string]string) error {
	limits := a.server.cfg.Commands
	if limits.Fields < 28 || limits.Value < 2000 {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if len(fields) > limits.Fields {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	for _, v := range fields {
		if len(v) > min(4096, limits.Value) {
			return commands.Fail(commands.PrerequisiteFailed)
		}
	}
	return nil
}

func resultReceipt(v financialERPResultView) map[string]string {
	return map[string]string{"mode": "synthetic", "result": v.Basis.ID, "digest": v.Review.Digest, "recorded_state": v.Review.State, "receipt": "historical_decision", "application_state": v.ApplicationState, "application_receipt": v.ApplicationReceipt}
}

func (a *financialERP) resultFields(v financialERPResultView) (map[string]string, error) {
	f := resultReceipt(v)
	f["receipt"], f["current_eligibility"], f["source"] = "historical_evidence", "unchecked", v.Source
	f["source_digest"], f["packet_digest"], f["document_digest"] = v.Basis.SourceDigest, v.Basis.PacketDigest, v.Basis.DocumentDigest
	f["proposed_by"], f["proposal_reason"], f["proposed_at"] = v.Review.Proposer, v.Review.Reason, v.Review.At.Format(time.RFC3339Nano)
	for _, event := range []struct {
		prefix string
		value  financialDocumentDecision
	}{{"confirmation", v.Review.Confirmation}, {"revocation", v.Review.Revocation}} {
		if event.value.Actor != "" {
			f[event.prefix+"_by"], f[event.prefix+"_reason"], f[event.prefix+"_at"] = event.value.Actor, event.value.Reason, event.value.At.Format(time.RFC3339Nano)
		}
	}
	for key, value := range map[string]any{"packet": v.Basis.Packet, "preparation": v.Basis.Preparation, "document": v.Basis.Document, "comparison": v.Basis.Comparison, "total": v.Basis.Total} {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, commands.Fail(commands.PrerequisiteFailed)
		}
		f[key] = string(raw)
	}
	if err := a.boundedResultFields(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (a *financialERP) admitPublicResult(b financialERPResultBasis, source string) error {
	event := financialDocumentDecision{Actor: strings.Repeat("a", 128), Reason: strings.Repeat("r", 2000), At: time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)}
	v := financialERPResultView{Basis: b, Source: source, ApplicationState: "not_started", ApplicationReceipt: strings.Repeat("d", 64), Review: financialDocumentReview{Digest: strings.Repeat("d", 64), Proposer: event.Actor, Reason: event.Reason, At: event.At, State: "confirmed", Confirmation: event, Revocation: event}}
	_, err := a.resultFields(v)
	return err
}

func (a *financialERP) resultApplicationOwner(b financialERPResultBasis, digest string) (financialERPResultApplication, bool, error) {
	app, ok := a.resultApplications[[2]string{b.Workspace, b.ID}]
	external, preparation := [3]string{b.Workspace, b.Packet.Connection, b.Packet.External}, [2]string{b.Workspace, b.Packet.Preparation}
	if !ok {
		if a.resultExternals[external] == b.ID || a.resultPreparations[preparation] == b.ID {
			return app, false, commands.Fail(commands.EvidenceStale)
		}
		return app, false, nil
	}
	copy := app
	copy.receipt = ""
	if !validRequestID(app.actor) || !reflect.DeepEqual(copy, resultIntent(b, app.actor, digest)) || app.receipt != "" && !validPlanHash(app.receipt) || a.resultExternals[external] != b.ID || a.resultPreparations[preparation] != b.ID {
		return app, true, commands.Fail(commands.EvidenceStale)
	}
	return app, true, nil
}

func (a *financialERP) checkResultReceipt(b financialERPResultBasis, app financialERPResultApplication) error {
	snapshot, err := a.ledger.ResultOf(b.Workspace, b.Packet.Preparation)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if !resultSnapshotMatches(snapshot, b, app, true) {
		return commands.Fail(commands.EvidenceStale)
	}
	receipt, err := resultApplicationDigest(app.digest, app.actor, snapshot)
	if err != nil || receipt != app.receipt {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}
