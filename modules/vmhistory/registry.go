package vmhistory

import (
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type span struct {
	spec   Span
	print  string
	owner  *asset
	cpus   *big.Int
	memory *big.Int
	disk   *big.Int
	state  State
	reason VoidReason
}

type reading struct {
	spec   Reading
	print  string
	micro  *big.Int
	state  State
	reason VoidReason
}

type counter struct {
	spec     Counter
	print    string
	owner    *asset
	factor   *big.Int
	state    State
	reason   VoidReason
	readings map[int64]*reading
	live     []*reading
}

type asset struct {
	spec     Asset
	print    string
	spans    int
	live     []*span
	counters []*counter
	change   uint64
}

type source struct {
	assets   map[string]*asset
	spans    map[string]*span
	counters map[string]*counter
	sequence uint64
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]map[string]*source
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, clock: clock, workspaces: map[string]map[string]*source{}}, nil
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

func instant(at time.Time) bool {
	return !at.IsZero() && at.Round(0).Nanosecond() == 0
}

func stamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return strconv.FormatInt(at.Unix(), 10)
}

func (r *Registry) origin(workspaceID, name string) *source {
	return r.workspaces[workspaceID][name]
}

func (home *source) touch(owner *asset) {
	home.sequence++
	owner.change = home.sequence
}

func (owner *asset) holds(from, to time.Time) bool {
	return !from.Before(owner.spec.Created) && (owner.spec.Deleted.IsZero() || !to.After(owner.spec.Deleted))
}

func (r *Registry) PutAsset(workspaceID, name string, spec Asset) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(spec.ID) || (spec.Kind != VM && spec.Kind != Host) || spec.Revision < 1 || spec.Revision > maxNumber {
		return ErrInvalid
	}
	if !instant(spec.Created) || (!spec.Deleted.IsZero() && (!instant(spec.Deleted) || !spec.Deleted.After(spec.Created))) {
		return ErrInvalid
	}
	spec.Created = spec.Created.UTC().Round(0)
	if !spec.Deleted.IsZero() {
		spec.Deleted = spec.Deleted.UTC().Round(0)
	}
	print := join(string(spec.Kind), stamp(spec.Created), stamp(spec.Deleted))
	home := r.origin(workspaceID, name)
	var held *asset
	if home != nil {
		held = home.assets[spec.ID]
	}
	if held != nil && held.spec.Revision == spec.Revision {
		if held.print != print {
			return ErrConflict
		}
		return nil
	}
	if held != nil && spec.Revision < held.spec.Revision {
		return ErrRejected
	}
	if spec.Created.After(r.now) || spec.Deleted.After(r.now) {
		return ErrRejected
	}
	if held != nil {
		revised := &asset{spec: spec}
		if count := len(held.live); count > 0 && !revised.holds(held.live[0].spec.From, held.live[count-1].spec.To) {
			return ErrRejected
		}
		for _, meter := range held.counters {
			if count := len(meter.live); meter.state == Live && count > 0 && !revised.holds(meter.live[0].spec.At, meter.live[count-1].spec.At) {
				return ErrRejected
			}
		}
		held.spec, held.print = spec, print
		home.touch(held)
		return nil
	}
	if home == nil && len(r.workspaces[workspaceID]) >= r.limits.Sources || home != nil && len(home.assets) >= r.limits.Assets {
		return ErrLimit
	}
	if home == nil {
		home = &source{assets: map[string]*asset{}, spans: map[string]*span{}, counters: map[string]*counter{}}
		if r.workspaces[workspaceID] == nil {
			r.workspaces[workspaceID] = map[string]*source{}
		}
		r.workspaces[workspaceID][name] = home
	}
	held = &asset{spec: spec, print: print}
	home.assets[spec.ID] = held
	home.touch(held)
	return nil
}

