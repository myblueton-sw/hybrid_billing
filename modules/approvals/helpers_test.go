package approvals

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "approvals-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Admins: 4, Proposals: 20, Authorities: 8, Requests: 10, Items: 4, Executions: 10}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

func digest(c byte) string { return strings.Repeat(string(c), 64) }

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func kind(err error) string {
	var refusal Refusal
	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "other: " + err.Error()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func second[T any](_ T, err error) error { return err }

func expect(t testing.TB, want string, err error, what string) {
	t.Helper()
	if kind(err) != want {
		t.Errorf("%s: want %s, got %s", what, want, kind(err))
	}
}

var proposals int

func administer(t testing.TB, r *Registry, p Proposal) {
	t.Helper()
	proposals++
	p.ID = "p" + strconv.Itoa(proposals)
	must(r.Propose("ws", p, human("adm1")))
	must(r.Confirm("ws", p.ID, human("adm2")))
}

func policy(stage string, revision, approvals int, required ...string) Proposal {
	return Proposal{Kind: ProposePolicy, Policy: Policy{Stage: stage, Revision: revision, Approvals: approvals, Required: required, Validity: 24 * time.Hour}}
}

func grant(id, approver, stage, limit string) Proposal {
	return Proposal{Kind: ProposeGrant, Authority: Authority{ID: id, Approver: approver, Stage: stage, Currency: "USD", Limit: limit, From: "2026-01-01"}}
}

func setup(t testing.TB, approvals int) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	must(r.Open("ws", []string{"adm1", "adm2"}, human("installer")))
	administer(t, r, policy(StageIssue, 1, approvals, "result", "erp_draft"))
	administer(t, r, grant("g-ann", "ann", StageIssue, "1000"))
	administer(t, r, grant("g-bob", "bob", StageIssue, "500"))
	return r, clock
}

func digests() map[string]string {
	return map[string]string{"result": digest('a'), "erp_draft": digest('b')}
}

func subject(id, target, amount string) Subject {
	return Subject{ID: id, Stage: StageIssue, Target: target, Author: "writer", Digests: digests(), Amount: amount, Currency: "USD"}
}

func runOf(id, request, target, amount string) Execution {
	return Execution{ID: id, Request: request, Stage: StageIssue, Target: target, Digests: digests(), Amount: amount, Currency: "USD"}
}

var approve = Decision{Verdict: VerdictApprove}
