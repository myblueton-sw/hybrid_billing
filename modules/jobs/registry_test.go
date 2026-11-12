package jobs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	start      = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	testLimits = Limits{Jobs: 4, Tasks: 6, Attempts: 5, PerType: 4, PerWorkspace: 2, Batch: 8}
	testPolicy = Policy{Lease: 10 * time.Minute, Life: 25 * time.Minute, Attempts: 3, BackoffBase: time.Minute, BackoffCap: 3 * time.Minute}
)

type bench struct {
	t   testing.TB
	r   *Registry
	now *time.Time
}

func open(t testing.TB, limits Limits) bench {
	t.Helper()
	now := start
	registry, err := NewRegistry(limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return bench{t: t, r: registry, now: &now}
}

func (b bench) pass(duration time.Duration) { *b.now = b.now.Add(duration) }

func specs(texts ...string) []TaskSpec {
	made := []TaskSpec{}
	for _, text := range texts {
		name, repeatable := strings.CutSuffix(text, "*")
		made = append(made, TaskSpec{ID: name, Input: "input-" + name, Repeatable: repeatable})
	}
	return made
}

func (b bench) create(workspace, id, kind string, policy Policy, tasks ...string) {
	b.t.Helper()
	if created, err := b.r.Create(workspace, id, kind, policy, specs(tasks...)); err != nil || !created {
		b.t.Fatalf("create %s/%s: %v %v", workspace, id, created, err)
	}
}

func (b bench) lease(kinds string, limit int) []Lease {
	b.t.Helper()
	leases, err := b.r.Lease("worker-1", strings.Split(kinds, ","), limit)
	if err != nil {
		b.t.Fatalf("lease: %v", err)
	}
	return leases
}

func (b bench) one(kinds string) Lease {
	b.t.Helper()
	leases := b.lease(kinds, 1)
	if len(leases) != 1 {
		b.t.Fatalf("%d leases; want one", len(leases))
	}
	return leases[0]
}

func names(leases []Lease) string {
	parts := []string{}
	for _, lease := range leases {
		parts = append(parts, lease.WorkspaceID+"/"+lease.JobID+"/"+lease.TaskID)
	}
	return strings.Join(parts, " ")
}

func (b bench) job(workspace, id string) JobView {
	b.t.Helper()
	view, err := b.r.Job(workspace, id)
	if err != nil {
		b.t.Fatalf("job %s/%s: %v", workspace, id, err)
	}
	counts := map[TaskState]int{}
	for _, shown := range view.Tasks {
		counts[shown.State]++
	}
	total := 0
	for state, count := range view.Counts {
		total += count
		if counts[state] != count {
			b.t.Errorf("the count of %s is %d for %d tasks", state, count, counts[state])
		}
	}
	if total != len(view.Tasks) || view.Attention != counts[Unknown] {
		b.t.Errorf("the detail of %s/%s does not add up: %+v attention %d", workspace, id, view.Counts, view.Attention)
	}
	return view
}

func (b bench) shape(workspace, id string) string {
	b.t.Helper()
	view := b.job(workspace, id)
	parts := []string{string(view.State)}
	if view.Reason != "" {
		parts[0] += "(" + string(view.Reason) + ")"
	}
	for _, shown := range view.Tasks {
		text := fmt.Sprintf("%s=%s/%d", shown.ID, shown.State, shown.Attempts)
		if shown.Code != "" {
			text += "/" + shown.Code
		}
		if shown.Open {
			text += "/open"
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

func (b bench) is(name, workspace, id, want string) {
	b.t.Helper()
	if got := b.shape(workspace, id); got != want {
		b.t.Errorf("%s:\n got %s\nwant %s", name, got, want)
	}
}

func (b bench) err(name string, err, want error) {
	b.t.Helper()
	if !errors.Is(err, want) && !(err == nil && want == nil) {
		b.t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }

func TestCreation(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "close", testPolicy, "a", "b*")
	b.is("a new job", "w1", "j1", "queued a=pending/0 b=pending/0")
	view := b.job("w1", "j1")
	if view.Type != "close" || view.Policy != testPolicy || view.Revision != 1 || !view.Tasks[1].Repeatable || view.Tasks[0].Repeatable || view.Tasks[0].Input != "input-a" || !view.AsOf.Equal(start) {
		t.Errorf("the stored job: %+v", view)
	}
	created, err := b.r.Create("w1", "j1", "close", testPolicy, specs("a", "b*"))
	if created || err != nil {
		t.Errorf("the same creation again: %v %v", created, err)
	}
	other := testPolicy
	other.Attempts = 2
	for name, attempt := range map[string]func() (bool, error){
		"another type":   func() (bool, error) { return b.r.Create("w1", "j1", "send", testPolicy, specs("a", "b*")) },
		"another policy": func() (bool, error) { return b.r.Create("w1", "j1", "close", other, specs("a", "b*")) },
		"another task":   func() (bool, error) { return b.r.Create("w1", "j1", "close", testPolicy, specs("a", "c*")) },
		"another order":  func() (bool, error) { return b.r.Create("w1", "j1", "close", testPolicy, specs("b*", "a")) },
		"another flag":   func() (bool, error) { return b.r.Create("w1", "j1", "close", testPolicy, specs("a", "b")) },
		"a task fewer":   func() (bool, error) { return b.r.Create("w1", "j1", "close", testPolicy, specs("a")) },
		"another input": func() (bool, error) {
			return b.r.Create("w1", "j1", "close", testPolicy, []TaskSpec{{ID: "a", Input: "x"}, {ID: "b", Input: "input-b", Repeatable: true}})
		},
	} {
		b.err(name, second(attempt()), ErrConflict)
	}
	bad := map[string]func(*Policy){
		"no lease duration": func(p *Policy) { p.Lease = 0 }, "a life shorter than the lease": func(p *Policy) { p.Life = p.Lease - 1 }, "no attempts": func(p *Policy) { p.Attempts = 0 },
		"no backoff": func(p *Policy) { p.BackoffBase = 0 }, "a cap below the base": func(p *Policy) { p.BackoffCap = p.BackoffBase - 1 },
	}
	for name, change := range bad {
		policy := testPolicy
		change(&policy)
		b.err(name, second(b.r.Create("w1", "j2", "close", policy, specs("a"))), ErrInvalid)
	}
	for name, attempt := range map[string]func() (bool, error){
		"no tasks":                 func() (bool, error) { return b.r.Create("w1", "j2", "close", testPolicy, nil) },
		"a task twice":             func() (bool, error) { return b.r.Create("w1", "j2", "close", testPolicy, specs("a", "a*")) },
		"a task name that is text": func() (bool, error) { return b.r.Create("w1", "j2", "close", testPolicy, specs("a b")) },
		"an input that is text": func() (bool, error) {
			return b.r.Create("w1", "j2", "close", testPolicy, []TaskSpec{{ID: "a", Input: "see mail"}})
		},
		"no input":                func() (bool, error) { return b.r.Create("w1", "j2", "close", testPolicy, []TaskSpec{{ID: "a"}}) },
		"a type that is text":     func() (bool, error) { return b.r.Create("w1", "j2", "month close", testPolicy, specs("a")) },
		"no workspace":            func() (bool, error) { return b.r.Create("", "j2", "close", testPolicy, specs("a")) },
		"a job name that is text": func() (bool, error) { return b.r.Create("w1", "j/2", "close", testPolicy, specs("a")) },
	} {
		b.err(name, second(attempt()), ErrInvalid)
	}
	six := testPolicy
	six.Attempts = 6
	b.err("a budget above the limit", second(b.r.Create("w1", "j2", "close", six, specs("a"))), ErrLimit)
	b.err("more tasks than a job may have", second(b.r.Create("w1", "j2", "close", testPolicy, specs("a", "b", "c", "d", "e", "f", "g"))), ErrLimit)
	if _, err := b.r.Job("w1", "j2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a refused creation left a job: %v", err)
	}
	for _, id := range []string{"j2", "j3", "j4"} {
		b.create("w1", id, "close", testPolicy, "a")
	}
	b.err("a fifth job in the workspace", second(b.r.Create("w1", "j5", "close", testPolicy, specs("a"))), ErrLimit)
	created, err = b.r.Create("w1", "j1", "close", testPolicy, specs("a", "b*"))
	if created || err != nil {
		t.Errorf("repeating a creation in a full workspace: %v %v", created, err)
	}
	b.create("w2", "j1", "send", testPolicy, "z")
	b.is("the same identifier in another workspace", "w2", "j1", "queued z=pending/0")
	b.is("the first job is untouched", "w1", "j1", "queued a=pending/0 b=pending/0")
	if _, err := b.r.Job("w3", "j1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a job of an unknown workspace: %v", err)
	}
	log, err := b.r.Transitions("w1", "j1")
	if err != nil || len(log) != 1 || log[0] != (Transition{Revision: 1, At: start, Event: EventCreated, State: Queued}) {
		t.Errorf("the record of a creation: %+v %v", log, err)
	}
}

func TestBackoff(t *testing.T) {
	policy := Policy{Lease: time.Minute, Life: time.Minute, Attempts: 5, BackoffBase: time.Minute, BackoffCap: 5 * time.Minute}
	for attempt, want := range map[int]time.Duration{1: time.Minute, 2: 2 * time.Minute, 3: 4 * time.Minute, 4: 5 * time.Minute, 5: 5 * time.Minute, 70: 5 * time.Minute} {
		if got := policy.backoff(attempt); got != want {
			t.Errorf("after attempt %d: %s; want %s", attempt, got, want)
		}
	}
	wide := Policy{BackoffBase: time.Hour, BackoffCap: 1 << 62}
	if got := wide.backoff(200); got != 1<<62 {
		t.Errorf("a delay that would overflow: %d", got)
	}
	exact := Policy{BackoffBase: time.Minute, BackoffCap: 4 * time.Minute}
	if got := exact.backoff(3); got != 4*time.Minute {
		t.Errorf("a delay equal to the cap: %s", got)
	}
}

func TestATaskThroughItsAttempts(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "close", testPolicy, "a", "b")
	first := b.one("close")
	if first.TaskID != "a" || first.Attempt != 1 || first.Token != 1 || first.Input != "input-a" || first.Checkpoint != "" || !first.Expiry.Equal(start.Add(10*time.Minute)) || first.WorkspaceID != "w1" || first.JobID != "j1" {
		t.Errorf("the first lease: %+v", first)
	}
	b.is("after the first grant", "w1", "j1", "running a=leased/1/open b=pending/0")
	b.err("a checkpoint", b.r.Checkpoint(first.Ref(), "page-3"), nil)
	b.pass(4 * time.Minute)
	report, err := b.r.Fail(first.Ref(), "timeout", false)
	if err != nil || report != (Report{State: Pending}) {
		t.Errorf("a retryable failure: %+v %v", report, err)
	}
	b.is("after it", "w1", "j1", "running a=pending/1/timeout b=pending/0")
	if view := b.job("w1", "j1"); !view.Tasks[0].NotBefore.Equal(start.Add(5 * time.Minute)) {
		t.Errorf("the first backoff: %s", view.Tasks[0].NotBefore)
	}
	next := b.one("close")
	if next.TaskID != "b" {
		t.Errorf("a task in backoff was leased: %+v", next)
	}
	b.pass(time.Minute - time.Second)
	if got := b.lease("close", 1); len(got) != 0 {
		t.Errorf("leased a second before the backoff ended: %s", names(got))
	}
	b.pass(time.Second)
	again := b.one("close")
	if again.TaskID != "a" || again.Attempt != 2 || again.Token != 3 || again.Checkpoint != "page-3" {
		t.Errorf("the second attempt: %+v", again)
	}
	b.err("the first attempt's token on the second attempt", second(b.r.Complete(first.Ref(), "done")), ErrStale)
	b.err("a repeat of the first failure after a later grant", second(b.r.Fail(first.Ref(), "timeout", false)), ErrStale)
	report, err = b.r.Fail(again.Ref(), "timeout", false)
	if err != nil || report.State != Pending {
		t.Errorf("the second failure: %+v %v", report, err)
	}
	if view := b.job("w1", "j1"); !view.Tasks[0].NotBefore.Equal(b.now.Add(2 * time.Minute)) {
		t.Errorf("the second backoff: %s", view.Tasks[0].NotBefore)
	}
	b.pass(2 * time.Minute)
	last := b.one("close")
	report, err = b.r.Fail(last.Ref(), "timeout", false)
	if err != nil || report.State != TaskFailed {
		t.Errorf("a retryable failure at the end of the budget: %+v %v", report, err)
	}
	b.is("the budget used up", "w1", "j1", "running a=failed/3/timeout b=leased/1/open")
	repeated, err := b.r.Fail(last.Ref(), "timeout", false)
	if err != nil || repeated != (Report{State: TaskFailed, Replay: true}) {
		t.Errorf("the same failure report again: %+v %v", repeated, err)
	}
	b.err("the same report with another code", second(b.r.Fail(last.Ref(), "refused", false)), ErrStale)
	b.err("the same report as permanent", second(b.r.Fail(last.Ref(), "timeout", true)), ErrStale)
	b.err("a completion after an accepted failure", second(b.r.Complete(last.Ref(), "done")), ErrStale)
	done, err := b.r.Complete(next.Ref(), "final")
	if err != nil || done != (Report{State: TaskSucceeded}) {
		t.Errorf("a completion: %+v %v", done, err)
	}
	b.is("a partial result is failed, never succeeded", "w1", "j1", "failed a=failed/3/timeout b=succeeded/1")
	if view := b.job("w1", "j1"); view.Counts[TaskSucceeded] != 1 || view.Counts[TaskFailed] != 1 || view.Tasks[1].Checkpoint != "final" {
		t.Errorf("the detail of a partial result: %+v", view)
	}
	replay, err := b.r.Complete(next.Ref(), "final")
	if err != nil || replay != (Report{State: TaskSucceeded, Replay: true}) {
		t.Errorf("the same completion again: %+v %v", replay, err)
	}
	b.err("the same completion with another checkpoint", second(b.r.Complete(next.Ref(), "other")), ErrStale)
	b.err("a failure after an accepted completion", second(b.r.Fail(next.Ref(), "timeout", true)), ErrStale)
	b.err("an unknown outcome after an accepted completion", second(b.r.Unknown(next.Ref(), "lost")), ErrStale)
	b.err("an extension of a closed attempt", second(b.r.Extend(next.Ref())), ErrStale)
	b.err("a checkpoint on a closed attempt", b.r.Checkpoint(next.Ref(), "x"), ErrStale)
	before := b.job("w1", "j1")
	b.err("a cancellation of a terminal job", b.r.Cancel("w1", "j1", Requested), ErrRejected)
	b.pass(time.Hour)
	if got := b.lease("close", 4); len(got) != 0 {
		t.Errorf("a terminal job was leased: %s", names(got))
	}
	after := b.job("w1", "j1")
	after.AsOf = before.AsOf
	if !reflect.DeepEqual(before, after) {
		t.Errorf("a terminal job changed")
	}
	log, _ := b.r.Transitions("w1", "j1")
	events := []string{}
	for index, entry := range log {
		events = append(events, fmt.Sprintf("%s:%s:%s", entry.Event, entry.Task, entry.State))
		if entry.Revision != uint64(index)+1 || (index > 0 && entry.At.Before(log[index-1].At)) {
			t.Errorf("transition %d: %+v", index, entry)
		}
	}
	want := "created::queued granted:a:running failed:a:running granted:b:running granted:a:running failed:a:running granted:a:running failed:a:running completed:b:failed"
	if strings.Join(events, " ") != want || log[1].Worker != "worker-1" || before.Revision != uint64(len(log)) {
		t.Errorf("the record:\n got %s\nwant %s", strings.Join(events, " "), want)
	}
	log[0].Event = "tampered"
	if fresh, _ := b.r.Transitions("w1", "j1"); fresh[0].Event != EventCreated {
		t.Errorf("the record aliases what it returns")
	}
}

func TestPermanentFailureAndSuccess(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "close", testPolicy, "a", "b")
	leases := b.lease("close", 2)
	report, err := b.r.Fail(leases[0].Ref(), "rejected", true)
	if err != nil || report.State != TaskFailed {
		t.Errorf("a permanent failure: %+v %v", report, err)
	}
	b.is("no fail-fast", "w1", "j1", "running a=failed/1/rejected b=leased/1/open")
	b.r.Complete(leases[1].Ref(), "done")
	b.is("after the rest ended", "w1", "j1", "failed a=failed/1/rejected b=succeeded/1")

	b.create("w1", "j2", "close", testPolicy, "a", "b")
	leases = b.lease("close", 2)
	b.r.Complete(leases[0].Ref(), "one")
	b.is("one of two done", "w1", "j2", "running a=succeeded/1 b=leased/1/open")
	b.r.Complete(leases[1].Ref(), "two")
	b.is("the whole manifest done", "w1", "j2", "succeeded a=succeeded/1 b=succeeded/1")
	for name, ref := range map[string]Ref{
		"a token of zero":            {WorkspaceID: "w1", JobID: "j2", TaskID: "a"},
		"the token of another task":  {WorkspaceID: "w1", JobID: "j2", TaskID: "a", Token: leases[1].Token},
		"a token that was never out": {WorkspaceID: "w1", JobID: "j2", TaskID: "a", Token: 999},
	} {
		b.err(name, second(b.r.Complete(ref, "one")), ErrStale)
	}
	for name, ref := range map[string]Ref{
		"an unknown task":      {WorkspaceID: "w1", JobID: "j2", TaskID: "z", Token: 1},
		"an unknown job":       {WorkspaceID: "w1", JobID: "j9", TaskID: "a", Token: 1},
		"an unknown workspace": {WorkspaceID: "w9", JobID: "j2", TaskID: "a", Token: leases[0].Token},
	} {
		b.err(name, second(b.r.Complete(ref, "one")), ErrNotFound)
		b.err(name+", failure", second(b.r.Fail(ref, "x", true)), ErrNotFound)
		b.err(name+", unknown", second(b.r.Unknown(ref, "x")), ErrNotFound)
		b.err(name+", extension", second(b.r.Extend(ref)), ErrNotFound)
		b.err(name+", checkpoint", b.r.Checkpoint(ref, "x"), ErrNotFound)
		b.err(name+", resolution", second(b.r.Resolve(ref.WorkspaceID, ref.JobID, ref.TaskID, ref.Token, ResolveFailed)), ErrNotFound)
	}
	bad := Ref{WorkspaceID: "w 1", JobID: "j2", TaskID: "a", Token: 1}
	b.err("a workspace that is text", second(b.r.Complete(bad, "one")), ErrInvalid)
	open := b.create
	open("w1", "j3", "close", testPolicy, "a")
	lease := b.one("close")
	b.err("a checkpoint that is text", second(b.r.Complete(lease.Ref(), "page 3")), ErrInvalid)
	b.err("a stored checkpoint that is text", b.r.Checkpoint(lease.Ref(), "page 3"), ErrInvalid)
	b.err("a code that is text", second(b.r.Fail(lease.Ref(), "it broke", true)), ErrInvalid)
	b.err("an unknown outcome with text", second(b.r.Unknown(lease.Ref(), "no answer")), ErrInvalid)
	b.err("an unknown resolution", second(b.r.Resolve("w1", "j3", "a", lease.Token, "maybe")), ErrInvalid)
	b.is("refused reports change nothing", "w1", "j3", "running a=leased/1/open")
}

