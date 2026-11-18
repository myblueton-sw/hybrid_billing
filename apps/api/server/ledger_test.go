package server

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (f world) restartWith(change func(*Config)) (world, error) {
	f.t.Helper()
	f.s.Close()
	cfg := f.s.cfg
	if change != nil {
		change(&cfg)
	}
	s, err := New(cfg)
	if err != nil {
		return world{}, err
	}
	s.manual = true
	if err := s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		f.t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		return world{}, err
	}
	s.counter = 2
	return world{t: f.t, s: s, h: s.Handler(), now: f.now, mu: f.mu, value: f.value}, nil
}

func (f world) restart(change func(*Config)) world {
	f.t.Helper()
	g, err := f.restartWith(change)
	if err != nil {
		f.t.Fatal(err)
	}
	return g
}

func ledgerFile(f world) string { return filepath.Join(f.s.cfg.Data, decisionledger.RelativeFile) }

func (f world) ledgerState(a actor) map[string]any {
	f.t.Helper()
	return f.accessRead(a, "").body["ledger"].(map[string]any)
}

func (f world) revocationRows(a actor) []map[string]any {
	f.t.Helper()
	out := []map[string]any{}
	for _, r := range f.accessRead(a, "").body["revocations"].([]any) {
		out = append(out, r.(map[string]any))
	}
	return out
}

func listingOf(t testing.TB, dir string) string {
	t.Helper()
	out := []string{}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Name() != ActivationFile && d.Name() != ReaderSecretFile {
			info, _ := d.Info()
			out = append(out, path+" "+info.ModTime().String()+" "+fmt.Sprint(info.Size()))
		}
		return nil
	})
	sort.Strings(out)
	return strings.Join(out, "\n")
}

func corrupt(t testing.TB, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[10] ^= 1
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRestartReplaysRevocations(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	made := f.grantVia(root, second, SampleMember, nil)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatal(got.raw)
	}
	if got := f.revokeVia(root, second, made); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.advance(time.Minute)
	g := f.restart(nil)
	if found, _ := g.s.grantView("reader-" + SampleAdmin2); !found.Closed {
		t.Fatal("the revoked seed grant is open after the restart")
	}
	for _, gr := range g.accessRead(g.as(AdminLogin), "").body["grants"].([]any) {
		if row := gr.(map[string]any); row["id"] == "reader-"+SampleAdmin2 && row["state"] != "revoked" {
			t.Errorf("a confirmed revocation reads %v", row["state"])
		}
	}
	root = g.as(AdminLogin)
	rows := g.revocationRows(root)
	if len(rows) != 2 || rows[0]["target"] != made || rows[0]["target_present"] != false || rows[0]["replay"] != "absent" || rows[1]["replay"] != "reapplied" ||
		rows[1]["origin"] != "ledger_replay" || rows[1]["state"] != "revocation_complete" || rows[1]["job_state"] != "succeeded" || rows[1]["recorded_at"] != stamp(start) ||
		rows[1]["verified"] != true || !strings.HasPrefix(rows[1]["receipt"].(map[string]any)["watermark"].(string), "2:") || rows[1]["request"] != "access-request-2" {
		t.Fatalf("rebuilt %+v", rows)
	}
	states := map[string]string{}
	for _, it := range rows[1]["cleanup"].([]any) {
		item := it.(map[string]any)
		states[item["item"].(string)] = item["state"].(string) + "/" + item["reason"].(string)
	}
	for item, want := range map[string]string{"C1": "done/server_restarted", "C2": "done/server_restarted", "C6": "done/server_restarted", "C3": "not_applicable/human_grants_only",
		"C4": "not_applicable/no_store", "C8": "not_applicable/nothing_keyed_to_grants", "C5": "outside_scope/cannot_recall", "C7": "outside_scope/no_job_runner",
		"C9": "not_applicable/no_dependent_delegation"} {
		if states[item] != want {
			t.Errorf("%s: %s", item, states[item])
		}
	}
	page := g.accessRead(root, "").body
	if page["started_at"] != stamp(start.Add(time.Minute)) || len(page["requests"].([]any)) != 0 {
		t.Errorf("started_at %v", page["started_at"])
	}
	g.setPassword(MemberLogin, pw)
	asked := g.requestGrant(root, SampleMember, nil)
	if requestOf(asked) != "access-request-4" {
		t.Errorf("request counter %s", asked.raw)
	}
	second = g.as(Admin2Login)
	if got := g.revokeVia(second, root, "reader-"+SampleAdmin); got.body["job_id"] != "revocation-3" {
		t.Errorf("revocation counter %s", got.raw)
	}
}

