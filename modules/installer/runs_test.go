package installer

import (
	"strings"
	"testing"
	"time"
)

func TestAcceptance(t *testing.T) {
	f := standard(t)
	f.observe("s1", p1, empty())
	v := f.preview("s1", "1.0.0")
	plan := func(version, hash, admin string) Change {
		return Change{Kind: ChangePlan, Acceptance: Acceptance{Site: "s1", Version: version, Hash: hash, Admin: admin}}
	}
	expect(t, "form", f.try(plan("1.0.0", "x", "cat")), ErrInvalid)
	expect(t, "admin form", f.try(plan("1.0.0", v.Hash, "")), ErrInvalid)
	expect(t, "version form", f.try(plan("1", v.Hash, "cat")), ErrInvalid)
	expect(t, "missing release", f.try(plan("9.0.0", v.Hash, "cat")), ErrNotFound)
	reason(t, "wrong hash", f.try(plan("1.0.0", sum("x"), "cat")), ReasonPlanChanged)
	p, err := f.r.Propose(plan("1.0.0", v.Hash, "cat"), ann)
	f.must(err)
	o := empty()
	o.Volumes = []string{"ledger"}
	f.observe("s1", p1, o)
	reason(t, "changed before approval", second(f.r.Approve(p.Number, bob)), ReasonPlanChanged)
	conflicted := f.preview("s1", "1.0.0")
	reason(t, "conflicts", f.try(plan("1.0.0", conflicted.Hash, "cat")), ReasonConflicts)
}

func TestStartRun(t *testing.T) {
	f := standard(t)
	f.observe("s1", p1, empty())
	p := f.accept("s1", "1.0.0")
	reason(t, "assistant", second(f.r.StartRun("s1", p.Number, bot)), ReasonAINotAllowed)
	expect(t, "site form", second(f.r.StartRun("", p.Number, ann)), ErrInvalid)
	expect(t, "missing site", second(f.r.StartRun("s9", p.Number, ann)), ErrNotFound)
	expect(t, "missing proposal", second(f.r.StartRun("s1", 999, ann)), ErrNotFound)
	reason(t, "other site", second(f.r.StartRun("s2", p.Number, ann)), ReasonNotApproved)
	reason(t, "not a plan", second(f.r.StartRun("s1", 1, ann)), ReasonNotApproved)
	pending, _ := f.r.Propose(Change{Kind: ChangePlan, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: p.Change.Acceptance.Hash, Admin: "dan"}}, ann)
	reason(t, "pending", second(f.r.StartRun("s1", pending.Number, ann)), ReasonNotApproved)
	o := empty()
	o.Services = []string{"api"}
	f.observe("s1", p1, o)
	reason(t, "changed host", second(f.r.StartRun("s1", p.Number, ann)), ReasonPlanChanged)
	f.observe("s1", p1, empty())
	run, err := f.r.StartRun("s1", p.Number, ann)
	f.must(err)
	want := "preflight,artifact:api,artifact:worker,volume:config,volume:ledger,service:api,service:worker,register_admin"
	if run.ID != "run-1" || strings.Join(run.Steps, ",") != want || run.State != RunOpen || run.Admin != "cat" || run.Probe != "probe-1" || run.Hash != p.Change.Acceptance.Hash {
		t.Fatalf("run %+v", run)
	}
	run.Steps[0] = "x"
	if again, _ := f.r.Run("s1", "run-1"); again.Steps[0] != StepPreflight {
		t.Error("run view shares steps")
	}
	reason(t, "used", second(f.r.StartRun("s1", p.Number, ann)), ReasonAcceptanceUsed)
	second := f.accept("s1", "1.0.0")
	reason(t, "run open", second2(f.r.StartRun("s1", second.Number, ann)), ReasonRunOpen)
	f.must(second2(f.r.Interrupt("s1", "run-1", "maintenance", ann)))
	reason(t, "interrupted is active", second2(f.r.StartRun("s1", second.Number, ann)), ReasonRunOpen)
	f.must(second2(f.r.Cancel("s1", "run-1", ann)))
	run, err = f.r.StartRun("s1", second.Number, ann)
	if err != nil || run.ID != "run-2" {
		t.Fatalf("second run %+v %v", run, err)
	}
	expect(t, "run form", second2(f.r.Run("s1", "")), ErrInvalid)
	expect(t, "run missing", second2(f.r.Run("s1", "run-9")), ErrNotFound)
	expect(t, "run site", second2(f.r.Run("s9", "run-1")), ErrNotFound)
}

