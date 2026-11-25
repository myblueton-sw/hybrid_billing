package server

import (
	"errors"
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

type approvalPrincipal struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type approvalTarget struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Entity   *string `json:"entity"`
	Customer *string `json:"customer"`
	Period   *string `json:"period"`
}
type approvalAmount struct {
	Currency   string  `json:"currency"`
	Basis      string  `json:"basis"`
	Base       *string `json:"base"`
	Candidate  *string `json:"candidate"`
	Difference *string `json:"difference"`
	ERP        *string `json:"erp"`
}
type approvalERP struct {
	Applicability string  `json:"applicability"`
	Revision      *string `json:"external_revision"`
	Observed      *string `json:"observed_at"`
}
type approvalBlocker struct {
	Code string `json:"code"`
	Text string `json:"text"`
}
type approvalEvidence struct {
	Input        *string           `json:"input_digest"`
	Result       *string           `json:"result_digest"`
	Revision     *string           `json:"revision"`
	Dependencies string            `json:"dependencies"`
	Completeness string            `json:"completeness"`
	Validation   string            `json:"validation"`
	Blockers     []approvalBlocker `json:"blockers"`
}
type approvalAuthority struct {
	Separation string  `json:"separation"`
	Limit      string  `json:"limit"`
	Until      *string `json:"valid_until"`
}
type approvalRow struct {
	ID        string             `json:"id"`
	Kind      string             `json:"kind"`
	Status    string             `json:"status"`
	Target    approvalTarget     `json:"target"`
	Author    *approvalPrincipal `json:"author"`
	Requester *approvalPrincipal `json:"requester"`
	Policy    int                `json:"policy_revision"`
	Approver  *approvalPrincipal `json:"approver"`
	Updated   string             `json:"updated_at"`
	Deadline  *string            `json:"deadline"`
	Decided   *string            `json:"decided_at"`
	Amounts   []approvalAmount   `json:"amounts"`
	ERP       approvalERP        `json:"erp"`
	Evidence  approvalEvidence   `json:"evidence"`
	Authority approvalAuthority  `json:"authority"`
}

var errApprovalProjection = errors.New("approval evidence cannot be represented")

func approvalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func approvalPerson(id string) *approvalPrincipal { return &approvalPrincipal{ID: id, Label: id} }

func (s *Server) approvalRows(ws string) ([]approvalRow, error) {
	if s.faulty("approvals") || len(s.approvalIndex) > approvalCap || len(s.financialCharges) > approvalCap || s.financialApprovalCount() > approvalCap {
		return nil, errApprovalProjection
	}
	rows := []approvalRow{}
	seen := map[string]bool{}
	for _, index := range s.approvalIndex {
		if index.Workspace != ws {
			continue
		}
		if seen[index.ID] {
			return nil, errApprovalProjection
		}
		seen[index.ID] = true
		v, err := s.approvals.RequestOf(ws, index.ID)
		if err != nil {
			return nil, err
		}
		row, err := s.projectApprovalRow(ws, index, v, false)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for id, binding := range s.financialCharges {
		if binding.Workspace != ws {
			continue
		}
		if seen[id] {
			return nil, errApprovalProjection
		}
		seen[id] = true
		v, _, err := s.financialChargeRecord(ws, id)
		if err != nil {
			return nil, err
		}
		row, err := s.projectApprovalRow(ws, approvalIndex{Workspace: ws, ID: id, Target: v.Subject.Target, Basis: "rated_candidate_total"}, v, true)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	for id, binding := range s.financialFinals {
		if binding.Workspace != ws {
			continue
		}
		if seen[id] {
			return nil, errApprovalProjection
		}
		seen[id] = true
		v, b, err := s.financialFinalRecord(ws, id)
		if err != nil {
			return nil, err
		}
		for _, decision := range v.Decisions {
			if financialFinalContributorCheck(b.Candidate, b.Requester, decision.Actor) != nil {
				return nil, errApprovalProjection
			}
		}
		row, err := s.projectApprovalRow(ws, approvalIndex{Workspace: ws, ID: id, Target: v.Subject.Target, Basis: financialFinalAmountProfile}, v, true)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

func (s *Server) projectApprovalRow(ws string, index approvalIndex, v approvals.RequestView, governed bool) (approvalRow, error) {
	p, err := s.approvals.PolicyAt(ws, v.Subject.Stage, v.Policy)
	if err != nil || p.Revision != v.Policy || p.Approvals != 1 || v.Subject.Target != index.Target || (!governed && index.Basis != "gross") || (governed && !((index.Basis == "rated_candidate_total" && v.Subject.Stage == approvals.StageCharge) || (index.Basis == financialFinalAmountProfile && v.Subject.Stage == approvals.StageERPFinal))) || len(v.Subject.Entries) != 0 || len(v.Decisions) > 1 {
		return approvalRow{}, errApprovalProjection
	}
	kind := ""
	switch v.Subject.Stage {
	case approvals.StageCharge:
		kind = "billing_result"
	case approvals.StageERPFinal:
		kind = "final_erp_amount"
	default:
		return approvalRow{}, errApprovalProjection
	}
	row := approvalRow{ID: index.ID, Kind: kind, Status: v.State, Target: approvalTarget{ID: index.Target, Label: index.Target}, Author: approvalPerson(v.Subject.Author), Requester: approvalPerson(v.Requester), Policy: v.Policy, Updated: v.Written.UTC().Format(time.RFC3339Nano), Amounts: []approvalAmount{{Currency: v.Subject.Currency, Basis: index.Basis, Candidate: approvalString(v.Subject.Amount)}}, ERP: approvalERP{Applicability: "unknown"}, Evidence: approvalEvidence{Input: approvalString(v.Subject.Digests["input"]), Result: approvalString(v.Subject.Digests["result"]), Revision: approvalString(v.Digest), Dependencies: "unknown", Completeness: "unknown", Validation: "unknown", Blockers: []approvalBlocker{{Code: "producer_unbound", Text: "Source validation and stage prerequisites are not connected."}}}, Authority: approvalAuthority{Separation: "unknown", Limit: "unknown"}}
	if governed {
		row.Evidence.Blockers = []approvalBlocker{{Code: "current_checks_unverified", Text: "Current prerequisites and permission were not rechecked for this historical read."}}
	}
	switch v.State {
	case approvals.StatePending:
		if len(v.Decisions) != 0 {
			return approvalRow{}, errApprovalProjection
		}
		if !governed {
			row.Status = "blocked"
		}
	case approvals.StateSuperseded:
		row.Updated = v.Ended.UTC().Format(time.RFC3339Nano)
	case approvals.StateApproved, approvals.StateRejected:
		if len(v.Decisions) != 1 {
			return approvalRow{}, errApprovalProjection
		}
	default:
		return approvalRow{}, errApprovalProjection
	}
	if len(v.Decisions) == 1 {
		d := v.Decisions[0]
		expected := approvals.VerdictApprove
		if v.State == approvals.StateRejected {
			expected = approvals.VerdictReject
		}
		if len(d.Decision.Exclude) != 0 || d.Actor == v.Subject.Author || d.Actor == v.Requester || d.Decision.Verdict != expected {
			return approvalRow{}, errApprovalProjection
		}
		row.Approver = approvalPerson(d.Actor)
		row.Decided = approvalString(d.At.UTC().Format(time.RFC3339Nano))
		if v.State != approvals.StateSuperseded {
			row.Updated = *row.Decided
		}
	}
	return row, nil
}

func (s *Server) financialApprovalCount() int {
	return len(s.approvalIndex) + len(s.financialCharges) + len(s.financialFinals)
}
