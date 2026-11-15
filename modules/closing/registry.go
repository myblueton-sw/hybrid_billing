package closing

import (
	"sync"
	"time"
)

type pendingWaiver struct {
	waiver    Waiver
	requester string
}

type item struct {
	target   string
	stage    *stage
	state    string
	waiver   *pendingWaiver
	approver string
	events   []ItemEvent
	claimed  map[time.Time]int
}

type stage struct {
	index     int
	view      StageView
	items     []*item
	byTarget  map[string]*item
	owed      map[*item]bool
	requester string
	approver  string
}

func (st *stage) sync(it *item) {
	if it.state == ItemOpen && it.claimed[st.view.CurrentDue] < len(steps) {
		st.owed[it] = true
		return
	}
	delete(st.owed, it)
}

type cycle struct {
	view     CycleView
	calendar *calendar
	stages   []*stage
	byStage  map[string]*stage
	seq      int
	events   int
	last     Addition
	adder    string
}

type space struct {
	calendars map[string][]*calendar
	cycles    map[string]*cycle
	targets   int
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
	if config.Zones == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	return &Registry{config: config, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	if l.Lead < minLead || l.Lead > maxLead {
		return false
	}
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Calendars, maxCalendars}, {l.Cycles, maxCycles}, {l.Targets, maxTargets}, {l.History, maxHistory}, {l.Claims, maxClaims}}
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

func (a Actor) human() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(RefusalAINotAllowed)
	case KindSystem:
		return refuse(RefusalNotHuman)
	}
	return nil
}

