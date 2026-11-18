package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

const (
	stateConfigured    = "configured"
	stateNotConfigured = "not_configured"
	notConnected       = "not_connected"
	notIssued          = "not_issued"
	alreadyRequested   = "already_requested"
	notRequested       = "not_requested"
	ownRequest         = "own_request"
	notDead            = "not_dead"
	shownDeliveries    = 50
)

type availability struct {
	Allowed bool   `json:"allowed"`
	Blocked string `json:"blocked"`
}

func allow(blocked string) availability {
	return availability{Allowed: blocked == "", Blocked: blocked}
}

type clientGrantRow struct {
	ID        string   `json:"id"`
	Action    string   `json:"action"`
	Kind      string   `json:"kind"`
	Scope     scopeRow `json:"scope"`
	NotBefore string   `json:"not_before"`
	NotAfter  string   `json:"not_after"`
	State     string   `json:"state"`
}

type clientRow struct {
	ID     string           `json:"id"`
	Kind   string           `json:"kind"`
	State  string           `json:"state"`
	Grants []clientGrantRow `json:"grants"`
}

type serviceUserRow struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	Owner       string `json:"owner"`
	OwnerLogin  string `json:"owner_login"`
	Credentials int    `json:"usable_credentials"`
}

type clientsSection struct {
	State        string           `json:"state"`
	Clients      []clientRow      `json:"clients"`
	ServiceUsers []serviceUserRow `json:"service_users"`
	ManageOn     string           `json:"manage_on"`
	OAuthClients string           `json:"oauth_clients"`
}

type pluginWorkspaceRow struct {
	ID           string `json:"id"`
	Runtime      string `json:"runtime"`
	Verification bool   `json:"verification"`
}

type publisherKeyRow struct {
	Version    string          `json:"version"`
	NotBefore  string          `json:"not_before"`
	NotAfter   string          `json:"not_after"`
	Retired    bool            `json:"retired"`
	Compromise *installMarkRow `json:"compromise"`
}

type publisherRow struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Revocation *installMarkRow   `json:"revocation"`
	Keys       []publisherKeyRow `json:"keys"`
}

type screenFieldRow struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Source string `json:"source"`
}

type screenRow struct {
	ID        string           `json:"id"`
	Component string           `json:"component"`
	Label     string           `json:"label"`
	Fields    []screenFieldRow `json:"fields"`
}

type storageRow struct {
	Namespace     string `json:"namespace"`
	Owner         string `json:"owner"`
	RetentionDays int    `json:"retention_days"`
	Deletion      string `json:"deletion"`
	Migration     string `json:"migration"`
}

type pluginVersionRow struct {
	Plugin       string          `json:"plugin"`
	Version      string          `json:"version"`
	Publisher    string          `json:"publisher"`
	Key          string          `json:"key_version"`
	Digest       string          `json:"digest"`
	CoreMin      string          `json:"core_min"`
	CoreMax      string          `json:"core_max"`
	Capabilities []string        `json:"capabilities"`
	Egress       []string        `json:"egress"`
	Endpoints    []string        `json:"endpoints"`
	Timeout      int64           `json:"timeout_seconds"`
	MemoryMiB    int             `json:"memory_mib"`
	Concurrency  int             `json:"concurrency"`
	Storage      storageRow      `json:"storage"`
	Screens      []screenRow     `json:"screens"`
	Vulnerable   *installMarkRow `json:"vulnerable"`
}

type endpointRow struct {
	ID            string `json:"id"`
	Fingerprint   string `json:"fingerprint"`
	Approver      string `json:"approver"`
	ApproverLogin string `json:"approver_login"`
}

type pluginInstallRow struct {
	Plugin     string        `json:"plugin"`
	Version    string        `json:"version"`
	State      string        `json:"state"`
	Reason     string        `json:"reason"`
	Revision   uint64        `json:"revision"`
	Endpoints  []endpointRow `json:"endpoints"`
	Disposal   *storageRow   `json:"disposal"`
	CanStop    bool          `json:"can_stop"`
	StopBlock  string        `json:"stop_blocked"`
	CanDisable bool          `json:"can_disable"`
	DisBlock   string        `json:"disable_blocked"`
	Tokens     string        `json:"tokens"`
	Runtime    string        `json:"runtime"`
}

