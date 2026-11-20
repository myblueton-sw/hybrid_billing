package server

import (
	"errors"
	"net/http"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type candidateSummary struct {
	ID        string `json:"id"`
	Digest    string `json:"digest"`
	Customer  string `json:"customer"`
	Entity    string `json:"legal_entity"`
	From      string `json:"from"`
	To        string `json:"to"`
	Total     string `json:"rating_total"`
	Currency  string `json:"currency"`
	CostBasis string `json:"cost_basis"`
	Lines     int    `json:"lines"`
	Excluded  int    `json:"excluded"`
	Held      bool   `json:"has_held_rule"`
}

type candidateLine struct {
	ID       string `json:"id"`
	Class    string `json:"class"`
	SKU      string `json:"sku"`
	Service  string `json:"service"`
	Group    string `json:"group"`
	Cost     string `json:"cost"`
	Quantity string `json:"quantity"`
	Currency string `json:"currency"`
}

type candidateCharge struct {
	Rule   string `json:"rule"`
	Kind   string `json:"kind"`
	Line   string `json:"line"`
	Amount string `json:"amount"`
}

type candidateExclusion struct {
	Line   string `json:"line"`
	Reason string `json:"reason"`
}

type candidateStep struct {
	Name      string `json:"name"`
	Basis     string `json:"basis"`
	Parameter string `json:"parameter"`
	Outcome   string `json:"outcome"`
}

type candidateTrace struct {
	Rule  string          `json:"rule"`
	Total string          `json:"total"`
	Steps []candidateStep `json:"steps"`
}

type candidateOriginRow struct {
	Kind     string `json:"kind"`
	Document string `json:"document"`
	Key      string `json:"key"`
	Chunk    int    `json:"chunk"`
	Position int    `json:"position"`
	Digest   string `json:"digest"`
	Line     string `json:"line"`
}

type candidateProvenance struct {
	InputEligibility *candidateRecordedInput `json:"recorded_input,omitempty"`
	Source           candidateSource         `json:"source_manifest"`
	Lifecycle        candidateLifecycle      `json:"recorded_lifecycle"`
	Run              string                  `json:"normalization_run"`
	Provider         string                  `json:"provider"`
	SourceVersion    string                  `json:"source_version"`
	Target           string                  `json:"target"`
	MappingRevision  uint64                  `json:"mapping_revision"`
	Manifest         uint64                  `json:"manifest_reference"`
	Profile          string                  `json:"profile"`
	ProfileRevision  string                  `json:"profile_revision"`
	MappingDigest    string                  `json:"mapping_digest"`
	TargetDigest     string                  `json:"target_digest"`
	Rows             []candidateOriginRow    `json:"rows"`
}

type candidateRecordedInput struct {
	Review           string `json:"review"`
	Digest           string `json:"digest"`
	ScopeRevision    uint64 `json:"scope_revision"`
	SourceRun        string `json:"source_run"`
	Document         string `json:"document"`
	DocumentRevision int    `json:"document_revision"`
	DocumentDigest   string `json:"document_digest"`
	ConfirmedBy      string `json:"confirmed_by"`
	ConfirmedAt      string `json:"confirmed_at"`
}

type candidateDetail struct {
	candidateSummary
	InputLines []candidateLine      `json:"input_lines"`
	Charges    []candidateCharge    `json:"charges"`
	Exclusions []candidateExclusion `json:"exclusions"`
	Traces     []candidateTrace     `json:"traces"`
	Provenance candidateProvenance  `json:"provenance"`
}

type candidateResponse struct {
	Schema            int                 `json:"schema"`
	ReadOnly          bool                `json:"read_only"`
	SourceMode        string              `json:"source_mode"`
	AmountBasis       string              `json:"amount_basis"`
	ApprovalReadiness string              `json:"approval_readiness"`
	Commands          string              `json:"commands"`
	Count             int                 `json:"count"`
	Records           *[]candidateSummary `json:"records,omitempty"`
	Record            *candidateDetail    `json:"record,omitempty"`
}

func summarizeCandidate(v chargeCandidate, workspace, id string) (candidateSummary, error) {
	in, result := v.Envelope.Input, v.Envelope.Result
	if v.Envelope.Schema != "hb_charge_candidate_v5" && v.Envelope.Schema != "hb_charge_candidate_v6" ||
		(v.Envelope.Schema == "hb_charge_candidate_v6") != (in.Normalized.InputEligibility != nil) || !validCandidateInput(in) || in.SourceMode != "synthetic" || in.Workspace != workspace || in.ID != id {
		return candidateSummary{}, errCandidateInvalid
	}
	return candidateSummary{ID: in.ID, Digest: v.Digest, Customer: in.Customer, Entity: in.Entity, From: in.From, To: in.To, Total: result.Total, Currency: result.Currency,
		CostBasis: string(result.CostBasis), Lines: result.Lines, Excluded: len(result.Excluded), Held: !result.NoRuleHeld}, nil
}

func (s *chargeCandidates) summaries(workspace string) ([]candidateSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := []string{}
	for key := range s.records {
		if key[0] == workspace {
			ids = append(ids, key[1])
		}
	}
	sort.Strings(ids)
	out := make([]candidateSummary, 0, len(ids))
	for _, id := range ids {
		v, err := decodeCandidate(s.records[[2]string{workspace, id}])
		if err != nil {
			return nil, err
		}
		row, err := summarizeCandidate(v, workspace, id)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func detailCandidate(v chargeCandidate, summary candidateSummary) candidateDetail {
	out := candidateDetail{candidateSummary: summary, InputLines: []candidateLine{}, Charges: []candidateCharge{}, Exclusions: []candidateExclusion{}, Traces: []candidateTrace{}}
	for _, line := range v.Envelope.Input.Lines {
		out.InputLines = append(out.InputLines, candidateLine{line.ID, string(line.Class), line.SKU, line.Service, line.Group, line.Cost, line.Quantity, line.Currency})
	}
	for _, charge := range v.Envelope.Result.Charges {
		out.Charges = append(out.Charges, candidateCharge{charge.Rule, charge.Kind, charge.Line, charge.Amount})
	}
	for _, excluded := range v.Envelope.Result.Excluded {
		out.Exclusions = append(out.Exclusions, candidateExclusion{excluded.Line, excluded.Reason})
	}
	for _, trace := range v.Envelope.Result.Traces {
		row := candidateTrace{Rule: trace.Rule, Total: trace.Total, Steps: []candidateStep{}}
		for _, step := range trace.Steps {
			row.Steps = append(row.Steps, candidateStep{step.Name, step.Basis, step.Parameter, step.Outcome})
		}
		out.Traces = append(out.Traces, row)
	}
	origin := v.Envelope.Input.Normalized
	out.Provenance = candidateProvenance{Source: origin.Source, Lifecycle: origin.Lifecycle, Run: origin.Run, Provider: origin.Mapping.Provider, SourceVersion: origin.Mapping.SourceVersion, Target: origin.Mapping.Target,
		MappingRevision: origin.MappingRevision, Manifest: origin.Manifest, Profile: origin.Profile, ProfileRevision: origin.ProfileRevision, MappingDigest: origin.MappingDigest, TargetDigest: origin.TargetDigest, Rows: []candidateOriginRow{}}
	if input := origin.InputEligibility; input != nil {
		out.Provenance.InputEligibility = &candidateRecordedInput{Review: input.Review.ID, Digest: input.Review.Evidence.Digest, ScopeRevision: input.Publication.Revision, SourceRun: input.SourceRun, Document: input.Document, DocumentRevision: input.DocumentRevision, DocumentDigest: input.DocumentDigest, ConfirmedBy: input.Review.Decision.Actor, ConfirmedAt: stamp(input.Review.Decision.At)}
	}
	for _, row := range origin.Rows {
		out.Provenance.Rows = append(out.Provenance.Rows, candidateOriginRow{string(row.Key.Kind), row.Key.Document, row.Key.Key, row.Chunk, row.Position, row.Digest, row.Line})
	}
	return out
}

func (s *Server) readChargeCandidates(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	if !sameToken(r.Header.Get(csrfHeader), found.csrf) {
		fail(w, http.StatusForbidden, CodeCSRF, request)
		return
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return
	}
	decision := s.gate.Decide(answer, ConsoleClient, gate.Request{Action: gate.ViewCost, Kind: candidateKind, Fields: []string{candidateField}, Resource: gate.Resource{WorkspaceID: answer.Workspace()}})
	if !decision.Allowed {
		code := commands.PermissionDenied
		if decision.Code == gate.AuthorityUnavailable {
			code = commands.AuthorityUnavailable
		}
		failCode(w, code, request)
		return
	}
	if r.URL.RawQuery != "" {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	out := candidateResponse{Schema: 1, ReadOnly: true, SourceMode: "synthetic", AmountBasis: "rating_total", ApprovalReadiness: "unverified", Commands: "unconnected"}
	id := r.PathValue("id")
	if id == "" {
		rows, err := s.chargeProducer.candidates.summaries(answer.Workspace())
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		out.Count, out.Records = len(rows), &rows
	} else {
		if !validRequestID(id) {
			failCode(w, commands.InvalidRequest, request)
			return
		}
		v, err := s.chargeProducer.candidates.read(answer.Workspace(), id)
		if errors.Is(err, errCandidateMissing) {
			failCode(w, commands.UnsupportedCommand, request)
			return
		}
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		summary, err := summarizeCandidate(v, answer.Workspace(), id)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		detail := detailCandidate(v, summary)
		out.Count, out.Record = 1, &detail
	}
	writeJSON(w, http.StatusOK, request, out)
}

func candidateReadPath(detail bool) map[string]any {
	parameters := []any{}
	operation := "charge.candidates.list"
	if detail {
		operation = "charge.candidates.detail"
		parameters = append(parameters, map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "pattern": "^[A-Za-z0-9._:-]{1,128}$"}})
	}
	return map[string]any{"get": map[string]any{"operationId": operation, "x-action": string(gate.ViewCost), "x-field": candidateField, "parameters": parameters, "x-session": true, "x-csrf": true,
		"x-scope": "whole authenticated workspace", "x-query": "all query parameters refused", "x-complete-list-bound": 1000,
		"responses": map[string]any{
			"200": map[string]any{"description": "Complete sorted candidate summaries, or one allowlisted detail. Exact decimal strings; synthetic rating totals, unverified approval readiness. The commands=unconnected marker describes downstream financial actions, not the separate pricing_policy.capture command. Optional provenance.recorded_input contains historical review/digest/scope_revision, source_run, document identity/revision/digest and confirmation actor/time; it does not assert current input eligibility. Detail provenance.recorded_lifecycle contains contract, journal_sequence, from, to_exclusive and active segments with contract_revision. Detail provenance.source_manifest contains the recorded source subject, manifest, content and attribution digests, source amount/currency fields, admission observations and per-row identity/artifact/digest/amount/currency. Both describe evidence recorded at capture, not current source or lifecycle validity, settlement, ownership or financial eligibility. No raw source cells or complete internal envelope."},
			"401": readCodes(readError(commands.Unauthenticated), string(commands.Unauthenticated)),
			"403": readCodes(readError(commands.PermissionDenied), CodeCSRF, string(commands.PermissionDenied)),
			"404": readCodes(readError(commands.UnsupportedCommand), string(commands.UnsupportedCommand)),
			"422": readCodes(readError(commands.InvalidRequest), string(commands.InvalidRequest)),
			"503": readCodes(readError(commands.DependencyUnavailable), string(commands.DependencyUnavailable), string(commands.AuthorityUnavailable)),
		}}}
}
