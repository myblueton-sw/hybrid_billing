package assistant

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type tool struct {
	view ToolView
}

type conversation struct {
	view Conversation
	jobs map[string]bool
}

type job struct {
	serial  uint64
	view    JobView
	targets map[string]bool
	cards   map[string]*card
}

type card struct {
	serial   uint64
	view     CardView
	job      *job
	revision int
}

type space struct {
	tools         map[string]*tool
	conversations map[string]*conversation
	jobs          map[string]*job
	cards         map[string]*card
}

type Registry struct {
	mu         sync.Mutex
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	serial     uint64
	protected  map[string]bool
	workspaces map[string]*space
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Commands == nil || !validID(config.Client) || len(config.Protected) == 0 || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	protected := map[string]bool{}
	for _, kind := range config.Protected {
		if !validID(kind) {
			return nil, ErrInvalid
		}
		protected[kind] = true
	}
	return &Registry{config: config, limits: limits, clock: clock, protected: protected, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	if l.Duration < minDuration || l.Duration > maxDuration || l.Retention < minRetention || l.Retention > maxRetention {
		return false
	}
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Tools, maxTools}, {l.Jobs, maxJobs}, {l.Conversations, maxConversations}, {l.Cards, maxCards},
		{l.Calls, maxCalls}, {l.Targets, maxTargets}, {l.Fields, maxFields}, {l.Bytes, maxBytes}}
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

func validText(text string) bool {
	return len(text) <= maxTextBytes && utf8.ValidString(text)
}

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) is(id string) bool { return a.Kind == KindHuman && a.ID == id }

func (a Actor) human() error {
	switch a.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	}
	return nil
}

