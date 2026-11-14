package cases

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

type modelEntry struct {
	id, kind, body, actor, side string
}

type modelStep struct {
	kind, actor, owner, reason, outcome string
	due                                 time.Time
}

type modelCase struct {
	kind, state           string
	awaiting              bool
	due, original         time.Time
	entries               []modelEntry
	public, notes         map[string]int
	steps                 map[[2]string]modelStep
	counted, msgs, others int
	closings              int
	opener                Actor
}

type model struct {
	cases     map[[2]string]*modelCase
	customers map[string]int
	limits    Limits
}

func (m *model) expectCall(r *Registry, now time.Time, rng *rand.Rand, actors []Actor) (string, string, error) {
	actor := actors[rng.Intn(len(actors))]
	customer := []string{"c1", "c2"}[rng.Intn(2)]
	id := []string{"a", "b", "c"}[rng.Intn(3)]
	key := [2]string{customer, id}
	c := m.cases[key]
	hidden := actor.Side == SideCustomer && actor.Customer != customer
	operator := actor.Side == SideOperator
	live := c != nil && (c.state == CaseOpen || c.state == CaseAnswered)
	switch op := rng.Intn(9); op {
	case 0:
		k := []string{KindInquiry, KindDispute}[rng.Intn(2)]
		body := digest("12"[rng.Intn(2)])
		cs := Case{ID: id, Kind: k, Customer: customer, Subjects: []Subject{{Document: "inv"}}, Topic: "amount", Body: body}
		_, err := r.OpenCase("ws", cs, actor)
		want := "ok"
		switch {
		case hidden:
			want = "not_found"
		case c != nil:
			if c.kind != k || c.opener != actor || c.entries[0].body != body {
				want = "conflict"
			}
		case len(m.cases) >= m.limits.Cases || m.customers[customer] >= m.limits.CustomerCases:
			want = "limit"
		default:
			m.cases[key] = &modelCase{kind: k, state: CaseOpen, awaiting: true, public: map[string]int{}, notes: map[string]int{}, steps: map[[2]string]modelStep{}, opener: actor,
				entries: []modelEntry{{body: body}}}
			m.customers[customer]++
		}
		return "open", want, err
	case 1, 2, 3:
		entryKind := []string{EntryMessage, EntryAnswer, EntryNote}[op-1]
		eid := []string{"e1", "e2", "e3", "e4"}[rng.Intn(4)]
		body := digest("34"[rng.Intn(2)])
		var err error
		switch entryKind {
		case EntryMessage:
			_, err = r.Reply("ws", customer, id, actor, Entry{ID: eid, Body: body})
		case EntryAnswer:
			_, err = r.Answer("ws", customer, id, actor, Entry{ID: eid, Body: body})
		default:
			_, err = r.Note("ws", customer, id, actor, Entry{ID: eid, Body: body})
		}
		if c == nil || hidden {
			return entryKind, "not_found", err
		}
		space := c.public
		if entryKind == EntryNote {
			space = c.notes
			if !operator {
				return entryKind, "rejected", err
			}
		}
		if index, ok := space[eid]; ok {
			old := c.entries[index]
			if old.kind != entryKind || old.body != body || old.actor != actor.ID || old.side != actor.Side {
				return entryKind, "conflict", err
			}
			return entryKind, "ok", err
		}
		if (entryKind == EntryMessage) != !operator || entryKind != EntryNote && !live {
			return entryKind, "rejected", err
		}
		if entryKind == EntryMessage && c.msgs >= m.limits.Messages || entryKind != EntryMessage && c.others >= m.limits.Entries {
			return entryKind, "limit", err
		}
		space[eid] = len(c.entries)
		c.entries = append(c.entries, modelEntry{id: eid, kind: entryKind, body: body, actor: actor.ID, side: actor.Side})
		switch entryKind {
		case EntryAnswer:
			c.state, c.awaiting, c.others = CaseAnswered, false, c.others+1
		case EntryNote:
			c.others++
		default:
			if c.state == CaseAnswered {
				c.due = time.Time{}
			}
			c.state, c.awaiting, c.msgs = CaseOpen, true, c.msgs+1
		}
		return entryKind, "ok", err
	default:
		kinds := []string{caseAssign, caseDue, caseClose, caseReopen, caseWithdraw}
		kind := kinds[op-4]
		st := CaseStep{Step: []string{"s1", "s2", "s3"}[rng.Intn(3)]}
		switch kind {
		case caseAssign:
			st.Owner = []string{"o1", "o2"}[rng.Intn(2)]
		case caseDue:
			st.Due = now.Add(time.Duration(1+rng.Intn(3)) * time.Hour)
			st.Reason = []string{"initial", "follow_up"}[rng.Intn(2)]
		case caseClose:
			if c != nil && c.kind == KindDispute || c == nil && rng.Intn(2) == 0 {
				st.Closing = Closing{Outcome: OutcomeNotUpheld}
			} else {
				st.Closing = Closing{Outcome: OutcomeAnswered}
			}
		}
		_, err := r.step("ws", customer, id, actor, kind, st)
		if c == nil || hidden {
			return kind, "not_found", err
		}
		sk := [2]string{actor.Side, st.Step}
		if old, ok := c.steps[sk]; ok {
			if old.kind != kind || old.actor != actor.ID || old.owner != st.Owner || old.reason != st.Reason || !old.due.Equal(st.Due) || old.outcome != st.Closing.Outcome {
				return kind, "conflict", err
			}
			return kind, "ok", err
		}
		allowed := false
		switch kind {
		case caseAssign:
			allowed = operator && live
		case caseDue:
			allowed = operator && live && (st.Reason == "initial") == c.original.IsZero()
		case caseClose:
			allowed = operator && live && (st.Closing.Outcome == OutcomeAnswered) == (c.kind == KindInquiry)
		case caseReopen:
			allowed = c.state == CaseClosed
		case caseWithdraw:
			allowed = !operator && live
		}
		if !allowed {
			return kind, "rejected", err
		}
		if kind != caseClose && kind != caseWithdraw && c.counted >= m.limits.Steps {
			return kind, "limit", err
		}
		c.steps[sk] = modelStep{kind: kind, actor: actor.ID, owner: st.Owner, reason: st.Reason, outcome: st.Closing.Outcome, due: st.Due}
		switch kind {
		case caseAssign:
			c.counted++
		case caseDue:
			if c.original.IsZero() {
				c.original = st.Due
			}
			c.due, c.counted = st.Due, c.counted+1
		case caseClose:
			c.state = CaseClosed
			c.closings++
		case caseReopen:
			c.state, c.awaiting, c.due, c.counted = CaseOpen, true, time.Time{}, c.counted+1
		case caseWithdraw:
			c.state = CaseWithdrawn
		}
		return kind, "ok", err
	}
}