func second2[T any](_ T, err error) error { return err }

func TestStartRunUsesOverrides(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0", Volumes: []VolumeOverride{{Volume: "ledger", Host: "ledger-prod"}}}})
	run := f.started("s1", p1)
	if strings.Join(run.Steps, ",") != "preflight,artifact:api,artifact:worker,volume:config,volume:ledger-prod,service:api,service:worker,register_admin" {
		t.Errorf("steps %v", run.Steps)
	}
}

func TestRunHistoryBound(t *testing.T) {
	f := standard(t)
	for index := 0; index < limits.Runs+2; index++ {
		run := f.started("s1", p1)
		f.must(second(f.r.Cancel("s1", run.ID, ann)))
	}
	s := f.r.sites["s1"]
	if len(s.order) != limits.Runs || s.order[0] != "run-3" {
		t.Errorf("history %v", s.order)
	}
	expect(t, "purged", second(f.r.Run("s1", "run-1")), ErrNotFound)
	g := setupWith(t, Limits{Keys: 2, KeyLife: limits.KeyLife, Releases: 2, Sites: 2, Destinations: 1, Reports: 1, Runs: 1, ObservationLife: time.Hour, BootstrapLife: time.Hour, Proposals: 8})
	g.approve(keyChange("k1"))
	g.approve(signed(fixture("1.0.0")))
	g.site("s1", "probe-1")
	first := g.started("s1", p1)
	g.must(second(g.r.Cancel("s1", first.ID, ann)))
	kept := g.started("s1", p1)
	if order := g.r.sites["s1"].order; len(order) != 1 || order[0] != kept.ID {
		t.Errorf("history %v", order)
	}
}

func TestCompleteSteps(t *testing.T) {
	f := standard(t)
	run := f.started("s1", p1)
	expect(t, "form", second(f.r.Complete("s1", run.ID, "", Evidence{Digest: sum("x")}, p1)), ErrInvalid)
	expect(t, "evidence", second(f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: "x"}, p1)), ErrInvalid)
	reason(t, "foreign probe", second(f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: sum("x")}, p2)), ReasonNotProbe)
	expect(t, "missing run", second(f.r.Complete("s1", "run-9", StepPreflight, Evidence{Digest: sum("x")}, p1)), ErrNotFound)
	reason(t, "order", second(f.r.Complete("s1", run.ID, "artifact:api", Evidence{Digest: sum("api1.0.0")}, p1)), ReasonOutOfOrder)
	done, err := f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: sum("pre")}, p1)
	if err != nil || done.Done != 1 {
		t.Fatal(done, err)
	}
	if again, err := f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: sum("pre")}, p1); err != nil || again.Done != 1 {
		t.Errorf("repeat %+v %v", again, err)
	}
	reason(t, "repeat other evidence", second(f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: sum("other")}, p1)), ReasonOutOfOrder)
	f.must(second(f.r.Complete("s1", run.ID, "artifact:api", Evidence{Digest: sum("api1.0.0")}, p1)))
	reason(t, "mismatch", second(f.r.Complete("s1", run.ID, "artifact:worker", Evidence{Digest: sum("evil")}, p1)), ReasonDigestMismatch)
	failed, _ := f.r.Run("s1", run.ID)
	if failed.State != RunFailed || failed.Reason != ReasonDigestMismatch {
		t.Errorf("failed %+v", failed)
	}
	reason(t, "closed", second(f.r.Complete("s1", run.ID, "artifact:worker", Evidence{Digest: sum("worker1.0.0")}, p1)), ReasonRunClosed)
	reason(t, "resume failed", second(f.r.Resume("s1", run.ID, ann)), ReasonRunClosed)
	g := standard(t)
	r2 := g.started("s1", p1)
	r2 = g.finish("s1", r2.ID, p1)
	if r2.Steps[r2.Done] != StepRegisterAdmin {
		t.Fatalf("finished %+v", r2)
	}
	reason(t, "admin by complete", second(g.r.Complete("s1", r2.ID, StepRegisterAdmin, Evidence{Digest: sum("x")}, p1)), ReasonNotAllowedState)
	g.must(second(g.r.Interrupt("s1", r2.ID, "maintenance", p1)))
	reason(t, "interrupted", second(g.r.Complete("s1", r2.ID, StepRegisterAdmin, Evidence{Digest: sum("x")}, p1)), ReasonNotAllowedState)
}

