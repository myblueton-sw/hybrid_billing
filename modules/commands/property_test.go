package commands

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type modelOperation struct {
	target      string
	expected    string
	amount      string
	user        string
	description string
	code        Code
	operation   string
	revision    string
}

func TestCallsAgreeWithAReferenceModel(t *testing.T) {
	r := rand.New(rand.NewSource(176))
	reached := map[string]int{}
	users := []string{"u0", "u1", "u2"}
	targets := map[string]string{"t0": "c0", "t1": "c0", "t2": "c1"}
	names := []string{"t0", "t1", "t2", "t9"}
	for history := 0; history < 60; history++ {
		f := setup(t)
		f.h.targets = map[string]*Target{}
		revisions := map[string]string{}
		for target, customer := range targets {
			f.h.targets["w1/"+target] = &Target{CustomerID: customer, Revision: "r1", AuthorID: "u0"}
			revisions[target] = "r1"
		}
		issue, view := map[string]string{}, map[string]string{}
		for _, user := range users {
			f.member(user)
		}
		operations := map[string]*modelOperation{}
		applies, calls := 0, 0
		for step := 0; step < 150; step++ {
			user := users[r.Intn(len(users))]
			retry := ""
			if len(operations) > 0 && r.Intn(5) == 0 {
				keys := make([]string, 0, len(operations))
				for key := range operations {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				retry = keys[r.Intn(len(keys))]
				user = operations[retry].user
			}
			switch r.Intn(10) {
			case 0:
				customer := []string{"c0", "c1"}[r.Intn(2)]
				if _, held := issue[user+"/"+customer]; !held {
					issue[user+"/"+customer] = f.grant(user, gate.Issue, customer)
				} else if r.Intn(3) == 0 {
					f.must(f.a.RevokeGrant(issue[user+"/"+customer], 1))
					delete(issue, user+"/"+customer)
					reached["revoked"]++
				}
				continue
			case 1:
				customer := []string{"c0", "c1"}[r.Intn(2)]
				if _, held := view[user+"/"+customer]; !held {
					view[user+"/"+customer] = f.grant(user, gate.View, customer, "quantity")
				}
				continue
			}
			mode := []string{"normal", "normal", "normal", "normal", "named", "unknown"}[r.Intn(6)]
			switch mode {
			case "named":
				f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(InvalidStage) }
			case "unknown":
				f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, errors.New("synthetic") }
			default:
				f.h.apply = nil
			}
			target := names[r.Intn(len(names))]
			if retry != "" {
				target = operations[retry].target
			}
			customer, exists := targets[target]
			answer := f.answer(user)
			calls++
			if retry == "" && r.Intn(4) == 0 {
				got := f.c.Execute(Call{Answer: answer, Request: "r", Command: "statement.read", Target: target})
				_, permitted := view[user+"/"+customer]
				want := Code("")
				switch {
				case !exists || !permitted:
					want = PermissionDenied
				default:
					applies++
					if mode == "named" {
						want = InvalidStage
					} else if mode == "unknown" {
						want = InternalError
					}
				}
				if (got.Err == nil) != (want == "") || (got.Err != nil && got.Err.Code != want) {
					t.Fatalf("history %d step %d: read of %s by %s: %+v; model %q", history, step, target, user, got, want)
				}
				if want == "" && (got.Fields["revision_read"] != revisions[target] || got.Revision != revisions[target] || got.Operation != "") {
					t.Fatalf("history %d step %d: read result %+v; model revision %s", history, step, got, revisions[target])
				}
				reached["read "+string(want)]++
			} else {
				key := fmt.Sprintf("k%d", r.Intn(40))
				amount := []string{"1", "2"}[r.Intn(2)]
				expected := revisions[target]
				if !exists || r.Intn(6) == 0 {
					expected = "r0"
				}
				if retry != "" {
					key, amount, expected = retry, operations[retry].amount, operations[retry].expected
				}
				got := f.c.Execute(Call{Answer: answer, Request: "r", Command: "statement.adjust", Target: target, Key: key, Expected: expected, Rationale: "synthetic",
					Fields: []Value{{"amount", amount}}})
				description := target + "|" + expected + "|" + amount
				_, permitted := issue[user+"/"+customer]
				stored := operations[key]
				want, replay := Code(""), false
				switch {
				case !exists || !permitted:
					want = PermissionDenied
				case stored != nil && (stored.user != user || stored.description != description):
					want = IdempotencyConflict
				case stored != nil:
					want, replay = stored.code, true
				case expected != revisions[target]:
					want = RevisionConflict
				default:
					applies++
					switch mode {
					case "named":
						want = InvalidStage
					case "unknown":
						want = InternalError
						operations[key] = &modelOperation{target: target, expected: expected, amount: amount, user: user, description: description, code: InternalError, operation: got.operationID()}
					default:
						revisions[target] = fmt.Sprintf("r%d", applies+1)
						operations[key] = &modelOperation{target: target, expected: expected, amount: amount, user: user, description: description, operation: got.operationID(), revision: revisions[target]}
					}
				}
				if (got.Err == nil) != (want == "") || (got.Err != nil && got.Err.Code != want) || got.Replay != replay {
					t.Fatalf("history %d step %d: adjust of %s by %s with %s: %+v %+v; model %q replay %v", history, step, target, user, key, got, got.Err, want, replay)
				}
				if replay && (got.operationID() != stored.operation || (want == "" && got.Revision != stored.revision)) {
					t.Fatalf("history %d step %d: a replay returned another operation: %+v; model %+v", history, step, got, stored)
				}
				if !replay && want == "" && (got.Revision != revisions[target] || got.Fields["echo"] != amount || got.operationID() == "") {
					t.Fatalf("history %d step %d: result %+v; model revision %s", history, step, got, revisions[target])
				}
				label := "adjust " + string(want)
				if replay {
					label += " replay"
				}
				reached[label]++
			}
			if _, got := f.h.count(); got != applies {
				t.Fatalf("history %d step %d: %d applies; model %d", history, step, got, applies)
			}
			if records := f.c.Journal(0, 1000000); len(records) != calls {
				t.Fatalf("history %d step %d: %d records for %d admitted calls", history, step, len(records), calls)
			}
			f.c.mu.Lock()
			held := len(f.c.operations)
			f.c.mu.Unlock()
			if held != len(operations) {
				t.Fatalf("history %d step %d: %d stored operations; model %d", history, step, held, len(operations))
			}
		}
	}
	for _, category := range []string{"read ", "read permission_denied", "read invalid_stage", "read internal_error", "adjust ", "adjust  replay", "adjust permission_denied",
		"adjust idempotency_conflict", "adjust revision_conflict", "adjust invalid_stage", "adjust internal_error", "adjust internal_error replay", "revoked"} {
		if reached[category] < 20 {
			t.Errorf("the generator reached %q only %d times", category, reached[category])
		}
	}
	t.Logf("%v", reached)
}

func (r Result) operationID() string {
	if r.Err != nil {
		return r.Err.Operation
	}
	return r.Operation
}
