package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestFinancialDraftPublicContextIgnoresInactiveFutureBindingActor(t *testing.T) {
	f, member, _, spec, current := configuredDraftWorld(t)
	login := activateProtectedMember(f, "future-binding-author")
	grantAdditionalTermCapability(t, f, termsauthority.BindTerm, "future-binding-author")
	futureSpec := current.Spec
	futureSpec.ID, futureSpec.From, futureSpec.To = "future-context-binding", "2027-01-01", "2027-12-31"
	future, err := bindTermAs(f, f.as(login), futureSpec)
	f.must(err)
	future, err = approveBindingAs(f, f.as("setup-approver@example.invalid"), future)
	f.must(err)
	removed := 0
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == "future-binding-author" && grant.Terms.Kind == termsauthority.ResourceKind {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
			removed++
		}
	}
	if removed == 0 {
		t.Fatal("future binding author still authorized")
	}
	read := draftContextRead(f, member, spec.Application.FinalRequest, spec.Application.FinalDigest)
	assertDraftContext(t, read, map[string]string{"readiness": "ready", "submit_action": "ready", "binding_count": "1"})
	if f.s.financialIssuance.bindingIntents[future.Spec.ID].Digest != future.Digest {
		t.Fatal("context rewrote inactive binding")
	}
}
