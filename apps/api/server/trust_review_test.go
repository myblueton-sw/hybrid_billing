package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func TestAnOlderMismatchedLedgerIsNotPublished(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	f.s.Close()
	marker := filepath.Join(f.s.cfg.Data, decisionledger.MarkerName)
	f.must(os.WriteFile(marker, []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"x","g":3}`), 0o600))
	g := f.restart(nil)
	current, _ := g.s.policy.Policy(SampleWorkspace)
	trust := g.trustNow(t)
	if current.Epoch != 3<<32+1 || current.Revision.MaxAge != time.Hour || trust["older_record_ignored"] != true || trust["origin"] != originSeed ||
		len(g.s.trust.List(SampleWorkspace)) != 0 {
		t.Errorf("a record older than the marker generation was published %+v %+v", current, trust)
	}
	h := setup(t, SeedSample).ready()
	h.activated(t, "60")
	h.s.Close()
	h.appendGarbage(t)
	k := h.restart(func(c *Config) { c.Access.AcceptNewLedger = true })
	replaced := k.s.ledger.Status().ReplacedFile
	k.s.Close()
	old, err := os.ReadFile(filepath.Join(k.s.cfg.Data, decisionledger.DirName, replaced))
	k.must(err)
	k.must(os.WriteFile(ledgerFile(k), old, 0o600))
	m := k.restart(func(c *Config) { c.Access.AcceptNewLedger = false })
	st := m.s.ledger.Status()
	if current, _ := m.s.policy.Policy(SampleWorkspace); st.Isolation != decisionledger.IsolationMismatch || current.Epoch != 1<<32+1 || current.Revision.MaxAge != time.Hour ||
		!m.s.trustOrigin.olderIgnored {
		t.Errorf("a copied-back older ledger %+v %+v", st, current)
	}
}

func TestTheIgnoredOlderRecordStillTightensTheDefault(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "10")
	if got := f.trustActivate(p.c, id); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.s.Close()
	marker := filepath.Join(f.s.cfg.Data, decisionledger.MarkerName)
	f.must(os.WriteFile(marker, []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"x","g":3}`), 0o600))
	g := f.restart(nil)
	current, _ := g.s.policy.Policy(SampleWorkspace)
	trust := g.trustNow(t)
	if current.Revision.MaxAge != time.Hour || len(current.Revision.StepUp) != 1 || current.Revision.StepUp[0].MaxAge != 10*time.Minute || current.Epoch != 3<<32+1 ||
		fmt.Sprint(trust["stricter_fields"]) != "[max_age step_up]" {
		t.Errorf("the stricter of the ignored record and the default %+v %v", current.Revision, trust["stricter_fields"])
	}
	base := trustchange.Candidate{Methods: []string{"oidc", "password"}, MFA: authpolicy.MFANone, MaxAge: 2 * time.Hour,
		StepUp: []trustchange.Rule{{Action: "a", MaxAge: time.Hour}, {Action: "b", MaxAge: 30 * time.Minute}}}
	other := trustchange.Candidate{Methods: []string{"saml"}, MFA: authpolicy.MFAEveryone, MaxAge: 3 * time.Hour,
		StepUp: []trustchange.Rule{{Action: "a", MaxAge: 2 * time.Hour, MFA: true}}}
	out, fields := stricter(base, other)
	if fmt.Sprint(out.Methods) != "[saml]" || out.MFA != authpolicy.MFAEveryone || out.MaxAge != 2*time.Hour || fmt.Sprint(out.StepUp) != "[{a 1h0m0s true} {b 30m0s false}]" ||
		fmt.Sprint(fields) != "[methods mfa step_up]" {
		t.Errorf("stricter %+v %v", out, fields)
	}
	same, none := stricter(base, base)
	if fmt.Sprint(same) != fmt.Sprint(base) || len(none) != 0 {
		t.Errorf("identical %+v %v", same, none)
	}
}

