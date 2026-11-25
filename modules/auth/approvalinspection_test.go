package auth

import "testing"

func TestApprovalAuthorityInspectionUsesCurrentDirectScope(t *testing.T) {
	f := setupSource(t)
	f.grant("approval", "alice", terms(Approve, One("c1")))
	for _, tc := range []struct {
		name, subject, author string
		request               Request
		allowed               bool
	}{
		{"current", "alice", "bob", ask(Approve, "c1"), true},
		{"foreign customer", "alice", "bob", ask(Approve, "c2"), false},
		{"wrong action", "alice", "bob", ask(Issue, "c1"), false},
		{"self", "alice", "alice", ask(Approve, "c1"), false},
		{"unknown author", "alice", "missing", ask(Approve, "c1"), false},
		{"unknown actor", "missing", "bob", ask(Approve, "c1"), false},
		{"service actor", "robot", "bob", ask(Approve, "c1"), false},
		{"no grant", "carol", "bob", ask(Approve, "c1"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := f.InspectApprovalAuthority(tc.subject, tc.author, tc.request); got.Allowed != tc.allowed || len(got.Delegations) != 0 {
				t.Fatal(got)
			}
		})
	}
	if f.Explain("alice", ask(Approve, "c1")).Allowed {
		t.Fatal("ordinary explanation gained approval capability")
	}
	f.source.join("w1", "alice", 1, false)
	if f.InspectApprovalAuthority("alice", "bob", ask(Approve, "c1")).Allowed {
		t.Fatal("inactive membership")
	}
	f.source.join("w1", "alice", 1, true)
	f.source.set("alice", Suspended, 2)
	if f.InspectApprovalAuthority("alice", "bob", ask(Approve, "c1")).Allowed {
		t.Fatal("inactive subject")
	}
}

func TestApprovalAuthorityInspectionRevocationExpiryAndConsistency(t *testing.T) {
	for _, mode := range []string{"revoke", "expire", "generation", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			f := setupSource(t)
			f.grant("approval", "alice", terms(Approve, One("c1")))
			switch mode {
			case "revoke":
				if err := f.RevokeGrant("approval", 1); err != nil {
					t.Fatal(err)
				}
			case "expire":
				*f.now = always.NotAfter.Add(1)
			case "generation":
				f.source.changeAt = f.source.reads + 2
			case "unavailable":
				f.SetAvailable(false)
			}
			got := f.InspectApprovalAuthority("alice", "bob", ask(Approve, "c1"))
			if got.Allowed || mode == "unavailable" && got.Code != AuthorityUnavailable {
				t.Fatal(got)
			}
		})
	}
}
