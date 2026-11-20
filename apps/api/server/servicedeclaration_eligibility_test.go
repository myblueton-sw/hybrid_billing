package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func TestServiceDeclarationEveryIncreaseRequiresCurrentResponsibleOwner(t *testing.T) {
	for name, change := range map[string]func(world, actor){
		"user state": func(f world, a actor) { f.must(f.s.identity.Installation().SuspendUser(seedOperator, SampleMember, 1)) },
		"membership": func(f world, a actor) {
			f.must(f.s.identity.Workspace().SuspendMembership(a.verifier(f), SampleWorkspace, SampleMember, 1))
		},
		"expired role": func(f world, a actor) {
			f.must(f.s.identity.Workspace().SetRolePeriod(a.verifier(f), SampleWorkspace, SampleMember, start.Add(time.Minute), 1, start))
			f.advance(time.Minute)
			f.sweep()
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root := f.as(AdminLogin)
			created := f.run(root, "service.create", servicesTarget, f.declaredFields(map[string]string{"name": "owned", "owner": SampleMember}))
			if created.status != 200 {
				t.Fatal("initial owner not eligible", created.status, created.body["code"])
			}
			svc := "svc-owned"
			f.issued(root, svc, "1")
			change(f, root)
			got := f.run(root, "service.issue_credential", svc, map[string]string{"days": "1"})
			f.failed(t, "owner lost before another issue", got, commands.PrerequisiteFailed)
			creds, err := f.s.identity.Workspace().Credentials(root.verifier(f), SampleWorkspace, svc)
			f.must(err)
			if len(creds) != 1 {
				t.Fatal("ineligible owner received another credential")
			}
			got = f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": "1"}))
			f.failed(t, "owner lost before declare", got, commands.PrerequisiteFailed)
			if row, _ := f.s.serviceDeclaration(SampleWorkspace, svc, 0); row.Version != 1 {
				t.Fatal("ineligible owner changed declaration")
			}
			got = f.run(root, "service.suspend", svc, nil)
			if got.status != 200 {
				t.Fatal("owner loss blocked reduction", got.status)
			}
			got = f.run(root, "service.reinstate", svc, nil)
			f.failed(t, "owner lost before reinstate", got, commands.PrerequisiteFailed)
			got = f.run(root, "service.retire", svc, nil)
			if got.status != 200 {
				t.Fatal("owner loss blocked retirement", got.status)
			}
		})
	}
}

func namedServiceCatalog(customer, legal, contract string) []byte {
	var data map[string]any
	json.Unmarshal(testServiceCatalog(), &data)
	scope := data["audiences"].([]any)[0].(map[string]any)["scopes"].([]any)[0].(map[string]any)
	for key, id := range map[string]string{"customer": customer, "legal_entity": legal, "contract": contract} {
		if id != "" {
			scope[key] = map[string]any{"id": id}
		}
	}
	encoded, _ := json.Marshal(data)
	return encoded
}

func TestServiceDeclarationNamedResourcesAreCanonicalAndCurrentlyEligible(t *testing.T) {
	for name, restriction := range map[string][3]string{"missing customer": {"missing", "", ""}, "internal unit is not customer": {"sample-unit-north-ops", "", ""}, "missing legal entity": {"", "missing", ""}, "missing contract": {"", "", "missing"}, "suspended contract": {"", "", "sample-contract-02"}, "ended contract": {"", "", "sample-contract-03"}, "not started contract": {"", "", "sample-contract-05"}, "mismatched customer": {"sample-le-south", "", "sample-contract-01"}} {
		t.Run(name, func(t *testing.T) {
			f := setupWith(t, func(c *Config) {
				c.ServiceCatalog = namedServiceCatalog(restriction[0], restriction[1], restriction[2])
			})
			root := f.as(AdminLogin)
			got := f.run(root, "service.create", servicesTarget, f.declaredFields(map[string]string{"name": "named", "owner": SampleAdmin}))
			f.failed(t, "invalid named scope", got, commands.InvalidRequest, "scopes")
			count, err := f.s.identity.Workspace().ServiceCount(root.verifier(f), SampleWorkspace)
			f.must(err)
			if count != 0 || len(f.s.serviceDeclarations) != 0 {
				t.Fatal("invalid scope left partial identity/declaration")
			}
		})
	}
	for name, restriction := range map[string][3]string{"customer": {"sample-le-north", "", ""}, "legal entity": {"", "sample-le-south", ""}, "contract": {"sample-le-north", "", "sample-contract-01"}} {
		t.Run("valid "+name, func(t *testing.T) {
			f := setupWith(t, func(c *Config) {
				c.ServiceCatalog = namedServiceCatalog(restriction[0], restriction[1], restriction[2])
			})
			root := f.as(AdminLogin)
			svc := f.newService(root, "valid")
			f.issued(root, svc, "1")
		})
	}
}

func TestServiceDeclarationRechecksOrganizationParentAndNamedScope(t *testing.T) {
	for _, named := range []bool{false, true} {
		t.Run("suspension", func(t *testing.T) {
			f := setupWith(t, func(c *Config) {
				if named {
					c.ServiceCatalog = namedServiceCatalog("", "sample-le-south", "")
				}
			})
			root := f.as(AdminLogin)
			organization := "sample-unit-north-ops"
			affected := "sample-le-north"
			if named {
				organization = "sample-le-seller"
				affected = "sample-le-south"
			}
			got := f.run(root, "service.create", servicesTarget, f.declaredFields(map[string]string{"name": "checks", "owner": SampleAdmin, "organization": organization}))
			if got.status != 200 {
				t.Fatal("initial declaration failed", got.status, got.body["code"])
			}
			party, version, err := f.s.orgs.Party(SampleWorkspace, affected)
			f.must(err)
			party.State = organizations.Suspended
			_, err = f.s.orgs.PutParty(party, version)
			f.must(err)
			before := f.s.Revision()
			got = f.run(root, "service.issue_credential", "svc-checks", map[string]string{"days": "1"})
			f.failed(t, "current reference lost eligibility", got, commands.PrerequisiteFailed)
			creds, err := f.s.identity.Workspace().Credentials(root.verifier(f), SampleWorkspace, "svc-checks")
			f.must(err)
			if len(creds) != 0 || f.s.Revision() != before {
				t.Fatal("reference failure left credential effects")
			}
			if record, known := f.s.serviceDeclaration(SampleWorkspace, "svc-checks", 1); !known || record.Organization != organization {
				t.Fatal("ineligibility destroyed historical declaration")
			}
		})
	}
}

func TestServiceDeclarationContractUsesAcceptanceDateNotFinalScheduledState(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.ServiceCatalog = namedServiceCatalog("sample-le-north", "", "sample-contract-01") })
	root := f.as(AdminLogin)
	sequence, err := f.s.lifecycles.Sequence(SampleWorkspace, "sample-contract-01")
	f.must(err)
	_, err = f.s.lifecycles.Amend(SampleWorkspace, "sample-contract-01", sequence, "future-suspend", nil, []contracts.Event{{ID: "scheduled-suspension", Kind: contracts.Suspend, From: contractDay(2026, 11, 17)}}, "")
	f.must(err)
	svc := f.newService(root, "timed")
	f.issued(root, svc, "1")
	f.advance(24 * time.Hour)
	root = f.as(AdminLogin)
	got := f.run(root, "service.issue_credential", svc, map[string]string{"days": "1"})
	f.failed(t, "current contract became suspended", got, commands.PrerequisiteFailed)
}
