package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/reconcile"
)

type candidateDocument struct {
	Mode     string
	Scope    inputreview.Scope
	ID       string
	Revision int
	Currency string
	Field    string
	Basis    normalize.CostKind
	Total    string
}

type candidateDocumentKey struct {
	Scope inputreview.Scope
	ID    string
}

type candidateDocumentRevision struct {
	Revision int
	Digest   string
}

type candidateBasisBinding struct {
	ID         string
	SourceRun  string
	Print      string
	Document   candidateDocument
	Bytes      []byte
	Digest     string
	Comparison string
	Units      []inputreview.UnitBinding
	Spec       reconcile.RunSpec
	Parts      []reconcile.Part
	Opened     bool
}

type candidateInputReviews struct {
	producer  *normalizedCharges
	basis     *reconcile.Registry
	registry  *inputreview.Registry
	bindings  map[[2]string]*candidateBasisBinding
	documents map[candidateDocumentKey]candidateDocumentRevision
	envelopes map[string][]byte
}

func newCandidateInputReviews(p *normalizedCharges, authority inputreview.Authority, clock func() time.Time) (*candidateInputReviews, error) {
	if p == nil || p.domain == nil {
		return nil, errCandidateInvalid
	}
	basis, err := reconcile.NewRegistry(reconcile.Limits{Runs: 1000, Parts: 1000, Lines: 1000, Page: 100, Listed: 1000}, nil, p.sourceProfile, clock)
	if err != nil {
		return nil, err
	}
	r := &candidateInputReviews{producer: p, basis: basis, bindings: make(map[[2]string]*candidateBasisBinding), documents: make(map[candidateDocumentKey]candidateDocumentRevision), envelopes: make(map[string][]byte)}
	r.registry, err = inputreview.NewRegistry(1000, authority, r, clock)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *candidateInputReviews) prepareBindingLocked(workspace, sourceRun, id string, body []byte, units []inputreview.UnitBinding) (*candidateBasisBinding, error) {
	if !validRequestID(workspace) || !validRequestID(sourceRun) || !validRequestID(id) || len(body) == 0 || len(body) > candidateBytes {
		return nil, errCandidateInvalid
	}
	shape := json.NewDecoder(bytes.NewReader(body))
	shape.UseNumber()
	if !candidateDocumentShape(shape, 0) {
		return nil, errCandidateInvalid
	}
	var document candidateDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, errCandidateInvalid
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errCandidateInvalid
	}
	remaining := candidateBytes
	if !candidateText(reflect.ValueOf(document), &remaining) || document.Mode != "synthetic" || !validRequestID(document.ID) || document.Revision < 1 || document.Revision > 999999 || document.Basis != normalize.Billed || !validRequestID(document.Field) {
		return nil, errCandidateInvalid
	}
	snapshot, err := r.producer.comparisonSnapshotLocked(workspace, sourceRun, units)
	if err != nil {
		return nil, err
	}
	comparison, err := inputreview.CompareSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	if comparison.Evidence.Verdict != inputreview.Match || document.Scope != snapshot.Scope {
		return nil, errCandidateInvalid
	}
	found := false
	for _, field := range snapshot.Target.Fields {
		if field.Name == document.Field && field.Kind == normalize.Cost && field.Cost == document.Basis {
			found = true
		}
	}
	if !found {
		return nil, errCandidateInvalid
	}
	total, err := r.producer.sourceAmount(document.Total)
	if err != nil {
		return nil, err
	}
	documentSum := sha256.Sum256(body)
	digest := hex.EncodeToString(documentSum[:])
	documentKey := candidateDocumentKey{document.Scope, document.ID}
	documentID, err := normalizedDigest(documentKey)
	if err != nil {
		return nil, err
	}
	runID, err := normalizedDigest([2]string{workspace, id})
	if err != nil {
		return nil, err
	}
	held := &candidateBasisBinding{ID: id, SourceRun: sourceRun, Document: document, Bytes: append([]byte(nil), body...), Digest: digest, Comparison: comparison.Evidence.Digest, Units: append([]inputreview.UnitBinding(nil), units...)}
	for _, row := range snapshot.Rows {
		cell := row.Cells[document.Field]
		currency := row.Cells[snapshot.Target.CostCurrency]
		if cell.State != normalize.Value || currency.State != normalize.Value || currency.Text != document.Currency {
			return nil, errCandidateInvalid
		}
		amount, err := r.producer.sourceAmount(cell.Text)
		if err != nil {
			return nil, err
		}
		identity, err := normalizedDigest(struct {
			Scope inputreview.Scope
			Key   normalize.RowKey
		}{snapshot.Scope, row.Key})
		if err != nil {
			return nil, err
		}
		held.Parts = append(held.Parts, reconcile.Part{ID: identity, Amount: amount, Currency: currency.Text, Kind: reconcile.Billed, Source: identity})
	}
	sort.Slice(held.Parts, func(i, j int) bool { return held.Parts[i].ID < held.Parts[j].ID })
	held.Spec = reconcile.RunSpec{WorkspaceID: workspace, ID: runID, Basis: reconcile.ProviderDocument, Issuer: document.Scope.Provider, Account: document.Scope.Account, Document: documentID, Revision: document.Revision, Currency: document.Currency, Total: total, Input: comparison.Evidence.Digest, Parts: len(held.Parts), Rounding: "0"}
	held.Print, err = normalizedDigest(struct {
		ID, SourceRun string
		Body          []byte
		Spec          reconcile.RunSpec
		Parts         []reconcile.Part
		Units         []inputreview.UnitBinding
	}{id, sourceRun, body, held.Spec, held.Parts, held.Units})
	if err != nil {
		return nil, err
	}
	return held, nil
}

