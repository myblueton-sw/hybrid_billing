package vmhistory

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

type mAsset struct {
	kind             AssetKind
	created, deleted int64
	revision         int
	change           uint64
}

type mSpan struct {
	id, asset, print string
	from, to         int64
	values           [3]*big.Int
	organization     string
	voided           bool
	reason           VoidReason
}

type mCounter struct {
	id, asset, print string
	gap              int64
	factor           int64
	voided           bool
	reason           VoidReason
}

type mReading struct {
	counter    string
	at         int64
	text       string
	micro      *big.Int
	generation int
	revision   int
	voided     bool
	reason     VoidReason
}

type world struct {
	limits   Limits
	now      int64
	assets   map[string]*mAsset
	spans    []*mSpan
	counters []*mCounter
	readings []*mReading
	sequence uint64
}

func newWorld(limits Limits, now int64) *world {
	return &world{limits: limits, now: now, assets: map[string]*mAsset{}}
}

func (w *world) touch(asset string) {
	w.sequence++
	w.assets[asset].change = w.sequence
}

func (w *world) span(id string) *mSpan {
	for _, held := range w.spans {
		if held.id == id {
			return held
		}
	}
	return nil
}

func (w *world) counter(id string) *mCounter {
	for _, held := range w.counters {
		if held.id == id {
			return held
		}
	}
	return nil
}

func (w *world) reading(counter string, at int64) *mReading {
	for _, held := range w.readings {
		if held.counter == counter && held.at == at {
			return held
		}
	}
	return nil
}

func (w *world) live(counter string) []*mReading {
	var out []*mReading
	for _, held := range w.readings {
		if held.counter == counter && !held.voided {
			out = append(out, held)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].at < out[b].at })
	return out
}

func within(created, deleted, from, to int64) bool {
	return from >= created && (deleted < 0 || to <= deleted)
}

func (w *world) putAsset(id string, kind AssetKind, created, deleted int64, revision int) error {
	if (kind != VM && kind != Host) || revision < 1 || (deleted >= 0 && deleted <= created) {
		return ErrInvalid
	}
	held := w.assets[id]
	if held != nil && held.revision == revision {
		if held.kind != kind || held.created != created || held.deleted != deleted {
			return ErrConflict
		}
		return nil
	}
	if held != nil && revision < held.revision {
		return ErrRejected
	}
	if created > w.now || deleted > w.now {
		return ErrRejected
	}
	if held != nil {
		for _, span := range w.spans {
			if span.asset == id && !span.voided && !within(created, deleted, span.from, span.to) {
				return ErrRejected
			}
		}
		for _, counter := range w.counters {
			if counter.asset != id || counter.voided {
				continue
			}
			for _, reading := range w.live(counter.id) {
				if !within(created, deleted, reading.at, reading.at) {
					return ErrRejected
				}
			}
		}
		held.kind, held.created, held.deleted, held.revision = kind, created, deleted, revision
		w.touch(id)
		return nil
	}
	if len(w.assets) >= w.limits.Assets {
		return ErrLimit
	}
	w.assets[id] = &mAsset{kind: kind, created: created, deleted: deleted, revision: revision}
	w.touch(id)
	return nil
}

var modelUnits = map[string]int64{"B": 1, "kB": 1000, "KiB": 1024}

func modelValue(text, unit string) (*big.Int, bool) {
	if text == "" && unit == "" {
		return nil, true
	}
	factor, known := modelUnits[unit]
	var number int64
	if _, err := fmt.Sscanf(text, "%d", &number); err != nil || !known || fmt.Sprint(number) != text || number < 0 {
		return nil, false
	}
	return big.NewInt(number * factor), true
}

