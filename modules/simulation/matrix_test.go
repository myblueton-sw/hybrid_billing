package simulation

import (
	"testing"
	"time"
)

func tested(event string, checks ...string) Change {
	return Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: event, Coverage: CoverageTested, Scenario: "s1", Checks: checks}}
}

func status(t testing.TB, f world, event string) CellView {
	t.Helper()
	cells, err := f.r.Matrix("p1")
	f.must(err)
	return cells[eventIndex(event)]
}

func TestMatrixLinks(t *testing.T) {
	f := standard(t)
	reason(t, "no expectation", f.try(tested(EventOverage, "krw-total")), ReasonNoExpectation)
	f.expect("s1", standardChecks()...)
	f.expect("f1", Check{Name: "rows", Kind: CheckCount, Value: "3"})
	reason(t, "not covered", f.try(tested(EventLateCredit, "krw-total")), ReasonNotCovered)
	reason(t, "unknown check", f.try(tested(EventOverage, "nope")), ReasonNotCovered)
	reason(t, "no amount", f.try(tested(EventOverage, "lines")), ReasonNoAmount)
	f.approve(tested(EventPartialCSVError, "lines"))
	missing := tested(EventOverage, "krw-total")
	missing.Cell.Scenario = "nope"
	expect(t, "scenario missing", f.try(missing), ErrNotFound)
	formatCell := tested(EventOverage, "rows")
	formatCell.Cell.Scenario = "f1"
	reason(t, "format", f.try(formatCell), ReasonWrongPurpose)
	f.approve(Change{Kind: ChangeProfile, Profile: Profile{ID: "p2"}})
	otherProfile := tested(EventOverage, "krw-total")
	otherProfile.Cell.Profile = "p2"
	reason(t, "other profile", f.try(otherProfile), ReasonNotCovered)
	undecided := Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventLateCredit, Coverage: CoverageUndecided, Decision: "owner-fx", Scenario: "s1", Checks: []string{"krw-total"}}}
	reason(t, "undecided link checked", f.try(undecided), ReasonNotCovered)
	unsupported := Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventFXChange, Coverage: CoverageUnsupported, Dataset: "syn", Evidence: sum("limit")}}
	reason(t, "synthetic source", f.try(unsupported), ReasonWrongOrigin)
	unsupported.Cell.Dataset = "nope"
	expect(t, "dataset missing", f.try(unsupported), ErrNotFound)
	expect(t, "matrix missing", second(f.r.Matrix("nope")), ErrNotFound)
	expect(t, "profile repeat", f.try(Change{Kind: ChangeProfile, Profile: Profile{ID: "p1"}}), ErrConflict)
}

func TestMatrixStatuses(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	f.approve(tested(EventOverage, "krw-total", "usd-total"))
	f.approve(tested(EventPartialCSVError, "lines"))
	f.approve(Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventFXChange, Coverage: CoverageUnsupported, Dataset: "pub", Evidence: sum("limit")}})
	f.approve(Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventLateCredit, Coverage: CoverageUndecided, Decision: "owner-credit"}})
	f.approve(Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventRefund, Coverage: CoverageUndecided, Decision: "owner-refund", Scenario: "s1", Checks: []string{"krw-total"}}})
	check := func(name string, want map[string]string) {
		t.Helper()
		for event, s := range want {
			if got := status(t, f, event); got.Status != s {
				t.Errorf("%s %s = %s", name, event, got.Status)
			}
		}
	}
	check("before runs", map[string]string{EventOverage: CellPending, EventFXChange: CellUnsupported, EventLateCredit: CellUndecided, EventRefund: CellUndecided, EventNoUpfront: CellOpen})
	good := results(standardChecks())
	f.record("s1", "1.0.0", "r", "o", good)
	check("single", map[string]string{EventOverage: CellPending})
	f.record("s1", "1.0.0", "r", "o", good)
	check("reproduced", map[string]string{EventOverage: CellPassed, EventPartialCSVError: CellPassed, EventRefund: CellUndecided})
	if v := status(t, f, EventOverage); v.Engine != "1.0.0" || v.Rules != sum("r") {
		t.Errorf("engine %+v", v)
	}
	if v := status(t, f, EventFXChange); v.Engine != "" {
		t.Errorf("unsupported engine %+v", v)
	}
	summary := f.r.Summary()
	want := map[string]int{CellPassed: 2, CellUnsupported: 1, CellUndecided: 2, CellOpen: 13, CellFailed: 0, CellPending: 0}
	for key, n := range want {
		if summary[key] != n {
			t.Errorf("summary %s = %d", key, summary[key])
		}
	}
	bad := results(standardChecks())
	bad[0].Value = "7"
	f.record("s1", "1.0.0", "r", "o", bad)
	check("non reproducible", map[string]string{EventOverage: CellFailed})
	f.record("s1", "2.0.0", "r", "z", bad)
	check("fail", map[string]string{EventOverage: CellFailed})
	f.approve(Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventOverage, Coverage: CoverageUndecided, Decision: "owner-overage"}})
	f.approve(Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventOverage, Coverage: CoverageUnsupported, Dataset: "conv", Evidence: sum("x")}})
	v := status(t, f, EventOverage)
	if v.Status != CellUnsupported || v.Revisions != 2 || v.Failures != 1 || len(v.History) != 2 || v.History[0].Status != CellFailed || v.History[0].Replaced.Coverage != CoverageTested ||
		v.History[1].Status != CellUndecided || v.History[0].Replaced.Checks[0] != "krw-total" {
		t.Errorf("history %+v", v)
	}
	v.History[0].Replaced.Checks[0] = "mutated"
	v.Cell.Event = "mutated"
	if again := status(t, f, EventOverage); again.History[0].Replaced.Checks[0] != "krw-total" {
		t.Error("matrix read shares memory")
	}
	for index := 0; index < maxHistory+2; index++ {
		f.approve(Change{Kind: ChangeCell, Cell: Cell{Profile: "p1", Event: EventOverage, Coverage: CoverageUndecided, Decision: "d" + itoa(index)}})
		if v := status(t, f, EventOverage); len(v.History) != min(v.Revisions, maxHistory) {
			t.Fatalf("history %d after %d", len(v.History), v.Revisions)
		}
	}
	if v := status(t, f, EventOverage); len(v.History) != maxHistory || v.Failures != 1 || v.Revisions != maxHistory+4 || v.History[maxHistory-1].Replaced.Decision != "d"+itoa(maxHistory) {
		t.Errorf("rolled history %d %d %d", len(v.History), v.Failures, v.Revisions)
	}
	if v := status(t, f, EventNoUpfront); v.History == nil || len(v.History) != 0 {
		t.Error("empty history")
	}
}