func (r *candidateInputReviews) prepareLocked(workspace, sourceRun, id string, body []byte, units []inputreview.UnitBinding) (inputreview.Record, error) {
	prepared, err := r.prepareBindingLocked(workspace, sourceRun, id, body, units)
	if err != nil {
		return inputreview.Record{}, err
	}
	key := [2]string{workspace, id}
	held := r.bindings[key]
	if held != nil && held.Print != prepared.Print {
		return inputreview.Record{}, errCandidateConflict
	}
	docKey := candidateDocumentKey{prepared.Document.Scope, prepared.Document.ID}
	latest := r.documents[docKey]
	if latest.Revision > prepared.Document.Revision || latest.Revision == prepared.Document.Revision && latest.Digest != prepared.Digest {
		return inputreview.Record{}, errCandidateConflict
	}
	if held == nil {
		if len(r.bindings) >= 1000 {
			return inputreview.Record{}, errCandidateLimit
		}
		if _, err := r.basis.Run(workspace, prepared.Spec.ID); err == nil {
			return inputreview.Record{}, errCandidateConflict
		} else if !errors.Is(err, reconcile.ErrNotFound) {
			return inputreview.Record{}, err
		}
		held = prepared
		r.bindings[key] = held
		r.documents[docKey] = candidateDocumentRevision{prepared.Document.Revision, prepared.Digest}
	}
	if !held.Opened {
		if _, err := r.basis.Run(workspace, held.Spec.ID); err == nil {
			return inputreview.Record{}, errCandidateConflict
		} else if !errors.Is(err, reconcile.ErrNotFound) {
			return inputreview.Record{}, err
		}
	}
	state, err := r.basis.Open(held.Spec)
	if err != nil {
		return inputreview.Record{}, err
	}
	if state != reconcile.Open && state != reconcile.Closed {
		return inputreview.Record{}, errCandidateInvalid
	}
	held.Opened = true
	for _, part := range held.Parts {
		state, err = r.basis.Post(workspace, held.Spec.ID, part)
		if err != nil {
			return inputreview.Record{}, err
		}
		if state != reconcile.Open && state != reconcile.Closed {
			return inputreview.Record{}, errCandidateInvalid
		}
	}
	state, _, err = r.basis.Close(workspace, held.Spec.ID)
	if err != nil {
		return inputreview.Record{}, err
	}
	if state != reconcile.Closed {
		return inputreview.Record{}, errCandidateInvalid
	}
	evidence, envelope, err := r.currentEvidenceLocked(workspace, id)
	if err != nil {
		return inputreview.Record{}, err
	}
	record, err := r.registry.Compare(workspace, id, id)
	if err != nil {
		return inputreview.Record{}, err
	}
	if record.Evidence.Digest != evidence.Digest {
		return inputreview.Record{}, errCandidateConflict
	}
	r.envelopes[evidence.Digest] = append([]byte(nil), envelope...)
	return record, nil
}

func candidateDocumentShape(decoder *json.Decoder, depth int) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	if delimiter, object := token.(json.Delim); object {
		if delimiter != '{' || depth > 1 {
			return false
		}
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			name, valid := key.(string)
			name = strings.ToLower(name)
			if err != nil || !valid || seen[name] || len(seen) >= 16 {
				return false
			}
			seen[name] = true
			if !candidateDocumentShape(decoder, depth+1) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim('}')
	}
	if depth == 0 {
		return false
	}
	switch token.(type) {
	case string, json.Number:
		return true
	}
	return false
}
