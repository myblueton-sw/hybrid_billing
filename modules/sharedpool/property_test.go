package sharedpool

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

var errorKinds = []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit, ErrSelfCheck}

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

func ratOf(text string) *big.Rat {
	value, read := new(big.Rat).SetString(text)
	if !read {
		panic(text)
	}
	return value
}

type mRun struct {
	spec        RunSpec
	state       State
	judged      bool
	published   bool
	chunks      map[int][]Weight
	hold        string
	alternative *Alternative
	approval    Approval
	reason      WithdrawReason
	plan        [][]Weight
}

type mVersion struct {
	spec Spec
	runs map[string]*mRun
}

type world struct {
	limits   Limits
	pools    map[string]map[int]*mVersion
	sequence map[string]uint64
}

func specKey(s Spec) string {
	return fmt.Sprintf("%s|%d|%d|%s|%s|%s|%s", s.Currency, s.PeriodStart.Unix(), s.PeriodEnd.Unix(), s.Amount, s.Quantum, s.Capacity, s.ResidualBearer)
}

func chunkKey(weights []Weight) string {
	sorted := append([]Weight(nil), weights...)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a].Target < sorted[b].Target })
	return fmt.Sprint(sorted)
}

var formed = map[string]bool{"": true, "0": true, "1": true, "2": true, "3.5": true, "60": true, "40": true, "0.0": true, "-1": false, "x": false}

func (w *world) open(spec Spec) error {
	amount, read := new(big.Rat).SetString(spec.Amount)
	quanta := map[string]bool{"1": true, "0.1": true, "0.01": true}
	if spec.Version < 1 || !spec.PeriodStart.Before(spec.PeriodEnd) || !read || !quanta[spec.Quantum] || !new(big.Rat).Quo(amount, ratOf(spec.Quantum)).IsInt() || (spec.Capacity == "") != (spec.ResidualBearer == "") {
		return ErrInvalid
	}
	if spec.Capacity != "" && (ratOf(spec.Capacity).Sign() <= 0 || !spec.ResidualBearer.residual()) {
		return ErrInvalid
	}
	versions := w.pools[spec.Pool]
	if held := versions[spec.Version]; held != nil {
		if specKey(held.spec) != specKey(spec) {
			return ErrConflict
		}
		return nil
	}
	if versions == nil && len(w.pools) >= w.limits.Pools || len(versions) >= w.limits.Versions {
		return ErrLimit
	}
	if versions == nil {
		versions = map[int]*mVersion{}
		w.pools[spec.Pool] = versions
	}
	versions[spec.Version] = &mVersion{spec: spec, runs: map[string]*mRun{}}
	w.sequence[spec.Pool]++
	return nil
}

func (w *world) find(pool string, version int, id string) (*mVersion, *mRun) {
	held := w.pools[pool][version]
	if held == nil {
		return nil, nil
	}
	return held, held.runs[id]
}

