package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"testing"
)

type failedDocumentLookup struct{ financialERPLedger }

func (f failedDocumentLookup) SubmissionOf(string, string) (erpledger.SubmissionView, error) {
	return erpledger.SubmissionView{}, errors.New("synthetic lookup unavailable")
}

func TestFinancialERPRefusesMissingDependencies(t *testing.T) {
	f := setup(t, SeedSample).ready()
	var missing *erpledger.Registry
	if _, err := newFinancialERP(f.s, missing); err == nil {
		t.Fatal("typed nil accepted")
	}
	w := explicitDocumentWorld(t)
	w.f.s.chargeProducer = nil
	if _, err := w.a.retainDocumentLocked(w.basis); err == nil {
		t.Fatal("missing producer accepted")
	}
	if len(w.a.owners) != 0 {
		t.Fatal("missing dependency created owner")
	}
}

func TestFinancialERPCurrentEvidencePrecedesOwnershipAndRecovery(t *testing.T) {
	for _, mode := range []string{"mapping", "survey", "foreign-identical", "lookup", "capacity", "unknown-stale", "missing-document"} {
		t.Run(mode, func(t *testing.T) {
			w := explicitDocumentWorld(t)
			want := commands.EvidenceStale
			expectedOwners := 0
			switch mode {
			case "mapping":
				m := w.basis.Mappings[1].Mapping
				m.External = "changed-product"
				_, err := w.ledger.Map(SampleWorkspace, m, erpledger.Actor{ID: "mapper", Kind: erpledger.KindHuman})
				w.f.must(err)
			case "survey":
				s := w.basis.Survey.Survey
				s.Evidence = "changed-survey"
				_, err := w.ledger.Survey(SampleWorkspace, s, erpledger.Actor{ID: "surveyor", Kind: erpledger.KindHuman})
				w.f.must(err)
			case "foreign-identical":
				candidate, err := w.f.s.chargeProducer.candidates.financialCandidate(SampleWorkspace, w.basis.Candidate)
				w.f.must(err)
				request, _, err := w.f.s.financialChargeRecord(SampleWorkspace, w.basis.Request)
				w.f.must(err)
				sub, _, err := documentSubmission(w.basis, candidate, request.Subject, "new")
				w.f.must(err)
				_, err = w.ledger.Prepare(SampleWorkspace, sub, erpledger.Actor{ID: SampleAdmin2, Kind: erpledger.KindHuman})
				w.f.must(err)
			case "lookup":
				w.a.ledger = failedDocumentLookup{w.ledger}
				want = commands.DependencyUnavailable
			case "capacity":
				for i := 0; i < financialDocumentLimit; i++ {
					w.a.owners[[2]string{SampleWorkspace, string(rune(i))}] = financialERPOwnership{}
				}
				want = commands.PrerequisiteFailed
				expectedOwners = financialDocumentLimit
			case "unknown-stale":
				w.a.afterPrepare = func() error { return errors.New("synthetic interruption") }
				if _, err := w.prepare("new"); err == nil {
					t.Fatal("expected interruption")
				}
				w.a.afterPrepare = nil
				m := w.basis.Mappings[1].Mapping
				m.External = "changed-product"
				_, err := w.ledger.Map(SampleWorkspace, m, erpledger.Actor{ID: "mapper", Kind: erpledger.KindHuman})
				w.f.must(err)
				expectedOwners = 1
			case "missing-document":
				w.digest = "unknown"
			}
			_, err := w.prepare("new")
			var failure *commands.Failure
			if !errors.As(err, &failure) || failure.Code != want {
				t.Fatal(err, want)
			}
			if len(w.a.owners) != expectedOwners {
				t.Fatal("unexpected ownership", len(w.a.owners))
			}
			if mode != "foreign-identical" && mode != "unknown-stale" {
				if _, err = w.ledger.SubmissionOf(SampleWorkspace, "new"); !errors.Is(err, erpledger.ErrNotFound) {
					t.Fatal("unexpected effect", err)
				}
			}
		})
	}
}
