package server

import (
	"encoding/json"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func TestFinancialSetupMutationSourceDriftAndOriginalIdentity(t *testing.T) {
	f, root, _, _, fields := setupMutationWorld(t)
	a := f.s.financialIssuance
	f.s.fault = func(at string) bool { return at == "issuance-configuration-propose-after-intent" }
	first := draftRun(f, root, "issuance_setup.propose_configuration", "unknown-context", fields)
	f.code(t, "unknown intent", first, 500, string(commands.InternalError))
	f.s.fault = nil
	before := copyConfiguration(a.configurations["unknown-context"])
	for _, key := range []string{"region", "on", "languages"} {
		changed := maps.Clone(fields)
		switch key {
		case "region":
			changed[key] = "JP"
		case "on":
			changed[key] = "2026-11-17"
		case "languages":
			changed[key] = "en"
		}
		f.code(t, "changed original arguments", draftRun(f, root, "issuance_setup.propose_configuration", "unknown-context", changed), 409, string(commands.EvidenceStale))
	}
	var envelope map[string]any
	f.must(json.Unmarshal([]byte(chargeBody(f, "changed-reason", "unknown-context", fields)), &envelope))
	envelope["rationale"] = "Different rationale"
	raw, err := json.Marshal(envelope)
	f.must(err)
	f.code(t, "changed original rationale", f.call("issuance_setup.propose_configuration", string(raw), root.cookie, root.csrf), 409, string(commands.EvidenceStale))
	profile, v, err := f.s.orgs.BillingProfile(SampleWorkspace, before.Spec.BillingProfile)
	f.must(err)
	profile.State = organizations.Suspended
	_, err = f.s.orgs.PutBillingProfile(profile, v)
	f.must(err)
	revision := f.s.Revision()
	f.code(t, "changed source during unknown", draftRun(f, root, "issuance_setup.propose_configuration", "unknown-context", fields), 409, string(commands.EvidenceStale))
	if f.s.Revision() != revision || !reflect.DeepEqual(before, a.configurations[before.Spec.ID]) {
		t.Fatal("source drift replaced unknown intent")
	}
	if got := setupConfigurationsRead(f, root, before.Spec.Contract, "", ""); configurationPageRow(t, got, 0)["state"] != "proposal_unknown" {
		t.Fatal("unknown history lost")
	}
}

func TestFinancialSetupMutationCurrentRightsAndHold(t *testing.T) {
	for _, mode := range []string{"inspect", "proposer", "stepup", "hold"} {
		t.Run(mode, func(t *testing.T) {
			f, root, second, _, fields := setupMutationWorld(t)
			proposal := draftRun(f, root, "issuance_setup.propose_configuration", "rights-context", fields)
			f.ok(t, "proposal", proposal, "proposed", "")
			args := map[string]string{"digest": field(proposal, "digest")}
			body := chargeBody(f, "confirmation-replay", "rights-context", args)
			confirmed := f.call("issuance_setup.confirm_configuration", body, second.cookie, second.csrf)
			f.ok(t, "confirmed", confirmed, "confirmed", "")
			status, code := 403, string(commands.PermissionDenied)
			switch mode {
			case "inspect", "proposer":
				for _, g := range f.s.admin.Grants() {
					match := mode == "inspect" && g.SubjectID == SampleAdmin2 && g.Terms.Kind == protectedauthority.ResourceKind
					match = match || mode == "proposer" && g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation
					if match {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			case "stepup":
				f.advance(16 * time.Minute)
				code = CodeReauthentication
			case "hold":
				f.s.enterHold("Public configuration writes held")
				status, code = 503, string(commands.DependencyUnavailable)
			}
			before := copyConfiguration(f.s.financialIssuance.configurations["rights-context"])
			revision := f.s.Revision()
			f.code(t, "current boundary before cached receipt", f.call("issuance_setup.confirm_configuration", body, second.cookie, second.csrf), status, code)
			if f.s.Revision() != revision || !reflect.DeepEqual(before, f.s.financialIssuance.configurations["rights-context"]) {
				t.Fatal("denied replay changed setup")
			}
		})
	}
}
