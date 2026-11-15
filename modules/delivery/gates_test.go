package delivery

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRegistryNeedsDependenciesAndLimits(t *testing.T) {
	f := open(t, testLimits())
	for _, c := range []Config{{Random: f.random}, {Packages: f.pkgs}} {
		if _, err := NewRegistry(c, testLimits(), f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatal("config")
		}
	}
	if _, err := NewRegistry(Config{Packages: f.pkgs, Random: f.random}, testLimits(), nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("clock")
	}
	for _, change := range []func(*Limits){
		func(l *Limits) { l.Workspaces = 0 }, func(l *Limits) { l.Channels = maxChannels + 1 }, func(l *Limits) { l.Deliveries = 0 },
		func(l *Limits) { l.Queue = 0 }, func(l *Limits) { l.Attempts = maxAttempts + 1 }, func(l *Limits) { l.Lease = minLease - 1 },
		func(l *Limits) { l.Lease = maxLease + 1 },
	} {
		l := testLimits()
		change(&l)
		if _, err := NewRegistry(Config{Packages: f.pkgs, Random: f.random}, l, f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatalf("limits %+v", l)
		}
	}
	for _, l := range []Limits{{maxWorkspaces, maxChannels, maxDeliveries, maxQueue, maxAttempts, maxLease}, {1, 1, 1, 1, 1, minLease}} {
		if _, err := NewRegistry(Config{Packages: f.pkgs, Random: f.random}, l, f.clock.read); err != nil {
			t.Fatal(err)
		}
	}
	if (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" || !errors.Is(Refusal{Reason: "x"}, ErrRejected) {
		t.Fatal("refusal")
	}
	if !validID(strings.Repeat("Z", maxIDBytes)) || validID(strings.Repeat("z", maxIDBytes+1)) || validID("a b") || validID("") || !validID("Az09._:-") {
		t.Fatal("identifiers")
	}
}

