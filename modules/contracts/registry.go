package contracts

import (
	"reflect"
	"sync"
	"time"
)

type key struct {
	workspace string
	contract  string
}

type held struct {
	event     Event
	added     uint64
	withdrawn uint64
}

type gathered struct {
	sequence uint64
	write    string
	evidence Evidence
	resolved uint64
}

type contract struct {
	sequence uint64
	events   []held
	evidence []gathered
	log      []Record
	writes   map[string]int
	state    SettlementState
	entered  uint64
	batch    uint64
}

type Registry struct {
	mu        sync.Mutex
	built     bool
	limits    Limits
	clock     func() time.Time
	contracts map[key]*contract
	perSpace  map[string]int
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, clock: clock, contracts: map[key]*contract{}, perSpace: map[string]int{}}, nil
}

func (c *contract) holding(at uint64) []Event {
	events := []Event{}
	if c == nil {
		return events
	}
	for _, kept := range c.events {
		if kept.added <= at && (kept.withdrawn == 0 || kept.withdrawn > at) {
			events = append(events, kept.event)
		}
	}
	return ordered(events)
}

func content(record Record) Record {
	record.Sequence, record.At, record.State = 0, time.Time{}, ""
	return record
}

func copyRecord(record Record) Record {
	record.Withdrawals = append([]string(nil), record.Withdrawals...)
	record.Events = append([]Event(nil), record.Events...)
	record.Evidence.Resolves = append([]string(nil), record.Evidence.Resolves...)
	return record
}

func (r *Registry) begin(workspace, id string, expected uint64, record Record) (*contract, Outcome, bool, error) {
	if r == nil || !r.built {
		return nil, Outcome{}, false, ErrInvalid
	}
	if !validID(workspace) || !validID(id) || !validID(record.Write) {
		return nil, Outcome{}, false, ErrInvalid
	}
	found := r.contracts[key{workspace, id}]
	if found != nil {
		if index, stored := found.writes[record.Write]; stored {
			if reflect.DeepEqual(content(found.log[index]), content(copyRecord(record))) {
				return found, Outcome{Sequence: found.log[index].Sequence, Replay: true}, true, nil
			}
			return nil, Outcome{}, false, ErrConflict
		}
	}
	current := uint64(0)
	if found != nil {
		current = found.sequence
	}
	if expected != current {
		return nil, Outcome{}, false, ErrConflict
	}
	return found, Outcome{}, false, nil
}

func (r *Registry) room(found *contract, reserved bool) bool {
	used := 0
	if found != nil {
		used = len(found.log)
	}
	if reserved {
		return used < r.limits.Writes+r.limits.Reserve
	}
	return used < r.limits.Writes
}

func (found *contract) commit(record Record, state SettlementState, at time.Time) Outcome {
	found.sequence++
	record = copyRecord(record)
	record.Sequence, record.At, record.State = found.sequence, at, state
	found.state = state
	found.writes[record.Write] = len(found.log)
	found.log = append(found.log, record)
	return Outcome{Sequence: found.sequence}
}

func stopping(withdrawals []string, events []Event) bool {
	if len(withdrawals) > 0 || len(events) == 0 {
		return false
	}
	for _, event := range events {
		if event.Kind != End && event.Kind != Suspend {
			return false
		}
	}
	return true
}

func knownCancel(reason CancelReason) bool {
	return reason == RecordedInError || reason == ContractContinues
}

