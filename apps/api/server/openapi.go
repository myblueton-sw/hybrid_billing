package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const description = "Command catalogue served on loopback by the pre-launch API service under /api/v1. State is in memory and resets on restart. No operation is anonymous: every command needs a session cookie and the CSRF header."

func (s *Server) Document() ([]byte, error) {
	raw, err := s.commands.OpenAPI("Hybrid Billing pre-launch API", "0.1.0")
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	doc["servers"] = []any{map[string]any{"url": "/api/v1"}}
	info := doc["info"].(map[string]any)
	info["description"] = description
	listed := doc["x-errors"].([]any)
	props := doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"].(map[string]any)["properties"].(map[string]any)
	enum, next := props["code"].(map[string]any), props["next_action"].(map[string]any)
	codes := enum["enum"].([]any)
	for _, b := range boundaryCodes {
		listed = append(listed, map[string]any{"code": b.code, "status": b.status, "retry": false, "next_action": string(b.next)})
		codes = append(codes, b.code)
	}
	doc["x-errors"], enum["enum"], next["enum"] = listed, codes, append(next["enum"].([]any), NextReauthenticate)
	for path, item := range doc["paths"].(map[string]any) {
		if command, ok := strings.CutPrefix(path, "/commands/"); ok && s.stepUp(command) {
			item.(map[string]any)["post"].(map[string]any)["x-step-up"] = true
		}
	}
	doc["x-reads"] = map[string]any{"/identity/users": usersReadPath()}
	return json.MarshalIndent(doc, "", "  ")
}

func readCodes(entry map[string]any, codes ...string) map[string]any {
	sort.Strings(codes)
	entry["x-codes"] = codes
	return entry
}

func withCondition(entry map[string]any) map[string]any {
	entry["x-fields-when"] = "cursor"
	return entry
}

func readError(code commands.Code, fields ...string) map[string]any {
	next := readRules[code].next
	if rule, ok := moduleRules[code]; ok {
		next = rule.next
	}
	out := map[string]any{"description": string(code), "x-code": string(code), "x-next-action": string(next),
		"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Error"}}}}
	if len(fields) > 0 {
		out["x-fields"] = fields
	}
	return out
}

func usersReadPath() map[string]any {
	params := []any{map[string]any{"name": "q", "in": "query", "schema": map[string]any{"type": "string", "minLength": 1, "x-max-bytes": userSearchBytes}},
		map[string]any{"name": "after", "in": "query", "schema": map[string]any{"type": "string", "pattern": `^[ad]:[A-Za-z0-9._:-]{1,128}$`}},
		map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": userPageMax, "default": userPageDefault}}}
	for _, name := range []string{"status", "membership", "role", "period", "signin", "invitation", "holder"} {
		params = append(params, map[string]any{"name": name, "in": "query", "schema": map[string]any{"type": "string", "enum": userFilterValues[name]}})
	}
	return map[string]any{"get": map[string]any{"operationId": "identity.users", "parameters": params, "x-order": usersOrder,
		"x-refused-parameters": map[string]any{"method": string(commands.UnsupportedCapability), "workspace": string(commands.InvalidRequest)},
		"responses": map[string]any{"200": map[string]any{"description": "one page of people"},
			"401": readCodes(readError(commands.Unauthenticated), string(commands.Unauthenticated)),
			"403": readCodes(readError(commands.PermissionDenied), CodeCSRF, string(commands.PermissionDenied)),
			"409": readCodes(readError(commands.RevisionConflict, "after"), string(commands.RevisionConflict)),
			"422": readCodes(withCondition(readError(commands.InvalidRequest, "after")), string(commands.InvalidRequest)),
			"501": readCodes(readError(commands.UnsupportedCapability), string(commands.UnsupportedCapability)),
			"503": readCodes(readError(commands.DependencyUnavailable), string(commands.AuthorityUnavailable), string(commands.DependencyUnavailable))}}}
}

func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	doc, err := s.Document()
	s.domain.RUnlock()
	if err != nil {
		failCode(w, commands.InternalError, request)
		return
	}
	headers(w, request)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}
