package approvals

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func currentFixture(t *testing.T) (*Registry, *testClock, Execution) {
	t.Helper()
	r, clock := setup(t, 1)
	must(r.Request("ws", subject("current", "invoice", "100"), human("requester")))
	must(r.Decide("ws", "current", approve, human("ann")))
	return r, clock, runOf("candidate", "current", "invoice", "100")
}

func TestCheckCurrentDoesNotConsume(t *testing.T) {
	r, _, expected := currentFixture(t)
	before := must(r.RequestOf("ws", "current"))
	for i := 0; i < 100; i++ {
		expect(t, "ok", r.CheckCurrent("ws", expected, human("ops")), "repeated check")
	}
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := r.CheckCurrent("ws", expected, human("ops")); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	after := must(r.RequestOf("ws", "current"))
	if !reflect.DeepEqual(before, after) || len(r.workspaces["ws"].executions) != 0 || len(r.workspaces["ws"].keys) != 0 {
		t.Fatal("checks changed domain records")
	}
	expect(t, "not_found", second(r.ExecutionOf("ws", expected.ID)), "no execution record")
	made := must(r.Execute("ws", expected, human("ops")))
	if made.Execution.ID != expected.ID {
		t.Fatal("actual execution changed identity")
	}
	expect(t, "ok", r.CheckCurrent("ws", expected, Actor{ID: "assistant", Kind: KindAssistant}), "valid nonhuman check")
	expected.ID = "another"
	expect(t, "ok", r.CheckCurrent("ws", expected, Actor{ID: "system", Kind: KindSystem}), "check consumed key")
	expect(t, ReasonAlreadyExecuted, second(r.Execute("ws", expected, human("ops"))), "check did not reopen consumed key")
}

func TestCheckCurrentDoesNotReplayHistoricalSuccess(t *testing.T) {
	r, _, expected := currentFixture(t)
	must(r.Execute("ws", expected, human("ops")))
	must(r.Report("ws", expected.ID, Outcome{Result: ResultSucceeded}, human("ops")))
	must(r.Revoke("ws", "g-ann", human("adm1")))
	expect(t, ReasonAuthorityLost, r.CheckCurrent("ws", expected, human("ops")), "current authority after historical success")
	expect(t, "ok", second(r.Execute("ws", expected, human("ops"))), "historical replay unchanged")
}

func TestCheckCurrentIDIsNotAReservation(t *testing.T) {
	r, _, expected := currentFixture(t)
	must(r.Execute("ws", expected, human("ops")))
	must(r.Request("ws", subject("other-request", "other-target", "200"), human("requester")))
	must(r.Decide("ws", "other-request", approve, human("ann")))
	changed := runOf(expected.ID, "other-request", "other-target", "200")
	expect(t, "ok", r.CheckCurrent("ws", changed, human("ops")), "used identifier shape only")
	expect(t, "conflict", second(r.Execute("ws", changed, human("ops"))), "actual identifier conflict retained")
}

func TestCheckCurrentValidityAndClockRollback(t *testing.T) {
	r, clock, expected := currentFixture(t)
	clock.advance(24 * time.Hour)
	expect(t, "ok", r.CheckCurrent("ws", expected, human("ops")), "exact validity boundary")
	clock.advance(time.Nanosecond)
	expect(t, ReasonExpired, r.CheckCurrent("ws", expected, human("ops")), "after validity")
	clock.advance(-time.Hour)
	expect(t, ReasonExpired, r.CheckCurrent("ws", expected, human("ops")), "clock rollback cannot revive approval")
}

func TestCheckCurrentExpectedDimensions(t *testing.T) {
	for name, change := range map[string]func(*Execution){
		"stage":          func(e *Execution) { e.Stage = StageDeliver },
		"target":         func(e *Execution) { e.Target = "other" },
		"currency":       func(e *Execution) { e.Currency = "EUR" },
		"amount":         func(e *Execution) { e.Amount = "101" },
		"digest":         func(e *Execution) { e.Digests = map[string]string{"result": digest('c'), "erp_draft": digest('b')} },
		"missing digest": func(e *Execution) { e.Digests = map[string]string{"result": digest('a')} },
		"item":           func(e *Execution) { e.Item = "line"; e.ItemDigest = digest('a') },
	} {
		t.Run(name, func(t *testing.T) {
			r, _, expected := currentFixture(t)
			change(&expected)
			expect(t, ReasonSubjectChanged, r.CheckCurrent("ws", expected, human("ops")), "changed expected subject")
		})
	}
	r, _, expected := currentFixture(t)
	expected.Amount = "100.00000000"
	expect(t, "ok", r.CheckCurrent("ws", expected, human("ops")), "exact decimal normalization")
}

