package installer

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

type snapshot struct {
	observation string
	revision    int
	probe       int
}

type modelSite struct {
	id        string
	probe     int
	revision  int
	alt       bool
	observed  bool
	at        time.Time
	ports     []string
	services  []string
	volumes   []string
	accepted  *snapshot
	number    uint64
	run       string
	state     string
	done      int
	host      string
	installed bool
	seq       int
	paused    int
	code      string
	expiry    time.Time
}

var (
	weights      = []int{0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 5, 5, 5, 6, 10, 10, 10, 11, 11, 13, 13, 14}
	poolPorts    = []string{"1/tcp", "8443/tcp", "9000/tcp", "9000/udp"}
	poolServices = []string{"api", "other", "worker"}
	poolVolumes  = []string{"config", "ledger", "ledger-alt", "other"}
)

func pick(rng *rand.Rand, pool []string) []string {
	out := []string{}
	for _, value := range pool {
		if rng.Intn(4) == 0 {
			out = append(out, value)
		}
	}
	return out
}

func (m *modelSite) ledger() string {
	if m.alt {
		return "ledger-alt"
	}
	return "ledger"
}

func (m *modelSite) content() string {
	return strings.Join(m.ports, ",") + "|" + strings.Join(m.services, ",") + "|" + strings.Join(m.volumes, ",")
}

func (m *modelSite) conflicts(allowed map[string]bool) bool {
	has := func(list []string, value string) bool {
		for _, v := range list {
			if v == value {
				return true
			}
		}
		return false
	}
	for _, s := range []string{"api", "worker"} {
		if has(m.services, s) && !allowed["service:"+s] {
			return true
		}
	}
	for _, p := range []string{"8443/tcp", "9000/tcp", "9000/udp"} {
		if has(m.ports, p) && !allowed["port:"+p] {
			return true
		}
	}
	for _, v := range []string{"config", m.ledger()} {
		if has(m.volumes, v) && !allowed["volume:"+v] {
			return true
		}
	}
	return false
}

func (m *modelSite) allowed() map[string]bool {
	out := map[string]bool{}
	if m.done > 3 {
		out["volume:config"] = true
	}
	if m.done > 4 {
		out["volume:"+m.host] = true
	}
	if m.done > 5 {
		out["service:api"], out["port:8443/tcp"] = true, true
	}
	if m.done > 6 {
		out["service:worker"], out["port:9000/tcp"], out["port:9000/udp"] = true, true, true
	}
	return out
}

func (m *modelSite) fresh(now time.Time) bool {
	return m.observed && !now.After(m.at.Add(limits.ObservationLife))
}

func (m *modelSite) previewable(now time.Time, revoked bool) bool {
	return !revoked && !m.installed && m.fresh(now)
}

func (m *modelSite) current() snapshot {
	return snapshot{observation: m.content(), revision: m.revision, probe: m.probe}
}

func (m *modelSite) invalidate() {
	if m.state == RunOpen || m.state == RunInterrupted {
		m.state = RunInvalidated
	}
	m.accepted = nil
	m.observed = false
	m.code = ""
}

func probeActor(m *modelSite) Actor {
	return Actor{ID: m.id + "-probe-" + strings.Repeat("p", m.probe+1), Kind: KindSystem}
}

func toObservation(m *modelSite) Observation {
	o := empty()
	for _, p := range m.ports {
		parts := strings.SplitN(p, "/", 2)
		number := map[string]int{"1": 1, "8443": 8443, "9000": 9000}[parts[0]]
		o.Ports = append(o.Ports, Port{Number: number, Protocol: parts[1]})
	}
	sort.Slice(o.Ports, func(a, b int) bool { return portLess(o.Ports[a], o.Ports[b]) })
	o.Services = append([]string(nil), m.services...)
	o.Volumes = append([]string(nil), m.volumes...)
	return o
}

