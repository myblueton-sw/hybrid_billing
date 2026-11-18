package server

import (
	"errors"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

const bulkTarget = "bulk-revocations"

type bulkCommand struct {
	s     *Server
	stage bool
	owned bool
	apply func(*Server, commands.Context, string) (commands.Outcome, error)
}

func (s *Server) callerUser() (string, bool) {
	found, ok := s.sessions.get(s.caller)
	if !ok {
		return "", false
	}
	user, known := s.dir.userOf(found.login)
	return user, known
}

func (h bulkCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	if h.stage {
		if target != bulkTarget {
			return commands.Target{}, commands.NotFound
		}
		return commands.Target{Revision: s.Revision()}, commands.Found
	}
	user, ok := s.callerUser()
	if !ok || !s.bulk.Visible(workspace, user, target) {
		return commands.Target{}, commands.NotFound
	}
	if v, found := s.bulk.Get(workspace, user, target); h.owned && (!found || v.Owner != user) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (h bulkCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	decision := s.gate.Decide(s.answer, ConsoleClient, gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind,
		Resource: gate.Resource{WorkspaceID: ctx.Workspace}})
	if !decision.Allowed {
		return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
	}
	s.bulk.Expire()
	return h.apply(s, ctx, ctx.User)
}

func bulkError(err error) error {
	switch {
	case errors.Is(err, bulkrevoke.ErrInvalid):
		return commands.Fail(commands.InvalidRequest, "users")
	case errors.Is(err, bulkrevoke.ErrTooMany), errors.Is(err, bulkrevoke.ErrOpenDraft), errors.Is(err, bulkrevoke.ErrPending), errors.Is(err, bulkrevoke.ErrEmpty):
		return commands.Fail(commands.PrerequisiteFailed)
	case errors.Is(err, bulkrevoke.ErrNotFound):
		return commands.Fail(commands.PermissionDenied)
	case errors.Is(err, bulkrevoke.ErrDigest):
		return commands.Fail(commands.InvalidRequest, "digest")
	}
	return commands.Fail(commands.InvalidStage)
}

func addTargets(s *Server, ctx commands.Context, user string) (commands.Outcome, error) {
	ids := strings.Split(ctx.Fields["users"], "\n")
	v, err := s.bulk.Add(ctx.Workspace, user, ctx.Fields["draft"], ids)
	if err != nil {
		return commands.Outcome{}, bulkError(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "draft", Fields: map[string]string{"draft": v.ID, "count": strconv.Itoa(v.Count),
		"duplicates": strconv.Itoa(v.Duplicates)}}, nil
}

func discardBulk(s *Server, ctx commands.Context, user string) (commands.Outcome, error) {
	v, err := s.bulk.Discard(ctx.Workspace, user, ctx.Target)
	if err != nil {
		return commands.Outcome{}, bulkError(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "discarded", Fields: map[string]string{"bulk": v.ID}}, nil
}

func (s *Server) bulkDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.RevokeAccessInBulk, Kind: gate.InstallationKind}
	define := func(id string, stage bool, fields []commands.Field, apply func(*Server, commands.Context, string) (commands.Outcome, error)) commands.Definition {
		return commands.Definition{ID: id, Permission: permission, Mutating: true, Rationale: true, Fields: fields, Handler: bulkCommand{s: s, stage: stage, owned: !stage, apply: apply}}
	}
	return []commands.Definition{
		define("bulk_revocation.add_targets", true, []commands.Field{{Name: "users", Type: commands.Text, Required: true}, {Name: "draft", Type: commands.ID}}, addTargets),
		define("bulk_revocation.prepare", false, []commands.Field{{Name: "effect", Type: commands.Enum, Required: true,
			Values: []string{string(bulkrevoke.SuspendMembership), string(bulkrevoke.SuspendUser)}}}, prepareBulk),
		define("bulk_revocation.execute", false, []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}, executeBulk),
		define("bulk_revocation.discard", false, nil, discardBulk),
	}
}
