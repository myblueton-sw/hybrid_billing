package server

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

const (
	originSeed     = "seed"
	originProcess  = "this_process"
	originReplay   = "ledger_replay"
	originCarried  = "carried"
	seedPolicyAge  = 12 * time.Hour
	maxEpochNumber = 1<<32 - 1
)

type trustOrigin struct {
	origin       string
	carriedFrom  string
	carriedLine  int
	unconfirmed  bool
	activatedAt  time.Time
	change       string
	resetAt      time.Time
	olderIgnored bool
	stricter     []string
}

type trustHold struct {
	reason string
	since  time.Time
}

type trustProgress struct {
	ended     int
	remaining int
}

func (s *Server) generation() uint64 {
	if s.ledger == nil {
		return 0
	}
	return s.ledger.Status().Generation
}

func (s *Server) currentEpoch(workspace string) uint64 {
	if current, err := s.policy.Policy(workspace); err == nil {
		return current.Epoch
	}
	return s.generation() << 32
}

func (s *Server) trustBase(workspace string) (trustchange.Base, authpolicy.Revision) {
	current, err := s.policy.Policy(workspace)
	if err != nil {
		return trustchange.Base{Epoch: s.generation() << 32, Digest: trustchange.EmptyDigest()}, authpolicy.Revision{}
	}
	return trustchange.Base{Epoch: current.Epoch, Policy: current.Number, Digest: trustchange.ConfigDigest(candidateOf(current.Revision))}, current.Revision
}

func candidateOf(r authpolicy.Revision) trustchange.Candidate {
	c := trustchange.Candidate{Methods: append([]string(nil), r.Methods...), MFA: r.MFA, MaxAge: r.MaxAge}
	for _, rule := range r.StepUp {
		c.StepUp = append(c.StepUp, trustchange.Rule{Action: rule.Action, MaxAge: rule.MaxAge, MFA: rule.MFA})
	}
	return c
}

func revisionFor(workspace string, c trustchange.Candidate) authpolicy.Revision {
	r := authpolicy.Revision{Workspace: workspace, Methods: append([]string(nil), c.Methods...), MFA: c.MFA, MaxAge: c.MaxAge}
	for _, rule := range c.StepUp {
		r.StepUp = append(r.StepUp, authpolicy.StepUp{Action: rule.Action, MaxAge: rule.MaxAge, MFA: rule.MFA})
	}
	return r
}

func configOf(c trustchange.Candidate) decisionledger.Config {
	out := decisionledger.Config{Methods: append([]string{}, c.Methods...), MFA: c.MFA, MaxAgeSeconds: int64(c.MaxAge / time.Second), StepUp: []decisionledger.StepRule{}}
	for _, rule := range c.StepUp {
		out.StepUp = append(out.StepUp, decisionledger.StepRule{Action: rule.Action, MaxAgeSeconds: int64(rule.MaxAge / time.Second), MFA: rule.MFA})
	}
	return out
}

func candidateFrom(c decisionledger.Config) trustchange.Candidate {
	out := trustchange.Candidate{Methods: append([]string(nil), c.Methods...), MFA: c.MFA, MaxAge: time.Duration(c.MaxAgeSeconds) * time.Second}
	for _, rule := range c.StepUp {
		out.StepUp = append(out.StepUp, trustchange.Rule{Action: rule.Action, MaxAge: time.Duration(rule.MaxAgeSeconds) * time.Second, MFA: rule.MFA})
	}
	return out
}

func nextEpoch(current, generation uint64) (uint64, bool) {
	if authpolicy.EpochGeneration(current) == generation && authpolicy.EpochNumber(current) == maxEpochNumber {
		return 0, false
	}
	return max(current+1, generation<<32+1), true
}

func (s *Server) publishTrust(p authpolicy.Publication, origin trustOrigin) error {
	if _, err := s.policy.Publish(p); err != nil {
		return err
	}
	s.trustOrigin = origin
	return nil
}

func latestTrust(records []decisionledger.Record, workspace string) (decisionledger.Record, bool) {
	for i := len(records) - 1; i >= 0; i-- {
		if d := records[i].Decision; d.Trust != nil && d.Workspace == workspace {
			return records[i], true
		}
	}
	return decisionledger.Record{}, false
}

func problemLine(p decisionledger.Prior) int {
	if p.Isolation == decisionledger.IsolationIntegrity || p.Isolation == decisionledger.IsolationOrder {
		return p.ProblemLine
	}
	return 0
}

