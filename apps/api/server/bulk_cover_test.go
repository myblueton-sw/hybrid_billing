package server

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestBulkGrantFaults(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Seed = SeedNone; c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	f.s.fault = func(at string) bool { return at == "bulk-grant" }
	if got := f.auth("redeem", `{"token":"`+f.links()[""]+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, f.pre()); got.status != 204 {
		t.Fatal(got.raw)
	}
	if !strings.Contains(logs.String(), "bulk revocation grant failed at activation") {
		t.Error("a failed bulk grant is not logged")
	}
	g := setup(t, SeedSample)
	g.s.fault = func(at string) bool { return at == "bulk-grant" }
	if err := g.s.Start(); !errors.Is(err, errPartial) {
		t.Errorf("a failed sample bulk grant %v", err)
	}
	h := setupWith(t, func(c *Config) { c.Seed = SeedNone })
	if got := h.auth("redeem", `{"token":"`+h.links()[""]+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, h.pre()); got.status != 204 {
		t.Fatal(got.raw)
	}
	if !h.s.holdsRight(EmptyAdmin, gate.RevokeAccessInBulk, h.s.clock()) {
		t.Error("the bootstrap administrator does not hold the bulk right")
	}
}

func TestIdentitySettingsValidation(t *testing.T) {
	for name, change := range map[string]func(*AccessSettings){
		"role_period_warning_days": func(s *AccessSettings) { s.RolePeriodMaxDays, s.RoleWarningDays = 10, 11 },
		"clock_base":               func(s *AccessSettings) { s.ClockBase = "host" },
		"bulk_revocation_pending":  func(s *AccessSettings) { s.BulkPending = 0 },
	} {
		s := DefaultAccessSettings()
		change(&s)
		if _, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: s}); !errors.Is(err, ErrConfig) || !strings.HasSuffix(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	list := IdentitySettingList()
	list[0].Name = "x"
	if IdentitySettingList()[0].Name != "role_period_max_days" || len(IdentitySettingList()) != 9 {
		t.Error("the identity settings list aliases")
	}
	partial := AccessSettings{RequestHours: 24}.withDefaults()
	if partial.RequestHours != 24 || partial.EndedKept != 0 || partial.BulkMaxTargets != 500 {
		t.Errorf("a partial access group %+v", partial)
	}
}

func TestBulkReadsRefuseAndPage(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	for _, path := range []string{"", "/bulk-revocation-1", "/bulk-revocation-1/items", "/bulk-revocation-1/targets"} {
		if got := f.bulkRead(actor{}, path); got.status != 401 {
			t.Errorf("%q without a session %d", path, got.status)
		}
		if got := f.bulkRead(f.as(MemberLogin), path); got.status != 403 {
			t.Errorf("%q for a member %d", path, got.status)
		}
	}
	for _, path := range []string{"?limit=0", "/x/items?limit=101", "/x/items?outcome=revoked", "/x/items?after=01", "/x/targets?after=-1", "/x/items?after=a"} {
		if got := f.bulkRead(root, path); got.status != 422 {
			t.Errorf("%q %d", path, got.status)
		}
	}
	for _, path := range []string{"/bulk-revocation-9", "/bulk-revocation-9/items", "/bulk-revocation-9/targets"} {
		if got := f.bulkRead(root, path); got.status != 404 {
			t.Errorf("%q unknown %d", path, got.status)
		}
	}
	for i := 0; i < 3; i++ {
		id, _ := f.prepared(t, root, "suspend_membership", SampleMember)
		f.ok(t, "discard", f.run(root, "bulk_revocation.discard", id, nil), "discarded", "")
	}
	first := f.bulkRead(root, "?limit=2")
	if len(first.body["jobs"].([]any)) != 2 || first.body["next"] == "" {
		t.Fatalf("first page %s", first.raw)
	}
	rest := f.bulkRead(root, "?limit=2&after="+first.body["next"].(string))
	if len(rest.body["jobs"].([]any)) != 1 || rest.body["next"] != "" {
		t.Errorf("rest %s", rest.raw)
	}
	if gone := f.bulkRead(root, "?after=bulk-revocation-77"); len(gone.body["jobs"].([]any)) != 0 {
		t.Errorf("after an unknown row %s", gone.raw)
	}
}

