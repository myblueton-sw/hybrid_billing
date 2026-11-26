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

const queryTokenBytes = 4096

func (s *Server) queryMAC(domain string, body []byte) []byte {
	mac := hmac.New(sha256.New, s.serviceQueryKey)
	mac.Write([]byte(domain))
	mac.Write([]byte{0})
	mac.Write(body)
	return mac.Sum(nil)
}

func (s *Server) sealQueryToken(domain string, value any) (string, bool) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(s.queryMAC(domain, body)), true
}

func (s *Server) openQueryToken(domain, raw string, out any) bool {
	if len(raw) > queryTokenBytes {
		return false
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, s.queryMAC(domain, body)) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return false
	}
	_, err = decoder.Token()
	return err == io.EOF
}

type queryBinding struct {
	Area      string
	Target    string
	Owner     sessionOwner
	Caller    string
	Query     string
	Basis     string
	Reference time.Time
	Expires   time.Time
}

func (b queryBinding) continues(prior queryBinding, now time.Time) bool {
	return prior.Area == b.Area && prior.Target == b.Target && prior.Owner == b.Owner && prior.Caller == b.Caller && prior.Query == b.Query && prior.Basis == b.Basis &&
		now.Before(prior.Expires) && !prior.Reference.After(now)
}

func queryExpiry(sessionEnd, idleEnd, now time.Time, seconds int) time.Time {
	expiry := earlier(sessionEnd, now.Add(time.Duration(seconds)*time.Second))
	if !idleEnd.IsZero() {
		expiry = earlier(expiry, idleEnd)
	}
	return expiry
}

type scanBudget struct {
	limit   int
	scanned int
	stopped bool
}

func (b *scanBudget) reserve(units int) bool {
	if units > b.remaining() {
		b.stopped = true
		return false
	}
	b.scanned += units
	return true
}

func (b *scanBudget) remaining() int { return b.limit - b.scanned }
