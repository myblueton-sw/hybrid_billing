package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

const (
	recoveryShown      = 50
	notShown           = "not_shown"
	alreadyProposed    = "already_proposed"
	notConsoleKind     = "not_console_kind"
	modeRecoveryResume = "recovery_resume"
	modeLockRelease    = "lock_release"
	modeNone           = "none"
	phaseNotRecovering = "not_recovering"
	recoveryOpen       = "open"
	recoveryNone       = "none"
	callsFromRecovery  = "recovery"
)

var recoveryLater = []string{"migration", "balances"}

type recoveryDomainRow struct {
	Domain   string `json:"domain"`
	Status   string `json:"status"`
	Mismatch []int  `json:"mismatch_chunks"`
}

type recoveryOpenRow struct {
	State           string              `json:"state"`
	ID              string              `json:"id"`
	Phase           string              `json:"phase"`
	AsOf            string              `json:"as_of"`
	PointGeneration uint64              `json:"point_generation"`
	Backup          string              `json:"backup_reference"`
	ManifestDigest  string              `json:"manifest_digest"`
	Domains         []recoveryDomainRow `json:"domains"`
}

type recoveryHistoryRow struct {
	Recovery       string `json:"recovery"`
	ManifestDigest string `json:"manifest_digest"`
	Generation     uint64 `json:"generation"`
	Outcome        string `json:"outcome"`
	Lost           int    `json:"lost_count"`
}

type recoverySetRow struct {
	Installation string               `json:"installation"`
	State        string               `json:"state"`
	Generation   uint64               `json:"generation"`
	Recovery     recoveryOpenRow      `json:"recovery"`
	History      []recoveryHistoryRow `json:"history"`
	HistoryTotal int                  `json:"history_total"`
	Begin        string               `json:"begin"`
	Abandon      string               `json:"abandon"`
}

type recoveryCallRow struct {
	ID       string `json:"id"`
	Writer   string `json:"writer"`
	Deadline string `json:"deadline"`
	Key      string `json:"key"`
	Series   string `json:"series"`
	Number   int64  `json:"number"`
}

type recoveryLockRow struct {
	By        string `json:"by"`
	ByLogin   string `json:"by_login"`
	Rationale string `json:"rationale"`
	At        string `json:"at"`
}

type recoveryIsolationRow struct {
	Installation string            `json:"installation"`
	State        string            `json:"state"`
	Generation   uint64            `json:"generation"`
	StoppedAt    string            `json:"stopped_at"`
	CanLock      bool              `json:"can_lock"`
	LockBlocked  string            `json:"lock_blocked"`
	Writers      []string          `json:"writers"`
	Verifier     string            `json:"verifier"`
	CallsSource  string            `json:"open_calls_source"`
	OpenCalls    []recoveryCallRow `json:"open_calls"`
	OpenTotal    int               `json:"open_calls_total"`
	OtherScope   bool              `json:"other_scope_present"`
	Permits      string            `json:"permits"`
	Locks        []recoveryLockRow `json:"locks"`
}

type recoveryExternalRow struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type recoveryTargetRow struct {
	Key       string `json:"key"`
	Series    string `json:"series"`
	Number    int64  `json:"number"`
	Fenced    bool   `json:"fenced"`
	HighWater *int64 `json:"high_water"`
}

type recoveryReconcileRow struct {
	Installation string                `json:"installation"`
	Externals    []recoveryExternalRow `json:"externals"`
	Unknown      []recoveryTargetRow   `json:"unknown_targets"`
	UnknownTotal int                   `json:"unknown_total"`
	OtherScope   bool                  `json:"other_scope_present"`
	Skipped      int                   `json:"skipped"`
	Reconciler   string                `json:"reconciler"`
}

type recoveryEventRow struct {
	Sequence uint64 `json:"sequence"`
	Kind     string `json:"kind"`
	Domain   string `json:"domain"`
	At       string `json:"at"`
}

type recoveryResumeRow struct {
	Installation  string             `json:"installation"`
	Mode          string             `json:"mode"`
	Phase         string             `json:"phase"`
	Pending       []recoveryEventRow `json:"pending_events"`
	PendingTotal  int                `json:"pending_total"`
	Lost          []uint64           `json:"lost"`
	LostDigest    string             `json:"lost_digest"`
	CanUnlock     bool               `json:"can_propose_unlock"`
	UnlockBlocked string             `json:"unlock_blocked"`
}

type recoverySummaryRow struct {
	Installation     string `json:"installation"`
	Skew             int64  `json:"skew_seconds"`
	Writer           string `json:"writer"`
	External         string `json:"external"`
	ExternalKind     string `json:"external_kind"`
	Verifier         string `json:"verifier"`
	AsOf             string `json:"as_of"`
	PointGeneration  uint64 `json:"point_generation"`
	ManifestDigest   string `json:"manifest_digest"`
	UnlockGeneration uint64 `json:"unlock_generation"`
	LostDigest       string `json:"lost_digest"`
}

