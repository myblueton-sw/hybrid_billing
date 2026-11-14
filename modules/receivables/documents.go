package receivables

import "time"

func receivableForm(v Receivable) bool {
	if !validID(v.Document) || !validID(v.Customer) || !validID(v.Entity) || !validCurrency(v.Currency) || !validID(v.Contract) || v.IssuedAt.IsZero() || !obligations[v.Obligation] {
		return false
	}
	if v.Replaces == "" {
		return v.DueBasis == ""
	}
	return v.Obligation == ObligationPayable && validID(v.Replaces) && v.Replaces != v.Document && (v.DueBasis == BasisInherit || v.DueBasis == BasisReissue)
}

func viewDocument(d *document) DocumentView {
	v := d.view
	v.Extensions = make([]ExtensionView, 0, len(d.extensions))
	for _, e := range d.extensions {
		v.Extensions = append(v.Extensions, e.view)
	}
	v.Allocations = make([]AllocationView, 0, len(d.allocations))
	for _, a := range d.allocations {
		v.Allocations = append(v.Allocations, a.view)
	}
	v.Reversals = make([]ReversalView, 0, len(d.reversals))
	for _, rev := range d.reversals {
		v.Reversals = append(v.Reversals, rev.view)
	}
	v.Holds = make([]HoldView, 0, len(d.holds))
	for _, h := range d.holds {
		v.Holds = append(v.Holds, h.view)
	}
	return v
}

func (r *Registry) Issue(workspace string, v Receivable, actor string) (DocumentView, error) {
	if !validID(workspace) || !validID(actor) || !receivableForm(v) {
		return DocumentView{}, ErrInvalid
	}
	v.IssuedAt = v.IssuedAt.UTC().Round(0)
	amount, printed, ok := r.amount(v.Amount)
	if !ok {
		return DocumentView{}, ErrInvalid
	}
	v.Amount = printed
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	var replaced *document
	if v.Replaces != "" {
		if s == nil || s.documents[v.Replaces] == nil {
			return DocumentView{}, ErrNotFound
		}
		replaced = s.documents[v.Replaces]
	}
	if s != nil {
		if old := s.documents[v.Document]; old != nil {
			if old.view.Receivable != v || old.view.Author != actor {
				return DocumentView{}, ErrConflict
			}
			return viewDocument(old), nil
		}
	}
	location := time.UTC
	if v.Obligation == ObligationPayable && s != nil && r.zoneOf(s, v.Contract) != nil {
		location = r.zoneOf(s, v.Contract)
	}
	date := dateOf(v.IssuedAt, location)
	if !validDate(date) {
		return DocumentView{}, ErrInvalid
	}
	if v.IssuedAt.After(now) {
		return DocumentView{}, ErrRejected
	}
	made := &document{amount: amount}
	if v.Obligation == ObligationPayable {
		if s == nil || r.zoneOf(s, v.Contract) == nil {
			return DocumentView{}, refuse(ReasonTermMissing)
		}
		b, err := r.cover(s, v.Contract, date)
		if err != nil {
			return DocumentView{}, err
		}
		if replaced != nil {
			p := replaced.view.Receivable
			if p.Obligation != ObligationPayable || p.Customer != v.Customer || p.Entity != v.Entity || p.Currency != v.Currency || p.Contract != v.Contract || replaced.view.ReplacedBy != "" {
				return DocumentView{}, ErrRejected
			}
		}
		t := s.terms[[2]string{b.view.Binding.Term, itoa(b.view.Binding.Revision)}].view.Term
		snap := Snapshot{Binding: b.view.Binding.ID, Term: t.ID, Revision: t.Revision, NetDays: t.NetDays, Mode: t.Mode, Zone: b.view.Binding.Zone, IssueDate: date, OriginalDue: addDays(date, t.NetDays)}
		snap.EffectiveDue = snap.OriginalDue
		if v.DueBasis == BasisInherit {
			snap.OriginalDue, snap.EffectiveDue = replaced.view.Snapshot.OriginalDue, replaced.view.Snapshot.EffectiveDue
		}
		made.binding, made.location, made.view.Snapshot = b, location, snap
	}
	if s != nil && len(s.documents) >= r.limits.Receivables {
		return DocumentView{}, ErrLimit
	}
	s, err := r.open(workspace)
	if err != nil {
		return DocumentView{}, err
	}
	made.view.Receivable, made.view.Author, made.view.Written = v, actor, now
	made.view.Digest = documentDigest(workspace, v, made.view.Snapshot, actor)
	s.documents[v.Document] = made
	if made.binding != nil {
		made.binding.issued = append(made.binding.issued, made)
	}
	if replaced != nil {
		replaced.view.ReplacedBy = v.Document
	}
	return viewDocument(made), nil
}

