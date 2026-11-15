package events

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	start   = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	limits  = Limits{Workspaces: 3, Types: 4, Events: 4, Retention: time.Hour, Page: 2, Sources: 3, Seen: 4, InboxRetention: time.Hour, Revisions: 3, Reservation: time.Minute}
	ann     = Actor{ID: "ann", Kind: KindHuman}
	bob     = Actor{ID: "bob", Kind: KindHuman}
	cat     = Actor{ID: "cat", Kind: KindHuman}
	bot     = Actor{ID: "bot", Kind: KindAssistant}
	relay   = Actor{ID: "relay", Kind: KindSystem}
	invoice = Type{Name: "invoice.issued", SchemaVersion: "v1", Fields: []string{"invoice_id", "customer_id"}}
)

type world struct {
	t   testing.TB
	now *time.Time
	r   *Registry
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	r, err := NewRegistry(l, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, now: &now, r: r}
}

func setup(t testing.TB) world { return setupWith(t, limits) }

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) advance(d time.Duration) { *f.now = f.now.Add(d) }

func (f world) register(t Type) {
	f.t.Helper()
	f.must(second(f.r.ProposeType("w1", t, ann)))
	f.must(second(f.r.ApproveType("w1", t.Name, t.SchemaVersion, bob)))
}

func event(id string) Event {
	return Event{ID: id, Source: "urn:billing", Type: "invoice.issued", SchemaVersion: "v1", Subject: "inv-1", ObjectRevision: "1",
		Data: map[string]string{"invoice_id": "inv-1"}}
}

