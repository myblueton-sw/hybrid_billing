package plugins

import (
	"testing"
	"time"
)

func TestInstallLifecycle(t *testing.T) {
	f := standard(t)
	expect(t, "form", f.try(at(ChangeInstall, "w1", "ledger", "1")), ErrInvalid)
	expect(t, "plugin form", f.try(at(ChangeInstall, "w1", "", "1.0.0")), ErrInvalid)
	expect(t, "missing version", f.try(at(ChangeInstall, "w1", "ledger", "3.0.0")), ErrNotFound)
	expect(t, "missing plugin", f.try(at(ChangeInstall, "w1", "other", "1.0.0")), ErrNotFound)
	expect(t, "enable missing", f.try(at(ChangeEnable, "w1", "ledger", "")), ErrNotFound)
	expect(t, "enable form", f.try(at(ChangeEnable, "w1", "", "")), ErrInvalid)
	f.install("w1", "ledger", "1.0.0")
	v := f.state("w1", "ledger")
	if v.State != StateInstalled || v.Version != "1.0.0" || v.Revision != 1 || len(v.Endpoints) != 0 || v.Disposal != (Storage{}) {
		t.Fatalf("installed %+v", v)
	}
	reason(t, "install twice", f.try(at(ChangeInstall, "w1", "ledger", "1.1.0")), ReasonNotAllowedState)
	expect(t, "other workspace", second(f.r.Installation("w2", "ledger")), ErrNotFound)
	reason(t, "stop installed", second(f.r.Stop("w1", "ledger", ReasonOperator, ann)), ReasonNotAllowedState)
	if d, err := f.r.Disable("w1", "ledger", ann); err != nil || d.Revision != 1 {
		t.Errorf("disable installed %+v %v", d, err)
	}
	f.enable("w1", "ledger")
	if v := f.state("w1", "ledger"); v.State != StateEnabled || v.Revision != 2 {
		t.Fatalf("enabled %+v", v)
	}
	reason(t, "enable twice", f.try(at(ChangeEnable, "w1", "ledger", "")), ReasonNotAllowedState)
	stopped, err := f.r.Stop("w1", "ledger", ReasonTimeout, run1)
	f.must(err)
	if stopped.State != StateStopped || stopped.Reason != ReasonTimeout || stopped.Revision != 3 {
		t.Fatalf("stopped %+v", stopped)
	}
	if again, err := f.r.Stop("w1", "ledger", ReasonTimeout, ann); err != nil || again.Revision != 3 {
		t.Errorf("stop twice %+v %v", again, err)
	}
	f.enable("w1", "ledger")
	disabled, err := f.r.Disable("w1", "ledger", ann)
	f.must(err)
	if disabled.State != StateInstalled || disabled.Revision != 5 || disabled.Reason != "" {
		t.Fatalf("disabled %+v", disabled)
	}
	f.enable("w1", "ledger")
	f.must(second(f.r.Stop("w1", "ledger", ReasonOperator, ann)))
	if d, err := f.r.Disable("w1", "ledger", ann); err != nil || d.State != StateInstalled {
		t.Errorf("disable stopped %+v %v", d, err)
	}
	f.approve(at(ChangeRemove, "w1", "ledger", ""))
	removed := f.state("w1", "ledger")
	if removed.State != StateRemoved || removed.Disposal.Deletion != RetainForEvidence || removed.Disposal.Namespace != "ledger-data" {
		t.Fatalf("removed %+v", removed)
	}
	reason(t, "remove twice", f.try(at(ChangeRemove, "w1", "ledger", "")), ReasonNotAllowedState)
	reason(t, "enable removed", f.try(at(ChangeEnable, "w1", "ledger", "")), ReasonNotAllowedState)
	reason(t, "disable removed", second(f.r.Disable("w1", "ledger", ann)), ReasonNotAllowedState)
	reason(t, "stop removed", second(f.r.Stop("w1", "ledger", ReasonOperator, ann)), ReasonNotAllowedState)
	reason(t, "upgrade removed", f.try(at(ChangeUpgrade, "w1", "ledger", "1.1.0")), ReasonNotAllowedState)
	expect(t, "remove missing", f.try(at(ChangeRemove, "w2", "ledger", "")), ErrNotFound)
	f.install("w1", "ledger", "1.1.0")
	if v := f.state("w1", "ledger"); v.State != StateInstalled || v.Revision <= removed.Revision || v.Disposal != (Storage{}) {
		t.Errorf("reinstalled %+v", v)
	}
	if _, err := f.r.Version("ledger", "1.0.0"); err != nil {
		t.Error("versions must stay readable after removal")
	}
}

