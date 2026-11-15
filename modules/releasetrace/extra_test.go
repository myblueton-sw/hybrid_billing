package releasetrace

import "testing"

func TestCoverageEdges(t *testing.T) {
	for _, s := range []string{"1.0.0", "0.10.999999999"} {
		if !semver(s) {
			t.Errorf("semver %s", s)
		}
	}
	for _, s := range []string{"01.0.0", "1.0", "1.a.0", "1..0", "1.0.1234567890", "1.0.0.0"} {
		if semver(s) {
			t.Errorf("semver %s accepted", s)
		}
	}
	f := setup(t)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("R1", false, TestRef{ID: "t1"})})
	f.by(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b")[:40]}}, eve, fay)
	f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h", Identity: "r"}}, bob, gus)
	reason(t, "requirement approver", second(f.r.Record(Result{Namespace: "v1", Test: "t1", Build: "1.0.0", Passed: true, Evidence: sum("e")}, Actor{ID: "r", Kind: KindSystem})), ReasonSelfVerification)
	g := standard(t)
	for _, id := range []string{"B2", "B1"} {
		b := req(id, false, TestRef{ID: "x" + id})
		b.Namespace, b.Blocking = "other", true
		g.approve(Change{Kind: ChangeRequirement, Requirement: b})
	}
	g.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: Key{"other", "B2"}, Reason: "x"}})
	g.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: Key{"other", "B1"}, Reason: "x"}})
	v, _ := g.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("R1")}, Paths: []string{"p1"}})
	if len(v.RetiredBlocking) != 2 || v.RetiredBlocking[0].ID != "B1" || v.RetiredBlocking[1].ID != "B2" {
		t.Errorf("retired order %+v", v.RetiredBlocking)
	}
}
