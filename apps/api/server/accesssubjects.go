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
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	subjectsOrder        = "user_id_asc"
	subjectsCursorDomain = "access-subjects-v1"
)

type subjectQuery struct {
	Q     string `json:"q"`
	ID    string `json:"id"`
	Limit int    `json:"limit"`
}

type subjectCursor struct {
	Owner     sessionOwner `json:"owner"`
	Caller    string       `json:"caller"`
	Query     string       `json:"query"`
	Basis     string       `json:"basis"`
	Reference time.Time    `json:"reference"`
	Expires   time.Time    `json:"expires"`
	After     string       `json:"after"`
	Matched   uint64       `json:"matched"`
}

func (c subjectCursor) binding() queryBinding {
	return queryBinding{Area: "subjects", Owner: c.Owner, Caller: c.Caller, Query: c.Query, Basis: c.Basis, Reference: c.Reference, Expires: c.Expires}
}

type listedSubject struct {
	subjectRow
	RevocationUnconfirmed bool `json:"revocation_unconfirmed"`
}

type subjectsPage struct {
	Rows        []listedSubject   `json:"rows"`
	SearchInput string            `json:"search_input"`
	Query       subjectQuery      `json:"query"`
	Next        string            `json:"next"`
	Reference   string            `json:"reference_at"`
	Observed    string            `json:"observed_at"`
	Revision    string            `json:"revision"`
	Order       string            `json:"order"`
	Count       serviceQueryCount `json:"count"`
	Scanned     int               `json:"work_units"`
	ScanStopped bool              `json:"scan_stopped"`
}

func parseSubjectQuery(values url.Values, pageLimit int) (subjectQuery, string, string) {
	query := subjectQuery{Limit: min(25, pageLimit)}
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
		case "id":
			if !validWorkspace(value) {
				return query, "", name
			}
			query.ID = value
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
	if query.ID != "" && (query.Q != "" || after != "") {
		return query, "", "id"
	}
	return query, after, ""
}

func (s *Server) subjectsBasis(actor identity.Verifier) (string, bool) {
	people, err := s.identity.Workspace().ServiceQueryBasis(actor, s.bootWorkspace)
	if err != nil {
		return "", false
	}
	changes := s.changes.Version()
	if changes == math.MaxUint64 {
		return "", false
	}
	body, _ := json.Marshal(struct {
		Subjects   uint64
		Directory  uint64
		Changes    uint64
		Generation uint64
	}{people.Subjects, s.dir.version, changes, s.admin.Generation()})
	return hex.EncodeToString(s.queryMAC("access-subjects-source-v1", body)), true
}

func (s *Server) unconfirmedHolders(at time.Time) (map[string]bool, bool) {
	views, more := s.changes.UnconfirmedRevocations(s.bootWorkspace, s.cfg.Access.ServiceQueryScan)
	if more {
		return nil, false
	}
	out := map[string]bool{}
	for _, v := range views {
		if v.Target.Kind == accesschange.TargetDelegation {
			if d, ok := s.admin.DelegationAt(v.Target.ID, at); ok {
				out[d.DelegateID] = true
			}
		} else if g, ok := s.admin.GrantAt(v.Target.ID, at); ok && g.SubjectID != "" {
			out[g.SubjectID] = true
		}
	}
	return out, true
}