func TestReplayNeverClosesProtectedGrants(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.Close()
	l, err := decisionledger.Open(decisionledger.Options{Dir: f.s.cfg.Data, MaxBytes: mib, Clock: f.s.clock, Random: rand.Reader})
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range []decisionledger.Decision{
		{Workspace: SampleWorkspace, Decision: decisionledger.RevokeGrant, Target: "reader-console", Revocation: "revocation-40", RequestedBy: SampleAdmin},
		{Workspace: SampleWorkspace, Decision: decisionledger.RevokeGrant, Target: "installation-" + SampleAdmin, Revocation: "revocation-41", RequestedBy: SampleAdmin},
		{Workspace: SampleWorkspace, Decision: decisionledger.RevokeDelegation, Target: "del-gone", Revocation: "revocation-42", RequestedBy: SampleAdmin},
		{Workspace: "other-workspace", Decision: decisionledger.RevokeGrant, Target: "reader-" + SampleAdmin, Revocation: "revocation-43", RequestedBy: SampleAdmin, Request: "access-request-50"},
	} {
		if _, err := l.Commit(d); err != nil {
			t.Fatal(i, err)
		}
	}
	l.Close()
	g := f.restart(nil)
	for _, id := range []string{"reader-console", "installation-" + SampleAdmin, "reader-" + SampleAdmin} {
		if found, _ := g.s.grantView(id); found.Closed {
			t.Errorf("%s was closed by replay", id)
		}
	}
	rows := g.revocationRows(g.as(AdminLogin))
	if len(rows) != 3 || rows[1]["replay"] != "protected" || rows[2]["replay"] != "protected" || rows[0]["replay"] != "absent" || rows[0]["target_kind"] != "delegation" {
		t.Errorf("rows %+v", rows)
	}
	g.setPassword(MemberLogin, pw)
	if got := g.requestGrant(g.as(AdminLogin), SampleMember, nil); requestOf(got) != "access-request-51" {
		t.Errorf("counters follow every workspace %s", got.raw)
	}
}

