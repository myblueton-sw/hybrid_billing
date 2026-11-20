package server

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type serviceAdmission struct {
	Recorded  string `json:"recorded"`
	Certainty string `json:"certainty"`
	Limit     string `json:"limit"`
}

func (s *Server) serviceAdmission(actor identity.Verifier) (serviceAdmission, error) {
	count, err := s.identity.Workspace().ServiceCount(actor, s.bootWorkspace)
	if err != nil {
		return serviceAdmission{}, err
	}
	return serviceAdmission{Recorded: strconv.Itoa(count), Certainty: "exact", Limit: strconv.Itoa(maxServices)}, nil
}

type serviceDeclarationView struct {
	At                    string            `json:"recorded_at"`
	State                 string            `json:"state"`
	Version               string            `json:"version"`
	Purpose               string            `json:"purpose"`
	Organization          string            `json:"organization"`
	OrganizationName      string            `json:"organization_name"`
	OrganizationNameState string            `json:"organization_name_state"`
	Catalog               string            `json:"catalog_version"`
	Scopes                []serviceScopeRef `json:"scopes"`
}

type serviceCredentialSummary struct {
	State    string `json:"state"`
	Recorded string `json:"recorded"`
	Active   string `json:"active"`
	Expired  string `json:"expired"`
	Revoked  string `json:"revoked"`
	Earliest string `json:"earliest_unexpired_expiry"`
}

type serviceAccountRow struct {
	ID             string                   `json:"id"`
	State          string                   `json:"state"`
	Version        string                   `json:"version"`
	Owner          string                   `json:"owner"`
	OwnerLogin     string                   `json:"owner_login"`
	Creator        string                   `json:"creator"`
	CreatorLogin   string                   `json:"creator_login"`
	Declaration    serviceDeclarationView   `json:"declaration"`
	Credentials    serviceCredentialSummary `json:"credential_summary"`
	RecentUseState string                   `json:"recent_use_state"`
}

func (s *Server) serviceDeclarationView(id string, version uint64) serviceDeclarationView {
	out := serviceDeclarationView{State: "missing", Scopes: []serviceScopeRef{}, OrganizationNameState: "unavailable"}
	record, known := s.serviceDeclaration(s.bootWorkspace, id, version)
	if !known {
		return out
	}
	out.At = record.At.UTC().Format(time.RFC3339Nano)
	out.State = "recorded"
	out.Version = strconv.FormatUint(record.Version, 10)
	out.Purpose = record.Purpose
	out.Organization = record.Organization
	out.Catalog = record.Catalog
	out.Scopes = record.Scopes
	out.OrganizationName = orgName(record.Organization)
	if out.OrganizationName != "" {
		out.OrganizationNameState = "available"
	}
	return out
}

func (s *Server) serviceAccountView(service identity.ServiceInfo, aggregate serviceAggregate) serviceAccountRow {
	row := serviceAccountRow{ID: service.ID, State: serviceState(service.State), Version: strconv.FormatUint(service.Epoch, 10), Owner: service.Owner, OwnerLogin: s.loginOf(service.Owner), Creator: service.Creator, CreatorLogin: s.loginOf(service.Creator), Declaration: s.serviceDeclarationView(service.ID, 0), RecentUseState: "unavailable",
		Credentials: serviceCredentialSummary{State: "exact_at_reference", Recorded: strconv.FormatUint(aggregate.Total, 10), Active: strconv.FormatUint(aggregate.Active, 10), Expired: strconv.FormatUint(aggregate.Expired, 10), Revoked: strconv.FormatUint(aggregate.Revoked, 10)}}
	if !aggregate.Earliest.IsZero() {
		row.Credentials.Earliest = aggregate.Earliest.UTC().Format(time.RFC3339Nano)
	}
	return row
}

