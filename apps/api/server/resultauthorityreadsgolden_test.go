package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

var resultAuthorityGoldenSnapshots = map[string]string{
	"empty_history":   "00b44f95b48af4df8939e81eed8b4b9954b5ad58853b9eb9ba8acd3928ea61d3",
	"recipients":      "672a6d9e8906aa3c22f9541e23c8375bc8c9dc7a5be5cfdedf83fe9ff1aa56fb",
	"pending_history": "d354e99d344b1db544f50143b9a2ae0e01bf98eb14a4e5215c3c8c231e555ca0",
	"applied_history": "1481b1d9c6c935d8aba1bc8268f6f7b75af80104c5f38c41ce3fffae7c96d4fa",
	"ceilings":        "208f69fd372197686c601286569f874bcab34c98455fcccef5fa59c5baf313d1",
}

func TestResultAuthorityReadSnapshotsAreUnchanged(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	got := map[string]string{}
	got["empty_history"] = field(resultAuthorityReadCall(f, root, "history", nil), "snapshot")
	got["recipients"] = field(resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "recipients", "capability": "apply"}), "snapshot")
	ceiling := f.run(root, "result_authority.propose_console", "golden-ceiling", resultPermissionTimes(f))
	f.ok(t, "proposal", ceiling, inputauthority.Pending, "")
	got["pending_history"] = field(resultAuthorityReadCall(f, root, "history", nil), "snapshot")
	f.ok(t, "approval", resultPermissionApprove(f, second, "golden-ceiling", ceiling), inputauthority.Applied, "")
	got["applied_history"] = field(resultAuthorityReadCall(f, root, "history", nil), "snapshot")
	got["ceilings"] = field(resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"}), "snapshot")
	for name, value := range got {
		if !validPlanHash(value) || value != resultAuthorityGoldenSnapshots[name] {
			t.Errorf("%s snapshot %q", name, value)
		}
	}
	if len(got) != len(resultAuthorityGoldenSnapshots) {
		t.Fatal("golden snapshot set differs")
	}
}
