package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

type deliveriesPage struct {
	Revision     string        `json:"revision"`
	ServerTime   string        `json:"server_time"`
	Subscription string        `json:"subscription"`
	Order        string        `json:"order"`
	RowsCanMove  bool          `json:"rows_can_move"`
	Deliveries   []deliveryRow `json:"deliveries"`
	Next         string        `json:"next"`
	Total        int           `json:"total"`
}

func needsAction(d webhooks.DeliveryView) bool { return d.Kind == webhooks.KindDead || d.Hold != "" }

func (s *Server) listDeliveries(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, reader, ok := s.admitIntegrations(w, r)
	if !ok {
		return
	}
	after, limit, valid := 0, shownDeliveries, true
	values := r.URL.Query()
	for name, given := range values {
		switch {
		case len(given) != 1 || given[0] == "":
			valid = false
		case name == "after":
			n, err := strconv.Atoi(given[0])
			after, valid = n, valid && err == nil && n >= 1 && strconv.Itoa(n) == given[0]
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			limit, valid = n, valid && err == nil && n >= 1 && n <= shownDeliveries && strconv.Itoa(n) == given[0]
		case name != "subscription":
			valid = false
		}
	}
	if !valid || values.Get("subscription") == "" {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	if !reader.any(channelRights) {
		failCode(w, commands.PermissionDenied, request)
		return
	}
	if s.faulty("integrations-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	var e *subscriptionEntry
	if answer.Workspace() == s.bootWorkspace {
		e = s.integ.subscription(values.Get("subscription"))
	}
	state := ""
	if e != nil {
		v, err := s.subscriptionOf(e.id)
		switch {
		case errors.Is(err, webhooks.ErrNotFound):
			e = nil
		case err != nil:
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		state = v.State
	}
	if e == nil {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	all, err := s.deliveriesFor(e.id, reader)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page := deliveriesPage{Revision: s.Revision(), ServerTime: stamp(reader.now), Subscription: e.id, Order: requirementsOrder, RowsCanMove: true}
	var listed []webhooks.DeliveryView
	listed, page.Total, page.Next = othersAfter(all, after, limit)
	page.Deliveries = s.deliveryRows(listed, state, reader)
	writeJSON(w, http.StatusOK, request, page)
}

func othersAfter(all []webhooks.DeliveryView, after, limit int) ([]webhooks.DeliveryView, int, string) {
	listed, total := []webhooks.DeliveryView{}, 0
	for _, d := range all {
		if needsAction(d) {
			continue
		}
		if total++; total > after && len(listed) < limit {
			listed = append(listed, d)
		}
	}
	if after+len(listed) < total {
		return listed, total, strconv.Itoa(after + len(listed))
	}
	return listed, total, ""
}
