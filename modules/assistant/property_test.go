package assistant

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type modelCall struct {
	tool, target, outcome string
	at                    time.Time
}

func TestBudgetProperty(t *testing.T) {
	answer := setup(t).answer("ann")
	for seed := int64(1); seed <= 200; seed++ {
		random := rand.New(rand.NewSource(seed))
		c := &fake{entries: []commands.Entry{
			{ID: "x.read", Permission: commands.Permission{Action: "view", Kind: "statement"}, Supported: true},
			{ID: "x.draft", Permission: commands.Permission{Action: "request_approval", Kind: "statement"}, Mutating: true, Supported: true,
				Fields: []commands.Field{{Name: "amount", Type: commands.Decimal}, {Name: "count", Type: commands.Integer}}},
			{ID: "x.issue", Permission: commands.Permission{Action: "issue", Kind: "statement"}, Mutating: true, Supported: true},
		}}
		c.result = func(call commands.Call) commands.Result {
			if len(call.Target) == 2 && call.Target[1] == '1' {
				return commands.Result{Err: &commands.Error{Code: commands.PermissionDenied}}
			}
			return commands.Result{Status: commands.Completed}
		}
		l := limits
		l.Calls, l.Targets, l.Duration = 1+random.Intn(8), 1+random.Intn(4), time.Duration(1+random.Intn(10))*time.Minute
		now := start
		r, err := NewRegistry(Config{Commands: c, Client: "asst", Protected: protected}, l, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		for _, tool := range []Tool{{Command: "x.read", Stage: StageRead}, {Command: "x.draft", Stage: StageDraft}, {Command: "x.issue", Stage: StageRequest}} {
			if second(r.ProposeTool("w1", tool, ann)) != nil || second(r.ApproveTool("w1", tool.Command, bob)) != nil {
				t.Fatal("exposure")
			}
		}
		if second(r.CreateJob("w1", Job{ID: "j1"}, ann)) != nil || second(r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: now.Add(time.Hour)}, ann)) != nil ||
			second(r.Attach("w1", "j1", "c1", ann)) != nil {
			t.Fatal("job")
		}
		calls, targets, closed := 0, map[string]bool{}, false
		var ledger []modelCall
		for step := 0; step < 30; step++ {
			now = now.Add(time.Duration(random.Intn(40)) * time.Second)
			if random.Intn(40) == 0 {
				if second(r.CloseJob("w1", "j1", ann)) != nil {
					t.Fatal("close")
				}
				closed = true
			}
			tool := []string{"x.read", "x.draft", "x.issue", "x.none"}[random.Intn(4)]
			target := []string{"t1", "t2", "t3", "t4", "t5", "t6"}[random.Intn(6)]
			var input []commands.Value
			if random.Intn(5) == 0 {
				input = append(input, commands.Value{Name: "amount", Value: "1"})
			}
			if random.Intn(2) == 0 {
				input = append(input, commands.Value{Name: "count", Value: "2"})
			}
			reply, err := r.Invoke("w1", "j1", Invocation{Conversation: "c1", Tool: tool, Target: target, Expected: "r1", Input: input, Answer: answer}, bot)
			decimal := false
			for _, value := range input {
				decimal = decimal || value.Name == "amount" && tool == "x.draft"
			}
			want := ""
			switch {
			case closed:
				want = ReasonJobClosed
			case !now.Before(start.Add(l.Duration)):
				want = ReasonBudget
			case tool == "x.none":
				want = ReasonNotExposed
			case tool == "x.issue":
				want = ReasonRequestOnly
			case decimal:
				want = ReasonModelAmount
			case calls >= l.Calls || !targets[target] && len(targets) >= l.Targets:
				want = ReasonBudget
			}
			if want != "" {
				var refusal Refusal
				if !errors.As(err, &refusal) || refusal.Reason != want {
					t.Fatalf("seed %d step %d: %v; want %s", seed, step, err, want)
				}
				continue
			}
			calls++
			targets[target] = true
			outcome := "completed"
			if target == "t1" {
				outcome = "permission_denied"
			}
			ledger = append(ledger, modelCall{tool, target, outcome, now})
			if err != nil || reply.Calls != calls || reply.Targets != len(targets) || reply.Status != outcome {
				t.Fatalf("seed %d step %d: %+v %v", seed, step, reply, err)
			}
		}
		job, _ := r.JobOf("w1", "j1")
		if job.Calls != calls || job.Targets != len(targets) || len(job.Ledger) != len(ledger) || len(c.calls) != calls {
			t.Fatalf("seed %d: job %+v; model %d %d %d", seed, job, calls, len(targets), len(ledger))
		}
		for index, entry := range job.Ledger {
			m := ledger[index]
			if entry.Sequence != index+1 || entry.Kind != EntryCall || entry.Tool != m.tool || entry.Target != m.target || entry.Outcome != m.outcome || !entry.At.Equal(m.at) {
				t.Fatalf("seed %d entry %d: %+v; want %+v", seed, index, entry, m)
			}
		}
	}
}

