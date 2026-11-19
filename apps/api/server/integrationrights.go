package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
)

const (
	subWithdrawKind       = "subscription_withdraw"
	resumeWithdrawKind    = "subscription_withdraw_resume"
	redeliverWithdrawKind = "letter_withdraw_redeliver"
	dismissWithdrawKind   = "letter_withdraw_dismiss"
	reapproveKind         = "redelivery_reapprove"

	resumeRequestTarget = "webhook-resume-request:"
	letterRequestTarget = "webhook-letter-request:"
	redeliveryTarget    = "webhook-redelivery:"

	requestResume    = "resume"
	requestRedeliver = "redeliver"
	requestDismiss   = "dismiss"
	requestTombstone = 10000

	stateNotPermitted  = "not_permitted"
	blockedNoRight     = "no_right"
	blockedLostRight   = "requester_lost_right"
	integrationClosed  = "closed"
	integrationGrantID = "integration-"
)

var integrationRight = map[string]gate.Action{
	pluginProposeKind:     gate.AdministerModules,
	pluginWithdrawKind:    gate.AdministerModules,
	pluginApproveKind:     gate.ApproveModules,
	pluginStopKind:        gate.StopModules,
	pluginDisableKind:     gate.StopModules,
	subProposeKind:        gate.AdministerChannels,
	subRequestKind:        gate.AdministerChannels,
	subWithdrawKind:       gate.AdministerChannels,
	resumeWithdrawKind:    gate.AdministerChannels,
	subApproveKind:        gate.ApproveChannels,
	subResumeKind:         gate.ApproveChannels,
	subPauseKind:          gate.StopChannels,
	subRevokeKind:         gate.StopChannels,
	letterRedeliverKind:   gate.ReprocessEvents,
	letterAgainKind:       gate.ReprocessEvents,
	letterDismissKind:     gate.ReprocessEvents,
	letterDropKind:        gate.ReprocessEvents,
	redeliverWithdrawKind: gate.ReprocessEvents,
	dismissWithdrawKind:   gate.ReprocessEvents,
	reapproveKind:         gate.ReprocessEvents,
}

var authorKinds = map[string]bool{pluginWithdrawKind: true, subWithdrawKind: true, resumeWithdrawKind: true, redeliverWithdrawKind: true, dismissWithdrawKind: true}

var moduleRights = []gate.Action{gate.AdministerModules, gate.ApproveModules, gate.StopModules}

var channelRights = []gate.Action{gate.AdministerChannels, gate.ApproveChannels, gate.StopChannels, gate.ReprocessEvents}

func rightOf(kind string) gate.Action {
	right, ok := integrationRight[kind]
	if !ok {
		panic("integration command without a right: " + kind)
	}
	return right
}

type standing int

const (
	qualifies standing = iota
	lost
	unknown
)

var starterOperator = identity.Operator{Reference: "system:integration-starter", Reason: "integration starter recheck"}

func (s *Server) standingOf(user string, right gate.Action, now time.Time) (standing, commands.Code) {
	if s.faulty("starter-authority") {
		return unknown, commands.AuthorityUnavailable
	}
	explained := s.explainRight(user, right, now)
	if explained.Code == gate.AuthorityUnavailable {
		return unknown, commands.AuthorityUnavailable
	}
	if s.faulty("starter-identity") || s.faulty("starter-identity-"+user) {
		return unknown, commands.DependencyUnavailable
	}
	info, found, err := s.findUser(user)
	if err != nil {
		return unknown, commands.DependencyUnavailable
	}
	member, joined, err := s.identity.Installation().Member(starterOperator, s.bootWorkspace, user)
	if err != nil {
		return unknown, commands.DependencyUnavailable
	}
	if found && joined && explained.Allowed && info.State == identity.Active && member.Effective &&
		member.Role == identity.Administrator {
		return qualifies, ""
	}
	return lost, ""
}

type viewer struct {
	s      *Server
	id     string
	rights map[gate.Action]bool
	now    time.Time
	memo   map[string]standing
}

func (v *viewer) has(right gate.Action) bool { return v.rights[right] }

func (v *viewer) any(rights []gate.Action) bool {
	for _, right := range rights {
		if v.rights[right] {
			return true
		}
	}
	return false
}

