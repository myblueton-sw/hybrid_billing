package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

var (
	recoveryAnn  = recovery.Actor{ID: "test-ann", Kind: recovery.KindHuman}
	recoveryBob  = recovery.Actor{ID: "test-bob", Kind: recovery.KindHuman}
	standbyActor = recovery.Actor{ID: recoveryStandbyW, Kind: recovery.KindSystem}
	sampleLost   = recovery.LostDigest(nil)
)

func (f world) recoveryRead(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/recovery", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func sectionRow(t testing.TB, got reply, section, installation string) map[string]any {
	t.Helper()
	return rowWith(t, got, section, "rows", "installation", installation)
}

func hasRow(got reply, section, installation string) bool {
	rows, _ := part(got, section)["rows"].([]any)
	for _, item := range rows {
		if item.(map[string]any)["installation"] == installation {
			return true
		}
	}
	return false
}

func recoveryProposalRowOf(t testing.TB, got reply, number uint64) map[string]any {
	t.Helper()
	list, _ := got.body["proposals"].([]any)
	for _, item := range list {
		if row := item.(map[string]any); fmt.Sprint(row["number"]) == fmt.Sprint(number) {
			return row
		}
	}
	t.Fatalf("no proposal %d in %s", number, got.raw)
	return nil
}

func proposalNumbers(got reply) string {
	out := []string{}
	list, _ := got.body["proposals"].([]any)
	for _, item := range list {
		row := item.(map[string]any)
		out = append(out, fmt.Sprintf("%v:%v:%v", row["number"], row["kind"], row["state"]))
	}
	return strings.Join(out, ",")
}

func recoveryBlocked(row map[string]any) string {
	return fmt.Sprint(row["can_approve"], " ", row["approve_blocked"], " ", row["can_withdraw"], " ", row["withdraw_blocked"])
}

func unlockState(row map[string]any) string {
	return fmt.Sprint(row["mode"], " ", row["phase"], " ", row["can_propose_unlock"], " ", row["unlock_blocked"])
}

func unlockFields(command string) map[string]string {
	if command == recoveryCommand+recoveryUnlockKind {
		return map[string]string{"lost": sampleLost}
	}
	return map[string]string{}
}

func proposalTargetOf(n uint64) string { return fmt.Sprintf("%s%d", recoveryProposalTarget, n) }

func installTargetOf(id string) string { return recoveryInstallTarget + id }

func (f world) statusOf(id string) string {
	f.t.Helper()
	state, generation, stopped, err := f.s.recovery.Status(id)
	if err != nil {
		f.t.Fatal(err)
	}
	return fmt.Sprint(state, " ", generation, " ", stampOrEmpty(stopped))
}

func (f world) isolationOf(got reply, id string) string {
	f.t.Helper()
	row := sectionRow(f.t, got, "isolation", id)
	return fmt.Sprint(row["state"], " ", row["generation"], " ", row["stopped_at"])
}

func (f world) sampleBegin(id string, generation uint64) recovery.Change {
	manifest := recoveryManifest("")
	return recovery.Change{Kind: recovery.ChangeBegin, Begin: recovery.RestorePoint{Installation: id, Generation: generation, Backup: sampleHash("test-backup"),
		ManifestDigest: recovery.ManifestDigest(manifest), AsOf: f.s.clock().Add(-time.Minute), Manifest: manifest}}
}

func (f world) moduleApprove(c recovery.Change) recovery.ProposalView {
	f.t.Helper()
	p, err := f.s.recovery.Propose(c, recoveryAnn)
	if err != nil {
		f.t.Fatal(err)
	}
	if p, err = f.s.recovery.Approve(p.Number, recoveryBob); err != nil {
		f.t.Fatal(err)
	}
	return p
}

func TestRecoveryReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.recoveryRead(root)
	if got.status != 200 || got.body["state"] != "configured" || got.body["now"] != stamp(start) || got.body["revision"] != f.s.Revision() {
		t.Fatalf("read %s", got.raw)
	}
	for section, want := range map[string]string{"sets": "configured", "isolation": "configured", "reconciliation": "configured", "resume": "configured",
		"migration": "not_connected"} {
		if part(got, section)["state"] != want {
			t.Errorf("%s state %v", section, part(got, section)["state"])
		}
	}
	if s := fmt.Sprint(part(got, "migration")["modules"]); s != "[migration balances]" {
		t.Errorf("migration %s", s)
	}
	primary := sectionRow(t, got, "sets", recoveryPrimary)
	open := primary["recovery"].(map[string]any)
	manifest := recoveryManifest("")
	if s := fmt.Sprint(primary["state"], primary["generation"], primary["begin"], primary["abandon"], primary["history"], primary["history_total"]); s != "recovering2not_connectednot_connected[] 0" {
		t.Errorf("primary set %s", s)
	}
	if s := fmt.Sprint(open["state"], open["id"], open["phase"], open["as_of"], open["point_generation"]); s != "openrecovery-1verify2026-11-16T08:00:00Z1" {
		t.Errorf("open recovery %s", s)
	}
	if open["backup_reference"] != sampleHash("sample-backup-1") || open["manifest_digest"] != recovery.ManifestDigest(manifest) {
		t.Errorf("restore point %v", open)
	}
	if s := fmt.Sprint(open["domains"]); s != "[map[domain:database mismatch_chunks:[] status:verified] map[domain:evidence mismatch_chunks:[] status:verified] "+
		"map[domain:keys mismatch_chunks:[] status:verified] map[domain:policies mismatch_chunks:[] status:verified] map[domain:modules mismatch_chunks:[] status:verified] "+
		"map[domain:numbering mismatch_chunks:[] status:verified] map[domain:submissions mismatch_chunks:[] status:verified] map[domain:outbox mismatch_chunks:[1] status:mismatch] "+
		"map[domain:deletions mismatch_chunks:[] status:verified]]" {
		t.Errorf("domains %s", s)
	}
	if s := fmt.Sprint(sectionRow(t, got, "sets", recoveryStandby)["recovery"]); s != "map[as_of: backup_reference: domains:[] id: manifest_digest: phase: point_generation:0 state:none]" {
		t.Errorf("standby set %s", s)
	}
	iso := sectionRow(t, got, "isolation", recoveryPrimary)
	if s := fmt.Sprint(iso); s != "map[can_lock:false generation:2 installation:sample-primary lock_blocked:recovering lock_history:map[next: total:0] locks:[] open_calls:[map[deadline:2026-11-16T09:01:00Z "+
		"id:call-2 key:sample-invoice-2 number:2 series:sample-invoice-series writer:sample-erp-writer]] open_calls_next: open_calls_source:recovery open_calls_total:1 other_scope_present:false "+
		"permits:not_shown state:recovering stopped_at:2026-11-16T09:00:00Z verifier:sample-verifier writers:[sample-erp-writer sample-issuance-writer]]" {
		t.Errorf("primary isolation %s", s)
	}
	if s := fmt.Sprint(sectionRow(t, got, "isolation", recoveryStandby)); s != "map[can_lock:false generation:2 installation:sample-standby lock_blocked:locked lock_history:map[next: total:1] "+
		"locks:[map[at:2026-11-16T09:00:00Z by:sample-admin by_login:admin@example.invalid number:1 rationale:Sample data]] open_calls:[] open_calls_next: open_calls_source:not_shown open_calls_total:0 "+
		"other_scope_present:false permits:not_shown state:locked stopped_at:2026-11-16T09:00:00Z verifier: writers:[sample-standby-writer]]" {
		t.Errorf("standby isolation %s", s)
	}
	if s := fmt.Sprint(sectionRow(t, got, "reconciliation", recoveryPrimary)); s != "map[externals:[map[id:sample-erp-ledger kind:erp status:pending] "+
		"map[id:sample-issuance-register kind:issuance status:pending]] installation:sample-primary other_scope_present:false reconciler:not_connected skipped:0 "+
		"unknown_next: unknown_targets:[map[fenced:true high_water:2 key:sample-invoice-2 number:2 series:sample-invoice-series]] unknown_total:1]" {
		t.Errorf("reconciliation %s", s)
	}
	if hasRow(got, "reconciliation", recoveryStandby) {
		t.Error("standby has no recovery to reconcile")
	}
	resume := sectionRow(t, got, "resume", recoveryPrimary)
	if s := fmt.Sprint(unlockState(resume), resume["pending_events"], resume["pending_total"], resume["lost"], resume["lost_digest"] == sampleLost); s !=
		"recovery_resume verify false incomplete[map[at:2026-11-16T09:00:00Z domain:evidence kind:delete sequence:1] map[at:2026-11-16T09:00:00Z domain:database "+
			"kind:revoke_access sequence:2]] 2 [] true" {
		t.Errorf("primary resume %s", s)
	}
	if s := unlockState(sectionRow(t, got, "resume", recoveryStandby)); s != "lock_release not_recovering false already_proposed" {
		t.Errorf("standby resume %s", s)
	}
	if s := proposalNumbers(got); s != "10:unlock:pending,9:writer:applied,8:installation:applied,7:begin:applied,6:external:applied,5:external:applied,4:verifier:applied,"+
		"3:writer:applied,2:writer:applied,1:installation:applied" {
		t.Errorf("proposals %s", s)
	}
	unlock := recoveryProposalRowOf(t, got, 10)
	if s := fmt.Sprint(recoveryBlocked(unlock), " ", unlock["proposer_login"], " ", unlock["target"], " ", unlock["installation"], " ", unlock["summary"]); s != "true  false not_proposer "+
		"admin2@example.invalid recovery-proposal-10 sample-standby map[as_of: external: external_kind: installation:sample-standby lost_digest:"+sampleLost+
		" manifest_digest: point_generation:0 skew_seconds:0 unlock_generation:2 verifier: writer:]" {
		t.Errorf("unlock proposal %s", s)
	}
	begin := recoveryProposalRowOf(t, got, 7)
	if s := fmt.Sprint(recoveryBlocked(begin), " ", begin["closed_by_login"], " ", begin["closing_rationale"], " ", begin["summary"]); s != "false not_pending false not_pending "+
		"admin2@example.invalid Sample data map[as_of:2026-11-16T08:00:00Z external: external_kind: installation:sample-primary lost_digest: manifest_digest:"+
		recovery.ManifestDigest(manifest)+" point_generation:1 skew_seconds:0 unlock_generation:0 verifier: writer:]" {
		t.Errorf("begin proposal %s", s)
	}
	if s := fmt.Sprint(recoveryProposalRowOf(t, got, 1)["summary"].(map[string]any)["skew_seconds"], recoveryProposalRowOf(t, got, 6)["summary"].(map[string]any)["external_kind"],
		recoveryProposalRowOf(t, got, 4)["summary"].(map[string]any)["verifier"], recoveryProposalRowOf(t, got, 3)["summary"].(map[string]any)["writer"]); s !=
		"60issuancesample-verifiersample-issuance-writer" {
		t.Errorf("summaries %s", s)
	}
	checks := 0
	f.s.fault = func(step string) bool {
		if step == "recovery-check" {
			checks++
		}
		return false
	}
	f.recoveryRead(root)
	f.s.fault = nil
	if checks != 2 {
		t.Errorf("the read checked %d changes; only the pending unlock and the primary resume need a check", checks)
	}
	second := f.recoveryRead(f.as(Admin2Login))
	if s := recoveryBlocked(recoveryProposalRowOf(t, second, 10)); s != "false own_proposal true " {
		t.Errorf("proposer view %s", s)
	}
	for _, id := range []string{recoveryPrimary, recoveryStandby} {
		if f.isolationOf(got, id) != f.statusOf(id) {
			t.Errorf("%s isolation %s status %s", id, f.isolationOf(got, id), f.statusOf(id))
		}
	}
	v, err := f.s.recovery.Recovery(recoveryPrimary)
	if err != nil || fmt.Sprint(v.State, " ", v.Generation, " ", stamp(v.StoppedAt)) != f.isolationOf(got, recoveryPrimary) {
		t.Errorf("recovery view %+v %v", v, err)
	}
}

