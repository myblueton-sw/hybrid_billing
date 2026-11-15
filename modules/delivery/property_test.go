package delivery

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/disclosure"
)

type mAttempt struct {
	token    string
	started  time.Time
	reported bool
}

type mDelivery struct {
	id       string
	pair     int
	state    string
	approved bool
	resend   bool
	running  int
	attempts []*mAttempt
}

type model struct {
	deliveries map[string]*mDelivery
	order      []string
	current    [2]string
	uncertain  [2]bool
	open       [2]bool
	resent     map[string]int
}

var pairs = [2]Delivery{{Package: "p1", Channel: "mail-acme"}, {Package: "p2", Channel: "portal-acme"}}

func (m *model) queued() int {
	n := 0
	for _, d := range m.deliveries {
		if d.state == StateWaiting || d.state == StateInFlight {
			n++
		}
	}
	return n
}

func (m *model) mark(d *mDelivery) { m.uncertain[d.pair] = true }

func TestDeliveryMatchesAModelUnderRandomOperations(t *testing.T) {
	lease := testLimits().Lease
	stats := map[string]int{}
	for seed := int64(1); seed <= 60; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := standard(t)
		m := &model{deliveries: map[string]*mDelivery{}, open: [2]bool{true, false}, resent: map[string]int{}}
		next := 0
		pick := func() *mDelivery {
			if len(m.order) == 0 {
				return nil
			}
			return m.deliveries[m.order[rng.Intn(len(m.order))]]
		}
		for step := 0; step < 300; step++ {
			now := f.clock.read()
			switch rng.Intn(10) {
			case 0:
				pair := rng.Intn(2)
				next++
				id := "d" + strconv.Itoa(next)
				q := pairs[pair]
				q.ID = id
				v, err := f.r.Request("ws", q, human("requester"))
				switch {
				case m.current[pair] != "":
					if err != nil || !v.Duplicate || v.Delivery.ID != m.current[pair] {
						t.Fatalf("seed %d: expected duplicate of %s: %+v %v", seed, m.current[pair], v, err)
					}
					stats["duplicate"]++
				case len(m.deliveries) >= testLimits().Deliveries || m.queued() >= testLimits().Queue:
					expect(t, "limit", err, "limit")
				default:
					if err != nil || v.PossibleDuplicate != m.uncertain[pair] {
						t.Fatalf("seed %d: request %+v %v model %v", seed, v, err, m.uncertain[pair])
					}
					m.deliveries[id] = &mDelivery{id: id, pair: pair, state: StateWaiting}
					m.order = append(m.order, id)
					m.current[pair] = id
				}
			case 1, 2:
				d := pick()
				if d == nil {
					continue
				}
				if d.state == StateInFlight && !now.Before(d.attempts[d.running-1].started.Add(lease)) {
					m.mark(d)
					d.running, d.state = 0, StateWaiting
					stats["superseded"]++
				}
				s, err := f.r.Begin("ws", d.id, system("worker"))
				switch {
				case d.state == StateInFlight:
					expect(t, "running", err, ReasonAttemptRunning)
				case d.state != StateWaiting:
					expect(t, "not waiting", err, ReasonNotWaiting)
				case !d.approved && (!m.open[d.pair] || d.resend):
					expect(t, "approval", err, ReasonNotApproved)
				case len(d.attempts) >= testLimits().Attempts:
					expect(t, "exhausted", err, "limit")
					d.state = StateExhausted
				default:
					if err != nil || s.PossibleDuplicate != (m.uncertain[d.pair] || d.resend) || s.Number != len(d.attempts)+1 {
						t.Fatalf("seed %d: begin %+v %v model %v", seed, s, err, m.uncertain[d.pair])
					}
					d.attempts = append(d.attempts, &mAttempt{token: s.Token, started: now})
					d.running, d.state = len(d.attempts), StateInFlight
					stats["begun"]++
				}
			case 3, 4:
				d := pick()
				if d == nil || len(d.attempts) == 0 {
					continue
				}
				index := rng.Intn(len(d.attempts))
				a := d.attempts[index]
				result := []string{ResultAccepted, ResultRejected, ResultUnknown}[rng.Intn(3)]
				_, err := f.r.Report("ws", d.id, a.token, Outcome{Result: result, Reference: "r"}, system("worker"))
				if a.reported {
					if err != nil && reason(err) != "conflict" {
						t.Fatalf("seed %d: repeat report %v", seed, err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("seed %d: report %v", seed, err)
				}
				a.reported = true
				if d.running == index+1 {
					d.running, d.state = 0, StateWaiting
				} else {
					stats["late"]++
				}
				switch result {
				case ResultUnknown:
					m.mark(d)
				case ResultAccepted:
					if index+1 < len(d.attempts) || m.current[d.pair] != "" && m.current[d.pair] != d.id {
						m.mark(d)
						stats["cross"]++
					}
					if m.current[d.pair] == "" {
						m.current[d.pair] = d.id
					}
					if d.state != StateConfirmed {
						d.state, d.running = StateAccepted, 0
					}
				}
			case 5:
				d := pick()
				if d == nil {
					continue
				}
				_, err := f.r.Cancel("ws", d.id, human("ops"))
				switch d.state {
				case StateAccepted, StateConfirmed:
					expect(t, "final", err, ReasonFinal)
				case StateCancelled:
					expect(t, "repeat", err, "ok")
				default:
					expect(t, "cancel", err, "ok")
					if d.running > 0 {
						m.mark(d)
					}
					d.state, d.running = StateCancelled, 0
					if m.current[d.pair] == d.id {
						m.current[d.pair] = ""
					}
				}
			case 6:
				d := pick()
				if d == nil {
					continue
				}
				_, err := f.r.Approve("ws", d.id, human("lead"))
				if !d.approved && d.state == StateWaiting {
					expect(t, "approve", err, "ok")
					d.approved = true
				}
			case 7:
				d := pick()
				if d == nil {
					continue
				}
				next++
				id := "d" + strconv.Itoa(next)
				_, err := f.r.Resend("ws", d.id, Delivery{ID: id}, "again", human("ops"))
				final := d.state == StateAccepted || d.state == StateConfirmed || d.state == StateCancelled || d.state == StateStopped || d.state == StateExhausted
				switch {
				case !final || m.current[d.pair] != "" && m.current[d.pair] != d.id:
					expect(t, "not resendable", err, ReasonNotResendable)
				case len(m.deliveries) >= testLimits().Deliveries || m.queued() >= testLimits().Queue:
					expect(t, "limit", err, "limit")
				default:
					expect(t, "resend", err, "ok")
					m.deliveries[id] = &mDelivery{id: id, pair: d.pair, state: StateWaiting, resend: true}
					m.order = append(m.order, id)
					m.current[d.pair] = id
					stats["resend"]++
				}
			case 8:
				f.clock.advance(time.Duration(rng.Intn(25)) * time.Minute)
			case 9:
				d := pick()
				if d == nil || d.state != StateAccepted {
					continue
				}
				must(f.r.Confirm("ws", d.id, "receipt", system("worker")))
				d.state = StateConfirmed
			}
			for _, id := range m.order {
				want := m.deliveries[id]
				got := must(f.r.Of("ws", id))
				if got.State != want.state || len(got.Attempts) != len(want.attempts) || (got.Approver != "") != want.approved {
					t.Fatalf("seed %d step %d: %s got %s/%d model %s/%d", seed, step, id, got.State, len(got.Attempts), want.state, len(want.attempts))
				}
			}
		}
	}
	for _, key := range []string{"duplicate", "superseded", "begun", "late", "cross", "resend"} {
		if stats[key] < 5 {
			t.Fatalf("weak exploration: %v", stats)
		}
	}
}

func TestRealDisclosurePackages(t *testing.T) {
	clock := &testClock{now: start}
	d, err := disclosure.NewRegistry(disclosure.Config{Keys: func(string) ([]byte, error) { return []byte("key"), nil }, Random: &counter{}},
		disclosure.Limits{Workspaces: 1, Fields: 4, Policies: 2, Recipients: 2, Projections: 4, Packages: 4, Links: 4, Rows: 4, Combinations: 1}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	staff := func(id string) disclosure.Actor { return disclosure.Actor{ID: id, Kind: disclosure.KindHuman} }
	must(d.Classify("ws", disclosure.Field{Name: "customer", Class: disclosure.ClassCustomer}, staff("steward")))
	must(d.Classify("ws", disclosure.Field{Name: "amount", Class: disclosure.ClassCustomer}, staff("steward")))
	p := must(d.Propose("ws", disclosure.Policy{ID: "std", Roles: map[string][]string{disclosure.RoleCustomer: {"customer", "amount"}}}, staff("author")))
	must(d.Confirm("ws", "std", p.Revision, staff("approver")))
	g := must(d.Grant("ws", disclosure.Recipient{ID: "acme", Role: disclosure.RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, staff("author")))
	must(d.ConfirmGrant("ws", "acme", g.Revision, staff("approver")))
	proj := must(d.Project("ws", disclosure.Request{Policy: "std", Recipient: "acme", Rows: []disclosure.Row{{"customer": "acme", "amount": "10"}}}, disclosure.Actor{ID: "job", Kind: disclosure.KindSystem}))
	pk := must(d.Package("ws", proj.Digest, disclosure.Actor{ID: "job", Kind: disclosure.KindSystem}))
	r, err := NewRegistry(Config{Packages: d, Random: &counter{}}, testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	must(r.Register("ws", Channel{ID: "portal", Kind: ChannelPortal, Recipient: "acme", Target: "p"}, human("ops")))
	expect(t, "recipient actor from disclosure", second(r.Register("ws", Channel{ID: "x", Kind: ChannelPortal, Recipient: "acme", Target: "p"}, human("acme"))), ReasonRecipientActor)
	v := must(r.Request("ws", Delivery{ID: "d1", Package: pk.ID, Channel: "portal"}, system("scheduler")))
	if v.FileDigest != pk.FileDigest || v.Recipient != "acme" || v.Policy != "std" {
		t.Fatalf("bound: %+v", v)
	}
	must(r.Approve("ws", "d1", human("lead")))
	a := must(r.Begin("ws", "d1", system("worker")))
	must(r.Report("ws", "d1", a.Token, Outcome{Result: ResultRejected, Reference: "busy"}, system("worker")))
	p2 := must(d.Propose("ws", disclosure.Policy{ID: "std", Roles: map[string][]string{disclosure.RoleCustomer: {"customer"}}}, staff("author")))
	must(d.Confirm("ws", "std", p2.Revision, staff("approver")))
	expect(t, "superseded policy", second(r.Begin("ws", "d1", system("worker"))), ReasonPackageNotLive)
	if got := must(r.Of("ws", "d1")); got.State != StateStopped || got.Reason != disclosure.ReasonPolicySuperseded {
		t.Fatalf("stopped: %+v", got)
	}
	proj = must(d.Project("ws", disclosure.Request{Policy: "std", Recipient: "acme", Rows: []disclosure.Row{{"customer": "acme", "amount": "11"}}}, disclosure.Actor{ID: "job", Kind: disclosure.KindSystem}))
	pk = must(d.Package("ws", proj.Digest, disclosure.Actor{ID: "job", Kind: disclosure.KindSystem}))
	must(r.Request("ws", Delivery{ID: "d2", Package: pk.ID, Channel: "portal"}, system("scheduler")))
	must(r.Approve("ws", "d2", human("lead")))
	must(d.RevokeRecipient("ws", "acme", staff("ops")))
	expect(t, "revoked grant", second(r.Begin("ws", "d2", system("worker"))), ReasonPackageNotLive)
	if got := must(r.Of("ws", "d2")); got.Reason != disclosure.ReasonRecipientRevoked {
		t.Fatalf("revoked: %+v", got)
	}
}
