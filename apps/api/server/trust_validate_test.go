package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func (f world) validationItems(id string) map[string]trustchange.Item {
	out := map[string]trustchange.Item{}
	for _, item := range f.view(id).Validation.Items {
		out[item.Item] = item
	}
	return out
}

func (f world) blockedBy(t *testing.T, name string, p admins, change func(), item, detail string) {
	t.Helper()
	change()
	id := changeOf(t, f.trustPropose(p.a, fmt.Sprint(40+len(f.s.trust.List(SampleWorkspace))), "0"))
	got := f.trustValidate(p.b, id)
	items := f.validationItems(id)
	if got.status != 200 || got.body["business_state"] != trustchange.ResultBlocked || items[item].Result != resultBlocked || !strings.Contains(items[item].Detail, detail) {
		t.Errorf("%s: %s %+v", name, got.raw, items[item])
	}
	if !strings.Contains(fmt.Sprint(got.body["fields"]), item) || f.view(id).State != trustchange.StateDraft {
		t.Errorf("%s: blocked items %v", name, got.body["fields"])
	}
	f.code(t, name+" approve", f.trustApprove(p.b, id), 422, string(commands.InvalidStage))
}

func TestValidationItems(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.a, "60", "0"))
	if got := f.trustValidate(p.b, id); got.status != 200 || got.body["business_state"] != trustchange.ResultValidated {
		t.Fatal(got.raw)
	}
	items := f.validationItems(id)
	for item, want := range map[string]string{itemTestLogin: resultVerified, itemRecovery: resultVerified, itemIssuer: resultNotApplicable, itemDiagnostics: resultNotApplicable,
		itemExternal: resultNotApplicable, itemMFA: resultInformation, itemAffected: resultInformation, itemSessions: resultInformation, itemRecord: resultVerified} {
		if items[item].Result != want {
			t.Errorf("%s %+v", item, items[item])
		}
	}
	if items[itemAffected].Detail != "3" || items[itemSessions].Detail != "3" || items[itemMFA].Reason != "unchanged" || items[itemIssuer].Reason != "local_password" {
		t.Errorf("items %+v", items)
	}
	row := f.policyRead(p.c).body["changes"].([]any)[0].(map[string]any)
	impact := row["impact"].(map[string]any)
	if fmt.Sprint(impact["members"]) != "3" || fmt.Sprint(impact["sessions"]) != "3" || len(impact["affected_ids"].([]any)) != 3 || impact["signs_out_everyone"] != true ||
		len(row["validation"].(map[string]any)["items"].([]any)) != 9 {
		t.Errorf("row %+v", row)
	}
	first := f.view(id).Validation.Digest
	f.advance(time.Minute)
	f.signedIn(Admin3Login, pw)
	if got := f.trustValidate(p.b, id); got.status != 200 || f.view(id).Validation.Digest != first {
		t.Errorf("times and session counts changed the evidence %s", got.raw)
	}
	if f.view(id).History[len(f.view(id).History)-1].State != trustchange.StateValidated {
		t.Error("revalidation history")
	}
}