func TestReplayClosesDelegationsAndHonoursTheKeptLimit(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.RevocationsKept, c.Access.EndedKept = 50, 50
	})
	f.setPassword(MemberLogin, pw)
	addDelegation(f, "del-a", SampleAdmin, SampleAdmin2, "")
	f.s.Close()
	l, err := decisionledger.Open(decisionledger.Options{Dir: f.s.cfg.Data, MaxBytes: mib, Clock: f.s.clock, Random: rand.Reader})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 52; i++ {
		target := fmt.Sprintf("gone-%d", i)
		decision := decisionledger.RevokeGrant
		switch i {
		case 47, 48:
			target = "reader-" + SampleAdmin2
		case 49, 51:
			target, decision = "del-a", decisionledger.RevokeDelegation
		}
		if _, err := l.Commit(decisionledger.Decision{Workspace: SampleWorkspace, Decision: decision, Target: target, Revocation: fmt.Sprintf("revocation-%d", i+1), RequestedBy: SampleAdmin}); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()
	s, err := New(f.s.cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820})
	g := world{t: t, s: s, now: f.now, mu: f.mu, value: f.value}
	s.AddStep("test-delegation", func() (bool, error) {
		addDelegation(g, "del-a", SampleAdmin, SampleAdmin2, "")
		return true, nil
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	g.h = s.Handler()
	rows := g.revocationRows(g.as(AdminLogin))
	if len(rows) != 50 || rows[0]["target"] != "del-a" || rows[0]["replay"] != "reapplied" || rows[0]["target_present"] != true {
		t.Errorf("rows %d %+v", len(rows), rows[0])
	}
	if d, _ := g.s.delegationView("del-a"); !d.Closed {
		t.Error("the delegation was not closed by replay")
	}
}

func TestLedgerIsolation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatal(got.raw)
	}
	pendingAdd := f.requestGrant(root, SampleMember, nil)
	f.s.Close()
	corrupt(t, ledgerFile(f))
	g := f.restart(nil)
	root, second = g.as(AdminLogin), g.as(Admin2Login)
	state := g.ledgerState(root)
	if state["state"] != "isolated" || state["isolation_reason"] != "integrity_failed" || fmt.Sprint(state["problem_line"]) != "1" {
		t.Fatalf("ledger %+v", state)
	}
	if found, _ := g.s.grantView("reader-" + SampleAdmin2); found.Closed {
		t.Error("isolation replayed the failed file")
	}
	before := listingOf(t, g.s.cfg.Data)
	g.setPassword(MemberLogin, pw)
	g.code(t, "grant addition", g.requestGrant(root, SampleMember, nil), 503, string(commands.DependencyUnavailable))
	v, _, err := g.s.changes.Propose(accesschange.Proposal{Workspace: SampleWorkspace, Change: accesschange.Change{Kind: accesschange.ChangeAdd, Subject: SampleMember, Action: "view",
		ResourceKind: ReaderKind, Fields: []string{ReaderField}, Customer: "all", Days: 1}, Basis: "x", CeilingUntil: start.Add(readerValidity),
		Proposer: accesschange.Actor{User: SampleAdmin, Human: true}, Allowed: true})
	if err != nil || pendingAdd.status != 200 {
		t.Fatal(err)
	}
	g.code(t, "approve an addition", g.run(second, "access.approve_request", v.ID, map[string]string{"digest": v.Digest}), 503, string(commands.DependencyUnavailable))
	asked := g.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	got := g.approveRequest(second, asked)
	if got.status != 202 {
		t.Fatalf("revocation in isolation %s", got.raw)
	}
	if found, _ := g.s.grantView("reader-" + SampleAdmin2); !found.Closed {
		t.Error("the deny was not applied in this process")
	}
	addDelegation(g, "del-i", SampleAdmin, SampleAdmin2, "")
	if got := g.run(root, "access.revoke_delegation", "del-i", nil); got.status != 202 {
		t.Errorf("delegation revocation in isolation %s", got.raw)
	}
	for id, want := range map[string]string{"reader-" + SampleAdmin2: "revocation_unconfirmed", "reader-" + SampleAdmin: "active"} {
		for _, gr := range g.accessRead(root, "").body["grants"].([]any) {
			if row := gr.(map[string]any); row["id"] == id && row["state"] != want {
				t.Errorf("grant %s state %v", id, row["state"])
			}
		}
	}
	if d := g.delegationRow(root, "del-i"); d["state"] != "revocation_unconfirmed" {
		t.Errorf("delegation state %+v", d)
	}
	for _, row := range g.revocationRows(root) {
		if row["state"] != "decision_unconfirmed" || row["job_state"] != "waiting" || row["job_reason"] != "decision_unconfirmed" ||
			row["receipt"].(map[string]any)["reason"] != "ledger_isolated" || row["verified"] != false || row["completed_at"] != "" {
			t.Errorf("row %+v", row)
		}
		if cleanup := fmt.Sprint(row["cleanup"]); !strings.Contains(cleanup, "not_started") || strings.Contains(cleanup, " done ") {
			t.Errorf("cleanup before confirmation %s", cleanup)
		}
	}
	g.withdrawn(t, root, second)
	if after := listingOf(t, g.s.cfg.Data); after != before {
		t.Errorf("isolation touched files\n%s\n%s", before, after)
	}
	h := g.restart(nil)
	if found, _ := h.s.grantView("reader-" + SampleAdmin2); found.Closed {
		t.Error("a revocation made in isolation was replayed")
	}
	replaced := h.restart(func(c *Config) { c.Access.AcceptNewLedger, c.Access.AcceptTrustReset = true, true })
	st := replaced.ledgerState(replaced.as(AdminLogin))
	if st["state"] != "ok" || st["replace_reason"] != "integrity_failed" || st["replaces"] == "" || !strings.HasPrefix(st["replaced_file"].(string), "ledger.jsonl.replaced-") {
		t.Errorf("replaced %+v", st)
	}
	if _, err := replaced.restartWith(func(c *Config) { c.Access.AcceptNewLedger = true }); !errors.Is(err, ErrConfig) {
		t.Errorf("the flag left on %v", err)
	}
}