func (w *world) start(pool string, version int, spec RunSpec) (State, error) {
	if (spec.Purpose != Showback && spec.Purpose != Simulation && spec.Purpose != Chargeback) || spec.Chunks < 1 {
		return "", ErrInvalid
	}
	home, held := w.find(pool, version, spec.ID)
	if home == nil {
		return "", ErrNotFound
	}
	if spec.Chunks > w.limits.Chunks {
		return "", ErrLimit
	}
	if held != nil {
		if held.spec != spec {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if len(home.runs) >= w.limits.Runs {
		return "", ErrLimit
	}
	home.runs[spec.ID] = &mRun{spec: spec, state: Staging, chunks: map[int][]Weight{}}
	w.sequence[pool]++
	return Staging, nil
}

func (w *world) stage(pool string, version int, id string, position int, weights []Weight) (State, error) {
	if position < 1 || len(weights) == 0 {
		return "", ErrInvalid
	}
	if len(weights) > w.limits.ChunkWeights {
		return "", ErrLimit
	}
	seen := map[string]bool{}
	for _, weight := range weights {
		if seen[weight.Target] || weight.Target[0] == ':' || !weight.Bearer.known() || !formed[weight.Weight] {
			return "", ErrInvalid
		}
		seen[weight.Target] = true
	}
	_, held := w.find(pool, version, id)
	if held == nil {
		return "", ErrNotFound
	}
	if earlier, staged := held.chunks[position]; staged {
		if chunkKey(earlier) != chunkKey(weights) {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Staging || position > held.spec.Chunks {
		return "", ErrRejected
	}
	count := 0
	for _, other := range held.chunks {
		count += len(other)
		for _, weight := range other {
			if seen[weight.Target] {
				return "", ErrRejected
			}
		}
	}
	if count+len(weights) > w.limits.Targets {
		return "", ErrLimit
	}
	held.chunks[position] = append([]Weight(nil), weights...)
	w.sequence[pool]++
	return held.state, nil
}

func (held *mRun) all() []Weight {
	var out []Weight
	for _, part := range held.chunks {
		out = append(out, part...)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Target < out[b].Target })
	return out
}

func (w *world) seal(pool string, version int, id string) (State, error) {
	home, held := w.find(pool, version, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.judged {
		return held.state, nil
	}
	if held.state != Staging || len(held.chunks) != held.spec.Chunks {
		return "", ErrRejected
	}
	sum := new(big.Rat)
	unknown := false
	for _, weight := range held.all() {
		if weight.Weight == "" {
			unknown = true
			continue
		}
		sum.Add(sum, ratOf(weight.Weight))
	}
	held.judged, held.state = true, Sealed
	switch {
	case unknown:
		held.state, held.hold = Held, WeightNotKnown
	case home.spec.Capacity == "" && sum.Sign() == 0:
		held.state, held.hold = Held, ZeroDenominator
	case home.spec.Capacity != "" && sum.Cmp(ratOf(home.spec.Capacity)) > 0:
		held.state, held.hold = Held, OverCapacity
	}
	w.sequence[pool]++
	return held.state, nil
}

func (w *world) settle(pool string, version int, id string, alternative Alternative) (State, error) {
	if alternative.Approver == "" || alternative.Reference == "" || !alternative.Bearer.residual() {
		return "", ErrInvalid
	}
	_, held := w.find(pool, version, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.alternative != nil {
		if *held.alternative != alternative {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Held || held.hold != ZeroDenominator {
		return "", ErrRejected
	}
	held.state, held.alternative = Sealed, &alternative
	w.sequence[pool]++
	return Sealed, nil
}

func (w *world) charged(pool string) []*mRun {
	var out []*mRun
	for _, home := range w.pools[pool] {
		for _, held := range home.runs {
			if held.state == Published && held.spec.Purpose == Chargeback {
				out = append(out, held)
			}
		}
	}
	return out
}

func (w *world) versionOf(pool string, target *mRun) *mVersion {
	for _, home := range w.pools[pool] {
		for _, held := range home.runs {
			if held == target {
				return home
			}
		}
	}
	return nil
}

func (w *world) publish(pool string, version int, id string, approval Approval) (State, error) {
	home, held := w.find(pool, version, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.published {
		if held.approval != approval {
			return "", ErrConflict
		}
		return held.state, nil
	}
	charged := held.spec.Purpose == Chargeback
	if held.state != Sealed || charged != (approval.Approver != "") || charged != (approval.Reference != "") {
		return "", ErrRejected
	}
	if charged {
		for _, other := range w.charged(pool) {
			against := w.versionOf(pool, other).spec
			if home.spec.PeriodStart.Before(against.PeriodEnd) && against.PeriodStart.Before(home.spec.PeriodEnd) {
				return "", ErrRejected
			}
		}
	}
	held.state, held.published, held.approval = Published, true, approval
	w.sequence[pool]++
	return Published, nil
}

func (w *world) withdraw(pool string, version int, id string, reason WithdrawReason) error {
	if reason != EnteredInError && reason != ByOperator {
		return ErrInvalid
	}
	_, held := w.find(pool, version, id)
	switch {
	case held == nil:
		return ErrNotFound
	case held.state == Withdrawn && held.reason != reason:
		return ErrConflict
	case held.state == Withdrawn:
		return nil
	}
	held.state, held.reason = Withdrawn, reason
	w.sequence[pool]++
	return nil
}

type span struct{ from, to time.Time }

func spans() []span {
	october, november := month(10)
	_, december := month(11)
	return []span{{october, november}, {november, december}, {october.Add(15 * day), november.Add(15 * day)}, {october.Add(-30 * day), october}}
}

type generator struct {
	t       *testing.T
	rnd     *rand.Rand
	r       *Registry
	w       *world
	seed    int64
	reached map[string]int
	counted map[string]bool
	order   []address
}

type address struct {
	pool    string
	version int
	run     string
}

func (g *generator) pick(options ...string) string { return options[g.rnd.Intn(len(options))] }

func (g *generator) once(key string, where any) {
	if mark := fmt.Sprint(key, "|", g.seed, "|", where); !g.counted[mark] {
		g.counted[mark] = true
		g.reached[key]++
	}
}

func (g *generator) same(what string, where any, got, want error) {
	g.t.Helper()
	if got != want {
		g.t.Fatalf("seed %d %s %v: the registry said %v, the model %v", g.seed, what, where, got, want)
	}
	if want == nil {
		g.reached[what+"/ok"]++
		return
	}
	g.once(what+"/"+kindOf(want), where)
}

func (g *generator) checkShares(at address) {
	t := g.t
	home, held := g.w.find(at.pool, at.version, at.run)
	var shares []Share
	for after := ""; ; {
		page, err := g.r.Shares("ws1", at.pool, at.version, at.run, after, 3)
		if err != nil || page.Purpose != held.spec.Purpose || page.State != held.state || page.Sequence != g.w.sequence[at.pool] {
			t.Fatalf("seed %d %v: %+v %v", g.seed, at, page, err)
		}
		shares = append(shares, page.Items...)
		if !page.More {
			break
		}
		after = page.Items[len(page.Items)-1].Target
	}
	staged := held.all()
	amount, quantum := ratOf(home.spec.Amount), ratOf(home.spec.Quantum)
	sum := new(big.Rat)
	for _, weight := range staged {
		sum.Add(sum, ratOf(weight.Weight))
	}
	denominator := sum
	wantResidual, residualWeight := false, new(big.Rat)
	if home.spec.Capacity != "" {
		denominator = ratOf(home.spec.Capacity)
		residualWeight.Sub(denominator, sum)
		wantResidual = residualWeight.Sign() > 0
	}
	settled := held.alternative != nil
	if settled {
		wantResidual = true
	}
	if len(shares) != len(staged)+map[bool]int{true: 1, false: 0}[wantResidual] {
		t.Fatalf("seed %d %v: %d shares for %d targets, residual %v", g.seed, at, len(shares), len(staged), wantResidual)
	}
	classes := map[Bearer]*big.Rat{}
	total, others := new(big.Rat), new(big.Rat)
	residual := ""
	index := 0
	type pair struct{ weight, amount *big.Rat }
	var pairs []pair
	for position, share := range shares {
		if position > 0 && shares[position-1].Target >= share.Target {
			t.Fatalf("seed %d %v: shares out of order", g.seed, at)
		}
		value := ratOf(share.Amount)
		if classes[share.Bearer] == nil {
			classes[share.Bearer] = new(big.Rat)
		}
		classes[share.Bearer].Add(classes[share.Bearer], value)
		total.Add(total, value)
		weight := new(big.Rat)
		if share.Residual {
			residual = share.Amount
			bearer := home.spec.ResidualBearer
			weight.Set(residualWeight)
			if settled {
				bearer = held.alternative.Bearer
				weight.SetInt64(0)
			}
			if share.Target != residualID || share.Bearer != bearer || ratOf(share.Weight).Cmp(weight) != 0 || !wantResidual {
				t.Fatalf("seed %d %v: residual %+v", g.seed, at, share)
			}
		} else {
			others.Add(others, value)
			if index >= len(staged) || share.Target != staged[index].Target || share.Bearer != staged[index].Bearer || share.Weight != staged[index].Weight {
				t.Fatalf("seed %d %v: share %+v against what was staged", g.seed, at, share)
			}
			weight = ratOf(staged[index].Weight)
			index++
		}
		if !new(big.Rat).Quo(value, quantum).IsInt() || value.Sign() < 0 {
			t.Fatalf("seed %d %v: a share of %s at quantum %s", g.seed, at, share.Amount, home.spec.Quantum)
		}
		if settled {
			if (share.Residual && value.Cmp(amount) != 0) || (!share.Residual && value.Sign() != 0) {
				t.Fatalf("seed %d %v: a settled share %+v", g.seed, at, share)
			}
			continue
		}
		exact := new(big.Rat).Mul(amount, new(big.Rat).Quo(weight, denominator))
		if distance := new(big.Rat).Abs(new(big.Rat).Sub(value, exact)); distance.Cmp(quantum) >= 0 || (weight.Sign() == 0 && value.Sign() != 0) {
			t.Fatalf("seed %d %v: share %s, exact %s", g.seed, at, share.Amount, exact.FloatString(6))
		}
		for _, other := range pairs {
			if order := other.weight.Cmp(weight); order > 0 && other.amount.Cmp(value) < 0 || order < 0 && other.amount.Cmp(value) > 0 {
				t.Fatalf("seed %d %v: a larger weight has a smaller share", g.seed, at)
			}
		}
		pairs = append(pairs, pair{weight, value})
	}
	whole := new(big.Rat).Set(others)
	if residual != "" {
		whole.Add(whole, ratOf(residual))
	}
	if total.Cmp(amount) != 0 || whole.Cmp(amount) != 0 {
		t.Fatalf("seed %d %v: the shares add up to %s of %s", g.seed, at, total.FloatString(4), home.spec.Amount)
	}
	view, err := g.r.Run("ws1", at.pool, at.version, at.run)
	if err != nil || len(view.ByBearer) != len(classes) || view.Residual != residual {
		t.Fatalf("seed %d %v: %+v %v", g.seed, at, view, err)
	}
	for class, want := range classes {
		if ratOf(view.ByBearer[class]).Cmp(want) != 0 {
			t.Fatalf("seed %d %v: the sum of %s is %s, the shares read add up to %s", g.seed, at, class, view.ByBearer[class], want.FloatString(4))
		}
	}
	wantReason := map[bool]string{true: BelowCapacity, false: ""}[wantResidual]
	if settled {
		wantReason = RecordedAlternative
	}
	if view.ResidualReason != wantReason {
		t.Fatalf("seed %d %v: residual reason %q", g.seed, at, view.ResidualReason)
	}
	if wantResidual {
		g.once("residual/"+wantReason, at)
	}
	if settled {
		return
	}
	other, err := NewRegistry(Limits{Pools: 1, Versions: 1, Runs: 1, Chunks: len(staged), ChunkWeights: len(staged), Targets: len(staged), Page: 3, MaxPeriod: 40 * day, Intermediate: 40}, profileOf(t, 12, 4, 40), func() time.Time { return start })
	if err != nil {
		t.Fatal(err)
	}
	if err := other.OpenPool(home.spec); err != nil {
		t.Fatal(err)
	}
	shuffled := append([]Weight(nil), staged...)
	g.rnd.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
	cuts := 1 + g.rnd.Intn(len(shuffled))
	if _, err := other.StartRun("ws1", at.pool, at.version, RunSpec{ID: "again", Purpose: Simulation, Chunks: cuts}); err != nil {
		t.Fatal(err)
	}
	for position := cuts; position >= 1; position-- {
		first, last := (position-1)*len(shuffled)/cuts, position*len(shuffled)/cuts
		if _, err := other.StageChunk("ws1", at.pool, at.version, "again", position, shuffled[first:last]); err != nil {
			t.Fatal(err)
		}
	}
	if state, err := other.Seal("ws1", at.pool, at.version, "again"); err != nil || state != Sealed {
		t.Fatal(state, err)
	}
	if _, err := other.Publish("ws1", at.pool, at.version, "again", Approval{}); err != nil {
		t.Fatal(err)
	}
	var again []Share
	for after := ""; ; {
		page, err := other.Shares("ws1", at.pool, at.version, "again", after, 3)
		if err != nil {
			t.Fatal(err)
		}
		again = append(again, page.Items...)
		if !page.More {
			break
		}
		after = page.Items[len(page.Items)-1].Target
	}
	if shareText(again) != shareText(shares) {
		t.Fatalf("seed %d %v: the same targets in another order and cut:\n%s\n%s", g.seed, at, shareText(shares), shareText(again))
	}
	g.reached["shuffled"]++
}

func (g *generator) step() {
	rnd := g.rnd
	choice := rnd.Intn(100)
	if choice < 12 || len(g.w.pools) == 0 {
		period := spans()[rnd.Intn(len(spans()))]
		spec := Spec{WorkspaceID: "ws1", Pool: g.pick("p0", "p0", "p1"), Version: 1 + rnd.Intn(5), Currency: "KRW", PeriodStart: period.from, PeriodEnd: period.to, Amount: g.pick("100", "7", "0", "1000000", "100"), Quantum: g.pick("1", "1", "0.1", "0.01"),
			Capacity: g.pick("", "", "200", "100", "100", "10")}
		if spec.Capacity != "" {
			spec.ResidualBearer = Bearer(g.pick("msp", "unattributed"))
		}
		switch rnd.Intn(20) {
		case 0:
			spec.Amount = "12.5"
		case 1:
			spec.ResidualBearer = Organization
		case 2:
			spec.PeriodEnd = spec.PeriodStart
		}
		g.same("pool", fmt.Sprint(spec.Pool, spec.Version), g.r.OpenPool(spec), g.w.open(spec))
		return
	}
	if choice < 26 {
		at := address{g.pick("p0", "p0", "p1"), 1 + rnd.Intn(5), g.pick("r0", "r1", "r2", "r3", "r4")}
		spec := RunSpec{ID: at.run, Purpose: Purpose(g.pick("showback", "simulation", "chargeback")), Chunks: 1 + rnd.Intn(3)}
		if rnd.Intn(20) == 0 {
			spec.Chunks = 5
		}
		if rnd.Intn(25) == 0 {
			spec.Purpose = ""
		}
		_, before := g.w.find(at.pool, at.version, at.run)
		state, err := g.r.StartRun("ws1", at.pool, at.version, spec)
		want, wantErr := g.w.start(at.pool, at.version, spec)
		g.same("run", at, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
		if _, held := g.w.find(at.pool, at.version, at.run); held != nil && before == nil {
			g.order = append(g.order, at)
			targets := []string{"t0", "t1", "t2", "t3", "1x", "_y"}
			rnd.Shuffle(len(targets), func(a, b int) { targets[a], targets[b] = targets[b], targets[a] })
			count := spec.Chunks + rnd.Intn(len(targets)-spec.Chunks+1)
			kind := rnd.Intn(12)
			held.plan = make([][]Weight, spec.Chunks)
			for index := 0; index < count; index++ {
				weight := g.pick("1", "2", "3.5", "60", "40", "0", "1", "2")
				if kind < 2 {
					weight = g.pick("0", "0.0")
				}
				if kind == 2 && index == 0 {
					weight = ""
				}
				position := index % spec.Chunks
				held.plan[position] = append(held.plan[position], w(targets[index], Bearer(g.pick("organization", "organization", "msp", "unattributed")), weight))
			}
		}
		return
	}
	if len(g.order) == 0 {
		return
	}
	at := g.order[rnd.Intn(len(g.order))]
	_, held := g.w.find(at.pool, at.version, at.run)
	if choice >= 76 && choice < 82 {
		for _, other := range g.order {
			if _, made := g.w.find(other.pool, other.version, other.run); made.hold == ZeroDenominator && (made.state == Held || rnd.Intn(3) == 0) {
				at, held = other, made
				break
			}
		}
	}
	switch {
	case choice < 62:
		position := 1 + rnd.Intn(held.spec.Chunks)
		for candidate := 1; candidate <= held.spec.Chunks; candidate++ {
			if _, staged := held.chunks[candidate]; !staged && rnd.Intn(3) > 0 {
				position = candidate
				break
			}
		}
		weights := append([]Weight(nil), held.plan[position-1]...)
		switch rnd.Intn(25) {
		case 0:
			weights[0].Weight = g.pick("-1", "x")
		case 1:
			weights = append(weights, weights[0])
		case 2:
			weights[0].Weight = "40"
		case 3:
			position = held.spec.Chunks + 1
		case 4:
			weights = append(weights, held.plan[position%held.spec.Chunks][0])
		case 5:
			weights = nil
		case 6:
			weights[0].Target = ":x"
		case 7:
			for len(weights) < 5 {
				weights = append(weights, w(fmt.Sprintf("extra%d", len(weights)), MSP, "1"))
			}
		}
		if rnd.Intn(4) == 0 {
			rnd.Shuffle(len(weights), func(a, b int) { weights[a], weights[b] = weights[b], weights[a] })
		}
		state, err := g.r.StageChunk("ws1", at.pool, at.version, at.run, position, weights)
		want, wantErr := g.w.stage(at.pool, at.version, at.run, position, weights)
		g.same("chunk", at, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
	case choice < 76:
		state, err := g.r.Seal("ws1", at.pool, at.version, at.run)
		want, wantErr := g.w.seal(at.pool, at.version, at.run)
		g.same("seal", at, err, wantErr)
		if state != want {
			g.t.Fatalf("seed %d %v: sealed as %s, the model %s", g.seed, at, state, want)
		}
	case choice < 82:
		alternative := Alternative{Approver: "approver", Reference: g.pick("d1", "d1", "d2"), Bearer: Bearer(g.pick("msp", "unattributed", "msp", "organization"))}
		state, err := g.r.Settle("ws1", at.pool, at.version, at.run, alternative)
		want, wantErr := g.w.settle(at.pool, at.version, at.run, alternative)
		g.same("settle", at, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
	case choice < 94:
		approval := Approval{}
		if (held.spec.Purpose == Chargeback) != (rnd.Intn(8) == 0) {
			approval = Approval{Approver: "approver", Reference: g.pick("doc-1", "doc-1", "doc-2")}
		}
		was := held.published
		state, err := g.r.Publish("ws1", at.pool, at.version, at.run, approval)
		want, wantErr := g.w.publish(at.pool, at.version, at.run, approval)
		g.same("publish", at, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
		if wantErr == nil && !was {
			g.checkShares(at)
			g.once("published/"+string(held.spec.Purpose), at)
		}
	default:
		reason := WithdrawReason(g.pick("entered_in_error", "operator", "operator", ""))
		if rnd.Intn(3) > 0 && held.state != Withdrawn && held.state != Published && held.state != Held {
			return
		}
		g.same("withdraw", at, g.r.Withdraw("ws1", at.pool, at.version, at.run, reason), g.w.withdraw(at.pool, at.version, at.run, reason))
		if rnd.Intn(3) == 0 {
			other := map[WithdrawReason]WithdrawReason{EnteredInError: ByOperator, ByOperator: EnteredInError, "": ByOperator}[reason]
			g.same("withdraw", at, g.r.Withdraw("ws1", at.pool, at.version, at.run, other), g.w.withdraw(at.pool, at.version, at.run, other))
		}
	}
}

func (g *generator) audit() {
	t := g.t
	for _, at := range g.order {
		home, held := g.w.find(at.pool, at.version, at.run)
		view, err := g.r.Run("ws1", at.pool, at.version, at.run)
		unknown := 0
		for _, weight := range held.all() {
			if weight.Weight == "" {
				unknown++
			}
		}
		if err != nil || view.State != held.state || view.Purpose != held.spec.Purpose || view.Staged != len(held.chunks) || view.Declared != held.spec.Chunks || view.Targets != len(held.all()) || view.NotKnown != unknown || view.Hold != held.hold ||
			view.WasPublished != held.published || view.Approval != held.approval || view.WithdrawReason != held.reason || (view.Alternative == nil) != (held.alternative == nil) || view.Amount != home.spec.Amount {
			t.Fatalf("seed %d %v: %+v %v against %+v", g.seed, at, view, err, held)
		}
		g.once("state/"+string(view.State), at)
		if view.Hold != "" {
			g.once("hold/"+view.Hold, at)
		}
		_, err = g.r.Shares("ws1", at.pool, at.version, at.run, "", 1)
		if (err == nil) != held.published || (err != nil && err != ErrRejected) || (!held.published && (view.ByBearer != nil || view.Residual != "" || view.ResidualReason != "")) {
			t.Fatalf("seed %d %v: shares readable %v, published %v: %+v", g.seed, at, err == nil, held.published, view)
		}
		if held.state == Sealed {
			g.r.mu.Lock()
			stored := g.r.workspaces["ws1"][at.pool].versions[at.version].runs[at.run].shares
			sum := new(big.Rat)
			for _, share := range stored {
				sum.Add(sum, ratOf(share.Amount))
			}
			g.r.mu.Unlock()
			if sum.Cmp(ratOf(home.spec.Amount)) != 0 || len(stored) < len(held.all()) {
				t.Fatalf("seed %d %v: the stored shares of a sealed run add up to %s", g.seed, at, sum.FloatString(4))
			}
		}
	}
	for _, pool := range []string{"p0", "p1"} {
		charged := g.w.charged(pool)
		for index, a := range charged {
			for _, b := range charged[:index] {
				x, y := g.w.versionOf(pool, a).spec, g.w.versionOf(pool, b).spec
				if x.PeriodStart.Before(y.PeriodEnd) && y.PeriodStart.Before(x.PeriodEnd) {
					t.Fatalf("seed %d: two published chargeback runs of %s overlap", g.seed, pool)
				}
			}
		}
		for _, period := range spans() {
			billed, err := g.r.Chargeback("ws1", pool, period.from, period.to)
			want := PublishedChargeback{Sequence: g.w.sequence[pool]}
			for _, held := range charged {
				if spec := g.w.versionOf(pool, held).spec; spec.PeriodStart.Equal(period.from) && spec.PeriodEnd.Equal(period.to) {
					want.Version, want.Run = spec.Version, held.spec.ID
				}
			}
			if billed != want || (err == nil) != (want.Run != "") || (err != nil && err != ErrNotFound) {
				t.Fatalf("seed %d %s: chargeback %+v %v, the model %+v", g.seed, pool, billed, err, want)
			}
			if want.Run != "" {
				g.once("billed", fmt.Sprint(pool, want.Version, want.Run))
			}
		}
	}
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	reached := map[string]int{}
	limits := Limits{Pools: 2, Versions: 4, Runs: 4, Chunks: 4, ChunkWeights: 4, Targets: 7, Page: 3, MaxPeriod: 40 * day, Intermediate: 40}
	for seed := int64(1); seed <= 300; seed++ {
		r, err := NewRegistry(limits, profileOf(t, 12, 4, 40), func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		g := &generator{t: t, rnd: rand.New(rand.NewSource(seed)), r: r, w: &world{limits: limits, pools: map[string]map[int]*mVersion{}, sequence: map[string]uint64{}}, seed: seed, reached: reached, counted: map[string]bool{}}
		for step := 0; step < 180; step++ {
			g.step()
			if step%20 == 19 {
				g.audit()
			}
		}
		g.audit()
	}
	least := map[string]int{"shuffled": 300, "billed": 60, "published/chargeback": 60, "published/showback": 60, "published/simulation": 60, "residual/" + BelowCapacity: 60, "residual/" + RecordedAlternative: 15, "settle/ok": 30,
		"state/staging": 200, "state/held": 60, "state/sealed": 100, "state/published": 150, "state/withdrawn": 100, "hold/" + WeightNotKnown: 20, "hold/" + ZeroDenominator: 20, "hold/" + OverCapacity: 20}
	for _, key := range []string{"pool/ok", "run/ok", "chunk/ok", "seal/ok", "publish/ok", "withdraw/ok"} {
		least[key] = 100
	}
	for _, key := range []string{"pool/" + kindOf(ErrInvalid), "pool/" + kindOf(ErrConflict), "pool/" + kindOf(ErrLimit), "run/" + kindOf(ErrInvalid), "run/" + kindOf(ErrNotFound), "run/" + kindOf(ErrConflict), "run/" + kindOf(ErrLimit),
		"chunk/" + kindOf(ErrInvalid), "chunk/" + kindOf(ErrConflict), "chunk/" + kindOf(ErrRejected), "chunk/" + kindOf(ErrLimit), "seal/" + kindOf(ErrRejected), "settle/" + kindOf(ErrInvalid), "settle/" + kindOf(ErrConflict), "settle/" + kindOf(ErrRejected),
		"publish/" + kindOf(ErrConflict), "publish/" + kindOf(ErrRejected), "withdraw/" + kindOf(ErrInvalid), "withdraw/" + kindOf(ErrConflict)} {
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
	weights := [][]Weight{{w("a", Organization, "60"), w("b", Organization, "25")}, {w("c", Organization, "10"), w("d", MSP, "3")}, {w("e", Unattributed, "1"), w("f", Organization, "0")}, {w("g", Organization, "7")}}
	reference := newHarness(t, nil)
	reference.pool(capped(1, 10, "1000000", "200"))
	reference.run(1, "ordered", Simulation, 4)
	for position, part := range weights {
		reference.stage(1, "ordered", position+1, part...)
	}
	reference.seal(1, "ordered")
	reference.publish(1, "ordered")
	want := reference.shares(1, "ordered")
	for round := 0; round < 40; round++ {
		r, err := NewRegistry(testLimits(), profileOf(t, 12, 4, 40), func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		h := &harness{t: t, r: r}
		h.pool(capped(1, 10, "1000000", "200"))
		h.pool(capped(2, 10, "500", ""))
		h.run(1, "run", Chargeback, 4)
		h.run(2, "rival", Chargeback, 1)
		h.stage(2, "rival", 1, w("a", Organization, "1"))
		h.seal(2, "rival")
		begin := make(chan struct{})
		var group sync.WaitGroup
		work := func(task func()) {
			group.Add(1)
			go func() {
				defer group.Done()
				<-begin
				task()
			}()
		}
		for twice := 0; twice < 2; twice++ {
			for position := 4; position >= 1; position-- {
				work(func() {
					if _, err := r.StageChunk("ws1", "shared-1", 1, "run", position, weights[position-1]); err != nil {
						t.Errorf("chunk %d: %v", position, err)
					}
				})
			}
		}
		for sealer := 0; sealer < 3; sealer++ {
			work(func() {
				state, err := r.Seal("ws1", "shared-1", 1, "run")
				if err == nil && state != Sealed {
					t.Errorf("sealed as %s", state)
				}
				if err != nil && err != ErrRejected {
					t.Errorf("seal: %v", err)
				}
			})
		}
		for reader := 0; reader < 3; reader++ {
			work(func() {
				for spin := 0; spin < 50; spin++ {
					view, err := r.Run("ws1", "shared-1", 1, "run")
					if err != nil || view.ByBearer != nil || view.Residual != "" || (view.State != Staging && view.State != Sealed) {
						t.Errorf("a run before publication: %+v %v", view, err)
						return
					}
					if _, err := r.Shares("ws1", "shared-1", 1, "run", "", 4); err != ErrRejected {
						t.Errorf("shares before publication: %v", err)
						return
					}
				}
			})
		}
		close(begin)
		group.Wait()
		if state := h.seal(1, "run"); state != Sealed {
			t.Fatalf("round %d: %s", round, state)
		}
		begin = make(chan struct{})
		published := make([]error, 2)
		work(func() { _, published[0] = r.Publish("ws1", "shared-1", 1, "run", approved) })
		work(func() { _, published[1] = r.Publish("ws1", "shared-1", 2, "rival", approved) })
		close(begin)
		group.Wait()
		if (published[0] == nil) == (published[1] == nil) || (published[0] != nil && published[0] != ErrRejected) || (published[1] != nil && published[1] != ErrRejected) {
			t.Fatalf("round %d: two chargeback runs of one period: %v %v", round, published[0], published[1])
		}
		const winner, version = "run", 1
		if published[1] == nil {
			if err := r.Withdraw("ws1", "shared-1", 2, "rival", ByOperator); err != nil {
				t.Fatal(err)
			}
			h.publish(1, "run")
		}
		if got := h.shares(1, "run"); got != want {
			t.Fatalf("round %d: chunks staged by several goroutines in any order:\n got %s\nwant %s", round, got, want)
		}
		begin = make(chan struct{})
		for reader := 0; reader < 4; reader++ {
			work(func() {
				gone := false
				var last uint64
				for spin := 0; spin < 200; spin++ {
					billed, found := r.Chargeback("ws1", "shared-1", from, to)
					page, err := r.Shares("ws1", "shared-1", version, winner, "", 4)
					if err != nil || (found != nil && found != ErrNotFound) || billed.Sequence < last || len(page.Items) != 4 {
						t.Errorf("reads: %+v %v %v", billed, found, err)
						return
					}
					last = billed.Sequence
					if found != nil {
						gone = true
					} else if gone || billed.Run != winner {
						t.Errorf("%+v is billed after the withdrawal was seen: %v", billed, gone)
						return
					}
					if gone && page.Sequence >= billed.Sequence && page.State != Withdrawn {
						t.Errorf("a page of %s after the withdrawal", page.State)
						return
					}
				}
			})
		}
		work(func() {
			for spin := 0; spin < round*10; spin++ {
				if _, err := r.Run("ws1", "shared-1", version, winner); err != nil {
					t.Errorf("run: %v", err)
				}
			}
			if err := r.Withdraw("ws1", "shared-1", version, winner, EnteredInError); err != nil {
				t.Errorf("withdraw: %v", err)
			}
		})
		close(begin)
		group.Wait()
		if _, err := r.Chargeback("ws1", "shared-1", from, to); err != ErrNotFound {
			t.Fatalf("round %d: %v", round, err)
		}
	}
}