type modelCard struct {
	state, confirmer, decliner string
	attempts                   int
}

func TestCardProperty(t *testing.T) {
	w := setup(t)
	people := map[string]Actor{"ann": ann, "bob": bob}
	held := map[string]Confirmation{"ann": {Answer: w.answer("ann")}, "bob": {Answer: w.answer("bob")}}
	for seed := int64(1); seed <= 200; seed++ {
		random := rand.New(rand.NewSource(seed))
		outcome := commands.Result{}
		c := &fake{entries: []commands.Entry{{ID: "x.issue", Permission: commands.Permission{Action: "issue", Kind: "statement"}, Mutating: true, Supported: true}}}
		c.result = func(commands.Call) commands.Result { return outcome }
		now := start
		r, err := NewRegistry(Config{Commands: c, Client: "asst", Protected: protected}, limits, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		if second(r.ProposeTool("w1", Tool{Command: "x.issue", Stage: StageRequest}, ann)) != nil || second(r.ApproveTool("w1", "x.issue", bob)) != nil ||
			second(r.CreateJob("w1", Job{ID: "j1"}, ann)) != nil || second(r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: now.Add(time.Hour)}, ann)) != nil ||
			second(r.Attach("w1", "j1", "c1", ann)) != nil {
			t.Fatal("setup")
		}
		model, digests := map[string]*modelCard{}, map[string]string{}
		for _, id := range []string{"k1", "k2", "k3"} {
			view, err := r.Prepare("w1", "j1", "c1", Card{ID: id, Tool: "x.issue", Target: "t1", Expected: "r1"}, bot)
			if err != nil {
				t.Fatal(err)
			}
			model[id], digests[id] = &modelCard{state: CardPending}, view.Digest
		}
		executed := 0
		for step := 0; step < 40; step++ {
			now = now.Add(time.Duration(random.Intn(400)) * time.Second)
			expired := !now.Before(start.Add(limits.Duration))
			id := []string{"k1", "k2", "k3"}[random.Intn(3)]
			person := []string{"ann", "bob"}[random.Intn(2)]
			m := model[id]
			if m.state == CardPending && expired {
				m.state = CardExpired
			}
			want := ""
			var view CardView
			if random.Intn(4) == 0 {
				view, err = r.Decline("w1", id, "no", people[person])
				switch {
				case m.state == CardExpired:
					want = ReasonCardExpired
				case m.state == CardConfirmed || m.state == CardDeclined && m.decliner != person:
					want = ReasonAlreadyDecided
				case m.state != CardDeclined:
					m.state, m.decliner = CardDeclined, person
				}
			} else {
				outcome = []commands.Result{{Status: commands.Completed, Operation: "op"}, {Err: &commands.Error{Code: commands.InternalError, Operation: "op"}},
					{Err: &commands.Error{Code: commands.PermissionDenied}}}[random.Intn(3)]
				confirmation := held[person]
				confirmation.Digest = digests[id]
				view, err = r.Confirm("w1", id, confirmation, people[person])
				switch {
				case m.state == CardConfirmed && m.confirmer != person, m.state == CardDeclined:
					want = ReasonAlreadyDecided
				case m.state == CardConfirmed:
				case m.state == CardExpired:
					want = ReasonCardExpired
				case m.state == CardUncertain && m.confirmer != person:
					want = ReasonOutcomeUncertain
				case m.attempts >= maxAttempts:
					want = ReasonBudget
				default:
					executed++
					m.attempts++
					switch {
					case outcome.Err == nil:
						m.state, m.confirmer = CardConfirmed, person
					case outcome.Err.Operation != "":
						m.state, m.confirmer = CardUncertain, person
					}
				}
			}
			if want != "" {
				var refusal Refusal
				if !errors.As(err, &refusal) || refusal.Reason != want {
					t.Fatalf("seed %d step %d %s: %v; want %s (model %+v)", seed, step, id, err, want, *m)
				}
				continue
			}
			if err != nil || view.State != m.state || view.Attempts != m.attempts || view.Confirmer != m.confirmer || view.Decliner != m.decliner {
				t.Fatalf("seed %d step %d %s: %+v %v; model %+v", seed, step, id, view, err, *m)
			}
		}
		if len(c.calls) != executed {
			t.Fatalf("seed %d: executed %d; model %d", seed, len(c.calls), executed)
		}
		for id, m := range model {
			if view, _ := r.CardOf("w1", id); view.State != m.state && !(m.state == CardPending && !now.Before(start.Add(limits.Duration))) {
				t.Fatalf("seed %d %s: %s; model %s", seed, id, view.State, m.state)
			}
		}
	}
}
