package server

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

type serviceDetailRead struct {
	user  string
	s     *Server
	actor identity.Verifier
	area  string
	basis string
	now   time.Time
}

type serviceEligibility struct {
	State           string `json:"state"`
	CanIssue        bool   `json:"can_issue"`
	CanReinstate    bool   `json:"can_reinstate"`
	CanDeclare      bool   `json:"can_declare"`
	CanSuspend      bool   `json:"can_suspend"`
	CanRetire       bool   `json:"can_retire"`
	StepUpState     string `json:"step_up_state"`
	ExecutionState  string `json:"execution_state"`
	LiveCredentials string `json:"live_credentials"`
	CredentialLimit string `json:"credential_limit"`
	OperatorBlocked bool   `json:"operator_blocked"`
}

type serviceAccountDetail struct {
	Workspace      string             `json:"workspace"`
	Admission      serviceAdmission   `json:"admission"`
	Row            serviceAccountRow  `json:"row"`
	Eligibility    serviceEligibility `json:"eligibility"`
	Revision       string             `json:"revision"`
	Observed       string             `json:"observed_at"`
	Settings       []settingRow       `json:"settings"`
	Limits         serviceLimits      `json:"limits"`
	CatalogVersion string             `json:"catalog_version"`
}

func (s *Server) beginServiceDetail(w http.ResponseWriter, r *http.Request, area string, ids ...string) (*serviceDetailRead, bool) {
	request := r.Header.Get("X-Request-Id")
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return nil, false
	}
	if answer.Workspace() != s.bootWorkspace {
		failCode(w, commands.PermissionDenied, request)
		return nil, false
	}
	if len(r.URL.RawQuery) > 8192 {
		invalidFields(w, request, "query")
		return nil, false
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		invalidFields(w, request, "query")
		return nil, false
	}
	for name, list := range values {
		if name != "workspace" || len(list) != 1 || list[0] != s.bootWorkspace {
			invalidFields(w, request, name)
			return nil, false
		}
	}
	for _, id := range ids {
		if !validWorkspace(id) {
			failCode(w, commands.PermissionDenied, request)
			return nil, false
		}
	}
	basis, err := s.serviceBasis(found.verifier, area)
	if err != nil {
		serviceQueryFailure(w, r, err)
		return nil, false
	}
	return &serviceDetailRead{user: answer.User(), s: s, actor: found.verifier, area: area, basis: basis, now: s.clock()}, true
}

func (read *serviceDetailRead) write(w http.ResponseWriter, r *http.Request, value any) {
	if _, _, ok := read.s.admitted(w, r); !ok {
		return
	}
	basis, err := read.s.serviceBasis(read.actor, read.area)
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	if basis != read.basis {
		failRead(w, commands.RevisionConflict, r.Header.Get("X-Request-Id"))
		return
	}
	writeJSON(w, http.StatusOK, r.Header.Get("X-Request-Id"), value)
}

