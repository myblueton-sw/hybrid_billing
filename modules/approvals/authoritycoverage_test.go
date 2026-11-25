package approvals

import (
	"reflect"
	"testing"
	"time"
)

func coverageRegistry(t *testing.T) (*Registry, *testClock) {
	t.Helper()
	c := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), testLimits(), c.read)
	if err != nil {
		t.Fatal(err)
	}
	must(r.Open("ws", []string{"a", "b"}, human("i")))
	for _, a := range []Authority{
		{ID: "owned", Approver: "ann", Stage: StageERPFinal, Currency: "USD", Limit: "10.00000001", From: "2026-11-14", To: "2026-11-15"},
		{ID: "larger", Approver: "ann", Stage: StageERPFinal, Currency: "USD", Limit: "100", From: "2026-11-14", To: "2026-11-15"},
	} {
		must(r.Propose("ws", Proposal{ID: a.ID, Kind: ProposeGrant, Authority: a}, human("a")))
		must(r.Confirm("ws", a.ID, human("b")))
	}
	return r, c
}

func TestNamedAuthorityCoverageExactAndIsolated(t *testing.T) {
	r, _ := coverageRegistry(t)
	before := must(r.AuthorityOf("ws", "owned"))
	for _, amount := range []string{"0", "-0", "10.00000001", "-10.00000001", "10.00000000", "9.99999999"} {
		expect(t, "ok", r.CheckAuthorityCoverage("ws", "owned", "ann", StageERPFinal, "USD", amount), amount)
	}
	for _, amount := range []string{"10.00000002", "-10.00000002", "99"} {
		expect(t, ReasonNoAuthority, r.CheckAuthorityCoverage("ws", "owned", "ann", StageERPFinal, "USD", amount), amount)
	}
	expect(t, "ok", r.CheckAuthorityCoverage("ws", "larger", "ann", StageERPFinal, "USD", "99"), "larger exists separately")
	for _, tc := range []struct{ workspace, id, actor, stage, currency, want string }{
		{"ws", "owned", "bob", StageERPFinal, "USD", ReasonNoAuthority},
		{"ws", "owned", "ann", StageCharge, "USD", ReasonNoAuthority},
		{"ws", "owned", "ann", StageERPFinal, "EUR", ReasonNoAuthority},
		{"foreign", "owned", "ann", StageERPFinal, "USD", "not_found"},
		{"ws", "missing", "ann", StageERPFinal, "USD", "not_found"},
		{"", "owned", "ann", StageERPFinal, "USD", "invalid"},
		{"ws", "", "ann", StageERPFinal, "USD", "invalid"},
		{"ws", "owned", "bad actor", StageERPFinal, "USD", "invalid"},
		{"ws", "owned", "ann", "unknown", "USD", "invalid"},
		{"ws", "owned", "ann", StageERPFinal, "usd", "invalid"},
	} {
		expect(t, tc.want, r.CheckAuthorityCoverage(tc.workspace, tc.id, tc.actor, tc.stage, tc.currency, "1"), tc.want)
	}
	for _, amount := range []string{"", "1e2", "NaN", "999999999999999", "0.000000001"} {
		expect(t, "invalid", r.CheckAuthorityCoverage("ws", "owned", "ann", StageERPFinal, "USD", amount), amount)
	}
	after := must(r.AuthorityOf("ws", "owned"))
	if !reflect.DeepEqual(before, after) || len(r.workspaces["ws"].authorities) != 2 || len(r.workspaces["ws"].requests) != 0 || len(r.workspaces["ws"].proposals) != 2 {
		t.Fatal("coverage changed authority/request state")
	}
}

func TestNamedAuthorityCoverageDatesRevocationAndRollback(t *testing.T) {
	r, c := coverageRegistry(t)
	check := func(want string) {
		t.Helper()
		expect(t, want, r.CheckAuthorityCoverage("ws", "owned", "ann", StageERPFinal, "USD", "1"), want)
	}
	check("ok")
	c.advance(24 * time.Hour)
	check("ok")
	c.advance(24 * time.Hour)
	check(ReasonNoAuthority)
	c.advance(-48 * time.Hour)
	check(ReasonNoAuthority)
	r, _ = coverageRegistry(t)
	must(r.Revoke("ws", "owned", human("a")))
	check(ReasonNoAuthority)
	r, _ = coverageRegistry(t)
	future := Authority{ID: "future", Approver: "ann", Stage: StageERPFinal, Currency: "USD", Limit: "10", From: "2026-11-15", To: "2026-11-15"}
	must(r.Propose("ws", Proposal{ID: "future", Kind: ProposeGrant, Authority: future}, human("a")))
	must(r.Confirm("ws", "future", human("b")))
	expect(t, ReasonNoAuthority, r.CheckAuthorityCoverage("ws", "future", "ann", StageERPFinal, "USD", "1"), "not yet valid")
}
