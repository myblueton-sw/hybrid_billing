package erpledger

type PreparationCheck struct {
	Total    Amounts
	Mappings []string
}

type preparation struct {
	total    amounts
	lines    map[string]amounts
	mappings []string
}

func (r *Registry) freshPreparation(s *space, c *connection, sub Submission, actor Actor) (preparation, error) {
	if err := actor.refusal(); err != nil {
		return preparation{}, err
	}
	if c == nil {
		return preparation{}, refuse(ReasonUnknownConnection)
	}
	scale := c.view.Connection.Scale
	total, lines := zero(), map[string]amounts{}
	for _, l := range sub.Lines {
		net, a := r.units(l.Net, scale)
		tax, b := r.units(l.Tax, scale)
		quantity, d := r.units(l.Quantity, scale)
		if !a || !b || !d {
			return preparation{}, ErrInvalid
		}
		lines[l.ID] = amounts{net, tax, quantity}
		total.add(lines[l.ID])
	}
	if !c.caps[CommandPrepare] {
		return preparation{}, refuse(ReasonUnsupported)
	}
	survey := s.surveys[surveyKey{sub.Connection, sub.Customer, sub.Period}]
	if survey == nil {
		return preparation{}, refuse(ReasonNotSurveyed)
	}
	if survey.Survey.Outcome != SurveyComplete {
		return preparation{}, refuse(ReasonLedgerUnknown)
	}
	used, ok := s.mapped(sub)
	if !ok {
		return preparation{}, refuse(ReasonMappingMissing)
	}
	if sub.Path == PathOrderBased {
		a := s.anchors[sub.Anchor]
		if a == nil || a.Connection != sub.Connection || a.Customer != sub.Customer || a.Kind == AnchorPayable {
			return preparation{}, refuse(ReasonUnknownAnchor)
		}
	}
	for _, l := range sub.Lines {
		for _, a := range s.covering[l.Source] {
			if blocks(sub.Purpose, a.Kind) && !(sub.Path == PathOrderBased && a.ID == sub.Anchor) {
				return preparation{}, refuse(ReasonAlreadyReflected)
			}
		}
	}
	for _, l := range sub.Lines {
		for _, other := range s.bySource[sourceKey(sub.Purpose, l.Source)] {
			if other.live() {
				return preparation{}, refuse(ReasonAlreadySubmitted)
			}
		}
	}
	if len(sub.Lines) == 0 {
		return preparation{}, refuse(ReasonEmpty)
	}
	if len(sub.Lines) > r.limits.Lines || len(s.submissions) >= r.limits.Submissions ||
		!r.bounded(total.net, scale) || !r.bounded(total.tax, scale) || !r.bounded(total.quantity, scale) {
		return preparation{}, ErrLimit
	}
	return preparation{total: total, lines: lines, mappings: used}, nil
}

func (r *Registry) CheckPreparation(workspace string, sub Submission, actor Actor) (PreparationCheck, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !r.submissionForm(sub) || !actor.valid() {
		return PreparationCheck{}, ErrInvalid
	}
	s, c := r.connection(workspace, sub.Connection)
	if s != nil && s.submissions[sub.Key] != nil {
		return PreparationCheck{}, ErrConflict
	}
	checked, err := r.freshPreparation(s, c, sub, actor)
	if err != nil {
		return PreparationCheck{}, err
	}
	return PreparationCheck{Total: checked.total.show(c.view.Connection.Scale), Mappings: append([]string(nil), checked.mappings...)}, nil
}
