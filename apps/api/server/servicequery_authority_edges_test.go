package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func TestServiceQueryRevokedCallerCannotPublishProjectedDetail(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "revoked-reader")
	people, err := f.s.identity.Workspace().People(root.verifier(f), SampleWorkspace, []string{SampleAdmin})
	f.must(err)
	if len(people) != 1 {
		t.Fatal("reader identity premise")
	}
	revoked := false
	f.s.fault = func(at string) bool {
		if at == "service-declaration" && !revoked {
			revoked = true
			f.must(f.s.identity.Installation().SuspendUser(seedOperator, SampleAdmin, people[0].User.Epoch))
		}
		return false
	}
	got := f.serviceQuery(root, "/api/v1/identity/services/"+service)
	if !revoked || got.status != 401 || got.body["row"] != nil || got.body["eligibility"] != nil || got.body["admission"] != nil {
		t.Fatal("caller lost authority but projected detail was published", got.status, got.body["code"])
	}
}

func TestServiceQueryUnavailableEvidencePreservesOnlySafeReductions(t *testing.T) {
	for _, dependency := range []string{"service-declaration", "roster"} {
		t.Run(dependency, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			service := f.newService(root, "evidence-loss")
			f.s.fault = func(at string) bool { return at == dependency }
			got := f.serviceQuery(root, "/api/v1/identity/services/"+service)
			if got.status != 200 {
				t.Fatal("known account should retain truthful unavailable evidence", got.status)
			}
			flags := got.body["eligibility"].(map[string]any)
			if flags["state"] != "unavailable" || flags["can_issue"] != false || flags["can_declare"] != false || flags["can_reinstate"] != false || flags["can_suspend"] != true || flags["can_retire"] != true {
				t.Fatal("unavailable evidence widened actions or prevented safe reductions")
			}
			declaration := got.body["row"].(map[string]any)["declaration"].(map[string]any)
			if dependency == "service-declaration" && (declaration["state"] != "unavailable" || declaration["purpose"] != "" || len(declaration["scopes"].([]any)) != 0) {
				t.Fatal("unavailable declaration was invented or confused with a known empty grant")
			}
		})
	}
}

func TestServiceQuerySuspendedOwnerDoesNotBlockProtectiveReductions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	created := f.run(root, "service.create", servicesTarget, f.declaredFields(map[string]string{"name": "owner-changed", "owner": SampleMember}))
	if created.status != 200 {
		t.Fatal("designated owner premise", created.status)
	}
	people, err := f.s.identity.Workspace().People(root.verifier(f), SampleWorkspace, []string{SampleMember})
	f.must(err)
	f.must(f.s.identity.Installation().SuspendUser(seedOperator, SampleMember, people[0].User.Epoch))
	got := f.serviceQuery(root, "/api/v1/identity/services/svc-owner-changed")
	if got.status != 200 {
		t.Fatal("canonical owner-loss observation", got.status)
	}
	flags := got.body["eligibility"].(map[string]any)
	if flags["state"] != "ineligible" || flags["can_issue"] != false || flags["can_declare"] != false || flags["can_reinstate"] != false || flags["can_suspend"] != true || flags["can_retire"] != true {
		t.Fatal("designated owner confused with current eligibility")
	}
	row := got.body["row"].(map[string]any)
	if row["owner"] != SampleMember || row["state"] != "active" {
		t.Fatal("read silently transferred ownership or changed lifecycle")
	}
}

func TestServiceQueryIneligibleOrganizationStillAllowsDeclarationRepair(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "organization-changed")
	party, version, err := f.s.orgs.Party(SampleWorkspace, "sample-le-seller")
	f.must(err)
	party.State = organizations.Suspended
	_, err = f.s.orgs.PutParty(party, version)
	f.must(err)
	got := f.serviceQuery(root, "/api/v1/identity/services/"+service)
	if got.status != 200 {
		t.Fatal("current declaration prerequisite observation", got.status)
	}
	flags := got.body["eligibility"].(map[string]any)
	if flags["state"] != "ineligible" || flags["can_issue"] != false || flags["can_declare"] != true || flags["can_suspend"] != true || flags["can_retire"] != true {
		t.Fatal("ineligible declaration blocked repair or widened issuance")
	}
	refused := f.run(root, "service.issue_credential", service, map[string]string{"days": "1"})
	if refused.status != 422 || refused.body["code"] != string(commands.PrerequisiteFailed) {
		t.Fatal("ineligible declaration allowed issuance or returned the wrong refusal", refused.status)
	}
	f.assertServiceQueryIDs(root, "/api/v1/identity/service-credentials?service="+service, nil)
}