func TestLifecycleActors(t *testing.T) {
	f := running(t)
	reason(t, "disable assistant", second(f.r.Disable("w1", "ledger", bot)), ReasonAINotAllowed)
	reason(t, "disable system", second(f.r.Disable("w1", "ledger", run1)), ReasonNotHuman)
	expect(t, "disable form", second(f.r.Disable("", "ledger", ann)), ErrInvalid)
	expect(t, "disable plugin form", second(f.r.Disable("w1", "", ann)), ErrInvalid)
	expect(t, "disable missing", second(f.r.Disable("w9", "ledger", ann)), ErrNotFound)
	expect(t, "disable missing plugin", second(f.r.Disable("w1", "other", ann)), ErrNotFound)
	reason(t, "stop assistant", second(f.r.Stop("w1", "ledger", ReasonOperator, Actor{ID: "run-1", Kind: KindAssistant})), ReasonAINotAllowed)
	reason(t, "stop foreign runtime", second(f.r.Stop("w2", "ledger", ReasonTimeout, run1)), ReasonNotRuntime)
	expect(t, "stop reason", second(f.r.Stop("w1", "ledger", "", ann)), ErrInvalid)
	expect(t, "stop actor", second(f.r.Stop("w1", "ledger", ReasonOperator, Actor{ID: "x"})), ErrInvalid)
	expect(t, "stop missing", second(f.r.Stop("w1", "other", ReasonOperator, ann)), ErrNotFound)
	if f.state("w2", "ledger").State != StateEnabled {
		t.Error("foreign stop changed another workspace")
	}
	f.must(second(f.r.Stop("w1", "ledger", ReasonTimeout, run1)))
	if f.state("w2", "ledger").State != StateEnabled || f.state("v1", "ledger").State != StateEnabled {
		t.Error("stop changed another workspace")
	}
	expect(t, "view form", second(f.r.Installation("w1", "")), ErrInvalid)
}

func TestVersionSwitches(t *testing.T) {
	f := standard(t)
	f.version(manifest("ledger", "1.2.0"))
	f.install("w1", "ledger", "1.1.0")
	f.install("w2", "ledger", "1.1.0")
	f.enable("w1", "ledger")
	reason(t, "upgrade down", f.try(at(ChangeUpgrade, "w1", "ledger", "1.0.0")), ReasonDirection)
	reason(t, "upgrade same", f.try(at(ChangeUpgrade, "w1", "ledger", "1.1.0")), ReasonDirection)
	reason(t, "rollback up", f.try(at(ChangeRollback, "w1", "ledger", "1.2.0")), ReasonDirection)
	reason(t, "rollback same", f.try(at(ChangeRollback, "w1", "ledger", "1.1.0")), ReasonDirection)
	expect(t, "switch missing version", f.try(at(ChangeUpgrade, "w1", "ledger", "5.0.0")), ErrNotFound)
	expect(t, "switch missing installation", f.try(at(ChangeUpgrade, "v1", "ledger", "1.2.0")), ErrNotFound)
	before := f.state("w1", "ledger")
	f.approve(at(ChangeUpgrade, "w1", "ledger", "1.2.0"))
	after := f.state("w1", "ledger")
	if after.Version != "1.2.0" || after.State != StateEnabled || after.Revision != before.Revision+1 {
		t.Fatalf("upgraded %+v", after)
	}
	if f.state("w2", "ledger").Version != "1.1.0" {
		t.Error("upgrade changed another workspace")
	}
	f.approve(at(ChangeRollback, "w1", "ledger", "1.0.0"))
	if v := f.state("w1", "ledger"); v.Version != "1.0.0" || v.State != StateEnabled {
		t.Fatalf("rolled back %+v", v)
	}
	f.must(second(f.r.Stop("w1", "ledger", ReasonOperator, ann)))
	f.approve(at(ChangeUpgrade, "w1", "ledger", "1.1.0"))
	if v := f.state("w1", "ledger"); v.State != StateStopped {
		t.Errorf("switch of a stopped installation %+v", v)
	}
	f.approve(at(ChangeUpgrade, "w2", "ledger", "1.2.0"))
	if v := f.state("w2", "ledger"); v.State != StateInstalled {
		t.Errorf("switch of an installed installation %+v", v)
	}
}