func TestRecoveryReadWithoutSeed(t *testing.T) {
	none := setup(t, SeedNone).ready()
	tok := none.links()[""]
	if got := none.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, none.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := none.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	a := actor{cookie, csrf}
	got := none.recoveryRead(a)
	if got.status != 200 || got.raw != `{"revision":"`+none.s.Revision()+`","now":"2026-11-16T09:00:00Z","state":"not_configured"}`+"\n" {
		t.Fatalf("empty %q", got.raw)
	}
	for _, c := range []struct{ command, target string }{{"recovery.lock", installTargetOf(recoveryStandby)}, {"recovery.approve", proposalTargetOf(1)}} {
		none.code(t, c.command, none.runWith(a, none.revision(a), c.command, c.target, unlockFields(c.command)), 403, "permission_denied")
	}
}

func TestRecoveryAccess(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, root := f.as(MemberLogin), f.as(AdminLogin)
	f.code(t, "member read", f.recoveryRead(member), 403, "permission_denied")
	f.code(t, "anonymous", f.recoveryRead(actor{}), 401, "unauthenticated")
	targets := map[string]string{recoveryApproveKind: proposalTargetOf(10), recoveryWithdrawKind: proposalTargetOf(10), recoveryLockKind: installTargetOf(recoveryStandby),
		recoveryUnlockKind: installTargetOf(recoveryStandby)}
	before := f.s.Revision()
	for kind, target := range targets {
		f.code(t, "member "+kind, f.runWith(member, f.revision(root), recoveryCommand+kind, target, unlockFields(recoveryCommand+kind)), 403, "permission_denied")
		h := recoveryHandler{s: f.s, kind: kind}
		if _, res := h.Resolve("other-ws", target); res != commands.NotFound {
			t.Errorf("%s resolved in another workspace", kind)
		}
		if _, res := h.Resolve(SampleWorkspace, target); res != commands.Found {
			t.Errorf("%s did not resolve %s", kind, target)
		}
	}
	f.s.bootWorkspace = "elsewhere"
	if got := f.recoveryRead(root); got.status != 200 || got.body["state"] != "not_configured" || got.body["sets"] != nil {
		t.Errorf("other workspace %s", got.raw)
	}
	for kind, target := range targets {
		f.code(t, "other workspace "+kind, f.runWith(root, f.s.Revision(), recoveryCommand+kind, target, unlockFields(recoveryCommand+kind)), 403, "permission_denied")
	}
	f.s.bootWorkspace = SampleWorkspace
	for _, c := range []struct{ command, target string }{
		{"recovery.approve", "recovery-proposal-010"}, {"recovery.approve", "recovery-proposal-x"},
		{"recovery.approve", "recovery-proposal-99"}, {"recovery.approve", "10"}, {"recovery.withdraw", "recovery-proposal-"},
		{"recovery.approve", "recovery-proposal-18446744073709551616"}, {"recovery.approve", installTargetOf(recoveryStandby)},
		{"recovery.lock", "recovery-installation:sample-absent"}, {"recovery.lock", recoveryStandby}, {"recovery.lock", "recovery-writer:" + recoveryStandbyW},
		{"recovery.lock", proposalTargetOf(10)}, {"recovery.propose_unlock", "recovery-installation:"},
	} {
		f.code(t, c.command+" "+c.target, f.run(root, c.command, c.target, unlockFields(c.command)), 403, "permission_denied")
	}
	if f.s.Revision() != before {
		t.Errorf("refused commands moved the revision %s %s", before, f.s.Revision())
	}
	long := strings.Repeat("r", maxRationale+1)
	f.unchanged(t, "long reason", f.admin(root, "recovery.approve", proposalTargetOf(10), long, map[string]string{}), "invalid_request")
	f.unchanged(t, "no reason", f.admin(root, "recovery.approve", proposalTargetOf(10), "", map[string]string{}), "invalid_request")
	f.unchanged(t, "no lost", f.run(root, "recovery.propose_unlock", installTargetOf(recoveryStandby), nil), "invalid_request")
}

