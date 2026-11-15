package assistant

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (f world) card(id, tool, target, expected string, input ...commands.Value) CardView {
	f.t.Helper()
	view, err := f.r.Prepare("w1", "j1", "c1", Card{ID: id, Tool: tool, Target: target, Expected: expected, Input: input, Note: "synthetic note"}, bot)
	f.must(err)
	return view
}

func (f world) confirm(view CardView, person string, actor Actor) (CardView, error) {
	return f.r.Confirm("w1", view.Card.ID, Confirmation{Digest: view.Digest, Rationale: "human rationale", Answer: f.answer(person)}, actor)
}

func cards(t *testing.T) world {
	f := setup(t)
	f.expose("statement.read", StageRead, "total")
	f.expose("statement.issue", StageRequest)
	f.expose("statement.approve", StageRequest)
	f.expose("statement.download", StageRequest)
	f.job("j1", "c1")
	return f
}

func TestPrepareForms(t *testing.T) {
	f := cards(t)
	good := Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1"}
	change := func(edit func(*Card)) Card {
		c := good
		edit(&c)
		return c
	}
	for name, c := range map[string]Card{
		"no identifier":         change(func(c *Card) { c.ID = "" }),
		"no tool":               change(func(c *Card) { c.Tool = "" }),
		"no target":             change(func(c *Card) { c.Target = "" }),
		"an expected as text":   change(func(c *Card) { c.Expected = "a b" }),
		"a repeated input":      change(func(c *Card) { c.Input = []commands.Value{{Name: "a", Value: "1"}, {Name: "a", Value: "1"}} }),
		"a long note":           change(func(c *Card) { c.Note = strings.Repeat("a", maxTextBytes+1) }),
		"a note not in utf8":    change(func(c *Card) { c.Note = "\xff" }),
		"an input name as text": change(func(c *Card) { c.Input = []commands.Value{{Name: "", Value: "1"}} }),
	} {
		expect(t, name, second(f.r.Prepare("w1", "j1", "c1", c, bot)), ErrInvalid)
	}
	expect(t, "no workspace", second(f.r.Prepare("", "j1", "c1", good, bot)), ErrInvalid)
	expect(t, "no job", second(f.r.Prepare("w1", "", "c1", good, bot)), ErrInvalid)
	expect(t, "no conversation", second(f.r.Prepare("w1", "j1", "", good, bot)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Prepare("w1", "j1", "c1", good, Actor{Kind: KindAssistant})), ErrInvalid)
	reason(t, "a human preparing", second(f.r.Prepare("w1", "j1", "c1", good, ann)), ReasonNotAssistant)
	expect(t, "an unknown job", second(f.r.Prepare("w1", "j9", "c1", good, bot)), ErrNotFound)
	reason(t, "an unattached conversation", second(f.r.Prepare("w1", "j1", "c9", good, bot)), ReasonNotAttached)
	reason(t, "a tool that is not exposed", second(f.r.Prepare("w1", "j1", "c1", change(func(c *Card) { c.Tool = "statement.cost" }), bot)), ReasonNotExposed)
	reason(t, "a read tool", second(f.r.Prepare("w1", "j1", "c1", change(func(c *Card) { c.Tool = "statement.read" }), bot)), ReasonNotRequest)
	reason(t, "an amount from the model", second(f.r.Prepare("w1", "j1", "c1", change(func(c *Card) { c.Input = []commands.Value{{Name: "amount", Value: "5"}} }), bot)), ReasonModelAmount)
	expect(t, "card of no workspace", second(f.r.CardOf("", "k1")), ErrInvalid)
	expect(t, "card of no identifier", second(f.r.CardOf("w1", "")), ErrInvalid)
	expect(t, "card of an unknown workspace", second(f.r.CardOf("w9", "k1")), ErrNotFound)
	expect(t, "card of an unknown card", second(f.r.CardOf("w1", "k1")), ErrNotFound)
	if f.h.count() != 0 {
		t.Errorf("preparing executed a command")
	}
}

