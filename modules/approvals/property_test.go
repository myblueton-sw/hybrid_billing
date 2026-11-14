package approvals

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type modelRequest struct {
	target, amount, requester, state string
	approvers                        map[string]bool
	decided                          map[string]string
	approvedAt                       time.Time
	executed                         bool
}

func TestApprovalsAgainstAModel(t *testing.T) {
	limits := map[string]int{"ann": 1000, "bob": 500}
	for seed := int64(1); seed <= 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		need := 1 + rng.Intn(2)
		r, clock := setup(t, need)
		revoked := map[string]bool{}
		reqs := map[string]*modelRequest{}
		order := []string{}
		executedKeys := map[string]bool{}
		n := 0
		for step := 0; step < 80; step++ {
			trace := fmt.Sprintf("seed %d step %d", seed, step)
			now := clock.read()
			switch op := rng.Intn(7); op {
			case 0:
				if len(order) >= 10 {
					continue
				}
				n++
				id := fmt.Sprintf("q%d", n)
				target := []string{"t1", "t2"}[rng.Intn(2)]
				amount := []string{"10", "-40", "60", "2000"}[rng.Intn(4)]
				requester := []string{"req", "ann", "bot"}[rng.Intn(3)]
				kind := KindHuman
				if requester == "bot" {
					kind = KindAssistant
				}
				must(r.Request("ws", subject(id, target, amount), Actor{ID: requester, Kind: kind}))
				for _, other := range reqs {
					if other.target == target && (other.state == StatePending || other.state == StateApproved) {
						other.state = StateSuperseded
					}
				}
				reqs[id] = &modelRequest{target: target, amount: amount, requester: requester, state: StatePending, approvers: map[string]bool{}, decided: map[string]string{}}
				order = append(order, id)
			case 1, 2:
				if len(order) == 0 {
					continue
				}
				id := order[rng.Intn(len(order))]
				m := reqs[id]
				who := []string{"ann", "bob", "writer", "req", "ann"}[rng.Intn(5)]
				verdict := VerdictApprove
				d := approve
				if rng.Intn(5) == 0 {
					verdict, d = VerdictReject, Decision{Verdict: VerdictReject, Reason: "other"}
				}
				_, err := r.Decide("ws", id, d, human(who))
				want := "ok"
				abs := map[string]int{"10": 10, "-40": 40, "60": 60, "2000": 2000}[m.amount]
				switch {
				case m.decided[who] != "":
					if m.decided[who] != verdict {
						want = "conflict"
					}
				case m.state != StatePending:
					want = ReasonNotPending
				case who == "writer" || who == m.requester:
					want = ReasonSelfApproval
				case verdict == VerdictApprove && (revoked[who] || limits[who] < abs):
					want = ReasonNoAuthority
				}
				if kind(err) != want {
					t.Fatalf("%s decide %s by %s: want %s got %s", trace, id, who, want, kind(err))
				}
				if err == nil && m.decided[who] == "" {
					m.decided[who] = verdict
					if verdict == VerdictReject {
						m.state = StateRejected
					} else {
						m.approvers[who] = true
						if len(m.approvers) >= need {
							m.state, m.approvedAt = StateApproved, now
						}
					}
				}
			case 3:
				if len(order) == 0 {
					continue
				}
				id := order[rng.Intn(len(order))]
				m := reqs[id]
				e := runOf(fmt.Sprintf("e%d", step), id, m.target, m.amount)
				changed := rng.Intn(4) == 0
				if changed {
					e.Digests["erp_draft"] = digest('f')
				}
				_, err := r.Execute("ws", e, human("ops"))
				want := "ok"
				abs := map[string]int{"10": 10, "-40": 40, "60": 60, "2000": 2000}[m.amount]
				lost := false
				for a := range m.approvers {
					if revoked[a] || limits[a] < abs {
						lost = true
					}
				}
				switch {
				case m.state == StatePending || m.state == StateRejected:
					want = ReasonNotApproved
				case m.state == StateSuperseded:
					want = ReasonSuperseded
				case m.state == StateWithdrawn:
					want = ReasonWithdrawn
				case now.Sub(m.approvedAt) > 24*time.Hour:
					want = ReasonExpired
				case changed:
					want = ReasonSubjectChanged
				case lost:
					want = ReasonAuthorityLost
				case executedKeys[m.target]:
					want = ReasonAlreadyExecuted
				}
				if kind(err) != want {
					t.Fatalf("%s execute %s: want %s got %s", trace, id, want, kind(err))
				}
				if err == nil {
					executedKeys[m.target], m.executed = true, true
				}
			case 4:
				if len(order) == 0 {
					continue
				}
				id := order[rng.Intn(len(order))]
				m := reqs[id]
				who := human(m.requester)
				if m.requester == "bot" {
					who = Actor{ID: "bot", Kind: KindAssistant}
				}
				_, err := r.Withdraw("ws", id, who)
				want := "ok"
				switch {
				case m.state == StateWithdrawn:
				case m.state != StatePending && m.state != StateApproved || m.executed:
					want = "rejected"
				}
				if kind(err) != want {
					t.Fatalf("%s withdraw %s: want %s got %s", trace, id, want, kind(err))
				}
				if err == nil {
					m.state = StateWithdrawn
				}
			case 5:
				if !revoked["ann"] && rng.Intn(4) == 0 {
					must(r.Revoke("ws", "g-ann", human("adm1")))
					revoked["ann"] = true
				}
			case 6:
				clock.advance(time.Duration(rng.Intn(10)) * time.Hour)
			}
			for _, id := range order {
				v := must(r.RequestOf("ws", id))
				if v.State != reqs[id].state {
					t.Fatalf("%s: %s is %s, model %s", trace, id, v.State, reqs[id].state)
				}
			}
		}
	}
}
