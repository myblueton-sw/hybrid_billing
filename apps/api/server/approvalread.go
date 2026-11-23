package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type approvalSelection struct {
	Kind string `json:"kind"`
	Q    string `json:"q"`
}
type approvalPage struct {
	Schema     int               `json:"schema"`
	Numeric    string            `json:"numeric_format"`
	ReadOnly   bool              `json:"read_only"`
	Commands   string            `json:"commands"`
	Source     string            `json:"source_mode"`
	Scope      string            `json:"auth_scope"`
	Snapshot   string            `json:"snapshot"`
	Permission string            `json:"permission_revision"`
	Actor      approvalPrincipal `json:"actor"`
	View       string            `json:"view"`
	Selection  approvalSelection `json:"selection"`
	Cursor     *string           `json:"cursor"`
	Next       *string           `json:"next_cursor"`
	Observed   string            `json:"observed_at"`
	Total      int               `json:"total"`
	Records    []approvalRow     `json:"records"`
}

func approvalDigest(v any) string { b, _ := json.Marshal(v); return approvalHash(string(b)) }
func approvalHex(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && s == strings.ToLower(s)
}
func approvalQuery(q url.Values) bool {
	for key, values := range q {
		if len(values) != 1 || !contains([]string{"view", "kind", "q", "cursor", "snapshot", "permission_revision"}, key) {
			return false
		}
	}
	if !contains([]string{"pending", "decisions", "history", "recognition", "delegation"}, q.Get("view")) || !contains([]string{"", "billing_result", "final_erp_amount", "adjustment", "policy", "monthly_recognition"}, q.Get("kind")) {
		return false
	}
	term := q.Get("q")
	if !utf8.ValidString(term) || len(utf16.Encode([]rune(term))) > 128 {
		return false
	}
	for _, c := range term {
		if c < 32 || c == 127 {
			return false
		}
	}
	for _, key := range []string{"snapshot", "permission_revision"} {
		if q.Has(key) && !approvalHex(q.Get(key)) {
			return false
		}
	}
	if q.Has("cursor") && (len(q.Get("cursor")) > 128 || q.Get("cursor") == "" || !q.Has("snapshot") || !q.Has("permission_revision")) {
		return false
	}
	return q.Get("view") != "recognition" || q.Get("kind") == "monthly_recognition"
}

func (s *Server) approvalCursor(offset int, snapshot string) string {
	n := strconv.Itoa(offset)
	mac := hmac.New(sha256.New, s.approvalKey)
	b, _ := json.Marshal([]string{n, snapshot})
	mac.Write(b)
	return n + "_" + hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) readApprovals(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return
	}
	decision := s.gate.Decide(answer, ConsoleClient, gate.Request{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Resource: gate.Resource{WorkspaceID: answer.Workspace()}})
	if !decision.Allowed {
		failCode(w, commands.PermissionDenied, request)
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || !approvalQuery(q) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	if q.Get("view") == "delegation" || q.Get("view") == "recognition" || contains([]string{"adjustment", "policy", "monthly_recognition"}, q.Get("kind")) {
		failModule(w, commands.UnsupportedCapability, http.StatusNotImplemented, false, commands.NextNone, request)
		return
	}
	rows, err := s.approvalRows(answer.Workspace())
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	filtered := []approvalRow{}
	for _, row := range rows {
		if q.Get("kind") != "" && row.Kind != q.Get("kind") {
			continue
		}
		if q.Get("view") == "pending" && (row.Status == "approved" || row.Status == "rejected") {
			continue
		}
		if q.Get("view") == "decisions" && (row.Approver == nil || row.Approver.ID != answer.User()) {
			continue
		}
		if term := strings.ToLower(q.Get("q")); !strings.Contains(strings.ToLower(row.ID+" "+row.Target.ID), term) {
			continue
		}
		filtered = append(filtered, row)
	}
	scope := approvalDigest([]string{hex.EncodeToString(found.verifier[:]), answer.Workspace(), answer.User()})
	permission := approvalDigest([]any{scope, s.admin.Generation(), s.identity.Subjects().Generation(), answer.Epoch(), answer.MembershipVersion()})
	snapshot := approvalDigest([]any{scope, permission, s.Revision(), q.Get("view"), q.Get("kind"), q.Get("q"), filtered})
	if (q.Has("snapshot") && q.Get("snapshot") != snapshot) || (q.Has("permission_revision") && q.Get("permission_revision") != permission) {
		fail(w, http.StatusConflict, string(commands.RevisionConflict), request)
		return
	}
	offset := 0
	if q.Has("cursor") {
		n, _, ok := strings.Cut(q.Get("cursor"), "_")
		offset, err = strconv.Atoi(n)
		if !ok || err != nil || offset < 25 || offset%25 != 0 || offset >= len(filtered) || !hmac.Equal([]byte(q.Get("cursor")), []byte(s.approvalCursor(offset, snapshot))) {
			failCode(w, commands.InvalidRequest, request)
			return
		}
	}
	end := min(offset+25, len(filtered))
	page := approvalPage{Schema: 1, Numeric: "decimal-strings-read-v1", ReadOnly: true, Commands: "unconnected", Source: "synthetic", Scope: scope, Snapshot: snapshot, Permission: permission, Actor: approvalPrincipal{ID: answer.User(), Label: answer.User()}, View: q.Get("view"), Selection: approvalSelection{Kind: q.Get("kind"), Q: q.Get("q")}, Cursor: approvalString(q.Get("cursor")), Observed: stamp(s.clock()), Total: len(filtered), Records: filtered[offset:end]}
	if end < len(filtered) {
		page.Next = approvalString(s.approvalCursor(end, snapshot))
	}
	writeJSON(w, http.StatusOK, request, page)
}
