package costpool

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

type span struct{ from, to time.Time }

func spans() []span {
	october, november := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	return []span{{october.Add(-30 * day), october}, {october, november}, {november, november.Add(30 * day)}, {october, october.Add(15 * day)}, {october.Add(10 * day), november.Add(4 * day)}}
}

var (
	workspaceIDs = []string{"ws1", "ws2"}
	poolIDs      = []string{"dc-1", "dc-2"}
	errorKinds   = []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit, ErrSelfCheck}
)

func kindOf(err error) string {
	if err == nil {
		return "ok"
	}
	for _, kind := range errorKinds {
		if errors.Is(err, kind) {
			return kind.Error()[:12]
		}
	}
	return "other"
}

type address struct {
	ws, pool string
	number   int
}

type generator struct {
	t       *testing.T
	rnd     *rand.Rand
	r       *Registry
	w       *world
	limits  Limits
	order   []address
	reached map[string]int
	counted map[string]bool
	gone    map[address]bool
}

func (g *generator) pick(options ...string) string { return options[g.rnd.Intn(len(options))] }

func (g *generator) once(key string, where any) {
	if mark := fmt.Sprint(key, "|", where); !g.counted[mark] {
		g.counted[mark] = true
		g.reached[key]++
	}
}

func (g *generator) same(what string, at address, got, want error) {
	g.t.Helper()
	if got != want {
		g.t.Fatalf("%s: the registry said %v, the model %v", what, got, want)
	}
	if want == nil {
		g.reached[what+"/ok"]++
		return
	}
	g.once(what+"/"+kindOf(want), at)
}

func (g *generator) open(spec Spec) bool {
	state, err := g.r.Open(spec)
	want, wantErr := g.w.open(spec)
	fresh := wantErr == nil && want == Draft && !g.known(address{spec.WorkspaceID, spec.Pool, spec.Version})
	g.same("open", address{spec.WorkspaceID, spec.Pool, spec.Version}, err, wantErr)
	if state != want {
		g.t.Fatalf("open: %v, model %v", state, want)
	}
	if fresh {
		g.order = append(g.order, address{spec.WorkspaceID, spec.Pool, spec.Version})
	}
	return fresh
}

func (g *generator) known(at address) bool {
	for _, other := range g.order {
		if other == at {
			return true
		}
	}
	return false
}

func (g *generator) asset(at address, spec Asset) {
	state, err := g.r.PutAsset(at.ws, at.pool, at.number, spec)
	want, wantErr := g.w.asset(at.ws, at.pool, at.number, spec)
	g.same("asset", at, err, wantErr)
	if state != want {
		g.t.Fatalf("asset: %v, model %v", state, want)
	}
}

func (g *generator) component(at address, spec Component) {
	state, err := g.r.PutComponent(at.ws, at.pool, at.number, spec)
	want, wantErr := g.w.component(at.ws, at.pool, at.number, spec)
	g.same("component", at, err, wantErr)
	if state != want {
		g.t.Fatalf("component %+v: %v, model %v", spec, state, want)
	}
}

func (g *generator) basis(at address, spec Basis) {
	state, err := g.r.SetBasis(at.ws, at.pool, at.number, spec)
	want, wantErr := g.w.setBasis(at.ws, at.pool, at.number, spec)
	g.same("basis", at, err, wantErr)
	if state != want {
		g.t.Fatalf("basis: %v, model %v", state, want)
	}
}

func (g *generator) record(at address, approval Approval) {
	before := g.w.find(at.ws, at.pool, at.number)
	was := before != nil && before.recorded
	state, err := g.r.RecordApproval(at.ws, at.pool, at.number, approval)
	want, wantErr := g.w.record(at.ws, at.pool, at.number, approval)
	g.same("record", at, err, wantErr)
	if state != want {
		g.t.Fatalf("record: %v, model %v", state, want)
	}
	if wantErr == nil && !was {
		g.reached["recorded"]++
		if before.spec.Supersedes != 0 {
			g.reached["supersessions"]++
		}
		g.checkShares(at, before)
	}
}

func (g *generator) withdraw(at address, reason WithdrawReason) {
	g.same("withdraw", at, g.r.Withdraw(at.ws, at.pool, at.number, reason), g.w.withdraw(at.ws, at.pool, at.number, reason))
}

