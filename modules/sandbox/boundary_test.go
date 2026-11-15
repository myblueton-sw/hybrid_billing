package sandbox

import (
	"strconv"
	"testing"
)

func TestRegisterCaps(t *testing.T) {
	cases := []struct {
		name   string
		fill   func(f world, n int)
		cap    int
		change Change
	}{
		{"environments", func(f world, n int) {
			for len(f.r.environments) < n {
				id := "x" + strconv.Itoa(len(f.r.environments))
				f.r.environments[id] = &environment{view: Environment{ID: id, EvidencePath: "/" + id}}
			}
		}, maxEnvironments, env(Environment{ID: "new", Mode: ModeProduction, Database: "new_db", KeyRing: "new_ring", EvidencePath: "/new", NumberPrefix: "NEW-"})},
		{"destinations", func(f world, n int) {
			e := f.r.environments["demo"]
			for len(e.destinations) < n {
				e.destinations = append(e.destinations, "x")
			}
		}, maxDestinations, dst(Destination{ID: "new", Environment: "demo", Kind: DestinationMailbox, Host: "mail.mock.test", Port: 25})},
		{"schemas", func(f world, n int) {
			for len(f.r.schemas) < n {
				f.r.schemas["x"+strconv.Itoa(len(f.r.schemas))] = &Schema{}
			}
		}, maxSchemas, Change{Kind: ChangeSchema, Schema: Schema{ID: "new", Fields: []FieldSpec{{Key: "mode", Kind: FieldFlag}}}}},
	}
	full := standard(t)
	for len(full.r.schemas) < maxSchemas {
		full.r.schemas["x"+strconv.Itoa(len(full.r.schemas))] = &Schema{}
	}
	reason(t, "rule before room", full.try(Change{Kind: ChangeSchema, Schema: Schema{ID: "s", Fields: []FieldSpec{{Key: "api_token", Kind: FieldFlag}}}}), ReasonSensitiveKey)
	for _, c := range cases {
		f := standard(t)
		c.fill(f, c.cap-1)
		if err := f.try(c.change); err != nil {
			t.Errorf("%s exact fit: %v", c.name, err)
		}
		g := standard(t)
		c.fill(g, c.cap)
		expect(t, c.name, g.try(c.change), ErrLimit)
	}
}

func TestPendingCaps(t *testing.T) {
	f := setup(t)
	propose := func(who Actor, id string) error {
		return second(f.r.Propose(Change{Kind: ChangeSchema, Schema: Schema{ID: id, Fields: []FieldSpec{{Key: "mode", Kind: FieldFlag}}}}, who))
	}
	for index := 0; index < maxMine; index++ {
		f.must(propose(ann, "a"+strconv.Itoa(index)))
	}
	expect(t, "mine", propose(ann, "a-over"), ErrLimit)
	people := []Actor{bob, cat, {ID: "dan", Kind: KindHuman}}
	for _, who := range people {
		for index := 0; index < maxMine; index++ {
			f.must(propose(who, who.ID+strconv.Itoa(index)))
		}
	}
	expect(t, "total", propose(Actor{ID: "eve", Kind: KindHuman}, "e"), ErrLimit)
	if len(f.r.pending) != maxPending {
		t.Errorf("pending %d", len(f.r.pending))
	}
}

func TestDoneHistory(t *testing.T) {
	f := setup(t)
	first := f.approve(env(prod))
	for index := 2; index < maxDone; index++ {
		f.r.done = append(f.r.done, uint64(100000+index))
	}
	f.approve(Change{Kind: ChangeSchema, Schema: schema()})
	if _, err := f.r.Proposal(first.Number); err != nil || len(f.r.done) != maxDone {
		t.Fatal("pruned early")
	}
	last := f.approve(env(demo))
	expect(t, "pruned", second(f.r.Proposal(first.Number)), ErrNotFound)
	if _, err := f.r.Proposal(last.Number); err != nil || len(f.r.done) != maxDone {
		t.Errorf("kept %v %d", err, len(f.r.done))
	}
}
