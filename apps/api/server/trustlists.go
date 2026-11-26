package server

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

const (
	trustChangesOrder        = "created_desc"
	trustChangesCursorDomain = "trust-changes-v1"
	evidenceNotEvaluated     = "not_evaluated"
	evidenceNotApplicable    = "not_applicable"
)

var trustChangeGroups = map[string][]string{
	"open":   {trustchange.StateDraft, trustchange.StateValidated, trustchange.StateApproved, trustchange.StateUnknown},
	"closed": {trustchange.StateActivated, trustchange.StateRejected, trustchange.StateCancelled, trustchange.StateSuperseded},
}

var trustChangeStates = []string{"open", "closed", trustchange.StateDraft, trustchange.StateValidated, trustchange.StateApproved, trustchange.StateUnknown,
	trustchange.StateActivated, trustchange.StateRejected, trustchange.StateCancelled, trustchange.StateSuperseded}

type listedCan struct {
	Revise   bool `json:"revise"`
	Validate bool `json:"validate"`
	Reject   bool `json:"reject"`
	Cancel   bool `json:"cancel"`
}

type listedChange struct {
	ID                string      `json:"id"`
	Number            uint64      `json:"number"`
	State             string      `json:"state"`
	ChangeRevision    uint64      `json:"change_revision"`
	Candidate         revisionRow `json:"candidate"`
	Proposer          string      `json:"proposer"`
	Validator         string      `json:"validator"`
	Approver          string      `json:"approver"`
	Activator         string      `json:"activator"`
	SupersededBy      string      `json:"superseded_by"`
	EarlierRun        bool        `json:"earlier_run"`
	RunStartedAt      string      `json:"run_started_at"`
	ValidationResult  string      `json:"validation_result"`
	EvidenceCurrent   string      `json:"evidence_current"`
	InvalidationState string      `json:"invalidation_state"`
	DecideInDetail    bool        `json:"decide_in_detail"`
	Can               listedCan   `json:"can"`
	WhyNot            whyRow      `json:"why_not"`
	ActUntil          string      `json:"act_until"`
}

type trustChangeDetail struct {
	Change   changeRow `json:"change"`
	Revision string    `json:"revision"`
	Observed string    `json:"observed_at"`
}

func (c rowContext) listed(v trustchange.View) listedChange {
	open := trustchange.Open(v.State)
	can, why := c.actions(v, true)
	row := listedChange{ID: v.ID, Number: v.Number, State: v.State, ChangeRevision: v.Revision, Candidate: revisionOf(revisionFor(v.Workspace, v.Candidate)),
		Proposer: v.Proposer, Validator: v.Validator, Approver: v.Approver, Activator: v.Activator, SupersededBy: v.SupersededBy, EarlierRun: v.EarlierRun,
		RunStartedAt: stampOrEmpty(v.RunStartedAt), EvidenceCurrent: evidenceNotApplicable, DecideInDetail: open,
		Can: listedCan{Revise: can.Revise, Validate: can.Validate, Reject: can.Reject, Cancel: can.Cancel}, WhyNot: why, ActUntil: c.actUntil(v)}
	if v.Validation != nil {
		row.ValidationResult = v.Validation.Result
		if open {
			row.EvidenceCurrent = evidenceNotEvaluated
		}
	}
	if v.Activation != nil {
		row.InvalidationState = progressOf(c.s.progress[v.ID], v.EarlierRun).State
	}
	return row
}

func (s *Server) trustBasis(*listRead) (string, bool) {
	return hex.EncodeToString(s.queryMAC("trust-changes-source-v1", []byte(s.trust.Run()+"/"+strconv.FormatUint(s.trust.Version(), 10)))), true
}

func (s *Server) queryTrustChanges(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	read, ok := s.openList(w, r, "trust-changes", trustChangesCursorDomain, trustChangeStates, s.trustBasis)
	if !ok {
		return
	}
	read.exact = true
	states := trustChangeGroups[read.query.State]
	if states == nil && read.query.State != "" {
		states = []string{read.query.State}
	}
	page := s.trust.Page(trustchange.Query{Workspace: s.bootWorkspace, States: states, Match: read.query.Q, After: read.cursor.After, Limit: min(read.query.Limit, trustchange.MaxPage)})
	if !page.Known {
		failRead(w, commands.RevisionConflict, read.request, "after")
		return
	}
	you := read.found.owner.User
	now := s.clock()
	ctx := s.trustRows(read.found, you, now, s.rightsOf(you, now))
	rows := []listedChange{}
	for _, v := range page.Views {
		rows = append(rows, ctx.listed(v))
		read.cursor.After = v.ID
	}
	read.budget.reserve(len(rows))
	read.cursor.Matched = uint64(page.Total)
	closeList(s, w, read, trustChangesCursorDomain, trustChangesOrder, rows, !page.More)
}

func (s *Server) readTrustChange(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if answer.Workspace() != s.bootWorkspace || len(r.URL.RawQuery) != 0 || !validWorkspace(id) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	v, known := s.trust.Get(id, s.bootWorkspace)
	if !known {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	you := answer.User()
	now := s.clock()
	writeJSON(w, http.StatusOK, request, trustChangeDetail{Change: s.trustRows(found, you, now, s.rightsOf(you, now)).row(v), Revision: s.Revision(), Observed: now.UTC().Format(time.RFC3339Nano)})
}

func trustChangePaths() map[string]any {
	str := map[string]any{"type": "string"}
	row := map[string]any{"type": "object", "properties": map[string]any{"id": str}}
	return map[string]any{
		"/identity/policy/changes": map[string]any{"get": map[string]any{"operationId": "identity.policy.changes", "x-row": row, "x-filters": []string{"q", "state", "limit", "after"}, "x-order": trustChangesOrder,
			"responses": serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"rows": map[string]any{"type": "array", "items": row}}}, true),
			"x-listed":  "kept trust changes of the installation's sign-in policy, newest first by creation; the count is exact",
			"x-filter":  "state is one of the eight states or a group: open is draft, validated, independently_approved and activation_unknown, closed is activated, rejected, cancelled and superseded (the group open differs from the module's notion, which leaves activation_unknown out); q matches a part of the change identifier",
			"x-time":    "a row is the change as it is when the page is read; can and why_not are evaluated then and the command decides; for a change awaiting a decision why_not leaves out evidence_stale, evidence_current is not_evaluated once it has a validation and not_applicable otherwise, and decide_in_detail says that approval and activation are answered by the detail read",
			"x-cursor":  "bound to the filters, the caller's session, the reference time and the run and version of the trust change registry; 409 revision_conflict when a change was recorded since or the cursor's change is no longer kept"}},
		"/identity/policy/changes/{id}": map[string]any{"get": map[string]any{"operationId": "identity.policy.change",
			"responses": serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"change": row}}, false),
			"x-listed":  "one kept trust change in full, with evidence currency, impact and what the caller may do; an unknown identifier answers invalid_request"}},
	}
}
