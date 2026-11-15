package retention

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRegistrationAndReferences(t *testing.T) {
	f := setup(t)
	good := Artifact{ID: "a", Class: ClassSource, Tenants: []string{"t1", "t2"}, Anchor: start, Stores: []string{"db", "obj"}}
	f.must(f.r.Register(good, ingest))
	f.must(f.r.Register(good, ingest))
	local := good
	local.Anchor = start.In(time.FixedZone("x", 3600))
	f.must(f.r.Register(local, ingest))
	for name, mutate := range map[string]func(*Artifact){
		"class": func(a *Artifact) { a.Class = ClassFinal }, "anchor": func(a *Artifact) { a.Anchor = start.Add(-time.Second) },
		"stores": func(a *Artifact) { a.Stores = []string{"db"} }, "stores same length": func(a *Artifact) { a.Stores = []string{"db", "zz"} },
		"tenants": func(a *Artifact) { a.Tenants = []string{"t1", "t3"} },
	} {
		x := good
		mutate(&x)
		expect(t, "conflict "+name, f.r.Register(x, ingest), ErrConflict)
	}
	for name, mutate := range map[string]func(*Artifact){
		"id": func(a *Artifact) { a.ID = "" }, "class": func(a *Artifact) { a.Class = "logs" }, "tenants": func(a *Artifact) { a.Tenants = nil },
		"tenant order": func(a *Artifact) { a.Tenants = []string{"b", "a"} }, "zero anchor": func(a *Artifact) { a.Anchor = time.Time{} },
		"future anchor": func(a *Artifact) { a.Anchor = start.Add(time.Nanosecond) }, "stores": func(a *Artifact) { a.Stores = nil },
		"store form": func(a *Artifact) { a.Stores = []string{"a b"} }, "17 stores": func(a *Artifact) { a.Stores = ids("s", 17) },
		"65 tenants": func(a *Artifact) { a.Tenants = ids("t", 65) },
	} {
		x := good
		x.ID = "n"
		mutate(&x)
		expect(t, "form "+name, f.r.Register(x, ingest), ErrInvalid)
	}
	edge := Artifact{ID: "e", Class: ClassClaim, Tenants: ids("t", 64), Anchor: start, Stores: ids("s", 16)}
	f.must(f.r.Register(edge, ingest))
	expect(t, "65th tenant", f.r.AddTenant("e", "zzz", ingest), ErrLimit)
	reason(t, "register by human", f.r.Register(good, ann), ReasonNotSystem)
	reason(t, "register by assistant", f.r.Register(good, bot), ReasonAINotAllowed)
	expect(t, "register actor", f.r.Register(good, Actor{ID: "x", Kind: "robot"}), ErrInvalid)
	f.put("b", ClassFinal, 0, one)
	for _, x := range []Reference{{From: "a", To: "a", Kind: Requires}, {From: "", To: "a", Kind: Requires}, {From: "a", To: "", Kind: Requires}, {From: "a", To: "b", Kind: "needs"}} {
		expect(t, "ref form", f.r.Refer(x, ingest), ErrInvalid)
	}
	expect(t, "ref missing", f.r.Refer(Reference{From: "a", To: "zz", Kind: Requires}, ingest), ErrNotFound)
	expect(t, "ref missing from", f.r.Refer(Reference{From: "zz", To: "a", Kind: Requires}, ingest), ErrNotFound)
	f.must(f.r.Refer(Reference{From: "b", To: "a", Kind: Requires}, ingest))
	f.must(f.r.Refer(Reference{From: "b", To: "a", Kind: Requires}, ingest))
	f.must(f.r.Refer(Reference{From: "b", To: "a", Kind: Mentions}, ingest))
	if f.r.requiredBy["a"] != 1 || len(f.r.requires["b"]) != 1 {
		t.Error("idempotent requires")
	}
	reason(t, "ref by human", f.r.Refer(Reference{From: "a", To: "b", Kind: Mentions}, ann), ReasonNotSystem)
	expect(t, "artifact missing", second(f.r.Artifact("zz")), ErrNotFound)
	expect(t, "access missing", second(f.r.Access("zz")), ErrNotFound)
	v, _ := f.r.Artifact("a")
	v.Tenants[0] = "zz"
	if w, _ := f.r.Artifact("a"); w.Tenants[0] != "t1" {
		t.Error("artifact shares memory")
	}
	g := setup(t)
	for len(g.r.artifacts) < maxArtifacts-1 {
		break
	}
	g.r.artifacts = map[string]*artifact{}
	for i := 0; i < 3; i++ {
		g.put("x"+itoa(i), ClassSource, 0, one)
	}
	g.put("y", ClassSource, 0, one)
	g.r.refs = map[[3]string]bool{}
	for len(g.r.refs) < 10000000 {
		g.r.refs[[3]string{"z", itoa(len(g.r.refs)), Mentions}] = true
		if len(g.r.refs) == 2 {
			break
		}
	}
	h := setup(t)
	h.put("p", ClassSource, 0, one)
	h.put("q", ClassSource, 0, one)
	for len(h.r.refs) < 10000000 {
		h.r.refs[[3]string{"z", itoa(len(h.r.refs)), Mentions}] = true
	}
	expect(t, "refs cap", h.r.Refer(Reference{From: "p", To: "q", Kind: Mentions}, ingest), ErrLimit)
	delete(h.r.refs, [3]string{"z", "0", Mentions})
	h.must(h.r.Refer(Reference{From: "p", To: "q", Kind: Mentions}, ingest))
	h.r.refs = nil
	for len(h.r.artifacts) < 10000000 {
		h.r.artifacts["z"+itoa(len(h.r.artifacts))] = &artifact{}
	}
	expect(t, "artifacts cap", h.r.Register(Artifact{ID: "new", Class: ClassSource, Tenants: one, Anchor: start, Stores: []string{"db"}}, ingest), ErrLimit)
	h.must(h.r.Register(Artifact{ID: "p", Class: ClassSource, Tenants: one, Anchor: start, Stores: []string{"db"}}, ingest))
}

