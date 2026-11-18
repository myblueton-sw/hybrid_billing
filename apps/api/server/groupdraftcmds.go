package server

import (
	"encoding/hex"
	"errors"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const (
	draftMembersPerCall = 31
	draftNameRunes      = 80
)

func draftName(raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > draftNameRunes {
		return "", false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	return name, true
}

func memberLines(raw, field string) ([]string, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) > draftMembersPerCall {
		return nil, refuse(commands.InvalidRequest, field)
	}
	seen := map[string]bool{}
	for _, id := range lines {
		if !validWorkspace(id) || seen[id] {
			return nil, refuse(commands.InvalidRequest, field)
		}
		seen[id] = true
	}
	return lines, nil
}

func (s *Server) joinLines(ws string, active uint64, group string, members []string, day relationships.Day, field string) ([]relationships.Timeline, error) {
	parties, err := s.orgs.Parties(ws)
	if s.faulty("draft-parties") {
		err = errors.New("parties unreadable")
	}
	if err != nil {
		return nil, refuse(commands.DependencyUnavailable)
	}
	party := map[string]bool{}
	for _, p := range parties {
		party[p.ID] = true
	}
	number := active
	if s.faulty("draft-roles") {
		number++
	}
	out := []relationships.Timeline{}
	for _, member := range members {
		if party[member] || member == group {
			return nil, refuse(commands.InvalidRequest, field)
		}
		listed, target, entries := false, false, []relationships.Entry(nil)
		var errs []error
		for _, kind := range relationKinds {
			is, err := s.rels.IsTarget(ws, number, kind, member)
			line, has, lineErr := s.rels.TimelineOf(ws, number, kind, member)
			errs = append(errs, err, lineErr)
			target, listed = target || is, listed || has
			if kind == relationships.ViewGroup {
				entries = line.Entries
			}
		}
		if err := errors.Join(errs...); err != nil {
			return nil, refuse(commands.DependencyUnavailable)
		}
		built := buildTimeline(entries, "join", group, day)
		if target || !listed || len(built) > s.relLimits.EntriesPerTimeline {
			return nil, refuse(commands.InvalidRequest, field)
		}
		out = append(out, relationships.Timeline{Kind: relationships.ViewGroup, Member: member, Entries: built})
	}
	return out, nil
}

type draftCommand struct {
	s    *Server
	kind string
}

func (h draftCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	if (h.kind == "draft") != (target == viewGroupDraftsTarget) || (h.kind != "draft" && s.drafts.get(workspace, target) == nil) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (h draftCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	var state string
	var fields map[string]string
	var err error
	switch h.kind {
	case "draft":
		state, fields, err = h.draft(ctx)
	case "edit":
		state, err = h.edit(ctx)
	case "discard":
		state, err = h.discard(ctx)
	default:
		state, fields, err = h.propose(ctx)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	s.pruneDrafts()
	s.prunePlans()
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: state, Fields: fields}, nil
}

func (s *Server) readyDraft(ws string) (uint64, error) {
	active, err := s.activeRevision(ws)
	if err != nil {
		return 0, refuse(commands.DependencyUnavailable)
	}
	return active, nil
}

func (h draftCommand) draft(ctx commands.Context) (string, map[string]string, error) {
	s := h.s
	ws := s.bootWorkspace
	group, existing := ctx.Fields["group"]
	rawName, named := ctx.Fields["name"]
	name, nameOK := draftName(rawName)
	if existing == named || (named && !nameOK) {
		return "", nil, refuse(commands.InvalidRequest, "name")
	}
	members, err := memberLines(ctx.Fields["members"], "members")
	if err != nil {
		return "", nil, err
	}
	if len(members) > s.cfg.Org.DraftMembers {
		return "", nil, refuse(commands.PrerequisiteFailed)
	}
	open, err := s.openDrafts(ws)
	if err != nil {
		return "", nil, refuse(commands.DependencyUnavailable)
	}
	if open >= s.cfg.Org.DraftsOpen {
		return "", nil, refuse(commands.PrerequisiteFailed)
	}
	active, err := s.readyDraft(ws)
	if err != nil {
		return "", nil, err
	}
	if existing {
		is, err := s.rels.IsTarget(ws, active, relationships.ViewGroup, group)
		if err != nil || !is {
			return "", nil, refuse(commands.InvalidRequest, "group")
		}
	} else {
		group = s.drafts.groupID()
	}
	if _, err := s.joinLines(ws, active, group, members, today(s.clock()), "members"); err != nil {
		return "", nil, err
	}
	now := s.clock()
	sort.Strings(members)
	g := &groupDraft{workspace: ws, id: "vgd-" + hex.EncodeToString(random(8)), group: group, newGroup: !existing, name: name, members: members, count: len(members),
		author: ctx.User, editors: []string{ctx.User}, created: now, updated: now}
	s.drafts.put(g)
	return draftOpen, map[string]string{"draft": g.id, "group": group}, nil
}

func (s *Server) openDraft(ctx commands.Context) (*groupDraft, error) {
	g := s.drafts.get(s.bootWorkspace, ctx.Target)
	state, err := s.draftState(g)
	if err != nil {
		return nil, refuse(commands.DependencyUnavailable)
	}
	if state != draftOpen {
		return nil, refuse(commands.InvalidStage)
	}
	return g, nil
}

func (h draftCommand) edit(ctx commands.Context) (string, error) {
	s := h.s
	g, err := s.openDraft(ctx)
	if err != nil {
		return "", err
	}
	rawName, named := ctx.Fields["name"]
	rawAdd, adding := ctx.Fields["add"]
	rawRemove, removing := ctx.Fields["remove"]
	if !named && !adding && !removing {
		return "", refuse(commands.InvalidRequest)
	}
	name, nameOK := draftName(rawName)
	if named && (!g.newGroup || !nameOK) {
		return "", refuse(commands.InvalidRequest, "name")
	}
	current := map[string]bool{}
	for _, id := range g.members {
		current[id] = true
	}
	add, remove := []string{}, []string{}
	if adding {
		if add, err = memberLines(rawAdd, "add"); err != nil {
			return "", err
		}
	}
	if removing {
		if remove, err = memberLines(rawRemove, "remove"); err != nil {
			return "", err
		}
	}
	for _, id := range add {
		if current[id] {
			return "", refuse(commands.InvalidRequest, "add")
		}
	}
	for _, id := range remove {
		if !current[id] {
			return "", refuse(commands.InvalidRequest, "remove")
		}
	}
	size := len(g.members) + len(add) - len(remove)
	switch {
	case size < 1:
		return "", refuse(commands.InvalidRequest, "remove")
	case size > s.cfg.Org.DraftMembers:
		return "", refuse(commands.PrerequisiteFailed)
	}
	if len(add) > 0 {
		active, err := s.readyDraft(g.workspace)
		if err != nil {
			return "", err
		}
		if _, err := s.joinLines(g.workspace, active, g.group, add, today(s.clock()), "add"); err != nil {
			return "", err
		}
	}
	for _, id := range remove {
		delete(current, id)
	}
	for _, id := range add {
		current[id] = true
	}
	members := make([]string, 0, len(current))
	for id := range current {
		members = append(members, id)
	}
	sort.Strings(members)
	if named {
		g.name = name
	}
	g.members, g.count, g.updated = members, len(members), s.clock()
	if !slices.Contains(g.editors, ctx.User) {
		g.editors = append(g.editors, ctx.User)
	}
	return draftOpen, nil
}

func (h draftCommand) discard(ctx commands.Context) (string, error) {
	s := h.s
	g, err := s.openDraft(ctx)
	if err != nil {
		return "", err
	}
	g.discarded, g.updated = s.clock(), s.clock()
	s.drafts.close(g)
	return draftDiscarded, nil
}

func (h draftCommand) propose(ctx commands.Context) (string, map[string]string, error) {
	s := h.s
	g, err := s.openDraft(ctx)
	if err != nil {
		return "", nil, err
	}
	ws := g.workspace
	now := s.clock()
	from, ok := parseDay(ctx.Fields["from"])
	if !ok || from.Before(today(now)) {
		return "", nil, refuse(commands.InvalidRequest, "from")
	}
	raw := ctx.Fields["base"]
	base, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || strconv.FormatUint(base, 10) != raw {
		return "", nil, refuse(commands.InvalidRequest, "base")
	}
	active, err := s.readyDraft(ws)
	if err != nil {
		return "", nil, err
	}
	if base != active {
		return "", nil, refuse(commands.RevisionConflict)
	}
	if err := s.groupUsable(g, active); err != nil {
		return "", nil, err
	}
	built, err := s.joinLines(ws, active, g.group, g.members, from, "members")
	if err != nil {
		return "", nil, err
	}
	s.discardStale(ws, active, now)
	id, err := s.sealPlan(ws, active, built)
	if err != nil {
		return "", nil, err
	}
	change := planChange{Kind: string(relationships.ViewGroup), Action: "join", Target: g.group, From: ctx.Fields["from"]}
	snapshot := &planDraftRow{ID: g.id, Name: g.name, Group: g.group, NewGroup: g.newGroup, MemberCount: g.count}
	s.plans.put(&planMeta{workspace: ws, id: id, proposer: ctx.User, created: now, reason: ctx.Rationale, change: change, draft: g.id, snapshot: snapshot, editors: slices.Clone(g.editors)})
	g.plans, g.updated = append(g.plans, id), now
	return string(s.planState(s.plans.get(ws, id)).State), map[string]string{"plan": id}, nil
}

func (s *Server) groupUsable(g *groupDraft, active uint64) error {
	for _, kind := range relationKinds {
		is, err := s.rels.IsTarget(g.workspace, active, kind, g.group)
		_, member, lineErr := s.rels.TimelineOf(g.workspace, active, kind, g.group)
		if err := errors.Join(err, lineErr); err != nil {
			return refuse(commands.DependencyUnavailable)
		}
		switch {
		case !g.newGroup && kind == relationships.ViewGroup && !is:
			return refuse(commands.InvalidRequest, "group")
		case g.newGroup && (is || member):
			return refuse(commands.InvalidStage)
		}
	}
	return nil
}

func (s *Server) draftDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	define := func(id, kind string, f []commands.Field) commands.Definition {
		return commands.Definition{ID: id, Permission: permission, Mutating: true, Rationale: true, Fields: f, Handler: draftCommand{s: s, kind: kind}}
	}
	return []commands.Definition{
		define("view_group.draft", "draft", []commands.Field{{Name: "name", Type: commands.Text}, {Name: "group", Type: commands.ID}, {Name: "members", Type: commands.Text, Required: true}}),
		define("view_group.edit", "edit", []commands.Field{{Name: "name", Type: commands.Text}, {Name: "add", Type: commands.Text}, {Name: "remove", Type: commands.Text}}),
		define("view_group.discard_draft", "discard", nil),
		define("view_group.propose", "propose", []commands.Field{{Name: "from", Type: commands.Text, Required: true}, {Name: "base", Type: commands.Integer, Required: true}}),
	}
}
