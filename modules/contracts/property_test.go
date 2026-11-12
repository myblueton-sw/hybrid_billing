package contracts

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type modelDay struct {
	state    State
	revision uint64
}

func calendar(year, month, dayOfMonth int) time.Time {
	return time.Date(year, time.Month(month), dayOfMonth, 0, 0, 0, 0, time.UTC)
}

func toDay(t testing.TB, date time.Time) Day {
	t.Helper()
	made, err := NewDay(date.Year(), int(date.Month()), date.Day())
	if err != nil {
		t.Fatal(err)
	}
	return made
}

type history struct {
	events []Event
	dates  map[string]time.Time
}

func generate(t testing.TB, random *rand.Rand, origin time.Time, steps int) history {
	made := history{dates: map[string]time.Time{}}
	cursor := origin.AddDate(0, 0, random.Intn(40))
	state, revision := NotStarted, uint64(0)
	for index := 0; index < steps; index++ {
		kind := Kind("")
		next := revision
		switch state {
		case NotStarted:
			kind, next = Start, uint64(1+random.Intn(3))
		case Active:
			kind = []Kind{Suspend, Change, Renew, End, Change, Suspend}[random.Intn(6)]
		case Suspended:
			kind = []Kind{Resume, Change, Renew, End, Resume, Resume}[random.Intn(6)]
		case Ended:
			return made
		}
		if kind == Change {
			next = revision%3 + 1
		}
		event := Event{ID: fmt.Sprintf("e%d", index), Kind: kind, From: toDay(t, cursor)}
		if kind == Start || kind == Change {
			event.Revision = next
			revision = next
		}
		switch kind {
		case Start, Resume:
			state = Active
		case Suspend:
			state = Suspended
		case End:
			state = Ended
		}
		made.events = append(made.events, event)
		made.dates[event.ID] = cursor
		cursor = cursor.AddDate(0, 0, 1+random.Intn(25))
	}
	return made
}

func (h history) on(date time.Time) modelDay {
	current := modelDay{state: NotStarted}
	for _, event := range h.events {
		if h.dates[event.ID].After(date) {
			continue
		}
		switch event.Kind {
		case Start:
			current = modelDay{Active, event.Revision}
		case Change:
			current.revision = event.Revision
		case Suspend:
			current.state = Suspended
		case Resume:
			current.state = Active
		case End:
			current = modelDay{Ended, 0}
		}
	}
	return current
}