type recoveryProposalRow struct {
	Number           uint64             `json:"number"`
	Target           string             `json:"target"`
	Kind             string             `json:"kind"`
	Installation     string             `json:"installation"`
	State            string             `json:"state"`
	Created          string             `json:"created_at"`
	Proposer         string             `json:"proposer"`
	ProposerLogin    string             `json:"proposer_login"`
	Approver         string             `json:"approver"`
	ApproverLogin    string             `json:"approver_login"`
	Rationale        string             `json:"rationale"`
	ClosedBy         string             `json:"closed_by"`
	ClosedByLogin    string             `json:"closed_by_login"`
	ClosingRationale string             `json:"closing_rationale"`
	Summary          recoverySummaryRow `json:"summary"`
	CanApprove       bool               `json:"can_approve"`
	ApproveBlocked   string             `json:"approve_blocked"`
	CanWithdraw      bool               `json:"can_withdraw"`
	WithdrawBlocked  string             `json:"withdraw_blocked"`
}

type recoverySetsSection struct {
	State string           `json:"state"`
	Rows  []recoverySetRow `json:"rows"`
}

type recoveryIsolationSection struct {
	State string                 `json:"state"`
	Rows  []recoveryIsolationRow `json:"rows"`
}

type recoveryReconcileSection struct {
	State string                 `json:"state"`
	Rows  []recoveryReconcileRow `json:"rows"`
}

type recoveryMigrationSection struct {
	State   string   `json:"state"`
	Modules []string `json:"modules"`
}

type recoveryResumeSection struct {
	State string              `json:"state"`
	Rows  []recoveryResumeRow `json:"rows"`
}

type recoveryPage struct {
	Revision       string                   `json:"revision"`
	Now            string                   `json:"now"`
	State          string                   `json:"state"`
	Sets           recoverySetsSection      `json:"sets"`
	Isolation      recoveryIsolationSection `json:"isolation"`
	Reconciliation recoveryReconcileSection `json:"reconciliation"`
	Migration      recoveryMigrationSection `json:"migration"`
	Resume         recoveryResumeSection    `json:"resume"`
	Proposals      []recoveryProposalRow    `json:"proposals"`
}

type recoveryFacts struct {
	install    *recoveryInstall
	state      string
	generation uint64
	stopped    time.Time
	view       *recovery.RecoveryView
	history    []recovery.History
}

func stampOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return stamp(t)
}

func checkReason(err error) string {
	var refusal recovery.Refusal
	switch {
	case err == nil:
		return ""
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, recovery.ErrNotFound):
		return "not_found"
	case errors.Is(err, recovery.ErrLimit):
		return "limit"
	case errors.Is(err, recovery.ErrConflict):
		return "conflict"
	}
	return "invalid"
}

func (s *Server) checkBlocker(c recovery.Change) string {
	return checkReason(s.recoveryCall("check", func() error { return s.recovery.Check(c) }))
}

func (s *Server) visibleInstalls(workspace string) []*recoveryInstall {
	out := []*recoveryInstall{}
	for _, i := range s.rec.installs {
		if workspace == s.bootWorkspace && i.tenant == workspace {
			out = append(out, i)
		}
	}
	return out
}

func (s *Server) factsOf(i *recoveryInstall) (recoveryFacts, error) {
	f := recoveryFacts{install: i}
	var errs [3]error
	f.state, f.generation, f.stopped, errs[0] = s.recoveryStatus(i.id)
	var v recovery.RecoveryView
	errs[1] = s.recoveryCall("recovery", func() (err error) { v, err = s.recovery.Recovery(i.id); return })
	switch {
	case errs[1] == nil:
		f.view = &v
	case errors.Is(errs[1], recovery.ErrNotFound) && errs[0] == nil:
		errs[1] = nil
	}
	errs[2] = s.recoveryCall("history", func() (err error) { f.history, err = s.recovery.History(i.id); return })
	return f, errors.Join(errs[:]...)
}