func (f world) withdrawn(t *testing.T, root, second actor) {
	t.Helper()
	asked := f.run(second, "access.request_revoke", "reader-"+SampleAdmin, nil)
	if got := f.run(second, "access.withdraw_request", requestOf(asked), nil); got.status != 200 {
		t.Errorf("withdraw %s", got.raw)
	}
	asked = f.run(second, "access.request_revoke", "reader-"+SampleAdmin, nil)
	if got := f.run(root, "access.approve_request", requestOf(asked), map[string]string{"digest": "x"}); got.status != 422 {
		t.Errorf("wrong digest %s", got.raw)
	}
	if got := f.run(second, "access.withdraw_request", requestOf(asked), nil); got.status != 200 {
		t.Errorf("withdraw %s", got.raw)
	}
}

func TestLedgerMissingWithMarkerIsolates(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.Close()
	os.Remove(ledgerFile(f))
	g := f.restart(nil)
	if st := g.ledgerState(g.as(AdminLogin)); st["state"] != "isolated" || st["isolation_reason"] != "missing" || st["created_at"] != "" || st["ledger_id"] != "" {
		t.Errorf("%+v", st)
	}
	if _, err := os.Stat(ledgerFile(f)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("a missing ledger was recreated without the operator")
	}
	h := g.restart(func(c *Config) { c.Access.AcceptNewLedger, c.Access.AcceptTrustReset = true, true })
	if st := h.ledgerState(h.as(AdminLogin)); st["state"] != "ok" || st["replaces"] != "unknown" || st["replace_reason"] != "missing" {
		t.Errorf("replaced %+v", st)
	}
}

func TestTornTailIsRecoveredAtStart(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.Close()
	file, _ := os.OpenFile(ledgerFile(f), os.O_WRONLY|os.O_APPEND, 0o600)
	file.Write([]byte(`{"v":1,"seq":2`))
	file.Close()
	g := f.restart(nil)
	st := g.ledgerState(g.as(AdminLogin))
	recovery, _ := st["recovery"].(map[string]any)
	if st["state"] != "recovered" || recovery == nil || fmt.Sprint(recovery["truncated_bytes"]) != "14" || fmt.Sprint(st["records"]) != "2" {
		t.Errorf("recovered %+v", st)
	}
	g.setPassword(MemberLogin, pw)
	if got := g.requestGrant(g.as(AdminLogin), SampleMember, nil); got.status != 200 {
		t.Errorf("a recovered ledger allows additions %s", got.raw)
	}
}

func (f world) stoppedLedger() {
	f.t.Helper()
	f.s.ledger.Close()
}

func TestLedgerStopped(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	pending := f.requestGrant(root, SampleMember, nil)
	f.stoppedLedger()
	if st := f.ledgerState(root); st["state"] != "stopped" {
		t.Fatalf("%+v", st)
	}
	f.code(t, "proposal", f.requestGrant(root, SampleMember, map[string]string{"days": "5"}), 503, string(commands.DependencyUnavailable))
	f.code(t, "approval", f.approveRequest(second, pending), 503, string(commands.DependencyUnavailable))
	got := f.revokeVia(root, second, "reader-"+SampleAdmin2)
	rows := f.revocationRows(root)
	if got.status != 202 || rows[0]["state"] != "decision_unconfirmed" || rows[0]["receipt"].(map[string]any)["reason"] != "ledger_stopped" {
		t.Errorf("revocation on a stopped ledger %s %+v", got.raw, rows)
	}
	if !strings.Contains(logs.String(), `"msg":"decision_ledger_failed","state":"stopped"`) {
		t.Errorf("log %s", logs.String())
	}
}

var fullOnce sync.Once
var fullLedger, fullMarker, overLedger []byte

func buildFullLedger(t testing.TB) {
	fullOnce.Do(func() {
		dir := filepath.Join(os.TempDir(), "hb298-full-"+fmt.Sprint(time.Now().UnixNano()))
		os.Mkdir(dir, 0o700)
		defer os.RemoveAll(dir)
		big := decisionledger.Decision{Workspace: "other-workspace", Decision: decisionledger.RevokeGrant, Target: "gone", RequestedBy: SampleAdmin}
		for len(big.EndedDelegations) < 40 {
			big.EndedDelegations = append(big.EndedDelegations, "delegation-with-a-long-identifier")
		}
		commitUntilFull := func(bound int64, extra int) {
			l, err := decisionledger.Open(decisionledger.Options{Dir: dir, MaxBytes: bound, Clock: func() time.Time { return start }, Random: rand.Reader})
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			for i := 0; extra < 0 || i < extra; i++ {
				big.Revocation = fmt.Sprintf("revocation-%d", l.Status().Records+1)
				if _, err := l.Commit(big); err != nil {
					return
				}
			}
		}
		commitUntilFull(mib, -1)
		fullLedger, _ = os.ReadFile(filepath.Join(dir, decisionledger.RelativeFile))
		fullMarker, _ = os.ReadFile(filepath.Join(dir, decisionledger.MarkerName))
		commitUntilFull(2*mib, 2)
		overLedger, _ = os.ReadFile(filepath.Join(dir, decisionledger.RelativeFile))
	})
}

