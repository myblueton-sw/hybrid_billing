package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const (
	childrenSome      = "some"
	childrenNone      = "none"
	childrenUnchecked = "unchecked"
	groupDetached     = "detached"
	phaseParties      = "p"
	phaseOthers       = "m"
)

type treeNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	State    string `json:"state"`
	Source   string `json:"source"`
	Children string `json:"children"`
}

type treePage struct {
	Revision      string     `json:"revision"`
	Relationship  uint64     `json:"relationship_revision"`
	Active        uint64     `json:"active_revision"`
	On            string     `json:"on"`
	Parent        string     `json:"parent"`
	Group         string     `json:"group"`
	Nodes         []treeNode `json:"nodes"`
	Next          string     `json:"next"`
	ScanStopped   bool       `json:"scan_stopped"`
	RegistryEmpty bool       `json:"registry_empty"`
}

type treeQuery struct {
	on       relationships.Day
	parent   string
	group    string
	phase    string
	after    string
	limit    int
	revision uint64
	pinned   bool
}

func (s *Server) treeQuery(r *http.Request) (treeQuery, bool) {
	q := r.URL.Query()
	for name, values := range q {
		switch name {
		case "on", "revision", "parent", "group", "after", "limit":
			if len(values) != 1 {
				return treeQuery{}, false
			}
		default:
			return treeQuery{}, false
		}
	}
	now := s.clock().UTC()
	out := treeQuery{limit: orgDefaultLimit, phase: phaseParties}
	out.on, _ = relationships.NewDay(now.Year(), int(now.Month()), now.Day())
	if q.Has("on") {
		day, ok := parseDay(q.Get("on"))
		if !ok {
			return out, false
		}
		out.on = day
	}
	if q.Has("parent") {
		out.parent = q.Get("parent")
		if !validWorkspace(out.parent) || q.Has("group") {
			return out, false
		}
	}
	if q.Has("group") {
		out.group = q.Get("group")
		if out.group != groupDetached {
			return out, false
		}
	}
	if q.Has("after") {
		phase, id, found := strings.Cut(q.Get("after"), ":")
		if !found || (phase != phaseParties && phase != phaseOthers) || (id != "" && !validWorkspace(id)) || (id == "" && phase == phaseParties) || (phase == phaseOthers && out.group != "") {
			return out, false
		}
		out.phase, out.after = phase, id
	}
	if q.Has("limit") {
		raw := q.Get("limit")
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > RelationshipLimits().PageSize-1 || strconv.Itoa(n) != raw {
			return out, false
		}
		out.limit = n
	}
	if q.Has("revision") {
		raw := q.Get("revision")
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || strconv.FormatUint(n, 10) != raw {
			return out, false
		}
		out.revision, out.pinned = n, true
	}
	return out, true
}

type placement struct {
	party  organizations.Party
	parent string
	source string
	held   bool
}

type treeScan struct {
	s       *Server
	ws      string
	number  uint64
	on      relationships.Day
	parties []placement
	byID    map[string]int
	known   map[string]string
	budget  int
}

var partyRank = map[organizations.PartyKind]int{organizations.LegalEntity: 0, organizations.InternalUnit: 1}

func rankOf(kind organizations.PartyKind) int {
	if rank, known := partyRank[kind]; known {
		return rank
	}
	return len(partyRank)
}

func (c *treeScan) place(parties []organizations.Party) error {
	c.byID = map[string]int{}
	for _, p := range parties {
		at := placement{party: p, parent: p.ParentID, source: "registry"}
		line, member, err := c.s.rels.TimelineOf(c.ws, c.number, relationships.Hierarchy, p.ID)
		if err != nil {
			return err
		}
		if member {
			at.parent, at.source, at.held = "", "none", true
			for _, e := range line.Entries {
				end, bounded := e.Interval.End()
				if !c.on.Before(e.Interval.Start()) && (!bounded || c.on.Before(end)) {
					at.parent, at.source, at.held = e.Target, "relationship", false
				}
			}
		} else if p.ParentID == "" {
			at.source = "none"
		}
		c.parties = append(c.parties, at)
	}
	sort.SliceStable(c.parties, func(a, b int) bool {
		ra, rb := rankOf(c.parties[a].party.Kind), rankOf(c.parties[b].party.Kind)
		return ra < rb || (ra == rb && c.parties[a].party.ID < c.parties[b].party.ID)
	})
	for i, at := range c.parties {
		c.byID[at.party.ID] = i
	}
	return nil
}

