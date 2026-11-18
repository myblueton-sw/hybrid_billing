package bulkrevoke

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"
)

func TestPropertyCountsAndStates(t *testing.T) {
	r := rand.New(rand.NewPCG(299, 7))
	for round := 0; round < 200; round++ {
		f := setup(t, Limits{MaxTargets: 40, MaxMatching: 40, Pending: 5, Kept: 50, Lifetime: time.Hour})
		n := 1 + r.IntN(30)
		users := []string{}
		for i := 0; i < n; i++ {
			users = append(users, "u"+strconv.Itoa(i))
		}
		d := f.add("ann", "", users...)
		targets, exclusions := []Target{}, []Exclusion{}
		for _, u := range users {
			if r.IntN(5) == 0 {
				exclusions = append(exclusions, Exclusion{User: u, Reason: ExcludedNotFound})
			} else {
				targets = append(targets, target(u))
			}
		}
		if len(targets) == 0 {
			continue
		}
		v, _, err := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", targets, exclusions, Impact{})
		if err != nil {
			t.Fatal(err)
		}
		held := map[string]string{}
		for _, x := range targets {
			if r.IntN(6) == 0 {
				held[x.User] = HeldChanged
			}
		}
		v, err = f.r.Execute("w1", v.ID, v.Digest, "bob", 3, held)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.r.Execute("w1", v.ID, v.Digest, "cat", 3, nil); err == nil {
			t.Fatal("executed twice")
		}
		check := func(stage string) View {
			got := mustGet(f, v.ID)
			c := got.Counts
			if c != count(got) {
				t.Fatalf("round %d %s: kept counts %+v, recounted %+v", round, stage, c, count(got))
			}
			if c.Targets != c.Succeeded+c.Held+c.Failed+c.Unprocessed || c.Targets != len(targets) || c.Excluded != len(exclusions) {
				t.Fatalf("round %d %s: %+v", round, stage, c)
			}
			for _, item := range got.Items {
				if item.State == DenyConfirmed && (item.Receipt.Reason != "" || item.Receipt.Sequence <= got.Header.Sequence) {
					t.Fatalf("round %d: confirmed without an advancing receipt %+v", round, item)
				}
				if (item.State == CleanupRunning || item.State == Complete) && item.Receipt.Sequence == 0 {
					t.Fatalf("round %d: cleanup of an unconfirmed item %+v", round, item)
				}
				if _, in := held[item.Target.User]; in != (item.Outcome == OutcomeHeld) && item.Outcome != OutcomeExcluded {
					t.Fatalf("round %d: hold %+v", round, item)
				}
			}
			return got
		}
		check("executed")
		header := Receipt{Sequence: 10}
		if r.IntN(8) == 0 {
			header = Receipt{Reason: "ledger_stopped"}
		}
		f.r.Header("w1", v.ID, header)
		got := check("header")
		seq := uint64(10)
		failed := false
		for _, item := range got.Items {
			if item.Outcome != OutcomeUnprocessed || item.State != Received {
				continue
			}
			receipt := Receipt{Sequence: seq + 1}
			switch {
			case failed || r.IntN(15) == 0:
				failed, receipt = true, Receipt{Reason: "commit_unknown"}
			case r.IntN(20) == 0:
				receipt = Receipt{Sequence: 1}
			}
			seq++
			f.r.Record("w1", v.ID, []int{item.Index}, []Receipt{receipt})
		}
		check("records")
		for _, item := range mustGet(f, v.ID).Items {
			f.r.Cleaning("w1", v.ID, item.Index)
			switch r.IntN(4) {
			case 0:
				f.r.Failed("w1", v.ID, item.Index, "attempts_spent")
			case 1:
			default:
				f.r.Cleaned("w1", v.ID, item.Index, nil)
			}
			check("cleanup")
		}
		final := check("final")
		state, reason := JobState(final, Cleaning{State: CleaningSucceeded})
		unconfirmed := false
		for _, item := range final.Items {
			unconfirmed = unconfirmed || item.State == Unconfirmed
		}
		if unconfirmed != (state == JobWaiting && reason == ReasonUnconfirmed) {
			t.Fatalf("round %d: job %s %s", round, state, reason)
		}
	}
}
