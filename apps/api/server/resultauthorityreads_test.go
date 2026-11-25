package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func resultAuthorityReadCall(f world, a actor, kind string, fields map[string]string) reply {
	if fields == nil {
		fields = map[string]string{}
	}
	return documentRead(f, a, "result_authority."+kind, SampleWorkspace, fields)
}
func TestResultAuthorityReadContext(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	_ = f.as(MemberLogin)
	root := f.as(AdminLogin)
	fields := map[string]string{"collection": "recipients", "capability": "apply"}
	got := resultAuthorityReadCall(f, root, "context", fields)
	if got.status != 200 || field(got, "current_eligibility") != "unchecked" || !validPlanHash(field(got, "snapshot")) {
		t.Fatal(got.raw)
	}
	found := false
	for i := 0; i < 10; i++ {
		raw := field(got, fmt.Sprintf("row_%02d", i))
		if raw == "" {
			break
		}
		var row map[string]string
		f.must(json.Unmarshal([]byte(raw), &row))
		if row["user"] == SampleAdmin {
			t.Fatal("proposer offered")
		}
		if row["user"] == SampleMember {
			found = true
		}
	}
	if !found {
		t.Fatal("ordinary human omitted")
	}
	empty := resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"})
	if empty.status != 200 || field(empty, "row_00") != "" {
		t.Fatal(empty.raw)
	}
	approved := provisionResultCeiling(t, f)
	got = resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"})
	var ceiling map[string]string
	f.must(json.Unmarshal([]byte(field(got, "row_00")), &ceiling))
	if ceiling["ceiling"] != field(approved, "grant") || ceiling["capability"] != "apply" {
		t.Fatal(got.raw)
	}
	other := resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "inspect"})
	if other.status != 200 || field(other, "row_00") != "" {
		t.Fatal(other.raw)
	}
	f.code(t, "cross capability cursor", resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "inspect", "after": ceiling["ceiling"], "snapshot": field(got, "snapshot")}), 409, string(commands.EvidenceStale))
}
func TestResultAuthorityHistoryExpirySnapshot(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for i := 0; i < 11; i++ {
		f.ok(t, "proposal", f.run(root, "result_authority.propose_console", fmt.Sprintf("proposal-%02d", i), resultPermissionTimes(f)), inputauthority.Pending, "")
	}
	before := f.s.Revision()
	first := resultAuthorityReadCall(f, root, "history", nil)
	if first.status != 200 || field(first, "next") != "proposal-09" {
		t.Fatal(first.raw)
	}
	fields := map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")}
	last := resultAuthorityReadCall(f, root, "history", fields)
	if last.status != 200 || field(last, "row_00") == "" || field(last, "row_01") != "" || field(last, "next") != "" {
		t.Fatal(last.raw)
	}
	f.advance(2 * time.Hour)
	root = f.as(AdminLogin)
	if f.s.Revision() != before {
		t.Fatal("clock unexpectedly changed service revision")
	}
	f.code(t, "expired projection continuation", resultAuthorityReadCall(f, root, "history", fields), 409, string(commands.EvidenceStale))
	fresh := resultAuthorityReadCall(f, root, "history", nil)
	if fresh.status != 200 || field(fresh, "snapshot") == field(first, "snapshot") {
		t.Fatal(fresh.raw)
	}
	var row map[string]string
	f.must(json.Unmarshal([]byte(field(fresh, "row_00")), &row))
	if row["state"] != inputauthority.Expired {
		t.Fatal(row)
	}
	if f.s.Revision() != before {
		t.Fatal("read changed service revision")
	}
}
func TestResultAuthorityReadFailures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	f.code(t, "ordinary human cannot enumerate", resultAuthorityReadCall(f, member, "history", nil), 403, string(commands.PermissionDenied))
	f.code(t, "invalid capability", resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "all"}), 422, string(commands.InvalidRequest))
	f.code(t, "unpaired cursor", resultAuthorityReadCall(f, root, "history", map[string]string{"after": "missing"}), 409, string(commands.EvidenceStale))
	provisionResultCeiling(t, f)
	f.s.fault = func(at string) bool { return at == "result-authority-lookup" }
	f.code(t, "unavailable current effect", resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"}), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	f.s.resultAuthority.ceilings["orphan"] = inputauthority.Spec{}
	f.code(t, "orphan ceiling", resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"}), 409, string(commands.EvidenceStale))
}

func TestResultAuthorityReadBoundsAndUnknown(t *testing.T) {
	t.Run("recipient limit", func(t *testing.T) {
		f := setup(t, SeedSample).ready()
		p := newPopulation(f)
		for i := 0; i < 1001; i++ {
			p.member(fmt.Sprintf("result-user-%04d", i), fmt.Sprintf("result-user-%04d@example.invalid", i), identity.Member)
		}
		f.code(t, "complete recipient bound", resultAuthorityReadCall(f, f.as(AdminLogin), "context", map[string]string{"collection": "recipients", "capability": "apply"}), 422, string(commands.PrerequisiteFailed))
	})
	t.Run("unknown reads never settle", func(t *testing.T) {
		f := setup(t, SeedSample).ready()
		root, second := f.as(AdminLogin), f.as(Admin2Login)
		proposed := f.run(root, "result_authority.propose_console", "unknown-read", resultPermissionTimes(f))
		f.s.fault = func(at string) bool { return at == "result-authority-after-grant" }
		f.code(t, "unknown application", resultPermissionApprove(f, second, "unknown-read", proposed), 500, string(commands.InternalError))
		f.s.fault = nil
		revision, count := f.s.Revision(), len(f.s.admin.Grants())
		for _, kind := range []string{"history", "context"} {
			fields := map[string]string{}
			if kind == "context" {
				fields = map[string]string{"collection": "ceilings", "capability": "apply"}
			}
			got := resultAuthorityReadCall(f, root, kind, fields)
			if got.status != 200 {
				t.Fatal(got.raw)
			}
		}
		read := resultPermissionRead(f, root, "unknown-read")
		if field(read, "state") != inputauthority.Unknown || len(f.s.resultAuthority.ceilings) != 0 || f.s.Revision() != revision || len(f.s.admin.Grants()) != count {
			t.Fatal("read reconciled unknown effect")
		}
		f.s.domain.Lock()
		f.s.enterHold("synthetic-read-test")
		f.s.domain.Unlock()
		if got := resultAuthorityReadCall(f, root, "history", nil); got.status != 200 {
			t.Fatal(got.raw)
		}
	})
}

func TestResultAuthorityReadFractionalCeilingAndReverseOwnership(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	_ = f.as(MemberLogin)
	f.advance(500 * time.Millisecond)
	root := f.as(AdminLogin)
	approved := provisionResultCeiling(t, f)
	got := resultAuthorityReadCall(f, root, "context", map[string]string{"collection": "ceilings", "capability": "apply"})
	var row map[string]string
	f.must(json.Unmarshal([]byte(field(got, "row_00")), &row))
	if row["not_before"] != f.s.clock().Format(time.RFC3339Nano) {
		t.Fatal("fractional validity lost", row)
	}
	human := f.run(root, "result_authority.propose_human", "fractional-human", map[string]string{"capability": "apply", "recipient": SampleMember, "ceiling": row["ceiling"], "not_before": row["not_before"], "not_after": row["not_after"]})
	f.ok(t, "exact returned boundary accepted", human, inputauthority.Pending, "")
	delete(f.s.resultAuthority.ceilings, field(approved, "grant"))
	f.code(t, "missing applied ceiling reverse association", resultAuthorityReadCall(f, root, "history", nil), 409, string(commands.EvidenceStale))
}
