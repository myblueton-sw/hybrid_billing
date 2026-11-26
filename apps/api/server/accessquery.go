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
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

const (
	sourceGrant           = "grant"
	sourceDelegation      = "delegation"
	permissionsOrder      = "source_then_id"
	accessCursorDomain    = "access-query-v1"
	accessBasisDomain     = "access-source-v1"
	permissionSearchBytes = 64
)

var permissionStates = []string{"active", "pending", "expired", "revoked", stateUnconfirmed, "ended"}

type permissionRow struct {
	Source     string   `json:"source"`
	ID         string   `json:"id"`
	Subject    string   `json:"subject"`
	Client     string   `json:"client"`
	Delegator  string   `json:"delegator"`
	Parent     string   `json:"parent"`
	Redelegate bool     `json:"redelegate"`
	Action     string   `json:"action"`
	Kind       string   `json:"kind"`
	Fields     []string `json:"fields"`
	Scope      scopeRow `json:"scope"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	State      string   `json:"state"`
	Effective  bool     `json:"effective"`
	Version    uint64   `json:"version"`
}

type permissionQuery struct {
	Q             string `json:"q"`
	Subject       string `json:"subject"`
	Action        string `json:"action"`
	Customer      string `json:"customer"`
	LegalEntity   string `json:"legal_entity"`
	Source        string `json:"source"`
	State         string `json:"state"`
	ExpiresBefore string `json:"expires_before"`
	Limit         int    `json:"limit"`
	expires       time.Time
}

type permissionCursor struct {
	Area      string       `json:"area"`
	Owner     sessionOwner `json:"owner"`
	Caller    string       `json:"caller"`
	Query     string       `json:"query"`
	Basis     string       `json:"basis"`
	Reference time.Time    `json:"reference"`
	Expires   time.Time    `json:"expires"`
	Source    string       `json:"source"`
	After     string       `json:"after"`
	Matched   uint64       `json:"matched"`
}

func (c permissionCursor) binding() queryBinding {
	return queryBinding{Area: c.Area, Owner: c.Owner, Caller: c.Caller, Query: c.Query, Basis: c.Basis, Reference: c.Reference, Expires: c.Expires}
}

type permissionsPage struct {
	Rows        []permissionRow   `json:"rows"`
	SearchInput string            `json:"search_input"`
	Query       permissionQuery   `json:"query"`
	Next        string            `json:"next"`
	Reference   string            `json:"reference_at"`
	Observed    string            `json:"observed_at"`
	Revision    string            `json:"revision"`
	Order       string            `json:"order"`
	Count       serviceQueryCount `json:"count"`
	Scanned     int               `json:"work_units"`
	ScanStopped bool              `json:"scan_stopped"`
}

type permissionDetail struct {
	Workspace string        `json:"workspace"`
	Row       permissionRow `json:"row"`
	Revision  string        `json:"revision"`
	Observed  string        `json:"observed_at"`
}

func parsePermissionQuery(values url.Values, pageLimit int) (permissionQuery, string, string) {
	query := permissionQuery{Limit: min(25, pageLimit)}
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
		case "subject":
			if !validWorkspace(value) {
				return query, "", name
			}
			query.Subject = value
		case "action":
			if !validWorkspace(value) {
				return query, "", name
			}
			query.Action = value
		case "customer":
			if !validWorkspace(value) {
				return query, "", name
			}
			query.Customer = value
		case "legal_entity":
			if !validWorkspace(value) {
				return query, "", name
			}
			query.LegalEntity = value
		case "source":
			if value != sourceGrant && value != sourceDelegation {
				return query, "", name
			}
			query.Source = value
		case "state":
			if !oneServiceValue(value, permissionStates...) {
				return query, "", name
			}
			query.State = value
		case "expires_before":
			at, err := time.Parse(time.RFC3339, value)
			if err != nil || at.UTC().Format(time.RFC3339) != value {
				return query, "", name
			}
			query.ExpiresBefore, query.expires = value, at
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

func (q permissionQuery) digest() string {
	raw, _ := json.Marshal(q)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func selectorApplies(stored, wanted string) bool {
	return wanted == "" || stored == wanted || stored == "all"
}

func (q permissionQuery) matches(row permissionRow, notAfter time.Time) bool {
	if q.Source != "" && row.Source != q.Source || q.Subject != "" && row.Subject != q.Subject || q.Action != "" && row.Action != q.Action ||
		!selectorApplies(row.Scope.Customer, q.Customer) || !selectorApplies(row.Scope.LegalEntity, q.LegalEntity) || q.State != "" && row.State != q.State {
		return false
	}
	if q.ExpiresBefore != "" && !notAfter.Before(q.expires) {
		return false
	}
	return q.Q == "" || strings.Contains(strings.ToLower(row.ID), q.Q) || strings.Contains(strings.ToLower(row.Subject), q.Q)
}

func (s *Server) accessBasis() (string, bool) {
	version := s.changes.Version()
	if version == math.MaxUint64 {
		return "", false
	}
	body, _ := json.Marshal(struct {
		Generation uint64
		Changes    uint64
		Held       bool
	}{s.admin.Generation(), version, s.held()})
	return hex.EncodeToString(s.queryMAC(accessBasisDomain, body)), true
}

func grantPermission(g gate.GrantView, at time.Time, unconfirmed map[string]bool) permissionRow {
	return permissionRow{Source: sourceGrant, ID: g.ID, Subject: g.SubjectID, Client: g.ClientID, Action: string(g.Terms.Action), Kind: g.Terms.Kind,
		Fields: append([]string{}, g.Terms.Fields...), Scope: scopeOf(g.Terms), NotBefore: stamp(g.Validity.NotBefore), NotAfter: stamp(g.Validity.NotAfter),
		State: closedState(stateOf(g.Closed, g.Validity, at), unconfirmed[accesschange.TargetGrant+"/"+g.ID]), Effective: g.Effective, Version: g.Version}
}

func delegationPermission(d gate.DelegationView, at time.Time, unconfirmed map[string]bool) permissionRow {
	state := closedState(stateOf(d.Closed, d.Validity, at), unconfirmed[accesschange.TargetDelegation+"/"+d.ID])
	if state == "active" && !d.Effective {
		state = "ended"
	}
	return permissionRow{Source: sourceDelegation, ID: d.ID, Subject: d.DelegateID, Delegator: d.DelegatorID, Parent: d.ParentID, Redelegate: d.AllowRedelegation,
		Action: string(d.Terms.Action), Kind: d.Terms.Kind, Fields: append([]string{}, d.Terms.Fields...), Scope: scopeOf(d.Terms),
		NotBefore: stamp(d.Validity.NotBefore), NotAfter: stamp(d.Validity.NotAfter), State: state, Effective: d.Effective, Version: d.Version}
}

func (s *Server) queryPermissions(w http.ResponseWriter, r *http.Request) {
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
	query, after, field := parsePermissionQuery(values, s.cfg.Access.ServiceQueryPage)
	if field != "" {
		invalidFields(w, request, field)
		return
	}
	if s.faulty("access-query") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	basis, available := s.accessBasis()
	if !available {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	now := s.clock()
	cursor := permissionCursor{Area: "permissions", Owner: found.owner, Caller: found.handle, Query: query.digest(), Basis: basis, Reference: now,
		Expires: queryExpiry(answer.NotAfter(), found.idleUntil, now, s.cfg.Access.ServiceQuerySeconds), Source: sourceGrant}
	if query.Source == sourceDelegation {
		cursor.Source = sourceDelegation
	}
	if after != "" {
		var prior permissionCursor
		if !s.openQueryToken(accessCursorDomain, after, &prior) || !cursor.binding().continues(prior.binding(), now) || prior.Source != sourceGrant && prior.Source != sourceDelegation ||
			prior.After != "" && !validWorkspace(prior.After) || prior.Reference.IsZero() || !prior.Reference.Before(prior.Expires) {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		cursor = prior
	}
	budget := scanBudget{limit: s.cfg.Access.ServiceQueryScan}
	unconfirmed := s.unconfirmedTargets()
	rows := []permissionRow{}
	exhausted := false
	for len(rows) < query.Limit && !exhausted && budget.remaining() > 0 {
		batch := min(budget.remaining(), s.cfg.Access.ServiceQueryPage)
		var candidates []permissionRow
		var ends []time.Time
		more := false
		if cursor.Source == sourceGrant {
			var page []gate.GrantView
			page, more = s.admin.GrantPage(cursor.After, batch, cursor.Reference)
			for _, g := range page {
				if gate.InstallationAction(g.Terms.Action) {
					candidates, ends = append(candidates, permissionRow{ID: g.ID}), append(ends, time.Time{})
					continue
				}
				candidates, ends = append(candidates, grantPermission(g, cursor.Reference, unconfirmed)), append(ends, g.Validity.NotAfter)
			}
		} else {
			var page []gate.DelegationView
			page, more = s.admin.DelegationPage(cursor.After, batch, cursor.Reference)
			for _, d := range page {
				candidates, ends = append(candidates, delegationPermission(d, cursor.Reference, unconfirmed)), append(ends, d.Validity.NotAfter)
			}
		}
		taken := 0
		for i, row := range candidates {
			if len(rows) == query.Limit {
				break
			}
			budget.reserve(1)
			taken++
			cursor.After = row.ID
			if row.Source == "" || !query.matches(row, ends[i]) {
				continue
			}
			if cursor.Matched == math.MaxUint64 {
				failCode(w, commands.DependencyUnavailable, request)
				return
			}
			cursor.Matched++
			rows = append(rows, row)
		}
		if taken < len(candidates) {
			break
		}
		if !more {
			if cursor.Source == sourceGrant && query.Source != sourceGrant {
				cursor.Source, cursor.After = sourceDelegation, ""
			} else {
				exhausted = true
			}
		}
	}
	current, available := s.accessBasis()
	if !available || current != cursor.Basis {
		failRead(w, commands.RevisionConflict, request, "after")
		return
	}
	page := permissionsPage{Rows: rows, SearchInput: values.Get("q"), Query: query, Reference: cursor.Reference.UTC().Format(time.RFC3339Nano), Observed: s.clock().UTC().Format(time.RFC3339Nano),
		Revision: s.Revision(), Order: permissionsOrder, Count: serviceQueryCount{Value: strconv.FormatUint(cursor.Matched, 10), Certainty: "exact"}, Scanned: budget.scanned}
	if !exhausted {
		next, sealed := s.sealQueryToken(accessCursorDomain, cursor)
		if !sealed {
			failCode(w, commands.InternalError, request)
			return
		}
		page.Next, page.Count.Certainty, page.ScanStopped = next, "lower_bound", budget.remaining() == 0
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) readPermission(source string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := r.Header.Get("X-Request-Id")
		s.domain.RLock()
		defer s.domain.RUnlock()
		_, answer, ok := s.admitted(w, r)
		if !ok {
			return
		}
		id := r.PathValue("id")
		if answer.Workspace() != s.bootWorkspace || len(r.URL.RawQuery) != 0 || !validWorkspace(id) {
			failCode(w, commands.InvalidRequest, request)
			return
		}
		if s.faulty("access-query") {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		now := s.clock()
		unconfirmed := s.unconfirmedTargets()
		var row permissionRow
		known := false
		if source == sourceGrant {
			var g gate.GrantView
			if g, known = s.admin.GrantAt(id, now); known && !gate.InstallationAction(g.Terms.Action) {
				row = grantPermission(g, now, unconfirmed)
			} else {
				known = false
			}
		} else {
			var d gate.DelegationView
			if d, known = s.admin.DelegationAt(id, now); known {
				row = delegationPermission(d, now, unconfirmed)
			}
		}
		if !known {
			failCode(w, commands.InvalidRequest, request)
			return
		}
		writeJSON(w, http.StatusOK, request, permissionDetail{Workspace: s.bootWorkspace, Row: row, Revision: s.Revision(), Observed: now.UTC().Format(time.RFC3339Nano)})
	}
}

func permissionReadPaths() map[string]any {
	str := map[string]any{"type": "string"}
	boolean := map[string]any{"type": "boolean"}
	row := map[string]any{"type": "object", "properties": map[string]any{
		"source": map[string]any{"type": "string", "enum": []string{sourceGrant, sourceDelegation}}, "id": str, "subject": map[string]any{"type": "string", "x-meaning": "the holder: a grant's subject or a delegation's delegate; empty for a client grant"},
		"client": str, "delegator": str, "parent": map[string]any{"type": "string", "x-meaning": "the delegation this one was redelegated from, if any"}, "redelegate": boolean, "action": str, "kind": str,
		"fields": map[string]any{"type": "array", "items": str}, "scope": map[string]any{"type": "object"}, "not_before": str, "not_after": str,
		"state": map[string]any{"type": "string", "enum": permissionStates, "x-meaning": "evaluated at reference_at in the list and at observed_at in a detail read"}, "effective": boolean, "version": map[string]any{"type": "integer"}}}
	filters := []string{"q", "subject", "action", "customer", "legal_entity", "source", "state", "expires_before", "limit", "after"}
	return map[string]any{
		"/identity/access/permissions": map[string]any{"get": map[string]any{"operationId": "identity.access.permissions", "x-row": row, "x-filters": filters, "x-order": permissionsOrder,
			"responses":   serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"rows": map[string]any{"type": "array", "items": row}}}, true),
			"x-work":      "work_units counts every stored row examined, including grants that are not listed",
			"x-cursor":    "bound to the list, the filters, the caller's session, the reference time and the source basis; 409 revision_conflict when it no longer fits",
			"x-count":     "exact on the last page, a lower bound before it",
			"x-never":     "grants for installation actions are not listed",
			"x-filter-id": "customer and legal_entity list every permission that applies to the named one: those scoped to it and those that cover every one; the value all lists only those that cover every one"}},
		"/identity/access/permissions/grant/{id}":      map[string]any{"get": map[string]any{"operationId": "identity.access.grant", "x-row": row, "responses": serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"row": row}}, false), "x-unknown": "an unknown identifier, a grant for an installation action and a delegation's identifier answer alike"}},
		"/identity/access/permissions/delegation/{id}": map[string]any{"get": map[string]any{"operationId": "identity.access.delegation", "x-row": row, "responses": serviceReadResponses(map[string]any{"type": "object", "properties": map[string]any{"row": row}}, false), "x-unknown": "an unknown identifier and a grant's identifier answer alike"}},
	}
}
