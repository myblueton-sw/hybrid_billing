package components

import (
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type component struct {
	view     ComponentView
	fee      exact.Fraction
	rate     exact.Fraction
	included map[string]exact.Fraction
	all      map[string]bool
}

type clause struct {
	view ClauseView
	rate exact.Fraction
	comp *component
}

type incident struct {
	view IncidentView
}

type credit struct {
	view   CreditView
	clause *clause
	amount exact.Fraction
}

type space struct {
	pricing     map[string]*PricingView
	components  map[string]*component
	contracts   map[string][]*component
	incidents   map[string]*incident
	counts      map[string]int
	clauses     map[string]*clause
	byContract  map[string][]*clause
	credits     map[string]*credit
	perContract map[string]int
	documents   map[string]string
	increases   map[string]string
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

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{pricing: map[string]*PricingView{}, components: map[string]*component{}, contracts: map[string][]*component{}, incidents: map[string]*incident{},
		counts: map[string]int{}, clauses: map[string]*clause{}, byContract: map[string][]*clause{}, credits: map[string]*credit{}, perContract: map[string]int{},
		documents: map[string]string{}, increases: map[string]string{}}
	r.workspaces[workspace] = s
	return s, nil
}

func (r *Registry) number(text string, positive bool) (exact.Fraction, string, bool) {
	if len(text) > maxNumberBytes {
		return exact.Fraction{}, "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || value.Sign() < 0 || positive && value.Sign() == 0 {
		return exact.Fraction{}, "", false
	}
	made, ok := exact.Of(value)
	return made, value.String(), ok
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }

func round(f exact.Fraction, p Pricing) exact.Fraction {
	units, _ := f.Quanta(p.Scale, p.Mode)
	out, _ := exact.Units(units, p.Scale)
	return out
}

func fits(f exact.Fraction, scale int) bool {
	units, _ := f.Quanta(scale, "toward_zero")
	back, _ := exact.Units(units, scale)
	return same(back, f)
}

const dateLayout = "2006-01-02"

func month(start, end string) bool {
	if !validDate(start) || !validDate(end) {
		return false
	}
	first, _ := time.Parse(dateLayout, start)
	return first.Day() == 1 && first.AddDate(0, 1, -1).Format(dateLayout) == end
}

func endOf(to string) string {
	if to == "" {
		return "9999-12-31"
	}
	return to
}

func overlap(aFrom, aTo, bFrom, bTo string) bool {
	return aFrom <= endOf(bTo) && bFrom <= endOf(aTo)
}

func refuse(reason string) error { return Refusal{Reason: reason} }