func TestStorageRules(t *testing.T) {
	f := standard(t)
	moved := manifest("ledger", "2.0.0")
	moved.Storage.Namespace, moved.Storage.RetentionDays = "elsewhere", 31
	moved.Screens[0].Fields[0].Source = "elsewhere"
	f.version(moved)
	weaker := manifest("ledger", "2.1.0")
	weaker.Storage.Deletion, weaker.Storage.RetentionDays = DeleteOnRemoval, 0
	f.version(weaker)
	shorter := manifest("ledger", "2.2.0")
	shorter.Storage.RetentionDays = 29
	f.version(shorter)
	longer := manifest("ledger", "2.3.0")
	longer.Storage.RetentionDays = 31
	longer.Storage.Migration = MigrationCopyForward
	f.version(longer)
	f.install("w1", "ledger", "1.1.0")
	reason(t, "namespace", f.try(at(ChangeUpgrade, "w1", "ledger", "2.0.0")), ReasonStorageChanged)
	reason(t, "deletion", f.try(at(ChangeUpgrade, "w1", "ledger", "2.1.0")), ReasonStorageChanged)
	reason(t, "retention", f.try(at(ChangeUpgrade, "w1", "ledger", "2.2.0")), ReasonStorageChanged)
	f.approve(at(ChangeUpgrade, "w1", "ledger", "2.3.0"))
	reason(t, "shorter rollback", f.try(at(ChangeRollback, "w1", "ledger", "1.1.0")), ReasonStorageChanged)
	other := manifest("audit", "1.0.0")
	other.Storage.Namespace = "ledger-data"
	other.Screens[0].Fields[0].Source = "ledger-data"
	f.version(other)
	reason(t, "live namespace", f.try(at(ChangeInstall, "w1", "audit", "1.0.0")), ReasonNamespaceTaken)
	f.install("w2", "audit", "1.0.0")
	f.approve(at(ChangeRemove, "w1", "ledger", ""))
	reason(t, "retained namespace", f.try(at(ChangeInstall, "w1", "audit", "1.0.0")), ReasonNamespaceTaken)
	reason(t, "weaker reinstall", f.try(at(ChangeInstall, "w1", "ledger", "2.1.0")), ReasonStorageChanged)
	reason(t, "shorter reinstall", f.try(at(ChangeInstall, "w1", "ledger", "1.1.0")), ReasonStorageChanged)
	reason(t, "moved reinstall", f.try(at(ChangeInstall, "w1", "ledger", "2.0.0")), ReasonStorageChanged)
	f.advance(31 * 24 * time.Hour)
	reason(t, "retention boundary", f.try(at(ChangeInstall, "w1", "audit", "1.0.0")), ReasonNamespaceTaken)
	f.advance(time.Nanosecond)
	f.install("w1", "audit", "1.0.0")
	f.install("w1", "ledger", "2.0.0")
}

func TestDeletedStorageIsReleased(t *testing.T) {
	f := standard(t)
	gone := manifest("scratch", "1.0.0")
	gone.Storage = Storage{Namespace: "shared", Owner: "acme", Deletion: DeleteOnRemoval, Migration: MigrationNone}
	gone.Screens[0].Fields[0].Source = "shared"
	f.version(gone)
	next := manifest("audit", "1.0.0")
	next.Storage.Namespace = "shared"
	next.Screens[0].Fields[0].Source = "shared"
	f.version(next)
	f.install("w1", "scratch", "1.0.0")
	f.approve(at(ChangeRemove, "w1", "scratch", ""))
	if v := f.state("w1", "scratch"); v.Disposal.Deletion != DeleteOnRemoval {
		t.Errorf("disposal %+v", v)
	}
	f.install("w1", "audit", "1.0.0")
}

