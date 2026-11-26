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
				f.s.domain.Lock()
				err := f.s.admin.RevokeGrant(removed+"-"+subject, 1)
				f.s.domain.Unlock()
				if err != nil {
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
				for _, path := range []string{"", "/sample-charge-01"} {
					if got := f.candidateRead(root, path); got.status != candidateStatus {
						t.Fatalf("candidate %s: want %d got %d %s", path, candidateStatus, got.status, got.raw)
					}
				}
			})
		}
	}
}
