package incidents

import (
	"sort"
	"strconv"
	"time"
)

func (s *space) settingsOf(person string) (Settings, *time.Location) {
	if set := s.settings[person]; set != nil {
		return set.view, set.location
	}
	return Settings{Recipient: person, Zone: defaultZoneName, Digest: DigestImmediate}, time.UTC
}

func quiet(set Settings, location *time.Location, at time.Time) bool {
	if set.Quiet == nil {
		return false
	}
	local := at.In(location)
	minute := local.Hour()*60 + local.Minute()
	if set.Quiet.Start < set.Quiet.End {
		return minute >= set.Quiet.Start && minute < set.Quiet.End
	}
	return minute >= set.Quiet.Start || minute < set.Quiet.End
}

func cadence(digest string) time.Duration {
	if digest == DigestHourly {
		return hour
	}
	return day
}

type copyKey struct {
	target   string
	incident string
}

type change struct {
	key    [2]string
	reason string
}

type batch struct {
	immediate []string
	digest    []string
	via       map[string][]string
	severity  map[string]string
}

func (s *space) snoozed(person, incident string, at time.Time) bool {
	until, ok := s.snoozes[[2]string{person, incident}]
	return ok && until.After(at)
}

func (s *space) withhold(changes *[]change, target, incident, reason string) {
	key := [2]string{target, incident}
	if s.withheld[key] != reason {
		*changes = append(*changes, change{key: key, reason: reason})
	}
}

func (r *Registry) due(s *space, target string, inc *incident, at time.Time) bool {
	set, location := s.settingsOf(target)
	critical := inc.view.Severity == SeverityCritical
	if !critical && quiet(set, location, at) {
		return false
	}
	if last, ok := s.handed[[2]string{target, inc.view.ID}]; ok && rank[inc.view.Severity] <= rank[last.severity] {
		interval := set.Renotify[inc.view.Severity]
		if interval == 0 || at.Sub(last.at) < interval {
			return false
		}
	}
	if p, ok := s.pending[[2]string{target, inc.view.ID}]; ok && at.Sub(p.planned) < hour && !(critical && p.severity != SeverityCritical) {
		return false
	}
	return true
}

