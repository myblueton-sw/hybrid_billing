package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"testing"
)

func TestServiceDeclarationCreateAndUpdateReplayAfterOrganizationSuspension(t *testing.T) {
	for _, command := range []string{"service.create", "service.declare"} {
		t.Run(command, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			target := servicesTarget
			fields := f.declaredFields(map[string]string{"name": "replayed", "owner": SampleAdmin})
			if command == "service.declare" {
				target = f.newService(root, "replayed")
				fields = f.declaredFields(map[string]string{"declaration_version": "1", "purpose": "Updated purpose"})
			}
			body := serviceDeclarationBody("original", "original-key", target, f.s.Revision(), fields)
			accepted := f.call(command, body, root.cookie, root.csrf)
			operation, ok := accepted.body["operation_id"].(string)
			if accepted.status != 200 || !ok || operation == "" {
				t.Fatal("initial declaration failed", accepted.status, accepted.body["code"])
			}
			party, version, err := f.s.orgs.Party(SampleWorkspace, "sample-le-seller")
			f.must(err)
			party.State = organizations.Suspended
			_, err = f.s.orgs.PutParty(party, version)
			f.must(err)
			before := f.s.Revision()
			record, known := f.s.serviceDeclaration(SampleWorkspace, "svc-replayed", 0)
			if !known {
				t.Fatal("accepted declaration missing")
			}
			replay := f.call(command, body, root.cookie, root.csrf)
			if replay.status != 200 || replay.body["replay"] != true || replay.body["operation_id"] != operation || f.s.Revision() != before {
				t.Fatal("accepted declaration was revalidated or repeated", replay.status, replay.body["code"])
			}
			after, _ := f.s.serviceDeclaration(SampleWorkspace, "svc-replayed", 0)
			if after.Version != record.Version {
				t.Fatal("replay appended declaration")
			}
		})
	}
}
