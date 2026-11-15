package incidents

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRegistryNeedsDependenciesAndLimits(t *testing.T) {
	f := open(t, testLimits())
	for _, c := range []Config{{Zones: time.LoadLocation}, {Permissions: f.perms}} {
		if _, err := NewRegistry(c, testLimits(), f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatal("config")
		}
	}
	if _, err := NewRegistry(Config{Permissions: f.perms, Zones: time.LoadLocation}, testLimits(), nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("clock")
	}
	for _, l := range []Limits{{0, 1, 1, 1, 1, 1}, {1, maxIncidents + 1, 1, 1, 1, 1}, {1, 1, 0, 1, 1, 1}, {1, 1, 1, maxRecipients + 1, 1, 1}, {1, 1, 1, 1, 0, 1}, {1, 1, 1, 1, maxSubscriptions + 1, 1}, {1, 1, 1, 1, 1, 0}} {
		if _, err := NewRegistry(Config{Permissions: f.perms, Zones: time.LoadLocation}, l, f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatal(l)
		}
	}
	if _, err := NewRegistry(Config{Permissions: f.perms, Zones: time.LoadLocation}, Limits{maxWorkspaces, maxIncidents, maxSignals, maxRecipients, maxSubscriptions, maxNotes}, f.clock.read); err != nil {
		t.Fatal(err)
	}
	if (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" || !errors.Is(Refusal{Reason: "x"}, ErrRejected) {
		t.Fatal("refusal")
	}
	if !validID(strings.Repeat("Z", maxIDBytes)) || validID(strings.Repeat("z", maxIDBytes+1)) || validID("!a") || validID("a!") || validID("") || !validID("Az09._:-") {
		t.Fatal("identifiers")
	}
	if cadence(DigestHourly) != hour || cadence(DigestDaily) != day {
		t.Fatal("cadence")
	}
}

