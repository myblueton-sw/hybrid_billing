package server

import (
	"errors"
	"reflect"
	"slices"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

type financialChangeSpec struct {
	Attestation                                                                   string                           `json:",omitempty"`
	Configuration                                                                 *financialConfigurationReference `json:"configuration,omitempty"`
	ID, Profile, ProfileDigest, Field, Channel, ChannelDigest, Provenance, Reason string
	After                                                                         protected.Ref
	Instruction                                                                   protected.Instruction
	Effective                                                                     time.Time
}

type financialChangeVerification struct {
	Attestation              string `json:",omitempty"`
	Actor, Evidence, Outcome string
}
type financialChangeDecision struct {
	Actor    string
	Approval protected.Approval
}
type financialChangeRecord struct {
	Configuration                                    *financialIssuanceConfiguration `json:"configuration,omitempty"`
	Workspace, Requester, RulesDigest, Digest, Phase string
	Spec                                             financialChangeSpec
	Profile                                          financialIssuanceProfile
	Channel                                          financialChannelRecord
	Before                                           protected.Ref
	Module                                           protected.View
	Verification                                     *financialChangeVerification
	Decisions                                        []financialChangeDecision
}

func copyProtectedView(v protected.View) protected.View {
	if v.Verification != nil {
		vv := *v.Verification
		v.Verification = &vv
	}
	v.Decisions = slices.Clone(v.Decisions)
	return v
}
func copyChange(p financialChangeRecord) financialChangeRecord {
	p.Spec.Configuration = copyConfigurationReference(p.Spec.Configuration)
	p.Configuration = copyConfigurationSnapshot(p.Configuration)
	p.Profile = copyIssuanceProfile(p.Profile)
	p.Channel = copyChannelRecord(p.Channel)
	p.Module = copyProtectedView(p.Module)
	if p.Verification != nil {
		v := *p.Verification
		p.Verification = &v
	}
	p.Decisions = slices.Clone(p.Decisions)
	return p
}
func changeDigest(p financialChangeRecord) (string, error) {
	if err := setupOriginShape(p.Spec.Profile, p.Spec.ProfileDigest, p.Spec.Configuration, p.Profile, p.Configuration); err != nil {
		return "", err
	}
	domain := "issuance_protected_change"
	if p.Configuration != nil {
		domain = "issuance_configuration_change_v1"
	}
	p.Digest, p.Phase = "", ""
	p.Module = protected.View{}
	p.Verification = nil
	p.Decisions = nil
	return financialEvidenceDigest(domain, p)
}
func (p financialChangeRecord) change() protected.Change {
	return protected.Change{ID: p.Spec.ID, Subject: p.Channel.Subject.Subject, Field: p.Spec.Field, After: p.Spec.After, Provenance: p.Spec.Provenance, Instruction: p.Spec.Instruction, Effective: p.Spec.Effective}
}
func issuanceChangeScope(subject, field string) string { return subject + "/" + field }
func validChangeSpec(s financialChangeSpec) bool {
	i := s.Instruction
	if s.Attestation != "" && !validPlanHash(s.Attestation) || !validRequestID(s.ID) || !validSetupReference(s.Profile, s.ProfileDigest, s.Configuration) || !validRequestID(s.Channel) || !validPlanHash(s.ChannelDigest) || !validRequestID(s.After.ID) || !validPlanHash(s.After.Fingerprint) || !documentReason(s.Reason) || s.Effective.IsZero() || !validRequestID(i.Evidence) {
		return false
	}
	if s.Provenance != protected.ProvenanceStaff && s.Provenance != protected.ProvenanceDocument && s.Provenance != protected.ProvenancePortal {
		return false
	}
	if i.Channel != "" && !validRequestID(i.Channel) {
		return false
	}
	if i.Kind == protected.ChannelInPerson {
		return i.Sender == "" && i.Channel == ""
	}
	return (i.Kind == protected.ChannelEmail || i.Kind == protected.ChannelPhone || i.Kind == protected.ChannelPortal || i.Kind == protected.ChannelPostal) && validPlanHash(i.Sender)
}
func changeViewEqual(actual, retained protected.View) bool {
	if reflect.DeepEqual(actual, retained) {
		return true
	}
	if actual.State != protected.StateLapsed || retained.State != protected.StateRequested && retained.State != protected.StateConfirmed {
		return false
	}
	expected := copyProtectedView(retained)
	deadline := expected.Change.Effective
	if expected.State == protected.StateConfirmed {
		if expected.Verification == nil {
			return false
		}
		expiry := expected.Verification.At.Add(expected.Rules.Validity)
		if expiry.Before(deadline) {
			deadline = expiry
		}
	}
	expected.State, expected.Ended = protected.StateLapsed, deadline
	return reflect.DeepEqual(actual, expected)
}
func validChangeProgress(p financialChangeRecord) bool {
	if p.Phase != "recorded" && p.Phase != "request_unknown" && p.Phase != "verification_unknown" && p.Phase != "decision_unknown" {
		return false
	}
	v := p.Module
	if p.Verification == nil {
		if v.Verification != nil || len(p.Decisions) != 0 {
			return false
		}
	} else {
		if v.Verification == nil {
			if p.Phase != "verification_unknown" {
				return false
			}
		} else {
			expected := protected.Verification{Channel: p.Spec.Channel, Evidence: p.Verification.Evidence, Outcome: p.Verification.Outcome, Digest: v.Digest}
			if v.Verification.Verification != expected || v.Verification.Verifier != p.Verification.Actor {
				return false
			}
		}
	}
	if len(p.Decisions) != len(v.Decisions) && !(p.Phase == "decision_unknown" && len(p.Decisions) == len(v.Decisions)+1) {
		return false
	}
	for i, d := range v.Decisions {
		if p.Decisions[i].Actor != d.Approver || p.Decisions[i].Approval != d.Approval {
			return false
		}
	}
	return true
}
func (a *financialIssuance) changeModule(p financialChangeRecord) (protected.View, error) {
	if !validChangeProgress(p) {
		return protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-change-lookup") {
		return protected.View{}, commands.Fail(commands.DependencyUnavailable)
	}
	v, err := a.fields.Of(p.Workspace, p.Spec.ID)
	if errors.Is(err, protected.ErrNotFound) && p.Phase == "request_unknown" && p.Module.Digest == "" {
		return protected.View{}, nil
	}
	if err != nil {
		return protected.View{}, err
	}
	if p.Module.Digest == "" || !changeViewEqual(v, p.Module) || v.Change != p.change() || v.Requester != p.Requester || v.Before != p.Before {
		return protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	return v, nil
}
func (a *financialIssuance) changeTrail(workspace, subject, field string) ([]protected.View, error) {
	if a.server.faulty("issuance-change-lookup") {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	trail, err := a.fields.Trail(workspace, subject, field)
	if err != nil {
		return nil, err
	}
	ids := a.changeOrders[issuanceChangeScope(subject, field)]
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		seen[id] = true
	}
	for id, p := range a.changes {
		if p.Workspace == workspace && p.Channel.Subject.Subject == subject && p.Spec.Field == field && !seen[id] {
			return nil, commands.Fail(commands.EvidenceStale)
		}
	}
	at := 0
	for _, id := range ids {
		p, ok := a.changes[id]
		digest, e := changeDigest(p)
		if !ok || e != nil || digest != p.Digest || p.Workspace != workspace || p.Spec.ID != id || p.Channel.Subject.Subject != subject || p.Spec.Field != field {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		v, e := a.changeModule(p)
		if e != nil {
			return nil, e
		}
		if v.Digest == "" {
			if id != ids[len(ids)-1] {
				return nil, commands.Fail(commands.EvidenceStale)
			}
			continue
		}
		if at >= len(trail) || !reflect.DeepEqual(trail[at], v) {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		at++
	}
	if at != len(trail) {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	return trail, nil
}
func (a *financialIssuance) currentChange(p financialChangeRecord, actor, capability string) (protected.View, error) {
	digest, err := changeDigest(p)
	if err != nil || digest != p.Digest {
		return protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.protectedCaller(p.Workspace, actor, capability); err != nil {
		return protected.View{}, err
	}
	profile, configured, err := a.freshSetupOrigin(p.Workspace, p.Spec.Profile, p.Spec.ProfileDigest, actor, p.Spec.Configuration)
	if err != nil {
		return protected.View{}, err
	}
	if !reflect.DeepEqual(profile, p.Profile) || !reflect.DeepEqual(configured, p.Configuration) {
		return protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	return a.changeStanding(p)
}

func (a *financialIssuance) changeStanding(p financialChangeRecord) (protected.View, error) {
	rules, err := a.channelRules(p.Workspace)
	if err != nil {
		return protected.View{}, err
	}
	if rules.Digest != p.RulesDigest {
		return protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	channel, ok := a.channels[p.Spec.Channel]
	if !ok || !reflect.DeepEqual(channel, p.Channel) {
		return protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.currentChannel(channel); err != nil {
		return protected.View{}, err
	}
	if err := a.protectedActor(p.Workspace, p.Requester, protectedauthority.RequestChange); err != nil {
		return protected.View{}, err
	}
	if p.Spec.Attestation != "" {
		if err := a.heldAttestation(p.Workspace, p.Spec.Instruction.Evidence, p.Spec.Attestation, financialAttestationUse{Purpose: "change_instruction", Target: p.Spec.ID, TargetDigest: p.Digest}); err != nil {
			return protected.View{}, err
		}
	}
	if p.Verification != nil && p.Verification.Attestation != "" {
		if err := a.heldAttestation(p.Workspace, p.Verification.Evidence, p.Verification.Attestation, financialAttestationUse{Purpose: "change_verification", Target: p.Spec.ID, TargetDigest: p.Digest}); err != nil {
			return protected.View{}, err
		}
	}
	if p.Verification != nil {
		if err := a.protectedActor(p.Workspace, p.Verification.Actor, protectedauthority.VerifyChange); err != nil {
			return protected.View{}, err
		}
	}
	for _, d := range p.Decisions {
		if err := a.protectedActor(p.Workspace, d.Actor, protectedauthority.ApproveChange); err != nil {
			return protected.View{}, err
		}
	}
	if p.Spec.Instruction.Channel != "" {
		incoming, ok := a.channels[p.Spec.Instruction.Channel]
		if !ok || incoming.State != "admitted" || incoming.Subject.Subject != p.Channel.Subject.Subject || !reflect.DeepEqual(incoming.Subject, p.Channel.Subject) || incoming.Spec.Kind != p.Spec.Instruction.Kind || incoming.Spec.Ref.Fingerprint != p.Spec.Instruction.Sender {
			return protected.View{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentChannel(incoming); err != nil {
			return protected.View{}, err
		}
	}
	if _, err := a.changeTrail(p.Workspace, p.Channel.Subject.Subject, p.Spec.Field); err != nil {
		return protected.View{}, err
	}
	return a.changeModule(p)
}
