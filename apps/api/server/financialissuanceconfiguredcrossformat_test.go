package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func TestFinancialConfiguredTermsCrossFormatOwnershipAndOverlap(t *testing.T) {
	for _, configuredFirst := range []bool{false, true} {
		name := "legacy-first"
		if configuredFirst {
			name = "configured-first"
		}
		t.Run(name, func(t *testing.T) {
			f, author, approver, legacySpec := termsWorld(t)
			a := f.s.financialIssuance
			cs := configurationSpec()
			cs.Zone = "Asia/Tokyo"
			cp, err := proposeConfigurationAs(f, f.as(AdminLogin), cs)
			f.must(err)
			c, err := confirmConfigurationAs(f, f.as(Admin2Login), cp)
			f.must(err)
			r := configurationReference(c)
			configuredSpec := legacySpec
			configuredSpec.Configuration = &r
			configuredSpec.Profile = ""
			configuredSpec.ProfileDigest = ""
			firstSpec, otherSpec := legacySpec, configuredSpec
			if configuredFirst {
				firstSpec, otherSpec = configuredSpec, legacySpec
			}
			f.s.fault = func(at string) bool { return at == "issuance-term-after-register" }
			_, err = recordTermAs(f, author, firstSpec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			firstUnknown := copyFinancialTerm(a.termIntents[firstSpec.ID])
			_, err = recordTermAs(f, author, otherSpec)
			documentFailure(t, err, commands.EvidenceStale)
			if !reflect.DeepEqual(a.termIntents[firstSpec.ID], firstUnknown) || len(a.termScopes) != 1 {
				t.Fatal("other format replaced unknown term identity")
			}
			first, err := recordTermAs(f, author, firstSpec)
			f.must(err)
			first, err = approveTermAs(f, approver, first)
			f.must(err)
			otherSpec.ID = "other-format-term"
			other, err := recordTermAs(f, author, otherSpec)
			f.must(err)
			other, err = approveTermAs(f, approver, other)
			f.must(err)
			bs := bindingSpec(first)
			f.s.fault = func(at string) bool { return at == "issuance-binding-after-register" }
			_, err = bindTermAs(f, author, bs)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			retained := copyFinancialBinding(a.bindingIntents[bs.ID])
			competitor := bindingSpec(other)
			_, err = bindTermAs(f, author, competitor)
			documentFailure(t, err, commands.EvidenceStale)
			competitor.ID = "overlapping-other-format"
			_, err = bindTermAs(f, author, competitor)
			documentFailure(t, err, commands.PrerequisiteFailed)
			if len(a.bindingIntents) != 1 || !reflect.DeepEqual(a.bindingIntents[bs.ID], retained) {
				t.Fatal("other format bypassed uncertain binding overlap")
			}
			firstBinding, err := bindTermAs(f, author, bs)
			f.must(err)
			_, err = approveBindingAs(f, approver, firstBinding)
			f.must(err)
			_, err = bindTermAs(f, author, competitor)
			documentFailure(t, err, commands.PrerequisiteFailed)
			competitor.From = "2027-01-01"
			competitor.To = "2027-12-31"
			adjacent, err := bindTermAs(f, author, competitor)
			f.must(err)
			_, err = approveBindingAs(f, approver, adjacent)
			f.must(err)
			if len(a.bindingIntents) != 2 || len(a.termIntents) != 2 {
				t.Fatal("adjacent mixed-format binding not retained")
			}
		})
	}
}

func TestFinancialConfiguredChangesCrossFormatUnknownTrail(t *testing.T) {
	for _, configuredFirst := range []bool{false, true} {
		name := "legacy-first"
		if configuredFirst {
			name = "configured-first"
		}
		t.Run(name, func(t *testing.T) {
			f, requester, verifier, approver, legacy := protectedChangeWorld(t, 24*time.Hour, 1)
			a := f.s.financialIssuance
			cp, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
			f.must(err)
			c, err := confirmConfigurationAs(f, f.as(Admin3Login), cp)
			f.must(err)
			r := configurationReference(c)
			legacy.Field = protected.FieldPaymentAccount
			legacy.Channel = a.channels["mature-entity"].Spec.ID
			legacy.ChannelDigest = a.channels["mature-entity"].Digest
			configured := legacy
			configured.Configuration = &r
			configured.Profile = ""
			configured.ProfileDigest = ""
			first, other := legacy, configured
			if configuredFirst {
				first, other = configured, legacy
			}
			f.s.fault = func(at string) bool { return at == "issuance-change-after-request" }
			_, err = requestChangeAs(f, requester, first)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			retained := copyChange(a.changes[first.ID])
			_, err = requestChangeAs(f, requester, other)
			documentFailure(t, err, commands.EvidenceStale)
			other.ID = "other-format-change"
			_, err = requestChangeAs(f, requester, other)
			documentFailure(t, err, commands.PrerequisiteFailed)
			if len(a.changes) != 1 || !reflect.DeepEqual(a.changes[first.ID], retained) {
				t.Fatal("mixed origin bypassed unknown trail ownership")
			}
			recovered, err := requestChangeAs(f, requester, first)
			f.must(err)
			recovered, err = verifyChangeAs(f, verifier, recovered)
			f.must(err)
			_, err = decideChangeAs(f, approver, recovered)
			f.must(err)
			other.Effective = first.Effective.Add(time.Minute)
			other.After.ID = "next-reference"
			next, err := requestChangeAs(f, requester, other)
			f.must(err)
			if next.Before != first.After || len(a.changeOrders[issuanceChangeScope(retained.Channel.Subject.Subject, first.Field)]) != 2 {
				t.Fatal("mixed-format predecessor lost authoritative trail")
			}
		})
	}
}
