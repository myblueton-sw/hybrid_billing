package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func refusedReply(got reply) bool { return got.status == 401 || got.status == 403 }

func (f world) as3() (actor, bool) {
	cookie, csrf, err := f.s.openFor(LoginTriple(Admin3Login), SampleWorkspace, false)
	return actor{cookie, csrf}, err == nil
}

func (f world) do3(kind, target string, fields map[string]string) reply {
	f.t.Helper()
	a, ok := f.as3()
	if !ok {
		return reply{status: 401}
	}
	return f.runWith(a, f.s.Revision(), integrationCommand+kind, target, fields)
}

type lossPath struct {
	name    string
	lose    func(f world, root actor)
	restore func(f world, root actor)
}

func (f world) mustOK(t testing.TB, name string, got reply) {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("%s: %s", name, got.raw)
	}
}

func lossPaths(t *testing.T) []lossPath {
	return []lossPath{
		{"demotion", func(f world, root actor) {
			f.mustOK(t, "demote", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}))
		}, func(f world, root actor) {
			f.setPassword(Admin3Login, pw)
			f.mustOK(t, "promote", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator", "role_until": f.at(48 * time.Hour)}))
		}},
		{"membership suspension", func(f world, root actor) {
			f.mustOK(t, "suspend membership", f.run(root, "identity.suspend_membership", SampleAdmin3, nil))
		}, func(f world, root actor) {
			f.mustOK(t, "reinstate membership", f.run(root, "identity.reinstate_membership", SampleAdmin3, nil))
		}},
		{"user suspension", func(f world, root actor) {
			f.mustOK(t, "suspend user", f.run(root, "identity.suspend_user", SampleAdmin3, nil))
		}, func(f world, root actor) {
			f.mustOK(t, "reinstate user", f.run(root, "identity.reinstate_user", SampleAdmin3, nil))
		}},
		{"role period", func(f world, root actor) {
			f.mustOK(t, "period", f.period(root, SampleAdmin3, f.at(2*time.Minute)))
			f.advance(3 * time.Minute)
			f.sweep()
		}, nil},
		{"bulk revocation", func(f world, root actor) {
			id, digest := f.prepared(t, root, "suspend_membership", SampleAdmin3)
			if got := f.execute(root, id, digest); got.status != 202 {
				t.Fatalf("execute %s", got.raw)
			}
		}, nil},
		{"grant revoked", func(f world, _ actor) {
			for _, g := range f.s.admin.Grants() {
				if g.SubjectID == SampleAdmin3 && gate.InstallationAction(g.Terms.Action) && g.Terms.Action != gate.AdministerInstallation && !g.Closed {
					if err := f.s.admin.RevokeGrant(g.ID, g.Version); err != nil {
						t.Fatal(err)
					}
				}
			}
		}, nil},
		{"console grant removed", func(f world, _ actor) {
			for _, g := range f.s.admin.Grants() {
				if g.ClientID == ConsoleClient && gate.InstallationAction(g.Terms.Action) && !g.Closed && g.Terms.Action != gate.AdministerInstallation {
					if err := f.s.admin.RevokeGrant(g.ID, g.Version); err != nil {
						t.Fatal(err)
					}
				}
			}
		}, nil},
	}
}

func TestRealRevocationPathsTakeEffectPerArea(t *testing.T) {
	checks := []struct {
		right gate.Action
		probe func(f world) bool
	}{
		{gate.AdministerAPIClients, func(f world) bool {
			a, ok := f.as3()
			if !ok {
				return false
			}
			got := f.integrations(a)
			return got.status == 200 && areaState(got, "clients") == "shown"
		}},
		{gate.AdministerModules, func(f world) bool {
			return !refusedReply(f.do3(pluginProposeKind, samplePluginAt, map[string]string{"kind": "remove"}))
		}},
		{gate.ApproveChannels, func(f world) bool {
			return !refusedReply(f.do3(subApproveKind, subscriptionAt(samplePayments), map[string]string{}))
		}},
		{gate.ReprocessEvents, func(f world) bool {
			return !refusedReply(f.do3(letterDismissKind, letterAt(sampleLetter), map[string]string{}))
		}},
	}
	for _, path := range lossPaths(t) {
		for _, c := range checks {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			f.setRights(t, SampleAdmin3, gate.AdministerInstallation, c.right)
			path.lose(f, root)
			if c.probe(f) {
				t.Errorf("%s: %s still works", path.name, c.right)
			}
			if path.restore != nil {
				path.restore(f, root)
				if !c.probe(f) {
					t.Errorf("%s: %s did not come back within its validity", path.name, c.right)
				}
			}
		}
	}
}

