package server

import (
	"net/http"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestIssuanceStepHolderReadsSetupButNotContracts(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	j := confirmJourneyConfiguration(t, f)
	c := j.configuration
	j.grant(t, f, "issuance_protected_authority", protectedauthority.Inspect, SampleMember)
	j.grant(t, f, "issuance_term_authority", termsauthority.Inspect, SampleMember)
	member := f.as(MemberLogin)
	for _, path := range []string{"/api/v1/contracts", "/api/v1/contracts/" + c.Spec.Contract, "/api/v1/organizations"} {
		got := f.do("GET", path, "", func(r *http.Request) {
			r.AddCookie(member.cookie)
			r.Header.Set(csrfHeader, member.csrf)
		})
		f.code(t, "step holder reads "+path, got, 403, string(commands.PermissionDenied))
	}
	if got := setupContextRead(f, member, c.Spec.Contract, "2026-11-16"); got.status != 200 {
		t.Fatal("context", got.raw)
	}
	if got := documentRead(f, member, "issuance_setup.configurations", c.Spec.Contract, map[string]string{}); got.status != 200 {
		t.Fatal("configurations", got.raw)
	}
	if got := documentRead(f, member, "issuance_setup.options", c.Spec.Contract, map[string]string{"on": "2026-11-16", "scope_digest": field(setupContextRead(f, member, c.Spec.Contract, "2026-11-16"), "scope_digest")}); got.status != 200 {
		t.Fatal("options", got.raw)
	}
	for _, read := range []string{"protected", "protected_channels", "attestations", "references", "terms"} {
		if got := protectedSetupRead(f, member, read, c, nil); got.status != 200 {
			t.Fatal(read, got.raw)
		}
	}
	if got := protectedSetupRead(f, member, "protected_changes", c, map[string]string{"field": "payment_account"}); got.status != 200 {
		t.Fatal("protected_changes", got.raw)
	}
}
