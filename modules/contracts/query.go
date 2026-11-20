package contracts

import (
	"errors"
	"math"
	"sort"
	"time"
)

var ErrQueryUnavailable = errors.New("contract query source is unavailable")

type positionPoint struct {
	from     Day
	position Position
}

func indexPositions(events []Event) []positionPoint {
	points := make([]positionPoint, len(events))
	position := Position{State: NotStarted}
	for i, event := range events {
		position, _ = step(position, event)
		points[i] = positionPoint{event.From, position}
	}
	return points
}

func (r *Registry) CurrentPosition(workspace, id string, on Day) (Position, uint64, error) {
	if r == nil || !on.Valid() {
		return Position{}, 0, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	found, err := r.at(workspace, id, 0)
	if err != nil {
		return Position{}, 0, err
	}
	at := sort.Search(len(found.positions), func(i int) bool { return on.Before(found.positions[i].from) })
	if at == 0 {
		return Position{State: NotStarted}, found.sequence, nil
	}
	return found.positions[at-1].position, found.sequence, nil
}

func (r *Registry) QueryVersion(workspace string) (uint64, error) {
	if r == nil || !validID(workspace) {
		return 0, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.built {
		return 0, ErrInvalid
	}
	version := r.queryVersions[workspace]
	if version == math.MaxUint64 {
		return 0, ErrQueryUnavailable
	}
	return version, nil
}

func (r *Registry) commit(workspace string, found *contract, record Record, state SettlementState, at time.Time) Outcome {
	result := found.commit(record, state, at)
	if r.queryVersions[workspace] < math.MaxUint64 {
		r.queryVersions[workspace]++
	}
	return result
}
