package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type serviceCredentialRow struct {
	credentialRow
	Service      string `json:"service"`
	ServiceState string `json:"service_state"`
	CurrentState string `json:"current_state"`
	Observed     string `json:"observed_at"`
}

func (walk *serviceWalk) nextService() (identity.ServiceInfo, error) {
	s := walk.s
	if walk.cursor.PendingService != "" {
		return s.identity.Workspace().ServiceUser(walk.actor, s.bootWorkspace, walk.cursor.PendingService)
	}
	if walk.query.Service != "" {
		row, err := s.identity.Workspace().ServiceUser(walk.actor, s.bootWorkspace, walk.query.Service)
		if err == nil {
			walk.cursor.PendingService = row.ID
		}
		return row, err
	}
	batch, err := s.identity.Workspace().ServiceUsers(walk.actor, s.bootWorkspace, walk.cursor.AfterService, 1)
	if err != nil {
		return identity.ServiceInfo{}, err
	}
	if len(batch) != 1 {
		return identity.ServiceInfo{}, errServiceViewChanged
	}
	walk.cursor.PendingService = batch[0].ID
	return batch[0], nil
}

func (s *Server) serviceCredentialView(actor identity.Verifier, service identity.ServiceInfo, info identity.CredentialInfo, reference time.Time) serviceCredentialRow {
	now := s.clock()
	current := credentialState(info, now)
	row := serviceCredentialRow{credentialRow: credentialRow{ID: info.ID, NotBefore: info.Validity.NotBefore.UTC().Format(time.RFC3339Nano), NotAfter: info.Validity.NotAfter.UTC().Format(time.RFC3339Nano), State: credentialState(info, reference), Usable: current == "active" && !now.Before(info.Validity.NotBefore) && service.State == identity.Active}, Service: service.ID, ServiceState: serviceState(service.State), CurrentState: current, Observed: now.UTC().Format(time.RFC3339Nano)}
	row.Replaces, row.ReplacedBy = info.Replaces, info.ReplacedBy
	switch {
	case current == "active" && info.ReplacedBy != "":
		row.Rotation = rotationEnding
	case current == "active" && info.Replaces != "":
		if _, replaced, err := s.identity.Workspace().Credential(actor, s.bootWorkspace, info.Replaces); err == nil && credentialState(replaced, now) == "active" {
			row.Rotation = rotationReplacing
		}
	}
	recorded, known := s.dir.collected[info.ID]
	if current != "active" || !known || recorded.service != service.ID {
		return row
	}
	if pickup, open := s.dir.credentialPickup(info.ID); open && now.Before(pickup.expires) {
		if until, live := s.pickupLive(pickup); live {
			row.Pickup = pickupOpen
			row.PickupEnd = until.UTC().Format(time.RFC3339Nano)
			return row
		}
	}
	if recorded.collected {
		row.Pickup = pickupCollected
	} else {
		row.Pickup = pickupNever
	}
	return row
}

func (walk *serviceWalk) credentialMatches(service identity.ServiceInfo, info identity.CredentialInfo) bool {
	q := walk.query
	state := credentialState(info, walk.cursor.Reference)
	if q.State != "all" && q.State != state {
		return false
	}
	if q.Q != "" && !strings.Contains(strings.ToLower(service.ID+"\n"+info.ID), q.Q) {
		return false
	}
	if q.Collection == "uncollected" {
		recorded, known := walk.s.dir.collected[info.ID]
		if state != "active" || !known || recorded.service != service.ID || recorded.collected {
			return false
		}
	}
	return true
}

func (walk *serviceWalk) credentials() ([]serviceCredentialRow, bool, error) {
	s := walk.s
	rows := []serviceCredentialRow{}
	if s.faulty("services") || s.faulty("credentials") {
		return nil, false, identity.ErrUnavailable
	}
	if walk.query.Service != "" {
		_, err := s.identity.Workspace().ServiceUser(walk.actor, s.bootWorkspace, walk.query.Service)
		if identity.ReasonOf(err) == identity.ReasonUnknown {
			return rows, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		walk.total = 1
	} else {
		total, err := s.identity.Workspace().ServiceCount(walk.actor, s.bootWorkspace)
		if err != nil {
			return nil, false, err
		}
		walk.total = uint64(total)
	}
	if walk.cursor.Completed > walk.total {
		return nil, false, errServiceViewChanged
	}
	for walk.cursor.Completed < walk.total && len(rows) < walk.query.Limit && walk.remaining() > 0 {
		service, err := walk.nextService()
		if err != nil {
			return nil, false, err
		}
		walk.scanned++
		complete := false
		for len(rows) < walk.query.Limit && walk.remaining() > 0 {
			limit := min(walk.remaining(), s.cfg.Identity.PageSize, walk.query.Limit-len(rows))
			batch, err := s.identity.Workspace().CredentialPage(walk.actor, s.bootWorkspace, service.ID, walk.cursor.AfterCredential, limit)
			if err != nil {
				return nil, false, err
			}
			for _, info := range batch {
				walk.scanned++
				walk.cursor.AfterCredential = info.ID
				if !walk.credentialMatches(service, info) {
					continue
				}
				rows = append(rows, s.serviceCredentialView(walk.actor, service, info, walk.cursor.Reference))
				if err := walk.matched(); err != nil {
					return nil, false, err
				}
			}
			if len(batch) < limit {
				complete = true
				break
			}
		}
		if !complete {
			break
		}
		walk.finishService(service.ID)
	}
	return rows, walk.cursor.Completed < walk.total, nil
}

func (s *Server) queryServiceCredentials(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	walk, ok := s.beginServiceQuery(w, r, "credentials", "")
	if !ok {
		return
	}
	rows, more, err := walk.credentials()
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	writeServiceQueryPage(w, r, walk, rows, more, "service_id_asc_credential_id_asc")
}
