package corrections

import "sort"

func (r *Registry) Explain(workspace, id, credit string, e Explanation, actor Actor) (Note, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(credit) || !reasons[e.Reason] || !validID(e.Evidence) || !actor.valid() {
		return Note{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return Note{}, err
	}
	s := r.workspaces[workspace]
	o, err := r.originalOf(workspace, s, id)
	if err != nil {
		return Note{}, err
	}
	if found := o.notes[credit]; found != nil {
		if actor.is(found.Actor) && found.Explanation == e {
			return *found, nil
		}
		return Note{}, ErrConflict
	}
	if s != nil && s.documents[credit] {
		return Note{}, refuse(RefusalLinked)
	}
	l, err := r.live(workspace, o)
	if err != nil {
		return Note{}, err
	}
	if !contains(l.unlinked, credit) {
		return Note{}, refuse(RefusalNotUnlinked)
	}
	d := l.snapshots[credit]
	if d.Issuer == actor.ID && d.IssuerKind == KindHuman {
		return Note{}, refuse(RefusalSelfExplain)
	}
	if s, err = r.open(workspace); err != nil {
		return Note{}, err
	}
	if s.records >= r.limits.Records {
		return Note{}, ErrLimit
	}
	s.records++
	s.originals[o.id] = o
	note := &Note{Credit: credit, Explanation: e, Approver: d.Approver, Issuer: d.Issuer, Actor: actor.ID, At: now}
	o.notes[credit] = note
	o.linked[credit] = true
	s.documents[credit] = true
	return *note, nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (r *Registry) RecordFinal(workspace, id string, f Final, actor Actor) (Record, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(f.Reference) || !canonical(f.Amount) || f.Observed.IsZero() || !actor.valid() {
		return Record{}, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return Record{}, err
	}
	s := r.workspaces[workspace]
	o, err := r.originalOf(workspace, s, id)
	if err != nil {
		return Record{}, err
	}
	f.Observed = f.Observed.UTC().Round(0)
	l, err := r.live(workspace, o)
	if err != nil {
		return Record{}, err
	}
	for _, existing := range o.finals {
		if existing.Final.Reference == f.Reference {
			if existing.Final == f && existing.Actor == actor.ID && existing.Kind == actor.Kind {
				return judge(o, l, existing), nil
			}
			return Record{}, ErrConflict
		}
	}
	if _, ok := scaled(f.Amount, o.snap.Draft.Scale); !ok {
		return Record{}, refuse(RefusalScale)
	}
	if f.Observed.After(now) {
		return Record{}, refuse(RefusalFuture)
	}
	if s, err = r.open(workspace); err != nil {
		return Record{}, err
	}
	if s.records >= r.limits.Records {
		return Record{}, ErrLimit
	}
	record := Record{Final: f, Actor: actor.ID, Kind: actor.Kind, At: now}
	s.records++
	s.originals[o.id] = o
	o.finals = append(o.finals, record)
	return judge(o, l, record), nil
}

func judge(o *original, l live, record Record) Record {
	amount, _ := scaled(record.Final.Amount, o.snap.Draft.Scale)
	record.Expected = format(l.effective, o.snap.Draft.Scale)
	record.Outcome = OutcomeDiverged
	if amount.Cmp(l.effective) == 0 {
		record.Outcome = OutcomeMatched
	}
	return record
}

func (r *Registry) EffectiveOf(workspace, id string) (Effective, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return Effective{}, ErrInvalid
	}
	o, err := r.originalOf(workspace, r.workspaces[workspace], id)
	if err != nil {
		return Effective{}, err
	}
	l, err := r.live(workspace, o)
	if err != nil {
		return Effective{}, err
	}
	out := Effective{Original: id, Total: format(o.total, o.snap.Draft.Scale), Effective: format(l.effective, o.snap.Draft.Scale), Unlinked: l.unlinked}
	for _, c := range o.applied {
		out.Applied = append(out.Applied, c.view)
	}
	for _, note := range o.notes {
		out.Notes = append(out.Notes, *note)
	}
	sort.Slice(out.Notes, func(a, b int) bool { return out.Notes[a].Credit < out.Notes[b].Credit })
	if o.open != nil {
		v := o.open.view
		out.Open = &v
	}
	for _, record := range o.finals {
		out.Finals = append(out.Finals, judge(o, l, record))
	}
	return out, nil
}