func TestInstallationRoom(t *testing.T) {
	l := limits
	l.Installations = 1
	f := setupWith(t, l)
	f.publisher("acme")
	f.version(manifest("ledger", "1.0.0"))
	f.version(manifest("audit", "1.0.0"))
	f.workspace("w1", "run-1", false)
	f.install("w1", "ledger", "1.0.0")
	expect(t, "room", f.try(at(ChangeInstall, "w1", "audit", "1.0.0")), ErrLimit)
	f.approve(at(ChangeRemove, "w1", "ledger", ""))
	f.install("w1", "audit", "1.0.0")
}

func TestMarks(t *testing.T) {
	f := running(t)
	f.version(manifest("ledger", "1.2.0"))
	expect(t, "form", second(f.r.MarkVulnerable("ledger", "1.1.0", "", ann)), ErrInvalid)
	reason(t, "assistant", second(f.r.MarkVulnerable("ledger", "1.1.0", "CVE", bot)), ReasonAINotAllowed)
	expect(t, "missing", second(f.r.MarkVulnerable("ledger", "4.0.0", "CVE", ann)), ErrNotFound)
	expect(t, "target", second(f.r.ConfirmMark(Target{Kind: "other"}, bob)), ErrInvalid)
	m, err := f.r.MarkVulnerable("ledger", "1.1.0", "Remote code execution", ann)
	f.must(err)
	if m.By != "ann" || m.Reason != "Remote code execution" || m.ConfirmedBy != "" {
		t.Fatalf("mark %+v", m)
	}
	for _, w := range []string{"w1", "w2"} {
		if v := f.state(w, "ledger"); v.State != StateStopped || v.Reason != ReasonVulnerable {
			t.Errorf("%s %+v", w, v)
		}
	}
	if v := f.state("v1", "ledger"); v.State != StateEnabled {
		t.Errorf("verification workspace stopped %+v", v)
	}
	again, _ := f.r.MarkVulnerable("ledger", "1.1.0", "Another reason", bob)
	if again.By != "ann" || again.Reason != "Remote code execution" {
		t.Errorf("repeat mark %+v", again)
	}
	reason(t, "enable vulnerable", f.try(at(ChangeEnable, "w1", "ledger", "")), ReasonVulnerable)
	f.approve(at(ChangeUpgrade, "w2", "ledger", "1.2.0"))
	reason(t, "rollback to vulnerable", f.try(at(ChangeRollback, "w2", "ledger", "1.1.0")), ReasonVulnerable)
	view, _ := f.r.Version("ledger", "1.1.0")
	if view.Vulnerable == nil || view.Vulnerable.By != "ann" {
		t.Errorf("version mark %+v", view)
	}
	view.Vulnerable.By = "x"
	if v, _ := f.r.Version("ledger", "1.1.0"); v.Vulnerable.By != "ann" {
		t.Error("mark view shares storage")
	}
	reason(t, "self confirm", second(f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, ann)), ReasonSelfApproval)
	reason(t, "clear by other", second(f.r.ClearMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, bob)), ReasonNotProposer)
	confirmed, err := f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, bob)
	f.must(err)
	if confirmed.ConfirmedBy != "bob" {
		t.Errorf("confirmed %+v", confirmed)
	}
	if again, err := f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, bob); err != nil || again.ConfirmedBy != "bob" {
		t.Errorf("repeat confirm %+v %v", again, err)
	}
	reason(t, "confirm by third", second(f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, cat)), ReasonNotPending)
	reason(t, "clear confirmed", second(f.r.ClearMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, ann)), ReasonNotPending)
	expect(t, "confirm unmarked", second(f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.0.0"}, bob)), ErrNotFound)
	reason(t, "confirm assistant", second(f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}, bot)), ReasonAINotAllowed)
	expect(t, "confirm missing", second(f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "9.0.0"}, bob)), ErrNotFound)
}