func (r *Registry) PutSpan(workspaceID, name string, spec Span) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(spec.ID) || !validID(spec.Asset) || (spec.Organization != "" && !validID(spec.Organization)) {
		return "", ErrInvalid
	}
	if !instant(spec.From) || !instant(spec.To) || !spec.To.After(spec.From) {
		return "", ErrInvalid
	}
	made := &span{spec: spec, state: Live}
	made.spec.From, made.spec.To = spec.From.UTC().Round(0), spec.To.UTC().Round(0)
	if spec.CPUs != "" {
		count, formed := whole(spec.CPUs, maxCPUDigits)
		if !formed || count.Cmp(big.NewInt(maxCPUs)) > 0 {
			return "", ErrInvalid
		}
		made.cpus = count
	}
	var formed, sized bool
	if made.memory, formed = bytesOf(spec.Memory); !formed {
		return "", ErrInvalid
	}
	if made.disk, sized = bytesOf(spec.Disk); !sized {
		return "", ErrInvalid
	}
	made.print = join(spec.Asset, stamp(made.spec.From), stamp(made.spec.To), "="+spec.CPUs, spec.Memory.Value, spec.Memory.Unit, spec.Disk.Value, spec.Disk.Unit, spec.Organization)
	home := r.origin(workspaceID, name)
	if home == nil || home.assets[spec.Asset] == nil {
		return "", ErrNotFound
	}
	owner := home.assets[spec.Asset]
	if earlier := home.spans[spec.ID]; earlier != nil {
		if earlier.print != made.print {
			return "", ErrConflict
		}
		return earlier.state, nil
	}
	if !owner.holds(made.spec.From, made.spec.To) || made.spec.To.After(r.now) {
		return "", ErrRejected
	}
	position := sort.Search(len(owner.live), func(index int) bool { return owner.live[index].spec.To.After(made.spec.From) })
	if position < len(owner.live) && owner.live[position].spec.From.Before(made.spec.To) {
		return "", ErrRejected
	}
	if owner.spans >= r.limits.Spans {
		return "", ErrLimit
	}
	made.owner = owner
	owner.spans++
	owner.live = append(owner.live, nil)
	copy(owner.live[position+1:], owner.live[position:])
	owner.live[position] = made
	home.spans[spec.ID] = made
	home.touch(owner)
	return Live, nil
}

func (r *Registry) VoidSpan(workspaceID, name, id string, reason VoidReason) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || !reason.known() {
		return "", ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.spans[id] == nil {
		return "", ErrNotFound
	}
	held := home.spans[id]
	if held.state == Voided {
		if held.reason != reason {
			return "", ErrConflict
		}
		return Voided, nil
	}
	owner := held.owner
	position := sort.Search(len(owner.live), func(index int) bool { return !owner.live[index].spec.From.Before(held.spec.From) })
	owner.live = append(owner.live[:position], owner.live[position+1:]...)
	held.state, held.reason = Voided, reason
	home.touch(owner)
	return Voided, nil
}

func (r *Registry) PutCounter(workspaceID, name string, spec Counter) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(spec.ID) || !validID(spec.Asset) || energyUnits[spec.Unit] == nil || spec.MaxGap < 1 || spec.MaxGap > r.limits.MaxGap {
		return "", ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.assets[spec.Asset] == nil {
		return "", ErrNotFound
	}
	owner := home.assets[spec.Asset]
	print := join(spec.Asset, spec.Unit, strconv.FormatInt(spec.MaxGap, 10))
	if earlier := home.counters[spec.ID]; earlier != nil {
		if earlier.print != print {
			return "", ErrConflict
		}
		return earlier.state, nil
	}
	if len(owner.counters) >= r.limits.Counters {
		return "", ErrLimit
	}
	made := &counter{spec: spec, print: print, owner: owner, factor: energyUnits[spec.Unit], state: Live, readings: map[int64]*reading{}}
	position := sort.Search(len(owner.counters), func(index int) bool { return owner.counters[index].spec.ID > spec.ID })
	owner.counters = append(owner.counters, nil)
	copy(owner.counters[position+1:], owner.counters[position:])
	owner.counters[position] = made
	home.counters[spec.ID] = made
	home.touch(owner)
	return Live, nil
}

func (r *Registry) VoidCounter(workspaceID, name, id string, reason VoidReason) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || !reason.known() {
		return "", ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.counters[id] == nil {
		return "", ErrNotFound
	}
	held := home.counters[id]
	if held.state == Voided {
		if held.reason != reason {
			return "", ErrConflict
		}
		return Voided, nil
	}
	held.state, held.reason = Voided, reason
	home.touch(held.owner)
	return Voided, nil
}

