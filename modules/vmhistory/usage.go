package vmhistory

import (
	"math/big"
	"sort"
	"time"
)

type tally struct {
	seconds  int64
	sums     [3]*big.Int
	unstated [3]int64
}

func (t *tally) add(seconds int64, values [3]*big.Int) {
	t.seconds += seconds
	for index, value := range values {
		if value == nil {
			t.unstated[index] += seconds
			continue
		}
		if t.sums[index] == nil {
			t.sums[index] = new(big.Int)
		}
		t.sums[index].Add(t.sums[index], new(big.Int).Mul(value, big.NewInt(seconds)))
	}
}

func (t *tally) entry(organization string) Entry {
	out := Entry{Organization: organization, Seconds: t.seconds}
	for index, target := range []*Dimension{&out.CPU, &out.Memory, &out.Disk} {
		target.NotStatedSeconds = t.unstated[index]
		if t.sums[index] != nil {
			target.Integral = t.sums[index].String()
		}
	}
	return out
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func earlier(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func configuration(owner *asset, from, to time.Time, window int64) Configuration {
	out := Configuration{Uncovered: window}
	byOrganization := map[string]*tally{}
	var unstated [3]bool
	first := sort.Search(len(owner.live), func(index int) bool { return owner.live[index].spec.To.After(from) })
	for _, held := range owner.live[first:] {
		if !held.spec.From.Before(to) {
			break
		}
		seconds := int64(earlier(held.spec.To, to).Sub(later(held.spec.From, from)) / time.Second)
		entry := byOrganization[held.spec.Organization]
		if entry == nil {
			entry = &tally{}
			byOrganization[held.spec.Organization] = entry
		}
		values := [3]*big.Int{held.cpus, held.memory, held.disk}
		entry.add(seconds, values)
		for index, value := range values {
			unstated[index] = unstated[index] || value == nil
		}
		out.Covered += seconds
	}
	out.Uncovered -= out.Covered
	names := make([]string, 0, len(byOrganization))
	for name := range byOrganization {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry := byOrganization[name].entry(name)
		if name == "" {
			out.None = &entry
			continue
		}
		out.Entries = append(out.Entries, entry)
	}
	if out.Uncovered > 0 {
		out.Reasons = append(out.Reasons, NotCovered)
	}
	for index, reason := range []string{CPUNotStated, MemoryNotStated, DiskNotStated} {
		if unstated[index] {
			out.Reasons = append(out.Reasons, reason)
		}
	}
	out.Result = Incomplete
	if len(out.Reasons) == 0 {
		out.Result = Complete
	}
	return out
}

type stretch struct {
	from, to time.Time
}

func union(parts []stretch, from, to time.Time) int64 {
	sort.Slice(parts, func(a, b int) bool { return parts[a].from.Before(parts[b].from) })
	var total int64
	reached := from
	for _, part := range parts {
		start, end := later(part.from, reached), earlier(part.to, to)
		if end.After(start) {
			total += int64(end.Sub(start) / time.Second)
			reached = end
		}
	}
	return total
}

func shared(parts []stretch) bool {
	sort.Slice(parts, func(a, b int) bool { return parts[a].from.Before(parts[b].from) })
	var best time.Time
	for _, part := range parts {
		if part.from.Before(best) {
			return true
		}
		best = part.to
	}
	return false
}

func energy(owner *asset, from, to time.Time, window int64) Energy {
	out := Energy{Result: Incomplete, Uncovered: window}
	var all, inside []stretch
	var paired []stretch
	found := map[string]bool{}
	total := new(big.Int)
	contributes := false
	meters := 0
	for _, meter := range owner.counters {
		if meter.state != Live {
			continue
		}
		meters++
		entry := CounterEnergy{Counter: meter.spec.ID}
		sum := new(big.Int)
		counted := false
		first := sort.Search(len(meter.live), func(index int) bool { return meter.live[index].spec.At.After(from) })
		if first > 0 {
			first--
		}
		for index := first; index+1 < len(meter.live); index++ {
			a, b := meter.live[index], meter.live[index+1]
			if !a.spec.At.Before(to) {
				break
			}
			part := stretch{from: a.spec.At, to: b.spec.At}
			all = append(all, part)
			switch {
			case a.spec.Generation != b.spec.Generation:
				found[CounterReset] = true
			case b.micro.Cmp(a.micro) < 0:
				found[CounterDecrease] = true
			case int64(b.spec.At.Sub(a.spec.At)/time.Second) > meter.spec.MaxGap:
				found[CounterGap] = true
			default:
				paired = append(paired, part)
				if a.spec.At.Before(from) || b.spec.At.After(to) {
					found[BoundaryNotRead] = true
					continue
				}
				inside = append(inside, part)
				sum.Add(sum, new(big.Int).Sub(b.micro, a.micro))
				entry.Seconds += int64(b.spec.At.Sub(a.spec.At) / time.Second)
				counted = true
			}
		}
		if counted {
			entry.Energy = wattHours(sum)
			total.Add(total, sum)
			contributes = true
		}
		out.Counters = append(out.Counters, entry)
	}
	if meters == 0 {
		out.Reasons = []string{NoCounter}
		return out
	}
	out.Covered = union(inside, from, to)
	out.Uncovered = window - out.Covered
	found[CountersOverlap] = shared(paired)
	found[NotRead] = union(all, from, to) < window
	for _, reason := range []string{CountersOverlap, CounterReset, CounterDecrease, CounterGap, BoundaryNotRead, NotRead} {
		if found[reason] {
			out.Reasons = append(out.Reasons, reason)
		}
	}
	if contributes && !found[CountersOverlap] {
		out.Total = wattHours(total)
	}
	if len(out.Reasons) == 0 {
		out.Result = Complete
	}
	return out
}

func (r *Registry) Usage(workspaceID, name, id string, from, to time.Time) (Result, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !validID(id) || !instant(from) || !instant(to) || !to.After(from) || to.Round(0).Sub(from.Round(0)) > r.limits.MaxPeriod {
		return Result{}, ErrInvalid
	}
	home := r.origin(workspaceID, name)
	if home == nil || home.assets[id] == nil {
		return Result{}, ErrNotFound
	}
	owner := home.assets[id]
	start, end := later(from.UTC().Round(0), owner.spec.Created), to.UTC().Round(0)
	if !owner.spec.Deleted.IsZero() {
		end = earlier(end, owner.spec.Deleted)
	}
	out := Result{Asset: id, Sequence: home.sequence, Change: owner.change, Configuration: Configuration{Result: NotApplicable}, Energy: Energy{Result: NotApplicable}}
	if !end.After(start) {
		return out, nil
	}
	out.WindowStart, out.WindowEnd, out.Window = start, end, int64(end.Sub(start)/time.Second)
	out.Configuration = configuration(owner, start, end, out.Window)
	out.Energy = energy(owner, start, end, out.Window)
	return out, nil
}
