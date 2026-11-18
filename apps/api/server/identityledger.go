package server

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	noteReapplied = "reapplied_at_start"
	noteAbsent    = "target_not_present"
	noteRole      = "role_differs_after_restart"
	noteAnchor    = "kept_as_administrator_without_end"
	noteIgnored   = "increase_not_confirmed"
	noteAwaiting  = "applies_when_membership_returns"
	noteLater     = "superseded_by_a_confirmed_increase"
)

var replayOperator = identity.Operator{Reference: "system:ledger-replay", Reason: "decision ledger replay"}

type aspect struct {
	set    bool
	state  string
	reason identity.SuspendReason
	seq    uint64
}

type endAspect struct {
	set   bool
	until time.Time
	role  string
	seq   uint64
}

type replayState struct {
	user   map[string]aspect
	member map[string]aspect
	end    map[string]endAspect
	seqs   map[string][]uint64
}

func parseEnd(raw string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, raw)
	return t.UTC()
}

func (s *Server) replayInputs(records []decisionledger.Record, degraded bool) replayState {
	confirmed, aborted := map[uint64]bool{}, map[uint64]bool{}
	for _, rec := range records {
		switch rec.Decision.Decision {
		case decisionledger.IncreaseConfirmed:
			confirmed[rec.Decision.IntentSequence] = true
		case decisionledger.IncreaseAborted:
			aborted[rec.Decision.IntentSequence] = true
		}
	}
	st := replayState{user: map[string]aspect{}, member: map[string]aspect{}, end: map[string]endAspect{}, seqs: map[string][]uint64{}}
	for _, rec := range records {
		d := rec.Decision
		if d.Workspace != s.bootWorkspace || !decisionledger.IdentityKind(d.Decision) {
			continue
		}
		switch d.Decision {
		case decisionledger.SuspendUser:
			st.user[d.Target] = aspect{set: true, state: string(identity.Suspended), seq: rec.Sequence}
		case decisionledger.SuspendMembership:
			st.member[d.Target] = aspect{set: true, state: string(identity.MembershipSuspended), reason: identity.BulkRevocation, seq: rec.Sequence}
		case decisionledger.RolePeriodEnded:
			st.member[d.Target] = aspect{set: true, state: string(identity.MembershipSuspended), reason: identity.RolePeriodEnded, seq: rec.Sequence}
			st.end[d.Target] = endAspect{set: true, until: parseEnd(d.Until), role: d.Role, seq: rec.Sequence}
		case decisionledger.RolePeriodSet:
			st.end[d.Target] = endAspect{set: true, until: parseEnd(d.Until), role: d.Role, seq: rec.Sequence}
		case decisionledger.IncreaseIntent:
			if degraded || !confirmed[rec.Sequence] || aborted[rec.Sequence] {
				s.replayNotes[strconv.FormatUint(rec.Sequence, 10)] = noteIgnored
				continue
			}
			switch d.Increase {
			case decisionledger.IncreaseReinstateUser:
				st.user[d.Target] = aspect{set: true, state: string(identity.Active), seq: rec.Sequence}
			case decisionledger.IncreaseReinstateMembership:
				st.member[d.Target] = aspect{set: true, state: string(identity.MembershipActive), seq: rec.Sequence}
			default:
				if m := st.member[d.Target]; m.set && m.reason == identity.RolePeriodEnded {
					st.member[d.Target] = aspect{set: true, state: string(identity.MembershipActive), seq: rec.Sequence}
				}
				until := time.Time{}
				if d.Until != decisionledger.NoEnd {
					until = parseEnd(d.Until)
				}
				st.end[d.Target] = endAspect{set: true, until: until, role: d.Role, seq: rec.Sequence}
			}
		default:
			continue
		}
		st.seqs[d.Target] = append(st.seqs[d.Target], rec.Sequence)
	}
	return st
}

func (s *Server) operatorMembers() map[string]identity.MemberInfo {
	list, _ := s.identity.Installation().Members(replayOperator, s.bootWorkspace)
	out := map[string]identity.MemberInfo{}
	for _, m := range list {
		out[m.User] = m
	}
	return out
}

