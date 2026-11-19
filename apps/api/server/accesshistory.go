package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

const historyDay = 24 * time.Hour

type historySession struct {
	Handle  string `json:"handle"`
	Current bool   `json:"current"`
	Active  bool   `json:"active"`
}

type historyDevice struct {
	Browser string `json:"browser"`
	OS      string `json:"os"`
}

type historyEvent struct {
	At        string          `json:"at"`
	Kind      string          `json:"kind"`
	Reason    string          `json:"reason"`
	Workspace *string         `json:"workspace"`
	Session   *historySession `json:"session"`
	Device    *historyDevice  `json:"device"`
	Count     int64           `json:"count"`
}

type historyRecording struct {
	State  string  `json:"state"`
	Since  *string `json:"since"`
	Reason *string `json:"reason"`
}

type historyNote struct {
	Kind  string  `json:"kind"`
	From  *string `json:"from"`
	To    *string `json:"to"`
	Count *int64  `json:"count"`
}

type historyPage struct {
	Events        []historyEvent   `json:"events"`
	Next          *string          `json:"next"`
	RetentionDays int              `json:"retention_days"`
	Source        string           `json:"retention_source"`
	PeriodSince   *string          `json:"period_since"`
	EffectiveDays int              `json:"effective_days"`
	AvailableFrom *string          `json:"available_from"`
	Recording     historyRecording `json:"recording"`
	Notes         []historyNote    `json:"notes"`
	Now           string           `json:"now"`
}

func (s *Server) openHistory() (bool, error) {
	a := s.cfg.Access
	source := SourceDefault
	if a.Given["access_history_days"] {
		source = SourceStartFlag
	}
	h, err := accesshistory.Open(accesshistory.Options{Dir: s.cfg.Data, Days: a.HistoryDays, DaysSource: source, MaxBytes: int64(a.HistoryMaxMiB) * mib,
		ClockGap: time.Duration(a.HistoryClockGap) * historyDay, AcceptShortening: a.AcceptShortening, AcceptClock: a.AcceptHistoryClock,
		ReleaseMarker: a.ReleaseMarker, ReleaseReason: a.ReleaseReason, Clock: s.clock, Manual: s.manual,
		ClockUnverified: func() bool { _, beyond := s.time.leadBeyond(); return beyond },
		Holds:           s.historyHolds,
		Log:             func(event string, args ...any) { s.log.Info(event, args...) }})
	if err != nil {
		if errors.Is(err, accesshistory.ErrConfig) || errors.Is(err, accesshistory.ErrLocked) {
			return false, fmt.Errorf("%w: access history: %w", ErrConfig, err)
		}
		return false, fmt.Errorf("%w: access history: %w", ErrStart, err)
	}
	s.history = h
	return false, nil
}

func (s *Server) historyHolds() []accesshistory.Hold {
	if s.retention == nil {
		return nil
	}
	placed := func(id string) time.Time {
		if v, ok := s.holdPlaced.Load(id); ok {
			return v.(time.Time)
		}
		return time.Time{}
	}
	out := []accesshistory.Hold{}
	for _, v := range s.retention.HoldsCovering(retention.ClassIdentity) {
		hold := v.Hold
		out = append(out, accesshistory.Hold{ID: hold.ID, Owner: hold.Owner, Reason: hold.Reason, Tenants: append([]string{}, hold.Tenants...),
			Classes: append([]string{}, hold.Classes...), PlacedAt: placed(hold.ID), Released: v.Released,
			Covers: func(tenant string) bool { return hold.Covers(tenant, retention.ClassIdentity) }})
	}
	return out
}

func (s *Server) offer(r accesshistory.Record) {
	if s.history != nil {
		s.history.Offer(r)
	}
}

func (s *Server) recordSession(kind, reason string, found session) {
	user, ok := s.dir.userOf(found.login)
	if !ok {
		return
	}
	s.offer(accesshistory.Record{Kind: kind, Reason: reason, User: user, Workspace: found.workspace, Handle: handleOf(found.verifier)})
}

func (s *Server) recordEnded(found session, cause string) {
	s.recordSession(accesshistory.KindSessionEnded, cause, found)
}

func (s *Server) recordSignedIn(user, workspace string, v identity.Verifier, r *http.Request) {
	browser, system := accesshistory.Classify(r.Header.Get("User-Agent"))
	s.offer(accesshistory.Record{Kind: accesshistory.KindSignedIn, User: user, Workspace: workspace, Handle: handleOf(v), Browser: browser, OS: system})
}

