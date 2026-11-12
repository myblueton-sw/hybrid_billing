package gate

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/auth"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type adapter struct{ view *identity.Subjects }

func (a adapter) Subject(id string) (auth.Subject, bool) {
	kind, state, epoch, found := a.view.Subject(id)
	if !found {
		return auth.Subject{}, false
	}
	subject := auth.Subject{ID: id, State: auth.Suspended, Epoch: epoch}
	switch kind {
	case identity.Human:
		subject.Kind = auth.Human
	case identity.Service:
		subject.Kind = auth.Service
	default:
		return auth.Subject{}, false
	}
	if state == identity.Active {
		subject.State = auth.Active
	}
	return subject, true
}

func (a adapter) Member(workspaceID, id string) (uint64, bool) { return a.view.Member(workspaceID, id) }

func (a adapter) Generation() uint64 { return a.view.Generation() }

type Gate struct {
	boundary    *identity.Boundary
	view        *identity.Subjects
	permissions *auth.Registry
	clock       func() time.Time
}

type Administration struct{ permissions *auth.Registry }

func New(registry *identity.Registry, catalog Catalog) (*Gate, *Administration, error) {
	if registry == nil || registry.Clock() == nil {
		return nil, nil, ErrInvalid
	}
	view := registry.Subjects()
	permissions, err := auth.NewRegistryWithSubjects(registry.InstallationID(), catalog, registry.Clock(), adapter{view: view})
	if err != nil {
		return nil, nil, err
	}
	return &Gate{boundary: registry.Boundary(), view: view, permissions: permissions, clock: registry.Clock()}, &Administration{permissions: permissions}, nil
}

func refusal(code Code, reason Reason, actor string) Decision {
	return Decision{Code: code, Reason: reason, ActorID: actor}
}

func (g *Gate) ready() bool { return g != nil && g.permissions != nil }

func (g *Gate) confirm(answer identity.Answer) (Decision, bool) {
	if err := g.boundary.Recheck(answer); errors.Is(err, identity.ErrUnavailable) {
		return refusal(AuthorityUnavailable, ReasonUnavailable, ""), false
	} else if err != nil {
		return refusal(PermissionDenied, ReasonIdentity, ""), false
	}
	return Decision{}, true
}

func (g *Gate) Admit(answer Answer) Admission {
	if !g.ready() {
		return AdmitUnavailable
	}
	denial, ok := g.confirm(answer)
	switch {
	case ok:
		return Admitted
	case denial.Code == AuthorityUnavailable:
		return AdmitUnavailable
	default:
		return NotCurrent
	}
}

func (g *Gate) Describes(action Action, kind string, fields []string) bool {
	return g.ready() && g.permissions.Describes(action, kind, fields)
}

func (g *Gate) Clock() func() time.Time {
	if !g.ready() {
		return nil
	}
	return g.clock
}

func (g *Gate) run(answer identity.Answer, client, session, workspace string, decide func(auth.Principal) auth.Decision) Decision {
	if !g.ready() {
		return refusal(AuthorityUnavailable, ReasonUnavailable, "")
	}
	if denial, ok := g.confirm(answer); !ok {
		return denial
	}
	if workspace != answer.Workspace() {
		return refusal(PermissionDenied, ReasonWorkspace, answer.User())
	}
	if answer.Kind() != identity.Human && client != "" {
		return refusal(PermissionDenied, ReasonPrincipal, answer.User())
	}
	principal := auth.Principal{Installation: answer.Installation(), SubjectID: answer.User(), Epoch: answer.Epoch(), ClientID: client, SessionID: session}
	decided := decide(principal)
	result := Decision{Allowed: decided.Allowed, Code: decided.Code, Reason: decided.Reason, ActorID: decided.ActorID, ClientID: decided.ClientID,
		SessionID: decided.SessionID, Delegations: decided.Delegations}
	if !decided.Allowed {
		return result
	}
	if _, ok := g.confirm(answer); !ok {
		return refusal(PermissionDenied, ReasonIdentity, answer.User())
	}
	result.Stamp = Stamp{issued: true, permission: decided.Stamp, answer: answer}
	return result
}

func (g *Gate) Decide(answer identity.Answer, client string, request Request) Decision {
	return g.run(answer, client, "", request.Resource.WorkspaceID, func(principal auth.Principal) auth.Decision {
		return g.permissions.Decide(principal, request)
	})
}

func supported(answer identity.Answer, session string) bool {
	return session != "" && answer.Kind() == identity.Human
}

