package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/reconcile"
)

func installPartialBasis(t *testing.T, r *candidateInputReviews, body []byte) *candidateBasisBinding {
	t.Helper()
	held, err := r.prepareBindingLocked(SampleWorkspace, "sample-normalized-charge", "partial", body, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.bindings[[2]string{SampleWorkspace, "partial"}] = held
	r.documents[candidateDocumentKey{held.Document.Scope, held.Document.ID}] = candidateDocumentRevision{held.Document.Revision, held.Digest}
	if _, err := r.basis.Open(held.Spec); err != nil {
		t.Fatal(err)
	}
	held.Opened = true
	return held
}

func TestCandidateBasisPartialRetryAndChangedDocument(t *testing.T) {
	r, _, document := candidateBasisFixture(t)
	body := candidateDocumentBytes(t, document)
	held := installPartialBasis(t, r, body)
	if _, err := r.basis.Post(SampleWorkspace, held.Spec.ID, held.Parts[0]); err != nil {
		t.Fatal(err)
	}
	document.Total = "16"
	if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "partial", candidateDocumentBytes(t, document), nil); !errors.Is(err, errCandidateConflict) {
		t.Fatal("changed partial retry", err)
	}
	record, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "partial", body, nil)
	if err != nil || record.State != inputreview.Awaiting {
		t.Fatal("owned retry", record, err)
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(record, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateBasisCompletePartsCannotHideSwappedAmounts(t *testing.T) {
	for _, kind := range []string{"missing", "swapped", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			r, _, document := candidateBasisFixture(t)
			held := installPartialBasis(t, r, candidateDocumentBytes(t, document))
			first, second := held.Parts[0], held.Parts[1]
			if kind == "swapped" {
				first.Amount, second.Amount = second.Amount, first.Amount
			}
			if _, err := r.basis.Post(SampleWorkspace, held.Spec.ID, first); err != nil {
				t.Fatal(err)
			}
			if kind == "duplicate" {
				second.Source = first.Source
				if _, err := r.basis.Post(SampleWorkspace, held.Spec.ID, second); err == nil {
					t.Fatal("duplicate source accepted")
				}
			} else if kind == "swapped" {
				if _, err := r.basis.Post(SampleWorkspace, held.Spec.ID, second); err != nil {
					t.Fatal(err)
				}
				if _, verdict, err := r.basis.Close(SampleWorkspace, held.Spec.ID); err != nil || verdict != reconcile.Explained {
					t.Fatal("aggregate fixture", verdict, err)
				}
			}
			if _, err := r.Compare(SampleWorkspace, "partial"); err == nil {
				t.Fatal("incomplete or swapped basis accepted")
			}
			if _, err := r.registry.Record(SampleWorkspace, "partial"); !errors.Is(err, inputreview.ErrNotFound) {
				t.Fatal("invalid basis created review", err)
			}
		})
	}
}

func TestCandidateDocumentOriginalBytesAndDuplicateFields(t *testing.T) {
	r, _, document := candidateBasisFixture(t)
	body := candidateDocumentBytes(t, document)
	duplicate := append(append([]byte(nil), body[:len(body)-1]...), []byte(`,"Total":"15"}`)...)
	if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "duplicate", duplicate, nil); err == nil {
		t.Fatal("duplicate document key")
	}
	record, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "original", body, nil)
	if err != nil {
		t.Fatal(err)
	}
	spaced := append([]byte(" \n"), body...)
	if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "other", spaced, nil); !errors.Is(err, errCandidateConflict) {
		t.Fatal("changed original bytes at same revision", err)
	}
	held := r.bindings[[2]string{SampleWorkspace, "original"}]
	if !bytes.Equal(body, held.Bytes) {
		t.Fatal("original bytes not retained")
	}
	held.Bytes[0] = 'x'
	if _, err := r.registry.Confirm(candidateReviewIntent(record, 0)); err == nil {
		t.Fatal("corrupted original accepted")
	}
}

