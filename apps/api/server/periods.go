package server

import (
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	endAt         = "at"
	endNone       = "none"
	fieldUntil    = "role_until"
	roleEndPrefix = "role-end-"
	ledgerOK      = "confirmed"
	ledgerNot     = "unconfirmed"
)

var systemRolePeriod = identity.Operator{Reference: decisionledger.RoleSystem, Reason: "role period ended"}

func periodIncrease(old, next time.Time, restoring bool) (bool, bool) {
	switch {
	case restoring:
		return true, true
	case old.Equal(next):
		return false, false
	case next.IsZero():
		return true, true
	case old.IsZero():
		return false, true
	}
	return next.After(old), true
}

func restoring(m identity.MemberInfo) bool {
	return m.SuspendedFor == identity.RolePeriodEnded || m.State == identity.MembershipActive && !m.Effective
}

func untilText(t time.Time) string {
	if t.IsZero() {
		return decisionledger.NoEnd
	}
	return stamp(t)
}

func (s *Server) anchors(members map[string]identity.MemberInfo, now time.Time) map[string]bool {
	out := map[string]bool{}
	for user, m := range members {
		if m.Role == identity.Administrator && m.Effective && m.UserState == identity.Active && m.Until.IsZero() && s.dir.holds(user, now) {
			out[user] = true
		}
	}
	return out
}

func (s *Server) strandsAnchors(members map[string]identity.MemberInfo, leaving map[string]bool, now time.Time) bool {
	before := s.anchors(members, now)
	if len(before) == 0 {
		return false
	}
	for user := range leaving {
		delete(before, user)
	}
	return len(before) == 0
}

