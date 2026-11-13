package fx

import (
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

type refRev struct {
	value *big.Rat
	unit  int
	state string
}

type refVersion struct {
	in        VersionInput
	effective time.Time
	approved  bool
	withdrawn bool
}

type model struct {
	obs      map[Key][]*refRev
	policies map[place][]*refVersion
	ids      map[place]string
	local    time.Time
}

func rat(text string) *big.Rat {
	made, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return made
}

func (m *model) give(k Key) (string, *refRev) {
	list := m.obs[k]
	if len(list) == 0 {
		return "none", nil
	}
	top := list[len(list)-1]
	switch top.state {
	case ObservationRecorded, ObservationApproved:
		return "value", top
	case ObservationVoid:
		return "void", nil
	}
	return "pending", nil
}

func (m *model) on(at place, date time.Time) *refVersion {
	var found *refVersion
	for _, v := range m.policies[at] {
		if !v.effective.After(date) && (found == nil || v.effective.After(found.effective) || (v.effective.Equal(found.effective) && v.in.Version > found.in.Version)) {
			found = v
		}
	}
	return found
}

func (m *model) legs(v VersionInput, n Need, day time.Time) []Key {
	d := dateText(day)
	switch v.Construction {
	case ConstructionDirect:
		return []Key{{v.RateSource, v.Series, n.Source, n.Target, d}}
	case ConstructionInverse:
		return []Key{{v.RateSource, v.Series, n.Target, n.Source, d}}
	}
	return []Key{{v.RateSource, v.Series, n.Source, v.Pivot, d}, {v.RateSource, v.Series, n.Target, v.Pivot, d}}
}

func roundRat(value *big.Rat, scale int, mode string) *big.Rat {
	factor := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	scaled := new(big.Rat).Mul(value, factor)
	floor := new(big.Int).Div(scaled.Num(), scaled.Denom())
	part := new(big.Rat).Sub(scaled, new(big.Rat).SetInt(floor))
	result := floor
	if part.Sign() != 0 {
		ceiling := new(big.Int).Add(floor, big.NewInt(1))
		half := part.Cmp(big.NewRat(1, 2))
		positive := scaled.Sign() > 0
		up := false
		switch mode {
		case "ceiling":
			up = true
		case "toward_zero":
			up = !positive
		case "away_from_zero":
			up = positive
		case "half_up":
			up = half > 0 || (half == 0 && positive)
		case "half_even":
			up = half > 0 || (half == 0 && floor.Bit(0) == 1)
		}
		if up {
			result = ceiling
		}
	}
	return new(big.Rat).Quo(new(big.Rat).SetInt(result), factor)
}

func (m *model) resolve(n Need) (string, string, *big.Rat, *big.Rat) {
	if n.Source == n.Target {
		return EntrySameCurrency, "", nil, nil
	}
	date, _ := parseDate(n.Date)
	if date.After(m.local) {
		return EntryHeld, ReasonDateNotReached, nil, nil
	}
	var chosen *refVersion
	pick := func(level, keyed string) *refVersion {
		return m.on(place{level, keyed, n.Purpose, n.Source, n.Target}, date)
	}
	held := ""
	if v := pick(LevelCustomer, n.Customer); v != nil && (v.withdrawn || v.in.Method != MethodEnded) {
		if v.withdrawn {
			held = ReasonWithdrawnInForce
		} else {
			chosen = v
		}
	} else if v := pick(LevelContract, n.Contract); v == nil || (!v.withdrawn && v.in.Method == MethodEnded) {
		held = ReasonContractNotStated
	} else if v.withdrawn {
		held = ReasonWithdrawnInForce
	} else if v.in.Method != MethodDefaultAllowed {
		chosen = v
	} else if d := pick(LevelMSP, ""); d == nil {
		held = ReasonNoDefaultPolicy
	} else if d.withdrawn {
		held = ReasonWithdrawnInForce
	} else {
		chosen = d
	}
	if held != "" {
		return EntryHeld, held, nil, nil
	}
	v := chosen.in
	var quoted *big.Rat
	unit := 1
	switch v.Method {
	case MethodFixed:
		if !chosen.approved {
			return EntryHeld, ReasonAwaitingApproval, nil, nil
		}
		quoted, unit = rat(v.Value), v.Unit
	case MethodDated:
		window := 0
		if v.Rule == RulePreviousRecorded && date.Before(m.local) {
			window = v.MaxDays
		}
		legs := len(m.legs(v, n, date))
		seen := make([]bool, legs)
		for back := 0; back <= window && quoted == nil; back++ {
			day := date.AddDate(0, 0, -back)
			states := []string{}
			found := []*refRev{}
			for index, k := range m.legs(v, n, day) {
				state, rev := m.give(k)
				states = append(states, state)
				if rev != nil {
					seen[index] = true
					found = append(found, rev)
				}
			}
			if len(found) < legs && !date.Before(m.local) {
				return EntryHeld, ReasonDateNotReached, nil, nil
			}
			for _, state := range states {
				if state == "pending" {
					return EntryHeld, ReasonAwaitingApproval, nil, nil
				}
			}
			if len(found) == legs {
				if legs == 2 {
					a := new(big.Rat).Quo(found[0].value, big.NewRat(int64(found[0].unit), 1))
					b := new(big.Rat).Quo(found[1].value, big.NewRat(int64(found[1].unit), 1))
					quoted = new(big.Rat).Quo(a, b)
				} else {
					quoted, unit = found[0].value, found[0].unit
				}
			}
		}
		if quoted == nil {
			all := true
			for _, s := range seen {
				all = all && s
			}
			if all && legs == 2 {
				return EntryHeld, ReasonLegsNotOnOneDate, nil, nil
			}
			return EntryHeld, ReasonNoRateRecorded, nil, nil
		}
	default:
		begin := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
		if v.Month == MonthPrevious {
			begin = begin.AddDate(0, -1, 0)
		}
		end := begin.AddDate(0, 1, 0)
		if m.local.Before(end) {
			return EntryHeld, ReasonMonthNotEnded, nil, nil
		}
		legs := len(m.legs(v, n, begin))
		sums := []*big.Rat{new(big.Rat), new(big.Rat)}
		counts, largest := []int{0, 0}, []int{0, 0}
		for day := begin; day.Before(end); day = day.AddDate(0, 0, 1) {
			for index, k := range m.legs(v, n, day) {
				state, rev := m.give(k)
				if state == "pending" {
					return EntryHeld, ReasonAwaitingApproval, nil, nil
				}
				if rev != nil {
					sums[index].Add(sums[index], new(big.Rat).Quo(rev.value, big.NewRat(int64(rev.unit), 1)))
					counts[index]++
					if rev.unit > largest[index] {
						largest[index] = rev.unit
					}
				}
			}
		}
		for index := 0; index < legs; index++ {
			if counts[index] < v.LeastDays {
				return EntryHeld, ReasonNotEnoughDays, nil, nil
			}
		}
		mean := func(index int) *big.Rat { return new(big.Rat).Quo(sums[index], big.NewRat(int64(counts[index]), 1)) }
		if legs == 2 {
			quoted = new(big.Rat).Quo(mean(0), mean(1))
		} else {
			quoted, unit = new(big.Rat).Mul(mean(0), big.NewRat(int64(largest[0]), 1)), largest[0]
		}
	}
	factor := new(big.Rat).Add(big.NewRat(1, 1), rat(v.Spread))
	exactQuote := new(big.Rat).Mul(quoted, factor)
	inverse := v.Construction == ConstructionInverse
	if inverse {
		exactQuote = new(big.Rat).Quo(quoted, factor)
	}
	rounded := roundRat(exactQuote, v.QuoteScale, v.QuoteMode)
	if rounded.Sign() <= 0 {
		return EntryHeld, ReasonRateRoundsToZero, nil, nil
	}
	distance := new(big.Rat).Abs(new(big.Rat).Sub(rounded, exactQuote))
	smaller := rounded
	if exactQuote.Cmp(rounded) < 0 {
		smaller = exactQuote
	}
	if new(big.Rat).Quo(distance, smaller).Cmp(rat(v.Tolerance)) > 0 {
		return EntryHeld, ReasonRatePrecisionLost, nil, nil
	}
	per := big.NewRat(int64(unit), 1)
	if inverse {
		return EntryRate, chosenReason(m, n, chosen), new(big.Rat).Quo(per, rounded), rat(v.Fee)
	}
	return EntryRate, chosenReason(m, n, chosen), new(big.Rat).Quo(rounded, per), rat(v.Fee)
}

func chosenReason(m *model, n Need, chosen *refVersion) string {
	for _, level := range []string{LevelCustomer, LevelContract, LevelMSP} {
		keyed := map[string]string{LevelCustomer: n.Customer, LevelContract: n.Contract, LevelMSP: ""}[level]
		for _, v := range m.policies[place{level, keyed, n.Purpose, n.Source, n.Target}] {
			if v == chosen {
				return map[string]string{LevelCustomer: ReasonCustomerException, LevelContract: ReasonContract, LevelMSP: ReasonMSPDefault}[level]
			}
		}
	}
	return "?"
}

func ratText(value *big.Rat) string {
	if value.IsInt() {
		return value.Num().String()
	}
	return value.Num().String() + "/" + value.Denom().String()
}

var pairs = [][2]string{{"USD", "KRW"}, {"EUR", "KRW"}, {"JPY", "KRW"}, {"KRW", "USD"}}

func TestRandomFactsAgainstAReference(t *testing.T) {
	random := rand.New(rand.NewSource(2082))
	met := map[string]int{}
	modes := []string{"toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"}
	values := map[[2]string][]string{{"USD", "KRW"}: {"1350", "1349.5", "1351.25"}, {"EUR", "KRW"}: {"1450", "1461.7"}, {"JPY", "KRW"}: {"905.25", "9.0525"},
		{"EUR", "USD"}: {"1.08", "1.0912"}, {"KRW", "USD"}: {"0.74", "0.7407"}}
	keys := [][2]string{{"USD", "KRW"}, {"EUR", "KRW"}, {"JPY", "KRW"}, {"EUR", "USD"}, {"KRW", "USD"}}
	for round := 0; round < 400; round++ {
		limits := testLimits()
		limits.Keys, limits.Revisions, limits.Policies, limits.Needs, limits.Lines, limits.Snapshots = 1000, 4, 40, 40, 40, 4
		r, err := NewRegistry(testProfile(t), limits, func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		clean := random.Intn(3) == 0
		m := &model{obs: map[Key][]*refRev{}, policies: map[place][]*refVersion{}, ids: map[place]string{}, local: time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)}
		for day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC); day.Before(m.local.AddDate(0, 0, 1)); day = day.AddDate(0, 0, 1) {
			for _, pair := range keys {
				k := Key{"bank", "deal", pair[0], pair[1], dateText(day)}
				for revision := 1; revision <= random.Intn(4) && random.Intn(5) != 0; revision++ {
					unit := 1
					if pair[0] == "JPY" || pair[0] == "KRW" {
						unit = []int{1, 100, 1000}[random.Intn(3)]
					}
					value := values[pair][random.Intn(len(values[pair]))]
					origin := []string{OriginFeed, OriginFeed, OriginFeed, OriginManual}[random.Intn(4)]
					if clean {
						origin = OriginFeed
					}
					evidence := ""
					if origin != OriginFeed {
						evidence = "doc"
					}
					if _, err := r.RecordObservation("ws", ObservationInput{Key: k, Revision: revision, Unit: unit, Value: value, Origin: origin, Evidence: evidence, Recorder: "rec"}); err != nil {
						t.Fatalf("record: %v", err)
					}
					rev := &refRev{value: rat(value), unit: unit, state: ObservationRecorded}
					if origin != OriginFeed {
						rev.state = ObservationAwaiting
						if random.Intn(3) != 0 {
							r.RecordObservationApproval("ws", k, revision, "boss", "ok")
							rev.state = ObservationApproved
						}
					}
					if !clean && random.Intn(12) == 0 {
						r.VoidObservation("ws", k, revision, "rec", "doc", "mistake")
						rev.state = ObservationVoidAwaiting
						if random.Intn(2) == 0 {
							r.RecordVoidApproval("ws", k, revision, "boss", "ok")
							rev.state = ObservationVoid
						}
					}
					m.obs[k] = append(m.obs[k], rev)
				}
			}
		}
		for _, pair := range pairs {
			for _, level := range []string{LevelCustomer, LevelContract, LevelMSP} {
				if random.Intn(3) == 0 {
					continue
				}
				keyed := map[string]string{LevelCustomer: "cust-1", LevelContract: "con-1", LevelMSP: ""}[level]
				at := place{level, keyed, "billing", pair[0], pair[1]}
				methods := []string{MethodDated, MethodDated, MethodMonthlyAverage, MethodFixed}
				if level != LevelMSP {
					methods = append(methods, MethodEnded)
				}
				if level == LevelContract {
					methods = append(methods, MethodDefaultAllowed, MethodDefaultAllowed)
				}
				id := fmt.Sprintf("%s-%s-%s", level, pair[0], pair[1])
				for version := 1; version <= 1+random.Intn(2); version++ {
					v := VersionInput{Version: version, Effective: []string{"2026-01-01", "2026-11-05"}[version-1], Author: "author", Method: methods[random.Intn(len(methods))]}
					if v.Method != MethodEnded && v.Method != MethodDefaultAllowed {
						v.Spread, v.Fee = []string{"0", "0.005", "0.01"}[random.Intn(3)], []string{"0", "0.1"}[random.Intn(2)]
						v.QuoteScale, v.QuoteMode = []int{0, 2, 4}[random.Intn(3)], modes[random.Intn(6)]
						v.Tolerance, v.Quantum, v.AmountMode, v.Order = []string{"0.0001", "0.01"}[random.Intn(2)], []string{"1", "0.01"}[random.Intn(2)], modes[random.Intn(6)], Order
					}
					construction := []string{ConstructionDirect, ConstructionInverse, ConstructionCross}
					switch v.Method {
					case MethodFixed:
						v.Construction, v.Unit, v.Value, v.Evidence = []string{ConstructionDirect, ConstructionInverse}[random.Intn(2)], 1, "1300", "doc"
					case MethodDated, MethodMonthlyAverage:
						v.RateSource, v.Series = "bank", "deal"
						v.Construction = ConstructionDirect
						if pair == [2]string{"EUR", "KRW"} {
							v.Construction = construction[2*random.Intn(2)]
						}
						if pair == [2]string{"KRW", "USD"} {
							v.Construction = construction[random.Intn(2)]
						}
						if v.Construction == ConstructionCross {
							v.Pivot = "USD"
						}
						if v.Method == MethodDated {
							v.Rule = []string{RuleHold, RulePreviousRecorded}[random.Intn(2)]
							if v.Rule == RulePreviousRecorded {
								v.MaxDays = 1 + random.Intn(4)
							}
						} else {
							v.Month, v.Averaging, v.LeastDays = []string{MonthSame, MonthPrevious}[random.Intn(2)], Averaging, 1+random.Intn(12)
						}
					}
					if _, err := r.RecordPolicyVersion("ws", id, PolicyAttributes{Level: level, Key: keyed, Purpose: "billing", Source: pair[0], Target: pair[1]}, v); err != nil {
						t.Fatalf("policy %+v: %v", v, err)
					}
					effective, _ := parseDate(v.Effective)
					made := &refVersion{in: v, effective: effective}
					if v.Method == MethodFixed && random.Intn(3) != 0 {
						r.RecordPolicyApproval("ws", id, version, "boss", "ok")
						made.approved = true
					}
					if random.Intn(8) == 0 {
						r.WithdrawPolicyVersion("ws", id, version, "boss", "mistake")
						made.withdrawn = true
					}
					made.in = v
					m.policies[at] = append(m.policies[at], made)
				}
			}
		}
		needs := []Need{}
		for index := 0; index < 30; index++ {
			pair := pairs[random.Intn(len(pairs))]
			if random.Intn(15) == 0 {
				pair = [2]string{"KRW", "KRW"}
			}
			day := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -random.Intn(30))
			needs = append(needs, Need{ID: fmt.Sprintf("n%d", index), Purpose: "billing", Source: pair[0], Target: pair[1], Customer: "cust-1", Contract: "con-1", Date: dateText(day)})
		}
		snap, err := r.Prepare("ws", "run", needs)
		if err != nil {
			t.Fatalf("round %d prepare: %v", round, err)
		}
		lines := []Line{}
		for index, n := range needs {
			kindOf, reasonOf, sales, fee := m.resolve(n)
			e := snap.Entries[index]
			if e.Kind != kindOf || (kindOf != EntrySameCurrency && e.Reason != reasonOf) {
				t.Fatalf("round %d need %+v: %s %s, the reference %s %s\n%+v", round, n, e.Kind, e.Reason, kindOf, reasonOf, e)
			}
			met[kindOf+" "+reasonOf]++
			for _, o := range e.Observations {
				if o.Revision != len(m.obs[o.Key]) {
					t.Fatalf("round %d need %+v: revision %d of %d used", round, n, o.Revision, len(m.obs[o.Key]))
				}
			}
			if e.DaysElapsed > e.MaxDays || (e.Method == MethodDated && e.Construction == ConstructionCross && kindOf == EntryRate && e.Observations[0].Key.Date != e.Observations[1].Key.Date) {
				t.Fatalf("round %d need %+v: dates %+v", round, n, e)
			}
			if kindOf == EntryRate {
				if e.SalesRate != ratText(sales) {
					t.Fatalf("round %d need %+v: rate %s, the reference %s", round, n, e.SalesRate, ratText(sales))
				}
				met[e.Method+" "+e.Construction]++
				if e.DaysElapsed > 0 {
					met["fallback"]++
				}
				if e.VoidsPassed > 0 {
					met["void passed"]++
				}
				amount := []string{"1", "-12.5", "1234.56", "0.07"}[random.Intn(4)]
				lines = append(lines, line("l"+n.ID, n.ID, amount, n.Source, StateSource))
				exactAmount := new(big.Rat).Mul(rat(amount), sales)
				want := roundRat(exactAmount, quanta[e.Quantum], e.AmountMode)
				wantFee := roundRat(new(big.Rat).Mul(exactAmount, fee), quanta[e.Quantum], e.AmountMode)
				expectations[snapKey(round, "l"+n.ID)] = [2]string{ratText(want), ratText(wantFee)}
			}
		}
		out, err := r.Convert("ws", "run", lines)
		if err != nil {
			t.Fatalf("round %d convert: %v", round, err)
		}
		for _, l := range out.Lines {
			want := expectations[snapKey(round, l.Line)]
			if ratText(rat(l.Amount)) != want[0] || ratText(rat(l.Fee)) != want[1] || l.Kind != ResultConverted || ratText(rat(l.Total)) != ratText(new(big.Rat).Add(rat(l.Amount), rat(l.Fee))) {
				t.Fatalf("round %d line %s: %s fee %s, the reference %v", round, l.Line, l.Amount, l.Fee, want)
			}
			met["converted"]++
		}
		again, _ := r.Snapshot("ws", "run")
		if again.ContentDigest != snap.ContentDigest {
			t.Fatal("a snapshot changed")
		}
	}
	required := []string{
		"rate customer_exception", "rate contract", "rate msp_default",
		"held contract_not_stated", "held no_default_policy", "held withdrawn_in_force", "held date_not_reached", "held month_not_ended",
		"held awaiting_approval", "held no_rate_recorded", "held rate_rounds_to_zero", "held legs_not_on_one_date", "held not_enough_days", "held rate_precision_lost",
		"same_currency ", "dated direct", "dated inverse", "dated cross", "monthly_average direct", "monthly_average inverse", "monthly_average cross",
		"fixed direct", "fixed inverse", "fallback", "void passed", "converted",
	}
	names := []string{}
	for name := range met {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range required {
		if met[name] < 3 {
			t.Errorf("%s met %d times; met: %v", name, met[name], names)
		}
	}
}