func (g *generator) allShares(at address) []Share {
	var out []Share
	after := ""
	for {
		page, err := g.r.Shares(at.ws, at.pool, at.number, after, g.limits.Page)
		if err != nil {
			g.t.Fatal(err)
		}
		if len(page) == 0 {
			return out
		}
		out = append(out, page...)
		after = page[len(page)-1].Target
	}
}

func (g *generator) checkShares(at address, held *modelVersion) {
	t := g.t
	sums := held.sums()
	shares := g.allShares(at)
	targets := append([]Target(nil), held.basis.Targets...)
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	if len(shares) != len(targets) {
		t.Fatalf("%d shares for %d targets", len(shares), len(targets))
	}
	quantum, capacity, sum := ratOf(held.spec.Quantum), ratOf(held.basis.Capacity), new(big.Rat)
	for index, share := range shares {
		target := targets[index]
		if share.Target != target.ID || share.Bearer != target.Bearer || share.Weight != target.Weight || share.Idle != target.Idle {
			t.Fatalf("share %+v against target %+v", share, target)
		}
		amount := ratOf(share.Amount)
		exact := new(big.Rat).Mul(sums.total, new(big.Rat).Quo(ratOf(target.Weight), capacity))
		if distance := new(big.Rat).Abs(new(big.Rat).Sub(amount, exact)); distance.Cmp(quantum) >= 0 || amount.Sign() < 0 || !multiple(amount, held.spec.Quantum) {
			t.Fatalf("share %s of %s, exact %s at quantum %s", share.Amount, ratText(sums.total), exact.FloatString(8), held.spec.Quantum)
		}
		if ratOf(target.Weight).Sign() == 0 && amount.Sign() != 0 {
			t.Fatalf("a weight of zero has the share %s", share.Amount)
		}
		for other := 0; other < index; other++ {
			order := ratOf(targets[other].Weight).Cmp(ratOf(target.Weight))
			earlier := ratOf(shares[other].Amount)
			difference := new(big.Rat).Sub(earlier, amount)
			if order > 0 && difference.Sign() < 0 || order < 0 && difference.Sign() > 0 || order == 0 && difference.Abs(difference).Cmp(quantum) > 0 {
				t.Fatalf("weights %s and %s have the shares %s and %s", targets[other].Weight, target.Weight, shares[other].Amount, share.Amount)
			}
		}
		sum.Add(sum, amount)
	}
	if sum.Cmp(sums.total) != 0 {
		t.Fatalf("the shares add up to %s, the model's total is %s", ratText(sum), ratText(sums.total))
	}
	if len(shares) > 1 {
		g.reached["shares/several"]++
	}
}