func TestPendingLinkMissing(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	f.approve(tested(EventOverage, "krw-total"))
	f.expect("s1", amount("krw-sum", "1", "KRW"))
	good := []Result{{Name: "krw-sum", Value: "1", Currency: "KRW"}}
	f.record("s1", "1.0.0", "r", "o", good)
	f.record("s1", "1.0.0", "r", "o", good)
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictPass {
		t.Fatalf("current %+v", c)
	}
	if v := status(t, f, EventOverage); v.Status != CellPending {
		t.Errorf("link missing %+v", v)
	}
}

func TestProofAndDemo(t *testing.T) {
	f := standard(t)
	expect(t, "scenario missing", f.try(Change{Kind: ChangeTarget, Target: Target{ID: "aws", Kind: TargetCSP, Scenarios: []string{"nope"}}}), ErrNotFound)
	f.approve(Change{Kind: ChangeTarget, Target: Target{ID: "aws", Kind: TargetCSP, Scenarios: []string{"f1", "s1"}}})
	expect(t, "repeat", f.try(Change{Kind: ChangeTarget, Target: Target{ID: "aws", Kind: TargetCSP, Scenarios: []string{"s1"}}}), ErrConflict)
	expect(t, "status missing", second(f.r.Status("nope")), ErrNotFound)
	s, err := f.r.Status("aws")
	if err != nil || s.Demo != DemoPending || s.Integration != IntegrationNotProven || s.Target.Scenarios[1] != "s1" {
		t.Errorf("initial %+v %v", s, err)
	}
	f.expect("s1", standardChecks()...)
	f.expect("f1", Check{Name: "rows", Kind: CheckCount, Value: "3"})
	rows := []Result{{Name: "rows", Value: "3"}}
	good := results(standardChecks())
	for index := 0; index < 2; index++ {
		f.record("s1", "1.0.0", "r", "o", good)
	}
	if s, _ := f.r.Status("aws"); s.Demo != DemoPending {
		t.Errorf("partial demo %+v", s)
	}
	for index := 0; index < 2; index++ {
		f.record("f1", "1.0.0", "r", "rows", rows)
	}
	if s, _ := f.r.Status("aws"); s.Demo != DemoPassed || s.Integration != IntegrationNotProven {
		t.Errorf("demo pass leaks %+v", s)
	}
	for _, ev := range []string{sum("o"), sum("sheet-s11")} {
		reason(t, "run output", f.try(Change{Kind: ChangeProof, Proof: Proof{Target: "aws", Environment: "prod-1", Evidence: ev}}), ReasonRunOutput)
	}
	f.advance(time.Hour)
	f.approveBy(Change{Kind: ChangeProof, Proof: Proof{Target: "aws", Environment: "prod-1", Evidence: sum("conn")}}, cat, dan)
	s, _ = f.r.Status("aws")
	if s.Integration != IntegrationUnclassified || s.Proof.Environment != "prod-1" || s.Proof.Evidence != sum("conn") || s.Proof.Approver != "dan" || s.Proof.Count != 1 || s.Demo != DemoPassed ||
		!s.Proof.At.Equal(start.Add(time.Hour)) {
		t.Errorf("proof %+v", s)
	}
	f.approve(Change{Kind: ChangeProof, Proof: Proof{Target: "aws", Environment: "prod-2", Evidence: sum("conn2")}})
	bad := results(standardChecks())
	bad[0].Value = "2"
	f.record("s1", "1.0.0", "r", "o", bad)
	s, _ = f.r.Status("aws")
	if s.Proof.Count != 2 || s.Proof.Environment != "prod-2" || s.Demo != DemoFailed || s.Integration != IntegrationUnclassified {
		t.Errorf("replaced %+v", s)
	}
	s.Target.Scenarios[0] = "mutated"
	if again, _ := f.r.Status("aws"); again.Target.Scenarios[0] != "f1" {
		t.Error("status read shares memory")
	}
	g := standard(t)
	g.approve(Change{Kind: ChangeTarget, Target: Target{ID: "erp", Kind: TargetERP, Scenarios: []string{"f1", "s1"}}})
	g.expect("s1", standardChecks()...)
	g.expect("f1", Check{Name: "rows", Kind: CheckCount, Value: "3"})
	g.record("f1", "1.0.0", "r", "rows", []Result{{Name: "rows", Value: "4"}})
	if s, _ := g.r.Status("erp"); s.Demo != DemoFailed {
		t.Errorf("failure before pending %+v", s)
	}
}
