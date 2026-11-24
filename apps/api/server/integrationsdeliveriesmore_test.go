package server

import (
	"fmt"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func TestIntegrationsReadSaysWhichDeliveryListsWereCut(t *testing.T) {
	list := make([]webhooks.DeliveryView, shownDeliveries+1)
	for n := range list {
		list[n].ID = fmt.Sprint("delivery-", n)
	}
	if got, more := firstDeliveries(list, shownDeliveries); len(got) != shownDeliveries || !more || got[shownDeliveries-1].ID != list[shownDeliveries-1].ID {
		t.Fatalf("one more than the shown number: %d listed, cut %v", len(got), more)
	}
	if got, more := firstDeliveries(list[:shownDeliveries], shownDeliveries); len(got) != shownDeliveries || more {
		t.Fatalf("exactly the shown number: %d listed, cut %v", len(got), more)
	}
	if got, more := firstDeliveries(nil, shownDeliveries); len(got) != 0 || more {
		t.Fatalf("no deliveries: %d listed, cut %v", len(got), more)
	}
	f := setup(t, SeedSample).ready()
	feed := part(f.integrations(f.as(AdminLogin)), "events")
	if more, ok := feed["deliveries_more"].([]any); !ok || len(more) != 0 {
		t.Fatalf("the sample's short lists are reported as cut: %v", feed["deliveries_more"])
	}
	if len(feed["deliveries"].([]any)) == 0 {
		t.Fatal("the sample has no deliveries to list")
	}
}
