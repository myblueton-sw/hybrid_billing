package decisionledger

import (
	"math/rand/v2"
	"os"
	"strconv"
	"testing"
)

func TestPropertyReopenKeepsTheWatermark(t *testing.T) {
	r := rand.New(rand.NewPCG(298, 1))
	for round := 0; round < 20; round++ {
		o := options(t)
		l := must(t, o)
		n := r.IntN(12)
		for i := 0; i < n; i++ {
			d := sample(strconv.Itoa(i))
			if r.IntN(2) == 0 {
				d.Decision, d.Request, d.RequestDigest, d.ApprovedBy = RevokeDelegation, "", "", ""
			}
			for j := r.IntN(4); j > 0; j-- {
				d.EndedDelegations = append(d.EndedDelegations, "delegation-"+strconv.Itoa(j))
			}
			if _, err := l.Commit(d); err != nil {
				t.Fatal(err)
			}
		}
		before := l.Status()
		l.Close()
		again := must(t, o)
		after := again.Status()
		again.Close()
		if after.Watermark != before.Watermark || after.Records != uint64(n)+1 || after.State != StateOK {
			t.Fatalf("round %d: %+v vs %+v", round, after, before)
		}
	}
}

func TestPropertyAnySingleByteMutationIsDetected(t *testing.T) {
	o := options(t)
	l := must(t, o)
	for i := 0; i < 3; i++ {
		commit(t, l, strconv.Itoa(i))
	}
	l.Close()
	original, _ := os.ReadFile(ledgerPath(o))
	r := rand.New(rand.NewPCG(298, 2))
	positions := []int{0, len(original) - 1, len(original) - 2}
	for i := 0; i < 150; i++ {
		positions = append(positions, r.IntN(len(original)))
	}
	for _, at := range positions {
		mutated := append([]byte{}, original...)
		mutated[at] ^= byte(1 + r.IntN(255))
		os.WriteFile(ledgerPath(o), mutated, 0o600)
		got := must(t, o)
		st := got.Status()
		got.Close()
		if st.State == StateOK {
			t.Fatalf("mutation at %d read as ok", at)
		}
	}
}
