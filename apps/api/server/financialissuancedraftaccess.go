package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func (a *financialIssuance) draftOwnedPermission(workspace, actor, capability string) error {
	authority := a.server.draftAuthority
	if authority == nil {
		return commands.Fail(commands.UnsupportedCapability)
	}
	records, err := authority.registry.Records(workspace)
	if err != nil {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	applied := map[string]inputauthority.Spec{}
	for _, r := range records {
		if r.State != inputauthority.Applied || r.Decision == nil || r.Decision.GrantStatus != inputauthority.ActiveGrant || r.Proposal.Spec.Workspace != workspace || r.Proposal.Spec.Capability != capability {
			continue
		}
		id := r.Decision.GrantID
		if id != authority.domain.grantPrefix()+r.Digest || authority.bindings[id] != r.Proposal.Spec {
			continue
		}
		grant, ok := a.server.grantView(id)
		if !ok || !grant.Active || specHumanIneffective(grant) || !authority.sameGrant(grant, id, r.Proposal.Spec) {
			continue
		}
		applied[id] = r.Proposal.Spec
	}
	for _, spec := range applied {
		if spec.Kind != inputauthority.Human || spec.Recipient != actor {
			continue
		}
		ceiling, ok := applied[spec.CeilingID]
		if ok && authority.ceilings[spec.CeilingID] == ceiling && ceiling.Kind == inputauthority.Console && ceiling.Recipient == ConsoleClient && ceiling.CeilingID == "" {
			return nil
		}
	}
	return commands.Fail(commands.PermissionDenied)
}

func (a *financialIssuance) draftActor(workspace, actor, capability string) error {
	if a == nil || a.server == nil || a.server.cfg.Seed != SeedSample {
		return commands.Fail(commands.UnsupportedCapability)
	}
	if workspace != a.server.bootWorkspace || actor == "" {
		return commands.Fail(commands.PermissionDenied)
	}
	terms, err := draftAuthorityTerms(workspace, capability)
	if err != nil {
		return commands.Fail(commands.InvalidRequest)
	}
	info, found, err := a.server.findUser(actor)
	if err != nil {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	if !found || info.Kind != identity.Human || info.State != identity.Active {
		return commands.Fail(commands.PermissionDenied)
	}
	decision := a.server.admin.Explain(actor, gate.Request{Fields: terms.Fields, Action: terms.Action, Kind: draftauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: workspace}})
	if decision.Code == gate.AuthorityUnavailable {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	if !decision.Allowed || len(decision.Delegations) != 0 {
		return commands.Fail(commands.PermissionDenied)
	}
	return a.draftOwnedPermission(workspace, actor, capability)
}

func (a *financialIssuance) draftCaller(workspace, actor, capability string) error {
	if a == nil || a.server == nil || a.server.cfg.Seed != SeedSample {
		return commands.Fail(commands.UnsupportedCapability)
	}
	s := a.server
	if workspace != s.bootWorkspace || actor != s.answer.User() || workspace != s.answer.Workspace() || s.answer.Kind() != identity.Human {
		return commands.Fail(commands.PermissionDenied)
	}
	session, ok := s.sessions.get(s.caller)
	if !ok || session.workspace != workspace {
		return commands.Fail(commands.Unauthenticated)
	}
	actual, code := s.access(session)
	if code != "" {
		return commands.Fail(code)
	}
	if actual.User() != actor || actual.Workspace() != workspace || actual.Kind() != identity.Human {
		return commands.Fail(commands.PermissionDenied)
	}
	terms, err := draftAuthorityTerms(workspace, capability)
	if err != nil {
		return commands.Fail(commands.InvalidRequest)
	}
	decision := s.gate.Decide(actual, ConsoleClient, gate.Request{Fields: terms.Fields, Action: terms.Action, Kind: draftauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: workspace}})
	if decision.Code == gate.AuthorityUnavailable {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	if !decision.Allowed || len(decision.Delegations) != 0 {
		return commands.Fail(commands.PermissionDenied)
	}
	if s.held() {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return a.draftOwnedPermission(workspace, actor, capability)
}

func (a *financialIssuance) draftOperation(workspace, actor, capability string) error {
	if capability != draftauthority.Submit && capability != draftauthority.Approve {
		return commands.Fail(commands.InvalidRequest)
	}
	if err := a.draftCaller(workspace, actor, capability); err != nil {
		return err
	}
	return a.draftCaller(workspace, actor, draftauthority.Inspect)
}
