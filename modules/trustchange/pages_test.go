package trustchange

import (
	"reflect"
	"testing"
)

func mixed(t *testing.T) world {
	t.Helper()
	f := setup(t)
	if err := f.r.Restore([]View{{ID: "trust-11111111-7", Workspace: "w1", State: StateActivated, Activation: &Activation{Epoch: 1}}}); err != nil {
		t.Fatal(err)
	}
	for n, who := range []string{"ann", "bob", "cat", "ann", "bob", "cat", "ann"} {
		v := f.propose(who, 30+n)
		switch n {
		case 1:
			f.r.Cancel(v.ID, "w1", who, v.Revision, "")
		case 2:
			f.validated("dan", v)
		case 3:
			f.r.Reject(v.ID, "w1", "dan", v.Revision, "no")
		case 4:
			f.approved("eve", f.validated("dan", v))
		case 5:
			x := f.approved("eve", f.validated("dan", v))
			if _, err := f.r.Activate(approval(x, "fay"), func(View) Outcome { return Outcome{Unknown: "ledger_unconfirmed"} }); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := f.r.Propose(Proposal{Workspace: "w2", Route: RouteLocalPassword, Proposer: "ann", Candidate: candidate(9), Base: base0}); err != nil {
		t.Fatal(err)
	}
	return f
}

func ids(views []View) []string {
	out := []string{}
	for _, v := range views {
		out = append(out, v.ID)
	}
	return out
}

func walk(t *testing.T, r *Registry, q Query) []View {
	t.Helper()
	out := []View{}
	total := r.Page(q).Total
	for {
		p := r.Page(q)
		if !p.Known || len(p.Views) > q.Limit || p.Total != total {
			t.Fatalf("page %+v; total on the first page %d", p, total)
		}
		out = append(out, p.Views...)
		if !p.More {
			return out
		}
		if len(p.Views) == 0 {
			t.Fatal("more without rows")
		}
		q.After = p.Views[len(p.Views)-1].ID
	}
}

func TestPagesAgreeWithTheWholeList(t *testing.T) {
	f := mixed(t)
	whole := f.r.List("w1")
	if len(whole) != 8 {
		t.Fatalf("fixture has %d", len(whole))
	}
	seen := map[string]bool{}
	for _, v := range whole {
		seen[v.State] = true
	}
	for _, state := range []string{StateDraft, StateValidated, StateApproved, StateActivated, StateRejected, StateCancelled, StateUnknown} {
		if !seen[state] {
			t.Fatalf("fixture lacks %s", state)
		}
	}
	filters := map[string][]string{"all": nil, "draft": {StateDraft}, "waiting": {StateDraft, StateValidated, StateApproved, StateUnknown},
		"ended": {StateActivated, StateRejected, StateCancelled, StateSuperseded}, "none": {StateSuperseded}}
	for name, states := range filters {
		for _, match := range []string{"", "11111111", "-3", "zz"} {
			q := Query{Workspace: "w1", States: states, Match: match}
			want := []View{}
			for _, v := range whole {
				if q.admits(&v) {
					want = append(want, v)
				}
			}
			for _, limit := range []int{1, 2, 3, MaxPage} {
				q.Limit = limit
				got := walk(t, f.r, q)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s %q limit %d: %v; want %v", name, match, limit, ids(got), ids(want))
				}
				if p := f.r.Page(q); p.Total != len(want) || p.More != (len(want) > limit) {
					t.Errorf("%s %q limit %d: total %d more %v; want %d", name, match, limit, p.Total, p.More, len(want))
				}
			}
		}
	}
	if got := walk(t, f.r, Query{Workspace: "w1", Limit: 3, Match: "-3"}); len(got) != 1 {
		t.Errorf("a search that matches one found %v", ids(got))
	}
	drafts := walk(t, f.r, Query{Workspace: "w1", Limit: 3, States: []string{StateDraft}})
	if len(drafts) != 2 {
		t.Fatalf("drafts %v", ids(drafts))
	}
	if got := walk(t, f.r, Query{Workspace: "w1", Limit: 2, States: filters["waiting"]}); len(got) != 5 {
		t.Errorf("waiting %v; want five", ids(got))
	}
	if got := walk(t, f.r, Query{Workspace: "w1", Limit: 2, States: filters["ended"]}); len(got) != 3 {
		t.Errorf("ended %v; want three", ids(got))
	}
	var rejected View
	for _, v := range whole {
		if v.State == StateRejected {
			rejected = v
		}
	}
	p := f.r.Page(Query{Workspace: "w1", Limit: 5, States: []string{StateDraft}, After: rejected.ID})
	if !p.Known || p.Total != 2 || len(p.Views) != 1 || p.Views[0].ID != drafts[1].ID || p.More {
		t.Errorf("after a kept change the filter does not admit: %+v; want the older draft %s", p, drafts[1].ID)
	}
}

func TestAPageStaysInsideItsWorkspaceAndItsLimits(t *testing.T) {
	f := mixed(t)
	other := f.r.List("w2")
	if len(other) != 1 {
		t.Fatalf("fixture %d", len(other))
	}
	if p := f.r.Page(Query{Workspace: "w2", Limit: 5}); p.Total != 1 || len(p.Views) != 1 || p.Views[0].ID != other[0].ID {
		t.Errorf("the other workspace %+v", p)
	}
	if p := f.r.Page(Query{Workspace: "w3", Limit: 5}); !p.Known || p.Total != 0 || len(p.Views) != 0 || p.More {
		t.Errorf("an empty workspace %+v", p)
	}
	for name, q := range map[string]Query{
		"an anchor of another workspace": {Workspace: "w1", Limit: 5, After: other[0].ID},
		"an anchor that does not exist":  {Workspace: "w1", Limit: 5, After: "trust-0a1b2c3d-999"},
		"no limit":                       {Workspace: "w1"},
		"a limit above the largest":      {Workspace: "w1", Limit: MaxPage + 1},
	} {
		if p := f.r.Page(q); p.Known || len(p.Views) != 0 || p.More || p.Total != 0 {
			t.Errorf("%s: %+v", name, p)
		}
	}
	if a, b := f.r.Page(Query{Workspace: "w1", Limit: 5, After: other[0].ID}), f.r.Page(Query{Workspace: "w1", Limit: 5, After: "trust-0a1b2c3d-999"}); !reflect.DeepEqual(a, b) {
		t.Errorf("an anchor of another workspace answers %+v, an unknown one %+v", a, b)
	}
	page := f.r.Page(Query{Workspace: "w1", Limit: 2})
	page.Views[0].History[0].By = "changed"
	page.Views[0].Candidate.Methods[0] = "changed"
	if again := f.r.Page(Query{Workspace: "w1", Limit: 2}); again.Views[0].History[0].By == "changed" || again.Views[0].Candidate.Methods[0] == "changed" {
		t.Error("a page hands out the registry's own rows")
	}
}

func TestAnAnchorThatWasPrunedIsNotKnown(t *testing.T) {
	f := setup(t)
	first := f.propose("ann", 1)
	f.r.Cancel(first.ID, "w1", "ann", 1, "")
	if p := f.r.Page(Query{Workspace: "w1", Limit: 5, After: first.ID}); !p.Known || len(p.Views) != 0 {
		t.Fatalf("before pruning %+v", p)
	}
	for n := 0; n < MaxKept; n++ {
		v := f.propose("ann", 2+n)
		f.r.Cancel(v.ID, "w1", "ann", 1, "")
	}
	if _, kept := f.r.Get(first.ID, "w1"); kept {
		t.Fatal("the first change was not pruned")
	}
	if p := f.r.Page(Query{Workspace: "w1", Limit: 5, After: first.ID}); p.Known {
		t.Errorf("a pruned anchor %+v", p)
	}
}

func TestTheVersionAdvancesOnEveryChange(t *testing.T) {
	f := setup(t)
	last := f.r.Version()
	step := func(name string) {
		t.Helper()
		now := f.r.Version()
		if now <= last {
			t.Errorf("%s left the version at %d", name, now)
		}
		if p := f.r.Page(Query{Workspace: "w1", Limit: 1}); p.Version != now || f.r.Summary("w1").Version != now {
			t.Errorf("%s: the reads state another version", name)
		}
		last = now
	}
	still := func(name string) {
		t.Helper()
		if now := f.r.Version(); now != last {
			t.Errorf("%s moved the version to %d", name, now)
		}
	}
	a := f.propose("ann", 30)
	step("propose")
	f.r.List("w1")
	f.r.Get(a.ID, "w1")
	f.r.Page(Query{Workspace: "w1", Limit: 5})
	f.r.Summary("w1")
	still("reading")
	refused(t, "a refused command", second(f.r.Cancel(a.ID, "w1", "bob", a.Revision, "")), ReasonNotProposer)
	still("a refused command")
	a, _ = f.r.Revise(a.ID, "w1", "ann", a.Revision, candidate(31), base0, "again")
	step("revise")
	a = f.validated("dan", a)
	step("validate")
	a = f.approved("eve", a)
	step("approve")
	b := f.propose("bob", 40)
	step("a second proposal")
	f.r.Reject(b.ID, "w1", "dan", b.Revision, "no")
	step("reject")
	c := f.propose("bob", 41)
	step("a third proposal")
	f.r.Cancel(c.ID, "w1", "bob", c.Revision, "")
	step("cancel")
	if _, err := f.r.Activate(approval(a, "fay"), receipt(2)); err != nil {
		t.Fatal(err)
	}
	step("activate")
	if err := f.r.Restore([]View{{ID: "trust-11111111-1", Workspace: "w1", State: StateActivated}}); err != nil {
		t.Fatal(err)
	}
	step("restore")
	if err := f.r.Restore([]View{{ID: "x", State: StateActivated}}); err != ErrInvalid {
		t.Fatal(err)
	}
	still("a refused restore")
	g := setup(t)
	for n := 0; n < MaxKept; n++ {
		v := g.propose("ann", 1+n)
		g.r.Cancel(v.ID, "w1", "ann", 1, "")
	}
	v := g.propose("ann", 900)
	before := g.r.Version()
	if len(g.r.order) != MaxKept {
		t.Fatalf("kept %d", len(g.r.order))
	}
	g.r.Cancel(v.ID, "w1", "ann", 1, "")
	if g.r.Version() != before+1 {
		t.Errorf("a cancel without pruning moved the version by %d", g.r.Version()-before)
	}
	g.propose("ann", 901)
	if g.r.Version() != before+3 {
		t.Errorf("a proposal that prunes one moved the version by %d; want one for the proposal and one for the pruning", g.r.Version()-before-1)
	}
}

func TestTheSummaryNamesTheNewestWaitingActivationAndEarlierRuns(t *testing.T) {
	f := setup(t)
	if s := f.r.Summary("w1"); s.Unknown != "" || s.EarlierRun {
		t.Errorf("an empty registry %+v", s)
	}
	g := mixed(t)
	var want string
	for _, v := range g.r.List("w1") {
		if v.State == StateUnknown && want == "" {
			want = v.ID
		}
	}
	if s := g.r.Summary("w1"); want == "" || s.Unknown != want || !s.EarlierRun {
		t.Errorf("summary %+v; want %s from an earlier run", s, want)
	}
	if s := g.r.Summary("w2"); s.Unknown != "" || s.EarlierRun {
		t.Errorf("another workspace %+v", s)
	}
	for _, v := range g.r.List("w1") {
		if v.State == StateApproved {
			if _, err := g.r.Activate(approval(v, "fay"), func(View) Outcome { return Outcome{Unknown: "ledger_unconfirmed"} }); err != nil {
				t.Fatal(err)
			}
		}
	}
	waiting := walk(t, g.r, Query{Workspace: "w1", Limit: 5, States: []string{StateUnknown}})
	if s := g.r.Summary("w1"); len(waiting) != 2 || s.Unknown != want || s.Unknown != waiting[0].ID {
		t.Errorf("with two waiting activations %+v; want the newest of %v", s, ids(waiting))
	}
	h := setup(t)
	h.propose("ann", 5)
	if s := h.r.Summary("w1"); s.Unknown != "" || s.EarlierRun {
		t.Errorf("only this run's draft %+v", s)
	}
	if err := h.r.Restore([]View{{ID: "trust-11111111-2", Workspace: "w1", State: StateActivated}}); err != nil {
		t.Fatal(err)
	}
	h.propose("bob", 6)
	if s := h.r.Summary("w1"); !s.EarlierRun {
		t.Errorf("an earlier run's change between two of this run %+v", s)
	}
}
