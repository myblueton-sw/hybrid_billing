package pricingpolicy

import (
	"errors"
	"testing"
)

func TestRecordsAreSortedWorkspaceBoundedAndDefensive(t *testing.T) {
	f := setup(t)
	f.propose(t, "z")
	f.propose(t, "a")
	rows, err := f.r.Records("ws")
	if err != nil || len(rows) != 2 || rows[0].Proposal.ID != "a" || rows[1].Proposal.ID != "z" {
		t.Fatal(rows, err)
	}
	rows[0].Proposal.Spec.Policy.Rules[0].ID = "changed"
	rows[0].Proposal.Spec.Policy.Eligible[0] = "changed"
	original, err := f.r.Record("ws", "a")
	if err != nil || original.Proposal.Spec.Policy.Rules[0].ID == "changed" || original.Proposal.Spec.Policy.Eligible[0] == "changed" {
		t.Fatal("enumeration mutated records", err)
	}
	if rows, err := f.r.Records("other"); err != nil || len(rows) != 0 {
		t.Fatal("workspace leak", rows, err)
	}
	if _, err := f.r.Records(""); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}
