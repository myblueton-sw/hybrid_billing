package decisionledger

import "time"

type stepLine struct {
	Action        string `json:"action"`
	MaxAgeSeconds int64  `json:"max_age_seconds"`
	MFA           bool   `json:"mfa"`
}

type configLine struct {
	Methods       []string   `json:"methods"`
	MFA           string     `json:"mfa"`
	MaxAgeSeconds int64      `json:"max_age_seconds"`
	StepUp        []stepLine `json:"step_up"`
}

type trustLine struct {
	Change           string     `json:"change"`
	ChangeRevision   uint64     `json:"change_revision"`
	ChangeDigest     string     `json:"change_digest"`
	EvidenceDigest   string     `json:"evidence_digest"`
	OldEpoch         uint64     `json:"old_epoch"`
	NewEpoch         uint64     `json:"new_epoch"`
	OldPolicy        uint64     `json:"old_policy"`
	NewPolicy        uint64     `json:"new_policy"`
	OldDigest        string     `json:"old_digest"`
	NewDigest        string     `json:"new_digest"`
	Config           configLine `json:"config"`
	ProposedBy       string     `json:"proposed_by"`
	ValidatedBy      string     `json:"validated_by"`
	ApprovedBy       string     `json:"approved_by"`
	ActivatedBy      string     `json:"activated_by"`
	ApprovedAt       string     `json:"approved_at"`
	RunStartedAt     string     `json:"run_started_at"`
	InvalidateBelow  uint64     `json:"invalidate_below_epoch"`
	AffectedSessions uint64     `json:"affected_sessions"`
	Origin           string     `json:"origin"`
	CarriedFrom      string     `json:"carried_from,omitempty"`
	CarriedLine      int        `json:"carried_line,omitempty"`
}

func lineOf(t *Trust) *trustLine {
	if t == nil {
		return nil
	}
	c := configLine{Methods: append([]string{}, t.Config.Methods...), MFA: t.Config.MFA, MaxAgeSeconds: t.Config.MaxAgeSeconds, StepUp: []stepLine{}}
	for _, r := range t.Config.StepUp {
		c.StepUp = append(c.StepUp, stepLine{Action: r.Action, MaxAgeSeconds: r.MaxAgeSeconds, MFA: r.MFA})
	}
	return &trustLine{Change: t.Change, ChangeRevision: t.ChangeRevision, ChangeDigest: t.ChangeDigest, EvidenceDigest: t.EvidenceDigest, OldEpoch: t.OldEpoch,
		NewEpoch: t.NewEpoch, OldPolicy: t.OldPolicy, NewPolicy: t.NewPolicy, OldDigest: t.OldDigest, NewDigest: t.NewDigest, Config: c, ProposedBy: t.ProposedBy,
		ValidatedBy: t.ValidatedBy, ApprovedBy: t.ApprovedBy, ActivatedBy: t.ActivatedBy, ApprovedAt: stamp(t.ApprovedAt), RunStartedAt: stamp(t.RunStartedAt),
		InvalidateBelow: t.InvalidateBelow, AffectedSessions: t.AffectedSessions, Origin: t.Origin, CarriedFrom: t.CarriedFrom, CarriedLine: t.CarriedLine}
}

func (l *trustLine) trust() *Trust {
	c := Config{Methods: append([]string{}, l.Config.Methods...), MFA: l.Config.MFA, MaxAgeSeconds: l.Config.MaxAgeSeconds, StepUp: []StepRule{}}
	for _, r := range l.Config.StepUp {
		c.StepUp = append(c.StepUp, StepRule{Action: r.Action, MaxAgeSeconds: r.MaxAgeSeconds, MFA: r.MFA})
	}
	approved, _ := time.Parse(time.RFC3339Nano, l.ApprovedAt)
	started, _ := time.Parse(time.RFC3339Nano, l.RunStartedAt)
	return &Trust{Change: l.Change, ChangeRevision: l.ChangeRevision, ChangeDigest: l.ChangeDigest, EvidenceDigest: l.EvidenceDigest, OldEpoch: l.OldEpoch,
		NewEpoch: l.NewEpoch, OldPolicy: l.OldPolicy, NewPolicy: l.NewPolicy, OldDigest: l.OldDigest, NewDigest: l.NewDigest, Config: c, ProposedBy: l.ProposedBy,
		ValidatedBy: l.ValidatedBy, ApprovedBy: l.ApprovedBy, ActivatedBy: l.ActivatedBy, ApprovedAt: approved, RunStartedAt: started,
		InvalidateBelow: l.InvalidateBelow, AffectedSessions: l.AffectedSessions, Origin: l.Origin, CarriedFrom: l.CarriedFrom, CarriedLine: l.CarriedLine}
}

func (c configLine) shaped() bool {
	if len(c.Methods) == 0 || c.MFA == "" || c.MaxAgeSeconds <= 0 || c.StepUp == nil {
		return false
	}
	for _, m := range c.Methods {
		if m == "" {
			return false
		}
	}
	for _, r := range c.StepUp {
		if r.Action == "" || r.MaxAgeSeconds <= 0 {
			return false
		}
	}
	return true
}

func (l *trustLine) shaped() bool {
	origin := l.Origin == OriginActivation && l.CarriedFrom == "" && l.CarriedLine == 0 || l.Origin == OriginCarried && l.CarriedFrom != "" && l.CarriedLine >= 0
	actors := l.ProposedBy != "" && l.ValidatedBy != "" && l.ApprovedBy != "" && l.ActivatedBy != ""
	return origin && actors && l.Change != "" && l.ChangeRevision > 0 && hexID(l.ChangeDigest, 64) && hexID(l.EvidenceDigest, 64) && hexID(l.OldDigest, 64) &&
		hexID(l.NewDigest, 64) && l.NewEpoch&(1<<32-1) != 0 && l.NewEpoch>>32 < MaxGeneration && l.NewEpoch > l.OldEpoch && l.NewPolicy > l.OldPolicy && l.InvalidateBelow == l.NewEpoch &&
		validTime(l.ApprovedAt) && validTime(l.RunStartedAt) && l.Config.shaped()
}

func Fits(d Decision) bool {
	rec := line{V: FormatV1, Seq: 1 << 62, Prev: genesisPrev, At: stamp(time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)), Kind: KindDecision,
		Workspace: d.Workspace, Decision: d.Decision, Trust: lineOf(d.Trust)}
	return rec.shaped() && len(encode(rec)) <= RecordLimit
}
