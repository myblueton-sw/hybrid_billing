package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const (
	requestsOrder           = "number_desc"
	revocationsOrder        = "recorded_desc"
	requestsCursorDomain    = "access-requests-v1"
	revocationsCursorDomain = "access-revocations-v1"
)

type listQuery struct {
	Q     string `json:"q"`
	State string `json:"state"`
	Limit int    `json:"limit"`
}

type listCursor struct {
	Area      string       `json:"area"`
	Owner     sessionOwner `json:"owner"`
	Caller    string       `json:"caller"`
	Query     string       `json:"query"`
	Basis     string       `json:"basis"`
	Reference time.Time    `json:"reference"`
	Expires   time.Time    `json:"expires"`
	Number    uint64       `json:"number"`
	After     string       `json:"after"`
	Ended     int          `json:"ended"`
	Matched   uint64       `json:"matched"`
}

func (c listCursor) binding() queryBinding {
	return queryBinding{Area: c.Area, Owner: c.Owner, Caller: c.Caller, Query: c.Query, Basis: c.Basis, Reference: c.Reference, Expires: c.Expires}
}

type listedRequest struct {
	changeRequestRow
	SubjectLogin   string `json:"subject_login"`
	ProposerLogin  string `json:"proposer_login"`
	DecidedByLogin string `json:"decided_by_login"`
}

type listPage[T any] struct {
	Rows        []T               `json:"rows"`
	SearchInput string            `json:"search_input"`
	Query       listQuery         `json:"query"`
	Next        string            `json:"next"`
	Reference   string            `json:"reference_at"`
	Observed    string            `json:"observed_at"`
	Revision    string            `json:"revision"`
	Order       string            `json:"order"`
	Count       serviceQueryCount `json:"count"`
	Scanned     int               `json:"work_units"`
	ScanStopped bool              `json:"scan_stopped"`
}

func parseListQuery(values url.Values, pageLimit int, states []string) (listQuery, string, string) {
	query := listQuery{Limit: min(25, pageLimit)}
	after := ""
	for name, given := range values {
		if len(given) != 1 || given[0] == "" {
			return query, "", name
		}
		value := given[0]
		switch name {
		case "q":
			trimmed := strings.TrimSpace(value)
			if trimmed == "" || len(trimmed) > permissionSearchBytes || !utf8.ValidString(trimmed) || strings.ContainsFunc(trimmed, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
				return query, "", name
			}
			query.Q = strings.ToLower(trimmed)
		case "state":
			known := false
			for _, state := range states {
				known = known || state == value
			}
			if !known {
				return query, "", name
			}
			query.State = value
		case "limit":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > pageLimit || strconv.Itoa(n) != value {
				return query, "", name
			}
			query.Limit = n
		case "after":
			if len(value) > queryTokenBytes {
				return query, "", name
			}
			after = value
		default:
			return query, "", name
		}
	}
	return query, after, ""
}

type listRead struct {
	found   session
	values  url.Values
	query   listQuery
	cursor  listCursor
	budget  scanBudget
	request string
}

func (s *Server) listBasis(read *listRead) (string, bool) {
	access, ok := s.accessBasis()
	people, okPeople := s.subjectsBasis(read.found.verifier)
	if !ok || !okPeople {
		return "", false
	}
	return hex.EncodeToString(s.queryMAC("access-lists-source-v1", []byte(access+"/"+people+"/"+strconv.FormatBool(s.ledgerBlocksIncrease())))), true
}

