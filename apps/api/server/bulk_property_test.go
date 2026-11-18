package server

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"maps"
	"math/rand/v2"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
)

func modelMatches(m *modelPerson, filter url.Values, now time.Time) bool {
	for key := range filter {
		if !m.holds(key, filter.Get(key), now) {
			return false
		}
	}
	return true
}

func modelSweep(people map[string]*modelPerson, now time.Time) {
	for _, m := range people {
		if m.membership == "active" && !m.until.IsZero() && !now.Before(m.until) {
			m.membership = "suspended"
		}
	}
}

func modelReason(m *modelPerson, filter url.Values, now time.Time) string {
	switch {
	case m.status == "departed":
		return bulkrevoke.ExcludedNotFound
	case !modelMatches(m, filter, now):
		return bulkrevoke.ExcludedNoLongerMatches
	case m.membership == membershipNone:
		return bulkrevoke.ExcludedNoMembership
	case m.member(now) != "active":
		return bulkrevoke.ExcludedAlreadyRevoked
	}
	return "target"
}

func modelCanonical(filter url.Values) string {
	out := ""
	for _, key := range slices.Sorted(maps.Keys(filter)) {
		value := filter.Get(key)
		out += strconv.Itoa(len(key)) + ":" + key + strconv.Itoa(len(value)) + ":" + value
	}
	return out
}

func modelDigest(effect string, filter url.Values, matched int, fixedAt time.Time, revision string, v bulkrevoke.View) string {
	sum := sha256.New()
	put := func(texts ...string) {
		for _, text := range texts {
			var size [8]byte
			binary.BigEndian.PutUint64(size[:], uint64(len(text)))
			sum.Write(size[:])
			sum.Write([]byte(text))
		}
	}
	put("bulk_revocation_v1", effect, "filter", modelCanonical(filter), strconv.Itoa(matched), fixedAt.UTC().Format(time.RFC3339Nano), revision, strconv.Itoa(len(v.Targets)))
	targets := append([]bulkrevoke.Target(nil), v.Targets...)
	sort.Slice(targets, func(a, b int) bool { return targets[a].User < targets[b].User })
	for _, t := range targets {
		until := ""
		if !t.Until.IsZero() {
			until = t.Until.UTC().Format(time.RFC3339Nano)
		}
		put(t.User, t.Login, t.Role, until, strconv.FormatUint(t.Version, 10), strconv.FormatUint(t.Epoch, 10), t.UserState, t.MembershipState, strconv.FormatBool(t.Administrator))
	}
	exclusions := append([]bulkrevoke.Exclusion(nil), v.Exclusions...)
	sort.Slice(exclusions, func(a, b int) bool { return exclusions[a].User < exclusions[b].User })
	put(strconv.Itoa(len(exclusions)))
	for _, x := range exclusions {
		put(x.User, x.Login, x.Reason)
	}
	return hex.EncodeToString(sum.Sum(nil))
}

type countWant struct {
	count            int
	complete, over   bool
	budget, maxMatch int
}

func modelCount(order []string, matches func(string) bool, budget, limit int) countWant {
	out := countWant{complete: len(order) == 0, budget: budget, maxMatch: limit}
	for i, id := range order {
		if i == budget {
			break
		}
		if matches(id) {
			out.count++
			if out.count == limit+1 {
				out.over, out.complete = true, i == len(order)-1
				break
			}
		}
		out.complete = i == len(order)-1
	}
	return out
}

func mutate(p *population, rng *rand.Rand, people map[string]*modelPerson, ids []string) {
	if len(ids) == 0 {
		return
	}
	now := p.f.s.clock()
	for n := 0; n < 3+rng.IntN(6); n++ {
		m := people[ids[rng.IntN(len(ids))]]
		switch rng.IntN(5) {
		case 0:
			if m.status == "active" {
				p.suspendUser(m.id)
				m.status = "suspended"
			}
		case 1:
			if m.membership == "active" {
				p.suspendMembership(m.id)
				m.membership = "suspended"
			}
		case 2:
			if m.membership == "active" {
				m.until = now.Add(time.Duration(1+rng.IntN(20)) * time.Minute)
				p.until(m.id, m.until)
			}
		case 3:
			if m.status != "departed" {
				p.depart(m.id)
				*m = modelPerson{id: m.id, status: "departed", membership: membershipNone, signin: []string{signinNotSet}}
			}
		default:
			if m.status != "departed" {
				p.holder(m.id)
				m.holder = true
			}
		}
	}
}

