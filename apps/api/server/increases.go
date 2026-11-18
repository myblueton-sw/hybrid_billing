package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

var errIncreaseApply = errors.New("an increase stopped after its intent was recorded")

const (
	durableYes     = "yes"
	durableNo      = "no"
	durableUnknown = "unknown"
)

type increaseStep struct {
	kind    string
	version uint64
	role    string
	until   string
	check   func() error
	apply   func() error
}

func refusal(err error) bool {
	for _, kind := range []error{identity.ErrInvalid, identity.ErrRefused, identity.ErrConflict, identity.ErrLastAdministrator, identity.ErrLimit, identity.ErrUnavailable} {
		if errors.Is(err, kind) {
			return true
		}
	}
	return false
}

func (s *Server) applyOnce(apply func() error) (err error) {
	defer func() {
		if recover() != nil {
			err = errIncreaseApply
		}
	}()
	switch {
	case s.faulty("increase-apply"):
		return errIncreaseApply
	case s.faulty("increase-refuse"):
		return identity.ErrConflict
	case s.faulty("increase-panic"):
		panic("increase")
	}
	return apply()
}

func (s *Server) runIncrease(c *change, step increaseStep) (map[string]string, error) {
	if err := step.check(); err != nil {
		return nil, mapIdentity(err)
	}
	intent, reason := s.commitIdentity(decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: c.ctx.Target, TargetVersion: step.version,
		Increase: step.kind, Role: step.role, Until: step.until, RequestedBy: c.ctx.User})
	if reason != "" {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	if err := s.applyOnce(step.apply); err != nil {
		if refusal(err) {
			s.commitIdentity(decisionledger.Decision{Decision: decisionledger.IncreaseAborted, Target: c.ctx.Target, IntentSequence: intent.Sequence})
			return nil, mapIdentity(err)
		}
		s.ledger.Stop()
		s.log.Error("increase_apply_failed", "target", c.ctx.Target, "intent", intent.Sequence)
		return nil, errIncreaseApply
	}
	if s.faulty("increase-confirm") {
		s.ledger.Stop()
	}
	_, reason = s.commitIdentity(decisionledger.Decision{Decision: decisionledger.IncreaseConfirmed, Target: c.ctx.Target, IntentSequence: intent.Sequence})
	durable := durableYes
	switch reason {
	case "":
	case decisionledger.ReasonCommitUnknown:
		durable = durableUnknown
	default:
		durable = durableNo
	}
	if reason != "" {
		s.ledger.Stop()
		s.log.Error("increase_not_durable", "target", c.ctx.Target, "intent", intent.Sequence, "reason", reason)
	}
	return map[string]string{"durable": durable, "ledger": s.markLedger(c.ctx.Target, reason)}, nil
}
