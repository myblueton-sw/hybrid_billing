package server

import (
	"fmt"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
)

type clockChoice struct {
	flag        string
	ledgerTime  time.Time
	wallTime    time.Time
	offset      time.Duration
	recorded    bool
	line        int
	provisional bool
}

func short(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func (s *Server) setClockBase() error {
	stamps, err := decisionledger.Scan(s.cfg.Data)
	if err != nil {
		return fmt.Errorf("%w: decision ledger pre-scan: %w", ErrStart, err)
	}
	tolerance := time.Duration(s.cfg.Access.ClockSkewHours) * time.Hour
	wall := s.time.Wall()
	var chosen *decisionledger.Stamp
	for i := range stamps {
		if stamps[i].Clock != nil {
			chosen = &stamps[i]
		}
	}
	latest, ahead := time.Time{}, false
	for _, st := range stamps {
		against := wall
		if chosen != nil && st.Line <= chosen.Line {
			if chosen.Clock.Choice == decisionledger.ChoiceWall {
				continue
			}
			against = wall.Add(chosen.Clock.Offset)
		}
		latest = maxTime(latest, st.At)
		ahead = ahead || st.At.Sub(against) > tolerance
	}
	flag := s.cfg.Access.ClockBase
	s.clockChoice = clockChoice{flag: flag, ledgerTime: latest, wallTime: wall}
	if chosen != nil {
		s.clockChoice.line = chosen.Line
	}
	hours := s.cfg.Access.ClockSkewHours
	switch {
	case ahead && flag == "":
		return fmt.Errorf("%w: The decision ledger's latest time %s is more than %d hours ahead of this host's clock %s. Correct the host clock, or restart with -clock-base=ledger or -clock-base=wall.",
			ErrStart, short(latest), hours, short(wall))
	case flag != "" && !ahead && chosen == nil:
		return s.flagRefused()
	case flag != "" && !ahead:
		s.clockChoice.provisional = true
		s.time.rebase(maxTime(latest, wall), tolerance)
	case flag == ClockBaseLedger:
		s.clockChoice.offset = latest.Sub(wall)
		s.time.rebase(maxTime(latest, wall), tolerance)
	case flag == ClockBaseWall:
		s.time.rebase(wall, tolerance)
	default:
		s.time.rebase(maxTime(latest, wall), tolerance)
	}
	return nil
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (s *Server) flagRefused() error {
	return fmt.Errorf("%w: -clock-base is accepted only when the decision ledger's latest time is more than %d hours ahead of this host's clock, or when its latest clock base choice cannot be verified",
		ErrConfig, s.cfg.Access.ClockSkewHours)
}

func (s *Server) verifiedChoice(line int) bool {
	for _, rec := range s.ledger.Verified() {
		if rec.Sequence == uint64(line) && rec.Decision.Decision == decisionledger.ClockBaseChoice {
			return true
		}
	}
	return false
}

func (s *Server) settleClockBase() error {
	line, flag := s.clockChoice.line, s.clockChoice.flag
	verified := line > 0 && s.verifiedChoice(line)
	switch {
	case line > 0 && !verified && flag == "":
		return fmt.Errorf("%w: The clock base choice at decision ledger line %d could not be verified. Correct the host clock, or restart with -clock-base=ledger or -clock-base=wall.",
			ErrStart, line)
	case s.clockChoice.provisional && verified:
		return s.flagRefused()
	case flag == "":
		return nil
	}
	c := s.clockChoice
	_, reason := s.commitIdentity(decisionledger.Decision{Decision: decisionledger.ClockBaseChoice, Clock: &decisionledger.ClockBase{LedgerTime: c.ledgerTime,
		WallTime: c.wallTime, Choice: c.flag, Offset: c.offset}})
	s.clockChoice.recorded = reason == ""
	s.log.Info("clock_base_chosen", "choice", c.flag, "ledger_time", stamp(c.ledgerTime), "wall_time", stamp(c.wallTime), "recorded", s.clockChoice.recorded)
	return nil
}
