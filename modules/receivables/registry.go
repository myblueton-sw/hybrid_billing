package receivables

import (
	"strconv"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type term struct {
	view TermView
}

type binding struct {
	original Binding
	view     BindingView
	location *time.Location
	issued   []*document
}

type document struct {
	view        DocumentView
	binding     *binding
	amount      exact.Fraction
	location    *time.Location
	extensions  []*extension
	allocations []*allocation
	reversals   []*reversal
	holds       []*hold
}

type extension struct {
	view ExtensionView
	doc  *document
}

type allocation struct {
	view     AllocationView
	doc      *document
	amount   exact.Fraction
	reversal *reversal
}

type reversal struct {
	view  ReversalView
	alloc *allocation
}

type hold struct {
	view HoldView
	doc  *document
}

type space struct {
	terms         map[[2]string]*term
	bindings      map[string]*binding
	contracts     map[string][]*binding
	documents     map[string]*document
	extensions    map[string]*extension
	allocations   map[string]*allocation
	reversals     map[string]*reversal
	holds         map[string]*hold
	confirmations map[[2]string]Confirmation
}

type Registry struct {
	mu         sync.Mutex
	profile    decimal.Profile
	limits     Limits
	clock      func() time.Time
	zones      func(string) (*time.Location, error)
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(profile decimal.Profile, limits Limits, clock func() time.Time, zones func(string) (*time.Location, error)) (*Registry, error) {
	if !profile.Valid() || !limits.valid() || clock == nil || zones == nil {
		return nil, ErrInvalid
	}
	return &Registry{profile: profile, limits: limits, clock: clock, zones: zones, workspaces: map[string]*space{}}, nil
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

func (r *Registry) space(workspace string) *space {
	s := r.workspaces[workspace]
	if s == nil {
		s = &space{terms: map[[2]string]*term{}, bindings: map[string]*binding{}, contracts: map[string][]*binding{}, documents: map[string]*document{},
			extensions: map[string]*extension{}, allocations: map[string]*allocation{}, reversals: map[string]*reversal{}, holds: map[string]*hold{},
			confirmations: map[[2]string]Confirmation{}}
		r.workspaces[workspace] = s
	}
	return s
}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	return r.space(workspace), nil
}

func (r *Registry) amount(text string) (exact.Fraction, string, bool) {
	if len(text) > maxNumberBytes {
		return exact.Fraction{}, "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || value.Sign() <= 0 {
		return exact.Fraction{}, "", false
	}
	made, ok := exact.Of(value)
	return made, value.String(), ok
}

func refuse(reason string) error { return Refusal{Reason: reason} }

func dateOf(t time.Time, loc *time.Location) string { return t.In(loc).Format(dateLayout) }

const dateLayout = "2006-01-02"

func civil(date string) time.Time {
	parsed, _ := time.Parse(dateLayout, date)
	return parsed
}

func addDays(date string, days int) string { return civil(date).AddDate(0, 0, days).Format(dateLayout) }

func daysBetween(from, to string) int { return int(civil(to).Sub(civil(from)) / (24 * time.Hour)) }

func zoneForm(name string) bool { return name != "" && name != "Local" && len(name) <= maxZoneBytes }

func itoa(n int) string { return strconv.Itoa(n) }
