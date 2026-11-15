package events

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func (f world) bind(source, connection string) {
	f.t.Helper()
	f.must(second(f.r.BindSource("w1", source, connection, ann)))
	f.must(second(f.r.ApproveSource("w1", source, bob)))
}

func incoming(id, subject, revision string) Event {
	return Event{ID: id, Source: "urn:erp", Type: "invoice.posted", SchemaVersion: "v1", Subject: subject, ObjectRevision: revision,
		Data: map[string]string{"document": "d1"}}
}

func (f world) receive(e Event) Reception {
	f.t.Helper()
	got, err := f.r.Receive("w1", "c1", []string{e.ID}, e, relay)
	f.must(err)
	return got
}

func TestBindings(t *testing.T) {
	f := setup(t)
	expect(t, "no workspace", second(f.r.BindSource("", "urn:erp", "c1", ann)), ErrInvalid)
	expect(t, "no source", second(f.r.BindSource("w1", "", "c1", ann)), ErrInvalid)
	expect(t, "no connection", second(f.r.BindSource("w1", "urn:erp", "", ann)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.BindSource("w1", "urn:erp", "c1", Actor{ID: "ann"})), ErrInvalid)
	reason(t, "an assistant binding", second(f.r.BindSource("w1", "urn:erp", "c1", bot)), ReasonAINotAllowed)
	view, err := f.r.BindSource("w1", "urn:erp", "c1", ann)
	if err != nil || view.State != StateProposed || view.Connection != "c1" {
		t.Fatalf("binding %+v %v", view, err)
	}
	f.must(second(f.r.BindSource("w1", "urn:erp", "c1", ann)))
	expect(t, "another proposer", second(f.r.BindSource("w1", "urn:erp", "c1", bob)), ErrConflict)
	expect(t, "another connection", second(f.r.BindSource("w1", "urn:erp", "c2", ann)), ErrConflict)
	expect(t, "approve no source", second(f.r.ApproveSource("w1", "", bob)), ErrInvalid)
	reason(t, "an assistant approving", second(f.r.ApproveSource("w1", "urn:erp", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.r.ApproveSource("w9", "urn:erp", bob)), ErrNotFound)
	expect(t, "an unknown source", second(f.r.ApproveSource("w1", "urn:other", bob)), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveSource("w1", "urn:erp", ann)), ReasonSelfApproval)
	if active, err := f.r.ApproveSource("w1", "urn:erp", bob); err != nil || active.State != StateActive || active.Approver != "bob" {
		t.Errorf("approval %+v %v", active, err)
	}
	f.must(second(f.r.ApproveSource("w1", "urn:erp", bob)))
	reason(t, "another approver", second(f.r.ApproveSource("w1", "urn:erp", cat)), ReasonNotPending)
	expect(t, "binding an active source", second(f.r.BindSource("w1", "urn:erp", "c2", ann)), ErrConflict)
	if gone, err := f.r.UnbindSource("w1", "urn:erp", cat); err != nil || gone.State != StateRevoked {
		t.Errorf("unbinding %+v %v", gone, err)
	}
	reason(t, "approving an unbound source", second(f.r.ApproveSource("w1", "urn:erp", bob)), ReasonNotPending)
	expect(t, "unbinding an unknown source", second(f.r.UnbindSource("w1", "urn:nothing", ann)), ErrNotFound)
	if again, err := f.r.BindSource("w1", "urn:erp", "c2", ann); err != nil || again.Connection != "c2" || again.State != StateProposed {
		t.Errorf("binding again after unbinding %+v %v", again, err)
	}
	f.must(second(f.r.BindSource("w1", "urn:a", "c1", ann)))
	f.must(second(f.r.BindSource("w1", "urn:b", "c1", ann)))
	expect(t, "a fourth source", second(f.r.BindSource("w1", "urn:c", "c1", ann)), ErrLimit)
	l := limits
	l.Workspaces = 1
	g := setupWith(t, l)
	g.must(second(g.r.BindSource("w1", "urn:erp", "c1", ann)))
	expect(t, "a second workspace", second(g.r.BindSource("w2", "urn:erp", "c1", ann)), ErrLimit)
}