func placeLedger(t testing.TB, data string, ledger []byte) {
	t.Helper()
	buildFullLedger(t)
	os.MkdirAll(filepath.Join(data, decisionledger.DirName), 0o700)
	if os.WriteFile(filepath.Join(data, decisionledger.RelativeFile), ledger, 0o600) != nil || os.WriteFile(filepath.Join(data, decisionledger.MarkerName), fullMarker, 0o600) != nil {
		t.Fatal("place ledger")
	}
}

func fullWorld(t *testing.T) world {
	t.Helper()
	f := setup(t, SeedSample)
	os.MkdirAll(f.s.cfg.Data, 0o700)
	buildFullLedger(t)
	placeLedger(t, f.s.cfg.Data, fullLedger)
	f.s.cfg.Access.LedgerMaxMiB = 1
	return f.ready()
}

func TestLedgerFullAndBound(t *testing.T) {
	f := fullWorld(t)
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if st := f.ledgerState(root); st["state"] != "full" {
		t.Fatalf("a ledger within one record of the bound %+v", st)
	}
	f.code(t, "proposal", f.requestGrant(root, SampleMember, nil), 503, string(commands.DependencyUnavailable))
	got := f.revokeVia(root, second, "reader-"+SampleAdmin2)
	if rows := f.revocationRows(root); got.status != 202 || rows[0]["receipt"].(map[string]any)["reason"] != "ledger_full" {
		t.Errorf("revocation on a full ledger %+v", rows)
	}
	over := setup(t, SeedSample)
	os.MkdirAll(over.s.cfg.Data, 0o700)
	placeLedger(t, over.s.cfg.Data, overLedger)
	over.s.cfg.Access.LedgerMaxMiB = 1
	if err := over.s.Start(); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "decision_ledger_max_mib") {
		t.Errorf("a bound below the file %v", err)
	}
}

func TestHeldLockFailsStart(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	g := setup(t, SeedSample)
	g.s.cfg.Data = f.s.cfg.Data
	err := g.s.Start()
	if !errors.Is(err, ErrStart) || !strings.Contains(err.Error(), "in use by another process") || g.s.counter != 19 {
		t.Errorf("held lock %v counter %d", err, g.s.counter)
	}
	for _, c := range []struct {
		err  error
		want error
	}{{decisionledger.ErrLocked, ErrStart}, {decisionledger.ErrBound, ErrConfig}, {decisionledger.ErrNothingToReplace, ErrConfig}, {decisionledger.ErrUnsafe, ErrStart}} {
		if !errors.Is(ledgerStartError(c.err), c.want) {
			t.Errorf("%v", c.err)
		}
	}
	unsafe := setup(t, SeedSample)
	os.Mkdir(unsafe.s.cfg.Data, 0o700)
	os.Mkdir(filepath.Join(unsafe.s.cfg.Data, decisionledger.DirName), 0o755)
	os.Chmod(filepath.Join(unsafe.s.cfg.Data, decisionledger.DirName), 0o755)
	if err := unsafe.s.Start(); !errors.Is(err, ErrStart) {
		t.Errorf("an open decisions directory %v", err)
	}
	var none Server
	if none.Close() != nil || !none.ledgerBlocksIncrease() || none.ledgerRow().State != "isolated" {
		t.Error("a server without a ledger")
	}
	if _, err := (ledgerAdapter{&none}).Commit(accesschange.Entry{}); err == nil {
		t.Error("commit without a ledger")
	}
}