func TestClearMarkRestartsNothing(t *testing.T) {
	f := running(t)
	target := Target{Kind: MarkVulnerable, Plugin: "ledger", Version: "1.1.0"}
	f.must(second(f.r.MarkVulnerable("ledger", "1.1.0", "Suspected", ann)))
	cleared, err := f.r.ClearMark(target, ann)
	f.must(err)
	if cleared.By != "ann" {
		t.Errorf("cleared %+v", cleared)
	}
	if v := f.state("w1", "ledger"); v.State != StateStopped {
		t.Errorf("clear restarted %+v", v)
	}
	expect(t, "clear twice", second(f.r.ClearMark(target, ann)), ErrNotFound)
	f.enable("w1", "ledger")
	reason(t, "clear assistant", second(f.r.ClearMark(target, bot)), ReasonAINotAllowed)
}

func TestCompromiseAndRevocation(t *testing.T) {
	f := running(t)
	f.publisher("beta")
	other := manifest("audit", "1.0.0")
	other.Publisher = "beta"
	f.version(other)
	f.install("w1", "audit", "1.0.0")
	f.enable("w1", "audit")
	expect(t, "compromise form", second(f.r.CompromiseKey("acme", "", "Leaked", ann)), ErrInvalid)
	expect(t, "compromise missing", second(f.r.CompromiseKey("acme", "k9", "Leaked", ann)), ErrNotFound)
	expect(t, "revoke form", second(f.r.RevokePublisher("", "Gone", ann)), ErrInvalid)
	expect(t, "revoke missing", second(f.r.RevokePublisher("zeta", "Gone", ann)), ErrNotFound)
	f.must(second(f.r.CompromiseKey("acme", "k1", "Leaked signing key", ann)))
	for _, w := range []string{"w1", "w2", "v1"} {
		if v := f.state(w, "ledger"); v.State != StateStopped || v.Reason != ReasonRevoked {
			t.Errorf("%s %+v", w, v)
		}
	}
	if v := f.state("w1", "audit"); v.State != StateEnabled {
		t.Errorf("other publisher stopped %+v", v)
	}
	p, _ := f.r.Publisher("acme")
	if !p.Keys[0].Retired || p.Keys[0].Compromise == nil {
		t.Errorf("key %+v", p.Keys[0])
	}
	reason(t, "enable revoked", f.try(at(ChangeEnable, "v1", "ledger", "")), ReasonRevoked)
	f.must(second(f.r.ClearMark(Target{Kind: MarkCompromise, Publisher: "acme", Key: "k1"}, ann)))
	f.enable("v1", "ledger")
	f.must(second(f.r.RevokePublisher("beta", "Contract ended", ann)))
	if v := f.state("w1", "audit"); v.State != StateStopped || v.Reason != ReasonRevoked {
		t.Errorf("revoked publisher %+v", v)
	}
	if v := f.state("v1", "ledger"); v.State != StateEnabled {
		t.Errorf("revocation stopped another publisher %+v", v)
	}
	f.must(second(f.r.ConfirmMark(Target{Kind: MarkRevocation, Publisher: "beta"}, bob)))
	pv, _ := f.r.Publisher("beta")
	if pv.Revocation == nil || pv.Revocation.ConfirmedBy != "bob" {
		t.Errorf("revocation %+v", pv)
	}
	reason(t, "install revoked", f.try(at(ChangeInstall, "w2", "audit", "1.0.0")), ReasonRevoked)
}

