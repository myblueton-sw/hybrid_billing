package server

import "github.com/myblueton-sw/hybrid_billing/modules/commands"

func sessionOperationSummarySchema() map[string]any {
	text := map[string]any{"type": "string"}
	count := map[string]any{"type": "integer", "minimum": 0}
	unknownCount := map[string]any{"type": []string{"integer", "null"}, "minimum": 0}
	properties := map[string]any{
		"operation": text, "selection": text, "kind": map[string]any{"type": "string", "enum": []string{kindOne, kindOthers}},
		"state":          map[string]any{"type": "string", "enum": []string{"unconfirmed", stateRunning, stateComplete, stateIncomplete}},
		"recorded_count": count, "ended_count": unknownCount, "gone_count": unknownCount, "not_included_count": unknownCount,
		"blocked_at": map[string]any{"type": []string{"string", "null"}}, "list_generation": text,
		"current_generation": map[string]any{"type": []string{"string", "null"}}, "generation_changed": map[string]any{"type": []string{"boolean", "null"}},
		"digest": text, "replay": map[string]any{"type": "boolean"},
		"cleanup": map[string]any{"type": []string{"object", "null"}, "properties": map[string]any{"done": count, "already_ended": count, "failed": count, "pending": count}},
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties,
		"required":    []string{"operation", "selection", "kind", "state", "recorded_count", "ended_count", "gone_count", "not_included_count", "blocked_at", "list_generation", "current_generation", "generation_changed", "digest", "cleanup", "replay"},
		"description": "Bounded operation summary. Unconfirmed intent reports null outcome, cleanup and current-generation fields; recorded count is the frozen target count, never an assertion of effects."}
}

func sessionSelectionSchema(preview bool) map[string]any {
	text := map[string]any{"type": "string"}
	properties := map[string]any{"kind": map[string]any{"type": "string", "enum": []string{kindOne, kindOthers}}, "selection": text, "operation": text, "state": map[string]any{"type": "string", "enum": []string{"building", "sealed", "applying", "accepted"}},
		"reference_at": text, "expires_at": text, "generation": text, "observed": map[string]any{"type": "integer", "minimum": 0},
		"count": map[string]any{"type": []string{"integer", "null"}, "minimum": 0}, "digest": text, "after": text,
		"settings": map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{"type": "object"}}}
	required := []string{"kind", "selection", "state", "reference_at", "expires_at", "generation", "observed", "count", "digest", "after", "settings"}
	if preview {
		properties["rows"] = map[string]any{"type": "array", "maxItems": 100, "items": sessionQueryRowSchema()}
		properties["next"] = text
		required = append(required, "rows", "next")
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required,
		"description": "Building count is null and observed is progress. Sealed count is exact for immutable recorded targets. Preview describes the frozen observation, not current eligibility. Settings expose page, scan, lifetime, pending, targets and global record budget with effective/default/source values."}
}

func sessionSelectionResponses(schema map[string]any) map[string]any {
	return map[string]any{"200": map[string]any{"description": "Authenticated scoped bounded result", "content": map[string]any{"application/json": map[string]any{"schema": schema}}},
		"401": selfError(string(commands.Unauthenticated), commands.NextAnswer, false), "403": selfError(CodeCSRF, commands.NextNone, false),
		"404": selfError(string(commands.UnsupportedCommand), commands.NextNone, false), "409": readError(commands.RevisionConflict, "after"),
		"413": selfError(CodeBodyTooLarge, commands.NextNone, false), "422": selfError(string(commands.InvalidRequest), commands.NextCorrect, false),
		"429": selfError(string(commands.RateLimited), commands.NextRetry, true), "500": selfError(string(commands.InternalError), commands.NextNone, false),
		"503": selfError(string(commands.AuthorityUnavailable), commands.NextRetry, true)}
}

