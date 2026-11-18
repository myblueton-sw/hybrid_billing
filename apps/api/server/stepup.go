package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	CodeReauthentication = "reauthentication_required"
	NextReauthenticate   = "reauthenticate"
	ActionSensitive      = "command.sensitive"
	stepUpDefault        = 15 * time.Minute
	validateCommand      = "trust.validate"
	maxRequestID         = 128
)

var exempt = map[string]bool{}

type postRoute struct {
	path   string
	stepUp bool
}

var postRoutes = []postRoute{
	{"/api/v1/commands/{command}", true},
	{"/api/v1/me/sessions/end", true},
	{"/api/v1/me/password", false},
	{"/api/v1/me/reauthenticate", false},
	{"/api/v1/auth/login", false},
	{"/api/v1/auth/logout", false},
	{"/api/v1/auth/link", false},
	{"/api/v1/auth/redeem", false},
}

func (s *Server) stepUp(command string) bool { return s.mutating[command] && !exempt[command] }

func stepUpAction(command string) string {
	if strings.HasPrefix(command, "trust.") {
		return authpolicy.ActionPolicyChange
	}
	return ActionSensitive
}

func tighten(limit time.Duration, mfa bool, r authpolicy.Revision, action string) (time.Duration, bool) {
	for _, rule := range r.StepUp {
		if rule.Action == action {
			limit, mfa = min(limit, rule.MaxAge), mfa || rule.MFA
		}
	}
	return limit, mfa
}

func (s *Server) stepUpTerms(workspace, user, command, target string) (time.Duration, bool) {
	action := stepUpAction(command)
	limit, mfa := stepUpDefault, false
	if current, err := s.policy.Policy(workspace); err == nil {
		limit, mfa = tighten(limit, mfa, current.Revision, action)
	}
	if command != validateCommand || s.trust == nil || !s.holdsRight(user, gate.TrustValidate, s.clock()) {
		return limit, mfa
	}
	view, ok := s.trust.Get(target, workspace)
	if !ok {
		return limit, mfa
	}
	candidate := revisionFor(workspace, view.Candidate)
	limit, mfa = tighten(limit, mfa, candidate, action)
	return min(limit, candidate.MaxAge), mfa
}

func (s *Server) stepUpCheck(caller identity.Verifier, user, command, target string) commands.Code {
	found, ok := s.sessions.get(caller)
	if !ok {
		return commands.Unauthenticated
	}
	limit, mfa := s.stepUpTerms(found.workspace, user, command, target)
	if mfa {
		return commands.PrerequisiteFailed
	}
	now := s.clock()
	if found.authenticated.After(now) || now.Sub(found.authenticated) > limit {
		return CodeReauthentication
	}
	return ""
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestID {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func echoRequest(id string) string {
	if validRequestID(id) {
		return id
	}
	return commands.Marker
}

func refusalOf(code commands.Code) (int, commands.NextAction) {
	switch code {
	case CodeReauthentication:
		return http.StatusForbidden, NextReauthenticate
	case commands.PrerequisiteFailed:
		return http.StatusUnprocessableEntity, commands.NextCorrect
	}
	return http.StatusUnauthorized, commands.NextAnswer
}

func stepUpRefusal(code commands.Code, request string) commands.Result {
	request = echoRequest(request)
	status, next := refusalOf(code)
	return commands.Result{Request: request, Err: &commands.Error{Code: code, Status: status, Next: next, Request: request}}
}

func (s *Server) stepUpRoute(w http.ResponseWriter, found session, user, route, request string) bool {
	code := s.stepUpCheck(found.verifier, user, route, "")
	if code == "" {
		return true
	}
	s.cfg.Log.Info("step_up_required", "user", user, "route", route)
	status, next := refusalOf(code)
	failModule(w, code, status, false, next, request)
	return false
}
