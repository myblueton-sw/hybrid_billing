package server

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

func computedImpact(*planMeta) (planImpact, error) { return impactAll(impactComputed), nil }

func failingImpact(*planMeta) (planImpact, error) {
	return planImpact{}, errors.New("impact evidence unreadable")
}

func (f world) activeRelationships(t testing.TB) uint64 {
	t.Helper()
	n, err := f.s.rels.Active(SampleWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestActivationHeldWithoutImpactEvidence(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if f.s.plans.impact != nil {
		t.Fatal("a plain server has an impact source")
	}
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	row := f.planRow(second, id)
	if row["state"] != "sealed" || row["blocked"] != "impact_not_computed" || row["can_activate"] != false ||
		fmt.Sprint(row["impact"]) != "map[erp:not_computed money:not_computed publication:not_computed]" {
		t.Fatalf("held row %+v", row)
	}
	if page := f.plansRead(second); fmt.Sprint(page.body["impact"]) != "map[erp:not_computed money:not_computed publication:not_computed]" {
		t.Errorf("legacy page impact %+v", page.body["impact"])
	}
	before := f.activeRelationships(t)
	f.failed(t, "held activation", f.run(second, "relationship.activate", id, nil), commands.PrerequisiteFailed)
	if after := f.activeRelationships(t); after != before || f.planRow(second, id)["state"] != "sealed" || f.s.plans.get(SampleWorkspace, id).activated != 0 {
		t.Fatalf("a held activation moved the revision %d -> %d", before, after)
	}
	got := f.run(second, "relationship.discard", id, nil)
	if got.status != 200 || got.body["business_state"] != "discarded" {
		t.Fatalf("discard of a held plan %+v", got)
	}
	if r := f.planRow(root, id); r["closed"] != "discarded" || r["blocked"] != "" || r["can_activate"] != false || fmt.Sprint(r["impact"]) != "map[erp:not_computed money:not_computed publication:not_computed]" {
		t.Errorf("discarded held row %+v", r)
	}
}

func TestEachMissingImpactPartHoldsActivation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	cases := map[string]planImpact{
		"money not computed":       {Money: impactNotComputed, Publication: impactComputed, ERP: impactComputed},
		"publication unknown":      {Money: impactComputed, Publication: impactUnknown, ERP: impactComputed},
		"erp missing":              {Money: impactComputed, Publication: impactComputed},
		"erp spelled otherwise":    {Money: impactComputed, Publication: impactComputed, ERP: "Computed"},
		"money missing":            {Publication: impactComputed, ERP: impactComputed},
		"publication not computed": {Money: impactComputed, Publication: impactNotComputed, ERP: impactComputed},
	}
	for name, value := range cases {
		v := value
		f.s.plans.impact = func(*planMeta) (planImpact, error) { return v, nil }
		if r := f.planRow(second, id); r["blocked"] != "impact_not_computed" || r["can_activate"] != false {
			t.Errorf("%s: row %+v", name, r)
		}
		f.failed(t, name, f.run(second, "relationship.activate", id, nil), commands.PrerequisiteFailed)
	}
	f.s.plans.impact = computedImpact
	if r := f.planRow(second, id); r["blocked"] != "" || r["can_activate"] != true {
		t.Fatalf("computed row %+v", r)
	}
	act := f.run(second, "relationship.activate", id, nil)
	if act.status != 200 || act.body["business_state"] != "activated" || f.activeRelationships(t) != 2 {
		t.Fatalf("activation with impact %+v", act)
	}
}

func TestImpactIsReadPerPlan(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	ready := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	held := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-02", "end", "", "2026-12-01"))
	f.s.plans.impact = func(m *planMeta) (planImpact, error) {
		if m.id == ready {
			return impactAll(impactComputed), nil
		}
		return planImpact{Money: impactComputed, Publication: impactNotComputed, ERP: impactComputed}, nil
	}
	if r := f.planRow(second, held); r["blocked"] != "impact_not_computed" || fmt.Sprint(r["impact"]) != "map[erp:computed money:computed publication:not_computed]" {
		t.Errorf("held plan %+v", r)
	}
	if r := f.planRow(second, ready); r["can_activate"] != true || fmt.Sprint(r["impact"]) != "map[erp:computed money:computed publication:computed]" {
		t.Errorf("ready plan %+v", r)
	}
	f.failed(t, "held plan", f.run(second, "relationship.activate", held, nil), commands.PrerequisiteFailed)
	if got := f.run(second, "relationship.activate", ready, nil); got.status != 200 {
		t.Fatalf("ready plan %+v", got)
	}
}

func TestImpactAccessorFailureHoldsActivation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	id := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	closed := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-02", "end", "", "2026-12-01"))
	f.run(root, "relationship.discard", closed, nil)
	f.s.plans.impact = failingImpact
	if r := f.planRow(second, id); r["blocked"] != "impact_unavailable" || r["can_activate"] != false || fmt.Sprint(r["impact"]) != "map[erp:unknown money:unknown publication:unknown]" {
		t.Errorf("unreadable impact row %+v", r)
	}
	if r := f.planRow(root, id); r["blocked"] != "impact_unavailable" {
		t.Errorf("unreadable impact before own proposal %+v", r)
	}
	if r := f.planRow(second, closed); r["blocked"] != "" || fmt.Sprint(r["impact"]) != "map[erp:unknown money:unknown publication:unknown]" {
		t.Errorf("closed row %+v", r)
	}
	f.failed(t, "unreadable impact", f.run(second, "relationship.activate", id, nil), commands.DependencyUnavailable)
	f.failed(t, "own proposal with unreadable impact", f.run(root, "relationship.activate", id, nil), commands.DependencyUnavailable)
	if f.activeRelationships(t) != 1 || f.planRow(second, id)["state"] != "sealed" {
		t.Error("an unreadable impact activated the plan")
	}
	if got := f.run(second, "relationship.discard", id, nil); got.status != 200 {
		t.Errorf("discard with unreadable impact %+v", got)
	}
}

