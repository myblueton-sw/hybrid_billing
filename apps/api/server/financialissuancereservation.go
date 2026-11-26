package server

import (
	"errors"
	"reflect"
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

type financialKindReservation struct {
	Workspace, Proposer, Confirmer, Reason, Digest, State string
	ProfileID, ProfileDigest                              string
	Spec                                                  financialIssuanceProfileSpec
	Kind                                                  issuance.Kind
}

func copyKindReservation(r financialKindReservation) financialKindReservation {
	r.Spec.Languages = slices.Clone(r.Spec.Languages)
	r.Kind.Languages = slices.Clone(r.Kind.Languages)
	return r
}

func kindReservationDigest(r financialKindReservation) (string, error) {
	r.Confirmer, r.Digest, r.State, r.ProfileID, r.ProfileDigest = "", "", "", "", ""
	return financialEvidenceDigest("issuance_kind_reservation", r)
}

func (a *financialIssuance) reservationKind(workspace, id string) (issuance.KindView, error) {
	if a.server.faulty("issuance-reservation-kind-lookup") {
		return issuance.KindView{}, commands.Fail(commands.DependencyUnavailable)
	}
	return a.documents.KindOf(workspace, id)
}

func (a *financialIssuance) currentKindReservation(workspace string, r financialKindReservation) error {
	digest, err := kindReservationDigest(r)
	kind, shapeErr := issuanceProfileKind(r.Spec)
	if err != nil || shapeErr != nil || digest != r.Digest || r.Workspace != workspace || !reflect.DeepEqual(kind, r.Kind) || !reflect.DeepEqual(r.Spec.Languages, kind.Languages) || r.Proposer == "" || !documentReason(r.Reason) {
		return commands.Fail(commands.EvidenceStale)
	}
	if r.State != "proposed" && r.State != "confirmed" || r.State == "proposed" && r.Confirmer != "" || r.State == "confirmed" && (r.Confirmer == "" || r.Confirmer == r.Proposer) {
		return commands.Fail(commands.EvidenceStale)
	}
	for _, governor := range []string{r.Proposer, r.Confirmer} {
		if governor != "" {
			if err := a.server.issuanceProfileGovernorStanding(workspace, governor); err != nil {
				return err
			}
		}
	}
	v, lookupErr := a.reservationKind(workspace, r.Kind.ID)
	if lookupErr != nil && !errors.Is(lookupErr, issuance.ErrNotFound) {
		return lookupErr
	}
	p, exists := a.profiles[r.Spec.ID]
	if r.ProfileID == "" {
		if r.ProfileDigest != "" || exists || !errors.Is(lookupErr, issuance.ErrNotFound) {
			return commands.Fail(commands.EvidenceStale)
		}
		return nil
	}
	if r.State != "confirmed" || !exists || r.ProfileID != r.Spec.ID || p.Spec.ID != r.ProfileID || p.Digest != r.ProfileDigest || p.ReservationDigest != r.Digest || p.Source.Basis.Workspace != workspace || !reflect.DeepEqual(p.Spec, r.Spec) || !reflect.DeepEqual(p.Kind, r.Kind) || a.scopes[p.Source.Basis.Subject.Target] != p.Spec.ID {
		return commands.Fail(commands.EvidenceStale)
	}
	immutable := copyIssuanceProfile(p)
	immutable.Digest, immutable.State, immutable.Confirmer = "", "", ""
	expected, err := financialEvidenceDigest("issuance_profile", immutable)
	if err != nil || expected != p.Digest {
		return commands.Fail(commands.EvidenceStale)
	}
	if p.State == "registration_unknown" && errors.Is(lookupErr, issuance.ErrNotFound) && p.Confirmer == "" {
		return nil
	}
	if lookupErr != nil || !reflect.DeepEqual(v.Kind, p.Kind) || v.Definer != p.Proposer {
		return commands.Fail(commands.EvidenceStale)
	}
	switch p.State {
	case "registration_unknown", "proposed":
		if p.Confirmer == "" && v.State == issuance.StateProposed && v.Approver == "" {
			return nil
		}
	case "confirmation_unknown":
		if p.Confirmer != "" && p.Confirmer != p.Proposer && (v.State == issuance.StateProposed && v.Approver == "" || v.State == issuance.StateActive && v.Approver == p.Confirmer) {
			return nil
		}
	case "confirmed":
		if p.Confirmer != "" && p.Confirmer != p.Proposer && v.State == issuance.StateActive && v.Approver == p.Confirmer {
			return nil
		}
	}
	return commands.Fail(commands.EvidenceStale)
}

func (a *financialIssuance) profileReservation(workspace string, p financialIssuanceProfile) error {
	r, ok := a.kindReservations[p.Spec.ID]
	if !ok || r.ProfileID != p.Spec.ID || r.ProfileDigest != p.Digest || r.Digest != p.ReservationDigest {
		return commands.Fail(commands.EvidenceStale)
	}
	return a.currentKindReservation(workspace, r)
}

func (a *financialIssuance) proposeKindReservation(workspace, actor, reason string, spec financialIssuanceProfileSpec) (financialKindReservation, error) {
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return financialKindReservation{}, err
	}
	kind, err := issuanceProfileKind(spec)
	if err != nil || !documentReason(reason) {
		return financialKindReservation{}, commands.Fail(commands.InvalidRequest)
	}
	spec.Languages = slices.Clone(kind.Languages)
	r := financialKindReservation{Workspace: workspace, Proposer: actor, Reason: reason, Spec: spec, Kind: kind}
	r.Digest, err = kindReservationDigest(r)
	if err != nil {
		return financialKindReservation{}, err
	}
	if old, ok := a.kindReservations[spec.ID]; ok {
		if old.Digest != r.Digest || old.Proposer != actor {
			return financialKindReservation{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentKindReservation(workspace, old); err != nil {
			return financialKindReservation{}, err
		}
		return copyKindReservation(old), nil
	}
	if len(a.kindReservations) >= financialIssuanceLimit {
		return financialKindReservation{}, commands.Fail(commands.PrerequisiteFailed)
	}
	r.State = "proposed"
	if err := a.currentKindReservation(workspace, r); err != nil {
		return financialKindReservation{}, err
	}
	if a.server.faulty("issuance-reservation-before-propose") {
		return financialKindReservation{}, commands.Fail(commands.DependencyUnavailable)
	}
	a.kindReservations[spec.ID] = copyKindReservation(r)
	if a.server.faulty("issuance-reservation-after-propose") {
		return financialKindReservation{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyKindReservation(r), nil
}

func (a *financialIssuance) confirmKindReservation(workspace, id, digest, actor string) (financialKindReservation, error) {
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return financialKindReservation{}, err
	}
	r, ok := a.kindReservations[id]
	if !ok || r.Spec.ID != id || r.Digest != digest {
		return financialKindReservation{}, commands.Fail(commands.EvidenceStale)
	}
	if r.Proposer == actor {
		return financialKindReservation{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if r.Confirmer != "" && r.Confirmer != actor {
		return financialKindReservation{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentKindReservation(workspace, r); err != nil {
		return financialKindReservation{}, err
	}
	if r.State == "confirmed" {
		return copyKindReservation(r), nil
	}
	if a.server.faulty("issuance-reservation-before-confirm") {
		return financialKindReservation{}, commands.Fail(commands.DependencyUnavailable)
	}
	r.Confirmer, r.State = actor, "confirmed"
	a.kindReservations[id] = copyKindReservation(r)
	if a.server.faulty("issuance-reservation-after-confirm") {
		return financialKindReservation{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyKindReservation(r), nil
}
