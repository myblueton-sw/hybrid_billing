package gate

import "testing"

func TestApprovalAuthorityInspectionWithoutImpersonation(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.member("carol")
	id := f.grant("bob", Approve)
	f.grant("root", Approve)
	f.grant("carol", Approve)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "bob"))
	for _, tc := range []struct {
		subject, author string
		allowed         bool
	}{
		{"bob", "ann", true}, {"bob", "bob", false}, {"bob", "svc", false}, {"root", "svc", false}, {"carol", "svc", true}, {"ann", "bob", false},
	} {
		got := f.g.InspectApprovalAuthority(tc.subject, tc.author, ask(Approve))
		if got.Allowed != tc.allowed || len(got.Delegations) != 0 {
			t.Fatal(tc, got)
		}
	}
	if f.g.CheckApproval(Answer{}, "ann", ask(Approve)).Allowed {
		t.Fatal("inspection authenticated a session")
	}
	f.must(f.a.RevokeGrant(id, 1))
	if f.g.InspectApprovalAuthority("bob", "ann", ask(Approve)).Allowed {
		t.Fatal("revoked grant")
	}
	var absent *Gate
	if absent.InspectApprovalAuthority("bob", "ann", ask(Approve)).Code != AuthorityUnavailable {
		t.Fatal("missing authority")
	}
}

func TestApprovalAuthorityInspectionRefusesIdentityChanges(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.member("carol")
	f.grant("bob", Approve)
	*f.hook = func() { f.must(f.in.SuspendUser(operator, "carol", 1)) }
	*f.armed = 1
	got := f.g.InspectApprovalAuthority("bob", "ann", ask(Approve))
	if got.Allowed || got.Reason != ReasonStamp {
		t.Fatal(got)
	}
	if got = f.g.InspectApprovalAuthority("bob", "ann", ask(Approve)); !got.Allowed {
		t.Fatal("stable current authority", got)
	}
	f.must(f.in.SuspendUser(operator, "bob", 1))
	if f.g.InspectApprovalAuthority("bob", "ann", ask(Approve)).Allowed {
		t.Fatal("inactive approver")
	}
}
