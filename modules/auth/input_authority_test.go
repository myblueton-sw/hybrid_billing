package auth

import (
	"errors"
	"testing"
	"time"
)

func TestInputAuthorityRequiresDirectHumanAndClientIntersection(t *testing.T) {
	f := setup(t)
	request := ask(ConfirmInput, "")
	f.expect("no default authority", viaApp("alice"), request, false, ReasonNoGrant)
	if err := f.AddGrant(Grant{ID: "robot-input", SubjectID: "robot", Terms: terms(ConfirmInput, All()), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("service grant: %v", err)
	}
	f.grant("human", "alice", terms(ConfirmInput, All()))
	f.expect("subject authority", who("alice"), request, true, ReasonAllowed)
	f.expect("missing ceiling", viaApp("alice"), request, false, ReasonClient)
	f.clientGrant("ceiling", terms(ConfirmInput, All()))
	f.expect("intersection", viaApp("alice"), request, true, ReasonAllowed)
	f.expect("ceiling without user grant", viaApp("bob"), request, false, ReasonNoGrant)
	principal := Principal{Installation: "synthetic-installation", ClientID: "app"}
	if f.Decide(principal, request).Allowed {
		t.Fatal("client-only principal confirmed")
	}
	f.expect("service execution", viaApp("robot"), request, false, ReasonPrincipal)
	explained := f.Explain("alice", request)
	if !explained.Allowed || len(explained.Delegations) != 0 {
		t.Fatalf("direct explanation: %+v", explained)
	}
	for _, action := range []Action{Approve, ApproveSupportSession} {
		f.expect("approval semantics", who("alice"), ask(action, ""), false, ReasonApprovalRequired)
	}
	f.expect("no financial issue authority", viaApp("alice"), ask(Issue, ""), false, ReasonNoGrant)
}

func TestInputAuthorityCannotBeDelegatedOrUsedInSupport(t *testing.T) {
	f := setup(t)
	direct := terms(ConfirmInput, All())
	f.grant("human", "alice", direct)
	for _, parent := range []string{"", "parent"} {
		err := f.AddDelegation(Delegation{ID: "delegate", DelegatorID: "alice", DelegateID: "bob", ParentID: parent, Terms: direct, Validity: always, AllowRedelegation: true})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("delegation parent %q: %v", parent, err)
		}
	}
	f.grant("support-approval", "dave", terms(ApproveSupportSession, One("c1")))
	if err := f.session("s", "alice", "dave", "quantity"); err != nil {
		t.Fatal(err)
	}
	principal := who("alice")
	principal.SessionID = "s"
	f.expect("support session", principal, ask(ConfirmInput, "c1"), false, ReasonSupportScope)
	f.expect("support read unchanged", principal, ask(View, "c1", "quantity"), true, ReasonAllowed)
	f.homes("bob")
	view := terms(View, One("c1"), "quantity")
	f.grant("view", "alice", view)
	if err := f.delegate("ordinary", "alice", "bob", "", false, view, always); err != nil {
		t.Fatal(err)
	}
	f.expect("ordinary delegation unchanged", who("bob"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	f.expect("delegate has no input authority", who("bob"), ask(ConfirmInput, "c1"), false, ReasonNoGrant)
}

func TestInputAuthorityRechecksCurrentGrantsAndIdentity(t *testing.T) {
	changes := map[string]func(sourced){
		"user revoked": func(f sourced) {
			if err := f.RevokeGrant("human", 1); err != nil {
				t.Fatal(err)
			}
		},
		"client revoked": func(f sourced) {
			if err := f.RevokeGrant("ceiling", 1); err != nil {
				t.Fatal(err)
			}
		},
		"suspended":           func(f sourced) { f.source.set("alice", Suspended, 1) },
		"epoch changed":       func(f sourced) { f.source.set("alice", Active, 2) },
		"membership removed":  func(f sourced) { f.source.join("w1", "alice", 1, false) },
		"membership rejoined": func(f sourced) { f.source.join("w1", "alice", 2, true) },
		"expired":             func(f sourced) { *f.now = always.NotAfter },
		"client suspended": func(f sourced) {
			if _, err := f.PutClient(Client{ID: "app", State: Suspended}, 1); err != nil {
				t.Fatal(err)
			}
		},
		"source kind changed": func(f sourced) {
			subject := f.source.subjects["alice"]
			subject.Kind = Service
			f.source.subjects["alice"] = subject
			f.source.generation++
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f := setupSource(t)
			f.grant("human", "alice", terms(ConfirmInput, All()))
			f.clientGrant("ceiling", terms(ConfirmInput, All()))
			request := ask(ConfirmInput, "")
			first := f.Decide(viaApp("alice"), request)
			if !first.Allowed {
				t.Fatal(first.Reason)
			}
			change(f)
			if f.Decide(viaApp("alice"), request).Allowed || f.Recheck(first.Stamp, viaApp("alice"), request).Allowed {
				t.Fatal("changed authority remained usable")
			}
		})
	}
	for _, client := range []bool{false, true} {
		t.Run(map[bool]string{false: "user expiry", true: "ceiling expiry"}[client], func(t *testing.T) {
			f := setup(t)
			grant := Grant{ID: "short", SubjectID: "alice", Terms: terms(ConfirmInput, All()), Validity: Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(time.Hour)}}
			if client {
				grant.SubjectID = ""
				grant.ClientID = "app"
				f.grant("human", "alice", terms(ConfirmInput, All()))
			} else {
				f.clientGrant("ceiling", terms(ConfirmInput, All()))
			}
			if err := f.AddGrant(grant); err != nil {
				t.Fatal(err)
			}
			request := ask(ConfirmInput, "")
			first := f.Decide(viaApp("alice"), request)
			if !first.Allowed {
				t.Fatal(first.Reason)
			}
			*f.now = grant.Validity.NotAfter
			if f.Recheck(first.Stamp, viaApp("alice"), request).Allowed {
				t.Fatal("expired authority survived recheck")
			}
		})
	}
}

