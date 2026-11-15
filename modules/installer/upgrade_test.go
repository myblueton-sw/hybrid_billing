package installer

import (
	"strings"
	"testing"
	"time"
)

func second3[T any](_ T, err error) error { return err }

func upgraded(version string, from ...string) Release {
	rel := fixture(version)
	rel.UpgradeFrom = from
	rel.Services = []Service{
		{Name: "api", Artifact: "api", Ports: []Port{{Number: 8443, Protocol: ProtocolTCP}}, Volumes: []string{"config", "ledger"}},
		{Name: "report", Artifact: "worker", Ports: []Port{{Number: 9100, Protocol: ProtocolTCP}}, Volumes: []string{"archive"}},
	}
	rel.Volumes = []Volume{{Name: "archive", Data: true}, {Name: "config"}, {Name: "ledger", Data: true}}
	return rel
}

func (f world) install(site string, probe Actor) {
	f.t.Helper()
	run := f.finish(site, f.started(site, probe).ID, probe)
	code, err := f.r.IssueBootstrap(site, run.ID, time.Minute, probe)
	f.must(err)
	f.must(second(f.r.RegisterAdmin(code, cat)))
}

func hostOf(site string, rel Release) Observation {
	o := empty()
	o.Services = []string{"api", "worker"}
	o.Volumes = []string{"config", "ledger"}
	o.Ports = []Port{{Number: 8443, Protocol: ProtocolTCP}, {Number: 9000, Protocol: ProtocolTCP}, {Number: 9000, Protocol: ProtocolUDP}}
	return o
}

func (f world) upgradeTo(site, version string) ProposalView {
	f.t.Helper()
	v, err := f.r.PreviewUpgrade(site, version)
	f.must(err)
	return f.approve(Change{Kind: ChangeUpgrade, Acceptance: Acceptance{Site: site, Version: version, Hash: v.Hash}})
}

func (f world) drive(site, id string, probe Actor, through string) RunView {
	f.t.Helper()
	run, err := f.r.Run(site, id)
	f.must(err)
	digests := map[string]string{}
	for _, a := range f.r.releases[run.Version].release.Artifacts {
		digests["artifact:"+a.Name] = a.Digest
	}
	for run.Done < len(run.Steps) {
		step := run.Steps[run.Done]
		d := digests[step]
		if d == "" {
			d = sum(step)
		}
		run, err = f.r.Complete(site, id, step, Evidence{Digest: d}, probe)
		f.must(err)
		if step == through {
			break
		}
	}
	return run
}

func upgradeWorld(t *testing.T) world {
	f := standard(t)
	f.approve(signed(upgraded("2.0.0", "1.0.0")))
	f.install("s1", p1)
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	return f
}

func TestInstallRecord(t *testing.T) {
	f := upgradeWorld(t)
	v, err := f.r.Installed("s1")
	f.must(err)
	if v.Version != "1.0.0" || v.State != StateInstalled || len(v.Services) != 2 || v.Services[0].Name != "api" || v.Services[0].Ports[0] != "8443/tcp" ||
		len(v.Volumes) != 2 || v.Volumes[1] != (OwnedVolume{Host: "ledger", Logical: "ledger", Data: true}) || len(v.History) != 1 || v.History[0].Kind != RunInstall {
		t.Fatalf("installed %+v", v)
	}
	v.Services[0].Ports[0] = "x"
	if again, _ := f.r.Installed("s1"); again.Services[0].Ports[0] != "8443/tcp" {
		t.Error("view shares ports")
	}
	expect(t, "not installed", second(f.r.Installed("s2")), ErrNotFound)
	expect(t, "form", second(f.r.Installed("")), ErrInvalid)
	reason(t, "install again", second(f.r.Preview("s1", "1.0.0")), ReasonAlreadyInstalled)
}

