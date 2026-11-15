package thirdparty

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

var permissive = map[string]bool{"MIT": true, "Apache-2.0": true, "BSD-2-Clause": true, "BSD-3-Clause": true, "ISC": true, "Zlib": true, "OFL-1.1": true, "CC-BY-4.0": true}

func modelAllowed(license, dist, mod string) bool {
	switch {
	case dist == DistNotShipped:
		return true
	case license == "PolyForm-Noncommercial-1.0.0":
		return false
	case permissive[license]:
		return true
	case license == "MPL-2.0":
		return dist != DistBundled || mod == ModUnmodified
	case strings.HasPrefix(license, "LGPL") || license == "CC-BY-SA-4.0":
		return dist != DistBundled
	}
	return dist == DistToolOnly
}

func TestThirdPartyProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	reached := map[string]int{}
	licenseList := []string{"MIT", "Apache-2.0", "MPL-2.0", "LGPL-3.0-only", "GPL-3.0-only", "AGPL-3.0-only", "CC-BY-SA-4.0", "PolyForm-Noncommercial-1.0.0", "OFL-1.1"}
	dists := []string{DistBundled, DistSeparate, DistToolOnly, DistNotShipped}
	for seed := 0; seed < runs; seed++ {
		rng := rand.New(rand.NewSource(int64(seed)))
		f := setup(t)
		state := map[string]string{}
		view := map[string]Component{}
		ids := []string{}
		for step := 0; step < 25; step++ {
			id := "c" + itoa(step)
			c := comp(id, "n"+itoa(rng.Intn(15)), "1.0."+itoa(step))
			c.License = licenseList[rng.Intn(len(licenseList))]
			c.Distribution = dists[rng.Intn(4)]
			if c.Distribution == DistToolOnly || c.Distribution == DistNotShipped {
				c.Artifact = ""
			}
			if rng.Intn(3) == 0 {
				c.Modification, c.Patch = ModPatched, sum("p"+id)
			}
			c.Passed, c.Offline = rng.Intn(6) > 0, rng.Intn(6) > 0
			if rng.Intn(6) == 0 {
				c.Telemetry = TelemetryRequired
			}
			for _, other := range ids {
				if rng.Intn(5) == 0 {
					c.Dependencies = append(c.Dependencies, other)
				}
			}
			sort.Strings(c.Dependencies)
			shipped := c.Distribution == DistBundled || c.Distribution == DistSeparate
			want := ""
			for _, d := range c.Dependencies {
				if shipped && (view[d].Distribution == DistToolOnly || view[d].Distribution == DistNotShipped) {
					want = ReasonToolDependency
				}
			}
			err := f.try(Change{Kind: ChangeComponent, Component: c})
			if want != "" {
				reason(t, "registration", err, want)
				reached["reg-"+want]++
				continue
			}
			f.must(err)
			ids = append(ids, id)
			view[id], state[id] = c, StateCandidate
			if rng.Intn(4) == 0 {
				continue
			}
			want = ""
			switch {
			case !c.Passed:
				want = ReasonTestsFailed
			case !c.Offline && shipped:
				want = ReasonNotOffline
			case c.Telemetry == TelemetryRequired:
				want = ReasonTelemetryRequired
			case !modelAllowed(c.License, c.Distribution, c.Modification):
				want = ReasonLicenseBoundary
			}
			for _, d := range c.Dependencies {
				in := c.Distribution
				if view[d].Distribution == DistSeparate {
					in = DistSeparate
				}
				switch {
				case want != "":
				case state[d] != StateAdopted:
					want = ReasonDependencyNotAdopted
				case !modelAllowed(view[d].License, in, view[d].Modification):
					want = ReasonDependencyBoundary
				}
			}
			err = f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: id}})
			if want == "" {
				f.must(err)
				state[id] = StateAdopted
				reached["adopted"]++
			} else {
				reason(t, "adopt "+id, err, want)
				reached[want]++
			}
		}
		for _, id := range ids {
			if state[id] == StateAdopted && rng.Intn(8) == 0 {
				f.approve(Change{Kind: ChangeRevoke, Decision: Decision{Component: id, Reason: "test"}})
				state[id] = StateRevoked
				reached["revoked"]++
			}
		}
		pick := []string{}
		if seed%2 == 0 {
			taken, in := map[string]bool{}, map[string]bool{}
			for _, id := range ids {
				c := view[id]
				ok := state[id] == StateAdopted && (c.Distribution == DistBundled || c.Distribution == DistSeparate) && !taken[strings.ToLower(c.Name)]
				for _, d := range c.Dependencies {
					ok = ok && in[d]
				}
				if ok {
					pick, in[id], taken[strings.ToLower(c.Name)] = append(pick, id), true, true
				}
			}
		} else {
			for _, id := range ids {
				if rng.Intn(2) == 0 {
					pick = append(pick, id)
				}
			}
		}
		if len(pick) == 0 {
			continue
		}
		sort.Strings(pick)
		listed := map[string]bool{}
		for _, id := range pick {
			listed[id] = true
		}
		expected := []string{}
		names := map[string]bool{}
		for _, id := range pick {
			c := view[id]
			switch state[id] {
			case StateRevoked:
				expected = append(expected, id+"/"+ProblemRevoked)
			case StateCandidate:
				expected = append(expected, id+"/"+ProblemNotAdopted)
			}
			shipped := c.Distribution == DistBundled || c.Distribution == DistSeparate
			if !shipped {
				expected = append(expected, id+"/"+ProblemNotShipped)
			}
			for _, d := range c.Dependencies {
				if shipped && !listed[d] {
					expected = append(expected, id+"/"+ProblemMissingDependency)
					break
				}
			}
			if names[strings.ToLower(c.Name)] {
				expected = append(expected, id+"/"+ProblemDuplicateName)
			}
			names[strings.ToLower(c.Name)] = true
		}
		release := Release{Version: "v", Commit: sum("p")[:40], Components: pick}
		check, err := f.r.CheckRelease(release)
		f.must(err)
		got := []string{}
		for _, p := range check.Problems {
			got = append(got, p.Component+"/"+p.Problem)
		}
		sort.Strings(got)
		sort.Strings(expected)
		if fmt.Sprint(got) != fmt.Sprint(expected) || check.Ready != (len(expected) == 0) {
			t.Fatalf("seed %d problems %v, model %v", seed, got, expected)
		}
		if check.Ready {
			var buf []byte
			put := func(s string) {
				var n [8]byte
				binary.BigEndian.PutUint64(n[:], uint64(len(s)))
				buf = append(append(buf, n[:]...), s...)
			}
			for _, s := range []string{"hybrid-billing/notice-bundle/v1", release.Version, release.Commit, fmt.Sprint(len(pick))} {
				put(s)
			}
			for _, id := range pick {
				c := view[id]
				for _, s := range []string{c.ID, c.Name, c.Version, c.Commit, c.License, c.LicenseText, c.Notice, c.Artifact, c.Patch, c.Distribution, c.Modification} {
					put(s)
				}
			}
			bundle := sha256.Sum256(buf)
			if check.Bundle != hex.EncodeToString(bundle[:]) {
				t.Fatalf("seed %d bundle", seed)
			}
		}
		if check.Ready {
			reached["ready"]++
		} else {
			reached["blocked"]++
		}
	}
	for _, name := range []string{"adopted", ReasonLicenseBoundary, ReasonDependencyNotAdopted, ReasonDependencyBoundary, ReasonTestsFailed, ReasonNotOffline, "reg-" + ReasonToolDependency, "ready", "blocked", "revoked"} {
		if reached[name] == 0 {
			t.Errorf("%s never reached", name)
		}
	}
	t.Logf("reached %v", reached)
}
