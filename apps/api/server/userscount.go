package server

import (
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type usersCount struct {
	Count       int    `json:"count"`
	Complete    bool   `json:"complete"`
	OverLimit   bool   `json:"over_limit"`
	Limit       int    `json:"limit"`
	ScanBudget  int    `json:"scan_budget"`
	ListVersion string `json:"list_version"`
	ServerTime  string `json:"server_time"`
}

func (s *Server) countUsers(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	values := r.URL.Query()
	query, code, _ := parseUsersQuery(values)
	if code == "" && (values.Has("after") || values.Has("limit")) {
		code = commands.InvalidRequest
	}
	if code != "" {
		failRead(w, code, request)
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	if answer.Workspace() != s.bootWorkspace {
		failCode(w, commands.PermissionDenied, request)
		return
	}
	now := s.clock()
	walk := s.matchWalk(found.verifier, query.filter, now, s.cfg.Access.UserCountScan)
	complete, err := walk.all(s.dir.order, s.dir.departed)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, usersCount{Count: len(walk.matched), Complete: complete, OverLimit: len(walk.matched) > walk.limit, Limit: walk.limit,
		ScanBudget: walk.budget, ListVersion: s.listVersion(), ServerTime: stamp(now)})
}
