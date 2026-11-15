package assistant

import (
	"strings"
	"testing"
	"time"
)

func TestConversations(t *testing.T) {
	f := setup(t)
	good := Conversation{ID: "c1", Owner: "ann", Expires: start.Add(time.Hour)}
	for name, c := range map[string]struct {
		workspace    string
		conversation Conversation
		actor        Actor
	}{
		"no workspace":     {"", good, ann},
		"no identifier":    {"w1", Conversation{Owner: "ann", Expires: good.Expires}, ann},
		"no owner":         {"w1", Conversation{ID: "c1", Expires: good.Expires}, ann},
		"no expiry":        {"w1", Conversation{ID: "c1", Owner: "ann"}, ann},
		"an invalid actor": {"w1", good, Actor{ID: "ann"}},
	} {
		expect(t, name, second(f.r.OpenConversation(c.workspace, c.conversation, c.actor)), ErrInvalid)
	}
	reason(t, "an assistant opening", second(f.r.OpenConversation("w1", good, bot)), ReasonAINotAllowed)
	reason(t, "another person opening", second(f.r.OpenConversation("w1", good, bob)), ReasonNotOwner)
	for name, expires := range map[string]time.Time{"now": start, "the past": start.Add(-time.Second), "after a year": start.Add(maxConversationAge + time.Second)} {
		reason(t, "an expiry "+name, second(f.r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: expires}, ann)), ReasonConversationExpiry)
	}
	longest, err := f.r.OpenConversation("w1", Conversation{ID: "c2", Owner: "ann", Expires: start.Add(maxConversationAge).In(time.FixedZone("x", 3600))}, ann)
	if err != nil || longest.Expires.Location() != time.UTC {
		t.Errorf("a year ahead %+v %v", longest, err)
	}
	opened, err := f.r.OpenConversation("w1", good, ann)
	if err != nil || opened != good {
		t.Errorf("opening %+v %v", opened, err)
	}
	if again, err := f.r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: good.Expires.In(time.FixedZone("y", -3600))}, ann); err != nil || again != good {
		t.Errorf("the repeated opening %+v %v", again, err)
	}
	expect(t, "another expiry", second(f.r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: good.Expires.Add(time.Second)}, ann)), ErrConflict)
	expect(t, "another opener", second(f.r.OpenConversation("w1", good, bob)), ErrConflict)
	expect(t, "an assistant repeat", second(f.r.OpenConversation("w1", good, bot)), ErrRejected)
	l := limits
	l.Conversations = 1
	g := setupWith(t, l)
	g.conversation("c1", ann)
	expect(t, "a second conversation", second(g.r.OpenConversation("w1", Conversation{ID: "c2", Owner: "ann", Expires: start.Add(time.Hour)}, ann)), ErrLimit)
}

func TestDeleteConversation(t *testing.T) {
	f := setup(t)
	f.expose("statement.read", StageRead, "total")
	f.job("j1", "c1")
	f.must(second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)))
	expect(t, "no workspace", f.r.DeleteConversation("", "c1", ann), ErrInvalid)
	expect(t, "no identifier", f.r.DeleteConversation("w1", "", ann), ErrInvalid)
	expect(t, "an invalid actor", f.r.DeleteConversation("w1", "c1", Actor{ID: "ann"}), ErrInvalid)
	reason(t, "an assistant deleting", f.r.DeleteConversation("w1", "c1", bot), ReasonAINotAllowed)
	expect(t, "an unknown workspace", f.r.DeleteConversation("w9", "c1", ann), ErrNotFound)
	expect(t, "an unknown conversation", f.r.DeleteConversation("w1", "c9", ann), ErrNotFound)
	reason(t, "another person deleting", f.r.DeleteConversation("w1", "c1", bob), ReasonNotOwner)
	reason(t, "a system deleting before expiry", f.r.DeleteConversation("w1", "c1", sweeper), ReasonConversationExpiry)
	f.must(f.r.DeleteConversation("w1", "c1", ann))
	expect(t, "deleting again", f.r.DeleteConversation("w1", "c1", ann), ErrNotFound)
	job, err := f.r.JobOf("w1", "j1")
	if err != nil || len(job.Ledger) != 1 || job.Calls != 1 {
		t.Errorf("deleting a conversation changed the ledger %+v %v", job, err)
	}
	if records := f.c.Journal(0, 1000); len(records) != 1 {
		t.Errorf("deleting a conversation changed the commands journal: %d", len(records))
	}
	reason(t, "invoking from a deleted conversation", second(f.r.Invoke("w1", "j1", f.call("c1", "statement.read", "t1", ""), bot)), ReasonNotAttached)
	f.conversation("c2", ann)
	f.must(second(f.r.Attach("w1", "j1", "c2", ann)))
	if reply, err := f.r.Invoke("w1", "j1", f.call("c2", "statement.read", "t1", ""), bot); err != nil || reply.Calls != 2 {
		t.Errorf("resuming from a new conversation %+v %v", reply, err)
	}
	f.advance(24 * time.Hour)
	f.must(f.r.DeleteConversation("w1", "c2", sweeper))
}

