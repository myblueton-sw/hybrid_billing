package erpledger

import (
	"math/big"
	"sort"
	"time"
)

type link struct {
	view   LinkView
	values amounts
	sub    *submission
}

type document struct {
	view   DocumentView
	conn   *connection
	rows   map[string]amounts
	links  []*link
	author string
	at     time.Time
}

func (r *Registry) documentForm(d Document) bool {
	if !validID(d.Connection) || !validID(d.External) || d.Revision < 1 || len(d.Rows) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, row := range d.Rows {
		if !validID(row.ID) || seen[row.ID] || !validID(row.Product) || !r.number(row.Net) || !r.number(row.Tax) || !r.number(row.Quantity) {
			return false
		}
		seen[row.ID] = true
	}
	return true
}

func sameDocument(a, b Document) bool {
	if a.Connection != b.Connection || a.External != b.External || a.Revision != b.Revision || len(a.Rows) != len(b.Rows) {
		return false
	}
	for index := range a.Rows {
		if a.Rows[index] != b.Rows[index] {
			return false
		}
	}
	return true
}

func (d *document) copy() DocumentView {
	v := d.view
	v.Document.Rows = append([]Row(nil), v.Document.Rows...)
	v.Links = nil
	for _, l := range d.links {
		v.Links = append(v.Links, l.view)
	}
	return v
}

func (r *Registry) parse(scale int, net, tax, quantity string) (amounts, bool) {
	n, a := r.units(net, scale)
	t, b := r.units(tax, scale)
	q, c := r.units(quantity, scale)
	return amounts{n, t, q}, a && b && c
}

func (r *Registry) Document(workspace string, d Document, actor Actor) (DocumentView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !r.documentForm(d) || !actor.valid() {
		return DocumentView{}, ErrInvalid
	}
	s, c := r.connection(workspace, d.Connection)
	if c == nil {
		return DocumentView{}, ErrNotFound
	}
	k := externalKey(d.Connection, d.External)
	existing := s.documents[k]
	if actor.Kind == KindAssistant {
		return DocumentView{}, refuse(ReasonAINotAllowed)
	}
	if existing != nil && d.Revision <= existing.view.Document.Revision {
		if sameDocument(existing.view.Document, d) {
			return existing.copy(), nil
		}
		return DocumentView{}, ErrConflict
	}
	scale := c.view.Connection.Scale
	rows, total := map[string]amounts{}, zero()
	for _, row := range d.Rows {
		values, ok := r.parse(scale, row.Net, row.Tax, row.Quantity)
		if !ok {
			return DocumentView{}, ErrInvalid
		}
		rows[row.ID] = values
		total.add(values)
	}
	if len(d.Rows) > r.limits.Rows || existing == nil && len(s.documents) >= r.limits.Documents ||
		!r.bounded(total.net, scale) || !r.bounded(total.tax, scale) || !r.bounded(total.quantity, scale) {
		return DocumentView{}, ErrLimit
	}
	d.Rows = append([]Row(nil), d.Rows...)
	if existing == nil {
		existing = &document{conn: c}
		s.documents[k] = existing
	}
	existing.view.Document, existing.view.Total, existing.view.Author, existing.view.At = d, total.show(scale), actor.ID, now
	existing.rows = rows
	for _, l := range existing.links {
		row, ok := rows[l.view.Link.Row]
		if !ok || !fits(l.values, row) {
			l.view.Stale = true
		}
	}
	existing.refit()
	return existing.copy(), nil
}

func (d *document) refit() {
	sums := map[string]amounts{}
	for _, l := range d.links {
		if l.view.Stale {
			continue
		}
		sum, ok := sums[l.view.Link.Row]
		if !ok {
			sum = zero()
		}
		sum.add(l.values)
		sums[l.view.Link.Row] = sum
		if !fits(sum, d.rows[l.view.Link.Row]) {
			l.view.Stale = true
		}
	}
}

func signed(part, whole *big.Int) bool {
	if part.Sign() == 0 {
		return true
	}
	return part.Sign() == whole.Sign() && new(big.Int).Abs(part).Cmp(new(big.Int).Abs(whole)) <= 0
}

func fits(part, whole amounts) bool {
	return signed(part.net, whole.net) && signed(part.tax, whole.tax) && signed(part.quantity, whole.quantity)
}

func (s *space) allocated(match func(*link) bool) amounts {
	sum := zero()
	for _, d := range s.documents {
		for _, l := range d.links {
			if !l.view.Stale && match(l) {
				sum.add(l.values)
			}
		}
	}
	return sum
}

