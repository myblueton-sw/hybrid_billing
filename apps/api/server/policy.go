package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	policyTarget   = "policy"
	proposalPrefix = "proposal-"
	maxRecords     = 200
	appliedShown   = 10
	minPolicyAge   = 5
	maxPolicyAge   = 720
)

type record struct {
	number uint64
	reason string
}

type stepRow struct {
	Action        string `json:"action"`
	MaxAgeSeconds int64  `json:"max_age_seconds"`
	MFA           bool   `json:"mfa"`
}

type revisionRow struct {
	Methods       []string  `json:"methods"`
	MFA           string    `json:"mfa"`
	MaxAgeSeconds int64     `json:"max_age_seconds"`
	StepUp        []stepRow `json:"step_up"`
}

type policyRow struct {
	revisionRow
	Number   uint64 `json:"number"`
	Proposer string `json:"proposer"`
	Approver string `json:"approver"`
	At       string `json:"at"`
}

type proposalRow struct {
	Number      uint64      `json:"number"`
	Target      string      `json:"target"`
	State       string      `json:"state"`
	Proposer    string      `json:"proposer"`
	Approver    string      `json:"approver"`
	Reason      string      `json:"reason"`
	Revision    revisionRow `json:"revision"`
	CanApprove  bool        `json:"can_approve"`
	CanWithdraw bool        `json:"can_withdraw"`
	ActUntil    string      `json:"act_until"`
}

type policyPage struct {
	Revision   string        `json:"revision"`
	Policy     *policyRow    `json:"policy"`
	Proposals  []proposalRow `json:"proposals"`
	Now        string        `json:"now"`
	SignedInAt string        `json:"signed_in_at"`
	ActUntil   string        `json:"act_until"`
	You        string        `json:"you"`
}

func revisionOf(r authpolicy.Revision) revisionRow {
	out := revisionRow{Methods: append([]string{}, r.Methods...), MFA: r.MFA, MaxAgeSeconds: int64(r.MaxAge / time.Second), StepUp: []stepRow{}}
	for _, rule := range r.StepUp {
		out.StepUp = append(out.StepUp, stepRow{Action: rule.Action, MaxAgeSeconds: int64(rule.MaxAge / time.Second), MFA: rule.MFA})
	}
	return out
}

func window(r authpolicy.Revision) time.Duration {
	limit := r.MaxAge
	for _, rule := range r.StepUp {
		if rule.Action == authpolicy.ActionPolicyChange && rule.MaxAge < limit {
			limit = rule.MaxAge
		}
	}
	return limit
}

func (s *Server) listPolicy(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, request, s.policyPage(answer.User(), found.created))
}

func (s *Server) policyPage(you string, signed time.Time) policyPage {
	page := policyPage{Revision: s.Revision(), Proposals: []proposalRow{}, Now: stamp(s.clock()), SignedInAt: stamp(signed), You: you}
	limit := time.Duration(-1)
	if current, err := s.policy.Policy(s.bootWorkspace); err == nil {
		page.Policy = &policyRow{revisionRow: revisionOf(current.Revision), Number: current.Number, Proposer: current.Proposer, Approver: current.Approver, At: stamp(current.At)}
		limit = window(current.Revision)
		page.ActUntil = stamp(signed.Add(limit))
	}
	applied := 0
	for i := len(s.records) - 1; i >= 0; i-- {
		rec := s.records[i]
		view, err := s.policy.Proposal(rec.number)
		if err != nil || (view.State == authpolicy.ProposalApplied && applied >= appliedShown) {
			continue
		}
		row := proposalRow{Number: view.Number, Target: proposalPrefix + strconv.FormatUint(view.Number, 10), State: view.State, Proposer: view.Proposer,
			Approver: view.Approver, Reason: rec.reason, Revision: revisionOf(view.Change.Revision)}
		if view.State == authpolicy.ProposalApplied {
			applied++
		} else {
			row.CanApprove, row.CanWithdraw = view.Proposer != you, view.Proposer == you
			act := window(view.Change.Revision)
			if limit >= 0 && limit < act {
				act = limit
			}
			row.ActUntil = stamp(signed.Add(act))
		}
		page.Proposals = append(page.Proposals, row)
	}
	return page
}

func (s *Server) recordProposal(number uint64, reason string) {
	for _, rec := range s.records {
		if rec.number == number {
			return
		}
	}
	s.records = append(s.records, record{number: number, reason: reason})
}

func (s *Server) pruneRecords() {
	kept := s.records[:0]
	for _, rec := range s.records {
		if _, err := s.policy.Proposal(rec.number); err == nil {
			kept = append(kept, rec)
		}
	}
	s.records = kept
	if len(s.records) > maxRecords {
		s.records = append([]record(nil), s.records[len(s.records)-maxRecords:]...)
	}
}

func minutes(raw string, low, high int) (int, bool) {
	if len(raw) == 0 || len(raw) > 4 || (len(raw) > 1 && raw[0] == '0') || strings.TrimLeft(raw, "0123456789") != "" {
		return 0, false
	}
	n, _ := strconv.Atoi(raw)
	return n, n >= low && n <= high
}

