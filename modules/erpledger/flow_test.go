package erpledger

import (
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestCommandGates(t *testing.T) {
	f := newFixture(t)
	must(f.r.Prepare("w1", invoice("k1", "api", line("l1", "s1", "1")), human("b")))
	expect(t, "form", second(f.r.Command("w1", "k1", "delete", human("b"))), "invalid")
	for _, name := range []string{CommandConnect, CommandMapping, CommandPrepare} {
		expect(t, name, second(f.r.Command("w1", "k1", name, human("b"))), "invalid")
	}
	expect(t, "ws", second(f.r.Command("", "k1", CommandPost, human("b"))), "invalid")
	expect(t, "key", second(f.r.Command("w1", "", CommandPost, human("b"))), "invalid")
	expect(t, "actor", second(f.r.Command("w1", "k1", CommandPost, human(""))), "invalid")
	expect(t, "missing", second(f.r.Command("w1", "nope", CommandPost, human("b"))), "not_found")
	expect(t, "missing ws", second(f.r.Command("w9", "k1", CommandPost, human("b"))), "not_found")
	expect(t, "assistant", second(f.r.Command("w1", "k1", CommandSubmit, assistant("bot"))), ReasonAINotAllowed)
	for _, name := range []string{CommandPost, CommandCorrect, CommandStatus, CommandReadDraft, CommandBalance} {
		expect(t, "prepared "+name, second(f.r.Command("w1", "k1", name, human("b"))), ReasonWrongState)
	}
	v := must(f.r.Command("w1", "k1", CommandSubmit, human("b")))
	if v.State != StateSent || v.Watch == nil || !v.Watch.Next.Equal(start.Add(time.Hour)) || v.Watch.Owner != "erp-ops" {
		t.Fatalf("sent %+v", v)
	}
	expect(t, "submit twice", second(f.r.Command("w1", "k1", CommandSubmit, human("b"))), ReasonWrongState)
	expect(t, "post sent", second(f.r.Command("w1", "k1", CommandPost, human("b"))), ReasonWrongState)
	for _, name := range []string{CommandStatus, CommandReadDraft, CommandBalance} {
		must(f.r.Command("w1", "k1", name, human("b")))
	}
	f.observe(t, "api", "k1", "D-1", 1, StateDraft, SourceWebhook)
	expect(t, "correct draft", second(f.r.Command("w1", "k1", CommandCorrect, human("b"))), ReasonWrongState)
	must(f.r.Command("w1", "k1", CommandPost, human("b")))
	f.observe(t, "api", "k1", "D-1", 2, StatePosted, SourceWebhook)
	got := must(f.r.Command("w1", "k1", CommandCorrect, human("b")))
	if len(got.Commands) != 6 || got.Commands[5].Name != CommandCorrect {
		t.Fatalf("commands %+v", got.Commands)
	}
	got.Commands[0].Name = "x"
	got.Watch.Owner = "x"
	if v := must(f.r.SubmissionOf("w1", "k1")); v.Commands[0].Name != CommandSubmit || v.Watch.Owner != "erp-ops" {
		t.Fatal("aliasing")
	}
	file := fileConnection("file")
	file.Capabilities = append(file.Capabilities, CommandPost)
	f.connect(t, file)
	must(f.r.VerifyTemplate("w1", "file", "e", human("qa")))
	f.send(t, invoice("k2", "file", line("l1", "s2", "1")))
	must(f.r.Prepare("w1", invoice("k3", "api", line("l1", "s3", "1")), human("b")))
	must(f.r.Anchor("w1", Anchor{ID: "c", Connection: "api", Kind: AnchorContract, External: "C", Customer: "acme", Sources: []string{"s3"}}, human("a")))
	expect(t, "diverged submit", second(f.r.Command("w1", "k3", CommandSubmit, human("b"))), ReasonDiverged)
	expect(t, "diverged status", second(f.r.Command("w1", "k3", CommandStatus, human("b"))), ReasonWrongState)
}

func TestObservationGates(t *testing.T) {
	f := newFixture(t)
	f.connect(t, apiConnection("api2"))
	f.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	f.send(t, invoice("k2", "api", line("l1", "s2", "1")))
	must(f.r.Prepare("w1", invoice("k3", "api", line("l1", "s3", "1")), human("b")))
	good := Observation{ID: "o", Connection: "api", Key: "k1", Revision: 1, State: StateUnknown, Source: SourcePoll}
	bad := []func(*Observation){
		func(o *Observation) { o.ID = "" }, func(o *Observation) { o.Connection = "" }, func(o *Observation) { o.Key = "" },
		func(o *Observation) { o.Key = "a b" }, func(o *Observation) { o.External = "a b" }, func(o *Observation) { o.Revision = 0 },
		func(o *Observation) { o.State = StateSent }, func(o *Observation) { o.Source = "email" },
		func(o *Observation) { o.State, o.Source = StateAbsent, SourceWebhook }, func(o *Observation) { o.State = StateDraft },
		func(o *Observation) { o.State = StatePosted },
	}
	for i, change := range bad {
		o := good
		change(&o)
		if got := kind(second(f.r.Observe("w1", o, system("c")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Observe("", good, system("c"))), "invalid")
	expect(t, "actor", second(f.r.Observe("w1", good, system(""))), "invalid")
	expect(t, "unknown connection", second(f.r.Observe("w1", Observation{ID: "o", Connection: "nope", Key: "k1", Revision: 1, State: StateUnknown, Source: SourcePoll}, system("c"))), "not_found")
	expect(t, "unknown ws", second(f.r.Observe("w9", good, system("c"))), "not_found")
	expect(t, "assistant", second(f.r.Observe("w1", good, assistant("bot"))), ReasonAINotAllowed)
	o := good
	o.Connection = "api2"
	expect(t, "other connection key", second(f.r.Observe("w1", o, system("c"))), "conflict")
	v := must(f.r.Observe("w1", Observation{ID: "x1", Connection: "api", Key: "k1", External: "INV-1", Revision: 1, State: StateDraft, Source: SourceWebhook}, human("ops")))
	if v.Outcome != "applied" || v.Actor != "ops" {
		t.Fatalf("applied %+v", v)
	}
	expect(t, "key and ref differ", second(f.r.Observe("w1", Observation{ID: "x2", Connection: "api", Key: "k2", External: "INV-1", Revision: 1, State: StateDraft, Source: SourceWebhook}, system("c"))), "conflict")
	expect(t, "ref changes", second(f.r.Observe("w1", Observation{ID: "x3", Connection: "api", Key: "k1", External: "INV-9", Revision: 2, State: StatePosted, Source: SourceWebhook}, system("c"))), "conflict")
	expect(t, "prepared", second(f.r.Observe("w1", Observation{ID: "x4", Connection: "api", Key: "k3", Revision: 1, State: StateUnknown, Source: SourcePoll}, system("c"))), ReasonWrongState)
	if again := must(f.r.Observe("w1", Observation{ID: "x1", Connection: "api", Key: "k1", External: "INV-1", Revision: 1, State: StateDraft, Source: SourceWebhook}, system("other"))); again.Actor != "ops" {
		t.Fatal("repeat")
	}
	expect(t, "id reuse", second(f.r.Observe("w1", Observation{ID: "x1", Connection: "api", Key: "k1", Revision: 9, State: StateUnknown, Source: SourcePoll}, system("c"))), "conflict")
	byRef := f.observe(t, "api", "", "INV-1", 2, StatePosted, SourceWebhook)
	if byRef.Outcome != "applied" {
		t.Fatalf("by ref %+v", byRef)
	}
	if v := must(f.r.SubmissionOf("w1", "k1")); v.State != StatePosted || v.Posted != start {
		t.Fatalf("posted %+v", v)
	}
	if again := f.observe(t, "api", "k1", "INV-1", 3, StatePosted, SourcePoll); again.Outcome != "repeated" {
		t.Fatalf("repeated %+v", again)
	}
	expect(t, "ref taken", second(f.r.Observe("w1", Observation{ID: "x5", Connection: "api", Key: "k2", External: "INV-1", Revision: 1, State: StatePosted, Source: SourceWebhook}, system("c"))), "conflict")
	f.observe(t, "api", "k2", "", 1, StateUnknown, SourceWebhook)
	if v := f.observe(t, "api", "k2", "", 2, StateUnknown, SourceWebhook); v.Outcome != "repeated" {
		t.Fatalf("unknown repeat %+v", v)
	}
	f.observe(t, "api", "k2", "", 3, StateRejected, SourceWebhook)
	if v := must(f.r.SubmissionOf("w1", "k2")); v.State != StateRejected {
		t.Fatalf("rejected %+v", v)
	}
	must(f.r.Prepare("w1", invoice("k4", "api", line("l1", "s2", "1")), human("b")))
	f.observe(t, "api", "k2", "INV-2", 4, StatePosted, SourceWebhook)
	if a, b := must(f.r.SubmissionOf("w1", "k2")), must(f.r.SubmissionOf("w1", "k4")); !a.Diverged || !b.Diverged {
		t.Fatalf("spread %+v %+v", a, b)
	}
	small := newFixture(t)
	for i := 0; i < 60; i++ {
		_, _ = small.r.Observe("w1", Observation{ID: "n" + strings.Repeat("x", i+1), Connection: "api", External: "E", Revision: 1, State: StatePosted, Source: SourceWebhook}, system("c"))
	}
	expect(t, "limit", second(small.r.Observe("w1", Observation{ID: "last", Connection: "api", External: "E", Revision: 1, State: StatePosted, Source: SourceWebhook}, system("c"))), "limit")
	expect(t, "orphans ws", second(f.r.Orphans("w9")), "not_found")
	orphans := must(small.r.Orphans("w1"))
	orphans[0].Actor = "x"
	if must(small.r.Orphans("w1"))[0].Actor != "c" {
		t.Fatal("orphan aliasing")
	}
}

func TestResolveAndWithdrawGates(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	f.observe(t, "api", "k1", "INV-1", 1, StatePosted, SourceWebhook)
	res := Resolution{Outcome: ResolveKeep, Evidence: "e"}
	expect(t, "form outcome", second(f.r.Resolve("w1", "k1", Resolution{Outcome: "x", Evidence: "e"}, human("l"))), "invalid")
	expect(t, "form evidence", second(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep}, human("l"))), "invalid")
	expect(t, "ws", second(f.r.Resolve("", "k1", res, human("l"))), "invalid")
	expect(t, "key", second(f.r.Resolve("w1", "", res, human("l"))), "invalid")
	expect(t, "actor", second(f.r.Resolve("w1", "k1", res, human(""))), "invalid")
	expect(t, "missing", second(f.r.Resolve("w1", "nope", res, human("l"))), "not_found")
	expect(t, "not diverged", second(f.r.Resolve("w1", "k1", res, human("l"))), ReasonWrongState)
	f.observe(t, "api", "k1", "INV-1", 2, StateRejected, SourceWebhook)
	expect(t, "assistant", second(f.r.Resolve("w1", "k1", res, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "self", second(f.r.Resolve("w1", "k1", res, human("biller"))), ReasonSelfApproval)
	kept := must(f.r.Resolve("w1", "k1", res, human("lead")))
	if kept.Diverged || kept.State != StatePosted || kept.Resolution.Actor != "lead" {
		t.Fatalf("kept %+v", kept)
	}
	if again := must(f.r.Resolve("w1", "k1", res, human("lead"))); again.Resolution.At != kept.Resolution.At {
		t.Fatal("repeat")
	}
	expect(t, "repeat other", second(f.r.Resolve("w1", "k1", res, human("other"))), ReasonWrongState)
	f.observe(t, "api", "k1", "INV-1", 3, StateRejected, SourceWebhook)
	acc := must(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveAccept, Evidence: "e2"}, human("lead")))
	if acc.State != StateRejected {
		t.Fatalf("accept rejected %+v", acc)
	}
	must(f.r.Prepare("w1", invoice("k2", "api", line("l1", "s1", "1")), human("b")))
	must(f.r.Anchor("w1", Anchor{ID: "p", Connection: "api", Kind: AnchorPrepaid, External: "P", Customer: "acme", Sources: []string{"s9"}}, human("a")))
	g := newFixture(t)
	must(g.r.Prepare("w1", invoice("k1", "api", line("l1", "s1", "1")), human("b")))
	must(g.r.Anchor("w1", Anchor{ID: "p", Connection: "api", Kind: AnchorPrepaid, External: "P", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	expect(t, "accept without observation", second(g.r.Resolve("w1", "k1", Resolution{Outcome: ResolveAccept, Evidence: "e"}, human("lead"))), ReasonWrongState)
	sup := must(g.r.Resolve("w1", "k1", Resolution{Outcome: ResolveSuperseded, Evidence: "e"}, human("lead")))
	if sup.State != StateSuperseded || sup.Diverged {
		t.Fatalf("superseded %+v", sup)
	}
	expect(t, "withdraw form", second(g.r.Withdraw("w1", "", human("b"))), "invalid")
	expect(t, "withdraw ws", second(g.r.Withdraw("", "k1", human("b"))), "invalid")
	expect(t, "withdraw actor", second(g.r.Withdraw("w1", "k1", human(""))), "invalid")
	expect(t, "withdraw missing", second(g.r.Withdraw("w1", "nope", human("b"))), "not_found")
	expect(t, "withdraw superseded", second(g.r.Withdraw("w1", "k1", human("b"))), ReasonWrongState)
	must(g.r.Prepare("w1", invoice("k2", "api", line("l1", "s2", "1")), human("b")))
	expect(t, "withdraw assistant", second(g.r.Withdraw("w1", "k2", assistant("bot"))), ReasonAINotAllowed)
	w := must(g.r.Withdraw("w1", "k2", human("x")))
	if w.State != StateWithdrawn || w.WithdrawnBy != "x" {
		t.Fatalf("withdrawn %+v", w)
	}
	if again := must(g.r.Withdraw("w1", "k2", human("x"))); again.State != StateWithdrawn {
		t.Fatal("withdraw repeat")
	}
	expect(t, "withdraw other", second(g.r.Withdraw("w1", "k2", human("y"))), ReasonWrongState)
	must(g.r.Prepare("w1", invoice("k3", "api", line("l1", "s2", "1")), human("b")))
	h := newFixture(t)
	h.send(t, invoice("a", "api", line("l1", "s1", "1")))
	h.observe(t, "api", "a", "", 1, StateUnknown, SourcePoll)
	h.observe(t, "api", "a", "", 2, StateAbsent, SourcePoll)
	h.send(t, invoice("b", "api", line("l1", "s1", "1")))
	h.observe(t, "api", "a", "INV-B", 3, StatePosted, SourceWebhook)
	if v := must(h.r.SubmissionOf("w1", "a")); v.External != "INV-B" || v.State != StateAbsent || !v.Diverged {
		t.Fatalf("diverged reference %+v", v)
	}
	expect(t, "reference of diverged", second(h.r.Observe("w1", Observation{ID: "hb", Connection: "api", Key: "b", External: "INV-B", Revision: 1, State: StateDraft, Source: SourceWebhook}, system("c"))), "conflict")
	routed := h.observe(t, "api", "", "INV-B", 4, StatePosted, SourcePoll)
	if routed.Outcome != "repeated" && routed.Outcome != "diverged" {
		t.Fatalf("routed %+v", routed)
	}
	m := newFixture(t)
	m.send(t, invoice("a", "api", line("l1", "s1", "1")))
	m.observe(t, "api", "a", "", 1, StateUnknown, SourcePoll)
	m.observe(t, "api", "a", "", 2, StateAbsent, SourcePoll)
	m.observe(t, "api", "a", "INV-A", 3, StatePosted, SourceWebhook)
	got := must(m.r.Resolve("w1", "a", Resolution{Outcome: ResolveAccept, Evidence: "e"}, human("lead")))
	if got.State != StatePosted || got.External != "INV-A" || !got.Posted.Equal(start) {
		t.Fatalf("accept posted %+v", got)
	}
}

func TestWatchAndChecks(t *testing.T) {
	f := newFixture(t)
	expect(t, "due form", second(f.r.Due("w1", time.Time{})), "invalid")
	expect(t, "due ws", second(f.r.Due("", start)), "invalid")
	expect(t, "due missing", second(f.r.Due("w9", start)), "not_found")
	must(f.r.Prepare("w1", invoice("k0", "api", line("l1", "s0", "1")), human("b")))
	f.send(t, invoice("k1", "api", line("l1", "s1", "1")))
	f.send(t, invoice("k2", "api", line("l1", "s2", "1")))
	if due := must(f.r.Due("w1", start.Add(time.Hour-1))); len(due) != 0 {
		t.Fatalf("early %v", due)
	}
	if due := must(f.r.Due("w1", start.Add(time.Hour))); strings.Join(due, ",") != "k1,k2" {
		t.Fatalf("due %v", due)
	}
	expect(t, "form", second(f.r.Checked("w1", "k1", "slow", system("p"))), "invalid")
	expect(t, "ws", second(f.r.Checked("", "k1", "timeout", system("p"))), "invalid")
	expect(t, "key", second(f.r.Checked("w1", "", "timeout", system("p"))), "invalid")
	expect(t, "actor", second(f.r.Checked("w1", "k1", "timeout", system(""))), "invalid")
	expect(t, "missing", second(f.r.Checked("w1", "nope", "timeout", system("p"))), "not_found")
	expect(t, "assistant", second(f.r.Checked("w1", "k1", "timeout", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "not watched", second(f.r.Checked("w1", "k0", "timeout", system("p"))), ReasonWrongState)
	f.clock.advance(time.Minute)
	v := must(f.r.Checked("w1", "k1", "error", human("ops")))
	if v.Watch.Retries != 1 || !v.Watch.Last.Equal(start.Add(time.Minute)) || !v.Watch.Next.Equal(start.Add(time.Minute+time.Hour)) || v.State != StateUnknown {
		t.Fatalf("checked %+v", v)
	}
	v = must(f.r.Checked("w1", "k1", "timeout", system("p")))
	if v.Watch.Retries != 2 || v.State != StateUnknown {
		t.Fatalf("second %+v", v)
	}
	f.observe(t, "api", "k1", "INV-1", 1, StatePosted, SourcePoll)
	v = must(f.r.SubmissionOf("w1", "k1"))
	if v.Watch.Retries != 0 || v.Watch.Revision != 1 {
		t.Fatalf("poll reset %+v", v.Watch)
	}
	f.observe(t, "api", "k2", "D", 1, StateDraft, SourceWebhook)
	if v := must(f.r.SubmissionOf("w1", "k2")); v.Watch.Revision != 0 || v.State != StateDraft {
		t.Fatalf("webhook does not reset %+v", v.Watch)
	}
	f.observe(t, "api", "k2", "D", 2, StateRejected, SourcePoll)
	f.clock.advance(30*24*time.Hour - time.Minute - 1)
	if due := must(f.r.Due("w1", f.clock.read())); strings.Join(due, ",") != "k1" {
		t.Fatalf("posted watched %v", due)
	}
	must(f.r.Checked("w1", "k1", "timeout", system("p")))
	f.clock.advance(time.Minute + time.Nanosecond)
	if due := must(f.r.Due("w1", f.clock.read().Add(48*time.Hour))); len(due) != 0 {
		t.Fatalf("posted after window %v", due)
	}
	expect(t, "unwatched", second(f.r.Checked("w1", "k1", "timeout", system("p"))), ReasonWrongState)
	f.observe(t, "api", "k1", "INV-1", 2, StateRejected, SourceWebhook)
	if due := must(f.r.Due("w1", f.clock.read().Add(48*time.Hour))); strings.Join(due, ",") != "k1" {
		t.Fatalf("diverged watched %v", due)
	}
	if v := must(f.r.Checked("w1", "k1", "timeout", system("p"))); v.State != StatePosted {
		t.Fatalf("check keeps posted %+v", v)
	}
}

func doc(external string, revision int, rows ...Row) Document {
	return Document{Connection: "api", External: external, Revision: revision, Rows: rows}
}

func row(id, net, tax, qty string) Row {
	return Row{ID: id, Product: "P", Net: net, Tax: tax, Quantity: qty}
}

func TestDocumentAndLinkGates(t *testing.T) {
	f := newFixture(t)
	f.connect(t, apiConnection("api2"))
	f.send(t, invoice("k1", "api", Line{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "2", Net: "100", Tax: "10"}, Line{ID: "l2", Source: "s2", Product: "compute", TaxCode: "vat10", Quantity: "0", Net: "-20", Tax: "0"}))
	good := doc("D", 1, row("r1", "100", "10", "2"), row("r2", "-20", "0", "0"))
	bad := []func(*Document){
		func(d *Document) { d.Connection = "" }, func(d *Document) { d.External = "" }, func(d *Document) { d.Revision = 0 },
		func(d *Document) { d.Rows = nil }, func(d *Document) { d.Rows = []Row{row("r", "1", "0", "1"), row("r", "1", "0", "1")} },
		func(d *Document) { d.Rows = []Row{{ID: "", Product: "P", Net: "1", Tax: "0", Quantity: "1"}} },
		func(d *Document) { d.Rows = []Row{{ID: "r", Product: "", Net: "1", Tax: "0", Quantity: "1"}} },
		func(d *Document) { d.Rows = []Row{row("r", "x", "0", "1")} }, func(d *Document) { d.Rows = []Row{row("r", "1", "x", "1")} },
		func(d *Document) { d.Rows = []Row{row("r", "1", "0", "x")} },
	}
	for i, change := range bad {
		d := good
		change(&d)
		if got := kind(second(f.r.Document("w1", d, system("c")))); got != "invalid" {
			t.Fatalf("doc %d: %s", i, got)
		}
	}
	for _, r := range []Row{row("r", "1.5", "0", "1"), row("r", "1", "0.5", "1"), row("r", "1", "0", "0.5")} {
		expect(t, "scale", second(f.r.Document("w1", doc("S", 1, r), system("c"))), "invalid")
	}
	expect(t, "ws", second(f.r.Document("", good, system("c"))), "invalid")
	expect(t, "actor", second(f.r.Document("w1", good, system(""))), "invalid")
	expect(t, "unknown connection", second(f.r.Document("w9", good, system("c"))), "not_found")
	expect(t, "assistant", second(f.r.Document("w1", good, assistant("bot"))), ReasonAINotAllowed)
	v := must(f.r.Document("w1", good, system("c")))
	if v.Total != (Amounts{"80", "10", "2"}) {
		t.Fatalf("total %+v", v.Total)
	}
	if again := must(f.r.Document("w1", good, human("other"))); again.Author != "c" {
		t.Fatal("repeat")
	}
	changed := doc("D", 1, row("r1", "99", "10", "2"))
	expect(t, "same revision changed", second(f.r.Document("w1", changed, system("c"))), "conflict")
	l := Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "60", Tax: "6", Quantity: "1"}
	expect(t, "link form", second(f.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "x", Tax: "0", Quantity: "0"}, human("b"))), "invalid")
	for i, change := range []func(*Link){
		func(x *Link) { x.Connection = "" }, func(x *Link) { x.External = "" }, func(x *Link) { x.Row = "" }, func(x *Link) { x.Key = "" },
		func(x *Link) { x.Line = "" }, func(x *Link) { x.Tax = "x" }, func(x *Link) { x.Quantity = "x" },
	} {
		x := l
		change(&x)
		if got := kind(second(f.r.Link("w1", x, human("b")))); got != "invalid" {
			t.Fatalf("link %d: %s", i, got)
		}
	}
	expect(t, "link ws", second(f.r.Link("", l, human("b"))), "invalid")
	expect(t, "link actor", second(f.r.Link("w1", l, human(""))), "invalid")
	expect(t, "link unknown connection", second(f.r.Link("w9", l, human("b"))), "not_found")
	x := l
	x.External = "nope"
	expect(t, "unknown document", second(f.r.Link("w1", x, human("b"))), ReasonUnknownDocument)
	expect(t, "assistant", second(f.r.Link("w1", l, assistant("bot"))), ReasonAINotAllowed)
	x = l
	x.Key = "nope"
	expect(t, "unknown submission", second(f.r.Link("w1", x, human("b"))), ReasonUnknownSubmission)
	x = l
	x.Line = "l9"
	expect(t, "unknown line", second(f.r.Link("w1", x, human("b"))), ReasonUnknownSubmission)
	x = l
	x.Row = "r9"
	expect(t, "unknown row", second(f.r.Link("w1", x, human("b"))), ReasonUnknownDocument)
	must(f.r.Document("w1", Document{Connection: "api2", External: "D", Revision: 1, Rows: []Row{row("r1", "100", "10", "2")}}, system("c")))
	x = l
	x.Connection = "api2"
	expect(t, "connection mismatch", second(f.r.Link("w1", x, human("b"))), ReasonConnectionMismatch)
	x = l
	x.Net = "60.5"
	expect(t, "link scale", second(f.r.Link("w1", x, human("b"))), "invalid")
	for _, over := range []Link{
		{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "101", Tax: "0", Quantity: "0"},
		{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "-1", Tax: "0", Quantity: "0"},
		{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "0", Tax: "11", Quantity: "0"},
		{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "0", Tax: "0", Quantity: "3"},
		{Connection: "api", External: "D", Row: "r2", Key: "k1", Line: "l2", Net: "-20", Tax: "0", Quantity: "1"},
		{Connection: "api", External: "D", Row: "r2", Key: "k1", Line: "l1", Net: "-20", Tax: "0", Quantity: "0"},
	} {
		expect(t, "over", second(f.r.Link("w1", over, human("b"))), ReasonOverAllocated)
	}
	got := must(f.r.Link("w1", l, human("b")))
	if len(got.Links) != 1 {
		t.Fatalf("links %+v", got.Links)
	}
	if again := must(f.r.Link("w1", l, human("b"))); len(again.Links) != 1 || again.Links[0].Author != "b" {
		t.Fatal("link repeat")
	}
	expect(t, "link repeat other", second(f.r.Link("w1", l, human("c"))), "conflict")
	x = l
	x.Row = "r2"
	expect(t, "link system", second(f.r.Link("w1", x, system("job"))), ReasonNotHuman)
	x = l
	x.Net = "10"
	expect(t, "link changed", second(f.r.Link("w1", x, human("b"))), "conflict")
	more := Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l2", Net: "0", Tax: "0", Quantity: "0"}
	must(f.r.Link("w1", more, human("b")))
	f.send(t, invoice("k5", "api", Line{ID: "l1", Source: "s5", Product: "compute", TaxCode: "vat10", Quantity: "2", Net: "100", Tax: "10"}))
	expect(t, "row exhausted", second(f.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k5", Line: "l1", Net: "41", Tax: "0", Quantity: "0"}, human("b"))), ReasonOverAllocated)
	must(f.r.Document("w1", doc("E", 1, row("e1", "100", "10", "2"), row("e2", "5", "0", "1")), system("c")))
	must(f.r.Link("w1", Link{Connection: "api", External: "E", Row: "e1", Key: "k1", Line: "l1", Net: "40", Tax: "4", Quantity: "1"}, human("b")))
	expect(t, "line exhausted", second(f.r.Link("w1", Link{Connection: "api", External: "E", Row: "e2", Key: "k1", Line: "l1", Net: "1", Tax: "0", Quantity: "0"}, human("b"))), ReasonOverAllocated)
	f.observe(t, "api", "k5", "", 1, StateUnknown, SourcePoll)
	f.observe(t, "api", "k5", "", 2, StateAbsent, SourcePoll)
	expect(t, "retired submission", second(f.r.Link("w1", Link{Connection: "api", External: "E", Row: "e1", Key: "k5", Line: "l1", Net: "1", Tax: "0", Quantity: "0"}, human("b"))), ReasonWrongState)
	must(f.r.Link("w1", Link{Connection: "api", External: "D", Row: "r2", Key: "k1", Line: "l2", Net: "-20", Tax: "0", Quantity: "0"}, human("b")))
	shrunk := must(f.r.Document("w1", doc("D", 2, row("r1", "50", "10", "2")), system("c")))
	if !shrunk.Links[0].Stale || shrunk.Links[1].Stale || !shrunk.Links[2].Stale {
		t.Fatalf("stale after removal and shrink %+v", shrunk.Links)
	}
	expect(t, "older revision", second(f.r.Document("w1", doc("D", 1, row("r1", "100", "10", "2")), system("c"))), "conflict")
	if v := must(f.r.DocumentOf("w1", "api", "D")); v.Document.Revision != 2 {
		t.Fatalf("document of %+v", v)
	}
	expect(t, "document of", second(f.r.DocumentOf("w1", "api", "nope")), "not_found")
	expect(t, "document of ws", second(f.r.DocumentOf("w9", "api", "D")), "not_found")
	rec := must(f.r.Reconcile("w1", "k1"))
	if rec.Matched || rec.Lines[0].Linked.Net != "40" || len(rec.Documents) != 2 || rec.Documents[0].Attributed.Net != "0" {
		t.Fatalf("after stale %+v", rec)
	}
	expect(t, "reconcile missing", second(f.r.Reconcile("w1", "nope")), "not_found")
	flip := must(f.r.Document("w1", doc("E", 2, row("e1", "-100", "10", "2")), system("c")))
	if !flip.Links[0].Stale {
		t.Fatal("sign flip not stale")
	}
	g := newFixture(t)
	g.send(t, invoice("k1", "api", line("l1", "s1", "100"), line("l2", "s2", "100")))
	must(g.r.Document("w1", doc("D", 1, row("r1", "100", "0", "2")), system("c")))
	must(g.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l1", Net: "60", Tax: "0", Quantity: "1"}, human("b")))
	must(g.r.Link("w1", Link{Connection: "api", External: "D", Row: "r1", Key: "k1", Line: "l2", Net: "40", Tax: "0", Quantity: "1"}, human("b")))
	refit := must(g.r.Document("w1", doc("D", 2, row("r1", "70", "0", "2")), system("c")))
	if refit.Links[0].Stale || !refit.Links[1].Stale {
		t.Fatalf("refit %+v", refit.Links)
	}
	limits := newFixture(t)
	for i := 0; i < 6; i++ {
		must(limits.r.Document("w1", doc("L"+string(rune('0'+i)), 1, row("r", "1", "0", "1")), system("c")))
	}
	expect(t, "documents", second(limits.r.Document("w1", doc("L9", 1, row("r", "1", "0", "1")), system("c"))), "limit")
	if v := must(limits.r.Document("w1", doc("L0", 2, row("r", "2", "0", "1")), system("c"))); v.Total.Net != "2" {
		t.Fatal("replace at limit")
	}
	expect(t, "rows", second(limits.r.Document("w1", doc("L0", 3, row("1", "1", "0", "1"), row("2", "1", "0", "1"), row("3", "1", "0", "1"), row("4", "1", "0", "1"), row("5", "1", "0", "1")), system("c"))), "limit")
	_ = big.NewInt
}