func (g *generator) plan(at address, spec Spec) []func() {
	rnd := g.rnd
	var steps []func()
	serial := 0
	next := func(prefix string) string {
		serial++
		return fmt.Sprintf("%s%d", prefix, serial)
	}
	clean := spec.Supersedes != 0 && rnd.Intn(4) > 0
	amount := func() string {
		if clean {
			return "100"
		}
		switch rnd.Intn(14) {
		case 0:
			return "250.5"
		case 1:
			return "0.0001"
		case 2:
			return ""
		case 3:
			return "0"
		}
		return g.pick("100", "100", "40000", "1000.25")
	}
	if rnd.Intn(2) == 0 {
		id := next("asset")
		method := Kind(g.pick("depreciation", "lease"))
		steps = append(steps, func() { g.asset(at, Asset{ID: id, Kind: Server, Method: method, Acquisition: g.pick("", "40000")}) })
		if rnd.Intn(8) > 0 || clean {
			value := amount()
			steps = append(steps, func() {
				g.component(at, Component{ID: "method-" + id, Kind: method, Amount: value, Asset: id, Evidence: "doc"})
			})
		}
		if rnd.Intn(3) == 0 {
			value := amount()
			steps = append(steps, func() {
				g.component(at, Component{ID: "ops-" + id, Kind: Operations, Amount: value, Asset: id, Evidence: "doc"})
			})
		}
	}
	fee := next("fee")
	value := amount()
	steps = append(steps, func() { g.component(at, Component{ID: fee, Kind: Facility, Amount: value, Evidence: "doc"}) })
	if rnd.Intn(2) == 0 {
		id, value, grade := next("power"), amount(), Grade(g.pick("measured", "estimated", "not_known"))
		model := &Power{Grade: grade, Source: "meter", PUE: "1.4", From: spec.PeriodStart, To: spec.PeriodEnd}
		if grade == GradeNotKnown && rnd.Intn(2) == 0 {
			model.Source, model.PUE = "", ""
		}
		container := ""
		if rnd.Intn(2) == 0 {
			container = fee
		}
		steps = append(steps, func() {
			g.component(at, Component{ID: id, Kind: PowerCost, Amount: value, Evidence: "doc", IncludedIn: container, Power: model})
		})
	}
	if rnd.Intn(3) == 0 {
		id, value := next("lic"), amount()
		steps = append(steps, func() { g.component(at, Component{ID: id, Kind: Licence, Amount: value, Evidence: "doc"}) })
	}
	made := Basis{ID: "basis", Measure: Measure(g.pick("reserved_capacity", "fixed_agreement"))}
	total := new(big.Rat)
	for index, count := 0, 1+rnd.Intn(3)+rnd.Intn(12)/11; index < count; index++ {
		weight := g.pick("100", "100", "250.5", "0", "40000", "0.0001")
		if clean {
			weight = "100"
		}
		bearer := Bearer(g.pick("organization", "organization", "msp", "unattributed"))
		made.Targets = append(made.Targets, Target{ID: fmt.Sprintf("t%d", rnd.Intn(1000)*10+index), Bearer: bearer, Weight: weight, Idle: bearer != Organization && rnd.Intn(3) == 0})
		total.Add(total, ratOf(weight))
	}
	made.Capacity = ratText(total)
	formedAmount[made.Capacity] = true
	if rnd.Intn(15) == 0 && !clean {
		made.Capacity = "100"
	}
	rnd.Shuffle(len(steps), func(i, j int) {
		if rnd.Intn(4) == 0 {
			steps[i], steps[j] = steps[j], steps[i]
		}
	})
	position := rnd.Intn(len(steps) + 1)
	steps = append(steps[:position], append([]func(){func() { g.basis(at, made) }}, steps[position:]...)...)
	approval := Approval{Approver: "approver", Reference: g.pick("ref-1", "ref-1", "ref-2")}
	return append(steps, func() { g.record(at, approval) })
}

func (g *generator) fresh() {
	rnd := g.rnd
	ws, pool := workspaceIDs[rnd.Intn(8)/7], poolIDs[rnd.Intn(4)/3]
	period := spans()[rnd.Intn(3)]
	if rnd.Intn(4) == 0 {
		period = spans()[rnd.Intn(5)]
	}
	spec := Spec{WorkspaceID: ws, Pool: pool, Version: 1 + rnd.Intn(16), Currency: g.pick("KRW", "KRW", "KRW", "USD"), PeriodStart: period.from, PeriodEnd: period.to, Owner: "owner",
		Quantum: g.pick("1", "1", "1", "1", "0.1", "0.01", "0.001", "0.0001")}
	if rnd.Intn(2) == 0 {
		var live []int
		for number := 1; number <= 16; number++ {
			if held := g.w.find(ws, pool, number); held != nil && held.recorded && !held.superseded && number < 16 {
				live = append(live, number)
			}
		}
		if len(live) > 0 {
			named := g.w.find(ws, pool, live[rnd.Intn(len(live))])
			spec.Supersedes, spec.Currency, spec.PeriodStart, spec.PeriodEnd = named.spec.Version, named.spec.Currency, named.spec.PeriodStart, named.spec.PeriodEnd
			spec.Version = named.spec.Version + 1 + rnd.Intn(16-named.spec.Version)
			for number := named.spec.Version + 1; number <= 16 && rnd.Intn(5) > 0; number++ {
				if g.w.find(ws, pool, number) == nil {
					spec.Version = number
					break
				}
			}
			if rnd.Intn(10) == 0 {
				spec.Currency = "EUR"
			}
		}
	}
	switch rnd.Intn(30) {
	case 0:
		spec.Quantum = "10"
	case 1:
		spec.PeriodEnd = spec.PeriodStart
	case 2:
		spec.Supersedes = spec.Version
	case 3:
		spec.Supersedes = 1
		spec.Version += 1
	case 4:
		spec.PeriodEnd = spec.PeriodStart.Add(41 * day)
	}
	if g.open(spec) {
		at := address{ws, pool, spec.Version}
		g.w.find(ws, pool, spec.Version).plan = g.plan(at, spec)
	}
}

