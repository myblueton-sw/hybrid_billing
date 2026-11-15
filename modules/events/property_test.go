package events

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type modelReservation struct {
	token, id, subject string
	value              uint64
	expires            time.Time
}

func TestInboxProperty(t *testing.T) {
	l := limits
	l.Seen, l.Revisions = 1000, 1000
	for seed := int64(1); seed <= 100; seed++ {
		random := rand.New(rand.NewSource(seed))
		f := setupWith(t, l)
		f.bind("urn:erp", "c1")
		f.bind("urn:crm", "c1")
		type record struct {
			value uint64
			id    string
			data  string
		}
		committed := map[string]record{}
		seen := map[string]struct {
			at   time.Time
			data string
		}{}
		var open []modelReservation
		data := map[string]string{}
		applied := map[string]int{}
		for step := 0; step < 120; step++ {
			f.advance(time.Duration(random.Intn(1200)) * time.Second)
			now := *f.now
			if len(open) > 0 && random.Intn(3) == 0 {
				index := random.Intn(len(open))
				pick := open[index]
				open = append(open[:index:index], open[index+1:]...)
				if random.Intn(2) == 0 {
					f.must(second(f.r.Commit("w1", pick.token, relay)))
					if pick.subject != "" && pick.value > 0 {
						committed[pick.subject] = record{pick.value, pick.id, data[pick.token]}
					}
					seen[pick.id] = struct {
						at   time.Time
						data string
					}{now, data[pick.token]}
					if pick.value > 0 {
						applied[pick.id+"/"+data[pick.token]]++
					}
				} else {
					f.must(f.r.Abandon("w1", pick.token, relay))
				}
				continue
			}
			source := []string{"urn:erp", "urn:crm"}[random.Intn(2)]
			subject := []string{"s1", "s2", ""}[random.Intn(3)]
			value := uint64(1 + random.Intn(6))
			revisionText := strconv.FormatUint(value, 10)
			if subject == "" || random.Intn(5) == 0 {
				value, revisionText = 0, ""
			}
			id := source + "/" + subject + "-" + revisionText
			content := []string{"d1", "d2"}[random.Intn(2)]
			e := Event{ID: strconv.Itoa(len(id)) + "-" + strconv.Itoa(int(value)) + "-" + subject + "-" + source[4:], Source: source, Type: "t", SchemaVersion: "v1",
				Subject: subject, ObjectRevision: revisionText, Data: map[string]string{"document": content}}
			key := e.Source + "|" + e.ID
			scope := ""
			if subject != "" {
				scope = source + "|" + subject
			}
			got, err := f.r.Receive("w1", "c1", []string{e.ID}, e, relay)
			f.must(err)
			want := DecisionAccepted
			stored, recorded := committed[scope]
			if at, ok := seen[key]; ok && now.Before(at.at.Add(l.InboxRetention)) {
				want = DecisionDuplicate
				if at.data != content {
					want = DecisionConflict
				}
			} else if value > 0 && recorded && stored.value == value && stored.id == key && stored.data == content {
				want = DecisionDuplicate
			} else if value > 0 && recorded && stored.value >= value {
				want = DecisionStale
			} else {
				for _, res := range open {
					if (scope != "" && res.subject == scope || res.id == key) && now.Before(res.expires) {
						want = DecisionInProgress
					}
				}
			}
			if got.Decision != want {
				t.Fatalf("seed %d step %d: %s got %s; want %s", seed, step, key, got.Decision, want)
			}
			if want == DecisionAccepted {
				kept := open[:0]
				for _, res := range open {
					if scope == "" || res.subject != scope {
						if res.id != key {
							kept = append(kept, res)
						}
					}
				}
				open = append(kept, modelReservation{token: got.Token, id: key, subject: scope, value: value, expires: now.Add(l.Reservation)})
				data[got.Token] = content
			}
		}
		for id, count := range applied {
			if count > 1 {
				t.Fatalf("seed %d: %s applied %d times", seed, id, count)
			}
		}
	}
}

func TestCursorProperty(t *testing.T) {
	l := limits
	l.Events, l.Page = 1000, 3
	for seed := int64(1); seed <= 100; seed++ {
		random := rand.New(rand.NewSource(seed))
		f := setupWith(t, l)
		f.register(invoice)
		var times []time.Time
		removed := 0
		for step := 0; step < 60; step++ {
			f.advance(time.Duration(random.Intn(25)) * time.Minute)
			now := *f.now
			for removed < len(times) && !now.Before(times[removed].Add(l.Retention)) {
				removed++
			}
			if random.Intn(2) == 0 {
				f.must(second(f.r.Publish("w1", event("e"+strconv.Itoa(step)), relay)))
				times = append(times, now)
				continue
			}
			after := uint64(random.Intn(len(times) + 2))
			page, err := f.r.Read("w1", after, 5, relay)
			switch {
			case after > uint64(len(times)):
				expect(t, "beyond the head", err, ErrInvalid)
			case after < uint64(removed):
				expect(t, "below the floor", err, ErrExpired)
			default:
				f.must(err)
				want := min(len(times)-int(after), l.Page)
				if len(page.Events) != want || page.Head != uint64(len(times)) || want > 0 && page.Events[0].Sequence != after+1 {
					t.Fatalf("seed %d step %d: after %d got %d events, head %d", seed, step, after, len(page.Events), page.Head)
				}
			}
		}
	}
}
