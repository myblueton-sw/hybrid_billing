package reconcile

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
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
	negative := strings.HasPrefix(text, "-")
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(text, "-"), ".")
	for len(fraction) < 4 {
		fraction += "0"
	}
	var value int64
	for _, c := range whole + fraction {
		value = value*10 + int64(c-'0')
	}
	if negative {
		return -value
	}
	return value
}

func spell(random *rand.Rand, value int64) string {
	sign := ""
	if value < 0 {
		sign, value = "-", -value
	}
	text := fmt.Sprintf("%d.%04d", value/10000, value%10000)
	for strings.HasSuffix(text, "0") && random.Intn(2) == 0 {
		text = strings.TrimSuffix(text, "0")
	}
	return sign + strings.TrimSuffix(text, ".")
}

type mLine struct {
	category Category
	amount   int64
	item     string
	print    string
}

type mRun struct {
	spec      RunSpec
	subject   string
	document  string
	state     State
	wasClosed bool
	parts     map[string]string
	amounts   map[string]int64
	sources   map[string]bool
	lines     map[string]mLine
	verdict   string
	reason    VoidReason
}

type model struct {
	runs    map[string]*mRun
	holder  map[string]string
	highest map[string]int
	maxima  map[string]int64
}

func (m *model) superseded(r *mRun) bool {
	return r.spec.Basis == ProviderDocument && r.spec.Revision < m.highest[r.document]
}

func (m *model) counts(r *mRun) bool { return r.state != Voided && !m.superseded(r) }

func (m *model) linked(item string) bool {
	for _, r := range m.runs {
		if !m.counts(r) {
			continue
		}
		for _, line := range r.lines {
			if line.item == item {
				return true
			}
		}
	}
	return false
}

func (m *model) open(spec RunSpec) string {
	subject, document := "", ""
	switch spec.Basis {
	case ProviderDocument:
		document = spec.Issuer + "/" + spec.Account + "/" + spec.Document
		subject = fmt.Sprintf("doc/%s/%d/%s", document, spec.Revision, spec.Currency)
	case IndependentPrice:
		subject = "price/" + spec.Customer + "/" + spec.Period + "/" + spec.Charge
	default:
		subject = "pool/" + spec.Customer + "/" + spec.Period + "/" + spec.Pool
	}
	if earlier := m.runs[spec.ID]; earlier != nil {
		a, b := earlier.spec, spec
		same := a.Basis == b.Basis && earlier.subject == subject && a.Input == b.Input && a.Parts == b.Parts && a.QuantityComplete == b.QuantityComplete && a.Cost == b.Cost
		if same && a.Basis == ProviderDocument {
			same = units(a.Total) == units(b.Total) && units(a.Rounding) == units(b.Rounding)
		}
		if !same {
			return "conflict"
		}
		return "ok:" + string(earlier.state)
	}
	if m.holder[subject] != "" {
		return "conflict"
	}
	made := &mRun{spec: spec, subject: subject, document: document, state: Closed, wasClosed: true, parts: map[string]string{}, amounts: map[string]int64{}, sources: map[string]bool{}, lines: map[string]mLine{}}
	if spec.Basis == ProviderDocument {
		if spec.Revision < m.highest[document] || units(spec.Rounding) > m.maxima[spec.Currency] {
			return "rejected"
		}
		made.state, made.wasClosed = Open, false
		if spec.Revision > m.highest[document] {
			m.highest[document] = spec.Revision
		}
	}
	m.runs[spec.ID], m.holder[subject] = made, spec.ID
	return "ok:" + string(made.state)
}

func (m *model) usable(r *mRun, currency string) bool {
	return r.spec.Basis == ProviderDocument && r.state == Open && !m.superseded(r) && currency == r.spec.Currency
}

