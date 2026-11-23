package server

import (
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

type financialDraftApplicationRequest struct {
	Schema                    string
	Configuration             financialConfigurationReference
	FinalRequest, FinalDigest string
}

func validDraftOriginRequest(s financialDraftSpec) bool {
	if s.Application == nil {
		return validRequestID(s.Profile) && validPlanHash(s.ProfileDigest)
	}
	r := s.Application
	return s.Profile == "" && s.ProfileDigest == "" && r.Schema == "hb_issuance_application_request_v1" && validConfigurationReference(&r.Configuration) && validRequestID(r.FinalRequest) && validPlanHash(r.FinalDigest)
}

func draftOriginSource(p financialDraftRecord) (financialIssuanceSourceView, error) {
	bad := func() (financialIssuanceSourceView, error) {
		return financialIssuanceSourceView{}, commands.Fail(commands.EvidenceStale)
	}
	if !validDraftOriginRequest(p.Spec) {
		return bad()
	}
	if p.Spec.Application != nil {
		if p.Application == nil || !reflect.DeepEqual(p.Profile, financialIssuanceProfile{}) {
			return bad()
		}
		r, v := p.Spec.Application, *p.Application
		if err := applicationApplicability(v); err != nil {
			return bad()
		}
		digest, err := applicationDigest(v)
		if err != nil || digest != v.Digest || r.Configuration != configurationReference(v.Configuration) || r.FinalRequest != v.Source.Basis.FinalRequest || r.FinalDigest != v.Source.Basis.FinalDigest || p.Workspace != v.Source.Basis.Workspace {
			return bad()
		}
		return v.Source, nil
	}
	v := copyIssuanceProfile(p.Profile)
	if p.Application != nil || v.Spec.ID != p.Spec.Profile || v.Digest != p.Spec.ProfileDigest || v.State != "confirmed" || v.Source.Basis.Workspace != p.Workspace {
		return bad()
	}
	v.Digest, v.State, v.Confirmer = "", "", ""
	digest, err := financialEvidenceDigest("issuance_profile", v)
	if err != nil || digest != p.Profile.Digest {
		return bad()
	}
	digest, err = financialEvidenceDigest("issuance_source", v.Source.Basis)
	if err != nil || digest != v.Source.Digest {
		return bad()
	}
	return v.Source, nil
}

func (a *financialIssuance) currentDraftOrigin(workspace, actor, capability string, spec financialDraftSpec) (financialDraftRecord, error) {
	if err := a.draftOperation(workspace, actor, capability); err != nil {
		return financialDraftRecord{}, err
	}
	if !validDraftOriginRequest(spec) {
		return financialDraftRecord{}, commands.Fail(commands.InvalidRequest)
	}
	p := financialDraftRecord{Workspace: workspace, Spec: spec}
	if spec.Application == nil {
		v, err := a.draftProfile(workspace, spec.Profile, spec.ProfileDigest, actor, capability)
		p.Profile = v
		return p, err
	}
	r := spec.Application
	v, err := a.draftApplication(workspace, actor, capability, r.Configuration.ID, r.Configuration.Digest, r.FinalRequest, r.FinalDigest)
	p.Application = &v
	return p, err
}

func (a *financialIssuance) draftScopeIntegrity(p financialDraftRecord, requireReverse bool) error {
	source, err := draftOriginSource(p)
	if err != nil {
		return err
	}
	target := source.Basis.Subject.Target
	if target == "" || requireReverse && a.draftScopes[target] != p.Spec.ID {
		return commands.Fail(commands.EvidenceStale)
	}
	for key, id := range a.draftScopes {
		if id == p.Spec.ID && key != target {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	for id, owned := range a.drafts {
		s, err := draftOriginSource(owned)
		digest, digestErr := draftIntentDigest(owned)
		if err != nil || digestErr != nil || digest != owned.Digest || !validRequestID(owned.Workspace) || s.Basis.Subject.Target == "" || id != owned.Spec.ID || id != p.Spec.ID && owned.Workspace == p.Workspace && s.Basis.Subject.Target == target {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	return nil
}

func (a *financialIssuance) draftOriginReferences(p financialDraftRecord, at time.Time) ([]issuance.Reference, error) {
	if _, err := draftOriginSource(p); err != nil {
		return nil, err
	}
	if p.Application != nil {
		return a.configuredReferences(p.Application.Configuration, at)
	}
	return a.draftReferences(p.Profile, at)
}

func (a *financialIssuance) draftOriginCoverage(p financialDraftRecord, id, digest string, at time.Time) (receivables.Coverage, error) {
	if _, err := draftOriginSource(p); err != nil {
		return receivables.Coverage{}, err
	}
	if p.Application != nil {
		return a.configuredCoverage(p.Application.Configuration, id, digest, at)
	}
	return a.draftCoverage(p.Profile, id, digest, at)
}
