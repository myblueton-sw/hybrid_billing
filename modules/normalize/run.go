package normalize

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
	"sort"
	"strconv"
	"time"
)

const (
	normal = iota
	inError
	quarantined
)

type found struct {
	field int
	rank  int
	code  string
}

type record struct {
	chunk    int
	position int
	key      RowKey
	class    int
	cells    []OutCell
	numbers  []*number
	lacking  []int
	findings []found
}

type run struct {
	spec            RunSpec
	print           string
	revision        *revision
	expected        map[int]map[string]*number
	state           State
	chunks          map[int]string
	records         map[int][]*record
	keys            map[RowKey][]*record
	unused          map[string]bool
	checked         int
	counts          [3]int
	held            int
	sums            map[string]map[string]string
	format          *Verdict
	billing         *Verdict
	rows            []*record
	findings        []Finding
	findingsTotal   int
	apart           []QuarantinedRow
	names           Listed
	profileID       string
	profileRevision string
	abandon         AbandonReason
	discard         DiscardReason
	contradiction   int
	opened          time.Time
	changed         time.Time
}

func (r *Registry) find(workspaceID, id string) (*workspace, *run) {
	ws := r.workspaces[workspaceID]
	if ws == nil {
		return nil, nil
	}
	return ws, ws.runs[id]
}

func (r *Registry) Open(spec RunSpec) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(spec.WorkspaceID) || !validID(spec.ID) || !spec.Mapping.valid() || spec.Revision < 1 {
		return "", ErrInvalid
	}
	if !validID(spec.Input.Scope) || !validID(spec.Input.Period) || spec.Input.Manifest < 1 || spec.Chunks < 1 || spec.Rows < spec.Chunks {
		return "", ErrInvalid
	}
	prints := []string{spec.WorkspaceID, spec.ID, spec.Mapping.Provider, spec.Mapping.SourceVersion, spec.Mapping.Target, strconv.FormatUint(spec.Revision, 10),
		spec.Input.Scope, spec.Input.Period, strconv.FormatUint(spec.Input.Manifest, 10), strconv.Itoa(spec.Chunks), strconv.Itoa(spec.Rows)}
	if spec.Window != nil {
		if !spec.Window.From.Before(spec.Window.To) {
			return "", ErrInvalid
		}
		prints = append(prints, "window", spec.Window.From.UTC().Format(time.RFC3339Nano), spec.Window.To.UTC().Format(time.RFC3339Nano))
	}
	fields := make([]string, 0, len(spec.Expected))
	for field := range spec.Expected {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	parsed := map[string]map[string]*number{}
	for _, field := range fields {
		if !validID(field) {
			return "", ErrInvalid
		}
		prints = append(prints, "expected", field)
		currencies := make([]string, 0, len(spec.Expected[field]))
		for currency := range spec.Expected[field] {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		parsed[field] = map[string]*number{}
		for _, currency := range currencies {
			sum, code := parseNumber(r.profile, spec.Expected[field][currency])
			if !validCurrency(currency) || code != "" {
				return "", ErrInvalid
			}
			parsed[field][currency] = sum
			prints = append(prints, currency, sum.text())
		}
	}
	print := join(prints...)
	ws := r.workspaces[spec.WorkspaceID]
	if ws != nil {
		if held := ws.runs[spec.ID]; held != nil {
			if held.print != print {
				return "", ErrConflict
			}
			return held.state, nil
		}
	}
	revisions := r.mappings[spec.Mapping]
	if spec.Revision > uint64(len(revisions)) {
		return "", ErrNotFound
	}
	bound := revisions[spec.Revision-1]
	if spec.Revision != uint64(len(revisions)) || spec.Window != nil && !bound.period {
		return "", ErrRejected
	}
	expected := map[int]map[string]*number{}
	for field, sums := range parsed {
		position, known := bound.target.index[field]
		if !known || bound.target.stated.Fields[position].Kind != Cost || bound.entries[position].Missing != "" {
			return "", ErrRejected
		}
		expected[position] = sums
	}
	if spec.Chunks > r.limits.Chunks {
		return "", ErrLimit
	}
	if spec.Rows/spec.Chunks > r.limits.ChunkRows || spec.Rows/spec.Chunks == r.limits.ChunkRows && spec.Rows%spec.Chunks != 0 {
		return "", ErrInvalid
	}
	if spec.Rows > r.limits.Rows || ws != nil && len(ws.runs) >= r.limits.Runs {
		return "", ErrLimit
	}
	if ws == nil {
		ws = &workspace{runs: map[string]*run{}}
		r.workspaces[spec.WorkspaceID] = ws
	}
	stored := spec
	if spec.Window != nil {
		stored.Window = &Window{From: spec.Window.From.UTC(), To: spec.Window.To.UTC()}
	}
	stored.Expected = map[string]map[string]string{}
	for field, sums := range parsed {
		stored.Expected[field] = map[string]string{}
		for currency, sum := range sums {
			stored.Expected[field][currency] = sum.text()
		}
	}
	ws.runs[spec.ID] = &run{spec: stored, print: print, revision: bound, expected: expected, state: Open, chunks: map[int]string{}, records: map[int][]*record{},
		keys: map[RowKey][]*record{}, unused: map[string]bool{}, profileID: r.profile.ID(), profileRevision: r.profile.Revision(), opened: r.last, changed: r.last}
	return Open, nil
}

func put(h hash.Hash, field string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(field)))
	h.Write(size[:])
	h.Write([]byte(field))
}

