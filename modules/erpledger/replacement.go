package erpledger

import "reflect"

func (r *Registry) ReplaceResult(workspace, key string, expected ResultView, next Document, allocations []Link, actor Actor) (ResultView, error) {
	return r.replaceResult(workspace, key, expected, next, allocations, actor, true)
}

func (r *Registry) CheckResultReplacement(workspace, key string, expected ResultView, next Document, allocations []Link, actor Actor) error {
	_, err := r.replaceResult(workspace, key, expected, next, allocations, actor, false)
	return err
}

func (r *Registry) replaceResult(workspace, key string, expected ResultView, next Document, allocations []Link, actor Actor, publish bool) (ResultView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) || !actor.valid() || !r.documentForm(next) {
		return ResultView{}, ErrInvalid
	}
	if err := actor.refusal(); err != nil {
		return ResultView{}, err
	}
	s, sub, err := r.find(workspace, key)
	if err != nil {
		return ResultView{}, err
	}
	current := resultView(s, sub)
	if !reflect.DeepEqual(current, expected) {
		return ResultView{}, ErrConflict
	}
	if sub.view.State != StatePrepared || sub.view.Diverged || len(sub.view.Commands) != 0 {
		return ResultView{}, refuse(ReasonWrongState)
	}
	if len(current.Documents) != 1 || !current.Reconciliation.Matched {
		return ResultView{}, ErrConflict
	}
	previous := current.Documents[0]
	if next.Connection != sub.view.Submission.Connection || next.Connection != previous.Document.Connection || next.External != previous.Document.External || next.Revision <= previous.Document.Revision {
		return ResultView{}, ErrConflict
	}
	pinned, ok := s.mapped(sub.view.Submission)
	if !ok || !reflect.DeepEqual(pinned, sub.view.Mappings) {
		return ResultView{}, ErrConflict
	}
	old := s.documents[externalKey(next.Connection, next.External)]
	if len(next.Rows) > r.limits.Rows || len(old.links)+len(allocations) > r.limits.Links {
		return ResultView{}, ErrLimit
	}
	active := []Link{}
	for _, link := range old.links {
		if link.sub != sub || link.view.Link.Key != key || link.view.Link.Connection != next.Connection || link.view.Link.External != next.External || !validID(link.view.Author) {
			return ResultView{}, ErrConflict
		}
		if !link.view.Stale {
			active = append(active, link.view.Link)
		}
	}
	if _, err := r.replacementLinks(s, sub, old.view.Document, old.rows, active, old.view.Author); err != nil {
		return ResultView{}, ErrConflict
	}
	rows, total := map[string]amounts{}, zero()
	for _, row := range next.Rows {
		values, ok := r.parse(sub.conn.view.Connection.Scale, row.Net, row.Tax, row.Quantity)
		if !ok {
			return ResultView{}, ErrInvalid
		}
		rows[row.ID] = values
		total.add(values)
	}
	scale := sub.conn.view.Connection.Scale
	if !r.bounded(total.net, scale) || !r.bounded(total.tax, scale) || !r.bounded(total.quantity, scale) {
		return ResultView{}, ErrLimit
	}
	links, err := r.replacementLinks(s, sub, next, rows, allocations, actor.ID)
	if err != nil {
		return ResultView{}, err
	}
	candidate := &document{conn: old.conn, rows: rows, author: actor.ID, at: now}
	next.Rows = append([]Row(nil), next.Rows...)
	candidate.view = DocumentView{Document: next, Total: total.show(scale), Author: actor.ID, At: now}
	for _, oldLink := range old.links {
		retained := *oldLink
		retained.view.Stale = true
		retained.values = zero()
		retained.values.add(oldLink.values)
		candidate.links = append(candidate.links, &retained)
	}
	candidate.links = append(candidate.links, links...)
	detached := *s
	detached.documents = make(map[string]*document, len(s.documents))
	for name, doc := range s.documents {
		detached.documents[name] = doc
	}
	detached.documents[externalKey(next.Connection, next.External)] = candidate
	result := resultView(&detached, sub)
	if !result.Reconciliation.Matched || len(result.Documents) != 1 || len(result.Reconciliation.Documents) != 1 || result.Reconciliation.Documents[0].Attributed != candidate.view.Total || candidate.view.Total != sub.view.Total {
		return ResultView{}, ErrConflict
	}
	if publish {
		s.documents[externalKey(next.Connection, next.External)] = candidate
	}
	return result, nil
}

func (r *Registry) replacementLinks(s *space, sub *submission, doc Document, rows map[string]amounts, allocations []Link, author string) ([]*link, error) {
	if len(allocations) != len(doc.Rows) || len(allocations) != len(sub.view.Submission.Lines) {
		return nil, ErrInvalid
	}
	products := map[string]string{}
	for _, row := range doc.Rows {
		products[row.ID] = row.Product
	}
	lineProducts := map[string]string{}
	for _, line := range sub.view.Submission.Lines {
		mapped := s.mappings[mappingKey{doc.Connection, "product", line.Product}]
		if mapped == nil {
			return nil, ErrConflict
		}
		lineProducts[line.ID] = mapped.Mapping.External
	}
	seenRows, seenLines := map[string]bool{}, map[string]bool{}
	result := make([]*link, 0, len(allocations))
	for _, allocation := range allocations {
		row, rowFound := rows[allocation.Row]
		line, lineFound := sub.lines[allocation.Line]
		if allocation.Connection != doc.Connection || allocation.External != doc.External || allocation.Key != sub.view.Submission.Key || !rowFound || !lineFound || seenRows[allocation.Row] || seenLines[allocation.Line] || products[allocation.Row] != lineProducts[allocation.Line] {
			return nil, ErrInvalid
		}
		if !r.number(allocation.Net) || !r.number(allocation.Tax) || !r.number(allocation.Quantity) {
			return nil, ErrInvalid
		}
		values, ok := r.parse(sub.conn.view.Connection.Scale, allocation.Net, allocation.Tax, allocation.Quantity)
		if !ok || !values.equal(row) || !values.equal(line) {
			return nil, ErrInvalid
		}
		seenRows[allocation.Row] = true
		seenLines[allocation.Line] = true
		result = append(result, &link{view: LinkView{Link: allocation, Author: author}, values: values, sub: sub})
	}
	return result, nil
}
