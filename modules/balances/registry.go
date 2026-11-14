package balances

import (
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type lot struct {
	number    int
	source    string
	record    string
	amount    exact.Fraction
	remainder exact.Fraction
	expiry    time.Time
}

type account struct {
	a         Account
	author    string
	digest    string
	writtenAt time.Time
	openings  []*opening
	increases []*increase
	apps      []*application
	live      []*application
	lots      []*lot
}

type opening struct {
	o           Opening
	acct        *account
	state       string
	author      string
	approver    string
	withdrawnBy string
	digest      string
	writtenAt   time.Time
	approvedAt  time.Time
	withdrawnAt time.Time
	lot         *lot
	reversals   []*reversal
}

type increase struct {
	i           Increase
	acct        *account
	state       string
	author      string
	approver    string
	withdrawnBy string
	digest      string
	writtenAt   time.Time
	approvedAt  time.Time
	withdrawnAt time.Time
	lot         *lot
	reversals   []*reversal
}

type hold struct {
	lot      *lot
	amount   exact.Fraction
	returned exact.Fraction
}

type target struct {
	customer string
	entity   string
	currency string
	amount   string
	apps     []*application
}

type application struct {
	a           Application
	acct        *account
	tgt         *target
	state       string
	applied     exact.Fraction
	holds       []*hold
	expiry      time.Time
	reversals   []*reversal
	digest      string
	reservedBy  string
	reservedAt  time.Time
	confirmedBy string
	confirmedAt time.Time
	releasedBy  string
	releasedAt  time.Time
}

type reversal struct {
	r          Reversal
	state      string
	author     string
	approver   string
	digest     string
	writtenAt  time.Time
	approvedAt time.Time
}

type space struct {
	accounts          map[string]*account
	keys              map[Account]string
	openings          map[string]*opening
	migrations        map[string]*opening
	increases         map[string]*increase
	references        map[[2]string]*increase
	importedIncreases map[[2]string]*increase
	importedApps      map[[2]string]*application
	applications      map[string]*application
	targets           map[[2]string]*target
	reversals         map[string]*reversal
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

func (r *Registry) space(workspace string) *space {
	s := r.workspaces[workspace]
	if s == nil {
		s = &space{accounts: map[string]*account{}, keys: map[Account]string{}, openings: map[string]*opening{}, migrations: map[string]*opening{},
			increases: map[string]*increase{}, references: map[[2]string]*increase{}, importedIncreases: map[[2]string]*increase{}, importedApps: map[[2]string]*application{}, applications: map[string]*application{},
			targets: map[[2]string]*target{}, reversals: map[string]*reversal{}}
		r.workspaces[workspace] = s
	}
	return s
}

func (r *Registry) lookup(workspace, id string) *account {
	s := r.workspaces[workspace]
	if s == nil {
		return nil
	}
	return s.accounts[id]
}

func (r *Registry) amount(text string, positive bool) (string, bool) {
	if len(text) > maxNumberBytes {
		return "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || value.Sign() < 0 || positive && value.Sign() == 0 {
		return "", false
	}
	return value.String(), true
}

func (r *Registry) value(text string) exact.Fraction {
	parsed, _ := decimal.Parse(r.profile, text)
	made, _ := exact.Of(parsed)
	return made
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }

func least(values ...exact.Fraction) exact.Fraction {
	out := values[0]
	for _, v := range values[1:] {
		if v.Less(out) {
			out = v
		}
	}
	return out
}

func today(now time.Time) string { return now.Format("2006-01-02") }

func expiryOf(date string) time.Time {
	if date == "" {
		return time.Time{}
	}
	parsed, _ := time.Parse("2006-01-02", date)
	return parsed.Add(24 * time.Hour)
}

func (l *lot) expired(now time.Time) bool { return !l.expiry.IsZero() && !now.Before(l.expiry) }

func (r *Registry) held(acct *account, l *lot, now time.Time) exact.Fraction {
	total := exact.Whole(0)
	for _, app := range acct.live {
		if app.state != StateReserved || !now.Before(app.expiry) {
			continue
		}
		for _, h := range app.holds {
			if h.lot == l {
				total = total.Plus(h.amount)
			}
		}
	}
	return total
}

func (r *Registry) available(acct *account, now time.Time) exact.Fraction {
	total := exact.Whole(0)
	for _, l := range acct.lots {
		if !l.expired(now) {
			total = total.Plus(l.remainder.Minus(r.held(acct, l, now)))
		}
	}
	return total
}

func (a *account) pending() bool {
	for _, o := range a.openings {
		if o.state == StatePending {
			return true
		}
	}
	return false
}

func (a *account) opened() *opening {
	for _, o := range a.openings {
		if o.state != StateWithdrawn {
			return o
		}
	}
	return nil
}

func (s *space) covered(acct *account, i Import) bool {
	if own := acct.opened(); own != nil && own.o.AsOf >= i.Occurred {
		return true
	}
	for _, o := range s.openings {
		if o.state != StateWithdrawn && o.o.PriorLedger == i.Ledger && o.o.AsOf >= i.Occurred {
			return true
		}
	}
	return false
}

func importKey(i Import) [2]string { return [2]string{i.Ledger, i.Transaction} }

func (r *Registry) imported(s *space, i Import, now time.Time) bool {
	if i == (Import{}) {
		return false
	}
	if inc := s.importedIncreases[importKey(i)]; inc != nil && inc.state != StateWithdrawn {
		return true
	}
	app := s.importedApps[importKey(i)]
	return app != nil && (app.state == StateConfirmed || app.state == StateReserved && now.Before(app.expiry))
}

func (a *account) prune(now time.Time) {
	kept := a.live[:0]
	for _, app := range a.live {
		if app.state == StateReserved && now.Before(app.expiry) {
			kept = append(kept, app)
		}
	}
	for index := len(kept); index < len(a.live); index++ {
		a.live[index] = nil
	}
	a.live = kept
}

func (s *space) importedBefore(ledger, asOf string, now time.Time) bool {
	for key, inc := range s.importedIncreases {
		if key[0] == ledger && inc.state != StateWithdrawn && inc.i.Import.Occurred <= asOf {
			return true
		}
	}
	for key, app := range s.importedApps {
		if key[0] == ledger && app.a.Import.Occurred <= asOf && (app.state == StateConfirmed || app.state == StateReserved && now.Before(app.expiry)) {
			return true
		}
	}
	return false
}

func (r *Registry) importRefused(s *space, acct *account, i Import, now time.Time) error {
	if i == (Import{}) {
		return nil
	}
	if i.Occurred > today(now) || s.covered(acct, i) {
		return ErrRejected
	}
	return nil
}

func accountForm(a Account) bool {
	if !validID(a.ID) || !validID(a.Customer) || !validID(a.Entity) || !validCurrency(a.Currency) || !balanceKinds[a.Kind] || !authorities[a.Authority] {
		return false
	}
	return (a.Authority == AuthorityExternal) == validID(a.Ledger) && optionalID(a.Ledger)
}

func viewAccount(acct *account) AccountView {
	return AccountView{Account: acct.a, Author: acct.author, Digest: acct.digest, WrittenAt: acct.writtenAt}
}

func (r *Registry) OpenAccount(workspace string, a Account, author string) (AccountView, error) {
	if !validID(workspace) || !validID(author) || !accountForm(a) {
		return AccountView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	if acct := r.lookup(workspace, a.ID); acct != nil {
		if acct.a != a {
			return AccountView{}, ErrConflict
		}
		return viewAccount(acct), nil
	}
	key := Account{Customer: a.Customer, Entity: a.Entity, Currency: a.Currency, Kind: a.Kind}
	s := r.workspaces[workspace]
	if s != nil && s.keys[key] != "" {
		return AccountView{}, ErrRejected
	}
	if s != nil && len(s.accounts) >= r.limits.Accounts {
		return AccountView{}, ErrLimit
	}
	s = r.space(workspace)
	acct := &account{a: a, author: author, writtenAt: now, digest: accountDigest(workspace, a)}
	s.accounts[a.ID] = acct
	s.keys[key] = a.ID
	return viewAccount(acct), nil
}

func (r *Registry) Account(workspace, id string) (AccountView, error) {
	if !validID(workspace) || !validID(id) {
		return AccountView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	acct := r.lookup(workspace, id)
	if acct == nil {
		return AccountView{}, ErrNotFound
	}
	return viewAccount(acct), nil
}