func (m *model) check(t *testing.T, r *Registry, now time.Time, actors []Actor, trace string) {
	for key, c := range m.cases {
		for _, actor := range actors {
			v, err := r.Case("ws", key[0], key[1], actor)
			if actor.Side == SideCustomer && actor.Customer != key[0] {
				if kind(err) != "not_found" {
					t.Fatalf("%s: %v sees %v", trace, actor, key)
				}
				continue
			}
			if err != nil || v.State != c.state || !v.Due.Equal(c.due) || len(v.Closings) != map[bool]int{true: c.closings}[actor.Side == SideOperator] {
				t.Fatalf("%s: %v %+v vs %+v %v", trace, key, v, c, err)
			}
			overdue := c.state == CaseOpen && c.awaiting && !c.due.IsZero() && !now.Before(c.due)
			if v.ResponseOverdue != overdue {
				t.Fatalf("%s: overdue %v", trace, key)
			}
			want := []modelEntry{}
			for _, e := range c.entries[1:] {
				if actor.Side == SideOperator || e.kind != EntryNote {
					want = append(want, e)
				}
			}
			if len(v.Entries) != len(want) {
				t.Fatalf("%s: entries %v %d %d", trace, key, len(v.Entries), len(want))
			}
			for index, e := range v.Entries {
				w := want[index]
				if e.ID != w.id || e.Kind != w.kind || e.Body != w.body || e.Side != w.side {
					t.Fatalf("%s: entry %+v %+v", trace, e, w)
				}
				if actor.Side == SideCustomer && (e.Kind == EntryNote || strings.HasPrefix(e.Actor, "op") || e.Side == SideOperator && e.Digest != "") {
					t.Fatalf("%s: a customer sees internal data %+v", trace, e)
				}
			}
			if actor.Side == SideCustomer && (strings.HasPrefix(v.OpenedBy, "op") || v.Owner != "" || v.ClosedBy != "" || v.DueHistory != nil || v.Digest != "" || !v.OriginalDue.IsZero()) {
				t.Fatalf("%s: a customer view shows internal fields %+v", trace, v)
			}
			if actor.Side == SideOperator && !v.OriginalDue.Equal(c.original) {
				t.Fatalf("%s: original due", trace)
			}
		}
	}
}

func TestCasesAgainstAModel(t *testing.T) {
	actors := []Actor{op("op1"), op("op2"), cust("u1", "c1"), cust("u2", "c1"), cust("u3", "c2")}
	for seed := int64(1); seed <= 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		l := Limits{Workspaces: 2, Findings: 2, Cases: 5, CustomerCases: 3, Steps: 3, Messages: 3, Entries: 3, Subjects: 2, Page: 50}
		r, clock := openWith(t, l)
		m := &model{cases: map[[2]string]*modelCase{}, customers: map[string]int{}, limits: l}
		trace := fmt.Sprintf("seed %d", seed)
		for step := 0; step < 120; step++ {
			if rng.Intn(5) == 0 {
				clock.advance(time.Duration(rng.Intn(90)) * time.Minute)
			}
			now := clock.read()
			name, want, err := m.expectCall(r, now, rng, actors)
			if kind(err) != want {
				t.Fatalf("%s step %d %s: want %s got %s", trace, step, name, want, kind(err))
			}
			m.check(t, r, clock.read(), actors, fmt.Sprintf("%s step %d %s", trace, step, name))
		}
	}
}

