package costpool

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

var formedAmount = map[string]bool{"": true, "0": true, "100": true, "250.5": true, "0.0001": true, "1000.25": true, "40000": true, "-1": false, "0.00001": false, "x": false}

func ratOf(text string) *big.Rat {
	value, read := new(big.Rat).SetString(text)
	if !read {
		panic(text)
	}
	return value
}

func ratText(value *big.Rat) string {
	text := value.FloatString(12)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

type modelVersion struct {
	spec       Spec
	state      State
	recorded   bool
	superseded bool
	by         int
	approval   Approval
	reason     WithdrawReason
	assets     map[string]Asset
	components map[string]Component
	basis      *Basis
	plan       []func()
}

func (held *modelVersion) current() bool { return held.state == ApprovalRecorded && !held.superseded }

type world struct {
	limits   Limits
	pools    map[string]map[int]*modelVersion
	sequence map[string]uint64
}

func newWorld(limits Limits) *world {
	return &world{limits: limits, pools: map[string]map[int]*modelVersion{}, sequence: map[string]uint64{}}
}

func (w *world) find(ws, pool string, number int) *modelVersion { return w.pools[ws+"/"+pool][number] }

func headerKey(s Spec) string {
	return fmt.Sprintf("%s|%d|%d|%s|%s|%d", s.Currency, s.PeriodStart.Unix(), s.PeriodEnd.Unix(), s.Owner, s.Quantum, s.Supersedes)
}

func componentKey(c Component) string {
	model := ""
	if c.Power != nil {
		model = fmt.Sprintf("%s|%s|%s|%d|%d", c.Power.Grade, c.Power.Source, c.Power.PUE, c.Power.From.Unix(), c.Power.To.Unix())
	}
	c.Power = nil
	return fmt.Sprintf("%+v|%s", c, model)
}

func basisKey(b Basis) string {
	targets := append([]Target(nil), b.Targets...)
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	return fmt.Sprintf("%s|%s|%s|%v", b.ID, b.Measure, b.Capacity, targets)
}

func (w *world) pools0(ws string) int {
	count := 0
	for key := range w.pools {
		if strings.HasPrefix(key, ws+"/") {
			count++
		}
	}
	return count
}

func (w *world) open(spec Spec) (State, error) {
	quanta := map[string]bool{"1": true, "0.1": true, "0.01": true, "0.001": true, "0.0001": true}
	if spec.Version < 1 || spec.Supersedes < 0 || spec.Supersedes >= spec.Version || !quanta[spec.Quantum] || !spec.PeriodStart.Before(spec.PeriodEnd) || spec.PeriodEnd.Sub(spec.PeriodStart) > w.limits.MaxPeriod || spec.Owner == "" {
		return "", ErrInvalid
	}
	key := spec.WorkspaceID + "/" + spec.Pool
	if held := w.pools[key][spec.Version]; held != nil {
		if headerKey(held.spec) != headerKey(spec) {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if spec.Supersedes != 0 {
		named := w.pools[key][spec.Supersedes]
		if named == nil || !named.recorded || named.superseded || named.spec.Currency != spec.Currency || !named.spec.PeriodStart.Equal(spec.PeriodStart) || !named.spec.PeriodEnd.Equal(spec.PeriodEnd) {
			return "", ErrRejected
		}
	}
	if w.pools[key] == nil && w.pools0(spec.WorkspaceID) >= w.limits.Pools || len(w.pools[key]) >= w.limits.Versions {
		return "", ErrLimit
	}
	if w.pools[key] == nil {
		w.pools[key] = map[int]*modelVersion{}
	}
	w.pools[key][spec.Version] = &modelVersion{spec: spec, state: Draft, assets: map[string]Asset{}, components: map[string]Component{}}
	w.sequence[key]++
	return Draft, nil
}

func (w *world) asset(ws, pool string, number int, spec Asset) (State, error) {
	if !spec.Kind.known() || (spec.Method != Depreciation && spec.Method != Lease) || !formedAmount[spec.Acquisition] {
		return "", ErrInvalid
	}
	held := w.find(ws, pool, number)
	if held == nil {
		return "", ErrNotFound
	}
	if earlier, stored := held.assets[spec.ID]; stored {
		if earlier != spec {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	if len(held.assets) >= w.limits.Assets {
		return "", ErrLimit
	}
	held.assets[spec.ID] = spec
	w.sequence[ws+"/"+pool]++
	return held.state, nil
}

func modelFormed(c Component) bool {
	if !c.Kind.known() || c.Evidence == "" || !formedAmount[c.Amount] {
		return false
	}
	if (c.Kind == Depreciation || c.Kind == Lease) && c.Asset == "" {
		return false
	}
	if c.IncludedIn != "" && (c.IncludedIn == c.ID || (c.Kind != PowerCost && c.Kind != Licence && c.Kind != Operations)) {
		return false
	}
	if (c.Kind == PowerCost) != (c.Power != nil) {
		return false
	}
	if c.Power == nil {
		return true
	}
	model := c.Power
	if !model.From.Before(model.To) {
		return false
	}
	pue := model.PUE == "" || ratOf(model.PUE).Cmp(big.NewRat(1, 1)) >= 0
	switch model.Grade {
	case Measured, Estimated:
		return model.Source != "" && model.PUE != "" && pue
	case GradeNotKnown:
		return pue
	}
	return false
}

func (held *modelVersion) methodHolder(asset string) bool {
	for _, c := range held.components {
		if c.Asset == asset && (c.Kind == Depreciation || c.Kind == Lease) {
			return true
		}
	}
	return false
}

func (w *world) component(ws, pool string, number int, spec Component) (State, error) {
	if !modelFormed(spec) {
		return "", ErrInvalid
	}
	held := w.find(ws, pool, number)
	if held == nil {
		return "", ErrNotFound
	}
	if earlier, stored := held.components[spec.ID]; stored {
		if componentKey(earlier) != componentKey(spec) {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	if spec.Asset != "" {
		owner, exists := held.assets[spec.Asset]
		if !exists {
			return "", ErrRejected
		}
		if (spec.Kind == Depreciation || spec.Kind == Lease) && (owner.Method != spec.Kind || held.methodHolder(spec.Asset)) {
			return "", ErrRejected
		}
	}
	if spec.Power != nil && (spec.Power.From.Before(held.spec.PeriodStart) || spec.Power.To.After(held.spec.PeriodEnd)) {
		return "", ErrRejected
	}
	if spec.IncludedIn != "" {
		container, exists := held.components[spec.IncludedIn]
		if !exists || (container.Kind != Facility && container.Kind != Lease) {
			return "", ErrRejected
		}
	}
	if len(held.components) >= w.limits.Components {
		return "", ErrLimit
	}
	held.components[spec.ID] = spec
	w.sequence[ws+"/"+pool]++
	return held.state, nil
}

func (w *world) setBasis(ws, pool string, number int, spec Basis) (State, error) {
	if (spec.Measure != ReservedCapacity && spec.Measure != FixedAgreement) || len(spec.Targets) == 0 || !formedAmount[spec.Capacity] || spec.Capacity == "" || ratOf(spec.Capacity).Sign() <= 0 {
		return "", ErrInvalid
	}
	if len(spec.Targets) > w.limits.Targets {
		return "", ErrLimit
	}
	sum := new(big.Rat)
	ids := map[string]bool{}
	for _, target := range spec.Targets {
		if ids[target.ID] || !target.Bearer.known() || !formedAmount[target.Weight] || target.Weight == "" || (target.Idle && target.Bearer == Organization) {
			return "", ErrInvalid
		}
		ids[target.ID] = true
		sum.Add(sum, ratOf(target.Weight))
	}
	if sum.Cmp(ratOf(spec.Capacity)) != 0 {
		return "", ErrInvalid
	}
	held := w.find(ws, pool, number)
	if held == nil {
		return "", ErrNotFound
	}
	if held.basis != nil {
		if basisKey(*held.basis) != basisKey(spec) {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	copied := spec
	copied.Targets = append([]Target(nil), spec.Targets...)
	held.basis = &copied
	w.sequence[ws+"/"+pool]++
	return held.state, nil
}

type modelSums struct {
	total           *big.Rat
	summed, unknown int
	figures         *Figures
	assets          map[string]*big.Rat
}

func (held *modelVersion) sums() modelSums {
	out := modelSums{total: new(big.Rat), assets: map[string]*big.Rat{}, figures: &Figures{ByKind: map[Kind]string{}, ByGrade: map[Grade]string{}, ByBearer: map[Bearer]string{}}}
	kinds, grades, level := map[Kind]*big.Rat{}, map[Grade]*big.Rat{}, new(big.Rat)
	for _, c := range held.components {
		if c.IncludedIn != "" {
			if c.Amount == "" {
				out.figures.IncludedNotKnown++
			} else {
				out.figures.IncludedKnown++
			}
			continue
		}
		out.summed++
		if c.Amount == "" {
			out.unknown++
			continue
		}
		amount := ratOf(c.Amount)
		out.total.Add(out.total, amount)
		if kinds[c.Kind] == nil {
			kinds[c.Kind] = new(big.Rat)
		}
		kinds[c.Kind].Add(kinds[c.Kind], amount)
		if c.Asset == "" {
			out.figures.HasPoolLevel = true
			level.Add(level, amount)
		} else {
			if out.assets[c.Asset] == nil {
				out.assets[c.Asset] = new(big.Rat)
			}
			out.assets[c.Asset].Add(out.assets[c.Asset], amount)
		}
		if c.Power != nil {
			if grades[c.Power.Grade] == nil {
				grades[c.Power.Grade] = new(big.Rat)
			}
			grades[c.Power.Grade].Add(grades[c.Power.Grade], amount)
		}
	}
	for kind, sum := range kinds {
		out.figures.ByKind[kind] = ratText(sum)
	}
	for grade, sum := range grades {
		out.figures.ByGrade[grade] = ratText(sum)
	}
	if out.figures.HasPoolLevel {
		out.figures.PoolLevel = ratText(level)
	}
	for _, owned := range held.assets {
		if owned.Acquisition != "" {
			out.figures.PricesStated++
		} else {
			out.figures.PricesNotStated++
		}
	}
	out.figures.Total, out.figures.AssetSums = ratText(out.total), len(out.assets)
	return out
}

func (held *modelVersion) without() int {
	count := 0
	for id := range held.assets {
		if !held.methodHolder(id) {
			count++
		}
	}
	return count
}

func (w *world) overlapping(ws, pool string, held *modelVersion) []int {
	var out []int
	for number, other := range w.pools[ws+"/"+pool] {
		if other != held && number != held.spec.Supersedes && other.current() && held.spec.PeriodStart.Before(other.spec.PeriodEnd) && other.spec.PeriodStart.Before(held.spec.PeriodEnd) {
			out = append(out, number)
		}
	}
	sort.Ints(out)
	return out
}

func multiple(total *big.Rat, quantum string) bool {
	return new(big.Rat).Quo(total, ratOf(quantum)).IsInt()
}

func (w *world) record(ws, pool string, number int, approval Approval) (State, error) {
	if approval.Approver == "" || approval.Reference == "" {
		return "", ErrInvalid
	}
	held := w.find(ws, pool, number)
	if held == nil {
		return "", ErrNotFound
	}
	if held.recorded {
		if held.approval != approval {
			return "", ErrConflict
		}
		return held.state, nil
	}
	sums := held.sums()
	if held.state != Draft || sums.summed == 0 || sums.unknown > 0 || held.without() > 0 || held.basis == nil {
		return "", ErrRejected
	}
	var named *modelVersion
	if held.spec.Supersedes != 0 {
		if named = w.find(ws, pool, held.spec.Supersedes); named.superseded {
			return "", ErrRejected
		}
	}
	if len(w.overlapping(ws, pool, held)) > 0 || !multiple(sums.total, held.spec.Quantum) {
		return "", ErrRejected
	}
	held.state, held.recorded, held.approval = ApprovalRecorded, true, approval
	if named != nil {
		named.superseded, named.by = true, number
	}
	w.sequence[ws+"/"+pool]++
	return held.state, nil
}

func (w *world) withdraw(ws, pool string, number int, reason WithdrawReason) error {
	if reason != EnteredInError && reason != ByOperator {
		return ErrInvalid
	}
	held := w.find(ws, pool, number)
	switch {
	case held == nil:
		return ErrNotFound
	case held.state == Withdrawn && held.reason != reason:
		return ErrConflict
	case held.state == Withdrawn:
		return nil
	}
	held.state, held.reason = Withdrawn, reason
	w.sequence[ws+"/"+pool]++
	return nil
}

func standingOf(state State, recorded, superseded bool) string {
	switch {
	case state == Withdrawn:
		return "not_current[withdrawn]"
	case !recorded:
		return "not_current[no_approval_recorded]"
	case superseded:
		return "not_current[superseded]"
	}
	return "current[]"
}
