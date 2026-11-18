package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func TestTrustInputsAndDependencies(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := changeOf(t, f.trustPropose(p.a, "60", "10"))
	for name, c := range map[string]struct {
		command string
		fields  map[string]string
		status  int
		code    commands.Code
	}{
		"a negative epoch":       {"trust.propose", map[string]string{"max_age_minutes": "60", "step_up_minutes": "0", "expected_epoch": "-1", "expected_policy": "1"}, 422, commands.InvalidRequest},
		"a negative policy":      {"trust.propose", map[string]string{"max_age_minutes": "60", "step_up_minutes": "0", "expected_epoch": "1", "expected_policy": "-1"}, 422, commands.InvalidRequest},
		"a negative revision":    {"trust.cancel", map[string]string{"expected_change_revision": "-1"}, 422, commands.InvalidRequest},
		"validate revision":      {"trust.validate", map[string]string{"expected_change_revision": "-1", "change_digest": "x", "expected_epoch": "1"}, 422, commands.InvalidRequest},
		"validate another epoch": {"trust.validate", map[string]string{"expected_change_revision": "1", "change_digest": "x", "expected_epoch": "2"}, 409, commands.AuthorityEpochConflict},
		"approve another epoch":  {"trust.approve", map[string]string{"change_digest": "x", "evidence_digest": "y", "expected_epoch": "2"}, 409, commands.AuthorityEpochConflict},
	} {
		target := id
		if c.command == "trust.propose" {
			target = trustTarget
		}
		f.code(t, name, f.run(p.a, c.command, target, c.fields), c.status, string(c.code))
	}
	v := f.view(id)
	revision := f.revision(p.b)
	prepared := func(command string, fields map[string]string) string {
		raw, _ := json.Marshal(map[string]any{"request_id": "q-" + command, "target_id": id, "idempotency_key": "q-" + command, "expected_revision": revision,
			"rationale": "x", "body": fields})
		return string(raw)
	}
	validate := prepared("trust.validate", map[string]string{"expected_change_revision": "1", "change_digest": v.ChangeDigest, "expected_epoch": "1"})
	approve := prepared("trust.approve", map[string]string{"change_digest": v.ChangeDigest, "evidence_digest": "x", "expected_epoch": "1"})
	f.s.fault = func(at string) bool { return at == "roster" }
	f.code(t, "validate without a roster", f.call("trust.validate", validate, p.b.cookie, p.b.csrf), 503, string(commands.DependencyUnavailable))
	f.code(t, "approve without a roster", f.call("trust.approve", approve, p.b.cookie, p.b.csrf), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	f.publishDirect(authpolicy.Revision{MaxAge: 90 * time.Minute})
	p = admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
	f.code(t, "a change on an old epoch", f.trustValidate(p.b, id), 409, string(commands.AuthorityEpochConflict))
	f.s.domain.Lock()
	_, err := (trustCommand{s: f.s, kind: "validate"}).Apply(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin2, Target: id, Rationale: "x",
		Fields: map[string]string{"expected_change_revision": "1", "change_digest": v.ChangeDigest, "expected_epoch": f.epochText()}})
	f.s.domain.Unlock()
	if failure, ok := err.(*commands.Failure); !ok || failure.Code != commands.PermissionDenied {
		t.Errorf("a validation without the caller's session %v", err)
	}
	f.s.domain.Lock()
	f.s.caller = p.b.verifier(f)
	_, err = (trustCommand{s: f.s, kind: "validate"}).Apply(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin, Target: id, Rationale: "x",
		Fields: map[string]string{"expected_change_revision": "1", "change_digest": v.ChangeDigest, "expected_epoch": f.epochText()}})
	f.s.caller = identity.Verifier{}
	f.s.domain.Unlock()
	if failure, ok := err.(*commands.Failure); !ok || failure.Code != commands.PermissionDenied {
		t.Errorf("a validation by another user's session %v", err)
	}
	orphan := f.signedIn(Admin2Login, pw).verifier(f)
	f.s.sessions.drop(orphan)
	f.s.domain.Lock()
	f.s.caller = orphan
	_, err = (trustCommand{s: f.s, kind: "validate"}).Apply(commands.Context{Workspace: SampleWorkspace, User: SampleAdmin2, Target: id, Rationale: "x",
		Fields: map[string]string{"expected_change_revision": "1", "change_digest": v.ChangeDigest, "expected_epoch": f.epochText()}})
	f.s.caller = identity.Verifier{}
	f.s.domain.Unlock()
	if failure, ok := err.(*commands.Failure); !ok || failure.Code != commands.PermissionDenied {
		t.Errorf("an identity session without a console session %v", err)
	}
}

