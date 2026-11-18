package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const (
	orgCap          = 500
	orgDefaultLimit = 50
)

var relationKinds = []relationships.Kind{relationships.Hierarchy, relationships.Payer, relationships.ContractPool, relationships.AllocationPool, relationships.ViewGroup}

type partyRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Parent string `json:"parent"`
	State  string `json:"state"`
}

type profileRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Issuer    string `json:"issuer"`
	Buyer     string `json:"buyer"`
	Payer     string `json:"payer"`
	Recipient string `json:"recipient"`
	Path      string `json:"path"`
	Currency  string `json:"currency"`
	State     string `json:"state"`
}

type namedRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type historyRow struct {
	Target string `json:"target"`
	Name   string `json:"name"`
	From   string `json:"from"`
	Until  string `json:"until"`
}

type timelineRow struct {
	Member  string       `json:"member"`
	Name    string       `json:"name"`
	Targets []namedRow   `json:"targets"`
	History []historyRow `json:"history"`
}

type workspaceRow struct {
	ID      string `json:"id"`
	Profile string `json:"profile"`
}

type orgPage struct {
	Revision     string        `json:"revision"`
	Relationship uint64        `json:"relationship_revision"`
	Active       uint64        `json:"active_revision"`
	On           string        `json:"on"`
	Workspace    workspaceRow  `json:"workspace"`
	Parties      []partyRow    `json:"parties"`
	Profiles     []profileRow  `json:"profiles"`
	Kind         string        `json:"kind"`
	Members      []timelineRow `json:"members"`
	Next         string        `json:"next"`
	Filters      filterEcho    `json:"filters"`
	Provider     string        `json:"provider_filter"`
	ScanStopped  bool          `json:"scan_stopped"`
	ScanBudget   int           `json:"scan_budget"`
}

type orgQuery struct {
	on       relationships.Day
	kind     relationships.Kind
	after    string
	limit    int
	revision uint64
	pinned   bool
	filter   orgFilter
}

func parseDay(raw string) (relationships.Day, bool) {
	t, err := time.Parse(time.DateOnly, raw)
	if err != nil || t.Format(time.DateOnly) != raw {
		return relationships.Day{}, false
	}
	d, err := relationships.NewDay(t.Year(), int(t.Month()), t.Day())
	return d, err == nil
}

func (s *Server) orgQuery(r *http.Request) (orgQuery, bool) {
	q := r.URL.Query()
	now := s.clock().UTC()
	out := orgQuery{kind: relationships.Hierarchy, after: q.Get("after"), limit: orgDefaultLimit}
	out.on, _ = relationships.NewDay(now.Year(), int(now.Month()), now.Day())
	if raw := q.Get("on"); raw != "" {
		day, ok := parseDay(raw)
		if !ok {
			return out, false
		}
		out.on = day
	}
	if raw := q.Get("kind"); raw != "" {
		known := false
		for _, k := range relationKinds {
			known = known || string(k) == raw
		}
		if !known {
			return out, false
		}
		out.kind = relationships.Kind(raw)
	}
	if out.after != "" && !validWorkspace(out.after) {
		return out, false
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > RelationshipLimits().PageSize-1 || strconv.Itoa(n) != raw {
			return out, false
		}
		out.limit = n
	}
	if raw := q.Get("revision"); raw != "" {
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || strconv.FormatUint(n, 10) != raw {
			return out, false
		}
		out.revision, out.pinned = n, true
	}
	var ok bool
	out.filter, ok = parseFilter(q, out.after)
	return out, ok
}

func covering(entries []relationships.Entry, day relationships.Day) []namedRow {
	out := []namedRow{}
	for _, e := range entries {
		end, bounded := e.Interval.End()
		if !day.Before(e.Interval.Start()) && (!bounded || day.Before(end)) {
			out = append(out, namedRow{ID: e.Target, Name: orgName(e.Target)})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

func historyOf(entries []relationships.Entry) []historyRow {
	out := []historyRow{}
	for _, e := range entries {
		row := historyRow{Target: e.Target, Name: orgName(e.Target), From: e.Interval.Start().ISO()}
		if end, bounded := e.Interval.End(); bounded {
			row.Until = end.ISO()
		}
		out = append(out, row)
	}
	return out
}

func (s *Server) readOrganizations(ws string, q orgQuery) (orgPage, error) {
	if s.faulty("organizations") {
		return orgPage{}, organizations.ErrNotFound
	}
	workspace, _, errWorkspace := s.orgs.Workspace(ws)
	parties, errParties := s.orgs.Parties(ws)
	profiles, errProfiles := s.orgs.BillingProfiles(ws)
	active, errActive := s.rels.Active(ws)
	number := active
	if q.pinned && errActive == nil {
		if q.revision > active {
			return orgPage{}, errInvalidQuery
		}
		number = q.revision
	}
	if err := errors.Join(errWorkspace, errParties, errProfiles, errActive); err != nil {
		return orgPage{}, err
	}
	if len(parties) > orgCap || len(profiles) > orgCap {
		return orgPage{}, errors.New("organization listing above the pre-launch cap")
	}
	if q.filter.entity != "" && !legalEntity(parties, q.filter.entity) {
		return orgPage{}, errInvalidQuery
	}
	scan := &orgScan{s: s, ws: ws, number: number, q: q, parties: map[string]bool{}}
	page := orgPage{Revision: s.Revision(), Relationship: number, Active: active, On: q.on.ISO(), Workspace: workspaceRow{ID: ws, Profile: string(workspace.Profile)}, Parties: []partyRow{}, Profiles: []profileRow{}, Kind: string(q.kind), Members: []timelineRow{},
		Filters: q.filter.echo(), Provider: providerFilter, ScanBudget: s.cfg.Org.FilterScan}
	for _, p := range parties {
		scan.parties[p.ID] = true
		page.Parties = append(page.Parties, partyRow{ID: p.ID, Name: orgName(p.ID), Kind: string(p.Kind), Parent: p.ParentID, State: string(p.State)})
	}
	for _, p := range profiles {
		page.Profiles = append(page.Profiles, profileRow{ID: p.ID, Name: orgName(p.ID), Issuer: p.IssuerID, Buyer: p.BuyerID, Payer: p.PayerID,
			Recipient: p.RecipientID, Path: string(p.Path), Currency: p.Currency, State: string(p.State)})
	}
	found, err := scan.page(s.cfg.Org.FilterScan)
	if err != nil {
		return orgPage{}, err
	}
	page.Next, page.ScanStopped = found.next, found.stopped
	for _, line := range found.lines {
		page.Members = append(page.Members, timelineRow{Member: line.Member, Name: orgName(line.Member), Targets: covering(line.Entries, q.on), History: historyOf(line.Entries)})
	}
	return page, nil
}

var errInvalidQuery = errors.New("organization query is invalid")

func (s *Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
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
	q, valid := s.orgQuery(r)
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page, err := s.readOrganizations(answer.Workspace(), q)
	switch {
	case errors.Is(err, errInvalidQuery):
		failCode(w, commands.InvalidRequest, request)
	case err != nil:
		failCode(w, commands.DependencyUnavailable, request)
	default:
		writeJSON(w, http.StatusOK, request, page)
	}
}