func TestRaiseAndResolveGates(t *testing.T) {
	f := open(t, testLimits())
	good := signal("s1", "c", "k1", SeverityWarning, false, start)
	for _, bad := range []Signal{{Condition: "c", Subject: "k", Severity: SeverityInfo, Occurred: start}, {ID: "s", Subject: "k", Severity: SeverityInfo, Occurred: start},
		{ID: "s", Condition: "c", Severity: SeverityInfo, Occurred: start}, {ID: "s", Condition: "c", Subject: "k", Severity: "fatal", Occurred: start},
		{ID: "s", Condition: "c", Subject: "k", Severity: SeverityInfo}} {
		expect(t, "form", second(f.r.Raise("ws", bad, system("m"))), "invalid")
	}
	expect(t, "workspace", second(f.r.Raise("", good, system("m"))), "invalid")
	expect(t, "actor", second(f.r.Raise("ws", good, Actor{ID: "m"})), "invalid")
	expect(t, "assistant", second(f.r.Raise("ws", good, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "human", second(f.r.Raise("ws", good, human("ops"))), RefusalNotSystem)
	future := good
	future.Occurred = start.Add(time.Second)
	expect(t, "future", second(f.r.Raise("ws", future, system("m"))), RefusalFutureOccurrence)
	inc := f.raise(t, good)
	if again := must(f.r.Raise("ws", good, system("other"))); again.ID != inc.ID || again.Count != 1 {
		t.Fatal("repeat")
	}
	changed := good
	changed.Severity = SeverityCritical
	expect(t, "changed", second(f.r.Raise("ws", changed, system("m"))), "conflict")
	res := Resolution{Reason: ReasonFalseAlarm, Evidence: "e"}
	expect(t, "resolve form", second(f.r.Resolve("ws", inc.ID, Resolution{Reason: "x", Evidence: "e"}, human("ops"))), "invalid")
	expect(t, "resolve evidence", second(f.r.Resolve("ws", inc.ID, Resolution{Reason: ReasonFixed}, human("ops"))), "invalid")
	expect(t, "resolve workspace", second(f.r.Resolve("", inc.ID, res, human("ops"))), "invalid")
	expect(t, "resolve id", second(f.r.Resolve("ws", "", res, human("ops"))), "invalid")
	expect(t, "resolve actor", second(f.r.Resolve("ws", inc.ID, res, Actor{ID: "x"})), "invalid")
	expect(t, "resolve assistant", second(f.r.Resolve("ws", inc.ID, res, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "resolve unknown", second(f.r.Resolve("ws", "inc-9", res, human("ops"))), "not_found")
	expect(t, "resolve workspace unknown", second(f.r.Resolve("ws9", inc.ID, res, human("ops"))), "not_found")
	f.perms.deny("ops", "k1")
	expect(t, "not permitted", second(f.r.Resolve("ws", inc.ID, res, human("ops"))), RefusalNotPermitted)
	f.perms.failing["ops"] = true
	expect(t, "unavailable", second(f.r.Resolve("ws", inc.ID, res, human("ops"))), RefusalPermissionUnavail)
	delete(f.perms.failing, "ops")
	f.perms.allow("ops", "k1")
	must(f.r.Resolve("ws", inc.ID, res, human("ops")))
	if must(f.r.Resolve("ws", inc.ID, res, human("ops"))).Resolver != "ops" {
		t.Fatal("resolve repeat")
	}
	expect(t, "other resolver", second(f.r.Resolve("ws", inc.ID, res, human("lead"))), RefusalResolved)
	expect(t, "other resolution", second(f.r.Resolve("ws", inc.ID, Resolution{Reason: ReasonFixed, Evidence: "e"}, human("ops"))), RefusalResolved)
	expect(t, "incidentof form", second(f.r.IncidentOf("", inc.ID, system("w"))), "invalid")
	expect(t, "incidentof id", second(f.r.IncidentOf("ws", "", system("w"))), "invalid")
	expect(t, "incidentof unknown", second(f.r.IncidentOf("ws9", inc.ID, system("w"))), "not_found")
	l := testLimits()
	l.Signals, l.Workspaces = 1, 1
	g := open(t, l)
	g.raise(t, good)
	expect(t, "signals", second(g.r.Raise("ws", signal("s2", "c", "k1", SeverityWarning, false, start), system("m"))), "limit")
	expect(t, "workspaces", second(g.r.Raise("ws2", good, system("m"))), "limit")
	l = testLimits()
	l.Incidents = 1
	h := open(t, l)
	h.raise(t, good)
	h.raise(t, signal("s2", "c", "k1", SeverityWarning, false, start))
	expect(t, "incidents", second(h.r.Raise("ws", signal("s3", "c", "k2", SeverityWarning, false, start), system("m"))), "limit")
	for _, r := range []string{ReasonFixed, ReasonSuperseded, ReasonOther} {
		k := open(t, testLimits())
		x := k.raise(t, good)
		must(k.r.Resolve("ws", x.ID, Resolution{Reason: r, Evidence: "e"}, system("auto")))
	}
}

func TestBlockedGates(t *testing.T) {
	f := open(t, testLimits())
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, true, start))
	f.raise(t, signal("s2", "d", "k1", SeverityWarning, true, start))
	f.raise(t, signal("s3", "e", "k1", SeverityWarning, false, start))
	expect(t, "form", second(f.r.Blocked("ws", "", system("w"))), "invalid")
	expect(t, "workspace", second(f.r.Blocked("", "k1", system("w"))), "invalid")
	expect(t, "actor", second(f.r.Blocked("ws", "k1", Actor{ID: "w"})), "invalid")
	expect(t, "assistant", second(f.r.Blocked("ws", "k1", assistant("bot"))), RefusalAINotAllowed)
	f.perms.deny("ops", "k1")
	expect(t, "not permitted", second(f.r.Blocked("ws", "k1", human("ops"))), RefusalNotPermitted)
	f.perms.failing["lead"] = true
	expect(t, "unavailable", second(f.r.Blocked("ws", "k1", human("lead"))), RefusalPermissionUnavail)
	got := must(f.r.Blocked("ws", "k1", human("ana")))
	if len(got) != 2 || got[0].ID != "inc-1" || got[1].ID != "inc-2" {
		t.Fatal(got)
	}
	if len(must(f.r.Blocked("ws9", "k1", system("w")))) != 0 {
		t.Fatal("empty workspace")
	}
}

func TestSettingsGates(t *testing.T) {
	f := open(t, testLimits())
	sub := Subscription{Recipient: "ana", Condition: "c", Minimum: SeverityInfo}
	for _, bad := range []Subscription{{Condition: "c", Minimum: SeverityInfo}, {Recipient: "ana", Minimum: SeverityInfo}, {Recipient: "ana", Condition: "c", Minimum: "x"}} {
		expect(t, "form", second(f.r.Subscribe("ws", bad, human("ana"))), "invalid")
	}
	expect(t, "workspace", second(f.r.Subscribe("", sub, human("ana"))), "invalid")
	expect(t, "actor", second(f.r.Subscribe("ws", sub, Actor{ID: "ana"})), "invalid")
	expect(t, "assistant", second(f.r.Subscribe("ws", sub, assistant("ana"))), RefusalAINotAllowed)
	expect(t, "system", second(f.r.Subscribe("ws", sub, system("ana"))), RefusalNotHuman)
	expect(t, "not self", second(f.r.Subscribe("ws", sub, human("bob"))), RefusalNotSelf)
	must(f.r.Subscribe("ws", sub, human("ana")))
	sub.Minimum = SeverityCritical
	must(f.r.Subscribe("ws", sub, human("ana")))
	good := Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate}
	for _, change := range []func(*Settings){
		func(s *Settings) { s.Recipient = "" }, func(s *Settings) { s.Zone = "" }, func(s *Settings) { s.Zone = "Local" }, func(s *Settings) { s.Zone = strings.Repeat("Z", maxIDBytes+1) },
		func(s *Settings) { s.Digest = "weekly" }, func(s *Settings) { s.Alternate = "a b" }, func(s *Settings) { s.Away = true },
		func(s *Settings) { s.Quiet = &Quiet{Start: -1, End: 10} }, func(s *Settings) { s.Quiet = &Quiet{Start: minutesInDay, End: 10} },
		func(s *Settings) { s.Quiet = &Quiet{Start: 10, End: -1} }, func(s *Settings) { s.Quiet = &Quiet{Start: 10, End: minutesInDay} },
		func(s *Settings) { s.Quiet = &Quiet{Start: 10, End: 10} }, func(s *Settings) { s.Renotify = map[string]time.Duration{"x": time.Hour} },
		func(s *Settings) { s.Renotify = map[string]time.Duration{SeverityInfo: minRenotify - 1} },
		func(s *Settings) { s.Renotify = map[string]time.Duration{SeverityInfo: maxRenotify + 1} },
	} {
		s := good
		change(&s)
		expect(t, "settings form", second(f.r.Configure("ws", s, human("ana"))), "invalid")
	}
	expect(t, "settings workspace", second(f.r.Configure("", good, human("ana"))), "invalid")
	expect(t, "settings actor", second(f.r.Configure("ws", good, Actor{ID: "ana"})), "invalid")
	expect(t, "settings system", second(f.r.Configure("ws", good, system("ana"))), RefusalNotHuman)
	expect(t, "settings not self", second(f.r.Configure("ws", good, human("bob"))), RefusalNotSelf)
	self := good
	self.Alternate = "ana"
	expect(t, "alternate self", second(f.r.Configure("ws", self, human("ana"))), RefusalAlternateIsSelf)
	zone := good
	zone.Zone = "Mars/Base"
	expect(t, "zone", second(f.r.Configure("ws", zone, human("ana"))), RefusalZoneUnavailable)
	edges := good
	edges.Quiet = &Quiet{Start: 0, End: minutesInDay - 1}
	edges.Renotify = map[string]time.Duration{SeverityInfo: 0, SeverityWarning: minRenotify, SeverityCritical: maxRenotify}
	v := must(f.r.Configure("ws", edges, human("ana")))
	v.Quiet.Start = 99
	v.Renotify[SeverityInfo] = time.Hour
	edges.Quiet.End = 5
	if got := must(f.r.Configure("ws", good, human("ana"))); got.Quiet != nil {
		t.Fatal("settings replaced")
	}
	l := testLimits()
	l.Recipients, l.Workspaces = 1, 1
	g := open(t, l)
	g.subscribe(t, "ana", "c", SeverityInfo)
	g.subscribe(t, "ana", "d", SeverityInfo)
	must(g.r.Configure("ws", good, human("ana")))
	expect(t, "recipients by subscription", second(g.r.Subscribe("ws", Subscription{Recipient: "bob", Condition: "c", Minimum: SeverityInfo}, human("bob"))), "limit")
	bob := good
	bob.Recipient = "bob"
	expect(t, "recipients by settings", second(g.r.Configure("ws", bob, human("bob"))), "limit")
	expect(t, "workspaces", second(g.r.Subscribe("ws2", sub, human("ana"))), "limit")
	expect(t, "workspaces settings", second(g.r.Configure("ws2", good, human("ana"))), "limit")
	h := open(t, l)
	must(h.r.Configure("ws", good, human("ana")))
	h.subscribe(t, "ana", "c", SeverityInfo)
}

func TestSnoozeGates(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	inc := f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	until := start.Add(time.Hour)
	expect(t, "form", second(f.r.Snooze("ws", inc.ID, time.Time{}, human("ana"))), "invalid")
	expect(t, "workspace", second(f.r.Snooze("", inc.ID, until, human("ana"))), "invalid")
	expect(t, "id", second(f.r.Snooze("ws", "", until, human("ana"))), "invalid")
	expect(t, "actor", second(f.r.Snooze("ws", inc.ID, until, Actor{ID: "ana"})), "invalid")
	expect(t, "system", second(f.r.Snooze("ws", inc.ID, until, system("ana"))), RefusalNotHuman)
	expect(t, "unknown", second(f.r.Snooze("ws", "inc-9", until, human("ana"))), "not_found")
	expect(t, "unknown workspace", second(f.r.Snooze("ws9", inc.ID, until, human("ana"))), "not_found")
	expect(t, "not subscribed", second(f.r.Snooze("ws", inc.ID, until, human("bob"))), RefusalNotSubscribed)
	f.perms.deny("ana", "k1")
	expect(t, "not permitted", second(f.r.Snooze("ws", inc.ID, until, human("ana"))), RefusalNotPermitted)
	f.perms.allow("ana", "k1")
	f.perms.failing["ana"] = true
	expect(t, "unavailable", second(f.r.Snooze("ws", inc.ID, until, human("ana"))), RefusalPermissionUnavail)
	delete(f.perms.failing, "ana")
	expect(t, "past", second(f.r.Snooze("ws", inc.ID, start, human("ana"))), RefusalSnoozeOutOfRange)
	expect(t, "too far", second(f.r.Snooze("ws", inc.ID, start.Add(maxSnooze+time.Second), human("ana"))), RefusalSnoozeOutOfRange)
	if got := must(f.r.Snooze("ws", inc.ID, start.Add(maxSnooze), human("ana"))); !got.Equal(start.Add(maxSnooze)) {
		t.Fatal(got)
	}
	must(f.r.Snooze("ws", inc.ID, until, human("ana")))
	f.clock.advance(time.Hour)
	if len(f.cycle(t)) != 1 {
		t.Fatal("a repeat replaces the snooze")
	}
	must(f.r.Resolve("ws", inc.ID, Resolution{Reason: ReasonFixed, Evidence: "e"}, system("auto")))
	expect(t, "resolved", second(f.r.Snooze("ws", inc.ID, f.clock.read().Add(time.Hour), human("ana"))), RefusalResolved)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	other := f.raise(t, signal("s2", "d", "k1", SeverityWarning, false, f.clock.read()))
	expect(t, "alternate of a recipient without that condition", second(f.r.Snooze("ws", other.ID, f.clock.read().Add(time.Hour), human("bob"))), RefusalNotSubscribed)
}

func TestPlanDispatchAndOpenGates(t *testing.T) {
	f := open(t, testLimits())
	expect(t, "plan form", second(f.r.Plan("", system("p"))), "invalid")
	expect(t, "plan actor", second(f.r.Plan("ws", Actor{ID: "p"})), "invalid")
	expect(t, "plan human", second(f.r.Plan("ws", human("p"))), RefusalNotSystem)
	if got := must(f.r.Plan("ws9", system("p"))); got != nil {
		t.Fatal("empty workspace")
	}
	f.subscribe(t, "ana", "c", SeverityWarning)
	f.subscribe(t, "bob", "c", SeverityInfo)
	f.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	f.raise(t, signal("s2", "x", "k1", SeverityCritical, false, start))
	planned := must(f.r.Plan("ws", system("p")))
	if len(planned) != 1 || planned[0].Target != "bob" {
		t.Fatalf("minimum severity and condition: %+v", planned)
	}
	expect(t, "dispatch form", second(f.r.Dispatch("ws", "", system("s"))), "invalid")
	expect(t, "dispatch workspace", second(f.r.Dispatch("", planned[0].ID, system("s"))), "invalid")
	expect(t, "dispatch actor", second(f.r.Dispatch("ws", planned[0].ID, Actor{ID: "s"})), "invalid")
	expect(t, "dispatch human", second(f.r.Dispatch("ws", planned[0].ID, human("s"))), RefusalNotSystem)
	expect(t, "dispatch unknown", second(f.r.Dispatch("ws", "note-9", system("s"))), "not_found")
	expect(t, "dispatch unknown workspace", second(f.r.Dispatch("ws9", planned[0].ID, system("s"))), "not_found")
	expect(t, "open before handoff", second(f.r.Open("ws", planned[0].ID, "inc-1", human("bob"))), RefusalNotTarget)
	f.perms.failing["bob"] = true
	n := must(f.r.Dispatch("ws", planned[0].ID, system("s")))
	if len(n.Delivered) != 0 {
		t.Fatal("unavailable at dispatch")
	}
	delete(f.perms.failing, "bob")
	if w := must(f.r.Withholdings("ws", system("w"))); len(w) != 1 || w[0].Reason != WithheldUnavailable {
		t.Fatal(w)
	}
	f.perms.failing["bob"] = true
	if got := must(f.r.Plan("ws", system("p"))); len(got) != 0 {
		t.Fatal("unavailable at planning")
	}
	delete(f.perms.failing, "bob")
	again := must(f.r.Plan("ws", system("p")))
	inc := must(f.r.IncidentOf("ws", "inc-1", system("w")))
	must(f.r.Resolve("ws", inc.ID, Resolution{Reason: ReasonFixed, Evidence: "e"}, system("auto")))
	if d := must(f.r.Dispatch("ws", again[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("closed at dispatch")
	}
	w := must(f.r.Withholdings("ws", system("w")))
	if w[len(w)-1].Reason != WithheldClosed {
		t.Fatal(w)
	}
	expect(t, "open form", second(f.r.Open("", "note-1", "inc-1", human("bob"))), "invalid")
	expect(t, "open note", second(f.r.Open("ws", "", "inc-1", human("bob"))), "invalid")
	expect(t, "open incident", second(f.r.Open("ws", "note-1", "", human("bob"))), "invalid")
	expect(t, "open actor", second(f.r.Open("ws", "note-1", "inc-1", Actor{ID: "bob"})), "invalid")
	expect(t, "open system", second(f.r.Open("ws", "note-1", "inc-1", system("bob"))), RefusalNotHuman)
	expect(t, "open workspace", second(f.r.Open("ws9", "note-1", "inc-1", human("bob"))), RefusalNotTarget)
	expect(t, "withholdings form", second(f.r.Withholdings("", system("w"))), "invalid")
	if got := must(f.r.Withholdings("ws9", system("w"))); got != nil {
		t.Fatal("empty withholdings")
	}
}

func TestDispatchHoldsSnoozedAndQuietIncidents(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	inc := f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(f.r.Plan("ws", system("p")))
	must(f.r.Snooze("ws", inc.ID, start.Add(time.Hour), human("ana")))
	if d := must(f.r.Dispatch("ws", p[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("snoozed at dispatch")
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	g.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	q := must(g.r.Plan("ws", system("p")))
	must(g.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Quiet: &Quiet{Start: 11 * 60, End: 13 * 60}}, human("ana")))
	if d := must(g.r.Dispatch("ws", q[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("quiet at dispatch")
	}
	if w := must(g.r.Withholdings("ws", system("w"))); len(w) != 0 {
		t.Fatal("quiet hours defer without a withholding")
	}
	h := open(t, testLimits())
	h.subscribe(t, "ana", "c", SeverityInfo)
	h.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	r := must(h.r.Plan("ws", system("p")))
	h.perms.deny("ana", "k1")
	must(h.r.Dispatch("ws", r[0].ID, system("s")))
	h.perms.allow("ana", "k1")
	h.clock.advance(2 * time.Hour)
	must(h.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Renotify: map[string]time.Duration{SeverityWarning: time.Hour}}, human("ana")))
	if got := h.cycle(t); len(got) != 1 || len(got[0].Delivered) != 1 {
		t.Fatal("after a refusal clears, the incident is due again")
	}
	h.clock.advance(time.Hour)
	if got := h.cycle(t); len(got) != 1 {
		t.Fatal("re-notification after the interval")
	}
}

func TestNotificationLimitIsAtomic(t *testing.T) {
	l := testLimits()
	l.Notifications = 1
	f := open(t, l)
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.subscribe(t, "bob", "c", SeverityInfo)
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	expect(t, "plan limit", second(f.r.Plan("ws", system("p"))), "limit")
	if w := must(f.r.Withholdings("ws", system("w"))); len(w) != 0 {
		t.Fatal("nothing committed")
	}
	g := open(t, l)
	g.subscribe(t, "ana", "c", SeverityInfo)
	g.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(g.r.Plan("ws", system("p")))
	g.perms.deny("ana", "k1")
	expect(t, "dispatch limit", second(g.r.Dispatch("ws", p[0].ID, system("s"))), "limit")
}

func TestMutationDrivenEdges(t *testing.T) {
	f := open(t, testLimits())
	for index := 0; index < 12; index++ {
		f.raise(t, signal("b"+strconv.Itoa(index), "c"+strconv.Itoa(index), "k1", SeverityWarning, true, start))
		if index == 5 {
			f.clock.advance(time.Second)
		}
	}
	blocked := must(f.r.Blocked("ws", "k1", system("w")))
	for index := 1; index < len(blocked); index++ {
		a, b := blocked[index-1], blocked[index]
		if b.Opened.Before(a.Opened) || a.Opened.Equal(b.Opened) && b.ID < a.ID {
			t.Fatalf("blocked order: %+v", blocked)
		}
	}
	l := testLimits()
	l.Recipients = 2
	g := open(t, l)
	for _, c := range []string{"c", "d", "e"} {
		g.subscribe(t, "ana", c, SeverityInfo)
	}
	g.subscribe(t, "bob", "c", SeverityInfo)
	must(g.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Quiet: &Quiet{Start: 60, End: 0}}, human("ana")))
	zones := open(t, testLimits())
	zones.r.config.Zones = func(string) (*time.Location, error) { return nil, nil }
	expect(t, "nil zone", second(zones.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate}, human("ana"))), RefusalZoneUnavailable)
	one := Settings{Quiet: &Quiet{Start: 60, End: 61}}
	if !quiet(one, time.UTC, time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)) || quiet(one, time.UTC, time.Date(2026, 1, 1, 1, 1, 0, 0, time.UTC)) {
		t.Fatal("one-minute window")
	}
	wrap := Settings{Quiet: &Quiet{Start: 22 * 60, End: 7 * 60}}
	if !quiet(wrap, time.UTC, time.Date(2026, 1, 1, 22, 0, 0, 0, time.UTC)) || quiet(wrap, time.UTC, time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)) || quiet(wrap, time.UTC, time.Date(2026, 1, 1, 21, 59, 0, 0, time.UTC)) {
		t.Fatal("window across midnight edges")
	}
}

