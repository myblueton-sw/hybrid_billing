package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func serviceDeclarationBody(request, key, target, revision string, fields map[string]string) string {
	body, _ := json.Marshal(map[string]any{"request_id": request, "idempotency_key": key, "target_id": target, "expected_revision": revision, "rationale": "Synthetic declaration check", "body": fields})
	return string(body)
}

func TestServiceDeclarationsCreateUpdateRetainAndIsolateVersions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "metadata")
	first, known := f.s.serviceDeclaration(SampleWorkspace, svc, 0)
	if !known || first.Version != 1 || first.Organization != "sample-le-seller" || first.Catalog != f.s.serviceCatalog.Version() || len(first.Scopes) != 1 {
		t.Fatal("creation did not atomically record declaration")
	}
	first.Scopes[0].Scope = "changed-copy"
	again, _ := f.s.serviceDeclaration(SampleWorkspace, svc, 1)
	if again.Scopes[0].Scope != "read" {
		t.Fatal("caller modified retained declaration")
	}
	for version := uint64(1); version < 7; version++ {
		fields := f.declaredFields(map[string]string{"purpose": "변경 목적 " + strconv.FormatUint(version, 10), "declaration_version": strconv.FormatUint(version, 10)})
		got := f.run(root, "service.declare", svc, fields)
		if got.status != 200 || got.body["business_state"] != "service_declared" {
			t.Fatal("declaration update failed", got.status, got.body["code"])
		}
		result := got.body["fields"].(map[string]any)
		if len(result) != 1 || result["declaration_version"] != strconv.FormatUint(version+1, 10) {
			t.Fatal("declaration outcome exposed payload or wrong version")
		}
	}
	current, _ := f.s.serviceDeclaration(SampleWorkspace, svc, 0)
	if current.Version != 7 {
		t.Fatal("current declaration did not advance")
	}
	if _, known := f.s.serviceDeclaration(SampleWorkspace, svc, 2); known {
		t.Fatal("pruned version still available")
	}
	if _, known := f.s.serviceDeclaration(SampleWorkspace, svc, 3); !known {
		t.Fatal("retained boundary unavailable")
	}
	if _, known := f.s.serviceDeclaration("foreign", svc, 7); known {
		t.Fatal("declaration crossed workspace")
	}
	before := f.s.Revision()
	got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": "2"}))
	f.failed(t, "stale declaration", got, commands.RevisionConflict)
	if f.s.Revision() != before {
		t.Fatal("stale declaration changed revision")
	}
	f.run(root, "service.retire", svc, nil)
	got = f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": "7"}))
	f.failed(t, "retired declaration", got, commands.InvalidStage)
}

func TestServiceDeclarationValidationFailsBeforeIdentityEffects(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for name, change := range map[string]func(map[string]string){"purpose": func(m map[string]string) { m["purpose"] = " " }, "organization": func(m map[string]string) { m["organization"] = "foreign-org" }, "catalog": func(m map[string]string) { m["catalog_version"] = strings.Repeat("0", 64) }, "scope": func(m map[string]string) { m["scopes"] = `[{"audience":"test-service-api","scope":"missing"}]` }, "malformed scope": func(m map[string]string) { m["scopes"] = `null` }} {
		fields := f.declaredFields(map[string]string{"name": "refused", "owner": SampleAdmin})
		change(fields)
		before := f.s.Revision()
		got := f.run(root, "service.create", servicesTarget, fields)
		if got.status == 200 {
			t.Fatal("invalid declaration created identity", name)
		}
		count, err := f.s.identity.Workspace().ServiceCount(root.verifier(f), SampleWorkspace)
		f.must(err)
		if count != 0 || len(f.s.serviceDeclarations) != 0 || f.s.Revision() != before {
			t.Fatal("failed declaration left partial state", name)
		}
	}
}

