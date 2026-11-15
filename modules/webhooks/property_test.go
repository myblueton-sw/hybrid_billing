package webhooks

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestDeliveryProperty(t *testing.T) {
	l := limits
	l.Dead, l.Attempts, l.MaxAge = 1000, 3, 24*time.Hour
	for seed := int64(1); seed <= 60; seed++ {
		random := rand.New(rand.NewSource(seed))
		f := setupWith(t, l)
		f.subscribe("s1", "d1")
		f.subscribe("s2", "d2", "other")
		published := map[string][]string{}
		sequence := map[string]uint64{}
		count := uint64(0)
		ended := map[string][]string{}
		for step := 0; step < 150; step++ {
			if random.Intn(10) == 0 {
				outage := random.Intn(2) == 0
				f.fk.set(func(k *fakes) {
					k.allowErr = nil
					if outage {
						k.allowErr = boom
					}
				})
			}
			if random.Intn(15) == 0 {
				id := []string{"s1", "s2"}[random.Intn(2)]
				if view, _ := f.r.SubscriptionOf("w1", id); view.State == SubscriptionActive {
					f.must(second(f.r.PauseSubscription("w1", id, ann)))
				} else {
					f.must(second(f.r.RequestResume("w1", id, ann)))
					f.must(second(f.r.ApproveResume("w1", id, bob)))
				}
			}
			switch random.Intn(4) {
			case 0:
				kind := []string{invoice.Name, "other"}[random.Intn(2)]
				id := "e" + strconv.Itoa(step)
				f.publish(id, kind)
				count++
				sequence[id] = count
				if kind == invoice.Name {
					published["s1"] = append(published["s1"], id)
				} else {
					published["s2"] = append(published["s2"], id)
				}
			default:
				f.advance(time.Duration(random.Intn(90)) * time.Second)
				leases, err := f.r.Dispatch("w1", 5, relay)
				f.must(err)
				for _, lease := range leases {
					if random.Intn(5) == 0 {
						continue
					}
					if _, err := f.r.Confirm("w1", lease.Delivery, lease.Token, relay); err != nil {
						continue
					}
					status := []string{StatusDelivered, StatusDelivered, StatusFailed, StatusUnknown, StatusRejected, StatusThrottled}[random.Intn(6)]
					view, err := f.r.Report("w1", lease.Delivery, lease.Token, Outcome{Status: status, RetryAfter: time.Duration(random.Intn(5)) * time.Second}, relay)
					f.must(err)
					if view.State == JobSucceeded || view.State == JobFailed {
						ended[lease.Subscription] = append(ended[lease.Subscription], lease.EventID)
					}
				}
			}
		}
		f.fk.set(func(k *fakes) { k.allowErr = nil })
		for _, id := range []string{"s1", "s2"} {
			if view, _ := f.r.SubscriptionOf("w1", id); view.State != SubscriptionActive {
				f.must(second(f.r.RequestResume("w1", id, ann)))
				f.must(second(f.r.ApproveResume("w1", id, bob)))
			}
		}
		for round := 0; round < 400; round++ {
			for _, id := range []string{"s1", "s2"} {
				if view, _ := f.r.SubscriptionOf("w1", id); view.State == SubscriptionSuspended {
					f.must(second(f.r.RequestResume("w1", id, ann)))
					f.must(second(f.r.ApproveResume("w1", id, bob)))
				}
			}
			f.advance(2 * time.Minute)
			leases, err := f.r.Dispatch("w1", 5, relay)
			f.must(err)
			for _, lease := range leases {
				f.confirm(lease)
				f.report(lease, StatusDelivered, 0)
				ended[lease.Subscription] = append(ended[lease.Subscription], lease.EventID)
			}
		}
		for _, id := range []string{"s1", "s2"} {
			view, _ := f.r.SubscriptionOf("w1", id)
			kept := []string{}
			for _, event := range published[id] {
				lost := false
				for _, gap := range view.Gaps {
					lost = lost || sequence[event] > gap.After && sequence[event] <= gap.To
				}
				if !lost {
					kept = append(kept, event)
				}
			}
			published[id] = kept
			if strconv.Itoa(len(ended[id])) != strconv.Itoa(len(published[id])) {
				t.Fatalf("seed %d %s: ended %v; published %v", seed, id, ended[id], published[id])
			}
			for index := range published[id] {
				if ended[id][index] != published[id][index] {
					t.Fatalf("seed %d %s: order %v; published %v", seed, id, ended[id], published[id])
				}
			}
		}
	}
}
