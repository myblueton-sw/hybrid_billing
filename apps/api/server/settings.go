package server

import (
	"fmt"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
)

const (
	SourceDefault   = "default"
	SourceStartFlag = "start_flag"
)

type AccessSettings struct {
	RequestHours        int
	PendingPerProposer  int
	PendingPerWorkspace int
	EndedKept           int
	RevocationsKept     int
	EndedShown          int
	LedgerMaxMiB        int
	Given               map[string]bool
	AcceptNewLedger     bool
	AcceptTrustReset    bool
	CarryTrustFrom      string
	TrustGeneration     uint64
}

type AccessSetting struct {
	Name    string
	Flag    string
	Unit    string
	Default int
	Min     int
	Max     int
	field   func(*AccessSettings) *int
}

func (a AccessSetting) Set(s *AccessSettings, value int) { *a.field(s) = value }

func (a AccessSetting) Value(s AccessSettings) int { return *a.field(&s) }

var accessSettings = []AccessSetting{
	{"access_request_hours", "access-request-hours", "hours", 168, 1, 720, func(s *AccessSettings) *int { return &s.RequestHours }},
	{"access_pending_per_proposer", "access-pending-per-proposer", "requests", 25, 1, 100, func(s *AccessSettings) *int { return &s.PendingPerProposer }},
	{"access_pending_per_workspace", "access-pending-per-workspace", "requests", 200, 1, 1000, func(s *AccessSettings) *int { return &s.PendingPerWorkspace }},
	{"access_ended_kept", "access-ended-kept", "requests", 1000, 50, 10000, func(s *AccessSettings) *int { return &s.EndedKept }},
	{"access_revocations_kept", "access-revocations-kept", "revocations", 1000, 50, 10000, func(s *AccessSettings) *int { return &s.RevocationsKept }},
	{"access_ended_shown", "access-ended-shown", "requests", 50, 1, 200, func(s *AccessSettings) *int { return &s.EndedShown }},
	{"decision_ledger_max_mib", "decision-ledger-max-mib", "MiB", 256, 1, 4096, func(s *AccessSettings) *int { return &s.LedgerMaxMiB }},
}

func AccessSettingList() []AccessSetting { return append([]AccessSetting(nil), accessSettings...) }

func DefaultAccessSettings() AccessSettings {
	s := AccessSettings{Given: map[string]bool{}}
	for _, a := range accessSettings {
		a.Set(&s, a.Default)
	}
	return s
}

func (s AccessSettings) numbersZero() bool {
	for _, a := range accessSettings {
		if a.Value(s) != 0 {
			return false
		}
	}
	return true
}

func (s AccessSettings) withDefaults() AccessSettings {
	if !s.numbersZero() {
		return s
	}
	out := DefaultAccessSettings()
	out.AcceptNewLedger, out.AcceptTrustReset, out.CarryTrustFrom, out.TrustGeneration = s.AcceptNewLedger, s.AcceptTrustReset, s.CarryTrustFrom, s.TrustGeneration
	return out
}

func settingError(name string) error { return fmt.Errorf("%w: %s", ErrConfig, name) }

func (s AccessSettings) validate() error {
	for _, a := range accessSettings {
		if v := a.Value(s); v < a.Min || v > a.Max {
			return settingError(a.Name)
		}
	}
	switch {
	case s.PendingPerWorkspace < s.PendingPerProposer:
		return settingError("access_pending_per_workspace")
	case s.EndedShown > s.EndedKept:
		return settingError("access_ended_shown")
	}
	return nil
}

func (s AccessSettings) limits() accesschange.Limits {
	return accesschange.Limits{Lifetime: time.Duration(s.RequestHours) * time.Hour, PerProposer: s.PendingPerProposer, PerWorkspace: s.PendingPerWorkspace,
		EndedKept: s.EndedKept, RevocationsKept: s.RevocationsKept}
}

type settingRow struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	Unit    string `json:"unit"`
	Default int    `json:"default"`
	Min     int    `json:"min"`
	Max     int    `json:"max"`
	Source  string `json:"source"`
}

func (s *Server) settingRows() []settingRow {
	out := []settingRow{}
	for _, a := range accessSettings {
		source := SourceDefault
		if s.cfg.Access.Given[a.Name] {
			source = SourceStartFlag
		}
		out = append(out, settingRow{Name: a.Name, Value: a.Value(s.cfg.Access), Unit: a.Unit, Default: a.Default, Min: a.Min, Max: a.Max, Source: source})
	}
	return out
}
