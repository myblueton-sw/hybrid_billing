package plugins

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type modelInstall struct {
	present  bool
	state    string
	version  string
	revision uint64
}

type modelToken struct {
	text      string
	workspace string
	plugin    string
	revision  uint64
	expiry    time.Time
}

type model struct {
	installs   map[[2]string]*modelInstall
	vulnerable map[[2]string]bool
	compromise bool
	revoked    map[string]bool
	confirmed  map[string]bool
	tokens     []modelToken
	now        time.Time
	licenses   map[[2]string]string
	permitted  map[string]bool
}

var (
	propertyWorkspaces = []string{"w1", "w2", "v1"}
	propertyPlugins    = []string{"ledger", "audit"}
	propertyVersions   = []string{"1.0.0", "1.1.0", "1.2.0"}
	propertyRuntimes   = map[string]Actor{"w1": run1, "w2": run2, "v1": runV}
	propertyLicenses   = []string{"Apache-2.0", "GPL-3.0-only", "LicenseRef-closed", "MIT"}
)

func publisherOf(plugin string) string {
	if plugin == "audit" {
		return "beta"
	}
	return "acme"
}

func (m *model) runnable(workspace, plugin, number string) bool {
	if !m.trusted(plugin) {
		return false
	}
	if !m.permitted[m.licenses[[2]string{plugin, number}]] {
		return false
	}
	return !m.vulnerable[[2]string{plugin, number}] || workspace == "v1"
}

func (m *model) install(workspace, plugin string) *modelInstall {
	key := [2]string{workspace, plugin}
	if m.installs[key] == nil {
		m.installs[key] = &modelInstall{}
	}
	return m.installs[key]
}

func (m *model) sweep() {
	for key, i := range m.installs {
		if i.present && i.state == StateEnabled && !m.runnable(key[0], key[1], i.version) {
			i.state = StateStopped
			i.revision++
		}
	}
}

func (m *model) live(workspace, plugin string) int {
	i := m.install(workspace, plugin)
	count := 0
	for _, t := range m.tokens {
		if t.workspace == workspace && t.plugin == plugin && m.now.Before(t.expiry) && t.revision == i.revision {
			count++
		}
	}
	return count
}

func (m *model) valid(t modelToken) bool {
	i := m.install(t.workspace, t.plugin)
	return i.present && i.state == StateEnabled && i.revision == t.revision && m.now.Before(t.expiry)
}

func propertyLicensing(rng *rand.Rand) (map[[2]string]string, map[string]bool, []string) {
	licenses, permitted, list := map[[2]string]string{}, map[string]bool{}, []string{}
	for _, plugin := range propertyPlugins {
		for _, number := range propertyVersions {
			licenses[[2]string{plugin, number}] = propertyLicenses[rng.Intn(len(propertyLicenses))]
		}
	}
	for _, license := range propertyLicenses {
		if rng.Intn(3) > 0 {
			permitted[license] = true
			list = append(list, license)
		}
	}
	return licenses, permitted, list
}

func propertyWorld(t *testing.T, licenses map[[2]string]string, list []string) world {
	f := setup(t)
	f.r.config.Licenses = list
	f.publisher("acme")
	f.publisher("beta")
	for _, plugin := range propertyPlugins {
		for _, number := range propertyVersions {
			m := manifest(plugin, number)
			m.Publisher = publisherOf(plugin)
			m.License = licenses[[2]string{plugin, number}]
			f.version(m)
			view, err := f.r.Version(plugin, number)
			if err != nil || view.Permitted != containsText(list, m.License) {
				t.Fatalf("%s %s permitted %v %v", plugin, number, view.Permitted, err)
			}
		}
	}
	f.workspace("w1", "run-1", false)
	f.workspace("w2", "run-2", false)
	f.workspace("v1", "run-v", true)
	return f
}

func TestLifecycleProperty(t *testing.T) {
	sequences, steps := 120, 160
	if testing.Short() {
		sequences, steps = 20, 80
	}
	for seed := 0; seed < sequences; seed++ {
		runProperty(t, int64(seed), steps)
	}
}