func TestJobs(t *testing.T) {
	f := setup(t)
	for name, c := range map[string]struct {
		workspace string
		job       Job
		actor     Actor
	}{
		"no workspace":          {"", Job{ID: "j1"}, ann},
		"no identifier":         {"w1", Job{}, ann},
		"a long purpose":        {"w1", Job{ID: "j1", Purpose: strings.Repeat("a", maxTextBytes+1)}, ann},
		"a purpose not in utf8": {"w1", Job{ID: "j1", Purpose: "\xff"}, ann},
		"an invalid actor":      {"w1", Job{ID: "j1"}, Actor{Kind: KindHuman}},
	} {
		expect(t, name, second(f.r.CreateJob(c.workspace, c.job, c.actor)), ErrInvalid)
	}
	reason(t, "an assistant creating", second(f.r.CreateJob("w1", Job{ID: "j1"}, bot)), ReasonAINotAllowed)
	made, err := f.r.CreateJob("w1", Job{ID: "j1", Purpose: strings.Repeat("a", maxTextBytes)}, ann)
	if err != nil || made.Owner != "ann" || !made.Created.Equal(start) || made.Closed || len(made.Ledger) != 0 {
		t.Errorf("creation %+v %v", made, err)
	}
	f.advance(time.Minute)
	if again, err := f.r.CreateJob("w1", Job{ID: "j1", Purpose: strings.Repeat("a", maxTextBytes)}, ann); err != nil || !again.Created.Equal(start) {
		t.Errorf("the repeated creation %+v %v", again, err)
	}
	expect(t, "another purpose", second(f.r.CreateJob("w1", Job{ID: "j1", Purpose: "b"}, ann)), ErrConflict)
	expect(t, "another creator", second(f.r.CreateJob("w1", Job{ID: "j1", Purpose: strings.Repeat("a", maxTextBytes)}, bob)), ErrConflict)
	expect(t, "job of no workspace", second(f.r.JobOf("", "j1")), ErrInvalid)
	expect(t, "job of no identifier", second(f.r.JobOf("w1", "")), ErrInvalid)
	expect(t, "job of an unknown workspace", second(f.r.JobOf("w9", "j1")), ErrNotFound)
	expect(t, "job of an unknown job", second(f.r.JobOf("w1", "j9")), ErrNotFound)
	l := limits
	l.Jobs = 1
	g := setupWith(t, l)
	g.must(second(g.r.CreateJob("w1", Job{ID: "j1"}, ann)))
	expect(t, "a second job", second(g.r.CreateJob("w1", Job{ID: "j2"}, ann)), ErrLimit)
	l.Workspaces = 1
	h := setupWith(t, l)
	h.must(second(h.r.CreateJob("w1", Job{ID: "j1"}, ann)))
	expect(t, "a second workspace job", second(h.r.CreateJob("w2", Job{ID: "j1"}, ann)), ErrLimit)
	expect(t, "a second workspace conversation", second(h.r.OpenConversation("w2", Conversation{ID: "c1", Owner: "ann", Expires: start.Add(time.Hour)}, ann)), ErrLimit)
}