func TestAnEpochOfAnotherGenerationIsRefusedByTheLedger(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	f.trustAdmins()
	if _, err := f.s.policy.Publish(authpolicy.Publication{Revision: revisionFor(SampleWorkspace, trustchange.Candidate{Methods: []string{authpolicy.MethodPassword},
		MFA: authpolicy.MFANone, MaxAge: seedPolicyAge}), Epoch: 5<<32 + 1, Number: 2}); err != nil {
		t.Fatal(err)
	}
	p := admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
	id := f.approvedChange(p, "60", "0")
	f.code(t, "refused", f.trustActivate(p.c, id), 503, string(commands.DependencyUnavailable))
	if f.view(id).State != trustchange.StateApproved || !strings.Contains(logs.String(), "trust_activation_refused") || f.s.held() {
		t.Errorf("a refused commit %+v", f.view(id))
	}
}

func TestReadEdgeRows(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	first := f.approvedChange(p, "60", "10")
	second := changeOf(t, f.trustPropose(p.b, "90", "0"))
	if got := f.trustValidate(p.c, second); got.status != 200 {
		t.Fatal(got.raw)
	}
	g, _ := f.s.grantView("trust-trust_validate-console")
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	row := f.policyRead(p.a).body["changes"].([]any)[0].(map[string]any)
	if row["id"] != second || row["why_not"].(map[string]any)["approve"] != "evidence_stale" {
		t.Errorf("approver view of stale evidence %+v", row["why_not"])
	}
	h := setup(t, SeedSample).ready()
	q := h.trustAdmins()
	one := h.approvedChange(q, "60", "10")
	two := changeOf(t, h.trustPropose(q.b, "90", "0"))
	if got := h.trustActivate(q.c, one); got.status != 200 {
		t.Fatal(got.raw)
	}
	page := h.policyRead(h.signedIn(AdminLogin, pw)).body
	for _, raw := range page["changes"].([]any) {
		if r := raw.(map[string]any); r["id"] == two && (r["state"] != trustchange.StateSuperseded || r["superseded_by"] != one ||
			r["superseded_epoch"].(map[string]any)["value"] != "2") {
			t.Errorf("superseded row %+v", r)
		}
	}
	k := h.restart(nil)
	if current, _ := k.s.policy.Policy(SampleWorkspace); len(current.Revision.StepUp) != 1 || current.Revision.StepUp[0].MaxAge != 10*time.Minute {
		t.Errorf("a replayed step-up rule %+v", current.Revision)
	}
	c := rowContext{}
	facts := trustFacts{members: 150}
	for i := 0; i < 150; i++ {
		facts.affected = append(facts.affected, "user-"+strconv.Itoa(i))
	}
	if got := c.impact(trustchange.View{}, facts); len(got.AffectedIDs) != maxAffectedShown || got.Members != 150 {
		t.Errorf("impact %+v", got)
	}
	_ = first
}