func ids(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = prefix + itoa(100000+i)
	}
	return out
}

func TestForms(t *testing.T) {
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
	for _, c := range []Change{
		{Kind: "x"}, {Kind: ChangePolicy, Policy: Policy{Class: "x", Period: 1}}, {Kind: ChangePolicy, Policy: Policy{Class: ClassSource}},
		{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 36501}}, {Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 1, Grace: -1}},
		{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 1, Grace: 366}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: "x", Artifacts: []string{"a"}, Chunk: 1, Lease: time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Chunk: 1, Lease: time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"b", "a"}, Chunk: 1, Lease: time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: ids("a", 100001), Chunk: 1, Lease: time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Lease: time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1001, Lease: time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: time.Minute - 1, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: 24*time.Hour + 1, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: time.Hour}},
		{Kind: ChangeRelease, Release: Release{Hold: "", Reason: "r"}}, {Kind: ChangeRelease, Release: Release{Hold: "h"}},
	} {
		if _, err := clean(c); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v accepted", c)
		}
	}
	for _, c := range []Change{
		{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 36500, Grace: 365}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: ids("a", 100000), Chunk: 1000, Lease: 24 * time.Hour, Reason: "r"}},
		{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: time.Minute, Reason: "r"}},
	} {
		if _, err := clean(c); err != nil {
			t.Errorf("%s rejected", c.Kind)
		}
	}
	for _, h := range []Hold{{ID: "", Owner: "o", Reason: "r", Artifacts: []string{"a"}}, {ID: "h", Reason: "r", Artifacts: []string{"a"}}, {ID: "h", Owner: "o", Artifacts: []string{"a"}},
		{ID: "h", Owner: "o", Reason: "r"}, {ID: "h", Owner: "o", Reason: "r", Classes: []string{ClassSource}, Artifacts: []string{"a"}},
		{ID: "h", Owner: "o", Reason: "r", Tenants: []string{"b", "a"}}, {ID: "h", Owner: "o", Reason: "r", Tenants: ids("t", 65)},
		{ID: "h", Owner: "o", Reason: "r", Tenants: one, Classes: []string{"x"}}, {ID: "h", Owner: "o", Reason: "r", Artifacts: ids("a", 10001)}} {
		if h.valid() {
			t.Errorf("hold %+v accepted", h.ID)
		}
	}
	if !(Hold{ID: "h", Owner: "o", Reason: "r", Tenants: ids("t", 64), Classes: Classes, Artifacts: ids("a", 10000)}).valid() {
		t.Error("hold edge")
	}
	if !validText(strings.Repeat("a", 200)) || validText(strings.Repeat("a", 201)) || validText("a\x01") || validText("a\x7f") || validText("a\u0080") || validText("a\u009f") || !validText("a ") || validText("\xff") || !validText(" ") {
		t.Error("text")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("id %q", id)
		}
	}
	for _, id := range []string{"", "/", "a b", "@", "[", "`", "{", "!", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	if !contains([]string{"a", "c", "e"}, "e") || contains([]string{"a", "c", "e"}, "d") || contains(nil, "a") || contains([]string{"a", "c", "e"}, "f") || !contains([]string{"a", "c", "e"}, "a") {
		t.Error("contains")
	}
	if sameStrings([]string{"a"}, []string{"a", "b"}) || !sameStrings(nil, []string{}) {
		t.Error("same")
	}
}

