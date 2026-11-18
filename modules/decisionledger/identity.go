package decisionledger

import (
	"time"
)

const (
	BulkRevocationJob = "bulk_revocation_job"
	SuspendMembership = "suspend_membership"
	SuspendUser       = "suspend_user"
	RolePeriodEnded   = "role_period_ended"
	RolePeriodSet     = "role_period_set"
	IncreaseIntent    = "increase_intent"
	IncreaseConfirmed = "increase_confirmed"
	IncreaseAborted   = "increase_aborted"
	ClockBaseChoice   = "clock_base"

	IncreaseReinstateUser       = "reinstate_user"
	IncreaseReinstateMembership = "reinstate_membership"
	IncreaseRolePeriod          = "role_period_set"

	ChoiceLedger = "ledger"
	ChoiceWall   = "wall"
	NoEnd        = "none"
	RoleSystem   = "system:role-period"
)

var identityKinds = map[string]bool{BulkRevocationJob: true, SuspendMembership: true, SuspendUser: true, RolePeriodEnded: true, RolePeriodSet: true,
	IncreaseIntent: true, IncreaseConfirmed: true, IncreaseAborted: true, ClockBaseChoice: true}

func IdentityKind(decision string) bool { return identityKinds[decision] }

func (l line) format() int {
	if l.Kind == KindDecision && identityKinds[l.Decision] {
		return FormatV2
	}
	return FormatV1
}

func (l line) identityEmpty() bool {
	return l.Role == "" && l.Until == "" && l.Increase == "" && l.IntentSequence == 0 && l.Effect == "" && l.Targets == 0 && l.Held == 0 && l.Excluded == 0 &&
		l.clockEmpty()
}

func (l line) clockEmpty() bool {
	return l.LedgerTime == "" && l.WallTime == "" && l.Choice == "" && l.OffsetNanos == 0
}

func validRole(role string) bool { return role == "member" || role == "administrator" }

func validUntil(until string, none bool) bool { return none && until == NoEnd || validTime(until) }

func digestHex(s string) bool { return hexID(s, 64) }

func (l line) noRequest() bool {
	return l.Request == "" && l.RequestDigest == "" && l.ApprovedBy == "" && l.DenyGeneration == 0
}

func (l line) noCounts() bool {
	return l.Effect == "" && l.Targets == 0 && l.Held == 0 && l.Excluded == 0
}

func (l line) identityShaped() bool {
	if !identityKinds[l.Decision] || l.Trust != nil || len(l.EndedDelegations) != 0 {
		return false
	}
	if l.Decision != ClockBaseChoice && !l.clockEmpty() {
		return false
	}
	switch l.Decision {
	case BulkRevocationJob:
		return l.Target != "" && l.TargetVersion == 0 && l.Revocation == "" && l.Request == "" && digestHex(l.RequestDigest) && l.RequestedBy != "" && l.ApprovedBy != "" &&
			(l.Effect == SuspendMembership || l.Effect == SuspendUser) && l.Targets > 0 && l.Held <= l.Targets && l.Role == "" && l.Until == "" &&
			l.Increase == "" && l.IntentSequence == 0
	case SuspendMembership, SuspendUser:
		return l.Target != "" && l.TargetVersion > 0 && l.Revocation != "" && l.Request != "" && digestHex(l.RequestDigest) && l.RequestedBy != "" &&
			l.ApprovedBy == "" && l.noCounts() && l.Role == "" && l.Until == "" && l.Increase == "" && l.IntentSequence == 0
	case RolePeriodEnded:
		return l.Target != "" && l.TargetVersion > 0 && l.Revocation != "" && l.RequestedBy == RoleSystem && l.noRequest() && l.noCounts() && validRole(l.Role) &&
			validUntil(l.Until, false) && l.Increase == "" && l.IntentSequence == 0
	case RolePeriodSet:
		return l.Target != "" && l.TargetVersion > 0 && l.Revocation == "" && l.RequestedBy != "" && l.noRequest() && l.noCounts() && validRole(l.Role) &&
			validUntil(l.Until, false) && l.Increase == "" && l.IntentSequence == 0
	case IncreaseIntent:
		period := l.Increase == IncreaseRolePeriod
		return l.Target != "" && l.TargetVersion > 0 && l.Revocation == "" && l.RequestedBy != "" && l.noRequest() && l.noCounts() && l.IntentSequence == 0 &&
			(period && validRole(l.Role) && validUntil(l.Until, true) || !period && (l.Increase == IncreaseReinstateUser || l.Increase == IncreaseReinstateMembership) &&
				l.Role == "" && l.Until == "")
	case IncreaseConfirmed, IncreaseAborted:
		return l.Target != "" && l.TargetVersion == 0 && l.Revocation == "" && l.RequestedBy == "" && l.noRequest() && l.noCounts() && l.Role == "" && l.Until == "" &&
			l.Increase == "" && l.IntentSequence > 0 && l.IntentSequence < l.Seq
	}
	return l.Target == "" && l.TargetVersion == 0 && l.Revocation == "" && l.RequestedBy == "" && l.noRequest() && l.noCounts() && l.Role == "" && l.Until == "" &&
		l.Increase == "" && l.IntentSequence == 0 && validTime(l.LedgerTime) && validTime(l.WallTime) &&
		(l.Choice == ChoiceWall && l.OffsetNanos == 0 || l.Choice == ChoiceLedger)
}

func (l line) decision() Decision {
	d := Decision{Workspace: l.Workspace, Decision: l.Decision, Target: l.Target, TargetVersion: l.TargetVersion, Revocation: l.Revocation, Request: l.Request,
		RequestDigest: l.RequestDigest, RequestedBy: l.RequestedBy, ApprovedBy: l.ApprovedBy, DenyGeneration: l.DenyGeneration, EndedDelegations: l.EndedDelegations,
		Role: l.Role, Until: l.Until, Increase: l.Increase, IntentSequence: l.IntentSequence, Effect: l.Effect, Targets: l.Targets, Held: l.Held, Excluded: l.Excluded}
	if l.Decision == ClockBaseChoice {
		ledgerAt, _ := time.Parse(time.RFC3339Nano, l.LedgerTime)
		wallAt, _ := time.Parse(time.RFC3339Nano, l.WallTime)
		d.Clock = &ClockBase{LedgerTime: ledgerAt, WallTime: wallAt, Choice: l.Choice, Offset: time.Duration(l.OffsetNanos)}
	}
	return d
}

func lineFor(d Decision) line {
	rec := line{Kind: KindDecision, Workspace: d.Workspace, Decision: d.Decision, Target: d.Target, TargetVersion: d.TargetVersion, Revocation: d.Revocation,
		Request: d.Request, RequestDigest: d.RequestDigest, RequestedBy: d.RequestedBy, ApprovedBy: d.ApprovedBy, DenyGeneration: d.DenyGeneration,
		EndedDelegations: append([]string(nil), d.EndedDelegations...), Trust: lineOf(d.Trust), Role: d.Role, Until: d.Until, Increase: d.Increase,
		IntentSequence: d.IntentSequence, Effect: d.Effect, Targets: d.Targets, Held: d.Held, Excluded: d.Excluded}
	if d.Clock != nil {
		rec.LedgerTime, rec.WallTime, rec.Choice, rec.OffsetNanos = stamp(d.Clock.LedgerTime), stamp(d.Clock.WallTime), d.Clock.Choice, int64(d.Clock.Offset)
	}
	return rec
}
