package hawkeye

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func errName(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "notfound"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "unknown"
}

func units(text string) int64 {
	whole, fraction, _ := strings.Cut(text, ".")
	for len(fraction) < 4 {
		fraction += "0"
	}
	var value int64
	for _, c := range whole + fraction {
		value = value*10 + int64(c-'0')
	}
	return value
}

type mStatement struct {
	supersedes   string
	supersededBy string
	live         string
}

type mReception struct {
	spec      Spec
	state     State
	wasClosed bool
	lines     map[string]string
	posted    map[string]Line
	reason    VoidReason
}

type model struct {
	receptions map[string]*mReception
	statements map[string]*mStatement
	sequence   map[string]uint64
}

func headerPrint(s Spec) string {
	var totals []string
	for currency, total := range s.Totals {
		totals = append(totals, fmt.Sprintf("%s=%d", currency, units(total)))
	}
	sort.Strings(totals)
	return fmt.Sprint(s.Source, s.Statement, s.CostOrigin, s.PeriodStart.Unix(), s.PeriodEnd.Unix(), s.Revision, s.Supersedes, s.CorrectionReason, s.Mock, s.Status, s.Tier, s.Coverage, s.Lines, s.Declaration, totals)
}

func (m *model) open(s Spec) string {
	if earlier := m.receptions[s.ID]; earlier != nil {
		if headerPrint(earlier.spec) != headerPrint(s) {
			return "conflict"
		}
		return "ok:" + string(earlier.state)
	}
	key := s.Source + "|" + s.Statement
	known := m.statements[key]
	if known != nil && known.live != "" {
		return "conflict:" + known.live
	}
	var named *mStatement
	switch {
	case known != nil && known.supersedes != "":
		if s.Supersedes != known.supersedes {
			return "rejected"
		}
	case s.Supersedes != "":
		named = m.statements[s.Source+"|"+s.Supersedes]
		if named == nil || named.supersededBy != "" {
			return "rejected"
		}
		for at, guard := known, 0; at != nil && at.supersededBy != "" && guard < 100; at, guard = m.statements[s.Source+"|"+at.supersededBy], guard+1 {
			if at.supersededBy == s.Supersedes {
				return "rejected"
			}
		}
	}
	if known == nil {
		known = &mStatement{}
		m.statements[key] = known
	}
	if named != nil {
		known.supersedes, named.supersededBy = s.Supersedes, s.Statement
	}
	known.live = s.ID
	m.receptions[s.ID] = &mReception{spec: s, state: Open, lines: map[string]string{}, posted: map[string]Line{}}
	m.sequence[s.Source]++
	return "ok:open"
}

func (m *model) post(id string, l Line) string {
	r := m.receptions[id]
	if r == nil {
		return "notfound"
	}
	print := fmt.Sprint(units(l.Cost), l.Currency, l.Measured, l.Energy)
	if earlier, known := r.lines[l.Team]; known {
		if earlier != print {
			return "conflict"
		}
		return "ok:" + string(r.state)
	}
	if r.state != Open || m.statements[r.spec.Source+"|"+r.spec.Statement].supersededBy != "" || len(r.lines) >= r.spec.Lines {
		return "rejected"
	}
	r.lines[l.Team], r.posted[l.Team] = print, l
	m.sequence[r.spec.Source]++
	return "ok:open"
}

func (m *model) close(id string) string {
	r := m.receptions[id]
	if r == nil {
		return "notfound"
	}
	if r.wasClosed {
		return "ok:" + string(r.state)
	}
	if r.state != Open || m.statements[r.spec.Source+"|"+r.spec.Statement].supersededBy != "" {
		return "rejected"
	}
	r.state, r.wasClosed = Closed, true
	m.sequence[r.spec.Source]++
	return "ok:closed"
}

func (m *model) void(id string, reason VoidReason) string {
	r := m.receptions[id]
	if r == nil {
		return "notfound"
	}
	if r.state == Voided {
		if r.reason != reason {
			return "conflict"
		}
		return "ok"
	}
	r.state, r.reason = Voided, reason
	m.statements[r.spec.Source+"|"+r.spec.Statement].live = ""
	m.sequence[r.spec.Source]++
	return "ok"
}