func (g *generator) step() {
	rnd := g.rnd
	if len(g.order) == 0 || rnd.Intn(6) == 0 {
		g.fresh()
		return
	}
	at := g.order[rnd.Intn(len(g.order))]
	held := g.w.find(at.ws, at.pool, at.number)
	if rnd.Intn(25) == 0 {
		for _, other := range g.order {
			if made := g.w.find(other.ws, other.pool, other.number); made.recorded && made.spec.Supersedes != 0 && made.state != Withdrawn {
				g.withdraw(other, EnteredInError)
				return
			}
		}
	}
	switch choice := rnd.Intn(100); {
	case choice < 70:
		if len(held.plan) == 0 {
			g.fresh()
			return
		}
		next := held.plan[0]
		held.plan = held.plan[1:]
		next()
	case choice < 75:
		g.withdraw(at, WithdrawReason(g.pick("entered_in_error", "operator", "operator", "")))
	case choice < 77 && held.basis != nil:
		other := *held.basis
		if rnd.Intn(2) == 0 {
			other.ID = "basis-2"
		}
		g.basis(at, other)
	case choice < 82:
		g.record(at, Approval{Approver: "approver", Reference: g.pick("ref-1", "ref-2", "")})
	case choice < 86:
		g.open(held.spec)
	case choice < 89:
		spec := held.spec
		spec.Owner = "other"
		g.open(spec)
	case choice < 92:
		g.component(at, Component{ID: g.pick("fee1", "fee2", "extra", "lic3"), Kind: Kind(g.pick("facility", "licence", "rent")), Amount: g.pick("100", "-1", "0.00001", "x", "250.5"), Evidence: "doc", Asset: g.pick("", "", "asset1", "nowhere"), IncludedIn: g.pick("", "", "fee1", "missing")})
	case choice < 94:
		if rnd.Intn(2) == 0 {
			g.component(at, cost(g.pick("extra", "extra2", "extra3", "extra4"), Licence, "100"))
			return
		}
		from := held.spec.PeriodStart
		g.component(at, Component{ID: "wide", Kind: PowerCost, Amount: "100", Evidence: "doc", Power: &Power{Grade: Grade(g.pick("measured", "not_known", "guess")), Source: g.pick("m", ""), PUE: g.pick("1.4", "", "0.5"), From: from.Add(-time.Duration(rnd.Intn(2)) * time.Hour), To: from.Add(time.Hour)}})
	case choice < 96:
		g.asset(at, Asset{ID: g.pick("asset1", "asset9", "asset8", "asset7"), Kind: AssetKind(g.pick("server", "gpu", "")), Method: Kind(g.pick("depreciation", "lease", "facility")), Acquisition: g.pick("", "40000", "-1")})
	case choice < 98:
		made := Basis{ID: "basis", Measure: Measure(g.pick("reserved_capacity", "measured_usage")), Capacity: g.pick("100", "0", "200")}
		for index, count := 0, rnd.Intn(6); index < count; index++ {
			made.Targets = append(made.Targets, Target{ID: fmt.Sprintf("x%d", index-rnd.Intn(2)*rnd.Intn(2)), Bearer: Bearer(g.pick("organization", "msp", "")), Weight: "100", Idle: rnd.Intn(4) == 0})
		}
		g.basis(at, made)
	default:
		missing := address{at.ws, "dc-9", at.number}
		switch rnd.Intn(5) {
		case 4:
			g.basis(missing, shares("100"))
		case 0:
			g.record(missing, approved)
		case 1:
			g.withdraw(missing, ByOperator)
		case 2:
			g.component(missing, cost("fee", Facility, "100"))
		default:
			g.asset(missing, Asset{ID: "a", Kind: Server, Method: Lease})
		}
	}
}

