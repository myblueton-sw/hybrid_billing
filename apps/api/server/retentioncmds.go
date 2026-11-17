package server

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

const (
	retentionCommand        = "retention."
	retentionProposalTarget = "retention-proposal-"
	retentionClassTarget    = "retention-class:"
	retentionArtifactTarget = "retention-artifact:"
	retentionTenantTarget   = "retention-tenant:"
	retentionHoldTarget     = "retention-hold:"
	retentionRunTarget      = "retention-run-"
	retentionApproveKind    = "approve"
	retentionWithdrawKind   = "withdraw"
	retentionPolicyKind     = "propose_policy"
	retentionHoldKind       = "place_hold"
	retentionReleaseKind    = "propose_release"
	retentionStopKind       = "stop_run"
	retentionMaxPeriod      = 36500
	retentionMaxGrace       = 365
	retentionHeld           = "held"
	retentionPartlyHeld     = "partially_preserved"
	retentionLiveHolds      = 100
)

var retentionImpactFields = []string{"impact_as_of", "impact_count", "impact_digest"}

func mapRetention(err error) error {
	var refusal retention.Refusal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refusal):
		switch refusal.Reason {
		case retention.ReasonSelfApproval, retention.ReasonNotProposer:
			return refuse(commands.PrerequisiteFailed)
		case retention.ReasonAINotAllowed, retention.ReasonNotHuman, retention.ReasonNotSystem:
			return refuse(commands.InternalError)
		}
		return refuse(commands.InvalidStage)
	case errors.Is(err, retention.ErrNotFound), errors.Is(err, retention.ErrConflict):
		return refuse(commands.InvalidStage)
	case errors.Is(err, retention.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	}
	return refuse(commands.InternalError)
}

func canonicalNumber(raw string) (uint64, bool) {
	number, err := strconv.ParseUint(raw, 10, 64)
	return number, err == nil && strconv.FormatUint(number, 10) == raw
}

func (s *Server) retentionEntryOf(target, workspace string) *retentionEntry {
	raw, found := strings.CutPrefix(target, retentionProposalTarget)
	number, ok := canonicalNumber(raw)
	if !found || !ok || workspace != s.bootWorkspace {
		return nil
	}
	e := s.ret.entry(number)
	if e == nil || !s.ret.visibleEntry(e, workspace) {
		return nil
	}
	return e
}

func (s *Server) retentionClassOf(target, workspace string) string {
	class, found := strings.CutPrefix(target, retentionClassTarget)
	if !found || workspace != s.bootWorkspace {
		return ""
	}
	for _, c := range retention.Classes {
		if c == class {
			return c
		}
	}
	return ""
}

func (s *Server) retentionHoldScopeOf(target, workspace string) (string, string) {
	if workspace != s.bootWorkspace {
		return "", ""
	}
	if id, found := strings.CutPrefix(target, retentionArtifactTarget); found && s.ret.visibleArtifact(id, workspace) {
		return id, ""
	}
	if tenant, found := strings.CutPrefix(target, retentionTenantTarget); found && tenant == workspace {
		return "", tenant
	}
	return "", ""
}

func (s *Server) retentionHoldOf(target, workspace string) *retentionHold {
	id, found := strings.CutPrefix(target, retentionHoldTarget)
	if !found || workspace != s.bootWorkspace || !s.ret.visibleHold(id, workspace) {
		return nil
	}
	return s.ret.hold(id)
}

func (s *Server) retentionRunOf(target, workspace string) *retentionRun {
	raw, found := strings.CutPrefix(target, retentionRunTarget)
	_, ok := canonicalNumber(raw)
	id := retentionRunPrefix + raw
	if !found || !ok || workspace != s.bootWorkspace || !s.ret.visibleRun(id, workspace) {
		return nil
	}
	return s.ret.run(id)
}

type retentionHandler struct {
	s    *Server
	kind string
}

