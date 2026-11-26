package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestCapturedSessionProjectionTracksActualTrustAgeAndRoleTransitions(t *testing.T) {
	for _, transition := range []string{"trust", "age", "role"} {
		t.Run(transition, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			policy := authpolicy.Revision{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAge: time.Hour}
			if transition == "age" {
				policy.MaxAge = time.Minute
			}
			if transition == "role" {
				policy.MFA = authpolicy.MFAAdministrators
			}
			f.directPolicy(policy)
			old := f.signedIn(MemberLogin, pw)
			caller := f.signedIn(MemberLogin, pw)
			captured, known := f.s.sessions.get(verifierOfActor(old))
			if !known {
				t.Fatal("real captured session premise")
			}
			before := f.s.sessionView(captured, verifierOfActor(caller))
			if before.State != "active" || !before.CanEnd || before.Current {
				t.Fatal("fresh captured peer was not active")
			}
			want := "revoked"
			switch transition {
			case "trust":
				f.directPolicy(policy)
			case "age":
				f.advance(2 * time.Minute)
				want = "expired"
			case "role":
				root := f.as(AdminLogin)
				people, err := f.s.identity.Workspace().People(root.verifier(f), SampleWorkspace, []string{SampleMember})
				f.must(err)
				if len(people) != 1 || people[0].Member == nil {
					t.Fatal("member transition premise")
				}
				f.must(f.s.identity.Workspace().ChangeRole(root.verifier(f), SampleWorkspace, SampleMember, identity.Administrator, people[0].Member.Version))
				want = "blocked"
			}
			after := f.s.sessionView(captured, verifierOfActor(caller))
			if after.State != want || after.CanEnd || after.Current || after.Handle != before.Handle || after.SignedIn != before.SignedIn {
				t.Fatal("captured projection confused current lifecycle or lost identity", transition, after.State)
			}
		})
	}
}
