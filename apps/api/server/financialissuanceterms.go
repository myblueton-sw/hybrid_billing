package server

import (
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type financialTermSpec struct {
	ID, Profile, ProfileDigest, Mode, Reason string
	Revision, NetDays                        int
}
type financialTermRecord struct {
	Spec                            financialTermSpec
	Profile                         financialIssuanceProfile
	Term                            receivables.Term
	Module                          receivables.TermView
	Author, Approver, Digest, State string
}

func copyFinancialTerm(p financialTermRecord) financialTermRecord {
	p.Profile = copyIssuanceProfile(p.Profile)
	return p
}
func termIntentDigest(p financialTermRecord) (string, error) {
	return financialEvidenceDigest("issuance_term_intent", struct {
		Spec    financialTermSpec
		Profile financialIssuanceProfile
		Term    receivables.Term
		Author  string
	}{p.Spec, p.Profile, p.Term, p.Author})
}
func (a *financialIssuance) currentTerm(workspace string, p financialTermRecord, actor, capability string) error {
	digest, err := termIntentDigest(p)
	if err != nil || digest != p.Digest || a.termScopes[p.Profile.Source.Basis.Subject.Target] != p.Spec.ID {
		return commands.Fail(commands.EvidenceStale)
	}
	profile, err := a.termProfile(workspace, p.Spec.Profile, p.Spec.ProfileDigest, actor, capability)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(profile, p.Profile) {
		return commands.Fail(commands.EvidenceStale)
	}
	if err := a.termActor(workspace, p.Author, termsauthority.RecordTerm); err != nil {
		return err
	}
	if p.Approver != "" {
		if err := a.termActor(workspace, p.Approver, termsauthority.ApproveTerm); err != nil {
			return err
		}
	}
	return nil
}
func (a *financialIssuance) termReadback(workspace string, p financialTermRecord) (financialTermRecord, error) {
	v, err := a.terms.Term(workspace, p.Term.ID, p.Term.Revision)
	if err != nil || v != p.Module || v.Term != p.Term || v.Author != p.Author || v.Digest == "" {
		return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State == "pending" && (v.State != receivables.StatePending || v.Approver != "") || p.State == "approved" && (v.State != receivables.StateApproved || v.Approver != p.Approver) {
		return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State != "pending" && p.State != "approved" {
		return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialTerm(p), nil
}
func (a *financialIssuance) recordTerm(workspace, actor string, spec financialTermSpec) (financialTermRecord, error) {
	if err := a.termCaller(workspace, actor, termsauthority.RecordTerm); err != nil {
		return financialTermRecord{}, err
	}
	if !validRequestID(spec.ID) || spec.Mode != receivables.ModeCalendar || spec.Revision < 1 || spec.Revision > 1000000 || spec.NetDays < 0 || spec.NetDays > 365 || !documentReason(spec.Reason) {
		return financialTermRecord{}, commands.Fail(commands.InvalidRequest)
	}
	profile, err := a.termProfile(workspace, spec.Profile, spec.ProfileDigest, actor, termsauthority.RecordTerm)
	if err != nil {
		return financialTermRecord{}, err
	}
	id, err := financialEvidenceDigest("issuance_term", spec.ID)
	if err != nil {
		return financialTermRecord{}, err
	}
	p := financialTermRecord{Spec: spec, Profile: profile, Term: receivables.Term{ID: "issued-term-" + id, Revision: spec.Revision, NetDays: spec.NetDays, Mode: spec.Mode}, Author: actor}
	p.Digest, err = termIntentDigest(p)
	if err != nil {
		return financialTermRecord{}, err
	}
	if old, ok := a.termIntents[spec.ID]; ok {
		if old.Author != actor || old.Digest != p.Digest {
			return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentTerm(workspace, old, actor, termsauthority.RecordTerm); err != nil {
			return financialTermRecord{}, err
		}
		if old.State != "registration_unknown" {
			return a.termReadback(workspace, old)
		}
		p = old
	} else {
		if len(a.termIntents) >= financialIssuanceLimit || a.termScopes[profile.Source.Basis.Subject.Target] != "" {
			return financialTermRecord{}, commands.Fail(commands.PrerequisiteFailed)
		}
		if _, err := a.terms.Term(workspace, p.Term.ID, p.Term.Revision); !errors.Is(err, receivables.ErrNotFound) {
			return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-term-before-register") {
			return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State = "registration_unknown"
		a.termIntents[spec.ID] = copyFinancialTerm(p)
		a.termScopes[profile.Source.Basis.Subject.Target] = spec.ID
	}
	v, err := a.terms.RecordTerm(workspace, p.Term, actor)
	if err != nil {
		return financialTermRecord{}, err
	}
	if v.Term != p.Term || v.Author != actor || v.State != receivables.StatePending || v.Approver != "" || p.Module.Digest != "" && v != p.Module {
		return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = v
	a.termIntents[spec.ID] = copyFinancialTerm(p)
	if a.server.faulty("issuance-term-after-register") {
		return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "pending"
	a.termIntents[spec.ID] = copyFinancialTerm(p)
	if a.server.faulty("issuance-term-after-register-publication") {
		return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialTerm(p), nil
}
func (a *financialIssuance) approveTerm(workspace, id, digest, actor string) (financialTermRecord, error) {
	if err := a.termCaller(workspace, actor, termsauthority.ApproveTerm); err != nil {
		return financialTermRecord{}, err
	}
	p, ok := a.termIntents[id]
	if !ok || p.Digest != digest {
		return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if p.Author == actor {
		return financialTermRecord{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Approver != "" && p.Approver != actor {
		return financialTermRecord{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentTerm(workspace, p, actor, termsauthority.ApproveTerm); err != nil {
		return financialTermRecord{}, err
	}
	if p.State == "approved" {
		return a.termReadback(workspace, p)
	}
	if p.State != "pending" && p.State != "approval_unknown" {
		return financialTermRecord{}, commands.Fail(commands.PrerequisiteFailed)
	}
	v, err := a.terms.Term(workspace, p.Term.ID, p.Term.Revision)
	if err != nil || v != p.Module || v.Term != p.Term || v.Author != p.Author || v.State != receivables.StatePending && (p.State != "approval_unknown" || v.State != receivables.StateApproved || v.Approver != actor) {
		return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-term-before-approve") {
		return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "approval_unknown"
	p.Approver = actor
	a.termIntents[id] = copyFinancialTerm(p)
	v, err = a.terms.ApproveTerm(workspace, p.Term.ID, p.Term.Revision, actor)
	if err != nil {
		return financialTermRecord{}, err
	}
	if v.Term != p.Term || v.Author != p.Author || v.Approver != actor || v.State != receivables.StateApproved || v.Digest != p.Module.Digest || v.Written != p.Module.Written {
		return financialTermRecord{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = v
	a.termIntents[id] = copyFinancialTerm(p)
	if a.server.faulty("issuance-term-after-approve") {
		return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "approved"
	a.termIntents[id] = copyFinancialTerm(p)
	if a.server.faulty("issuance-term-after-approve-publication") {
		return financialTermRecord{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialTerm(p), nil
}
