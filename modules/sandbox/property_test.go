package sandbox

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

type modelEnv struct {
	mode, db, ring, path, prefix string
}

type modelDst struct {
	env, kind, host string
	port            int
	enabled         bool
}

type sandboxModel struct {
	envs map[string]modelEnv
	dsts map[string]modelDst
}

func (m *sandboxModel) register(id string, e modelEnv) string {
	if _, ok := m.envs[id]; ok {
		return "conflict"
	}
	if (e.mode == ModeDemo) != strings.HasPrefix(e.prefix, "DEMO") {
		return ReasonDemoPrefix
	}
	if e.mode == ModeDemo {
		production := false
		for _, o := range m.envs {
			production = production || o.mode == ModeProduction
		}
		if !production {
			return ReasonNoProduction
		}
	}
	for _, o := range m.envs {
		a, b := strings.Split(o.path, "/"), strings.Split(e.path, "/")
		short, long := a, b
		if len(a) > len(b) {
			short, long = b, a
		}
		nested := true
		for i := range short {
			nested = nested && short[i] == long[i]
		}
		if o.db == e.db || o.ring == e.ring || nested || o.prefix == e.prefix {
			return ReasonNamespaceShared
		}
	}
	m.envs[id] = e
	return ""
}

func (m *sandboxModel) allow(env, kind, host string, port int) string {
	e := m.envs[env]
	var hit *modelDst
	for _, d := range m.dsts {
		if d.host == host && d.port == port {
			d := d
			hit = &d
		}
	}
	mock := strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".invalid") || strings.HasSuffix(host, ".example")
	if e.mode == ModeProduction {
		if hit != nil || mock {
			return ReasonMockDestination
		}
		return ""
	}
	if hit == nil || !hit.enabled || hit.env != env || hit.kind != kind {
		return ReasonNotMock
	}
	return ""
}

func (m *sandboxModel) inside(env string, t Target) bool {
	e := m.envs[env]
	switch t.Kind {
	case TargetDatabase:
		return t.Name == e.db
	case TargetKeyRing:
		return t.Name == e.ring
	case TargetNumbers:
		return t.Name == e.prefix
	case TargetEvidence:
		return t.Name == e.path || strings.HasPrefix(t.Name, e.path+"/")
	case TargetConnection:
		d, ok := m.dsts[t.Name]
		return ok && d.env == env
	}
	rest, ok := strings.CutPrefix(t.Name, env)
	return ok && (rest == "" || len(rest) > 1 && rest[0] == '-')
}

func outcome(err error) string {
	var refusal Refusal
	switch {
	case err == nil:
		return ""
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, ErrConflict):
		return "conflict"
	}
	return err.Error()
}

func TestSandboxProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	reached := map[string]int{}
	for seed := 0; seed < runs; seed++ {
		runSandboxProperty(t, int64(seed), reached)
	}
	for _, name := range []string{"", ReasonNamespaceShared, ReasonNoProduction, ReasonDemoPrefix, ReasonNotMock, ReasonMockDestination, ReasonOutsideEnv, "reset-ok", ReasonNotReal, ReasonCrossEnvironment, ReasonForeignNumber, "dst-", "dst-" + ReasonNotDemo, "dst-" + ReasonNotReserved, "dst-conflict"} {
		if reached[name] == 0 {
			t.Errorf("%q never reached", name)
		}
	}
	t.Logf("reached %v", reached)
}

