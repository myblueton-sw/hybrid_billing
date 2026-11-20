package pricingpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func validID(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._:-", c)) {
			return false
		}
	}
	return true
}
func validDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}
func validReason(s string) bool {
	if !utf8.ValidString(s) || len(s) > 1024 || strings.TrimSpace(s) == "" {
		return false
	}
	for _, c := range s {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func instant(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 && offset == 0
}
func interval(from, to time.Time) bool { return instant(from) && instant(to) && from.Before(to) }
func validScope(s Scope) bool {
	return validID(s.Workspace) && validID(s.Customer) && validID(s.Entity) && validID(s.Contract) && validID(s.BillingProfile)
}
func validTarget(t Target) bool {
	return t.ContractRevision > 0 && t.BillingRevision > 0 && len(t.Currency) == 3 && t.Currency[0] >= 'A' && t.Currency[0] <= 'Z' && t.Currency[1] >= 'A' && t.Currency[1] <= 'Z' && t.Currency[2] >= 'A' && t.Currency[2] <= 'Z' && (t.Path == rating.Direct || t.Path == rating.Resale) && validDigest(t.Fingerprint)
}
func bounded(v reflect.Value, left *int) bool {
	switch v.Kind() {
	case reflect.String:
		if v.Len() > *left || !utf8.ValidString(v.String()) {
			return false
		}
		*left -= v.Len()
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return true
		}
		for i := 0; i < v.NumField(); i++ {
			if !bounded(v.Field(i), left) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if !bounded(v.Index(i), left) {
				return false
			}
		}
	}
	return true
}
func form(s Spec) bool {
	if s.Mode != "synthetic" || !validScope(s.Scope) || !interval(s.From, s.To) || !validID(s.Profile.ID) || !validID(s.Profile.Revision) || len(s.Policy.Rules) < 1 || len(s.Policy.Rules) > 1000 || len(s.Policy.Eligible) > 4 {
		return false
	}
	for _, n := range []int{s.Profile.InputIntegerDigits, s.Profile.InputScale, s.Profile.ResultIntegerDigits, s.Profile.ResultScale, s.Profile.IntermediateDigits, s.Limits.Intermediate} {
		if n < 1 || n > 80 {
			return false
		}
	}
	for _, n := range []int{s.Limits.Rules, s.Limits.Tiers, s.Limits.Discounts, s.Limits.Lines} {
		if n < 1 || n > 1000 {
			return false
		}
	}
	tiers, discounts := 0, 0
	for _, r := range s.Policy.Rules {
		tiers += len(r.Tiers)
		discounts += len(r.Discounts)
		if tiers > 1000 || discounts > 1000 {
			return false
		}
	}
	left := MaxBytes
	return bounded(reflect.ValueOf(s), &left)
}
func digest(p Proposal) (string, error) {
	data, err := json.Marshal(p)
	if err != nil || len(data) > MaxBytes {
		return "", ErrInvalid
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
func compile(s Spec) error {
	profile, err := decimal.NewProfile(s.Profile)
	if err != nil {
		return ErrInvalid
	}
	if _, err = rating.Compile(profile, s.Limits, s.Policy); err != nil {
		return ErrInvalid
	}
	return nil
}

func ValidateSpec(s Spec) error {
	if !form(s) {
		return ErrInvalid
	}
	return compile(s)
}
func clone(in Record) Record {
	out := in
	out.Proposal.Spec.From = in.Proposal.Spec.From.UTC()
	out.Proposal.Spec.To = in.Proposal.Spec.To.UTC()
	policy := &out.Proposal.Spec.Policy
	policy.Eligible = append([]rating.Class(nil), policy.Eligible...)
	policy.Rules = append([]rating.Rule(nil), policy.Rules...)
	for i := range policy.Rules {
		policy.Rules[i].Tiers = append([]rating.Tier(nil), policy.Rules[i].Tiers...)
		policy.Rules[i].Discounts = append([]rating.Discount(nil), policy.Rules[i].Discounts...)
	}
	if in.Approval != nil {
		d := *in.Approval
		out.Approval = &d
	}
	if in.Revocation != nil {
		d := *in.Revocation
		out.Revocation = &d
	}
	return out
}
