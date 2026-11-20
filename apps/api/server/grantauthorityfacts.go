package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func (a *serverGrantAuthority) Check(workspace, actor string) (string, error) {
	return a.s.governor(workspace, actor)
}

func (s *Server) governor(workspace, actor string) (string, error) {
	if err := s.governorCaller(workspace, s.answer.User()); err != nil {
		return "", err
	}
	facts, err := s.accessFacts(s.caller)
	if err != nil {
		return "", err
	}
	member, ok := facts.members[actor]
	user := facts.users[actor]
	if !ok || user.Kind != identity.Human || user.State != identity.Active || member.Role != identity.Administrator || !member.Effective || member.UserState != identity.Active {
		return "", inputauthority.ErrDenied
	}
	request := gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: workspace}}
	decision := s.admin.ExplainInstallation(actor, ConsoleClient, request, facts.now)
	if !decision.Allowed {
		if decision.Code == gate.AuthorityUnavailable {
			return "", gate.ErrUnavailable
		}
		return "", inputauthority.ErrDenied
	}
	return inputAuthorityFingerprint(struct {
		Person     string
		Generation uint64
	}{facts.person(actor), s.admin.Generation()}), nil
}

func (a *serverGrantAuthority) Current(spec inputauthority.Spec) (string, error) {
	if err := a.caller(spec.Workspace, a.s.answer.User()); err != nil {
		return "", err
	}
	s := a.s
	if _, err := a.domain.terms(spec.Workspace, spec.Capability); err != nil {
		return "", err
	}
	if spec.Workspace != s.bootWorkspace {
		return "", inputauthority.ErrDenied
	}
	generation := s.admin.Generation()
	if spec.Kind == inputauthority.Console {
		if spec.Recipient != ConsoleClient || spec.CeilingID != "" {
			return "", inputauthority.ErrInvalid
		}
		return inputAuthorityFingerprint(struct {
			Workspace, Client string
			Generation        uint64
		}{spec.Workspace, ConsoleClient, generation}), nil
	}
	if spec.Kind != inputauthority.Human {
		return "", inputauthority.ErrInvalid
	}
	facts, err := s.accessFacts(s.caller)
	if err != nil {
		return "", err
	}
	member, ok := facts.members[spec.Recipient]
	user := facts.users[spec.Recipient]
	if !ok || !member.Effective || member.UserState != identity.Active || user.State != identity.Active || user.Kind != identity.Human {
		return "", inputauthority.ErrDenied
	}
	ceiling, ok := a.ceilings[spec.CeilingID]
	if !ok || ceiling.Kind != inputauthority.Console || ceiling.Recipient != ConsoleClient || ceiling.Workspace != spec.Workspace || ceiling.Capability != spec.Capability {
		return "", inputauthority.ErrDenied
	}
	effect, err := a.Lookup(spec.CeilingID)
	if err != nil {
		return "", err
	}
	if !effect.Found || effect.Spec != ceiling || effect.Status != inputauthority.ActiveGrant || facts.now.Before(ceiling.NotBefore) || spec.NotBefore.Before(ceiling.NotBefore) || spec.NotAfter.After(ceiling.NotAfter) {
		return "", inputauthority.ErrDenied
	}
	grant, ok := s.grantView(spec.CeilingID)
	if !ok || !grant.Active {
		return "", inputauthority.ErrDenied
	}
	return inputAuthorityFingerprint(struct {
		Person                     string
		Generation, CeilingVersion uint64
		Ceiling                    string
		Spec                       inputauthority.Spec
	}{facts.person(spec.Recipient), generation, grant.Version, spec.CeilingID, ceiling}), nil
}

func (a *serverGrantAuthority) Lookup(id string) (inputauthority.Effect, error) {
	if err := a.caller(a.s.bootWorkspace, a.s.answer.User()); err != nil {
		return inputauthority.Effect{}, err
	}
	if a.s.faulty(a.domain.faultPrefix() + "lookup") {
		return inputauthority.Effect{}, gate.ErrUnavailable
	}
	spec, known := a.bindings[id]
	if !known {
		return inputauthority.Effect{}, inputauthority.ErrConflict
	}
	grant, found := a.s.grantView(id)
	if !found {
		return inputauthority.Effect{}, nil
	}
	if !a.sameGrant(grant, id, spec) {
		return inputauthority.Effect{}, inputauthority.ErrConflict
	}
	return inputauthority.Effect{Found: true, Spec: spec, Status: inputGrantStatus(grant, a.s.clock())}, nil
}

func (a *serverGrantAuthority) Apply(id string, spec inputauthority.Spec) error {
	if err := a.caller(spec.Workspace, a.s.answer.User()); err != nil {
		return err
	}
	pinned, ok := a.bindings[id]
	if !ok || pinned != spec {
		return inputauthority.ErrConflict
	}
	if a.s.faulty(a.domain.faultPrefix() + "before-grant") {
		return gate.ErrUnavailable
	}
	grant, err := a.grant(id, spec)
	if err != nil {
		return err
	}
	if err := a.s.admin.AddGrant(grant); err != nil {
		return err
	}
	if a.s.faulty(a.domain.faultPrefix() + "after-grant") {
		return gate.ErrUnavailable
	}
	return nil
}
