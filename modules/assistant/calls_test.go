package assistant

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestInvokeForms(t *testing.T) {
	f := setup(t)
	f.expose("statement.read", StageRead, "total")
	f.job("j1", "c1")
	good := f.call("c1", "statement.read", "t1", "")
	change := func(edit func(*Invocation)) Invocation {
		c := good
		edit(&c)
		return c
	}
	for name, c := range map[string]Invocation{
		"no conversation":       change(func(c *Invocation) { c.Conversation = "" }),
		"no tool":               change(func(c *Invocation) { c.Tool = "" }),
		"no target":             change(func(c *Invocation) { c.Target = "" }),
		"an expected as text":   change(func(c *Invocation) { c.Expected = "a b" }),
		"an input name as text": change(func(c *Invocation) { c.Input = []commands.Value{{Name: "a b", Value: "1"}} }),
		"a repeated input":      change(func(c *Invocation) { c.Input = []commands.Value{{Name: "a", Value: "1"}, {Name: "a", Value: "2"}} }),
		"a long input": change(func(c *Invocation) {
			c.Input = []commands.Value{{Name: "a", Value: strings.Repeat("x", maxTextBytes+1)}}
		}),
		"an input not in utf8": change(func(c *Invocation) { c.Input = []commands.Value{{Name: "a", Value: "\xff"}} }),
	} {
		expect(t, name, second(f.r.Invoke("w1", "j1", c, bot)), ErrInvalid)
	}
	expect(t, "no workspace", second(f.r.Invoke("", "j1", good, bot)), ErrInvalid)
	expect(t, "no job", second(f.r.Invoke("w1", "", good, bot)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Invoke("w1", "j1", good, Actor{ID: "bot"})), ErrInvalid)
	reason(t, "a human invoking", second(f.r.Invoke("w1", "j1", good, ann)), ReasonNotAssistant)
	reason(t, "a system invoking", second(f.r.Invoke("w1", "j1", good, sweeper)), ReasonNotAssistant)
	expect(t, "an unknown workspace", second(f.r.Invoke("w9", "j1", good, bot)), ErrNotFound)
	expect(t, "an unknown job", second(f.r.Invoke("w1", "j9", good, bot)), ErrNotFound)
	reason(t, "an unknown conversation", second(f.r.Invoke("w1", "j1", change(func(c *Invocation) { c.Conversation = "c9" }), bot)), ReasonNotAttached)
	f.conversation("c2", ann)
	reason(t, "an unattached conversation", second(f.r.Invoke("w1", "j1", change(func(c *Invocation) { c.Conversation = "c2" }), bot)), ReasonNotAttached)
	reason(t, "no answer", second(f.r.Invoke("w1", "j1", change(func(c *Invocation) { c.Answer = gate.Answer{} }), bot)), ReasonNotOwner)
	reason(t, "another person's answer", second(f.r.Invoke("w1", "j1", change(func(c *Invocation) { c.Answer = f.answer("bob") }), bot)), ReasonNotOwner)
	reason(t, "a tool that is not exposed", second(f.r.Invoke("w1", "j1", change(func(c *Invocation) { c.Tool = "statement.issue" }), bot)), ReasonNotExposed)
	f.must(second(f.r.ProposeTool("w1", Tool{Command: "statement.cost", Stage: StageRequest}, ann)))
	reason(t, "a proposed tool", second(f.r.Invoke("w1", "j1", change(func(c *Invocation) { c.Tool = "statement.cost" }), bot)), ReasonNotExposed)
	if f.h.count() != 0 || len(f.c.Journal(0, 100)) != 0 {
		t.Errorf("a refused call reached the executor")
	}
	if job, _ := f.r.JobOf("w1", "j1"); job.Calls != 0 || len(job.Ledger) != 0 {
		t.Errorf("a refused call was counted %+v", job)
	}
}