func TestInputAuthorityWholeWorkspaceDoesNotAcceptNarrowGrants(t *testing.T) {
	for _, dimension := range []string{"customer", "entity", "contract", "period", "workspace", "kind"} {
		t.Run(dimension, func(t *testing.T) {
			f := setup(t)
			grant := terms(ConfirmInput, All())
			switch dimension {
			case "customer":
				grant.Scope.Customer = One("c1")
			case "entity":
				grant.Scope.LegalEntity = One("le1")
			case "contract":
				grant.Scope.Contract = One("k1")
			case "period":
				grant.Period = Months(month(t, 2026, 1), month(t, 2027, 1))
			case "workspace":
				grant.Scope.WorkspaceID = "w2"
			case "kind":
				grant.Kind = "allocation_pool"
			}
			f.grant("narrow", "alice", grant)
			f.clientGrant("ceiling", terms(ConfirmInput, All()))
			f.expect("whole workspace", viaApp("alice"), ask(ConfirmInput, ""), false, ReasonNoGrant)
		})
	}
}

func TestInputAuthorityNarrowClientCeilingsRefuseWholeWorkspace(t *testing.T) {
	for _, dimension := range []string{"customer", "entity", "contract", "period", "workspace", "kind"} {
		t.Run(dimension, func(t *testing.T) {
			f := setup(t)
			grant := terms(ConfirmInput, All())
			switch dimension {
			case "customer":
				grant.Scope.Customer = One("c1")
			case "entity":
				grant.Scope.LegalEntity = One("le1")
			case "contract":
				grant.Scope.Contract = One("k1")
			case "period":
				grant.Period = Months(month(t, 2026, 1), month(t, 2027, 1))
			case "workspace":
				grant.Scope.WorkspaceID = "w2"
			case "kind":
				grant.Kind = "allocation_pool"
			}
			f.grant("human", "alice", terms(ConfirmInput, All()))
			f.clientGrant("narrow", grant)
			f.expect("whole workspace ceiling", viaApp("alice"), ask(ConfirmInput, ""), false, ReasonClient)
		})
	}
}
