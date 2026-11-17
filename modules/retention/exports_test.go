package retention

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

func snapshot(r *Registry) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := fmt.Sprint(r.counter, r.tokens, r.runSeq, r.unfinished, r.tombstones, r.pending, r.done, r.refs, r.requiredBy, r.requires, r.backups, r.policies)
	ids := []string{}
	for id := range r.artifacts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		a := r.artifacts[id]
		out += fmt.Sprint("|", id, a.view, a.deleted, a.damaged, a.stores, a.storeAt, a.run)
	}
	holds := []string{}
	for id := range r.holds {
		holds = append(holds, id)
	}
	sort.Strings(holds)
	for _, id := range holds {
		h := r.holds[id]
		out += fmt.Sprint("#", id, h.view, h.placer, h.released, r.live[id] != nil)
	}
	runs := []string{}
	for id := range r.runs {
		runs = append(runs, id)
	}
	sort.Strings(runs)
	for _, id := range runs {
		x := r.runs[id]
		out += fmt.Sprint("&", id, x.plan, x.revision, x.outcomes, x.chunks, x.stopped, x.stopper, x.by)
	}
	numbers := []uint64{}
	for n := range r.proposals {
		numbers = append(numbers, n)
	}
	sort.Slice(numbers, func(a, b int) bool { return numbers[a] < numbers[b] })
	for _, n := range numbers {
		p := r.proposals[n]
		out += fmt.Sprint("@", n, p.view, p.code)
	}
	return out
}

func refused(reason string) error { return Refusal{Reason: reason} }

func (f world) agree(name string, c Change, proposer Actor, want error) {
	f.t.Helper()
	before := snapshot(f.r)
	got := f.r.Check(c, proposer.ID, "", Impact{})
	if after := snapshot(f.r); after != before {
		f.t.Errorf("%s: check changed state", name)
	}
	p, proposed := f.r.Propose(c, proposer)
	if fmt.Sprint(got) != fmt.Sprint(proposed) || fmt.Sprint(got) != fmt.Sprint(want) {
		f.t.Errorf("%s: check %v propose %v want %v", name, got, proposed, want)
	}
	if proposed == nil {
		f.must(second(f.r.Withdraw(p.Number, proposer)))
	}
}

func (f world) agreeApprove(name string, number uint64, approver Actor, reviewed Impact, want error) {
	f.t.Helper()
	p, err := f.r.Proposal(number)
	f.must(err)
	before := snapshot(f.r)
	got := f.r.Check(p.Change, p.Proposer, approver.ID, reviewed)
	if after := snapshot(f.r); after != before {
		f.t.Errorf("%s: check changed state", name)
	}
	_, approved := f.r.Approve(number, approver, reviewed)
	if fmt.Sprint(got) != fmt.Sprint(approved) || fmt.Sprint(got) != fmt.Sprint(want) {
		f.t.Errorf("%s: check %v approve %v want %v", name, got, approved, want)
	}
}

func (f world) propose(c Change, by Actor) ProposalView {
	f.t.Helper()
	p, err := f.r.Propose(c, by)
	f.must(err)
	return p
}

func (f world) deleteAll(id string, i int, store string, ids ...string) {
	f.t.Helper()
	c, err := f.r.Claim(id, i, worker)
	f.must(err)
	f.must(f.r.Report(id, i, c.Token, store, ids, nil, worker))
}

func planOf(class string, ids ...string) Change {
	return Change{Kind: ChangePlan, Plan: DeletionPlan{Class: class, Artifacts: ids, Chunk: 1, Lease: time.Hour, Reason: "expired"}}
}

func releaseOf(hold string) Change {
	return Change{Kind: ChangeRelease, Release: Release{Hold: hold, Reason: "done"}}
}