func TestInvokeRead(t *testing.T) {
	f := setup(t)
	f.expose("statement.read", StageRead, "zeta", "yank", "total", "name", "omega")
	f.job("j1", "c1")
	reply, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", "", commands.Value{Name: "ref", Value: "x1"}), bot)
	f.must(err)
	want := map[string]string{"name": "가나", "total": "12345678", "yank": "abcdefgh"}
	if reply.Status != "completed" || reply.Code != "" || reply.Operation != "" || reply.Replay || len(reply.Fields) != len(want) || strings.Join(reply.Truncated, ",") != "name,total" ||
		reply.Calls != 1 || reply.Remaining != limits.Calls-1 || reply.Targets != 1 {
		t.Errorf("reply %+v", reply)
	}
	for name, value := range want {
		if reply.Fields[name] != value {
			t.Errorf("field %s = %q; want %q", name, reply.Fields[name], value)
		}
	}
	record := f.lastRecord()
	if record.Client != "asst" || record.User != "ann" || record.Command != "statement.read" || record.Request != "assistant-1-1" {
		t.Errorf("the command journal %+v", record)
	}
	if got := f.h.last(); got.Client != "asst" || got.Fields["ref"] != "x1" {
		t.Errorf("the handler context %+v", got)
	}
	job, err := f.r.JobOf("w1", "j1")
	f.must(err)
	entry := Entry{Sequence: 1, Kind: EntryCall, Tool: "statement.read", Target: "t1", Outcome: "completed", At: start}
	if len(job.Ledger) != 1 || job.Ledger[0] != entry || job.Calls != 1 || job.Targets != 1 {
		t.Errorf("ledger %+v", job)
	}
	job.Ledger[0].Tool = "changed"
	if again, _ := f.r.JobOf("w1", "j1"); again.Ledger[0].Tool != "statement.read" {
		t.Errorf("the ledger is shared with the caller")
	}
	other, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t2", ""), bot)
	if err != nil || other.Code != "permission_denied" || other.Status != "permission_denied" || other.Fields != nil || other.Calls != 2 || other.Targets != 2 {
		t.Errorf("another customer's target %+v %v", other, err)
	}
	f.h.fields = map[string]string{"zeta": strings.Repeat("é", 4)}
	exact, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)
	if err != nil || exact.Fields["zeta"] != strings.Repeat("é", 4) || len(exact.Truncated) != 0 || exact.Targets != 2 {
		t.Errorf("a value of exactly the byte limit %+v %v", exact, err)
	}
	f.h.fields = map[string]string{"zeta": "é" + strings.Repeat("a", 7)}
	if cut, _ := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot); cut.Fields["zeta"] != "é"+strings.Repeat("a", 6) {
		t.Errorf("a cut after a two-byte letter %q", cut.Fields["zeta"])
	}
	f.h.fields = map[string]string{"zeta": "a" + strings.Repeat("가", 3)}
	if cut, _ := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot); cut.Fields["zeta"] != "a가가" {
		t.Errorf("a cut inside a letter %q", cut.Fields["zeta"])
	}
	reason(t, "the call budget", second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)), ReasonBudget)
}

