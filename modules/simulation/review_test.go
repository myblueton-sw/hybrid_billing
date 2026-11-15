package simulation

import (
	"strconv"
	"sync"
	"testing"
)

func TestHarnessKindAndCrossRoles(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	x := Run{Scenario: "s1", Engine: "1.0.0", Rules: sum("r"), Output: sum("o")}
	reason(t, "human with bound id", second(f.r.Record(x, Actor{ID: runner.ID, Kind: KindHuman})), ReasonNotHarness)
	reason(t, "assistant with bound id", second(f.r.Record(x, Actor{ID: runner.ID, Kind: KindAssistant})), ReasonAINotAllowed)
	reason(t, "unbound before lookup", second(f.r.Record(Run{Scenario: "nope", Engine: "1.0.0", Rules: sum("r"), Output: sum("o")}, other)), ReasonNotHarness)
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: "runner-2"}}, bob, cat)
	reason(t, "proposer is expectation approver", second(f.r.Record(x, other)), ReasonSelfVerification)
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h3", Identity: "runner-3"}}, cat, ann)
	reason(t, "approver is expectation author", second(f.r.Record(x, Actor{ID: "runner-3", Kind: KindSystem})), ReasonSelfVerification)
	if c, _ := f.r.Current("s1"); c.Expected != 1 {
		t.Errorf("expected version %+v", c)
	}
}

func TestEvidenceAgainstRunOutputs(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	fresh := billing("s10")
	fresh.Events = sum("regenerated")
	f.approve(Change{Kind: ChangeScenario, Scenario: fresh})
	pending, err := f.r.Propose(Change{Kind: ChangeExpectation, Expectation: Expectation{Scenario: "s10", Version: 1, Method: MethodIndependent, Evidence: sum("later"), Checks: standardChecks()}}, ann)
	f.must(err)
	f.approve(Change{Kind: ChangeTarget, Target: Target{ID: "aws", Kind: TargetCSP, Scenarios: []string{"s1"}}})
	proof, err := f.r.Propose(Change{Kind: ChangeProof, Proof: Proof{Target: "aws", Environment: "prod", Evidence: sum("later")}}, cat)
	f.must(err)
	f.record("s1", "1.0.0", "r", "later", nil)
	reason(t, "expectation approval", second(f.r.Approve(pending.Number, bob)), ReasonRunOutput)
	reason(t, "proof approval", second(f.r.Approve(proof.Number, dan)), ReasonRunOutput)
	reason(t, "expectation proposal", f.try(Change{Kind: ChangeExpectation, Expectation: Expectation{Scenario: "s10", Version: 1, Method: MethodManual, Evidence: sum("later"), Checks: standardChecks()}}), ReasonRunOutput)
}

func TestConcurrentProposals(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	var wg sync.WaitGroup
	for index := 0; index < 6; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for n := 0; n < 10; n++ {
				p, err := f.r.Propose(Change{Kind: ChangeProfile, Profile: Profile{ID: "c" + strconv.Itoa(index) + "-" + strconv.Itoa(n)}}, ann)
				if err != nil {
					t.Error(err)
					return
				}
				if _, err := f.r.Approve(p.Number, bob); err != nil {
					t.Error(err)
				}
				f.r.Record(Run{Scenario: "s1", Engine: "1.0.0", Rules: sum("r"), Output: sum("o")}, runner)
				f.r.Matrix("p1")
			}
		}(index)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		f.r.RevokeHarness("h1", cat)
	}()
	wg.Wait()
	if len(f.r.profiles) != 61 || !f.r.harnesses["h1"].revoked {
		t.Errorf("profiles %d", len(f.r.profiles))
	}
}