func TestPrepare(t *testing.T) {
	f := cards(t)
	input := []commands.Value{{Name: "count", Value: "2"}}
	view := f.card("k1", "statement.issue", "t1", "r1", input...)
	serial := f.r.workspaces["w1"].cards["k1"].serial
	digest := cardDigest("w1", "j1", serial, Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Input: input}, 1)
	if view.State != CardPending || view.Job != "j1" || view.Digest != digest || len(view.Digest) != 64 || view.Card.Note != "synthetic note" || view.Attempts != 0 {
		t.Errorf("card %+v", view)
	}
	if cardDigest("w2", "j1", serial, view.Card, 1) == digest || cardDigest("w1", "j1", serial, view.Card, 2) == digest || cardDigest("w1", "j1", serial+1, view.Card, 1) == digest ||
		cardDigest("w1", "j1", serial, Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r2", Input: input}, 1) == digest {
		t.Errorf("the digest ignores the workspace, the revision or the expected revision")
	}
	view.Card.Input[0].Value = "9"
	input[0].Value = "8"
	if stored, _ := f.r.CardOf("w1", "k1"); stored.Card.Input[0].Value != "2" {
		t.Errorf("the card is shared with the caller")
	}
	if again, err := f.r.Prepare("w1", "j1", "c1", Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Input: []commands.Value{{Name: "count", Value: "2"}}, Note: "synthetic note"}, bot); err != nil || again.Digest != digest {
		t.Errorf("a repeated card %+v %v", again, err)
	}
	for name, other := range map[string]Card{
		"another input":   {ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Input: []commands.Value{{Name: "count", Value: "3"}}, Note: "synthetic note"},
		"no input":        {ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Note: "synthetic note"},
		"another note":    {ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Input: []commands.Value{{Name: "count", Value: "2"}}},
		"another target":  {ID: "k1", Tool: "statement.issue", Target: "t3", Expected: "r1", Input: []commands.Value{{Name: "count", Value: "2"}}, Note: "synthetic note"},
		"another tool":    {ID: "k1", Tool: "statement.approve", Target: "t1", Expected: "r1", Input: []commands.Value{{Name: "count", Value: "2"}}, Note: "synthetic note"},
		"another version": {ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r2", Input: []commands.Value{{Name: "count", Value: "2"}}, Note: "synthetic note"},
	} {
		expect(t, name, second(f.r.Prepare("w1", "j1", "c1", other, bot)), ErrConflict)
	}
	_, err := f.r.CreateJob("w1", Job{ID: "j2"}, ann)
	f.must(err)
	f.must(second(f.r.Attach("w1", "j2", "c1", ann)))
	expect(t, "the same card in another job", second(f.r.Prepare("w1", "j2", "c1", Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Input: []commands.Value{{Name: "count", Value: "2"}}, Note: "synthetic note"}, bot)), ErrConflict)
	f.card("k2", "statement.approve", "t1", "r1")
	f.card("k3", "statement.download", "t1", "")
	reason(t, "a fourth card", second(f.r.Prepare("w1", "j1", "c1", Card{ID: "k4", Tool: "statement.issue", Target: "t1", Expected: "r1"}, bot)), ReasonBudget)
	job, _ := f.r.JobOf("w1", "j1")
	if job.Cards != 3 || len(job.Ledger) != 3 || job.Ledger[0] != (Entry{Sequence: 1, Kind: EntryCard, Tool: "statement.issue", Target: "t1", Card: "k1", Outcome: CardPending, At: start}) {
		t.Errorf("job %+v", job)
	}
}

