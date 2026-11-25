package server

import (
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialFinalSubjectFreshnessAndNoBusinessMutation(t *testing.T) {
	for _, mode := range []string{"current", "unknown", "ledger-drift", "result-proposer", "result-confirmer", "charge-approver", "ended-session"} {
		t.Run(mode, func(t *testing.T) {
			w, _, digest := publicResult(t)
			f := w.f
			proposal := f.run(w.actors[AdminLogin], "erp_final_policy.propose", "boundary-final-policy", financialFields("2"))
			f.ok(t, "propose final policy", proposal, "pending", "")
			f.ok(t, "confirm final policy", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", "boundary-final-policy", map[string]string{"digest": field(proposal, "digest")}), "confirmed", "")
			if mode == "unknown" {
				w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
			}
			applied := f.run(w.actors[Admin2Login], "erp_result.apply", "public-result", map[string]string{"digest": digest})
			if mode == "unknown" {
				f.code(t, "interrupted application", applied, 500, "internal_error")
			} else {
				f.ok(t, "apply result", applied, "recorded", "")
			}
			w.a.afterResultWrite = nil
			switch mode {
			case "ledger-drift":
				app := w.a.resultApplications[[2]string{SampleWorkspace, "public-result"}]
				doc := app.document
				doc.Revision++
				_, err := w.ledger.Document(SampleWorkspace, doc, erpledger.Actor{ID: SampleAdmin2, Kind: erpledger.KindHuman})
				f.must(err)
			case "result-proposer", "result-confirmer", "charge-approver":
				subject, action := SampleAdmin2, gate.ProposeERPResult
				if mode == "result-confirmer" {
					subject, action = SampleAdmin3, gate.ConfirmERPResult
				}
				if mode == "charge-approver" {
					charge, err := f.s.approvals.RequestOf(SampleWorkspace, "document-charge")
					f.must(err)
					subject, action = charge.Decisions[0].Actor, gate.Approve
				}
				count := 0
				for _, grant := range f.s.admin.Grants() {
					if grant.SubjectID == subject && grant.Terms.Action == action {
						f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
						count++
					}
				}
				if count == 0 {
					t.Fatal("fixture revoked no grants")
				}
			}
			before, err := w.ledger.ResultOf(SampleWorkspace, "result-preparation")
			f.must(err)
			chargeBefore, err := f.s.approvals.RequestOf(SampleWorkspace, "document-charge")
			f.must(err)
			revision := f.s.Revision()
			grants := f.s.admin.Grants()
			applications := len(w.a.resultApplications)
			app := w.a.resultApplications[[2]string{SampleWorkspace, "public-result"}]
			err = inputAuthorityAs(f, w.actors[Admin3Login], func() error {
				if mode == "ended-session" {
					f.s.endProbe(f.s.caller)
				}
				_, err := f.s.financialFinalSubjectLocked(SampleWorkspace, "boundary-request", "public-result", digest, SampleAdmin3)
				return err
			})
			if mode == "current" {
				f.must(err)
			} else {
				want := commands.PermissionDenied
				switch mode {
				case "unknown":
					want = commands.PrerequisiteFailed
				case "ledger-drift":
					want = commands.EvidenceStale
				case "ended-session":
					want = commands.Unauthenticated
				}
				documentFailure(t, err, want)
			}
			after, readErr := w.ledger.ResultOf(SampleWorkspace, "result-preparation")
			f.must(readErr)
			chargeAfter, readErr := f.s.approvals.RequestOf(SampleWorkspace, "document-charge")
			f.must(readErr)
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(chargeBefore, chargeAfter) || !reflect.DeepEqual(grants, f.s.admin.Grants()) || revision != f.s.Revision() || applications != len(w.a.resultApplications) || !reflect.DeepEqual(app, w.a.resultApplications[[2]string{SampleWorkspace, "public-result"}]) {
				t.Fatal("subject construction changed business state")
			}
			_, readErr = f.s.approvals.RequestOf(SampleWorkspace, "boundary-request")
			if !errors.Is(readErr, approvals.ErrNotFound) {
				t.Fatalf("builder created a request: %v", readErr)
			}
		})
	}
}

func TestFinancialFinalSubjectMissingUpstreamOptionalEvidence(t *testing.T) {
	for _, required := range []string{"input,result", "result,rule"} {
		t.Run(required, func(t *testing.T) {
			w, packet := resultWorldFromDocument(t, explicitDocumentWorldWithRequired(t, "result"))
			f := w.f
			f.s.documentERP = w.a
			result := confirmResult(t, w, proposeResult(t, w, packet, "optional-result"))
			_, err := applyResult(w, result)
			f.must(err)
			upstream, err := f.s.approvals.RequestOf(SampleWorkspace, "document-charge")
			f.must(err)
			if upstream.State != approvals.StateApproved || len(upstream.Subject.Digests) != 1 || upstream.Subject.Digests["result"] == "" {
				t.Fatal("fixture did not establish a valid result-only charge approval")
			}
			fields := financialFields("2")
			fields["required"] = required
			p := f.run(w.actors[AdminLogin], "erp_final_policy.propose", "optional-final-policy", fields)
			f.ok(t, "propose final policy", p, "pending", "")
			f.ok(t, "confirm final policy", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", "optional-final-policy", map[string]string{"digest": field(p, "digest")}), "confirmed", "")
			err = inputAuthorityAs(f, w.actors[Admin3Login], func() error {
				_, err := f.s.currentFinancialChargeLocked(SampleWorkspace, w.basis.Request, w.basis.RequestDigest, SampleAdmin3)
				if err != nil {
					t.Fatalf("upstream became stale: %v", err)
				}
				_, err = f.s.financialFinalSubjectLocked(SampleWorkspace, "missing-optional-request", result.Basis.ID, result.Review.Digest, SampleAdmin3)
				return err
			})
			documentFailure(t, err, commands.PrerequisiteFailed)
		})
	}
}