func roundRat(value *big.Rat, scale int, mode decimal.Mode) *big.Rat {
	shift := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	scaled := new(big.Rat).Mul(value, new(big.Rat).SetInt(shift))
	floor := new(big.Int).Div(scaled.Num(), scaled.Denom())
	fraction := new(big.Rat).Sub(scaled, new(big.Rat).SetInt(floor))
	exact := fraction.Sign() == 0
	half := fraction.Cmp(big.NewRat(1, 2))
	up := false
	switch mode {
	case decimal.Floor:
	case decimal.Ceiling:
		up = !exact
	case decimal.TowardZero:
		up = !exact && scaled.Sign() < 0
	case decimal.AwayFromZero:
		up = !exact && scaled.Sign() > 0
	case decimal.HalfUp:
		up = half > 0 || (half == 0 && scaled.Sign() > 0)
	case decimal.HalfEven:
		up = half > 0 || (half == 0 && floor.Bit(0) == 1)
	}
	if up {
		floor.Add(floor, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(floor, shift)
}

func rat(t testing.TB, value decimal.Decimal) *big.Rat {
	t.Helper()
	parsed, ok := new(big.Rat).SetString(value.String())
	if !ok {
		t.Fatalf("%s is not a number", value)
	}
	return parsed
}

func TestTheRoundingOfTheModel(t *testing.T) {
	for _, c := range []struct {
		value string
		mode  decimal.Mode
		want  string
	}{{"5/2", decimal.HalfUp, "3"}, {"-5/2", decimal.HalfUp, "-3"}, {"5/2", decimal.HalfEven, "2"}, {"7/2", decimal.HalfEven, "4"}, {"-5/2", decimal.HalfEven, "-2"},
		{"-7/2", decimal.HalfEven, "-4"}, {"7/3", decimal.Floor, "2"}, {"-7/3", decimal.Floor, "-3"}, {"7/3", decimal.Ceiling, "3"}, {"-7/3", decimal.Ceiling, "-2"},
		{"7/3", decimal.TowardZero, "2"}, {"-7/3", decimal.TowardZero, "-2"}, {"7/3", decimal.AwayFromZero, "3"}, {"-7/3", decimal.AwayFromZero, "-3"},
		{"4", decimal.Ceiling, "4"}, {"-4", decimal.AwayFromZero, "-4"}, {"8/3", decimal.HalfUp, "3"}, {"-8/3", decimal.HalfEven, "-3"}} {
		value, _ := new(big.Rat).SetString(c.value)
		if got := roundRat(value, 0, c.mode).RatString(); got != c.want {
			t.Errorf("%s by %s: %s; want %s", c.value, c.mode, got, c.want)
		}
	}
	if got := roundRat(big.NewRat(2, 3), 2, decimal.HalfUp).FloatString(2); got != "0.67" {
		t.Errorf("two thirds to the cent: %s", got)
	}
}

func refine(t testing.TB, random *rand.Rand, segments []Segment) []Segment {
	refined := []Segment{}
	for _, segment := range segments {
		for segment.Interval.Days() > 1 && random.Intn(2) == 0 {
			cut, err := segment.Interval.start.Add(1 + random.Intn(segment.Interval.Days()-1))
			if err != nil {
				t.Fatal(err)
			}
			refined = append(refined, Segment{Interval: Interval{start: segment.Interval.start, end: cut}, State: segment.State, Revision: segment.Revision})
			segment.Interval.start = cut
		}
		refined = append(refined, segment)
	}
	return refined
}

func TestSegmentsAndProrationAgreeWithADayByDayModel(t *testing.T) {
	random := rand.New(rand.NewSource(178))
	p := profile(t)
	modes := []decimal.Mode{decimal.TowardZero, decimal.AwayFromZero, decimal.Floor, decimal.Ceiling, decimal.HalfUp, decimal.HalfEven}
	reached := map[string]int{}
	origin := calendar(2027, 11, 20)
	for round := 0; round < 1500; round++ {
		lived := generate(t, random, origin, 1+random.Intn(8))
		earlier := history{events: lived.events[:random.Intn(len(lived.events)+1)], dates: lived.dates}
		first := origin.AddDate(0, 0, random.Intn(160)-20)
		length := []int{1, 7, 28, 29, 30, 31, 90, 365, 366}[random.Intn(9)]
		period := Interval{start: toDay(t, first), end: toDay(t, first.AddDate(0, 0, length))}
		if period.Days() != length {
			t.Fatalf("the period from %s has %d days; the calendar says %d", first.Format("2006-01-02"), period.Days(), length)
		}
		scale := random.Intn(3)
		sign := []int64{1, 1, 1, -1}[random.Intn(4)]
		amounts := map[uint64]decimal.Decimal{}
		model := map[uint64]*big.Rat{}
		unit := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
		for revision := uint64(1); revision <= 3; revision++ {
			units := sign * random.Int63n(100000000)
			switch random.Intn(12) {
			case 0:
				units = 0
			case 1:
				units = sign * random.Int63n(40)
			}
			model[revision] = new(big.Rat).Mul(new(big.Rat).SetInt64(units), unit)
			amounts[revision] = dec(t, model[revision].FloatString(scale))
		}
		basis := Basis{WorkspaceID: "w1", ContractID: "c1", ChargeID: "base", Period: period, Point: 2, Days: ActualDays,
			Suspended: []SuspendedRule{BillSuspended, SkipSuspended}[random.Intn(2)], Scale: scale, Mode: modes[random.Intn(len(modes))]}
		billable := func(h history, date time.Time) (*big.Rat, bool) {
			at := h.on(date)
			if at.state == Active || (at.state == Suspended && basis.Suspended == BillSuspended) {
				return new(big.Rat).Quo(model[at.revision], new(big.Rat).SetInt64(int64(length))), true
			}
			return nil, false
		}
		exactTotal := func(h history) *big.Rat {
			sum := new(big.Rat)
			for offset := 0; offset < length; offset++ {
				if share, billed := billable(h, first.AddDate(0, 0, offset)); billed {
					sum.Add(sum, share)
				}
			}
			return sum
		}
		segments, err := Segments(lived.events, period)
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		covered := 0
		for index, segment := range segments {
			if index > 0 && (segments[index-1].Interval.end != segment.Interval.start || (segments[index-1].State == segment.State && segments[index-1].Revision == segment.Revision)) {
				t.Fatalf("round %d: segments %s", round, shape(segments))
			}
			for offset := 0; offset < segment.Interval.Days(); offset++ {
				want := lived.on(first.AddDate(0, 0, covered))
				if segment.State != want.state || segment.Revision != want.revision {
					t.Fatalf("round %d: day %s is %s %d in the segments and %s %d in the model", round, first.AddDate(0, 0, covered).Format("2006-01-02"), segment.State, segment.Revision, want.state, want.revision)
				}
				covered++
			}
		}
		if covered != length || segments[0].Interval.start != period.start || segments[len(segments)-1].Interval.end != period.end {
			t.Fatalf("round %d: the segments cover %d of %d days", round, covered, length)
		}
		result, err := Prorate(p, basis, amounts, segments)
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		want := roundRat(exactTotal(lived), scale, basis.Mode)
		if rat(t, result.Total).Cmp(want) != 0 {
			t.Fatalf("round %d: total %s; the model says %s", round, result.Total, want.FloatString(scale))
		}
		sum, perRevision, offset, next := new(big.Rat), map[uint64]*big.Rat{}, 0, 0
		running, previous := new(big.Rat), new(big.Rat)
		for _, segment := range segments {
			exact, billed := new(big.Rat), false
			for inner := 0; inner < segment.Interval.Days(); inner++ {
				if share, counted := billable(lived, first.AddDate(0, 0, offset+inner)); counted {
					exact.Add(exact, share)
					billed = true
				}
			}
			offset += segment.Interval.Days()
			if !billed {
				continue
			}
			line := result.Lines[next]
			next++
			amount := rat(t, line.Amount)
			if line.Interval != segment.Interval || line.Revision != segment.Revision || line.Days != segment.Interval.Days() {
				t.Fatalf("round %d: line %+v for segment %+v", round, line, segment)
			}
			if amount.Sign()*exact.Sign() < 0 || new(big.Rat).Abs(new(big.Rat).Sub(amount, exact)).Cmp(unit) > 0 {
				t.Fatalf("round %d: line %s for the exact share %s", round, line.Amount, exact.FloatString(6))
			}
			if amount.Sign() == 0 && exact.Sign() != 0 {
				reached["a zero line for a share"]++
			}
			running.Add(running, exact)
			pinned := roundRat(running, scale, basis.Mode)
			if amount.Cmp(new(big.Rat).Sub(pinned, previous)) != 0 {
				t.Fatalf("round %d: line %s; the model's cumulative rounding gives %s", round, line.Amount, new(big.Rat).Sub(pinned, previous).FloatString(scale))
			}
			previous = pinned
			sum.Add(sum, amount)
			if perRevision[line.Revision] == nil {
				perRevision[line.Revision] = new(big.Rat)
			}
			perRevision[line.Revision].Add(perRevision[line.Revision], amount)
		}
		if next != len(result.Lines) || sum.Cmp(rat(t, result.Total)) != 0 || len(perRevision) != len(result.Revisions) {
			t.Fatalf("round %d: %d lines summing to %s for the total %s", round, len(result.Lines), sum.FloatString(scale), result.Total)
		}
		for _, entry := range result.Revisions {
			if perRevision[entry.Revision] == nil || perRevision[entry.Revision].Cmp(rat(t, entry.Total)) != 0 || rat(t, entry.Price).Cmp(model[entry.Revision]) != 0 {
				t.Fatalf("round %d: the total of terms %d", round, entry.Revision)
			}
		}
		refined, err := Prorate(p, basis, amounts, refine(t, random, segments))
		if err != nil || !refined.Total.Equal(result.Total) {
			t.Fatalf("round %d: refining the segments changed the total from %s to %s (%v)", round, result.Total, refined.Total, err)
		}
		if len(refined.Lines) > len(result.Lines) {
			reached["refined"]++
		}
		if len(segments) == 1 && segments[0].State == Active && !result.Total.Equal(amounts[segments[0].Revision]) {
			t.Fatalf("round %d: a full active period bills %s for the price %s", round, result.Total, amounts[segments[0].Revision])
		}
		beforeSegments, err := Segments(earlier.events, period)
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		known := basis
		known.Point = 1
		before, err := Prorate(p, known, amounts, beforeSegments)
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		difference, err := Correction(p, before, result)
		wantDifference := new(big.Rat).Sub(want, roundRat(exactTotal(earlier), scale, basis.Mode))
		if err != nil || rat(t, difference.Total).Cmp(wantDifference) != 0 {
			t.Fatalf("round %d: correction %s; the model says %s (%v)", round, difference.Total, wantDifference.FloatString(scale), err)
		}
		parts := new(big.Rat)
		for _, entry := range difference.Revisions {
			parts.Add(parts, rat(t, entry.Amount))
		}
		if parts.Cmp(wantDifference) != 0 {
			t.Fatalf("round %d: the differences per terms sum to %s", round, parts.FloatString(scale))
		}
		reached[fmt.Sprintf("%d segments", min(len(segments), 4))]++
		reached["mode "+basis.Mode.String()]++
		reached[string(basis.Suspended)]++
		if sign < 0 {
			reached["credits"]++
		}
		if len(result.Lines) == 0 {
			reached["nothing billed"]++
		}
		if len(result.Revisions) > 1 {
			reached["several terms billed"]++
		}
		if difference.Total.Sign() != 0 {
			reached["a correction that is not zero"]++
		}
		if len(segments) == 1 && segments[0].State == Active {
			reached["a full active period"]++
		}
	}
	for _, name := range []string{"1 segments", "2 segments", "3 segments", "4 segments", "mode floor", "mode ceiling", "mode half_even", "mode half_up", "mode toward_zero", "mode away_from_zero",
		"bill_suspended", "skip_suspended", "credits", "nothing billed", "several terms billed", "a correction that is not zero", "a full active period", "refined", "a zero line for a share"} {
		floor := 20
		if name == "a zero line for a share" {
			floor = 1
		}
		if reached[name] < floor {
			t.Errorf("%s was reached %d times", name, reached[name])
		}
	}
	t.Logf("%v", reached)
}

type snapshot struct {
	life       []Event
	settlement Settlement
	position   Position
}

func TestRandomWritesKeepTheInvariantAndThePast(t *testing.T) {
	random := rand.New(rand.NewSource(1780))
	probe := day(t, "2026-06-15")
	reached := map[string]int{}
	for round := 0; round < 450; round++ {
		b := open(t, Limits{Contracts: 2, Writes: 40, Reserve: 3, Batch: 5})
		past := []snapshot{}
		serial := 0
		for step := 0; step < 60; step++ {
			current := b.sequence()
			held, _ := b.r.Lifecycle("w1", "c1", current)
			var err error
			kind := ""
			switch choice := random.Intn(10); {
			case choice < 5:
				kind = "batch"
				withdrawals, events := []string{}, []Event{}
				for _, event := range held {
					if random.Intn(6) == 0 {
						withdrawals = append(withdrawals, event.ID)
					}
				}
				for count := random.Intn(3); count > 0; count-- {
					serial++
					made, _ := NewDay(2026, 6, 1+random.Intn(28))
					event := Event{ID: fmt.Sprintf("e%d", serial), Kind: []Kind{Start, Change, Renew, Suspend, Resume, End, End}[random.Intn(7)], From: made}
					if event.Kind == Start || event.Kind == Change {
						event.Revision = uint64(1 + random.Intn(2))
					}
					events = append(events, event)
				}
				cancel := CancelReason("")
				if random.Intn(3) == 0 {
					cancel = RecordedInError
				}
				_, err = b.r.Amend("w1", "c1", current, b.next(), withdrawals, events, cancel)
			case choice < 6:
				kind = "missing"
				err = b.missing(fmt.Sprintf("gap-%d-%d", round, step), NotAttempted)
			case choice < 8:
				kind = "secured"
				resolves := []string{}
				view, _ := b.r.Settlement("w1", "c1", current)
				for _, kept := range view.Evidence {
					if kept.Evidence.Kind == Missing && !kept.Resolved && random.Intn(4) != 0 {
						resolves = append(resolves, kept.Write)
					}
				}
				err = b.secured(fmt.Sprintf("2026-06-%02d", 1+random.Intn(30)), resolves...)
			case choice < 9:
				kind = "settle"
				err = b.settle(fmt.Sprintf("2026-06-%02d", 1+random.Intn(30)))
			default:
				kind = "reopen"
				err = b.reopen(LateSource)
			}
			after := b.sequence()
			if err == nil {
				reached[kind+" written"]++
				if after != current+1 {
					t.Fatalf("round %d: a write moved the sequence from %d to %d", round, current, after)
				}
			} else {
				reached[kind+" refused"]++
				if after != current {
					t.Fatalf("round %d: a refused %s moved the sequence", round, kind)
				}
				continue
			}
			b.check()
			for at := len(past); uint64(at) <= after; at++ {
				life, _ := b.r.Lifecycle("w1", "c1", uint64(at))
				view, _ := b.r.Settlement("w1", "c1", uint64(at))
				position, _ := b.r.StateOn("w1", "c1", uint64(at), probe)
				past = append(past, snapshot{life, view, position})
			}
			if view := past[after].settlement; view.State == Settled && kind == "settle" {
				reached["settled"]++
				_, over := ended(past[after].life)
				for _, kept := range view.Evidence {
					if kept.Evidence.Kind == Missing && !kept.Resolved {
						t.Fatalf("round %d: settled with an open gap", round)
					}
				}
				if over {
					records, _ := b.r.Records("w1", "c1")
					settling := records[len(records)-1]
					lastChange := uint64(0)
					for _, record := range records {
						if record.Kind == WriteBatch || record.Kind == WriteReopen {
							lastChange = record.Sequence
						}
					}
					last := past[after].life[len(past[after].life)-1].From
					if len(view.Evidence) == 0 {
						t.Fatalf("round %d: an ended contract settled without evidence", round)
					}
					proof := view.Evidence[len(view.Evidence)-1]
					if proof.Evidence.Kind != Secured || proof.Sequence <= lastChange || proof.Evidence.CoveredUntil.Before(last) || settling.AsOf.Before(last) || settling.Kind != WriteSettle {
						t.Fatalf("round %d: settled on %+v after the last change at %d for the end %s as of %s", round, proof, lastChange, last, settling.AsOf)
					}
					reached["settled after an end"]++
				}
			}
		}
		for at, kept := range past {
			life, _ := b.r.Lifecycle("w1", "c1", uint64(at))
			view, _ := b.r.Settlement("w1", "c1", uint64(at))
			position, _ := b.r.StateOn("w1", "c1", uint64(at), probe)
			if !reflect.DeepEqual(kept, snapshot{life, view, position}) {
				t.Fatalf("round %d: what was known at %d changed", round, at)
			}
		}
	}
	for _, name := range []string{"batch written", "batch refused", "missing written", "secured written", "secured refused", "settle written", "settle refused", "reopen written", "reopen refused", "settled", "settled after an end"} {
		if reached[name] < 20 {
			t.Errorf("%s was reached %d times", name, reached[name])
		}
	}
	t.Logf("%v", reached)
}
