package server

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func resultReviewDigest(basis, actor, reason string) (string, error) {
	return normalizedDigest(struct{ Schema, Basis, Actor, Reason string }{"hb_erp_result_proposal_v1", basis, actor, reason})
}
func (a *financialERP) resultNow() (time.Time, error) {
	at := a.server.clock().UTC().Round(0)
	if at.IsZero() || at.Year() < 1970 || at.Year() > 9999 || at.Before(a.resultTime) {
		return time.Time{}, commands.Fail(commands.DependencyUnavailable)
	}
	return at, nil
}
func resultReviewValid(r financialDocumentReview) bool {
	if !validPlanHash(r.Digest) || !validRequestID(r.Proposer) || !documentReason(r.Reason) || r.At.IsZero() {
		return false
	}
	for _, e := range []financialDocumentDecision{r.Confirmation, r.Revocation} {
		if e.Actor == "" {
			if e.Reason != "" || !e.At.IsZero() {
				return false
			}
			continue
		}
		if !validRequestID(e.Actor) || !documentReason(e.Reason) || e.At.IsZero() || e.At.Before(r.At) {
			return false
		}
	}
	if r.Confirmation.Actor == r.Proposer || r.Confirmation.Actor != "" && r.Revocation.Actor != "" && r.Revocation.At.Before(r.Confirmation.At) {
		return false
	}
	return r.State == "pending" && r.Confirmation.Actor == "" && r.Revocation.Actor == "" || r.State == "confirmed" && r.Confirmation.Actor != "" && r.Revocation.Actor == "" || r.State == "revoked" && r.Revocation.Actor != ""
}
func (a *financialERP) resultRecord(workspace, id, digest string) (financialERPResultRecord, financialERPResultBasis, error) {
	bad := func() (financialERPResultRecord, financialERPResultBasis, error) {
		return financialERPResultRecord{}, financialERPResultBasis{}, commands.Fail(commands.EvidenceStale)
	}
	r, ok := a.results[[2]string{workspace, id}]
	if !ok || r.review.Digest != digest || !resultReviewValid(r.review) || len(r.basis) > financialDocumentBytes {
		return bad()
	}
	var b financialERPResultBasis
	if json.Unmarshal(r.basis, &b) != nil || b.Workspace != workspace || b.ID != id {
		return bad()
	}
	p, err := parseResultPacket(r.raw)
	if err != nil || !reflect.DeepEqual(p, b.Packet) || resultSourceDigest(r.raw) != b.SourceDigest {
		return bad()
	}
	packetDigest, err := normalizedDigest(p)
	if err != nil || packetDigest != b.PacketDigest {
		return bad()
	}
	actual, err := normalizedDigest(b)
	if err != nil || actual != r.digest {
		return bad()
	}
	reviewDigest, err := resultReviewDigest(actual, r.review.Proposer, r.review.Reason)
	if err != nil || reviewDigest != digest {
		return bad()
	}
	return r, b, nil
}
func (a *financialERP) resultView(r financialERPResultRecord, b financialERPResultBasis) financialERPResultView {
	state, receipt := "not_started", ""
	if app, ok := a.resultApplications[[2]string{b.Workspace, b.ID}]; ok {
		state = "unknown"
		receipt = app.receipt
		if receipt != "" {
			state = "recorded"
		}
	}
	return financialERPResultView{Basis: b, Review: r.review, CurrentEligibility: "unchecked", ApplicationState: state, ApplicationReceipt: receipt, Source: r.raw}
}
func (a *financialERP) proposeResultLocked(workspace, id, actor, reason, raw string) (financialERPResultView, error) {
	fail := func(err error) (financialERPResultView, error) { return financialERPResultView{}, err }
	if err := a.resultPermission(workspace, actor, gate.ProposeERPResult); err != nil {
		return fail(err)
	}
	if !validRequestID(id) || !documentReason(reason) {
		return fail(commands.Fail(commands.InvalidRequest))
	}
	if a.server.held() {
		return fail(commands.Fail(commands.PrerequisiteFailed))
	}
	b, err := a.buildResultBasis(workspace, id, raw)
	if err != nil {
		return fail(err)
	}
	body, err := resultBasisBytes(b)
	if err != nil {
		return fail(err)
	}
	basisDigest, err := normalizedDigest(b)
	if err != nil {
		return fail(err)
	}
	digest, err := resultReviewDigest(basisDigest, actor, reason)
	if err != nil {
		return fail(err)
	}
	key := [2]string{workspace, id}
	if old, ok := a.results[key]; ok {
		if old.review.Digest != digest {
			return fail(commands.Fail(commands.EvidenceStale))
		}
		record, basis, err := a.resultRecord(workspace, id, digest)
		if err != nil {
			return fail(err)
		}
		return a.resultView(record, basis), nil
	}
	if len(a.results) >= financialDocumentLimit {
		return fail(commands.Fail(commands.PrerequisiteFailed))
	}
	at, err := a.resultNow()
	if err != nil {
		return fail(err)
	}
	record := financialERPResultRecord{raw: raw, basis: body, digest: basisDigest, review: financialDocumentReview{Digest: digest, Proposer: actor, Reason: reason, State: "pending", At: at}}
	a.results[key] = record
	a.resultTime = at
	return a.resultView(record, b), nil
}
func sameResultExternal(a, b financialERPResultBasis) bool {
	return a.Workspace == b.Workspace && a.Packet.Connection == b.Packet.Connection && a.Packet.External == b.Packet.External
}
func (a *financialERP) resultWatermark(b financialERPResultBasis) (int, string, error) {
	version, digest := 0, ""
	for key, r := range a.results {
		if key[0] != b.Workspace || r.review.Confirmation.Actor == "" {
			continue
		}
		_, other, err := a.resultRecord(key[0], key[1], r.review.Digest)
		if err != nil {
			return 0, "", err
		}
		if !sameResultExternal(b, other) {
			continue
		}
		if other.Packet.Revision == b.Packet.Revision && other.ID != b.ID {
			return 0, "", commands.Fail(commands.EvidenceStale)
		}
		if other.Packet.Revision > version {
			version = other.Packet.Revision
			digest = r.review.Digest
		}
	}
	return version, digest, nil
}
func (a *financialERP) decideResultLocked(workspace, id, digest, actor, reason string, revoke bool) (financialERPResultView, error) {
	fail := func(err error) (financialERPResultView, error) { return financialERPResultView{}, err }
	action := gate.ConfirmERPResult
	if revoke {
		action = gate.RevokeERPResult
	}
	if err := a.resultPermission(workspace, actor, action); err != nil {
		return fail(err)
	}
	if !documentReason(reason) {
		return fail(commands.Fail(commands.InvalidRequest))
	}
	r, b, err := a.resultRecord(workspace, id, digest)
	if err != nil {
		return fail(err)
	}
	state, previous := "confirmed", r.review.Confirmation
	if revoke {
		state = "revoked"
		previous = r.review.Revocation
	} else {
		if actor == r.review.Proposer {
			return fail(commands.Fail(commands.SelfApprovalDenied))
		}
		if a.server.held() || r.review.State == "revoked" {
			return fail(commands.Fail(commands.PrerequisiteFailed))
		}
		if err := a.resultStanding(workspace, r.review.Proposer, gate.ProposeERPResult); err != nil {
			return fail(err)
		}
		if _, _, err = a.resultWatermark(b); err != nil {
			return fail(err)
		}
	}
	if r.review.State == state {
		if previous.Actor != actor || previous.Reason != reason {
			return fail(commands.Fail(commands.EvidenceStale))
		}
		return a.resultView(r, b), nil
	}
	at, err := a.resultNow()
	if err != nil {
		return fail(err)
	}
	event := financialDocumentDecision{Actor: actor, Reason: reason, At: at}
	if revoke {
		r.review.Revocation = event
	} else {
		r.review.Confirmation = event
	}
	r.review.State = state
	a.results[[2]string{workspace, id}] = r
	a.resultTime = at
	return a.resultView(r, b), nil
}
func (a *financialERP) inspectResultLocked(workspace, id, digest, actor string) (financialERPResultView, error) {
	if err := a.resultPermission(workspace, actor, gate.ViewCost); err != nil {
		return financialERPResultView{}, err
	}
	r, b, err := a.resultRecord(workspace, id, digest)
	if err != nil {
		return financialERPResultView{}, err
	}
	if _, _, err := a.resultApplicationOwner(b, r.review.Digest); err != nil {
		return financialERPResultView{}, err
	}
	return a.resultView(r, b), nil
}
