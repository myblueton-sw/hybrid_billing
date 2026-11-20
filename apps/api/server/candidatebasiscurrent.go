package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/reconcile"
)

type candidateBasisEnvelope struct {
	Mode           string
	Comparison     json.RawMessage
	Document       []byte
	DocumentDigest string
	Run            reconcile.RunView
	Parts          []reconcile.Part
	Readiness      reconcile.Readiness
}

func (r *candidateInputReviews) currentEvidenceLocked(workspace, id string) (inputreview.Evidence, []byte, error) {
	held := r.bindings[[2]string{workspace, id}]
	if held == nil || !held.Opened {
		return inputreview.Evidence{}, nil, errCandidateInvalid
	}
	current := r.documents[candidateDocumentKey{held.Document.Scope, held.Document.ID}]
	sum := sha256.Sum256(held.Bytes)
	if current.Revision != held.Document.Revision || current.Digest != held.Digest || hex.EncodeToString(sum[:]) != held.Digest {
		return inputreview.Evidence{}, nil, errCandidateConflict
	}
	prepared, err := r.prepareBindingLocked(workspace, held.SourceRun, id, held.Bytes, held.Units)
	if err != nil {
		return inputreview.Evidence{}, nil, err
	}
	if prepared.Print != held.Print || prepared.Comparison != held.Comparison {
		return inputreview.Evidence{}, nil, errCandidateConflict
	}
	comparison, err := r.producer.compareSourceLocked(workspace, held.SourceRun, held.Units)
	if err != nil {
		return inputreview.Evidence{}, nil, err
	}
	if comparison.Evidence.Digest != held.Comparison || comparison.Evidence.Verdict != inputreview.Match {
		return inputreview.Evidence{}, nil, errCandidateConflict
	}
	run, err := r.basis.Run(workspace, held.Spec.ID)
	if err != nil {
		return inputreview.Evidence{}, nil, err
	}
	if !reflect.DeepEqual(run.Spec, held.Spec) || run.State != reconcile.Closed || !run.WasClosed || run.Superseded || run.Posted != len(held.Parts) || run.Lines != 0 || len(run.Categories) != 0 {
		return inputreview.Evidence{}, nil, errCandidateInvalid
	}
	parts := []reconcile.Part{}
	for after := 0; after < run.Posted; {
		page, err := r.basis.Parts(workspace, held.Spec.ID, after, 100)
		if err != nil {
			return inputreview.Evidence{}, nil, err
		}
		if len(page) == 0 || len(page) > 100 || after+len(page) > run.Posted {
			return inputreview.Evidence{}, nil, errCandidateInvalid
		}
		parts = append(parts, page...)
		after += len(page)
	}
	if !reflect.DeepEqual(parts, prepared.Parts) {
		return inputreview.Evidence{}, nil, errCandidateConflict
	}
	readiness, err := r.basis.Readiness(workspace, held.Spec.ID)
	if err != nil {
		return inputreview.Evidence{}, nil, err
	}
	envelope, err := json.Marshal(candidateBasisEnvelope{Mode: "synthetic", Comparison: comparison.Envelope, Document: held.Bytes, DocumentDigest: held.Digest, Run: run, Parts: parts, Readiness: readiness})
	if err != nil {
		return inputreview.Evidence{}, nil, err
	}
	if len(envelope) > candidateBytes {
		return inputreview.Evidence{}, nil, errCandidateLimit
	}
	sum = sha256.Sum256(envelope)
	evidence := inputreview.Evidence{Scope: held.Document.Scope, Run: id, Digest: hex.EncodeToString(sum[:]), Method: "synthetic-source-document-v1", Verdict: inputreview.Match}
	if run.Verdict != reconcile.Explained || run.Residual != "0" || run.Explained != "0" || readiness.Verdict != reconcile.Ready || readiness.Cost != reconcile.Determined || len(readiness.Reasons) != 0 {
		evidence.Verdict = inputreview.Difference
		evidence.Findings = []string{"basis_not_ready"}
	}
	return evidence, envelope, nil
}

func (r *candidateInputReviews) Compare(workspace, id string) (inputreview.Evidence, error) {
	evidence, _, err := r.currentEvidenceLocked(workspace, id)
	return evidence, err
}