func (r *Registry) Amend(workspace, id string, expected uint64, write string, withdrawals []string, events []Event, cancel CancelReason) (Outcome, error) {
	record := Record{Write: write, Kind: WriteBatch, Withdrawals: withdrawals, Events: events, Cancel: cancel}
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, outcome, replay, err := r.begin(workspace, id, expected, record)
	if err != nil || replay {
		return outcome, err
	}
	now := r.clock()
	if len(withdrawals)+len(events) == 0 || (cancel != "" && !knownCancel(cancel)) {
		return Outcome{}, ErrInvalid
	}
	if len(withdrawals)+len(events) > r.limits.Batch {
		return Outcome{}, ErrLimit
	}
	added := map[string]bool{}
	for _, event := range events {
		if !event.valid() {
			return Outcome{}, ErrInvalid
		}
		if added[event.ID] {
			return Outcome{}, ErrConflict
		}
		added[event.ID] = true
	}
	removed := map[string]bool{}
	for _, name := range withdrawals {
		if !validID(name) {
			return Outcome{}, ErrInvalid
		}
		if removed[name] {
			return Outcome{}, ErrRejected
		}
		removed[name] = true
	}
	state := NoSettlement
	if found != nil {
		state = found.state
		for _, kept := range found.events {
			if added[kept.event.ID] {
				return Outcome{}, ErrConflict
			}
		}
	}
	if state == Settled {
		return Outcome{}, ErrState
	}
	remaining := []Event{}
	matched := 0
	for _, event := range found.holding(expected) {
		if removed[event.ID] {
			matched++
			continue
		}
		remaining = append(remaining, event)
	}
	if matched != len(removed) {
		return Outcome{}, ErrRejected
	}
	after := ordered(append(remaining, events...))
	if !validLifecycle(after) {
		return Outcome{}, ErrRejected
	}
	_, over := ended(after)
	final := over || len(after) == 0
	if (cancel != "") != (state == PendingFinal && !final) {
		return Outcome{}, ErrRejected
	}
	if !r.room(found, stopping(withdrawals, events)) {
		return Outcome{}, ErrLimit
	}
	if found == nil {
		if r.perSpace[workspace] >= r.limits.Contracts {
			return Outcome{}, ErrLimit
		}
		found = &contract{writes: map[string]int{}, state: NoSettlement}
		r.contracts[key{workspace, id}] = found
		r.perSpace[workspace]++
	}
	next := found.sequence + 1
	for index := range found.events {
		if found.events[index].withdrawn == 0 && removed[found.events[index].event.ID] {
			found.events[index].withdrawn = next
		}
	}
	for _, event := range events {
		found.events = append(found.events, held{event: event, added: next})
	}
	result := NoSettlement
	if final {
		result = PendingFinal
	}
	found.batch = next
	return found.commit(record, result, now), nil
}

func (found *contract) gap(write string) *gathered {
	for index := range found.evidence {
		if found.evidence[index].write == write && found.evidence[index].evidence.Kind == Missing {
			return &found.evidence[index]
		}
	}
	return nil
}

func (r *Registry) RecordEvidence(workspace, id string, expected uint64, write string, evidence Evidence) (Outcome, error) {
	record := Record{Write: write, Kind: WriteEvidence, Evidence: evidence}
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, outcome, replay, err := r.begin(workspace, id, expected, record)
	if err != nil || replay {
		return outcome, err
	}
	now := r.clock()
	if !evidence.valid() {
		return Outcome{}, ErrInvalid
	}
	if found == nil {
		return Outcome{}, ErrNotFound
	}
	if evidence.Kind == Secured && found.state != PendingFinal {
		return Outcome{}, ErrState
	}
	for _, name := range evidence.Resolves {
		if open := found.gap(name); open == nil || open.resolved != 0 {
			return Outcome{}, ErrRejected
		}
	}
	if !r.room(found, evidence.Kind == Missing) {
		return Outcome{}, ErrLimit
	}
	next := found.sequence + 1
	for _, name := range evidence.Resolves {
		found.gap(name).resolved = next
	}
	kept := evidence
	kept.Resolves = append([]string(nil), evidence.Resolves...)
	found.evidence = append(found.evidence, gathered{sequence: next, write: write, evidence: kept})
	return found.commit(record, found.state, now), nil
}

func (found *contract) open() bool {
	for _, kept := range found.evidence {
		if kept.evidence.Kind == Missing && kept.resolved == 0 {
			return true
		}
	}
	return false
}