func TestRecoveryOtherTenantIsHidden(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	first, second := recovery.Actor{ID: SampleAdmin, Kind: recovery.KindHuman}, recovery.Actor{ID: SampleAdmin2, Kind: recovery.KindHuman}
	e, err := f.s.recoveryPropose(recovery.Change{Kind: recovery.ChangeInstallation, Installation: recovery.Installation{ID: "tenant-x", Tenant: "other-tenant"}}, first, "x")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.recoveryApprove(e, second, "x"); err != nil {
		t.Fatal(err)
	}
	if err := f.s.recoveryLock(f.s.rec.install("tenant-x"), first, "x"); err != nil {
		t.Fatal(err)
	}
	pending, err := f.s.recoveryPropose(recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: "tenant-x", Generation: 2, Lost: sampleLost}}, second, "x")
	if err != nil {
		t.Fatal(err)
	}
	got := f.recoveryRead(root)
	if strings.Contains(got.raw, "tenant-x") || strings.Contains(got.raw, "other-tenant") || got.status != 200 {
		t.Errorf("other tenant shown %s", got.raw)
	}
	for _, c := range []struct{ command, target string }{{"recovery.lock", installTargetOf("tenant-x")}, {"recovery.propose_unlock", installTargetOf("tenant-x")},
		{"recovery.approve", proposalTargetOf(e.view.Number)}, {"recovery.approve", proposalTargetOf(pending.view.Number)}, {"recovery.withdraw", proposalTargetOf(pending.view.Number)}} {
		f.code(t, c.command+" "+c.target, f.run(root, c.command, c.target, unlockFields(c.command)), 403, "permission_denied")
	}
	hidden := f.readAs(root, "/api/v1/recovery/locks?installation=tenant-x")
	absent := f.readAs(root, "/api/v1/recovery/locks?installation=no-such-installation")
	if hidden.status != 422 || absent.status != 422 || strings.Contains(hidden.raw, "tenant-x") || strings.Contains(hidden.raw, "other-tenant") || hidden.body["code"] != absent.body["code"] {
		t.Errorf("another tenant's lock log: %d %s, a missing one: %d", hidden.status, hidden.raw, absent.status)
	}
	if own := f.readAs(root, "/api/v1/recovery/locks?installation="+recoveryStandby+"&limit=1"); own.status != 200 || len(keptNumbers(own.body["locks"])) != 1 {
		t.Errorf("the reader's own installation: %d", own.status)
	}
	f.s.rec.installs = nil
	if got := f.recoveryRead(root); got.body["state"] != "not_configured" {
		t.Errorf("no visible installation %s", got.raw)
	}
}

