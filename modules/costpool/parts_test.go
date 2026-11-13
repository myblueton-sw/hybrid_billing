package costpool

import (
	"fmt"
	"testing"
	"time"
)

func change[T any](base T, apply func(*T)) T {
	apply(&base)
	return base
}

func TestAssets(t *testing.T) {
	h := newHarness(t, nil)
	h.open(header(1, 10, 0))
	good := Asset{ID: "srv-1", Kind: Server, Method: Depreciation, Acquisition: "20000000"}
	for name, apply := range map[string]func(*Asset){
		"identifier": func(a *Asset) { a.ID = "" }, "kind": func(a *Asset) { a.Kind = "rack" }, "no kind": func(a *Asset) { a.Kind = "" }, "a method that is another kind": func(a *Asset) { a.Method = Facility },
		"no method": func(a *Asset) { a.Method = "" }, "a negative price": func(a *Asset) { a.Acquisition = "-1" }, "a negative zero": func(a *Asset) { a.Acquisition = "-0" },
		"a price that is no number": func(a *Asset) { a.Acquisition = "free" }, "a price with an exponent": func(a *Asset) { a.Acquisition = "2e7" }, "a price finer than the profile reads": func(a *Asset) { a.Acquisition = "1.23456" },
	} {
		before := h.r.dump()
		wantErr(t, second(h.r.PutAsset("ws1", "dc-1", 1, change(good, apply))), ErrInvalid, name)
		h.unchanged(before, name)
	}
	for _, c := range []struct {
		ws, pool string
		number   int
		want     error
	}{{"ws2", "dc-1", 1, ErrNotFound}, {"ws1", "dc-2", 1, ErrNotFound}, {"ws1", "dc-1", 2, ErrNotFound}, {"ws1", "dc-1", 0, ErrInvalid}, {"", "dc-1", 1, ErrInvalid}, {"ws1", "", 1, ErrInvalid}} {
		wantErr(t, second(h.r.PutAsset(c.ws, c.pool, c.number, good)), c.want, "asset")
		wantErr(t, second(h.r.PutComponent(c.ws, c.pool, c.number, cost("fee", Facility, "1"))), c.want, "component")
		wantErr(t, second(h.r.SetBasis(c.ws, c.pool, c.number, shares("1"))), c.want, "basis")
		wantErr(t, second(h.r.RecordApproval(c.ws, c.pool, c.number, approved)), c.want, "record")
		wantErr(t, h.r.Withdraw(c.ws, c.pool, c.number, ByOperator), c.want, "withdraw")
		wantErr(t, second(h.r.Standing(c.ws, c.pool, c.number)), c.want, "standing")
		wantErr(t, second(h.r.Version(c.ws, c.pool, c.number)), c.want, "version")
		wantErr(t, second(h.r.Assets(c.ws, c.pool, c.number, "", 1)), c.want, "assets")
		wantErr(t, second(h.r.Components(c.ws, c.pool, c.number, "", 1)), c.want, "components")
		wantErr(t, second(h.r.Targets(c.ws, c.pool, c.number, "", 1)), c.want, "targets")
		wantErr(t, second(h.r.Shares(c.ws, c.pool, c.number, "", 1)), c.want, "shares")
		wantErr(t, second(h.r.AssetSums(c.ws, c.pool, c.number, "", 1)), c.want, "asset sums")
	}
	wantErr(t, second(h.r.PutAsset("ws1", "dc-1", 9, change(good, func(a *Asset) { a.Kind = "" }))), ErrInvalid, "malformed before the version is looked up")
	h.asset(1, good)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.asset(1, good)
	for name, apply := range map[string]func(*Asset){
		"kind": func(a *Asset) { a.Kind = GPU }, "method": func(a *Asset) { a.Method = Lease }, "price": func(a *Asset) { a.Acquisition = "1" },
		"the spelling of the price": func(a *Asset) { a.Acquisition = "20000000.0" }, "no price": func(a *Asset) { a.Acquisition = "" },
	} {
		wantErr(t, second(h.r.PutAsset("ws1", "dc-1", 1, change(good, apply))), ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	for index, kind := range []AssetKind{GPU, Storage, Network, FacilityRoom, OtherAsset} {
		h.asset(1, Asset{ID: fmt.Sprintf("a%d", index), Kind: kind, Method: Lease})
	}
	before = h.r.dump()
	wantErr(t, second(h.r.PutAsset("ws1", "dc-1", 1, Asset{ID: "a9", Kind: Server, Method: Lease})), ErrLimit, "a seventh asset")
	wantErr(t, second(h.r.PutAsset("ws1", "dc-1", 1, change(good, func(a *Asset) { a.Kind = GPU }))), ErrConflict, "a conflict before no room")
	h.unchanged(before, "the limit of assets")
	page, err := h.r.Assets("ws1", "dc-1", 1, "a1", 4)
	if err != nil || len(page) != 4 || page[0].ID != "a2" || page[3] != good {
		t.Fatalf("%+v %v", page, err)
	}
	if err := h.r.Withdraw("ws1", "dc-1", 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	if state, err := h.r.PutAsset("ws1", "dc-1", 1, good); err != nil || state != Withdrawn {
		t.Fatal(state, err)
	}
	h2 := newHarness(t, nil)
	h2.open(header(1, 10, 0))
	if err := h2.r.Withdraw("ws1", "dc-1", 1, ByOperator); err != nil {
		t.Fatal(err)
	}
	wantErr(t, second(h2.r.PutAsset("ws1", "dc-1", 1, good)), ErrRejected, "a new asset for a version that is not a draft")
	wantErr(t, second(h2.r.PutComponent("ws1", "dc-1", 1, cost("fee", Facility, "1"))), ErrRejected, "a new component")
	wantErr(t, second(h2.r.SetBasis("ws1", "dc-1", 1, shares("1"))), ErrRejected, "a basis")
	wantErr(t, second(h2.r.RecordApproval("ws1", "dc-1", 1, approved)), ErrRejected, "an approval")
}

func TestComponents(t *testing.T) {
	h := newHarness(t, nil)
	h.open(header(1, 10, 0))
	from, to := month(10)
	h.asset(1, Asset{ID: "srv-1", Kind: Server, Method: Depreciation})
	h.asset(1, Asset{ID: "rack-1", Kind: FacilityRoom, Method: Lease})
	h.put(1, cost("fee", Facility, "200000"))
	meter := power("p1", "100000", Measured, 10)
	malformed := map[string]Component{
		"identifier":                           cost("", Facility, "1"),
		"kind":                                 cost("x", "rent", "1"),
		"no kind":                              cost("x", "", "1"),
		"no evidence":                          change(cost("x", Facility, "1"), func(c *Component) { c.Evidence = "" }),
		"no evidence for an unknown amount":    change(cost("x", Facility, ""), func(c *Component) { c.Evidence = "" }),
		"a negative amount":                    cost("x", Facility, "-1"),
		"a negative zero":                      cost("x", Facility, "-0"),
		"an amount that is no number":          cost("x", Facility, "many"),
		"an amount with five places":           cost("x", Facility, "0.00001"),
		"an amount beyond the profile":         cost("x", Facility, "1000000000000"),
		"a malformed asset":                    change(cost("x", Facility, "1"), func(c *Component) { c.Asset = "srv 1" }),
		"a depreciation without an asset":      cost("x", Depreciation, "1"),
		"a lease without an asset":             cost("x", Lease, "1"),
		"a malformed container":                change(cost("x", Licence, "1"), func(c *Component) { c.IncludedIn = "a b" }),
		"included in itself":                   change(cost("x", Licence, "1"), func(c *Component) { c.IncludedIn = "x" }),
		"a facility fee that is included":      change(cost("x", Facility, "1"), func(c *Component) { c.IncludedIn = "fee" }),
		"a depreciation that is included":      change(cost("x", Depreciation, "1"), func(c *Component) { c.Asset, c.IncludedIn = "srv-1", "fee" }),
		"a lease that is included":             change(cost("x", Lease, "1"), func(c *Component) { c.Asset, c.IncludedIn = "rack-1", "fee" }),
		"power without a model":                cost("x", PowerCost, "1"),
		"a model on another kind":              change(cost("x", Operations, "1"), func(c *Component) { c.Power = meter.Power }),
		"a grade":                              change(meter, func(c *Component) { c.Power = &Power{Grade: "guessed", Source: "m", PUE: "1.2", From: from, To: to} }),
		"no grade":                             change(meter, func(c *Component) { c.Power = &Power{Source: "m", PUE: "1.2", From: from, To: to} }),
		"measured without a source":            change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, PUE: "1.2", From: from, To: to} }),
		"estimated without a source":           change(meter, func(c *Component) { c.Power = &Power{Grade: Estimated, PUE: "1.2", From: from, To: to} }),
		"measured without a PUE":               change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", From: from, To: to} }),
		"estimated without a PUE":              change(meter, func(c *Component) { c.Power = &Power{Grade: Estimated, Source: "m", From: from, To: to} }),
		"a PUE below one":                      change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", PUE: "0.9999", From: from, To: to} }),
		"a PUE below one for an unknown grade": change(meter, func(c *Component) { c.Power = &Power{Grade: GradeNotKnown, PUE: "0.5", From: from, To: to} }),
		"a PUE of negative zero":               change(meter, func(c *Component) { c.Power = &Power{Grade: GradeNotKnown, PUE: "-0", From: from, To: to} }),
		"a negative PUE":                       change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", PUE: "-2", From: from, To: to} }),
		"a malformed source":                   change(meter, func(c *Component) { c.Power = &Power{Grade: GradeNotKnown, Source: "m 1", From: from, To: to} }),
		"a model period backwards":             change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", PUE: "1.2", From: to, To: from} }),
		"a model period of no length":          change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", PUE: "1.2", From: from, To: from} }),
		"a model period with a fraction": change(meter, func(c *Component) {
			c.Power = &Power{Grade: Measured, Source: "m", PUE: "1.2", From: from.Add(time.Millisecond), To: to}
		}),
		"a model without a period": change(meter, func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", PUE: "1.2"} }),
	}
	for name, spec := range malformed {
		before := h.r.dump()
		wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 1, spec)), ErrInvalid, name)
		h.unchanged(before, name)
		wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 9, spec)), ErrInvalid, name+" before the version is looked up")
	}
	rejected := map[string]Component{
		"an asset that does not exist":      change(cost("x", Operations, "1"), func(c *Component) { c.Asset = "srv-9" }),
		"a depreciation for a leased asset": change(cost("x", Depreciation, "1"), func(c *Component) { c.Asset = "rack-1" }),
		"a lease for a depreciated asset":   change(cost("x", Lease, "1"), func(c *Component) { c.Asset = "srv-1" }),
		"a model that starts before the version": change(meter, func(c *Component) {
			c.Power = &Power{Grade: Measured, Source: "m", PUE: "1.2", From: from.Add(-time.Second), To: to}
		}),
		"a model that ends after the version": change(meter, func(c *Component) {
			c.Power = &Power{Grade: Measured, Source: "m", PUE: "1.2", From: from, To: to.Add(time.Second)}
		}),
		"a container that does not exist": change(cost("x", Licence, "1"), func(c *Component) { c.IncludedIn = "fee-9" }),
	}
	for name, spec := range rejected {
		before := h.r.dump()
		wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 1, spec)), ErrRejected, name)
		h.unchanged(before, name)
		wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 9, spec)), ErrNotFound, name+" for a version that does not exist")
	}
	dep := change(cost("dep", Depreciation, "400000"), func(c *Component) { c.Asset = "srv-1" })
	lease := change(cost("lease", Lease, "300000"), func(c *Component) { c.Asset = "rack-1" })
	h.put(1, dep, lease, cost("lic", Licence, "100000"), change(cost("ops", Operations, "50000"), func(c *Component) { c.Asset = "srv-1" }))
	for name, spec := range map[string]Component{
		"a second depreciation of an asset": change(dep, func(c *Component) { c.ID = "dep-2" }),
		"a second lease of an asset":        change(lease, func(c *Component) { c.ID = "lease-2" }),
		"included in a licence":             change(cost("x", Operations, "1"), func(c *Component) { c.IncludedIn = "lic" }),
		"included in a depreciation":        change(cost("x", Operations, "1"), func(c *Component) { c.IncludedIn = "dep" }),
		"included in an operating cost":     change(cost("x", Licence, "1"), func(c *Component) { c.IncludedIn = "ops" }),
	} {
		wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 1, spec)), ErrRejected, name)
	}
	inside := change(meter, func(c *Component) { c.IncludedIn = "fee" })
	edge := change(power("p2", "150000", Estimated, 10), func(c *Component) {
		c.Asset, c.Power.From, c.Power.To = "srv-1", to.Add(-time.Second).In(time.FixedZone("kst", 9*3600)), to
	})
	unknown := change(cost("p3", PowerCost, ""), func(c *Component) {
		c.Power, c.IncludedIn = &Power{Grade: GradeNotKnown, From: from, To: from.Add(time.Second)}, "lease"
	})
	h.put(1, inside, edge, unknown, change(cost("ops2", Operations, ""), func(c *Component) { c.IncludedIn = "lease" }))
	h.clock.advance(time.Hour)
	before := h.r.dump()
	h.put(1, inside, edge, unknown, dep)
	for name, spec := range map[string]Component{
		"kind":                      change(cost("lic", Operations, "100000"), func(c *Component) { c.Evidence = "doc-lic" }),
		"amount":                    cost("lic", Licence, "100001"),
		"the spelling of an amount": cost("lic", Licence, "100000.0"),
		"an amount not known":       cost("lic", Licence, ""),
		"asset":                     change(cost("lic", Licence, "100000"), func(c *Component) { c.Asset = "srv-1" }),
		"evidence":                  change(cost("lic", Licence, "100000"), func(c *Component) { c.Evidence = "doc-9" }),
		"inclusion":                 change(cost("lic", Licence, "100000"), func(c *Component) { c.IncludedIn = "fee" }),
		"no inclusion":              meter,
		"grade":                     change(inside, func(c *Component) { c.Power = change(*c.Power, func(p *Power) { p.Grade = Estimated }).ptr() }),
		"source":                    change(inside, func(c *Component) { c.Power = change(*c.Power, func(p *Power) { p.Source = "meter-2" }).ptr() }),
		"PUE":                       change(inside, func(c *Component) { c.Power = change(*c.Power, func(p *Power) { p.PUE = "1.40" }).ptr() }),
		"model start": change(inside, func(c *Component) {
			c.Power = change(*c.Power, func(p *Power) { p.From = from.Add(time.Second) }).ptr()
		}),
		"model end":                        change(inside, func(c *Component) { c.Power = change(*c.Power, func(p *Power) { p.To = to.Add(-time.Second) }).ptr() }),
		"an amount under a key that moved": change(cost("lic", Licence, ""), func(c *Component) { c.Asset = "100000" }),
	} {
		wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 1, spec)), ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	caller := change(cost("p4", PowerCost, "1"), func(c *Component) { c.Power = &Power{Grade: Measured, Source: "m", PUE: "1", From: from, To: to} })
	h.put(1, caller)
	caller.Power.Source = "changed"
	page, err := h.r.Components("ws1", "dc-1", 1, "p3", 4)
	if err != nil || len(page) != 1 || page[0].ID != "p4" || page[0].Power.Source != "m" || page[0].Power.From.Location() != time.UTC {
		t.Fatalf("%+v %v", page, err)
	}
	if zoned, err := h.r.Components("ws1", "dc-1", 1, "p1", 1); err != nil || zoned[0].ID != "p2" || zoned[0].Power.From.Location() != time.UTC || !zoned[0].Power.From.Equal(to.Add(-time.Second)) {
		t.Fatalf("a model is kept in UTC: %+v %v", zoned, err)
	}
	page[0].Power.Source = "changed"
	if again, _ := h.r.Components("ws1", "dc-1", 1, "p3", 4); again[0].Power.Source != "m" {
		t.Fatal("a page shares a model")
	}
	first, err := h.r.Components("ws1", "dc-1", 1, "", 4)
	if err != nil || len(first) != 4 || first[0].ID != "dep" || first[1].ID != "fee" || first[2].ID != "lease" || first[3].ID != "lic" {
		t.Fatalf("%+v %v", first, err)
	}
	if view := h.view(1); view.Components != 10 || view.Assets != 2 || view.Blockers.Summed != 7 || view.Blockers.WithoutAmount != 0 {
		t.Fatalf("%+v %+v", view, view.Blockers)
	}
	h.put(1, cost("c11", Operations, "1"), cost("c12", Operations, ""))
	fine, err := NewRegistry(testLimits(), mustProfile(t, 6), h.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fine.Open(header(1, 10, 0)); err != nil {
		t.Fatal(err)
	}
	wantErr(t, second(fine.PutComponent("ws1", "dc-1", 1, cost("x", Operations, "0.00001"))), ErrInvalid, "five places under a profile that reads six")
	if _, err := fine.PutComponent("ws1", "dc-1", 1, cost("x", Operations, "0.0001")); err != nil {
		t.Fatal(err)
	}
	if _, err := fine.PutAsset("ws1", "dc-1", 1, Asset{ID: "a", Kind: Server, Method: Lease, Acquisition: "0.000001"}); err != nil {
		t.Fatalf("a price is bound by the profile only: %v", err)
	}
	before = h.r.dump()
	wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 1, cost("c13", Operations, "1"))), ErrLimit, "a thirteenth component")
	wantErr(t, second(h.r.PutComponent("ws1", "dc-1", 1, change(cost("c13", Operations, "1"), func(c *Component) { c.Asset = "srv-9" }))), ErrRejected, "rejected before no room")
	h.unchanged(before, "the limit of components")
	if got := h.view(1).Blockers; got.Summed != 9 || got.WithoutAmount != 1 || got.Multiple != MultipleNotDetermined {
		t.Fatalf("%+v", got)
	}
}

