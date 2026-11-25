package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

type serverPricingPolicyAuthority struct{ s *Server }

func pricingPolicyAction(action string) gate.Action {
	switch action {
	case pricingpolicy.ProposeAction:
		return gate.ProposePricingPolicy
	case pricingpolicy.ApproveAction:
		return gate.ApprovePricingPolicy
	case pricingpolicy.RevokeAction:
		return gate.RevokePricingPolicy
	case pricingpolicy.UseAction:
		return gate.UsePricingPolicy
	}
	return ""
}

func pricingPolicyDecision(allowed bool, code gate.Code, delegations []string) error {
	if code == gate.AuthorityUnavailable {
		return gate.ErrUnavailable
	}
	if !allowed || len(delegations) != 0 {
		return pricingpolicy.ErrRejected
	}
	return nil
}

func (a serverPricingPolicyAuthority) caller(workspace string, action gate.Action, fields []string) error {
	s := a.s
	if s.cfg.Seed != SeedSample || workspace != s.bootWorkspace || workspace != s.answer.Workspace() || s.answer.User() == "" || s.answer.Kind() != identity.Human {
		return pricingpolicy.ErrRejected
	}
	found, ok := s.sessions.get(s.caller)
	if !ok || found.workspace != workspace || s.sessions.isDenied(s.caller) || s.stale(found) {
		return pricingpolicy.ErrRejected
	}
	decision := s.gate.Decide(s.answer, ConsoleClient, gate.Request{Action: action, Kind: pricingauthority.ResourceKind, Fields: fields, Resource: gate.Resource{WorkspaceID: workspace}})
	return pricingPolicyDecision(decision.Allowed, decision.Code, decision.Delegations)
}

func (a serverPricingPolicyAuthority) Check(scope pricingpolicy.Scope, actor, action string) error {
	permission := pricingPolicyAction(action)
	if permission == "" || actor == "" {
		return pricingpolicy.ErrRejected
	}
	if actor == a.s.answer.User() {
		return a.caller(scope.Workspace, permission, nil)
	}
	if action != pricingpolicy.ProposeAction {
		return pricingpolicy.ErrRejected
	}
	if err := a.caller(scope.Workspace, gate.ApprovePricingPolicy, nil); err != nil {
		return err
	}
	user, ok, err := a.s.findUser(actor)
	if err != nil {
		return gate.ErrUnavailable
	}
	if !ok || user.State != identity.Active {
		return pricingpolicy.ErrRejected
	}
	decision := a.s.admin.Explain(actor, gate.Request{Action: permission, Kind: pricingauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: scope.Workspace}})
	return pricingPolicyDecision(decision.Allowed, decision.Code, decision.Delegations)
}

func (a serverPricingPolicyAuthority) inspect(workspace string) error {
	return a.caller(workspace, gate.ViewCost, []string{pricingauthority.EvidenceField})
}
