package server

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestResultAuthorityClientFixture(t *testing.T) {
	path := os.Getenv("HB_RESULT_AUTHORITY_CLIENT_FIXTURE")
	if path == "" {
		t.Skip("explicit synthetic authority client fixture output required")
	}
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	_ = f.as(MemberLogin)
	f.advance(500 * time.Millisecond)
	root := f.as(AdminLogin)
	console := provisionResultCeiling(t, f)
	fields := resultPermissionTimes(f)
	fields["recipient"], fields["ceiling"] = SampleMember, field(console, "grant")
	human := f.run(root, "result_authority.propose_human", "client-human", fields)
	f.ok(t, "human proposal", human, "pending", "")
	calls := map[string]reply{
		"history":    resultAuthorityReadCall(f, root, "history", nil),
		"recipients": resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "recipients", "capability": "apply"}),
		"ceilings":   resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"}),
		"inspection": resultPermissionRead(f, root, "client-human"),
	}
	output := map[string]json.RawMessage{}
	for key, call := range calls {
		if call.status != 200 {
			t.Fatalf("%s response status %d", key, call.status)
		}
		output[key] = json.RawMessage(call.raw)
	}
	raw, err := json.Marshal(output)
	f.must(err)
	f.must(os.WriteFile(path, raw, 0600))
}