func (s *Server) replayIdentity() (bool, error) {
	records := s.ledger.Decisions()
	st := s.ledger.Status()
	degraded := st.State == decisionledger.StateIsolated || st.NotDurable
	s.restoreCounters(records)
	inputs := s.replayInputs(records, degraded)
	now := s.clock()
	before := s.operatorMembers()
	anchorsBefore := len(s.anchors(before, now))
	changed := false
	users := []string{}
	for user := range inputs.seqs {
		users = append(users, user)
	}
	sort.Strings(users)
	for _, user := range users {
		m, member := before[user]
		u, ua := inputs.user[user], inputs.member[user]
		rec := identity.Recorded{User: user}
		if u.set && u.state == string(identity.Suspended) {
			rec.UserState = identity.Suspended
		}
		if !member && (ua.set && ua.state == string(identity.MembershipSuspended) || inputs.end[user].set) {
			s.awaiting[user] = awaitingReplay{member: ua, end: inputs.end[user], seqs: inputs.seqs[user]}
			s.noteAll(inputs.seqs[user], noteAwaiting)
		}
		if member && ua.set && ua.state == string(identity.MembershipSuspended) {
			rec.Membership, rec.Reason = identity.MembershipSuspended, ua.reason
		}
		e := inputs.end[user]
		if e.set && member && e.role != string(m.Role) {
			s.noteAll(inputs.seqs[user], noteRole)
			inputs.end[user] = endAspect{}
		}
		if rec.UserState == "" && rec.Membership == "" {
			if member && !e.set {
				s.noteAll(inputs.seqs[user], noteLater)
			}
			continue
		}
		if err := s.identity.Installation().ApplyRecorded(replayOperator, s.bootWorkspace, rec); err != nil {
			s.noteAll(inputs.seqs[user], noteAbsent)
			continue
		}
		s.noteAll(inputs.seqs[user], noteReapplied)
		changed = true
	}
	changed = s.applyEnds(inputs, users, now) || changed
	if anchorsBefore > 0 && len(s.anchors(s.operatorMembers(), now)) == 0 {
		for _, user := range users {
			s.stranded = append(s.stranded, inputs.seqs[user]...)
		}
		sort.Slice(s.stranded, func(a, b int) bool { return s.stranded[a] < s.stranded[b] })
		s.log.Error("administration_stranded", "records", s.stranded)
	}
	s.rebuildJobs(records)
	return changed, nil
}

func (s *Server) noteAll(seqs []uint64, note string) {
	for _, seq := range seqs {
		key := strconv.FormatUint(seq, 10)
		if s.replayNotes[key] == "" || note != noteReapplied {
			s.replayNotes[key] = note
		}
	}
}

func (s *Server) applyEnds(inputs replayState, users []string, now time.Time) bool {
	members := s.operatorMembers()
	pending := map[string]time.Time{}
	for _, user := range users {
		if e := inputs.end[user]; e.set {
			if _, member := members[user]; member {
				pending[user] = e.until
			}
		}
	}
	anchorAfter := func(skip map[string]bool) bool {
		for user, m := range members {
			until := m.Until
			if t, ok := pending[user]; ok && !skip[user] {
				until = t
			}
			if m.Role == identity.Administrator && m.State == identity.MembershipActive && m.UserState == identity.Active && until.IsZero() && s.dir.holds(user, now) {
				return true
			}
		}
		return false
	}
	skip := map[string]bool{}
	if !anchorAfter(skip) {
		for _, user := range users {
			if _, ok := pending[user]; ok && members[user].Role == identity.Administrator {
				skip[user] = true
				s.noteAll(inputs.seqs[user], noteAnchor)
				if anchorAfter(skip) {
					break
				}
			}
		}
	}
	changed := false
	for _, user := range users {
		until, ok := pending[user]
		if !ok || skip[user] {
			continue
		}
		if s.identity.Installation().ApplyRecorded(replayOperator, s.bootWorkspace, identity.Recorded{User: user, SetUntil: true, Until: until}) == nil {
			s.noteAll(inputs.seqs[user], noteReapplied)
			changed = true
		}
	}
	return changed
}