func TestUpgradePreview(t *testing.T) {
	f := upgradeWorld(t)
	v, err := f.r.PreviewUpgrade("s1", "2.0.0")
	f.must(err)
	if v.Kind != RunUpgrade || v.Basis != BasisRecordName || len(v.Conflicts) != 0 {
		t.Fatalf("preview %+v", v)
	}
	if got := items(v.Services); got != "service:api:replace:,service:report:create:,service:worker:remove:" {
		t.Errorf("services %s", got)
	}
	if got := items(v.Ports); got != "port:api:8443/tcp:reuse:,port:report:9100/tcp:bind:" {
		t.Errorf("ports %s", got)
	}
	if got := items(v.Volumes); got != "volume:archive:create:,volume:config:retain:,volume:ledger:retain:" {
		t.Errorf("volumes %s", got)
	}
	if len(v.Data) != 2 || v.Data[0] != (DataEffect{Volume: "archive", Effect: EffectNew}) || v.Data[1] != (DataEffect{Volume: "ledger", Effect: EffectRetained}) {
		t.Errorf("data %+v", v.Data)
	}
	expect(t, "form", second(f.r.PreviewUpgrade("s1", "x")), ErrInvalid)
	expect(t, "site form", second(f.r.PreviewUpgrade("", "2.0.0")), ErrInvalid)
	expect(t, "missing", second(f.r.PreviewUpgrade("s1", "9.0.0")), ErrNotFound)
	expect(t, "missing site", second(f.r.PreviewUpgrade("s9", "2.0.0")), ErrNotFound)
	reason(t, "not installed", second(f.r.PreviewUpgrade("s2", "2.0.0")), ReasonNotInstalled)
	reason(t, "direction", second(f.r.PreviewUpgrade("s1", "1.0.0")), ReasonDirection)
	f.approve(signed(fixture("1.5.0")))
	reason(t, "not upgradable", second(f.r.PreviewUpgrade("s1", "1.5.0")), ReasonNotUpgradable)
	odd := upgraded("2.1.0", "1.0.0")
	odd.Platforms = []Platform{{OS: "linux", Arch: "arm64"}}
	f.approve(signed(odd))
	reason(t, "platform", second(f.r.PreviewUpgrade("s1", "2.1.0")), ReasonUnsupported)
	f.advance(11 * time.Minute)
	reason(t, "stale", second(f.r.PreviewUpgrade("s1", "2.0.0")), ReasonObservationStale)
	f.r.sites["s1"].observation = nil
	expect(t, "no observation", second(f.r.PreviewUpgrade("s1", "2.0.0")), ErrNotFound)
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	f.must(second(f.r.RevokeRelease("2.0.0", "Bad", ann)))
	reason(t, "revoked", second(f.r.PreviewUpgrade("s1", "2.0.0")), ReasonRevoked)
}