func TestInvokeDraft(t *testing.T) {
	f := setup(t)
	f.expose("statement.draft", StageDraft, "state")
	f.expose("statement.issue", StageRequest)
	f.job("j1", "c1")
	reason(t, "a request tool", second(f.r.Invoke("w1", "j1", f.call("c1", "statement.issue", "t1", "r1"), bot)), ReasonRequestOnly)
	reason(t, "an amount from the model", second(f.r.Invoke("w1", "j1", f.call("c1", "statement.draft", "t1", "r1", commands.Value{Name: "amount", Value: "1.00"}), bot)), ReasonModelAmount)
	input := []commands.Value{{Name: "note", Value: "synthetic"}, {Name: "count", Value: "2"}}
	first, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.draft", "t1", "r1", input...), bot)
	if err != nil || first.Status != "completed" || first.Operation == "" || first.Replay {
		t.Fatalf("a draft %+v %v", first, err)
	}
	applied := f.h.count()
	again, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.draft", "t1", "r1", input[1], input[0]), bot)
	if err != nil || !again.Replay || again.Operation != first.Operation || f.h.count() != applied || again.Calls != 2 {
		t.Errorf("a resent draft %+v %v", again, err)
	}
	stale, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.draft", "t1", "r1", commands.Value{Name: "count", Value: "3"}), bot)
	if err != nil || stale.Code != "revision_conflict" || stale.Operation != "" {
		t.Errorf("a draft against an old revision %+v %v", stale, err)
	}
	f.h.panics = true
	broken, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.draft", "t1", f.h.revision("t1"), commands.Value{Name: "count", Value: "4"}), bot)
	if err != nil || broken.Code != "internal_error" || broken.Operation == "" {
		t.Errorf("a failing handler %+v %v", broken, err)
	}
	job, _ := f.r.JobOf("w1", "j1")
	if len(job.Ledger) != 4 || !job.Ledger[1].Replay || job.Ledger[3].Operation != broken.Operation || job.Ledger[3].Outcome != "internal_error" {
		t.Errorf("ledger %+v", job.Ledger)
	}
}

func TestCallKeys(t *testing.T) {
	c := &fake{entries: []commands.Entry{
		{ID: "x.read", Permission: commands.Permission{Action: "view", Kind: "statement"}, Supported: true},
		{ID: "x.draft", Permission: commands.Permission{Action: "view", Kind: "statement"}, Mutating: true, Supported: true},
	}}
	now := start
	r, err := NewRegistry(Config{Commands: c, Client: "asst", Protected: protected}, limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []Tool{{Command: "x.read", Stage: StageRead}, {Command: "x.draft", Stage: StageSimulate}} {
		if _, err := r.ProposeTool("w1", tool, ann); err != nil {
			t.Fatal(err)
		}
		if _, err := r.ApproveTool("w1", tool.Command, bob); err != nil {
			t.Fatal(err)
		}
	}
	for _, step := range []error{second(r.CreateJob("w1", Job{ID: "j1"}, ann)), second(r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: now.Add(time.Hour)}, ann)),
		second(r.Attach("w1", "j1", "c1", ann))} {
		if step != nil {
			t.Fatal(step)
		}
	}
	answer := setup(t).answer("ann")
	input := []commands.Value{{Name: "b", Value: "2"}, {Name: "a", Value: "1"}}
	for _, call := range []Invocation{
		{Conversation: "c1", Tool: "x.read", Target: "t1", Input: input, Answer: answer},
		{Conversation: "c1", Tool: "x.draft", Target: "t1", Expected: "r1", Input: input, Answer: answer},
	} {
		if _, err := r.Invoke("w1", "j1", call, bot); err != nil {
			t.Fatal(err)
		}
	}
	read, draft := c.calls[0], c.calls[1]
	if read.Key != "" || read.Expected != "" || read.Client != "asst" || read.Request != "assistant-1-1" || read.Rationale != "" || len(read.Fields) != 2 {
		t.Errorf("a reading call %+v", read)
	}
	if draft.Key != key(callTag, "w1", "j1", "1", "x.draft", "t1", "r1", "a", "1", "b", "2") || draft.Expected != "r1" || draft.Request != "assistant-1-2" {
		t.Errorf("a drafting call %+v", draft)
	}
	if draft.Fields[0] != input[0] {
		t.Errorf("the input order was changed in the call")
	}
	if len(draft.Key) != 64 || key(callTag, "w2", "j1", "1", "x.draft", "t1", "r1", "a", "1", "b", "2") == draft.Key {
		t.Errorf("keys do not differ by workspace")
	}
	if key(callTag, "ab", "c") == key(callTag, "a", "bc") || key(callTag, "a") == key(cardTag, "a") {
		t.Errorf("keys are ambiguous")
	}
	c.result = func(commands.Call) commands.Result {
		return commands.Result{Err: &commands.Error{Code: commands.InternalError, Operation: "op-x"}}
	}
	reply, err := r.Invoke("w1", "j1", Invocation{Conversation: "c1", Tool: "x.draft", Target: "t1", Expected: "r1", Answer: answer}, bot)
	if err != nil || reply.Status != "internal_error" || reply.Code != "internal_error" || reply.Operation != "op-x" || reply.Fields != nil {
		t.Errorf("an error result %+v %v", reply, err)
	}
}

