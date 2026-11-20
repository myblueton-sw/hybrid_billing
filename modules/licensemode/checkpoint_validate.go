package licensemode

import (
	"crypto/ed25519"
	"encoding/hex"
	"reflect"
	"time"
)

func checkpointTime(t time.Time) bool { return t.IsZero() || validTime(t.Unix()) }
func checkpointMode(s string) bool {
	return s == "" || oneOf(s, ModeNoncommercial, ModeEvaluation, ModeCommercial)
}
func checkpointStatus(s string) bool {
	return s == "" || oneOf(s, StateNoncommercial, StateEvaluationPending, StateEvaluationActive, StateEvaluationExpired, StateCommercialPending, StateCommercialActive, StateCommercialExpired, StateUnverified, StateUnselected)
}
func checkpointOutcome(s string) bool {
	return oneOf(s, OutcomeInvalid, OutcomeUnverifiable, OutcomeWrongKind, OutcomeScopeMismatch, OutcomeTimeUnavailable, OutcomeClockAnomaly, ReasonLocalClockBehind, OutcomeNotYetValid, OutcomeExpired, OutcomeSuperseded, OutcomeDuplicateRight, OutcomeBadSupersedes, OutcomeAccepted, OutcomeRepeat)
}
func checkpointScope(scope []string) bool {
	if len(scope) < 1 || len(scope) > 32 {
		return false
	}
	for i, s := range scope {
		if !validID(s) || i > 0 && scope[i-1] >= s {
			return false
		}
	}
	return true
}
func validateCheckpoint(c checkpoint) bool {
	if !validID(c.Installation) || !validID(c.Product) || !checkpointTime(c.Now) || c.Now.IsZero() || !checkpointTime(c.High) {
		return false
	}
	if c.Keys == nil || c.Grants == nil || c.Documents == nil || c.Proposals == nil || len(c.Grants) > 1 || len(c.Documents) > maxRecords || len(c.Pending) > maxPending || len(c.Done) > maxDone || len(c.Proposals) != len(c.Pending)+len(c.Done) {
		return false
	}
	active := 0
	for id, k := range c.Keys {
		if id != k.Key.ID || !k.Key.valid() {
			return false
		}
		public, _ := hex.DecodeString(k.Key.Public)
		proof, _ := hex.DecodeString(k.Key.Proof)
		if !ed25519.Verify(public, proofMessage(id, k.Key.Issuer), proof) {
			return false
		}
		if !k.Compromised && !k.Retired {
			active++
		}
	}
	if active > maxKeys || !validateCheckpointProposals(c) {
		return false
	}
	s := c.State
	if !checkpointMode(s.Mode) || !checkpointStatus(s.Last) || !checkpointTime(s.LastAt) || (s.Last == "") != s.LastAt.IsZero() || s.LastAt.After(c.Now.Add(maxAhead)) || len(s.Modes) > maxHistory || len(s.Submissions) > maxHistory {
		return false
	}
	if len(s.Modes) == 0 && s.Mode != "" || len(s.Modes) > 0 && s.Modes[len(s.Modes)-1].Mode != s.Mode {
		return false
	}
	for i, m := range s.Modes {
		if m.Mode == "" || !checkpointMode(m.Mode) || !validID(m.By) || m.At.IsZero() || !checkpointTime(m.At) || m.At.After(c.Now) || i > 0 && m.At.Before(s.Modes[i-1].At) {
			return false
		}
	}
	for i, v := range s.Submissions {
		if !checkpointOutcome(v.Outcome) || len(v.Digest) != 64 || !validHexBytes(v.Digest) || v.At.IsZero() || !checkpointTime(v.At) || v.At.After(c.Now) || i > 0 && v.At.Before(s.Submissions[i-1].At) {
			return false
		}
	}
	if (s.Grant == "") != (len(c.Grants) == 0) {
		return false
	}
	for id, g := range c.Grants {
		a := g.View
		if id != s.Grant || id != a.Grant || !validID(id) || !validID(a.Subject) || !validText(a.Terms) || !validID(a.AcceptedBy) || !checkpointScope(a.Scope) || a.Start.IsZero() || !checkpointTime(a.Start) || a.Start.Nanosecond() != 0 || !a.End.Equal(a.Start.Add(evaluationLength)) || !checkpointTime(a.End) {
			return false
		}
		if _, ok := c.Keys[a.KeyID]; !ok {
			return false
		}
	}
	if len(s.Documents) != len(c.Documents) || c.Accepted != uint64(len(c.Documents)) {
		return false
	}
	seen := map[string]bool{}
	for index, id := range s.Documents {
		d, ok := c.Documents[id]
		if !ok || seen[id] || d.Order != uint64(index+1) || d.Claims.ID != id {
			return false
		}
		seen[id] = true
		claims, ok := Decode(Encode(d.Claims))
		if !ok || !reflect.DeepEqual(claims, d.Claims) || claims.Kind != DocCommercial || claims.Installation != c.Installation || claims.Product != c.Product {
			return false
		}
		key, ok := c.Keys[claims.KeyID]
		if !ok || key.Key.Issuer != claims.Issuer {
			return false
		}
	}
	return true
}
func validateCheckpointProposals(c checkpoint) bool {
	seen := map[uint64]bool{}
	mine := map[string]int{}
	check := func(number uint64, state string) bool {
		p, ok := c.Proposals[number]
		if !ok || seen[number] || number == 0 || number > c.Counter || p.Number != number || p.State != state || !validID(p.Proposer) {
			return false
		}
		seen[number] = true
		clean, err := clean(p.Change)
		if err != nil || clean != p.Change {
			return false
		}
		if p.Change.Kind == ChangeKey {
			k := p.Change.Key
			public, _ := hex.DecodeString(k.Public)
			proof, _ := hex.DecodeString(k.Proof)
			if !ed25519.Verify(public, proofMessage(k.ID, k.Issuer), proof) {
				return false
			}
		}
		if state == ProposalPending {
			if p.Approver != "" {
				return false
			}
			mine[p.Proposer]++
			if mine[p.Proposer] > maxMine {
				return false
			}
		} else if !validID(p.Approver) || p.Proposer == p.Approver {
			return false
		}
		return true
	}
	for i, n := range c.Pending {
		if i > 0 && c.Pending[i-1] >= n || !check(n, ProposalPending) {
			return false
		}
	}
	for _, n := range c.Done {
		if !check(n, ProposalApplied) {
			return false
		}
	}
	return true
}