func (c *treeScan) partyNode(at placement) treeNode {
	return treeNode{ID: at.party.ID, Name: orgName(at.party.ID), Kind: string(at.party.Kind), State: string(at.party.State), Source: at.source}
}

func (c *treeScan) under(parent string) []placement {
	out := []placement{}
	for _, at := range c.parties {
		if at.parent == parent {
			out = append(out, at)
		}
	}
	return out
}

func (c *treeScan) parentOf(id string) (string, bool, error) {
	if i, party := c.byID[id]; party {
		at := c.parties[i]
		return at.parent, !at.held && at.parent != "", nil
	}
	target, has, err := c.s.rels.TargetOn(c.ws, c.number, relationships.Hierarchy, id, c.on)
	return target, has, err
}

func (c *treeScan) detached() ([]placement, error) {
	const unvisited, onPath, done = 0, 1, 2
	colour := map[string]int{}
	found := map[string]bool{}
	steps, most := 0, (orgCap+1)*(RelationshipLimits().Depth+2)
	for _, start := range c.parties {
		if start.party.Kind != organizations.LegalEntity || start.source != "relationship" {
			continue
		}
		path := []string{}
		for cur := start.party.ID; ; {
			if colour[cur] == done {
				break
			}
			if colour[cur] == onPath {
				for i := len(path) - 1; i >= 0; i-- {
					if at, party := c.byID[path[i]]; party && c.parties[at].party.Kind == organizations.LegalEntity {
						found[path[i]] = true
					}
					if path[i] == cur {
						break
					}
				}
				break
			}
			if steps++; steps > most {
				return nil, errors.New("organization tree walk above its bound")
			}
			colour[cur] = onPath
			path = append(path, cur)
			next, has, err := c.parentOf(cur)
			if err != nil {
				return nil, err
			}
			if !has {
				break
			}
			cur = next
		}
		for _, id := range path {
			colour[id] = done
		}
	}
	out := []placement{}
	for _, at := range c.parties {
		if found[at.party.ID] {
			out = append(out, at)
		}
	}
	return out, nil
}

func (c *treeScan) anyMember(target string) (string, error) {
	if state, known := c.known[target]; known {
		return state, nil
	}
	if c.budget == 0 {
		return childrenUnchecked, nil
	}
	got, err := c.s.rels.MembersOnAfter(c.ws, c.number, relationships.Hierarchy, target, c.on, "", 1, c.budget)
	if err != nil {
		return "", err
	}
	c.budget -= got.Examined
	state := childrenUnchecked
	switch {
	case len(got.Members) > 0:
		state = childrenSome
	case got.Whole:
		state = childrenNone
	}
	if state != childrenUnchecked {
		c.known[target] = state
	}
	return state, nil
}

func (c *treeScan) accounts(parent, cursor string, want int) ([]string, string, bool, error) {
	out := []string{}
	for {
		if c.budget == 0 {
			return out, cursor, true, nil
		}
		got, err := c.s.rels.MembersOnAfter(c.ws, c.number, relationships.Hierarchy, parent, c.on, cursor, want-len(out), c.budget)
		if err != nil {
			return nil, "", false, err
		}
		c.budget -= got.Examined
		cursor = got.Last
		for _, id := range got.Members {
			if _, party := c.byID[id]; !party {
				out = append(out, id)
			}
		}
		if got.Whole || len(out) == want {
			return out, cursor, false, nil
		}
	}
}

func (c *treeScan) unknownRoots(cursor string, want int) ([]string, string, bool, error) {
	size := RelationshipLimits().PageSize
	out := []string{}
	for {
		ids, err := c.s.rels.Targets(c.ws, c.number, relationships.Hierarchy, cursor, size)
		if err != nil {
			return nil, "", false, err
		}
		for _, id := range ids {
			if len(out) == want {
				return out, cursor, false, nil
			}
			if c.budget == 0 {
				return out, cursor, true, nil
			}
			untouched := c.budget == c.s.cfg.Org.FilterScan
			c.budget--
			_, party := c.byID[id]
			_, member, err := c.s.rels.TimelineOf(c.ws, c.number, relationships.Hierarchy, id)
			if err != nil {
				return nil, "", false, err
			}
			if party || member {
				cursor = id
				continue
			}
			state, err := c.anyMember(id)
			if err != nil {
				return nil, "", false, err
			}
			if state == childrenUnchecked && !untouched {
				return out, cursor, true, nil
			}
			cursor = id
			if state != childrenNone {
				out = append(out, id)
			}
		}
		if len(ids) < size {
			return out, "", false, nil
		}
	}
}