func TestVerificationWorkspaces(t *testing.T) {
	f := standard(t)
	f.must(second(f.r.MarkVulnerable("ledger", "1.0.0", "Known flaw", ann)))
	reason(t, "production install", f.try(at(ChangeInstall, "w1", "ledger", "1.0.0")), ReasonVulnerable)
	f.install("v1", "ledger", "1.0.0")
	f.enable("v1", "ledger")
	f.install("w1", "ledger", "1.1.0")
	reason(t, "production rollback", f.try(at(ChangeRollback, "w1", "ledger", "1.0.0")), ReasonVulnerable)
	f.approve(at(ChangeUpgrade, "v1", "ledger", "1.1.0"))
	f.approve(at(ChangeRollback, "v1", "ledger", "1.0.0"))
	if v := f.state("v1", "ledger"); v.State != StateEnabled || v.Version != "1.0.0" {
		t.Errorf("verification %+v", v)
	}
	text := f.token("v1", runV, events)
	if g, err := f.r.CheckToken(text, events); err != nil || g.Version != "1.0.0" {
		t.Errorf("verification token %+v %v", g, err)
	}
	reason(t, "verification command", third(f.r.IssueToken("v1", "ledger", []Capability{call}, minTokenLife, runV)), ReasonIsolated)
	reason(t, "verification endpoint", f.try(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "v1"}, Plugin: "ledger",
		Endpoint: Endpoint{ID: "api", Fingerprint: fingerprint("a"), Evidence: fingerprint("e")}}), ReasonIsolated)
	reason(t, "verification egress", f.r.CheckEgress("v1", "ledger", "crm", runV), ReasonIsolated)
}

func TestEndpoints(t *testing.T) {
	f := running(t)
	endpoint := func(workspace, id, print string) Change {
		return Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: workspace}, Plugin: "ledger", Endpoint: Endpoint{ID: id, Fingerprint: fingerprint(print), Evidence: fingerprint("evidence")}}
	}
	expect(t, "form", f.try(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Plugin: "ledger", Endpoint: Endpoint{ID: "api", Fingerprint: "x", Evidence: fingerprint("e")}}), ErrInvalid)
	expect(t, "evidence form", f.try(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Plugin: "ledger", Endpoint: Endpoint{ID: "api", Fingerprint: fingerprint("e")}}), ErrInvalid)
	expect(t, "id form", f.try(endpoint("w1", "", "a")), ErrInvalid)
	expect(t, "plugin form", f.try(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Endpoint: Endpoint{ID: "api", Fingerprint: fingerprint("a"), Evidence: fingerprint("e")}}), ErrInvalid)
	expect(t, "missing installation", f.try(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Plugin: "other", Endpoint: Endpoint{ID: "api", Fingerprint: fingerprint("a"), Evidence: fingerprint("e")}}), ErrNotFound)
	reason(t, "undeclared", f.try(endpoint("w1", "web", "a")), ReasonNotDeclared)
	f.approve(endpoint("w1", "api", "a"))
	v := f.state("w1", "ledger")
	if len(v.Endpoints) != 1 || v.Endpoints[0].Approver != "bob" || v.Endpoints[0].Endpoint.Fingerprint != fingerprint("a") {
		t.Fatalf("endpoint %+v", v)
	}
	expect(t, "shared fingerprint", f.try(endpoint("w2", "api", "a")), ErrConflict)
	f.approve(endpoint("w1", "api", "b"))
	expect(t, "tombstone", f.try(endpoint("w2", "api", "a")), ErrConflict)
	f.approve(endpoint("w2", "api", "c"))
	got, err := f.r.CheckEndpoint("w1", "ledger", "api", fingerprint("b"), run1)
	f.must(err)
	if got.Endpoint.Evidence != fingerprint("evidence") {
		t.Errorf("checked %+v", got)
	}
	reason(t, "old fingerprint", second(f.r.CheckEndpoint("w1", "ledger", "api", fingerprint("a"), run1)), ReasonFingerprintMismatch)
	reason(t, "foreign runtime", second(f.r.CheckEndpoint("w1", "ledger", "api", fingerprint("b"), run2)), ReasonNotRuntime)
	reason(t, "human caller", second(f.r.CheckEndpoint("w1", "ledger", "api", fingerprint("b"), ann)), ReasonNotRuntime)
	expect(t, "check form", second(f.r.CheckEndpoint("w1", "ledger", "", fingerprint("b"), run1)), ErrInvalid)
	expect(t, "check fingerprint form", second(f.r.CheckEndpoint("w1", "ledger", "api", "zz", run1)), ErrInvalid)
	f.approve(at(ChangeRemove, "w2", "ledger", ""))
	f.install("w2", "ledger", "1.1.0")
	f.enable("w2", "ledger")
	expect(t, "unapproved endpoint", second(f.r.CheckEndpoint("w2", "ledger", "api", fingerprint("c"), run2)), ErrNotFound)
	expect(t, "removed tombstone", f.try(endpoint("w2", "api", "c")), ErrConflict)
	f.approve(endpoint("w2", "api", "d"))
	f.approve(endpoint("w2", "api", "e"))
	f.approve(endpoint("w2", "api", "g"))
	expect(t, "workspace fingerprints", f.try(endpoint("w2", "api", "h")), ErrLimit)
	f.approve(endpoint("w1", "api", "f"))
	f.must(second(f.r.Stop("w1", "ledger", ReasonOperator, ann)))
	reason(t, "stopped", second(f.r.CheckEndpoint("w1", "ledger", "api", fingerprint("f"), run1)), ReasonNotEnabled)
}

