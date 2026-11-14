package erpledger

import (
	"math/big"
	"sort"
	"strconv"
	"time"
)

type amounts struct {
	net, tax, quantity *big.Int
}

func zero() amounts { return amounts{new(big.Int), new(big.Int), new(big.Int)} }

func (a amounts) add(b amounts) {
	a.net.Add(a.net, b.net)
	a.tax.Add(a.tax, b.tax)
	a.quantity.Add(a.quantity, b.quantity)
}

func (a amounts) equal(b amounts) bool {
	return a.net.Cmp(b.net) == 0 && a.tax.Cmp(b.tax) == 0 && a.quantity.Cmp(b.quantity) == 0
}

func (a amounts) show(scale int) Amounts {
	return Amounts{Net: show(a.net, scale), Tax: show(a.tax, scale), Quantity: show(a.quantity, scale)}
}

type submission struct {
	view  SubmissionView
	lines map[string]amounts
	conn  *connection
}

func (s *submission) live() bool {
	return s.view.Diverged || !retired[s.view.State]
}

func (s *submission) watched(now time.Time) bool {
	switch {
	case s.view.Watch == nil:
		return false
	case s.view.Diverged, s.view.State == StateSent, s.view.State == StateDraft, s.view.State == StateUnknown:
		return true
	}
	return s.view.State == StatePosted && now.Before(s.view.Posted.Add(postedWatch))
}

func (s *submission) copy() SubmissionView {
	v := s.view
	v.Submission.Lines = append([]Line(nil), v.Submission.Lines...)
	v.Mappings = append([]string(nil), v.Mappings...)
	v.Commands = append([]CommandView(nil), v.Commands...)
	if v.Watch != nil {
		w := *v.Watch
		v.Watch = &w
	}
	if v.Resolution != nil {
		res := *v.Resolution
		v.Resolution = &res
	}
	return v
}

func (r *Registry) submissionForm(sub Submission) bool {
	if !validID(sub.Key) || !validID(sub.Connection) || sub.Purpose != PurposeInvoice && sub.Purpose != PurposePayable ||
		!validID(sub.Customer) || !validMonth(sub.Period) {
		return false
	}
	switch {
	case sub.Path == PathDirect && sub.Anchor == "":
	case sub.Path == PathOrderBased && validID(sub.Anchor) && sub.Purpose == PurposeInvoice:
	default:
		return false
	}
	ids, sources := map[string]bool{}, map[string]bool{}
	for _, l := range sub.Lines {
		if !validID(l.ID) || ids[l.ID] || !validID(l.Source) || sources[l.Source] || !validID(l.Product) || !validID(l.TaxCode) ||
			!r.number(l.Quantity) || !r.number(l.Net) || !r.number(l.Tax) {
			return false
		}
		ids[l.ID], sources[l.Source] = true, true
	}
	return true
}

func sameSubmission(a, b Submission) bool {
	if a.Key != b.Key || a.Connection != b.Connection || a.Purpose != b.Purpose || a.Customer != b.Customer || a.Period != b.Period ||
		a.Path != b.Path || a.Anchor != b.Anchor || len(a.Lines) != len(b.Lines) {
		return false
	}
	for index := range a.Lines {
		if a.Lines[index] != b.Lines[index] {
			return false
		}
	}
	return true
}

func blocks(purpose, kind string) bool {
	if purpose == PurposePayable {
		return kind == AnchorPayable
	}
	return kind != AnchorPayable
}

func sourceKey(purpose, source string) string { return purpose + "|" + source }

