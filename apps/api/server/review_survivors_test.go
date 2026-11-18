package server

import (
	"bytes"
	"encoding/json"
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
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func TestEveryProposalStoreCountsOnlyThePersonsPendingProposals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	s := f.s
	const who, other = "user-target", "user-other"
	s.ret.entries = []*retentionEntry{{view: retention.ProposalView{Proposer: who, State: retention.ProposalPending}}, {view: retention.ProposalView{Proposer: who, State: retention.ProposalPending}}, {view: retention.ProposalView{Proposer: who}},
		{view: retention.ProposalView{Proposer: other, State: retention.ProposalPending}}}
	s.rec.entries = []*recoveryEntry{{view: recovery.ProposalView{Proposer: who, State: recovery.ProposalPending}}, {view: recovery.ProposalView{Proposer: who, State: recovery.ProposalPending}}, {view: recovery.ProposalView{Proposer: who}},
		{view: recovery.ProposalView{Proposer: other, State: recovery.ProposalPending}}}
	s.install.entries = []*installEntry{{view: installer.ProposalView{Proposer: who, State: installer.ProposalPending}}, {view: installer.ProposalView{Proposer: who, State: installer.ProposalPending}}, {view: installer.ProposalView{Proposer: who}},
		{view: installer.ProposalView{Proposer: other, State: installer.ProposalPending}}}
	s.integ.entries = []*pluginEntry{{view: plugins.ProposalView{Proposer: who, State: plugins.ProposalPending}}, {view: plugins.ProposalView{Proposer: who, State: plugins.ProposalPending}}, {view: plugins.ProposalView{Proposer: who}},
		{view: plugins.ProposalView{Proposer: other, State: plugins.ProposalPending}}}
	s.integ.redeliver = map[string]letterRequest{"a": {by: who}, "b": {by: other}, "e": {by: who}}
	s.integ.dismiss = map[string]letterRequest{"c": {by: who}, "d": {by: other}}
	proposed := 0
	for _, e := range s.integ.subscriptions {
		v, err := s.subscriptionOf(e.id)
		e.proposer = other
		if err == nil && v.State == webhooks.SubscriptionProposed && proposed == 0 {
			e.proposer, proposed = who, 1
		} else if err == nil && v.State != webhooks.SubscriptionProposed {
			e.proposer = who
		}
	}
	extra := []*subscriptionEntry{{id: "missing", proposer: who}}
	for _, e := range s.integ.subscriptions {
		if v, err := s.subscriptionOf(e.id); err == nil && v.State != webhooks.SubscriptionProposed {
			extra = append(extra, &subscriptionEntry{id: e.id, proposer: who})
		}
	}
	s.integ.subscriptions = append(s.integ.subscriptions, extra...)
	s.plans.byKey = map[string]*planMeta{"p1": {proposer: who}, "p5": {proposer: who}, "p2": {proposer: who, closed: "x"}, "p3": {proposer: who, activated: 1}, "p4": {proposer: other}}
	s.ev.entries = []*evidenceEntry{{proposer: who, state: entryPending}, {proposer: who, state: entryPending}, {proposer: who, state: "approved"}, {proposer: other, state: entryPending}}
	want := map[string]int{"retention_proposals": 2, "recovery_proposals": 2, "installation_proposals": 2, "integration_proposals": 2 + 3 + proposed,
		"relationship_plans": 2, "release_evidence_proposals": 2}
	if proposed != 1 {
		t.Fatal("the sample seed has no proposed subscription")
	}
	for _, store := range proposalStores {
		if n, ok := want[store.name]; ok {
			if got := store.count(s, who); got != n {
				t.Errorf("%s counted %d, want %d", store.name, got, n)
			}
		}
	}
}

func TestMemberStateNamesAnEndedActiveMembership(t *testing.T) {
	for _, c := range []struct {
		m    identity.MemberInfo
		want string
	}{
		{identity.MemberInfo{State: identity.MembershipActive, Effective: true}, "active"},
		{identity.MemberInfo{State: identity.MembershipActive}, "ended"},
		{identity.MemberInfo{State: identity.MembershipSuspended}, string(identity.MembershipSuspended)},
		{identity.MemberInfo{State: identity.MembershipSuspended, Effective: true}, string(identity.MembershipSuspended)},
	} {
		if got := memberState(c.m); got != c.want {
			t.Errorf("%+v: %s", c.m, got)
		}
	}
}

