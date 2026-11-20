package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func TestInputAuthorityCompleteGrantEquality(t *testing.T) {
	spec := inputauthority.Spec{Capability: inputauthority.Confirm, Workspace: "w1", Kind: inputauthority.Human, Recipient: "member", CeilingID: "ceiling", NotBefore: start, NotAfter: start.Add(time.Hour)}
	base := gate.GrantView{Grant: inputAuthorityGrant("grant", spec), Version: 1}
	if !sameInputAuthorityGrant(base, "grant", spec) {
		t.Fatal("exact grant differs")
	}
	changes := map[string]func(*gate.GrantView){
		"id": func(g *gate.GrantView) { g.ID = "other" }, "subject": func(g *gate.GrantView) { g.SubjectID = "other" }, "client": func(g *gate.GrantView) { g.ClientID = ConsoleClient },
		"action": func(g *gate.GrantView) { g.Terms.Action = gate.Issue }, "kind": func(g *gate.GrantView) { g.Terms.Kind = candidateKind }, "workspace": func(g *gate.GrantView) { g.Terms.Scope.WorkspaceID = "other" },
		"customer": func(g *gate.GrantView) { g.Terms.Scope.Customer = gate.One("c1") }, "entity": func(g *gate.GrantView) { g.Terms.Scope.LegalEntity = gate.One("e1") }, "contract": func(g *gate.GrantView) { g.Terms.Scope.Contract = gate.One("k1") },
		"period": func(g *gate.GrantView) {
			first, _ := gate.NewMonth(2026, 1)
			last, _ := gate.NewMonth(2027, 1)
			g.Terms.Period = gate.Months(first, last)
		}, "fields": func(g *gate.GrantView) { g.Terms.Fields = []string{"evidence"} },
		"before": func(g *gate.GrantView) { g.Validity.NotBefore = g.Validity.NotBefore.Add(time.Second) }, "after": func(g *gate.GrantView) { g.Validity.NotAfter = g.Validity.NotAfter.Add(time.Second) },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			g := base
			change(&g)
			if sameInputAuthorityGrant(g, "grant", spec) {
				t.Fatal("mismatched grant accepted")
			}
		})
	}
}

func TestInputAuthorityUnknownRevokedEffectIsNotRecreated(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	record := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console", inputauthority.Console, ""))
	f.s.fault = func(at string) bool { return at == "input-authority-after-grant" }
	if err := inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(record)); return err }); !errors.Is(err, inputauthority.ErrUnknown) {
		t.Fatal(err)
	}
	id := "input-grant-" + record.Digest
	f.s.fault = nil
	if err := f.s.admin.RevokeGrant(id, 1); err != nil {
		t.Fatal(err)
	}
	before := len(f.s.admin.Grants())
	resolved := approveInputAuthority(t, f, a, second, record)
	if resolved.Decision.GrantStatus != inputauthority.RevokedGrant || len(f.s.admin.Grants()) != before {
		t.Fatal("historical grant revived")
	}
	err := inputAuthorityAs(f, root, func() error {
		_, err := a.proposeLocked(inputAuthorityProposal(f, "human", inputauthority.Human, id))
		return err
	})
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal("historical revoked provenance used as current ceiling", err)
	}
}

func TestInputAuthoritySuspendedConsoleAndUnrelatedChange(t *testing.T) {
	for _, mode := range []string{"suspended", "unrelated"} {
		t.Run(mode, func(t *testing.T) {
			f, a, root, second := inputAuthoritySetup(t)
			record := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console", inputauthority.Console, ""))
			if mode == "suspended" {
				if _, err := f.s.admin.PutClient(gate.Client{ID: ConsoleClient, State: gate.Suspended}, 1); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := f.s.admin.PutClient(gate.Client{ID: "unrelated", State: gate.Active}, 0); err != nil {
					t.Fatal(err)
				}
			}
			err := inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(record)); return err })
			if err == nil {
				t.Fatal("changed authority accepted")
			}
			if _, found := f.s.grantView("input-grant-" + record.Digest); found {
				t.Fatal("grant applied after change")
			}
		})
	}
}

func TestInputAuthorityTargetRequiresCurrentContainedCeiling(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	record := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console", inputauthority.Console, ""))
	made := approveInputAuthority(t, f, a, second, record)
	for _, mode := range []string{"earlier", "later", "unknown", "client"} {
		t.Run(mode, func(t *testing.T) {
			p := inputAuthorityProposal(f, "human-"+mode, inputauthority.Human, made.Decision.GrantID)
			switch mode {
			case "earlier":
				p.Spec.NotBefore = p.Spec.NotBefore.Add(-time.Second)
			case "later":
				p.Spec.NotAfter = p.Spec.NotAfter.Add(time.Second)
			case "unknown":
				p.Spec.CeilingID = "not-approved"
			case "client":
				p.Spec.Kind = inputauthority.Console
				p.Spec.Recipient = "other-console"
				p.Spec.CeilingID = ""
			}
			if err := inputAuthorityAs(f, root, func() error { _, err := a.proposeLocked(p); return err }); err == nil {
				t.Fatal("invalid ceiling accepted")
			}
		})
	}
}

func TestInputAuthorityExpiredUnknownEffectResolvesHistorically(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	record := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console", inputauthority.Console, ""))
	f.s.fault = func(at string) bool { return at == "input-authority-after-grant" }
	if err := inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(record)); return err }); !errors.Is(err, inputauthority.ErrUnknown) {
		t.Fatal(err)
	}
	f.s.fault = nil
	f.advance(13 * time.Hour)
	second = f.as(Admin2Login)
	before := len(f.s.admin.Grants())
	resolved := approveInputAuthority(t, f, a, second, record)
	if resolved.Decision.GrantStatus != inputauthority.ExpiredGrant || len(f.s.admin.Grants()) != before {
		t.Fatal("expired grant recreated")
	}
}

func TestInputAuthorityPortsRecheckEndedCallerAndUnavailableAuthority(t *testing.T) {
	for _, mode := range []string{"ended", "unavailable", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			f, a, root, _ := inputAuthoritySetup(t)
			record := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console", inputauthority.Console, ""))
			err := inputAuthorityAs(f, root, func() error {
				switch mode {
				case "ended":
					if err := f.s.identity.Boundary().EndSession(f.s.caller); err != nil {
						return err
					}
				case "unavailable":
					if err := f.s.admin.SetAvailable(false); err != nil {
						return err
					}
				case "lookup":
					f.s.fault = func(at string) bool { return at == "input-authority-lookup" }
				}
				_, err := a.Lookup("input-grant-" + record.Digest)
				return err
			})
			if err == nil {
				t.Fatal("unavailable lookup reported authoritative absence")
			}
		})
	}
}
