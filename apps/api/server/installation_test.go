package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestInstallationAdministrationSeed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if err := f.s.admin.SetInstallationWorkspace(f.s.Workspace()); err != nil {
		t.Errorf("the installation workspace is the configured one: %v", err)
	}
	if err := f.s.admin.SetInstallationWorkspace("another-workspace"); !errors.Is(err, gate.ErrConflict) {
		t.Errorf("the installation workspace is already set: %v", err)
	}
	b := f.s.identity.Boundary()
	request := gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: f.s.Workspace()}}
	for _, c := range []struct {
		login   string
		allowed bool
	}{{AdminLogin, true}, {Admin2Login, false}} {
		verifier := verifierOf(random(tokenBytes))
		if err := b.OpenSession(LoginTriple(c.login), verifier, time.Minute, time.Minute); err != nil {
			t.Fatal(err)
		}
		answer, err := b.Access(verifier, f.s.Workspace())
		if err != nil {
			t.Fatal(err)
		}
		got := f.s.gate.Decide(answer, ConsoleClient, request)
		if got.Allowed != c.allowed || (!c.allowed && got.Reason != gate.ReasonInstallationScope) {
			t.Errorf("%s: %+v", c.login, got)
		}
	}
	none := setup(t, SeedNone).ready()
	for _, w := range []world{f, none} {
		if err := w.s.admin.SetInstallationWorkspace(w.s.bootWorkspace); err != nil {
			t.Errorf("the installation workspace is the armed bootstrap workspace: %v", err)
		}
	}
	if err := none.s.admin.SetInstallationWorkspace("another-workspace"); !errors.Is(err, gate.ErrConflict) {
		t.Errorf("without a seed the installation workspace is set: %v", err)
	}
}
