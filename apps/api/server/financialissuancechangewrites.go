package server

import (
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func (a *financialIssuance) requestProtectedChange(workspace, actor string, spec financialChangeSpec) (financialChangeRecord, error) {
	if err := a.protectedCaller(workspace, actor, protectedauthority.RequestChange); err != nil {
		return financialChangeRecord{}, err
	}
	profile, configured, err := a.freshSetupOrigin(workspace, spec.Profile, spec.ProfileDigest, actor, spec.Configuration)
	if err != nil {
		return financialChangeRecord{}, err
	}
	if !validChangeSpec(spec) {
		return financialChangeRecord{}, commands.Fail(commands.InvalidRequest)
	}
	spec.Effective = spec.Effective.UTC().Round(0)
	subject, err := a.setupChangeSubject(profile, configured, spec.Field)
	if err != nil {
		return financialChangeRecord{}, err
	}
	channel, ok := a.channels[spec.Channel]
	if !ok || channel.Workspace != workspace || channel.Digest != spec.ChannelDigest || channel.State != "admitted" || !reflect.DeepEqual(subject, channel.Subject) {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.currentChannel(channel); err != nil {
		return financialChangeRecord{}, err
	}
	rules, err := a.channelRules(workspace)
	if err != nil {
		return financialChangeRecord{}, err
	}
	if actor == channel.Registrar || actor == channel.Admitter {
		return financialChangeRecord{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if spec.Instruction.Channel == spec.Channel || spec.Instruction.Sender == channel.Spec.Ref.Fingerprint {
		return financialChangeRecord{}, commands.Fail(commands.InvalidRequest)
	}
	if spec.Instruction.Channel == "" && spec.Instruction.Sender != "" {
		if a.server.faulty("issuance-change-channel-lookup") {
			return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
		}
		declared, err := a.fields.ChannelsOf(workspace, subject.Subject)
		if err != nil {
			return financialChangeRecord{}, err
		}
		for _, v := range declared {
			if v.Channel.Ref.Fingerprint == spec.Instruction.Sender {
				return financialChangeRecord{}, commands.Fail(commands.InvalidRequest)
			}
		}
	}
	if spec.Instruction.Channel != "" {
		incoming, ok := a.channels[spec.Instruction.Channel]
		if !ok || incoming.State != "admitted" || !reflect.DeepEqual(incoming.Subject, subject) || incoming.Spec.Kind != spec.Instruction.Kind || incoming.Spec.Ref.Fingerprint != spec.Instruction.Sender {
			return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentChannel(incoming); err != nil {
			return financialChangeRecord{}, err
		}
	}
	if old, ok := a.changes[spec.ID]; ok {
		if old.Workspace != workspace || old.Requester != actor || !reflect.DeepEqual(old.Spec, spec) {
			return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
		}
		view, err := a.currentChange(old, actor, protectedauthority.RequestChange)
		if err != nil {
			return financialChangeRecord{}, err
		}
		if old.Phase != "request_unknown" {
			out := copyChange(old)
			out.Module = view
			return out, nil
		}
		if view.Digest != "" {
			old.Module = view
			old.Phase = "recorded"
			a.changes[spec.ID] = copyChange(old)
			return copyChange(old), nil
		}
		return a.writeProtectedRequest(old, rules)
	}
	now := a.server.clock()
	if !spec.Effective.After(now) || spec.Effective.After(now.Add(rules.Rules.MaxLead)) || channel.Module.Admitted.After(now.Add(-rules.Rules.ChannelAge)) {
		return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if len(a.changes) >= financialIssuanceLimit {
		return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	trail, err := a.changeTrail(workspace, subject.Subject, spec.Field)
	if err != nil {
		return financialChangeRecord{}, err
	}
	key := issuanceChangeScope(subject.Subject, spec.Field)
	for _, id := range a.changeOrders[key] {
		if a.changes[id].Phase != "recorded" {
			return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
		}
	}
	var before protected.Ref
	for _, v := range trail {
		if v.State == protected.StateRequested || v.State == protected.StateConfirmed {
			return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
		}
		if v.State == protected.StateScheduled {
			if !spec.Effective.After(v.Change.Effective) {
				return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
			}
			before = v.Change.After
		}
	}
	if before == spec.After {
		return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if _, err := a.fields.Of(workspace, spec.ID); !errors.Is(err, protected.ErrNotFound) {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p := financialChangeRecord{Workspace: workspace, Requester: actor, RulesDigest: rules.Digest, Spec: spec, Profile: profile, Configuration: configured, Channel: copyChannelRecord(channel), Before: before}
	p.Digest, err = changeDigest(p)
	if err != nil {
		return financialChangeRecord{}, err
	}
	use := financialAttestationUse{Purpose: "change_instruction", Target: spec.ID, TargetDigest: p.Digest}
	if spec.Attestation != "" {
		if err := a.attestationFor(workspace, spec.Instruction.Evidence, spec.Attestation, use.Purpose, spec.ID, "", actor, use); err != nil {
			return financialChangeRecord{}, err
		}
	}
	if a.server.faulty("issuance-change-before-request") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.Phase = "request_unknown"
	a.changes[spec.ID] = copyChange(p)
	a.changeOrders[key] = append(a.changeOrders[key], spec.ID)
	if spec.Attestation != "" {
		a.consumeAttestation(spec.Instruction.Evidence, use)
	}
	return a.writeProtectedRequest(p, rules)
}
func (a *financialIssuance) writeProtectedRequest(p financialChangeRecord, rules financialProtectedRules) (financialChangeRecord, error) {
	v, err := a.fields.Request(p.Workspace, p.change(), protected.Actor{ID: p.Requester, Kind: protected.KindHuman})
	if err != nil {
		return financialChangeRecord{}, err
	}
	if v.Change != p.change() || v.Before != p.Before || v.Requester != p.Requester || v.Rules != rules.Rules || v.Revision != rules.Module.Revision || v.State != protected.StateRequested || v.Verification != nil || len(v.Decisions) != 0 || v.Digest == "" {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = copyProtectedView(v)
	a.changes[p.Spec.ID] = copyChange(p)
	if a.server.faulty("issuance-change-after-request") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.Phase = "recorded"
	a.changes[p.Spec.ID] = copyChange(p)
	if a.server.faulty("issuance-change-after-request-publication") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyChange(p), nil
}

func (a *financialIssuance) verifyProtectedChange(workspace, id, digest, actor, evidence, outcome string) (financialChangeRecord, error) {
	return a.verifyProtectedChangeAttested(workspace, id, digest, actor, evidence, outcome, "")
}

func (a *financialIssuance) verifyProtectedChangeAttested(workspace, id, digest, actor, evidence, outcome, attestation string) (financialChangeRecord, error) {
	if err := a.protectedCaller(workspace, actor, protectedauthority.VerifyChange); err != nil {
		return financialChangeRecord{}, err
	}
	p, ok := a.changes[id]
	if !ok || p.Spec.ID != id || p.Workspace != workspace || p.Digest != digest {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if !validRequestID(evidence) || (outcome != protected.OutcomeConfirmed && outcome != protected.OutcomeDenied) {
		return financialChangeRecord{}, commands.Fail(commands.InvalidRequest)
	}
	intent := financialChangeVerification{Actor: actor, Evidence: evidence, Outcome: outcome, Attestation: attestation}
	if p.Verification != nil && *p.Verification != intent {
		return financialChangeRecord{}, commands.Fail(commands.PermissionDenied)
	}
	use := financialAttestationUse{Purpose: "change_verification", Target: id, TargetDigest: p.Digest}
	if attestation != "" {
		if err := a.attestationFor(workspace, evidence, attestation, use.Purpose, id, p.Digest, actor, use); err != nil {
			return financialChangeRecord{}, err
		}
	}
	v, err := a.currentChange(p, actor, protectedauthority.VerifyChange)
	if err != nil {
		return financialChangeRecord{}, err
	}
	if actor == p.Requester || actor == p.Channel.Registrar || actor == p.Channel.Admitter {
		return financialChangeRecord{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Phase == "decision_unknown" {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	if p.Verification != nil && v.Verification != nil {
		p.Module = v
		p.Phase = "recorded"
		a.changes[id] = copyChange(p)
		return copyChange(p), nil
	}
	if (p.Phase != "recorded" && p.Phase != "verification_unknown") || v.State != protected.StateRequested {
		return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if a.server.faulty("issuance-change-before-verify") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.Verification = &intent
	p.Phase = "verification_unknown"
	a.changes[id] = copyChange(p)
	if attestation != "" {
		a.consumeAttestation(evidence, use)
	}
	verification := protected.Verification{Channel: p.Spec.Channel, Evidence: evidence, Outcome: outcome, Digest: v.Digest}
	actual, err := a.fields.Verify(workspace, id, verification, protected.Actor{ID: actor, Kind: protected.KindHuman})
	if err != nil {
		return financialChangeRecord{}, err
	}
	if actual.Verification == nil || actual.Verification.Verification != verification || actual.Verification.Verifier != actor || actual.Verification.Digest == "" {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	expected := copyProtectedView(v)
	expected.Verification = actual.Verification
	expected.State = protected.StateConfirmed
	if outcome == protected.OutcomeDenied {
		expected.State = protected.StateRejected
		expected.EndedBy = actor
		expected.Ended = actual.Verification.At
	}
	if !reflect.DeepEqual(expected, actual) {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = copyProtectedView(actual)
	a.changes[id] = copyChange(p)
	if a.server.faulty("issuance-change-after-verify") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.Phase = "recorded"
	a.changes[id] = copyChange(p)
	if a.server.faulty("issuance-change-after-verification-publication") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyChange(p), nil
}

func (a *financialIssuance) decideProtectedChange(workspace, id, digest, actor, verdict, reason string) (financialChangeRecord, error) {
	if err := a.protectedCaller(workspace, actor, protectedauthority.ApproveChange); err != nil {
		return financialChangeRecord{}, err
	}
	p, ok := a.changes[id]
	if !ok || p.Spec.ID != id || p.Workspace != workspace || p.Digest != digest {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	valid := verdict == protected.VerdictApprove && reason == "" || verdict == protected.VerdictReject && (reason == "incorrect_details" || reason == "missing_evidence" || reason == "suspected_fraud" || reason == "other")
	if !valid {
		return financialChangeRecord{}, commands.Fail(commands.InvalidRequest)
	}
	v, err := a.currentChange(p, actor, protectedauthority.ApproveChange)
	if err != nil {
		return financialChangeRecord{}, err
	}
	if v.Verification == nil {
		return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if actor == p.Requester || actor == v.Verification.Verifier || actor == p.Channel.Registrar || actor == p.Channel.Admitter {
		return financialChangeRecord{}, commands.Fail(commands.SelfApprovalDenied)
	}
	approval := protected.Approval{Verdict: verdict, Reason: reason, Digest: v.Verification.Digest}
	if p.Phase == "decision_unknown" && (len(p.Decisions) == 0 || p.Decisions[len(p.Decisions)-1].Actor != actor) {
		return financialChangeRecord{}, commands.Fail(commands.PermissionDenied)
	}
	found := -1
	for i, d := range p.Decisions {
		if d.Actor == actor {
			if d.Approval != approval {
				return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
			}
			found = i
		}
	}
	if found >= 0 && found < len(v.Decisions) {
		p.Module = v
		p.Phase = "recorded"
		a.changes[id] = copyChange(p)
		return copyChange(p), nil
	}

	if (p.Phase != "recorded" && p.Phase != "decision_unknown") || v.State != protected.StateConfirmed {
		return financialChangeRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if a.server.faulty("issuance-change-before-decide") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	if found < 0 {
		p.Decisions = append(p.Decisions, financialChangeDecision{Actor: actor, Approval: approval})
	}
	p.Phase = "decision_unknown"
	a.changes[id] = copyChange(p)
	actual, err := a.fields.Approve(workspace, id, approval, protected.Actor{ID: actor, Kind: protected.KindHuman})
	if err != nil {
		return financialChangeRecord{}, err
	}
	if len(actual.Decisions) != len(v.Decisions)+1 {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	decision := actual.Decisions[len(actual.Decisions)-1]
	if decision.Approver != actor || decision.Approval != approval {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	expected := copyProtectedView(v)
	expected.Decisions = append(expected.Decisions, decision)
	if verdict == protected.VerdictReject {
		expected.State = protected.StateRejected
		expected.EndedBy = actor
		expected.Ended = decision.At
	} else if len(expected.Decisions) >= expected.Rules.Approvals {
		expected.State = protected.StateScheduled
		expected.Scheduled = decision.At
	}
	if !reflect.DeepEqual(actual, expected) {
		return financialChangeRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = copyProtectedView(actual)
	a.changes[id] = copyChange(p)
	if a.server.faulty("issuance-change-after-decide") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.Phase = "recorded"
	a.changes[id] = copyChange(p)
	if a.server.faulty("issuance-change-after-decision-publication") {
		return financialChangeRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyChange(p), nil
}
