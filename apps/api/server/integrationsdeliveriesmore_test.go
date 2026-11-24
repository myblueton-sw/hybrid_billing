package server

import (
	"fmt"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func TestIntegrationsReadListsEveryDeliveryNeedingActionAndSaysWhichListsWereCut(t *testing.T) {
	of := func(kinds ...string) []webhooks.DeliveryView {
		list := make([]webhooks.DeliveryView, len(kinds))
		for n, kind := range kinds {
			list[n] = webhooks.DeliveryView{ID: fmt.Sprint("delivery-", n), Kind: kind}
		}
		return list
	}
	repeat := func(kind string, n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = kind
		}
		return out
	}
	count := func(list []webhooks.DeliveryView, kind string) int {
		n := 0
		for _, d := range list {
			if d.Kind == kind {
				n++
			}
		}
		return n
	}
	queued, dead := repeat(webhooks.KindRedelivery, shownDeliveries+5), repeat(webhooks.KindDead, shownDeliveries+30)
	crowded := of(append(append([]string{webhooks.KindOpen}, queued...), dead...)...)
	got, more := needingActionAnd(crowded, shownDeliveries)
	if count(got, webhooks.KindDead) != len(dead) || len(got) != len(dead)+shownDeliveries || !more || got[0].Kind != webhooks.KindOpen {
		t.Fatalf("a crowded subscription: %d listed, %d dead letters of %d, cut %v", len(got), count(got, webhooks.KindDead), len(dead), more)
	}
	if got, more := needingActionAnd(of(dead...), shownDeliveries); len(got) != len(dead) || more {
		t.Fatalf("only dead letters, more than the shown number of others: %d listed, cut %v", len(got), more)
	}
	if got, more := needingActionAnd(of(repeat(webhooks.KindCancelled, shownDeliveries)...), shownDeliveries); len(got) != shownDeliveries || more {
		t.Fatalf("exactly the shown number of others: %d listed, cut %v", len(got), more)
	}
	if got, more := needingActionAnd(of(append(repeat(webhooks.KindCancelled, shownDeliveries+1), webhooks.KindDead)...), shownDeliveries); len(got) != shownDeliveries+1 || !more || got[shownDeliveries].Kind != webhooks.KindDead {
		t.Fatalf("one other too many, with a dead letter after it: %d listed, cut %v", len(got), more)
	}
	held := of(repeat(webhooks.KindRedelivery, shownDeliveries+3)...)
	held[shownDeliveries+2].Hold = "approver_lost_right"
	if got, more := needingActionAnd(held, shownDeliveries); len(got) != shownDeliveries+1 || !more || got[shownDeliveries].ID != held[shownDeliveries+2].ID {
		t.Fatalf("a held redelivery behind the shown number of others: %d listed, cut %v", len(got), more)
	}
	if got, more := needingActionAnd(nil, shownDeliveries); len(got) != 0 || more {
		t.Fatalf("no deliveries: %d listed, cut %v", len(got), more)
	}
	f := setup(t, SeedSample).ready()
	whole := deliveryPage
	deliveryPage = 1
	paged := part(f.integrations(f.as(AdminLogin)), "events")
	deliveryPage = whole
	feed := part(f.integrations(f.as(AdminLogin)), "events")
	if more, ok := feed["deliveries_more"].([]any); !ok || len(more) != 0 {
		t.Fatalf("the sample's short lists are reported as cut: %v", feed["deliveries_more"])
	}
	if n := len(feed["deliveries"].([]any)); n == 0 || fmt.Sprint(feed["deliveries"]) != fmt.Sprint(paged["deliveries"]) {
		t.Fatalf("the sample lists %d deliveries in one page and other rows when read one at a time: %v", n, paged["deliveries"])
	}
}