func TestInterruptResumeCancel(t *testing.T) {
	f := standard(t)
	run := f.started("s1", p1)
	expect(t, "form", second(f.r.Interrupt("s1", run.ID, "", ann)), ErrInvalid)
	expect(t, "actor", second(f.r.Interrupt("s1", run.ID, "x", Actor{ID: "x"})), ErrInvalid)
	reason(t, "assistant", second(f.r.Interrupt("s1", run.ID, "x", bot)), ReasonAINotAllowed)
	reason(t, "foreign probe", second(f.r.Interrupt("s1", run.ID, "x", p2)), ReasonNotProbe)
	expect(t, "missing", second(f.r.Interrupt("s1", "run-9", "x", ann)), ErrNotFound)
	reason(t, "resume open", second(f.r.Resume("s1", run.ID, ann)), ReasonNotAllowedState)
	f.must(second(f.r.Complete("s1", run.ID, StepPreflight, Evidence{Digest: sum("p")}, p1)))
	f.must(second(f.r.Complete("s1", run.ID, "artifact:api", Evidence{Digest: sum("api1.0.0")}, p1)))
	f.must(second(f.r.Complete("s1", run.ID, "artifact:worker", Evidence{Digest: sum("worker1.0.0")}, p1)))
	f.must(second(f.r.Complete("s1", run.ID, "volume:config", Evidence{Digest: sum("v")}, p1)))
	f.must(second(f.r.Complete("s1", run.ID, "volume:ledger", Evidence{Digest: sum("v")}, p1)))
	f.must(second(f.r.Complete("s1", run.ID, "service:api", Evidence{Digest: sum("s")}, p1)))
	stopped, err := f.r.Interrupt("s1", run.ID, "power", p1)
	if err != nil || stopped.State != RunInterrupted || stopped.Reason != ReasonInterrupted || stopped.Note != "power" {
		t.Fatal(stopped, err)
	}
	if again, err := f.r.Interrupt("s1", run.ID, "power", ann); err != nil || again.State != RunInterrupted {
		t.Errorf("repeat interrupt %v", err)
	}
	reason(t, "resume assistant", second(f.r.Resume("s1", run.ID, bot)), ReasonAINotAllowed)
	expect(t, "resume missing", second(f.r.Resume("s1", "run-9", ann)), ErrNotFound)
	f.advance(11 * time.Minute)
	reason(t, "stale", second(f.r.Resume("s1", run.ID, ann)), ReasonObservationStale)
	created := empty()
	created.Volumes = []string{"config", "ledger"}
	created.Services = []string{"api"}
	created.Ports = []Port{{Number: 8443, Protocol: ProtocolTCP}}
	f.observe("s1", p1, created)
	resumed, err := f.r.Resume("s1", run.ID, ann)
	if err != nil || resumed.State != RunOpen || resumed.Reason != "" || resumed.Done != 6 {
		t.Fatalf("resumed %+v %v", resumed, err)
	}
	f.must(second(f.r.Interrupt("s1", run.ID, "power", p1)))
	foreign := created
	foreign.Services = []string{"api", "worker"}
	f.observe("s1", p1, foreign)
	reason(t, "host changed", second(f.r.Resume("s1", run.ID, ann)), ReasonHostChanged)
	if v, _ := f.r.Run("s1", run.ID); v.State != RunInvalidated || v.Reason != ReasonHostChanged {
		t.Errorf("invalidated %+v", v)
	}
	reason(t, "cancel invalidated", second(f.r.Cancel("s1", run.ID, ann)), ReasonRunClosed)
	reason(t, "interrupt invalidated", second(f.r.Interrupt("s1", run.ID, "x", ann)), ReasonRunClosed)
	g := standard(t)
	r := g.started("s1", p1)
	reason(t, "cancel assistant", second(g.r.Cancel("s1", r.ID, bot)), ReasonAINotAllowed)
	expect(t, "cancel missing", second(g.r.Cancel("s1", "run-9", ann)), ErrNotFound)
	c, err := g.r.Cancel("s1", r.ID, ann)
	if err != nil || c.State != RunCancelled {
		t.Fatal(c, err)
	}
	if again, err := g.r.Cancel("s1", r.ID, ann); err != nil || again.State != RunCancelled {
		t.Error("repeat cancel")
	}
	g.observe("s1", p1, empty())
	g.must(second(g.r.Preview("s1", "1.0.0")))
}

