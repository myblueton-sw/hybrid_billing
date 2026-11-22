package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type financialERPLedger interface {
	ResultOf(string, string) (erpledger.ResultView, error)
	ReplaceResult(string, string, erpledger.ResultView, erpledger.Document, []erpledger.Link, erpledger.Actor) (erpledger.ResultView, error)
	CheckResultReplacement(string, string, erpledger.ResultView, erpledger.Document, []erpledger.Link, erpledger.Actor) error
	DocumentOf(string, string, string) (erpledger.DocumentView, error)
	Document(string, erpledger.Document, erpledger.Actor) (erpledger.DocumentView, error)
	Link(string, erpledger.Link, erpledger.Actor) (erpledger.DocumentView, error)
	ConnectionOf(string, string) (erpledger.ConnectionView, error)
	MappingOf(string, string, string, string) (erpledger.MappingView, error)
	SurveyOf(string, string, string, string) (erpledger.SurveyView, error)
	SubmissionOf(string, string) (erpledger.SubmissionView, error)
	CheckPreparation(string, erpledger.Submission, erpledger.Actor) (erpledger.PreparationCheck, error)
	Prepare(string, erpledger.Submission, erpledger.Actor) (erpledger.SubmissionView, error)
}

var errFinancialERPUncertain = errors.New("ERP preparation outcome remains unresolved")

type financialDocumentRecord struct {
	raw    []byte
	digest string
	review financialDocumentReview
}
type financialERPOwnership struct {
	basis, actor, receipt string
	submission            erpledger.Submission
	total                 erpledger.Amounts
	mappings              []string
}

type financialERP struct {
	server             *Server
	ledger             financialERPLedger
	documents          map[[2]string]financialDocumentRecord
	owners             map[[2]string]financialERPOwnership
	results            map[[2]string]financialERPResultRecord
	resultApplications map[[2]string]financialERPResultApplication
	resultExternals    map[[3]string]string
	resultPreparations map[[2]string]string
	resultTime         time.Time
	afterResultWrite   func(string) error
	documentTime       time.Time
	afterPrepare       func() error
}

func newFinancialERP(s *Server, ledger financialERPLedger) (*financialERP, error) {
	if s == nil || s.cfg.Seed != SeedSample || ledger == nil || reflect.ValueOf(ledger).Kind() == reflect.Pointer && reflect.ValueOf(ledger).IsNil() {
		return nil, commands.Fail(commands.UnsupportedCapability)
	}
	return &financialERP{server: s, ledger: ledger, documents: map[[2]string]financialDocumentRecord{}, owners: map[[2]string]financialERPOwnership{}, results: map[[2]string]financialERPResultRecord{}, resultApplications: map[[2]string]financialERPResultApplication{}, resultExternals: map[[3]string]string{}, resultPreparations: map[[2]string]string{}}, nil
}

func (a *financialERP) validateDocumentLocked(b financialDocumentBasis) (financialDocumentRecord, error) {
	if a.server.chargeProducer == nil {
		return financialDocumentRecord{}, commands.Fail(commands.UnsupportedCapability)
	}
	raw, err := documentBytes(b)
	if err != nil {
		return financialDocumentRecord{}, err
	}
	if b.Workspace != a.server.bootWorkspace {
		return financialDocumentRecord{}, errCandidateInvalid
	}
	v, binding, err := a.server.financialChargeRecord(b.Workspace, b.Request)
	if err != nil || v.Digest != b.RequestDigest || binding.Candidate != b.Candidate || binding.CandidateDigest != b.CandidateDigest {
		return financialDocumentRecord{}, errCandidateConflict
	}
	c, err := a.server.chargeProducer.candidates.financialCandidate(b.Workspace, b.Candidate)
	if err != nil {
		return financialDocumentRecord{}, err
	}
	if _, _, err = documentSubmission(b, c, binding.Subject, b.ID); err != nil {
		return financialDocumentRecord{}, err
	}
	if err = a.currentDocumentERP(b); err != nil {
		return financialDocumentRecord{}, err
	}
	digest, err := normalizedDigest(b)
	if err != nil {
		return financialDocumentRecord{}, err
	}
	return financialDocumentRecord{raw: append([]byte(nil), raw...), digest: digest}, nil
}

func (a *financialERP) document(workspace, id, digest string) (financialDocumentBasis, error) {
	record, ok := a.documents[[2]string{workspace, id}]
	if !ok || record.digest != digest {
		return financialDocumentBasis{}, errCandidateMissing
	}
	var b financialDocumentBasis
	if err := json.Unmarshal(record.raw, &b); err != nil {
		return b, errCandidateConflict
	}
	actual, err := normalizedDigest(b)
	if err != nil || actual != digest || b.Workspace != workspace || b.ID != id {
		return b, errCandidateConflict
	}
	return b, nil
}

