package server

import (
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

type pricingPolicyCommand struct {
	s    *Server
	kind string
}

func (h pricingPolicyCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func pricingPolicyError(err error) error {
	switch {
	case errors.Is(err, pricingpolicy.ErrInvalid), errors.Is(err, errCandidateInvalid), errors.Is(err, organizations.ErrInvalid), errors.Is(err, contracts.ErrInvalid):
		return commands.Fail(commands.InvalidRequest)
	case errors.Is(err, pricingpolicy.ErrNotFound), errors.Is(err, pricingpolicy.ErrRejected), errors.Is(err, organizations.ErrNotFound), errors.Is(err, contracts.ErrNotFound):
		return commands.Fail(commands.PermissionDenied)
	case errors.Is(err, pricingpolicy.ErrConflict), errors.Is(err, errCandidateConflict), errors.Is(err, organizations.ErrConflict), errors.Is(err, contracts.ErrConflict):
		return commands.Fail(commands.EvidenceStale)
	case errors.Is(err, pricingpolicy.ErrLimit), errors.Is(err, errCandidateLimit), errors.Is(err, contracts.ErrLimit), errors.Is(err, organizations.ErrSuspended), errors.Is(err, contracts.ErrRejected):
		return commands.Fail(commands.PrerequisiteFailed)
	default:
		return commands.Fail(commands.DependencyUnavailable)
	}
}

func pricingPreparation(id string, spec pricingpolicy.Spec, target pricingpolicy.Target) (string, error) {
	return normalizedDigest(struct {
		Version, ID string
		Spec        pricingpolicy.Spec
		Target      pricingpolicy.Target
	}{"synthetic-pricing-preparation-v1", id, spec, target})
}

func pricingTargetFields(target pricingpolicy.Target) map[string]string {
	return map[string]string{
		"contract_revision":  strconv.FormatUint(target.ContractRevision, 10),
		"billing_revision":   strconv.FormatUint(target.BillingRevision, 10),
		"target_fingerprint": target.Fingerprint,
	}
}

func (h pricingPolicyCommand) prepare(ctx commands.Context) (pricingpolicy.Spec, pricingpolicy.Target, string, error) {
	spec, err := parsePricingSpec(ctx.Fields["spec"], ctx.Workspace)
	if err != nil {
		return spec, pricingpolicy.Target{}, "", err
	}
	encoded, err := pricingSpecJSON(spec)
	if err != nil || len(encoded) > h.s.cfg.Commands.Value {
		return spec, pricingpolicy.Target{}, "", pricingpolicy.ErrInvalid
	}
	target, err := (candidatePricingTargets{h.s.chargeProducer}).Current(spec.Scope, spec.From, spec.To)
	if err != nil {
		return spec, target, "", err
	}
	if target.Currency != spec.Policy.Currency || target.Path != spec.Policy.Path {
		return spec, target, "", pricingpolicy.ErrConflict
	}
	digest, err := pricingPreparation(ctx.Target, spec, target)
	return spec, target, digest, err
}

func pricingPolicyReceipt(record pricingpolicy.Record) map[string]string {
	return map[string]string{"mode": "synthetic", "policy": record.Proposal.ID, "digest": record.Digest, "recorded_state": record.State, "receipt": "historical_decision"}
}

func (h pricingPolicyCommand) inspect(record pricingpolicy.Record) (commands.Outcome, error) {
	spec, err := pricingSpecJSON(record.Proposal.Spec)
	if err != nil {
		return commands.Outcome{}, err
	}
	fields := pricingPolicyReceipt(record)
	fields["spec"] = spec
	fields["proposed_by"] = record.Proposal.Proposer
	fields["proposal_reason"] = record.Proposal.Rationale
	fields["proposed_at"] = record.ProposedAt.Format(time.RFC3339Nano)
	for name, value := range pricingTargetFields(record.Proposal.Target) {
		fields[name] = value
	}
	for _, decision := range []struct {
		prefix string
		value  *pricingpolicy.Decision
	}{{"approval", record.Approval}, {"revocation", record.Revocation}} {
		if decision.value != nil {
			fields[decision.prefix+"_by"] = decision.value.Actor
			fields[decision.prefix+"_reason"] = decision.value.Rationale
			fields[decision.prefix+"_at"] = decision.value.At.Format(time.RFC3339Nano)
		}
	}
	fields["current_target"] = "unavailable"
	p := record.Proposal
	current, err := (candidatePricingTargets{h.s.chargeProducer}).Current(p.Spec.Scope, p.Spec.From, p.Spec.To)
	if err == nil {
		fields["current_target"] = "changed"
		if current == p.Target {
			fields["current_target"] = "matched"
		}
	}
	return commands.Outcome{Fields: fields}, nil
}

func (h pricingPolicyCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	r := h.s.chargeProducer.pricingPolicies
	if r == nil || h.s.cfg.Commands.Fields < 19 || h.s.cfg.Commands.Value < 1024 {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	authority := serverPricingPolicyAuthority{s: h.s}
	if h.kind == "prepare" || h.kind == "inspect" {
		if err := authority.inspect(ctx.Workspace); err != nil {
			return commands.Outcome{}, pricingPolicyError(err)
		}
	} else {
		if err := authority.Check(pricingpolicy.Scope{Workspace: ctx.Workspace}, ctx.User, "pricing_policy."+h.kind); err != nil {
			return commands.Outcome{}, pricingPolicyError(err)
		}
	}
	if h.kind == "prepare" || h.kind == "propose" {
		spec, target, digest, err := h.prepare(ctx)
		if err != nil {
			return commands.Outcome{}, pricingPolicyError(err)
		}
		if h.kind == "prepare" {
			fields := pricingTargetFields(target)
			fields["mode"], fields["policy"], fields["preparation_digest"] = "synthetic", ctx.Target, digest
			return commands.Outcome{Fields: fields}, nil
		}
		for name, want := range pricingTargetFields(target) {
			if ctx.Fields[name] != want {
				return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
			}
		}
		if ctx.Fields["preparation_digest"] != digest {
			return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
		}
		record, err := r.Propose(ctx.Target, ctx.User, ctx.Rationale, spec)
		if err != nil {
			return commands.Outcome{}, pricingPolicyError(err)
		}
		return commands.Outcome{Fields: pricingPolicyReceipt(record), BusinessState: record.State, ResultRevision: h.s.Next()}, nil
	}
	record, err := r.Record(ctx.Workspace, ctx.Target)
	if err != nil {
		return commands.Outcome{}, pricingPolicyError(err)
	}
	if h.kind == "inspect" {
		out, err := h.inspect(record)
		if err != nil {
			return commands.Outcome{}, pricingPolicyError(err)
		}
		return out, nil
	}
	switch h.kind {
	case "approve":
		if record.Proposal.Proposer == ctx.User {
			return commands.Outcome{}, commands.Fail(commands.SelfApprovalDenied)
		}
		record, err = r.Approve(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User, ctx.Rationale)
	case "revoke":
		record, err = r.Revoke(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User, ctx.Rationale)
	default:
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err != nil {
		return commands.Outcome{}, pricingPolicyError(err)
	}
	return commands.Outcome{Fields: pricingPolicyReceipt(record), BusinessState: record.State, ResultRevision: h.s.Next()}, nil
}

func (s *Server) pricingPolicyDefinitions() []commands.Definition {
	var out []commands.Definition
	for _, kind := range []string{"prepare", "propose", "approve", "revoke", "inspect"} {
		mutation := kind == "propose" || kind == "approve" || kind == "revoke"
		permission := commands.Permission{Action: pricingPolicyAction("pricing_policy." + kind), Kind: pricingauthority.ResourceKind}
		if !mutation {
			permission.Action, permission.Fields = gate.ViewCost, []string{pricingauthority.EvidenceField}
		}
		var fields []commands.Field
		switch kind {
		case "prepare", "propose":
			fields = append(fields, commands.Field{Name: "spec", Type: commands.Text, Required: true})
			if kind == "propose" {
				for _, name := range []string{"contract_revision", "billing_revision", "target_fingerprint", "preparation_digest"} {
					fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
				}
			}
		case "approve", "revoke":
			fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
		}
		refusals := []commands.Code{commands.EvidenceStale}
		if kind == "approve" {
			refusals = append(refusals, commands.SelfApprovalDenied)
		}
		out = append(out, commands.Definition{ID: "pricing_policy." + kind, Permission: permission, Mutating: mutation, Rationale: mutation, Fields: fields, Refusals: refusals, Handler: pricingPolicyCommand{s: s, kind: kind}})
	}
	return out
}
