package server

import (
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

const (
	trustTarget    = "trust"
	trustJobPrefix = "trust-activation-"
	reasonPublish  = "publish_failed"
)

var trustRefusals = []commands.Code{commands.SelfApprovalDenied, commands.AuthorityEpochConflict, commands.ApprovalStale, commands.EvidenceStale,
	commands.IndependentApprovalRequired}

type trustCommand struct {
	s    *Server
	kind string
}

func (h trustCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace || s.trust == nil {
		return commands.Target{}, commands.NotFound
	}
	if h.kind == "propose" && target == trustTarget {
		return commands.Target{Revision: s.Revision()}, commands.Found
	}
	if _, ok := s.trust.Get(target, workspace); !ok {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func mapTrust(err error) error {
	var refusal trustchange.Refusal
	if !errors.As(err, &refusal) {
		return err
	}
	switch refusal.Reason {
	case trustchange.ReasonStage, trustchange.ReasonDuplicate:
		return commands.Fail(commands.InvalidStage)
	case trustchange.ReasonSelf:
		return commands.Fail(commands.SelfApprovalDenied)
	case trustchange.ReasonEpoch:
		return commands.Fail(commands.AuthorityEpochConflict)
	case trustchange.ReasonApprovalStale:
		return commands.Fail(commands.ApprovalStale)
	case trustchange.ReasonEvidenceStale:
		return commands.Fail(commands.EvidenceStale)
	case trustchange.ReasonNeedsApproval:
		return commands.Fail(commands.IndependentApprovalRequired)
	case trustchange.ReasonLimit:
		return commands.Fail(commands.DependencyUnavailable)
	case trustchange.ReasonUnchanged, trustchange.ReasonInvalid:
		return commands.Fail(commands.InvalidRequest, "max_age_minutes")
	}
	return commands.Fail(commands.PermissionDenied)
}

func number(fields map[string]string, name string) (uint64, error) {
	raw := fields[name]
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != raw {
		return 0, commands.Fail(commands.InvalidRequest, name)
	}
	return n, nil
}

func (s *Server) trustUnavailable(activate bool) bool {
	if s.held() || s.ledger == nil {
		return true
	}
	switch s.ledger.Status().State {
	case decisionledger.StateOK, decisionledger.StateRecovered:
		return false
	case decisionledger.StateFull:
		return activate
	}
	return true
}

func (s *Server) principalOf(user string) (authpolicy.Principal, time.Time, bool) {
	found, ok := s.sessions.get(s.caller)
	answer, err := s.identity.Boundary().Access(s.caller, s.bootWorkspace)
	if !ok || err != nil || answer.User() != user {
		return authpolicy.Principal{}, time.Time{}, false
	}
	role := authpolicy.RoleMember
	if answer.Role() == identity.Administrator {
		role = authpolicy.RoleAdministrator
	}
	return authpolicy.Principal{User: user, Workspace: answer.Workspace(), Role: role, Kind: authpolicy.KindHuman}, found.authenticated, true
}

func proposedCandidate(fields map[string]string) (trustchange.Candidate, error) {
	maxAge, ok := minutes(fields["max_age_minutes"], minPolicyAge, maxPolicyAge)
	if !ok {
		return trustchange.Candidate{}, commands.Fail(commands.InvalidRequest, "max_age_minutes")
	}
	step, ok := minutes(fields["step_up_minutes"], 0, maxAge)
	if !ok || (step != 0 && step < minPolicyAge) {
		return trustchange.Candidate{}, commands.Fail(commands.InvalidRequest, "step_up_minutes")
	}
	c := trustchange.Candidate{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAge: time.Duration(maxAge) * time.Minute}
	if step > 0 {
		c.StepUp = []trustchange.Rule{{Action: authpolicy.ActionPolicyChange, MaxAge: time.Duration(step) * time.Minute}}
	}
	return c, nil
}

func expectEpoch(fields map[string]string, base trustchange.Base, policy bool) error {
	epoch, err := number(fields, "expected_epoch")
	if err != nil {
		return err
	}
	if epoch != base.Epoch {
		return commands.Fail(commands.AuthorityEpochConflict)
	}
	if !policy {
		return nil
	}
	n, err := number(fields, "expected_policy")
	if err != nil {
		return err
	}
	if n != base.Policy {
		return commands.Fail(commands.AuthorityEpochConflict)
	}
	return nil
}

func trustOutcome(v trustchange.View, state string) commands.Outcome {
	return commands.Outcome{BusinessState: state, Fields: map[string]string{"change": v.ID, "change_revision": strconv.FormatUint(v.Revision, 10),
		"change_digest": v.ChangeDigest}}
}

func (h trustCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	if s.trustUnavailable(h.kind == "activate") {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	base, current := s.trustBase(ctx.Workspace)
	out, err := h.apply(ctx, base, current)
	if err != nil {
		return commands.Outcome{}, mapTrust(err)
	}
	if !out.Accepted {
		out.ResultRevision = s.Next()
	}
	return out, nil
}

func (h trustCommand) apply(ctx commands.Context, base trustchange.Base, current authpolicy.Revision) (commands.Outcome, error) {
	s := h.s
	switch h.kind {
	case "propose":
		return h.propose(ctx, base)
	case "validate":
		return h.validate(ctx, base, current)
	case "approve", "activate":
		return h.decide(ctx, base)
	}
	revision, err := number(ctx.Fields, "expected_change_revision")
	if err != nil {
		return commands.Outcome{}, err
	}
	end, state := s.trust.Reject, trustchange.StateRejected
	if h.kind == "cancel" {
		end, state = s.trust.Cancel, trustchange.StateCancelled
	}
	v, err := end(ctx.Target, ctx.Workspace, ctx.User, revision, ctx.Rationale)
	if err != nil {
		return commands.Outcome{}, err
	}
	return trustOutcome(v, state), nil
}

func (h trustCommand) propose(ctx commands.Context, base trustchange.Base) (commands.Outcome, error) {
	s := h.s
	c, err := proposedCandidate(ctx.Fields)
	if err != nil {
		return commands.Outcome{}, err
	}
	if err := expectEpoch(ctx.Fields, base, true); err != nil {
		return commands.Outcome{}, err
	}
	var v trustchange.View
	if ctx.Target == trustTarget {
		v, err = s.trust.Propose(trustchange.Proposal{Workspace: ctx.Workspace, Route: trustchange.RouteLocalPassword, Proposer: ctx.User, Candidate: c, Base: base,
			Rationale: ctx.Rationale})
	} else {
		revision, rerr := number(ctx.Fields, "expected_change_revision")
		if rerr != nil {
			return commands.Outcome{}, rerr
		}
		v, err = s.trust.Revise(ctx.Target, ctx.Workspace, ctx.User, revision, c, base, ctx.Rationale)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	return trustOutcome(v, trustchange.StateDraft), nil
}

func (h trustCommand) validate(ctx commands.Context, base trustchange.Base, current authpolicy.Revision) (commands.Outcome, error) {
	s := h.s
	revision, err := number(ctx.Fields, "expected_change_revision")
	if err != nil {
		return commands.Outcome{}, err
	}
	if err := expectEpoch(ctx.Fields, base, false); err != nil {
		return commands.Outcome{}, err
	}
	v, _ := s.trust.Get(ctx.Target, ctx.Workspace)
	principal, authenticated, ok := s.principalOf(ctx.User)
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
	}
	ev, _, err := s.evidenceFor(v, principal, authenticated, base, current)
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	v, err = s.trust.Validate(ctx.Target, ctx.Workspace, ctx.User, revision, ctx.Fields["change_digest"], base, ev, ctx.Rationale)
	if err != nil {
		return commands.Outcome{}, err
	}
	out := trustOutcome(v, v.Validation.Result)
	out.Fields["evidence_digest"] = v.Validation.Digest
	blocked := ""
	for _, item := range v.Validation.Items {
		if item.Result == resultBlocked {
			blocked += item.Item + " "
		}
	}
	if blocked != "" {
		out.Fields["blocked_items"] = blocked[:len(blocked)-1]
	}
	return out, nil
}

func (h trustCommand) decide(ctx commands.Context, base trustchange.Base) (commands.Outcome, error) {
	s := h.s
	if err := expectEpoch(ctx.Fields, base, h.kind == "activate"); err != nil {
		return commands.Outcome{}, err
	}
	v, _ := s.trust.Get(ctx.Target, ctx.Workspace)
	recomputed, err := s.recomputed(v, s.caller, base)
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	a := trustchange.Approval{ID: ctx.Target, Workspace: ctx.Workspace, Actor: ctx.User, ChangeDigest: ctx.Fields["change_digest"], EvidenceDigest: ctx.Fields["evidence_digest"],
		Recomputed: recomputed, Base: base, Rationale: ctx.Rationale}
	if h.kind == "approve" {
		v, err = s.trust.Approve(a)
		if err != nil {
			return commands.Outcome{}, err
		}
		return trustOutcome(v, trustchange.StateApproved), nil
	}
	return s.activateTrust(a, base, ctx.At)
}

func (s *Server) activateTrust(a trustchange.Approval, base trustchange.Base, at time.Time) (commands.Outcome, error) {
	var epoch uint64
	var progress trustProgress
	commit := func(intent trustchange.View) trustchange.Outcome {
		next, ok := nextEpoch(base.Epoch, s.generation())
		pub := authpolicy.Publication{Revision: revisionFor(intent.Workspace, intent.Candidate), Epoch: next, Number: base.Policy + 1, Proposer: intent.Proposer,
			Validator: intent.Validator, Approver: intent.Approver, Activator: intent.Activator}
		if !ok || s.policy.CanPublish(pub) != nil {
			return trustchange.Outcome{Err: commands.Fail(commands.DependencyUnavailable)}
		}
		epoch = next
		affected := len(s.sessions.below(intent.Workspace, next))
		receipt, err := s.ledger.Commit(s.trustDecision(intent, next, pub.Number, affected))
		var ce *decisionledger.CommitError
		if errors.As(err, &ce) && ce.Reason == decisionledger.ReasonCommitUnknown {
			s.log.Error("trust_activation_unconfirmed", "change", intent.ID)
			s.enterHold(decisionledger.ReasonCommitUnknown)
			s.sweep(intent.Workspace, ^uint64(0))
			return trustchange.Outcome{Unknown: decisionledger.ReasonCommitUnknown}
		}
		if err != nil {
			s.log.Error("trust_activation_refused", "change", intent.ID)
			return trustchange.Outcome{Err: commands.Fail(commands.DependencyUnavailable)}
		}
		origin := trustOrigin{origin: originProcess, activatedAt: at, change: intent.ID}
		if s.faulty("trust-publish") || s.publishTrust(pub, origin) != nil {
			s.log.Error("trust_publish_failed", "change", intent.ID, "sequence", receipt.Sequence)
			s.enterHold(reasonPublish)
			s.sweep(intent.Workspace, ^uint64(0))
			return trustchange.Outcome{Unknown: reasonPublish}
		}
		progress = s.sweep(intent.Workspace, next)
		return trustchange.Outcome{Activation: trustchange.Activation{Epoch: next, Sequence: receipt.Sequence, Watermark: receipt.Watermark, At: at}}
	}
	v, err := s.trust.Activate(a, commit)
	if err != nil {
		return commands.Outcome{}, err
	}
	if v.State == trustchange.StateUnknown {
		return commands.Outcome{Accepted: true, JobID: trustJobPrefix + v.ID, Fields: map[string]string{"change": v.ID, "reason": v.Unknown}}, nil
	}
	s.progress[v.ID] = progress
	return commands.Outcome{BusinessState: trustchange.StateActivated, Fields: map[string]string{"change": v.ID, "epoch": strconv.FormatUint(epoch, 10),
		"epoch_generation": strconv.FormatUint(authpolicy.EpochGeneration(epoch), 10), "epoch_number": strconv.FormatUint(authpolicy.EpochNumber(epoch), 10),
		"ledger_sequence": strconv.FormatUint(v.Activation.Sequence, 10), "sessions_ended": strconv.Itoa(progress.ended),
		"sessions_remaining": strconv.Itoa(progress.remaining), "your_session_ended": "true"}}, nil
}

func (s *Server) sweep(workspace string, epoch uint64) trustProgress {
	ended := s.sessions.below(workspace, epoch)
	for _, v := range ended {
		s.endOne(v, accesshistory.CauseTrustChange)
	}
	return trustProgress{ended: len(ended), remaining: len(s.sessions.below(workspace, epoch))}
}

func (s *Server) trustDefinitions() []commands.Definition {
	permission := func(right gate.Action) commands.Permission {
		return commands.Permission{Action: right, Kind: gate.InstallationKind}
	}
	field := func(name string, t commands.FieldType, required bool) commands.Field {
		return commands.Field{Name: name, Type: t, Required: required}
	}
	revision := field("expected_change_revision", commands.Integer, true)
	epoch := field("expected_epoch", commands.Integer, true)
	policy := field("expected_policy", commands.Integer, true)
	change := field("change_digest", commands.Text, true)
	evidence := field("evidence_digest", commands.Text, true)
	propose := []commands.Field{field("max_age_minutes", commands.Integer, true), field("step_up_minutes", commands.Integer, true), epoch, policy,
		field("expected_change_revision", commands.Integer, false)}
	define := func(id string, right gate.Action, fields []commands.Field) commands.Definition {
		return commands.Definition{ID: "trust." + id, Permission: permission(right), Mutating: true, Rationale: true, Fields: fields, Refusals: trustRefusals,
			Handler: trustCommand{s: s, kind: id}}
	}
	return []commands.Definition{
		define("propose", gate.TrustPropose, propose),
		define("validate", gate.TrustValidate, []commands.Field{revision, change, epoch}),
		define("approve", gate.TrustApprove, []commands.Field{change, evidence, epoch}),
		define("activate", gate.TrustActivate, []commands.Field{change, evidence, epoch, policy}),
		define("reject", gate.TrustApprove, []commands.Field{revision}),
		define("cancel", gate.TrustPropose, []commands.Field{revision}),
	}
}
