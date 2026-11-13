package costpool

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type asset struct {
	spec      Asset
	print     string
	component bool
}

type component struct {
	spec   Component
	print  string
	amount decimal.Decimal
	known  bool
}

type basis struct {
	spec    Basis
	print   string
	weights map[string]decimal.Decimal
}

type version struct {
	spec         Spec
	print        string
	home         *pool
	state        State
	recorded     bool
	superseded   bool
	supersededBy int
	approval     Approval
	reason       WithdrawReason
	assets       map[string]*asset
	components   map[string]*component
	basis        *basis
	figures      *Figures
	shares       []Share
	assetSums    []AssetSum
	opened       time.Time
	changed      time.Time
}

type pool struct {
	versions map[int]*version
	sequence uint64
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	profile    decimal.Profile
	clock      func() time.Time
	now        time.Time
	zero       decimal.Decimal
	one        decimal.Decimal
	workspaces map[string]map[string]*pool
}

func NewRegistry(limits Limits, profile decimal.Profile, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || !profile.Valid() || clock == nil {
		return nil, ErrInvalid
	}
	zero, err := decimal.Parse(profile, "0")
	if err != nil {
		return nil, ErrInvalid
	}
	one, err := decimal.Parse(profile, "1")
	if err != nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, profile: profile, clock: clock, zero: zero, one: one, workspaces: map[string]map[string]*pool{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock().UTC().Round(0); read.After(r.now) {
		r.now = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func join(fields ...string) string {
	var out strings.Builder
	for _, field := range fields {
		out.WriteString(strconv.Itoa(len(field)))
		out.WriteByte(':')
		out.WriteString(field)
	}
	return out.String()
}

func whole(from, to time.Time) bool {
	from, to = from.Round(0), to.Round(0)
	return from.Nanosecond() == 0 && to.Nanosecond() == 0 && from.Before(to)
}

func (r *Registry) period(from, to time.Time) bool {
	return whole(from, to) && to.Round(0).Sub(from.Round(0)) <= r.limits.MaxPeriod
}

func (r *Registry) number(text string) (decimal.Decimal, bool) {
	if strings.HasPrefix(text, "-") {
		return decimal.Decimal{}, false
	}
	value, err := decimal.Parse(r.profile, text)
	return value, err == nil
}

func (r *Registry) atLeast(value, bound decimal.Decimal) bool {
	order, err := decimal.Cmp(r.profile, value, bound)
	return err == nil && order >= 0
}

func (r *Registry) find(workspaceID, name string, number int) *version {
	if home := r.workspaces[workspaceID][name]; home != nil {
		return home.versions[number]
	}
	return nil
}

func (held *version) current() bool {
	return held.state == ApprovalRecorded && !held.superseded
}

func (held *version) overlaps(other *version) bool {
	return held.spec.PeriodStart.Before(other.spec.PeriodEnd) && other.spec.PeriodStart.Before(held.spec.PeriodEnd)
}

func (held *version) touch(r *Registry) {
	held.changed = r.now
	held.home.sequence++
}

func (r *Registry) Open(spec Spec) (State, error) {
	defer r.leave()
	if !r.enter() {
		return "", ErrInvalid
	}
	if !validID(spec.WorkspaceID) || !validID(spec.Pool) || !validID(spec.Owner) || spec.Version > maxVersion || !validCurrency(spec.Currency) {
		return "", ErrInvalid
	}
	if _, parsed := r.number(spec.Quantum); !validQuantum(spec.Quantum) || !parsed || !r.period(spec.PeriodStart, spec.PeriodEnd) || spec.Supersedes < 0 || spec.Supersedes >= spec.Version {
		return "", ErrInvalid
	}
	spec.PeriodStart, spec.PeriodEnd = spec.PeriodStart.UTC().Round(0), spec.PeriodEnd.UTC().Round(0)
	print := join(spec.Currency, spec.PeriodStart.Format(time.RFC3339), spec.PeriodEnd.Format(time.RFC3339), spec.Owner, spec.Quantum, strconv.Itoa(spec.Supersedes))
	if held := r.find(spec.WorkspaceID, spec.Pool, spec.Version); held != nil {
		if held.print != print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	pools := r.workspaces[spec.WorkspaceID]
	home := pools[spec.Pool]
	if spec.Supersedes != 0 {
		var named *version
		if home != nil {
			named = home.versions[spec.Supersedes]
		}
		if named == nil || !named.recorded || named.superseded || named.spec.Currency != spec.Currency || !named.spec.PeriodStart.Equal(spec.PeriodStart) || !named.spec.PeriodEnd.Equal(spec.PeriodEnd) {
			return "", ErrRejected
		}
	}
	if home == nil && len(pools) >= r.limits.Pools || home != nil && len(home.versions) >= r.limits.Versions {
		return "", ErrLimit
	}
	if pools == nil {
		pools = map[string]*pool{}
		r.workspaces[spec.WorkspaceID] = pools
	}
	if home == nil {
		home = &pool{versions: map[int]*version{}}
		pools[spec.Pool] = home
	}
	made := &version{spec: spec, print: print, home: home, state: Draft, assets: map[string]*asset{}, components: map[string]*component{}, opened: r.now}
	home.versions[spec.Version] = made
	made.touch(r)
	return Draft, nil
}

func (r *Registry) PutAsset(workspaceID, name string, number int, spec Asset) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(spec.ID) || !spec.Kind.known() || (spec.Method != Depreciation && spec.Method != Lease) {
		return "", ErrInvalid
	}
	if _, parsed := r.number(spec.Acquisition); spec.Acquisition != "" && !parsed {
		return "", ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return "", ErrNotFound
	}
	print := join(string(spec.Kind), string(spec.Method), spec.Acquisition)
	if earlier := held.assets[spec.ID]; earlier != nil {
		if earlier.print != print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	if len(held.assets) >= r.limits.Assets {
		return "", ErrLimit
	}
	held.assets[spec.ID] = &asset{spec: spec, print: print}
	held.touch(r)
	return held.state, nil
}

func (r *Registry) shape(spec Component) (*component, bool) {
	if !validID(spec.ID) || !spec.Kind.known() || !validID(spec.Evidence) || (spec.Asset != "" && !validID(spec.Asset)) {
		return nil, false
	}
	made := &component{spec: spec}
	if spec.Amount != "" {
		amount, parsed := r.number(spec.Amount)
		if _, fraction, _ := strings.Cut(spec.Amount, "."); !parsed || len(fraction) > maxAmountFraction {
			return nil, false
		}
		made.amount, made.known = amount, true
	}
	if (spec.Kind == Depreciation || spec.Kind == Lease) && spec.Asset == "" {
		return nil, false
	}
	if spec.IncludedIn != "" && (!validID(spec.IncludedIn) || spec.IncludedIn == spec.ID || (spec.Kind != PowerCost && spec.Kind != Licence && spec.Kind != Operations)) {
		return nil, false
	}
	prints := []string{string(spec.Kind), "=" + spec.Amount, spec.Asset, spec.Evidence, spec.IncludedIn}
	if (spec.Kind == PowerCost) != (spec.Power != nil) {
		return nil, false
	}
	if spec.Power != nil {
		model := *spec.Power
		stated := model.Grade == Measured || model.Grade == Estimated
		if (!stated && model.Grade != GradeNotKnown) || !whole(model.From, model.To) {
			return nil, false
		}
		if (stated || model.Source != "") && !validID(model.Source) {
			return nil, false
		}
		if stated || model.PUE != "" {
			if ratio, parsed := r.number(model.PUE); !parsed || !r.atLeast(ratio, r.one) {
				return nil, false
			}
		}
		model.From, model.To = model.From.UTC().Round(0), model.To.UTC().Round(0)
		made.spec.Power = &model
		prints = append(prints, "power", string(model.Grade), model.Source, model.PUE, model.From.Format(time.RFC3339), model.To.Format(time.RFC3339))
	}
	made.print = join(prints...)
	return made, true
}

func (r *Registry) PutComponent(workspaceID, name string, number int, spec Component) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 {
		return "", ErrInvalid
	}
	made, formed := r.shape(spec)
	if !formed {
		return "", ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return "", ErrNotFound
	}
	if earlier := held.components[spec.ID]; earlier != nil {
		if earlier.print != made.print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	var owner *asset
	if spec.Asset != "" {
		if owner = held.assets[spec.Asset]; owner == nil {
			return "", ErrRejected
		}
	}
	if spec.Kind == Depreciation || spec.Kind == Lease {
		if owner.spec.Method != spec.Kind || owner.component {
			return "", ErrRejected
		}
	}
	if model := made.spec.Power; model != nil && (model.From.Before(held.spec.PeriodStart) || model.To.After(held.spec.PeriodEnd)) {
		return "", ErrRejected
	}
	if spec.IncludedIn != "" {
		if container := held.components[spec.IncludedIn]; container == nil || (container.spec.Kind != Facility && container.spec.Kind != Lease) {
			return "", ErrRejected
		}
	}
	if len(held.components) >= r.limits.Components {
		return "", ErrLimit
	}
	held.components[spec.ID] = made
	if spec.Kind == Depreciation || spec.Kind == Lease {
		owner.component = true
	}
	held.touch(r)
	return held.state, nil
}

func (r *Registry) SetBasis(workspaceID, name string, number int, spec Basis) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(spec.ID) || (spec.Measure != ReservedCapacity && spec.Measure != FixedAgreement) {
		return "", ErrInvalid
	}
	capacity, parsed := r.number(spec.Capacity)
	if !parsed || capacity.Sign() <= 0 {
		return "", ErrInvalid
	}
	if len(spec.Targets) > r.limits.Targets {
		return "", ErrLimit
	}
	made := &basis{spec: spec, weights: map[string]decimal.Decimal{}}
	made.spec.Targets = append([]Target(nil), spec.Targets...)
	sort.Slice(made.spec.Targets, func(a, b int) bool { return made.spec.Targets[a].ID < made.spec.Targets[b].ID })
	prints := []string{spec.ID, string(spec.Measure), spec.Capacity}
	sum := r.zero
	for _, target := range made.spec.Targets {
		weight, parsed := r.number(target.Weight)
		if _, twice := made.weights[target.ID]; !validID(target.ID) || twice || !target.Bearer.known() || !parsed || (target.Idle && target.Bearer == Organization) {
			return "", ErrInvalid
		}
		var err error
		if sum, err = decimal.Add(r.profile, sum, weight); err != nil {
			return "", ErrInvalid
		}
		made.weights[target.ID] = weight
		prints = append(prints, target.ID, string(target.Bearer), target.Weight, strconv.FormatBool(target.Idle))
	}
	if order, err := decimal.Cmp(r.profile, sum, capacity); err != nil || order != 0 {
		return "", ErrInvalid
	}
	made.print = join(prints...)
	held := r.find(workspaceID, name, number)
	if held == nil {
		return "", ErrNotFound
	}
	if held.basis != nil {
		if held.basis.print != made.print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	held.basis = made
	held.touch(r)
	return held.state, nil
}