func (v *viewer) lost(user string, right gate.Action) bool {
	key := user + "/" + string(right)
	got, seen := v.memo[key]
	if !seen {
		got, _ = v.s.standingOf(user, right, v.now)
		v.memo[key] = got
	}
	return user != "" && got == lost
}

func (v *viewer) blocked(kind, state string) string {
	if !v.has(rightOf(kind)) {
		return blockedNoRight
	}
	return state
}

func (v *viewer) approval(kind, starter string, pending bool, state string) string {
	switch {
	case !v.has(rightOf(kind)):
		return blockedNoRight
	case pending && v.lost(starter, starterRight(kind)):
		return blockedLostRight
	}
	return state
}

func starterRight(kind string) gate.Action {
	switch kind {
	case pluginApproveKind:
		return gate.AdministerModules
	case subApproveKind, subResumeKind:
		return gate.AdministerChannels
	}
	return gate.ReprocessEvents
}

func (s *Server) admitIntegrations(w http.ResponseWriter, r *http.Request) (session, identity.Answer, *viewer, bool) {
	request := r.Header.Get("X-Request-Id")
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return found, identity.Answer{}, nil, false
	}
	if !sameToken(r.Header.Get(csrfHeader), found.csrf) {
		fail(w, http.StatusForbidden, CodeCSRF, request)
		return found, identity.Answer{}, nil, false
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return found, answer, nil, false
	}
	now := s.clock()
	v := &viewer{s: s, id: answer.User(), rights: map[gate.Action]bool{}, now: now, memo: map[string]standing{}}
	decide := func(right gate.Action) (bool, bool) {
		decision := s.gate.Decide(gate.Answer(answer), ConsoleClient, gate.Request{Action: right, Kind: gate.InstallationKind,
			Resource: gate.Resource{WorkspaceID: answer.Workspace()}})
		return decision.Allowed, decision.Code != gate.AuthorityUnavailable && !s.faulty("integrations-authority-"+string(right))
	}
	admitted, known := decide(gate.AdministerInstallation)
	for _, right := range gate.IntegrationRights() {
		allowed, sure := decide(right)
		v.rights[right], admitted, known = allowed, admitted || allowed, known && sure
	}
	if !known {
		failCode(w, commands.AuthorityUnavailable, request)
		return found, answer, nil, false
	}
	if !admitted {
		failCode(w, commands.PermissionDenied, request)
		return found, answer, nil, false
	}
	return found, answer, v, true
}

func (v *viewer) rightsRow() map[string]bool {
	out := map[string]bool{}
	for _, right := range gate.IntegrationRights() {
		out[string(right)] = v.rights[right]
	}
	return out
}

type withheld struct {
	State string `json:"state"`
}

type requestRecord struct {
	id     string
	kind   string
	author string
	object string
	parent string
}

type requestBook struct {
	records map[string]*requestRecord
	order   map[string][]string
	counter int
	limit   int
}

func newRequestBook() *requestBook {
	return &requestBook{records: map[string]*requestRecord{}, order: map[string][]string{}, limit: requestTombstone}
}

func (c *integrationStore) newRequest(kind, author, object, parent string) letterRequest {
	b := c.requests
	b.counter++
	prefix := "letter-request-"
	if kind == requestResume {
		prefix = "resume-request-"
	}
	rec := &requestRecord{id: prefix + strconv.Itoa(b.counter), kind: kind, author: author, object: object, parent: parent}
	b.records[rec.id] = rec
	b.order[kind] = append(b.order[kind], rec.id)
	for len(b.order[kind]) > b.limit {
		evicted := false
		for index, id := range b.order[kind] {
			if id != rec.id && !c.liveRequest(b.records[id]) {
				delete(b.records, id)
				b.order[kind] = append(b.order[kind][:index:index], b.order[kind][index+1:]...)
				evicted = true
				break
			}
		}
		if !evicted {
			break
		}
	}
	return letterRequest{id: rec.id, by: author}
}

func (c *integrationStore) liveRequest(rec *requestRecord) bool {
	switch rec.kind {
	case requestResume:
		e := c.subscription(rec.object)
		return e != nil && e.resume != nil && e.resume.id == rec.id
	case requestRedeliver:
		return c.redeliver[rec.object].id == rec.id
	}
	return c.dismiss[rec.object].id == rec.id
}

func (c *integrationStore) record(target, prefix string) *requestRecord {
	id, found := strings.CutPrefix(target, prefix)
	if !found {
		return nil
	}
	return c.requests.records[id]
}

