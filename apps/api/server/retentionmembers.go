package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type retentionRunArtifactsPage struct {
	Revision   string                    `json:"revision"`
	ServerTime string                    `json:"server_time"`
	Order      string                    `json:"order"`
	Run        string                    `json:"run"`
	Artifacts  []retentionRunArtifactRow `json:"artifacts"`
	Next       string                    `json:"next"`
	Total      int                       `json:"total"`
}

type retentionHoldOutcomesPage struct {
	Revision   string                `json:"revision"`
	ServerTime string                `json:"server_time"`
	Order      string                `json:"order"`
	Hold       string                `json:"hold"`
	Outcomes   []retentionOutcomeRow `json:"outcomes"`
	Next       string                `json:"next"`
}

func memberLimit(given []string, present bool) (int, bool) {
	if !present {
		return retentionShown, true
	}
	if len(given) != 1 {
		return 0, false
	}
	n, err := strconv.Atoi(given[0])
	return n, err == nil && n >= 1 && n <= retentionShown && strconv.Itoa(n) == given[0]
}

func (s *Server) memberFacts(w http.ResponseWriter, r *http.Request, parent string, valid func(map[string][]string) bool) (*retentionFacts, string, bool) {
	request := r.Header.Get("X-Request-Id")
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return nil, request, false
	}
	values := r.URL.Query()
	for name, given := range values {
		if name != parent && name != "after" && name != "limit" || len(given) != 1 || given[0] == "" {
			failCode(w, commands.InvalidRequest, request)
			return nil, request, false
		}
	}
	if len(values[parent]) != 1 || !valid(values) {
		failCode(w, commands.InvalidRequest, request)
		return nil, request, false
	}
	if answer.Workspace() != s.bootWorkspace {
		failCode(w, commands.InvalidRequest, request)
		return nil, request, false
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return nil, request, false
	}
	return s.retentionFactsWithoutRoster(answer.User(), answer.Workspace()), request, true
}

func (s *Server) listRetentionRunArtifacts(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	after, limit := 0, 0
	f, request, ok := s.memberFacts(w, r, "run", func(values map[string][]string) bool {
		valid := true
		if given, present := values["after"]; present {
			n, err := strconv.Atoi(given[0])
			after, valid = n, err == nil && n >= 1 && strconv.Itoa(n) == given[0]
		}
		var fits bool
		limit, fits = memberLimit(values["limit"], len(values["limit"]) > 0)
		return valid && fits
	})
	if !ok {
		return
	}
	target := r.URL.Query().Get("run")
	number, prefixed := strings.CutPrefix(target, retentionRunTarget)
	id := retentionRunPrefix + number
	if !prefixed || number == "" || runTarget(id) != target || !s.ret.visibleRun(id, f.workspace) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	v, err := s.retentionRunView(id)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page := retentionRunArtifactsPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: retentionOrder, Run: target, Artifacts: []retentionRunArtifactRow{}}
	for _, artifact := range v.Plan.Artifacts {
		if !s.ret.visibleArtifact(artifact, f.workspace) {
			continue
		}
		page.Total++
		if page.Total <= after {
			continue
		}
		if len(page.Artifacts) == limit {
			page.Next = strconv.Itoa(after + limit)
			continue
		}
		page.Artifacts = append(page.Artifacts, retentionRunArtifactRow{ID: artifact, Outcome: v.Outcomes[artifact]})
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) listRetentionHoldOutcomes(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	after, limit := "", 0
	f, request, ok := s.memberFacts(w, r, "hold", func(values map[string][]string) bool {
		valid := true
		if given, present := values["after"]; present {
			after, valid = given[0], retentionID(given[0])
		}
		var fits bool
		limit, fits = memberLimit(values["limit"], len(values["limit"]) > 0)
		return valid && fits
	})
	if !ok {
		return
	}
	id := r.URL.Query().Get("hold")
	h := s.ret.hold(id)
	if h == nil || h.workspace != f.workspace {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	outcomes, err := f.visibleOutcomes(id, after, limit)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page := retentionHoldOutcomesPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: retentionOrder, Hold: id, Outcomes: []retentionOutcomeRow{}}
	for _, o := range outcomes[:min(len(outcomes), limit)] {
		page.Outcomes = append(page.Outcomes, retentionOutcomeRow{Artifact: o.Artifact, Outcome: o.Outcome, Stores: o.Stores})
	}
	if len(outcomes) > limit {
		page.Next = page.Outcomes[len(page.Outcomes)-1].Artifact
	}
	writeJSON(w, http.StatusOK, request, page)
}
