package exports

import (
	"bytes"

	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type artifact struct {
	view          Artifact
	rows          map[string][3]string
	results       map[string]Result
	registrations int
	reference     string
	state         string
	handoff       *Handoff
}

type space struct {
	artifacts map[string]*artifact
}

type Registry struct {
	mu         sync.Mutex
	split      int
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if !validID(config.Generator) || !validID(config.Schema) || !config.Numbers.Valid() || limits.Workspaces < 1 || limits.Workspaces > maxWorkspaces ||
		limits.Artifacts < 1 || limits.Artifacts > maxArtifacts || limits.Lines < 1 || limits.Lines > maxLines || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{split: sheetRows, config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func refuse(reason string) error { return Refusal{Reason: reason} }

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func hashOf(texts ...string) string {
	sum := sha256.New()
	for _, text := range texts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		sum.Write(size[:])
		sum.Write([]byte(text))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

type built struct {
	files    []File
	contract Contract
	data     string
	inputs   string
	rows     map[string][3]string
}

func (r *Registry) amount(text string, scale int) (*big.Int, bool) {
	if len(text) > maxNumberBytes {
		return nil, false
	}
	value, err := decimal.Parse(r.config.Numbers, text)
	if err != nil || value.Scale() > scale {
		return nil, false
	}
	return new(big.Int).Mul(value.Coefficient(), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale-value.Scale())), nil)), true
}

func (r *Registry) form(req Request) bool {
	s := req.Spec
	if !validID(req.ID) || len(req.ID) > maxArtifactID || req.Kind != KindSettlement && req.Kind != KindCustomer || !validZone(req.TimeZone) || !validID(s.Connection) || !validID(s.Submission) ||
		!validDigest(s.Approval) || !validID(s.Customer) || !validMonth(s.Period) || !validCurrency(s.Currency) || s.Scale < 0 || s.Scale > maxScale ||
		len(s.Lines) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, l := range s.Lines {
		if !validID(l.Row) || seen[l.Row] || !validText(l.Source) || !validText(l.Product) || !validText(l.TaxCode) {
			return false
		}
		for _, amount := range []string{l.Quantity, l.Net, l.Tax} {
			if _, ok := r.amount(amount, s.Scale); !ok {
				return false
			}
		}
		seen[l.Row] = true
	}
	return true
}

func digestRows(texts []string, rows [][]string) []string {
	texts = append(texts, strconv.Itoa(len(rows)))
	for _, row := range rows {
		texts = append(texts, strconv.Itoa(len(row)))
		texts = append(texts, row...)
	}
	return texts
}

func (r *Registry) build(req Request) (built, bool) {
	generator, schema, split := r.config.Generator, r.config.Schema, r.split
	s := req.Spec
	lines := append([]Line(nil), s.Lines...)
	sort.Slice(lines, func(i, j int) bool { return lines[i].Row < lines[j].Row })
	contract := Contract{Encoding: "UTF-8", Separator: ",", LineEnd: "\n", Decimal: ".", Sign: "leading_minus", Period: "YYYY-MM", TimeZone: req.TimeZone, Scale: s.Scale, SortKey: "row", ViewingBOM: true, ViewingPrecision: "not_precision_safe", Profile: r.config.Numbers.ID(), ProfileRevision: r.config.Numbers.Revision()}
	settlement := req.Kind == KindSettlement
	header := [][]string{{"field", "value"}, {"customer", s.Customer}, {"period", s.Period}, {"currency", s.Currency}, {"scale", strconv.Itoa(s.Scale)}}
	if settlement {
		header = [][]string{{"field", "value"}, {"submission", s.Submission}, {"connection", s.Connection}, {"customer", s.Customer}, {"period", s.Period}, {"currency", s.Currency}, {"scale", strconv.Itoa(s.Scale)}, {"approval", s.Approval}}
	}
	detailHead := []string{"row", "product", "tax_code", "quantity", "net", "tax"}
	if settlement {
		detailHead = []string{"row", "source", "product", "tax_code", "quantity", "net", "tax"}
	}
	detail := [][]string{detailHead}
	result := [][]string{{"row", "submission", "erp_reference", "erp_state", "net", "tax", "quantity"}}
	taxes := map[string][2]*big.Int{}
	totals := [3]*big.Int{new(big.Int), new(big.Int), new(big.Int)}
	rows := map[string][3]string{}
	for _, l := range lines {
		q, _ := r.amount(l.Quantity, s.Scale)
		n, _ := r.amount(l.Net, s.Scale)
		x, _ := r.amount(l.Tax, s.Scale)
		qs, ns, xs := show(q, s.Scale), show(n, s.Scale), show(x, s.Scale)
		rows[l.Row] = [3]string{ns, xs, qs}
		if settlement {
			detail = append(detail, []string{l.Row, l.Source, l.Product, l.TaxCode, qs, ns, xs})
		} else {
			detail = append(detail, []string{l.Row, l.Product, l.TaxCode, qs, ns, xs})
		}
		result = append(result, []string{l.Row, s.Submission, "", "", ns, xs, qs})
		t, ok := taxes[l.TaxCode]
		if !ok {
			t = [2]*big.Int{new(big.Int), new(big.Int)}
			taxes[l.TaxCode] = t
		}
		t[0].Add(t[0], n)
		t[1].Add(t[1], x)
		totals[0].Add(totals[0], n)
		totals[1].Add(totals[1], x)
		totals[2].Add(totals[2], q)
	}
	sums := append([]*big.Int(nil), totals[:]...)
	for _, t := range taxes {
		sums = append(sums, t[0], t[1])
	}
	for _, v := range sums {
		if _, err := decimal.New(r.config.Numbers, v, s.Scale); err != nil {
			return built{}, false
		}
	}
	codes := make([]string, 0, len(taxes))
	for code := range taxes {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	tax := [][]string{{"tax_code", "net", "tax"}}
	for _, code := range codes {
		tax = append(tax, []string{code, show(taxes[code][0], s.Scale), show(taxes[code][1], s.Scale)})
	}
	recon := [][]string{{"lines", "net", "tax", "quantity"}, {strconv.Itoa(len(lines)), show(totals[0], s.Scale), show(totals[1], s.Scale), show(totals[2], s.Scale)}}
	versions := [][]string{{"generator", generator}, {"schema", schema}}
	sheets := []sheet{{"header", append(append([][]string(nil), header...), versions...)}}
	for part, first := 1, 1; first < len(detail) || part == 1; part++ {
		last := first + split
		if last > len(detail) {
			last = len(detail)
		}
		name := "detail"
		if part > 1 {
			name = "detail-" + strconv.Itoa(part)
		}
		sheets = append(sheets, sheet{name, append([][]string{detail[0]}, detail[first:last]...)})
		first = last
	}
	sheets = append(sheets, sheet{"tax", tax})
	if settlement {
		sheets = append(sheets, sheet{"reconciliation", recon}, sheet{"result", result})
	}
	texts := []string{dataLabel, req.Kind, contract.Encoding, contract.Separator, contract.LineEnd, contract.Decimal, contract.Sign, contract.Period, strconv.Itoa(contract.Scale), contract.SortKey, contract.ViewingPrecision, contract.Profile, contract.ProfileRevision}
	for _, part := range [][][]string{header, detail, tax} {
		texts = digestRows(texts, part)
	}
	if settlement {
		texts = digestRows(digestRows(texts, recon), result)
	}
	files := []File{
		{Name: req.ID + ".xlsx", Media: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Bytes: workbook(sheets)},
		{Name: req.ID + ".detail.csv", Media: "text/csv", Bytes: csvBytes(detail, true)},
	}
	if settlement {
		machine := append([][]string{{machineMarker, schema}}, detail...)
		files = append(files, File{Name: req.ID + ".machine.csv", Media: "text/csv", Bytes: csvBytes(machine, false)})
	}
	for i := range files {
		sum := sha256.Sum256(files[i].Bytes)
		files[i].Digest = hex.EncodeToString(sum[:])
	}
	in := []string{inputsLabel, req.ID, req.Kind, req.TimeZone, s.Connection, s.Submission, s.Approval, s.Customer, s.Period, s.Currency, strconv.Itoa(s.Scale)}
	for _, l := range lines {
		in = append(in, l.Row, l.Source, l.Product, l.TaxCode, l.Quantity, l.Net, l.Tax)
	}
	return built{files: files, contract: contract, data: hashOf(texts...), inputs: hashOf(in...), rows: rows}, true
}

func copyArtifact(a Artifact) Artifact {
	a.Request.Spec.Lines = append([]Line(nil), a.Request.Spec.Lines...)
	files := make([]File, len(a.Files))
	for i, f := range a.Files {
		f.Bytes = append([]byte(nil), f.Bytes...)
		files[i] = f
	}
	a.Files = files
	if a.Confirmed != nil {
		c := *a.Confirmed
		a.Confirmed = &c
	}
	return a
}

func (r *Registry) oversized(req Request) bool {
	return len(req.Spec.Lines) > r.limits.Lines || req.Kind == KindSettlement && len(req.Spec.Lines) > maxSettlement
}

func (r *Registry) Generate(workspace string, req Request, actor Actor) (Artifact, error) {
	now := r.enter()
	defer r.leave()
	if r.oversized(req) {
		return Artifact{}, ErrLimit
	}
	if !validID(workspace) || !r.form(req) || !actor.valid() {
		return Artifact{}, ErrInvalid
	}
	if actor.Kind == KindAssistant {
		return Artifact{}, refuse(ReasonAINotAllowed)
	}
	b, fits := r.build(req)
	s := r.workspaces[workspace]
	if s != nil {
		if seen := s.artifacts[req.ID]; seen != nil {
			if seen.view.Inputs == b.inputs && seen.view.Volatile.Actor == actor.ID {
				return copyArtifact(seen.view), nil
			}
			return Artifact{}, ErrConflict
		}
	}
	if !fits {
		return Artifact{}, ErrLimit
	}
	if s == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return Artifact{}, ErrLimit
		}
		s = &space{artifacts: map[string]*artifact{}}
		r.workspaces[workspace] = s
	} else if len(s.artifacts) >= r.limits.Artifacts {
		return Artifact{}, ErrLimit
	}
	req.Spec.Lines = append([]Line(nil), req.Spec.Lines...)
	view := Artifact{Request: req, Files: b.files, Contract: b.contract, Generator: r.config.Generator, Schema: r.config.Schema, Data: b.data, Inputs: b.inputs, Volatile: Volatile{Actor: actor.ID, At: now}}
	s.artifacts[req.ID] = &artifact{view: view, rows: b.rows, results: map[string]Result{}}
	return copyArtifact(view), nil
}