func TestChannelGates(t *testing.T) {
	f := open(t, testLimits())
	c := Channel{ID: "mail", Kind: ChannelMail, Recipient: "acme", Target: "box"}
	for _, bad := range []Channel{{Kind: ChannelMail, Recipient: "acme", Target: "box"}, {ID: "m", Kind: "fax", Recipient: "acme", Target: "box"},
		{ID: "m", Kind: ChannelMail, Target: "box"}, {ID: "m", Kind: ChannelMail, Recipient: "acme"}} {
		expect(t, "form", second(f.r.Register("ws", bad, human("ops"))), "invalid")
	}
	expect(t, "workspace", second(f.r.Register("", c, human("ops"))), "invalid")
	expect(t, "actor", second(f.r.Register("ws", c, Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.Register("ws", c, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Register("ws", c, system("job"))), ReasonNotHuman)
	expect(t, "recipient", second(f.r.Register("ws", c, human("acme"))), ReasonRecipientActor)
	v := must(f.r.Register("ws", c, human("ops")))
	if must(f.r.Register("ws", c, human("ops"))).Registrar != v.Registrar {
		t.Fatal("repeat")
	}
	expect(t, "other registrar", second(f.r.Register("ws", c, human("other"))), "conflict")
	moved := c
	moved.Target = "box2"
	expect(t, "target change", second(f.r.Register("ws", moved, human("ops"))), "conflict")
	l := testLimits()
	l.Channels, l.Workspaces = 1, 1
	g := open(t, l)
	must(g.r.Register("ws", c, human("ops")))
	expect(t, "channels", second(g.r.Register("ws", Channel{ID: "b", Kind: ChannelMail, Recipient: "acme", Target: "box"}, human("ops"))), "limit")
	expect(t, "workspaces", second(g.r.Register("ws2", c, human("ops"))), "limit")

	d := proven()
	expect(t, "diagnose form", second(f.r.Diagnose("ws", "mail", Diagnosis{Evidence: Evidence{Kind: EvidenceChannelTest}}, human("tester"))), "invalid")
	expect(t, "diagnose kind", second(f.r.Diagnose("ws", "mail", Diagnosis{Evidence: Evidence{Reference: "r"}}, human("tester"))), "invalid")
	expect(t, "diagnose workspace", second(f.r.Diagnose("", "mail", d, human("tester"))), "invalid")
	expect(t, "diagnose id", second(f.r.Diagnose("ws", "", d, human("tester"))), "invalid")
	expect(t, "diagnose actor", second(f.r.Diagnose("ws", "mail", d, Actor{ID: "x"})), "invalid")
	expect(t, "diagnose recipient", second(f.r.Diagnose("ws", "mail", d, human("acme"))), ReasonRecipientActor)
	expect(t, "diagnose unknown", second(f.r.Diagnose("ws", "nope", d, human("tester"))), "not_found")
	expect(t, "diagnose workspace unknown", second(f.r.Diagnose("ws9", "mail", d, human("tester"))), "not_found")
	expect(t, "open before diagnosis", second(f.r.Open("ws", "mail", 1, human("lead"))), ReasonStaleDiagnosis)
	first := must(f.r.Diagnose("ws", "mail", d, human("tester")))
	if must(f.r.Diagnose("ws", "mail", d, human("tester"))).Revision != first.Revision {
		t.Fatal("diagnosis repeat")
	}
	if must(f.r.Diagnose("ws", "mail", d, human("tester2"))).Revision != 2 {
		t.Fatal("another diagnoser makes a revision")
	}
	expect(t, "open form", second(f.r.Open("ws", "mail", 0, human("lead"))), "invalid")
	expect(t, "open workspace", second(f.r.Open("", "mail", 2, human("lead"))), "invalid")
	expect(t, "open id", second(f.r.Open("ws", "", 2, human("lead"))), "invalid")
	expect(t, "open actor", second(f.r.Open("ws", "mail", 2, Actor{ID: "x"})), "invalid")
	expect(t, "open recipient", second(f.r.Open("ws", "mail", 2, human("beta"))), ReasonRecipientActor)
	expect(t, "open unknown", second(f.r.Open("ws", "nope", 2, human("lead"))), "not_found")
	expect(t, "open future", second(f.r.Open("ws", "mail", 3, human("lead"))), ReasonStaleDiagnosis)
	v = must(f.r.Open("ws", "mail", 2, human("lead")))
	if must(f.r.Open("ws", "mail", 2, human("lead"))).Opener != "lead" || !v.Open {
		t.Fatal("open repeat")
	}
	expect(t, "other opener", second(f.r.Open("ws", "mail", 2, human("boss"))), "conflict")
	expect(t, "close form", second(f.r.Close("ws", "", human("lead"))), "invalid")
	expect(t, "close workspace", second(f.r.Close("", "mail", human("lead"))), "invalid")
	expect(t, "close actor", second(f.r.Close("ws", "mail", Actor{ID: "x"})), "invalid")
	expect(t, "close system", second(f.r.Close("ws", "mail", system("job"))), ReasonNotHuman)
	expect(t, "close unknown", second(f.r.Close("ws", "nope", human("lead"))), "not_found")
	must(f.r.Close("ws", "mail", human("lead")))
	if must(f.r.Close("ws", "mail", human("lead"))).Open {
		t.Fatal("close repeat")
	}
	expect(t, "other closer", second(f.r.Close("ws", "mail", human("boss"))), "conflict")
	if must(f.r.Open("ws", "mail", 2, human("lead"))).Closer != "" {
		t.Fatal("reopening clears the closer")
	}
	expect(t, "channelof form", second(f.r.ChannelOf("", "mail")), "invalid")
	expect(t, "channelof id", second(f.r.ChannelOf("ws", "")), "invalid")
	expect(t, "channelof unknown", second(f.r.ChannelOf("ws", "nope")), "not_found")
	got := must(f.r.ChannelOf("ws", "mail"))
	got.Diagnoses[0].Author = "mutated"
	if must(f.r.ChannelOf("ws", "mail")).Diagnoses[0].Author != "tester" {
		t.Fatal("a view shares diagnoses")
	}
}

func TestRequestGates(t *testing.T) {
	f := standard(t)
	ok := Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}
	for _, bad := range []Delivery{{Package: "p1", Channel: "mail-acme"}, {ID: "d", Channel: "mail-acme"}, {ID: "d", Package: "p1"}} {
		expect(t, "form", second(f.r.Request("ws", bad, system("s"))), "invalid")
	}
	expect(t, "workspace", second(f.r.Request("", ok, system("s"))), "invalid")
	expect(t, "actor", second(f.r.Request("ws", ok, Actor{ID: "s"})), "invalid")
	expect(t, "assistant", second(f.r.Request("ws", ok, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "recipient", second(f.r.Request("ws", ok, human("acme"))), ReasonRecipientActor)
	expect(t, "unknown channel", second(f.r.Request("ws", Delivery{ID: "d", Package: "p1", Channel: "nope"}, system("s"))), "not_found")
	expect(t, "unknown package", second(f.r.Request("ws", Delivery{ID: "d", Package: "nope", Channel: "mail-acme"}, system("s"))), ReasonUnavailable)
	must(f.r.Request("ws", ok, system("s")))
	if must(f.r.Request("ws", ok, system("s"))).Duplicate {
		t.Fatal("an identical repeat is not a duplicate")
	}
	expect(t, "other requester", second(f.r.Request("ws", ok, system("t"))), "conflict")
	expect(t, "other content", second(f.r.Request("ws", Delivery{ID: "d1", Package: "p2", Channel: "mail-acme"}, system("s"))), "conflict")
	l := testLimits()
	l.Queue = 1
	g := open(t, l)
	must(g.r.Register("ws", Channel{ID: "m", Kind: ChannelMail, Recipient: "acme", Target: "t"}, human("ops")))
	g.pkgs.put("ws", pkg("a", "acme"))
	g.pkgs.put("ws", pkg("b", "acme"))
	must(g.r.Request("ws", Delivery{ID: "x", Package: "a", Channel: "m"}, system("s")))
	expect(t, "queue", second(g.r.Request("ws", Delivery{ID: "y", Package: "b", Channel: "m"}, system("s"))), "limit")
	must(g.r.Cancel("ws", "x", human("ops")))
	must(g.r.Request("ws", Delivery{ID: "y", Package: "b", Channel: "m"}, system("s")))
	l.Queue, l.Deliveries = 10, 1
	h := open(t, l)
	must(h.r.Register("ws", Channel{ID: "m", Kind: ChannelMail, Recipient: "acme", Target: "t"}, human("ops")))
	h.pkgs.put("ws", pkg("a", "acme"))
	h.pkgs.put("ws", pkg("b", "acme"))
	must(h.r.Request("ws", Delivery{ID: "x", Package: "a", Channel: "m"}, system("s")))
	expect(t, "deliveries", second(h.r.Request("ws", Delivery{ID: "y", Package: "b", Channel: "m"}, system("s"))), "limit")
	expect(t, "of form", second(f.r.Of("", "d1")), "invalid")
	expect(t, "of id", second(f.r.Of("ws", "")), "invalid")
	expect(t, "of unknown", second(f.r.Of("ws", "nope")), "not_found")
	expect(t, "of workspace", second(f.r.Of("ws9", "d1")), "not_found")
}

func TestApproveAndCancelGates(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p2", Channel: "portal-acme"}, human("requester")))
	expect(t, "form", second(f.r.Approve("ws", "", human("lead"))), "invalid")
	expect(t, "workspace", second(f.r.Approve("", "d1", human("lead"))), "invalid")
	expect(t, "actor", second(f.r.Approve("ws", "d1", Actor{ID: "x"})), "invalid")
	expect(t, "system", second(f.r.Approve("ws", "d1", system("job"))), ReasonNotHuman)
	expect(t, "recipient", second(f.r.Approve("ws", "d1", human("acme"))), ReasonRecipientActor)
	expect(t, "unknown", second(f.r.Approve("ws", "nope", human("lead"))), "not_found")
	f.pkgs.lapse("ws", "p2", "recipient_revoked")
	expect(t, "lapsed", second(f.r.Approve("ws", "d1", human("lead"))), ReasonPackageNotLive)
	f.pkgs.fail = true
	expect(t, "packages down", second(f.r.Approve("ws", "d1", human("lead"))), ReasonUnavailable)
	f.pkgs.fail = false
	v := pkg("p2", "beta")
	f.pkgs.put("ws", v)
	expect(t, "recipient changed", second(f.r.Approve("ws", "d1", human("lead"))), ReasonRecipientMismatch)
	f.pkgs.put("ws", pkg("p2", "acme"))
	must(f.r.Approve("ws", "d1", human("lead")))
	if must(f.r.Approve("ws", "d1", human("lead"))).Approver != "lead" {
		t.Fatal("repeat")
	}
	expect(t, "second approver", second(f.r.Approve("ws", "d1", human("boss"))), ReasonAlreadyApproved)
	expect(t, "cancel form", second(f.r.Cancel("ws", "", human("ops"))), "invalid")
	expect(t, "cancel workspace", second(f.r.Cancel("", "d1", human("ops"))), "invalid")
	expect(t, "cancel actor", second(f.r.Cancel("ws", "d1", Actor{ID: "x"})), "invalid")
	expect(t, "cancel assistant", second(f.r.Cancel("ws", "d1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "cancel unknown", second(f.r.Cancel("ws", "nope", human("ops"))), "not_found")
	s := must(f.r.Begin("ws", "d1", system("worker")))
	must(f.r.Cancel("ws", "d1", human("ops")))
	if must(f.r.Cancel("ws", "d1", human("ops"))).State != StateCancelled {
		t.Fatal("cancel repeat")
	}
	expect(t, "other canceller", second(f.r.Cancel("ws", "d1", human("boss"))), "conflict")
	expect(t, "approve cancelled", second(f.r.Approve("ws", "d1", human("boss"))), ReasonAlreadyApproved)
	late := must(f.r.Report("ws", "d1", s.Token, Outcome{Result: ResultRejected, Reference: "x"}, system("worker")))
	if late.State != StateCancelled || !late.Attempts[0].Late {
		t.Fatalf("late rejection after cancel: %+v", late)
	}
	must(f.r.Request("ws", Delivery{ID: "d2", Package: "p2", Channel: "portal-acme"}, human("requester")))
	must(f.r.Cancel("ws", "d2", human("ops")))
	expect(t, "approve not waiting", second(f.r.Approve("ws", "d2", human("lead"))), ReasonNotWaiting)
	must(f.r.Request("ws", Delivery{ID: "d3", Package: "p1", Channel: "mail-acme"}, system("s")))
	a := must(f.r.Begin("ws", "d3", system("worker")))
	must(f.r.Report("ws", "d3", a.Token, Outcome{Result: ResultAccepted, Reference: "ok"}, system("worker")))
	expect(t, "cancel accepted", second(f.r.Cancel("ws", "d3", human("ops"))), ReasonFinal)
}

func TestBeginGates(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("s")))
	expect(t, "form", second(f.r.Begin("ws", "", system("w"))), "invalid")
	expect(t, "workspace", second(f.r.Begin("", "d1", system("w"))), "invalid")
	expect(t, "actor", second(f.r.Begin("ws", "d1", Actor{ID: "w"})), "invalid")
	expect(t, "assistant", second(f.r.Begin("ws", "d1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "recipient", second(f.r.Begin("ws", "d1", system("acme"))), ReasonRecipientActor)
	expect(t, "unknown", second(f.r.Begin("ws", "nope", system("w"))), "not_found")
	f.random.fail = true
	expect(t, "random", second(f.r.Begin("ws", "d1", system("w"))), ReasonRandomUnavailable)
	f.random.fail = false
	must(f.r.Request("ws", Delivery{ID: "d0", Package: "p2", Channel: "portal-acme"}, human("requester")))
	must(f.r.Approve("ws", "d0", human("lead")))
	must(f.r.Begin("ws", "d0", system("w")))
	f.random.stuck = true
	f.clock.advance(testLimits().Lease)
	expect(t, "token collision", second(f.r.Begin("ws", "d0", system("w"))), ReasonRandomUnavailable)
	f.random.stuck = false
	f.pkgs.fail = true
	expect(t, "packages down", second(f.r.Begin("ws", "d1", system("w"))), ReasonUnavailable)
	f.pkgs.fail = false
	must(f.r.Request("ws", Delivery{ID: "d2", Package: "p2", Channel: "mail-acme"}, system("s")))
	f.pkgs.put("ws", pkg("p2", "beta"))
	expect(t, "recipient moved", second(f.r.Begin("ws", "d2", system("w"))), ReasonPackageNotLive)
	if v := must(f.r.Of("ws", "d2")); v.State != StateStopped || v.Reason != ReasonRecipientMismatch {
		t.Fatalf("stopped on recipient: %+v", v)
	}
	expect(t, "stopped", second(f.r.Begin("ws", "d2", system("w"))), ReasonNotWaiting)
	a := must(f.r.Begin("ws", "d1", system("w")))
	f.clock.advance(testLimits().Lease - time.Second)
	expect(t, "lease not expired", second(f.r.Begin("ws", "d1", system("w"))), ReasonAttemptRunning)
	f.clock.advance(time.Second)
	b := must(f.r.Begin("ws", "d1", system("w")))
	if !b.PossibleDuplicate || b.Number != 2 {
		t.Fatal(b)
	}
	v := must(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultAccepted, Reference: "late"}, system("w")))
	if v.State != StateAccepted || !v.Attempts[0].Late || !v.Attempts[0].Superseded || !v.PossibleDuplicate {
		t.Fatalf("late accept with a newer attempt: %+v", v)
	}
	v = must(f.r.Report("ws", "d1", b.Token, Outcome{Result: ResultAccepted, Reference: "dup"}, system("w")))
	if v.State != StateAccepted || !v.Attempts[1].Late {
		t.Fatalf("second accept: %+v", v)
	}
	must(f.r.Confirm("ws", "d1", "receipt", system("w")))
	expect(t, "begin confirmed", second(f.r.Begin("ws", "d1", system("w"))), ReasonNotWaiting)
}

func TestReportAndConfirmGates(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("s")))
	must(f.r.Request("ws", Delivery{ID: "d2", Package: "p2", Channel: "mail-acme"}, system("s")))
	a := must(f.r.Begin("ws", "d1", system("w")))
	o := Outcome{Result: ResultRejected, Reference: "x"}
	expect(t, "form", second(f.r.Report("ws", "d1", "short", o, system("w"))), "invalid")
	expect(t, "workspace", second(f.r.Report("", "d1", a.Token, o, system("w"))), "invalid")
	expect(t, "id", second(f.r.Report("ws", "", a.Token, o, system("w"))), "invalid")
	expect(t, "result", second(f.r.Report("ws", "d1", a.Token, Outcome{Result: "maybe", Reference: "x"}, system("w"))), "invalid")
	expect(t, "reference", second(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultRejected}, system("w"))), "invalid")
	expect(t, "actor", second(f.r.Report("ws", "d1", a.Token, o, Actor{ID: "w"})), "invalid")
	expect(t, "assistant", second(f.r.Report("ws", "d1", a.Token, o, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "recipient", second(f.r.Report("ws", "d1", a.Token, o, human("acme"))), ReasonRecipientActor)
	expect(t, "unknown delivery", second(f.r.Report("ws", "nope", a.Token, o, system("w"))), "not_found")
	expect(t, "unknown token", second(f.r.Report("ws", "d1", strings.Repeat("0", 32), o, system("w"))), "not_found")
	expect(t, "token of another delivery", second(f.r.Report("ws", "d2", a.Token, o, system("w"))), "not_found")
	must(f.r.Report("ws", "d1", a.Token, o, system("w")))
	if must(f.r.Report("ws", "d1", a.Token, o, system("w"))).State != StateWaiting {
		t.Fatal("repeat")
	}
	expect(t, "other result", second(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultAccepted, Reference: "x"}, system("w"))), "conflict")
	expect(t, "other reference", second(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultRejected, Reference: "y"}, system("w"))), "conflict")
	expect(t, "other reporter", second(f.r.Report("ws", "d1", a.Token, o, system("v"))), "conflict")
	expect(t, "confirm waiting", second(f.r.Confirm("ws", "d1", "r", system("w"))), ReasonNotAccepted)
	expect(t, "confirm form", second(f.r.Confirm("ws", "d1", "", system("w"))), "invalid")
	expect(t, "confirm workspace", second(f.r.Confirm("", "d1", "r", system("w"))), "invalid")
	expect(t, "confirm id", second(f.r.Confirm("ws", "", "r", system("w"))), "invalid")
	expect(t, "confirm actor", second(f.r.Confirm("ws", "d1", "r", Actor{ID: "w"})), "invalid")
	expect(t, "confirm recipient", second(f.r.Confirm("ws", "d1", "r", human("acme"))), ReasonRecipientActor)
	expect(t, "confirm unknown", second(f.r.Confirm("ws", "nope", "r", system("w"))), "not_found")
	b := must(f.r.Begin("ws", "d1", system("w")))
	must(f.r.Report("ws", "d1", b.Token, Outcome{Result: ResultAccepted, Reference: "ok"}, system("w")))
	must(f.r.Confirm("ws", "d1", "r", system("w")))
	if must(f.r.Confirm("ws", "d1", "r", system("w"))).Receipt != "r" {
		t.Fatal("confirm repeat")
	}
	expect(t, "other receipt", second(f.r.Confirm("ws", "d1", "s", system("w"))), "conflict")
	expect(t, "other confirmer", second(f.r.Confirm("ws", "d1", "r", system("v"))), "conflict")
	if must(f.r.Of("ws", "d1")).Confirmer != "w" {
		t.Fatal("confirmer")
	}
	c := must(f.r.Begin("ws", "d2", system("w")))
	f.clock.advance(testLimits().Lease)
	must(f.r.Begin("ws", "d2", system("w")))
	must(f.r.Report("ws", "d2", c.Token, Outcome{Result: ResultUnknown, Reference: "late-unknown"}, system("w")))
	if v := must(f.r.Of("ws", "d2")); v.State != StateInFlight || !v.Attempts[0].Late {
		t.Fatalf("late unknown leaves the newer attempt running: %+v", v)
	}
}

