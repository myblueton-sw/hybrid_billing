package server

import (
	"encoding/json"
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const description = "Command catalogue served on loopback by the pre-launch API service under /api/v1. State is in memory and resets on restart. No operation is anonymous: every command needs a session cookie and the CSRF header."

func (s *Server) Document() ([]byte, error) {
	raw, err := s.commands.OpenAPI("Hybrid Billing pre-launch API", "0.1.0")
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	doc["servers"] = []any{map[string]any{"url": "/api/v1"}}
	info := doc["info"].(map[string]any)
	info["description"] = description
	listed := doc["x-errors"].([]any)
	enum := doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"].(map[string]any)["properties"].(map[string]any)["code"].(map[string]any)
	codes := enum["enum"].([]any)
	for _, b := range boundaryCodes {
		listed = append(listed, map[string]any{"code": b.code, "status": b.status, "retry": false, "next_action": string(commands.NextNone)})
		codes = append(codes, b.code)
	}
	doc["x-errors"], enum["enum"] = listed, codes
	return json.MarshalIndent(doc, "", "  ")
}

func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	doc, err := s.Document()
	s.domain.RUnlock()
	if err != nil {
		failCode(w, commands.InternalError, request)
		return
	}
	headers(w, request)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}
