package recognition

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type monthKey struct {
	profile string
	month   string
}

type profile struct {
	view   ProfileView
	closed map[string]bool
}

type batch struct {
	view    BatchView
	net     *big.Int
	tax     *big.Int
	subs    []*SubmissionView
	carrier *adjustment
	reverse *batch
	target  *batch
}

type adjustment struct {
	view    AdjustmentView
	net     *big.Int
	tax     *big.Int
	carrier *batch
}

type space struct {
	profiles    map[string]*profile
	batches     map[string]*batch
	byMonth     map[monthKey][]*batch
	submissions map[string]*SubmissionView
	byKey       map[string]*batch
	externals   map[string]string
	adjustments map[string]*adjustment
	adjByMonth  map[monthKey][]*adjustment
	clearances  map[string]map[string]*ClearanceView
	cleared     map[monthKey]string
}

type Registry struct {
	mu         sync.Mutex
	numbers    decimal.Profile
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(numbers decimal.Profile, limits Limits, clock func() time.Time) (*Registry, error) {
	if !numbers.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{numbers: numbers, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
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

var monthNames = map[string]bool{"01": true, "02": true, "03": true, "04": true, "05": true, "06": true, "07": true, "08": true, "09": true, "10": true, "11": true, "12": true}

func validMonth(text string) bool {
	if len(text) != 7 || text[:2] != "20" || text[4] != '-' || !monthNames[text[5:]] {
		return false
	}
	return text[2] >= '0' && text[2] <= '9' && text[3] >= '0' && text[3] <= '9'
}

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (a Actor) refusal() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (r *Registry) number(text string) bool {
	if len(text) > maxNumberBytes {
		return false
	}
	_, err := decimal.Parse(r.numbers, text)
	return err == nil
}

func (r *Registry) units(text string, scale int) (*big.Int, bool) {
	value, _ := decimal.Parse(r.numbers, text)
	if value.Scale() > scale {
		return nil, false
	}
	out := new(big.Int).Mul(value.Coefficient(), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale-value.Scale())), nil))
	return out, true
}

func (r *Registry) bounded(units *big.Int, scale int) bool {
	_, err := decimal.New(r.numbers, units, scale)
	return err == nil
}

func show(units *big.Int, scale int) string {
	negative := units.Sign() < 0
	digits := new(big.Int).Abs(units).String()
	for len(digits) <= scale {
		digits = "0" + digits
	}
	out := digits
	if scale > 0 {
		out = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if negative {
		out = "-" + out
	}
	return out
}

func hashOf(texts ...string) string {
	sum := sha256.New()
	for _, text := range texts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		sum.Write(size[:])
		sum.Write([]byte(text))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func flag(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (p Profile) valid() bool {
	return validID(p.ID) && validID(p.Entity) && validID(p.Ledger) && validID(p.Contract) && bases[p.Basis] &&
		taxBases[p.TaxBasis] && validCurrency(p.Currency) && p.Scale >= 0 && p.Scale <= maxScale && methods[p.Method]
}

func (r *Registry) workspace(id string) *space {
	return r.workspaces[id]
}

func (r *Registry) Define(workspace string, p Profile, actor Actor) (ProfileView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !p.valid() || !actor.valid() {
		return ProfileView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s != nil {
		if seen := s.profiles[p.ID]; seen != nil {
			if seen.view.Profile == p && actor.is(seen.view.Author) {
				return seen.copy(), nil
			}
			return ProfileView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return ProfileView{}, err
	}
	if s == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return ProfileView{}, ErrLimit
		}
		s = &space{
			profiles: map[string]*profile{}, batches: map[string]*batch{}, byMonth: map[monthKey][]*batch{},
			submissions: map[string]*SubmissionView{}, byKey: map[string]*batch{}, externals: map[string]string{},
			adjustments: map[string]*adjustment{}, adjByMonth: map[monthKey][]*adjustment{},
			clearances: map[string]map[string]*ClearanceView{}, cleared: map[monthKey]string{},
		}
		r.workspaces[workspace] = s
	} else if len(s.profiles) >= r.limits.Profiles {
		return ProfileView{}, ErrLimit
	}
	made := &profile{view: ProfileView{Profile: p, Author: actor.ID, At: now}, closed: map[string]bool{}}
	s.profiles[p.ID] = made
	return made.copy(), nil
}

func (p *profile) copy() ProfileView {
	v := p.view
	v.Closed = append([]string(nil), v.Closed...)
	return v
}

func (r *Registry) Close(workspace, id, period string, actor Actor) (ProfileView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validMonth(period) || !actor.valid() {
		return ProfileView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.profiles[id] == nil {
		return ProfileView{}, ErrNotFound
	}
	p := s.profiles[id]
	if err := actor.refusal(); err != nil {
		return ProfileView{}, err
	}
	if !p.closed[period] {
		p.closed[period] = true
		at := len(p.view.Closed)
		for at > 0 && p.view.Closed[at-1] > period {
			at--
		}
		p.view.Closed = append(p.view.Closed[:at], append([]string{period}, p.view.Closed[at:]...)...)
	}
	return p.copy(), nil
}

func (r *Registry) ProfileOf(workspace, id string) (ProfileView, error) {
	r.enter()
	defer r.leave()
	s := r.workspace(workspace)
	if s == nil || s.profiles[id] == nil {
		return ProfileView{}, ErrNotFound
	}
	return s.profiles[id].copy(), nil
}

func joined(list []string) string { return strings.Join(list, ",") }