func TestExpiryAndTheOpenAttempt(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "close", testPolicy, "once", "again*")
	leases := b.lease("close", 2)
	once, again := leases[0], leases[1]
	b.pass(10*time.Minute - time.Nanosecond)
	if view := b.job("w1", "j1"); view.Overdue != 0 {
		t.Errorf("overdue before the expiry: %d", view.Overdue)
	}
	expiry, err := b.r.Extend(once.Ref())
	if err != nil || !expiry.Equal(b.now.Add(10*time.Minute)) {
		t.Errorf("an extension an instant before the expiry: %s %v", expiry, err)
	}
	b.pass(time.Nanosecond)
	if view := b.job("w1", "j1"); view.Overdue != 1 || view.State != Running || view.Tasks[1].State != Leased {
		t.Errorf("a read at the expiry sweeps or hides it: overdue %d, %s", view.Overdue, view.State)
	}
	b.err("an extension at the expiry", second(b.r.Extend(again.Ref())), ErrStale)
	b.is("the repeatable task is back after the sweep", "w1", "j1", "running once=leased/1/open again=pending/1/open")
	if view := b.job("w1", "j1"); !view.Tasks[1].NotBefore.Equal(b.now.Add(time.Minute)) {
		t.Errorf("the backoff is not measured from the sweep: %s", view.Tasks[1].NotBefore)
	}
	b.err("a checkpoint after the expiry", b.r.Checkpoint(again.Ref(), "p"), ErrStale)
	late, err := b.r.Complete(again.Ref(), "late")
	if err != nil || late.State != TaskSucceeded {
		t.Errorf("the holder's completion after its lease expired, with no later grant: %+v %v", late, err)
	}
	b.is("accepted", "w1", "j1", "running once=leased/1/open again=succeeded/1")

	b.pass(8 * time.Minute)
	expiry, err = b.r.Extend(once.Ref())
	if err != nil || !expiry.Equal(start.Add(25*time.Minute)) {
		t.Errorf("an extension is not held to the longest life: %s %v", expiry, err)
	}
	b.pass(7*time.Minute - time.Nanosecond)
	if view := b.job("w1", "j1"); view.Overdue != 0 {
		t.Errorf("overdue an instant before the longest life")
	}
	b.pass(time.Nanosecond)
	b.err("the sweep", b.r.Sweep(), nil)
	b.is("a task that is not repeatable waits for an answer", "w1", "j1", "waiting(outcome_unknown) once=unknown/1/lease_expired/open again=succeeded/1")
	if view := b.job("w1", "j1"); view.Attention != 1 || view.Overdue != 0 {
		t.Errorf("attention %d overdue %d", view.Attention, view.Overdue)
	}
	if got := b.lease("close", 4); len(got) != 0 {
		t.Errorf("an unknown task was leased again: %s", names(got))
	}
	report, err := b.r.Unknown(once.Ref(), "no-answer")
	if err != nil || report != (Report{State: Unknown, Replay: true}) {
		t.Errorf("an unknown report on a task that is already unknown: %+v %v", report, err)
	}
	if view := b.job("w1", "j1"); view.Tasks[0].Code != CodeLeaseExpired {
		t.Errorf("the first code was replaced: %s", view.Tasks[0].Code)
	}
	revision := b.job("w1", "j1").Revision
	report, err = b.r.Fail(once.Ref(), "confirmed-not-sent", false)
	if err != nil || report.State != Pending {
		t.Errorf("the holder confirming that nothing happened: %+v %v", report, err)
	}
	b.is("back in line without a resolution", "w1", "j1", "running once=pending/1/confirmed-not-sent again=succeeded/1")
	if b.job("w1", "j1").Revision != revision+1 {
		t.Errorf("the revision after the late failure")
	}

	c := open(t, testLimits)
	one := testPolicy
	one.Attempts = 1
	c.create("w1", "j1", "close", one, "again*")
	lease := c.one("close")
	c.pass(10 * time.Minute)
	late, err = c.r.Complete(lease.Ref(), "late")
	if err != nil || late.State != TaskSucceeded {
		t.Errorf("a completion in the same call as the sweep of the last attempt: %+v %v", late, err)
	}
	c.is("the last attempt of a repeatable task", "w1", "j1", "succeeded again=succeeded/1/lease_expired")
	c.create("w1", "j2", "close", one, "again*")
	lease = c.one("close")
	c.pass(10 * time.Minute)
	c.r.Sweep()
	c.is("a sweep never makes a task final", "w1", "j2", "waiting(outcome_unknown) again=unknown/1/lease_expired/open")
	c.err("a retry beyond the budget", second(c.r.Resolve("w1", "j2", "again", lease.Token, ResolveRetry)), ErrRejected)
	report, err = c.r.Fail(lease.Ref(), "not-sent", false)
	if err != nil || report.State != TaskFailed {
		t.Errorf("a late retryable failure at the end of the budget: %+v %v", report, err)
	}
	c.is("failed with the holder's code", "w1", "j2", "failed again=failed/1/not-sent")

	d := open(t, testLimits)
	d.create("w1", "j1", "close", testPolicy, "again*")
	old := d.one("close")
	d.pass(13 * time.Minute)
	d.r.Sweep()
	if view := d.job("w1", "j1"); !view.Tasks[0].NotBefore.Equal(d.now.Add(time.Minute)) {
		t.Errorf("the backoff after an expiry is measured from the sweep: %s", view.Tasks[0].NotBefore)
	}
	revision = d.job("w1", "j1").Revision
	report, err = d.r.Fail(old.Ref(), "not-sent", false)
	if view := d.job("w1", "j1"); err != nil || report.State != Pending || !view.Tasks[0].NotBefore.Equal(d.now.Add(time.Minute)) || view.Revision != revision+1 || view.Tasks[0].Open {
		t.Errorf("a late retryable failure on a swept task: %+v %v %+v", report, err, view.Tasks[0])
	}
	d.pass(time.Minute)
	fresh := d.one("close")
	if fresh.Attempt != 2 || fresh.Token <= old.Token {
		t.Errorf("the next grant: %+v", fresh)
	}
	d.pass(10 * time.Minute)
	d.r.Sweep()
	d.pass(2 * time.Minute)
	third := d.one("close")
	d.err("the completion of a worker whose task was granted again", second(d.r.Complete(fresh.Ref(), "late")), ErrStale)
	d.err("its unknown report", second(d.r.Unknown(fresh.Ref(), "lost")), ErrStale)
	d.is("the later holder is untouched", "w1", "j1", "running again=leased/3/not-sent/open")
	d.pass(10 * time.Minute)
	d.r.Sweep()
	d.is("expiry of the last attempt", "w1", "j1", "waiting(outcome_unknown) again=unknown/3/lease_expired/open")
	if view := d.job("w1", "j1"); view.Tasks[0].Token != third.Token {
		t.Errorf("the latest token")
	}
	back := open(t, testLimits)
	back.create("w1", "j1", "close", testPolicy, "a")
	held := back.one("close")
	back.pass(9 * time.Minute)
	back.err("a checkpoint", back.r.Checkpoint(held.Ref(), "p"), nil)
	back.pass(-time.Hour)
	if view := back.job("w1", "j1"); !view.AsOf.Equal(start.Add(9 * time.Minute)) {
		t.Errorf("the registry's time went backwards to %s", view.AsOf)
	}
	back.pass(time.Hour + time.Minute)
	back.err("an extension after the clock recovered", second(back.r.Extend(held.Ref())), ErrStale)
}