func (meter *counter) position(at time.Time) int {
	return sort.Search(len(meter.live), func(index int) bool { return !meter.live[index].spec.At.Before(at) })
}

func (r *Registry) PutReadings(workspaceID, name, id string, specs []Reading) ([]State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || len(specs) == 0 {
		return nil, ErrInvalid
	}
	seen := map[int64]bool{}
	for _, spec := range specs {
		if !instant(spec.At) || seen[spec.At.Unix()] || spec.Generation < 1 || spec.Generation > maxNumber || spec.Revision < 1 || spec.Revision > maxNumber {
			return nil, ErrInvalid
		}
		if _, formed := microOf(spec.Value, million); !formed {
			return nil, ErrInvalid
		}
		seen[spec.At.Unix()] = true
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.counters[id] == nil {
		return nil, ErrNotFound
	}
	if len(specs) > r.limits.Batch {
		return nil, ErrLimit
	}
	meter := home.counters[id]
	owner := meter.owner
	states := make([]State, len(specs))
	made := make([]*reading, len(specs))
	fresh, stale, added := 0, false, 0
	for index, spec := range specs {
		spec.At = spec.At.UTC().Round(0)
		print := join(spec.Value, strconv.Itoa(spec.Generation))
		earlier := meter.readings[spec.At.Unix()]
		switch {
		case earlier != nil && earlier.spec.Revision == spec.Revision && earlier.print != print:
			return nil, ErrConflict
		case earlier != nil && earlier.spec.Revision == spec.Revision:
			states[index] = earlier.state
			continue
		case earlier != nil && spec.Revision < earlier.spec.Revision:
			stale = true
		case earlier == nil:
			added++
		}
		micro, _ := microOf(spec.Value, meter.factor)
		made[index] = &reading{spec: spec, print: print, micro: micro, state: Live}
		states[index] = Live
		fresh++
	}
	if fresh == 0 {
		return states, nil
	}
	if stale || meter.state != Live {
		return nil, ErrRejected
	}
	replaced := map[int64]bool{}
	for _, entry := range made {
		if entry == nil {
			continue
		}
		if !owner.holds(entry.spec.At, entry.spec.At) || entry.spec.At.After(r.now) {
			return nil, ErrRejected
		}
		replaced[entry.spec.At.Unix()] = true
	}
	next := make([]*reading, 0, len(meter.live)+fresh)
	for _, kept := range meter.live {
		if !replaced[kept.spec.At.Unix()] {
			next = append(next, kept)
		}
	}
	for _, entry := range made {
		if entry != nil {
			next = append(next, entry)
		}
	}
	sort.Slice(next, func(a, b int) bool { return next[a].spec.At.Before(next[b].spec.At) })
	for index := 1; index < len(next); index++ {
		if next[index].spec.Generation < next[index-1].spec.Generation {
			return nil, ErrRejected
		}
	}
	if len(meter.readings)+added > r.limits.Readings {
		return nil, ErrLimit
	}
	for _, entry := range made {
		if entry != nil {
			meter.readings[entry.spec.At.Unix()] = entry
		}
	}
	meter.live = next
	home.touch(owner)
	return states, nil
}

func (r *Registry) VoidReading(workspaceID, name, id string, at time.Time, revision int, reason VoidReason) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || !instant(at) || revision < 1 || revision > maxNumber || !reason.known() {
		return "", ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.counters[id] == nil || home.counters[id].readings[at.Unix()] == nil {
		return "", ErrNotFound
	}
	meter := home.counters[id]
	held := meter.readings[at.Unix()]
	if held.state == Voided && held.spec.Revision == revision {
		if held.reason != reason {
			return "", ErrConflict
		}
		return Voided, nil
	}
	if meter.state != Live || held.spec.Revision != revision {
		return "", ErrRejected
	}
	position := meter.position(held.spec.At)
	meter.live = append(meter.live[:position], meter.live[position+1:]...)
	held.state, held.reason = Voided, reason
	home.touch(meter.owner)
	return Voided, nil
}
