package protected

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type modelChange struct {
	id, subject, field, requester, verifier, channel string
	after, before                                    Ref
	effective, confirmedAt, ended                    time.Time
	state, endedBy                                   string
	scheduled                                        time.Time
	approvers                                        map[string]bool
	digest, confirmation                             string
	viaChannel                                       string
}

type model struct {
	now      time.Time
	changes  map[string]*modelChange
	order    []*modelChange
	approval int
	validity time.Duration
	lead     time.Duration
	retired  map[string]bool
}

var modelChannels = map[string]struct {
	subject string
	print   string
}{
	"ch-phone": {"acme", fingerprint('a')},
	"ch-mail":  {"acme", fingerprint('b')},
	"ch-beta":  {"beta", fingerprint('c')},
}

var modelCustodians = map[string]bool{"onboard": true, "onboard2": true}

func (m *model) lapse(c *modelChange) {
	if c.state != StateRequested && c.state != StateConfirmed {
		return
	}
	end := c.effective
	if c.state == StateConfirmed && c.confirmedAt.Add(m.validity).Before(end) {
		end = c.confirmedAt.Add(m.validity)
	}
	if m.now.After(end) {
		c.state, c.ended = StateLapsed, end
	}
}

func (m *model) lapseAll() {
	for _, c := range m.order {
		m.lapse(c)
	}
}

func (m *model) scheduledLast(subject, field string) *modelChange {
	var last *modelChange
	for _, c := range m.order {
		if c.subject == subject && c.field == field && c.state == StateScheduled {
			last = c
		}
	}
	return last
}

func (m *model) request(id, subject, field string, after Ref, effective time.Time, provenance, via string, actor Actor) string {
	if actor.Kind == KindAssistant {
		return ReasonAINotAllowed
	}
	if actor.Kind == KindSystem {
		return ReasonNotHuman
	}
	if provenance == ProvenanceAssistant {
		return ReasonLLMGenerated
	}
	m.lapseAll()
	for _, c := range m.order {
		if c.subject == subject && c.field == field && (c.state == StateRequested || c.state == StateConfirmed) {
			return ReasonChangePending
		}
	}
	if !effective.After(m.now) || effective.After(m.now.Add(m.lead)) {
		return ReasonEffectiveOutOfRange
	}
	var before Ref
	if last := m.scheduledLast(subject, field); last != nil {
		if !effective.After(last.effective) {
			return ReasonOutOfOrder
		}
		before = last.after
	}
	if before == after {
		return ReasonNoChange
	}
	c := &modelChange{id: id, subject: subject, field: field, requester: actor.ID, after: after, before: before, effective: effective, state: StateRequested, approvers: map[string]bool{}, viaChannel: via}
	m.changes[id] = c
	m.order = append(m.order, c)
	return "ok"
}

func (m *model) verify(c *modelChange, channel string, actor Actor, sender string, denied, badDigest bool) string {
	if actor.Kind == KindAssistant {
		return ReasonAINotAllowed
	}
	if actor.Kind == KindSystem {
		return ReasonNotHuman
	}
	m.lapse(c)
	if c.state == StateLapsed {
		return ReasonLapsed
	}
	if c.state != StateRequested {
		return ReasonNotPending
	}
	if badDigest {
		return ReasonDigestChanged
	}
	if actor.ID == c.requester {
		return ReasonSelfVerification
	}
	info, ok := modelChannels[channel]
	if !ok || info.subject != c.subject {
		return ReasonUnknownChannel
	}
	if channel == c.viaChannel || info.print == sender {
		return ReasonSameChannel
	}
	if m.retired[channel] {
		return ReasonChannelRetired
	}
	if modelCustodians[c.requester] || modelCustodians[actor.ID] {
		return ReasonSelfChannel
	}
	c.state, c.verifier, c.channel, c.confirmedAt = StateConfirmed, actor.ID, channel, m.now
	if denied {
		c.state, c.ended, c.endedBy = StateRejected, m.now, actor.ID
	}
	return "ok"
}

func (m *model) approve(c *modelChange, actor Actor, reject, badDigest bool) string {
	if actor.Kind == KindAssistant {
		return ReasonAINotAllowed
	}
	if actor.Kind == KindSystem {
		return ReasonNotHuman
	}
	m.lapse(c)
	switch c.state {
	case StateLapsed:
		return ReasonLapsed
	case StateRequested:
		return ReasonNotConfirmed
	case StateConfirmed:
	default:
		return ReasonNotPending
	}
	if badDigest {
		return ReasonDigestChanged
	}
	if m.retired[c.channel] {
		return ReasonChannelRetired
	}
	if actor.ID == c.requester || actor.ID == c.verifier || modelCustodians[actor.ID] {
		return ReasonSelfApproval
	}
	if c.approvers[actor.ID] {
		return ReasonAlreadyDecided
	}
	if reject {
		c.state, c.ended, c.endedBy = StateRejected, m.now, actor.ID
		return "ok"
	}
	c.approvers[actor.ID] = true
	if len(c.approvers) >= m.approval {
		c.state, c.scheduled = StateScheduled, m.now
	}
	return "ok"
}

