package incidents

import (
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"
)

type mIncident struct {
	id         string
	key        [2]string
	severity   int
	first      time.Time
	last       time.Time
	count      int
	late       int
	resolved   bool
	resolvedAt time.Time
	recurs     string
	blocking   bool
}

func TestIncidentsAndNotificationsMatchAModel(t *testing.T) {
	severities := []string{SeverityInfo, SeverityWarning, SeverityCritical}
	stats := map[string]int{}
	for seed := int64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := open(t, Limits{Workspaces: 1, Incidents: 500, Signals: 2000, Recipients: 4, Subscriptions: 10, Notifications: 5000})
		f.subscribe(t, "ana", "c", SeverityInfo)
		renotify := 30 * time.Minute
		must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Renotify: map[string]time.Duration{SeverityInfo: renotify, SeverityWarning: renotify, SeverityCritical: renotify}}, human("ana")))
		var incidents []*mIncident
		open := map[[2]string]*mIncident{}
		handed := map[string]time.Time{}
		handedSeverity := map[string]int{}
		snoozed := map[string]time.Time{}
		denied := map[string]bool{}
		next := 0
		for step := 0; step < 120; step++ {
			now := f.clock.read()
			subject := "k" + strconv.Itoa(rng.Intn(3))
			key := [2]string{"c", subject}
			switch rng.Intn(7) {
			case 0, 1:
				next++
				sev := rng.Intn(3)
				occurred := now.Add(-time.Duration(rng.Intn(180)) * time.Minute)
				blocking := rng.Intn(4) == 0
				got := must(f.r.Raise("ws", signal("s"+strconv.Itoa(next), "c", subject, severities[sev], blocking, occurred), system("m")))
				var target *mIncident
				for _, inc := range incidents {
					if inc.key == key && inc.resolved && !occurred.After(inc.resolvedAt) {
						target = inc
						break
					}
				}
				switch {
				case target != nil:
					target.late++
					if occurred.Before(target.first) {
						target.first = occurred
					}
					stats["late"]++
				case open[key] != nil:
					target = open[key]
					target.count++
					if occurred.Before(target.first) {
						target.first = occurred
					}
					if occurred.After(target.last) {
						target.last = occurred
					}
					if sev+1 > target.severity {
						target.severity = sev + 1
					}
					target.blocking = target.blocking || blocking
				default:
					target = &mIncident{id: "inc-" + strconv.Itoa(len(incidents)+1), key: key, severity: sev + 1, first: occurred, last: occurred, count: 1, blocking: blocking}
					for _, old := range incidents {
						if old.key == key && old.resolved {
							target.recurs = old.id
						}
					}
					incidents = append(incidents, target)
					open[key] = target
				}
				if got.ID != target.id || got.Count != target.count || got.Late != target.late || !got.First.Equal(target.first) || !got.Last.Equal(target.last) || rank[got.Severity] != target.severity || got.Recurs != target.recurs || got.Blocking != target.blocking {
					t.Fatalf("seed %d: raise %+v model %+v", seed, got, target)
				}
			case 2:
				if inc := open[key]; inc != nil {
					must(f.r.Resolve("ws", inc.id, Resolution{Reason: ReasonFixed, Evidence: "e"}, system("auto")))
					inc.resolved, inc.resolvedAt = true, now
					delete(open, key)
					stats["resolved"]++
				}
			case 3:
				if denied[subject] {
					f.perms.allow("ana", subject)
				} else {
					f.perms.deny("ana", subject)
				}
				denied[subject] = !denied[subject]
			case 4:
				if inc := open[key]; inc != nil && !denied[subject] {
					until := now.Add(time.Duration(1+rng.Intn(90)) * time.Minute)
					must(f.r.Snooze("ws", inc.id, until, human("ana")))
					snoozed[inc.id] = until
				}
			case 5:
				f.clock.advance(time.Duration(rng.Intn(45)) * time.Minute)
			case 6:
				var want []string
				for _, inc := range incidents {
					if inc.resolved || snoozed[inc.id].After(now) || denied[inc.key[1]] {
						continue
					}
					if at, ok := handed[inc.id]; ok && inc.severity <= handedSeverity[inc.id] && now.Sub(at) < renotify {
						continue
					}
					want = append(want, inc.id)
				}
				var got []string
				for _, n := range f.cycle(t) {
					got = append(got, n.Delivered...)
				}
				sort.Strings(want)
				sort.Strings(got)
				if len(got) != len(want) {
					t.Fatalf("seed %d step %d: delivered %v model %v", seed, step, got, want)
				}
				for index := range got {
					if got[index] != want[index] {
						t.Fatalf("seed %d step %d: delivered %v model %v", seed, step, got, want)
					}
				}
				for _, id := range got {
					handed[id] = now
					for _, inc := range incidents {
						if inc.id == id {
							handedSeverity[id] = inc.severity
						}
					}
				}
				stats["delivered"] += len(got)
			}
			want := 0
			if inc := open[key]; inc != nil && inc.blocking {
				want = 1
			}
			if blocked := must(f.r.Blocked("ws", subject, system("w"))); len(blocked) != want {
				t.Fatalf("seed %d: blocked %d model %d", seed, len(blocked), want)
			}
		}
	}
	if stats["late"] < 10 || stats["resolved"] < 50 || stats["delivered"] < 100 {
		t.Fatalf("weak exploration: %v", stats)
	}
}