func requestKindOf(kind string) string {
	switch kind {
	case resumeWithdrawKind:
		return requestResume
	case redeliverWithdrawKind:
		return requestRedeliver
	}
	return requestDismiss
}

type ownPendingRow struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Target string `json:"target"`
	Object string `json:"object"`
	State  string `json:"state"`
}

func (s *Server) ownPending(reader string) []ownPendingRow {
	c := s.integ
	out := []ownPendingRow{}
	for _, e := range c.entries {
		if e.view.State == plugins.ProposalPending && e.view.Proposer == reader {
			number := strconv.FormatUint(e.view.Number, 10)
			out = append(out, ownPendingRow{Kind: "plugin_proposal", ID: number, Target: pluginProposalTarget + number, Object: pluginSummary(e.view.Change).Plugin,
				State: "pending"})
		}
	}
	for _, e := range c.subscriptions {
		if e.proposed && e.proposer == reader {
			out = append(out, ownPendingRow{Kind: "subscription_proposal", ID: e.id, Target: subscriptionTarget + e.id, Object: e.id, State: "pending"})
		}
		if e.resume != nil && e.resume.by == reader {
			out = append(out, ownPendingRow{Kind: "resume_request", ID: e.resume.id, Target: resumeRequestTarget + e.resume.id, Object: e.id, State: "pending"})
		}
	}
	for _, id := range c.requests.order[requestRedeliver] {
		if rec := c.requests.records[id]; rec != nil && rec.author == reader && c.liveRequest(rec) {
			out = append(out, ownPendingRow{Kind: "redeliver_request", ID: id, Target: letterRequestTarget + id, Object: rec.object, State: "pending"})
		}
	}
	for _, id := range c.requests.order[requestDismiss] {
		if rec := c.requests.records[id]; rec != nil && rec.author == reader && c.liveRequest(rec) {
			out = append(out, ownPendingRow{Kind: "dismiss_request", ID: id, Target: letterRequestTarget + id, Object: rec.object, State: "pending"})
		}
	}
	return out
}

func (s *Server) grantIntegration(user string, rights []gate.Action, now time.Time) error {
	if s.faulty("integration-grant") {
		return errPartial
	}
	scope := gate.Scope{WorkspaceID: s.Workspace(), Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}
	validity := gate.Validity{NotBefore: now, NotAfter: now.Add(installationGrants)}
	for _, right := range rights {
		terms := gate.Terms{Action: right, Kind: gate.InstallationKind, Period: gate.AnyPeriod(), Scope: scope}
		err := s.admin.AddGrant(gate.Grant{ID: integrationGrantID + string(right) + "-console", ClientID: ConsoleClient, Terms: terms, Validity: validity})
		if err != nil && !errors.Is(err, gate.ErrConflict) {
			return err
		}
	}
	added := []string{}
	for index, right := range rights {
		terms := gate.Terms{Action: right, Kind: gate.InstallationKind, Period: gate.AnyPeriod(), Scope: scope}
		id := integrationGrantID + string(right) + "-" + user
		err := s.admin.AddGrant(gate.Grant{ID: id, SubjectID: user, Terms: terms, Validity: validity})
		if err == nil && s.faulty("integration-grant-"+strconv.Itoa(index)) {
			added, err = append(added, id), errPartial
		}
		if err != nil {
			return errors.Join(err, s.rollbackGrants(added))
		}
		added = append(added, id)
	}
	return nil
}

var errRollback = errors.New("integration grants rolled back incompletely")

func (s *Server) rollbackGrants(ids []string) error {
	failed := false
	for _, id := range ids {
		version := uint64(0)
		for _, g := range s.admin.Grants() {
			if g.ID == id {
				version = g.Version
			}
		}
		if s.faulty("integration-rollback") || s.admin.RevokeGrant(id, version) != nil {
			failed = true
		}
	}
	if failed {
		return errRollback
	}
	return nil
}

var operatorRights = []gate.Action{gate.StopModules, gate.StopChannels, gate.ReprocessEvents}

func (s *Server) seedIntegrationGrants(now time.Time) error {
	return run(func() error { return s.grantIntegration(SampleAdmin, gate.IntegrationRights(), now) },
		func() error { return s.grantIntegration(SampleAdmin2, gate.IntegrationRights(), now) },
		func() error { return s.grantIntegration(SampleAdmin3, operatorRights, now) })
}
