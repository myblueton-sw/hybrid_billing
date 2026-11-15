package recovery

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"time"
)

var (
	itemKinds  = map[string]bool{ItemSubmitted: true, ItemIssued: true, ItemDelivered: true, ItemPaid: true}
	eventKinds = map[string]bool{EventDelete: true, EventPurge: true, EventHold: true, EventExtend: true, EventRevokeKey: true, EventRevokeAccess: true}
	domainSet  = map[string]bool{}
)

func init() {
	for _, d := range domains {
		domainSet[d] = true
	}
}

func (r *Registry) fence(s *site, target Target) {
	key := [2]string{target.Workspace, target.Key}
	if !s.fenced[key] {
		s.fenced[key] = true
		r.keys++
	}
	if target.Series != "" {
		series := [2]string{target.Workspace, target.Series}
		if target.Number > s.high[series] {
			s.high[series] = target.Number
		}
	}
}

func (r *Registry) room(s *site, keys, series int) bool {
	return len(s.fenced)+keys <= maxKeys && r.keys+keys <= maxKeysTotal && len(s.high)+series <= maxSeries
}

func (r *Registry) begin(s *site, p RestorePoint, now time.Time) {
	r.reclaim(s, now)
	if s.current == nil {
		s.recoveries++
		s.current = &recovery{id: "recovery-" + strconv.Itoa(s.recoveries), listed: map[string]bool{}}
	}
	if s.state == StateServing {
		r.stop(s, now)
	}
	r.bump(s)
	s.state = StateRecovering
	rec := s.current
	rec.point, rec.digest, rec.beganSeq = &p, p.ManifestDigest, s.eventSeq
	rec.domains, rec.externals, rec.applied, rec.lost, rec.reapplying = map[string]*DomainStatus{}, map[string]*pages{}, map[uint64]bool{}, nil, false
	for _, d := range domains {
		rec.domains[d] = &DomainStatus{Domain: d, Status: DomainPending, Mismatch: []int{}}
	}
	for id := range s.externals {
		rec.externals[id] = &pages{next: 1}
	}
	after := p.AsOf.Add(-s.view.Skew)
	candidates := []*call{}
	for _, c := range s.open {
		candidates = append(candidates, c)
	}
	for _, c := range s.closed {
		if (c.view.Outcome == OutcomeConfirmed || c.view.Outcome == OutcomeUnknown) && (c.at.After(after) || c.closed.After(after)) {
			candidates = append(candidates, c)
		}
	}
	for _, c := range candidates {
		keys, series := 0, 0
		if !s.fenced[[2]string{c.view.Target.Workspace, c.view.Target.Key}] {
			keys = 1
		}
		if _, known := s.high[[2]string{c.view.Target.Workspace, c.view.Target.Series}]; c.view.Target.Series != "" && !known {
			series = 1
		}
		if !r.room(s, keys, series) {
			rec.skipped++
			continue
		}
		r.fence(s, c.view.Target)
		if c.view.Outcome != OutcomeConfirmed && !rec.listed[c.view.ID] {
			rec.listed[c.view.ID] = true
			rec.unknown = append(rec.unknown, c.view.Target)
		}
	}
}

func (r *Registry) remember(s *site, outcome string) {
	s.history = append(s.history, History{Recovery: s.current.id, ManifestDigest: s.current.digest, Generation: s.generation, Outcome: outcome, Lost: len(s.current.lost)})
	if len(s.history) > maxHistory {
		s.history = s.history[1:]
	}
}

func (r *Registry) abandon(s *site) {
	r.bump(s)
	r.remember(s, "abandoned")
	rec := s.current
	rec.point, rec.domains, rec.externals, rec.applied, rec.lost, rec.reapplying = nil, nil, nil, nil, nil, false
}

func (r *Registry) unlock(s *site) {
	if s.current != nil {
		r.remember(s, "completed")
		s.current = nil
	}
	s.state, s.stoppedAt, s.inflight = StateServing, time.Time{}, time.Time{}
}

func (r *Registry) verifying(id string, actor Actor) (*site, *recovery, error) {
	if !actor.valid() {
		return nil, nil, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case actor.Kind != KindSystem || actor.ID != s.verifier:
		return nil, nil, refuse(ReasonNotVerifier)
	case s.current == nil:
		return nil, nil, refuse(ReasonNotRecovering)
	case s.current.point == nil:
		return nil, nil, refuse(ReasonNoRestorePoint)
	}
	return s, s.current, nil
}