type pluginSummaryRow struct {
	Publisher     string `json:"publisher"`
	PublisherName string `json:"publisher_name"`
	Key           string `json:"key_version"`
	KeyNotBefore  string `json:"key_not_before"`
	KeyNotAfter   string `json:"key_not_after"`
	Plugin        string `json:"plugin"`
	Version       string `json:"version"`
	Workspace     string `json:"workspace"`
	Runtime       string `json:"runtime"`
	Endpoint      string `json:"endpoint"`
	Fingerprint   string `json:"fingerprint"`
}

type pluginProposalRow struct {
	Number           uint64           `json:"number"`
	Target           string           `json:"target"`
	Kind             string           `json:"kind"`
	State            string           `json:"state"`
	Created          string           `json:"created_at"`
	Proposer         string           `json:"proposer"`
	ProposerLogin    string           `json:"proposer_login"`
	Approver         string           `json:"approver"`
	ApproverLogin    string           `json:"approver_login"`
	Rationale        string           `json:"rationale"`
	ClosedBy         string           `json:"closed_by"`
	ClosedByLogin    string           `json:"closed_by_login"`
	ClosingRationale string           `json:"closing_rationale"`
	Summary          pluginSummaryRow `json:"summary"`
	CanApprove       bool             `json:"can_approve"`
	ApproveBlocked   string           `json:"approve_blocked"`
	CanWithdraw      bool             `json:"can_withdraw"`
	WithdrawBlocked  string           `json:"withdraw_blocked"`
}

type modulesSection struct {
	State         string              `json:"state"`
	Workspace     *pluginWorkspaceRow `json:"workspace"`
	Publishers    []publisherRow      `json:"publishers"`
	Plugins       []pluginVersionRow  `json:"plugins"`
	Installations []pluginInstallRow  `json:"installations"`
	Proposals     []pluginProposalRow `json:"proposals"`
}

type destinationRow struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Source string `json:"source"`
}

type subscriptionKeyRow struct {
	Version       string `json:"version"`
	NotBefore     string `json:"not_before"`
	NotAfter      string `json:"not_after"`
	State         string `json:"state"`
	Proposer      string `json:"proposer"`
	ProposerLogin string `json:"proposer_login"`
	Approver      string `json:"approver"`
	ApproverLogin string `json:"approver_login"`
}

type gapRow struct {
	After uint64 `json:"after"`
	To    uint64 `json:"to"`
}

type subscriptionHistoryRow struct {
	Action     string `json:"action"`
	Actor      string `json:"actor"`
	ActorLogin string `json:"actor_login"`
	Rationale  string `json:"rationale"`
}

type subscriptionRow struct {
	ID            string                   `json:"id"`
	Target        string                   `json:"target"`
	Subscriber    string                   `json:"subscriber"`
	Destination   string                   `json:"destination"`
	Types         []string                 `json:"types"`
	State         string                   `json:"state"`
	Reason        string                   `json:"reason"`
	Cursor        uint64                   `json:"cursor"`
	Lag           uint64                   `json:"lag"`
	Gaps          []gapRow                 `json:"gaps"`
	Created       string                   `json:"created_at"`
	Rationale     string                   `json:"rationale"`
	Proposer      string                   `json:"proposer"`
	ProposerLogin string                   `json:"proposer_login"`
	Approver      string                   `json:"approver"`
	ApproverLogin string                   `json:"approver_login"`
	Resumer       string                   `json:"resumer"`
	ResumerLogin  string                   `json:"resumer_login"`
	History       []subscriptionHistoryRow `json:"history"`
	Keys          []subscriptionKeyRow     `json:"keys"`
	Signer        string                   `json:"signer"`
	Actions       map[string]availability  `json:"actions"`
}

type channelsSection struct {
	State         string            `json:"state"`
	Destinations  []destinationRow  `json:"destinations"`
	Subscribers   []string          `json:"subscribers"`
	Subscriptions []subscriptionRow `json:"subscriptions"`
}

type typeRow struct {
	Name          string   `json:"name"`
	SchemaVersion string   `json:"schema_version"`
	Fields        []string `json:"fields"`
	State         string   `json:"state"`
	Proposer      string   `json:"proposer"`
	ProposerLogin string   `json:"proposer_login"`
	Approver      string   `json:"approver"`
	ApproverLogin string   `json:"approver_login"`
}

