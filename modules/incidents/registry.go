package incidents

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

type incident struct {
	view   Incident
	number int
}

type signalRecord struct {
	signal   Signal
	incident *incident
}

type settings struct {
	view     Settings
	location *time.Location
}

type sent struct {
	at       time.Time
	severity string
}

type pending struct {
	notification *notification
	planned      time.Time
	severity     string
}

type notification struct {
	view    Notification
	via     map[string][]string
	planned map[string]string
}

type space struct {
	incidents     map[string]*incident
	open          map[[2]string]*incident
	resolved      map[[2]string][]*incident
	signals       map[string]*signalRecord
	subscriptions map[[2]string]*Subscription
	settings      map[string]*settings
	snoozes       map[[2]string]time.Time
	notifications map[string]*notification
	order         []*notification
	handed        map[[2]string]sent
	pending       map[[2]string]pending
	withheld      map[[2]string]string
	withholdings  []Withholding
	lastDigest    map[string]time.Time
	pendingDigest map[string]*notification
	notes         int
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
	if config.Permissions == nil || config.Zones == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Incidents, maxIncidents}, {l.Signals, maxSignals}, {l.Recipients, maxRecipients}, {l.Subscriptions, maxSubscriptions}, {l.Notifications, maxNotes}}
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

func (a Actor) only(kind string) error {
	switch {
	case a.Kind == KindAssistant:
		return refuse(RefusalAINotAllowed)
	case a.Kind != kind && kind == KindHuman:
		return refuse(RefusalNotHuman)
	case a.Kind != kind:
		return refuse(RefusalNotSystem)
	}
	return nil
}

func (a Actor) reader() error {
	if a.Kind == KindAssistant {
		return refuse(RefusalAINotAllowed)
	}
	return nil
}

var rank = map[string]int{SeverityInfo: 1, SeverityWarning: 2, SeverityCritical: 3}

var reasons = map[string]bool{ReasonFixed: true, ReasonFalseAlarm: true, ReasonSuperseded: true, ReasonOther: true}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{incidents: map[string]*incident{}, open: map[[2]string]*incident{}, resolved: map[[2]string][]*incident{}, signals: map[string]*signalRecord{},
		subscriptions: map[[2]string]*Subscription{}, settings: map[string]*settings{}, snoozes: map[[2]string]time.Time{}, notifications: map[string]*notification{},
		handed: map[[2]string]sent{}, pending: map[[2]string]pending{}, withheld: map[[2]string]string{}, lastDigest: map[string]time.Time{}, pendingDigest: map[string]*notification{}}
	r.workspaces[workspace] = s
	return s, nil
}

func (r *Registry) permitted(workspace, person, subject string) (bool, error) {
	ok, err := r.config.Permissions.Allowed(workspace, person, subject)
	if err != nil {
		return false, refuse(RefusalPermissionUnavail)
	}
	return ok, nil
}

func signalForm(s Signal) bool {
	return validID(s.ID) && validID(s.Condition) && validID(s.Subject) && rank[s.Severity] > 0 && !s.Occurred.IsZero()
}

