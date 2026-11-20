package server

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

type serviceDeclarationKey struct{ workspace, service string }

type serviceScopeRef struct {
	Audience string `json:"audience"`
	Scope    string `json:"scope"`
}

type serviceDeclaration struct {
	Version      uint64
	Purpose      string
	Organization string
	Catalog      string
	Scopes       []serviceScopeRef
	At           time.Time
}

type preparedServiceDeclaration struct {
	key     serviceDeclarationKey
	history []serviceDeclaration
}

func parseServiceScopes(raw string, maxCount, maxBytes int) ([]serviceScopeRef, bool) {
	if len(raw) == 0 || len(raw) > maxBytes {
		return nil, false
	}
	d := json.NewDecoder(strings.NewReader(raw))
	if expectDelim(d, '[') != nil {
		return nil, false
	}
	rows := []serviceScopeRef{}
	seen := map[serviceScopeRef]bool{}
	for d.More() {
		if len(rows) >= maxCount || expectDelim(d, '{') != nil {
			return nil, false
		}
		row := serviceScopeRef{}
		fields := map[string]bool{}
		for d.More() {
			name, err := stringValue(d)
			if err != nil || fields[name] || name != "audience" && name != "scope" {
				return nil, false
			}
			value, err := stringValue(d)
			if err != nil || !validWorkspace(value) {
				return nil, false
			}
			fields[name] = true
			if name == "audience" {
				row.Audience = value
			} else {
				row.Scope = value
			}
		}
		if expectDelim(d, '}') != nil || !fields["audience"] || !fields["scope"] || seen[row] {
			return nil, false
		}
		seen[row] = true
		rows = append(rows, row)
	}
	if expectDelim(d, ']') != nil || len(rows) == 0 {
		return nil, false
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, false
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		return a.Audience < b.Audience || a.Audience == b.Audience && a.Scope < b.Scope
	})
	return rows, true
}

func validServicePurpose(value string, maxBytes int) bool {
	return value != "" && len(value) <= maxBytes && strings.TrimSpace(value) == value && utf8.ValidString(value) && strings.IndexFunc(value, func(c rune) bool {
		return unicode.IsControl(c) || c >= 0x202A && c <= 0x202E || c >= 0x2066 && c <= 0x2069 || c == 0x200E || c == 0x200F
	}) < 0
}

