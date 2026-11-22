package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"reflect"
	"slices"
)

type financialDraftSpec struct{ ID, Profile, ProfileDigest, Language string }
type financialDraftRecord struct {
	Workspace, Submitter, FinalRequester, Digest, State string
	Spec                                                financialDraftSpec
	Profile                                             financialIssuanceProfile
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
	p.Draft = copyModuleDraft(p.Draft)
	p.Module = copyModuleDraftView(p.Module)
	return p
}
func draftIntentDigest(p financialDraftRecord) (string, error) {
	p.Digest, p.State = "", ""
	p.Module = issuance.DraftView{}
	return financialEvidenceDigest("issuance_draft_intent", p)
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
	profile, err := a.draftProfile(workspace, spec.Profile, spec.ProfileDigest, actor, draftauthority.Submit)
	if err != nil {
		return financialDraftRecord{}, err
	}
	draft, err := issuanceDraftFromSource(profile, spec.ID, spec.Language)
	if err != nil {
		return financialDraftRecord{}, err
	}
	final, _, err := a.server.financialFinalRecord(workspace, profile.Source.Basis.FinalRequest)
	if err != nil {
		return financialDraftRecord{}, err
	}
	if final.Digest != profile.Source.Basis.FinalDigest || final.Requester == "" {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p := financialDraftRecord{Workspace: workspace, Submitter: actor, FinalRequester: final.Requester, Spec: spec, Profile: profile, Draft: draft}
	p.Digest, err = draftIntentDigest(p)
	if err != nil {
		return financialDraftRecord{}, err
	}
	target := profile.Source.Basis.Subject.Target
	old, exists := a.drafts[spec.ID]
	if exists {
		actualDigest, e := draftIntentDigest(old)
		if e != nil || actualDigest != old.Digest || old.Digest != p.Digest || old.Spec != spec || old.Submitter != actor || old.Workspace != workspace || a.draftScopes[target] != spec.ID {
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
	for id, owned := range a.drafts {
		if owned.Workspace == workspace && owned.Profile.Source.Basis.Subject.Target == target && id != spec.ID {
			return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
		}
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
		a.drafts[spec.ID] = copyFinancialDraft(p)
		return copyFinancialDraft(p), nil
	}
	if p.Module.Digest != "" || p.State == "submitted" {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-draft-before-submit") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "submission_unknown"
	a.drafts[spec.ID] = copyFinancialDraft(p)
	a.draftScopes[target] = spec.ID
	if a.server.faulty("issuance-draft-after-intent") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	v, err = a.documents.Submit(workspace, draft, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
	if err != nil {
		p.State = "submission_refused"
		a.drafts[spec.ID] = copyFinancialDraft(p)
		return financialDraftRecord{}, err
	}
	p.Module = copyModuleDraftView(v)
	a.drafts[spec.ID] = copyFinancialDraft(p)
	if !unapprovedDraftMatches(p, v) {
		return financialDraftRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-draft-after-submit") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "submitted"
	a.drafts[spec.ID] = copyFinancialDraft(p)
	if a.server.faulty("issuance-draft-after-submission-publication") {
		return financialDraftRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialDraft(p), nil
}