func TestRunProperty(t *testing.T) {
	sequences, steps := 150, 200
	if testing.Short() {
		sequences, steps = 20, 80
	}
	for seed := 0; seed < sequences; seed++ {
		runRunProperty(t, int64(seed), steps)
	}
}

func runRunProperty(t *testing.T, seed int64, steps int) {
	rng := rand.New(rand.NewSource(seed))
	f := setup(t)
	f.approve(keyChange("k1"))
	f.approve(signed(fixture("1.0.0")))
	sites := []*modelSite{{id: "a"}, {id: "b"}}
	for _, m := range sites {
		f.approve(Change{Kind: ChangeSite, Site: Site{ID: m.id, Tenant: "t", Platform: linux, Probe: probeActor(m).ID}})
	}
	revoked := false
	now := start
	for step := 0; step < steps; step++ {
		m := sites[rng.Intn(len(sites))]
		var err error
		want := true
		op := weights[rng.Intn(len(weights))]
		if rng.Intn(40) == 0 {
			op = 7 + rng.Intn(3)
		}
		switch op {
		case 0:
			m.ports, m.services, m.volumes = []string{}, []string{}, []string{}
			if rng.Intn(2) == 0 {
				m.ports, m.services, m.volumes = pick(rng, poolPorts), pick(rng, poolServices), pick(rng, poolVolumes)
			}
			_, err = f.r.Observe(m.id, toObservation(m), probeActor(m))
			m.observed, m.at = true, now
			m.seq++
		case 1:
			want = m.previewable(now, revoked) && !m.conflicts(nil)
			var v PreviewView
			v, err = f.r.Preview(m.id, "1.0.0")
			if err == nil {
				var p ProposalView
				p, err = f.r.Propose(Change{Kind: ChangePlan, Acceptance: Acceptance{Site: m.id, Version: "1.0.0", Hash: v.Hash, Admin: "cat"}}, ann)
				if err == nil {
					p, err = f.r.Approve(p.Number, bob)
				}
				if err == nil {
					snap := m.current()
					m.accepted, m.number = &snap, p.Number
				}
			}
		case 2:
			activeRun := m.state == RunOpen || m.state == RunInterrupted
			want = m.accepted != nil && !activeRun && m.previewable(now, revoked) && *m.accepted == m.current()
			if m.number == 0 {
				continue
			}
			var run RunView
			run, err = f.r.StartRun(m.id, m.number, ann)
			if err == nil {
				m.run, m.state, m.done, m.host, m.accepted = run.ID, RunOpen, 0, m.ledger(), nil
			}
		case 3:
			if m.run == "" {
				continue
			}
			run, _ := f.r.Run(m.id, m.run)
			want = m.state == RunOpen && m.done < 7
			name := run.Steps[min(m.done, 7)]
			digest := sum(name)
			for _, a := range fixture("1.0.0").Artifacts {
				if name == "artifact:"+a.Name {
					digest = a.Digest
				}
			}
			_, err = f.r.Complete(m.id, m.run, name, Evidence{Digest: digest}, probeActor(m))
			if want {
				m.done++
			}
		case 4:
			if m.run == "" {
				continue
			}
			want = m.state == RunOpen || m.state == RunInterrupted
			_, err = f.r.Interrupt(m.id, m.run, "power", ann)
			if m.state == RunOpen {
				m.paused, m.code = m.seq, ""
			}
			if want {
				m.state = RunInterrupted
			}
		case 5:
			if m.run == "" {
				continue
			}
			ready := m.state == RunInterrupted && m.seq > m.paused && m.fresh(now)
			want = ready && !m.conflicts(m.allowed())
			_, err = f.r.Resume(m.id, m.run, ann)
			switch {
			case want:
				m.state = RunOpen
			case ready:
				m.state = RunInvalidated
			}
		case 6:
			if m.run == "" {
				continue
			}
			want = m.state == RunOpen || m.state == RunInterrupted || m.state == RunCancelled
			_, err = f.r.Cancel(m.id, m.run, ann)
			if want {
				m.state, m.code = RunCancelled, ""
			}
		case 7:
			m.alt = !m.alt
			c := SiteConfig{Site: m.id, Version: "1.0.0"}
			if m.alt {
				c.Volumes = []VolumeOverride{{Volume: "ledger", Host: "ledger-alt"}}
			}
			err = f.try(Change{Kind: ChangeConfig, Config: c})
			m.revision++
			m.invalidate()
		case 8:
			if m.probe >= probesPerSite-1 {
				continue
			}
			m.probe++
			err = f.try(Change{Kind: ChangeProbe, Site: Site{ID: m.id, Probe: probeActor(m).ID}})
			m.invalidate()
		case 9:
			if revoked {
				_, err = f.r.ClearMark(Target{Kind: MarkRelease, Version: "1.0.0"}, ann)
			} else {
				_, err = f.r.RevokeRelease("1.0.0", "Suspected", ann)
				for _, other := range sites {
					if other.state == RunOpen || other.state == RunInterrupted {
						other.state, other.code = RunInvalidated, ""
					}
					other.accepted = nil
				}
			}
			revoked = !revoked
		case 10:
			if m.run == "" {
				continue
			}
			want = m.state == RunOpen && m.done == 7
			var code string
			ttl := time.Duration(1+rng.Intn(5)) * time.Minute
			code, err = f.r.IssueBootstrap(m.id, m.run, ttl, probeActor(m))
			if err == nil {
				m.code, m.expiry = code, now.Add(ttl)
			}
		case 13:
			if m.code == "" {
				continue
			}
			want = m.state == RunOpen && m.done == 7 && now.Before(m.expiry)
			_, err = f.r.RegisterAdmin(m.code, cat)
			m.code = ""
			if want {
				m.state, m.installed, m.done = RunCompleted, true, 8
			}
		case 14:
			if m.run == "" || m.probe == 0 {
				continue
			}
			want = false
			old := Actor{ID: m.id + "-probe-" + strings.Repeat("p", m.probe), Kind: KindSystem}
			_, err = f.r.Complete(m.id, m.run, StepPreflight, Evidence{Digest: sum(StepPreflight)}, old)
			if err == nil || !strings.Contains(err.Error(), ReasonNotProbe) {
				t.Fatalf("seed %d step %d: old probe accepted: %v", seed, step, err)
			}
		case 11, 12:
			d := time.Duration(rng.Intn(6)) * time.Minute
			f.advance(d)
			now = now.Add(d)
		}
		if (err == nil) != want {
			t.Fatalf("seed %d step %d op %d site %s: err %v want success %v (%+v)", seed, step, op, m.id, err, want, *m)
		}
		for _, s := range sites {
			if s.run == "" {
				continue
			}
			got, err := f.r.Run(s.id, s.run)
			if err != nil || got.State != s.state || got.Done != s.done {
				t.Fatalf("seed %d step %d site %s: got %+v %v want %s/%d", seed, step, s.id, got, err, s.state, s.done)
			}
			activeCount := 0
			for _, r := range f.r.sites[s.id].runs {
				if active(r.view.State) {
					activeCount++
					if r.view.Probe != f.r.sites[s.id].view.Probe || r.view.Revision != f.r.sites[s.id].revision {
						t.Fatalf("seed %d step %d: active run under another probe or revision", seed, step)
					}
				}
			}
			if activeCount > 1 {
				t.Fatalf("seed %d step %d: %d active runs", seed, step, activeCount)
			}
			if c := f.r.sites[s.id].code; c != nil {
				owner := f.r.sites[s.id].runs[f.r.codes[*c].run]
				if owner.view.State != RunOpen || owner.view.Steps[owner.view.Done] != StepRegisterAdmin {
					t.Fatalf("seed %d step %d: live code outside its step", seed, step)
				}
			}
		}
	}
}