func serviceKnownFailure(w http.ResponseWriter, r *http.Request, err error) {
	if identity.ReasonOf(err) == identity.ReasonUnknown || errors.Is(err, organizations.ErrNotFound) {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	serviceQueryFailure(w, r, err)
}

type serviceCredentialEligibility struct {
	CanRevoke      bool   `json:"can_revoke"`
	StepUpState    string `json:"step_up_state"`
	ExecutionState string `json:"execution_state"`
}

func (s *Server) serviceStepUpState(actor identity.Verifier, user, command, target string) (bool, string) {
	switch s.stepUpCheck(actor, user, command, target) {
	case "":
		return true, "ready"
	case CodeReauthentication:
		return false, "reauthentication_required"
	case commands.PrerequisiteFailed:
		return false, "prerequisite_failed"
	default:
		return false, "unavailable"
	}
}

func (s *Server) currentCredentialEligibility(actor identity.Verifier, user, id string, info identity.CredentialInfo) serviceCredentialEligibility {
	ready, state := s.serviceStepUpState(actor, user, "service.revoke_credential", id)
	execution := "ready"
	allowed := len(s.stranded) == 0
	if !allowed {
		execution = "dependency_unavailable"
	}
	return serviceCredentialEligibility{CanRevoke: ready && allowed && !info.Revoked, StepUpState: state, ExecutionState: execution}
}

func (s *Server) currentServiceEligibility(actor identity.Verifier, user string, service identity.ServiceInfo, at time.Time) serviceEligibility {
	out := serviceEligibility{State: "ineligible", StepUpState: "ready", ExecutionState: "ready"}
	ready, stepState := s.serviceStepUpState(actor, user, "service.issue_credential", service.ID)
	out.StepUpState = stepState
	execution := len(s.stranded) == 0
	if !execution {
		out.ExecutionState = "dependency_unavailable"
	} else if s.held() {
		out.ExecutionState = "trust_hold"
	}
	out.CanSuspend = execution && ready && service.State == identity.Active && service.Epoch < math.MaxUint64
	out.CanRetire = execution && ready && service.State != identity.Departed && service.Epoch < math.MaxUint64
	controls, err := s.identity.Workspace().ServiceControls(actor, s.bootWorkspace, service.ID)
	if err != nil {
		out.State = "unavailable"
		return out
	}
	out.LiveCredentials = strconv.Itoa(controls.LiveCredentials)
	out.CredentialLimit = strconv.Itoa(controls.CredentialLimit)
	out.OperatorBlocked = controls.OperatorBlocked
	owner, err := s.eligibleServiceOwner(actor, service.Owner)
	if err != nil {
		out.State = "unavailable"
		return out
	}
	if !owner || service.State == identity.Departed {
		return out
	}
	if s.faulty("service-declaration") {
		out.State = "unavailable"
		return out
	}
	expanding := execution && ready && !s.held()
	record, known := s.serviceDeclaration(s.bootWorkspace, service.ID, 0)
	versionRoom := known && record.Version < math.MaxUint64 || !known && len(s.serviceDeclarations) < maxServices
	out.CanDeclare = expanding && versionRoom
	if !known {
		return out
	}
	if err := s.checkServiceDeclaration(record, at); err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) && failure.Code == commands.DependencyUnavailable {
			out.State = "unavailable"
		}
		return out
	}
	out.State = "eligible"
	out.CanIssue = expanding && service.State == identity.Active && controls.CanAllocateCredential && controls.LiveCredentials < controls.CredentialLimit && s.cfg.Identity.CredentialValidity >= 24*time.Hour
	out.CanReinstate = expanding && service.State == identity.Suspended && !controls.OperatorBlocked && controls.CanTransition
	return out
}

func (s *Server) queryServiceLimits() serviceLimits {
	limits := serviceLimits{CommandValueBytes: strconv.Itoa(s.cfg.Commands.Value), RationaleBytes: min(maxRationale, s.cfg.Commands.Rationale), BodyBytes: maxBodyBytes, CredentialDays: int(s.cfg.Identity.CredentialValidity / (24 * time.Hour)), CredentialsPerService: strconv.Itoa(s.cfg.Identity.CredentialsPerServiceUser), Services: maxServices, ReplaySeconds: int(s.cfg.Commands.Replay / time.Second)}
	limits.PickupMinutes, limits.OverlapMinutes, limits.OverlapMaxMinutes = s.overlapBounds()
	return limits
}

func (s *Server) readServiceAccount(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	id := r.PathValue("id")
	read, ok := s.beginServiceDetail(w, r, "service_detail", id)
	if !ok {
		return
	}
	if s.faulty("services") {
		serviceQueryFailure(w, r, identity.ErrUnavailable)
		return
	}
	service, err := s.identity.Workspace().ServiceUser(read.actor, s.bootWorkspace, id)
	if err != nil {
		serviceKnownFailure(w, r, err)
		return
	}
	admission, err := s.serviceAdmission(read.actor)
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	row := s.serviceAccountView(service, serviceAggregate{})
	row.Credentials = serviceCredentialSummary{State: "not_evaluated"}
	if s.faulty("service-declaration") {
		row.Declaration = serviceDeclarationView{State: "unavailable", OrganizationNameState: "unavailable", Scopes: []serviceScopeRef{}}
	}
	read.write(w, r, serviceAccountDetail{Workspace: s.bootWorkspace, Admission: admission, Row: row, Eligibility: s.currentServiceEligibility(read.actor, read.user, service, read.now), Revision: s.Revision(), Observed: read.now.UTC().Format(time.RFC3339Nano), Settings: append(s.rowsOf(serviceQuerySettings), s.rowsOf(serviceDeclarationSettings)...), Limits: s.queryServiceLimits(), CatalogVersion: s.serviceCatalog.Version()})
}