func runProperty(t *testing.T, seed int64, steps int) {
	rng := rand.New(rand.NewSource(seed))
	licenses, permitted, list := propertyLicensing(rng)
	f := propertyWorld(t, licenses, list)
	m := &model{installs: map[[2]string]*modelInstall{}, vulnerable: map[[2]string]bool{}, revoked: map[string]bool{}, confirmed: map[string]bool{}, now: start,
		licenses: licenses, permitted: permitted}
	for step := 0; step < steps; step++ {
		w := propertyWorkspaces[rng.Intn(len(propertyWorkspaces))]
		p := propertyPlugins[rng.Intn(len(propertyPlugins))]
		v := propertyVersions[rng.Intn(len(propertyVersions))]
		i := m.install(w, p)
		var err error
		var want bool
		op := rng.Intn(14)
		switch op {
		case 0:
			want = (!i.present || i.state == StateRemoved) && m.runnable(w, p, v)
			err = f.tryChecked(t, at(ChangeInstall, w, p, v))
			if want {
				i.present, i.state, i.version = true, StateInstalled, v
				i.revision++
			}
		case 1:
			want = i.present && (i.state == StateInstalled || i.state == StateStopped) && m.runnable(w, p, i.version)
			err = f.tryChecked(t, at(ChangeEnable, w, p, ""))
			if want {
				i.state = StateEnabled
				i.revision++
			}
		case 2:
			want = i.present && i.state != StateRemoved
			_, err = f.r.Disable(w, p, ann)
			if want && i.state != StateInstalled {
				i.state = StateInstalled
				i.revision++
			}
		case 3:
			want = i.present && (i.state == StateEnabled || i.state == StateStopped)
			_, err = f.r.Stop(w, p, ReasonTimeout, propertyRuntimes[w])
			if want && i.state == StateEnabled {
				i.state = StateStopped
				i.revision++
			}
		case 4, 5:
			kind, order := ChangeUpgrade, 1
			if op == 5 {
				kind, order = ChangeRollback, -1
			}
			want = i.present && i.state != StateRemoved && compare(v, i.version) == order && m.runnable(w, p, v)
			err = f.tryChecked(t, at(kind, w, p, v))
			if want {
				i.version = v
				i.revision++
			}
		case 6:
			want = i.present && i.state != StateRemoved
			err = f.tryChecked(t, at(ChangeRemove, w, p, ""))
			if want {
				i.state = StateRemoved
				i.revision++
			}
		case 7:
			want = true
			_, err = f.r.MarkVulnerable(p, v, "Flaw", ann)
			m.vulnerable[[2]string{p, v}] = true
			m.sweep()
		case 8:
			key := [2]string{p, v}
			want = m.vulnerable[key] && !m.confirmed["v"+p+v]
			_, err = f.r.ClearMark(Target{Kind: MarkVulnerable, Plugin: p, Version: v}, ann)
			if want {
				delete(m.vulnerable, key)
			}
		case 9:
			want = true
			if rng.Intn(2) == 0 {
				_, err = f.r.CompromiseKey("acme", "k1", "Leaked", ann)
				m.compromise = true
			} else {
				_, err = f.r.RevokePublisher("beta", "Ended", ann)
				m.revoked["beta"] = true
			}
			m.sweep()
		case 10:
			if rng.Intn(2) == 0 {
				want = m.compromise && !m.confirmed["compromise"]
				_, err = f.r.ClearMark(Target{Kind: MarkCompromise, Publisher: "acme", Key: "k1"}, ann)
				if want {
					m.compromise = false
				}
			} else {
				want = m.revoked["beta"] && !m.confirmed["beta"]
				_, err = f.r.ClearMark(Target{Kind: MarkRevocation, Publisher: "beta"}, ann)
				if want {
					delete(m.revoked, "beta")
				}
			}
		case 11:
			key := [2]string{p, v}
			want = m.vulnerable[key]
			_, err = f.r.ConfirmMark(Target{Kind: MarkVulnerable, Plugin: p, Version: v}, bob)
			if want {
				m.confirmed["v"+p+v] = true
			}
		case 12:
			ttl := time.Duration(1+rng.Intn(120)) * time.Second
			want = i.present && i.state == StateEnabled && m.live(w, p) < limits.Tokens
			var text string
			text, _, err = f.r.IssueToken(w, p, []Capability{events}, ttl, propertyRuntimes[w])
			if want {
				m.tokens = append(m.tokens, modelToken{text: text, workspace: w, plugin: p, revision: i.revision, expiry: m.now.Add(ttl)})
			}
		case 13:
			d := time.Duration(rng.Intn(90)) * time.Second
			f.advance(d)
			m.now = m.now.Add(d)
			want = true
		}
		if target, ready := m.licenseStep(op, i, v); ready && m.trusted(p) && !m.permitted[m.licenses[[2]string{p, target}]] {
			var refusal Refusal
			if !errors.As(err, &refusal) || refusal.Reason != ReasonLicenseNotPermitted {
				t.Fatalf("seed %d step %d op %d %s %s %s: %v; want %s", seed, step, op, w, p, v, err, ReasonLicenseNotPermitted)
			}
		}
		if (err == nil) != want {
			t.Fatalf("seed %d step %d op %d %s %s %s: err %v want success %v", seed, step, op, w, p, v, err, want)
		}
		for key, mi := range m.installs {
			got, err := f.r.Installation(key[0], key[1])
			if !mi.present {
				expect(t, "absent", err, ErrNotFound)
				continue
			}
			if got.State == StateEnabled && !m.runnable(key[0], key[1], got.Version) {
				t.Fatalf("seed %d step %d %v: enabled but not runnable", seed, step, key)
			}
			if err != nil || got.State != mi.state || got.Version != mi.version || got.Revision != mi.revision {
				t.Fatalf("seed %d step %d %v: got %+v %v want %+v", seed, step, key, got, err, *mi)
			}
		}
		for _, token := range m.tokens {
			_, err := f.r.CheckToken(token.text, events)
			if (err == nil) != m.valid(token) {
				t.Fatalf("seed %d step %d token %+v: %v", seed, step, token, err)
			}
		}
	}
}

func containsText(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (m *model) trusted(plugin string) bool {
	return !m.revoked[publisherOf(plugin)] && !(publisherOf(plugin) == "acme" && m.compromise)
}

func (m *model) licenseStep(op int, i *modelInstall, v string) (string, bool) {
	live := i.present && i.state != StateRemoved
	switch op {
	case 0:
		return v, !live
	case 1:
		return i.version, live && (i.state == StateInstalled || i.state == StateStopped)
	case 4:
		return v, live && compare(v, i.version) == 1
	case 5:
		return v, live && compare(v, i.version) == -1
	}
	return "", false
}

func (f world) tryChecked(t *testing.T, c Change) error {
	t.Helper()
	p, err := f.r.Propose(c, ann)
	if err != nil {
		return err
	}
	checked := f.r.Recheck(p.Number)
	_, err = f.r.Approve(p.Number, bob)
	if fmt.Sprint(checked) != fmt.Sprint(err) {
		t.Fatalf("%s: recheck %v, approve %v", c.Kind, checked, err)
	}
	return err
}
