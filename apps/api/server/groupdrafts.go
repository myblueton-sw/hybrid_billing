package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const (
	viewGroupDraftsTarget = "view-group-drafts"
	draftMembersShown     = 50
	draftsPageSize        = 50
	draftOpen             = "open"
	draftProposed         = "proposed"
	draftActivated        = "activated"
	draftDiscarded        = "discarded"
)

type groupDraft struct {
	workspace string
	id        string
	group     string
	newGroup  bool
	name      string
	members   []string
	count     int
	author    string
	editors   []string
	created   time.Time
	updated   time.Time
	discarded time.Time
	plans     []string
	activated uint64
	pruned    bool
	seq       uint64
	closedSeq uint64
}

type groupLabel struct {
	name     string
	revision uint64
}

type draftStore struct {
	byKey   map[string]*groupDraft
	labels  map[string]groupLabel
	seq     uint64
	groupID func() string
}

func newDraftStore() *draftStore {
	return &draftStore{byKey: map[string]*groupDraft{}, labels: map[string]groupLabel{}, groupID: func() string { return "group-" + hex.EncodeToString(random(8)) }}
}

func (d *draftStore) put(g *groupDraft) {
	d.seq++
	g.seq = d.seq
	d.byKey[planKey(g.workspace, g.id)] = g
}

func (d *draftStore) get(workspace, id string) *groupDraft { return d.byKey[planKey(workspace, id)] }

func (d *draftStore) close(g *groupDraft) {
	d.seq++
	g.closedSeq = d.seq
}

func (s *Server) label(ws, id string) string {
	if name := orgName(id); name != "" {
		return name
	}
	return s.drafts.labels[planKey(ws, id)].name
}

func (s *Server) labelMatches(ws, id, term string) bool {
	name := s.drafts.labels[planKey(ws, id)].name
	return name != "" && strings.Contains(strings.ToLower(name), strings.ToLower(term))
}

func (s *Server) labelled(ws string, rows []namedRow) []namedRow {
	for i := range rows {
		rows[i].Name = s.label(ws, rows[i].ID)
	}
	return rows
}

func (s *Server) labelledHistory(ws string, rows []historyRow) []historyRow {
	for i := range rows {
		rows[i].Name = s.label(ws, rows[i].Target)
	}
	return rows
}

func (s *Server) latestPlan(g *groupDraft) (relationships.PlanInfo, bool, error) {
	if len(g.plans) == 0 {
		return relationships.PlanInfo{}, false, nil
	}
	info, err := s.rels.Plan(g.workspace, g.plans[len(g.plans)-1])
	if s.faulty("draft-plan") {
		err = relationships.ErrNotFound
	}
	return info, true, err
}

func (s *Server) draftState(g *groupDraft) (string, error) {
	if !g.discarded.IsZero() {
		return draftDiscarded, nil
	}
	info, has, err := s.latestPlan(g)
	switch {
	case err != nil:
		return "", err
	case has && (info.State == relationships.PlanDraft || info.State == relationships.PlanSealed):
		return draftProposed, nil
	case has && info.State == relationships.PlanActivated:
		return draftActivated, nil
	}
	return draftOpen, nil
}

func (s *Server) workspaceDrafts(ws string) []*groupDraft {
	out := []*groupDraft{}
	for _, g := range s.drafts.byKey {
		if g.workspace == ws {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].seq > out[b].seq })
	return out
}

func (s *Server) openDrafts(ws string) (int, error) {
	n := 0
	for _, g := range s.workspaceDrafts(ws) {
		state, err := s.draftState(g)
		if err != nil {
			return 0, err
		}
		if state == draftOpen || state == draftProposed {
			n++
		}
	}
	return n, nil
}

func (s *Server) draftActivated(ws, id string, number uint64) {
	g := s.drafts.get(ws, id)
	g.activated = number
	s.drafts.close(g)
	key := planKey(ws, g.group)
	if _, named := s.drafts.labels[key]; g.newGroup && !named {
		s.drafts.labels[key] = groupLabel{name: g.name, revision: number}
	}
	s.pruneDrafts()
}

func (s *Server) pruneDrafts() {
	discarded, activated := []*groupDraft{}, []*groupDraft{}
	for _, g := range s.drafts.byKey {
		switch {
		case !g.discarded.IsZero():
			discarded = append(discarded, g)
		case g.activated != 0 && !g.pruned:
			activated = append(activated, g)
		}
	}
	newest := func(list []*groupDraft) {
		sort.Slice(list, func(a, b int) bool { return list[a].closedSeq > list[b].closedSeq })
	}
	newest(discarded)
	for _, g := range discarded[min(len(discarded), s.cfg.Org.DraftsKept):] {
		delete(s.drafts.byKey, planKey(g.workspace, g.id))
	}
	newest(activated)
	for _, g := range activated[min(len(activated), s.cfg.Org.ActivatedKept):] {
		g.members, g.name, g.pruned = nil, "", true
	}
}

type draftPlanRow struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Closed  string `json:"closed"`
	Blocked string `json:"blocked"`
	Rule    string `json:"rule"`
}

type draftRow struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Group       string        `json:"group"`
	GroupLabel  string        `json:"group_label"`
	NewGroup    bool          `json:"new_group"`
	MemberCount int           `json:"member_count"`
	Members     []namedRow    `json:"members"`
	Author      string        `json:"author"`
	AuthorLogin string        `json:"author_login"`
	Created     string        `json:"created"`
	Updated     string        `json:"updated"`
	Discarded   string        `json:"discarded"`
	State       string        `json:"state"`
	Activated   uint64        `json:"activated_revision"`
	Pruned      bool          `json:"pruned"`
	Plan        *draftPlanRow `json:"plan"`
}

