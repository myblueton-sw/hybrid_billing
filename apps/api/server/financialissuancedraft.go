package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"reflect"
	"slices"
)

type financialDraftSpec struct {
	ID, Profile, ProfileDigest, Language string
	Application                          *financialDraftApplicationRequest `json:"application,omitempty"`
}
type financialDraftRecord struct {
	Workspace, Submitter, FinalRequester, Digest, State string
	Spec                                                financialDraftSpec
	Profile                                             financialIssuanceProfile
	Application                                         *financialIssuanceApplication `json:"application,omitempty"`
	Draft                                               issuance.Draft
	Module                                              issuance.DraftView
}

func copyModuleDraft(d issuance.Draft) issuance.Draft {
	d.Lines = slices.Clone(d.Lines)
	if d.Detail != nil {
		v := *d.Detail
		d.Detail = &v
	}
	return d
}
func copyModuleDraftView(v issuance.DraftView) issuance.DraftView {
	v.Draft = copyModuleDraft(v.Draft)
	v.References = slices.Clone(v.References)
	if v.Coverage != nil {
		c := *v.Coverage
		v.Coverage = &c
	}
	return v
}
func copyFinancialDraft(p financialDraftRecord) financialDraftRecord {
	p.Profile = copyIssuanceProfile(p.Profile)
	if p.Spec.Application != nil {
		v := *p.Spec.Application
		p.Spec.Application = &v
	}
	if p.Application != nil {
		v := copyIssuanceApplication(*p.Application)
		p.Application = &v
	}
	p.Draft = copyModuleDraft(p.Draft)
	p.Module = copyModuleDraftView(p.Module)
	return p
}
func draftIntentDigest(p financialDraftRecord) (string, error) {
	if _, err := draftOriginSource(p); err != nil {
		return "", err
	}
	domain := "issuance_draft_intent"
	if p.Application != nil {
		domain = "issuance_application_draft_v1"
	}
	p.Digest, p.State = "", ""
	p.Module = issuance.DraftView{}
	return financialEvidenceDigest(domain, p)
}
func unapprovedDraftMatches(p financialDraftRecord, v issuance.DraftView) bool {
	return reflect.DeepEqual(v.Draft, p.Draft) && v.Revision == 1 && v.Digest != "" && v.Submitter == p.Submitter && v.Approver == "" && len(v.References) == 0 && v.Coverage == nil && v.Number == "" && !v.At.IsZero()
}
func (a *financialIssuance) draftModule(p financialDraftRecord) (issuance.DraftView, bool, error) {
	if a.server.faulty("issuance-draft-lookup") {
		return issuance.DraftView{}, false, commands.Fail(commands.DependencyUnavailable)
	}
	v, err := a.documents.DraftOf(p.Workspace, p.Spec.ID)
	if errors.Is(err, issuance.ErrNotFound) {
		return issuance.DraftView{}, false, nil
	}
	if err != nil {
		return issuance.DraftView{}, false, err
	}
	return v, true, nil
}
func (a *financialIssuance) submitDraft(workspace, actor string, spec financialDraftSpec) (financialDraftRecord, error) {
	return a.submitDraftTracked(workspace, actor, spec, nil, nil)
}

func (a *financialIssuance) submitDraftTracked(workspace, actor string, spec financialDraftSpec, effect *financialDraftEffect, admit func(financialDraftRecord) error) (financialDraftRecord, error) {
	p, err := a.currentDraftOrigin(workspace, actor, draftauthority.Submit, spec)
	if err != nil {
		return financialDraftRecord{}, err
	}
	source, err := draftOriginSource(p)
	if err != nil {
		return financialDraftRecord{}, err
	}
	var draft issuance.Draft
	if p.Application != nil {
		draft, err = issuanceDraftFromApplication(*p.Application, spec.ID, spec.Language)
	} else {
		draft, err = issuanceDraftFromSource(p.Profile, spec.ID, spec.Language)
	}
	if err != nil {
		return financialDraftRecord{}, err
	}
	final, _, err := a.server.financialFinalRecord(workspace, source.Basis.FinalRequest)
	if err != nil {
		return financialDraftRecord{}, err
	}
	if final.Digest != source.Basis.FinalDigest || final.Requester == "" {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Submitter, p.FinalRequester, p.Draft = actor, final.Requester, draft
	p.Digest, err = draftIntentDigest(p)
	if err != nil {
		return financialDraftRecord{}, err
	}
	target := source.Basis.Subject.Target
	old, exists := a.drafts[spec.ID]
	if exists {
		actualDigest, e := draftIntentDigest(old)
		if e != nil || actualDigest != old.Digest || old.Digest != p.Digest || !reflect.DeepEqual(old.Spec, spec) || old.Submitter != actor || old.Workspace != workspace || a.draftScopes[target] != spec.ID {
			return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if old.State != "submitted" && old.State != "submission_unknown" && old.State != "submission_refused" {
			return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
		}
		p = copyFinancialDraft(old)
	} else {
		if len(a.drafts) >= financialIssuanceLimit || a.draftScopes[target] != "" {
			return financialDraftRecord{}, commands.Fail(commands.PrerequisiteFailed)
		}
	}
	if err := a.draftScopeIntegrity(p, exists); err != nil {
		return financialDraftRecord{}, err
	}
	if admit != nil {
		if err := admit(p); err != nil {
			return financialDraftRecord{}, err
		}
	}
	if exists && p.State == "submitted" {
		owned, err := a.ownedDraft(workspace, spec.ID, p.Digest)
		if err != nil {
			return financialDraftRecord{}, err
		}
		if _, err := a.draftPhaseReadback(owned); err != nil {
			return financialDraftRecord{}, err
		}
		return copyFinancialDraft(p), nil
	}
	v, found, err := a.draftModule(p)
	if err != nil {
		return financialDraftRecord{}, err
	}
	if found {
		if !exists || p.Module.Digest == "" || !unapprovedDraftMatches(p, v) || !reflect.DeepEqual(v, p.Module) {
			return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
		}
		p.State = "submitted"
		a.storeDraft(p, effect, true)
		return copyFinancialDraft(p), nil
	}
	if p.Module.Digest != "" || p.State == "submitted" {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-draft-before-submit") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "submission_unknown"
	a.storeDraft(p, effect, false)
	if a.draftScopes[target] != spec.ID {
		a.draftScopes[target] = spec.ID
		if effect != nil {
			effect.retainedChanged = true
		}
	}
	if a.server.faulty("issuance-draft-after-intent") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	effect.attempt()
	v, err = a.documents.Submit(workspace, draft, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
	if err != nil {
		p.State = "submission_refused"
		a.storeDraft(p, effect, false)
		return financialDraftRecord{}, err
	}
	p.Module = copyModuleDraftView(v)
	a.storeDraft(p, effect, true)
	if !unapprovedDraftMatches(p, v) {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-draft-after-submit") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "submitted"
	a.storeDraft(p, effect, true)
	if a.server.faulty("issuance-draft-after-submission-publication") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialDraft(p), nil
}
