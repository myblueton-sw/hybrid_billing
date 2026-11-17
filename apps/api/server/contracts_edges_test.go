package server

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
)

func TestContractDetailPagesRecords(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-01"
	for n := 0; n < recordsShown+1; n++ {
		f.advance(time.Second)
		ok(t, "gap", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": fmt.Sprintf("f-%d", n)}))
	}
	detail := f.contract(root, id, "")
	records := detail.body["records"].([]any)
	if len(records) != recordsShown || fmt.Sprint(detail.body["earlier_records"]) != "2" || fmt.Sprint(records[0].(map[string]any)["sequence"]) != "102" {
		t.Errorf("records %d %v", len(records), detail.body["earlier_records"])
	}
}

func TestContractHintsForSettlement(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-02"
	ok(t, "end", f.run(root, "contract.end", id, map[string]string{"from": "2026-11-16"}))
	settle := func(want string) {
		t.Helper()
		if a := actionOf(f.contract(root, id, ""), "contract.settle"); a["blocked"] != want {
			t.Errorf("settle hint %+v want %s", a, want)
		}
	}
	settle("evidence_needed")
	ok(t, "short", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "short", "covered_until": "2026-11-15"}))
	settle("evidence_needed")
	f.code(t, "short settle", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-16"}), 422, "prerequisite_failed")
	ok(t, "full", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "full", "covered_until": "2026-11-16"}))
	settle("")
	ok(t, "settle", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-16"}))
	settle("not_pending_final")
	detail := f.contract(root, id, "")
	if a := actionOf(detail, "contract.withdraw"); a["blocked"] != "settled" {
		t.Errorf("withdraw hint %+v", a)
	}
	end := detail.body["events"].([]any)[2].(map[string]any)
	if end["withdraw_blocked"] != "settled" {
		t.Errorf("event hint %+v", end)
	}
	f.code(t, "withdraw settled", f.run(root, "contract.withdraw", id, map[string]string{"event": end["id"].(string), "cancel": "recorded_in_error"}), 422, "invalid_stage")
	other := f.contract(root, "sample-contract-01", "")
	if a := actionOf(other, "contract.withdraw"); a["blocked"] != "nothing_withdrawable" {
		t.Errorf("nothing %+v", a)
	}
}

func TestContractWriteLimitKeepsReserve(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.lifecycleLimits = contracts.Limits{Contracts: 10, Writes: 2, Reserve: 1, Batch: 8}
	small, err := contracts.NewRegistry(f.s.lifecycleLimits, f.s.clock)
	if err != nil {
		t.Fatal(err)
	}
	f.s.lifecycles = small
	id := "sample-contract-01"
	start := contracts.Event{ID: "ev-s", Kind: contracts.Start, From: contractDay(2026, 1, 1), Revision: 1}
	if _, err := small.Amend(SampleWorkspace, id, 0, "cw-s", nil, []contracts.Event{start}, ""); err != nil {
		t.Fatal(err)
	}
	ok(t, "renew", f.run(root, "contract.renew", id, map[string]string{"from": "2026-12-01"}))
	f.code(t, "limit", f.run(root, "contract.renew", id, map[string]string{"from": "2026-12-05"}), 422, "prerequisite_failed")
	ok(t, "end uses reserve", f.run(root, "contract.end", id, map[string]string{"from": "2026-12-10"}))
	f.code(t, "reserve used", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f"}), 422, "prerequisite_failed")
}

func TestContractSeedFailures(t *testing.T) {
	fresh := func(limits contracts.Limits) *Server {
		s, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedNone})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.orgWorkspace(); err != nil {
			t.Fatal(err)
		}
		if err := s.seedOrganizations(); err != nil {
			t.Fatal(err)
		}
		s.lifecycles, _ = contracts.NewRegistry(limits, s.clock)
		return s
	}
	wide := ContractLimits()
	s := fresh(wide)
	if err := s.seedContracts(); err != nil {
		t.Fatal(err)
	}
	if err := s.seedContracts(); err == nil {
		t.Error("second seed accepted")
	}
	s = fresh(wide)
	event := contracts.Event{ID: "ev-x", Kind: contracts.Start, From: contractDay(2026, 1, 1), Revision: 1}
	if _, err := s.lifecycles.Amend(s.Workspace(), "sample-contract-01", 0, "cw-x", nil, []contracts.Event{event}, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.seedContracts(); err == nil {
		t.Error("amend conflict accepted")
	}
	for _, limits := range []contracts.Limits{{Contracts: 10, Writes: 1, Reserve: 1, Batch: 8}, {Contracts: 10, Writes: 2, Reserve: 1, Batch: 8}} {
		if err := fresh(limits).seedContracts(); err == nil {
			t.Errorf("limits %+v accepted", limits)
		}
	}
	if err := s.seedWrite(s.Workspace(), "absent", "cw-a", nil); !errors.Is(err, contracts.ErrNotFound) {
		t.Errorf("absent %v", err)
	}
}

