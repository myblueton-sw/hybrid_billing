package server

import (
	"errors"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

const (
	evidenceCommand        = "release_evidence."
	traceProposalTarget    = "trace-proposal-"
	perfProposalTarget     = "perf-proposal-"
	traceRequirementTarget = "trace-requirement-"
	perfProfileTarget      = "perf-profile:"
	evidenceApproveKind    = "approve"
	evidenceWithdrawKind   = "withdraw"
	evidenceRetireReqKind  = "retire_requirement"
	evidenceRetireProfKind = "retire_profile"
)

func mapTrace(err error) error {
	var refusal releasetrace.Refusal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refusal):
		switch refusal.Reason {
		case releasetrace.ReasonSelfApproval, releasetrace.ReasonNotProposer:
			return refuse(commands.PrerequisiteFailed)
		case releasetrace.ReasonAINotAllowed, releasetrace.ReasonNotHuman:
			return refuse(commands.InternalError)
		}
		return refuse(commands.InvalidStage)
	case errors.Is(err, releasetrace.ErrNotFound), errors.Is(err, releasetrace.ErrConflict):
		return refuse(commands.InvalidStage)
	case errors.Is(err, releasetrace.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	}
	return refuse(commands.InternalError)
}

func mapPerf(err error) error {
	var refusal perfgate.Refusal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refusal):
		switch refusal.Reason {
		case perfgate.ReasonSelfApproval, perfgate.ReasonNotProposer:
			return refuse(commands.PrerequisiteFailed)
		case perfgate.ReasonAINotAllowed, perfgate.ReasonNotHuman:
			return refuse(commands.InternalError)
		}
		return refuse(commands.InvalidStage)
	case errors.Is(err, perfgate.ErrNotFound), errors.Is(err, perfgate.ErrConflict):
		return refuse(commands.InvalidStage)
	case errors.Is(err, perfgate.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	}
	return refuse(commands.InternalError)
}

func (s *Server) evidenceEntryOf(target, workspace string) *evidenceEntry {
	if workspace != s.bootWorkspace {
		return nil
	}
	for module, prefix := range map[string]string{moduleTrace: traceProposalTarget, modulePerf: perfProposalTarget} {
		raw, found := strings.CutPrefix(target, prefix)
		if number, ok := canonicalNumber(raw); found && ok {
			return s.ev.entry(module, number)
		}
	}
	return nil
}

func (s *Server) evidenceRequirementOf(target, workspace string) (releasetrace.Key, bool) {
	raw, found := strings.CutPrefix(target, traceRequirementTarget)
	index, ok := canonicalNumber(raw)
	if !found || !ok || workspace != s.bootWorkspace || index == 0 || index > uint64(len(s.ev.requirements)) {
		return releasetrace.Key{}, false
	}
	return s.ev.requirements[index-1], true
}

func (s *Server) evidenceProfileOf(target, workspace string) string {
	id, found := strings.CutPrefix(target, perfProfileTarget)
	if !found || workspace != s.bootWorkspace || !s.ev.hasProfile(id) {
		return ""
	}
	return id
}

type evidenceHandler struct {
	s    *Server
	kind string
}