func TestServiceDeclarationLegacyCompletionAndCapacity(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Commands = CommandLimits(); c.Commands.Burst = 1000 })
	root := f.as(AdminLogin)
	actor := root.verifier(f)
	for i := 0; i <= maxServices; i++ {
		id := fmt.Sprintf("svc-legacy-%03d", i)
		f.must(f.s.identity.Workspace().CreateServiceUser(actor, SampleWorkspace, id, SampleAdmin))
	}
	legacy := "svc-legacy-000"
	got := f.run(root, "service.issue_credential", legacy, map[string]string{"days": "1"})
	f.failed(t, "legacy has no declaration", got, commands.PrerequisiteFailed)
	for i := 0; i < maxServices; i++ {
		got := f.run(root, "service.declare", fmt.Sprintf("svc-legacy-%03d", i), f.declaredFields(map[string]string{"declaration_version": "0"}))
		if got.status != 200 {
			t.Fatal("legacy completion failed before declaration capacity", i, got.status, got.body["code"])
		}
	}
	before := f.s.Revision()
	got = f.run(root, "service.declare", "svc-legacy-200", f.declaredFields(map[string]string{"declaration_version": "0"}))
	f.failed(t, "declaration key capacity", got, commands.PrerequisiteFailed)
	if len(f.s.serviceDeclarations) != maxServices || f.s.Revision() != before {
		t.Fatal("capacity refusal changed state")
	}
	got = f.run(root, "service.declare", legacy, f.declaredFields(map[string]string{"declaration_version": "1"}))
	if got.status != 200 {
		t.Fatal("full key capacity prevented existing-key update", got.status, got.body["code"])
	}
}

func TestServiceDeclarationReplaySurvivesLaterIneligibilityWithoutAnotherEffect(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "recovery")
	body := serviceDeclarationBody("issue-original", "issue-key", svc, f.s.Revision(), map[string]string{"days": "1"})
	accepted := f.call("service.issue_credential", body, root.cookie, root.csrf)
	if accepted.status != 200 {
		t.Fatal("initial issue failed", accepted.status, accepted.body["code"])
	}
	operation, ok := accepted.body["operation_id"].(string)
	if !ok || operation == "" {
		t.Fatal("accepted issue has no operation identifier")
	}
	fields := accepted.body["fields"].(map[string]any)
	ref := fields["secret_ref"].(string)
	party, version, err := f.s.orgs.Party(SampleWorkspace, "sample-le-seller")
	f.must(err)
	party.State = organizations.Suspended
	_, err = f.s.orgs.PutParty(party, version)
	f.must(err)
	before := f.s.Revision()
	creds, err := f.s.identity.Workspace().Credentials(root.verifier(f), SampleWorkspace, svc)
	f.must(err)
	got := f.run(root, "service.issue_credential", svc, map[string]string{"days": "1"})
	f.failed(t, "new issue ineligible", got, commands.PrerequisiteFailed)
	replay := f.call("service.issue_credential", body, root.cookie, root.csrf)
	if replay.status != 200 || replay.body["replay"] != true || replay.body["operation_id"] != accepted.body["operation_id"] {
		t.Fatal("accepted issue could not replay truthfully", replay.status, replay.body["code"])
	}
	after, err := f.s.identity.Workspace().Credentials(root.verifier(f), SampleWorkspace, svc)
	f.must(err)
	if len(after) != len(creds) || f.s.Revision() != before {
		t.Fatal("refusal/replay repeated effects")
	}
	pickup := f.collect(root, ref)
	if pickup.status != 200 {
		t.Fatal("declaration ineligibility lost accepted secret pickup", pickup.status)
	}
	got = f.run(root, "service.suspend", svc, nil)
	if got.status != 200 {
		t.Fatal("reduction was blocked")
	}
	got = f.run(root, "service.reinstate", svc, nil)
	f.failed(t, "ineligible reinstate", got, commands.PrerequisiteFailed)
}

func TestServiceDeclarationVersionOverflowRefusesWithoutMutation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "overflow")
	key := serviceDeclarationKey{SampleWorkspace, svc}
	history := f.s.serviceDeclarations[key]
	history[len(history)-1].Version = math.MaxUint64
	before := f.s.Revision()
	got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": strconv.FormatUint(math.MaxUint64, 10)}))
	f.failed(t, "declaration version overflow", got, commands.PrerequisiteFailed)
	if f.s.Revision() != before || f.s.serviceDeclarations[key][len(history)-1].Version != math.MaxUint64 {
		t.Fatal("overflow mutated declaration")
	}
}
