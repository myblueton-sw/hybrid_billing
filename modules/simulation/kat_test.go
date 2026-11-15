package simulation

import "testing"

var synDigests = []string{sum("bytes-syn")}

func TestKnownAnswers(t *testing.T) {
	if got := fingerprint(billing("s1"), synDigests); got != "f1cbbe4c64621053f4d2525c3b4ea6e1f99c86ee88ce914955cd1d1377986850" {
		t.Errorf("fingerprint %s", got)
	}
	list := []Result{{Name: "krw-total", Value: "1000.5", Currency: "KRW"}, {Name: "lines", Value: "12"}}
	if got := digest(encodeResults(list)); got != "df946c82a77b249a131e7c0edd3de0c1bae8bbfd53aa1aab756eedc67789cbeb" {
		t.Errorf("results %s", got)
	}
	moved := billing("s1")
	moved.ID, moved.FixtureVersion, moved.Purpose, moved.Profile, moved.Covers = "other", "9.9.9", PurposeFormat, "", nil
	if fingerprint(moved, synDigests) != fingerprint(billing("s1"), synDigests) {
		t.Error("non-input fields change the fingerprint")
	}
	inputs := map[string]func(*Scenario){
		"topology":   func(s *Scenario) { s.Topology = TopologyGroup },
		"deployment": func(s *Scenario) { s.Deployment = DeploymentCloud },
		"seed":       func(s *Scenario) { s.Seed = "8" },
		"start":      func(s *Scenario) { s.Start = s.Start.Add(1) },
		"zone":       func(s *Scenario) { s.Zone = "UTC" },
		"relations":  func(s *Scenario) { s.Relations = "rel-2" },
		"policy":     func(s *Scenario) { s.Policy = "pol-2" },
		"fx":         func(s *Scenario) { s.FX = "fx-2" },
		"events":     func(s *Scenario) { s.Events = sum("x") },
		"count":      func(s *Scenario) { s.EventCount = 101 },
	}
	for name, mutate := range inputs {
		s := billing("s1")
		mutate(&s)
		if fingerprint(s, synDigests) == fingerprint(billing("s1"), synDigests) {
			t.Errorf("%s not in fingerprint", name)
		}
	}
	if fingerprint(billing("s1"), []string{sum("bytes-syn"), sum("x")}) == fingerprint(billing("s1"), synDigests) {
		t.Error("dataset digests not in fingerprint")
	}
	f := standard(t)
	copied := synthetic("syn-copy")
	copied.Digest = sum("bytes-syn")
	f.approve(Change{Kind: ChangeDataset, Dataset: copied})
	renamed := billing("s3")
	renamed.Datasets, renamed.Events = []string{"syn-copy"}, billing("s1").Events
	f.approve(Change{Kind: ChangeScenario, Scenario: renamed})
	if f.r.scenarios["s3"].fingerprint != f.r.scenarios["s1"].fingerprint {
		t.Error("a renamed copy of a dataset escapes the fingerprint")
	}
	for _, d := range []Dataset{synthetic("a-one"), synthetic("b-two"), synthetic("z-one"), synthetic("c-two")} {
		if d.ID == "z-one" {
			d.Digest = sum("bytes-a-one")
		}
		if d.ID == "c-two" {
			d.Digest = sum("bytes-b-two")
		}
		f.approve(Change{Kind: ChangeDataset, Dataset: d})
	}
	pair, swapped := billing("s4"), billing("s5")
	pair.Datasets, swapped.Datasets = []string{"a-one", "b-two"}, []string{"c-two", "z-one"}
	f.approve(Change{Kind: ChangeScenario, Scenario: pair})
	swapped.Events = pair.Events
	f.approve(Change{Kind: ChangeScenario, Scenario: swapped})
	if f.r.scenarios["s4"].fingerprint != f.r.scenarios["s5"].fingerprint {
		t.Error("renamed copies in another ID order escape the fingerprint")
	}
	if digest(encodeResults([]Result{{Name: "a", Value: "1", Currency: ""}})) == digest(encodeResults([]Result{{Name: "a", Value: "1", Currency: "KRW"}})) {
		t.Error("currency not in results digest")
	}
}
