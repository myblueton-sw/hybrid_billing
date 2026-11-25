package server

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

func TestRecoveryReadFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	for _, step := range []string{"recovery-read", "recovery-status", "recovery-recovery", "recovery-history", "recovery-fenced", "recovery-high_water"} {
		at = step
		f.code(t, step, f.recoveryRead(root), 503, "dependency_unavailable")
	}
	at = "recovery-check"
	got := f.recoveryRead(root)
	if got.status != 200 || sectionRow(t, got, "resume", recoveryPrimary)["unlock_blocked"] != "invalid" || recoveryProposalRowOf(t, got, 10)["approve_blocked"] != "invalid" {
		t.Errorf("check fault %s", got.raw)
	}
	at = ""
	if got := f.recoveryRead(root); got.status != 200 {
		t.Errorf("read after faults %s", got.raw)
	}
	f.s.rec.installs = append(f.s.rec.installs, &recoveryInstall{id: "sample-absent", tenant: SampleWorkspace})
	f.code(t, "unknown stored installation", f.recoveryRead(root), 503, "dependency_unavailable")
}

func TestRecoveryCommandFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	for _, c := range []struct{ step, command, target string }{
		{"recovery-approve", "recovery.approve", proposalTargetOf(10)},
		{"recovery-withdraw", "recovery.withdraw", proposalTargetOf(10)},
		{"recovery-status", "recovery.lock", installTargetOf(recoveryStandby)},
		{"recovery-status", "recovery.propose_unlock", installTargetOf(recoveryStandby)},
	} {
		at = c.step
		f.code(t, c.step+" "+c.command, f.run(root, c.command, c.target, unlockFields(c.command)), 500, "internal_error")
	}
	at = ""
	f.ok(t, "release", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "applied", "")
	at = "recovery-lock"
	f.code(t, "lock fault", f.run(root, "recovery.lock", installTargetOf(recoveryStandby), nil), 500, "internal_error")
	if len(f.s.rec.install(recoveryStandby).locks) != 1 || f.statusOf(recoveryStandby) != "serving 2 " {
		t.Errorf("lock fault changed the store %s", f.statusOf(recoveryStandby))
	}
	at = ""
	f.ok(t, "lock", f.run(root, "recovery.lock", installTargetOf(recoveryStandby), nil), "locked", "")
	at = "recovery-propose"
	f.code(t, "propose fault", f.run(root, "recovery.propose_unlock", installTargetOf(recoveryStandby), unlockFields("recovery.propose_unlock")), 500, "internal_error")
	f.s.fault = nil
	if len(f.s.rec.entries) != 10 || f.s.rec.pendingUnlock(recoveryStandby) || f.statusOf(recoveryStandby) != "locked 3 2026-11-16T09:00:00Z" {
		t.Errorf("faults changed the store %d %s", len(f.s.rec.entries), f.statusOf(recoveryStandby))
	}
}