func (m *model) verdict(id string, listed int) string {
	r := m.receptions[id]
	if r.state == Voided {
		return "not_eligible[voided]"
	}
	contends := func(o *mReception) bool {
		return o.state != Voided && !o.spec.Mock && m.statements[o.spec.Source+"|"+o.spec.Statement].supersededBy == ""
	}
	var reasons []string
	add := func(applies bool, reason string) {
		if applies {
			reasons = append(reasons, reason)
		}
	}
	add(r.state == Open, NotClosed)
	add(r.spec.Mock, IsMock)
	add(r.spec.Status == Provisional, IsProvisional)
	add(r.spec.Tier == Estimated || r.spec.Tier == Unavailable, NotMeasured)
	add(r.spec.Coverage <= 9499, CoverageBelow)
	add(r.spec.Declaration == Derived, DeclarationDerived)
	add(m.statements[r.spec.Source+"|"+r.spec.Statement].supersededBy != "", IsSuperseded)
	var others []string
	if contends(r) {
		for other, o := range m.receptions {
			if other != id && o.spec.Source == r.spec.Source && contends(o) && o.spec.PeriodEnd.After(r.spec.PeriodStart) && r.spec.PeriodEnd.After(o.spec.PeriodStart) {
				others = append(others, other)
			}
		}
		sort.Strings(others)
	}
	add(len(others) > 0, PeriodConflict)
	if r.state == Closed {
		sums := map[string]int64{}
		unmeasured := false
		for _, line := range r.posted {
			sums[line.Currency] += units(line.Cost)
			unmeasured = unmeasured || !line.Measured
		}
		add(unmeasured, UnmeasuredLines)
		add(len(r.posted) != r.spec.Lines, IncompleteList)
		differ := len(sums) != len(r.spec.Totals)
		for currency, sum := range sums {
			if total, declared := r.spec.Totals[currency]; !declared || units(total) != sum {
				differ = true
			}
		}
		add(differ, TotalsDiffer)
	}
	if len(reasons) == 0 {
		return "eligible[]"
	}
	text := fmt.Sprintf("not_eligible%v", reasons)
	if len(others) > 0 {
		total := len(others)
		if total > listed {
			others = others[:listed]
		}
		text += fmt.Sprintf(" with %v of %d", others, total)
	}
	return text
}