func (s *Server) checkUntil(c *change, raw string) (time.Time, error) {
	setter := c.members[c.ctx.User].Until
	bad := commands.Fail(commands.InvalidRequest, fieldUntil)
	if raw == "" {
		if !setter.IsZero() {
			return time.Time{}, bad
		}
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	longest := time.Duration(s.cfg.Access.RolePeriodMaxDays) * 24 * time.Hour
	if err != nil || !strings.HasSuffix(raw, "Z") || t.Second() != 0 || t.Nanosecond() != 0 || t.Before(c.now.Add(time.Minute)) || t.After(c.now.Add(longest)) ||
		!setter.IsZero() && t.After(setter) {
		return time.Time{}, bad
	}
	return t.UTC(), nil
}

func requestedEnd(fields map[string]string) (string, error) {
	end, raw := fields["end"], fields[fieldUntil]
	if end == endAt && raw == "" || end == endNone && raw != "" {
		return "", commands.Fail(commands.InvalidRequest, fieldUntil)
	}
	return raw, nil
}

func (s *Server) periodAnchors(c *change, until time.Time) error {
	if until.IsZero() {
		return nil
	}
	if s.strandsAnchors(c.members, map[string]bool{c.ctx.Target: true}, c.now) {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return nil
}

func (s *Server) commitIdentity(d decisionledger.Decision) (decisionledger.Receipt, string) {
	d.Workspace = s.bootWorkspace
	if s.ledger == nil {
		return decisionledger.Receipt{}, decisionledger.ReasonIsolated
	}
	receipt, err := s.ledger.Commit(d)
	if err != nil {
		st := s.ledger.Status()
		s.log.Error("decision_ledger_failed", "state", string(st.State), "sequence", st.Records, "decision", d.Decision)
		return decisionledger.Receipt{}, commitReason(err)
	}
	return receipt, ""
}

func commitReason(err error) string {
	if found, ok := err.(*decisionledger.CommitError); ok {
		return found.Reason
	}
	return decisionledger.ReasonCommitFailed
}

func (s *Server) recordDecrease(c *change, m identity.MemberInfo, role identity.Role, until time.Time) map[string]string {
	_, reason := s.commitIdentity(decisionledger.Decision{Decision: decisionledger.RolePeriodSet, Target: c.ctx.Target, TargetVersion: m.Version,
		Role: string(role), Until: stamp(until), RequestedBy: c.ctx.User})
	return map[string]string{"role_until": untilText(until), "ledger": s.markLedger(c.ctx.Target, reason)}
}

func (s *Server) decrease(apply func() error) error {
	if s.faulty("period-apply") {
		return identity.ErrUnavailable
	}
	return apply()
}

func (s *Server) markLedger(user, reason string) string {
	state := ledgerOK
	if reason != "" {
		state = ledgerNot
	}
	s.memberLedger[user] = state
	return state
}

func setRolePeriod(s *Server, c *change) (string, map[string]string, error) {
	raw, err := requestedEnd(c.ctx.Fields)
	if err != nil {
		return "", nil, err
	}
	m, member := c.members[c.ctx.Target]
	if c.ctx.Target == c.ctx.User {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if !member {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	until, err := s.checkUntil(c, raw)
	if err != nil {
		return "", nil, err
	}
	restores := restoring(m)
	increase, changed := periodIncrease(m.Until, until, restores)
	if !changed {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	if m.Role == identity.Administrator && !restores {
		if err := s.periodAnchors(c, until); err != nil {
			return "", nil, err
		}
	}
	state := "role_period_set"
	if restores {
		state = "role_period_restored"
	}
	ws := s.identity.Workspace()
	if increase {
		if err := s.increaseAllowed(true); err != nil {
			return "", nil, err
		}
		fields, err := s.runIncrease(c, increaseStep{kind: decisionledger.IncreaseRolePeriod, version: m.Version, role: string(m.Role), until: untilText(until),
			check: func() error {
				return ws.CheckSetRolePeriod(s.caller, s.bootWorkspace, c.ctx.Target, until, m.Version, c.now)
			},
			apply: func() error {
				return ws.SetRolePeriod(s.caller, s.bootWorkspace, c.ctx.Target, until, m.Version, c.now)
			}})
		if err != nil {
			return "", nil, err
		}
		fields["role_until"] = untilText(until)
		return state, fields, nil
	}
	if err := s.decrease(func() error {
		return ws.SetRolePeriod(s.caller, s.bootWorkspace, c.ctx.Target, until, m.Version, c.now)
	}); err != nil {
		return "", nil, mapIdentity(err)
	}
	return state, s.recordDecrease(c, m, m.Role, until), nil
}

func (s *Server) sweepRoles() bool {
	if s.bootWorkspace == "" {
		return false
	}
	now := s.clock()
	expired, _ := s.identity.Installation().ExpireRoles(systemRolePeriod, s.bootWorkspace, now)
	changed := false
	for _, e := range expired {
		if e.Blocked {
			s.log.Error("role_expiry_blocked", "user", e.User)
			continue
		}
		changed = true
		if login, known := s.dir.loginOf(e.User); known {
			s.endSessions(login)
		}
		s.dir.dropPickups(func(p pickup) bool { return p.target == e.User })
		s.roleEnds++
		_, reason := s.commitIdentity(decisionledger.Decision{Decision: decisionledger.RolePeriodEnded, Target: e.User, TargetVersion: e.Version, Role: string(e.Role),
			Until: stamp(e.Until), Revocation: roleEndPrefix + strconv.FormatUint(s.roleEnds, 10), RequestedBy: decisionledger.RoleSystem})
		state := s.markLedger(e.User, reason)
		s.log.Info("role_period_ended", "user", e.User, "ledger", state)
	}
	return changed
}

func (s *Server) holdPeriodDecrease(caller identity.Verifier, call commands.Call, command string) bool {
	members, err := s.roster(caller)
	m, ok := members[call.Target]
	fields := map[string]string{}
	for _, v := range call.Fields {
		fields[v.Name] = v.Value
	}
	if err != nil || !ok || command == "identity.change_role" && fields["role"] != string(identity.Member) && fields["role"] != string(identity.Administrator) {
		return false
	}
	until := time.Time{}
	switch raw := fields[fieldUntil]; {
	case raw != "":
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return false
		}
		until = t.UTC()
	case command == "identity.change_role" && fields["end"] != endNone:
		until = m.Until
	}
	increase, _ := periodIncrease(m.Until, until, restoring(m))
	return !increase && !(command == "identity.change_role" && promotes(identity.Role(fields["role"]), m))
}
