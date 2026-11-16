package server

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func mutatingMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func (s *Server) requestID() string {
	return encode(random(16))
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/openapi.json", s.openapi)
	mux.HandleFunc("GET /api/v1/session", s.current)
	mux.HandleFunc("POST /api/v1/commands/{command}", s.command)
	mux.HandleFunc("GET /api/v1/identity/users", s.listUsers)
	mux.HandleFunc("GET /api/v1/identity/links/{ref}", s.collect)
	mux.HandleFunc("GET /api/v1/identity/policy", s.listPolicy)
	mux.HandleFunc("GET /api/v1/identity/access", s.listAccess)
	mux.HandleFunc("GET /api/v1/identity/access/preview", s.previewAccess)
	mux.HandleFunc("GET /api/v1/identity/services", s.listServices)
	mux.HandleFunc("GET /api/v1/me", s.readSelf)
	mux.HandleFunc("GET /api/v1/organizations", s.listOrganizations)
	mux.HandleFunc("POST /api/v1/me/password", s.changePassword)
	mux.HandleFunc("POST /api/v1/me/sessions/end", s.endSelfSessions)
	for path, raw := range s.cfg.Readers {
		target, _ := readerTarget(raw)
		mux.HandleFunc("GET "+path, s.reader(target))
	}
	mux.HandleFunc("GET /api/v1/auth/csrf", s.presession)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("POST /api/v1/auth/link", s.peek)
	mux.HandleFunc("POST /api/v1/auth/redeem", s.redeem)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		failCode(w, commands.UnsupportedCommand, r.Header.Get("X-Request-Id"))
	})
	return s.guard(mux)
}

func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := s.requestID()
		r.Header.Set("X-Request-Id", request)
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.Error("handler panic", "request", request, "stack", string(debug.Stack()))
				failCode(w, commands.InternalError, request)
			}
		}()
		if !s.allowedHost(r.Host) {
			fail(w, http.StatusMisdirectedRequest, CodeMisdirected, request)
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			fail(w, http.StatusForbidden, CodeOrigin, request)
			return
		}
		if mutatingMethod(r.Method) && !s.origins[r.Header.Get("Origin")] {
			fail(w, http.StatusForbidden, CodeOrigin, request)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func tooLarge(err error) bool {
	var limit *http.MaxBytesError
	return errors.As(err, &limit)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, r.Header.Get("X-Request-Id"), map[string]string{"status": "ok"})
}

type sessionBody struct {
	User      string `json:"user"`
	Workspace string `json:"workspace"`
	Role      string `json:"role"`
	CSRF      string `json:"csrf_token"`
}

func (s *Server) current(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	if r.Header.Get(clientHeader) != ConsoleClient {
		fail(w, http.StatusForbidden, CodeOrigin, request)
		return
	}
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return
	}
	writeJSON(w, http.StatusOK, request, sessionBody{User: answer.User(), Workspace: answer.Workspace(), Role: string(answer.Role()), CSRF: found.csrf})
}
