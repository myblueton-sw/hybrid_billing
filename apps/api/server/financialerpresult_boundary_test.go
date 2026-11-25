package server

import (
	"errors"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialERPResultCurrentAuthorityAndEvidence(t *testing.T) {
	for _, mode := range []string{"proposer", "confirmer", "consumer", "console", "document", "mapping", "hold", "wrong-actor", "wrong-digest", "foreign-workspace"} {
		t.Run(mode, func(t *testing.T) {
			w, p := resultWorld(t)
			v := confirmResult(t, w, proposeResult(t, w, p, "current"))
			w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
			_, err := applyResult(w, v)
			if !errors.Is(err, errFinancialERPResultUncertain) {
				t.Fatal(err)
			}
			w.a.afterResultWrite = nil
			switch mode {
			case "proposer", "confirmer", "consumer", "console":
				subject, action := SampleAdmin2, gate.ProposeERPResult
				if mode == "confirmer" {
					subject, action = SampleAdmin3, gate.ConfirmERPResult
				}
				if mode == "consumer" || mode == "console" {
					action = gate.ApplyERPResult
				}
				for _, g := range w.f.s.admin.Grants() {
					if g.Terms.Action == action && (mode == "console" && g.ClientID == ConsoleClient || mode != "console" && g.SubjectID == subject) {
						w.f.must(w.f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			case "document":
				_, err := decideDocument(w, Admin3Login, financialDocumentView{Basis: w.basis, Review: financialDocumentReview{Digest: w.digest}}, true, "Withdraw document")
				w.f.must(err)
			case "mapping":
				m := w.basis.Mappings[0].Mapping
				m.External = "replacement"
				_, err := w.ledger.Map(SampleWorkspace, m, erpledger.Actor{ID: "mapper", Kind: erpledger.KindHuman})
				w.f.must(err)
			case "hold":
				w.f.s.enterHold("result-test")
			}
			if mode == "wrong-actor" {
				resultGrantTo(t, w.f, "apply", SampleAdmin)
				refreshResultActors(w)
				_, err = resultAs(w, AdminLogin, func(actor string) (financialERPResultView, error) {
					return w.a.applyResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor)
				})
			} else if mode == "wrong-digest" {
				v.Review.Digest = p.PreparationDigest
				_, err = applyResult(w, v)
			} else if mode == "foreign-workspace" {
				_, err = resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
					return w.a.applyResultLocked("foreign", v.Basis.ID, v.Review.Digest, actor)
				})
			} else {
				_, err = applyResult(w, v)
			}
			if err == nil {
				t.Fatal("changed facts accepted")
			}
			if mode == "console" {
				documentFailure(t, err, commands.PermissionDenied)
			}
			if mode == "wrong-actor" {
				documentFailure(t, err, commands.EvidenceStale)
			}
			app := w.a.resultApplications[[2]string{SampleWorkspace, "current"}]
			if app.receipt != "" || app.actor != SampleAdmin2 || len(w.a.resultApplications) != 1 {
				t.Fatal("unresolved ownership lost")
			}
			actual, err := w.ledger.DocumentOf(SampleWorkspace, p.Connection, p.External)
			w.f.must(err)
			if len(actual.Links) != 0 {
				t.Fatal("recovery wrote after denial")
			}
		})
	}
}

func TestFinancialERPResultHistoricalProposalAndHeldRevocation(t *testing.T) {
	w, p := resultWorld(t)
	_, err := decideDocument(w, Admin3Login, financialDocumentView{Basis: w.basis, Review: financialDocumentReview{Digest: w.digest}}, true, "Withdraw document")
	w.f.must(err)
	resultGrantTo(t, w.f, "confirm", SampleAdmin2)
	refreshResultActors(w)
	v := proposeResult(t, w, p, "historical")
	_, err = resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
		return w.a.decideResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor, "Self confirm", false)
	})
	documentFailure(t, err, commands.SelfApprovalDenied)
	v = confirmResult(t, w, v)
	_, err = applyResult(w, v)
	documentFailure(t, err, commands.EvidenceStale)
	w.f.s.enterHold("result-test")
	w.f.s.chargeProducer = nil
	revoked, err := resultAs(w, Admin3Login, func(actor string) (financialERPResultView, error) {
		return w.a.decideResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor, "Stop use", true)
	})
	w.f.must(err)
	if revoked.Review.Confirmation != v.Review.Confirmation || revoked.Review.State != "revoked" {
		t.Fatal("history lost")
	}
	read, err := resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
		return w.a.inspectResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor)
	})
	w.f.must(err)
	if read.Source != v.Source || read.CurrentEligibility != "unchecked" {
		t.Fatal("historical evidence changed")
	}
}