func TestTheWallLeadLogsOncePerEpisode(t *testing.T) {
	var logs bytes.Buffer
	src := &scripted{wall: time.Now()}
	c := newServerClock(src.Wall, src.Elapsed, slog.New(slog.NewJSONHandler(&logs, nil)))
	c.rebase(src.Wall(), time.Hour)
	src.step(2*time.Hour, 0)
	c.now()
	c.now()
	src.step(-time.Hour-30*time.Minute, time.Minute)
	c.now()
	src.step(2*time.Hour, 0)
	c.now()
	if n := strings.Count(logs.String(), "clock_wall_lead"); n != 2 {
		t.Errorf("wall lead logged %d times over two episodes", n)
	}
	exact := &scripted{wall: time.Now()}
	d := newServerClock(exact.Wall, exact.Elapsed, slog.New(slog.DiscardHandler))
	d.rebase(exact.Wall(), time.Hour)
	exact.step(time.Hour, 0)
	d.now()
	if _, beyond := d.leadBeyond(); beyond {
		t.Error("a lead of exactly the tolerance was flagged")
	}
}

func TestABulkRationaleOfTheFullLengthIsAccepted(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	got := f.admin(root, "bulk_revocation.add_targets", bulkTarget, strings.Repeat("r", maxRationale), map[string]string{"users": SampleMember})
	if got.status != 200 {
		t.Errorf("a %d-byte rationale %s", maxRationale, got.raw)
	}
}

func TestSuspendUserRecordsCarryTheUserEpoch(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.ok(t, "a new version", f.period(root, SampleAdmin3, f.at(2*time.Hour)), "role_period_set", "")
	f.ok(t, "another version", f.period(root, SampleAdmin3, f.at(time.Hour)), "role_period_set", "")
	id, digest := f.prepared(t, root, "suspend_user", SampleAdmin3)
	v, _ := f.s.bulk.Get(f.s.bootWorkspace, "", id)
	want := v.Targets[0]
	if want.Epoch == want.Version {
		t.Fatalf("epoch and version must differ for this test: %+v", want)
	}
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	for _, rec := range f.s.ledger.Decisions() {
		if rec.Decision.Decision == decisionledger.SuspendUser && rec.Decision.TargetVersion != want.Epoch {
			t.Errorf("suspend_user recorded version %d, epoch %d", rec.Decision.TargetVersion, want.Epoch)
		}
	}
}

func TestReplayKeepsTheNewestJobsUpToTheLimit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for i := 1; i <= 11; i++ {
		f.craft(t, decisionledger.Decision{Decision: decisionledger.BulkRevocationJob, Target: "bulk-revocation-" + itoa(i), RequestDigest: digestOf("x"), Effect: "suspend_user",
			RequestedBy: SampleAdmin, ApprovedBy: SampleAdmin2, Targets: 1})
	}
	g := f.restart(func(c *Config) { c.Access.BulkKept = 10 })
	_, first := g.s.bulk.Get(SampleWorkspace, "", "bulk-revocation-1")
	_, last := g.s.bulk.Get(SampleWorkspace, "", "bulk-revocation-11")
	_, second := g.s.bulk.Get(SampleWorkspace, "", "bulk-revocation-2")
	if first || !last || !second {
		t.Errorf("kept: oldest %v, second %v, newest %v", first, second, last)
	}
}

func TestReplayRebuildsTheItemsOfAnAccountJob(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.craft(t, decisionledger.Decision{Decision: decisionledger.BulkRevocationJob, Target: "bulk-revocation-7", RequestDigest: digestOf("x"), Effect: "suspend_user",
		RequestedBy: SampleAdmin, ApprovedBy: SampleAdmin, Targets: 1})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendUser, Target: SampleAdmin3, TargetVersion: 1, Revocation: "bulk-revocation-7:0",
		Request: "bulk-revocation-7", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	g := f.restart(nil)
	v, ok := g.s.bulk.Get(SampleWorkspace, "", "bulk-revocation-7")
	if !ok || len(v.Items) != 1 || v.Items[0].Target.User != SampleAdmin3 {
		t.Errorf("rebuilt account job %+v", v)
	}
}

