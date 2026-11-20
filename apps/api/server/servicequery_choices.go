package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/servicecatalog"
)

type serviceOwnerRow struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	Role     string `json:"role"`
	From     string `json:"from"`
	Until    string `json:"until"`
	Eligible bool   `json:"eligible_at_reference"`
}

type serviceOrganizationRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NameState string `json:"name_state"`
	Kind      string `json:"kind"`
	Parent    string `json:"parent"`
}

type serviceAudienceRow struct {
	ID         string `json:"id"`
	Resource   string `json:"resource"`
	ScopeCount int    `json:"configured_scope_count"`
}

type serviceScopeRow struct {
	Workspace    string                   `json:"workspace"`
	ID           string                   `json:"id"`
	Audience     string                   `json:"audience"`
	Resource     string                   `json:"resource"`
	Restrictions servicecatalog.ScopeSpec `json:"restrictions"`
}

func memberEligibleAt(person identity.Person, at time.Time) bool {
	member := person.Member
	return person.User.Kind == identity.Human && person.User.State == identity.Active && member != nil && member.State == identity.MembershipActive && !at.Before(member.From) && (member.Until.IsZero() || at.Before(member.Until))
}

func ownerChoice(s *Server, person identity.Person, reference time.Time) serviceOwnerRow {
	row := serviceOwnerRow{ID: person.User.ID, Login: s.loginOf(person.User.ID), Eligible: memberEligibleAt(person, reference)}
	if person.Member != nil {
		row.Role = string(person.Member.Role)
		if !person.Member.From.IsZero() {
			row.From = person.Member.From.UTC().Format(time.RFC3339Nano)
		}
		if !person.Member.Until.IsZero() {
			row.Until = person.Member.Until.UTC().Format(time.RFC3339Nano)
		}
	}
	return row
}

func (q serviceQuery) textMatches(values ...string) bool {
	return q.Q == "" || strings.Contains(strings.ToLower(strings.Join(values, "\n")), q.Q)
}

func (walk *serviceWalk) owners() ([]serviceOwnerRow, bool, error) {
	s := walk.s
	rows := []serviceOwnerRow{}
	more := true
	if s.faulty("roster") || s.faulty("people") {
		return nil, false, identity.ErrUnavailable
	}
	for more && len(rows) < walk.query.Limit && walk.remaining() > 0 {
		limit := min(walk.remaining(), s.cfg.Identity.PageSize, walk.query.Limit-len(rows))
		batch, err := s.identity.Workspace().MemberPositions(walk.actor, s.bootWorkspace, walk.cursor.After, limit)
		if err != nil {
			return nil, false, err
		}
		walk.scanned += batch.Scanned
		walk.cursor.After = batch.After
		more = batch.More
		for _, person := range batch.Rows {
			row := ownerChoice(s, person, walk.cursor.Reference)
			if !row.Eligible || !walk.query.textMatches(row.ID, row.Login) {
				continue
			}
			rows = append(rows, row)
			if err := walk.matched(); err != nil {
				return nil, false, err
			}
		}
	}
	return rows, more, nil
}

func organizationChoice(party organizations.Party) serviceOrganizationRow {
	row := serviceOrganizationRow{ID: party.ID, Name: orgName(party.ID), Kind: string(party.Kind), Parent: party.ParentID, NameState: "unavailable"}
	if row.Name != "" {
		row.NameState = "available"
	}
	return row
}

func (walk *serviceWalk) organizations() ([]serviceOrganizationRow, bool, error) {
	s := walk.s
	rows := []serviceOrganizationRow{}
	more := true
	for more && len(rows) < walk.query.Limit && walk.remaining() > 0 {
		limit := min(walk.remaining(), walk.query.Limit-len(rows))
		batch, err := s.orgs.PartyPage(s.bootWorkspace, walk.cursor.After, limit)
		if err != nil {
			return nil, false, err
		}
		more = len(batch) == limit
		for _, party := range batch {
			walk.scanned++
			walk.cursor.After = party.ID
			eligible, _, err := s.orgs.EligibleParty(s.bootWorkspace, party.ID)
			if errors.Is(err, organizations.ErrSuspended) {
				continue
			}
			if err != nil {
				return nil, false, err
			}
			row := organizationChoice(eligible)
			if !walk.query.textMatches(row.ID, row.Name) {
				continue
			}
			rows = append(rows, row)
			if err := walk.matched(); err != nil {
				return nil, false, err
			}
		}
	}
	return rows, more, nil
}

