package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func inputAuthorityAs(f world, actor actor, run func() error) error {
	verifier := actor.verifier(f)
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	answer, err := f.s.identity.Boundary().Access(verifier, SampleWorkspace)
	if err != nil {
		return err
	}
	previousCaller, previousAnswer := f.s.caller, f.s.answer
	f.s.caller, f.s.answer = verifier, answer
	defer func() { f.s.caller, f.s.answer = previousCaller, previousAnswer }()
	return run()
}

func inputAuthoritySetup(t *testing.T) (world, *serverInputAuthority, actor, actor) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a, err := newServerInputAuthority(f.s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 100})
	if err != nil {
		t.Fatal(err)
	}
	return f, a, f.as(AdminLogin), f.as(Admin2Login)
}

func inputAuthorityProposal(f world, id, kind, ceiling string) inputauthority.Proposal {
	recipient := ConsoleClient
	if kind == inputauthority.Human {
		recipient = SampleMember
	}
	return inputauthority.Proposal{ID: id, Spec: inputauthority.Spec{Capability: inputauthority.Confirm, Workspace: SampleWorkspace, Kind: kind, Recipient: recipient, CeilingID: ceiling, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(12 * time.Hour)}, Proposer: SampleAdmin, Rationale: "Synthetic authority proposal"}
}

func inputAuthorityIntent(r inputauthority.Record) inputauthority.Intent {
	return inputauthority.Intent{Workspace: SampleWorkspace, ID: r.Proposal.ID, Actor: SampleAdmin2, Digest: r.Digest, Rationale: "Independent synthetic authority review"}
}

func proposeInputAuthority(t *testing.T, f world, a *serverInputAuthority, actor actor, p inputauthority.Proposal) inputauthority.Record {
	t.Helper()
	var record inputauthority.Record
	err := inputAuthorityAs(f, actor, func() error { var err error; record, err = a.proposeLocked(p); return err })
	if err != nil {
		t.Fatal(err)
	}
	return record
}
func approveInputAuthority(t *testing.T, f world, a *serverInputAuthority, actor actor, r inputauthority.Record) inputauthority.Record {
	t.Helper()
	var record inputauthority.Record
	err := inputAuthorityAs(f, actor, func() error { var err error; record, err = a.approveLocked(inputAuthorityIntent(r)); return err })
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestInputAuthorityActualSeparateGrants(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	request := gate.Request{Action: gate.ConfirmInput, Kind: inputauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	check := func(actor actor, want bool) {
		t.Helper()
		err := inputAuthorityAs(f, actor, func() error {
			if got := f.s.gate.Decide(f.s.answer, ConsoleClient, request); got.Allowed != want {
				t.Fatalf("confirmation allowed %v want %v reason %s", got.Allowed, want, got.Reason)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	member := f.as(MemberLogin)
	check(root, false)
	check(second, false)
	check(member, false)
	console := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console-1", inputauthority.Console, ""))
	madeConsole := approveInputAuthority(t, f, a, second, console)
	check(root, false)
	check(second, false)
	check(member, false)
	human := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "human-1", inputauthority.Human, madeConsole.Decision.GrantID))
	madeHuman := approveInputAuthority(t, f, a, second, human)
	check(member, true)
	check(root, false)
	check(second, false)
	if err := f.s.admin.RevokeGrant(madeHuman.Decision.GrantID, 1); err != nil {
		t.Fatal(err)
	}
	check(member, false)
}

func TestInputAuthorityUnknownConsoleMustResolveBeforeHumanProposal(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	console := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console-1", inputauthority.Console, ""))
	f.s.fault = func(at string) bool { return at == "input-authority-after-grant" }
	err := inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(console)); return err })
	if !errors.Is(err, inputauthority.ErrUnknown) {
		t.Fatal(err)
	}
	id := "input-grant-" + console.Digest
	if _, ok := a.ceilings[id]; ok {
		t.Fatal("unresolved ceiling recorded as approved")
	}
	human := inputAuthorityProposal(f, "human-1", inputauthority.Human, id)
	err = inputAuthorityAs(f, root, func() error { _, err := a.proposeLocked(human); return err })
	if err == nil {
		t.Fatal("unresolved ceiling granted human authority")
	}
	f.s.fault = nil
	before := len(f.s.admin.Grants())
	resolved := approveInputAuthority(t, f, a, second, console)
	if resolved.Decision.GrantID != id || len(f.s.admin.Grants()) != before {
		t.Fatal("reconciliation recreated grant")
	}
	if _, ok := a.ceilings[id]; !ok {
		t.Fatal("resolved ceiling provenance missing")
	}
	proposeInputAuthority(t, f, a, root, human)
}

func TestInputAuthorityProposerIsRecheckedUnderAnotherSession(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	console := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console-1", inputauthority.Console, ""))
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleAdmin && grant.Terms.Action == gate.AdministerInstallation {
			if err := f.s.admin.RevokeGrant(grant.ID, grant.Version); err != nil {
				t.Fatal(err)
			}
		}
	}
	err := inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(console)); return err })
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatalf("stored proposer authority not checked: %v", err)
	}
	if _, found := f.s.grantView("input-grant-" + console.Digest); found {
		t.Fatal("revoked proposer produced grant")
	}
}