func TestCheckCurrentStatesAndIsolation(t *testing.T) {
	for name, change := range map[string]func(*Registry){
		ReasonWithdrawn:  func(r *Registry) { must(r.Withdraw("ws", "current", human("requester"))) },
		ReasonSuperseded: func(r *Registry) { must(r.Request("ws", subject("replacement", "invoice", "100"), human("requester"))) },
	} {
		t.Run(name, func(t *testing.T) {
			r, _, expected := currentFixture(t)
			change(r)
			expect(t, name, r.CheckCurrent("ws", expected, human("ops")), "ended approval")
		})
	}
	r, _, expected := currentFixture(t)
	administer(t, r, policy(StageIssue, 2, 1, "result", "erp_draft"))
	expect(t, ReasonSuperseded, r.CheckCurrent("ws", expected, human("ops")), "changed policy supersedes public requests")
	must(r.Request("ws", subject("pending", "pending-target", "10"), human("requester")))
	pending := runOf("pending-check", "pending", "pending-target", "10")
	expect(t, ReasonNotApproved, r.CheckCurrent("ws", pending, human("ops")), "pending")
	must(r.Decide("ws", "pending", Decision{Verdict: VerdictReject, Reason: "other"}, human("bob")))
	expect(t, ReasonNotApproved, r.CheckCurrent("ws", pending, human("ops")), "rejected")
	expect(t, "not_found", r.CheckCurrent("foreign", expected, human("ops")), "foreign workspace")
	expected.Request = "absent"
	expect(t, "not_found", r.CheckCurrent("ws", expected, human("ops")), "unknown request")
	expect(t, "invalid", r.CheckCurrent("", expected, human("ops")), "invalid workspace")
	expect(t, "invalid", r.CheckCurrent("ws", expected, Actor{ID: "ops", Kind: "invalid"}), "invalid actor")
	expected.ID = ""
	expect(t, "invalid", r.CheckCurrent("ws", expected, human("ops")), "invalid expected form")
}

func TestCheckCurrentBulkExclusion(t *testing.T) {
	r, _ := setup(t, 1)
	sub := subject("bulk", "batch", "200")
	sub.Entries = []Entry{{Item: "one", Scope: "customer-one", Amount: "100", Digest: digest('c')}, {Item: "two", Scope: "customer-two", Amount: "100", Digest: digest('d')}}
	must(r.Request("ws", sub, human("requester")))
	must(r.Decide("ws", "bulk", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "one", Reason: "data_missing"}}}, human("ann")))
	expected := runOf("bulk-check", "bulk", "batch", "100")
	expected.Item = "one"
	expected.ItemDigest = digest('c')
	expect(t, ReasonItemIneligible, r.CheckCurrent("ws", expected, human("ops")), "excluded item")
	expected.Item = "two"
	expected.ItemDigest = digest('d')
	expect(t, "ok", r.CheckCurrent("ws", expected, human("ops")), "included item")
	if len(r.workspaces["ws"].executions) != 0 {
		t.Fatal("item check consumed execution")
	}
}

func TestCheckCurrentAuthorityDateBoundary(t *testing.T) {
	r, clock, expected := currentFixture(t)
	must(r.Revoke("ws", "g-ann", human("adm1")))
	dated := grant("dated-ann", "ann", StageIssue, "1000")
	dated.Authority.From = start.Format(time.DateOnly)
	dated.Authority.To = dated.Authority.From
	administer(t, r, dated)
	clock.advance(15*time.Hour - time.Nanosecond)
	expect(t, "ok", r.CheckCurrent("ws", expected, human("ops")), "last instant of authority date")
	clock.advance(time.Nanosecond)
	expect(t, ReasonAuthorityLost, r.CheckCurrent("ws", expected, human("ops")), "next UTC date")
}

func TestCheckCurrentConcurrentRevocation(t *testing.T) {
	r, _, expected := currentFixture(t)
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			err := r.CheckCurrent("ws", expected, human("ops"))
			if err != nil && kind(err) != ReasonAuthorityLost {
				t.Error(err)
			}
		}()
	}
	must(r.Revoke("ws", "g-ann", human("adm1")))
	group.Wait()
	expect(t, ReasonAuthorityLost, r.CheckCurrent("ws", expected, human("ops")), "after completed revocation")
	if len(r.workspaces["ws"].executions) != 0 || len(r.workspaces["ws"].keys) != 0 {
		t.Fatal("concurrent checks wrote execution state")
	}
}