func TestFindingsAgainstAModel(t *testing.T) {
	legal := map[string]map[string]string{
		StepConfirm: {FindingInReview: FindingConfirmedNormal},
		StepAdjust:  {FindingInReview: FindingAdjustmentNeeded},
		StepResolve: {FindingAdjustmentNeeded: FindingResolved},
		StepAssign:  {FindingInReview: FindingInReview, FindingConfirmedNormal: FindingConfirmedNormal, FindingAdjustmentNeeded: FindingAdjustmentNeeded, FindingResolved: FindingResolved},
	}
	reopen := map[[2]string]bool{{FindingConfirmedNormal, "new_evidence"}: true, {FindingConfirmedNormal, "decision_revised"}: true, {FindingAdjustmentNeeded, "new_evidence"}: true,
		{FindingAdjustmentNeeded, "decision_revised"}: true, {FindingResolved, "new_evidence"}: true, {FindingResolved, "resolution_failed"}: true}
	reasons := map[string][]string{StepConfirm: {"duplicate", "expected_change"}, StepAdjust: {"rate_error"}, StepReopen: {"new_evidence", "decision_revised", "resolution_failed"}}
	kinds := []string{StepConfirm, StepAdjust, StepResolve, StepReopen, StepAssign}
	actors := []Actor{op("a"), op("b"), op("c")}
	for seed := int64(1); seed <= 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		r, _ := openWith(t, Limits{Workspaces: 1, Findings: 3, Cases: 1, CustomerCases: 1, Steps: 3, Messages: 1, Entries: 1, Subjects: 1, Page: 10})
		type mf struct {
			state, author, reopener, owner string
			counted, steps                 int
			seen                           map[string]Transition
			kinds                          map[string]string
			actors                         map[string]string
		}
		fs := map[string]*mf{}
		for step := 0; step < 80; step++ {
			id := []string{"x", "y"}[rng.Intn(2)]
			actor := actors[rng.Intn(len(actors))]
			f := fs[id]
			if f == nil || rng.Intn(10) == 0 {
				_, err := r.RecordFinding("ws", spike(id, "c"), actor)
				want := "ok"
				if f != nil && f.author != actor.ID {
					want = "conflict"
				}
				if f == nil {
					fs[id] = &mf{state: FindingInReview, author: actor.ID, owner: "owner-1", seen: map[string]Transition{}, kinds: map[string]string{}, actors: map[string]string{}}
				}
				if kind(err) != want {
					t.Fatalf("seed %d record: %s %s", seed, want, kind(err))
				}
				continue
			}
			k := kinds[rng.Intn(len(kinds))]
			tr := Transition{Step: []string{"p", "q", "s", "u"}[rng.Intn(4)]}
			switch k {
			case StepAssign:
				tr.Owner = []string{"o1", "o2"}[rng.Intn(2)]
			case StepResolve:
				tr.Evidence, tr.Remedy = digest('e'), Remedy{Kind: "source_correction", Reference: "fix"}
			default:
				tr.Evidence, tr.Reason = digest('e'), reasons[k][rng.Intn(len(reasons[k]))]
			}
			v, err := r.move("ws", id, actor, k, tr)
			want := "ok"
			to, allowed := legal[k][f.state]
			if k == StepReopen {
				to, allowed = FindingInReview, reopen[[2]string{f.state, tr.Reason}]
			}
			switch {
			case f.kinds[tr.Step] != "":
				if f.kinds[tr.Step] != k || f.seen[tr.Step] != tr || f.actors[tr.Step] != actor.ID {
					want = "conflict"
				}
			case !allowed || (k == StepConfirm || k == StepAdjust) && (actor.ID == f.author || actor.ID == f.reopener):
				want = "rejected"
			case (k == StepAssign || k == StepReopen) && f.counted >= 3:
				want = "limit"
			default:
				f.kinds[tr.Step], f.seen[tr.Step], f.actors[tr.Step] = k, tr, actor.ID
				f.state = to
				f.steps++
				switch k {
				case StepAssign:
					f.owner, f.counted = tr.Owner, f.counted+1
				case StepReopen:
					f.reopener, f.counted = actor.ID, f.counted+1
				}
			}
			if kind(err) != want {
				t.Fatalf("seed %d step %d %s: want %s got %s", seed, step, k, want, kind(err))
			}
			if err == nil && (v.State != f.state || v.Owner != f.owner || len(v.Steps) != f.steps) {
				t.Fatalf("seed %d step %d: %+v vs %+v", seed, step, v, f)
			}
		}
	}
}
