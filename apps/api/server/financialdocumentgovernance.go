package server

import (
	"bytes"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type financialDocumentDecision struct {
	Actor, Reason string
	At            time.Time
}
type financialDocumentReview struct {
	Digest, Proposer, Reason, State string
	At                              time.Time
	Confirmation, Revocation        financialDocumentDecision
}
type financialDocumentView struct {
	Basis              financialDocumentBasis
	BasisDigest        string
	Review             financialDocumentReview
	CurrentEligibility string
}

func documentView(record financialDocumentRecord, b financialDocumentBasis) financialDocumentView {
	return financialDocumentView{Basis: b, BasisDigest: record.digest, Review: record.review, CurrentEligibility: "unchecked"}
}

func documentReason(reason string) bool {
	return utf8.ValidString(reason) && strings.TrimSpace(reason) != "" && len(reason) <= 2000
}
func documentReviewDigest(basis, actor, reason string) (string, error) {
	return normalizedDigest(struct{ Schema, Basis, Actor, Reason string }{"hb_document_proposal_v1", basis, actor, reason})
}
func (a *financialERP) documentNow() (time.Time, error) {
	at := a.server.clock().UTC().Round(0)
	if at.IsZero() || at.Year() < 1970 || at.Year() > 9999 || at.Before(a.documentTime) {
		return time.Time{}, commands.Fail(commands.DependencyUnavailable)
	}
	return at, nil
}
func (a *financialERP) governedDocument(workspace, id, digest string) (financialDocumentRecord, financialDocumentBasis, error) {
	record, ok := a.documents[[2]string{workspace, id}]
	if !ok || digest == "" || record.review.Digest != digest {
		return financialDocumentRecord{}, financialDocumentBasis{}, commands.Fail(commands.EvidenceStale)
	}
	b, err := a.document(workspace, id, record.digest)
	if err != nil {
		return record, b, commands.Fail(commands.EvidenceStale)
	}
	actual, err := documentReviewDigest(record.digest, record.review.Proposer, record.review.Reason)
	if err != nil || actual != digest || !validRequestID(record.review.Proposer) || !documentReason(record.review.Reason) || record.review.At.IsZero() {
		return record, b, commands.Fail(commands.EvidenceStale)
	}
	r := record.review
	for _, event := range []financialDocumentDecision{r.Confirmation, r.Revocation} {
		if event.Actor == "" {
			if event.Reason != "" || !event.At.IsZero() {
				return record, b, commands.Fail(commands.EvidenceStale)
			}
			continue
		}
		if !validRequestID(event.Actor) || !documentReason(event.Reason) || event.At.IsZero() || event.At.Before(r.At) {
			return record, b, commands.Fail(commands.EvidenceStale)
		}
	}
	if r.Confirmation.Actor == r.Proposer || r.Revocation.Actor != "" && r.Confirmation.Actor != "" && r.Revocation.At.Before(r.Confirmation.At) {
		return record, b, commands.Fail(commands.EvidenceStale)
	}
	valid := r.State == "pending" && r.Confirmation.Actor == "" && r.Revocation.Actor == "" || r.State == "confirmed" && r.Confirmation.Actor != "" && r.Revocation.Actor == "" || r.State == "revoked" && r.Revocation.Actor != ""
	if !valid {
		return record, b, commands.Fail(commands.EvidenceStale)
	}
	return record, b, nil
}
func (a *financialERP) currentDocumentLocked(b financialDocumentBasis, actor string) (financialDocumentRecord, error) {
	if a.server.held() {
		return financialDocumentRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if _, err := a.server.currentFinancialChargeLocked(b.Workspace, b.Request, b.RequestDigest, actor); err != nil {
		return financialDocumentRecord{}, err
	}
	return a.validateDocumentLocked(b)
}
func (a *financialERP) proposeDocumentLocked(b financialDocumentBasis, actor, reason string) (financialDocumentView, error) {
	fail := func(err error) (financialDocumentView, error) { return financialDocumentView{}, err }
	if err := a.documentPermission(b.Workspace, actor, gate.ProposeDocumentEvidence); err != nil {
		return fail(err)
	}
	if !documentReason(reason) {
		return fail(commands.Fail(commands.InvalidRequest))
	}
	record, err := a.currentDocumentLocked(b, actor)
	if err != nil {
		return fail(err)
	}
	digest, err := documentReviewDigest(record.digest, actor, reason)
	if err != nil {
		return fail(err)
	}
	key := [2]string{b.Workspace, b.ID}
	if old, ok := a.documents[key]; ok {
		if !bytes.Equal(old.raw, record.raw) || old.review.Digest != digest {
			return fail(commands.Fail(commands.EvidenceStale))
		}
		_, basis, err := a.governedDocument(b.Workspace, b.ID, digest)
		return documentView(old, basis), err
	}
	if len(a.documents) >= financialDocumentLimit {
		return fail(commands.Fail(commands.PrerequisiteFailed))
	}
	at, err := a.documentNow()
	if err != nil {
		return fail(err)
	}
	record.review = financialDocumentReview{Digest: digest, Proposer: actor, Reason: reason, State: "pending", At: at}
	a.documents[key] = record
	a.documentTime = at
	basis, err := a.document(b.Workspace, b.ID, record.digest)
	return documentView(record, basis), err
}
func (a *financialERP) decideDocumentLocked(workspace, id, digest, actor, reason string, revoke bool) (financialDocumentView, error) {
	fail := func(err error) (financialDocumentView, error) { return financialDocumentView{}, err }
	action := gate.ConfirmDocumentEvidence
	if revoke {
		action = gate.RevokeDocumentEvidence
	}
	if err := a.documentPermission(workspace, actor, action); err != nil {
		return fail(err)
	}
	if !documentReason(reason) {
		return fail(commands.Fail(commands.InvalidRequest))
	}
	record, b, err := a.governedDocument(workspace, id, digest)
	if err != nil {
		return fail(err)
	}
	if !revoke {
		if record.review.Proposer == actor {
			return fail(commands.Fail(commands.SelfApprovalDenied))
		}
		if err = a.documentProposer(workspace, record.review.Proposer); err != nil {
			return fail(err)
		}
		if _, err = a.currentDocumentLocked(b, actor); err != nil {
			return fail(err)
		}
		if record.review.State == "revoked" {
			return fail(commands.Fail(commands.PrerequisiteFailed))
		}
	}
	state, previous := "confirmed", record.review.Confirmation
	if revoke {
		state, previous = "revoked", record.review.Revocation
	}
	if record.review.State == state {
		if previous.Actor != actor || previous.Reason != reason {
			return fail(commands.Fail(commands.EvidenceStale))
		}
		return documentView(record, b), nil
	}
	at, err := a.documentNow()
	if err != nil {
		return fail(err)
	}
	decision := financialDocumentDecision{Actor: actor, Reason: reason, At: at}
	if revoke {
		record.review.Revocation = decision
	} else {
		record.review.Confirmation = decision
	}
	record.review.State = state
	a.documents[[2]string{workspace, id}] = record
	a.documentTime = at
	return documentView(record, b), nil
}
func (a *financialERP) inspectDocumentLocked(workspace, id, digest, actor string) (financialDocumentView, error) {
	if err := a.documentPermission(workspace, actor, gate.ViewCost); err != nil {
		return financialDocumentView{}, err
	}
	record, b, err := a.governedDocument(workspace, id, digest)
	return documentView(record, b), err
}
