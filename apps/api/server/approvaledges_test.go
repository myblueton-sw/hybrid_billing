package server

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func approvalHuman(id string) approvals.Actor {
	return approvals.Actor{ID: id, Kind: approvals.KindHuman}
}
func approvalAdminister(t *testing.T, s *Server, p approvals.Proposal) {
	t.Helper()
	if _, err := s.approvals.Propose(SampleWorkspace, p, approvalHuman(SampleAdmin)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.approvals.Confirm(SampleWorkspace, p.ID, approvalHuman(SampleAdmin2)); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalReadHistoricalAndUnsupportedProjection(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	approvalAdminister(t, f.s, approvals.Proposal{ID: "sample-grant-3", Kind: approvals.ProposeGrant, Authority: approvals.Authority{ID: "grant-3", Approver: SampleAdmin3, Stage: approvals.StageCharge, Currency: "USD", Limit: "5000", From: f.s.clock().Format(time.DateOnly)}})
	if _, err := f.s.approvals.Decide(SampleWorkspace, "sample-approval-001", approvals.Decision{Verdict: approvals.VerdictApprove}, approvalHuman(SampleAdmin3)); err != nil {
		t.Fatal(err)
	}
	got := f.approvalRead(root, "view=history&q=sample-approval-001")
	if got.status != 200 {
		t.Fatalf("history: %v", got)
	}
	row := got.body["records"].([]any)[0].(map[string]any)
	if row["status"] != "approved" || row["approver"].(map[string]any)["id"] != SampleAdmin3 || row["authority"].(map[string]any)["limit"] != "unknown" || row["deadline"] != nil {
		t.Fatalf("historical row: %v", row)
	}
	f.advance(25 * time.Hour)
	rows, err := f.s.approvalRows(SampleWorkspace)
	if err != nil || rows[0].Status != "approved" || rows[0].Deadline != nil {
		t.Fatalf("historical expiry: %v %v", rows, err)
	}
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "sample-approval-003")
	if err != nil {
		t.Fatal(err)
	}
	v.Subject.ID = "sample-replacement"
	if _, err = f.s.approvals.Request(SampleWorkspace, v.Subject, approvalHuman(SampleAdmin)); err != nil {
		t.Fatal(err)
	}
	rows, err = f.s.approvalRows(SampleWorkspace)
	if err != nil || rows[2].Status != "superseded" {
		t.Fatalf("superseded: %v %v", rows, err)
	}
	f.s.approvalIndex[0].Target = "wrong"
	if _, err = f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("metadata disagreement accepted")
	}
	f.s.approvalIndex[0].Target = "sample-result-001"
	f.s.approvalIndex = append(f.s.approvalIndex, f.s.approvalIndex[0])
	if _, err = f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("duplicate index accepted")
	}
}

func TestApprovalReadPartialAndPolicyMismatch(t *testing.T) {
	f := setup(t, SeedSample).ready()
	approvalAdminister(t, f.s, approvals.Proposal{ID: "two-approvals", Kind: approvals.ProposePolicy, Policy: approvals.Policy{Stage: approvals.StageCharge, Revision: 2, Approvals: 2, Required: []string{"input", "result"}, Validity: time.Hour}})
	if _, err := f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("policy mismatch accepted")
	}
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "sample-approval-001")
	if err != nil {
		t.Fatal(err)
	}
	v.Subject.ID = "two-approver-subject"
	if _, err = f.s.approvals.Request(SampleWorkspace, v.Subject, approvalHuman(SampleAdmin)); err != nil {
		t.Fatal(err)
	}
	f.s.approvalIndex = []approvalIndex{{Workspace: SampleWorkspace, ID: v.Subject.ID, Target: v.Subject.Target, Basis: "gross"}}
	if _, err = f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("multi-approval policy accepted before a decision")
	}
	approvalAdminister(t, f.s, approvals.Proposal{ID: "grant-proposal", Kind: approvals.ProposeGrant, Authority: approvals.Authority{ID: "grant", Approver: SampleAdmin3, Stage: approvals.StageCharge, Currency: "USD", Limit: "5000", From: f.s.clock().Format(time.DateOnly)}})
	if _, err = f.s.approvals.Decide(SampleWorkspace, v.Subject.ID, approvals.Decision{Verdict: approvals.VerdictApprove}, approvalHuman(SampleAdmin3)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("partial decision accepted")
	}
}

