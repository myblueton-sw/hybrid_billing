package protected

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

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

const day = 24 * time.Hour

func fingerprint(c byte) string { return strings.Repeat(string(c), 64) }

func ref(id string, c byte) Ref { return Ref{ID: id, Fingerprint: fingerprint(c)} }

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func testLimits() Limits { return Limits{Workspaces: 3, Channels: 6, Changes: 6} }

func testRules() Rules {
	return Rules{Approvals: 1, Validity: 2 * day, MaxLead: 30 * day, ChannelAge: day}
}

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

func expect(t *testing.T, label string, err error, want string) {
	t.Helper()
	if got := kind(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type fixture struct {
	r     *Registry
	clock *testClock
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{r: r, clock: clock}
}

func ready(t *testing.T) fixture {
	t.Helper()
	f := newFixture(t)
	f.rules(t, "w1", testRules())
	f.channel(t, "w1", Channel{ID: "ch-phone", Subject: "acme", Kind: ChannelPhone, Ref: ref("vault:phone", 'a'), Evidence: "contract:1"})
	f.channel(t, "w1", Channel{ID: "ch-mail", Subject: "acme", Kind: ChannelEmail, Ref: ref("vault:mail", 'b'), Evidence: "contract:1"})
	f.clock.advance(2 * day)
	return f
}

func (f fixture) rules(t *testing.T, workspace string, rules Rules) RulesView {
	t.Helper()
	proposed := must(f.r.Propose(workspace, rules, human("admin")))
	return must(f.r.Confirm(workspace, proposed.Revision, human("admin2")))
}

func (f fixture) channel(t *testing.T, workspace string, c Channel) ChannelView {
	t.Helper()
	must(f.r.Register(workspace, c, human("onboard")))
	return must(f.r.Admit(workspace, c.ID, "admission:"+c.ID, human("onboard2")))
}

func account(id string, after Ref, effective time.Time) Change {
	return Change{
		ID: id, Subject: "acme", Field: FieldPaymentAccount, After: after, Provenance: ProvenanceDocument,
		Instruction: Instruction{Kind: ChannelEmail, Channel: "ch-mail", Sender: fingerprint('b'), Evidence: "evidence:" + id},
		Effective:   effective,
	}
}

func confirm(id, channel, digest string) Verification {
	return Verification{Channel: channel, Evidence: "call:" + id, Outcome: OutcomeConfirmed, Digest: digest}
}

func approve(digest string) Approval { return Approval{Verdict: VerdictApprove, Digest: digest} }

func (f fixture) schedule(t *testing.T, c Change) View {
	t.Helper()
	v := must(f.r.Request("w1", c, human("clerk")))
	v = must(f.r.Verify("w1", c.ID, confirm(c.ID, "ch-phone", v.Digest), human("checker")))
	return must(f.r.Approve("w1", c.ID, approve(v.Verification.Digest), human("boss")))
}