func TestResendGates(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("s")))
	next := Delivery{ID: "d2"}
	expect(t, "form", second(f.r.Resend("ws", "d1", Delivery{}, "why", human("ops"))), "invalid")
	expect(t, "reason", second(f.r.Resend("ws", "d1", next, "", human("ops"))), "invalid")
	expect(t, "workspace", second(f.r.Resend("", "d1", next, "why", human("ops"))), "invalid")
	expect(t, "id", second(f.r.Resend("ws", "", next, "why", human("ops"))), "invalid")
	expect(t, "actor", second(f.r.Resend("ws", "d1", next, "why", Actor{ID: "x"})), "invalid")
	expect(t, "system", second(f.r.Resend("ws", "d1", next, "why", system("job"))), ReasonNotHuman)
	expect(t, "unknown", second(f.r.Resend("ws", "nope", next, "why", human("ops"))), "not_found")
	expect(t, "waiting", second(f.r.Resend("ws", "d1", next, "why", human("ops"))), ReasonNotResendable)
	must(f.r.Cancel("ws", "d1", human("ops")))
	f.pkgs.lapse("ws", "p1", "policy_suspended")
	expect(t, "lapsed", second(f.r.Resend("ws", "d1", next, "why", human("ops"))), ReasonPackageNotLive)
	f.pkgs.put("ws", pkg("p1", "beta"))
	expect(t, "recipient moved", second(f.r.Resend("ws", "d1", next, "why", human("ops"))), ReasonRecipientMismatch)
	f.pkgs.fail = true
	expect(t, "packages down", second(f.r.Resend("ws", "d1", next, "why", human("ops"))), ReasonUnavailable)
	f.pkgs.fail = false
	f.pkgs.put("ws", pkg("p1", "acme"))
	v := must(f.r.Resend("ws", "d1", next, "why", human("ops")))
	if must(f.r.Resend("ws", "d1", next, "why", human("ops"))).Delivery != v.Delivery {
		t.Fatal("repeat")
	}
	expect(t, "other reason", second(f.r.Resend("ws", "d1", next, "other", human("ops"))), "conflict")
	expect(t, "other actor", second(f.r.Resend("ws", "d1", next, "why", human("boss"))), "conflict")
}