func TestFilterListsMatchAnIndependentModel(t *testing.T) {
	sizes := []int{20, 60, 150, 300}
	reached := map[string]int{}
	for round, size := range sizes {
		f := setupWith(t, func(c *Config) { c.Passwords.Links = 10000 })
		p := newPopulation(f)
		rng := rand.New(rand.NewPCG(uint64(round)+1, 323))
		people := generatePeople(t, p, rng, size)
		f.advance(time.Duration(rng.IntN(240)) * time.Minute)
		p.refresh()
		modelSweep(people, f.s.clock())
		root := f.as(AdminLogin)
		for trial := 0; trial < 10; trial++ {
			swept(f, p, root)
			now := f.s.clock()
			filter := randomFilter(rng, people)
			want := []string{}
			for id, m := range people {
				if modelMatches(m, filter, now) {
					want = append(want, id)
				}
			}
			slices.Sort(want)
			listed, _ := f.walk(root, filter.Encode())
			seedMatches := map[string]bool{}
			for _, id := range listed {
				if people[id] == nil {
					seedMatches[id] = true
				}
			}
			order := append(append([]string{}, f.s.dir.order...), f.s.dir.departed...)
			for c := 0; c < 3; c++ {
				budget, limit := 1+rng.IntN(len(order)+5), 1+rng.IntN(len(order)+5)
				f.s.cfg.Access.UserCountScan, f.s.cfg.Access.BulkMaxMatching = budget, limit
				expected := modelCount(order, func(id string) bool {
					if m := people[id]; m != nil {
						return modelMatches(m, filter, now)
					}
					return seedMatches[id]
				}, budget, limit)
				got := f.count(root, "?"+filter.Encode())
				if got.status != 200 || whole(got.body["count"]) != expected.count || got.body["complete"] != expected.complete || got.body["over_limit"] != expected.over {
					t.Fatalf("round %d trial %d %v budget %d limit %d: %s; model %+v", round, trial, filter, budget, limit, got.raw, expected)
				}
				reached["count complete "+strconv.FormatBool(expected.complete)+" over "+strconv.FormatBool(expected.over)]++
			}
			f.s.cfg.Access.UserCountScan, f.s.cfg.Access.BulkMaxMatching = 20000, 5000
			fields := map[string]string{}
			for key := range filter {
				fields[key] = filter.Get(key)
			}
			fixed := f.run(root, "bulk_revocation.add_matching", bulkTarget, fields)
			if len(want) == 0 && len(seedMatches) == 0 {
				if code(fixed) != "prerequisite_failed" {
					t.Fatalf("round %d trial %d %v: nobody matches but %s", round, trial, filter, fixed.raw)
				}
				reached["none"]++
				continue
			}
			if fixed.status != 200 {
				t.Fatalf("round %d trial %d %v: %s", round, trial, filter, fixed.raw)
			}
			draft := field(fixed, "draft")
			if got := generatedOnly(f.draftUsers(SampleAdmin, draft), people); !slices.Equal(got, want) {
				t.Fatalf("round %d trial %d %v: fixed %v; model %v", round, trial, filter, got, want)
			}
			mutate(p, rng, people, want)
			f.advance(time.Duration(rng.IntN(25)) * time.Minute)
			p.refresh()
			root = f.as(AdminLogin)
			now = f.s.clock()
			modelSweep(people, now)
			swept(f, p, root)
			prepared := f.prepare(root, draft, "suspend_membership")
			targets := 0
			for _, id := range want {
				if modelReason(people[id], filter, now) == "target" {
					targets++
				}
			}
			if code(prepared) == "invalid_stage" {
				if targets > 0 {
					t.Fatalf("round %d trial %d: refused with %d model targets", round, trial, targets)
				}
				reached["empty"]++
				f.discard(root, draft)
				continue
			}
			if prepared.status != 200 {
				t.Fatalf("round %d trial %d: %s", round, trial, prepared.raw)
			}
			id := field(prepared, "bulk")
			reasons := f.reasons(root, id)
			for _, user := range want {
				if got, model := reasons[user], modelReason(people[user], filter, now); got != model {
					t.Fatalf("round %d trial %d %v: %s %s; model %s %+v", round, trial, filter, user, got, model, *people[user])
				}
				reached[reasons[user]]++
			}
			v, _ := f.s.bulk.Get(SampleWorkspace, SampleAdmin, id)
			fixedAt, _ := time.Parse(time.RFC3339, field(fixed, "fixed_at"))
			if len(v.Targets)+len(v.Exclusions) != whole(field(fixed, "matched")) ||
				modelDigest("suspend_membership", filter, whole(field(fixed, "matched")), fixedAt, field(fixed, "fixed_revision"), v) != field(prepared, "digest") {
				t.Fatalf("round %d trial %d: digest or counts %s", round, trial, prepared.raw)
			}
			f.discard(root, id)
		}
	}
	for _, key := range []string{"target", bulkrevoke.ExcludedNoLongerMatches, bulkrevoke.ExcludedNotFound, bulkrevoke.ExcludedAlreadyRevoked, "count complete true over false",
		"count complete false over false", "count complete false over true"} {
		if reached[key] == 0 {
			t.Errorf("never reached %s (%v)", key, reached)
		}
	}
}
