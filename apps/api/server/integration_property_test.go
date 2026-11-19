package server

import (
	"math/rand"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

var stableKinds = []string{pluginApproveKind, pluginProposeKind, pluginStopKind, pluginDisableKind, subProposeKind, subApproveKind, subPauseKind, subRequestKind,
	subResumeKind, subRevokeKind, letterRedeliverKind, letterDismissKind}

func randomRights(r *rand.Rand) []gate.Action {
	out := []gate.Action{}
	for _, right := range gate.IntegrationRights() {
		if r.Intn(2) == 0 {
			out = append(out, right)
		}
	}
	return out
}

func TestCommandsAreRefusedExactlyWhenTheRightIsMissing(t *testing.T) {
	for seed := int64(1); seed <= 25; seed++ {
		r := rand.New(rand.NewSource(seed))
		f := setup(t, SeedSample).ready()
		rights := randomRights(r)
		f.setRights(t, SampleAdmin3, rights...)
		for step := 0; step < 12; step++ {
			if r.Intn(4) == 0 {
				rights = randomRights(r)
				f.setRights(t, SampleAdmin3, rights...)
			}
			kind := stableKinds[r.Intn(len(stableKinds))]
			holds := false
			for _, right := range rights {
				holds = holds || string(right) == wantRights[kind]
			}
			got := f.runWith(f.as(Admin3Login), f.s.Revision(), integrationCommand+kind, matrixTarget(kind), fieldsFor(kind))
			denied := got.status == 403 && got.body["code"] == string(commands.PermissionDenied)
			if denied == holds {
				t.Fatalf("seed %d step %d: %s with %v: %s", seed, step, kind, rights, got.raw)
			}
		}
	}
}

func TestTheReadAgreesWithRandomRights(t *testing.T) {
	for seed := int64(1); seed <= 25; seed++ {
		r := rand.New(rand.NewSource(seed + 100))
		f := setup(t, SeedSample).ready()
		rights := randomRights(r)
		f.setRights(t, SampleAdmin3, append(rights, gate.AdministerInstallation)...)
		has := map[gate.Action]bool{}
		names := []string{}
		for _, right := range rights {
			has[right] = true
		}
		for _, right := range gate.IntegrationRights() {
			if has[right] {
				names = append(names, string(right))
			}
		}
		got := f.integrations(f.as(Admin3Login))
		if got.status != 200 || rightsText(got) != joinNames(names) {
			t.Fatalf("seed %d rights %s want %v", seed, got.raw, names)
		}
		modules := has[gate.AdministerModules] || has[gate.ApproveModules] || has[gate.StopModules]
		channels := has[gate.AdministerChannels] || has[gate.ApproveChannels] || has[gate.StopChannels] || has[gate.ReprocessEvents]
		if (areaState(got, "clients") == "shown") != has[gate.AdministerAPIClients] || (areaState(got, "modules") == "shown") != modules ||
			(areaState(got, "channels") == "shown") != channels || (areaState(got, "events") == "shown") != channels {
			t.Fatalf("seed %d areas", seed)
		}
		walkActions(got.body, func(a map[string]any) {
			if a["blocked"] == blockedNoRight && a["allowed"] == true {
				t.Errorf("seed %d: a no_right action is allowed", seed)
			}
		})
	}
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ","
		}
		out += n
	}
	return out
}

func walkActions(v any, visit func(map[string]any)) {
	switch x := v.(type) {
	case map[string]any:
		if _, ok := x["blocked"]; ok {
			visit(x)
		}
		for _, child := range x {
			walkActions(child, visit)
		}
	case []any:
		for _, child := range x {
			walkActions(child, visit)
		}
	}
}
