package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func setupConfigurationsRead(f world, who actor, contract, after, snapshot string) reply {
	return documentRead(f, who, "issuance_setup.configurations", contract, map[string]string{"after": after, "snapshot": snapshot})
}
func configurationPageRow(t *testing.T, got reply, n int) map[string]string {
	t.Helper()
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	var row map[string]string
	if err := json.Unmarshal([]byte(field(got, fmt.Sprintf("row_%02d", n))), &row); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestFinancialSetupConfigurationsInterruptedHistory(t *testing.T) {
	for _, point := range []string{"proposed", "confirmed", "propose-after-intent", "propose-after-module", "confirm-after-intent", "confirm-after-module"} {
		t.Run(point, func(t *testing.T) {
			f, reader := setupContextWorld(t)
			first, second := f.as(AdminLogin), f.as(Admin2Login)
			a := f.s.financialIssuance
			if point == "propose-after-intent" || point == "propose-after-module" {
				f.s.fault = func(at string) bool { return at == "issuance-configuration-"+point }
			}
			p, err := proposeConfigurationAs(f, first, configurationSpec())
			if point == "propose-after-intent" || point == "propose-after-module" {
				documentFailure(t, err, commands.DependencyUnavailable)
			} else {
				f.must(err)
			}
			f.s.fault = nil
			if point == "confirmed" || point == "confirm-after-intent" || point == "confirm-after-module" {
				if point != "confirmed" {
					f.s.fault = func(at string) bool { return at == "issuance-configuration-"+point }
				}
				_, err = confirmConfigurationAs(f, second, p)
				if point == "confirmed" {
					f.must(err)
				} else {
					documentFailure(t, err, commands.DependencyUnavailable)
				}
				f.s.fault = nil
			}
			retained := copyConfiguration(a.configurations[configurationSpec().ID])
			revision := f.s.Revision()
			got := setupConfigurationsRead(f, reader, configurationSpec().Contract, "", "")
			row := configurationPageRow(t, got, 0)
			if row["state"] != retained.State || row["configuration"] != retained.Spec.ID || field(got, "count") != "1" || field(got, "current_eligibility") != "unchecked" {
				t.Fatal("incorrect historical projection", got.raw)
			}
			if point == "propose-after-intent" && (row["module_observation"] != "absent" || row["proposed_at"] != "") {
				t.Fatal("invented proposal observation")
			}
			if point == "confirm-after-module" && (row["module_observation"] != "active" || row["activated_at"] != "") {
				t.Fatal("unknown confirmation promoted")
			}
			profile, v, err := f.s.orgs.BillingProfile(SampleWorkspace, retained.Spec.BillingProfile)
			f.must(err)
			profile.State = organizations.Suspended
			_, err = f.s.orgs.PutBillingProfile(profile, v)
			f.must(err)
			for _, grant := range f.s.admin.Grants() {
				if grant.SubjectID == SampleAdmin {
					f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
				}
			}
			f.s.contracts = nil
			f.s.enterHold("Historical setup inspection")
			historical := setupConfigurationsRead(f, reader, retained.Spec.Contract, "", "")
			if historical.status != 200 || field(historical, "row_00") != field(got, "row_00") || field(historical, "snapshot") != field(got, "snapshot") {
				t.Fatal("live standing rewrote history", historical.raw)
			}
			if f.s.Revision() != revision || !reflect.DeepEqual(retained, a.configurations[retained.Spec.ID]) {
				t.Fatal("read reconciled configuration")
			}
		})
	}
}

func TestFinancialSetupConfigurationsPagination(t *testing.T) {
	f, reader := setupContextWorld(t)
	root := f.as(AdminLogin)
	base, err := f.s.orgs.ContractRevision(SampleWorkspace, "sample-contract-01", 2)
	f.must(err)
	for i := 1; i <= 11; i++ {
		if i > 2 {
			base.Revision = uint64(i)
			f.must(f.s.orgs.AppendContractRevision(base))
		}
		spec := configurationSpec()
		spec.ID = fmt.Sprintf("history-%02d", i)
		spec.ContractRevision = uint64(i)
		_, err := proposeConfigurationAs(f, root, spec)
		f.must(err)
	}
	first := setupConfigurationsRead(f, reader, base.ContractID, "", "")
	if first.status != 200 || field(first, "count") != "11" || field(first, "next") != "history-10" || field(first, "row_10") != "" {
		t.Fatal(first.raw)
	}

	otherSpec := protectedPermissionSpec(f, "inspect")
	otherSpec.Kind, otherSpec.Recipient = inputauthority.Human, SampleAdmin3
	for id := range f.s.protectedAuthority.ceilings {
		otherSpec.CeilingID = id
	}
	proposed, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "other-history-reader", Spec: otherSpec, Rationale: "Separate authorized history viewer"})
	f.must(err)
	_, err = protectedPermissionDecision(f, f.as(Admin2Login), proposed)
	f.must(err)
	other := f.as(Admin3Login)
	f.code(t, "other viewer snapshot", setupConfigurationsRead(f, other, base.ContractID, field(first, "next"), field(first, "snapshot")), 409, string(commands.EvidenceStale))
	if got := setupConfigurationsRead(f, other, base.ContractID, "", ""); got.status != 200 {
		t.Fatal(got.raw)
	}
	last := setupConfigurationsRead(f, reader, base.ContractID, field(first, "next"), field(first, "snapshot"))
	if configurationPageRow(t, last, 0)["configuration"] != "history-11" || field(last, "next") != "" || field(last, "row_01") != "" {
		t.Fatal(last.raw)
	}
	empty := setupConfigurationsRead(f, reader, "sample-contract-02", "", "")
	if empty.status != 200 || field(empty, "count") != "0" || field(empty, "row_00") != "" {
		t.Fatal("foreign contract disclosed", empty.raw)
	}
	f.code(t, "foreign snapshot", setupConfigurationsRead(f, reader, "sample-contract-02", "history-10", field(first, "snapshot")), 409, string(commands.EvidenceStale))
	f.code(t, "cursor requires snapshot", setupConfigurationsRead(f, reader, base.ContractID, "history-10", ""), 422, string(commands.InvalidRequest))
	p := f.s.financialIssuance.configurations["history-11"]
	_, err = confirmConfigurationAs(f, f.as(Admin2Login), p)
	f.must(err)
	f.code(t, "changed later row", setupConfigurationsRead(f, reader, base.ContractID, "history-10", field(first, "snapshot")), 409, string(commands.EvidenceStale))
}