func TestInputAuthorityWrappersRejectForgedActorsAndWorkspaces(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	p := inputAuthorityProposal(f, "console-1", inputauthority.Console, "")
	forged := p
	forged.Proposer = SampleAdmin2
	if err := inputAuthorityAs(f, root, func() error { _, err := a.proposeLocked(forged); return err }); !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal(err)
	}
	foreign := p
	foreign.Spec.Workspace = "other"
	if err := inputAuthorityAs(f, root, func() error { _, err := a.proposeLocked(foreign); return err }); !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal(err)
	}
	record := proposeInputAuthority(t, f, a, root, p)
	if err := inputAuthorityAs(f, root, func() error { _, err := a.approveLocked(inputAuthorityIntent(record)); return err }); !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal(err)
	}
	if err := f.s.identity.Boundary().EndSession(second.verifier(f)); err != nil {
		t.Fatal(err)
	}
	if err := inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(record)); return err }); err == nil {
		t.Fatal("ended session approved")
	}
}

func TestInputAuthorityCeilingRevocationAndGrantCollision(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	console := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console-1", inputauthority.Console, ""))
	made := approveInputAuthority(t, f, a, second, console)
	if err := f.s.admin.RevokeGrant(made.Decision.GrantID, 1); err != nil {
		t.Fatal(err)
	}
	err := inputAuthorityAs(f, root, func() error {
		_, err := a.proposeLocked(inputAuthorityProposal(f, "human-1", inputauthority.Human, made.Decision.GrantID))
		return err
	})
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal("revoked ceiling accepted", err)
	}
	p := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "console-2", inputauthority.Console, ""))
	id := "input-grant-" + p.Digest
	if err := f.s.admin.AddGrant(inputAuthorityGrant(id, p.Proposal.Spec)); err != nil {
		t.Fatal(err)
	}
	err = inputAuthorityAs(f, second, func() error { _, err := a.approveLocked(inputAuthorityIntent(p)); return err })
	if err == nil {
		t.Fatal("preexisting grant adopted")
	}
	if _, ok := a.ceilings[id]; ok {
		t.Fatal("collision received provenance")
	}
}

func TestInputAuthorityLookupComparesCompleteGrant(t *testing.T) {
	f, a, root, _ := inputAuthoritySetup(t)
	p := inputAuthorityProposal(f, "console-1", inputauthority.Console, "")
	changes := map[string]func(*gate.Grant){
		"action": func(g *gate.Grant) { g.Terms.Action = gate.Issue }, "kind": func(g *gate.Grant) { g.Terms.Kind = candidateKind },
		"workspace": func(g *gate.Grant) { g.Terms.Scope.WorkspaceID = "other" }, "customer": func(g *gate.Grant) { g.Terms.Scope.Customer = gate.One("c1") },
		"entity": func(g *gate.Grant) { g.Terms.Scope.LegalEntity = gate.One("le1") }, "contract": func(g *gate.Grant) { g.Terms.Scope.Contract = gate.One("k1") },
		"validity": func(g *gate.Grant) { g.Validity.NotAfter = g.Validity.NotAfter.Add(time.Hour) },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			id := "wrong-" + name
			a.bindings[id] = p.Spec
			grant := inputAuthorityGrant(id, p.Spec)
			change(&grant)
			if err := f.s.admin.AddGrant(grant); err != nil {
				t.Fatal(err)
			}
			err := inputAuthorityAs(f, root, func() error { _, err := a.Lookup(id); return err })
			if !errors.Is(err, inputauthority.ErrConflict) {
				t.Fatalf("mismatch %s: %v", name, err)
			}
		})
	}
}

func inputAuthorityGrant(id string, spec inputauthority.Spec) gate.Grant {
	grant, _ := (&serverGrantAuthority{domain: inputGrantAuthority}).grant(id, spec)
	return grant
}
func sameInputAuthorityGrant(g gate.GrantView, id string, spec inputauthority.Spec) bool {
	return (&serverGrantAuthority{domain: inputGrantAuthority}).sameGrant(g, id, spec)
}
