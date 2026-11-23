package server

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type financialPolicyCommand struct {
	s     *Server
	kind  string
	stage financialPolicyStage
}

func (h financialPolicyCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	domain, err := h.s.financialPolicyDomain(h.stage)
	if err != nil {
		return commands.Target{}, commands.Unavailable
	}
	if workspace != h.s.bootWorkspace || !validRequestID(target) || h.kind == "current" && target != domain.stage {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func financialPolicyError(err error) error {
	var failure *commands.Failure
	if errors.As(err, &failure) {
		return err
	}
	var refusal approvals.Refusal
	if errors.As(err, &refusal) && refusal.Reason == approvals.ReasonStaleProposal {
		return commands.Fail(commands.EvidenceStale)
	}
	switch {
	case errors.Is(err, approvals.ErrInvalid):
		return commands.Fail(commands.InvalidRequest)
	case errors.Is(err, approvals.ErrConflict):
		return commands.Fail(commands.EvidenceStale)
	case errors.Is(err, approvals.ErrNotFound), errors.Is(err, approvals.ErrRejected), errors.Is(err, inputauthority.ErrDenied):
		return commands.Fail(commands.PermissionDenied)
	case errors.Is(err, approvals.ErrLimit):
		return commands.Fail(commands.PrerequisiteFailed)
	default:
		return commands.Fail(commands.DependencyUnavailable)
	}
}

func financialPolicyFields(p approvals.Policy) map[string]string {
	return map[string]string{"mode": "synthetic", "stage": p.Stage, "policy_revision": strconv.Itoa(p.Revision), "approvals": strconv.Itoa(p.Approvals), "validity_hours": strconv.FormatInt(int64(p.Validity/time.Hour), 10), "required": strings.Join(p.Required, ",")}
}

func financialPolicyReceipt(record approvals.ProposalView, binding financialPolicyBinding) map[string]string {
	fields := financialPolicyFields(record.Proposal.Policy)
	fields["proposal"], fields["digest"], fields["recorded_state"] = record.Proposal.ID, record.Digest, record.State
	fields["receipt"] = "historical_decision"
	fields["proposed_by"], fields["proposal_reason"], fields["proposed_at"] = record.Proposer, binding.ProposalReason, record.Written.Format(time.RFC3339Nano)
	if record.State == approvals.StateConfirmed {
		fields["confirmed_by"], fields["confirmation_reason"], fields["confirmed_at"] = record.Confirmer, binding.ConfirmationReason, record.Confirmed.Format(time.RFC3339Nano)
	}
	return fields
}

func (s *Server) financialPolicyBoundsForStage(ctx commands.Context, mutation bool, stage financialPolicyStage) error {
	domain, err := s.financialPolicyDomain(stage)
	if err != nil {
		return err
	}
	if s.cfg.Seed != SeedSample || s.approvals == nil || domain.bindings == nil || s.cfg.Commands.Fields < 16 || s.cfg.Commands.Value < 128 || s.cfg.Commands.Value < s.cfg.Commands.Rationale {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if mutation && (strings.TrimSpace(ctx.Rationale) == "" || len(ctx.Rationale) > s.cfg.Commands.Rationale) {
		return approvals.ErrInvalid
	}
	return nil
}

func (h financialPolicyCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	domain, stageErr := h.s.financialPolicyDomain(h.stage)
	if stageErr != nil {
		return commands.Outcome{}, stageErr
	}
	mutation := h.kind == "propose" || h.kind == "confirm"
	if err := h.s.financialPolicyBoundsForStage(ctx, mutation, h.stage); err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	if err := h.s.governorCaller(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	if err := h.s.financialGovernor(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	if h.kind == "current" {
		if ctx.Target != domain.stage {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
		}
		policy, err := h.s.approvals.PolicyOf(ctx.Workspace, domain.stage)
		if err != nil {
			return commands.Outcome{}, financialPolicyError(err)
		}
		fields := financialPolicyFields(policy)
		fields["receipt"] = "current_policy"
		return commands.Outcome{Fields: fields}, nil
	}
	var record approvals.ProposalView
	var binding financialPolicyBinding
	var err error
	switch h.kind {
	case "propose":
		record, binding, err = h.s.proposeFinancialPolicyForStage(ctx, h.stage)
	case "confirm":
		record, binding, err = h.s.confirmFinancialPolicyForStage(ctx, h.stage)
	case "inspect":
		record, binding, err = h.s.financialPolicyRecordForStage(ctx.Workspace, ctx.Target, h.stage)
	default:
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err != nil {
		return commands.Outcome{}, financialPolicyError(err)
	}
	out := commands.Outcome{Fields: financialPolicyReceipt(record, binding)}
	if mutation {
		out.BusinessState, out.ResultRevision = record.State, h.s.Next()
	}
	return out, nil
}

func (s *Server) financialPolicyDefinitions() []commands.Definition {
	var definitions []commands.Definition
	for _, stage := range []financialPolicyStage{chargePolicyStage, erpFinalPolicyStage} {
		domain, _ := s.financialPolicyDomain(stage)
		for _, kind := range []string{"propose", "confirm", "inspect", "current"} {
			mutation := kind == "propose" || kind == "confirm"
			var fields []commands.Field
			if kind == "propose" {
				for _, name := range []string{"policy_revision", "approvals", "validity_hours", "required"} {
					fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
				}
			}
			if kind == "confirm" {
				fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
			}
			refusals := []commands.Code{commands.EvidenceStale}
			if kind == "confirm" {
				refusals = append(refusals, commands.SelfApprovalDenied)
			}
			definitions = append(definitions, commands.Definition{ID: domain.prefix + "." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Mutating: mutation, Rationale: mutation, Fields: fields, Refusals: refusals, Handler: financialPolicyCommand{s: s, kind: kind, stage: stage}})
		}
	}
	return definitions
}