func (m *model) withdraw(c *modelChange, actor Actor) string {
	if actor.Kind == KindAssistant {
		return ReasonAINotAllowed
	}
	if actor.Kind == KindSystem {
		return ReasonNotHuman
	}
	m.lapseAll()
	switch c.state {
	case StateRequested, StateConfirmed:
	case StateScheduled:
		if !m.now.Before(c.effective) {
			return ReasonAlreadyEffective
		}
		seen := false
		for _, other := range m.order {
			if other == c {
				seen = true
				continue
			}
			if seen && other.subject == c.subject && other.field == c.field && (other.state == StateRequested || other.state == StateConfirmed || other.state == StateScheduled) {
				return ReasonHasSuccessor
			}
		}
	default:
		return ReasonNotPending
	}
	c.state, c.ended, c.endedBy = StateWithdrawn, m.now, actor.ID
	return "ok"
}

func (m *model) resolve(subject, field string, at time.Time) (string, Ref) {
	var id string
	var r Ref
	for _, c := range m.order {
		if c.subject == subject && c.field == field && c.state == StateScheduled && !c.effective.After(at) {
			id, r = c.id, c.after
		}
	}
	return id, r
}

func TestPropertyAgainstModel(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(1); seed <= 600; seed++ {
		for state, count := range runModel(t, seed) {
			seen[state] += count
		}
	}
	for _, state := range []string{StateRequested, StateConfirmed, StateScheduled, StateRejected, StateWithdrawn, StateLapsed} {
		if seen[state] < 20 {
			t.Fatalf("state %s reached %d times", state, seen[state])
		}
	}
}