func recoverySteps(t *testing.T) map[string]int {
	t.Helper()
	s := seedOnly(t)
	counts := map[string]int{}
	s.fault = func(name string) bool {
		counts[name]++
		return false
	}
	if err := s.seedRecovery(s.clock()); err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestRecoverySeed(t *testing.T) {
	s := seedOnly(t)
	if err := s.seedRecovery(s.clock()); err != nil {
		t.Fatal(err)
	}
	c := s.rec
	primary, standby := c.install(recoveryPrimary), c.install(recoveryStandby)
	if len(c.entries) != 10 || len(c.installs) != 2 || fmt.Sprint(primary.writers) != "[sample-erp-writer sample-issuance-writer]" || primary.verifier != recoveryVerifier ||
		fmt.Sprint(primary.externals) != "[{sample-primary sample-erp-ledger erp} {sample-primary sample-issuance-register issuance}]" || primary.skew != time.Minute ||
		fmt.Sprint(standby.writers) != "[sample-standby-writer]" || len(standby.locks) != 1 || standby.locks[0].by != SampleAdmin || standby.tenant != SampleWorkspace {
		t.Fatalf("seed store %+v %+v", primary, standby)
	}
	if err := s.seedRecovery(s.clock()); !errors.Is(err, recovery.ErrConflict) {
		t.Errorf("second seed %v", err)
	}
	counts := recoverySteps(t)
	if fmt.Sprint(counts) != "map[recovery-approve:9 recovery-call:2 recovery-done:1 recovery-lock:1 recovery-permit:1 recovery-propose:10 recovery-record:2 recovery-report:9]" {
		t.Errorf("seed calls %v", counts)
	}
	for step, total := range counts {
		for n := 1; n <= total; n++ {
			s := seedOnly(t)
			seen := 0
			s.fault = func(name string) bool {
				if name == step {
					seen++
				}
				return name == step && seen == n
			}
			if err := s.seedRecovery(s.clock()); !errors.Is(err, recovery.ErrInvalid) {
				t.Errorf("%s %d: %v", step, n, err)
			}
		}
	}
	broken := freshInstaller(t)
	broken.fault = func(name string) bool { return name == "recovery-report" }
	if err := broken.Start(); !errors.Is(err, recovery.ErrInvalid) {
		t.Errorf("start with a failing seed %v", err)
	}
}

func TestRecoveryStoreAndTargets(t *testing.T) {
	c := newRecoveryStore()
	c.keep(&recoveryEntry{view: recovery.ProposalView{Number: 1, State: recovery.ProposalPending}})
	for n := uint64(2); n <= installKept+1; n++ {
		c.keep(&recoveryEntry{view: recovery.ProposalView{Number: n, State: recovery.ProposalApplied}})
	}
	if len(c.entries) != installKept || c.entry(1) == nil || c.entry(2) != nil || c.entry(installKept+1) == nil {
		t.Errorf("kept %d", len(c.entries))
	}
	for index := range c.entries {
		c.entries[index].view.State = recovery.ProposalPending
	}
	c.keep(&recoveryEntry{view: recovery.ProposalView{Number: 999, State: recovery.ProposalPending}})
	if len(c.entries) != installKept+1 {
		t.Errorf("pending proposals dropped %d", len(c.entries))
	}
	orphan := &recoveryEntry{view: recovery.ProposalView{Change: recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: "nowhere"}}}}
	if c.tenantOf(orphan) != "" || c.install("nowhere") != nil {
		t.Error("orphan proposal has a tenant")
	}
	f := setup(t, SeedSample).ready()
	f.s.recordApplied(recovery.Change{Kind: recovery.ChangeWriter, Writer: recovery.Writer{Installation: "nowhere", ID: "w"}})
	if f.s.rec.install("nowhere") != nil {
		t.Error("an unknown installation was recorded")
	}
	for target, want := range map[string]uint64{proposalTargetOf(1): 1, proposalTargetOf(10): 10} {
		if e := f.s.recoveryEntryOf(target, SampleWorkspace); e == nil || e.view.Number != want {
			t.Errorf("%s: %v", target, e)
		}
	}
	for _, target := range []string{"", recoveryProposalTarget, proposalTargetOf(0), proposalTargetOf(11), "recovery-proposal-+9", "recovery-proposal-09", "recovery-proposal-9x",
		"recovery-proposal-18446744073709551616", "x-recovery-proposal-9", "installation-proposal-9", "plugin-proposal-9"} {
		if f.s.recoveryEntryOf(target, SampleWorkspace) != nil {
			t.Errorf("%s resolved", target)
		}
	}
	if f.s.recoveryEntryOf(proposalTargetOf(1), "other") != nil || f.s.recoveryInstallOf(installTargetOf(recoveryPrimary), "other") != nil ||
		f.s.recoveryInstallOf(recoveryPrimary, SampleWorkspace) != nil || f.s.recoveryInstallOf(installTargetOf(recoveryPrimary), SampleWorkspace) == nil {
		t.Error("installation targets")
	}
	for n := uint64(100); n < 160; n++ {
		f.s.rec.keep(&recoveryEntry{view: recovery.ProposalView{Number: n, State: recovery.ProposalWithdrawn, Proposer: SampleAdmin,
			Change: recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: recoveryStandby}}}})
	}
	for _, i := range f.s.rec.installs {
		for n := 0; n < recoveryShown+1; n++ {
			i.locks = append(i.locks, recoveryLockRecord{by: SampleAdmin, rationale: "x", at: start})
		}
	}
	got := f.recoveryRead(f.as(AdminLogin))
	rows := got.body["proposals"].([]any)
	if n, first, waiting, oldest := listedShape(rows); n != recoveryShown || first != 159 || waiting == 0 || oldest != 110+waiting {
		t.Errorf("page %d first %d waiting %d oldest closed %d", n, first, waiting, oldest)
	}
}

