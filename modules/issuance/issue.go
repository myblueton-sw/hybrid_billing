package issuance

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

type need struct {
	field   string
	subject string
	reason  string
}

func needs(k Kind, d Draft) []need {
	var out []need
	if payable(k) {
		out = append(out, need{protected.FieldPaymentAccount, SubjectEntity + d.Entity, ReasonAccountUnresolved})
	}
	out = append(out, need{protected.FieldDocumentRecipient, SubjectCustomer + d.Customer, ReasonRecipientUnresolved})
	if k.Statutory {
		out = append(out, need{protected.FieldTaxIdentifier, SubjectEntity + d.Entity, ReasonTaxIDUnresolved})
		out = append(out, need{protected.FieldTaxIdentifier, SubjectCustomer + d.Customer, ReasonTaxIDUnresolved})
	}
	return append(out, need{protected.FieldDocumentTemplate, SubjectKind + k.ID, ReasonTemplateUnresolved})
}

var termReasons = map[string]bool{ReasonTermMissing: true, ReasonTermPending: true, ReasonTermConflict: true}

func (r *Registry) bind(workspace string, k Kind, d Draft, at time.Time) ([]Reference, *receivables.Coverage, *time.Location, error) {
	var refs []Reference
	for _, n := range needs(k, d) {
		found, err := r.config.Fields.Resolve(workspace, n.subject, n.field, at)
		if err != nil || !validID(found.Change) || !validID(found.Ref.ID) || found.Effective.IsZero() || found.Effective.After(at) {
			return nil, nil, nil, refuse(n.reason)
		}
		refs = append(refs, Reference{Field: n.field, Subject: n.subject, Change: found.Change, Ref: found.Ref, Effective: found.Effective.UTC()})
	}
	if !payable(k) {
		location := r.zone(k.Zone)
		if location == nil {
			return nil, nil, nil, refuse(ReasonZoneUnavailable)
		}
		return refs, nil, location, nil
	}
	c, err := r.config.Terms.Cover(workspace, d.Contract, at)
	if err != nil {
		var refusal receivables.Refusal
		if errors.As(err, &refusal) && termReasons[refusal.Reason] {
			return nil, nil, nil, refuse(refusal.Reason)
		}
		return nil, nil, nil, refuse(ReasonTermUnavailable)
	}
	if c.Mode != receivables.ModeCalendar || c.Contract != d.Contract || c.NetDays < 0 || c.NetDays > maxNetDays {
		return nil, nil, nil, refuse(ReasonModeUnsupported)
	}
	location := r.zone(c.Zone)
	if location == nil {
		return nil, nil, nil, refuse(ReasonZoneUnavailable)
	}
	if issue, due := dates(c.NetDays, at, location); c.Date != issue || c.Due != due {
		return nil, nil, nil, refuse(ReasonCoverageMismatch)
	}
	return refs, &c, location, nil
}

func dates(net int, at time.Time, location *time.Location) (string, string) {
	issue := at.In(location).Format(time.DateOnly)
	day, _ := time.Parse(time.DateOnly, issue)
	return issue, day.AddDate(0, 0, net).Format(time.DateOnly)
}

func sameReferences(a, b []Reference) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func sameCoverage(a, b *receivables.Coverage) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Binding == b.Binding && a.Term == b.Term && a.Revision == b.Revision && a.NetDays == b.NetDays && a.Zone == b.Zone
}

func due(k Kind, c *receivables.Coverage, at time.Time, location *time.Location) Due {
	if c == nil {
		return Due{Status: DueNotApplicable, Zone: k.Zone, IssueDate: at.In(location).Format(time.DateOnly)}
	}
	issue, due := dates(c.NetDays, at, location)
	return Due{Status: DueApplicable, Zone: c.Zone, IssueDate: issue, Binding: c.Binding, Term: c.Term, Revision: c.Revision, NetDays: c.NetDays,
		Mode: c.Mode, OriginalDue: due, EffectiveDue: due}
}

func encodeSnapshot(v Snapshot) string {
	e := (&encoder{}).text(snapshotLabel).text(v.Workspace).text(v.Number).flag(v.External)
	encodeDraft(e, v.Draft)
	e.number(int64(v.Revision)).text(v.DraftDigest).text(v.Submitter).text(v.Approver).text(v.Issuer).text(v.IssuerKind).text(v.IssuedAt.Format(time.RFC3339Nano))
	e.text(v.Class).text(v.Region).text(v.Tax).flag(v.Statutory).text(v.TaxEvidence).text(v.TaxReviewer)
	e.number(int64(len(v.References)))
	for _, ref := range v.References {
		e.text(ref.Field).text(ref.Subject).text(ref.Change).text(ref.Ref.ID).text(ref.Ref.Fingerprint).text(ref.Effective.Format(time.RFC3339Nano))
	}
	d := v.Due
	e.text(d.Status).text(d.Zone).text(d.IssueDate).text(d.Binding).text(d.Term).number(int64(d.Revision)).number(int64(d.NetDays)).text(d.Mode).text(d.OriginalDue).text(d.EffectiveDue)
	return e.digest()
}

func copySnapshot(v Snapshot) Snapshot {
	v.Draft = copyDraft(v.Draft)
	v.References = append([]Reference(nil), v.References...)
	return v
}

func viewDocument(d *document) DocumentView {
	v := d.view
	v.Snapshot = copySnapshot(v.Snapshot)
	v.Attempts = append([]Attempt(nil), v.Attempts...)
	v.Downloads = append([]Download(nil), v.Downloads...)
	return v
}

func (r *Registry) prepare(workspace, id string, actor Actor) (*space, *draft, *kind, error) {
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return nil, nil, nil, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return nil, nil, nil, err
	}
	s, d, err := r.lookupDraft(workspace, id)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, d, s.kinds[d.current.Draft.Kind], nil
}

