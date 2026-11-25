package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

const (
	retentionOrder    = "identifier"
	retentionIDs      = 20
	retentionIDBytes  = 128
	retentionIDsParam = "id"
)

type retentionArtifactEntry struct {
	Class string `json:"class"`
	retentionArtifactRow
}

type retentionArtifactsPage struct {
	Revision   string                   `json:"revision"`
	ServerTime string                   `json:"server_time"`
	Order      string                   `json:"order"`
	Class      string                   `json:"class"`
	Q          string                   `json:"q"`
	Artifacts  []retentionArtifactEntry `json:"artifacts"`
	Next       string                   `json:"next"`
	Matching   int                      `json:"matching"`
	Total      int                      `json:"total"`
}

type retentionReferencesPage struct {
	Revision   string                  `json:"revision"`
	ServerTime string                  `json:"server_time"`
	Order      string                  `json:"order"`
	Q          string                  `json:"q"`
	References []retentionReferenceRow `json:"references"`
	Next       string                  `json:"next"`
	Matching   int                     `json:"matching"`
	Total      int                     `json:"total"`
}

type idQuery struct {
	class  string
	after  string
	limit  int
	term   string
	wanted map[string]bool
}

func retentionID(id string) bool {
	if id == "" || len(id) > retentionIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func parseIDQuery(values url.Values, classed bool) (idQuery, bool) {
	out := idQuery{limit: retentionShown}
	if named, byID := values[retentionIDsParam]; byID {
		if !classed || len(values) != 1 || len(named) != 1 {
			return idQuery{}, false
		}
		out.wanted = map[string]bool{}
		for _, id := range strings.Split(named[0], ",") {
			if !retentionID(id) || out.wanted[id] || len(out.wanted) == retentionIDs {
				return idQuery{}, false
			}
			out.wanted[id] = true
		}
		return out, true
	}
	for name, given := range values {
		valid := len(given) == 1 && given[0] != ""
		switch {
		case !valid:
		case name == "class" && classed:
			out.class = given[0]
			valid = false
			for _, class := range retention.Classes {
				valid = valid || class == out.class
			}
		case name == "after":
			out.after, valid = given[0], retentionID(given[0])
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			out.limit, valid = n, err == nil && n >= 1 && n <= retentionShown && strconv.Itoa(n) == given[0]
		case name == "q":
			out.term, valid = parseSearch(given[0])
		default:
			valid = false
		}
		if !valid {
			return idQuery{}, false
		}
	}
	return out, !classed || out.class != ""
}

func (q idQuery) matches(id string) bool {
	return keptQuery{term: q.term}.matches(id)
}

func (s *Server) retentionListFacts(w http.ResponseWriter, r *http.Request, classed bool) (*retentionFacts, idQuery, string, bool) {
	request := r.Header.Get("X-Request-Id")
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return nil, idQuery{}, request, false
	}
	q, valid := parseIDQuery(r.URL.Query(), classed)
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return nil, idQuery{}, request, false
	}
	if answer.Workspace() != s.bootWorkspace {
		return nil, q, request, true
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return nil, idQuery{}, request, false
	}
	f, err := s.retentionFactsFor(found.verifier, answer.User(), answer.Workspace(), s.clock())
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return nil, idQuery{}, request, false
	}
	return f, q, request, true
}

func (s *Server) listRetentionArtifacts(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	f, q, request, ok := s.retentionListFacts(w, r, true)
	if !ok {
		return
	}
	page := retentionArtifactsPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: retentionOrder, Class: q.class, Q: q.term, Artifacts: []retentionArtifactEntry{}}
	if f == nil {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	last := ""
	for _, id := range f.visibleIDs() {
		class := s.ret.artifact(id).class
		if q.wanted != nil {
			if !q.wanted[id] {
				continue
			}
		} else {
			if class != q.class {
				continue
			}
			page.Total++
			if !q.matches(id) {
				continue
			}
		}
		page.Matching++
		if q.wanted == nil {
			if id <= q.after {
				continue
			}
			if len(page.Artifacts) == q.limit {
				page.Next = last
				continue
			}
		}
		row, err := f.artifactRow(id)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Artifacts, last = append(page.Artifacts, retentionArtifactEntry{Class: class, retentionArtifactRow: row}), id
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) listRetentionReferences(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	f, q, request, ok := s.retentionListFacts(w, r, false)
	if !ok {
		return
	}
	page := retentionReferencesPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: retentionOrder, Q: q.term, References: []retentionReferenceRow{}}
	if f == nil {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	last := ""
	for _, id := range f.visibleIDs() {
		_, standing, err := f.of(id)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		if !standing.Referenced {
			continue
		}
		page.Total++
		if !q.matches(id) {
			continue
		}
		page.Matching++
		if id <= q.after {
			continue
		}
		if len(page.References) == q.limit {
			page.Next = last
			continue
		}
		page.References, last = append(page.References, retentionReferenceRow{ID: id, Class: s.ret.artifact(id).class}), id
	}
	writeJSON(w, http.StatusOK, request, page)
}