func (s *Server) readServiceCredential(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	id := r.PathValue("id")
	read, ok := s.beginServiceDetail(w, r, "credential_detail", id)
	if !ok {
		return
	}
	if s.faulty("credentials") || s.faulty("services") {
		serviceQueryFailure(w, r, identity.ErrUnavailable)
		return
	}
	serviceID, info, err := s.identity.Workspace().Credential(read.actor, s.bootWorkspace, id)
	if err != nil {
		serviceKnownFailure(w, r, err)
		return
	}
	service, err := s.identity.Workspace().ServiceUser(read.actor, s.bootWorkspace, serviceID)
	if err != nil {
		serviceKnownFailure(w, r, err)
		return
	}
	read.write(w, r, map[string]any{"workspace": s.bootWorkspace, "eligibility": s.currentCredentialEligibility(read.actor, read.user, id, info), "row": s.serviceCredentialView(read.actor, service, info, read.now), "revision": s.Revision(), "observed_at": read.now.UTC().Format(time.RFC3339Nano), "limits": s.queryServiceLimits()})
}

func (s *Server) readServiceOwner(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	id := r.PathValue("id")
	read, ok := s.beginServiceDetail(w, r, "owner_detail", id)
	if !ok {
		return
	}
	if s.faulty("roster") || s.faulty("people") {
		serviceQueryFailure(w, r, identity.ErrUnavailable)
		return
	}
	people, err := s.identity.Workspace().People(read.actor, s.bootWorkspace, []string{id})
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	if len(people) != 1 || people[0].User.Kind != identity.Human || people[0].Member == nil {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	read.write(w, r, map[string]any{"workspace": s.bootWorkspace, "row": ownerChoice(s, people[0], read.now), "revision": s.Revision(), "observed_at": read.now.UTC().Format(time.RFC3339Nano)})
}

func (s *Server) readServiceOrganization(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	id := r.PathValue("id")
	read, ok := s.beginServiceDetail(w, r, "organization_detail", id)
	if !ok {
		return
	}
	party, _, err := s.orgs.Party(s.bootWorkspace, id)
	if err != nil {
		serviceKnownFailure(w, r, err)
		return
	}
	state := "eligible"
	if _, _, err = s.orgs.EligibleParty(s.bootWorkspace, id); err != nil {
		if errors.Is(err, organizations.ErrSuspended) {
			state = "ineligible"
		} else {
			state = "unavailable"
		}
	}
	read.write(w, r, map[string]any{"workspace": s.bootWorkspace, "row": organizationChoice(party), "eligibility": state, "revision": s.Revision(), "observed_at": read.now.UTC().Format(time.RFC3339Nano)})
}

func (s *Server) readServiceScope(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	audienceID, scopeID := r.PathValue("id"), r.PathValue("scope")
	read, ok := s.beginServiceDetail(w, r, "scope_detail", audienceID, scopeID)
	if !ok {
		return
	}
	audience, err := s.serviceCatalog.Audience(s.bootWorkspace, audienceID)
	if err != nil {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	scope, err := s.serviceCatalog.Scope(s.bootWorkspace, audienceID, scopeID)
	if err != nil {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	state := "eligible"
	if err = s.checkServiceTerms(scope.Terms, read.now); err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) && failure.Code == commands.InvalidRequest {
			state = "ineligible"
		} else {
			state = "unavailable"
		}
	}
	read.write(w, r, map[string]any{"workspace": s.bootWorkspace, "row": scopeChoice(audience, scope), "eligibility": state, "catalog_version": s.serviceCatalog.Version(), "revision": s.Revision(), "observed_at": read.now.UTC().Format(time.RFC3339Nano)})
}