func (r *Registry) Raise(workspace string, sig Signal, actor Actor) (Incident, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !signalForm(sig) {
		return Incident{}, ErrInvalid
	}
	if err := actor.only(KindSystem); err != nil {
		return Incident{}, err
	}
	sig.Occurred = sig.Occurred.UTC().Round(0)
	if s := r.workspaces[workspace]; s != nil && s.signals[sig.ID] != nil {
		found := s.signals[sig.ID]
		if found.signal == sig {
			return found.incident.view, nil
		}
		return Incident{}, ErrConflict
	}
	if sig.Occurred.After(now) {
		return Incident{}, refuse(RefusalFutureOccurrence)
	}
	s, err := r.open(workspace)
	if err != nil {
		return Incident{}, err
	}
	if len(s.signals) >= r.limits.Signals {
		return Incident{}, ErrLimit
	}
	key := [2]string{sig.Condition, sig.Subject}
	var target *incident
	for _, old := range s.resolved[key] {
		if !sig.Occurred.After(old.view.ResolvedAt) {
			target = old
			break
		}
	}
	switch {
	case target != nil:
		target.view.Late++
		if sig.Occurred.Before(target.view.First) {
			target.view.First = sig.Occurred
		}
	case s.open[key] != nil:
		target = s.open[key]
		v := &target.view
		v.Count++
		if sig.Occurred.Before(v.First) {
			v.First = sig.Occurred
		}
		if sig.Occurred.After(v.Last) {
			v.Last = sig.Occurred
		}
		if rank[sig.Severity] > rank[v.Severity] {
			v.Severity = sig.Severity
		}
		v.Blocking = v.Blocking || sig.Blocking
	default:
		if len(s.incidents) >= r.limits.Incidents {
			return Incident{}, ErrLimit
		}
		target = &incident{number: len(s.incidents) + 1}
		target.view = Incident{ID: "inc-" + strconv.Itoa(target.number), Condition: sig.Condition, Subject: sig.Subject, Severity: sig.Severity,
			Blocking: sig.Blocking, First: sig.Occurred, Last: sig.Occurred, Count: 1, Opened: now}
		if history := s.resolved[key]; len(history) > 0 {
			target.view.Recurs = history[len(history)-1].view.ID
		}
		s.incidents[target.view.ID] = target
		s.open[key] = target
	}
	s.signals[sig.ID] = &signalRecord{signal: sig, incident: target}
	return target.view, nil
}

