package server

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

const mib = 1 << 20

type recoveryRow struct {
	At             string `json:"at"`
	TruncatedBytes int64  `json:"truncated_bytes"`
}

type ledgerRow struct {
	State           string       `json:"state"`
	IsolationReason string       `json:"isolation_reason"`
	LedgerID        string       `json:"ledger_id"`
	CreatedAt       string       `json:"created_at"`
	Replaces        string       `json:"replaces"`
	ReplaceReason   string       `json:"replace_reason"`
	ReplacedFile    string       `json:"replaced_file"`
	Records         uint64       `json:"records"`
	Watermark       string       `json:"watermark"`
	CheckedAt       string       `json:"checked_at"`
	ProblemLine     int          `json:"problem_line"`
	Recovery        *recoveryRow `json:"recovery"`
	NotDurable      bool         `json:"not_durable"`
	File            string       `json:"file"`
}

func ledgerStartError(err error) error {
	switch {
	case errors.Is(err, ErrConfig):
		return err
	case errors.Is(err, decisionledger.ErrLocked):
		return fmt.Errorf("%w: %w", ErrStart, err)
	case errors.Is(err, decisionledger.ErrBound):
		return fmt.Errorf("%w: decision_ledger_max_mib is below the decision ledger size", ErrConfig)
	case errors.Is(err, decisionledger.ErrNothingToReplace):
		return fmt.Errorf("%w: accept-new-decision-ledger: %w", ErrConfig, err)
	}
	return fmt.Errorf("%w: decision ledger: %w", ErrStart, err)
}

func (s *Server) openLedger() (bool, error) {
	l, err := decisionledger.Open(decisionledger.Options{Dir: s.cfg.Data, MaxBytes: int64(s.cfg.Access.LedgerMaxMiB) * mib, AcceptNew: s.cfg.Access.AcceptNewLedger,
		FailStartSync: s.faulty("ledger-start-sync"), FailSync: func() bool { return s.faulty("ledger-sync") }, Clock: s.clock, Random: rand.Reader,
		Successor: s.trustSuccessor})
	if err != nil {
		return false, ledgerStartError(err)
	}
	s.ledger = l
	st := l.Status()
	s.log.Info("decision_ledger_opened", "state", string(st.State), "records", st.Records)
	if err := s.settleClockBase(); err != nil {
		l.Close()
		return false, err
	}
	changed, err := s.replayLedger()
	if err != nil {
		l.Close()
		return changed, err
	}
	replayed, err := s.replayIdentity()
	return changed || replayed, err
}

func (s *Server) Close() error {
	s.stopOnce.Do(func() {
		if s.stop != nil {
			close(s.stop)
		}
	})
	s.workers.Wait()
	var errs []error
	if s.history != nil {
		errs = append(errs, s.history.Close())
	}
	if s.ledger != nil {
		errs = append(errs, s.ledger.Close())
	}
	return errors.Join(errs...)
}

func protectedGrant(g gate.GrantView) bool {
	return g.ClientID != "" || gate.InstallationAction(g.Terms.Action)
}

func (s *Server) replayRevoke(revoke func() error) error {
	if s.faulty("replay") {
		return gate.ErrUnavailable
	}
	return revoke()
}

func (s *Server) replayOne(d decisionledger.Decision) (string, bool, bool, error) {
	if d.Decision == decisionledger.RevokeDelegation {
		found, ok := s.delegationView(d.Target)
		switch {
		case !ok:
			return accesschange.ReplayAbsent, false, false, nil
		case found.Closed:
			return accesschange.ReplayReapplied, true, false, nil
		}
		return accesschange.ReplayReapplied, true, true, s.replayRevoke(func() error { return s.admin.RevokeDelegation(found.ID, found.Version) })
	}
	found, ok := s.grantView(d.Target)
	switch {
	case !ok:
		return accesschange.ReplayAbsent, false, false, nil
	case protectedGrant(found):
		return accesschange.ReplayProtected, true, false, nil
	case found.Closed:
		return accesschange.ReplayReapplied, true, false, nil
	}
	return accesschange.ReplayReapplied, true, true, s.replayRevoke(func() error { return s.admin.RevokeGrant(found.ID, found.Version) })
}

func restartCleanup(ended []string) []accesschange.Item {
	return []accesschange.Item{
		{Item: "C1", State: accesschange.ItemDone, Reason: "server_restarted"},
		{Item: "C2", State: accesschange.ItemDone, Reason: "server_restarted"},
		{Item: "C3", State: accesschange.ItemNotApplicable, Reason: "human_grants_only"},
		{Item: "C4", State: accesschange.ItemNotApplicable, Reason: "no_store"},
		{Item: "C5", State: accesschange.ItemOutside, Reason: "cannot_recall"},
		{Item: "C6", State: accesschange.ItemDone, Reason: "server_restarted"},
		{Item: "C7", State: accesschange.ItemOutside, Reason: "no_job_runner"},
		{Item: "C8", State: accesschange.ItemNotApplicable, Reason: "nothing_keyed_to_grants"},
		chainItem(ended),
	}
}

