package adjustments

import (
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type line struct {
	source   SourceLine
	quantity exact.Fraction
	price    exact.Fraction
	priced   bool
	amount   exact.Fraction
}

type entry struct {
	view   EntryView
	at     int
	value  exact.Fraction
	remove int
}

type draft struct {
	view     View
	lines    []*line
	byLine   map[string]*line
	entries  []*entry
	byEntry  map[string]*entry
	result   computed
	checkers map[string]bool
}

type space struct {
	drafts   map[string]*draft
	requests map[string]string
	used     map[string]string
	runs     map[string]string
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

func (r *Registry) find(workspace, id string) (*draft, error) {
	s := r.workspaces[workspace]
	if s == nil || s.drafts[id] == nil {
		return nil, ErrNotFound
	}
	return s.drafts[id], nil
}

func (r *Registry) number(text string, signed, positive bool) (exact.Fraction, bool) {
	if len(text) > maxNumberBytes {
		return exact.Fraction{}, false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || !signed && value.Sign() < 0 || positive && value.Sign() == 0 {
		return exact.Fraction{}, false
	}
	return exact.Of(value)
}

func (r *Registry) bounded(f exact.Fraction) bool {
	text, ok := format(f)
	if !ok {
		return false
	}
	scale := 0
	if index := strings.IndexByte(text, '.'); index >= 0 {
		scale = len(text) - index - 1
	}
	units, _ := new(big.Int).SetString(strings.Replace(text, ".", "", 1), 10)
	_, err := decimal.New(r.profile, units, scale)
	return err == nil
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }

func fits(f exact.Fraction, scale int) bool {
	units, _ := f.Quanta(scale, "toward_zero")
	back, _ := exact.Units(units, scale)
	return same(back, f)
}

func round(f exact.Fraction, b Base) exact.Fraction {
	units, _ := f.Quanta(b.Scale, b.Mode)
	out, _ := exact.Units(units, b.Scale)
	return out
}

func refuse(reason string) error { return Refusal{Reason: reason} }

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func validDate(text string) bool {
	if len(text) != 10 || text < "2000-01-01" || text > "2099-12-31" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", text)
	return err == nil && parsed.Format("2006-01-02") == text
}

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validActor(a Actor) bool { return validID(a.ID) && kinds[a.Kind] }