func (r *Registry) find(workspace, id string) (*artifact, error) {
	s := r.workspaces[workspace]
	if s == nil || s.artifacts[id] == nil {
		return nil, ErrNotFound
	}
	return s.artifacts[id], nil
}

func (r *Registry) ArtifactOf(workspace, id string) (Artifact, error) {
	r.enter()
	defer r.leave()
	a, err := r.find(workspace, id)
	if err != nil {
		return Artifact{}, err
	}
	return copyArtifact(a.view), nil
}

func (r *Registry) Confirm(workspace, id string, actor Actor) (Artifact, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return Artifact{}, ErrInvalid
	}
	a, err := r.find(workspace, id)
	if err != nil {
		return Artifact{}, err
	}
	if a.view.Confirmed != nil {
		if actor.Kind == KindHuman && a.view.Confirmed.Actor == actor.ID {
			return copyArtifact(a.view), nil
		}
		return Artifact{}, refuse(ReasonConfirmed)
	}
	switch actor.Kind {
	case KindAssistant:
		return Artifact{}, refuse(ReasonAINotAllowed)
	case KindSystem:
		return Artifact{}, refuse(ReasonNotHuman)
	}
	if actor.ID == a.view.Volatile.Actor {
		return Artifact{}, refuse(ReasonSelfApproval)
	}
	a.view.Confirmed = &Volatile{Actor: actor.ID, At: now}
	return copyArtifact(a.view), nil
}

