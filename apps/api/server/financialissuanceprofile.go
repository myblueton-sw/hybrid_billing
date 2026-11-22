package server

import (
	"errors"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

type financialIssuanceProfileSpec struct {
	ID, Mode, Profile, Region, Tax, Zone string
	Languages                            []string
}

type financialIssuanceProfile struct {
	Spec                                       financialIssuanceProfileSpec
	Source                                     financialIssuanceSourceView
	Kind                                       issuance.Kind
	Proposer, Confirmer, Reason, Digest, State string
}

func issuanceProfileKind(spec financialIssuanceProfileSpec) (issuance.Kind, error) {
	bad := func() (issuance.Kind, error) { return issuance.Kind{}, commands.Fail(commands.InvalidRequest) }
	if !validRequestID(spec.ID) || spec.Mode != "erp_issuance" || spec.Profile != "synthetic_erp_document_v1" || len(spec.Region) != 2 || spec.Region[0] < 'A' || spec.Region[0] > 'Z' || spec.Region[1] < 'A' || spec.Region[1] > 'Z' || !validRequestID(spec.Tax) || len(spec.Zone) > 128 || spec.Zone == "" || spec.Zone == "Local" || len(spec.Languages) < 1 || len(spec.Languages) > 3 {
		return bad()
	}
	if _, err := time.LoadLocation(spec.Zone); err != nil {
		return bad()
	}
	seen := map[string]bool{}
	for _, language := range spec.Languages {
		if seen[language] || (language != "en" && language != "ko" && language != "ar") {
			return bad()
		}
		seen[language] = true
	}
	identity, err := financialEvidenceDigest("issuance_kind", spec.ID)
	if err != nil {
		return issuance.Kind{}, err
	}
	languages := slices.Clone(spec.Languages)
	slices.Sort(languages)
	return issuance.Kind{ID: "issued-kind-" + identity, Class: issuance.ClassERP, Region: spec.Region, Tax: spec.Tax, Zone: spec.Zone, Languages: languages}, nil
}

func copyIssuanceProfile(p financialIssuanceProfile) financialIssuanceProfile {
	p.Spec.Languages = slices.Clone(p.Spec.Languages)
	p.Kind.Languages = slices.Clone(p.Kind.Languages)
	p.Source.Basis.Subject.Digests = maps.Clone(p.Source.Basis.Subject.Digests)
	p.Source.Basis.Subject.Entries = slices.Clone(p.Source.Basis.Subject.Entries)
	p.Source.Basis.Decision.Decision.Exclude = slices.Clone(p.Source.Basis.Decision.Decision.Exclude)
	p.Source.Basis.Contract.Components = slices.Clone(p.Source.Basis.Contract.Components)
	p.Source.Basis.Rows = slices.Clone(p.Source.Basis.Rows)
	p.Source.Basis.Contributors = slices.Clone(p.Source.Basis.Contributors)
	return p
}

func (a *financialIssuance) profileCaller(workspace, actor string) error {
	if a == nil || a.server == nil || a.server.cfg.Seed != SeedSample {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if a.server.held() {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	if err := a.server.governorCaller(workspace, actor); err != nil {
		return err
	}
	_, err := a.server.governor(workspace, actor)
	return err
}

func (a *financialIssuance) currentProfileSource(workspace string, p financialIssuanceProfile, actor string) error {
	immutable := copyIssuanceProfile(p)
	immutable.Digest = ""
	immutable.State = ""
	immutable.Confirmer = ""
	expected, err := financialEvidenceDigest("issuance_profile", immutable)
	if err != nil || expected != p.Digest || a.scopes[p.Source.Basis.Subject.Target] != p.Spec.ID {
		return commands.Fail(commands.EvidenceStale)
	}
	if _, err := a.server.governor(workspace, p.Proposer); err != nil {
		return err
	}
	if p.Confirmer != "" {
		if _, err := a.server.governor(workspace, p.Confirmer); err != nil {
			return err
		}
	}
	source, err := a.server.financialIssuanceSourceLocked(workspace, p.Source.Basis.FinalRequest, p.Source.Basis.FinalDigest, actor)
	if err != nil {
		return err
	}
	if source.Digest != p.Source.Digest || !reflect.DeepEqual(source, p.Source) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialIssuance) proposeProfile(workspace, request, digest, actor, reason string, spec financialIssuanceProfileSpec) (financialIssuanceProfile, error) {
	if err := a.profileCaller(workspace, actor); err != nil {
		return financialIssuanceProfile{}, err
	}
	kind, err := issuanceProfileKind(spec)
	if err != nil || !documentReason(reason) {
		return financialIssuanceProfile{}, commands.Fail(commands.InvalidRequest)
	}
	spec.Languages = slices.Clone(kind.Languages)
	source, err := a.server.financialIssuanceSourceLocked(workspace, request, digest, actor)
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	p := financialIssuanceProfile{Spec: spec, Source: source, Kind: kind, Proposer: actor, Reason: reason}
	p.Digest, err = financialEvidenceDigest("issuance_profile", p)
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	if old, ok := a.profiles[spec.ID]; ok {
		if old.Proposer != actor || old.Digest != p.Digest {
			return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentProfileSource(workspace, old, actor); err != nil {
			return financialIssuanceProfile{}, err
		}
		if old.State != "registration_unknown" {
			return a.profileReadback(workspace, old)
		}
		p = old
	} else {
		if len(a.profiles) >= financialIssuanceLimit || a.scopes[source.Basis.Subject.Target] != "" {
			return financialIssuanceProfile{}, commands.Fail(commands.PrerequisiteFailed)
		}
		if _, err := a.documents.KindOf(workspace, kind.ID); !errors.Is(err, issuance.ErrNotFound) {
			return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-profile-before-register") {
			return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State = "registration_unknown"
		a.profiles[spec.ID] = copyIssuanceProfile(p)
		a.scopes[source.Basis.Subject.Target] = spec.ID
	}
	view, err := a.documents.Define(workspace, p.Kind, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	if view.State != issuance.StateProposed || view.Definer != actor || !reflect.DeepEqual(view.Kind, p.Kind) {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-profile-after-register") {
		return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "proposed"
	a.profiles[spec.ID] = copyIssuanceProfile(p)
	if a.server.faulty("issuance-profile-after-proposal-publication") {
		return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyIssuanceProfile(p), nil
}

func (a *financialIssuance) profileReadback(workspace string, p financialIssuanceProfile) (financialIssuanceProfile, error) {
	v, err := a.documents.KindOf(workspace, p.Kind.ID)
	if err != nil || !reflect.DeepEqual(v.Kind, p.Kind) || v.Definer != p.Proposer {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State == "proposed" && (v.State != issuance.StateProposed || v.Approver != "") || p.State == "confirmed" && (v.State != issuance.StateActive || v.Approver != p.Confirmer) {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State != "proposed" && p.State != "confirmed" {
		return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyIssuanceProfile(p), nil
}

func (a *financialIssuance) confirmProfile(workspace, id, digest, actor string) (financialIssuanceProfile, error) {
	if err := a.profileCaller(workspace, actor); err != nil {
		return financialIssuanceProfile{}, err
	}
	p, ok := a.profiles[id]
	if !ok || p.Source.Basis.Workspace != workspace || p.Digest != digest {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if p.Proposer == actor {
		return financialIssuanceProfile{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Confirmer != "" && p.Confirmer != actor {
		return financialIssuanceProfile{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentProfileSource(workspace, p, actor); err != nil {
		return financialIssuanceProfile{}, err
	}
	if p.State == "confirmed" {
		return a.profileReadback(workspace, p)
	}
	if p.State != "proposed" && p.State != "confirmation_unknown" {
		return financialIssuanceProfile{}, commands.Fail(commands.PrerequisiteFailed)
	}
	v, err := a.documents.KindOf(workspace, p.Kind.ID)
	if err != nil || !reflect.DeepEqual(v.Kind, p.Kind) || v.Definer != p.Proposer || v.State != issuance.StateProposed && (p.State != "confirmation_unknown" || v.State != issuance.StateActive || v.Approver != actor) {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-profile-before-confirm") {
		return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "confirmation_unknown"
	p.Confirmer = actor
	a.profiles[id] = copyIssuanceProfile(p)
	v, err = a.documents.ApproveKind(workspace, p.Kind.ID, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	if v.State != issuance.StateActive || v.Approver != actor {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-profile-after-confirm") {
		return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "confirmed"
	a.profiles[id] = copyIssuanceProfile(p)
	if a.server.faulty("issuance-profile-after-confirmation-publication") {
		return financialIssuanceProfile{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyIssuanceProfile(p), nil
}