func setRow(f recoveryFacts) recoverySetRow {
	row := recoverySetRow{Installation: f.install.id, State: f.state, Generation: f.generation, Recovery: recoveryOpenRow{State: recoveryNone, Domains: []recoveryDomainRow{}},
		History: []recoveryHistoryRow{}, HistoryTotal: len(f.history), Begin: notConnected, Abandon: notConnected}
	if v := f.view; v != nil {
		row.Recovery = recoveryOpenRow{State: recoveryOpen, ID: v.ID, Phase: v.Phase, AsOf: stampOrEmpty(v.AsOf), PointGeneration: v.PointGeneration, Backup: v.Backup,
			ManifestDigest: v.ManifestDigest, Domains: []recoveryDomainRow{}}
		for _, d := range v.Domains {
			row.Recovery.Domains = append(row.Recovery.Domains, recoveryDomainRow{Domain: d.Domain, Status: d.Status, Mismatch: append([]int{}, d.Mismatch...)})
		}
	}
	for index := len(f.history) - 1; index >= 0 && len(row.History) < recoveryShown; index-- {
		h := f.history[index]
		row.History = append(row.History, recoveryHistoryRow{Recovery: h.Recovery, ManifestDigest: h.ManifestDigest, Generation: h.Generation, Outcome: h.Outcome, Lost: h.Lost})
	}
	return row
}

func (s *Server) isolationRow(f recoveryFacts, workspace string) recoveryIsolationRow {
	i := f.install
	row := recoveryIsolationRow{Installation: i.id, State: f.state, Generation: f.generation, StoppedAt: stampOrEmpty(f.stopped), Writers: append([]string{}, i.writers...),
		Verifier: i.verifier, CallsSource: notShown, OpenCalls: []recoveryCallRow{}, Permits: notShown, Locks: []recoveryLockRow{}}
	if f.state != recovery.StateServing {
		row.LockBlocked = f.state
	}
	row.CanLock = row.LockBlocked == ""
	if v := f.view; v != nil {
		row.CallsSource = callsFromRecovery
		for _, c := range v.OpenCalls {
			if c.Target.Workspace != workspace {
				row.OtherScope = true
				continue
			}
			row.OpenTotal++
			if len(row.OpenCalls) < recoveryShown {
				row.OpenCalls = append(row.OpenCalls, recoveryCallRow{ID: c.ID, Writer: c.Writer, Deadline: stamp(c.Deadline), Key: c.Target.Key, Series: c.Target.Series,
					Number: c.Target.Number})
			}
		}
	}
	for _, l := range i.locks {
		row.Locks = append(row.Locks, recoveryLockRow{By: l.by, ByLogin: s.loginOf(l.by), Rationale: l.rationale, At: stamp(l.at)})
	}
	return row
}

func (s *Server) reconcileRow(f recoveryFacts, workspace string) (recoveryReconcileRow, error) {
	v, id := f.view, f.install.id
	row := recoveryReconcileRow{Installation: id, Externals: []recoveryExternalRow{}, Unknown: []recoveryTargetRow{}, Skipped: v.Skipped, Reconciler: notConnected}
	for _, x := range v.Externals {
		row.Externals = append(row.Externals, recoveryExternalRow{ID: x.ID, Kind: x.Kind, Status: x.Status})
	}
	errs := []error{}
	for _, t := range v.Unknown {
		if t.Workspace != workspace {
			row.OtherScope = true
			continue
		}
		row.UnknownTotal++
		if len(row.Unknown) >= recoveryShown {
			continue
		}
		target := recoveryTargetRow{Key: t.Key, Series: t.Series, Number: t.Number}
		errs = append(errs, s.recoveryCall("fenced", func() (err error) { target.Fenced, err = s.recovery.Fenced(id, workspace, t.Key); return }))
		if t.Series != "" {
			var high int64
			errs = append(errs, s.recoveryCall("high_water", func() (err error) { high, err = s.recovery.HighWater(id, workspace, t.Series); return }))
			target.HighWater = &high
		}
		row.Unknown = append(row.Unknown, target)
	}
	return row, errors.Join(errs...)
}

func (s *Server) resumeRow(f recoveryFacts) recoveryResumeRow {
	row := recoveryResumeRow{Installation: f.install.id, Mode: modeNone, Phase: phaseNotRecovering, Pending: []recoveryEventRow{}, Lost: []uint64{}}
	switch v := f.view; {
	case v != nil:
		row.Mode, row.Phase, row.PendingTotal, row.Lost = modeRecoveryResume, v.Phase, len(v.Pending), append(row.Lost, v.Lost...)
		for _, e := range v.Pending {
			if len(row.Pending) < recoveryShown {
				row.Pending = append(row.Pending, recoveryEventRow{Sequence: e.Sequence, Kind: e.Event.Kind, Domain: e.Event.Domain, At: stamp(e.At)})
			}
		}
	case f.state == recovery.StateLocked:
		row.Mode = modeLockRelease
	}
	row.LostDigest = recovery.LostDigest(row.Lost)
	row.UnlockBlocked = alreadyProposed
	if !s.rec.pendingUnlock(f.install.id) {
		row.UnlockBlocked = s.checkBlocker(recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: f.install.id, Generation: f.generation,
			Lost: row.LostDigest}})
	}
	row.CanUnlock = row.UnlockBlocked == ""
	return row
}