func TestPartialRunLeavesConflicts(t *testing.T) {
	f := standard(t)
	run := f.started("s1", p1)
	f.finish("s1", run.ID, p1)
	f.must(second(f.r.Cancel("s1", run.ID, ann)))
	o := empty()
	o.Volumes = []string{"config", "ledger"}
	o.Services = []string{"api", "worker"}
	f.observe("s1", p1, o)
	v := f.preview("s1", "1.0.0")
	if len(v.Conflicts) != 4 {
		t.Errorf("partial install conflicts %s", items(v.Conflicts))
	}
}

func TestInvalidationByChanges(t *testing.T) {
	changes := map[string]func(f world){
		"probe":  func(f world) { f.approve(Change{Kind: ChangeProbe, Site: Site{ID: "s1", Probe: "probe-new"}}) },
		"config": func(f world) { f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0"}}) },
		"revoke": func(f world) { f.must(second(f.r.RevokeRelease("1.0.0", "Bad build", ann))) },
		"key":    func(f world) { f.must(second(f.r.CompromiseKey("k1", "Leaked", ann))) },
	}
	reasons := map[string]string{"probe": ReasonProbeChanged, "config": ReasonConfigChanged, "revoke": ReasonReleaseRevoked, "key": ReasonReleaseRevoked}
	for name, change := range changes {
		for _, interrupted := range []bool{false, true} {
			f := standard(t)
			run := f.started("s1", p1)
			other := f.started("s2", p2)
			run = f.finish("s1", run.ID, p1)
			code, err := f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)
			f.must(err)
			if interrupted {
				f.must(second(f.r.Interrupt("s1", run.ID, "power", ann)))
			}
			spare := f.accept("s1", "1.0.0")
			change(f)
			v, _ := f.r.Run("s1", run.ID)
			if v.State != RunInvalidated || v.Reason != reasons[name] {
				t.Errorf("%s %v: %+v", name, interrupted, v)
			}
			reason(t, name+" code", second(f.r.RegisterAdmin(code, cat)), ReasonUnknownCode)
			if name == "probe" || name == "config" {
				if s := f.r.sites["s1"]; s.observation != nil {
					t.Errorf("%s kept the observation", name)
				}
				if o, _ := f.r.Run("s2", other.ID); o.State != RunOpen {
					t.Errorf("%s changed another site", name)
				}
			}
			f.r.sites["s1"].observation = nil
			f.observe("s1", Actor{ID: f.r.sites["s1"].view.Probe, Kind: KindSystem}, empty())
			reason(t, name+" spare", second(f.r.StartRun("s1", spare.Number, ann)), ReasonPlanChanged)
		}
	}
	f := standard(t)
	f.observe("s1", p1, empty())
	spare := f.accept("s1", "1.0.0")
	f.must(second(f.r.RevokeRelease("1.0.0", "Suspected", ann)))
	f.must(second(f.r.ClearMark(Target{Kind: MarkRelease, Version: "1.0.0"}, ann)))
	reason(t, "no revival", second(f.r.StartRun("s1", spare.Number, ann)), ReasonPlanChanged)
}

