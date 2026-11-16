package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

const (
	ReaderKind       = "focus_data"
	ReaderField      = "rows"
	ReaderSecretFile = "reader-secret"
	readerTimeout    = 25 * time.Second
	readerValidity   = 30 * 24 * time.Hour
)

type requestKey struct{}

var ReaderPaths = []string{"/api/focus-overview", "/api/focus-commitments", "/api/focus-marketplace", "/api/focus-receipts", "/api/focus-connections",
	"/api/focus-collection-status"}

func DefaultReaders() map[string]string {
	out := map[string]string{}
	for _, p := range ReaderPaths[:3] {
		out[p] = "http://127.0.0.1:8806"
	}
	for _, p := range ReaderPaths[3:] {
		out[p] = "http://127.0.0.1:8810"
	}
	return out
}

func readerTarget(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Path != "" || u.RawQuery != "" || u.User != nil || u.Hostname() != "127.0.0.1" || u.Port() == "" {
		return nil, false
	}
	return u, true
}

func validReaders(readers map[string]string) bool {
	for path, target := range readers {
		if !contains(ReaderPaths, path) {
			return false
		}
		if _, ok := readerTarget(target); !ok {
			return false
		}
	}
	return true
}

func (s *Server) writeReaderSecret() error {
	path := filepath.Join(s.cfg.Data, ReaderSecretFile)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s.readerSecret = encode(random(tokenBytes))
	return writeExclusive(path, []byte(s.readerSecret+"\n"))
}

var forwarded = []string{"Accept", "Accept-Language", "If-None-Match"}

func (s *Server) reader(target *url.URL) http.HandlerFunc {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = readerTimeout
	transport.Proxy = nil
	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			out := r.Out
			out.URL.Scheme, out.URL.Host, out.Host = target.Scheme, target.Host, target.Host
			keep := http.Header{}
			for _, name := range forwarded {
				if v := r.In.Header.Values(name); len(v) > 0 {
					keep[name] = v
				}
			}
			keep.Set("X-HB-Local-Client", "1")
			keep.Set("X-HB-Gate-Secret", s.readerSecret)
			out.Header = keep
		},
		ModifyResponse: func(resp *http.Response) error {
			for name := range resp.Header {
				if strings.HasPrefix(name, "Set-Cookie") || strings.HasPrefix(name, "Access-Control") {
					resp.Header.Del(name)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) {
			request, _ := r.Context().Value(requestKey{}).(string)
			failCode(w, commands.DependencyUnavailable, request)
		},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		request := r.Header.Get("X-Request-Id")
		if r.Method != http.MethodGet {
			failCode(w, commands.UnsupportedCommand, request)
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
		decision := s.gate.Decide(answer, ConsoleClient, gate.Request{Action: gate.View, Kind: ReaderKind, Fields: []string{ReaderField},
			Resource: gate.Resource{WorkspaceID: answer.Workspace()}})
		if !decision.Allowed {
			failCode(w, commands.PermissionDenied, request)
			return
		}
		proxy.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestKey{}, request)))
	}
}