type requestRow struct {
	By        string `json:"by"`
	ByLogin   string `json:"by_login"`
	Rationale string `json:"rationale"`
}

type deliveryRow struct {
	ID             string                  `json:"id"`
	Target         string                  `json:"target"`
	Subscription   string                  `json:"subscription"`
	Sequence       uint64                  `json:"sequence"`
	Source         string                  `json:"source"`
	EventID        string                  `json:"event_id"`
	Kind           string                  `json:"kind"`
	State          string                  `json:"state"`
	Reason         string                  `json:"reason"`
	Attempts       int                     `json:"attempts"`
	OutcomeUnknown bool                    `json:"outcome_unknown"`
	Redeliver      *requestRow             `json:"redeliver_request"`
	Dismiss        *requestRow             `json:"dismiss_request"`
	Actions        map[string]availability `json:"actions"`
}

type outboxRow struct {
	Floor uint64 `json:"floor"`
	Head  uint64 `json:"head"`
}

type eventsSection struct {
	State      string        `json:"state"`
	Types      []typeRow     `json:"types"`
	Outbox     outboxRow     `json:"outbox"`
	Deliveries []deliveryRow `json:"deliveries"`
	Dispatcher string        `json:"dispatcher"`
}

type integrationsPage struct {
	Revision string          `json:"revision"`
	Now      string          `json:"now"`
	State    string          `json:"state"`
	Clients  clientsSection  `json:"clients"`
	Modules  modulesSection  `json:"modules"`
	Channels channelsSection `json:"channels"`
	Events   eventsSection   `json:"events"`
}

func configured(present bool) string {
	if present {
		return stateConfigured
	}
	return stateNotConfigured
}

func (s *Server) markOf(m *plugins.Mark) *installMarkRow {
	if m == nil {
		return nil
	}
	return &installMarkRow{By: m.By, ByLogin: s.loginOf(m.By), Reason: m.Reason, ConfirmedBy: m.ConfirmedBy, ConfirmedByLogin: s.loginOf(m.ConfirmedBy)}
}

func (s *Server) clientsOf(verifier identity.Verifier, now time.Time) (clientsSection, error) {
	console := clientRow{ID: ConsoleClient, Kind: "first_party_console", State: "active", Grants: []clientGrantRow{}}
	for _, g := range s.admin.Grants() {
		if g.ClientID != ConsoleClient || gate.InstallationAction(g.Terms.Action) {
			continue
		}
		console.Grants = append(console.Grants, clientGrantRow{ID: g.ID, Action: string(g.Terms.Action), Kind: g.Terms.Kind, Scope: scopeOf(g.Terms),
			NotBefore: stamp(g.Validity.NotBefore), NotAfter: stamp(g.Validity.NotAfter), State: stateOf(g.Closed, g.Validity, now)})
	}
	out := clientsSection{State: stateConfigured, Clients: []clientRow{console}, ServiceUsers: []serviceUserRow{}, ManageOn: "M04", OAuthClients: notConnected}
	services, err := s.serviceUsers(verifier)
	for _, svc := range services {
		if err != nil {
			break
		}
		var rows []credentialRow
		rows, err = s.credentialRows(verifier, svc, now)
		usable := 0
		for _, c := range rows {
			if c.Usable {
				usable++
			}
		}
		out.ServiceUsers = append(out.ServiceUsers, serviceUserRow{ID: svc.ID, State: serviceState(svc.State), Owner: svc.Owner, OwnerLogin: s.loginOf(svc.Owner),
			Credentials: usable})
	}
	return out, err
}

func (s *Server) publisherOf(id string) (publisherRow, error) {
	var v plugins.PublisherView
	if err := s.pluginCall("publisher", func() (err error) { v, err = s.plugins.Publisher(id); return }); err != nil {
		return publisherRow{}, err
	}
	row := publisherRow{ID: v.Publisher.ID, Name: v.Publisher.Name, Revocation: s.markOf(v.Revocation), Keys: []publisherKeyRow{}}
	for _, k := range v.Keys {
		row.Keys = append(row.Keys, publisherKeyRow{Version: k.Version, NotBefore: stamp(k.NotBefore), NotAfter: stamp(k.NotAfter), Retired: k.Retired,
			Compromise: s.markOf(k.Compromise)})
	}
	return row, nil
}