func TestReplayRebuildsTheItemsOfAMembershipJob(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.craft(t, decisionledger.Decision{Decision: decisionledger.BulkRevocationJob, Target: "bulk-revocation-6", RequestDigest: digestOf("x"), Effect: "suspend_membership",
		RequestedBy: SampleAdmin, ApprovedBy: SampleAdmin, Targets: 1})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendMembership, Target: SampleAdmin3, TargetVersion: 1, Revocation: "bulk-revocation-6:0",
		Request: "bulk-revocation-6", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	g := f.restart(nil)
	v, ok := g.s.bulk.Get(SampleWorkspace, "", "bulk-revocation-6")
	if !ok || len(v.Items) != 1 || v.Items[0].Target.User != SampleAdmin3 {
		t.Errorf("rebuilt membership job %+v", v)
	}
}

func TestReplayKeepsTheRecordedEndOfAnEndedPeriod(t *testing.T) {
	f := setup(t, SeedSample).ready()
	end := f.s.clock().Add(-time.Minute).UTC().Round(0)
	f.craft(t, decisionledger.Decision{Decision: decisionledger.RolePeriodEnded, Target: SampleAdmin3, TargetVersion: 2, Role: "administrator", Until: stamp(end),
		Revocation: "role-end-1", RequestedBy: decisionledger.RoleSystem})
	g := f.restart(nil)
	if m := g.membership(SampleAdmin3); m.State != identity.MembershipSuspended || !m.Until.Equal(end) {
		t.Errorf("an ended period after restart %+v (end %v)", m, end)
	}
}

func TestARecordAppliedAtAcceptanceIsNotedAsReapplied(t *testing.T) {
	f := setup(t, SeedSample).ready()
	seq := f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendMembership, Target: SampleMember, TargetVersion: 1, Revocation: "bulk-revocation-4:0",
		Request: "bulk-revocation-4", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	g := f.restart(nil)
	g.withMember()
	if note := g.s.replayNotes[itoa(int(seq))]; note != noteReapplied {
		t.Errorf("note after acceptance %q", note)
	}
}

func TestALedgerExactlyAtTheToleranceStartsWithoutAFlag(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.aheadLedger(t, 24*time.Hour)
	if _, err := f.restartWith(nil); err != nil {
		t.Errorf("a ledger exactly 24 hours ahead %v", err)
	}
}

func TestAnEarlierVerifiedChoiceDoesNotCoverALaterDamagedOne(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.aheadLedger(t, 48*time.Hour)
	g := f.restart(func(c *Config) { c.Access.ClockBase = ClockBaseWall })
	g.advance(48 * time.Hour)
	g.withMember()
	g.ok(t, "a second record in the future", g.period(g.as(AdminLogin), SampleMember, g.at(2*time.Hour)), "role_period_set", "")
	g.advance(-48 * time.Hour)
	h := g.restart(func(c *Config) { c.Access.ClockBase = ClockBaseWall })
	choices := []int{}
	for i, line := range h.ledgerLines(t) {
		if bytes.Contains(line, []byte(`"`+decisionledger.ClockBaseChoice+`"`)) {
			choices = append(choices, i+1)
		}
	}
	if len(choices) != 2 || choices[1]-choices[0] < 2 {
		t.Fatalf("two separated clock base choices expected %v", choices)
	}
	h.breakLine(t, choices[1]-1)
	if _, err := h.restartWith(func(c *Config) { c.Access.ClockBase = "" }); !errors.Is(err, ErrStart) || !strings.Contains(err.Error(), "could not be verified") {
		t.Errorf("a damaged latest choice behind a verified earlier one %v", err)
	}
}