func TestReceiveForms(t *testing.T) {
	f := setup(t)
	f.bind("urn:erp", "c1")
	e := incoming("e1", "inv-1", "5")
	expect(t, "no workspace", second(f.r.Receive("", "c1", []string{"e1"}, e, relay)), ErrInvalid)
	expect(t, "no connection", second(f.r.Receive("w1", "", []string{"e1"}, e, relay)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Receive("w1", "c1", []string{"e1"}, e, Actor{ID: "x"})), ErrInvalid)
	expect(t, "an invalid event", second(f.r.Receive("w1", "c1", []string{"e1"}, Event{ID: "e1"}, relay)), ErrInvalid)
	expect(t, "an invalid verified identifier", second(f.r.Receive("w1", "c1", []string{"a b"}, e, relay)), ErrInvalid)
	expect(t, "too many verified identifiers", second(f.r.Receive("w1", "c1", make([]string, maxVerified+1), e, relay)), ErrInvalid)
	reason(t, "a human receiving", second(f.r.Receive("w1", "c1", []string{"e1"}, e, ann)), ReasonNotSystem)
	reason(t, "an identifier outside the push", second(f.r.Receive("w1", "c1", []string{"e2"}, e, relay)), ReasonUnverified)
	reason(t, "another connection", second(f.r.Receive("w1", "c2", []string{"e1"}, e, relay)), ReasonSourceNotBound)
	spoofed := e
	spoofed.Source = "urn:other"
	reason(t, "an unbound source", second(f.r.Receive("w1", "c1", []string{"e1"}, spoofed, relay)), ReasonSourceNotBound)
	reason(t, "an unknown workspace", second(f.r.Receive("w9", "c1", []string{"e1"}, e, relay)), ReasonSourceNotBound)
	f.must(second(f.r.BindSource("w1", "urn:new", "c1", ann)))
	pending := e
	pending.Source = "urn:new"
	reason(t, "a source that is not approved", second(f.r.Receive("w1", "c1", []string{"e1"}, pending, relay)), ReasonSourceNotBound)
	expect(t, "commit no token", second(f.r.Commit("w1", "", relay)), ErrInvalid)
	expect(t, "commit no workspace", second(f.r.Commit("", "t", relay)), ErrInvalid)
	reason(t, "a human committing", second(f.r.Commit("w1", "t", ann)), ReasonNotSystem)
	expect(t, "an unknown token", second(f.r.Commit("w1", "t", relay)), ErrStale)
	expect(t, "a token of an unknown workspace", second(f.r.Commit("w9", "t", relay)), ErrStale)
	expect(t, "abandon an unknown token", f.r.Abandon("w1", "t", relay), ErrStale)
	expect(t, "abandon an invalid actor", f.r.Abandon("w1", "t", Actor{}), ErrInvalid)
}