func storageOf(st plugins.Storage) storageRow {
	return storageRow{Namespace: st.Namespace, Owner: st.Owner, RetentionDays: st.RetentionDays, Deletion: st.Deletion, Migration: st.Migration}
}

func (s *Server) versionOf(plugin, version string) (pluginVersionRow, error) {
	var v plugins.VersionView
	if err := s.pluginCall("version", func() (err error) { v, err = s.plugins.Version(plugin, version); return }); err != nil {
		return pluginVersionRow{}, err
	}
	m := v.Manifest
	row := pluginVersionRow{Plugin: m.Plugin, Version: m.Version, Publisher: m.Publisher, Key: m.Key, Digest: m.Digest, CoreMin: m.Core.Min, CoreMax: m.Core.Max,
		Capabilities: []string{}, Egress: append([]string{}, m.Egress...), Endpoints: append([]string{}, m.Endpoints...), Timeout: int64(m.Limits.Timeout / time.Second),
		MemoryMiB: m.Limits.MemoryMiB, Concurrency: m.Limits.Concurrency, Storage: storageOf(m.Storage), Screens: []screenRow{}, Vulnerable: s.markOf(v.Vulnerable)}
	for _, c := range m.Capabilities {
		row.Capabilities = append(row.Capabilities, c.Kind+":"+c.Target)
	}
	for _, sc := range m.Screens {
		screen := screenRow{ID: sc.ID, Component: sc.Component, Label: sc.Label, Fields: []screenFieldRow{}}
		for _, f := range sc.Fields {
			screen.Fields = append(screen.Fields, screenFieldRow{Name: f.Name, Label: f.Label, Source: f.Source})
		}
		row.Screens = append(row.Screens, screen)
	}
	return row, nil
}

func (s *Server) installationOf(plugin string) (plugins.InstallationView, error) {
	var v plugins.InstallationView
	err := s.pluginCall("installation", func() (err error) { v, err = s.plugins.Installation(s.integ.workspace, plugin); return })
	return v, err
}

func (s *Server) installRowOf(v plugins.InstallationView) pluginInstallRow {
	row := pluginInstallRow{Plugin: v.Plugin, Version: v.Version, State: v.State, Reason: v.Reason, Revision: v.Revision, Endpoints: []endpointRow{},
		Tokens: notIssued, Runtime: notConnected}
	for _, e := range v.Endpoints {
		row.Endpoints = append(row.Endpoints, endpointRow{ID: e.Endpoint.ID, Fingerprint: e.Endpoint.Fingerprint, Approver: e.Approver, ApproverLogin: s.loginOf(e.Approver)})
	}
	if v.State == plugins.StateRemoved {
		disposal := storageOf(v.Disposal)
		row.Disposal = &disposal
	}
	if v.State != plugins.StateEnabled {
		row.StopBlock = plugins.ReasonNotEnabled
	}
	if v.State != plugins.StateEnabled && v.State != plugins.StateStopped {
		row.DisBlock = plugins.ReasonNotAllowedState
	}
	row.CanStop, row.CanDisable = row.StopBlock == "", row.DisBlock == ""
	return row
}

func recheckReason(err error) (string, error) {
	var refusal plugins.Refusal
	switch {
	case err == nil:
		return "", nil
	case errors.As(err, &refusal):
		return refusal.Reason, nil
	case errors.Is(err, plugins.ErrNotFound):
		return "not_found", nil
	case errors.Is(err, plugins.ErrConflict):
		return "conflict", nil
	case errors.Is(err, plugins.ErrLimit):
		return "limit", nil
	}
	return "", err
}

func (s *Server) pluginBlockers(v plugins.ProposalView, reader string) (string, string, error) {
	if v.State != plugins.ProposalPending {
		return plugins.ReasonNotPending, plugins.ReasonNotPending, nil
	}
	withdraw := ""
	if v.Proposer != reader {
		withdraw = plugins.ReasonNotProposer
	}
	if v.Proposer == reader {
		return ownProposal, withdraw, nil
	}
	approve, err := recheckReason(s.pluginCall("recheck", func() error { return s.plugins.Recheck(v.Number) }))
	return approve, withdraw, err
}

