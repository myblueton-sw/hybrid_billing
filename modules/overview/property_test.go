package overview

import (
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"
)

type modelFact struct {
	fact     Fact
	received time.Time
}

type snapshot struct {
	token string
	facts int
	at    time.Time
}

func modelRows(facts []modelFact, n int, evaluated time.Time, stale time.Duration) map[Subject]Row {
	current := map[string]modelFact{}
	for _, mf := range facts[:n] {
		if c, ok := current[mf.fact.Source.ID]; !ok || mf.fact.Source.Revision > c.fact.Source.Revision {
			current[mf.fact.Source.ID] = mf
		}
	}
	bySubject := map[Subject][]modelFact{}
	for _, mf := range current {
		bySubject[mf.fact.Subject] = append(bySubject[mf.fact.Subject], mf)
	}
	order := map[string]int{}
	for index, mf := range facts {
		order[mf.fact.ID] = index
	}
	out := map[Subject]Row{}
	for s, list := range bySubject {
		sort.Slice(list, func(i, j int) bool {
			a, b := list[i], list[j]
			if !a.fact.Occurred.Equal(b.fact.Occurred) {
				return a.fact.Occurred.Before(b.fact.Occurred)
			}
			return order[a.fact.ID] < order[b.fact.ID]
		})
		lead := list[len(list)-1]
		row := Row{Subject: s, Stage: lead.fact.Stage, Due: lead.fact.Due, Owner: lead.fact.Owner, Next: lead.fact.Next, Amounts: map[string]string{}, Status: StatusCurrent}
		reported := map[string]string{}
		for _, mf := range list {
			for k := range mf.fact.Amounts {
				if other, ok := reported[k]; ok && other != mf.fact.Source.ID {
					panic("two current sources report " + k)
				}
				reported[k] = mf.fact.Source.ID
			}
		}
		for _, mf := range list {
			if mf.fact.Stage == StageBlocked {
				row.Stage = StageBlocked
			}
			for k, v := range mf.fact.Amounts {
				row.Amounts[k] = v
			}
			if mf.received.After(row.Received) {
				row.Received = mf.received
			}
		}
		if row.Received.Before(evaluated.Add(-stale)) {
			row.Status = StatusStale
		}
		out[s] = row
	}
	return out
}

func TestProjectionMatchesAModel(t *testing.T) {
	stats := map[string]int{}
	for seed := int64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := open(t, testLimits())
		var facts []modelFact
		var snaps []snapshot
		revisions := map[string]int{}
		subjects := []Subject{subjectOf("acme", "k1"), subjectOf("acme", "k2"), subjectOf("beta", "k1")}
		next := 0
		for step := 0; step < 60; step++ {
			f.clock.advance(time.Duration(rng.Intn(40)) * time.Minute)
			source := "s" + strconv.Itoa(rng.Intn(4))
			revision := revisions[source] + 1
			if rng.Intn(4) == 0 && revisions[source] > 2 {
				revision = revisions[source] - 1 - rng.Intn(2)
			}
			next++
			x := Fact{ID: "f" + strconv.Itoa(next), Subject: subjects[rng.Intn(len(subjects))], Source: Source{ID: source, Revision: revision},
				Stage: []string{StageRated, StageIssued, StageBlocked, StageSettled}[rng.Intn(4)], Occurred: f.clock.read().Add(-time.Duration(rng.Intn(120)) * time.Minute),
				Amounts: map[string]string{}}
			if rng.Intn(2) == 0 {
				x.Due = "2026-12-0" + strconv.Itoa(1+rng.Intn(9))
				x.Owner, x.Next = "o"+strconv.Itoa(rng.Intn(3)), "n"+strconv.Itoa(rng.Intn(3))
			}
			meaning := []string{MeaningSales, MeaningOutstanding, MeaningAllocation, MeaningCost}[rng.Intn(4)]
			if rng.Intn(3) > 0 {
				x.Amounts[meaning] = strconv.Itoa(rng.Intn(1000) - 300)
			}
			want := "ok"
			used := false
			for _, mf := range facts {
				if mf.fact.Source.ID == source && mf.fact.Source.Revision == revision {
					used = true
				}
			}
			if used {
				want = ReasonRevisionConflict
			} else if revision > revisions[source] {
				current := map[string]modelFact{}
				for _, mf := range facts {
					if c, ok := current[mf.fact.Source.ID]; !ok || mf.fact.Source.Revision > c.fact.Source.Revision {
						current[mf.fact.Source.ID] = mf
					}
				}
				for id, mf := range current {
					if id == source || mf.fact.Subject != x.Subject {
						continue
					}
					for k := range x.Amounts {
						if _, ok := mf.fact.Amounts[k]; ok {
							want = ReasonMeaningOwned
						}
					}
				}
			}
			_, err := f.r.Ingest("ws", x, system("feed"))
			if reason(err) != want {
				t.Fatalf("seed %d: ingest %v model %s", seed, err, want)
			}
			if want == "ok" {
				facts = append(facts, modelFact{fact: Fact{ID: x.ID, Subject: x.Subject, Source: x.Source, Stage: x.Stage, Due: x.Due, Owner: x.Owner, Next: x.Next, Amounts: x.Amounts, Occurred: x.Occurred}, received: f.clock.read()})
				if revision > revisions[source] {
					revisions[source] = revision
				}
				stats[want]++
			} else {
				stats[want]++
			}
			if rng.Intn(5) == 0 {
				p := must(f.r.Query("ws", Query{Limit: 1}, human("admin")))
				snaps = append(snaps, snapshot{token: p.AsOf, facts: len(facts), at: p.Evaluated})
			}
		}
		snaps = append(snaps, snapshot{facts: len(facts)})
		for _, snap := range snaps {
			p := must(f.r.Query("ws", Query{AsOf: snap.token, Limit: 100}, human("admin")))
			at := snap.at
			if snap.token == "" {
				at = p.Evaluated
			}
			want := modelRows(facts, snap.facts, at, testLimits().Stale)
			if p.Count != len(want) {
				t.Fatalf("seed %d: count %d model %d", seed, p.Count, len(want))
			}
			for _, row := range p.Rows {
				w := want[row.Subject]
				if row.Stage != w.Stage || row.Due != w.Due || row.Owner != w.Owner || row.Next != w.Next || !row.Received.Equal(w.Received) || row.Status != w.Status || len(row.Amounts) != len(w.Amounts) {
					t.Fatalf("seed %d: row %+v model %+v", seed, row, w)
				}
				for k, v := range w.Amounts {
					if row.Amounts[k] != v {
						t.Fatalf("seed %d: amount %s %s model %s", seed, k, row.Amounts[k], v)
					}
				}
			}
		}
	}
	if stats["ok"] < 100 || stats[ReasonMeaningOwned] < 5 || stats[ReasonRevisionConflict] < 5 {
		t.Fatalf("weak exploration: %v", stats)
	}
}
