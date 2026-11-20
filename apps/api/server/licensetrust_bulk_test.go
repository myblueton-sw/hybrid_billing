package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
)

func TestLicenseTrustBulkImpactCountsPending(t *testing.T) {
	f := licenseWorld(t, nil)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
	fields, _ := trustKeyFields()
	targets := []string{}
	for i := 1; i <= 3; i++ {
		fields["not_after"] = fmt.Sprint(start.Add(time.Duration(i) * time.Hour).Unix())
		targets = append(targets, licenseProposalTarget(t, f.run(second, "licenses.propose_key", licenseTarget, fields)))
	}
	licenseProposalTarget(t, f.run(root, "licenses.propose_key", licenseTarget, fields))
	f.must(f.s.license.Close())
	f.s.license = nil
	f.must(f.s.openLicense())
	id, _ := f.prepared(t, root, "suspend_user", SampleAdmin2)
	read := f.bulkRead(root, "/"+id)
	found := false
	for _, value := range read.body["impact"].(map[string]any)["pending"].([]any) {
		row := value.(map[string]any)
		if row["store"] == "license_key_changes" {
			found = true
			if whole(row["count"]) != 3 || row["rechecked"] != false {
				t.Fatal(row)
			}
		}
	}
	if !found {
		t.Fatal("license proposals omitted from impact")
	}
	requireLicenseOutcome(t, f.run(root, "licenses.approve_key_change", targets[0], nil), licensemode.ProposalApplied)
	requireLicenseOutcome(t, f.run(second, "licenses.withdraw_key_change", targets[1], nil), licensemode.ProposalWithdrawn)
	proposers, err := f.s.license.PendingProposers()
	f.must(err)
	counts := map[string]int{}
	for _, id := range proposers {
		counts[id]++
	}
	if len(proposers) != 2 || counts[SampleAdmin2] != 1 || counts[SampleAdmin] != 1 {
		t.Fatal("pending inventory included closed proposals or lost a proposer")
	}
	proposers[0] = "mutated"
	again, err := f.s.license.PendingProposers()
	f.must(err)
	for _, id := range again {
		if id == "mutated" {
			t.Fatal("pending inventory shares mutable storage")
		}
	}
}
func TestLicenseTrustBulkFailureDoesNotPrepare(t *testing.T) {
	for _, failedInitialization := range []bool{false, true} {
		t.Run(fmt.Sprint(failedInitialization), func(t *testing.T) {
			f := licenseWorld(t, nil)
			root := f.as(AdminLogin)
			if failedInitialization {
				f.must(os.Mkdir(f.s.licenseDir(), 0700))
				f.code(t, "failed initialization", f.run(root, "licenses.initialize", licenseTarget, nil), 500, "internal_error")
			} else {
				requireLicenseOutcome(t, f.run(root, "licenses.initialize", licenseTarget, nil), "initialized")
				f.must(os.Remove(filepath.Join(f.s.licenseDir(), "state.json")))
			}
			staged := f.stage(root, "", SampleAdmin2)
			draft := field(staged, "draft")
			got := f.prepare(root, draft, "suspend_user")
			f.code(t, "unavailable impact", got, 500, "internal_error")
			if got.body["next_action"] != "inspect_operation" {
				t.Fatal(got.raw)
			}
			if _, err := f.s.bulk.DraftView(f.s.bootWorkspace, SampleAdmin, draft); err != nil {
				t.Fatal("failed impact changed the draft")
			}
		})
	}
}