func TestProposals(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 10}}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, worker)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{Kind: KindHuman})), ErrInvalid)
	expect(t, "form", second(f.r.Propose(Change{Kind: "x"}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if again, _ := f.r.Propose(c, ann); again.Number != p.Number {
		t.Error("repeat")
	}
	q, _ := f.r.Propose(c, cat)
	reason(t, "self", second(f.r.Approve(p.Number, ann, Impact{})), ReasonSelfApproval)
	reason(t, "approve system", second(f.r.Approve(p.Number, worker, Impact{})), ReasonNotHuman)
	expect(t, "missing", second(f.r.Approve(99, bob, Impact{})), ErrNotFound)
	f.must(second(f.r.Approve(p.Number, bob, Impact{})))
	if again, err := f.r.Approve(p.Number, bob, Impact{}); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "applied", second(f.r.Approve(p.Number, cat, Impact{})), ReasonNotPending)
	reason(t, "withdraw other", second(f.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(f.r.Withdraw(q.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := f.r.Withdraw(q.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(f.r.Proposal(q.Number)), ErrNotFound)
	f.put("a", ClassSource, 20, one)
	pp, _ := f.r.Propose(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: time.Hour, Reason: "r"}}, ann)
	pp.Change.Plan.Artifacts[0] = "zz"
	if got, _ := f.r.Proposal(pp.Number); got.Change.Plan.Artifacts[0] != "a" {
		t.Error("proposal shares memory")
	}
	expect(t, "plan missing", f.try(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"zz"}, Chunk: 1, Lease: time.Hour, Reason: "r"}}), ErrNotFound)
	f.put("fx", ClassFinal, 20, one)
	expect(t, "plan class", f.try(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"fx"}, Chunk: 1, Lease: time.Hour, Reason: "r"}}), ErrInvalid)
	f.must(second(f.r.Approve(pp.Number, bob, Impact{})))
	reason(t, "in plan", f.try(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: time.Hour, Reason: "s"}}), ReasonInPlan)
	cl, _ := f.r.Claim("run/1", 0, worker)
	f.must(f.r.Report("run/1", 0, cl.Token, "db", []string{"a"}, nil, worker))
	reason(t, "already deleted", f.try(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"a"}, Chunk: 1, Lease: time.Hour, Reason: "t"}}), ReasonAlreadyDeleted)
	u := setup(t)
	u.policy(ClassSource, 1, 0)
	for n := 0; n < 100; n++ {
		u.put("u"+itoa(n), ClassSource, 5, one)
		u.plan(ClassSource, 1, "u"+itoa(n))
	}
	u.put("last", ClassSource, 5, one)
	expect(t, "unfinished cap", u.try(Change{Kind: ChangePlan, Plan: DeletionPlan{Class: ClassSource, Artifacts: []string{"last"}, Chunk: 1, Lease: time.Hour, Reason: "r"}}), ErrLimit)
	cl, _ = u.r.Claim("run/1", 0, worker)
	u.must(u.r.Report("run/1", 0, cl.Token, "db", nil, []string{"u0"}, worker))
	u.plan(ClassSource, 1, "last")
	h := setup(t)
	people := []Actor{ann, bob, cat, dan}
	for _, who := range people {
		for n := 0; n < 250; n++ {
			h.must(second(h.r.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: n + 1}}, who)))
		}
		expect(t, "mine", second(h.r.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassFinal, Period: 1}}, who)), ErrLimit)
	}
	expect(t, "total", second(h.r.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassFinal, Period: 1}}, Actor{ID: "eve", Kind: KindHuman})), ErrLimit)
	d := setup(t)
	first := d.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 1}})
	for n := 2; n < 1000; n++ {
		d.r.done = append(d.r.done, uint64(100000+n))
	}
	d.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 2}})
	if _, err := d.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	d.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 3}})
	expect(t, "pruned", second(d.r.Proposal(first.Number)), ErrNotFound)
	k := setup(t)
	k.put("a", ClassSource, 0, one)
	for len(k.r.live) < 9999 {
		k.r.live["z"+itoa(len(k.r.live))] = &hold{}
	}
	k.must(second(k.r.PlaceHold(Hold{ID: "h", Owner: "o", Reason: "r", Artifacts: []string{"a"}}, cat)))
	expect(t, "holds cap", second(k.r.PlaceHold(Hold{ID: "h2", Owner: "o", Reason: "r", Artifacts: []string{"a"}}, cat)), ErrLimit)
}