type admission struct {
	refs     []Reference
	coverage *receivables.Coverage
	location *time.Location
}

func (r *Registry) admit(workspace string, s *space, d *draft, k *kind, at time.Time) (admission, error) {
	v := d.current
	switch {
	case v.Approver == "":
		return admission{}, refuse(ReasonNotApproved)
	case k.view.Kind.Statutory && k.view.TaxReviewer == "":
		return admission{}, refuse(ReasonTaxUnverified)
	}
	if err := r.rules(workspace, s, k, v.Draft); err != nil {
		return admission{}, err
	}
	refs, coverage, location, err := r.bind(workspace, k.view.Kind, v.Draft, at)
	if err != nil {
		return admission{}, err
	}
	if !sameReferences(v.References, refs) || !sameCoverage(v.Coverage, coverage) {
		return admission{}, refuse(ReasonReferenceChanged)
	}
	return admission{refs: refs, coverage: coverage, location: location}, nil
}

func (r *Registry) record(workspace string, s *space, d *draft, k *kind, number string, a admission, external bool, actor Actor, at time.Time) DocumentView {
	v := d.current
	snap := Snapshot{Workspace: workspace, Number: number, External: external, Draft: copyDraft(v.Draft), Revision: v.Revision, DraftDigest: v.Digest,
		Submitter: v.Submitter, Approver: v.Approver, Issuer: actor.ID, IssuerKind: actor.Kind, IssuedAt: at, Class: k.view.Kind.Class, Region: k.view.Kind.Region,
		Tax: k.view.Kind.Tax, Statutory: k.view.Kind.Statutory, TaxEvidence: k.view.TaxEvidence, TaxReviewer: k.view.TaxReviewer, References: a.refs,
		Due: due(k.view.Kind, a.coverage, at, a.location)}
	snap.Digest = encodeSnapshot(snap)
	doc := &document{view: DocumentView{Snapshot: snap, File: FileNone}}
	d.document = doc
	d.current.Number = number
	s.documents[v.Draft.ID] = doc
	if k.view.Kind.Credit {
		original := s.documents[v.Draft.Original]
		original.credits = append(original.credits, v.Draft.ID)
	}
	return viewDocument(doc)
}

func (r *Registry) Issue(workspace, id string, actor Actor) (DocumentView, error) {
	now := r.enter()
	defer r.leave()
	s, d, k, err := r.prepare(workspace, id, actor)
	if err != nil {
		return DocumentView{}, err
	}
	if d.document != nil {
		if snap := d.document.view.Snapshot; snap.Issuer == actor.ID && snap.IssuerKind == actor.Kind && !snap.External {
			return viewDocument(d.document), nil
		}
		return DocumentView{}, refuse(ReasonAlreadyIssued)
	}
	v, err := r.seriesFor(s, k.view.Kind.ID)
	if err != nil {
		return DocumentView{}, err
	}
	a, err := r.admit(workspace, s, d, k, now)
	if err != nil {
		return DocumentView{}, err
	}
	if len(s.documents) >= r.limits.Documents {
		return DocumentView{}, ErrLimit
	}
	n := d.reserved
	if n == nil {
		if n, err = r.take(s, v, id, actor, now); err != nil {
			return DocumentView{}, err
		}
		d.reserved = n
	}
	n.State, n.Actor, n.At = NumberIssued, actor.ID, now
	return r.record(workspace, s, d, k, n.Number, a, false, actor, now), nil
}

func (r *Registry) RecordExternal(workspace, id string, x External, actor Actor) (DocumentView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(x.Number) || x.IssuedAt.IsZero() {
		return DocumentView{}, ErrInvalid
	}
	s, d, k, err := r.prepare(workspace, id, actor)
	if err != nil {
		return DocumentView{}, err
	}
	at := x.IssuedAt.UTC().Round(0)
	if d.document != nil {
		snap := d.document.view.Snapshot
		if snap.Issuer == actor.ID && snap.IssuerKind == actor.Kind && snap.External && snap.Number == x.Number && snap.IssuedAt.Equal(at) {
			return viewDocument(d.document), nil
		}
		return DocumentView{}, refuse(ReasonAlreadyIssued)
	}
	switch {
	case k.view.Kind.Class != ClassERP:
		return DocumentView{}, refuse(ReasonNotERP)
	case at.After(now):
		return DocumentView{}, refuse(ReasonFutureInstant)
	case d.current.Approver != "" && at.Before(d.current.At):
		return DocumentView{}, refuse(ReasonBeforeApproval)
	}
	a, err := r.admit(workspace, s, d, k, at)
	if err != nil {
		return DocumentView{}, err
	}
	key := k.view.Kind.ID + "\x00" + x.Number
	if s.external[key] {
		return DocumentView{}, refuse(ReasonExternalInUse)
	}
	if len(s.documents) >= r.limits.Documents {
		return DocumentView{}, ErrLimit
	}
	s.external[key] = true
	return r.record(workspace, s, d, k, x.Number, a, true, actor, at), nil
}

func (r *Registry) DocumentOf(workspace, id string) (DocumentView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return DocumentView{}, ErrInvalid
	}
	s := r.space(workspace)
	if s == nil || s.documents[id] == nil {
		return DocumentView{}, ErrNotFound
	}
	return viewDocument(s.documents[id]), nil
}

func (r *Registry) CreditsOf(workspace, id string) ([]string, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return nil, ErrInvalid
	}
	s := r.space(workspace)
	if s == nil || s.documents[id] == nil {
		return nil, ErrNotFound
	}
	return append([]string{}, s.documents[id].credits...), nil
}