func pluginSummary(c plugins.Change) pluginSummaryRow {
	row := pluginSummaryRow{Publisher: firstOf(c.Publisher.ID, c.Key.Publisher, c.Manifest.Publisher), PublisherName: c.Publisher.Name, Key: c.Key.Version,
		Plugin: firstOf(c.Plugin, c.Manifest.Plugin), Version: firstOf(c.Version, c.Manifest.Version), Workspace: c.Workspace.ID, Runtime: c.Workspace.Runtime,
		Endpoint: c.Endpoint.ID, Fingerprint: c.Endpoint.Fingerprint}
	if c.Kind == plugins.ChangeKey {
		row.KeyNotBefore, row.KeyNotAfter = stamp(c.Key.NotBefore), stamp(c.Key.NotAfter)
	}
	return row
}

func (s *Server) pluginProposalOf(e *pluginEntry, reader string) (pluginProposalRow, error) {
	v := e.view
	row := pluginProposalRow{Number: v.Number, Target: pluginProposalTarget + strconv.FormatUint(v.Number, 10), Kind: v.Change.Kind, State: v.State,
		Created: stamp(e.created), Proposer: v.Proposer, ProposerLogin: s.loginOf(v.Proposer), Approver: v.Approver, ApproverLogin: s.loginOf(v.Approver),
		Rationale: e.rationale, ClosedBy: e.closedBy, ClosedByLogin: s.loginOf(e.closedBy), ClosingRationale: e.closing, Summary: pluginSummary(v.Change)}
	var err error
	row.ApproveBlocked, row.WithdrawBlocked, err = s.pluginBlockers(v, reader)
	row.CanApprove, row.CanWithdraw = row.ApproveBlocked == "", row.WithdrawBlocked == ""
	return row, err
}

func (s *Server) modulesOf(reader string) (modulesSection, error) {
	c := s.integ
	out := modulesSection{State: configured(len(c.publishers) > 0 || len(c.plugins) > 0 || c.workspace != ""), Publishers: []publisherRow{},
		Plugins: []pluginVersionRow{}, Installations: []pluginInstallRow{}, Proposals: []pluginProposalRow{}}
	errs := []error{}
	if c.workspace != "" {
		var w plugins.Workspace
		err := s.pluginCall("workspace", func() (err error) { w, err = s.plugins.Workspace(c.workspace); return })
		out.Workspace, errs = &pluginWorkspaceRow{ID: w.ID, Runtime: w.Runtime, Verification: w.Verification}, append(errs, err)
	}
	for _, id := range c.publishers {
		row, err := s.publisherOf(id)
		out.Publishers, errs = append(out.Publishers, row), append(errs, err)
	}
	for _, p := range c.plugins {
		for _, version := range p.versions {
			row, err := s.versionOf(p.id, version)
			out.Plugins, errs = append(out.Plugins, row), append(errs, err)
		}
		if c.workspace == "" {
			continue
		}
		v, err := s.installationOf(p.id)
		if errors.Is(err, plugins.ErrNotFound) {
			continue
		}
		out.Installations, errs = append(out.Installations, s.installRowOf(v)), append(errs, err)
	}
	for index := len(c.entries) - 1; index >= 0 && len(out.Proposals) < installShown; index-- {
		row, err := s.pluginProposalOf(c.entries[index], reader)
		out.Proposals, errs = append(out.Proposals, row), append(errs, err)
	}
	return out, errors.Join(errs...)
}

func (s *Server) admits(view webhooks.SubscriptionView) string {
	sub := view.Subscription
	state, err := integrationDestinations{s: s}.State(s.bootWorkspace, sub.Destination)
	if err != nil {
		return webhooks.ReasonDependency
	}
	if state != webhooks.DestinationActive {
		return webhooks.ReasonDestinationInactive
	}
	allowed, err := integrationPermits{s: s}.Subscribe(s.bootWorkspace, sub.Subscriber, sub.Types)
	switch {
	case err != nil:
		return webhooks.ReasonDependency
	case !allowed:
		return webhooks.ReasonSubscribeRefused
	}
	return ""
}

func pausedOrSuspended(state string) bool {
	return state == webhooks.SubscriptionPaused || state == webhooks.SubscriptionSuspended
}

