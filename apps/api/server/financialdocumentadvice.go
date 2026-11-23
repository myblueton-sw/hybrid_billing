package server

import (
	"encoding/json"
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type documentAdviceState struct {
	State string        `json:"state"`
	Code  commands.Code `json:"code"`
}

func documentAdviceCode(err error) commands.Code {
	if err == nil {
		return ""
	}
	var failure *commands.Failure
	if errors.As(err, &failure) {
		switch failure.Code {
		case commands.PermissionDenied, commands.Unauthenticated, commands.AuthorityUnavailable, commands.DependencyUnavailable, commands.UnsupportedCapability, commands.PrerequisiteFailed, commands.EvidenceStale, commands.SelfApprovalDenied:
			return failure.Code
		}
		return commands.DependencyUnavailable
	}
	normalized := financialChargeEvidenceError(err)
	if errors.As(normalized, &failure) && failure.Code == commands.EvidenceStale {
		return commands.EvidenceStale
	}
	return commands.DependencyUnavailable
}

func documentPermissionAdvice(err error) documentAdviceState {
	code := documentAdviceCode(err)
	switch code {
	case "":
		return documentAdviceState{State: "allowed"}
	case commands.PermissionDenied, commands.Unauthenticated:
		return documentAdviceState{State: "denied", Code: code}
	case commands.AuthorityUnavailable:
		return documentAdviceState{State: "unavailable", Code: code}
	default:
		return documentAdviceState{State: "unavailable", Code: commands.DependencyUnavailable}
	}
}

func documentReadinessAdvice(err error) documentAdviceState {
	code := documentAdviceCode(err)
	switch code {
	case "":
		return documentAdviceState{State: "ready"}
	case commands.AuthorityUnavailable, commands.DependencyUnavailable, commands.UnsupportedCapability:
		return documentAdviceState{State: "unavailable", Code: code}
	default:
		return documentAdviceState{State: "blocked", Code: code}
	}
}

func (a *financialERP) documentReadiness(ctx commands.Context, action string, permission documentAdviceState, record financialDocumentRecord, b financialDocumentBasis) documentAdviceState {
	if permission.State != "allowed" {
		return documentAdviceState{State: "unchecked", Code: permission.Code}
	}
	state := record.review.State
	if action == "confirm" && state != "pending" || action == "prepare" && state != "confirmed" || action == "revoke" && state == "revoked" {
		return documentAdviceState{State: "inapplicable", Code: commands.PrerequisiteFailed}
	}
	if action == "confirm" && record.review.Proposer == ctx.User {
		return documentAdviceState{State: "inapplicable", Code: commands.SelfApprovalDenied}
	}
	if action == "revoke" {
		_, err := a.documentNow()
		return documentReadinessAdvice(err)
	}
	if a.server.held() {
		return documentAdviceState{State: "blocked", Code: commands.PrerequisiteFailed}
	}
	if action == "confirm" {
		if err := a.documentProposer(ctx.Workspace, record.review.Proposer); err != nil {
			return documentReadinessAdvice(err)
		}
	}
	_, err := a.currentDocumentLocked(b, ctx.User)
	if err == nil {
		_, err = a.documentNow()
	}
	return documentReadinessAdvice(err)
}

func (a *financialERP) documentAdviceLocked(ctx commands.Context) (map[string]string, error) {
	if err := a.documentPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	record, b, err := a.governedDocument(ctx.Workspace, ctx.Target, ctx.Fields["digest"])
	if err != nil {
		return nil, err
	}
	permissions := map[string]documentAdviceState{}
	readiness := map[string]documentAdviceState{}
	for name, action := range map[string]gate.Action{"propose": gate.ProposeDocumentEvidence, "confirm": gate.ConfirmDocumentEvidence, "revoke": gate.RevokeDocumentEvidence, "prepare": gate.PrepareERPRecord} {
		permissions[name] = documentPermissionAdvice(a.documentPermission(ctx.Workspace, ctx.User, action))
		if name != "propose" {
			readiness[name] = a.documentReadiness(ctx, name, permissions[name], record, b)
		}
	}
	p, err := json.Marshal(permissions)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	r, err := json.Marshal(readiness)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	fields := map[string]string{"mode": "synthetic", "document": ctx.Target, "digest": record.review.Digest, "current_eligibility": "advisory", "permissions": string(p), "readiness": string(r), "preparation_scope": "packet_only_key_capacity_and_source_coverage_unchecked"}
	if err = a.boundedDocumentFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
