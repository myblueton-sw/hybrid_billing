package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type financialConfigurationReference struct {
	Schema, ID, Digest string
}

func configurationReference(c financialIssuanceConfiguration) financialConfigurationReference {
	return financialConfigurationReference{Schema: "hb_issuance_configuration_reference_v1", ID: c.Spec.ID, Digest: c.Digest}
}

func validConfigurationReference(r *financialConfigurationReference) bool {
	return r != nil && r.Schema == "hb_issuance_configuration_reference_v1" && validRequestID(r.ID) && validPlanHash(r.Digest)
}

func copyChannelSelector(s financialChannelSelector) financialChannelSelector {
	if s.Configuration != nil {
		r := *s.Configuration
		s.Configuration = &r
	}
	return s
}

func validChannelSelectorOrigin(s financialChannelSelector) bool {
	if s.Kind == "configuration_kind" {
		return validConfigurationReference(s.Configuration) && s.Contract == "" && s.BillingProfile == "" && s.Reservation == "" && s.ReservationDigest == "" && s.Revision == 0
	}
	if s.Configuration != nil {
		return false
	}
	switch s.Kind {
	case "customer":
		return validRequestID(s.Contract) && s.Revision != 0 && s.BillingProfile == "" && s.Reservation == "" && s.ReservationDigest == ""
	case "entity":
		return validRequestID(s.BillingProfile) && s.Contract == "" && s.Revision == 0 && s.Reservation == "" && s.ReservationDigest == ""
	case "kind":
		return validRequestID(s.Reservation) && validPlanHash(s.ReservationDigest) && s.Contract == "" && s.Revision == 0 && s.BillingProfile == ""
	}
	return false
}

func configuredChannelSubject(s financialChannelSelector, c financialIssuanceConfiguration) financialChannelSubject {
	return copyChannelSubject(financialChannelSubject{Selector: s, Subject: "kind:" + c.Kind.ID, Workspace: c.Scope.Workspace, WorkspaceVersion: c.Scope.WorkspaceVersion, ReservedKind: c.Kind, Configuration: &c})
}

func channelOrigin(p financialChannelRecord) error {
	bad := func() error { return commands.Fail(commands.EvidenceStale) }
	if !validChannelSelectorOrigin(p.Spec.Selector) || !reflect.DeepEqual(p.Spec.Selector, p.Subject.Selector) {
		return bad()
	}
	if p.Spec.Selector.Configuration == nil {
		if p.Subject.Configuration != nil {
			return bad()
		}
		return nil
	}
	if p.Subject.Configuration == nil {
		return bad()
	}
	c := *p.Subject.Configuration
	digest, err := configurationDigest(c)
	if err != nil || digest != c.Digest || c.Schema != "hb_issuance_configuration_v1" || c.Workspace != p.Workspace || c.State != "confirmed" || !configurationActive(c) || c.ActivatedAt != c.Active.At || configurationReference(c) != *p.Spec.Selector.Configuration {
		return bad()
	}
	if !reflect.DeepEqual(p.Subject, configuredChannelSubject(p.Spec.Selector, c)) {
		return bad()
	}
	return nil
}