func (r *Registry) Report(id string, inv Inventory, actor Actor) (DomainStatus, error) {
	r.enter()
	defer r.leave()
	if !domainSet[inv.Domain] || !inv.valid() {
		return DomainStatus{}, ErrInvalid
	}
	_, rec, err := r.verifying(id, actor)
	if err != nil {
		return DomainStatus{}, err
	}
	if rec.reapplying {
		return DomainStatus{}, refuse(ReasonPhase)
	}
	var want Inventory
	for _, m := range rec.point.Manifest {
		if m.Domain == inv.Domain {
			want = m
		}
	}
	status := &DomainStatus{Domain: inv.Domain, Status: DomainVerified, Mismatch: []int{}}
	longest := len(want.Chunks)
	if len(inv.Chunks) > longest {
		longest = len(inv.Chunks)
	}
	for index := 0; index < longest; index++ {
		if index >= len(want.Chunks) || index >= len(inv.Chunks) || want.Chunks[index] != inv.Chunks[index] {
			if len(status.Mismatch) < maxMismatch {
				status.Mismatch = append(status.Mismatch, index)
			}
			status.Status = DomainMismatch
		}
	}
	if want.Items != inv.Items {
		status.Status = DomainMismatch
	}
	rec.domains[inv.Domain] = status
	out := *status
	out.Mismatch = append([]int{}, status.Mismatch...)
	return out, nil
}

func (it Item) valid(now time.Time) bool {
	return validID(it.Workspace) && validID(it.Key) && itemKinds[it.Kind] && !it.At.IsZero() && !it.At.After(now) &&
		(it.Series == "" && it.Number == 0 || validID(it.Series) && it.Number > 0)
}

func pageDigest(p Page) string {
	var e encoder
	e.count(p.Number)
	e.number(p.From.UnixNano())
	e.number(p.Through.UnixNano())
	e.text(strconv.FormatBool(p.Final))
	e.text(strconv.FormatBool(p.Rotated))
	e.count(len(p.Items))
	for _, it := range p.Items {
		for _, value := range []string{it.Workspace, it.Key, it.Kind, it.Series} {
			e.text(value)
		}
		e.number(it.At.UnixNano())
		e.number(it.Number)
	}
	sum := sha256.Sum256(e.out.Bytes())
	return hex.EncodeToString(sum[:])
}

func (r *Registry) maxCall(s *site) time.Duration {
	longest := s.stopCall
	for _, w := range s.writers {
		if !w.removed && w.view.MaxCall > longest {
			longest = w.view.MaxCall
		}
	}
	return longest
}

func (r *Registry) Reconcile(id, external string, p Page, actor Actor) (ExternalStatus, error) {
	now := r.enter()
	defer r.leave()
	if !validID(external) || p.Number < 1 || p.From.IsZero() || p.Through.Before(p.From) || p.Through.After(now) || len(p.Items) > maxItems {
		return ExternalStatus{}, ErrInvalid
	}
	for _, it := range p.Items {
		if !it.valid(now) {
			return ExternalStatus{}, ErrInvalid
		}
	}
	s, rec, err := r.verifying(id, actor)
	if err != nil {
		return ExternalStatus{}, err
	}
	x, ok := s.externals[external]
	if !ok {
		return ExternalStatus{}, ErrNotFound
	}
	pg := rec.externals[external]
	digest := pageDigest(p)
	status := ExternalStatus{ID: external, Kind: x.Kind, Status: ExternalPending}
	switch {
	case p.Number < pg.next && pg.digests[p.Number-1] == digest:
		if pg.final {
			status.Status = ExternalReconciled
		}
		return status, nil
	case p.Number < pg.next:
		return ExternalStatus{}, ErrConflict
	case pg.final:
		return ExternalStatus{}, refuse(ReasonNotPending)
	case p.Number > pg.next:
		return ExternalStatus{}, refuse(ReasonOutOfOrder)
	case pg.next > 1 && p.From.After(pg.through):
		return ExternalStatus{}, refuse(ReasonCoverage)
	case pg.next > maxPages:
		return ExternalStatus{}, ErrLimit
	}
	from, through := p.From, p.Through
	if pg.next > 1 {
		from = pg.from
		if pg.through.After(through) {
			through = pg.through
		}
	}
	if p.Final {
		skew := s.view.Skew
		required := s.stoppedAt.Add(r.maxCall(s) + skew)
		if s.inflight.Add(skew).After(required) {
			required = s.inflight.Add(skew)
		}
		if from.After(rec.point.AsOf.Add(-skew)) || through.Before(required) || !p.Rotated {
			return ExternalStatus{}, refuse(ReasonCoverage)
		}
	}
	keys, series := map[[2]string]bool{}, map[[2]string]bool{}
	for _, it := range p.Items {
		if k := [2]string{it.Workspace, it.Key}; !s.fenced[k] {
			keys[k] = true
		}
		if sr := [2]string{it.Workspace, it.Series}; it.Series != "" && it.Kind != ItemDelivered && it.Kind != ItemPaid {
			if _, known := s.high[sr]; !known {
				series[sr] = true
			}
		}
	}
	if !r.room(s, len(keys), len(series)) {
		return ExternalStatus{}, ErrLimit
	}
	for _, it := range p.Items {
		target := Target{Workspace: it.Workspace, Key: it.Key}
		if it.Kind == ItemIssued || it.Kind == ItemSubmitted {
			target.Series, target.Number = it.Series, it.Number
		}
		r.fence(s, target)
	}
	pg.digests = append(pg.digests, digest)
	pg.from, pg.through, pg.final = from, through, p.Final
	pg.next++
	if p.Final {
		status.Status = ExternalReconciled
	}
	return status, nil
}

