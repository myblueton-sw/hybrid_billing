package installer

import (
	"strings"
	"testing"
)

func TestUpgradeConstants(t *testing.T) {
	if maxUpgradeFrom != 16 || maxDetached != 32 || maxHistory != 100 {
		t.Error("constants")
	}
	if compare("1.0.0", "2.0.0") != -1 || compare("2.0.0", "1.0.0") != 1 || compare("1.0.0", "1.0.0") != 0 {
		t.Error("compare")
	}
}

func TestAcceptanceForms(t *testing.T) {
	f := upgradeWorld(t)
	v, _ := f.r.PreviewUpgrade("s1", "2.0.0")
	expect(t, "upgrade admin", f.try(Change{Kind: ChangeUpgrade, Acceptance: Acceptance{Site: "s1", Version: "2.0.0", Hash: v.Hash, Admin: "cat"}}), ErrInvalid)
	expect(t, "upgrade run", f.try(Change{Kind: ChangeUpgrade, Acceptance: Acceptance{Site: "s1", Version: "2.0.0", Hash: v.Hash, Run: "run-1"}}), ErrInvalid)
	expect(t, "plan run", f.try(Change{Kind: ChangePlan, Acceptance: Acceptance{Site: "s1", Version: "2.0.0", Hash: v.Hash, Admin: "cat", Run: "run-1"}}), ErrInvalid)
	expect(t, "rollback admin", f.try(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: "run-1", Admin: "cat"}}), ErrInvalid)
	reason(t, "rollback on installed", second(f.r.Propose(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: "run-1"}}, ann)), ReasonNotNeeded)
}

func TestRollbackDedup(t *testing.T) {
	f := upgradeWorld(t)
	f.failAt("s1", "2.0.0", "volume:archive")
	first := f.r.lastFailed(f.r.sites["s1"])
	o := hostOf("s1", fixture("1.0.0"))
	o.Volumes = []string{"archive", "config", "ledger"}
	f.observe("s1", p1, o)
	v, _ := f.r.PreviewRollback("s1")
	a, err := f.r.Propose(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: first}}, ann)
	f.must(err)
	reason(t, "other run", second(f.r.Propose(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: "run-9"}}, cat)), ReasonPlanChanged)
	again, _ := f.r.Propose(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Run: first}}, ann)
	if again.Number != a.Number {
		t.Error("dedup")
	}
}

func TestRollbackRechecks(t *testing.T) {
	cases := map[string]func(f world){
		"probe": func(f world) {
			f.approve(Change{Kind: ChangeProbe, Site: Site{ID: "s1", Probe: "probe-x"}})
			o := hostOf("s1", fixture("1.0.0"))
			o.Volumes = []string{"archive", "config", "ledger"}
			f.observe("s1", Actor{ID: "probe-x", Kind: KindSystem}, o)
		},
		"revoke": func(f world) { f.must(second(f.r.RevokeRelease("1.0.0", "Bad", ann))) },
		"host": func(f world) {
			o := hostOf("s1", fixture("1.0.0"))
			o.Services = []string{"api"}
			o.Volumes = []string{"archive", "config", "ledger"}
			f.observe("s1", p1, o)
		},
	}
	want := map[string]string{"probe": ReasonPlanChanged, "revoke": ReasonPlanChanged, "host": ReasonPlanChanged}
	for name, change := range cases {
		f := upgradeWorld(t)
		f.failAt("s1", "2.0.0", "volume:archive")
		o := hostOf("s1", fixture("1.0.0"))
		o.Volumes = []string{"archive", "config", "ledger"}
		f.observe("s1", p1, o)
		p := f.rollback("s1")
		change(f)
		reason(t, name, second(f.r.StartRun("s1", p.Number, ann)), want[name])
	}
}

