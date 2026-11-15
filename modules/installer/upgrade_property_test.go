package installer

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
)

func hostView(f world, site string) Observation {
	s := f.r.sites[site]
	services, volumes, ports := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for name, keys := range s.record.services {
		if !s.removed[name] {
			services[name] = true
			for _, key := range keys {
				ports[key] = true
			}
		}
	}
	for name, keys := range s.created {
		services[name] = true
		for _, key := range keys {
			ports[key] = true
		}
	}
	for host, vol := range s.record.volumes {
		if !vol.detached {
			volumes[host] = true
		}
	}
	for host := range s.createdVols {
		volumes[host] = true
	}
	o := empty()
	o.Services, o.Volumes = sortedKeys(services), sortedKeys(volumes)
	for key := range ports {
		parts := strings.SplitN(key, "/", 2)
		o.Ports = append(o.Ports, Port{Number: atoi(parts[0]), Protocol: parts[1]})
	}
	sort.Slice(o.Ports, func(a, b int) bool { return portLess(o.Ports[a], o.Ports[b]) })
	return o
}

func atoi(text string) int {
	n := 0
	for _, c := range text {
		n = n*10 + int(c-'0')
	}
	return n
}

func TestUpgradeProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	for seed := 0; seed < runs; seed++ {
		runUpgradeProperty(t, int64(seed))
	}
}

func runUpgradeProperty(t *testing.T, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	f := standard(t)
	f.approve(signed(upgraded("2.0.0", "1.0.0")))
	f.approve(signed(upgraded("3.0.0", "1.0.0", "2.0.0")))
	f.install("s1", p1)
	version, touched, failedSince, rollbacks := "1.0.0", false, 0, map[string]int{}
	everData := map[string]bool{"ledger": true}
	var current string
	for step := 0; step < 60; step++ {
		s := f.r.sites["s1"]
		f.observe("s1", p1, hostView(f, "s1"))
		switch rng.Intn(5) {
		case 0, 1:
			if current != "" {
				continue
			}
			target := []string{"2.0.0", "3.0.0"}[rng.Intn(2)]
			v, err := f.r.PreviewUpgrade("s1", target)
			if err != nil || len(v.Conflicts) > 0 {
				continue
			}
			p := f.approve(Change{Kind: ChangeUpgrade, Acceptance: Acceptance{Site: "s1", Version: target, Hash: v.Hash}})
			run, err := f.r.StartRun("s1", p.Number, ann)
			f.must(err)
			current = run.ID
		case 2:
			if current == "" {
				continue
			}
			run, _ := f.r.Run("s1", current)
			stop := rng.Intn(len(run.Steps))
			for run.Done < len(run.Steps) && run.Done <= stop {
				name := run.Steps[run.Done]
				if strings.HasPrefix(name, "service:") {
					touched = true
				}
				run = f.drive("s1", current, p1, name)
			}
			if run.State == RunCompleted {
				version, touched, failedSince, current = run.Version, false, 0, ""
				for _, vol := range f.r.releases[run.Version].release.Volumes {
					if vol.Data {
						everData[vol.Name] = true
					}
				}
			}
		case 3:
			if current == "" {
				continue
			}
			run, _ := f.r.Cancel("s1", current, ann)
			if run.Done >= 2 {
				failedSince++
			}
			current = ""
		case 4:
			if current != "" {
				continue
			}
			v, err := f.r.PreviewRollback("s1")
			if touched && err == nil {
				t.Fatalf("seed %d: rollback offered after a service step", seed)
			}
			if err != nil || len(v.Conflicts) > 0 {
				continue
			}
			last := f.r.lastFailed(s)
			p := f.approve(Change{Kind: ChangeRollback, Acceptance: Acceptance{Site: "s1", Version: version, Hash: v.Hash, Run: last}})
			run, err := f.r.StartRun("s1", p.Number, ann)
			f.must(err)
			run = f.drive("s1", run.ID, p1, StepVerify)
			if run.State == RunCompleted {
				rollbacks[last]++
				failedSince = 0
			}
		}
		inst, _ := f.r.Installed("s1")
		if inst.Version != version {
			t.Fatalf("seed %d step %d: version %s want %s", seed, step, inst.Version, version)
		}
		if (inst.State == StateUpgradeFailed) != (failedSince > 0) {
			t.Fatalf("seed %d step %d: state %s failed %d", seed, step, inst.State, failedSince)
		}
		for host := range everData {
			if s.record.volumes[host] == nil && version != "1.0.0" {
				t.Fatalf("seed %d: data volume %s left the owned set", seed, host)
			}
		}
		for run, count := range rollbacks {
			if count > 1 {
				t.Fatalf("seed %d: run %s rolled back twice", seed, run)
			}
		}
	}
}