func (h *harness) recomputed(id string) string {
	h.t.Helper()
	view := h.view(id)
	if view.State == Voided {
		return "not_eligible[voided]"
	}
	var reasons []string
	add := func(applies bool, reason string) {
		if applies {
			reasons = append(reasons, reason)
		}
	}
	add(view.State == Open, NotClosed)
	add(view.Spec.Mock, IsMock)
	add(view.Spec.Status != Final, IsProvisional)
	add(view.Spec.Tier != Measured, NotMeasured)
	add(view.Spec.Coverage < 9500, CoverageBelow)
	add(view.Spec.Declaration != SourceHeader, DeclarationDerived)
	add(view.SupersededBy != "", IsSuperseded)
	var others []string
	if !view.Spec.Mock && view.SupersededBy == "" {
		for after := ""; ; {
			rows, err := h.r.Receptions("ws1", view.Spec.Source, after, h.r.limits.Page)
			if err != nil {
				h.t.Fatal(err)
			}
			if len(rows) == 0 {
				break
			}
			after = rows[len(rows)-1].ID
			for _, row := range rows {
				if row.ID != id && row.State != Voided && !row.Mock && !row.Superseded && view.Spec.PeriodStart.Before(row.PeriodEnd) && row.PeriodStart.Before(view.Spec.PeriodEnd) {
					others = append(others, row.ID)
				}
			}
		}
	}
	add(len(others) > 0, PeriodConflict)
	if view.State == Closed {
		sums := map[string]int64{}
		count, unmeasured := 0, false
		for {
			page, err := h.r.Lines("ws1", id, count, h.r.limits.Page)
			if err != nil {
				h.t.Fatal(err)
			}
			if len(page) == 0 {
				break
			}
			count += len(page)
			for _, line := range page {
				sums[line.Currency] += units(line.Cost)
				unmeasured = unmeasured || !line.Measured
			}
		}
		add(unmeasured, UnmeasuredLines)
		add(count != view.Spec.Lines, IncompleteList)
		differ := len(sums) != len(view.Spec.Totals)
		for currency, sum := range sums {
			if total, declared := view.Spec.Totals[currency]; !declared || units(total) != sum {
				differ = true
			}
		}
		add(differ, TotalsDiffer)
	}
	if len(reasons) == 0 {
		return "eligible[]"
	}
	text := fmt.Sprintf("not_eligible%v", reasons)
	if len(others) > 0 {
		shown := others
		if len(shown) > h.r.limits.Listed {
			shown = shown[:h.r.limits.Listed]
		}
		text += fmt.Sprintf(" with %v of %d", shown, len(others))
	}
	return text
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	random := rand.New(rand.NewSource(19400))
	reached := map[string]int{}
	type span struct{ from, to time.Time }
	var spans []span
	for number := 10; number <= 12; number++ {
		from, to := month(number)
		spans = append(spans, span{from, to})
	}
	half, _ := month(10)
	spans = append(spans, span{half.Add(15 * day), half.Add(31 * day)})
	for history := 0; history < 300; history++ {
		h := newHarness(t, func(l *Limits) { l.Page = 4; l.Listed = 2 })
		m := &model{receptions: map[string]*mReception{}, statements: map[string]*mStatement{}, sequence: map[string]uint64{}}
		var ids []string
		superseded := map[string]bool{}
		conflicted := map[string]bool{}
		voidedAfterSuperseding := map[string]bool{}
		step := 0
		check := func(what, got, want string) {
			t.Helper()
			if got != want {
				t.Fatalf("history %d step %d %s:\n got %s\nwant %s", history, step, what, got, want)
			}
			reached[what+" "+strings.SplitN(want, ":", 2)[0]]++
		}
		pick := func() string {
			if len(ids) == 0 || random.Intn(20) == 0 {
				return "nowhere"
			}
			if random.Intn(10) < 6 {
				var open []string
				for _, id := range ids {
					if m.receptions[id].state == Open {
						open = append(open, id)
					}
				}
				if len(open) > 0 {
					return open[random.Intn(len(open))]
				}
			}
			return ids[random.Intn(len(ids))]
		}
		for step = 0; step < 70; step++ {
			voidedNow := false
			switch roll := random.Intn(100); {
			case roll < 24:
				chosen := spans[random.Intn(len(spans))]
				if random.Intn(8) > 0 {
					chosen = spans[random.Intn(3)]
				}
				spec := stmt(fmt.Sprintf("r%02d", len(ids)+1), fmt.Sprintf("s%d", 1+random.Intn(6)), 10)
				spec.PeriodStart, spec.PeriodEnd, spec.Revision = chosen.from, chosen.to, 1+random.Intn(3)
				if random.Intn(7) == 0 {
					spec.Source = "hawk-2"
				}
				if random.Intn(4) == 0 {
					spec.Supersedes, spec.CorrectionReason = fmt.Sprintf("s%d", 1+random.Intn(6)), "recount"
					if spec.Supersedes == spec.Statement {
						spec.Supersedes, spec.CorrectionReason = "", ""
					}
				}
				switch random.Intn(24) {
				case 0:
					spec.Mock, spec.Supersedes, spec.CorrectionReason = true, "", ""
				case 1:
					spec.Status = Provisional
				case 2:
					spec.Tier = Estimated
				case 3:
					spec.Coverage = 9000
				case 4:
					spec.Declaration = Derived
				}
				if len(ids) > 0 && random.Intn(8) == 0 {
					spec = m.receptions[ids[random.Intn(len(ids))]].spec
					if random.Intn(3) == 0 {
						spec.Coverage = 9999
					}
				}
				wasLive := ""
				if known := m.statements[spec.Source+"|"+spec.Statement]; known != nil && known.live == "" && m.receptions[spec.ID] == nil {
					wasLive = "again"
				}
				state, occupant, err := h.r.Open(spec)
				got := errName(err)
				if err == nil {
					got += ":" + string(state)
				} else if occupant != "" {
					got += ":" + occupant
				}
				want := m.open(spec)
				check("open", got, want)
				if want == "ok:open" && !contains(ids, spec.ID) {
					ids = append(ids, spec.ID)
					if spec.Supersedes != "" {
						reached["supersession"]++
					}
					if wasLive != "" {
						reached["received again after a void"]++
					}
				}
			case roll < 60:
				id := pick()
				line := team([]string{"a", "b", "c", "a", "b", "c", "d"}[random.Intn(7)], "100")
				if r := m.receptions[id]; r != nil && random.Intn(3) > 0 {
					line.Team = []string{"a", "b", "c"}[len(r.lines)%3]
				}
				switch random.Intn(25) {
				case 0:
					line.Cost = "99.5"
				case 1:
					line.Measured = false
				case 2:
					line.Currency = "USD"
				}
				state, err := h.r.Post("ws1", id, line)
				got := errName(err)
				if err == nil {
					got += ":" + string(state)
				}
				check("post", got, m.post(id, line))
			case roll < 78:
				id := pick()
				state, err := h.r.Close("ws1", id)
				got := errName(err)
				if err == nil {
					got += ":" + string(state)
				}
				check("close", got, m.close(id))
			case roll < 86:
				id, reason := pick(), []VoidReason{WrongReport, WrongReport, ReissuedAtSource}[random.Intn(3)]
				if r := m.receptions[id]; r != nil && r.state != Voided && r.spec.Supersedes != "" {
					voidedAfterSuperseding[id] = true
				}
				check("void", errName(h.r.Void("ws1", id, reason)), m.void(id, reason))
				voidedNow = true
			default:
				id := pick()
				_, err := h.r.Eligibility("ws1", id)
				want := "ok"
				if m.receptions[id] == nil {
					want = "notfound"
				}
				check("eligibility", errName(err), want)
			}
			eligible := map[string][]string{}
			live := map[string]int{}
			for _, id := range ids {
				modelled := m.receptions[id]
				view := h.view(id)
				known := m.statements[modelled.spec.Source+"|"+modelled.spec.Statement]
				if view.State != modelled.state || view.SupersededBy != known.supersededBy || view.Lines != len(modelled.lines) {
					t.Fatalf("history %d step %d: %s is %s by %q with %d lines, model %s by %q with %d", history, step, id, view.State, view.SupersededBy, view.Lines, modelled.state, known.supersededBy, len(modelled.lines))
				}
				key := modelled.spec.Source + "|" + modelled.spec.Statement
				if view.SupersededBy != "" {
					superseded[key] = true
				} else if superseded[key] {
					t.Fatalf("history %d step %d: %s is no longer superseded", history, step, key)
				}
				if view.State != Voided {
					live[key]++
				}
				got, want := h.verdict(id), h.recomputed(id)
				if modelled := m.verdict(id, 2); got != want || got != modelled {
					t.Fatalf("history %d step %d reception %s:\n got %s\nfrom the registry's reads %s\nfrom the model %s", history, step, id, got, want, modelled)
				}
				for _, reason := range strings.Fields(strings.SplitN(strings.SplitN(got, "[", 2)[1], "]", 2)[0]) {
					reached["reason "+reason]++
				}
				if strings.Contains(got, PeriodConflict) {
					conflicted[id] = true
				} else if conflicted[id] && view.State != Voided {
					reached["conflict removed"]++
					if voidedNow {
						reached["conflict removed by a void"]++
					}
					delete(conflicted, id)
				}
				if got == "eligible[]" {
					reached["eligible"]++
					eligible[modelled.spec.Source] = append(eligible[modelled.spec.Source], id)
				}
			}
			names, namedBy, by := map[string]map[string]bool{}, map[string]map[string]bool{}, map[string]string{}
			for _, id := range ids {
				view := h.view(id)
				key := view.Spec.Source + "|" + view.Spec.Statement
				if earlier, seen := by[key]; seen && earlier != view.SupersededBy {
					t.Fatalf("history %d step %d: two receptions of %s disagree about what superseded it", history, step, key)
				}
				by[key] = view.SupersededBy
				if view.Spec.Supersedes != "" {
					target := view.Spec.Source + "|" + view.Spec.Supersedes
					if names[key] == nil {
						names[key] = map[string]bool{}
					}
					if namedBy[target] == nil {
						namedBy[target] = map[string]bool{}
					}
					names[key][target], namedBy[target][key] = true, true
				}
			}
			for _, id := range ids {
				spec := m.receptions[id].spec
				key := spec.Source + "|" + spec.Statement
				if len(names[key]) > 1 || len(namedBy[key]) > 1 {
					t.Fatalf("history %d step %d: %s supersedes %d statements and is superseded by %d", history, step, key, len(names[key]), len(namedBy[key]))
				}
				if (len(namedBy[key]) == 1) != (by[key] != "") || len(namedBy[key]) == 1 && !namedBy[key][spec.Source+"|"+by[key]] {
					t.Fatalf("history %d step %d: %s is superseded by %q, named by %v", history, step, key, by[key], namedBy[key])
				}
				at := key
				for hops := 0; hops < 50 && len(names[at]) == 1; hops++ {
					next := ""
					if hops == 49 {
						t.Fatalf("history %d step %d: a circle through %s", history, step, key)
					}
					if names[at][key] {
						t.Fatalf("history %d step %d: a circle through %s", history, step, key)
					}
					for _, other := range ids {
						if candidate := m.receptions[other].spec.Source + "|" + m.receptions[other].spec.Statement; names[at][candidate] {
							next = candidate
						}
					}
					at = next
				}
			}
			for key, count := range live {
				if count > 1 {
					t.Fatalf("history %d step %d: %d receptions that are not voided for %s", history, step, count, key)
				}
			}
			for name, list := range eligible {
				for i, first := range list {
					for _, other := range list[i+1:] {
						a, b := m.receptions[first].spec, m.receptions[other].spec
						if a.PeriodStart.Before(b.PeriodEnd) && b.PeriodStart.Before(a.PeriodEnd) {
							t.Fatalf("history %d step %d: %s and %s of %s are both eligible and overlap", history, step, first, other, name)
						}
					}
				}
			}
			for _, name := range []string{"hawk-1", "hawk-2"} {
				for _, chosen := range spans {
					out, err := h.r.Counting("ws1", name, chosen.from, chosen.to)
					want := ""
					for _, id := range eligible[name] {
						if spec := m.receptions[id].spec; spec.PeriodStart.Equal(chosen.from) && spec.PeriodEnd.Equal(chosen.to) {
							want = id
						}
					}
					if out.Reception != want || out.Sequence != m.sequence[name] || (err == nil) != (want != "") {
						t.Fatalf("history %d step %d: counting of %s is %+v %v, want %q at %d", history, step, name, out, err, want, m.sequence[name])
					}
					if want != "" {
						reached["counting hit"]++
					}
				}
			}
		}
		reached["conflicts"] += len(conflicted)
		reached["supersession then void"] += len(voidedAfterSuperseding)
	}
	for name, least := range map[string]int{"open ok": 300, "open conflict": 30, "open rejected": 30, "post ok": 300, "post conflict": 10, "post rejected": 30, "post notfound": 3, "close ok": 100, "close rejected": 10,
		"void ok": 100, "void conflict": 3, "eligible": 500, "counting hit": 500, "supersession": 100, "supersession then void": 20, "received again after a void": 20, "conflicts": 50, "conflict removed": 50, "conflict removed by a void": 20,
		"reason voided": 20, "reason not_closed": 20, "reason mock": 20, "reason provisional": 20, "reason not_measured": 20, "reason coverage_below_required": 20, "reason declaration_derived": 20, "reason superseded": 20,
		"reason period_conflict": 20, "reason unmeasured_lines": 20, "reason incomplete_list": 20, "reason totals_differ": 20} {
		if reached[name] < least {
			t.Errorf("the histories reached %q %d times, fewer than %d", name, reached[name], least)
		}
	}
	if t.Failed() {
		t.Log(reached)
	}
}