func (g *generator) audit() {
	t := g.t
	for _, ws := range workspaceIDs {
		for _, pool := range poolIDs {
			versions := g.w.pools[ws+"/"+pool]
			var rows []Row
			for after := 0; ; {
				page, err := g.r.Versions(ws, pool, after, g.limits.Page)
				if err != nil {
					t.Fatal(err)
				}
				if len(page) == 0 {
					break
				}
				rows = append(rows, page...)
				after = page[len(page)-1].Version
			}
			if len(rows) != len(versions) || len(rows) > g.limits.Versions {
				t.Fatalf("%s/%s lists %d versions, the model has %d", ws, pool, len(rows), len(versions))
			}
			exact := map[span]int{}
			var current []Row
			for index, row := range rows {
				at := address{ws, pool, row.Version}
				held := versions[row.Version]
				view, err := g.r.Version(ws, pool, row.Version)
				if held == nil || err != nil || (index > 0 && rows[index-1].Version >= row.Version) {
					t.Fatalf("%v: %v", at, err)
				}
				if view.State != held.state || view.Recorded != held.recorded || view.Superseded != held.superseded || view.SupersededBy != held.by || view.Approval != held.approval || view.WithdrawReason != held.reason ||
					headerKey(view.Spec) != headerKey(held.spec) || view.Assets != len(held.assets) || view.Components != len(held.components) || row.State != view.State || row.Superseded != view.Superseded ||
					!row.PeriodStart.Equal(view.Spec.PeriodStart) || !row.PeriodEnd.Equal(view.Spec.PeriodEnd) {
					t.Fatalf("%v: %+v against %+v", at, view, held)
				}
				standing, err := g.r.Standing(ws, pool, row.Version)
				got := fmt.Sprintf("%s%v", standing.Result, standing.Reasons)
				if err != nil || got != standingOf(held.state, held.recorded, held.superseded) || got != standingOf(view.State, view.Recorded, view.Superseded) {
					t.Fatalf("%v stands as %s, the model %s, its own view %s", at, got, standingOf(held.state, held.recorded, held.superseded), standingOf(view.State, view.Recorded, view.Superseded))
				}
				g.once("standing/"+got, at)
				if g.gone[at] && !view.Superseded {
					t.Fatalf("%v was superseded and is not any more", at)
				}
				if view.Superseded {
					g.gone[at] = true
					by, err := g.r.Version(ws, pool, view.SupersededBy)
					if err != nil || by.Spec.Supersedes != row.Version || !by.Recorded || !view.Recorded {
						t.Fatalf("%v superseded by %d: %+v %v", at, view.SupersededBy, by, err)
					}
					if by.State == Withdrawn {
						g.once("superseder/withdrawn", at)
					}
				}
				if view.Spec.Supersedes != 0 && view.Recorded {
					named, err := g.r.Version(ws, pool, view.Spec.Supersedes)
					if err != nil || !named.Superseded || named.SupersededBy != row.Version {
						t.Fatalf("%v had its approval recorded and what it names is %+v", at, named)
					}
				}
				var targets []Target
				if held.basis != nil {
					for after := ""; ; {
						page, err := g.r.Targets(ws, pool, row.Version, after, g.limits.Page)
						if err != nil {
							t.Fatal(err)
						}
						if len(page) == 0 {
							break
						}
						targets = append(targets, page...)
						after = page[len(page)-1].ID
					}
					read := Basis{ID: view.Basis.ID, Measure: view.Basis.Measure, Capacity: view.Basis.Capacity, Targets: targets}
					if view.Basis.Targets != nil || view.Targets != len(held.basis.Targets) || basisKey(read) != basisKey(*held.basis) || !sort.SliceIsSorted(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID }) {
						t.Fatalf("%v: the basis read %+v, the model %+v", at, read, *held.basis)
					}
				} else if view.Basis != nil || view.Targets != 0 {
					t.Fatalf("%v has a basis the model lacks", at)
				}
				sums := held.sums()
				if (view.Blockers != nil) != (held.state == Draft) || (view.Figures != nil) != held.recorded {
					t.Fatalf("%v: blockers %v figures %v in state %s", at, view.Blockers != nil, view.Figures != nil, held.state)
				}
				if blockers := view.Blockers; blockers != nil {
					over := g.w.overlapping(ws, pool, held)
					multipleWant := MultipleNotDetermined
					if sums.unknown == 0 {
						multipleWant = map[bool]string{true: MultipleYes, false: MultipleNo}[multiple(sums.total, held.spec.Quantum)]
					}
					shown := over
					if len(shown) > g.limits.Listed {
						shown = shown[:g.limits.Listed]
					}
					named := held.spec.Supersedes != 0 && g.w.find(ws, pool, held.spec.Supersedes).superseded
					if blockers.Summed != sums.summed || blockers.WithoutAmount != sums.unknown || blockers.AssetsWithoutComponent != held.without() || blockers.BasisSet != (held.basis != nil) || blockers.Multiple != multipleWant ||
						blockers.Overlapping.Total != len(over) || fmt.Sprint(blockers.Overlapping.Items) != fmt.Sprint(shown) || blockers.NamedSuperseded != named {
						t.Fatalf("%v: %+v, the model %d %d %d %v %s %v %v", at, blockers, sums.summed, sums.unknown, held.without(), held.basis != nil, multipleWant, over, named)
					}
					g.once("multiple/"+blockers.Multiple, at)
					if len(over) > 0 {
						g.once("overlap/told", at)
					}
				}
				if view.Figures != nil {
					classes := map[Bearer]*big.Rat{}
					idle, any := new(big.Rat), false
					for _, share := range g.allShares(at) {
						if classes[share.Bearer] == nil {
							classes[share.Bearer] = new(big.Rat)
						}
						classes[share.Bearer].Add(classes[share.Bearer], ratOf(share.Amount))
						if share.Idle {
							any = true
							idle.Add(idle, ratOf(share.Amount))
						}
					}
					whole := new(big.Rat)
					for class, sum := range classes {
						sums.figures.ByBearer[class] = ratText(sum)
						whole.Add(whole, sum)
					}
					if any {
						sums.figures.HasIdle, sums.figures.Idle = true, ratText(idle)
					}
					if flat(view.Figures) != flat(sums.figures) || whole.Cmp(sums.total) != 0 {
						t.Fatalf("%v figures\n got %s\nwant %s", at, flat(view.Figures), flat(sums.figures))
					}
					var listed []string
					after := ""
					for {
						page, err := g.r.AssetSums(ws, pool, row.Version, after, g.limits.Page)
						if err != nil {
							t.Fatal(err)
						}
						if len(page) == 0 {
							break
						}
						for _, sum := range page {
							listed = append(listed, sum.Asset+"="+sum.Sum)
						}
						after = page[len(page)-1].Asset
					}
					var wanted []string
					for id, sum := range sums.assets {
						wanted = append(wanted, id+"="+ratText(sum))
					}
					sort.Strings(wanted)
					if !reflect.DeepEqual(listed, wanted) {
						t.Fatalf("%v asset sums %v, the model %v", at, listed, wanted)
					}
					for grade := range view.Figures.ByGrade {
						g.once("grade/"+string(grade), at)
					}
					if view.Figures.IncludedKnown+view.Figures.IncludedNotKnown > 0 {
						g.once("included", at)
					}
				}
				if standing.Result == Current {
					for _, other := range current {
						if row.PeriodStart.Before(other.PeriodEnd) && other.PeriodStart.Before(row.PeriodEnd) {
							t.Fatalf("%v and version %d are both current and overlap", at, other.Version)
						}
					}
					current = append(current, row)
					exact[span{row.PeriodStart, row.PeriodEnd}] = row.Version
				}
			}
			for _, period := range spans() {
				counted, err := g.r.Counting(ws, pool, period.from, period.to)
				want := exact[period]
				if counted.Version != want || (err == nil) != (want != 0) || (err != nil && err != ErrNotFound) || counted.Sequence != g.w.sequence[ws+"/"+pool] {
					t.Fatalf("%s/%s counting %+v %v, want %d at %d", ws, pool, counted, err, want, g.w.sequence[ws+"/"+pool])
				}
				if want != 0 {
					g.once("counting", address{ws, pool, want})
				}
			}
		}
	}
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	reached := map[string]int{}
	limits := Limits{Pools: 2, Versions: 12, Assets: 1, Components: 4, Targets: 3, Page: 2, Listed: 1, MaxPeriod: 40 * day, Intermediate: 40}
	for seed := int64(1); seed <= 300; seed++ {
		clock := &testClock{at: start}
		r, err := NewRegistry(limits, profileOf(t, 12, 14), clock.read)
		if err != nil {
			t.Fatal(err)
		}
		g := &generator{t: t, rnd: rand.New(rand.NewSource(seed)), r: r, w: newWorld(limits), limits: limits, reached: reached, counted: map[string]bool{}, gone: map[address]bool{}}
		for step := 0; step < 120; step++ {
			g.step()
			if step%20 == 19 {
				g.audit()
			}
			clock.advance(time.Second)
		}
		g.audit()
	}
	least := map[string]int{"recorded": 600, "supersessions": 60, "shares/several": 300, "counting": 400, "superseder/withdrawn": 10, "overlap/told": 50, "included": 50,
		"standing/current[]": 300, "standing/not_current[withdrawn]": 100, "standing/not_current[no_approval_recorded]": 300, "standing/not_current[superseded]": 50,
		"multiple/yes": 100, "multiple/no": 30, "multiple/not_determined": 30, "grade/measured": 20, "grade/estimated": 20, "grade/not_known": 20}
	for _, key := range []string{"open/ok", "asset/ok", "component/ok", "basis/ok", "record/ok", "withdraw/ok"} {
		least[key] = 300
	}
	for _, key := range []string{"open/" + kindOf(ErrInvalid), "open/" + kindOf(ErrConflict), "open/" + kindOf(ErrRejected), "open/" + kindOf(ErrLimit),
		"asset/" + kindOf(ErrInvalid), "asset/" + kindOf(ErrNotFound), "asset/" + kindOf(ErrConflict), "asset/" + kindOf(ErrRejected), "asset/" + kindOf(ErrLimit),
		"component/" + kindOf(ErrInvalid), "component/" + kindOf(ErrNotFound), "component/" + kindOf(ErrConflict), "component/" + kindOf(ErrRejected), "component/" + kindOf(ErrLimit),
		"basis/" + kindOf(ErrInvalid), "basis/" + kindOf(ErrNotFound), "basis/" + kindOf(ErrConflict), "basis/" + kindOf(ErrRejected), "basis/" + kindOf(ErrLimit),
		"record/" + kindOf(ErrInvalid), "record/" + kindOf(ErrNotFound), "record/" + kindOf(ErrConflict), "record/" + kindOf(ErrRejected),
		"withdraw/" + kindOf(ErrInvalid), "withdraw/" + kindOf(ErrNotFound), "withdraw/" + kindOf(ErrConflict)} {
		least[key] = 15
	}
	keys := make([]string, 0, len(least))
	for key := range least {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if reached[key] < least[key] {
			t.Errorf("%s reached %d times, want at least %d", key, reached[key], least[key])
		}
	}
	if t.Failed() {
		t.Logf("%v", reached)
	}
}