func TestEndpointRoomAndSwitch(t *testing.T) {
	build := func(endpoints int) world {
		l := limits
		l.Endpoints = endpoints
		f := setupWith(t, l)
		f.publisher("acme")
		two := manifest("ledger", "1.0.0")
		two.Endpoints = []string{"api", "web"}
		f.version(two)
		f.version(manifest("ledger", "1.1.0"))
		none := manifest("ledger", "1.2.0")
		none.Endpoints = nil
		f.version(none)
		f.workspace("w1", "run-1", false)
		f.install("w1", "ledger", "1.0.0")
		return f
	}
	endpoint := func(id, print string) Change {
		return Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Plugin: "ledger", Endpoint: Endpoint{ID: id, Fingerprint: fingerprint(print), Evidence: fingerprint("e")}}
	}
	f := build(1)
	f.approve(endpoint("web", "a"))
	expect(t, "room", f.try(endpoint("api", "b")), ErrLimit)
	f.approve(endpoint("web", "c"))
	g := build(2)
	g.approve(endpoint("web", "a"))
	g.approve(endpoint("api", "b"))
	if v := g.state("w1", "ledger"); len(v.Endpoints) != 2 || v.Endpoints[0].Endpoint.ID != "api" || v.Endpoints[1].Endpoint.ID != "web" {
		t.Errorf("endpoint order %+v", v)
	}
	g.approve(at(ChangeUpgrade, "w1", "ledger", "1.1.0"))
	if v := g.state("w1", "ledger"); len(v.Endpoints) != 1 || v.Endpoints[0].Endpoint.ID != "api" {
		t.Errorf("switch kept %+v", v)
	}
	g.approve(at(ChangeUpgrade, "w1", "ledger", "1.2.0"))
	g.approve(at(ChangeRollback, "w1", "ledger", "1.1.0"))
	if v := g.state("w1", "ledger"); len(v.Endpoints) != 0 {
		t.Errorf("rollback restored endpoints %+v", v)
	}
}

func TestStaleDuplicateEnable(t *testing.T) {
	f := standard(t)
	f.install("w1", "ledger", "1.1.0")
	first, err := f.r.Propose(at(ChangeEnable, "w1", "ledger", ""), ann)
	f.must(err)
	second, err := f.r.Propose(at(ChangeEnable, "w1", "ledger", ""), cat)
	f.must(err)
	f.must(func() error { _, err := f.r.Approve(first.Number, bob); return err }())
	f.must(func() error { _, err := f.r.Disable("w1", "ledger", ann); return err }())
	if _, err := f.r.Approve(second.Number, bob); err != nil {
		t.Fatal(err)
	}
	if v := f.state("w1", "ledger"); v.State != StateEnabled {
		t.Errorf("a pending second-human enable applies after a disable: %+v", v)
	}
}

func TestRefusalError(t *testing.T) {
	err := refuse(ReasonScope)
	if err.Error() != ErrRejected.Error()+": scope" || !errorsIs(err, ErrRejected) {
		t.Errorf("refusal %v", err)
	}
}