func (r *Registry) Record(id string, e Event, actor Actor) (EventView, error) {
	now := r.enter()
	defer r.leave()
	if !actor.valid() || !eventKinds[e.Kind] || !domainSet[e.Domain] || !validID(e.Key) {
		return EventView{}, ErrInvalid
	}
	s, err := r.located(id)
	if err != nil {
		return EventView{}, err
	}
	w := s.writers[actor.ID]
	if actor.Kind != KindSystem || actor.ID != s.verifier && (w == nil || w.removed) {
		return EventView{}, refuse(ReasonNotWriter)
	}
	if len(s.events) >= maxEvents || r.eventsTotal >= maxEventsTotal {
		return EventView{}, ErrLimit
	}
	s.eventSeq++
	ev := &event{view: EventView{Sequence: s.eventSeq, Event: e, At: now}}
	s.events = append(s.events, ev)
	r.eventsTotal++
	return ev.view, nil
}

func (r *Registry) window(s *site) []*event {
	rec := s.current
	lower := rec.point.AsOf.Add(-s.view.Skew)
	out := []*event{}
	for _, ev := range s.events {
		if ev.view.At.After(lower) && ev.view.Sequence <= rec.beganSeq {
			out = append(out, ev)
		}
	}
	return out
}

func lostOf(rec *recovery) map[uint64]bool {
	out := map[uint64]bool{}
	for _, seq := range rec.lost {
		out[seq] = true
	}
	return out
}

func (r *Registry) pending(s *site) []*event {
	rec := s.current
	lost := lostOf(rec)
	out := []*event{}
	for _, ev := range r.window(s) {
		if !rec.applied[ev.view.Sequence] && !lost[ev.view.Sequence] {
			out = append(out, ev)
		}
	}
	return out
}

func verifiedAll(rec *recovery) bool {
	for _, d := range rec.domains {
		if d.Status != DomainVerified {
			return false
		}
	}
	return true
}

func (r *Registry) settle(id string, seq uint64, actor Actor) (*site, *recovery, *event, error) {
	s, rec, err := r.verifying(id, actor)
	if err != nil {
		return nil, nil, nil, err
	}
	if !verifiedAll(rec) {
		return nil, nil, nil, refuse(ReasonPhase)
	}
	for _, ev := range r.window(s) {
		if ev.view.Sequence == seq {
			return s, rec, ev, nil
		}
	}
	return nil, nil, nil, ErrNotFound
}

func (r *Registry) next(s *site, seq uint64) error {
	if p := r.pending(s); len(p) == 0 || p[0].view.Sequence != seq {
		return refuse(ReasonOutOfOrder)
	}
	return nil
}

func (r *Registry) Reapplied(id string, seq uint64, actor Actor) (EventView, error) {
	r.enter()
	defer r.leave()
	s, rec, ev, err := r.settle(id, seq, actor)
	switch {
	case err != nil:
		return EventView{}, err
	case rec.applied[seq]:
		return ev.view, nil
	case lostOf(rec)[seq]:
		return EventView{}, ErrConflict
	}
	if err := r.next(s, seq); err != nil {
		return EventView{}, err
	}
	rec.applied[seq], rec.reapplying = true, true
	return ev.view, nil
}