func TestResumeDefensive(t *testing.T) {
	cases := map[string]func(f world){
		"revoked":  func(f world) { f.r.releases["1.0.0"].revocation = &Mark{By: "x", Reason: "r"} },
		"revision": func(f world) { f.r.sites["s1"].revision++ },
		"probe":    func(f world) { f.r.sites["s1"].view.Probe = "probe-z" },
	}
	want := map[string]string{"revoked": ReasonReleaseRevoked, "revision": ReasonConfigChanged, "probe": ReasonProbeChanged}
	for name, change := range cases {
		f := standard(t)
		run := f.started("s1", p1)
		f.must(second(f.r.Interrupt("s1", run.ID, "power", ann)))
		change(f)
		reason(t, name, second(f.r.Resume("s1", run.ID, ann)), want[name])
		if v, _ := f.r.Run("s1", run.ID); v.State != RunInvalidated {
			t.Errorf("%s %+v", name, v)
		}
	}
}

func TestMarks(t *testing.T) {
	f := standard(t)
	expect(t, "form", second(f.r.RevokeRelease("1", "r", ann)), ErrInvalid)
	expect(t, "reason", second(f.r.RevokeRelease("1.0.0", "", ann)), ErrInvalid)
	expect(t, "missing", second(f.r.RevokeRelease("9.0.0", "r", ann)), ErrNotFound)
	expect(t, "key form", second(f.r.CompromiseKey("", "r", ann)), ErrInvalid)
	expect(t, "key missing", second(f.r.CompromiseKey("k9", "r", ann)), ErrNotFound)
	reason(t, "assistant", second(f.r.RevokeRelease("1.0.0", "r", bot)), ReasonAINotAllowed)
	expect(t, "target", second(f.r.ConfirmMark(Target{Kind: "x"}, bob)), ErrInvalid)
	m, err := f.r.RevokeRelease("1.0.0", "Bad build", ann)
	if err != nil || m.By != "ann" {
		t.Fatal(m, err)
	}
	if again, _ := f.r.RevokeRelease("1.0.0", "Other", bob); again.Reason != "Bad build" {
		t.Error("repeat mark")
	}
	_, mark, _ := f.r.Release("1.0.0")
	if mark == nil || mark.By != "ann" {
		t.Errorf("mark view %+v", mark)
	}
	target := Target{Kind: MarkRelease, Version: "1.0.0"}
	reason(t, "self confirm", second(f.r.ConfirmMark(target, ann)), ReasonSelfApproval)
	reason(t, "clear other", second(f.r.ClearMark(target, bob)), ReasonNotProposer)
	reason(t, "confirm assistant", second(f.r.ConfirmMark(target, bot)), ReasonAINotAllowed)
	c, err := f.r.ConfirmMark(target, bob)
	if err != nil || c.ConfirmedBy != "bob" {
		t.Fatal(c, err)
	}
	if again, err := f.r.ConfirmMark(target, bob); err != nil || again.ConfirmedBy != "bob" {
		t.Error("repeat confirm")
	}
	reason(t, "third", second(f.r.ConfirmMark(target, cat)), ReasonNotPending)
	reason(t, "clear confirmed", second(f.r.ClearMark(target, ann)), ReasonNotPending)
	expect(t, "unmarked", second(f.r.ConfirmMark(Target{Kind: MarkCompromise, Key: "k1"}, bob)), ErrNotFound)
	f.must(second(f.r.CompromiseKey("k1", "Leaked", ann)))
	if !f.r.keys["k1"].retired {
		t.Error("compromise did not retire")
	}
	f.must(second(f.r.ClearMark(Target{Kind: MarkCompromise, Key: "k1"}, ann)))
	if !f.r.keys["k1"].retired {
		t.Error("clear un-retired the key")
	}
}