func TestHoldComesAfterStageChecks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	first := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	stale := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-03", "set", "sample-unit-north-ops", "2026-12-01"))
	if r := f.planRow(root, first); r["blocked"] != "impact_not_computed" {
		t.Errorf("own proposal without impact reads %+v", r)
	}
	f.failed(t, "own proposal without impact", f.run(root, "relationship.activate", first, nil), commands.PrerequisiteFailed)
	f.s.fault = func(at string) bool { return at == "plan-active" }
	f.failed(t, "active unreadable before the hold", f.run(second, "relationship.activate", first, nil), commands.DependencyUnavailable)
	f.s.fault = nil
	f.s.plans.impact = computedImpact
	if f.run(second, "relationship.activate", first, nil).status != 200 {
		t.Fatal("activate first")
	}
	f.s.plans.impact = nil
	if r := f.planRow(second, stale); r["blocked"] != "stale" {
		t.Errorf("stale before held %+v", r)
	}
	f.failed(t, "stale without impact", f.run(second, "relationship.activate", stale, nil), commands.InvalidStage)
	f.s.plans.impact = failingImpact
	if r := f.planRow(second, stale); r["blocked"] != "stale" {
		t.Errorf("stale before unreadable impact %+v", r)
	}
	f.failed(t, "stale with unreadable impact", f.run(second, "relationship.activate", stale, nil), commands.InvalidStage)
	f.s.plans.impact = nil
	today := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-04", "end", "", "2026-11-16"))
	f.advance(48 * time.Hour)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	if r := f.planRow(second, today); r["blocked"] != "effective_date_passed" {
		t.Errorf("passed before held %+v", r)
	}
	f.failed(t, "passed without impact", f.run(second, "relationship.activate", today, nil), commands.InvalidStage)
	f.s.plans.impact = failingImpact
	if r := f.planRow(second, today); r["blocked"] != "effective_date_passed" {
		t.Errorf("passed before unreadable impact %+v", r)
	}
	f.failed(t, "passed with unreadable impact", f.run(second, "relationship.activate", today, nil), commands.InvalidStage)
	if n, _ := f.s.rels.Active(SampleWorkspace); n != 2 {
		t.Errorf("active %d", n)
	}
	if info, _ := f.s.rels.Plan(SampleWorkspace, today); info.State != relationships.PlanSealed {
		t.Errorf("passed plan %v", info.State)
	}
}