func (s *Server) trustSuccessor(p decisionledger.Prior) (decisionledger.Successor, error) {
	cfg := s.cfg.Access
	next, known := p.Next()
	stated := false
	switch {
	case !known && cfg.TrustGeneration == 0:
		return decisionledger.Successor{}, fmt.Errorf("%w: trust-generation: the decision ledger marker and the ledger are both unusable; state the new generation", ErrConfig)
	case !known && cfg.TrustGeneration >= decisionledger.MaxGeneration:
		return decisionledger.Successor{}, fmt.Errorf("%w: trust-generation must be below %d", ErrConfig, uint64(decisionledger.MaxGeneration))
	case !known:
		next, stated = cfg.TrustGeneration, true
	case cfg.TrustGeneration != 0:
		return decisionledger.Successor{}, fmt.Errorf("%w: trust-generation: the generation is known from the marker or the ledger", ErrConfig)
	}
	if floor, ok := p.Floor(); ok && next <= floor {
		if stated {
			return decisionledger.Successor{}, fmt.Errorf("%w: trust-generation must be at least %d, above every generation found in the marker, the current ledger and the retained replaced ledgers", ErrConfig, floor+1)
		}
		next = floor + 1
	}
	if next >= decisionledger.MaxGeneration {
		return decisionledger.Successor{}, fmt.Errorf("%w: %w", ErrConfig, decisionledger.ErrGenerations)
	}
	line := problemLine(p)
	out := decisionledger.Successor{Generation: next, GenerationStated: stated, ReplacedLine: line}
	rec, found := latestTrust(p.Verified, s.bootWorkspace)
	matched := p.MarkerUsable && p.LedgerID != "" && p.LedgerID == p.MarkerID
	switch {
	case cfg.CarryTrustFrom != "" && (cfg.AcceptTrustReset || cfg.CarryTrustFrom != p.LedgerID || !found):
		return decisionledger.Successor{}, fmt.Errorf("%w: carry-trust-from does not name the replaced ledger or it has no trust record for this workspace", ErrConfig)
	case cfg.CarryTrustFrom == "" && !(matched && found) && !cfg.AcceptTrustReset:
		return decisionledger.Successor{}, fmt.Errorf("%w: the replaced decision ledger has no carryable trust record; pass accept-trust-reset or carry-trust-from", ErrConfig)
	case cfg.CarryTrustFrom == "" && matched && found && cfg.AcceptTrustReset:
		return decisionledger.Successor{}, fmt.Errorf("%w: accept-trust-reset: the replaced ledger has a carryable trust record, which is carried", ErrConfig)
	case cfg.CarryTrustFrom == "" && !(matched && found):
		out.TrustReset = true
		return out, nil
	}
	t := *rec.Decision.Trust
	t.OldEpoch, t.NewEpoch = t.NewEpoch, next<<32+1
	t.InvalidateBelow, t.Origin, t.CarriedFrom, t.CarriedLine = t.NewEpoch, decisionledger.OriginCarried, p.LedgerID+":"+strconv.FormatUint(rec.Sequence, 10), line
	out.CarriedFrom = p.LedgerID
	out.Carry = []decisionledger.Decision{{Workspace: s.bootWorkspace, Decision: decisionledger.TrustActivation, Trust: &t}}
	return out, nil
}

func earlierView(rec decisionledger.Record) trustchange.View {
	t := rec.Decision.Trust
	return trustchange.View{ID: t.Change, Workspace: rec.Decision.Workspace, Route: trustchange.RouteLocalPassword, Revision: t.ChangeRevision, OldEpoch: t.OldEpoch,
		OldPolicy: t.OldPolicy, OldDigest: t.OldDigest, NewDigest: t.NewDigest, ChangeDigest: t.ChangeDigest, Candidate: candidateFrom(t.Config),
		State: trustchange.StateActivated, Proposer: t.ProposedBy, Validator: t.ValidatedBy, Approver: t.ApprovedBy, Activator: t.ActivatedBy, ApprovedAt: t.ApprovedAt,
		Validation: &trustchange.Validation{Result: trustchange.ResultValidated, Digest: t.EvidenceDigest},
		Activation: &trustchange.Activation{Epoch: t.NewEpoch, Sequence: rec.Sequence, Watermark: rec.Watermark, At: rec.At}, RunStartedAt: t.RunStartedAt,
		History: []trustchange.Entry{{State: trustchange.StateActivated, By: t.ActivatedBy, At: rec.At}}}
}

func (s *Server) trustRecords() []decisionledger.Record {
	if s.ledger == nil {
		return nil
	}
	return s.ledger.Verified()
}

