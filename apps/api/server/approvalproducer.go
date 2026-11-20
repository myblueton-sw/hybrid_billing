package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const candidateBytes = 1 << 20

var (
	errCandidateInvalid  = errors.New("invalid charge candidate")
	errCandidateConflict = errors.New("charge candidate identity conflict")
	errCandidateLimit    = errors.New("charge candidate limit exceeded")
	errCandidateMissing  = errors.New("charge candidate not found")
)

type chargeSource struct {
	ID       string
	Revision string
	Digest   string
}

type chargeInput struct {
	SourceMode string
	Normalized normalizedOrigin
	Workspace  string
	ID         string
	Customer   string
	Entity     string
	From       string
	To         string
	Sources    []chargeSource
	Profile    decimal.ProfileSpec
	Limits     rating.Limits
	Policy     rating.Policy
	Lines      []rating.Line
}

type chargeEnvelope struct {
	Schema string
	Input  chargeInput
	Result rating.Result
}

type chargeCandidate struct {
	Envelope chargeEnvelope
	Digest   string
}

type chargeRecord struct {
	input    []byte
	envelope []byte
	digest   string
}

type chargeCandidates struct {
	mu      sync.Mutex
	records map[[2]string]chargeRecord
}

func candidateText(v reflect.Value, remaining *int) bool {
	switch v.Kind() {
	case reflect.String:
		if v.Len() > *remaining || !utf8.ValidString(v.String()) {
			return false
		}
		*remaining -= v.Len()
		return true
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !candidateText(v.Field(i), remaining) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if !candidateText(v.Index(i), remaining) {
				return false
			}
		}
	}
	return true
}

func candidateForm(in chargeInput) bool {
	if in.SourceMode != "synthetic" {
		return false
	}
	for _, id := range []string{in.Workspace, in.ID, in.Customer, in.Entity, in.Profile.ID, in.Profile.Revision} {
		if !validRequestID(id) {
			return false
		}
	}
	from, e1 := time.Parse(time.DateOnly, in.From)
	to, e2 := time.Parse(time.DateOnly, in.To)
	if e1 != nil || e2 != nil || from.Format(time.DateOnly) != in.From || to.Format(time.DateOnly) != in.To || from.After(to) {
		return false
	}
	if len(in.Sources) < 1 || len(in.Sources) > 1000 || len(in.Lines) < 1 || len(in.Lines) > 1000 || len(in.Policy.Rules) > 1000 || len(in.Policy.Eligible) > 4 {
		return false
	}
	p := in.Profile
	for _, n := range []int{p.InputIntegerDigits, p.InputScale, p.ResultIntegerDigits, p.ResultScale, p.IntermediateDigits, in.Limits.Intermediate} {
		if n < 1 || n > 80 {
			return false
		}
	}
	for _, n := range []int{in.Limits.Rules, in.Limits.Tiers, in.Limits.Discounts, in.Limits.Lines} {
		if n < 1 || n > 1000 {
			return false
		}
	}
	for _, rule := range in.Policy.Rules {
		if len(rule.Tiers) > 1000 || len(rule.Discounts) > 1000 {
			return false
		}
	}
	seen := make(map[string]bool)
	for _, source := range in.Sources {
		if !validRequestID(source.ID) || !validRequestID(source.Revision) || seen[source.ID] || len(source.Digest) != 64 {
			return false
		}
		for _, c := range source.Digest {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return false
			}
		}
		seen[source.ID] = true
	}
	remaining := candidateBytes
	return candidateText(reflect.ValueOf(in), &remaining)
}

func decodeCandidate(record chargeRecord) (chargeCandidate, error) {
	var out chargeCandidate
	if err := json.Unmarshal(record.envelope, &out.Envelope); err != nil {
		return chargeCandidate{}, err
	}
	out.Digest = record.digest
	return out, nil
}

func (s *chargeCandidates) capture(in chargeInput) (chargeCandidate, error) {
	if !candidateForm(in) {
		return chargeCandidate{}, errCandidateInvalid
	}
	encoded, err := json.Marshal(in)
	if err != nil {
		return chargeCandidate{}, err
	}
	if len(encoded) > candidateBytes {
		return chargeCandidate{}, errCandidateLimit
	}
	var copied chargeInput
	if err := json.Unmarshal(encoded, &copied); err != nil {
		return chargeCandidate{}, err
	}
	key := [2]string{copied.Workspace, copied.ID}
	s.mu.Lock()
	defer s.mu.Unlock()
	if prior, found := s.records[key]; found {
		if !bytes.Equal(prior.input, encoded) {
			return chargeCandidate{}, errCandidateConflict
		}
		return decodeCandidate(prior)
	}
	if len(s.records) >= 1000 {
		return chargeCandidate{}, errCandidateLimit
	}
	profile, err := decimal.NewProfile(copied.Profile)
	if err != nil {
		return chargeCandidate{}, err
	}
	compiled, err := rating.Compile(profile, copied.Limits, copied.Policy)
	if err != nil {
		return chargeCandidate{}, err
	}
	result, err := compiled.Rate(copied.Lines)
	if err != nil {
		return chargeCandidate{}, err
	}
	envelope, err := json.Marshal(chargeEnvelope{Schema: "hb_charge_candidate_v2", Input: copied, Result: result})
	if err != nil {
		return chargeCandidate{}, err
	}
	if len(envelope) > candidateBytes {
		return chargeCandidate{}, errCandidateLimit
	}
	hash := sha256.Sum256(envelope)
	record := chargeRecord{input: encoded, envelope: envelope, digest: hex.EncodeToString(hash[:])}
	out, err := decodeCandidate(record)
	if err != nil {
		return chargeCandidate{}, err
	}
	if s.records == nil {
		s.records = make(map[[2]string]chargeRecord)
	}
	s.records[key] = record
	return out, nil
}

func (s *chargeCandidates) read(workspace, id string) (chargeCandidate, error) {
	if !validRequestID(workspace) || !validRequestID(id) {
		return chargeCandidate{}, errCandidateInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, found := s.records[[2]string{workspace, id}]
	if !found {
		return chargeCandidate{}, errCandidateMissing
	}
	return decodeCandidate(record)
}