func TestBulkReadsPageExactlyAndAnswerOnce(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember, SampleAdmin, "user-00000000000000ff")
	f.execute(root, id, digest)
	f.s.drainCleanup()
	if job := f.job(root, id); job["ledger"].(map[string]any)["reason"] != "" {
		t.Errorf("a recorded job reads as unconfirmed %+v", job["ledger"])
	}
	if got := f.bulkRead(root, "?limit=1"); got.status != 200 || got.body["next"] != "" || len(got.body["jobs"].([]any)) != 1 {
		t.Errorf("a page that holds every job %s", got.raw)
	}
	if got := f.bulkRead(root, "/"+id+"/items?after=0&limit=1"); got.status != 200 || got.body["next"] != "1" {
		t.Errorf("an offset of zero %s", got.raw)
	}
	if got := f.bulkRead(root, "/"+id+"/items?after=2&limit=1"); got.status != 200 || got.body["next"] != "" {
		t.Errorf("the last page %s", got.raw)
	}
	anonymous := f.bulkRead(actor{}, "")
	var one map[string]any
	if anonymous.status != 401 || json.Unmarshal([]byte(anonymous.raw), &one) != nil {
		t.Errorf("an unauthenticated read %d %q", anonymous.status, anonymous.raw)
	}
}

func TestClassifyNamesEveryExclusion(t *testing.T) {
	f := setup(t, SeedSample).ready()
	users := map[string]identity.UserInfo{
		"service":   {ID: "service", Kind: identity.Service, State: identity.Active},
		"departed":  {ID: "departed", Kind: identity.Human, State: identity.Departed},
		"loner":     {ID: "loner", Kind: identity.Human, State: identity.Active},
		"ended":     {ID: "ended", Kind: identity.Human, State: identity.Active},
		"suspended": {ID: "suspended", Kind: identity.Human, State: identity.Suspended},
		"fine":      {ID: "fine", Kind: identity.Human, State: identity.Active},
		"me":        {ID: "me", Kind: identity.Human, State: identity.Active},
	}
	members := map[string]identity.MemberInfo{
		"service":   {User: "service", Role: identity.Member, State: identity.MembershipActive, Effective: true},
		"departed":  {User: "departed", Role: identity.Member, State: identity.MembershipActive, Effective: true},
		"ended":     {User: "ended", Role: identity.Member, State: identity.MembershipActive},
		"suspended": {User: "suspended", Role: identity.Member, State: identity.MembershipActive, Effective: true},
		"fine":      {User: "fine", Role: identity.Member, State: identity.MembershipActive, Effective: true},
		"me":        {User: "me", Role: identity.Administrator, State: identity.MembershipActive, Effective: true},
	}
	r := roster{users: users, members: members}
	ids := []string{"me", "unknown", "service", "departed", "loner", "ended", "suspended", "fine"}
	reasons := func(effect bulkrevoke.Effect) map[string]string {
		targets, exclusions := f.s.classify("me", ids, effect, r, f.s.clock(), nil)
		out := map[string]string{}
		for _, x := range exclusions {
			out[x.User] = x.Reason
		}
		for _, t := range targets {
			out[t.User] = "target"
		}
		return out
	}
	membership := map[string]string{"me": bulkrevoke.ExcludedSelf, "unknown": bulkrevoke.ExcludedNotFound, "service": bulkrevoke.ExcludedNotFound,
		"departed": bulkrevoke.ExcludedNotFound, "loner": bulkrevoke.ExcludedNoMembership, "ended": bulkrevoke.ExcludedAlreadyRevoked, "suspended": "target", "fine": "target"}
	account := map[string]string{"me": bulkrevoke.ExcludedSelf, "unknown": bulkrevoke.ExcludedNotFound, "service": bulkrevoke.ExcludedNotFound,
		"departed": bulkrevoke.ExcludedNotFound, "loner": bulkrevoke.ExcludedNotFound, "ended": "target", "suspended": bulkrevoke.ExcludedAlreadyRevoked, "fine": "target"}
	for effect, want := range map[bulkrevoke.Effect]map[string]string{bulkrevoke.SuspendMembership: membership, bulkrevoke.SuspendUser: account} {
		got := reasons(effect)
		for id, reason := range want {
			if got[id] != reason {
				t.Errorf("%s %s: %q, want %q", effect, id, got[id], reason)
			}
		}
	}
}

func TestActiveAdministratorsNeedAnEffectiveRoleAndAnActiveUser(t *testing.T) {
	got := activeAdministrators(map[string]identity.MemberInfo{
		"a": {Role: identity.Administrator, Effective: true, UserState: identity.Active},
		"b": {Role: identity.Administrator, UserState: identity.Active},
		"c": {Role: identity.Administrator, Effective: true, UserState: identity.Suspended},
		"d": {Role: identity.Member, Effective: true, UserState: identity.Active},
	})
	if len(got) != 1 || !got["a"] {
		t.Errorf("active administrators %v", got)
	}
}

