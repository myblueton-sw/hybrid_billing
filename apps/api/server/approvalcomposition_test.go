package server

import "testing"

func TestApprovalCandidateGrantSeparation(t *testing.T) {
	for _, subject := range []string{SampleAdmin, "console"} {
		for _, removed := range []string{"approval", "candidate"} {
			t.Run(subject+"/"+removed, func(t *testing.T) {
				f := setup(t, SeedSample).ready()
				defer f.s.Close()
				root := f.as(AdminLogin)
				if got := f.approvalRead(root, "view=pending"); got.status != 200 {
					t.Fatal("approval baseline", got.raw)
				}
				if got := f.candidateRead(root, ""); got.status != 200 {
					t.Fatal("candidate baseline", got.raw)
				}
				if err := f.s.admin.RevokeGrant(removed+"-"+subject, 1); err != nil {
					t.Fatal(err)
				}
				approvalStatus, candidateStatus := 200, 200
				if removed == "approval" {
					approvalStatus = 403
				} else {
					candidateStatus = 403
				}
				if got := f.approvalRead(root, "view=pending"); got.status != approvalStatus {
					t.Fatalf("approval: want %d got %d %s", approvalStatus, got.status, got.raw)
				}
				if got := f.candidateRead(root, ""); got.status != candidateStatus {
					t.Fatalf("candidate: want %d got %d %s", candidateStatus, got.status, got.raw)
				}
			})
		}
	}
}
