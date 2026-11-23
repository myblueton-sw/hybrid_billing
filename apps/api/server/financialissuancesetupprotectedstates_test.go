package server

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func confirmedSetupConfiguration(t *testing.T) (world, actor, financialIssuanceConfiguration) {
	t.Helper()
	f, reader := setupContextWorld(t)
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	return f, reader, c
}

func TestFinancialSetupProtectedAccessAndConfiguration(t *testing.T) {
	w := configuredSetup(t)
	f, c := w.f, w.configuration
	completeConfiguredSetup(t, w)
	root, member := f.as(AdminLogin), f.as(MemberLogin)
	missing := c
	missing.Spec.ID = "missing-configuration"
	for _, who := range []actor{root, member} {
		for _, target := range []financialIssuanceConfiguration{c, missing} {
			for _, command := range []string{"protected", "protected_channels"} {
				f.code(t, "inspection required", protectedSetupRead(f, who, command, target, nil), 403, string(commands.PermissionDenied))
			}
			f.code(t, "inspection required", protectedSetupRead(f, who, "protected_changes", target, map[string]string{"field": protected.FieldPaymentAccount}), 403, string(commands.PermissionDenied))
		}
	}
	grantSetupInspectionForTest(f, true)
	f.code(t, "administrator without inspection", protectedSetupRead(f, root, "protected", c, nil), 403, string(commands.PermissionDenied))
	f.code(t, "unknown configuration", protectedSetupRead(f, member, "protected", missing, nil), 403, string(commands.PermissionDenied))
	foreign := c
	foreign.Spec.Contract = "sample-contract-02"
	f.code(t, "other contract", protectedSetupRead(f, member, "protected", foreign, nil), 403, string(commands.PermissionDenied))
	stale := c
	stale.Digest = strings.Repeat("0", 64)
	f.code(t, "stale digest", protectedSetupRead(f, member, "protected_channels", stale, nil), 409, string(commands.EvidenceStale))
	stale.Digest = "short"
	f.code(t, "malformed digest", protectedSetupRead(f, member, "protected", stale, nil), 422, string(commands.InvalidRequest))
	f.code(t, "cursor on summary", protectedSetupRead(f, member, "protected", c, map[string]string{"after": "x"}), 422, string(commands.InvalidRequest))
	f.code(t, "cursor without snapshot", protectedSetupRead(f, member, "protected_channels", c, map[string]string{"after": "setup-customer"}), 422, string(commands.InvalidRequest))

	base, err := f.s.orgs.ContractRevision(SampleWorkspace, c.Spec.Contract, c.Spec.ContractRevision)
	f.must(err)
	base.Revision++
	f.must(f.s.orgs.AppendContractRevision(base))
	spec := configurationSpec()
	spec.ID, spec.ContractRevision = "pending-configuration", base.Revision
	pending, err := proposeConfigurationAs(f, root, spec)
	f.must(err)
	f.code(t, "unconfirmed configuration", protectedSetupRead(f, member, "protected", pending, nil), 422, string(commands.PrerequisiteFailed))

	first := protectedSetupRead(f, member, "protected", c, nil)
	channels := protectedSetupRead(f, member, "protected_channels", c, nil)
	changes := protectedSetupRead(f, member, "protected_changes", c, map[string]string{"field": protected.FieldDocumentRecipient})
	if first.status != 200 || channels.status != 200 || changes.status != 200 {
		t.Fatal(first.raw, channels.raw, changes.raw)
	}
	profile, v, err := f.s.orgs.BillingProfile(SampleWorkspace, c.Spec.BillingProfile)
	f.must(err)
	profile.State = organizations.Suspended
	_, err = f.s.orgs.PutBillingProfile(profile, v)
	f.must(err)
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleAdmin || grant.SubjectID == SampleAdmin2 || grant.SubjectID == SampleAdmin3 || strings.HasPrefix(grant.SubjectID, "setup-") || strings.HasPrefix(grant.SubjectID, "configured-") {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	f.s.contracts = nil
	f.s.enterHold("Historical protected setup inspection")
	before := captureProtectedSetup(f)
	for name, pair := range map[string][2]reply{
		"summary":  {first, protectedSetupRead(f, member, "protected", c, nil)},
		"channels": {channels, protectedSetupRead(f, member, "protected_channels", c, nil)},
		"changes":  {changes, protectedSetupRead(f, member, "protected_changes", c, map[string]string{"field": protected.FieldDocumentRecipient})},
	} {
		if pair[1].status != 200 || !reflect.DeepEqual(pair[0].body["fields"], pair[1].body["fields"]) {
			t.Fatalf("%s: live standing rewrote history: %s", name, pair[1].raw)
		}
	}
	if !reflect.DeepEqual(before, captureProtectedSetup(f)) {
		t.Fatal("held read changed state")
	}
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleMember {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	f.code(t, "revoked reader", protectedSetupRead(f, member, "protected", c, nil), 403, string(commands.PermissionDenied))
	f.s.endProbe(member.verifier(f))
	f.code(t, "ended session", protectedSetupRead(f, member, "protected_channels", c, nil), 401, string(commands.Unauthenticated))
	f.s.cfg.Seed = SeedNone
	f.code(t, "non sample", protectedSetupRead(f, root, "protected", c, nil), 501, string(commands.UnsupportedCapability))
}

func TestFinancialSetupProtectedRulesStates(t *testing.T) {
	cases := []struct{ point, state, observation string }{
		{"none", "none", "absent"}, {"before-propose", "none", "absent"}, {"after-propose", "registration_unknown", "pending"},
		{"after-propose-publication", "proposed", "pending"}, {"before-confirm", "proposed", "pending"}, {"after-confirm", "confirmation_unknown", "active"},
		{"after-confirm-publication", "confirmed", "active"}, {"unowned", "none", "unowned"},
	}
	for _, tc := range cases {
		t.Run(tc.point, func(t *testing.T) {
			f, reader, c := confirmedSetupConfiguration(t)
			a := f.s.financialIssuance
			switch {
			case tc.point == "unowned":
				if _, err := a.fields.Propose(SampleWorkspace, issuanceRules(), protected.Actor{ID: "foreign-governor", Kind: protected.KindHuman}); err != nil {
					t.Fatal(err)
				}
			case tc.point != "none":
				confirm := strings.Contains(tc.point, "confirm")
				var p financialProtectedRules
				var err error
				if confirm {
					p, err = proposeRulesAs(f, f.as(AdminLogin), "state-rules", issuanceRules())
					f.must(err)
				}
				f.s.fault = func(at string) bool { return at == "issuance-protected-rules-"+tc.point }
				if confirm {
					_, err = confirmRulesAs(f, f.as(Admin2Login), p)
				} else {
					_, err = proposeRulesAs(f, f.as(AdminLogin), "state-rules", issuanceRules())
				}
				documentFailure(t, err, commands.DependencyUnavailable)
				f.s.fault = nil
			}
			before := captureProtectedSetup(f)
			got := protectedSetupRead(f, reader, "protected", c, nil)
			if got.status != 200 || field(got, "rules_state") != tc.state || field(got, "rules_observation") != tc.observation || field(got, "channels") != "0" || field(got, "current_eligibility") != "unchecked" {
				t.Fatal(got.raw)
			}
			owned := tc.state != "none"
			if (field(got, "rules_digest") != "") != owned || (field(got, "rules_approvals") != "") != owned || (field(got, "rules_proposed_at") != "") != owned {
				t.Fatal("rule values do not follow ownership", got.raw)
			}
			if (field(got, "rules_confirmer") != "") != (tc.observation == "active") || field(got, "rules_pending_confirmer") != "" || (field(got, "rules_confirmed_at") != "") != (tc.observation == "active") {
				t.Fatal("confirmation facts differ", got.raw)
			}
			if tc.point == "unowned" && strings.Contains(got.raw, "foreign-governor") {
				t.Fatal("unowned rules disclosed")
			}
			if !reflect.DeepEqual(before, captureProtectedSetup(f)) {
				t.Fatal("rules read changed state")
			}
			f.s.fault = func(at string) bool { return at == "issuance-setup-protected-lookup" }
			f.code(t, "lookup fault is not absence", protectedSetupRead(f, reader, "protected", c, nil), 503, string(commands.DependencyUnavailable))
			f.s.fault = nil
			if owned {
				a.protectedRules.Reason = "changed"
				f.code(t, "changed retained rules", protectedSetupRead(f, reader, "protected", c, nil), 409, string(commands.EvidenceStale))
			}
		})
	}
}

func TestFinancialSetupProtectedChannelStates(t *testing.T) {
	cases := []struct{ point, state, observation string }{
		{"before-register", "", ""}, {"after-register", "registration_unknown", "registered"}, {"after-registration-publication", "registered", "registered"},
		{"before-admit", "registered", "registered"}, {"after-admit", "admission_unknown", "admitted"}, {"after-admission-publication", "admitted", "admitted"},
	}
	for _, tc := range cases {
		t.Run(tc.point, func(t *testing.T) {
			f, registrar, admitter, spec := configurationChannelWorld(t)
			grantSetupInspectionForTest(f, true)
			reader := f.as(MemberLogin)
			a := f.s.financialIssuance
			c := a.configurations[spec.Selector.Configuration.ID]
			admission := strings.Contains(tc.point, "admi")
			var p financialChannelRecord
			var err error
			if admission {
				p, err = registerChannelAs(f, registrar, spec)
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-channel-"+tc.point }
			if admission {
				_, err = admitChannelAs(f, admitter, p)
			} else {
				_, err = registerChannelAs(f, registrar, spec)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			before := captureProtectedSetup(f)
			got := protectedSetupRead(f, reader, "protected_channels", c, nil)
			if tc.state == "" {
				if got.status != 200 || field(got, "count") != "0" || field(got, "row_00") != "" {
					t.Fatal(got.raw)
				}
				return
			}
			row := protectedSetupRow(t, got, 0)
			retained := a.channels[spec.ID]
			if row["state"] != tc.state || row["observation"] != tc.observation || row["subject_kind"] != "document_kind" || row["registrar"] != retained.Registrar || row["admitter"] != retained.Module.Admitter || row["pending_admitter"] != map[bool]string{true: retained.Admitter}[retained.Module.Admitter == ""] || (row["admitted_at"] != "") != (tc.observation == "admitted") {
				t.Fatalf("row %v", row)
			}
			if (row["request_not_before"] != "") != (tc.state == "admitted") {
				t.Fatalf("maturity fact for %s: %v", tc.state, row["request_not_before"])
			}
			if !reflect.DeepEqual(before, captureProtectedSetup(f)) {
				t.Fatal("channel read changed state")
			}
			if _, err := a.fields.Retire(SampleWorkspace, spec.ID, protected.Actor{ID: "retiring-actor", Kind: protected.KindHuman}); err != nil {
				t.Fatal(err)
			}
			retired := protectedSetupRow(t, protectedSetupRead(f, reader, "protected_channels", c, nil), 0)
			if retired["state"] != tc.state || retired["observation"] != "retired" || retired["retired_by"] != "retiring-actor" || retired["retired_at"] == "" || retired["request_not_before"] != "" {
				t.Fatalf("retired row %v", retired)
			}
			if !reflect.DeepEqual(a.channels[spec.ID], retained) {
				t.Fatal("retirement observation rewrote retained channel")
			}
		})
	}
}

func TestFinancialSetupProtectedChannelScopeAndPaging(t *testing.T) {
	f, registrar, admitter, spec := configurationChannelWorld(t)
	grantSetupInspectionForTest(f, true)
	reader := f.as(MemberLogin)
	a := f.s.financialIssuance
	c := a.configurations[spec.Selector.Configuration.ID]
	for i := 1; i <= 11; i++ {
		s := spec
		s.ID = fmt.Sprintf("page-channel-%02d", i)
		_, err := registerChannelAs(f, registrar, s)
		f.must(err)
	}
	base, err := f.s.orgs.ContractRevision(SampleWorkspace, c.Spec.Contract, c.Spec.ContractRevision)
	f.must(err)
	base.Revision++
	f.must(f.s.orgs.AppendContractRevision(base))
	other := spec
	other.ID, other.Selector = "other-revision-channel", financialChannelSelector{Kind: "customer", Contract: base.ContractID, Revision: base.Revision}
	_, err = registerChannelAs(f, registrar, other)
	f.must(err)
	foreign := protected.Channel{ID: "module-only-channel", Subject: protectedSetupSubject(c, "entity"), Kind: protected.ChannelPhone, Ref: protected.Ref{ID: "module-only-reference", Fingerprint: strings.Repeat("c", 64)}, Evidence: "module-only-evidence"}
	if _, err := a.fields.Register(SampleWorkspace, foreign, protected.Actor{ID: "foreign-registrar", Kind: protected.KindHuman}); err != nil {
		t.Fatal(err)
	}

	summary := protectedSetupRead(f, reader, "protected", c, nil)
	if field(summary, "channels") != "11" || field(summary, "channels_other_provenance") != "1" || field(summary, "channels_unowned_module") != "1" {
		t.Fatal(summary.raw)
	}
	first := protectedSetupRead(f, reader, "protected_channels", c, nil)
	if first.status != 200 || field(first, "count") != "11" || field(first, "next") != "page-channel-10" || field(first, "row_10") != "" || field(first, "other_provenance") != "1" || field(first, "unowned_module") != "1" {
		t.Fatal(first.raw)
	}
	for _, hidden := range []string{"other-revision-channel", "module-only", "foreign-registrar"} {
		if strings.Contains(first.raw, hidden) || strings.Contains(summary.raw, hidden) {
			t.Fatalf("out-of-scope channel disclosed: %s", hidden)
		}
	}
	cursor := map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")}
	last := protectedSetupRead(f, reader, "protected_channels", c, cursor)
	if protectedSetupRow(t, last, 0)["channel"] != "page-channel-11" || field(last, "next") != "" || field(last, "row_01") != "" || field(last, "snapshot") != field(first, "snapshot") {
		t.Fatal(last.raw)
	}
	f.code(t, "unknown cursor", protectedSetupRead(f, reader, "protected_channels", c, map[string]string{"after": "other-revision-channel", "snapshot": cursor["snapshot"]}), 409, string(commands.EvidenceStale))
	f.code(t, "channel snapshot on changes", protectedSetupRead(f, reader, "protected_changes", c, map[string]string{"field": protected.FieldDocumentTemplate, "after": cursor["after"], "snapshot": cursor["snapshot"]}), 409, string(commands.EvidenceStale))

	grantSetupInspectionAs(f, SampleAdmin3)
	f.code(t, "other reader snapshot", protectedSetupRead(f, f.as(Admin3Login), "protected_channels", c, cursor), 409, string(commands.EvidenceStale))

	_, err = admitChannelAs(f, admitter, a.channels["page-channel-11"])
	f.must(err)
	f.code(t, "changed later row", protectedSetupRead(f, reader, "protected_channels", c, cursor), 409, string(commands.EvidenceStale))

	limits := f.s.cfg.Commands
	f.s.cfg.Commands.Value = 512
	f.code(t, "reduced transport", protectedSetupRead(f, reader, "protected_channels", c, nil), 501, string(commands.UnsupportedCapability))
	f.s.cfg.Commands = limits

	p := a.channels["page-channel-01"]
	p.Registrar = "changed-registrar"
	a.channels["page-channel-01"] = p
	f.code(t, "changed retained channel", protectedSetupRead(f, reader, "protected_channels", c, nil), 409, string(commands.EvidenceStale))
}