func TestConcurrentUse(t *testing.T) {
	from, to := month(10)
	raced := 0
	for round := 0; round < 40; round++ {
		r, err := NewRegistry(testLimits(), profileOf(t, 12, 14), func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		h := &harness{t: t, r: r}
		h.whole(header(1, 10, 0), "1200000")
		h.fill(header(2, 10, 1), "900000")
		h.fill(header(3, 10, 1), "800000")
		for _, number := range []int{4, 5} {
			spec := header(number, 11, 0)
			spec.PeriodStart = spec.PeriodStart.Add(time.Duration(number) * day)
			h.fill(spec, "100")
		}
		h.open(header(6, 9, 0))
		h.basis(6, shares("1", "1"))
		begin := make(chan struct{})
		var group sync.WaitGroup
		results := make([]error, 6)
		added := make([]bool, 8)
		run := func(work func()) {
			group.Add(1)
			go func() {
				defer group.Done()
				<-begin
				work()
			}()
		}
		for index, number := range []int{2, 3, 4, 5} {
			run(func() { _, results[index] = r.RecordApproval("ws1", "dc-1", number, approved) })
		}
		componentsMay, recordMay := make(chan struct{}), make(chan struct{})
		var reported sync.WaitGroup
		reported.Add(len(added))
		if round%3 != 0 {
			close(componentsMay)
		}
		if round%3 != 1 {
			close(recordMay)
		} else {
			go func() {
				reported.Wait()
				close(recordMay)
			}()
		}
		for index := range added {
			run(func() {
				defer reported.Done()
				<-componentsMay
				switch _, err := r.PutComponent("ws1", "dc-1", 6, cost("fee", Facility, "500")); err {
				case nil:
					added[index] = true
				case ErrRejected:
				default:
					t.Errorf("component: %v", err)
				}
			})
		}
		run(func() {
			<-recordMay
			_, results[4] = r.RecordApproval("ws1", "dc-1", 6, approved)
			if round%3 == 0 {
				close(componentsMay)
			}
		})
		for reader := 0; reader < 3; reader++ {
			run(func() {
				for spin := 0; spin < 50; spin++ {
					counted, err := r.Counting("ws1", "dc-1", from, to)
					if err != nil || (counted.Version != 1 && counted.Version != 2 && counted.Version != 3) {
						t.Errorf("counting: %+v %v", counted, err)
						return
					}
					if _, err := r.Versions("ws1", "dc-1", 0, 4); err != nil {
						t.Errorf("versions: %v", err)
					}
				}
			})
		}
		close(begin)
		group.Wait()
		if (results[0] == nil) == (results[1] == nil) || (results[0] != nil && results[0] != ErrRejected) || (results[1] != nil && results[1] != ErrRejected) {
			t.Fatalf("round %d: two versions naming the same one: %v %v", round, results[0], results[1])
		}
		if (results[2] == nil) == (results[3] == nil) || (results[2] != nil && results[2] != ErrRejected) || (results[3] != nil && results[3] != ErrRejected) {
			t.Fatalf("round %d: two overlapping versions: %v %v", round, results[2], results[3])
		}
		winner := map[bool]int{true: 2, false: 3}[results[0] == nil]
		if view := h.view(1); !view.Superseded || view.SupersededBy != winner || h.standing(winner) != "current[]" || h.standing(5-winner) != "not_current[no_approval_recorded]" {
			t.Fatalf("round %d: %+v", round, view)
		}
		any := false
		for _, did := range added {
			any = any || did
		}
		view := h.view(6)
		if round%3 == 0 && results[4] != ErrRejected || round%3 == 1 && results[4] != nil {
			t.Fatalf("round %d: the approval in a fixed order gave %v", round, results[4])
		}
		switch {
		case results[4] == nil:
			raced++
			for _, did := range added {
				if !did {
					t.Fatalf("round %d: an identical component was refused although it is stored", round)
				}
			}
			if view.Components != 1 || view.Figures.Total != "500" || h.shares(6) != "t1=250(organization/1) t2=250(organization/1)" {
				t.Fatalf("round %d: %+v", round, view)
			}
		case results[4] == ErrRejected:
			if view.State != Draft || view.Components != 1 || !any {
				t.Fatalf("round %d: %+v", round, view)
			}
		default:
			t.Fatalf("round %d: %v", round, results[4])
		}
		wantSequence := uint64(4 + 3 + 3 + 3 + 3 + 2 + 2 + 1)
		if results[4] == nil {
			wantSequence++
		}
		if counted, err := r.Counting("ws1", "dc-1", from, to); err != nil || counted.Version != winner || counted.Sequence != wantSequence {
			t.Fatalf("round %d: %+v %v, want the number %d", round, counted, err, wantSequence)
		}
		begin = make(chan struct{})
		for reader := 0; reader < 4; reader++ {
			run(func() {
				gone := false
				for spin := 0; spin < 200; spin++ {
					standing, err := r.Standing("ws1", "dc-1", winner)
					counted, found := r.Counting("ws1", "dc-1", from, to)
					if err != nil || (found != nil && found != ErrNotFound) {
						t.Errorf("reads: %v %v", err, found)
						return
					}
					switch {
					case standing.Result == NotCurrent && len(standing.Reasons) == 1 && standing.Reasons[0] == IsWithdrawn:
						gone = true
					case standing.Result != Current || gone:
						t.Errorf("%+v after the withdrawal was seen: %v", standing, gone)
						return
					}
					if found == nil && (gone || counted.Version != winner) {
						t.Errorf("%+v counts, withdrawal seen: %v", counted, gone)
						return
					}
				}
			})
		}
		run(func() {
			for spin := 0; spin < round*10; spin++ {
				if _, err := r.Standing("ws1", "dc-1", winner); err != nil {
					t.Errorf("standing: %v", err)
				}
			}
			if err := r.Withdraw("ws1", "dc-1", winner, ByOperator); err != nil {
				t.Errorf("withdraw: %v", err)
			}
		})
		close(begin)
		group.Wait()
		if _, err := r.Counting("ws1", "dc-1", from, to); err != ErrNotFound || h.standing(winner) != "not_current[withdrawn]" || h.standing(1) != "not_current[superseded]" {
			t.Fatalf("round %d: %v", round, err)
		}
	}
	if raced < 13 {
		t.Errorf("the approval was recorded after the components in %d of 40 rounds", raced)
	}
}