func (r *Registry) digest(rows []Row) (string, bool) {
	h := sha256.New()
	for _, row := range rows {
		if !row.Key.valid() || len(row.Cells) > r.limits.Cells {
			return "", false
		}
		names := make([]string, 0, len(row.Cells))
		for name, cell := range row.Cells {
			if !validID(name) || len(cell.Text) > r.limits.Text || cell.Null && cell.Text != "" {
				return "", false
			}
			names = append(names, name)
		}
		sort.Strings(names)
		put(h, string(row.Key.Kind))
		put(h, row.Key.Document)
		put(h, row.Key.Key)
		put(h, strconv.Itoa(len(names)))
		for _, name := range names {
			put(h, name)
			if row.Cells[name].Null {
				put(h, "null")
			} else {
				put(h, "text")
				put(h, row.Cells[name].Text)
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

func (r *Registry) drop(ws *workspace, held *run) {
	ws.rows -= held.held
	held.held = 0
	held.records, held.keys, held.rows, held.findings, held.apart, held.unused = nil, nil, nil, nil, nil, nil
}

func (r *Registry) Check(workspaceID, id string, chunk int, rows []Row) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || chunk < 1 || len(rows) == 0 || len(rows) > r.limits.ChunkRows {
		return "", ErrInvalid
	}
	sent, formed := r.digest(rows)
	if !formed {
		return "", ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return "", ErrNotFound
	}
	if chunk > held.spec.Chunks {
		return "", ErrInvalid
	}
	earlier, known := held.chunks[chunk]
	if known && earlier == sent {
		return held.state, nil
	}
	if held.state != Open {
		return "", ErrRejected
	}
	if known {
		held.state, held.contradiction, held.changed = Inconsistent, chunk, r.last
		r.drop(ws, held)
		return held.state, nil
	}
	if len(rows) > r.limits.Rows-ws.rows {
		return "", ErrLimit
	}
	made := make([]*record, len(rows))
	for position, row := range rows {
		made[position] = r.judge(held, chunk, position+1, row)
		others := held.keys[row.Key]
		if len(others) > 0 {
			made[position].class = quarantined
			for _, other := range others {
				held.counts[other.class]--
				other.class = quarantined
				held.counts[quarantined]++
			}
		}
		held.counts[made[position].class]++
		held.keys[row.Key] = append(others, made[position])
	}
	held.chunks[chunk] = sent
	held.records[chunk] = made
	held.checked += len(rows)
	held.held += len(rows)
	ws.rows += len(rows)
	return held.state, nil
}

func (r *Registry) judge(held *run, chunk, position int, row Row) *record {
	bound := held.revision
	aim := bound.target
	made := &record{chunk: chunk, position: position, key: row.Key, cells: make([]OutCell, len(bound.entries)), numbers: make([]*number, len(bound.entries))}
	for name := range row.Cells {
		if !bound.sources[name] {
			held.unused[name] = true
		}
	}
	texts := make([]bool, len(bound.entries))
	times := make([]*time.Time, len(bound.entries))
	note := func(field, rank int, code string) {
		made.findings = append(made.findings, found{field, rank, code})
	}
	costly := false
	for index, entry := range bound.entries {
		field := aim.stated.Fields[index]
		if entry.Missing != "" {
			made.cells[index] = OutCell{State: Unmapped}
			continue
		}
		cell, provided := row.Cells[entry.Source]
		switch {
		case !provided:
			made.cells[index] = OutCell{State: NotProvided}
		case cell.Null:
			made.cells[index] = OutCell{State: Null}
		default:
			texts[index] = true
			costly = costly || field.Kind == Cost
			code, text := "", cell.Text
			switch {
			case cell.Text == "" && field.Kind != Text:
				code = EmptyText
			case field.Kind.numeric():
				var parsed *number
				if parsed, code = parseNumber(r.profile, cell.Text); code == "" {
					parsed.apply(entry)
					if parsed.fits(r.profile) {
						made.numbers[index], text = parsed, parsed.text()
					} else {
						code = OutsideProfile
					}
				}
			case field.Kind == Time:
				if at, valid := parseTime(cell.Text); valid {
					times[index], text = &at, at.Format(time.RFC3339Nano)
				} else {
					code = NotTime
				}
			case field.Kind == Currency && !validCurrency(cell.Text):
				code = NotCurrency
			}
			if code != "" {
				note(index, 0, code)
			}
			made.cells[index] = OutCell{State: Value, Text: text}
		}
		if !texts[index] {
			if field.Format && provided {
				note(index, 1, RequiredNull)
			} else if field.Format {
				note(index, 1, RequiredNotProvided)
			}
			if field.Billing {
				made.lacking = append(made.lacking, index)
			}
		}
	}
	if costly && !texts[aim.currency] {
		note(aim.currency, 2, CurrencyMissing)
	}
	if bound.period {
		if held.spec.Window != nil {
			for _, index := range []int{aim.start, aim.end} {
				if !texts[index] {
					note(index, 3, PeriodMissing)
				}
			}
		}
		if start, end := times[aim.start], times[aim.end]; start != nil && end != nil {
			if !start.Before(*end) {
				note(aim.start, 4, PeriodInverted)
			}
			if window := held.spec.Window; window != nil && (start.Before(window.From) || end.After(window.To)) {
				note(aim.start, 5, OutsideWindow)
			}
		}
	}
	sort.SliceStable(made.findings, func(i, j int) bool {
		if made.findings[i].field != made.findings[j].field {
			return made.findings[i].field < made.findings[j].field
		}
		return made.findings[i].rank < made.findings[j].rank
	})
	if len(made.findings) > 0 {
		made.class = inError
	}
	return made
}

func (r *Registry) Close(workspaceID, id string) (State, []string, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return "", nil, ErrInvalid
	}
	_, held := r.find(workspaceID, id)
	if held == nil {
		return "", nil, ErrNotFound
	}
	if held.state == Closed || held.state == Discarded {
		return held.state, nil, nil
	}
	if held.state != Open {
		return "", nil, ErrRejected
	}
	var reasons []string
	if len(held.chunks) != held.spec.Chunks {
		reasons = append(reasons, ChunksMissing)
	}
	if held.checked != held.spec.Rows {
		reasons = append(reasons, RowCountDiffers)
	}
	if len(reasons) > 0 {
		return "", reasons, ErrRejected
	}
	bound := held.revision
	aim := bound.target
	sums := map[int]map[string]*number{}
	lacking := false
	for chunk := 1; chunk <= held.spec.Chunks; chunk++ {
		for _, made := range held.records[chunk] {
			switch made.class {
			case quarantined:
				held.apart = append(held.apart, QuarantinedRow{Chunk: made.chunk, Position: made.position, Key: made.key})
			case inError:
				for _, finding := range made.findings {
					held.findingsTotal++
					if len(held.findings) < r.limits.Findings {
						held.findings = append(held.findings, Finding{Chunk: made.chunk, Position: made.position, Key: made.key, Field: aim.stated.Fields[finding.field].Name, Code: finding.code})
					}
				}
			default:
				held.rows = append(held.rows, made)
				lacking = lacking || len(made.lacking) > 0
				for index, value := range made.numbers {
					if value == nil || aim.stated.Fields[index].Kind != Cost {
						continue
					}
					if sums[index] == nil {
						sums[index] = map[string]*number{}
					}
					currency := made.cells[aim.currency].Text
					if sums[index][currency] == nil {
						sums[index][currency] = &number{coefficient: new(big.Int)}
					}
					sums[index][currency].add(value)
				}
				made.numbers = nil
			}
		}
	}
	held.records, held.keys = nil, nil
	names := make([]string, 0, len(held.unused))
	for name := range held.unused {
		names = append(names, name)
	}
	sort.Strings(names)
	held.names.Total = len(names)
	if len(names) > r.limits.Unused {
		names = names[:r.limits.Unused]
	}
	held.names.Items, held.unused = names, nil

	format := &Verdict{Result: Conformant}
	billing := &Verdict{Result: Ready}
	if !bound.complete {
		format.Result = Partial
		format.Reasons = append(format.Reasons, MappingPartial)
	}
	if !bound.possible() {
		billing.Reasons = append(billing.Reasons, MappingGap)
	}
	if held.counts[inError] > 0 {
		format.Result = Nonconformant
		format.Reasons = append(format.Reasons, RowsInError)
		billing.Reasons = append(billing.Reasons, RowsInError)
	}
	if held.counts[quarantined] > 0 {
		format.Result = Nonconformant
		format.Reasons = append(format.Reasons, RowsQuarantined)
		billing.Reasons = append(billing.Reasons, RowsQuarantined)
	}
	if lacking {
		billing.Reasons = append(billing.Reasons, RowsLacking)
	}
	unexpected, differ, outside := false, false, false
	held.sums = map[string]map[string]string{}
	for index, field := range aim.stated.Fields {
		if field.Kind != Cost || bound.entries[index].Missing != "" {
			continue
		}
		stated, given := held.expected[index]
		if !given && field.Billing {
			unexpected = true
		}
		if given && len(stated) != len(sums[index]) {
			differ = true
		}
		for currency, sum := range sums[index] {
			if held.sums[field.Name] == nil {
				held.sums[field.Name] = map[string]string{}
			}
			held.sums[field.Name][currency] = sum.text()
			if !sum.fits(r.profile) {
				outside = true
			}
			if given && (stated[currency] == nil || !stated[currency].equals(sum)) {
				differ = true
			}
		}
	}
	if unexpected {
		billing.Reasons = append(billing.Reasons, TotalsNotExpected)
	}
	if differ {
		billing.Reasons = append(billing.Reasons, TotalsDiffer)
	}
	if bound.period && held.spec.Window == nil {
		billing.Reasons = append(billing.Reasons, WindowNotDeclared)
	}
	if outside {
		billing.Reasons = append(billing.Reasons, SumOutsideProfile)
	}
	if len(billing.Reasons) > 0 {
		billing.Result = NotReady
	}
	held.format, held.billing = format, billing
	held.state, held.changed = Closed, r.last
	return held.state, nil, nil
}

func (r *Registry) Abandon(workspaceID, id string, reason AbandonReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Abandoned {
		if held.abandon != reason {
			return ErrConflict
		}
		return nil
	}
	if held.state != Open {
		return ErrRejected
	}
	held.state, held.abandon, held.changed = Abandoned, reason, r.last
	r.drop(ws, held)
	return nil
}

func (r *Registry) Discard(workspaceID, id string, reason DiscardReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Discarded {
		if held.discard != reason {
			return ErrConflict
		}
		return nil
	}
	if held.state != Closed {
		return ErrRejected
	}
	held.state, held.discard, held.changed = Discarded, reason, r.last
	r.drop(ws, held)
	return nil
}