func (r *Registry) Plan(workspace string, actor Actor) ([]Notification, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.only(KindSystem); err != nil {
		return nil, err
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, nil
	}
	var open []*incident
	for _, inc := range s.open {
		open = append(open, inc)
	}
	sort.Slice(open, func(i, j int) bool { return open[i].number < open[j].number })
	var subs []*Subscription
	for _, sub := range s.subscriptions {
		subs = append(subs, sub)
	}
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Recipient < subs[j].Recipient || subs[i].Recipient == subs[j].Recipient && subs[i].Condition < subs[j].Condition
	})
	var changes []change
	batches := map[string]*batch{}
	var targets []string
	cleared := map[[2]string]bool{}
	type candidate struct {
		inc    *incident
		target string
		vias   []string
	}
	candidates := map[copyKey]*candidate{}
	var keys []copyKey
	for _, inc := range open {
		for _, sub := range subs {
			if sub.Condition != inc.view.Condition || rank[inc.view.Severity] < rank[sub.Minimum] {
				continue
			}
			recipient := sub.Recipient
			target := recipient
			if set, _ := s.settingsOf(recipient); set.Away {
				target = set.Alternate
			}
			if s.snoozed(recipient, inc.view.ID, now) || target != recipient && s.snoozed(target, inc.view.ID, now) {
				continue
			}
			if alt, _ := s.settingsOf(target); target != recipient && alt.Away {
				s.withhold(&changes, recipient, inc.view.ID, WithheldNoTarget)
				continue
			}
			key := copyKey{target, inc.view.ID}
			if candidates[key] == nil {
				candidates[key] = &candidate{inc: inc, target: target}
				keys = append(keys, key)
			}
			candidates[key].vias = append(candidates[key].vias, recipient)
		}
	}
	answers := map[[2]string]error{}
	for _, key := range keys {
		c := candidates[key]
		inc, target := c.inc, c.target
		if !r.due(s, target, inc, now) {
			continue
		}
		pkey := [2]string{target, inc.view.Subject}
		answer, asked := answers[pkey]
		if !asked {
			ok, err := r.permitted(workspace, target, inc.view.Subject)
			answer = err
			if err == nil && !ok {
				answer = refuse(RefusalNotPermitted)
			}
			answers[pkey] = answer
		}
		switch {
		case reasonOf(answer) == RefusalPermissionUnavail:
			s.withhold(&changes, target, inc.view.ID, WithheldUnavailable)
			continue
		case answer != nil:
			s.withhold(&changes, target, inc.view.ID, WithheldRefused)
			continue
		}
		cleared[[2]string{target, inc.view.ID}] = true
		for _, via := range c.vias {
			cleared[[2]string{via, inc.view.ID}] = true
		}
		b := batches[target]
		if b == nil {
			b = &batch{via: map[string][]string{}, severity: map[string]string{}}
			batches[target] = b
			targets = append(targets, target)
		}
		b.via[inc.view.ID] = c.vias
		b.severity[inc.view.ID] = inc.view.Severity
		tset, _ := s.settingsOf(target)
		if inc.view.Severity == SeverityCritical || tset.Digest == DigestImmediate {
			b.immediate = append(b.immediate, inc.view.ID)
		} else {
			b.digest = append(b.digest, inc.view.ID)
		}
	}
	sort.Strings(targets)
	var made []*notification
	for _, target := range targets {
		b := batches[target]
		if len(b.immediate) > 0 {
			made = append(made, s.draft(target, b.immediate, false, b, now))
		}
		set, _ := s.settingsOf(target)
		last, ok := s.lastDigest[target]
		waiting := s.pendingDigest[target]
		if len(b.digest) > 0 && (!ok || !now.Before(last.Add(cadence(set.Digest)))) && (waiting == nil || now.Sub(waiting.view.Planned) >= hour) {
			made = append(made, s.draft(target, b.digest, true, b, now))
		}
	}
	changes = unique(changes)
	if s.notes+len(made)+len(changes) > r.limits.Notifications {
		return nil, ErrLimit
	}
	s.commit(changes, now)
	for key := range cleared {
		delete(s.withheld, key)
	}
	var out []Notification
	for _, n := range made {
		n.view.ID = "note-" + strconv.Itoa(len(s.order)+1)
		s.order = append(s.order, n)
		s.notifications[n.view.ID] = n
		for _, id := range n.view.Incidents {
			s.pending[[2]string{n.view.Target, id}] = pending{notification: n, planned: now, severity: n.planned[id]}
		}
		if n.view.Digest {
			s.pendingDigest[n.view.Target] = n
		}
		out = append(out, viewNote(n))
	}
	s.notes += len(made) + len(changes)
	return out, nil
}

func (s *space) draft(target string, ids []string, digest bool, b *batch, now time.Time) *notification {
	n := &notification{view: Notification{Target: target, Incidents: append([]string(nil), ids...), Digest: digest, Planned: now}, via: map[string][]string{}, planned: map[string]string{}}
	for _, id := range ids {
		n.via[id] = append([]string(nil), b.via[id]...)
		n.planned[id] = b.severity[id]
	}
	return n
}

func viewNote(n *notification) Notification {
	v := n.view
	v.Incidents = append([]string(nil), v.Incidents...)
	v.Delivered = append([]string(nil), v.Delivered...)
	return v
}