func runModel(t *testing.T, seed int64) map[string]int {
	rng := rand.New(rand.NewSource(seed))
	clock := &testClock{now: start}
	r := must(NewRegistry(Limits{Workspaces: 2, Channels: 10, Changes: 1000}, clock.read))
	rules := Rules{Approvals: 1 + rng.Intn(2), Validity: time.Duration(1+rng.Intn(3)) * day, MaxLead: 10 * day, ChannelAge: day}
	p := must(r.Propose("w", rules, human("admin")))
	must(r.Confirm("w", p.Revision, human("admin2")))
	for id, info := range modelChannels {
		kind := ChannelPhone
		if id == "ch-mail" {
			kind = ChannelEmail
		}
		must(r.Register("w", Channel{ID: id, Subject: info.subject, Kind: kind, Ref: Ref{ID: "vault:" + id, Fingerprint: info.print}, Evidence: "e"}, human("onboard")))
		must(r.Admit("w", id, "e", human("onboard2")))
	}
	clock.advance(2 * day)
	m := &model{now: clock.read(), changes: map[string]*modelChange{}, approval: rules.Approvals, validity: rules.Validity, lead: rules.MaxLead, retired: map[string]bool{}}
	actors := []Actor{human("clerk"), human("checker"), human("boss"), human("cfo"), human("onboard"), assistant("bot"), system("job")}
	subjects := []string{"acme", "beta"}
	fieldsUsed := []string{FieldPaymentAccount, FieldDocumentRecipient}
	refs := []Ref{ref("r1", 'a'), ref("r2", 'b'), ref("r3", 'c')}
	senders := map[string]string{}
	reached := map[string]map[string]bool{}
	pick := func() *modelChange {
		if len(m.order) == 0 {
			return nil
		}
		return m.order[rng.Intn(len(m.order))]
	}
	for step := 0; step < 100; step++ {
		label := "seed " + strconv.FormatInt(seed, 10) + " step " + strconv.Itoa(step)
		switch op := rng.Intn(10); {
		case op < 3:
			id := "c" + strconv.Itoa(step)
			subject := subjects[rng.Intn(2)]
			field := fieldsUsed[rng.Intn(2)]
			c := Change{ID: id, Subject: subject, Field: field, After: refs[rng.Intn(3)], Provenance: ProvenanceDocument, Effective: m.now.Add(time.Duration(rng.Intn(12*24)) * time.Hour)}
			if rng.Intn(6) == 0 {
				c.Provenance = ProvenanceAssistant
			}
			via := ""
			if subject == "acme" && rng.Intn(2) == 0 {
				via = "ch-mail"
				c.Instruction = Instruction{Kind: ChannelEmail, Channel: via, Sender: fingerprint('b'), Evidence: "e"}
			} else {
				c.Instruction = Instruction{Kind: ChannelEmail, Sender: fingerprint('e'), Evidence: "e"}
			}
			actor := actors[rng.Intn(len(actors))]
			want := m.request(id, subject, field, c.After, c.Effective, c.Provenance, via, actor)
			v, err := r.Request("w", c, actor)
			if got := kind(err); got != want {
				t.Fatalf("%s request: got %s want %s", label, got, want)
			}
			if err == nil {
				m.changes[id].digest = v.Digest
				senders[id] = c.Instruction.Sender
				if v.Before != m.changes[id].before {
					t.Fatalf("%s before", label)
				}
			}
		case op < 5:
			c := pick()
			if c == nil {
				continue
			}
			channels := []string{"ch-phone", "ch-mail", "ch-beta"}
			channel := channels[rng.Intn(3)]
			actor := actors[rng.Intn(len(actors))]
			if c.verifier == actor.ID && c.channel == channel {
				continue
			}
			outcome := OutcomeConfirmed
			if rng.Intn(8) == 0 {
				outcome = OutcomeDenied
			}
			digest := c.digest
			if rng.Intn(10) == 0 {
				digest = fingerprint('0')
			}
			want := m.verify(c, channel, actor, senders[c.id], outcome == OutcomeDenied, digest != c.digest)
			v, err := r.Verify("w", c.id, Verification{Channel: channel, Evidence: "e", Outcome: outcome, Digest: digest}, actor)
			if got := kind(err); got != want {
				t.Fatalf("%s verify %s via %s by %s: got %s want %s", label, c.id, channel, actor.ID, got, want)
			}
			if err == nil && outcome == OutcomeConfirmed {
				c.confirmation = v.Verification.Digest
			}
		case op < 7:
			c := pick()
			if c == nil || c.confirmation == "" || c.state == StateRejected {
				continue
			}
			actor := actors[rng.Intn(len(actors))]
			reject := rng.Intn(8) == 0
			if c.approvers[actor.ID] {
				reject = true
			}
			digest := c.confirmation
			if rng.Intn(10) == 0 {
				digest = fingerprint('0')
			}
			want := m.approve(c, actor, reject, digest != c.confirmation)
			a := approve(digest)
			if reject {
				a = Approval{Verdict: VerdictReject, Reason: "other", Digest: digest}
			}
			_, err := r.Approve("w", c.id, a, actor)
			if got := kind(err); got != want {
				t.Fatalf("%s approve %s by %s: got %s want %s", label, c.id, actor.ID, got, want)
			}
		case op < 8:
			c := pick()
			if c == nil {
				continue
			}
			actor := actors[rng.Intn(len(actors))]
			if c.state == StateWithdrawn {
				continue
			}
			want := m.withdraw(c, actor)
			_, err := r.Withdraw("w", c.id, actor)
			if got := kind(err); got != want {
				t.Fatalf("%s withdraw %s: got %s want %s", label, c.id, got, want)
			}
		case op < 9:
			if rng.Intn(25) == 0 && !m.retired["ch-phone"] {
				must(r.Retire("w", "ch-phone", human("ops")))
				m.retired["ch-phone"] = true
			}
		default:
			d := time.Duration(rng.Intn(48)) * time.Hour
			clock.advance(d)
			m.now = m.now.Add(d)
		}
		for _, subject := range subjects {
			for _, field := range fieldsUsed {
				for _, offset := range []time.Duration{0, 3 * day, 6 * day, 12 * day} {
					at := m.now.Add(offset)
					id, want := m.resolve(subject, field, at)
					got, err := r.Resolve("w", subject, field, at)
					if id == "" {
						if err != ErrNotFound {
							t.Fatalf("%s resolve %s/%s: %v %+v", label, subject, field, err, got)
						}
						continue
					}
					if err != nil || got.Change != id || got.Ref != want {
						t.Fatalf("%s resolve %s/%s: got %+v want %s", label, subject, field, got, id)
					}
				}
				m.lapseAll()
				for _, c := range m.order {
					if reached[c.id] == nil {
						reached[c.id] = map[string]bool{}
					}
					reached[c.id][c.state] = true
				}
				views := must(r.Trail("w", subject, field))
				index := 0
				for _, c := range m.order {
					if c.subject != subject || c.field != field {
						continue
					}
					v := views[index]
					index++
					if v.Change.ID != c.id || v.State != c.state || v.Before != c.before {
						t.Fatalf("%s trail %s: got %s/%s want %s", label, c.id, v.Change.ID, v.State, c.state)
					}
					if v.EndedBy != c.endedBy || !v.Scheduled.Equal(c.scheduled) {
						t.Fatalf("%s actors %s: got %s/%v want %s/%v", label, c.id, v.EndedBy, v.Scheduled, c.endedBy, c.scheduled)
					}
					if (c.state == StateLapsed || c.state == StateRejected || c.state == StateWithdrawn) && !v.Ended.Equal(c.ended) {
						t.Fatalf("%s ended %s: got %v want %v", label, c.id, v.Ended, c.ended)
					}
				}
				if index != len(views) {
					t.Fatalf("%s trail length", label)
				}
			}
		}
	}
	counts := map[string]int{}
	for _, states := range reached {
		for state := range states {
			counts[state]++
		}
	}
	return counts
}
