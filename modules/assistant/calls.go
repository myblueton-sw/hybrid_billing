package assistant

import (
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func inputForm(input []commands.Value) bool {
	seen := map[string]bool{}
	for _, v := range input {
		if !validID(v.Name) || seen[v.Name] || !validText(v.Value) {
			return false
		}
		seen[v.Name] = true
	}
	return true
}

func decimalInput(e commands.Entry, input []commands.Value) bool {
	types := map[string]commands.FieldType{}
	for _, f := range e.Fields {
		types[f.Name] = f.Type
	}
	for _, v := range input {
		if types[v.Name] == commands.Decimal {
			return true
		}
	}
	return false
}

func (r *Registry) working(workspace, jobID, conversationID string, now time.Time) (*space, *job, error) {
	s := r.workspaces[workspace]
	if s == nil || s.jobs[jobID] == nil {
		return nil, nil, ErrNotFound
	}
	j := s.jobs[jobID]
	c := s.conversations[conversationID]
	switch {
	case c == nil || !c.jobs[jobID]:
		return nil, nil, refuse(ReasonNotAttached)
	case !now.Before(c.view.Expires):
		return nil, nil, refuse(ReasonConversation)
	case j.view.Closed:
		return nil, nil, refuse(ReasonJobClosed)
	case !now.Before(j.view.Created.Add(r.limits.Duration)):
		return nil, nil, refuse(ReasonBudget)
	}
	return s, j, nil
}

func (r *Registry) exposed(s *space, command string) (*tool, commands.Entry, error) {
	t := s.tools[command]
	if t == nil || t.view.State != StateActive {
		return nil, commands.Entry{}, refuse(ReasonNotExposed)
	}
	e, ok := r.entry(command)
	if !ok {
		return nil, commands.Entry{}, refuse(ReasonNotExposed)
	}
	if err := r.admissible(e, t.view.Tool.Stage); err != nil {
		return nil, commands.Entry{}, err
	}
	return t, e, nil
}

func (j *job) log(e Entry, now time.Time) {
	e.Sequence, e.At = len(j.view.Ledger)+1, now
	j.view.Ledger = append(j.view.Ledger, e)
}

func outcome(result commands.Result) string {
	if result.Err != nil {
		return string(result.Err.Code)
	}
	return string(result.Status)
}

func (r *Registry) bound(t *tool, fields map[string]string) (map[string]string, []string) {
	out := map[string]string{}
	var truncated []string
	for _, name := range t.view.Tool.Results {
		value, ok := fields[name]
		if !ok {
			continue
		}
		if len(out) >= r.limits.Fields {
			break
		}
		if len(value) > r.limits.Bytes {
			cut := r.limits.Bytes
			for cut > 0 && !utf8.RuneStart(value[cut]) {
				cut--
			}
			value = value[:cut]
			truncated = append(truncated, name)
		}
		out[name] = value
	}
	return out, truncated
}

func (r *Registry) Invoke(workspace, jobID string, call Invocation, actor Actor) (Reply, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(jobID) || !validID(call.Conversation) || !validID(call.Tool) || !validID(call.Target) ||
		call.Expected != "" && !validID(call.Expected) || !inputForm(call.Input) || !actor.valid() {
		return Reply{}, ErrInvalid
	}
	if err := actor.assistant(); err != nil {
		return Reply{}, err
	}
	s, j, err := r.working(workspace, jobID, call.Conversation, now)
	if err != nil {
		return Reply{}, err
	}
	if !ownerAnswer(call.Answer, workspace, j.view.Owner) {
		return Reply{}, refuse(ReasonNotOwner)
	}
	t, e, err := r.exposed(s, call.Tool)
	if err != nil {
		return Reply{}, err
	}
	switch {
	case t.view.Tool.Stage == StageRequest:
		return Reply{}, refuse(ReasonRequestOnly)
	case decimalInput(e, call.Input):
		return Reply{}, refuse(ReasonModelAmount)
	case j.view.Calls >= r.limits.Calls || !j.targets[call.Target] && len(j.targets) >= r.limits.Targets:
		return Reply{}, refuse(ReasonBudget)
	}
	j.view.Calls++
	j.targets[call.Target] = true
	j.view.Targets = len(j.targets)
	request := "assistant-" + strconv.FormatUint(j.serial, 10) + "-" + strconv.Itoa(j.view.Calls)
	c := commands.Call{Answer: call.Answer, Client: r.config.Client, Request: request, Command: call.Tool, Target: call.Target, Fields: append([]commands.Value(nil), call.Input...)}
	if e.Mutating {
		c.Expected = call.Expected
		c.Key = key(callTag, append([]string{workspace, jobID, strconv.FormatUint(j.serial, 10), call.Tool, call.Target, call.Expected}, inputParts(call.Input)...)...)
	}
	result := r.config.Commands.Execute(c)
	reply := Reply{Status: outcome(result), Operation: result.Operation, Replay: result.Replay, Calls: j.view.Calls, Remaining: r.limits.Calls - j.view.Calls, Targets: j.view.Targets}
	if result.Err != nil {
		reply.Code = string(result.Err.Code)
		reply.Operation = result.Err.Operation
	} else {
		reply.Fields, reply.Truncated = r.bound(t, result.Fields)
	}
	j.log(Entry{Kind: EntryCall, Tool: call.Tool, Target: call.Target, Operation: reply.Operation, Outcome: reply.Status, Replay: result.Replay}, now)
	return reply, nil
}

func cardDigest(workspace, jobID string, serial uint64, c Card, revision int) string {
	return key(cardTag, append([]string{workspace, c.ID, strconv.FormatUint(serial, 10), jobID, c.Tool, strconv.Itoa(revision), c.Target, c.Expected}, inputParts(c.Input)...)...)
}

func copyCard(c Card) Card {
	c.Input = append([]commands.Value(nil), c.Input...)
	return c
}

func viewCard(c *card) CardView {
	v := c.view
	v.Card = copyCard(v.Card)
	return v
}

func (r *Registry) Prepare(workspace, jobID, conversationID string, c Card, actor Actor) (CardView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(jobID) || !validID(conversationID) || !validID(c.ID) || !validID(c.Tool) || !validID(c.Target) ||
		c.Expected != "" && !validID(c.Expected) || !inputForm(c.Input) || !validText(c.Note) || !actor.valid() {
		return CardView{}, ErrInvalid
	}
	if err := actor.assistant(); err != nil {
		return CardView{}, err
	}
	s, j, err := r.working(workspace, jobID, conversationID, now)
	if err != nil {
		return CardView{}, err
	}
	if found := s.cards[c.ID]; found != nil {
		r.settle(found, now)
		if found.job == j && sameCard(found.view.Card, c) {
			return viewCard(found), nil
		}
		return CardView{}, ErrConflict
	}
	t, e, err := r.exposed(s, c.Tool)
	if err != nil {
		return CardView{}, err
	}
	switch {
	case t.view.Tool.Stage != StageRequest:
		return CardView{}, refuse(ReasonNotRequest)
	case decimalInput(e, c.Input):
		return CardView{}, refuse(ReasonModelAmount)
	case len(j.cards) >= r.limits.Cards:
		return CardView{}, refuse(ReasonBudget)
	}
	c = copyCard(c)
	r.serial++
	made := &card{serial: r.serial, view: CardView{Card: c, Job: jobID, Digest: cardDigest(workspace, jobID, r.serial, c, t.view.Revision), State: CardPending}, job: j, revision: t.view.Revision}
	s.cards[c.ID] = made
	j.cards[c.ID] = made
	j.view.Cards = len(j.cards)
	j.log(Entry{Kind: EntryCard, Tool: c.Tool, Target: c.Target, Card: c.ID, Outcome: CardPending}, now)
	return viewCard(made), nil
}

func sameCard(a, b Card) bool {
	if a.ID != b.ID || a.Tool != b.Tool || a.Target != b.Target || a.Expected != b.Expected || a.Note != b.Note || len(a.Input) != len(b.Input) {
		return false
	}
	left, right := inputParts(a.Input), inputParts(b.Input)
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (r *Registry) settle(c *card, now time.Time) {
	if c.view.State == CardPending && r.closed(c.job, now) {
		c.view.State = CardExpired
	}
}

func (r *Registry) CardOf(workspace, id string) (CardView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return CardView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil || s.cards[id] == nil {
		return CardView{}, ErrNotFound
	}
	r.settle(s.cards[id], now)
	return viewCard(s.cards[id]), nil
}

func (r *Registry) Confirm(workspace, id string, conf Confirmation, actor Actor) (CardView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || len(conf.Digest) != 64 || !validText(conf.Rationale) || !actor.valid() {
		return CardView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CardView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.cards[id] == nil {
		return CardView{}, ErrNotFound
	}
	c := s.cards[id]
	r.settle(c, now)
	if !ownerAnswer(conf.Answer, workspace, actor.ID) {
		return CardView{}, refuse(ReasonNotConfirmer)
	}
	if conf.Digest != c.view.Digest {
		return CardView{}, refuse(ReasonDigestChanged)
	}
	switch c.view.State {
	case CardConfirmed:
		if c.view.Confirmer == actor.ID {
			return viewCard(c), nil
		}
		return CardView{}, refuse(ReasonAlreadyDecided)
	case CardUncertain:
		if c.view.Confirmer != actor.ID {
			return CardView{}, refuse(ReasonOutcomeUncertain)
		}
	case CardDeclined:
		return CardView{}, refuse(ReasonAlreadyDecided)
	case CardExpired:
		return CardView{}, refuse(ReasonCardExpired)
	}
	if c.view.Attempts >= maxAttempts {
		return CardView{}, refuse(ReasonBudget)
	}
	t, e, err := r.exposed(s, c.view.Card.Tool)
	if err != nil {
		return CardView{}, err
	}
	if t.view.Revision != c.revision || t.view.Tool.Stage != StageRequest {
		return CardView{}, refuse(ReasonExposureChanged)
	}
	call := commands.Call{Answer: conf.Answer, Request: "card-" + strconv.FormatUint(c.serial, 10) + "-" + strconv.Itoa(c.view.Attempts+1), Command: c.view.Card.Tool, Target: c.view.Card.Target,
		Fields: append([]commands.Value(nil), c.view.Card.Input...)}
	if e.Rationale {
		call.Rationale = conf.Rationale
	}
	if e.Mutating {
		call.Expected = c.view.Card.Expected
		call.Key = key(confirmTag, workspace, strconv.FormatUint(c.serial, 10), c.view.Digest, actor.ID)
	}
	result := r.config.Commands.Execute(call)
	c.view.Attempts++
	c.view.Code = outcome(result)
	switch {
	case result.Err == nil:
		c.view.State, c.view.Confirmer, c.view.Operation = CardConfirmed, actor.ID, result.Operation
	case result.Err.Operation != "":
		c.view.State, c.view.Confirmer, c.view.Operation = CardUncertain, actor.ID, result.Err.Operation
	}
	c.job.log(Entry{Kind: EntryConfirm, Tool: c.view.Card.Tool, Target: c.view.Card.Target, Card: id, Operation: c.view.Operation, Outcome: c.view.Code, Replay: result.Replay}, now)
	return viewCard(c), nil
}

func (r *Registry) Decline(workspace, id, reason string, actor Actor) (CardView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(reason) || !actor.valid() {
		return CardView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CardView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.cards[id] == nil {
		return CardView{}, ErrNotFound
	}
	c := s.cards[id]
	r.settle(c, now)
	switch c.view.State {
	case CardDeclined:
		if actor.is(c.view.Decliner) {
			return viewCard(c), nil
		}
		return CardView{}, refuse(ReasonAlreadyDecided)
	case CardExpired:
		return CardView{}, refuse(ReasonCardExpired)
	case CardConfirmed:
		return CardView{}, refuse(ReasonAlreadyDecided)
	}
	c.view.State, c.view.Decliner, c.view.Reason = CardDeclined, actor.ID, reason
	c.job.log(Entry{Kind: EntryDecline, Tool: c.view.Card.Tool, Target: c.view.Card.Target, Card: id, Outcome: OutcomeDeclined}, now)
	return viewCard(c), nil
}
