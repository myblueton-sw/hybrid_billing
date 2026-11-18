package server

import (
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const noneFact = "none"

type accessFacts struct {
	s       *Server
	members map[string]identity.MemberInfo
	users   map[string]identity.UserInfo
	grants  []gate.GrantView
	now     time.Time
}

func (s *Server) accessFacts(actor identity.Verifier) (accessFacts, error) {
	members, err := s.roster(actor)
	if err != nil {
		return accessFacts{}, err
	}
	users := map[string]identity.UserInfo{}
	for after := ""; ; {
		page, err := s.users(after)
		if err != nil {
			return accessFacts{}, err
		}
		for _, u := range page {
			users[u.ID] = u
		}
		if len(page) < 100 {
			break
		}
		after = page[len(page)-1].ID
	}
	return accessFacts{s: s, members: members, users: users, grants: s.admin.Grants(), now: s.clock()}, nil
}

func (f accessFacts) person(user string) string {
	m, u := f.members[user], f.users[user]
	return strings.Join([]string{user, strconv.FormatUint(m.Version, 10), string(m.Role), string(m.State), string(m.UserState), strconv.FormatUint(u.Epoch, 10),
		string(u.State), string(u.Kind)}, "/")
}

func (f accessFacts) holdsAdmin(user string) bool {
	for _, g := range f.grants {
		if g.SubjectID == user && g.Terms.Action == gate.AdministerInstallation && g.Effective && g.Terms.Scope.WorkspaceID == f.s.bootWorkspace {
			return true
		}
	}
	return false
}

func (f accessFacts) eligible(user string) bool {
	m, ok := f.members[user]
	return ok && m.Role == identity.Administrator && m.State == identity.MembershipActive && m.UserState == identity.Active &&
		f.users[user].Kind == identity.Human && f.holdsAdmin(user)
}

func (s *Server) pairRequest(c accesschange.Change) gate.Request {
	customer := ""
	if c.Customer != "all" {
		customer = c.Customer
	}
	return gate.Request{Action: gate.Action(c.Action), Kind: c.ResourceKind, Fields: append([]string(nil), c.Fields...),
		Resource: gate.Resource{WorkspaceID: s.bootWorkspace, CustomerID: customer}}
}

func (f accessFacts) covered(user string, c accesschange.Change) bool {
	return f.s.admin.Explain(user, f.s.pairRequest(c)).Allowed
}

func (f accessFacts) ceilingFor(c accesschange.Change) (gate.GrantView, bool) {
	var found gate.GrantView
	for _, g := range f.grants {
		if g.ClientID == ConsoleClient && g.Active && !gate.InstallationAction(g.Terms.Action) && g.Terms.Scope.WorkspaceID == f.s.bootWorkspace && fullScope(g.Terms) &&
			string(g.Terms.Action) == c.Action && g.Terms.Kind == c.ResourceKind && len(c.Fields) > 0 == (len(g.Terms.Fields) > 0) && covers(g.Terms.Fields, c.Fields) &&
			g.Validity.NotAfter.After(found.Validity.NotAfter) {
			found = g
		}
	}
	return found, found.ID != ""
}

func (f accessFacts) equivalent(c accesschange.Change) string {
	for _, g := range f.grants {
		if g.SubjectID == c.Subject && !g.Closed && f.now.Before(g.Validity.NotAfter) && string(g.Terms.Action) == c.Action && g.Terms.Kind == c.ResourceKind &&
			strings.Join(g.Terms.Fields, ",") == strings.Join(c.Fields, ",") && selectorText(g.Terms.Scope.Customer) == c.Customer {
			return g.ID
		}
	}
	return noneFact
}

func grantFact(g gate.GrantView, ok bool) string {
	if !ok {
		return noneFact
	}
	return strings.Join([]string{g.ID, strconv.FormatUint(g.Version, 10), strconv.FormatBool(g.Closed), g.SubjectID, g.ClientID, string(g.Terms.Action)}, "/")
}

func (f accessFacts) basis(c accesschange.Change, proposer string) string {
	parts := []string{"proposer:" + f.person(proposer) + "/" + strconv.FormatBool(f.holdsAdmin(proposer))}
	if c.Kind == accesschange.ChangeAdd {
		ceiling, ok := f.ceilingFor(c)
		parts = append(parts, "ceiling:"+grantFact(ceiling, ok)+"/"+stamp(ceiling.Validity.NotAfter), "subject:"+f.person(c.Subject), "equivalent:"+f.equivalent(c),
			"covered:"+strconv.FormatBool(f.covered(proposer, c)))
	} else {
		g, ok := f.s.grantView(c.Grant)
		parts = append(parts, "grant:"+grantFact(g, ok))
	}
	return strings.Join(parts, ";")
}

func (s *Server) humanActor(user string) error {
	_, human, err := s.findUser(user)
	switch {
	case err != nil:
		return commands.Fail(commands.DependencyUnavailable)
	case !human:
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return nil
}

func mapChange(err error) error {
	var refusal accesschange.Refusal
	if !errors.As(err, &refusal) {
		var failure *commands.Failure
		if errors.As(err, &failure) {
			return failure
		}
		return mapAccess(err)
	}
	switch refusal.Reason {
	case accesschange.ReasonNotPending, accesschange.ReasonExpired, accesschange.ReasonSuperseded, accesschange.ReasonPendingTarget, accesschange.ReasonNotFound:
		return commands.Fail(commands.InvalidStage)
	case accesschange.ReasonDigest:
		return commands.Fail(commands.InvalidRequest, "digest")
	case accesschange.ReasonExpiry:
		return commands.Fail(commands.InvalidRequest, "days")
	}
	return commands.Fail(commands.PrerequisiteFailed)
}

type requestCommand struct {
	s    *Server
	kind string
}

func (h requestCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	var found bool
	switch h.kind {
	case "request_grant":
		found = target == grantsTarget
	case "request_revoke":
		_, found = s.accessGrant(target)
	default:
		n, ok := accesschange.Number(target, accesschange.RequestPrefix)
		_, known := s.changes.Request(n, workspace)
		found = ok && known
	}
	if !found || workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (h requestCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	if err := s.humanActor(ctx.User); err != nil {
		return commands.Outcome{}, err
	}
	f, err := s.accessFacts(s.caller)
	if err != nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	actor := accesschange.Actor{User: ctx.User, Human: true}
	switch h.kind {
	case "request_grant":
		return h.requestGrant(ctx, f, actor)
	case "request_revoke":
		return h.requestRevoke(ctx, f, actor)
	case "approve":
		return h.approve(ctx, f, actor)
	}
	n, _ := accesschange.Number(ctx.Target, accesschange.RequestPrefix)
	var v accesschange.View
	if h.kind == "reject" {
		v, err = s.changes.Reject(n, ctx.Workspace, actor, f.eligible(ctx.User), ctx.Rationale)
	} else {
		v, err = s.changes.Withdraw(n, ctx.Workspace, actor)
	}
	if err != nil {
		return commands.Outcome{}, mapChange(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: v.State, Fields: map[string]string{"request": v.ID}}, nil
}

func (h requestCommand) addChange(ctx commands.Context, f accessFacts) (accesschange.Change, gate.GrantView, error) {
	subject := ctx.Fields["subject"]
	if m, ok := f.members[subject]; !ok || m.State != identity.MembershipActive || m.UserState != identity.Active {
		return accesschange.Change{}, gate.GrantView{}, commands.Fail(commands.InvalidRequest, "subject")
	}
	fields, ok := splitFields(ctx.Fields["fields"])
	if !ok {
		return accesschange.Change{}, gate.GrantView{}, commands.Fail(commands.InvalidRequest, "fields")
	}
	c := accesschange.Change{Kind: accesschange.ChangeAdd, Subject: subject, Action: ctx.Fields["action"], ResourceKind: ctx.Fields["kind"], Fields: fields,
		Customer: ctx.Fields["customer"]}
	ceiling, ok := f.ceilingFor(c)
	if !ok {
		return accesschange.Change{}, gate.GrantView{}, commands.Fail(commands.InvalidRequest, "action")
	}
	if c.Customer != "all" && (!validWorkspace(c.Customer) || c.ResourceKind == ReaderKind) {
		return accesschange.Change{}, gate.GrantView{}, commands.Fail(commands.InvalidRequest, "customer")
	}
	most := int(ceiling.Validity.NotAfter.Sub(f.now) / (24 * time.Hour))
	if c.Days, ok = minutes(ctx.Fields["days"], 1, most); !ok {
		return accesschange.Change{}, gate.GrantView{}, commands.Fail(commands.InvalidRequest, "days")
	}
	return c, ceiling, nil
}

func requested(s *Server, v accesschange.View) commands.Outcome {
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: "requested", Fields: map[string]string{"request": v.ID, "digest": v.Digest}}
}

func (h requestCommand) requestGrant(ctx commands.Context, f accessFacts, actor accesschange.Actor) (commands.Outcome, error) {
	s := h.s
	c, ceiling, err := h.addChange(ctx, f)
	if err != nil {
		return commands.Outcome{}, err
	}
	if s.ledgerBlocksIncrease() {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	covered := f.covered(ctx.User, c)
	switch {
	case c.Subject == ctx.User || !covered:
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	case f.equivalent(c) != noneFact:
		return commands.Outcome{}, commands.Fail(commands.InvalidStage)
	}
	v, _, err := s.changes.Propose(accesschange.Proposal{Workspace: ctx.Workspace, Change: c, Basis: f.basis(c, ctx.User), CeilingUntil: ceiling.Validity.NotAfter,
		Proposer: actor, Allowed: covered, Rationale: ctx.Rationale})
	if err != nil {
		return commands.Outcome{}, mapChange(err)
	}
	return requested(s, v), nil
}

func (h requestCommand) requestRevoke(ctx commands.Context, f accessFacts, actor accesschange.Actor) (commands.Outcome, error) {
	s := h.s
	g, _ := s.grantView(ctx.Target)
	switch {
	case protectedGrant(g) || g.SubjectID == ctx.User:
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	case g.Closed:
		return commands.Outcome{}, commands.Fail(commands.InvalidStage)
	}
	c := accesschange.Change{Kind: accesschange.ChangeRevoke, Grant: g.ID}
	v, _, err := s.changes.Propose(accesschange.Proposal{Workspace: ctx.Workspace, Change: c, Holder: g.SubjectID, Basis: f.basis(c, ctx.User), Proposer: actor,
		Allowed: true, Rationale: ctx.Rationale})
	if err != nil {
		return commands.Outcome{}, mapChange(err)
	}
	return requested(s, v), nil
}

func (f accessFacts) standing(v accesschange.View, approver string) accesschange.Standing {
	allowed := f.eligible(approver)
	if v.Change.Kind == accesschange.ChangeAdd {
		allowed = allowed && f.covered(approver, v.Change) && f.covered(v.Proposer, v.Change)
	} else if g, ok := f.s.grantView(v.Change.Grant); ok && protectedGrant(g) {
		allowed = false
	}
	return accesschange.Standing{Basis: f.basis(v.Change, v.Proposer), Allowed: allowed}
}

func (s *Server) applyAdd(v accesschange.View, now time.Time) (string, error) {
	customer := gate.All()
	if v.Change.Customer != "all" {
		customer = gate.One(v.Change.Customer)
	}
	id := "grant-" + hex.EncodeToString(random(8))
	terms := gate.Terms{Action: gate.Action(v.Change.Action), Kind: v.Change.ResourceKind, Fields: append([]string(nil), v.Change.Fields...), Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: s.bootWorkspace, Customer: customer, LegalEntity: gate.All(), Contract: gate.All()}}
	err := gate.ErrChanged
	if !s.faulty("access") {
		err = s.admin.AddGrant(gate.Grant{ID: id, SubjectID: v.Change.Subject, Terms: terms, Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Duration(v.Change.Days) * 24 * time.Hour)}})
	}
	if err != nil {
		return "", mapAccess(err)
	}
	return id, nil
}

func (h requestCommand) approve(ctx commands.Context, f accessFacts, actor accesschange.Actor) (commands.Outcome, error) {
	s := h.s
	n, _ := accesschange.Number(ctx.Target, accesschange.RequestPrefix)
	v, _ := s.changes.Request(n, ctx.Workspace)
	add := v.Change.Kind == accesschange.ChangeAdd
	if add && s.ledgerBlocksIncrease() {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	apply := func(a accesschange.Applier) (string, error) {
		if add {
			return s.applyAdd(v, f.now)
		}
		g, _ := s.grantView(v.Change.Grant)
		rv, err := a.Revoke(accesschange.Revocation{Workspace: ctx.Workspace, Target: accesschange.Target{Kind: accesschange.TargetGrant, ID: g.ID, Version: g.Version},
			RequestedBy: v.Proposer, ApprovedBy: ctx.User, Request: v.ID, RequestDigest: v.Digest}, s.grantEffects(g))
		return rv.ID, err
	}
	got, err := s.changes.Approve(n, ctx.Workspace, actor, ctx.Fields["digest"], f.standing(v, ctx.User), apply)
	if err != nil {
		return commands.Outcome{}, mapChange(err)
	}
	if s.faulty("access-approve") {
		return commands.Outcome{}, errPartial
	}
	if add {
		return commands.Outcome{ResultRevision: s.Next(), BusinessState: "granted", Fields: map[string]string{"request": got.ID, "grant": got.Result}}, nil
	}
	return commands.Outcome{Accepted: true, JobID: got.Result, Fields: map[string]string{"request": got.ID, "revocation": got.Result}}, nil
}

type changeRequestRow struct {
	ID                string   `json:"id"`
	Change            string   `json:"change"`
	Subject           string   `json:"subject"`
	Grant             string   `json:"grant"`
	Action            string   `json:"action"`
	Kind              string   `json:"kind"`
	Fields            []string `json:"fields"`
	Customer          string   `json:"customer"`
	Days              int      `json:"days"`
	Digest            string   `json:"digest"`
	State             string   `json:"state"`
	Current           bool     `json:"current"`
	Proposer          string   `json:"proposer"`
	ProposedAt        string   `json:"proposed_at"`
	ExpiresAt         string   `json:"expires_at"`
	Reason            string   `json:"reason"`
	DecidedBy         string   `json:"decided_by"`
	DecidedAt         string   `json:"decided_at"`
	DecisionReason    string   `json:"decision_reason"`
	Result            string   `json:"result"`
	EligibleApprovers int      `json:"eligible_approvers"`
	CanApprove        bool     `json:"can_approve"`
	CanReject         bool     `json:"can_reject"`
	CanWithdraw       bool     `json:"can_withdraw"`
	MayEnd            []string `json:"may_end"`
}

func (f accessFacts) mayEnd(holder string) []string {
	out := []string{}
	for _, d := range f.s.admin.Delegations() {
		if d.Effective && !d.Closed && (d.DelegatorID == holder || d.DelegateID == holder) {
			out = append(out, d.ID)
		}
	}
	return out
}

func (f accessFacts) changeRequestRow(v accesschange.View, you string) changeRequestRow {
	c := v.Change
	row := changeRequestRow{ID: v.ID, Change: c.Kind, Subject: c.Subject, Grant: c.Grant, Action: c.Action, Kind: c.ResourceKind, Fields: append([]string{}, c.Fields...),
		Customer: c.Customer, Days: c.Days, Digest: v.Digest, State: v.State, Proposer: v.Proposer, ProposedAt: stamp(v.ProposedAt), ExpiresAt: stamp(v.ExpiresAt),
		Reason: v.Rationale, DecidedBy: v.DecidedBy, DecidedAt: stampOrEmpty(v.DecidedAt), DecisionReason: v.DecisionReason, Result: v.Result, MayEnd: []string{}}
	if c.Kind == accesschange.ChangeRevoke {
		row.Subject = v.Holder
		if g, ok := f.s.grantView(c.Grant); ok {
			row.Action, row.Kind, row.Fields, row.Customer = string(g.Terms.Action), g.Terms.Kind, append([]string{}, g.Terms.Fields...), selectorText(g.Terms.Scope.Customer)
		}
	}
	if v.State != accesschange.StatePending {
		return row
	}
	row.Current = f.basis(c, v.Proposer) == v.Basis
	for user := range f.members {
		if f.eligible(user) && user != v.Proposer && (c.Kind == accesschange.ChangeRevoke || user != c.Subject) {
			row.EligibleApprovers++
		}
	}
	standing := f.standing(v, you)
	row.CanApprove = row.Current && standing.Allowed && you != v.Proposer && (c.Kind == accesschange.ChangeRevoke || (you != c.Subject && !f.s.ledgerBlocksIncrease()))
	row.CanReject = f.eligible(you) && you != v.Proposer && (c.Kind == accesschange.ChangeAdd || you != v.Holder)
	row.CanWithdraw = you == v.Proposer
	if c.Kind == accesschange.ChangeRevoke {
		row.MayEnd = f.mayEnd(v.Holder)
	}
	return row
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

func (s *Server) requestDefinitions(permission commands.Permission) []commands.Definition {
	fields := []commands.Field{{Name: "subject", Type: commands.ID, Required: true}, {Name: "action", Type: commands.Text, Required: true}, {Name: "kind", Type: commands.Text, Required: true},
		{Name: "fields", Type: commands.Text}, {Name: "customer", Type: commands.Text, Required: true}, {Name: "days", Type: commands.Integer, Required: true}}
	return []commands.Definition{
		{ID: "access.request_grant", Permission: permission, Mutating: true, Rationale: true, Fields: fields, Handler: requestCommand{s: s, kind: "request_grant"}},
		{ID: "access.request_revoke", Permission: permission, Mutating: true, Rationale: true, Handler: requestCommand{s: s, kind: "request_revoke"}},
		{ID: "access.approve_request", Permission: permission, Mutating: true, Rationale: true, Fields: []commands.Field{{Name: "digest", Type: commands.Text, Required: true}},
			Handler: requestCommand{s: s, kind: "approve"}},
		{ID: "access.reject_request", Permission: permission, Mutating: true, Rationale: true, Handler: requestCommand{s: s, kind: "reject"}},
		{ID: "access.withdraw_request", Permission: permission, Mutating: true, Rationale: true, Handler: requestCommand{s: s, kind: "withdraw"}},
	}
}