func (f world) pendingBy3(t *testing.T, approval string) (string, actor) {
	t.Helper()
	root := f.as(AdminLogin)
	switch approval {
	case pluginApproveKind:
		got := f.do3(pluginProposeKind, samplePluginAt, map[string]string{"kind": "remove"})
		f.mustOK(t, "propose", got)
		return pluginProposalTarget + got.body["fields"].(map[string]any)["proposal"].(string), f.as(Admin2Login)
	case subApproveKind:
		got := f.do3(subProposeKind, webhooksTarget, map[string]string{"destination": "sample-erp-webhook", "types": typePayment})
		f.mustOK(t, "propose subscription", got)
		return got.body["fields"].(map[string]any)["target"].(string), f.as(Admin2Login)
	case subResumeKind:
		f.mustOK(t, "pause", f.run(root, integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
		f.mustOK(t, "request resume", f.do3(subRequestKind, subscriptionAt(sampleSubscription), map[string]string{}))
		return subscriptionAt(sampleSubscription), f.as(Admin2Login)
	case letterAgainKind:
		f.mustOK(t, "withdraw seed request", f.run(root, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-1", nil))
		f.mustOK(t, "request redelivery", f.do3(letterRedeliverKind, letterAt(sampleLetter), map[string]string{}))
		return letterAt(sampleLetter), f.as(Admin2Login)
	}
	f.mustOK(t, "request dismissal", f.do3(letterDismissKind, letterAt(sampleLetter), map[string]string{}))
	return letterAt(sampleLetter), f.as(Admin2Login)
}

func TestApprovalsCloseWorkWhoseStarterLostTheRight(t *testing.T) {
	approvals := []string{pluginApproveKind, subApproveKind, subResumeKind, letterAgainKind, letterDropKind}
	for _, path := range lossPaths(t) {
		if path.name == "console grant removed" {
			continue
		}
		for _, approval := range approvals {
			f := setup(t, SeedSample).ready()
			f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
			target, approver := f.pendingBy3(t, approval)
			before := f.integrations(approver)
			path.lose(f, f.as(AdminLogin))
			lostBefore := f.integrations(approver)
			revision := f.s.Revision()
			got := f.run(approver, integrationCommand+approval, target, nil)
			if got.status != 200 || got.body["business_state"] != integrationClosed || fmt.Sprint(got.body["fields"]) != "map[reason:requester_lost_right target:"+target+"]" {
				t.Errorf("%s %s: %s", path.name, approval, got.raw)
				continue
			}
			if f.s.Revision() == revision {
				t.Errorf("%s %s: the closing did not move the revision", path.name, approval)
			}
			after := f.integrations(approver)
			switch approval {
			case pluginApproveKind:
				number := strings.TrimPrefix(target, pluginProposalTarget)
				p := rowWith(t, after, "modules", "proposals", "target", target)
				if p["state"] != integrationClosed || p["closing_reason"] != blockedLostRight || p["closed_request"] == "" || p["closed_by"] != "" ||
					p["closure_trigger"] != SampleAdmin2 || p["closure_trigger_login"] != Admin2Login {
					t.Errorf("%s closed proposal %v", path.name, p)
				}
				if b := rowWith(t, lostBefore, "modules", "proposals", "target", target)["approve_blocked"]; b != blockedLostRight {
					t.Errorf("%s read blocker %v", path.name, b)
				}
				if b := rowWith(t, before, "modules", "proposals", "target", target)["approve_blocked"]; b == blockedLostRight {
					t.Errorf("%s blocker before the loss", path.name)
				}
				_ = number
			case subApproveKind:
				id := strings.TrimPrefix(target, subscriptionTarget)
				s := subscriptionRowFor(t, after, id)
				if s["state"] != webhooks.SubscriptionRevoked || s["reason"] != webhooks.ReasonRequesterLostRight {
					t.Errorf("%s closed subscription %v %v", path.name, s["state"], s["reason"])
				}
				if b := subscriptionRowFor(t, lostBefore, id)["actions"].(map[string]any)["approve"].(map[string]any)["blocked"]; b != blockedLostRight {
					t.Errorf("%s subscription blocker %v", path.name, b)
				}
			case subResumeKind:
				s := subscriptionRowFor(t, after, sampleSubscription)
				history := fmt.Sprint(s["history"])
				if s["resumer"] != "" || !strings.Contains(history, "resume_closed") || !strings.Contains(history, "reason:requester_lost_right") {
					t.Errorf("%s resume %v %s", path.name, s["resumer"], history)
				}
			default:
				d := deliveryRowFor(t, after, sampleLetter)
				if d["redeliver_request"] != nil && approval == letterAgainKind || d["dismiss_request"] != nil && approval == letterDropKind {
					t.Errorf("%s letter request kept %v", path.name, d)
				}
			}
		}
	}
}

func TestClosingFreesTheSlotAndStateChecksComeFirst(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
	target, approver := f.pendingBy3(t, letterDropKind)
	f.setRights(t, SampleAdmin3, gate.AdministerInstallation)
	f.mustOK(t, "closing", f.run(approver, integrationCommand+letterDropKind, target, nil))
	f.unchanged(t, "no pending request now", f.run(approver, integrationCommand+letterDropKind, target, nil), "invalid_stage")
	f.setRights(t, SampleAdmin3, gate.ReprocessEvents, gate.AdministerInstallation)
	f.mustOK(t, "the slot is free", f.do3(letterDismissKind, target, map[string]string{}))
	g := setup(t, SeedSample).ready()
	root := g.as(AdminLogin)
	g.unchanged(t, "approve resume with no request", g.run(root, integrationCommand+subResumeKind, subscriptionAt(sampleSubscription), nil), "invalid_stage")
	g.unchanged(t, "approve dismissal with only a redelivery request", g.run(g.as(Admin2Login), integrationCommand+letterDropKind, letterAt(sampleLetter), nil), "invalid_stage")
	g.ok(t, "approve 8", g.run(root, integrationCommand+pluginApproveKind, pluginProposalAt(8), nil), "applied", "")
	g.mustOK(t, "demote the proposer", g.run(root, "identity.change_role", SampleAdmin2, map[string]string{"role": "member"}))
	g.unchanged(t, "an applied proposal is not closed", g.run(root, integrationCommand+pluginApproveKind, pluginProposalAt(8), nil), "invalid_stage")
	h := setup(t, SeedSample).ready()
	h.s.integ.entry(9).view.Proposer = ""
	h.code(t, "an empty starter", h.run(h.as(Admin2Login), integrationCommand+pluginApproveKind, pluginProposalAt(9), nil), 500, "internal_error")
	for step, code := range map[string]string{"starter-authority": "authority_unavailable", "starter-identity": "dependency_unavailable"} {
		w := setup(t, SeedSample).ready()
		revision := w.s.Revision()
		w.s.fault = func(s string) bool { return s == step }
		w.code(t, step, w.run(w.as(Admin2Login), integrationCommand+pluginApproveKind, pluginProposalAt(9), nil), 503, code)
		w.s.fault = nil
		if w.s.integ.entry(9).view.State != "pending" || w.s.Revision() != revision {
			t.Errorf("%s closed or moved", step)
		}
	}
}

func TestARevocationRacingAnApprovalIsOneOrTheOther(t *testing.T) {
	for round := 0; round < 20; round++ {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
		target, approver := f.pendingBy3(t, pluginApproveKind)
		var got reply
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); got = f.run(approver, integrationCommand+pluginApproveKind, target, nil) }()
		go func() {
			defer wg.Done()
			for _, g := range f.s.admin.Grants() {
				if g.SubjectID == SampleAdmin3 && g.Terms.Action == gate.AdministerModules && !g.Closed {
					_ = f.s.admin.RevokeGrant(g.ID, g.Version)
				}
			}
		}()
		wg.Wait()
		state := got.body["business_state"]
		if got.status != 200 || state != "applied" && state != integrationClosed {
			t.Fatalf("round %d: %s", round, got.raw)
		}
	}
}

func (f world) heldRedelivery(t *testing.T) (string, actor, actor) {
	t.Helper()
	root := f.as(AdminLogin)
	f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
	f.mustOK(t, "withdraw seed request", f.run(root, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-1", nil))
	f.mustOK(t, "request redelivery", f.do3(letterRedeliverKind, letterAt(sampleLetter), map[string]string{}))
	got := f.run(f.as(Admin2Login), integrationCommand+letterAgainKind, letterAt(sampleLetter), nil)
	f.mustOK(t, "approve redelivery", got)
	return got.body["fields"].(map[string]any)["redelivery"].(string), root, f.as(Admin2Login)
}

func (f world) dispatch() error {
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	_, err := f.s.dispatchDeliveries(1, webhooks.Actor{ID: "test-relay", Kind: webhooks.KindSystem})
	return err
}

func TestQueuedRedeliveriesAreHeldOrClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, root, second := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	f.s.fault = func(step string) bool { return step == "starter-identity" }
	if err := f.dispatch(); err == nil {
		t.Error("an unknown standing did not skip the round")
	}
	f.s.fault = nil
	if row := deliveryRowFor(t, f.integrations(root), id); row["hold"] != "" || row["state"] != "queued" {
		t.Errorf("an unknown round changed the redelivery %v", row)
	}
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	row := deliveryRowFor(t, f.integrations(root), id)
	if row["hold"] != webhooks.ReasonApproverLostRight || row["target"] != redeliveryTarget+id || row["requester"] != SampleAdmin3 || row["approver"] != SampleAdmin2 {
		t.Fatalf("held %v", row)
	}
	if a := row["actions"].(map[string]any)["reapprove"].(map[string]any); a["allowed"] != true {
		t.Errorf("root cannot re-approve %v", a)
	}
	f.code(t, "the lost approver", f.run(second, integrationCommand+reapproveKind, redeliveryTarget+id, nil), 403, "permission_denied")
	f.unchanged(t, "the requester", f.do3(reapproveKind, redeliveryTarget+id, map[string]string{}), "prerequisite_failed")
	f.code(t, "a letter target", f.run(root, integrationCommand+reapproveKind, letterAt(sampleLetter), nil), 403, "permission_denied")
	f.ok(t, "re-approve", f.run(root, integrationCommand+reapproveKind, redeliveryTarget+id, nil), "queued", "")
	if row := deliveryRowFor(t, f.integrations(root), id); row["hold"] != "" || row["approver"] != SampleAdmin {
		t.Errorf("released %v", row)
	}
	f.code(t, "re-approve twice", f.run(root, integrationCommand+reapproveKind, redeliveryTarget+id, nil), 403, "permission_denied")
	g := setup(t, SeedSample).ready()
	gid, groot, _ := g.heldRedelivery(t)
	g.mustOK(t, "demote the requester", g.run(groot, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}))
	if err := g.dispatch(); err != nil {
		t.Fatal(err)
	}
	closed := deliveryRowFor(t, g.integrations(groot), gid)
	if closed["kind"] != webhooks.KindCancelled || closed["reason"] != webhooks.ReasonRequesterLostRight || closed["hold"] != "" {
		t.Errorf("closed %v", closed)
	}
	g.code(t, "re-approve a closed redelivery", g.run(groot, integrationCommand+reapproveKind, redeliveryTarget+gid, nil), 403, "permission_denied")
	h := setup(t, SeedSample).ready()
	hid, hroot, _ := h.heldRedelivery(t)
	h.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := h.dispatch(); err != nil {
		t.Fatal(err)
	}
	h.mustOK(t, "demote the requester", h.run(hroot, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}))
	h.ok(t, "re-approval closes it", h.run(hroot, integrationCommand+reapproveKind, redeliveryTarget+hid, nil), integrationClosed, "")
	if row := deliveryRowFor(t, h.integrations(hroot), hid); row["kind"] != webhooks.KindCancelled {
		t.Errorf("not closed %v", row)
	}
	active := setup(t, SeedSample).ready()
	active.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := active.dispatch(); err != nil {
		t.Fatal(err)
	}
	if s := subscriptionRowFor(t, active.integrations(active.as(AdminLogin)), sampleSubscription); s["state"] != "active" {
		t.Errorf("an approved subscription stopped %v", s["state"])
	}
}

func TestDispatchHasOneCaller(t *testing.T) {
	sites := calls(t, ".", "Dispatch")
	for _, c := range sites {
		if c.file != "integrationheld.go" || c.function != "dispatchDeliveries" || c.receiver != "s.webhooks" {
			t.Errorf("Dispatch called from %+v", c)
		}
	}
	if len(sites) != 1 {
		t.Errorf("callers %+v", sites)
	}
}

func TestAuthorsWithdrawTheirOwnWork(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
	proposal := f.do3(pluginProposeKind, samplePluginAt, map[string]string{"kind": "remove"})
	f.mustOK(t, "propose", proposal)
	ptarget := proposal.body["fields"].(map[string]any)["target"].(string)
	sub := f.do3(subProposeKind, webhooksTarget, map[string]string{"destination": "sample-erp-webhook", "types": typePayment})
	f.mustOK(t, "propose subscription", sub)
	starget := sub.body["fields"].(map[string]any)["target"].(string)
	f.setRights(t, SampleAdmin3, gate.AdministerInstallation)
	read := f.integrations(f.as(Admin3Login))
	if areaState(read, "modules") != "withheld" || len(read.body["own_pending"].([]any)) != 2 {
		t.Fatalf("own pending %s", read.raw)
	}
	f.code(t, "another administrator", f.run(f.as(Admin2Login), integrationCommand+pluginWithdrawKind, ptarget, nil), 403, "permission_denied")
	n := keys.Add(1)
	body, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("req-%d", n), "target_id": ptarget, "idempotency_key": fmt.Sprintf("key-%d", n),
		"expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{}})
	a3 := f.as(Admin3Login)
	first := f.call(integrationCommand+pluginWithdrawKind, string(body), a3.cookie, a3.csrf)
	if first.status != 200 || first.body["business_state"] != "withdrawn" {
		t.Fatalf("withdraw %s", first.raw)
	}
	again := f.call(integrationCommand+pluginWithdrawKind, string(body), a3.cookie, a3.csrf)
	if again.status != 200 || again.body["replay"] != true {
		t.Errorf("replay %s", again.raw)
	}
	f.ok(t, "withdraw the subscription", f.do3(subWithdrawKind, starget, map[string]string{}), "revoked", "")
	f.mustOK(t, "demote", f.run(f.as(AdminLogin), "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}))
	replay := f.call(integrationCommand+pluginWithdrawKind, string(body), a3.cookie, a3.csrf)
	if replay.status != 401 && replay.status != 403 {
		t.Errorf("a replay after demotion %s", replay.raw)
	}
	if got := f.integrations(f.as(Admin3Login)); got.status != 403 {
		t.Errorf("a member read P02 %s", got.raw)
	}
}

func TestLetterRequestWithdrawalIsPerKindAndPerRequest(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.mustOK(t, "request dismissal", f.run(second, integrationCommand+letterDismissKind, letterAt(sampleLetter), nil))
	redeliver, dismiss := letterRequestTarget+"letter-request-1", letterRequestTarget+"letter-request-2"
	f.code(t, "another person's request", f.run(second, integrationCommand+redeliverWithdrawKind, redeliver, nil), 403, "permission_denied")
	f.code(t, "the other kind", f.run(root, integrationCommand+dismissWithdrawKind, redeliver, nil), 403, "permission_denied")
	f.code(t, "the other kind on one's own", f.run(second, integrationCommand+redeliverWithdrawKind, dismiss, nil), 403, "permission_denied")
	n := keys.Add(1)
	body, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("req-%d", n), "target_id": redeliver, "idempotency_key": fmt.Sprintf("key-%d", n),
		"expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{}})
	first := f.call(integrationCommand+redeliverWithdrawKind, string(body), root.cookie, root.csrf)
	if first.status != 200 || first.body["business_state"] != "withdrawn" {
		t.Fatalf("withdraw %s", first.raw)
	}
	f.mustOK(t, "B requests", f.run(second, integrationCommand+letterRedeliverKind, letterAt(sampleLetter), nil))
	replay := f.call(integrationCommand+redeliverWithdrawKind, string(body), root.cookie, root.csrf)
	if replay.status != 200 || replay.body["replay"] != true {
		t.Errorf("A's replay %s", replay.raw)
	}
	if r := f.s.integ.redeliver[sampleLetter]; r.by != SampleAdmin2 || r.id != "letter-request-3" {
		t.Errorf("B's request %+v", r)
	}
	f.mustOK(t, "A requests again", f.run(second, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-3", nil))
	f.mustOK(t, "A files req-4", f.run(root, integrationCommand+letterRedeliverKind, letterAt(sampleLetter), nil))
	f.unchanged(t, "a fresh key on the older request", f.run(root, integrationCommand+redeliverWithdrawKind, redeliver, nil), "invalid_stage")
	if r := f.s.integ.redeliver[sampleLetter]; r.id != "letter-request-4" {
		t.Errorf("the newer request was touched %+v", r)
	}
}

func TestRequestTombstonesAreCapped(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.s.integ.requests.limit = 2
	f.mustOK(t, "withdraw 1", f.run(root, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-1", nil))
	f.mustOK(t, "request 2", f.run(second, integrationCommand+letterRedeliverKind, letterAt(sampleLetter), nil))
	f.mustOK(t, "withdraw 2", f.run(second, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-2", nil))
	f.mustOK(t, "request 3", f.run(root, integrationCommand+letterRedeliverKind, letterAt(sampleLetter), nil))
	if f.s.integ.requests.records["letter-request-1"] != nil || f.s.integ.requests.records["letter-request-3"] == nil {
		t.Errorf("eviction kept %v", f.s.integ.requests.order)
	}
	f.code(t, "an evicted request", f.run(root, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-1", nil), 403, "permission_denied")
	f.mustOK(t, "request a dismissal", f.run(second, integrationCommand+letterDismissKind, letterAt(sampleLetter), nil))
	f.s.integ.requests.limit = 0
	f.s.integ.newRequest(requestDismiss, SampleAdmin, "elsewhere", sampleSubscription)
	if f.s.integ.requests.records["letter-request-4"] == nil {
		t.Error("a live request was evicted")
	}
}

func TestSeedAndBootstrapGrantAllOrNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	holds := func(user string) []string {
		out := []string{}
		for _, right := range gate.IntegrationRights() {
			if f.s.holdsRight(user, right, f.s.clock()) {
				out = append(out, string(right))
			}
		}
		return out
	}
	all := fmt.Sprint(gate.IntegrationRights())
	if fmt.Sprint(holds(SampleAdmin)) != all || fmt.Sprint(holds(SampleAdmin2)) != all ||
		fmt.Sprint(holds(SampleAdmin3)) != "[stop_modules stop_channels reprocess_events]" || len(holds(SampleMember)) != 0 {
		t.Errorf("seed profiles %v %v %v", holds(SampleAdmin), holds(SampleAdmin2), holds(SampleAdmin3))
	}
	for k := 0; k < 8; k++ {
		g := setup(t, SeedSample).ready()
		g.setPassword(MemberLogin, pw)
		g.s.fault = func(step string) bool { return step == fmt.Sprintf("integration-grant-%d", k) }
		if err := g.s.grantIntegration(SampleMember, gate.IntegrationRights(), g.s.clock()); !errors.Is(err, errPartial) {
			t.Errorf("k=%d error %v", k, err)
		}
		g.s.fault = nil
		for _, gr := range g.s.admin.Grants() {
			if gr.SubjectID == SampleMember && !gr.Closed && strings.HasPrefix(gr.ID, integrationGrantID) {
				t.Errorf("k=%d kept %s", k, gr.ID)
			}
		}
		clients := 0
		for _, gr := range g.s.admin.Grants() {
			if gr.ClientID == ConsoleClient && strings.HasPrefix(gr.ID, integrationGrantID) && !gr.Closed {
				clients++
			}
		}
		if clients != 8 {
			t.Errorf("k=%d console grants %d", k, clients)
		}
	}
	g := setup(t, SeedSample).ready()
	g.setPassword(MemberLogin, pw)
	g.s.fault = func(step string) bool { return step == "integration-grant-3" || step == "integration-rollback" }
	if err := g.s.grantIntegration(SampleMember, gate.IntegrationRights(), g.s.clock()); !errors.Is(err, errRollback) {
		t.Errorf("rollback failure %v", err)
	}
	b := setup(t, SeedNone).ready()
	boot := b.firstAdmin("boot@example.invalid")
	if got := b.integrations(boot); got.status != 200 || rightsText(got) != strings.Join(strings.Fields(strings.Trim(all, "[]")), ",") {
		t.Errorf("bootstrap rights %s", got.raw)
	}
	c := setup(t, SeedNone).ready()
	c.s.fault = func(step string) bool { return step == "integration-grant-2" }
	cboot := c.firstAdmin("boot@example.invalid")
	c.s.fault = nil
	got := c.integrations(cboot)
	if got.status != 200 || rightsText(got) != "" || areaState(got, "modules") != "withheld" || areaState(got, "clients") != "withheld" {
		t.Errorf("a failed bootstrap grant %s", got.raw)
	}
}

func TestM03NeverOffersIntegrationRights(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for _, right := range gate.IntegrationRights() {
		got := f.requestGrant(root, SampleMember, map[string]string{"action": string(right), "kind": gate.InstallationKind, "fields": ""})
		f.code(t, string(right), got, 422, "invalid_request")
	}
	strip := func(r reply) string {
		delete(r.body, "request_id")
		raw, _ := json.Marshal(r.body)
		return fmt.Sprint(r.status) + string(raw)
	}
	unknown := strip(f.run(root, "access.request_revoke", "grant-unknown", nil))
	listed, rows := "", 0
	for next, pages := "", 0; pages < 200; pages++ {
		query := "?limit=50"
		if next != "" {
			query += "&after=" + url.QueryEscape(next)
		}
		got := f.permissions(root, query)
		if got.status != 200 {
			t.Fatalf("the permissions list: %s", got.raw)
		}
		listed += got.raw
		rows += len(got.body["rows"].([]any))
		if next, _ = got.body["next"].(string); next == "" {
			break
		}
	}
	if rows == 0 {
		t.Fatal("the permissions list is empty; its silence about integration grants would prove nothing")
	}
	checked := 0
	read := reply{raw: listed}
	for _, g := range f.s.admin.Grants() {
		if !strings.HasPrefix(g.ID, integrationGrantID) {
			continue
		}
		if got := strip(f.run(root, "access.request_revoke", g.ID, nil)); got != unknown {
			t.Errorf("%s answers %s", g.ID, got)
		}
		if !protectedGrant(g) {
			t.Errorf("%s is not protected from replay", g.ID)
		}
		if strings.Contains(read.raw, g.ID) || strings.Contains(read.raw, string(g.Terms.Action)) {
			t.Errorf("M03 lists %s", g.ID)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("there is no integration grant to look for")
	}
}

func TestTheIntegrationsReadIsDocumented(t *testing.T) {
	f := setup(t, SeedSample).ready()
	raw, err := f.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for path, item := range doc["paths"].(map[string]any) {
		kind, ok := strings.CutPrefix(path, "/commands/"+integrationCommand)
		if !ok {
			continue
		}
		permission := item.(map[string]any)["post"].(map[string]any)["x-permission"].(map[string]any)
		if permission["action"] != wantRights[kind] || (permission["author_only"] == true) != wantAuthor[kind] {
			t.Errorf("%s %v", path, permission)
		}
	}
	read := doc["x-reads"].(map[string]any)["/integrations"].(map[string]any)["get"].(map[string]any)
	responses := fmt.Sprint(read["responses"])
	if fmt.Sprint(read["x-admission"]) != "[administer_api_clients administer_modules approve_modules stop_modules administer_channels approve_channels stop_channels reprocess_events administer_installation]" ||
		fmt.Sprint(read["x-area-rights"].(map[string]any)["clients"]) != "[administer_api_clients]" || read["x-blocked-first"] != "no_right" ||
		!strings.Contains(responses, CodeCSRF) || !strings.Contains(responses, "dependency_unavailable") || !strings.Contains(responses, "authority_unavailable") {
		t.Errorf("read %v", read)
	}
}
