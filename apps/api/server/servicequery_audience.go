package server

import (
	"net/http"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) readServiceAudience(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	id := r.PathValue("id")
	read, ok := s.beginServiceDetail(w, r, "audience_detail", id)
	if !ok {
		return
	}
	audience, err := s.serviceCatalog.Audience(s.bootWorkspace, id)
	if err != nil {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	row := serviceAudienceRow{ID: audience.ID, Resource: audience.Resource, ScopeCount: audience.ScopeCount}
	read.write(w, r, map[string]any{"workspace": s.bootWorkspace, "row": row, "catalog_version": s.serviceCatalog.Version(), "revision": s.Revision(), "observed_at": read.now.UTC().Format(time.RFC3339Nano)})
}