func (h retentionHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	var found bool
	switch h.kind {
	case retentionApproveKind, retentionWithdrawKind:
		found = s.retentionEntryOf(target, workspace) != nil
	case retentionPolicyKind:
		found = s.retentionClassOf(target, workspace) != ""
	case retentionHoldKind:
		id, tenant := s.retentionHoldScopeOf(target, workspace)
		found = id != "" || tenant != ""
	case retentionReleaseKind:
		found = s.retentionHoldOf(target, workspace) != nil
	default:
		found = s.retentionRunOf(target, workspace) != nil
	}
	if !found {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

type retentionResult struct {
	state  string
	fields map[string]string
}

func retentionEntryResult(e *retentionEntry) retentionResult {
	number := strconv.FormatUint(e.view.Number, 10)
	return retentionResult{state: e.view.State, fields: map[string]string{"proposal": number, "target": retentionProposalTarget + number}}
}

func reviewedImpact(fields map[string]string, kind string) (retention.Impact, error) {
	given := []string{}
	for _, name := range retentionImpactFields {
		if _, ok := fields[name]; ok {
			given = append(given, name)
		}
	}
	switch {
	case len(given) == 0:
		return retention.Impact{}, nil
	case kind != retention.ChangePolicy:
		return retention.Impact{}, refuse(commands.InvalidRequest, given[0])
	}
	at, atErr := time.Parse(time.RFC3339Nano, fields["impact_as_of"])
	count, countErr := strconv.Atoi(fields["impact_count"])
	digest := fields["impact_digest"]
	switch {
	case atErr != nil:
		return retention.Impact{}, refuse(commands.InvalidRequest, "impact_as_of")
	case countErr != nil || count < 0:
		return retention.Impact{}, refuse(commands.InvalidRequest, "impact_count")
	case !validPlanHash(digest):
		return retention.Impact{}, refuse(commands.InvalidRequest, "impact_digest")
	}
	return retention.Impact{AsOf: at.UTC(), Count: count, Digest: digest}, nil
}

func (s *Server) approveRetention(ctx commands.Context, reader retention.Actor) (retentionResult, error) {
	e := s.retentionEntryOf(ctx.Target, ctx.Workspace)
	if e.view.State != retention.ProposalPending || e.view.Change.Kind == retention.ChangePlan {
		return retentionResult{}, refuse(commands.InvalidStage)
	}
	reviewed, err := reviewedImpact(ctx.Fields, e.view.Change.Kind)
	if err != nil {
		return retentionResult{}, err
	}
	if err := s.retentionApprove(e, reader, reviewed, ctx.Rationale); err != nil {
		return retentionResult{}, mapRetention(err)
	}
	return retentionEntryResult(e), nil
}

func (s *Server) withdrawRetention(ctx commands.Context, reader retention.Actor) (retentionResult, error) {
	e := s.retentionEntryOf(ctx.Target, ctx.Workspace)
	if e.view.State != retention.ProposalPending {
		return retentionResult{}, refuse(commands.InvalidStage)
	}
	if err := s.retentionWithdraw(e, reader, ctx.Rationale); err != nil {
		return retentionResult{}, mapRetention(err)
	}
	return retentionEntryResult(e), nil
}

func boundedField(fields map[string]string, name string, low, high int) (int, error) {
	value, err := strconv.Atoi(fields[name])
	if err != nil || value < low || value > high {
		return 0, refuse(commands.InvalidRequest, name)
	}
	return value, nil
}

func (s *Server) proposePolicy(ctx commands.Context, reader retention.Actor) (retentionResult, error) {
	class := s.retentionClassOf(ctx.Target, ctx.Workspace)
	period, err := boundedField(ctx.Fields, "period", 1, retentionMaxPeriod)
	if err != nil {
		return retentionResult{}, err
	}
	grace, err := boundedField(ctx.Fields, "grace", 0, retentionMaxGrace)
	if err != nil {
		return retentionResult{}, err
	}
	if s.ret.pendingPolicy(class) != nil {
		return retentionResult{}, refuse(commands.InvalidStage)
	}
	current, err := s.retentionPolicy(class)
	switch {
	case err == nil && current.Policy.Period == period && current.Policy.Grace == grace:
		return retentionResult{}, refuse(commands.InvalidStage)
	case err != nil && !errors.Is(err, retention.ErrNotFound):
		return retentionResult{}, mapRetention(err)
	}
	e, err := s.retentionPropose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: class, Period: period, Grace: grace}}, reader, ctx.Workspace,
		ctx.Rationale)
	if err != nil {
		return retentionResult{}, mapRetention(err)
	}
	return retentionEntryResult(e), nil
}

func (s *Server) nextHoldID() (string, error) {
	for {
		id := retentionHoldPrefix + strconv.FormatUint(s.ret.holdSeq, 10)
		_, err := s.retentionHold(id)
		switch {
		case errors.Is(err, retention.ErrNotFound):
			return id, nil
		case err != nil:
			return "", err
		}
		s.ret.holdSeq++
	}
}

