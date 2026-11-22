package server

import (
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func (a *financialIssuance) configuredConsumption(c financialIssuanceConfiguration, at time.Time) error {
	if at.IsZero() || at.Before(c.ActivatedAt) {
		return commands.Fail(commands.EvidenceStale)
	}
	actual, err := a.ownedConfirmedConfiguration(c.Workspace, c.Spec.ID, c.Digest)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, c) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialIssuance) configuredReferences(c financialIssuanceConfiguration, at time.Time) ([]issuance.Reference, error) {
	if err := a.configuredConsumption(c, at); err != nil {
		return nil, err
	}
	fields := []string{protected.FieldPaymentAccount, protected.FieldDocumentRecipient, protected.FieldDocumentTemplate}
	out := make([]issuance.Reference, 0, len(fields))
	for _, field := range fields {
		subject, err := a.configuredChangeSubject(c, field)
		if err != nil {
			return nil, err
		}
		if _, err := a.changeTrail(c.Workspace, subject.Subject, field); err != nil {
			return nil, err
		}
		for _, id := range a.changeOrders[issuanceChangeScope(subject.Subject, field)] {
			if a.changes[id].Phase != "recorded" {
				return nil, commands.Fail(commands.DependencyUnavailable)
			}
		}
		resolved, err := a.fields.Resolve(c.Workspace, subject.Subject, field, at)
		if err != nil {
			return nil, err
		}
		p, v, err := a.ownedChangeForConsumption(c.Workspace, resolved.Change)
		if err != nil {
			return nil, err
		}
		if p.Configuration == nil || !reflect.DeepEqual(*p.Configuration, c) || p.Spec.Field != field || !reflect.DeepEqual(p.Channel.Subject, subject) || v.Change.Subject != subject.Subject || resolved.Ref != p.Spec.After || !resolved.Effective.Equal(p.Spec.Effective) || resolved.Effective.After(at) {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		out = append(out, issuance.Reference{Field: field, Subject: subject.Subject, Change: resolved.Change, Ref: resolved.Ref, Effective: resolved.Effective.UTC()})
	}
	return out, nil
}

func (a *financialIssuance) configuredCoverage(c financialIssuanceConfiguration, id, digest string, at time.Time) (receivables.Coverage, error) {
	if err := a.configuredConsumption(c, at); err != nil {
		return receivables.Coverage{}, err
	}
	p, err := a.ownedBindingForConsumption(c.Workspace, id, digest)
	if err != nil {
		return receivables.Coverage{}, err
	}
	if p.Term.Configuration == nil || !reflect.DeepEqual(*p.Term.Configuration, c) || p.Binding.Contract != c.Scope.Contract.ContractID || p.Binding.Zone != c.Spec.Zone {
		return receivables.Coverage{}, commands.Fail(commands.EvidenceStale)
	}
	return a.bindingCoverage(c.Workspace, p, at)
}
