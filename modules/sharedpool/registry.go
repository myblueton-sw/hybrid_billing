package sharedpool

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type staged struct {
	weight Weight
	value  decimal.Decimal
	known  bool
}

type chunk struct {
	weights []staged
	print   string
}

type run struct {
	spec        RunSpec
	home        *version
	state       State
	judged      bool
	published   bool
	chunks      map[int]*chunk
	targets     map[string]int
	hold        string
	weightSum   string
	denominator string
	alternative *Alternative
	approval    Approval
	reason      WithdrawReason
	shares      []Share
	byBearer    map[Bearer]string
	residual    string
	residualWhy string
	started     time.Time
	changed     time.Time
}

type version struct {
	spec     Spec
	print    string
	home     *pool
	amount   decimal.Decimal
	quantum  decimal.Decimal
	capacity decimal.Decimal
	runs     map[string]*run
}

type pool struct {
	versions map[int]*version
	charged  []*run
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
	return &Registry{built: true, limits: limits, profile: profile, clock: clock, zero: zero, workspaces: map[string]map[string]*pool{}}, nil
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

func (r *Registry) period(from, to time.Time) bool {
	from, to = from.Round(0), to.Round(0)
	return from.Nanosecond() == 0 && to.Nanosecond() == 0 && from.Before(to) && to.Sub(from) <= r.limits.MaxPeriod
}

func (r *Registry) number(text string) (decimal.Decimal, bool) {
	if strings.HasPrefix(text, "-") {
		return decimal.Decimal{}, false
	}
	value, err := decimal.Parse(r.profile, text)
	return value, err == nil
}

func (r *Registry) find(workspaceID, name string, number int) *version {
	if home := r.workspaces[workspaceID][name]; home != nil {
		return home.versions[number]
	}
	return nil
}

func (r *Registry) lookup(workspaceID, name string, number int, id string) *run {
	if held := r.find(workspaceID, name, number); held != nil {
		return held.runs[id]
	}
	return nil
}

func (held *run) touch(r *Registry) {
	held.changed = r.now
	held.home.home.sequence++
}

func (r *Registry) OpenPool(spec Spec) error {
	defer r.leave()
	if !r.enter() || !validID(spec.WorkspaceID) || !validID(spec.Pool) || spec.Version < 1 || spec.Version > maxVersion || !validCurrency(spec.Currency) || !r.period(spec.PeriodStart, spec.PeriodEnd) {
		return ErrInvalid
	}
	quantum, sized := r.number(spec.Quantum)
	amount, formed := r.number(spec.Amount)
	if _, fraction, _ := strings.Cut(spec.Amount, "."); !validQuantum(spec.Quantum) || !sized || !formed || len(fraction) > maxAmountFraction || amount.Scale() > quantum.Scale() {
		return ErrInvalid
	}
	made := &version{amount: amount, quantum: quantum, runs: map[string]*run{}}
	if spec.Capacity == "" {
		if spec.ResidualBearer != "" {
			return ErrInvalid
		}
	} else {
		capacity, stated := r.number(spec.Capacity)
		if !stated || capacity.Sign() <= 0 || !spec.ResidualBearer.residual() {
			return ErrInvalid
		}
		made.capacity = capacity
	}
	spec.PeriodStart, spec.PeriodEnd = spec.PeriodStart.UTC().Round(0), spec.PeriodEnd.UTC().Round(0)
	made.spec = spec
	made.print = join(spec.Currency, spec.PeriodStart.Format(time.RFC3339), spec.PeriodEnd.Format(time.RFC3339), spec.Amount, spec.Quantum, "="+spec.Capacity, string(spec.ResidualBearer))
	if held := r.find(spec.WorkspaceID, spec.Pool, spec.Version); held != nil {
		if held.print != made.print {
			return ErrConflict
		}
		return nil
	}
	pools := r.workspaces[spec.WorkspaceID]
	home := pools[spec.Pool]
	if home == nil && len(pools) >= r.limits.Pools || home != nil && len(home.versions) >= r.limits.Versions {
		return ErrLimit
	}
	if pools == nil {
		pools = map[string]*pool{}
		r.workspaces[spec.WorkspaceID] = pools
	}
	if home == nil {
		home = &pool{versions: map[int]*version{}}
		pools[spec.Pool] = home
	}
	made.home = home
	home.versions[spec.Version] = made
	home.sequence++
	return nil
}

func (r *Registry) StartRun(workspaceID, name string, number int, spec RunSpec) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(spec.ID) || (spec.Purpose != Showback && spec.Purpose != Simulation && spec.Purpose != Chargeback) || spec.Chunks < 1 {
		return "", ErrInvalid
	}
	home := r.find(workspaceID, name, number)
	if home == nil {
		return "", ErrNotFound
	}
	if spec.Chunks > r.limits.Chunks {
		return "", ErrLimit
	}
	if held := home.runs[spec.ID]; held != nil {
		if held.spec != spec {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if len(home.runs) >= r.limits.Runs {
		return "", ErrLimit
	}
	made := &run{spec: spec, home: home, state: Staging, chunks: map[int]*chunk{}, targets: map[string]int{}, started: r.now}
	home.runs[spec.ID] = made
	made.touch(r)
	return Staging, nil
}

func (r *Registry) StageChunk(workspaceID, name string, number int, id string, position int, weights []Weight) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) || position < 1 || len(weights) == 0 {
		return "", ErrInvalid
	}
	if len(weights) > r.limits.ChunkWeights {
		return "", ErrLimit
	}
	made := &chunk{weights: make([]staged, len(weights))}
	seen := map[string]bool{}
	for index, weight := range weights {
		entry := staged{weight: weight}
		if !validID(weight.Target) || weight.Target[0] == ':' || seen[weight.Target] || !weight.Bearer.known() {
			return "", ErrInvalid
		}
		if weight.Weight != "" {
			value, formed := r.number(weight.Weight)
			if !formed {
				return "", ErrInvalid
			}
			entry.value, entry.known = value, true
		}
		seen[weight.Target] = true
		made.weights[index] = entry
	}
	sort.Slice(made.weights, func(a, b int) bool { return made.weights[a].weight.Target < made.weights[b].weight.Target })
	prints := make([]string, 0, 3*len(weights))
	for _, entry := range made.weights {
		prints = append(prints, entry.weight.Target, string(entry.weight.Bearer), "="+entry.weight.Weight)
	}
	made.print = join(prints...)
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return "", ErrNotFound
	}
	if earlier := held.chunks[position]; earlier != nil {
		if earlier.print != made.print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Staging || position > held.spec.Chunks {
		return "", ErrRejected
	}
	for _, entry := range made.weights {
		if held.targets[entry.weight.Target] != 0 {
			return "", ErrRejected
		}
	}
	if len(held.targets)+len(weights) > r.limits.Targets {
		return "", ErrLimit
	}
	for _, entry := range made.weights {
		held.targets[entry.weight.Target] = position
	}
	held.chunks[position] = made
	held.touch(r)
	return held.state, nil
}

func (r *Registry) Withdraw(workspaceID, name string, number int, id string, reason WithdrawReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) || (reason != EnteredInError && reason != ByOperator) {
		return ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Withdrawn {
		if held.reason != reason {
			return ErrConflict
		}
		return nil
	}
	home := held.home.home
	for index, other := range home.charged {
		if other == held {
			home.charged = append(home.charged[:index], home.charged[index+1:]...)
			break
		}
	}
	held.state, held.reason = Withdrawn, reason
	held.touch(r)
	return nil
}
