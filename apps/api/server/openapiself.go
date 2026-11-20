package server

import (
	"sort"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func selfEndSchema() map[string]any {
	str := map[string]any{"type": "string"}
	item := map[string]any{"type": "object", "properties": map[string]any{"item": str, "state": map[string]any{"type": "string",
		"enum": []string{itemDone, itemPending, itemFailed, itemCleared, itemOutside, itemNotApply}}, "reason": str}}
	row := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"handle", "signed_in", "result", "items"}, "properties": map[string]any{"handle": str, "signed_in": str,
		"result": map[string]any{"type": "string", "enum": []string{resultEnded, resultGone}}, "items": map[string]any{"type": "array", "items": item}}}
	ref := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"handle", "signed_in"}, "properties": map[string]any{"handle": str, "signed_in": str}}
	counts := map[string]any{"type": "object", "properties": map[string]any{"done": map[string]any{"type": "integer"}, "already_ended": map[string]any{"type": "integer"},
		"failed": map[string]any{"type": "integer"}, "pending": map[string]any{"type": "integer"}}}
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"operation", "kind", "count", "state", "blocked_at", "list_generation", "current_generation", "generation_changed", "digest", "cleanup", "sessions", "not_included", "replay"}, "properties": map[string]any{"operation": str, "kind": map[string]any{"type": "string", "enum": []string{kindOthers, kindOne}},
		"count": map[string]any{"type": "integer"}, "state": map[string]any{"type": "string", "enum": []string{stateRunning, stateComplete, stateIncomplete}},
		"blocked_at": str, "list_generation": str, "current_generation": str, "generation_changed": map[string]any{"type": "boolean"}, "digest": str,
		"cleanup": counts, "sessions": map[string]any{"type": "array", "items": row}, "not_included": map[string]any{"type": "array", "items": ref},
		"replay": map[string]any{"type": "boolean"}}}
}

func selfPaths() map[string]any {
	hexOf := func(n int) map[string]any {
		return map[string]any{"type": "string", "pattern": "^[0-9a-f]{" + strconv.Itoa(n) + "}$"}
	}
	body := map[string]any{"oneOf": []any{
		map[string]any{"type": "object", "additionalProperties": false, "required": []string{"session", "generation", "handles"}, "properties": map[string]any{
			"session": map[string]any{"type": "string", "enum": []string{kindOthers}}, "generation": hexOf(32),
			"handles": map[string]any{"type": "array", "minItems": 1, "maxItems": selfEndHandles, "uniqueItems": true, "items": hexOf(16)}}},
		map[string]any{"type": "object", "additionalProperties": false, "required": []string{"session"}, "properties": map[string]any{"session": hexOf(16)}},
		map[string]any{"type": "object", "additionalProperties": false, "required": []string{"selection", "digest", "count"}, "properties": map[string]any{
			"selection": map[string]any{"type": "string", "maxLength": 64}, "digest": hexOf(64), "count": map[string]any{"type": "integer", "minimum": 1, "maximum": 10000}}}}}
	end := map[string]any{"operationId": "me.sessions.end", "x-step-up": true,
		"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": body}}},
		"responses": map[string]any{"202": map[string]any{"description": "access blocked for the confirmed sessions that were still present; cleanup follows",
			"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"oneOf": []any{selfEndSchema(), sessionOperationSummarySchema()}}}}},
			"404": selfError(string(commands.UnsupportedCommand), commands.NextNone, false),
			"409": selfError(string(commands.RevisionConflict), commands.NextInspect, false),
			"429": selfError(string(commands.RateLimited), commands.NextRetry, true),
			"401": selfError(string(commands.Unauthenticated), commands.NextAnswer, false),
			"403": selfError(CodeReauthentication, NextReauthenticate, false, CodeCSRF),
			"413": selfError(CodeBodyTooLarge, commands.NextNone, false),
			"422": selfError(string(commands.InvalidRequest), commands.NextCorrect, false, string(commands.PrerequisiteFailed)),
			"500": selfError(string(commands.InternalError), commands.NextNone, false),
			"503": selfError(string(commands.AuthorityUnavailable), commands.NextRetry, true)}}
	read := map[string]any{"operationId": "me.read", "responses": map[string]any{"200": map[string]any{"description": "the caller's own account and sessions",
		"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{
			"sessions_generation": hexOf(32), "session_ends": map[string]any{"type": "array", "maxItems": selfEndsKept, "items": selfEndSchema()}}}}}},
		"401": selfError(string(commands.Unauthenticated), commands.NextAnswer, false),
		"403": selfError(CodeCSRF, commands.NextNone, false),
		"500": selfError(string(commands.InternalError), commands.NextNone, false),
		"503": selfError(string(commands.AuthorityUnavailable), commands.NextRetry, true)}}
	return map[string]any{"/me/sessions/selections": map[string]any{"post": sessionSelectionPrepareOperation()},
		"/me/sessions/selections/{reference}":       map[string]any{"get": sessionSelectionPreviewOperation()},
		"/me/sessions/operations":                   map[string]any{"get": sessionOperationReadContract("list")},
		"/me/sessions/operations/{operation}":       map[string]any{"get": sessionOperationReadContract("summary")},
		"/me/sessions/operations/{operation}/items": map[string]any{"get": sessionOperationReadContract("items")},
		"/me/sessions":                              map[string]any{"get": sessionQueryOperation()}, "/me/sessions/{handle}": map[string]any{"get": sessionDetailOperation()}, "/me": map[string]any{"get": read}, "/me/sessions/end": map[string]any{"post": end}, "/me/access-history": map[string]any{"get": historyOperation()}}
}

