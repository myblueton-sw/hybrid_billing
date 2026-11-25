package server

import (
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const ledgerChunk = 256

func (s *Server) stillMatches(draft bulkrevoke.View, r roster, now time.Time) func(string) bool {
	if draft.Selection != bulkrevoke.Filter {
		return nil
	}
	matcher := newMatcher(filterOf(draft.Filter))
	return func(id string) bool {
		var member *identity.MemberInfo
		if m, ok := r.members[id]; ok {
			member = &m
		}
		return matcher.matches(s.row(r.users[id], member, now), member, now)
	}
}

func prepareBulk(s *Server, ctx commands.Context, user string) (commands.Outcome, error) {
	now := s.clock()
	draft, err := s.bulk.DraftView(ctx.Workspace, user, ctx.Target)
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.InvalidStage)
	}
	ids, _ := s.bulk.DraftUsers(ctx.Workspace, user, ctx.Target)
	r, err := s.everyone()
	if err != nil {
		return commands.Outcome{}, mapIdentity(err)
	}
	effect := bulkrevoke.Effect(ctx.Fields["effect"])
	targets, exclusions := s.classify(user, ids, effect, r, now, s.stillMatches(draft, r, now))
	if marked, _ := s.bulk.DraftChosen(ctx.Workspace, user, ctx.Target); len(marked) > 0 {
		kept := targets[:0]
		for _, t := range targets {
			if marked[t.User] {
				exclusions = append(exclusions, bulkrevoke.Exclusion{User: t.User, Login: t.Login, Reason: bulkrevoke.ExcludedChosen})
				continue
			}
			kept = append(kept, t)
		}
		targets = kept
	}
	leaving := map[string]bool{}
	for _, t := range targets {
		if !s.mayChangeRole(user, t.Role, now) {
			return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
		}
		leaving[t.User] = true
	}
	if s.strandsAnchors(r.members, leaving, now) {
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	}
	impact, err := s.impact(targets, r, now)
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	v, _, err := s.bulk.Prepare(ctx.Workspace, user, ctx.Target, effect, s.Revision(), targets, exclusions, impact)
	if errors.Is(err, bulkrevoke.ErrEmpty) && draft.Selection == bulkrevoke.Filter {
		return commands.Outcome{}, commands.Fail(commands.InvalidStage)
	}
	if err != nil {
		return commands.Outcome{}, bulkError(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "prepared", Fields: map[string]string{"bulk": v.ID, "digest": v.Digest,
		"targets": strconv.Itoa(len(v.Targets)), "excluded": strconv.Itoa(len(v.Exclusions)), "duplicates": strconv.Itoa(v.Duplicates)}}, nil
}

func changedSince(t bulkrevoke.Target, r roster) bool {
	u, known := r.users[t.User]
	m, member := r.members[t.User]
	return !known || !member || u.Epoch != t.Epoch || string(u.State) != t.UserState || m.Version != t.Version || memberState(m) != t.MembershipState ||
		string(m.Role) != t.Role
}

func (s *Server) deny(effect bulkrevoke.Effect, executor string, targets []bulkrevoke.Target) error {
	if len(targets) == 0 {
		return nil
	}
	list := make([]identity.Target, 0, len(targets))
	for _, t := range targets {
		version := t.Version
		if effect == bulkrevoke.SuspendUser {
			version = t.Epoch
		}
		list = append(list, identity.Target{User: t.User, Version: version})
	}
	if s.faulty("bulk-deny") {
		return identity.ErrConflict
	}
	if effect == bulkrevoke.SuspendUser {
		return s.identity.Installation().SuspendUsers(identity.Operator{Reference: executor, Reason: "bulk revocation"}, list)
	}
	return s.identity.Workspace().SuspendMemberships(s.caller, s.bootWorkspace, list, identity.BulkRevocation)
}

