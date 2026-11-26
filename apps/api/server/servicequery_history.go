package server

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (s *Server) queryServiceDeclarations(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	id := r.PathValue("id")
	walk, ok := s.beginServiceQuery(w, r, "history", id)
	if !ok {
		return
	}
	if !validWorkspace(id) {
		failCode(w, commands.PermissionDenied, r.Header.Get("X-Request-Id"))
		return
	}
	if s.faulty("services") || s.faulty("service-declaration") {
		serviceQueryFailure(w, r, identity.ErrUnavailable)
		return
	}
	if _, err := s.identity.Workspace().ServiceUser(walk.actor, s.bootWorkspace, id); err != nil {
		serviceKnownFailure(w, r, err)
		return
	}
	history := s.serviceDeclarations[serviceDeclarationKey{s.bootWorkspace, id}]
	rows := []serviceDeclarationView{}
	walk.availability = "recorded"
	if len(history) == 0 {
		walk.availability = "not_recorded"
	}
	if walk.query.Version != "" {
		version, _ := strconv.ParseUint(walk.query.Version, 10, 64)
		if row := s.serviceDeclarationView(id, version); row.State == "recorded" {
			rows = append(rows, row)
			walk.cursor.Matched = 1
		} else {
			walk.availability = "version_unavailable"
		}
		walk.scanned = 1
		writeServiceQueryPage(w, r, walk, rows, false, "declaration_version_desc")
		return
	}
	at := len(history) - 1
	if walk.cursor.After != "" {
		before, err := strconv.ParseUint(walk.cursor.After, 10, 64)
		if err != nil || before == 0 || strconv.FormatUint(before, 10) != walk.cursor.After {
			serviceQueryFailure(w, r, errServiceViewChanged)
			return
		}
		at = sort.Search(len(history), func(i int) bool { return history[i].Version >= before }) - 1
	}
	for at >= 0 && len(rows) < walk.query.Limit && walk.remaining() > 0 {
		walk.scanned++
		version := history[at].Version
		rows = append(rows, s.serviceDeclarationView(id, version))
		walk.cursor.After = strconv.FormatUint(version, 10)
		if err := walk.matched(); err != nil {
			serviceQueryFailure(w, r, err)
			return
		}
		at--
	}
	writeServiceQueryPage(w, r, walk, rows, at >= 0, "declaration_version_desc")
}
