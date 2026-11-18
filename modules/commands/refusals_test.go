package commands

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

var trustCodes = []Code{SelfApprovalDenied, AuthorityEpochConflict, ApprovalStale, EvidenceStale, IndependentApprovalRequired}

func TestDeclaredRefusals(t *testing.T) {
	f, ann := basic(t)
	declared := &stub{targets: map[string]*Target{"w1/t1": {CustomerID: "c1", Revision: "r1", AuthorID: "ann"}}}
	f.must(f.c.Register(Definition{ID: "statement.trusted", Permission: Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Rationale: true,
		Refusals: trustCodes, Handler: declared, Fields: []Field{{Name: "amount", Type: Decimal, Required: true}}}))
	for _, code := range trustCodes {
		rule := errorRules[code]
		declared.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(code) }
		call := adjust(ann, "k-"+string(code), "1")
		call.Command = "statement.trusted"
		got := f.code("a declared "+string(code), f.c.Execute(call), code)
		if got.Err.Status != rule.status || got.Err.Next != rule.next || got.Err.Operation != "" {
			t.Errorf("%s: %+v", code, got.Err)
		}
		f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(code) }
		undeclared := f.code("an undeclared "+string(code), f.c.Execute(adjust(ann, "u-"+string(code), "1")), InternalError)
		if undeclared.Err.Operation == "" {
			t.Errorf("%s: an undeclared code is not an operation failure %+v", code, undeclared.Err)
		}
	}
	if errorRules[SelfApprovalDenied] != (errorRule{403, false, NextNone}) {
		t.Errorf("self approval rule %+v", errorRules[SelfApprovalDenied])
	}
	for name, d := range map[string]Definition{
		"an unknown declaration":   {ID: "x.one", Permission: Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Refusals: []Code{InvalidStage}},
		"a repeated declaration":   {ID: "x.two", Permission: Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Refusals: []Code{ApprovalStale, ApprovalStale}},
		"self approval twice over": {ID: "x.three", Permission: Permission{Action: gate.Approve, Kind: "statement"}, Approval: true, Mutating: true, Refusals: []Code{SelfApprovalDenied}},
	} {
		if err := f.c.Register(d); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	document, err := f.c.OpenAPI("t", "1")
	f.must(err)
	var parsed map[string]any
	f.must(json.Unmarshal(document, &parsed))
	responses := func(command string) map[string]any {
		return parsed["paths"].(map[string]any)["/commands/"+command].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)
	}
	described := func(command, status string) string {
		return responses(command)[status].(map[string]any)["description"].(string)
	}
	for status, want := range map[string]string{"403": "permission_denied, self_approval_denied", "409": "approval_stale, authority_epoch_conflict, evidence_stale, idempotency_conflict, idempotency_key_expired, revision_conflict",
		"422": "independent_approval_required, invalid_request, invalid_stage, prerequisite_failed"} {
		if got := described("statement.trusted", status); got != want {
			t.Errorf("declared %s: %q", status, got)
		}
	}
	if got := described("statement.adjust", "409"); got != "idempotency_conflict, idempotency_key_expired, revision_conflict" {
		t.Errorf("undeclared 409: %q", got)
	}
	for _, e := range f.c.Catalogue() {
		if e.ID == "statement.trusted" && len(e.Refusals) != len(trustCodes) {
			t.Errorf("catalogue %+v", e)
		}
	}
}