func (m *model) post(id string, p Part) string {
	r := m.runs[id]
	if r == nil {
		return "notfound"
	}
	print := fmt.Sprintf("%d/%s/%s/%s", units(p.Amount), p.Currency, p.Kind, p.Source)
	if earlier, known := r.parts[p.ID]; known {
		if earlier != print {
			return "conflict"
		}
		return "ok:" + string(r.state)
	}
	if r.sources[p.Source] {
		return "conflict"
	}
	if !m.usable(r, p.Currency) || p.Kind != Billed || len(r.parts) >= r.spec.Parts {
		return "rejected"
	}
	r.parts[p.ID], r.amounts[p.ID], r.sources[p.Source] = print, units(p.Amount), true
	return "ok:" + string(r.state)
}

func (m *model) explain(id string, l Line) string {
	r := m.runs[id]
	if r == nil {
		return "notfound"
	}
	item := ""
	if l.Category != Rounding {
		item = string(l.ItemType) + "/" + l.Item
		if l.ItemType == DocumentLine {
			item += fmt.Sprintf("/%s/%d", r.document, r.spec.Revision)
		}
	}
	print := fmt.Sprintf("%s/%d/%s/%s/%s", l.Category, units(l.Amount), l.Currency, item, l.Evidence)
	if earlier, known := r.lines[l.ID]; known {
		if earlier.print != print {
			return "conflict"
		}
		return "ok:" + string(r.state)
	}
	if !m.usable(r, l.Currency) {
		return "rejected"
	}
	if l.Category == Rounding {
		size := units(l.Amount)
		if size < 0 {
			size = -size
		}
		for _, line := range r.lines {
			if line.category == Rounding {
				return "rejected"
			}
		}
		if size > units(r.spec.Rounding) {
			return "rejected"
		}
	} else if m.linked(item) {
		return "rejected"
	}
	r.lines[l.ID] = mLine{category: l.Category, amount: units(l.Amount), item: item, print: print}
	return "ok:" + string(r.state)
}

func (m *model) residual(r *mRun) int64 {
	residual := units(r.spec.Total)
	for _, amount := range r.amounts {
		residual -= amount
	}
	for _, line := range r.lines {
		residual -= line.amount
	}
	return residual
}

func (m *model) close(id string) string {
	r := m.runs[id]
	if r == nil {
		return "notfound"
	}
	if r.wasClosed {
		return "ok:" + string(r.state) + ":" + r.verdict
	}
	if r.state != Open || m.superseded(r) || len(r.parts) != r.spec.Parts {
		return "rejected"
	}
	r.state, r.wasClosed, r.verdict = Closed, true, Unexplained
	if m.residual(r) == 0 {
		r.verdict = Explained
	}
	return "ok:closed:" + r.verdict
}

func (m *model) void(id string, reason VoidReason) string {
	r := m.runs[id]
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
	delete(m.holder, r.subject)
	return "ok"
}

