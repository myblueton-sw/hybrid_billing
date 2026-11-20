package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceCreationRequiresAnAuthoritativeDeclaration(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	before := f.s.Revision()
	got := f.run(root, "service.create", servicesTarget, map[string]string{"name": "undeclared", "owner": SampleAdmin})
	if got.status != 422 || got.body["code"] != string(commands.InvalidRequest) {
		t.Fatal("service creation accepted missing required declaration", got.status, got.body["code"])
	}
	rows, err := f.s.identity.Workspace().ServiceUsers(root.verifier(f), SampleWorkspace, "", 100)
	f.must(err)
	if len(rows) != 0 || f.s.Revision() != before {
		t.Fatal("refused declaration changed identity or revision")
	}
}
