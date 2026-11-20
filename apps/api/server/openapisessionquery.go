package server

import "github.com/myblueton-sw/hybrid_billing/modules/commands"

func sessionQueryRowSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"handle", "current", "signed_in", "ends", "state", "can_end", "not_included_operation"}, "properties": map[string]any{
		"handle": map[string]any{"type": "string", "pattern": "^[0-9a-f]{16}$"}, "current": map[string]any{"type": "boolean"},
		"signed_in": map[string]any{"type": "string", "format": "date-time"}, "ends": map[string]any{"type": "string", "format": "date-time"},
		"state": map[string]any{"type": "string", "enum": []string{"active", "blocked", "expired", "revoked", "unavailable"}}, "can_end": map[string]any{"type": "boolean"}, "not_included_operation": map[string]any{"type": "string", "description": "Retained accepted others-operation reference proving confirmed-set exclusion, or empty; no temporal or current-active assertion."},
	}, "x-search-fields": []string{"handle", "signed_in", "ends"}}
}

func sessionQueryOperation() map[string]any {
	text := map[string]any{"type": "string"}
	date := map[string]any{"type": "string", "format": "date-time", "maxLength": 40, "x-fractional-digits-max": 9}
	query := func(name string, schema map[string]any) map[string]any {
		return map[string]any{"name": name, "in": "query", "required": false, "schema": schema}
	}
	normalized := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"session", "workspace", "since", "until", "q", "limit"}, "properties": map[string]any{
		"session": map[string]any{"type": "string", "enum": []string{"all", "current", "other"}}, "workspace": text, "since": text, "until": text, "q": text, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}}}
	count := map[string]any{"type": "object", "required": []string{"value", "certainty"}, "properties": map[string]any{"value": map[string]any{"type": "integer", "minimum": 0},
		"certainty": map[string]any{"type": "string", "enum": []string{"exact", "lower_bound"}}}, "description": "Matching recorded sessions, including distinctly labelled inactive/unavailable records; not an active-session total. Lower-bound progress is not an exact final total."}
	setting := map[string]any{"type": "object", "required": []string{"name", "value", "unit", "default", "min", "max", "source"}, "properties": map[string]any{
		"name": text, "value": map[string]any{"type": "integer"}, "unit": text, "default": map[string]any{"type": "integer"}, "min": map[string]any{"type": "integer"}, "max": map[string]any{"type": "integer"}, "source": map[string]any{"type": "string", "enum": []string{SourceDefault, SourceStartFlag}}}}
	page := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"rows", "next", "query", "reference_at", "generation", "order", "count", "scan_stopped", "settings"}, "properties": map[string]any{
		"rows": map[string]any{"type": "array", "maxItems": 100, "items": sessionQueryRowSchema()}, "next": map[string]any{"type": "string", "maxLength": 4096}, "query": normalized, "reference_at": date, "generation": text,
		"order": map[string]any{"type": "string", "enum": []string{sessionOrder}}, "count": count, "scan_stopped": map[string]any{"type": "boolean"}, "settings": map[string]any{"type": "array", "maxItems": 3, "items": setting}}}
	return map[string]any{"operationId": "me.sessions.query", "x-order": sessionOrder, "x-max-query-bytes": 8192, "x-admission": "authenticated canonical owner, user epoch, current workspace and caller; current authority on every page",
		"x-continuation": "Signed process-local cursor binds normalized filters, page size, caller, owner epoch, workspace, index/domain revisions, reference time and earliest eligibility/lifetime transition. Empty rows with continuation or scan_stopped do not mean zero matches.",
		"parameters": []any{query("session", map[string]any{"type": "string", "enum": []string{"all", "current", "other"}, "default": "all"}), query("workspace", map[string]any{"type": "string", "x-allowed": "current workspace only"}),
			query("since", date), query("until", date), query("q", map[string]any{"type": "string", "minLength": 1, "x-max-bytes": 64}), query("limit", map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "x-default-setting": "session_query_page_size"}), query("after", map[string]any{"type": "string", "maxLength": 4096})},
		"responses": map[string]any{"200": map[string]any{"description": "Bounded recorded-session page with truthful count basis and configurable scan/lifetime settings", "content": map[string]any{"application/json": map[string]any{"schema": page}}},
			"401": selfError(string(commands.Unauthenticated), commands.NextAnswer, false), "403": selfError(CodeCSRF, commands.NextNone, false),
			"409": readError(commands.RevisionConflict, "after"), "422": selfError(string(commands.InvalidRequest), commands.NextCorrect, false), "500": selfError(string(commands.InternalError), commands.NextNone, false), "503": selfError(string(commands.AuthorityUnavailable), commands.NextRetry, true)}}
}

func sessionDetailOperation() map[string]any {
	return map[string]any{"operationId": "me.sessions.detail", "x-admission": "current owner/incarnation/workspace; independent of displayed list page",
		"parameters": []any{map[string]any{"name": "handle", "in": "path", "required": true, "schema": map[string]any{"type": "string", "pattern": "^[0-9a-f]{16}$"}}},
		"responses": map[string]any{"200": map[string]any{"description": "Current known-ID session state and permitted action", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{
			"type": "object", "additionalProperties": false, "required": []string{"session", "reference_at"}, "properties": map[string]any{"session": sessionQueryRowSchema(), "reference_at": map[string]any{"type": "string", "format": "date-time"}}}}}},
			"401": selfError(string(commands.Unauthenticated), commands.NextAnswer, false), "403": selfError(CodeCSRF, commands.NextNone, false),
			"404": selfError(string(commands.UnsupportedCommand), commands.NextNone, false), "422": selfError(string(commands.InvalidRequest), commands.NextCorrect, false), "500": selfError(string(commands.InternalError), commands.NextNone, false), "503": selfError(string(commands.AuthorityUnavailable), commands.NextRetry, true)}}
}