func runSandboxProperty(t *testing.T, seed int64, reached map[string]int) {
	rng := rand.New(rand.NewSource(seed))
	f := setup(t)
	m := &sandboxModel{envs: map[string]modelEnv{}, dsts: map[string]modelDst{}}
	names := []string{"a", "b", "c", "ab", "a1"}
	segments := []string{"/s", "/s/a", "/s/ab", "/t", "/s/a/x"}
	prefixes := []string{"INV-", "INVA-", "DEMO-", "DEMOA-", "DM-"}
	hosts := []string{"m.test", "n.test", "o.invalid", "p.example", "real.com"}
	kindsUsed := []string{DestinationERP, DestinationMailbox}
	for step := 0; step < 60; step++ {
		switch rng.Intn(5) {
		case 0:
			id := names[rng.Intn(len(names))]
			e := modelEnv{mode: []string{ModeDemo, ModeProduction}[rng.Intn(2)], db: "db" + strconv.Itoa(rng.Intn(4)), ring: "r" + strconv.Itoa(rng.Intn(4)),
				path: segments[rng.Intn(len(segments))], prefix: prefixes[rng.Intn(len(prefixes))]}
			if rng.Intn(4) > 0 {
				e.prefix = map[string][]string{ModeDemo: {"DEMO-", "DEMOA-"}, ModeProduction: {"INV-", "INVA-", "DM-"}}[e.mode][rng.Intn(2)]
			}
			err := f.try(env(Environment{ID: id, Mode: e.mode, Database: e.db, KeyRing: e.ring, EvidencePath: e.path, NumberPrefix: e.prefix}))
			want := m.register(id, e)
			reached[want]++
			if outcome(err) != want {
				t.Fatalf("seed %d step %d register %s %+v: %v, model %q", seed, step, id, e, err, want)
			}
		case 1:
			id := "d" + strconv.Itoa(step)
			envID := names[rng.Intn(len(names))]
			d := modelDst{env: envID, kind: kindsUsed[rng.Intn(2)], host: hosts[rng.Intn(len(hosts))], port: 80 + rng.Intn(2), enabled: true}
			err := f.try(dst(Destination{ID: id, Environment: envID, Kind: d.kind, Host: d.host, Port: d.port}))
			e, ok := m.envs[envID]
			taken := false
			for _, o := range m.dsts {
				taken = taken || o.host == d.host && o.port == d.port
			}
			want := ""
			switch {
			case !ok:
				want = ErrNotFound.Error()
			case taken:
				want = "conflict"
			case e.mode != ModeDemo:
				want = ReasonNotDemo
			case d.host == "real.com":
				want = ReasonNotReserved
			}
			reached["dst-"+want]++
			if outcome(err) != want {
				t.Fatalf("seed %d step %d destination %+v: %v, model %q", seed, step, d, err, want)
			}
			if want == "" {
				m.dsts[id] = d
			}
		case 2:
			if len(m.dsts) > 0 && rng.Intn(3) == 0 {
				for id, d := range m.dsts {
					f.must(f.r.DisableDestination(id, ann))
					d.enabled = false
					m.dsts[id] = d
					break
				}
			}
			envID := names[rng.Intn(len(names))]
			if _, ok := m.envs[envID]; !ok {
				continue
			}
			kind, host, port := kindsUsed[rng.Intn(2)], hosts[rng.Intn(len(hosts))], 80+rng.Intn(2)
			want := m.allow(envID, kind, host, port)
			reached[want]++
			if got := outcome(f.r.Allow(Request{Environment: envID, Kind: kind, Host: host, Port: port})); got != want {
				t.Fatalf("seed %d step %d allow %s %s %s %d: %q, model %q", seed, step, envID, kind, host, port, got, want)
			}
		case 3:
			envID := names[rng.Intn(len(names))]
			e, ok := m.envs[envID]
			if !ok || e.mode != ModeDemo {
				continue
			}
			pick := func() Target {
				switch rng.Intn(6) {
				case 0:
					return Target{Kind: TargetDatabase, Name: "db" + strconv.Itoa(rng.Intn(4))}
				case 1:
					return Target{Kind: TargetEvidence, Name: segments[rng.Intn(len(segments))]}
				case 2:
					return Target{Kind: TargetVolume, Name: names[rng.Intn(len(names))] + []string{"", "-x", "x"}[rng.Intn(3)]}
				case 3:
					return Target{Kind: TargetKeyRing, Name: "r" + strconv.Itoa(rng.Intn(4))}
				case 4:
					return Target{Kind: TargetNumbers, Name: prefixes[rng.Intn(len(prefixes))]}
				}
				return Target{Kind: TargetConnection, Name: "d" + strconv.Itoa(rng.Intn(step+1))}
			}
			plan := []Target{pick()}
			if second := pick(); rng.Intn(2) == 0 && less(plan[0], second) {
				plan = append(plan, second)
			}
			_, err := f.r.BeginReset(Plan{Environment: envID, Targets: plan}, tool)
			want := ""
			for _, target := range plan {
				if !m.inside(envID, target) {
					want = ReasonOutsideEnv
				}
			}
			if want == "" {
				reached["reset-ok"]++
			} else {
				reached[want]++
			}
			if outcome(err) != want {
				t.Fatalf("seed %d step %d reset %s %+v: %v, model %q", seed, step, envID, plan, err, want)
			}
		default:
			envID, source := names[rng.Intn(len(names))], names[rng.Intn(len(names))]
			e, ok := m.envs[envID]
			if !ok {
				continue
			}
			origin := []string{OriginReal, OriginSynthetic, OriginPublic, OriginConverted}[rng.Intn(4)]
			number := ""
			if rng.Intn(2) == 0 {
				number = prefixes[rng.Intn(len(prefixes))] + []string{"", "1"}[rng.Intn(2)]
			}
			db := e.db
			if rng.Intn(4) == 0 {
				db = "db" + strconv.Itoa(rng.Intn(4))
			}
			err := f.r.Admit(Input{Environment: envID, Source: source, Database: db, Origin: origin, Number: number})
			want := ""
			switch {
			case source != envID || db != e.db:
				want = ReasonCrossEnvironment
			case e.mode == ModeProduction && origin != OriginReal:
				want = ReasonNotReal
			case number != "" && (len(number) <= len(e.prefix) || number[:len(e.prefix)] != e.prefix):
				want = ReasonForeignNumber
			}
			reached[want]++
			if outcome(err) != want {
				t.Fatalf("seed %d step %d admit %s %s %s %s: %v, model %q", seed, step, envID, source, db, origin, err, want)
			}
		}
	}
}
