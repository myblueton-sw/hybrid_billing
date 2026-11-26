package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const profileTargets = 20

type evidenceProfilesPage struct {
	Revision   string               `json:"revision"`
	ServerTime string               `json:"server_time"`
	Order      string               `json:"order"`
	Q          string               `json:"q"`
	Profiles   []evidenceProfileRow `json:"profiles"`
	Next       string               `json:"next"`
	Matching   int                  `json:"matching"`
	Total      int                  `json:"total"`
}

func profileTargetsOf(raw string) ([]string, bool) {
	out, seen := []string{}, map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		id, found := strings.CutPrefix(part, perfProfileTarget)
		if !found || id == "" || len(id) > maxProfileTargetLen || seen[id] || len(out) == profileTargets {
			return nil, false
		}
		out, seen[id] = append(out, id), true
	}
	return out, true
}

func (s *Server) listEvidenceProfiles(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	named, byTarget := values["target"]
	var q keptQuery
	var wanted []string
	valid := true
	if byTarget {
		valid = len(values) == 1 && len(named) == 1
		if valid {
			wanted, valid = profileTargetsOf(named[0])
		}
	} else {
		q, valid = parseKeptQuery(values)
	}
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	now := s.clock()
	page := evidenceProfilesPage{Revision: s.Revision(), ServerTime: stamp(now), Order: requirementsOrder, Q: q.term, Profiles: []evidenceProfileRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("release-evidence-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f, err := s.evidenceFactsFor(found.verifier, answer.User(), now)
	var statuses map[string]string
	if err == nil {
		statuses, _, err = f.profileStatuses()
	}
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	profiles := s.ev.profiles
	page.Total = len(profiles)
	last := 0
	for index, id := range profiles {
		switch {
		case byTarget:
			listed := false
			for _, want := range wanted {
				listed = listed || want == id
			}
			if !listed {
				continue
			}
		case !q.matches(id, perfProfileTarget+id):
			continue
		}
		page.Matching++
		if !byTarget {
			if index < int(q.before) {
				continue
			}
			if len(page.Profiles) == q.limit {
				page.Next = strconv.Itoa(last)
				continue
			}
		}
		row, err := f.profileAt(index, statuses)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Profiles, last = append(page.Profiles, row), index+1
	}
	writeJSON(w, http.StatusOK, request, page)
}