func (r *Registry) Resolve(workspace, id string, res Resolution, actor Actor) (Incident, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !reasons[res.Reason] || !validID(res.Evidence) || !actor.valid() {
		return Incident{}, ErrInvalid
	}
	if err := actor.reader(); err != nil {
		return Incident{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.incidents[id] == nil {
		return Incident{}, ErrNotFound
	}
	inc := s.incidents[id]
	if actor.Kind == KindHuman {
		ok, err := r.permitted(workspace, actor.ID, inc.view.Subject)
		if err != nil {
			return Incident{}, err
		}
		if !ok {
			return Incident{}, refuse(RefusalNotPermitted)
		}
	}
	if inc.view.Resolved {
		if inc.view.Resolution == res && inc.view.Resolver == actor.ID {
			return inc.view, nil
		}
		return Incident{}, refuse(RefusalResolved)
	}
	inc.view.Resolved, inc.view.Resolution, inc.view.Resolver, inc.view.ResolvedAt = true, res, actor.ID, now
	key := [2]string{inc.view.Condition, inc.view.Subject}
	delete(s.open, key)
	s.resolved[key] = append(s.resolved[key], inc)
	return inc.view, nil
}

func (r *Registry) IncidentOf(workspace, id string, actor Actor) (Incident, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return Incident{}, ErrInvalid
	}
	if err := actor.only(KindSystem); err != nil {
		return Incident{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.incidents[id] == nil {
		return Incident{}, ErrNotFound
	}
	return s.incidents[id].view, nil
}

func (r *Registry) Blocked(workspace, subject string, actor Actor) ([]Incident, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(subject) || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.reader(); err != nil {
		return nil, err
	}
	if actor.Kind == KindHuman {
		ok, err := r.permitted(workspace, actor.ID, subject)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, refuse(RefusalNotPermitted)
		}
	}
	var out []Incident
	if s := r.workspaces[workspace]; s != nil {
		for _, inc := range s.open {
			if inc.view.Subject == subject && inc.view.Blocking {
				out = append(out, inc.view)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Opened.Before(out[j].Opened) || out[i].Opened.Equal(out[j].Opened) && out[i].ID < out[j].ID
	})
	return out, nil
}

func (r *Registry) recipients(s *space) int {
	seen := map[string]bool{}
	for key := range s.subscriptions {
		seen[key[0]] = true
	}
	for key := range s.settings {
		seen[key] = true
	}
	return len(seen)
}

func (r *Registry) Subscribe(workspace string, sub Subscription, actor Actor) (Subscription, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(sub.Recipient) || !validID(sub.Condition) || rank[sub.Minimum] == 0 || !actor.valid() {
		return Subscription{}, ErrInvalid
	}
	if err := actor.only(KindHuman); err != nil {
		return Subscription{}, err
	}
	if actor.ID != sub.Recipient {
		return Subscription{}, refuse(RefusalNotSelf)
	}
	s, err := r.open(workspace)
	if err != nil {
		return Subscription{}, err
	}
	key := [2]string{sub.Recipient, sub.Condition}
	if !r.known(s, sub.Recipient) && s.settings[sub.Recipient] == nil && r.recipients(s) >= r.limits.Recipients ||
		s.subscriptions[key] == nil && len(s.subscriptions) >= r.limits.Subscriptions {
		return Subscription{}, ErrLimit
	}
	copied := sub
	s.subscriptions[key] = &copied
	return copied, nil
}

func (r *Registry) known(s *space, recipient string) bool {
	for key := range s.subscriptions {
		if key[0] == recipient {
			return true
		}
	}
	return false
}

func (r *Registry) Configure(workspace string, set Settings, actor Actor) (Settings, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set.Recipient) || !actor.valid() || set.Zone == "" || set.Zone == "Local" || len(set.Zone) > maxIDBytes ||
		set.Digest != DigestImmediate && set.Digest != DigestHourly && set.Digest != DigestDaily || set.Alternate != "" && !validID(set.Alternate) || set.Away && set.Alternate == "" {
		return Settings{}, ErrInvalid
	}
	if set.Quiet != nil && (set.Quiet.Start < 0 || set.Quiet.Start >= minutesInDay || set.Quiet.End < 0 || set.Quiet.End >= minutesInDay || set.Quiet.Start == set.Quiet.End) {
		return Settings{}, ErrInvalid
	}
	for severity, interval := range set.Renotify {
		if rank[severity] == 0 || interval != 0 && (interval < minRenotify || interval > maxRenotify) {
			return Settings{}, ErrInvalid
		}
	}
	if err := actor.only(KindHuman); err != nil {
		return Settings{}, err
	}
	if actor.ID != set.Recipient {
		return Settings{}, refuse(RefusalNotSelf)
	}
	if set.Alternate == set.Recipient {
		return Settings{}, refuse(RefusalAlternateIsSelf)
	}
	location, err := r.config.Zones(set.Zone)
	if err != nil || location == nil {
		return Settings{}, refuse(RefusalZoneUnavailable)
	}
	s, err := r.open(workspace)
	if err != nil {
		return Settings{}, err
	}
	if s.settings[set.Recipient] == nil && !r.known(s, set.Recipient) && r.recipients(s) >= r.limits.Recipients {
		return Settings{}, ErrLimit
	}
	set = copySettings(set)
	s.settings[set.Recipient] = &settings{view: set, location: location}
	return copySettings(set), nil
}

func copySettings(set Settings) Settings {
	if set.Quiet != nil {
		q := *set.Quiet
		set.Quiet = &q
	}
	renotify := map[string]time.Duration{}
	for k, v := range set.Renotify {
		renotify[k] = v
	}
	set.Renotify = renotify
	return set
}

func (r *Registry) Snooze(workspace, id string, until time.Time, actor Actor) (time.Time, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || until.IsZero() || !actor.valid() {
		return time.Time{}, ErrInvalid
	}
	if err := actor.only(KindHuman); err != nil {
		return time.Time{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.incidents[id] == nil {
		return time.Time{}, ErrNotFound
	}
	inc := s.incidents[id]
	if s.subscriptions[[2]string{actor.ID, inc.view.Condition}] == nil && !s.forwardsTo(actor.ID, inc.view.Condition) {
		return time.Time{}, refuse(RefusalNotSubscribed)
	}
	ok, err := r.permitted(workspace, actor.ID, inc.view.Subject)
	if err != nil {
		return time.Time{}, err
	}
	switch {
	case !ok:
		return time.Time{}, refuse(RefusalNotPermitted)
	case inc.view.Resolved:
		return time.Time{}, refuse(RefusalResolved)
	}
	until = until.UTC().Round(0)
	if !until.After(now) || until.Sub(now) > maxSnooze {
		return time.Time{}, refuse(RefusalSnoozeOutOfRange)
	}
	s.snoozes[[2]string{actor.ID, id}] = until
	return until, nil
}

func (s *space) forwardsTo(person, condition string) bool {
	for recipient, set := range s.settings {
		if set.view.Away && set.view.Alternate == person && s.subscriptions[[2]string{recipient, condition}] != nil {
			return true
		}
	}
	return false
}
