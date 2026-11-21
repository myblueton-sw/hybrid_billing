package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func (f world) approvalRead(a actor, q string) reply {
	f.t.Helper()
	return f.do("GET", "/api/approval-review?"+q, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
	})
}

func TestApprovalReadModuleAndPages(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	first := f.approvalRead(root, "view=pending")
	if first.status != 200 {
		t.Fatalf("first: %v", first)
	}
	rows := first.body["records"].([]any)
	if len(rows) != 25 || fmt.Sprint(first.body["total"]) != "29" || first.body["commands"] != "financial_charge_v1" {
		t.Fatalf("page: %v", first.body)
	}
	row := rows[0].(map[string]any)
	amount := row["amounts"].([]any)[0].(map[string]any)
	if amount["candidate"] != "1234.56789012" || amount["base"] != nil || row["status"] != "blocked" || row["deadline"] != nil {
		t.Fatalf("projection: %v", row)
	}
	q := url.Values{"view": {"pending"}, "cursor": {first.body["next_cursor"].(string)}, "snapshot": {first.body["snapshot"].(string)}, "permission_revision": {first.body["permission_revision"].(string)}}
	second := f.approvalRead(root, q.Encode())
	if second.status != 200 || len(second.body["records"].([]any)) != 4 || second.body["next_cursor"] != nil {
		t.Fatalf("second: %v", second)
	}
	q.Set("q", "other")
	if got := f.approvalRead(root, q.Encode()); got.status != 409 {
		t.Fatalf("query reuse: %v", got)
	}
	other := f.as(Admin2Login)
	q.Del("q")
	if got := f.approvalRead(other, q.Encode()); got.status != 409 {
		t.Fatalf("session reuse: %v", got)
	}
	decisions := f.approvalRead(other, "view=decisions")
	if decisions.status != 200 || fmt.Sprint(decisions.body["total"]) != "1" {
		t.Fatalf("decisions: %v", decisions)
	}
	if got := f.approvalRead(root, "view=decisions"); got.status != 200 || fmt.Sprint(got.body["total"]) != "0" {
		t.Fatalf("own decisions: %v", got)
	}
}

func TestApprovalReadBoundary(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	if got := f.approvalRead(actor{}, "view=pending"); got.status != 401 {
		t.Fatalf("anonymous: %v", got)
	}
	if got := f.approvalRead(f.as(MemberLogin), "view=pending"); got.status != 403 {
		t.Fatalf("member: %v", got)
	}
	for _, q := range []string{"view=pending&view=history", "view=pending&workspace=other", "view=pending&q=%00", "view=pending&q=%FF", "view=pending&cursor=x", "view=pending&snapshot=" + strings.Repeat("A", 64), "view=pending&q=%zz", "view=recognition"} {
		if got := f.approvalRead(root, q); got.status != 422 {
			t.Errorf("invalid %s: %v", q, got)
		}
	}
	for _, q := range []string{"view=recognition&kind=monthly_recognition", "view=delegation", "view=pending&kind=adjustment", "view=pending&kind=policy"} {
		if got := f.approvalRead(root, q); got.status != 501 {
			t.Errorf("unconnected %s: %v", q, got)
		}
	}
	if got := f.approvalRead(root, "view=history&q=absent"); got.status != 200 || fmt.Sprint(got.body["total"]) != "0" {
		t.Fatalf("empty: %v", got)
	}
	f.s.fault = func(step string) bool { return step == "approvals" }
	if got := f.approvalRead(root, "view=pending"); got.status != 503 {
		t.Fatalf("fault: %v", got)
	}
}

func TestApprovalReadUnsupportedStateFailsBeforeFiltering(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	_, err := f.s.approvals.Withdraw(SampleWorkspace, "sample-approval-029", approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.approvalRead(root, "view=decisions&q=absent"); got.status != 503 {
		t.Fatalf("hidden unsupported state: %v", got)
	}
}
