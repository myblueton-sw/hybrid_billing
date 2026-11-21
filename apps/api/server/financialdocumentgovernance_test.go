package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func documentAs(w documentWorld, login string, run func(string) (financialDocumentView, error)) (financialDocumentView, error) {
	var out financialDocumentView
	err := inputAuthorityAs(w.f, w.actors[login], func() error { var err error; out, err = run(w.f.s.answer.User()); return err })
	return out, err
}
func documentFailure(t *testing.T, err error, want commands.Code) {
	t.Helper()
	var failure *commands.Failure
	if !errors.As(err, &failure) || failure.Code != want {
		if failure == nil {
			t.Fatalf("non-command error %T: %v, want %s", err, err, want)
		}
		t.Fatalf("got code=%s, want code=%s", failure.Code, want)
	}
}
func pendingDocument(t *testing.T, w documentWorld, id string) financialDocumentView {
	t.Helper()
	b := cloneDocumentValue(t, w.basis)
	b.ID = id
	v, err := documentAs(w, Admin2Login, func(actor string) (financialDocumentView, error) {
		return w.a.proposeDocumentLocked(b, actor, "Explicit packet proposal")
	})
	w.f.must(err)
	if v.Review.State != "pending" {
		t.Fatal(v.Review)
	}
	return v
}
func decideDocument(w documentWorld, login string, v financialDocumentView, revoke bool, reason string) (financialDocumentView, error) {
	return documentAs(w, login, func(actor string) (financialDocumentView, error) {
		return w.a.decideDocumentLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor, reason, revoke)
	})
}
func inspectDocument(w documentWorld, login, id, digest string) (financialDocumentView, error) {
	return documentAs(w, login, func(actor string) (financialDocumentView, error) {
		return w.a.inspectDocumentLocked(SampleWorkspace, id, digest, actor)
	})
}

func TestFinancialDocumentGovernedHistoryAndImmutablePacket(t *testing.T) {
	w := explicitDocumentWorld(t)
	before := w.f.s.Revision()
	v := pendingDocument(t, w, "governed")
	if v.BasisDigest == v.Review.Digest || v.Review.Proposer != SampleAdmin2 {
		t.Fatal("proposal provenance")
	}
	v.Basis.Lines[0].Tax = "99"
	v.Review.Proposer = "tampered"
	got, err := inspectDocument(w, Admin2Login, "governed", v.Review.Digest)
	w.f.must(err)
	if got.CurrentEligibility != "unchecked" || got.Basis.Lines[0].Tax == "99" || got.Review.Proposer != SampleAdmin2 {
		t.Fatal("mutable retained packet")
	}
	changed := cloneDocumentValue(t, got.Basis)
	changed.Lines[0].Quantity = "4"
	_, err = documentAs(w, Admin2Login, func(actor string) (financialDocumentView, error) {
		return w.a.proposeDocumentLocked(changed, actor, "Explicit packet proposal")
	})
	documentFailure(t, err, commands.EvidenceStale)
	confirmed, err := decideDocument(w, Admin3Login, got, false, "Independent confirmation")
	w.f.must(err)
	again, err := decideDocument(w, Admin3Login, got, false, "Independent confirmation")
	w.f.must(err)
	if again.Review != confirmed.Review || confirmed.Review.Confirmation.Actor != SampleAdmin3 {
		t.Fatal("confirmation history changed")
	}
	_, err = decideDocument(w, Admin3Login, got, false, "Changed reason")
	documentFailure(t, err, commands.EvidenceStale)
	revoked, err := decideDocument(w, Admin3Login, got, true, "Evidence withdrawn")
	w.f.must(err)
	if revoked.Review.State != "revoked" || revoked.Review.Confirmation != confirmed.Review.Confirmation {
		t.Fatal("revocation erased history")
	}
	_, err = decideDocument(w, Admin3Login, got, false, "Independent confirmation")
	documentFailure(t, err, commands.PrerequisiteFailed)
	same, err := decideDocument(w, Admin3Login, got, true, "Evidence withdrawn")
	w.f.must(err)
	if same.Review != revoked.Review || w.f.s.Revision() != before {
		t.Fatal("historical mutation or private revision")
	}
}

