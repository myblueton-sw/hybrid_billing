package normalize

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

var modelDecimal = regexp.MustCompile(`^-?[0-9]{1,12}(\.[0-9]{1,4})?$`)

var modelTimes = map[string]int64{
	"2026-10-05T00:00:00Z":      time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC).Unix(),
	"2026-10-06T00:00:00Z":      time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC).Unix(),
	"2026-10-05T09:00:00+09:00": time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC).Unix(),
	"2026-10-01T00:00:00Z":      time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix(),
	"2026-11-01T00:00:00Z":      time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC).Unix(),
	"2026-09-30T00:00:00Z":      time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC).Unix(),
	"2026-11-02T00:00:00Z":      time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC).Unix(),
}

const micro = 1000000

func modelMicro(text string) int64 {
	negative := strings.HasPrefix(text, "-")
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(text, "-"), ".")
	for len(fraction) < 6 {
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

func modelText(value int64) string {
	sign := ""
	if value < 0 {
		sign, value = "-", -value
	}
	text := strings.TrimRight(fmt.Sprintf("%d.%06d", value/micro, value%micro), "0")
	return sign + strings.TrimSuffix(text, ".")
}

type modelRow struct {
	class    int
	findings int
	lacking  bool
	costs    map[string]int64
	texts    map[string]string
	currency string
}

type modelRun struct {
	fields  []Field
	entries []Entry
	window  bool
	rows    map[string]*modelRow
	order   []string
	seen    map[RowKey][]string
}

func (m *modelRun) judge(row Row) *modelRow {
	out := &modelRow{costs: map[string]int64{}, texts: map[string]string{}}
	text := map[string]bool{}
	instants := map[string]*int64{}
	costly := false
	for index, entry := range m.entries {
		field := m.fields[index]
		if entry.Missing != "" {
			continue
		}
		cell, present := row.Cells[entry.Source]
		if present && !cell.Null {
			text[field.Name] = true
			costly = costly || field.Kind == Cost
			switch {
			case cell.Text == "" && field.Kind != Text:
				out.findings++
			case field.Kind == Cost || field.Kind == Quantity:
				if !modelDecimal.MatchString(cell.Text) {
					out.findings++
					break
				}
				value := modelMicro(cell.Text)
				switch {
				case entry.Transform == Negate:
					value = -value
				case entry.Transform == Scale && entry.Power > 0:
					for power := 0; power < entry.Power; power++ {
						value *= 10
					}
				case entry.Transform == Scale:
					for power := 0; power > entry.Power; power-- {
						if value%10 != 0 {
							out.findings++
							value = -1
							break
						}
						value /= 10
					}
				}
				if field.Kind == Cost {
					out.costs[field.Name] = value
				}
				out.texts[field.Name] = modelText(value)
			case field.Kind == Time:
				if at, valid := modelTimes[cell.Text]; valid {
					instants[field.Name] = &at
				} else {
					out.findings++
				}
			case field.Kind == Currency:
				if cell.Text != "USD" && cell.Text != "EUR" {
					out.findings++
				} else {
					out.currency = cell.Text
				}
			}
			continue
		}
		if field.Format {
			out.findings++
		}
		if field.Billing {
			out.lacking = true
		}
	}
	if costly && !text["cur"] {
		out.findings++
	}
	if m.entries[5].Missing == "" {
		if m.window {
			for _, name := range []string{"start", "end"} {
				if !text[name] {
					out.findings++
				}
			}
		}
		if from, to := instants["start"], instants["end"]; from != nil && to != nil {
			if *from >= *to {
				out.findings++
			}
			if m.window && (*from < modelTimes["2026-10-01T00:00:00Z"] || *to > modelTimes["2026-11-01T00:00:00Z"]) {
				out.findings++
			}
		}
	}
	if out.findings > 0 {
		out.class = inError
	}
	return out
}

func (m *modelRun) add(id string, row Row) {
	made := m.judge(row)
	others := m.seen[row.Key]
	if len(others) > 0 {
		made.class = quarantined
		for _, other := range others {
			m.rows[other].class = quarantined
		}
	}
	m.seen[row.Key] = append(others, id)
	m.rows[id] = made
	m.order = append(m.order, id)
}

func (m *modelRun) counts() string {
	var tally [3]int
	for _, row := range m.rows {
		tally[row.class]++
	}
	return fmt.Sprintf("%d=%d+%d+%d", len(m.rows), tally[normal], tally[inError], tally[quarantined])
}

func (m *modelRun) sums() map[string]map[string]int64 {
	out := map[string]map[string]int64{}
	for _, row := range m.rows {
		if row.class != normal {
			continue
		}
		for field, value := range row.costs {
			if out[field] == nil {
				out[field] = map[string]int64{}
			}
			out[field][row.currency] += value
		}
	}
	return out
}

func (m *modelRun) verdicts(expected map[string]map[string]string) (string, int) {
	var format, billing []string
	partial, gap, hasCost := false, false, false
	for index, entry := range m.entries {
		field := m.fields[index]
		if field.Kind == Cost && field.Billing {
			hasCost = true
		}
		if entry.Missing != "" && entry.Missing != NotApplicable {
			partial = true
		}
		if entry.Missing != "" && field.Billing {
			gap = true
		}
	}
	var tally [3]int
	lacking, findings := false, 0
	for _, row := range m.rows {
		tally[row.class]++
		if row.class == normal && row.lacking {
			lacking = true
		}
		if row.class == inError {
			findings += row.findings
		}
	}
	result := Conformant
	if partial {
		result = Partial
		format = append(format, MappingPartial)
	}
	if gap || !hasCost {
		billing = append(billing, MappingGap)
	}
	if tally[inError] > 0 {
		result = Nonconformant
		format = append(format, RowsInError)
		billing = append(billing, RowsInError)
	}
	if tally[quarantined] > 0 {
		result = Nonconformant
		format = append(format, RowsQuarantined)
		billing = append(billing, RowsQuarantined)
	}
	if lacking {
		billing = append(billing, RowsLacking)
	}
	sums := m.sums()
	unexpected, differ := false, false
	for index, entry := range m.entries {
		field := m.fields[index]
		if field.Kind != Cost || entry.Missing != "" {
			continue
		}
		stated, given := expected[field.Name]
		if !given {
			unexpected = unexpected || field.Billing
			continue
		}
		wanted := map[string]int64{}
		for currency, sum := range stated {
			wanted[currency] = modelMicro(sum)
		}
		actual := sums[field.Name]
		if actual == nil {
			actual = map[string]int64{}
		}
		if !reflect.DeepEqual(wanted, actual) {
			differ = true
		}
	}
	if unexpected {
		billing = append(billing, TotalsNotExpected)
	}
	if differ {
		billing = append(billing, TotalsDiffer)
	}
	if m.entries[5].Missing == "" && !m.window {
		billing = append(billing, WindowNotDeclared)
	}
	ready := Ready
	if len(billing) > 0 {
		ready = NotReady
	}
	return fmt.Sprintf("%s%v %s%v", result, format, ready, billing), findings
}

type generated struct {
	target   Target
	entries  []Entry
	rows     []Row
	window   bool
	expected map[string]map[string]string
}

func pick(random *rand.Rand, options ...string) string { return options[random.Intn(len(options))] }

func generate(random *rand.Rand) generated {
	out := generated{target: testTarget(), entries: fullEntries()}
	out.target.Fields[1].Billing = random.Intn(3) == 0
	out.target.Fields[0].Billing = random.Intn(12) != 0
	for _, index := range []int{1, 3} {
		switch random.Intn(5) {
		case 0:
			out.entries[index].Transform = Negate
		case 1:
			out.entries[index].Transform, out.entries[index].Power = Scale, 2
		case 2:
			out.entries[index].Transform, out.entries[index].Power = Scale, -3
		}
	}
	if random.Intn(3) == 0 {
		out.entries[2] = missing2("list", random.Intn(3) == 0)
	}
	if random.Intn(5) == 0 {
		out.entries[5], out.entries[6] = Entry{Field: "start", Missing: NotSupported}, Entry{Field: "end", Missing: NotSupported}
	}
	if random.Intn(8) == 0 {
		out.entries[0] = Entry{Field: "billed", Missing: NotSupported}
	}
	if random.Intn(6) == 0 && !out.target.Fields[1].Billing {
		out.entries[1] = Entry{Field: "effective", Missing: NotProvidedBySource}
	}
	out.window = out.entries[5].Missing == "" && random.Intn(10) < 7
	count := 1 + random.Intn(28)
	clean := random.Intn(3) == 0
	for index := 0; index < count; index++ {
		key := ev(fmt.Sprintf("k%d", index))
		if !clean && random.Intn(9) == 0 {
			key = ev(fmt.Sprintf("k%d", random.Intn(count)))
		}
		if random.Intn(7) == 0 {
			key = RowKey{Kind: LocatorKeys, Document: pick(random, "d1", "d2"), Key: fmt.Sprintf("k%d", random.Intn(count))}
		}
		row := Row{Key: key, Cells: map[string]Cell{}}
		put := func(name string, values ...string) {
			roll := random.Intn(100)
			switch {
			case clean || roll < 82:
				row.Cells[name] = Cell{Text: values[random.Intn(3)]}
			case roll < 88:
				row.Cells[name] = Cell{Null: true}
			case roll < 93:
			default:
				row.Cells[name] = Cell{Text: values[3+random.Intn(len(values)-3)]}
			}
		}
		put("BilledCost", "1.5", "200", "-0.25", "x", "", "0.12345", "1e2")
		put("EffectiveCost", "3000", "0.5", "0", "abc", "", "0.0001", "0.03")
		put("ListCost", "7", "0.0001", "12.34", "?", "")
		put("Usage", "1000", "0", "25.5", "many", "", "0.001")
		put("BillingCurrency", "USD", "USD", "EUR", "usd", "", "KRWX")
		put("PeriodStart", "2026-10-05T00:00:00Z", "2026-10-01T00:00:00Z", "2026-10-05T09:00:00+09:00", "soon", "", "2026-09-30T00:00:00Z", "2026-10-06T00:00:00Z")
		put("PeriodEnd", "2026-10-06T00:00:00Z", "2026-11-01T00:00:00Z", "2026-10-06T00:00:00Z", "later", "", "2026-11-02T00:00:00Z", "2026-10-05T00:00:00Z")
		put("ServiceName", "compute", "storage", "", "", "db")
		if random.Intn(6) == 0 {
			row.Cells["Extra"] = Cell{Text: "ignored"}
		}
		out.rows = append(out.rows, row)
	}
	return out
}

func missing2(field string, applicable bool) Entry {
	if applicable {
		return Entry{Field: field, Missing: NotApplicable}
	}
	return Entry{Field: field, Missing: NotSupported}
}

func chunked(random *rand.Rand, rows []Row) [][]Row {
	var chunks [][]Row
	for from := 0; from < len(rows); {
		size := 1 + random.Intn(6)
		if size > len(rows)-from {
			size = len(rows) - from
		}
		chunks = append(chunks, rows[from:from+size])
		from += size
	}
	return chunks
}

func runGenerated(t *testing.T, random *rand.Rand, made generated, compare bool) (RunView, []string) {
	t.Helper()
	h := newHarness(t, nil)
	if err := h.r.DeclareTarget(made.target); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.DeclareMapping(mapping, 0, made.entries); err != nil {
		t.Fatal(err)
	}
	chunks := chunked(random, made.rows)
	s := spec("r1", len(chunks), len(made.rows))
	if !made.window {
		s.Window = nil
	}
	s.Expected = made.expected
	h.open(s)
	m := &modelRun{fields: made.target.Fields, entries: made.entries, window: made.window, rows: map[string]*modelRow{}, seen: map[RowKey][]string{}}
	arrived := map[int]bool{}
	order := random.Perm(len(chunks))
	for step := 0; step < len(order); step++ {
		index := order[step]
		for _, repeat := range []int{index, order[random.Intn(step+1)]} {
			h.check("r1", repeat+1, chunks[repeat]...)
		}
		if !arrived[index] {
			arrived[index] = true
			for position, row := range chunks[index] {
				m.add(fmt.Sprintf("%d.%d", index+1, position+1), row)
			}
		}
		view := h.view("r1")
		if view.Normal+view.Errors+view.Quarantined != view.Checked {
			t.Fatalf("counts do not add up: %s", counts(view))
		}
		if compare && counts(view) != m.counts() {
			t.Fatalf("after chunk %d: %s, model %s", index+1, counts(view), m.counts())
		}
	}
	view := h.close("r1")
	var normalRows []string
	for _, row := range h.rows("r1") {
		id := fmt.Sprintf("%d.%d", row.Chunk, row.Position)
		line := fmt.Sprintf("%v", row.Key)
		for _, field := range []string{"billed", "effective", "list", "qty", "cur", "start", "end", "svc", "note"} {
			line += fmt.Sprintf(" %s=%s:%s", field, row.Cells[field].State, row.Cells[field].Text)
		}
		normalRows = append(normalRows, line+fmt.Sprintf(" %v", row.Lacking))
		if !compare {
			continue
		}
		modelled := m.rows[id]
		if modelled == nil || modelled.class != normal {
			t.Fatalf("row %s is normal, model says %+v", id, modelled)
		}
		for field, want := range modelled.texts {
			if got := row.Cells[field]; got.State != Value || got.Text != want {
				t.Fatalf("row %s field %s: %+v, model %s", id, field, got, want)
			}
		}
		if (len(row.Lacking) > 0) != modelled.lacking {
			t.Fatalf("row %s lacking %v, model %v", id, row.Lacking, modelled.lacking)
		}
	}
	sort.Strings(normalRows)
	if compare {
		want, findings := m.verdicts(made.expected)
		if got := verdicts(view); got != want || view.FindingsTotal != findings || counts(view) != m.counts() {
			t.Fatalf("verdicts:\n got %s (%d findings, %s)\nwant %s (%d findings, %s)\n%s", got, view.FindingsTotal, counts(view), want, findings, m.counts(), h.findings("r1"))
		}
		sums := map[string]map[string]string{}
		for field, byCurrency := range m.sums() {
			sums[field] = map[string]string{}
			for currency, sum := range byCurrency {
				sums[field][currency] = modelText(sum)
			}
		}
		if !reflect.DeepEqual(view.Sums, sums) {
			t.Fatalf("sums %v, model %v", view.Sums, sums)
		}
	}
	return view, normalRows
}

func TestRandomRunsAgainstAReferenceModel(t *testing.T) {
	random := rand.New(rand.NewSource(18800))
	reached := map[string]int{}
	for round := 0; round < 1500; round++ {
		made := generate(random)
		probe := &modelRun{fields: made.target.Fields, entries: made.entries, window: made.window, rows: map[string]*modelRow{}, seen: map[RowKey][]string{}}
		for index, row := range made.rows {
			probe.add(fmt.Sprint(index), row)
		}
		made.expected = map[string]map[string]string{}
		summed := probe.sums()
		for _, field := range []string{"billed", "effective", "list"} {
			byCurrency, present := summed[field]
			if !present || random.Intn(6) == 0 {
				continue
			}
			made.expected[field] = map[string]string{}
			for _, currency := range []string{"EUR", "USD"} {
				sum, counted := byCurrency[currency]
				if !counted {
					continue
				}
				if random.Intn(12) == 0 {
					sum += 100
				}
				if random.Intn(15) != 0 {
					made.expected[field][currency] = modelText(sum)
				}
			}
			if random.Intn(15) == 0 {
				made.expected[field]["KRW"] = "0"
			}
		}
		if made.entries[0].Missing == "" && made.expected["billed"] == nil && random.Intn(2) == 0 {
			made.expected["billed"] = map[string]string{}
		}
		for field, sums := range made.expected {
			for currency, sum := range sums {
				if !modelDecimal.MatchString(sum) {
					delete(made.expected[field], currency)
				}
			}
		}
		view, _ := runGenerated(t, random, made, true)
		reached[view.Format.Result+"+"+view.Billing.Result]++
		for _, reason := range append(append([]string{}, view.Format.Reasons...), view.Billing.Reasons...) {
			reached[reason]++
		}
		if view.Quarantined > 0 && view.Errors > 0 && view.Normal > 0 {
			reached["all three classes"]++
		}
	}
	for _, name := range []string{"conformant+ready", "conformant+not_ready", "partial+ready", "partial+not_ready", "nonconformant+not_ready", MappingPartial, RowsInError, RowsQuarantined,
		MappingGap, RowsLacking, TotalsNotExpected, TotalsDiffer, WindowNotDeclared, "all three classes"} {
		if reached[name] < 5 {
			t.Errorf("the runs reached %q %d times: %v", name, reached[name], reached)
		}
	}
	if reached["nonconformant+ready"] != 0 {
		t.Fatal("a nonconformant run was ready")
	}
}

func TestChunkingAndOrderOfArrivalDoNotChangeTheResult(t *testing.T) {
	random := rand.New(rand.NewSource(18801))
	for round := 0; round < 400; round++ {
		made := generate(random)
		made.expected = map[string]map[string]string{}
		if made.entries[0].Missing == "" {
			made.expected["billed"] = map[string]string{"USD": "1.5"}
		}
		first, firstRows := runGenerated(t, random, made, false)
		second, secondRows := runGenerated(t, random, made, false)
		if verdicts(first) != verdicts(second) || counts(first) != counts(second) || !reflect.DeepEqual(first.Sums, second.Sums) || first.FindingsTotal != second.FindingsTotal ||
			!reflect.DeepEqual(firstRows, secondRows) || !reflect.DeepEqual(first.Unused, second.Unused) {
			t.Fatalf("round %d: %s %s / %s %s", round, verdicts(first), counts(first), verdicts(second), counts(second))
		}
	}
}
