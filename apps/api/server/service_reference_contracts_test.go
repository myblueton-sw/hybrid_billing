package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func assertServiceReferenceFailure(t *testing.T, err error, code commands.Code, field string) {
	t.Helper()
	var failure *commands.Failure
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatal("service reference validator returned another failure class")
	}
	if field != "" && (len(failure.Fields) != 1 || failure.Fields[0] != field) {
		t.Fatal("service reference validator attributed refusal to another field")
	}
}

func TestServiceImplicitContractPartiesRemainEligibleWithIndependentDeclaredOrganization(t *testing.T) {
	for _, affected := range []string{"sample-le-north", "sample-le-seller"} {
		t.Run(affected, func(t *testing.T) {
			f := setupWith(t, func(c *Config) { c.ServiceCatalog = namedServiceCatalog("", "", "sample-contract-01") })
			root := f.as(AdminLogin)
			fields := f.declaredFields(map[string]string{"name": "healthy-reference", "owner": SampleAdmin, "organization": "sample-le-south"})
			if got := f.run(root, "service.create", servicesTarget, fields); got.status != 200 {
				t.Fatal("healthy implicit contract reference was not accepted", got.status)
			}
			f.issued(root, "svc-healthy-reference", "1")
			party, version, err := f.s.orgs.Party(SampleWorkspace, affected)
			f.must(err)
			party.State = organizations.Suspended
			_, err = f.s.orgs.PutParty(party, version)
			f.must(err)
			if _, _, err := f.s.orgs.EligibleParty(SampleWorkspace, "sample-le-south"); err != nil {
				t.Fatal("independently active declaration organization premise")
			}
			before, declarations, pickups := f.s.Revision(), len(f.s.serviceDeclarations), len(f.s.dir.pickups)
			fields["name"] = "refused-reference"
			got := f.run(root, "service.create", servicesTarget, fields)
			if got.status != 422 || got.body["code"] != string(commands.InvalidRequest) {
				t.Fatal("implicit contract party ineligibility was not refused before effects")
			}
			f.failed(t, "current implicit contract party", got, commands.InvalidRequest, "scopes")
			count, err := f.s.identity.Workspace().ServiceCount(root.verifier(f), SampleWorkspace)
			f.must(err)
			credentials, err := f.s.identity.Workspace().Credentials(root.verifier(f), SampleWorkspace, "svc-healthy-reference")
			f.must(err)
			if count != 1 || len(credentials) != 1 || len(f.s.serviceDeclarations) != declarations || len(f.s.dir.pickups) != pickups || f.s.Revision() != before {
				t.Fatal("refused implicit contract party changed existing or new service effects")
			}
		})
	}
}

func TestServiceLifecycleRevisionDriftRefusesBeforeServiceEffects(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.ServiceCatalog = namedServiceCatalog("", "", "sample-contract-01") })
	root := f.as(AdminLogin)
	sequence, err := f.s.lifecycles.Sequence(SampleWorkspace, "sample-contract-01")
	f.must(err)
	_, err = f.s.lifecycles.Amend(SampleWorkspace, "sample-contract-01", sequence, "controlled-reference-drift", nil, []contracts.Event{{ID: "absent-party-revision", Kind: contracts.Change, From: contractToday(start), Revision: 99}}, "")
	f.must(err)
	position, _, err := f.s.lifecycles.CurrentPosition(SampleWorkspace, "sample-contract-01", contractToday(start))
	f.must(err)
	if position.State != contracts.Active || position.Revision != 99 {
		t.Fatal("controlled public-module revision drift premise")
	}
	before, pickups := f.s.Revision(), len(f.s.dir.pickups)
	got := f.run(root, "service.create", servicesTarget, f.declaredFields(map[string]string{"name": "drift-refusal", "owner": SampleAdmin}))
	if got.status != 503 || got.body["code"] != string(commands.DependencyUnavailable) {
		t.Fatal("unavailable contract organization revision was not dependency refusal", got.status)
	}
	count, err := f.s.identity.Workspace().ServiceCount(root.verifier(f), SampleWorkspace)
	f.must(err)
	if count != 0 || len(f.s.serviceDeclarations) != 0 || len(f.s.dir.pickups) != pickups || f.s.Revision() != before {
		t.Fatal("controlled reference drift left partial service effects")
	}
}

func TestServiceConfiguredTermsPreserveTenantAndReferenceDateBoundaries(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.ServiceCatalog = namedServiceCatalog("", "", "sample-contract-05") })
	local, err := f.s.serviceCatalog.Scope(SampleWorkspace, "test-service-api", "read")
	f.must(err)
	later := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	position, _, err := f.s.lifecycles.CurrentPosition(SampleWorkspace, "sample-contract-05", contractToday(later))
	f.must(err)
	parties, err := f.s.orgs.ContractParties(SampleWorkspace, "sample-contract-05", position.Revision)
	f.must(err)
	if position.State != contracts.Active || parties.SellerID != "" {
		t.Fatal("actual active sellerless contract premise")
	}
	if err := f.s.checkServiceTerms(local.Terms, later); err != nil {
		t.Fatal("valid actual sellerless terms were refused")
	}
	foreign, err := f.s.serviceCatalog.Scope(EmptyWorkspace, "test-service-api", "read")
	f.must(err)
	if foreign.Terms.Scope.WorkspaceID != EmptyWorkspace {
		t.Fatal("actually configured foreign scope premise")
	}
	assertServiceReferenceFailure(t, f.s.checkServiceTerms(foreign.Terms, later), commands.InvalidRequest, "scopes")
	unrepresentable := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	assertServiceReferenceFailure(t, f.s.checkServiceTerms(local.Terms, unrepresentable), commands.DependencyUnavailable, "")
}

func TestServiceDeclarationHelperRejectsMalformedOrganizationBeforePreparation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	before := f.s.Revision()
	fields := f.declaredFields(map[string]string{"organization": "malformed!"})
	_, err := f.s.prepareServiceDeclaration("svc-helper-contract", fields, 0, start)
	assertServiceReferenceFailure(t, err, commands.InvalidRequest, "organization")
	if len(f.s.serviceDeclarations) != 0 || f.s.Revision() != before {
		t.Fatal("helper validation published a declaration or revision")
	}
}