func TestConfirmForms(t *testing.T) {
	f := cards(t)
	view := f.card("k1", "statement.issue", "t1", "r1")
	good := Confirmation{Digest: view.Digest, Rationale: "human rationale", Answer: f.answer("ann")}
	expect(t, "no workspace", second(f.r.Confirm("", "k1", good, ann)), ErrInvalid)
	expect(t, "no card", second(f.r.Confirm("w1", "", good, ann)), ErrInvalid)
	expect(t, "a short digest", second(f.r.Confirm("w1", "k1", Confirmation{Digest: view.Digest[1:], Answer: good.Answer}, ann)), ErrInvalid)
	expect(t, "a long rationale", second(f.r.Confirm("w1", "k1", Confirmation{Digest: view.Digest, Rationale: strings.Repeat("a", maxTextBytes+1), Answer: good.Answer}, ann)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Confirm("w1", "k1", good, Actor{ID: "ann"})), ErrInvalid)
	reason(t, "an assistant confirming", second(f.r.Confirm("w1", "k1", good, bot)), ReasonAINotAllowed)
	reason(t, "a system confirming", second(f.r.Confirm("w1", "k1", good, sweeper)), ReasonNotHuman)
	expect(t, "an unknown workspace", second(f.r.Confirm("w9", "k1", good, ann)), ErrNotFound)
	expect(t, "an unknown card", second(f.r.Confirm("w1", "k9", good, ann)), ErrNotFound)
	reason(t, "another person's answer", second(f.r.Confirm("w1", "k1", good, bob)), ReasonNotConfirmer)
	reason(t, "no answer", second(f.r.Confirm("w1", "k1", Confirmation{Digest: view.Digest, Answer: gate.Answer{}}, ann)), ReasonNotConfirmer)
	reason(t, "a changed digest", second(f.r.Confirm("w1", "k1", Confirmation{Digest: strings.Repeat("0", 64), Answer: good.Answer}, ann)), ReasonDigestChanged)
	if f.h.count() != 0 {
		t.Errorf("a refused confirmation executed a command")
	}
}

func TestConfirm(t *testing.T) {
	f := cards(t)
	view := f.card("k1", "statement.issue", "t1", "r1", commands.Value{Name: "count", Value: "2"})
	done, err := f.confirm(view, "ann", ann)
	if err != nil || done.State != CardConfirmed || done.Confirmer != "ann" || done.Operation == "" || done.Code != "completed" || done.Attempts != 1 {
		t.Fatalf("confirmation %+v %v", done, err)
	}
	got, record := f.h.last(), f.lastRecord()
	if got.Client != "" || got.Rationale != "human rationale" || got.Fields["count"] != "2" || got.User != "ann" || record.Client != "" || record.Request != "card-2-1" {
		t.Errorf("the executed call %+v %+v", got, record)
	}
	if again, err := f.confirm(view, "ann", ann); err != nil || again.Attempts != 1 || again.Operation != done.Operation {
		t.Errorf("confirming again %+v %v", again, err)
	}
	reason(t, "another person confirming", second(f.confirm(view, "bob", bob)), ReasonAlreadyDecided)
	reason(t, "declining a confirmed card", second(f.r.Decline("w1", "k1", "no", bob)), ReasonAlreadyDecided)
	job, _ := f.r.JobOf("w1", "j1")
	last := job.Ledger[len(job.Ledger)-1]
	if last.Kind != EntryConfirm || last.Card != "k1" || last.Operation != done.Operation || last.Outcome != "completed" || last.Tool != "statement.issue" || last.Target != "t1" {
		t.Errorf("ledger %+v", last)
	}
	download := f.card("k2", "statement.download", "t1", "")
	if done, err := f.confirm(download, "ann", ann); err != nil || done.State != CardConfirmed || done.Operation != "" {
		t.Errorf("a non-mutating card %+v %v", done, err)
	}
}

func TestConfirmApproval(t *testing.T) {
	f := cards(t)
	f.grant("ann", "", gate.Approve, "c1")
	view := f.card("k1", "statement.approve", "t1", "r1")
	refused, err := f.confirm(view, "ann", ann)
	if err != nil || refused.State != CardPending || refused.Code != "self_approval_denied" || refused.Attempts != 1 || refused.Confirmer != "" {
		t.Errorf("the author confirming the approval %+v %v", refused, err)
	}
	done, err := f.confirm(view, "bob", bob)
	if err != nil || done.State != CardConfirmed || done.Confirmer != "bob" || done.Attempts != 2 {
		t.Errorf("an independent approver %+v %v", done, err)
	}
	if f.lastRecord().Client != "" {
		t.Errorf("the approval ran through the assistant client")
	}
}

