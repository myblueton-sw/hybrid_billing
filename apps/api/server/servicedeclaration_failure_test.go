package server

import (
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceDeclarationDependencyFailureLeavesAllEffectsUntouched(t *testing.T) {
	for _, fault := range []string{"roster", "services", "organizations", "service-declaration", "contract-read"} {
		for _, command := range []string{"service.create", "service.declare", "service.issue_credential", "service.reinstate"} {
			t.Run(fault+"/"+command, func(t *testing.T) {
				f := setupWith(t, func(c *Config) { c.ServiceCatalog = namedServiceCatalog("sample-le-north", "", "sample-contract-01") })
				root := f.as(AdminLogin)
				svc := f.newService(root, "existing")
				if command == "service.reinstate" {
					if got := f.run(root, "service.suspend", svc, nil); got.status != 200 {
						t.Fatal("suspend premise", got.status)
					}
				}
				target := svc
				fields := map[string]string{"days": "1"}
				switch command {
				case "service.create":
					target = servicesTarget
					fields = f.declaredFields(map[string]string{"name": "new", "owner": SampleAdmin})
				case "service.declare":
					fields = f.declaredFields(map[string]string{"declaration_version": "1"})
				case "service.reinstate":
					fields = map[string]string{}
				}
				before := f.s.Revision()
				f.s.fault = func(at string) bool { return at == fault }
				got := f.runWith(root, before, command, target, fields)
				f.failed(t, "dependency must fail closed", got, commands.DependencyUnavailable)
				f.s.fault = nil
				if f.s.Revision() != before || len(f.s.serviceDeclarations) != 1 {
					t.Fatal("dependency failure changed revision/declaration keys")
				}
				record, _ := f.s.serviceDeclaration(SampleWorkspace, svc, 0)
				if record.Version != 1 {
					t.Fatal("dependency failure revised declaration")
				}
				count, err := f.s.identity.Workspace().ServiceCount(root.verifier(f), SampleWorkspace)
				f.must(err)
				credentials, err := f.s.identity.Workspace().Credentials(root.verifier(f), SampleWorkspace, svc)
				f.must(err)
				if count != 1 || len(credentials) != 0 || len(f.s.dir.pickups) != 0 {
					t.Fatal("dependency failure left identity/credential/pickup effects")
				}
				if command == "service.reinstate" {
					row, err := f.s.identity.Workspace().ServiceUser(root.verifier(f), SampleWorkspace, svc)
					f.must(err)
					if serviceState(row.State) != "suspended" {
						t.Fatal("dependency failure reactivated service")
					}
				}
			})
		}
	}
}

func TestServiceDeclarationRejectsMalformedVersionsAndUnscopedOrganization(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "versions")
	for _, version := range []string{"-1", "01", "18446744073709551616"} {
		got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": version}))
		f.failed(t, "noncanonical declaration version", got, commands.InvalidRequest, "declaration_version")
	}
	for _, catalog := range []string{"short", strings.Repeat("A", 64)} {
		got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": "1", "catalog_version": catalog}))
		f.failed(t, "malformed catalog fingerprint", got, commands.InvalidRequest, "catalog_version")
	}
	got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": "1", "organization": "bad organization"}))
	f.failed(t, "noncanonical organization", got, commands.InvalidRequest, "organization")
	row, _ := f.s.serviceDeclaration(SampleWorkspace, svc, 0)
	if row.Version != 1 {
		t.Fatal("invalid command revised declaration")
	}
}
