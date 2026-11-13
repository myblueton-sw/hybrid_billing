package sharedpool

import (
	"sort"
	"time"
)

func (r *Registry) Pool(workspaceID, name string, number int) (Spec, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 {
		return Spec{}, ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return Spec{}, ErrNotFound
	}
	return held.spec, nil
}

func (r *Registry) Versions(workspaceID, name string, after, limit int) ([]VersionRow, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	home := r.workspaces[workspaceID][name]
	if home == nil {
		return nil, ErrNotFound
	}
	numbers := make([]int, 0, len(home.versions))
	for number := range home.versions {
		if number > after {
			numbers = append(numbers, number)
		}
	}
	sort.Ints(numbers)
	if len(numbers) > limit {
		numbers = numbers[:limit]
	}
	var out []VersionRow
	for _, number := range numbers {
		spec := home.versions[number].spec
		out = append(out, VersionRow{Version: number, Currency: spec.Currency, PeriodStart: spec.PeriodStart, PeriodEnd: spec.PeriodEnd})
	}
	return out, nil
}

func (r *Registry) Run(workspaceID, name string, number int, id string) (RunView, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) {
		return RunView{}, ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return RunView{}, ErrNotFound
	}
	view := RunView{ID: held.spec.ID, Purpose: held.spec.Purpose, State: held.state, Declared: held.spec.Chunks, Staged: len(held.chunks), Targets: len(held.targets), NotKnown: held.unknown(), Hold: held.hold,
		WeightSum: held.weightSum, Amount: held.home.spec.Amount, Denominator: held.denominator, Approval: held.approval, WasPublished: held.published, WithdrawReason: held.reason, Started: held.started, Changed: held.changed}
	if held.alternative != nil {
		recorded := *held.alternative
		view.Alternative = &recorded
	}
	if held.published {
		view.ByBearer = map[Bearer]string{}
		for class, sum := range held.byBearer {
			view.ByBearer[class] = sum
		}
		view.Residual, view.ResidualReason = held.residual, held.residualWhy
	}
	return view, nil
}

func (r *Registry) Shares(workspaceID, name string, number int, id, after string, limit int) (SharePage, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return SharePage{}, ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return SharePage{}, ErrNotFound
	}
	if !held.published {
		return SharePage{}, ErrRejected
	}
	out := SharePage{State: held.state, Purpose: held.spec.Purpose, Sequence: held.home.home.sequence}
	first := sort.Search(len(held.shares), func(index int) bool { return held.shares[index].Target > after })
	for _, share := range held.shares[first:] {
		if len(out.Items) == limit {
			out.More = true
			break
		}
		out.Items = append(out.Items, share)
	}
	return out, nil
}

func (r *Registry) Chunk(workspaceID, name string, number int, id string, position int, after string, limit int) (WeightPage, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) || position < 1 || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return WeightPage{}, ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil || held.chunks[position] == nil {
		return WeightPage{}, ErrNotFound
	}
	weights := held.chunks[position].weights
	first := sort.Search(len(weights), func(index int) bool { return weights[index].weight.Target > after })
	var out WeightPage
	for _, entry := range weights[first:] {
		if len(out.Items) == limit {
			out.More = true
			break
		}
		out.Items = append(out.Items, entry.weight)
	}
	return out, nil
}

func (r *Registry) Runs(workspaceID, name string, number int, after string, limit int) ([]RunRow, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	home := r.find(workspaceID, name, number)
	if home == nil {
		return nil, ErrNotFound
	}
	ids := make([]string, 0, len(home.runs))
	for id := range home.runs {
		if id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	var out []RunRow
	for _, id := range ids {
		held := home.runs[id]
		out = append(out, RunRow{ID: id, Purpose: held.spec.Purpose, State: held.state})
	}
	return out, nil
}

func (r *Registry) Chargeback(workspaceID, name string, from, to time.Time) (PublishedChargeback, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !r.period(from, to) {
		return PublishedChargeback{}, ErrInvalid
	}
	home := r.workspaces[workspaceID][name]
	if home == nil {
		return PublishedChargeback{}, ErrNotFound
	}
	for _, held := range home.charged {
		if held.home.spec.PeriodStart.Equal(from) && held.home.spec.PeriodEnd.Equal(to) {
			return PublishedChargeback{Version: held.home.spec.Version, Run: held.spec.ID, Sequence: home.sequence}, nil
		}
	}
	return PublishedChargeback{Sequence: home.sequence}, ErrNotFound
}
