package approvals

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestConfirmedPolicyHistoryAndDefensiveReads(t *testing.T) {
	r, _ := setup(t, 1)
	original := must(r.PolicyOf("ws", StageIssue))
	pending := policy(StageIssue, 2, 2, "result", "input")
	pending.ID = "history-policy-two"
	proposed := must(r.Propose("ws", pending, human("adm1")))
	if _, err := r.PolicyAt("ws", StageIssue, 2); !errors.Is(err, ErrNotFound) {
		t.Fatal("pending became policy history", err)
	}
	read := must(r.ProposalOf("ws", pending.ID))
	read.Proposal.Policy.Required[0] = "changed"
	if got := must(r.ProposalOf("ws", pending.ID)); !reflect.DeepEqual(got, proposed) {
		t.Fatal("proposal alias changed history")
	}
	list := must(r.Proposals("ws"))
	for i := range list {
		if i > 0 && list[i-1].Proposal.ID >= list[i].Proposal.ID {
			t.Fatal("unsorted proposals")
		}
		if len(list[i].Proposal.Policy.Required) > 0 {
			list[i].Proposal.Policy.Required[0] = "changed"
		}
	}
	must(r.Confirm("ws", pending.ID, human("adm2")))
	administer(t, r, policy(StageIssue, 3, 3, "result", "fx"))
	for rev, count := range map[int]int{1: 1, 2: 2, 3: 3} {
		p := must(r.PolicyAt("ws", StageIssue, rev))
		if p.Revision != rev || p.Approvals != count {
			t.Fatal(p)
		}
		p.Required[0] = "changed"
		if again := must(r.PolicyAt("ws", StageIssue, rev)); again.Required[0] == "changed" {
			t.Fatal("policy alias escaped")
		}
	}
	if got := must(r.PolicyAt("ws", StageIssue, 1)); !reflect.DeepEqual(got, original) {
		t.Fatal("historical policy changed", got)
	}
	if current := must(r.PolicyOf("ws", StageIssue)); current.Revision != 3 {
		t.Fatal("history rewound current policy")
	}
	for _, tc := range []struct {
		workspace, stage string
		revision         int
		want             error
	}{{"ws", StageCharge, 1, ErrNotFound}, {"foreign", StageIssue, 1, ErrNotFound}, {"ws", StageIssue, 4, ErrNotFound}, {"ws", StageIssue, 0, ErrInvalid}, {"ws", "other", 1, ErrInvalid}} {
		if _, err := r.PolicyAt(tc.workspace, tc.stage, tc.revision); !errors.Is(err, tc.want) {
			t.Fatal(tc, err)
		}
	}
	for _, ws := range []string{"foreign", ""} {
		if _, err := r.Proposals(ws); err == nil {
			t.Fatal("unknown workspace listed")
		}
		if _, err := r.ProposalOf(ws, pending.ID); err == nil {
			t.Fatal("unknown workspace proposal read")
		}
	}
	if _, err := r.ProposalOf("ws", "absent"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := r.ProposalOf("ws", ""); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}

func TestPolicyHistoryDoesNotAuthorizeOldRequest(t *testing.T) {
	r, _ := setup(t, 1)
	must(r.Request("ws", subject("req", "target", "15"), human("writer")))
	must(r.Decide("ws", "req", approve, human("ann")))
	expected := runOf("effect", "req", "target", "15")
	if err := r.CheckCurrent("ws", expected, human("runner")); err != nil {
		t.Fatal(err)
	}
	administer(t, r, policy(StageIssue, 2, 1, "result", "erp_draft"))
	if p := must(r.PolicyAt("ws", StageIssue, 1)); p.Validity != 24*time.Hour {
		t.Fatal(p)
	}
	if err := r.CheckCurrent("ws", expected, human("runner")); err == nil {
		t.Fatal("historical policy authorized superseded request")
	}
	if _, err := r.Execute("ws", expected, human("runner")); err == nil {
		t.Fatal("historical policy consumed execution")
	}
	if got := must(r.RequestOf("ws", "req")); got.State != StateSuperseded || len(got.Decisions) != 1 {
		t.Fatal(got)
	}
}

func TestPolicyHistoryRejectsInconsistentConfirmedDuplicates(t *testing.T) {
	r, _ := setup(t, 1)
	p := policy(StageIssue, 1, 2, "result")
	p.ID = "corrupted-history"
	r.workspaces["ws"].proposals[p.ID] = &ProposalView{Proposal: p, State: StateConfirmed}
	if _, err := r.PolicyAt("ws", StageIssue, 1); !errors.Is(err, ErrConflict) {
		t.Fatal("inconsistent history accepted", err)
	}
}
