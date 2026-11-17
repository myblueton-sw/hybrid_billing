package server

import (
	"errors"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

const (
	integrationCommand   = "integrations."
	pluginProposalTarget = "plugin-proposal-"
	pluginTarget         = "plugin:"
	webhooksTarget       = "webhooks"
	subscriptionTarget   = "webhook-subscription:"
	letterTarget         = "webhook-letter:"
	subscriptionPrefix   = "subscription-"
	maxSubscriptionTypes = 8

	pluginApproveKind   = "plugin_approve"
	pluginWithdrawKind  = "plugin_withdraw"
	pluginProposeKind   = "plugin_propose"
	pluginStopKind      = "plugin_stop"
	pluginDisableKind   = "plugin_disable"
	subProposeKind      = "subscription_propose"
	subApproveKind      = "subscription_approve"
	subPauseKind        = "subscription_pause"
	subRequestKind      = "subscription_request_resume"
	subResumeKind       = "subscription_approve_resume"
	subRevokeKind       = "subscription_revoke"
	letterRedeliverKind = "letter_request_redeliver"
	letterAgainKind     = "letter_approve_redeliver"
	letterDismissKind   = "letter_request_dismiss"
	letterDropKind      = "letter_approve_dismiss"
)

func mapIntegration(err error) error {
	var pluginRefusal plugins.Refusal
	var hookRefusal webhooks.Refusal
	reason := ""
	switch {
	case err == nil:
		return nil
	case errors.As(err, &pluginRefusal):
		reason = pluginRefusal.Reason
	case errors.As(err, &hookRefusal):
		reason = hookRefusal.Reason
	case errors.Is(err, plugins.ErrNotFound), errors.Is(err, webhooks.ErrNotFound), errors.Is(err, plugins.ErrConflict), errors.Is(err, webhooks.ErrConflict),
		errors.Is(err, plugins.ErrStale), errors.Is(err, webhooks.ErrStale):
		return refuse(commands.InvalidStage)
	case errors.Is(err, plugins.ErrLimit), errors.Is(err, webhooks.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	default:
		return refuse(commands.InternalError)
	}
	switch reason {
	case plugins.ReasonSelfApproval, plugins.ReasonNotProposer, webhooks.ReasonDestinationInactive, webhooks.ReasonSubscribeRefused:
		return refuse(commands.PrerequisiteFailed)
	case webhooks.ReasonDependency:
		return refuse(commands.DependencyUnavailable)
	case plugins.ReasonAINotAllowed, plugins.ReasonNotHuman:
		return refuse(commands.InternalError)
	}
	return refuse(commands.InvalidStage)
}

func (s *Server) pluginEntryOf(target string) *pluginEntry {
	raw, found := strings.CutPrefix(target, pluginProposalTarget)
	number, err := strconv.ParseUint(raw, 10, 64)
	if !found || err != nil || strconv.FormatUint(number, 10) != raw {
		return nil
	}
	return s.integ.entry(number)
}

func (s *Server) pluginOf(target string) *pluginRecord {
	id, found := strings.CutPrefix(target, pluginTarget)
	if !found {
		return nil
	}
	return s.integ.plugin(id)
}

func (s *Server) subscriptionEntryOf(target string) *subscriptionEntry {
	id, found := strings.CutPrefix(target, subscriptionTarget)
	if !found {
		return nil
	}
	return s.integ.subscription(id)
}

func letterOf(target string) string {
	id, _ := strings.CutPrefix(target, letterTarget)
	return id
}

func (s *Server) letterEntryOf(target string) *subscriptionEntry {
	if !strings.HasPrefix(target, letterTarget) {
		return nil
	}
	return s.integ.letter(letterOf(target))
}

type integrationHandler struct {
	s    *Server
	kind string
}

func (h integrationHandler) found(target string) bool {
	s := h.s
	switch {
	case h.kind == pluginApproveKind, h.kind == pluginWithdrawKind:
		return s.pluginEntryOf(target) != nil
	case strings.HasPrefix(h.kind, "plugin_"):
		return s.pluginOf(target) != nil
	case h.kind == subProposeKind:
		return target == webhooksTarget && len(s.integ.types) > 0
	case strings.HasPrefix(h.kind, "subscription_"):
		return s.subscriptionEntryOf(target) != nil
	}
	return s.letterEntryOf(target) != nil
}

func (h integrationHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !h.found(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

type integrationResult struct {
	state  string
	fields map[string]string
}

func proposalResult(e *pluginEntry) integrationResult {
	number := strconv.FormatUint(e.view.Number, 10)
	return integrationResult{state: e.view.State, fields: map[string]string{"proposal": number, "target": pluginProposalTarget + number}}
}

func (s *Server) pluginApproveCommand(ctx commands.Context, reader plugins.Actor) (integrationResult, error) {
	e := s.pluginEntryOf(ctx.Target)
	if e.view.State != plugins.ProposalPending {
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	if err := s.pluginApprove(e, reader, ctx.Rationale); err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return proposalResult(e), nil
}

func (s *Server) pluginWithdrawCommand(ctx commands.Context, reader plugins.Actor) (integrationResult, error) {
	e := s.pluginEntryOf(ctx.Target)
	if err := s.pluginWithdraw(e, reader, ctx.Rationale); err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return proposalResult(e), nil
}

func (s *Server) pluginProposeCommand(ctx commands.Context, reader plugins.Actor) (integrationResult, error) {
	p := s.pluginOf(ctx.Target)
	kind, version := ctx.Fields["kind"], ctx.Fields["version"]
	versioned := kind == plugins.ChangeInstall || kind == plugins.ChangeUpgrade || kind == plugins.ChangeRollback
	known := false
	for _, v := range p.versions {
		known = known || v == version
	}
	switch {
	case versioned != known, !versioned && version != "":
		return integrationResult{}, refuse(commands.InvalidRequest, "version")
	case s.integ.workspace == "":
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	e, err := s.pluginPropose(plugins.Change{Kind: kind, Workspace: plugins.Workspace{ID: s.integ.workspace}, Plugin: p.id, Version: version}, reader, ctx.Rationale)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return proposalResult(e), nil
}

func (s *Server) pluginRuntimeCommand(ctx commands.Context, reader plugins.Actor, stop bool) (integrationResult, error) {
	p := s.pluginOf(ctx.Target)
	current, err := s.installationOf(p.id)
	switch {
	case err != nil:
		return integrationResult{}, mapIntegration(err)
	case stop && current.State != plugins.StateEnabled, !stop && current.State == plugins.StateInstalled:
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	var v plugins.InstallationView
	if stop {
		err = s.pluginCall("stop", func() (err error) {
			v, err = s.plugins.Stop(s.integ.workspace, p.id, plugins.ReasonOperator, reader)
			return
		})
	} else {
		err = s.pluginCall("disable", func() (err error) { v, err = s.plugins.Disable(s.integ.workspace, p.id, reader); return })
	}
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return integrationResult{state: v.State, fields: map[string]string{"target": ctx.Target}}, nil
}

func subscriptionResult(v webhooks.SubscriptionView) integrationResult {
	id := v.Subscription.ID
	return integrationResult{state: v.State, fields: map[string]string{"subscription": id, "target": subscriptionTarget + id}}
}

func (s *Server) requestedTypes(text string) ([]string, bool) {
	types := strings.Split(text, ",")
	seen := map[string]bool{}
	for _, name := range types {
		if seen[name] || !s.integ.approvedType(name) {
			return nil, false
		}
		seen[name] = true
	}
	return types, len(types) <= maxSubscriptionTypes
}

func sameTypes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]bool{}
	for _, name := range a {
		set[name] = true
	}
	for _, name := range b {
		if !set[name] {
			return false
		}
	}
	return true
}

func (s *Server) duplicateOf(want webhooks.Subscription) (*subscriptionEntry, webhooks.SubscriptionView, error) {
	for _, e := range s.integ.subscriptions {
		v, err := s.subscriptionOf(e.id)
		if errors.Is(err, webhooks.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, v, err
		}
		sub := v.Subscription
		if (v.State == webhooks.SubscriptionProposed || v.State == webhooks.SubscriptionActive) && sub.Subscriber == want.Subscriber &&
			sub.Destination == want.Destination && sameTypes(sub.Types, want.Types) {
			return e, v, nil
		}
	}
	return nil, webhooks.SubscriptionView{}, nil
}

func (s *Server) subscriptionProposeCommand(ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	destination := ctx.Fields["destination"]
	types, ok := s.requestedTypes(ctx.Fields["types"])
	if _, listed := destinationState(destination); !listed {
		return integrationResult{}, refuse(commands.InvalidRequest, "destination")
	}
	if !ok {
		return integrationResult{}, refuse(commands.InvalidRequest, "types")
	}
	want := webhooks.Subscription{ID: subscriptionPrefix + strconv.Itoa(s.integ.counter+1), Subscriber: sampleSubscriber, Destination: destination, Types: types}
	same, current, err := s.duplicateOf(want)
	switch {
	case err != nil:
		return integrationResult{}, mapIntegration(err)
	case same != nil && (current.Proposer != reader.ID || current.State != webhooks.SubscriptionProposed):
		return integrationResult{}, refuse(commands.InvalidStage)
	case same != nil:
		want.ID = same.id
	}
	_, v, err := s.subscriptionPropose(want, reader, ctx.Rationale)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	if same == nil {
		s.integ.counter++
	}
	return subscriptionResult(v), nil
}

func (s *Server) subscriptionCommand(kind string, ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	e := s.subscriptionEntryOf(ctx.Target)
	current, err := s.subscriptionOf(e.id)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	type step struct {
		silent bool
		action string
		call   func(string, string, webhooks.Actor) (webhooks.SubscriptionView, error)
	}
	w := s.webhooks
	steps := map[string]step{
		subApproveKind: {current.State != webhooks.SubscriptionProposed, "approve", w.ApproveSubscription},
		subPauseKind:   {false, "pause", w.PauseSubscription},
		subRequestKind: {current.Resumer != "", "request_resume", w.RequestResume},
		subResumeKind:  {false, "approve_resume", w.ApproveResume},
		subRevokeKind:  {current.State == webhooks.SubscriptionRevoked, "revoke", w.RevokeSubscription},
	}
	st := steps[kind]
	if st.silent {
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	v, err := s.subscriptionStep(e, st.action, reader, ctx.Rationale, st.call)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return subscriptionResult(v), nil
}

func (s *Server) letterCommand(kind string, ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	e, letter := s.letterEntryOf(ctx.Target), letterOf(ctx.Target)
	ws, w := s.bootWorkspace, s.webhooks
	requests := s.integ.redeliver
	if kind == letterDismissKind || kind == letterDropKind {
		requests = s.integ.dismiss
	}
	fields := map[string]string{"delivery": letter, "target": ctx.Target}
	var err error
	state := "requested"
	switch kind {
	case letterRedeliverKind, letterDismissKind:
		if _, pending := requests[letter]; pending {
			return integrationResult{}, refuse(commands.InvalidStage)
		}
		call := w.RequestRedeliver
		if kind == letterDismissKind {
			call = w.RequestDismiss
		}
		if err = s.webhookCall(strings.TrimPrefix(kind, "letter_"), func() error { return call(ws, e.id, letter, reader) }); err == nil {
			requests[letter] = letterRequest{by: reader.ID, rationale: ctx.Rationale}
		}
	case letterAgainKind:
		var fresh webhooks.DeliveryView
		err = s.webhookCall("approve_redeliver", func() (err error) { fresh, err = w.ApproveRedeliver(ws, e.id, letter, reader); return })
		state, fields["redelivery"] = fresh.State, fresh.ID
	default:
		err = s.webhookCall("approve_dismiss", func() error { return w.ApproveDismiss(ws, e.id, letter, reader) })
		state = "dismissed"
	}
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	if kind == letterAgainKind || kind == letterDropKind {
		s.integ.dropLetter(e, letter)
	}
	return integrationResult{state: state, fields: fields}, nil
}

func (h integrationHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	reader := plugins.Actor{ID: ctx.User, Kind: plugins.KindHuman}
	hook := webhooks.Actor{ID: ctx.User, Kind: webhooks.KindHuman}
	var result integrationResult
	var err error
	switch {
	case h.kind == pluginApproveKind:
		result, err = s.pluginApproveCommand(ctx, reader)
	case h.kind == pluginWithdrawKind:
		result, err = s.pluginWithdrawCommand(ctx, reader)
	case h.kind == pluginProposeKind:
		result, err = s.pluginProposeCommand(ctx, reader)
	case h.kind == pluginStopKind, h.kind == pluginDisableKind:
		result, err = s.pluginRuntimeCommand(ctx, reader, h.kind == pluginStopKind)
	case h.kind == subProposeKind:
		result, err = s.subscriptionProposeCommand(ctx, hook)
	case strings.HasPrefix(h.kind, "subscription_"):
		result, err = s.subscriptionCommand(h.kind, ctx, hook)
	default:
		result, err = s.letterCommand(h.kind, ctx, hook)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: result.state, Fields: result.fields}, nil
}

func (s *Server) integrationDefinitions() []commands.Definition {
	define := func(permission commands.Permission, kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: integrationCommand + kind, Permission: permission, Mutating: true, Rationale: true, Fields: f,
			Handler: integrationHandler{s: s, kind: kind}}
	}
	pluginPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	subscriptionPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	letterPermission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	kinds := []string{plugins.ChangeInstall, plugins.ChangeUpgrade, plugins.ChangeRollback, plugins.ChangeEnable, plugins.ChangeRemove}
	return []commands.Definition{
		define(pluginPermission, pluginApproveKind),
		define(pluginPermission, pluginWithdrawKind),
		define(pluginPermission, pluginProposeKind, commands.Field{Name: "kind", Type: commands.Enum, Required: true, Values: kinds},
			commands.Field{Name: "version", Type: commands.Text, Required: false}),
		define(pluginPermission, pluginStopKind),
		define(pluginPermission, pluginDisableKind),
		define(subscriptionPermission, subProposeKind, commands.Field{Name: "destination", Type: commands.Text, Required: true},
			commands.Field{Name: "types", Type: commands.Text, Required: true}),
		define(subscriptionPermission, subApproveKind),
		define(subscriptionPermission, subPauseKind),
		define(subscriptionPermission, subRequestKind),
		define(subscriptionPermission, subResumeKind),
		define(subscriptionPermission, subRevokeKind),
		define(letterPermission, letterRedeliverKind),
		define(letterPermission, letterAgainKind),
		define(letterPermission, letterDismissKind),
		define(letterPermission, letterDropKind),
	}
}
