package server

import (
	"encoding/json"
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
)

func TestERPFinalFinancialPermissionReplayRechecksModuleGovernor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	_, err := f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "add-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "add-governor", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	body, _ := json.Marshal(map[string]any{"request_id": "replay", "target_id": "replay", "idempotency_key": "permission-replay", "expected_revision": f.s.Revision(), "rationale": "Synthetic permission request", "body": financialPermissionFields(f)})
	f.ok(t, "proposal", f.call("erp_final_permission.propose", string(body), root.cookie, root.csrf), inputauthority.Pending, "")
	_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "remove-governor", Kind: approvals.ProposeRemoveAdmin, Admin: SampleAdmin}, approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "remove-governor", approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman})
	f.must(err)
	before := len(f.s.admin.Grants())
	f.code(t, "same-key replay after removal", f.call("erp_final_permission.propose", string(body), root.cookie, root.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "inspect after removal", documentRead(f, root, "erp_final_permission.inspect", "replay", map[string]string{}), 403, string(commands.PermissionDenied))
	if len(f.s.admin.Grants()) != before {
		t.Fatal("lost governor produced effect")
	}
}

func TestERPFinalRequestPermissionReplayRechecksModuleGovernor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	_, err := f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "add-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "add-governor", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	body, _ := json.Marshal(map[string]any{"request_id": "replay", "target_id": "replay", "idempotency_key": "permission-replay", "expected_revision": f.s.Revision(), "rationale": "Synthetic permission request", "body": requestPermissionTimes(f)})
	f.ok(t, "proposal", f.call("erp_final_request_permission.propose_console", string(body), root.cookie, root.csrf), inputauthority.Pending, "")
	_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "remove-governor", Kind: approvals.ProposeRemoveAdmin, Admin: SampleAdmin}, approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "remove-governor", approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman})
	f.must(err)
	before := len(f.s.admin.Grants())
	f.code(t, "same-key replay after removal", f.call("erp_final_request_permission.propose_console", string(body), root.cookie, root.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "inspect after removal", documentRead(f, root, "erp_final_request_permission.inspect", "replay", map[string]string{}), 403, string(commands.PermissionDenied))
	if len(f.s.admin.Grants()) != before {
		t.Fatal("lost governor produced effect")
	}
}
