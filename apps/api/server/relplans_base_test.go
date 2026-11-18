package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (f world) proposeOn(a actor, base string, fields map[string]string) reply {
	f.t.Helper()
	fields["base"] = base
	return f.run(a, "relationship.propose", relationshipsTarget, fields)
}

func hierarchyMove() map[string]string {
	return map[string]string{"kind": "hierarchy", "member": "sample-acct-01", "action": "set", "target": "sample-unit-north-data", "from": "2026-12-01"}
}

func TestProposeWithBaseEqualToActive(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.proposeOn(root, "1", hierarchyMove())
	id := planID(t, got)
	if got.body["business_state"] != "sealed" {
		t.Fatalf("state %+v", got.body)
	}
	if row := f.planRow(root, id); fmt.Sprint(row["base"]) != "1" || row["change"].(map[string]any)["member"] != "sample-acct-01" {
		t.Errorf("row %+v", row)
	}
	plain := f.proposeChange(root, "hierarchy", "sample-acct-02", "end", "", "2026-12-05")
	if planID(t, plain) == "" || plain.body["business_state"] != "sealed" {
		t.Errorf("without base %+v", plain.body)
	}
}

func TestProposeWithStaleBaseCreatesNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	loaded := f.plansRead(root)
	if fmt.Sprint(loaded.body["active_revision"]) != "1" {
		t.Fatalf("loaded %+v", loaded.body)
	}
	other := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-02", "end", "", "2026-12-05"))
	f.s.plans.impact = computedImpact
	if act := f.run(second, "relationship.activate", other, nil); act.status != 200 || act.body["fields"].(map[string]any)["revision"] != "2" {
		t.Fatalf("activate %+v", act)
	}
	before := len(f.s.plans.byKey)
	open := fmt.Sprint(f.plansRead(root).body["open_plans"])
	got := f.proposeOn(root, "1", hierarchyMove())
	f.failed(t, "stale base", got, commands.RevisionConflict)
	if got.status != 409 || got.body["fields"] != nil {
		t.Errorf("refusal %d %s", got.status, got.raw)
	}
	if len(f.s.plans.byKey) != before || fmt.Sprint(f.plansRead(root).body["open_plans"]) != open {
		t.Errorf("a plan was created: %d -> %d", before, len(f.s.plans.byKey))
	}
	f.failed(t, "future base", f.proposeOn(root, "3", hierarchyMove()), commands.RevisionConflict)
	if id := planID(t, f.proposeOn(root, "2", hierarchyMove())); fmt.Sprint(f.planRow(root, id)["base"]) != "2" {
		t.Error("the current base was not accepted")
	}
}

func TestProposeBaseValues(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, raw := range []string{"-1", "01", "", "18446744073709551616", "1.0", "x"} {
		f.failed(t, "base "+raw, f.proposeOn(root, raw, hierarchyMove()), commands.InvalidRequest, "base")
	}
	if len(f.s.plans.byKey) != 0 {
		t.Errorf("plans %d", len(f.s.plans.byKey))
	}
}

func TestProposeBaseCheckOrder(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	past := hierarchyMove()
	past["from"] = "2020-01-01"
	f.failed(t, "past date first", f.proposeOn(root, "-1", past), commands.InvalidRequest, "from")
	bad := hierarchyMove()
	bad["action"] = "join"
	f.failed(t, "change first", f.proposeOn(root, "9", bad), commands.InvalidRequest, "action")
	f.s.fault = func(at string) bool { return at == "plan-active" }
	f.failed(t, "base before active", f.proposeOn(root, "-1", hierarchyMove()), commands.InvalidRequest, "base")
	f.failed(t, "active unreadable", f.proposeOn(root, "9", hierarchyMove()), commands.DependencyUnavailable)
	f.s.fault = nil
	unknown := hierarchyMove()
	unknown["member"] = "sample-nobody"
	f.failed(t, "conflict before ids", f.proposeOn(root, "9", unknown), commands.RevisionConflict)
	f.failed(t, "ids with current base", f.proposeOn(root, "1", unknown), commands.InvalidRequest, "member")
}

func TestProposeBaseAtTheHandler(t *testing.T) {
	f := setup(t, SeedSample).ready()
	h := relationshipCommand{s: f.s, kind: "propose"}
	apply := func(base string) *commands.Failure {
		fields := hierarchyMove()
		fields["base"] = base
		_, err := h.Apply(commands.Context{User: SampleAdmin, Rationale: "sample reason", Fields: fields})
		failure, _ := err.(*commands.Failure)
		return failure
	}
	for base, want := range map[string]string{"01": "invalid_request [base]", "001": "invalid_request [base]", "-1": "invalid_request [base]", "9": "revision_conflict []", "0": "revision_conflict []"} {
		if got := apply(base); got == nil || fmt.Sprint(got.Code, " ", got.Fields) != want {
			t.Errorf("base %q: %+v, want %s", base, got, want)
		}
	}
	for _, wide := range []string{"4294967296", "18446744073709551615"} {
		f.failed(t, "wide base "+wide, f.proposeOn(f.as(AdminLogin), wide, hierarchyMove()), commands.RevisionConflict)
	}
	if len(f.s.plans.byKey) != 0 {
		t.Errorf("plans %d", len(f.s.plans.byKey))
	}
}

func TestProposeBaseInTheCommandSchema(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := map[string]any{}
	if err := json.Unmarshal(f.mustOpenAPI(), &doc); err != nil {
		t.Fatal(err)
	}
	var at any = doc
	for _, key := range []string{"paths", "/commands/relationship.propose", "post", "requestBody", "content", "application/json", "schema", "properties", "body"} {
		at = at.(map[string]any)[key]
	}
	body := at.(map[string]any)
	base := body["properties"].(map[string]any)["base"].(map[string]any)
	if base["x-type"] != "integer" || base["type"] != "string" {
		t.Errorf("base schema %v", base)
	}
	if strings.Contains(fmt.Sprint(body["required"]), "base") || fmt.Sprint(body["required"]) != "[action from kind member]" {
		t.Errorf("required %v", body["required"])
	}
}