func (r *Registry) Settle(workspace, id string, expected uint64, write string, asOf Day) (Outcome, error) {
	record := Record{Write: write, Kind: WriteSettle, AsOf: asOf}
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, outcome, replay, err := r.begin(workspace, id, expected, record)
	if err != nil || replay {
		return outcome, err
	}
	now := r.clock()
	if !asOf.Valid() {
		return Outcome{}, ErrInvalid
	}
	if found == nil {
		return Outcome{}, ErrNotFound
	}
	if found.state != PendingFinal {
		return Outcome{}, ErrState
	}
	if found.open() {
		return Outcome{}, ErrRejected
	}
	if last, over := ended(found.holding(found.sequence)); over {
		if len(found.evidence) == 0 {
			return Outcome{}, ErrRejected
		}
		latest := found.evidence[len(found.evidence)-1]
		if latest.evidence.Kind != Secured || latest.sequence <= found.entered || latest.sequence <= found.batch ||
			latest.evidence.CoveredUntil.Before(last) || asOf.Before(last) {
			return Outcome{}, ErrRejected
		}
	}
	if !r.room(found, false) {
		return Outcome{}, ErrLimit
	}
	return found.commit(record, Settled, now), nil
}

func (r *Registry) Reopen(workspace, id string, expected uint64, write string, reason ReopenReason) (Outcome, error) {
	record := Record{Write: write, Kind: WriteReopen, Reopen: reason}
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, outcome, replay, err := r.begin(workspace, id, expected, record)
	if err != nil || replay {
		return outcome, err
	}
	now := r.clock()
	if reason != LateSource && reason != LateCredit && reason != Correcting {
		return Outcome{}, ErrInvalid
	}
	if found == nil {
		return Outcome{}, ErrNotFound
	}
	if found.state != Settled {
		return Outcome{}, ErrState
	}
	if !r.room(found, true) {
		return Outcome{}, ErrLimit
	}
	found.entered = found.sequence + 1
	return found.commit(record, PendingFinal, now), nil
}

func (r *Registry) at(workspace, id string, at uint64) (*contract, error) {
	if r == nil || !r.built || !validID(workspace) || !validID(id) {
		return nil, ErrInvalid
	}
	found := r.contracts[key{workspace, id}]
	if found == nil || at > found.sequence {
		return nil, ErrNotFound
	}
	return found, nil
}

func (r *Registry) Sequence(workspace, id string) (uint64, error) {
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, err := r.at(workspace, id, 0)
	if err != nil {
		return 0, err
	}
	return found.sequence, nil
}

func (r *Registry) Lifecycle(workspace, id string, at uint64) ([]Event, error) {
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, err := r.at(workspace, id, at)
	if err != nil {
		return nil, err
	}
	return found.holding(at), nil
}

func (r *Registry) StateOn(workspace, id string, at uint64, day Day) (Position, error) {
	if !day.Valid() {
		return Position{}, ErrInvalid
	}
	events, err := r.Lifecycle(workspace, id, at)
	if err != nil {
		return Position{}, err
	}
	return positionOn(events, day), nil
}

func (r *Registry) SegmentsAt(workspace, id string, at uint64, period Interval) ([]Segment, error) {
	events, err := r.Lifecycle(workspace, id, at)
	if err != nil {
		return nil, err
	}
	return Segments(events, period)
}

func (r *Registry) Settlement(workspace, id string, at uint64) (Settlement, error) {
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, err := r.at(workspace, id, at)
	if err != nil {
		return Settlement{}, err
	}
	view := Settlement{State: NoSettlement, Evidence: []EvidenceRecord{}}
	for _, record := range found.log {
		if record.Sequence > at {
			break
		}
		view.State = record.State
		if record.Kind == WriteReopen {
			view.Reopenings++
		}
	}
	for _, kept := range found.evidence {
		if kept.sequence > at {
			break
		}
		evidence := kept.evidence
		evidence.Resolves = append([]string(nil), evidence.Resolves...)
		view.Evidence = append(view.Evidence, EvidenceRecord{Sequence: kept.sequence, Write: kept.write, Evidence: evidence,
			Resolved: kept.resolved != 0 && kept.resolved <= at})
	}
	return view, nil
}

func (r *Registry) Records(workspace, id string) ([]Record, error) {
	if r != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	found, err := r.at(workspace, id, 0)
	if err != nil {
		return nil, err
	}
	records := make([]Record, len(found.log))
	for index, record := range found.log {
		records[index] = copyRecord(record)
	}
	return records, nil
}
