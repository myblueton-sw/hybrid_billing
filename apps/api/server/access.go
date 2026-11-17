package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
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
	Effective  bool     `json:"effective"`
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
	Revision    string             `json:"revision"`
	Grants      []grantRow         `json:"grants"`
	Delegations []delegationRow    `json:"delegations"`
	Catalog     []ceilingRow       `json:"catalog"`
	Subjects    []subjectRow       `json:"subjects"`
	Now         string             `json:"now"`
	You         string             `json:"you"`
	StartedAt   string             `json:"started_at"`
	Settings    []settingRow       `json:"settings"`
	Requests    []changeRequestRow `json:"requests"`
	Revocations []revocationRow    `json:"revocations"`
	Ledger      ledgerRow          `json:"ledger"`
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

const stateUnconfirmed = "revocation_unconfirmed"

func closedState(state string, unconfirmed bool) string {
	if state == "revoked" && unconfirmed {
		return stateUnconfirmed
	}
	return state
}

func (s *Server) unconfirmedTargets() map[string]bool {
	out := map[string]bool{}
	for _, v := range s.changes.Revocations(s.bootWorkspace) {
		if v.State == accesschange.RevocationReceived || v.State == accesschange.RevocationUnconfirmed {
			out[v.Target.Kind+"/"+v.Target.ID] = true
		}
	}
	return out
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
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	subjects, err := s.grantees(found.verifier)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	facts, err := s.accessFacts(found.verifier)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	now := s.clock()
	page := accessPage{Revision: s.Revision(), Grants: []grantRow{}, Delegations: []delegationRow{}, Catalog: s.ceiling(), Subjects: subjects, Now: stamp(now),
		You: answer.User(), StartedAt: stamp(s.startedAt), Settings: s.settingRows(), Requests: facts.requestRows(answer.User()), Revocations: s.revocationRows(),
		Ledger: s.ledgerRow()}
	unconfirmed := s.unconfirmedTargets()
	for _, g := range s.admin.Grants() {
		if g.Terms.Action == gate.AdministerInstallation {
			continue
		}
		page.Grants = append(page.Grants, grantRow{ID: g.ID, Subject: g.SubjectID, Client: g.ClientID, Action: string(g.Terms.Action), Kind: g.Terms.Kind,
			Fields: append([]string{}, g.Terms.Fields...), Scope: scopeOf(g.Terms), NotBefore: stamp(g.Validity.NotBefore), NotAfter: stamp(g.Validity.NotAfter),
			State: closedState(stateOf(g.Closed, g.Validity, now), unconfirmed[accesschange.TargetGrant+"/"+g.ID]), Effective: g.Effective, Version: g.Version})
	}
	for _, d := range s.admin.Delegations() {
		state := closedState(stateOf(d.Closed, d.Validity, now), unconfirmed[accesschange.TargetDelegation+"/"+d.ID])
		if state == "active" && !d.Effective {
			state = "ended"
		}
		page.Delegations = append(page.Delegations, delegationRow{ID: d.ID, Delegator: d.DelegatorID, Delegate: d.DelegateID, Parent: d.ParentID, Redelegate: d.AllowRedelegation,
			Action: string(d.Terms.Action), Kind: d.Terms.Kind, Fields: append([]string{}, d.Terms.Fields...), Scope: scopeOf(d.Terms),
			NotBefore: stamp(d.Validity.NotBefore), NotAfter: stamp(d.Validity.NotAfter), State: state, Effective: d.Effective, Version: d.Version})
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

func (s *Server) accessDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	return append(s.requestDefinitions(permission),
		commands.Definition{ID: "access.revoke_delegation", Permission: permission, Mutating: true, Rationale: true, Handler: delegationCommand{s: s}})
}