func TestLedgerHoldsNoRationaleOrLogin(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	secret := "confidential-rationale-298"
	asked := f.admin(root, "access.request_grant", grantsTarget, secret, map[string]string{"subject": SampleMember, "action": "view", "kind": ReaderKind, "fields": ReaderField, "customer": "all", "days": "3"})
	fields := asked.body["fields"].(map[string]any)
	f.admin(second, "access.approve_request", fields["request"].(string), secret, map[string]string{"digest": fields["digest"].(string)})
	revoke := f.admin(root, "access.request_revoke", "reader-"+SampleAdmin2, secret, map[string]string{})
	fields = revoke.body["fields"].(map[string]any)
	if got := f.admin(second, "access.approve_request", fields["request"].(string), secret, map[string]string{"digest": fields["digest"].(string)}); got.status != 202 {
		t.Fatal(got.raw)
	}
	raw, _ := os.ReadFile(ledgerFile(f))
	for _, forbidden := range []string{secret, AdminLogin, Admin2Login, MemberLogin, "@example"} {
		if bytes.Contains(raw, []byte(forbidden)) || strings.Contains(logs.String(), secret) {
			t.Errorf("%s leaked", forbidden)
		}
	}
	if !bytes.Contains(raw, []byte(`"requested_by":"`+SampleAdmin+`","approved_by":"`+SampleAdmin2+`"`)) {
		t.Errorf("ledger %s", raw)
	}
}

func addDelegation(f world, id, from, to, parent string) {
	f.t.Helper()
	now := f.s.clock()
	terms := gate.Terms{Action: gate.View, Kind: ReaderKind, Fields: []string{ReaderField}, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	if err := f.s.admin.AddDelegation(gate.Delegation{ID: id, DelegatorID: from, DelegateID: to, ParentID: parent, AllowRedelegation: true, Terms: terms,
		Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}}); err != nil {
		f.t.Fatal(err)
	}
}

func TestStartFailsAfterTheLedgerWhenTheLinkCannotBeWritten(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.Close()
	s, _ := New(f.s.cfg)
	s.AddStep("read-only", func() (bool, error) { return true, os.Chmod(f.s.cfg.Data, 0o500) })
	defer os.Chmod(f.s.cfg.Data, 0o700)
	if s.Start() == nil {
		t.Error("link write fails")
	}
	s.Close()
}