func TestContractHintsAreExactProperty(t *testing.T) {
	commandsOf := []string{"contract.suspend", "contract.resume", "contract.renew", "contract.end", "contract.withdraw", "contract.evidence.secured", "contract.evidence.missing", "contract.settle", "contract.reopen"}
	ids := []string{"sample-contract-01", "sample-contract-02", "sample-contract-03", "sample-contract-04", "sample-contract-05"}
	for run := 0; run < 12; run++ {
		rng := rand.New(rand.NewSource(int64(run)))
		f := setup(t, SeedSample).ready()
		root := f.as(AdminLogin)
		moved := 0
		for step := 0; step < 30; step++ {
			if days := 1 + rng.Intn(3); rng.Intn(4) == 0 && moved+days <= 20 {
				moved += days
				f.advance(time.Duration(days) * 24 * time.Hour)
				root = f.as(AdminLogin)
			}
			id := ids[rng.Intn(len(ids))]
			detail := f.contract(root, id, "")
			if detail.status != 200 {
				t.Fatalf("detail %s", detail.raw)
			}
			today := detail.body["today"].(string)
			command := commandsOf[rng.Intn(len(commandsOf))]
			hint := actionOf(detail, command)
			fields := map[string]string{}
			switch command {
			case "contract.withdraw":
				events := detail.body["events"].([]any)
				e := events[rng.Intn(len(events))].(map[string]any)
				fields["event"] = e["id"].(string)
				if e["cancel_required"] == true {
					fields["cancel"] = "contract_continues"
				}
				hint = map[string]any{"allowed": e["withdraw_blocked"] == ""}
			case "contract.evidence.secured":
				fields = map[string]string{"kind": "secured", "reference": fmt.Sprintf("ref-%d", step), "covered_until": today}
			case "contract.settle":
				fields["as_of"] = today
			case "contract.reopen":
				fields["reason"] = "correction"
			default:
				fields["from"] = today
				if e, _ := hint["earliest"].(string); e != "" {
					fields["from"] = e
				}
			}
			if command == "contract.evidence.missing" {
				fields = map[string]string{"kind": "missing", "reason": "late_source_received", "follow_up": fmt.Sprintf("f-%d", step)}
			}
			name := command
			if len(name) > len("contract.evidence") && name[:len("contract.evidence")] == "contract.evidence" {
				name = "contract.evidence"
			}
			got := f.run(root, name, id, fields)
			if (got.status == 200) != (hint["allowed"] == true) {
				t.Fatalf("run %d step %d %s %s %v hint %v got %s", run, step, command, id, fields, hint, got.raw)
			}
			for _, row := range f.contractsRead(root, "").body["contracts"].([]any) {
				r := row.(map[string]any)
				seq, _ := f.s.lifecycles.Sequence(SampleWorkspace, r["id"].(string))
				position, _ := f.s.lifecycles.StateOn(SampleWorkspace, r["id"].(string), seq, contractToday(f.s.clock()))
				if r["state"] != string(position.State) {
					t.Fatalf("state %v vs %v", r, position)
				}
				if r["settlement"] == "settled" && fmt.Sprint(r["open_gaps"]) != "0" {
					t.Fatalf("settled with gap %v", r)
				}
			}
		}
	}
}

