package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"reflect"
)

func (a *financialIssuance) ownedConfirmedProfile(workspace, id, digest string) (financialIssuanceProfile, error) {
	p, ok := a.profiles[id]
	if !ok || p.Spec.ID != id || p.Source.Basis.Workspace != workspace || p.Digest != digest || p.State != "confirmed" || p.Proposer == "" || p.Confirmer == "" || p.Proposer == p.Confirmer {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	sourceDigest, err := financialEvidenceDigest("issuance_source", p.Source.Basis)
	if err != nil || sourceDigest != p.Source.Digest {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.profileReservation(workspace, p); err != nil {
		return financialIssuanceProfile{}, err
	}
	immutable := copyIssuanceProfile(p)
	immutable.Digest, immutable.State, immutable.Confirmer = "", "", ""
	expected, err := financialEvidenceDigest("issuance_profile", immutable)
	if err != nil || expected != p.Digest || a.scopes[p.Source.Basis.Subject.Target] != id {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	for _, governor := range []string{p.Proposer, p.Confirmer} {
		if err := a.server.issuanceProfileGovernorStanding(workspace, governor); err != nil {
			return financialIssuanceProfile{}, err
		}
	}
	return a.profileReadback(workspace, p)
}

func (a *financialIssuance) profileCurrentFinancialSource(workspace, actor string, p financialIssuanceProfile) (financialIssuanceProfile, error) {
	source, err := a.server.financialIssuanceSourceLocked(workspace, p.Source.Basis.FinalRequest, p.Source.Basis.FinalDigest, actor)
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	if !reflect.DeepEqual(source, p.Source) {
		return financialIssuanceProfile{}, commands.Fail(commands.EvidenceStale)
	}
	return copyIssuanceProfile(p), nil
}

func (a *financialIssuance) draftProfile(workspace, id, digest, actor, capability string) (financialIssuanceProfile, error) {
	if err := a.draftOperation(workspace, actor, capability); err != nil {
		return financialIssuanceProfile{}, err
	}
	p, err := a.ownedConfirmedProfile(workspace, id, digest)
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	return a.profileCurrentFinancialSource(workspace, actor, p)
}