func (r *Registry) Unapplicable(id string, seq uint64, reason string, actor Actor) (EventView, error) {
	r.enter()
	defer r.leave()
	if !validText(reason) {
		return EventView{}, ErrInvalid
	}
	s, rec, ev, err := r.settle(id, seq, actor)
	switch {
	case err != nil:
		return EventView{}, err
	case lostOf(rec)[seq]:
		return ev.view, nil
	case rec.applied[seq]:
		return EventView{}, ErrConflict
	case ev.view.Event.Kind == EventRevokeKey || ev.view.Event.Kind == EventRevokeAccess:
		return EventView{}, refuse(ReasonMustApply)
	}
	if err := r.next(s, seq); err != nil {
		return EventView{}, err
	}
	rec.lost, rec.reapplying = append(rec.lost, seq), true
	return ev.view, nil
}

func lostDigest(list []uint64) string {
	var e encoder
	e.count(len(list))
	for _, seq := range list {
		e.text(strconv.FormatUint(seq, 10))
	}
	sum := sha256.Sum256(e.out.Bytes())
	return hex.EncodeToString(sum[:])
}

func (r *Registry) checkUnlock(s *site, u Unlock, now time.Time) error {
	r.reclaim(s, now)
	rec := s.current
	if rec == nil {
		switch {
		case s.state != StateLocked:
			return refuse(ReasonNotRecovering)
		case len(s.open) > 0:
			return refuse(ReasonOpenCalls)
		}
		return nil
	}
	if rec.point == nil {
		return refuse(ReasonNoRestorePoint)
	}
	for _, x := range s.externals {
		if !rec.externals[x.ID].final {
			return refuse(ReasonIncomplete)
		}
	}
	switch {
	case !verifiedAll(rec) || len(r.pending(s)) > 0:
		return refuse(ReasonIncomplete)
	case u.Lost != lostDigest(rec.lost):
		return refuse(ReasonLost)
	}
	kinds := map[string]bool{}
	for _, x := range s.externals {
		kinds[x.Kind] = true
	}
	for _, w := range s.writers {
		if !kinds[w.view.Kind] {
			return refuse(ReasonKindCoverage)
		}
	}
	return nil
}

func (r *Registry) Recovery(id string) (RecoveryView, error) {
	now := r.enter()
	defer r.leave()
	s, err := r.located(id)
	if err != nil {
		return RecoveryView{}, err
	}
	r.reclaim(s, now)
	rec := s.current
	if rec == nil {
		return RecoveryView{}, ErrNotFound
	}
	v := RecoveryView{ID: rec.id, State: s.state, Generation: s.generation, StoppedAt: s.stoppedAt, Domains: []DomainStatus{}, Externals: []ExternalStatus{}, Pending: []EventView{},
		Lost: append([]uint64{}, rec.lost...), Open: len(s.open), OpenCalls: []CallView{}, Unknown: append([]Target{}, rec.unknown...), Skipped: rec.skipped, Phase: PhaseVerify}
	for _, c := range s.open {
		v.OpenCalls = append(v.OpenCalls, c.view)
	}
	sort.Slice(v.OpenCalls, func(a, b int) bool { return v.OpenCalls[a].ID < v.OpenCalls[b].ID })
	if rec.point == nil {
		return v, nil
	}
	v.AsOf, v.Backup, v.ManifestDigest, v.PointGeneration = rec.point.AsOf, rec.point.Backup, rec.point.ManifestDigest, rec.point.Generation
	for _, d := range domains {
		status := *rec.domains[d]
		status.Mismatch = append([]int{}, status.Mismatch...)
		v.Domains = append(v.Domains, status)
	}
	for _, id := range sortedExternals(s) {
		status := ExternalStatus{ID: id, Kind: s.externals[id].Kind, Status: ExternalPending}
		if rec.externals[id].final {
			status.Status = ExternalReconciled
		}
		v.Externals = append(v.Externals, status)
	}
	for _, ev := range r.pending(s) {
		v.Pending = append(v.Pending, ev.view)
	}
	switch {
	case !verifiedAll(rec), !rec.reapplying && len(v.Pending) > 0:
	case len(v.Pending) > 0:
		v.Phase = PhaseReapply
	default:
		v.Phase = PhaseReady
	}
	return v, nil
}

func (r *Registry) History(id string) ([]History, error) {
	r.enter()
	defer r.leave()
	s, err := r.located(id)
	if err != nil {
		return nil, err
	}
	return append([]History{}, s.history...), nil
}
