package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const approvalKind = "approval_review"
const approvalField = "approval_evidence"
const approvalCap = 1000

type approvalIndex struct {
	Workspace string
	ID        string
	Target    string
	Basis     string
}

func (s *Server) initApprovals() error {
	profile, err := decimal.NewProfile(decimal.ProfileSpec{ID: "prelaunch-approval", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		return err
	}
	s.approvalKey = random(32)
	s.financialCharges = make(map[string]financialChargeBinding)
	s.financialPolicies = make(map[string]financialPolicyBinding)
	s.erpFinalPolicies = make(map[string]financialPolicyBinding)
	s.financialAuthorities = make(map[string]financialAuthorityBinding)
	s.approvals, err = approvals.NewRegistry(profile, approvals.Limits{Workspaces: 100, Admins: 4, Proposals: 100, Authorities: 100, Requests: approvalCap, Items: 100, Executions: 1000}, s.clock)
	return err
}

func approvalHash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func (s *Server) seedApprovals(now time.Time) error {
	ws := s.Workspace()
	terms := gate.Terms{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: ws, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	validity := gate.Validity{NotBefore: now, NotAfter: now.Add(readerValidity)}
	if err := s.admin.AddGrant(gate.Grant{ID: "approval-console", ClientID: ConsoleClient, Terms: terms, Validity: validity}); err != nil {
		return err
	}
	for _, user := range []string{SampleAdmin, SampleAdmin2, SampleAdmin3} {
		if err := s.admin.AddGrant(gate.Grant{ID: "approval-" + user, SubjectID: user, Terms: terms, Validity: validity}); err != nil {
			return err
		}
	}
	human := func(id string) approvals.Actor { return approvals.Actor{ID: id, Kind: approvals.KindHuman} }
	if _, err := s.approvals.Open(ws, []string{SampleAdmin, SampleAdmin2}, human(SampleAdmin)); err != nil {
		return err
	}
	for _, stage := range []string{approvals.StageCharge, approvals.StageERPFinal} {
		p := approvals.Proposal{ID: "sample-" + stage, Kind: approvals.ProposePolicy, Policy: approvals.Policy{Stage: stage, Revision: 1, Approvals: 1, Required: []string{"input", "result"}, Validity: 24 * time.Hour}}
		if _, err := s.approvals.Propose(ws, p, human(SampleAdmin)); err != nil {
			return err
		}
		if _, err := s.approvals.Confirm(ws, p.ID, human(SampleAdmin2)); err != nil {
			return err
		}
	}
	for i := 1; i <= 30; i++ {
		id := fmt.Sprintf("sample-approval-%03d", i)
		stage := approvals.StageCharge
		if i%2 == 0 {
			stage = approvals.StageERPFinal
		}
		subject := approvals.Subject{ID: id, Stage: stage, Target: fmt.Sprintf("sample-result-%03d", i), Author: SampleAdmin, Amount: "1234.56789012", Currency: "USD", Digests: map[string]string{"input": approvalHash(id + ":input"), "result": approvalHash(id + ":result")}}
		if _, err := s.approvals.Request(ws, subject, human(SampleAdmin)); err != nil {
			return err
		}
		if i == 30 {
			if _, err := s.approvals.Decide(ws, id, approvals.Decision{Verdict: approvals.VerdictReject, Reason: "missing_evidence"}, human(SampleAdmin2)); err != nil {
				return err
			}
		}
		s.approvalIndex = append(s.approvalIndex, approvalIndex{Workspace: ws, ID: id, Target: subject.Target, Basis: "gross"})
	}
	return nil
}