func (s *Server) restoreCounters(records []decisionledger.Record) {
	for _, rec := range records {
		d := rec.Decision
		if n, ok := bulkrevoke.Number(d.Revocation, roleEndPrefix); ok {
			s.roleEnds = max(s.roleEnds, n)
		}
		for _, id := range []string{d.Target, d.Request} {
			if n, ok := bulkrevoke.Number(id, bulkrevoke.ManifestPrefix); ok {
				s.bulk.Restore(n, nil)
			}
		}
	}
}

func itemIndex(revocation, id string) (int, bool) {
	rest, ok := strings.CutPrefix(revocation, id+":")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil && n >= 0
}

func (s *Server) rebuildJobs(records []decisionledger.Record) {
	jobs, targets := map[string]*bulkrevoke.View{}, map[string]int{}
	order := []string{}
	for _, rec := range records {
		d := rec.Decision
		if d.Workspace != s.bootWorkspace || d.Decision != decisionledger.BulkRevocationJob {
			continue
		}
		n, ok := bulkrevoke.Number(d.Target, bulkrevoke.ManifestPrefix)
		if !ok {
			continue
		}
		jobs[d.Target] = &bulkrevoke.View{Number: n, Workspace: d.Workspace, Effect: bulkrevoke.Effect(d.Effect), Owner: d.RequestedBy, Executor: d.ApprovedBy,
			Digest: d.RequestDigest, DenyGeneration: d.DenyGeneration, RecordedAt: rec.At, ExecutedAt: rec.At, DenyAt: rec.At, Selection: bulkrevoke.Explicit,
			ReplayHeld: int(d.Held), ReplayExcluded: int(d.Excluded), Header: bulkrevoke.Receipt{Sequence: rec.Sequence, Watermark: rec.Watermark}, Headed: true}
		order, targets[d.Target] = append(order, d.Target), int(d.Targets)
	}
	for _, rec := range records {
		d := rec.Decision
		v := jobs[d.Request]
		if v == nil || d.Decision != decisionledger.SuspendMembership && d.Decision != decisionledger.SuspendUser || d.Workspace != s.bootWorkspace {
			continue
		}
		index, ok := itemIndex(d.Revocation, d.Request)
		if !ok {
			continue
		}
		note := s.replayNotes[strconv.FormatUint(rec.Sequence, 10)]
		login, _ := s.dir.loginOf(d.Target)
		cleanup := bulkCleanupItems(true)
		v.Items = append(v.Items, bulkrevoke.Item{Index: index, Target: bulkrevoke.Target{User: d.Target, Login: login}, Outcome: bulkrevoke.OutcomeSucceeded,
			State: bulkrevoke.Complete, Receipt: bulkrevoke.Receipt{Sequence: rec.Sequence, Watermark: rec.Watermark}, Cleanup: cleanup, Note: note})
	}
	kept := []bulkrevoke.View{}
	for i := len(order) - 1; i >= 0 && len(kept) < s.cfg.Access.BulkKept; i-- {
		v := jobs[order[i]]
		sort.Slice(v.Items, func(a, b int) bool { return v.Items[a].Index < v.Items[b].Index })
		v.ReplayMissing = max(0, targets[order[i]]-v.ReplayHeld-len(v.Items))
		kept = append(kept, *v)
	}
	s.bulk.Restore(0, kept)
}

type awaitingReplay struct {
	member aspect
	end    endAspect
	seqs   []uint64
}

func (s *Server) applyAwaiting(user string) {
	w, ok := s.awaiting[user]
	if !ok {
		return
	}
	delete(s.awaiting, user)
	m := s.operatorMembers()[user]
	rec := identity.Recorded{User: user}
	if w.member.set && w.member.state == string(identity.MembershipSuspended) {
		rec.Membership, rec.Reason = identity.MembershipSuspended, w.member.reason
	}
	if w.end.set && w.end.role == string(m.Role) {
		rec.SetUntil, rec.Until = true, w.end.until
	}
	if rec.Membership == "" && !rec.SetUntil {
		s.noteAll(w.seqs, noteRole)
		return
	}
	if s.identity.Installation().ApplyRecorded(replayOperator, s.bootWorkspace, rec) == nil {
		for _, seq := range w.seqs {
			s.replayNotes[strconv.FormatUint(seq, 10)] = noteReapplied
		}
		s.log.Info("replayed_at_acceptance", "user", user)
	}
}
