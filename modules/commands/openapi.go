package commands

import (
	"encoding/json"
	"sort"
	"strconv"
)

const decimalPattern = `^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`

type object = map[string]any

func fieldSchema(field Field) object {
	switch field.Type {
	case Decimal:
		return object{"type": "string", "pattern": decimalPattern, "x-type": "decimal"}
	case Integer:
		return object{"type": "string", "pattern": `^-?(0|[1-9][0-9]*)$`, "x-type": "integer"}
	case Boolean:
		return object{"type": "string", "enum": []string{"false", "true"}, "x-type": "boolean"}
	case Timestamp:
		return object{"type": "string", "format": "date-time", "pattern": instantForm.String(), "x-type": "timestamp"}
	case Enum:
		values := append([]string(nil), field.Values...)
		sort.Strings(values)
		return object{"type": "string", "enum": values, "x-type": "enum"}
	case ID:
		return object{"type": "string", "pattern": `^[A-Za-z0-9._:-]{1,128}$`, "x-type": "id"}
	}
	return object{"type": "string", "x-type": "text"}
}

func reference(name string) object { return object{"$ref": "#/components/schemas/" + name} }

func (r *Registry) OpenAPI(title, version string) ([]byte, error) {
	if r == nil || !r.built {
		return nil, ErrUnavailable
	}
	if title == "" || version == "" {
		return nil, ErrInvalid
	}
	identifier := object{"type": "string", "pattern": `^[A-Za-z0-9._:-]{1,128}$`}
	codes := make([]string, 0, len(errorRules))
	for code := range errorRules {
		codes = append(codes, string(code))
	}
	sort.Strings(codes)
	actions := []string{string(NextAnswer), string(NextCorrect), string(NextInspect), string(NextNone), string(NextOperation), string(NextRetry)}
	sort.Strings(actions)
	early := map[Code]bool{Unauthenticated: true, AuthorityUnavailable: true, DependencyUnavailable: true, UnsupportedCapability: true, InternalError: true}
	paths := object{}
	for _, entry := range r.Catalogue() {
		properties, required := object{}, []string{}
		for _, field := range entry.Fields {
			properties[field.Name] = fieldSchema(field)
			if field.Required {
				required = append(required, field.Name)
			}
		}
		sort.Strings(required)
		body := object{"type": "object", "additionalProperties": false, "properties": properties, "x-max-value-bytes": r.limits.Value}
		if len(required) > 0 {
			body["required"] = required
		}
		request := object{"request_id": identifier, "target_id": identifier, "body": body}
		needed := []string{"body", "request_id", "target_id"}
		if entry.Mutating {
			request["idempotency_key"], request["expected_revision"] = identifier, identifier
			needed = append(needed, "expected_revision", "idempotency_key")
		}
		if entry.Rationale {
			request["rationale"] = object{"type": "string", "minLength": 1, "x-max-bytes": r.limits.Rationale}
			needed = append(needed, "rationale")
		}
		sort.Strings(needed)
		responses := object{}
		if entry.Supported {
			responses["200"] = object{"description": "completed", "content": object{"application/json": object{"schema": reference("Result")}}}
		}
		statuses := map[int][]string{}
		for code, rule := range errorRules {
			switch {
			case code == UnsupportedCommand, !entry.Supported && !early[code]:
			case (!entry.Mutating && code == IdempotencyConflict) || (!entry.Approval && code == SelfApprovalDenied):
			default:
				statuses[rule.status] = append(statuses[rule.status], string(code))
			}
		}
		if entry.Mutating && entry.Supported {
			responses["202"] = object{"description": "accepted; not business success", "content": object{"application/json": object{"schema": reference("Result")}}}
		}
		for status, names := range statuses {
			sort.Strings(names)
			responses[strconv.Itoa(status)] = object{"description": joined(names), "content": object{"application/json": object{"schema": reference("Error")}}}
		}
		permission := object{"action": string(entry.Permission.Action), "kind": entry.Permission.Kind, "approval": entry.Approval}
		if len(entry.Permission.Fields) > 0 {
			permission["fields"] = entry.Permission.Fields
		}
		paths["/commands/"+entry.ID] = object{"post": object{
			"operationId": entry.ID, "x-permission": permission, "x-supported": entry.Supported, "x-mutating": entry.Mutating,
			"requestBody": object{"required": true, "content": object{"application/json": object{"schema": object{
				"type": "object", "additionalProperties": false, "properties": request, "required": needed}}}},
			"responses": responses,
		}}
	}
	catalogue := []object{}
	for _, code := range codes {
		rule := errorRules[Code(code)]
		listed := object{"code": code, "status": rule.status, "retry": rule.retry, "next_action": string(rule.next)}
		if Code(code) == RevisionConflict {
			listed["retry_for_reading_command"] = true
		}
		catalogue = append(catalogue, listed)
	}
	document := object{
		"openapi": "3.1.0",
		"info": object{"title": title, "version": version,
			"description": "Provisional rendering of the command catalogue. It is not a published contract, no HTTP path is deployed, and no operation is anonymous: the authentication method is not decided."},
		"x-provisional": true, "x-anonymous-access": false, "x-errors": catalogue,
		"paths": paths,
		"components": object{"schemas": object{
			"Result": object{"type": "object", "required": []string{"request_id", "status"}, "properties": object{
				"request_id": identifier, "status": object{"type": "string", "enum": []string{"accepted", "completed"}}, "operation_id": identifier, "target_id": identifier,
				"result_revision": identifier, "business_state": identifier, "job_id": identifier, "replay": object{"type": "boolean"},
				"as_of": object{"type": "string", "format": "date-time"}, "fields": object{"type": "object", "additionalProperties": object{"type": "string"}}}},
			"Error": object{"type": "object", "required": []string{"code", "next_action", "request_id", "retry"}, "properties": object{
				"code": object{"type": "string", "enum": codes}, "retry": object{"type": "boolean"}, "next_action": object{"type": "string", "enum": actions}, "request_id": identifier,
				"operation_id": identifier, "fields": object{"type": "array", "items": object{"type": "string"}}}},
		}},
	}
	return json.MarshalIndent(document, "", "  ")
}

func joined(names []string) string {
	text := ""
	for index, name := range names {
		if index > 0 {
			text += ", "
		}
		text += name
	}
	return text
}
