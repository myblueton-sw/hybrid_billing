package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func (f world) unknownActivation(t *testing.T, fault string) (string, admins) {
	t.Helper()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	f.s.fault = func(step string) bool { return step == fault }
	got := f.trustActivate(p.c, id)
	f.s.fault = nil
	if got.status != 202 || got.body["job_id"] != trustJobPrefix+id || got.body["status"] != "accepted" {
		t.Fatalf("unknown activation %s", got.raw)
	}
	return id, p
}

func TestActivationUnknownHoldsTheWorkspace(t *testing.T) {
	for _, c := range []struct{ fault, reason string }{{"ledger-sync", decisionledger.ReasonCommitUnknown}, {"trust-publish", reasonPublish}} {
		t.Run(c.reason, func(t *testing.T) {
			var logs bytes.Buffer
			f := setupWith(t, func(cfg *Config) { cfg.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
			id, p := f.unknownActivation(t, c.fault)
			for name, a := range map[string]actor{"proposer": p.a, "approver": p.b, "activator": p.c} {
				if _, ok := f.s.sessions.get(a.verifier(f)); ok {
					t.Errorf("the %s session survived", name)
				}
			}
			if v := f.view(id); v.State != trustchange.StateUnknown || v.Unknown != c.reason || v.Activation != nil {
				t.Errorf("change %+v", v)
			}
			current, _ := f.s.policy.Policy(SampleWorkspace)
			if c.reason == decisionledger.ReasonCommitUnknown && (current.Epoch != 1 || current.Revision.MaxAge.Hours() != 12) {
				t.Errorf("an unconfirmed activation was applied %+v", current)
			}
			a := f.signedIn(AdminLogin, pw)
			page := f.policyRead(a).body
			hold := page["trust"].(map[string]any)["hold"].(map[string]any)
			row := page["changes"].([]any)[0].(map[string]any)
			job := row["job"].(map[string]any)
			if hold["reason"] != c.reason || job["state"] != "waiting" || job["reason"] != c.reason || row["can"].(map[string]any)["activate"] != false {
				t.Errorf("read %+v %+v", hold, job)
			}
			f.code(t, "trust commands are frozen", f.trustPropose(a, "45", "0"), 503, string(commands.DependencyUnavailable))
			if !strings.Contains(logs.String(), "trust_hold_entered") || strings.Contains(logs.String(), "sample reason") {
				t.Errorf("logs %s", logs.String())
			}
			g := f.restart(nil)
			after, _ := g.s.policy.Policy(SampleWorkspace)
			if after.Epoch != 2 || after.Revision.MaxAge.Minutes() != 60 || g.s.hold != nil || g.s.trustOrigin.origin != originReplay {
				t.Errorf("the restart did not apply the written record %+v", after)
			}
		})
	}
	f := setup(t, SeedSample).ready()
	f.unknownActivation(t, "ledger-sync")
	f.s.Close()
	data, _ := os.ReadFile(ledgerFile(f))
	lines := bytes.SplitAfter(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	f.must(os.WriteFile(ledgerFile(f), bytes.Join(lines[:len(lines)-1], nil), 0o600))
	g := f.restart(nil)
	if current, _ := g.s.policy.Policy(SampleWorkspace); current.Epoch != 1 {
		t.Errorf("a record that was not written was applied %+v", current)
	}
}

func TestTrustRefusedWhileTheLedgerIsNotUsable(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	draft := changeOf(t, f.trustPropose(p.a, "45", "0"))
	f.stoppedLedger()
	for name, got := range map[string]reply{
		"propose":  f.trustPropose(p.b, "30", "0"),
		"validate": f.trustValidate(p.b, draft),
		"approve":  f.trustApprove(p.b, id),
		"activate": f.trustActivate(p.c, id),
		"reject":   f.trustEnd(p.b, "trust.reject", draft),
		"cancel":   f.trustEnd(p.a, "trust.cancel", draft),
	} {
		f.code(t, "stopped "+name, got, 503, string(commands.DependencyUnavailable))
	}
	row := f.policyRead(p.c).body["changes"].([]any)[1].(map[string]any)
	if row["why_not"].(map[string]any)["activate"] != "trust_unavailable" || row["can"].(map[string]any)["validate"] != false {
		t.Errorf("read %+v", row)
	}
	g := setupWith(t, func(c *Config) { c.Access = DefaultAccessSettings(); c.Access.LedgerMaxMiB = 1 })
	q := g.trustAdmins()
	gid := g.approvedChange(q, "60", "0")
	pad := decisionledger.Decision{Workspace: SampleWorkspace, Decision: decisionledger.RevokeGrant, Target: "grant-pad", Revocation: "revocation-pad",
		RequestedBy: SampleAdmin, EndedDelegations: []string{strings.Repeat("d", 1500)}}
	for g.s.ledger.Status().State == decisionledger.StateOK {
		if _, err := g.s.ledger.Commit(pad); err != nil {
			break
		}
	}
	if g.s.ledger.Status().State != decisionledger.StateFull {
		t.Fatalf("ledger %+v", g.s.ledger.Status())
	}
	g.code(t, "full activate", g.trustActivate(q.c, gid), 503, string(commands.DependencyUnavailable))
	if got := g.trustPropose(q.a, "30", "0"); got.status != 200 {
		t.Errorf("a full ledger refuses activation only %s", got.raw)
	}
	h := setup(t, SeedSample).ready()
	r := h.trustAdmins()
	h.s.Close()
	h.must(os.WriteFile(ledgerFile(h), []byte("x\n"), 0o600))
	k := h.restart(nil)
	a := k.as(AdminLogin)
	k.code(t, "isolated propose", k.trustPropose(a, "30", "0"), 503, string(commands.DependencyUnavailable))
	if st := k.policyRead(a).body["ledger"].(map[string]any); st["state"] != "isolated" || st["isolation_reason"] != "integrity_failed" {
		t.Errorf("ledger %+v", st)
	}
	_ = r
}

func TestEpochOverflowRefusesActivation(t *testing.T) {
	if _, ok := nextEpoch(maxEpochNumber, 0); ok {
		t.Error("the epoch number overflows")
	}
	if e, ok := nextEpoch(5, 2); !ok || e != 2<<32+1 {
		t.Errorf("a newer generation starts at 1: %d", e)
	}
	if e, ok := nextEpoch(maxEpochNumber, 1); !ok || e != 1<<32+1 {
		t.Errorf("an older generation %d", e)
	}
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	_, err := f.s.policy.Publish(authpolicy.Publication{Revision: revisionFor(SampleWorkspace, trustchange.Candidate{Methods: []string{authpolicy.MethodPassword},
		MFA: authpolicy.MFANone, MaxAge: seedPolicyAge}), Epoch: maxEpochNumber, Number: 2})
	f.must(err)
	p = admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
	v := f.view(id)
	f.s.domain.Lock()
	f.s.caller = p.c.verifier(f)
	_, err = f.s.activateTrust(trustchange.Approval{ID: id, Workspace: SampleWorkspace, Actor: SampleAdmin3, ChangeDigest: v.ChangeDigest, EvidenceDigest: v.Validation.Digest,
		Recomputed: v.Validation.Digest, Base: trustchange.Base{Epoch: v.OldEpoch, Policy: v.OldPolicy, Digest: v.OldDigest}}, trustchange.Base{Epoch: maxEpochNumber, Policy: 2}, start)
	f.s.domain.Unlock()
	if failure, ok := err.(*commands.Failure); !ok || failure.Code != commands.DependencyUnavailable || f.view(id).State != trustchange.StateApproved {
		t.Errorf("overflow %v %+v", err, f.view(id))
	}
	g := setup(t, SeedSample).ready()
	q := g.trustAdmins()
	_, err = g.s.policy.Publish(authpolicy.Publication{Revision: revisionFor(SampleWorkspace, trustchange.Candidate{Methods: []string{authpolicy.MethodPassword},
		MFA: authpolicy.MFANone, MaxAge: seedPolicyAge}), Epoch: maxEpochNumber, Number: 2})
	g.must(err)
	q = admins{g.signedIn(AdminLogin, pw), g.signedIn(Admin2Login, pw), g.signedIn(Admin3Login, pw)}
	gid := changeOf(t, g.trustPropose(q.a, "60", "0"))
	if got := g.trustValidate(q.b, gid); got.body["business_state"] != trustchange.ResultBlocked || g.validationItems(gid)[itemRecord].Reason != "epoch_exhausted" {
		t.Errorf("an exhausted generation blocks validation %s", got.raw)
	}
}

func TestOldEpochSessionsAreRefusedEverywhere(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	p := f.trustAdmins()
	id := f.approvedChange(p, "60", "0")
	if r := f.trustActivate(p.c, id); r.status != 200 {
		t.Fatal(r.raw)
	}
	f.setPassword(MemberLogin, pw)
	stale := func() actor {
		a := f.signedIn(AdminLogin, pw)
		v := a.verifier(f)
		f.s.sessions.mu.Lock()
		f.s.sessions.byID[v].trustEpoch = 1
		f.s.sessions.mu.Unlock()
		return a
	}
	paths := map[string]func(a actor) reply{
		"read": func(a actor) reply { return f.policyRead(a) },
		"me":   func(a actor) reply { return f.me(a) },
		"reader": func(a actor) reply {
			return f.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(a.cookie) })
		},
		"command": func(a actor) reply {
			return f.call("identity.suspend_user", `{"request_id":"r","target_id":"x","idempotency_key":"k","expected_revision":"r1","rationale":"x","body":{}}`, a.cookie, a.csrf)
		},
		"reauthenticate": func(a actor) reply { return f.reauth(a, pw) },
	}
	for name, run := range paths {
		a := stale()
		f.code(t, name, run(a), 401, string(commands.Unauthenticated))
		if _, ok := f.s.sessions.get(a.verifier(f)); ok {
			t.Errorf("%s kept the old-epoch session", name)
		}
	}
	a := stale()
	result := f.s.execute("identity.suspend_user", commands.Call{Answer: gate.Answer(identity.Answer{}), Request: "q"}, a.verifier(f))
	if result.Err == nil || result.Err.Code != commands.Unauthenticated || result.Err.Status != 401 {
		t.Errorf("a command queued during activation %+v", result.Err)
	}
	if p := progressOf(trustProgress{ended: 2, remaining: 1}, false); p.State != "in_progress" || p.Items[0].Result != "in_progress" {
		t.Errorf("progress %+v", p)
	}
	if p := progressOf(trustProgress{}, true); p.State != "complete" || p.Items[0].Reason != "server_restarted" {
		t.Errorf("replayed progress %+v", p)
	}
}