func TestUnknownOutcomesAndResolutions(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "send", testPolicy, "a", "b", "c", "d")
	leases := b.lease("send", 2)
	a, second2 := leases[0], leases[1]
	report, err := b.r.Unknown(a.Ref(), "response-lost")
	if err != nil || report != (Report{State: Unknown}) {
		t.Errorf("an unknown outcome: %+v %v", report, err)
	}
	b.is("an unknown outcome beside other work", "w1", "j1", "running a=unknown/1/response-lost/open b=leased/1/open c=pending/0 d=pending/0")
	if view := b.job("w1", "j1"); view.Attention != 1 {
		t.Errorf("attention %d", view.Attention)
	}
	more := b.lease("send", 4)
	if names(more) != "w1/j1/c" {
		t.Errorf("an unknown outcome did not free its slot, or the limit of two per workspace is not kept: %s", names(more))
	}
	for _, lease := range []Lease{second2, more[0]} {
		b.r.Complete(lease.Ref(), "done")
	}
	last := b.one("send")
	b.r.Complete(last.Ref(), "done")
	b.is("only the unknown outcome is left", "w1", "j1", "waiting(outcome_unknown) a=unknown/1/response-lost/open b=succeeded/1 c=succeeded/1 d=succeeded/1")
	b.err("a resolution with the token of another attempt", second(b.r.Resolve("w1", "j1", "a", last.Token, ResolveSucceeded)), ErrStale)
	b.err("a resolution of a task that is not unknown", second(b.r.Resolve("w1", "j1", "b", second2.Token, ResolveFailed)), ErrRejected)
	b.create("w1", "j9", "archive", testPolicy, "never")
	b.err("a resolution with the token of zero on a task that was never granted", second(b.r.Resolve("w1", "j9", "never", 0, ResolveFailed)), ErrStale)
	report, err = b.r.Resolve("w1", "j1", "a", a.Token, ResolveRetry)
	if err != nil || report != (Report{State: Pending}) {
		t.Errorf("a retry: %+v %v", report, err)
	}
	b.is("back to running", "w1", "j1", "running a=pending/1/response-lost b=succeeded/1 c=succeeded/1 d=succeeded/1")
	again, err := b.r.Resolve("w1", "j1", "a", a.Token, ResolveRetry)
	if err != nil || again != (Report{State: Pending, Replay: true}) {
		t.Errorf("the same resolution again: %+v %v", again, err)
	}
	b.err("the same token with another value", second(b.r.Resolve("w1", "j1", "a", a.Token, ResolveSucceeded)), ErrConflict)
	b.err("the worker's completion after the resolution", second(b.r.Complete(a.Ref(), "late")), ErrStale)
	b.err("its failure after the resolution", second(b.r.Fail(a.Ref(), "x", true)), ErrStale)
	retried := b.one("send")
	if retried.TaskID != "a" || retried.Attempt != 2 {
		t.Errorf("the grant after a retry is at once and counts: %+v", retried)
	}
	b.err("the old resolution after a later grant", second(b.r.Resolve("w1", "j1", "a", a.Token, ResolveRetry)), ErrStale)
	b.r.Unknown(retried.Ref(), "response-lost")
	learned, err := b.r.Complete(retried.Ref(), "found-in-target")
	if err != nil || learned.State != TaskSucceeded {
		t.Errorf("the holder learning the outcome after reporting it unknown: %+v %v", learned, err)
	}
	b.is("the last resolution completes the manifest", "w1", "j1", "succeeded a=succeeded/2/response-lost b=succeeded/1 c=succeeded/1 d=succeeded/1")

	for resolution, want := range map[Resolution]string{
		ResolveSucceeded: "succeeded a=succeeded/1/lost",
		ResolveFailed:    "failed a=failed/1/lost",
	} {
		c := open(t, testLimits)
		c.create("w1", "j1", "send", testPolicy, "a")
		lease := c.one("send")
		c.r.Unknown(lease.Ref(), "lost")
		c.is("a job whose only task is unknown", "w1", "j1", "waiting(outcome_unknown) a=unknown/1/lost/open")
		if report, err := c.r.Resolve("w1", "j1", "a", lease.Token, resolution); err != nil || report.Replay {
			t.Errorf("%s: %+v %v", resolution, report, err)
		}
		c.is("resolved as "+string(resolution), "w1", "j1", want)
		if repeat, err := c.r.Resolve("w1", "j1", "a", lease.Token, resolution); err != nil || !repeat.Replay {
			t.Errorf("%s again: %+v %v", resolution, repeat, err)
		}
	}
	e := open(t, testLimits)
	three := testPolicy
	three.Attempts = 2
	e.create("w1", "j1", "send", three, "a")
	for attempt := 1; attempt <= 2; attempt++ {
		lease := e.one("send")
		e.r.Unknown(lease.Ref(), "lost")
		want := error(nil)
		if attempt == 2 {
			want = ErrRejected
		}
		e.err(fmt.Sprintf("the operator's retry after attempt %d", attempt), second(e.r.Resolve("w1", "j1", "a", lease.Token, ResolveRetry)), want)
	}
	e.is("an operator's retries draw on the same budget", "w1", "j1", "waiting(outcome_unknown) a=unknown/2/lost/open")
	f := open(t, testLimits)
	f.create("w1", "j1", "send", testPolicy, "a*")
	lease := f.one("send")
	f.pass(10 * time.Minute)
	f.r.Sweep()
	report, err = f.r.Unknown(lease.Ref(), "lost")
	if err != nil || report.State != Unknown {
		t.Errorf("an unknown report by the holder of a swept repeatable task: %+v %v", report, err)
	}
	f.pass(time.Hour)
	if got := f.lease("send", 1); len(got) != 0 {
		t.Errorf("leased although its holder said the outcome is unknown: %s", names(got))
	}
}

