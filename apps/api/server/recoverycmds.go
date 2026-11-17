package server

import (
	"errors"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

const (
	recoveryCommand        = "recovery."
	recoveryProposalTarget = "recovery-proposal-"
	recoveryInstallTarget  = "recovery-installation:"
	recoveryApproveKind    = "approve"
	recoveryWithdrawKind   = "withdraw"
	recoveryLockKind       = "lock"
	recoveryUnlockKind     = "propose_unlock"
)

func mapRecovery(err error) error {
	var refusal recovery.Refusal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refusal):
		switch refusal.Reason {
		case recovery.ReasonSelfApproval, recovery.ReasonNotProposer:
			return refuse(commands.PrerequisiteFailed)
		case recovery.ReasonAINotAllowed, recovery.ReasonNotHuman, recovery.ReasonNotWriter, recovery.ReasonNotVerifier:
			return refuse(commands.InternalError)
		}
		return refuse(commands.InvalidStage)
	case errors.Is(err, recovery.ErrNotFound), errors.Is(err, recovery.ErrConflict):
		return refuse(commands.InvalidStage)
	case errors.Is(err, recovery.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	}
	return refuse(commands.InternalError)
}

func (s *Server) recoveryEntryOf(target, workspace string) *recoveryEntry {
	raw, found := strings.CutPrefix(target, recoveryProposalTarget)
	number, err := strconv.ParseUint(raw, 10, 64)
	if !found || err != nil || strconv.FormatUint(number, 10) != raw || workspace != s.bootWorkspace {
		return nil
	}
	e := s.rec.entry(number)
	if e == nil || s.rec.tenantOf(e) != workspace {
		return nil
	}
	return e
}

func (s *Server) recoveryInstallOf(target, workspace string) *recoveryInstall {
	id, found := strings.CutPrefix(target, recoveryInstallTarget)
	i := s.rec.install(id)
	if !found || i == nil || workspace != s.bootWorkspace || i.tenant != workspace {
		return nil
	}
	return i
}

type recoveryHandler struct {
	s    *Server
	kind string
}

func (h recoveryHandler) proposal() bool {
	return h.kind == recoveryApproveKind || h.kind == recoveryWithdrawKind
}

func (h recoveryHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	found := s.recoveryInstallOf(target, workspace) != nil
	if h.proposal() {
		found = s.recoveryEntryOf(target, workspace) != nil
	}
	if !found {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

type recoveryResult struct {
	state  string
	fields map[string]string
}

func entryResult(e *recoveryEntry) recoveryResult {
	number := strconv.FormatUint(e.view.Number, 10)
	return recoveryResult{state: e.view.State, fields: map[string]string{"proposal": number, "target": recoveryProposalTarget + number}}
}

func (s *Server) approveRecovery(ctx commands.Context, reader recovery.Actor) (recoveryResult, error) {
	e := s.recoveryEntryOf(ctx.Target, ctx.Workspace)
	if e.view.State != recovery.ProposalPending || e.view.Change.Kind != recovery.ChangeUnlock {
		return recoveryResult{}, refuse(commands.InvalidStage)
	}
	if err := s.recoveryApprove(e, reader, ctx.Rationale); err != nil {
		return recoveryResult{}, mapRecovery(err)
	}
	return entryResult(e), nil
}

func (s *Server) withdrawRecovery(ctx commands.Context, reader recovery.Actor) (recoveryResult, error) {
	e := s.recoveryEntryOf(ctx.Target, ctx.Workspace)
	if e.view.State != recovery.ProposalPending {
		return recoveryResult{}, refuse(commands.InvalidStage)
	}
	if err := s.recoveryWithdraw(e, reader, ctx.Rationale); err != nil {
		return recoveryResult{}, mapRecovery(err)
	}
	return entryResult(e), nil
}

func (s *Server) lockRecovery(ctx commands.Context, reader recovery.Actor) (recoveryResult, error) {
	i := s.recoveryInstallOf(ctx.Target, ctx.Workspace)
	state, _, _, err := s.recoveryStatus(i.id)
	switch {
	case err != nil:
		return recoveryResult{}, mapRecovery(err)
	case state != recovery.StateServing:
		return recoveryResult{}, refuse(commands.InvalidStage)
	}
	if err := s.recoveryLock(i, reader, ctx.Rationale); err != nil {
		return recoveryResult{}, mapRecovery(err)
	}
	return recoveryResult{state: recovery.StateLocked, fields: map[string]string{"installation": i.id, "target": ctx.Target}}, nil
}

func (s *Server) proposeUnlock(ctx commands.Context, reader recovery.Actor) (recoveryResult, error) {
	i := s.recoveryInstallOf(ctx.Target, ctx.Workspace)
	lost := ctx.Fields["lost"]
	state, generation, _, err := s.recoveryStatus(i.id)
	switch {
	case err != nil:
		return recoveryResult{}, mapRecovery(err)
	case state == recovery.StateServing, s.rec.pendingUnlock(i.id):
		return recoveryResult{}, refuse(commands.InvalidStage)
	case !validPlanHash(lost), state == recovery.StateLocked && lost != recovery.LostDigest(nil):
		return recoveryResult{}, refuse(commands.InvalidRequest, "lost")
	}
	e, err := s.recoveryPropose(recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: i.id, Generation: generation, Lost: lost}}, reader,
		ctx.Rationale)
	if err != nil {
		return recoveryResult{}, mapRecovery(err)
	}
	return entryResult(e), nil
}

func (h recoveryHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	reader := recovery.Actor{ID: ctx.User, Kind: recovery.KindHuman}
	var result recoveryResult
	var err error
	switch h.kind {
	case recoveryApproveKind:
		result, err = s.approveRecovery(ctx, reader)
	case recoveryWithdrawKind:
		result, err = s.withdrawRecovery(ctx, reader)
	case recoveryLockKind:
		result, err = s.lockRecovery(ctx, reader)
	default:
		result, err = s.proposeUnlock(ctx, reader)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: result.state, Fields: result.fields}, nil
}

func (s *Server) recoveryDefinitions() []commands.Definition {
	define := func(permission commands.Permission, kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: recoveryCommand + kind, Permission: permission, Mutating: true, Rationale: true, Fields: f, Handler: recoveryHandler{s: s, kind: kind}}
	}
	proposalPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	containmentPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	return []commands.Definition{
		define(proposalPermission, recoveryApproveKind),
		define(proposalPermission, recoveryWithdrawKind),
		define(proposalPermission, recoveryUnlockKind, commands.Field{Name: "lost", Type: commands.Text, Required: true}),
		define(containmentPermission, recoveryLockKind),
	}
}
