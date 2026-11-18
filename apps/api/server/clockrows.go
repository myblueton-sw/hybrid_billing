package server

import "time"

type clockBaseRow struct {
	Choice     string `json:"choice"`
	LedgerTime string `json:"ledger_time"`
	WallTime   string `json:"wall_time"`
	Recorded   bool   `json:"recorded"`
}

type leadRow struct {
	WallTime   string `json:"wall_time"`
	ServerTime string `json:"server_time"`
}

type recoveryState struct {
	Stranded bool     `json:"stranded"`
	Records  []uint64 `json:"records"`
}

type clockRows struct {
	ServerTime  string        `json:"server_time"`
	WallTime    string        `json:"wall_time"`
	ClockBehind bool          `json:"clock_behind"`
	ClockAhead  bool          `json:"clock_ahead"`
	ClockBase   clockBaseRow  `json:"clock_base"`
	WallLead    *leadRow      `json:"wall_lead_beyond_tolerance"`
	Recovery    recoveryState `json:"recovery_state"`
}

func (s *Server) clockRows(now time.Time) clockRows {
	wall := s.time.Wall()
	out := clockRows{ServerTime: stamp(now), WallTime: stamp(wall), ClockBehind: now.Sub(wall) > clockDiagnostic, ClockAhead: wall.Sub(now) > clockDiagnostic,
		ClockBase: clockBaseRow{Choice: "default"}, Recovery: recoveryState{Stranded: len(s.stranded) > 0, Records: append([]uint64{}, s.stranded...)}}
	if c := s.clockChoice; c.flag != "" {
		out.ClockBase = clockBaseRow{Choice: c.flag, LedgerTime: stamp(c.ledgerTime), WallTime: stamp(c.wallTime), Recorded: c.recorded}
	}
	if lead, beyond := s.time.leadBeyond(); beyond {
		out.WallLead = &leadRow{WallTime: stamp(lead.wall), ServerTime: stamp(lead.server)}
	}
	return out
}
