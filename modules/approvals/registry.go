package approvals

import (
	"sort"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type authority struct {
	view  AuthorityView
	limit exact.Fraction
}

type entry struct {
	e      Entry
	amount exact.Fraction
}

type request struct {
	view      RequestView
	amount    exact.Fraction
	entries   map[string]*entry
	excluded  map[string]string
	approvers []string
	deciders  map[string]int
	executed  map[string]string
}

type execution struct {
	view ExecutionView
	req  *request
}

type space struct {
	admins      map[string]bool
	proposals   map[string]*ProposalView
	policies    map[string]Policy
	authorities map[string]*authority
	requests    map[string]*request
	targets     map[[2]string][]*request
	executions  map[string]*execution
	keys        map[[2]string]string
}

type Registry struct {
	mu         sync.Mutex
	profile    decimal.Profile
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(profile decimal.Profile, limits Limits, clock func() time.Time) (*Registry, error) {
	if !profile.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{profile: profile, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func (r *Registry) number(text string, signed bool) (exact.Fraction, string, bool) {
	if len(text) > maxNumberBytes {
		return exact.Fraction{}, "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || !signed && value.Sign() < 0 {
		return exact.Fraction{}, "", false
	}
	made, ok := exact.Of(value)
	return made, value.String(), ok
}

func absolute(f exact.Fraction) exact.Fraction {
	if f.Sign() < 0 {
		return exact.Whole(0).Minus(f)
	}
	return f
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }

func refuse(reason string) error { return Refusal{Reason: reason} }

func validActor(a Actor) bool { return validID(a.ID) && kinds[a.Kind] }

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sameDigests(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func copyDigests(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *space) active(id string, now time.Time) *authority {
	a := s.authorities[id]
	if a == nil || !a.view.Revoked.IsZero() {
		return nil
	}
	today := now.Format("2006-01-02")
	if today < a.view.Authority.From || a.view.Authority.To != "" && today > a.view.Authority.To {
		return nil
	}
	return a
}

func (s *space) covering(approver, stage, currency string, amount exact.Fraction, now time.Time) bool {
	for id, a := range s.authorities {
		k := a.view.Authority
		if k.Approver == approver && k.Stage == stage && k.Currency == currency && s.active(id, now) != nil && !a.limit.Less(absolute(amount)) {
			return true
		}
	}
	return false
}
