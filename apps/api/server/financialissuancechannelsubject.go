package server

import (
	"reflect"
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

type financialChannelSelector struct {
	Kind, Contract, BillingProfile, Reservation, ReservationDigest string
	Revision                                                       uint64
}

type financialChannelParty struct {
	Party   organizations.Party
	Version uint64
}

type financialChannelSubject struct {
	Selector         financialChannelSelector
	Subject          string
	Workspace        organizations.Workspace
	WorkspaceVersion uint64
	Parties          []financialChannelParty
	Contract         organizations.ContractRevision
	BillingProfile   organizations.BillingProfile
	BillingVersion   uint64
	ReservationSpec  financialIssuanceProfileSpec
	ReservedKind     issuance.Kind
}

func copyChannelSubject(p financialChannelSubject) financialChannelSubject {
	p.Parties = slices.Clone(p.Parties)
	p.Contract.Components = slices.Clone(p.Contract.Components)
	p.ReservationSpec.Languages = slices.Clone(p.ReservationSpec.Languages)
	p.ReservedKind.Languages = slices.Clone(p.ReservedKind.Languages)
	return p
}

func (a *financialIssuance) channelParty(workspace, id string) ([]financialChannelParty, string, error) {
	p, version, err := a.server.orgs.Party(workspace, id)
	if err != nil || version == 0 || p.ID != id || p.WorkspaceID != workspace || p.State != organizations.Active {
		return nil, "", commands.Fail(commands.EvidenceStale)
	}
	parties := []financialChannelParty{{p, version}}
	switch p.Kind {
	case organizations.LegalEntity:
		if p.ParentID != "" {
			return nil, "", commands.Fail(commands.EvidenceStale)
		}
		return parties, p.ID, nil
	case organizations.InternalUnit:
		parent, pv, err := a.server.orgs.Party(workspace, p.ParentID)
		if err != nil || pv == 0 || parent.ID != p.ParentID || parent.WorkspaceID != workspace || parent.Kind != organizations.LegalEntity || parent.State != organizations.Active || parent.ParentID != "" {
			return nil, "", commands.Fail(commands.EvidenceStale)
		}
		return append(parties, financialChannelParty{parent, pv}), parent.ID, nil
	}
	return nil, "", commands.Fail(commands.EvidenceStale)
}

func (a *financialIssuance) channelSubject(workspace string, sel financialChannelSelector) (financialChannelSubject, error) {
	bad := func() (financialChannelSubject, error) {
		return financialChannelSubject{}, commands.Fail(commands.InvalidRequest)
	}
	if workspace != a.server.bootWorkspace {
		return financialChannelSubject{}, commands.Fail(commands.PermissionDenied)
	}
	p := financialChannelSubject{Selector: sel}
	var err error
	p.Workspace, p.WorkspaceVersion, err = a.server.orgs.Workspace(workspace)
	if err != nil || p.WorkspaceVersion == 0 || p.Workspace.ID != workspace || p.Workspace.State != organizations.Active {
		return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
	}
	switch sel.Kind {
	case "customer":
		if !validRequestID(sel.Contract) || sel.Revision == 0 || sel.BillingProfile != "" || sel.Reservation != "" || sel.ReservationDigest != "" {
			return bad()
		}
		p.Contract, err = a.server.orgs.ContractRevision(workspace, sel.Contract, sel.Revision)
		if err != nil || p.Contract.WorkspaceID != workspace || p.Contract.ContractID != sel.Contract || p.Contract.Revision != sel.Revision {
			return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
		}
		p.Parties, _, err = a.channelParty(workspace, p.Contract.CustomerID)
		p.Subject = "customer:" + p.Contract.CustomerID
	case "entity":
		if !validRequestID(sel.BillingProfile) || sel.Contract != "" || sel.Revision != 0 || sel.Reservation != "" || sel.ReservationDigest != "" {
			return bad()
		}
		p.BillingProfile, p.BillingVersion, err = a.server.orgs.BillingProfile(workspace, sel.BillingProfile)
		if err != nil || p.BillingVersion == 0 || p.BillingProfile.ID != sel.BillingProfile || p.BillingProfile.WorkspaceID != workspace || p.BillingProfile.State != organizations.Active {
			return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
		}
		var entity string
		p.Parties, entity, err = a.channelParty(workspace, p.BillingProfile.IssuerID)
		p.Subject = "entity:" + entity
	case "kind":
		if !validRequestID(sel.Reservation) || !validPlanHash(sel.ReservationDigest) || sel.Contract != "" || sel.Revision != 0 || sel.BillingProfile != "" {
			return bad()
		}
		r, ok := a.kindReservations[sel.Reservation]
		if !ok || r.Spec.ID != sel.Reservation || r.Digest != sel.ReservationDigest || r.State != "confirmed" {
			return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentKindReservation(workspace, r); err != nil {
			return financialChannelSubject{}, err
		}
		p.ReservationSpec, p.ReservedKind = r.Spec, r.Kind
		p.Subject = "kind:" + r.Kind.ID
	default:
		return bad()
	}
	if err != nil {
		return financialChannelSubject{}, err
	}
	if !validRequestID(p.Subject) || len(p.Subject) > 128 {
		return bad()
	}
	return copyChannelSubject(p), nil
}

func (a *financialIssuance) currentChannelSubject(workspace string, p financialChannelSubject) error {
	current, err := a.channelSubject(workspace, p.Selector)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, p) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}
