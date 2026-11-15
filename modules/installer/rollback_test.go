package installer

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func (f world) failAt(site, version, step string) RunView {
	f.t.Helper()
	run, err := f.r.StartRun(site, f.upgradeTo(site, version).Number, ann)
	f.must(err)
	if step != StepPreflight {
		run = f.drive(site, run.ID, p1, step)
	}
	run, err = f.r.Cancel(site, run.ID, ann)
	f.must(err)
	return run
}

func (f world) rollback(site string) ProposalView {
	f.t.Helper()
	v, err := f.r.PreviewRollback(site)
	f.must(err)
	inst, _ := f.r.Installed(site)
	return f.approve(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: site, Version: inst.Version, Hash: v.Hash, Run: f.r.lastFailed(f.r.sites[site])}})
}

func TestRollbackFlow(t *testing.T) {
	f := upgradeWorld(t)
	f.failAt("s1", "2.0.0", StepPreflight)
	reason(t, "no change step", second(f.r.PreviewRollback("s1")), ReasonNotNeeded)
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	failed := f.failAt("s1", "2.0.0", "volume:archive")
	inst, _ := f.r.Installed("s1")
	if inst.State != StateUpgradeFailed || inst.Version != "1.0.0" || inst.Touched {
		t.Fatalf("failed state %+v", inst)
	}
	o := hostOf("s1", fixture("1.0.0"))
	o.Volumes = []string{"archive", "config", "ledger"}
	f.observe("s1", p1, o)
	v, err := f.r.PreviewRollback("s1")
	f.must(err)
	if v.Kind != RunRollback || items(v.Services) != "service:api:restart:,service:worker:restart:" || items(v.Leftovers) != "volume:archive:leftover:" || len(v.Conflicts) != 0 {
		t.Fatalf("rollback preview %+v", v)
	}
	if items(v.Ports) != "port:api:8443/tcp:reuse:,port:worker:9000/tcp:reuse:,port:worker:9000/udp:reuse:" {
		t.Errorf("ports %s", items(v.Ports))
	}
	expect(t, "form", second(f.r.PreviewRollback("")), ErrInvalid)
	reason(t, "not installed", second(f.r.PreviewRollback("s2")), ReasonNotInstalled)
	bad := Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: "run-1"}}
	reason(t, "wrong run", f.try(bad), ReasonPlanChanged)
	expect(t, "form run", f.try(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash}}), ErrInvalid)
	reason(t, "wrong version", f.try(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "2.0.0", Hash: v.Hash, Run: failed.ID}}), ReasonPlanChanged)
	first := f.rollback("s1")
	second2 := f.approve(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: failed.ID, Admin: ""}})
	_ = second2
	run, err := f.r.StartRun("s1", first.Number, ann)
	f.must(err)
	if strings.Join(run.Steps, ",") != "preflight,artifact:api,artifact:worker,service:api,service:worker,verify" || run.Kind != RunRollback || run.Version != "1.0.0" {
		t.Fatalf("rollback run %+v", run)
	}
	f.must(second(f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: sum("p")}, p1)))
	reason(t, "digest", second(f.r.Complete("s1", run.ID, "artifact:api", Evidence{Digest: sum("evil")}, p1)), ReasonDigestMismatch)
	if inst, _ := f.r.Installed("s1"); inst.State != StateUpgradeFailed {
		t.Error("failed rollback cleared state")
	}
	f.observe("s1", p1, o)
	retry := f.rollback("s1")
	run, err = f.r.StartRun("s1", retry.Number, ann)
	f.must(err)
	run = f.drive("s1", run.ID, p1, StepVerify)
	inst, _ = f.r.Installed("s1")
	if run.State != RunCompleted || inst.State != StateInstalled || inst.Version != "1.0.0" || inst.History[len(inst.History)-1].Kind != RunRollback {
		t.Fatalf("rolled back %+v %+v", run, inst)
	}
	f.observe("s1", p1, o)
	reason(t, "replay", second(f.r.StartRun("s1", second2.Number, ann)), ReasonRolledBack)
	reason(t, "not needed", second(f.r.PreviewRollback("s1")), ReasonNotNeeded)
}

