package adjustments

import (
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func baseForm(b Base) bool {
	return validID(b.Run) && validDigest(b.Digest) && validCurrency(b.Currency) && b.Scale >= 0 && b.Scale <= maxScale && exact.ValidMode(b.Mode) &&
		(b.Issued && validID(b.Document) || !b.Issued && b.Document == "")
}

func (r *Registry) draftForm(d Draft) ([]*line, bool) {
	if !validID(d.ID) || !validID(d.Customer) || !validID(d.Contract) || !validDate(d.From) || !validDate(d.To) || d.To < d.From || !baseForm(d.Base) || len(d.Lines) == 0 {
		return nil, false
	}
	seen := map[string]bool{}
	var out []*line
	for _, s := range d.Lines {
		if !validID(s.ID) || !validID(s.Charge) || seen[s.ID] {
			return nil, false
		}
		seen[s.ID] = true
		l := &line{source: s}
		var ok bool
		if l.quantity, ok = r.number(s.Quantity, false, false); !ok {
			return nil, false
		}
		if l.amount, ok = r.number(s.Amount, true, false); !ok || !fits(l.amount, d.Base.Scale) {
			return nil, false
		}
		if s.Price != "" {
			if l.price, ok = r.number(s.Price, false, false); !ok {
				return nil, false
			}
			l.priced = true
		}
		out = append(out, l)
	}
	return out, true
}

func copyDraft(d Draft) Draft {
	d.Lines = append([]SourceLine(nil), d.Lines...)
	return d
}

func sameDraft(a, b Draft) bool {
	if a.ID != b.ID || a.Customer != b.Customer || a.Contract != b.Contract || a.From != b.From || a.To != b.To || a.Base != b.Base || len(a.Lines) != len(b.Lines) {
		return false
	}
	for index := range a.Lines {
		if a.Lines[index] != b.Lines[index] {
			return false
		}
	}
	return true
}

func (r *Registry) Open(workspace string, d Draft, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if len(d.Lines) > r.limits.Lines {
		return View{}, ErrLimit
	}
	lines, ok := r.draftForm(d)
	if !validID(workspace) || !ok || !validActor(actor) {
		return View{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s != nil && s.drafts[d.ID] != nil {
		known := s.drafts[d.ID]
		if !sameDraft(known.view.Draft, d) || known.view.Opener != actor {
			return View{}, ErrConflict
		}
		return viewOf(known), nil
	}
	made := &draft{lines: lines, byLine: map[string]*line{}, byEntry: map[string]*entry{}, checkers: map[string]bool{}}
	for _, l := range lines {
		made.byLine[l.source.ID] = l
	}
	made.view = View{Draft: copyDraft(d), Opener: actor, Opened: now, SourceDigest: sourceDigest(workspace, d), Revision: 1, State: StateOpen}
	result, ok := r.compute(made, nil)
	if !ok {
		return View{}, ErrLimit
	}
	if s == nil {
		if len(r.workspaces) >= r.limits.Workspaces {
			return View{}, ErrLimit
		}
		s = &space{drafts: map[string]*draft{}, requests: map[string]string{}, used: map[string]string{}, runs: map[string]string{}}
		r.workspaces[workspace] = s
	}
	if len(s.drafts) >= r.limits.Drafts {
		return View{}, ErrLimit
	}
	made.result = result
	r.seal(workspace, made)
	s.drafts[d.ID] = made
	return viewOf(made), nil
}

func (r *Registry) entryForm(e Entry) (exact.Fraction, bool) {
	if !validID(e.ID) || !entries[e.Kind] || !reasons[e.Reason] || !validDate(e.From) || !validDate(e.To) || e.To < e.From || len(e.Evidence) == 0 {
		return exact.Fraction{}, false
	}
	if e.Line != "" && !validID(e.Line) || e.Finding != "" && !validID(e.Finding) || e.Reason == ReasonSourceError && e.Finding == "" {
		return exact.Fraction{}, false
	}
	seen := map[string]bool{}
	for _, ref := range e.Evidence {
		if !validID(ref) || seen[ref] {
			return exact.Fraction{}, false
		}
		seen[ref] = true
	}
	switch e.Kind {
	case EntryQuantity:
		if e.Line == "" || e.Amount != "" {
			return exact.Fraction{}, false
		}
		return r.number(e.Target, false, false)
	case EntryFixed:
		if e.Target != "" {
			return exact.Fraction{}, false
		}
		return r.number(e.Amount, true, true)
	}
	if e.Amount != "" {
		return exact.Fraction{}, false
	}
	return r.number(e.Target, true, false)
}

func copyEntry(e Entry) Entry {
	e.Evidence = append([]string(nil), e.Evidence...)
	return e
}

func sameEntry(a, b Entry) bool {
	if a.ID != b.ID || a.Kind != b.Kind || a.Line != b.Line || a.Target != b.Target || a.Amount != b.Amount || a.Reason != b.Reason || a.Finding != b.Finding ||
		a.From != b.From || a.To != b.To || len(a.Evidence) != len(b.Evidence) {
		return false
	}
	for index := range a.Evidence {
		if a.Evidence[index] != b.Evidence[index] {
			return false
		}
	}
	return true
}

func open(d *draft) error {
	switch d.view.State {
	case StateReleased:
		return refuse(RefuseReleased)
	case StateWithdrawn:
		return refuse(RefuseWithdrawn)
	}
	return nil
}

func (r *Registry) Add(workspace, id string, revision int, e Entry, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	value, ok := r.entryForm(e)
	if !validID(workspace) || !validID(id) || revision < 1 || !ok || !validActor(actor) {
		return View{}, ErrInvalid
	}
	d, err := r.find(workspace, id)
	if err != nil {
		return View{}, err
	}
	if e.Kind != EntryQuantity && !fits(value, d.view.Draft.Base.Scale) {
		return View{}, ErrInvalid
	}
	if known := d.byEntry[e.ID]; known != nil {
		if sameEntry(known.view.Entry, e) && known.view.Author == actor && known.at == revision {
			return viewOf(d), nil
		}
		return View{}, ErrConflict
	}
	if e.Line != "" && d.byLine[e.Line] == nil {
		return View{}, ErrNotFound
	}
	if err := open(d); err != nil {
		return View{}, err
	}
	if revision != d.view.Revision {
		return View{}, refuse(RefuseStaleRevision)
	}
	if e.From < d.view.Draft.From || e.To > d.view.Draft.To {
		return View{}, refuse(RefuseOutsidePeriod)
	}
	if e.Kind == EntryQuantity && !d.byLine[e.Line].priced {
		return View{}, refuse(RefuseNoUnitPrice)
	}
	if len(e.Evidence) > r.limits.Evidence || len(d.entries) >= r.limits.Entries {
		return View{}, ErrLimit
	}
	made := &entry{view: EntryView{Entry: copyEntry(e), Author: actor, Written: now, Active: true}, at: revision, value: value}
	result, ok := r.compute(d, append(activeOf(d.entries), made))
	if !ok {
		return View{}, ErrLimit
	}
	if result.amounts[len(result.amounts)-1].Sign() == 0 {
		return View{}, refuse(RefuseZeroDifference)
	}
	if result.total.Sign() < 0 {
		return View{}, refuse(RefuseNegativeTotal)
	}
	d.entries = append(d.entries, made)
	d.byEntry[e.ID] = made
	d.result = result
	d.view.Revision++
	r.seal(workspace, d)
	return viewOf(d), nil
}

func (r *Registry) Remove(workspace, id string, revision int, entryID string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || revision < 1 || !validID(entryID) || !validActor(actor) {
		return View{}, ErrInvalid
	}
	d, err := r.find(workspace, id)
	if err != nil {
		return View{}, err
	}
	e := d.byEntry[entryID]
	if e == nil {
		return View{}, ErrNotFound
	}
	if !e.view.Active {
		if e.view.RemovedBy == actor && e.remove == revision {
			return viewOf(d), nil
		}
		return View{}, refuse(RefuseEntryRemoved)
	}
	if err := open(d); err != nil {
		return View{}, err
	}
	if revision != d.view.Revision {
		return View{}, refuse(RefuseStaleRevision)
	}
	e.view.Active = false
	result, ok := r.compute(d, activeOf(d.entries))
	if !ok {
		e.view.Active = true
		return View{}, ErrLimit
	}
	e.view.RemovedBy, e.view.Removed, e.remove = actor, now, revision
	d.result = result
	d.view.Revision++
	r.seal(workspace, d)
	return viewOf(d), nil
}

func (r *Registry) seal(workspace string, d *draft) {
	d.view.Digest = draftDigest(workspace, d)
}

func draftDigest(workspace string, d *draft) string {
	v := d.view
	p := newPrinter(draftLabel)
	p.fields(workspace, v.Draft.ID, v.Draft.Customer, v.Draft.Contract, v.Draft.From, v.Draft.To, v.SourceDigest, strconv.Itoa(v.Revision))
	active := activeOf(d.entries)
	p.field(strconv.Itoa(len(active)))
	for index, e := range active {
		x := e.view.Entry
		p.fields(x.ID, x.Kind, x.Line, x.Target, x.Amount, x.Reason, x.Finding, x.From, x.To, e.view.Author.ID, e.view.Author.Kind, strconv.Itoa(len(x.Evidence)))
		p.fields(x.Evidence...)
		diff := d.result.differences[index]
		p.fields(diff.Current, diff.Target, diff.Quantity, diff.Amount)
	}
	p.fields(show(d.result.source), show(d.result.adjustment), show(d.result.total), show(d.result.gross))
	return p.hex()
}

func sourceDigest(workspace string, d Draft) string {
	b := d.Base
	p := newPrinter(sourceLabel)
	p.fields(workspace, d.ID, b.Run, b.Digest, strconv.FormatBool(b.Issued), b.Document, b.Currency, strconv.Itoa(b.Scale), b.Mode, strconv.Itoa(len(d.Lines)))
	for _, l := range d.Lines {
		p.fields(l.ID, l.Charge, l.Quantity, l.Price, l.Amount)
	}
	return p.hex()
}

func viewOf(d *draft) View {
	v := d.view
	v.Draft = copyDraft(v.Draft)
	v.Entries = nil
	seen := map[string]bool{}
	for _, e := range d.entries {
		ev := e.view
		ev.Entry = copyEntry(ev.Entry)
		v.Entries = append(v.Entries, ev)
		if e.view.Active && ev.Entry.Finding != "" && !seen[ev.Entry.Finding] {
			seen[ev.Entry.Finding] = true
			v.Findings = append(v.Findings, ev.Entry.Finding)
		}
	}
	v.Differences = append([]Difference(nil), d.result.differences...)
	for index, l := range d.lines {
		v.Lines = append(v.Lines, LineView{Line: l.source, Adjusted: show(d.result.adjusted[index])})
	}
	v.Source, v.Adjustment, v.Total, v.Gross = show(d.result.source), show(d.result.adjustment), show(d.result.total), show(d.result.gross)
	if d.view.Check == nil || d.view.Check.Check.Revision != d.view.Revision {
		v.Subject = ""
	}
	if d.view.Check != nil {
		check := *d.view.Check
		v.Check = &check
	}
	if d.view.Outcome != nil {
		outcome := *d.view.Outcome
		outcome.Differences = append([]Difference(nil), outcome.Differences...)
		outcome.Release.Approvers = append([]Actor(nil), outcome.Release.Approvers...)
		v.Outcome = &outcome
	}
	return v
}

func (r *Registry) Of(workspace, id string) (View, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return View{}, ErrInvalid
	}
	d, err := r.find(workspace, id)
	if err != nil {
		return View{}, err
	}
	return viewOf(d), nil
}