func (w *world) putSpan(spec Span, from, to int64) (State, error) {
	if to <= from {
		return "", ErrInvalid
	}
	made := &mSpan{id: spec.ID, asset: spec.Asset, from: from, to: to, organization: spec.Organization}
	var formed [3]bool
	made.values[0], formed[0] = modelValue(spec.CPUs, map[bool]string{true: "", false: "B"}[spec.CPUs == ""])
	made.values[1], formed[1] = modelValue(spec.Memory.Value, spec.Memory.Unit)
	made.values[2], formed[2] = modelValue(spec.Disk.Value, spec.Disk.Unit)
	if !formed[0] || !formed[1] || !formed[2] {
		return "", ErrInvalid
	}
	made.print = fmt.Sprintf("%s|%d|%d|%s|%+v|%+v|%s", spec.Asset, from, to, spec.CPUs, spec.Memory, spec.Disk, spec.Organization)
	owner := w.assets[spec.Asset]
	if owner == nil {
		return "", ErrNotFound
	}
	if earlier := w.span(spec.ID); earlier != nil {
		if earlier.print != made.print {
			return "", ErrConflict
		}
		return map[bool]State{true: Voided, false: Live}[earlier.voided], nil
	}
	if !within(owner.created, owner.deleted, from, to) || to > w.now {
		return "", ErrRejected
	}
	count := 0
	for _, other := range w.spans {
		if other.asset != spec.Asset {
			continue
		}
		count++
		if !other.voided && from < other.to && other.from < to {
			return "", ErrRejected
		}
	}
	if count >= w.limits.Spans {
		return "", ErrLimit
	}
	w.spans = append(w.spans, made)
	w.touch(spec.Asset)
	return Live, nil
}

func (w *world) voidSpan(id string, reason VoidReason) (State, error) {
	if !reason.known() {
		return "", ErrInvalid
	}
	held := w.span(id)
	switch {
	case held == nil:
		return "", ErrNotFound
	case held.voided && held.reason != reason:
		return "", ErrConflict
	case held.voided:
		return Voided, nil
	}
	held.voided, held.reason = true, reason
	w.touch(held.asset)
	return Voided, nil
}

func (w *world) putCounter(spec Counter) (State, error) {
	factor := map[string]int64{"Wh": 1, "kWh": 1000}[spec.Unit]
	if factor == 0 || spec.MaxGap < 1 || spec.MaxGap > w.limits.MaxGap {
		return "", ErrInvalid
	}
	if w.assets[spec.Asset] == nil {
		return "", ErrNotFound
	}
	print := fmt.Sprintf("%+v", spec)
	if earlier := w.counter(spec.ID); earlier != nil {
		if earlier.print != print {
			return "", ErrConflict
		}
		return map[bool]State{true: Voided, false: Live}[earlier.voided], nil
	}
	count := 0
	for _, other := range w.counters {
		if other.asset == spec.Asset {
			count++
		}
	}
	if count >= w.limits.Counters {
		return "", ErrLimit
	}
	w.counters = append(w.counters, &mCounter{id: spec.ID, asset: spec.Asset, print: print, gap: spec.MaxGap, factor: factor})
	w.touch(spec.Asset)
	return Live, nil
}

func (w *world) voidCounter(id string, reason VoidReason) (State, error) {
	if !reason.known() {
		return "", ErrInvalid
	}
	held := w.counter(id)
	switch {
	case held == nil:
		return "", ErrNotFound
	case held.voided && held.reason != reason:
		return "", ErrConflict
	case held.voided:
		return Voided, nil
	}
	held.voided, held.reason = true, reason
	w.touch(held.asset)
	return Voided, nil
}

func modelMicro(text string, factor int64) (*big.Int, bool) {
	value, read := new(big.Rat).SetString(text)
	if !read || strings.ContainsAny(text, "eE-+/ ") || strings.HasSuffix(text, ".") || strings.HasPrefix(text, ".") {
		return nil, false
	}
	value.Mul(value, big.NewRat(1000000*factor, 1))
	if !value.IsInt() {
		return nil, false
	}
	return value.Num(), true
}

type mInput struct {
	at         int64
	text       string
	generation int
	revision   int
}