func TestRecoveryRowsAreBounded(t *testing.T) {
	v := recovery.RecoveryView{Phase: recovery.PhaseVerify}
	history := []recovery.History{}
	for n := 0; n < recoveryShown+5; n++ {
		v.OpenCalls = append(v.OpenCalls, recovery.CallView{ID: fmt.Sprint("call-", n), Target: recovery.Target{Workspace: SampleWorkspace, Key: fmt.Sprint("k-", n)}})
		v.Pending = append(v.Pending, recovery.EventView{Sequence: uint64(n + 1)})
		v.Unknown = append(v.Unknown, recovery.Target{Workspace: SampleWorkspace, Key: fmt.Sprint("k-", n)})
		history = append(history, recovery.History{Recovery: fmt.Sprint("recovery-", n)})
	}
	f := setup(t, SeedSample).ready()
	facts := recoveryFacts{install: f.s.rec.install(recoveryPrimary), state: recovery.StateRecovering, view: &v, history: history}
	set, iso, resume := setRow(facts), f.s.isolationRow(facts, SampleWorkspace), f.s.resumeRow(facts)
	row, err := f.s.reconcileRow(facts, SampleWorkspace)
	if len(set.History) != recoveryShown || set.HistoryTotal != recoveryShown+5 || set.History[0].Recovery != "recovery-54" || len(iso.OpenCalls) != recoveryShown ||
		iso.OpenTotal != recoveryShown+5 || len(resume.Pending) != recoveryShown || resume.PendingTotal != recoveryShown+5 || len(row.Unknown) != recoveryShown ||
		row.UnknownTotal != recoveryShown+5 || err != nil {
		t.Errorf("bounds %d %d %d %d %v", len(set.History), len(iso.OpenCalls), len(resume.Pending), len(row.Unknown), err)
	}
}

func TestMapRecoveryErrors(t *testing.T) {
	cases := map[error]commands.Code{
		recovery.Refusal{Reason: recovery.ReasonSelfApproval}:   commands.PrerequisiteFailed,
		recovery.Refusal{Reason: recovery.ReasonNotProposer}:    commands.PrerequisiteFailed,
		recovery.Refusal{Reason: recovery.ReasonAINotAllowed}:   commands.InternalError,
		recovery.Refusal{Reason: recovery.ReasonNotHuman}:       commands.InternalError,
		recovery.Refusal{Reason: recovery.ReasonNotWriter}:      commands.InternalError,
		recovery.Refusal{Reason: recovery.ReasonNotVerifier}:    commands.InternalError,
		recovery.Refusal{Reason: recovery.ReasonNotPending}:     commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonLocked}:         commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonRecovering}:     commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonGeneration}:     commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonNotRecovering}:  commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonOpenCalls}:      commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonNoRestorePoint}: commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonIncomplete}:     commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonLost}:           commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonKindCoverage}:   commands.InvalidStage,
		recovery.Refusal{Reason: recovery.ReasonIdentityTaken}:  commands.InvalidStage,
		recovery.ErrNotFound: commands.InvalidStage,
		recovery.ErrConflict: commands.InvalidStage,
		recovery.ErrLimit:    commands.PrerequisiteFailed,
		recovery.ErrInvalid:  commands.InternalError,
		recovery.ErrRejected: commands.InternalError,
		errors.New("other"):  commands.InternalError,
	}
	for err, want := range cases {
		var failure *commands.Failure
		if !errors.As(mapRecovery(err), &failure) || failure.Code != want || len(failure.Fields) != 0 {
			t.Errorf("%v: %v", err, failure)
		}
	}
	if mapRecovery(nil) != nil {
		t.Error("nil mapped")
	}
	for err, want := range map[error]string{nil: "", recovery.Refusal{Reason: "x"}: "x", recovery.ErrNotFound: "not_found", recovery.ErrLimit: "limit",
		recovery.ErrConflict: "conflict", recovery.ErrInvalid: "invalid", errors.New("other"): "invalid"} {
		if got := checkReason(err); got != want {
			t.Errorf("%v: %q", err, got)
		}
	}
}