func TestForwardedCopiesAndDispatchEdges(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	inc := f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(f.r.Plan("ws", system("p")))
	if len(p) != 1 || p[0].ID != "note-1" || p[0].Target != "bob" {
		t.Fatalf("forwarded only: %+v", p)
	}
	must(f.r.Snooze("ws", inc.ID, start.Add(time.Hour), human("bob")))
	if d := must(f.r.Dispatch("ws", p[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("the alternate's own snooze holds its forwarded copy at dispatch")
	}
	if w := must(f.r.Withholdings("ws", system("w"))); len(w) != 1 || w[0].Reason != WithheldSnoozed {
		t.Fatal(w)
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	g.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	g.perms.failing["ana"] = true
	must(g.r.Plan("ws", system("p")))
	if w := must(g.r.Withholdings("ws", system("w"))); len(w) != 1 || w[0].Reason != WithheldUnavailable {
		t.Fatal(w)
	}
	h := open(t, testLimits())
	h.subscribe(t, "ana", "c", SeverityInfo)
	must(h.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Renotify: map[string]time.Duration{SeverityWarning: 10 * time.Minute}}, human("ana")))
	h.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p1 := must(h.r.Plan("ws", system("p")))
	h.clock.advance(time.Hour)
	p2 := must(h.r.Plan("ws", system("p")))
	must(h.r.Dispatch("ws", p2[0].ID, system("s")))
	h.clock.advance(10 * time.Minute)
	if d := must(h.r.Dispatch("ws", p1[0].ID, system("s"))); len(d.Delivered) != 1 {
		t.Fatal("the older notification goes out once the interval has passed")
	}
	k := open(t, testLimits())
	k.subscribe(t, "ana", "c", SeverityInfo)
	k.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	q1 := must(k.r.Plan("ws", system("p")))
	k.raise(t, signal("s2", "c", "k1", SeverityCritical, false, start))
	must(k.r.Plan("ws", system("p")))
	must(k.r.Dispatch("ws", q1[0].ID, system("s")))
	if again := must(k.r.Plan("ws", system("p"))); len(again) != 0 {
		t.Fatal("dispatching one notification keeps another's pending state")
	}
	l := testLimits()
	l.Notifications = 2
	m := open(t, l)
	m.subscribe(t, "ana", "c", SeverityInfo)
	m.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	r := must(m.r.Plan("ws", system("p")))
	m.perms.deny("ana", "k1")
	if d := must(m.r.Dispatch("ws", r[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("exactly at the limit")
	}
}
