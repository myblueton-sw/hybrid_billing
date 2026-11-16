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
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const grantsTarget = "grants"

var previewActions = []gate.Action{gate.View, gate.ViewCost, gate.ChangeCustomerPolicy, gate.ChangeSellingPolicy, gate.RequestApproval, gate.Issue, gate.Download, gate.Deliver}

type scopeRow struct {
	Customer    string `json:"customer"`
	LegalEntity string `json:"legal_entity"`
	Contract    string `json:"contract"`
	Period      string `json:"period"`
}

type grantRow struct {
	ID        string   `json:"id"`
	Subject   string   `json:"subject"`
	Client    string   `json:"client"`
	Action    string   `json:"action"`
	Kind      string   `json:"kind"`
	Fields    []string `json:"fields"`
	Scope     scopeRow `json:"scope"`
	NotBefore string   `json:"not_before"`
	NotAfter  string   `json:"not_after"`
	State     string   `json:"state"`
	Effective bool     `json:"effective"`
	Version   uint64   `json:"version"`
}

type delegationRow struct {
	ID         string   `json:"id"`
	Delegator  string   `json:"delegator"`
	Delegate   string   `json:"delegate"`
	Parent     string   `json:"parent"`
	Redelegate bool     `json:"redelegate"`
	Action     string   `json:"action"`
	Kind       string   `json:"kind"`
	Fields     []string `json:"fields"`
	Scope      scopeRow `json:"scope"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	State      string   `json:"state"`
	Version    uint64   `json:"version"`
}

type ceilingRow struct {
	Action string   `json:"action"`
	Kind   string   `json:"kind"`
	Fields []string `json:"fields"`
	Until  string   `json:"until"`
}

type subjectRow struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
}

type accessPage struct {
	Revision    string          `json:"revision"`
	Grants      []grantRow      `json:"grants"`
	Delegations []delegationRow `json:"delegations"`
	Catalog     []ceilingRow    `json:"catalog"`
	Subjects    []subjectRow    `json:"subjects"`
	Now         string          `json:"now"`
}

type decisionRow struct {
	Action      string   `json:"action"`
	Field       string   `json:"field"`
	Allowed     bool     `json:"allowed"`
	Reason      string   `json:"reason"`
	Delegations []string `json:"delegations"`
}

func selectorText(s gate.Selector) string {
	if s.Any() {
		return "all"
	}
	return s.ID()
}

func scopeOf(t gate.Terms) scopeRow {
	period := "any"
	if !t.Period.Any() {
		period = strconv.Itoa(int(t.Period.From())) + "-" + strconv.Itoa(int(t.Period.To()))
	}
	return scopeRow{Customer: selectorText(t.Scope.Customer), LegalEntity: selectorText(t.Scope.LegalEntity), Contract: selectorText(t.Scope.Contract), Period: period}
}

func stateOf(closed bool, v gate.Validity, now time.Time) string {
	switch {
	case closed:
		return "revoked"
	case now.Before(v.NotBefore):
		return "pending"
	case !now.Before(v.NotAfter):
		return "expired"
	}
	return "active"
}

func fullScope(t gate.Terms) bool {
	return t.Scope.Customer.Any() && t.Scope.LegalEntity.Any() && t.Scope.Contract.Any() && t.Period.Any()
}

func (s *Server) ceiling() []ceilingRow {
	out := []ceilingRow{}
	for _, g := range s.admin.Grants() {
		if g.ClientID == ConsoleClient && g.Active && g.Terms.Action != gate.AdministerInstallation && g.Terms.Scope.WorkspaceID == s.bootWorkspace && fullScope(g.Terms) {
			out = append(out, ceilingRow{Action: string(g.Terms.Action), Kind: g.Terms.Kind, Fields: append([]string{}, g.Terms.Fields...), Until: stamp(g.Validity.NotAfter)})
		}
	}
	return out
}

func (s *Server) grantees(actor identity.Verifier) ([]subjectRow, error) {
	members, err := s.roster(actor)
	if err != nil {
		return nil, err
	}
	out := []subjectRow{}
	for user, m := range members {
		if m.State == identity.MembershipActive && m.UserState == identity.Active {
			login, _ := s.dir.loginOf(user)
			out = append(out, subjectRow{UserID: user, Login: login})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].UserID < out[b].UserID })
	return out, nil
}

func (s *Server) listAccess(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, _, ok := s.admitted(w, r)
	if !ok {
		return
	}
	subjects, err := s.grantees(found.verifier)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	now := s.clock()
	page := accessPage{Revision: s.Revision(), Grants: []grantRow{}, Delegations: []delegationRow{}, Catalog: s.ceiling(), Subjects: subjects, Now: stamp(now)}
	for _, g := range s.admin.Grants() {
		if g.Terms.Action == gate.AdministerInstallation {
			continue
		}
		page.Grants = append(page.Grants, grantRow{ID: g.ID, Subject: g.SubjectID, Client: g.ClientID, Action: string(g.Terms.Action), Kind: g.Terms.Kind,
			Fields: append([]string{}, g.Terms.Fields...), Scope: scopeOf(g.Terms), NotBefore: stamp(g.Validity.NotBefore), NotAfter: stamp(g.Validity.NotAfter),
			State: stateOf(g.Closed, g.Validity, now), Effective: g.Effective, Version: g.Version})
	}
	for _, d := range s.admin.Delegations() {
		page.Delegations = append(page.Delegations, delegationRow{ID: d.ID, Delegator: d.DelegatorID, Delegate: d.DelegateID, Parent: d.ParentID, Redelegate: d.AllowRedelegation,
			Action: string(d.Terms.Action), Kind: d.Terms.Kind, Fields: append([]string{}, d.Terms.Fields...), Scope: scopeOf(d.Terms),
			NotBefore: stamp(d.Validity.NotBefore), NotAfter: stamp(d.Validity.NotAfter), State: stateOf(d.Closed, d.Validity, now), Version: d.Version})
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) catalogFields(class gate.FieldClass) []string {
	out := []string{}
	for name, c := range Catalog(s.cfg.Kinds, s.cfg.Fields).Fields {
		if c == class {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Server) previewKind(kind string) bool {
	for _, k := range Catalog(s.cfg.Kinds, s.cfg.Fields).Kinds {
		if k == kind && k != gate.InstallationKind {
			return true
		}
	}
	return false
}

func (s *Server) previewAccess(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	q := r.URL.Query()
	subject, kind, customer := q.Get("subject"), q.Get("kind"), q.Get("customer")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, _, ok := s.admitted(w, r)
	if !ok {
		return
	}
	subjects, err := s.grantees(found.verifier)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	known := false
	for _, row := range subjects {
		known = known || row.UserID == subject
	}
	if !known || !s.previewKind(kind) || (customer != "" && !validWorkspace(customer)) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	out := []decisionRow{}
	for _, action := range previewActions {
		fields := []string{""}
		switch action {
		case gate.View, gate.Download:
			fields = s.catalogFields(gate.General)
		case gate.ViewCost:
			fields = s.catalogFields(gate.Cost)
		}
		for _, field := range fields {
			req := gate.Request{Action: action, Kind: kind, Resource: gate.Resource{WorkspaceID: s.bootWorkspace, CustomerID: customer}}
			if field != "" {
				req.Fields = []string{field}
			}
			e := s.admin.Explain(subject, req)
			out = append(out, decisionRow{Action: string(action), Field: field, Allowed: e.Allowed, Reason: string(e.Reason), Delegations: append([]string{}, e.Delegations...)})
		}
	}
	writeJSON(w, http.StatusOK, request, map[string]any{"revision": s.Revision(), "subject": subject, "kind": kind, "customer": customer, "decisions": out})
}

func mapAccess(err error) error {
	switch {
	case errors.Is(err, gate.ErrConflict), errors.Is(err, gate.ErrNotFound):
		return commands.Fail(commands.InvalidStage)
	case errors.Is(err, gate.ErrChanged), errors.Is(err, gate.ErrNotBacked):
		return commands.Fail(commands.PrerequisiteFailed)
	case errors.Is(err, gate.ErrUnavailable):
		return commands.Fail(commands.DependencyUnavailable)
	}
	return commands.Fail(commands.InvalidRequest)
}

type accessCommand struct {
	s    *Server
	kind string
}

func (h accessCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	switch h.kind {
	case "grant":
		if target == grantsTarget {
			return commands.Target{Revision: s.Revision()}, commands.Found
		}
	case "revoke_grant":
		if _, ok := s.grantView(target); ok {
			return commands.Target{Revision: s.Revision()}, commands.Found
		}
	default:
		if _, ok := s.delegationView(target); ok {
			return commands.Target{Revision: s.Revision()}, commands.Found
		}
	}
	return commands.Target{}, commands.NotFound
}

func (s *Server) grantView(id string) (gate.GrantView, bool) {
	for _, g := range s.admin.Grants() {
		if g.ID == id {
			return g, true
		}
	}
	return gate.GrantView{}, false
}

func (s *Server) delegationView(id string) (gate.DelegationView, bool) {
	for _, d := range s.admin.Delegations() {
		if d.ID == id {
			return d, true
		}
	}
	return gate.DelegationView{}, false
}

func splitFields(raw string) ([]string, bool) {
	if raw == "" {
		return nil, true
	}
	out := strings.Split(raw, ",")
	sort.Strings(out)
	for i, f := range out {
		if f == "" || (i > 0 && out[i-1] == f) {
			return nil, false
		}
	}
	return out, true
}

func covers(have, want []string) bool {
	for _, f := range want {
		found := false
		for _, h := range have {
			found = found || h == f
		}
		if !found {
			return false
		}
	}
	return true
}

func (h accessCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	now := s.clock()
	switch h.kind {
	case "grant":
		return h.grant(ctx, now)
	case "revoke_grant":
		g, _ := s.grantView(ctx.Target)
		if g.ClientID != "" || g.Terms.Action == gate.AdministerInstallation || g.SubjectID == ctx.User {
			return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
		}
		if err := s.admin.RevokeGrant(g.ID, g.Version); err != nil {
			return commands.Outcome{}, mapAccess(err)
		}
		return commands.Outcome{ResultRevision: s.Next(), BusinessState: "grant_revoked"}, nil
	}
	d, _ := s.delegationView(ctx.Target)
	if err := s.admin.RevokeDelegation(d.ID, d.Version); err != nil {
		return commands.Outcome{}, mapAccess(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "delegation_revoked"}, nil
}

func (h accessCommand) grant(ctx commands.Context, now time.Time) (commands.Outcome, error) {
	s := h.s
	subject := ctx.Fields["subject"]
	if subject == ctx.User {
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	}
	subjects, err := s.grantees(s.caller)
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	known := false
	for _, row := range subjects {
		known = known || row.UserID == subject
	}
	if !known {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "subject")
	}
	fields, ok := splitFields(ctx.Fields["fields"])
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "fields")
	}
	action, kind := gate.Action(ctx.Fields["action"]), ctx.Fields["kind"]
	var until time.Time
	for _, c := range s.ceiling() {
		if c.Action == string(action) && c.Kind == kind && len(fields) > 0 == (len(c.Fields) > 0) && covers(c.Fields, fields) {
			if at, _ := time.Parse(time.RFC3339, c.Until); at.After(until) {
				until = at
			}
		}
	}
	if until.IsZero() {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "action")
	}
	customer := gate.All()
	switch raw := ctx.Fields["customer"]; {
	case raw == "all":
	case validWorkspace(raw) && kind != ReaderKind:
		customer = gate.One(raw)
	default:
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "customer")
	}
	most := int(until.Sub(now) / (24 * time.Hour))
	days, ok := minutes(ctx.Fields["days"], 1, most)
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "days")
	}
	id := "grant-" + hex.EncodeToString(random(8))
	terms := gate.Terms{Action: action, Kind: kind, Fields: fields, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: s.bootWorkspace, Customer: customer, LegalEntity: gate.All(), Contract: gate.All()}}
	if s.faulty("access") {
		err = gate.ErrChanged
	} else {
		err = s.admin.AddGrant(gate.Grant{ID: id, SubjectID: subject, Terms: terms, Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Duration(days) * 24 * time.Hour)}})
	}
	if err != nil {
		return commands.Outcome{}, mapAccess(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "granted", Fields: map[string]string{"grant": id}}, nil
}

func (s *Server) accessDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	fields := []commands.Field{{Name: "subject", Type: commands.ID, Required: true}, {Name: "action", Type: commands.Text, Required: true}, {Name: "kind", Type: commands.Text, Required: true},
		{Name: "fields", Type: commands.Text}, {Name: "customer", Type: commands.Text, Required: true}, {Name: "days", Type: commands.Integer, Required: true}}
	return []commands.Definition{
		{ID: "access.grant", Permission: permission, Mutating: true, Rationale: true, Fields: fields, Handler: accessCommand{s: s, kind: "grant"}},
		{ID: "access.revoke_grant", Permission: permission, Mutating: true, Rationale: true, Handler: accessCommand{s: s, kind: "revoke_grant"}},
		{ID: "access.revoke_delegation", Permission: permission, Mutating: true, Rationale: true, Handler: accessCommand{s: s, kind: "revoke_delegation"}},
	}
}