func (a Actor) notAssistant() error {
	if a.Kind == KindAssistant {
		return refuse(RefusalAINotAllowed)
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
	s := &space{calendars: map[string][]*calendar{}, cycles: map[string]*cycle{}}
	r.workspaces[workspace] = s
	return s, nil
}

func calendarForm(c Calendar) bool {
	if !validID(c.ID) || c.Zone == "" || c.Zone == "Local" || len(c.Zone) > maxIDBytes || len(c.Weekend) > maxWeekend || len(c.Holidays) > maxHolidays || c.Close < 1 || c.Close > endOfDay {
		return false
	}
	from, ok := parseDate(c.Covers.From)
	to, valid := parseDate(c.Covers.To)
	if !ok || !valid || to.Before(from) {
		return false
	}
	seen := map[time.Weekday]bool{}
	for _, day := range c.Weekend {
		if day < time.Sunday || day > time.Saturday || seen[day] {
			return false
		}
		seen[day] = true
	}
	for _, day := range c.Holidays {
		if _, ok := parseDate(day); !ok {
			return false
		}
	}
	return true
}

func copyCalendar(c Calendar) Calendar {
	c.Weekend = append([]time.Weekday(nil), c.Weekend...)
	c.Holidays = append([]string(nil), c.Holidays...)
	return c
}

func sameCalendar(a, b Calendar) bool {
	if a.ID != b.ID || a.Zone != b.Zone || a.Close != b.Close || a.Covers != b.Covers || len(a.Weekend) != len(b.Weekend) || len(a.Holidays) != len(b.Holidays) {
		return false
	}
	for index := range a.Weekend {
		if a.Weekend[index] != b.Weekend[index] {
			return false
		}
	}
	for index := range a.Holidays {
		if a.Holidays[index] != b.Holidays[index] {
			return false
		}
	}
	return true
}

func viewCalendar(c *calendar) CalendarView {
	v := c.view
	v.Calendar = copyCalendar(v.Calendar)
	return v
}

func (r *Registry) ProposeCalendar(workspace string, c Calendar, actor Actor) (CalendarView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !calendarForm(c) {
		return CalendarView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CalendarView{}, err
	}
	if s := r.workspaces[workspace]; s != nil {
		revisions := s.calendars[c.ID]
		if last := len(revisions) - 1; last >= 0 && revisions[last].view.State == StateProposed {
			if actor.is(revisions[last].view.Proposer) && sameCalendar(revisions[last].view.Calendar, c) {
				return viewCalendar(revisions[last]), nil
			}
			return CalendarView{}, refuse(RefusalCalendarPending)
		}
	}
	location, err := r.config.Zones(c.Zone)
	if err != nil || location == nil {
		return CalendarView{}, refuse(RefusalZoneUnavailable)
	}
	s, err := r.open(workspace)
	if err != nil {
		return CalendarView{}, err
	}
	count := 0
	for _, list := range s.calendars {
		count += len(list)
	}
	if count >= r.limits.Calendars {
		return CalendarView{}, ErrLimit
	}
	c = copyCalendar(c)
	from, _ := parseDate(c.Covers.From)
	to, _ := parseDate(c.Covers.To)
	made := &calendar{view: CalendarView{Calendar: c, Revision: len(s.calendars[c.ID]) + 1, State: StateProposed, Proposer: actor.ID}, location: location,
		weekend: map[time.Weekday]bool{}, holidays: map[string]bool{}, from: from, to: to}
	for _, day := range c.Weekend {
		made.weekend[day] = true
	}
	for _, day := range c.Holidays {
		made.holidays[day] = true
	}
	s.calendars[c.ID] = append(s.calendars[c.ID], made)
	return viewCalendar(made), nil
}

func (r *Registry) ApproveCalendar(workspace, id string, revision int, actor Actor) (CalendarView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || revision < 1 || !actor.valid() {
		return CalendarView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CalendarView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || revision > len(s.calendars[id]) {
		return CalendarView{}, ErrNotFound
	}
	c := s.calendars[id][revision-1]
	if c.view.State == StateActive {
		if actor.is(c.view.Approver) {
			return viewCalendar(c), nil
		}
		return CalendarView{}, ErrConflict
	}
	if actor.ID == c.view.Proposer {
		return CalendarView{}, refuse(RefusalSelfApproval)
	}
	c.view.State, c.view.Approver = StateActive, actor.ID
	return viewCalendar(c), nil
}

var stageOrder = map[string]int{StageCollect: 1, StageValidate: 2, StageApprove: 3, StageIssue: 4, StageDeliver: 5, StageSettle: 6}

var bases = map[string]string{BasisInternal: EvidenceInternal, BasisERP: EvidenceERP, BasisFile: EvidenceFile}

func cycleForm(c Cycle) bool {
	if _, err := time.Parse("2006-01", c.Period); err != nil || !validID(c.ID) || !validID(c.Calendar) || c.Revision < 1 || len(c.Stages) == 0 || len(c.Period) != 7 {
		return false
	}
	last := 0
	for _, spec := range c.Stages {
		order := stageOrder[spec.Stage]
		if order <= last || bases[spec.Basis] == "" || spec.Rule.Days < 1 || spec.Rule.Days > maxDays {
			return false
		}
		last = order
	}
	return true
}

func copyCycle(c Cycle) Cycle {
	c.Stages = append([]StageSpec(nil), c.Stages...)
	return c
}

func sameCycle(a, b Cycle) bool {
	if a.ID != b.ID || a.Period != b.Period || a.Calendar != b.Calendar || a.Revision != b.Revision || len(a.Stages) != len(b.Stages) {
		return false
	}
	for index := range a.Stages {
		if a.Stages[index] != b.Stages[index] {
			return false
		}
	}
	return true
}

func (c *cycle) snapshot() CycleView {
	v := c.view
	v.Cycle = copyCycle(v.Cycle)
	v.Sequence = c.seq
	v.Targets = append([]string(nil), v.Targets...)
	v.Stages = nil
	for _, st := range c.stages {
		sv := st.view
		sv.History = append([]DueChange(nil), sv.History...)
		if sv.Pending != nil {
			p := *sv.Pending
			sv.Pending = &p
		}
		v.Stages = append(v.Stages, sv)
	}
	return v
}

func (r *Registry) OpenCycle(workspace string, c Cycle, actor Actor) (CycleView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !cycleForm(c) {
		return CycleView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return CycleView{}, err
	}
	s := r.workspaces[workspace]
	if s != nil && s.cycles[c.ID] != nil {
		found := s.cycles[c.ID]
		if actor.is(found.view.Opener) && sameCycle(found.view.Cycle, c) {
			return found.snapshot(), nil
		}
		return CycleView{}, ErrConflict
	}
	if s == nil || c.Revision > len(s.calendars[c.Calendar]) {
		return CycleView{}, ErrNotFound
	}
	cal := s.calendars[c.Calendar][c.Revision-1]
	if cal.view.State != StateActive {
		return CycleView{}, refuse(RefusalNotApproved)
	}
	var stages []*stage
	var previous time.Time
	for index, spec := range c.Stages {
		due, err := cal.deadline(c.Period, spec.Rule)
		if err != nil {
			return CycleView{}, err
		}
		if due.Before(previous) {
			return CycleView{}, refuse(RefusalStageOrder)
		}
		previous = due
		stages = append(stages, &stage{index: index, view: StageView{Spec: spec, OriginalDue: due, CurrentDue: due}, byTarget: map[string]*item{}, owed: map[*item]bool{}})
	}
	if len(s.cycles) >= r.limits.Cycles {
		return CycleView{}, ErrLimit
	}
	made := &cycle{view: CycleView{Cycle: copyCycle(c), Opener: actor.ID, Opened: now}, calendar: cal, stages: stages, byStage: map[string]*stage{}, seq: 1}
	for _, st := range stages {
		made.byStage[st.view.Spec.Stage] = st
	}
	s.cycles[c.ID] = made
	return made.snapshot(), nil
}

func (r *Registry) cycle(workspace, id string) (*space, *cycle, error) {
	s := r.workspaces[workspace]
	if s == nil || s.cycles[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.cycles[id], nil
}

func (r *Registry) CycleOf(workspace, id string) (CycleView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return CycleView{}, ErrInvalid
	}
	_, c, err := r.cycle(workspace, id)
	if err != nil {
		return CycleView{}, err
	}
	return c.snapshot(), nil
}

func (r *Registry) AddTargets(workspace, id string, a Addition, actor Actor) (CycleView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(a.Reason) || len(a.Targets) == 0 || !actor.valid() {
		return CycleView{}, ErrInvalid
	}
	seen := map[string]bool{}
	for _, target := range a.Targets {
		if !validID(target) || seen[target] {
			return CycleView{}, ErrInvalid
		}
		seen[target] = true
	}
	if err := actor.human(); err != nil {
		return CycleView{}, err
	}
	s, c, err := r.cycle(workspace, id)
	if err != nil {
		return CycleView{}, err
	}
	if actor.is(c.adder) && sameAddition(c.last, a) {
		return c.snapshot(), nil
	}
	for _, target := range a.Targets {
		if c.stages[0].byTarget[target] != nil {
			return CycleView{}, refuse(RefusalTargetPresent)
		}
	}
	if s.targets+len(a.Targets) > r.limits.Targets {
		return CycleView{}, ErrLimit
	}
	if err := r.room(c, len(a.Targets)*len(c.stages)); err != nil {
		return CycleView{}, err
	}
	c.seq++
	c.last, c.adder = Addition{Targets: append([]string(nil), a.Targets...), Reason: a.Reason}, actor.ID
	s.targets += len(a.Targets)
	for _, target := range a.Targets {
		c.view.Targets = append(c.view.Targets, target)
		for _, st := range c.stages {
			it := &item{target: target, stage: st, state: ItemOpen, claimed: map[time.Time]int{}}
			it.events = append(it.events, ItemEvent{Sequence: c.seq, Kind: EventAdded, Actor: actor.ID, At: now, Reason: a.Reason})
			st.items = append(st.items, it)
			st.byTarget[target] = it
			st.sync(it)
		}
	}
	return c.snapshot(), nil
}

func sameAddition(a, b Addition) bool {
	if a.Reason != b.Reason || len(a.Targets) != len(b.Targets) {
		return false
	}
	for index := range a.Targets {
		if a.Targets[index] != b.Targets[index] {
			return false
		}
	}
	return true
}

func (r *Registry) room(c *cycle, events int) error {
	if c.events+events > r.limits.History {
		return ErrLimit
	}
	c.events += events
	return nil
}
