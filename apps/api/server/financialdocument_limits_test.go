package server

import (
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialDocumentPublicOutputAdmissionAndMaximumReasons(t *testing.T) {
	w := explicitDocumentWorld(t)
	f := w.f
	a := w.actors[Admin2Login]
	decider := w.actors[Admin3Login]
	packet := packetJSON(t, packetFromBasis(w.basis))
	limits := f.s.cfg.Commands
	for _, mode := range []string{"fields", "value", "expanded"} {
		f.s.cfg.Commands = limits
		want, status := commands.UnsupportedCapability, 501
		switch mode {
		case "fields":
			f.s.cfg.Commands.Fields = 16
		case "value":
			f.s.cfg.Commands.Value = 1999
		case "expanded":
			f.s.cfg.Commands.Value = 2000
			want, status = commands.PrerequisiteFailed, 422
		}
		f.code(t, mode, f.run(a, "document_evidence.propose", mode, map[string]string{"packet": packet}), status, string(want))
		if len(f.s.documentERP.documents) != 0 {
			t.Fatal("failed admission retained packet")
		}
	}
	f.s.cfg.Commands = limits
	reason := strings.Repeat("\"", 2000)
	p := f.admin(a, "document_evidence.propose", "long-reasons", reason, map[string]string{"packet": packet})
	f.ok(t, "max proposal reason", p, "pending", "")
	digest := field(p, "digest")
	f.ok(t, "max confirmation reason", f.admin(decider, "document_evidence.confirm", "long-reasons", reason, map[string]string{"digest": digest}), "confirmed", "")
	f.ok(t, "max revocation reason", f.admin(decider, "document_evidence.revoke", "long-reasons", reason, map[string]string{"digest": digest}), "revoked", "")
	r := documentRead(f, a, "document_evidence.inspect", "long-reasons", map[string]string{"digest": digest})
	if r.status != 200 || field(r, "proposal_reason") != reason || field(r, "confirmation_reason") != reason || field(r, "revocation_reason") != reason || len(r.body["fields"].(map[string]any)) != 17 {
		t.Fatal(r.raw)
	}
	f.s.cfg.Commands.Value = 2000
	f.code(t, "read refuses unavailable representation", documentRead(f, a, "document_evidence.inspect", "long-reasons", map[string]string{"digest": digest}), 422, string(commands.PrerequisiteFailed))
}

func TestFinancialDocumentPublicReplayAndSelfConfirmation(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	a := w.actors[Admin2Login]
	documentGrantTo(t, f, "confirm", SampleAdmin2)
	f.code(t, "actual self confirmation", f.run(a, "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), 403, string(commands.SelfApprovalDenied))
	body := chargeBody(f, "proposal-replay", "replay-document", map[string]string{"packet": packetJSON(t, packetFromBasis(w.basis))})
	f.ok(t, "proposal", f.call("document_evidence.propose", body, a.cookie, a.csrf), "pending", "")
	if r := f.call("document_evidence.propose", body, a.cookie, a.csrf); r.status != 200 || r.body["replay"] != true {
		t.Fatal(r.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.ClientID == ConsoleClient && g.Terms.Action == gate.ProposeDocumentEvidence {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked console before replay", f.call("document_evidence.propose", body, a.cookie, a.csrf), 403, string(commands.PermissionDenied))
	f.s.endProbe(a.verifier(f))
	f.code(t, "ended session before replay", f.call("document_evidence.propose", body, a.cookie, a.csrf), 401, string(commands.Unauthenticated))
}

func TestFinancialDocumentRuntimeSeedsNoAuthority(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if f.s.documentERP == nil || len(f.s.documentERP.documents) != 0 || len(f.s.documentERP.owners) != 0 {
		t.Fatal("incorrect synthetic seed")
	}
	for _, grant := range f.s.admin.Grants() {
		if grant.Terms.Kind == "synthetic_document_evidence" {
			t.Fatal("default document authority")
		}
	}
}
