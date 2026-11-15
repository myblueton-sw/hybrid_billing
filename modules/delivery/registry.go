package delivery

import (
	"sync"
	"time"
)

type channel struct {
	view ChannelView
}

type delivery struct {
	view    View
	key     [2]string
	running int
}

type ticket struct {
	delivery *delivery
	index    int
}

type space struct {
	channels   map[string]*channel
	deliveries map[string]*delivery
	byKey      map[[2]string]*delivery
	uncertain  map[[2]string]bool
	tokens     map[string]ticket
	queued     int
}

type Registry struct {
	mu         sync.Mutex
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Packages == nil || config.Random == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	if l.Lease < minLease || l.Lease > maxLease {
		return false
	}
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Channels, maxChannels}, {l.Deliveries, maxDeliveries}, {l.Queue, maxQueue}, {l.Attempts, maxAttempts}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return true
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func refuse(reason string) error { return Refusal{Reason: reason} }

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (r *Registry) staff(a Actor) error {
	switch {
	case a.Kind == KindAssistant:
		return refuse(ReasonAINotAllowed)
	case a.Kind == KindSystem:
		return refuse(ReasonNotHuman)
	case r.config.Packages.Recipient(a.ID):
		return refuse(ReasonRecipientActor)
	}
	return nil
}

func (r *Registry) worker(a Actor) error {
	switch {
	case a.Kind == KindAssistant:
		return refuse(ReasonAINotAllowed)
	case r.config.Packages.Recipient(a.ID):
		return refuse(ReasonRecipientActor)
	}
	return nil
}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{channels: map[string]*channel{}, deliveries: map[string]*delivery{}, byKey: map[[2]string]*delivery{}, uncertain: map[[2]string]bool{}, tokens: map[string]ticket{}}
	r.workspaces[workspace] = s
	return s, nil
}

var kinds = map[string]bool{ChannelERP: true, ChannelMail: true, ChannelPortal: true}

func viewChannel(c *channel) ChannelView {
	v := c.view
	v.Diagnoses = append([]DiagnosisView(nil), v.Diagnoses...)
	return v
}

func (r *Registry) Register(workspace string, c Channel, actor Actor) (ChannelView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !validID(c.ID) || !kinds[c.Kind] || !validID(c.Recipient) || !validID(c.Target) {
		return ChannelView{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return ChannelView{}, err
	}
	if s := r.workspaces[workspace]; s != nil && s.channels[c.ID] != nil {
		found := s.channels[c.ID]
		if actor.is(found.view.Registrar) && found.view.Channel == c {
			return viewChannel(found), nil
		}
		return ChannelView{}, ErrConflict
	}
	s, err := r.open(workspace)
	if err != nil {
		return ChannelView{}, err
	}
	if len(s.channels) >= r.limits.Channels {
		return ChannelView{}, ErrLimit
	}
	made := &channel{view: ChannelView{Channel: c, Registrar: actor.ID, Registered: now}}
	s.channels[c.ID] = made
	return viewChannel(made), nil
}

func (r *Registry) channel(workspace, id string) (*space, *channel, error) {
	s := r.workspaces[workspace]
	if s == nil || s.channels[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.channels[id], nil
}

func (r *Registry) Diagnose(workspace, id string, d Diagnosis, actor Actor) (DiagnosisView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() || !validID(d.Evidence.Kind) || !validID(d.Evidence.Reference) {
		return DiagnosisView{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return DiagnosisView{}, err
	}
	_, c, err := r.channel(workspace, id)
	if err != nil {
		return DiagnosisView{}, err
	}
	if last := len(c.view.Diagnoses) - 1; last >= 0 && c.view.Diagnoses[last].Diagnosis == d && actor.is(c.view.Diagnoses[last].Author) {
		return c.view.Diagnoses[last], nil
	}
	if d.Evidence.Kind != EvidenceChannelTest {
		return DiagnosisView{}, refuse(ReasonExportEvidence)
	}
	v := DiagnosisView{Diagnosis: d, Revision: len(c.view.Diagnoses) + 1, Author: actor.ID, At: now}
	c.view.Diagnoses = append(c.view.Diagnoses, v)
	c.view.Open, c.view.Opener, c.view.Opened = false, "", 0
	return v, nil
}

func (r *Registry) Open(workspace, id string, revision int, actor Actor) (ChannelView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || revision < 1 || !actor.valid() {
		return ChannelView{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return ChannelView{}, err
	}
	_, c, err := r.channel(workspace, id)
	if err != nil {
		return ChannelView{}, err
	}
	if c.view.Open && c.view.Opened == revision {
		if actor.is(c.view.Opener) {
			return viewChannel(c), nil
		}
		return ChannelView{}, ErrConflict
	}
	last := len(c.view.Diagnoses)
	if c.view.Channel.Kind == ChannelERP {
		return ChannelView{}, refuse(ReasonERPGateUnbound)
	}
	if last == 0 || revision != last {
		return ChannelView{}, refuse(ReasonStaleDiagnosis)
	}
	d := c.view.Diagnoses[last-1]
	switch {
	case actor.ID == d.Author:
		return ChannelView{}, refuse(ReasonSelfApproval)
	case !d.Diagnosis.FieldControl || !d.Diagnosis.AttachmentControl:
		return ChannelView{}, refuse(ReasonCapabilityUnproven)
	case d.Diagnosis.AddsAttachments:
		return ChannelView{}, refuse(ReasonAddsAttachments)
	}
	c.view.Open, c.view.Opener, c.view.Opened, c.view.Closer = true, actor.ID, revision, ""
	return viewChannel(c), nil
}

func (r *Registry) Close(workspace, id string, actor Actor) (ChannelView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return ChannelView{}, ErrInvalid
	}
	if err := r.staff(actor); err != nil {
		return ChannelView{}, err
	}
	_, c, err := r.channel(workspace, id)
	if err != nil {
		return ChannelView{}, err
	}
	if !c.view.Open {
		if actor.is(c.view.Closer) {
			return viewChannel(c), nil
		}
		return ChannelView{}, ErrConflict
	}
	c.view.Open, c.view.Closer = false, actor.ID
	return viewChannel(c), nil
}

func (r *Registry) ChannelOf(workspace, id string) (ChannelView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return ChannelView{}, ErrInvalid
	}
	_, c, err := r.channel(workspace, id)
	if err != nil {
		return ChannelView{}, err
	}
	return viewChannel(c), nil
}
