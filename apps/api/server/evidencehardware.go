package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type evidenceHardwarePage struct {
	Revision   string                `json:"revision"`
	ServerTime string                `json:"server_time"`
	Order      string                `json:"order"`
	Q          string                `json:"q"`
	Hardware   []evidenceHardwareRow `json:"hardware"`
	Next       string                `json:"next"`
	Matching   int                   `json:"matching"`
	Total      int                   `json:"total"`
}

func (s *Server) hardwareAt(index int) (evidenceHardwareRow, error) {
	h, err := s.perfHardware(s.ev.hardware[index])
	return evidenceHardwareRow{Number: index + 1, ID: h.ID, CPUs: h.CPUs, MemoryMiB: h.MemoryMiB, DiskGiB: h.DiskGiB, Storage: h.Storage, Spec: h.Spec}, err
}

func (s *Server) listEvidenceHardware(w http.ResponseWriter, r *http.Request) {
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
	page := evidenceHardwarePage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Order: requirementsOrder, Q: q.term, Hardware: []evidenceHardwareRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("release-evidence-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page.Total = len(s.ev.hardware)
	last := 0
	for index, id := range s.ev.hardware {
		if !q.matches(id) {
			continue
		}
		page.Matching++
		if index < int(q.before) {
			continue
		}
		if len(page.Hardware) == q.limit {
			page.Next = strconv.Itoa(last)
			continue
		}
		row, err := s.hardwareAt(index)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Hardware, last = append(page.Hardware, row), index+1
	}
	writeJSON(w, http.StatusOK, request, page)
}