func (s *Server) replayTrust() (bool, error) {
	records := s.trustRecords()
	taken := map[string]bool{}
	views := []trustchange.View{}
	for _, rec := range records {
		t := rec.Decision.Trust
		if t == nil {
			continue
		}
		if run, ok := trustchange.RunOf(t.Change); ok {
			taken[run] = true
			if rec.Decision.Workspace == s.bootWorkspace && t.Origin == decisionledger.OriginActivation {
				views = append(views, earlierView(rec))
			}
		}
	}
	run, err := trustchange.Draw(s.random, taken)
	if err != nil {
		return false, err
	}
	s.trust, err = trustchange.New(run, s.clock)
	if err != nil {
		return false, err
	}
	st := decisionledger.Status{}
	if s.ledger != nil {
		st = s.ledger.Status()
	}
	rec, ok := latestTrust(records, s.bootWorkspace)
	older := ok && st.Isolation == decisionledger.IsolationMismatch && st.MarkerUsable && rec.Decision.Trust.NewEpoch>>32 < st.Generation
	if !older {
		if err := s.trust.Restore(views); err != nil {
			return false, fmt.Errorf("restore the trust changes of earlier runs: %w", err)
		}
	}
	if ok && !older {
		t := rec.Decision.Trust
		origin := trustOrigin{origin: originReplay, unconfirmed: st.NotDurable, activatedAt: rec.At, change: t.Change}
		if t.Origin == decisionledger.OriginCarried {
			origin.origin, origin.carriedFrom, origin.carriedLine = originCarried, t.CarriedFrom, t.CarriedLine
		}
		return true, s.publishTrust(authpolicy.Publication{Revision: revisionFor(s.bootWorkspace, candidateFrom(t.Config)), Epoch: t.NewEpoch, Number: t.NewPolicy,
			Proposer: t.ProposedBy, Validator: t.ValidatedBy, Approver: t.ApprovedBy, Activator: t.ActivatedBy}, origin)
	}
	origin := trustOrigin{origin: originSeed, olderIgnored: older}
	if st.Genesis.TrustReset {
		origin.resetAt = st.CreatedAt
	}
	if s.cfg.Seed != SeedSample && !older {
		s.trustOrigin = origin
		return false, nil
	}
	seed := trustchange.Candidate{Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFANone, MaxAge: seedPolicyAge}
	switch {
	case s.cfg.Seed != SeedSample:
		seed, origin.stricter = stricter(trustchange.Candidate{MFA: authpolicy.MFANone, MaxAge: math.MaxInt64}, candidateFrom(rec.Decision.Trust.Config))
	case older:
		seed, origin.stricter = stricter(seed, candidateFrom(rec.Decision.Trust.Config))
	}
	return true, s.publishTrust(authpolicy.Publication{Revision: revisionFor(s.bootWorkspace, seed), Epoch: s.generation()<<32 + 1, Number: 1}, origin)
}

func (s *Server) trustDecision(v trustchange.View, epoch, policy uint64, affected int) decisionledger.Decision {
	return decisionledger.Decision{Workspace: v.Workspace, Decision: decisionledger.TrustActivation, Trust: &decisionledger.Trust{Change: v.ID, ChangeRevision: v.Revision,
		ChangeDigest: v.ChangeDigest, EvidenceDigest: v.Validation.Digest, OldEpoch: v.OldEpoch, NewEpoch: epoch, OldPolicy: v.OldPolicy, NewPolicy: policy,
		OldDigest: v.OldDigest, NewDigest: v.NewDigest, Config: configOf(v.Candidate), ProposedBy: v.Proposer, ValidatedBy: v.Validator, ApprovedBy: v.Approver,
		ActivatedBy: v.Activator, ApprovedAt: v.ApprovedAt, RunStartedAt: s.startedAt, InvalidateBelow: epoch, AffectedSessions: uint64(affected),
		Origin: decisionledger.OriginActivation}}
}

func stricter(base, other trustchange.Candidate) (trustchange.Candidate, []string) {
	out := trustchange.Candidate{MFA: base.MFA, MaxAge: min(base.MaxAge, other.MaxAge)}
	fields := []string{}
	for _, m := range base.Methods {
		if slices.Contains(other.Methods, m) {
			out.Methods = append(out.Methods, m)
		}
	}
	if len(out.Methods) == 0 {
		out.Methods = append([]string(nil), other.Methods...)
	}
	if !slices.Equal(out.Methods, base.Methods) {
		fields = append(fields, "methods")
	}
	if mfaRank(other.MFA) > mfaRank(base.MFA) {
		out.MFA = other.MFA
		fields = append(fields, "mfa")
	}
	if out.MaxAge != base.MaxAge {
		fields = append(fields, "max_age")
	}
	rules := map[string]trustchange.Rule{}
	for _, r := range append(append([]trustchange.Rule(nil), base.StepUp...), other.StepUp...) {
		if have, ok := rules[r.Action]; ok {
			r.MaxAge, r.MFA = min(r.MaxAge, have.MaxAge), r.MFA || have.MFA
		}
		r.MaxAge = min(r.MaxAge, out.MaxAge)
		rules[r.Action] = r
	}
	for _, action := range slices.Sorted(maps.Keys(rules)) {
		out.StepUp = append(out.StepUp, rules[action])
	}
	if !slices.Equal(out.StepUp, base.StepUp) {
		fields = append(fields, "step_up")
	}
	return out, fields
}
