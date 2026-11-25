package server

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialERPResultPublicCachedSuccessRechecks(t *testing.T) {
	for _, mode := range []string{"rights", "ledger"} {
		t.Run(mode, func(t *testing.T) {
			w, p, digest := publicResult(t)
			f := w.f
			a := w.actors[Admin2Login]
			body := chargeBody(f, "cached-application", "public-result", map[string]string{"digest": digest})
			f.ok(t, "first application", f.call("erp_result.apply", body, a.cookie, a.csrf), "recorded", "")
			receipt := w.a.resultApplications[[2]string{SampleWorkspace, "public-result"}].receipt
			if mode == "rights" {
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ApplyERPResult {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			} else {
				doc, err := w.ledger.DocumentOf(SampleWorkspace, p.Connection, p.External)
				f.must(err)
				doc.Document.Revision++
				doc.Document.Rows[0].Product = "changed-product"
				_, err = w.ledger.Document(SampleWorkspace, doc.Document, erpledger.Actor{ID: SampleAdmin2, Kind: erpledger.KindHuman})
				f.must(err)
			}
			got := f.call("erp_result.apply", body, a.cookie, a.csrf)
			if mode == "rights" {
				f.code(t, "rights before cache", got, 403, string(commands.PermissionDenied))
			} else {
				f.code(t, "ledger before cache", got, 409, string(commands.EvidenceStale))
			}
			if w.a.resultApplications[[2]string{SampleWorkspace, "public-result"}].receipt != receipt {
				t.Fatal("historical receipt erased")
			}
		})
	}
}

func TestFinancialERPResultPublicProjectionBounds(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	f := w.f
	a := w.actors[Admin2Login]
	raw, err := json.Marshal(p)
	f.must(err)
	source := string(raw) + strings.Repeat(" ", 2100-len(raw))
	proposal := f.run(a, "result_evidence.propose", "bounded-result", map[string]string{"packet": source})
	f.ok(t, "admitted source", proposal, "pending", "")
	digest := field(proposal, "digest")
	before := f.s.Revision()
	f.s.cfg.Commands.Value = 2000
	for _, command := range []string{"result_evidence.inspect", "erp_result.inspect"} {
		got := documentRead(f, a, command, "bounded-result", map[string]string{"digest": digest})
		f.code(t, "clean projection refusal", got, 422, string(commands.PrerequisiteFailed))
		if field(got, "source") != "" || f.s.Revision() != before {
			t.Fatal("failed projection disclosed or mutated")
		}
	}
	f.s.cfg.Commands.Value = 1999
	got := documentRead(f, a, "result_evidence.history", p.Preparation, map[string]string{"digest": p.PreparationDigest})
	f.code(t, "unsupported history representation", got, 501, string(commands.UnsupportedCapability))
	if field(got, "row_00") != "" || f.s.Revision() != before {
		t.Fatal("history refusal disclosed or mutated")
	}
}

func publicResult(t *testing.T, beforeFinance ...func(world)) (documentWorld, financialERPResultPacket, string) {
	t.Helper()
	w, p := resultWorld(t, beforeFinance...)
	w.f.s.documentERP = w.a
	raw, err := json.Marshal(p)
	w.f.must(err)
	proposed := w.f.run(w.actors[Admin2Login], "result_evidence.propose", "public-result", map[string]string{"packet": string(raw)})
	w.f.ok(t, "result proposal", proposed, "pending", "")
	digest := field(proposed, "digest")
	confirmed := w.f.run(w.actors[Admin3Login], "result_evidence.confirm", "public-result", map[string]string{"digest": digest})
	w.f.ok(t, "independent result confirmation", confirmed, "confirmed", "")
	return w, p, digest
}

func TestFinancialERPResultDefaultRuntimeAssembly(t *testing.T) {
	w, documentDigest := publicDocument(t)
	f := w.f
	f.ok(t, "confirm runtime document", f.run(w.actors[Admin3Login], "document_evidence.confirm", "public-document", map[string]string{"digest": documentDigest}), "confirmed", "")
	preparation := f.run(w.actors[Admin2Login], "erp_preparation.prepare", "runtime-preparation", map[string]string{"document": "public-document", "digest": documentDigest})
	f.ok(t, "runtime preparation", preparation, "prepared", "")
	for _, entry := range []struct{ capability, recipient string }{{"propose", SampleAdmin2}, {"apply", SampleAdmin2}, {"confirm", SampleAdmin3}} {
		resultGrantTo(t, f, entry.capability, entry.recipient)
	}
	refreshResultActors(w)
	actual, err := w.a.ledger.SubmissionOf(SampleWorkspace, "runtime-preparation")
	f.must(err)
	p := financialERPResultPacket{Mode: "synthetic", Profile: "synthetic_result_one_to_one_v1", Evidence: "runtime-reported-source", Preparation: "runtime-preparation", PreparationDigest: field(preparation, "digest"), Connection: actual.Submission.Connection, External: "runtime-result", Revision: 1, Currency: "USD", Customer: actual.Submission.Customer, Period: "2026-10"}
	for i, line := range actual.Submission.Lines {
		p.Rows = append(p.Rows, financialERPResultRow{ID: []string{"reported-a", "reported-b"}[i], Line: line.ID, Product: "synthetic-product", Unit: "document-unit", Net: []string{"10.25", "4.75"}[i], Tax: []string{"0.25", "0.75"}[i], Quantity: []string{"2", "3"}[i]})
	}
	raw, err := json.Marshal(p)
	f.must(err)
	proposed := f.run(w.actors[Admin2Login], "result_evidence.propose", "runtime-result", map[string]string{"packet": string(raw)})
	f.ok(t, "runtime result proposal", proposed, "pending", "")
	digest := field(proposed, "digest")
	f.ok(t, "runtime result confirmation", f.run(w.actors[Admin3Login], "result_evidence.confirm", "runtime-result", map[string]string{"digest": digest}), "confirmed", "")
	f.ok(t, "runtime result application", f.run(w.actors[Admin2Login], "erp_result.apply", "runtime-result", map[string]string{"digest": digest}), "recorded", "")
	final, err := w.a.ledger.ResultOf(SampleWorkspace, p.Preparation)
	f.must(err)
	if !final.Reconciliation.Matched || len(final.Documents) != 1 || len(final.Documents[0].Links) != 2 {
		t.Fatal("runtime did not reconcile actual ledger")
	}
}

func TestFinancialERPResultPublicJourney(t *testing.T) {
	w, p, digest := publicResult(t)
	f := w.f
	a := w.actors[Admin2Login]
	read := documentRead(f, a, "result_evidence.inspect", "public-result", map[string]string{"digest": digest})
	if read.status != 200 || field(read, "current_eligibility") != "unchecked" || field(read, "source") == "" || field(read, "application_state") != "not_started" {
		t.Fatal(read.raw)
	}
	applied := f.run(a, "erp_result.apply", "public-result", map[string]string{"digest": digest})
	f.ok(t, "result applied", applied, "recorded", "")
	if !validPlanHash(field(applied, "application_receipt")) || field(applied, "source") != "" {
		t.Fatal("invalid mutation projection")
	}
	actual, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
	f.must(err)
	if !actual.Reconciliation.Matched || len(actual.Documents) != 1 || len(actual.Documents[0].Links) != 2 || actual.Submission.State != erpledger.StatePrepared || len(actual.Submission.Commands) != 0 {
		t.Fatal(actual)
	}
	history := documentRead(f, a, "result_evidence.history", p.Preparation, map[string]string{"digest": p.PreparationDigest})
	if history.status != 200 || field(history, "row_00") == "" {
		t.Fatal(history.raw)
	}
	denied := documentRead(f, w.actors[AdminLogin], "result_evidence.inspect", "public-result", map[string]string{"digest": digest})
	f.code(t, "dedicated Cost permission", denied, 403, string(commands.PermissionDenied))
}

func TestFinancialERPResultPublicInterruptedRecovery(t *testing.T) {
	for _, stage := range []string{"document", "link:row-0", "link:row-1"} {
		t.Run(stage, func(t *testing.T) {
			w, p, digest := publicResult(t)
			f := w.f
			a := w.actors[Admin2Login]
			w.a.afterResultWrite = func(at string) error {
				if at == stage {
					return errors.New("lost response")
				}
				return nil
			}
			body := chargeBody(f, "interrupted-result", "public-result", map[string]string{"digest": digest})
			first := f.call("erp_result.apply", body, a.cookie, a.csrf)
			if first.status != 500 || first.body["operation_id"] == nil || first.body["next_action"] != string(commands.NextOperation) {
				t.Fatal(first.raw)
			}
			w.a.afterResultWrite = nil
			replay := f.call("erp_result.apply", body, a.cookie, a.csrf)
			if replay.status != 500 || replay.body["operation_id"] != first.body["operation_id"] {
				t.Fatal(replay.raw)
			}
			var changed map[string]any
			f.must(json.Unmarshal([]byte(body), &changed))
			changed["request_id"] = "new-request-same-key"
			newRequest, err := json.Marshal(changed)
			f.must(err)
			sameKey := f.call("erp_result.apply", string(newRequest), a.cookie, a.csrf)
			if sameKey.status != 500 || sameKey.body["operation_id"] != first.body["operation_id"] {
				t.Fatal(sameKey.raw)
			}
			observed := documentRead(f, a, "erp_result.inspect", "public-result", map[string]string{"digest": digest})
			if observed.status != 200 || field(observed, "application_state") != "unknown" || field(observed, "application_receipt") != "" {
				t.Fatal(observed.raw)
			}
			recovered := f.run(a, "erp_result.apply", "public-result", map[string]string{"digest": digest})
			f.ok(t, "owned recovery", recovered, "recorded", "")
			actual, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
			f.must(err)
			if len(actual.Documents) != 1 || len(actual.Documents[0].Links) != 2 {
				t.Fatal("duplicate recovery effects")
			}
		})
	}
}
