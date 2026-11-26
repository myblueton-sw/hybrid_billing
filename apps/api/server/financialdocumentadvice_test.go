package server

import (
	"encoding/json"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func documentAdviceRead(w documentWorld, a actor, digest string) reply {
	return documentRead(w.f, a, "document_evidence.advice", "public-document", map[string]string{"digest": digest})
}
func documentAdviceValues(t *testing.T, r reply, kind string) map[string]documentAdviceState {
	t.Helper()
	if r.status != 200 {
		t.Fatal(r.raw)
	}
	var v map[string]documentAdviceState
	if err := json.Unmarshal([]byte(field(r, kind)), &v); err != nil {
		t.Fatal(err, r.raw)
	}
	return v
}
func TestDocumentAdvicePermissionReadinessLifecycle(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	proposer, decider := w.actors[Admin2Login], w.actors[Admin3Login]
	before := f.s.Revision()
	r := documentAdviceRead(w, proposer, digest)
	permissions := documentAdviceValues(t, r, "permissions")
	ready := documentAdviceValues(t, r, "readiness")
	if permissions["prepare"].State != "allowed" || permissions["confirm"].State != "denied" || ready["prepare"].State != "inapplicable" || ready["confirm"].State != "unchecked" || len(ready) != 3 || len(permissions) != 4 {
		t.Fatal(permissions, ready)
	}
	if field(r, "current_eligibility") != "advisory" || field(r, "preparation_scope") != "packet_only_key_capacity_and_source_coverage_unchecked" {
		t.Fatal(r.raw)
	}
	r = documentAdviceRead(w, decider, digest)
	if documentAdviceValues(t, r, "readiness")["confirm"].State != "ready" {
		t.Fatal(r.raw)
	}
	if f.s.Revision() != before || len(w.a.owners) != 0 {
		t.Fatal("advice changed state")
	}
	f.s.enterHold("synthetic advice test")
	r = documentAdviceRead(w, decider, digest)
	ready = documentAdviceValues(t, r, "readiness")
	if ready["confirm"].State != "blocked" || ready["revoke"].State != "ready" || documentAdviceValues(t, r, "permissions")["confirm"].State != "allowed" {
		t.Fatal(r.raw)
	}
	f.s.hold = nil
	f.ok(t, "confirm", f.run(decider, "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), "confirmed", "")
	r = documentAdviceRead(w, proposer, digest)
	if documentAdviceValues(t, r, "readiness")["prepare"].State != "ready" {
		t.Fatal(r.raw)
	}
	if len(w.a.owners) != 0 {
		t.Fatal("advice reserved preparation")
	}
	f.ok(t, "revoke", f.run(decider, "document_evidence.revoke", "public-document", map[string]string{"digest": digest}), "revoked", "")
	r = documentAdviceRead(w, decider, digest)
	if documentAdviceValues(t, r, "readiness")["revoke"].State != "inapplicable" {
		t.Fatal(r.raw)
	}
	r = documentAdviceRead(w, proposer, digest)
	if documentAdviceValues(t, r, "readiness")["prepare"].State != "inapplicable" {
		t.Fatal(r.raw)
	}
}
func TestDocumentAdviceProposerAndCurrentEvidence(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	documentGrantTo(t, f, "confirm", SampleAdmin2)
	proposer := f.as(Admin2Login)
	decider := w.actors[Admin3Login]
	r := documentAdviceRead(w, proposer, digest)
	self := documentAdviceValues(t, r, "readiness")["confirm"]
	if self.State != "inapplicable" || self.Code != commands.SelfApprovalDenied {
		t.Fatal(r.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ProposeDocumentEvidence {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	r = documentAdviceRead(w, decider, digest)
	state := documentAdviceValues(t, r, "readiness")["confirm"]
	if state.State != "blocked" || state.Code != commands.PermissionDenied || documentAdviceValues(t, r, "permissions")["confirm"].State != "allowed" {
		t.Fatal(r.raw)
	}
	f.code(t, "wrong digest", documentAdviceRead(w, decider, "wrong"), 409, string(commands.EvidenceStale))
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin3 && g.Terms.Action == gate.ViewCost && g.Terms.Kind == "synthetic_document_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "Cost before lookup", documentAdviceRead(w, decider, "wrong"), 403, string(commands.PermissionDenied))
}

func TestDocumentAdviceClockAndClosedCodes(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	w.a.documentTime = f.s.clock().AddDate(0, 0, 1)
	r := documentAdviceRead(w, w.actors[Admin3Login], digest)
	ready := documentAdviceValues(t, r, "readiness")
	for _, action := range []string{"confirm", "revoke"} {
		if ready[action].State != "unavailable" || ready[action].Code != commands.DependencyUnavailable {
			t.Fatal(r.raw)
		}
	}
	if documentAdviceValues(t, r, "permissions")["confirm"].State != "allowed" {
		t.Fatal(r.raw)
	}
	for _, code := range []commands.Code{commands.AuthorityUnavailable, commands.DependencyUnavailable} {
		if v := documentPermissionAdvice(commands.Fail(code)); v.State != "unavailable" || v.Code != code {
			t.Fatal(v)
		}
	}
	if v := documentPermissionAdvice(commands.Fail(commands.PermissionDenied)); v.State != "denied" {
		t.Fatal(v)
	}
	if v := documentReadinessAdvice(commands.Fail(commands.EvidenceStale)); v.State != "blocked" {
		t.Fatal(v)
	}
	if v := documentAdviceCode(commands.Fail(commands.Code("internal-unreviewed-detail"))); v != commands.DependencyUnavailable {
		t.Fatal("unknown code escaped", v)
	}
}