func contains(list []string, id string) bool {
	for _, known := range list {
		if known == id {
			return true
		}
	}
	return false
}

func TestConcurrentUse(t *testing.T) {
	h := newHarness(t, nil)
	spec := stmt("r1", "s1", 10)
	spec.Lines, spec.Totals = 20, map[string]string{"KRW": "250"}
	h.open(spec)
	var wait sync.WaitGroup
	var failures atomic.Int64
	var accepted sync.Map
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for step := 0; step < 20; step++ {
				name := fmt.Sprintf("team-%02d", (step*7+worker*3)%20)
				switch _, err := h.r.Post("ws1", "r1", team(name, "12.5")); err {
				case nil:
					accepted.Store(name, true)
				case ErrRejected:
				default:
					failures.Add(1)
				}
				h.r.Eligibility("ws1", "r1")
				from, to := month(10)
				h.r.Counting("ws1", "hawk-1", from, to)
			}
		}(worker)
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		if _, err := h.r.Close("ws1", "r1"); err != nil {
			failures.Add(1)
		}
	}()
	wait.Wait()
	count := 0
	accepted.Range(func(_, _ any) bool { count++; return true })
	view := h.view("r1")
	if failures.Load() != 0 || view.State != Closed || view.Lines != count || units(view.Sums["KRW"]) != int64(count)*125000 {
		t.Fatalf("%d failures, %d lines for %d accepted teams, sums %v", failures.Load(), view.Lines, count, view.Sums)
	}
	if _, err := h.r.Post("ws1", "r1", team("team-new", "1")); err != ErrRejected {
		t.Fatal(err)
	}
	for round := 0; round < 200; round++ {
		race := newHarness(t, nil)
		race.open(stmt("base", "s-base", 10))
		race.post("base", team("team-a", "100"), team("team-b", "150.5"), team("team-c", "49.50"))
		var opened, later atomic.Int64
		var closeErr error
		var group sync.WaitGroup
		for worker := 0; worker < 2; worker++ {
			group.Add(2)
			go func(worker int) {
				defer group.Done()
				if _, _, err := race.r.Open(stmt(fmt.Sprintf("twice-%d", worker), "s-one", 11)); err == nil {
					opened.Add(1)
				}
			}(worker)
			go func(worker int) {
				defer group.Done()
				if _, _, err := race.r.Open(superseding(fmt.Sprintf("later-%d", worker), fmt.Sprintf("s-later-%d", worker), 12, "s-base")); err == nil {
					later.Add(1)
				}
			}(worker)
		}
		group.Add(2)
		go func() {
			defer group.Done()
			race.r.Eligibility("ws1", "base")
			from, to := month(10)
			race.r.Counting("ws1", "hawk-1", from, to)
		}()
		go func() {
			defer group.Done()
			_, closeErr = race.r.Close("ws1", "base")
			race.r.Void("ws1", "base", WrongReport)
		}()
		group.Wait()
		base := race.view("base")
		if opened.Load() != 1 || later.Load() != 1 || base.SupersededBy == "" || race.verdict("base") != "not_eligible[voided]" || (closeErr == nil) != base.WasClosed || (closeErr != nil && closeErr != ErrRejected) {
			t.Fatalf("round %d: %d receptions of one statement, %d statements superseding one, close %v, was closed %v", round, opened.Load(), later.Load(), closeErr, base.WasClosed)
		}
	}
}
