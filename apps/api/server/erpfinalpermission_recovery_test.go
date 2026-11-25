package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func TestERPFinalPermissionUnknownInactiveRecovery(t *testing.T) {
	for _, status := range []string{inputauthority.RevokedGrant, inputauthority.ExpiredGrant} {
		for _, request := range []bool{false, true} {
			name := "decision"
			if request {
				name = "request"
			}
			t.Run(name+"/"+status, func(t *testing.T) {
				f := setup(t, SeedSample).ready()
				f.setPassword(MemberLogin, pw)
				first, second := f.as(AdminLogin), f.as(Admin2Login)
				prefix, fault, grantPrefix, kind := "erp_final_permission", "erp-final-permission-after-grant", "erp-final-permission-", "propose"
				fields := financialPermissionFields(f)
				ceiling := ""
				if request {
					prefix, fault, grantPrefix, kind = "erp_final_request_permission", "erp-final-request-after-grant", "erp-final-request-", "propose_human"
					p := f.run(first, prefix+".propose_console", "console", requestPermissionTimes(f))
					f.ok(t, "ceiling proposal", p, inputauthority.Pending, "")
					c := finalPermissionApprove(t, f, second, prefix, "console", p)
					ceiling = field(c, "grant")
					fields["ceiling"] = ceiling
				}
				p := f.run(first, prefix+"."+kind, "uncertain", fields)
				f.ok(t, "proposal", p, inputauthority.Pending, "")
				digest := map[string]string{"digest": field(p, "digest")}
				f.s.fault = func(at string) bool { return at == fault }
				f.code(t, "interrupted application", f.run(second, prefix+".approve", "uncertain", digest), 500, "internal_error")
				f.s.fault = nil
				if status == inputauthority.ExpiredGrant {
					f.advance(13 * time.Hour)
					second = f.as(Admin2Login)
				} else {
					id := grantPrefix + field(p, "digest")
					if got := f.revokeVia(first, second, id); got.status != 202 {
						t.Fatal(got.raw)
					}
					if ceiling != "" {
						g, _ := f.s.grantView(ceiling)
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
				count := len(f.s.admin.Grants())
				recovered := finalPermissionApprove(t, f, second, prefix, "uncertain", p)
				if field(recovered, "recorded_grant_status") != status || len(f.s.admin.Grants()) != count {
					t.Fatal("recovery recreated inactive rights")
				}
			})
		}
	}
}