func (g *Gate) DecideInSupport(answer identity.Answer, session string, request Request) Decision {
	if g.ready() && !supported(answer, session) {
		return refusal(PermissionDenied, ReasonSupportSession, "")
	}
	return g.run(answer, "", session, request.Resource.WorkspaceID, func(principal auth.Principal) auth.Decision {
		return g.permissions.Decide(principal, request)
	})
}

func (g *Gate) Recheck(stamp Stamp, client string, request Request) Decision {
	if g.ready() && !stamp.issued {
		return refusal(PermissionDenied, ReasonStamp, "")
	}
	return g.run(stamp.answer, client, "", request.Resource.WorkspaceID, func(principal auth.Principal) auth.Decision {
		return g.permissions.Recheck(stamp.permission, principal, request)
	})
}

func (g *Gate) RecheckInSupport(stamp Stamp, session string, request Request) Decision {
	if g.ready() && !stamp.issued {
		return refusal(PermissionDenied, ReasonStamp, "")
	}
	if g.ready() && !supported(stamp.answer, session) {
		return refusal(PermissionDenied, ReasonSupportSession, "")
	}
	return g.run(stamp.answer, "", session, request.Resource.WorkspaceID, func(principal auth.Principal) auth.Decision {
		return g.permissions.Recheck(stamp.permission, principal, request)
	})
}

func (g *Gate) ownAuthor(approver, author string, decided auth.Decision) auth.Decision {
	if !decided.Allowed {
		return decided
	}
	if creator, owner, service := g.view.Service(author); service && (creator == approver || owner == approver) {
		return auth.Decision{Code: SelfApprovalDenied, Reason: ReasonSelfApproval, ActorID: approver}
	}
	return decided
}

func (g *Gate) CheckApproval(answer identity.Answer, author string, request Request) Decision {
	return g.run(answer, "", "", request.Resource.WorkspaceID, func(principal auth.Principal) auth.Decision {
		return g.ownAuthor(principal.SubjectID, author, g.permissions.CheckApproval(principal, author, request))
	})
}

func (g *Gate) RecheckApproval(stamp Stamp, author string, request Request) Decision {
	if g.ready() && !stamp.issued {
		return refusal(PermissionDenied, ReasonStamp, "")
	}
	return g.run(stamp.answer, "", "", request.Resource.WorkspaceID, func(principal auth.Principal) auth.Decision {
		return g.ownAuthor(principal.SubjectID, author, g.permissions.RecheckApproval(stamp.permission, principal, author, request))
	})
}

func (g *Gate) AddSupportSession(session SupportSession, approver identity.Answer) error {
	if !g.ready() {
		return ErrUnavailable
	}
	if denial, ok := g.confirm(approver); !ok {
		if denial.Code == AuthorityUnavailable {
			return ErrUnavailable
		}
		return ErrDenied
	}
	if approver.Kind() != identity.Human || session.WorkspaceID != approver.Workspace() {
		return ErrDenied
	}
	principal := auth.Principal{Installation: approver.Installation(), SubjectID: approver.User(), Epoch: approver.Epoch()}
	if err := g.permissions.AddSupportSession(session, principal); err != nil {
		return err
	}
	if _, ok := g.confirm(approver); !ok {
		_ = g.permissions.EndSupportSession(session.ID, 1)
		return ErrDenied
	}
	return nil
}

func (a *Administration) ready() bool { return a != nil && a.permissions != nil }

func (a *Administration) PutClient(client Client, expected uint64) (uint64, error) {
	if !a.ready() {
		return 0, ErrUnavailable
	}
	return a.permissions.PutClient(client, expected)
}

func (a *Administration) AddGrant(grant Grant) error {
	if !a.ready() {
		return ErrUnavailable
	}
	return a.permissions.AddGrant(grant)
}

func (a *Administration) RevokeGrant(id string, expected uint64) error {
	if !a.ready() {
		return ErrUnavailable
	}
	return a.permissions.RevokeGrant(id, expected)
}

func (a *Administration) AddDelegation(delegation Delegation) error {
	if !a.ready() {
		return ErrUnavailable
	}
	return a.permissions.AddDelegation(delegation)
}

func (a *Administration) RevokeDelegation(id string, expected uint64) error {
	if !a.ready() {
		return ErrUnavailable
	}
	return a.permissions.RevokeDelegation(id, expected)
}

func (a *Administration) EndSupportSession(id string, expected uint64) error {
	if !a.ready() {
		return ErrUnavailable
	}
	return a.permissions.EndSupportSession(id, expected)
}

func (a *Administration) SetAvailable(available bool) error {
	if !a.ready() {
		return ErrUnavailable
	}
	a.permissions.SetAvailable(available)
	return nil
}