func TestReceive(t *testing.T) {
	f := setup(t)
	f.bind("urn:erp", "c1")
	first := f.receive(incoming("e5", "inv-1", "5"))
	if first.Decision != DecisionAccepted || first.Token == "" || first.Event.Data["document"] != "d1" {
		t.Fatalf("reception %+v", first)
	}
	if got := f.receive(incoming("e5", "inv-1", "5")); got.Decision != DecisionInProgress {
		t.Errorf("a redelivery of the reserved event %+v", got)
	}
	if got := f.receive(incoming("e4", "inv-1", "4")); got.Decision != DecisionInProgress {
		t.Errorf("an older event behind a pending newer one %+v", got)
	}
	if got := f.receive(incoming("e6", "inv-1", "6")); got.Decision != DecisionInProgress {
		t.Errorf("a newer event behind a pending one %+v", got)
	}
	expect(t, "a token of another workspace", second(f.r.Commit("w2", first.Token, relay)), ErrStale)
	done, err := f.r.Commit("w1", first.Token, relay)
	if err != nil || done.Decision != DecisionAccepted || done.Event.ID != "e5" {
		t.Fatalf("commit %+v %v", done, err)
	}
	expect(t, "a second commit", second(f.r.Commit("w1", first.Token, relay)), ErrStale)
	if got := f.receive(incoming("e5", "inv-1", "5")); got.Decision != DecisionDuplicate {
		t.Errorf("a duplicate %+v", got)
	}
	changed := incoming("e5", "inv-1", "5")
	changed.Data = map[string]string{"document": "d2"}
	if got := f.receive(changed); got.Decision != DecisionConflict {
		t.Errorf("the same identifier with other content %+v", got)
	}
	if got := f.receive(incoming("e4", "inv-1", "4")); got.Decision != DecisionStale {
		t.Errorf("an older event after the newer one committed %+v", got)
	}
	if got := f.receive(incoming("e5b", "inv-1", "5")); got.Decision != DecisionStale {
		t.Errorf("another event at the committed revision %+v", got)
	}
	f.advance(time.Hour)
	if got := f.receive(incoming("e5", "inv-1", "5")); got.Decision != DecisionDuplicate {
		t.Errorf("a redelivery after the seen identifier expired %+v", got)
	}
	other := incoming("e5", "inv-1", "5")
	other.Data = map[string]string{"document": "d9"}
	if got := f.receive(other); got.Decision != DecisionStale {
		t.Errorf("a changed event at the committed revision after expiry %+v", got)
	}
	newer := f.receive(incoming("e6", "inv-1", "6"))
	if newer.Decision != DecisionAccepted {
		t.Fatalf("a newer event %+v", newer)
	}
	f.must(f.r.Abandon("w1", newer.Token, relay))
	if got := f.receive(incoming("e6", "inv-1", "6")); got.Decision != DecisionAccepted {
		t.Errorf("the event after an abandon %+v", got)
	}
	plain := f.receive(Event{ID: "p1", Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
	if plain.Decision != DecisionAccepted {
		t.Fatalf("an event without a subject %+v", plain)
	}
	if got := f.receive(Event{ID: "p1", Source: "urn:erp", Type: "note", SchemaVersion: "v1"}); got.Decision != DecisionInProgress {
		t.Errorf("a pending event without a subject %+v", got)
	}
	unranked := f.receive(Event{ID: "u1", Source: "urn:erp", Type: "note", SchemaVersion: "v1", Subject: "inv-9"})
	if unranked.Decision != DecisionAccepted {
		t.Errorf("an event with a subject and no revision %+v", unranked)
	}
}

func TestReservations(t *testing.T) {
	f := setup(t)
	f.bind("urn:erp", "c1")
	crashed := f.receive(incoming("e1", "inv-1", "1"))
	f.advance(time.Minute - time.Nanosecond)
	if got := f.receive(incoming("e1", "inv-1", "1")); got.Decision != DecisionInProgress {
		t.Errorf("just before the reservation expires %+v", got)
	}
	f.advance(time.Nanosecond)
	again := f.receive(incoming("e1", "inv-1", "1"))
	if again.Decision != DecisionAccepted || again.Token == crashed.Token {
		t.Fatalf("a redelivery after a crash %+v", again)
	}
	expect(t, "the replaced reservation", second(f.r.Commit("w1", crashed.Token, relay)), ErrStale)
	f.advance(2 * time.Minute)
	if done, err := f.r.Commit("w1", again.Token, relay); err != nil || done.Decision != DecisionAccepted {
		t.Errorf("a late commit of an unreplaced reservation %+v %v", done, err)
	}
	old := f.receive(incoming("e2", "inv-2", "2"))
	f.advance(time.Minute)
	newer := f.receive(incoming("e3", "inv-2", "3"))
	if newer.Decision != DecisionAccepted {
		t.Fatalf("a newer event replacing an expired reservation %+v", newer)
	}
	expect(t, "a late commit replaced by the subject", second(f.r.Commit("w1", old.Token, relay)), ErrStale)
	f.must(second(f.r.Commit("w1", newer.Token, relay)))
	l := limits
	l.Seen, l.Revisions = 10, 10
	g := setupWith(t, l)
	g.bind("urn:erp", "c1")
	late := g.receive(incoming("e2", "inv-1", "2"))
	g.r.workspaces["w1"].revisions[subjectKey{"urn:erp", "inv-1"}] = revision{value: 3, id: "e3"}
	expect(t, "a commit behind a newer revision", second(g.r.Commit("w1", late.Token, relay)), ErrStale)
	if _, seen := g.r.workspaces["w1"].seen[eventKey{"urn:erp", "e2"}]; seen {
		t.Errorf("a refused commit marked the event as seen")
	}
}

func TestInboxCapacity(t *testing.T) {
	f := setup(t)
	f.bind("urn:erp", "c1")
	for index, id := range []string{"a", "b", "c"} {
		got := f.receive(Event{ID: id, Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
		if index < 2 {
			f.must(second(f.r.Commit("w1", got.Token, relay)))
		}
	}
	f.receive(Event{ID: "d", Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
	reason(t, "a full inbox", second(f.r.Receive("w1", "c1", []string{"e"}, Event{ID: "e", Source: "urn:erp", Type: "note", SchemaVersion: "v1"}, relay)), ReasonInboxCapacity)
	f.advance(time.Minute)
	if got := f.receive(Event{ID: "e", Source: "urn:erp", Type: "note", SchemaVersion: "v1"}); got.Decision != DecisionAccepted {
		t.Errorf("after expired reservations were swept %+v", got)
	}
	f.advance(time.Hour)
	if got := f.receive(Event{ID: "f", Source: "urn:erp", Type: "note", SchemaVersion: "v1"}); got.Decision != DecisionAccepted {
		t.Errorf("after seen identifiers expired %+v", got)
	}
	if s := f.r.workspaces["w1"]; len(s.seen) != 0 || s.queued != 0 || len(s.queue) != 0 {
		t.Errorf("the seen queue was not swept and compacted: %d seen, %d queued, %d slots", len(s.seen), s.queued, len(s.queue))
	}
	g := setup(t)
	g.bind("urn:erp", "c1")
	for _, subject := range []string{"s1", "s2"} {
		g.must(second(g.r.Commit("w1", g.receive(incoming("e-"+subject, subject, "1")).Token, relay)))
	}
	pending := g.receive(incoming("e-s3", "s3", "1"))
	g.advance(time.Minute)
	reason(t, "a new subject beyond the revision bound", second(g.r.Receive("w1", "c1", []string{"e-s4"}, incoming("e-s4", "s4", "1"), relay)), ReasonRevisionLimit)
	if got := g.receive(incoming("e-s1b", "s1", "2")); got.Decision != DecisionAccepted {
		t.Errorf("an existing subject at the revision bound %+v", got)
	}
	g.must(second(g.r.Commit("w1", pending.Token, relay)))
	if len(g.r.workspaces["w1"].revisions) != 3 || g.r.workspaces["w1"].fresh != 0 {
		t.Errorf("%d revision records, %d fresh", len(g.r.workspaces["w1"].revisions), g.r.workspaces["w1"].fresh)
	}
	h := setup(t)
	h.bind("urn:erp", "c1")
	for _, subject := range []string{"s1", "s2"} {
		h.must(second(h.r.Commit("w1", h.receive(incoming("e-"+subject, subject, "1")).Token, relay)))
	}
	h.receive(incoming("e-s3", "s3", "1"))
	h.advance(time.Minute)
	if got := h.receive(incoming("e-s3", "s3", "1")); got.Decision != DecisionAccepted {
		t.Errorf("a redelivery of a crashed new subject at the revision bound %+v", got)
	}
	if h.r.workspaces["w1"].fresh != 1 {
		t.Errorf("fresh %d", h.r.workspaces["w1"].fresh)
	}
}

func TestSeenQueue(t *testing.T) {
	f := setup(t)
	f.bind("urn:erp", "c1")
	for _, id := range []string{"a", "b"} {
		f.must(second(f.r.Commit("w1", f.receive(Event{ID: id, Source: "urn:erp", Type: "note", SchemaVersion: "v1"}).Token, relay)))
		f.advance(time.Minute)
	}
	f.advance(time.Hour - 2*time.Minute)
	f.receive(Event{ID: "x", Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
	if s := f.r.workspaces["w1"]; len(s.seen) != 1 || len(s.queue) != 1 || s.queued != 0 {
		t.Errorf("compaction at exactly half: %d seen, %d slots, %d queued", len(s.seen), len(s.queue), s.queued)
	}
	f.must(second(f.r.Commit("w1", f.receive(Event{ID: "c", Source: "urn:erp", Type: "note", SchemaVersion: "v1"}).Token, relay)))
	f.must(second(f.r.Commit("w1", f.receive(Event{ID: "d", Source: "urn:erp", Type: "note", SchemaVersion: "v1"}).Token, relay)))
	f.advance(time.Minute)
	f.receive(Event{ID: "y", Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
	if s := f.r.workspaces["w1"]; s.queued != 1 || len(s.queue) != 3 {
		t.Errorf("compaction below half: %d queued, %d slots", s.queued, len(s.queue))
	}
}

func TestBodyEscaping(t *testing.T) {
	stored := Stored{Event: Event{ID: "e1", Source: "urn:a&b<c>", Type: "t", SchemaVersion: "v1", Data: map[string]string{}}, Time: start, SpecVersion: "1.0"}
	if got := string(Body(stored)); !strings.Contains(got, "\"source\":\"urn:a\\u0026b\\u003cc\\u003e\"") {
		t.Errorf("body %s", got)
	}
}

func TestExpiringQueueStaysBounded(t *testing.T) {
	l := limits
	l.Seen = 10000
	f := setupWith(t, l)
	f.bind("urn:erp", "c1")
	held := f.receive(Event{ID: "held", Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
	f.advance(time.Hour)
	for index := 0; index < 500; index++ {
		got := f.receive(Event{ID: "e" + strconv.Itoa(index), Source: "urn:erp", Type: "note", SchemaVersion: "v1"})
		if index%2 == 0 {
			f.must(second(f.r.Commit("w1", got.Token, relay)))
		} else {
			f.must(f.r.Abandon("w1", got.Token, relay))
		}
		if s := f.r.workspaces["w1"]; len(s.expiring)-s.expired > 2*len(s.reservations)+1 {
			t.Fatalf("step %d: %d queued for %d reservations", index, len(s.expiring)-s.expired, len(s.reservations))
		}
	}
	if done, err := f.r.Commit("w1", held.Token, relay); err != nil || done.Decision != DecisionAccepted {
		t.Errorf("the old expired reservation at the head %+v %v", done, err)
	}
	if s := f.r.workspaces["w1"]; len(s.expiring)-s.expired > 1 {
		t.Errorf("%d queued after every reservation ended", len(s.expiring)-s.expired)
	}
}

func TestEdges(t *testing.T) {
	for _, name := range []string{"az", "a0", "a9"} {
		if !validField(name) {
			t.Errorf("field %q is refused", name)
		}
	}
	full := Event{ID: "e1", Source: "s", Type: "t", SchemaVersion: "v1", Data: map[string]string{}}
	for index := 0; index < maxFields; index++ {
		full.Data["f"+strings.Repeat("a", index)] = "x"
	}
	if !eventForm(full) {
		t.Errorf("an event with the maximum number of fields is refused")
	}
	f := setup(t)
	f.bind("urn:erp", "c1")
	verified := make([]string, maxVerified)
	for index := range verified {
		verified[index] = "v" + strconv.Itoa(index)
	}
	verified[0] = "e1"
	if got, err := f.r.Receive("w1", "c1", verified, incoming("e1", "s1", "1"), relay); err != nil || got.Decision != DecisionAccepted {
		t.Errorf("the maximum number of verified identifiers %+v %v", got, err)
	}
	g := setup(t)
	g.bind("urn:erp", "c1")
	equal := g.receive(incoming("e2", "inv-1", "2"))
	g.r.workspaces["w1"].revisions[subjectKey{"urn:erp", "inv-1"}] = revision{value: 2, id: "other"}
	expect(t, "a commit at an equal committed revision", second(g.r.Commit("w1", equal.Token, relay)), ErrStale)
	s := &space{events: []Stored{{Sequence: 1, Time: start}, {Sequence: 2, Time: start.Add(time.Hour)}}, published: map[eventKey]uint64{}}
	f.r.sweep(s, start.Add(time.Hour))
	if s.first != 0 || len(s.events) != 1 {
		t.Errorf("the outbox was not compacted at exactly half: first %d, %d events", s.first, len(s.events))
	}
	q := &space{seen: map[eventKey]seenEvent{{id: "a"}: {at: start}, {id: "b"}: {at: start.Add(time.Hour)}}, queue: []eventKey{{id: "a"}, {id: "b"}}}
	f.r.sweepSeen(q, start.Add(time.Hour))
	if q.queued != 0 || len(q.queue) != 1 {
		t.Errorf("the seen queue was not compacted at exactly half: %d queued, %d slots", q.queued, len(q.queue))
	}
	old := &reservation{token: "old", expires: start}
	young := &reservation{token: "young", expires: start.Add(time.Hour)}
	r := &space{reservations: map[string]*reservation{"old": old, "young": young}, pending: map[eventKey]string{}, subjects: map[subjectKey]string{},
		expiring: []*reservation{old, young}}
	f.r.sweepReservations(r, start)
	if r.expired != 0 || len(r.expiring) != 1 || r.expiring[0] != young {
		t.Errorf("the expiry queue was not compacted at exactly half: %d expired, %d slots", r.expired, len(r.expiring))
	}
	gone := []*reservation{{gone: true}, {gone: true}, {gone: true}}
	live := &reservation{token: "live", expires: start.Add(time.Hour)}
	tidy := &space{reservations: map[string]*reservation{"live": live}, expiring: append([]*reservation{live}, gone...)}
	f.r.tidy(tidy)
	if tidy.expired != 0 || len(tidy.expiring) != 1 || tidy.expiring[0] != live {
		t.Errorf("the rebuilt queue %d expired, %d slots", tidy.expired, len(tidy.expiring))
	}
}