func (s *Server) queryAccessSubjects(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	if answer.Workspace() != s.bootWorkspace {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	if len(r.URL.RawQuery) > 8192 {
		invalidFields(w, request, "query")
		return
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		invalidFields(w, request, "query")
		return
	}
	query, after, field := parseSubjectQuery(values, s.cfg.Access.ServiceQueryPage)
	if field != "" {
		invalidFields(w, request, field)
		return
	}
	if s.faulty("access-query") || s.faulty("roster") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	basis, available := s.subjectsBasis(found.verifier)
	if !available {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	now := s.clock()
	raw, _ := json.Marshal(query)
	sum := sha256.Sum256(raw)
	cursor := subjectCursor{Owner: found.owner, Caller: found.handle, Query: hex.EncodeToString(sum[:]), Basis: basis, Reference: now,
		Expires: queryExpiry(answer.NotAfter(), found.idleUntil, now, s.cfg.Access.ServiceQuerySeconds)}
	if after != "" {
		var prior subjectCursor
		if !s.openQueryToken(subjectsCursorDomain, after, &prior) || !cursor.binding().continues(prior.binding(), now) || prior.After == "" || !validWorkspace(prior.After) ||
			prior.Reference.IsZero() || !prior.Reference.Before(prior.Expires) {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		cursor = prior
	}
	budget := scanBudget{limit: s.cfg.Access.ServiceQueryScan}
	unconfirmed, within := s.unconfirmedHolders(now)
	if !within {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	rows := []listedSubject{}
	exhausted := false
	if query.ID != "" {
		people, err := s.identity.Workspace().People(found.verifier, s.bootWorkspace, []string{query.ID})
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		budget.reserve(1)
		for _, person := range people {
			if person.Member != nil && person.Member.Effective && person.Member.UserState == identity.Active {
				login, _ := s.dir.loginOf(person.User.ID)
				rows, cursor.Matched = append(rows, listedSubject{subjectRow{UserID: person.User.ID, Login: login}, unconfirmed[person.User.ID]}), 1
			}
		}
		exhausted = true
	}
	for len(rows) < query.Limit && !exhausted && budget.remaining() > 0 {
		batch, err := s.identity.Workspace().MemberPositions(found.verifier, s.bootWorkspace, cursor.After, min(budget.remaining(), s.cfg.Identity.PageSize, s.cfg.Access.ServiceQueryPage))
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		budget.reserve(batch.Scanned)
		taken := 0
		for _, person := range batch.Rows {
			if len(rows) == query.Limit {
				break
			}
			taken++
			cursor.After = person.User.ID
			if person.Member == nil || !person.Member.Effective || person.Member.UserState != identity.Active {
				continue
			}
			login, _ := s.dir.loginOf(person.User.ID)
			if query.Q != "" && !strings.Contains(strings.ToLower(login), query.Q) && !strings.Contains(strings.ToLower(person.User.ID), query.Q) {
				continue
			}
			if cursor.Matched == math.MaxUint64 {
				failCode(w, commands.DependencyUnavailable, request)
				return
			}
			cursor.Matched++
			rows = append(rows, listedSubject{subjectRow{UserID: person.User.ID, Login: login}, unconfirmed[person.User.ID]})
		}
		if taken < len(batch.Rows) {
			break
		}
		cursor.After = batch.After
		exhausted = !batch.More
	}
	current, available := s.subjectsBasis(found.verifier)
	if !available || current != cursor.Basis {
		failRead(w, commands.RevisionConflict, request, "after")
		return
	}
	page := subjectsPage{Rows: rows, SearchInput: values.Get("q"), Query: query, Reference: cursor.Reference.UTC().Format(time.RFC3339Nano), Observed: s.clock().UTC().Format(time.RFC3339Nano),
		Revision: s.Revision(), Order: subjectsOrder, Count: serviceQueryCount{Value: strconv.FormatUint(cursor.Matched, 10), Certainty: "exact"}, Scanned: budget.scanned}
	if !exhausted {
		next, sealed := s.sealQueryToken(subjectsCursorDomain, cursor)
		if !sealed {
			failCode(w, commands.InternalError, request)
			return
		}
		page.Next, page.Count.Certainty, page.ScanStopped = next, "lower_bound", budget.remaining() == 0
	}
	writeJSON(w, http.StatusOK, request, page)
}

func subjectsReadPath() map[string]any {
	str := map[string]any{"type": "string"}
	row := map[string]any{"type": "object", "properties": map[string]any{"revocation_unconfirmed": map[string]any{"type": "boolean"}, "user_id": str, "login": map[string]any{"type": "string", "x-meaning": "the sign-in name as the directory has it; empty when there is none"}}}
	return map[string]any{"get": map[string]any{"operationId": "identity.access.subjects", "x-row": row, "x-filters": []string{"q", "id", "limit", "after"}, "x-order": subjectsOrder,
		"responses": serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"rows": map[string]any{"type": "array", "items": row}}}, true),
		"x-listed":  "members of the caller's workspace whose membership is in effect and whose account is active: the people a grant can be requested for",
		"x-filter":  "q matches the sign-in name or the user id without regard to case; id names exactly one person and takes neither q nor a cursor",
		"x-flag":    "revocation_unconfirmed is true when a kept revocation that is not confirmed targets a grant this person holds or a delegation this person receives; 503 dependency_unavailable when more such revocations are kept than the scan limit",
		"x-time":    "membership and account standing are read when each page is read, not at the reference time; changes recorded in the identity service end the cursor, but a membership period that begins or ends during a walk can change later pages without ending it",
		"x-cursor":  "bound to the filters, the caller's session, the reference time and the membership and directory versions; 409 revision_conflict when it no longer fits"}}
}