func (a Actor) assistant() error {
	if a.Kind != KindAssistant {
		return refuse(ReasonNotAssistant)
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
	s := &space{tools: map[string]*tool{}, conversations: map[string]*conversation{}, jobs: map[string]*job{}, cards: map[string]*card{}}
	r.workspaces[workspace] = s
	return s, nil
}

func (r *Registry) entry(command string) (commands.Entry, bool) {
	for _, e := range r.config.Commands.Catalogue() {
		if e.ID == command {
			return e, true
		}
	}
	return commands.Entry{}, false
}

var stages = map[string]bool{StageRead: true, StageDraft: true, StageSimulate: true, StageRequest: true}

func (r *Registry) admissible(e commands.Entry, stage string) error {
	switch {
	case !e.Supported:
		return refuse(ReasonUnsupported)
	case r.protected[e.Permission.Kind]:
		return refuse(ReasonProtected)
	}
	switch stage {
	case StageRead:
		if e.Mutating || e.Permission.Action != gate.View {
			return refuse(ReasonStage)
		}
	case StageDraft, StageSimulate:
		if !e.Mutating || e.Approval || e.Permission.Action != gate.View && e.Permission.Action != gate.RequestApproval {
			return refuse(ReasonStage)
		}
	}
	return nil
}

func toolForm(t Tool) bool {
	if !validID(t.Command) || !stages[t.Stage] || len(t.Results) > maxFields {
		return false
	}
	seen := map[string]bool{}
	for _, field := range t.Results {
		if !validID(field) || seen[field] {
			return false
		}
		seen[field] = true
	}
	return true
}

func copyTool(t Tool) Tool {
	t.Results = append([]string(nil), t.Results...)
	sort.Strings(t.Results)
	return t
}

func sameTool(a, b Tool) bool {
	a, b = copyTool(a), copyTool(b)
	return a.Command == b.Command && a.Stage == b.Stage && strings.Join(a.Results, "\x00") == strings.Join(b.Results, "\x00")
}

func viewTool(t *tool) ToolView {
	v := t.view
	v.Tool = copyTool(v.Tool)
	return v
}

func (r *Registry) ProposeTool(workspace string, t Tool, actor Actor) (ToolView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !toolForm(t) {
		return ToolView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ToolView{}, err
	}
	s := r.workspaces[workspace]
	if s != nil && s.tools[t.Command] != nil {
		found := s.tools[t.Command]
		if found.view.State == StateActive && sameTool(found.view.Tool, t) {
			return viewTool(found), nil
		}
		if found.view.State == StateProposed {
			if actor.is(found.view.Proposer) && sameTool(found.view.Tool, t) {
				return viewTool(found), nil
			}
			return ToolView{}, ErrConflict
		}
	}
	e, ok := r.entry(t.Command)
	if !ok {
		return ToolView{}, ErrNotFound
	}
	if err := r.admissible(e, t.Stage); err != nil {
		return ToolView{}, err
	}
	s, err := r.open(workspace)
	if err != nil {
		return ToolView{}, err
	}
	revision := 1
	if found := s.tools[t.Command]; found != nil {
		revision = found.view.Revision + 1
	} else if len(s.tools) >= r.limits.Tools {
		return ToolView{}, ErrLimit
	}
	made := &tool{view: ToolView{Tool: copyTool(t), Revision: revision, State: StateProposed, Proposer: actor.ID}}
	s.tools[t.Command] = made
	r.expire(s, t.Command)
	return viewTool(made), nil
}

func (r *Registry) ApproveTool(workspace, command string, actor Actor) (ToolView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(command) || !actor.valid() {
		return ToolView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ToolView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.tools[command] == nil {
		return ToolView{}, ErrNotFound
	}
	t := s.tools[command]
	switch t.view.State {
	case StateActive:
		if actor.is(t.view.Approver) {
			return viewTool(t), nil
		}
		return ToolView{}, ErrConflict
	case StateWithdrawn:
		return ToolView{}, refuse(ReasonNotPending)
	}
	if actor.ID == t.view.Proposer {
		return ToolView{}, refuse(ReasonSelfApproval)
	}
	e, ok := r.entry(command)
	if !ok {
		return ToolView{}, ErrNotFound
	}
	if err := r.admissible(e, t.view.Tool.Stage); err != nil {
		return ToolView{}, err
	}
	t.view.State, t.view.Approver = StateActive, actor.ID
	return viewTool(t), nil
}

func (r *Registry) WithdrawTool(workspace, command string, actor Actor) (ToolView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(command) || !actor.valid() {
		return ToolView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return ToolView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.tools[command] == nil {
		return ToolView{}, ErrNotFound
	}
	t := s.tools[command]
	if t.view.State == StateWithdrawn {
		return viewTool(t), nil
	}
	t.view.State = StateWithdrawn
	r.expire(s, command)
	return viewTool(t), nil
}

func (r *Registry) expire(s *space, command string) {
	for _, c := range s.cards {
		if c.view.Card.Tool == command && c.view.State == CardPending {
			c.view.State = CardExpired
		}
	}
}

func (r *Registry) OpenConversation(workspace string, c Conversation, actor Actor) (Conversation, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(c.ID) || !validID(c.Owner) || c.Expires.IsZero() || !actor.valid() {
		return Conversation{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return Conversation{}, err
	}
	c.Expires = c.Expires.UTC().Round(0)
	if s := r.workspaces[workspace]; s != nil && s.conversations[c.ID] != nil {
		if s.conversations[c.ID].view == c && actor.is(c.Owner) {
			return c, nil
		}
		return Conversation{}, ErrConflict
	}
	switch {
	case actor.ID != c.Owner:
		return Conversation{}, refuse(ReasonNotOwner)
	case !c.Expires.After(now) || c.Expires.Sub(now) > maxConversationAge:
		return Conversation{}, refuse(ReasonConversationExpiry)
	}
	s, err := r.open(workspace)
	if err != nil {
		return Conversation{}, err
	}
	if len(s.conversations) >= r.limits.Conversations {
		return Conversation{}, ErrLimit
	}
	s.conversations[c.ID] = &conversation{view: c, jobs: map[string]bool{}}
	return c, nil
}

func (r *Registry) DeleteConversation(workspace, id string, actor Actor) error {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return ErrInvalid
	}
	if actor.Kind == KindAssistant {
		return refuse(ReasonAINotAllowed)
	}
	s := r.workspaces[workspace]
	if s == nil || s.conversations[id] == nil {
		return ErrNotFound
	}
	c := s.conversations[id]
	switch {
	case actor.Kind == KindHuman && actor.ID != c.view.Owner:
		return refuse(ReasonNotOwner)
	case actor.Kind == KindSystem && now.Before(c.view.Expires):
		return refuse(ReasonConversationExpiry)
	}
	delete(s.conversations, id)
	return nil
}

func (r *Registry) CreateJob(workspace string, j Job, actor Actor) (JobView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(j.ID) || !validText(j.Purpose) || !actor.valid() {
		return JobView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return JobView{}, err
	}
	if s := r.workspaces[workspace]; s != nil && s.jobs[j.ID] != nil {
		found := s.jobs[j.ID]
		if actor.is(found.view.Owner) && found.view.Job == j {
			return viewJob(found), nil
		}
		return JobView{}, ErrConflict
	}
	s, err := r.open(workspace)
	if err != nil {
		return JobView{}, err
	}
	if len(s.jobs) >= r.limits.Jobs {
		return JobView{}, ErrLimit
	}
	r.serial++
	made := &job{serial: r.serial, view: JobView{Job: j, Owner: actor.ID, Created: now}, targets: map[string]bool{}, cards: map[string]*card{}}
	s.jobs[j.ID] = made
	return viewJob(made), nil
}

func viewJob(j *job) JobView {
	v := j.view
	v.Ledger = append([]Entry(nil), v.Ledger...)
	return v
}

func (r *Registry) JobOf(workspace, id string) (JobView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return JobView{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil || s.jobs[id] == nil {
		return JobView{}, ErrNotFound
	}
	return viewJob(s.jobs[id]), nil
}

func (r *Registry) Attach(workspace, jobID, conversationID string, actor Actor) (JobView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(jobID) || !validID(conversationID) || !actor.valid() {
		return JobView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return JobView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.jobs[jobID] == nil || s.conversations[conversationID] == nil {
		return JobView{}, ErrNotFound
	}
	j, c := s.jobs[jobID], s.conversations[conversationID]
	switch {
	case actor.ID != j.view.Owner || actor.ID != c.view.Owner:
		return JobView{}, refuse(ReasonNotOwner)
	case !now.Before(c.view.Expires):
		return JobView{}, refuse(ReasonConversation)
	case r.closed(j, now):
		return JobView{}, refuse(ReasonJobClosed)
	}
	c.jobs[jobID] = true
	return viewJob(j), nil
}

func (r *Registry) closed(j *job, now time.Time) bool {
	return j.view.Closed || !now.Before(j.view.Created.Add(r.limits.Duration))
}

func (r *Registry) CloseJob(workspace, id string, actor Actor) (JobView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return JobView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return JobView{}, err
	}
	s := r.workspaces[workspace]
	if s == nil || s.jobs[id] == nil {
		return JobView{}, ErrNotFound
	}
	j := s.jobs[id]
	if actor.ID != j.view.Owner {
		return JobView{}, refuse(ReasonNotOwner)
	}
	if !j.view.Closed {
		j.view.Closed, j.view.ClosedAt = true, now
	}
	return viewJob(j), nil
}

func (r *Registry) Purge(workspace string, actor Actor) (int, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() {
		return 0, ErrInvalid
	}
	if actor.Kind != KindSystem {
		return 0, refuse(ReasonNotSystem)
	}
	s := r.workspaces[workspace]
	if s == nil {
		return 0, nil
	}
	purged := 0
	for id, j := range s.jobs {
		ended := j.view.ClosedAt
		if !j.view.Closed {
			ended = j.view.Created.Add(r.limits.Duration)
		}
		if now.Before(ended.Add(r.limits.Retention)) {
			continue
		}
		for cardID := range j.cards {
			delete(s.cards, cardID)
		}
		for _, c := range s.conversations {
			delete(c.jobs, id)
		}
		delete(s.jobs, id)
		purged++
	}
	return purged, nil
}

func key(tag string, parts ...string) string {
	var b strings.Builder
	b.WriteString(tag)
	for _, part := range parts {
		b.WriteString(strconv.Itoa(len(part)))
		b.WriteByte(':')
		b.WriteString(part)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func inputParts(input []commands.Value) []string {
	sorted := append([]commands.Value(nil), input...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	var parts []string
	for _, v := range sorted {
		parts = append(parts, v.Name, v.Value)
	}
	return parts
}

func ownerAnswer(answer gate.Answer, workspace, person string) bool {
	return answer.User() == person && answer.Workspace() == workspace && answer.Kind() == identity.Human
}