func TestAttach(t *testing.T) {
	f := setup(t)
	f.must(second(f.r.CreateJob("w1", Job{ID: "j1"}, ann)))
	f.conversation("c1", ann)
	f.conversation("cb", bob)
	for name, c := range map[string]struct{ workspace, job, conversation string }{
		"no workspace": {"", "j1", "c1"}, "no job": {"w1", "", "c1"}, "no conversation": {"w1", "j1", ""},
	} {
		expect(t, name, second(f.r.Attach(c.workspace, c.job, c.conversation, ann)), ErrInvalid)
	}
	expect(t, "an invalid actor", second(f.r.Attach("w1", "j1", "c1", Actor{ID: "ann"})), ErrInvalid)
	reason(t, "an assistant attaching", second(f.r.Attach("w1", "j1", "c1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.r.Attach("w9", "j1", "c1", ann)), ErrNotFound)
	expect(t, "an unknown job", second(f.r.Attach("w1", "j9", "c1", ann)), ErrNotFound)
	expect(t, "an unknown conversation", second(f.r.Attach("w1", "j1", "c9", ann)), ErrNotFound)
	reason(t, "another person's conversation", second(f.r.Attach("w1", "j1", "cb", ann)), ReasonNotOwner)
	reason(t, "another person's job", second(f.r.Attach("w1", "j1", "cb", bob)), ReasonNotOwner)
	if view, err := f.r.Attach("w1", "j1", "c1", ann); err != nil || view.Job.ID != "j1" {
		t.Errorf("attaching %+v %v", view, err)
	}
	f.must(second(f.r.Attach("w1", "j1", "c1", ann)))
	_, err := f.r.OpenConversation("w1", Conversation{ID: "short", Owner: "ann", Expires: start.Add(time.Minute)}, ann)
	f.must(err)
	f.advance(time.Minute)
	reason(t, "an expired conversation", second(f.r.Attach("w1", "j1", "short", ann)), ReasonConversation)
	f.advance(limits.Duration - time.Minute - time.Nanosecond)
	f.conversation("c2", ann)
	f.must(second(f.r.Attach("w1", "j1", "c2", ann)))
	f.advance(time.Nanosecond)
	f.conversation("c3", ann)
	reason(t, "a job past its duration", second(f.r.Attach("w1", "j1", "c3", ann)), ReasonJobClosed)
	f.must(second(f.r.CreateJob("w1", Job{ID: "j2"}, ann)))
	f.must(second(f.r.CloseJob("w1", "j2", ann)))
	reason(t, "a closed job", second(f.r.Attach("w1", "j2", "c3", ann)), ReasonJobClosed)
}

func TestCloseJob(t *testing.T) {
	f := setup(t)
	f.must(second(f.r.CreateJob("w1", Job{ID: "j1"}, ann)))
	expect(t, "no workspace", second(f.r.CloseJob("", "j1", ann)), ErrInvalid)
	expect(t, "no job", second(f.r.CloseJob("w1", "", ann)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.CloseJob("w1", "j1", Actor{ID: "ann"})), ErrInvalid)
	reason(t, "an assistant closing", second(f.r.CloseJob("w1", "j1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.r.CloseJob("w9", "j1", ann)), ErrNotFound)
	expect(t, "an unknown job", second(f.r.CloseJob("w1", "j9", ann)), ErrNotFound)
	reason(t, "another person closing", second(f.r.CloseJob("w1", "j1", bob)), ReasonNotOwner)
	f.advance(time.Minute)
	closed, err := f.r.CloseJob("w1", "j1", ann)
	if err != nil || !closed.Closed || !closed.ClosedAt.Equal(start.Add(time.Minute)) {
		t.Errorf("closing %+v %v", closed, err)
	}
	f.advance(time.Minute)
	if again, err := f.r.CloseJob("w1", "j1", ann); err != nil || !again.ClosedAt.Equal(start.Add(time.Minute)) {
		t.Errorf("closing again %+v %v", again, err)
	}
}

func TestClockNeverGoesBack(t *testing.T) {
	f := setup(t)
	f.advance(time.Hour)
	f.must(second(f.r.CreateJob("w1", Job{ID: "j1"}, ann)))
	f.advance(-30 * time.Minute)
	made, err := f.r.CreateJob("w1", Job{ID: "j2"}, ann)
	if err != nil || !made.Created.Equal(start.Add(time.Hour)) {
		t.Errorf("a job after the clock went back %+v %v", made, err)
	}
}