func expect(t testing.TB, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || want == nil && err != nil {
		t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var refusal Refusal
	if !errors.As(err, &refusal) || refusal.Reason != want {
		t.Errorf("%s: %v; want %s", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }

func TestNewRegistry(t *testing.T) {
	clock := func() time.Time { return start }
	if _, err := NewRegistry(limits, clock); err != nil {
		t.Fatal(err)
	}
	expect(t, "no clock", second(NewRegistry(limits, nil)), ErrInvalid)
	counts := map[string]struct {
		field func(*Limits) *int
		max   int
	}{
		"workspaces": {func(l *Limits) *int { return &l.Workspaces }, maxWorkspaces}, "types": {func(l *Limits) *int { return &l.Types }, maxTypes},
		"events": {func(l *Limits) *int { return &l.Events }, maxEvents}, "page": {func(l *Limits) *int { return &l.Page }, maxPage},
		"sources": {func(l *Limits) *int { return &l.Sources }, maxSources}, "seen": {func(l *Limits) *int { return &l.Seen }, maxSeen},
		"revisions": {func(l *Limits) *int { return &l.Revisions }, maxRevisions},
	}
	for name, c := range counts {
		for value, ok := range map[int]bool{0: false, 1: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			if _, err := NewRegistry(l, clock); ok != (err == nil) {
				t.Errorf("%s %d: %v", name, value, err)
			}
		}
	}
	for name, c := range map[string]struct {
		field    func(*Limits) *time.Duration
		min, max time.Duration
	}{
		"retention":       {func(l *Limits) *time.Duration { return &l.Retention }, minRetention, maxRetention},
		"inbox retention": {func(l *Limits) *time.Duration { return &l.InboxRetention }, minRetention, maxRetention},
		"reservation":     {func(l *Limits) *time.Duration { return &l.Reservation }, minReservation, maxReservation},
	} {
		for value, ok := range map[time.Duration]bool{c.min - 1: false, c.min: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			if _, err := NewRegistry(l, clock); ok != (err == nil) {
				t.Errorf("%s %v: %v", name, value, err)
			}
		}
	}
	for _, refusal := range []error{Refusal{Reason: ReasonStale()}, Expired{}} {
		if refusal.Error() == "" {
			t.Errorf("an empty error text")
		}
	}
	if !errors.Is(Expired{}, ErrExpired) || !errors.Is(Refusal{}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Errorf("error wrapping")
	}
}

func ReasonStale() string { return DecisionStale }

func TestForms(t *testing.T) {
	for _, id := range []string{"x", strings.Repeat("a", maxIDBytes), "A.z_0:-9", "Z", "z", "9"} {
		if !validID(id) {
			t.Errorf("%q is refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("a", maxIDBytes+1), "a/b", "a b", "é", "a@", "[", "`", "{"} {
		if validID(id) {
			t.Errorf("%q is accepted", id)
		}
	}
	for _, name := range []string{"a", "z", "invoice_id", "a9", strings.Repeat("a", maxFieldName)} {
		if !validField(name) {
			t.Errorf("field %q is refused", name)
		}
	}
	for _, name := range []string{"", "A", "9a", "_a", "a-b", "a.b", "aZ", "a{", "a`", strings.Repeat("a", maxFieldName+1)} {
		if validField(name) {
			t.Errorf("field %q is accepted", name)
		}
	}
	for _, source := range []string{"urn:billing", "https://example.com/x", strings.Repeat("a", maxSourceBytes), "~"} {
		if !validSource(source) {
			t.Errorf("source %q is refused", source)
		}
	}
	for _, source := range []string{"", "a b", "https://user@example.com", "a\\b", "é", "\x7f", strings.Repeat("a", maxSourceBytes+1)} {
		if validSource(source) {
			t.Errorf("source %q is accepted", source)
		}
	}
	for text, want := range map[string]uint64{"0": 0, "1": 1, "9": 9, "18446744073709551615": 18446744073709551615} {
		if got, ok := parseRevision(text); !ok || got != want {
			t.Errorf("revision %q: %d %v", text, got, ok)
		}
	}
	for _, text := range []string{"", "01", "00", "1a", "-1", "18446744073709551616", "123456789012345678901", "/", ":"} {
		if _, ok := parseRevision(text); ok {
			t.Errorf("revision %q is accepted", text)
		}
	}
}

func TestTypes(t *testing.T) {
	f := setup(t)
	many := make([]string, maxFields+1)
	for index := range many {
		many[index] = "f" + strings.Repeat("a", index)
	}
	for name, c := range map[string]struct {
		workspace string
		t         Type
		actor     Actor
	}{
		"no workspace":      {"", invoice, ann},
		"no name":           {"w1", Type{SchemaVersion: "v1"}, ann},
		"no version":        {"w1", Type{Name: "x"}, ann},
		"a field as text":   {"w1", Type{Name: "x", SchemaVersion: "v1", Fields: []string{"A"}}, ann},
		"a repeated field":  {"w1", Type{Name: "x", SchemaVersion: "v1", Fields: []string{"a", "a"}}, ann},
		"too many fields":   {"w1", Type{Name: "x", SchemaVersion: "v1", Fields: many}, ann},
		"an invalid actor":  {"w1", invoice, Actor{ID: "ann"}},
		"an empty actor id": {"w1", invoice, Actor{Kind: KindHuman}},
	} {
		expect(t, name, second(f.r.ProposeType(c.workspace, c.t, c.actor)), ErrInvalid)
	}
	f.must(second(f.r.ProposeType("w1", Type{Name: "max", SchemaVersion: "v1", Fields: many[:maxFields]}, ann)))
	reason(t, "an assistant proposing", second(f.r.ProposeType("w1", invoice, bot)), ReasonAINotAllowed)
	reason(t, "a system proposing", second(f.r.ProposeType("w1", invoice, relay)), ReasonNotHuman)
	view, err := f.r.ProposeType("w1", invoice, ann)
	if err != nil || view.State != StateProposed || strings.Join(view.Type.Fields, ",") != "customer_id,invoice_id" {
		t.Fatalf("proposal %+v %v", view, err)
	}
	view.Type.Fields[0] = "changed"
	if again, err := f.r.ProposeType("w1", Type{Name: "invoice.issued", SchemaVersion: "v1", Fields: []string{"invoice_id", "customer_id"}}, ann); err != nil || again.Type.Fields[0] != "customer_id" {
		t.Errorf("a repeated proposal %+v %v", again, err)
	}
	expect(t, "another proposer", second(f.r.ProposeType("w1", invoice, bob)), ErrConflict)
	expect(t, "other fields", second(f.r.ProposeType("w1", Type{Name: "invoice.issued", SchemaVersion: "v1", Fields: []string{"invoice_id", "other"}}, ann)), ErrConflict)
	expect(t, "fewer fields", second(f.r.ProposeType("w1", Type{Name: "invoice.issued", SchemaVersion: "v1", Fields: []string{"invoice_id"}}, ann)), ErrConflict)
	expect(t, "approve no workspace", second(f.r.ApproveType("", "invoice.issued", "v1", bob)), ErrInvalid)
	expect(t, "approve no name", second(f.r.ApproveType("w1", "", "v1", bob)), ErrInvalid)
	expect(t, "approve no version", second(f.r.ApproveType("w1", "invoice.issued", "", bob)), ErrInvalid)
	expect(t, "approve an invalid actor", second(f.r.ApproveType("w1", "invoice.issued", "v1", Actor{ID: "bob"})), ErrInvalid)
	reason(t, "an assistant approving", second(f.r.ApproveType("w1", "invoice.issued", "v1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.r.ApproveType("w9", "invoice.issued", "v1", bob)), ErrNotFound)
	expect(t, "an unknown type", second(f.r.ApproveType("w1", "invoice.issued", "v2", bob)), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveType("w1", "invoice.issued", "v1", ann)), ReasonSelfApproval)
	if active, err := f.r.ApproveType("w1", "invoice.issued", "v1", bob); err != nil || active.State != StateActive || active.Approver != "bob" {
		t.Errorf("approval %+v %v", active, err)
	}
	f.must(second(f.r.ApproveType("w1", "invoice.issued", "v1", bob)))
	reason(t, "another approver", second(f.r.ApproveType("w1", "invoice.issued", "v1", cat)), ReasonNotPending)
	expect(t, "proposing an active type", second(f.r.ProposeType("w1", invoice, ann)), ErrConflict)
	f.must(second(f.r.ProposeType("w1", Type{Name: "a", SchemaVersion: "v1"}, ann)))
	f.must(second(f.r.ProposeType("w1", Type{Name: "b", SchemaVersion: "v1"}, ann)))
	expect(t, "a fifth type", second(f.r.ProposeType("w1", Type{Name: "c", SchemaVersion: "v1"}, ann)), ErrLimit)
	l := limits
	l.Workspaces = 1
	g := setupWith(t, l)
	g.must(second(g.r.ProposeType("w1", invoice, ann)))
	expect(t, "a second workspace", second(g.r.ProposeType("w2", invoice, ann)), ErrLimit)
}

func TestPublish(t *testing.T) {
	f := setup(t)
	reason(t, "a workspace without types", second(f.r.Publish("w1", event("e1"), relay)), ReasonUnknownType)
	f.register(invoice)
	for name, change := range map[string]func(*Event){
		"no identifier":        func(e *Event) { e.ID = "" },
		"no source":            func(e *Event) { e.Source = "" },
		"no type":              func(e *Event) { e.Type = "" },
		"no schema version":    func(e *Event) { e.SchemaVersion = "" },
		"a subject as text":    func(e *Event) { e.Subject = "a b" },
		"a revision with zero": func(e *Event) { e.ObjectRevision = "01" },
		"a field as text":      func(e *Event) { e.Data = map[string]string{"A": "x"} },
		"a value as text":      func(e *Event) { e.Data = map[string]string{"invoice_id": "Acme Corp"} },
		"too many fields": func(e *Event) {
			e.Data = map[string]string{}
			for index := 0; index <= maxFields; index++ {
				e.Data["f"+strings.Repeat("a", index)] = "x"
			}
		},
	} {
		e := event("e1")
		change(&e)
		expect(t, name, second(f.r.Publish("w1", e, relay)), ErrInvalid)
	}
	expect(t, "no workspace", second(f.r.Publish("", event("e1"), relay)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Publish("w1", event("e1"), Actor{ID: "relay"})), ErrInvalid)
	reason(t, "a human publishing", second(f.r.Publish("w1", event("e1"), ann)), ReasonNotSystem)
	other := event("e1")
	other.SchemaVersion = "v2"
	reason(t, "an unregistered version", second(f.r.Publish("w1", other, relay)), ReasonUnknownType)
	f.must(second(f.r.ProposeType("w1", Type{Name: "pending", SchemaVersion: "v1"}, ann)))
	pending := event("e1")
	pending.Type, pending.Data = "pending", nil
	reason(t, "a type that is not approved", second(f.r.Publish("w1", pending, relay)), ReasonUnknownType)
	extra := event("e1")
	extra.Data = map[string]string{"invoice_id": "inv-1", "amount": "100"}
	reason(t, "a field outside the type", second(f.r.Publish("w1", extra, relay)), ReasonUnknownField)
	first, err := f.r.Publish("w1", event("e1"), relay)
	if err != nil || first.Sequence != 1 || !first.Time.Equal(start) || first.SpecVersion != "1.0" {
		t.Fatalf("publication %+v %v", first, err)
	}
	first.Event.Data["invoice_id"] = "changed"
	f.advance(time.Minute)
	if again, err := f.r.Publish("w1", event("e1"), relay); err != nil || again.Sequence != 1 || !again.Time.Equal(start) || again.Event.Data["invoice_id"] != "inv-1" {
		t.Errorf("a repeat %+v %v", again, err)
	}
	changed := event("e1")
	changed.Subject = "inv-2"
	expect(t, "the same identifier with other content", second(f.r.Publish("w1", changed, relay)), ErrConflict)
	for name, change := range map[string]func(*Event){
		"another type":     func(e *Event) { e.Type = "x" },
		"another version":  func(e *Event) { e.SchemaVersion = "v2" },
		"another revision": func(e *Event) { e.ObjectRevision = "2" },
		"another value":    func(e *Event) { e.Data = map[string]string{"invoice_id": "inv-2"} },
		"another field":    func(e *Event) { e.Data = map[string]string{"customer_id": "inv-1"} },
		"more data":        func(e *Event) { e.Data = map[string]string{"invoice_id": "inv-1", "customer_id": "c1"} },
	} {
		e := event("e1")
		change(&e)
		expect(t, name, second(f.r.Publish("w1", e, relay)), ErrConflict)
	}
	f.advance(-30 * time.Second)
	later, err := f.r.Publish("w1", event("e2"), relay)
	if err != nil || later.Sequence != 2 || !later.Time.Equal(start.Add(time.Minute)) {
		t.Errorf("a later event %+v %v", later, err)
	}
	f.r.workspaces["w1"].last = start.Add(time.Hour)
	if clamped, _ := f.r.Publish("w1", event("e3"), relay); !clamped.Time.Equal(start.Add(time.Hour)) {
		t.Errorf("an event time went back: %v", clamped.Time)
	}
	f.must(second(f.r.Publish("w1", event("e4"), relay)))
	expect(t, "a fifth event", second(f.r.Publish("w1", event("e5"), relay)), ErrLimit)
	f.advance(2 * time.Hour)
	f.must(second(f.r.Publish("w1", event("e5"), relay)))
	if again, err := f.r.Publish("w1", event("e1"), relay); err != nil || again.Sequence != 6 {
		t.Errorf("a repeat after retention %+v %v", again, err)
	}
}

func TestBody(t *testing.T) {
	stored := Stored{Event: Event{ID: "e1", Source: "urn:billing", Type: "invoice.issued", SchemaVersion: "v1", Subject: "inv-1", ObjectRevision: "7",
		Data: map[string]string{"invoice_id": "inv-1", "customer_id": "c1"}}, Sequence: 3, Time: start, SpecVersion: "1.0"}
	want := `{"data":{"customer_id":"c1","invoice_id":"inv-1"},"id":"e1","objectrevision":"7","schemaversion":"v1","source":"urn:billing","specversion":"1.0","subject":"inv-1","time":"2026-11-15T09:00:00Z","type":"invoice.issued"}`
	if got := string(Body(stored)); got != want {
		t.Errorf("body %s", got)
	}
	stored.Event.Subject, stored.Event.ObjectRevision, stored.Event.Data = "", "", nil
	if got := string(Body(stored)); got != `{"data":{},"id":"e1","schemaversion":"v1","source":"urn:billing","specversion":"1.0","time":"2026-11-15T09:00:00Z","type":"invoice.issued"}` {
		t.Errorf("a body without subject %s", got)
	}
}

func TestRead(t *testing.T) {
	f := setup(t)
	if page, err := f.r.Read("w1", 0, 5, relay); err != nil || page.Head != 0 || len(page.Events) != 0 {
		t.Errorf("a fresh workspace %+v %v", page, err)
	}
	expect(t, "a cursor in a fresh workspace", second(f.r.Read("w1", 1, 5, relay)), ErrInvalid)
	expect(t, "no workspace", second(f.r.Read("", 0, 5, relay)), ErrInvalid)
	expect(t, "no limit", second(f.r.Read("w1", 0, 0, relay)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Read("w1", 0, 1, Actor{ID: "x"})), ErrInvalid)
	reason(t, "a human reading", second(f.r.Read("w1", 0, 1, ann)), ReasonNotSystem)
	f.register(invoice)
	for _, id := range []string{"e1", "e2", "e3"} {
		f.must(second(f.r.Publish("w1", event(id), relay)))
		f.advance(time.Minute)
	}
	page, err := f.r.Read("w1", 0, 5, relay)
	if err != nil || len(page.Events) != 2 || page.Events[0].Sequence != 1 || page.Head != 3 {
		t.Fatalf("a page %+v %v", page, err)
	}
	page.Events[0].Event.Data["invoice_id"] = "changed"
	if again, _ := f.r.Read("w1", 0, 1, relay); len(again.Events) != 1 || again.Events[0].Event.Data["invoice_id"] != "inv-1" {
		t.Errorf("the page is shared with the caller")
	}
	if rest, _ := f.r.Read("w1", 2, 5, relay); len(rest.Events) != 1 || rest.Events[0].Sequence != 3 {
		t.Errorf("the rest %+v", rest)
	}
	if end, err := f.r.Read("w1", 3, 5, relay); err != nil || len(end.Events) != 0 {
		t.Errorf("at the head %+v %v", end, err)
	}
	expect(t, "beyond the head", second(f.r.Read("w1", 4, 5, relay)), ErrInvalid)
	f.advance(time.Hour - 3*time.Minute)
	if page, err := f.r.Read("w1", 1, 5, relay); err != nil || page.Events[0].Sequence != 2 {
		t.Errorf("after the first expired %+v %v", page, err)
	}
	var expired Expired
	if err := second(f.r.Read("w1", 0, 5, relay)); !errors.As(err, &expired) || expired.Resync != (Resync{Floor: 1, Head: 3}) {
		t.Errorf("below the floor %v", err)
	}
	f.advance(2 * time.Minute)
	err = second(f.r.Read("w1", 0, 5, relay))
	if !errors.As(err, &expired) || expired.Resync != (Resync{Floor: 3, Head: 3}) {
		t.Errorf("after a full sweep %v", err)
	}
	if page, err := f.r.Read("w1", 3, 5, relay); err != nil || len(page.Events) != 0 {
		t.Errorf("at the floor %+v %v", page, err)
	}
	if f.r.workspaces["w1"].first != 0 || len(f.r.workspaces["w1"].events) != 0 {
		t.Errorf("the outbox was not compacted")
	}
	f.must(second(f.r.Publish("w1", event("e4"), relay)))
	f.must(second(f.r.Publish("w1", event("e5"), relay)))
	f.must(second(f.r.Publish("w1", event("e6"), relay)))
	f.advance(time.Hour)
	f.must(second(f.r.Publish("w1", event("e7"), relay)))
	f.advance(-time.Hour)
	if s := f.r.workspaces["w1"]; s.first != 0 || len(s.events) != 1 || s.floor != 6 {
		t.Errorf("compaction at exactly half: first %d, %d events, floor %d", s.first, len(s.events), s.floor)
	}
}

func TestDigest(t *testing.T) {
	e := Event{ID: "e1", Source: "s", Type: "t", SchemaVersion: "v1", Subject: "x", ObjectRevision: "2", Data: map[string]string{"b": "2", "a": "1"}}
	var parts []byte
	for _, value := range []string{"e1", "s", "t", "v1", "x", "2", "2", "a", "1", "b", "2"} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		parts = append(append(parts, size[:]...), value...)
	}
	want := sha256.Sum256(parts)
	if got := Digest(e); got != want {
		t.Errorf("digest %s", hex.EncodeToString(got[:]))
	}
}
