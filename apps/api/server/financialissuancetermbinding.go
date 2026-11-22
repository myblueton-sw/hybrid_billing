package server

import (
	"errors"
	"reflect"
	"slices"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type financialTermBindingSpec struct{ ID, Term, TermDigest, From, To, Zone, Reason string }
type financialTermBinding struct {
	Spec                            financialTermBindingSpec
	Term                            financialTermRecord
	Binding                         receivables.Binding
	Module                          receivables.BindingView
	Author, Approver, Digest, State string
}

func copyFinancialBinding(p financialTermBinding) financialTermBinding {
	p.Term = copyFinancialTerm(p.Term)
	p.Module.Ends = slices.Clone(p.Module.Ends)
	return p
}
func bindingIntentDigest(p financialTermBinding) (string, error) {
	return financialEvidenceDigest("issuance_term_binding", struct {
		Spec    financialTermBindingSpec
		Term    financialTermRecord
		Binding receivables.Binding
		Author  string
	}{p.Spec, p.Term, p.Binding, p.Author})
}
func termBindingDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && s >= "2000-01-01" && s <= "2099-12-31" && len(s) == 10
}
func termBindingEnd(s string) string {
	if s == "" {
		return "9999-12-31"
	}
	return s
}
func (a *financialIssuance) currentBinding(workspace string, p financialTermBinding, actor, capability string) error {
	digest, err := bindingIntentDigest(p)
	if err != nil || digest != p.Digest {
		return commands.Fail(commands.EvidenceStale)
	}
	term, ok := a.termIntents[p.Spec.Term]
	if !ok || term.State != "approved" || term.Digest != p.Spec.TermDigest || !reflect.DeepEqual(term, p.Term) {
		return commands.Fail(commands.EvidenceStale)
	}
	if err := a.currentTerm(workspace, term, actor, capability); err != nil {
		return err
	}
	if _, err := a.termReadback(workspace, term); err != nil {
		return err
	}
	if err := a.termActor(workspace, p.Author, termsauthority.BindTerm); err != nil {
		return err
	}
	if p.Approver != "" {
		if err := a.termActor(workspace, p.Approver, termsauthority.ApproveBinding); err != nil {
			return err
		}
	}
	return nil
}
func (a *financialIssuance) bindingReadback(workspace string, p financialTermBinding) (financialTermBinding, error) {
	v, err := a.terms.BindingOf(workspace, p.Binding.ID)
	if err != nil || !reflect.DeepEqual(v, p.Module) || v.Binding != p.Binding || v.Author != p.Author || v.Digest == "" || v.WithdrawnBy != "" || v.PendingEnd != "" || v.EndAuthor != "" || len(v.Ends) != 0 {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State == "pending" && (v.State != receivables.StatePending || v.Approver != "") || p.State == "approved" && (v.State != receivables.StateApproved || v.Approver != p.Approver) {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State != "pending" && p.State != "approved" {
		return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialBinding(p), nil
}
func (a *financialIssuance) bindTerm(workspace, actor string, spec financialTermBindingSpec) (financialTermBinding, error) {
	if err := a.termCaller(workspace, actor, termsauthority.BindTerm); err != nil {
		return financialTermBinding{}, err
	}
	if !validRequestID(spec.ID) || !termBindingDate(spec.From) || spec.To != "" && (!termBindingDate(spec.To) || spec.To < spec.From) || (spec.Reason != "initial" && spec.Reason != "renegotiation" && spec.Reason != "correction") {
		return financialTermBinding{}, commands.Fail(commands.InvalidRequest)
	}
	term, ok := a.termIntents[spec.Term]
	if !ok || term.Digest != spec.TermDigest || term.State != "approved" {
		return financialTermBinding{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if spec.Zone != term.Profile.Spec.Zone {
		return financialTermBinding{}, commands.Fail(commands.InvalidRequest)
	}
	if err := a.currentTerm(workspace, term, actor, termsauthority.BindTerm); err != nil {
		return financialTermBinding{}, err
	}
	if _, err := a.termReadback(workspace, term); err != nil {
		return financialTermBinding{}, err
	}
	id, err := financialEvidenceDigest("issuance_binding", spec.ID)
	if err != nil {
		return financialTermBinding{}, err
	}
	p := financialTermBinding{Spec: spec, Term: copyFinancialTerm(term), Binding: receivables.Binding{ID: "issued-binding-" + id, Contract: term.Profile.Source.Basis.Contract.ContractID, Term: term.Term.ID, Revision: term.Term.Revision, Zone: spec.Zone, From: spec.From, To: spec.To, Reason: spec.Reason}, Author: actor}
	p.Digest, err = bindingIntentDigest(p)
	if err != nil {
		return financialTermBinding{}, err
	}
	if old, ok := a.bindingIntents[spec.ID]; ok {
		if old.Author != actor || old.Digest != p.Digest {
			return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentBinding(workspace, old, actor, termsauthority.BindTerm); err != nil {
			return financialTermBinding{}, err
		}
		if old.State != "registration_unknown" {
			return a.bindingReadback(workspace, old)
		}
		p = old
	} else {
		if len(a.bindingIntents) >= financialIssuanceLimit {
			return financialTermBinding{}, commands.Fail(commands.PrerequisiteFailed)
		}
		for _, other := range a.bindingIntents {
			if other.Binding.Contract == p.Binding.Contract && p.Binding.From <= termBindingEnd(other.Binding.To) && other.Binding.From <= termBindingEnd(p.Binding.To) {
				return financialTermBinding{}, commands.Fail(commands.PrerequisiteFailed)
			}
		}
		if _, err := a.terms.BindingOf(workspace, p.Binding.ID); !errors.Is(err, receivables.ErrNotFound) {
			return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-binding-before-register") {
			return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State = "registration_unknown"
		a.bindingIntents[spec.ID] = copyFinancialBinding(p)
	}
	v, err := a.terms.Bind(workspace, p.Binding, actor)
	if err != nil {
		return financialTermBinding{}, err
	}
	if v.Binding != p.Binding || v.Author != actor || v.State != receivables.StatePending || v.Approver != "" || v.WithdrawnBy != "" || v.PendingEnd != "" || len(v.Ends) != 0 || p.Module.Digest != "" && !reflect.DeepEqual(v, p.Module) {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = v
	a.bindingIntents[spec.ID] = copyFinancialBinding(p)
	if a.server.faulty("issuance-binding-after-register") {
		return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "pending"
	a.bindingIntents[spec.ID] = copyFinancialBinding(p)
	if a.server.faulty("issuance-binding-after-register-publication") {
		return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialBinding(p), nil
}
func (a *financialIssuance) approveTermBinding(workspace, id, digest, actor string) (financialTermBinding, error) {
	if err := a.termCaller(workspace, actor, termsauthority.ApproveBinding); err != nil {
		return financialTermBinding{}, err
	}
	p, ok := a.bindingIntents[id]
	if !ok || p.Digest != digest {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	if p.Author == actor {
		return financialTermBinding{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Approver != "" && p.Approver != actor {
		return financialTermBinding{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentBinding(workspace, p, actor, termsauthority.ApproveBinding); err != nil {
		return financialTermBinding{}, err
	}
	if p.State == "approved" {
		return a.bindingReadback(workspace, p)
	}
	if p.State != "pending" && p.State != "approval_unknown" {
		return financialTermBinding{}, commands.Fail(commands.PrerequisiteFailed)
	}
	v, err := a.terms.BindingOf(workspace, p.Binding.ID)
	if err != nil || !reflect.DeepEqual(v, p.Module) || v.Binding != p.Binding || v.Author != p.Author || v.State != receivables.StatePending && (p.State != "approval_unknown" || v.State != receivables.StateApproved || v.Approver != actor) {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-binding-before-approve") {
		return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "approval_unknown"
	p.Approver = actor
	a.bindingIntents[id] = copyFinancialBinding(p)
	v, err = a.terms.ApproveBinding(workspace, p.Binding.ID, actor)
	if err != nil {
		return financialTermBinding{}, err
	}
	if v.Binding != p.Binding || v.Author != p.Author || v.Approver != actor || v.State != receivables.StateApproved || v.Digest != p.Module.Digest || v.Written != p.Module.Written || v.WithdrawnBy != "" || v.PendingEnd != "" || len(v.Ends) != 0 {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = v
	a.bindingIntents[id] = copyFinancialBinding(p)
	if a.server.faulty("issuance-binding-after-approve") {
		return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "approved"
	a.bindingIntents[id] = copyFinancialBinding(p)
	if a.server.faulty("issuance-binding-after-approve-publication") {
		return financialTermBinding{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyFinancialBinding(p), nil
}
func (a *financialIssuance) termCoverage(workspace, id, digest, actor string, at time.Time) (receivables.Coverage, error) {
	if err := a.termCaller(workspace, actor, termsauthority.ApproveBinding); err != nil {
		return receivables.Coverage{}, err
	}
	p, ok := a.bindingIntents[id]
	if !ok || p.Digest != digest || p.State != "approved" {
		return receivables.Coverage{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.currentBinding(workspace, p, actor, termsauthority.ApproveBinding); err != nil {
		return receivables.Coverage{}, err
	}
	if _, err := a.bindingReadback(workspace, p); err != nil {
		return receivables.Coverage{}, err
	}
	coverage, err := a.terms.Cover(workspace, p.Binding.Contract, at)
	if err != nil {
		return receivables.Coverage{}, err
	}
	if coverage.Contract != p.Binding.Contract || coverage.Binding != p.Binding.ID || coverage.Term != p.Term.Term.ID || coverage.Revision != p.Term.Term.Revision || coverage.NetDays != p.Term.Term.NetDays || coverage.Mode != p.Term.Term.Mode || coverage.Zone != p.Binding.Zone {
		return receivables.Coverage{}, commands.Fail(commands.EvidenceStale)
	}
	return coverage, nil
}
