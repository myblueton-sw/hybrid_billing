package server

import (
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

const retentionTombstoneSeal = "retention-tombstone"

type retentionTombstonesPage struct {
	Revision   string                  `json:"revision"`
	ServerTime string                  `json:"server_time"`
	Order      string                  `json:"order"`
	Q          string                  `json:"q"`
	Window     int                     `json:"window"`
	Tombstones []retentionTombstoneRow `json:"tombstones"`
	Next       string                  `json:"next"`
	Matching   int                     `json:"matching"`
	Total      int                     `json:"total"`
}

type tombstoneToken struct {
	Artifact string `json:"a"`
}

type tombstoneQuery struct {
	after string
	limit int
	term  string
}

func parseTombstoneQuery(values url.Values) (tombstoneQuery, bool) {
	out := tombstoneQuery{limit: keptPage}
	for name, given := range values {
		valid := len(given) == 1 && given[0] != ""
		switch {
		case !valid:
		case name == "after":
			out.after, valid = given[0], len(given[0]) <= queryTokenBytes
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			out.limit, valid = n, err == nil && n >= 1 && n <= keptPageMax && strconv.Itoa(n) == given[0]
		case name == "q":
			out.term, valid = parseSearch(given[0])
		default:
			valid = false
		}
		if !valid {
			return tombstoneQuery{}, false
		}
	}
	return out, true
}

func (s *Server) tombstoneCursor(artifact string) string {
	token, _ := s.sealQueryToken(retentionTombstoneSeal, tombstoneToken{Artifact: artifact})
	return token
}

type tombstoneIndex struct {
	mu   sync.Mutex
	of   *retention.Registry
	by   *retentionStore
	ws   string
	upTo uint64
	rows []retention.Tombstone
}

func (f *retentionFacts) windowTombstones() ([]retention.Tombstone, error) {
	s := f.s
	if f.workspace != s.bootWorkspace {
		return []retention.Tombstone{}, nil
	}
	x := &s.tombstoneWindow
	x.mu.Lock()
	defer x.mu.Unlock()
	var newest uint64
	if err := s.retentionCall("watermark", func() error { newest = s.retention.Watermark(); return nil }); err != nil {
		return nil, err
	}
	if x.of != s.retention || x.by != s.ret || x.ws != f.workspace || newest < x.upTo {
		x.of, x.by, x.ws, x.upTo, x.rows = s.retention, s.ret, f.workspace, 0, nil
	}
	for x.upTo < newest {
		var list []retention.Tombstone
		if err := s.retentionCall("tombstones", func() (err error) {
			list, err = s.retention.Tombstones(x.upTo, int(min(newest-x.upTo, retentionWindow)))
			return
		}); err != nil {
			return nil, err
		}
		if len(list) == 0 {
			break
		}
		for _, t := range list {
			if s.ret.visibleArtifact(t.Artifact, f.workspace) {
				x.rows = append(x.rows, t)
			}
		}
		if extra := len(x.rows) - retentionWindow; extra > 0 {
			x.rows = append([]retention.Tombstone{}, x.rows[extra:]...)
		}
		x.upTo += uint64(len(list))
	}
	visible := make([]retention.Tombstone, 0, len(x.rows))
	for index := len(x.rows) - 1; index >= 0; index-- {
		visible = append(visible, x.rows[index])
	}
	return visible, nil
}

func (f *retentionFacts) tombstoneRow(x retention.Tombstone) retentionTombstoneRow {
	row := retentionTombstoneRow{Artifact: x.Artifact, Class: x.Class, At: stamp(x.At), Decision: notConnected}
	row.Run, row.OtherScope = f.runRef(x.Run)
	return row
}

func (s *Server) listRetentionTombstones(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	q, valid := parseTombstoneQuery(r.URL.Query())
	var token tombstoneToken
	if valid && q.after != "" && !s.openQueryToken(retentionTombstoneSeal, q.after, &token) {
		valid = false
	}
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	now := s.clock()
	page := retentionTombstonesPage{Revision: s.Revision(), ServerTime: stamp(now), Order: closedPlansOrder, Q: q.term, Window: retentionWindow, Tombstones: []retentionTombstoneRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f := s.retentionFactsWithoutRoster(answer.User(), answer.Workspace())
	visible, err := f.windowTombstones()
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page.Total = len(visible)
	from := 0
	if q.after != "" {
		from = -1
		for index, x := range visible {
			if x.Artifact == token.Artifact {
				from = index + 1
				break
			}
		}
		if from < 0 {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
	}
	kq := keptQuery{term: q.term}
	last := ""
	for index, x := range visible {
		row := f.tombstoneRow(x)
		label := ""
		if row.Run != nil {
			label = row.Run.Label
		}
		if !kq.matches(x.Artifact, x.Class, label) {
			continue
		}
		page.Matching++
		if index < from {
			continue
		}
		if len(page.Tombstones) == q.limit {
			page.Next = s.tombstoneCursor(last)
			continue
		}
		page.Tombstones = append(page.Tombstones, row)
		last = x.Artifact
	}
	writeJSON(w, http.StatusOK, request, page)
}