var expectations = map[string][2]string{}

func snapKey(round int, id string) string { return fmt.Sprintf("%d/%s", round, id) }
func TestPreparingBesideARevision(t *testing.T) {
	for round := 0; round < 100; round++ {
		r, _ := open(t)
		policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
		feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
		gate := make(chan struct{})
		var group sync.WaitGroup
		var snap Snapshot
		group.Add(2)
		go func() {
			defer group.Done()
			<-gate
			snap, _ = r.Prepare("ws", "run", []Need{need("n", "USD", "KRW", "2026-11-10")})
		}()
		go func() {
			defer group.Done()
			<-gate
			r.RecordObservation("ws", ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 2, Unit: 1, Value: "1360", Origin: OriginFeed, Recorder: "loader"})
		}()
		close(gate)
		group.Wait()
		e := snap.Entries[0]
		if !(e.SalesRate == "1350" && e.Observations[0].Revision == 1) && !(e.SalesRate == "1360" && e.Observations[0].Revision == 2) {
			t.Fatalf("a snapshot of half a revision: %+v", e)
		}
	}
}

func TestConversionsBesideOneAnother(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	prepare(t, r, "run", need("n", "USD", "KRW", "2026-11-10"))
	gate := make(chan struct{})
	var group sync.WaitGroup
	results := make([]string, 8)
	for index := range results {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-gate
			for round := 0; round < 50; round++ {
				out, err := r.Convert("ws", "run", []Line{line("a", "n", "900", "USD", StateSource)})
				results[index] = kind(err) + " " + shown(out) + " " + out.Digest
			}
		}(index)
	}
	group.Add(1)
	go func() {
		defer group.Done()
		<-gate
		r.RecordObservation("ws", ObservationInput{Key: key("USD", "KRW", "2026-11-10"), Revision: 2, Unit: 1, Value: "1360", Origin: OriginFeed, Recorder: "loader"})
		r.Prepare("ws", "other", []Need{need("n", "USD", "KRW", "2026-11-10")})
	}()
	close(gate)
	group.Wait()
	snap, _ := r.Snapshot("ws", "run")
	for index, result := range results {
		if result != "ok a=1215000 KRW fee 0 total 1215000 quoted 1215000 "+snap.Digest {
			t.Fatalf("goroutine %d: %s", index, result)
		}
	}
}
