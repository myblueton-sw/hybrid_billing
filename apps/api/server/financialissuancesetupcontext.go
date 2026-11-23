package server

import (
	"errors"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

type issuanceSetupContextCommand struct {
	s    *Server
	kind string
}

func (s *Server) issuanceSetupBoundary(command string, call commands.Call) commands.Code {
	kind := strings.TrimPrefix(command, "issuance_setup.")
	if command != "issuance_setup.context" && command != "issuance_setup.configurations" && !(command == "issuance_setup."+kind && setupMutation(kind)) {
		return ""
	}
	if s.cfg.Seed != SeedSample || s.financialIssuance == nil {
		return commands.UnsupportedCapability
	}
	if call.Client != ConsoleClient {
		return commands.PermissionDenied
	}
	if err := s.financialIssuance.protectedReadCaller(call.Answer.Workspace(), call.Answer.User()); err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) {
			return failure.Code
		}
		return commands.DependencyUnavailable
	}
	if setupMutation(kind) {
		fields := map[string]string{}
		for _, v := range call.Fields {
			fields[v.Name] = v.Value
		}
		if err := s.financialIssuance.setupMutationIdentity(call.Answer.Workspace(), call.Answer.User(), kind, call.Target, call.Rationale, fields); err != nil {
			var failure *commands.Failure
			if errors.As(configurationPublicError(err, nil), &failure) {
				return failure.Code
			}
			return commands.DependencyUnavailable
		}
	}
	return ""
}

func (h issuanceSetupContextCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func (h issuanceSetupContextCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	if h.s.cfg.Seed != SeedSample || h.s.financialIssuance == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if setupMutation(h.kind) {
		return h.s.financialIssuance.applySetupMutation(h.kind, ctx)
	}
	var fields map[string]string
	var err error
	if h.kind == "configurations" {
		fields, err = h.s.financialIssuance.setupConfigurationFields(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields["after"], ctx.Fields["snapshot"])
	} else {
		fields, err = h.s.financialIssuance.setupContextFields(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields["on"])
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{Fields: fields}, nil
}

func (s *Server) issuanceSetupDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.ViewCost, Kind: protectedauthority.ResourceKind, Fields: []string{protectedauthority.EvidenceField}}
	return []commands.Definition{
		{ID: "issuance_setup.context", Permission: permission, Fields: []commands.Field{{Name: "on", Type: commands.Text, Required: true}}, Refusals: []commands.Code{commands.EvidenceStale}, Handler: issuanceSetupContextCommand{s: s, kind: "context"}},
		{ID: "issuance_setup.configurations", Permission: permission, Fields: []commands.Field{{Name: "after", Type: commands.Text}, {Name: "snapshot", Type: commands.Text}}, Refusals: []commands.Code{commands.EvidenceStale}, Handler: issuanceSetupContextCommand{s: s, kind: "configurations"}},
	}
}

func (a *financialIssuance) setupContextFields(workspace, actor, id, rawDate string) (map[string]string, error) {
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	on, valid := parseContractDay(rawDate)
	if !valid || !validRequestID(id) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	s := a.server
	if s.contracts == nil || s.lifecycles == nil || s.orgs == nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	meta := s.contracts.get(workspace, id)
	if meta == nil {
		return nil, commands.Fail(commands.PermissionDenied)
	}
	if meta.workspace != workspace || meta.id != id || !validRequestID(meta.profile) {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	view, err := s.contractRead(workspace, id, on)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	if view.position.State != contracts.Active {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if view.position.Revision == 0 || view.seq == 0 {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	_, version, err := s.orgs.BillingProfile(workspace, meta.profile)
	if err != nil || version == 0 {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	spec := financialConfigurationSpec{Contract: id, ContractRevision: view.position.Revision, BillingProfile: meta.profile, BillingRevision: version}
	scope, err := a.configurationScope(workspace, spec)
	if err != nil {
		return nil, err
	}
	digest, err := financialEvidenceDigest("issuance_setup_scope_v1", struct {
		Workspace, Contract, Profile, On string
		Sequence                         uint64
		Position                         contracts.Position
		Scope                            financialConfigurationScope
	}{workspace, id, meta.profile, rawDate, view.seq, view.position, scope})
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	fields := map[string]string{
		"schema": "hb_issuance_setup_context_v1", "contract": id, "contract_name": meta.name, "on": rawDate,
		"lifecycle_sequence": strconv.FormatUint(view.seq, 10), "contract_revision": strconv.FormatUint(view.position.Revision, 10),
		"billing_profile": meta.profile, "billing_profile_name": orgName(meta.profile), "billing_revision": strconv.FormatUint(version, 10),
		"buyer": scope.BillingProfile.BuyerID, "payer": scope.BillingProfile.PayerID, "issuer": scope.BillingProfile.IssuerID,
		"currency": scope.BillingProfile.Currency, "scope_digest": digest, "profile_basis": "current", "current_eligibility": "unchecked",
	}
	if err := a.boundedDraftFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