func TestCandidateBasisSameDocumentReprepareRequiresExplicitVoid(t *testing.T) {
	r, _, document := candidateBasisFixture(t)
	body := candidateDocumentBytes(t, document)
	first, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "first", body, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "second", body, nil); err == nil {
		t.Fatal("two live runs for one document")
	}
	prior := r.bindings[[2]string{SampleWorkspace, "first"}]
	if err := r.basis.Void(SampleWorkspace, prior.Spec.ID, reconcile.WrongFigures); err != nil {
		t.Fatal(err)
	}
	next, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "second", body, nil)
	if err != nil || next.State != inputreview.Awaiting {
		t.Fatal("same original after explicit void", next, err)
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(first, 0)); err == nil {
		t.Fatal("voided original review")
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(next, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateBasisCurrentProducerChangesInvalidateConfirmation(t *testing.T) {
	for _, kind := range []string{"unpublished source", "published source", "mapping"} {
		t.Run(kind, func(t *testing.T) {
			r, _, document := candidateBasisFixture(t)
			record, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "review", candidateDocumentBytes(t, document), nil)
			if err != nil {
				t.Fatal(err)
			}
			confirmed, err := r.registry.Confirm(candidateReviewIntent(record, 0))
			if err != nil {
				t.Fatal(err)
			}
			p := r.producer
			if kind == "mapping" {
				run, err := p.normal.Run(SampleWorkspace, "sample-normalized-charge")
				if err != nil {
					t.Fatal(err)
				}
				mapping, err := p.normal.Revision(run.Spec.Mapping, run.Spec.Revision)
				if err != nil {
					t.Fatal(err)
				}
				for i := range mapping.Entries {
					if mapping.Entries[i].Field == "class" {
						mapping.Entries[i].Source = "service"
					}
				}
				if revision, err := p.normal.DeclareMapping(run.Spec.Mapping, run.Spec.Revision, mapping.Entries); err != nil || revision != 2 {
					t.Fatal(revision, err)
				}
			} else {
				held := p.sources[[2]string{SampleWorkspace, "sample-normalized-charge"}]
				var packet sourcePacket
				if err := json.Unmarshal(held.Packet, &packet); err != nil {
					t.Fatal(err)
				}
				next := nextSourcePacket(t, packet, "new-source", 2)
				next.Artifacts[0].Rows[0].Cells["service"] = normalize.Cell{Text: "new-service"}
				if kind == "published source" {
					if _, err := p.assembleSourceLocked(next); err != nil {
						t.Fatal(err)
					}
				} else {
					prepared, err := p.prepareSource(next)
					if err != nil {
						t.Fatal(err)
					}
					assembly, err := p.reserveSource(prepared)
					if err != nil {
						t.Fatal(err)
					}
					for i := range prepared.specs {
						if err := p.stageSourceArtifact(prepared, assembly, i); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			if _, err := r.registry.Confirm(candidateReviewIntent(record, 0)); err == nil {
				t.Fatal("changed producer retained confirmation")
			}
			if history, _ := r.registry.Record(SampleWorkspace, "review"); !reflect.DeepEqual(history, confirmed) {
				t.Fatal("producer change rewrote receipt")
			}
		})
	}
}

func TestCandidateBasisFailedNewerPreparationDoesNotRestoreOldDocument(t *testing.T) {
	r, _, document := candidateBasisFixture(t)
	limited, err := reconcile.NewRegistry(reconcile.Limits{Runs: 1, Parts: 10, Lines: 10, Page: 100, Listed: 10}, nil, r.producer.sourceProfile, func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	r.basis = limited
	first, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "first", candidateDocumentBytes(t, document), nil)
	if err != nil {
		t.Fatal(err)
	}
	confirmed, err := r.registry.Confirm(candidateReviewIntent(first, 0))
	if err != nil {
		t.Fatal(err)
	}
	document.Revision = 2
	if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "second", candidateDocumentBytes(t, document), nil); !errors.Is(err, reconcile.ErrLimit) {
		t.Fatal("newer preparation capacity", err)
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(first, 0)); err == nil {
		t.Fatal("known newer document rolled back")
	}
	if old, _ := r.registry.Record(SampleWorkspace, "first"); !reflect.DeepEqual(old, confirmed) {
		t.Fatal("failed preparation rewrote history")
	}
}
