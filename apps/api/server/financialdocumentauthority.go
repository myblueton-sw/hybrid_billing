package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (a *financialERP) documentPermission(workspace, actor string, action gate.Action) error {
	s := a.server
	if workspace != s.bootWorkspace || actor == "" || actor != s.answer.User() || workspace != s.answer.Workspace() || s.answer.Kind() != identity.Human {
		return commands.Fail(commands.PermissionDenied)
	}
	found, ok := s.sessions.get(s.caller)
	if !ok || found.workspace != workspace {
		return commands.Fail(commands.Unauthenticated)
	}
	actual, code := s.access(found)
	if code != "" {
		return commands.Fail(code)
	}
	if actual.User() != actor || actual.Workspace() != workspace {
		return commands.Fail(commands.PermissionDenied)
	}
	req := gate.Request{Action: action, Kind: documentauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: workspace}}
	if action == gate.ViewCost {
		req.Fields = []string{documentauthority.EvidenceField}
	}
	switch action {
	case gate.ProposeDocumentEvidence, gate.ConfirmDocumentEvidence, gate.RevokeDocumentEvidence, gate.PrepareERPRecord, gate.ViewCost:
	default:
		return commands.Fail(commands.PermissionDenied)
	}
	d := s.gate.Decide(s.answer, ConsoleClient, req)
	if d.Code == gate.AuthorityUnavailable {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	if !d.Allowed || len(d.Delegations) != 0 {
		return commands.Fail(commands.PermissionDenied)
	}
	return nil
}

func (a *financialERP) documentProposer(workspace, actor string) error {
	user, ok, err := a.server.findUser(actor)
	if err != nil {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	if !ok || user.Kind != identity.Human || user.State != identity.Active {
		return commands.Fail(commands.PermissionDenied)
	}
	d := a.server.admin.Explain(actor, gate.Request{Action: gate.ProposeDocumentEvidence, Kind: documentauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: workspace}})
	if d.Code == gate.AuthorityUnavailable {
		return commands.Fail(commands.AuthorityUnavailable)
	}
	if !d.Allowed || len(d.Delegations) != 0 {
		return commands.Fail(commands.PermissionDenied)
	}
	return nil
}
