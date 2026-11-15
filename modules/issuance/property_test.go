package issuance

import (
	"math/big"
	"math/rand"
	"strconv"
	"sync"
	"testing"
)

type modelNumber struct {
	draft  string
	state  string
	reason string
}

func TestNumberingMatchesAModel(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewSource(seed))
		l := testLimits()
		l.Drafts, l.Documents, l.Numbers = 40, 40, 200
		f := open(t, l)
		f.setup(t, "ws")
		var model []modelNumber
		live := map[string]int{}
		issued := map[string]bool{}
		submitted := map[string]bool{}
		for step := 0; step < 120; step++ {
			id := "d" + strconv.Itoa(rng.Intn(12))
			switch rng.Intn(4) {
			case 0:
				if submitted[id] {
					continue
				}
				f.ready(t, "ws", draftOf(id, "inv"))
				submitted[id] = true
			case 1:
				n, err := f.r.Reserve("ws", id, human("clerk"))
				switch {
				case !submitted[id]:
					expect(t, "reserve unknown", err, "not_found")
				case issued[id]:
					expect(t, "reserve issued", err, ReasonAlreadyIssued)
				case live[id] > 0:
					if err != nil || n.Number != format(Series{Prefix: "INV-", Width: 6}, int64(live[id])) {
						t.Fatalf("seed %d: repeat reserve %v %+v", seed, err, n)
					}
				default:
					model = append(model, modelNumber{draft: id, state: NumberReserved})
					live[id] = len(model)
					if err != nil || n.Value != int64(len(model)) {
						t.Fatalf("seed %d: reserve %v %+v", seed, err, n)
					}
				}
			case 2:
				reasonText := []string{CancelAbandoned, CancelOperator, CancelOther}[rng.Intn(3)]
				_, err := f.r.Cancel("ws", id, reasonText, human("clerk"))
				switch {
				case !submitted[id]:
					expect(t, "cancel unknown", err, "not_found")
				case issued[id]:
					expect(t, "cancel issued", err, ReasonAlreadyIssued)
				case live[id] > 0:
					if err != nil {
						t.Fatalf("seed %d: cancel %v", seed, err)
					}
					model[live[id]-1].state, model[live[id]-1].reason = NumberCancelled, reasonText
					delete(live, id)
				default:
					if err == nil {
						continue
					}
				}
			case 3:
				_, err := f.r.Issue("ws", id, human("issuer"))
				switch {
				case !submitted[id]:
					expect(t, "issue unknown", err, "not_found")
				case issued[id]:
					if err != nil {
						t.Fatalf("seed %d: issue repeat %v", seed, err)
					}
				default:
					if err != nil {
						t.Fatalf("seed %d: issue %v", seed, err)
					}
					if live[id] == 0 {
						model = append(model, modelNumber{draft: id})
						live[id] = len(model)
					}
					model[live[id]-1].state = NumberIssued
					issued[id] = true
				}
			}
		}
		numbers := must(f.r.Numbers("ws", "s-inv"))
		if len(numbers) != len(model) {
			t.Fatalf("seed %d: %d numbers, model %d", seed, len(numbers), len(model))
		}
		seen := map[string]bool{}
		for index, n := range numbers {
			m := model[index]
			if n.Value != int64(index+1) || n.Draft != m.draft || n.State != m.state || n.Reason != m.reason || seen[n.Number] {
				t.Fatalf("seed %d: number %d %+v model %+v", seed, index, n, m)
			}
			seen[n.Number] = true
		}
		for id := range issued {
			doc := must(f.r.DocumentOf("ws", id))
			if model[live[id]-1].draft != id || doc.Snapshot.Number != numbers[live[id]-1].Number {
				t.Fatalf("seed %d: document %s", seed, id)
			}
		}
	}
}

func TestConcurrentIssuesGetDistinctNumbers(t *testing.T) {
	l := testLimits()
	l.Drafts, l.Documents, l.Numbers = 200, 200, 200
	f := open(t, l)
	f.setup(t, "ws")
	const count = 64
	for index := 0; index < count; index++ {
		f.ready(t, "ws", draftOf("d"+strconv.Itoa(index), "inv"))
	}
	var wg sync.WaitGroup
	results := make([]string, count)
	for index := 0; index < count; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			id := "d" + strconv.Itoa(index)
			if index%2 == 0 {
				must(f.r.Reserve("ws", id, human("clerk")))
			}
			results[index] = must(f.r.Issue("ws", id, system("job"))).Snapshot.Number
		}(index)
	}
	wg.Wait()
	seen := map[string]bool{}
	for _, n := range results {
		if seen[n] {
			t.Fatalf("duplicate %s", n)
		}
		seen[n] = true
	}
	numbers := must(f.r.Numbers("ws", "s-inv"))
	if len(numbers) != count {
		t.Fatal(len(numbers))
	}
	for index, n := range numbers {
		if n.Value != int64(index+1) || n.State != NumberIssued || !seen[n.Number] {
			t.Fatalf("number %+v", n)
		}
	}
}

func TestCreditRoomMatchesAModel(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		rng := rand.New(rand.NewSource(seed))
		l := testLimits()
		l.Drafts, l.Documents, l.Numbers = 40, 40, 80
		f := open(t, l)
		f.setup(t, "ws")
		originalCents := int64(1 + rng.Intn(50000))
		original := draftOf("o", "inv", Line{Text: "Usage", Amount: render(strconv.FormatInt(originalCents, 10), 2)})
		f.issued(t, original)
		used := int64(0)
		for index := 0; index < 12; index++ {
			cents := int64(1 + rng.Intn(int(originalCents/3+2)))
			c := draftOf("c"+strconv.Itoa(index), "crn", Line{Text: "Credit", Amount: render(strconv.FormatInt(cents, 10), 2)})
			c.Original = "o"
			_, err := f.r.Submit("ws", c, human("clerk"))
			fits := used+cents <= originalCents
			if fits != (err == nil) {
				t.Fatalf("seed %d: submit %d used %d of %d: %v", seed, cents, used, originalCents, err)
			}
			if !fits {
				expect(t, "room", err, ReasonExceedsOriginal)
				continue
			}
			f.ready(t, "ws", c)
			must(f.r.Issue("ws", c.ID, human("issuer")))
			used += cents
		}
		total := new(big.Int)
		for _, id := range s(f).documents["o"].credits {
			value, _ := amount(s(f).documents[id].view.Snapshot.Draft.Total, 2)
			total.Add(total, value)
		}
		if total.Int64() != used || used > originalCents {
			t.Fatalf("seed %d: credited %v model %d", seed, total, used)
		}
	}
}