func TestAPromotionWithoutALaterEndIsRefusedWhileTheLedgerCannotRecord(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "an end", f.period(root, SampleMember, f.at(time.Hour)), "role_period_set", "")
	f.stoppedLedger()
	f.failed(t, "a promotion keeping the end", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), commands.DependencyUnavailable)
}

func TestAPromotionIsRefusedAfterAWallLead(t *testing.T) {
	src := &scripted{wall: start}
	f := setupWith(t, func(c *Config) { c.Clock, c.Elapsed = src.Wall, src.Elapsed })
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "suspend a membership", f.run(root, "identity.suspend_membership", SampleAdmin3, nil), "membership_suspended", "")
	src.step(25*time.Hour, time.Minute)
	f.failed(t, "a promotion after a lead", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), commands.DependencyUnavailable)
	f.failed(t, "a membership reinstatement after a lead", f.run(root, "identity.reinstate_membership", SampleAdmin3, nil), commands.DependencyUnavailable)
}

func TestAPrepareWithoutAnEffectNamesTheField(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	staged := f.stage(root, "", SampleMember)
	f.fieldRefused(t, "no effect", f.run(root, "bulk_revocation.prepare", field(staged, "draft"), nil), "effect")
}

func TestImpactCountsRolesRightsPickupsAndOwnedServices(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.as(Admin2Login)
	v := verifierOf(random(tokenBytes))
	f.must(f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute))
	f.must(f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, "svc-a", SampleAdmin2))
	f.must(f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, "svc-b", SampleAdmin2))
	f.must(f.s.identity.Workspace().CreateServiceUser(v, SampleWorkspace, "svc-c", SampleAdmin))
	f.s.domain.Lock()
	sessions := f.s.sessionsOf(SampleAdmin2)
	f.s.dir.pickups["by-session"] = pickup{session: sessions[0], target: SampleAdmin, expires: f.s.clock().Add(time.Hour)}
	now := f.s.clock()
	trust, bulk := 0, 0
	for _, user := range []string{SampleAdmin2, SampleAdmin3, SampleMember} {
		for _, right := range trustRights {
			if f.s.holdsRight(user, right, now) {
				trust++
				break
			}
		}
		if f.s.holdsRight(user, gate.RevokeAccessInBulk, now) {
			bulk++
		}
	}
	f.s.domain.Unlock()
	id, _ := f.prepared(t, root, "suspend_membership", SampleAdmin2, SampleAdmin3, SampleMember)
	impact := f.bulkRead(root, "/"+id).body["impact"].(map[string]any)
	if whole(impact["administrators"]) != 2 || whole(impact["members"]) != 1 || whole(impact["pickups"]) != 1 || whole(impact["service_users"]) != 2 ||
		whole(impact["trust_right_holders"]) != trust || whole(impact["bulk_right_holders"]) != bulk || bulk == 0 {
		t.Errorf("impact %+v (trust %d, bulk %d)", impact, trust, bulk)
	}
}

func TestClockDiagnosticsAtTheThreshold(t *testing.T) {
	src := &scripted{wall: start}
	f := setupWith(t, func(c *Config) { c.Clock, c.Elapsed = src.Wall, src.Elapsed })
	root := f.as(AdminLogin)
	src.step(-clockDiagnostic, 0)
	if rows := f.list(root, "").body; rows["clock_behind"] != false || rows["recovery_state"].(map[string]any)["stranded"] != false {
		t.Errorf("exactly the threshold behind %+v %+v", rows["clock_behind"], rows["recovery_state"])
	}
	src.step(-time.Second, 0)
	if rows := f.list(root, "").body; rows["clock_behind"] != true {
		t.Errorf("past the threshold behind %+v", rows["clock_behind"])
	}
}

func TestASetRolePeriodNeedsTheEndField(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.fieldRefused(t, "no end field", f.run(root, "identity.set_role_period", SampleMember, map[string]string{"role_until": f.at(time.Hour)}), "end")
}
