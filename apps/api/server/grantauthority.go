package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/financialrequestauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type serverGrantAuthority struct {
	domain   grantAuthorityDomain
	limits   inputauthority.Limits
	s        *Server
	registry *inputauthority.Registry
	bindings map[string]inputauthority.Spec
	ceilings map[string]inputauthority.Spec
}

func newServerGrantAuthority(s *Server, limits inputauthority.Limits, domain grantAuthorityDomain) (*serverGrantAuthority, error) {
	if s == nil || s.cfg.Seed != SeedSample || (domain != draftGrantAuthority && domain != protectedGrantAuthority && domain != termGrantAuthority && domain != erpFinalPermissionDomain && domain != erpFinalRequestDomain && domain != inputGrantAuthority && domain != pricingGrantAuthority && domain != financialPermissionDomain && domain != financialRequestDomain && domain != documentGrantAuthority && domain != resultGrantAuthority) {
		return nil, inputauthority.ErrInvalid
	}
	a := &serverGrantAuthority{s: s, domain: domain, limits: limits, bindings: make(map[string]inputauthority.Spec), ceilings: make(map[string]inputauthority.Spec)}
	var err error
	switch domain {
	case draftGrantAuthority:
		a.registry, err = draftauthority.NewRegistry(limits, a, a, a, s.clock)
	case protectedGrantAuthority:
		a.registry, err = protectedauthority.NewRegistry(limits, a, a, a, s.clock)
	case termGrantAuthority:
		a.registry, err = termsauthority.NewRegistry(limits, a, a, a, s.clock)
	case erpFinalPermissionDomain:
		a.registry, err = financialpermission.NewERPFinalRegistry(limits, a, a, a, s.clock)
	case erpFinalRequestDomain:
		a.registry, err = financialrequestauthority.NewERPFinalRegistry(limits, a, a, a, s.clock)
	case resultGrantAuthority:
		a.registry, err = resultauthority.NewRegistry(limits, a, a, a, s.clock)
	case documentGrantAuthority:
		a.registry, err = documentauthority.NewRegistry(limits, a, a, a, s.clock)
	case inputGrantAuthority:
		a.registry, err = inputauthority.NewRegistry(limits, a, a, a, s.clock)
	case pricingGrantAuthority:
		a.registry, err = pricingauthority.NewRegistry(limits, a, a, a, s.clock)
	case financialPermissionDomain:
		a.registry, err = financialpermission.NewRegistry(limits, a, a, a, s.clock)
	case financialRequestDomain:
		a.registry, err = financialrequestauthority.NewRegistry(limits, a, a, a, s.clock)
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func inputAuthorityFingerprint(value any) string {
	body, _ := json.Marshal(value)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (a *serverGrantAuthority) caller(workspace, actor string) error {
	return a.s.governorCaller(workspace, actor)
}

func (s *Server) governorCaller(workspace, actor string) error {
	if workspace != s.bootWorkspace || actor == "" || actor != s.answer.User() || s.answer.Workspace() != workspace {
		return inputauthority.ErrDenied
	}
	request := gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: workspace}}
	decision := s.gate.Decide(s.answer, ConsoleClient, request)
	if !decision.Allowed {
		if decision.Code == gate.AuthorityUnavailable {
			return gate.ErrUnavailable
		}
		return inputauthority.ErrDenied
	}
	return nil
}

func (a *serverGrantAuthority) proposeLocked(p inputauthority.Proposal) (inputauthority.Record, error) {
	if err := a.caller(p.Spec.Workspace, p.Proposer); err != nil {
		return inputauthority.Record{}, err
	}
	record, err := a.registry.Propose(p)
	if err != nil {
		return inputauthority.Record{}, err
	}
	id := a.domain.grantPrefix() + record.Digest
	if prior, ok := a.bindings[id]; ok && prior != record.Proposal.Spec {
		return inputauthority.Record{}, inputauthority.ErrConflict
	}
	a.bindings[id] = record.Proposal.Spec
	return record, nil
}

func (a *serverGrantAuthority) approveLocked(i inputauthority.Intent) (inputauthority.Record, error) {
	if err := a.caller(i.Workspace, i.Actor); err != nil {
		return inputauthority.Record{}, err
	}
	record, err := a.registry.Approve(i)
	if err != nil {
		return record, err
	}
	if record.State == inputauthority.Applied && record.Proposal.Spec.Kind == inputauthority.Console {
		id := record.Decision.GrantID
		if prior, ok := a.ceilings[id]; ok && prior != record.Proposal.Spec {
			return inputauthority.Record{}, inputauthority.ErrConflict
		}
		a.ceilings[id] = record.Proposal.Spec
	}
	return record, nil
}

func (a *serverGrantAuthority) rejectLocked(i inputauthority.Intent) (inputauthority.Record, error) {
	if err := a.caller(i.Workspace, i.Actor); err != nil {
		return inputauthority.Record{}, err
	}
	return a.registry.Reject(i)
}
func (a *serverGrantAuthority) withdrawLocked(i inputauthority.Intent) (inputauthority.Record, error) {
	if err := a.caller(i.Workspace, i.Actor); err != nil {
		return inputauthority.Record{}, err
	}
	return a.registry.Withdraw(i)
}
func (a *serverGrantAuthority) recordLocked(workspace, id string) (inputauthority.Record, error) {
	if err := a.caller(workspace, a.s.answer.User()); err != nil {
		return inputauthority.Record{}, err
	}
	return a.registry.Record(workspace, id)
}

func (a *serverGrantAuthority) grant(id string, spec inputauthority.Spec) (gate.Grant, error) {
	if a.domain == draftGrantAuthority && !a.draftShape(spec) {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	if a.domain == protectedGrantAuthority && !a.protectedShape(spec) {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	if a.domain == termGrantAuthority && !a.termShape(spec) {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	if a.domain == resultGrantAuthority && !a.resultShape(spec) {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	if a.domain == documentGrantAuthority && !a.documentShape(spec) {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	if (a.domain == financialRequestDomain || a.domain == erpFinalRequestDomain) && !a.financialRequestShape(spec) {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	if (a.domain == financialPermissionDomain || a.domain == erpFinalPermissionDomain) && (a.s == nil || spec.Workspace != a.s.bootWorkspace || spec.Kind != financialpermission.DirectHuman || spec.CeilingID != "") {
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	terms, err := a.domain.terms(spec.Workspace, spec.Capability)
	if err != nil {
		return gate.Grant{}, err
	}
	grant := gate.Grant{ID: id, Terms: terms, Validity: gate.Validity{NotBefore: spec.NotBefore, NotAfter: spec.NotAfter}}
	switch spec.Kind {
	case inputauthority.Console:
		grant.ClientID = spec.Recipient
	case inputauthority.Human:
		grant.SubjectID = spec.Recipient
	case financialpermission.DirectHuman:
		if a.domain != financialPermissionDomain && a.domain != erpFinalPermissionDomain {
			return gate.Grant{}, inputauthority.ErrInvalid
		}
		grant.SubjectID = spec.Recipient
	default:
		return gate.Grant{}, inputauthority.ErrInvalid
	}
	return grant, nil
}
func (a *serverGrantAuthority) sameGrant(g gate.GrantView, id string, spec inputauthority.Spec) bool {
	expected, err := a.grant(id, spec)
	return err == nil && g.ID == expected.ID && g.SubjectID == expected.SubjectID && g.ClientID == expected.ClientID && g.Terms.Action == expected.Terms.Action && g.Terms.Kind == expected.Terms.Kind && g.Terms.Scope.WorkspaceID == spec.Workspace && fullScope(g.Terms) && slices.Equal(g.Terms.Fields, expected.Terms.Fields) && g.Validity.NotBefore.Equal(spec.NotBefore) && g.Validity.NotAfter.Equal(spec.NotAfter)
}

func inputGrantStatus(g gate.GrantView, now time.Time) string {
	if g.Closed {
		return inputauthority.RevokedGrant
	}
	if !now.Before(g.Validity.NotAfter) {
		return inputauthority.ExpiredGrant
	}
	return inputauthority.ActiveGrant
}
