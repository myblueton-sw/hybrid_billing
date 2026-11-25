package server

import (
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

type retentionTombstonesPage struct {
	Revision   string                  `json:"revision"`
	ServerTime string                  `json:"server_time"`
	Order      string                  `json:"order"`
	Q          string                  `json:"q"`
	Watermark  uint64                  `json:"watermark"`
	Window     int                     `json:"window"`
	Tombstones []retentionTombstoneRow `json:"tombstones"`
	Next       string                  `json:"next"`
	Matching   int                     `json:"matching"`
	Total      int                     `json:"total"`
}

func (f *retentionFacts) windowTombstones() (uint64, []retention.Tombstone, error) {
	s := f.s
	var watermark uint64
	if err := s.retentionCall("watermark", func() error { watermark = s.retention.Watermark(); return nil }); err != nil {
		return 0, nil, err
	}
	after := uint64(0)
	if watermark > retentionWindow {
		after = watermark - retentionWindow
	}
	var list []retention.Tombstone
	if err := s.retentionCall("tombstones", func() (err error) { list, err = s.retention.Tombstones(after, retentionWindow); return }); err != nil {
		return 0, nil, err
	}
	visible := []retention.Tombstone{}
	for index := len(list) - 1; index >= 0; index-- {
		if s.ret.visibleArtifact(list[index].Artifact, f.workspace) {
			visible = append(visible, list[index])
		}
	}
	return watermark, visible, nil
}

func (f *retentionFacts) tombstoneRow(x retention.Tombstone) retentionTombstoneRow {
	row := retentionTombstoneRow{Sequence: x.Sequence, Artifact: x.Artifact, Class: x.Class, At: stamp(x.At), Decision: notConnected}
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
	q, valid := parseKeptQuery(r.URL.Query())
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
	var visible []retention.Tombstone
	var err error
	f := s.retentionFactsWithoutRoster(answer.User(), answer.Workspace())
	page.Watermark, visible, err = f.windowTombstones()
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page.Total = len(visible)
	walk := keptWalk{q: q}
	for _, x := range visible {
		row := f.tombstoneRow(x)
		label := ""
		if row.Run != nil {
			label = row.Run.Label
		}
		if walk.take(x.Sequence, q.term == "" || q.matches(x.Artifact, x.Class, label)) {
			page.Tombstones = append(page.Tombstones, row)
		}
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}