func TestRecoveryCommandSet(t *testing.T) {
	f := setup(t, SeedSample).ready()
	ids := []string{}
	for id, mutating := range f.s.mutating {
		if strings.HasPrefix(id, recoveryCommand) {
			if !mutating {
				t.Errorf("%s is not mutating", id)
			}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if strings.Join(ids, " ") != "recovery.approve recovery.lock recovery.propose_unlock recovery.withdraw" {
		t.Errorf("commands %v", ids)
	}
}

func selectorOn(x ast.Expr, owner string) bool {
	sel, ok := x.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == owner
}

func packageSelector(n ast.Node, pkg string) (string, bool) {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	id, ok := sel.X.(*ast.Ident)
	return sel.Sel.Name, ok && id.Name == pkg
}

func TestRecoveryActorsAndEffectsAreConfined(t *testing.T) {
	systemPaths := map[string]bool{"Permit": true, "Call": true, "Done": true, "Record": true, "Report": true}
	never := map[string]bool{"Reconcile": true, "Reapplied": true, "Unapplicable": true, "RemoveWriter": true}
	found := map[string]int{}
	for _, src := range serverSources(t) {
		commandFile := src.name == "recovery.go" || src.name == "recoverycmds.go"
		for _, decl := range src.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			seed := fn.Name.Name == "seedRecovery"
			ast.Inspect(fn, func(n ast.Node) bool {
				if name, ok := packageSelector(n, "recovery"); ok && (name == "KindSystem" || name == "KindAssistant" || name == "ManifestDigest") {
					found[name]++
					if !seed {
						t.Errorf("%s %s names recovery.%s", src.name, fn.Name.Name, name)
					}
				}
				if sel, ok := n.(*ast.SelectorExpr); ok && selectorOn(sel.X, "recovery") {
					if systemPaths[sel.Sel.Name] {
						found[sel.Sel.Name]++
					}
					if systemPaths[sel.Sel.Name] && !seed || never[sel.Sel.Name] {
						t.Errorf("%s %s calls recovery %s", src.name, fn.Name.Name, sel.Sel.Name)
					}
				}
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				kind, inPackage := packageSelector(lit.Type, "recovery")
				if !inPackage {
					return true
				}
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok || fmt.Sprint(kv.Key) != "Kind" {
						continue
					}
					value, _ := packageSelector(kv.Value, "recovery")
					if kind == "Change" && !seed && value != "ChangeUnlock" {
						t.Errorf("%s %s builds a %s change", src.name, fn.Name.Name, value)
					}
					if kind == "Actor" && commandFile && value != "KindHuman" {
						t.Errorf("%s %s builds a %s actor", src.name, fn.Name.Name, value)
					}
					if kind == "Actor" && value == "KindHuman" {
						found["human actors"]++
					}
				}
				return true
			})
		}
		if commandFile {
			ast.Inspect(src.file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && (strings.Contains(lit.Value, "system") || strings.Contains(lit.Value, "assistant")) {
					t.Errorf("%s has the literal %s", src.name, lit.Value)
				}
				return true
			})
		}
	}
	if fmt.Sprint(found) != "map[Call:1 Done:1 KindSystem:2 ManifestDigest:1 Permit:1 Record:1 Report:1 human actors:3]" {
		t.Errorf("found %v", found)
	}
}