func TestCancellation(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "close", testPolicy, "a", "b")
	b.err("an unknown reason", b.r.Cancel("w1", "j1", "tired"), ErrInvalid)
	b.err("an unknown job", b.r.Cancel("w1", "j9", Requested), ErrNotFound)
	b.err("another workspace", b.r.Cancel("w2", "j1", Requested), ErrNotFound)
	b.err("a cancellation before any lease", b.r.Cancel("w1", "j1", Superseded), nil)
	b.is("cancelled at once, not queued", "w1", "j1", "cancelled a=cancelled/0 b=cancelled/0")
	revision := b.job("w1", "j1").Revision
	b.err("the same cancellation again", b.r.Cancel("w1", "j1", Superseded), nil)
	b.err("another reason afterwards", b.r.Cancel("w1", "j1", Requested), ErrConflict)
	if view := b.job("w1", "j1"); view.Revision != revision || view.Cancel != Superseded {
		t.Errorf("a repeated cancellation wrote: %+v", view)
	}
	if got := b.lease("close", 4); len(got) != 0 {
		t.Errorf("a cancelled job was leased: %s", names(got))
	}

	b.create("w1", "j2", "close", testPolicy, "a", "b", "c*", "d*", "e")
	leases := b.lease("close", 2)
	a, second2 := leases[0], leases[1]
	b.r.Complete(a.Ref(), "done")
	third := b.one("close")
	b.pass(10 * time.Minute)
	fourth := b.lease("close", 2)
	if names(fourth) != "w1/j2/d w1/j2/e" {
		t.Fatalf("the fixture: %s", names(fourth))
	}
	b.is("before the cancellation", "w1", "j2", "running a=succeeded/1 b=unknown/1/lease_expired/open c=pending/1/open d=leased/1/open e=leased/1/open")
	b.r.Fail(fourth[1].Ref(), "busy", false)
	b.err("a withdrawal of the permission", b.r.Cancel("w1", "j2", PermissionWithdrawn), nil)
	b.is("work stops and nothing is claimed about open attempts", "w1", "j2",
		"waiting(cancel_pending) a=succeeded/1 b=unknown/1/lease_expired/open c=unknown/1/cancelled_while_open/open d=unknown/1/cancelled_while_open/open e=cancelled/1/busy")
	b.pass(time.Hour)
	if got := b.lease("close", 4); len(got) != 0 {
		t.Errorf("leased after a cancellation was requested: %s", names(got))
	}
	b.err("a checkpoint by a holder whose lease the cancellation ended", b.r.Checkpoint(fourth[0].Ref(), "p"), ErrStale)
	b.err("its extension", second(b.r.Extend(fourth[0].Ref())), ErrStale)
	b.err("a retry while a cancellation is pending", second(b.r.Resolve("w1", "j2", "b", second2.Token, ResolveRetry)), ErrRejected)
	report, err := b.r.Complete(third.Ref(), "late")
	if err != nil || report.State != TaskSucceeded {
		t.Errorf("the late completion of a swept repeatable task after the cancellation: %+v %v", report, err)
	}
	report, err = b.r.Fail(fourth[0].Ref(), "stopped", false)
	if err != nil || report.State != TaskCancelled {
		t.Errorf("a retryable failure while a cancellation is pending: %+v %v", report, err)
	}
	b.is("still waiting for the last unknown outcome", "w1", "j2",
		"waiting(cancel_pending) a=succeeded/1 b=unknown/1/lease_expired/open c=succeeded/1/cancelled_while_open d=cancelled/1/stopped e=cancelled/1/busy")
	b.err("the same cancellation while pending", b.r.Cancel("w1", "j2", PermissionWithdrawn), nil)
	b.err("another reason while pending", b.r.Cancel("w1", "j2", Requested), ErrConflict)
	if _, err := b.r.Resolve("w1", "j2", "b", second2.Token, ResolveFailed); err != nil {
		t.Errorf("the resolution: %v", err)
	}
	b.is("cancelled once every outcome is accounted for", "w1", "j2",
		"cancelled a=succeeded/1 b=failed/1/lease_expired c=succeeded/1/cancelled_while_open d=cancelled/1/stopped e=cancelled/1/busy")
	b.err("the same cancellation once cancelled", b.r.Cancel("w1", "j2", PermissionWithdrawn), nil)
	b.err("a report on a task of the cancelled job", second(b.r.Complete(second2.Ref(), "late")), ErrStale)

	c := open(t, testLimits)
	c.create("w1", "j1", "close", testPolicy, "a")
	lease := c.one("close")
	c.err("a cancellation with the only task leased", c.r.Cancel("w1", "j1", Requested), nil)
	c.is("not terminal while the effect may have happened", "w1", "j1", "waiting(cancel_pending) a=unknown/1/cancelled_while_open/open")
	c.r.Complete(lease.Ref(), "done")
	c.is("cancelled with its task succeeded", "w1", "j1", "cancelled a=succeeded/1/cancelled_while_open")

	d := open(t, testLimits)
	d.create("w1", "j1", "close", testPolicy, "again*", "once")
	both := d.lease("close", 2)
	d.pass(10 * time.Minute)
	d.err("a cancellation over leases that expired and were not swept", d.r.Cancel("w1", "j1", Requested), nil)
	d.is("the sweep comes first and neither task is closed", "w1", "j1", "waiting(cancel_pending) again=unknown/1/cancelled_while_open/open once=unknown/1/lease_expired/open")
	d.r.Complete(both[0].Ref(), "late")
	d.r.Resolve("w1", "j1", "once", both[1].Token, ResolveSucceeded)
	d.is("accounted for", "w1", "j1", "cancelled again=succeeded/1/cancelled_while_open once=succeeded/1/lease_expired")
}

