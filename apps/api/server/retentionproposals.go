package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

var retentionKinds = []string{retention.ChangePolicy, retention.ChangeRelease, retention.ChangePlan}

type retentionProposalsPage struct {
	Revision   string                 `json:"revision"`
	ServerTime string                 `json:"server_time"`
	Order      string                 `json:"order"`
	Q          string                 `json:"q"`
	Kind       string                 `json:"kind"`
	Proposals  []retentionProposalRow `json:"proposals"`
	Next       string                 `json:"next"`
	Matching   int                    `json:"matching"`
	History    keptFacts              `json:"proposal_history"`
}

func retentionPending(e *retentionEntry) bool { return e.view.State == retention.ProposalPending }

func retentionNumber(e *retentionEntry) uint64 { return e.view.Number }

func (f *retentionFacts) visibleEntries() []*retentionEntry {
	out := []*retentionEntry{}
	for _, e := range f.s.ret.entries {
		if f.s.ret.visibleEntry(e, f.workspace) {
			out = append(out, e)
		}
	}
	return out
}

func (s *Server) listRetentionProposals(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	kinds, named := values["kind"]
	values.Del("kind")
	q, valid := parseKeptQuery(values)
	kind := ""
	if named {
		valid = valid && len(kinds) == 1
		if valid {
			kind, valid = kinds[0], false
			for _, known := range retentionKinds {
				valid = valid || known == kind
			}
		}
	}
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	now := s.clock()
	page := retentionProposalsPage{Revision: s.Revision(), ServerTime: stamp(now), Order: closedPlansOrder, Q: q.term, Kind: kind, Proposals: []retentionProposalRow{}}
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	if s.faulty("retention-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f, err := s.retentionFactsFor(found.verifier, answer.User(), answer.Workspace(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	visible := f.visibleEntries()
	page.History = keptFacts{Kept: installKept, Dropped: s.ret.dropped}
	walk := keptWalk{q: q}
	for index := len(visible) - 1; index >= 0; index-- {
		e := visible[index]
		v := e.view
		if retentionPending(e) || kind != "" && v.Change.Kind != kind {
			continue
		}
		page.History.Total++
		number := strconv.FormatUint(v.Number, 10)
		if walk.take(v.Number, q.matches(number, retentionProposalTarget+number, v.Change.Kind, v.State, v.Proposer, s.loginOf(v.Proposer), e.rationale)) {
			row, err := f.proposalRow(e)
			if err != nil {
				failCode(w, commands.DependencyUnavailable, request)
				return
			}
			page.Proposals = append(page.Proposals, row)
		}
	}
	page.Next, page.Matching = walk.Next, walk.Matching
	writeJSON(w, http.StatusOK, request, page)
}