func TestRecoveryCommandsActAsTheHumanReader(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "approve", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "applied", "")
	if v, err := f.s.recovery.Proposal(10); err != nil || v.Approver != SampleAdmin || v.Proposer != SampleAdmin2 {
		t.Errorf("approver %+v %v", v, err)
	}
	f.ok(t, "lock", f.run(second, "recovery.lock", installTargetOf(recoveryStandby), nil), "locked", "")
	if l := f.s.rec.install(recoveryStandby).locks; l[len(l)-1].by != SampleAdmin2 {
		t.Errorf("lock by %+v", l)
	}
	f.ok(t, "propose", f.run(second, "recovery.propose_unlock", installTargetOf(recoveryStandby), map[string]string{"lost": sampleLost}), "pending", "")
	if v, err := f.s.recovery.Proposal(11); err != nil || v.Proposer != SampleAdmin2 {
		t.Errorf("proposer %+v %v", v, err)
	}
}

func TestRecoveryConcurrentReadsCommandsAndChecks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if got := f.recoveryRead(second); got.status != 200 {
					t.Errorf("concurrent read %d", got.status)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				f.s.recovery.Check(recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: recoveryPrimary, Generation: 2, Lost: sampleLost}})
			}
		}()
	}
	for i := 0; i < 3; i++ {
		for _, c := range []struct{ command, target, fields string }{{"approve", proposalTargetOf(uint64(10 + i)), ""},
			{"lock", installTargetOf(recoveryStandby), ""}, {"propose_unlock", installTargetOf(recoveryStandby), `"lost":"` + sampleLost + `"`}} {
			body := fmt.Sprintf(`{"request_id":"c%d%s","target_id":"%s","idempotency_key":"c%d%s","expected_revision":"%s","rationale":"x","body":{%s}}`, i, c.command, c.target, i,
				c.command, f.s.Revision(), c.fields)
			who := root
			if c.command == "propose_unlock" {
				who = second
			}
			f.call(recoveryCommand+c.command, body, who.cookie, who.csrf)
		}
	}
	wg.Wait()
}

func TestRecoveryLeaksNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	secrets := []string{"sample-outbox-tampered", "sample-bundle-1", "sample-user-1"}
	for _, d := range recoveryDomains {
		secrets = append(secrets, "sample-"+d+"-a", "sample-"+d+"-b")
	}
	for _, m := range [][]recovery.Inventory{recoveryManifest(""), recoveryManifest(recovery.DomainOutbox)} {
		for _, inv := range m {
			for _, c := range inv.Chunks {
				raw, _ := hex.DecodeString(c.Checksum)
				secrets = append(secrets, c.Checksum, strings.ToUpper(c.Checksum), base64.StdEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(raw), string(raw))
			}
		}
	}
	raws := map[string]string{"read": f.recoveryRead(root).raw, "second read": f.recoveryRead(second).raw}
	for _, c := range []struct {
		who            actor
		command        string
		target         string
		fields         map[string]string
		expectedStatus int
	}{
		{root, "approve", proposalTargetOf(10), nil, 200},
		{second, "lock", installTargetOf(recoveryStandby), nil, 200},
		{second, "propose_unlock", installTargetOf(recoveryStandby), map[string]string{"lost": sampleLost}, 200},
		{second, "withdraw", proposalTargetOf(11), nil, 200},
		{root, "lock", installTargetOf(recoveryPrimary), nil, 422},
		{root, "propose_unlock", installTargetOf(recoveryPrimary), map[string]string{"lost": sampleLost}, 422},
	} {
		got := f.run(c.who, recoveryCommand+c.command, c.target, c.fields)
		if got.status != c.expectedStatus {
			t.Fatalf("%s: %s", c.command, got.raw)
		}
		raws[c.command+" "+c.target] = got.raw
	}
	f.advance(2 * time.Minute)
	raws["final read"] = f.recoveryRead(root).raw
	for name, raw := range raws {
		for _, secret := range secrets {
			if strings.Contains(raw, secret) {
				t.Errorf("%s carries %q", name, secret)
			}
		}
		for _, word := range []string{"token", "expiry", "checksum", "items", "\"key\":\"sample-bundle", "permit\""} {
			if strings.Contains(raw, word) {
				t.Errorf("%s names %s", name, word)
			}
		}
	}
	if !strings.Contains(raws["read"], recovery.ManifestDigest(recoveryManifest(""))) || !strings.Contains(raws["read"], sampleHash("sample-backup-1")) {
		t.Error("read hides the restore point digests")
	}
}