func TestEvidenceEdgeItems(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	base, _ := f.s.trustBase(SampleWorkspace)
	rules := []trustchange.Rule{}
	for i := 0; i < 60; i++ {
		rules = append(rules, trustchange.Rule{Action: "action-" + strings.Repeat("x", 30) + strconv.Itoa(100+i), MaxAge: time.Hour})
	}
	big, err := f.s.trust.Propose(trustchange.Proposal{Workspace: SampleWorkspace, Route: trustchange.RouteLocalPassword, Proposer: SampleAdmin, Base: base,
		Candidate: trustchange.Candidate{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAge: 2 * time.Hour, StepUp: rules}})
	f.must(err)
	if got := f.trustValidate(p.b, big.ID); got.body["business_state"] != trustchange.ResultBlocked || f.validationItems(big.ID)[itemRecord].Reason != "record_limit" {
		t.Errorf("an oversized record %s", got.raw)
	}
	v := f.view(big.ID)
	f.s.domain.Lock()
	f.s.caller = p.b.verifier(f)
	ev, _, err := f.s.evidenceFor(v, authpolicy.Principal{User: SampleAdmin2, Workspace: SampleWorkspace, Role: authpolicy.RoleAdministrator, Kind: authpolicy.KindHuman},
		f.s.clock(), base, authpolicy.Revision{MFA: authpolicy.MFAEveryone})
	f.s.caller = identity.Verifier{}
	f.s.domain.Unlock()
	f.must(err)
	for _, item := range ev.Items {
		if item.Item == itemMFA && item.Reason != "weakens_mfa" {
			t.Errorf("weakening %+v", item)
		}
	}
}

func TestTrustWithoutALedger(t *testing.T) {
	f := setup(t, SeedSample).ready()
	ledger := f.s.ledger
	f.s.ledger = nil
	defer func() { f.s.ledger = ledger }()
	if f.s.generation() != 0 || f.s.trustRecords() != nil || !f.s.trustUnavailable(false) {
		t.Error("no ledger")
	}
	if row := f.s.trustLedger(); row.State != string(decisionledger.StateIsolated) || row.IsolationReason != string(decisionledger.IsolationMissing) {
		t.Errorf("row %+v", row)
	}
}

func TestTrustGrantFaults(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Seed = SeedNone; c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	f.s.fault = func(at string) bool { return at == "trust-grant" }
	if got := f.auth("redeem", `{"token":"`+f.links()[""]+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, f.pre()); got.status != 204 {
		t.Fatal(got.raw)
	}
	if !strings.Contains(logs.String(), "trust grant failed at activation") {
		t.Error("a failed trust grant is not logged")
	}
	g := setup(t, SeedSample)
	g.s.fault = func(at string) bool { return at == "trust-grant" }
	if err := g.s.Start(); !errors.Is(err, errPartial) {
		t.Errorf("a failed sample trust grant %v", err)
	}
}

func TestReplayFailsClosedOnACraftedLedger(t *testing.T) {
	for name, craft := range map[string]func(*decisionledger.Trust){
		"an unknown method": func(t *decisionledger.Trust) {
			t.Config.Methods, t.Change = []string{"carrier-pigeon"}, t.Change+"9"
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.activated(t, "60")
			rec := f.s.ledger.Decisions()[len(f.s.ledger.Decisions())-1].Decision
			next := *rec.Trust
			next.OldEpoch, next.NewEpoch, next.InvalidateBelow, next.OldPolicy, next.NewPolicy = 2, 3, 3, 2, 3
			craft(&next)
			rec.Trust = &next
			if _, err := f.s.ledger.Commit(rec); err != nil {
				t.Fatal(err)
			}
			_, err := f.restartWith(nil)
			if err == nil || !strings.Contains(err.Error(), "request is incomplete") {
				t.Errorf("a crafted ledger was replayed %v", err)
			}
		})
	}
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	rec := f.s.ledger.Decisions()[len(f.s.ledger.Decisions())-1].Decision
	twice := *rec.Trust
	twice.OldEpoch, twice.NewEpoch, twice.InvalidateBelow, twice.OldPolicy, twice.NewPolicy = 2, 3, 3, 2, 3
	rec.Trust = &twice
	var ce *decisionledger.CommitError
	if _, err := f.s.ledger.Commit(rec); !errors.As(err, &ce) || ce.Reason != decisionledger.ReasonCommitFailed {
		t.Errorf("a second activation record for one change %v", err)
	}
	p := decisionledger.Prior{MarkerUsable: true, MarkerID: strings.Repeat("a", 32), Retained: []decisionledger.Retained{{File: "r", GenesisRead: true, Generation: 5}}}
	g := setup(t, SeedSample)
	g.s.cfg.Access.AcceptTrustReset = true
	if next, err := g.s.trustSuccessor(p); err != nil || next.Generation != 7 || !next.TrustReset {
		t.Errorf("a successor above a retained generation %+v %v", next, err)
	}
}
