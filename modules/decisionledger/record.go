package decisionledger

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

var genesisPrev = strings.Repeat("0", 64)

type line struct {
	V                int        `json:"v"`
	Seq              uint64     `json:"seq"`
	Prev             string     `json:"prev"`
	At               string     `json:"at"`
	Kind             string     `json:"kind"`
	LedgerID         string     `json:"ledger_id,omitempty"`
	CreatedAt        string     `json:"created_at,omitempty"`
	Replaces         string     `json:"replaces,omitempty"`
	ReplaceReason    string     `json:"replace_reason,omitempty"`
	TruncatedBytes   int64      `json:"truncated_bytes,omitempty"`
	RecoveredAt      string     `json:"recovered_at,omitempty"`
	G                uint64     `json:"g,omitempty"`
	TrustReset       bool       `json:"trust_reset,omitempty"`
	GenerationStated bool       `json:"generation_stated,omitempty"`
	CarriedLedger    string     `json:"carried_from_ledger,omitempty"`
	ReplacedLine     int        `json:"replaced_line,omitempty"`
	Workspace        string     `json:"workspace,omitempty"`
	Decision         string     `json:"decision,omitempty"`
	Target           string     `json:"target,omitempty"`
	TargetVersion    uint64     `json:"target_version,omitempty"`
	Revocation       string     `json:"revocation,omitempty"`
	Request          string     `json:"request,omitempty"`
	RequestDigest    string     `json:"request_digest,omitempty"`
	RequestedBy      string     `json:"requested_by,omitempty"`
	ApprovedBy       string     `json:"approved_by,omitempty"`
	DenyGeneration   uint64     `json:"deny_generation,omitempty"`
	EndedDelegations []string   `json:"ended_delegations,omitempty"`
	Trust            *trustLine `json:"trust,omitempty"`
	Sum              string     `json:"sum"`
}

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func encode(l line) []byte {
	l.Sum = ""
	bare, _ := json.Marshal(l)
	l.Sum = hashOf(bare)
	out, _ := json.Marshal(l)
	return out
}

func hexID(s string, size int) bool {
	if len(s) != size {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}

func validTime(s string) bool {
	at, err := time.Parse(time.RFC3339Nano, s)
	return err == nil && stamp(at) == s
}

func (l line) shaped() bool {
	switch l.Kind {
	case KindGenesis:
		return hexID(l.LedgerID, 32) && validTime(l.CreatedAt) && (l.Replaces == "") == (l.ReplaceReason == "") && l.decisionEmpty() && l.TruncatedBytes == 0 &&
			l.RecoveredAt == "" && (l.Replaces != "" || !l.annotated()) && l.ReplacedLine >= 0 && l.G < MaxGeneration
	case KindRecovery:
		return l.TruncatedBytes > 0 && validTime(l.RecoveredAt) && l.genesisEmpty() && l.decisionEmpty()
	case KindDecision:
		return l.Workspace != "" && l.genesisEmpty() && l.TruncatedBytes == 0 && l.RecoveredAt == "" && (l.revocationShaped() || l.trustShaped())
	}
	return false
}

func (l line) annotated() bool {
	return l.TrustReset || l.GenerationStated || l.CarriedLedger != "" || l.ReplacedLine != 0
}

func (l line) revocationShaped() bool {
	return (l.Decision == RevokeGrant || l.Decision == RevokeDelegation) && l.Target != "" && l.Revocation != "" && l.RequestedBy != "" && l.Trust == nil
}

func (l line) trustShaped() bool {
	return l.Decision == TrustActivation && l.Target == "" && l.TargetVersion == 0 && l.Revocation == "" && l.Request == "" && l.RequestDigest == "" &&
		l.RequestedBy == "" && l.ApprovedBy == "" && l.DenyGeneration == 0 && len(l.EndedDelegations) == 0 && l.Trust != nil && l.Trust.shaped()
}

func (l line) genesisEmpty() bool {
	return l.LedgerID == "" && l.CreatedAt == "" && l.Replaces == "" && l.ReplaceReason == "" && l.G == 0 && !l.annotated()
}

func (l line) decisionEmpty() bool {
	return l.Workspace == "" && l.Decision == "" && l.Target == "" && l.TargetVersion == 0 && l.Revocation == "" && l.Request == "" && l.RequestDigest == "" &&
		l.RequestedBy == "" && l.ApprovedBy == "" && l.DenyGeneration == 0 && len(l.EndedDelegations) == 0 && l.Trust == nil
}

func decodeLine(data []byte) (line, bool) {
	var l line
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&l) != nil {
		return line{}, false
	}
	return l, bytes.Equal(encode(l), data)
}

