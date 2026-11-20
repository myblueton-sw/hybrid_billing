package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func serviceIdentifierSchema() map[string]any {
	return map[string]any{"type": "string", "pattern": "^[A-Za-z0-9._:-]{1,128}$", "maxLength": 128}
}

func serviceRestrictionSchema() map[string]any {
	identifier := serviceIdentifierSchema()
	selector := map[string]any{"oneOf": []any{serviceObject(map[string]any{"any": map[string]any{"type": "boolean", "const": true}}, "any"), serviceObject(map[string]any{"id": identifier}, "id")}}
	month := map[string]any{"type": "string", "pattern": "^(19[0-9]{2}|[2-9][0-9]{3})-(0[1-9]|1[0-2])$"}
	period := map[string]any{"oneOf": []any{serviceObject(map[string]any{"any": map[string]any{"type": "boolean", "const": true}}, "any"), serviceObject(map[string]any{"from": month, "until": month}, "from", "until")}, "x-until-exclusive": true, "x-until-after-from": true}
	fields := map[string]any{"type": "array", "items": identifier, "uniqueItems": true}
	result := serviceObject(map[string]any{"id": identifier, "action": map[string]any{"type": "string", "enum": []string{string(gate.View), string(gate.ViewCost), string(gate.ChangeCustomerPolicy), string(gate.ChangeSellingPolicy), string(gate.RequestApproval), string(gate.Issue), string(gate.Download), string(gate.Deliver)}}, "kind": identifier, "fields": fields, "customer": selector, "legal_entity": selector, "contract": selector, "period": period}, "id", "action", "kind", "fields", "customer", "legal_entity", "contract", "period")
	result["allOf"] = []any{map[string]any{"if": map[string]any{"properties": map[string]any{"action": map[string]any{"enum": []string{string(gate.View), string(gate.ViewCost), string(gate.Download)}}}}, "then": map[string]any{"properties": map[string]any{"fields": map[string]any{"minItems": 1}}}, "else": map[string]any{"properties": map[string]any{"fields": map[string]any{"maxItems": 0}}}}}
	result["x-vocabulary"] = "actual configured gate kinds/fields and field classes are validated by the producer; no effective grant is created"
	return result
}

func serviceCredentialEligibilitySchema() map[string]any {
	return serviceObject(map[string]any{"can_revoke": map[string]any{"type": "boolean"}, "step_up_state": map[string]any{"type": "string", "enum": []string{"ready", "reauthentication_required", "prerequisite_failed", "unavailable"}}, "execution_state": map[string]any{"type": "string", "enum": []string{"ready", "dependency_unavailable", "trust_hold"}}}, "can_revoke", "step_up_state", "execution_state")
}

func serviceCountSchema() map[string]any {
	return map[string]any{"type": "string", "pattern": "^(0|[1-9][0-9]*)$"}
}

func serviceStringSchema() map[string]any { return map[string]any{"type": "string"} }

func serviceObject(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}

func serviceDeclarationSchema() map[string]any {
	str := serviceStringSchema()
	ref := serviceObject(map[string]any{"audience": str, "scope": str}, "audience", "scope")
	return serviceObject(map[string]any{"state": map[string]any{"type": "string", "enum": []string{"recorded", "missing", "unavailable"}}, "recorded_at": str, "version": str, "purpose": str, "organization": str, "organization_name": str, "organization_name_state": map[string]any{"type": "string", "enum": []string{"available", "unavailable"}}, "catalog_version": str, "scopes": map[string]any{"type": "array", "items": ref, "maxItems": 64}}, "state", "scopes")
}

func serviceAccountSchema() map[string]any {
	str := serviceStringSchema()
	summary := serviceObject(map[string]any{"state": map[string]any{"type": "string", "enum": []string{"exact_at_reference", "not_evaluated"}}, "recorded": str, "active": str, "expired": str, "revoked": str, "earliest_unexpired_expiry": str}, "state")
	summary["x-count-values"] = "canonical unsigned decimal when exact_at_reference; empty strings when not_evaluated"
	return serviceObject(map[string]any{"id": str, "state": map[string]any{"type": "string", "enum": []string{"active", "suspended", "retired"}}, "version": serviceCountSchema(), "owner": str, "owner_login": str, "creator": str, "creator_login": str, "declaration": serviceDeclarationSchema(), "credential_summary": summary, "recent_use_state": map[string]any{"const": "unavailable"}}, "id", "state", "version", "owner", "creator", "declaration", "credential_summary", "recent_use_state")
}

