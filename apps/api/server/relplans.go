package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const (
	relationshipsTarget = "relationships"
	closedPlansKept     = 200
	planMembersShown    = 50
	plansShown          = 50
)

var errPlanPartial = errors.New("relationship plan stopped after a change")

type planChange struct {
	Kind   string `json:"kind"`
	Member string `json:"member"`
	Action string `json:"action"`
	Target string `json:"target"`
	From   string `json:"from"`
}

type planMeta struct {
	workspace string
	id        string
	proposer  string
	created   time.Time
	reason    string
	change    planChange
	closed    string
	activated uint64
	seq       uint64
}

type planStore struct {
	byKey  map[string]*planMeta
	seq    uint64
	impact func(*planMeta) (planImpact, error)
}

const (
	impactComputed    = "computed"
	impactNotComputed = "not_computed"
	impactUnknown     = "unknown"
)

type planImpact struct {
	Money       string `json:"money"`
	Publication string `json:"publication"`
	ERP         string `json:"erp"`
}

func impactAll(state string) planImpact {
	return planImpact{Money: state, Publication: state, ERP: state}
}

func (p planImpact) computed() bool {
	return p.Money == impactComputed && p.Publication == impactComputed && p.ERP == impactComputed
}

func (s *Server) impactOf(m *planMeta) (planImpact, error) {
	if s.plans.impact == nil {
		return impactAll(impactNotComputed), nil
	}
	return s.plans.impact(m)
}

func newPlanStore() *planStore { return &planStore{byKey: map[string]*planMeta{}} }

func planKey(workspace, id string) string { return workspace + "\x00" + id }

func (p *planStore) put(m *planMeta) {
	p.seq++
	m.seq = p.seq
	p.byKey[planKey(m.workspace, m.id)] = m
}

func (p *planStore) get(workspace, id string) *planMeta { return p.byKey[planKey(workspace, id)] }

func (s *Server) planState(m *planMeta) relationships.PlanInfo {
	info, _ := s.rels.Plan(m.workspace, m.id)
	return info
}

func (s *Server) prunePlans() {
	closed := []*planMeta{}
	for _, m := range s.plans.byKey {
		if state := s.planState(m).State; state != relationships.PlanDraft && state != relationships.PlanSealed {
			closed = append(closed, m)
		}
	}
	if len(closed) <= closedPlansKept {
		return
	}
	sort.Slice(closed, func(a, b int) bool { return closed[a].seq > closed[b].seq })
	for _, m := range closed[closedPlansKept:] {
		delete(s.plans.byKey, planKey(m.workspace, m.id))
	}
}

func (s *Server) activeRevision(ws string) (uint64, error) {
	if s.faulty("plan-active") {
		return 0, relationships.ErrInvalid
	}
	return s.rels.Active(ws)
}

func (s *Server) timelinesPage(ws string, revision uint64, kind relationships.Kind, after string, limit int) ([]relationships.Timeline, error) {
	if s.faulty("plan-timelines") {
		return nil, relationships.ErrInvalid
	}
	return s.rels.Timelines(ws, revision, kind, after, limit)
}

func today(now time.Time) relationships.Day {
	u := now.UTC()
	d, _ := relationships.NewDay(u.Year(), int(u.Month()), u.Day())
	return d
}

func (s *Server) timelineOf(ws string, revision uint64, kind relationships.Kind, member string) ([]relationships.Entry, error) {
	after := ""
	limit := RelationshipLimits().PageSize
	for {
		page, err := s.timelinesPage(ws, revision, kind, after, limit)
		if err != nil {
			return nil, err
		}
		for _, line := range page {
			if line.Member == member {
				return line.Entries, nil
			}
			if line.Member > member {
				return nil, nil
			}
		}
		if len(page) < limit {
			return nil, nil
		}
		after = page[len(page)-1].Member
	}
}

