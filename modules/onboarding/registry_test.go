package onboarding

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	start      = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	testLimits = Limits{Projects: 3, Writes: 200, Reserve: 3, MaxValidity: 48 * time.Hour, MaxSetup: 8 * time.Hour}
)

const day = 24 * time.Hour

type desk struct {
	t      testing.TB
	r      *Registry
	now    *time.Time
	space  string
	id     string
	writes *int
}

func open(t testing.TB, limits Limits, path Path) desk {
	t.Helper()
	now := start
	registry, err := NewRegistry(limits, "v1", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	d := desk{t: t, r: registry, now: &now, space: "w1", id: "p1", writes: &count}
	if created, err := registry.Create("w1", "p1", "source-1", path, "hash-1"); err != nil || !created {
		t.Fatalf("create: %v %v", created, err)
	}
	return d
}

func (d desk) pass(duration time.Duration) { *d.now = d.now.Add(duration) }

func (d desk) sequence() uint64 {
	view, err := d.r.Readiness(d.space, d.id)
	if err != nil {
		d.t.Fatal(err)
	}
	return view.Sequence
}

func (d desk) next() string {
	*d.writes++
	return fmt.Sprintf("write-%d", *d.writes)
}

func (d desk) record(name string, given Run) error {
	if given.Validity == 0 {
		given.Validity = day
	}
	if given.Version == "" {
		given.Version = "v1"
	}
	if given.Detail == "" {
		given.Detail = "detail-" + name
	}
	_, err := d.r.RecordRun(d.space, d.id, d.sequence(), name, given)
	return err
}

func (d desk) run(name string, stage Stage, outcome Outcome, role Role) error {
	return d.record(name, Run{Stage: stage, Outcome: outcome, Role: role})
}

func (d desk) reach(stage Stage, name string) error {
	_, err := d.r.Reach(d.space, d.id, d.sequence(), d.next(), stage, name)
	return err
}

func (d desk) grant(role Role) error {
	duration, condition := time.Duration(0), Condition("")
	if role == Setup {
		duration, condition = 2*time.Hour, AfterSetupComplete
	}
	_, err := d.r.GrantRole(d.space, d.id, d.sequence(), d.next(), role, "holder-"+string(role), duration, condition)
	return err
}

func (d desk) withdraw(role Role) error {
	_, err := d.r.WithdrawRole(d.space, d.id, d.next(), role)
	return err
}

func (d desk) change(hash string, from Stage) error {
	_, err := d.r.ChangeConfig(d.space, d.id, d.next(), hash, from)
	return err
}

func (d desk) bill2(expected uint64) (string, error) {
	_, _, err := d.r.OpenBilling(d.space, d.id, expected, d.next())
	return "", err
}

func (d desk) bill() (string, error) {
	_, blockers, err := d.r.OpenBilling(d.space, d.id, d.sequence(), d.next())
	parts := []string{}
	for _, blocker := range blockers {
		parts = append(parts, string(blocker.Stage)+":"+string(blocker.Reason))
	}
	return strings.Join(parts, " "), err
}

var short = map[Stage]string{Authenticated: "auth", OperationalRead: "read", FirstData: "data", FullCoverage: "cover", FocusConformance: "focus", Reconciled: "recon", BillingOpen: "bill"}

func (d desk) shape() string {
	d.t.Helper()
	view, err := d.r.Readiness(d.space, d.id)
	if err != nil {
		d.t.Fatal(err)
	}
	parts := []string{}
	for _, stage := range view.Stages {
		text := short[stage.Stage] + "=" + string(stage.Status)
		if stage.Reason != "" {
			text += "/" + string(stage.Reason)
		}
		if stage.Empty {
			text += "/empty"
		}
		if !stage.Prerequisites {
			text += "/alone"
		}
		parts = append(parts, text)
	}
	state := "closed"
	if view.BillingOpen {
		state = "open"
	}
	return fmt.Sprintf("%s | %d/%d %s", strings.Join(parts, " "), view.Progress, view.Required, state)
}

func (d desk) is(name, want string) {
	d.t.Helper()
	if got := d.shape(); got != want {
		d.t.Errorf("%s:\n got %s\nwant %s", name, got, want)
	}
}

func (d desk) ok(name string, err error) {
	d.t.Helper()
	if err != nil {
		d.t.Errorf("%s: %v", name, err)
	}
}

func (d desk) no(name string, err, want error) {
	d.t.Helper()
	if !errors.Is(err, want) {
		d.t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func (d desk) prove(suffix string, stages ...Stage) {
	d.t.Helper()
	role := CollectorReader
	if view, _ := d.r.Readiness(d.space, d.id); !view.Path.connected() {
		role = ""
	}
	for _, stage := range stages {
		name := short[stage] + suffix
		d.ok("run "+name, d.run(name, stage, Passed, role))
		d.ok("reach "+name, d.reach(stage, name))
	}
}

func ready(t testing.TB) desk {
	d := open(t, testLimits, ExistingAccount)
	d.ok("the collector's grant", d.grant(CollectorReader))
	d.prove("-1", order[:6]...)
	return d
}

const allHold = "auth=holding read=holding data=holding cover=holding focus=holding recon=holding"

func second[T any](_ T, err error) error { return err }

func TestProjectsAndPaths(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	d.is("a new connected project", "auth=missing read=missing/alone data=missing/alone cover=missing/alone focus=missing/alone recon=missing/alone bill=missing/alone | 0/7 closed")
	view, _ := d.r.Readiness("w1", "p1")
	if view.Path != ExistingAccount || view.Source != "source-1" || view.Hash != "hash-1" || view.Generation != 1 || view.Sequence != 0 || !view.AsOf.Equal(start) || len(view.Roles) != 0 {
		t.Errorf("the stored project: %+v", view)
	}
	if created, err := d.r.Create("w1", "p1", "source-1", ExistingAccount, "hash-1"); created || err != nil {
		t.Errorf("the same creation again: %v %v", created, err)
	}
	for name, attempt := range map[string]func() (bool, error){
		"another source": func() (bool, error) { return d.r.Create("w1", "p1", "source-2", ExistingAccount, "hash-1") },
		"another path":   func() (bool, error) { return d.r.Create("w1", "p1", "source-1", FileImport, "hash-1") },
		"another hash":   func() (bool, error) { return d.r.Create("w1", "p1", "source-1", ExistingAccount, "hash-2") },
	} {
		d.no(name, second(attempt()), ErrConflict)
	}
	for name, attempt := range map[string]func() (bool, error){
		"no workspace":                func() (bool, error) { return d.r.Create("", "p2", "s", FileImport, "h") },
		"a project that is text":      func() (bool, error) { return d.r.Create("w1", "p 2", "s", FileImport, "h") },
		"a source with a slash":       func() (bool, error) { return d.r.Create("w1", "p2", "arn:aws:iam::1:role/x", FileImport, "h") },
		"a source that is an address": func() (bool, error) { return d.r.Create("w1", "p2", "ops@example.invalid", FileImport, "h") },
		"an unknown path":             func() (bool, error) { return d.r.Create("w1", "p2", "s", "reseller", "h") },
		"the path none":               func() (bool, error) { return d.r.Create("w1", "p2", "s", NoAlternative, "h") },
		"no hash":                     func() (bool, error) { return d.r.Create("w1", "p2", "s", FileImport, "") },
	} {
		d.no(name, second(attempt()), ErrInvalid)
	}
	for _, id := range []string{"p2", "p3"} {
		if created, err := d.r.Create("w1", id, "source-1", ParentOperator, "hash-1"); !created || err != nil {
			t.Errorf("project %s: %v %v", id, created, err)
		}
	}
	d.no("a fourth project in the workspace", second(d.r.Create("w1", "p4", "s", FileImport, "h")), ErrLimit)
	if created, err := d.r.Create("w1", "p1", "source-1", ExistingAccount, "hash-1"); created || err != nil {
		t.Errorf("repeating a creation in a full workspace: %v %v", created, err)
	}
	if created, err := d.r.Create("w2", "p1", "other", FileImport, "hash-9"); !created || err != nil {
		t.Errorf("the same identifier in another workspace: %v %v", created, err)
	}
	other := d
	other.space = "w2"
	other.is("a file-import project has five stages and no roles", "data=missing cover=missing/alone focus=missing/alone recon=missing/alone bill=missing/alone | 0/5 closed")
	d.is("the first project is untouched", "auth=missing read=missing/alone data=missing/alone cover=missing/alone focus=missing/alone recon=missing/alone bill=missing/alone | 0/7 closed")
	if _, err := d.r.Readiness("w3", "p1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a project of an unknown workspace: %v", err)
	}
	if _, err := d.r.Runs("w2", "p2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a project of another workspace: %v", err)
	}
	if len(Required(ParentOperator)) != 7 || len(Required(FileImport)) != 5 || Required("x") != nil || Required(FileImport)[0] != FirstData {
		t.Errorf("the required stages")
	}
	other.no("a role on a file path", other.grant(CollectorReader), ErrRejected)
	other.no("a withdrawal on a file path", other.withdraw(CollectorReader), ErrRejected)
	other.no("a run under a role on a file path", other.run("r1", FirstData, Passed, CollectorReader), ErrInvalid)
	other.no("a run for a stage outside the path", other.run("r1", Authenticated, Passed, ""), ErrInvalid)
	other.no("a change naming a stage outside the path", other.change("hash-10", OperationalRead), ErrInvalid)
	other.prove("-1", FirstData, FullCoverage, FocusConformance, Reconciled)
	if blockers, err := other.bill(); err != nil {
		t.Errorf("a file-path project opening on four evidence stages: %v %s", err, blockers)
	}
	other.is("open without any role", "data=holding cover=holding focus=holding recon=holding bill=holding | 5/5 open")
}

func TestOutcomesByStage(t *testing.T) {
	stages := order[:6]
	want := map[Outcome]string{
		Passed: "auth read data cover focus recon", PassedNoUsage: "data cover", InsufficientPermission: "auth read data cover focus recon",
		AwaitingFirstExport: "data cover", PeriodUnsupported: "data cover recon", PartialFile: "data cover focus", UnsupportedScope: "auth read data", Failed: "auth read data cover focus recon",
	}
	onFile := map[Outcome]string{Passed: "data cover focus recon", PassedNoUsage: "data cover", InsufficientPermission: "", AwaitingFirstExport: "", PeriodUnsupported: "data cover recon",
		PartialFile: "data cover focus", UnsupportedScope: "data", Failed: "data cover focus recon"}
	for _, path := range []Path{ExistingAccount, ParentOperator, FileImport} {
		d := open(t, testLimits, path)
		role := Role("")
		table := onFile
		if path.connected() {
			d.ok("grant", d.grant(CollectorReader))
			role, table = CollectorReader, want
		}
		for outcome, expected := range table {
			accepted := []string{}
			for _, stage := range stages {
				given := Run{Stage: stage, Outcome: outcome, Role: role}
				if outcome == Failed {
					given.Code = "quota"
				}
				if outcome == UnsupportedScope {
					given.Alternative = NoAlternative
				}
				name := fmt.Sprintf("%s-%s", outcome, short[stage])
				if err := d.record(name, given); err == nil {
					accepted = append(accepted, short[stage])
					if reachErr := d.reach(stage, name); (reachErr == nil) != outcome.reaches() && !(outcome == PassedNoUsage && stage == FullCoverage) {
						t.Errorf("%s on %s reaching: %v", outcome, stage, reachErr)
					}
				} else if !errors.Is(err, ErrInvalid) {
					t.Errorf("%s on %s: %v", outcome, stage, err)
				}
			}
			if strings.Join(accepted, " ") != expected {
				t.Errorf("%s on %s is accepted for %q; want %q", outcome, path, strings.Join(accepted, " "), expected)
			}
		}
		d.no("a run that examines the opening", d.run("bill-run", BillingOpen, Passed, role), ErrInvalid)
		runs, err := d.r.Runs("w1", "p1")
		if err != nil || len(runs) == 0 || runs[0].Sequence == 0 || !runs[0].Until.Equal(runs[0].At.Add(day)) || runs[0].Generation != 1 || (path.connected() && runs[0].Grant != 1) || runs[0].Run.Detail == "" {
			t.Errorf("the stored runs: %+v %v", runs[:1], err)
		}
	}
	d := open(t, testLimits, ExistingAccount)
	d.ok("grant", d.grant(CollectorReader))
	base := Run{Stage: FirstData, Outcome: Passed, Role: CollectorReader}
	bad := map[string]func(*Run){
		"an unknown outcome": func(r *Run) { r.Outcome = "fine" }, "an unknown stage": func(r *Run) { r.Stage = "paid" }, "no role on a connected path": func(r *Run) { r.Role = "" },
		"an unknown role": func(r *Run) { r.Role = "auditor" }, "a negative validity": func(r *Run) { r.Validity = -time.Hour },
		"a detail that is text": func(r *Run) { r.Detail = "see the mail" }, "a code on a run that did not fail": func(r *Run) { r.Code = "quota" },
		"a failure without a code": func(r *Run) { r.Outcome = Failed }, "a failure with text": func(r *Run) { r.Outcome, r.Code = Failed, "it broke" },
		"an alternative on a passing run": func(r *Run) { r.Alternative = FileImport }, "an unsupported scope without an alternative": func(r *Run) { r.Outcome = UnsupportedScope },
		"the own path as the alternative": func(r *Run) { r.Outcome, r.Alternative = UnsupportedScope, ExistingAccount }, "an unknown alternative": func(r *Run) { r.Outcome, r.Alternative = UnsupportedScope, "reseller" },
		"a version that is text": func(r *Run) { r.Version = "version one" },
	}
	for name, change := range bad {
		given := base
		change(&given)
		d.no(name, d.record("bad", given), ErrInvalid)
	}
	long := base
	long.Validity = 48*time.Hour + time.Nanosecond
	d.no("a validity above the limit", d.record("bad", long), ErrLimit)
	foreign := base
	foreign.Version = "v2"
	d.no("a run by another version of the product", d.record("bad", foreign), ErrRejected)
	d.no("a run under a role that has no grant", d.run("bad", FirstData, Passed, ExportWriter), ErrRejected)
	for _, alternative := range []Path{ParentOperator, FileImport, NoAlternative} {
		given := Run{Stage: Authenticated, Outcome: UnsupportedScope, Role: CollectorReader, Alternative: alternative}
		d.ok("an unsupported scope pointing to "+string(alternative), d.record("alt-"+string(alternative), given))
	}
	limit := base
	limit.Validity = 48 * time.Hour
	d.ok("a validity exactly at the limit", d.record("edge", limit))
	if d.sequence() != 5 {
		t.Errorf("a refused run was written: sequence %d", d.sequence())
	}
}

func TestStagesAreSeparateAndRestOnTheCollector(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	d.ok("the setup grant", d.grant(Setup))
	d.ok("the writer's grant", d.grant(ExportWriter))
	d.ok("the collector's grant", d.grant(CollectorReader))
	d.ok("a run under setup", d.run("by-setup", OperationalRead, Passed, Setup))
	d.ok("a run under the writer", d.run("by-writer", OperationalRead, Passed, ExportWriter))
	d.no("evidence from the setup access", d.reach(OperationalRead, "by-setup"), ErrRejected)
	d.no("evidence from the writer's access", d.reach(OperationalRead, "by-writer"), ErrRejected)
	d.prove("-1", Reconciled)
	d.is("a later stage alone is not progress", "auth=missing read=missing/alone data=missing/alone cover=missing/alone focus=missing/alone recon=holding/alone bill=missing/alone | 0/7 closed")
	d.prove("-1", Authenticated, FirstData)
	d.is("progress stops at the first gap, and a stage reached for the first time invalidates what was shown after it", "auth=holding read=missing data=holding/alone cover=missing/alone focus=missing/alone recon=stale/upstream_renewed/alone bill=missing/alone | 1/7 closed")
	d.ok("a run of another stage", d.run("wrong", FullCoverage, Passed, CollectorReader))
	d.no("a run that examined another stage", d.reach(OperationalRead, "wrong"), ErrRejected)
	d.no("an unknown run", d.reach(OperationalRead, "nothing"), ErrNotFound)
	d.no("the opening reached like a stage", d.reach(BillingOpen, "wrong"), ErrInvalid)
	d.no("a stage that does not exist", d.reach("paid", "wrong"), ErrInvalid)
	d.ok("a failing run", d.run("denied", OperationalRead, InsufficientPermission, CollectorReader))
	d.no("a run that did not pass", d.reach(OperationalRead, "denied"), ErrRejected)
	d.prove("-0", order[:6]...)
	d.is("every stage shown while setup is active", allHold+" bill=missing | 6/7 closed")
	blockers, err := d.bill()
	if !errors.Is(err, ErrRejected) || blockers != "billing_open:setup_active operational_read:operational_read_under_setup" {
		t.Errorf("opening while setup is active: %q %v", blockers, err)
	}
	d.ok("setup withdrawn", d.withdraw(Setup))
	d.is("withdrawing setup changes no stage", allHold+" bill=missing | 6/7 closed")
	view, _ := d.r.Readiness("w1", "p1")
	active := map[Role]bool{}
	for _, role := range view.Roles {
		active[role.Role] = role.Active
		if role.Role == Setup && (role.Condition != AfterSetupComplete || !role.Expiry.Equal(start.Add(2*time.Hour)) || role.Holder != "holder-setup" || role.Grant != 1) {
			t.Errorf("the setup grant: %+v", role)
		}
	}
	if active[Setup] || !active[ExportWriter] || !active[CollectorReader] {
		t.Errorf("withdrawing setup touched another role: %v", active)
	}
	blockers, err = d.bill()
	if !errors.Is(err, ErrRejected) || blockers != "operational_read:operational_read_under_setup" {
		t.Errorf("opening on a read shown while setup was active: %q %v", blockers, err)
	}
	d.prove("-2", OperationalRead)
	d.is("a read shown again without setup is a renewal", allHold+" bill=missing | 6/7 closed")
	if blockers, err := d.bill(); err != nil {
		t.Errorf("opening after the read was shown without setup: %q %v", blockers, err)
	}
	d.is("open", allHold+" bill=holding | 7/7 open")
	blockers, err = d.bill()
	if !errors.Is(err, ErrRejected) || blockers != "" {
		t.Errorf("opening twice: %q %v", blockers, err)
	}
	d.ok("the collector withdrawn", d.withdraw(CollectorReader))
	d.is("every stage rests on the collector's grant", "auth=stale/role_ended read=stale/role_ended/alone data=stale/role_ended/alone cover=stale/role_ended/alone focus=stale/role_ended/alone recon=stale/role_ended/alone bill=stale/evidence_lapsed/alone | 0/7 closed")
	d.ok("the collector granted again", d.grant(CollectorReader))
	d.is("a new grant revives nothing", "auth=stale/role_ended read=stale/role_ended/alone data=stale/role_ended/alone cover=stale/role_ended/alone focus=stale/role_ended/alone recon=stale/role_ended/alone bill=stale/evidence_lapsed/alone | 0/7 closed")
	d.no("an old run under the ended grant", d.reach(OperationalRead, "read-2"), ErrRejected)
	d.no("a grant while one is active", d.grant(CollectorReader), ErrRejected)
	d.no("a withdrawal of a role that is not active", d.withdraw(Setup), ErrRejected)
	d.no("an unknown role withdrawn", d.withdraw("auditor"), ErrInvalid)
	d.prove("-3", order[:6]...)
	d.is("shown again under the new grant, billing stays closed", allHold+" bill=stale/evidence_lapsed | 6/7 closed")
	if blockers, err := d.bill(); err != nil {
		t.Errorf("opening again: %q %v", blockers, err)
	}
	d.is("open again only by an opening", allHold+" bill=holding | 7/7 open")
}

func TestSetupGrants(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	seq := func() uint64 { return d.sequence() }
	for name, attempt := range map[string]func() (Result, error){
		"setup without a duration":  func() (Result, error) { return d.r.GrantRole("w1", "p1", seq(), "g", Setup, "h", 0, OnExpiry) },
		"setup without a condition": func() (Result, error) { return d.r.GrantRole("w1", "p1", seq(), "g", Setup, "h", time.Hour, "") },
		"setup with its own condition": func() (Result, error) {
			return d.r.GrantRole("w1", "p1", seq(), "g", Setup, "h", time.Hour, "when done")
		},
		"a reader with a duration": func() (Result, error) {
			return d.r.GrantRole("w1", "p1", seq(), "g", CollectorReader, "h", time.Hour, "")
		},
		"a writer with a condition": func() (Result, error) { return d.r.GrantRole("w1", "p1", seq(), "g", ExportWriter, "h", 0, OnRequest) },
		"an unknown role":           func() (Result, error) { return d.r.GrantRole("w1", "p1", seq(), "g", "auditor", "h", 0, "") },
		"a holder that is an address": func() (Result, error) {
			return d.r.GrantRole("w1", "p1", seq(), "g", ExportWriter, "ops@example.invalid", 0, "")
		},
		"a negative duration": func() (Result, error) { return d.r.GrantRole("w1", "p1", seq(), "g", Setup, "h", -time.Hour, OnExpiry) },
		"a reader with a negative duration": func() (Result, error) {
			return d.r.GrantRole("w1", "p1", seq(), "g", CollectorReader, "h", -time.Hour, "")
		},
		"a writer with a condition of its own": func() (Result, error) {
			return d.r.GrantRole("w1", "p1", seq(), "g", ExportWriter, "h", 0, "when done")
		},
	} {
		d.no(name, second(attempt()), ErrInvalid)
	}
	d.no("a setup grant longer than the limit", second(d.r.GrantRole("w1", "p1", 0, "g", Setup, "h", 8*time.Hour+time.Nanosecond, OnExpiry)), ErrLimit)
	for _, condition := range []Condition{AfterSetupComplete, OnExpiry, OnRequest} {
		if _, err := d.r.GrantRole("w1", "p1", d.sequence(), d.next(), Setup, "h", 8*time.Hour, condition); err != nil {
			t.Errorf("a setup grant with the condition %s: %v", condition, err)
		}
		d.ok("its withdrawal", d.withdraw(Setup))
	}
	d.ok("the collector", d.grant(CollectorReader))
	d.ok("setup", d.grant(Setup))
	d.ok("a read during setup", d.run("during", OperationalRead, Passed, CollectorReader))
	d.pass(2*time.Hour - time.Nanosecond)
	d.no("a grant an instant before the expiry", d.grant(Setup), ErrRejected)
	d.pass(time.Nanosecond)
	d.ok("a read at the instant setup expired", d.run("after", OperationalRead, Passed, CollectorReader))
	runs, _ := d.r.Runs("w1", "p1")
	if !runs[0].UnderSetup || runs[1].UnderSetup {
		t.Errorf("the marks of the two reads: %v %v", runs[0].UnderSetup, runs[1].UnderSetup)
	}
	d.prove("-1", Authenticated)
	d.ok("a new setup grant once the old one expired", d.grant(Setup))
	d.ok("a read under it", d.run("again", OperationalRead, Passed, CollectorReader))
	d.ok("its withdrawal at the same reading of the clock", d.withdraw(Setup))
	runs, _ = d.r.Runs("w1", "p1")
	if last := runs[len(runs)-1]; last.ID != "again" || !last.UnderSetup {
		t.Errorf("a run and the withdrawal of setup at the same time: the run is not marked as recorded under setup")
	}
	d.ok("reached", d.reach(OperationalRead, "again"))
	d.prove("-1", FirstData, FullCoverage, FocusConformance, Reconciled)
	if blockers, err := d.bill(); !errors.Is(err, ErrRejected) || blockers != "operational_read:operational_read_under_setup" {
		t.Errorf("opening on a read marked as recorded under setup: %q %v", blockers, err)
	}
}

func TestEvidenceExpiresAndIsRenewed(t *testing.T) {
	d := ready(t)
	if blockers, err := d.bill(); err != nil {
		t.Fatalf("opening: %q %v", blockers, err)
	}
	d.pass(day - time.Nanosecond)
	d.is("an instant before the evidence expires", allHold+" bill=holding | 7/7 open")
	d.ok("a renewal just in time", d.run("auth-2", Authenticated, Passed, CollectorReader))
	d.ok("reached", d.reach(Authenticated, "auth-2"))
	d.is("a renewal changes nothing else", allHold+" bill=holding | 7/7 open")
	d.pass(time.Nanosecond)
	d.is("at the expiry", "auth=holding read=stale/expired data=stale/expired/alone cover=stale/expired/alone focus=stale/expired/alone recon=stale/expired/alone bill=stale/evidence_lapsed/alone | 1/7 closed")
	sequence := d.sequence()
	blockers, err := d.bill()
	if !errors.Is(err, ErrRejected) || blockers != "operational_read:expired first_data:expired full_coverage:expired focus_conformance:expired reconciled:expired" {
		t.Errorf("opening on expired evidence: %q %v", blockers, err)
	}
	if d.sequence() != sequence {
		t.Errorf("a refused opening wrote")
	}
	d.pass(-3 * day)
	d.is("a clock that goes back between two reads revives nothing", "auth=holding read=stale/expired data=stale/expired/alone cover=stale/expired/alone focus=stale/expired/alone recon=stale/expired/alone bill=stale/evidence_lapsed/alone | 1/7 closed")
	d.no("a run of the old evidence reached again", d.reach(OperationalRead, "read-1"), ErrRejected)
	d.ok("a run recorded while the clock is behind", d.run("read-2", OperationalRead, Passed, CollectorReader))
	if runs, _ := d.r.Runs("w1", "p1"); !runs[len(runs)-1].At.Equal(start.Add(day)) {
		t.Errorf("a write after the clock went back is dated %s", runs[len(runs)-1].At)
	}
	d.pass(3 * day)
	d.ok("the lapsed stage reached again", d.reach(OperationalRead, "read-2"))
	d.is("reaching a lapsed stage invalidates what came after it", "auth=holding read=holding data=stale/upstream_renewed cover=stale/upstream_renewed/alone focus=stale/upstream_renewed/alone recon=stale/upstream_renewed/alone bill=stale/evidence_lapsed/alone | 2/7 closed")
	d.prove("-2", FirstData, FullCoverage, FocusConformance, Reconciled)
	d.is("billing does not come back by itself", allHold+" bill=stale/evidence_lapsed | 6/7 closed")
	if blockers, err := d.bill(); err != nil {
		t.Errorf("opening again: %q %v", blockers, err)
	}
	d.is("open again", allHold+" bill=holding | 7/7 open")

	e := ready(t)
	e.ok("a spare run for a later stage", e.run("spare-recon", Reconciled, Passed, CollectorReader))
	e.ok("a spare run for the stage itself", e.run("spare-data", FirstData, Passed, CollectorReader))
	e.pass(day)
	e.prove("-2", Authenticated, OperationalRead)
	e.no("an expired spare run", e.reach(FirstData, "spare-data"), ErrRejected)
	f := ready(t)
	f.ok("the collector's first data again", f.run("data-2", FirstData, Passed, CollectorReader))
	f.ok("a spare reconciliation", f.run("spare-recon", Reconciled, Passed, CollectorReader))
	f.ok("a configuration change from first data", f.change("hash-2", FirstData))
	f.no("a run from before the change", f.reach(FirstData, "data-2"), ErrRejected)
	f.ok("the reconciliation shown under the new configuration", f.run("recon-2", Reconciled, Passed, CollectorReader))
	f.ok("reached although first data is stale", f.reach(Reconciled, "recon-2"))
	f.is("a later stage shown while an earlier one is stale", "auth=holding read=holding data=stale/config_changed cover=stale/config_changed/alone focus=stale/config_changed/alone recon=holding/alone bill=missing/alone | 2/7 closed")
	f.ok("a spare reconciliation under the new configuration", f.run("spare-2", Reconciled, Passed, CollectorReader))
	f.prove("-3", FirstData)
	f.is("first data reached again invalidates the reconciliation that was shown", "auth=holding read=holding data=holding cover=stale/config_changed focus=stale/config_changed/alone recon=stale/upstream_renewed/alone bill=missing/alone | 3/7 closed")
	f.no("the spare run recorded before first data was replaced", f.reach(Reconciled, "spare-2"), ErrRejected)
	g := open(t, testLimits, ExistingAccount)
	g.ok("the collector", g.grant(CollectorReader))
	g.prove("-1", Authenticated, OperationalRead, FirstData)
	g.ok("a spare coverage for a stage never reached", g.run("spare-cover", FullCoverage, Passed, CollectorReader))
	g.pass(day)
	g.prove("-2", Authenticated, OperationalRead)
	g.ok("a fresh coverage", g.run("cover-2", FullCoverage, Passed, CollectorReader))
	g.pass(-day)
	g.prove("-2", FirstData)
	g.no("a coverage recorded before first data was replaced, for a stage never reached", g.reach(FullCoverage, "cover-2"), ErrRejected)
	g.ok("a coverage recorded afterwards", g.run("cover-3", FullCoverage, Passed, CollectorReader))
	g.ok("reached", g.reach(FullCoverage, "cover-3"))
	g.no("an older run than the current one", g.reach(FullCoverage, "cover-3"), ErrRejected)
}

func TestEveryOperationMovesTheRegistrysTime(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	d.ok("the collector", d.grant(CollectorReader))
	d.pass(5 * time.Hour)
	if created, err := d.r.Create("w1", "p2", "source-2", FileImport, "hash-1"); !created || err != nil {
		t.Fatalf("a creation: %v %v", created, err)
	}
	d.pass(-5 * time.Hour)
	d.ok("a run after a creation that read a later clock", d.run("auth-1", Authenticated, Passed, CollectorReader))
	if runs, _ := d.r.Runs("w1", "p1"); !runs[0].At.Equal(start.Add(5 * time.Hour)) {
		t.Errorf("the run is dated %s although a creation had read the clock five hours later", runs[0].At)
	}
}

func TestRefusalsAndSmallRules(t *testing.T) {
	d := ready(t)
	sequence := d.sequence()
	for name, c := range map[string]struct {
		err  error
		want error
	}{
		"a stale expected sequence before an unknown role": {second(d.r.GrantRole("w1", "p1", 0, "g1", "auditor", "h", 0, "")), ErrConflict},
		"a stored identifier before a malformed run":       {second(d.r.RecordRun("w1", "p1", sequence, "auth-1", Run{Stage: Authenticated, Validity: -1})), ErrConflict},
		"a malformed identifier before everything":         {second(d.r.RecordRun("w1", "p1", 0, "a b", Run{})), ErrInvalid},
		"a stale expected sequence on a grant":             {second(d.r.GrantRole("w1", "p1", sequence-1, "g2", ExportWriter, "h", 0, "")), ErrConflict},
		"a stale expected sequence on an opening":          {second(d.bill2(sequence - 1)), ErrConflict},
		"a run name that is text":                          {second(d.r.Reach("w1", "p1", sequence, "x1", Authenticated, "auth 1")), ErrInvalid},
		"a withdrawal of setup that was never granted":     {d.withdraw(Setup), ErrRejected},
	} {
		d.no(name, c.err, c.want)
	}
	d.ok("an older spare run", d.run("auth-old", Authenticated, Passed, CollectorReader))
	d.ok("a newer run", d.run("auth-new", Authenticated, Passed, CollectorReader))
	d.ok("the newer one reached", d.reach(Authenticated, "auth-new"))
	d.no("a run recorded before the current evidence", d.reach(Authenticated, "auth-old"), ErrRejected)
	d.ok("setup", d.grant(Setup))
	d.pass(2 * time.Hour)
	d.no("a withdrawal of a setup grant that expired", d.withdraw(Setup), ErrRejected)
	d.ok("a change", d.change("hash-2", BillingOpen))
	if created, err := d.r.Create("w1", "p1", "source-1", ExistingAccount, "hash-1"); created || err != nil {
		t.Errorf("repeating a creation after the configuration changed: %v %v", created, err)
	}
	d.no("repeating it with the present hash", second(d.r.Create("w1", "p1", "source-1", ExistingAccount, "hash-2")), ErrConflict)
	before, _ := d.r.Readiness("w1", "p1")
	d.pass(time.Hour)
	d.no("a refused write", d.reach(Authenticated, "nothing"), ErrNotFound)
	d.pass(-time.Hour)
	if after, _ := d.r.Readiness("w1", "p1"); !after.AsOf.Equal(before.AsOf.Add(time.Hour)) {
		t.Errorf("a refused write did not move the registry's time: %s", after.AsOf)
	}
	d.pass(3 * time.Hour)
	d.r.Runs("w1", "p1")
	d.pass(-3 * time.Hour)
	if after, _ := d.r.Readiness("w1", "p1"); !after.AsOf.Equal(before.AsOf.Add(3 * time.Hour)) {
		t.Errorf("a read of the runs did not move the registry's time: %s", after.AsOf)
	}
	file := open(t, testLimits, FileImport)
	for _, role := range []Role{Setup, ExportWriter, CollectorReader} {
		file.no("granting "+string(role)+" on a file path", file.grant(role), ErrRejected)
		file.no("withdrawing "+string(role)+" on a file path", file.withdraw(role), ErrRejected)
	}
	last := open(t, Limits{Projects: 1, Writes: 14, Reserve: 1, MaxValidity: 48 * time.Hour, MaxSetup: 8 * time.Hour}, ExistingAccount)
	last.ok("the collector", last.grant(CollectorReader))
	last.prove("-1", order[:6]...)
	last.ok("the opening", second(last.bill()))
	last.ok("a change that is also the last write of the reserve", last.change("hash-2", BillingOpen))
	last.is("the first reason is kept when the reserve is used up by a change", allHold+" bill=stale/config_changed | 6/7 closed")
}

func TestEmptyData(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	d.ok("the collector", d.grant(CollectorReader))
	d.prove("-1", Authenticated, OperationalRead)
	d.ok("coverage without usage", d.run("cover-empty", FullCoverage, PassedNoUsage, CollectorReader))
	d.no("empty coverage while first data is missing", d.reach(FullCoverage, "cover-empty"), ErrRejected)
	d.ok("first data without usage", d.run("data-empty", FirstData, PassedNoUsage, CollectorReader))
	d.ok("reached", d.reach(FirstData, "data-empty"))
	d.ok("coverage with usage", d.run("cover-full", FullCoverage, Passed, CollectorReader))
	d.no("coverage with usage on empty first data", d.reach(FullCoverage, "cover-full"), ErrRejected)
	d.ok("empty coverage again", d.run("cover-empty-2", FullCoverage, PassedNoUsage, CollectorReader))
	d.ok("empty coverage on empty first data", d.reach(FullCoverage, "cover-empty-2"))
	d.prove("-1", FocusConformance, Reconciled)
	d.is("normal no-usage is shown and is not a failure", "auth=holding read=holding data=holding/empty cover=holding/empty focus=holding recon=holding bill=missing | 6/7 closed")
	blockers, err := d.bill()
	if !errors.Is(err, ErrRejected) || blockers != "first_data:empty full_coverage:empty" {
		t.Errorf("opening on empty data: %q %v", blockers, err)
	}
	d.ok("first data with usage", d.run("data-full", FirstData, Passed, CollectorReader))
	d.ok("reached", d.reach(FirstData, "data-full"))
	d.is("a change of emptiness is not a renewal", "auth=holding read=holding data=holding cover=stale/upstream_renewed/empty focus=stale/upstream_renewed/alone recon=stale/upstream_renewed/alone bill=missing/alone | 3/7 closed")
	d.ok("empty coverage again", d.run("cover-empty-3", FullCoverage, PassedNoUsage, CollectorReader))
	d.no("empty coverage on first data with usage", d.reach(FullCoverage, "cover-empty-3"), ErrRejected)
	d.prove("-2", FullCoverage, FocusConformance, Reconciled)
	if blockers, err := d.bill(); err != nil {
		t.Errorf("opening once the data is there: %q %v", blockers, err)
	}
	d.ok("first data without usage again", d.run("data-empty-2", FirstData, PassedNoUsage, CollectorReader))
	d.ok("reached", d.reach(FirstData, "data-empty-2"))
	d.is("billing that was open closes when the data turns empty", "auth=holding read=holding data=holding/empty cover=stale/upstream_renewed focus=stale/upstream_renewed/alone recon=stale/upstream_renewed/alone bill=stale/upstream_renewed/alone | 3/7 closed")
}

func TestConfigurationChanges(t *testing.T) {
	for index, from := range order {
		d := ready(t)
		if blockers, err := d.bill(); err != nil {
			t.Fatalf("opening: %q %v", blockers, err)
		}
		d.ok("a change", d.change("hash-2", from))
		parts := []string{}
		for place, stage := range order[:6] {
			text := short[stage] + "=holding"
			if place >= index {
				text = short[stage] + "=stale/config_changed"
				if place > index {
					text += "/alone"
				}
			}
			parts = append(parts, text)
		}
		bill := "bill=stale/config_changed"
		if index < 6 {
			bill += "/alone"
		}
		d.is("a change from "+string(from), fmt.Sprintf("%s %s | %d/7 closed", strings.Join(parts, " "), bill, index))
		if view, _ := d.r.Readiness("w1", "p1"); view.Generation != 2 || view.Hash != "hash-2" {
			t.Errorf("the generation after a change: %+v", view)
		}
		if index == 6 {
			if blockers, err := d.bill(); err != nil {
				t.Errorf("a change that names only the opening lets billing open again at once (a stated limit): %q %v", blockers, err)
			}
		}
	}
	d := ready(t)
	d.ok("a spare run", d.run("spare", FullCoverage, Passed, CollectorReader))
	d.ok("a change from coverage", d.change("hash-2", FullCoverage))
	d.ok("a return to the first hash", d.change("hash-1", BillingOpen))
	d.no("a run of the first generation after the return to its hash", d.reach(FullCoverage, "spare"), ErrRejected)
	d.no("a run of an earlier generation for a stage that still holds", d.reach(Authenticated, "auth-1"), ErrRejected)
	d.ok("a change with the same hash", d.change("hash-1", Reconciled))
	if view, _ := d.r.Readiness("w1", "p1"); view.Generation != 4 {
		t.Errorf("a change with the same hash did not start a generation: %d", view.Generation)
	}
	d.is("earlier stages keep holding on the evidence of an older generation", "auth=holding read=holding data=holding cover=stale/config_changed focus=stale/config_changed/alone recon=stale/config_changed/alone bill=missing/alone | 3/7 closed")
	d.ok("a renewal under the current generation", d.run("auth-4", Authenticated, Passed, CollectorReader))
	d.ok("reached", d.reach(Authenticated, "auth-4"))
	d.no("a stage that does not exist", d.change("hash-3", "paid"), ErrInvalid)
	d.no("a hash that is text", d.change("hash 3", Authenticated), ErrInvalid)
}

func TestTheOpeningDoesNotSurviveAMomentWithoutItsConditions(t *testing.T) {
	d := ready(t)
	d.ok("the opening", second(d.bill()))
	d.ok("setup granted after the opening", d.grant(Setup))
	d.is("a setup grant closes billing", allHold+" bill=stale/setup_granted | 6/7 closed")
	d.pass(2 * time.Hour)
	d.is("its expiry does not bring billing back", allHold+" bill=stale/setup_granted | 6/7 closed")
	if blockers, err := d.bill(); err != nil {
		t.Errorf("opening again once setup is gone: %q %v", blockers, err)
	}
	d.ok("setup granted again", d.grant(Setup))
	d.ok("withdrawn", d.withdraw(Setup))
	d.is("its withdrawal does not bring billing back", allHold+" bill=stale/setup_granted | 6/7 closed")
	d.ok("the opening again", second(d.bill()))
	d.is("open", allHold+" bill=holding | 7/7 open")

	e := ready(t)
	e.ok("setup", e.grant(Setup))
	e.ok("a read recorded under setup, kept aside", e.run("marked", OperationalRead, Passed, CollectorReader))
	e.ok("setup withdrawn", e.withdraw(Setup))
	if blockers, err := e.bill(); err != nil {
		t.Fatalf("opening on the unmarked read: %q %v", blockers, err)
	}
	e.ok("the read replaced by the one recorded under setup", e.reach(OperationalRead, "marked"))
	e.is("billing closes and the opening is spent", allHold+" bill=stale/evidence_lapsed | 6/7 closed")
	e.prove("-2", OperationalRead)
	e.is("replacing it again by a read recorded without setup does not reopen billing", allHold+" bill=stale/evidence_lapsed | 6/7 closed")
	if blockers, err := e.bill(); err != nil {
		t.Errorf("opening again: %q %v", blockers, err)
	}

	f := ready(t)
	f.ok("the opening", second(f.bill()))
	f.pass(day)
	f.is("lapsed by the clock, not yet met by a write", "auth=stale/expired read=stale/expired/alone data=stale/expired/alone cover=stale/expired/alone focus=stale/expired/alone recon=stale/expired/alone bill=stale/evidence_lapsed/alone | 0/7 closed")
	f.ok("a change that names only the opening", f.change("hash-2", BillingOpen))
	f.is("the first reason of an invalidation is kept", "auth=stale/expired read=stale/expired/alone data=stale/expired/alone cover=stale/expired/alone focus=stale/expired/alone recon=stale/expired/alone bill=stale/evidence_lapsed/alone | 0/7 closed")
}

func TestWritesRepeatsAndLimits(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	first, err := d.r.GrantRole("w1", "p1", 0, "w-grant", CollectorReader, "holder", 0, "")
	if err != nil || first != (Result{Sequence: 1}) {
		t.Fatalf("%+v %v", first, err)
	}
	again, err := d.r.GrantRole("w1", "p1", 99, "w-grant", CollectorReader, "holder", 0, "")
	if err != nil || again != (Result{Sequence: 1, Replay: true}) {
		t.Errorf("the same grant again, with another expected sequence: %+v %v", again, err)
	}
	d.no("the same identifier with another holder", second(d.r.GrantRole("w1", "p1", 1, "w-grant", CollectorReader, "other", 0, "")), ErrConflict)
	given := Run{Stage: Authenticated, Outcome: Passed, Role: CollectorReader, Validity: day, Version: "v1", Detail: "d"}
	if _, err := d.r.RecordRun("w1", "p1", 1, "w-run", given); err != nil {
		t.Fatal(err)
	}
	if repeat, err := d.r.RecordRun("w1", "p1", 1, "w-run", given); err != nil || repeat != (Result{Sequence: 2, Replay: true}) {
		t.Errorf("the same run again: %+v %v", repeat, err)
	}
	other := given
	other.Detail = "e"
	d.no("the same identifier with another detail", second(d.r.RecordRun("w1", "p1", 2, "w-run", other)), ErrConflict)
	d.no("the identifier of a run used for a grant", second(d.r.GrantRole("w1", "p1", 2, "w-run", ExportWriter, "h", 0, "")), ErrConflict)
	d.no("the identifier of a grant used for a withdrawal", second(d.r.WithdrawRole("w1", "p1", "w-grant", CollectorReader)), ErrConflict)
	d.no("a stale expected sequence", second(d.r.RecordRun("w1", "p1", 1, "w-run-2", given)), ErrConflict)
	d.no("an expected sequence ahead", second(d.r.Reach("w1", "p1", 9, "w-reach", Authenticated, "w-run")), ErrConflict)
	if _, err := d.r.Reach("w1", "p1", 2, "w-reach", Authenticated, "w-run"); err != nil {
		t.Fatal(err)
	}
	if repeat, err := d.r.Reach("w1", "p1", 0, "w-reach", Authenticated, "w-run"); err != nil || !repeat.Replay || repeat.Sequence != 3 {
		t.Errorf("the same reach again: %+v %v", repeat, err)
	}
	d.no("the same identifier reaching another stage", second(d.r.Reach("w1", "p1", 3, "w-reach", OperationalRead, "w-run")), ErrConflict)
	if _, err := d.r.ChangeConfig("w1", "p1", "w-change", "hash-2", BillingOpen); err != nil {
		t.Fatal(err)
	}
	if repeat, err := d.r.ChangeConfig("w1", "p1", "w-change", "hash-2", BillingOpen); err != nil || !repeat.Replay || d.sequence() != 4 {
		t.Errorf("the same change again: %+v %v", repeat, err)
	}
	d.no("the same identifier with another first stage", second(d.r.ChangeConfig("w1", "p1", "w-change", "hash-2", Authenticated)), ErrConflict)
	if _, err := d.r.WithdrawRole("w1", "p1", "w-withdraw", CollectorReader); err != nil {
		t.Fatal(err)
	}
	if repeat, err := d.r.WithdrawRole("w1", "p1", "w-withdraw", CollectorReader); err != nil || !repeat.Replay || repeat.Sequence != 5 {
		t.Errorf("the same withdrawal again, although the role is no longer active: %+v %v", repeat, err)
	}
	for name, err := range map[string]error{
		"a write identifier that is text": second(d.r.WithdrawRole("w1", "p1", "w x", CollectorReader)),
		"no workspace":                    second(d.r.ChangeConfig("", "p1", "w", "h", BillingOpen)),
		"a project that is text":          second(d.r.Reach("w1", "p 1", 0, "w", Authenticated, "r")),
	} {
		d.no(name, err, ErrInvalid)
	}
	d.no("a write for an unknown project", second(d.r.WithdrawRole("w1", "p9", "w", CollectorReader)), ErrNotFound)
	d.no("a write for a project of another workspace", second(d.r.ChangeConfig("w2", "p1", "w", "h", BillingOpen)), ErrNotFound)

	o := ready(t)
	if _, blockers, err := o.r.OpenBilling("w1", "p1", o.sequence(), "w-open"); err != nil {
		t.Fatalf("%v %v", blockers, err)
	}
	o.pass(day)
	if repeat, _, err := o.r.OpenBilling("w1", "p1", 0, "w-open"); err != nil || !repeat.Replay {
		t.Errorf("the same opening again after it lapsed: %+v %v", repeat, err)
	}
	o.is("a repeated opening reopens nothing", "auth=stale/expired read=stale/expired/alone data=stale/expired/alone cover=stale/expired/alone focus=stale/expired/alone recon=stale/expired/alone bill=stale/evidence_lapsed/alone | 0/7 closed")

	l := open(t, Limits{Projects: 1, Writes: 14, Reserve: 2, MaxValidity: 48 * time.Hour, MaxSetup: 8 * time.Hour}, ExistingAccount)
	l.ok("the collector", l.grant(CollectorReader))
	l.prove("-1", order[:6]...)
	if blockers, err := l.bill(); err != nil {
		t.Fatalf("opening as the last ordinary write: %q %v", blockers, err)
	}
	if l.sequence() != 14 {
		t.Fatalf("sequence %d", l.sequence())
	}
	l.is("open at the write limit", allHold+" bill=holding | 7/7 open")
	l.no("a run at the limit", l.run("more", Authenticated, Passed, CollectorReader), ErrLimit)
	l.no("a reach at the limit", l.reach(Authenticated, "auth-1"), ErrLimit)
	l.no("a grant at the limit", l.grant(ExportWriter), ErrLimit)
	l.no("an opening at the limit", second(l.bill()), ErrLimit)
	l.no("a withdrawal of a role that is not active, at the limit", l.withdraw(Setup), ErrRejected)
	l.ok("a change from the reserve", second(l.r.ChangeConfig("w1", "p1", "w-reserve", "hash-2", BillingOpen)))
	l.is("closed by the change", allHold+" bill=stale/config_changed | 6/7 closed")
	l.ok("a withdrawal as the last write of the reserve", l.withdraw(CollectorReader))
	l.no("a change when the reserve is used up", l.change("hash-3", Authenticated), ErrLimit)
	if repeat, err := l.r.ChangeConfig("w1", "p1", "w-reserve", "hash-2", BillingOpen); err != nil || !repeat.Replay {
		t.Errorf("a stored write is still answered when the reserve is used up: %+v %v", repeat, err)
	}
	m := open(t, Limits{Projects: 1, Writes: 15, Reserve: 1, MaxValidity: 48 * time.Hour, MaxSetup: 8 * time.Hour}, ExistingAccount)
	m.ok("the writer", m.grant(ExportWriter))
	m.ok("the collector", m.grant(CollectorReader))
	m.prove("-1", order[:6]...)
	m.ok("the opening", second(m.bill()))
	m.is("open with one write of reserve left", allHold+" bill=holding | 7/7 open")
	m.ok("an unrelated withdrawal that uses up the reserve", m.withdraw(ExportWriter))
	m.is("a project that can no longer record a change does not keep billing open", allHold+" bill=stale/capacity_exhausted | 6/7 closed")
	view, _ := m.r.Readiness("w1", "p1")
	if len(view.Blockers) != 1 || view.Blockers[0] != (Blocker{Stage: BillingOpen, Reason: CapacityExhausted}) {
		t.Errorf("the blockers once the reserve is used up: %+v", view.Blockers)
	}
}

func TestOneWriteForOneExpectedSequence(t *testing.T) {
	d := open(t, testLimits, ExistingAccount)
	d.ok("the collector", d.grant(CollectorReader))
	var group sync.WaitGroup
	results := make(chan error, 24)
	for index := 0; index < 24; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			given := Run{Stage: Authenticated, Outcome: Passed, Role: CollectorReader, Validity: day, Version: "v1", Detail: "d"}
			_, err := d.r.RecordRun("w1", "p1", 1, fmt.Sprintf("run-%d", index), given)
			results <- err
			d.r.Readiness("w1", "p1")
		}()
	}
	group.Wait()
	close(results)
	written := 0
	for err := range results {
		if err == nil {
			written++
		} else if !errors.Is(err, ErrConflict) {
			t.Errorf("a losing write ended with %v", err)
		}
	}
	if runs, _ := d.r.Runs("w1", "p1"); written != 1 || len(runs) != 1 || d.sequence() != 2 {
		t.Errorf("%d writes for one expected sequence", written)
	}
}