func TestRecoveryQualification(t *testing.T) {
	two := setup(t, SeedSample).ready()
	for _, login := range []string{AdminLogin, Admin2Login} {
		two.setPassword(login, pw)
	}
	p := admins{two.signedIn(AdminLogin, pw), two.signedIn(Admin2Login, pw), two.as(Admin3Login)}
	two.blockedBy(t, "two qualifying people", p, func() {}, itemRecovery, "three_distinct_people")
	f := setup(t, SeedSample).ready()
	p = f.trustAdmins()
	revoke := func(id string) func() {
		return func() {
			g, _ := f.s.grantView(id)
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.blockedBy(t, "a gate refusal", p, func() {
		for _, right := range trustRights {
			revoke("trust-" + string(right) + "-" + SampleAdmin3)()
		}
	}, itemRecovery, "three_distinct_people")
	g := setup(t, SeedSample).ready()
	q := g.trustAdmins()
	g.blockedBy(t, "no activate holder", q, func() {
		for _, user := range []string{SampleAdmin, SampleAdmin2, SampleAdmin3} {
			v, _ := g.s.grantView("trust-" + string(gate.TrustActivate) + "-" + user)
			g.must(g.s.admin.RevokeGrant(v.ID, v.Version))
		}
	}, itemRecovery, "missing:trust_activate")
	h := setup(t, SeedSample).ready()
	r := h.trustAdmins()
	h.blockedBy(t, "awaiting a reset link", r, func() {
		h.s.dir.links[Admin3Login] = outstanding{purpose: passwords.PurposeReset, expires: h.s.clock().Add(time.Hour)}
	}, itemRecovery, "three_distinct_people")
	k := setup(t, SeedSample).ready()
	s := k.trustAdmins()
	k.blockedBy(t, "suspended", s, func() {
		if got := k.run(s.a, "identity.suspend_user", SampleAdmin3, nil); got.status != 200 {
			t.Fatal(got.raw)
		}
	}, itemRecovery, "three_distinct_people")
	m := setup(t, SeedSample).ready()
	u := m.trustAdmins()
	m.blockedBy(t, "an authentication block", u, func() {
		at := authpolicy.Authentication{Method: authpolicy.MethodPassword, At: m.s.clock()}
		admin := func(user string) authpolicy.Principal {
			return authpolicy.Principal{User: user, Workspace: SampleWorkspace, Role: authpolicy.RoleAdministrator, Kind: authpolicy.KindHuman}
		}
		plan, err := m.s.policy.Propose(authpolicy.Change{Kind: authpolicy.ChangePlan, Plan: authpolicy.Plan{Workspace: SampleWorkspace, Users: []string{SampleAdmin3}, Chunk: 1,
			Reason: "incident"}}, admin(SampleAdmin), at)
		m.must(err)
		_, err = m.s.policy.Approve(plan.Number, admin(SampleAdmin2), at)
		m.must(err)
	}, itemRecovery, "three_distinct_people")
}

func TestCandidateThatLocksEveryoneOutIsBlocked(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	base, _ := f.s.trustBase(SampleWorkspace)
	v, err := f.s.trust.Propose(trustchange.Proposal{Workspace: SampleWorkspace, Route: trustchange.RouteLocalPassword, Proposer: SampleAdmin, Base: base,
		Candidate: trustchange.Candidate{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFAAdministrators, MaxAge: time.Hour}})
	f.must(err)
	got := f.trustValidate(p.b, v.ID)
	items := f.validationItems(v.ID)
	if got.body["business_state"] != trustchange.ResultBlocked || items[itemTestLogin].Reason != "test_login_failed" || items[itemTestLogin].Detail != authpolicy.ReasonMFARequired ||
		items[itemRecovery].Result != resultBlocked || items[itemMFA].Reason != "unchanged" {
		t.Errorf("%s %+v", got.raw, items)
	}
	weaker, err := f.s.trust.Propose(trustchange.Proposal{Workspace: SampleWorkspace, Route: trustchange.RouteLocalPassword, Proposer: SampleAdmin2, Base: base,
		Candidate: trustchange.Candidate{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAge: 2 * time.Hour}})
	f.must(err)
	f.publishDirect(authpolicy.Revision{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFAAdministrators, MaxAge: 12 * time.Hour})
	if got := refusalReason(authpolicy.ErrInvalid); got != "invalid" {
		t.Errorf("reason %s", got)
	}
	_, current := f.s.trustBase(SampleWorkspace)
	if mfaRank(weaker.Candidate.MFA) >= mfaRank(current.MFA) {
		t.Error("weakening is not detected")
	}
}

func TestEvidenceStale(t *testing.T) {
	for name, change := range map[string]func(f world, p admins){
		"role": func(f world, p admins) {
			f.run(p.a, "identity.change_role", SampleAdmin2, map[string]string{"role": "member"})
		},
		"membership": func(f world, p admins) { f.run(p.a, "identity.suspend_membership", SampleAdmin2, nil) },
		"user":       func(f world, p admins) { f.run(p.a, "identity.suspend_user", SampleAdmin2, nil) },
		"grant": func(f world, p admins) {
			g, _ := f.s.grantView("trust-" + string(gate.TrustApprove) + "-" + SampleAdmin2)
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		},
		"client grant": func(f world, p admins) {
			g, _ := f.s.grantView("trust-" + string(gate.TrustValidate) + "-console")
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		},
		"a member promoted": func(f world, p admins) {
			f.setPassword(MemberLogin, pw)
			f.run(p.a, "identity.change_role", SampleMember, map[string]string{"role": "administrator"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			p := f.trustAdmins()
			id := f.approvedChange(p, "60", "0")
			change(f, p)
			row := f.policyRead(p.c).body["changes"].([]any)[0].(map[string]any)
			if row["validation"].(map[string]any)["evidence_current"] != false || row["why_not"].(map[string]any)["activate"] != "evidence_stale" {
				t.Errorf("read %+v", row["why_not"])
			}
			f.code(t, "activate", f.trustActivate(p.c, id), 409, string(commands.EvidenceStale))
		})
	}
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	g, _ := f.s.grantView("trust-" + string(gate.TrustApprove) + "-" + SampleAdmin2)
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	terms := g.Terms
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "trust-short", SubjectID: SampleAdmin2, Terms: terms, Validity: gate.Validity{NotBefore: start, NotAfter: start.Add(10 * time.Minute)}}))
	id := f.approvedChange(p, "60", "0")
	f.advance(11 * time.Minute)
	if f.reauth(p.c, pw).status != 204 {
		t.Fatal("reauthenticate")
	}
	f.code(t, "an expired approver grant", f.trustActivate(p.c, id), 409, string(commands.EvidenceStale))
	if got := f.trustValidate(p.c, id); got.status != 200 || f.view(id).Approver != "" || f.view(id).State != trustchange.StateValidated {
		t.Errorf("revalidation clears the approval %s %+v", got.raw, f.view(id))
	}
	if h := f.view(id).History; h[len(h)-2].Rationale != trustchange.HistoryInvalidated {
		t.Errorf("history %+v", h)
	}
}