func TestReadsAndRuns(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	for _, id := range []string{"a", "b", "c"} {
		f.put(id, ClassSource, 5, one)
	}
	p, _ := f.r.Preview(ClassSource, "", 2)
	if len(p.Eligible) != 2 || !p.More {
		t.Errorf("page %+v", p)
	}
	p, _ = f.r.Preview(ClassSource, "b", 2)
	if len(p.Eligible) != 1 || p.Eligible[0] != "c" || p.More {
		t.Errorf("after %+v", p)
	}
	for _, x := range []struct {
		c string
		n int
	}{{"x", 1}, {ClassSource, 0}, {ClassSource, 10001}} {
		expect(t, "preview form", second(f.r.Preview(x.c, "", x.n)), ErrInvalid)
	}
	if p, _ := f.r.Preview(ClassSource, "", 10000); len(p.Eligible) != 3 {
		t.Error("max page")
	}
	expect(t, "run missing", second(f.r.Run("run/9")), ErrNotFound)
	id := f.plan(ClassSource, 3, "a", "b", "c")
	for _, call := range []func() error{
		func() error { return second(f.r.Claim("run/9", 0, worker)) }, func() error { return second(f.r.Retry("run/9", 0, worker)) },
		func() error { return f.r.Report("run/9", 0, 1, "db", []string{"a"}, nil, worker) }, func() error { return second(f.r.Proceed("run/9", 0, 1, "a", worker)) },
	} {
		expect(t, "missing run", call(), ErrNotFound)
	}
	expect(t, "chunk range", second(f.r.Claim(id, 1, worker)), ErrInvalid)
	expect(t, "chunk negative", second(f.r.Claim(id, -1, worker)), ErrInvalid)
	reason(t, "claim human", second(f.r.Claim(id, 0, ann)), ReasonNotSystem)
	reason(t, "retry unclaimed", second(f.r.Retry(id, 0, worker)), ReasonNotClaimed)
	c, _ := f.r.Claim(id, 0, worker)
	for name, call := range map[string]func() error{
		"store":          func() error { return f.r.Report(id, 0, c.Token, "", []string{"a"}, nil, worker) },
		"empty":          func() error { return f.r.Report(id, 0, c.Token, "db", nil, nil, worker) },
		"order":          func() error { return f.r.Report(id, 0, c.Token, "db", []string{"b", "a"}, nil, worker) },
		"both":           func() error { return f.r.Report(id, 0, c.Token, "db", []string{"a"}, []string{"a"}, worker) },
		"foreign":        func() error { return f.r.Report(id, 0, c.Token, "db", []string{"zz"}, nil, worker) },
		"wrong store":    func() error { return f.r.Report(id, 0, c.Token, "obj", []string{"a"}, nil, worker) },
		"failed foreign": func() error { return f.r.Report(id, 0, c.Token, "db", nil, []string{"zz"}, worker) },
		"failed store":   func() error { return f.r.Report(id, 0, c.Token, "obj", nil, []string{"a"}, worker) },
	} {
		expect(t, "report "+name, call(), ErrInvalid)
	}
	f.advance(30 * time.Minute)
	reason(t, "retry leased", second(f.r.Retry(id, 0, worker)), ReasonNotClaimed)
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"a"}, []string{"b", "c"}, worker))
	rc, _ := f.r.Retry(id, 0, worker)
	reason(t, "retry while leased", second(f.r.Retry(id, 0, worker)), ReasonLeased)
	f.advance(2 * time.Hour)
	rc2, err := f.r.Retry(id, 0, worker)
	if err != nil || rc2.Token == rc.Token || len(rc2.Deleting) != 2 {
		t.Errorf("retry after lease %+v %v", rc2, err)
	}
	f.advance(2 * time.Hour)
	if again, err := f.r.Claim(id, 0, worker); err != nil || len(again.Deleting) != 2 {
		t.Errorf("claim reissue %+v %v", again, err)
	}
	if f.access("a") != AccessDeleted || f.access("b") != AccessBlocked {
		t.Error("access")
	}
	expect(t, "tombstone limit", second(f.r.Tombstones(0, 0)), ErrInvalid)
	expect(t, "tombstone big", second(f.r.Tombstones(0, 1001)), ErrInvalid)
	if ts, _ := f.r.Tombstones(5, 1000); len(ts) != 0 {
		t.Error("after end")
	}
	if ts, _ := f.r.Tombstones(0, 1000); len(ts) != 1 || ts[0].Sequence != 1 {
		t.Error("first")
	}
	v := f.run(id)
	v.Plan.Artifacts[0] = "zz"
	if w := f.run(id); w.Plan.Artifacts[0] != "a" {
		t.Error("run shares memory")
	}
}