func TestSelectionIsFairAndLimited(t *testing.T) {
	b := open(t, Limits{Jobs: 4, Tasks: 6, Attempts: 5, PerType: 3, PerWorkspace: 2, Batch: 8})
	b.create("w1", "old", "close", testPolicy, "a", "b", "c", "d")
	b.create("w2", "new", "close", testPolicy, "p", "q")
	b.create("w1", "later", "close", testPolicy, "x")
	order := []string{}
	for index := 0; index < 7; index++ {
		lease := b.one("close")
		order = append(order, lease.WorkspaceID+"/"+lease.JobID+"/"+lease.TaskID)
		b.r.Complete(lease.Ref(), "done")
	}
	want := "w1/old/a w2/new/p w1/old/b w2/new/q w1/old/c w1/old/d w1/later/x"
	if strings.Join(order, " ") != want {
		t.Errorf("a single worker taking one task at a time:\n got %s\nwant %s", strings.Join(order, " "), want)
	}

	c := open(t, Limits{Jobs: 4, Tasks: 6, Attempts: 5, PerType: 3, PerWorkspace: 2, Batch: 8})
	c.create("w1", "j1", "close", testPolicy, "a", "b", "c")
	c.create("w2", "j1", "close", testPolicy, "p", "q", "r")
	c.create("w3", "j1", "close", testPolicy, "x")
	c.create("w1", "j2", "send", testPolicy, "s", "t", "u")
	got := c.lease("close", 8)
	if names(got) != "w1/j1/a w2/j1/p w3/j1/x" {
		t.Errorf("three per type, fewest leased first: %s", names(got))
	}
	if more := c.lease("close", 8); len(more) != 0 {
		t.Errorf("leased beyond the limit per type: %s", names(more))
	}
	sends := c.lease("send,close", 8)
	if names(sends) != "w1/j2/s w1/j2/t" {
		t.Errorf("two per workspace and type, another type unaffected: %s", names(sends))
	}
	c.r.Complete(got[2].Ref(), "done")
	next := c.lease("close,send", 8)
	if names(next) != "w1/j1/b" {
		t.Errorf("of two pairs with one lease each, the one granted longer ago: %s", names(next))
	}
	c.r.Complete(got[0].Ref(), "done")
	c.r.Complete(next[0].Ref(), "done")
	after := c.lease("close", 8)
	if names(after) != "w1/j1/c w2/j1/q" {
		t.Errorf("the pair with fewer leases first, then the one granted longer ago: %s", names(after))
	}
	for name, attempt := range map[string]func() ([]Lease, error){
		"no worker":             func() ([]Lease, error) { return c.r.Lease("", []string{"close"}, 1) },
		"a worker that is text": func() ([]Lease, error) { return c.r.Lease("worker one", []string{"close"}, 1) },
		"no types":              func() ([]Lease, error) { return c.r.Lease("w", nil, 1) },
		"a type twice":          func() ([]Lease, error) { return c.r.Lease("w", []string{"close", "close"}, 1) },
		"a type that is text":   func() ([]Lease, error) { return c.r.Lease("w", []string{"month close"}, 1) },
		"a limit of zero":       func() ([]Lease, error) { return c.r.Lease("w", []string{"close"}, 0) },
	} {
		c.err(name, second(attempt()), ErrInvalid)
	}
	c.err("more leases than a call may ask for", second(c.r.Lease("w", []string{"close"}, 9)), ErrLimit)
	if none, err := c.r.Lease("w", []string{"archive"}, 1); err != nil || len(none) != 0 {
		t.Errorf("a type without jobs: %v %v", none, err)
	}
}

