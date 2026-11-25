package server

import (
	"errors"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
)

const (
	evidenceRunsShown   = 20
	evidenceRunPage     = 1000
	maxProfileTargetLen = maxTargetBytes - len(perfProfileTarget)
)

type evidenceHardwareRow struct {
	Number    int    `json:"number"`
	ID        string `json:"id"`
	CPUs      int64  `json:"cpus"`
	MemoryMiB int64  `json:"memory_mib"`
	DiskGiB   int64  `json:"disk_gib"`
	Storage   string `json:"storage"`
	Spec      string `json:"spec"`
}

type evidenceAxes struct {
	RowsRun           int64 `json:"rows_run"`
	RowsDay           int64 `json:"rows_day"`
	RowsMonth         int64 `json:"rows_month"`
	Customers         int64 `json:"customers"`
	Accounts          int64 `json:"accounts"`
	SharedPools       int64 `json:"shared_pools"`
	SkewPermille      int64 `json:"skew_permille"`
	FanOut            int64 `json:"fan_out"`
	ConcurrentCloses  int64 `json:"concurrent_closes"`
	PeakWindowMinutes int64 `json:"peak_window_minutes"`
	RetentionMonths   int64 `json:"retention_months"`
	FailureInjections int64 `json:"failure_injections"`
	BacklogRows       int64 `json:"backlog_rows"`
}

type evidenceBudget struct {
	ID         string           `json:"id"`
	State      string           `json:"state"`
	Hardware   string           `json:"hardware"`
	Limits     map[string]int64 `json:"limits"`
	Proposer   *evidencePerson  `json:"proposer"`
	Approver   *evidencePerson  `json:"approver"`
	ApprovalBy *evidencePerson  `json:"approval_by"`
	ApprovedBy *evidencePerson  `json:"approved_by"`
	ApprovedAt *string          `json:"approved_at"`
}

type evidenceRunRow struct {
	Number      uint64            `json:"number"`
	Build       string            `json:"build"`
	Hardware    string            `json:"hardware"`
	Rows        int64             `json:"rows"`
	DurationS   int64             `json:"duration_s"`
	Mismatches  int64             `json:"mismatches"`
	Interrupted bool              `json:"interrupted"`
	Metrics     map[string]*int64 `json:"metrics"`
	Harness     string            `json:"harness"`
	At          string            `json:"at"`
	Verdict     string            `json:"verdict"`
}

type evidenceProfileRow struct {
	Number        int                          `json:"number"`
	Target        string                       `json:"target"`
	ID            string                       `json:"id"`
	Gate          string                       `json:"gate"`
	Axes          evidenceAxes                 `json:"axes"`
	Dataset       string                       `json:"dataset"`
	Expected      string                       `json:"expected"`
	Retired       bool                         `json:"retired"`
	ActiveBudget  *evidenceBudget              `json:"active_budget"`
	Status        *string                      `json:"status"`
	Runs          evidenceList[evidenceRunRow] `json:"runs"`
	Retire        *uint64                      `json:"retire"`
	CanRetire     bool                         `json:"can_retire"`
	RetireBlocked string                       `json:"retire_blocked"`
}

type evidenceProfileStatus struct {
	Profile  string `json:"profile"`
	Status   string `json:"status"`
	Run      uint64 `json:"run"`
	Failures int    `json:"failures"`
}

type evidenceGateStatus struct {
	Gate     string                  `json:"gate"`
	Status   string                  `json:"status"`
	Profiles []evidenceProfileStatus `json:"profiles"`
	Retired  []evidenceProfileStatus `json:"retired"`
}

type evidenceReadiness struct {
	Ready bool                 `json:"ready"`
	Gates []evidenceGateStatus `json:"gates"`
}

type evidenceLoad struct {
	Origin          string                            `json:"origin"`
	Hardware        evidenceList[evidenceHardwareRow] `json:"hardware"`
	HardwareNext    string                            `json:"hardware_next"`
	Profiles        evidenceList[evidenceProfileRow]  `json:"profiles"`
	ProfilesNext    string                            `json:"profiles_next"`
	Readiness       *evidenceReadiness                `json:"readiness"`
	Measurement     string                            `json:"measurement"`
	PlanningTargets string                            `json:"planning_targets"`
}

func axesOf(a perfgate.Axes) evidenceAxes {
	return evidenceAxes{RowsRun: a.RowsRun, RowsDay: a.RowsDay, RowsMonth: a.RowsMonth, Customers: a.Customers, Accounts: a.Accounts, SharedPools: a.SharedPools,
		SkewPermille: a.SkewPermille, FanOut: a.FanOut, ConcurrentCloses: a.ConcurrentCloses, PeakWindowMinutes: a.PeakWindowMinutes, RetentionMonths: a.RetentionMonths,
		FailureInjections: a.FailureInjections, BacklogRows: a.BacklogRows}
}