func (w *world) putReadings(id string, batch []mInput) ([]State, error) {
	if len(batch) == 0 {
		return nil, ErrInvalid
	}
	seen := map[int64]bool{}
	for _, input := range batch {
		if _, formed := modelMicro(input.text, 1); !formed || seen[input.at] || input.generation < 1 || input.revision < 1 {
			return nil, ErrInvalid
		}
		seen[input.at] = true
	}
	meter := w.counter(id)
	if meter == nil {
		return nil, ErrNotFound
	}
	if len(batch) > w.limits.Batch {
		return nil, ErrLimit
	}
	owner := w.assets[meter.asset]
	states := make([]State, len(batch))
	var fresh []mInput
	stale, added := false, 0
	for index, input := range batch {
		earlier := w.reading(id, input.at)
		states[index] = Live
		switch {
		case earlier != nil && earlier.revision == input.revision && (earlier.text != input.text || earlier.generation != input.generation):
			return nil, ErrConflict
		case earlier != nil && earlier.revision == input.revision:
			states[index] = map[bool]State{true: Voided, false: Live}[earlier.voided]
			continue
		case earlier != nil && input.revision < earlier.revision:
			stale = true
		case earlier == nil:
			added++
		}
		fresh = append(fresh, input)
	}
	if len(fresh) == 0 {
		return states, nil
	}
	if stale || meter.voided {
		return nil, ErrRejected
	}
	after := map[int64]int{}
	for _, held := range w.live(id) {
		after[held.at] = held.generation
	}
	for _, input := range fresh {
		if !within(owner.created, owner.deleted, input.at, input.at) || input.at > w.now {
			return nil, ErrRejected
		}
		after[input.at] = input.generation
	}
	times := make([]int64, 0, len(after))
	for at := range after {
		times = append(times, at)
	}
	sort.Slice(times, func(a, b int) bool { return times[a] < times[b] })
	for index := 1; index < len(times); index++ {
		if after[times[index]] < after[times[index-1]] {
			return nil, ErrRejected
		}
	}
	stored := 0
	for _, held := range w.readings {
		if held.counter == id {
			stored++
		}
	}
	if stored+added > w.limits.Readings {
		return nil, ErrLimit
	}
	for _, input := range fresh {
		micro, _ := modelMicro(input.text, meter.factor)
		made := &mReading{counter: id, at: input.at, text: input.text, micro: micro, generation: input.generation, revision: input.revision}
		if earlier := w.reading(id, input.at); earlier != nil {
			*earlier = *made
			continue
		}
		w.readings = append(w.readings, made)
	}
	w.touch(meter.asset)
	return states, nil
}

func (w *world) voidReading(id string, at int64, revision int, reason VoidReason) (State, error) {
	if revision < 1 || !reason.known() {
		return "", ErrInvalid
	}
	meter := w.counter(id)
	if meter == nil || w.reading(id, at) == nil {
		return "", ErrNotFound
	}
	held := w.reading(id, at)
	if held.voided && held.revision == revision {
		if held.reason != reason {
			return "", ErrConflict
		}
		return Voided, nil
	}
	if meter.voided || held.revision != revision {
		return "", ErrRejected
	}
	held.voided, held.reason = true, reason
	w.touch(meter.asset)
	return Voided, nil
}

type mSegment struct {
	from, to int64
	kind     string
	energy   *big.Int
	meter    string
}

func (w *world) segments(asset string) (out []mSegment, meters []string) {
	for _, meter := range w.counters {
		if meter.asset != asset || meter.voided {
			continue
		}
		meters = append(meters, meter.id)
		live := w.live(meter.id)
		for index := 1; index < len(live); index++ {
			a, b := live[index-1], live[index]
			part := mSegment{from: a.at, to: b.at, meter: meter.id, kind: "paired", energy: new(big.Int).Sub(b.micro, a.micro)}
			switch {
			case a.generation != b.generation:
				part.kind = CounterReset
			case part.energy.Sign() < 0:
				part.kind = CounterDecrease
			case b.at-a.at > meter.gap:
				part.kind = CounterGap
			}
			out = append(out, part)
		}
	}
	sort.Strings(meters)
	return out, meters
}

func microText(value *big.Int) string {
	text := new(big.Rat).SetFrac(value, big.NewInt(1000000)).FloatString(6)
	return strings.TrimRight(strings.TrimRight(text, "0"), ".")
}

