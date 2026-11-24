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

func (h integrationHandler) author(target string) (string, bool) {
	s := h.s
	switch h.kind {
	case pluginWithdrawKind:
		if e := s.pluginEntryOf(target); e != nil {
			return e.view.Proposer, true
		}
	case subWithdrawKind:
		if e := s.subscriptionEntryOf(target); e != nil {
			return e.proposer, true
		}
	case resumeWithdrawKind, redeliverWithdrawKind, dismissWithdrawKind:
		prefix := letterRequestTarget
		if h.kind == resumeWithdrawKind {
			prefix = resumeRequestTarget
		}
		if rec := s.integ.record(target, prefix); rec != nil && rec.kind == requestKindOf(h.kind) {
			return rec.author, true
		}
	}
	return "", false
}

func (h integrationHandler) found(target string) bool {
	s := h.s
	switch {
	case authorKinds[h.kind]:
		_, ok := h.author(target)
		return ok
	case h.kind == reapproveKind:
		id, ok := strings.CutPrefix(target, redeliveryTarget)
		_, held := s.integ.held[id]
		return ok && held
	case h.kind == pluginApproveKind:
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
	author, _ := h.author(target)
	return commands.Target{Revision: h.s.Revision(), AuthorID: author}, commands.Found
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
	gone, err := s.starterCheck(e.view.Proposer, gate.AdministerModules)
	switch {
	case err != nil:
		return integrationResult{}, err
	case gone:
		var view plugins.ProposalView
		if err := s.pluginCall("close", func() (err error) {
			view, err = s.plugins.Close(e.view.Number, plugins.ReasonRequesterLostRight)
			return
		}); err != nil {
			return integrationResult{}, mapIntegration(err)
		}
		e.view, e.closedBy, e.closing, e.reason, e.request, e.trigger = view, "", "", blockedLostRight, ctx.Request, reader.ID
		return closedResult(ctx.Target), nil
	}
	if err := s.pluginApprove(e, reader, ctx.Rationale); err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return proposalResult(e), nil
}

func (s *Server) pluginWithdrawCommand(ctx commands.Context, reader plugins.Actor) (integrationResult, error) {
	e := s.pluginEntryOf(ctx.Target)
	if e.view.State != plugins.ProposalPending || e.view.Proposer != ctx.User {
		return integrationResult{}, refuse(commands.InvalidStage)
	}
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
	e, existing, err := s.pluginProposeOnce(plugins.Change{Kind: kind, Workspace: plugins.Workspace{ID: s.integ.workspace}, Plugin: p.id, Version: version}, reader, ctx.Rationale)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	result := proposalResult(e)
	if existing {
		result.fields["existing"] = "true"
	}
	return result, nil
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

func resuming(v webhooks.SubscriptionView) bool {
	return v.Resumer != "" && pausedOrSuspended(v.State)
}

func (s *Server) closeSubscriptionWork(e *subscriptionEntry, kind string, ctx commands.Context, by string) (integrationResult, error) {
	var err error
	var view webhooks.SubscriptionView
	action := "proposal_closed"
	if kind == subApproveKind {
		err = s.webhookCall("close", func() (err error) {
			view, err = s.webhooks.CloseSubscription(s.bootWorkspace, e.id, webhooks.ReasonRequesterLostRight)
			return
		})
	} else {
		action = "resume_closed"
		err = s.webhookCall("close_resume", func() (err error) {
			view, err = s.webhooks.CloseResume(s.bootWorkspace, e.id, webhooks.ReasonRequesterLostRight)
			return
		})
	}
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	e.history = append(e.history, actionRecord{action: action, actor: by, reason: blockedLostRight, request: ctx.Request})
	e.settle(view)
	return closedResult(ctx.Target), nil
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
		subApproveKind:  {current.State != webhooks.SubscriptionProposed, "approve", w.ApproveSubscription},
		subPauseKind:    {false, "pause", w.PauseSubscription},
		subRequestKind:  {current.Resumer != "", "request_resume", w.RequestResume},
		subResumeKind:   {!resuming(current), "approve_resume", w.ApproveResume},
		subRevokeKind:   {current.State == webhooks.SubscriptionRevoked, "revoke", w.RevokeSubscription},
		subWithdrawKind: {current.State != webhooks.SubscriptionProposed || current.Proposer != ctx.User, "withdraw", w.WithdrawSubscription},
	}
	st := steps[kind]
	if st.silent {
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	if kind == subApproveKind || kind == subResumeKind {
		starter := current.Proposer
		if kind == subResumeKind {
			starter = current.Resumer
		}
		gone, err := s.starterCheck(starter, gate.AdministerChannels)
		if err != nil {
			return integrationResult{}, err
		}
		if gone {
			return s.closeSubscriptionWork(e, kind, ctx, reader.ID)
		}
	}
	v, err := s.subscriptionStep(e, st.action, reader, ctx.Rationale, st.call)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	if kind == subRequestKind {
		r := s.integ.newRequest(requestResume, reader.ID, e.id, e.id)
		r.rationale = ctx.Rationale
		e.resume = &r
	}
	return subscriptionResult(v), nil
}

func (s *Server) resumeWithdrawCommand(ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	rec := s.integ.record(ctx.Target, resumeRequestTarget)
	e := s.integ.subscription(rec.object)
	if e == nil || e.resume == nil || e.resume.id != rec.id || e.resume.by != ctx.User {
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	v, err := s.subscriptionStep(e, "withdraw_resume", reader, ctx.Rationale, s.webhooks.WithdrawResume)
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	e.resume = nil
	return integrationResult{state: "withdrawn", fields: map[string]string{"request": rec.id, "subscription": v.Subscription.ID, "target": ctx.Target}}, nil
}

func (s *Server) letterWithdrawCommand(kind string, ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	rec := s.integ.record(ctx.Target, letterRequestTarget)
	requests, dismiss := s.integ.redeliver, kind == dismissWithdrawKind
	if dismiss {
		requests = s.integ.dismiss
	}
	live, pending := requests[rec.object]
	e := s.integ.letter(rec.object)
	if !pending || live.id != rec.id || live.by != ctx.User || e == nil {
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	err := s.webhookCall("withdraw_request", func() error { return s.webhooks.WithdrawRequest(s.bootWorkspace, e.id, rec.object, dismiss, reader) })
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	delete(requests, rec.object)
	return integrationResult{state: "withdrawn", fields: map[string]string{"request": rec.id, "delivery": rec.object, "target": ctx.Target}}, nil
}

func (s *Server) letterCommand(kind string, ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	e, letter := s.letterEntryOf(ctx.Target), letterOf(ctx.Target)
	ws, w := s.bootWorkspace, s.webhooks
	requests := s.integ.redeliver
	dismiss := kind == letterDismissKind || kind == letterDropKind
	if dismiss {
		requests = s.integ.dismiss
	}
	fields := map[string]string{"delivery": letter, "target": ctx.Target}
	var err error
	state := "requested"
	pending, waiting := requests[letter]
	switch kind {
	case letterRedeliverKind, letterDismissKind:
		if waiting {
			return integrationResult{}, refuse(commands.InvalidStage)
		}
		call := w.RequestRedeliver
		kindOf := requestRedeliver
		if dismiss {
			call, kindOf = w.RequestDismiss, requestDismiss
		}
		if err = s.webhookCall(strings.TrimPrefix(kind, "letter_"), func() error { return call(ws, e.id, letter, reader) }); err == nil {
			r := s.integ.newRequest(kindOf, reader.ID, letter, e.id)
			r.rationale = ctx.Rationale
			requests[letter] = r
			fields["request"] = r.id
		}
	default:
		if !waiting {
			return integrationResult{}, refuse(commands.InvalidStage)
		}
		gone, failed := s.starterCheck(pending.by, gate.ReprocessEvents)
		if failed != nil {
			return integrationResult{}, failed
		}
		if gone {
			if closing := s.webhookCall("close_request", func() error {
				return w.CloseRequest(ws, e.id, letter, dismiss, webhooks.ReasonRequesterLostRight)
			}); closing != nil {
				return integrationResult{}, mapIntegration(closing)
			}
			delete(requests, letter)
			action := "redeliver_request_closed"
			if dismiss {
				action = "dismiss_request_closed"
			}
			e.history = append(e.history, actionRecord{action: action, actor: reader.ID, reason: blockedLostRight, request: ctx.Request})
			return closedResult(ctx.Target), nil
		}
		if kind == letterAgainKind {
			var fresh webhooks.DeliveryView
			err = s.webhookCall("approve_redeliver", func() (err error) { fresh, err = w.ApproveRedeliver(ws, e.id, letter, reader); return })
			state, fields["redelivery"] = fresh.State, fresh.ID
		} else {
			err = s.webhookCall("approve_dismiss", func() error { return w.ApproveDismiss(ws, e.id, letter, reader) })
			state = "dismissed"
		}
		if err == nil {
			s.integ.dropLetter(e, letter)
		}
	}
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	return integrationResult{state: state, fields: fields}, nil
}

func (s *Server) reapproveCommand(ctx commands.Context, reader webhooks.Actor) (integrationResult, error) {
	id := strings.TrimPrefix(ctx.Target, redeliveryTarget)
	sub := s.integ.held[id]
	var list []webhooks.DeliveryView
	err := s.webhookCall("held", func() (err error) {
		list, err = s.allDeliveries(sub, reader)
		return
	})
	if err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	var held *webhooks.DeliveryView
	for index := range list {
		if list[index].ID == id && list[index].Hold != "" {
			held = &list[index]
		}
	}
	if held == nil {
		delete(s.integ.held, id)
		return integrationResult{}, refuse(commands.InvalidStage)
	}
	gone, err := s.starterCheck(held.Requester, gate.ReprocessEvents)
	if err != nil {
		return integrationResult{}, err
	}
	if gone {
		if err := s.webhookCall("close_redelivery", func() (err error) {
			_, err = s.webhooks.CloseRedelivery(s.bootWorkspace, sub, id, webhooks.ReasonRequesterLostRight)
			return
		}); err != nil {
			return integrationResult{}, mapIntegration(err)
		}
		delete(s.integ.held, id)
		if e := s.integ.subscription(sub); e != nil {
			e.history = append(e.history, actionRecord{action: "redelivery_closed", actor: reader.ID, reason: blockedLostRight, request: ctx.Request})
		}
		return closedResult(ctx.Target), nil
	}
	var fresh webhooks.DeliveryView
	if err := s.webhookCall("reapprove", func() (err error) {
		fresh, err = s.webhooks.ReapproveRedelivery(s.bootWorkspace, sub, id, reader)
		return
	}); err != nil {
		return integrationResult{}, mapIntegration(err)
	}
	delete(s.integ.held, id)
	return integrationResult{state: fresh.State, fields: map[string]string{"redelivery": id, "target": ctx.Target}}, nil
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
	case h.kind == resumeWithdrawKind:
		result, err = s.resumeWithdrawCommand(ctx, hook)
	case h.kind == redeliverWithdrawKind, h.kind == dismissWithdrawKind:
		result, err = s.letterWithdrawCommand(h.kind, ctx, hook)
	case h.kind == reapproveKind:
		result, err = s.reapproveCommand(ctx, hook)
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
	define := func(kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: integrationCommand + kind, Permission: commands.Permission{Action: rightOf(kind), Kind: gate.InstallationKind},
			Author: authorKinds[kind], Mutating: true, Rationale: true, Fields: f, Handler: integrationHandler{s: s, kind: kind}}
	}
	kinds := []string{plugins.ChangeInstall, plugins.ChangeUpgrade, plugins.ChangeRollback, plugins.ChangeEnable, plugins.ChangeRemove}
	return []commands.Definition{
		define(pluginApproveKind),
		define(pluginWithdrawKind),
		define(pluginProposeKind, commands.Field{Name: "kind", Type: commands.Enum, Required: true, Values: kinds},
			commands.Field{Name: "version", Type: commands.Text, Required: false}),
		define(pluginStopKind),
		define(pluginDisableKind),
		define(subProposeKind, commands.Field{Name: "destination", Type: commands.Text, Required: true},
			commands.Field{Name: "types", Type: commands.Text, Required: true}),
		define(subApproveKind),
		define(subPauseKind),
		define(subRequestKind),
		define(subResumeKind),
		define(subRevokeKind),
		define(subWithdrawKind),
		define(resumeWithdrawKind),
		define(letterRedeliverKind),
		define(letterAgainKind),
		define(letterDismissKind),
		define(letterDropKind),
		define(redeliverWithdrawKind),
		define(dismissWithdrawKind),
		define(reapproveKind),
	}
}