func serviceCredentialSchema() map[string]any {
	str := serviceStringSchema()
	pickup := map[string]any{"type": "string", "enum": []string{pickupOpen, pickupCollected, pickupNever}, "x-present-when": "credential currently active and its delivery is recorded",
		"x-meaning": map[string]string{pickupOpen: "the original issuing session could collect the secret now; another administrator is not granted collection", pickupCollected: "the server returned the secret once; it does not confirm the secret was stored", pickupNever: "the secret is recorded as not returned and the original issuing session cannot collect at this observation; metadata alone does not certify irreversible deletion"}}
	until := map[string]any{"type": "string", "format": "date-time", "x-present-when": "secret_pickup is open", "x-meaning": "estimated earliest pickup/issuing-session absolute/idle end at this observation; activity can move it later and sign-out can end it sooner"}
	row := serviceObject(map[string]any{"id": str, "service": str, "service_state": str, "not_before": map[string]any{"type": "string", "format": "date-time"}, "not_after": map[string]any{"type": "string", "format": "date-time"}, "state": str, "current_state": str, "usable": map[string]any{"type": "boolean"}, "observed_at": map[string]any{"type": "string", "format": "date-time"}, "secret_pickup": pickup, "secret_pickup_until": until}, "id", "service", "service_state", "not_before", "not_after", "state", "current_state", "usable", "observed_at")
	row["x-no-secret-or-pickup-reference"] = true
	row["x-usable"] = "current period/service-state metadata only; service-request authentication and effective grants remain unconnected"
	return row
}

func serviceLimitsSchema() map[string]any {
	return serviceObject(map[string]any{"credential_days": map[string]any{"type": "integer"}, "credentials_per_service": serviceCountSchema(), "services": map[string]any{"type": "integer"}, "replay_seconds": map[string]any{"type": "integer", "x-meaning": "how long an idempotency key replays its stored outcome"}}, "credential_days", "credentials_per_service", "services", "replay_seconds")
}

func serviceAdmissionSchema() map[string]any {
	return serviceObject(map[string]any{"recorded": serviceCountSchema(), "limit": serviceCountSchema(), "certainty": map[string]any{"const": "exact"}}, "recorded", "limit", "certainty")
}

func serviceEligibilitySchema() map[string]any {
	str := serviceStringSchema()
	boolean := map[string]any{"type": "boolean"}
	out := serviceObject(map[string]any{"state": map[string]any{"type": "string", "enum": []string{"eligible", "ineligible", "unavailable"}}, "can_issue": boolean, "can_reinstate": boolean, "can_declare": boolean, "can_suspend": boolean, "can_retire": boolean, "step_up_state": map[string]any{"type": "string", "enum": []string{"ready", "reauthentication_required", "prerequisite_failed", "unavailable"}}, "execution_state": map[string]any{"type": "string", "enum": []string{"ready", "trust_hold", "dependency_unavailable"}}, "live_credentials": str, "credential_limit": str, "operator_blocked": boolean}, "state", "can_issue", "can_reinstate", "can_declare", "can_suspend", "can_retire", "step_up_state", "execution_state")
	out["x-preconditions"] = "observed actor/stage/owner/declaration/hold/stranded/step-up/operator/capacity/version/allocator checks; valid submitted fields, rate limits, idempotency and later changes remain execution checks"
	return out
}