func (r *Registry) Dispatch(workspace, id string, actor Actor) (Notification, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return Notification{}, ErrInvalid
	}
	if err := actor.only(KindSystem); err != nil {
		return Notification{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.notifications[id] == nil {
		return Notification{}, ErrNotFound
	}
	n := s.notifications[id]
	if n.view.HandedOff {
		return viewNote(n), nil
	}
	target := n.view.Target
	set, location := s.settingsOf(target)
	var changes []change
	var delivered []string
	for _, incID := range n.view.Incidents {
		inc := s.incidents[incID]
		key := [2]string{target, incID}
		if last, ok := s.handed[key]; ok && !last.at.Before(n.view.Planned) && rank[inc.view.Severity] <= rank[last.severity] {
			if interval := set.Renotify[inc.view.Severity]; interval == 0 || now.Sub(last.at) < interval {
				continue
			}
		}
		current, awake := false, false
		for _, via := range n.via[incID] {
			if s.routes(via, target) {
				current = true
				awake = awake || !s.snoozed(via, incID, now) && !s.snoozed(target, incID, now)
			}
		}
		switch {
		case inc.view.Resolved:
			s.withhold(&changes, target, incID, WithheldClosed)
			continue
		case !current:
			s.withhold(&changes, target, incID, WithheldNoTarget)
			continue
		case !awake:
			s.withhold(&changes, target, incID, WithheldSnoozed)
			continue
		case inc.view.Severity != SeverityCritical && quiet(set, location, now):
			continue
		}
		ok, err := r.permitted(workspace, target, inc.view.Subject)
		switch {
		case err != nil:
			s.withhold(&changes, target, incID, WithheldUnavailable)
			continue
		case !ok:
			s.withhold(&changes, target, incID, WithheldRefused)
			continue
		}
		delivered = append(delivered, incID)
	}
	changes = unique(changes)
	if s.notes+len(changes) > r.limits.Notifications {
		return Notification{}, ErrLimit
	}
	s.commit(changes, now)
	s.notes += len(changes)
	for _, incID := range n.view.Incidents {
		key := [2]string{target, incID}
		if p, ok := s.pending[key]; ok && p.notification == n {
			delete(s.pending, key)
		}
	}
	for _, incID := range delivered {
		s.handed[[2]string{target, incID}] = sent{at: now, severity: s.incidents[incID].view.Severity}
	}
	if n.view.Digest && len(delivered) > 0 {
		s.lastDigest[target] = now
	}
	if s.pendingDigest[target] == n {
		delete(s.pendingDigest, target)
	}
	n.view.HandedOff, n.view.Handoff, n.view.Delivered = true, now, delivered
	return viewNote(n), nil
}

func (r *Registry) Open(workspace, noteID, incID string, actor Actor) (Incident, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(noteID) || !validID(incID) || !actor.valid() {
		return Incident{}, ErrInvalid
	}
	if err := actor.only(KindHuman); err != nil {
		return Incident{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.notifications[noteID] == nil {
		return Incident{}, refuse(RefusalNotTarget)
	}
	n := s.notifications[noteID]
	listed := false
	for _, id := range n.view.Delivered {
		listed = listed || id == incID
	}
	if n.view.Target != actor.ID || !n.view.HandedOff || !listed {
		return Incident{}, refuse(RefusalNotTarget)
	}
	inc := s.incidents[incID]
	ok, err := r.permitted(workspace, actor.ID, inc.view.Subject)
	if err != nil {
		return Incident{}, err
	}
	if !ok {
		return Incident{}, refuse(RefusalPermissionRevoked)
	}
	return inc.view, nil
}

func (r *Registry) Withholdings(workspace string, actor Actor) ([]Withholding, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.only(KindSystem); err != nil {
		return nil, err
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, nil
	}
	return append([]Withholding(nil), s.withholdings...), nil
}

func unique(changes []change) []change {
	seen := map[[2]string]bool{}
	var out []change
	for _, c := range changes {
		if !seen[c.key] {
			seen[c.key] = true
			out = append(out, c)
		}
	}
	return out
}

func (s *space) commit(changes []change, now time.Time) {
	for _, c := range changes {
		s.withheld[c.key] = c.reason
		s.withholdings = append(s.withholdings, Withholding{Target: c.key[0], Incident: c.key[1], Reason: c.reason, At: now})
	}
}

func (s *space) routes(via, target string) bool {
	set, _ := s.settingsOf(via)
	if via == target {
		return !set.Away
	}
	alternate, _ := s.settingsOf(target)
	return set.Away && set.Alternate == target && !alternate.Away
}

func reasonOf(err error) string {
	if refusal, ok := err.(Refusal); ok {
		return refusal.Reason
	}
	return ""
}