func (a *financialERP) currentDocumentERP(b financialDocumentBasis) error {
	conn, err := a.ledger.ConnectionOf(b.Workspace, b.Connection.Connection.ID)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if !reflect.DeepEqual(conn, b.Connection) {
		return commands.Fail(commands.EvidenceStale)
	}
	for _, want := range b.Mappings {
		got, err := a.ledger.MappingOf(b.Workspace, want.Mapping.Connection, want.Mapping.Kind, want.Mapping.Platform)
		if err != nil {
			return commands.Fail(commands.DependencyUnavailable)
		}
		if got != want {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	survey, err := a.ledger.SurveyOf(b.Workspace, b.Survey.Survey.Connection, b.Survey.Survey.Customer, b.Survey.Survey.Period)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if survey != b.Survey {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialERP) prepareLocked(workspace, id, digest, key, consumer string) (erpledger.SubmissionView, error) {
	fail := func(code commands.Code) (erpledger.SubmissionView, error) {
		return erpledger.SubmissionView{}, commands.Fail(code)
	}
	if !validRequestID(key) {
		return fail(commands.InvalidRequest)
	}
	if err := a.documentPermission(workspace, consumer, gate.PrepareERPRecord); err != nil {
		return erpledger.SubmissionView{}, err
	}
	record, b, err := a.governedDocument(workspace, id, digest)
	if err != nil {
		return fail(commands.EvidenceStale)
	}
	if record.review.State != "confirmed" {
		return fail(commands.PrerequisiteFailed)
	}
	if _, err := a.documentNow(); err != nil {
		return erpledger.SubmissionView{}, err
	}
	subject, err := a.server.currentFinancialChargeLocked(workspace, b.Request, b.RequestDigest, consumer)
	if err != nil {
		return erpledger.SubmissionView{}, err
	}
	if a.server.held() {
		return fail(commands.PrerequisiteFailed)
	}
	candidate, err := a.server.chargeProducer.candidates.financialCandidate(workspace, b.Candidate)
	if err != nil {
		return fail(commands.EvidenceStale)
	}
	sub, total, err := documentSubmission(b, candidate, subject, key)
	if err != nil {
		return fail(commands.EvidenceStale)
	}
	if err = a.currentDocumentERP(b); err != nil {
		return erpledger.SubmissionView{}, err
	}
	ownerKey := [2]string{workspace, key}
	old, owned := a.owners[ownerKey]
	if owned && (old.basis != digest || old.actor != consumer || !reflect.DeepEqual(old.submission, sub) || old.total != total) {
		return fail(commands.EvidenceStale)
	}
	actual, lookup := a.ledger.SubmissionOf(workspace, key)
	if lookup != nil && !errors.Is(lookup, erpledger.ErrNotFound) {
		if owned {
			return erpledger.SubmissionView{}, errFinancialERPUncertain
		}
		return fail(commands.DependencyUnavailable)
	}
	if !owned && lookup == nil {
		return fail(commands.EvidenceStale)
	}
	if owned && lookup == nil {
		return a.reconcile(ownerKey, old, actual)
	}
	if owned && old.receipt != "" {
		return erpledger.SubmissionView{}, errFinancialERPUncertain
	}
	actor := erpledger.Actor{ID: consumer, Kind: erpledger.KindHuman}
	checked, err := a.ledger.CheckPreparation(workspace, sub, actor)
	if err != nil {
		return erpledger.SubmissionView{}, err
	}
	if checked.Total != total {
		return fail(commands.EvidenceStale)
	}
	prospective := financialERPOwnership{basis: digest, actor: consumer, submission: sub, total: total, mappings: append([]string(nil), checked.Mappings...)}
	if err := a.admitPublicPreparation(key, prospective); err != nil {
		return erpledger.SubmissionView{}, err
	}
	if !owned {
		if len(a.owners) >= financialDocumentLimit {
			return fail(commands.PrerequisiteFailed)
		}
		old = financialERPOwnership{basis: digest, actor: consumer, submission: sub, total: total, mappings: append([]string(nil), checked.Mappings...)}
		a.owners[ownerKey] = old
	} else if !reflect.DeepEqual(old.mappings, checked.Mappings) {
		return fail(commands.EvidenceStale)
	}
	actual, err = a.ledger.Prepare(workspace, sub, actor)
	if err != nil {
		return erpledger.SubmissionView{}, errFinancialERPUncertain
	}
	if a.afterPrepare != nil {
		if err = a.afterPrepare(); err != nil {
			return erpledger.SubmissionView{}, errFinancialERPUncertain
		}
	}
	return a.reconcile(ownerKey, old, actual)
}

func (a *financialERP) reconcile(key [2]string, owner financialERPOwnership, v erpledger.SubmissionView) (erpledger.SubmissionView, error) {
	if !matchingPreparation(owner, v) {
		return erpledger.SubmissionView{}, errFinancialERPUncertain
	}
	owner.receipt = v.Digest
	a.owners[key] = owner
	return v, nil
}

func matchingPreparation(owner financialERPOwnership, v erpledger.SubmissionView) bool {
	return reflect.DeepEqual(owner.submission, v.Submission) && owner.actor == v.Author && owner.total == v.Total && reflect.DeepEqual(owner.mappings, v.Mappings) && v.State == erpledger.StatePrepared && len(v.Digest) == 64 && (owner.receipt == "" || owner.receipt == v.Digest) && len(v.Commands) == 0 && v.External == "" && !v.Diverged
}