func (s *Server) servicePageSchema(row map[string]any) map[string]any {
	str := serviceStringSchema()
	return serviceObject(map[string]any{"rows": map[string]any{"type": "array", "maxItems": 100, "items": row}, "search_input": map[string]any{"type": "string", "maxLength": 8192, "description": "Exact decoded request q for correlation only; query.q is the applied normalized search and cursor scope"}, "query": map[string]any{"type": "object"}, "next": map[string]any{"type": "string", "maxLength": 4096}, "reference_at": map[string]any{"type": "string", "format": "date-time"}, "observed_at": map[string]any{"type": "string", "format": "date-time"}, "revision": str, "order": str, "count": serviceObject(map[string]any{"value": serviceCountSchema(), "certainty": map[string]any{"type": "string", "enum": []string{"exact", "lower_bound"}}}, "value", "certainty"), "work_units": map[string]any{"type": "integer", "minimum": 0, "maximum": 10000}, "scan_stopped": map[string]any{"type": "boolean"}, "settings": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, "limits": serviceLimitsSchema(), "catalog_version": str, "availability": map[string]any{"type": "string", "enum": []string{"recorded", "not_recorded", "version_unavailable"}}, "admission": serviceAdmissionSchema()}, "rows", "search_input", "query", "next", "reference_at", "observed_at", "revision", "order", "count", "work_units", "scan_stopped", "settings", "limits", "catalog_version")
}

func serviceReadResponses(schema map[string]any, continuation bool) map[string]any {
	responses := map[string]any{"200": map[string]any{"description": "authorized metadata in the stated scope", "content": map[string]any{"application/json": map[string]any{"schema": schema}}}, "401": readError(commands.Unauthenticated), "403": readCodes(readError(commands.PermissionDenied), string(commands.PermissionDenied), CodeCSRF), "409": readError(commands.RevisionConflict), "422": readError(commands.InvalidRequest), "503": readCodes(readError(commands.DependencyUnavailable), string(commands.DependencyUnavailable), string(commands.AuthorityUnavailable))}
	if continuation {
		responses["409"] = readError(commands.RevisionConflict, "after")
	}
	return responses
}