func TestBootstrap(t *testing.T) {
	f := standard(t)
	run := f.started("s1", p1)
	reason(t, "early", second(f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)), ReasonOutOfOrder)
	run = f.finish("s1", run.ID, p1)
	expect(t, "ttl low", second(f.r.IssueBootstrap("s1", run.ID, time.Minute-1, p1)), ErrInvalid)
	expect(t, "ttl high", second(f.r.IssueBootstrap("s1", run.ID, time.Hour+1, p1)), ErrInvalid)
	reason(t, "foreign probe", second(f.r.IssueBootstrap("s1", run.ID, time.Minute, p2)), ReasonNotProbe)
	f.random.err = boom
	reason(t, "entropy", second(f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)), ReasonDependency)
	f.random.err = nil
	first, err := f.r.IssueBootstrap("s1", run.ID, time.Hour, p1)
	f.must(err)
	if !strings.HasPrefix(first, codePrefix) || len(first) != 47 {
		t.Fatalf("code %s", first)
	}
	f.random.repeat = true
	reason(t, "collision", second(f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)), ReasonDependency)
	f.random.repeat = false
	code, err := f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)
	f.must(err)
	reason(t, "replaced", second(f.r.RegisterAdmin(first, cat)), ReasonUnknownCode)
	for hash := range f.r.codes {
		if strings.Contains(string(hash[:]), code) || hash == codeHash(code) && len(f.r.codes) != 1 {
			t.Error("raw or extra code stored")
		}
	}
	expect(t, "form", second(f.r.RegisterAdmin("admin-token", cat)), ErrInvalid)
	expect(t, "non-canonical", second(f.r.RegisterAdmin(codePrefix+strings.Repeat("A", 42)+"B", cat)), ErrInvalid)
	reason(t, "assistant", second(f.r.RegisterAdmin(code, bot)), ReasonAINotAllowed)
	reason(t, "unknown", second(f.r.RegisterAdmin(codePrefix+strings.Repeat("A", 43), cat)), ReasonUnknownCode)
	reason(t, "nominee", second(f.r.RegisterAdmin(code, ann)), ReasonNotNominee)
	expect(t, "no admin yet", second(f.r.Administrator("s1")), ErrNotFound)
	done, err := f.r.RegisterAdmin(code, cat)
	if err != nil || done.State != RunCompleted || done.Done != len(done.Steps) {
		t.Fatalf("registered %+v %v", done, err)
	}
	if admin, err := f.r.Administrator("s1"); err != nil || admin != "cat" {
		t.Errorf("admin %s %v", admin, err)
	}
	expect(t, "admin form", second(f.r.Administrator("")), ErrInvalid)
	expect(t, "admin missing", second(f.r.Administrator("s9")), ErrNotFound)
	reason(t, "replay", second(f.r.RegisterAdmin(code, cat)), ReasonUnknownCode)
	reason(t, "closed", second(f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)), ReasonRunClosed)
	f.observe("s1", p1, empty())
	reason(t, "installed", second(f.r.Preview("s1", "1.0.0")), ReasonAlreadyInstalled)
	if len(f.r.codes) != 0 {
		t.Error("codes left")
	}
}