func TestUpgradeConflicts(t *testing.T) {
	f := upgradeWorld(t)
	o := hostOf("s1", fixture("1.0.0"))
	o.Services = []string{"report", "worker"}
	o.Volumes = []string{"archive", "ledger"}
	o.Ports = append(o.Ports, Port{Number: 9100, Protocol: ProtocolTCP})
	sortPorts(o.Ports)
	f.observe("s1", p1, o)
	v, err := f.r.PreviewUpgrade("s1", "2.0.0")
	f.must(err)
	want := "volume:archive:create:existing_volume,service:api:replace:owned_missing,volume:config:retain:owned_missing,port:report:9100/tcp:bind:port_in_use,service:report:create:service_exists"
	if got := items(v.Conflicts); got != want {
		t.Errorf("conflicts %s", got)
	}
	moved := upgraded("2.2.0", "1.0.0")
	moved.Services[1].Ports = []Port{{Number: 9000, Protocol: ProtocolTCP}}
	f.approve(signed(moved))
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, _ = f.r.PreviewUpgrade("s1", "2.2.0")
	if got := items(v.Conflicts); got != "port:report:9000/tcp:bind:port_moved" {
		t.Errorf("moved %s", got)
	}
	dropped := upgraded("2.3.0", "1.0.0")
	dropped.Services[0].Volumes = []string{"config"}
	dropped.Volumes = []Volume{{Name: "archive", Data: true}, {Name: "config"}}
	f.approve(signed(dropped))
	v, _ = f.r.PreviewUpgrade("s1", "2.3.0")
	if got := items(v.Conflicts); got != "volume:ledger:retain:data_volume_removed" {
		t.Errorf("dropped %s", got)
	}
	cooled := upgraded("2.4.0", "1.0.0")
	cooled.Volumes[2].Data = false
	f.approve(signed(cooled))
	v, _ = f.r.PreviewUpgrade("s1", "2.4.0")
	if got := items(v.Conflicts); got != "volume:ledger:create:data_volume_removed" {
		t.Errorf("cooled %s", got)
	}
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "2.0.0", Volumes: []VolumeOverride{{Volume: "archive", Host: "ledger"}, {Volume: "ledger", Host: "ledger-2"}}}})
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, _ = f.r.PreviewUpgrade("s1", "2.0.0")
	if got := items(v.Conflicts); got != "volume:ledger:create:volume_remapped" {
		t.Errorf("remapped %s", got)
	}
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "2.2.0"}})
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, _ = f.r.PreviewUpgrade("s1", "2.0.0")
	if len(v.Conflicts) != 0 {
		t.Errorf("empty config %s", items(v.Conflicts))
	}
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0", Volumes: []VolumeOverride{{Volume: "config", Host: "cfg"}}}})
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, _ = f.r.PreviewUpgrade("s1", "2.0.0")
	if got := items(v.Conflicts); got != "config:1.0.0::config_mismatch" {
		t.Errorf("mismatch %s", got)
	}
	dup := upgraded("2.5.0", "1.0.0")
	dup.Services[1].Ports = []Port{{Number: 8443, Protocol: ProtocolTCP}}
	f.approve(signed(dup))
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "2.5.0", Volumes: []VolumeOverride{{Volume: "archive", Host: "config"}}}})
	f.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, _ = f.r.PreviewUpgrade("s1", "2.5.0")
	want = "port:api:8443/tcp:reuse:port_duplicate,port:report:8443/tcp:bind:port_duplicate,port:report:8443/tcp:bind:port_moved,volume:config:create:volume_duplicate,volume:config:retain:volume_duplicate,volume:config:create:volume_remapped"
	if got := items(v.Conflicts); got != want {
		t.Errorf("duplicates %s", got)
	}
}

func sortPorts(ports []Port) {
	for i := 1; i < len(ports); i++ {
		for j := i; j > 0 && portLess(ports[j], ports[j-1]); j-- {
			ports[j], ports[j-1] = ports[j-1], ports[j]
		}
	}
}

func TestUpgradeRun(t *testing.T) {
	f := upgradeWorld(t)
	p := f.upgradeTo("s1", "2.0.0")
	reason(t, "run open preview", second(f.r.PreviewRollback("s1")), ReasonNotNeeded)
	run, err := f.r.StartRun("s1", p.Number, ann)
	f.must(err)
	want := "preflight,quiesce,backup,artifact:api,artifact:worker,volume:archive,remove:worker,service:api,service:report,verify"
	if run.Kind != RunUpgrade || strings.Join(run.Steps, ",") != want {
		t.Fatalf("run %+v", run)
	}
	reason(t, "preview while open", second(f.r.PreviewUpgrade("s1", "2.0.0")), ReasonRunOpen)
	reason(t, "rollback while open", second(f.r.PreviewRollback("s1")), ReasonRunOpen)
	reason(t, "accept while open", f.try(Change{Kind: ChangeUpgrade, Acceptance: Acceptance{Site: "s1", Version: "2.0.0", Hash: run.Hash}}), ReasonRunOpen)
	run = f.drive("s1", run.ID, p1, StepBackup)
	if run.Backup != sum(StepBackup) || strings.Join(run.Scope, ",") != "ledger" {
		t.Errorf("backup %+v", run)
	}
	reason(t, "no codes", second(f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)), ReasonNotAllowedState)
	run = f.drive("s1", run.ID, p1, "volume:archive")
	if f.r.sites["s1"].touched {
		t.Fatal("touched too early")
	}
	run = f.drive("s1", run.ID, p1, "remove:worker")
	if !f.r.sites["s1"].touched || f.r.sites["s1"].touchedAt != "2.0.0" {
		t.Fatal("not touched when reaching the first service step")
	}
	f.must(second(f.r.Interrupt("s1", run.ID, "pause", ann)))
	o := hostOf("s1", fixture("1.0.0"))
	o.Services, o.Volumes = []string{"api"}, []string{"archive", "config", "ledger"}
	f.observe("s1", p1, o)
	f.must(second(f.r.Resume("s1", run.ID, ann)))
	run = f.drive("s1", run.ID, p1, "service:api")
	run = f.drive("s1", run.ID, p1, StepVerify)
	if run.State != RunCompleted {
		t.Fatalf("completed %+v", run)
	}
	v, _ := f.r.Installed("s1")
	h := v.History[len(v.History)-1]
	if v.Version != "2.0.0" || v.State != StateInstalled || v.Touched || len(v.Services) != 2 || v.Services[1].Name != "report" || len(v.Volumes) != 3 || len(v.History) != 2 ||
		h.Kind != RunUpgrade || h.From != "1.0.0" || h.To != "2.0.0" || h.Run != run.ID || h.Backup != sum(StepBackup) || len(v.Backups) != 0 {
		t.Fatalf("installed %+v", v)
	}
	reason(t, "nothing to roll back", second(f.r.PreviewRollback("s1")), ReasonNotNeeded)
}