func (h evidenceHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	var found bool
	switch h.kind {
	case evidenceApproveKind, evidenceWithdrawKind:
		found = s.evidenceEntryOf(target, workspace) != nil
	case evidenceRetireReqKind:
		_, found = s.evidenceRequirementOf(target, workspace)
	default:
		found = s.evidenceProfileOf(target, workspace) != ""
	}
	if !found {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func evidenceResult(e *evidenceEntry) commands.Outcome {
	number := strconv.FormatUint(e.number, 10)
	return commands.Outcome{BusinessState: e.state, Fields: map[string]string{"module": e.module, "proposal": number, "target": evidenceProposalTarget(e.module, e.number)}}
}

func (s *Server) approveEvidence(ctx commands.Context) (*evidenceEntry, error) {
	e := s.evidenceEntryOf(ctx.Target, ctx.Workspace)
	if e.state != entryPending || e.kind != releasetrace.ChangeRetire {
		return nil, refuse(commands.InvalidStage)
	}
	if e.module == moduleTrace {
		return e, mapTrace(s.traceApprove(e, releasetrace.Actor{ID: ctx.User, Kind: releasetrace.KindHuman}, ctx.Rationale))
	}
	return e, mapPerf(s.perfApprove(e, perfgate.Actor{ID: ctx.User, Kind: perfgate.KindHuman}, ctx.Rationale))
}

func (s *Server) withdrawEvidence(ctx commands.Context) (*evidenceEntry, error) {
	e := s.evidenceEntryOf(ctx.Target, ctx.Workspace)
	if e.state != entryPending {
		return nil, refuse(commands.InvalidStage)
	}
	if e.module == moduleTrace {
		return e, mapTrace(s.traceWithdraw(e, releasetrace.Actor{ID: ctx.User, Kind: releasetrace.KindHuman}, ctx.Rationale))
	}
	return e, mapPerf(s.perfWithdraw(e, perfgate.Actor{ID: ctx.User, Kind: perfgate.KindHuman}, ctx.Rationale))
}

func (s *Server) retireRequirement(ctx commands.Context) (*evidenceEntry, error) {
	k, _ := s.evidenceRequirementOf(ctx.Target, ctx.Workspace)
	reason := ctx.Fields["reason"]
	if !releasetrace.ValidText(reason) {
		return nil, refuse(commands.InvalidRequest, "reason")
	}
	var replacement releasetrace.Key
	if raw, given := ctx.Fields["replacement"]; given {
		other, ok := s.evidenceRequirementOf(raw, ctx.Workspace)
		if !ok || other == k {
			return nil, refuse(commands.InvalidRequest, "replacement")
		}
		replacement = other
	}
	build := ""
	if s.ev.candidate != nil {
		build = s.ev.candidate.Build
	}
	v, err := s.traceRequirement(k, build)
	switch {
	case err != nil:
		return nil, mapTrace(err)
	case v.Retired, s.ev.pendingRetire(k) != nil:
		return nil, refuse(commands.InvalidStage)
	}
	e, err := s.tracePropose(releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: k, Replacement: replacement, Reason: reason}},
		releasetrace.Actor{ID: ctx.User, Kind: releasetrace.KindHuman}, ctx.Rationale)
	return e, mapTrace(err)
}

func (s *Server) retireProfile(ctx commands.Context) (*evidenceEntry, error) {
	id := s.evidenceProfileOf(ctx.Target, ctx.Workspace)
	_, retired, err := s.perfProfile(id)
	switch {
	case err != nil:
		return nil, mapPerf(err)
	case retired, s.ev.pendingProfileRetire(id) != nil:
		return nil, refuse(commands.InvalidStage)
	}
	e, err := s.perfPropose(perfgate.Change{Kind: perfgate.ChangeRetire, Retire: perfgate.Retire{Profile: id}}, perfgate.Actor{ID: ctx.User, Kind: perfgate.KindHuman},
		ctx.Rationale)
	return e, mapPerf(err)
}

func (h evidenceHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	var e *evidenceEntry
	var err error
	switch h.kind {
	case evidenceApproveKind:
		e, err = s.approveEvidence(ctx)
	case evidenceWithdrawKind:
		e, err = s.withdrawEvidence(ctx)
	case evidenceRetireReqKind:
		e, err = s.retireRequirement(ctx)
	default:
		e, err = s.retireProfile(ctx)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	out := evidenceResult(e)
	out.ResultRevision = s.Next()
	return out, nil
}

func (s *Server) evidenceDefinitions() []commands.Definition {
	define := func(permission commands.Permission, kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: evidenceCommand + kind, Permission: permission, Mutating: true, Rationale: true, Fields: f, Handler: evidenceHandler{s: s, kind: kind}}
	}
	proposalPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	retirePermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	return []commands.Definition{
		define(proposalPermission, evidenceApproveKind),
		define(proposalPermission, evidenceWithdrawKind),
		define(retirePermission, evidenceRetireReqKind, commands.Field{Name: "reason", Type: commands.Text, Required: true},
			commands.Field{Name: "replacement", Type: commands.ID}),
		define(retirePermission, evidenceRetireProfKind),
	}
}