func TestBulkCommandEdges(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.fieldRefused(t, "a malformed person", f.stage(root, "", "bad id"), "users")
	f.failed(t, "another target", f.run(root, "bulk_revocation.add_targets", "elsewhere", map[string]string{"users": SampleMember}), commands.PermissionDenied)
	h := bulkCommand{s: f.s}
	if _, res := h.Resolve("other", "x"); res != commands.NotFound {
		t.Error("another workspace")
	}
	if _, res := h.Resolve(SampleWorkspace, "bulk-draft-1"); res != commands.NotFound {
		t.Error("without a session")
	}
	long := strings.Repeat("r", 300)
	got := f.admin(root, "bulk_revocation.add_targets", bulkTarget, long, map[string]string{"users": SampleMember})
	f.failed(t, "a long rationale", got, commands.InvalidRequest)
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	f.failed(t, "prepare a prepared list", f.prepare(root, id, "suspend_user"), commands.InvalidStage)
	staged := f.stage(root, "", SampleAdmin3)
	f.s.fault = func(at string) bool { return at == "roster" }
	f.failed(t, "execute without the roster", f.runWith(root, f.s.Revision(), "bulk_revocation.execute", id, map[string]string{"digest": digest}), commands.DependencyUnavailable)
	f.failed(t, "prepare without the roster", f.runWith(root, f.s.Revision(), "bulk_revocation.prepare", field(staged, "draft"), map[string]string{"effect": "suspend_user"}),
		commands.DependencyUnavailable)
	f.s.fault = func(at string) bool { return at == "bulk-deny" }
	f.failed(t, "a deny conflict", f.execute(root, id, digest), commands.InvalidStage)
	f.s.fault = nil
	delete(f.s.dir.holders, SampleAdmin)
	f.failed(t, "the role ceiling", f.prepare(root, field(staged, "draft"), "suspend_user"), commands.PermissionDenied)
	f.s.dir.holders[SampleAdmin] = f.s.clock().Add(time.Hour)
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.failed(t, "discard an executed list", f.run(root, "bulk_revocation.discard", id, nil), commands.InvalidStage)
	f.s.domain.Lock()
	f.s.answer = gate.Answer{}
	_, err := executeBulk(f.s, commands.Context{Workspace: SampleWorkspace, Target: id}, SampleAdmin)
	f.s.domain.Unlock()
	if failure, ok := err.(*commands.Failure); !ok || failure.Code != commands.PrerequisiteFailed {
		t.Errorf("a non-human executor %v", err)
	}
	f.must(f.s.grantBulk(SampleAdmin3, f.s.clock()))
	f.withoutThirdAdmin()
	f.failed(t, "a bulk holder without administer_installation", f.runWith(f.as(Admin3Login), f.s.Revision(), "bulk_revocation.add_targets", bulkTarget,
		map[string]string{"users": SampleMember}), commands.PermissionDenied)
}

func TestBulkImpactCountsEveryStore(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	member := f.withMember()
	p := f.trustAdmins()
	f.trustPropose(p.b, "60", "0")
	f.requestGrant(f.as(Admin2Login), SampleMember, nil)
	f.proposeChange(f.as(Admin2Login), "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01")
	v := verifierOf(random(tokenBytes))
	f.must(f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute))
	for i := 0; i < 101; i++ {
		f.must(f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, "svc-"+itoa(i), SampleAdmin2))
	}
	f.s.dir.pickups["p"] = pickup{target: SampleAdmin2, expires: f.s.clock().Add(time.Hour)}
	id, _ := f.prepared(t, root, "suspend_user", SampleAdmin2)
	impact := f.bulkRead(root, "/"+id).body["impact"].(map[string]any)
	counts := map[string]int{}
	for _, row := range impact["pending"].([]any) {
		counts[row.(map[string]any)["store"].(string)] = whole(row.(map[string]any)["count"])
	}
	if counts["trust_changes"] != 1 || counts["access_requests"] != 1 || counts["relationship_plans"] != 1 || whole(impact["service_users"]) != 101 ||
		whole(impact["pickups"]) != 1 || impact["other_workspaces"] != "" {
		t.Errorf("impact %+v", impact)
	}
	_ = member
}

func TestProposalStoresCoverEveryProposingNamespace(t *testing.T) {
	f := setup(t, SeedSample).ready()
	known := map[string]bool{}
	for _, store := range proposalStores {
		known[store.prefix] = true
	}
	for _, store := range fallibleProposalStores {
		known[store.prefix] = true
	}
	exempt := map[string]string{"contract": "lifecycle events, no proposer awaiting approval", "bulk_revocation": "drafts change no access", "view_group": "its proposals are relationship plans, counted under relationship"}
	for id := range f.s.mutating {
		namespace, verb, _ := strings.Cut(id, ".")
		if !strings.Contains(verb, "propose") && !strings.Contains(verb, "request") && !strings.Contains(verb, "withdraw") && verb != "discard" {
			continue
		}
		if !known[namespace] && exempt[namespace] == "" {
			t.Errorf("%s creates or withdraws proposals but no proposal store is enumerated", id)
		}
	}
}