func TestRollbackRefusals(t *testing.T) {
	f := upgradeWorld(t)
	f.failAt("s1", "2.0.0", "remove:worker")
	reason(t, "touched", second(f.r.PreviewRollback("s1")), ReasonDataTouched)
	g := upgradeWorld(t)
	g.failAt("s1", "2.0.0", StepQuiesce)
	g.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, err := g.r.PreviewRollback("s1")
	g.must(err)
	if len(v.Conflicts) != 0 {
		t.Errorf("quiesce-only rollback %s", items(v.Conflicts))
	}
	p := g.rollback("s1")
	g.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "2.0.0"}})
	g.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	reason(t, "retired", second(g.r.StartRun("s1", p.Number, ann)), ReasonPlanChanged)
	o := hostOf("s1", fixture("1.0.0"))
	o.Services = []string{"api"}
	g.observe("s1", p1, o)
	v, _ = g.r.PreviewRollback("s1")
	if items(v.Conflicts) != "service:worker:restart:owned_missing" {
		t.Errorf("missing %s", items(v.Conflicts))
	}
	o.Volumes = []string{"config"}
	g.observe("s1", p1, o)
	v, _ = g.r.PreviewRollback("s1")
	if !strings.Contains(items(v.Conflicts), "volume:ledger:retain:owned_missing") {
		t.Errorf("volume missing %s", items(v.Conflicts))
	}
	g.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	g.must(second(g.r.RevokeRelease("1.0.0", "Bad", ann)))
	reason(t, "revoked", second(g.r.PreviewRollback("s1")), ReasonRevoked)
	g.advance(11 * time.Minute)
	g.must(second(g.r.ClearMark(Target{Kind: MarkRelease, Version: "1.0.0"}, ann)))
	reason(t, "stale", second(g.r.PreviewRollback("s1")), ReasonObservationStale)
}

func TestRemovedServicesAndAdoption(t *testing.T) {
	f := upgradeWorld(t)
	f.approve(signed(upgraded("3.0.0", "1.0.0", "2.0.0")))
	failed := f.failAt("s1", "2.0.0", "service:api")
	if !strings.HasPrefix(failed.Steps[failed.Done-1], "service:") {
		t.Fatal(failed)
	}
	o := hostOf("s1", fixture("1.0.0"))
	o.Services = []string{"api"}
	o.Volumes = []string{"archive", "config", "ledger"}
	f.observe("s1", p1, o)
	v, err := f.r.PreviewUpgrade("s1", "2.0.0")
	f.must(err)
	if len(v.Conflicts) != 0 || !strings.Contains(items(v.Volumes), "volume:archive:adopt_created:") {
		t.Fatalf("adoption %s / %s", items(v.Conflicts), items(v.Volumes))
	}
	f.approve(signed(fixture("1.9.0")))
	reason(t, "not upgradable", second(f.r.PreviewUpgrade("s1", "1.9.0")), ReasonNotUpgradable)
	low := upgraded("2.0.5", "1.0.0")
	f.approve(signed(low))
	reason(t, "data form other", second(f.r.PreviewUpgrade("s1", "2.0.5")), ReasonDataForm)
	f.must(second(f.r.PreviewUpgrade("s1", "3.0.0")))
	second3 := f.failAt("s1", "3.0.0", "service:api")
	if f.r.sites["s1"].touchedAt != "3.0.0" || second3.State != RunCancelled {
		t.Fatalf("touched at %s", f.r.sites["s1"].touchedAt)
	}
	f.observe("s1", p1, o)
	reason(t, "back to 2.0.0", second(f.r.PreviewUpgrade("s1", "2.0.0")), ReasonDataForm)
	remap := upgraded("4.0.0", "1.0.0", "3.0.0")
	remap.Services[1].Volumes = []string{"old"}
	remap.Volumes = []Volume{{Name: "config"}, {Name: "ledger", Data: true}, {Name: "old", Data: true}}
	f.approve(signed(remap))
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "4.0.0", Volumes: []VolumeOverride{{Volume: "old", Host: "archive"}}}})
	f.observe("s1", p1, o)
	v, _ = f.r.PreviewUpgrade("s1", "4.0.0")
	if !strings.Contains(items(v.Conflicts), "volume:archive:create:volume_remapped") {
		t.Errorf("adopted remap %s", items(v.Conflicts))
	}
}