func TestContractHintsAtLimitsAndHorizon(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	limits := ContractLimits()
	small, _ := contracts.NewRegistry(limits, f.s.clock)
	f.s.lifecycles = small
	id := "sample-contract-01"
	start := contracts.Event{ID: "ev-s", Kind: contracts.Start, From: contractDay(2026, 1, 1), Revision: 1}
	if _, err := small.Amend(SampleWorkspace, id, 0, "cw-s", nil, []contracts.Event{start}, ""); err != nil {
		t.Fatal(err)
	}
	far := contracts.Event{ID: "ev-far", Kind: contracts.Renew, From: mustAdd(contractDay(2026, 11, 16), horizonDays)}
	if _, err := small.Amend(SampleWorkspace, id, 1, "cw-far", nil, []contracts.Event{far}, ""); err != nil {
		t.Fatal(err)
	}
	detail := f.contract(root, id, "")
	if a := actionOf(detail, "contract.renew"); a["blocked"] != "horizon" {
		t.Errorf("horizon %+v", a)
	}
	for n := 2; n < limits.Writes; n++ {
		seq, _ := small.Sequence(SampleWorkspace, id)
		if _, err := small.RecordEvidence(SampleWorkspace, id, seq, fmt.Sprintf("cw-g%d", n), contracts.Evidence{Kind: contracts.Missing, Reason: contracts.NotAttempted, OwnerID: "o", FollowUp: "f"}); err != nil {
			t.Fatal(n, err)
		}
	}
	detail = f.contract(root, id, "")
	want := map[string]string{"contract.renew": "horizon", "contract.withdraw": "nothing_withdrawable", "contract.evidence.missing": "", "contract.suspend": "horizon"}
	for command, blocked := range want {
		if a := actionOf(detail, command); a["blocked"] != blocked {
			t.Errorf("%s %+v", command, a)
		}
	}
	for _, e := range detail.body["events"].([]any) {
		if row := e.(map[string]any); row["id"] == "ev-far" && row["withdraw_blocked"] != "write_limit" {
			t.Errorf("far %+v", row)
		}
	}
	f.code(t, "withdraw at limit", f.run(root, "contract.withdraw", id, map[string]string{"event": "ev-far"}), 422, "prerequisite_failed")
	for n := 0; n < limits.Reserve; n++ {
		f.advance(time.Second)
		ok(t, "reserve", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f"}))
	}
	if a := actionOf(f.contract(root, id, ""), "contract.evidence.missing"); a["blocked"] != "write_limit" {
		t.Errorf("reserve hint %+v", a)
	}
}

func TestContractReserveForSuspendAndReopen(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.lifecycleLimits = contracts.Limits{Contracts: 10, Writes: 3, Reserve: 2, Batch: 8}
	small, _ := contracts.NewRegistry(f.s.lifecycleLimits, f.s.clock)
	f.s.lifecycles = small
	id := "sample-contract-04"
	events := []contracts.Event{{ID: "ev-s", Kind: contracts.Start, From: contractDay(2026, 2, 1), Revision: 1}, {ID: "ev-e", Kind: contracts.End, From: contractDay(2026, 8, 1)}}
	if _, err := small.Amend(SampleWorkspace, id, 0, "cw-s", nil, events, ""); err != nil {
		t.Fatal(err)
	}
	ok(t, "secured", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-08-01"}))
	ok(t, "settle", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-08-01"}))
	if a := actionOf(f.contract(root, id, ""), "contract.reopen"); a["allowed"] != true {
		t.Errorf("reopen hint %+v", a)
	}
	ok(t, "reopen in reserve", f.run(root, "contract.reopen", id, map[string]string{"reason": "late_credit"}))
	if a := actionOf(f.contract(root, id, ""), "contract.settle"); a["blocked"] != "evidence_needed" {
		t.Errorf("settle %+v", a)
	}
	other := "sample-contract-01"
	if _, err := small.Amend(SampleWorkspace, other, 0, "cw-o", nil, []contracts.Event{{ID: "ev-o", Kind: contracts.Start, From: contractDay(2026, 1, 1), Revision: 1}}, ""); err != nil {
		t.Fatal(err)
	}
	ok(t, "renew", f.run(root, "contract.renew", other, map[string]string{"from": "2026-12-01"}))
	ok(t, "renew2", f.run(root, "contract.renew", other, map[string]string{"from": "2026-12-02"}))
	if a := actionOf(f.contract(root, other, ""), "contract.renew"); a["blocked"] != "write_limit" {
		t.Errorf("renew at limit %+v", a)
	}
	if a := actionOf(f.contract(root, other, ""), "contract.suspend"); a["allowed"] != true {
		t.Errorf("suspend reserve %+v", a)
	}
	ok(t, "suspend in reserve", f.run(root, "contract.suspend", other, map[string]string{"from": "2026-12-03"}))
}

func TestContractWithdrawKeepsGapsAndAging(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-01"
	end := ok(t, "end", f.run(root, "contract.end", id, map[string]string{"from": "2026-11-18"}))["event"].(string)
	ok(t, "gap", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "provider_refused", "follow_up": "f"}))
	got := f.run(root, "contract.withdraw", id, map[string]string{"event": end, "cancel": "recorded_in_error"})
	if got.status != 200 || got.body["business_state"] != "none" {
		t.Fatalf("withdraw %s", got.raw)
	}
	if row := contractRowOf(t, f.contractsRead(root, ""), id); fmt.Sprint(row["open_gaps"]) != "1" || row["end"] != "" {
		t.Errorf("gap kept %+v", row)
	}
	renew := ok(t, "renew", f.run(root, "contract.renew", id, map[string]string{"from": "2026-11-17"}))["event"].(string)
	f.advance(2 * 24 * time.Hour)
	root = f.as(AdminLogin)
	f.code(t, "aged", f.run(root, "contract.withdraw", id, map[string]string{"event": renew}), 422, "invalid_request")
}

func TestContractReadFailsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.contracts.byKey[planKey(SampleWorkspace, "sample-contract-01")].profile = "missing"
	f.code(t, "profile", f.contractsRead(root, ""), 503, "dependency_unavailable")
	f.s.contracts.byKey[planKey(SampleWorkspace, "sample-contract-01")].profile = "sample-profile-north"
	id := "sample-contract-05"
	seq, _ := f.s.lifecycles.Sequence(SampleWorkspace, id)
	if _, err := f.s.lifecycles.Amend(SampleWorkspace, id, seq, "cw-empty", []string{"ev-05-start"}, nil, ""); err != nil {
		t.Fatal(err)
	}
	f.code(t, "empty list", f.contractsRead(root, ""), 503, "dependency_unavailable")
	f.code(t, "empty detail", f.contract(root, id, ""), 503, "dependency_unavailable")
	f.code(t, "empty evidence", f.run(root, "contract.evidence", id, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f"}), 503, "dependency_unavailable")
}

func TestContractHintMutantsKilled(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := "sample-contract-01"
	end := ok(t, "end", f.run(root, "contract.end", id, map[string]string{"from": "2026-11-16"}))["event"].(string)
	ok(t, "secured", f.run(root, "contract.evidence", id, map[string]string{"kind": "secured", "reference": "r1", "covered_until": "2026-11-16"}))
	ok(t, "withdraw", f.run(root, "contract.withdraw", id, map[string]string{"event": end, "cancel": "contract_continues"}))
	ok(t, "end again", f.run(root, "contract.end", id, map[string]string{"from": "2026-11-17"}))
	f.advance(24 * time.Hour)
	root = f.as(AdminLogin)
	if a := actionOf(f.contract(root, id, ""), "contract.settle"); a["blocked"] != "evidence_needed" {
		t.Errorf("batch fence %+v", a)
	}
	f.code(t, "stale evidence", f.run(root, "contract.settle", id, map[string]string{"as_of": "2026-11-17"}), 422, "prerequisite_failed")
	f.code(t, "no reason", f.run(root, "contract.reopen", "sample-contract-04", map[string]string{}), 422, "invalid_request")
	ok(t, "end 02", f.run(root, "contract.end", "sample-contract-02", map[string]string{"from": "2026-11-17"}))
	ok(t, "secured 02", f.run(root, "contract.evidence", "sample-contract-02", map[string]string{"kind": "secured", "reference": "r2", "covered_until": "2026-11-17"}))
	ok(t, "settle 02", f.run(root, "contract.settle", "sample-contract-02", map[string]string{"as_of": "2026-11-17"}))
	settled := f.contract(root, "sample-contract-02", "")
	for _, e := range settled.body["events"].([]any) {
		if row := e.(map[string]any); row["kind"] == "end" && (row["withdraw_blocked"] != "settled" || row["cancel_required"] != false) {
			t.Errorf("settled event %+v", row)
		}
	}

	g := setup(t, SeedSample).ready()
	admin := g.as(AdminLogin)
	g.s.lifecycleLimits = contracts.Limits{Contracts: 10, Writes: 3, Reserve: 2, Batch: 8}
	small, _ := contracts.NewRegistry(g.s.lifecycleLimits, g.s.clock)
	g.s.lifecycles = small
	target := "sample-contract-02"
	events := []contracts.Event{{ID: "ev-s", Kind: contracts.Start, From: contractDay(2026, 1, 1), Revision: 1}, {ID: "ev-e", Kind: contracts.End, From: contractDay(2026, 11, 16)}}
	if _, err := small.Amend(SampleWorkspace, target, 0, "cw-s", nil, events, ""); err != nil {
		t.Fatal(err)
	}
	gap := ok(t, "gap", g.run(admin, "contract.evidence", target, map[string]string{"kind": "missing", "reason": "not_attempted", "follow_up": "f"}))["write"].(string)
	ok(t, "resolve", g.run(admin, "contract.evidence", target, map[string]string{"kind": "secured", "reference": "r", "covered_until": "2026-11-16", "resolves": gap}))
	full := g.contract(admin, target, "")
	for _, command := range []string{"contract.settle", "contract.evidence.secured"} {
		if a := actionOf(full, command); a["blocked"] != "write_limit" {
			t.Errorf("%s %+v", command, a)
		}
	}
	g.code(t, "settle at limit", g.run(admin, "contract.settle", target, map[string]string{"as_of": "2026-11-16"}), 422, "prerequisite_failed")
	limitFar := g.contract(admin, target, "")
	for _, e := range limitFar.body["events"].([]any) {
		if row := e.(map[string]any); row["kind"] == "end" && (row["withdraw_blocked"] != "write_limit" || row["cancel_required"] != false) {
			t.Errorf("limit event %+v", row)
		}
	}
}
