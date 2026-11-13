package costpool

import (
	"sort"
	"time"
)

func judge(held *version) Verdict {
	out := Verdict{Result: NotCurrent}
	switch {
	case held.state == Withdrawn:
		out.Reasons = []string{IsWithdrawn}
	case !held.recorded:
		out.Reasons = []string{NoApprovalRecorded}
	case held.superseded:
		out.Reasons = []string{IsSuperseded}
	default:
		out.Result = Current
	}
	return out
}

func (r *Registry) Standing(workspaceID, name string, number int) (Verdict, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 {
		return Verdict{}, ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return Verdict{}, ErrNotFound
	}
	return judge(held), nil
}

func (r *Registry) Counting(workspaceID, name string, from, to time.Time) (Counted, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || !r.period(from, to) {
		return Counted{}, ErrInvalid
	}
	home := r.workspaces[workspaceID][name]
	if home == nil {
		return Counted{}, ErrNotFound
	}
	for number, held := range home.versions {
		if held.current() && held.spec.PeriodStart.Equal(from) && held.spec.PeriodEnd.Equal(to) {
			return Counted{Version: number, Sequence: home.sequence}, nil
		}
	}
	return Counted{Sequence: home.sequence}, ErrNotFound
}

func (r *Registry) blockers(held *version) *Blockers {
	figures, fits := r.total(held)
	out := &Blockers{Summed: figures.summed, WithoutAmount: figures.unknown, AssetsWithoutComponent: held.without(), BasisSet: held.basis != nil, Multiple: MultipleNotDetermined}
	if held.spec.Supersedes != 0 {
		out.NamedSuperseded = held.home.versions[held.spec.Supersedes].superseded
	}
	if fits && figures.unknown == 0 {
		out.Multiple = MultipleYes
		if figures.total.Scale() > r.quantum(held).Scale() {
			out.Multiple = MultipleNo
		}
	}
	overlapping := r.overlapping(held)
	out.Overlapping.Total = len(overlapping)
	if len(overlapping) > r.limits.Listed {
		overlapping = overlapping[:r.limits.Listed]
	}
	out.Overlapping.Items = overlapping
	return out
}

func (r *Registry) Version(workspaceID, name string, number int) (View, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 {
		return View{}, ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return View{}, ErrNotFound
	}
	view := View{Spec: held.spec, State: held.state, Recorded: held.recorded, Superseded: held.superseded, SupersededBy: held.supersededBy, Approval: held.approval, WithdrawReason: held.reason,
		Opened: held.opened, Changed: held.changed, Assets: len(held.assets), Components: len(held.components)}
	if held.basis != nil {
		shown := held.basis.spec
		shown.Targets = nil
		view.Basis, view.Targets = &shown, len(held.basis.spec.Targets)
	}
	if held.state == Draft {
		view.Blockers = r.blockers(held)
	}
	if held.figures != nil {
		copied := *held.figures
		copied.ByKind, copied.ByGrade, copied.ByBearer = map[Kind]string{}, map[Grade]string{}, map[Bearer]string{}
		for kind, sum := range held.figures.ByKind {
			copied.ByKind[kind] = sum
		}
		for grade, sum := range held.figures.ByGrade {
			copied.ByGrade[grade] = sum
		}
		for class, sum := range held.figures.ByBearer {
			copied.ByBearer[class] = sum
		}
		view.Figures = &copied
	}
	return view, nil
}

func (r *Registry) page(workspaceID, name string, number int, after string, limit int) (*version, error) {
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || (after != "" && !validID(after)) || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return nil, ErrNotFound
	}
	return held, nil
}

func next(ids []string, after string, limit int) []string {
	sort.Strings(ids)
	start := sort.SearchStrings(ids, after)
	if after != "" && start < len(ids) && ids[start] == after {
		start++
	}
	ids = ids[start:]
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids
}

func (r *Registry) Assets(workspaceID, name string, number int, after string, limit int) ([]Asset, error) {
	defer r.leave()
	held, err := r.page(workspaceID, name, number, after, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(held.assets))
	for id := range held.assets {
		ids = append(ids, id)
	}
	var out []Asset
	for _, id := range next(ids, after, limit) {
		out = append(out, held.assets[id].spec)
	}
	return out, nil
}

func (r *Registry) Components(workspaceID, name string, number int, after string, limit int) ([]Component, error) {
	defer r.leave()
	held, err := r.page(workspaceID, name, number, after, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(held.components))
	for id := range held.components {
		ids = append(ids, id)
	}
	var out []Component
	for _, id := range next(ids, after, limit) {
		shown := held.components[id].spec
		if shown.Power != nil {
			model := *shown.Power
			shown.Power = &model
		}
		out = append(out, shown)
	}
	return out, nil
}

func (r *Registry) Targets(workspaceID, name string, number int, after string, limit int) ([]Target, error) {
	defer r.leave()
	held, err := r.page(workspaceID, name, number, after, limit)
	if err != nil {
		return nil, err
	}
	if held.basis == nil {
		return nil, ErrRejected
	}
	targets := held.basis.spec.Targets
	start := sort.Search(len(targets), func(index int) bool { return targets[index].ID > after })
	end := len(targets)
	if end-start > limit {
		end = start + limit
	}
	return append([]Target(nil), targets[start:end]...), nil
}

func (r *Registry) AssetSums(workspaceID, name string, number int, after string, limit int) ([]AssetSum, error) {
	defer r.leave()
	held, err := r.page(workspaceID, name, number, after, limit)
	if err != nil {
		return nil, err
	}
	if held.figures == nil {
		return nil, ErrRejected
	}
	start := sort.Search(len(held.assetSums), func(index int) bool { return held.assetSums[index].Asset > after })
	end := len(held.assetSums)
	if end-start > limit {
		end = start + limit
	}
	return append([]AssetSum(nil), held.assetSums[start:end]...), nil
}

func (r *Registry) Shares(workspaceID, name string, number int, after string, limit int) ([]Share, error) {
	defer r.leave()
	held, err := r.page(workspaceID, name, number, after, limit)
	if err != nil {
		return nil, err
	}
	if held.figures == nil {
		return nil, ErrRejected
	}
	start := sort.Search(len(held.shares), func(index int) bool { return held.shares[index].Target > after })
	end := len(held.shares)
	if end-start > limit {
		end = start + limit
	}
	return append([]Share(nil), held.shares[start:end]...), nil
}

func (r *Registry) Versions(workspaceID, name string, after, limit int) ([]Row, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || after < 0 || limit < 1 || limit > r.limits.Page {
		return nil, ErrInvalid
	}
	home := r.workspaces[workspaceID][name]
	if home == nil {
		return nil, nil
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
	var out []Row
	for _, number := range numbers {
		held := home.versions[number]
		out = append(out, Row{Version: number, PeriodStart: held.spec.PeriodStart, PeriodEnd: held.spec.PeriodEnd, State: held.state, Superseded: held.superseded})
	}
	return out, nil
}
