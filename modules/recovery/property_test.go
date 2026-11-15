package recovery

import (
	"math/rand"
	"testing"
	"time"
)

type modelEvent struct {
	seq  uint64
	at   time.Time
	kind string
}

type modelCall struct {
	key    string
	number int64
	at     time.Time
}

type modelSite struct {
	id         string
	writer     Actor
	verifier   Actor
	gen        uint64
	state      string
	open       bool
	point      bool
	asOf       time.Time
	beganSeq   uint64
	verified   bool
	reconciled map[string]bool
	events     []modelEvent
	settled    map[uint64]bool
	lost       []uint64
	tokens     []Token
	fenced     map[string]bool
	high       int64
	seq        uint64
	stopped    time.Time
	calls      []modelCall
}

func (m *modelSite) pending() []uint64 {
	out := []uint64{}
	for _, e := range m.events {
		if e.at.After(m.asOf.Add(-time.Minute)) && e.seq <= m.beganSeq && !m.settled[e.seq] {
			out = append(out, e.seq)
		}
	}
	return out
}

func TestRecoveryProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	for seed := 0; seed < runs; seed++ {
		runRecoveryProperty(t, int64(seed))
	}
}

func runRecoveryProperty(t *testing.T, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	f := standard(t)
	sites := []*modelSite{
		{id: "i1", writer: erp, verifier: ver, gen: 1, state: StateServing, fenced: map[string]bool{}, settled: map[uint64]bool{}},
		{id: "i2", writer: Actor{ID: "b-w-erp", Kind: KindSystem}, verifier: Actor{ID: "b-v-1", Kind: KindSystem}, gen: 1, state: StateServing, fenced: map[string]bool{}, settled: map[uint64]bool{}},
	}
	keys := []string{"k1", "k2", "k3"}
	for step := 0; step < 120; step++ {
		m := sites[rng.Intn(2)]
		var err error
		want := true
		switch op := rng.Intn(11); op {
		case 0:
			want = m.state == StateServing
			var tok Token
			tok, err = f.r.Permit(m.id, m.writer.ID, m.writer)
			if err == nil {
				m.tokens = append(m.tokens, tok)
			}
		case 1:
			if len(m.tokens) == 0 {
				continue
			}
			tok := m.tokens[rng.Intn(len(m.tokens))]
			key := keys[rng.Intn(len(keys))]
			number := int64(1 + rng.Intn(5))
			want = m.state == StateServing && tok.Generation == m.gen && f.at().Before(tok.Expiry) && !m.fenced[key] && number > m.high
			var c CallView
			c, err = f.r.Call(m.id, m.writer.ID, tok, Target{Workspace: "ws", Key: key, Series: "inv", Number: number}, m.writer)
			if err == nil {
				f.must(second(f.r.Done(m.id, c.ID, OutcomeConfirmed, m.writer)))
				m.calls = append(m.calls, modelCall{key: key, number: number, at: f.at()})
			}
		case 2:
			want = m.state == StateServing
			err = f.r.Lock(m.id, "stop", ann)
			if m.state == StateLocked {
				want = true
			}
			if err == nil && m.state == StateServing {
				m.gen++
				m.state = StateLocked
				m.stopped = f.at()
			}
		case 3:
			want = !(m.open && m.point)
			asOf := f.at().Add(-time.Duration(rng.Intn(3)) * time.Minute)
			err = f.try(Change{Kind: ChangeBegin, Begin: f.point(m.id, asOf)})
			if want && err == nil {
				if m.state == StateServing {
					m.stopped = f.at()
				}
				for _, c := range m.calls {
					if c.at.After(asOf.Add(-time.Minute)) {
						m.fenced[c.key] = true
						if c.number > m.high {
							m.high = c.number
						}
					}
				}
				m.gen++
				m.state, m.open, m.point, m.asOf, m.verified = StateRecovering, true, true, asOf, false
				m.reconciled, m.settled, m.lost = map[string]bool{}, map[uint64]bool{}, nil
				m.beganSeq = m.seq
			}
		case 4:
			if !m.open || !m.point || len(m.settled) > 0 {
				continue
			}
			for _, inv := range manifest() {
				f.must(second(f.r.Report(m.id, inv, m.verifier)))
			}
			m.verified = true
		case 5:
			if !m.open || !m.point {
				continue
			}
			x := []string{"erp-x", "iss-x"}[rng.Intn(2)]
			if m.reconciled[x] {
				continue
			}
			f.advance(5 * time.Minute)
			key := keys[rng.Intn(len(keys))]
			number := int64(rng.Intn(6))
			items := []Item{{Workspace: "ws", Key: key, Kind: ItemDelivered, At: f.at()}}
			if number > 0 {
				items = []Item{{Workspace: "ws", Key: key, Kind: ItemIssued, At: f.at(), Series: "inv", Number: number}}
			}
			through := f.at().Add(-time.Duration(rng.Intn(4)) * time.Minute)
			if through.Before(m.asOf.Add(-time.Hour)) {
				through = m.asOf.Add(-time.Hour)
			}
			want = !through.Before(m.stopped.Add(3 * time.Minute))
			_, err = f.r.Reconcile(m.id, x, Page{Number: 1, From: m.asOf.Add(-time.Hour), Through: through, Final: true, Rotated: true, Items: items}, m.verifier)
			if err != nil {
				break
			}
			m.reconciled[x] = true
			m.fenced[key] = true
			if number > m.high {
				m.high = number
			}
		case 6:
			ev, e := f.r.Record(m.id, Event{Kind: []string{EventDelete, EventHold, EventRevokeKey}[rng.Intn(3)], Domain: DomainEvidence, Key: "e"}, m.writer)
			f.must(e)
			m.seq = ev.Sequence
			m.events = append(m.events, modelEvent{seq: ev.Sequence, at: ev.At, kind: ev.Event.Kind})
		case 7:
			if !m.open || !m.point || !m.verified {
				continue
			}
			p := m.pending()
			if len(p) == 0 {
				continue
			}
			kind := ""
			for _, e := range m.events {
				if e.seq == p[0] {
					kind = e.kind
				}
			}
			if rng.Intn(3) == 0 {
				want = kind != EventRevokeKey
				_, err = f.r.Unapplicable(m.id, p[0], "gone", m.verifier)
				if err == nil {
					m.settled[p[0]] = true
					m.lost = append(m.lost, p[0])
				}
				break
			}
			_, err = f.r.Reapplied(m.id, p[0], m.verifier)
			f.must(err)
			m.settled[p[0]] = true
		case 8:
			ready := m.open && m.point && m.verified && m.reconciled["erp-x"] && m.reconciled["iss-x"] && len(m.pending()) == 0
			want = ready || !m.open && m.state == StateLocked
			s := f.r.installations[m.id]
			err = f.try(Change{Kind: ChangeUnlock, Unlock: Unlock{Installation: m.id, Generation: s.generation, Lost: lostDigest(m.lost)}})
			if want && err == nil {
				m.state, m.open, m.point = StateServing, false, false
			}
		case 9:
			if !m.open || !m.point {
				continue
			}
			err = f.try(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: m.id, Recovery: f.r.installations[m.id].current.id, Generation: m.gen}})
			if err == nil {
				m.gen++
				m.point = false
			}
		case 10:
			f.advance(time.Duration(rng.Intn(90)) * time.Second)
			continue
		}
		if (err == nil) != want {
			t.Fatalf("seed %d step %d site %s: err %v want %v (%+v)", seed, step, m.id, err, want, *m)
		}
		for _, x := range sites {
			s := f.r.installations[x.id]
			if s.generation != x.gen || s.state != x.state {
				t.Fatalf("seed %d step %d %s: generation %d/%d state %s/%s", seed, step, x.id, s.generation, x.gen, s.state, x.state)
			}
			for key := range x.fenced {
				if !s.fenced[[2]string{"ws", key}] {
					t.Fatalf("seed %d: fence on %s shrank", seed, key)
				}
			}
			if s.high[[2]string{"ws", "inv"}] < x.high {
				t.Fatalf("seed %d: high water shrank", seed)
			}
			for _, tok := range x.tokens {
				if tok.Generation != x.gen && tok.Expiry.After(f.at()) {
					if _, err := f.r.Call(x.id, x.writer.ID, tok, Target{Workspace: "ws", Key: "free"}, x.writer); err == nil {
						t.Fatalf("seed %d: past-generation token accepted", seed)
					}
				}
			}
		}
	}
}