func TestUncertain(t *testing.T) {
	f := cards(t)
	view := f.card("k1", "statement.issue", "t1", "r1")
	f.h.panics = true
	unsure, err := f.confirm(view, "bob", bob)
	if err != nil || unsure.State != CardUncertain || unsure.Operation == "" || unsure.Code != "internal_error" || unsure.Confirmer != "bob" {
		t.Fatalf("a failing command %+v %v", unsure, err)
	}
	applied := f.h.count()
	reason(t, "another person confirming", second(f.confirm(view, "ann", ann)), ReasonOutcomeUncertain)
	f.h.panics = false
	repeat, err := f.confirm(view, "bob", bob)
	if err != nil || repeat.State != CardUncertain || repeat.Operation != unsure.Operation || repeat.Attempts != 2 || f.h.count() != applied {
		t.Errorf("the same confirmer repeating %+v %v", repeat, err)
	}
	job, _ := f.r.JobOf("w1", "j1")
	if last := job.Ledger[len(job.Ledger)-1]; !last.Replay || last.Operation != unsure.Operation {
		t.Errorf("the replay is not recorded %+v", last)
	}
	f.must(second(f.r.WithdrawTool("w1", "statement.issue", ann)))
	if stored, _ := f.r.CardOf("w1", "k1"); stored.State != CardUncertain {
		t.Errorf("withdrawing expired an uncertain card")
	}
	reason(t, "a withdrawn tool", second(f.confirm(view, "bob", bob)), ReasonNotExposed)
	f.expose("statement.issue", StageRequest)
	reason(t, "a new exposure", second(f.confirm(view, "bob", bob)), ReasonExposureChanged)
	declined, err := f.r.Decline("w1", "k1", "inspected", ann)
	if err != nil || declined.State != CardDeclined || declined.Decliner != "ann" || declined.Reason != "inspected" {
		t.Errorf("declining an uncertain card %+v %v", declined, err)
	}
	if again, err := f.r.Decline("w1", "k1", "inspected", ann); err != nil || again.State != CardDeclined {
		t.Errorf("declining again %+v %v", again, err)
	}
	reason(t, "another person declining", second(f.r.Decline("w1", "k1", "inspected", bob)), ReasonAlreadyDecided)
	reason(t, "confirming a declined card", second(f.confirm(view, "bob", bob)), ReasonAlreadyDecided)
	job, _ = f.r.JobOf("w1", "j1")
	if last := job.Ledger[len(job.Ledger)-1]; last.Kind != EntryDecline || last.Outcome != OutcomeDeclined || last.Card != "k1" {
		t.Errorf("ledger %+v", last)
	}
}

func TestPendingAfterRefusal(t *testing.T) {
	f := cards(t)
	view := f.card("k1", "statement.issue", "t2", "r1")
	refused, err := f.confirm(view, "ann", ann)
	if err != nil || refused.State != CardPending || refused.Code != "permission_denied" || refused.Operation != "" {
		t.Errorf("a refused confirmation %+v %v", refused, err)
	}
	if done, err := f.r.Decline("w1", "k1", "refused", bob); err != nil || done.State != CardDeclined {
		t.Errorf("declining a pending card %+v %v", done, err)
	}
}

func TestCardExpiry(t *testing.T) {
	f := cards(t)
	issue := f.card("k1", "statement.issue", "t1", "r1")
	approve := f.card("k2", "statement.approve", "t1", "r1")
	f.must(second(f.r.WithdrawTool("w1", "statement.issue", ann)))
	if stored, _ := f.r.CardOf("w1", "k1"); stored.State != CardExpired {
		t.Errorf("withdrawing kept a pending card")
	}
	if stored, _ := f.r.CardOf("w1", "k2"); stored.State != CardPending {
		t.Errorf("withdrawing expired another tool's card")
	}
	reason(t, "confirming an expired card", second(f.confirm(issue, "ann", ann)), ReasonCardExpired)
	reason(t, "declining an expired card", second(f.r.Decline("w1", "k1", "late", ann)), ReasonCardExpired)
	f.advance(limits.Duration)
	if stored, _ := f.r.CardOf("w1", "k2"); stored.State != CardExpired {
		t.Errorf("a pending card after the job duration is %s", stored.State)
	}
	reason(t, "confirming after the job duration", second(f.confirm(approve, "bob", bob)), ReasonCardExpired)
	g := cards(t)
	held := g.card("k1", "statement.approve", "t1", "r1")
	g.must(second(g.r.ProposeTool("w1", Tool{Command: "statement.approve", Stage: StageRequest, Results: []string{"x"}}, ann)))
	reason(t, "a card of a changed tool", second(g.confirm(held, "bob", bob)), ReasonCardExpired)
	h := cards(t)
	open := h.card("k1", "statement.approve", "t1", "r1")
	h.advance(limits.Duration - time.Nanosecond)
	if done, err := h.confirm(open, "bob", bob); err != nil || done.State != CardConfirmed {
		t.Errorf("confirming just before the job duration %+v %v", done, err)
	}
	k := cards(t)
	pending := k.card("k1", "statement.approve", "t1", "r1")
	k.must(second(k.r.CloseJob("w1", "j1", ann)))
	reason(t, "confirming a card of a closed job", second(k.confirm(pending, "bob", bob)), ReasonCardExpired)
}

