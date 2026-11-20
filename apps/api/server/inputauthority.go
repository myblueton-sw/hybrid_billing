package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type serverInputAuthority struct {
	s        *Server
	registry *inputauthority.Registry
	bindings map[string]inputauthority.Spec
	ceilings map[string]inputauthority.Spec
}

func newServerInputAuthority(s *Server, limits inputauthority.Limits) (*serverInputAuthority, error) {
	if s == nil || s.cfg.Seed != SeedSample {
		return nil, inputauthority.ErrInvalid
	}
	a := &serverInputAuthority{s: s, bindings: make(map[string]inputauthority.Spec), ceilings: make(map[string]inputauthority.Spec)}
	var err error
	a.registry, err = inputauthority.NewRegistry(limits, a, a, a, s.clock)
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

func (a *serverInputAuthority) caller(workspace, actor string) error {
	if workspace != a.s.bootWorkspace || actor == "" || actor != a.s.answer.User() || a.s.answer.Workspace() != workspace {
		return inputauthority.ErrDenied
	}
	request := gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: workspace}}
	decision := a.s.gate.Decide(a.s.answer, ConsoleClient, request)
	if !decision.Allowed {
		if decision.Code == gate.AuthorityUnavailable {
			return gate.ErrUnavailable
		}
		return inputauthority.ErrDenied
	}
	return nil
}

func (a *serverInputAuthority) proposeLocked(p inputauthority.Proposal) (inputauthority.Record, error) {
	if err := a.caller(p.Spec.Workspace, p.Proposer); err != nil {
		return inputauthority.Record{}, err
	}
	record, err := a.registry.Propose(p)
	if err != nil {
		return inputauthority.Record{}, err
	}
	id := "input-grant-" + record.Digest
	if prior, ok := a.bindings[id]; ok && prior != record.Proposal.Spec {
		return inputauthority.Record{}, inputauthority.ErrConflict
	}
	a.bindings[id] = record.Proposal.Spec
	return record, nil
}

func (a *serverInputAuthority) approveLocked(i inputauthority.Intent) (inputauthority.Record, error) {
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

func (a *serverInputAuthority) rejectLocked(i inputauthority.Intent) (inputauthority.Record, error) {
	if err := a.caller(i.Workspace, i.Actor); err != nil {
		return inputauthority.Record{}, err
	}
	return a.registry.Reject(i)
}
func (a *serverInputAuthority) withdrawLocked(i inputauthority.Intent) (inputauthority.Record, error) {
	if err := a.caller(i.Workspace, i.Actor); err != nil {
		return inputauthority.Record{}, err
	}
	return a.registry.Withdraw(i)
}
func (a *serverInputAuthority) recordLocked(workspace, id string) (inputauthority.Record, error) {
	if err := a.caller(workspace, a.s.answer.User()); err != nil {
		return inputauthority.Record{}, err
	}
	return a.registry.Record(workspace, id)
}

func inputAuthorityTerms(workspace string) gate.Terms {
	return gate.Terms{Action: gate.ConfirmInput, Kind: inputauthority.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}
}
func inputAuthorityGrant(id string, spec inputauthority.Spec) gate.Grant {
	grant := gate.Grant{ID: id, Terms: inputAuthorityTerms(spec.Workspace), Validity: gate.Validity{NotBefore: spec.NotBefore, NotAfter: spec.NotAfter}}
	if spec.Kind == inputauthority.Console {
		grant.ClientID = spec.Recipient
	} else {
		grant.SubjectID = spec.Recipient
	}
	return grant
}

func sameInputAuthorityGrant(g gate.GrantView, id string, spec inputauthority.Spec) bool {
	expected := inputAuthorityGrant(id, spec)
	return g.ID == expected.ID && g.SubjectID == expected.SubjectID && g.ClientID == expected.ClientID && g.Terms.Action == expected.Terms.Action && g.Terms.Kind == expected.Terms.Kind && g.Terms.Scope.WorkspaceID == spec.Workspace && fullScope(g.Terms) && len(g.Terms.Fields) == 0 && g.Validity.NotBefore.Equal(spec.NotBefore) && g.Validity.NotAfter.Equal(spec.NotAfter)
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