func (r *Registry) Prepare(workspace string, sub Submission, actor Actor) (SubmissionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !r.submissionForm(sub) || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	s, c := r.connection(workspace, sub.Connection)
	if s != nil {
		if seen := s.submissions[sub.Key]; seen != nil {
			if sameSubmission(seen.view.Submission, sub) && actor.is(seen.view.Author) {
				return seen.copy(), nil
			}
			return SubmissionView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return SubmissionView{}, err
	}
	if c == nil {
		return SubmissionView{}, refuse(ReasonUnknownConnection)
	}
	scale := c.view.Connection.Scale
	total, lines := zero(), map[string]amounts{}
	for _, l := range sub.Lines {
		net, a := r.units(l.Net, scale)
		tax, b := r.units(l.Tax, scale)
		quantity, d := r.units(l.Quantity, scale)
		if !a || !b || !d {
			return SubmissionView{}, ErrInvalid
		}
		lines[l.ID] = amounts{net, tax, quantity}
		total.add(lines[l.ID])
	}
	if !c.caps[CommandPrepare] {
		return SubmissionView{}, refuse(ReasonUnsupported)
	}
	survey := s.surveys[surveyKey{sub.Connection, sub.Customer, sub.Period}]
	if survey == nil {
		return SubmissionView{}, refuse(ReasonNotSurveyed)
	}
	if survey.Survey.Outcome != SurveyComplete {
		return SubmissionView{}, refuse(ReasonLedgerUnknown)
	}
	used, ok := s.mapped(sub)
	if !ok {
		return SubmissionView{}, refuse(ReasonMappingMissing)
	}
	if sub.Path == PathOrderBased {
		a := s.anchors[sub.Anchor]
		if a == nil || a.Connection != sub.Connection || a.Customer != sub.Customer || a.Kind == AnchorPayable {
			return SubmissionView{}, refuse(ReasonUnknownAnchor)
		}
	}
	for _, l := range sub.Lines {
		for _, a := range s.covering[l.Source] {
			if blocks(sub.Purpose, a.Kind) && !(sub.Path == PathOrderBased && a.ID == sub.Anchor) {
				return SubmissionView{}, refuse(ReasonAlreadyReflected)
			}
		}
	}
	for _, l := range sub.Lines {
		for _, other := range s.bySource[sourceKey(sub.Purpose, l.Source)] {
			if other.live() {
				return SubmissionView{}, refuse(ReasonAlreadySubmitted)
			}
		}
	}
	if len(sub.Lines) == 0 {
		return SubmissionView{}, refuse(ReasonEmpty)
	}
	if len(sub.Lines) > r.limits.Lines || len(s.submissions) >= r.limits.Submissions ||
		!r.bounded(total.net, scale) || !r.bounded(total.tax, scale) || !r.bounded(total.quantity, scale) {
		return SubmissionView{}, ErrLimit
	}
	sub.Lines = append([]Line(nil), sub.Lines...)
	made := &submission{lines: lines, conn: c}
	made.view = SubmissionView{Submission: sub, State: StatePrepared, Total: total.show(scale), Mappings: used, Author: actor.ID, Prepared: now}
	made.view.Digest = submissionDigest(workspace, made.view)
	s.submissions[sub.Key] = made
	for _, l := range sub.Lines {
		k := sourceKey(sub.Purpose, l.Source)
		s.bySource[k] = append(s.bySource[k], made)
	}
	return made.copy(), nil
}

func (s *space) mapped(sub Submission) ([]string, bool) {
	needed := map[mappingKey]bool{{sub.Connection, "customer", sub.Customer}: true}
	for _, l := range sub.Lines {
		needed[mappingKey{sub.Connection, "product", l.Product}] = true
		needed[mappingKey{sub.Connection, "tax", l.TaxCode}] = true
	}
	var used []string
	for k := range needed {
		m := s.mappings[k]
		if m == nil {
			return nil, false
		}
		used = append(used, k.kind+":"+k.platform+":"+m.Mapping.External+":"+strconv.Itoa(m.Revision))
	}
	sort.Strings(used)
	return used, true
}

func submissionDigest(workspace string, v SubmissionView) string {
	sub := v.Submission
	texts := []string{submissionLabel, workspace, sub.Key, sub.Connection, sub.Purpose, sub.Customer, sub.Period, sub.Path, sub.Anchor, strconv.Itoa(len(sub.Lines))}
	for _, l := range sub.Lines {
		texts = append(texts, l.ID, l.Source, l.Product, l.TaxCode, l.Quantity, l.Net, l.Tax)
	}
	texts = append(texts, v.Mappings...)
	texts = append(texts, v.Total.Net, v.Total.Tax, v.Total.Quantity, v.Author, v.Prepared.Format(time.RFC3339Nano))
	return hashOf(texts...)
}

func (r *Registry) find(workspace, key string) (*space, *submission, error) {
	s := r.workspaces[workspace]
	if s == nil || s.submissions[key] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.submissions[key], nil
}

func (r *Registry) Command(workspace, key, name string, actor Actor) (SubmissionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) || !capabilities[name] || name == CommandConnect || name == CommandMapping || name == CommandPrepare || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	_, sub, err := r.find(workspace, key)
	if err != nil {
		return SubmissionView{}, err
	}
	if err := actor.refusal(); err != nil {
		return SubmissionView{}, err
	}
	if !sub.conn.caps[name] {
		return SubmissionView{}, refuse(ReasonUnsupported)
	}
	state := sub.view.State
	allowed := false
	switch name {
	case CommandSubmit:
		allowed = state == StatePrepared
	case CommandPost:
		allowed = state == StateDraft
	case CommandCorrect:
		allowed = state == StatePosted
	default:
		allowed = state != StatePrepared && state != StateWithdrawn
	}
	if !allowed {
		return SubmissionView{}, refuse(ReasonWrongState)
	}
	if sub.view.Diverged && (name == CommandSubmit || name == CommandPost || name == CommandCorrect) {
		return SubmissionView{}, refuse(ReasonDiverged)
	}
	if name == CommandSubmit && sub.conn.view.Connection.Transport == TransportFile && !sub.conn.view.Verified {
		return SubmissionView{}, refuse(ReasonTemplateUnverified)
	}
	if len(sub.view.Commands) >= maxCommands {
		return SubmissionView{}, ErrLimit
	}
	sub.view.Commands = append(sub.view.Commands, CommandView{Name: name, Actor: actor.ID, At: now})
	if name == CommandSubmit {
		sub.view.State = StateSent
		sub.view.Watch = &Watch{Next: now.Add(sub.conn.view.Connection.Interval), Owner: sub.conn.view.Connection.Owner}
	}
	return sub.copy(), nil
}

