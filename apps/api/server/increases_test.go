package server

import (
	"bytes"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) suspendedMember(t *testing.T) actor {
	t.Helper()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "suspend", f.run(root, "identity.suspend_user", SampleMember, nil), "user_suspended", "")
	return root
}

func TestARefusedIncreaseLeavesNoRecord(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	before, _ := readLedger(f)
	f.failed(t, "reinstating an active user", f.run(root, "identity.reinstate_user", SampleMember, nil), commands.InvalidStage)
	f.failed(t, "reinstating an active membership", f.run(root, "identity.reinstate_membership", SampleMember, nil), commands.InvalidStage)
	if after, _ := readLedger(f); !bytes.Equal(before, after) {
		t.Error("a refused increase wrote to the ledger")
	}
}

func TestIncreaseIntentApplyConfirm(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.suspendedMember(t)
	f.ok(t, "reinstate", f.run(root, "identity.reinstate_user", SampleMember, nil), "user_active", "map[durable:yes ledger:confirmed]")
	intents := f.records(t, decisionledger.IncreaseIntent)
	if len(intents) != 1 || intents[0]["increase"] != decisionledger.IncreaseReinstateUser || intents[0]["target_version"] != 2.0 {
		t.Errorf("intent %+v", intents)
	}
	if len(f.records(t, decisionledger.IncreaseConfirmed)) != 1 {
		t.Error("confirm")
	}
}

func TestIncreaseFaults(t *testing.T) {
	for name, c := range map[string]struct {
		fault   func(*int) func(string) bool
		status  int
		code    string
		durable string
		applied bool
		state   string
	}{
		"intent not confirmed": {func(n *int) func(string) bool {
			return func(at string) bool { return at == "ledger-sync" }
		}, 503, string(commands.DependencyUnavailable), "", false, "stopped"},
		"refused apply": {func(n *int) func(string) bool { return func(at string) bool { return at == "increase-refuse" } }, 422, "", "", false, "ok"},
		"failed apply":  {func(n *int) func(string) bool { return func(at string) bool { return at == "increase-apply" } }, 500, string(commands.InternalError), "", false, "stopped"},
		"panic":         {func(n *int) func(string) bool { return func(at string) bool { return at == "increase-panic" } }, 500, string(commands.InternalError), "", false, "stopped"},
		"confirm refused": {func(n *int) func(string) bool { return func(at string) bool { return at == "increase-confirm" } },
			200, "", durableNo, true, "stopped"},
		"confirm unknown": {func(n *int) func(string) bool {
			return func(at string) bool {
				if at == "ledger-sync" {
					*n++
					return *n == 2
				}
				return false
			}
		}, 200, "", durableUnknown, true, "stopped"},
	} {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.suspendedMember(t)
			count := 0
			f.s.fault = c.fault(&count)
			got := f.run(root, "identity.reinstate_user", SampleMember, nil)
			f.s.fault = nil
			if got.status != c.status || c.code != "" && got.body["code"] != c.code || c.durable != "" && field(got, "durable") != c.durable {
				t.Fatalf("%s", got.raw)
			}
			if _, state, _, _ := f.s.identity.Subjects().Subject(SampleMember); (state == identity.Active) != c.applied {
				t.Errorf("applied %s", state)
			}
			if st := f.s.ledger.Status().State; string(st) != c.state {
				t.Errorf("ledger %s", st)
			}
			if name == "refused apply" && len(f.records(t, decisionledger.IncreaseAborted)) != 1 {
				t.Error("no abort record")
			}
			g := f.restart(nil)
			if _, state, _, _ := g.s.identity.Subjects().Subject(SampleMember); state != identity.Active && name != "confirm unknown" {
				t.Errorf("after restart the seed state is active, got %s", state)
			}
		})
	}
}

func TestOneInstantForTheDryRunAndTheApply(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "an end", f.period(root, SampleMember, f.at(2*time.Minute)), "role_period_set", "")
	moved := false
	f.s.fault = func(at string) bool {
		if at == "increase-apply" {
			return false
		}
		if at == "ledger-sync" && !moved {
			moved = true
			f.advance(time.Hour)
		}
		return false
	}
	got := f.period(root, SampleMember, f.at(time.Minute*3))
	f.s.fault = nil
	if got.status != 200 {
		t.Fatalf("an increase judged at its first instant %s", got.raw)
	}
}