func TestBoundsAtTheEdge(t *testing.T) {
	f := upgradeWorld(t)
	s := f.r.sites["s1"]
	for index := 0; index < maxHistory-1; index++ {
		s.failedRuns = append(s.failedRuns, "f")
		s.backups = append(s.backups, backup{run: "b"})
	}
	run, _ := f.r.StartRun("s1", f.upgradeTo("s1", "2.0.0").Number, ann)
	f.drive("s1", run.ID, p1, StepBackup)
	f.must(second(f.r.Cancel("s1", run.ID, ann)))
	if len(s.failedRuns) != maxHistory || len(s.backups) != maxHistory || s.failedRuns[0] != "f" {
		t.Errorf("edge %d %d", len(s.failedRuns), len(s.backups))
	}
	s.history = nil
	for index := 0; index < maxHistory; index++ {
		s.remember(History{Run: names(maxHistory + 1)[index]})
	}
	s.remember(History{Run: "last"})
	if len(s.history) != maxHistory || s.history[0].Run != names(maxHistory + 1)[1] || s.history[maxHistory-1].Run != "last" {
		t.Error("history order")
	}
}

func TestSortConflicts(t *testing.T) {
	list := []Item{{Kind: "volume", Target: "a", Conflict: "x"}, {Kind: "service", Target: "b", Conflict: "x"}, {Kind: "service", Target: "a", Conflict: "x"}, {Kind: "port", Target: "z", Conflict: "a"}}
	sortConflicts(list)
	if items(list) != "port:z::a,service:a::x,service:b::x,volume:a::x" {
		t.Errorf("sorted %s", items(list))
	}
}

func TestDetachedCapEdge(t *testing.T) {
	f := upgradeWorld(t)
	rec := f.r.sites["s1"].record
	for _, name := range names(maxDetached) {
		rec.volumes[name] = &ownedVolume{logical: name, detached: true}
	}
	o := hostOf("s1", fixture("1.0.0"))
	o.Volumes = append(names(maxDetached), "config", "ledger")
	sortStrings(o.Volumes)
	o.Granted = []string{PermissionRunService}
	f.observe("s1", p1, o)
	v, err := f.r.PreviewUpgrade("s1", "2.0.0")
	f.must(err)
	if listed(v.Permissions, PermissionRunService) || !listed(v.Permissions, PermissionWriteConfig) {
		t.Errorf("permissions %v", v.Permissions)
	}
}

func TestTransitionHashInputs(t *testing.T) {
	f := upgradeWorld(t)
	s, rel := f.r.sites["s1"], f.r.releases["2.0.0"]
	v, _ := f.r.PreviewUpgrade("s1", "2.0.0")
	base := f.r.transitionHash(upgradeTag, s, rel, v)
	if base != v.Hash {
		t.Fatal("hash")
	}
	changes := map[string]func(){
		"tag":        func() {},
		"failed":     func() { s.failed = true },
		"touched":    func() { s.touched = true },
		"touched at": func() { s.touchedAt = "9.0.0" },
		"created":    func() { s.created["x"] = []string{"1/tcp"} },
		"created p":  func() { s.created["x"] = []string{"2/tcp"} },
		"vols":       func() { s.createdVols["x"] = &ownedVolume{logical: "x"} },
		"vol data":   func() { s.createdVols["x"] = &ownedVolume{logical: "x", data: true} },
		"vol det":    func() { s.createdVols["x"] = &ownedVolume{logical: "x", detached: true} },
		"vol logic":  func() { s.createdVols["x"] = &ownedVolume{logical: "y"} },
		"removed":    func() { s.removed["x"] = true },
		"failed run": func() { s.failedRuns = []string{"run-9"} },
		"backup":     func() { s.backups = []backup{{run: "r", digest: "d"}} },
		"scope":      func() { s.backups = []backup{{run: "r", digest: "d", scope: []string{"ledger"}}} },
		"record":     func() { s.record.version = "0.9.0" },
		"leftovers":  func() { v.Leftovers = []Item{{Kind: "volume", Target: "x"}} },
		"revision":   func() { s.revision++ },
	}
	seen := map[string]string{base: "base"}
	for name, change := range changes {
		g := upgradeWorld(t)
		s, rel = g.r.sites["s1"], g.r.releases["2.0.0"]
		v, _ = g.r.PreviewUpgrade("s1", "2.0.0")
		change()
		tag := upgradeTag
		if name == "tag" {
			tag = rollbackTag
		}
		got := g.r.transitionHash(tag, s, rel, v)
		if other, ok := seen[got]; ok {
			t.Errorf("%s hashes like %s", name, other)
		}
		seen[got] = name
	}
	if boolText(true) != "1" || boolText(false) != "0" {
		t.Error("bool text")
	}
}