func TestValidTextAgreesWithPlaceHold(t *testing.T) {
	f := setup(t)
	f.put("a1", ClassSource, 1, one)
	cases := []string{"", "a", strings.Repeat("a", maxTextBytes), strings.Repeat("a", maxTextBytes+1), strings.Repeat("a", maxTextBytes-1) + "é",
		strings.Repeat("a", maxTextBytes-2) + "é", "a\x1f", "a\x7f", "a\u0085", "a\u009f", "a ", "\xff", "tab\t", "line\n", " ", "한국어", "العربية"}
	for index, text := range cases {
		_, err := f.r.PlaceHold(Hold{ID: "h" + itoa(index), Owner: "ann", Reason: text, Artifacts: []string{"a1"}}, ann)
		if ValidText(text) != (err == nil) {
			t.Errorf("%q: valid %v, place %v", text, ValidText(text), err)
		}
		if err != nil && err != ErrInvalid {
			t.Errorf("%q: %v", text, err)
		}
	}
}

func TestCheckMatchesPropose(t *testing.T) {
	f := setup(t)
	for _, id := range []string{"s1", "s2", "s3", "s4", "s5"} {
		f.put(id, ClassSource, 100, one)
	}
	f.put("f1", ClassFinal, 100, one)
	f.policy(ClassSource, 90, 0)
	f.agree("malformed", Change{Kind: "other"}, ann, ErrInvalid)
	f.agree("malformed policy", Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 0}}, ann, ErrInvalid)
	f.agree("malformed release", Change{Kind: ChangeRelease, Release: Release{Hold: "h1"}}, ann, ErrInvalid)
	f.agree("new policy", Change{Kind: ChangePolicy, Policy: Policy{Class: ClassFinal, Period: 10}}, ann, nil)
	f.agree("shortening proposal", Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 10}}, ann, nil)
	f.agree("plan unknown", planOf(ClassSource, "zz"), ann, ErrNotFound)
	f.agree("plan class", planOf(ClassSource, "f1"), ann, ErrInvalid)
	f.agree("plan", planOf(ClassSource, "s1"), ann, nil)
	gone := f.plan(ClassSource, 1, "s2")
	f.deleteAll(gone, 0, "db", "s2")
	f.agree("plan deleted", planOf(ClassSource, "s2"), ann, refused(ReasonAlreadyDeleted))
	f.plan(ClassSource, 1, "s1")
	f.agree("plan in run", planOf(ClassSource, "s1"), ann, refused(ReasonInPlan))
	f.r.unfinished = maxUnfinished
	f.agree("plan limit", planOf(ClassSource, "s3"), ann, ErrLimit)
	f.r.unfinished = 1
	f.agree("release unknown", releaseOf("h1"), ann, ErrNotFound)
	f.must(second(f.r.PlaceHold(Hold{ID: "h1", Owner: "ann", Reason: "dispute", Artifacts: []string{"s3"}}, ann)))
	f.agree("release by placer", releaseOf("h1"), ann, refused(ReasonPlacer))
	f.agree("release", releaseOf("h1"), bob, nil)
	f.by(releaseOf("h1"), bob, cat)
	f.agree("release released", releaseOf("h1"), bob, refused(ReasonReleased))
	if f.r.Check(planOf(ClassSource, "s3"), "", "", Impact{}) != nil || f.r.Check(releaseOf("h1"), "", "", Impact{}) == nil {
		t.Error("check without actors")
	}
}