func TestBootstrapExpiryAndStale(t *testing.T) {
	f := standard(t)
	run := f.finish("s1", f.started("s1", p1).ID, p1)
	code, _ := f.r.IssueBootstrap("s1", run.ID, time.Minute, p1)
	f.advance(time.Minute - time.Nanosecond)
	reason(t, "nominee", second(f.r.RegisterAdmin(code, ann)), ReasonNotNominee)
	f.advance(time.Nanosecond)
	reason(t, "expired", second(f.r.RegisterAdmin(code, cat)), ReasonExpired)
	reason(t, "deleted", second(f.r.RegisterAdmin(code, cat)), ReasonUnknownCode)
	cases := map[string]func(f world, id string){
		"state":    func(f world, id string) { f.r.sites["s1"].runs[id].view.State = RunInterrupted },
		"step":     func(f world, id string) { f.r.sites["s1"].runs[id].view.Done-- },
		"probe":    func(f world, id string) { f.r.sites["s1"].view.Probe = "probe-z" },
		"revision": func(f world, id string) { f.r.sites["s1"].revision++ },
		"run":      func(f world, id string) { delete(f.r.sites["s1"].runs, id) },
	}
	for name, change := range cases {
		g := standard(t)
		r := g.finish("s1", g.started("s1", p1).ID, p1)
		c, _ := g.r.IssueBootstrap("s1", r.ID, time.Minute, p1)
		change(g, r.ID)
		reason(t, name, second(g.r.RegisterAdmin(c, cat)), ReasonStaleCode)
		reason(t, name+" deleted", second(g.r.RegisterAdmin(c, cat)), ReasonUnknownCode)
	}
	h := standard(t)
	r := h.finish("s1", h.started("s1", p1).ID, p1)
	c, _ := h.r.IssueBootstrap("s1", r.ID, time.Minute, p1)
	h.must(second(h.r.Cancel("s1", r.ID, ann)))
	reason(t, "cancel deletes", second(h.r.RegisterAdmin(c, cat)), ReasonUnknownCode)
	i := standard(t)
	r = i.finish("s1", i.started("s1", p1).ID, p1)
	i.r.sites["s1"].runs[r.ID].view.Steps[2] = "artifact:other"
	reason(t, "foreign digest skipped", second(i.r.Complete("s1", r.ID, "x", Evidence{Digest: sum("x")}, p1)), ReasonOutOfOrder)
}