func TestAccessReadNeedsTheUserList(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.fault = func(at string) bool { return at == "users" }
	f.code(t, "read without users", f.accessRead(root, ""), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	if _, err := (delegationCommand{s: f.s}).Apply(commands.Context{User: "nobody"}); err.(*commands.Failure).Code != commands.PrerequisiteFailed {
		t.Errorf("a delegation revocation by an unknown actor %v", err)
	}
	for i := 0; i < 120; i++ {
		f.must(f.s.identity.Installation().CreateUser(seedOperator, fmt.Sprintf("user-bulk-%03d", i), LoginTriple(fmt.Sprintf("bulk-%03d@example.invalid", i)), "bulk@example.invalid"))
	}
	f.s.caller = root.verifier(f)
	facts, err := f.s.accessFacts(f.s.caller)
	f.s.caller = [32]byte{}
	if err != nil || len(facts.users) < 123 {
		t.Errorf("paged users %d %v", len(facts.users), err)
	}
}

func flipInLine(t testing.TB, path string, line, offset int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	at := 0
	for i := 1; i < line; i++ {
		at += bytes.IndexByte(data[at:], '\n') + 1
	}
	data[at+offset] ^= 1
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestIsolationReappliesTheVerifiedPrefix(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatal(got.raw)
	}
	if got := f.revokeVia(second, root, "reader-"+SampleAdmin); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.s.Close()
	flipInLine(t, ledgerFile(f), 3, 30)
	g := f.restart(nil)
	root = g.as(AdminLogin)
	before := listingOf(t, g.s.cfg.Data)
	if found, _ := g.s.grantView("reader-" + SampleAdmin2); !found.Closed {
		t.Error("a revocation recorded before the damaged line came back")
	}
	if found, _ := g.s.grantView("reader-" + SampleAdmin); found.Closed {
		t.Error("the damaged record was re-applied")
	}
	st := g.ledgerState(root)
	if st["state"] != "isolated" || st["isolation_reason"] != "integrity_failed" || fmt.Sprint(st["problem_line"]) != "3" {
		t.Fatalf("ledger %+v", st)
	}
	rows := g.revocationRows(root)
	if len(rows) != 1 || rows[0]["origin"] != "ledger_replay" || rows[0]["replay"] != "reapplied" || rows[0]["target"] != "reader-"+SampleAdmin2 {
		t.Errorf("rows %+v", rows)
	}
	for _, gr := range g.accessRead(root, "").body["grants"].([]any) {
		if row := gr.(map[string]any); row["id"] == "reader-"+SampleAdmin2 && row["state"] != "revoked" {
			t.Errorf("a re-applied revocation reads %v", row["state"])
		}
	}
	g.setPassword(MemberLogin, pw)
	g.code(t, "grant addition", g.requestGrant(root, SampleMember, nil), 503, string(commands.DependencyUnavailable))
	addDelegation(g, "del-p", SampleAdmin, SampleAdmin2, "")
	if got := g.run(root, "access.revoke_delegation", "del-p", nil); got.status != 202 || g.revocationRows(root)[0]["receipt"].(map[string]any)["reason"] != "ledger_isolated" {
		t.Errorf("a new decision is still refused by the ledger %s", got.raw)
	}
	if after := listingOf(t, g.s.cfg.Data); after != before {
		t.Errorf("isolation wrote\n%s\n%s", before, after)
	}
	h := g.restart(nil)
	if found, _ := h.s.grantView("reader-" + SampleAdmin2); !found.Closed {
		t.Error("the prefix is re-applied on every isolated start")
	}
}

func TestReplayFailureRefusesStart(t *testing.T) {
	for _, kind := range []string{"grant", "delegation"} {
		f := setup(t, SeedSample).ready()
		f.s.Close()
		l, err := decisionledger.Open(decisionledger.Options{Dir: f.s.cfg.Data, MaxBytes: mib, Clock: f.s.clock, Random: rand.Reader})
		if err != nil {
			t.Fatal(err)
		}
		d := decisionledger.Decision{Workspace: SampleWorkspace, Decision: decisionledger.RevokeGrant, Target: "reader-" + SampleAdmin2, Revocation: "revocation-1", RequestedBy: SampleAdmin}
		if kind == "delegation" {
			d.Decision, d.Target = decisionledger.RevokeDelegation, "del-r"
		}
		if _, err := l.Commit(d); err != nil {
			t.Fatal(err)
		}
		l.Close()
		s, _ := New(f.s.cfg)
		s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820})
		g := world{t: t, s: s, now: f.now, mu: f.mu, value: f.value}
		s.AddStep("test-delegation", func() (bool, error) {
			addDelegation(g, "del-r", SampleAdmin, SampleAdmin2, "")
			return true, nil
		})
		s.fault = func(at string) bool { return at == "replay" }
		err = s.Start()
		if !errors.Is(err, ErrStart) || !strings.Contains(err.Error(), "replay could not re-apply revocation-1") {
			t.Errorf("%s: %v", kind, err)
		}
		reopened, err := decisionledger.Open(decisionledger.Options{Dir: f.s.cfg.Data, MaxBytes: mib, Clock: f.s.clock, Random: rand.Reader})
		if err != nil {
			t.Errorf("%s: the ledger lock was kept after a refused start %v", kind, err)
		} else {
			reopened.Close()
		}
	}
}

func TestAFinalRecordWithoutItsNewlineStaysApplied(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.s.Close()
	data, _ := os.ReadFile(ledgerFile(f))
	os.WriteFile(ledgerFile(f), data[:len(data)-1], 0o600)
	g := f.restart(nil)
	if found, _ := g.s.grantView("reader-" + SampleAdmin2); !found.Closed {
		t.Error("the grant revoked by the final record came back")
	}
	st := g.ledgerState(g.as(AdminLogin))
	if st["state"] != "isolated" || st["isolation_reason"] != "integrity_failed" || fmt.Sprint(st["problem_line"]) != "2" || st["recovery"] != nil {
		t.Errorf("ledger %+v", st)
	}
	if after, _ := os.ReadFile(ledgerFile(f)); !bytes.Equal(after, data[:len(data)-1]) {
		t.Error("the file changed")
	}
}

func TestAForeignMarkerAndADamagedLineReplayNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatal(got.raw)
	}
	if got := f.revokeVia(second, root, "reader-"+SampleAdmin); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.s.Close()
	flipInLine(t, ledgerFile(f), 3, 30)
	os.WriteFile(filepath.Join(f.s.cfg.Data, decisionledger.MarkerName), []byte(`{"ledger_id":"`+strings.Repeat("c", 32)+`","created_at":"x"}`), 0o600)
	g := f.restart(nil)
	for _, id := range []string{"reader-" + SampleAdmin2, "reader-" + SampleAdmin} {
		if found, _ := g.s.grantView(id); found.Closed {
			t.Errorf("%s was replayed from a foreign ledger", id)
		}
	}
	if st := g.ledgerState(g.as(AdminLogin)); st["state"] != "isolated" || st["isolation_reason"] != "id_mismatch" || len(g.revocationRows(g.as(AdminLogin))) != 0 {
		t.Errorf("ledger %+v", st)
	}
}