func proposalNumber(target string) uint64 {
	raw, ok := strings.CutPrefix(target, proposalPrefix)
	if !ok || raw == "" || strings.TrimLeft(raw, "0123456789") != "" || raw[0] == '0' {
		return 0
	}
	n, _ := strconv.ParseUint(raw, 10, 64)
	return n
}

func mapPolicy(err error) error {
	var refusal authpolicy.Refusal
	switch {
	case errors.As(err, &refusal) && refusal.Reason == authpolicy.ReasonNotPending, errors.Is(err, authpolicy.ErrNotFound):
		return commands.Fail(commands.InvalidStage)
	case errors.Is(err, authpolicy.ErrRejected):
		return commands.Fail(commands.PrerequisiteFailed)
	case errors.Is(err, authpolicy.ErrLimit):
		return commands.Fail(commands.DependencyUnavailable)
	}
	return commands.Fail(commands.InvalidRequest)
}

type policyCommand struct {
	s    *Server
	kind string
}

func (h policyCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	if h.kind == "propose" {
		if target != policyTarget {
			return commands.Target{}, commands.NotFound
		}
		return commands.Target{Revision: s.Revision()}, commands.Found
	}
	n := proposalNumber(target)
	if !s.recorded(n) {
		return commands.Target{}, commands.NotFound
	}
	if _, err := s.policy.Proposal(n); err != nil {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (s *Server) recorded(n uint64) bool {
	for _, rec := range s.records {
		if rec.number == n {
			return true
		}
	}
	return false
}

func (s *Server) actorOf() (authpolicy.Principal, authpolicy.Authentication, bool) {
	found, ok := s.sessions.get(s.caller)
	if !ok {
		return authpolicy.Principal{}, authpolicy.Authentication{}, false
	}
	answer, err := s.identity.Boundary().Access(s.caller, s.bootWorkspace)
	if err != nil {
		return authpolicy.Principal{}, authpolicy.Authentication{}, false
	}
	role := authpolicy.RoleMember
	if answer.Role() == identity.Administrator {
		role = authpolicy.RoleAdministrator
	}
	return authpolicy.Principal{User: answer.User(), Workspace: answer.Workspace(), Role: role, Kind: authpolicy.KindHuman},
		authpolicy.Authentication{Method: authpolicy.MethodPassword, At: found.created}, true
}

func (h policyCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	s.pruneRecords()
	p, a, ok := s.actorOf()
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
	}
	switch h.kind {
	case "propose":
		maxAge, ok := minutes(ctx.Fields["max_age_minutes"], minPolicyAge, maxPolicyAge)
		if !ok {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "max_age_minutes")
		}
		step, ok := minutes(ctx.Fields["step_up_minutes"], 0, maxAge)
		if !ok || (step != 0 && step < minPolicyAge) {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "step_up_minutes")
		}
		rev := authpolicy.Revision{Workspace: s.bootWorkspace, Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAge: time.Duration(maxAge) * time.Minute}
		if step > 0 {
			rev.StepUp = []authpolicy.StepUp{{Action: authpolicy.ActionPolicyChange, MaxAge: time.Duration(step) * time.Minute}}
		}
		view, err := s.policy.Propose(authpolicy.Change{Kind: authpolicy.ChangeRevision, Revision: rev}, p, a)
		if err != nil {
			return commands.Outcome{}, mapPolicy(err)
		}
		if s.recorded(view.Number) {
			return commands.Outcome{}, commands.Fail(commands.InvalidStage)
		}
		s.recordProposal(view.Number, ctx.Rationale)
		return commands.Outcome{ResultRevision: s.Next(), BusinessState: "proposed", Fields: map[string]string{"proposal": proposalPrefix + strconv.FormatUint(view.Number, 10)}}, nil
	case "approve":
		n := proposalNumber(ctx.Target)
		if view, _ := s.policy.Proposal(n); view.State != authpolicy.ProposalPending {
			return commands.Outcome{}, commands.Fail(commands.InvalidStage)
		}
		if _, err := s.policy.Approve(n, p, a); err != nil {
			return commands.Outcome{}, mapPolicy(err)
		}
		return commands.Outcome{ResultRevision: s.Next(), BusinessState: "applied"}, nil
	}
	n := proposalNumber(ctx.Target)
	if _, err := s.policy.Withdraw(n, p, a); err != nil {
		return commands.Outcome{}, mapPolicy(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "withdrawn"}, nil
}

func (s *Server) policyDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	ages := []commands.Field{{Name: "max_age_minutes", Type: commands.Integer, Required: true}, {Name: "step_up_minutes", Type: commands.Integer, Required: true}}
	return []commands.Definition{
		{ID: "policy.propose", Permission: permission, Mutating: true, Rationale: true, Fields: ages, Handler: policyCommand{s: s, kind: "propose"}},
		{ID: "policy.approve", Permission: permission, Mutating: true, Rationale: true, Handler: policyCommand{s: s, kind: "approve"}},
		{ID: "policy.withdraw", Permission: permission, Mutating: true, Rationale: true, Handler: policyCommand{s: s, kind: "withdraw"}},
	}
}
