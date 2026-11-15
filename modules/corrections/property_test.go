package corrections

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

func cents(value int64) string { return format(big.NewInt(value), 2) }

func plain(value int64) string {
	whole, fraction := value/100, value%100
	switch {
	case fraction == 0:
		return strconv.FormatInt(whole, 10)
	case fraction%10 == 0:
		return strconv.FormatInt(whole, 10) + "." + strconv.FormatInt(fraction/10, 10)
	case fraction < 10:
		return strconv.FormatInt(whole, 10) + ".0" + strconv.FormatInt(fraction, 10)
	}
	return strconv.FormatInt(whole, 10) + "." + strconv.FormatInt(fraction, 10)
}

func TestLedgerMatchesAModelAndNeverSticks(t *testing.T) {
	applied, stale, explained := 0, 0, 0
	for seed := int64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := open(t, Limits{Workspaces: 1, Corrections: 200, Records: 200})
		total := int64(1000 + rng.Intn(100000))
		f.issue(t, "ws", draft("o", "inv", cents(total), ""))
		effective := total
		unlinked := map[string]bool{}
		open := ""
		state := ""
		var target, base int64
		next := 0
		name := func(prefix string) string { next++; return prefix + strconv.Itoa(next) }
		for step := 0; step < 150; step++ {
			switch rng.Intn(6) {
			case 0:
				id := name("c")
				want := int64(rng.Intn(int(effective + 2)))
				_, err := f.r.Request("ws", request(id, "o", name("k"), cents(want)), human("requester"))
				switch {
				case len(unlinked) > 0:
					expect(t, "unlinked", err, RefusalUnlinked)
				case open != "":
					expect(t, "pending", err, RefusalPending)
				case want == effective:
					expect(t, "no change", err, RefusalNoChange)
				case want > effective:
					expect(t, "debit", err, RefusalDebit)
				default:
					if err != nil {
						t.Fatalf("seed %d: request %v", seed, err)
					}
					v := must(f.r.Of("ws", id))
					if v.Base.Effective != plain(effective) || v.Difference != "-"+plain(effective-want) {
						t.Fatalf("seed %d: base %+v model %d %d", seed, v, effective, want)
					}
					open, state, target, base = id, StateRequested, want, effective
				}
			case 1:
				if open == "" || state != StateRequested {
					continue
				}
				v := must(f.r.Of("ws", open))
				_, err := f.r.Approve("ws", open, v.Digest, human("checker"))
				if base != effective || len(unlinked) > 0 {
					expect(t, "stale", err, RefusalBaseChanged)
					continue
				}
				if err != nil {
					t.Fatalf("seed %d: approve %v", seed, err)
				}
				state = StateApproved
			case 2:
				if open == "" {
					continue
				}
				_, err := f.r.Withdraw("ws", open, human("requester"))
				if state == StateApproved && len(unlinked) > 0 {
					expect(t, "withdraw", err, RefusalCreditIssued)
					continue
				}
				if err != nil {
					t.Fatalf("seed %d: withdraw %v", seed, err)
				}
				open = ""
			case 3:
				if open == "" || state != StateApproved {
					continue
				}
				amount := base - target
				if amount > effective {
					continue
				}
				id := name("y")
				f.credit(t, id, cents(amount), "o")
				effective -= amount
				_, err := f.r.Apply("ws", open, id, system("linker"))
				if base != effective+amount || len(unlinked) > 0 {
					expect(t, "apply stale", err, RefusalBaseChanged)
					unlinked[id] = true
					stale++
					continue
				}
				if err != nil {
					t.Fatalf("seed %d: apply %v", seed, err)
				}
				applied++
				open = ""
			case 4:
				if effective == 0 {
					continue
				}
				amount := int64(1 + rng.Intn(int(effective)))
				id := name("x")
				f.credit(t, id, cents(amount), "o")
				effective -= amount
				unlinked[id] = true
			case 5:
				for id := range unlinked {
					must(f.r.Explain("ws", "o", id, Explanation{Reason: ReasonOther, Evidence: "memo"}, human("auditor")))
					delete(unlinked, id)
					explained++
					break
				}
			}
			e := must(f.r.EffectiveOf("ws", "o"))
			if e.Effective != plain(effective) || effective < 0 || len(e.Unlinked) != len(unlinked) || (e.Open == nil) != (open == "") {
				t.Fatalf("seed %d step %d: %+v model %d %v %q", seed, step, e, effective, unlinked, open)
			}
		}
		for id := range unlinked {
			must(f.r.Explain("ws", "o", id, Explanation{Reason: ReasonOther, Evidence: "memo"}, human("auditor")))
		}
		if open != "" {
			must(f.r.Withdraw("ws", open, human("requester")))
		}
		if effective > 0 {
			must(f.r.Request("ws", request("final", "o", "final", "0"), human("requester")))
		}
		credits := must(f.is.CreditsOf("ws", "o"))
		sum := int64(0)
		for _, id := range credits {
			d := must(f.is.DocumentOf("ws", id))
			value, _ := scaled(d.Snapshot.Draft.Total, 2)
			sum += value.Int64()
		}
		if total-sum != effective {
			t.Fatalf("seed %d: issuance %d model %d", seed, total-sum, effective)
		}
	}
	if applied < 10 || stale < 1 || explained < 10 {
		t.Fatalf("weak exploration: applied %d stale %d explained %d", applied, stale, explained)
	}
}