func (r *Registry) Document(workspace, id string) (DocumentView, error) {
	if !validID(workspace) || !validID(id) {
		return DocumentView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.documents[id] == nil {
		return DocumentView{}, ErrNotFound
	}
	return viewDocument(s.documents[id]), nil
}

func (r *Registry) find(workspace, id string) *document {
	s := r.workspaces[workspace]
	if s == nil {
		return nil
	}
	return s.documents[id]
}

func extensionForm(e Extension) bool {
	return validID(e.ID) && validID(e.Document) && validDate(e.Due) && extendReasons[e.Reason]
}

func (r *Registry) Extend(workspace string, e Extension, actor string) (ExtensionView, error) {
	if !validID(workspace) || !validID(actor) || !extensionForm(e) {
		return ExtensionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	d := r.find(workspace, e.Document)
	if d == nil {
		return ExtensionView{}, ErrNotFound
	}
	s := r.workspaces[workspace]
	if old := s.extensions[e.ID]; old != nil {
		if old.view.Extension != e || old.view.Author != actor {
			return ExtensionView{}, ErrConflict
		}
		return old.view, nil
	}
	if d.view.Receivable.Obligation != ObligationPayable || e.Due <= d.view.Snapshot.EffectiveDue {
		return ExtensionView{}, ErrRejected
	}
	if len(d.extensions) >= r.limits.Extensions {
		return ExtensionView{}, ErrLimit
	}
	made := &extension{doc: d, view: ExtensionView{Extension: e, State: StatePending, Author: actor, Written: now, Digest: extensionDigest(workspace, e, actor)}}
	s.extensions[e.ID] = made
	d.extensions = append(d.extensions, made)
	return made.view, nil
}

func (r *Registry) ApproveExtension(workspace, id, actor string) (ExtensionView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ExtensionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.extensions[id] == nil {
		return ExtensionView{}, ErrNotFound
	}
	e := s.extensions[id]
	if e.view.State == StateApproved && e.view.Approver == actor {
		return e.view, nil
	}
	d := e.doc
	if e.view.State != StatePending || e.view.Author == actor || e.view.Extension.Due <= d.view.Snapshot.EffectiveDue {
		return ExtensionView{}, ErrRejected
	}
	previous := d.view.Snapshot.EffectiveDue
	late := daysBetween(previous, dateOf(now, d.location))
	if late < 0 {
		late = 0
	}
	e.view.State, e.view.Approver, e.view.ApprovedAt, e.view.PreviousDue, e.view.DaysOverdue = StateApproved, actor, now, previous, late
	d.view.Snapshot.EffectiveDue = e.view.Extension.Due
	return e.view, nil
}

func (r *Registry) WithdrawExtension(workspace, id, actor string) (ExtensionView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ExtensionView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.extensions[id] == nil {
		return ExtensionView{}, ErrNotFound
	}
	e := s.extensions[id]
	if e.view.State == StateWithdrawn && e.view.WithdrawnBy == actor {
		return e.view, nil
	}
	if e.view.State != StatePending {
		return ExtensionView{}, ErrRejected
	}
	e.view.State, e.view.WithdrawnBy = StateWithdrawn, actor
	return e.view, nil
}

func holdForm(h Hold) bool {
	return validID(h.ID) && validID(h.Document) && holdKinds[h.Kind] && validID(h.Reference)
}

func (r *Registry) Hold(workspace string, h Hold, actor string) (HoldView, error) {
	if !validID(workspace) || !validID(actor) || !holdForm(h) {
		return HoldView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	d := r.find(workspace, h.Document)
	if d == nil {
		return HoldView{}, ErrNotFound
	}
	s := r.workspaces[workspace]
	if old := s.holds[h.ID]; old != nil {
		if old.view.Hold != h || old.view.Author != actor {
			return HoldView{}, ErrConflict
		}
		return old.view, nil
	}
	if len(d.holds) >= r.limits.Holds {
		return HoldView{}, ErrLimit
	}
	made := &hold{doc: d, view: HoldView{Hold: h, State: StateApproved, Author: actor, Written: now, Digest: holdDigest(workspace, h, actor)}}
	s.holds[h.ID] = made
	d.holds = append(d.holds, made)
	return made.view, nil
}

func (r *Registry) ReleaseHold(workspace, id, actor string) (HoldView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return HoldView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.holds[id] == nil {
		return HoldView{}, ErrNotFound
	}
	h := s.holds[id]
	if h.view.State == StateReleased {
		if h.view.ReleasedBy == actor {
			return h.view, nil
		}
		return HoldView{}, ErrRejected
	}
	h.view.State, h.view.ReleasedBy, h.view.ReleasedAt = StateReleased, actor, now
	return h.view, nil
}

func today(d *document, now time.Time) string { return dateOf(now, d.location) }