func TestResumeUpgradeDefensive(t *testing.T) {
	f := upgradeWorld(t)
	run, _ := f.r.StartRun("s1", f.upgradeTo("s1", "2.0.0").Number, ann)
	f.drive("s1", run.ID, p1, StepBackup)
	f.must(second(f.r.Interrupt("s1", run.ID, "pause", ann)))
	o := hostOf("s1", fixture("1.0.0"))
	o.Services = []string{"api", "report", "worker"}
	f.observe("s1", p1, o)
	reason(t, "host changed", second(f.r.Resume("s1", run.ID, ann)), ReasonHostChanged)
	if inst, _ := f.r.Installed("s1"); inst.State != StateUpgradeFailed {
		t.Error("invalidated upgrade did not set the failed state")
	}
	g := upgradeWorld(t)
	g.approve(signed(upgraded("2.0.1", "1.0.0")))
	failRun, _ := g.r.StartRun("s1", g.upgradeTo("s1", "2.0.1").Number, ann)
	g.drive("s1", failRun.ID, p1, "remove:worker")
	g.must(second(g.r.Interrupt("s1", failRun.ID, "pause", ann)))
	g.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	g.must(second(g.r.RevokeRelease("2.0.1", "Bad", ann)))
	if inst, _ := g.r.Installed("s1"); inst.State != StateUpgradeFailed || !inst.Touched {
		t.Errorf("revoked upgrade %+v", inst)
	}
}

func TestDetachedCapAndRetirement(t *testing.T) {
	f := upgradeWorld(t)
	rec := f.r.sites["s1"].record
	for _, name := range names(maxDetached) {
		rec.volumes[name] = &ownedVolume{logical: name, detached: true}
	}
	o := hostOf("s1", fixture("1.0.0"))
	o.Volumes = append([]string{}, "config", "ledger")
	f.observe("s1", p1, o)
	f.must(second(f.r.PreviewUpgrade("s1", "2.0.0")))
	bare := upgraded("2.0.1", "1.0.0")
	bare.Services[0].Volumes = []string{"ledger"}
	bare.Volumes = []Volume{{Name: "archive", Data: true}, {Name: "ledger", Data: true}}
	f.approve(signed(bare))
	f.must(second(f.r.PreviewUpgrade("s1", "2.0.1")))
	o.Volumes = append(names(maxDetached), "config", "ledger")
	sortStrings(o.Volumes)
	f.observe("s1", p1, o)
	expect(t, "cap", second(f.r.PreviewUpgrade("s1", "2.0.1")), ErrLimit)
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	p := f.upgradeTo("s1", "2.0.0")
	f.approve(Change{Kind: ChangeProbe, Site: Site{ID: "s1", Probe: "probe-x"}})
	f.observe("s1", Actor{ID: "probe-x", Kind: KindSystem}, hostOf("s1", fixture("1.0.0")))
	reason(t, "retired", second(f.r.StartRun("s1", p.Number, ann)), ReasonPlanChanged)
}