func (walk *serviceWalk) audiences() ([]serviceAudienceRow, bool, error) {
	s := walk.s
	rows := []serviceAudienceRow{}
	more := true
	for more && len(rows) < walk.query.Limit && walk.remaining() > 0 {
		limit := min(walk.remaining(), walk.query.Limit-len(rows))
		batch, next, err := s.serviceCatalog.AudiencePage(s.bootWorkspace, walk.cursor.After, limit)
		if err != nil {
			return nil, false, err
		}
		more = next != ""
		for _, audience := range batch {
			walk.scanned++
			walk.cursor.After = audience.ID
			if !walk.query.textMatches(audience.ID, audience.Resource) {
				continue
			}
			rows = append(rows, serviceAudienceRow{audience.ID, audience.Resource, audience.ScopeCount})
			if err := walk.matched(); err != nil {
				return nil, false, err
			}
		}
	}
	return rows, more, nil
}

func scopeSelector(selector gate.Selector) servicecatalog.SelectorSpec {
	if selector.Any() {
		return servicecatalog.SelectorSpec{Any: true}
	}
	return servicecatalog.SelectorSpec{ID: selector.ID()}
}

func scopeMonth(month gate.Month) string {
	return fmt.Sprintf("%04d-%02d", int(month)/12, int(month)%12+1)
}

func scopeChoice(audience servicecatalog.Audience, scope servicecatalog.Scope) serviceScopeRow {
	terms := scope.Terms
	period := servicecatalog.PeriodSpec{Any: true}
	if !terms.Period.Any() {
		period = servicecatalog.PeriodSpec{From: scopeMonth(terms.Period.From()), Until: scopeMonth(terms.Period.To())}
	}
	spec := servicecatalog.ScopeSpec{ID: scope.ID, Action: terms.Action, Kind: terms.Kind, Fields: append([]string{}, terms.Fields...), Customer: scopeSelector(terms.Scope.Customer), LegalEntity: scopeSelector(terms.Scope.LegalEntity), Contract: scopeSelector(terms.Scope.Contract), Period: period}
	return serviceScopeRow{Workspace: audience.Workspace, ID: scope.ID, Audience: audience.ID, Resource: audience.Resource, Restrictions: spec}
}

func (walk *serviceWalk) scopes(audience servicecatalog.Audience) ([]serviceScopeRow, bool, error) {
	s := walk.s
	rows := []serviceScopeRow{}
	more := true
	for more && len(rows) < walk.query.Limit {
		if !walk.reserve(8) {
			break
		}
		batch, next, err := s.serviceCatalog.ScopePage(s.bootWorkspace, audience.ID, walk.cursor.After, 1)
		if err != nil {
			return nil, false, err
		}
		more = next != ""
		for _, scope := range batch {
			walk.cursor.After = scope.ID
			if !walk.query.textMatches(scope.ID) {
				continue
			}
			if err := s.checkServiceTerms(scope.Terms, walk.cursor.Reference); err != nil {
				var failure *commands.Failure
				if errors.As(err, &failure) && failure.Code == commands.InvalidRequest {
					continue
				}
				return nil, false, err
			}
			rows = append(rows, scopeChoice(audience, scope))
			if err := walk.matched(); err != nil {
				return nil, false, err
			}
		}
	}
	return rows, more, nil
}

func (s *Server) queryServiceOwners(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	walk, ok := s.beginServiceQuery(w, r, "owners", "")
	if !ok {
		return
	}
	rows, more, err := walk.owners()
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	writeServiceQueryPage(w, r, walk, rows, more, "owner_id_asc")
}

func (s *Server) queryServiceOrganizations(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	walk, ok := s.beginServiceQuery(w, r, "organizations", "")
	if !ok {
		return
	}
	rows, more, err := walk.organizations()
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	writeServiceQueryPage(w, r, walk, rows, more, "organization_id_asc")
}

func (s *Server) queryServiceAudiences(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	walk, ok := s.beginServiceQuery(w, r, "audiences", "")
	if !ok {
		return
	}
	rows, more, err := walk.audiences()
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	writeServiceQueryPage(w, r, walk, rows, more, "audience_id_asc")
}

func (s *Server) queryServiceScopes(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	target := r.PathValue("id")
	walk, ok := s.beginServiceQuery(w, r, "scopes", target)
	if !ok {
		return
	}
	audience, err := s.serviceCatalog.Audience(s.bootWorkspace, target)
	if err != nil {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	rows, more, err := walk.scopes(audience)
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	writeServiceQueryPage(w, r, walk, rows, more, "scope_id_asc")
}