func TestSweepEdges(t *testing.T) {
	var none Server
	none.identity, _ = identity.New(Installation, IdentityLimits(), time.Now)
	if none.sweepRoles() {
		t.Error("a sweep before start")
	}
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	ws := f.s.identity.Workspace()
	v := verifierOf(random(tokenBytes))
	f.must(f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute))
	end := f.s.clock().Add(time.Minute)
	f.must(ws.SetRolePeriod(v, SampleWorkspace, SampleAdmin2, end, f.membership(SampleAdmin2).Version, f.s.clock()))
	f.s.dir.pickups["p"] = pickup{target: SampleAdmin2, expires: f.s.clock().Add(time.Hour)}
	f.advance(time.Minute)
	f.sweep()
	v = verifierOf(random(tokenBytes))
	f.must(f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute))
	end = f.s.clock().Add(time.Minute)
	f.must(ws.SetRolePeriod(v, SampleWorkspace, SampleAdmin3, end, f.membership(SampleAdmin3).Version, f.s.clock()))
	f.must(ws.SetRolePeriod(v, SampleWorkspace, SampleAdmin, end, f.membership(SampleAdmin).Version, f.s.clock()))
	f.advance(time.Minute)
	f.sweep()
	if !strings.Contains(logs.String(), "role_expiry_blocked") {
		t.Error("a blocked end is not logged")
	}
	if _, ok := f.s.dir.pickups["p"]; ok {
		t.Error("a pickup survived the end")
	}
	if commitReason(errors.New("x")) != decisionledger.ReasonCommitFailed {
		t.Error("a foreign error")
	}
	if _, ok := f.s.dir.userOf("nobody@example.invalid"); ok {
		t.Error("an unknown login")
	}
	if f.s.strandsAnchors(map[string]identity.MemberInfo{}, map[string]bool{"x": true}, f.s.clock()) {
		t.Error("no anchors to strand")
	}
}

func TestHoldDecidesThePeriodDirection(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "an end", f.period(root, SampleMember, f.at(time.Hour)), "role_period_set", "")
	f.ok(t, "an end on the third", f.period(root, SampleAdmin3, f.at(time.Hour)), "role_period_set", "")
	f.s.domain.Lock()
	f.s.enterHold("test")
	f.s.domain.Unlock()
	f.failed(t, "a later end during the hold", f.period(root, SampleMember, f.at(2*time.Hour)), commands.DependencyUnavailable)
	f.failed(t, "removing the end during the hold", f.period(root, SampleMember, ""), commands.DependencyUnavailable)
	f.ok(t, "an earlier end during the hold", f.period(root, SampleMember, f.at(30*time.Minute)), "role_period_set", "")
	f.failed(t, "a demotion that removes an end", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "end": endNone}), commands.DependencyUnavailable)
	f.failed(t, "a demotion with a later end", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "role_until": f.at(2 * time.Hour)}), commands.DependencyUnavailable)
	f.ok(t, "a demotion keeping the end", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}), "role_member", "")
	f.ok(t, "a shorter end on the same role", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "role_until": f.at(30 * time.Minute)}), "role_member", "")
	f.failed(t, "a promotion during the hold", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator", "role_until": f.at(20 * time.Minute)}), commands.DependencyUnavailable)
	f.failed(t, "a malformed end during the hold", f.run(root, "identity.set_role_period", SampleMember, map[string]string{"end": endAt, "role_until": "x"}),
		commands.DependencyUnavailable)
	f.failed(t, "an unknown person during the hold", f.period(root, "user-nobody", ""), commands.DependencyUnavailable)
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Errorf("bulk execution during the hold %s", got.raw)
	}
}

func TestDecreaseFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.s.fault = func(at string) bool { return at == "period-apply" }
	f.failed(t, "a failed shortening", f.period(root, SampleMember, f.at(time.Hour)), commands.DependencyUnavailable)
	f.s.fault = func(at string) bool { return at == "increase-refuse" }
	f.ok(t, "an end", f.period(root, SampleMember, f.at(time.Hour)), "role_period_set", "")
	f.failed(t, "a refused lengthening", f.period(root, SampleMember, f.at(2*time.Hour)), commands.InvalidStage)
	f.failed(t, "a refused promotion", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator", "role_until": f.at(time.Hour)}),
		commands.InvalidStage)
	f.failed(t, "a refused reinstatement", f.run(root, "identity.reinstate_membership", SampleMember, nil), commands.InvalidStage)
	f.s.fault = nil
}

var _ = bulkrevoke.Draft