func (s *Server) placeHold(ctx commands.Context, reader retention.Actor) (retentionResult, error) {
	s.ret.holdSeq++
	artifact, tenant := s.retentionHoldScopeOf(ctx.Target, ctx.Workspace)
	owner, reason, class := ctx.Fields["owner"], ctx.Fields["reason"], ctx.Fields["class"]
	grantees, err := s.grantees(s.caller)
	if err != nil {
		return retentionResult{}, mapIdentity(err)
	}
	known := false
	for _, g := range grantees {
		known = known || g.UserID == owner
	}
	switch {
	case !known:
		return retentionResult{}, refuse(commands.InvalidRequest, "owner")
	case !retention.ValidText(reason):
		return retentionResult{}, refuse(commands.InvalidRequest, "reason")
	case class != "" && tenant == "":
		return retentionResult{}, refuse(commands.InvalidRequest, "class")
	}
	live, err := s.ret.liveHolds(ctx.Workspace, func(id string) (bool, error) {
		v, err := s.retentionHold(id)
		return v.Released, err
	})
	switch {
	case err != nil:
		return retentionResult{}, mapRetention(err)
	case live >= retentionLiveHolds:
		return retentionResult{}, refuse(commands.PrerequisiteFailed)
	}
	id, err := s.nextHoldID()
	if err != nil {
		return retentionResult{}, mapRetention(err)
	}
	h := retention.Hold{ID: id, Owner: owner, Reason: reason}
	if artifact != "" {
		h.Artifacts = []string{artifact}
	} else {
		h.Tenants = []string{tenant}
	}
	if class != "" {
		h.Classes = []string{class}
	}
	v, err := s.retentionPlaceHold(h, reader, ctx.Workspace, ctx.Rationale)
	if err != nil {
		return retentionResult{}, mapRetention(err)
	}
	fields := map[string]string{"hold": id, "target": retentionHoldTarget + id}
	state := retentionHeld
	for _, key := range retentionHoldCounts {
		fields[key] = strconv.Itoa(v.Counts[key])
		if v.Counts[key] > 0 {
			state = retentionPartlyHeld
		}
	}
	return retentionResult{state: state, fields: fields}, nil
}

func (s *Server) proposeRelease(ctx commands.Context, reader retention.Actor) (retentionResult, error) {
	h := s.retentionHoldOf(ctx.Target, ctx.Workspace)
	reason := ctx.Fields["reason"]
	if !retention.ValidText(reason) {
		return retentionResult{}, refuse(commands.InvalidRequest, "reason")
	}
	v, err := s.retentionHold(h.id)
	switch {
	case err != nil:
		return retentionResult{}, mapRetention(err)
	case v.Released, s.ret.pendingRelease(h.id) != nil:
		return retentionResult{}, refuse(commands.InvalidStage)
	}
	e, err := s.retentionPropose(retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: h.id, Reason: reason}}, reader, ctx.Workspace, ctx.Rationale)
	if err != nil {
		return retentionResult{}, mapRetention(err)
	}
	return retentionEntryResult(e), nil
}

func (s *Server) stopRun(ctx commands.Context, reader retention.Actor) (retentionResult, error) {
	r := s.retentionRunOf(ctx.Target, ctx.Workspace)
	v, err := s.retentionRunView(r.id)
	switch {
	case err != nil:
		return retentionResult{}, mapRetention(err)
	case v.State != retention.RunRunning:
		return retentionResult{}, refuse(commands.InvalidStage)
	}
	err = s.retentionCall("stop", func() (err error) { v, err = s.retention.Stop(r.id, reader); return })
	if err != nil {
		return retentionResult{}, mapRetention(err)
	}
	return retentionResult{state: v.State, fields: map[string]string{"run": r.id, "target": ctx.Target, "stopped": strconv.Itoa(v.Counts[retention.OutcomeStopped])}}, nil
}

func (h retentionHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	reader := retention.Actor{ID: ctx.User, Kind: retention.KindHuman}
	var result retentionResult
	var err error
	switch h.kind {
	case retentionApproveKind:
		result, err = s.approveRetention(ctx, reader)
	case retentionWithdrawKind:
		result, err = s.withdrawRetention(ctx, reader)
	case retentionPolicyKind:
		result, err = s.proposePolicy(ctx, reader)
	case retentionHoldKind:
		result, err = s.placeHold(ctx, reader)
	case retentionReleaseKind:
		result, err = s.proposeRelease(ctx, reader)
	default:
		result, err = s.stopRun(ctx, reader)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: result.state, Fields: result.fields}, nil
}

func (s *Server) retentionDefinitions() []commands.Definition {
	define := func(permission commands.Permission, kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: retentionCommand + kind, Permission: permission, Mutating: true, Rationale: true, Fields: f, Handler: retentionHandler{s: s, kind: kind}}
	}
	proposalPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	holdPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	runPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	return []commands.Definition{
		define(proposalPermission, retentionApproveKind, commands.Field{Name: "impact_as_of", Type: commands.Timestamp},
			commands.Field{Name: "impact_count", Type: commands.Integer}, commands.Field{Name: "impact_digest", Type: commands.Text}),
		define(proposalPermission, retentionWithdrawKind),
		define(proposalPermission, retentionPolicyKind, commands.Field{Name: "period", Type: commands.Integer, Required: true},
			commands.Field{Name: "grace", Type: commands.Integer, Required: true}),
		define(holdPermission, retentionHoldKind, commands.Field{Name: "owner", Type: commands.ID, Required: true},
			commands.Field{Name: "reason", Type: commands.Text, Required: true},
			commands.Field{Name: "class", Type: commands.Enum, Values: append([]string{}, retention.Classes...)}),
		define(holdPermission, retentionReleaseKind, commands.Field{Name: "reason", Type: commands.Text, Required: true}),
		define(runPermission, retentionStopKind),
	}
}
