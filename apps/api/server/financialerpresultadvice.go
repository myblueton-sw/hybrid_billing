package server

import (
	"encoding/json"
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (a *financialERP) resultApplicationObservation(b financialERPResultBasis, r financialERPResultRecord, actor string, recover bool) error {
	app, owned, err := a.resultApplicationOwner(b, r.review.Digest)
	if err != nil {
		return err
	}
	if owned != recover || owned && (app.actor != actor || app.receipt != "") {
		return commands.Fail(commands.EvidenceStale)
	}
	if owned && app.predecessor != "" {
		return a.replacementObservation(b, app)
	}
	if !owned && (a.resultExternals[[3]string{b.Workspace, b.Packet.Connection, b.Packet.External}] != "" || a.resultPreparations[[2]string{b.Workspace, b.Packet.Preparation}] != "") {
		_, err := a.replacementIntent(b, actor, r.review.Digest)
		return err
	}
	if !owned {
		if a.resultExternals[[3]string{b.Workspace, b.Packet.Connection, b.Packet.External}] != "" || a.resultPreparations[[2]string{b.Workspace, b.Packet.Preparation}] != "" {
			return commands.Fail(commands.EvidenceStale)
		}
		if len(a.resultApplications) >= financialDocumentLimit || len(a.resultExternals) >= financialDocumentLimit || len(a.resultPreparations) >= financialDocumentLimit {
			return commands.Fail(commands.PrerequisiteFailed)
		}
		app = resultIntent(b, actor, r.review.Digest)
		if err := resultApplicationAdmission(b, app); err != nil {
			return err
		}
	}
	document, lookup := a.ledger.DocumentOf(b.Workspace, b.Packet.Connection, b.Packet.External)
	if lookup != nil && !errors.Is(lookup, erpledger.ErrNotFound) {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if lookup == nil && (!owned || !resultDocumentMatches(document, app, b.Total)) {
		return commands.Fail(commands.EvidenceStale)
	}
	snapshot, err := a.ledger.ResultOf(b.Workspace, b.Packet.Preparation)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if !resultSnapshotMatches(snapshot, b, app, false) || (!owned || lookup != nil) && len(snapshot.Documents) != 0 {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialERP) resultReadiness(ctx commands.Context, action string, permission documentAdviceState, r financialERPResultRecord, b financialERPResultBasis) documentAdviceState {
	if permission.State != "allowed" {
		return documentAdviceState{State: "unchecked", Code: permission.Code}
	}
	if err := a.admitPublicResult(b, r.raw); err != nil {
		return documentReadinessAdvice(err)
	}
	inapplicable := func(code commands.Code) documentAdviceState {
		return documentAdviceState{State: "inapplicable", Code: code}
	}
	if action == "confirm" {
		if r.review.State != "pending" {
			return inapplicable(commands.PrerequisiteFailed)
		}
		if ctx.User == r.review.Proposer {
			return inapplicable(commands.SelfApprovalDenied)
		}
	} else if action == "revoke" {
		if r.review.State == "revoked" {
			return inapplicable(commands.PrerequisiteFailed)
		}
		_, err := a.resultNow()
		return documentReadinessAdvice(err)
	} else {
		app, owned, err := a.resultApplicationOwner(b, r.review.Digest)
		if err != nil {
			return documentReadinessAdvice(err)
		}
		if r.review.State != "confirmed" || action == "apply" && owned || action == "recover" && (!owned || app.receipt != "") {
			return inapplicable(commands.PrerequisiteFailed)
		}
	}
	if a.server.held() {
		return documentAdviceState{State: "blocked", Code: commands.PrerequisiteFailed}
	}
	if action == "confirm" {
		if err := a.resultStanding(ctx.Workspace, r.review.Proposer, gate.ProposeERPResult); err != nil {
			return documentReadinessAdvice(err)
		}
		if _, _, err := a.resultWatermark(b); err != nil {
			return documentReadinessAdvice(err)
		}
		_, err := a.resultNow()
		return documentReadinessAdvice(err)
	}
	if err := a.currentResult(b, r, ctx.User); err != nil {
		return documentReadinessAdvice(err)
	}
	return documentReadinessAdvice(a.resultApplicationObservation(b, r, ctx.User, action == "recover"))
}

func (a *financialERP) resultAdviceLocked(ctx commands.Context) (map[string]string, error) {
	if err := a.resultPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	r, b, err := a.resultRecord(ctx.Workspace, ctx.Target, ctx.Fields["digest"])
	if err != nil {
		return nil, err
	}
	app, owned, err := a.resultApplicationOwner(b, r.review.Digest)
	if err != nil {
		return nil, err
	}
	permissions, readiness := map[string]documentAdviceState{}, map[string]documentAdviceState{}
	for name, action := range map[string]gate.Action{"confirm": gate.ConfirmERPResult, "revoke": gate.RevokeERPResult, "apply": gate.ApplyERPResult} {
		permission := documentPermissionAdvice(a.resultPermission(ctx.Workspace, ctx.User, action))
		permissions[name] = permission
		readiness[name] = a.resultReadiness(ctx, name, permission, r, b)
	}
	readiness["recover"] = a.resultReadiness(ctx, "recover", permissions["apply"], r, b)
	p, err := json.Marshal(permissions)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	ready, err := json.Marshal(readiness)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	view := a.resultView(r, b)
	actor := ""
	if owned {
		actor = app.actor
	}
	fields := map[string]string{"mode": "synthetic", "result": ctx.Target, "digest": r.review.Digest, "revision": ctx.Resolved, "permissions": string(p), "readiness": string(ready), "application_state": view.ApplicationState, "applying_actor": actor, "application_receipt": view.ApplicationReceipt, "current_eligibility": "advisory_only"}
	context, err := a.resultApplicationContext(b)
	if err != nil {
		return nil, err
	}
	fields["application_context"] = context
	if err = a.boundedResultFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
