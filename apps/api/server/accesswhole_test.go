package server

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

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

func (f accessFacts) requestRows(you string) []changeRequestRow {
	pending, ended := []changeRequestRow{}, []changeRequestRow{}
	views := f.s.changes.Requests(f.s.bootWorkspace)
	sort.Slice(views, func(a, b int) bool { return views[a].Number > views[b].Number })
	for _, v := range views {
		row := f.changeRequestRow(v, you)
		if v.State == accesschange.StatePending {
			pending = append(pending, row)
		} else if len(ended) < f.s.cfg.Access.EndedShown {
			ended = append(ended, row)
		}
	}
	return append(pending, ended...)
}

func (s *Server) revocationRows() []revocationRow {
	out := []revocationRow{}
	for _, v := range s.changes.Revocations(s.bootWorkspace) {
		out = append(out, s.revocationRow(v))
	}
	return out
}

func (s *Server) formerPopulations(actor identity.Verifier, you string) (map[string]any, error) {
	subjects, err := s.grantees(actor)
	if err != nil {
		return nil, err
	}
	facts, err := s.accessFacts(actor)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	grants, delegations := []grantRow{}, []delegationRow{}
	unconfirmed := s.unconfirmedTargets()
	for _, g := range s.admin.Grants() {
		if gate.InstallationAction(g.Terms.Action) {
			continue
		}
		grants = append(grants, grantRow{ID: g.ID, Subject: g.SubjectID, Client: g.ClientID, Action: string(g.Terms.Action), Kind: g.Terms.Kind,
			Fields: append([]string{}, g.Terms.Fields...), Scope: scopeOf(g.Terms), NotBefore: stamp(g.Validity.NotBefore), NotAfter: stamp(g.Validity.NotAfter),
			State: closedState(stateOf(g.Closed, g.Validity, now), unconfirmed[accesschange.TargetGrant+"/"+g.ID]), Effective: g.Effective, Version: g.Version})
	}
	for _, d := range s.admin.Delegations() {
		state := closedState(stateOf(d.Closed, d.Validity, now), unconfirmed[accesschange.TargetDelegation+"/"+d.ID])
		if state == "active" && !d.Effective {
			state = "ended"
		}
		delegations = append(delegations, delegationRow{ID: d.ID, Delegator: d.DelegatorID, Delegate: d.DelegateID, Parent: d.ParentID, Redelegate: d.AllowRedelegation,
			Action: string(d.Terms.Action), Kind: d.Terms.Kind, Fields: append([]string{}, d.Terms.Fields...), Scope: scopeOf(d.Terms),
			NotBefore: stamp(d.Validity.NotBefore), NotAfter: stamp(d.Validity.NotAfter), State: state, Effective: d.Effective, Version: d.Version})
	}
	raw, _ := json.Marshal(map[string]any{"grants": grants, "delegations": delegations, "subjects": subjects, "requests": facts.requestRows(you), "revocations": s.revocationRows()})
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (f world) withFormerPopulations(a actor, got reply) reply {
	f.t.Helper()
	if got.status != http.StatusOK || a.cookie == nil {
		return got
	}
	found, ok := f.s.lookup(func() *http.Request {
		r, _ := http.NewRequest("GET", "/", nil)
		r.AddCookie(a.cookie)
		return r
	}())
	if !ok {
		return got
	}
	f.s.domain.RLock()
	former, err := f.s.formerPopulations(found.verifier, found.owner.User)
	f.s.domain.RUnlock()
	if err != nil {
		return got
	}
	for name, rows := range former {
		if _, present := got.body[name]; present {
			f.t.Fatalf("the whole read still carries %s", name)
		}
		got.body[name] = rows
	}
	return got
}