func (s *Server) subscriptionActions(v webhooks.SubscriptionView, reader string) map[string]availability {
	approve, pause, request, resume, revoke := "", "", "", "", ""
	switch {
	case v.State != webhooks.SubscriptionProposed:
		approve = plugins.ReasonNotPending
	case v.Proposer == reader:
		approve = ownProposal
	default:
		approve = s.admits(v)
	}
	if v.State != webhooks.SubscriptionActive {
		pause = webhooks.ReasonNotActive
	}
	switch {
	case !pausedOrSuspended(v.State):
		request = webhooks.ReasonNotPending
	case v.Resumer != "":
		request = alreadyRequested
	}
	switch {
	case v.Resumer == "" || !pausedOrSuspended(v.State):
		resume = webhooks.ReasonNotPending
	case v.Resumer == reader:
		resume = ownRequest
	default:
		resume = s.admits(v)
	}
	if v.State == webhooks.SubscriptionRevoked {
		revoke = webhooks.ReasonNotPending
	}
	return map[string]availability{"approve": allow(approve), "pause": allow(pause), "request_resume": allow(request), "approve_resume": allow(resume),
		"revoke": allow(revoke)}
}

func (s *Server) subscriptionOf(id string) (webhooks.SubscriptionView, error) {
	var v webhooks.SubscriptionView
	err := s.webhookCall("subscription", func() (err error) { v, err = s.webhooks.SubscriptionOf(s.bootWorkspace, id); return })
	return v, err
}

func (s *Server) subscriptionRowOf(e *subscriptionEntry, v webhooks.SubscriptionView, head uint64, reader string) (subscriptionRow, error) {
	sub := v.Subscription
	row := subscriptionRow{ID: sub.ID, Target: subscriptionTarget + sub.ID, Subscriber: sub.Subscriber, Destination: sub.Destination, Types: append([]string{}, sub.Types...),
		State: v.State, Reason: v.Reason, Cursor: v.Cursor, Gaps: []gapRow{}, Created: stamp(e.created), Rationale: e.rationale, Proposer: v.Proposer,
		ProposerLogin: s.loginOf(v.Proposer), Approver: v.Approver, ApproverLogin: s.loginOf(v.Approver), Resumer: v.Resumer, ResumerLogin: s.loginOf(v.Resumer),
		History: []subscriptionHistoryRow{}, Keys: []subscriptionKeyRow{}, Signer: notConnected, Actions: s.subscriptionActions(v, reader)}
	if head > v.Cursor {
		row.Lag = head - v.Cursor
	}
	for _, g := range v.Gaps {
		row.Gaps = append(row.Gaps, gapRow{After: g.After, To: g.To})
	}
	for _, h := range e.history {
		row.History = append(row.History, subscriptionHistoryRow{Action: h.action, Actor: h.actor, ActorLogin: s.loginOf(h.actor), Rationale: h.rationale})
	}
	var keys []webhooks.KeyView
	err := s.webhookCall("keys", func() (err error) {
		keys, err = s.webhooks.Keys(s.bootWorkspace, sub.ID, webhooks.Actor{ID: reader, Kind: webhooks.KindHuman})
		return
	})
	for _, k := range keys {
		row.Keys = append(row.Keys, subscriptionKeyRow{Version: k.Key.Version, NotBefore: stamp(k.Key.NotBefore), NotAfter: stamp(k.Key.NotAfter), State: k.State,
			Proposer: k.Proposer, ProposerLogin: s.loginOf(k.Proposer), Approver: k.Approver, ApproverLogin: s.loginOf(k.Approver)})
	}
	return row, err
}

func (s *Server) requestOf(r letterRequest, found bool) *requestRow {
	if !found {
		return nil
	}
	return &requestRow{By: r.by, ByLogin: s.loginOf(r.by), Rationale: r.rationale}
}

func letterActions(d webhooks.DeliveryView, state string, redeliver, dismiss *requestRow, reader string) map[string]availability {
	if d.Kind != webhooks.KindDead {
		return map[string]availability{"request_redeliver": allow(notDead), "approve_redeliver": allow(notDead), "request_dismiss": allow(notDead),
			"approve_dismiss": allow(notDead)}
	}
	request := func(r *requestRow) string {
		if r != nil {
			return alreadyRequested
		}
		return ""
	}
	approve := func(r *requestRow) string {
		switch {
		case r == nil:
			return notRequested
		case r.By == reader:
			return ownRequest
		}
		return ""
	}
	again := approve(redeliver)
	if again == "" && state == webhooks.SubscriptionRevoked {
		again = webhooks.ReasonRevoked
	}
	return map[string]availability{"request_redeliver": allow(request(redeliver)), "approve_redeliver": allow(again), "request_dismiss": allow(request(dismiss)),
		"approve_dismiss": allow(approve(dismiss))}
}