func TestBudgets(t *testing.T) {
	f := setup(t)
	f.expose("statement.read", StageRead, "total")
	f.job("j1", "c1")
	for _, target := range []string{"t1", "t3", "t4"} {
		f.must(second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", target, ""), bot)))
	}
	reason(t, "a fourth target", second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t5", ""), bot)), ReasonBudget)
	if reply, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot); err != nil || reply.Targets != 3 || reply.Calls != 4 {
		t.Errorf("a known target %+v %v", reply, err)
	}
	f.must(second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t3", ""), bot)))
	reason(t, "a sixth call", second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)), ReasonBudget)
	if f.h.count() != 5 {
		t.Errorf("executed %d calls", f.h.count())
	}
	f.job("j2", "c2")
	f.advance(limits.Duration - time.Nanosecond)
	f.must(second(f.r.Invoke("w1", "j2", f.call("c2", "statement.read", "t1", ""), bot)))
	f.advance(time.Nanosecond)
	reason(t, "a job past its duration", second(f.r.Invoke("w1", "j2", f.call("c2", "statement.read", "t1", ""), bot)), ReasonBudget)
	f.job("j3", "c3")
	f.must(second(f.r.CloseJob("w1", "j3", ann)))
	reason(t, "a closed job", second(f.r.Invoke("w1", "j3", f.call("c3", "statement.read", "t1", ""), bot)), ReasonJobClosed)
	_, err := f.r.CreateJob("w1", Job{ID: "j4"}, ann)
	f.must(err)
	_, err = f.r.OpenConversation("w1", Conversation{ID: "c4", Owner: "ann", Expires: f.now.Add(time.Minute)}, ann)
	f.must(err)
	f.must(second(f.r.Attach("w1", "j4", "c4", ann)))
	f.advance(time.Minute)
	reason(t, "an expired conversation", second(f.r.Invoke("w1", "j4", f.call("c4", "statement.read", "t1", ""), bot)), ReasonConversation)
}

func TestConcurrentBudget(t *testing.T) {
	f := setup(t)
	f.expose("statement.read", StageRead, "total")
	f.job("j1", "c1")
	calls := make([]Invocation, 20)
	for index := range calls {
		calls[index] = f.call("c1", "statement.read", "t1", "")
	}
	var wait sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for index := range calls {
		wait.Add(1)
		go func(call Invocation) {
			defer wait.Done()
			if _, err := f.r.Invoke("w1", "j1", call, bot); err == nil {
				mu.Lock()
				done++
				mu.Unlock()
			}
		}(calls[index])
	}
	wait.Wait()
	if job, _ := f.r.JobOf("w1", "j1"); done != limits.Calls || job.Calls != limits.Calls || len(job.Ledger) != limits.Calls || f.h.count() != limits.Calls {
		t.Errorf("done %d, job %+v, executed %d", done, job, f.h.count())
	}
}

func TestNarrowResults(t *testing.T) {
	l := limits
	l.Bytes = 2
	f := setupWith(t, l)
	f.expose("statement.read", StageRead, "name", "zeta")
	f.job("j1", "c1")
	f.h.fields = map[string]string{"name": "가", "zeta": "ab"}
	reply, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)
	if err != nil || reply.Fields["name"] != "" || reply.Fields["zeta"] != "ab" || strings.Join(reply.Truncated, ",") != "name" {
		t.Errorf("a letter wider than the limit %+v %v", reply, err)
	}
}
