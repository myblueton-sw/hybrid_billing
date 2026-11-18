package server

import (
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type bulkCounts struct {
	Targets     int `json:"targets"`
	Excluded    int `json:"excluded"`
	Duplicates  int `json:"duplicates"`
	Succeeded   int `json:"succeeded"`
	Held        int `json:"held"`
	Failed      int `json:"failed"`
	Unprocessed int `json:"unprocessed"`
}

type bulkLedger struct {
	FirstSequence uint64 `json:"first_sequence"`
	LastSequence  uint64 `json:"last_sequence"`
	Watermark     string `json:"watermark"`
	Reason        string `json:"reason"`
}

type bulkCleanup struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

type bulkRow struct {
	ID         string      `json:"id"`
	State      string      `json:"state"`
	Origin     string      `json:"origin"`
	Effect     string      `json:"effect"`
	Selection  string      `json:"selection"`
	Preparer   string      `json:"preparer"`
	PreparedAt string      `json:"prepared_at"`
	ExpiresAt  string      `json:"expires_at"`
	Executor   string      `json:"executor"`
	ExecutedAt string      `json:"executed_at"`
	DenyAt     string      `json:"deny_at"`
	Digest     string      `json:"digest"`
	Counts     bulkCounts  `json:"counts"`
	Staged     int         `json:"staged"`
	JobState   string      `json:"job_state"`
	JobReason  string      `json:"job_reason"`
	Ledger     bulkLedger  `json:"ledger"`
	Cleanup    bulkCleanup `json:"cleanup"`
	RecordedAt string      `json:"recorded_at"`
}

type bulkPage struct {
	Revision   string       `json:"revision"`
	StartedAt  string       `json:"started_at"`
	CanPrepare bool         `json:"can_prepare"`
	OpenDraft  bool         `json:"open_draft"`
	Pending    int          `json:"pending"`
	Next       string       `json:"next"`
	Jobs       []bulkRow    `json:"jobs"`
	Settings   []settingRow `json:"settings"`
	Ledger     string       `json:"ledger_state"`
	clockRows
}

type bulkPendingRow struct {
	Store     string `json:"store"`
	Count     int    `json:"count"`
	Rechecked bool   `json:"rechecked"`
}

type bulkImpactRow struct {
	ComputedAt                 string           `json:"computed_at"`
	People                     int              `json:"people"`
	Administrators             int              `json:"administrators"`
	Members                    int              `json:"members"`
	InstallationAdministrators int              `json:"installation_administrators"`
	TrustHolders               int              `json:"trust_right_holders"`
	BulkHolders                int              `json:"bulk_right_holders"`
	Sessions                   int              `json:"sessions"`
	Pickups                    int              `json:"pickups"`
	ApiKeys                    string           `json:"api_keys"`
	Pending                    []bulkPendingRow `json:"pending"`
	PendingJobs                string           `json:"pending_jobs"`
	ServiceUsers               int              `json:"service_users"`
	AnchorsAfter               int              `json:"anchors_after"`
	OtherWorkspaces            string           `json:"other_workspaces"`
}

type bulkDetail struct {
	Revision   string         `json:"revision"`
	ServerTime string         `json:"server_time"`
	Job        bulkRow        `json:"job"`
	Impact     bulkImpactRow  `json:"impact"`
	Exclusions map[string]int `json:"exclusions"`
	Outside    []string       `json:"outside_scope"`
}

type bulkCleanupRow struct {
	Item   string `json:"item"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type bulkReceiptRow struct {
	Sequence  uint64 `json:"sequence"`
	Watermark string `json:"watermark"`
	Reason    string `json:"reason"`
}

type bulkItemRow struct {
	UserID    string           `json:"user_id"`
	Login     string           `json:"login"`
	Role      string           `json:"role"`
	RoleUntil string           `json:"role_until"`
	Outcome   string           `json:"outcome"`
	Reason    string           `json:"reason"`
	State     string           `json:"state"`
	Receipt   bulkReceiptRow   `json:"receipt"`
	Cleanup   []bulkCleanupRow `json:"cleanup"`
	Note      string           `json:"replay_note"`
}

type itemsPage struct {
	Revision string        `json:"revision"`
	Total    int           `json:"total"`
	Next     string        `json:"next"`
	Items    []bulkItemRow `json:"items"`
}

type bulkTargetRow struct {
	UserID          string `json:"user_id"`
	Login           string `json:"login"`
	Role            string `json:"role"`
	RoleUntil       string `json:"role_until"`
	UserState       string `json:"user_state"`
	MembershipState string `json:"membership_state"`
	Administrator   bool   `json:"installation_administrator"`
}

type exclusionRow struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
	Reason string `json:"reason"`
}

type exclusionsPage struct {
	Revision   string         `json:"revision"`
	Total      int            `json:"total"`
	Next       string         `json:"next"`
	Exclusions []exclusionRow `json:"exclusions"`
}

type bulkTargetsPage struct {
	Revision string          `json:"revision"`
	Total    int             `json:"total"`
	Next     string          `json:"next"`
	Targets  []bulkTargetRow `json:"targets"`
}

var outsideScope = []string{"B7", "B8", "B9", "B10"}

func (s *Server) bulkRow(v bulkrevoke.View) bulkRow {
	row := bulkRow{ID: v.ID, State: string(v.State), Origin: v.Origin, Effect: string(v.Effect), Selection: v.Selection, Preparer: v.Owner,
		PreparedAt: stampOr(v.PreparedAt), ExpiresAt: stampOr(v.ExpiresAt), Executor: v.Executor, ExecutedAt: stampOr(v.ExecutedAt), DenyAt: stampOr(v.DenyAt),
		Digest: v.Digest, Counts: bulkCounts(v.Counts), RecordedAt: stampOr(v.RecordedAt), Staged: v.Count}
	if v.State != bulkrevoke.Executed {
		return row
	}
	row.JobState, row.JobReason = bulkrevoke.JobState(v, s.cleaning(v))
	row.Cleanup.Done, row.Cleanup.Total = s.cleanupProgress(v)
	row.Ledger = bulkLedger{FirstSequence: v.Header.Sequence, LastSequence: v.Header.Sequence, Watermark: v.Header.Watermark, Reason: v.Header.Reason}
	for _, item := range v.Items {
		switch {
		case item.Receipt.Reason != "" && row.Ledger.Reason == "":
			row.Ledger.Reason = item.Receipt.Reason
		case item.Receipt.Sequence > row.Ledger.LastSequence:
			row.Ledger.LastSequence, row.Ledger.Watermark = item.Receipt.Sequence, item.Receipt.Watermark
		}
	}
	if v.ReplayMissing > 0 && row.Ledger.Reason == "" {
		row.Ledger.Reason = bulkrevoke.RecordNotConfirmed
	}
	return row
}

func (s *Server) bulkReader(w http.ResponseWriter, r *http.Request) (identity.Answer, string, bool) {
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return answer, "", false
	}
	return answer, answer.User(), true
}

func (s *Server) listBulk(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	after, limit, ok := pageLimit(r)
	if !ok {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	answer, user, ok := s.bulkReader(w, r)
	if !ok {
		return
	}
	now := s.clock()
	pending, open := s.bulk.Limits(s.bootWorkspace, user)
	page := bulkPage{Revision: s.Revision(), StartedAt: stamp(s.startedAt), CanPrepare: s.canBulk(answer) && len(s.stranded) == 0, OpenDraft: open, Pending: pending,
		Jobs: []bulkRow{}, Settings: s.identitySettingRows(), Ledger: s.ledgerRow().State, clockRows: s.clockRows(now)}
	rows := s.bulk.List(s.bootWorkspace, user)
	start := 0
	if after != "" {
		start = len(rows)
		for i, v := range rows {
			if v.ID == after {
				start = i + 1
			}
		}
	}
	for _, v := range rows[start:min(len(rows), start+limit)] {
		full, _ := s.bulk.Get(s.bootWorkspace, user, v.ID)
		page.Jobs = append(page.Jobs, s.bulkRow(full))
	}
	if start+limit < len(rows) {
		page.Next = page.Jobs[len(page.Jobs)-1].ID
	}
	writeJSON(w, http.StatusOK, request, page)
}

func impactOf(i bulkrevoke.Impact, effect bulkrevoke.Effect) bulkImpactRow {
	out := bulkImpactRow{ComputedAt: stampOr(i.ComputedAt), People: i.People, Administrators: i.Administrators, Members: i.Members,
		InstallationAdministrators: i.InstallationAdministrators, TrustHolders: i.TrustHolders, BulkHolders: i.BulkHolders, Sessions: i.Sessions, Pickups: i.Pickups,
		ApiKeys: "not_applicable", PendingJobs: "not_applicable", ServiceUsers: i.ServiceUsers, AnchorsAfter: i.AnchorsAfter, Pending: []bulkPendingRow{}}
	for _, p := range i.Pending {
		out.Pending = append(out.Pending, bulkPendingRow(p))
	}
	if effect == bulkrevoke.SuspendMembership {
		out.OtherWorkspaces = "not_changed"
	}
	return out
}

func (s *Server) readBulk(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, user, ok := s.bulkReader(w, r)
	if !ok {
		return
	}
	v, found := s.bulk.Get(s.bootWorkspace, user, r.PathValue("id"))
	if !found {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	detail := bulkDetail{Revision: s.Revision(), ServerTime: stamp(s.clock()), Job: s.bulkRow(v), Impact: impactOf(v.Impact, v.Effect), Outside: outsideScope,
		Exclusions: map[string]int{bulkrevoke.ExcludedSelf: 0, bulkrevoke.ExcludedNotFound: 0, bulkrevoke.ExcludedNoMembership: 0, bulkrevoke.ExcludedAlreadyRevoked: 0}}
	for _, x := range v.Exclusions {
		detail.Exclusions[x.Reason]++
	}
	writeJSON(w, http.StatusOK, request, detail)
}

func offsetLimit(r *http.Request) (int, int, bool) {
	after, limit, ok := pageLimit(r)
	if !ok {
		return 0, 0, false
	}
	if after == "" {
		return 0, limit, true
	}
	n, err := strconv.Atoi(after)
	return n, limit, err == nil && n >= 0 && strconv.Itoa(n) == after
}

func nextOffset(from, shown, total int) string {
	if from+shown >= total {
		return ""
	}
	return strconv.Itoa(from + shown)
}

func (s *Server) readBulkItems(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	after, limit, ok := offsetLimit(r)
	outcome := bulkrevoke.Outcome(r.URL.Query().Get("outcome"))
	switch outcome {
	case "", bulkrevoke.OutcomeExcluded, bulkrevoke.OutcomeHeld, bulkrevoke.OutcomeSucceeded, bulkrevoke.OutcomeFailed, bulkrevoke.OutcomeUnprocessed:
	default:
		ok = false
	}
	if !ok {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, user, ok := s.bulkReader(w, r)
	if !ok {
		return
	}
	items, total, found := s.bulk.Items(s.bootWorkspace, user, r.PathValue("id"), after, limit, outcome)
	if !found {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	page := itemsPage{Revision: s.Revision(), Total: total, Next: nextOffset(after, len(items), total), Items: []bulkItemRow{}}
	for _, item := range items {
		row := bulkItemRow{UserID: item.Target.User, Login: item.Target.Login, Role: item.Target.Role, RoleUntil: stampOr(item.Target.Until), Outcome: string(item.Outcome),
			Reason: item.Reason, State: item.State, Receipt: bulkReceiptRow(item.Receipt), Cleanup: []bulkCleanupRow{}, Note: item.Note}
		for _, c := range item.Cleanup {
			row.Cleanup = append(row.Cleanup, bulkCleanupRow(c))
		}
		page.Items = append(page.Items, row)
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) readBulkTargets(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	after, limit, ok := offsetLimit(r)
	if !ok {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, user, ok := s.bulkReader(w, r)
	if !ok {
		return
	}
	switch r.URL.Query().Get("kind") {
	case "", "excluded":
	default:
		failCode(w, commands.InvalidRequest, request)
		return
	}
	if r.URL.Query().Get("kind") == "excluded" {
		rows, total, found := s.bulk.Exclusions(s.bootWorkspace, user, r.PathValue("id"), after, limit)
		if !found {
			failCode(w, commands.UnsupportedCommand, request)
			return
		}
		page := exclusionsPage{Revision: s.Revision(), Total: total, Next: nextOffset(after, len(rows), total), Exclusions: []exclusionRow{}}
		for _, x := range rows {
			page.Exclusions = append(page.Exclusions, exclusionRow{UserID: x.User, Login: x.Login, Reason: x.Reason})
		}
		writeJSON(w, http.StatusOK, request, page)
		return
	}
	targets, total, found := s.bulk.Targets(s.bootWorkspace, user, r.PathValue("id"), after, limit)
	if !found {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	page := bulkTargetsPage{Revision: s.Revision(), Total: total, Next: nextOffset(after, len(targets), total), Targets: []bulkTargetRow{}}
	for _, t := range targets {
		page.Targets = append(page.Targets, bulkTargetRow{UserID: t.User, Login: t.Login, Role: t.Role, RoleUntil: stampOr(t.Until), UserState: t.UserState,
			MembershipState: t.MembershipState, Administrator: t.Administrator})
	}
	writeJSON(w, http.StatusOK, request, page)
}
