package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"
)

type sessionCursor struct {
	Owner     sessionOwner `json:"owner"`
	Caller    string       `json:"caller"`
	Query     string       `json:"query"`
	Version   uint64       `json:"version"`
	Revision  string       `json:"revision"`
	Reference time.Time    `json:"reference"`
	Expires   time.Time    `json:"expires"`
	After     string       `json:"after"`
	Pinned    bool         `json:"pinned"`
	Matched   int          `json:"matched"`
}

func (s *Server) sealSessionCursor(cursor sessionCursor) string {
	body, _ := json.Marshal(cursor)
	mac := hmac.New(sha256.New, s.sessionCursorKey)
	_, _ = mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) openSessionCursor(raw string) (sessionCursor, bool) {
	var out sessionCursor
	if len(raw) > 4096 {
		return out, false
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return out, false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return out, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return out, false
	}
	mac := hmac.New(sha256.New, s.sessionCursorKey)
	_, _ = mac.Write(body)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return out, false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil {
		return out, false
	}
	if _, err = decoder.Token(); err != io.EOF {
		return out, false
	}
	return out, true
}