func (f *evidenceFacts) budget(profile string) (*evidenceBudget, error) {
	s := f.s
	v, err := s.perfActiveBudget(profile)
	switch {
	case errors.Is(err, perfgate.ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	}
	b := &evidenceBudget{ID: v.Budget.ID, State: v.State, Hardware: v.Budget.Hardware, Limits: map[string]int64{}, Proposer: s.evidencePerson(v.Proposer),
		Approver: s.evidencePerson(v.Approver), ApprovalBy: s.evidencePerson(v.ApprovalBy), ApprovedBy: s.evidencePerson(v.ApprovedBy)}
	for _, l := range v.Budget.Limits {
		b.Limits[l.Metric] = l.Value
	}
	if !v.ApprovedAt.IsZero() {
		b.ApprovedAt = optional(stamp(v.ApprovedAt))
	}
	return b, nil
}

func runRow(v perfgate.RunView) evidenceRunRow {
	x := v.Run
	row := evidenceRunRow{Number: v.Number, Build: x.Build, Hardware: x.Hardware, Rows: x.Rows, DurationS: x.DurationS, Mismatches: x.Mismatches, Interrupted: x.Interrupted,
		Metrics: map[string]*int64{}, Harness: v.Harness, At: stamp(v.At), Verdict: v.Verdict}
	for _, metric := range perfgate.Metrics {
		row.Metrics[metric] = nil
	}
	for _, m := range x.Metrics {
		value := m.Value
		row.Metrics[m.Metric] = &value
	}
	return row
}

func (f *evidenceFacts) profileRow(id string, statuses map[string]string) (evidenceProfileRow, error) {
	s := f.s
	p, retired, err := s.perfProfile(id)
	if err != nil {
		return evidenceProfileRow{}, err
	}
	row := evidenceProfileRow{Target: perfProfileTarget + id, ID: p.ID, Gate: p.Gate, Axes: axesOf(p.Axes), Dataset: p.Dataset, Expected: p.Expected, Retired: retired}
	if row.ActiveBudget, err = f.budget(id); err != nil {
		return evidenceProfileRow{}, err
	}
	if statuses != nil {
		row.Status = optional(statuses[id])
	}
	runs, err := s.perfRuns(id)
	if err != nil {
		return evidenceProfileRow{}, err
	}
	row.Runs, _ = listOf(len(runs), evidenceRunsShown, func(index int) (evidenceRunRow, error) { return runRow(runs[len(runs)-1-index]), nil })
	if e := s.ev.pendingProfileRetire(id); e != nil {
		number := e.number
		row.Retire = &number
	}
	switch {
	case retired:
		row.RetireBlocked = perfgate.ReasonRetired
	case row.Retire != nil:
		row.RetireBlocked = alreadyProposed
	case !f.eligible:
		row.RetireBlocked = noEligibleApprover
	case len(id) > maxProfileTargetLen:
		row.RetireBlocked = targetTooLong
	}
	row.CanRetire = row.RetireBlocked == ""
	return row, nil
}

func statusRows(list []perfgate.ProfileStatus) []evidenceProfileStatus {
	out := []evidenceProfileStatus{}
	for _, x := range list {
		out = append(out, evidenceProfileStatus{Profile: x.Profile, Status: x.Status, Run: x.Run, Failures: x.Failures})
	}
	return out
}

func (f *evidenceFacts) load() (evidenceLoad, error) {
	s := f.s
	section := evidenceLoad{Origin: f.origin, Measurement: notConnected, PlanningTargets: notConnected}
	statuses, readiness, err := f.profileStatuses()
	if err != nil {
		return evidenceLoad{}, err
	}
	section.Readiness = readiness
	if section.Hardware, err = listOf(len(s.ev.hardware), evidenceShown, s.hardwareAt); err != nil {
		return evidenceLoad{}, err
	}
	if section.Hardware.More {
		section.HardwareNext = strconv.Itoa(len(section.Hardware.Rows))
	}
	profiles := s.ev.profiles
	section.Profiles, err = listOf(len(profiles), evidenceShown, func(index int) (evidenceProfileRow, error) { return f.profileAt(index, statuses) })
	if section.Profiles.More {
		section.ProfilesNext = strconv.Itoa(len(section.Profiles.Rows))
	}
	return section, err
}

func (f *evidenceFacts) profileStatuses() (map[string]string, *evidenceReadiness, error) {
	if f.candidate == nil {
		return nil, nil, nil
	}
	v, err := f.s.perfReadiness(f.build)
	if err != nil {
		return nil, nil, err
	}
	statuses := map[string]string{}
	readiness := &evidenceReadiness{Ready: v.Ready, Gates: []evidenceGateStatus{}}
	for _, g := range v.Gates {
		readiness.Gates = append(readiness.Gates, evidenceGateStatus{Gate: g.Gate, Status: g.Status, Profiles: statusRows(g.Profiles), Retired: statusRows(g.Retired)})
		for _, x := range append(append([]perfgate.ProfileStatus{}, g.Profiles...), g.Retired...) {
			statuses[x.Profile] = x.Status
		}
	}
	return statuses, readiness, nil
}

func (f *evidenceFacts) profileAt(index int, statuses map[string]string) (evidenceProfileRow, error) {
	row, err := f.profileRow(f.s.ev.profiles[index], statuses)
	row.Number = index + 1
	return row, err
}

func perfSummary(c perfgate.Change) map[string]any {
	switch c.Kind {
	case perfgate.ChangeProfile:
		return map[string]any{"profile": c.Profile.ID, "gate": c.Profile.Gate}
	case perfgate.ChangeHardware:
		return map[string]any{"hardware": c.Hardware.ID}
	case perfgate.ChangeBudget:
		return map[string]any{"budget": c.Budget.ID, "profile": c.Budget.Profile, "hardware": c.Budget.Hardware}
	case perfgate.ChangeHarness:
		return map[string]any{"harness": c.Harness.ID}
	case perfgate.ChangeRetire:
		return map[string]any{"profile": c.Retire.Profile}
	}
	if c.Approval.Budget != "" {
		return map[string]any{"budget": c.Approval.Budget}
	}
	return map[string]any{}
}