func recoverySummary(c recovery.Change) recoverySummaryRow {
	return recoverySummaryRow{Installation: changeScope(c), Skew: int64(c.Installation.Skew / time.Second), Writer: c.Writer.ID, External: c.External.ID,
		ExternalKind: c.External.Kind, Verifier: c.Verifier.ID, AsOf: stampOrEmpty(c.Begin.AsOf), PointGeneration: c.Begin.Generation,
		ManifestDigest: c.Begin.ManifestDigest, UnlockGeneration: c.Unlock.Generation, LostDigest: c.Unlock.Lost}
}

func (s *Server) recoveryBlockers(v recovery.ProposalView, reader string) (string, string) {
	if v.State != recovery.ProposalPending {
		blocked := recovery.ReasonNotPending
		if v.State == recoveryRetired {
			blocked = recoveryRetired
		}
		return blocked, blocked
	}
	withdraw := ""
	if v.Proposer != reader {
		withdraw = recovery.ReasonNotProposer
	}
	switch {
	case v.Change.Kind != recovery.ChangeUnlock:
		return notConsoleKind, withdraw
	case v.Proposer == reader:
		return ownProposal, withdraw
	}
	return s.checkBlocker(v.Change), withdraw
}

func (s *Server) recoveryProposalRow(e *recoveryEntry, reader string) recoveryProposalRow {
	v := e.view
	row := recoveryProposalRow{Number: v.Number, Target: recoveryProposalTarget + strconv.FormatUint(v.Number, 10), Kind: v.Change.Kind, Installation: changeScope(v.Change),
		State: v.State, Created: stamp(e.created), Proposer: v.Proposer, ProposerLogin: s.loginOf(v.Proposer), Approver: v.Approver, ApproverLogin: s.loginOf(v.Approver),
		Rationale: e.rationale, ClosedBy: e.closedBy, ClosedByLogin: s.loginOf(e.closedBy), ClosingRationale: e.closing, Summary: recoverySummary(v.Change)}
	row.ApproveBlocked, row.WithdrawBlocked = s.recoveryBlockers(v, reader)
	row.CanApprove, row.CanWithdraw = row.ApproveBlocked == "", row.WithdrawBlocked == ""
	return row
}

func (s *Server) recoveryPage(reader, workspace string, installs []*recoveryInstall, now time.Time) (recoveryPage, error) {
	if s.faulty("recovery-read") {
		return recoveryPage{}, recovery.ErrInvalid
	}
	page := recoveryPage{Revision: s.Revision(), Now: stamp(now), State: stateConfigured, Sets: recoverySetsSection{State: stateConfigured, Rows: []recoverySetRow{}},
		Isolation: recoveryIsolationSection{State: stateConfigured, Rows: []recoveryIsolationRow{}}, Reconciliation: recoveryReconcileSection{Rows: []recoveryReconcileRow{}},
		Migration: recoveryMigrationSection{State: notConnected, Modules: append([]string{}, recoveryLater...)},
		Resume:    recoveryResumeSection{State: stateConfigured, Rows: []recoveryResumeRow{}}, Proposals: []recoveryProposalRow{}}
	errs := []error{}
	for _, i := range installs {
		f, err := s.factsOf(i)
		if err != nil {
			return recoveryPage{}, err
		}
		page.Sets.Rows = append(page.Sets.Rows, setRow(f))
		page.Isolation.Rows = append(page.Isolation.Rows, s.isolationRow(f, workspace))
		if f.view != nil {
			row, err := s.reconcileRow(f, workspace)
			page.Reconciliation.Rows, errs = append(page.Reconciliation.Rows, row), append(errs, err)
		}
		page.Resume.Rows = append(page.Resume.Rows, s.resumeRow(f))
	}
	page.Reconciliation.State = configured(len(page.Reconciliation.Rows) > 0)
	for index := len(s.rec.entries) - 1; index >= 0 && len(page.Proposals) < recoveryShown; index-- {
		if e := s.rec.entries[index]; s.rec.tenantOf(e) == workspace {
			page.Proposals = append(page.Proposals, s.recoveryProposalRow(e, reader))
		}
	}
	return page, errors.Join(errs...)
}

func (s *Server) readRecovery(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	now := s.clock()
	installs := s.visibleInstalls(answer.Workspace())
	if len(installs) == 0 {
		writeJSON(w, http.StatusOK, request, installationEmpty{Revision: s.Revision(), Now: stamp(now), State: stateNotConfigured})
		return
	}
	page, err := s.recoveryPage(answer.User(), answer.Workspace(), installs, now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, page)
}