func (r *Registry) SubmissionOf(workspace, key string) (SubmissionView, error) {
	r.enter()
	defer r.leave()
	_, sub, err := r.find(workspace, key)
	if err != nil {
		return SubmissionView{}, err
	}
	return sub.copy(), nil
}

func (r *Registry) Due(workspace string, at time.Time) ([]string, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || at.IsZero() {
		return nil, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	var out []string
	for key, sub := range s.submissions {
		if sub.watched(now) && !sub.view.Watch.Next.After(at) {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (r *Registry) Checked(workspace, key, outcome string, actor Actor) (SubmissionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) || !checkOutcomes[outcome] || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	_, sub, err := r.find(workspace, key)
	if err != nil {
		return SubmissionView{}, err
	}
	if actor.Kind == KindAssistant {
		return SubmissionView{}, refuse(ReasonAINotAllowed)
	}
	if !sub.watched(now) {
		return SubmissionView{}, refuse(ReasonWrongState)
	}
	w := sub.view.Watch
	w.Last, w.Next, w.Retries = now, now.Add(sub.conn.view.Connection.Interval), w.Retries+1
	if sub.view.State == StateSent {
		sub.view.State = StateUnknown
	}
	return sub.copy(), nil
}

func (r *Registry) Withdraw(workspace, key string, actor Actor) (SubmissionView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(key) || !actor.valid() {
		return SubmissionView{}, ErrInvalid
	}
	_, sub, err := r.find(workspace, key)
	if err != nil {
		return SubmissionView{}, err
	}
	if sub.view.State == StateWithdrawn && actor.is(sub.view.WithdrawnBy) {
		return sub.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return SubmissionView{}, err
	}
	if sub.view.State != StatePrepared {
		return SubmissionView{}, refuse(ReasonWrongState)
	}
	sub.view.State, sub.view.WithdrawnBy, sub.view.Diverged = StateWithdrawn, actor.ID, false
	return sub.copy(), nil
}