func sessionSelectionPrepareOperation() map[string]any {
	body := map[string]any{"oneOf": []any{
		map[string]any{"type": "object", "additionalProperties": false, "maxProperties": 0},
		map[string]any{"type": "object", "additionalProperties": false, "required": []string{"session"}, "properties": map[string]any{"session": map[string]any{"type": "string", "pattern": "^[0-9a-f]{16}$"}}},
		map[string]any{"type": "object", "additionalProperties": false, "required": []string{"selection", "after"}, "properties": map[string]any{
			"selection": map[string]any{"type": "string", "minLength": 1, "maxLength": 64}, "after": map[string]any{"type": "string", "minLength": 1, "maxLength": 4096}}}}}
	return map[string]any{"operationId": "me.sessions.selection.prepare", "x-max-body-bytes": 8192,
		"x-admission":    "Current canonical owner, epoch, workspace and original caller; CSRF required. No query parameters. Duplicate/unknown JSON keys and null scalars are rejected.",
		"x-continuation": "Empty object prepares all other recorded sessions; a session handle freezes exactly one noncurrent own session immediately under the same limits, with kind one. Kind and target participate in reuse/digest/application; no inferred target reconstruction. Signed process-local resumable scan for kind others. Source revision, eligibility transition or expiry invalidates unfinished preparation. Duplicate continuation does not append twice. Global quota covers building, sealed, accepted, result, held and retry references; capacity is reserved before effects.",
		"requestBody":    map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": body}}},
		"responses":      sessionSelectionResponses(sessionSelectionSchema(false))}
}

func sessionPageParameters() []any {
	return []any{map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "x-default-setting": "session_query_page_size"}},
		map[string]any{"name": "after", "in": "query", "schema": map[string]any{"type": "string", "minLength": 1, "maxLength": 4096}}}
}

func sessionSelectionPreviewOperation() map[string]any {
	parameters := append(sessionPageParameters(), map[string]any{"name": "reference", "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
	return map[string]any{"operationId": "me.sessions.selection.preview", "parameters": parameters, "x-max-query-bytes": 8192,
		"x-admission":    "Original caller plus current canonical owner/epoch/workspace. Missing/foreign references are indistinguishable 404. Accepted references survive preparation expiry while retained; acceptance replay uses the identical selection/digest/count before live generation or preparation-expiry checks.",
		"x-continuation": "Signed cursor binds reference, immutable digest, caller, owner, page size and lifetime. Building preview has no target rows. Unknown/duplicate/empty query values are rejected.",
		"responses":      sessionSelectionResponses(sessionSelectionSchema(true))}
}

func sessionOperationReadContract(kind string) map[string]any {
	text := map[string]any{"type": "string"}
	properties := map[string]any{"next": text, "reference_at": text, "generation": text}
	parameters := sessionPageParameters()
	schema := sessionOperationSummarySchema()
	if kind == "list" {
		properties["rows"] = map[string]any{"type": "array", "maxItems": 100, "items": schema}
		properties["accepted_count"] = map[string]any{"type": "integer", "minimum": 0, "description": "Exact retained accepted operations for this canonical owner/epoch/workspace; excludes unconfirmed intent."}
		schema = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"rows", "accepted_count", "next", "reference_at", "generation"}, "properties": properties}
	} else {
		parameters = append(parameters, map[string]any{"name": "operation", "in": "path", "required": true, "schema": text})
		if kind == "summary" {
			parameters = parameters[len(parameters)-1:]
		} else {
			parameters = append(parameters, map[string]any{"name": "view", "in": "query", "schema": map[string]any{"type": "string", "enum": []string{"results", "not_included"}, "default": "results"}})
			legacy := selfEndSchema()["properties"].(map[string]any)
			properties["operation"], properties["view"] = text, text
			properties["count"] = map[string]any{"type": "integer", "minimum": 0}
			properties["rows"] = map[string]any{"type": "array", "maxItems": 100, "items": map[string]any{"oneOf": []any{legacy["sessions"].(map[string]any)["items"], legacy["not_included"].(map[string]any)["items"]}}}
			schema = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"operation", "view", "rows", "count", "next", "reference_at", "generation"}, "properties": properties}
		}
	}
	return map[string]any{"operationId": "me.sessions.operations." + kind, "parameters": parameters, "x-max-query-bytes": 8192,
		"x-admission":    "Current canonical owner/epoch/workspace. Accepted operation recovery permits another session of the same owner; selection mutation remains original-caller bound. Unconfirmed intent is visible only to its original caller, with null outcomes; its item read returns 503. Missing/foreign operations return 404.",
		"x-continuation": "Signed cursor binds caller, owner, view, page size and retained-operation version. Acceptance, eviction and cleanup changes invalidate cursors with 409. Summary accepts no query. Lists contain accepted operations only. Unknown/duplicate/empty query values are rejected.",
		"responses":      sessionSelectionResponses(schema)}
}
