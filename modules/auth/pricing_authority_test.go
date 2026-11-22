package auth

import (
	"errors"
	"testing"
)

func directGovernanceActions() []Action {
	return []Action{RecordIssuanceTerm, ApproveIssuanceTerm, BindIssuanceTerm, ApproveIssuanceTermBinding, ProposeERPResult, ConfirmERPResult, RevokeERPResult, ApplyERPResult, ProposePricingPolicy, ApprovePricingPolicy, RevokePricingPolicy, UsePricingPolicy, ProposeDocumentEvidence, ConfirmDocumentEvidence, RevokeDocumentEvidence, PrepareERPRecord}
}

func TestDirectGovernanceAuthorityDirectHumanIntersectionAndIsolation(t *testing.T) {
	for _, action := range directGovernanceActions() {
		t.Run(string(action), func(t *testing.T) {
			f := setup(t)
			req := ask(action, "")
			f.expect("no default", viaApp("alice"), req, false, ReasonNoGrant)
			if err := f.AddGrant(Grant{ID: "robot-pricing", SubjectID: "robot", Terms: terms(action, All()), Validity: always}); !errors.Is(err, ErrInvalid) {
				t.Fatal("service grant accepted", err)
			}
			f.grant("human", "alice", terms(action, All()))
			f.expect("missing console", viaApp("alice"), req, false, ReasonClient)
			f.clientGrant("ceiling", terms(action, All()))
			f.expect("intersection", viaApp("alice"), req, true, ReasonAllowed)
			f.expect("missing human", viaApp("bob"), req, false, ReasonNoGrant)
			f.expect("service execution", viaApp("robot"), req, false, ReasonPrincipal)
			if f.Decide(Principal{Installation: "synthetic-installation", ClientID: "app"}, req).Allowed {
				t.Fatal("client-only execution")
			}
			for _, other := range append(directGovernanceActions(), ConfirmInput, Issue) {
				if other != action {
					f.expect("different capability", viaApp("alice"), ask(other, ""), false, ReasonNoGrant)
				}
			}
			f.expect("financial approval unchanged", viaApp("alice"), ask(Approve, ""), false, ReasonApprovalRequired)
			for _, parent := range []string{"", "parent"} {
				if err := f.AddDelegation(Delegation{ID: "d", DelegatorID: "alice", DelegateID: "bob", ParentID: parent, Terms: terms(action, All()), Validity: always, AllowRedelegation: true}); !errors.Is(err, ErrInvalid) {
					t.Fatal("delegated governance accepted", err)
				}
			}
			f.grant("support", "dave", terms(ApproveSupportSession, One("c1")))
			if err := f.session("s", "alice", "dave", "quantity"); err != nil {
				t.Fatal(err)
			}
			principal := who("alice")
			principal.SessionID = "s"
			f.expect("support execution", principal, ask(action, "c1"), false, ReasonSupportScope)
		})
	}
}

func TestDirectGovernanceAuthorityCurrentFactsAndScope(t *testing.T) {
	for _, action := range directGovernanceActions() {
		t.Run(string(action), func(t *testing.T) {
			changes := map[string]func(sourced){
				"user revoked": func(f sourced) {
					if err := f.RevokeGrant("human", 1); err != nil {
						t.Fatal(err)
					}
				},
				"console revoked": func(f sourced) {
					if err := f.RevokeGrant("ceiling", 1); err != nil {
						t.Fatal(err)
					}
				},
				"suspended":          func(f sourced) { f.source.set("alice", Suspended, 1) },
				"membership removed": func(f sourced) { f.source.join("w1", "alice", 1, false) },
				"expired":            func(f sourced) { *f.now = always.NotAfter },
			}
			for name, change := range changes {
				t.Run(name, func(t *testing.T) {
					f := setupSource(t)
					f.grant("human", "alice", terms(action, All()))
					f.clientGrant("ceiling", terms(action, All()))
					req := ask(action, "")
					first := f.Decide(viaApp("alice"), req)
					if !first.Allowed {
						t.Fatal(first.Reason)
					}
					change(f)
					if f.Decide(viaApp("alice"), req).Allowed || f.Recheck(first.Stamp, viaApp("alice"), req).Allowed {
						t.Fatal("stale authority allowed")
					}
				})
			}
			for _, client := range []bool{false, true} {
				for _, dim := range []string{"workspace", "customer", "entity", "contract", "period", "kind"} {
					t.Run(dim+map[bool]string{true: "-console", false: "-human"}[client], func(t *testing.T) {
						f := setup(t)
						narrow := terms(action, All())
						switch dim {
						case "workspace":
							narrow.Scope.WorkspaceID = "w2"
						case "customer":
							narrow.Scope.Customer = One("c1")
						case "entity":
							narrow.Scope.LegalEntity = One("le1")
						case "contract":
							narrow.Scope.Contract = One("k1")
						case "period":
							narrow.Period = Months(month(t, 2026, 1), month(t, 2027, 1))
						case "kind":
							narrow.Kind = "allocation_pool"
						}
						reason := ReasonNoGrant
						if client {
							f.grant("human", "alice", terms(action, All()))
							f.clientGrant("ceiling", narrow)
							reason = ReasonClient
						} else {
							f.grant("human", "alice", narrow)
							f.clientGrant("ceiling", terms(action, All()))
						}
						f.expect("whole workspace", viaApp("alice"), ask(action, ""), false, reason)
					})
				}
			}
		})
	}
}
