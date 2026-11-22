package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
	"reflect"
	"time"
)

func (a *financialIssuance) ownedBindingForConsumption(workspace, id, digest string) (financialTermBinding, error) {
	fail := func() (financialTermBinding, error) {
		return financialTermBinding{}, commands.Fail(commands.EvidenceStale)
	}
	p, ok := a.bindingIntents[id]
	expected, err := bindingIntentDigest(p)
	if !ok || p.Spec.ID != id || p.Digest != digest || err != nil || expected != digest || p.State != "approved" || p.Author == "" || p.Approver == "" || p.Author == p.Approver {
		return fail()
	}
	term, ok := a.termIntents[p.Spec.Term]
	termDigest, err := termIntentDigest(term)
	if !ok || term.Spec.ID != p.Spec.Term || term.State != "approved" || term.Author == "" || term.Approver == "" || term.Author == term.Approver || err != nil || termDigest != term.Digest || term.Digest != p.Spec.TermDigest || !reflect.DeepEqual(term, p.Term) || a.termScopes[term.Profile.Source.Basis.Subject.Target] != term.Spec.ID {
		return fail()
	}
	profile, err := a.ownedConfirmedProfile(workspace, term.Spec.Profile, term.Spec.ProfileDigest)
	if err != nil {
		return financialTermBinding{}, err
	}
	if !reflect.DeepEqual(profile, term.Profile) {
		return fail()
	}
	for _, right := range []struct{ actor, capability string }{{term.Author, termsauthority.RecordTerm}, {term.Approver, termsauthority.ApproveTerm}, {p.Author, termsauthority.BindTerm}, {p.Approver, termsauthority.ApproveBinding}} {
		if err := a.termActor(workspace, right.actor, right.capability); err != nil {
			return financialTermBinding{}, err
		}
	}
	if _, err := a.termReadback(workspace, term); err != nil {
		return financialTermBinding{}, err
	}
	return a.bindingReadback(workspace, p)
}

func (a *financialIssuance) draftCoverage(current financialIssuanceProfile, id, digest string, at time.Time) (receivables.Coverage, error) {
	fail := func() (receivables.Coverage, error) {
		return receivables.Coverage{}, commands.Fail(commands.EvidenceStale)
	}
	workspace := current.Source.Basis.Workspace
	p, err := a.ownedBindingForConsumption(workspace, id, digest)
	if err != nil {
		return receivables.Coverage{}, err
	}
	old := p.Term.Profile.Source.Basis
	source := current.Source.Basis
	if !reflect.DeepEqual(old.Contract, source.Contract) || old.Customer != source.Customer || current.Spec.Zone != p.Binding.Zone || at.IsZero() {
		return fail()
	}
	coverage, err := a.terms.Cover(workspace, source.Contract.ContractID, at)
	if err != nil {
		return receivables.Coverage{}, err
	}
	if coverage.Contract != p.Binding.Contract || coverage.Binding != p.Binding.ID || coverage.Term != p.Term.Term.ID || coverage.Revision != p.Term.Term.Revision || coverage.NetDays != p.Term.Term.NetDays || coverage.Mode != p.Term.Term.Mode || coverage.Zone != p.Binding.Zone {
		return fail()
	}
	location, err := time.LoadLocation(coverage.Zone)
	if err != nil {
		return receivables.Coverage{}, commands.Fail(commands.DependencyUnavailable)
	}
	date := at.In(location).Format(time.DateOnly)
	day, err := time.Parse(time.DateOnly, date)
	if err != nil || coverage.Mode != receivables.ModeCalendar || coverage.Date != date || coverage.Due != day.AddDate(0, 0, coverage.NetDays).Format(time.DateOnly) {
		return fail()
	}
	return coverage, nil
}