func (s *Server) recordRefused(user, reason string, r *http.Request) {
	if user == "" {
		return
	}
	browser, system := accesshistory.Classify(r.Header.Get("User-Agent"))
	s.offer(accesshistory.Record{Kind: accesshistory.KindSignInRefused, User: user, Reason: reason, Browser: browser, OS: system})
}

func (s *Server) sampleIdentity(user string) bool {
	if s.cfg.Seed != SeedSample {
		return false
	}
	switch user {
	case SampleAdmin, SampleAdmin2, SampleAdmin3, SampleMember:
		return true
	}
	return false
}

func (s *Server) incarnation(user string) error {
	if s.history == nil {
		return nil
	}
	if s.faulty("incarnation") {
		return errPartial
	}
	return s.history.Incarnation(user)
}

func encodeCursor(w, r uint64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(w, 10) + "-" + strconv.FormatUint(r, 10)))
}

func decodeCursor(raw string) (uint64, uint64, bool) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0, 0, false
	}
	a, c, ok := strings.Cut(string(b), "-")
	if !ok || !digits(a) || !digits(c) {
		return 0, 0, false
	}
	w, err1 := strconv.ParseUint(a, 10, 64)
	r, err2 := strconv.ParseUint(c, 10, 64)
	return w, r, err1 == nil && err2 == nil
}

func digits(s string) bool {
	if s == "" || len(s) > 20 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseHistoryQuery(r *http.Request) (accesshistory.Query, []string) {
	q := accesshistory.Query{Limit: accesshistory.DefaultPage}
	bad := []string{}
	for key, values := range r.URL.Query() {
		if len(values) != 1 {
			bad = append(bad, key)
			continue
		}
		v := values[0]
		switch key {
		case "limit":
			n, err := strconv.Atoi(v)
			if err != nil || !digits(v) || n < 1 || n > accesshistory.MaxPage {
				bad = append(bad, key)
			}
			q.Limit = n
		case "before":
			w, rr, ok := decodeCursor(v)
			if !ok {
				bad = append(bad, key)
			}
			q.CursorW, q.CursorR = w, rr
		case "from", "to":
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				bad = append(bad, key)
			}
			if key == "from" {
				q.From = t.UTC()
			} else {
				q.To = t.UTC()
			}
		default:
			bad = append(bad, key)
		}
	}
	if len(bad) == 0 && !q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To) {
		bad = append(bad, "from")
	}
	return q, bad
}

func optStamp(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	v := stamp(t)
	return &v
}

func (s *Server) readHistory(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	found, answer, ok := s.self(w, r)
	if !ok {
		s.domain.RUnlock()
		return
	}
	q, bad := parseHistoryQuery(r)
	if len(bad) > 0 {
		s.domain.RUnlock()
		invalidFields(w, request, bad...)
		return
	}
	if s.history == nil {
		s.domain.RUnlock()
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	own, _ := s.ownSessions(found)
	s.domain.RUnlock()
	active := map[string]bool{}
	for _, o := range own {
		active[handleOf(o.verifier)] = true
	}
	q.Workspace = found.workspace
	page := s.history.Read(answer.User(), q)
	out := historyPage{Events: []historyEvent{}, RetentionDays: page.Days, Source: page.Source, PeriodSince: optStamp(page.PeriodSince), EffectiveDays: page.EffectiveDays,
		AvailableFrom: optStamp(page.AvailableFrom), Notes: []historyNote{}, Now: stamp(s.clock())}
	if page.More {
		next := encodeCursor(page.NextW, page.NextR)
		out.Next = &next
	}
	current := handleOf(found.verifier)
	for _, e := range page.Events {
		ev := historyEvent{At: stamp(e.At), Kind: e.Kind, Reason: e.Reason, Count: e.Count}
		if e.Workspace != "" {
			ws := e.Workspace
			ev.Workspace = &ws
		}
		if e.Handle != "" {
			ev.Session = &historySession{Handle: e.Handle, Current: e.Handle == current, Active: active[e.Handle]}
		}
		if e.Browser != "" {
			ev.Device = &historyDevice{Browser: e.Browser, OS: e.OS}
		}
		out.Events = append(out.Events, ev)
	}
	out.Recording = historyRecording{State: page.Recording.State}
	if page.Recording.State == "stopped" {
		out.Recording.Since = optStamp(page.Recording.Since)
		reason := page.Recording.Reason
		out.Recording.Reason = &reason
	}
	for _, n := range page.Notes {
		out.Notes = append(out.Notes, historyNote{Kind: n.Kind, From: optStamp(n.From), To: optStamp(n.To), Count: n.Count})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, request, out)
}