func (c *treeScan) children(node *treeNode) error {
	if len(c.under(node.ID)) > 0 {
		node.Children = childrenSome
		return nil
	}
	state, err := c.anyMember(node.ID)
	node.Children = state
	return err
}

func (c *treeScan) page(q treeQuery) ([]treeNode, string, bool, error) {
	var first []placement
	switch {
	case q.group == groupDetached:
		found, err := c.detached()
		if err != nil {
			return nil, "", false, err
		}
		first = found
	case q.parent == "":
		for _, at := range c.parties {
			if !at.held && at.parent == "" {
				first = append(first, at)
			}
		}
	default:
		first = c.under(q.parent)
	}
	nodes := []treeNode{}
	next, stopped := "", false
	cursor := ""
	if q.phase == phaseParties {
		from := 0
		if q.after != "" {
			at, known := c.byID[q.after]
			if !known {
				return nil, "", false, errInvalidQuery
			}
			from = sort.Search(len(first), func(i int) bool { return c.byID[first[i].party.ID] > at })
		}
		rest := first[from:]
		for _, at := range rest[:min(len(rest), q.limit)] {
			nodes = append(nodes, c.partyNode(at))
		}
		if len(rest) > q.limit {
			next = phaseParties + ":" + nodes[len(nodes)-1].ID
		}
	} else {
		cursor = q.after
	}
	if next == "" && q.group == "" {
		want := q.limit - len(nodes)
		var ids []string
		var last string
		var err error
		kind, source := "account", "relationship"
		if q.parent == "" {
			kind, source = "unknown", "none"
			ids, last, stopped, err = c.unknownRoots(cursor, want+1)
		} else {
			ids, last, stopped, err = c.accounts(q.parent, cursor, want+1)
		}
		if err != nil {
			return nil, "", false, err
		}
		if len(ids) > want {
			ids, stopped = ids[:want], false
			next = phaseOthers + ":" + cursor
			if want > 0 {
				next = phaseOthers + ":" + ids[want-1]
			}
		} else if stopped {
			next = phaseOthers + ":" + last
		}
		for _, id := range ids {
			nodes = append(nodes, treeNode{ID: id, Name: c.s.label(c.ws, id), Kind: kind, Source: source})
		}
	}
	for i := range nodes {
		if err := c.children(&nodes[i]); err != nil {
			return nil, "", false, err
		}
	}
	return nodes, next, stopped, nil
}

func (s *Server) readTree(ws string, q treeQuery) (treePage, error) {
	if s.faulty("organizations") {
		return treePage{}, errors.New("organization tree fault")
	}
	parties, errParties := s.orgs.Parties(ws)
	active, errActive := s.rels.Active(ws)
	if err := errors.Join(errParties, errActive); err != nil {
		return treePage{}, err
	}
	number := active
	if q.pinned {
		if q.revision > active {
			return treePage{}, errInvalidQuery
		}
		number = q.revision
	}
	if len(parties) > orgCap {
		return treePage{}, errors.New("organization listing above the pre-launch cap")
	}
	scan := &treeScan{s: s, ws: ws, number: number, on: q.on, budget: s.cfg.Org.FilterScan, known: map[string]string{}}
	if err := scan.place(parties); err != nil {
		return treePage{}, err
	}
	nodes, next, stopped, err := scan.page(q)
	if err != nil {
		return treePage{}, err
	}
	return treePage{Revision: s.Revision(), Relationship: number, Active: active, On: q.on.ISO(), Parent: q.parent, Group: q.group, Nodes: nodes, Next: next, ScanStopped: stopped, RegistryEmpty: len(parties) == 0}, nil
}

func (s *Server) listTree(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	if r.URL.Query().Has("provider") {
		failModule(w, commands.UnsupportedCapability, http.StatusNotImplemented, false, commands.NextNone, request)
		return
	}
	q, valid := s.treeQuery(r)
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page, err := s.readTree(answer.Workspace(), q)
	switch {
	case errors.Is(err, errInvalidQuery):
		failCode(w, commands.InvalidRequest, request)
	case err != nil:
		failCode(w, commands.DependencyUnavailable, request)
	default:
		writeJSON(w, http.StatusOK, request, page)
	}
}