func TestRecoveryOtherScopeCalls(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "release", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "applied", "")
	tok, err := f.s.recovery.Permit(recoveryStandby, recoveryStandbyW, standbyActor)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []recovery.Target{{Workspace: "other-workspace", Key: "other-key-1", Series: "other-series", Number: 5},
		{Workspace: SampleWorkspace, Key: "sample-delivery-1"}} {
		if _, err := f.s.recovery.Call(recoveryStandby, recoveryStandbyW, tok, target, standbyActor); err != nil {
			t.Fatal(err)
		}
	}
	f.moduleApprove(f.sampleBegin(recoveryStandby, 2))
	got := f.recoveryRead(second)
	iso := sectionRow(t, got, "isolation", recoveryStandby)
	if s := fmt.Sprint(iso["open_calls_total"], iso["other_scope_present"], len(iso["open_calls"].([]any))); s != "1 true 1" {
		t.Errorf("isolation %s %v", s, iso)
	}
	rec := sectionRow(t, got, "reconciliation", recoveryStandby)
	if s := fmt.Sprint(rec["unknown_targets"], rec["unknown_total"], rec["other_scope_present"]); s != "[map[fenced:true high_water:<nil> key:sample-delivery-1 number:0 series:]] 1 true" {
		t.Errorf("reconciliation %s", s)
	}
	for _, word := range []string{"other-workspace", "other-key-1", "other-series"} {
		if strings.Contains(got.raw, word) {
			t.Errorf("read carries %s", word)
		}
	}
	if s := unlockState(sectionRow(t, got, "resume", recoveryStandby)); s != "recovery_resume verify false incomplete" {
		t.Errorf("resume %s", s)
	}
	if f.isolationOf(got, recoveryStandby) != f.statusOf(recoveryStandby) {
		t.Errorf("status %s", f.statusOf(recoveryStandby))
	}
}

