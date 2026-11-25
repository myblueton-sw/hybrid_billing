package server

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

type financialProtectedRules struct {
	ID, Workspace, Proposer, Confirmer, Reason, Digest, State string
	Rules                                                     protected.Rules
	Module                                                    protected.RulesView
}

func protectedRuleDigest(p financialProtectedRules) (string, error) {
	return financialEvidenceDigest("issuance_protected_rules", struct {
		ID, Workspace, Proposer, Reason string
		Rules                           protected.Rules
	}{p.ID, p.Workspace, p.Proposer, p.Reason, p.Rules})
}
func (a *financialIssuance) protectedRuleCaller(workspace, actor string) error {
	if a == nil || a.server == nil || a.server.cfg.Seed != SeedSample {
		return commands.Fail(commands.UnsupportedCapability)
	}
	s := a.server
	if workspace != s.bootWorkspace || workspace != s.answer.Workspace() || actor != s.answer.User() || s.answer.Kind() != identity.Human {
		return commands.Fail(commands.PermissionDenied)
	}
	session, ok := s.sessions.get(s.caller)
	if !ok || session.workspace != workspace {
		return commands.Fail(commands.Unauthenticated)
	}
	current, code := s.access(session)
	if code != "" {
		return commands.Fail(code)
	}
	if current.User() != actor || current.Workspace() != workspace || current.Kind() != identity.Human {
		return commands.Fail(commands.PermissionDenied)
	}
	return a.profileCaller(workspace, actor)
}
func (a *financialIssuance) currentProtectedRules(workspace string, p financialProtectedRules) error {
	digest, err := protectedRuleDigest(p)
	if err != nil || p.Workspace != workspace || digest != p.Digest {
		return commands.Fail(commands.EvidenceStale)
	}
	if _, err := a.server.governor(workspace, p.Proposer); err != nil {
		return err
	}
	if p.Confirmer != "" {
		if _, err := a.server.governor(workspace, p.Confirmer); err != nil {
			return err
		}
	}
	return nil
}
func (a *financialIssuance) protectedRulesModule(workspace string, p financialProtectedRules) (bool, error) {
	state, err := a.fields.RulesOf(workspace)
	if err != nil || len(state.History) != 1 || state.History[0] != p.Module || p.Module.Rules != p.Rules || p.Module.Proposer != p.Proposer || p.Module.Revision != 1 || p.Module.Discarder != "" {
		return false, commands.Fail(commands.EvidenceStale)
	}
	if state.Pending != nil && state.Active == nil && *state.Pending == p.Module && p.Module.Confirmer == "" {
		return false, nil
	}
	if state.Active != nil && state.Pending == nil && *state.Active == p.Module && p.Module.Confirmer == p.Confirmer && p.Confirmer != "" {
		return true, nil
	}
	return false, commands.Fail(commands.EvidenceStale)
}
func (a *financialIssuance) protectedRulesReadback(workspace string, p financialProtectedRules) (financialProtectedRules, error) {
	active, err := a.protectedRulesModule(workspace, p)
	if err != nil {
		return financialProtectedRules{}, err
	}
	if p.State == "proposed" && active || p.State == "confirmed" && !active {
		return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
	}
	if p.State != "proposed" && p.State != "confirmed" {
		return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
	}
	return p, nil
}
func (a *financialIssuance) proposeProtectedRules(workspace, id, actor, reason string, rules protected.Rules) (financialProtectedRules, error) {
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return financialProtectedRules{}, err
	}
	if !validRequestID(id) || !documentReason(reason) || rules.Approvals < 1 || rules.Approvals > 3 || rules.Validity < time.Hour || rules.Validity > 30*24*time.Hour || rules.MaxLead < time.Hour || rules.MaxLead > 366*24*time.Hour || rules.ChannelAge < 24*time.Hour || rules.ChannelAge > 90*24*time.Hour {
		return financialProtectedRules{}, commands.Fail(commands.InvalidRequest)
	}
	p := financialProtectedRules{ID: id, Workspace: workspace, Proposer: actor, Reason: reason, Rules: rules}
	var err error
	p.Digest, err = protectedRuleDigest(p)
	if err != nil {
		return financialProtectedRules{}, err
	}
	if a.protectedRules != nil {
		old := *a.protectedRules
		if old.ID != id || old.Proposer != actor || old.Digest != p.Digest {
			return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentProtectedRules(workspace, old); err != nil {
			return financialProtectedRules{}, err
		}
		if old.State != "registration_unknown" {
			return a.protectedRulesReadback(workspace, old)
		}
		p = old
		if p.Module.Revision != 0 {
			if active, err := a.protectedRulesModule(workspace, p); err != nil || active {
				if err != nil {
					return financialProtectedRules{}, err
				}
				return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
			}
		}
	} else {
		if _, err := a.fields.RulesOf(workspace); !errors.Is(err, protected.ErrNotFound) {
			return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-protected-rules-before-propose") {
			return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State = "registration_unknown"
		saved := p
		a.protectedRules = &saved
	}
	view, err := a.fields.Propose(workspace, rules, protected.Actor{ID: actor, Kind: protected.KindHuman})
	if err != nil {
		return financialProtectedRules{}, err
	}
	if view.Rules != rules || view.Proposer != actor || view.Revision != 1 || view.Confirmer != "" || view.Discarder != "" || p.Module.Revision != 0 && view != p.Module {
		return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = view
	saved := p
	a.protectedRules = &saved
	if a.server.faulty("issuance-protected-rules-after-propose") {
		return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "proposed"
	saved = p
	a.protectedRules = &saved
	if a.server.faulty("issuance-protected-rules-after-propose-publication") {
		return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
	}
	return p, nil
}
func (a *financialIssuance) confirmProtectedRules(workspace, id, digest, actor string) (financialProtectedRules, error) {
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return financialProtectedRules{}, err
	}
	if a.protectedRules == nil || a.protectedRules.ID != id || a.protectedRules.Digest != digest {
		return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
	}
	p := *a.protectedRules
	if p.Proposer == actor {
		return financialProtectedRules{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Confirmer != "" && p.Confirmer != actor {
		return financialProtectedRules{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentProtectedRules(workspace, p); err != nil {
		return financialProtectedRules{}, err
	}
	if p.State == "confirmed" {
		return a.protectedRulesReadback(workspace, p)
	}
	if p.State != "proposed" && p.State != "confirmation_unknown" {
		return financialProtectedRules{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if _, err := a.protectedRulesModule(workspace, p); err != nil {
		return financialProtectedRules{}, err
	}
	if a.server.faulty("issuance-protected-rules-before-confirm") {
		return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "confirmation_unknown"
	p.Confirmer = actor
	saved := p
	a.protectedRules = &saved
	view, err := a.fields.Confirm(workspace, p.Module.Revision, protected.Actor{ID: actor, Kind: protected.KindHuman})
	if err != nil {
		return financialProtectedRules{}, err
	}
	if view.Rules != p.Rules || view.Proposer != p.Proposer || view.Confirmer != actor || view.Revision != p.Module.Revision || view.Proposed != p.Module.Proposed || view.Discarder != "" {
		return financialProtectedRules{}, commands.Fail(commands.EvidenceStale)
	}
	p.Module = view
	saved = p
	a.protectedRules = &saved
	if a.server.faulty("issuance-protected-rules-after-confirm") {
		return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "confirmed"
	saved = p
	a.protectedRules = &saved
	if a.server.faulty("issuance-protected-rules-after-confirm-publication") {
		return financialProtectedRules{}, commands.Fail(commands.DependencyUnavailable)
	}
	return p, nil
}