func (r *Registry) Link(workspace string, l Link, actor Actor) (DocumentView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(l.Connection) || !validID(l.External) || !validID(l.Row) || !validID(l.Key) || !validID(l.Line) ||
		!r.number(l.Net) || !r.number(l.Tax) || !r.number(l.Quantity) || !actor.valid() {
		return DocumentView{}, ErrInvalid
	}
	s, c := r.connection(workspace, l.Connection)
	if c == nil {
		return DocumentView{}, ErrNotFound
	}
	d := s.documents[externalKey(l.Connection, l.External)]
	if d == nil {
		return DocumentView{}, refuse(ReasonUnknownDocument)
	}
	for _, seen := range d.links {
		if !seen.view.Stale && seen.view.Link.Row == l.Row && seen.view.Link.Key == l.Key && seen.view.Link.Line == l.Line {
			if seen.view.Link == l && actor.is(seen.view.Author) {
				return d.copy(), nil
			}
			return DocumentView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return DocumentView{}, err
	}
	sub := s.submissions[l.Key]
	if sub == nil || sub.lines[l.Line].net == nil {
		return DocumentView{}, refuse(ReasonUnknownSubmission)
	}
	if sub.conn != c {
		return DocumentView{}, refuse(ReasonConnectionMismatch)
	}
	row, ok := d.rows[l.Row]
	if !ok {
		return DocumentView{}, refuse(ReasonUnknownDocument)
	}
	if !sub.live() {
		return DocumentView{}, refuse(ReasonWrongState)
	}
	values, fine := r.parse(c.view.Connection.Scale, l.Net, l.Tax, l.Quantity)
	if !fine {
		return DocumentView{}, ErrInvalid
	}
	line := sub.lines[l.Line]
	byRow := s.allocated(func(x *link) bool {
		return x.view.Link.External == l.External && x.view.Link.Connection == l.Connection && x.view.Link.Row == l.Row
	})
	byLine := s.allocated(func(x *link) bool { return x.view.Link.Key == l.Key && x.view.Link.Line == l.Line })
	byRow.add(values)
	byLine.add(values)
	if !fits(byRow, row) || !fits(byLine, line) {
		return DocumentView{}, refuse(ReasonOverAllocated)
	}
	if len(d.links) >= r.limits.Links {
		return DocumentView{}, ErrLimit
	}
	d.links = append(d.links, &link{view: LinkView{Link: l, Author: actor.ID}, values: values, sub: sub})
	return d.copy(), nil
}

func (r *Registry) DocumentOf(workspace, connection, external string) (DocumentView, error) {
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.documents[externalKey(connection, external)] == nil {
		return DocumentView{}, ErrNotFound
	}
	return s.documents[externalKey(connection, external)].copy(), nil
}

func (r *Registry) Reconcile(workspace, key string) (Reconciliation, error) {
	r.enter()
	defer r.leave()
	s, sub, err := r.find(workspace, key)
	if err != nil {
		return Reconciliation{}, err
	}
	return reconciliation(s, sub), nil
}

func reconciliation(s *space, sub *submission) Reconciliation {
	key := sub.view.Submission.Key
	scale := sub.conn.view.Connection.Scale
	out := Reconciliation{Key: key, Matched: true}
	for _, l := range sub.view.Submission.Lines {
		linked := s.allocated(func(x *link) bool { return x.view.Link.Key == key && x.view.Link.Line == l.ID })
		matched := linked.equal(sub.lines[l.ID])
		out.Lines = append(out.Lines, LineResult{Line: l.ID, Linked: linked.show(scale), Matched: matched})
		out.Matched = out.Matched && matched
	}
	var docs []*document
	for _, d := range s.documents {
		for _, l := range d.links {
			if l.view.Link.Key == key {
				docs = append(docs, d)
				break
			}
		}
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].view.Document.Connection+"|"+docs[i].view.Document.External < docs[j].view.Document.Connection+"|"+docs[j].view.Document.External
	})
	for _, d := range docs {
		attributed := zero()
		for _, l := range d.links {
			if !l.view.Stale && l.sub.live() {
				attributed.add(l.values)
			}
		}
		out.Documents = append(out.Documents, DocumentTotals{Connection: d.view.Document.Connection, External: d.view.Document.External, Total: d.view.Total, Attributed: attributed.show(d.conn.view.Connection.Scale)})
	}
	return out
}
