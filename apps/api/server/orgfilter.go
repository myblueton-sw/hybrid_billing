package server

import (
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const (
	SubjectAny     = "any"
	SubjectAccount = "account"
	SubjectParty   = "party"
	orgMemberCap   = 50
	orgSearchBytes = 64
	providerFilter = "not_connected"
)

type orgFilter struct {
	subject      string
	entity       string
	unattributed bool
	q            string
	members      []string
	target       string
}

type filterEcho struct {
	Subject      string `json:"subject"`
	Entity       string `json:"entity"`
	Unattributed bool   `json:"unattributed"`
	Q            string `json:"q"`
	MemberCount  int    `json:"member_count"`
	Target       string `json:"target"`
}

func (f orgFilter) echo() filterEcho {
	return filterEcho{Subject: f.subject, Entity: f.entity, Unattributed: f.unattributed, Q: f.q, MemberCount: len(f.members), Target: f.target}
}

func searchTerm(q url.Values) (string, bool) {
	raw := q.Get("q")
	return raw, len(raw) <= orgSearchBytes && utf8.ValidString(raw)
}

func parseFilter(q url.Values, after string) (orgFilter, bool) {
	out := orgFilter{subject: SubjectAny, entity: q.Get("entity"), target: q.Get("target")}
	switch raw := q.Get("subject"); raw {
	case "", SubjectAny:
	case SubjectAccount, SubjectParty:
		out.subject = raw
	default:
		return out, false
	}
	switch q.Get("unattributed") {
	case "":
	case "true":
		out.unattributed = true
	default:
		return out, false
	}
	term, ok := searchTerm(q)
	out.q = term
	if !ok || (out.entity != "" && !validWorkspace(out.entity)) || (out.target != "" && !validWorkspace(out.target)) {
		return out, false
	}
	listed := q["member"]
	if len(listed) > orgMemberCap || (len(listed) > 0 && (after != "" || q.Has("limit"))) {
		return out, false
	}
	seen := map[string]bool{}
	for _, id := range listed {
		if !validWorkspace(id) {
			return out, false
		}
		if !seen[id] {
			seen[id] = true
			out.members = append(out.members, id)
		}
	}
	sort.Strings(out.members)
	return out, true
}

func legalEntity(parties []organizations.Party, id string) bool {
	for _, p := range parties {
		if p.ID == id {
			return p.Kind == organizations.LegalEntity
		}
	}
	return false
}

func matchesSearch(id, term string) bool {
	term = strings.ToLower(term)
	return strings.Contains(strings.ToLower(id), term) || strings.Contains(strings.ToLower(orgName(id)), term)
}

func coveringIDs(entries []relationships.Entry, day relationships.Day) []string {
	out := []string{}
	for _, row := range covering(entries, day) {
		out = append(out, row.ID)
	}
	return out
}

type orgScan struct {
	s       *Server
	ws      string
	number  uint64
	q       orgQuery
	parties map[string]bool
	targets []string
}

func (c *orgScan) match(line relationships.Timeline) (bool, error) {
	f := c.q.filter
	if (f.subject == SubjectParty && !c.parties[line.Member]) || (f.subject == SubjectAccount && c.parties[line.Member]) || !matchesSearch(line.Member, f.q) {
		return false, nil
	}
	on := coveringIDs(line.Entries, c.q.on)
	if (f.unattributed && len(on) > 0) || (f.target != "" && !contains(on, f.target)) {
		return false, nil
	}
	if f.entity == "" || line.Member == f.entity {
		return true, nil
	}
	chain, err := c.s.rels.Chain(c.ws, c.number, line.Member, c.q.on)
	return contains(chain, f.entity), err
}

func (c *orgScan) timeline(member string) (relationships.Timeline, error) {
	line, _, err := c.s.rels.TimelineOf(c.ws, c.number, c.q.kind, member)
	line.Kind, line.Member = c.q.kind, member
	return line, err
}

func (c *orgScan) lookup(member string) (relationships.Timeline, bool, error) {
	line := relationships.Timeline{Kind: c.q.kind, Member: member}
	kinds := []relationships.Kind{c.q.kind}
	if c.q.filter.subject == SubjectAccount {
		kinds = relationKinds
	}
	listed := false
	for _, kind := range kinds {
		found, has, err := c.s.rels.TimelineOf(c.ws, c.number, kind, member)
		if err != nil {
			return line, false, err
		}
		listed = listed || has
		if kind == c.q.kind {
			line.Entries = found.Entries
		}
	}
	return line, listed, nil
}

func (c *orgScan) exact() ([]relationships.Timeline, error) {
	out := []relationships.Timeline{}
	for _, member := range c.q.filter.members {
		line, has, err := c.lookup(member)
		if err != nil {
			return nil, err
		}
		if !has {
			continue
		}
		ok, err := c.match(line)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, line)
		}
	}
	return out, nil
}

func (c *orgScan) chunk(cursor string) ([]relationships.Timeline, bool, error) {
	size := RelationshipLimits().PageSize
	f := c.q.filter
	var ids []string
	switch {
	case f.target != "":
		if c.targets == nil {
			members, err := c.s.rels.MembersOn(c.ws, c.number, c.q.kind, f.target, c.q.on)
			if err != nil {
				return nil, false, err
			}
			c.targets = members
		}
		start := sort.Search(len(c.targets), func(i int) bool { return c.targets[i] > cursor })
		ids = c.targets[start:min(len(c.targets), start+size)]
	case f.subject == SubjectAccount:
		members, err := c.s.rels.Members(c.ws, c.number, cursor, size)
		if err != nil {
			return nil, false, err
		}
		ids = members
	default:
		lines, err := c.s.rels.Timelines(c.ws, c.number, c.q.kind, cursor, size)
		return lines, len(lines) < size, err
	}
	out := make([]relationships.Timeline, 0, len(ids))
	for _, id := range ids {
		line, err := c.timeline(id)
		if err != nil {
			return nil, false, err
		}
		out = append(out, line)
	}
	return out, len(ids) < size, nil
}

type scanned struct {
	lines   []relationships.Timeline
	next    string
	stopped bool
}

func (c *orgScan) page(budget int) (scanned, error) {
	if len(c.q.filter.members) > 0 {
		lines, err := c.exact()
		return scanned{lines: lines}, err
	}
	out := scanned{}
	cursor, examined := c.q.after, 0
	for {
		lines, ended, err := c.chunk(cursor)
		if err != nil {
			return scanned{}, err
		}
		for _, line := range lines {
			if examined == budget {
				out.next, out.stopped = cursor, true
				return out, nil
			}
			examined++
			cursor = line.Member
			ok, err := c.match(line)
			if err != nil {
				return scanned{}, err
			}
			if !ok {
				continue
			}
			if len(out.lines) == c.q.limit {
				out.next = out.lines[len(out.lines)-1].Member
				return out, nil
			}
			out.lines = append(out.lines, line)
		}
		if ended {
			return out, nil
		}
	}
}