type draftsPage struct {
	Revision  string       `json:"revision"`
	Active    uint64       `json:"active_revision"`
	Settings  []settingRow `json:"settings"`
	Open      int          `json:"open"`
	OpenLimit int          `json:"open_limit"`
	Drafts    []draftRow   `json:"drafts"`
	Next      string       `json:"next"`
}

func (s *Server) draftRow(g *groupDraft, state string, active uint64, reader string, now time.Time) draftRow {
	row := draftRow{ID: g.id, Name: g.name, Group: g.group, GroupLabel: s.label(g.workspace, g.group), NewGroup: g.newGroup, MemberCount: g.count, Members: []namedRow{},
		Author: g.author, AuthorLogin: s.loginOf(g.author), Created: stamp(g.created), Updated: stamp(g.updated), State: state, Activated: g.activated, Pruned: g.pruned}
	if !g.discarded.IsZero() {
		row.Discarded = stamp(g.discarded)
	}
	if row.GroupLabel == "" {
		row.GroupLabel = g.group
	}
	for _, id := range g.members[:min(len(g.members), draftMembersShown)] {
		row.Members = append(row.Members, namedRow{ID: id, Name: s.label(g.workspace, id)})
	}
	info, has, _ := s.latestPlan(g)
	if !has {
		return row
	}
	plan := &draftPlanRow{ID: g.plans[len(g.plans)-1], State: string(info.State)}
	if len(info.Violations) > 0 {
		plan.Rule = string(info.Violations[0].Rule)
	}
	if m := s.plans.get(g.workspace, plan.ID); m != nil {
		plan.Closed = m.closed
		if info.State == relationships.PlanSealed {
			impact, impactErr := s.impactOf(m)
			plan.Blocked = blockedOf(m, info, active, impact, impactErr, reader, now)
		}
	}
	row.Plan = plan
	return row
}

func draftLimit(raw string, max int) (int, bool) {
	if raw == "" {
		return draftsPageSize, true
	}
	n, err := strconv.Atoi(raw)
	return n, err == nil && n >= 1 && n <= max && strconv.Itoa(n) == raw
}

func (s *Server) readDrafts(ws, reader, after string, limit int) (draftsPage, error) {
	active, err := s.activeRevision(ws)
	if err != nil {
		return draftsPage{}, err
	}
	now := s.clock()
	page := draftsPage{Revision: s.Revision(), Active: active, Settings: s.orgSettingRows(), OpenLimit: s.cfg.Org.DraftsOpen, Drafts: []draftRow{}}
	first, rest := []*groupDraft{}, []*groupDraft{}
	states := map[*groupDraft]string{}
	for _, g := range s.workspaceDrafts(ws) {
		state, err := s.draftState(g)
		if err != nil {
			return draftsPage{}, err
		}
		states[g] = state
		if state == draftOpen || state == draftProposed {
			page.Open++
			first = append(first, g)
		} else {
			rest = append(rest, g)
		}
	}
	ordered := append(first, rest...)
	start := 0
	if after != "" {
		start = -1
		for i, g := range ordered {
			if g.id == after {
				start = i + 1
			}
		}
		if start < 0 {
			return draftsPage{}, errInvalidQuery
		}
	}
	shown := ordered[start:min(len(ordered), start+limit)]
	for _, g := range shown {
		page.Drafts = append(page.Drafts, s.draftRow(g, states[g], active, reader, now))
	}
	if start+limit < len(ordered) {
		page.Next = shown[len(shown)-1].id
	}
	return page, nil
}

func (s *Server) listDrafts(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	after := q.Get("after")
	limit, valid := draftLimit(q.Get("limit"), draftsPageSize)
	if !valid || (after != "" && !validWorkspace(after)) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page, err := s.readDrafts(answer.Workspace(), answer.User(), after, limit)
	switch {
	case errors.Is(err, errInvalidQuery):
		failCode(w, commands.InvalidRequest, request)
	case err != nil:
		failCode(w, commands.DependencyUnavailable, request)
	default:
		writeJSON(w, http.StatusOK, request, page)
	}
}

type draftMembersPage struct {
	Revision string     `json:"revision"`
	Draft    string     `json:"draft"`
	Count    int        `json:"member_count"`
	Pruned   bool       `json:"pruned"`
	Members  []namedRow `json:"members"`
	Next     string     `json:"next"`
}

func (s *Server) listDraftMembers(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	ws := answer.Workspace()
	g := s.drafts.get(ws, r.PathValue("id"))
	if g == nil {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	q := r.URL.Query()
	after := q.Get("after")
	limit, valid := draftLimit(q.Get("limit"), RelationshipLimits().PageSize-1)
	if !valid || (after != "" && !validWorkspace(after)) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page := draftMembersPage{Revision: s.Revision(), Draft: g.id, Count: g.count, Pruned: g.pruned, Members: []namedRow{}}
	start := sort.Search(len(g.members), func(i int) bool { return g.members[i] > after })
	shown := g.members[start:min(len(g.members), start+limit)]
	for _, id := range shown {
		page.Members = append(page.Members, namedRow{ID: id, Name: s.label(ws, id)})
	}
	if start+limit < len(g.members) {
		page.Next = shown[len(shown)-1]
	}
	writeJSON(w, http.StatusOK, request, page)
}