func TestARegistryThatIsNotBuilt(t *testing.T) {
	for name, limits := range map[string]Limits{"projects": {0, 1, 1, 1, 1}, "writes": {1, 0, 1, 1, 1}, "reserve": {1, 1, 0, 1, 1}, "validity": {1, 1, 1, 0, 1}, "setup": {1, 1, 1, 1, 0}} {
		if _, err := NewRegistry(limits, "v1", time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("limits without %s: %v", name, err)
		}
	}
	if _, err := NewRegistry(testLimits, "version one", time.Now); !errors.Is(err, ErrInvalid) {
		t.Errorf("a version that is text: %v", err)
	}
	if _, err := NewRegistry(testLimits, "v1", nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("no clock: %v", err)
	}
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		for name, err := range map[string]error{
			"create": second(r.Create("w1", "p1", "s", FileImport, "h")), "run": second(r.RecordRun("w1", "p1", 0, "w", Run{})), "reach": second(r.Reach("w1", "p1", 0, "w", FirstData, "r")),
			"change": second(r.ChangeConfig("w1", "p1", "w", "h", FirstData)), "grant": second(r.GrantRole("w1", "p1", 0, "w", ExportWriter, "h", 0, "")),
			"withdraw": second(r.WithdrawRole("w1", "p1", "w", ExportWriter)), "readiness": second(r.Readiness("w1", "p1")), "runs": second(r.Runs("w1", "p1")),
		} {
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("%s on a registry that is not built: %v", name, err)
			}
		}
		if _, _, err := r.OpenBilling("w1", "p1", 0, "w"); !errors.Is(err, ErrInvalid) {
			t.Errorf("open on a registry that is not built: %v", err)
		}
	}
	for id, valid := range map[string]bool{"a": true, "A.b_c:d-9": true, strings.Repeat("x", 128): true, strings.Repeat("x", 129): false, "": false, "a b": false, "a/b": false, "a@b": false} {
		if validID(id) != valid {
			t.Errorf("identifier %q", id)
		}
	}
}