func TestViewDetails(t *testing.T) {
	if validID("!a") || validID("a!") {
		t.Fatal("identifier characters")
	}
	f := standard(t)
	must(f.r.Diagnose("ws", "mail-acme", Diagnosis{FieldControl: true, AttachmentControl: true, Evidence: Evidence{Kind: EvidenceChannelTest, Reference: "again"}}, human("tester")))
	if c := must(f.r.ChannelOf("ws", "mail-acme")); c.Opened != 0 || c.Opener != "" || c.Open {
		t.Fatalf("a new diagnosis resets the opening: %+v", c)
	}
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, human("requester")))
	must(f.r.Approve("ws", "d1", human("lead")))
	a := must(f.r.Begin("ws", "d1", system("w")))
	must(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultRejected, Reference: "x"}, system("w")))
	must(f.r.Begin("ws", "d1", system("w")))
	v := must(f.r.Of("ws", "d1"))
	if v.Attempts[0].Number != 1 || v.Attempts[1].Number != 2 {
		t.Fatalf("attempt numbers: %+v", v.Attempts)
	}
	must(f.r.Cancel("ws", "d1", human("ops")))
	if f.r.workspaces["ws"].queued != 0 {
		t.Fatal("queue after cancel")
	}
	must(f.r.Report("ws", "d1", tokenOf(f, "d1", 1), Outcome{Result: ResultAccepted, Reference: "late"}, system("w")))
	if f.r.workspaces["ws"].queued != 0 {
		t.Fatalf("a late acceptance after cancel changed the queue: %d", f.r.workspaces["ws"].queued)
	}
}

func tokenOf(f fixture, id string, index int) string {
	for token, t := range f.r.workspaces["ws"].tokens {
		if t.delivery.view.Delivery.ID == id && t.index == index {
			return token
		}
	}
	return ""
}
