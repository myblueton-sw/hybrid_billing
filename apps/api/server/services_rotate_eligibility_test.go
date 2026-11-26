package server

import (
	"testing"
	"time"
)

func (f world) rotateEligibility(a actor, credential string) (bool, string, map[string]any) {
	f.t.Helper()
	got := f.serviceQuery(a, "/api/v1/identity/service-credentials/"+credential)
	if got.status != 200 {
		f.t.Fatalf("credential detail %+v", got)
	}
	e := got.body["eligibility"].(map[string]any)
	can, _ := e["can_rotate"].(bool)
	state, _ := e["rotate_step_up_state"].(string)
	return can, state, got.body["row"].(map[string]any)
}

func TestCredentialDetailSaysWhetherTheCredentialCanBeRotated(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "eligible")
	old, _, _ := f.issued(root, svc, "30")
	if can, state, _ := f.rotateEligibility(root, old); !can || state != "ready" {
		t.Fatalf("a live credential of an active declared account: %v %s", can, state)
	}
	next, _, _ := f.rotated(root, old, map[string]string{"days": "5", "overlap_minutes": "60"})
	if can, _, row := f.rotateEligibility(root, old); can || row["rotation"] != rotationEnding || row["replaced_by"] != next {
		t.Errorf("a credential a rotation has bounded: %v %+v", can, row)
	}
	if can, _, row := f.rotateEligibility(root, next); !can || row["rotation"] != rotationReplacing || row["replaces"] != old {
		t.Errorf("the credential that replaced it: %v %+v", can, row)
	}
	if got := f.run(root, "service.rotate_credential", old, map[string]string{"days": "5"}); got.status == 200 {
		t.Errorf("the command accepted what the read said could not be rotated: %+v", got.body)
	}
	if got := f.run(root, "service.revoke_credential", next, nil); got.status != 200 {
		t.Fatalf("revoke %+v", got)
	}
	if can, _, _ := f.rotateEligibility(root, next); can {
		t.Error("a revoked credential")
	}
	other := f.newService(root, "suspended")
	held, _, _ := f.issued(root, other, "30")
	if got := f.run(root, "service.suspend", other, nil); got.status != 200 {
		t.Fatalf("suspend %+v", got)
	}
	if can, _, _ := f.rotateEligibility(root, held); can {
		t.Error("a credential of a suspended account")
	}
	if got := f.run(root, "service.rotate_credential", held, map[string]string{"days": "5"}); got.status == 200 {
		t.Errorf("the command rotated a credential of a suspended account: %+v", got.body)
	}
	full := f.newService(root, "full")
	first, _, _ := f.issued(root, full, "30")
	for n := 1; n < f.s.cfg.Identity.CredentialsPerServiceUser; n++ {
		f.issued(root, full, "30")
	}
	if can, _, _ := f.rotateEligibility(root, first); can {
		t.Error("a credential of an account at its credential limit")
	}
	if got := f.run(root, "service.rotate_credential", first, map[string]string{"days": "5"}); got.status == 200 {
		t.Errorf("the command rotated at the limit: %+v", got.body)
	}
	late := f.newService(root, "expired")
	short, _, _ := f.issued(root, late, "1")
	f.advance(25 * time.Hour)
	if can, _, _ := f.rotateEligibility(f.as(AdminLogin), short); can {
		t.Error("an expired credential")
	}
}