func TestFinancialDocumentAuthorityBeforeLookupAndIndependentConfirmation(t *testing.T) {
	for _, mode := range []string{"self", "proposer-revoked", "confirmation-denied", "inspect-denied", "basis-digest"} {
		t.Run(mode, func(t *testing.T) {
			w := explicitDocumentWorld(t)
			v := pendingDocument(t, w, "pending")
			want := commands.PermissionDenied
			var err error
			switch mode {
			case "self":
				documentGrantTo(t, w.f, "confirm", SampleAdmin2)
				_, err = decideDocument(w, Admin2Login, v, false, "Self confirmation")
				want = commands.SelfApprovalDenied
			case "proposer-revoked":
				for _, g := range w.f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ProposeDocumentEvidence {
						w.f.must(w.f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
				_, err = decideDocument(w, Admin3Login, v, false, "Current proposer required")
			case "confirmation-denied":
				_, err = decideDocument(w, AdminLogin, v, false, "Administrator is not permission")
			case "inspect-denied":
				for _, id := range []string{v.Basis.ID, "missing"} {
					_, e := inspectDocument(w, AdminLogin, id, v.Review.Digest)
					documentFailure(t, e, commands.PermissionDenied)
				}
				_, err = inspectDocument(w, AdminLogin, v.Basis.ID, v.Review.Digest)
			case "basis-digest":
				v.Review.Digest = v.BasisDigest
				_, err = decideDocument(w, Admin3Login, v, false, "Basis alone insufficient")
				want = commands.EvidenceStale
			}
			documentFailure(t, err, want)
			retained := w.a.documents[[2]string{SampleWorkspace, "pending"}]
			if retained.review.State != "pending" || retained.review.Confirmation.Actor != "" {
				t.Fatal("refusal recorded confirmation")
			}
		})
	}
}

func TestFinancialDocumentRevocationPreservesUncertainERPHistory(t *testing.T) {
	w := explicitDocumentWorld(t)
	w.a.afterPrepare = func() error { return errors.New("synthetic interrupted receipt") }
	_, err := w.prepare("uncertain-revoked")
	if !errors.Is(err, errFinancialERPUncertain) {
		t.Fatal(err)
	}
	actual, err := w.ledger.SubmissionOf(SampleWorkspace, "uncertain-revoked")
	w.f.must(err)
	w.f.s.domain.Lock()
	w.f.s.enterHold("synthetic-test")
	w.f.s.domain.Unlock()
	w.f.s.chargeProducer = nil
	v, err := inspectDocument(w, Admin3Login, w.basis.ID, w.digest)
	w.f.must(err)
	revoked, err := decideDocument(w, Admin3Login, v, true, "Stop future use under hold")
	w.f.must(err)
	if revoked.Review.State != "revoked" {
		t.Fatal("not revoked")
	}
	_, err = w.prepare("uncertain-revoked")
	documentFailure(t, err, commands.PrerequisiteFailed)
	owner := w.a.owners[[2]string{SampleWorkspace, "uncertain-revoked"}]
	if owner.receipt != "" || len(w.a.owners) != 1 {
		t.Fatal("unknown ownership lost")
	}
	retained, err := w.ledger.SubmissionOf(SampleWorkspace, "uncertain-revoked")
	w.f.must(err)
	if retained.Digest != actual.Digest || retained.State != erpledger.StatePrepared {
		t.Fatal("revocation cancelled or changed effect")
	}
	history, err := inspectDocument(w, Admin2Login, w.basis.ID, w.digest)
	w.f.must(err)
	if history.Review.State != "revoked" || history.Review.Confirmation.Actor == "" {
		t.Fatal("history unavailable")
	}
}

func TestFinancialDocumentFreshChecksAndAtomicRefusals(t *testing.T) {
	for _, mode := range []string{"pending", "mapping", "hold", "clock", "capacity"} {
		t.Run(mode, func(t *testing.T) {
			w := explicitDocumentWorld(t)
			v := pendingDocument(t, w, "fresh")
			switch mode {
			case "pending":
				w.basis.ID, w.digest = v.Basis.ID, v.Review.Digest
				_, err := w.prepare("pending-effect")
				documentFailure(t, err, commands.PrerequisiteFailed)
			case "mapping":
				m := w.basis.Mappings[1].Mapping
				m.External = "changed"
				_, err := w.ledger.Map(SampleWorkspace, m, erpledger.Actor{ID: "mapper", Kind: erpledger.KindHuman})
				w.f.must(err)
				_, err = decideDocument(w, Admin3Login, v, false, "Stale mapping")
				documentFailure(t, err, commands.EvidenceStale)
			case "hold":
				w.f.s.domain.Lock()
				w.f.s.enterHold("synthetic-test")
				w.f.s.domain.Unlock()
				_, err := decideDocument(w, Admin3Login, v, false, "Held")
				documentFailure(t, err, commands.PrerequisiteFailed)
				_, err = decideDocument(w, Admin3Login, v, true, "Pending withdrawal during hold")
				w.f.must(err)
			case "clock":
				w.a.documentTime = w.f.s.clock().Add(time.Hour)
				_, err := decideDocument(w, Admin3Login, v, false, "Clock failure")
				documentFailure(t, err, commands.DependencyUnavailable)
				if w.a.documents[[2]string{SampleWorkspace, v.Basis.ID}].review.State != "pending" {
					t.Fatal("clock failure changed record")
				}
				_, err = w.prepare("clock-effect")
				documentFailure(t, err, commands.DependencyUnavailable)
			case "capacity":
				for i := len(w.a.documents); i < financialDocumentLimit; i++ {
					w.a.documents[[2]string{"other", string(rune(i))}] = financialDocumentRecord{}
				}
				b := cloneDocumentValue(t, w.basis)
				b.ID = "over-capacity"
				_, err := documentAs(w, Admin2Login, func(actor string) (financialDocumentView, error) {
					return w.a.proposeDocumentLocked(b, actor, "No orphan bytes")
				})
				documentFailure(t, err, commands.PrerequisiteFailed)
				if _, ok := w.a.documents[[2]string{SampleWorkspace, b.ID}]; ok {
					t.Fatal("orphan packet")
				}
			}
			if len(w.a.owners) != 0 {
				t.Fatal("refusal reserved effect")
			}
		})
	}
}

func TestFinancialDocumentConfirmationIsNotImplicitlyRevoked(t *testing.T) {
	w := explicitDocumentWorld(t)
	for _, g := range w.f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin3 && g.Terms.Action == gate.ConfirmDocumentEvidence {
			w.f.must(w.f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	v, err := w.prepare("explicit-revocation-model")
	w.f.must(err)
	if v.State != erpledger.StatePrepared {
		t.Fatal("confirmed evidence implicitly revoked")
	}
}

func TestFinancialDocumentUnknownPreparationRequiresCurrentDedicatedGrant(t *testing.T) {
	w := explicitDocumentWorld(t)
	w.a.afterPrepare = func() error { return errors.New("synthetic interruption") }
	_, err := w.prepare("permission-recovery")
	if !errors.Is(err, errFinancialERPUncertain) {
		t.Fatal(err)
	}
	actual, err := w.ledger.SubmissionOf(SampleWorkspace, "permission-recovery")
	w.f.must(err)
	for _, g := range w.f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.PrepareERPRecord {
			w.f.must(w.f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	_, err = w.prepare("permission-recovery")
	documentFailure(t, err, commands.PermissionDenied)
	owner := w.a.owners[[2]string{SampleWorkspace, "permission-recovery"}]
	retained, err := w.ledger.SubmissionOf(SampleWorkspace, "permission-recovery")
	w.f.must(err)
	if owner.receipt != "" || retained.Digest != actual.Digest || len(w.a.owners) != 1 {
		t.Fatal("refusal erased unknown effect")
	}
}

func TestFinancialDocumentConcurrentDecisionsAndPreparation(t *testing.T) {
	for _, mode := range []string{"confirm-revoke", "prepare-revoke"} {
		t.Run(mode, func(t *testing.T) {
			w := explicitDocumentWorld(t)
			v, err := inspectDocument(w, Admin3Login, w.basis.ID, w.digest)
			w.f.must(err)
			if mode == "confirm-revoke" {
				v = pendingDocument(t, w, "competing")
			}
			start := make(chan struct{})
			outcomes := make(chan error, 2)
			go func() {
				<-start
				_, e := decideDocument(w, Admin3Login, v, true, "Concurrent explicit revocation")
				outcomes <- e
			}()
			go func() {
				<-start
				if mode == "confirm-revoke" {
					_, e := decideDocument(w, Admin3Login, v, false, "Concurrent independent confirmation")
					outcomes <- e
					return
				}
				_, e := w.prepare("concurrent-preparation")
				outcomes <- e
			}()
			close(start)
			for i := 0; i < 2; i++ {
				if e := <-outcomes; e != nil {
					documentFailure(t, e, commands.PrerequisiteFailed)
				}
			}
			history, err := inspectDocument(w, Admin2Login, v.Basis.ID, v.Review.Digest)
			w.f.must(err)
			if history.Review.State != "revoked" {
				t.Fatal("revocation lost")
			}
			if mode == "prepare-revoke" {
				_, err = w.prepare("concurrent-preparation")
				documentFailure(t, err, commands.PrerequisiteFailed)
				actual, lookup := w.ledger.SubmissionOf(SampleWorkspace, "concurrent-preparation")
				if lookup != nil && !errors.Is(lookup, erpledger.ErrNotFound) {
					t.Fatal(lookup)
				}
				if lookup == nil && (actual.State != erpledger.StatePrepared || len(w.a.owners) != 1) {
					t.Fatal("unowned or changed concurrent effect")
				}
				if lookup != nil && len(w.a.owners) != 0 {
					t.Fatal("orphan concurrent owner")
				}
			}
		})
	}
}