func TestOneCompletionPerTaskUnderConcurrency(t *testing.T) {
	b := open(t, Limits{Jobs: 10, Tasks: 50, Attempts: 5, PerType: 16, PerWorkspace: 8, Batch: 4})
	for _, workspace := range []string{"w1", "w2", "w3"} {
		tasks := []string{}
		for index := 0; index < 40; index++ {
			tasks = append(tasks, fmt.Sprintf("t%d", index))
		}
		b.create(workspace, "j1", "close", testPolicy, tasks...)
	}
	var group sync.WaitGroup
	var mu sync.Mutex
	done := map[string]int{}
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for idle := 0; idle < 50; {
				leases, err := b.r.Lease(fmt.Sprintf("worker-%d", worker), []string{"close"}, 3)
				if err != nil || len(leases) == 0 {
					idle++
					continue
				}
				for _, lease := range leases {
					first, err := b.r.Complete(lease.Ref(), "done")
					again, err2 := b.r.Complete(lease.Ref(), "done")
					mu.Lock()
					if err == nil && !first.Replay && err2 == nil && again.Replay {
						done[lease.WorkspaceID+"/"+lease.TaskID]++
					} else {
						done["error"]++
					}
					mu.Unlock()
				}
			}
		}()
	}
	group.Wait()
	if len(done) != 120 || done["error"] != 0 {
		t.Errorf("%d tasks completed, %d errors", len(done), done["error"])
	}
	for name, count := range done {
		if count != 1 {
			t.Errorf("%s completed %d times", name, count)
		}
	}
	for _, workspace := range []string{"w1", "w2", "w3"} {
		if view := b.job(workspace, "j1"); view.State != Succeeded || view.Counts[TaskSucceeded] != 40 {
			t.Errorf("%s: %s %+v", workspace, view.State, view.Counts)
		}
	}
}