func sortStrings(list []string) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j] < list[j-1]; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func TestHistoryBounds(t *testing.T) {
	f := upgradeWorld(t)
	s := f.r.sites["s1"]
	for index := 0; index < maxHistory+5; index++ {
		s.remember(History{Kind: RunUpgrade, Run: "x"})
	}
	if len(s.history) != maxHistory {
		t.Error("history")
	}
	for index := 0; index < maxHistory+1; index++ {
		s.failedRuns = append(s.failedRuns, "x")
		s.backups = append(s.backups, backup{})
	}
	s.failedRuns, s.backups = s.failedRuns[:maxHistory], s.backups[:maxHistory]
	run, _ := f.r.StartRun("s1", f.upgradeTo("s1", "2.0.0").Number, ann)
	f.drive("s1", run.ID, p1, StepBackup)
	f.must(second(f.r.Cancel("s1", run.ID, ann)))
	if len(s.failedRuns) != maxHistory || len(s.backups) != maxHistory || s.failedRuns[maxHistory-1] != run.ID {
		t.Errorf("bounded failures %d %d", len(s.failedRuns), len(s.backups))
	}
}

func TestUpgradeEdges(t *testing.T) {
	bad := map[string][]string{"form": {"x"}, "same": {"2.0.0"}, "higher": {"3.0.0"}, "order": {"1.1.0", "1.0.0"}, "twice": {"1.0.0", "1.0.0"}}
	for name, from := range bad {
		if upgraded("2.0.0", from...).valid() {
			t.Errorf("%s accepted", name)
		}
	}
	many := []string{}
	for index := 0; index <= maxUpgradeFrom; index++ {
		many = append(many, "1.0."+itoa(index))
	}
	if upgraded("2.0.0", many...).valid() || !upgraded("2.0.0", many[:maxUpgradeFrom]...).valid() || !upgraded("2.0.0", "1.2.0", "1.10.0").valid() {
		t.Error("upgrade-from bounds")
	}
	f := upgradeWorld(t)
	warm := upgraded("2.0.2", "1.0.0")
	warm.Volumes[1].Data = true
	f.approve(signed(warm))
	v, _ := f.r.PreviewUpgrade("s1", "2.0.2")
	if !strings.Contains(effects(v.Data), "config=stale_content") {
		t.Errorf("warm %v", v.Data)
	}
	f.approve(signed(upgraded("3.0.0", "1.0.0", "2.0.0")))
	failed := f.failAt("s1", "2.0.0", "service:report")
	if failed.Steps[failed.Done-1] != "service:report" {
		t.Fatal(failed)
	}
	o := hostOf("s1", fixture("1.0.0"))
	o.Services = []string{"api", "report"}
	o.Volumes = []string{"archive", "config", "ledger"}
	o.Ports = append(o.Ports, Port{Number: 9100, Protocol: ProtocolTCP})
	sortPorts(o.Ports)
	f.observe("s1", p1, o)
	v, err := f.r.PreviewUpgrade("s1", "3.0.0")
	f.must(err)
	if !strings.Contains(items(v.Services), "service:report:adopt_created:") || !strings.Contains(items(v.Ports), "port:report:9100/tcp:reuse:") || len(v.Conflicts) != 0 {
		t.Errorf("adopted service %s %s %s", items(v.Services), items(v.Ports), items(v.Conflicts))
	}
	g := upgradeWorld(t)
	bare := upgraded("2.0.1", "1.0.0")
	bare.Services[0].Volumes = []string{"ledger"}
	bare.Volumes = []Volume{{Name: "archive", Data: true}, {Name: "ledger", Data: true}}
	g.approve(signed(bare))
	g.approve(signed(upgraded("3.0.0", "2.0.1")))
	run, _ := g.r.StartRun("s1", g.upgradeTo("s1", "2.0.1").Number, ann)
	g.drive("s1", run.ID, p1, StepVerify)
	h := hostOf("s1", fixture("1.0.0"))
	h.Services, h.Volumes = []string{"api", "report"}, []string{"archive", "config", "ledger"}
	g.observe("s1", p1, h)
	g.failAt("s1", "3.0.0", StepBackup)
	g.observe("s1", p1, h)
	rv, err := g.r.PreviewRollback("s1")
	g.must(err)
	if strings.Contains(items(rv.Volumes), "config") || items(rv.Volumes) != "volume:archive:retain:,volume:ledger:retain:" {
		t.Errorf("rollback volumes %s", items(rv.Volumes))
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