func TestFinancialSetupConfigurationsRejectCorruption(t *testing.T) {
	for _, mode := range []string{"digest", "scope", "parent", "document", "orphan", "duplicate", "state", "module_missing", "module_unobserved", "module_changed", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			f, reader := setupContextWorld(t)
			p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
			f.must(err)
			a := f.s.financialIssuance
			switch mode {
			case "digest":
				p.Digest = "changed"
			case "scope":
				p.Scope.Contract.Revision++
			case "parent":
				p.Scope.Parties[0].Party.ParentID = "unrelated-parent"
			case "document":
				p.Scope.Document.Currency = "EUR"
			case "orphan":
				a.configurationScopes["orphan"] = "absent"
			case "duplicate":
				a.configurationScopes["duplicate"] = p.Spec.ID
			case "state":
				p.State = "confirmation_unknown"
			case "module_changed":
				_, err = a.documents.ApproveKind(SampleWorkspace, p.Kind.ID, issuance.Actor{ID: SampleAdmin2, Kind: issuance.KindHuman})
				f.must(err)
			case "module_missing":
				a.documents = setup(t, SeedSample).ready().s.financialIssuance.documents
			case "module_unobserved":
				p.State = "proposal_unknown"
				p.Proposed = financialIssuanceConfiguration{}.Proposed
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-configuration-lookup" }
			}
			if mode == "scope" || mode == "parent" || mode == "document" {
				p.Digest, err = configurationDigest(p)
				f.must(err)
			}
			a.configurations[p.Spec.ID] = copyConfiguration(p)
			status, code := 409, commands.EvidenceStale
			if mode == "lookup" {
				status, code = 503, commands.DependencyUnavailable
			}
			got := setupConfigurationsRead(f, reader, p.Spec.Contract, "", "")
			f.code(t, mode, got, status, string(code))
			if got.body["fields"] != nil || !reflect.DeepEqual(a.configurations[p.Spec.ID], p) {
				t.Fatal("corrupt read disclosed or changed state")
			}
		})
	}
}

func TestFinancialSetupConfigurationsReaderAndBounds(t *testing.T) {
	f, reader := setupContextWorld(t)
	root := f.as(AdminLogin)
	f.code(t, "administrator without Inspect", setupConfigurationsRead(f, root, "sample-contract-01", "", ""), 403, string(commands.PermissionDenied))
	_, err := proposeConfigurationAs(f, root, configurationSpec())
	f.must(err)
	f.s.cfg.Commands.Value = 4095
	f.code(t, "unsupported projection budget", setupConfigurationsRead(f, reader, "sample-contract-01", "", ""), 501, string(commands.UnsupportedCapability))
	f.s.cfg.Commands.Value = 4096
	f.s.enterHold("Retained configuration held read")
	if got := setupConfigurationsRead(f, reader, "sample-contract-01", "", ""); got.status != 200 {
		t.Fatal(got.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked inspection", setupConfigurationsRead(f, reader, "sample-contract-01", "", ""), 403, string(commands.PermissionDenied))
}
