package server

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

const protectedWriteMargin = time.Minute

type protectedWriteStep struct {
	Kind, Capability string
	Action           gate.Action
	Rationale, Self  bool
	Fields           []string
	Optional         []string
}

var protectedWriteSteps = []protectedWriteStep{
	{Kind: "propose_protected_rules", Action: gate.AdministerInstallation, Rationale: true, Fields: []string{"rules", "approvals", "validity_seconds", "max_lead_seconds", "channel_age_seconds"}},
	{Kind: "confirm_protected_rules", Action: gate.AdministerInstallation, Rationale: true, Self: true, Fields: []string{"rules", "rules_digest"}},
	{Kind: "register_channel", Capability: protectedauthority.RegisterChannel, Action: gate.RegisterIssuanceChannel, Rationale: true, Fields: []string{"channel", "subject_kind", "kind", "reference", "attestation"}},
	{Kind: "admit_channel", Capability: protectedauthority.AdmitChannel, Action: gate.AdmitIssuanceChannel, Self: true, Fields: []string{"channel", "channel_digest", "attestation"}},
	{Kind: "request_change", Capability: protectedauthority.RequestChange, Action: gate.RequestIssuanceProtectedChange, Rationale: true, Self: true, Fields: []string{"change", "field", "reference", "channel", "channel_digest", "provenance", "instruction", "effective", "attestation"}, Optional: []string{"instruction_channel"}},
	{Kind: "verify_change", Capability: protectedauthority.VerifyChange, Action: gate.VerifyIssuanceProtectedChange, Self: true, Fields: []string{"change", "change_digest", "outcome", "attestation"}},
	{Kind: "decide_change", Capability: protectedauthority.ApproveChange, Action: gate.ApproveIssuanceProtectedChange, Self: true, Fields: []string{"change", "change_digest", "verdict"}, Optional: []string{"reason"}},
}

func protectedWriteCommand(kind string) (protectedWriteStep, bool) {
	for _, step := range protectedWriteSteps {
		if step.Kind == kind {
			return step, true
		}
	}
	return protectedWriteStep{}, false
}

func (s *Server) protectedWriteDefinitions() []commands.Definition {
	out := []commands.Definition{}
	for _, step := range protectedWriteSteps {
		fields := []commands.Field{{Name: "configuration", Type: commands.Text, Required: true}, {Name: "digest", Type: commands.Text, Required: true}}
		for _, name := range step.Fields {
			fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
		}
		for _, name := range step.Optional {
			fields = append(fields, commands.Field{Name: name, Type: commands.Text})
		}
		permission := commands.Permission{Action: step.Action, Kind: protectedauthority.ResourceKind}
		if step.Capability == "" {
			permission.Kind = gate.InstallationKind
		}
		refusals := []commands.Code{commands.EvidenceStale}
		if step.Self {
			refusals = append(refusals, commands.SelfApprovalDenied)
		}
		out = append(out, commands.Definition{ID: "issuance_setup." + step.Kind, Permission: permission, Mutating: true, Rationale: step.Rationale, Fields: fields, Refusals: refusals, Handler: issuanceSetupContextCommand{s: s, kind: step.Kind}})
	}
	return out
}

func (a *financialIssuance) protectedWriteCaller(step protectedWriteStep, workspace, actor string) error {
	if step.Capability != "" {
		return a.protectedCaller(workspace, actor, step.Capability)
	}
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return err
	}
	return a.protectedRuleCaller(workspace, actor)
}