const upgradeFromAnswer = "a1e4b3e29657993afc51b0b9d951e2f891c6a59fdd293604ad2f7eff74d7b22f"

func TestUpgradeFromEncoding(t *testing.T) {
	rel := fixture("1.0.0")
	rel.UpgradeFrom = []string{"0.9.0"}
	if got := digestOf(rel.encode()); got != upgradeFromAnswer {
		t.Errorf("encoding %s", got)
	}
	other := copyRelease(rel)
	other.UpgradeFrom = []string{"0.8.0"}
	none := copyRelease(rel)
	none.UpgradeFrom = nil
	if digestOf(other.encode()) == digestOf(rel.encode()) || digestOf(none.encode()) == digestOf(rel.encode()) {
		t.Error("upgrade-from not encoded")
	}
	c := copyRelease(rel)
	c.UpgradeFrom[0] = "x"
	if rel.UpgradeFrom[0] != "0.9.0" {
		t.Error("copy shares upgrade-from")
	}
}

func TestInstallCancelIsNotAFailedUpgrade(t *testing.T) {
	f := standard(t)
	run := f.started("s1", p1)
	run = f.finish("s1", run.ID, p1)
	f.must(second(f.r.Cancel("s1", run.ID, ann)))
	if f.r.sites["s1"].failed || len(f.r.sites["s1"].failedRuns) != 0 {
		t.Error("install cancel marked a failed upgrade")
	}
}

func TestCreatedDataVolumesKept(t *testing.T) {
	f := upgradeWorld(t)
	f.approve(signed(upgraded("3.0.0", "1.0.0", "2.0.0")))
	plain := fixture("2.5.0")
	plain.UpgradeFrom = []string{"1.0.0", "2.0.0"}
	f.approve(signed(plain))
	failed := f.failAt("s1", "2.0.0", "service:api")
	o := hostOf("s1", fixture("1.0.0"))
	o.Volumes = []string{"archive", "config", "ledger"}
	f.observe("s1", p1, o)
	run, err := f.r.StartRun("s1", f.upgradeTo("s1", "2.5.0").Number, ann)
	f.must(err)
	run = f.drive("s1", run.ID, p1, StepBackup)
	if strings.Join(run.Scope, ",") != "archive,ledger" {
		t.Errorf("scope %v", run.Scope)
	}
	f.drive("s1", run.ID, p1, StepVerify)
	inst, _ := f.r.Installed("s1")
	if inst.Volumes[0] != (OwnedVolume{Host: "archive", Logical: "archive", Data: true, Detached: true}) {
		t.Errorf("created data volume dropped %+v", inst.Volumes)
	}
	_ = failed
	f.approve(signed(upgraded("3.1.0", "2.5.0")))
	missing := hostOf("s1", fixture("1.0.0"))
	f.observe("s1", p1, missing)
	mv, err := f.r.PreviewUpgrade("s1", "3.1.0")
	f.must(err)
	if !strings.Contains(items(mv.Volumes), "volume:archive:create:") || !strings.Contains(effects(mv.Data), "archive=new") {
		t.Errorf("missing detached data volume %s %v", items(mv.Volumes), mv.Data)
	}
	g := upgradeWorld(t)
	g.failAt("s1", "2.0.0", "volume:archive")
	g.observe("s1", p1, hostOf("s1", fixture("1.0.0")))
	v, _ := g.r.PreviewUpgrade("s1", "2.0.0")
	if !strings.Contains(items(v.Volumes), "volume:archive:create:") {
		t.Errorf("missing adopted volume %s", items(v.Volumes))
	}
	h := upgradeWorld(t)
	h.failAt("s1", "2.0.0", "volume:archive")
	ho := hostOf("s1", fixture("1.0.0"))
	ho.Volumes = []string{"archive", "config", "ledger"}
	h.observe("s1", p1, ho)
	for _, name := range names(maxHistory) {
		h.r.sites["s1"].rolledOrder = append(h.r.sites["s1"].rolledOrder, name)
		h.r.sites["s1"].rolledBack[name] = true
	}
	p := h.rollback("s1")
	rb, _ := h.r.StartRun("s1", p.Number, ann)
	h.drive("s1", rb.ID, p1, StepVerify)
	hist, _ := h.r.Installed("s1")
	last := hist.History[len(hist.History)-1]
	if hs := h.r.sites["s1"]; len(hs.rolledOrder) != maxHistory || hs.rolledBack[names(maxHistory)[0]] || !hs.rolledBack[last.Runs[0]] {
		t.Error("rolled-back marks not bounded")
	}
	if last.Kind != RunRollback || len(last.Runs) != 1 || last.From != "" {
		t.Errorf("rollback history %+v", last)
	}
}