type scan struct {
	records   uint64
	size      int64
	torn      int64
	last      string
	problem   int
	order     bool
	ledgerID  string
	createdAt time.Time
	replaces  string
	reason    string
	g         uint64
	genesis   Successor
	epochs    map[string]uint64
	changes   map[string]bool
	highest   uint64
	decisions []Record
}

func (s *scan) accept(l line, data []byte) {
	at, _ := time.Parse(time.RFC3339Nano, l.At)
	s.last = hashOf(data)
	switch l.Kind {
	case KindGenesis:
		s.ledgerID, s.replaces, s.reason, s.g = l.LedgerID, l.Replaces, l.ReplaceReason, l.G
		s.genesis = Successor{Generation: l.G, TrustReset: l.TrustReset, GenerationStated: l.GenerationStated, CarriedFrom: l.CarriedLedger, ReplacedLine: l.ReplacedLine}
		s.createdAt, _ = time.Parse(time.RFC3339Nano, l.CreatedAt)
	case KindDecision:
		d := Decision{Workspace: l.Workspace, Decision: l.Decision, Target: l.Target, TargetVersion: l.TargetVersion, Revocation: l.Revocation, Request: l.Request,
			RequestDigest: l.RequestDigest, RequestedBy: l.RequestedBy, ApprovedBy: l.ApprovedBy, DenyGeneration: l.DenyGeneration, EndedDelegations: l.EndedDelegations}
		if l.Trust != nil {
			d.Trust = l.Trust.trust()
			s.epochs[l.Workspace] = l.Trust.NewEpoch
			s.changes[l.Workspace+"/"+l.Trust.Change] = true
			s.highest = max(s.highest, l.Trust.NewEpoch)
		}
		s.decisions = append(s.decisions, Record{Sequence: l.Seq, Watermark: strconv.FormatUint(l.Seq, 10) + ":" + s.last, At: at, Decision: d})
	}
	s.records++
	s.size += int64(len(data)) + 1
}

func (s *scan) next(data []byte) (line, bool) {
	l, ok := decodeLine(data)
	return l, ok && l.V == FormatV1 && l.Seq == s.records+1 && l.Prev == s.last && (l.Kind == KindGenesis) == (l.Seq == 1) && validTime(l.At) && l.shaped()
}

func (s *scan) ordered(l line) bool {
	if l.Trust == nil {
		return true
	}
	ok := l.Trust.NewEpoch > s.epochs[l.Workspace] && l.Trust.NewEpoch>>32 == s.g && !s.changes[l.Workspace+"/"+l.Trust.Change]
	s.order = !ok
	return ok
}

func (s *scan) confirmedTail(data []byte) (line, []byte, bool) {
	if l, ok := s.next(data); ok {
		return l, data, true
	}
	trimmed := data[:len(data)-1]
	l, ok := s.next(trimmed)
	return l, trimmed, ok
}

func verify(r io.ReaderAt, size int64) scan {
	s := scan{last: genesisPrev, epochs: map[string]uint64{}, changes: map[string]bool{}}
	reader := bufio.NewReaderSize(io.NewSectionReader(r, 0, size), LineLimit+1)
	for {
		data, err := reader.ReadSlice('\n')
		number := int(s.records) + 1
		if errors.Is(err, io.EOF) {
			if len(data) > 0 {
				if l, kept, ok := s.confirmedTail(data); ok {
					if s.ordered(l) {
						s.accept(l, kept)
					}
					s.problem = number
					return s
				}
			}
			s.torn = int64(len(data))
			return s
		}
		if err != nil {
			s.problem = number
			return s
		}
		data = data[:len(data)-1]
		l, ok := s.next(data)
		if !ok || !s.ordered(l) {
			s.problem = number
			return s
		}
		s.accept(l, data)
	}
}