func protectedWriteError(step protectedWriteStep, err error) error {
	var failure *commands.Failure
	if errors.As(err, &failure) {
		switch failure.Code {
		case commands.InvalidRequest, commands.PermissionDenied, commands.PrerequisiteFailed, commands.DependencyUnavailable, commands.UnsupportedCapability, commands.AuthorityUnavailable, commands.EvidenceStale, commands.RevisionConflict, commands.InvalidStage:
			return err
		case commands.SelfApprovalDenied:
			if step.Self {
				return err
			}
		}
		return commands.Fail(commands.PermissionDenied)
	}
	switch {
	case errors.Is(err, inputauthority.ErrDenied):
		return commands.Fail(commands.PermissionDenied)
	case errors.Is(err, protected.ErrConflict), errors.Is(err, protected.ErrInvalid):
		return commands.Fail(commands.EvidenceStale)
	case errors.Is(err, protected.ErrRejected), errors.Is(err, protected.ErrLimit), errors.Is(err, protected.ErrNotFound):
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return commands.Fail(commands.DependencyUnavailable)
}

type protectedWriteObservation struct {
	Rules       *financialProtectedRules
	Channel     *financialChannelRecord
	Change      *financialChangeRecord
	Orders      map[string][]string
	Attestation *financialAttestation
	Module      any
	Missing     bool
}

func (a *financialIssuance) observeProtectedWrite(workspace, family, id, attestation string) protectedWriteObservation {
	var out protectedWriteObservation
	if p, ok := a.attestations[attestation]; ok && attestation != "" {
		if p.Consumed != nil {
			use := *p.Consumed
			p.Consumed = &use
		}
		out.Attestation = &p
	}
	var err error
	switch family {
	case "rules":
		if a.protectedRules != nil {
			p := *a.protectedRules
			out.Rules = &p
		}
		out.Module, err = a.fields.RulesOf(workspace)
	case "channel":
		if p, ok := a.channels[id]; ok {
			p = copyChannelRecord(p)
			out.Channel = &p
		}
		out.Module, err = a.fields.ChannelOf(workspace, id)
	default:
		if p, ok := a.changes[id]; ok {
			p = copyChange(p)
			out.Change = &p
		}
		out.Orders = map[string][]string{}
		for key, ids := range a.changeOrders {
			out.Orders[key] = slices.Clone(ids)
		}
		out.Module, err = a.fields.Peek(workspace, id)
	}
	out.Missing = err != nil
	return out
}

func (a *financialIssuance) pendingAttestation(workspace, id, purpose, target, targetDigest, actor, retained string) (string, error) {
	p, ok := a.attestations[id]
	if !ok || !validRequestID(id) {
		return "", commands.Fail(commands.EvidenceStale)
	}
	use := financialAttestationUse{Purpose: purpose, Target: target, TargetDigest: retained}
	if p.Consumed != nil && retained == "" {
		return "", commands.Fail(commands.EvidenceStale)
	}
	if err := a.attestationFor(workspace, id, p.Digest, purpose, target, targetDigest, actor, use); err != nil {
		return "", err
	}
	return p.Digest, nil
}

func protectedWriteSeconds(text string, limit time.Duration) (time.Duration, bool) {
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil || n < 1 || n > int64(limit/time.Second) || strconv.FormatInt(n, 10) != text {
		return 0, false
	}
	return time.Duration(n) * time.Second, true
}

func (a *financialIssuance) applyProtectedWrite(kind string, ctx commands.Context) (commands.Outcome, error) {
	step, ok := protectedWriteCommand(kind)
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	workspace, actor, contract, f := ctx.Workspace, ctx.User, ctx.Target, ctx.Fields
	if err := a.protectedWriteCaller(step, workspace, actor); err != nil {
		return commands.Outcome{}, protectedWriteError(step, err)
	}
	invalid := func() (commands.Outcome, error) { return commands.Outcome{}, commands.Fail(commands.InvalidRequest) }
	stale := func() (commands.Outcome, error) { return commands.Outcome{}, commands.Fail(commands.EvidenceStale) }
	if !validRequestID(contract) || !validRequestID(f["configuration"]) || !validPlanHash(f["digest"]) || step.Rationale && !documentReason(ctx.Rationale) {
		return invalid()
	}
	c, err := a.confirmedSetupConfiguration(workspace, contract, f["configuration"], f["digest"])
	if err != nil {
		return commands.Outcome{}, protectedWriteError(step, err)
	}
	now := a.server.clock().UTC().Round(0)
	head := map[string]string{"contract": contract, "configuration": c.Spec.ID, "current_eligibility": "unchecked"}
	family, id, attestation := "rules", f["rules"], f["attestation"]
	var run func() (string, error)
	identifier := func(text string) bool { return validRequestID(text) && len(text) <= 128 }
	switch kind {
	case "propose_protected_rules":
		approvals, err := strconv.Atoi(f["approvals"])
		validity, v := protectedWriteSeconds(f["validity_seconds"], 30*24*time.Hour)
		lead, l := protectedWriteSeconds(f["max_lead_seconds"], 366*24*time.Hour)
		age, g := protectedWriteSeconds(f["channel_age_seconds"], 90*24*time.Hour)
		if !identifier(id) || err != nil || strconv.Itoa(approvals) != f["approvals"] || !v || !l || !g {
			return invalid()
		}
		rules := protected.Rules{Approvals: approvals, Validity: validity, MaxLead: lead, ChannelAge: age}
		run = func() (string, error) {
			p, err := a.proposeProtectedRules(workspace, id, actor, ctx.Rationale, rules)
			head["schema"], head["rules"], head["rules_digest"], head["state"] = "hb_issuance_protected_rules_receipt_v1", p.ID, p.Digest, p.State
			return p.State, err
		}
	case "confirm_protected_rules":
		if !identifier(id) || !validPlanHash(f["rules_digest"]) {
			return invalid()
		}
		run = func() (string, error) {
			p, err := a.confirmProtectedRules(workspace, id, f["rules_digest"], actor)
			head["schema"], head["rules"], head["rules_digest"], head["state"] = "hb_issuance_protected_rules_receipt_v1", p.ID, p.Digest, p.State
			return p.State, err
		}
	case "register_channel", "admit_channel":
		family, id = "channel", f["channel"]
		old, exists := a.channels[id]
		if !identifier(id) || !identifier(attestation) {
			return invalid()
		}
		receipt := func(p financialChannelRecord, err error) (string, error) {
			head["schema"], head["channel"], head["channel_digest"], head["state"] = "hb_issuance_protected_channel_receipt_v1", p.Spec.ID, p.Digest, p.State
			return p.State, err
		}
		if kind == "admit_channel" {
			if !validPlanHash(f["channel_digest"]) {
				return invalid()
			}
			if !exists || old.Workspace != workspace || protectedSetupChannelKind(c, old) == "" {
				return stale()
			}
			digest, err := a.pendingAttestation(workspace, attestation, "channel_admission", id, old.Digest, actor, old.Digest)
			if err != nil {
				return commands.Outcome{}, err
			}
			run = func() (string, error) {
				return receipt(a.admitChannelAttested(workspace, id, f["channel_digest"], actor, attestation, digest))
			}
			break
		}
		subject, channelKind := f["subject_kind"], f["kind"]
		ref, known := setupReference(f["reference"], "channel_"+subject)
		selector := financialChannelSelector{Kind: "customer", Contract: c.Spec.Contract, Revision: c.Spec.ContractRevision}
		switch subject {
		case "customer":
		case "entity":
			selector = financialChannelSelector{Kind: "entity", BillingProfile: c.Spec.BillingProfile}
		case "document_kind":
			reference := configurationReference(c)
			selector = financialChannelSelector{Kind: "configuration_kind", Configuration: &reference}
		default:
			return invalid()
		}
		if !known || channelKind != protected.ChannelEmail && channelKind != protected.ChannelPhone && channelKind != protected.ChannelPortal && channelKind != protected.ChannelPostal {
			return invalid()
		}
		retained := ""
		if exists {
			if old.Workspace != workspace || protectedSetupChannelKind(c, old) != subject {
				return stale()
			}
			retained = old.Digest
		}
		digest, err := a.pendingAttestation(workspace, attestation, "channel_registration", id, "", actor, retained)
		if err != nil {
			return commands.Outcome{}, err
		}
		spec := financialChannelSpec{ID: id, Kind: channelKind, Evidence: attestation, Reason: ctx.Rationale, Ref: ref, Selector: selector, Attestation: digest}
		run = func() (string, error) { return receipt(a.registerChannel(workspace, actor, spec)) }
	default:
		family, id = "change", f["change"]
		old, exists := a.changes[id]
		if !identifier(id) {
			return invalid()
		}
		owned := exists && old.Workspace == workspace && old.Configuration != nil && reflect.DeepEqual(*old.Configuration, c)
		receipt := func(p financialChangeRecord, err error) (string, error) {
			head["schema"], head["change"], head["change_digest"], head["phase"], head["state"] = "hb_issuance_protected_change_receipt_v1", p.Spec.ID, p.Digest, p.Phase, p.Module.State
			return p.Module.State, err
		}
		observed, _ := a.fields.Peek(workspace, id)
		late := func(deadline time.Time) bool { return now.Add(protectedWriteMargin).After(deadline) }
		switch kind {
		case "request_change":
			field, provenance, form := f["field"], f["provenance"], f["instruction"]
			ref, known := setupReference(f["reference"], field)
			effective, err := time.Parse(time.RFC3339Nano, f["effective"])
			if !protectedSetupField(field) || !known || err != nil || !identifier(attestation) || !identifier(f["channel"]) || !validPlanHash(f["channel_digest"]) ||
				provenance != protected.ProvenanceStaff && provenance != protected.ProvenanceDocument && provenance != protected.ProvenancePortal ||
				form != "in_person" && form != "registered_channel" && form != "unregistered_sender" || (form == "registered_channel") != identifier(f["instruction_channel"]) || form != "registered_channel" && f["instruction_channel"] != "" {
				return invalid()
			}
			public := func(channel string) (financialChannelRecord, bool) {
				p, ok := a.channels[channel]
				return p, ok && p.Workspace == workspace && p.Spec.Attestation != "" && protectedSetupChannelKind(c, p) != ""
			}
			if _, ok := public(f["channel"]); !ok || exists && !owned {
				return stale()
			}
			instruction := protected.Instruction{Kind: protected.ChannelInPerson, Evidence: attestation}
			switch form {
			case "registered_channel":
				incoming, ok := public(f["instruction_channel"])
				if !ok {
					return stale()
				}
				instruction = protected.Instruction{Kind: incoming.Spec.Kind, Channel: incoming.Spec.ID, Sender: incoming.Spec.Ref.Fingerprint, Evidence: attestation}
			case "unregistered_sender":
				sender, _ := setupReference("synthetic_unregistered_sender", "unregistered_sender")
				instruction = protected.Instruction{Kind: protected.ChannelEmail, Sender: sender.Fingerprint, Evidence: attestation}
			}
			effective = effective.UTC().Round(0)
			if !exists && late(effective) {
				return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
			}
			retained := ""
			if exists {
				retained = old.Digest
			}
			digest, err := a.pendingAttestation(workspace, attestation, "change_instruction", id, "", actor, retained)
			if err != nil {
				return commands.Outcome{}, err
			}
			reference := configurationReference(c)
			spec := financialChangeSpec{Configuration: &reference, ID: id, Field: field, Channel: f["channel"], ChannelDigest: f["channel_digest"], Provenance: provenance, Reason: ctx.Rationale, After: ref, Instruction: instruction, Effective: effective, Attestation: digest}
			run = func() (string, error) { return receipt(a.requestProtectedChange(workspace, actor, spec)) }
		case "verify_change":
			outcome := f["outcome"]
			if !validPlanHash(f["change_digest"]) || !identifier(attestation) || outcome != protected.OutcomeConfirmed && outcome != protected.OutcomeDenied {
				return invalid()
			}
			if !owned {
				return stale()
			}
			if old.Phase == "recorded" && old.Verification == nil && observed.State == protected.StateRequested && late(observed.Change.Effective) {
				return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
			}
			digest, err := a.pendingAttestation(workspace, attestation, "change_verification", id, old.Digest, actor, old.Digest)
			if err != nil {
				return commands.Outcome{}, err
			}
			run = func() (string, error) {
				return receipt(a.verifyProtectedChangeAttested(workspace, id, f["change_digest"], actor, attestation, outcome, digest))
			}
		default:
			attestation = ""
			verdict, reason := f["verdict"], f["reason"]
			if !validPlanHash(f["change_digest"]) || !(verdict == protected.VerdictApprove && reason == "" || verdict == protected.VerdictReject && (reason == "incorrect_details" || reason == "missing_evidence" || reason == "suspected_fraud" || reason == "other")) {
				return invalid()
			}
			if !owned {
				return stale()
			}
			decided := slices.ContainsFunc(old.Decisions, func(d financialChangeDecision) bool { return d.Actor == actor })
			if old.Phase == "recorded" && !decided && observed.State == protected.StateConfirmed && observed.Verification != nil {
				deadline := observed.Change.Effective
				if expiry := observed.Verification.At.Add(observed.Rules.Validity); expiry.Before(deadline) {
					deadline = expiry
				}
				if late(deadline) {
					return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
				}
			}
			run = func() (string, error) {
				return receipt(a.decideProtectedChange(workspace, id, f["change_digest"], actor, f["verdict"], f["reason"]))
			}
		}
	}
	before := a.observeProtectedWrite(workspace, family, id, attestation)
	state, err := run()
	changed := !reflect.DeepEqual(before, a.observeProtectedWrite(workspace, family, id, attestation))
	if err != nil {
		if changed {
			return commands.Outcome{}, errConfigurationUncertain
		}
		return commands.Outcome{}, protectedWriteError(step, err)
	}
	if err := a.boundedDraftFields(head); err != nil || state == "" {
		if changed {
			return commands.Outcome{}, errConfigurationUncertain
		}
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	return commands.Outcome{Fields: head, BusinessState: state, ResultRevision: a.server.Next()}, nil
}
