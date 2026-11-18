package server

import (
	"sort"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func selfEndSchema() map[string]any {
	str := map[string]any{"type": "string"}
	item := map[string]any{"type": "object", "properties": map[string]any{"item": str, "state": map[string]any{"type": "string",
		"enum": []string{itemDone, itemPending, itemFailed, itemCleared, itemOutside, itemNotApply}}, "reason": str}}
	row := map[string]any{"type": "object", "properties": map[string]any{"handle": str, "signed_in": str,
		"result": map[string]any{"type": "string", "enum": []string{resultEnded, resultGone}}, "items": map[string]any{"type": "array", "items": item}}}
	ref := map[string]any{"type": "object", "properties": map[string]any{"handle": str, "signed_in": str}}
	counts := map[string]any{"type": "object", "properties": map[string]any{"done": map[string]any{"type": "integer"}, "already_ended": map[string]any{"type": "integer"},
		"failed": map[string]any{"type": "integer"}, "pending": map[string]any{"type": "integer"}}}
	return map[string]any{"type": "object", "properties": map[string]any{"operation": str, "kind": map[string]any{"type": "string", "enum": []string{kindOthers, kindOne}},
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
		map[string]any{"type": "object", "additionalProperties": false, "required": []string{"session"}, "properties": map[string]any{"session": hexOf(16)}}}}
	end := map[string]any{"operationId": "me.sessions.end", "x-step-up": true,
		"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": body}}},
		"responses": map[string]any{"202": map[string]any{"description": "access blocked for the confirmed sessions that were still present; cleanup follows",
			"content": map[string]any{"application/json": map[string]any{"schema": selfEndSchema()}}},
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
	return map[string]any{"/me": map[string]any{"get": read}, "/me/sessions/end": map[string]any{"post": end}}
}

var selfNext = map[string]commands.NextAction{CodeCSRF: commands.NextNone, CodeReauthentication: NextReauthenticate,
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