func TestConcurrency(t *testing.T) {
	f := setup(t)
	f.policy(ClassTemporary, 1, 0)
	for n := 0; n < 40; n++ {
		f.put("a"+itoa(100+n), ClassTemporary, 5, one)
	}
	all := []string{}
	for n := 0; n < 40; n++ {
		all = append(all, "a"+itoa(100+n))
	}
	rid := f.plan(ClassTemporary, 10, all...)
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 20; m++ {
				f.r.Register(Artifact{ID: "c" + itoa(n*100+m), Class: ClassSource, Tenants: one, Anchor: start, Stores: []string{"db"}}, ingest)
				f.r.PlaceHold(Hold{ID: "h" + itoa(n*100+m), Owner: "o", Reason: "r", Artifacts: []string{"c" + itoa(n*100+m)}}, cat)
				for i := 0; i < 4; i++ {
					if c, err := f.r.Claim(rid, i, worker); err == nil {
						f.r.Report(rid, i, c.Token, "db", c.Deleting, nil, worker)
					}
				}
				f.r.Run(rid)
			}
		}(n)
	}
	wg.Wait()
	if v := f.run(rid); v.Counts[OutcomeDeleted] != 40 || f.r.Watermark() != 40 || len(f.r.holds) != 160 {
		t.Errorf("final %+v %d", v.Counts, len(f.r.holds))
	}
}
