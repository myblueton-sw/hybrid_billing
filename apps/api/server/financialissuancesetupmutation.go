package server

import (
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func setupMutation(kind string) bool {
	return kind == "propose_configuration" || kind == "confirm_configuration"
}

func (a *financialIssuance) setupProposalSpec(workspace, actor, target string, fields map[string]string) (financialConfigurationSpec, *financialConfigurationSetupContext, error) {
	setup := &financialConfigurationSetupContext{On: fields["on"], ScopeDigest: fields["scope_digest"]}
	if !validRequestID(target) || !validConfigurationSetup(setup) {
		return financialConfigurationSpec{}, nil, commands.Fail(commands.InvalidRequest)
	}
	context, err := a.setupContextFields(workspace, actor, fields["contract"], setup.On)
	if err != nil {
		return financialConfigurationSpec{}, nil, err
	}
	if context["scope_digest"] != setup.ScopeDigest {
		return financialConfigurationSpec{}, nil, commands.Fail(commands.EvidenceStale)
	}
	contractRevision, err := strconv.ParseUint(context["contract_revision"], 10, 64)
	if err != nil {
		return financialConfigurationSpec{}, nil, commands.Fail(commands.EvidenceStale)
	}
	billingRevision, err := strconv.ParseUint(context["billing_revision"], 10, 64)
	if err != nil {
		return financialConfigurationSpec{}, nil, commands.Fail(commands.EvidenceStale)
	}
	spec := financialConfigurationSpec{ID: target, Revision: 1, Contract: fields["contract"], ContractRevision: contractRevision, BillingProfile: context["billing_profile"], BillingRevision: billingRevision, Mode: fields["mode"], Profile: fields["profile"], Region: fields["region"], Tax: fields["tax"], Zone: fields["zone"], Languages: strings.Split(fields["languages"], ",")}
	kind, err := configurationKind(spec)
	if err != nil {
		return financialConfigurationSpec{}, nil, err
	}
	spec.Languages = kind.Languages
	return spec, setup, nil
}

func (a *financialIssuance) setupMutationIdentity(workspace, actor, kind, target, reason string, fields map[string]string) error {
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return err
	}
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return err
	}
	if !validRequestID(target) || !documentReason(reason) {
		return commands.Fail(commands.InvalidRequest)
	}
	old, exists := a.configurations[target]
	if exists {
		if old.Workspace != workspace || old.Spec.ID != target {
			return commands.Fail(commands.EvidenceStale)
		}
		if old.Setup == nil {
			return commands.Fail(commands.UnsupportedCapability)
		}
		if kind == "propose_configuration" && old.Proposer != actor {
			return commands.Fail(commands.PermissionDenied)
		}
		if kind == "confirm_configuration" {
			if old.Proposer == actor {
				return commands.Fail(commands.SelfApprovalDenied)
			}
			if old.Confirmer != "" && old.Confirmer != actor {
				return commands.Fail(commands.PermissionDenied)
			}
		}
		if err := a.currentConfiguration(workspace, old); err != nil {
			return err
		}
	}
	if kind == "propose_configuration" {
		spec, setup, err := a.setupProposalSpec(workspace, actor, target, fields)
		if err != nil {
			return err
		}
		if !exists && !setupSampleOption(spec) {
			return commands.Fail(commands.UnsupportedCapability)
		}
		if exists && (!reflect.DeepEqual(spec, old.Spec) || !reflect.DeepEqual(setup, old.Setup) || reason != old.Reason) {
			return commands.Fail(commands.EvidenceStale)
		}
		return nil
	}
	if !exists || old.Digest != fields["digest"] {
		return commands.Fail(commands.EvidenceStale)
	}
	context, err := a.setupContextFields(workspace, actor, old.Spec.Contract, old.Setup.On)
	if err != nil {
		return err
	}
	if context["scope_digest"] != old.Setup.ScopeDigest {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialIssuance) applySetupMutation(kind string, ctx commands.Context) (commands.Outcome, error) {
	if err := a.setupMutationIdentity(ctx.Workspace, ctx.User, kind, ctx.Target, ctx.Rationale, ctx.Fields); err != nil {
		return commands.Outcome{}, configurationPublicError(err, nil)
	}
	if a.documents == nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	effect := &financialConfigurationEffect{}
	var p financialIssuanceConfiguration
	var err error
	if kind == "propose_configuration" {
		var spec financialConfigurationSpec
		var setup *financialConfigurationSetupContext
		spec, setup, err = a.setupProposalSpec(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields)
		if err == nil {
			p, err = a.proposeConfigurationTracked(ctx.Workspace, ctx.User, ctx.Rationale, spec, setup, effect, a.admitConfigurationReceipt)
		}
	} else {
		p, err = a.confirmConfigurationTracked(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User, effect, a.admitConfigurationReceipt)
	}
	if err != nil {
		return commands.Outcome{}, configurationPublicError(err, effect)
	}
	fields := configurationReceipt(p)
	if err := a.boundedDraftFields(fields); err != nil {
		return commands.Outcome{}, configurationPublicError(err, effect)
	}
	return commands.Outcome{Fields: fields, BusinessState: p.State, ResultRevision: a.server.Next()}, nil
}

func (s *Server) issuanceSetupMutationDefinitions() []commands.Definition {
	out := []commands.Definition{}
	for _, kind := range []string{"propose_configuration", "confirm_configuration"} {
		names := []string{"digest"}
		if kind == "propose_configuration" {
			names = []string{"contract", "on", "scope_digest", "mode", "profile", "region", "tax", "zone", "languages"}
		}
		fields := []commands.Field{}
		for _, name := range names {
			fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
		}
		out = append(out, commands.Definition{ID: "issuance_setup." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Mutating: true, Rationale: true, Fields: fields, Refusals: []commands.Code{commands.EvidenceStale, commands.SelfApprovalDenied}, Handler: issuanceSetupContextCommand{s: s, kind: kind}})
	}
	for _, step := range setupAttestationSteps {
		text := func(name string, required bool) commands.Field {
			return commands.Field{Name: name, Type: commands.Text, Required: required}
		}
		scope := []commands.Field{text("configuration", true), text("digest", true), text("attestation", true)}
		attest := slices.Concat(scope, []commands.Field{text("target", true), text("statement", true), text("locator", false)})
		if step.Existing {
			attest = append(attest, text("target_digest", true))
		}
		permission := commands.Permission{Action: step.Action, Kind: protectedauthority.ResourceKind}
		refusals := []commands.Code{commands.EvidenceStale}
		if step.Existing {
			refusals = append(refusals, commands.SelfApprovalDenied)
		}
		out = append(out, commands.Definition{ID: "issuance_setup.attest_" + step.Purpose, Permission: permission, Mutating: true, Fields: attest, Refusals: refusals, Handler: issuanceSetupContextCommand{s: s, kind: "attest_" + step.Purpose}})
		out = append(out, commands.Definition{ID: "issuance_setup.withdraw_" + step.Purpose, Permission: permission, Mutating: true, Fields: scope, Refusals: []commands.Code{commands.EvidenceStale}, Handler: issuanceSetupContextCommand{s: s, kind: "withdraw_" + step.Purpose}})
	}
	return out
}