func historyOperation() map[string]any {
	str := map[string]any{"type": "string"}
	optional := map[string]any{"type": []string{"string", "null"}}
	enum := func(values ...string) map[string]any { return map[string]any{"type": "string", "enum": values} }
	query := func(name string, schema map[string]any) map[string]any {
		return map[string]any{"name": name, "in": "query", "required": false, "schema": schema}
	}
	event := map[string]any{"type": "object", "properties": map[string]any{"at": str,
		"kind": enum(accesshistory.KindSignedIn, accesshistory.KindSignInRefused, accesshistory.KindSignInRefusedMore, accesshistory.KindPasswordConfirmed,
			accesshistory.KindPasswordFailed, accesshistory.KindSignedOut, accesshistory.KindSessionEnded, accesshistory.KindPasswordChanged),
		"reason": str, "workspace": optional, "count": map[string]any{"type": "integer"},
		"session": map[string]any{"type": []string{"object", "null"}, "properties": map[string]any{"handle": str, "current": map[string]any{"type": "boolean"},
			"active": map[string]any{"type": "boolean"}}},
		"device": map[string]any{"type": []string{"object", "null"}, "properties": map[string]any{"browser": enum(accesshistory.Browsers...),
			"os": enum(accesshistory.Systems...)}}}}
	note := map[string]any{"type": "object", "properties": map[string]any{"kind": enum("unclean_stop", "dropped", "damaged", "recovered_tail", "early_deletion",
		"deletion_overdue", "purge_deferred", "held", "future_file"), "from": optional, "to": optional,
		"count": map[string]any{"type": []string{"integer", "null"}, "minimum": 1}}}
	page := map[string]any{"type": "object", "properties": map[string]any{"events": map[string]any{"type": "array", "maxItems": accesshistory.MaxPage, "items": event},
		"next": optional, "retention_days": map[string]any{"type": "integer"}, "retention_source": enum(SourceDefault, SourceStartFlag), "period_since": optional,
		"effective_days": map[string]any{"type": "integer"}, "available_from": optional,
		"recording": map[string]any{"type": "object", "properties": map[string]any{"state": enum("recording", "stopped"), "since": optional,
			"reason": map[string]any{"type": []string{"string", "null"}, "enum": []any{"write_failed", "full", "full_held", nil}}}},
		"notes": map[string]any{"type": "array", "items": note}, "now": str}}
	return map[string]any{"operationId": "me.access_history.read",
		"parameters": []any{query("limit", map[string]any{"type": "integer", "minimum": 1, "maximum": accesshistory.MaxPage, "default": accesshistory.DefaultPage}),
			query("before", map[string]any{"type": "string", "pattern": "^[A-Za-z0-9_-]+$"}), query("from", map[string]any{"type": "string", "format": "date-time"}),
			query("to", map[string]any{"type": "string", "format": "date-time"})},
		"responses": map[string]any{"200": map[string]any{"description": "the caller's own access history in the session's workspace, newest first",
			"content": map[string]any{"application/json": map[string]any{"schema": page}}},
			"401": selfError(string(commands.Unauthenticated), commands.NextAnswer, false),
			"403": selfError(CodeCSRF, commands.NextNone, false),
			"422": selfError(string(commands.InvalidRequest), commands.NextCorrect, false),
			"500": selfError(string(commands.InternalError), commands.NextNone, false),
			"503": selfError(string(commands.AuthorityUnavailable), commands.NextRetry, true, string(commands.DependencyUnavailable))}}
}

var selfNext = map[string]commands.NextAction{CodeCSRF: commands.NextNone, CodeReauthentication: NextReauthenticate, string(commands.DependencyUnavailable): commands.NextRetry,
	string(commands.InvalidRequest): commands.NextCorrect, string(commands.PrerequisiteFailed): commands.NextCorrect}

func selfError(code string, next commands.NextAction, retry bool, also ...string) map[string]any {
	codes := append([]string{code}, also...)
	sort.Strings(codes)
	actions := map[string]string{code: string(next)}
	for _, other := range also {
		actions[other] = string(selfNext[other])
	}
	return map[string]any{"description": code, "x-code": code, "x-codes": codes, "x-next-action": string(next), "x-next-actions": actions, "x-retry": retry,
		"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Error"}}}}
}