func TestApprovalReadCursorAndRevocation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.approvalRead(root, "view=pending")
	q := url.Values{"view": {"pending"}, "cursor": {got.body["next_cursor"].(string)}, "snapshot": {got.body["snapshot"].(string)}, "permission_revision": {got.body["permission_revision"].(string)}}
	q.Set("cursor", "25_"+strings.Repeat("0", 64))
	if x := f.approvalRead(root, q.Encode()); x.status != 422 {
		t.Fatalf("tamper: %v", x)
	}
	q.Set("cursor", got.body["next_cursor"].(string))
	f.s.counter++
	if x := f.approvalRead(root, q.Encode()); x.status != 409 {
		t.Fatalf("revision: %v", x)
	}
	if err := f.s.admin.RevokeGrant("approval-"+SampleAdmin, 1); err != nil {
		t.Fatal(err)
	}
	if x := f.approvalRead(root, "view=pending"); x.status != 403 {
		t.Fatalf("revocation: %v", x)
	}
	terms := gate.Terms{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.One("customer-one"), LegalEntity: gate.All(), Contract: gate.All()}}
	if err := f.s.admin.AddGrant(gate.Grant{ID: "narrow", SubjectID: SampleAdmin, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	if x := f.approvalRead(root, "view=pending"); x.status != 403 {
		t.Fatalf("narrow grant broadened: %v", x)
	}
}

func TestApprovalReadUTF16AndEmptyIndex(t *testing.T) {
	if approvalQuery(url.Values{"view": {"pending"}, "q": {strings.Repeat("😀", 65)}}) {
		t.Fatal("UTF16 limit exceeded")
	}
	if !approvalQuery(url.Values{"view": {"pending"}, "q": {strings.Repeat("😀", 64)}}) {
		t.Fatal("valid UTF16 query refused")
	}
	f := setup(t, SeedNone).ready()
	rows, err := f.s.approvalRows(EmptyWorkspace)
	if err != nil || len(rows) != 0 {
		t.Fatalf("empty seed: %v %v", rows, err)
	}
	f.s.approvalIndex = []approvalIndex{{Workspace: "foreign", ID: "missing", Target: "missing", Basis: "gross"}}
	rows, err = f.s.approvalRows(EmptyWorkspace)
	if err != nil || len(rows) != 0 {
		t.Fatalf("foreign index disclosed: %v %v", rows, err)
	}
	f.s.approvalIndex[0].Workspace = EmptyWorkspace
	if _, err = f.s.approvalRows(EmptyWorkspace); err == nil {
		t.Fatal("index/module disagreement accepted")
	}
}

func TestApprovalReadEmptyInstallationHTTP(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatalf("bootstrap: %v", got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	root := actor{cookie, csrf}
	terms := gate.Terms{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: EmptyWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	valid := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	for _, grant := range []gate.Grant{{ID: "empty-client", ClientID: ConsoleClient, Terms: terms, Validity: valid}, {ID: "empty-user", SubjectID: EmptyAdmin, Terms: terms, Validity: valid}} {
		if err = f.s.admin.AddGrant(grant); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.approvalRead(root, "view=pending"); got.status != 200 || len(got.body["records"].([]any)) != 0 {
		t.Fatalf("empty HTTP: %v", got)
	}
	f.s.approvalIndex = []approvalIndex{{Workspace: SampleWorkspace, ID: "foreign-missing", Target: "foreign", Basis: "gross"}}
	if got := f.approvalRead(root, "view=history"); got.status != 200 || len(got.body["records"].([]any)) != 0 {
		t.Fatalf("foreign HTTP: %v", got)
	}
	if got := f.approvalRead(root, "view=recognition&kind=monthly_recognition"); got.status != 501 {
		t.Fatalf("recognition not empty: %v", got)
	}
}

func TestApprovalReadSameCountPolicyRevisionMismatch(t *testing.T) {
	f := setup(t, SeedSample).ready()
	approvalAdminister(t, f.s, approvals.Proposal{ID: "new-single-policy", Kind: approvals.ProposePolicy, Policy: approvals.Policy{Stage: approvals.StageCharge, Revision: 2, Approvals: 1, Required: []string{"input", "result"}, Validity: time.Hour}})
	if _, err := f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("policy revision mismatch accepted with unchanged approval count")
	}
}

func TestApprovalReadKindsAndEndedSession(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, kind := range []string{"billing_result", "final_erp_amount"} {
		got := f.approvalRead(root, "view=history&kind="+kind)
		if got.status != 200 || len(got.body["records"].([]any)) != 15 {
			t.Fatalf("kind filter %s: status %d", kind, got.status)
		}
		for _, r := range got.body["records"].([]any) {
			if r.(map[string]any)["kind"] != kind {
				t.Fatal("foreign kind returned")
			}
		}
	}
	for _, q := range []string{"view=bad", "view=history&kind=bad"} {
		if got := f.approvalRead(root, q); got.status != 422 {
			t.Fatalf("invalid enum accepted: %d", got.status)
		}
	}
	for _, v := range f.s.sessions.byLogin(AdminLogin) {
		f.s.sessions.deny(v, AdminLogin, f.s.clock())
	}
	if got := f.approvalRead(root, "view=pending"); got.status != 401 {
		t.Fatalf("ended session read: %d", got.status)
	}
}

func TestApprovalReadUnsupportedStage(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	approvalAdminister(t, f.s, approvals.Proposal{ID: "issue-policy", Kind: approvals.ProposePolicy, Policy: approvals.Policy{Stage: approvals.StageIssue, Revision: 1, Approvals: 1, Required: []string{"input", "result"}, Validity: time.Hour}})
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "sample-approval-001")
	if err != nil {
		t.Fatal(err)
	}
	v.Subject.Stage = approvals.StageIssue
	v.Subject.ID = "sample-issue"
	if _, err = f.s.approvals.Request(SampleWorkspace, v.Subject, approvalHuman(SampleAdmin)); err != nil {
		t.Fatal(err)
	}
	f.s.approvalIndex = []approvalIndex{{Workspace: SampleWorkspace, ID: v.Subject.ID, Target: v.Subject.Target, Basis: "gross"}}
	if _, err = f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("unsupported stage projected as billing approval")
	}
	if got := f.approvalRead(root, "view=history&q=absent"); got.status != 503 {
		t.Fatalf("unsupported stage hidden by filter: %d", got.status)
	}
}

func TestApprovalReadSupersededDecisionAndSeedReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	approvalAdminister(t, f.s, approvals.Proposal{ID: "grant-superseded", Kind: approvals.ProposeGrant, Authority: approvals.Authority{ID: "grant-superseded", Approver: SampleAdmin3, Stage: approvals.StageCharge, Currency: "USD", Limit: "5000", From: f.s.clock().Format(time.DateOnly)}})
	if _, err := f.s.approvals.Decide(SampleWorkspace, "sample-approval-001", approvals.Decision{Verdict: approvals.VerdictApprove}, approvalHuman(SampleAdmin3)); err != nil {
		t.Fatal(err)
	}
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "sample-approval-001")
	if err != nil {
		t.Fatal(err)
	}
	v.Subject.ID = "sample-new-result"
	if _, err = f.s.approvals.Request(SampleWorkspace, v.Subject, approvalHuman(SampleAdmin)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.approvalRows(SampleWorkspace); err == nil {
		t.Fatal("superseded historical decision silently flattened")
	}
	before := len(f.s.approvalIndex)
	if err = f.s.seedApprovals(f.s.clock()); err == nil {
		t.Fatal("duplicate seed accepted")
	}
	if len(f.s.approvalIndex) != before {
		t.Fatal("failed seed appended rows")
	}
}

func TestApprovalReadItemizedSubjectUnavailable(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "sample-approval-001")
	if err != nil {
		t.Fatal(err)
	}
	v.Subject.ID = "sample-itemized"
	v.Subject.Entries = []approvals.Entry{{Item: "sample-line", Scope: "sample-scope", Amount: v.Subject.Amount, Digest: approvalHash("sample-line")}}
	if _, err = f.s.approvals.Request(SampleWorkspace, v.Subject, approvalHuman(SampleAdmin)); err != nil {
		t.Fatal(err)
	}
	f.s.approvalIndex = []approvalIndex{{Workspace: SampleWorkspace, ID: v.Subject.ID, Target: v.Subject.Target, Basis: "gross"}}
	if got := f.approvalRead(root, "view=history&q=absent"); got.status != 503 {
		t.Fatalf("itemized evidence hidden by filter: %d", got.status)
	}
}