type resultLookupLedger struct {
	financialERPLedger
	unavailable bool
}

func (l resultLookupLedger) DocumentOf(workspace, connection, external string) (erpledger.DocumentView, error) {
	if l.unavailable {
		return erpledger.DocumentView{}, errors.New("unavailable")
	}
	return l.financialERPLedger.DocumentOf(workspace, connection, external)
}

func TestFinancialERPResultForeignDocumentAndUnavailableLookup(t *testing.T) {
	for _, mode := range []string{"foreign", "unavailable", "foreign-link", "changed-revision"} {
		t.Run(mode, func(t *testing.T) {
			w, p := resultWorld(t)
			v := confirmResult(t, w, proposeResult(t, w, p, "foreign"))
			intent := resultIntent(v.Basis, SampleAdmin2, v.Review.Digest)
			if mode == "unavailable" {
				w.a.ledger = resultLookupLedger{financialERPLedger: w.a.ledger, unavailable: true}
			} else if mode == "foreign" {
				_, err := w.ledger.Document(SampleWorkspace, intent.document, erpledger.Actor{ID: "foreign-author", Kind: erpledger.KindHuman})
				w.f.must(err)
			} else {
				w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
				_, err := applyResult(w, v)
				if !errors.Is(err, errFinancialERPResultUncertain) {
					t.Fatal(err)
				}
				w.a.afterResultWrite = nil
				if mode == "foreign-link" {
					_, err = w.ledger.Link(SampleWorkspace, intent.links[0], erpledger.Actor{ID: "foreign-author", Kind: erpledger.KindHuman})
					w.f.must(err)
				} else {
					doc := intent.document
					doc.Revision++
					_, err = w.ledger.Document(SampleWorkspace, doc, erpledger.Actor{ID: SampleAdmin2, Kind: erpledger.KindHuman})
					w.f.must(err)
				}
			}
			_, err := applyResult(w, v)
			if err == nil {
				t.Fatal("foreign or unavailable accepted")
			}
			if mode == "foreign" || mode == "unavailable" {
				if len(w.a.resultApplications) != 0 {
					t.Fatal("preflight reserved operation")
				}
			} else if w.a.resultApplications[[2]string{SampleWorkspace, "foreign"}].receipt != "" {
				t.Fatal("conflicting effect marked complete")
			}
		})
	}
}

func TestFinancialERPResultConcurrentApplication(t *testing.T) {
	w, p := resultWorld(t)
	v := confirmResult(t, w, proposeResult(t, w, p, "concurrent"))
	var workers sync.WaitGroup
	receipts := make(chan string, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			got, err := applyResult(w, v)
			if err != nil {
				failures <- err
			} else {
				receipts <- got.ApplicationReceipt
			}
		}()
	}
	workers.Wait()
	close(receipts)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	receipt := ""
	count := 0
	for got := range receipts {
		if receipt != "" && receipt != got {
			t.Fatal("conflicting receipts")
		}
		receipt = got
		count++
	}
	actual, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
	w.f.must(err)
	if count != 8 || !validPlanHash(receipt) || len(actual.Documents) != 1 || len(actual.Documents[0].Links) != 2 || len(w.a.resultApplications) != 1 {
		t.Fatal("duplicate effects")
	}
}

func refreshResultActors(w documentWorld) {
	for login := range w.actors {
		for _, v := range w.f.s.sessions.byLogin(login) {
			w.f.s.endProbe(v)
		}
		w.actors[login] = w.f.as(login)
	}
}