func targetKind(decision string) string {
	if decision == decisionledger.RevokeDelegation {
		return accesschange.TargetDelegation
	}
	return accesschange.TargetGrant
}

func (s *Server) replayLedger() (bool, error) {
	records := s.ledger.Decisions()
	changed := false
	var requests, revocations uint64
	restored := []accesschange.Restored{}
	for _, rec := range records {
		d := rec.Decision
		if d.Trust != nil || decisionledger.IdentityKind(d.Decision) {
			continue
		}
		if n, ok := accesschange.Number(d.Request, accesschange.RequestPrefix); ok {
			requests = max(requests, n)
		}
		if n, ok := accesschange.Number(d.Revocation, accesschange.RevocationPrefix); ok {
			revocations = max(revocations, n)
		}
		if d.Workspace != s.bootWorkspace {
			continue
		}
		note, present, closed, err := s.replayOne(d)
		if err != nil {
			s.log.Error("decision_ledger_replay_failed", "revocation", d.Revocation, "target", d.Target)
			return changed, fmt.Errorf("%w: decision ledger replay could not re-apply %s (record %d): %w", ErrStart, d.Revocation, rec.Sequence, err)
		}
		changed = changed || closed
		restored = append(restored, accesschange.Restored{ID: d.Revocation, RecordedAt: rec.At, Revocation: accesschange.Revocation{Workspace: d.Workspace,
			Target: accesschange.Target{Kind: targetKind(d.Decision), ID: d.Target, Version: d.TargetVersion}, RequestedBy: d.RequestedBy, ApprovedBy: d.ApprovedBy,
			Request: d.Request, RequestDigest: d.RequestDigest}, Generation: d.DenyGeneration, Ended: d.EndedDelegations,
			Receipt: accesschange.Receipt{Sequence: rec.Sequence, Watermark: rec.Watermark}, Replay: note, Present: present, Cleanup: restartCleanup(d.EndedDelegations)})
	}
	st := s.ledger.Status()
	s.changes.Restore(restored, accesschange.Receipt{Sequence: st.Records, Watermark: st.Watermark}, requests, revocations)
	return changed, nil
}

func (s *Server) ledgerBlocksIncrease() bool {
	if _, beyond := s.time.leadBeyond(); beyond || s.ledger == nil {
		return true
	}
	switch s.ledger.Status().State {
	case decisionledger.StateOK, decisionledger.StateRecovered:
		return false
	}
	return true
}

type ledgerAdapter struct{ s *Server }

func (a ledgerAdapter) Commit(e accesschange.Entry) (accesschange.Receipt, error) {
	s := a.s
	if s.ledger == nil {
		return accesschange.Receipt{}, &decisionledger.CommitError{Reason: decisionledger.ReasonIsolated}
	}
	decision := decisionledger.RevokeGrant
	if e.Target.Kind == accesschange.TargetDelegation {
		decision = decisionledger.RevokeDelegation
	}
	receipt, err := s.ledger.Commit(decisionledger.Decision{Workspace: e.Workspace, Decision: decision, Target: e.Target.ID, TargetVersion: e.Target.Version,
		Revocation: e.Revocation, Request: e.Request, RequestDigest: e.RequestDigest, RequestedBy: e.RequestedBy, ApprovedBy: e.ApprovedBy,
		DenyGeneration: e.DenyGeneration, EndedDelegations: e.Ended})
	if err != nil {
		st := s.ledger.Status()
		s.log.Error("decision_ledger_failed", "state", string(st.State), "sequence", st.Records, "revocation", e.Revocation)
		return accesschange.Receipt{}, err
	}
	return accesschange.Receipt{Sequence: receipt.Sequence, Watermark: receipt.Watermark}, nil
}

func stampOrBlank(st decisionledger.Status) string {
	if st.CreatedAt.IsZero() {
		return ""
	}
	return stamp(st.CreatedAt)
}

func (s *Server) ledgerRow() ledgerRow {
	if s.ledger == nil {
		return ledgerRow{State: string(decisionledger.StateIsolated), IsolationReason: string(decisionledger.IsolationMissing), File: decisionledger.RelativeFile}
	}
	st := s.ledger.Status()
	row := ledgerRow{State: string(st.State), IsolationReason: string(st.Isolation), LedgerID: st.LedgerID, CreatedAt: stampOrBlank(st), Replaces: st.Replaces,
		ReplaceReason: st.ReplaceReason, ReplacedFile: st.ReplacedFile, Records: st.Records, Watermark: st.Watermark, CheckedAt: stamp(st.CheckedAt),
		ProblemLine: st.ProblemLine, NotDurable: st.NotDurable, File: st.File}
	if st.Recovery != nil {
		row.Recovery = &recoveryRow{At: stamp(st.Recovery.At), TruncatedBytes: st.Recovery.TruncatedBytes}
	}
	return row
}