func TestDiagnostics(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeDestination, Destination: Destination{Site: "s1", ID: "erp", Host: "erp.internal", Port: 443, Ranges: []string{"10.1.0.0/16"}}})
	f.approve(Change{Kind: ChangeDestination, Destination: Destination{Site: "s1", ID: "proxy", Host: "10.2.0.1", Port: 3128, Ranges: []string{"10.2.0.0/16"}}})
	expect(t, "check form", second(f.r.CheckTarget("s1", "", p1)), ErrInvalid)
	reason(t, "check foreign", second(f.r.CheckTarget("s1", "erp", p2)), ReasonNotProbe)
	reason(t, "not approved", second(f.r.CheckTarget("s1", "crm", p1)), ReasonNotApproved)
	tls := Result{Kind: DiagnoseTLS, Failure: "unknown_authority", Address: "10.1.2.3"}
	g, err := f.r.Report("s1", "erp", tls, p1)
	if err != nil || strings.Join(g.Steps, ",") != "add_internal_ca_to_product_trust,verify_chain" || g.Cause != "untrusted_issuer" {
		t.Fatalf("guidance %+v %v", g, err)
	}
	bad := map[string]Result{
		"kind":          {Kind: "http", Failure: "ok", Address: "10.1.2.3"},
		"pair":          {Kind: DiagnoseTLS, Failure: "refused", Address: "10.1.2.3"},
		"address":       {Kind: DiagnoseTLS, Failure: "expired", Address: "010.1.2.3"},
		"no address":    {Kind: DiagnoseTLS, Failure: "expired"},
		"zone":          {Kind: DiagnoseTCP, Failure: "refused", Address: "fe80::1%eth0"},
		"dns address":   {Kind: DiagnoseDNS, Failure: "dns_failure", Address: "10.1.2.3"},
		"via form":      {Kind: DiagnoseTLS, Failure: "expired", Address: "10.1.2.3", Via: "a b"},
		"upper address": {Kind: DiagnoseTLS, Failure: "expired", Address: "2001:DB8::1"},
	}
	for name, res := range bad {
		expect(t, name, second(f.r.Report("s1", "erp", res, p1)), ErrInvalid)
	}
	expect(t, "destination form", second(f.r.Report("s1", "", tls, p1)), ErrInvalid)
	reason(t, "foreign", second(f.r.Report("s1", "erp", tls, p2)), ReasonNotProbe)
	reason(t, "unapproved", second(f.r.Report("s1", "crm", tls, p1)), ReasonNotApproved)
	reason(t, "outside", second(f.r.Report("s1", "erp", Result{Kind: DiagnoseTLS, Failure: "expired", Address: "10.9.0.1"}, p1)), ReasonAddressNotApproved)
	reason(t, "via unknown", second(f.r.Report("s1", "erp", Result{Kind: DiagnoseProxy, Failure: "proxy_auth", Address: "10.1.2.3", Via: "other"}, p1)), ReasonNotApproved)
	reason(t, "via self", second(f.r.Report("s1", "erp", Result{Kind: DiagnoseProxy, Failure: "proxy_auth", Address: "10.1.2.3", Via: "erp"}, p1)), ReasonNotApproved)
	f.must(second(f.r.Report("s1", "erp", Result{Kind: DiagnoseProxy, Failure: "proxy_auth", Address: "10.2.0.1", Via: "proxy"}, p1)))
	dns, err := f.r.Report("s1", "erp", Result{Kind: DiagnoseDNS, Failure: "dns_failure"}, p1)
	if err != nil || dns.Steps[0] != "check_dns" {
		t.Errorf("dns %+v %v", dns, err)
	}
	f.must(second(f.r.Report("s1", "erp", Result{Kind: DiagnoseDNS, Failure: "ok", Address: "10.1.0.9"}, p1)))
	reports, err := f.r.Reports("s1")
	if err != nil || len(reports) != limits.Reports || reports[0].Kind != DiagnoseProxy || reports[2].Failure != "ok" {
		t.Errorf("reports %+v %v", reports, err)
	}
	expect(t, "reports form", second(f.r.Reports("")), ErrInvalid)
}

func TestRemainingRules(t *testing.T) {
	f := standard(t)
	f.observe("s1", p1, empty())
	v := f.preview("s1", "1.0.0")
	f.must(second(f.r.RevokeRelease("1.0.0", "Bad", ann)))
	reason(t, "plan revoked", f.try(Change{Kind: ChangePlan, Acceptance: Acceptance{Site: "s1", Version: "1.0.0", Hash: v.Hash, Admin: "cat"}}), ReasonRevoked)
	expect(t, "clear form", second(f.r.ClearMark(Target{Kind: "x"}, ann)), ErrInvalid)
	reason(t, "clear assistant", second(f.r.ClearMark(Target{Kind: MarkRelease, Version: "1.0.0"}, bot)), ReasonAINotAllowed)
	g := standard(t)
	g.observe("s1", p1, empty())
	p := g.accept("s1", "1.0.0")
	g.advance(11 * time.Minute)
	reason(t, "start stale", second(g.r.StartRun("s1", p.Number, ann)), ReasonObservationStale)
}

func TestRefusalError(t *testing.T) {
	err := refuse(ReasonExpired)
	if err.Error() != ErrRejected.Error()+": expired" {
		t.Error(err)
	}
	expect(t, "unwrap", err, ErrRejected)
}
