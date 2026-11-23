package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"reflect"
	"time"
)

func (a *financialIssuance) ownedChangeForConsumption(workspace, id string) (financialChangeRecord, protected.View, error) {
	fail := func() (financialChangeRecord, protected.View, error) {
		return financialChangeRecord{}, protected.View{}, commands.Fail(commands.EvidenceStale)
	}
	p, ok := a.changes[id]
	digest, err := changeDigest(p)
	if !ok || p.Workspace != workspace || p.Spec.ID != id || err != nil || digest != p.Digest || p.Phase != "recorded" {
		return fail()
	}
	if err := a.retainedSetupOrigin(workspace, p.Spec.Profile, p.Spec.ProfileDigest, p.Spec.Configuration, p.Profile, p.Configuration); err != nil {
		return financialChangeRecord{}, protected.View{}, err
	}
	v, err := a.changeStanding(p)
	if err != nil {
		return financialChangeRecord{}, protected.View{}, err
	}
	if v.State != protected.StateScheduled || v.Verification == nil || v.Verification.Verification.Outcome != protected.OutcomeConfirmed || len(v.Decisions) != v.Rules.Approvals || len(p.Decisions) != v.Rules.Approvals {
		return fail()
	}
	seen := map[string]bool{p.Requester: true, p.Channel.Registrar: true, p.Channel.Admitter: true}
	if seen[v.Verification.Verifier] {
		return fail()
	}
	seen[v.Verification.Verifier] = true
	for _, d := range v.Decisions {
		if seen[d.Approver] || d.Approval.Verdict != protected.VerdictApprove || d.Approval.Digest != v.Verification.Digest {
			return fail()
		}
		seen[d.Approver] = true
	}
	return copyChange(p), copyProtectedView(v), nil
}

func (a *financialIssuance) draftReferences(current financialIssuanceProfile, at time.Time) ([]issuance.Reference, error) {
	if current.Spec.Mode != "erp_issuance" || current.Spec.Profile != "synthetic_erp_document_v1" || current.Kind.Class != issuance.ClassERP || current.Kind.Credit || current.Kind.Statutory || at.IsZero() {
		return nil, commands.Fail(commands.UnsupportedCapability)
	}
	workspace := current.Source.Basis.Workspace
	fields := []string{protected.FieldPaymentAccount, protected.FieldDocumentRecipient, protected.FieldDocumentTemplate}
	out := make([]issuance.Reference, 0, len(fields))
	for _, field := range fields {
		subject, err := a.changeSubject(current, field)
		if err != nil {
			return nil, err
		}
		if _, err := a.changeTrail(workspace, subject.Subject, field); err != nil {
			return nil, err
		}
		for _, id := range a.changeOrders[issuanceChangeScope(subject.Subject, field)] {
			if a.changes[id].Phase != "recorded" {
				return nil, commands.Fail(commands.DependencyUnavailable)
			}
		}
		resolved, err := a.fields.Resolve(workspace, subject.Subject, field, at)
		if err != nil {
			return nil, err
		}
		p, v, err := a.ownedChangeForConsumption(workspace, resolved.Change)
		if err != nil {
			return nil, err
		}
		if p.Configuration != nil || p.Spec.Field != field || !reflect.DeepEqual(p.Channel.Subject, subject) || v.Change.Subject != subject.Subject || resolved.Ref != p.Spec.After || !resolved.Effective.Equal(p.Spec.Effective) || resolved.Effective.After(at) {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		out = append(out, issuance.Reference{Field: field, Subject: subject.Subject, Change: resolved.Change, Ref: resolved.Ref, Effective: resolved.Effective.UTC()})
	}
	return out, nil
}