func executeBulk(s *Server, ctx commands.Context, user string) (commands.Outcome, error) {
	if s.answer.Kind() != identity.Human {
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	}
	m, err := s.bulk.Begin(ctx.Workspace, ctx.Target, ctx.Fields["digest"])
	if err != nil {
		return commands.Outcome{}, bulkError(err)
	}
	now := s.clock()
	r, err := s.everyone()
	if err != nil {
		return commands.Outcome{}, mapIdentity(err)
	}
	held, denied, leaving := map[string]string{}, []bulkrevoke.Target{}, map[string]bool{}
	for _, t := range m.Targets {
		switch {
		case changedSince(t, r):
			held[t.User] = bulkrevoke.HeldChanged
		default:
			denied, leaving[t.User] = append(denied, t), true
		}
	}
	if s.strandsAnchors(r.members, leaving, now) {
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if err := s.deny(m.Effect, user, denied); err != nil {
		return commands.Outcome{}, mapIdentity(err)
	}
	generation := s.identity.Subjects().Generation()
	v, _ := s.bulk.Execute(ctx.Workspace, m.ID, m.Digest, user, generation, held)
	s.recordJob(v)
	s.startCleanup(v)
	s.log.Info("bulk_revocation_executed", "bulk", v.ID, "targets", v.Counts.Targets, "held", v.Counts.Held, "excluded", v.Counts.Excluded)
	return commands.Outcome{Accepted: true, JobID: v.ID, Fields: map[string]string{"bulk": v.ID, "targets": strconv.Itoa(v.Counts.Targets),
		"held": strconv.Itoa(v.Counts.Held), "excluded": strconv.Itoa(v.Counts.Excluded)}}, nil
}

func itemKind(effect bulkrevoke.Effect) string {
	if effect == bulkrevoke.SuspendUser {
		return decisionledger.SuspendUser
	}
	return decisionledger.SuspendMembership
}

func (s *Server) commitBatch(batch []decisionledger.Decision) ([]decisionledger.Receipt, string) {
	receipts, err := s.ledger.CommitBatch(batch)
	if err != nil {
		st := s.ledger.Status()
		s.log.Error("decision_ledger_failed", "state", string(st.State), "sequence", st.Records, "decision", batch[0].Decision)
		return nil, commitReason(err)
	}
	return receipts, ""
}

func (s *Server) recordJob(v bulkrevoke.View) {
	header, reason := s.commitIdentity(decisionledger.Decision{Decision: decisionledger.BulkRevocationJob, Target: v.ID, RequestDigest: v.Digest, Effect: string(v.Effect),
		RequestedBy: v.Owner, ApprovedBy: v.Executor, Targets: uint64(v.Counts.Targets), Held: uint64(v.Counts.Held), Excluded: uint64(v.Counts.Excluded),
		DenyGeneration: v.DenyGeneration})
	_ = s.bulk.Header(v.Workspace, v.ID, bulkrevoke.Receipt{Sequence: header.Sequence, Watermark: header.Watermark, Reason: reason})
	if reason != "" {
		for _, item := range v.Items {
			if item.Outcome == bulkrevoke.OutcomeUnprocessed {
				s.markLedger(item.Target.User, reason)
			}
		}
		return
	}
	pending := []bulkrevoke.Item{}
	for _, item := range v.Items {
		if item.Outcome == bulkrevoke.OutcomeUnprocessed {
			pending = append(pending, item)
		}
	}
	failed := ""
	for start := 0; start < len(pending); start += ledgerChunk {
		chunk := pending[start:min(start+ledgerChunk, len(pending))]
		indexes, batch := []int{}, []decisionledger.Decision{}
		for _, item := range chunk {
			version := item.Target.Version
			if v.Effect == bulkrevoke.SuspendUser {
				version = item.Target.Epoch
			}
			indexes = append(indexes, item.Index)
			batch = append(batch, decisionledger.Decision{Workspace: s.bootWorkspace, Decision: itemKind(v.Effect), Target: item.Target.User, TargetVersion: version,
				Revocation: v.ID + ":" + strconv.Itoa(item.Index), Request: v.ID, RequestDigest: v.Digest, RequestedBy: v.Executor, DenyGeneration: v.DenyGeneration})
		}
		receipts := make([]bulkrevoke.Receipt, len(chunk))
		if failed == "" {
			var got []decisionledger.Receipt
			got, failed = s.commitBatch(batch)
			for i := range got {
				receipts[i] = bulkrevoke.Receipt{Sequence: got[i].Sequence, Watermark: got[i].Watermark}
			}
		}
		for i := range receipts {
			if failed != "" {
				receipts[i] = bulkrevoke.Receipt{Reason: failed}
			}
			s.markLedger(chunk[i].Target.User, receipts[i].Reason)
		}
		_ = s.bulk.Record(v.Workspace, v.ID, indexes, receipts)
	}
}
