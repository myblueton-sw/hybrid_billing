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

func TestDeliveriesReadPagesTheDeliveriesThatNeedNoAction(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	feed := part(f.integrations(root), "events")
	if next, ok := feed["deliveries_next"].(map[string]any); !ok || len(next) != 0 {
		t.Fatalf("the sample's short lists have cursors: %v", feed["deliveries_next"])
	}
	bySubscription := map[string][]map[string]any{}
	for _, raw := range feed["deliveries"].([]any) {
		d := raw.(map[string]any)
		if d["kind"] != webhooks.KindDead && d["hold"] == "" {
			id := fmt.Sprint(d["subscription"])
			bySubscription[id] = append(bySubscription[id], d)
		}
	}
	subscription, want := "", []map[string]any(nil)
	for id, rows := range bySubscription {
		if len(rows) > len(want) {
			subscription, want = id, rows
		}
	}
	if len(want) == 0 {
		t.Fatal("the sample has no delivery that needs no action")
	}
	mixed := []webhooks.DeliveryView{}
	for n := 0; n < 130; n++ {
		d := webhooks.DeliveryView{ID: fmt.Sprint("d-", n), Kind: webhooks.KindRedelivery}
		switch {
		case n%13 == 5:
			d.Kind = webhooks.KindDead
		case n%13 == 9:
			d.Hold = "approver_lost_right"
		}
		mixed = append(mixed, d)
	}
	main, cut := needingActionAnd(mixed, shownDeliveries)
	got, passed := []string{}, shownDeliveries
	for _, d := range main {
		if !needsAction(d) {
			got = append(got, d.ID)
		}
	}
	for cut {
		rows, total, next := othersAfter(mixed, passed, 25)
		if total != 110 {
			t.Fatalf("the deliveries that need no action: %d", total)
		}
		for _, d := range rows {
			if needsAction(d) {
				t.Fatalf("%s needs action and is in the further rows", d.ID)
			}
			got = append(got, d.ID)
		}
		if next == "" {
			break
		}
		fmt.Sscan(next, &passed)
	}
	if len(got) != 110 || got[0] != "d-0" || got[50] != mixed[59].ID || got[109] != "d-129" {
		t.Fatalf("the main read's fifty and the further rows give %d of 110 in order: %s %s %s", len(got), got[0], got[50], got[len(got)-1])
	}
	if rows, total, next := othersAfter(mixed, 500, 25); len(rows) != 0 || total != 110 || next != "" {
		t.Errorf("past the end: %d rows, next %q", len(rows), next)
	}
	path := "/api/v1/integrations/deliveries?subscription=" + subscription
	entire := f.readAs(root, path)
	if rows, _ := entire.body["deliveries"].([]any); entire.status != 200 || len(rows) != len(want) || entire.body["next"] != "" || whole(entire.body["total"]) != len(want) || entire.body["order"] != "recorded" ||
		entire.body["rows_can_move"] != true || entire.body["subscription"] != subscription {
		t.Fatalf("the whole list: %s", entire.raw[:min(len(entire.raw), 300)])
	}
	seen, after := []string{}, ""
	for pages := 0; pages <= len(want); pages++ {
		page := f.readAs(root, path+"&limit=1"+after)
		rows, _ := page.body["deliveries"].([]any)
		if page.status != 200 || len(rows) != 1 || whole(page.body["total"]) != len(want) {
			t.Fatalf("page %d: %s", pages, page.raw[:min(len(page.raw), 300)])
		}
		row := rows[0].(map[string]any)
		if row["kind"] == webhooks.KindDead || row["hold"] != "" {
			t.Fatalf("a delivery that needs action is in the further read: %v", row["id"])
		}
		seen = append(seen, fmt.Sprint(row["id"]))
		if page.body["next"] == "" {
			break
		}
		after = "&after=" + fmt.Sprint(page.body["next"])
	}
	if len(seen) != len(want) {
		t.Fatalf("one at a time gives %d of %d", len(seen), len(want))
	}
	inMain := map[string]bool{}
	for _, d := range want {
		inMain[fmt.Sprint(d["id"])] = true
	}
	for _, id := range seen {
		if !inMain[id] {
			t.Errorf("%s is not among the main read's deliveries of the subscription", id)
		}
	}
	if past := f.readAs(root, path+"&after=500"); past.status != 200 || len(past.body["deliveries"].([]any)) != 0 || past.body["next"] != "" {
		t.Errorf("past the end: %s", past.raw[:min(len(past.raw), 200)])
	}
	base := "/api/v1/integrations/deliveries"
	for _, query := range []string{"", "?subscription=", "?subscription=never-proposed", "?after=1", "?subscription=" + subscription + "&after=0", "?subscription=" + subscription + "&after=x",
		"?subscription=" + subscription + "&limit=0", "?subscription=" + subscription + "&limit=51", "?subscription=" + subscription + "&q=a", "?subscription=" + subscription + "&subscription=" + subscription} {
		if refused := f.readAs(root, base+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "integrations-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, path); out.status != 422 {
		t.Errorf("another workspace: %d", out.status)
	}
}