func (s *Server) deliveriesOf(id, state, reader string) ([]deliveryRow, error) {
	var list []webhooks.DeliveryView
	err := s.webhookCall("deliveries", func() (err error) {
		list, err = s.webhooks.Deliveries(s.bootWorkspace, id, 0, shownDeliveries, webhooks.Actor{ID: reader, Kind: webhooks.KindHuman})
		return
	})
	out := []deliveryRow{}
	for _, d := range list {
		redeliver, dismiss := (*requestRow)(nil), (*requestRow)(nil)
		if d.Kind == webhooks.KindDead {
			r, found := s.integ.redeliver[d.ID]
			redeliver = s.requestOf(r, found)
			r, found = s.integ.dismiss[d.ID]
			dismiss = s.requestOf(r, found)
		}
		out = append(out, deliveryRow{ID: d.ID, Target: letterTarget + d.ID, Subscription: d.Subscription, Sequence: d.Sequence, Source: d.Source, EventID: d.EventID,
			Kind: d.Kind, State: d.State, Reason: d.Reason, Attempts: d.Attempts, OutcomeUnknown: d.OutcomeUnknown, Redeliver: redeliver, Dismiss: dismiss,
			Actions: letterActions(d, state, redeliver, dismiss, reader)})
	}
	return out, err
}

func (s *Server) channelsAndEvents(reader string) (channelsSection, eventsSection, error) {
	c := s.integ
	channels := channelsSection{State: configured(len(c.subscriptions) > 0), Destinations: []destinationRow{}, Subscribers: []string{sampleSubscriber},
		Subscriptions: []subscriptionRow{}}
	feed := eventsSection{State: configured(len(c.types) > 0), Types: []typeRow{}, Deliveries: []deliveryRow{}, Dispatcher: notConnected}
	for _, d := range webhookDestinations {
		channels.Destinations = append(channels.Destinations, destinationRow{ID: d.id, State: d.state, Source: allowlistSource})
	}
	for _, t := range c.types {
		feed.Types = append(feed.Types, typeRow{Name: t.name, SchemaVersion: t.version, Fields: append([]string{}, t.fields...), State: "active", Proposer: t.proposer,
			ProposerLogin: s.loginOf(t.proposer), Approver: t.approver, ApproverLogin: s.loginOf(t.approver)})
	}
	floor, head, err := newOutboxReader(s).Bounds(s.bootWorkspace)
	feed.Outbox = outboxRow{Floor: floor, Head: head}
	errs := []error{err}
	for _, e := range c.subscriptions {
		v, err := s.subscriptionOf(e.id)
		if errors.Is(err, webhooks.ErrNotFound) {
			continue
		}
		row, rerr := s.subscriptionRowOf(e, v, head, reader)
		rows, derr := s.deliveriesOf(e.id, v.State, reader)
		channels.Subscriptions, feed.Deliveries = append(channels.Subscriptions, row), append(feed.Deliveries, rows...)
		errs = append(errs, err, rerr, derr)
	}
	sort.SliceStable(feed.Deliveries, func(a, b int) bool { return feed.Deliveries[a].Sequence < feed.Deliveries[b].Sequence })
	return channels, feed, errors.Join(errs...)
}

func (s *Server) integrationsPage(verifier identity.Verifier, reader string, now time.Time) (integrationsPage, error) {
	if s.faulty("integrations-read") {
		return integrationsPage{}, errStandIn
	}
	page := integrationsPage{Revision: s.Revision(), Now: stamp(now), State: stateConfigured}
	var errs [3]error
	page.Clients, errs[0] = s.clientsOf(verifier, now)
	page.Modules, errs[1] = s.modulesOf(reader)
	page.Channels, page.Events, errs[2] = s.channelsAndEvents(reader)
	return page, errors.Join(errs[:]...)
}

func (s *Server) readIntegrations(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	now := s.clock()
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, installationEmpty{Revision: s.Revision(), Now: stamp(now), State: stateNotConfigured})
		return
	}
	page, err := s.integrationsPage(found.verifier, answer.User(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, page)
}