func (s *Server) openList(w http.ResponseWriter, r *http.Request, area, domain string, states []string) (*listRead, bool) {
	read := &listRead{request: r.Header.Get("X-Request-Id")}
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return nil, false
	}
	read.found = found
	if answer.Workspace() != s.bootWorkspace {
		failCode(w, commands.Unauthenticated, read.request)
		return nil, false
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(r.URL.RawQuery) > 8192 {
		invalidFields(w, read.request, "query")
		return nil, false
	}
	query, after, field := parseListQuery(values, s.cfg.Access.ServiceQueryPage, states)
	if field != "" {
		invalidFields(w, read.request, field)
		return nil, false
	}
	if s.faulty("access-query") {
		failCode(w, commands.DependencyUnavailable, read.request)
		return nil, false
	}
	basis, available := s.listBasis(read)
	if !available {
		failCode(w, commands.DependencyUnavailable, read.request)
		return nil, false
	}
	now := s.clock()
	raw, _ := json.Marshal(query)
	sum := sha256.Sum256(raw)
	read.values, read.query, read.budget = values, query, scanBudget{limit: s.cfg.Access.ServiceQueryScan}
	read.cursor = listCursor{Area: area, Owner: found.owner, Caller: found.handle, Query: hex.EncodeToString(sum[:]), Basis: basis, Reference: now,
		Expires: queryExpiry(answer.NotAfter(), found.idleUntil, now, s.cfg.Access.ServiceQuerySeconds)}
	if after != "" {
		var prior listCursor
		if !s.openQueryToken(domain, after, &prior) || !read.cursor.binding().continues(prior.binding(), now) || prior.Reference.IsZero() || !prior.Reference.Before(prior.Expires) ||
			prior.Number == 0 && prior.After == "" || prior.After != "" && !validWorkspace(prior.After) || prior.Ended < 0 {
			failRead(w, commands.RevisionConflict, read.request, "after")
			return nil, false
		}
		read.cursor = prior
	}
	return read, true
}

func closeList[T any](s *Server, w http.ResponseWriter, read *listRead, domain, order string, rows []T, exhausted bool) {
	current, available := s.listBasis(read)
	if !available || current != read.cursor.Basis {
		failRead(w, commands.RevisionConflict, read.request, "after")
		return
	}
	page := listPage[T]{Rows: rows, SearchInput: read.values.Get("q"), Query: read.query, Reference: read.cursor.Reference.UTC().Format(time.RFC3339Nano), Observed: s.clock().UTC().Format(time.RFC3339Nano),
		Revision: s.Revision(), Order: order, Count: serviceQueryCount{Value: strconv.FormatUint(read.cursor.Matched, 10), Certainty: "exact"}, Scanned: read.budget.scanned}
	if !exhausted {
		next, sealed := s.sealQueryToken(domain, read.cursor)
		if !sealed {
			failCode(w, commands.InternalError, read.request)
			return
		}
		page.Next, page.Count.Certainty, page.ScanStopped = next, "lower_bound", read.budget.remaining() == 0
	}
	writeJSON(w, http.StatusOK, read.request, page)
}

func (s *Server) queryAccessRequests(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	read, ok := s.openList(w, r, "requests", requestsCursorDomain, []string{"pending", "ended"})
	if !ok {
		return
	}
	facts, failure := s.boundedFacts(read.found.verifier)
	you := read.found.owner.User
	rows := []listedRequest{}
	exhausted := false
	for len(rows) < read.query.Limit && !exhausted && read.budget.remaining() > 0 {
		views, more := s.changes.RequestPage(s.bootWorkspace, read.cursor.Number, min(read.budget.remaining(), s.cfg.Access.ServiceQueryPage), read.cursor.Reference)
		taken := 0
		for _, v := range views {
			if len(rows) == read.query.Limit {
				break
			}
			read.budget.reserve(1)
			taken++
			read.cursor.Number = v.Number
			pending := v.State == accesschange.StatePending
			if !pending {
				if read.cursor.Ended >= s.cfg.Access.EndedShown {
					continue
				}
				read.cursor.Ended++
			}
			row := facts.changeRequestRow(v, you)
			if read.query.State == "pending" && !pending || read.query.State == "ended" && pending ||
				read.query.Q != "" && !strings.Contains(strings.ToLower(row.ID), read.query.Q) && !strings.Contains(strings.ToLower(row.Subject), read.query.Q) {
				continue
			}
			if read.cursor.Matched == math.MaxUint64 {
				failCode(w, commands.DependencyUnavailable, read.request)
				return
			}
			read.cursor.Matched++
			listed := listedRequest{changeRequestRow: row}
			listed.SubjectLogin, _ = s.dir.loginOf(row.Subject)
			listed.ProposerLogin, _ = s.dir.loginOf(row.Proposer)
			listed.DecidedByLogin, _ = s.dir.loginOf(row.DecidedBy)
			rows = append(rows, listed)
		}
		if taken < len(views) {
			break
		}
		exhausted = !more
	}
	if *failure != nil {
		failCode(w, commands.DependencyUnavailable, read.request)
		return
	}
	closeList(s, w, read, requestsCursorDomain, requestsOrder, rows, exhausted)
}

