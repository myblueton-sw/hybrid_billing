package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

var errScreenUnknown = errors.New("a redelivery starter could not be checked")

var deliveryPage = 200

func (e *subscriptionEntry) settle(v webhooks.SubscriptionView) {
	if v.State != webhooks.SubscriptionProposed {
		e.proposed = false
	}
	if v.Resumer == "" {
		e.resume = nil
	}
}

type screening struct {
	subscription string
	delivery     string
	close        bool
}

func screenable(d webhooks.DeliveryView) bool {
	return d.Kind == webhooks.KindRedelivery && d.State == webhooks.JobQueued && d.Hold == ""
}

func (s *Server) allDeliveries(subscription string, reader webhooks.Actor) ([]webhooks.DeliveryView, error) {
	out := []webhooks.DeliveryView{}
	for {
		var page []webhooks.DeliveryView
		err := s.webhookCall("screen", func() (err error) {
			page, err = s.webhooks.Deliveries(s.bootWorkspace, subscription, len(out), deliveryPage, reader)
			return
		})
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
		if len(page) < deliveryPage {
			return out, nil
		}
	}
}

func (s *Server) screenRedeliveries(inspector webhooks.Actor) ([]screening, error) {
	now := s.clock()
	out := []screening{}
	for _, e := range s.integ.subscriptions {
		list, err := s.allDeliveries(e.id, inspector)
		if errors.Is(err, webhooks.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, d := range list {
			if !screenable(d) {
				continue
			}
			requester, _ := s.standingOf(d.Requester, gate.ReprocessEvents, now)
			approver, _ := s.standingOf(d.Approver, gate.ReprocessEvents, now)
			switch {
			case requester == unknown || approver == unknown:
				return nil, errScreenUnknown
			case requester == lost:
				out = append(out, screening{subscription: e.id, delivery: d.ID, close: true})
			case approver == lost:
				out = append(out, screening{subscription: e.id, delivery: d.ID})
			}
		}
	}
	return out, nil
}

func (s *Server) dispatchDeliveries(limit int, actor webhooks.Actor) ([]webhooks.Lease, error) {
	found, err := s.screenRedeliveries(actor)
	if err != nil {
		return nil, err
	}
	for _, f := range found {
		if f.close {
			err = s.webhookCall("close_queued", func() (err error) {
				_, err = s.webhooks.CloseRedelivery(s.bootWorkspace, f.subscription, f.delivery, webhooks.ReasonRequesterLostRight)
				return
			})
		} else {
			err = s.webhookCall("hold", func() (err error) {
				_, err = s.webhooks.HoldRedelivery(s.bootWorkspace, f.subscription, f.delivery, webhooks.ReasonApproverLostRight)
				return
			})
		}
		if err != nil {
			return nil, err
		}
		if f.close {
			delete(s.integ.held, f.delivery)
		} else {
			s.integ.held[f.delivery] = f.subscription
		}
	}
	var leases []webhooks.Lease
	err = s.webhookCall("dispatch_round", func() (err error) { leases, err = s.webhooks.Dispatch(s.bootWorkspace, limit, actor); return })
	return leases, err
}

func (s *Server) starterCheck(starter string, right gate.Action) (bool, error) {
	if starter == "" {
		return false, refuse(commands.InternalError)
	}
	got, code := s.standingOf(starter, right, s.clock())
	switch got {
	case unknown:
		return false, refuse(code)
	case lost:
		return true, nil
	}
	return false, nil
}

func closedResult(target string) integrationResult {
	return integrationResult{state: integrationClosed, fields: map[string]string{"reason": blockedLostRight, "target": target}}
}