func emptyInstallation(t *testing.T) (world, actor) {
	f := setup(t, SeedNone).ready()
	tok := f.links()[""]
	if got := f.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, f.pre()); got.status != 204 {
		t.Fatal(got.raw)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	f.must(err)
	return f, actor{cookie, csrf}
}

func TestEmptyInstallationEnrolmentBlock(t *testing.T) {
	f, a := emptyInstallation(t)
	page := f.policyRead(a).body
	holders := page["holders"].(map[string]any)
	if fmt.Sprint(holders["qualifying_people"]) != "1" || holders["enrolment"] != "not_decided" || page["rights"].(map[string]any)["activate"] != true || page["policy"] != nil {
		t.Errorf("holders %+v", page)
	}
	propose := func(epoch, policy string) reply {
		return f.run(a, "trust.propose", trustTarget, map[string]string{"max_age_minutes": "60", "step_up_minutes": "0", "expected_epoch": epoch, "expected_policy": policy})
	}
	f.code(t, "a non-zero epoch", propose("1", "0"), 409, string(commands.AuthorityEpochConflict))
	f.code(t, "a non-zero policy", propose("0", "1"), 409, string(commands.AuthorityEpochConflict))
	id := changeOf(t, propose("0", "0"))
	v, _ := f.s.trust.Get(id, EmptyWorkspace)
	if v.OldEpoch != 0 || v.OldPolicy != 0 || v.OldDigest != trustchange.EmptyDigest() {
		t.Errorf("no policy base %+v", v)
	}
	got := f.run(a, "trust.validate", id, map[string]string{"expected_change_revision": "1", "change_digest": v.ChangeDigest, "expected_epoch": "0"})
	v, _ = f.s.trust.Get(id, EmptyWorkspace)
	if got.body["business_state"] != trustchange.ResultBlocked || v.Validation.Items[1].Reason != "recovery_unverified" {
		t.Errorf("enrolment block %s %+v", got.raw, v.Validation.Items[1])
	}
	if got := f.run(a, "trust.cancel", id, map[string]string{"expected_change_revision": "1"}); got.status != 200 {
		t.Errorf("cancel %s", got.raw)
	}
}
