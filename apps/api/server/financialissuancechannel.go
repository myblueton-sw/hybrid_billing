package server

import (
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

type financialChannelSpec struct {
	ID, Kind, Evidence, Reason string
	Ref                        protected.Ref
	Selector                   financialChannelSelector
}

type financialChannelRecord struct {
	Workspace, Registrar, Admitter, Admission, RulesDigest, Digest, State string
	Spec                                                                  financialChannelSpec
	Subject                                                               financialChannelSubject
	Module                                                                protected.ChannelView
}

func channelDigest(p financialChannelRecord) (string, error) {
	if err := channelOrigin(p); err != nil {
		return "", err
	}
	domain := "issuance_channel"
	if p.Spec.Selector.Configuration != nil {
		domain = "issuance_configuration_channel_v1"
	}
	p.Admitter, p.Admission, p.Digest, p.State = "", "", "", ""
	p.Module = protected.ChannelView{}
	return financialEvidenceDigest(domain, p)
}
func copyChannelRecord(p financialChannelRecord) financialChannelRecord {
	p.Spec.Selector = copyChannelSelector(p.Spec.Selector)
	p.Subject = copyChannelSubject(p.Subject)
	return p
}
func (a *financialIssuance) channelModule(workspace, id string) (protected.ChannelView, error) {
	if a.server.faulty("issuance-channel-lookup") {
		return protected.ChannelView{}, commands.Fail(commands.DependencyUnavailable)
	}
	return a.fields.ChannelOf(workspace, id)
}
func (p financialChannelRecord) channel() protected.Channel {
	return protected.Channel{ID: p.Spec.ID, Subject: p.Subject.Subject, Kind: p.Spec.Kind, Ref: p.Spec.Ref, Evidence: p.Spec.Evidence}
}
func validChannelSpec(p financialChannelSpec) bool {
	return validRequestID(p.ID) && len(p.ID) <= 128 && validRequestID(p.Evidence) && len(p.Evidence) <= 128 && validRequestID(p.Ref.ID) && len(p.Ref.ID) <= 128 && validPlanHash(p.Ref.Fingerprint) && documentReason(p.Reason) && (p.Kind == protected.ChannelEmail || p.Kind == protected.ChannelPhone || p.Kind == protected.ChannelPortal || p.Kind == protected.ChannelPostal)
}
func (a *financialIssuance) currentChannel(p financialChannelRecord) error {
	digest, err := channelDigest(p)
	if err != nil || digest != p.Digest || !validChannelSpec(p.Spec) || !reflect.DeepEqual(p.Spec.Selector, p.Subject.Selector) || p.Registrar == "" {
		return commands.Fail(commands.EvidenceStale)
	}
	rules, err := a.channelRules(p.Workspace)
	if err != nil {
		return err
	}
	if rules.Digest != p.RulesDigest {
		return commands.Fail(commands.EvidenceStale)
	}
	if err := a.currentChannelSubject(p.Workspace, p.Subject); err != nil {
		return err
	}
	if err := a.protectedActor(p.Workspace, p.Registrar, protectedauthority.RegisterChannel); err != nil {
		return err
	}
	if p.Admitter != "" {
		if p.Admitter == p.Registrar || !validRequestID(p.Admission) || len(p.Admission) > 128 {
			return commands.Fail(commands.EvidenceStale)
		}
		if err := a.protectedActor(p.Workspace, p.Admitter, protectedauthority.AdmitChannel); err != nil {
			return err
		}
	}
	v, err := a.channelModule(p.Workspace, p.Spec.ID)
	if p.State == "registration_unknown" && errors.Is(err, protected.ErrNotFound) && p.Module == (protected.ChannelView{}) && p.Admitter == "" && p.Admission == "" {
		return nil
	}
	if err != nil {
		if errors.Is(err, protected.ErrNotFound) {
			return commands.Fail(commands.EvidenceStale)
		}
		return err
	}
	if v != p.Module || v.Channel != p.channel() || v.Registrar != p.Registrar || v.Registered.IsZero() || v.RetiredBy != "" || !v.Retired.IsZero() {
		return commands.Fail(commands.EvidenceStale)
	}
	switch p.State {
	case "registered", "registration_unknown":
		if p.Admitter == "" && p.Admission == "" && v.Admitter == "" && v.Admission == "" && v.Admitted.IsZero() {
			return nil
		}
	case "admission_unknown":
		if p.Admitter != "" && (v.Admitter == "" && v.Admission == "" && v.Admitted.IsZero() || v.Admitter == p.Admitter && v.Admission == p.Admission && !v.Admitted.IsZero()) {
			return nil
		}
	case "admitted":
		if p.Admitter != "" && v.Admitter == p.Admitter && v.Admission == p.Admission && !v.Admitted.IsZero() {
			return nil
		}
	}
	return commands.Fail(commands.EvidenceStale)
}

func (a *financialIssuance) registerChannel(workspace, actor string, spec financialChannelSpec) (financialChannelRecord, error) {
	if err := a.protectedCaller(workspace, actor, protectedauthority.RegisterChannel); err != nil {
		return financialChannelRecord{}, err
	}
	if !validChannelSpec(spec) {
		return financialChannelRecord{}, commands.Fail(commands.InvalidRequest)
	}
	subject, err := a.channelSubject(workspace, spec.Selector)
	if err != nil {
		return financialChannelRecord{}, err
	}
	rules, err := a.channelRules(workspace)
	if err != nil {
		return financialChannelRecord{}, err
	}
	p := financialChannelRecord{Workspace: workspace, Registrar: actor, Spec: spec, Subject: subject, RulesDigest: rules.Digest}
	p.Digest, err = channelDigest(p)
	if err != nil {
		return financialChannelRecord{}, err
	}
	if old, ok := a.channels[spec.ID]; ok {
		if old.Workspace != workspace || old.Registrar != actor || old.Digest != p.Digest {
			return financialChannelRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentChannel(old); err != nil {
			return financialChannelRecord{}, err
		}
		if old.State != "registration_unknown" {
			return copyChannelRecord(old), nil
		}
		p = old
	} else {
		if len(a.channels) >= financialIssuanceLimit {
			return financialChannelRecord{}, commands.Fail(commands.PrerequisiteFailed)
		}
		_, err := a.channelModule(workspace, spec.ID)
		if !errors.Is(err, protected.ErrNotFound) {
			if err != nil {
				return financialChannelRecord{}, err
			}
			return financialChannelRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-channel-before-register") {
			return financialChannelRecord{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State = "registration_unknown"
		a.channels[spec.ID] = copyChannelRecord(p)
	}
	v, err := a.fields.Register(workspace, p.channel(), protected.Actor{ID: actor, Kind: protected.KindHuman})
	if err != nil {
		return financialChannelRecord{}, err
	}
	if v.Channel != p.channel() || v.Registrar != actor || v.Admitter != "" || v.RetiredBy != "" {
		return financialChannelRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = v
	a.channels[spec.ID] = copyChannelRecord(p)
	if a.server.faulty("issuance-channel-after-register") {
		return financialChannelRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "registered"
	a.channels[spec.ID] = copyChannelRecord(p)
	if a.server.faulty("issuance-channel-after-registration-publication") {
		return financialChannelRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyChannelRecord(p), nil
}

func (a *financialIssuance) admitChannel(workspace, id, digest, actor, evidence string) (financialChannelRecord, error) {
	if err := a.protectedCaller(workspace, actor, protectedauthority.AdmitChannel); err != nil {
		return financialChannelRecord{}, err
	}
	if !validRequestID(evidence) || len(evidence) > 128 {
		return financialChannelRecord{}, commands.Fail(commands.InvalidRequest)
	}
	p, ok := a.channels[id]
	if !ok || p.Spec.ID != id || p.Workspace != workspace || p.Digest != digest {
		return financialChannelRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if p.Registrar == actor {
		return financialChannelRecord{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Admitter != "" && (p.Admitter != actor || p.Admission != evidence) {
		return financialChannelRecord{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentChannel(p); err != nil {
		return financialChannelRecord{}, err
	}
	if p.State == "admitted" {
		return copyChannelRecord(p), nil
	}
	if p.State != "registered" && p.State != "admission_unknown" {
		return financialChannelRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if a.server.faulty("issuance-channel-before-admit") {
		return financialChannelRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.Admitter, p.Admission, p.State = actor, evidence, "admission_unknown"
	a.channels[id] = copyChannelRecord(p)
	v, err := a.fields.Admit(workspace, id, evidence, protected.Actor{ID: actor, Kind: protected.KindHuman})
	if err != nil {
		return financialChannelRecord{}, err
	}
	expected := p.Module
	expected.Admitter, expected.Admission, expected.Admitted = actor, evidence, v.Admitted
	if !reflect.DeepEqual(v, expected) || v.Admitted.IsZero() {
		return financialChannelRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = v
	a.channels[id] = copyChannelRecord(p)
	if a.server.faulty("issuance-channel-after-admit") {
		return financialChannelRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "admitted"
	a.channels[id] = copyChannelRecord(p)
	if a.server.faulty("issuance-channel-after-admission-publication") {
		return financialChannelRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyChannelRecord(p), nil
}