func (q serviceQuery) matchesService(row serviceAccountRow) bool {
	if q.State != "all" && q.State != row.State || q.Owner != "" && q.Owner != row.Owner || q.Organization != "" && q.Organization != row.Declaration.Organization || q.Declaration != "all" && q.Declaration != row.Declaration.State {
		return false
	}
	if q.Q != "" && !strings.Contains(strings.ToLower(row.ID+"\n"+row.Declaration.Purpose+"\n"+row.OwnerLogin+"\n"+row.Owner), q.Q) {
		return false
	}
	if q.Audience != "" {
		match := false
		for _, ref := range row.Declaration.Scopes {
			if ref.Audience == q.Audience && (q.Scope == "" || ref.Scope == q.Scope) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func (q serviceQuery) matchesExpiry(aggregate serviceAggregate) bool {
	switch q.Expiry {
	case "none":
		return aggregate.Total == 0
	case "expired":
		return aggregate.Expired > 0
	case "expiring":
		return aggregate.Soon
	case "valid":
		return aggregate.Active > 0
	}
	return true
}

func (aggregate *serviceAggregate) add(info identity.CredentialInfo, at time.Time, warning int) error {
	if aggregate.Total == math.MaxUint64 {
		return identity.ErrUnavailable
	}
	aggregate.Total++
	switch credentialState(info, at) {
	case "revoked":
		aggregate.Revoked++
	case "expired":
		aggregate.Expired++
	default:
		aggregate.Active++
		if aggregate.Earliest.IsZero() || info.Validity.NotAfter.Before(aggregate.Earliest) {
			aggregate.Earliest = info.Validity.NotAfter
		}
		if !info.Validity.NotAfter.After(at.Add(time.Duration(warning) * 24 * time.Hour)) {
			aggregate.Soon = true
		}
	}
	return nil
}

func (walk *serviceWalk) finishService(id string) {
	walk.cursor.AfterService = id
	walk.cursor.PendingService = ""
	walk.cursor.AfterCredential = ""
	walk.cursor.Aggregate = serviceAggregate{}
	walk.cursor.Completed++
}

func (walk *serviceWalk) accountCredentials(id string) (bool, error) {
	s := walk.s
	if s.faulty("credentials") {
		return false, identity.ErrUnavailable
	}
	for walk.remaining() > 0 {
		limit := min(walk.remaining(), s.cfg.Identity.PageSize)
		batch, err := s.identity.Workspace().CredentialPage(walk.actor, s.bootWorkspace, id, walk.cursor.AfterCredential, limit)
		if err != nil {
			return false, err
		}
		for _, credential := range batch {
			walk.scanned++
			if err := walk.cursor.Aggregate.add(credential, walk.cursor.Reference, s.cfg.Access.CredentialWarningDays); err != nil {
				return false, err
			}
			walk.cursor.AfterCredential = credential.ID
		}
		if len(batch) < limit {
			return true, nil
		}
	}
	return false, nil
}

func (walk *serviceWalk) accounts() ([]serviceAccountRow, bool, error) {
	s := walk.s
	rows := []serviceAccountRow{}
	if s.faulty("services") || s.faulty("service-declaration") {
		return nil, false, identity.ErrUnavailable
	}
	total, err := s.identity.Workspace().ServiceCount(walk.actor, s.bootWorkspace)
	if err != nil {
		return nil, false, err
	}
	walk.total = uint64(total)
	if walk.cursor.Completed > walk.total {
		return nil, false, errServiceViewChanged
	}
	for len(rows) < walk.query.Limit && walk.remaining() > 0 && walk.cursor.Completed < walk.total {
		service, err := walk.nextService()
		if err != nil {
			return nil, false, err
		}
		walk.scanned++
		row := s.serviceAccountView(service, serviceAggregate{})
		if !walk.query.matchesService(row) {
			walk.finishService(service.ID)
			continue
		}
		complete, err := walk.accountCredentials(service.ID)
		if err != nil {
			return nil, false, err
		}
		if !complete {
			break
		}
		aggregate := walk.cursor.Aggregate
		walk.finishService(service.ID)
		if !walk.query.matchesExpiry(aggregate) {
			continue
		}
		row = s.serviceAccountView(service, aggregate)
		rows = append(rows, row)
		if err := walk.matched(); err != nil {
			return nil, false, err
		}
	}
	return rows, walk.cursor.Completed < walk.total, nil
}

func (s *Server) queryServiceAccounts(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	walk, ok := s.beginServiceQuery(w, r, "accounts", "")
	if !ok {
		return
	}
	rows, more, err := walk.accounts()
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	writeServiceQueryPage(w, r, walk, rows, more, "service_id_asc")
}