func TestARegistryThatIsNotBuilt(t *testing.T) {
	for name, limits := range map[string]Limits{"jobs": {0, 1, 1, 1, 1, 1}, "tasks": {1, 0, 1, 1, 1, 1}, "attempts": {1, 1, 0, 1, 1, 1}, "per type": {1, 1, 1, 0, 1, 1},
		"per workspace": {1, 1, 1, 1, 0, 1}, "batch": {1, 1, 1, 1, 1, 0}, "per workspace above per type": {1, 1, 1, 1, 2, 1}} {
		if _, err := NewRegistry(limits, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("limits without %s: %v", name, err)
		}
	}
	if _, err := NewRegistry(testLimits, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("no clock: %v", err)
	}
	var none *Registry
	ref := Ref{WorkspaceID: "w1", JobID: "j1", TaskID: "a", Token: 1}
	for _, r := range []*Registry{none, {}} {
		for name, err := range map[string]error{
			"create": second(r.Create("w1", "j1", "close", testPolicy, specs("a"))), "lease": second(r.Lease("w", []string{"close"}, 1)), "sweep": r.Sweep(),
			"extend": second(r.Extend(ref)), "checkpoint": r.Checkpoint(ref, "p"), "complete": second(r.Complete(ref, "p")), "fail": second(r.Fail(ref, "x", true)),
			"unknown": second(r.Unknown(ref, "x")), "resolve": second(r.Resolve("w1", "j1", "a", 1, ResolveFailed)), "cancel": r.Cancel("w1", "j1", Requested),
			"job": second(r.Job("w1", "j1")), "transitions": second(r.Transitions("w1", "j1")),
		} {
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("%s on a registry that is not built: %v", name, err)
			}
		}
	}
	for state, terminal := range map[JobState]bool{Queued: false, Running: false, Waiting: false, Succeeded: true, Failed: true, Cancelled: true} {
		if state.Terminal() != terminal {
			t.Errorf("%s terminal", state)
		}
	}
	for id, valid := range map[string]bool{"a": true, "A.b_c:d-9": true, strings.Repeat("x", 128): true, strings.Repeat("x", 129): false, "": false, "a b": false} {
		if validID(id) != valid {
			t.Errorf("identifier %q", id)
		}
	}
}