func TestFinishVolumeRules(t *testing.T) {
	f := upgradeWorld(t)
	bare := upgraded("2.0.1", "1.0.0")
	bare.Services[0].Volumes = []string{"ledger"}
	bare.Volumes = []Volume{{Name: "archive", Data: true}, {Name: "ledger", Data: true}}
	f.approve(signed(bare))
	f.approve(signed(func() Release {
		r := upgraded("2.0.2", "2.0.1")
		r.Services, r.Volumes = bare.Services, bare.Volumes
		return r
	}()))
	f.approve(signed(func() Release {
		r := upgraded("2.0.3", "2.0.2")
		r.Services, r.Volumes = bare.Services, bare.Volumes
		return r
	}()))
	run, _ := f.r.StartRun("s1", f.upgradeTo("s1", "2.0.1").Number, ann)
	f.drive("s1", run.ID, p1, StepVerify)
	kept := hostOf("s1", fixture("1.0.0"))
	kept.Services, kept.Volumes = []string{"api", "report"}, []string{"archive", "config", "ledger"}
	f.observe("s1", p1, kept)
	run, _ = f.r.StartRun("s1", f.upgradeTo("s1", "2.0.2").Number, ann)
	f.drive("s1", run.ID, p1, StepVerify)
	if vol := f.r.sites["s1"].record.volumes["config"]; vol == nil || !vol.detached {
		t.Fatal("observed detached volume dropped")
	}
	kept.Volumes = []string{"archive", "ledger"}
	f.observe("s1", p1, kept)
	run, _ = f.r.StartRun("s1", f.upgradeTo("s1", "2.0.3").Number, ann)
	f.drive("s1", run.ID, p1, StepVerify)
	if f.r.sites["s1"].record.volumes["config"] != nil {
		t.Error("unobserved detached volume kept")
	}
	g := upgradeWorld(t)
	g.failAt("s1", "2.0.0", "volume:archive")
	o := hostOf("s1", fixture("1.0.0"))
	o.Volumes = []string{"archive", "config", "ledger"}
	g.observe("s1", p1, o)
	run, _ = g.r.StartRun("s1", g.upgradeTo("s1", "2.0.0").Number, ann)
	g.drive("s1", run.ID, p1, StepVerify)
	if vol := g.r.sites["s1"].record.volumes["archive"]; vol == nil || vol.detached || !vol.data {
		t.Errorf("adopted data volume %+v", vol)
	}
}