func TestCheckMatchesApprove(t *testing.T) {
	f := setup(t)
	for _, id := range []string{"s1", "s2", "s3", "s4"} {
		f.put(id, ClassSource, 100, one)
	}
	f.put("s5", ClassSource, 40, one)
	f.policy(ClassSource, 90, 0)
	short := f.propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 30}}, ann)
	if short.Change.Impact.Count != 1 {
		t.Fatalf("impact %+v", short.Change.Impact)
	}
	f.agreeApprove("no impact", short.Number, bob, Impact{}, refused(ReasonImpactRequired))
	wrong := short.Change.Impact
	wrong.Count++
	f.agreeApprove("wrong count", short.Number, bob, wrong, refused(ReasonImpactChanged))
	wrong = short.Change.Impact
	wrong.Digest = digestIDs([]string{"s4"})
	f.agreeApprove("wrong digest", short.Number, bob, wrong, refused(ReasonImpactChanged))
	future := short.Change.Impact
	future.AsOf = f.at().Add(time.Minute)
	f.agreeApprove("future impact", short.Number, bob, future, refused(ReasonImpactRequired))
	f.advance(impactWindow + time.Second)
	f.agreeApprove("stale impact", short.Number, bob, short.Change.Impact, refused(ReasonImpactRequired))
	fresh, err := f.r.Impact(short.Change.Policy)
	f.must(err)
	f.agreeApprove("fresh impact", short.Number, bob, fresh.Impact, nil)
	lengthen := f.propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 60}}, ann)
	f.agreeApprove("lengthening", lengthen.Number, bob, Impact{}, nil)
	f.must(second(f.r.PlaceHold(Hold{ID: "h1", Owner: "cat", Reason: "dispute", Artifacts: []string{"s1"}}, cat)))
	rel := f.propose(releaseOf("h1"), bob)
	f.agreeApprove("placer approves", rel.Number, cat, Impact{}, refused(ReasonPlacer))
	other := f.propose(releaseOf("h1"), ann)
	f.agreeApprove("release", rel.Number, dan, Impact{}, nil)
	f.agreeApprove("already released", other.Number, dan, Impact{}, refused(ReasonReleased))
	pending := f.propose(planOf(ClassSource, "s2"), ann)
	f.by(planOf(ClassSource, "s2"), bob, cat)
	f.agreeApprove("plan in run", pending.Number, bob, Impact{}, refused(ReasonInPlan))
	limited := f.propose(planOf(ClassSource, "s3"), ann)
	f.r.unfinished = maxUnfinished
	f.agreeApprove("plan limit", limited.Number, bob, Impact{}, ErrLimit)
	f.r.unfinished = 1
	f.agreeApprove("plan", limited.Number, bob, Impact{}, nil)
}

