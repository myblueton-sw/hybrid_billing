package server

import "github.com/myblueton-sw/hybrid_billing/modules/commands"

func setupSampleOption(spec financialConfigurationSpec) bool {
	return spec.Mode == "erp_issuance" && spec.Profile == "synthetic_erp_document_v1" && spec.Region == "US" && spec.Tax == "synthetic-tax" && spec.Zone == "UTC"
}

func (a *financialIssuance) setupOptionFields(workspace, actor, contract, on, digest string) (map[string]string, error) {
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	if !validPlanHash(digest) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	context, err := a.setupContextFields(workspace, actor, contract, on)
	if err != nil {
		return nil, err
	}
	if context["scope_digest"] != digest {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	fields := map[string]string{
		"schema": "hb_issuance_setup_options_v1", "contract": contract, "on": on, "scope_digest": digest,
		"current_eligibility": "unchecked", "option": "synthetic_erp_sample_v1", "applicability": "synthetic_only",
		"mode": "erp_issuance", "profile": "synthetic_erp_document_v1", "region": "US", "tax": "synthetic-tax", "zone": "UTC", "languages": "ar,en,ko",
	}
	if err := a.boundedDraftFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