func (w *world) usage(asset string, from, to int64) (Result, bool) {
	owner := w.assets[asset]
	start, end := from, to
	if owner.created > start {
		start = owner.created
	}
	if owner.deleted >= 0 && owner.deleted < end {
		end = owner.deleted
	}
	out := Result{Asset: asset, Sequence: w.sequence, Change: owner.change, Configuration: Configuration{Result: NotApplicable}, Energy: Energy{Result: NotApplicable}}
	if end <= start {
		return out, false
	}
	out.Window = end - start
	type sums struct {
		seconds  int64
		values   [3]*big.Int
		unstated [3]int64
	}
	entries := map[string]*sums{}
	var missing [3]bool
	configuration := Configuration{}
	segments, meters := w.segments(asset)
	energy := Energy{}
	read := int64(0)
	for second := start; second < end; second++ {
		for _, span := range w.spans {
			if span.asset != asset || span.voided || second < span.from || second >= span.to {
				continue
			}
			configuration.Covered++
			entry := entries[span.organization]
			if entry == nil {
				entry = &sums{}
				entries[span.organization] = entry
			}
			entry.seconds++
			for index, value := range span.values {
				if value == nil {
					entry.unstated[index]++
					missing[index] = true
					continue
				}
				if entry.values[index] == nil {
					entry.values[index] = new(big.Int)
				}
				entry.values[index].Add(entry.values[index], value)
			}
		}
		inAny, inPaired := false, false
		for _, part := range segments {
			if second >= part.from && second < part.to {
				inAny = true
				inPaired = inPaired || (part.kind == "paired" && part.from >= start && part.to <= end)
			}
		}
		if inAny {
			read++
		}
		if inPaired {
			energy.Covered++
		}
	}
	configuration.Uncovered = out.Window - configuration.Covered
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		held := entries[name]
		entry := Entry{Organization: name, Seconds: held.seconds}
		for index, target := range []*Dimension{&entry.CPU, &entry.Memory, &entry.Disk} {
			target.NotStatedSeconds = held.unstated[index]
			if held.values[index] != nil {
				target.Integral = held.values[index].String()
			}
		}
		if name == "" {
			configuration.None = &entry
		} else {
			configuration.Entries = append(configuration.Entries, entry)
		}
	}
	if configuration.Uncovered > 0 {
		configuration.Reasons = append(configuration.Reasons, NotCovered)
	}
	for index, reason := range []string{CPUNotStated, MemoryNotStated, DiskNotStated} {
		if missing[index] {
			configuration.Reasons = append(configuration.Reasons, reason)
		}
	}
	configuration.Result = map[bool]string{true: Complete, false: Incomplete}[len(configuration.Reasons) == 0]
	out.Configuration = configuration
	energy.Uncovered = out.Window - energy.Covered
	energy.Result = Incomplete
	if len(meters) == 0 {
		energy.Covered, energy.Uncovered = 0, out.Window
		energy.Reasons = []string{NoCounter}
		out.Energy = energy
		return out, false
	}
	found := map[string]bool{NotRead: read < out.Window}
	perMeter := map[string]*CounterEnergy{}
	sumsOf := map[string]*big.Int{}
	total, any := new(big.Int), false
	var counting []mSegment
	for _, part := range segments {
		if part.to <= start || part.from >= end {
			continue
		}
		if part.kind != "paired" {
			found[part.kind] = true
			continue
		}
		counting = append(counting, part)
		if part.from < start || part.to > end {
			found[BoundaryNotRead] = true
			continue
		}
		if sumsOf[part.meter] == nil {
			sumsOf[part.meter] = new(big.Int)
			perMeter[part.meter] = &CounterEnergy{Counter: part.meter}
		}
		sumsOf[part.meter].Add(sumsOf[part.meter], part.energy)
		perMeter[part.meter].Seconds += part.to - part.from
		total.Add(total, part.energy)
		any = true
	}
	for index, a := range counting {
		for _, b := range counting[:index] {
			if a.meter != b.meter && a.from < b.to && b.from < a.to {
				found[CountersOverlap] = true
			}
		}
	}
	for _, id := range meters {
		entry := CounterEnergy{Counter: id}
		if perMeter[id] != nil {
			entry = *perMeter[id]
			entry.Energy = microText(sumsOf[id])
		}
		energy.Counters = append(energy.Counters, entry)
	}
	for _, reason := range []string{CountersOverlap, CounterReset, CounterDecrease, CounterGap, BoundaryNotRead, NotRead} {
		if found[reason] {
			energy.Reasons = append(energy.Reasons, reason)
		}
	}
	if any && !found[CountersOverlap] {
		energy.Total = microText(total)
	}
	if len(energy.Reasons) == 0 {
		energy.Result = Complete
	}
	out.Energy = energy
	return out, true
}