func (s *Server) knownIn(ws string, revision uint64, check func(relationships.Kind, relationships.Timeline) bool, kinds ...relationships.Kind) (bool, error) {
	limit := RelationshipLimits().PageSize
	for _, kind := range kinds {
		after := ""
		for {
			page, err := s.timelinesPage(ws, revision, kind, after, limit)
			if err != nil {
				return false, err
			}
			for _, line := range page {
				if check(kind, line) {
					return true, nil
				}
			}
			if len(page) < limit {
				break
			}
			after = page[len(page)-1].Member
		}
	}
	return false, nil
}

func validChange(c planChange) (relationships.Kind, relationships.Day, string) {
	kind := relationships.Kind(c.Kind)
	exclusive := kind != relationships.ViewGroup
	if (c.Action == "set" || c.Action == "end") != exclusive {
		return kind, relationships.Day{}, "action"
	}
	if !validWorkspace(c.Member) {
		return kind, relationships.Day{}, "member"
	}
	if (c.Action != "end") != (c.Target != "") || (c.Target != "" && (!validWorkspace(c.Target) || c.Target == c.Member)) {
		return kind, relationships.Day{}, "target"
	}
	day, ok := parseDay(c.From)
	if !ok {
		return kind, day, "from"
	}
	return kind, day, ""
}

func refuse(code commands.Code, fields ...string) error { return commands.Fail(code, fields...) }

func (s *Server) checkIDs(ws string, active uint64, kind relationships.Kind, c planChange) error {
	parties, err := s.orgs.Parties(ws)
	if err != nil {
		return err
	}
	party := map[string]organizations.Party{}
	for _, p := range parties {
		party[p.ID] = p
	}
	if _, isParty := party[c.Member]; !isParty {
		found, err := s.knownIn(ws, active, func(_ relationships.Kind, line relationships.Timeline) bool { return line.Member == c.Member }, relationKinds...)
		if err != nil {
			return err
		}
		if !found {
			return refuse(commands.InvalidRequest, "member")
		}
	}
	if c.Target == "" {
		return nil
	}
	switch kind {
	case relationships.Hierarchy, relationships.Payer:
		p, ok := party[c.Target]
		if !ok || p.State != organizations.Active || (kind == relationships.Payer && p.Kind != organizations.LegalEntity) {
			return refuse(commands.InvalidRequest, "target")
		}
		return nil
	}
	found, err := s.knownIn(ws, active, func(_ relationships.Kind, line relationships.Timeline) bool {
		for _, e := range line.Entries {
			if e.Target == c.Target {
				return true
			}
		}
		return false
	}, kind)
	if err != nil {
		return err
	}
	if !found {
		return refuse(commands.InvalidRequest, "target")
	}
	return nil
}

func (s *Server) discardStale(ws string, active uint64, now time.Time) {
	for _, m := range s.plans.byKey {
		if m.workspace != ws {
			continue
		}
		info := s.planState(m)
		if info.State != relationships.PlanSealed {
			continue
		}
		from, _ := parseDay(m.change.From)
		reason := ""
		switch {
		case info.Base != active:
			reason = "discarded_stale"
		case from.Before(today(now)):
			reason = "discarded_expired"
		}
		if reason != "" && s.rels.Discard(ws, m.id) == nil {
			m.closed = reason
		}
	}
}

type relationshipCommand struct {
	s    *Server
	kind string
}

