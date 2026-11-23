package server

func selfAccountSchema() map[string]any {
	text := map[string]any{"type": "string"}
	flag := map[string]any{"type": "boolean"}
	number := map[string]any{"type": "integer", "minimum": 0}
	properties := map[string]any{"user_id": text, "login": text, "email": text, "workspace": text, "role": text,
		"identities":   map[string]any{"type": "array", "items": map[string]any{"type": "object", "required": []string{"method", "login"}, "properties": map[string]any{"method": text, "login": text}}},
		"password_set": flag, "installation_administrator": flag, "current_session": sessionQueryRowSchema(),
		"recorded_session_count":   map[string]any{"type": "integer", "minimum": 0, "description": "Exact retained recorded sessions in the current canonical owner/epoch/workspace index, including inactive/unavailable records; not active-device count."},
		"accepted_operation_count": map[string]any{"type": "integer", "minimum": 0, "description": "Exact retained accepted operations in the current canonical owner/epoch/workspace index; excludes unconfirmed intent."},
		"sessions_generation":      map[string]any{"type": "string", "pattern": "^[0-9a-f]{32}$", "description": "Opaque legacy compatibility token; separate from canonical session-query generation."},
		"idle_minutes":             number, "max_age_seconds": number, "signed_in_at": text, "authenticated_at": text, "now": text}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties,
		"required":    []string{"user_id", "login", "email", "workspace", "role", "identities", "password_set", "installation_administrator", "current_session", "recorded_session_count", "accepted_operation_count", "sessions_generation", "idle_minutes", "max_age_seconds", "signed_in_at", "authenticated_at", "now"},
		"description": "Bounded account metadata and current session. Session populations and expanded termination results are available only through dedicated bounded query/detail routes."}
}