func TestTheRecordOfAHistory(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "send", testPolicy, "a", "b*", "c")
	leases := b.lease("send", 2)
	b.r.Unknown(leases[0].Ref(), "lost")
	b.pass(10 * time.Minute)
	b.r.Sweep()
	b.r.Resolve("w1", "j1", "a", leases[0].Token, ResolveRetry)
	b.r.Cancel("w1", "j1", Requested)
	b.r.Complete(leases[1].Ref(), "late")
	log, _ := b.r.Transitions("w1", "j1")
	got := []string{}
	for _, entry := range log {
		got = append(got, fmt.Sprintf("%s:%s:%s:%s", entry.Event, entry.Task, entry.Worker, entry.State))
	}
	want := "created:::queued granted:a:worker-1:running granted:b:worker-1:running unknown:a:worker-1:running expired:b:worker-1:running resolved:a::running " +
		"cancel_requested:::waiting completed:b:worker-1:cancelled"
	if strings.Join(got, " ") != want {
		t.Errorf("the record:\n got %s\nwant %s", strings.Join(got, " "), want)
	}
	b.is("the end of that history", "w1", "j1", "cancelled a=cancelled/1/lost b=succeeded/1/cancelled_while_open c=cancelled/0")
}

func TestATokenOfAnotherWorkspaceNeverMatches(t *testing.T) {
	b := open(t, Limits{Jobs: 4, Tasks: 6, Attempts: 5, PerType: 4, PerWorkspace: 4, Batch: 8})
	b.create("w1", "j1", "close", testPolicy, "a")
	b.create("w2", "j1", "close", testPolicy, "a")
	leases := b.lease("close", 2)
	if names(leases) != "w1/j1/a w2/j1/a" {
		t.Fatalf("the fixture: %s", names(leases))
	}
	crossed := Ref{WorkspaceID: "w1", JobID: "j1", TaskID: "a", Token: leases[1].Token}
	b.err("a completion with the token of the same job and task in another workspace", second(b.r.Complete(crossed, "done")), ErrStale)
	b.err("a failure with it", second(b.r.Fail(crossed, "x", true)), ErrStale)
	b.err("an unknown report with it", second(b.r.Unknown(crossed, "x")), ErrStale)
	b.err("an extension with it", second(b.r.Extend(crossed)), ErrStale)
	b.err("a checkpoint with it", b.r.Checkpoint(crossed, "p"), ErrStale)
	b.err("a resolution with it", second(b.r.Resolve("w1", "j1", "a", leases[1].Token, ResolveSucceeded)), ErrStale)
	b.is("the first workspace", "w1", "j1", "running a=leased/1/open")
	b.is("the second workspace", "w2", "j1", "running a=leased/1/open")
	both := b.lease("close", 8)
	if len(both) != 0 {
		t.Errorf("leased again: %s", names(both))
	}
	b.create("w1", "j2", "close", testPolicy, "b", "c", "d")
	if more := b.lease("close", 8); names(more) != "w1/j2/b w1/j2/c" {
		t.Errorf("one workspace may hold every slot when the two limits are equal, and no more: %s", names(more))
	}
}

func TestReportsAgainAfterTheTaskMovedOn(t *testing.T) {
	b := open(t, testLimits)
	b.create("w1", "j1", "close", testPolicy, "a")
	lease := b.one("close")
	b.r.Fail(lease.Ref(), "busy", false)
	b.err("a cancellation", b.r.Cancel("w1", "j1", Requested), nil)
	again, err := b.r.Fail(lease.Ref(), "busy", false)
	if err != nil || again != (Report{State: TaskCancelled, Replay: true}) {
		t.Errorf("a repeated report tells the state the task has now: %+v %v", again, err)
	}
	one := testPolicy
	one.Attempts = 1
	b.create("w1", "j2", "close", one, "a")
	last := b.one("close")
	b.err("a cancellation with the last attempt leased", b.r.Cancel("w1", "j2", Requested), nil)
	report, err := b.r.Fail(last.Ref(), "busy", false)
	if err != nil || report.State != TaskFailed {
		t.Errorf("a retryable failure at the end of the budget while a cancellation is pending: %+v %v", report, err)
	}
	b.is("accounted for", "w1", "j2", "cancelled a=failed/1/busy")
}

func TestTheStateRule(t *testing.T) {
	cases := []struct {
		name            string
		counts          map[TaskState]int
		cancel, granted bool
		state           JobState
		reason          WaitReason
	}{
		{"nothing leased yet", map[TaskState]int{Pending: 3}, false, false, Queued, ""},
		{"work under way", map[TaskState]int{Pending: 1, Leased: 1, TaskSucceeded: 1}, false, true, Running, ""},
		{"only retries left", map[TaskState]int{Pending: 1, TaskSucceeded: 2}, false, true, Running, ""},
		{"an unknown outcome beside pending work", map[TaskState]int{Pending: 1, Unknown: 1}, false, true, Running, ""},
		{"an unknown outcome beside leased work", map[TaskState]int{Leased: 1, Unknown: 1}, false, true, Running, ""},
		{"only unknown outcomes left", map[TaskState]int{Unknown: 1, TaskSucceeded: 2}, false, true, Waiting, OutcomeUnknown},
		{"unknown beside failed", map[TaskState]int{Unknown: 1, TaskFailed: 2}, false, true, Waiting, OutcomeUnknown},
		{"everything succeeded", map[TaskState]int{TaskSucceeded: 3}, false, true, Succeeded, ""},
		{"a partial result", map[TaskState]int{TaskSucceeded: 2, TaskFailed: 1}, false, true, Failed, ""},
		{"everything failed", map[TaskState]int{TaskFailed: 3}, false, true, Failed, ""},
		{"cancelled before any lease", map[TaskState]int{TaskCancelled: 3}, true, false, Cancelled, ""},
		{"cancelled with leased work", map[TaskState]int{Leased: 1, TaskCancelled: 2}, true, true, Waiting, CancelPending},
		{"cancelled with unknown outcomes", map[TaskState]int{Unknown: 1, TaskCancelled: 2}, true, true, Waiting, CancelPending},
		{"cancelled and accounted for", map[TaskState]int{TaskSucceeded: 1, TaskFailed: 1, TaskCancelled: 1}, true, true, Cancelled, ""},
		{"cancelled although everything succeeded", map[TaskState]int{TaskSucceeded: 3}, true, true, Cancelled, ""},
	}
	for _, c := range cases {
		state, reason := derive(c.counts, c.cancel, c.granted)
		if state != c.state || reason != c.reason {
			t.Errorf("%s: %s %s; want %s %s", c.name, state, reason, c.state, c.reason)
		}
	}
}