func (m *model) readiness(r *mRun) string {
	switch {
	case r.state == Voided:
		return "not_ready[voided] undetermined"
	case r.spec.Basis == InternalCost:
		return "ready[] declared"
	case r.spec.Basis == IndependentPrice:
		cost := Undetermined
		if r.spec.Cost != "" {
			cost = Declared
		}
		if !r.spec.QuantityComplete {
			return "not_ready[quantity_incomplete] " + cost
		}
		return "ready[] " + cost
	}
	var reasons []string
	if r.state == Open {
		reasons = append(reasons, NotClosed)
	} else if r.verdict != Explained {
		reasons = append(reasons, UnexplainedResidue)
	}
	if m.superseded(r) {
		reasons = append(reasons, Superseded)
	}
	if len(reasons) == 0 {
		return "ready[] determined"
	}
	return fmt.Sprintf("not_ready%v undetermined", reasons)
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	random := rand.New(rand.NewSource(19200))
	reached := map[string]int{}
	balanced, nearMiss := 0, 0
	for history := 0; history < 400; history++ {
		h := newHarness(t, func(l *Limits) { l.Page = 50 })
		m := &model{runs: map[string]*mRun{}, holder: map[string]string{}, highest: map[string]int{}, maxima: map[string]int64{"USD": 500, "KRW": 10000}}
		var ids []string
		step := 0
		check := func(what, got, want string) {
			t.Helper()
			if got != want {
				t.Fatalf("history %d step %d %s:\n got %s\nwant %s", history, step, what, got, want)
			}
			reached[what+" "+strings.SplitN(want, ":", 2)[0]]++
		}
		pick := func() string {
			if len(ids) == 0 || random.Intn(25) == 0 {
				return "nowhere"
			}
			if random.Intn(10) < 6 {
				var usable []string
				for _, id := range ids {
					if r := m.runs[id]; m.usable(r, r.spec.Currency) {
						usable = append(usable, id)
					}
				}
				if len(usable) > 0 {
					return usable[random.Intn(len(usable))]
				}
			}
			if random.Intn(2) == 0 {
				return ids[len(ids)-1]
			}
			return ids[random.Intn(len(ids))]
		}
		for step = 0; step < 90; step++ {
			switch roll := random.Intn(100); {
			case roll < 16:
				var spec RunSpec
				id := fmt.Sprintf("run-%d", len(ids)+1)
				switch random.Intn(6) {
				case 0:
					spec = priced(id)
					spec.Charge, spec.QuantityComplete = fmt.Sprintf("charge-%d", random.Intn(3)), random.Intn(3) > 0
					if random.Intn(2) == 0 {
						spec.Cost, spec.Currency, spec.CostEvidence = "80", "USD", "ledger:4"
					}
				case 1:
					spec = pooled(id)
					spec.Pool = fmt.Sprintf("pool-%d", random.Intn(3))
				default:
					currency := []string{"USD", "USD", "KRW"}[random.Intn(3)]
					spec = doc(id, 1+random.Intn(3), currency, spell(random, int64(random.Intn(400000))-100000), 1+random.Intn(3))
					spec.Document = fmt.Sprintf("inv-%d", 1+random.Intn(3))
					spec.Rounding = []string{"0", "0.05", "0.0500", "0.01", "0.06"}[random.Intn(5)]
				}
				if len(ids) > 0 && random.Intn(6) == 0 {
					earlier := m.runs[ids[random.Intn(len(ids))]]
					spec = earlier.spec
					if random.Intn(3) == 0 && spec.Basis == ProviderDocument {
						spec.Parts++
					}
				}
				state, err := h.r.Open(spec)
				got := errName(err)
				if err == nil {
					got += ":" + string(state)
				}
				want := m.open(spec)
				check("open", got, want)
				if strings.HasPrefix(want, "ok") && m.runs[spec.ID] != nil && (len(ids) == 0 || ids[len(ids)-1] != spec.ID) && !contains(ids, spec.ID) {
					ids = append(ids, spec.ID)
				}
			case roll < 44:
				id := pick()
				name := fmt.Sprintf("p%d", 1+random.Intn(4))
				if r := m.runs[id]; r != nil && random.Intn(4) > 0 {
					name = fmt.Sprintf("p%d", len(r.parts)+1)
				}
				part := Part{ID: name, Amount: spell(random, int64(random.Intn(200000))-40000), Currency: "USD", Kind: Billed, Source: "src-" + name}
				if r := m.runs[id]; r != nil {
					part.Currency = r.spec.Currency
					if r.spec.Currency == "" {
						part.Currency = "USD"
					}
				}
				switch random.Intn(20) {
				case 0:
					part.Kind = []CostKind{Effective, List, Contracted, Sale}[random.Intn(4)]
				case 1:
					part.Currency = "EUR"
				case 2:
					part.Source = "src-p1"
				}
				state, err := h.r.Post("ws1", id, part)
				got := errName(err)
				if err == nil {
					got += ":" + string(state)
				}
				check("post", got, m.post(id, part))
			case roll < 62:
				id := pick()
				name := fmt.Sprintf("l%d", 1+random.Intn(5))
				category := categories[random.Intn(len(categories))]
				amount := int64(random.Intn(40000)) - 15000
				if amount == 0 {
					amount = 1
				}
				line := Line{ID: name, Category: category, Amount: spell(random, amount), Currency: "USD", Evidence: "ev"}
				if r := m.runs[id]; r != nil && r.spec.Currency != "" {
					line.Currency = r.spec.Currency
				}
				if category == Rounding {
					line.Amount = spell(random, []int64{100, -100, 500, -500, 501, 1}[random.Intn(6)])
				} else {
					line.ItemType = []ItemType{DocumentLine, InternalItem}[random.Intn(2)]
					line.Item = fmt.Sprintf("item-%d", random.Intn(6))
				}
				state, err := h.r.Explain("ws1", id, line)
				got := errName(err)
				if err == nil {
					got += ":" + string(state)
				}
				check("explain", got, m.explain(id, line))
			case roll < 76:
				id := pick()
				if r := m.runs[id]; r != nil && m.usable(r, r.spec.Currency) && len(r.parts) == r.spec.Parts && random.Intn(4) > 0 {
					missing := m.residual(r)
					kind := random.Intn(4)
					if kind == 0 {
						missing += []int64{1, -1}[random.Intn(2)]
					}
					if missing != 0 {
						line := Line{ID: fmt.Sprintf("fix-%d", step), Category: Credit, Amount: spell(random, missing), Currency: r.spec.Currency, ItemType: InternalItem, Item: fmt.Sprintf("fix-%d-%d", history, step), Evidence: "ev"}
						if _, err := h.r.Explain("ws1", id, line); errName(err) != strings.SplitN(m.explain(id, line), ":", 2)[0] {
							t.Fatalf("history %d step %d: the balancing line", history, step)
						}
					}
					if m.residual(r) == 0 {
						balanced++
					} else if m.residual(r) == 1 || m.residual(r) == -1 {
						nearMiss++
					}
				}
				state, verdict, err := h.r.Close("ws1", id)
				got := errName(err)
				if err == nil {
					got += ":" + string(state) + ":" + verdict
				}
				check("close", got, m.close(id))
			case roll < 80:
				id, reason := pick(), []VoidReason{WrongFigures, WrongFigures, VoidedBy}[random.Intn(3)]
				check("void", errName(h.r.Void("ws1", id, reason)), m.void(id, reason))
			default:
				id := pick()
				out, err := h.r.Readiness("ws1", id)
				if r := m.runs[id]; r == nil {
					check("readiness", errName(err), "notfound")
				} else {
					check("readiness", fmt.Sprintf("%s%v %s", out.Verdict, out.Reasons, out.Cost), m.readiness(r))
				}
			}
			ws := h.r.workspaces["ws1"]
			if ws == nil {
				continue
			}
			holders := map[string]int{}
			linked := map[string]string{}
			for id, held := range ws.runs {
				modelled := m.runs[id]
				if modelled == nil || held.state != modelled.state {
					t.Fatalf("history %d step %d: state of %s", history, step, id)
				}
				if held.state != Voided {
					holders[held.subject]++
				}
				out, _ := h.r.Readiness("ws1", id)
				if (held.state == Voided || ws.superseded(held)) && (out.Verdict == Ready || out.Cost != Undetermined) {
					t.Fatalf("history %d step %d: %s is voided or superseded and %s %s", history, step, id, out.Verdict, out.Cost)
				}
				if held.state != Voided && !ws.superseded(held) {
					for _, made := range held.lines {
						if made.stated.Category == Rounding {
							continue
						}
						if other, taken := linked[held.names(made.stated)]; taken && other != id {
							t.Fatalf("history %d step %d: an item is linked by %s and %s", history, step, other, id)
						}
						linked[held.names(made.stated)] = id
					}
				}
				if held.spec.Basis != ProviderDocument || !held.wasClosed {
					continue
				}
				view, _ := h.r.Run("ws1", id)
				recomputed := units(held.spec.Total)
				for after := 0; ; {
					page, err := h.r.Parts("ws1", id, after, 50)
					if err != nil {
						t.Fatal(err)
					}
					if len(page) == 0 {
						break
					}
					after += len(page)
					for _, posted := range page {
						recomputed -= units(posted.Amount)
					}
				}
				for after := 0; ; {
					page, err := h.r.Lines("ws1", id, after, 50)
					if err != nil {
						t.Fatal(err)
					}
					if len(page) == 0 {
						break
					}
					after += len(page)
					for _, made := range page {
						recomputed -= units(made.Amount)
					}
				}
				if recomputed != units(view.Residual) || (recomputed == 0) != (view.Verdict == Explained) {
					t.Fatalf("history %d step %d: residual of %s is %s, recomputed %d, verdict %s", history, step, id, view.Residual, recomputed, view.Verdict)
				}
			}
			for subject, count := range holders {
				if count > 1 {
					t.Fatalf("history %d step %d: %d runs hold %s", history, step, count, subject)
				}
			}
		}
	}
	for _, name := range []string{"open ok", "open conflict", "open rejected", "post ok", "post conflict", "post rejected", "post notfound", "explain ok", "explain conflict", "explain rejected", "explain notfound",
		"close ok", "close rejected", "close notfound", "void ok", "void conflict", "void notfound", "readiness ready[] determined", "readiness ready[] declared", "readiness ready[] undetermined",
		"readiness not_ready[voided] undetermined", "readiness not_ready[superseded] undetermined", "readiness not_ready[unexplained_residual] undetermined", "readiness not_ready[not_closed] undetermined"} {
		if reached[name] < 3 {
			t.Errorf("the histories reached %q %d times", name, reached[name])
		}
	}
	if balanced < 200 || nearMiss < 60 {
		t.Errorf("%d balanced runs and %d near misses were constructed", balanced, nearMiss)
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

func TestSplittingOrderAndReopeningKeepTheVerdict(t *testing.T) {
	random := rand.New(rand.NewSource(19201))
	for round := 0; round < 300; round++ {
		h := newHarness(t, nil)
		count := 2 + random.Intn(6)
		amounts := make([]int64, count)
		var total int64
		for index := range amounts {
			amounts[index] = int64(random.Intn(2000000)) - 500000
			total += amounts[index]
		}
		lines := make([]int64, 1+random.Intn(4))
		for index := range lines {
			lines[index] = int64(random.Intn(60000)) - 20000
			if lines[index] == 0 {
				lines[index] = 7
			}
			total += lines[index]
		}
		total += []int64{0, 0, 0, 1, -1}[random.Intn(5)]
		build := func(id, document string, split, shuffle bool) string {
			posted := append([]int64(nil), amounts...)
			if split {
				first := posted[0] / 3
				posted = append([]int64{first, posted[0] - first}, posted[1:]...)
			}
			spec := doc(id, 1, "USD", spell(random, total), len(posted))
			spec.Document = document
			h.open(spec)
			order := make([]int, len(posted))
			for index := range order {
				order[index] = index
			}
			if shuffle {
				random.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
			}
			for _, index := range order {
				h.post(id, bill(fmt.Sprintf("p%d", index), spell(random, posted[index])))
			}
			for _, index := range random.Perm(len(lines)) {
				h.explain(id, explain(fmt.Sprintf("l%d", index), categories[index%5], spell(random, lines[index]), fmt.Sprintf("line-%d", index)))
			}
			h.close(id)
			view := h.view(id)
			return view.Internal + " " + view.Explained + " " + view.Residual + " " + view.Verdict
		}
		first := build("a", "inv-a", false, false)
		if second := build("b", "inv-b", true, true); second != first {
			t.Fatalf("round %d: %s / %s", round, first, second)
		}
		if err := h.r.Void("ws1", "a", WrongFigures); err != nil {
			t.Fatal(err)
		}
		if again := build("c", "inv-a", false, true); again != first {
			t.Fatalf("round %d: reopened %s / %s", round, first, again)
		}
	}
}

func TestFiftyThousandParts(t *testing.T) {
	random := rand.New(rand.NewSource(192))
	h := newHarness(t, func(l *Limits) { l.Parts = 60000; l.Page = 5000 })
	var total int64
	amounts := make([]int64, 50000)
	for index := range amounts {
		amounts[index] = int64(random.Intn(40000000)) - 10000000
		total += amounts[index]
	}
	h.open(doc("big", 1, "USD", spell(random, total+30000), len(amounts)))
	for _, index := range random.Perm(len(amounts)) {
		h.post("big", bill(fmt.Sprintf("p%05d", index), spell(random, amounts[index])))
	}
	h.explain("big", explain("tax", Tax, "3", "line-1"))
	if verdict := h.close("big"); verdict != Explained {
		t.Fatal(sums(h.view("big")))
	}
	view := h.view("big")
	if units(view.Internal) != total || view.Residual != "0" || view.Posted != 50000 {
		t.Fatalf("%s, generator %d", sums(view), total)
	}
	seen := 0
	for {
		page, err := h.r.Parts("ws1", "big", seen, 5000)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for offset, posted := range page {
			if posted.ID != fmt.Sprintf("p%05d", seen+offset) || units(posted.Amount) != amounts[seen+offset] {
				t.Fatalf("part %d: %+v", seen+offset, posted)
			}
		}
		seen += len(page)
	}
	if seen != 50000 {
		t.Fatal(seen)
	}
}

func TestConcurrentUse(t *testing.T) {
	h := newHarness(t, nil)
	h.open(doc("d1", 1, "USD", "250", 20))
	var wait sync.WaitGroup
	var failures atomic.Int64
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for step := 0; step < 20; step++ {
				index := (step*7 + worker*3) % 20
				if state, err := h.r.Post("ws1", "d1", bill(fmt.Sprintf("p%02d", index), "12.5")); err != nil || (state != Open && state != Closed) {
					failures.Add(1)
				}
				h.r.Close("ws1", "d1")
				h.r.Readiness("ws1", "d1")
				h.r.Explain("ws1", "d1", explain(fmt.Sprintf("l%d", worker), Tax, "1", fmt.Sprintf("line-%d", worker)))
			}
		}(worker)
	}
	wait.Wait()
	view := h.view("d1")
	if failures.Load() != 0 || view.State != Closed || view.Posted != 20 || view.Internal != "250" {
		t.Fatalf("%d %s", failures.Load(), sums(view))
	}
	if units(view.Residual) != -int64(view.Lines)*10000 {
		t.Fatalf("%s with %d lines", view.Residual, view.Lines)
	}
	for round := 0; round < 200; round++ {
		race := newHarness(t, nil)
		var opened, voided atomic.Int64
		var group sync.WaitGroup
		for worker := 0; worker < 2; worker++ {
			group.Add(1)
			go func(worker int) {
				defer group.Done()
				if _, err := race.r.Open(doc(fmt.Sprintf("r%d", worker), 1, "USD", "10", 1)); err == nil {
					opened.Add(1)
				}
			}(worker)
		}
		group.Wait()
		if opened.Load() != 1 {
			t.Fatalf("round %d: %d runs opened for one key", round, opened.Load())
		}
		holder, _ := race.r.Holder("ws1", Subject{Basis: ProviderDocument, Issuer: "aws", Account: "acct-1", Document: "inv-100", Revision: 1, Currency: "USD"})
		race.post(holder, bill("p1", "10"))
		group.Add(3)
		go func() {
			defer group.Done()
			race.r.Close("ws1", holder)
		}()
		go func() {
			defer group.Done()
			if err := race.r.Void("ws1", holder, WrongFigures); err == nil {
				voided.Add(1)
			}
		}()
		go func() {
			defer group.Done()
			race.r.Open(doc("later", 2, "USD", "10", 1))
			race.r.Readiness("ws1", holder)
		}()
		group.Wait()
		out, _ := race.r.Readiness("ws1", holder)
		if voided.Load() != 1 || out.Verdict != NotReady || out.Cost != Undetermined {
			t.Fatalf("round %d: %+v", round, out)
		}
	}
}