func (r *Registry) Download(workspace, id, name string, actor Actor) ([]byte, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || name == "" || !actor.valid() {
		return nil, ErrInvalid
	}
	a, err := r.find(workspace, id)
	if err != nil {
		return nil, err
	}
	if actor.Kind == KindAssistant {
		return nil, refuse(ReasonAINotAllowed)
	}
	if a.view.Confirmed == nil {
		return nil, refuse(ReasonNotConfirmed)
	}
	for _, f := range a.view.Files {
		if f.Name == name {
			return append([]byte(nil), f.Bytes...), nil
		}
	}
	return nil, ErrNotFound
}

func (r *Registry) Compare(workspace, id string, req Request, actor Actor) (Comparison, error) {
	r.enter()
	defer r.leave()
	if r.oversized(req) {
		return Comparison{}, ErrLimit
	}
	if !validID(workspace) || !validID(id) || !r.form(req) || !actor.valid() {
		return Comparison{}, ErrInvalid
	}
	a, err := r.find(workspace, id)
	if err != nil {
		return Comparison{}, err
	}
	if actor.Kind == KindAssistant {
		return Comparison{}, refuse(ReasonAINotAllowed)
	}
	b, fits := r.build(req)
	if !fits {
		return Comparison{}, ErrLimit
	}
	return Comparison{DataEqual: b.data == a.view.Data, GeneratorChanged: r.config.Generator != a.view.Generator, SchemaChanged: r.config.Schema != a.view.Schema, Data: b.data}, nil
}

