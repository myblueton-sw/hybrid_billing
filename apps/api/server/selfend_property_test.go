package server

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"
)

type modelWorld struct {
	f      world
	t      *testing.T
	live   map[string]map[string]actor
	tokens map[string]map[string]bool
	last   map[string]string
	logins []string
	r      *rand.Rand
}

func (m *modelWorld) pick(login string) (actor, string, bool) {
	keys := []string{}
	for h := range m.live[login] {
		keys = append(keys, h)
	}
	if len(keys) == 0 {
		return actor{}, "", false
	}
	sort.Strings(keys)
	h := keys[m.r.Intn(len(keys))]
	return m.live[login][h], h, true
}

func (m *modelWorld) open(login string) {
	if len(m.live[login]) >= 9 {
		return
	}
	a := m.f.as(login)
	m.live[login][handleOfActor(a)] = a
}

func (m *modelWorld) check(step int) {
	for _, login := range m.logins {
		snap := m.f.s.sessions.snapshot(login)
		got := map[string]bool{}
		for _, s := range snap.sessions {
			got[handleOf(s.verifier)] = true
		}
		if len(got) != len(m.live[login]) {
			m.t.Fatalf("step %d %s: store %d model %d", step, login, len(got), len(m.live[login]))
		}
		for h := range m.live[login] {
			if !got[h] {
				m.t.Fatalf("step %d %s: model session %s missing", step, login, h)
			}
		}
		if snap.token != m.last[login] {
			if snap.token != "" && m.tokens[login][snap.token] {
				m.t.Fatalf("step %d %s: token repeated", step, login)
			}
			m.tokens[login][snap.token] = true
			m.last[login] = snap.token
		}
	}
}

func (m *modelWorld) tokensBefore() map[string]string {
	out := map[string]string{}
	for _, login := range m.logins {
		out[login] = m.f.s.sessions.snapshot(login).token
	}
	return out
}

func (m *modelWorld) endOthers(step int, login string) {
	a, current, ok := m.pick(login)
	if !ok {
		return
	}
	page := m.f.testSessionSnapshot(a)
	listed := []string{}
	for h := range m.live[login] {
		if h != current {
			listed = append(listed, h)
		}
	}
	sort.Strings(listed)
	confirmed := []string{}
	for _, h := range listed {
		if m.r.Intn(3) > 0 {
			confirmed = append(confirmed, h)
		}
	}
	if m.r.Intn(4) == 0 {
		confirmed = append(confirmed, fmt.Sprintf("%016x", m.r.Int63()))
	}
	if len(confirmed) == 0 {
		confirmed = append(confirmed, fmt.Sprintf("%016x", m.r.Int63()))
	}
	generation := fmt.Sprint(page["sessions_generation"])
	if m.r.Intn(3) == 0 {
		m.open(login)
	}
	others := map[string]bool{}
	for _, l := range m.logins {
		if l != login {
			others[l] = true
		}
	}
	before := m.tokensBefore()
	got := m.f.selfPost(a, "sessions/end", endJSON(generation, confirmed))
	if got.status != 202 {
		m.t.Fatalf("step %d: %+v", step, got)
	}
	want := map[string]string{}
	confirmedSet := map[string]bool{}
	for _, h := range confirmed {
		confirmedSet[h] = true
		if _, live := m.live[login][h]; live {
			want[h] = resultEnded
		} else {
			want[h] = resultGone
		}
	}
	if fmt.Sprint(resultsOf(got.body)) != fmt.Sprint(want) {
		m.t.Fatalf("step %d I2 results %v want %v", step, resultsOf(got.body), want)
	}
	notIncluded := []string{}
	for h := range m.live[login] {
		if h != current && !confirmedSet[h] {
			notIncluded = append(notIncluded, h)
		}
	}
	sort.Strings(notIncluded)
	if fmt.Sprint(notIncludedOf(got.body)) != fmt.Sprint(notIncluded) {
		m.t.Fatalf("step %d I2 not included %v want %v", step, notIncludedOf(got.body), notIncluded)
	}
	ended := 0
	for h, r := range want {
		if r == resultEnded {
			delete(m.live[login], h)
			ended++
		}
	}
	cleanup := got.body["cleanup"].(map[string]any)
	sum := 0
	for _, k := range []string{"done", "already_ended", "failed", "pending"} {
		sum += countOf(cleanup[k])
	}
	if fmt.Sprint(got.body["count"]) != fmt.Sprint(ended) || sum != ended {
		m.t.Fatalf("step %d I4 count %v sum %d ended %d", step, got.body["count"], sum, ended)
	}
	if m.f.me(a).status != 200 {
		m.t.Fatalf("step %d I3 current session ended", step)
	}
	after := m.tokensBefore()
	for l := range others {
		if before[l] != after[l] {
			m.t.Fatalf("step %d I5 %s token changed", step, l)
		}
	}
	replay := m.f.selfPost(a, "sessions/end", endJSON(generation, confirmed))
	if replay.body["replay"] != true || m.tokensBefore()[login] != after[login] {
		m.t.Fatalf("step %d I6 replay %+v", step, replay)
	}
}

func TestSelfEndModel(t *testing.T) {
	for seed := int64(1); seed <= 12; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			f := memberWorld(t)
			m := &modelWorld{f: f, t: t, live: map[string]map[string]actor{}, tokens: map[string]map[string]bool{}, last: map[string]string{},
				logins: []string{MemberLogin, AdminLogin}, r: rand.New(rand.NewSource(seed))}
			for _, l := range m.logins {
				m.live[l], m.tokens[l] = map[string]actor{}, map[string]bool{}
				m.open(l)
			}
			for step := 0; step < 60; step++ {
				login := m.logins[m.r.Intn(len(m.logins))]
				switch m.r.Intn(7) {
				case 0, 1:
					m.open(login)
				case 2:
					if a, h, ok := m.pick(login); ok && len(m.live[login]) > 1 {
						m.f.logout(a)
						delete(m.live[login], h)
					}
				case 3:
					m.endOthers(step, login)
				case 4:
					if a, current, ok := m.pick(login); ok {
						for h := range m.live[login] {
							if h != current {
								before := m.f.s.counter
								got := m.f.selfPost(a, "sessions/end", `{"session":"`+h+`"}`)
								if got.status != 202 || m.f.s.counter != before+1 {
									t.Fatalf("step %d end one %+v", step, got)
								}
								delete(m.live[login], h)
								break
							}
						}
					}
				case 5:
					if a, _, ok := m.pick(login); ok {
						before, tokens := m.f.s.counter, m.tokensBefore()
						got := m.f.selfPost(a, "sessions/end", `{"session":"others","generation":"x","handles":[]}`)
						if got.status != 422 || m.f.s.counter != before || fmt.Sprint(m.tokensBefore()) != fmt.Sprint(tokens) {
							t.Fatalf("step %d I7 %+v", step, got)
						}
					}
				default:
					m.f.advance(time.Duration(m.r.Intn(3000)) * time.Millisecond)
					m.f.drainSelf()
				}
				m.check(step)
			}
		})
	}
}