func validCatalogVersion(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (s *Server) eligibleServiceOwner(owner string) (bool, error) {
	if s.faulty("roster") {
		return false, identity.ErrUnavailable
	}
	rows, err := s.identity.Workspace().People(s.caller, s.bootWorkspace, []string{owner})
	if err != nil {
		return false, err
	}
	return len(rows) == 1 && rows[0].User.Kind == identity.Human && rows[0].User.State == identity.Active && rows[0].Member != nil && rows[0].Member.Effective, nil
}

func (s *Server) checkServiceParty(id string, legal bool) error {
	party, _, err := s.orgs.EligibleParty(s.bootWorkspace, id)
	if err != nil || legal && party.Kind != organizations.LegalEntity {
		return commands.Fail(commands.InvalidRequest, "scopes")
	}
	return nil
}

func (s *Server) checkServiceTerms(terms gate.Terms, now time.Time) error {
	if !s.gate.DescribesServiceTerms(terms) || terms.Scope.WorkspaceID != s.bootWorkspace {
		return commands.Fail(commands.InvalidRequest, "scopes")
	}
	for _, selector := range []gate.Selector{terms.Scope.Customer, terms.Scope.LegalEntity} {
		if !selector.Any() {
			if err := s.checkServiceParty(selector.ID(), true); err != nil {
				return err
			}
		}
	}
	if terms.Scope.Contract.Any() {
		return nil
	}
	if s.faulty("contract-read") {
		return commands.Fail(commands.DependencyUnavailable)
	}
	id := terms.Scope.Contract.ID()
	if s.contracts.get(s.bootWorkspace, id) == nil {
		return commands.Fail(commands.InvalidRequest, "scopes")
	}
	sequence, err := s.lifecycles.Sequence(s.bootWorkspace, id)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	position, err := s.lifecycles.StateOn(s.bootWorkspace, id, sequence, contractToday(now))
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if position.State != contracts.Active {
		return commands.Fail(commands.InvalidRequest, "scopes")
	}
	revision, err := s.orgs.ContractRevision(s.bootWorkspace, id, position.Revision)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if !terms.Scope.Customer.Any() && terms.Scope.Customer.ID() != revision.CustomerID {
		return commands.Fail(commands.InvalidRequest, "scopes")
	}
	if err := s.checkServiceParty(revision.CustomerID, true); err != nil {
		return err
	}
	if revision.SellerID != "" {
		return s.checkServiceParty(revision.SellerID, true)
	}
	return nil
}

func (s *Server) checkServiceDeclaration(value serviceDeclaration, now time.Time) error {
	if s.faulty("service-declaration") || s.faulty("organizations") {
		return commands.Fail(commands.DependencyUnavailable)
	}
	if value.Catalog != s.serviceCatalog.Version() {
		return commands.Fail(commands.RevisionConflict, "catalog_version")
	}
	if _, _, err := s.orgs.EligibleParty(s.bootWorkspace, value.Organization); err != nil {
		return commands.Fail(commands.InvalidRequest, "organization")
	}
	for _, ref := range value.Scopes {
		scope, err := s.serviceCatalog.Scope(s.bootWorkspace, ref.Audience, ref.Scope)
		if err != nil {
			return commands.Fail(commands.InvalidRequest, "scopes")
		}
		if err := s.checkServiceTerms(scope.Terms, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) serviceDeclaration(workspace, service string, version uint64) (serviceDeclaration, bool) {
	history := s.serviceDeclarations[serviceDeclarationKey{workspace, service}]
	if len(history) == 0 {
		return serviceDeclaration{}, false
	}
	if version == 0 {
		version = history[len(history)-1].Version
	}
	for _, record := range history {
		if record.Version == version {
			record.Scopes = append([]serviceScopeRef{}, record.Scopes...)
			return record, true
		}
	}
	return serviceDeclaration{}, false
}

func (s *Server) prepareServiceDeclaration(service string, fields map[string]string, expected uint64, now time.Time) (preparedServiceDeclaration, error) {
	key := serviceDeclarationKey{s.bootWorkspace, service}
	history := s.serviceDeclarations[key]
	current := uint64(0)
	if len(history) > 0 {
		current = history[len(history)-1].Version
	}
	if current != expected {
		return preparedServiceDeclaration{}, commands.Fail(commands.RevisionConflict, "declaration_version")
	}
	if current == math.MaxUint64 || len(history) == 0 && len(s.serviceDeclarations) >= maxServices {
		return preparedServiceDeclaration{}, commands.Fail(commands.PrerequisiteFailed)
	}
	purpose := fields["purpose"]
	if !validServicePurpose(purpose, s.cfg.Access.ServicePurposeBytes) {
		return preparedServiceDeclaration{}, commands.Fail(commands.InvalidRequest, "purpose")
	}
	if !validWorkspace(fields["organization"]) {
		return preparedServiceDeclaration{}, commands.Fail(commands.InvalidRequest, "organization")
	}
	if !validCatalogVersion(fields["catalog_version"]) {
		return preparedServiceDeclaration{}, commands.Fail(commands.InvalidRequest, "catalog_version")
	}
	refs, valid := parseServiceScopes(fields["scopes"], s.cfg.Access.ServiceDeclarationScopes, s.cfg.Commands.Value)
	if !valid {
		return preparedServiceDeclaration{}, commands.Fail(commands.InvalidRequest, "scopes")
	}
	record := serviceDeclaration{Version: current + 1, Purpose: purpose, Organization: fields["organization"], Catalog: fields["catalog_version"], Scopes: refs, At: now.UTC()}
	if err := s.checkServiceDeclaration(record, now); err != nil {
		return preparedServiceDeclaration{}, err
	}
	keep := min(len(history), s.cfg.Access.ServiceDeclarationVersions-1)
	next := make([]serviceDeclaration, keep+1)
	copy(next, history[len(history)-keep:])
	next[keep] = record
	return preparedServiceDeclaration{key: key, history: next}, nil
}

func (s *Server) requireServiceDeclaration(svc identity.ServiceInfo, now time.Time) error {
	eligible, err := s.eligibleServiceOwner(svc.Owner)
	if err != nil {
		return mapIdentity(err)
	}
	if !eligible {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	record, known := s.serviceDeclaration(s.bootWorkspace, svc.ID, 0)
	if !known {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	if err := s.checkServiceDeclaration(record, now); err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) && failure.Code == commands.DependencyUnavailable {
			return err
		}
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return nil
}

func declarationFields() []commands.Field {
	return []commands.Field{{Name: "purpose", Type: commands.Text, Required: true}, {Name: "organization", Type: commands.ID, Required: true}, {Name: "catalog_version", Type: commands.Text, Required: true}, {Name: "scopes", Type: commands.Text, Required: true}}
}

func (h serviceCommand) declare(ctx commands.Context, now time.Time) (string, map[string]string, error) {
	s := h.s
	svc, found, err := s.findService(ctx.Target)
	if err != nil {
		return "", nil, err
	}
	if !found || svc.State == identity.Departed {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	eligible, err := s.eligibleServiceOwner(svc.Owner)
	if err != nil {
		return "", nil, err
	}
	if !eligible {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	raw := ctx.Fields["declaration_version"]
	version, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || strconv.FormatUint(version, 10) != raw {
		return "", nil, commands.Fail(commands.InvalidRequest, "declaration_version")
	}
	prepared, err := s.prepareServiceDeclaration(svc.ID, ctx.Fields, version, now)
	if err != nil {
		return "", nil, err
	}
	s.serviceDeclarations[prepared.key] = prepared.history
	return "service_declared", map[string]string{"declaration_version": strconv.FormatUint(prepared.history[len(prepared.history)-1].Version, 10)}, nil
}