func replayScenario(t *testing.T, record decisionledger.Decision, step func(world) error) (world, uint64) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.s.Close()
	l, err := decisionledger.Open(decisionledger.Options{Dir: f.s.cfg.Data, MaxBytes: mib, Clock: f.s.clock, Random: rand.Reader})
	if err != nil {
		t.Fatal(err)
	}
	record.Workspace, record.Revocation, record.RequestedBy = SampleWorkspace, "revocation-1", SampleAdmin
	if _, err := l.Commit(record); err != nil {
		t.Fatal(err)
	}
	l.Close()
	s, _ := New(f.s.cfg)
	s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820})
	g := world{t: t, s: s, now: f.now, mu: f.mu, value: f.value}
	if step != nil {
		s.AddStep("test-step", func() (bool, error) { return true, step(g) })
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	counter := s.counter
	s.counter = 2
	g.h = s.Handler()
	t.Cleanup(func() { s.Close() })
	return g, counter
}

func TestReplayCountsOnlyWhatItCloses(t *testing.T) {
	grant := func(id string) decisionledger.Decision {
		return decisionledger.Decision{Decision: decisionledger.RevokeGrant, Target: id}
	}
	delegation := func(id string) decisionledger.Decision {
		return decisionledger.Decision{Decision: decisionledger.RevokeDelegation, Target: id}
	}
	add := func(g world) error { addDelegation(g, "del-s", SampleAdmin, SampleAdmin2, ""); return nil }
	for name, c := range map[string]struct {
		record  decisionledger.Decision
		step    func(world) error
		counter uint64
		present bool
		replay  string
	}{
		"absent grant":         {grant("gone"), nil, 20, false, "absent"},
		"protected grant":      {grant("reader-console"), nil, 20, true, "protected"},
		"open grant":           {grant("reader-" + SampleAdmin2), nil, 21, true, "reapplied"},
		"already closed grant": {grant("reader-" + SampleAdmin2), func(g world) error { return g.s.admin.RevokeGrant("reader-"+SampleAdmin2, 1) }, 21, true, "reapplied"},
		"absent delegation":    {delegation("del-s"), nil, 20, false, "absent"},
		"open delegation":      {delegation("del-s"), add, 22, true, "reapplied"},
		"already closed delegation": {delegation("del-s"), func(g world) error {
			add(g)
			return g.s.admin.RevokeDelegation("del-s", 1)
		}, 21, true, "reapplied"},
	} {
		t.Run(name, func(t *testing.T) {
			g, counter := replayScenario(t, c.record, c.step)
			rows := g.revocationRows(g.as(AdminLogin))
			if counter != c.counter || len(rows) != 1 || rows[0]["target_present"] != c.present || rows[0]["replay"] != c.replay {
				t.Errorf("counter %d rows %+v", counter, rows)
			}
		})
	}
}

func TestAStartSyncFailureStillReplaysAndStops(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.s.Close()
	s, _ := New(f.s.cfg)
	s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820})
	s.fault = func(at string) bool { return at == "ledger-start-sync" }
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.fault = nil
	g := world{t: t, s: s, h: s.Handler(), now: f.now, mu: f.mu, value: f.value}
	t.Cleanup(func() { s.Close() })
	if found, _ := g.s.grantView("reader-" + SampleAdmin2); !found.Closed {
		t.Error("a revoked grant came back after a start sync failure")
	}
	root = g.as(AdminLogin)
	st := g.ledgerState(root)
	if st["state"] != "stopped" || st["not_durable"] != true {
		t.Errorf("ledger %+v", st)
	}
	if rows := g.revocationRows(root); len(rows) != 1 || rows[0]["replay"] != "reapplied" {
		t.Errorf("rows %+v", rows)
	}
	g.setPassword(MemberLogin, pw)
	g.code(t, "grant addition", g.requestGrant(root, SampleMember, nil), 503, "dependency_unavailable")
	if g.ledgerState(g.as(AdminLogin))["records"] != st["records"] {
		t.Error("a record was written")
	}
}
