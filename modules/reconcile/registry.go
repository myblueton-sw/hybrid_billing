package reconcile

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type part struct {
	stated Part
	print  string
	amount *number
}

type line struct {
	stated Line
	print  string
	amount *number
}

type run struct {
	spec      RunSpec
	print     string
	subject   string
	document  string
	total     *number
	rounding  *number
	cost      *number
	state     State
	wasClosed bool
	parts     map[string]*part
	sources   map[string]bool
	lines     map[string]*line
	rounded   bool
	partIDs   []string
	lineIDs   []string
	internal  *number
	explained *number
	sums      map[Category]*number
	counts    map[Category]int
	residual  *number
	verdict   string
	reason    VoidReason
	opened    time.Time
	changed   time.Time
}

type workspace struct {
	runs     map[string]*run
	subjects map[string]*run
	highest  map[string]int
	byDoc    map[string][]*run
	items    map[string]*run
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	profile    decimal.Profile
	rounding   map[string]*number
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*workspace
}

func NewRegistry(limits Limits, rounding map[string]string, profile decimal.Profile, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || !profile.Valid() || clock == nil {
		return nil, ErrInvalid
	}
	bounds := map[string]*number{}
	for currency, text := range rounding {
		bound, parsed := parse(profile, text)
		if !validCurrency(currency) || !parsed || bound.sign() < 0 {
			return nil, ErrInvalid
		}
		bounds[currency] = bound
	}
	return &Registry{built: true, limits: limits, profile: profile, rounding: bounds, clock: clock, workspaces: map[string]*workspace{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock().UTC().Round(0); read.After(r.now) {
		r.now = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func join(fields ...string) string {
	var out strings.Builder
	for _, field := range fields {
		out.WriteString(strconv.Itoa(len(field)))
		out.WriteByte(':')
		out.WriteString(field)
	}
	return out.String()
}

func (s Subject) key() (string, string, bool) {
	none := s.Customer == "" && s.Period == "" && s.Charge == "" && s.Pool == ""
	bare := s.Issuer == "" && s.Account == "" && s.Document == "" && s.Revision == 0
	switch s.Basis {
	case ProviderDocument:
		if !none || !validID(s.Issuer) || !validID(s.Account) || !validID(s.Document) || s.Revision < 1 || s.Revision > maxRevision || !validCurrency(s.Currency) {
			return "", "", false
		}
		document := join(s.Issuer, s.Account, s.Document)
		return join(string(s.Basis), document, strconv.Itoa(s.Revision), s.Currency), document, true
	case IndependentPrice:
		if !bare || s.Currency != "" || s.Pool != "" || !validID(s.Customer) || !validID(s.Period) || !validID(s.Charge) {
			return "", "", false
		}
		return join(string(s.Basis), s.Customer, s.Period, s.Charge), "", true
	case InternalCost:
		if !bare || s.Currency != "" || s.Charge != "" || !validID(s.Customer) || !validID(s.Period) || !validID(s.Pool) {
			return "", "", false
		}
		return join(string(s.Basis), s.Customer, s.Period, s.Pool), "", true
	}
	return "", "", false
}

func (r *Registry) shape(s RunSpec) (*run, bool) {
	subject := Subject{Basis: s.Basis, Issuer: s.Issuer, Account: s.Account, Document: s.Document, Revision: s.Revision, Currency: s.Currency, Customer: s.Customer, Period: s.Period, Charge: s.Charge, Pool: s.Pool}
	if s.Basis != ProviderDocument {
		subject.Currency = ""
	}
	key, document, formed := subject.key()
	if !formed || !validID(s.WorkspaceID) || !validID(s.ID) {
		return nil, false
	}
	made := &run{spec: s, subject: key, document: document}
	prints := []string{s.ID, key}
	switch s.Basis {
	case ProviderDocument:
		if s.QuantityComplete || s.QuantityEvidence != "" || s.PriceEvidence != "" || s.Cost != "" || s.CostEvidence != "" || s.PoolEvidence != "" || s.AllocationEvidence != "" {
			return nil, false
		}
		total, parsed := parse(r.profile, s.Total)
		rounding, bounded := parse(r.profile, s.Rounding)
		if !parsed || !bounded || rounding.sign() < 0 || !validID(s.Input) || s.Parts < 1 {
			return nil, false
		}
		made.total, made.rounding = total, rounding
		made.spec.Total, made.spec.Rounding = total.text(), rounding.text()
		prints = append(prints, total.text(), s.Input, strconv.Itoa(s.Parts), rounding.text())
	case IndependentPrice:
		if s.Total != "" || s.Input != "" || s.Parts != 0 || s.Rounding != "" || s.PoolEvidence != "" || s.AllocationEvidence != "" || !validID(s.QuantityEvidence) || !validID(s.PriceEvidence) {
			return nil, false
		}
		if s.Cost != "" || s.Currency != "" || s.CostEvidence != "" {
			cost, parsed := parse(r.profile, s.Cost)
			if !parsed || cost.sign() < 0 || !validCurrency(s.Currency) || !validID(s.CostEvidence) {
				return nil, false
			}
			made.cost, made.spec.Cost = cost, cost.text()
			prints = append(prints, cost.text(), s.Currency, s.CostEvidence)
		}
		prints = append(prints, strconv.FormatBool(s.QuantityComplete), s.QuantityEvidence, s.PriceEvidence)
	case InternalCost:
		if s.Total != "" || s.Input != "" || s.Parts != 0 || s.Rounding != "" || s.QuantityComplete || s.QuantityEvidence != "" || s.PriceEvidence != "" || s.CostEvidence != "" {
			return nil, false
		}
		cost, parsed := parse(r.profile, s.Cost)
		if !parsed || cost.sign() < 0 || !validCurrency(s.Currency) || !validID(s.PoolEvidence) || !validID(s.AllocationEvidence) {
			return nil, false
		}
		made.cost, made.spec.Cost = cost, cost.text()
		prints = append(prints, cost.text(), s.Currency, s.PoolEvidence, s.AllocationEvidence)
	}
	made.print = join(prints...)
	return made, true
}

func (r *Registry) find(workspaceID, id string) (*workspace, *run) {
	ws := r.workspaces[workspaceID]
	if ws == nil {
		return nil, nil
	}
	return ws, ws.runs[id]
}

func (ws *workspace) superseded(held *run) bool {
	return held.spec.Basis == ProviderDocument && held.spec.Revision < ws.highest[held.document]
}

func (held *run) names(stated Line) string {
	if stated.ItemType == DocumentLine {
		return join(string(stated.ItemType), held.document, strconv.Itoa(held.spec.Revision), stated.Item)
	}
	return join(string(stated.ItemType), stated.Item)
}

func (ws *workspace) release(held *run) {
	for _, made := range held.lines {
		if made.stated.Category != Rounding && ws.items[held.names(made.stated)] == held {
			delete(ws.items, held.names(made.stated))
		}
	}
}

func (r *Registry) Open(spec RunSpec) (State, error) {
	defer r.leave()
	if !r.enter() {
		return "", ErrInvalid
	}
	made, formed := r.shape(spec)
	if !formed {
		return "", ErrInvalid
	}
	if spec.Parts > r.limits.Parts {
		return "", ErrLimit
	}
	ws := r.workspaces[spec.WorkspaceID]
	if ws != nil {
		if held := ws.runs[spec.ID]; held != nil {
			if held.print != made.print {
				return "", ErrConflict
			}
			return held.state, nil
		}
		if ws.subjects[made.subject] != nil {
			return "", ErrConflict
		}
	}
	if spec.Basis == ProviderDocument {
		if ws != nil && spec.Revision < ws.highest[made.document] {
			return "", ErrRejected
		}
		if bound := r.rounding[spec.Currency]; made.rounding.sign() > 0 && (bound == nil || made.rounding.exceeds(bound)) {
			return "", ErrRejected
		}
	}
	if ws != nil && len(ws.runs) >= r.limits.Runs {
		return "", ErrLimit
	}
	if ws == nil {
		ws = &workspace{runs: map[string]*run{}, subjects: map[string]*run{}, highest: map[string]int{}, byDoc: map[string][]*run{}, items: map[string]*run{}}
		r.workspaces[spec.WorkspaceID] = ws
	}
	made.state, made.opened, made.changed = Open, r.now, r.now
	made.parts, made.sources, made.lines = map[string]*part{}, map[string]bool{}, map[string]*line{}
	if spec.Basis != ProviderDocument {
		made.state, made.wasClosed = Closed, true
	} else {
		if spec.Revision > ws.highest[made.document] {
			ws.highest[made.document] = spec.Revision
			for _, earlier := range ws.byDoc[made.document] {
				ws.release(earlier)
			}
		}
		ws.byDoc[made.document] = append(ws.byDoc[made.document], made)
	}
	ws.runs[spec.ID] = made
	ws.subjects[made.subject] = made
	return made.state, nil
}

func (r *Registry) writable(ws *workspace, held *run, currency string) bool {
	return held.state == Open && !ws.superseded(held) && currency == held.spec.Currency
}

func (r *Registry) Post(workspaceID, id string, stated Part) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !validID(stated.ID) || !validCurrency(stated.Currency) || !stated.Kind.known() || !validID(stated.Source) {
		return "", ErrInvalid
	}
	amount, parsed := parse(r.profile, stated.Amount)
	if !parsed {
		return "", ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return "", ErrNotFound
	}
	print := join(stated.ID, amount.text(), stated.Currency, string(stated.Kind), stated.Source)
	if earlier := held.parts[stated.ID]; earlier != nil {
		if earlier.print != print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.sources[stated.Source] {
		return "", ErrConflict
	}
	if !r.writable(ws, held, stated.Currency) || stated.Kind != Billed || len(held.parts) >= held.spec.Parts {
		return "", ErrRejected
	}
	stated.Amount = amount.text()
	held.parts[stated.ID] = &part{stated: stated, print: print, amount: amount}
	held.sources[stated.Source] = true
	return held.state, nil
}

func (r *Registry) Explain(workspaceID, id string, stated Line) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !validID(stated.ID) || !stated.Category.known() || !validCurrency(stated.Currency) || !validID(stated.Evidence) {
		return "", ErrInvalid
	}
	amount, parsed := parse(r.profile, stated.Amount)
	if !parsed || amount.sign() == 0 {
		return "", ErrInvalid
	}
	if stated.Category == Rounding {
		if stated.ItemType != "" || stated.Item != "" {
			return "", ErrInvalid
		}
	} else if (stated.ItemType != DocumentLine && stated.ItemType != InternalItem) || !validID(stated.Item) {
		return "", ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return "", ErrNotFound
	}
	print := join(stated.ID, string(stated.Category), amount.text(), stated.Currency, string(stated.ItemType), stated.Item, stated.Evidence)
	if earlier := held.lines[stated.ID]; earlier != nil {
		if earlier.print != print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if !r.writable(ws, held, stated.Currency) {
		return "", ErrRejected
	}
	if stated.Category == Rounding {
		if held.rounded || amount.exceeds(held.rounding) {
			return "", ErrRejected
		}
	} else if ws.items[held.names(stated)] != nil {
		return "", ErrRejected
	}
	if len(held.lines) >= r.limits.Lines {
		return "", ErrLimit
	}
	stated.Amount = amount.text()
	held.lines[stated.ID] = &line{stated: stated, print: print, amount: amount}
	if stated.Category == Rounding {
		held.rounded = true
	} else {
		ws.items[held.names(stated)] = held
	}
	return held.state, nil
}

func (r *Registry) Close(workspaceID, id string) (State, string, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return "", "", ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return "", "", ErrNotFound
	}
	if held.wasClosed {
		return held.state, held.verdict, nil
	}
	if held.state != Open || ws.superseded(held) || len(held.parts) != held.spec.Parts {
		return "", "", ErrRejected
	}
	held.internal, held.explained = zero(), zero()
	held.sums, held.counts = map[Category]*number{}, map[Category]int{}
	for _, posted := range held.parts {
		held.internal.add(posted.amount, 1)
	}
	for _, made := range held.lines {
		if held.sums[made.stated.Category] == nil {
			held.sums[made.stated.Category] = zero()
		}
		held.sums[made.stated.Category].add(made.amount, 1)
		held.counts[made.stated.Category]++
		held.explained.add(made.amount, 1)
	}
	held.residual = held.total.copy()
	held.residual.add(held.internal, -1)
	held.residual.add(held.explained, -1)
	held.verdict = Unexplained
	if held.residual.sign() == 0 {
		held.verdict = Explained
	}
	held.state, held.wasClosed, held.changed = Closed, true, r.now
	return held.state, held.verdict, nil
}

func (r *Registry) Void(workspaceID, id string, reason VoidReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	ws, held := r.find(workspaceID, id)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Voided {
		if held.reason != reason {
			return ErrConflict
		}
		return nil
	}
	held.state, held.reason, held.changed = Voided, reason, r.now
	delete(ws.subjects, held.subject)
	ws.release(held)
	return nil
}
