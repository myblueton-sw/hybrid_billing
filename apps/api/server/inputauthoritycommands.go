package server

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type inputAuthorityCommand struct {
	s      *Server
	kind   string
	domain grantAuthorityDomain
}

func (h inputAuthorityCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func inputAuthorityError(err error) error {
	switch {
	case errors.Is(err, inputauthority.ErrUnknown):
		return err
	case errors.Is(err, inputauthority.ErrInvalid):
		return commands.Fail(commands.InvalidRequest)
	case errors.Is(err, inputauthority.ErrNotFound), errors.Is(err, inputauthority.ErrDenied):
		return commands.Fail(commands.PermissionDenied)
	case errors.Is(err, inputauthority.ErrConflict):
		return commands.Fail(commands.InvalidStage)
	case errors.Is(err, inputauthority.ErrLimit):
		return commands.Fail(commands.PrerequisiteFailed)
	case errors.Is(err, gate.ErrUnavailable), errors.Is(err, identity.ErrUnavailable):
		return commands.Fail(commands.DependencyUnavailable)
	}
	return err
}

func inputAuthorityFields(record inputauthority.Record) map[string]string {
	p := record.Proposal
	fields := map[string]string{"capability": p.Spec.Capability, "proposal": p.ID, "digest": record.Digest, "state": record.State, "mode": "synthetic", "kind": p.Spec.Kind, "recipient": p.Spec.Recipient, "ceiling": p.Spec.CeilingID, "proposer": p.Proposer, "not_before": p.Spec.NotBefore.Format(time.RFC3339Nano), "not_after": p.Spec.NotAfter.Format(time.RFC3339Nano), "proposed_at": record.ProposedAt.Format(time.RFC3339Nano), "expires_at": record.ExpiresAt.Format(time.RFC3339Nano), "rationale": p.Rationale}
	if d := record.Decision; d != nil {
		fields["decided_at"] = d.At.Format(time.RFC3339Nano)
		fields["decided_by"] = d.Actor
		fields["decision_reason"] = d.Rationale
		fields["grant"] = d.GrantID
		fields["recorded_grant_status"] = d.GrantStatus
	}
	return fields
}

func (h inputAuthorityCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	a := h.s.grantAuthority(h.domain)
	if a == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	var record inputauthority.Record
	var err error
	switch h.kind {
	case "propose_console", "propose_human":
		before, e1 := time.Parse(time.RFC3339Nano, ctx.Fields["not_before"])
		after, e2 := time.Parse(time.RFC3339Nano, ctx.Fields["not_after"])
		if e1 != nil || e2 != nil {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "not_before", "not_after")
		}
		spec := inputauthority.Spec{Capability: ctx.Fields["capability"], Workspace: ctx.Workspace, Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: before, NotAfter: after}
		if h.kind == "propose_human" {
			spec.Kind = inputauthority.Human
			spec.Recipient = ctx.Fields["recipient"]
			spec.CeilingID = ctx.Fields["ceiling"]
		}
		record, err = a.proposeLocked(inputauthority.Proposal{ID: ctx.Target, Spec: spec, Proposer: ctx.User, Rationale: ctx.Rationale})
	case "inspect":
		record, err = a.recordLocked(ctx.Workspace, ctx.Target)
	default:
		i := inputauthority.Intent{Workspace: ctx.Workspace, ID: ctx.Target, Actor: ctx.User, Digest: ctx.Fields["digest"], Rationale: ctx.Rationale}
		switch h.kind {
		case "approve":
			record, err = a.approveLocked(i)
		case "reject":
			record, err = a.rejectLocked(i)
		case "withdraw":
			record, err = a.withdrawLocked(i)
		default:
			return commands.Outcome{}, commands.Fail(commands.UnsupportedCommand)
		}
	}
	if err != nil {
		if errors.Is(err, inputauthority.ErrConflict) && record.Proposal.ID != "" && (record.State == inputauthority.Expired || record.State == inputauthority.Superseded) {
			h.s.counter++
		}
		return commands.Outcome{}, inputAuthorityError(err)
	}
	out := commands.Outcome{Fields: inputAuthorityFields(record)}
	if h.kind != "inspect" {
		out.ResultRevision = h.s.Next()
		out.BusinessState = record.State
	}
	return out, nil
}

func (s *Server) inputAuthorityDefinitions() []commands.Definition {
	return s.grantAuthorityDefinitions(inputGrantAuthority)
}
func (s *Server) pricingAuthorityDefinitions() []commands.Definition {
	return s.grantAuthorityDefinitions(pricingGrantAuthority)
}
func (s *Server) grantAuthorityDefinitions(domain grantAuthorityDomain) []commands.Definition {
	prefix := ""
	switch domain {
	case inputGrantAuthority:
		prefix = "input_authority"
	case pricingGrantAuthority:
		prefix = "pricing_authority"
	default:
		return nil
	}
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	times := []commands.Field{{Name: "capability", Type: commands.ID, Required: true}, {Name: "not_before", Type: commands.Text, Required: true}, {Name: "not_after", Type: commands.Text, Required: true}}
	human := append(append([]commands.Field{}, times...), commands.Field{Name: "recipient", Type: commands.ID, Required: true}, commands.Field{Name: "ceiling", Type: commands.ID, Required: true})
	digest := []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
	definitions := []commands.Definition{}
	for _, kind := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw", "inspect"} {
		var fields []commands.Field
		switch kind {
		case "propose_console":
			fields = times
		case "propose_human":
			fields = human
		case "inspect":
		default:
			fields = digest
		}
		definitions = append(definitions, commands.Definition{ID: prefix + "." + kind, Permission: permission, Mutating: kind != "inspect", Rationale: kind != "inspect", Fields: fields, Handler: inputAuthorityCommand{s: s, kind: kind, domain: domain}})
	}
	return definitions
}