func (r *Registry) RegisterResults(workspace, id string, data []byte, actor Actor) (Registration, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return Registration{}, ErrInvalid
	}
	a, err := r.find(workspace, id)
	if err != nil {
		return Registration{}, err
	}
	if actor.Kind == KindAssistant {
		return Registration{}, refuse(ReasonAINotAllowed)
	}
	if a.view.Request.Kind != KindSettlement {
		return Registration{}, refuse(ReasonNotSettlement)
	}
	if a.view.Confirmed == nil {
		return Registration{}, refuse(ReasonNotConfirmed)
	}
	if len(data) > maxResultBytes {
		return Registration{}, ErrLimit
	}
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	if !utf8.Valid(data) {
		return Registration{}, refuse(ReasonNotUTF8)
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	head, err := reader.Read()
	if err != nil || len(head) != 7 || strings.Join(head, "|") != "row|submission|erp_reference|erp_state|net|tax|quantity" {
		return Registration{}, refuse(ReasonWrongFormat)
	}
	var out []Result
	seen := map[string]bool{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) != 7 {
			return Registration{}, ErrInvalid
		}
		if len(seen) >= len(a.rows) {
			return Registration{}, ErrLimit
		}
		for i := range record {
			record[i] = raw(record[i])
		}
		row, submission, reference, state := record[0], record[1], record[2], record[3]
		if !validID(row) || !validID(submission) || seen[row] || reference != "" && !validID(reference) || !resultStates[state] || (state == "draft" || state == "posted") && reference == "" || state == "" && reference != "" {
			return Registration{}, ErrInvalid
		}
		seen[row] = true
		expected, ok := a.rows[row]
		if !ok {
			return Registration{}, Refusal{Reason: ReasonUnknownRow, Row: row}
		}
		if submission != a.view.Request.Spec.Submission {
			return Registration{}, Refusal{Reason: ReasonSubmissionMismatch, Row: row}
		}
		if record[4] != expected[0] || record[5] != expected[1] || record[6] != expected[2] {
			return Registration{}, Refusal{Reason: ReasonAmountChanged, Row: row}
		}
		if state == "" {
			continue
		}
		out = append(out, Result{Row: row, Reference: reference, State: state})
	}
	shared := a.reference
	for _, res := range out {
		if res.State != out[0].State || res.Reference != "" && shared != "" && res.Reference != shared {
			return Registration{}, refuse(ReasonMixedResults)
		}
		if shared == "" {
			shared = res.Reference
		}
	}
	for i := range out {
		if out[i].Reference == "" {
			out[i].Reference = shared
		}
	}
	var fresh []Result
	for _, res := range out {
		prior, ok := a.results[res.Row]
		if ok && prior == res {
			continue
		}
		if ok && !forward[prior.State][res.State] {
			return Registration{}, ErrConflict
		}
		fresh = append(fresh, res)
	}
	reg := Registration{Rows: out}
	if len(fresh) == 0 {
		if a.handoff != nil {
			h := *a.handoff
			reg.Handoff = &h
		}
		return reg, nil
	}
	if a.state != "" && out[0].State != a.state && !forward[a.state][out[0].State] {
		return Registration{}, ErrConflict
	}
	a.reference, a.state = shared, out[0].State
	for _, res := range fresh {
		a.results[res.Row] = res
	}
	a.registrations++
	spec := a.view.Request.Spec
	a.handoff = &Handoff{Observation: id + ":" + strconv.Itoa(a.registrations), Connection: spec.Connection, Submission: spec.Submission, Reference: a.reference, State: out[0].State, Revision: a.registrations}
	h := *a.handoff
	reg.Handoff = &h
	return reg, nil
}
