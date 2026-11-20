package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const (
	CodeMisdirected  = "misdirected_request"
	CodeOrigin       = "origin_refused"
	CodeCSRF         = "csrf_refused"
	CodeBodyTooLarge = "body_too_large"
)

type boundaryCode struct {
	code   string
	status int
	next   commands.NextAction
}

var boundaryCodes = []boundaryCode{{CodeBodyTooLarge, http.StatusRequestEntityTooLarge, commands.NextNone}, {CodeCSRF, http.StatusForbidden, commands.NextNone},
	{CodeLinkInvalid, http.StatusUnprocessableEntity, commands.NextNone}, {CodeMisdirected, http.StatusMisdirectedRequest, commands.NextNone},
	{CodeOrigin, http.StatusForbidden, commands.NextNone}, {CodePasswordForm, http.StatusUnprocessableEntity, commands.NextNone},
	{CodeWrongPassword, http.StatusUnprocessableEntity, commands.NextNone}, {CodePasswordLocked, http.StatusTooManyRequests, commands.NextNone},
	{CodeReauthentication, http.StatusForbidden, NextReauthenticate}}

type errorBody struct {
	Code         string   `json:"code"`
	Retry        bool     `json:"retry"`
	NextAction   string   `json:"next_action"`
	RequestID    string   `json:"request_id"`
	OperationID  string   `json:"operation_id,omitempty"`
	Fields       []string `json:"fields,omitempty"`
	RetryAfterMS int64    `json:"retry_after_ms,omitempty"`
}

type resultBody struct {
	RequestID      string            `json:"request_id"`
	Status         string            `json:"status"`
	OperationID    string            `json:"operation_id,omitempty"`
	TargetID       string            `json:"target_id,omitempty"`
	ResultRevision string            `json:"result_revision,omitempty"`
	BusinessState  string            `json:"business_state,omitempty"`
	JobID          string            `json:"job_id,omitempty"`
	Replay         bool              `json:"replay"`
	AsOf           string            `json:"as_of,omitempty"`
	Fields         map[string]string `json:"fields,omitempty"`
}

func headers(w http.ResponseWriter, request string) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	if request != "" {
		h.Set("X-Request-Id", request)
	}
}

func writeJSON(w http.ResponseWriter, status int, request string, body any) {
	headers(w, request)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func fail(w http.ResponseWriter, status int, code, request string) {
	writeJSON(w, status, request, errorBody{Code: code, NextAction: string(commands.NextNone), RequestID: request})
}

func failModule(w http.ResponseWriter, code commands.Code, status int, retry bool, next commands.NextAction, request string) {
	writeJSON(w, status, request, errorBody{Code: string(code), Retry: retry, NextAction: string(next), RequestID: request})
}

var moduleRules = map[commands.Code]struct {
	status int
	retry  bool
	next   commands.NextAction
}{
	commands.Unauthenticated:       {http.StatusUnauthorized, false, commands.NextAnswer},
	commands.InvalidRequest:        {http.StatusUnprocessableEntity, false, commands.NextCorrect},
	commands.AuthorityUnavailable:  {http.StatusServiceUnavailable, true, commands.NextRetry},
	commands.InternalError:         {http.StatusInternalServerError, false, commands.NextNone},
	commands.UnsupportedCommand:    {http.StatusNotFound, false, commands.NextNone},
	commands.DependencyUnavailable: {http.StatusServiceUnavailable, true, commands.NextRetry},
	commands.PermissionDenied:      {http.StatusForbidden, false, commands.NextNone},
	commands.SelfApprovalDenied:    {http.StatusForbidden, false, commands.NextNone},
}

func failCode(w http.ResponseWriter, code commands.Code, request string) {
	rule := moduleRules[code]
	failModule(w, code, rule.status, rule.retry, rule.next, request)
}

func writeResult(w http.ResponseWriter, result commands.Result) {
	if e := result.Err; e != nil {
		body := errorBody{Code: string(e.Code), Retry: e.Retry, NextAction: string(e.Next), RequestID: result.Request, OperationID: e.Operation, Fields: e.Fields,
			RetryAfterMS: e.RetryAfterMillis()}
		if body.RetryAfterMS > 0 {
			w.Header().Set("Retry-After", strconv.FormatInt((body.RetryAfterMS+999)/1000, 10))
		}
		writeJSON(w, e.Status, result.Request, body)
		return
	}
	status := http.StatusOK
	if result.Status == commands.Accepted {
		status = http.StatusAccepted
	}
	body := resultBody{RequestID: result.Request, Status: string(result.Status), OperationID: result.Operation, TargetID: result.Target,
		ResultRevision: result.Revision, BusinessState: result.State, JobID: result.Job, Replay: result.Replay, Fields: result.Fields}
	if !result.AsOf.IsZero() {
		body.AsOf = result.AsOf.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, status, result.Request, body)
}