func TestStanding(t *testing.T) {
	f := setup(t)
	two := []string{"t2"}
	f.put("c1", ClassClaim, 500, one)
	f.put("s-new", ClassSource, 10, one)
	f.put("s-due", ClassSource, 100, one)
	f.put("s-held", ClassSource, 100, one)
	f.put("s-t2", ClassSource, 100, two)
	f.put("s-rel", ClassSource, 100, one)
	f.put("s-ref", ClassSource, 100, one)
	f.put("s-a", ClassSource, 100, one)
	f.put("s-b", ClassSource, 100, one, "db", "obj")
	f.put("s-c", ClassSource, 100, one)
	f.put("f1", ClassFinal, 100, one)
	f.must(f.r.Refer(Reference{From: "f1", To: "s-ref", Kind: Requires}, ingest))
	f.policy(ClassSource, 90, 5)
	f.must(second(f.r.PlaceHold(Hold{ID: "h-art", Owner: "ann", Reason: "dispute", Artifacts: []string{"s-held"}}, ann)))
	f.must(second(f.r.PlaceHold(Hold{ID: "h-ten", Owner: "ann", Reason: "dispute", Tenants: two, Classes: []string{ClassSource}}, ann)))
	f.must(second(f.r.PlaceHold(Hold{ID: "h-rel", Owner: "ann", Reason: "dispute", Artifacts: []string{"s-rel"}}, ann)))
	f.by(releaseOf("h-rel"), bob, cat)
	run := f.plan(ClassSource, 1, "s-a", "s-b", "s-c")
	shown := func(id string) string {
		t.Helper()
		before := snapshot(f.r)
		v, err := f.r.Standing(id)
		f.must(err)
		if snapshot(f.r) != before {
			t.Errorf("standing of %s changed state", id)
		}
		if access := f.access(id); access != v.Access {
			t.Errorf("%s: standing %s access %s", id, v.Access, access)
		}
		due := "-"
		if !v.DueAt.IsZero() {
			due = fmt.Sprint(int(f.at().Sub(v.DueAt) / day))
		}
		return fmt.Sprint(v.Access, " ", v.Due, " ", due, " ", v.Held, " ", v.Referenced, " ", v.Damaged, " ", v.Run, " ", v.Stores)
	}
	want := map[string]string{
		"c1":     "ok false - false false false  map[]",
		"s-new":  "ok false -85 false false false  map[]",
		"s-due":  "ok true 5 false false false  map[]",
		"s-held": "ok true 5 true false false  map[]",
		"s-t2":   "ok true 5 true false false  map[]",
		"s-rel":  "ok true 5 false false false  map[]",
		"s-ref":  "ok true 5 false true false  map[]",
		"s-a":    "blocked true 5 false false false " + run + " map[]",
		"s-b":    "blocked true 5 false false false " + run + " map[]",
		"f1":     "ok false - false false false  map[]",
	}
	for id, w := range want {
		if got := shown(id); got != w {
			t.Errorf("%s: %s, want %s", id, got, w)
		}
	}
	c, err := f.r.Claim(run, 0, worker)
	f.must(err)
	if got := shown("s-a"); got != "blocked true 5 false false false "+run+" map[]" {
		t.Errorf("deleting %s", got)
	}
	f.must(f.r.Report(run, 0, c.Token, "db", []string{"s-a"}, nil, worker))
	if got := shown("s-a"); got != "deleted true 5 false false true  map[db:deleted]" {
		t.Errorf("deleted %s", got)
	}
	c, err = f.r.Claim(run, 1, worker)
	f.must(err)
	f.must(f.r.Report(run, 1, c.Token, "db", []string{"s-b"}, nil, worker))
	f.must(f.r.Report(run, 1, c.Token, "obj", nil, []string{"s-b"}, worker))
	if got := shown("s-b"); got != "blocked true 5 false false true "+run+" map[db:deleted obj:failed]" {
		t.Errorf("failed %s", got)
	}
	if got := shown("s-c"); got != "blocked true 5 false false false "+run+" map[]" {
		t.Errorf("pending %s", got)
	}
	f.policy(ClassFinal, 1, 0)
	gone := f.plan(ClassFinal, 1, "f1")
	f.deleteAll(gone, 0, "db", "f1")
	if got := shown("s-ref"); got != "ok true 5 false false false  map[]" {
		t.Errorf("requirer deleted %s", got)
	}
	f.must(second(f.r.PlaceHold(Hold{ID: "h-out", Owner: "ann", Reason: "late", Artifacts: []string{"s-a", "s-b"}}, ann)))
	outcomes, err := f.r.HoldOutcomes("h-out", "", 10)
	f.must(err)
	if len(outcomes) != 2 {
		t.Fatalf("outcomes %+v", outcomes)
	}
	for _, o := range outcomes {
		v, err := f.r.Standing(o.Artifact)
		f.must(err)
		if fmt.Sprint(v.Stores) != fmt.Sprint(o.Stores) {
			t.Errorf("%s: standing %v outcomes %v", o.Artifact, v.Stores, o.Stores)
		}
	}
	v, err := f.r.Standing("s-b")
	f.must(err)
	v.Stores["db"] = "changed"
	delete(v.Stores, "obj")
	if got := shown("s-b"); got != "blocked true 5 true false true "+run+" map[db:deleted obj:failed]" {
		t.Errorf("copy %s", got)
	}
	preview, err := f.r.Preview(ClassSource, "", 100)
	f.must(err)
	eligible, held, referenced := []string{}, 0, 0
	for _, id := range f.r.ids(ClassSource) {
		v, err := f.r.Standing(id)
		f.must(err)
		switch {
		case v.Access == AccessDeleted || v.Run != "" || !v.Due:
		case v.Held:
			held++
		case v.Referenced:
			referenced++
		default:
			eligible = append(eligible, id)
		}
	}
	if fmt.Sprint(eligible, held, referenced) != fmt.Sprint(preview.Eligible, preview.Held, preview.Referenced) {
		t.Errorf("standing %v %d %d preview %+v", eligible, held, referenced, preview)
	}
	if _, err := f.r.Standing("zz"); err != ErrNotFound {
		t.Errorf("unknown %v", err)
	}
}