func TestDeclineForms(t *testing.T) {
	f := cards(t)
	f.card("k1", "statement.issue", "t1", "r1")
	expect(t, "no workspace", second(f.r.Decline("", "k1", "no", ann)), ErrInvalid)
	expect(t, "no card", second(f.r.Decline("w1", "", "no", ann)), ErrInvalid)
	expect(t, "a reason as text", second(f.r.Decline("w1", "k1", "not now", ann)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Decline("w1", "k1", "no", Actor{ID: "ann"})), ErrInvalid)
	reason(t, "an assistant declining", second(f.r.Decline("w1", "k1", "no", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.r.Decline("w9", "k1", "no", ann)), ErrNotFound)
	expect(t, "an unknown card", second(f.r.Decline("w1", "k9", "no", ann)), ErrNotFound)
}

func TestPurge(t *testing.T) {
	f := cards(t)
	f.card("k1", "statement.issue", "t1", "r1")
	f.must(second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)))
	f.must(second(f.r.CreateJob("w1", Job{ID: "j2"}, ann)))
	f.must(second(f.r.Attach("w1", "j2", "c1", ann)))
	if n, err := f.r.Purge("", sweeper); n != 0 || err != ErrInvalid {
		t.Errorf("no workspace %d %v", n, err)
	}
	if n, _ := f.r.Purge("w1", ann); n != 0 {
		t.Errorf("a human purging counted %d", n)
	}
	expect(t, "an invalid actor", second(f.r.Purge("w1", Actor{Kind: KindSystem})), ErrInvalid)
	reason(t, "a human purging", second(f.r.Purge("w1", ann)), ReasonNotSystem)
	reason(t, "an assistant purging", second(f.r.Purge("w1", bot)), ReasonNotSystem)
	if n, err := f.r.Purge("w9", sweeper); n != 0 || err != nil {
		t.Errorf("an unknown workspace %d %v", n, err)
	}
	f.advance(time.Minute)
	f.must(second(f.r.CloseJob("w1", "j1", ann)))
	journal := len(f.c.Journal(0, 1000))
	f.advance(limits.Retention - time.Nanosecond)
	if n, err := f.r.Purge("w1", sweeper); n != 0 || err != nil {
		t.Errorf("purging before the retention %d %v", n, err)
	}
	f.advance(time.Nanosecond)
	if n, err := f.r.Purge("w1", sweeper); n != 1 || err != nil {
		t.Errorf("purging at the retention %d %v", n, err)
	}
	expect(t, "a purged job", second(f.r.JobOf("w1", "j1")), ErrNotFound)
	expect(t, "a purged card", second(f.r.CardOf("w1", "k1")), ErrNotFound)
	if _, err := f.r.JobOf("w1", "j2"); err != nil {
		t.Errorf("an open job within its retention was purged")
	}
	if len(f.c.Journal(0, 1000)) != journal {
		t.Errorf("purging changed the commands journal")
	}
	f.must(second(f.r.CreateJob("w1", Job{ID: "j1"}, ann)))
	f.conversation("c2", ann)
	f.must(second(f.r.Attach("w1", "j1", "c2", ann)))
	f.must(second(f.r.Prepare("w1", "j1", "c2", Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1"}, bot)))
	g := setup(t)
	g.job("j1", "c1")
	g.advance(limits.Duration + limits.Retention - time.Nanosecond)
	if n, _ := g.r.Purge("w1", sweeper); n != 0 {
		t.Errorf("an expired job purged before the retention")
	}
	g.advance(time.Nanosecond)
	if n, _ := g.r.Purge("w1", sweeper); n != 1 {
		t.Errorf("an expired job kept after the retention")
	}
}

func TestReuseAfterPurge(t *testing.T) {
	f := cards(t)
	f.expose("statement.draft", StageDraft)
	first := f.card("k1", "statement.issue", "t1", "r1")
	done, err := f.confirm(first, "ann", ann)
	f.must(err)
	drafted, err := f.r.Invoke("w1", "j1", f.call("c1", "statement.draft", "t3", "r1"), bot)
	f.must(err)
	f.must(second(f.r.CloseJob("w1", "j1", ann)))
	f.advance(limits.Retention)
	f.must(second(f.r.Purge("w1", sweeper)))
	f.h.targets["w1/t1"].Revision, f.h.targets["w1/t3"].Revision = "r1", "r1"
	f.job("j1", "c2")
	applied := f.h.count()
	again, err := f.r.Invoke("w1", "j1", f.call("c2", "statement.draft", "t3", "r1"), bot)
	if err != nil || again.Replay || again.Operation == drafted.Operation || f.h.count() != applied+1 {
		t.Errorf("a call of a reused job identifier replayed %+v %v", again, err)
	}
	view, err := f.r.Prepare("w1", "j1", "c2", Card{ID: "k1", Tool: "statement.issue", Target: "t1", Expected: "r1", Note: "synthetic note"}, bot)
	f.must(err)
	if view.Digest == first.Digest {
		t.Errorf("a reused card identifier has the old digest")
	}
	confirmed, err := f.confirm(view, "ann", ann)
	if err != nil || confirmed.State != CardConfirmed || confirmed.Operation == done.Operation || f.h.count() != applied+2 {
		t.Errorf("a confirmation of a reused card identifier replayed %+v %v", confirmed, err)
	}
}

func TestLongIdentifiers(t *testing.T) {
	f := cards(t)
	long := strings.Repeat("j", maxIDBytes)
	f.must(second(f.r.CreateJob("w1", Job{ID: long}, ann)))
	f.must(second(f.r.Attach("w1", long, "c1", ann)))
	if reply, err := f.r.Invoke("w1", long, f.call("c1", "statement.read", "t1", ""), bot); err != nil || reply.Status != "completed" {
		t.Errorf("a call of a long job identifier %+v %v", reply, err)
	}
	view, err := f.r.Prepare("w1", long, "c1", Card{ID: strings.Repeat("k", maxIDBytes), Tool: "statement.issue", Target: "t1", Expected: "r1"}, bot)
	f.must(err)
	if done, err := f.confirm(view, "ann", ann); err != nil || done.State != CardConfirmed {
		t.Errorf("a confirmation of a long card identifier %+v %v", done, err)
	}
}

func TestAttemptLimit(t *testing.T) {
	f := cards(t)
	view := f.card("k1", "statement.issue", "t2", "r1")
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if got, err := f.confirm(view, "ann", ann); err != nil || got.Attempts != attempt || got.State != CardPending {
			t.Fatalf("attempt %d %+v %v", attempt, got, err)
		}
	}
	reason(t, "an attempt above the limit", second(f.confirm(view, "ann", ann)), ReasonBudget)
	if done, err := f.r.Decline("w1", "k1", "refused", ann); err != nil || done.State != CardDeclined {
		t.Errorf("declining after the attempts %+v %v", done, err)
	}
}

func TestPrepareRepeatInAnyOrder(t *testing.T) {
	f := cards(t)
	input := []commands.Value{{Name: "count", Value: "2"}, {Name: "ref", Value: "x"}}
	first := f.card("k1", "statement.download", "t1", "", input...)
	again := f.card("k1", "statement.download", "t1", "", input[1], input[0])
	if again.Digest != first.Digest {
		t.Errorf("a resent card in another order %+v", again)
	}
	expect(t, "same names with another value", second(f.r.Prepare("w1", "j1", "c1", Card{ID: "k1", Tool: "statement.download", Target: "t1", Input: []commands.Value{input[0], {Name: "ref", Value: "y"}}, Note: "synthetic note"}, bot)), ErrConflict)
}