func (p Power) ptr() *Power { return &p }

func TestBasis(t *testing.T) {
	h := newHarness(t, nil)
	h.open(header(1, 10, 0))
	good := Basis{ID: "basis-1", Measure: ReservedCapacity, Capacity: "100", Targets: []Target{{ID: "org-b", Bearer: Organization, Weight: "20"}, {ID: "org-a", Bearer: Organization, Weight: "60.0"}, {ID: "spare", Bearer: MSP, Weight: "20", Idle: true}}}
	target := func(index int, apply func(*Target)) func(*Basis) {
		return func(b *Basis) {
			b.Targets = append([]Target(nil), b.Targets...)
			apply(&b.Targets[index])
		}
	}
	for name, apply := range map[string]func(*Basis){
		"identifier":                             func(b *Basis) { b.ID = "" },
		"a measure that claims a meter":          func(b *Basis) { b.Measure = "measured_usage" },
		"no measure":                             func(b *Basis) { b.Measure = "" },
		"no capacity":                            func(b *Basis) { b.Capacity = "" },
		"a capacity of zero":                     func(b *Basis) { b.Capacity = "0" },
		"a negative capacity":                    func(b *Basis) { b.Capacity = "-100" },
		"a weight of negative zero":              target(0, func(t *Target) { t.Weight = "-0" }),
		"no targets":                             func(b *Basis) { b.Targets = nil },
		"a target identifier":                    target(0, func(t *Target) { t.ID = "org b" }),
		"a target twice":                         target(0, func(t *Target) { t.ID = "org-a" }),
		"a bearer":                               target(0, func(t *Target) { t.Bearer = "customer" }),
		"no bearer":                              target(0, func(t *Target) { t.Bearer = "" }),
		"a negative weight":                      target(0, func(t *Target) { t.Weight = "-20" }),
		"no weight":                              target(0, func(t *Target) { t.Weight = "" }),
		"idle capacity borne by an organisation": target(0, func(t *Target) { t.Idle = true }),
		"weights below the capacity":             target(0, func(t *Target) { t.Weight = "19.9999" }),
		"weights above the capacity":             target(0, func(t *Target) { t.Weight = "20.0001" }),
		"a capacity that is the used part":       func(b *Basis) { b.Capacity = "80" },
	} {
		before := h.r.dump()
		wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 1, change(good, apply))), ErrInvalid, name)
		h.unchanged(before, name)
	}
	many := Basis{ID: "b", Measure: FixedAgreement, Capacity: "7"}
	for index := 0; index < 7; index++ {
		many.Targets = append(many.Targets, Target{ID: fmt.Sprintf("t%d", index), Bearer: MSP, Weight: "1"})
	}
	wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 1, many)), ErrLimit, "a seventh target")
	wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 9, many)), ErrLimit, "the number of targets is judged before the version is looked up")
	wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 1, change(many, func(b *Basis) { b.Targets = append([]Target(nil), b.Targets...); b.Targets[6].Weight = "x" }))), ErrLimit, "before the targets are read")
	wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 1, change(many, func(b *Basis) { b.Capacity = "0" }))), ErrInvalid, "after the basis's own fields")
	small, err := NewRegistry(testLimits(), profileOf(t, 3, 3), h.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := small.Open(header(1, 10, 0)); err != nil {
		t.Fatal(err)
	}
	wantErr(t, second(small.SetBasis("ws1", "dc-1", 1, Basis{ID: "b", Measure: FixedAgreement, Capacity: "999", Targets: []Target{{ID: "a", Bearer: MSP, Weight: "999"}, {ID: "b", Bearer: MSP, Weight: "1"}}})), ErrInvalid, "weights whose sum the profile cannot hold")
	wantErr(t, second(h.r.Targets("ws1", "dc-1", 1, "", 1)), ErrRejected, "the targets of a version without a basis")
	h.basis(1, good)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	reordered := change(good, func(b *Basis) { b.Targets = []Target{good.Targets[2], good.Targets[0], good.Targets[1]} })
	h.basis(1, reordered)
	for name, apply := range map[string]func(*Basis){
		"identifier": func(b *Basis) { b.ID = "basis-2" }, "measure": func(b *Basis) { b.Measure = FixedAgreement }, "the spelling of the capacity": func(b *Basis) { b.Capacity = "100.0" },
		"the spelling of a weight": target(1, func(t *Target) { t.Weight = "60" }), "a bearer": target(0, func(t *Target) { t.Bearer = MSP }), "the idle mark": target(2, func(t *Target) { t.Idle = false }),
		"a target's identifier": target(0, func(t *Target) { t.ID = "org-c" }),
		"weights moved": func(b *Basis) {
			b.Targets = []Target{{ID: "org-b", Bearer: Organization, Weight: "60.0"}, {ID: "org-a", Bearer: Organization, Weight: "20"}, good.Targets[2]}
		},
		"a target less": func(b *Basis) { b.Capacity, b.Targets = "80.0", b.Targets[:2] },
	} {
		wantErr(t, second(h.r.SetBasis("ws1", "dc-1", 1, change(good, apply))), ErrConflict, name)
	}
	h.unchanged(before, "repeats and conflicts")
	view := h.view(1)
	if view.Targets != 3 || view.Basis == nil || view.Basis.ID != "basis-1" || view.Basis.Capacity != "100" || view.Basis.Targets != nil || !view.Blockers.BasisSet {
		t.Fatalf("%+v", view)
	}
	page, err := h.r.Targets("ws1", "dc-1", 1, "org-a", 4)
	if err != nil || len(page) != 2 || page[0] != good.Targets[0] || page[1] != good.Targets[2] {
		t.Fatalf("%+v %v", page, err)
	}
	page[0].Weight = "999"
	if again, _ := h.r.Targets("ws1", "dc-1", 1, "", 2); len(again) != 2 || again[0].ID != "org-a" || again[0].Weight != "60.0" || again[1].Weight != "20" {
		t.Fatalf("a page shares the targets: %+v", again)
	}
	caller := Basis{ID: "b", Measure: FixedAgreement, Capacity: "2", Targets: []Target{{ID: "z", Bearer: Unattributed, Weight: "0", Idle: true}, {ID: "a", Bearer: MSP, Weight: "2"}}}
	h.open(header(2, 11, 0))
	h.basis(2, caller)
	if caller.Targets[0].ID != "z" {
		t.Fatal("the caller's targets were sorted")
	}
	caller.Targets[0].Weight = "9"
	if again, _ := h.r.Targets("ws1", "dc-1", 2, "", 4); again[1].Weight != "0" {
		t.Fatal("a basis shares the caller's targets")
	}
}