func TestDetachAndReattach(t *testing.T) {
	f := upgradeWorld(t)
	bare := upgraded("2.0.1", "1.0.0")
	bare.Services[0].Volumes = []string{"ledger"}
	bare.Volumes = []Volume{{Name: "archive", Data: true}, {Name: "ledger", Data: true}}
	f.approve(signed(bare))
	back := upgraded("3.0.0", "2.0.1")
	f.approve(signed(back))
	v, _ := f.r.PreviewUpgrade("s1", "2.0.1")
	if !strings.Contains(items(v.Volumes), "volume:config:detach:") {
		t.Fatalf("detach %s", items(v.Volumes))
	}
	run, _ := f.r.StartRun("s1", f.upgradeTo("s1", "2.0.1").Number, ann)
	f.drive("s1", run.ID, p1, StepVerify)
	inst, _ := f.r.Installed("s1")
	if inst.Volumes[1] != (OwnedVolume{Host: "config", Logical: "config", Detached: true}) {
		t.Fatalf("detached %+v", inst.Volumes)
	}
	gone := hostOf("s1", fixture("1.0.0"))
	gone.Services, gone.Volumes = []string{"api", "report"}, []string{"archive", "ledger"}
	f.observe("s1", p1, gone)
	if mv, _ := f.r.PreviewUpgrade("s1", "3.0.0"); !strings.Contains(items(mv.Volumes), "volume:config:create:") || len(mv.Conflicts) != 0 ||
		!listed(upgradeSteps(f.r.releases["3.0.0"].release, mv), "volume:config") {
		t.Errorf("missing detached volume %s %s", items(mv.Volumes), items(mv.Conflicts))
	}
	o := hostOf("s1", fixture("1.0.0"))
	o.Services, o.Volumes = []string{"api", "report"}, []string{"archive", "config", "ledger"}
	f.observe("s1", p1, o)
	v, _ = f.r.PreviewUpgrade("s1", "3.0.0")
	if !strings.Contains(items(v.Volumes), "volume:config:reattached:") || !strings.Contains(effects(v.Data), "config=stale_content") {
		t.Errorf("reattach %s %v", items(v.Volumes), v.Data)
	}
	o.Volumes = []string{"archive", "ledger"}
	f.observe("s1", p1, o)
	run, _ = f.r.StartRun("s1", f.upgradeTo("s1", "3.0.0").Number, ann)
	f.drive("s1", run.ID, p1, StepVerify)
	inst, _ = f.r.Installed("s1")
	if inst.Volumes[1].Detached {
		t.Errorf("reattached volume still detached %+v", inst.Volumes)
	}
}

func effects(list []DataEffect) string {
	out := []string{}
	for _, d := range list {
		out = append(out, d.Volume+"="+d.Effect)
	}
	return strings.Join(out, ",")
}