func TestRecoveryClockMovesCallsWithoutRevision(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	before := f.recoveryRead(root)
	revision := f.s.Revision()
	f.advance(2 * time.Minute)
	after := f.recoveryRead(root)
	iso := sectionRow(t, after, "isolation", recoveryPrimary)
	if s := fmt.Sprint(iso["open_calls"], iso["open_calls_total"]); s != "[] 0" || len(sectionRow(t, before, "isolation", recoveryPrimary)["open_calls"].([]any)) != 1 {
		t.Errorf("open calls after the deadline %s", s)
	}
	rec := sectionRow(t, after, "reconciliation", recoveryPrimary)
	if s := fmt.Sprint(rec["unknown_targets"]); s != "[map[fenced:true high_water:2 key:sample-invoice-2 number:2 series:sample-invoice-series]]" {
		t.Errorf("unknown after the deadline %s", s)
	}
	if f.s.Revision() != revision || after.body["revision"] != before.body["revision"] {
		t.Errorf("revision moved %s %s", revision, f.s.Revision())
	}
}

func TestRecoveryReadLeavesProposalsUnchanged(t *testing.T) {
	f := setup(t, SeedSample).ready()
	views := func() string {
		out := ""
		for n := uint64(1); n <= 10; n++ {
			v, err := f.s.recovery.Proposal(n)
			out += fmt.Sprint(v, err)
		}
		return out
	}
	before := views()
	for _, login := range []string{AdminLogin, Admin2Login} {
		if got := f.recoveryRead(f.as(login)); got.status != 200 {
			t.Fatal(got.raw)
		}
	}
	if views() != before {
		t.Error("the read changed a proposal")
	}
}

func TestRecoveryReadyResumeAndKindCoverage(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	first, other := recovery.Actor{ID: SampleAdmin, Kind: recovery.KindHuman}, recovery.Actor{ID: SampleAdmin2, Kind: recovery.KindHuman}
	register := func(c recovery.Change) {
		t.Helper()
		e, err := f.s.recoveryPropose(c, first, "test")
		if err != nil {
			t.Fatal(err)
		}
		if err := f.s.recoveryApprove(e, other, "test"); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"test-ready", "test-coverage"} {
		register(recovery.Change{Kind: recovery.ChangeInstallation, Installation: recovery.Installation{ID: id, Tenant: SampleWorkspace}})
		register(recovery.Change{Kind: recovery.ChangeVerifier, Verifier: recovery.Verifier{Installation: id, ID: id + "-verifier"}})
	}
	register(recovery.Change{Kind: recovery.ChangeWriter, Writer: recovery.Writer{Installation: "test-coverage", ID: "test-coverage-writer", Kind: recovery.WriterERP, MaxCall: time.Minute}})
	for _, id := range []string{"test-ready", "test-coverage"} {
		f.moduleApprove(f.sampleBegin(id, 1))
		for _, inv := range recoveryManifest("") {
			if _, err := f.s.recovery.Report(id, inv, recovery.Actor{ID: id + "-verifier", Kind: recovery.KindSystem}); err != nil {
				t.Fatal(err)
			}
		}
	}
	got := f.recoveryRead(root)
	if s := unlockState(sectionRow(t, got, "resume", "test-ready")) + "|" + unlockState(sectionRow(t, got, "resume", "test-coverage")); s !=
		"recovery_resume ready true |recovery_resume ready false kind_coverage" {
		t.Errorf("resume %s", s)
	}
	f.unchanged(t, "wrong lost", f.run(root, "recovery.propose_unlock", installTargetOf("test-ready"), map[string]string{"lost": sampleHash("other")}), "invalid_stage")
	f.unchanged(t, "kind coverage", f.run(root, "recovery.propose_unlock", installTargetOf("test-coverage"), map[string]string{"lost": sampleLost}), "invalid_stage")
	proposed := f.run(root, "recovery.propose_unlock", installTargetOf("test-ready"), map[string]string{"lost": sampleLost})
	f.ok(t, "propose resume", proposed, "pending", "")
	number := proposed.body["fields"].(map[string]any)["proposal"].(string)
	f.ok(t, "approve resume", f.run(second, "recovery.approve", recoveryProposalTarget+number, nil), "applied", "")
	if s := f.statusOf("test-ready"); s != "serving 2 " {
		t.Errorf("resumed %s", s)
	}
	after := sectionRow(t, f.recoveryRead(root), "sets", "test-ready")
	if s := fmt.Sprint(after["recovery"].(map[string]any)["state"], after["history"], after["history_total"]); s != "none[map[generation:2 lost_count:0 manifest_digest:"+
		recovery.ManifestDigest(recoveryManifest(""))+" outcome:completed recovery:recovery-1]] 1" {
		t.Errorf("history %s", s)
	}
}