func (s *Server) serviceQueryReadPaths() map[string]any {
	str := serviceStringSchema()
	owner := serviceObject(map[string]any{"id": str, "login": str, "role": str, "from": str, "until": str, "eligible_at_reference": map[string]any{"type": "boolean"}}, "id", "login", "role", "from", "until", "eligible_at_reference")
	organization := serviceObject(map[string]any{"id": str, "name": str, "name_state": map[string]any{"type": "string", "enum": []string{"available", "unavailable"}}, "kind": map[string]any{"type": "string", "enum": []string{"legal_entity", "internal_unit"}}, "parent": str}, "id", "name", "name_state", "kind", "parent")
	audience := serviceObject(map[string]any{"id": str, "resource": str, "configured_scope_count": map[string]any{"type": "integer", "minimum": 0, "maximum": 100000}}, "id", "resource", "configured_scope_count")
	scope := serviceObject(map[string]any{"workspace": str, "id": str, "audience": str, "resource": str, "restrictions": serviceRestrictionSchema()}, "workspace", "id", "audience", "resource", "restrictions")
	out := map[string]any{}
	lists := []struct {
		path, area, order string
		row               map[string]any
		filters           []string
	}{
		{"/identity/services", "accounts", "service_id_asc", serviceAccountSchema(), []string{"q", "organization", "owner", "audience", "scope", "expiry", "state", "declaration"}},
		{"/identity/service-credentials", "credentials", "service_id_asc_credential_id_asc", serviceCredentialSchema(), []string{"q", "service", "state", "collection"}},
		{"/identity/service-owners", "owners", "owner_id_asc", owner, []string{"q"}},
		{"/identity/service-organizations", "organizations", "organization_id_asc", organization, []string{"q"}},
		{"/identity/service-audiences", "audiences", "audience_id_asc", audience, []string{"q"}},
		{"/identity/service-audiences/{id}/scopes", "scopes", "scope_id_asc", scope, []string{"q"}},
		{"/identity/services/{id}/declarations", "history", "declaration_version_desc", serviceDeclarationSchema(), []string{"version"}},
	}
	for _, entry := range lists {
		parameters := []any{map[string]any{"name": "after", "in": "query", "schema": map[string]any{"type": "string", "maxLength": 4096}}, map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": s.cfg.Access.ServiceQueryPage}}, map[string]any{"name": "workspace", "in": "query", "schema": str, "description": "must equal the authenticated canonical boot workspace"}}
		for _, name := range entry.filters {
			parameters = append(parameters, map[string]any{"name": name, "in": "query", "schema": str})
		}
		if entry.area == "scopes" || entry.area == "history" {
			parameters = append(parameters, map[string]any{"name": "id", "in": "path", "required": true, "schema": str})
		}
		get := map[string]any{"operationId": "identity.services." + entry.area, "x-admission": "current_installation_administrator", "parameters": parameters, "x-order": entry.order, "x-count-certainty": "cumulative_lower_bound_until_end_of_source", "x-work-units": "bounded charged source positions; each scope reserves eight units conservatively", "x-reference": "all membership time predicates frozen at reference_at; current caller reauthorized every read", "x-cursor": "process/area/target/query/user/workspace/epoch/caller/source/reference/expiry bound, no lifetime extension", "x-never": "the secret and its pickup reference are never part of this read", "responses": serviceReadResponses(s.servicePageSchema(entry.row), true)}
		if entry.area == "accounts" {
			get["x-no-embedded-populations"] = true
			get["x-admission-count"] = "exact_workspace_recorded_accounts_independent_of_filters"
		}
		if entry.area == "credentials" {
			get["x-collection-membership"] = "recorded_not_returned_and_active_at_reference"
			get["x-pickup-observation"] = "current_original_issuing_session_availability"
		}
		out[entry.path] = map[string]any{"get": get}
	}
	details := []struct {
		path, area string
		schema     map[string]any
	}{
		{"/identity/services/{id}", "service_detail", serviceObject(map[string]any{"row": serviceAccountSchema(), "eligibility": serviceEligibilitySchema(), "admission": serviceAdmissionSchema(), "revision": str, "observed_at": str, "settings": map[string]any{"type": "array"}, "limits": serviceLimitsSchema(), "catalog_version": str}, "row", "eligibility", "admission", "revision", "observed_at", "settings", "limits", "catalog_version")},
		{"/identity/service-credentials/{id}", "credential_detail", serviceObject(map[string]any{"row": serviceCredentialSchema(), "eligibility": serviceCredentialEligibilitySchema(), "revision": str, "observed_at": str, "limits": serviceLimitsSchema()}, "row", "eligibility", "revision", "observed_at", "limits")},
		{"/identity/service-owners/{id}", "owner_detail", serviceObject(map[string]any{"row": owner, "revision": str, "observed_at": str}, "row", "revision", "observed_at")},
		{"/identity/service-organizations/{id}", "organization_detail", serviceObject(map[string]any{"row": organization, "eligibility": str, "revision": str, "observed_at": str}, "row", "eligibility", "revision", "observed_at")},
		{"/identity/service-audiences/{id}/scopes/{scope}", "scope_detail", serviceObject(map[string]any{"row": scope, "eligibility": str, "catalog_version": str, "revision": str, "observed_at": str}, "row", "eligibility", "catalog_version", "revision", "observed_at")},
	}
	for _, entry := range details {
		entry.schema["properties"].(map[string]any)["workspace"] = str
		entry.schema["required"] = append(entry.schema["required"].([]string), "workspace")
		parameters := []any{map[string]any{"name": "id", "in": "path", "required": true, "schema": str}, map[string]any{"name": "workspace", "in": "query", "schema": str}}
		if entry.area == "scope_detail" {
			parameters = append(parameters, map[string]any{"name": "scope", "in": "path", "required": true, "schema": str})
		}
		out[entry.path] = map[string]any{"get": map[string]any{"operationId": "identity.services." + entry.area, "x-admission": "current_installation_administrator", "parameters": parameters, "x-known-id-independent-of-list": true, "x-current-eligibility": true, "x-unknown-or-foreign": "uniform permission_denied without row/count metadata", "responses": serviceReadResponses(entry.schema, false)}}
	}
	return out
}
