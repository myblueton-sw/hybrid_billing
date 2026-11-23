package server

import (
	"fmt"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
)

const (
	SourceDefault   = "default"
	SourceStartFlag = "start_flag"
)

type AccessSettings struct {
	RequestHours            int
	PendingPerProposer      int
	PendingPerWorkspace     int
	EndedKept               int
	RevocationsKept         int
	EndedShown              int
	LedgerMaxMiB            int
	Given                   map[string]bool
	AcceptNewLedger         bool
	AcceptTrustReset        bool
	CarryTrustFrom          string
	TrustGeneration         uint64
	RolePeriodMaxDays       int
	ClockSkewHours          int
	RoleWarningDays         int
	BulkMaxTargets          int
	BulkMaxMatching         int
	BulkRetainedMax         int
	BulkMinutes             int
	BulkPending             int
	BulkKept                int
	UserFilterScan          int
	UserCountScan           int
	SessionQueryPage        int
	SessionQueryScan        int
	SessionQuerySeconds     int
	SessionSelectionPending int
	SessionSelectionTargets int
	SessionRecordBudget     int
	ClockBase               string
	HistoryDays             int
	HistoryMaxMiB           int
	HistoryClockGap         int
	AcceptShortening        int
	AcceptHistoryClock      string
	ReleaseMarker           string
	ReleaseReason           string
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

var identitySettings = []AccessSetting{
	{"role_period_max_days", "role-period-max-days", "days", 366, 1, 3660, func(s *AccessSettings) *int { return &s.RolePeriodMaxDays }},
	{"clock_skew_tolerance_hours", "clock-skew-tolerance-hours", "hours", 24, 1, 720, func(s *AccessSettings) *int { return &s.ClockSkewHours }},
	{"role_period_warning_days", "role-period-warning-days", "days", 14, 1, 90, func(s *AccessSettings) *int { return &s.RoleWarningDays }},
	{"bulk_revocation_max_targets", "bulk-revocation-max-targets", "people", 500, 1, 5000, func(s *AccessSettings) *int { return &s.BulkMaxTargets }},
	{"bulk_revocation_max_matching", "bulk-revocation-max-matching", "people", 5000, 1, 100000, func(s *AccessSettings) *int { return &s.BulkMaxMatching }},
	{"bulk_revocation_manifest_minutes", "bulk-revocation-manifest-minutes", "minutes", 30, 5, 1440, func(s *AccessSettings) *int { return &s.BulkMinutes }},
	{"bulk_revocation_pending", "bulk-revocation-pending", "lists", 20, 1, 200, func(s *AccessSettings) *int { return &s.BulkPending }},
	{"bulk_revocation_jobs_kept", "bulk-revocation-jobs-kept", "jobs", 100, 10, 1000, func(s *AccessSettings) *int { return &s.BulkKept }},
	{"bulk_revocation_retained_people_max", "bulk-revocation-retained-people-max", "people", 1100000, 10000, 20000000, func(s *AccessSettings) *int { return &s.BulkRetainedMax }},
}

var userSettings = []AccessSetting{
	{"user_filter_scan", "user-filter-scan", "people", 1000, 100, 10000, func(s *AccessSettings) *int { return &s.UserFilterScan }},
	{"user_count_scan", "user-count-scan", "people", 20000, 1000, 100000, func(s *AccessSettings) *int { return &s.UserCountScan }},
}

var historySettings = []AccessSetting{
	{"access_history_days", "access-history-days", "days", 90, 1, 1095, func(s *AccessSettings) *int { return &s.HistoryDays }},
	{"access_history_max_mib", "access-history-max-mib", "MiB", 256, 16, 1024, func(s *AccessSettings) *int { return &s.HistoryMaxMiB }},
	{"access_history_clock_gap_days", "access-history-clock-gap", "days", 7, 1, 365, func(s *AccessSettings) *int { return &s.HistoryClockGap }},
}

const (
	ClockBaseLedger = "ledger"
	ClockBaseWall   = "wall"
)

func AccessSettingList() []AccessSetting { return append([]AccessSetting(nil), accessSettings...) }

func IdentitySettingList() []AccessSetting { return append([]AccessSetting(nil), identitySettings...) }

func UserSettingList() []AccessSetting { return append([]AccessSetting(nil), userSettings...) }

func HistorySettingList() []AccessSetting { return append([]AccessSetting(nil), historySettings...) }

func allSettings() []AccessSetting {
	return append(append(append(append(AccessSettingList(), identitySettings...), userSettings...), historySettings...), append(sessionQuerySettings, sessionSelectionSettings...)...)
}

func DefaultAccessSettings() AccessSettings {
	s := AccessSettings{Given: map[string]bool{}}
	for _, a := range allSettings() {
		a.Set(&s, a.Default)
	}
	return s
}

func numbersZero(s AccessSettings, list []AccessSetting) bool {
	for _, a := range list {
		if a.Value(s) != 0 {
			return false
		}
	}
	return true
}

func (s AccessSettings) withDefaults() AccessSettings {
	out := DefaultAccessSettings()
	for _, group := range [][]AccessSetting{accessSettings, identitySettings, userSettings, historySettings, sessionQuerySettings, sessionSelectionSettings} {
		if !numbersZero(s, group) {
			for _, a := range group {
				a.Set(&out, a.Value(s))
			}
		}
	}
	if s.Given != nil {
		out.Given = s.Given
	}
	out.AcceptNewLedger, out.AcceptTrustReset, out.CarryTrustFrom, out.TrustGeneration, out.ClockBase = s.AcceptNewLedger, s.AcceptTrustReset, s.CarryTrustFrom, s.TrustGeneration, s.ClockBase
	out.AcceptShortening, out.AcceptHistoryClock, out.ReleaseMarker, out.ReleaseReason = s.AcceptShortening, s.AcceptHistoryClock, s.ReleaseMarker, s.ReleaseReason
	return out
}

func settingError(name string) error { return fmt.Errorf("%w: %s", ErrConfig, name) }

func (s AccessSettings) validate() error {
	for _, a := range allSettings() {
		if v := a.Value(s); v < a.Min || v > a.Max {
			return settingError(a.Name)
		}
	}
	switch {
	case s.SessionRecordBudget < 1+3*s.SessionSelectionTargets:
		return settingError("session_record_references")
	case s.PendingPerWorkspace < s.PendingPerProposer:
		return settingError("access_pending_per_workspace")
	case s.EndedShown > s.EndedKept:
		return settingError("access_ended_shown")
	case s.RoleWarningDays > s.RolePeriodMaxDays:
		return settingError("role_period_warning_days")
	case s.ClockBase != "" && s.ClockBase != ClockBaseLedger && s.ClockBase != ClockBaseWall:
		return settingError("clock_base")
	case s.BulkMaxMatching < s.BulkMaxTargets:
		return settingError("bulk_revocation_max_matching")
	case s.RetainedPeople() > s.BulkRetainedMax:
		return fmt.Errorf("%w: bulk_revocation_retained_people_max %d is below %s = %d", ErrConfig, s.BulkRetainedMax, retainedFormula(s), s.RetainedPeople())
	}
	return nil
}

func (s AccessSettings) RetainedPeople() int {
	return (2*s.BulkKept + s.BulkPending) * s.BulkMaxMatching
}

func retainedFormula(s AccessSettings) string {
	return fmt.Sprintf("(2 × bulk_revocation_jobs_kept %d + bulk_revocation_pending %d) × bulk_revocation_max_matching %d", s.BulkKept, s.BulkPending, s.BulkMaxMatching)
}

var settingNotes = map[string]string{
	"bulk_revocation_retained_people_max": "; start is refused when (2 × bulk_revocation_jobs_kept + bulk_revocation_pending) × bulk_revocation_max_matching exceeds it",
}

func SettingNote(name string) string { return settingNotes[name] }

func (s AccessSettings) limits() accesschange.Limits {
	return accesschange.Limits{Lifetime: time.Duration(s.RequestHours) * time.Hour, PerProposer: s.PendingPerProposer, PerWorkspace: s.PendingPerWorkspace,
		EndedKept: s.EndedKept, RevocationsKept: s.RevocationsKept}
}

func (s AccessSettings) bulkLimits() bulkrevoke.Limits {
	return bulkrevoke.Limits{MaxTargets: s.BulkMaxTargets, MaxMatching: s.BulkMaxMatching, Pending: s.BulkPending, Kept: s.BulkKept, Lifetime: time.Duration(s.BulkMinutes) * time.Minute}
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

func (s *Server) settingRows() []settingRow { return s.rowsOf(accessSettings) }

func (s *Server) bulkSettingRows() []settingRow {
	return append(s.rowsOf(identitySettings), s.rowsOf(userSettings)...)
}

func (s *Server) rowsOf(list []AccessSetting) []settingRow {
	out := []settingRow{}
	for _, a := range list {
		source := SourceDefault
		if s.cfg.Access.Given[a.Name] {
			source = SourceStartFlag
		}
		out = append(out, settingRow{Name: a.Name, Value: a.Value(s.cfg.Access), Unit: a.Unit, Default: a.Default, Min: a.Min, Max: a.Max, Source: source})
	}
	return out
}

var sessionQuerySettings = []AccessSetting{
	{"session_query_page_size", "session-query-page-size", "rows", 50, 1, 100, func(s *AccessSettings) *int { return &s.SessionQueryPage }},
	{"session_query_scan", "session-query-scan", "sessions", 1000, 100, 10000, func(s *AccessSettings) *int { return &s.SessionQueryScan }},
	{"session_query_lifetime_seconds", "session-query-lifetime-seconds", "seconds", 300, 30, 3600, func(s *AccessSettings) *int { return &s.SessionQuerySeconds }},
}

func SessionQuerySettingList() []AccessSetting {
	return append([]AccessSetting(nil), sessionQuerySettings...)
}

var sessionSelectionSettings = []AccessSetting{
	{"session_selection_pending", "session-selection-pending", "views per caller", 2, 1, 8, func(s *AccessSettings) *int { return &s.SessionSelectionPending }},
	{"session_selection_max_targets", "session-selection-max-targets", "targets", 10000, 64, 10000, func(s *AccessSettings) *int { return &s.SessionSelectionTargets }},
	{"session_record_references", "session-record-references", "references", 50000, 256, 1000000, func(s *AccessSettings) *int { return &s.SessionRecordBudget }},
}

func SessionSelectionSettingList() []AccessSetting {
	return append([]AccessSetting(nil), sessionSelectionSettings...)
}