func (h relationshipCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	if h.kind == "propose" {
		if target != relationshipsTarget {
			return commands.Target{}, commands.NotFound
		}
		return commands.Target{Revision: s.Revision()}, commands.Found
	}
	if s.plans.get(workspace, target) == nil {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (h relationshipCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	var state string
	var fields map[string]string
	var err error
	switch h.kind {
	case "propose":
		state, fields, err = h.propose(ctx)
	case "activate":
		state, fields, err = h.activate(ctx)
	default:
		state, err = h.discard(ctx)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	s.prunePlans()
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: state, Fields: fields}, nil
}

func (h relationshipCommand) propose(ctx commands.Context) (string, map[string]string, error) {
	s := h.s
	ws := s.bootWorkspace
	now := s.clock()
	c := planChange{Kind: ctx.Fields["kind"], Member: ctx.Fields["member"], Action: ctx.Fields["action"], Target: ctx.Fields["target"], From: ctx.Fields["from"]}
	kind, from, bad := validChange(c)
	if bad != "" {
		return "", nil, refuse(commands.InvalidRequest, bad)
	}
	if from.Before(today(now)) {
		return "", nil, refuse(commands.InvalidRequest, "from")
	}
	raw, pinned := ctx.Fields["base"]
	base, err := strconv.ParseUint(raw, 10, 64)
	if pinned && (err != nil || strconv.FormatUint(base, 10) != raw) {
		return "", nil, refuse(commands.InvalidRequest, "base")
	}
	active, err := s.activeRevision(ws)
	if err != nil {
		return "", nil, refuse(commands.DependencyUnavailable)
	}
	if pinned && base != active {
		return "", nil, refuse(commands.RevisionConflict)
	}
	if err := s.checkIDs(ws, active, kind, c); err != nil {
		if errors.As(err, new(*commands.Failure)) {
			return "", nil, err
		}
		return "", nil, refuse(commands.DependencyUnavailable)
	}
	entries, err := s.timelineOf(ws, active, kind, c.Member)
	if err != nil {
		return "", nil, refuse(commands.DependencyUnavailable)
	}
	built := buildTimeline(entries, c.Action, c.Target, from)
	if len(built) > RelationshipLimits().EntriesPerTimeline {
		return "", nil, refuse(commands.InvalidRequest, "action")
	}
	s.discardStale(ws, active, now)
	id := "plan-" + hex.EncodeToString(random(8))
	if s.faulty("plan-draft") {
		err = relationships.ErrConflict
	} else {
		err = s.rels.Draft(ws, id, active)
	}
	if err != nil {
		if errors.Is(err, relationships.ErrLimit) {
			return "", nil, refuse(commands.PrerequisiteFailed)
		}
		return "", nil, refuse(commands.InternalError)
	}
	var sealErr error
	if s.faulty("plan-chunk") {
		sealErr = relationships.ErrInvalid
	} else {
		sealErr = s.rels.AddChunk(ws, id, 0, []relationships.Timeline{{Kind: kind, Member: c.Member, Entries: built}})
	}
	if sealErr == nil {
		_, sealErr = s.rels.Seal(ws, id, 1)
	}
	if sealErr != nil && !errors.Is(sealErr, relationships.ErrRejected) {
		_ = s.rels.Discard(ws, id)
		return "", nil, errPlanPartial
	}
	s.plans.put(&planMeta{workspace: ws, id: id, proposer: ctx.User, created: now, reason: ctx.Rationale, change: c})
	return string(s.planState(s.plans.get(ws, id)).State), map[string]string{"plan": id}, nil
}

func (h relationshipCommand) activate(ctx commands.Context) (string, map[string]string, error) {
	s := h.s
	ws := s.bootWorkspace
	m := s.plans.get(ws, ctx.Target)
	info := s.planState(m)
	from, _ := parseDay(m.change.From)
	active, err := s.activeRevision(ws)
	switch {
	case err != nil:
		return "", nil, refuse(commands.DependencyUnavailable)
	case info.State != relationships.PlanSealed || info.Base != active || from.Before(today(s.clock())):
		return "", nil, refuse(commands.InvalidStage)
	}
	impact, impactErr := s.impactOf(m)
	switch {
	case impactErr != nil:
		return "", nil, refuse(commands.DependencyUnavailable)
	case !impact.computed():
		return "", nil, refuse(commands.PrerequisiteFailed)
	case m.proposer == "" || m.proposer == ctx.User:
		return "", nil, refuse(commands.PrerequisiteFailed)
	}
	var number uint64
	if s.faulty("plan-activate") {
		err = relationships.ErrConflict
	} else {
		number, err = s.rels.Activate(ws, m.id, info.Digest, active)
	}
	if err != nil {
		return "", nil, refuse(commands.InvalidStage)
	}
	m.activated = number
	return "activated", map[string]string{"revision": strconv.FormatUint(number, 10)}, nil
}

func (h relationshipCommand) discard(ctx commands.Context) (string, error) {
	s := h.s
	m := s.plans.get(s.bootWorkspace, ctx.Target)
	if err := s.rels.Discard(m.workspace, m.id); err != nil {
		return "", refuse(commands.InvalidStage)
	}
	m.closed = "discarded"
	return "discarded", nil
}

func (s *Server) relationshipDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	kinds := []string{}
	for _, k := range relationKinds {
		kinds = append(kinds, string(k))
	}
	fields := []commands.Field{
		{Name: "kind", Type: commands.Enum, Required: true, Values: kinds},
		{Name: "member", Type: commands.ID, Required: true},
		{Name: "action", Type: commands.Enum, Required: true, Values: []string{"set", "end", "join", "leave"}},
		{Name: "target", Type: commands.Text},
		{Name: "from", Type: commands.Text, Required: true},
		{Name: "base", Type: commands.Integer},
	}
	define := func(id, kind string, f []commands.Field) commands.Definition {
		return commands.Definition{ID: id, Permission: permission, Mutating: true, Rationale: true, Fields: f, Handler: relationshipCommand{s: s, kind: kind}}
	}
	return []commands.Definition{
		define("relationship.propose", "propose", fields),
		define("relationship.activate", "activate", nil),
		define("relationship.discard", "discard", nil),
	}
}

type kindRow struct {
	Kind          string `json:"kind"`
	Changed       int    `json:"changed"`
	EntriesBefore int    `json:"entries_before"`
	EntriesAfter  int    `json:"entries_after"`
	Earliest      string `json:"earliest"`
}

type violationRow struct {
	Kind   string `json:"kind"`
	Member string `json:"member"`
	Rule   string `json:"rule"`
}

type rangeRow struct {
	From  string `json:"from"`
	Until string `json:"until"`
}

type changedRow struct {
	Member  string     `json:"member"`
	Name    string     `json:"name"`
	Differs []rangeRow `json:"differs"`
	Before  []string   `json:"before"`
	After   []string   `json:"after"`
}

type planRow struct {
	ID            string         `json:"id"`
	State         string         `json:"state"`
	Closed        string         `json:"closed"`
	Created       string         `json:"created"`
	Proposer      string         `json:"proposer"`
	ProposerLogin string         `json:"proposer_login"`
	Reason        string         `json:"reason"`
	Change        planChange     `json:"change"`
	MemberName    string         `json:"member_name"`
	TargetName    string         `json:"target_name"`
	Base          uint64         `json:"base"`
	Stale         bool           `json:"stale"`
	Activated     uint64         `json:"activated_revision"`
	Kinds         []kindRow      `json:"kinds"`
	Violations    []violationRow `json:"violations"`
	Truncated     bool           `json:"truncated"`
	Members       []changedRow   `json:"members"`
	Impact        planImpact     `json:"impact"`
	CanActivate   bool           `json:"can_activate"`
	Blocked       string         `json:"blocked"`
}

type plansPage struct {
	Revision   string            `json:"revision"`
	Active     uint64            `json:"active_revision"`
	OpenPlans  int               `json:"open_plans"`
	OpenLimit  int               `json:"open_limit"`
	Impact     map[string]string `json:"impact"`
	Plans      []planRow         `json:"plans"`
	ReaderUser string            `json:"you"`
}

func (s *Server) planRow(m *planMeta, active uint64, reader string, now time.Time) planRow {
	info := s.planState(m)
	row := planRow{ID: m.id, State: string(info.State), Closed: m.closed, Created: stamp(m.created), Proposer: m.proposer, ProposerLogin: s.loginOf(m.proposer),
		Reason: m.reason, Change: m.change, MemberName: orgName(m.change.Member), TargetName: orgName(m.change.Target), Base: info.Base,
		Activated: m.activated, Kinds: []kindRow{}, Violations: []violationRow{}, Truncated: info.Truncated, Members: []changedRow{}}
	kinds := []string{}
	for k := range info.Kinds {
		kinds = append(kinds, string(k))
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		sum := info.Kinds[relationships.Kind(k)]
		kr := kindRow{Kind: k, Changed: sum.Changed, EntriesBefore: sum.EntriesBefore, EntriesAfter: sum.EntriesAfter}
		if sum.HasEarliest {
			kr.Earliest = sum.Earliest.ISO()
		}
		row.Kinds = append(row.Kinds, kr)
		if info.State == relationships.PlanSealed || info.State == relationships.PlanActivated {
			changes, _ := s.rels.ChangedMembers(m.workspace, m.id, relationships.Kind(k), "", planMembersShown)
			for _, c := range changes {
				cr := changedRow{Member: c.Member, Name: orgName(c.Member), Differs: []rangeRow{}, Before: append([]string{}, c.Before...), After: append([]string{}, c.After...)}
				for _, iv := range c.Differs {
					rr := rangeRow{From: iv.Start().ISO()}
					if end, bounded := iv.End(); bounded {
						rr.Until = end.ISO()
					}
					cr.Differs = append(cr.Differs, rr)
				}
				row.Members = append(row.Members, cr)
			}
		}
	}
	for _, v := range info.Violations {
		row.Violations = append(row.Violations, violationRow{Kind: string(v.Kind), Member: v.Member, Rule: string(v.Rule)})
	}
	impact, impactErr := s.impactOf(m)
	if impactErr != nil {
		impact = impactAll(impactUnknown)
	}
	row.Impact = impact
	from, _ := parseDay(m.change.From)
	if info.State == relationships.PlanSealed {
		row.Stale = info.Base != active
		switch {
		case row.Stale:
			row.Blocked = "stale"
		case from.Before(today(now)):
			row.Blocked = "effective_date_passed"
		case impactErr != nil:
			row.Blocked = "impact_unavailable"
		case !impact.computed():
			row.Blocked = "impact_not_computed"
		case m.proposer == "" || m.proposer == reader:
			row.Blocked = "own_proposal"
		default:
			row.CanActivate = true
		}
	}
	return row
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	ws := answer.Workspace()
	active, err := s.activeRevision(ws)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	now := s.clock()
	page := plansPage{Revision: s.Revision(), Active: active, OpenLimit: RelationshipLimits().OpenPlans, Plans: []planRow{}, ReaderUser: answer.User(),
		Impact: map[string]string{"money": "not_computed", "publication": "not_computed", "erp": "not_computed"}}
	mine := []*planMeta{}
	for _, m := range s.plans.byKey {
		if m.workspace == ws {
			mine = append(mine, m)
			if state := s.planState(m).State; state == relationships.PlanDraft || state == relationships.PlanSealed {
				page.OpenPlans++
			}
		}
	}
	sort.Slice(mine, func(a, b int) bool { return mine[a].seq > mine[b].seq })
	shown := []*planMeta{}
	for _, m := range mine {
		if state := s.planState(m).State; state == relationships.PlanDraft || state == relationships.PlanSealed {
			shown = append(shown, m)
		}
	}
	for _, m := range mine {
		if state := s.planState(m).State; state != relationships.PlanDraft && state != relationships.PlanSealed && len(shown) < plansShown {
			shown = append(shown, m)
		}
	}
	sort.Slice(shown, func(a, b int) bool { return shown[a].seq > shown[b].seq })
	mine = shown
	for _, m := range mine {
		page.Plans = append(page.Plans, s.planRow(m, active, answer.User(), now))
	}
	writeJSON(w, http.StatusOK, request, page)
}
