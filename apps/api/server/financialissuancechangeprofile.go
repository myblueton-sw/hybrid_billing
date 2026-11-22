package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func (a *financialIssuance) protectedProfile(workspace, id, digest, actor, capability string) (financialIssuanceProfile, error) {
	if err := a.protectedCaller(workspace, actor, capability); err != nil {
		return financialIssuanceProfile{}, err
	}
	p, err := a.ownedConfirmedProfile(workspace, id, digest)
	if err != nil {
		return financialIssuanceProfile{}, err
	}
	return a.profileCurrentFinancialSource(workspace, actor, p)
}

func (a *financialIssuance) changeSubject(p financialIssuanceProfile, field string) (financialChannelSubject, error) {
	var sel financialChannelSelector
	switch field {
	case protected.FieldDocumentRecipient:
		sel = financialChannelSelector{Kind: "customer", Contract: p.Source.Basis.Contract.ContractID, Revision: p.Source.Basis.Contract.Revision}
	case protected.FieldDocumentTemplate:
		sel = financialChannelSelector{Kind: "kind", Reservation: p.Spec.ID, ReservationDigest: p.ReservationDigest}
	case protected.FieldPaymentAccount:
		sel = financialChannelSelector{Kind: "entity", BillingProfile: p.Source.Basis.BillingProfile.ID}
	default:
		return financialChannelSubject{}, commands.Fail(commands.InvalidRequest)
	}
	subject, err := a.channelSubject(p.Source.Basis.Workspace, sel)
	if err != nil {
		return financialChannelSubject{}, err
	}
	switch sel.Kind {
	case "customer":
		if !reflect.DeepEqual(subject.Contract, p.Source.Basis.Contract) || subject.Subject != "customer:"+p.Source.Basis.Customer {
			return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
		}
	case "entity":
		if subject.BillingProfile != p.Source.Basis.BillingProfile || subject.BillingVersion != p.Source.Basis.BillingRevision || subject.Subject != "entity:"+p.Source.Basis.Document.IssuerLegalEntityID {
			return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
		}
	case "kind":
		if !reflect.DeepEqual(subject.ReservedKind, p.Kind) {
			return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
		}
	}
	return subject, nil
}