func (s *Server) queryAccessRevocations(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	read, ok := s.openList(w, r, "revocations", revocationsCursorDomain, nil)
	if !ok {
		return
	}
	rows := []revocationRow{}
	exhausted := false
	for len(rows) < read.query.Limit && !exhausted && read.budget.remaining() > 0 {
		views, more, known := s.changes.RevocationPage(s.bootWorkspace, read.cursor.After, min(read.budget.remaining(), s.cfg.Access.ServiceQueryPage))
		if !known {
			failRead(w, commands.RevisionConflict, read.request, "after")
			return
		}
		taken := 0
		for _, v := range views {
			if len(rows) == read.query.Limit {
				break
			}
			read.budget.reserve(1)
			taken++
			read.cursor.After = v.ID
			if read.query.Q != "" && !strings.Contains(strings.ToLower(v.ID), read.query.Q) && !strings.Contains(strings.ToLower(v.Target.ID), read.query.Q) {
				continue
			}
			if read.cursor.Matched == math.MaxUint64 {
				failCode(w, commands.DependencyUnavailable, read.request)
				return
			}
			read.cursor.Matched++
			rows = append(rows, s.revocationRow(v))
		}
		if taken < len(views) {
			break
		}
		exhausted = !more
	}
	closeList(s, w, read, revocationsCursorDomain, revocationsOrder, rows, exhausted)
}

func accessListPaths() map[string]any {
	str := map[string]any{"type": "string"}
	entry := func(operation, order, listed, filter string, filters []string) map[string]any {
		row := map[string]any{"type": "object", "properties": map[string]any{"id": str}}
		return map[string]any{"get": map[string]any{"operationId": operation, "x-row": row, "x-filters": filters, "x-order": order,
			"responses": serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"rows": map[string]any{"type": "array", "items": row}}}, true),
			"x-listed":  listed, "x-filter": filter,
			"x-time":   "a row's state is its state at the reference time; what the caller may do with it is evaluated when the page is read, and the command decides; the facts behind a row are looked up per person and per holder, each lookup up to the scan limit, and 503 dependency_unavailable answers when one holds more",
			"x-cursor": "bound to the filters, the caller's session, the reference time and the versions of the access, membership and directory sources; 409 revision_conflict when it no longer fits"}}
	}
	return map[string]any{
		"/identity/access/requests": entry("identity.access.requests", requestsOrder,
			"access change requests of the caller's workspace as GET /identity/access lists them: every pending request and the most recent ended ones up to the installation's shown limit; each row as there, with the sign-in names of the subject, the proposer and the decider",
			"state is pending or ended; q matches the request identifier or the subject's user id without regard to case", []string{"q", "state", "limit", "after"}),
		"/identity/access/revocations": entry("identity.access.revocations", revocationsOrder,
			"kept revocations of the caller's workspace, each row as GET /identity/access lists it; target_subject, target_subject_login, target_action, target_resource_kind and target_fields describe the revoked grant (its holder) or delegation (its delegate) as the record is now, and are empty when it is no longer present", "q matches the revocation identifier or the target identifier without regard to case", []string{"q", "limit", "after"}),
	}
}
