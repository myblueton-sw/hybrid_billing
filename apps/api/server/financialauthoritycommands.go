package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type financialAuthorityCommand struct {
	s     *Server
	kind  string
	stage financialPolicyStage
}

func (h financialAuthorityCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if _, err := h.s.financialAuthorityDomain(h.stage); err != nil {
		return commands.Target{}, commands.Unavailable
	}
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func financialAuthorityReceipt(record financialAuthorityRecord) map[string]string {
	p, b := record.proposal, record.binding
	a := p.Proposal.Authority
	fields := map[string]string{"mode": "synthetic", "receipt": "historical_decision", "proposal": p.Proposal.ID, "digest": p.Digest, "recorded_state": p.State, "authority": a.ID, "stage": a.Stage, "recipient": a.Approver, "currency": a.Currency, "limit": a.Limit, "from": a.From, "to": a.To, "proposed_by": p.Proposer, "proposal_reason": b.ProposalReason, "proposed_at": p.Written.Format(time.RFC3339Nano), "module_amount_authority": financialAmountStatus(record, record.asOf)}
	if record.authority != nil {
		fields["authority_as_of"] = record.asOf.Format(time.RFC3339Nano)
		fields["confirmed_by"], fields["confirmation_reason"], fields["confirmed_at"] = p.Confirmer, b.ConfirmationReason, p.Confirmed.Format(time.RFC3339Nano)
		if !record.authority.Revoked.IsZero() {
			fields["revoked_by"], fields["revocation_reason"], fields["revoked_at"] = record.authority.RevokedBy, b.RevocationReason, record.authority.Revoked.Format(time.RFC3339Nano)
		}
	}
	return fields
}

func (h financialAuthorityCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	mutation := h.kind != "inspect"
	if err := h.s.financialAuthorityBoundsForStage(ctx, mutation, h.stage); err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	if err := h.s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	if err := h.s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	var record financialAuthorityRecord
	var err error
	switch h.kind {
	case "propose":
		record, err = h.s.proposeFinancialAuthorityForStage(ctx, h.stage)
	case "confirm":
		record, err = h.s.confirmFinancialAuthorityForStage(ctx, h.stage)
	case "revoke":
		record, err = h.s.revokeFinancialAuthorityForStage(ctx, h.stage)
	case "inspect":
		record, err = h.s.financialAuthorityRecordForStage(ctx.Workspace, ctx.Target, h.stage)
	default:
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	out := commands.Outcome{Fields: financialAuthorityReceipt(record)}
	if mutation {
		out.BusinessState, out.ResultRevision = record.proposal.State, h.s.Next()
		if h.kind == "revoke" {
			out.BusinessState = "revoked"
		}
	}
	return out, nil
}

func (s *Server) financialAuthorityDefinitions() []commands.Definition {
	var out []commands.Definition
	for _, stage := range []financialPolicyStage{chargePolicyStage, erpFinalPolicyStage} {
		domain, _ := s.financialAuthorityDomain(stage)
		for _, kind := range []string{"propose", "confirm", "inspect", "revoke"} {
			var fields []commands.Field
			switch kind {
			case "propose":
				for _, name := range []string{"recipient", "currency", "limit", "from", "to"} {
					fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
				}
			case "confirm", "revoke":
				fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
			}
			mutation := kind != "inspect"
			out = append(out, commands.Definition{ID: domain.prefix + "." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Mutating: mutation, Rationale: mutation, Fields: fields, Refusals: []commands.Code{commands.EvidenceStale, commands.SelfApprovalDenied}, Handler: financialAuthorityCommand{s: s, kind: kind, stage: stage}})
		}
	}
	return out
}