func TestWithoutTheSampleSeedTheIgnoredRecordStillApplies(t *testing.T) {
	f := setup(t, SeedNone).ready()
	g0 := f.s.ledger.Status().Generation
	hex := func(c string) string { return strings.Repeat(c, 64) }
	epoch := g0<<32 + 2
	cfg := decisionledger.Config{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAgeSeconds: 3600,
		StepUp: []decisionledger.StepRule{{Action: "policy.change", MaxAgeSeconds: 600}}}
	if _, err := f.s.ledger.Commit(decisionledger.Decision{Workspace: EmptyWorkspace, Decision: decisionledger.TrustActivation, Trust: &decisionledger.Trust{
		Change: "trust-0a1b2c3d-1", ChangeRevision: 1, ChangeDigest: hex("c"), EvidenceDigest: hex("e"), OldEpoch: epoch - 1, NewEpoch: epoch, OldPolicy: 1, NewPolicy: 2,
		OldDigest: hex("0"), NewDigest: hex("1"), Config: cfg, ProposedBy: "user-a", ValidatedBy: "user-b", ApprovedBy: "user-b", ActivatedBy: "user-c",
		ApprovedAt: f.s.startedAt, RunStartedAt: f.s.startedAt, InvalidateBelow: epoch, AffectedSessions: 1, Origin: decisionledger.OriginActivation}}); err != nil {
		t.Fatal(err)
	}
	f.s.Close()
	marker := filepath.Join(f.s.cfg.Data, decisionledger.MarkerName)
	f.must(os.WriteFile(marker, []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"x","g":`+fmt.Sprint(g0+2)+`}`), 0o600))
	g := f.restart(nil)
	current, err := g.s.policy.Policy(EmptyWorkspace)
	o := g.s.trustOrigin
	if err != nil || current.Revision.MaxAge != time.Hour || len(current.Revision.StepUp) != 1 || current.Revision.StepUp[0].MaxAge != 10*time.Minute ||
		current.Epoch != (g0+2)<<32+1 || current.Number != 1 || o.origin != originSeed || !o.olderIgnored || fmt.Sprint(o.stricter) != "[methods max_age step_up]" ||
		fmt.Sprint(current.Revision.Methods) != "[password]" || current.Revision.MFA != authpolicy.MFANone {
		t.Errorf("the ignored record without the sample seed %v %+v %+v", err, current, o)
	}
}

func TestStatedGenerationBounds(t *testing.T) {
	f := setup(t, SeedSample)
	f.s.cfg.Access.AcceptTrustReset = true
	unusable := decisionledger.Prior{}
	for _, c := range []struct {
		n  uint64
		ok bool
	}{{decisionledger.MaxGeneration - 1, true}, {decisionledger.MaxGeneration, false}, {decisionledger.MaxGeneration + 1, false}, {^uint64(0), false}} {
		f.s.cfg.Access.TrustGeneration = c.n
		next, err := f.s.trustSuccessor(unusable)
		if (err == nil) != c.ok || c.ok && next.Generation != c.n || !c.ok && !strings.Contains(err.Error(), "trust-generation must be below") {
			t.Errorf("stated %d: %+v %v", c.n, next, err)
		}
	}
	f.s.cfg.Access.TrustGeneration = 0
	top := decisionledger.Prior{MarkerUsable: true, MarkerGeneration: decisionledger.MaxGeneration - 1}
	if _, err := f.s.trustSuccessor(top); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "exhausted") {
		t.Errorf("the last generation %v", err)
	}
}

func TestPendingAdministratorDoesNotQualify(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	f.blockedBy(t, "an administrator whose membership is only invited", p, func() {
		if got := f.run(p.a, "identity.remove_membership", SampleAdmin3, nil); got.status != 200 {
			t.Fatal(got.raw)
		}
		if got := f.run(p.a, "identity.invite", SampleAdmin3, map[string]string{"role": "administrator"}); got.status != 200 {
			t.Fatal(got.raw)
		}
	}, itemRecovery, "three_distinct_people")
}

func TestConcurrentActivationLoserGetsTheRevisionConflict(t *testing.T) {
	for round := 0; round < 10; round++ {
		f := setup(t, SeedSample).ready()
		p := f.trustAdmins()
		one, two := f.approvedChange(p, "60", "0"), f.approvedChange(p, "90", "0")
		bodies := []string{}
		actors := []actor{}
		for range 2 {
			a := f.signedIn(Admin3Login, pw)
			v := a.verifier(f)
			f.s.sessions.mu.Lock()
			f.s.sessions.byID[v].trustEpoch = ^uint64(0)
			f.s.sessions.mu.Unlock()
			actors = append(actors, a)
		}
		revision := f.revision(p.c)
		for i, id := range []string{one, two} {
			raw, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("c%d", i), "target_id": id, "idempotency_key": fmt.Sprintf("c%d", i),
				"expected_revision": revision, "rationale": "go", "body": f.decisionFields(id)})
			bodies = append(bodies, string(raw))
		}
		results := make([]reply, 2)
		var wg sync.WaitGroup
		for i := range bodies {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				results[i] = f.call("trust.activate", bodies[i], actors[i].cookie, actors[i].csrf)
			}(i)
		}
		wg.Wait()
		codes := []string{}
		for _, r := range results {
			codes = append(codes, strconv.Itoa(r.status)+":"+fmt.Sprint(r.body["code"]))
		}
		if !(codes[0] == "200:<nil>" && codes[1] == "409:revision_conflict" || codes[1] == "200:<nil>" && codes[0] == "409:revision_conflict") {
			t.Fatalf("round %d: %v", round, codes)
		}
	}
}

func TestM03RefusesTrustRights(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for _, right := range trustRights {
		got := f.requestGrant(root, SampleMember, map[string]string{"action": string(right), "kind": gate.InstallationKind, "fields": ""})
		f.code(t, string(right), got, 422, string(commands.InvalidRequest))
		if fmt.Sprint(got.body["fields"]) != "[action]" {
			t.Errorf("%s fields %v", right, got.body["fields"])
		}
	}
	strip := func(r reply) string {
		delete(r.body, "request_id")
		raw, _ := json.Marshal(r.body)
		return strconv.Itoa(r.status) + string(raw)
	}
	unknown := strip(f.run(root, "access.request_revoke", "grant-unknown", nil))
	for _, id := range []string{"trust-trust_approve-" + SampleAdmin2, "installation-" + SampleAdmin2, "trust-trust_propose-console"} {
		if got := strip(f.run(root, "access.request_revoke", id, nil)); got != unknown {
			t.Errorf("%s answers %s; an unknown grant answers %s", id, got, unknown)
		}
	}
}

func TestRecoveryConditionsOnTheirOwn(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	f.blockedBy(t, "a membership that is not active", p, func() {
		if got := f.run(p.a, "identity.suspend_membership", SampleAdmin3, nil); got.status != 200 {
			t.Fatal(got.raw)
		}
	}, itemRecovery, "three_distinct_people")
	g := setup(t, SeedSample).ready()
	g.trustAdmins()
	base, _ := g.s.trustBase(SampleWorkspace)
	v, err := g.s.trust.Propose(trustchange.Proposal{Workspace: SampleWorkspace, Route: trustchange.RouteLocalPassword, Proposer: SampleAdmin, Base: base,
		Candidate: trustchange.Candidate{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFAAdministrators, MaxAge: time.Hour}})
	g.must(err)
	q := admins{g.signedIn(AdminLogin, pw), g.signedIn(Admin2Login, pw), g.signedIn(Admin3Login, pw)}
	g.trustValidate(q.b, v.ID)
	if item := g.validationItems(v.ID)[itemRecovery]; item.Detail != "missing:trust_propose,trust_validate,trust_approve,trust_activate" {
		t.Errorf("not admitted under the candidate %+v", item)
	}
}

func TestHoldersAreReadyOnlyWithARecoveryPath(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.trustAdmins()
	a := f.as(AdminLogin)
	if h := f.policyRead(a).body["holders"].(map[string]any); h["ready"] != true || fmt.Sprint(h["qualifying_people"]) != "3" {
		t.Errorf("three full holders %+v", h)
	}
	for _, user := range []string{SampleAdmin, SampleAdmin2, SampleAdmin3} {
		for _, right := range []gate.Action{gate.TrustPropose, gate.TrustApprove, gate.TrustActivate} {
			g, _ := f.s.grantView("trust-" + string(right) + "-" + user)
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	if h := f.policyRead(a).body["holders"].(map[string]any); h["ready"] != false || fmt.Sprint(h["qualifying_people"]) != "3" ||
		fmt.Sprint(h["missing"]) != "[trust_propose trust_approve trust_activate]" {
		t.Errorf("three validate-only holders %+v", h)
	}
}

func TestStepUpIgnoresTheCandidateWithoutTheValidateRight(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id := changeOf(t, f.trustPropose(f.as(AdminLogin), "5", "0"))
	g, _ := f.s.grantView("trust-" + string(gate.TrustValidate) + "-" + SampleAdmin2)
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	known, _ := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, validateCommand, id)
	unknown, _ := f.s.stepUpTerms(SampleWorkspace, SampleAdmin2, validateCommand, "trust-00000000-9")
	holder, _ := f.s.stepUpTerms(SampleWorkspace, SampleAdmin3, validateCommand, id)
	if known != stepUpDefault || unknown != stepUpDefault || holder != 5*time.Minute {
		t.Errorf("windows %v %v %v", known, unknown, holder)
	}
}

func TestRevalidationWithTheSameEvidenceKeepsTheApproval(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	if got := f.trustValidate(p.a, id); got.status != 200 || f.view(id).State != trustchange.StateApproved || f.view(id).Approver != SampleAdmin2 {
		t.Fatalf("revalidation %s %+v", got.raw, f.view(id))
	}
	if got := f.trustActivate(p.c, id); got.status != 200 {
		t.Errorf("activation after an unchanged revalidation %s", got.raw)
	}
}

func TestTheSecondActivationOnAnEpochIsRefusedWithAFreshRevision(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	one, two := f.approvedChange(p, "60", "0"), f.approvedChange(p, "90", "0")
	stale := f.decisionFields(two)
	if got := f.trustActivate(p.c, one); got.status != 200 {
		t.Fatal(got.raw)
	}
	c := f.signedIn(Admin3Login, pw)
	f.code(t, "the old epoch", f.run(c, "trust.activate", two, stale), 409, string(commands.AuthorityEpochConflict))
	f.code(t, "the superseded change", f.trustActivate(c, two), 422, string(commands.InvalidStage))
}

func TestTrustFieldsAreRequired(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.a, "60", "0"))
	missing := f.run(p.b, "trust.validate", id, map[string]string{"expected_change_revision": "1", "expected_epoch": "1"})
	f.code(t, "no change digest", missing, 422, string(commands.InvalidRequest))
	if fmt.Sprint(missing.body["fields"]) != "[change_digest]" {
		t.Errorf("fields %v", missing.body["fields"])
	}
	f.trustValidate(p.b, id)
	fields := f.decisionFields(id)
	delete(fields, "evidence_digest")
	delete(fields, "expected_policy")
	noEvidence := f.run(p.b, "trust.approve", id, fields)
	f.code(t, "no evidence digest", noEvidence, 422, string(commands.InvalidRequest))
	if fmt.Sprint(noEvidence.body["fields"]) != "[evidence_digest]" {
		t.Errorf("fields %v", noEvidence.body["fields"])
	}
	for _, e := range f.s.commands.Catalogue() {
		if !strings.HasPrefix(e.ID, "trust.") {
			continue
		}
		for _, field := range e.Fields {
			if field.Required != (field.Name != "expected_change_revision" || e.ID != "trust.propose") {
				t.Errorf("%s %s required %v", e.ID, field.Name, field.Required)
			}
		}
	}
}
